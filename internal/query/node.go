package query

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// resolveSourcePath resolves relPath against the Engine's repo root and
// confines it there (T-03-06-Path, V5): an absolute path, an empty path,
// or any path that filepath.Clean/filepath.Rel resolves outside repoRoot
// is rejected before any filesystem call is made. This is the single path
// safety gate Node (file mode) and Explore share.
//
// The string-level Clean/Rel check alone is not sufficient (WR-03): if
// relPath or an intermediate directory is a symlink pointing outside
// repoRoot, the string-level check sees only the in-repo path text and
// would let it through, while os.ReadFile silently follows the symlink to
// wherever it actually points. After the string-level check passes,
// filepath.EvalSymlinks resolves both the repo root and the candidate
// path to their real, symlink-free form and re-verifies confinement
// against those resolved paths (comparing resolved-to-resolved, not
// resolved-to-unresolved, since the repo root itself may sit under a
// symlinked path — e.g. macOS's /tmp -> /private/tmp).
func (e *Engine) resolveSourcePath(relPath string) (string, error) {
	if e.repoRoot == "" {
		return "", fmt.Errorf("query: engine has no repo root configured for source reads")
	}
	if relPath == "" {
		return "", fmt.Errorf("query: empty file path")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("query: absolute path %q is not allowed", relPath)
	}

	cleaned := filepath.Clean(relPath)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("query: path %q escapes the repo root", relPath)
	}

	root, err := filepath.Abs(e.repoRoot)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, cleaned)

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("query: path %q escapes the repo root", relPath)
	}

	// WR-03: re-verify confinement after resolving symlinks, so a
	// symlink inside the repo cannot point outside repoRoot.
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedAbs, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	resolvedRel, err := filepath.Rel(resolvedRoot, resolvedAbs)
	if err != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("query: path %q escapes the repo root", relPath)
	}

	return abs, nil
}

// readSourceFile reads relPath fresh from disk, confined to the repo root
// (resolveSourcePath) — the shared "read fresh from disk" primitive Node
// (file mode) and Explore both use (D-05a).
func (e *Engine) readSourceFile(relPath string) ([]byte, error) {
	abs, err := e.resolveSourcePath(relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

// generatedFilePatterns is the documented GENERATED_PATTERNS list,
// captured as a constant list (D-07, RESEARCH §7,
// extraction/generated-detection.js:27-82) — the primary sort key for
// NODE-01's multi-def enumeration (generated files last). Do not
// reorder, add, or drop entries without re-verifying against the frozen
// RESEARCH capture; this is a byte-for-byte transcription of the regex
// set, not an approximation.
var generatedFilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.pb\.go$`),
	regexp.MustCompile(`\.pulsar\.go$`),
	regexp.MustCompile(`_grpc\.pb\.go$`),
	regexp.MustCompile(`_mock\.go$`),
	regexp.MustCompile(`_mocks\.go$`),
	regexp.MustCompile(`^mock_[^/]+\.go$`),
	regexp.MustCompile(`\.generated\.[jt]sx?$`),
	regexp.MustCompile(`\.gen\.[jt]sx?$`),
	regexp.MustCompile(`\.pb\.[jt]s$`),
	regexp.MustCompile(`_pb\.[jt]s$`),
	regexp.MustCompile(`_grpc_pb\.[jt]s$`),
	regexp.MustCompile(`\.min\.m?js$`),
	regexp.MustCompile(`_pb2(_grpc)?\.py$`),
	regexp.MustCompile(`_pb2\.pyi$`),
	regexp.MustCompile(`\.pb\.(cc|h)$`),
	regexp.MustCompile(`\.g\.cs$`),
	regexp.MustCompile(`Grpc\.cs$`),
	regexp.MustCompile(`OuterClass\.java$`),
	regexp.MustCompile(`Grpc\.java$`),
	regexp.MustCompile(`\.pb\.swift$`),
	regexp.MustCompile(`\.g\.dart$`),
	regexp.MustCompile(`\.freezed\.dart$`),
	regexp.MustCompile(`\.pb\.dart$`),
	regexp.MustCompile(`\.pbgrpc\.dart$`),
	regexp.MustCompile(`\.chopper\.dart$`),
	regexp.MustCompile(`\.generated\.rs$`),
}

// isGeneratedFile reports whether filePath matches the documented
// generated-file predicate (D-07) — used as NODE-01's multi-def sort's primary key so
// hand-written definitions surface before generated/vendored duplicates
// sharing the same symbol name.
func isGeneratedFile(filePath string) bool {
	for _, p := range generatedFilePatterns {
		if p.MatchString(filePath) {
			return true
		}
	}
	return false
}

// enumerateSymbolDefs collects EVERY node whose Name equals symbol — a
// full IterateNodes scan, the same D-03 base resolveSymbolNode uses —
// instead of resolving to a single winner (NODE-01, RESEARCH §6
// findSymbolMatches). The result is sorted generated-files-last
// (isGeneratedFile, primary key) then lowest-Id-first (secondary key —
// a documented divergence from the documented implicit,
// non-deterministic row order per RESEARCH Pattern 2: the documented
// sort has no secondary tie-break, so this is an intentional Go-side
// determinism improvement, not a byte-for-byte transcription of the
// ordering).
func (e *Engine) enumerateSymbolDefs(symbol string) ([]*schema.Node, error) {
	it, err := e.reader.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var matches []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.Name == symbol {
			matches = append(matches, n)
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(matches, func(i, j int) bool {
		gi, gj := isGeneratedFile(matches[i].FilePath), isGeneratedFile(matches[j].FilePath)
		if gi != gj {
			return !gi // non-generated first
		}
		return matches[i].Id < matches[j].Id
	})
	return matches, nil
}

// normalizeNarrowPath canonicalizes a path for narrowNodeMatches's
// fileHint comparison (RESEARCH §9): backslash->slash, then lowercased,
// so a hint typed with the "wrong" separator or casing still matches.
func normalizeNarrowPath(p string) string {
	return strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
}

// absInt32 returns the absolute value of n.
func absInt32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

// narrowNodeMatches filters matches by an optional fileHint and/or
// lineHint (NODE-03, RESEARCH §9) — a PURE in-memory filter over the
// already-enumerated slice: it opens no file handle and performs no
// fresh disk read keyed on the raw hint (T-01-06 — the file-READ path
// stays behind resolveSourcePath/readSourceFile). fileHint matches when
// the normalized node path ends with or contains the normalized hint;
// the working set is only replaced when the filtered result is
// non-empty. lineHint first prefers a def whose [StartLine,EndLine]
// contains it, falling back to the single nearest def by
// |StartLine-lineHint| when none contains it. The final result is never
// empty as long as matches was non-empty — every stage is a
// best-effort narrowing, never a hard filter that can zero out the set.
func narrowNodeMatches(matches []*schema.Node, fileHint string, lineHint *int) []*schema.Node {
	if len(matches) <= 1 {
		return matches
	}
	if fileHint == "" && lineHint == nil {
		return matches
	}

	narrowed := matches
	if fileHint != "" {
		fh := normalizeNarrowPath(fileHint)
		var byFile []*schema.Node
		for _, n := range narrowed {
			np := normalizeNarrowPath(n.FilePath)
			if strings.HasSuffix(np, fh) || strings.Contains(np, fh) {
				byFile = append(byFile, n)
			}
		}
		if len(byFile) > 0 {
			narrowed = byFile // only replace if non-empty
		}
	}

	if lineHint != nil && len(narrowed) > 1 {
		line := int32(*lineHint)
		var containing []*schema.Node
		for _, n := range narrowed {
			end := n.EndLine
			if end == 0 {
				end = n.StartLine
			}
			if n.StartLine <= line && end >= line {
				containing = append(containing, n)
			}
		}
		if len(containing) > 0 {
			narrowed = containing
		} else {
			nearest := narrowed[0]
			best := absInt32(nearest.StartLine - line)
			for _, n := range narrowed[1:] {
				if d := absInt32(n.StartLine - line); d < best {
					nearest, best = n, d
				}
			}
			narrowed = []*schema.Node{nearest}
		}
	}

	if len(narrowed) > 0 {
		return narrowed // final guard: never assign an empty working set
	}
	return matches
}

// resolveNodeForDetail resolves symbol to a concrete node for Node's
// symbol-detail mode. When file is empty, it reuses traverse.go's
// resolveSymbolNode (D-03 full-scan, deterministic lowest-Id tie-break) —
// no duplicated resolution logic. When file is given (the CLI's `-f`
// disambiguation flag, matching the golden `node "mergeStyle" -f
// internal/cli/finish.go` command), it scans for nodes matching both name
// and file, picking the lexicographically lowest Id among ties for the
// same determinism guarantee.
func (e *Engine) resolveNodeForDetail(symbol, file string) (*schema.Node, error) {
	if file == "" {
		return e.resolveSymbolNode(symbol)
	}

	it, err := e.reader.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var candidates []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.Name == symbol && n.FilePath == file {
			candidates = append(candidates, n)
		}
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("query: symbol %q not found in file %q", symbol, file)
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Id < candidates[j].Id })
	return candidates[0], nil
}

// Node renders symbol detail (QRY-02, D-05b) when symbol is non-empty, or
// a line-numbered verbatim file read when symbol is empty and file is
// given. file additionally disambiguates symbol when both are supplied.
// line is an optional NODE-03 narrowing hint (RESEARCH §9).
//
// Node is a THIN WRAPPER over buildNodeDetail (D-01, ENG-01): it gathers
// exactly once via the single shared builder — the same one
// (*Engine).NodeDetail calls — then switches on the returned Mode and
// calls the matching existing PURE render function
// (renderNumberedSource / RenderNode / RenderNodeMultiDef through a thin
// adapter closure over the already-built MultiDefDetail). Node never
// gathers a second time and never diverges from what NodeDetail would
// return for the same arguments — CLI, MCP and UI cannot disagree about
// what a node is.
func (e *Engine) Node(symbol, file string, line *int) (string, error) {
	d, err := e.buildNodeDetail(symbol, file, line)
	if err != nil {
		return "", err
	}

	switch d.Mode {
	case NodeDetailModeFile:
		return renderNumberedSource(d.File.Source), nil
	case NodeDetailModeSingleDef:
		return RenderNode(d.Definition.Node, d.Definition.Calls, d.Definition.CalledBy), nil
	default:
		// NodeDetailModeMultiDef: adapt MultiDefDetail's lazy Definition
		// method to RenderNodeMultiDef's nodeSectionFetch shape, without
		// rebuilding the reverse adjacency a second time — d.Multi was
		// already built once by buildNodeDetail above.
		fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
			dd, err := d.Multi.Definition(n)
			if err != nil {
				return nil, nil, nil, err
			}
			return dd.Source, dd.Calls, dd.CalledBy, nil
		}
		return RenderNodeMultiDef(d.Multi.Symbol, d.Multi.Matches, fetch)
	}
}

// fetchCalls resolves node's forward "calls" edges into their target
// nodes, skipping a dangling target rather than aborting the render
// (WR-04) — shared by the single-def and multi-def render paths so the
// edge-read logic exists in exactly one place.
func (e *Engine) fetchCalls(node *schema.Node) ([]*schema.Node, error) {
	it, err := e.reader.IterateEdges(node.Id)
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var calls []*schema.Node
	for it.Next() {
		edge := it.Edge()
		if edge.Kind != goextract.RefKindCalls {
			continue
		}
		target, err := e.reader.GetNode(edge.Target)
		if err != nil {
			// WR-04: a dangling calls-edge target is skipped rather than
			// aborting the whole node detail render.
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		calls = append(calls, target)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return calls, nil
}

// fetchCalledBy resolves node's reverse callers from a pre-built
// reverse-adjacency map (BuildReverseAdjacency), skipping a dangling
// source rather than aborting the render (WR-04). rev is passed in
// (rather than rebuilt per call) so the multi-def path — which fetches
// call trails for up to HARD_CAP candidates — builds it once and shares
// it, instead of paying its O(edges) cost once per candidate.
func (e *Engine) fetchCalledBy(node *schema.Node, rev map[string][]*schema.Edge) ([]*schema.Node, error) {
	var calledBy []*schema.Node
	for _, edge := range rev[node.Id] {
		src, err := e.reader.GetNode(edge.Source)
		if err != nil {
			// WR-04: skip a dangling reverse-edge source.
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		calledBy = append(calledBy, src)
	}
	return calledBy, nil
}

// renderSingleDefNode renders node via the original single-def path
// (RenderNode) — reduced by this plan's extraction (D-01) to calling
// buildSingleDefDetail and then RenderNode(d.Node, d.Calls, d.CalledBy):
// same calls, same order, same error handling, one new structured return
// type in between. Node() no longer calls this directly (it gathers once
// via buildNodeDetail and renders inline to avoid a second gather for
// the same request), but the function is kept — unchanged in name and
// behavior — as a standalone single-def render entry point built on the
// same builder buildNodeDetail uses.
func (e *Engine) renderSingleDefNode(node *schema.Node) (string, error) {
	d, err := e.buildSingleDefDetail(node)
	if err != nil {
		return "", err
	}
	return RenderNode(d.Node, d.Calls, d.CalledBy), nil
}

// renderMultiDefNode renders NODE-02's multi-def markdown for an
// overloaded symbol (RenderNodeMultiDef) — reduced by this plan's
// extraction (D-01) to calling buildMultiDefDetail and then adapting its
// lazy Definition method to RenderNodeMultiDef's nodeSectionFetch shape.
// The render function and the nodeSectionFetch signature are untouched,
// and so is the laziness: a candidate RenderNodeMultiDef never asks for
// is never read from disk. Node() no longer calls this directly, for the
// same single-gather reason renderSingleDefNode's doc comment states.
func (e *Engine) renderMultiDefNode(symbol string, matches []*schema.Node) (string, error) {
	md, err := e.buildMultiDefDetail(symbol, matches)
	if err != nil {
		return "", err
	}

	fetch := func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
		d, err := md.Definition(n)
		if err != nil {
			return nil, nil, nil, err
		}
		return d.Source, d.Calls, d.CalledBy, nil
	}

	return RenderNodeMultiDef(symbol, matches, fetch)
}
