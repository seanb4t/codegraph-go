package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
)

// TestDataSymbolIdentity covers the identity of data symbols. A field's
// id is the declaration id of its owner-qualified name, so it does not
// change when lines move. A Java field and a method of one name are two
// symbols with two ids. The references and calls of a method point at
// the final ids. A struct field is never a candidate for a bare call, in
// the index that a fresh run builds and in the one that a sync seeds
// from the store.
func TestDataSymbolIdentity(t *testing.T) {
	const box = "p/Box.java"
	boxSrc := `package p;

public class Box {
    int size;

    int size() {
        return size;
    }

    int area() {
        return size() * this.size;
    }
}
`
	root := writeFixture(t, map[string]string{
		box:      boxSrc,
		"go.mod": "module example.com/m\n\ngo 1.22\n",
		"g/t.go": "package g\n\ntype T struct {\n\tName int\n}\n",
		"g/u.go": "package g\n\nfunc use() int { return Name() }\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	fieldID := nodeid.DeclID(goextract.KindField, "Box.size", box, "")
	goField := nodeid.DeclID(goextract.KindField, "T.Name", "g/t.go", "")
	check := func(stage string) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		fields := declaredNodes(t, r, box, goextract.KindField, "size")
		methods := declaredNodes(t, r, box, goextract.KindMethod, "size")
		if len(fields) != 1 || fields[0].Id != fieldID {
			t.Fatalf("%s: field size = %v, want one node with id %s", stage, fields, fieldID)
		}
		if len(methods) != 1 || methods[0].Id == fieldID {
			t.Fatalf("%s: method size = %v, want one node apart from the field", stage, methods)
		}
		if refs := dataEdges(t, r, box, goextract.KindMethod, "Box.area", goextract.RefKindReferences); refs["field Box.size"] != "read" {
			t.Errorf("%s: area references %v, want a read of the field Box.size", stage, refs)
		}
		calls := dataEdges(t, r, box, goextract.KindMethod, "Box.area", goextract.RefKindCalls)
		if _, ok := calls["method Box.size"]; !ok || len(calls) != 1 {
			t.Errorf("%s: area calls %v, want only the method Box.size", stage, calls)
		}
		if got := declaredNodes(t, r, "g/t.go", goextract.KindField, "Name"); len(got) != 1 || got[0].Id != goField {
			t.Fatalf("%s: Go field Name = %v, want id %s", stage, got, goField)
		}
		use := declaredNodes(t, r, "g/u.go", goextract.KindFunction, "use")[0].Id
		for _, kind := range []string{goextract.RefKindCalls, goextract.RefKindReferences} {
			if hasEdge(t, r, use, kind, goField) {
				t.Errorf("%s: the bare call Name() has a %s edge to the field T.Name", stage, kind)
			}
		}
	}
	check("run")

	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(box, "// moved\n\n"+boxSrc)
	write("g/u.go", "package g\n\n// edited\nfunc use() int { return Name() }\n")
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	check("sync")
	requireSyncEqualsRun(t, root, storeDir)
}
