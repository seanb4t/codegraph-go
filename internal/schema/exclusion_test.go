package schema

import "testing"

// TestIsDirectoryExclusionCoversTheClosedEnum iterates every value of the
// ExclusionReason descriptor (asserting the closed set's size, so a future
// added value forces this test to be revisited rather than silently
// passing with an incomplete switch) and expects IsDirectoryExclusion true
// for exactly {DIR_VENDOR, DIR_DOTPREFIX} and false for every other
// defined value, including UNSPECIFIED.
func TestIsDirectoryExclusionCoversTheClosedEnum(t *testing.T) {
	desc := ExclusionReason(0).Descriptor().Values()
	if got := desc.Len(); got != 6 {
		t.Fatalf("ExclusionReason has %d values, want 6 — this test's exhaustive switch must be revisited if the enum grows", got)
	}

	wantDirectory := map[ExclusionReason]bool{
		ExclusionReason_EXCLUSION_REASON_UNSPECIFIED:           false,
		ExclusionReason_EXCLUSION_REASON_DIR_VENDOR:            true,
		ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX:         true,
		ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION: false,
		ExclusionReason_EXCLUSION_REASON_BUILD_TAG:             false,
		ExclusionReason_EXCLUSION_REASON_SIZE_LIMIT:            false,
	}
	if len(wantDirectory) != 6 {
		t.Fatalf("test fixture wantDirectory has %d entries, want 6 — this guard would be vacuous over the missing values", len(wantDirectory))
	}

	inspected := 0
	for i := 0; i < desc.Len(); i++ {
		num := ExclusionReason(desc.Get(i).Number())
		want, ok := wantDirectory[num]
		if !ok {
			t.Fatalf("ExclusionReason value %v (%d) is not covered by this test's fixture", desc.Get(i).Name(), num)
		}
		if got := IsDirectoryExclusion(num); got != want {
			t.Errorf("IsDirectoryExclusion(%v) = %v, want %v", desc.Get(i).Name(), got, want)
		}
		inspected++
	}
	if inspected != 6 {
		t.Fatalf("inspected %d enum values, want 6 — this guard is vacuous otherwise", inspected)
	}
}
