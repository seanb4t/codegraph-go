package nodeid_test

import (
	"reflect"
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestFinalizeLeavesNoProvisionalID places a provisional id in a random
// string field of a result. Finalize must either rewrite the field to a
// final id or fail with an error that names the path of the field, so no
// provisional id passes in silence. Map keys and values are not settable
// through reflect; TestFinalizeRewritesEveryIDField covers them.
func TestFinalizeLeavesNoProvisionalID(t *testing.T) {
	var rewritten, reported int
	rapid.Check(t, func(t *rapid.T) {
		a := nodeid.NewAllocator(testFile)
		owner := a.Declare(goextract.KindStruct, "Shape", "", 10)
		r := &goextract.FileResult{ImportPath: "shapes", RelPath: testFile, Language: "go", ContentHash: "hash"}
		r.Nodes = append(r.Nodes, goextract.ExtractedNode{Node: &schema.Node{Id: owner, Kind: goextract.KindStruct, Name: "Shape"}})
		var methods []string
		for i := range rapid.IntRange(1, 4).Draw(t, "methods") {
			name := "Shape.M" + strconv.Itoa(i)
			id := a.Declare(goextract.KindMethod, name, "(int)", uint(20+i*10))
			methods = append(methods, id)
			r.Nodes = append(r.Nodes, goextract.ExtractedNode{Node: &schema.Node{
				Id: id, Kind: goextract.KindMethod, Name: name, Signature: "(n int)",
				Params: []*schema.Param{{Name: "n", Type: "int"}},
			}})
			r.IntraEdges = append(r.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{Source: owner, Target: id, Kind: goextract.RefKindContains}})
		}
		for range rapid.IntRange(0, 3).Draw(t, "calls") {
			r.Unresolved = append(r.Unresolved, goextract.UnresolvedRef{
				FromID: rapid.SampledFrom(methods).Draw(t, "from"), Name: "helper", Kind: goextract.RefKindCalls,
			})
		}

		fields := stringFields(reflect.ValueOf(r), "result")
		field := rapid.SampledFrom(fields).Draw(t, "field")
		planted := rapid.SampledFrom(append(methods, owner)).Draw(t, "planted")
		field.value.SetString(planted)

		err := a.Finalize(r)
		switch {
		case err == nil:
			if field.value.String() == planted {
				t.Fatalf("%s still holds %s, and Finalize returned no error", field.path, planted)
			}
			rewritten++
		case err.Error() == "nodeid: "+field.path+" still holds a provisional id after Finalize":
			reported++
		default:
			t.Fatalf("Finalize with a provisional id in %s: err = %v, want nil or an error that names %s", field.path, err, field.path)
		}
	})
	if rewritten == 0 || reported == 0 {
		t.Fatalf("Finalize rewrote %d planted ids and reported %d, want both above zero", rewritten, reported)
	}
}

type stringField struct {
	path  string
	value reflect.Value
}

// stringFields returns the settable string fields reachable from v
// through exported fields, pointers and slice elements, with their paths
// in the form that Finalize reports.
func stringFields(v reflect.Value, path string) []stringField {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return stringFields(v.Elem(), path)
	case reflect.String:
		if v.CanSet() {
			return []stringField{{path: path, value: v}}
		}
	case reflect.Struct:
		var out []stringField
		for i := range v.NumField() {
			if f := v.Type().Field(i); f.IsExported() {
				out = append(out, stringFields(v.Field(i), path+"."+f.Name)...)
			}
		}
		return out
	case reflect.Slice, reflect.Array:
		var out []stringField
		for i := range v.Len() {
			out = append(out, stringFields(v.Index(i), path+"["+strconv.Itoa(i)+"]")...)
		}
		return out
	}
	return nil
}
