package dark

import "net/http"

type authConfig struct {
	sessionKey string
	loginURL   string
	check      func(*Session) bool
}

func defaultAuthConfig() authConfig {
	return authConfig{
		sessionKey: "user",
		loginURL:   "/login",
	}
}

// AuthOption configures the RequireAuth middleware.
type AuthOption func(*authConfig)

// AuthSessionKey sets the session key to check for authentication (default "user").
func AuthSessionKey(key string) AuthOption {
	return func(c *authConfig) { c.sessionKey = key }
}

// AuthLoginURL sets the URL to redirect unauthenticated users to (default "/login").
func AuthLoginURL(url string) AuthOption {
	return func(c *authConfig) { c.loginURL = url }
}

// AuthCheck sets a custom function to determine if a session is authenticated.
// When set, this overrides the default session key check.
func AuthCheck(fn func(*Session) bool) AuthOption {
	return func(c *authConfig) { c.check = fn }
}

// RequireAuth returns a middleware that redirects unauthenticated users to the login page.
// It requires the Sessions middleware to be applied first.
//
// Usage:
//
//	app.Group("/admin", "layouts/admin.tsx", func(g *dark.Group) {
//	    g.Use(dark.RequireAuth())
//	    g.Get("/dashboard", dark.Route{...})
//	})
func RequireAuth(opts ...AuthOption) MiddlewareFunc {
	cfg := defaultAuthConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := r.Context().Value(sessionContextKey).(*Session)
			if !ok {
				hxAwareRedirect(w, r, cfg.loginURL)
				return
			}

			authenticated := false
			if cfg.check != nil {
				authenticated = cfg.check(sess)
			} else {
				authenticated = sess.Get(cfg.sessionKey) != nil
			}

			if !authenticated {
				hxAwareRedirect(w, r, cfg.loginURL)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
