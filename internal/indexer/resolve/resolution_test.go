package resolve

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestResolutionIgnoresInsertionOrder checks design.md D3: the overload a
// call reaches is a function of the candidate set, not of the order the
// candidates arrived in. It runs the resolver over every order of three
// Java overloads of `add` for one call with a String argument.
func TestResolutionIgnoresInsertionOrder(t *testing.T) {
	param := func(name, typ string) *schema.Param { return &schema.Param{Name: name, Type: typ} }
	owner := &schema.Node{Id: "struct:Sink", Kind: goextract.KindStruct, Name: "Sink", FilePath: "Sink.java", Language: "java"}
	overloads := []*schema.Node{
		{Id: "method:add:int", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(int n)", Params: []*schema.Param{param("n", "int")}},
		{Id: "method:add:String", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(String s)", Params: []*schema.Param{param("s", "String")}},
		{Id: "method:add:String,int", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(String s, int n)", Params: []*schema.Param{param("s", "String"), param("n", "int")}},
	}
	call := goextract.UnresolvedRef{
		Kind: goextract.RefKindCalls, Name: "add", FromID: "method:use",
		ReceiverType: "Sink", ArgCount: 1, ArgTypes: []string{"String"},
	}
	file := goextract.FileResult{RelPath: "Use.java", ImportPath: "com.example", Language: "java"}

	var permute func(order []*schema.Node, rest []*schema.Node)
	permute = func(order, rest []*schema.Node) {
		if len(rest) == 0 {
			b := symbols.NewBuilder()
			b.AddNode(owner)
			b.Add(file.ImportPath, owner)
			for _, n := range order {
				b.Add(file.ImportPath, n)
				b.AddMember(owner.Id, n)
				b.AddNode(n)
			}
			rs := newTestResolver(b)
			out := rs.resolveCall(file, call, false)
			if out.target != "method:add:String" {
				t.Errorf("order %v: resolved to %q (set %v), want method:add:String", ids(order), out.target, out.set != nil)
			}
			return
		}
		for i, n := range rest {
			next := append(append([]*schema.Node(nil), rest[:i]...), rest[i+1:]...)
			permute(append(append([]*schema.Node(nil), order...), n), next)
		}
	}
	permute(nil, overloads)
}

func ids(nodes []*schema.Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Id
	}
	return out
}

// TestLibrarySupertypeNeverRejectsAnArgument checks canJudge: a parameter
// typed as a library interface, such as CharSequence, keeps its
// candidate for a String argument, because the relation between the two
// lives outside the index. A final parameter type still rejects.
func TestLibrarySupertypeNeverRejectsAnArgument(t *testing.T) {
	param := func(name, typ string) *schema.Param { return &schema.Param{Name: name, Type: typ} }
	owner := &schema.Node{Id: "struct:Sink", Kind: goextract.KindStruct, Name: "Sink", FilePath: "Sink.java", Language: "java"}
	overloads := []*schema.Node{
		{Id: "method:take:int", Kind: goextract.KindMethod, Name: "take", QualifiedName: "Sink.take", FilePath: "Sink.java", Language: "java",
			Signature: "(int n)", Params: []*schema.Param{param("n", "int")}},
		{Id: "method:take:CharSequence", Kind: goextract.KindMethod, Name: "take", QualifiedName: "Sink.take", FilePath: "Sink.java", Language: "java",
			Signature: "(CharSequence cs)", Params: []*schema.Param{param("cs", "CharSequence")}},
	}
	file := goextract.FileResult{RelPath: "Use.java", ImportPath: "com.example", Language: "java"}
	b := symbols.NewBuilder()
	b.AddNode(owner)
	b.Add(file.ImportPath, owner)
	for _, n := range overloads {
		b.Add(file.ImportPath, n)
		b.AddMember(owner.Id, n)
		b.AddNode(n)
	}
	rs := newTestResolver(b)
	call := goextract.UnresolvedRef{
		Kind: goextract.RefKindCalls, Name: "take", FromID: "method:use",
		ReceiverType: "Sink", ArgCount: 1, ArgTypes: []string{"String"},
	}
	out := rs.resolveCall(file, call, false)
	if out.target != "method:take:CharSequence" {
		t.Errorf("take(String) resolved to %q (set %v), want method:take:CharSequence", out.target, out.set != nil)
	}
	for _, c := range []struct {
		arg, param string
		want       bool
	}{
		{"String", "CharSequence", false},
		{"int", "Number", false},
		{"String", "int", true},
		{"Sink", "int", true},
		{"Sink", "Sink", true},
		{"Sink", "CharSequence", false},
	} {
		if got := rs.canJudge("java", c.arg, c.param); got != c.want {
			t.Errorf("canJudge(%s -> %s) = %v, want %v", c.arg, c.param, got, c.want)
		}
	}
}

// TestArityFits checks the argument-count rule for defaults, optional and
// variadic parameters, and for a spread argument.
func TestArityFits(t *testing.T) {
	p := func(name string) *schema.Param { return &schema.Param{Name: name} }
	withDefault := &schema.Param{Name: "d", HasDefault: true}
	optional := &schema.Param{Name: "o", Optional: true}
	variadic := &schema.Param{Name: "v", Variadic: true}
	cases := []struct {
		name   string
		params []*schema.Param
		argc   int32
		spread bool
		want   bool
	}{
		{"exact", []*schema.Param{p("a"), p("b")}, 2, false, true},
		{"too few", []*schema.Param{p("a"), p("b")}, 1, false, false},
		{"too many", []*schema.Param{p("a")}, 2, false, false},
		{"default left out", []*schema.Param{p("a"), withDefault}, 1, false, true},
		{"optional left out", []*schema.Param{p("a"), optional}, 1, false, true},
		{"variadic takes none", []*schema.Param{p("a"), variadic}, 1, false, true},
		{"variadic takes many", []*schema.Param{p("a"), variadic}, 5, false, true},
		{"spread meets the required count", []*schema.Param{p("a"), p("b"), p("c")}, 1, true, true},
		{"spread with too many fixed arguments", []*schema.Param{p("a")}, 3, true, false},
		{"spread into variadic", []*schema.Param{variadic}, 4, true, true},
	}
	for _, c := range cases {
		if got := arityFits(c.params, c.argc, c.spread); got != c.want {
			t.Errorf("%s: arityFits = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestUnqualifiedDropsTheQualifier checks that unqualified drops a Java
// package and a C++ namespace and keeps the type arguments.
func TestUnqualifiedDropsTheQualifier(t *testing.T) {
	if got := unqualified("java.util.List<String>"); got != "List<String>" {
		t.Errorf("unqualified(java.util.List<String>) = %q", got)
	}
	if got := unqualified("std::vector<int>"); got != "vector<int>" {
		t.Errorf("unqualified(std::vector<int>) = %q", got)
	}
}
