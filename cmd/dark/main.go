package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: dark new <project-name> [--ui react]")
			os.Exit(1)
		}
		cmdNew(os.Args[2], parseUILib())
	case "generate", "gen", "g":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: dark generate <route|island> <name>")
			os.Exit(1)
		}
		cmdGenerate(os.Args[2], os.Args[3])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`dark - CLI for the dark web framework

Usage:
  dark new <project-name> [--ui react]   Create a new dark project
  dark generate route <name>             Generate a page route
  dark generate island <name>            Generate an island component
  dark help                              Show this help

Options:
  --ui preact|react    UI library (default: preact)

Aliases:
  dark gen route <name>            Short for generate
  dark g route <name>              Shortest form
`)
}

type projectData struct {
	Name          string
	ModulePath    string
	ComponentName string
	UILib         string // "preact" or "react"
}

func cmdNew(name, uiLib string) {
	data := projectData{
		Name:       name,
		ModulePath: name,
		UILib:      uiLib,
	}

	dirs := []string{
		name,
		filepath.Join(name, "views", "layouts"),
		filepath.Join(name, "views", "pages"),
		filepath.Join(name, "public"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			fatal("create directory %s: %v", d, err)
		}
	}

	writeTemplate("templates/main.go.tmpl", filepath.Join(name, "main.go"), data)
	writeTemplate("templates/layout.tsx.tmpl", filepath.Join(name, "views", "layouts", "default.tsx"), data)
	writeTemplate("templates/index.tsx.tmpl", filepath.Join(name, "views", "pages", "index.tsx"), data)
	writeTemplate("templates/style.css.tmpl", filepath.Join(name, "public", "style.css"), data)
	writeTemplate("templates/Makefile.tmpl", filepath.Join(name, "Makefile"), data)

	// Write go.mod directly (not a template).
	goMod := fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire github.com/i2y/dark v0.0.0\n", name)
	if err := os.WriteFile(filepath.Join(name, "go.mod"), []byte(goMod), 0o644); err != nil {
		fatal("write go.mod: %v", err)
	}

	fmt.Printf("Created project: %s\n\n", name)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  make dev\n")
}

func cmdGenerate(kind, name string) {
	data := projectData{
		Name:          name,
		ComponentName: toPascalCase(name),
		UILib:         parseUILib(),
	}

	switch kind {
	case "route", "r":
		outPath := filepath.Join("views", "pages", name+".tsx")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			fatal("create directory: %v", err)
		}
		writeTemplate("templates/route.tsx.tmpl", outPath, data)
		fmt.Printf("Created route component: %s\n", outPath)
		fmt.Printf("\nAdd to your main.go:\n")
		fmt.Printf(`  app.Get("/%s", dark.Route{`+"\n", name)
		fmt.Printf(`    Component: "pages/%s.tsx",`+"\n", name)
		fmt.Printf(`    Loader: func(ctx dark.Context) (any, error) {` + "\n")
		fmt.Printf(`      return map[string]any{}, nil` + "\n")
		fmt.Printf(`    },` + "\n")
		fmt.Printf(`  })` + "\n")

	case "island", "i":
		outPath := filepath.Join("views", "islands", name+".tsx")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			fatal("create directory: %v", err)
		}
		writeTemplate("templates/island.tsx.tmpl", outPath, data)
		fmt.Printf("Created island component: %s\n", outPath)
		fmt.Printf("\nAdd to your main.go:\n")
		fmt.Printf(`  app.Island("%s", "islands/%s.tsx")`+"\n", name, name)

	default:
		fmt.Fprintf(os.Stderr, "Unknown generator: %s (use 'route' or 'island')\n", kind)
		os.Exit(1)
	}
}

func writeTemplate(tmplPath, outPath string, data any) {
	content, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		fatal("read template %s: %v", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		fatal("parse template %s: %v", tmplPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		fatal("create %s: %v", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		fatal("execute template %s: %v", tmplPath, err)
	}
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	var b strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]))
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func parseUILib() string {
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--ui" {
			lib := os.Args[i+1]
			if lib != "preact" && lib != "react" {
				fmt.Fprintf(os.Stderr, "Unknown UI library: %s (use 'preact' or 'react')\n", lib)
				os.Exit(1)
			}
			return lib
		}
	}
	return "preact"
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dark: "+format+"\n", args...)
	os.Exit(1)
}
