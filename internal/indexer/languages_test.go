package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestLanguageRegistry proves the registry resolves Go by both ID and
// extension and hands back a working parser + extractor — the seam every
// subsequent wave (discovery, extract, resolve, per-language extractors,
// dispatch, routing) depends on (D-01).
func TestLanguageRegistry(t *testing.T) {
	spec, ok := lookupLanguageByID("go")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"go\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".go" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected go spec Extensions to contain \".go\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".go")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".go\") to return ok=true")
	}
	if byExt.ID != "go" {
		t.Fatalf("expected .go to resolve to the go spec, got ID=%q", byExt.ID)
	}

	if spec.NewParser == nil {
		t.Fatal("expected a non-nil NewParser func")
	}
	p, err := spec.NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	if p == nil {
		t.Fatal("expected NewParser to return a non-nil parser")
	}
	defer p.Close()

	if spec.Extract == nil {
		t.Fatal("expected a non-nil Extract func")
	}
}

// TestLanguageRegistry_Java proves the registry resolves Java by both ID
// and extension, hands back a working parser + extractor, and that Java's
// ModuleKey degrades to path-based identity absent a resolvable descriptor
// (D-03) — mirroring TestLanguageRegistry's shape for Go (LANG-02).
func TestLanguageRegistry_Java(t *testing.T) {
	spec, ok := lookupLanguageByID("java")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"java\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".java" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected java spec Extensions to contain \".java\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".java")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".java\") to return ok=true")
	}
	if byExt.ID != "java" {
		t.Fatalf("expected .java to resolve to the java spec, got ID=%q", byExt.ID)
	}

	if spec.NewParser == nil {
		t.Fatal("expected a non-nil NewParser func")
	}
	p, err := spec.NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	if p == nil {
		t.Fatal("expected NewParser to return a non-nil parser")
	}
	defer p.Close()

	if spec.Extract == nil {
		t.Fatal("expected a non-nil Extract func")
	}

	if spec.ModuleKey == nil {
		t.Fatal("expected a non-nil ModuleKey func")
	}
	if got, want := spec.ModuleKey(nil, "sub/Widget.java"), "sub/Widget.java"; got != want {
		t.Errorf("ModuleKey(nil descriptor) = %q, want %q (D-03 path-identity fallback)", got, want)
	}
}

// TestLanguageRegistry_CSharp proves the registry resolves C# by both ID
// and extension, hands back a working parser + extractor, and that C#'s
// ModuleKey degrades to path-based identity absent a resolvable descriptor
// (D-03) — mirroring TestLanguageRegistry_Java's shape (LANG-03).
func TestLanguageRegistry_CSharp(t *testing.T) {
	spec, ok := lookupLanguageByID("csharp")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"csharp\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".cs" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected csharp spec Extensions to contain \".cs\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".cs")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".cs\") to return ok=true")
	}
	if byExt.ID != "csharp" {
		t.Fatalf("expected .cs to resolve to the csharp spec, got ID=%q", byExt.ID)
	}

	if spec.NewParser == nil {
		t.Fatal("expected a non-nil NewParser func")
	}
	p, err := spec.NewParser()
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	if p == nil {
		t.Fatal("expected NewParser to return a non-nil parser")
	}
	defer p.Close()

	if spec.Extract == nil {
		t.Fatal("expected a non-nil Extract func")
	}

	if spec.ModuleKey == nil {
		t.Fatal("expected a non-nil ModuleKey func")
	}
	if got, want := spec.ModuleKey(nil, "sub/Widget.cs"), "sub/Widget.cs"; got != want {
		t.Errorf("ModuleKey(nil descriptor) = %q, want %q (D-03 path-identity fallback)", got, want)
	}
}

// TestKindRouteAdditive proves the Phase 5 KindRoute constant was added
// additively — every pre-existing Kind* constant must remain unchanged.
func TestKindRouteAdditive(t *testing.T) {
	if goextract.KindRoute != "route" {
		t.Fatalf("expected goextract.KindRoute == \"route\", got %q", goextract.KindRoute)
	}

	existing := map[string]string{
		goextract.KindFile:      "file",
		goextract.KindFunction:  "function",
		goextract.KindMethod:    "method",
		goextract.KindStruct:    "struct",
		goextract.KindInterface: "interface",
		goextract.KindTypeAlias: "type_alias",
		goextract.KindConstant:  "constant",
		goextract.KindVariable:  "variable",
	}
	for got, want := range existing {
		if got != want {
			t.Fatalf("pre-existing Kind* constant changed: got %q, want %q", got, want)
		}
	}
}
