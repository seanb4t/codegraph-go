package uiserver

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
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

// copyCoverageFixtureForUI materializes an independent copy of the
// Phase 10 D-13 committed fixture (internal/indexer/testdata/coverage)
// under a fresh t.TempDir(), generates the two test-time-only files
// (huge.go, broken.py), and indexes it through the SAME ordering the
// real `codegraph init` production path uses (internal/cli/init.go):
// storeDir is pre-created via os.MkdirAll BEFORE indexer.Run, so
// DiscoverAll's walk sees ".codegraph" already on disk and records it as
// its own DIR_DOTPREFIX exclusion — unlike uiserverBuildTagFixture above
// (and internal/query/coverage_test.go's indexCoverageFixtureExternal),
// which deliberately avoid that phantom for a cleaner unit-level count.
// This is "the real listener" scenario the plan's fixture numbers are
// pinned against: excluded=7 / DIR_DOTPREFIX=2, not 6/1.
func copyCoverageFixtureForUI(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "coverage"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy coverage fixture: %v", err)
	}

	writeOversizeGoFileForUI(t, dst, "huge.go")
	if err := os.Symlink("missing-target.py", filepath.Join(dst, "broken.py")); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	storeDir := filepath.Join(dst, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dst, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	return dst
}

// writeOversizeGoFileForUI writes a syntactically valid, build-tag-free
// Go source file at exactly parser.MaxSourceBytes+1 bytes (4194305) into
// dir/name — the SIZE_LIMIT trigger. Reproduced from
// internal/indexer/coverage_fixture_test.go's writeOversizeGoFile
// (unexported test code in another package); the byte count is inlined
// rather than importing internal/parser solely for this constant, since
// no other test in this package needs the dependency.
func writeOversizeGoFileForUI(t *testing.T, dir, name string) {
	t.Helper()
	const size = 4*1024*1024 + 1 // parser.MaxSourceBytes + 1
	header := []byte("package tmp\n\nfunc Placeholder() {}\n\n// ")
	buf := make([]byte, size)
	copy(buf, header)
	for i := len(header); i < len(buf)-1; i++ {
		buf[i] = 'x'
	}
	buf[len(buf)-1] = '\n'
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// pathsOfRows extracts each wire CoverageRow's Path, for compact
// log/failure output.
func pathsOfRows(rows []*uiv1.CoverageRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.GetPath()
	}
	return out
}

// TestGetHealthCoverageMatchesFixture is HLT-05/HLT-06 pinned through
// the real listener: the full D-13 fixture indexed exactly as `codegraph
// init` indexes it (storeDir pre-created, so ".codegraph" itself is a
// second DIR_DOTPREFIX exclusion) produces excluded=7 where the
// Engine-level test (store outside the repo) produces 6 — the delta is
// explained and asserted here, not just prose.
func TestGetHealthCoverageMatchesFixture(t *testing.T) {
	dir := copyCoverageFixtureForUI(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	healthResp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	cov := healthResp.Msg.GetCoverage()
	if !cov.GetKnown() {
		t.Fatal("coverage.known = false, want true")
	}
	t.Logf("wire: discovered=%d indexed=%d excluded=%d extraction_failed=%d", cov.GetDiscovered(), cov.GetIndexed(), cov.GetExcluded(), cov.GetExtractionFailed())
	if cov.GetDiscovered() != 6 || cov.GetIndexed() != 1 || cov.GetExcluded() != 7 || cov.GetExtractionFailed() != 1 {
		t.Fatalf("coverage = %+v, want discovered=6 indexed=1 excluded=7 extraction_failed=1", cov)
	}
	wantByReason := map[string]int64{
		"EXCLUSION_REASON_UNSUPPORTED_EXTENSION": 2,
		"EXCLUSION_REASON_DIR_DOTPREFIX":         2,
		"EXCLUSION_REASON_DIR_VENDOR":            1,
		"EXCLUSION_REASON_BUILD_TAG":             1,
		"EXCLUSION_REASON_SIZE_LIMIT":            1,
	}
	byReason := cov.GetExcludedByReason()
	if len(byReason) != len(wantByReason) {
		t.Errorf("excluded_by_reason has %d keys, want %d: %+v", len(byReason), len(wantByReason), byReason)
	}
	for reason, want := range wantByReason {
		if got := byReason[reason]; got != want {
			t.Errorf("excluded_by_reason[%s] = %d, want %d", reason, got, want)
		}
	}
	dirLevel := byReason["EXCLUSION_REASON_DIR_VENDOR"] + byReason["EXCLUSION_REASON_DIR_DOTPREFIX"]
	if want := cov.GetIndexed() + cov.GetExtractionFailed() + (cov.GetExcluded() - dirLevel); cov.GetDiscovered() != want {
		t.Errorf("invariant broken: discovered (%d) != indexed+extraction_failed+(excluded-dirLevel) (%d)", cov.GetDiscovered(), want)
	}

	// The extra DIR_DOTPREFIX row over the Engine-level fixture test is
	// ".codegraph" itself — assert it is actually present as a row, not
	// merely counted.
	covResp, err := client.GetCoverage(context.Background(), connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 1000, Reason: uiv1.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX}))
	if err != nil {
		t.Fatalf("GetCoverage (DIR_DOTPREFIX filter): %v", err)
	}
	if got := pathsOfRows(covResp.Msg.GetRows()); !reflect.DeepEqual(sortedStrings(got), []string{".codegraph", ".hidden"}) {
		t.Fatalf("DIR_DOTPREFIX rows = %v, want exactly [.codegraph .hidden]", got)
	}
}

// sortedStrings returns a sorted copy of ss for order-independent slice
// comparison.
func sortedStrings(ss []string) []string {
	out := append([]string(nil), ss...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// TestGetCoveragePagesRoundTripThroughTheListener proves paging is
// stable through the real wire: page_size:3 pages until exhaustion
// concatenate to exactly the same 8 rows a single page_size:1000 call
// returns, the sole extraction-failed row (broken.py) is first, and
// every non-final token is opaque (contains no row path verbatim).
func TestGetCoveragePagesRoundTripThroughTheListener(t *testing.T) {
	dir := copyCoverageFixtureForUI(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	fullResp, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 1000}))
	if err != nil {
		t.Fatalf("GetCoverage (single page): %v", err)
	}
	fullRows := fullResp.Msg.GetRows()
	if len(fullRows) != 8 {
		t.Fatalf("single-page rows = %d, want exactly 8: %v", len(fullRows), pathsOfRows(fullRows))
	}
	if fullRows[0].GetKind() != uiv1.CoverageRowKind_COVERAGE_ROW_KIND_EXTRACTION_FAILED || fullRows[0].GetPath() != "broken.py" {
		t.Fatalf("rows[0] = %+v, want broken.py (EXTRACTION_FAILED) first", fullRows[0])
	}
	extractionFailedCount := 0
	for _, r := range fullRows {
		if r.GetKind() == uiv1.CoverageRowKind_COVERAGE_ROW_KIND_EXTRACTION_FAILED {
			extractionFailedCount++
		}
	}
	if extractionFailedCount != 1 {
		t.Errorf("extraction-failed row count = %d, want exactly 1", extractionFailedCount)
	}

	var pagedPaths []string
	var tok string
	for {
		resp, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 3, PageToken: tok}))
		if err != nil {
			t.Fatalf("GetCoverage (page, token=%q): %v", tok, err)
		}
		for _, r := range resp.Msg.GetRows() {
			pagedPaths = append(pagedPaths, r.GetPath())
			if resp.Msg.GetNextPageToken() != "" && strings.Contains(resp.Msg.GetNextPageToken(), r.GetPath()) {
				t.Errorf("token %q contains row path %q verbatim — tokens must be opaque", resp.Msg.GetNextPageToken(), r.GetPath())
			}
		}
		if resp.Msg.GetNextPageToken() == "" {
			break
		}
		tok = resp.Msg.GetNextPageToken()
	}
	if !reflect.DeepEqual(pagedPaths, pathsOfRows(fullRows)) {
		t.Fatalf("paged concatenation = %v, want equal to single-page result %v", pagedPaths, pathsOfRows(fullRows))
	}
}

// TestGetCoverageMalformedTokenIsInvalidArgument proves a malformed
// page_token maps to connect.CodeInvalidArgument over the real wire, and
// that the error text never carries the absolute fixture path (T-10-05)
// — plus a positive control that a valid token round-trip still
// succeeds.
func TestGetCoverageMalformedTokenIsInvalidArgument(t *testing.T) {
	dir := copyCoverageFixtureForUI(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	_, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageToken: "not*a*token"}))
	if err == nil {
		t.Fatal("GetCoverage with a malformed page_token succeeded, want an error")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want CodeInvalidArgument", code)
	}
	if strings.Contains(err.Error(), dir) {
		t.Errorf("error text %q contains the absolute fixture path %q, want it scrubbed", err.Error(), dir)
	}

	// Positive control: get a real token from page 1, then replay it.
	page1, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 3}))
	if err != nil {
		t.Fatalf("GetCoverage (page 1 for positive control): %v", err)
	}
	if page1.Msg.GetNextPageToken() == "" {
		t.Fatal("page 1 NextPageToken is empty, want non-empty (8 rows > page size 3)")
	}
	if _, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 3, PageToken: page1.Msg.GetNextPageToken()})); err != nil {
		t.Fatalf("GetCoverage (positive control, valid token): %v", err)
	}
}

// TestGetCoverageAbortsWhenGenerationChangedBetweenPages is WR-01's
// listener-level regression test: fetching page 1, mutating the store's
// generation marker directly (simulating a concurrent Sync), then
// replaying page 1's token must answer connect.CodeAborted over the real
// wire — never CodeInternal, never a silently wrong page. The mutation is
// a hand-set LastSyncUnixMs, never time.Now(), so this cannot flake on
// two commits landing in the same host millisecond.
func TestGetCoverageAbortsWhenGenerationChangedBetweenPages(t *testing.T) {
	dir := copyCoverageFixtureForUI(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	page1, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 1}))
	if err != nil {
		t.Fatalf("GetCoverage (page1): %v", err)
	}
	if page1.Msg.GetNextPageToken() == "" {
		t.Fatal("page1.NextPageToken is empty, want non-empty")
	}

	// Bump the store's generation marker directly, between the two page
	// fetches — the same shape internal/query/coverage_test.go's
	// TestCoverageRowsAbortsWhenGenerationChangedBetweenPages uses at the
	// Engine level, reproduced here to prove the classification survives
	// the wire (mapEngineError -> connect.CodeAborted).
	storeDir := filepath.Join(dir, ".codegraph", "store")
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if err := snap.Close(); err != nil {
		t.Fatalf("snap.Close: %v", err)
	}
	meta.LastSyncUnixMs = 999999
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit (bump generation): %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close: %v", err)
	}

	_, err = client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 1, PageToken: page1.Msg.GetNextPageToken()}))
	if err == nil {
		t.Fatal("GetCoverage with a stale-generation page_token succeeded, want an error")
	}
	if code := connect.CodeOf(err); code != connect.CodeAborted {
		t.Fatalf("code = %v, want CodeAborted", code)
	}
}

// TestGetCoveragePageSizeIsClampedOnTheWire is T-10-03's DoS mitigation
// asserted over the real wire: an oversized page_size clamps to
// CoverageMaxPageSize (1000), and a non-positive value falls back to
// CoverageDefaultPageSize (200).
func TestGetCoveragePageSizeIsClampedOnTheWire(t *testing.T) {
	dir := t.TempDir()
	writeCoverageFixtureFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	for i := 0; i < 1100; i++ {
		writeCoverageFixtureFile(t, dir, fmtPaddedNameForUI(i), "# generated\n")
	}
	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	resp, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 5000}))
	if err != nil {
		t.Fatalf("GetCoverage (oversized): %v", err)
	}
	if len(resp.Msg.GetRows()) != 1000 {
		t.Errorf("rows = %d, want exactly 1000", len(resp.Msg.GetRows()))
	}
	if resp.Msg.GetNextPageToken() == "" {
		t.Error("NextPageToken is empty, want non-empty (1101 exclusions > 1000)")
	}

	resp0, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 0}))
	if err != nil {
		t.Fatalf("GetCoverage (zero): %v", err)
	}
	if len(resp0.Msg.GetRows()) != 200 {
		t.Errorf("rows = %d, want exactly 200", len(resp0.Msg.GetRows()))
	}
}

// fmtPaddedNameForUI returns a deterministic, zero-padded markdown
// filename (n0000.md .. n1099.md) — the wire-level twin of
// internal/query/coverage_test.go's fmtPaddedName (unexported test code
// in another package).
func fmtPaddedNameForUI(i int) string {
	digits := "0123456789"
	b := []byte("n0000.md")
	for pos := 4; pos >= 1; pos-- {
		b[pos] = digits[i%10]
		i /= 10
	}
	return string(b)
}

// TestGetCoverageReasonFilterOnTheWire proves the reason filter reaches
// the wire correctly: a filtered call returns exactly the matching
// EXCLUDED rows, and an unfiltered call's extraction-failed row keeps
// its UNSPECIFIED reason and scrubbed detail.
func TestGetCoverageReasonFilterOnTheWire(t *testing.T) {
	dir := copyCoverageFixtureForUI(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	filtered, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{Reason: uiv1.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION}))
	if err != nil {
		t.Fatalf("GetCoverage (filtered): %v", err)
	}
	got := pathsOfRows(filtered.Msg.GetRows())
	if !reflect.DeepEqual(sortedStrings(got), []string{"go.mod", "notes.md"}) {
		t.Fatalf("filtered rows = %v, want exactly [go.mod notes.md]", got)
	}
	for _, r := range filtered.Msg.GetRows() {
		if r.GetKind() != uiv1.CoverageRowKind_COVERAGE_ROW_KIND_EXCLUDED {
			t.Errorf("row %+v Kind = %v, want COVERAGE_ROW_KIND_EXCLUDED", r, r.GetKind())
		}
	}

	unfiltered, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{PageSize: 1000}))
	if err != nil {
		t.Fatalf("GetCoverage (unfiltered): %v", err)
	}
	var brokenRow *uiv1.CoverageRow
	for _, r := range unfiltered.Msg.GetRows() {
		if r.GetPath() == "broken.py" {
			brokenRow = r
		}
	}
	if brokenRow == nil {
		t.Fatalf("unfiltered rows = %v, want a broken.py row", pathsOfRows(unfiltered.Msg.GetRows()))
	}
	if brokenRow.GetKind() != uiv1.CoverageRowKind_COVERAGE_ROW_KIND_EXTRACTION_FAILED {
		t.Errorf("broken.py Kind = %v, want COVERAGE_ROW_KIND_EXTRACTION_FAILED", brokenRow.GetKind())
	}
	if brokenRow.GetReason() != uiv1.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED {
		t.Errorf("broken.py Reason = %v, want EXCLUSION_REASON_UNSPECIFIED", brokenRow.GetReason())
	}
	if !strings.Contains(brokenRow.GetDetail(), "./broken.py") {
		t.Errorf("broken.py Detail = %q, want it to contain %q", brokenRow.GetDetail(), "./broken.py")
	}
	if strings.Contains(brokenRow.GetDetail(), dir) {
		t.Errorf("broken.py Detail = %q, must not contain the absolute fixture path %q", brokenRow.GetDetail(), dir)
	}
}

// TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse is D-06/D-15's
// contrast case asserted through the real listener on BOTH rpcs: a
// graph whose Meta.has_coverage is unset (a pre-Phase-10 graph)
// answers known=false with every count zero and no error — never 0/0,
// never CodeInternal.
func TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse(t *testing.T) {
	dir := uiserverBuildTagFixture(t)

	// Fabricate a pre-Phase-10 graph by flipping has_coverage back off
	// and clearing the c/ namespace on the ALREADY-INDEXED store — a
	// fresh open/write/close cycle distinct from any in-flight request,
	// since each RPC opens and closes its own Engine (SRV-04).
	storeDir := filepath.Join(dir, ".codegraph", "store")
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	meta, err := snap.GetMeta()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if err := snap.Close(); err != nil {
		t.Fatalf("snap.Close: %v", err)
	}
	meta.HasCoverage = false
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.DeleteAllExcludedFiles(); err != nil {
		t.Fatalf("DeleteAllExcludedFiles: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close: %v", err)
	}

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	ctx := context.Background()

	healthResp, err := client.GetHealth(ctx, connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	cov := healthResp.Msg.GetCoverage()
	if cov.GetKnown() {
		t.Fatal("GetHealth coverage.known = true, want false")
	}
	if cov.GetDiscovered() != 0 || cov.GetIndexed() != 0 || cov.GetExcluded() != 0 || cov.GetExtractionFailed() != 0 {
		t.Errorf("coverage counts = %+v, want all zero", cov)
	}
	if len(cov.GetExcludedByReason()) != 0 {
		t.Errorf("excluded_by_reason = %v, want empty", cov.GetExcludedByReason())
	}

	covResp, err := client.GetCoverage(ctx, connect.NewRequest(&uiv1.GetCoverageRequest{}))
	if err != nil {
		t.Fatalf("GetCoverage: %v", err)
	}
	if covResp.Msg.GetKnown() {
		t.Fatal("GetCoverage known = true, want false")
	}
	if len(covResp.Msg.GetRows()) != 0 {
		t.Errorf("Rows = %+v, want empty", covResp.Msg.GetRows())
	}
	if covResp.Msg.GetNextPageToken() != "" {
		t.Errorf("NextPageToken = %q, want empty", covResp.Msg.GetNextPageToken())
	}
}
