package tsextract

import (
	"errors"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// newTestParser returns a real CGo parser for the requested grammar
// ("ts"/"tsx"/"js"), closed automatically at the end of the test — mirrors
// csharpextract_test.go's newTestParser, one Parser per test.
func newTestParser(t *testing.T, grammar string) parser.Parser {
	t.Helper()
	var (
		p   *cgo.CGoParser
		err error
	)
	switch grammar {
	case "ts":
		p, err = cgo.NewTypeScriptParser()
	case "tsx":
		p, err = cgo.NewTSXParser()
	case "js":
		p, err = cgo.NewJavaScriptParser()
	default:
		t.Fatalf("newTestParser: unknown grammar %q", grammar)
	}
	if err != nil {
		t.Fatalf("cgo.New%sParser: %v", grammar, err)
	}
	t.Cleanup(func() { p.Close() })
	return p
}

func findNode(result goextract.FileResult, kind, name string) *goextract.ExtractedNode {
	for i := range result.Nodes {
		if result.Nodes[i].Node.Kind == kind && result.Nodes[i].Node.Name == name {
			return &result.Nodes[i]
		}
	}
	return nil
}

func hasNodeNamed(result goextract.FileResult, name string) bool {
	for _, n := range result.Nodes {
		if n.Node.Name == name {
			return true
		}
	}
	return false
}

func hasIntraEdge(result goextract.FileResult, source, target, kind string) bool {
	for _, e := range result.IntraEdges {
		if e.Edge.Source == source && e.Edge.Target == target && e.Edge.Kind == kind {
			return true
		}
	}
	return false
}

func hasUnresolved(result goextract.FileResult, kind, name, pkgAlias string) bool {
	for _, u := range result.Unresolved {
		if u.Kind == kind && u.Name == name && u.PkgAlias == pkgAlias {
			return true
		}
	}
	return false
}

// TestExtract_NodeKinds is table-driven across all three grammars,
// mirroring csharpextract_test.go's/javaextract_test.go's TestExtract_
// NodeKinds shape: one case per tree-sitter node type this SHARED
// extractor must map onto the codegraph vocabulary (LANG-05, D-06).
func TestExtract_NodeKinds(t *testing.T) {
	tests := []struct {
		name              string
		grammar           string
		src               string
		wantKind          string
		wantName          string
		wantQualifiedName string
		wantExported      bool
	}{
		{
			name:              "exported TS class",
			grammar:           "ts",
			src:               "export class Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "non-exported TS class",
			grammar:           "ts",
			src:               "class Helper {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Helper",
			wantQualifiedName: "Helper",
			wantExported:      false,
		},
		{
			name:              "exported TS interface",
			grammar:           "ts",
			src:               "export interface Reader {\n  read(): void;\n}\n",
			wantKind:          goextract.KindInterface,
			wantName:          "Reader",
			wantQualifiedName: "Reader",
			wantExported:      true,
		},
		{
			name:              "exported TS function",
			grammar:           "ts",
			src:               "export function add(a: number, b: number): number {\n  return a + b;\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "add",
			wantQualifiedName: "add",
			wantExported:      true,
		},
		{
			name:              "exported TS type alias",
			grammar:           "ts",
			src:               "export type ID = string;\n",
			wantKind:          goextract.KindTypeAlias,
			wantName:          "ID",
			wantQualifiedName: "ID",
			wantExported:      true,
		},
		{
			name:              "TS class method",
			grammar:           "ts",
			src:               "export class Widget {\n  render(): void {}\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "render",
			wantQualifiedName: "Widget.render",
			wantExported:      true,
		},
		{
			name:              "private TS class method",
			grammar:           "ts",
			src:               "export class Widget {\n  private helper(): void {}\n}\n",
			wantKind:          goextract.KindMethod,
			wantName:          "helper",
			wantQualifiedName: "Widget.helper",
			wantExported:      false,
		},
		{
			name:              "JS class",
			grammar:           "js",
			src:               "export class Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
		{
			name:              "JS function",
			grammar:           "js",
			src:               "export function add(a, b) {\n  return a + b;\n}\n",
			wantKind:          goextract.KindFunction,
			wantName:          "add",
			wantQualifiedName: "add",
			wantExported:      true,
		},
		{
			name:              "TSX class",
			grammar:           "tsx",
			src:               "export class Widget {}\n",
			wantKind:          goextract.KindStruct,
			wantName:          "Widget",
			wantQualifiedName: "Widget",
			wantExported:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestParser(t, tt.grammar)
			result, err := Extract(p, "m", "Widget."+tt.grammar, []byte(tt.src))
			if err != nil {
				t.Fatalf("Extract returned error: %v", err)
			}
			if result.Err != nil {
				t.Fatalf("FileResult.Err = %v, want nil", result.Err)
			}
			node := findNode(result, tt.wantKind, tt.wantName)
			if node == nil {
				t.Fatalf("no %s node named %q found in %+v", tt.wantKind, tt.wantName, result.Nodes)
			}
			if node.Node.QualifiedName != tt.wantQualifiedName {
				t.Errorf("QualifiedName = %q, want %q", node.Node.QualifiedName, tt.wantQualifiedName)
			}
			if node.Node.IsExported != tt.wantExported {
				t.Errorf("IsExported = %v, want %v", node.Node.IsExported, tt.wantExported)
			}
		})
	}
}

// TestExtract_TSXJSXFixtureParsesWithoutError proves a .tsx file containing
// JSX elements parses successfully via the tsx grammar and extracts its
// surrounding declaration (the plan's own acceptance criterion) without
// the JSX syntax breaking extraction.
func TestExtract_TSXJSXFixtureParsesWithoutError(t *testing.T) {
	src := `export function Card(props: { title: string }) {
  return (
    <div className="card">
      <h1>{props.title}</h1>
      <Icon />
    </div>
  );
}
`
	p := newTestParser(t, "tsx")
	result, err := Extract(p, "m", "Card.tsx", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.Err != nil {
		t.Fatalf("FileResult.Err = %v, want nil (JSX must not break extraction)", result.Err)
	}
	node := findNode(result, goextract.KindFunction, "Card")
	if node == nil {
		t.Fatalf("no function node named Card found in %+v", result.Nodes)
	}
	if !node.Node.IsExported {
		t.Error("Card.IsExported = false, want true")
	}
}

// TestExtract_MethodContainsEdge proves a method's contains edge sources
// from its declaring class's own node id.
func TestExtract_MethodContainsEdge(t *testing.T) {
	src := "export class Widget {\n  render(): void {}\n}\n"
	p := newTestParser(t, "ts")
	result, err := Extract(p, "m", "Widget.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	widget := findNode(result, goextract.KindStruct, "Widget")
	render := findNode(result, goextract.KindMethod, "render")
	if widget == nil || render == nil {
		t.Fatalf("expected both Widget and render nodes, got %+v", result.Nodes)
	}
	if !hasIntraEdge(result, widget.Node.Id, render.Node.Id, "contains") {
		t.Errorf("expected a contains edge Widget -> render, got %+v", result.IntraEdges)
	}
}

// TestExtract_NoFieldNodes proves a TS public_field_definition and a JS
// field_definition never emit a node — the field-skip precedent every
// priority-4 sibling extends.
func TestExtract_NoFieldNodes(t *testing.T) {
	t.Run("ts", func(t *testing.T) {
		src := "export class Widget {\n  name: string;\n  private age: number = 0;\n}\n"
		p := newTestParser(t, "ts")
		result, err := Extract(p, "m", "Widget.ts", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if hasNodeNamed(result, "name") || hasNodeNamed(result, "age") {
			t.Errorf("expected no field nodes, got %+v", result.Nodes)
		}
	})
	t.Run("js", func(t *testing.T) {
		src := "export class Widget {\n  name = 'x';\n}\n"
		p := newTestParser(t, "js")
		result, err := Extract(p, "m", "Widget.js", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if hasNodeNamed(result, "name") {
			t.Errorf("expected no field nodes, got %+v", result.Nodes)
		}
	})
}

// TestExtract_ExtendsImplements proves a TS `class X extends Y implements
// Z` and an interface's `extends` both emit a single, undistinguished
// RefKindEmbeds unresolved ref per listed supertype (RESEARCH Pattern 2),
// and that JS's own single-expression class_heritage shape (no
// `implements` keyword) works identically through the shared extractor.
func TestExtract_ExtendsImplements(t *testing.T) {
	t.Run("ts class extends+implements", func(t *testing.T) {
		src := "export class Widget extends Base implements Shape {\n  render(): void {}\n}\n"
		p := newTestParser(t, "ts")
		result, err := Extract(p, "m", "Widget.ts", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if !hasUnresolved(result, goextract.RefKindEmbeds, "Base", "") {
			t.Errorf("expected an embeds ref to Base, got %+v", result.Unresolved)
		}
		if !hasUnresolved(result, goextract.RefKindEmbeds, "Shape", "") {
			t.Errorf("expected an embeds ref to Shape, got %+v", result.Unresolved)
		}
	})
	t.Run("ts interface extends", func(t *testing.T) {
		src := "export interface Sub extends Base {}\n"
		p := newTestParser(t, "ts")
		result, err := Extract(p, "m", "Sub.ts", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if !hasUnresolved(result, goextract.RefKindEmbeds, "Base", "") {
			t.Errorf("expected an embeds ref to Base, got %+v", result.Unresolved)
		}
	})
	t.Run("js class extends (no implements keyword)", func(t *testing.T) {
		src := "class Base {}\nexport class Widget extends Base {\n  render() {}\n}\n"
		p := newTestParser(t, "js")
		result, err := Extract(p, "m", "Widget.js", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if !hasUnresolved(result, goextract.RefKindEmbeds, "Base", "") {
			t.Errorf("expected an embeds ref to Base, got %+v", result.Unresolved)
		}
	})
}

// TestExtract_Calls proves the three call-resolution disambiguation shapes:
// a PascalCase non-imported qualified call attempts a same-module
// resolution, a camelCase non-imported qualified call is forced through a
// synthetic non-matching alias (mirrors goextract's WR-02 fix), and a bare
// unqualified call attempts a same-module resolution.
func TestExtract_Calls(t *testing.T) {
	src := `class Helper {
  static assist(): void {}
}
export function run(myWidget: Widget): void {
  Helper.assist();
  myWidget.render();
  helper();
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "m", "run.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "assist", "") {
		t.Errorf("expected a same-module call attempt to assist (PascalCase Helper), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "render", "<local:myWidget>") {
		t.Errorf("expected a forced-unresolved call to render (camelCase local var), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "") {
		t.Errorf("expected a same-module call attempt to helper (bare identifier), got %+v", result.Unresolved)
	}
}

// TestExtract_Imports proves default/namespace/named import shapes all
// populate FileResult.Imports (local alias -> resolved target moduleKey)
// via relative-specifier resolution, and that a bare call/member-access
// through each imported binding shape resolves through the correct
// PkgAlias/Name pair (types.go's documented named-import resolution
// mechanism).
func TestExtract_Imports(t *testing.T) {
	src := `import { helper } from './util';
import Default from './main';
import * as NS from './ns';

export function run(): void {
  helper();
  Default();
  NS.doWork();
}
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "m", "src/app.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	wantImports := map[string]string{
		"helper":  "src/util",
		"Default": "src/main",
		"NS":      "src/ns",
	}
	for alias, want := range wantImports {
		if got := result.Imports[alias]; got != want {
			t.Errorf("Imports[%q] = %q, want %q", alias, got, want)
		}
	}

	if !hasUnresolved(result, goextract.RefKindImports, "./util", "") {
		t.Errorf("expected an imports dependency ref for ./util, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "helper", "helper") {
		t.Errorf("expected a named-import call to helper (self-referential PkgAlias trick), got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "Default", "Default") {
		t.Errorf("expected a default-import call to Default, got %+v", result.Unresolved)
	}
	if !hasUnresolved(result, goextract.RefKindCalls, "doWork", "NS") {
		t.Errorf("expected a namespace-import call NS.doWork, got %+v", result.Unresolved)
	}
}

// TestExtract_ModuleSpecifierResolution proves resolveModuleSpecifier
// resolves both a relative specifier (the priority tier) and a
// tsconfig.json paths-aliased specifier, and that an unresolvable bare
// package specifier leaves no Imports entry (an accepted, documented gap
// — external/node_modules resolution is out of scope).
func TestExtract_ModuleSpecifierResolution(t *testing.T) {
	t.Run("relative", func(t *testing.T) {
		target, ok := resolveModuleSpecifier("src/app/main.ts", "../util/helper")
		if !ok {
			t.Fatal("expected relative specifier to resolve")
		}
		if want := "src/util/helper"; target != want {
			t.Errorf("resolveModuleSpecifier = %q, want %q", target, want)
		}
	})

	t.Run("tsconfig paths alias", func(t *testing.T) {
		SetConfig(Config{BaseURL: ".", Paths: map[string][]string{"@app/*": {"src/app/*"}}})
		t.Cleanup(func() { SetConfig(Config{}) })

		p := newTestParser(t, "ts")
		src := "import { Foo } from '@app/foo';\nexport function run(): void {\n  Foo();\n}\n"
		result, err := Extract(p, "src/index", "src/index.ts", []byte(src))
		if err != nil {
			t.Fatalf("Extract returned error: %v", err)
		}
		if got, want := result.Imports["Foo"], "src/app/foo"; got != want {
			t.Errorf("Imports[Foo] = %q, want %q", got, want)
		}
	})

	t.Run("external package specifier does not resolve", func(t *testing.T) {
		SetConfig(Config{})
		target, ok := resolveModuleSpecifier("src/app.ts", "react")
		if ok {
			t.Errorf("expected an external package specifier not to resolve, got %q", target)
		}
	})
}

// TestExtract_ExportedConsts proves an `export const NAME = (...) => ...`
// (or a function/generator expression) emits a KindFunction node (the
// dominant modern TS/JS exported-function idiom), a plain `export const
// NAME = <value>` emits a KindConstant node, and a NON-exported top-level
// const emits no node at all (this plan's own bounded "exported consts"
// scope).
func TestExtract_ExportedConsts(t *testing.T) {
	src := `export const add = (a: number, b: number): number => a + b;
export const PI = 3.14;
const secret = "shh";
`
	p := newTestParser(t, "ts")
	result, err := Extract(p, "m", "consts.ts", []byte(src))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	fn := findNode(result, goextract.KindFunction, "add")
	if fn == nil {
		t.Fatalf("expected a function node named add, got %+v", result.Nodes)
	}
	if !fn.Node.IsExported {
		t.Error("add.IsExported = false, want true")
	}

	pi := findNode(result, goextract.KindConstant, "PI")
	if pi == nil {
		t.Fatalf("expected a constant node named PI, got %+v", result.Nodes)
	}
	if !pi.Node.IsExported {
		t.Error("PI.IsExported = false, want true")
	}

	if hasNodeNamed(result, "secret") {
		t.Errorf("expected no node for a non-exported top-level const, got %+v", result.Nodes)
	}
}

// TestExtract_ModuleKeyPassedThroughUnchanged proves moduleKey flows
// straight through to FileResult.ImportPath — TS/JS needs no parse-time
// override (unlike Java's package/C#'s namespace), since its cross-file
// identity is entirely path-derived (mirrors pyextract's identical
// decision, types.go's package doc).
func TestExtract_ModuleKeyPassedThroughUnchanged(t *testing.T) {
	p := newTestParser(t, "ts")
	result, err := Extract(p, "src/app", "src/app.ts", []byte("export class Widget {}\n"))
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.ImportPath != "src/app" {
		t.Errorf("ImportPath = %q, want %q (moduleKey passed through unchanged)", result.ImportPath, "src/app")
	}
}

// stubOversizedParser simulates the parser.Parser contract for a file that
// trips parser.MaxSourceBytes, mirroring csharpextract_test.go's/
// javaextract_test.go's stub of the same name.
type stubOversizedParser struct{}

func (stubOversizedParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	return nil, parser.ErrSourceTooLarge
}

func (stubOversizedParser) Close() error { return nil }

// TestExtract_OversizedFileSkippedNotFatal proves parser.ErrSourceTooLarge
// (or any Parse error) is recorded on FileResult.Err with a nil returned
// error — skip-not-fatal (RESEARCH Pitfall 4, threat T-05-DoS), enforced
// uniformly across all three grammars since they share one Extract.
func TestExtract_OversizedFileSkippedNotFatal(t *testing.T) {
	result, err := Extract(stubOversizedParser{}, "m", "Big.ts", []byte("export class Big {}\n"))
	if err != nil {
		t.Fatalf("Extract returned a non-nil error for a per-file skip: %v", err)
	}
	if result.Err == nil {
		t.Fatalf("FileResult.Err = nil, want parser.ErrSourceTooLarge")
	}
	if !errors.Is(result.Err, parser.ErrSourceTooLarge) {
		t.Errorf("FileResult.Err = %v, want wrapping parser.ErrSourceTooLarge", result.Err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes for a skipped file, got %+v", result.Nodes)
	}
}
