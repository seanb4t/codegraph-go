package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestDataReferenceMetadata_KindsAndStorage covers the declaration kinds
// of the data-symbols contract as the store holds them. Modifiers do not
// invent kinds: an uppercase Python assignment is a variable, and a Java
// private static final field and a TypeScript readonly field are fields.
// The visibility modifier stays node metadata. A JavaScript function
// expression bound to a nonexported constant is a function, and an alias
// to it keeps the constant kind. A Python annotation without a value is a variable. The
// access role of a references edge survives storage and a sync.
func TestDataReferenceMetadata_KindsAndStorage(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod":     "module example.com/m\n\ngo 1.22\n",
		"g/g.go":     "package g\n\nconst Max = 3\n\nvar count int\n\nfunc bump() {\n\tcount++\n\tcount = Max\n}\n",
		"m.py":       "MAX = 1\nx: int\n",
		"p/Box.java": "package p;\n\npublic class Box {\n    private static final int LIMIT = 1;\n}\n",
		"t.ts":       "export class Tag {\n  readonly label = \"a\";\n}\n",
		"m.js":       "const expr = function () {\n  return 1;\n};\nconst alias = expr;\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	check := func(stage string) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		for _, want := range []struct{ path, kind, name string }{
			{"g/g.go", goextract.KindConstant, "Max"},
			{"g/g.go", goextract.KindVariable, "count"},
			{"m.py", goextract.KindVariable, "MAX"},
			{"m.py", goextract.KindVariable, "x"},
			{"p/Box.java", goextract.KindField, "LIMIT"},
			{"t.ts", goextract.KindField, "label"},
			{"m.js", goextract.KindFunction, "expr"},
			{"m.js", goextract.KindConstant, "alias"},
		} {
			if got := declaredNodes(t, r, want.path, want.kind, want.name); len(got) != 1 {
				t.Errorf("%s: %s %s in %s = %d nodes, want 1", stage, want.kind, want.name, want.path, len(got))
			}
		}
		for _, kind := range []string{goextract.KindConstant, goextract.KindVariable} {
			if got := declaredNodes(t, r, "m.js", kind, "expr"); len(got) != 0 {
				t.Errorf("%s: m.js has %d %s nodes named expr beside its function node", stage, len(got), kind)
			}
		}
		if limit := declaredNodes(t, r, "p/Box.java", goextract.KindField, "LIMIT"); len(limit) == 1 && limit[0].Visibility != "private" {
			t.Errorf("%s: the field LIMIT has visibility %q, want its private modifier", stage, limit[0].Visibility)
		}
		refs := dataEdges(t, r, "g/g.go", goextract.KindFunction, "bump", goextract.RefKindReferences)
		if refs["variable count"] != goextract.AccessReadWrite || refs["constant Max"] != goextract.AccessRead {
			t.Errorf("%s: bump references %v, want count read_write and Max read", stage, refs)
		}
	}
	check("run")
	if err := os.WriteFile(filepath.Join(root, "g/g.go"), []byte("package g\n\n// edited\nconst Max = 3\n\nvar count int\n\nfunc bump() {\n\tcount++\n\tcount = Max\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	check("sync")
}
