package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/i2y/dark"
	"github.com/i2y/dark/desktop"
)

func init() { runtime.LockOSThread() }

type Note struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

var (
	mu      sync.Mutex
	noteSeq int
	notes   = []Note{
		{ID: 1, Title: "Welcome to Dark Desktop", Body: "This app combines SSR, Islands, htmx, and native desktop bindings.", CreatedAt: "2026-04-11 10:00"},
		{ID: 2, Title: "Try the features", Body: "Add notes with htmx, use the live clock Island, export via Go bindings.", CreatedAt: "2026-04-11 10:01"},
	}
)

func init() { noteSeq = len(notes) }

func main() {
	app, err := dark.New(
		dark.WithLayout("layouts/default.tsx"),
		dark.WithTemplateDir("views"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	app.Use(dark.Logger())
	app.Use(dark.Sessions([]byte("desktop-demo-secret")))

	app.Island("clock", "islands/clock.tsx")

	// --- Pages ---

	app.Get("/", dark.Route{
		Component: "pages/index.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			mu.Lock()
			n := make([]Note, len(notes))
			copy(n, notes)
			mu.Unlock()

			sess := ctx.Session()
			return map[string]any{
				"notes":    n,
				"username": sess.Get("username"),
				"flashes":  sess.Flashes(),
			}, nil
		},
	})

	// Add note (htmx)
	app.Post("/notes", dark.Route{
		Component: "pages/index.tsx",
		Action: func(ctx dark.Context) error {
			title := strings.TrimSpace(ctx.FormData().Get("title"))
			body := strings.TrimSpace(ctx.FormData().Get("body"))

			if title == "" {
				ctx.AddFieldError("title", "Title is required")
			} else if len(title) < 2 {
				ctx.AddFieldError("title", "Title must be at least 2 characters")
			}

			if ctx.HasErrors() {
				return nil
			}

			mu.Lock()
			noteSeq++
			notes = append(notes, Note{
				ID:        noteSeq,
				Title:     title,
				Body:      body,
				CreatedAt: time.Now().Format("2006-01-02 15:04"),
			})
			mu.Unlock()

			ctx.Session().Flash("notice", fmt.Sprintf("Note \"%s\" added!", title))
			return nil
		},
		Loader: func(ctx dark.Context) (any, error) {
			mu.Lock()
			n := make([]Note, len(notes))
			copy(n, notes)
			mu.Unlock()

			return map[string]any{
				"notes":    n,
				"username": ctx.Session().Get("username"),
				"flashes":  ctx.Session().Flashes(),
			}, nil
		},
	})

	// Delete note (htmx)
	app.Delete("/notes/{id}", dark.Route{
		Component: "pages/index.tsx",
		Action: func(ctx dark.Context) error {
			id := ctx.Param("id")
			mu.Lock()
			for i := range notes {
				if fmt.Sprint(notes[i].ID) == id {
					notes = append(notes[:i], notes[i+1:]...)
					break
				}
			}
			mu.Unlock()
			return nil
		},
		Loader: func(ctx dark.Context) (any, error) {
			mu.Lock()
			n := make([]Note, len(notes))
			copy(n, notes)
			mu.Unlock()

			return map[string]any{
				"notes":    n,
				"username": ctx.Session().Get("username"),
				"flashes":  ctx.Session().Flashes(),
			}, nil
		},
	})

	// Set username (session demo)
	app.Post("/username", dark.Route{
		Action: func(ctx dark.Context) error {
			name := strings.TrimSpace(ctx.FormData().Get("username"))
			if name != "" {
				ctx.Session().Set("username", name)
				ctx.Session().Flash("notice", fmt.Sprintf("Welcome, %s!", name))
			}
			return ctx.Redirect("/")
		},
	})

	// --- Desktop ---

	dsk := desktop.New(app.MustHandler(),
		desktop.WithTitle("Dark Notes"),
		desktop.WithSize(800, 900),
		desktop.WithMinSize(600, 500),
		desktop.WithDebug(true),
		desktop.WithOnReady(func(url string) {
			fmt.Println("Dark Notes running at", url)
		}),
	)

	// Export notes as JSON file
	dsk.Bind("export_notes", func() (string, error) {
		mu.Lock()
		data, err := json.MarshalIndent(notes, "", "  ")
		mu.Unlock()
		if err != nil {
			return "", err
		}

		dir, _ := os.UserHomeDir()
		path := filepath.Join(dir, "dark-notes-export.json")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", err
		}

		dsk.Emit("notification", map[string]any{
			"type":    "success",
			"message": fmt.Sprintf("Exported %d notes to %s", len(notes), path),
		})
		return path, nil
	})

	// Get system info
	dsk.Bind("system_info", func() map[string]any {
		hostname, _ := os.Hostname()
		dir, _ := os.UserHomeDir()
		return map[string]any{
			"hostname": hostname,
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"cpus":     runtime.NumCPU(),
			"goVersion": runtime.Version(),
			"homeDir":  dir,
		}
	})

	// Listen for title-update events from frontend
	dsk.On("update-title", func(data any) {
		if m, ok := data.(map[string]any); ok {
			if title, ok := m["title"].(string); ok {
				dsk.SetTitle(title)
			}
		}
	})

	if err := dsk.Run(); err != nil {
		log.Fatal(err)
	}
}
