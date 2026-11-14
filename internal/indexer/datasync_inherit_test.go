package indexer

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
)

// inheritStep is one edit of an inheritance sync scenario. A nil content
// removes the file. check, when set, runs on the synced store.
type inheritStep struct {
	name  string
	files map[string]*string
	check func(t *testing.T, r graphstore.Reader)
}

// countReader is the declaration of a scenario that reads a field named
// count.
type countReader struct{ path, kind, name string }

// countReads returns the ids of the fields named count that the
// declaration reader refers to.
func countReads(t *testing.T, r graphstore.Reader, reader countReader) []string {
	t.Helper()
	srcs := declaredNodes(t, r, reader.path, reader.kind, reader.name)
	if len(srcs) != 1 {
		t.Fatalf("%d %s %s in %s, want 1", len(srcs), reader.kind, reader.name, reader.path)
	}
	var out []string
	it, err := r.IterateEdges(srcs[0].Id)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindReferences {
			continue
		}
		n, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		if n.Kind == goextract.KindField && n.Name == "count" {
			out = append(out, n.Id)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	return out
}

// countField returns the id of the field count of owner in path.
func countField(t *testing.T, r graphstore.Reader, path, owner string) string {
	t.Helper()
	for _, n := range declaredNodes(t, r, path, goextract.KindField, "count") {
		if strings.HasSuffix(n.QualifiedName, owner+".count") {
			return n.Id
		}
	}
	t.Fatalf("no field %s.count in %s", owner, path)
	return ""
}

// readsCount returns a step check: reader refers to the field count of
// owner in path and to no other field named count. An empty path means
// that reader refers to no field named count.
func readsCount(reader countReader, path, owner string) func(*testing.T, graphstore.Reader) {
	return func(t *testing.T, r graphstore.Reader) {
		t.Helper()
		got := countReads(t, r, reader)
		if path == "" {
			if len(got) != 0 {
				t.Errorf("%s refers to the fields %v, want no field count", reader.name, got)
			}
			return
		}
		want := countField(t, r, path, owner)
		if len(got) != 1 || got[0] != want {
			t.Errorf("%s refers to the fields %v, want only %s.count %s", reader.name, got, owner, want)
		}
	}
}

func text(s string) *string { return &s }

// runInheritSteps indexes files, then applies each step and requires that
// the synced store equals a fresh index of the tree and passes the step's
// check. reparsed names the
// steps whose sync must reparse an exact number of files.
func runInheritSteps(t *testing.T, files map[string]string, steps []inheritStep, reparsed map[string]int) {
	t.Helper()
	root := writeFixture(t, files)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, s := range steps {
		for path, content := range s.files {
			if content == nil {
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
				continue
			}
			writeFile(t, filepath.Join(root, path), *content)
		}
		stats, err := Sync(root, storeDir, Options{})
		if err != nil {
			t.Fatalf("Sync after %q: %v", s.name, err)
		}
		t.Run(s.name, func(t *testing.T) {
			if want, ok := reparsed[s.name]; ok && stats.FilesReparsed != want {
				t.Errorf("the sync reparsed %d files, want %d", stats.FilesReparsed, want)
			}
			requireSyncEqualsRun(t, root, storeDir)
			if s.check != nil {
				r, done := openSnapshot(t, storeDir)
				defer done()
				s.check(t, r)
			}
		})
	}
}

// TestDataSymbolSyncEquivalence_Inheritance edits the classes that own
// fields while the files that read the fields stay unchanged. After each
// sync the store must equal a fresh index: a read through a subclass
// follows a shadowing field that appears or goes, a field that the base
// class gains, loses or moves to another class, and a call through a
// field follows a subclass that can replace it. An edit that changes no
// field, type or supertype reparses only its own file.
func TestDataSymbolSyncEquivalence_Inheritance(t *testing.T) {
	tsReader := countReader{"reader.ts", goextract.KindFunction, "use"}
	pyReader := countReader{"reader.py", goextract.KindMethod, "use"}
	javaReader := countReader{"p/Reader.java", goextract.KindMethod, "use"}
	// After the move, base.ts declares Other.count and no Base.count, and
	// the reader has no edge to the removed Base.count.
	fieldMovedToOther := func(t *testing.T, r graphstore.Reader) {
		t.Helper()
		other := countField(t, r, "base.ts", "Other")
		if fields := declaredNodes(t, r, "base.ts", goextract.KindField, "count"); len(fields) != 1 || fields[0].Id != other {
			t.Errorf("base.ts declares %d fields named count, want only Other.count", len(fields))
		}
		removed := nodeid.DeclID(goextract.KindField, "Base.count", "base.ts", "")
		use := declaredNodes(t, r, tsReader.path, tsReader.kind, tsReader.name)
		if len(use) != 1 {
			t.Fatalf("%d functions use in reader.ts, want 1", len(use))
		}
		if hasEdge(t, r, use[0].Id, goextract.RefKindReferences, removed) {
			t.Errorf("use still refers to the removed Base.count %s", removed)
		}
		readsCount(tsReader, "", "")(t, r)
	}
	t.Run("typescript", func(t *testing.T) {
		base := "import { run } from './lib';\n\nexport class Base {\n  count = 0;\n  hook = run;\n}\n"
		derived := "import { Base } from './base';\n\nexport class Derived extends Base {\n  fire() {\n    this.hook();\n  }\n}\n"
		runInheritSteps(t, map[string]string{
			"lib.ts":     "export function run() {\n  return 0;\n}\n",
			"base.ts":    base,
			"derived.ts": derived,
			"reader.ts":  "import { Derived } from './derived';\n\nexport function use(d: Derived) {\n  return d.count;\n}\n",
		}, []inheritStep{
			{"subclass gains a shadowing field", map[string]*string{"derived.ts": text("import { Base } from './base';\n\nexport class Derived extends Base {\n  count = 1;\n  fire() {\n    this.hook();\n  }\n}\n")}, readsCount(tsReader, "derived.ts", "Derived")},
			{"subclass loses the shadowing field", map[string]*string{"derived.ts": text(derived)}, readsCount(tsReader, "base.ts", "Base")},
			{"base loses the field", map[string]*string{"base.ts": text("import { run } from './lib';\n\nexport class Base {\n  hook = run;\n}\n")}, readsCount(tsReader, "", "")},
			{"base regains the field", map[string]*string{"base.ts": text(base)}, readsCount(tsReader, "base.ts", "Base")},
			{"a subclass can replace the callable field", map[string]*string{"sub.ts": text("import { Derived } from './derived';\n\nexport class Sub extends Derived {}\n")}, nil},
			{"the subclass goes", map[string]*string{"sub.ts": nil}, nil},
			{"the field moves to another class", map[string]*string{"base.ts": text("import { run } from './lib';\n\nexport class Base {\n  hook = run;\n}\n\nexport class Other {\n  count = 0;\n}\n")}, fieldMovedToOther},
			{"the field moves back", map[string]*string{"base.ts": text(base)}, readsCount(tsReader, "base.ts", "Base")},
			{"a method body changes", map[string]*string{"derived.ts": text("import { Base } from './base';\n\nexport class Derived extends Base {\n  fire() {\n    this.hook();\n    return 1;\n  }\n}\n")}, nil},
		}, map[string]int{"a method body changes": 1})
	})

	t.Run("python", func(t *testing.T) {
		base := "from lib import run\n\n\nclass Base:\n    count = 0\n    hook = run\n"
		derived := "from base import Base\n\n\nclass Derived(Base):\n    def fire(self):\n        self.hook()\n"
		runInheritSteps(t, map[string]string{
			"lib.py":     "def run():\n    return 0\n",
			"base.py":    base,
			"derived.py": derived,
			"reader.py":  "from derived import Derived\n\n\nclass Reader(Derived):\n    def use(self):\n        return self.count\n",
		}, []inheritStep{
			{"subclass gains a shadowing field", map[string]*string{"derived.py": text("from base import Base\n\n\nclass Derived(Base):\n    count = 1\n\n    def fire(self):\n        self.hook()\n")}, readsCount(pyReader, "derived.py", "Derived")},
			{"subclass loses the shadowing field", map[string]*string{"derived.py": text(derived)}, readsCount(pyReader, "base.py", "Base")},
			{"base loses the field", map[string]*string{"base.py": text("from lib import run\n\n\nclass Base:\n    hook = run\n")}, readsCount(pyReader, "", "")},
			{"base regains the field", map[string]*string{"base.py": text(base)}, readsCount(pyReader, "base.py", "Base")},
			{"a subclass can replace the callable field", map[string]*string{"sub.py": text("from derived import Derived\n\n\nclass Sub(Derived):\n    pass\n")}, nil},
			{"the subclass goes", map[string]*string{"sub.py": nil}, nil},
		}, nil)
	})

	t.Run("java", func(t *testing.T) {
		base := "package p;\n\npublic class Base {\n    protected int count;\n}\n"
		derived := "package p;\n\npublic class Derived extends Base {\n}\n"
		runInheritSteps(t, map[string]string{
			"p/Base.java":    base,
			"p/Derived.java": derived,
			"p/Reader.java":  "package p;\n\npublic class Reader {\n    int use(Derived d) {\n        return d.count;\n    }\n}\n",
		}, []inheritStep{
			{"subclass gains a shadowing field", map[string]*string{"p/Derived.java": text("package p;\n\npublic class Derived extends Base {\n    protected int count;\n}\n")}, readsCount(javaReader, "p/Derived.java", "Derived")},
			{"subclass loses the shadowing field", map[string]*string{"p/Derived.java": text(derived)}, readsCount(javaReader, "p/Base.java", "Base")},
			{"base loses the field", map[string]*string{"p/Base.java": text("package p;\n\npublic class Base {\n}\n")}, readsCount(javaReader, "", "")},
			{"base regains the field", map[string]*string{"p/Base.java": text(base)}, readsCount(javaReader, "p/Base.java", "Base")},
			{"the base field becomes private", map[string]*string{"p/Base.java": text("package p;\n\npublic class Base {\n    private int count;\n}\n")}, nil},
			{"the base field is protected again", map[string]*string{"p/Base.java": text(base)}, nil},
		}, nil)
	})
}

// TestOwnerDependents_UnindexedSupertype edits a file that holds a class
// whose supertype is not indexed. The store has no edge to that
// supertype, so an edit of a method body is not an owner change and
// reparses only its own file. A new shadowing field in the same file is
// an owner change and reparses the reader too.
func TestOwnerDependents_UnindexedSupertype(t *testing.T) {
	cases := []struct {
		name, derived, base string
		files               map[string]string
		// body is the derived file with a placeholder in a method body.
		// field is the derived file with a shadowing field count.
		body, field string
		reader      countReader
	}{
		{
			name:    "python",
			derived: "derived.py",
			base:    "base.py",
			files: map[string]string{
				"base.py":   "class Base:\n    count = 0\n",
				"reader.py": "from derived import Derived\n\n\nclass Reader(Derived):\n    def use(self):\n        return self.count\n",
			},
			body:   "from base import Base\n\n\nclass Derived(Base):\n    def fire(self):\n        return %s\n\n\nclass Oops(Exception):\n    pass\n",
			field:  "from base import Base\n\n\nclass Derived(Base):\n    count = 1\n\n    def fire(self):\n        return 1\n\n\nclass Oops(Exception):\n    pass\n",
			reader: countReader{"reader.py", goextract.KindMethod, "use"},
		},
		{
			name:    "typescript",
			derived: "derived.ts",
			base:    "base.ts",
			files: map[string]string{
				"base.ts":   "export class Base {\n  count = 0;\n}\n",
				"reader.ts": "import { Derived } from './derived';\n\nexport function use(d: Derived) {\n  return d.count;\n}\n",
			},
			body:   "import { Base } from './base';\nimport { Unindexed } from 'unindexed';\n\nexport class Derived extends Base {\n  fire() {\n    return %s;\n  }\n}\n\nexport class Oops extends Unindexed {}\n",
			field:  "import { Base } from './base';\nimport { Unindexed } from 'unindexed';\n\nexport class Derived extends Base {\n  count = 1;\n  fire() {\n    return 1;\n  }\n}\n\nexport class Oops extends Unindexed {}\n",
			reader: countReader{"reader.ts", goextract.KindFunction, "use"},
		},
		{
			name:    "java",
			derived: "p/Derived.java",
			base:    "p/Base.java",
			files: map[string]string{
				"p/Base.java":   "package p;\n\npublic class Base {\n    protected int count;\n}\n",
				"p/Reader.java": "package p;\n\npublic class Reader {\n    int use(Derived d) {\n        return d.count;\n    }\n}\n",
			},
			body:   "package p;\n\npublic class Derived extends Base {\n    int fire() {\n        return %s;\n    }\n}\n\nclass Oops extends RuntimeException {\n}\n",
			field:  "package p;\n\npublic class Derived extends Base {\n    protected int count;\n\n    int fire() {\n        return 1;\n    }\n}\n\nclass Oops extends RuntimeException {\n}\n",
			reader: countReader{"p/Reader.java", goextract.KindMethod, "use"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := map[string]string{c.derived: fmt.Sprintf(c.body, "0")}
			maps.Copy(files, c.files)
			runInheritSteps(t, files, []inheritStep{
				{"a method body changes", map[string]*string{c.derived: text(fmt.Sprintf(c.body, "1"))}, readsCount(c.reader, c.base, "Base")},
				{"the subclass gains a shadowing field", map[string]*string{c.derived: text(c.field)}, readsCount(c.reader, c.derived, "Derived")},
			}, map[string]int{"a method body changes": 1, "the subclass gains a shadowing field": 2})
		})
	}
}
