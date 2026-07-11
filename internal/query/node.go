package query

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
// given. file additionally disambiguates symbol when both are supplied
// (matching multiple same-named symbols to the one defined in file).
func (e *Engine) Node(symbol, file string) (string, error) {
	if symbol == "" {
		if file == "" {
			return "", fmt.Errorf("query: node requires a symbol name or a file path")
		}
		content, err := e.readSourceFile(file)
		if err != nil {
			return "", err
		}
		return renderNumberedSource(content), nil
	}

	node, err := e.resolveNodeForDetail(symbol, file)
	if err != nil {
		return "", err
	}

	it, err := e.reader.IterateEdges(node.Id)
	if err != nil {
		return "", err
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
			return "", err
		}
		calls = append(calls, target)
	}
	if err := it.Err(); err != nil {
		return "", err
	}

	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return "", err
	}
	var calledBy []*schema.Node
	for _, edge := range rev[node.Id] {
		src, err := e.reader.GetNode(edge.Source)
		if err != nil {
			// WR-04: skip a dangling reverse-edge source.
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return "", err
		}
		calledBy = append(calledBy, src)
	}

	return RenderNode(node, calls, calledBy), nil
}
