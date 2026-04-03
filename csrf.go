package dark

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// csrfTokenKey is the request-scoped key for the CSRF token, used by both
// the CSRF middleware (csrf.go) and the props injection (dark.go).
const csrfTokenKey = "_csrfToken"

// CSRFOption configures the CSRF middleware.
type CSRFOption func(*csrfConfig)

type csrfConfig struct {
	fieldName  string
	headerName string
	sessionKey string
}

func defaultCSRFConfig() csrfConfig {
	return csrfConfig{
		fieldName:  "_csrf",
		headerName: "X-CSRF-Token",
		sessionKey: "_csrf_token",
	}
}

// CSRFFieldName sets the form field name for the CSRF token.
func CSRFFieldName(name string) CSRFOption {
	return func(c *csrfConfig) { c.fieldName = name }
}

// CSRFHeaderName sets the header name for the CSRF token.
func CSRFHeaderName(name string) CSRFOption {
	return func(c *csrfConfig) { c.headerName = name }
}

// CSRF returns a middleware that provides CSRF protection.
// Requires the Sessions middleware to be applied first.
//
// On GET/HEAD/OPTIONS requests, a token is generated (if not already in the session)
// and stored in the request context for automatic injection into rendered pages.
//
// On state-mutating requests (POST/PUT/DELETE/PATCH), the token is validated from
// either the X-CSRF-Token header or the _csrf form field.
//
// Dark-specific integration:
//   - The token is automatically added to Loader props as _csrfToken
//   - A <meta name="csrf-token"> tag and htmx config script are injected into HTML
func CSRF(opts ...CSRFOption) MiddlewareFunc {
	cfg := defaultCSRFConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := r.Context().Value(sessionContextKey).(*Session)
			if !ok {
				http.Error(w, "CSRF middleware requires Sessions middleware", http.StatusInternalServerError)
				return
			}

			// Get or create token.
			token, _ := sess.Get(cfg.sessionKey).(string)
			if token == "" {
				token = generateCSRFToken()
				sess.Set(cfg.sessionKey, token)
			}

			// Store token in request context for props injection.
			r = SetValue(r, csrfTokenKey, token)

			// Safe methods: pass through.
			if isSafeMethod(r.Method) {
				next.ServeHTTP(&csrfResponseWriter{
					ResponseWriter: w,
					token:          token,
					headerName:     cfg.headerName,
				}, r)
				return
			}

			// Validate token on state-mutating methods.
			clientToken := r.Header.Get(cfg.headerName)
			if clientToken == "" {
				if err := r.ParseForm(); err == nil {
					clientToken = r.FormValue(cfg.fieldName)
				}
			}

			if clientToken == "" || clientToken != token {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`<div class="dark-csrf-error">CSRF token invalid</div>`))
				} else {
					http.Error(w, "Forbidden - CSRF token invalid", http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(&csrfResponseWriter{
				ResponseWriter: w,
				token:          token,
				headerName:     cfg.headerName,
			}, r)
		})
	}
}

// csrfResponseWriter injects CSRF meta tag and htmx config into HTML responses.
type csrfResponseWriter struct {
	http.ResponseWriter
	token      string
	headerName string
	injected   bool
}

func (cw *csrfResponseWriter) Write(b []byte) (int, error) {
	if !cw.injected {
		ct := cw.Header().Get("Content-Type")
		if strings.HasPrefix(ct, "text/html") {
			cw.injected = true
			html := string(b)
			injected := injectCSRF(html, cw.token, cw.headerName)
			return cw.ResponseWriter.Write([]byte(injected))
		}
	}
	return cw.ResponseWriter.Write(b)
}

func (cw *csrfResponseWriter) Flush() {
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (cw *csrfResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

func injectCSRF(html, token, headerName string) string {
	meta := `<meta name="csrf-token" content="` + token + `">`
	// htmx auto-configuration: attach CSRF token to all htmx requests.
	script := `<script>document.addEventListener("htmx:configRequest",function(e){e.detail.headers["` + headerName + `"]=document.querySelector('meta[name="csrf-token"]').content});</script>`

	tag := meta + "\n" + script
	return insertBeforeTag(html, "</head>", tag)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
