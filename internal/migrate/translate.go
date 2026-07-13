package migrate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Field mapping (D-02, D-05) — 07-RESEARCH.md §Field Mapping is the
// authoritative source; this comment is a pointer, not a re-derivation.
//
// DROPPED (D-05, unconditional — no proto field exists for these TS-only
// node attributes): is_async, is_static, is_abstract, decorators,
// type_parameters. Also dropped: nodes.updated_at (no per-node timestamp in
// the proto), files.indexed_at (no proto home).
//
// CARRIED, NOT DROPPED (D-05 correction confirmed by RESEARCH against
// graph.proto): TS nodes.start_column/end_column ARE modeled as
// Node.StartCol (field 9) / Node.EndCol (field 10). CONTEXT.md's D-05 list
// of "start_column/end_column if unmodeled" resolves to CARRIED — they are
// modeled, so they must not be dropped.
func nodeFromRow(row map[string]any) (*schema.Node, error) {
	return &schema.Node{
		Id:            asString(row["id"]),
		Kind:          asString(row["kind"]),
		Name:          asString(row["name"]),
		QualifiedName: asString(row["qualified_name"]),
		FilePath:      normalizeFilePath(asString(row["file_path"])),
		Language:      asString(row["language"]),
		StartLine:     int32(asInt64(row["start_line"])),
		EndLine:       int32(asInt64(row["end_line"])),
		StartCol:      int32(asInt64(row["start_column"])),
		EndCol:        int32(asInt64(row["end_column"])),
		Signature:     asString(row["signature"]),
		Docstring:     asString(row["docstring"]),
		Visibility:    asString(row["visibility"]),
		IsExported:    asBool(row["is_exported"]),
		// return_type is absent on an aged source (D-09.4); a missing map
		// key reads as nil -> asString(nil) == "" -- the proto3 zero value.
		ReturnType: asString(row["return_type"]),
	}, nil
}

// edgeFromRow maps a scanned edges row to schema.Edge. metadata JSON is
// flattened via flattenMetadata; a malformed metadata payload is a fail-loud
// error, never swallowed (07-RESEARCH.md Pitfall/§Field Mapping).
func edgeFromRow(row map[string]any) (*schema.Edge, error) {
	metadata, err := flattenMetadata(row["metadata"])
	if err != nil {
		return nil, fmt.Errorf("migrate: edge %s->%s (%s): %w", asString(row["source"]), asString(row["target"]), asString(row["kind"]), err)
	}
	return &schema.Edge{
		Source:     asString(row["source"]),
		Target:     asString(row["target"]),
		Kind:       asString(row["kind"]),
		Line:       int32(asInt64(row["line"])),
		Col:        int32(asInt64(row["col"])),
		Provenance: asString(row["provenance"]),
		Metadata:   metadata,
	}, nil
}

// fileFromRow maps a scanned files row to schema.File. errors is a JSON
// array parsed via parseErrorsJSON. indexed_at is DROPPED (no proto home).
// edge_count (proto field 5) has no TS source column (Pitfall 7) — left at
// its zero value here; recomputed at write time.
func fileFromRow(row map[string]any) (*schema.File, error) {
	errs, err := parseErrorsJSON(row["errors"])
	if err != nil {
		return nil, fmt.Errorf("migrate: file %s: %w", asString(row["path"]), err)
	}
	return &schema.File{
		Path:        normalizeFilePath(asString(row["path"])),
		ContentHash: asString(row["content_hash"]),
		Language:    asString(row["language"]),
		NodeCount:   asInt64(row["node_count"]),
		Errors:      errs,
		MtimeUnixNs: msToNs(row["modified_at"]),
		SizeBytes:   asInt64(row["size"]),
	}, nil
}

// flattenMetadata parses a TS edges.metadata JSON object and flattens it to
// map[string]string: a JSON string value decodes to its unescaped content
// ("foo" -> foo); any non-string value (number, bool, null, array, object)
// preserves its canonical JSON text, since it has no natural string form.
// Empty/NULL metadata returns a nil map. Malformed metadata JSON is an
// error — fail loud, never swallowed.
func flattenMetadata(v any) (map[string]string, error) {
	s := asString(v)
	if s == "" {
		return nil, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, fmt.Errorf("migrate: parse metadata json: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(raw))
	for k, val := range raw {
		var str string
		if err := json.Unmarshal(val, &str); err == nil {
			m[k] = str
		} else {
			m[k] = string(val)
		}
	}
	return m, nil
}

// parseErrorsJSON parses a TS files.errors JSON array of strings.
// NULL/empty -> nil (no errors recorded).
func parseErrorsJSON(v any) ([]string, error) {
	s := asString(v)
	if s == "" {
		return nil, nil
	}
	var errs []string
	if err := json.Unmarshal([]byte(s), &errs); err != nil {
		return nil, fmt.Errorf("migrate: parse errors json: %w", err)
	}
	return errs, nil
}

// msToNs converts a TS epoch-millisecond timestamp to nanoseconds
// (File.MtimeUnixNs). nil/absent -> 0. Integer-ms values are multiplied
// directly (exact, no float round-trip) to avoid float64's ~2^53 integer
// precision ceiling on a large ms*1e6 nanosecond value; a fractional-ms
// value (observed in the wild per RESEARCH, e.g. 1783108606938.7) is scanned
// as float64 and does not panic or truncate-crash.
func msToNs(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int64:
		return t * 1_000_000
	case float64:
		return int64(t * 1e6)
	case []byte:
		return msFromString(string(t))
	case string:
		return msFromString(t)
	default:
		return 0
	}
}

func msFromString(s string) int64 {
	if s == "" {
		return 0
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n * 1_000_000
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f * 1e6)
	}
	return 0
}

// normalizeFilePath converts backslash separators to forward slashes and
// strips a leading "./", drive-letter, or absolute-path prefix, so a
// Windows-origin or absolute-path TS index carries repo-relative
// forward-slash paths matching the new format's relPath convention
// (07-RESEARCH.md Pitfall 6, defensive — the captured corpus is already
// repo-relative forward-slash).
func normalizeFilePath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	p = strings.TrimPrefix(p, "./")
	if len(p) >= 2 && p[1] == ':' { // Windows drive letter, e.g. "C:/repo/..."
		p = p[2:]
	}
	p = strings.TrimPrefix(p, "/")
	return p
}

// asString coerces a ScanTable row value (nil, string, []byte, or any other
// concrete type the modernc.org/sqlite driver may return) to a string. nil
// (SQL NULL) -> "" (the proto3 zero value).
func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// asInt64 coerces a ScanTable row value to int64. nil -> 0.
func asInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int64:
		return t
	case float64:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

// asFloat64 coerces a ScanTable row value to float64. nil -> 0.
func asFloat64(v any) float64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return t
	case int64:
		return float64(t)
	case []byte:
		f, _ := strconv.ParseFloat(string(t), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

// asBool coerces a ScanTable row value (TS's INTEGER 0/1 boolean encoding)
// to bool: any nonzero integer value is true.
func asBool(v any) bool {
	return asInt64(v) != 0
}
