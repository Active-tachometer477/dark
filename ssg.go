package dark

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

// StaticRoute defines a route to be pre-rendered at build time.
type StaticRoute struct {
	// Path is the URL path to generate (e.g., "/" or "/about").
	Path string

	// Component is the TSX file path relative to the template directory.
	Component string

	// Layout is an optional layout override (comma-separated for nested layouts).
	Layout string

	// Loader is an optional data loader. If nil, empty props are used.
	Loader LoaderFunc

	// StaticPaths returns all concrete paths for parameterized routes.
	// For example, a route "/posts/{id}" might return ["/posts/1", "/posts/2"].
	// If set, Path is ignored and each returned path is generated.
	StaticPaths func() []string
}

// GenerateStaticSite pre-renders the given routes to static HTML files.
// Each route is rendered through the full dark SSR pipeline (Loader -> TSX -> layout)
// and written to outputDir as HTML files.
func (app *App) GenerateStaticSite(outputDir string, routes []StaticRoute) error {
	// Build the handler to initialize islands, layouts, etc.
	handler, err := app.Handler()
	if err != nil {
		return fmt.Errorf("dark: SSG handler init: %w", err)
	}

	for _, route := range routes {
		paths := []string{route.Path}
		if route.StaticPaths != nil {
			paths = route.StaticPaths()
		}

		for _, urlPath := range paths {
			if err := app.generatePage(handler, outputDir, urlPath, route); err != nil {
				return fmt.Errorf("dark: SSG generate %s: %w", urlPath, err)
			}
		}
	}

	// Copy client assets (islands JS, CSS).
	if err := app.copySSGAssets(outputDir); err != nil {
		return fmt.Errorf("dark: SSG copy assets: %w", err)
	}

	return nil
}

func (app *App) generatePage(handler http.Handler, outputDir, urlPath string, route StaticRoute) error {
	// Use a synthetic request through the actual handler if no component/loader override.
	// This ensures middleware, layouts, and all post-processing runs.
	req := httptest.NewRequest("GET", urlPath, nil)
	rec := httptest.NewRecorder()

	if route.Component != "" {
		// Direct render: skip the handler and use the renderer directly.
		ctx := &darkContext{w: rec, r: req}

		var props any
		if route.Loader != nil {
			var err error
			props, err = route.Loader(ctx)
			if err != nil {
				return err
			}
		}
		if props == nil {
			props = map[string]any{}
		}

		layouts := parseLayouts(route.Layout)
		output, css, err := app.renderer.render(route.Component, layouts, props, false)
		if err != nil {
			return err
		}

		output = app.postProcessHTML(output, css, false)

		rec.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(rec, output)
	} else {
		// Use the full handler pipeline.
		handler.ServeHTTP(rec, req)
	}

	if rec.Code != http.StatusOK {
		return fmt.Errorf("got status %d", rec.Code)
	}

	// Write to file.
	outPath := ssgOutputPath(outputDir, urlPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, rec.Body.Bytes(), 0o644)
}

func (app *App) copySSGAssets(outputDir string) error {
	darkDir := filepath.Join(outputDir, "_dark")

	// Copy island chunks.
	app.renderer.clientChunksMu.RLock()
	chunks := make(map[string]string, len(app.renderer.clientChunks))
	for k, v := range app.renderer.clientChunks {
		chunks[k] = v
	}
	app.renderer.clientChunksMu.RUnlock()

	if len(chunks) > 0 {
		islandsDir := filepath.Join(darkDir, "islands")
		if err := os.MkdirAll(islandsDir, 0o755); err != nil {
			return err
		}
		for name, content := range chunks {
			if err := os.WriteFile(filepath.Join(islandsDir, name), []byte(content), 0o644); err != nil {
				return err
			}
		}
	}

	// Copy page CSS.
	app.renderer.pageCSSMu.RLock()
	cssMap := make(map[string]string, len(app.renderer.pageCSS))
	for k, v := range app.renderer.pageCSS {
		cssMap[k] = v
	}
	app.renderer.pageCSSMu.RUnlock()

	if len(cssMap) > 0 {
		cssDir := filepath.Join(darkDir, "css")
		if err := os.MkdirAll(cssDir, 0o755); err != nil {
			return err
		}
		for hash, content := range cssMap {
			if err := os.WriteFile(filepath.Join(cssDir, hash+".css"), []byte(content), 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func ssgOutputPath(outputDir, urlPath string) string {
	urlPath = strings.TrimPrefix(urlPath, "/")
	if urlPath == "" {
		return filepath.Join(outputDir, "index.html")
	}
	if strings.HasSuffix(urlPath, "/") {
		return filepath.Join(outputDir, urlPath, "index.html")
	}
	return filepath.Join(outputDir, urlPath, "index.html")
}
