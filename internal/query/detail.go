package query

import (
	"fmt"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// NodeDetailMode discriminates which of NodeDetail's three pointer fields
// is populated (D-02): exactly one of File, Definition and Multi is
// non-nil for any given mode, matching the three shapes Node() can
// render — file-only source, a single definition, or an overloaded
// symbol's multiple definitions.
type NodeDetailMode int

const (
	// NodeDetailModeFile means File is populated (Definition and Multi are
	// nil) — Node()'s empty-symbol, file-only branch.
	NodeDetailModeFile NodeDetailMode = iota
	// NodeDetailModeSingleDef means Definition is populated (File and
	// Multi are nil) — a symbol that resolves to exactly one node.
	NodeDetailModeSingleDef
	// NodeDetailModeMultiDef means Multi is populated (File and
	// Definition are nil) — a symbol that resolves to more than one node.
	NodeDetailModeMultiDef
)

// NodeDetail is the structured ENG-01 seam Node() renders from (D-01,
// D-02, D-03): a plain Go sum type, not a generated protobuf message, so
// internal/query stays wire-agnostic and the RPC layer owns the mapping
// to wire messages. Mode identifies which of File, Definition and Multi
// is populated; the other two are nil.
type NodeDetail struct {
	Mode       NodeDetailMode
	File       *FileDetail
	Definition *DefinitionDetail
	Multi      *MultiDefDetail
}

// FileDetail is Node()'s file-only shape: the empty-symbol, file-path
// branch that reads a whole file's verbatim content via readSourceFile
// (renderNumberedSource's input).
type FileDetail struct {
	Path   string
	Source []byte
}

// DefinitionDetail is Node()'s single-definition shape — the exact
// argument tuple RenderNode already takes (node, calls, calledBy), plus
// Source for the multi-definition candidate case, where each candidate's
// verbatim body is read (renderNodeSection's input). The single-
// definition path never populates Source (RenderNode does not take one);
// a consumer that wants single-definition source asks for it explicitly
// through (*Engine).SourceFor rather than finding it here.
//
// Calls and CalledBy are exactly what fetchCalls/fetchCalledBy returned —
// nil stays nil, zero-length stays zero-length. The builders below never
// normalise these values (ENG-01): this type is a transcription of the
// fetchers' output, not a reinterpretation of it.
type DefinitionDetail struct {
	Node     *schema.Node
	Source   []byte
	Calls    []*schema.Node
	CalledBy []*schema.Node
}

// MultiDefDetail is Node()'s multi-definition shape, modelled as what it
// actually is: a LAZY per-candidate protocol, not a flat struct (D-02).
// RenderNodeMultiDef takes a fetch callback and invokes it lazily, only
// for matches below nodeMultiDefHardCap, with the body-budget decision
// interleaved into the same loop — a flat struct with one Node, one
// Calls and one CalledBy cannot represent that, and an eager gather over
// every match would perform I/O the current code never performs. Definition
// gathers exactly one candidate on demand, in the same source-then-calls-
// then-calledBy order the existing render closure uses, so a candidate the
// renderer never asks for is never read from disk and no readSourceFile
// error is surfaced where today there is none.
//
// MultiDefDetail imposes no cap of its own: Node()'s markdown budget
// (nodeMultiDefHardCap / nodeMultiDefBodyBudget) is RenderNodeMultiDef's
// business, and a wire consumer applies a different one.
type MultiDefDetail struct {
	Symbol     string
	Matches    []*schema.Node
	definition func(*schema.Node) (*DefinitionDetail, error)
}

// NewMultiDefDetail is MultiDefDetail's exported constructor, so a
// package outside internal/query (internal/uiserver) can declare mapper
// signatures over the type and construct one in its own tests.
func NewMultiDefDetail(symbol string, matches []*schema.Node, definition func(*schema.Node) (*DefinitionDetail, error)) *MultiDefDetail {
	return &MultiDefDetail{Symbol: symbol, Matches: matches, definition: definition}
}

// Definition gathers exactly one candidate's DefinitionDetail on demand —
// nothing is gathered until Definition is called, so a candidate the
// renderer skips is never read from disk. This is the property the whole
// lazy design exists to preserve.
func (m *MultiDefDetail) Definition(n *schema.Node) (*DefinitionDetail, error) {
	return m.definition(n)
}

// buildFileNodeDetail reads file fresh from disk through the existing
// repo-root-confined readSourceFile (T-03-06-Path) — no new read path —
// and returns it as a FileDetail. This is Node()'s file-only branch's
// entire gather step: there is nothing to gather beyond the read.
func (e *Engine) buildFileNodeDetail(file string) (*FileDetail, error) {
	content, err := e.readSourceFile(file)
	if err != nil {
		return nil, err
	}
	return &FileDetail{Path: file, Source: content}, nil
}

// buildSingleDefDetail performs exactly the body renderSingleDefNode
// performed before this extraction — fetchCalls, then the reverse
// adjacency (via the buildReverseAdjacency seam), then fetchCalledBy —
// and returns the populated struct with Source left nil. It returns
// whatever the fetchers returned without normalising: nil stays nil,
// zero-length stays zero-length (ENG-01).
func (e *Engine) buildSingleDefDetail(node *schema.Node) (*DefinitionDetail, error) {
	calls, err := e.fetchCalls(node)
	if err != nil {
		return nil, err
	}
	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return nil, err
	}
	calledBy, err := e.fetchCalledBy(node, rev)
	if err != nil {
		return nil, err
	}
	return &DefinitionDetail{Node: node, Calls: calls, CalledBy: calledBy}, nil
}

// buildMultiDefDetail builds the reverse adjacency ONCE (via the
// buildReverseAdjacency seam) and closes over it, exactly as
// renderMultiDefNode's closure already did — fetchCalledBy's own doc
// comment states why the map is shared: the multi-definition path fetches
// trails for up to nodeMultiDefHardCap candidates, and rebuilding the
// O(edges) map per candidate would multiply that scan. The returned
// MultiDefDetail's Definition gathers one candidate at a time, in
// source-then-calls-then-calledBy order — the same order
// renderMultiDefNode's fetch closure used — so which error surfaces
// first for a given candidate is unchanged.
func (e *Engine) buildMultiDefDetail(symbol string, matches []*schema.Node) (*MultiDefDetail, error) {
	rev, err := buildReverseAdjacency(e.reader)
	if err != nil {
		return nil, err
	}

	definition := func(n *schema.Node) (*DefinitionDetail, error) {
		source, err := e.readSourceFile(n.FilePath)
		if err != nil {
			return nil, err
		}
		calls, err := e.fetchCalls(n)
		if err != nil {
			return nil, err
		}
		calledBy, err := e.fetchCalledBy(n, rev)
		if err != nil {
			return nil, err
		}
		return &DefinitionDetail{Node: n, Source: source, Calls: calls, CalledBy: calledBy}, nil
	}

	return NewMultiDefDetail(symbol, matches, definition), nil
}

// buildNodeDetail reproduces Node()'s branch structure exactly — the
// empty-symbol file-only branch, the file-with-no-line exact-match fast
// path through resolveNodeForDetail, enumerateSymbolDefs, the not-found
// error, narrowNodeMatches, and the single-versus-multi split at
// len(matches) == 1 — returning the structured NodeDetail instead of
// rendering it. The branches are not reordered or merged and no error
// string differs from Node()'s own (NODE-04): the fast path exists
// specifically so every pre-existing exact-match caller gets byte-for-
// byte identical output, and the error strings are part of what the
// goldens and the CLI both observe.
//
// This is the ONE gather path per shape (D-01): Node() and
// (*Engine).NodeDetail both call this exact function, so CLI, MCP and UI
// cannot disagree about what a node is.
func (e *Engine) buildNodeDetail(symbol, file string, line *int) (NodeDetail, error) {
	if symbol == "" {
		if file == "" {
			return NodeDetail{}, fmt.Errorf("query: node requires a symbol name or a file path")
		}
		fd, err := e.buildFileNodeDetail(file)
		if err != nil {
			return NodeDetail{}, err
		}
		return NodeDetail{Mode: NodeDetailModeFile, File: fd}, nil
	}

	// NODE-04: file supplied with no line hint tries the pre-CR-02
	// exact-match single-def path first, returning immediately on
	// success — unchanged output for every existing exact-match caller.
	if file != "" && line == nil {
		if node, err := e.resolveNodeForDetail(symbol, file); err == nil {
			d, err := e.buildSingleDefDetail(node)
			if err != nil {
				return NodeDetail{}, err
			}
			return NodeDetail{Mode: NodeDetailModeSingleDef, Definition: d}, nil
		}
	}

	matches, err := e.enumerateSymbolDefs(symbol)
	if err != nil {
		return NodeDetail{}, err
	}
	if len(matches) == 0 {
		return NodeDetail{}, fmt.Errorf("query: symbol %q not found", symbol)
	}

	// NODE-03: narrow by substring file hint and/or line containment; a
	// no-op when neither hint is set (narrowNodeMatches returns matches
	// unchanged), preserving NODE-01/02's identical behavior for the
	// plain `node <symbol>` case.
	matches = narrowNodeMatches(matches, file, line)

	if len(matches) == 1 {
		d, err := e.buildSingleDefDetail(matches[0])
		if err != nil {
			return NodeDetail{}, err
		}
		return NodeDetail{Mode: NodeDetailModeSingleDef, Definition: d}, nil
	}

	md, err := e.buildMultiDefDetail(symbol, matches)
	if err != nil {
		return NodeDetail{}, err
	}
	return NodeDetail{Mode: NodeDetailModeMultiDef, Multi: md}, nil
}

// NodeDetail is the exported ENG-01 seam: the same gather Node() renders
// from (buildNodeDetail), returned structured and unrendered so a wire
// consumer (the UI RPC layer, from Phase 3) can map it without going
// through markdown. It is a plain Go type with no protobuf or wire
// dependency (D-03) — internal/query imports nothing from the UI wire
// layer.
func (e *Engine) NodeDetail(symbol, file string, line *int) (NodeDetail, error) {
	return e.buildNodeDetail(symbol, file, line)
}

// SourceFor reads relPath fresh from disk through the same repo-root-
// confined readSourceFile safety gate Node (file mode) and the multi-
// definition path already use (T-03-06-Path) — the only sanctioned way
// for a caller outside this package to obtain verbatim source. This is a
// wrapper over the existing confinement gate, not a second read path.
func (e *Engine) SourceFor(path string) ([]byte, error) {
	return e.readSourceFile(path)
}
