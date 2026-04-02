package dark

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// errorInfo holds structured data extracted from a render error for the dev overlay.
type errorInfo struct {
	title     string // e.g. "TypeError"
	message   string // e.g. "Cannot read property 'map' of undefined"
	component string // e.g. "pages/blog_post.tsx"
	frames    []errorFrame
	rawError  string // full error string as fallback
}

type errorFrame struct {
	file string
	line int
	col  int
}

// sourceSnippet holds source code lines around an error location.
type sourceSnippet struct {
	file      string
	errorLine int // 1-based
	lines     []numberedLine
}

type numberedLine struct {
	num     int
	text    string
	isError bool
}

var (
	componentPathRe = regexp.MustCompile(`dark: render ([^:]+):`)
	sourceLocRe     = regexp.MustCompile(`at ([^:]+):(\d+):(\d+)`)
	jsErrorTypeRe   = regexp.MustCompile(`(\w+Error): (.+)`)
)

// parseErrorInfo extracts structured info from a render error string.
func parseErrorInfo(errStr string) errorInfo {
	info := errorInfo{rawError: errStr}

	// Extract component path.
	if m := componentPathRe.FindStringSubmatch(errStr); len(m) >= 2 {
		info.component = m[1]
	}

	// Extract error type and message.
	if m := jsErrorTypeRe.FindStringSubmatch(errStr); len(m) >= 3 {
		info.title = m[1]
		info.message = m[2]
	} else {
		// Fallback: use the first meaningful line.
		for _, line := range strings.Split(errStr, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "at ") {
				info.message = line
				break
			}
		}
	}

	// Extract source-mapped stack frames.
	for _, m := range sourceLocRe.FindAllStringSubmatch(errStr, -1) {
		if len(m) >= 4 {
			line, _ := strconv.Atoi(m[2])
			col, _ := strconv.Atoi(m[3])
			info.frames = append(info.frames, errorFrame{file: m[1], line: line, col: col})
		}
	}

	return info
}

// readSourceSnippet reads source lines around the error location.
func readSourceSnippet(templateDir string, frame errorFrame, contextLines int) *sourceSnippet {
	if frame.file == "" || frame.line <= 0 {
		return nil
	}

	filePath := frame.file
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(templateDir, filePath)
	}

	// Guard against path traversal outside the template directory.
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil
	}
	absDir, _ := filepath.Abs(templateDir)
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	allLines := strings.Split(string(data), "\n")
	start := frame.line - contextLines - 1
	if start < 0 {
		start = 0
	}
	end := frame.line + contextLines
	if end > len(allLines) {
		end = len(allLines)
	}

	snippet := &sourceSnippet{
		file:      frame.file,
		errorLine: frame.line,
	}
	for i := start; i < end; i++ {
		snippet.lines = append(snippet.lines, numberedLine{
			num:     i + 1,
			text:    allLines[i],
			isError: i+1 == frame.line,
		})
	}
	return snippet
}

func (app *App) renderDevOverlay(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	info := parseErrorInfo(err.Error())

	var snippet *sourceSnippet
	if len(info.frames) > 0 {
		snippet = readSourceSnippet(app.config.templateDir, info.frames[0], 5)
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html><head><title>Dark Error</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,monospace;background:#1a1a2e;color:#e0e0e0;line-height:1.5}
.overlay{max-width:960px;margin:0 auto;padding:2rem}
.error-header{margin-bottom:1.5rem}
.error-badge{display:inline-block;background:#e94560;color:#fff;padding:2px 10px;border-radius:4px;font-size:.85em;font-weight:700;margin-right:.5rem}
.error-type{color:#e94560;font-size:1.4rem;font-weight:700}
.error-message{color:#f1f1f1;font-size:1.1rem;margin-top:.5rem;word-break:break-word}
.error-component{color:#888;font-size:.9rem;margin-top:.5rem}
.error-component span{color:#64b5f6;font-weight:600}
.section{background:#16213e;border-radius:8px;margin-bottom:1rem;overflow:hidden}
.section-title{padding:.75rem 1rem;background:#0f3460;color:#a0a0a0;font-size:.8rem;font-weight:600;text-transform:uppercase;letter-spacing:.05em}
.code-block{padding:0;margin:0;overflow-x:auto}
.code-line{display:flex;font-family:'SF Mono',Menlo,Consolas,monospace;font-size:.85rem;line-height:1.7}
.code-line:hover{background:rgba(255,255,255,.03)}
.code-line.error-line{background:rgba(233,69,96,.15)}
.line-num{min-width:3.5rem;padding:0 .75rem;text-align:right;color:#555;user-select:none;flex-shrink:0}
.error-line .line-num{color:#e94560;font-weight:700}
.line-text{padding-right:1rem;white-space:pre}
.error-line .line-text{color:#fff}
.stack-frame{padding:.4rem 1rem;font-family:'SF Mono',Menlo,Consolas,monospace;font-size:.85rem;color:#aaa}
.stack-frame .file{color:#64b5f6}
.stack-frame .loc{color:#e94560}
.raw-error{padding:1rem;color:#aaa;font-family:'SF Mono',Menlo,Consolas,monospace;font-size:.8rem;white-space:pre-wrap;word-break:break-word}
</style></head>
<body><div class="overlay">
`)

	// Error header.
	b.WriteString(`<div class="error-header">`)
	b.WriteString(`<span class="error-badge">500</span>`)
	if info.title != "" {
		fmt.Fprintf(&b, `<span class="error-type">%s</span>`, html.EscapeString(info.title))
	} else {
		b.WriteString(`<span class="error-type">Server Error</span>`)
	}
	if info.message != "" {
		fmt.Fprintf(&b, `<div class="error-message">%s</div>`, html.EscapeString(info.message))
	}
	if info.component != "" {
		fmt.Fprintf(&b, `<div class="error-component">in <span>%s</span></div>`, html.EscapeString(info.component))
	}
	b.WriteString(`</div>`)

	// Source code snippet.
	if snippet != nil {
		fmt.Fprintf(&b, `<div class="section"><div class="section-title">%s:%d</div><div class="code-block">`,
			html.EscapeString(snippet.file), snippet.errorLine)
		for _, ln := range snippet.lines {
			cls := "code-line"
			if ln.isError {
				cls += " error-line"
			}
			fmt.Fprintf(&b, `<div class="%s"><span class="line-num">%d</span><span class="line-text">%s</span></div>`,
				cls, ln.num, html.EscapeString(ln.text))
		}
		b.WriteString(`</div></div>`)
	}

	// Stack trace.
	if len(info.frames) > 0 {
		b.WriteString(`<div class="section"><div class="section-title">Stack Trace</div>`)
		for _, f := range info.frames {
			fmt.Fprintf(&b, `<div class="stack-frame"><span class="file">%s</span>:<span class="loc">%d:%d</span></div>`,
				html.EscapeString(f.file), f.line, f.col)
		}
		b.WriteString(`</div>`)
	}

	// Raw error (collapsed).
	fmt.Fprintf(&b, `<div class="section"><div class="section-title">Raw Error</div><div class="raw-error">%s</div></div>`,
		html.EscapeString(info.rawError))

	// Inject dev reload script so the overlay auto-refreshes when the developer saves a fix.
	b.WriteString(`</div>`)
	b.WriteString(devReloadScript)
	b.WriteString(`</body></html>`)

	io.WriteString(w, b.String())
}
