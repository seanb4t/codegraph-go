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
