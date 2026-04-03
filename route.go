package dark

// LoaderFunc fetches data for a route. The returned value is passed as props to the TSX component.
type LoaderFunc func(ctx Context) (any, error)

// ActionFunc handles mutations (e.g., form submissions).
type ActionFunc func(ctx Context) error

// Route defines a handler for a URL pattern with SSR rendering.
type Route struct {
	Component string       // TSX file path relative to the template directory
	Loader    LoaderFunc   // data loader (single)
	Loaders   []LoaderFunc // concurrent data loaders; results are merged into one props map
	Action    ActionFunc   // mutation handler
	Layout    string       // layout TSX file path relative to the template directory (nests inside global layout)
	Streaming *bool        // nil = use global default, true/false = per-route override
	Props     any          // optional: Go struct zero value for TypeScript type generation
}

// HandlerFunc handles an API request.
type HandlerFunc func(ctx Context) error

// APIRoute defines a handler for a JSON API endpoint.
type APIRoute struct {
	Handler HandlerFunc
}
