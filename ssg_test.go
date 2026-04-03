package dark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSGBasicGeneration(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	outputDir := t.TempDir()

	err = app.GenerateStaticSite(outputDir, []StaticRoute{
		{
			Path:      "/",
			Component: "simple.tsx",
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"name": "Static Hello"}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Static Hello") {
		t.Error("expected 'Static Hello' in generated HTML")
	}
}

func TestSSGMultipleRoutes(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	outputDir := t.TempDir()

	err = app.GenerateStaticSite(outputDir, []StaticRoute{
		{
			Path:      "/",
			Component: "simple.tsx",
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"name": "Home"}, nil
			},
		},
		{
			Path:      "/about",
			Component: "simple.tsx",
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"name": "About"}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "Home") {
		t.Error("expected 'Home' in index.html")
	}

	about, err := os.ReadFile(filepath.Join(outputDir, "about", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(about), "About") {
		t.Error("expected 'About' in about/index.html")
	}
}

func TestSSGStaticPaths(t *testing.T) {
	app, err := New(WithTemplateDir("_testdata"), WithPoolSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	outputDir := t.TempDir()

	err = app.GenerateStaticSite(outputDir, []StaticRoute{
		{
			Component: "simple.tsx",
			StaticPaths: func() []string {
				return []string{"/posts/1", "/posts/2", "/posts/3"}
			},
			Loader: func(ctx Context) (any, error) {
				return map[string]any{"name": "Post " + ctx.Request().URL.Path}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"1", "2", "3"} {
		content, err := os.ReadFile(filepath.Join(outputDir, "posts", id, "index.html"))
		if err != nil {
			t.Fatalf("posts/%s/index.html: %v", id, err)
		}
		if !strings.Contains(string(content), "Post /posts/"+id) {
			t.Errorf("expected post content in posts/%s/index.html", id)
		}
	}
}
