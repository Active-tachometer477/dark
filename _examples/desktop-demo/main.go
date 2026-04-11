package main

import (
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/i2y/dark"
	"github.com/i2y/dark/desktop"
)

func init() { runtime.LockOSThread() }

func main() {
	app, err := dark.New(
		dark.WithLayout("layouts/default.tsx"),
		dark.WithTemplateDir("views"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"message": "Hello from Go desktop!",
				"time":    time.Now().Format("15:04:05"),
			}, nil
		},
	})

	dsk := desktop.New(app.MustHandler(),
		desktop.WithTitle("Dark Desktop"),
		desktop.WithSize(720, 860),
		desktop.WithMinSize(500, 400),
		desktop.WithDebug(true),
		desktop.WithOnReady(func(url string) {
			fmt.Println("Desktop app running at", url)
		}),
	)

	// --- Go↔JS Bindings ---

	// Simple string binding
	dsk.Bind("greet", func(name string) string {
		return fmt.Sprintf("Hello, %s! (from Go at %s)", name, time.Now().Format("15:04:05"))
	})

	// Numeric binding
	dsk.Bind("add", func(a, b int) int {
		return a + b
	})

	// Return current server time
	dsk.Bind("server_time", func() string {
		return time.Now().Format("2006-01-02 15:04:05 MST")
	})

	// --- Events ---

	// Listen for "clicked" events from frontend
	dsk.On("clicked", func(data any) {
		fmt.Printf("JS clicked event: %v\n", data)
	})

	// Binding that triggers a Go→JS event
	var counter atomic.Int64
	dsk.Bind("request_notify", func() {
		n := counter.Add(1)
		dsk.Emit("notify", map[string]any{
			"message": fmt.Sprintf("Notification #%d from Go!", n),
			"time":    time.Now().Format("15:04:05"),
		})
	})

	// Periodic Go→JS event (every 5 seconds).
	// Early ticks no-op safely if the window hasn't loaded yet.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		var count int
		for range ticker.C {
			count++
			dsk.Emit("counter", map[string]any{"count": count})
		}
	}()

	if err := dsk.Run(); err != nil {
		log.Fatal(err)
	}
}
