package dark

import (
	"context"
	"strings"
	"testing"
)

func TestMCPBundlerClientBundle(t *testing.T) {
	cfg := defaultMCPConfig("test", "1.0.0")
	cfg.templateDir = "_testdata"
	cfg.minify = false

	b, err := newMCPBundler(cfg)
	if err != nil {
		t.Fatalf("newMCPBundler: %v", err)
	}
	defer b.close()

	js, _, err := b.BuildClientBundle("simple.tsx")
	if err != nil {
		t.Fatalf("BuildClientBundle: %v", err)
	}

	if js == "" {
		t.Fatal("client bundle JS is empty")
	}

	// Client bundle should contain the component logic and Preact rendering.
	if !strings.Contains(js, "getElementById") {
		t.Errorf("expected client bundle to contain getElementById, got (first 500 chars): %s", js[:min(500, len(js))])
	}
}

func TestMCPBundlerCache(t *testing.T) {
	cfg := defaultMCPConfig("test", "1.0.0")
	cfg.templateDir = "_testdata"
	cfg.minify = false

	b, err := newMCPBundler(cfg)
	if err != nil {
		t.Fatalf("newMCPBundler: %v", err)
	}
	defer b.close()

	js1, _, err := b.BuildClientBundle("simple.tsx")
	if err != nil {
		t.Fatalf("first BuildClientBundle: %v", err)
	}

	js2, _, err := b.BuildClientBundle("simple.tsx")
	if err != nil {
		t.Fatalf("second BuildClientBundle: %v", err)
	}

	if js1 != js2 {
		t.Error("expected cache hit to return identical JS")
	}
}

func TestMCPSSRRender(t *testing.T) {
	cfg := defaultMCPConfig("test", "1.0.0")
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1

	rendCfg := &config{
		poolSize:     cfg.poolSize,
		templateDir:  cfg.templateDir,
		dependencies: []string{"preact", "preact-render-to-string"},
	}
	r, err := newRenderer(rendCfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "MCP"}, true)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "Hello MCP") {
		t.Fatalf("expected 'Hello MCP', got: %s", html)
	}
}

func TestMCPAssembleHTML(t *testing.T) {
	props := map[string]any{"name": "Test"}
	html, err := assembleMCPHTML("<div>Hello Test</div>", "body{color:red}", props, "console.log('client')")
	if err != nil {
		t.Fatalf("assembleMCPHTML: %v", err)
	}

	checks := []struct {
		name    string
		content string
	}{
		{"doctype", "<!DOCTYPE html>"},
		{"ssr html", "Hello Test"},
		{"css", "body{color:red}"},
		{"props", `"name":"Test"`},
		{"bridge", "__dark_bridge"},
		{"client js", "console.log('client')"},
		{"app div", `<div id="app">`},
	}
	for _, c := range checks {
		if !strings.Contains(html, c.content) {
			t.Errorf("expected HTML to contain %s (%q)", c.name, c.content)
		}
	}
}

func TestMCPAppNewAndClose(t *testing.T) {
	app, err := NewMCPApp("test-server", "1.0.0",
		WithMCPTemplateDir("_testdata"),
		WithMCPPoolSize(1),
		WithMCPMinify(false),
	)
	if err != nil {
		t.Fatalf("NewMCPApp: %v", err)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestMCPAppAddUITool(t *testing.T) {
	app, err := NewMCPApp("test-server", "1.0.0",
		WithMCPTemplateDir("_testdata"),
		WithMCPPoolSize(1),
		WithMCPMinify(false),
	)
	if err != nil {
		t.Fatalf("NewMCPApp: %v", err)
	}
	defer app.Close()

	type Args struct {
		Name string `json:"name"`
	}

	AddUITool(app, "greet", UIToolDef{
		Description: "Show greeting",
		Component:   "simple.tsx",
	}, func(ctx context.Context, args Args) (map[string]any, error) {
		return map[string]any{"name": args.Name}, nil
	})

	// Verify tool was registered.
	app.mu.RLock()
	entry, ok := app.tools["greet"]
	app.mu.RUnlock()

	if !ok {
		t.Fatal("expected tool 'greet' to be registered")
	}
	if entry.component != "simple.tsx" {
		t.Errorf("expected component 'simple.tsx', got %q", entry.component)
	}
	if entry.resourceURI != "ui://test-server/greet.html" {
		t.Errorf("expected resourceURI 'ui://test-server/greet.html', got %q", entry.resourceURI)
	}
}

func TestMCPAppAddTextTool(t *testing.T) {
	app, err := NewMCPApp("test-server", "1.0.0",
		WithMCPTemplateDir("_testdata"),
		WithMCPPoolSize(1),
	)
	if err != nil {
		t.Fatalf("NewMCPApp: %v", err)
	}
	defer app.Close()

	type Args struct {
		Query string `json:"query"`
	}

	// Should not panic.
	AddTextTool(app, "search", "Search for stuff",
		func(ctx context.Context, args Args) (string, error) {
			return "results for " + args.Query, nil
		})
}

func TestMCPAppDashboardRoundTrip(t *testing.T) {
	app, err := NewMCPApp("test-server", "1.0.0",
		WithMCPTemplateDir("_testdata"),
		WithMCPPoolSize(1),
		WithMCPMinify(false),
	)
	if err != nil {
		t.Fatalf("NewMCPApp: %v", err)
	}
	defer app.Close()

	type Args struct {
		Title string `json:"title"`
	}

	AddUITool(app, "dashboard", UIToolDef{
		Description: "Show dashboard",
		Component:   "mcp_dashboard.tsx",
	}, func(ctx context.Context, args Args) (map[string]any, error) {
		return map[string]any{
			"title": args.Title,
			"items": []string{"alpha", "beta"},
		}, nil
	})

	// Verify the tool entry exists.
	app.mu.RLock()
	_, ok := app.tools["dashboard"]
	app.mu.RUnlock()
	if !ok {
		t.Fatal("expected tool 'dashboard' to be registered")
	}
}
