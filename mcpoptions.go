package dark

// mcpConfig holds MCP app configuration.
type mcpConfig struct {
	serverName    string
	serverVersion string
	templateDir   string
	uiLibrary     UILibrary
	minify        bool
	devMode       bool
}

func defaultMCPConfig(name, version string) *mcpConfig {
	return &mcpConfig{
		serverName:    name,
		serverVersion: version,
		templateDir:   "views",
		minify:        true,
	}
}

// MCPOption configures an MCPApp.
type MCPOption func(*mcpConfig)

// WithMCPTemplateDir sets the directory for TSX template files.
func WithMCPTemplateDir(dir string) MCPOption {
	return func(c *mcpConfig) { c.templateDir = dir }
}

// WithMCPMinify enables minification of client-side bundles (default: true).
func WithMCPMinify(enabled bool) MCPOption {
	return func(c *mcpConfig) { c.minify = enabled }
}

// WithMCPUILibrary selects the JSX library for MCP App client bundles.
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
