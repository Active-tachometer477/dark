package dark

import "runtime"

// mcpConfig holds MCP app configuration.
type mcpConfig struct {
	serverName    string
	serverVersion string
	templateDir   string
	poolSize      int
	uiLibrary     UILibrary
	minify        bool
	devMode       bool
}

func defaultMCPConfig(name, version string) *mcpConfig {
	return &mcpConfig{
		serverName:    name,
		serverVersion: version,
		templateDir:   "views",
		poolSize:      runtime.NumCPU(),
		minify:        true,
	}
}

// MCPOption configures an MCPApp.
type MCPOption func(*mcpConfig)

// WithMCPTemplateDir sets the directory for TSX template files.
func WithMCPTemplateDir(dir string) MCPOption {
	return func(c *mcpConfig) { c.templateDir = dir }
}

// WithMCPPoolSize sets the number of ramune RuntimePool workers for SSR.
func WithMCPPoolSize(n int) MCPOption {
	return func(c *mcpConfig) { c.poolSize = n }
}

// WithMCPMinify enables minification of client-side bundles (default: true).
func WithMCPMinify(enabled bool) MCPOption {
	return func(c *mcpConfig) { c.minify = enabled }
}

// WithMCPUILibrary selects the JSX library for MCP App SSR and client bundles.
func WithMCPUILibrary(lib UILibrary) MCPOption {
	return func(c *mcpConfig) { c.uiLibrary = lib }
}

// WithMCPDevMode enables development mode (source maps, no minification, cache invalidation).
func WithMCPDevMode(enabled bool) MCPOption {
	return func(c *mcpConfig) {
		c.devMode = enabled
		if enabled {
			c.minify = false
		}
	}
}
