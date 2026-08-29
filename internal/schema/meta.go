package schema

// SchemaVersion is the current on-disk schema version stamped into every
// Meta record (D-02).
//
// Additive-only discipline (D-02a): within a single SchemaVersion, fields
// on Node, Edge, File, and Meta are NEVER renumbered or reused. Retiring a
// field means adding its number to a `reserved` clause in graph.proto, not
// deleting or repurposing it. SchemaVersion itself is bumped ONLY when a
// genuinely breaking layout change is unavoidable — a well-formed additive
// change (a new field, a newly reserved range) never requires a bump.
const SchemaVersion uint32 = 1

// NewMeta returns a Meta record stamped with the current SchemaVersion.
// Callers should use this instead of constructing a Meta literal directly,
// so a SchemaVersion bump only needs to change in one place.
func NewMeta() *Meta {
	return &Meta{SchemaVersion: SchemaVersion}
}

// IsCurrentSchemaVersion reports whether m carries the schema version this
// build of codegraph-go was compiled against. A record with an older
// version is expected to still decode (protobuf's forward/backward
// compatibility, ARCH-01) but callers that need to gate on version drift
// can use this helper rather than comparing SchemaVersion inline.
func IsCurrentSchemaVersion(m *Meta) bool {
	if m == nil {
		return false
	}
	return m.GetSchemaVersion() == SchemaVersion
}

// IndexedCommitSHA reports the git commit m's index was built at (ENG-04,
// D-05). A nil m, or a m whose commit_sha field is absent or empty,
// returns ("", false) — the same "unknown, not an error" contract
// has_file_index's own absent case follows: a pre-upgrade graph (built
// before field 8 existed) and a non-git checkout are both indistinguishable
// from each other and both degrade the same way. Callers get one place to
// ask "do we know the indexed commit" rather than each re-deriving the
// empty-means-absent convention.
func IndexedCommitSHA(m *Meta) (string, bool) {
	if m == nil {
		return "", false
	}
	sha := m.GetCommitSha()
	if sha == "" {
		return "", false
	}
	return sha, true
}

// SHA1HexLen and SHA256HexLen are the two lowercase-hex lengths a
// well-formed git commit object id can have — promoted here from
// internal/indexer/commit.go (WR-07) so the ONE validator a commit SHA is
// checked against at WRITE time (internal/indexer/commit.go's
// resolveHeadCommitSHA) is also available to check it again at READ time,
// rather than trusting the stored value is well-formed forever.
//
// IN-05: exported (were gitSHA1HexLen/gitSHA256HexLen, unexported) so
// internal/indexer/commit.go can reference these directly instead of
// keeping its own copy "numerically identical to schema's" by comment
// alone, with nothing asserting that claim stayed true.
const (
	SHA1HexLen   = 40
	SHA256HexLen = 64
)

// IsCommitSHA reports whether s is a well-formed git commit object id:
// exactly SHA1HexLen or SHA256HexLen characters, every one a lowercase
// hex digit (WR-07). internal/indexer applies this at WRITE time before a
// commit_sha is ever stamped into a Meta record — but IndexedCommitSHA
// above returns whatever string a Meta record on disk happens to carry,
// unvalidated. A store built or edited by anything other than this
// binary's own indexer (the milestone-2 "CI-distributed indexes" shape
// .claude/CLAUDE.md names as this project's own architecture target) is
// not bound by that write-time guarantee, and the unvalidated value is
// used both as a git CLI argument (gitmeta.CommitOnRemoteTrackingBranch)
// and spliced raw into a rendered GitHub blob URL
// (uiserver.buildGitHubBlobURL) — callers that read a commit SHA out of a
// Meta record and pass it to either of those should call this first.
func IsCommitSHA(s string) bool {
	if len(s) != SHA1HexLen && len(s) != SHA256HexLen {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
