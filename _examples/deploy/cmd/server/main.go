// Production-ready dark application entry point.
//
// Environment variables:
//
//	PORT             - HTTP port (default "3000")
//	SESSION_SECRET   - HMAC key for session cookies (required in production)
//	DARK_DEV         - Set to "1" to enable dev mode
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/i2y/dark"
)

func main() {
	port := envOr("PORT", "3000")
	devMode := os.Getenv("DARK_DEV") == "1"

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" && !devMode {
		log.Fatal("SESSION_SECRET is required in production")
	}
	if secret == "" {
		secret = "dev-insecure-secret"
	}

	app, err := dark.New(
		dark.WithTemplateDir("views"),
		dark.WithLayout("layouts/default.tsx"),
		dark.WithDevMode(devMode),
		// Enable SSR caching in production for better performance.
		dark.WithSSRCache(1000),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	app.Use(dark.Logger())
	app.Use(app.RecoverWithErrorPage())
	app.Use(dark.Sessions([]byte(secret),
		dark.SessionSecure(!devMode), // Secure cookies in production (HTTPS)
	))
	app.Static("/static/", "public")

	// Register your routes here.
	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"message": "Hello from production!",
			}, nil
		},
	})

	app.APIGet("/api/health", dark.APIRoute{
		Handler: func(ctx dark.Context) error {
			return ctx.JSON(200, map[string]any{"status": "ok"})
		},
	})

	fmt.Printf("Listening on :%s (dev=%v)\n", port, devMode)
	log.Fatal(http.ListenAndServe(":"+port, app.Handler()))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
