package dark

import "runtime"

type config struct {
	poolSize          int
	templateDir       string
	layoutFile        string
	dependencies      []string
	devMode           bool
	errorComponent    string
	notFoundComponent string
	streaming         bool
	ssrCacheSize      int // max SSR output cache entries; 0 = disabled
}

func defaultConfig() *config {
	return &config{
		poolSize:     runtime.NumCPU(),
		templateDir:  "views",
		dependencies: []string{"preact", "preact-render-to-string"},
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

// WithDependencies adds additional npm dependencies beyond preact.
func WithDependencies(pkgs ...string) Option {
	return func(c *config) {
		c.dependencies = append(c.dependencies, pkgs...)
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
