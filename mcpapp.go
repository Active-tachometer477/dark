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

// MCPApp is an MCP server that uses dark's esbuild toolchain to bundle
// TSX components as self-contained MCP App UIs.
type MCPApp struct {
	server  *mcp.Server
	bundler *mcpBundler
	config  *mcpConfig
	tools   map[string]*mcpToolEntry
	mu      sync.RWMutex
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

	kit := resolveUIKit(cfg.uiLibrary)

	bundler, err := newMCPBundler(cfg, kit)
	if err != nil {
		return nil, fmt.Errorf("dark: failed to create MCP bundler: %w", err)
	}

	caps := &mcp.ServerCapabilities{}
	caps.AddExtension("io.modelcontextprotocol/ui", map[string]any{})

	server := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, &mcp.ServerOptions{
		Capabilities: caps,
	})

	return &MCPApp{
		server:  server,
		bundler: bundler,
		config:  cfg,
		tools:   make(map[string]*mcpToolEntry),
	}, nil
}

// Close releases all resources held by the MCP application.
func (m *MCPApp) Close() error {
	m.bundler.close()
	return nil
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
// dark bundles the component into a self-contained HTML resource and registers
// it with the MCP server. Tool results contain the props as JSON text, and
// the host renders the resource HTML in an iframe that receives the data
// via the MCP Apps postMessage protocol.
//
// AddUITool is a package-level function (not a method) because Go does not
// support generic methods.
func AddUITool[Args any](app *MCPApp, name string, def UIToolDef, handler func(ctx context.Context, args Args) (map[string]any, error)) error {
	resourceURI := fmt.Sprintf("ui://%s/%s.html", app.config.serverName, name)

	clientJS, clientCSS, err := app.bundler.BuildClientBundle(def.Component)
	if err != nil {
		return fmt.Errorf("dark: failed to build MCP client bundle for %s: %w", def.Component, err)
	}

	html := assembleMCPAppHTML(clientCSS, clientJS)

	app.mu.Lock()
	app.tools[name] = &mcpToolEntry{
		component:   def.Component,
		resourceURI: resourceURI,
	}
	app.mu.Unlock()

	// Register the HTML resource for this tool's UI.
	app.server.AddResource(
		&mcp.Resource{
			URI:      resourceURI,
			Name:     name,
			MIMEType: mcpAppMIMEType,
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			resHTML := html
			if app.config.devMode {
				if freshJS, freshCSS, err := app.bundler.BuildClientBundle(def.Component); err == nil {
					resHTML = assembleMCPAppHTML(freshCSS, freshJS)
				}
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      resourceURI,
					MIMEType: mcpAppMIMEType,
					Text:     resHTML,
				}},
			}, nil
		},
	)

	// Register the tool with _meta.ui linking to the resource.
	tool := &mcp.Tool{
		Name:        name,
		Description: def.Description,
		Meta: mcp.Meta{
			"ui":             map[string]any{"resourceUri": resourceURI},
			"ui/resourceUri": resourceURI,
		},
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

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(propsJSON)},
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
