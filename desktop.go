package dark

import (
	"fmt"
	"net"
	"net/http"
	"runtime"

	"github.com/crgimenes/glaze"
	_ "github.com/crgimenes/glaze/embedded"
)

// DesktopOptions configures a desktop window.
type DesktopOptions struct {
	// Title is the window title. Defaults to "App".
	Title string

	// Width and Height set the initial window dimensions.
	// Defaults to 1024x768.
	Width  int
	Height int

	// Debug enables browser developer tools.
	Debug bool

	// Addr is the listen address for the local HTTP server.
	// Defaults to "127.0.0.1:0" (random port).
	Addr string

	// OnReady is called with the local URL once the server is listening.
	OnReady func(url string)
}

// Desktop starts a local HTTP server with the app's handler, opens a native
// WebView window pointing to it, and blocks until the window is closed.
//
// The caller must pin the main goroutine to the main OS thread before calling
// this function. The recommended pattern:
//
//	func init() { runtime.LockOSThread() }
//
//	func main() {
//	    app, _ := dark.New(...)
//	    // ... register routes ...
//	    dark.Desktop(app, dark.DesktopOptions{Title: "My App"})
//	}
func Desktop(app *App, opts DesktopOptions) error {
	runtime.LockOSThread()

	if opts.Title == "" {
		opts.Title = "App"
	}
	if opts.Width <= 0 {
		opts.Width = 1024
	}
	if opts.Height <= 0 {
		opts.Height = 768
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:0"
	}

	handler, err := app.Handler()
	if err != nil {
		return fmt.Errorf("dark: desktop: %w", err)
	}

	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return fmt.Errorf("dark: desktop: listen %s: %w", opts.Addr, err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	srv := &http.Server{Handler: handler}
	defer srv.Close()
	go srv.Serve(ln)

	if opts.OnReady != nil {
		opts.OnReady(url)
	}

	wv, err := glaze.New(opts.Debug)
	if err != nil {
		return fmt.Errorf("dark: desktop: webview: %w", err)
	}

	wv.SetTitle(opts.Title)
	wv.SetSize(opts.Width, opts.Height, glaze.HintNone)
	wv.Navigate(url)
	wv.Run()
	wv.Destroy()

	return nil
}
