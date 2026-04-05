# Dark

A Go SSR web framework powered by [Preact](https://preactjs.com/) or [React](https://react.dev/), [htmx](https://htmx.org/), and Islands architecture.

Dark renders TSX components on the server using [ramune](https://github.com/i2y/ramune) (a JS/TS runtime for Go), fetches data with Go Loader/Action functions, and delivers interactive pages through htmx's HTML-over-the-wire approach with minimal client-side JavaScript.

## Requirements

Dark uses ramune for SSR, which supports two JS engine backends:

| | JSC (default) | QuickJS (`-tags quickjs`) |
|---|---|---|
| **Engine** | Apple JavaScriptCore via [purego](https://github.com/ebitengine/purego) | [modernc.org/quickjs](https://pkg.go.dev/modernc.org/quickjs) (pure Go) |
| **JIT** | Yes | No |
| **Platforms** | macOS, Linux | macOS, Linux, Windows |
| **System deps** | macOS: none. Linux: `apt install libjavascriptcoregtk-4.1-dev` | None |
| **Best for** | Production performance | Portability, zero-dependency deploys |

Both are pure Go builds -- no C compiler or Cgo required.

### Default (JavaScriptCore)

```bash
# macOS — no extra dependencies
go build .

# Linux
sudo apt install libjavascriptcoregtk-4.1-dev
go build .
```

### QuickJS backend

```bash
go build -tags quickjs .
```

No shared libraries needed. Works on all platforms including Windows. Trade-off: no JIT, so JS execution is slower (SSR render time increases). For most apps where the bottleneck is I/O (database, network), this is negligible.

## Built on net/http

Dark follows standard `net/http` conventions. There are no external router dependencies.

- Internal routing uses `http.NewServeMux` with Go 1.22+ enhanced patterns (`GET /users/{id}`)
- `app.Handler()` returns `(http.Handler, error)` — plug it into any Go HTTP stack
- Middleware is the standard `func(http.Handler) http.Handler` signature
- Dark does not own the server — you start it yourself with `http.ListenAndServe` or `http.Server`

```go
// Simple — MustHandler() panics on error (convenient for main)
http.ListenAndServe(":3000", app.MustHandler())

// With error handling
handler, err := app.Handler()
if err != nil {
    log.Fatal(err)
}
http.ListenAndServe(":3000", handler)

// With http.Server for full control
srv := &http.Server{
    Addr:         ":8080",
    Handler:      app.MustHandler(),
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
}
srv.ListenAndServe()
```

Any existing `net/http` middleware works with `app.Use()` out of the box.

## Features

- **Server-side rendering** — TSX templates rendered via Preact or React `renderToString` in a sandboxed JS runtime
- **Loader/Action pattern** — Go functions for data fetching and mutations, props passed as JSON
- **htmx integration** — HX-Request aware responses (full page vs HTML fragment)
- **Islands architecture** — selective client-side hydration with lazy loading (`load`, `idle`, `visible`)
- **Streaming SSR** — shell-first rendering for faster TTFB
- **Nested layouts** — composable layouts via route groups
- **Form validation** — field-level errors with form data preservation
- **Sessions** — HMAC-signed cookie sessions with flash messages
- **Authentication** — `RequireAuth` middleware with htmx-aware redirects
- **Head management** — per-page `<title>`, `<meta>`, and OpenGraph tags
- **API routes** — JSON endpoints alongside page routes
- **Dev mode** — hot reload, error overlay with source maps, TypeScript type generation
- **SSR caching** — LRU in-memory cache with ETag / 304 Not Modified
- **CSRF protection** — session-based tokens with automatic htmx/TSX integration
- **Concurrent loaders** — parallel data fetching with result merging
- **Static site generation** — pre-render routes to static HTML at build time
- **Scaffold CLI** — `dark new` / `dark generate` for project and component scaffolding

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/i2y/dark"
)

func main() {
    app, err := dark.New(
        dark.WithLayout("layouts/default.tsx"),
        dark.WithTemplateDir("views"),
        dark.WithDevMode(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer app.Close()

    app.Use(dark.Logger())
    app.Use(dark.Recover())

    app.Get("/", dark.Route{
        Component: "pages/index.tsx",
        Loader: func(ctx dark.Context) (any, error) {
            return map[string]any{"message": "Hello, Dark!"}, nil
        },
    })

    log.Fatal(http.ListenAndServe(":3000", app.MustHandler()))
}
```

Or use the scaffold CLI to generate a new project:

```bash
go install github.com/i2y/dark/cmd/dark@latest
dark new myapp
cd myapp && go mod tidy && make dev
```

The layout wraps every page. Each page component's output is passed as `children`. On htmx requests (`HX-Request` header), the layout is skipped and only the page fragment is returned.

```tsx
// views/layouts/default.tsx
import { h } from "preact"; // required — JSX transpiles to h() calls

export default function Layout({ children }) {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <title>My App</title>
        <script src="https://unpkg.com/htmx.org@2.0.4"></script>
      </head>
      <body>{children}</body>
    </html>
  );
}
```

```tsx
// views/pages/index.tsx
import { h } from "preact"; // required — JSX transpiles to h() calls

export default function IndexPage({ message }) {
  return <h1>{message}</h1>;
}
```

```
go run main.go
# => Listening on http://localhost:3000
```

## Routing

Routes use Go 1.22+ `ServeMux` patterns with `{param}` wildcards.

```go
app.Get("/", dark.Route{...})
app.Get("/users/{id}", dark.Route{...})
app.Post("/users/{id}/orders", dark.Route{...})
app.Put("/posts/{id}", dark.Route{...})
app.Delete("/posts/{id}", dark.Route{...})
app.Patch("/settings", dark.Route{...})
```

### Route struct

```go
dark.Route{
    Component: "pages/show.tsx",   // TSX file (relative to template dir)
    Loader:    loaderFunc,          // data fetching (single)
    Loaders:   []dark.LoaderFunc{...}, // concurrent data fetching (merged)
    Action:    actionFunc,          // mutations (POST/PUT/DELETE)
    Layout:    "layouts/extra.tsx", // per-route layout (nests inside global layout)
    Streaming: &boolVal,            // per-route streaming SSR override
    Props:     MyProps{},           // zero value for TypeScript type generation
}
```

### API routes

JSON endpoints that bypass the TSX rendering pipeline:

```go
app.APIGet("/api/status", dark.APIRoute{
    Handler: func(ctx dark.Context) error {
        return ctx.JSON(200, map[string]any{"status": "ok"})
    },
})

app.APIPost("/api/items", dark.APIRoute{
    Handler: func(ctx dark.Context) error {
        var input CreateItemRequest
        if err := ctx.BindJSON(&input); err != nil {
            return dark.NewAPIError(400, "invalid JSON")
        }
        // ...
        return ctx.JSON(201, item)
    },
})
```

## Route Groups

Groups share a URL prefix, layout, and middleware:

```go
app.Group("/admin", "layouts/admin.tsx", func(g *dark.Group) {
    g.Use(dark.RequireAuth())

    g.Get("/dashboard", dark.Route{
        Component: "pages/admin/dashboard.tsx",
        Loader:    dashboardLoader,
    })

    // Nested groups compose layouts
    g.Group("/settings", "layouts/settings.tsx", func(sg *dark.Group) {
        sg.Get("/profile", dark.Route{...})
    })
})
```

## Context

`dark.Context` wraps the request and response:

```go
ctx.Request() *http.Request
ctx.ResponseWriter() http.ResponseWriter
ctx.Param("id") string              // path parameter ({id})
ctx.Query("page") string            // query string
ctx.FormData() url.Values           // parsed form data
ctx.Redirect("/path") error         // redirect (htmx-aware)
ctx.SetHeader("X-Custom", "value")

// JSON
ctx.JSON(200, data) error
ctx.BindJSON(&input) error

// Validation
ctx.AddFieldError("email", "required")
ctx.HasErrors() bool
ctx.FieldErrors() []FieldError

// Head
ctx.SetTitle("Page Title")
ctx.AddMeta("description", "...")
ctx.AddOpenGraph("og:image", "...")

// Cookies
ctx.SetCookie("theme", "dark", dark.CookieMaxAge(86400))
ctx.GetCookie("theme") (string, error)
ctx.DeleteCookie("theme")

// Session (requires Sessions middleware)
ctx.Session() *Session

// Request-scoped values (set by middleware, read by loaders)
ctx.Set("key", value)
ctx.Get("key") any
```

## Sessions

HMAC-SHA256 signed cookie sessions:

```go
app.Use(dark.Sessions([]byte("secret-key-at-least-32-bytes"),
    dark.SessionName("app_session"),
    dark.SessionMaxAge(86400),
    dark.SessionSecure(true),
))
```

```go
// In a Loader/Action:
sess := ctx.Session()
sess.Set("user", username)
sess.Get("user")           // returns any
sess.Delete("user")
sess.Clear()

// Flash messages (available for one request)
sess.Flash("notice", "Saved!")
flashes := sess.Flashes()  // map[string]any
```

## Authentication

```go
// Basic usage — checks session key "user", redirects to "/login"
g.Use(dark.RequireAuth())

// Custom options
g.Use(dark.RequireAuth(
    dark.AuthSessionKey("account"),
    dark.AuthLoginURL("/auth/signin"),
    dark.AuthCheck(func(s *dark.Session) bool {
        return s.Get("role") == "admin"
    }),
))
```

## CSRF Protection

Session-based CSRF tokens with automatic htmx integration:

```go
app.Use(dark.Sessions(secret))
app.Use(dark.CSRF())
```

The middleware automatically:
- Generates a per-session token
- Injects `<meta name="csrf-token">` into `<head>`
- Injects an htmx config script that attaches `X-CSRF-Token` to all htmx requests
- Adds `_csrfToken` to Loader props (use in hidden form fields)
- Validates `X-CSRF-Token` header or `_csrf` form field on POST/PUT/DELETE/PATCH

```tsx
export default function Form({ _csrfToken }) {
  return (
    <form method="POST" action="/submit">
      <input type="hidden" name="_csrf" value={_csrfToken} />
      <button type="submit">Submit</button>
    </form>
  );
}
```

htmx forms require no extra setup — the token header is attached automatically.

## Concurrent Loaders

Fetch data from multiple sources in parallel:

```go
app.Get("/dashboard", dark.Route{
    Component: "pages/dashboard.tsx",
    Loaders: []dark.LoaderFunc{
        func(ctx dark.Context) (any, error) {
            return map[string]any{"user": fetchUser(ctx.Param("id"))}, nil
        },
        func(ctx dark.Context) (any, error) {
            return map[string]any{"activity": fetchActivity(ctx.Param("id"))}, nil
        },
        func(ctx dark.Context) (any, error) {
            return map[string]any{"notifications": fetchNotifications()}, nil
        },
    },
})
```

Results are merged into a single props map. If any loader returns an error, the request fails immediately.

## Middleware

Standard `func(http.Handler) http.Handler`:

```go
app.Use(dark.Logger())                 // request logging
app.Use(dark.Recover())                // panic recovery → 500
app.Use(app.RecoverWithErrorPage())    // panic recovery → custom error page
app.Use(dark.Sessions(secret))         // session management
```

Any existing net/http middleware works:

```go
app.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Frame-Options", "DENY")
        next.ServeHTTP(w, r)
    })
})
```

## Islands Architecture

Register interactive components for client-side hydration:

```go
app.Island("counter", "islands/counter.tsx")
```

```tsx
// views/islands/counter.tsx
import { h } from "preact";
import { useState } from "preact/hooks";

function Counter({ initial = 0 }) {
  const [count, setCount] = useState(initial);
  return <button onClick={() => setCount(count + 1)}>Count: {count}</button>;
}

// Wrap with dark.island() — loaded immediately by default
export default dark.island("counter", Counter);

// Lazy loading strategies:
// dark.island("counter", Counter, { load: "idle" })    — requestIdleCallback
// dark.island("counter", Counter, { load: "visible" }) — IntersectionObserver
```

Use in any page TSX:

```tsx
import Counter from "../islands/counter.tsx";

export default function Page() {
  return (
    <div>
      <h1>My Page</h1>
      <Counter initial={5} />
    </div>
  );
}
```

## Static Files

```go
app.Static("/static/", "public")
```

## Options

```go
dark.New(
    dark.WithPoolSize(4),                    // ramune RuntimePool workers (default: runtime.NumCPU())
    dark.WithTemplateDir("views"),           // TSX file directory (default: "views")
    dark.WithLayout("layouts/default.tsx"),   // global layout
    dark.WithDependencies("lodash"),          // npm packages (preact is always included)
    dark.WithDevMode(true),                  // hot reload + error overlay
    dark.WithStreaming(true),                // streaming SSR globally
    dark.WithSSRCache(1000),                 // LRU SSR output cache (enables ETag)
    dark.WithLogger(slog.Default()),         // structured logger for framework internals
    dark.WithErrorComponent("errors/500.tsx"),
    dark.WithNotFoundComponent("errors/404.tsx"),
)
```

## Project Structure

```
myapp/
├── main.go
├── views/
│   ├── layouts/
│   │   └── default.tsx
│   ├── pages/
│   │   ├── index.tsx
│   │   └── users/
│   │       └── show.tsx
│   ├── islands/
│   │   └── counter.tsx
│   └── errors/
│       ├── 404.tsx
│       └── 500.tsx
└── public/
    └── style.css
```

## Static Site Generation

Pre-render routes to static HTML at build time:

```go
err := app.GenerateStaticSite("dist", []dark.StaticRoute{
    {
        Path:      "/",
        Component: "pages/index.tsx",
        Loader:    indexLoader,
    },
    {
        Path:      "/about",
        Component: "pages/about.tsx",
    },
    {
        // Parameterized routes: StaticPaths returns all concrete paths
        Component: "pages/post.tsx",
        StaticPaths: func() []string {
            return []string{"/posts/1", "/posts/2", "/posts/3"}
        },
        Loader: postLoader,
    },
})
```

Output is written to `dist/` as `index.html` files with all CSS and island assets copied.

## React Support

Dark defaults to Preact but also supports React. Pass `WithUILibrary(dark.React)` to switch:

```go
app, err := dark.New(
    dark.WithUILibrary(dark.React),
    dark.WithLayout("layouts/default.tsx"),
    dark.WithTemplateDir("views"),
)
```

With React, components use standard React imports:

```tsx
// views/pages/index.tsx
import React from 'react';

export default function IndexPage({ message }) {
  return <h1>{message}</h1>;
}
```

Islands use React hooks directly:

```tsx
// views/islands/counter.tsx
import React, { useState } from 'react';

export default function Counter({ initial }) {
  const [count, setCount] = useState(initial || 0);
  return <button onClick={() => setCount(c => c + 1)}>{count}</button>;
}
```

MCP Apps also support React via `WithMCPUILibrary(dark.React)`.

## Scaffold CLI

Generate projects and components:

```bash
go install github.com/i2y/dark/cmd/dark@latest

# New project (defaults to Preact)
dark new myapp
cd myapp && go mod tidy && make dev

# New project with React
dark new myapp --ui react

# Generate components
dark generate route users    # → views/pages/users.tsx
dark generate island counter # → views/islands/counter.tsx
```

## MCP Apps (experimental)

> **Note:** This feature has not been fully tested yet. The API may change.

Dark supports [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview) — interactive HTML UIs returned by MCP tools, rendered inside host sandboxed iframes. Built on the official Go SDK [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

```go
mcpApp, err := dark.NewMCPApp("my-server", "1.0.0",
    dark.WithMCPTemplateDir("views"),
)
defer mcpApp.Close()

// UI tool: returns an interactive TSX component
if err := dark.AddUITool(mcpApp, "dashboard", dark.UIToolDef{
    Description: "Show analytics dashboard",
    Component:   "mcp/dashboard.tsx",
}, func(ctx context.Context, args DashboardArgs) (map[string]any, error) {
    return map[string]any{"data": fetchData(args.Period)}, nil
}); err != nil {
    log.Fatal(err)
}

// Text tool: standard MCP tool returning plain text
dark.AddTextTool(mcpApp, "stats", "Get statistics",
    func(ctx context.Context, args StatsArgs) (string, error) {
        return "Stats: ...", nil
    })

mcpApp.RunStdio(ctx)           // stdio transport
// or
mcpApp.StreamableHTTPHandler() // HTTP transport
```

The server declares the `io.modelcontextprotocol/ui` extension and registers each UI tool as a resource (`ui://{server}/{tool}.html`). Rendering pipeline:

1. At registration: esbuild bundles the component with UI library inlined → static app shell HTML
2. On tool call: Go handler returns props → sent as JSON text in the tool result
3. Host reads the resource → renders the HTML in a sandboxed iframe
4. MCP App Bridge (postMessage JSON-RPC, protocol `2026-01-26`) delivers tool results to the iframe
5. Component renders client-side with the received props

Example: [`examples/mcp-app/`](examples/mcp-app/)

## Examples

- **[hello](_examples/hello/)** — feature-rich demo: routing, layouts, sessions, islands, streaming SSR, form validation
- **[showcase](_examples/showcase/)** — CSRF, concurrent loaders, SSR cache + ETag, SSG, Context.Set/Get
- **[database](_examples/database/)** — SQLite CRUD with sessions and authentication
- **[deploy](_examples/deploy/)** — production setup with Dockerfile and Fly.io config
- **[mcp-app](examples/mcp-app/)** — MCP Apps: interactive UI tools via postMessage

## Deploy

See [_examples/deploy](_examples/deploy/) for a production-ready setup with Docker multi-stage build and Fly.io configuration.

## License

MIT
