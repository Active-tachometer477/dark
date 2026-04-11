package dark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// mcpBundler bundles TSX components into self-contained client-side JS
// for MCP Apps. The UI library (Preact or React) is bundled inline.
type mcpBundler struct {
	uikit          *uikit
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

// newMCPBundler creates a bundler and ensures the UI library is installed.
func newMCPBundler(cfg *mcpConfig, kit *uikit) (*mcpBundler, error) {
	nmDir, err := ensureNpmDeps([]string{"mcp"}, kit.clientPkg, kit.clientPkgCheck)
	if err != nil {
		return nil, fmt.Errorf("dark: failed to install client packages for MCP: %w", err)
	}
	return &mcpBundler{
		uikit:          kit,
		templateDir:    cfg.templateDir,
		nodeModulesDir: nmDir,
		minify:         cfg.minify,
		devMode:        cfg.devMode,
		cache:          make(map[string]*mcpBundleEntry),
	}, nil
}

// BuildClientBundle bundles a TSX component into client-side JS with the
// UI library inline and hydration logic. The result is cached by component path.
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

// buildClientBundle generates a client-side entry that imports the UI library
// and the component, then hydrates the #app element. It bundles everything into
// a single IIFE with the UI library inlined.
func (b *mcpBundler) buildClientBundle(absComponentPath string) (string, string, error) {
	entryCode := buildMCPClientEntryJS(absComponentPath, b.uikit)

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
		JSX:               api.JSXAutomatic,
		JSXImportSource:   b.uikit.jsxImportSource,
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
// It imports the UI library + the component, then renders on tool result
// notifications received via the app bridge (client-side rendering, no SSR).
func buildMCPClientEntryJS(absComponentPath string, kit *uikit) string {
	var sb strings.Builder
	sb.WriteString(kit.mcpImport)
	fmt.Fprintf(&sb, "import __Comp from '%s';\n", absComponentPath)
	sb.WriteString("var C = __Comp.default || __Comp;\n")
	fmt.Fprintf(&sb, `var app = document.getElementById('app');

__dark_bridge.onToolResult(function(result) {
  if (result && result.content) {
    for (var i = 0; i < result.content.length; i++) {
      var c = result.content[i];
      if (c.type === 'text' && c.text) {
        try {
          var props = JSON.parse(c.text);
          %s
        } catch(e) {}
      }
    }
  }
});

__dark_bridge.ready();
`, kit.mcpRender)
	return sb.String()
}

func (b *mcpBundler) close() error {
	return nil
}
