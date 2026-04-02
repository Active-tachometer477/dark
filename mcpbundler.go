package dark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/i2y/ramune"
)

// mcpBundler bundles TSX components into self-contained client-side JS
// for MCP Apps. Unlike the SSR renderer, Preact is bundled inline.
type mcpBundler struct {
	templateDir    string
	nodeModulesDir string
	minify         bool
	devMode        bool

	cacheMu sync.RWMutex
	cache   map[string]*mcpBundleEntry // absPath → entry
}

type mcpBundleEntry struct {
	clientJS string
	css      string
	modTime  int64
}

// newMCPBundler creates a bundler and ensures Preact is installed.
func newMCPBundler(cfg *mcpConfig) (*mcpBundler, error) {
	nmDir, err := ensureMCPPreactInstalled()
	if err != nil {
		return nil, fmt.Errorf("dark: failed to install preact for MCP: %w", err)
	}
	return &mcpBundler{
		templateDir:    cfg.templateDir,
		nodeModulesDir: nmDir,
		minify:         cfg.minify,
		devMode:        cfg.devMode,
		cache:          make(map[string]*mcpBundleEntry),
	}, nil
}

// ensureMCPPreactInstalled installs Preact into a deterministic cache dir.
func ensureMCPPreactInstalled() (string, error) {
	cacheDir, err := mcpCacheDir()
	if err != nil {
		return "", err
	}
	nmDir := filepath.Join(cacheDir, "node_modules")
	if _, err := os.Stat(filepath.Join(nmDir, "preact", "package.json")); err != nil {
		if err := ramune.InstallNpmPackages([]string{"preact"}, cacheDir); err != nil {
			return "", err
		}
	}
	return nmDir, nil
}

func mcpCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "dark", "mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// BuildClientBundle bundles a TSX component into client-side JS with Preact
// inline and hydration logic. The result is cached by component path.
func (b *mcpBundler) BuildClientBundle(component string) (js string, css string, err error) {
	fullPath := filepath.Join(b.templateDir, component)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("dark: failed to resolve MCP component path %s: %w", component, err)
	}

	b.cacheMu.RLock()
	entry, ok := b.cache[absPath]
	b.cacheMu.RUnlock()

	if ok {
		if !b.devMode {
			return entry.clientJS, entry.css, nil
		}
		info, err := os.Stat(absPath)
		if err == nil && info.ModTime().UnixNano() == entry.modTime {
			return entry.clientJS, entry.css, nil
		}
	}

	js, css, err = b.buildClientBundle(absPath)
	if err != nil {
		return "", "", err
	}

	var modTime int64
	if info, err := os.Stat(absPath); err == nil {
		modTime = info.ModTime().UnixNano()
	}
	b.cacheMu.Lock()
	b.cache[absPath] = &mcpBundleEntry{clientJS: js, css: css, modTime: modTime}
	b.cacheMu.Unlock()

	return js, css, nil
}

// buildClientBundle generates a client-side entry that imports Preact and
// the component, then hydrates the #app element. It bundles everything into
// a single IIFE with Preact inlined.
func (b *mcpBundler) buildClientBundle(absComponentPath string) (string, string, error) {
	// Generate entry JS (similar to island entry pattern).
	entryCode := buildMCPClientEntryJS(absComponentPath)

	tmpDir, err := os.MkdirTemp("", "dark-mcp-*")
	if err != nil {
		return "", "", fmt.Errorf("dark: failed to create temp dir for MCP bundle: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	entryFile := filepath.Join(tmpDir, "entry.jsx")
	if err := os.WriteFile(entryFile, []byte(entryCode), 0o644); err != nil {
		return "", "", fmt.Errorf("dark: failed to write MCP entry: %w", err)
	}

	opts := api.BuildOptions{
		EntryPoints:       []string{entryFile},
		Bundle:            true,
		Format:            api.FormatIIFE,
		Platform:          api.PlatformBrowser,
		Write:             false,
		Outdir:            "/",
		NodePaths:         []string{b.nodeModulesDir},
		JSX:               api.JSXTransform,
		JSXFactory:        "h",
		JSXFragment:       "Fragment",
		MinifySyntax:      b.minify,
		MinifyWhitespace:  b.minify,
		MinifyIdentifiers: b.minify,
		LogLevel:          api.LogLevelSilent,
	}
	if b.devMode {
		opts.Sourcemap = api.SourceMapInline
	}

	result := api.Build(opts)
	if len(result.Errors) > 0 {
		return "", "", fmt.Errorf("dark: MCP esbuild error for %s: %s", absComponentPath, result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return "", "", fmt.Errorf("dark: MCP esbuild produced no output for %s", absComponentPath)
	}

	js, css := extractJSAndCSS(result.OutputFiles)
	return js, css, nil
}

// buildMCPClientEntryJS generates the client-side entry module for an MCP App.
// It imports Preact + the component, hydrates SSR'd HTML, and listens for
// tool result updates via the app bridge.
func buildMCPClientEntryJS(absComponentPath string) string {
	var sb strings.Builder
	sb.WriteString("import { h, hydrate, render } from 'preact';\n")
	fmt.Fprintf(&sb, "import __Comp from '%s';\n", absComponentPath)
	sb.WriteString("var C = __Comp.default || __Comp;\n")
	sb.WriteString(`var app = document.getElementById('app');
var props = window.__dark_mcp_props || {};

hydrate(h(C, props), app);

__dark_bridge.onToolResult(function(result) {
  if (result && result.content) {
    for (var i = 0; i < result.content.length; i++) {
      if (result.content[i].text) {
        try {
          props = JSON.parse(result.content[i].text);
          render(h(C, props), app);
        } catch(e) {}
      }
    }
  }
});

__dark_bridge.ready();
`)
	return sb.String()
}

func (b *mcpBundler) close() error {
	return nil
}
