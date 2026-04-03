package dark

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func csrfTestClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func csrfReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func extractCSRFToken(t *testing.T, body string) string {
	t.Helper()
	idx := strings.Index(body, `<meta name="csrf-token" content="`)
	if idx < 0 {
		t.Fatal("CSRF meta tag not found in response")
	}
	start := idx + len(`<meta name="csrf-token" content="`)
	end := strings.Index(body[start:], `"`)
	return body[start : start+end]
}

func TestCSRFBlocksPostWithoutToken(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret-key-32bytes-long!!")))
	app.Use(CSRF())

	app.Post("/submit", Route{
		Action: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/submit", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestCSRFAllowsGetAndInjectsMetaTag(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret-key-32bytes-long!!")))
	app.Use(CSRF())

	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"message": "Hello"}, nil
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := csrfReadBody(t, resp)
	if !strings.Contains(body, `<meta name="csrf-token"`) {
		t.Error("expected CSRF meta tag in response")
	}
	if !strings.Contains(body, `htmx:configRequest`) {
		t.Error("expected htmx CSRF config script in response")
	}
}

func TestCSRFAllowsPostWithValidHeaderToken(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret-key-32bytes-long!!")))
	app.Use(CSRF())

	app.Get("/form", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"message": "form"}, nil
		},
	})

	app.Post("/submit", Route{
		Action: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := csrfTestClient()

	// GET to establish session and get token.
	getResp, err := client.Get(srv.URL + "/form")
	if err != nil {
		t.Fatal(err)
	}
	body := csrfReadBody(t, getResp)
	token := extractCSRFToken(t, body)

	// POST with valid token in header (cookie jar handles session automatically).
	req, _ := http.NewRequest("POST", srv.URL+"/submit", nil)
	req.Header.Set("X-CSRF-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCSRFAllowsPostWithFormField(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret-key-32bytes-long!!")))
	app.Use(CSRF())

	app.Get("/form", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			return map[string]any{"message": "form"}, nil
		},
	})

	app.Post("/submit", Route{
		Action: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := csrfTestClient()

	// GET to get token.
	getResp, err := client.Get(srv.URL + "/form")
	if err != nil {
		t.Fatal(err)
	}
	body := csrfReadBody(t, getResp)
	token := extractCSRFToken(t, body)

	// POST with token as form field.
	form := url.Values{"_csrf": {token}}
	resp, err := client.Post(srv.URL+"/submit", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCSRFTokenInProps(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret-key-32bytes-long!!")))
	app.Use(CSRF())

	var capturedToken any
	app.Get("/", Route{
		Component: "simple.tsx",
		Loader: func(ctx Context) (any, error) {
			capturedToken = ctx.Get("_csrfToken")
			return map[string]any{"message": "test"}, nil
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	http.Get(srv.URL + "/")

	if capturedToken == nil || capturedToken == "" {
		t.Error("expected CSRF token to be available via ctx.Get")
	}
}
