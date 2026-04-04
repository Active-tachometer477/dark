package dark

import (
	"strings"
	"testing"
)

func TestRenderSimpleComponent(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "World"}, true)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "Hello World") {
		t.Fatalf("expected HTML to contain 'Hello World', got: %s", html)
	}
	if !strings.Contains(html, "<div>") {
		t.Fatalf("expected HTML to contain '<div>', got: %s", html)
	}
}

func TestRenderWithLayout(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.layoutFile = "layout.tsx"
	cfg.poolSize = 1

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "World", "title": "Test Page"}, false)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "<html>") {
		t.Fatalf("expected HTML to contain '<html>', got: %s", html)
	}
	if !strings.Contains(html, "Hello World") {
		t.Fatalf("expected HTML to contain 'Hello World', got: %s", html)
	}
	if !strings.Contains(html, "Test Page") {
		t.Fatalf("expected HTML to contain 'Test Page', got: %s", html)
	}
}

func TestSSRCacheHit(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1
	cfg.ssrCacheSize = 100

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	props := map[string]any{"name": "Cached"}

	// First render: cache miss.
	html1, css1, err := r.render("simple.tsx", nil, props, true)
	if err != nil {
		t.Fatalf("render 1: %v", err)
	}

	// Second render with same props: should return identical result from cache.
	html2, css2, err := r.render("simple.tsx", nil, props, true)
	if err != nil {
		t.Fatalf("render 2: %v", err)
	}

	if html1 != html2 {
		t.Fatalf("expected identical HTML from cache, got:\n  1: %s\n  2: %s", html1, html2)
	}
	if css1 != css2 {
		t.Fatalf("expected identical CSS from cache")
	}
}

func TestSSRCacheMissOnDifferentProps(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1
	cfg.ssrCacheSize = 100

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html1, _, err := r.render("simple.tsx", nil, map[string]any{"name": "Alice"}, true)
	if err != nil {
		t.Fatalf("render Alice: %v", err)
	}

	html2, _, err := r.render("simple.tsx", nil, map[string]any{"name": "Bob"}, true)
	if err != nil {
		t.Fatalf("render Bob: %v", err)
	}

	if html1 == html2 {
		t.Fatalf("expected different HTML for different props, got same: %s", html1)
	}
	if !strings.Contains(html1, "Alice") || !strings.Contains(html2, "Bob") {
		t.Fatalf("expected Alice and Bob in respective outputs")
	}
}

func TestSSRCacheDisabledByDefault(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1
	// ssrCacheSize = 0 (default, disabled)

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	// Should work fine without caching.
	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "NoCacheTest"}, true)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "NoCacheTest") {
		t.Fatalf("expected NoCacheTest, got: %s", html)
	}
}

func TestSSRCacheEviction(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1
	cfg.ssrCacheSize = 2 // tiny cache

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	// Fill cache with 2 entries.
	r.render("simple.tsx", nil, map[string]any{"name": "A"}, true)
	r.render("simple.tsx", nil, map[string]any{"name": "B"}, true)

	// Third entry should trigger eviction (cache cleared).
	r.render("simple.tsx", nil, map[string]any{"name": "C"}, true)

	// Cache should still work after eviction.
	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "C"}, true)
	if err != nil {
		t.Fatalf("render after eviction: %v", err)
	}
	if !strings.Contains(html, "Hello C") {
		t.Fatalf("expected 'Hello C', got: %s", html)
	}
}

func TestRenderSkipLayoutForHtmx(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.layoutFile = "layout.tsx"
	cfg.poolSize = 1

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("simple.tsx", nil, map[string]any{"name": "World"}, true)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(html, "<html>") {
		t.Fatalf("expected NO <html> when layout is skipped, got: %s", html)
	}
	if !strings.Contains(html, "Hello World") {
		t.Fatalf("expected HTML to contain 'Hello World', got: %s", html)
	}
}

func TestRenderReactComponent(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.poolSize = 1
	cfg.uiLibrary = React

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("react_simple.tsx", nil, map[string]any{"name": "React"}, true)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// React's renderToString may insert <!-- --> comments between text nodes.
	if !strings.Contains(html, "React") {
		t.Fatalf("expected HTML to contain 'React', got: %s", html)
	}
	if !strings.Contains(html, "<div>") {
		t.Fatalf("expected HTML to contain '<div>', got: %s", html)
	}
}

func TestRenderReactWithLayout(t *testing.T) {
	cfg := defaultConfig()
	cfg.templateDir = "_testdata"
	cfg.layoutFile = "react_layout.tsx"
	cfg.poolSize = 1
	cfg.uiLibrary = React

	r, err := newRenderer(cfg)
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	defer r.close()

	html, _, err := r.render("react_simple.tsx", nil, map[string]any{"name": "React", "title": "Test Page"}, false)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "<html>") {
		t.Fatalf("expected HTML to contain '<html>', got: %s", html)
	}
	if !strings.Contains(html, "React") {
		t.Fatalf("expected HTML to contain 'React', got: %s", html)
	}
	if !strings.Contains(html, "Test Page") {
		t.Fatalf("expected HTML to contain 'Test Page', got: %s", html)
	}
}
