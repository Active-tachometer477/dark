package dark

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// FieldError represents a validation error for a specific form field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// HeadData holds metadata for the HTML <head> section.
type HeadData struct {
	Title string    `json:"title,omitempty"`
	Meta  []MetaTag `json:"meta,omitempty"`
}

// MetaTag represents a <meta> element.
type MetaTag struct {
	Name     string `json:"name,omitempty"`
	Property string `json:"property,omitempty"`
	Content  string `json:"content"`
}

// Context provides access to the request, response, and route parameters.
type Context interface {
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	Param(name string) string
	Query(name string) string
	FormData() url.Values
	Redirect(url string) error
	RenderError(err error) error
	SetHeader(key, value string)
	JSON(status int, data any) error
	BindJSON(v any) error
	AddFieldError(field, message string)
	HasErrors() bool
	FieldErrors() []FieldError
	SetTitle(title string)
	AddMeta(name, content string)
	AddOpenGraph(property, content string)
	SetCookie(name, value string, opts ...CookieOption)
	GetCookie(name string) (string, error)
	DeleteCookie(name string)
	Session() *Session
}

type darkContext struct {
	w           http.ResponseWriter
	r           *http.Request
	renderError error
	written     bool
	fieldErrors []FieldError
	head        HeadData
	query       url.Values // cached parsed query string
}

func (c *darkContext) Request() *http.Request {
	return c.r
}

func (c *darkContext) ResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *darkContext) Param(name string) string {
	return c.r.PathValue(name)
}

func (c *darkContext) Query(name string) string {
	if c.query == nil {
		c.query = c.r.URL.Query()
	}
	return c.query.Get(name)
}

func (c *darkContext) FormData() url.Values {
	if c.r.Form == nil {
		c.r.ParseForm()
	}
	return c.r.Form
}

func (c *darkContext) Redirect(url string) error {
	c.written = true
	hxAwareRedirect(c.w, c.r, url)
	return nil
}

// hxAwareRedirect sends an HTTP redirect, using HX-Redirect for htmx requests.
func hxAwareRedirect(w http.ResponseWriter, r *http.Request, url string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (c *darkContext) RenderError(err error) error {
	c.renderError = err
	return nil
}

func (c *darkContext) SetHeader(key, value string) {
	c.w.Header().Set(key, value)
}

func (c *darkContext) JSON(status int, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.w.WriteHeader(status)
	c.w.Write(b)
	c.written = true
	return nil
}

func (c *darkContext) BindJSON(v any) error {
	return json.NewDecoder(c.r.Body).Decode(v)
}

func (c *darkContext) SetTitle(title string) {
	c.head.Title = title
}

func (c *darkContext) AddMeta(name, content string) {
	c.head.Meta = append(c.head.Meta, MetaTag{Name: name, Content: content})
}

func (c *darkContext) AddOpenGraph(property, content string) {
	c.head.Meta = append(c.head.Meta, MetaTag{Property: property, Content: content})
}

func (c *darkContext) AddFieldError(field, message string) {
	c.fieldErrors = append(c.fieldErrors, FieldError{Field: field, Message: message})
}

func (c *darkContext) HasErrors() bool {
	return len(c.fieldErrors) > 0
}

func (c *darkContext) FieldErrors() []FieldError {
	out := make([]FieldError, len(c.fieldErrors))
	copy(out, c.fieldErrors)
	return out
}

func (c *darkContext) SetCookie(name, value string, opts ...CookieOption) {
	setCookie(c.w, name, value, opts...)
}

func (c *darkContext) GetCookie(name string) (string, error) {
	return getCookie(c.r, name)
}

func (c *darkContext) DeleteCookie(name string) {
	deleteCookie(c.w, name)
}

func (c *darkContext) Session() *Session {
	if sess, ok := c.r.Context().Value(sessionContextKey).(*Session); ok {
		return sess
	}
	panic("dark: Session() called without Sessions middleware; add app.Use(dark.Sessions(secret))")
}

func (c *darkContext) isHXRequest() bool {
	return c.r.Header.Get("HX-Request") == "true"
}
