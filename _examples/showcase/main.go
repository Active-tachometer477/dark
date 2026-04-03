// showcase demonstrates several dark framework features:
//
//   - Handler() (http.Handler, error) / MustHandler()
//   - WithLogger(*slog.Logger) — structured logging
//   - CSRF middleware with htmx auto-injection
//   - Context.Set / Context.Get — request-scoped values
//   - Concurrent Loaders (Route.Loaders)
//   - LRU SSR Cache + ETag / 304 Not Modified
//   - Static Site Generation (SSG)
package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/i2y/dark"
)

// ---------------------------------------------------------------------------
// Simulated data sources (each takes ~50ms to mimic a real service call)
// ---------------------------------------------------------------------------

func fetchUserProfile(userID string) map[string]any {
	time.Sleep(50 * time.Millisecond)
	return map[string]any{
		"id":     userID,
		"name":   "Alice",
		"email":  "alice@example.com",
		"avatar": "https://i.pravatar.cc/80?u=" + userID,
	}
}

func fetchRecentActivity(userID string) []map[string]any {
	time.Sleep(50 * time.Millisecond)
	return []map[string]any{
		{"action": "Logged in", "time": "2 minutes ago"},
		{"action": "Updated profile", "time": "1 hour ago"},
		{"action": "Created post", "time": "3 hours ago"},
	}
}

func fetchNotifications(userID string) []map[string]any {
	time.Sleep(50 * time.Millisecond)
	return []map[string]any{
		{"message": "New comment on your post", "unread": true},
		{"message": "Your report is ready", "unread": false},
	}
}

func fetchStats() map[string]any {
	time.Sleep(50 * time.Millisecond)
	return map[string]any{
		"visitors":  rand.Intn(5000) + 1000,
		"pageViews": rand.Intn(20000) + 5000,
		"signups":   rand.Intn(100) + 10,
	}
}

// ---------------------------------------------------------------------------
// Middleware: inject requestID via Context.Set / Context.Get
// ---------------------------------------------------------------------------

func requestIDMiddleware() dark.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use dark.SetValue to store a request-scoped value
			// that Loaders can retrieve via ctx.Get("requestID").
			reqID := fmt.Sprintf("req-%d", time.Now().UnixNano()%1_000_000)
			r = dark.SetValue(r, "requestID", reqID)
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	ssgMode := flag.Bool("ssg", false, "Generate static site to dist/ and exit")
	flag.Parse()

	// 1. WithLogger: custom structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app, err := dark.New(
		dark.WithTemplateDir("views"),
		dark.WithLayout("layouts/default.tsx"),
		dark.WithDevMode(!*ssgMode),
		dark.WithPoolSize(2),
		dark.WithSSRCache(100),       // Enable LRU SSR cache (+ ETag)
		dark.WithLogger(logger),      // Structured logging
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	// 2. Middleware stack: Logger → Recover → Sessions → CSRF → RequestID
	app.Use(dark.Logger())
	app.Use(dark.Recover())
	app.Use(dark.Sessions([]byte("showcase-secret-key-32-bytes!!")))
	app.Use(dark.CSRF()) // Auto-injects meta tag + htmx config
	app.Use(requestIDMiddleware())

	app.Static("/static/", "public")

	// ---------------------------------------------------------------------------
	// Routes
	// ---------------------------------------------------------------------------

	// Home page: single Loader
	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"title":     "Dark Showcase",
				"requestID": ctx.Get("requestID"), // Retrieved from middleware
			}, nil
		},
	})

	// Dashboard: concurrent Loaders (3 data sources fetched in parallel ~50ms total)
	app.Get("/dashboard", dark.Route{
		Component: "pages/dashboard.tsx",
		Loaders: []dark.LoaderFunc{
			func(ctx dark.Context) (any, error) {
				return map[string]any{"user": fetchUserProfile("42")}, nil
			},
			func(ctx dark.Context) (any, error) {
				return map[string]any{"activity": fetchRecentActivity("42")}, nil
			},
			func(ctx dark.Context) (any, error) {
				return map[string]any{"notifications": fetchNotifications("42")}, nil
			},
		},
	})

	// Stats: cached page with ETag support
	app.Get("/stats", dark.Route{
		Component: "pages/stats.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return fetchStats(), nil
		},
	})

	// Contact form: demonstrates CSRF protection with htmx
	app.Get("/contact", dark.Route{
		Component: "pages/contact.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{}, nil
		},
	})

	app.Post("/contact", dark.Route{
		Component: "pages/contact.tsx",
		Action: func(ctx dark.Context) error {
			name := ctx.FormData().Get("name")
			message := ctx.FormData().Get("message")
			if name == "" {
				ctx.AddFieldError("name", "Name is required")
			}
			if message == "" {
				ctx.AddFieldError("message", "Message is required")
			}
			if ctx.HasErrors() {
				return nil
			}
			logger.Info("contact form submitted", "name", name)
			return ctx.Redirect("/contact?success=1")
		},
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"success": ctx.Query("success") == "1",
			}, nil
		},
	})

	// ---------------------------------------------------------------------------
	// SSG mode: generate static site and exit
	// ---------------------------------------------------------------------------

	if *ssgMode {
		logger.Info("generating static site...")
		err := app.GenerateStaticSite("dist", []dark.StaticRoute{
			{
				Path:      "/",
				Component: "pages/index.tsx",
				Loader: func(ctx dark.Context) (any, error) {
					return map[string]any{
						"title":     "Dark Showcase (Static)",
						"requestID": "static-build",
					}, nil
				},
			},
			{
				Path:      "/stats",
				Component: "pages/stats.tsx",
				Loader: func(ctx dark.Context) (any, error) {
					return map[string]any{
						"visitors":  12345,
						"pageViews": 67890,
						"signups":   42,
					}, nil
				},
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		logger.Info("static site generated", "dir", "dist")
		return
	}

	// ---------------------------------------------------------------------------
	// 3. Handler() returns (http.Handler, error) — proper error handling
	// ---------------------------------------------------------------------------

	handler, err := app.Handler()
	if err != nil {
		log.Fatal("failed to build handler: ", err)
	}

	fmt.Println("Showcase running on http://localhost:3000")
	fmt.Println("  GET /           — Home (Context.Set/Get, CSRF meta)")
	fmt.Println("  GET /dashboard  — Concurrent Loaders (3 sources in ~50ms)")
	fmt.Println("  GET /stats      — SSR Cache + ETag")
	fmt.Println("  GET /contact    — CSRF-protected form + htmx")
	fmt.Println("  --ssg flag      — Static Site Generation")
	log.Fatal(http.ListenAndServe(":3000", handler))
}
