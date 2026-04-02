package dark

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newIslandApp(t *testing.T) *App {
	t.Helper()
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app.Island("counter", "island_counter.tsx")

	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Island Test", "count": 5}, nil
		},
	})

	return app
}

func TestIslandSSRMarkers(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "<dark-island") {
		t.Fatalf("expected <dark-island> marker in output, got: %s", html)
	}
	if !strings.Contains(html, `data-name="counter"`) {
		t.Fatalf("expected data-name=\"counter\" in output, got: %s", html)
	}
	if !strings.Contains(html, `data-props=`) {
		t.Fatalf("expected data-props attribute in output, got: %s", html)
	}
	if !strings.Contains(html, `data-load="load"`) {
		t.Fatalf("expected data-load=\"load\" in output, got: %s", html)
	}
}

func TestIslandSSRContent(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// The page title should be SSR'd.
	if !strings.Contains(html, "Island Test") {
		t.Fatalf("expected 'Island Test' in output, got: %s", html)
	}
	// The counter component should be SSR'd inside the island marker.
	if !strings.Contains(html, `class="counter"`) {
		t.Fatalf("expected counter component content in output, got: %s", html)
	}
}

func TestIslandWithLayout(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Layout should be applied.
	if !strings.Contains(html, "<html>") {
		t.Fatalf("expected layout <html>, got: %s", html)
	}
	// Inline boot script should be injected before </body>.
	if !strings.Contains(html, `<script type="module">`) {
		t.Fatalf("expected inline module script, got: %s", html)
	}
	// Modulepreload hints should be present for eagerly-loaded islands.
	if !strings.Contains(html, `<link rel="modulepreload"`) {
		t.Fatalf("expected modulepreload link, got: %s", html)
	}
	// Script should appear before </body>.
	scriptIdx := strings.Index(html, `<script type="module">`)
	bodyIdx := strings.Index(html, "</body>")
	if scriptIdx < 0 || bodyIdx < 0 || scriptIdx > bodyIdx {
		t.Fatalf("boot script should be before </body>, got: %s", html)
	}
}

func TestIslandHtmxSkipsScript(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET / with HX-Request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// htmx requests skip layout, so no script injection.
	if strings.Contains(html, `<script type="module">`) {
		t.Fatalf("htmx request should NOT have boot script, got: %s", html)
	}
	// But island markers should still be present.
	if !strings.Contains(html, "<dark-island") {
		t.Fatalf("expected <dark-island> marker even in htmx response, got: %s", html)
	}
}

// extractChunkURL extracts an island chunk URL from the page's inline manifest.
func extractChunkURL(t *testing.T, html string) string {
	t.Helper()
	// Find __dark_manifest JSON in the inline script.
	prefix := "var __dark_manifest = "
	idx := strings.Index(html, prefix)
	if idx < 0 {
		t.Fatalf("no __dark_manifest found in HTML")
	}
	start := idx + len(prefix)
	end := strings.Index(html[start:], ";")
	if end < 0 {
		t.Fatalf("no semicolon after __dark_manifest")
	}
	manifestJSON := html[start : start+end]

	var manifest map[string]string
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("failed to parse manifest JSON %q: %v", manifestJSON, err)
	}
	for _, url := range manifest {
		return url // return the first chunk URL
	}
	t.Fatalf("manifest is empty")
	return ""
}

func TestIslandClientBundleServed(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// Get the page to find the chunk URL.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	chunkURL := extractChunkURL(t, string(body))

	// Fetch the chunk.
	resp, err = http.Get(srv.URL + chunkURL)
	if err != nil {
		t.Fatalf("GET %s: %v", chunkURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("content-type: got %s, want javascript", ct)
	}
}

func TestIslandClientBundleContent(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithDevMode(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Island("counter", "island_counter.tsx")

	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Bundle Test", "count": 0}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// Get the page HTML.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	// Inline boot script should contain hydration and island selector code.
	if !strings.Contains(html, "dark-island") || !strings.Contains(html, "data-name") {
		t.Fatalf("inline boot script should contain island selector code")
	}
	if !strings.Contains(html, "import(") {
		t.Fatalf("inline boot script should use dynamic import()")
	}

	// Fetch the island chunk and verify it contains hydrate.
	chunkURL := extractChunkURL(t, html)
	resp, err = http.Get(srv.URL + chunkURL)
	if err != nil {
		t.Fatalf("GET %s: %v", chunkURL, err)
	}
	defer resp.Body.Close()

	chunkBody, _ := io.ReadAll(resp.Body)
	js := string(chunkBody)

	if !strings.Contains(js, "hydrate") {
		t.Fatalf("island chunk should contain 'hydrate', got length %d", len(js))
	}
}

func TestHydrationRuntimeContainsHtmxHandler(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithDevMode(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Island("counter", "island_counter.tsx")
	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Htmx Test", "count": 0}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// The htmx handler is now in the inline boot script in the page HTML.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "htmx:afterSettle") {
		t.Fatalf("inline boot script should contain htmx:afterSettle listener")
	}
	if !strings.Contains(html, "data-hydrated") {
		t.Fatalf("inline boot script should contain data-hydrated guard")
	}
}

func TestHydrationRuntimeContainsErrorBoundary(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithDevMode(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Island("counter", "island_counter.tsx")
	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Error Test", "count": 0}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// Error boundary is now in the per-island chunk.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	chunkURL := extractChunkURL(t, html)
	resp, err = http.Get(srv.URL + chunkURL)
	if err != nil {
		t.Fatalf("GET %s: %v", chunkURL, err)
	}
	defer resp.Body.Close()

	chunkBody, _ := io.ReadAll(resp.Body)
	js := string(chunkBody)

	if !strings.Contains(js, "hydration error") {
		t.Fatalf("island chunk should contain error boundary try-catch")
	}
	// In dev mode, the error overlay HTML should be included in the chunk.
	if !strings.Contains(js, "Island Error") {
		t.Fatalf("island chunk should contain dev error overlay in dev mode")
	}
}

func TestIslandSSRNoHydratedAttr(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Check that <dark-island> markers in the SSR output don't have data-hydrated.
	// The inline boot script will contain the string "data-hydrated" in JS code,
	// so we check only the part before the script tag.
	scriptIdx := strings.Index(html, "<script")
	htmlBeforeScript := html
	if scriptIdx >= 0 {
		htmlBeforeScript = html[:scriptIdx]
	}

	if strings.Contains(htmlBeforeScript, "data-hydrated") {
		t.Fatalf("SSR output should NOT contain data-hydrated attribute on island markers, got: %s", htmlBeforeScript)
	}
}

func TestIslandLoadingStrategies(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Island("counter", "island_counter.tsx")

	// Use a simple inline component that tests loading strategies.
	// We need a page component that uses island() with different load options.
	// For this test, we use page_with_island.tsx which defaults to 'load'.
	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Load Test", "count": 0}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Default loading strategy should be "load".
	if !strings.Contains(html, `data-load="load"`) {
		t.Fatalf("expected data-load=\"load\" for default strategy, got: %s", html)
	}
}

func TestIslandPerIslandChunks(t *testing.T) {
	app := newIslandApp(t)
	defer app.Close()

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	// Manifest should contain the "counter" island.
	if !strings.Contains(html, `"counter"`) {
		t.Fatalf("manifest should contain 'counter' island, got: %s", html)
	}

	// Chunk URL should be content-hashed.
	chunkURL := extractChunkURL(t, html)
	if !strings.Contains(chunkURL, "counter-") {
		t.Fatalf("chunk URL should contain island name, got: %s", chunkURL)
	}
}
