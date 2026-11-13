package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// tsDataSource is one module for TestTypeScriptDataReferences, valid
// TypeScript, JavaScript and TSX alike.
const tsDataSource = `import { run } from "./run";

let total = 0;
const handler = run;
const helper = () => 1;
const key = "k";

function use(v) { return v; }

class Base {
  count = 0;
  shared = 1;
  onFire = run;
  fire() { this.onFire(); }
}

class Derived extends Base {
  shared = 2;
  onDone = run;
  [key] = 3;
  bump() {
    this.count += 1;
    this.shared = 5;
    this.onDone();
  }
}

function scoped() {
  const total = 5;
  {
    let handler = 2;
    use(handler);
  }
  return total;
}

function reads() { return total; }

function writes() {
  total = 7;
  total++;
}

function callsHandler() {
  handler();
  helper();
}
`

// TestTypeScriptDataReferences covers the TypeScript, JavaScript and TSX
// references of the data-symbols contract. An unexported const that
// holds a function is read, never called, and a call through it reaches
// the function. An unexported arrow function is a callable target. A call
// through a class field reads the field and reaches the function the
// field holds. A field reference through this reaches the class that
// declares the name: the base class for an inherited field, the subclass
// for a field it redeclares. A computed field name declares nothing, a
// local in a function or block shadows the module binding of its name,
// and reads and writes carry their access role. A callable field of a
// class that another class extends establishes no call. A private TypeScript
// field of a base class is no reference target from its subclass.
func TestTypeScriptDataReferences(t *testing.T) {
	for _, ext := range []string{"ts", "js", "tsx"} {
		t.Run(ext, func(t *testing.T) {
			files := map[string]string{
				"run." + ext:  "export function run() { return 0; }\n",
				"data." + ext: tsDataSource,
			}
			if ext == "ts" {
				files["priv.ts"] = `class Vault {
  private hidden = 1;
  protected shown = 2;
}

class Thief extends Vault {
  look() { return this.hidden + this.shown; }
}
`
			}
			r, _ := runAndSnapshot(t, testtree.Write(t, files))
			data := "data." + ext

			handler := dataEdges(t, r, data, goextract.KindConstant, "handler", goextract.RefKindReferences)
			if _, ok := handler["function run"]; !ok {
				t.Errorf("handler references %v, want run", handler)
			}
			if calls := dataEdges(t, r, data, goextract.KindConstant, "handler", goextract.RefKindCalls); len(calls) != 0 {
				t.Errorf("handler calls %v, want none from an initializer that only names a function", calls)
			}

			calls := dataEdges(t, r, data, goextract.KindFunction, "callsHandler", goextract.RefKindCalls)
			if _, ok := calls["function run"]; !ok {
				t.Errorf("callsHandler calls %v, want run through handler", calls)
			}
			if _, ok := calls["function helper"]; !ok {
				t.Errorf("callsHandler calls %v, want the unexported arrow function helper", calls)
			}
			if _, ok := calls["constant handler"]; ok {
				t.Errorf("callsHandler calls the constant handler: %v", calls)
			}
			if got := dataEdges(t, r, data, goextract.KindFunction, "callsHandler", goextract.RefKindReferences)["constant handler"]; got != "read" {
				t.Errorf("callsHandler -> handler access = %q, want read", got)
			}

			bump := dataEdges(t, r, data, goextract.KindMethod, "Derived.bump", goextract.RefKindReferences)
			for target, want := range map[string]string{
				"field Base.count": "read_write", "field Derived.shared": "write", "field Derived.onDone": "read",
			} {
				if bump[target] != want {
					t.Errorf("bump -> %s access = %q, want %q (all: %v)", target, bump[target], want, bump)
				}
			}
			if _, ok := bump["field Base.shared"]; ok {
				t.Errorf("bump references Base.shared, which Derived redeclares: %v", bump)
			}
			bumpCalls := dataEdges(t, r, data, goextract.KindMethod, "Derived.bump", goextract.RefKindCalls)
			if _, ok := bumpCalls["function run"]; !ok || len(bumpCalls) != 1 {
				t.Errorf("bump calls %v, want only run through the field onDone", bumpCalls)
			}
			if fire := dataEdges(t, r, data, goextract.KindMethod, "Base.fire", goextract.RefKindCalls); len(fire) != 0 {
				t.Errorf("fire calls %v, want none: Derived extends Base, so a write through Derived can replace onFire", fire)
			}

			for kind, node := range map[string]string{goextract.KindField: "Derived.key"} {
				it, _ := r.IterateNodes()
				for it.Next() {
					if n := it.Node(); n.Kind == kind && (n.QualifiedName == node || n.QualifiedName == "Derived.[key]") {
						t.Errorf("computed field name became the node %s", n.QualifiedName)
					}
				}
				it.Close()
			}

			scoped := dataEdges(t, r, data, goextract.KindFunction, "scoped", goextract.RefKindReferences)
			for _, shadowed := range []string{"variable total", "constant handler"} {
				if _, ok := scoped[shadowed]; ok {
					t.Errorf("scoped references %s through a local: %v", shadowed, scoped)
				}
			}
			if got := dataEdges(t, r, data, goextract.KindFunction, "reads", goextract.RefKindReferences)["variable total"]; got != "read" {
				t.Errorf("reads -> total access = %q, want read", got)
			}
			if got := dataEdges(t, r, data, goextract.KindFunction, "writes", goextract.RefKindReferences)["variable total"]; got != "read_write" {
				t.Errorf("writes -> total access = %q, want read_write", got)
			}

			if ext == "ts" {
				look := dataEdges(t, r, "priv.ts", goextract.KindMethod, "Thief.look", goextract.RefKindReferences)
				if _, ok := look["field Vault.hidden"]; ok {
					t.Errorf("look references the private base field Vault.hidden: %v", look)
				}
				if look["field Vault.shown"] != "read" {
					t.Errorf("look -> Vault.shown access = %q, want read (all: %v)", look["field Vault.shown"], look)
				}
			}
		})
	}
}
