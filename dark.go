package dark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// App is the main dark application.
type App struct {
	config      *config
	renderer    *renderer
	middlewares []MiddlewareFunc
	islands     []islandEntry
	reloader    *devReloader
	routes      []registeredRoute
	staticDirs  []staticDir
}

type registeredRoute struct {
	method      string
	pattern     string
	page        *Route
	api         *APIRoute
	middlewares []MiddlewareFunc
}

type staticDir struct {
	prefix string
	dir    string
}

// New creates a new dark application.
func New(opts ...Option) (*App, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	rend, err := newRenderer(cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		config:   cfg,
		renderer: rend,
	}, nil
}

// Close releases all resources held by the application.
func (app *App) Close() error {
	if app.reloader != nil {
		app.reloader.close()
	}
	if app.renderer != nil {
		return app.renderer.close()
	}
	return nil
}

// Get registers a page route for GET requests.
func (app *App) Get(pattern string, route Route) {
	app.routes = append(app.routes, registeredRoute{method: "GET", pattern: pattern, page: &route})
}

// Post registers a page route for POST requests.
func (app *App) Post(pattern string, route Route) {
	app.routes = append(app.routes, registeredRoute{method: "POST", pattern: pattern, page: &route})
}

// Put registers a page route for PUT requests.
func (app *App) Put(pattern string, route Route) {
	app.routes = append(app.routes, registeredRoute{method: "PUT", pattern: pattern, page: &route})
}

// Delete registers a page route for DELETE requests.
func (app *App) Delete(pattern string, route Route) {
	app.routes = append(app.routes, registeredRoute{method: "DELETE", pattern: pattern, page: &route})
}

// Patch registers a page route for PATCH requests.
func (app *App) Patch(pattern string, route Route) {
	app.routes = append(app.routes, registeredRoute{method: "PATCH", pattern: pattern, page: &route})
}

// API registers an API route for the given HTTP method and pattern.
func (app *App) API(method, pattern string, route APIRoute) {
	app.routes = append(app.routes, registeredRoute{method: method, pattern: pattern, api: &route})
}

// APIGet registers an API route for GET requests.
func (app *App) APIGet(pattern string, route APIRoute) { app.API("GET", pattern, route) }

// APIPost registers an API route for POST requests.
func (app *App) APIPost(pattern string, route APIRoute) { app.API("POST", pattern, route) }

// APIPut registers an API route for PUT requests.
func (app *App) APIPut(pattern string, route APIRoute) { app.API("PUT", pattern, route) }

// APIDelete registers an API route for DELETE requests.
func (app *App) APIDelete(pattern string, route APIRoute) { app.API("DELETE", pattern, route) }

// APIPatch registers an API route for PATCH requests.
func (app *App) APIPatch(pattern string, route APIRoute) { app.API("PATCH", pattern, route) }

// Use adds a middleware to the application.
func (app *App) Use(mw MiddlewareFunc) {
	app.middlewares = append(app.middlewares, mw)
}

// Static registers a static file server for the given URL prefix and directory.
func (app *App) Static(urlPrefix, dir string) {
	// Ensure prefix ends with / for ServeMux subtree matching.
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix += "/"
	}
	app.staticDirs = append(app.staticDirs, staticDir{prefix: urlPrefix, dir: dir})
}

// Island registers a component for client-side hydration.
func (app *App) Island(name, tsxPath string) {
	app.islands = append(app.islands, islandEntry{name: name, tsxPath: tsxPath})
}

// Handler returns the application as an http.Handler with middleware applied.
func (app *App) Handler() http.Handler {
	return app.buildHandler()
}

func (app *App) buildHandler() http.Handler {
	// Build islands if any were registered.
	if len(app.islands) > 0 {
		if err := app.renderer.buildIslands(app.islands, app.config); err != nil {
			panic(fmt.Sprintf("dark: failed to build islands: %v", err))
		}
	}

	// Prepare route-specific layouts.
	if err := app.renderer.prepareRouteLayouts(app.routes); err != nil {
		panic(fmt.Sprintf("dark: failed to prepare route layouts: %v", err))
	}

	// Generate TypeScript types from Props fields (dev mode only).
	if app.config.devMode {
		if err := app.GenerateTypes(); err != nil {
			log.Printf("dark: type generation: %v", err)
		}
	}

	// Start dev reloader if in dev mode.
	if app.config.devMode && app.reloader == nil {
		reloader, err := newDevReloader(app.renderer, app.config, app.islands)
		if err != nil {
			log.Printf("dark: failed to start dev reloader: %v", err)
		} else {
			app.reloader = reloader
		}
	}

	mux := http.NewServeMux()

	// Special endpoints.
	if app.reloader != nil {
		mux.HandleFunc("GET /_dark/reload", app.reloader.ServeSSE)
	}
	if app.renderer.hasIslands {
		mux.HandleFunc("GET /_dark/islands/{file}", app.serveIslandsJS)
	}
	mux.HandleFunc("GET /_dark/css/", app.serveCSS)

	// Static directories.
	for _, sd := range app.staticDirs {
		mux.Handle(sd.prefix, http.StripPrefix(sd.prefix, http.FileServer(http.Dir(sd.dir))))
	}

	// Register user routes.
	for _, r := range app.routes {
		pattern := muxPattern(r.method, r.pattern)
		var h http.Handler
		if r.api != nil {
			h = http.HandlerFunc(app.apiHandler(r.api))
		} else {
			if app.shouldStream(r.page) && app.renderer.hasLayout {
				h = http.HandlerFunc(app.streamingPageHandler(r.page))
			} else {
				h = http.HandlerFunc(app.pageHandler(r.page))
			}
		}
		// Apply per-route middleware (from Group.Use).
		for i := len(r.middlewares) - 1; i >= 0; i-- {
			h = r.middlewares[i](h)
		}
		mux.Handle(pattern, h)
	}

	// Catch-all 404.
	mux.HandleFunc("/", app.notFoundHandler)

	// Apply middleware.
	var handler http.Handler = mux
	for i := len(app.middlewares) - 1; i >= 0; i-- {
		handler = app.middlewares[i](handler)
	}
	return handler
}

// muxPattern converts a method and dark pattern to a ServeMux pattern.
// "/" is converted to "GET /{$}" for exact root matching.
func muxPattern(method, pattern string) string {
	if pattern == "/" {
		return method + " /{$}"
	}
	return method + " " + pattern
}

// runActionAndLoader executes Action + Loader + validation/head merge.
// Returns (props, error, done). If done is true, the caller should return immediately.
func (app *App) runActionAndLoader(ctx *darkContext, route *Route) (any, error, bool) {
	if route.Action != nil {
		if err := route.Action(ctx); err != nil {
			return nil, err, false
		}
		if ctx.written {
			return nil, nil, true
		}
		if ctx.renderError != nil {
			return nil, ctx.renderError, false
		}
	}

	var props any
	if route.Loader != nil {
		var err error
		props, err = route.Loader(ctx)
		if err != nil {
			return nil, err, false
		}
	}
	if props == nil {
		props = map[string]any{}
	}

	if ctx.HasErrors() {
		props = mergeValidation(props, ctx.fieldErrors, ctx.FormData())
	}
	if ctx.head.Title != "" || len(ctx.head.Meta) > 0 {
		props = mergeHead(props, ctx.head)
	}

	return props, nil, false
}

func (app *App) shouldStream(route *Route) bool {
	if route.Streaming != nil {
		return *route.Streaming
	}
	return app.config.streaming
}

func (app *App) pageHandler(route *Route) http.HandlerFunc {
	layouts := parseLayouts(route.Layout) // precompute once
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &darkContext{w: w, r: r}

		props, err, done := app.runActionAndLoader(ctx, route)
		if done {
			return
		}
		if err != nil {
			app.renderError(w, r, err)
			return
		}

		if route.Component == "" {
			return
		}

		app.renderNonStreaming(w, r, ctx, route.Component, layouts, props)
	}
}

func (app *App) streamingPageHandler(route *Route) http.HandlerFunc {
	layouts := parseLayouts(route.Layout) // precompute once
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &darkContext{w: w, r: r}

		props, err, done := app.runActionAndLoader(ctx, route)
		if done {
			return
		}
		if err != nil {
			app.renderError(w, r, err)
			return
		}

		if route.Component == "" {
			return
		}

		// htmx partials and non-flusher responses fall back to non-streaming.
		flusher, canFlush := w.(http.Flusher)
		if ctx.isHXRequest() || !canFlush || !app.renderer.hasLayout {
			app.renderNonStreaming(w, r, ctx, route.Component, layouts, props)
			return
		}

		// Phase 1: Render shell (layout with placeholder).
		before, after, shellCSS, err := app.renderer.renderShell(layouts, props)
		if err != nil {
			app.renderError(w, r, err)
			return
		}

		if shellCSS != "" {
			hash := app.renderer.storePageCSS(shellCSS)
			link := fmt.Sprintf(`<link rel="stylesheet" href="/_dark/css/%s.css">`, hash)
			before = insertBeforeTag(before, "</head>", link)
		}

		// Post-process <dark-head> in the shell (from layout).
		var headContent string
		before, headContent = extractDarkHead(before)
		if headContent != "" {
			before = injectIntoHead(before, headContent)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, before)
		flusher.Flush()

		// Phase 2: Render component (no layout).
		componentHTML, componentCSS, err := app.renderer.render(route.Component, nil, props, true)
		if err != nil {
			// Head already sent — inject error as inline script.
			fmt.Fprintf(w, `<script>console.error(%q)</script>`, err.Error())
			fmt.Fprint(w, after)
			flusher.Flush()
			return
		}

		// Post-process <dark-head> in component output.
		var compHeadContent string
		componentHTML, compHeadContent = extractDarkHead(componentHTML)
		if compHeadContent != "" {
			// Can't inject into <head> since it's already sent; inject as inline tags.
			fmt.Fprint(w, "<!--dark-head-late-->"+compHeadContent+"<!--/dark-head-late-->")
		}

		// Inject component CSS as inline <style> since <head> is already sent.
		if componentCSS != "" {
			fmt.Fprintf(w, "<style>%s</style>", componentCSS)
		}

		fmt.Fprint(w, componentHTML)

		// Inject islands script if needed.
		if app.renderer.hasIslands && strings.Contains(componentHTML, "<dark-island") {
			io.WriteString(w, app.renderer.getIslandsScriptTag(componentHTML))
		}

		// Dev reload script.
		if app.config.devMode {
			fmt.Fprint(w, devReloadScript)
		}

		fmt.Fprint(w, after)
		flusher.Flush()
	}
}

// renderNonStreaming is the standard (non-streaming) render path.
func (app *App) renderNonStreaming(w http.ResponseWriter, r *http.Request, ctx *darkContext, component string, layouts []string, props any) {
	skipLayout := ctx.isHXRequest()
	output, css, err := app.renderer.render(component, layouts, props, skipLayout)
	if err != nil {
		app.renderError(w, r, err)
		return
	}

	if skipLayout {
		output = stripDarkHead(output)
	} else {
		var headContent string
		output, headContent = extractDarkHead(output)
		if headContent != "" {
			output = injectIntoHead(output, headContent)
		}
	}

	output = app.injectCSS(output, css, skipLayout)

	if app.config.devMode && !skipLayout {
		output = injectDevReloadScript(output)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, output)
}

func (app *App) apiHandler(route *APIRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &darkContext{w: w, r: r}
		if err := route.Handler(ctx); err != nil {
			app.writeAPIError(w, err)
			return
		}
		if !ctx.written {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func (app *App) writeAPIError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "Internal Server Error"

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		status = apiErr.Status
		message = apiErr.Message
	} else if app.config.devMode {
		message = err.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error":  message,
		"status": status,
	})
}

func (app *App) serveIslandsJS(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	content, ok := app.renderer.getClientChunk(file)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	if app.config.devMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	fmt.Fprint(w, content)
}

func (app *App) serveCSS(w http.ResponseWriter, r *http.Request) {
	// URL: /_dark/css/<hash>.css
	name := strings.TrimPrefix(r.URL.Path, "/_dark/css/")
	hash := strings.TrimSuffix(name, ".css")
	css := app.renderer.getPageCSS(hash)
	if css == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	if app.config.devMode {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	fmt.Fprint(w, css)
}

// injectCSS adds CSS to the HTML response.
// Full pages get a <link> tag in <head>; htmx partials get inline <style>.
func (app *App) injectCSS(htmlStr, css string, skipLayout bool) string {
	if css == "" {
		return htmlStr
	}
	if skipLayout {
		return "<style>" + css + "</style>" + htmlStr
	}
	hash := app.renderer.storePageCSS(css)
	link := fmt.Sprintf(`<link rel="stylesheet" href="/_dark/css/%s.css">`, hash)
	return insertBeforeTag(htmlStr, "</head>", link)
}

func (app *App) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	app.renderNotFound(w, r)
}

func (app *App) renderNotFound(w http.ResponseWriter, r *http.Request) {
	if app.renderer.hasNotFoundComponent {
		props := map[string]any{
			"path":       r.URL.Path,
			"statusCode": 404,
		}
		output, css, renderErr := app.renderer.renderNotFoundPage(props)
		if renderErr == nil {
			output = app.injectCSS(output, css, false)
			if app.config.devMode {
				output = injectDevReloadScript(output)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, output)
			return
		}
		log.Printf("dark: not-found component render failed: %v", renderErr)
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

func (app *App) renderError(w http.ResponseWriter, r *http.Request, err error) {
	// In dev mode, always use the rich dev overlay for maximum debugging info.
	if app.config.devMode {
		app.renderDevOverlay(w, err)
		return
	}

	props := map[string]any{
		"statusCode": 500,
		"message":    "Internal Server Error",
	}

	if app.renderer.hasErrorComponent {
		output, css, renderErr := app.renderer.renderErrorPage(props)
		if renderErr == nil {
			output = app.injectCSS(output, css, false)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, output)
			return
		}
		log.Printf("dark: error component render failed: %v", renderErr)
	}

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// propsAsMap ensures props is a map[string]any, using JSON round-trip as fallback.
func propsAsMap(props any) map[string]any {
	if m, ok := props.(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	if b, err := json.Marshal(props); err == nil {
		json.Unmarshal(b, &m)
	}
	return m
}

func mergeHead(props any, head HeadData) any {
	m := propsAsMap(props)
	m["_head"] = head
	return m
}

func mergeValidation(props any, errors []FieldError, formData url.Values) any {
	m := propsAsMap(props)

	errList := make([]map[string]any, len(errors))
	for i, e := range errors {
		errList[i] = map[string]any{"field": e.Field, "message": e.Message}
	}
	m["_errors"] = errList

	fd := make(map[string]any, len(formData))
	for k, v := range formData {
		if len(v) == 1 {
			fd[k] = v[0]
		} else {
			fd[k] = v
		}
	}
	m["_formData"] = fd

	return m
}

// parseLayouts splits a comma-separated layout string into a slice.
func parseLayouts(layout string) []string {
	if layout == "" {
		return nil
	}
	parts := strings.Split(layout, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
