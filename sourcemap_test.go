package dark

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeVLQ(t *testing.T) {
	// "A" = 0, "C" = 1, "D" = -1, "E" = 2
	tests := []struct {
		input    string
		expected []int
	}{
		{"AAAA", []int{0, 0, 0, 0}},
		{"AACA", []int{0, 0, 1, 0}},
		{"AADA", []int{0, 0, -1, 0}},
		{"AEAA", []int{0, 2, 0, 0}},
	}
	for _, tt := range tests {
		got := decodeVLQ(tt.input)
		if len(got) != len(tt.expected) {
			t.Fatalf("decodeVLQ(%q): got %v, want %v", tt.input, got, tt.expected)
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Fatalf("decodeVLQ(%q)[%d]: got %d, want %d", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestDecodeMappings(t *testing.T) {
	// Simple mapping: 3 generated lines, each with one segment.
	// Line 1: col 0 → source 0, line 0, col 0
	// Line 2: col 0 → source 0, line 1, col 0
	// Line 3: col 0 → source 0, line 2, col 0
	mappings := "AAAA;AACA;AACA"
	lines := decodeMappings(mappings)

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) != 1 {
			t.Fatalf("line %d: expected 1 segment, got %d", i, len(line))
		}
		if line[0].sourceLine != i {
			t.Fatalf("line %d: expected sourceLine=%d, got %d", i, i, line[0].sourceLine)
		}
	}
}

func makeInlineSourceMap(sources []string, mappings string) string {
	sm := map[string]any{
		"version":  3,
		"sources":  sources,
		"mappings": mappings,
	}
	data, _ := json.Marshal(sm)
	encoded := base64.StdEncoding.EncodeToString(data)
	return "var x = 1;\n//# sourceMappingURL=data:application/json;base64," + encoded
}

func TestParseInlineSourceMap(t *testing.T) {
	js := makeInlineSourceMap([]string{"test.tsx"}, "AAAA;AACA;AACA")
	sm, err := parseInlineSourceMap(js)
	if err != nil {
		t.Fatalf("parseInlineSourceMap: %v", err)
	}
	if len(sm.Sources) != 1 || sm.Sources[0] != "test.tsx" {
		t.Fatalf("expected sources=[test.tsx], got %v", sm.Sources)
	}
	if len(sm.mappingLines) != 3 {
		t.Fatalf("expected 3 mapping lines, got %d", len(sm.mappingLines))
	}
}

func TestSourceMapLookup(t *testing.T) {
	js := makeInlineSourceMap([]string{"hello.tsx"}, "AAAA;AACA;AACA;AACA;AACA")
	sm, err := parseInlineSourceMap(js)
	if err != nil {
		t.Fatalf("parseInlineSourceMap: %v", err)
	}

	// Generated line 1 → source line 1 (0-based: 0 → 1-based: 1)
	pos, ok := sm.lookup(1, 0)
	if !ok {
		t.Fatal("lookup(1,0) failed")
	}
	if pos.source != "hello.tsx" || pos.line != 1 {
		t.Fatalf("expected hello.tsx:1, got %s:%d", pos.source, pos.line)
	}

	// Generated line 3 → source line 3
	pos, ok = sm.lookup(3, 0)
	if !ok {
		t.Fatal("lookup(3,0) failed")
	}
	if pos.line != 3 {
		t.Fatalf("expected line 3, got %d", pos.line)
	}

	// Out of range
	_, ok = sm.lookup(100, 0)
	if ok {
		t.Fatal("expected lookup(100,0) to fail")
	}
}

func TestMapErrorWithSourceMap(t *testing.T) {
	js := makeInlineSourceMap([]string{"component.tsx"}, "AAAA;AACA;AACA;AACA;AACA")
	sm, _ := parseInlineSourceMap(js)

	errMsg := "ramune: Eval: TypeError: Cannot read property 'map' of undefined\n    at eval:3:10"
	mapped := mapErrorWithSourceMap(errMsg, sm)

	if !strings.Contains(mapped, "component.tsx:3:") {
		t.Fatalf("expected mapped source location, got: %s", mapped)
	}
	if strings.Contains(mapped, "eval:3:10") {
		t.Fatalf("expected original location to be replaced, got: %s", mapped)
	}
	// Error message should be preserved.
	if !strings.Contains(mapped, "Cannot read property 'map' of undefined") {
		t.Fatalf("expected error message preserved, got: %s", mapped)
	}
}

func TestMapErrorWithNilSourceMap(t *testing.T) {
	errMsg := "some error at eval:5:0"
	result := mapErrorWithSourceMap(errMsg, nil)
	if result != errMsg {
		t.Fatalf("expected unchanged message with nil source map, got: %s", result)
	}
}
