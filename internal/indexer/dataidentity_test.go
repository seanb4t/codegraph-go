package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestDataSymbolIdentity covers the identity of data symbols. A field's
// id does not change when its lines move. A Java field and a method of
// one name are two symbols with two ids. The references and calls of a
// method point at the final ids. A struct field is never a candidate for
// a bare call, in the index that a fresh run builds and in the one that a
// sync seeds from the store.
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

	ids := map[string]string{}
	// check returns the ids of the Java field size, the Java method size
	// and the Go field T.Name.
	check := func(stage string) (javaField, javaMethod, goField string) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		fields := declaredNodes(t, r, box, goextract.KindField, "size")
		methods := declaredNodes(t, r, box, goextract.KindMethod, "size")
		if len(fields) != 1 || len(methods) != 1 {
			t.Fatalf("%s: field size = %v and method size = %v, want one node each", stage, fields, methods)
		}
		if fields[0].Id == methods[0].Id {
			t.Fatalf("%s: field and method size share the id %s", stage, fields[0].Id)
		}
		if refs := dataEdges(t, r, box, goextract.KindMethod, "Box.area", goextract.RefKindReferences); refs["field Box.size"] != "read" {
			t.Errorf("%s: area references %v, want a read of the field Box.size", stage, refs)
		}
		calls := dataEdges(t, r, box, goextract.KindMethod, "Box.area", goextract.RefKindCalls)
		if _, ok := calls["method Box.size"]; !ok || len(calls) != 1 {
			t.Errorf("%s: area calls %v, want only the method Box.size", stage, calls)
		}
		goFields := declaredNodes(t, r, "g/t.go", goextract.KindField, "Name")
		if len(goFields) != 1 {
			t.Fatalf("%s: Go field Name = %v, want one node", stage, goFields)
		}
		use := declaredNodes(t, r, "g/u.go", goextract.KindFunction, "use")[0].Id
		for _, kind := range []string{goextract.RefKindCalls, goextract.RefKindReferences} {
			if hasEdge(t, r, use, kind, goFields[0].Id) {
				t.Errorf("%s: the bare call Name() has a %s edge to the field T.Name", stage, kind)
			}
		}
		return fields[0].Id, methods[0].Id, goFields[0].Id
	}
	ids["java field"], ids["java method"], ids["go field"] = check("run")

	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(box, "// moved\n\n"+boxSrc)
	write("g/t.go", "// moved\n\npackage g\n\ntype T struct {\n\tName int\n}\n")
	write("g/u.go", "package g\n\n// edited\nfunc use() int { return Name() }\n")
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	after := map[string]string{}
	after["java field"], after["java method"], after["go field"] = check("sync")
	for name, id := range ids {
		if after[name] != id {
			t.Errorf("the %s id changed from %s to %s when its lines moved", name, id, after[name])
		}
	}
	requireSyncEqualsRun(t, root, storeDir)
}
