package dark

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

// GenerateTypes generates TypeScript type definitions from Props fields on registered routes.
// Output is written to <templateDir>/_generated/props.d.ts.
func (app *App) GenerateTypes() error {
	types := collectRouteProps(app.routes)
	if len(types) == 0 {
		return nil
	}

	gen := newTSGenerator()
	for _, t := range types {
		gen.addType(t)
	}

	outDir := filepath.Join(app.config.templateDir, "_generated")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("dark: failed to create type output dir: %w", err)
	}

	content := gen.generate()
	outFile := filepath.Join(outDir, "props.d.ts")

	// Skip write if content is unchanged to avoid triggering file watchers.
	if existing, err := os.ReadFile(outFile); err == nil && string(existing) == content {
		return nil
	}

	if err := os.WriteFile(outFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("dark: failed to write types: %w", err)
	}

	return nil
}

func collectRouteProps(routes []registeredRoute) []reflect.Type {
	seen := map[reflect.Type]bool{}
	var types []reflect.Type
	for _, r := range routes {
		if r.page == nil || r.page.Props == nil {
			continue
		}
		t := reflect.TypeOf(r.page.Props)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct && !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	return types
}

// tsField holds resolved info for a single struct field.
type tsField struct {
	name     string
	tsType   string
	optional bool
}

type tsGenerator struct {
	interfaces map[string]string // name → interface body
	visited    map[reflect.Type]bool
}

func newTSGenerator() *tsGenerator {
	return &tsGenerator{
		interfaces: make(map[string]string),
		visited:    make(map[reflect.Type]bool),
	}
}

func (g *tsGenerator) addType(t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || g.visited[t] {
		return
	}
	g.visited[t] = true
	g.generateInterface(t)
}

func (g *tsGenerator) resolveFields(t reflect.Type) []tsField {
	var fields []tsField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		// Embedded (anonymous) structs: promote their fields to match JSON behavior.
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			fields = append(fields, g.resolveFields(f.Type)...)
			continue
		}
		if f.Anonymous && f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
			fields = append(fields, g.resolveFields(f.Type.Elem())...)
			continue
		}

		jsonName := jsonFieldName(f)
		if jsonName == "-" {
			continue
		}

		fields = append(fields, tsField{
			name:     jsonName,
			tsType:   g.goTypeToTS(f.Type),
			optional: isOmitempty(f),
		})
	}
	return fields
}

func (g *tsGenerator) generateInterface(t reflect.Type) {
	name := t.Name()
	if name == "" {
		return
	}

	fields := g.resolveFields(t)
	var lines []string
	for _, f := range fields {
		if f.optional {
			lines = append(lines, fmt.Sprintf("    %s?: %s;", f.name, f.tsType))
		} else {
			lines = append(lines, fmt.Sprintf("    %s: %s;", f.name, f.tsType))
		}
	}

	body := "export interface " + name + " {\n" + strings.Join(lines, "\n") + "\n}"
	g.interfaces[name] = body
}

func (g *tsGenerator) goTypeToTS(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		inner := g.goTypeToTS(t.Elem())
		return inner + " | null"
	}

	if t == timeType {
		return "string" // JSON marshals as ISO 8601 string
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		// []byte → string (JSON marshals as base64)
		if t.Elem().Kind() == reflect.Uint8 {
			return "string"
		}
		elem := g.goTypeToTS(t.Elem())
		if strings.Contains(elem, "|") {
			return "(" + elem + ")[]"
		}
		return elem + "[]"
	case reflect.Map:
		// JSON object keys are always strings regardless of Go key type.
		val := g.goTypeToTS(t.Elem())
		return "Record<string, " + val + ">"
	case reflect.Struct:
		name := t.Name()
		if name == "" {
			return g.inlineStruct(t)
		}
		// Recursively generate if not yet visited.
		if !g.visited[t] {
			g.visited[t] = true
			g.generateInterface(t)
		}
		return name
	case reflect.Interface:
		return "any"
	default:
		return "any"
	}
}

func (g *tsGenerator) inlineStruct(t reflect.Type) string {
	fields := g.resolveFields(t)
	var parts []string
	for _, f := range fields {
		if f.optional {
			parts = append(parts, f.name+"?: "+f.tsType)
		} else {
			parts = append(parts, f.name+": "+f.tsType)
		}
	}
	return "{ " + strings.Join(parts, "; ") + " }"
}

func (g *tsGenerator) generate() string {
	var b strings.Builder
	b.WriteString("// Auto-generated by dark. Do not edit.\n\n")

	// Generate DarkBaseProps from actual Go types.
	g.addType(reflect.TypeOf(HeadData{}))
	g.addType(reflect.TypeOf(FieldError{}))

	b.WriteString("export interface DarkBaseProps {\n")
	b.WriteString("    _head?: HeadData;\n")
	b.WriteString("    _errors?: FieldError[];\n")
	b.WriteString("    _formData?: Record<string, any>;\n")
	b.WriteString("}\n\n")

	// Sort interface names for deterministic output.
	names := make([]string, 0, len(g.interfaces))
	for name := range g.interfaces {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		b.WriteString(g.interfaces[name])
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name
	}
	return name
}

func isOmitempty(f reflect.StructField) bool {
	tag := f.Tag.Get("json")
	_, opts, _ := strings.Cut(tag, ",")
	for opts != "" {
		var opt string
		opt, opts, _ = strings.Cut(opts, ",")
		if opt == "omitempty" {
			return true
		}
	}
	return false
}
