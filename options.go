package dark

import (
	"io/fs"
	"log/slog"
	"runtime"
)

type config struct {
	poolSize          int
	templateDir       string
	viewsFS           fs.FS // optional: embedded filesystem for views
	layoutFile        string
	uiLibrary         UILibrary
	extraDeps         []string // additional npm dependencies beyond UI library
	devMode           bool
	errorComponent    string
	notFoundComponent string
	streaming         bool
	ssrCacheSize      int // max SSR output cache entries; 0 = disabled
	logger            *slog.Logger
}

func defaultConfig() *config {
	return &config{
		poolSize:    runtime.NumCPU(),
		templateDir: "views",
		logger:      slog.Default(),
	}
}

// Option configures the dark application.
type Option func(*config)

// WithPoolSize sets the number of ramune RuntimePool workers.
func WithPoolSize(n int) Option {
	return func(c *config) { c.poolSize = n }
}

// WithTemplateDir sets the directory for TSX template files.
func WithTemplateDir(dir string) Option {
	return func(c *config) { c.templateDir = dir }
}

// WithLayout sets the layout TSX file path relative to the template directory.
func WithLayout(file string) Option {
	return func(c *config) { c.layoutFile = file }
}

// WithUILibrary selects the JSX library for SSR and client-side hydration.
// Defaults to Preact. Use dark.React to switch to React/ReactDOM.
func WithUILibrary(lib UILibrary) Option {
	return func(c *config) { c.uiLibrary = lib }
}

// WithDependencies adds additional npm dependencies beyond the UI library.
func WithDependencies(pkgs ...string) Option {
	return func(c *config) {
		c.extraDeps = append(c.extraDeps, pkgs...)
	}
}

// WithDevMode enables development mode with cache invalidation on file changes.
func WithDevMode(enabled bool) Option {
	return func(c *config) { c.devMode = enabled }
}

// WithErrorComponent sets a TSX component for rendering 500 error pages.
func WithErrorComponent(file string) Option {
	return func(c *config) { c.errorComponent = file }
}

// WithNotFoundComponent sets a TSX component for rendering 404 pages.
func WithNotFoundComponent(file string) Option {
	return func(c *config) { c.notFoundComponent = file }
}

// WithStreaming enables streaming SSR (shell-first rendering for faster TTFB).
func WithStreaming(enabled bool) Option {
	return func(c *config) { c.streaming = enabled }
}

// WithSSRCache enables SSR output caching. maxEntries sets the maximum number of
// cached component+props combinations. When the cache is full, it is cleared.
// 0 (default) disables caching.
func WithSSRCache(maxEntries int) Option {
	return func(c *config) { c.ssrCacheSize = maxEntries }
}

// WithLogger sets the structured logger for dark's internal log output.
// Defaults to slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// WithViewsFS sets an fs.FS as the source for TSX view files.
// Files are extracted to a temporary directory for esbuild at startup
// and cleaned up on Close(). This takes precedence over WithTemplateDir.
// Use fs.Sub to strip the embed prefix if needed:
//
//	//go:embed views
//	var viewsFS embed.FS
//	sub, _ := fs.Sub(viewsFS, "views")
//	app, _ := dark.New(dark.WithViewsFS(sub))
func WithViewsFS(fsys fs.FS) Option {
	return func(c *config) { c.viewsFS = fsys }
}
