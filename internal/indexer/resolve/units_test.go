package resolve

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

func contains(source, target string) *schema.Edge {
	return &schema.Edge{Source: source, Target: target, Kind: goextract.RefKindContains}
}

func triple(e *schema.Edge) string {
	return e.Source + " -" + e.Kind + "-> " + e.Target
}

func sortedTriples(edges []*schema.Edge) []string {
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = triple(e)
	}
	sort.Strings(out)
	return slices.Compact(out)
}

// allGlobal returns one category of the whole-graph edges of every unit.
func allGlobal(g *Global, pick func(*globalUnit) []*schema.Edge) []*schema.Edge {
	var out []*schema.Edge
	for _, u := range g.units {
		out = append(out, pick(u)...)
	}
	return out
}

// referenceModel computes the retried calls and the overrides with the
// four rules of the single global pass, each written out here so that the
// test does not share the tables it checks:
//
//  1. The last contains edge of any source owns a member.
//  2. The last id wins for a name under one source.
//  3. Every member of kind method takes part in the overrides.
//  4. A synthesized implements edge is a supertype edge, for the overrides
//     and for the retry. withOverlay switches this rule off, so the test
//     can tell which results depend on it.
func referenceModel(view symbols.View, edges []*schema.Edge, pending []goextract.UnresolvedRef, withOverlay bool) (retried, overrides []string) {
	owner := make(map[string]string)
	lastByName := make(map[string]map[string]string)
	methodsByName := make(map[string]map[string][]string)
	supertypes := make(map[string][]string)
	structMethods := make(dispatch.TypeMethods)
	for _, e := range edges {
		switch e.Kind {
		case goextract.RefKindContains:
			owner[e.Target] = e.Source
			if lastByName[e.Source] == nil {
				lastByName[e.Source] = make(map[string]string)
				methodsByName[e.Source] = make(map[string][]string)
			}
			name := view.Name(e.Target)
			lastByName[e.Source][name] = e.Target
			if view.Kind(e.Target) == goextract.KindMethod {
				methodsByName[e.Source][name] = append(methodsByName[e.Source][name], e.Target)
				if view.Kind(e.Source) == goextract.KindStruct {
					structMethods[e.Source] = append(structMethods[e.Source], goextract.MethodSpec{Name: name, Arity: view.Arity(e.Target)})
				}
			}
		default:
			supertypes[e.Source] = append(supertypes[e.Source], e.Target)
		}
	}
	if withOverlay {
		for _, e := range dispatch.SynthesizeImplements(structMethods, view.InterfaceMethods(), nil) {
			supertypes[e.Source] = append(supertypes[e.Source], e.Target)
		}
	}

	for _, ref := range pending {
		ownerType, ok := owner[ref.FromID]
		if !ok {
			continue
		}
		symbols.WalkSupertypes(ownerType, supertypes, func(cur string) bool {
			id, found := lastByName[cur][ref.Name]
			if found {
				retried = append(retried, ref.FromID+" -calls-> "+id)
			}
			return found
		})
	}
	// The pass sorts the methods of each name before it pairs them, so the
	// first candidate with a matching signature key is the lowest id.
	for _, byName := range methodsByName {
		for _, ids := range byName {
			sort.Strings(ids)
		}
	}
	for typeID, byName := range methodsByName {
		for name, ids := range byName {
			for _, id := range ids {
				key := goextract.SignatureKeyOfNode(view.Node(id))
				symbols.WalkSupertypes(typeID, supertypes, func(cur string) bool {
					var byArity []string
					for _, cand := range methodsByName[cur][name] {
						if key != "" && goextract.SignatureKeyOfNode(view.Node(cand)) == key {
							overrides = append(overrides, id+" -overrides-> "+cand)
							return true
						}
						if view.Arity(cand) == view.Arity(id) {
							byArity = append(byArity, cand)
						}
					}
					if len(byArity) == 1 {
						overrides = append(overrides, id+" -overrides-> "+byArity[0])
						return true
					}
					return false
				})
			}
		}
	}
	sort.Strings(retried)
	sort.Strings(overrides)
	return slices.Compact(retried), slices.Compact(overrides)
}

// TestConformanceIndexMatchesHelpers draws lists of structural edges and
// pending calls, and compares the retried calls and the overrides of the
// one shared index with the reference model above.
func TestConformanceIndexMatchesHelpers(t *testing.T) {
	names := []string{"a", "b", "c"}
	var overlayOverrides, overlayRetries int
	rapid.Check(t, func(t *rapid.T) {
		b := symbols.NewBuilder()
		// Sources: a file node and four types, some of them interfaces with
		// method specs, so that a struct can satisfy one.
		sources := []string{"file"}
		b.AddNode(&schema.Node{Id: "file", Kind: goextract.KindFile, Name: "f.go"})
		var types []string
		for i := range 4 {
			id := "T" + strconv.Itoa(i)
			n := &schema.Node{Id: id, Kind: goextract.KindStruct, Name: id}
			if rapid.Bool().Draw(t, "interface") {
				n.Kind = goextract.KindInterface
				for range rapid.IntRange(1, 2).Draw(t, "specs") {
					n.InterfaceMethods = append(n.InterfaceMethods, &schema.MethodSpec{
						Name: rapid.SampledFrom(names).Draw(t, "specName"), Arity: rapid.Int32Range(0, 1).Draw(t, "specArity"),
					})
				}
			}
			b.AddNode(n)
			types = append(types, id)
			sources = append(sources, id)
		}
		// Members: methods and functions with few names, so that names
		// repeat and overloads exist.
		var members []string
		for i := range rapid.IntRange(1, 10).Draw(t, "members") {
			id := "m" + strconv.Itoa(i)
			kind := goextract.KindMethod
			if rapid.IntRange(0, 4).Draw(t, "function") == 0 {
				kind = goextract.KindFunction
			}
			b.AddNode(&schema.Node{Id: id, Kind: kind, Name: rapid.SampledFrom(names).Draw(t, "memberName")})
			b.SetArity(id, rapid.Int32Range(0, 1).Draw(t, "arity"))
			members = append(members, id)
		}
		// In half of the draws, one more struct satisfies one more
		// interface, and only that synthesized implements edge leads to
		// an override and to a retried call. Random draws alone reach the
		// overlay too seldom for the positive assertion below.
		gadget := rapid.Bool().Draw(t, "gadget")
		if gadget {
			b.AddNode(&schema.Node{Id: "G", Kind: goextract.KindStruct, Name: "G"})
			b.AddNode(&schema.Node{Id: "GI", Kind: goextract.KindInterface, Name: "GI", InterfaceMethods: []*schema.MethodSpec{{Name: "g"}}})
			b.AddNode(&schema.Node{Id: "G.g", Kind: goextract.KindMethod, Name: "g"})
			b.AddNode(&schema.Node{Id: "G.x", Kind: goextract.KindMethod, Name: "x"})
			b.AddNode(&schema.Node{Id: "GI.x", Kind: goextract.KindMethod, Name: "x"})
		}
		view := b.Freeze()

		var edges []*schema.Edge
		var pending []goextract.UnresolvedRef
		if gadget {
			edges = append(edges, contains("G", "G.g"), contains("G", "G.x"), contains("GI", "GI.x"))
			pending = append(pending, goextract.UnresolvedRef{Kind: goextract.RefKindCalls, FromID: "G.g", Name: "x"})
		}
		for range rapid.IntRange(1, 16).Draw(t, "edges") {
			if rapid.IntRange(0, 2).Draw(t, "supertype") == 0 {
				kind := rapid.SampledFrom([]string{goextract.RefKindEmbeds, goextract.EdgeKindExtends, goextract.EdgeKindImplements}).Draw(t, "kind")
				// An embeds edge between two interfaces changes the method
				// set of an interface. The model keeps that out, so the
				// draw turns it into an extends edge.
				src, dst := rapid.SampledFrom(types).Draw(t, "src"), rapid.SampledFrom(types).Draw(t, "dst")
				if kind == goextract.RefKindEmbeds && view.Kind(src) == goextract.KindInterface && view.Kind(dst) == goextract.KindInterface {
					kind = goextract.EdgeKindExtends
				}
				edges = append(edges, &schema.Edge{Source: src, Target: dst, Kind: kind})
				continue
			}
			// A member can get a contains edge from several sources, and
			// from a file, so the owner rule has something to decide.
			edges = append(edges, contains(rapid.SampledFrom(sources).Draw(t, "owner"), rapid.SampledFrom(members).Draw(t, "member")))
		}
		for range rapid.IntRange(0, 6).Draw(t, "pending") {
			pending = append(pending, goextract.UnresolvedRef{
				Kind: goextract.RefKindCalls, FromID: rapid.SampledFrom(members).Draw(t, "from"), Name: rapid.SampledFrom(names).Draw(t, "callName"),
			})
		}

		g := NewGlobal(view, symbols.NewHierarchy(view, nil), edges)
		var gotRetried []string
		for _, ref := range pending {
			if id, ok := g.retry(ref); ok {
				gotRetried = append(gotRetried, ref.FromID+" -calls-> "+id)
			}
		}
		sort.Strings(gotRetried)
		gotRetried = slices.Compact(gotRetried)
		gotOverrides := sortedTriples(allGlobal(g, func(u *globalUnit) []*schema.Edge { return u.overrides }))

		wantRetried, wantOverrides := referenceModel(view, edges, pending, true)
		if diff := cmp.Diff(wantRetried, gotRetried, cmpopts.EquateEmpty()); diff != "" {
			t.Fatalf("retried calls differ from the reference model (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantOverrides, gotOverrides, cmpopts.EquateEmpty()); diff != "" {
			t.Fatalf("overrides differ from the reference model (-want +got):\n%s", diff)
		}

		bareRetried, bareOverrides := referenceModel(view, edges, pending, false)
		if len(wantRetried) > len(bareRetried) {
			overlayRetries++
		}
		if len(wantOverrides) > len(bareOverrides) {
			overlayOverrides++
		}
	})
	// Positive assertion: the draws reached the overlay, so the property
	// above proved the fourth rule and not only the first three.
	if overlayRetries == 0 || overlayOverrides == 0 {
		t.Errorf("%d draws had a retried call and %d an override that only a synthesized implements edge gives; want at least one of each", overlayRetries, overlayOverrides)
	}
}

// barrierView returns a view over nodes, with arity 0 for every method.
func barrierView(decls []symbols.Declarations, stored func(*symbols.Builder)) symbols.View {
	b := symbols.NewBuilder()
	b.AddResults(decls)
	if stored != nil {
		stored(b)
	}
	return b.Freeze()
}

func node(id, kind, name, file string) *schema.Node {
	return &schema.Node{Id: id, Kind: kind, Name: name, QualifiedName: id, FilePath: file, Language: "java"}
}

func TestBarrierTwo(t *testing.T) {
	t.Run("a Go struct that satisfies an interface", func(t *testing.T) {
		iface := node("I", goextract.KindInterface, "I", "i.go")
		iface.InterfaceMethods = []*schema.MethodSpec{{Name: "Read"}}
		view := barrierView([]symbols.Declarations{
			{RelPath: "i.go", Nodes: []*schema.Node{iface}},
			{RelPath: "s.go", Nodes: []*schema.Node{node("S", goextract.KindStruct, "S", "s.go"), node("S.Read", goextract.KindMethod, "Read", "s.go")}},
		}, nil)
		g := NewGlobal(view, symbols.NewHierarchy(view, nil), []*schema.Edge{contains("S", "S.Read")})

		got := sortedTriples(g.units["s.go"].implements)
		if want := []string{"S -implements-> I"}; !slices.Equal(got, want) {
			t.Errorf("implements edges of the unit s.go = %v, want %v", got, want)
		}
		if len(g.units) != 1 {
			t.Errorf("%d units hold whole-graph edges, want only s.go", len(g.units))
		}
	})

	t.Run("a method with the name of a supertype method", func(t *testing.T) {
		view := barrierView([]symbols.Declarations{{RelPath: "a.java", Nodes: []*schema.Node{
			node("Base", goextract.KindStruct, "Base", "a.java"), node("Base.run", goextract.KindMethod, "run", "a.java"),
			node("Sub", goextract.KindStruct, "Sub", "a.java"), node("Sub.run", goextract.KindMethod, "run", "a.java"),
		}}}, nil)
		edges := []*schema.Edge{contains("Base", "Base.run"), contains("Sub", "Sub.run"), {Source: "Sub", Target: "Base", Kind: goextract.EdgeKindExtends}}
		g := NewGlobal(view, symbols.NewHierarchy(view, edges), edges)

		got := sortedTriples(g.units["a.java"].overrides)
		if want := []string{"Sub.run -overrides-> Base.run"}; !slices.Equal(got, want) {
			t.Errorf("override edges = %v, want %v", got, want)
		}
	})

	t.Run("an overload set whose members span two files", func(t *testing.T) {
		owner := node("Store", goextract.KindStruct, "Store", "")
		owner.Language = "csharp"
		put1, put2 := node("Store.Put/int", goextract.KindMethod, "Put", "A.cs"), node("Store.Put/string", goextract.KindMethod, "Put", "B.cs")
		put1.Language, put2.Language = "csharp", "csharp"
		put1.Params = []*schema.Param{{Type: "int"}}
		put2.Params = []*schema.Param{{Type: "string"}}
		edges := []*schema.Edge{contains("Store", put1.Id), contains("Store", put2.Id)}
		view := barrierView([]symbols.Declarations{
			{RelPath: "A.cs", Language: "csharp", Nodes: []*schema.Node{owner, put1}, Contains: edges[:1]},
			{RelPath: "B.cs", Language: "csharp", Nodes: []*schema.Node{owner, put2}, Contains: edges[1:]},
		}, nil)
		h := symbols.NewHierarchy(view, edges)
		sets := h.SetNodes()
		if len(sets) != 1 {
			t.Fatalf("the hierarchy declares %d overload sets, want 1", len(sets))
		}
		setID := sets[0].Id
		g := NewGlobal(view, h, edges)

		// The owner-to-set edge has the set as its owner, and the set spans
		// two files, so it is in the no-file unit. Each set-to-member edge
		// is in the unit of its member's file.
		for path, want := range map[string][]string{
			"":     {"Store -contains-> " + setID},
			"A.cs": {setID + " -contains-> " + put1.Id},
			"B.cs": {setID + " -contains-> " + put2.Id},
		} {
			u := g.units[path]
			if u == nil {
				t.Fatalf("the unit %q holds no whole-graph edge, want %v", path, want)
			}
			if got := sortedTriples(u.sets); !slices.Equal(got, want) {
				t.Errorf("set edges of the unit %q = %v, want %v", path, got, want)
			}
		}
		for _, m := range []string{put1.Id, put2.Id} {
			if _, ok := g.replaced[memberPair{"Store", m}]; !ok {
				t.Errorf("the pair (Store, %s) is not among the replaced member pairs", m)
			}
		}
	})

	t.Run("a pending call that resolves to a method in another file", func(t *testing.T) {
		view := barrierView([]symbols.Declarations{
			{RelPath: "base.go", Nodes: []*schema.Node{node("Base", goextract.KindStruct, "Base", "base.go"), node("Base.Hello", goextract.KindMethod, "Hello", "base.go")}},
			{RelPath: "sub.go", Nodes: []*schema.Node{node("Sub", goextract.KindStruct, "Sub", "sub.go")}},
			{RelPath: "run.go", Nodes: []*schema.Node{node("Sub.Run", goextract.KindMethod, "Run", "run.go")}},
		}, nil)
		edges := []*schema.Edge{
			contains("Base", "Base.Hello"),
			{Source: "Sub", Target: "Base", Kind: goextract.EdgeKindExtends},
			// The contains edge that the resolution of run.go makes: the
			// method is in another file than its type.
			contains("Sub", "Sub.Run"),
		}
		g := NewGlobal(view, symbols.NewHierarchy(view, edges), edges)

		got, ok := g.retry(goextract.UnresolvedRef{Kind: goextract.RefKindCalls, FromID: "Sub.Run", Name: "Hello"})
		if !ok || got != "Base.Hello" {
			t.Errorf("retry = %q, %v; want Base.Hello", got, ok)
		}
		if _, ok := g.retry(goextract.UnresolvedRef{Kind: goextract.RefKindCalls, FromID: "Sub.Run", Name: "Missing"}); ok {
			t.Errorf("retry resolved a name that no supertype declares")
		}
	})

	t.Run("a stored set that no result declares", func(t *testing.T) {
		view := barrierView(nil, func(b *symbols.Builder) {
			b.AddNode(node("Store", goextract.KindStruct, "Store", ""))
			b.AddNode(node("Store.Put/int", goextract.KindMethod, "Put", "A.cs"))
			b.AddNode(node("Store.Put/string", goextract.KindMethod, "Put", "B.cs"))
			b.AddStoredSet("set:Put", symbols.StoredSet{Owner: "Store", Members: []string{"Store.Put/int", "Store.Put/string"}})
		})
		g := NewGlobal(view, symbols.NewHierarchy(view, nil), nil)

		for path, member := range map[string]string{"A.cs": "Store.Put/int", "B.cs": "Store.Put/string"} {
			u := g.units[path]
			if u == nil {
				t.Fatalf("the unit %q holds no whole-graph edge", path)
			}
			if got, want := sortedTriples(u.sets), []string{"Store -contains-> " + member}; !slices.Equal(got, want) {
				t.Errorf("dissolution edges of the unit %q = %v, want %v", path, got, want)
			}
		}
		rest := sortedTriples(g.Rest(func(string) bool { return false }))
		if want := []string{"Store -contains-> Store.Put/int", "Store -contains-> Store.Put/string"}; !slices.Equal(rest, want) {
			t.Errorf("Rest = %v, want %v", rest, want)
		}
	})
}

func TestAssemblyRejectsForeignEdge(t *testing.T) {
	view := barrierView([]symbols.Declarations{
		{RelPath: "a.go", Nodes: []*schema.Node{node("A", goextract.KindFunction, "A", "a.go")}},
		{RelPath: "b.go", Nodes: []*schema.Node{node("B", goextract.KindFunction, "B", "b.go")}},
		{RelPath: "", Nodes: []*schema.Node{node("Shared", goextract.KindStruct, "Shared", ""), node("R", goextract.KindRoute, "GET /x", "a.go")}},
	}, nil)
	g := NewGlobal(view, symbols.NewHierarchy(view, nil), nil)
	file := func() *schema.File { return &schema.File{Path: "a.go"} }
	calls := func(source string) *schema.Edge { return &schema.Edge{Source: source, Target: "B", Kind: "calls"} }

	for _, tc := range []struct {
		name       string
		t          TypeOutput
		wantErr    bool
		wantEdges  []string
		wantNoFile []string
	}{
		{name: "an edge that another file owns", t: TypeOutput{file: file(), edges: []*schema.Edge{calls("B")}}, wantErr: true},
		{name: "an edge with the owner path \"\"", t: TypeOutput{file: file(), edges: []*schema.Edge{calls("Shared")}}, wantNoFile: []string{"Shared -calls-> B"}},
		{name: "an edge that the file owns", t: TypeOutput{file: file(), edges: []*schema.Edge{calls("A")}}, wantEdges: []string{"A -calls-> B"}},
		{name: "a file-unit edge whose source is a route", t: TypeOutput{file: file(), edges: []*schema.Edge{calls("R")}}, wantErr: true},
		{name: "a route-unit edge whose source is a function", t: TypeOutput{edges: []*schema.Edge{calls("A")}}, wantErr: true},
		{name: "a route-unit edge whose source is a route", t: TypeOutput{edges: []*schema.Edge{calls("R")}}, wantEdges: []string{"R -calls-> B"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			relPath := ""
			if tc.t.file != nil {
				relPath = tc.t.file.Path
			}
			u, err := g.Assemble(tc.t, CallOutput{relPath: relPath})
			if tc.wantErr {
				if !errors.Is(err, goextract.ErrInternal) {
					t.Fatalf("Assemble error = %v, want one that wraps goextract.ErrInternal", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}
			if got := sortedTriples(u.Edges); !slices.Equal(got, tc.wantEdges) {
				t.Errorf("unit edges = %v, want %v", got, tc.wantEdges)
			}
			if got := sortedTriples(u.NoFile.types); !slices.Equal(got, tc.wantNoFile) {
				t.Errorf("no-file edges = %v, want %v", got, tc.wantNoFile)
			}
		})
	}
}

// TestCollapseByUnitProperty draws edges over nodes with drawn file paths,
// splits them into units by the owner path, and checks that the union of
// the collapse of each unit equals one collapse of the whole list.
func TestCollapseByUnitProperty(t *testing.T) {
	kinds := []string{"calls", goextract.RefKindContains, goextract.EdgeKindExtends}
	rapid.Check(t, func(t *rapid.T) {
		paths := make(map[string]string)
		var ids []string
		for i := range rapid.IntRange(2, 6).Draw(t, "nodes") {
			id := "n" + strconv.Itoa(i)
			ids = append(ids, id)
			paths[id] = rapid.SampledFrom([]string{"", "a.go", "b.go", "c.go"}).Draw(t, "path")
		}
		var edges []*schema.Edge
		units := make(map[string][]*schema.Edge)
		for i := range rapid.IntRange(0, 30).Draw(t, "edges") {
			e := &schema.Edge{
				Source: rapid.SampledFrom(ids).Draw(t, "source"), Target: rapid.SampledFrom(ids).Draw(t, "target"),
				Kind: rapid.SampledFrom(kinds).Draw(t, "kind"),
				// Few positions, so that candidates tie on line and column.
				Line: rapid.Int32Range(1, 2).Draw(t, "line"), Col: rapid.Int32Range(0, 1).Draw(t, "col"),
				// A distinct value for each candidate: a different pick
				// among two that tie then shows in the comparison.
				Provenance: fmt.Sprintf("candidate-%d", i),
			}
			edges = append(edges, e)
			unit := paths[EdgeOwnerID(e)]
			units[unit] = append(units[unit], e)
		}

		var byUnit []*schema.Edge
		for _, list := range units {
			byUnit = append(byUnit, CollapseEdges(list)...)
		}
		sort.Slice(byUnit, func(i, j int) bool { return triple(byUnit[i]) < triple(byUnit[j]) })
		whole := CollapseEdges(edges)
		sort.Slice(whole, func(i, j int) bool { return triple(whole[i]) < triple(whole[j]) })

		if diff := cmp.Diff(whole, byUnit, protocmp.Transform(), cmpopts.EquateEmpty()); diff != "" {
			t.Fatalf("the collapse of each unit differs from one collapse of the whole list (-whole +units):\n%s", diff)
		}
	})
}
