package dark

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDevReloadSSEEndpoint(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithDevMode(true),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{Component: "simple.tsx"})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// The SSE endpoint should respond with text/event-stream.
	resp, err := http.Get(srv.URL + "/_dark/reload")
	if err != nil {
		t.Fatalf("GET /_dark/reload: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got: %s", ct)
	}

	// Read the initial "connected" message.
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read SSE: %v", err)
	}
	if !strings.Contains(line, "connected") {
		t.Fatalf("expected 'connected' message, got: %s", line)
	}
}

func TestDevReloadNoSSEInProdMode(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{Component: "simple.tsx"})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/_dark/reload")
	if err != nil {
		t.Fatalf("GET /_dark/reload: %v", err)
	}
	defer resp.Body.Close()

	// In prod mode, /_dark/reload should 404 since reloader is nil.
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 in prod mode, got: %d", resp.StatusCode)
	}
}

func TestDevReloadCacheInvalidation(t *testing.T) {
	// Create a temp dir with a TSX file.
	tmpDir := t.TempDir()
	tsxPath := filepath.Join(tmpDir, "test.tsx")
	if err := os.WriteFile(tsxPath, []byte(`import { h } from 'preact';
export default function Test() { return <div>version1</div>; }`), 0o644); err != nil {
		t.Fatalf("write tsx: %v", err)
	}

	app, err := New(
		WithTemplateDir(tmpDir),
		WithDevMode(true),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "test.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// First request.
	resp1, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if !strings.Contains(string(body1), "version1") {
		t.Fatalf("expected 'version1', got: %s", string(body1))
	}

	// Update the file.
	time.Sleep(10 * time.Millisecond) // Ensure mtime changes.
	if err := os.WriteFile(tsxPath, []byte(`import { h } from 'preact';
export default function Test() { return <div>version2</div>; }`), 0o644); err != nil {
		t.Fatalf("write tsx: %v", err)
	}

	// Wait for the watcher to pick up the change.
	time.Sleep(300 * time.Millisecond)

	// Second request should get the updated content.
	resp2, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	if !strings.Contains(string(body2), "version2") {
		t.Fatalf("expected 'version2' after cache invalidation, got: %s", string(body2))
	}
}

func TestIslandClientBundleNoCacheInDevMode(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithDevMode(true),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Island("counter", "island_counter.tsx")
	app.Get("/", Route{
		Component: "page_with_island.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"title": "Test", "count": 0}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// Fetch the page to find a chunk URL.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	chunkURL := extractChunkURL(t, string(body))

	// Fetch the chunk and verify cache headers.
	resp, err = http.Get(srv.URL + chunkURL)
	if err != nil {
		t.Fatalf("GET %s: %v", chunkURL, err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-cache" {
		t.Fatalf("expected no-cache in dev mode, got: %s", cc)
	}
}
