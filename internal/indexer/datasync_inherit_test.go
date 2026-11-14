package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

// inheritStep is one edit of an inheritance sync scenario. A nil content
// removes the file.
type inheritStep struct {
	name  string
	files map[string]*string
}

func text(s string) *string { return &s }

// runInheritSteps indexes files, then applies each step and requires that
// the synced store equals a fresh index of the tree. reparsed names the
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
	t.Run("typescript", func(t *testing.T) {
		base := "import { run } from './lib';\n\nexport class Base {\n  count = 0;\n  hook = run;\n}\n"
		derived := "import { Base } from './base';\n\nexport class Derived extends Base {\n  fire() {\n    this.hook();\n  }\n}\n"
		runInheritSteps(t, map[string]string{
			"lib.ts":     "export function run() {\n  return 0;\n}\n",
			"base.ts":    base,
			"derived.ts": derived,
			"reader.ts":  "import { Derived } from './derived';\n\nexport function use(d: Derived) {\n  return d.count;\n}\n",
		}, []inheritStep{
			{"subclass gains a shadowing field", map[string]*string{"derived.ts": text("import { Base } from './base';\n\nexport class Derived extends Base {\n  count = 1;\n  fire() {\n    this.hook();\n  }\n}\n")}},
			{"subclass loses the shadowing field", map[string]*string{"derived.ts": text(derived)}},
			{"base loses the field", map[string]*string{"base.ts": text("import { run } from './lib';\n\nexport class Base {\n  hook = run;\n}\n")}},
			{"base regains the field", map[string]*string{"base.ts": text(base)}},
			{"a subclass can replace the callable field", map[string]*string{"sub.ts": text("import { Derived } from './derived';\n\nexport class Sub extends Derived {}\n")}},
			{"the subclass goes", map[string]*string{"sub.ts": nil}},
			{"the field moves to another class", map[string]*string{"base.ts": text("import { run } from './lib';\n\nexport class Base {\n  hook = run;\n}\n\nexport class Other {\n  count = 0;\n}\n")}},
			{"the field moves back", map[string]*string{"base.ts": text(base)}},
			{"a method body changes", map[string]*string{"derived.ts": text("import { Base } from './base';\n\nexport class Derived extends Base {\n  fire() {\n    this.hook();\n    return 1;\n  }\n}\n")}},
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
			{"subclass gains a shadowing field", map[string]*string{"derived.py": text("from base import Base\n\n\nclass Derived(Base):\n    count = 1\n\n    def fire(self):\n        self.hook()\n")}},
			{"subclass loses the shadowing field", map[string]*string{"derived.py": text(derived)}},
			{"base loses the field", map[string]*string{"base.py": text("from lib import run\n\n\nclass Base:\n    hook = run\n")}},
			{"base regains the field", map[string]*string{"base.py": text(base)}},
			{"a subclass can replace the callable field", map[string]*string{"sub.py": text("from derived import Derived\n\n\nclass Sub(Derived):\n    pass\n")}},
			{"the subclass goes", map[string]*string{"sub.py": nil}},
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
			{"subclass gains a shadowing field", map[string]*string{"p/Derived.java": text("package p;\n\npublic class Derived extends Base {\n    protected int count;\n}\n")}},
			{"subclass loses the shadowing field", map[string]*string{"p/Derived.java": text(derived)}},
			{"base loses the field", map[string]*string{"p/Base.java": text("package p;\n\npublic class Base {\n}\n")}},
			{"base regains the field", map[string]*string{"p/Base.java": text(base)}},
			{"the base field becomes private", map[string]*string{"p/Base.java": text("package p;\n\npublic class Base {\n    private int count;\n}\n")}},
			{"the base field is protected again", map[string]*string{"p/Base.java": text(base)}},
		}, nil)
	})
}
