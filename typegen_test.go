package dark

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

type SimpleProps struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Active bool   `json:"active"`
}

type NestedProps struct {
	User  UserInfo `json:"user"`
	Score float64  `json:"score"`
	Tags  []string `json:"tags"`
}

type UserInfo struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type OptionalFieldProps struct {
	Title   string `json:"title"`
	Desc    string `json:"description,omitempty"`
	Hidden  string `json:"-"`
	private string
}

type PointerProps struct {
	Name   string    `json:"name"`
	Parent *UserInfo `json:"parent"`
}

type MapProps struct {
	Data   map[string]any `json:"data"`
	Counts map[string]int `json:"counts"`
}

type TimeProps struct {
	CreatedAt time.Time  `json:"createdAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type SliceOfStructProps struct {
	Posts []PostItem `json:"posts"`
}

type PostItem struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type BaseModel struct {
	ID int `json:"id"`
}

type EmbeddedProps struct {
	BaseModel
	Name string `json:"name"`
}

type ByteSliceProps struct {
	Data []byte `json:"data"`
	Name string `json:"name"`
}

func TestTypeGenEmbeddedStruct(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(EmbeddedProps{}))
	out := gen.generate()

	// Embedded fields should be promoted (flat), matching JSON behavior.
	if !strings.Contains(out, "id: number;") {
		t.Fatalf("expected promoted id field, got: %s", out)
	}
	if !strings.Contains(out, "name: string;") {
		t.Fatalf("expected name field, got: %s", out)
	}
	// Should NOT have a nested BaseModel field.
	if strings.Contains(out, "BaseModel") {
		t.Fatalf("expected embedded struct to be flattened, got: %s", out)
	}
}

func TestTypeGenByteSlice(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(ByteSliceProps{}))
	out := gen.generate()

	// []byte → string (JSON marshals as base64)
	if !strings.Contains(out, "data: string;") {
		t.Fatalf("expected []byte to map to string, got: %s", out)
	}
}

func TestTypeGenSimple(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(SimpleProps{}))
	out := gen.generate()

	if !strings.Contains(out, "export interface SimpleProps") {
		t.Fatalf("expected SimpleProps interface, got: %s", out)
	}
	if !strings.Contains(out, "name: string;") {
		t.Fatalf("expected name: string, got: %s", out)
	}
	if !strings.Contains(out, "count: number;") {
		t.Fatalf("expected count: number, got: %s", out)
	}
	if !strings.Contains(out, "active: boolean;") {
		t.Fatalf("expected active: boolean, got: %s", out)
	}
}

func TestTypeGenNested(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(NestedProps{}))
	out := gen.generate()

	if !strings.Contains(out, "export interface NestedProps") {
		t.Fatalf("expected NestedProps, got: %s", out)
	}
	if !strings.Contains(out, "user: UserInfo;") {
		t.Fatalf("expected user: UserInfo, got: %s", out)
	}
	if !strings.Contains(out, "export interface UserInfo") {
		t.Fatalf("expected UserInfo auto-generated, got: %s", out)
	}
	if !strings.Contains(out, "tags: string[];") {
		t.Fatalf("expected tags: string[], got: %s", out)
	}
}

func TestTypeGenOmitemptyAndHidden(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(OptionalFieldProps{}))
	out := gen.generate()

	if !strings.Contains(out, "title: string;") {
		t.Fatalf("expected title: string, got: %s", out)
	}
	if !strings.Contains(out, "description?: string;") {
		t.Fatalf("expected description?: string, got: %s", out)
	}
	if strings.Contains(out, "Hidden") || strings.Contains(out, "hidden") {
		t.Fatalf("expected json:\"-\" field to be excluded, got: %s", out)
	}
	if strings.Contains(out, "private") {
		t.Fatalf("expected unexported field to be excluded, got: %s", out)
	}
}

func TestTypeGenPointer(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(PointerProps{}))
	out := gen.generate()

	if !strings.Contains(out, "parent: UserInfo | null;") {
		t.Fatalf("expected parent: UserInfo | null, got: %s", out)
	}
	if !strings.Contains(out, "export interface UserInfo") {
		t.Fatalf("expected UserInfo generated from pointer, got: %s", out)
	}
}

func TestTypeGenMap(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(MapProps{}))
	out := gen.generate()

	if !strings.Contains(out, "data: Record<string, any>;") {
		t.Fatalf("expected data: Record<string, any>, got: %s", out)
	}
	if !strings.Contains(out, "counts: Record<string, number>;") {
		t.Fatalf("expected counts: Record<string, number>, got: %s", out)
	}
}

func TestTypeGenTime(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(TimeProps{}))
	out := gen.generate()

	if !strings.Contains(out, "createdAt: string;") {
		t.Fatalf("expected createdAt: string (time → string), got: %s", out)
	}
	if !strings.Contains(out, "deletedAt?: string | null;") {
		t.Fatalf("expected deletedAt?: string | null, got: %s", out)
	}
}

func TestTypeGenSliceOfStructs(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(SliceOfStructProps{}))
	out := gen.generate()

	if !strings.Contains(out, "posts: PostItem[];") {
		t.Fatalf("expected posts: PostItem[], got: %s", out)
	}
	if !strings.Contains(out, "export interface PostItem") {
		t.Fatalf("expected PostItem generated, got: %s", out)
	}
}

func TestTypeGenDarkBaseProps(t *testing.T) {
	gen := newTSGenerator()
	gen.addType(reflect.TypeOf(SimpleProps{}))
	out := gen.generate()

	if !strings.Contains(out, "export interface DarkBaseProps") {
		t.Fatalf("expected DarkBaseProps, got: %s", out)
	}
	if !strings.Contains(out, "_head?: HeadData") {
		t.Fatalf("expected _head?: HeadData in DarkBaseProps, got: %s", out)
	}
	if !strings.Contains(out, "_errors?: FieldError[]") {
		t.Fatalf("expected _errors?: FieldError[] in DarkBaseProps, got: %s", out)
	}
	// HeadData and FieldError should be generated as interfaces.
	if !strings.Contains(out, "export interface HeadData") {
		t.Fatalf("expected HeadData interface, got: %s", out)
	}
	if !strings.Contains(out, "export interface FieldError") {
		t.Fatalf("expected FieldError interface, got: %s", out)
	}
}

func TestTypeGenDeterministicOutput(t *testing.T) {
	gen1 := newTSGenerator()
	gen1.addType(reflect.TypeOf(NestedProps{}))
	gen1.addType(reflect.TypeOf(SimpleProps{}))
	out1 := gen1.generate()

	gen2 := newTSGenerator()
	gen2.addType(reflect.TypeOf(SimpleProps{}))
	gen2.addType(reflect.TypeOf(NestedProps{}))
	out2 := gen2.generate()

	if out1 != out2 {
		t.Fatalf("output should be deterministic regardless of add order:\n--- out1:\n%s\n--- out2:\n%s", out1, out2)
	}
}
