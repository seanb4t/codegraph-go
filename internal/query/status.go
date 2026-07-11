package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// StatusResult mirrors testdata/golden/corpus/weft-go/status.json's shape
// (QRY-09), with TS-SQLite-specific keys remapped to Go/Pebble-truthful
// values or dropped, per CONTEXT D-05 and RESEARCH Open Question 2. This
// is the plan's authoritative per-key decision table:
//
//	TS key                          | Go/Pebble rendering                        | Rationale
//	---------------------------------|---------------------------------------------|----------
//	initialized                      | true (Status only runs on an opened Engine) | Unchanged — same concept
//	version                          | fmt.Sprintf("%d", schema.SchemaVersion)     | No codegraph-go release-version concept exists yet; schema version is the closest stable Go analog
//	projectPath / indexPath          | "" (empty string, key kept)                 | Engine carries no path context in its read-only Reader-only design (files_modified excludes engine.go this plan) — trivially satisfies T-03-05-Leak by having nothing host-specific to leak, while keeping the key present for shape parity
//	fileCount / nodeCount / edgeCount | computed from IterateFiles/IterateNodes scans + Meta.EdgeCount | Unchanged concept, Go-sourced values
//	backend                          | "pebble" (a literal Pebble identifier)      | D-05's explicit example remapping
//	journalMode                      | dropped (key omitted)                       | No Pebble user-facing WAL/journal-mode analog (RESEARCH Open Question 2); D-05 permits dropping keys with no Go analog
//	nodesByKind / languages          | computed via a full IterateNodes() scan     | D-03's IterateNodes; Go-only until Phase 5 (a Go repo reads languages:["go"])
//	pendingChanges                   | {added:0,modified:0,removed:0}              | Phase-4 sync concept; present-but-inert placeholder (RESEARCH A2) — status reports health but does not reconcile drift this phase
//	worktreeMismatch                 | null                                         | Phase-4 sync concept; present-but-inert placeholder (RESEARCH A2)
//	index.builtWithVersion            | fmt.Sprintf("%d", schema.SchemaVersion)     | Same Go analog as top-level version — no separate release/build-version concept
//	index.builtWithExtractionVersion | uint32(schema.SchemaVersion)                | Go has one "extraction version" concept: the schema version stamped by NewMeta
//	index.currentExtractionVersion   | uint32(schema.SchemaVersion)                | This build's own SchemaVersion constant
//	index.reindexRecommended          | !schema.IsCurrentSchemaVersion(meta)        | Derived, not a literal placeholder — true when the stored Meta predates this build's schema version
//	index.state                      | "complete" if Meta exists, else "not_indexed" | Best-effort Go analog of TS's index lifecycle state
//	index.pendingRefs                | 0 (always)                                  | Phase 2 resolves all refs at index time (no unresolved-ref persistence in Go v1); inert placeholder matching the golden's own steady-state 0
//	(no dbSizeBytes / lastIndexed / *_at keys) | omitted entirely                  | Volatile fields per testdata/golden/README.md's stripping rules — never rendered
type StatusResult struct {
	Initialized      bool             `json:"initialized"`
	Version          string           `json:"version"`
	ProjectPath      string           `json:"projectPath"`
	IndexPath        string           `json:"indexPath"`
	FileCount        int64            `json:"fileCount"`
	NodeCount        int64            `json:"nodeCount"`
	EdgeCount        int64            `json:"edgeCount"`
	Backend          string           `json:"backend"`
	NodesByKind      map[string]int64 `json:"nodesByKind"`
	Languages        []string         `json:"languages"`
	PendingChanges   PendingChanges   `json:"pendingChanges"`
	WorktreeMismatch *string          `json:"worktreeMismatch"`
	Index            IndexHealth      `json:"index"`
}

// PendingChanges mirrors the golden's pendingChanges shape — a Phase-4
// sync concept rendered as an inert all-zero placeholder in Phase 3 (see
// StatusResult's mapping table).
type PendingChanges struct {
	Added    int `json:"added"`
	Modified int `json:"modified"`
	Removed  int `json:"removed"`
}

// IndexHealth mirrors the golden's index.* shape, with the TS
// version/extraction fields remapped to schema.SchemaVersion-derived
// values (see StatusResult's mapping table).
type IndexHealth struct {
	BuiltWithVersion           string `json:"builtWithVersion"`
	BuiltWithExtractionVersion uint32 `json:"builtWithExtractionVersion"`
	CurrentExtractionVersion   uint32 `json:"currentExtractionVersion"`
	ReindexRecommended         bool   `json:"reindexRecommended"`
	State                      string `json:"state"`
	PendingRefs                int    `json:"pendingRefs"`
}

// Status reports index health/counts (QRY-09) by scanning the frozen
// graph: fileCount from IterateFiles, nodeCount + nodesByKind + languages
// from a single IterateNodes scan, and edgeCount from GetMeta (avoiding a
// second full edge scan — the indexer stamps Meta.EdgeCount at index
// time, internal/indexer/resolve.go). A missing Meta record (a store
// that exists but was never indexed) is tolerated rather than treated as
// an error: counts fall back to the scanned values and index.state
// reports "not_indexed".
func (e *Engine) Status() (StatusResult, error) {
	fileIt, err := e.reader.IterateFiles()
	if err != nil {
		return StatusResult{}, err
	}
	defer fileIt.Close()

	var fileCount int64
	for fileIt.Next() {
		fileCount++
	}
	if err := fileIt.Err(); err != nil {
		return StatusResult{}, err
	}

	nodeIt, err := e.reader.IterateNodes()
	if err != nil {
		return StatusResult{}, err
	}
	defer nodeIt.Close()

	nodesByKind := make(map[string]int64)
	languageSet := make(map[string]bool)
	var nodeCount int64
	for nodeIt.Next() {
		n := nodeIt.Node()
		nodeCount++
		nodesByKind[n.Kind]++
		if n.Language != "" {
			languageSet[n.Language] = true
		}
	}
	if err := nodeIt.Err(); err != nil {
		return StatusResult{}, err
	}

	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}
	sort.Strings(languages)

	meta, err := e.reader.GetMeta()
	if err != nil && !errors.Is(err, graphstore.ErrNotFound) {
		return StatusResult{}, err
	}

	version := fmt.Sprintf("%d", schema.SchemaVersion)
	var edgeCount int64
	state := "not_indexed"
	reindexRecommended := true
	if meta != nil {
		edgeCount = meta.GetEdgeCount()
		state = "complete"
		reindexRecommended = !schema.IsCurrentSchemaVersion(meta)
	}

	return StatusResult{
		Initialized: true,
		Version:     version,
		FileCount:   fileCount,
		NodeCount:   nodeCount,
		EdgeCount:   edgeCount,
		Backend:     "pebble",
		NodesByKind: nodesByKind,
		Languages:   languages,
		Index: IndexHealth{
			BuiltWithVersion:           version,
			BuiltWithExtractionVersion: schema.SchemaVersion,
			CurrentExtractionVersion:   schema.SchemaVersion,
			ReindexRecommended:         reindexRecommended,
			State:                      state,
			PendingRefs:                0,
		},
	}, nil
}

// MarshalStatusJSON renders a StatusResult as --json output, matching
// search.go/traverse.go's Marshal* convention.
func MarshalStatusJSON(r StatusResult) ([]byte, error) {
	return json.Marshal(r)
}
