// Package migrate — this file is the durable, resumable migration cursor
// (D-06). A Progress record is persisted through the Wave-1 (07-02)
// graphstore.Writer.PutMigration/Reader.GetMigration pair under the target
// store's own m/migration meta key, so an interrupted migration can be
// resumed from the last-committed table+rowid rather than restarted from
// scratch.
package migrate

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// Progress is the resumable migration cursor: which source/target schema
// versions this run is bridging, the last table + rowid successfully
// committed, and whether the run is still in progress or done.
type Progress struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion uint32 `json:"target_schema_version"`
	LastTable           string `json:"last_table"`
	LastRowID           int64  `json:"last_row_id"`
	Status              string `json:"status"`

	// Reconciled source row counts, persisted when the cursor is stamped
	// StatusComplete (WR-01). finishFromComplete reads these back into the
	// resumed/recovered Result.Report so the CLI's "migrated/source"
	// reconciliation line shows the real source denominators instead of 0 —
	// on an in-place recovery the source is gone and cannot be re-counted, so
	// it must have been persisted at completion. Zero on in_progress cursors.
	SourceNodeCount int64 `json:"source_node_count,omitempty"`
	SourceEdgeCount int64 `json:"source_edge_count,omitempty"`
	SourceFileCount int64 `json:"source_file_count,omitempty"`
}

// Status values for Progress.Status.
const (
	StatusInProgress = "in_progress"
	StatusComplete   = "complete"
)

// saveProgress marshals p and stages it via w.PutMigration. It does NOT
// call w.Commit() — the caller commits the cursor (typically in its own
// small batch, immediately after each data batch) so the cursor advances
// durably alongside the migrated data (D-06).
func saveProgress(w graphstore.Writer, p Progress) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("migrate: marshal progress: %w", err)
	}
	if err := w.PutMigration(data); err != nil {
		return fmt.Errorf("migrate: save progress: %w", err)
	}
	return nil
}

// loadProgress reads the migration cursor via r.GetMigration. A store that
// has never had a cursor written returns (zero Progress, false, nil) — a
// clean "absent" signal, not an error, so a first run starts from the top.
// Any other read error, or non-JSON cursor bytes, is wrapped and returned:
// a lost or garbled cursor must fail loud rather than be silently treated
// as "start clean" (which would risk a double-migration).
func loadProgress(r graphstore.Reader) (Progress, bool, error) {
	data, err := r.GetMigration()
	if err != nil {
		if errors.Is(err, graphstore.ErrNotFound) {
			return Progress{}, false, nil
		}
		return Progress{}, false, fmt.Errorf("migrate: load progress: %w", err)
	}

	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return Progress{}, false, fmt.Errorf("migrate: load progress: corrupt cursor bytes: %w", err)
	}
	return p, true, nil
}
