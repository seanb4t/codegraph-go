package schema

// IsDirectoryExclusion reports whether r represents a directory-level
// exclusion record (D-02) — a pruned subtree represented by ONE record,
// never phantom per-file rows — as opposed to a file-level exclusion
// (unsupported extension, build tag, size limit). This is the ONE
// definition of "directory-level record" that both the indexer (writing
// records) and the query engine (computing the discovered-count
// denominator, D-01/research Pitfall 1) import, so the two can never
// disagree on which reasons count as directory-level.
func IsDirectoryExclusion(r ExclusionReason) bool {
	switch r {
	case ExclusionReason_EXCLUSION_REASON_DIR_VENDOR, ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX:
		return true
	default:
		return false
	}
}
