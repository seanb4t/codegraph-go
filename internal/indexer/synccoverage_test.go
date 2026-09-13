package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestDiffExclusionsUpsertsAbsentAndChangedOnly pins diffExclusions' pure
// diff behavior (D-07): only absent-or-changed fresh records are staged
// for upsert (T-10-13's no-op-preserving discipline), and every stored
// path missing from the fresh set is staged for prune (T-10-07). Both
// return slices come back byte-sorted by path regardless of input order.
func TestDiffExclusionsUpsertsAbsentAndChangedOnly(t *testing.T) {
	stored := map[string]*schema.ExcludedFile{
		"a.go": {
			Path:   "a.go",
			Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG,
			Detail: "darwin/arm64",
		},
		"gone.go": {
			Path:   "gone.go",
			Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG,
			Detail: "darwin/arm64",
		},
	}

	fresh := []*schema.ExcludedFile{
		{Path: "a.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, Detail: "darwin/arm64"},
		{Path: "b.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, Detail: "darwin/arm64"},
	}

	upserts, prunes := diffExclusions(stored, fresh)

	if len(upserts) != 1 || upserts[0].GetPath() != "b.go" {
		t.Fatalf("upserts = %v, want exactly [b.go] (a.go is unchanged and must not be re-written)", upserts)
	}
	if len(prunes) != 1 || prunes[0] != "gone.go" {
		t.Fatalf("prunes = %v, want exactly [gone.go]", prunes)
	}

	// A fresh record for "a.go" with a DIFFERENT detail IS upserted, even
	// though the path is already present in stored.
	freshChangedDetail := []*schema.ExcludedFile{
		{Path: "a.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, Detail: "linux/amd64"},
	}
	upsertsChanged, prunesChanged := diffExclusions(stored, freshChangedDetail)
	if len(upsertsChanged) != 1 || upsertsChanged[0].GetPath() != "a.go" {
		t.Fatalf("upsertsChanged = %v, want exactly [a.go] (detail changed)", upsertsChanged)
	}
	if len(prunesChanged) != 1 || prunesChanged[0] != "gone.go" {
		t.Fatalf("prunesChanged = %v, want exactly [gone.go]", prunesChanged)
	}

	// Sort order is byte order by path, independent of input order.
	freshUnsorted := []*schema.ExcludedFile{
		{Path: "z.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".z"},
		{Path: "b.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".b"},
		{Path: "a.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".a"},
	}
	upsertsSorted, _ := diffExclusions(map[string]*schema.ExcludedFile{}, freshUnsorted)
	if len(upsertsSorted) != 3 ||
		upsertsSorted[0].GetPath() != "a.go" ||
		upsertsSorted[1].GetPath() != "b.go" ||
		upsertsSorted[2].GetPath() != "z.go" {
		t.Fatalf("upsertsSorted = %v, want byte-sorted [a.go b.go z.go]", upsertsSorted)
	}

	// Empty stored + empty fresh -> nil, nil.
	upsertsEmpty, prunesEmpty := diffExclusions(map[string]*schema.ExcludedFile{}, nil)
	if upsertsEmpty != nil {
		t.Fatalf("upsertsEmpty = %v, want nil", upsertsEmpty)
	}
	if prunesEmpty != nil {
		t.Fatalf("prunesEmpty = %v, want nil", prunesEmpty)
	}
}
