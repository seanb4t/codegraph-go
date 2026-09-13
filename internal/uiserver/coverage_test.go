package uiserver

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// uiserverBuildTagFixture writes a temp repo with go.mod, main.go, and a
// //go:build ignore file, indexes it via a real indexer.Run, and returns
// its root — the same minimal tracer fixture as
// internal/query/coverage_test.go's buildTagRepoFixture (reproduced here
// rather than imported, since it is unexported test code in another
// package — server_test.go's copyGofixture/indexGofixture doc comment
// records the same rationale for its own fixture helpers).
func uiserverBuildTagFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCoverageFixtureFile(t, root, "go.mod", "module example.com/uiservercoverage\n\ngo 1.24\n")
	writeCoverageFixtureFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeCoverageFixtureFile(t, root, "tagged.go", "//go:build ignore\n\npackage main\n\nfunc Tagged() {}\n")

	// Phase 10 Plan 2: storeDir is deliberately NOT pre-created —
	// indexer.Run's own DiscoverAll walk runs BEFORE graphstore.Open
	// creates it, mirroring production's first-index ordering. Pre-
	// creating it here would make decision point 1 record a phantom
	// DIR_DOTPREFIX exclusion for ".codegraph" that a real from-scratch
	// index never sees.
	storeDir := filepath.Join(root, ".codegraph", "store")
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	return root
}

func writeCoverageFixtureFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, relPath), []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

// TestExclusionReasonEnumsAgree proves the UI-local uiv1.ExclusionReason
// enum and the schema codegraph.v1.ExclusionReason enum have identical
// (name, number) sets in BOTH directions (D-08) — so exclusionReasonToProto's
// numeric cast cannot silently skew.
func TestExclusionReasonEnumsAgree(t *testing.T) {
	uiValues := uiv1.ExclusionReason(0).Descriptor().Values()
	schemaValues := schema.ExclusionReason(0).Descriptor().Values()

	if got := uiValues.Len(); got != 6 {
		t.Fatalf("uiv1.ExclusionReason has %d values, want 6", got)
	}
	if uiValues.Len() != schemaValues.Len() {
		t.Fatalf("uiv1.ExclusionReason has %d values, schema.ExclusionReason has %d — want equal", uiValues.Len(), schemaValues.Len())
	}
	t.Logf("both enums have %d values", uiValues.Len())

	uiByNumber := make(map[int32]string, uiValues.Len())
	for i := 0; i < uiValues.Len(); i++ {
		v := uiValues.Get(i)
		uiByNumber[int32(v.Number())] = strings.TrimPrefix(string(v.Name()), "EXCLUSION_REASON_")
	}
	schemaByNumber := make(map[int32]string, schemaValues.Len())
	for i := 0; i < schemaValues.Len(); i++ {
		v := schemaValues.Get(i)
		schemaByNumber[int32(v.Number())] = strings.TrimPrefix(string(v.Name()), "EXCLUSION_REASON_")
	}

	checked := 0
	for num, name := range uiByNumber {
		other, ok := schemaByNumber[num]
		if !ok {
			t.Fatalf("uiv1.ExclusionReason value %s (%d) has no matching schema.ExclusionReason number", name, num)
		}
		if other != name {
			t.Fatalf("number %d: uiv1 name %q != schema name %q", num, name, other)
		}
		checked++
	}
	for num, name := range schemaByNumber {
		other, ok := uiByNumber[num]
		if !ok {
			t.Fatalf("schema.ExclusionReason value %s (%d) has no matching uiv1.ExclusionReason number", name, num)
		}
		if other != name {
			t.Fatalf("number %d: schema name %q != uiv1 name %q", num, name, other)
		}
		checked++
	}
	if checked != 12 {
		t.Fatalf("checked %d (name,number) pairs, want 12 (6 values x 2 directions) — this guard is under-checking", checked)
	}
}

// TestGetHealthCarriesCoverageForBuildTagExclusion proves GetHealth
// carries the coverage summary (Phase 10 HLT-05/HLT-06) and GetCoverage
// pages the matching row, both over a real Connect client against a real
// listener.
func TestGetHealthCarriesCoverageForBuildTagExclusion(t *testing.T) {
	dir := uiserverBuildTagFixture(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	healthResp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	cov := healthResp.Msg.GetCoverage()
	if !cov.GetKnown() {
		t.Fatal("GetHealth coverage.known = false, want true")
	}
	if cov.GetIndexed() != 1 {
		t.Errorf("coverage.indexed = %d, want 1", cov.GetIndexed())
	}
	if got := cov.GetExcludedByReason()["EXCLUSION_REASON_BUILD_TAG"]; got != 1 {
		t.Errorf("coverage.excluded_by_reason[BUILD_TAG] = %d, want 1", got)
	}
	if cov.GetDiscovered() != cov.GetIndexed()+cov.GetExtractionFailed()+cov.GetExcluded() {
		t.Errorf("discovered (%d) != indexed (%d) + extraction_failed (%d) + excluded (%d)", cov.GetDiscovered(), cov.GetIndexed(), cov.GetExtractionFailed(), cov.GetExcluded())
	}

	covResp, err := client.GetCoverage(context.Background(), connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 10}))
	if err != nil {
		t.Fatalf("GetCoverage: %v", err)
	}
	if !covResp.Msg.GetKnown() {
		t.Fatal("GetCoverage known = false, want true")
	}
	// Phase 10 Plan 2 also records go.mod itself as an
	// UNSUPPORTED_EXTENSION exclusion (decision point 2) — this fixture
	// now yields both records, not just tagged.go's.
	rows := covResp.Msg.GetRows()
	if len(rows) != 2 {
		t.Fatalf("GetCoverage rows = %+v, want exactly 2", rows)
	}
	byPath := make(map[string]*uiv1.CoverageRow, len(rows))
	for _, r := range rows {
		byPath[r.GetPath()] = r
	}
	row, ok := byPath["tagged.go"]
	if !ok {
		t.Fatalf("GetCoverage rows = %+v, want a tagged.go row", rows)
	}
	if row.GetKind() != uiv1.CoverageRowKind_COVERAGE_ROW_KIND_EXCLUDED {
		t.Errorf("row.Kind = %v, want COVERAGE_ROW_KIND_EXCLUDED", row.GetKind())
	}
	if row.GetReason() != uiv1.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Errorf("row.Reason = %v, want EXCLUSION_REASON_BUILD_TAG", row.GetReason())
	}
	if goModRow, ok := byPath["go.mod"]; !ok || goModRow.GetReason() != uiv1.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION {
		t.Fatalf("GetCoverage rows = %+v, want a go.mod UNSUPPORTED_EXTENSION row too", rows)
	}
	if covResp.Msg.GetNextPageToken() != "" {
		t.Errorf("NextPageToken = %q, want empty", covResp.Msg.GetNextPageToken())
	}
}
