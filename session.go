package dark

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- Cookie helpers ---

type cookieConfig struct {
	path     string
	maxAge   int
	secure   bool
	httpOnly bool
	sameSite http.SameSite
}

func defaultCookieConfig() cookieConfig {
	return cookieConfig{
		path:     "/",
		httpOnly: true,
		sameSite: http.SameSiteLaxMode,
	}
}

// CookieOption configures a cookie set via Context.SetCookie.
type CookieOption func(*cookieConfig)

// CookiePath sets the cookie path (default "/").
func CookiePath(path string) CookieOption { return func(c *cookieConfig) { c.path = path } }

// CookieMaxAge sets the cookie max age in seconds. 0 means session cookie.
func CookieMaxAge(seconds int) CookieOption { return func(c *cookieConfig) { c.maxAge = seconds } }

// CookieSecure sets the Secure flag.
func CookieSecure(enabled bool) CookieOption { return func(c *cookieConfig) { c.secure = enabled } }

// CookieHTTPOnly sets the HttpOnly flag (default true).
func CookieHTTPOnly(enabled bool) CookieOption {
	return func(c *cookieConfig) { c.httpOnly = enabled }
}

// CookieSameSite sets the SameSite attribute (default Lax).
func CookieSameSite(mode http.SameSite) CookieOption {
	return func(c *cookieConfig) { c.sameSite = mode }
}

func setCookie(w http.ResponseWriter, name, value string, opts ...CookieOption) {
	cfg := defaultCookieConfig()
	for _, o := range opts {
		o(&cfg)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     cfg.path,
		MaxAge:   cfg.maxAge,
		Secure:   cfg.secure,
		HttpOnly: cfg.httpOnly,
		SameSite: cfg.sameSite,
	})
}

func getCookie(r *http.Request, name string) (string, error) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

// --- Session ---

type contextKey struct{}

var sessionContextKey = &contextKey{}

// flashKey is the reserved session data key for flash messages.
const flashKey = "_flash"

// Session holds per-request session data backed by a signed cookie.
type Session struct {
	data     map[string]any
	flashes  map[string]any
	modified bool
}

// Get returns the session value for key, or nil if not set.
func (s *Session) Get(key string) any {
	return s.data[key]
}

// Set stores a value in the session.
func (s *Session) Set(key string, value any) {
	s.data[key] = value
	s.modified = true
}

// Delete removes a key from the session.
func (s *Session) Delete(key string) {
	delete(s.data, key)
	s.modified = true
}

// Clear removes all session data.
func (s *Session) Clear() {
	s.data = make(map[string]any)
	s.modified = true
}

// Flash sets a flash message that will be available in the next request.
func (s *Session) Flash(key string, value any) {
	fm, ok := s.data[flashKey].(map[string]any)
	if !ok {
		fm = make(map[string]any)
		s.data[flashKey] = fm
	}
	fm[key] = value
	s.modified = true
}

// Flashes returns and clears all flash messages from the previous request.
// Returns nil if there are no flashes.
func (s *Session) Flashes() map[string]any {
	if len(s.flashes) == 0 {
		return nil
	}
	out := s.flashes
	s.flashes = nil
	return out
}

// --- Session options ---

type sessionConfig struct {
	name     string
	maxAge   int
	path     string
	secure   bool
	httpOnly bool
	sameSite http.SameSite
}

func defaultSessionConfig() sessionConfig {
	return sessionConfig{
		name:     "_dark_session",
		maxAge:   86400,
		path:     "/",
		httpOnly: true,
		sameSite: http.SameSiteLaxMode,
	}
}

// SessionOption configures the session middleware.
type SessionOption func(*sessionConfig)

// SessionName sets the session cookie name (default "_dark_session").
func SessionName(name string) SessionOption { return func(c *sessionConfig) { c.name = name } }

// SessionMaxAge sets the session max age in seconds (default 86400 = 1 day).
func SessionMaxAge(seconds int) SessionOption { return func(c *sessionConfig) { c.maxAge = seconds } }

// SessionPath sets the session cookie path (default "/").
func SessionPath(path string) SessionOption { return func(c *sessionConfig) { c.path = path } }

// SessionSecure sets the Secure flag on the session cookie.
func SessionSecure(enabled bool) SessionOption { return func(c *sessionConfig) { c.secure = enabled } }

// SessionHTTPOnly sets the HttpOnly flag on the session cookie (default true).
func SessionHTTPOnly(enabled bool) SessionOption {
	return func(c *sessionConfig) { c.httpOnly = enabled }
}

// SessionSameSite sets the SameSite attribute on the session cookie (default Lax).
func SessionSameSite(mode http.SameSite) SessionOption {
	return func(c *sessionConfig) { c.sameSite = mode }
}

// --- Signed cookie codec ---

func encodeSignedCookie(data map[string]any, secret []byte) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(jsonBytes)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	result := payload + "." + sig
	if len(result) > 4096 {
		log.Printf("dark: session cookie exceeds 4KB (%d bytes); browsers may reject it", len(result))
	}
	return result, nil
}

func decodeSignedCookie(cookie string, secret []byte) (map[string]any, error) {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return nil, http.ErrNoCookie
	}
	payload, sigStr := parts[0], parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expectedSig := mac.Sum(nil)

	sig, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return nil, http.ErrNoCookie
	}
	if !hmac.Equal(sig, expectedSig) {
		return nil, http.ErrNoCookie
	}

	jsonBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, http.ErrNoCookie
	}
	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, http.ErrNoCookie
	}
	return data, nil
}

// --- Sessions middleware ---

// Sessions returns a middleware that provides cookie-based sessions with HMAC signing.
// The secret is used to sign and verify session cookies.
func Sessions(secret []byte, opts ...SessionOption) MiddlewareFunc {
	cfg := defaultSessionConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := &Session{data: make(map[string]any)}

			// Decode existing session cookie.
			if c, err := r.Cookie(cfg.name); err == nil {
				if data, err := decodeSignedCookie(c.Value, secret); err == nil {
					sess.data = data
				}
			}

			// Extract pending flashes from previous request.
			if f, ok := sess.data[flashKey]; ok {
				if fm, ok := f.(map[string]any); ok {
					sess.flashes = fm
				}
				delete(sess.data, "_flash")
				sess.modified = true
			}

			// Store session in request context.
			ctx := context.WithValue(r.Context(), sessionContextKey, sess)
			r = r.WithContext(ctx)

			// Wrap ResponseWriter for auto-save.
			sw := &sessionResponseWriter{
				ResponseWriter: w,
				session:        sess,
				secret:         secret,
				cfg:            &cfg,
			}

			next.ServeHTTP(sw, r)

			// Save if handler never triggered WriteHeader/Write.
			sw.saveIfNeeded()
		})
	}
}

// sessionResponseWriter wraps http.ResponseWriter to auto-save the session cookie
// before headers are sent.
type sessionResponseWriter struct {
	http.ResponseWriter
	session *Session
	secret  []byte
	cfg     *sessionConfig
	saved   bool
}

func (sw *sessionResponseWriter) WriteHeader(code int) {
	sw.saveIfNeeded()
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *sessionResponseWriter) Write(b []byte) (int, error) {
	sw.saveIfNeeded()
	return sw.ResponseWriter.Write(b)
}

func (sw *sessionResponseWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap supports http.ResponseController and middleware chaining.
func (sw *sessionResponseWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}

func (sw *sessionResponseWriter) saveIfNeeded() {
	if sw.saved || !sw.session.modified {
		return
	}
	sw.saved = true

	encoded, err := encodeSignedCookie(sw.session.data, sw.secret)
	if err != nil {
		log.Printf("dark: failed to encode session: %v", err)
		return
	}

	setCookie(sw.ResponseWriter, sw.cfg.name, encoded,
		CookiePath(sw.cfg.path),
		CookieMaxAge(sw.cfg.maxAge),
		CookieSecure(sw.cfg.secure),
		CookieHTTPOnly(sw.cfg.httpOnly),
		CookieSameSite(sw.cfg.sameSite),
	)
}
