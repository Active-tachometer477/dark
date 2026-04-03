package dark

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGroupUseMiddleware(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret")))

	// Public route — no group middleware.
	app.APIGet("/public", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"page": "public"})
		},
	})

	// Protected group with a blocking middleware.
	app.Group("/admin", "", func(g *Group) {
		g.Use(RequireAuth())

		g.APIGet("/dashboard", APIRoute{
			Handler: func(ctx Context) error {
				return ctx.JSON(200, map[string]any{"page": "dashboard"})
			},
		})
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Public route should work.
	resp, _ := client.Get(srv.URL + "/public")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "public") {
		t.Fatalf("expected public page, got: %s", body)
	}

	// Protected route without auth should redirect.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, _ = client.Get(srv.URL + "/admin/dashboard")
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("expected 302 redirect, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %s", loc)
	}
}

func TestRequireAuthPassesAuthenticated(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret")))

	app.APIPost("/login", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("user", "alice")
			return ctx.JSON(200, nil)
		},
	})

	app.Group("/admin", "", func(g *Group) {
		g.Use(RequireAuth())

		g.APIGet("/dashboard", APIRoute{
			Handler: func(ctx Context) error {
				return ctx.JSON(200, map[string]any{"user": ctx.Session().Get("user")})
			},
		})
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Login first.
	resp, _ := client.Post(srv.URL+"/login", "", nil)
	resp.Body.Close()

	// Now access protected route.
	resp, _ = client.Get(srv.URL + "/admin/dashboard")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "alice") {
		t.Fatalf("expected user alice, got: %s", body)
	}
}

func TestRequireAuthCustomCheck(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret")))

	app.APIPost("/login", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("role", "admin")
			return ctx.JSON(200, nil)
		},
	})

	app.Group("/admin", "", func(g *Group) {
		g.Use(RequireAuth(AuthCheck(func(s *Session) bool {
			return s.Get("role") == "admin"
		})))

		g.APIGet("/panel", APIRoute{
			Handler: func(ctx Context) error {
				return ctx.JSON(200, map[string]any{"ok": true})
			},
		})
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Without login — redirect.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, _ := client.Get(srv.URL + "/admin/panel")
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	// Login with role=admin.
	client.CheckRedirect = nil
	resp, _ = client.Post(srv.URL+"/login", "", nil)
	resp.Body.Close()

	// Now access — should work.
	resp, _ = client.Get(srv.URL + "/admin/panel")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "true") {
		t.Fatalf("expected ok:true, got: %s", body)
	}
}

func TestNestedGroupInheritsMiddleware(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret")))

	app.Group("/admin", "", func(g *Group) {
		g.Use(RequireAuth())

		g.APIGet("/top", APIRoute{
			Handler: func(ctx Context) error {
				return ctx.JSON(200, map[string]any{"page": "top"})
			},
		})

		g.Group("/settings", "", func(sg *Group) {
			sg.APIGet("/profile", APIRoute{
				Handler: func(ctx Context) error {
					return ctx.JSON(200, map[string]any{"page": "profile"})
				},
			})
		})
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Both should redirect — middleware inherited by nested group.
	resp, _ := client.Get(srv.URL + "/admin/top")
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("/admin/top: expected 302, got %d", resp.StatusCode)
	}

	resp, _ = client.Get(srv.URL + "/admin/settings/profile")
	resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("/admin/settings/profile: expected 302, got %d", resp.StatusCode)
	}
}

func TestRequireAuthHtmxRedirect(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret")))

	app.Group("/admin", "", func(g *Group) {
		g.Use(RequireAuth())

		g.APIGet("/data", APIRoute{
			Handler: func(ctx Context) error {
				return ctx.JSON(200, map[string]any{"ok": true})
			},
		})
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/admin/data", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("htmx redirect should return 200, got %d", resp.StatusCode)
	}
	if hxRedirect := resp.Header.Get("HX-Redirect"); hxRedirect != "/login" {
		t.Fatalf("expected HX-Redirect: /login, got %s", hxRedirect)
	}
}
