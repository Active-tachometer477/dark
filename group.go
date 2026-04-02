package dark

// Group defines a set of routes that share a common URL prefix, layout, and middleware.
type Group struct {
	app         *App
	prefix      string
	layout      string
	middlewares []MiddlewareFunc
}

// Group creates a route group with a shared URL prefix and layout.
// All routes registered within the group inherit the layout.
// Nested groups compose layouts from outer to inner.
func (app *App) Group(prefix, layout string, fn func(g *Group)) {
	g := &Group{app: app, prefix: prefix, layout: layout}
	fn(g)
}

// Use adds a middleware to the group. It applies only to routes in this group
// and any nested groups.
func (g *Group) Use(mw MiddlewareFunc) {
	g.middlewares = append(g.middlewares, mw)
}

// Group creates a nested group within this group.
func (g *Group) Group(prefix, layout string, fn func(g *Group)) {
	// Compose parent + child layout so nesting preserves outer layouts.
	composedLayout := layout
	if g.layout != "" && layout != "" {
		composedLayout = g.layout + "," + layout
	} else if g.layout != "" {
		composedLayout = g.layout
	}
	// Inherit parent middlewares.
	mws := make([]MiddlewareFunc, len(g.middlewares))
	copy(mws, g.middlewares)
	inner := &Group{app: g.app, prefix: g.prefix + prefix, layout: composedLayout, middlewares: mws}
	fn(inner)
}

func (g *Group) mergeLayouts(route Route) Route {
	if g.layout != "" {
		if route.Layout != "" {
			// Group layout wraps route layout: group is outer, route is inner.
			// Store as comma-separated for the renderer to split.
			route.Layout = g.layout + "," + route.Layout
		} else {
			route.Layout = g.layout
		}
	}
	return route
}

func (g *Group) copyMiddlewares() []MiddlewareFunc {
	if len(g.middlewares) == 0 {
		return nil
	}
	mws := make([]MiddlewareFunc, len(g.middlewares))
	copy(mws, g.middlewares)
	return mws
}

func (g *Group) addRoute(method, pattern string, route Route) {
	merged := g.mergeLayouts(route)
	g.app.routes = append(g.app.routes, registeredRoute{
		method: method, pattern: g.prefix + pattern, page: &merged, middlewares: g.copyMiddlewares(),
	})
}

// Get registers a page route for GET requests within the group.
func (g *Group) Get(pattern string, route Route) { g.addRoute("GET", pattern, route) }

// Post registers a page route for POST requests within the group.
func (g *Group) Post(pattern string, route Route) { g.addRoute("POST", pattern, route) }

// Put registers a page route for PUT requests within the group.
func (g *Group) Put(pattern string, route Route) { g.addRoute("PUT", pattern, route) }

// Delete registers a page route for DELETE requests within the group.
func (g *Group) Delete(pattern string, route Route) { g.addRoute("DELETE", pattern, route) }

// Patch registers a page route for PATCH requests within the group.
func (g *Group) Patch(pattern string, route Route) { g.addRoute("PATCH", pattern, route) }

// API registers an API route within the group.
func (g *Group) API(method, pattern string, route APIRoute) {
	g.app.routes = append(g.app.routes, registeredRoute{
		method: method, pattern: g.prefix + pattern, api: &route, middlewares: g.copyMiddlewares(),
	})
}

// APIGet registers an API route for GET requests within the group.
func (g *Group) APIGet(pattern string, route APIRoute) { g.API("GET", pattern, route) }

// APIPost registers an API route for POST requests within the group.
func (g *Group) APIPost(pattern string, route APIRoute) { g.API("POST", pattern, route) }

// APIPut registers an API route for PUT requests within the group.
func (g *Group) APIPut(pattern string, route APIRoute) { g.API("PUT", pattern, route) }

// APIDelete registers an API route for DELETE requests within the group.
func (g *Group) APIDelete(pattern string, route APIRoute) { g.API("DELETE", pattern, route) }

// APIPatch registers an API route for PATCH requests within the group.
func (g *Group) APIPatch(pattern string, route APIRoute) { g.API("PATCH", pattern, route) }
