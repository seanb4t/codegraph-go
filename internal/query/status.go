package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
//	nodesByKind / languages          | computed via a full IterateNodes() scan     | D-03's IterateNodes; reflects whichever LanguageSpecs are registered AND have discoverable files in the repo (a Go-only repo reads languages:["go"]; a repo with both Go and Python source, e.g. the weft corpus, reads languages:["go","python"] once Phase 5's Python extraction lands)
//	pendingChanges                   | {added:0,modified:0,removed:0}              | Phase-4 sync concept; the added/modified/removed COUNT breakdown remains an inert placeholder — computing it would require re-running Sync's diff at Status()-time (out of scope, RESEARCH A2). The plain existence of pending changes is now live via the new top-level `stale` field below (D-04a)
//	worktreeMismatch                 | null                                         | Phase-4 sync concept; present-but-inert placeholder (RESEARCH A2)
//	stale                            | live bool (D-04a)                           | true when `.codegraph/.sync-pending` exists (watcher/daemon signal) OR — no-daemon fallback — the newest on-disk source-file mtime is newer than Meta.last_sync_unix_ms; this is the field that makes the sync-pending concept real this phase, see computeStale
//	index.builtWithVersion            | fmt.Sprintf("%d", schema.SchemaVersion)     | Same Go analog as top-level version — no separate release/build-version concept
//	index.builtWithExtractionVersion | uint32(schema.SchemaVersion)                | Go has one "extraction version" concept: the schema version stamped by NewMeta
//	index.currentExtractionVersion   | uint32(schema.SchemaVersion)                | This build's own SchemaVersion constant
//	index.reindexRecommended          | !schema.IsCurrentSchemaVersion(meta)        | Derived, not a literal placeholder — true when the stored Meta predates this build's schema version
//	index.state                      | "complete" if Meta exists, else "not_indexed" | Best-effort Go analog of TS's index lifecycle state
//	index.pendingRefs                | 0 (always)                                  | Phase 2 resolves all refs at index time (no unresolved-ref persistence in Go v1); inert placeholder matching the golden's own steady-state 0
//	dbSizeBytes                      | filepath.WalkDir byte sum over .codegraph/store/ | D-07 — Pebble has no single-file page-count analog to SQLite; a recursive byte sum over the store dir (SSTables+WAL+MANIFEST) is the honest Go-truthful reading. Best-effort: an Engine with no repoRoot (New, not OpenAt) or an unreadable/missing store dir degrades to 0 rather than failing Status(). Reverses the golden-corpus strip on the Go side only — see D-08 and testdata/golden/README.md's volatile-fields table
//	filesByLanguage                  | map[string]int64, json:"-" (Go-internal only) | D-05 — genuinely NEW computation (not already-scanned data): computed in the existing IterateFiles() scan by reading fileIt.File().Language. NOT emitted in --json — TS's own --json derives `languages` from this map and discards the counts, so emitting the key here would be a NEW Go-vs-TS divergence in the exact shape the golden oracle guards. Exists solely to feed the human/markdown renderers (Phase 2 wave 2)
//	(no lastIndexed / *_at keys)      | omitted entirely                           | Volatile fields per testdata/golden/README.md's stripping rules — never rendered (dbSizeBytes above is the one documented exception, D-08)
type StatusResult struct {
	Initialized      bool             `json:"initialized"`
	Version          string           `json:"version"`
	ProjectPath      string           `json:"projectPath"`
	IndexPath        string           `json:"indexPath"`
	FileCount        int64            `json:"fileCount"`
	NodeCount        int64            `json:"nodeCount"`
	EdgeCount        int64            `json:"edgeCount"`
	DbSizeBytes      int64            `json:"dbSizeBytes"`
	Backend          string           `json:"backend"`
	NodesByKind      map[string]int64 `json:"nodesByKind"`
	FilesByLanguage  map[string]int64 `json:"-"`
	Languages        []string         `json:"languages"`
	PendingChanges   PendingChanges   `json:"pendingChanges"`
	WorktreeMismatch *string          `json:"worktreeMismatch"`
	Stale            bool             `json:"stale"`
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

// staleSidecarName is the watcher/daemon's pending-sync touch-file (D-04a,
// RESEARCH Pattern 4): present under the resolved .codegraph/ directory
// between the first debounced file event and the next successful sync
// commit, which removes it.
const staleSidecarName = ".sync-pending"

// shouldSkipStaleDir mirrors internal/indexer.ShouldSkipDir's directory
// exclusions (vendor/ and any dot-prefixed directory), duplicated here
// rather than imported to avoid an internal/query -> internal/indexer
// dependency edge — the same package-local-duplication precedent Plan
// 04-03 established for buildReverseAdjacency (04-03-SUMMARY.md).
func shouldSkipStaleDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}

// newestSourceMtime walks root (skipping vendor/ and dot-prefixed
// directories) and returns the newest regular file's modification time —
// the no-daemon staleness fallback's raw signal (D-04a).
func newestSourceMtime(root string) (time.Time, error) {
	var newest time.Time
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && shouldSkipStaleDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return newest, nil
}

// computeStale reports the live staleness signal (D-04a): true when the
// watcher/daemon's .codegraph/.sync-pending sidecar is present, or — the
// no-daemon fallback — when the newest on-disk source-file mtime under the
// repo root is newer than meta's last_sync_unix_ms. meta may be nil (no
// Meta record yet, e.g. a store that exists but was never indexed), in
// which case any discovered file is considered newer than the zero-value
// "never synced" timestamp. An Engine with no repoRoot configured (New,
// not OpenAt) has no filesystem context to check and reports not stale
// rather than erroring — Node/Explore's disk reads already reject that
// configuration outright, but Status must degrade safely (T-04-06-02).
func (e *Engine) computeStale(meta *schema.Meta) (bool, error) {
	if e.repoRoot == "" {
		return false, nil
	}

	sidecar := filepath.Join(e.repoRoot, codegraphDirName, staleSidecarName)
	if _, err := os.Stat(sidecar); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	newest, err := newestSourceMtime(e.repoRoot)
	if err != nil {
		return false, err
	}

	var lastSync int64
	if meta != nil {
		lastSync = meta.GetLastSyncUnixMs()
	}
	return newest.UnixMilli() > lastSync, nil
}

// dbSizeBytes sums every regular file's size under storeDir (D-07),
// mirroring newestSourceMtime's best-effort degrade-on-error philosophy
// with one refinement: a per-entry error DEEPER in the tree (e.g. an
// SSTable deleted mid-walk by a live Pebble compaction) is skipped rather
// than aborting the walk, so a partially unreadable store still yields a
// partial sum; but a failure to even start walking storeDir itself (e.g.
// it does not exist) is surfaced as this function's own error return,
// which Status() below deliberately swallows — DbSizeBytes stays 0 and
// the whole status call still succeeds (T-02-07's "a failed walk leaves
// DbSizeBytes at 0 without erroring Status()"). Note
// pebble.DB.Metrics().DiskSpaceUsage() exists in the pinned pebble/v2
// v2.1.6 and is a cheaper, more idiomatic future path, but it requires
// extending graphstore.GraphStore/Reader and giving Engine a store handle
// rather than just its snapshot Reader — plumbing this plan deliberately
// did not take on. Documented future optimization, not a blocker.
func dbSizeBytes(storeDir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(storeDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == storeDir {
				return err // can't even start the walk — surface it, caller degrades
			}
			return nil // best-effort: skip unreadable entries deeper in the tree, don't abort
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

// Status reports index health/counts (QRY-09) by scanning the frozen
// graph: fileCount + filesByLanguage from a single IterateFiles scan,
// nodeCount + nodesByKind from a single IterateNodes scan, and edgeCount
// from GetMeta (avoiding a second full edge scan — the indexer stamps
// Meta.EdgeCount at index time, internal/indexer/resolve.go). languages
// is derived from filesByLanguage (D-05: count > 0, sorted), not from a
// separate node-scan languageSet, so it reflects every file the indexer
// discovered and stored — including a file that yields zero extracted
// nodes — rather than only files with at least one resolved node. A
// missing Meta record (a store that exists but was never indexed) is
// tolerated rather than treated as an error: counts fall back to the
// scanned values and index.state reports "not_indexed".
func (e *Engine) Status() (StatusResult, error) {
	fileIt, err := e.reader.IterateFiles()
	if err != nil {
		return StatusResult{}, err
	}
	defer fileIt.Close()

	var fileCount int64
	filesByLang := make(map[string]int64)
	for fileIt.Next() {
		fileCount++
		if lang := fileIt.File().Language; lang != "" {
			filesByLang[lang]++
		}
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
	var nodeCount int64
	for nodeIt.Next() {
		n := nodeIt.Node()
		nodeCount++
		nodesByKind[n.Kind]++
	}
	if err := nodeIt.Err(); err != nil {
		return StatusResult{}, err
	}

	languages := make([]string, 0, len(filesByLang))
	for lang, count := range filesByLang {
		if count > 0 {
			languages = append(languages, lang)
		}
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

	stale, err := e.computeStale(meta)
	if err != nil {
		return StatusResult{}, err
	}

	// D-07: best-effort — an Engine with no repoRoot (New, not OpenAt) or
	// a missing/unreadable store dir degrades DbSizeBytes to 0 rather
	// than failing the whole status call (T-02-07).
	var dbSize int64
	if e.repoRoot != "" {
		if size, err := dbSizeBytes(filepath.Join(e.repoRoot, codegraphDirName, storeSubdir)); err == nil {
			dbSize = size
		}
	}

	return StatusResult{
		Initialized:     true,
		Version:         version,
		FileCount:       fileCount,
		NodeCount:       nodeCount,
		EdgeCount:       edgeCount,
		DbSizeBytes:     dbSize,
		Stale:           stale,
		Backend:         "pebble",
		NodesByKind:     nodesByKind,
		FilesByLanguage: filesByLang,
		Languages:       languages,
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
