package tsextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestTypeScriptDataDeclarations covers the TypeScript, JavaScript and
// TSX declarations of the data-symbols contract: module var, let and
// const bindings, exported or not; each identifier a destructuring
// pattern binds; a function for an arrow function or a function
// expression; and one field node for each class field with an identifier
// or a private name, owned by its class. A computed or quoted field name
// and a method declare no field.
func TestTypeScriptDataDeclarations(t *testing.T) {
	for _, tc := range []struct{ grammar, path, src string }{
		{"ts", "m.ts", `export const LIMIT: number = 3;
let count = 0, spare;
var legacy;
const { left, right: renamed, ...rest } = source;
const [first, , second = 2] = list;
const helper = () => 1;
export const exported = function () { return 2; };
export const exportedArrow = () => 3;
const _ = 1;
const key = "k";
export class Box {
  static instances = 0;
  readonly label: string = "box";
  private secret = 1;
  #hidden = 2;
  [key] = 3;
  "quoted" = 4;
  onChange = () => {};
  size() { return 1; }
}
`},
		{"js", "m.js", `export const LIMIT = 3;
let count = 0, spare;
var legacy;
const { left, right: renamed, ...rest } = source;
const [first, , second = 2] = list;
const helper = () => 1;
export const exported = function () { return 2; };
export const exportedArrow = () => 3;
const _ = 1;
const key = "k";
export class Box {
  static instances = 0;
  label = "box";
  secret = 1;
  #hidden = 2;
  [key] = 3;
  "quoted" = 4;
  onChange = () => {};
  size() { return 1; }
}
`},
		{"tsx", "m.tsx", `export const LIMIT = 3;
let count = 0, spare;
var legacy;
const { left, right: renamed, ...rest } = source;
const [first, , second = 2] = list;
const helper = () => <div />;
export const exported = function () { return 2; };
export const exportedArrow = () => 3;
const _ = 1;
const key = "k";
export class Box {
  static instances = 0;
  label = "box";
  private secret = 1;
  #hidden = 2;
  [key] = 3;
  "quoted" = 4;
  onChange = () => {};
  size() { return <span />; }
}
`},
	} {
		t.Run(tc.grammar, func(t *testing.T) {
			result, err := Extract(newTestParser(t, tc.grammar), "m", tc.path, []byte(tc.src))
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			type key struct{ kind, qname string }
			count := map[key]int{}
			byKey := map[key]*goextract.ExtractedNode{}
			for i := range result.Nodes {
				n := result.Nodes[i].Node
				k := key{n.Kind, n.QualifiedName}
				count[k]++
				byKey[k] = &result.Nodes[i]
			}
			want := []key{
				{goextract.KindConstant, "LIMIT"}, {goextract.KindVariable, "count"},
				{goextract.KindVariable, "spare"}, {goextract.KindVariable, "legacy"},
				{goextract.KindConstant, "left"}, {goextract.KindConstant, "renamed"},
				{goextract.KindConstant, "rest"}, {goextract.KindConstant, "first"},
				{goextract.KindConstant, "second"}, {goextract.KindConstant, "key"},
				{goextract.KindFunction, "helper"}, {goextract.KindFunction, "exported"},
				{goextract.KindFunction, "exportedArrow"}, {goextract.KindConstant, "_"},
				{goextract.KindField, "Box.instances"}, {goextract.KindField, "Box.label"},
				{goextract.KindField, "Box.secret"}, {goextract.KindField, "Box.#hidden"},
				{goextract.KindField, "Box.onChange"}, {goextract.KindMethod, "Box.size"},
			}
			for _, k := range want {
				if count[k] != 1 {
					t.Errorf("%s %s: %d nodes, want 1", k.kind, k.qname, count[k])
				}
			}
			for _, name := range []string{"helper", "exported", "exportedArrow"} {
				for _, kind := range []string{goextract.KindConstant, goextract.KindVariable} {
					if count[key{kind, name}] != 0 {
						t.Errorf("%s has a %s node beside its function node", name, kind)
					}
				}
			}
			for k := range count {
				switch k.qname {
				case "right", "Box.key", "Box.quoted", "Box.size", "Box.[key]", `Box."quoted"`:
					if k.kind != goextract.KindMethod {
						t.Errorf("unexpected %s node %s", k.kind, k.qname)
					}
				}
			}

			for name, exported := range map[string]bool{"LIMIT": true, "count": false} {
				var n *goextract.ExtractedNode
				for _, kind := range []string{goextract.KindConstant, goextract.KindVariable} {
					if found := byKey[key{kind, name}]; found != nil {
						n = found
					}
				}
				if n != nil && n.Node.IsExported != exported {
					t.Errorf("%s IsExported = %v, want %v", name, n.Node.IsExported, exported)
				}
			}

			box := byKey[key{goextract.KindStruct, "Box"}]
			if box == nil {
				t.Fatal("no Box class node")
			}
			for _, f := range []string{"Box.instances", "Box.label", "Box.secret", "Box.#hidden", "Box.onChange"} {
				fn := byKey[key{goextract.KindField, f}]
				if fn == nil {
					continue
				}
				if !hasIntraEdge(result, box.Node.Id, fn.Node.Id, goextract.RefKindContains) {
					t.Errorf("no contains edge from Box to %s", f)
				}
				if fn.Node.FilePath != tc.path || fn.Node.StartLine == 0 || fn.Node.Language != languageForPath(tc.path) {
					t.Errorf("%s has no source location or language: %+v", f, fn.Node)
				}
			}
			if h := byKey[key{goextract.KindField, "Box.#hidden"}]; h != nil && h.Node.Visibility != "private" {
				t.Errorf("Box.#hidden visibility = %q, want private", h.Node.Visibility)
			}
		})
	}
}
