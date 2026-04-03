package dark

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseErrorInfoBasic(t *testing.T) {
	errStr := `dark: render pages/blog.tsx: at pages/blog.tsx:5:10
ramune: Eval: TypeError: Cannot read property 'map' of undefined
    at eval:123:45`

	info := parseErrorInfo(errStr)

	if info.component != "pages/blog.tsx" {
		t.Fatalf("expected component pages/blog.tsx, got: %s", info.component)
	}
	if info.title != "TypeError" {
		t.Fatalf("expected title TypeError, got: %s", info.title)
	}
	if !strings.Contains(info.message, "Cannot read property") {
		t.Fatalf("expected message about 'map', got: %s", info.message)
	}
	if len(info.frames) == 0 {
		t.Fatal("expected at least one frame")
	}
	if info.frames[0].file != "pages/blog.tsx" {
		t.Fatalf("expected first frame file pages/blog.tsx, got: %s", info.frames[0].file)
	}
	if info.frames[0].line != 5 {
		t.Fatalf("expected first frame line 5, got: %d", info.frames[0].line)
	}
}

func TestParseErrorInfoNoSourceMap(t *testing.T) {
	errStr := "dark: render pages/foo.tsx: ramune: Eval: SyntaxError: unexpected token"

	info := parseErrorInfo(errStr)

	if info.component != "pages/foo.tsx" {
		t.Fatalf("expected component pages/foo.tsx, got: %s", info.component)
	}
	if info.title != "SyntaxError" {
		t.Fatalf("expected title SyntaxError, got: %s", info.title)
	}
}

func TestParseErrorInfoFallbackMessage(t *testing.T) {
	errStr := "some generic error without JS error type"

	info := parseErrorInfo(errStr)

	if info.message != errStr {
		t.Fatalf("expected fallback message, got: %s", info.message)
	}
}

func TestReadSourceSnippet(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	os.WriteFile(filepath.Join(dir, "test.tsx"), []byte(content), 0o644)

	snippet := readSourceSnippet(dir, errorFrame{file: "test.tsx", line: 5, col: 0}, 2)

	if snippet == nil {
		t.Fatal("expected snippet, got nil")
	}
	if snippet.errorLine != 5 {
		t.Fatalf("expected errorLine 5, got %d", snippet.errorLine)
	}
	// Should show lines 3-7 (5 ± 2)
	if len(snippet.lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(snippet.lines))
	}
	if snippet.lines[0].num != 3 {
		t.Fatalf("expected first line num 3, got %d", snippet.lines[0].num)
	}
	// Line 5 should be marked as error
	for _, ln := range snippet.lines {
		if ln.num == 5 && !ln.isError {
			t.Fatal("expected line 5 to be marked as error")
		}
		if ln.num != 5 && ln.isError {
			t.Fatalf("expected line %d NOT to be marked as error", ln.num)
		}
	}
}

func TestReadSourceSnippetMissingFile(t *testing.T) {
	snippet := readSourceSnippet("/nonexistent", errorFrame{file: "nope.tsx", line: 1, col: 0}, 2)
	if snippet != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestDevOverlayRendersHTML(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1), WithDevMode(true))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	// Trigger a render error with a broken component reference.
	app.Get("/broken", Route{
		Component: "nonexistent_component.tsx",
		Loader: func(ctx Context) (any, error) {
			return nil, nil
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/broken")
	if err != nil {
		t.Fatalf("GET /broken: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	// Should contain the new overlay structure.
	if !strings.Contains(s, "error-header") {
		t.Fatalf("expected rich error overlay, got: %s", s)
	}
	if !strings.Contains(s, "Raw Error") {
		t.Fatalf("expected Raw Error section, got: %s", s)
	}
}
