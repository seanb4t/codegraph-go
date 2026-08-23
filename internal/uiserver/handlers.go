package uiserver

import (
	"context"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// openEngine is query.OpenAt indirected behind a package-level func var —
// the SAME test-only control-seam shape internal/graphstore's
// openLockRetrySleep (internal/graphstore/pebble_store.go:90) already
// established in this tree, and the ONE precedent whose shape is a
// package-level var; it is that shape being copied here, not
// internal/daemon's onSyncStart (internal/daemon/daemon.go:113), which
// is the same IDEA — a test-only control seam with no exported setter —
// at struct-field scope instead, cited only as a same-family precedent.
// Production behavior is unchanged: the var defaults to query.OpenAt and
// has no exported setter; tests can count or substitute opens.
var openEngine = query.OpenAt

// withEngine opens an Engine at repoPath via openEngine, defers closing
// the returned io.Closer, runs fn, and maps any error through the single
// mapEngineError translation site. One open, one deferred close, one
// error-translation site — the boilerplate every later handler in this
// package reuses instead of repeating it per rpc (review MEDIUM 01-06).
// Nothing is retained on uiService across the call: no *query.Engine,
// graphstore.GraphStore or graphstore.Reader value outlives withEngine's
// return (SRV-04's per-call discipline).
func withEngine(ctx context.Context, repoPath string, fn func(*query.Engine) error) error {
	eng, closer, err := openEngine(repoPath)
	if err != nil {
		return mapEngineError(err)
	}
	defer closer.Close()

	if err := fn(eng); err != nil {
		return mapEngineError(err)
	}
	return nil
}

// mapEngineError is withEngine's single error-translation site. This
// plan's tracer needs only a generic mapping to a Connect error; SRV-04's
// degrade path (CodeUnavailable carrying a typed IndexingInProgress
// detail for a store locked past graphstore.Open's retry budget, D-14)
// and the remaining eight specific error-class mappers this phase's
// later plans document are added here by 01-11, not by this plan.
func mapEngineError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeInternal, err)
}

// uiService implements uiv1connect.UIServiceHandler over the
// repository's own internal/query.Engine. It holds repoPath and NOTHING
// ELSE reachable through an Engine or GraphStore — asserted structurally
// by TestUIServiceHoldsNoStoreTypedField (reflecting over this struct's
// field types) and behaviorally by TestUIServerHoldsNoStoreHandleBetweenCalls
// (a second caller can open the same store directly after two RPC calls
// complete). Every rpc opens, uses and closes its own Engine via
// withEngine.
type uiService struct {
	// repoPath is the single repository this v1 server answers for,
	// copied from uiserver.Options.RepoPath at construction time
	// (Listen). GetStatusRequest.path is reserved for a future
	// multi-repository milestone and is not consulted here.
	repoPath string
}

// GetStatus maps internal/query.StatusResult onto uiv1.GetStatusResponse
// (D-06), mirroring internal/query/status.go's own "table of what each
// field becomes and why" doc-comment discipline:
//
//	StatusResult field | GetStatusResponse field | Notes
//	--------------------|--------------------------|------
//	Initialized         | initialized              | Unchanged — Status only runs on a successfully opened Engine in this plan; SRV-04's degrade path (a later plan) is the first caller that can answer initialized=false
//	Version             | version                  | Unchanged — already a schema-version string (status.go's own D-06 mapping)
//	NodeCount           | node_count               | Unchanged
//	EdgeCount           | edge_count               | Unchanged
//	FileCount           | file_count               | Unchanged
//	Stale               | stale                    | Unchanged (D-04a)
//
// Every other StatusResult field (ProjectPath, IndexPath, DbSizeBytes,
// Backend, NodesByKind, EdgesByKind, FilesByLanguage, Languages,
// PendingChanges, WorktreeMismatch, Index) is out of this tracer's
// bounded GetStatusResponse scope and is not projected.
func (s *uiService) GetStatus(ctx context.Context, _ *connect.Request[uiv1.GetStatusRequest]) (*connect.Response[uiv1.GetStatusResponse], error) {
	var resp *uiv1.GetStatusResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Status(ctx)
		if err != nil {
			return err
		}
		resp = &uiv1.GetStatusResponse{
			Initialized: result.Initialized,
			Version:     result.Version,
			NodeCount:   result.NodeCount,
			EdgeCount:   result.EdgeCount,
			FileCount:   result.FileCount,
			Stale:       result.Stale,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
