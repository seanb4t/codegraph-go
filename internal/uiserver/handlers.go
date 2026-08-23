package uiserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
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

// mapEngineError is withEngine's single error-translation site. It
// classifies by errors.Is/errors.As against exported sentinels ONLY —
// never by matching on an error's message text, the discipline
// internal/cli/serve.go's own comment records so a permission error can
// never masquerade as something else (a message change must never
// silently reclassify a response):
//
//   - query.ErrNotFound or graphstore.ErrNotFound -> connect.CodeNotFound
//   - query.ErrInvalidArgument -> connect.CodeInvalidArgument
//   - anything else -> connect.CodeInternal
//
// graphstore.ErrStoreLocked's mapping to connect.CodeUnavailable with a
// typed IndexingInProgress detail (SRV-04's degrade path, D-14) is plan
// 01-11's deliverable — deliberately NOT partially implemented here,
// since a half-done CodeUnavailable branch is something 01-11 would have
// to unpick rather than build on.
func mapEngineError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, query.ErrNotFound), errors.Is(err, graphstore.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, query.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
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

// statusToProto maps internal/query.StatusResult onto uiv1.GetStatusResponse
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
//	(none — sourced separately) | commit_sha        | NOT a StatusResult field: read via (*query.Engine).IndexMeta + schema.IndexedCommitSHA inside the SAME withEngine call GetStatus already opens for Status, per ENG-04/D-05/D-06. Empty means unknown (pre-upgrade graph or non-git checkout), never an error.
//
// Every other StatusResult field (ProjectPath, IndexPath, DbSizeBytes,
// Backend, NodesByKind, EdgesByKind, FilesByLanguage, Languages,
// PendingChanges, WorktreeMismatch, Index) is out of this tracer's
// bounded GetStatusResponse scope and is not projected.
func statusToProto(result query.StatusResult, commitSHA string) *uiv1.GetStatusResponse {
	return &uiv1.GetStatusResponse{
		Initialized: result.Initialized,
		Version:     result.Version,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		FileCount:   result.FileCount,
		Stale:       result.Stale,
		CommitSha:   commitSHA,
	}
}

// locationToProto maps internal/query.Location onto uiv1.Location
// field-for-field. The one named mapper every Location-bearing response
// (SearchResponse, and Task 3's Callers/Callees/Impact/Affected
// responses) reuses instead of an inline conversion per call site.
func locationToProto(l query.Location) *uiv1.Location {
	return &uiv1.Location{
		Name:      l.Name,
		Kind:      l.Kind,
		FilePath:  l.FilePath,
		StartLine: l.StartLine,
	}
}

// locationsToProto maps a slice of internal/query.Location onto their
// uiv1.Location wire projections, preserving order.
func locationsToProto(locs []query.Location) []*uiv1.Location {
	out := make([]*uiv1.Location, len(locs))
	for i, l := range locs {
		out[i] = locationToProto(l)
	}
	return out
}

// nodeToProto maps a schema.Node onto its uiv1.Node wire projection,
// field-for-field, mirroring ui.proto's Node message comment. No rpc in
// this plan returns a Node directly (Search/Callers/Callees/Impact/
// Affected all return Location, the lighter projection) — this mapper
// exists now, alongside Location's, so plan 01-09's GetNodeDetail has a
// single existing call site to reuse rather than inventing its own.
func nodeToProto(n *schema.Node) *uiv1.Node {
	if n == nil {
		return nil
	}
	return &uiv1.Node{
		Id:            n.Id,
		Kind:          n.Kind,
		Name:          n.Name,
		QualifiedName: n.QualifiedName,
		FilePath:      n.FilePath,
		Language:      n.Language,
		StartLine:     n.StartLine,
		EndLine:       n.EndLine,
		StartCol:      n.StartCol,
		EndCol:        n.EndCol,
		Signature:     n.Signature,
		Docstring:     n.Docstring,
		Visibility:    n.Visibility,
		IsExported:    n.IsExported,
		ReturnType:    n.ReturnType,
	}
}

// GetStatus answers the health/counts read plus, since ENG-04, the
// indexed commit SHA (D-05/D-06). Both eng.Status and eng.IndexMeta run
// inside the SAME withEngine call — one open serves both reads, since
// Engine.reader is unexported and a second graphstore.Open on the same
// directory is lock-refused (see (*query.Engine).IndexMeta's doc
// comment). See statusToProto's mapping table for field-by-field detail.
func (s *uiService) GetStatus(ctx context.Context, _ *connect.Request[uiv1.GetStatusRequest]) (*connect.Response[uiv1.GetStatusResponse], error) {
	var resp *uiv1.GetStatusResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Status(ctx)
		if err != nil {
			return err
		}
		meta, err := eng.IndexMeta()
		if err != nil {
			return err
		}
		commitSHA, _ := schema.IndexedCommitSHA(meta)
		resp = statusToProto(result, commitSHA)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// Search answers internal/query.Engine.Search's Location matches over
// the wire, unchanged: term/kind/limit pass straight through to the
// Engine, whose own validateLimit/MaxLimit already bound them for every
// caller (a second copy at this layer would be a driftable duplicate of
// the same rule).
func (s *uiService) Search(ctx context.Context, req *connect.Request[uiv1.SearchRequest]) (*connect.Response[uiv1.SearchResponse], error) {
	var resp *uiv1.SearchResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		locs, err := eng.Search(req.Msg.GetTerm(), req.Msg.GetKind(), int(req.Msg.GetLimit()))
		if err != nil {
			return err
		}
		resp = &uiv1.SearchResponse{Locations: locationsToProto(locs)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// fileEntryToProto maps internal/query.FileEntry onto uiv1.FileEntry
// field-for-field.
func fileEntryToProto(e query.FileEntry) *uiv1.FileEntry {
	return &uiv1.FileEntry{
		Path:      e.Path,
		Language:  e.Language,
		NodeCount: e.NodeCount,
		EdgeCount: e.EdgeCount,
	}
}

// fileEntriesToProto maps a slice of internal/query.FileEntry onto their
// uiv1.FileEntry wire projections, preserving order.
func fileEntriesToProto(entries []query.FileEntry) []*uiv1.FileEntry {
	out := make([]*uiv1.FileEntry, len(entries))
	for i, e := range entries {
		out[i] = fileEntryToProto(e)
	}
	return out
}

// fileTreeNodeToProto recursively maps internal/query.FileTreeNode onto
// its uiv1.FileTreeNode wire projection, preserving the directory/leaf
// asymmetry (a directory carries children and no path/language; a leaf
// carries path/language and no children) exactly as the Go type already
// does — this mapper introduces no depth limit of its own: the Engine's
// own validateFilesDepth already bounds how deep a tree Files can ever
// return, so a second limit here would be dead code, not a safety net.
func fileTreeNodeToProto(n *query.FileTreeNode) *uiv1.FileTreeNode {
	if n == nil {
		return nil
	}
	out := &uiv1.FileTreeNode{
		Name:     n.Name,
		IsDir:    n.IsDir,
		Path:     n.Path,
		Language: n.Language,
	}
	if len(n.Children) > 0 {
		out.Children = make([]*uiv1.FileTreeNode, len(n.Children))
		for i, c := range n.Children {
			out.Children[i] = fileTreeNodeToProto(c)
		}
	}
	return out
}

// fileTreeToProto maps a slice of internal/query.FileTreeNode (a tree
// level, starting at the root's children) onto their wire projections.
func fileTreeToProto(nodes []*query.FileTreeNode) []*uiv1.FileTreeNode {
	out := make([]*uiv1.FileTreeNode, len(nodes))
	for i, n := range nodes {
		out[i] = fileTreeNodeToProto(n)
	}
	return out
}

// Files answers internal/query.Engine.Files' union result over the wire,
// preserving BOTH possible shapes rather than flattening them: a "flat"
// FilesResult populates only FilesResponse.files, a "tree" FilesResult
// populates only FilesResponse.tree, and format names which one a client
// should read — the same union contract internal/query/files.go's
// FilesResult already documents, carried onto the wire unchanged.
// pattern/filter/dir/depth/format pass straight through: Engine.Files'
// own validateFilesDepth and format rejection already bound them for
// every caller.
func (s *uiService) Files(ctx context.Context, req *connect.Request[uiv1.FilesRequest]) (*connect.Response[uiv1.FilesResponse], error) {
	var resp *uiv1.FilesResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Files(query.FilesOptions{
			Pattern: req.Msg.GetPattern(),
			Filter:  req.Msg.GetFilter(),
			Dir:     req.Msg.GetDir(),
			Depth:   int(req.Msg.GetDepth()),
			Format:  req.Msg.GetFormat(),
		})
		if err != nil {
			return err
		}
		resp = &uiv1.FilesResponse{
			Format: result.Format,
			Files:  fileEntriesToProto(result.Files),
			Tree:   fileTreeToProto(result.Tree),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
