package indexer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// paramCase is one declaration in one language with the parameters and
// return type name its node must record, and the signature key those
// parameters give under the uniform rule.
type paramCase struct {
	language, file, src string
	name                string
	params              []*schema.Param
	returnTypeName      string
	key                 string
}

var paramCases = []paramCase{
	{"go", "f.go", "package p\n\nfunc f(a, b int, c ...string) (int, error) { return 0, nil }\n", "f",
		[]*schema.Param{{Name: "a", Type: "int"}, {Name: "b", Type: "int"}, {Name: "c", Type: "string", Variadic: true}}, "(int, error)", "(int,int,string...)"},
	{"typescript", "f.ts", "export function f(a: number, b?: string, c = 1, ...rest: number[]): Promise<void> {}\n", "f",
		[]*schema.Param{{Name: "a", Type: "number"}, {Name: "b", Type: "string", Optional: true}, {Name: "c", HasDefault: true}, {Name: "rest", Type: "number[]", Variadic: true}}, "Promise<void>", "(number,string?,_?,number[]...)"},
	{"javascript", "f.js", "export function f(a, b = 1, ...rest) {}\n", "f",
		[]*schema.Param{{Name: "a"}, {Name: "b", HasDefault: true}, {Name: "rest", Variadic: true}}, "", "(_,_?,_...)"},
	{"python", "f.py", "def f(a, b: int, c=1, d: str = 'x', *args, e, **kw) -> int:\n    pass\n", "f",
		[]*schema.Param{{Name: "a"}, {Name: "b", Type: "int"}, {Name: "c", HasDefault: true}, {Name: "d", Type: "str", HasDefault: true}, {Name: "args", Variadic: true}, {Name: "e"}, {Name: "kw", Variadic: true}}, "int", "(_,int,_?,str?,_...,_,_...)"},
	{"java", "C.java", "class C { int f(int a, final String b, Object... rest) { return 0; } }\n", "f",
		[]*schema.Param{{Name: "a", Type: "int"}, {Name: "b", Type: "String"}, {Name: "rest", Type: "Object", Variadic: true}}, "int", "(int,String,Object...)"},
	{"csharp", "C.cs", "class C { int F(int a, string b = \"x\", params object[] rest) { return 0; } }\n", "F",
		[]*schema.Param{{Name: "a", Type: "int"}, {Name: "b", Type: "string", HasDefault: true}, {Name: "rest", Type: "object[]", Variadic: true}}, "int", "(int,string?,object[]...)"},
	{"kotlin", "f.kt", "fun f(a: Int, b: String = \"x\", vararg rest: Int): Int = 0\n", "f",
		[]*schema.Param{{Name: "a", Type: "Int"}, {Name: "b", Type: "String", HasDefault: true}, {Name: "rest", Type: "Int", Variadic: true}}, "Int", "(Int,String?,Int...)"},
	{"swift", "f.swift", "func move(to point: Point, by delta: Vector = .zero, _ xs: Int...) -> Bool { return true }\n", "move",
		[]*schema.Param{{Name: "point", Type: "Point"}, {Name: "delta", Type: "Vector", HasDefault: true}, {Name: "xs", Type: "Int", Variadic: true}}, "Bool", "(Point,Vector?,Int...)"},
	{"rust", "f.rs", "fn f(a: i32, b: &str) -> i32 { 0 }\n", "f",
		[]*schema.Param{{Name: "a", Type: "i32"}, {Name: "b", Type: "&str"}}, "i32", "(i32,&str)"},
	{"ruby", "f.rb", "def f(a, b = 1, *rest, c:, d: 2, **kw, &blk)\nend\n", "f",
		[]*schema.Param{{Name: "a"}, {Name: "b", HasDefault: true}, {Name: "rest", Variadic: true}, {Name: "c"}, {Name: "d", HasDefault: true}, {Name: "kw", Variadic: true}, {Name: "blk"}}, "", "(_,_?,_...,_,_?,_...,_)"},
	{"php", "f.php", "<?php\nfunction f(int $a, ?string $b = null, string ...$rest): int { return 0; }\n", "f",
		[]*schema.Param{{Name: "$a", Type: "int"}, {Name: "$b", Type: "?string", HasDefault: true}, {Name: "$rest", Type: "string", Variadic: true}}, "int", "(int,?string?,string...)"},
	{"c", "f.c", "int f(int a, const char *b, ...) { return 0; }\n", "f",
		[]*schema.Param{{Name: "a", Type: "int"}, {Name: "b", Type: "const char *"}, {Variadic: true}}, "int", "(int,const char *,_...)"},
}

// TestParamsAreRecordedPerLanguage extracts one declaration per language
// and checks the structured parameters, the return type name and the
// signature key the parameters give. The key rule is the same for every
// language (design.md D1 and D13).
func TestParamsAreRecordedPerLanguage(t *testing.T) {
	for _, c := range paramCases {
		t.Run(c.language, func(t *testing.T) {
			spec, ok := lookupLanguageByID(c.language)
			if !ok {
				t.Fatalf("no language %s", c.language)
			}
			p, err := spec.NewParser()
			if err != nil {
				t.Fatalf("NewParser: %v", err)
			}
			defer p.Close()
			res, err := spec.Extract(p, "mod", c.file, []byte(c.src))
			if err != nil || res.Err != nil {
				t.Fatalf("Extract: %v / %v", err, res.Err)
			}
			var node *schema.Node
			for _, en := range res.Nodes {
				if en.Node.Name == c.name && (en.Node.Kind == goextract.KindFunction || en.Node.Kind == goextract.KindMethod) {
					node = en.Node
				}
			}
			if node == nil {
				t.Fatalf("no function node named %s", c.name)
			}
			if diff := cmp.Diff(c.params, node.Params, cmpopts.IgnoreUnexported(schema.Param{})); diff != "" {
				t.Errorf("params differ (-want +got):\n%s", diff)
			}
			if node.ReturnTypeName != c.returnTypeName {
				t.Errorf("ReturnTypeName = %q, want %q", node.ReturnTypeName, c.returnTypeName)
			}
			if node.Arity != int32(len(c.params)) {
				t.Errorf("Arity = %d, want %d", node.Arity, len(c.params))
			}
			var params []goextract.Param
			for _, p := range node.Params {
				params = append(params, goextract.Param{Name: p.Name, Type: p.Type, Optional: p.Optional, HasDefault: p.HasDefault, Variadic: p.Variadic})
			}
			if key := goextract.SignatureKeyOf(params); key != c.key {
				t.Errorf("signature key = %q, want %q", key, c.key)
			}
			if c.language == "swift" && !strings.HasSuffix(node.QualifiedName, "move(to:by:_:)") {
				t.Errorf("Swift QualifiedName = %q, want the labels move(to:by:_:)", node.QualifiedName)
			}
		})
	}
}
