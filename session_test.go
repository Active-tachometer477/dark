package dark

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Signed cookie codec tests ---

func TestEncodeDecodeSigned(t *testing.T) {
	secret := []byte("test-secret")
	data := map[string]any{"user": "alice", "role": "admin"}

	encoded, err := encodeSignedCookie(data, secret)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := decodeSignedCookie(encoded, secret)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded["user"] != "alice" || decoded["role"] != "admin" {
		t.Fatalf("unexpected decoded data: %v", decoded)
	}
}

func TestDecodeSignedTampered(t *testing.T) {
	secret := []byte("test-secret")
	data := map[string]any{"user": "alice"}

	encoded, err := encodeSignedCookie(data, secret)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Tamper with the payload.
	tampered := "x" + encoded
	if _, err := decodeSignedCookie(tampered, secret); err == nil {
		t.Fatal("expected error for tampered cookie")
	}
}

func TestDecodeSignedWrongSecret(t *testing.T) {
	data := map[string]any{"user": "alice"}

	encoded, err := encodeSignedCookie(data, []byte("secret-a"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if _, err := decodeSignedCookie(encoded, []byte("secret-b")); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestDecodeSignedMalformed(t *testing.T) {
	secret := []byte("test-secret")
	for _, input := range []string{"", "noperiod", "a.b.c", "valid-ish.!!invalid-base64!!"} {
		if _, err := decodeSignedCookie(input, secret); err == nil {
			t.Fatalf("expected error for malformed input %q", input)
		}
	}
}

// --- Session struct tests ---

func TestSessionGetSetDelete(t *testing.T) {
	s := &Session{data: make(map[string]any)}

	if s.Get("x") != nil {
		t.Fatal("expected nil for missing key")
	}

	s.Set("x", 42)
	if s.Get("x") != 42 {
		t.Fatalf("expected 42, got %v", s.Get("x"))
	}

	s.Delete("x")
	if s.Get("x") != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestSessionClearUnit(t *testing.T) {
	s := &Session{data: map[string]any{"a": 1, "b": 2}}
	s.Clear()
	if s.Get("a") != nil || s.Get("b") != nil {
		t.Fatal("expected empty after clear")
	}
	if !s.modified {
		t.Fatal("expected modified after clear")
	}
}

func TestSessionFlash(t *testing.T) {
	s := &Session{data: make(map[string]any)}

	s.Flash("notice", "hello")
	s.Flash("error", "oops")

	// Simulate middleware: extract flashes for next request.
	encoded, _ := encodeSignedCookie(s.data, []byte("secret"))
	data, _ := decodeSignedCookie(encoded, []byte("secret"))

	s2 := &Session{data: data}
	if f, ok := s2.data["_flash"]; ok {
		if fm, ok := f.(map[string]any); ok {
			s2.flashes = fm
		}
		delete(s2.data, "_flash")
	}

	flashes := s2.Flashes()
	if flashes == nil {
		t.Fatal("expected flashes")
	}
	if flashes["notice"] != "hello" || flashes["error"] != "oops" {
		t.Fatalf("unexpected flashes: %v", flashes)
	}

	// Second read should return nil.
	if s2.Flashes() != nil {
		t.Fatal("expected nil on second Flashes() call")
	}
}

func TestSessionModifiedFlag(t *testing.T) {
	s := &Session{data: map[string]any{"x": 1}}

	s.Get("x")
	if s.modified {
		t.Fatal("Get should not set modified")
	}

	s.Set("y", 2)
	if !s.modified {
		t.Fatal("Set should set modified")
	}
}

// --- Integration tests ---

func newSessionClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func TestContextCookieHelpers(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.APIGet("/set", APIRoute{
		Handler: func(ctx Context) error {
			ctx.SetCookie("theme", "dark", CookieMaxAge(3600))
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})
	app.APIGet("/get", APIRoute{
		Handler: func(ctx Context) error {
			val, err := ctx.GetCookie("theme")
			if err != nil {
				return ctx.JSON(200, map[string]any{"value": ""})
			}
			return ctx.JSON(200, map[string]any{"value": val})
		},
	})
	app.APIGet("/del", APIRoute{
		Handler: func(ctx Context) error {
			ctx.DeleteCookie("theme")
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Set cookie.
	resp, _ := client.Get(srv.URL + "/set")
	resp.Body.Close()

	// Read cookie back.
	resp, _ = client.Get(srv.URL + "/get")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"dark"`) {
		t.Fatalf("expected cookie value 'dark', got: %s", body)
	}

	// Delete cookie.
	resp, _ = client.Get(srv.URL + "/del")
	resp.Body.Close()

	// Verify cookie is gone.
	resp, _ = client.Get(srv.URL + "/get")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"dark"`) {
		t.Fatalf("expected cookie to be deleted, got: %s", body)
	}
}

func TestSessionMiddlewareBasic(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret")))

	app.APIPost("/login", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("user", "alice")
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})
	app.APIGet("/me", APIRoute{
		Handler: func(ctx Context) error {
			user := ctx.Session().Get("user")
			return ctx.JSON(200, map[string]any{"user": user})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Login.
	resp, _ := client.Post(srv.URL+"/login", "", nil)
	resp.Body.Close()

	// Read session.
	resp, _ = client.Get(srv.URL + "/me")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"alice"`) {
		t.Fatalf("expected session user 'alice', got: %s", body)
	}
}

func TestSessionFlashAcrossRequests(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret")))

	app.APIPost("/flash", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Flash("notice", "saved!")
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})
	app.APIGet("/read", APIRoute{
		Handler: func(ctx Context) error {
			flashes := ctx.Session().Flashes()
			return ctx.JSON(200, map[string]any{"flashes": flashes})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	// Set flash.
	resp, _ := client.Post(srv.URL+"/flash", "", nil)
	resp.Body.Close()

	// Read flash — should be present.
	resp, _ = client.Get(srv.URL + "/read")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"saved!"`) {
		t.Fatalf("expected flash 'saved!', got: %s", body)
	}

	// Read again — should be gone.
	resp, _ = client.Get(srv.URL + "/read")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"saved!"`) {
		t.Fatalf("flash should be consumed after first read, got: %s", body)
	}
}

func TestSessionTamperedCookieStartsFresh(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret")))

	app.APIGet("/me", APIRoute{
		Handler: func(ctx Context) error {
			user := ctx.Session().Get("user")
			return ctx.JSON(200, map[string]any{"user": user})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	// Send a tampered session cookie.
	req, _ := http.NewRequest("GET", srv.URL+"/me", nil)
	req.AddCookie(&http.Cookie{Name: "_dark_session", Value: "tampered.garbage"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Should get null user (fresh session), not an error.
	if !strings.Contains(string(body), `"user":null`) {
		t.Fatalf("expected fresh session with null user, got: %s", body)
	}
}

func TestSessionNoMiddlewarePanics(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Recover())

	app.APIGet("/boom", APIRoute{
		Handler: func(ctx Context) error {
			_ = ctx.Session() // should panic
			return nil
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/boom")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 from panic, got %d", resp.StatusCode)
	}
}

func TestSessionCookieDefaults(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret")))

	app.APIPost("/set", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("x", 1)
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, _ := http.Post(srv.URL+"/set", "", nil)
	resp.Body.Close()

	// Check Set-Cookie header for defaults.
	setCookie := resp.Header.Get("Set-Cookie")
	if !strings.Contains(setCookie, "_dark_session=") {
		t.Fatalf("expected _dark_session cookie, got: %s", setCookie)
	}
	if !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("expected HttpOnly, got: %s", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax, got: %s", setCookie)
	}
}

func TestSessionCustomOptions(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("secret"),
		SessionName("my_sess"),
		SessionMaxAge(7200),
		SessionPath("/app"),
	))

	app.APIPost("/set", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("x", 1)
			return ctx.JSON(200, map[string]any{"ok": true})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	resp, _ := http.Post(srv.URL+"/set", "", nil)
	resp.Body.Close()

	setCookie := resp.Header.Get("Set-Cookie")
	if !strings.Contains(setCookie, "my_sess=") {
		t.Fatalf("expected my_sess cookie, got: %s", setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=7200") {
		t.Fatalf("expected Max-Age=7200, got: %s", setCookie)
	}
	if !strings.Contains(setCookie, "Path=/app") {
		t.Fatalf("expected Path=/app, got: %s", setCookie)
	}
}

func TestSessionClear(t *testing.T) {
	app, err := New(WithPoolSize(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer app.Close()

	app.Use(Sessions([]byte("test-secret")))

	app.APIPost("/login", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Set("user", "alice")
			return ctx.JSON(200, nil)
		},
	})
	app.APIPost("/logout", APIRoute{
		Handler: func(ctx Context) error {
			ctx.Session().Clear()
			return ctx.JSON(200, nil)
		},
	})
	app.APIGet("/me", APIRoute{
		Handler: func(ctx Context) error {
			return ctx.JSON(200, map[string]any{"user": ctx.Session().Get("user")})
		},
	})

	srv := httptest.NewServer(app.MustHandler())
	defer srv.Close()

	client := newSessionClient(t)

	client.Post(srv.URL+"/login", "", nil)

	resp, _ := client.Get(srv.URL + "/me")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"alice"`) {
		t.Fatalf("expected alice, got: %s", body)
	}

	client.Post(srv.URL+"/logout", "", nil)

	resp, _ = client.Get(srv.URL + "/me")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"alice"`) {
		t.Fatalf("expected session cleared, got: %s", body)
	}
}
