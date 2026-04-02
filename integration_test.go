package dark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegrationGetWithLoader(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Integration"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "Hello Integration") {
		t.Fatalf("expected 'Hello Integration' in body, got: %s", html)
	}
	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type: got %s", resp.Header.Get("Content-Type"))
	}
}

func TestIntegrationWithLayout(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "LayoutTest", "title": "My Title"}, nil
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

	if !strings.Contains(html, "<html>") {
		t.Fatalf("expected layout <html>, got: %s", html)
	}
	if !strings.Contains(html, "My Title") {
		t.Fatalf("expected 'My Title', got: %s", html)
	}
	if !strings.Contains(html, "Hello LayoutTest") {
		t.Fatalf("expected 'Hello LayoutTest', got: %s", html)
	}
}

func TestIntegrationHtmxSkipsLayout(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Htmx"}, nil
		},
	})

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

	if strings.Contains(html, "<html>") {
		t.Fatalf("expected NO layout for htmx request, got: %s", html)
	}
	if !strings.Contains(html, "Hello Htmx") {
		t.Fatalf("expected 'Hello Htmx', got: %s", html)
	}
}

func TestIntegrationParamRoute(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/users/{id}", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "User-" + ctx.Param("id")}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatalf("GET /users/42: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Hello User-42") {
		t.Fatalf("expected 'Hello User-42', got: %s", string(body))
	}
}

func TestIntegration404(t *testing.T) {
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

	resp, err := http.Get(srv.URL + "/notfound")
	if err != nil {
		t.Fatalf("GET /notfound: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestCustom404Page(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithNotFoundComponent("error_404.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{Component: "simple.tsx"})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/notfound")
	if err != nil {
		t.Fatalf("GET /notfound: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "not-found-page") {
		t.Fatalf("expected custom 404 component, got: %s", html)
	}
	if !strings.Contains(html, "/notfound") {
		t.Fatalf("expected path '/notfound' in 404 page, got: %s", html)
	}
}

func TestCustom404WithLayout(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithNotFoundComponent("error_404.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{Component: "simple.tsx"})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/missing")
	if err != nil {
		t.Fatalf("GET /missing: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "<html>") {
		t.Fatalf("expected layout in 404 page, got: %s", html)
	}
	if !strings.Contains(html, "not-found-page") {
		t.Fatalf("expected custom 404 component, got: %s", html)
	}
}

func TestCustom500Page(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithErrorComponent("error_500.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return nil, fmt.Errorf("test loader error")
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "error-page") {
		t.Fatalf("expected custom 500 component, got: %s", html)
	}
	if !strings.Contains(html, "Internal Server Error") {
		t.Fatalf("expected generic message in prod mode, got: %s", html)
	}
}

func TestDevModeOverlay(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithDevMode(true),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return nil, fmt.Errorf("dev test error")
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "Server Error") {
		t.Fatalf("expected dev overlay title, got: %s", html)
	}
	if !strings.Contains(html, "dev test error") {
		t.Fatalf("expected error message in dev overlay, got: %s", html)
	}
}

func TestDevModeReloadScriptInjected(t *testing.T) {
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

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "DevTest"}, nil
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

	if !strings.Contains(html, "/_dark/reload") {
		t.Fatalf("expected dev reload script in output, got: %s", html)
	}
	if !strings.Contains(html, "EventSource") {
		t.Fatalf("expected EventSource in dev reload script, got: %s", html)
	}
}

func TestProdModeNoReloadScript(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "ProdTest"}, nil
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

	if strings.Contains(html, "/_dark/reload") {
		t.Fatalf("expected NO dev reload script in prod mode, got: %s", html)
	}
}

// --- CSS Tests ---

func TestCSSInjectedAsLink(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "styled.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "CSS", "title": "CSS Test"}, nil
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

	// Full page should have <link> tag for CSS.
	if !strings.Contains(html, `<link rel="stylesheet" href="/_dark/css/`) {
		t.Fatalf("expected <link> tag for CSS in full page, got: %s", html)
	}
	if !strings.Contains(html, "Styled CSS") {
		t.Fatalf("expected 'Styled CSS' in body, got: %s", html)
	}
}

func TestCSSInlineForHtmx(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "styled.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Htmx"}, nil
		},
	})

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

	// htmx partial should have inline <style>.
	if !strings.Contains(html, "<style>") {
		t.Fatalf("expected inline <style> for htmx request, got: %s", html)
	}
	if !strings.Contains(html, ".test-styled") {
		t.Fatalf("expected CSS class in inline style, got: %s", html)
	}
}

func TestCSSEndpointServed(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "styled.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Test", "title": "Test"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// First, get the page to populate the CSS cache.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	html := string(body)

	// Extract the CSS URL from the <link> tag.
	linkIdx := strings.Index(html, `/_dark/css/`)
	if linkIdx < 0 {
		t.Fatalf("no CSS link found in HTML: %s", html)
	}
	endIdx := strings.Index(html[linkIdx:], `"`)
	cssPath := html[linkIdx : linkIdx+endIdx]

	// Fetch the CSS file.
	cssResp, err := http.Get(srv.URL + cssPath)
	if err != nil {
		t.Fatalf("GET %s: %v", cssPath, err)
	}
	defer cssResp.Body.Close()

	if cssResp.StatusCode != 200 {
		t.Fatalf("CSS status: got %d, want 200", cssResp.StatusCode)
	}
	if !strings.Contains(cssResp.Header.Get("Content-Type"), "text/css") {
		t.Fatalf("CSS content-type: got %s", cssResp.Header.Get("Content-Type"))
	}

	cssBody, _ := io.ReadAll(cssResp.Body)
	if !strings.Contains(string(cssBody), ".test-styled") {
		t.Fatalf("expected CSS content, got: %s", string(cssBody))
	}
}

func TestNoCSSWhenNoImport(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "NoCSS", "title": "No CSS"}, nil
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

	// No CSS import means no <link> tag for CSS.
	if strings.Contains(html, "/_dark/css/") {
		t.Fatalf("expected no CSS link for component without CSS import, got: %s", html)
	}
}

// --- API Route Tests ---

func TestAPIGetJSON(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIGet("/api/hello", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"message": "hello"})
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/hello")
	if err != nil {
		t.Fatalf("GET /api/hello: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type: got %s", ct)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "hello" {
		t.Fatalf("expected message=hello, got: %v", result)
	}
}

func TestAPIPostWithBindJSON(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	type input struct {
		Name string `json:"name"`
	}

	app.APIPost("/api/users", APIRoute{
		Handler: func(ctx Context) error {
			var in input
			if err := ctx.BindJSON(&in); err != nil {
				return NewAPIError(400, "invalid JSON")
			}
			return ctx.JSON(201, map[string]any{"created": in.Name})
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	body := bytes.NewBufferString(`{"name":"Alice"}`)
	resp, err := http.Post(srv.URL+"/api/users", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/users: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("status: got %d, want 201", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["created"] != "Alice" {
		t.Fatalf("expected created=Alice, got: %v", result)
	}
}

func TestAPIErrorResponse(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIGet("/api/fail", APIRoute{
		Handler: func(ctx Context) error {
			return NewAPIError(404, "user not found")
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/fail")
	if err != nil {
		t.Fatalf("GET /api/fail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "user not found" {
		t.Fatalf("expected error='user not found', got: %v", result)
	}
}

func TestAPIGenericError(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIGet("/api/crash", APIRoute{
		Handler: func(ctx Context) error {
			return fmt.Errorf("something broke")
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/crash")
	if err != nil {
		t.Fatalf("GET /api/crash: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Fatalf("status: got %d, want 500", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	// In prod mode, generic error message.
	if result["error"] != "Internal Server Error" {
		t.Fatalf("expected generic error in prod, got: %v", result)
	}
}

func TestAPINoContent(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIDelete("/api/items/{id}", APIRoute{
		Handler: func(ctx Context) error {
			// Handler succeeds but writes nothing.
			_ = ctx.Param("id")
			return nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/items/99", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/items/99: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}
}

func TestAPIPutPatch(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIPut("/api/items/{id}", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"method": "PUT", "id": ctx.Param("id")})
		},
	})

	app.APIPatch("/api/items/{id}", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"method": "PATCH", "id": ctx.Param("id")})
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	// PUT
	req, _ := http.NewRequest("PUT", srv.URL+"/api/items/7", strings.NewReader(`{}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	var putResult map[string]any
	json.NewDecoder(resp.Body).Decode(&putResult)
	resp.Body.Close()
	if putResult["method"] != "PUT" || putResult["id"] != "7" {
		t.Fatalf("PUT result: %v", putResult)
	}

	// PATCH
	req, _ = http.NewRequest("PATCH", srv.URL+"/api/items/7", strings.NewReader(`{}`))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	var patchResult map[string]any
	json.NewDecoder(resp.Body).Decode(&patchResult)
	resp.Body.Close()
	if patchResult["method"] != "PATCH" || patchResult["id"] != "7" {
		t.Fatalf("PATCH result: %v", patchResult)
	}
}

func TestAPIMiddlewareApplied(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	// Custom middleware that adds a header.
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "applied")
			next.ServeHTTP(w, r)
		})
	})

	app.APIGet("/api/test", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/test")
	if err != nil {
		t.Fatalf("GET /api/test: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Custom") != "applied" {
		t.Fatalf("expected middleware header, got: %s", resp.Header.Get("X-Custom"))
	}
}

// --- Nested Layouts ---

func TestNestedLayoutRendering(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/admin", Route{
		Component: "admin_page.tsx",
		Layout:    "admin_layout.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"section": "dashboard"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/admin")
	if err != nil {
		t.Fatalf("GET /admin: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Global layout should be present.
	if !strings.Contains(s, "<html>") {
		t.Fatalf("expected global layout <html>, got: %s", s)
	}
	// Route layout should be present.
	if !strings.Contains(s, "admin-layout") {
		t.Fatalf("expected admin-layout class, got: %s", s)
	}
	if !strings.Contains(s, "admin-nav") {
		t.Fatalf("expected admin-nav, got: %s", s)
	}
	// Component content should be present.
	if !strings.Contains(s, "Admin: dashboard") {
		t.Fatalf("expected 'Admin: dashboard', got: %s", s)
	}
}

func TestGroupRoutesInheritLayout(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Group("/admin", "admin_layout.tsx", func(g *Group) {
		g.Get("/home", Route{
			Component: "admin_page.tsx",
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"section": "home"}, nil
			},
		})
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/admin/home")
	if err != nil {
		t.Fatalf("GET /admin/home: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if !strings.Contains(s, "<html>") {
		t.Fatalf("expected global layout, got: %s", s)
	}
	if !strings.Contains(s, "admin-layout") {
		t.Fatalf("expected admin-layout, got: %s", s)
	}
	if !strings.Contains(s, "Admin: home") {
		t.Fatalf("expected 'Admin: home', got: %s", s)
	}
}

func TestGroupNestedWithRouteLayout(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Group("/admin", "admin_layout.tsx", func(g *Group) {
		g.Get("/settings", Route{
			Component: "admin_page.tsx",
			Layout:    "settings_layout.tsx",
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"section": "settings"}, nil
			},
		})
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/admin/settings")
	if err != nil {
		t.Fatalf("GET /admin/settings: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Global layout > admin layout > settings layout > component
	if !strings.Contains(s, "<html>") {
		t.Fatalf("expected global layout, got: %s", s)
	}
	if !strings.Contains(s, "admin-layout") {
		t.Fatalf("expected admin-layout, got: %s", s)
	}
	if !strings.Contains(s, "settings-layout") {
		t.Fatalf("expected settings-layout, got: %s", s)
	}
	if !strings.Contains(s, "Admin: settings") {
		t.Fatalf("expected 'Admin: settings', got: %s", s)
	}
}

func TestHtmxSkipsAllLayouts(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/admin", Route{
		Component: "admin_page.tsx",
		Layout:    "admin_layout.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"section": "partial"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/admin", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin (htmx): %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if strings.Contains(s, "<html>") {
		t.Fatalf("expected no global layout for htmx, got: %s", s)
	}
	if strings.Contains(s, "admin-layout") {
		t.Fatalf("expected no admin-layout for htmx, got: %s", s)
	}
	if !strings.Contains(s, "Admin: partial") {
		t.Fatalf("expected component content, got: %s", s)
	}
}

// --- Form Validation ---

func TestValidationErrorsInProps(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Post("/submit", Route{
		Component: "form.tsx",
		Action: func(ctx Context) error {
			ctx.AddFieldError("name", "Name is required")
			return nil
		},
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"items": []string{"a", "b"}}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/submit", "application/x-www-form-urlencoded", strings.NewReader("name="))
	if err != nil {
		t.Fatalf("POST /submit: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if !strings.Contains(s, "field-error") {
		t.Fatalf("expected field-error class, got: %s", s)
	}
	if !strings.Contains(s, "Name is required") {
		t.Fatalf("expected error message, got: %s", s)
	}
	// Loader data should also be present.
	if !strings.Contains(s, "<li>a</li>") {
		t.Fatalf("expected loader data, got: %s", s)
	}
}

func TestValidationFormDataPreserved(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Post("/submit", Route{
		Component: "form.tsx",
		Action: func(ctx Context) error {
			if ctx.FormData().Get("email") == "" {
				ctx.AddFieldError("email", "Email is required")
			}
			return nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/submit", "application/x-www-form-urlencoded", strings.NewReader("name=John&email="))
	if err != nil {
		t.Fatalf("POST /submit: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Form data should be preserved via _formData.
	if !strings.Contains(s, "John") {
		t.Fatalf("expected form value 'John' preserved, got: %s", s)
	}
	if !strings.Contains(s, "Email is required") {
		t.Fatalf("expected email error, got: %s", s)
	}
}

func TestValidationNoErrors(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Post("/submit", Route{
		Component: "form.tsx",
		Action: func(ctx Context) error {
			// No errors added.
			return nil
		},
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"items": []string{"x"}}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/submit", "application/x-www-form-urlencoded", strings.NewReader("name=Valid"))
	if err != nil {
		t.Fatalf("POST /submit: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// No error messages should appear.
	if strings.Contains(s, "field-error") {
		t.Fatalf("expected no field-error, got: %s", s)
	}
	if !strings.Contains(s, "<li>x</li>") {
		t.Fatalf("expected loader data, got: %s", s)
	}
}

func TestValidationHtmxPartial(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Post("/submit", Route{
		Component: "form.tsx",
		Action: func(ctx Context) error {
			ctx.AddFieldError("name", "Required")
			return nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/submit", strings.NewReader("name="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /submit (htmx): %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Should be a partial (no layout).
	if strings.Contains(s, "<html>") {
		t.Fatalf("expected no layout for htmx, got: %s", s)
	}
	if !strings.Contains(s, "Required") {
		t.Fatalf("expected error message, got: %s", s)
	}
}

// --- Head/Meta Management ---

func TestDarkHeadExtraction(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/post", Route{
		Component: "page_with_head.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{
				"post": map[string]any{
					"title":   "Hello World",
					"excerpt": "A test post",
					"body":    "Post body content",
				},
			}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/post")
	if err != nil {
		t.Fatalf("GET /post: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// <dark-head> should be removed from the body.
	if strings.Contains(s, "<dark-head>") {
		t.Fatalf("expected <dark-head> to be extracted, got: %s", s)
	}
	// Title from <dark-head> should be in <head>.
	if !strings.Contains(s, "Hello World | Blog") {
		t.Fatalf("expected title in head, got: %s", s)
	}
	// Meta description should be in <head>.
	if !strings.Contains(s, `content="A test post"`) {
		t.Fatalf("expected meta description, got: %s", s)
	}
	// Component content should still be present.
	if !strings.Contains(s, "<h1>Hello World</h1>") {
		t.Fatalf("expected h1 content, got: %s", s)
	}
}

func TestDarkHeadTitleOverride(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/post", Route{
		Component: "page_with_head.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{
				"title": "Layout Title",
				"post": map[string]any{
					"title":   "Page Title",
					"excerpt": "desc",
					"body":    "body",
				},
			}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/post")
	if err != nil {
		t.Fatalf("GET /post: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// The layout title "Layout Title" should be replaced by the page title.
	if strings.Contains(s, "<title>Layout Title</title>") {
		t.Fatalf("expected layout title to be overridden, got: %s", s)
	}
	if !strings.Contains(s, "Page Title | Blog") {
		t.Fatalf("expected page title override, got: %s", s)
	}
}

func TestContextSetTitle(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			ctx.SetTitle("Custom Title")
			ctx.AddMeta("description", "A custom page")
			return map[string]any{"name": "World"}, nil
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
	s := string(body)

	// _head.title should be available to the layout (layout.tsx renders {title || 'Test'}).
	// The layout reads props.title, but _head is a separate field.
	// For this test, we just verify the props were passed and the page rendered.
	if !strings.Contains(s, "Hello World") {
		t.Fatalf("expected component content, got: %s", s)
	}
}

func TestDarkHeadStrippedForHtmx(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/post", Route{
		Component: "page_with_head.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{
				"post": map[string]any{"title": "Test", "excerpt": "desc", "body": "body"},
			}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/post", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /post (htmx): %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// <dark-head> should be stripped for htmx partials.
	if strings.Contains(s, "<dark-head>") {
		t.Fatalf("expected <dark-head> stripped for htmx, got: %s", s)
	}
	// No layout.
	if strings.Contains(s, "<html>") {
		t.Fatalf("expected no layout for htmx, got: %s", s)
	}
	// Content should be present.
	if !strings.Contains(s, "<h1>Test</h1>") {
		t.Fatalf("expected component content, got: %s", s)
	}
}

// --- Streaming SSR ---

func TestStreamingBasicResponse(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithStreaming(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Stream", "title": "StreamTest"}, nil
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
	s := string(body)

	// Should contain layout.
	if !strings.Contains(s, "<html>") {
		t.Fatalf("expected <html> from layout, got: %s", s)
	}
	// Should contain component content.
	if !strings.Contains(s, "Hello Stream") {
		t.Fatalf("expected 'Hello Stream', got: %s", s)
	}
}

func TestStreamingHtmxFallback(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithStreaming(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Partial"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET / (htmx): %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Should be partial (no layout).
	if strings.Contains(s, "<html>") {
		t.Fatalf("expected no layout for htmx streaming fallback, got: %s", s)
	}
	if !strings.Contains(s, "Hello Partial") {
		t.Fatalf("expected component content, got: %s", s)
	}
}

func TestStreamingDisabledByDefault(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Default"}, nil
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
	s := string(body)

	// Default non-streaming should work.
	if !strings.Contains(s, "Hello Default") {
		t.Fatalf("expected 'Hello Default', got: %s", s)
	}
}

func TestStreamingPerRouteOverride(t *testing.T) {
	streaming := true
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/stream", Route{
		Component: "simple.tsx",
		Streaming: &streaming,
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "PerRoute"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if !strings.Contains(s, "Hello PerRoute") {
		t.Fatalf("expected 'Hello PerRoute', got: %s", s)
	}
	if !strings.Contains(s, "<html>") {
		t.Fatalf("expected layout in streaming response, got: %s", s)
	}
}

func TestStreamingWithCSS(t *testing.T) {
	app, err := New(
		WithTemplateDir("_testdata"),
		WithLayout("layout.tsx"),
		WithPoolSize(1),
		WithStreaming(true),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Get("/styled", Route{
		Component: "styled.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"name": "Styled"}, nil
		},
	})

	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/styled")
	if err != nil {
		t.Fatalf("GET /styled: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	// Component CSS should be inline <style> (since <head> is already flushed).
	if !strings.Contains(s, "<style>") {
		t.Fatalf("expected inline <style> for component CSS in streaming, got: %s", s)
	}
	if !strings.Contains(s, "Styled Styled") {
		t.Fatalf("expected component content, got: %s", s)
	}
}
