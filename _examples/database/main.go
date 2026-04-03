// Example: dark + SQLite database integration
//
// Demonstrates how to use dark with a relational database.
// Key patterns:
//   - Pass *sql.DB to handlers via closures
//   - Use Loaders for reads, Actions for writes
//   - Session-based auth guards DB routes
//
// Run: go run main.go
// Open: http://localhost:3000
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/i2y/dark"

	_ "modernc.org/sqlite"
)

type Todo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
	Owner string `json:"owner"`
}

func main() {
	db := initDB()
	defer db.Close()

	app, err := dark.New(
		dark.WithTemplateDir("views"),
		dark.WithLayout("layouts/default.tsx"),
		dark.WithDevMode(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	app.Use(dark.Logger())
	app.Use(app.RecoverWithErrorPage())
	app.Use(dark.Sessions([]byte("change-me-in-production")))

	// --- Public routes ---

	app.Get("/", dark.Route{
		Component: "pages/home.tsx",
		Loader: func(ctx dark.Context) (any, error) {
			sess := ctx.Session()
			return map[string]any{
				"user":    sess.Get("user"),
				"flashes": sess.Flashes(),
			}, nil
		},
	})

	app.Get("/login", dark.Route{
		Component: "pages/login.tsx",
	})

	app.Post("/login", dark.Route{
		Component: "pages/login.tsx",
		Action: func(ctx dark.Context) error {
			name := strings.TrimSpace(ctx.FormData().Get("username"))
			if name == "" {
				ctx.AddFieldError("username", "Username is required")
				return nil
			}
			ctx.Session().Set("user", name)
			ctx.Session().Flash("notice", "Logged in as "+name)
			return ctx.Redirect("/todos")
		},
	})

	app.Post("/logout", dark.Route{
		Action: func(ctx dark.Context) error {
			ctx.Session().Clear()
			return ctx.Redirect("/")
		},
	})

	// --- Protected routes (require login) ---

	app.Group("/todos", "", func(g *dark.Group) {
		g.Use(dark.RequireAuth())

		// List todos for the logged-in user.
		g.Get("", dark.Route{
			Component: "pages/todos.tsx",
			Loader: func(ctx dark.Context) (any, error) {
				user := ctx.Session().Get("user").(string)
				todos, err := listTodos(db, user)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"todos":   todos,
					"user":    user,
					"flashes": ctx.Session().Flashes(),
				}, nil
			},
		})

		// Add a new todo.
		g.Post("", dark.Route{
			Component: "pages/todos.tsx",
			Action: func(ctx dark.Context) error {
				title := strings.TrimSpace(ctx.FormData().Get("title"))
				if title == "" {
					ctx.AddFieldError("title", "Title is required")
					return nil
				}
				user := ctx.Session().Get("user").(string)
				if err := addTodo(db, user, title); err != nil {
					return err
				}
				ctx.Session().Flash("notice", "Added: "+title)
				return ctx.Redirect("/todos")
			},
			Loader: func(ctx dark.Context) (any, error) {
				user := ctx.Session().Get("user").(string)
				todos, err := listTodos(db, user)
				if err != nil {
					return nil, err
				}
				return map[string]any{"todos": todos, "user": user}, nil
			},
		})

		// Toggle todo done/undone.
		g.Post("/{id}/toggle", dark.Route{
			Action: func(ctx dark.Context) error {
				id := ctx.Param("id")
				user := ctx.Session().Get("user").(string)
				if err := toggleTodo(db, user, id); err != nil {
					return err
				}
				return ctx.Redirect("/todos")
			},
		})

		// Delete a todo.
		g.Delete("/{id}", dark.Route{
			Action: func(ctx dark.Context) error {
				id := ctx.Param("id")
				user := ctx.Session().Get("user").(string)
				if err := deleteTodo(db, user, id); err != nil {
					return err
				}
				ctx.Session().Flash("notice", "Deleted")
				return ctx.Redirect("/todos")
			},
		})
	})

	fmt.Println("Listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", app.MustHandler()))
}

// --- Database layer ---

func initDB() *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS todos (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT    NOT NULL,
		done  INTEGER NOT NULL DEFAULT 0,
		owner TEXT    NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}
	// Seed some data.
	for _, t := range []struct{ owner, title string }{
		{"alice", "Buy groceries"},
		{"alice", "Read dark docs"},
		{"alice", "Deploy to production"},
		{"bob", "Write tests"},
	} {
		db.Exec("INSERT INTO todos (title, done, owner) VALUES (?, 0, ?)", t.title, t.owner)
	}
	return db
}

func listTodos(db *sql.DB, owner string) ([]Todo, error) {
	rows, err := db.Query("SELECT id, title, done, owner FROM todos WHERE owner = ? ORDER BY id", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.Owner); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

func addTodo(db *sql.DB, owner, title string) error {
	_, err := db.Exec("INSERT INTO todos (title, done, owner) VALUES (?, 0, ?)", title, owner)
	return err
}

func toggleTodo(db *sql.DB, owner, id string) error {
	_, err := db.Exec("UPDATE todos SET done = 1 - done WHERE id = ? AND owner = ?", id, owner)
	return err
}

func deleteTodo(db *sql.DB, owner, id string) error {
	_, err := db.Exec("DELETE FROM todos WHERE id = ? AND owner = ?", id, owner)
	return err
}
