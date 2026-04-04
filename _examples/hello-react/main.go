package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/i2y/dark"
)

func main() {
	app, err := dark.New(
		dark.WithUILibrary(dark.React),
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

	app.Static("/static/", "public")
	app.Island("counter", "islands/counter.tsx")

	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"title":   "Hello React + Dark",
				"message": "This page is server-rendered with React and hydrated with Islands.",
				"count":   42,
			}, nil
		},
	})

	app.Get("/about", dark.Route{
		Component: "pages/about.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			return map[string]any{
				"title": "About",
				"features": []string{
					"React SSR via renderToString",
					"Islands architecture with hydrateRoot",
					"htmx for HTML-over-the-wire",
					"Go Loader/Action for data",
				},
			}, nil
		},
	})

	fmt.Println("Listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", app.MustHandler()))
}
