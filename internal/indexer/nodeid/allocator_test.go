package nodeid_test

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

const testFile = "web/tests/live.test.ts"

// TestSignatureKeyIsUniformAcrossLanguages checks that parameter lists
// from five languages key under one rule: types as written, `_` for an
// untyped parameter, `?` for an optional one, `...` for a variadic one,
// and no parameter names.
func TestSignatureKeyIsUniformAcrossLanguages(t *testing.T) {
	cases := []struct {
		name   string
		params []nodeid.ParamKey
		want   string
	}{
		{"Python def f(a, b)", []nodeid.ParamKey{{}, {}}, "(_,_)"},
		{"Python def f(a: int, b=1)", []nodeid.ParamKey{{Type: "int"}, {Optional: true}}, "(int,_?)"},
		{"Go func f(a, b int)", []nodeid.ParamKey{{Type: "int"}, {Type: "int"}}, "(int,int)"},
		{"Java foo(int n, String... parts)", []nodeid.ParamKey{{Type: "int"}, {Type: "String", Variadic: true}}, "(int,String...)"},
		{"Swift func move(to point: Point)", []nodeid.ParamKey{{Type: "Point"}}, "(Point)"},
		{"Kotlin fun f(x: Int = 0)", []nodeid.ParamKey{{Type: "Int", Optional: true}}, "(Int?)"},
		{"shell function deploy", nil, "()"},
		{"C# Write(params object[] args)", []nodeid.ParamKey{{Type: "object[]", Variadic: true}}, "(object[]...)"},
		{"spaces in a type", []nodeid.ParamKey{{Type: "Map<String,  Integer>"}}, "(Map<String, Integer>)"},
	}
	for _, c := range cases {
		if got := nodeid.SignatureKey(c.params); got != c.want {
			t.Errorf("%s: SignatureKey = %q, want %q", c.name, got, c.want)
		}
	}
}

// finalIDs declares one occurrence of a function per (start byte, key),
// in the order given, and returns the final id of each start byte.
func finalIDs(t *testing.T, decls map[uint]string) map[uint]string {
	t.Helper()
	a := nodeid.NewAllocator(testFile)
	var r goextract.FileResult
	byStart := make(map[uint]*schema.Node)
	for s, key := range decls {
		n := &schema.Node{Id: a.Declare(goextract.KindFunction, "foo", key, s)}
		byStart[s] = n
		r.Nodes = append(r.Nodes, goextract.ExtractedNode{Node: n})
	}
	if err := a.Finalize(&r); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	out := make(map[uint]string, len(byStart))
	for s, n := range byStart {
		out[s] = n.Id
	}
	return out
}

// TestFinalizeKeysBySignature checks that two declarations with different
// keys get different ids in any order, and that three with one key get
// DeclID plus two ordinals by source position. A second run gives the
// same ids.
func TestFinalizeKeysBySignature(t *testing.T) {
	ids := finalIDs(t, map[uint]string{300: "()", 100: "(int)", 200: "(int)", 50: "()", 400: "()"})

	if want := nodeid.DeclID(goextract.KindFunction, "foo", testFile, "(int)"); ids[100] != want {
		t.Errorf("foo(int) at byte 100 = %s, want DeclID %s", ids[100], want)
	}
	if want := nodeid.OrdinalID(goextract.KindFunction, "foo", testFile, "(int)", 2); ids[200] != want {
		t.Errorf("foo(int) at byte 200 = %s, want ordinal 2 %s", ids[200], want)
	}
	if want := nodeid.DeclID(goextract.KindFunction, "foo", testFile, "()"); ids[50] != want {
		t.Errorf("foo() at byte 50 = %s, want DeclID %s", ids[50], want)
	}
	if want := nodeid.OrdinalID(goextract.KindFunction, "foo", testFile, "()", 2); ids[300] != want {
		t.Errorf("foo() at byte 300 = %s, want ordinal 2 %s", ids[300], want)
	}
	if want := nodeid.OrdinalID(goextract.KindFunction, "foo", testFile, "()", 3); ids[400] != want {
		t.Errorf("foo() at byte 400 = %s, want ordinal 3 %s", ids[400], want)
	}
	seen := make(map[string]uint)
	for s, id := range ids {
		if other, dup := seen[id]; dup {
			t.Errorf("bytes %d and %d share id %s", s, other, id)
		}
		seen[id] = s
	}

	// Removing the first foo(int) must not change the ids of any foo().
	again := finalIDs(t, map[uint]string{300: "()", 200: "(int)", 50: "()", 400: "()"})
	for _, s := range []uint{50, 300, 400} {
		if again[s] != ids[s] {
			t.Errorf("foo() at byte %d changed from %s to %s when an overload was removed", s, ids[s], again[s])
		}
	}
	if again[200] != nodeid.DeclID(goextract.KindFunction, "foo", testFile, "(int)") {
		t.Errorf("the remaining foo(int) = %s, want DeclID", again[200])
	}
}

// TestDeclIDWithoutSignatureIsNodeID checks that a declaration without a
// signature key, such as a type, keeps the id it had before signature
// keys existed.
func TestDeclIDWithoutSignatureIsNodeID(t *testing.T) {
	if got, want := nodeid.DeclID(goextract.KindStruct, "Model", testFile, ""), nodeid.NodeID(goextract.KindStruct, "Model", testFile); got != want {
		t.Fatalf("DeclID with an empty key = %s, want NodeID %s", got, want)
	}
	if nodeid.DeclID(goextract.KindFunction, "foo", testFile, "()") == nodeid.NodeID(goextract.KindFunction, "foo", testFile) {
		t.Fatal("DeclID with the key \"()\" equals NodeID, so a function without parameters would collide with a keyless declaration")
	}
}

// TestFinalizeRewritesEveryIDField fills each field of a FileResult that
// holds a node id with provisional ids, and checks that Finalize leaves
// only final ids. A provisional id in a field that RewriteIDs does not
// know makes Finalize fail.
func TestFinalizeRewritesEveryIDField(t *testing.T) {
	build := func() (*nodeid.Allocator, *goextract.FileResult, string, string) {
		a := nodeid.NewAllocator(testFile)
		iface := a.Declare(goextract.KindInterface, "Shape", "", 10)
		method := a.Declare(goextract.KindMethod, "Shape.Area", "(int)", 50)
		r := &goextract.FileResult{
			Nodes: []goextract.ExtractedNode{
				{Node: &schema.Node{Id: iface}},
				{Node: &schema.Node{Id: method}},
			},
			IntraEdges: []goextract.IntraEdge{
				{Edge: &schema.Edge{Source: iface, Target: method, Kind: goextract.RefKindContains}},
			},
			Unresolved: []goextract.UnresolvedRef{
				{FromID: method, Name: "helper", Kind: goextract.RefKindCalls},
			},
			InterfaceMethods: map[string][]goextract.MethodSpec{iface: {{Name: "Area", Arity: 1}}},
			MethodArity:      map[string]int32{method: 1},
		}
		return a, r, iface, method
	}

	a, r, _, _ := build()
	if err := a.Finalize(r); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	iface := nodeid.DeclID(goextract.KindInterface, "Shape", testFile, "")
	method := nodeid.DeclID(goextract.KindMethod, "Shape.Area", testFile, "(int)")
	if r.Nodes[0].Node.Id != iface || r.Nodes[1].Node.Id != method {
		t.Errorf("node ids = %s, %s; want %s, %s", r.Nodes[0].Node.Id, r.Nodes[1].Node.Id, iface, method)
	}
	if e := r.IntraEdges[0].Edge; e.Source != iface || e.Target != method {
		t.Errorf("edge = %s -> %s, want %s -> %s", e.Source, e.Target, iface, method)
	}
	if r.Unresolved[0].FromID != method {
		t.Errorf("Unresolved[0].FromID = %s, want %s", r.Unresolved[0].FromID, method)
	}
	if _, ok := r.InterfaceMethods[iface]; !ok || len(r.InterfaceMethods) != 1 {
		t.Errorf("InterfaceMethods keys = %v, want only %s", r.InterfaceMethods, iface)
	}
	if r.MethodArity[method] != 1 || len(r.MethodArity) != 1 {
		t.Errorf("MethodArity = %v, want %s: 1", r.MethodArity, method)
	}

	a, r, _, prov := build()
	r.Nodes[1].Node.Signature = prov
	err := a.Finalize(r)
	if err == nil || !strings.Contains(err.Error(), "Signature") {
		t.Fatalf("Finalize with a provisional id in Node.Signature: err = %v, want an error that names Signature", err)
	}
}
