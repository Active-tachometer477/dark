package dark

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpAppMIMEType = "text/html;profile=mcp-app"

// MCPApp is an MCP server that uses dark's SSR + esbuild toolchain to render
// TSX components as self-contained MCP App UIs.
type MCPApp struct {
	server   *mcp.Server
	renderer *renderer
	bundler  *mcpBundler
	config   *mcpConfig
	tools    map[string]*mcpToolEntry
	mu       sync.RWMutex
}

type mcpToolEntry struct {
	component   string
	resourceURI string
}

// UIToolDef defines a UI tool's metadata.
type UIToolDef struct {
	Description string // Tool description for the LLM
	Component   string // TSX file path relative to template directory
	Title       string // Optional human-readable title
}

// NewMCPApp creates a new MCP application server with dark SSR rendering.
func NewMCPApp(name, version string, opts ...MCPOption) (*MCPApp, error) {
	cfg := defaultMCPConfig(name, version)
	for _, o := range opts {
		o(cfg)
	}

	rendCfg := &config{
		poolSize:     cfg.poolSize,
		templateDir:  cfg.templateDir,
		dependencies: []string{"preact", "preact-render-to-string"},
		devMode:      cfg.devMode,
	}
	rend, err := newRenderer(rendCfg)
	if err != nil {
		return nil, fmt.Errorf("dark: failed to create MCP renderer: %w", err)
	}

	bundler, err := newMCPBundler(cfg)
	if err != nil {
		rend.close()
		return nil, fmt.Errorf("dark: failed to create MCP bundler: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)

	return &MCPApp{
		server:   server,
		renderer: rend,
		bundler:  bundler,
		config:   cfg,
		tools:    make(map[string]*mcpToolEntry),
	}, nil
}

// Close releases all resources held by the MCP application.
func (m *MCPApp) Close() error {
	m.bundler.close()
	return m.renderer.close()
}

// Server returns the underlying mcp.Server for advanced configuration
// (adding prompts, resources, or non-UI tools directly).
func (m *MCPApp) Server() *mcp.Server {
	return m.server
}

// RunStdio runs the MCP server over stdio transport. Blocks until the client
// disconnects or the context is cancelled.
func (m *MCPApp) RunStdio(ctx context.Context) error {
	return m.server.Run(ctx, &mcp.StdioTransport{})
}

// StreamableHTTPHandler returns an http.Handler for Streamable HTTP transport.
func (m *MCPApp) StreamableHTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return m.server },
		nil,
	)
}

// AddUITool registers an MCP tool that returns an interactive TSX-based UI.
// The handler receives typed args and returns props for the TSX component.
// dark SSR-renders the component, then assembles a self-contained HTML with
// hydration support and returns it as an inline resource in the tool result.
//
// AddUITool is a package-level function (not a method) because Go does not
// support generic methods.
func AddUITool[Args any](app *MCPApp, name string, def UIToolDef, handler func(ctx context.Context, args Args) (map[string]any, error)) error {
	resourceURI := fmt.Sprintf("ui://%s/%s.html", app.config.serverName, name)

	clientJS, clientCSS, err := app.bundler.BuildClientBundle(def.Component)
	if err != nil {
		return fmt.Errorf("dark: failed to build MCP client bundle for %s: %w", def.Component, err)
	}

	app.mu.Lock()
	app.tools[name] = &mcpToolEntry{
		component:   def.Component,
		resourceURI: resourceURI,
	}
	app.mu.Unlock()

	tool := &mcp.Tool{
		Name:        name,
		Description: def.Description,
	}
	if def.Title != "" {
		tool.Title = def.Title
	}

	mcp.AddTool(app.server, tool,
		func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
			props, err := handler(ctx, args)
			if err != nil {
				return mcpErrorResult("Error: %v", err)
			}
			if props == nil {
				props = map[string]any{}
			}

			propsJSON, err := json.Marshal(props)
			if err != nil {
				return mcpErrorResult("props marshal error: %v", err)
			}

			ssrHTML, ssrCSS, err := app.renderer.render(def.Component, nil, props, true)
			if err != nil {
				return mcpErrorResult("SSR render error: %v", err)
			}

			js := clientJS
			cCSS := clientCSS
			if app.config.devMode {
				if freshJS, freshCSS, err := app.bundler.BuildClientBundle(def.Component); err == nil {
					js = freshJS
					cCSS = freshCSS
				}
			}

			html := assembleMCPHTML(ssrHTML, joinCSS(ssrCSS, cCSS), propsJSON, js)

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(propsJSON)},
					&mcp.EmbeddedResource{
						Resource: &mcp.ResourceContents{
							URI:      resourceURI,
							MIMEType: mcpAppMIMEType,
							Text:     html,
						},
					},
				},
			}, nil, nil
		},
	)
	return nil
}

// AddTextTool registers a standard text-returning MCP tool.
func AddTextTool[Args any](app *MCPApp, name, description string, handler func(ctx context.Context, args Args) (string, error)) {
	tool := &mcp.Tool{
		Name:        name,
		Description: description,
	}

	mcp.AddTool(app.server, tool,
		func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
			text, err := handler(ctx, args)
			if err != nil {
				return mcpErrorResult("Error: %v", err)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: text},
				},
			}, nil, nil
		},
	)
}

func mcpErrorResult(format string, args ...any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
		IsError: true,
	}, nil, nil
}

func joinCSS(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(p)
	}
	return b.String()
}
