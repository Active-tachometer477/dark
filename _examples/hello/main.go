package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/i2y/dark"
)

// --- In-memory data stores ---

type Task struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Done     bool   `json:"done"`
}

type BlogPost struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	Body    string `json:"body"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// --- Props types (used for TypeScript type generation) ---

type IndexPageProps struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type BlogListProps struct {
	Posts []BlogPost `json:"posts"`
}

type BlogPostProps struct {
	Post *BlogPost `json:"post"`
}

type ContactProps struct {
	Success string `json:"success,omitempty"`
}

type TaskPageProps struct {
	Tasks []Task `json:"tasks"`
}

var (
	taskMu  sync.Mutex
	taskSeq = 3
	tasks   = []Task{
		{ID: 1, Title: "Try nested layouts", Priority: "high", Done: false},
		{ID: 2, Title: "Test form validation", Priority: "medium", Done: false},
		{ID: 3, Title: "Check streaming SSR", Priority: "low", Done: true},
	}

	blogPosts = []BlogPost{
		{
			Slug:    "hello-dark",
			Title:   "Introducing Dark Framework",
			Excerpt: "A new Go SSR web framework powered by Preact and htmx.",
			Body:    "Dark is a Go web framework that brings the best of modern frontend to server-side rendering.\n\nIt uses Preact for templating, htmx for interactivity, and Islands architecture for selective hydration.\n\nWith Phase 5, Dark now supports nested layouts, form validation, head/meta management, and streaming SSR.",
			Author:  "Dark Team",
			Date:    "2026-04-02",
		},
		{
			Slug:    "streaming-ssr",
			Title:   "Faster TTFB with Streaming SSR",
			Excerpt: "How shell-first rendering improves perceived performance.",
			Body:    "Streaming SSR sends the HTML shell (header, navigation, CSS links) to the browser before the page component finishes rendering.\n\nThis means the browser can start parsing CSS and rendering the layout while the server is still computing the page content.\n\nIn Dark, enable it globally with WithStreaming(true) or per-route with Route.Streaming.",
			Author:  "Dark Team",
			Date:    "2026-04-02",
		},
		{
			Slug:    "nested-layouts",
			Title:   "Composable Layouts with Route Groups",
			Excerpt: "Build complex UIs with nested, composable layout components.",
			Body:    "Dark's layout system lets you nest layouts at multiple levels.\n\nThe global layout (WithLayout) provides the HTML shell. Route groups (app.Group) add shared section layouts like admin sidebars. Individual routes can add their own layout on top.\n\nAll layouts receive the same props and compose via Preact's children prop.",
			Author:  "Dark Team",
			Date:    "2026-04-02",
		},
	}
)

func findPost(slug string) *BlogPost {
	for i := range blogPosts {
		if blogPosts[i].Slug == slug {
			return &blogPosts[i]
		}
	}
	return nil
}

func main() {
	streaming := true

	app, err := dark.New(
		dark.WithLayout("layouts/default.tsx"),
		dark.WithTemplateDir("views"),
		dark.WithErrorComponent("errors/500.tsx"),
		dark.WithNotFoundComponent("errors/404.tsx"),
		dark.WithDevMode(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	app.Use(dark.Logger())
	app.Use(app.RecoverWithErrorPage())
	app.Use(dark.Sessions([]byte("demo-secret-change-in-production")))

	app.Static("/static/", "public")
	app.Island("counter", "islands/counter.tsx")

	// ==============================
	// Home page
	// ==============================
	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Dark — Go SSR Framework")
			ctx.AddMeta("description", "A Go SSR web framework powered by Preact, htmx, and Islands architecture")
			sess := ctx.Session()
			return map[string]any{
				"message": "Welcome to Dark",
				"count":   0,
				"user":    sess.Get("user"),
				"flashes": sess.Flashes(),
			}, nil
		},
	})

	// ==============================
	// Login / Logout — demonstrates Session helpers
	// ==============================
	app.Get("/login", dark.Route{
		Component: "pages/login.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Login | Dark App")
			if ctx.Session().Get("user") != nil {
				ctx.Redirect("/")
				return nil, nil
			}
			return nil, nil
		},
	})

	app.Post("/login", dark.Route{
		Component: "pages/login.tsx",
		Action: func(ctx dark.Context) error {
			username := strings.TrimSpace(ctx.FormData().Get("username"))
			if username == "" {
				ctx.AddFieldError("username", "Username is required")
				return nil
			}
			ctx.Session().Set("user", username)
			ctx.Session().Flash("notice", "Welcome back, "+username+"!")
			return ctx.Redirect("/")
		},
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Login | Dark App")
			return nil, nil
		},
	})

	app.Post("/logout", dark.Route{
		Action: func(ctx dark.Context) error {
			ctx.Session().Clear()
			ctx.Session().Flash("notice", "You have been logged out.")
			return ctx.Redirect("/")
		},
	})

	// ==============================
	// Blog — demonstrates Head/Meta + Streaming SSR
	// ==============================
	app.Get("/blog", dark.Route{
		Component: "pages/blog_list.tsx",
		Props:     BlogListProps{},
		Streaming: &streaming,
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Blog | Dark App")
			ctx.AddMeta("description", "Articles about the Dark framework")
			return BlogListProps{Posts: blogPosts}, nil
		},
	})

	app.Get("/blog/{slug}", dark.Route{
		Component: "pages/blog_post.tsx",
		Props:     BlogPostProps{},
		Streaming: &streaming,
		Loader: func(ctx dark.Context) (any, error) {
			post := findPost(ctx.Param("slug"))
			if post == nil {
				return nil, dark.NewAPIError(404, "Post not found")
			}
			return BlogPostProps{Post: post}, nil
		},
	})

	// ==============================
	// Contact — demonstrates Form Validation
	// ==============================
	app.Get("/contact", dark.Route{
		Component: "pages/contact.tsx",
		Props:     ContactProps{},
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Contact | Dark App")
			return nil, nil
		},
	})

	app.Post("/contact", dark.Route{
		Component: "pages/contact.tsx",
		Action: func(ctx dark.Context) error {
			form := ctx.FormData()
			name := strings.TrimSpace(form.Get("name"))
			email := strings.TrimSpace(form.Get("email"))
			message := strings.TrimSpace(form.Get("message"))

			if name == "" {
				ctx.AddFieldError("name", "Name is required")
			}
			if email == "" {
				ctx.AddFieldError("email", "Email is required")
			} else if !strings.Contains(email, "@") {
				ctx.AddFieldError("email", "Please enter a valid email address")
			}
			if message == "" {
				ctx.AddFieldError("message", "Message is required")
			} else if len(message) < 10 {
				ctx.AddFieldError("message", "Message must be at least 10 characters")
			}

			if ctx.HasErrors() {
				return nil
			}

			// Success! In a real app, you'd send the email here.
			log.Printf("Contact form: name=%s email=%s message=%s", name, email, message)
			return nil
		},
		Loader: func(ctx dark.Context) (any, error) {
			ctx.SetTitle("Contact | Dark App")
			if !ctx.HasErrors() && ctx.Request().Method == "POST" {
				return map[string]any{
					"success": ctx.FormData().Get("name"),
				}, nil
			}
			return nil, nil
		},
	})

	// ==============================
	// Admin — demonstrates Nested Layouts + Group + Auth + Form Validation
	// ==============================
	app.Group("/admin", "layouts/admin.tsx", func(g *dark.Group) {
		g.Use(dark.RequireAuth())

		// Task list
		g.Get("/tasks", dark.Route{
			Component: "pages/admin_tasks.tsx",
			Props:     TaskPageProps{},
			Loader: func(ctx dark.Context) (any, error) {
				ctx.SetTitle("Tasks | Admin | Dark App")
				taskMu.Lock()
				defer taskMu.Unlock()
				// Copy tasks so we don't hold the lock during render.
				t := make([]Task, len(tasks))
				copy(t, tasks)
				return map[string]any{"tasks": t}, nil
			},
		})

		// Add task (POST)
		g.Post("/tasks", dark.Route{
			Component: "pages/admin_tasks.tsx",
			Action: func(ctx dark.Context) error {
				title := strings.TrimSpace(ctx.FormData().Get("title"))
				if title == "" {
					ctx.AddFieldError("title", "Task title is required")
					return nil
				}
				priority := ctx.FormData().Get("priority")
				if priority == "" {
					priority = "medium"
				}
				taskMu.Lock()
				taskSeq++
				tasks = append(tasks, Task{
					ID:       taskSeq,
					Title:    title,
					Priority: priority,
					Done:     false,
				})
				taskMu.Unlock()
				return nil
			},
			Loader: func(ctx dark.Context) (any, error) {
				ctx.SetTitle("Tasks | Admin | Dark App")
				taskMu.Lock()
				defer taskMu.Unlock()
				t := make([]Task, len(tasks))
				copy(t, tasks)
				return map[string]any{"tasks": t}, nil
			},
		})

		// Toggle task
		g.Post("/tasks/{id}/toggle", dark.Route{
			Component: "pages/admin_tasks.tsx",
			Action: func(ctx dark.Context) error {
				id := ctx.Param("id")
				taskMu.Lock()
				defer taskMu.Unlock()
				for i := range tasks {
					if fmt.Sprint(tasks[i].ID) == id {
						tasks[i].Done = !tasks[i].Done
						break
					}
				}
				return nil
			},
			Loader: func(ctx dark.Context) (any, error) {
				taskMu.Lock()
				defer taskMu.Unlock()
				t := make([]Task, len(tasks))
				copy(t, tasks)
				return map[string]any{"tasks": t}, nil
			},
		})

		// Delete task
		g.Delete("/tasks/{id}", dark.Route{
			Component: "pages/admin_tasks.tsx",
			Action: func(ctx dark.Context) error {
				id := ctx.Param("id")
				taskMu.Lock()
				defer taskMu.Unlock()
				for i := range tasks {
					if fmt.Sprint(tasks[i].ID) == id {
						tasks = append(tasks[:i], tasks[i+1:]...)
						break
					}
				}
				return nil
			},
			Loader: func(ctx dark.Context) (any, error) {
				taskMu.Lock()
				defer taskMu.Unlock()
				t := make([]Task, len(tasks))
				copy(t, tasks)
				return map[string]any{"tasks": t}, nil
			},
		})

		// Settings page (nested layout demo)
		g.Get("/settings", dark.Route{
			Component: "pages/admin_settings.tsx",
		})
	})

	// ==============================
	// Broken page — for testing dev overlay error display
	// ==============================
	app.Get("/broken", dark.Route{
		Component: "pages/broken.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			// Intentionally pass nil items to trigger a TypeError in TSX
			return map[string]any{"items": nil}, nil
		},
	})

	// ==============================
	// API routes
	// ==============================
	app.APIGet("/api/status", dark.APIRoute{
		Handler: func(ctx dark.Context) error {
			return ctx.JSON(200, map[string]any{
				"status":  "ok",
				"version": "0.5.0",
			})
		},
	})

	fmt.Println("Listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", app.Handler()))
}
