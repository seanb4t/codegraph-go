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

// TestLanguageRegistry_Python proves the registry resolves Python by both
// ID and extension, hands back a working parser + extractor, and that
// Python's ModuleKey degrades to path-based identity absent a resolvable
// descriptor (D-03) — mirroring TestLanguageRegistry_Java/_CSharp's shape
// (LANG-04).
func TestLanguageRegistry_Python(t *testing.T) {
	spec, ok := lookupLanguageByID("python")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"python\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".py" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected python spec Extensions to contain \".py\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".py")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".py\") to return ok=true")
	}
	if byExt.ID != "python" {
		t.Fatalf("expected .py to resolve to the python spec, got ID=%q", byExt.ID)
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
	if got, want := spec.ModuleKey(nil, "sub/widget.py"), "sub/widget.py"; got != want {
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

// TestLanguageRegistry_TypeScript proves the registry resolves all three
// TS/JS grammars ("typescript"/"tsx"/"javascript") by both ID and every
// registered extension, hands back a working parser + shared extractor per
// registration, and that ModuleKey is UNCONDITIONALLY the extension-
// stripped relPath — even with a nil descriptor, diverging from every
// other priority-4 sibling's "nil descriptor -> raw relPath" convention
// (languages_typescript.go's own doc comment explains why: TS/JS's
// relative-specifier resolution inside tsextract.Extract must match this
// file's own key by construction, descriptor or not) — mirroring
// TestLanguageRegistry_Java/_CSharp/_Python's shape (LANG-05).
func TestLanguageRegistry_TypeScript(t *testing.T) {
	cases := []struct {
		id   string
		exts []string
	}{
		{id: "typescript", exts: []string{".ts"}},
		{id: "tsx", exts: []string{".tsx"}},
		{id: "javascript", exts: []string{".js", ".jsx", ".mjs", ".cjs"}},
	}

	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			spec, ok := lookupLanguageByID(c.id)
			if !ok {
				t.Fatalf("expected lookupLanguageByID(%q) to return ok=true", c.id)
			}

			for _, ext := range c.exts {
				byExt, ok := lookupLanguageByExt(ext)
				if !ok {
					t.Fatalf("expected lookupLanguageByExt(%q) to return ok=true", ext)
				}
				if byExt.ID != c.id {
					t.Fatalf("expected %s to resolve to the %s spec, got ID=%q", ext, c.id, byExt.ID)
				}
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
			if got, want := spec.ModuleKey(nil, "sub/Widget"+c.exts[0]), "sub/Widget"; got != want {
				t.Errorf("ModuleKey(nil descriptor) = %q, want %q (unconditional extension-stripped identity)", got, want)
			}
		})
	}
}

// TestLanguageRegistry_Rust proves the registry resolves Rust by both ID
// and extension, hands back a working parser + extractor, and that Rust's
// ModuleKey degrades to path-based identity absent a resolvable descriptor
// (D-03) — mirroring TestLanguageRegistry_Python's shape (LANG-06,
// mainstream tier).
func TestLanguageRegistry_Rust(t *testing.T) {
	spec, ok := lookupLanguageByID("rust")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"rust\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".rs" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected rust spec Extensions to contain \".rs\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".rs")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".rs\") to return ok=true")
	}
	if byExt.ID != "rust" {
		t.Fatalf("expected .rs to resolve to the rust spec, got ID=%q", byExt.ID)
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
	if got, want := spec.ModuleKey(nil, "sub/widget.rs"), "sub/widget.rs"; got != want {
		t.Errorf("ModuleKey(nil descriptor) = %q, want %q (D-03 path-identity fallback)", got, want)
	}
}

// TestLanguageRegistry_Ruby proves the registry resolves Ruby by both ID
// and extension, hands back a working parser + extractor, and that Ruby's
// ModuleKey is UNCONDITIONALLY the extension-stripped relPath — mirroring
// TestLanguageRegistry_TypeScript's own descriptor-independent pattern
// (LANG-06, mainstream tier).
func TestLanguageRegistry_Ruby(t *testing.T) {
	spec, ok := lookupLanguageByID("ruby")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"ruby\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".rb" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected ruby spec Extensions to contain \".rb\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".rb")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".rb\") to return ok=true")
	}
	if byExt.ID != "ruby" {
		t.Fatalf("expected .rb to resolve to the ruby spec, got ID=%q", byExt.ID)
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
	if got, want := spec.ModuleKey(nil, "sub/widget.rb"), "sub/widget"; got != want {
		t.Errorf("ModuleKey(nil descriptor) = %q, want %q (unconditional extension-stripped identity)", got, want)
	}
}

// TestLanguageRegistry_PHP proves the registry resolves PHP by both ID and
// extension, hands back a working parser + extractor, and that PHP's
// ModuleKey degrades to path-based identity absent a resolvable descriptor
// (D-03) — mirroring TestLanguageRegistry_CSharp's parse-time-override
// rationale (LANG-06, mainstream tier).
func TestLanguageRegistry_PHP(t *testing.T) {
	spec, ok := lookupLanguageByID("php")
	if !ok {
		t.Fatal("expected lookupLanguageByID(\"php\") to return ok=true")
	}

	foundExt := false
	for _, ext := range spec.Extensions {
		if ext == ".php" {
			foundExt = true
			break
		}
	}
	if !foundExt {
		t.Fatalf("expected php spec Extensions to contain \".php\", got %v", spec.Extensions)
	}

	byExt, ok := lookupLanguageByExt(".php")
	if !ok {
		t.Fatal("expected lookupLanguageByExt(\".php\") to return ok=true")
	}
	if byExt.ID != "php" {
		t.Fatalf("expected .php to resolve to the php spec, got ID=%q", byExt.ID)
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
	if got, want := spec.ModuleKey(nil, "sub/Widget.php"), "sub/Widget.php"; got != want {
		t.Errorf("ModuleKey(nil descriptor) = %q, want %q (D-03 path-identity fallback)", got, want)
	}
}

// TestPHPNamespaceFor proves the PSR-4 longest-prefix-match resolution
// (languages_php.go) picks the most specific directory mapping and joins
// the remaining path segments with PHP's own "\" separator.
func TestPHPNamespaceFor(t *testing.T) {
	psr4 := map[string]string{
		`App\`:       "src/",
		`App\Sub\`:   "src/sub/",
		`Vendor\Ns\`: "lib/",
	}

	cases := []struct {
		relPath string
		want    string
		wantOK  bool
	}{
		{relPath: "src/Widget.php", want: `App`, wantOK: true},
		{relPath: "src/Models/Widget.php", want: `App\Models`, wantOK: true},
		{relPath: "src/sub/Widget.php", want: `App\Sub`, wantOK: true},
		{relPath: "lib/Widget.php", want: `Vendor\Ns`, wantOK: true},
		{relPath: "unrelated/Widget.php", want: "", wantOK: false},
	}
	for _, c := range cases {
		got, ok := phpNamespaceFor(psr4, c.relPath)
		if ok != c.wantOK || got != c.want {
			t.Errorf("phpNamespaceFor(%q) = (%q, %v), want (%q, %v)", c.relPath, got, ok, c.want, c.wantOK)
		}
	}
}
