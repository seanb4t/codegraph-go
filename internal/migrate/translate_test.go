package migrate

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestNodeFromRow_GoldenSpotCheck reproduces
// testdata/golden/ts-schema.dump.sql's constant:0f0ec020... row verbatim,
// per column order in testdata/golden/ts-schema.sql (id, kind, name,
// qualified_name, file_path, language, start_line, end_line, start_column,
// end_column, docstring, signature, visibility, is_exported, is_async,
// is_static, is_abstract, decorators, type_parameters, return_type,
// updated_at).
func TestNodeFromRow_GoldenSpotCheck(t *testing.T) {
	row := map[string]any{
		"id":             "constant:0f0ec02010b45f3735f3f6e3367ec872",
		"kind":           "constant",
		"name":           "EdgeType",
		"qualified_name": "EdgeType",
		"file_path":      "internal/plan/emit.go",
		"language":       "go",
		"start_line":     int64(21),
		"end_line":       int64(21),
		"start_column":   int64(6),
		"end_column":     int64(25),
		"docstring":      "EdgeType is the bd dependency type weft emits for authored + derived edges.",
		"signature":      `= "blocks"`,
		"visibility":     nil,
		"is_exported":    int64(0),
		"return_type":    nil,
	}

	got, err := nodeFromRow(row)
	if err != nil {
		t.Fatalf("nodeFromRow: %v", err)
	}
	want := &schema.Node{
		Id:            "constant:0f0ec02010b45f3735f3f6e3367ec872",
		Kind:          "constant",
		Name:          "EdgeType",
		QualifiedName: "EdgeType",
		FilePath:      "internal/plan/emit.go",
		Language:      "go",
		StartLine:     21,
		EndLine:       21,
		StartCol:      6,
		EndCol:        25,
		Docstring:     "EdgeType is the bd dependency type weft emits for authored + derived edges.",
		Signature:     `= "blocks"`,
		Visibility:    "",
		IsExported:    false,
		ReturnType:    "",
	}
	assertNodeEqual(t, got, want)
}

func TestNodeFromRow_StartColEndColCarried(t *testing.T) {
	row := map[string]any{
		"id": "class:1aa9ad9ada394f639ed0f8104462aef5", "kind": "class", "name": "StopServerTest",
		"qualified_name": "StopServerTest", "file_path": "plugin/skills/sketch/scripts/tests/test_stop_server.py",
		"language": "python", "start_line": int64(50), "end_line": int64(160),
		"start_column": int64(0), "end_column": int64(79),
	}
	got, err := nodeFromRow(row)
	if err != nil {
		t.Fatalf("nodeFromRow: %v", err)
	}
	if got.StartCol != 0 || got.EndCol != 79 {
		t.Errorf("StartCol/EndCol not carried: got StartCol=%d EndCol=%d, want 0/79", got.StartCol, got.EndCol)
	}
}

func TestNodeFromRow_IsExportedNonzeroIsTrue(t *testing.T) {
	row := map[string]any{"id": "x", "is_exported": int64(1)}
	got, err := nodeFromRow(row)
	if err != nil {
		t.Fatalf("nodeFromRow: %v", err)
	}
	if !got.IsExported {
		t.Error("expected IsExported=true for is_exported=1")
	}
}

// TestNodeFromRow_AgedMissingReturnType proves a row missing return_type
// (as an aged source's ScanTable output would be, since the column is
// absent from the SELECT entirely) yields ReturnType="" rather than
// erroring — a missing map key reads as nil.
func TestNodeFromRow_AgedMissingReturnType(t *testing.T) {
	row := map[string]any{"id": "x", "kind": "k", "name": "n"} // no "return_type" key at all
	got, err := nodeFromRow(row)
	if err != nil {
		t.Fatalf("nodeFromRow: %v", err)
	}
	if got.ReturnType != "" {
		t.Errorf("ReturnType = %q, want empty for a row missing the column", got.ReturnType)
	}
}

// TestEdgeFromRow_GoldenFileSourceContains reproduces
// testdata/golden/ts-schema.dump.sql's file:cmd/weft/main.go -> import:...
// contains edge (source/target/kind, metadata/line/col/provenance all NULL).
func TestEdgeFromRow_GoldenFileSourceContains(t *testing.T) {
	row := map[string]any{
		"source":     "file:cmd/weft/main.go",
		"target":     "import:daa6c015f8ad8dc967790f8acd33dbd7",
		"kind":       "contains",
		"metadata":   nil,
		"line":       nil,
		"col":        nil,
		"provenance": nil,
	}
	got, err := edgeFromRow(row)
	if err != nil {
		t.Fatalf("edgeFromRow: %v", err)
	}
	want := &schema.Edge{
		Source: "file:cmd/weft/main.go",
		Target: "import:daa6c015f8ad8dc967790f8acd33dbd7",
		Kind:   "contains",
	}
	if got.Source != want.Source || got.Target != want.Target || got.Kind != want.Kind {
		t.Errorf("edgeFromRow = %+v, want %+v", got, want)
	}
	if got.Line != 0 || got.Col != 0 || got.Provenance != "" || got.Metadata != nil {
		t.Errorf("expected zero-valued nullable fields, got Line=%d Col=%d Provenance=%q Metadata=%v", got.Line, got.Col, got.Provenance, got.Metadata)
	}
}

func TestEdgeFromRow_NullableLineColProvenanceDefaults(t *testing.T) {
	row := map[string]any{"source": "a", "target": "b", "kind": "calls", "line": nil, "col": nil, "provenance": nil}
	got, err := edgeFromRow(row)
	if err != nil {
		t.Fatalf("edgeFromRow: %v", err)
	}
	if got.Line != 0 || got.Col != 0 || got.Provenance != "" {
		t.Errorf("got Line=%d Col=%d Provenance=%q, want all zero-valued", got.Line, got.Col, got.Provenance)
	}
}

func TestEdgeFromRow_NonNullLineColProvenance(t *testing.T) {
	row := map[string]any{"source": "a", "target": "b", "kind": "calls", "line": int64(42), "col": int64(7), "provenance": "ast"}
	got, err := edgeFromRow(row)
	if err != nil {
		t.Fatalf("edgeFromRow: %v", err)
	}
	if got.Line != 42 || got.Col != 7 || got.Provenance != "ast" {
		t.Errorf("got Line=%d Col=%d Provenance=%q, want 42/7/ast", got.Line, got.Col, got.Provenance)
	}
}

func TestEdgeFromRow_MetadataMalformedFailsLoud(t *testing.T) {
	row := map[string]any{"source": "a", "target": "b", "kind": "calls", "metadata": "{not valid json"}
	_, err := edgeFromRow(row)
	if err == nil {
		t.Fatal("expected edgeFromRow to error on malformed metadata JSON")
	}
}

func TestFileFromRow_Verbatim(t *testing.T) {
	row := map[string]any{
		"path":         ".goreleaser.yml",
		"content_hash": "c64d8069e307231ab0675f30c7dac7827d62a70613c0ec89a5113d9e9d44262f",
		"language":     "yaml",
		"node_count":   int64(0),
		"size":         int64(1117),
		"modified_at":  int64(1700000000000),
		"errors":       nil,
	}
	got, err := fileFromRow(row)
	if err != nil {
		t.Fatalf("fileFromRow: %v", err)
	}
	if got.Path != ".goreleaser.yml" {
		t.Errorf("Path = %q", got.Path)
	}
	if got.ContentHash != "c64d8069e307231ab0675f30c7dac7827d62a70613c0ec89a5113d9e9d44262f" {
		t.Errorf("ContentHash = %q", got.ContentHash)
	}
	if got.Language != "yaml" {
		t.Errorf("Language = %q", got.Language)
	}
	if got.SizeBytes != 1117 {
		t.Errorf("SizeBytes = %d, want 1117", got.SizeBytes)
	}
	if got.MtimeUnixNs != 1700000000000*1_000_000 {
		t.Errorf("MtimeUnixNs = %d, want %d", got.MtimeUnixNs, int64(1700000000000)*1_000_000)
	}
	if got.Errors != nil {
		t.Errorf("Errors = %v, want nil for NULL errors column", got.Errors)
	}
	if got.EdgeCount != 0 {
		t.Errorf("EdgeCount = %d, want 0 (no TS source column, Pitfall 7 — recomputed at write time)", got.EdgeCount)
	}
}

func TestFileFromRow_ErrorsArray(t *testing.T) {
	row := map[string]any{"path": "x", "errors": `["file too large","parse timeout"]`}
	got, err := fileFromRow(row)
	if err != nil {
		t.Fatalf("fileFromRow: %v", err)
	}
	want := []string{"file too large", "parse timeout"}
	if len(got.Errors) != len(want) || got.Errors[0] != want[0] || got.Errors[1] != want[1] {
		t.Errorf("Errors = %v, want %v", got.Errors, want)
	}
}

func TestFileFromRow_ErrorsMalformedFailsLoud(t *testing.T) {
	row := map[string]any{"path": "x", "errors": "not json"}
	if _, err := fileFromRow(row); err == nil {
		t.Fatal("expected fileFromRow to error on malformed errors JSON")
	}
}

func TestFlattenMetadata(t *testing.T) {
	tests := []struct {
		name    string
		in      any
		want    map[string]string
		wantErr bool
	}{
		{name: "nil", in: nil, want: nil},
		{name: "empty string", in: "", want: nil},
		{name: "empty object", in: "{}", want: nil},
		{
			name: "string value decodes unescaped",
			in:   `{"note":"foo"}`,
			want: map[string]string{"note": "foo"},
		},
		{
			name: "non-string values preserve canonical JSON text",
			in:   `{"count":3,"flag":true,"list":["a"]}`,
			want: map[string]string{"count": "3", "flag": "true", "list": `["a"]`},
		},
		{name: "malformed JSON errors", in: "{not valid", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := flattenMetadata(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("flattenMetadata: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("flattenMetadata(%v) = %v, want %v", tc.in, got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("flattenMetadata(%v)[%q] = %q, want %q", tc.in, k, got[k], v)
				}
			}
		})
	}
}

// fractionalMs is bound to a typed float64 variable (not an untyped
// constant) so its use as both the test input and the expected-value
// computation goes through the same runtime float64 rounding — an untyped
// constant expression is evaluated at arbitrary precision by the compiler
// and would silently disagree with msToNs's actual float64 arithmetic in
// the least-significant digits.
var fractionalMs float64 = 1783108606938.7

func TestMsToNs(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int64
	}{
		{name: "nil is zero", in: nil, want: 0},
		{name: "integer ms", in: int64(1700000000000), want: 1700000000000 * 1_000_000},
		{name: "fractional ms float64", in: fractionalMs, want: int64(fractionalMs * 1e6)},
		{name: "integer ms as string", in: "1700000000000", want: 1700000000000 * 1_000_000},
		{name: "empty string is zero", in: "", want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := msToNs(tc.in)
			if got != tc.want {
				t.Errorf("msToNs(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeFilePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"internal/plan/plan.go", "internal/plan/plan.go"},
		{"./internal/plan/plan.go", "internal/plan/plan.go"},
		{`internal\plan\plan.go`, "internal/plan/plan.go"},
		{`C:\repo\internal\plan\plan.go`, "repo/internal/plan/plan.go"},
		{"/abs/repo/internal/plan/plan.go", "abs/repo/internal/plan/plan.go"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := normalizeFilePath(tc.in)
			if got != tc.want {
				t.Errorf("normalizeFilePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseErrorsJSON(t *testing.T) {
	if got, err := parseErrorsJSON(nil); err != nil || got != nil {
		t.Errorf("parseErrorsJSON(nil) = (%v, %v), want (nil, nil)", got, err)
	}
	if got, err := parseErrorsJSON(""); err != nil || got != nil {
		t.Errorf(`parseErrorsJSON("") = (%v, %v), want (nil, nil)`, got, err)
	}
	got, err := parseErrorsJSON(`["a","b"]`)
	if err != nil {
		t.Fatalf("parseErrorsJSON: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf(`parseErrorsJSON(["a","b"]) = %v`, got)
	}
	if _, err := parseErrorsJSON("not json"); err == nil {
		t.Error("expected parseErrorsJSON to error on malformed input")
	}
}

// assertNodeEqual compares the fields nodeFromRow populates (skips proto
// bookkeeping fields).
func assertNodeEqual(t *testing.T, got, want *schema.Node) {
	t.Helper()
	var diffs []string
	if got.Id != want.Id {
		diffs = append(diffs, "Id")
	}
	if got.Kind != want.Kind {
		diffs = append(diffs, "Kind")
	}
	if got.Name != want.Name {
		diffs = append(diffs, "Name")
	}
	if got.QualifiedName != want.QualifiedName {
		diffs = append(diffs, "QualifiedName")
	}
	if got.FilePath != want.FilePath {
		diffs = append(diffs, "FilePath")
	}
	if got.Language != want.Language {
		diffs = append(diffs, "Language")
	}
	if got.StartLine != want.StartLine {
		diffs = append(diffs, "StartLine")
	}
	if got.EndLine != want.EndLine {
		diffs = append(diffs, "EndLine")
	}
	if got.StartCol != want.StartCol {
		diffs = append(diffs, "StartCol")
	}
	if got.EndCol != want.EndCol {
		diffs = append(diffs, "EndCol")
	}
	if got.Signature != want.Signature {
		diffs = append(diffs, "Signature")
	}
	if got.Docstring != want.Docstring {
		diffs = append(diffs, "Docstring")
	}
	if got.Visibility != want.Visibility {
		diffs = append(diffs, "Visibility")
	}
	if got.IsExported != want.IsExported {
		diffs = append(diffs, "IsExported")
	}
	if got.ReturnType != want.ReturnType {
		diffs = append(diffs, "ReturnType")
	}
	if len(diffs) > 0 {
		t.Errorf("nodeFromRow mismatch on fields [%s]:\n got:  %+v\n want: %+v", strings.Join(diffs, ","), got, want)
	}
}
