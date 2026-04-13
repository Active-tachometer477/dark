# dark/desktop

Native desktop window for dark applications, powered by [glaze](https://github.com/crgimenes/glaze) WebView.

Wraps your `http.Handler` in a native window with Go↔JS function bindings and a bidirectional event system — inspired by [Wails](https://wails.io/) and [Tauri](https://tauri.app/). All dark features (SSR, Islands, htmx, sessions) work unmodified in desktop mode.

Import this package only when building desktop apps — it pulls in the glaze WebView native library dependency.

## Quick Start

```go
package main

import (
    "runtime"

    "github.com/i2y/dark"
    "github.com/i2y/dark/desktop"
)

func init() { runtime.LockOSThread() }

func main() {
    app, _ := dark.New(
        dark.WithLayout("layouts/default.tsx"),
        dark.WithTemplateDir("views"),
    )
    defer app.Close()

    app.Get("/", dark.Route{
        Component: "pages/index.tsx",
        Loader:    indexLoader,
    })

    // Simple: one-liner convenience function
    desktop.Run(app.MustHandler(), desktop.WithTitle("My App"), desktop.WithSize(1024, 768))
}
```

## Full API

For Go↔JS bindings and events, use `desktop.New` + `dsk.Run`:

```go
func init() { runtime.LockOSThread() }

func main() {
    app, _ := dark.New(/* ... */)
    defer app.Close()
    // ... register routes, islands, middleware ...

    dsk := desktop.New(app.MustHandler(),
        desktop.WithTitle("My App"),
        desktop.WithSize(1280, 800),
        desktop.WithMinSize(640, 480),
        desktop.WithDebug(true),
        desktop.WithOnReady(func(url string) { fmt.Println("Running at", url) }),
    )

    // Expose Go functions to JavaScript (appear as globals, return Promises)
    dsk.Bind("readFile", func(path string) (string, error) {
        data, err := os.ReadFile(path)
        return string(data), err
    })

    // Bind all exported methods of a struct
    dsk.BindMethods("api", myService) // → window.api_get_user(), api_list_items(), etc.

    // Listen for events from frontend
    dsk.On("save", func(data any) {
        fmt.Println("save requested:", data)
    })

    // Wait for WebView to be ready before emitting events
    go func() {
        <-dsk.Ready()
        dsk.Emit("notify", map[string]any{"message": "Hello from Go!"})
    }()

    dsk.Run() // blocks until window closed
}
```

## Go↔JS Bindings

### Bind

Expose a Go function as a global JavaScript function. The function appears as `window.<name>(...)` and returns a Promise.

```go
dsk.Bind("greet", func(name string) string {
    return "Hello, " + name
})
```

```javascript
const msg = await greet("World"); // "Hello, World"
```

Functions may accept JSON-serializable arguments and return nothing, a value, an error, or (value, error).

### BindMethods

Expose all exported methods of a struct. Method names are converted to snake_case.

```go
type UserService struct { /* ... */ }
func (s *UserService) GetByID(id int) (*User, error) { /* ... */ }
func (s *UserService) ListAll() []User { /* ... */ }

dsk.BindMethods("users", &UserService{})
```

```javascript
const user = await users_get_by_id(42);
const all  = await users_list_all();
```

## Events

Bidirectional event system between Go and JavaScript.

### Go side

```go
// Listen for events from frontend
dsk.On("save", func(data any) {
    fmt.Println("save:", data)
})

// Send events to frontend (safe from any goroutine)
dsk.Emit("notify", map[string]any{"message": "Done!"})
```

### JavaScript side

The `window.dark` API is auto-injected into every page:

```javascript
// Listen for events from Go
dark.on("notify", (data) => {
    console.log(data.message); // "Done!"
});

// Unsubscribe
dark.off("notify");          // remove all listeners
dark.off("notify", handler); // remove specific listener

// Send events to Go
dark.emit("save", { draft: true });
```

## App Lifecycle

`Ready()` returns a channel that closes once the WebView and JS bridge are initialized. Use it to safely start background work:

```go
go func() {
    <-dsk.Ready()
    dsk.Emit("started", nil)
    dsk.SetTitle("Ready!")
}()
dsk.Run()
```

## Native Features

Built-in support for file dialogs, clipboard, and OS notifications — available from JavaScript via `window.dark.*` with no setup required.

### File Dialogs

Native open/save dialogs powered by [zenity](https://github.com/ncruces/zenity) (pure Go, no CGo).

```javascript
// Open a file (returns path or "" if cancelled)
const path = await dark.openFile({ title: "Open", filters: ["*.txt", "*.md"] });

// Open multiple files
const paths = await dark.openFiles({ title: "Select files" });

// Save file dialog
const savePath = await dark.saveFile({ title: "Save as", filename: "doc.txt" });

// Pick a folder
const dir = await dark.pickFolder({ title: "Select folder" });
```

Options: `title` (dialog title), `filename` (initial filename), `filters` (glob patterns).

### Clipboard

Read and write the system clipboard via [atotto/clipboard](https://github.com/atotto/clipboard).

```javascript
const text = await dark.readClipboard();
await dark.writeClipboard("copied text");
```

### OS Notifications

Native desktop notifications via [beeep](https://github.com/gen2brain/beeep).

```javascript
await dark.notify("Title", "Message body");
```

### External Links

Links to external URLs automatically open in the system's default browser instead of navigating inside the webview. Any `<a href="https://...">` pointing outside the internal server is intercepted.

```javascript
// Programmatic usage
await dark.openExternal("https://example.com");
```

Only `http` and `https` URLs are supported. Other schemes (`file://`, `javascript:`, etc.) are ignored for security.

## Window Control

### From JavaScript

```javascript
dark.setTitle("New Title");
dark.close();
```

### From Go

These methods are safe to call from any goroutine:

```go
dsk.SetTitle("Updated")
dsk.SetSize(800, 600)
dsk.Eval("console.log('hello')")
dsk.Terminate()
```

## Window Options

```go
desktop.WithTitle("App")           // window title (default: "App")
desktop.WithSize(1024, 768)        // initial dimensions (default: 1024x768)
desktop.WithMinSize(400, 300)      // minimum window size
desktop.WithMaxSize(1920, 1080)    // maximum window size
desktop.WithFixedSize()            // non-resizable window
desktop.WithDebug(true)            // enable browser DevTools
desktop.WithAddr("127.0.0.1:0")   // HTTP listen address (default: random port)
desktop.WithOnReady(func(url string) { ... })
```

## Threading

The WebView requires the main OS thread. Your `main` package must include:

```go
func init() { runtime.LockOSThread() }
```

`Run()` calls `runtime.LockOSThread()` internally as well. All `App` methods (`SetTitle`, `SetSize`, `Eval`, `Emit`, `Terminate`) are safe to call from any goroutine — they use `Dispatch` internally to post work to the UI thread.

## Example

See [`_examples/desktop-demo/`](../_examples/desktop-demo/) for a full example combining Islands, htmx, sessions, form validation, Go↔JS bindings, desktop events, file dialogs, clipboard, and notifications.
