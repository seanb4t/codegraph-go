package indexer

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/gencorpus"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/spill"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// spillCompare is the equality of a FileResult before and after the spill.
// Gob sends no zero value, so an empty slice or map comes back as nil, and
// the resolver treats the two the same. The decoded error is a new value,
// and go-cmp refuses to compare two error values on its own, so the
// comparer takes nil-ness and the Error() text.
var spillCompare = cmp.Options{
	protocmp.Transform(),
	cmpopts.EquateEmpty(),
	cmp.Comparer(func(a, b error) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}
		return a.Error() == b.Error()
	}),
}

// spillRoundTrip encodes and decodes r, and fails the test on a difference.
func spillRoundTrip(t interface {
	Helper()
	Fatalf(string, ...any)
}, r goextract.FileResult) goextract.FileResult {
	t.Helper()
	data, err := spill.Encode(&r)
	if err != nil {
		t.Fatalf("Encode %s: %v", r.RelPath, err)
	}
	got, err := spill.Decode(data)
	if err != nil {
		t.Fatalf("Decode %s: %v", r.RelPath, err)
	}
	if diff := cmp.Diff(r, got, spillCompare); diff != "" {
		t.Fatalf("%s changed in the spill (-before +after):\n%s", r.RelPath, diff)
	}
	return got
}

// markSetFields records "<Type>.<Field>" in set for each field of a
// goextract struct that holds a non-zero value somewhere in v.
func markSetFields(v reflect.Value, set map[string]bool) {
	switch v.Kind() {
	case reflect.Struct:
		if v.Type().PkgPath() != reflect.TypeFor[goextract.FileResult]().PkgPath() {
			return
		}
		for i := range v.NumField() {
			if f := v.Field(i); !f.IsZero() {
				set[v.Type().Name()+"."+v.Type().Field(i).Name] = true
				markSetFields(f, set)
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			markSetFields(v.Index(i), set)
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			markSetFields(it.Value(), set)
		}
	}
}

// goextractFields lists "<Type>.<Field>" for FileResult and each goextract
// struct that it reaches.
func goextractFields(typ reflect.Type, into map[string]bool) {
	switch typ.Kind() {
	case reflect.Slice, reflect.Map:
		goextractFields(typ.Elem(), into)
	case reflect.Struct:
		if typ.PkgPath() != reflect.TypeFor[goextract.FileResult]().PkgPath() {
			return
		}
		for i := range typ.NumField() {
			name := typ.Name() + "." + typ.Field(i).Name
			if !into[name] {
				into[name] = true
				goextractFields(typ.Field(i).Type, into)
			}
		}
	}
}

// TestSpillRoundTrip sends the extraction result of every fixture file and
// of 2,000 synthetic files through the spill codec. A field that the codec
// drops shows as a difference.
func TestSpillRoundTrip(t *testing.T) {
	trees := fixtureTrees(t)
	synthetic := t.TempDir()
	if _, err := gencorpus.Generate(gencorpus.Options{Seed: 42, FileCount: 2000, OutDir: synthetic}); err != nil {
		t.Fatalf("gencorpus.Generate: %v", err)
	}
	trees = append(trees, fixtureTree{"synthetic", synthetic})
	// No fixture has a call with a spread argument or a labelled argument,
	// so one small file adds both.
	calls := t.TempDir()
	writeFile(t, filepath.Join(calls, "calls.rb"), "def greet(name, punctuation:)\nend\n\ndef run(names)\n  greet(*names, punctuation: \"!\")\nend\n")
	trees = append(trees, fixtureTree{"calls", calls})
	// No fixture binds a variable to a function in its initializer or has
	// a syntax error, so one small Go tree adds both.
	bindings := t.TempDir()
	writeFile(t, filepath.Join(bindings, "go.mod"), "module example.com/b\n\ngo 1.22\n")
	writeFile(t, filepath.Join(bindings, "b.go"), "package b\n\nfunc f() {}\n\nvar g = f\n\nfunc broken() {\n\tg = (\n}\n")
	trees = append(trees, fixtureTree{"bindings", bindings})
	// No fixture reads a Java name that the enclosing class may inherit, so
	// one small Java tree adds one.
	inherited := t.TempDir()
	writeFile(t, filepath.Join(inherited, "Derived.java"), "package p;\n\npublic class Derived extends Base {\n    int read() {\n        return count;\n    }\n}\n")
	trees = append(trees, fixtureTree{"inherited", inherited})

	set := make(map[string]bool)
	results := 0
	for _, tr := range trees {
		files, _, err := discover.Discover(tr.root)
		if err != nil {
			t.Fatalf("Discover %s: %v", tr.root, err)
		}
		extracted, err := Extract(files, 0)
		if err != nil {
			t.Fatalf("Extract %s: %v", tr.root, err)
		}
		for _, r := range extracted {
			spillRoundTrip(t, r)
			markSetFields(reflect.ValueOf(r), set)
			results++
		}
	}

	// Positive assertion: the corpus sets every field, so the comparison
	// above saw every field carry a value. Err is the one exception: no
	// fixture fails to extract, and TestSpillRoundTripProperty covers it.
	all := make(map[string]bool)
	goextractFields(reflect.TypeFor[goextract.FileResult](), all)
	var unset []string
	for name := range all {
		if !set[name] && name != "FileResult.Err" {
			unset = append(unset, name)
		}
	}
	sort.Strings(unset)
	if len(unset) > 0 {
		t.Errorf("no result of the corpus sets these fields, so the round trip does not prove them: %s", strings.Join(unset, ", "))
	}
	if results < 2000 {
		t.Errorf("round-tripped %d results, want at least 2000", results)
	}
}

// drawSlice draws a slice of up to 3 elements. One draw in four gives an
// empty, non-nil slice, which gob sends as nothing.
func drawSlice[T any](t *rapid.T, label string, elem func(i int) T) []T {
	n := rapid.IntRange(-1, 3).Draw(t, label)
	if n < 0 {
		return []T{}
	}
	var out []T
	for i := range n {
		out = append(out, elem(i))
	}
	return out
}

// TestSpillRoundTripProperty round-trips drawn results: empty slices, empty
// maps, empty strings, and an error beside nodes. An error that wraps
// goextract.ErrInternal must still match it after the spill.
func TestSpillRoundTripProperty(t *testing.T) {
	str := rapid.SampledFrom([]string{"", "a", "pkg/file.go", "Ünïcode", "x y"})
	i32 := rapid.Int32Range(0, 5)
	rapid.Check(t, func(t *rapid.T) {
		s := func(label string) string { return str.Draw(t, label) }
		n := func(label string) int32 { return i32.Draw(t, label) }
		r := goextract.FileResult{
			ImportPath:        s("importPath"),
			RelPath:           s("relPath"),
			Language:          s("language"),
			ModuleKeyDeclared: rapid.Bool().Draw(t, "declared"),
			ContentHash:       s("hash"),
			MtimeUnixNs:       rapid.Int64().Draw(t, "mtime"),
			SizeBytes:         rapid.Int64().Draw(t, "size"),
		}
		r.Nodes = drawSlice(t, "nodes", func(int) goextract.ExtractedNode {
			return goextract.ExtractedNode{
				Node: &schema.Node{
					Id: s("nodeID"), Kind: s("nodeKind"), Name: s("nodeName"),
					StartLine: n("startLine"), IsExported: rapid.Bool().Draw(t, "exported"),
				},
				Scope: goextract.Range{StartLine: n("scopeStart"), EndLine: n("scopeEnd"), EndCol: n("scopeEndCol")},
			}
		})
		r.IntraEdges = drawSlice(t, "edges", func(int) goextract.IntraEdge {
			e := &schema.Edge{Source: s("edgeSource"), Kind: s("edgeKind"), Target: s("edgeTarget"), Line: n("edgeLine")}
			if rapid.Bool().Draw(t, "edgeMetadata") {
				e.Metadata = map[string]string{"k1": s("v1"), "k2": s("v2")}
			}
			return goextract.IntraEdge{Edge: e}
		})
		r.Unresolved = drawSlice(t, "unresolved", func(int) goextract.UnresolvedRef {
			return goextract.UnresolvedRef{
				FromID: s("fromID"), Name: s("refName"), PkgAlias: s("alias"), Kind: s("refKind"),
				Line: n("line"), Col: n("col"), EndLine: n("endLine"), EndCol: n("endCol"),
				ReceiverType: s("receiver"), ArgCount: n("argCount"), HasSpread: rapid.Bool().Draw(t, "spread"),
				ArgTypes:  drawSlice(t, "argTypes", func(int) string { return s("argType") }),
				ArgLabels: drawSlice(t, "argLabels", func(int) string { return s("argLabel") }),
			}
		})
		switch rapid.IntRange(0, 2).Draw(t, "maps") {
		case 1:
			r.Imports = map[string]string{}
			r.InterfaceMethods = map[string][]goextract.MethodSpec{}
			r.MethodArity = map[string]int32{}
		case 2:
			r.Imports = map[string]string{s("importAlias"): s("importPath2")}
			r.InterfaceMethods = map[string][]goextract.MethodSpec{
				s("iface"): drawSlice(t, "specs", func(int) goextract.MethodSpec {
					return goextract.MethodSpec{Name: s("specName"), Arity: n("specArity")}
				}),
			}
			r.MethodArity = map[string]int32{s("method"): n("arity")}
		}
		internal := false
		switch rapid.IntRange(0, 2).Draw(t, "err") {
		case 1:
			r.Err = errors.New(s("errText"))
		case 2:
			internal = true
			r.Err = fmt.Errorf("%w: %s", goextract.ErrInternal, s("errText"))
		}

		got := spillRoundTrip(t, r)
		if errors.Is(got.Err, goextract.ErrInternal) != internal {
			t.Fatalf("errors.Is(decoded.Err, ErrInternal) = %v, want %v", !internal, internal)
		}
	})
}
