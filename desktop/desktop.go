// Package desktop provides a native desktop window for dark applications.
// Import this package only when building desktop apps — it pulls in the
// WebView native library dependency.
//
// Simple usage with convenience Run:
//
//	func init() { runtime.LockOSThread() }
//
//	func main() {
//	    app, _ := dark.New(...)
//	    desktop.Run(app.MustHandler(), desktop.WithTitle("My App"))
//	}
//
// Full usage with Go↔JS bridge and events:
//
//	func init() { runtime.LockOSThread() }
//
//	func main() {
//	    app, _ := dark.New(...)
//	    dsk := desktop.New(app.MustHandler(),
//	        desktop.WithTitle("My App"),
//	        desktop.WithSize(1280, 800),
//	        desktop.WithMinSize(640, 480),
//	        desktop.WithDebug(true),
//	    )
//	    dsk.Bind("greet", func(name string) string {
//	        return "Hello, " + name
//	    })
//	    dsk.On("save", func(data any) {
//	        fmt.Println("save:", data)
//	    })
//	    dsk.Run()
//	}
package desktop

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"

	"github.com/crgimenes/glaze"
	_ "github.com/crgimenes/glaze/embedded"
)

type config struct {
	title     string
	width     int
	height    int
	minWidth  int
	minHeight int
	maxWidth  int
	maxHeight int
	fixedSize bool
	debug     bool
	addr      string
	onReady   func(url string)
}

func defaultConfig() *config {
	return &config{
		title:  "App",
		width:  1024,
		height: 768,
		addr:   "127.0.0.1:0",
	}
}

// Option configures the desktop App.
type Option func(*config)

// WithTitle sets the window title. Defaults to "App".
func WithTitle(title string) Option {
	return func(c *config) { c.title = title }
}

// WithSize sets the initial window dimensions. Defaults to 1024×768.
func WithSize(w, h int) Option {
	return func(c *config) { c.width = w; c.height = h }
}

// WithMinSize sets the minimum window dimensions.
func WithMinSize(w, h int) Option {
	return func(c *config) { c.minWidth = w; c.minHeight = h }
}

// WithMaxSize sets the maximum window dimensions.
func WithMaxSize(w, h int) Option {
	return func(c *config) { c.maxWidth = w; c.maxHeight = h }
}

// WithFixedSize makes the window non-resizable.
func WithFixedSize() Option {
	return func(c *config) { c.fixedSize = true }
}

// WithDebug enables browser developer tools in the WebView.
func WithDebug(debug bool) Option {
	return func(c *config) { c.debug = debug }
}

// WithAddr sets the listen address for the internal HTTP server.
// Defaults to "127.0.0.1:0" (random port).
func WithAddr(addr string) Option {
	return func(c *config) { c.addr = addr }
}

// WithOnReady registers a callback invoked with the local URL once the
// HTTP server is listening.
func WithOnReady(fn func(url string)) Option {
	return func(c *config) { c.onReady = fn }
}

// App wraps an http.Handler in a native desktop window with optional
// Go↔JS bindings and a bidirectional event system.
type App struct {
	handler http.Handler
	cfg     *config

	mu       sync.Mutex
	wv       glaze.WebView // set during Run, nil before; guarded by mu
	bindings []pendingBind
	methods  []pendingMethods
	handlers  map[string][]func(data any)
	ready     chan struct{}
	readyOnce sync.Once
}

// webview returns the current WebView, or nil if Run has not started.
func (a *App) webview() glaze.WebView {
	a.mu.Lock()
	wv := a.wv
	a.mu.Unlock()
	return wv
}

// New creates a desktop App. Call Bind, BindMethods, and On to configure
// the Go↔JS bridge, then call Run to open the window.
func New(handler http.Handler, opts ...Option) *App {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}
	return &App{
		handler:  handler,
		cfg:      cfg,
		handlers: make(map[string][]func(data any)),
		ready:    make(chan struct{}),
	}
}

// Run starts the internal HTTP server, opens a native WebView window,
// and blocks until the window is closed.
//
// The caller must pin the main goroutine to the main OS thread:
//
//	func init() { runtime.LockOSThread() }
func (a *App) Run() error {
	runtime.LockOSThread()

	ln, err := net.Listen("tcp", a.cfg.addr)
	if err != nil {
		return fmt.Errorf("desktop: listen %s: %w", a.cfg.addr, err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	srv := &http.Server{Handler: a.handler}
	defer srv.Close()
	go srv.Serve(ln)

	if a.cfg.onReady != nil {
		a.cfg.onReady(url)
	}

	wv, err := glaze.New(a.cfg.debug)
	if err != nil {
		return fmt.Errorf("desktop: webview: %w", err)
	}

	a.mu.Lock()
	a.wv = wv
	a.mu.Unlock()

	if err := a.applyBindings(); err != nil {
		wv.Destroy()
		return err
	}

	a.setupBridge()
	a.setupFeatures()
	a.readyOnce.Do(func() { close(a.ready) })

	wv.SetTitle(a.cfg.title)

	if a.cfg.fixedSize {
		wv.SetSize(a.cfg.width, a.cfg.height, glaze.HintFixed)
	} else {
		wv.SetSize(a.cfg.width, a.cfg.height, glaze.HintNone)
	}
	if a.cfg.minWidth > 0 && a.cfg.minHeight > 0 {
		wv.SetSize(a.cfg.minWidth, a.cfg.minHeight, glaze.HintMin)
	}
	if a.cfg.maxWidth > 0 && a.cfg.maxHeight > 0 {
		wv.SetSize(a.cfg.maxWidth, a.cfg.maxHeight, glaze.HintMax)
	}

	wv.Navigate(url)
	wv.Run()
	wv.Destroy()

	return nil
}

// Ready returns a channel that is closed once the WebView and JS bridge are
// initialized. Use this to safely wait before calling Emit or other methods
// from background goroutines:
//
//	go func() {
//	    <-dsk.Ready()
//	    dsk.Emit("started", nil)
//	}()
func (a *App) Ready() <-chan struct{} {
	return a.ready
}

// SetTitle changes the window title at runtime. Safe to call from any goroutine.
func (a *App) SetTitle(title string) {
	wv := a.webview()
	if wv == nil {
		return
	}
	wv.Dispatch(func() { wv.SetTitle(title) })
}

// SetSize changes the window dimensions at runtime. Safe to call from any goroutine.
func (a *App) SetSize(w, h int) {
	wv := a.webview()
	if wv == nil {
		return
	}
	wv.Dispatch(func() { wv.SetSize(w, h, glaze.HintNone) })
}

// Eval executes JavaScript in the WebView. Safe to call from any goroutine.
func (a *App) Eval(js string) {
	wv := a.webview()
	if wv == nil {
		return
	}
	wv.Dispatch(func() { wv.Eval(js) })
}

// Terminate closes the window and stops the event loop.
// Safe to call from any goroutine.
func (a *App) Terminate() {
	wv := a.webview()
	if wv == nil {
		return
	}
	wv.Terminate()
}

// Run is a convenience function for simple desktop apps that don't need
// Go↔JS bindings or events. For the full API, use New instead.
func Run(handler http.Handler, opts ...Option) error {
	return New(handler, opts...).Run()
}
