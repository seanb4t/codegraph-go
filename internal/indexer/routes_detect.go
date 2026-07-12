package indexer

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/routes"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// detectRoutes runs LANG-07's opt-in, per-framework route-detector
// registry (D-08/D-09) over repoRoot's already-discovered/-extracted
// files: for every language actually present, each registered
// routes.Detector's Signature is checked against that language's manifest
// text (read exactly once per language, never per detector — D-09), and
// only a language with at least one FIRED detector is re-parsed at all —
// a repo using none of the five priority frameworks costs one manifest
// read per present language and nothing else.
//
// Route nodes/edges are returned as plain slices, NOT wrapped in a
// synthetic goextract.FileResult, specifically so pipeline.go's caller can
// fold them directly into resolveRefsWithIndex's own nodes/edges
// accumulation (mirroring synthesizeGoImplements' append-after-the-main-
// loop shape in resolve.go) rather than minting a spurious, path-less
// schema.File record the way appending a fake FileResult would.
func detectRoutes(repoRoot string, files []DiscoveredFile, results []goextract.FileResult) ([]*schema.Node, []*schema.Edge, error) {
	detectors := routes.Registered()
	if len(detectors) == 0 {
		return nil, nil, nil
	}

	languagesPresent := make(map[string]bool)
	for _, f := range files {
		languagesPresent[f.Language] = true
	}

	manifestCache := make(map[string]string)
	active := make(map[string][]routes.Detector)
	for _, d := range detectors {
		if !languagesPresent[d.Language] {
			continue
		}
		text, cached := manifestCache[d.Language]
		if !cached {
			text = routeManifestText(d.Language, repoRoot)
			manifestCache[d.Language] = text
		}
		if d.Signature(text) {
			active[d.Language] = append(active[d.Language], d)
		}
	}
	if len(active) == 0 {
		return nil, nil, nil
	}

	resultByRelPath := make(map[string]*goextract.FileResult, len(results))
	for i := range results {
		resultByRelPath[results[i].RelPath] = &results[i]
	}

	globalByLang := make(map[string]map[string]string, len(active))
	for lang := range active {
		globalByLang[lang] = buildGlobalHandlerIndex(results, lang)
	}

	var nodes []*schema.Node
	var edges []*schema.Edge

	for _, f := range files {
		dets := active[f.Language]
		if len(dets) == 0 {
			continue
		}
		spec, ok := lookupLanguageByID(f.Language)
		if !ok || spec.NewParser == nil {
			continue
		}
		res := resultByRelPath[f.RelPath]
		if res == nil || res.Err != nil {
			// A file that failed Pass-1 extraction has no known
			// function/method nodes to resolve a handler against — skip
			// it rather than re-parsing for a route that could never
			// resolve anyway (D-06a: never a dangling edge).
			continue
		}

		src, err := os.ReadFile(f.AbsPath)
		if err != nil {
			// Matches every extractor's own per-file skip contract
			// (05-PATTERNS.md): one unreadable file never aborts the
			// whole detection pass.
			continue
		}

		routeNodes, routeEdges, err := detectRoutesInFile(spec, f, src, res, globalByLang[f.Language], dets)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, routeNodes...)
		edges = append(edges, routeEdges...)
	}

	return nodes, edges, nil
}

// detectRoutesInFile re-parses one file (via its own LanguageSpec's
// NewParser — the SAME grammar Pass 1 already used, Pitfall 1's per-
// language selection discipline) and runs every already-fired detector
// for that language's Walk over the resulting root node exactly once,
// converting each resolved routes.Route into a schema.Node/schema.Edge
// pair.
func detectRoutesInFile(spec LanguageSpec, f DiscoveredFile, src []byte, res *goextract.FileResult, globalIndex map[string]string, dets []routes.Detector) ([]*schema.Node, []*schema.Edge, error) {
	p, err := spec.NewParser()
	if err != nil {
		return nil, nil, err
	}
	defer p.Close()

	tree, err := p.Parse(src, nil)
	if err != nil {
		// A parse failure here mirrors every extractor's own skip
		// contract — Pass 1 already recorded (or will record) this
		// file's own extraction outcome; route detection simply
		// contributes nothing for it.
		return nil, nil, nil
	}
	defer tree.Close()

	native, ok := tree.Inner().(*tree_sitter.Tree)
	if !ok || native == nil {
		return nil, nil, nil
	}
	root := native.RootNode()
	if root == nil {
		return nil, nil, nil
	}

	resolver := newFileHandlerIndex(res, globalIndex)

	var nodes []*schema.Node
	var edges []*schema.Edge
	for _, d := range dets {
		for _, rt := range d.Walk(root, src, resolver) {
			if rt.HandlerID == "" {
				// D-06a: HandlerResolver's own contract already filters
				// this in every Detector's own Walk, but a defensive
				// second check here means a future detector's bug can
				// never produce a dangling edge past this point.
				continue
			}
			name := rt.HTTPMethod + " " + rt.Path
			qualifiedName := f.RelPath + "::route:" + rt.HTTPMethod + " " + rt.Path
			id := nodeid.NodeID(goextract.KindRoute, qualifiedName, f.RelPath)
			nodes = append(nodes, &schema.Node{
				Id:            id,
				Kind:          goextract.KindRoute,
				Name:          name,
				QualifiedName: qualifiedName,
				FilePath:      f.RelPath,
				Language:      f.Language,
				StartLine:     rt.Line,
				EndLine:       rt.Line,
				StartCol:      rt.Col,
			})
			edges = append(edges, &schema.Edge{
				Source:     id,
				Target:     rt.HandlerID,
				Kind:       goextract.RefKindCalls,
				Line:       rt.Line,
				Col:        rt.Col,
				Provenance: "heuristic",
				Metadata: map[string]string{
					"synthesizedBy": d.ID,
					"httpMethod":    rt.HTTPMethod,
					"routePath":     rt.Path,
				},
			})
		}
	}
	return nodes, edges, nil
}

// routeResultFrom wraps detectRoutes' plain node/edge slices into a
// single synthetic goextract.FileResult with RelPath left empty —
// resolveRefsWithIndex (resolve.go) special-cases RelPath=="" to skip
// minting a schema.File record for it while still folding its
// Nodes/IntraEdges into the SAME nodes/edges accumulation (and therefore
// the same collapseEdges/writeGraph commit) every other result flows
// through.
func routeResultFrom(nodes []*schema.Node, edges []*schema.Edge) goextract.FileResult {
	extractedNodes := make([]goextract.ExtractedNode, len(nodes))
	for i, n := range nodes {
		extractedNodes[i] = goextract.ExtractedNode{Node: n}
	}
	intraEdges := make([]goextract.IntraEdge, len(edges))
	for i, e := range edges {
		intraEdges[i] = goextract.IntraEdge{Edge: e}
	}
	return goextract.FileResult{
		Nodes:      extractedNodes,
		IntraEdges: intraEdges,
	}
}

// fileHandlerIndex implements routes.HandlerResolver against one file's
// own already-extracted function/method nodes (byLine/byName), falling
// back to a whole-language global index (built once per detection run by
// buildGlobalHandlerIndex) for a handler declared in a DIFFERENT file —
// the common case for Django's `path("x", views.some_view)` and any
// framework whose route-registration file is split from its handler
// module.
type fileHandlerIndex struct {
	byLine map[int32]string
	byName map[string]string
	global map[string]string
}

func newFileHandlerIndex(res *goextract.FileResult, global map[string]string) *fileHandlerIndex {
	idx := &fileHandlerIndex{
		byLine: make(map[int32]string),
		byName: make(map[string]string),
		global: global,
	}
	for _, en := range res.Nodes {
		if en.Node.Kind != goextract.KindFunction && en.Node.Kind != goextract.KindMethod {
			continue
		}
		idx.byLine[en.Node.StartLine] = en.Node.Id
		if existing, ok := idx.byName[en.Node.Name]; !ok || en.Node.Id < existing {
			idx.byName[en.Node.Name] = en.Node.Id
		}
	}
	return idx
}

func (h *fileHandlerIndex) ResolveByName(name string) (string, bool) {
	if id, ok := h.byName[name]; ok {
		return id, true
	}
	id, ok := h.global[name]
	return id, ok
}

func (h *fileHandlerIndex) ResolveByLine(line int32) (string, bool) {
	id, ok := h.byLine[line]
	return id, ok
}

// buildGlobalHandlerIndex composes a (name -> node id) index over every
// function/method node this language's own results declare, ANY file —
// the cross-file fallback fileHandlerIndex.ResolveByName consults. A
// same-name collision resolves deterministically (lowest node id wins),
// mirroring symbolIndex.addSymbol's own WR-01 tie-break — never last-
// write-wins, so the resolved handler is stable across runs regardless of
// file-processing order.
func buildGlobalHandlerIndex(results []goextract.FileResult, language string) map[string]string {
	out := make(map[string]string)
	for _, r := range results {
		if r.Err != nil || r.Language != language {
			continue
		}
		for _, en := range r.Nodes {
			if en.Node.Kind != goextract.KindFunction && en.Node.Kind != goextract.KindMethod {
				continue
			}
			if existing, ok := out[en.Node.Name]; !ok || en.Node.Id < existing {
				out[en.Node.Name] = en.Node.Id
			}
		}
	}
	return out
}

// routeManifestText reads language's own manifest file(s) at repoRoot,
// concatenated raw (never parsed) — exactly what every registered
// Detector.Signature for that language needs (D-09's opt-in gate) and
// nothing more. A missing/unreadable manifest degrades to an empty
// string, never an error (T-05-Manifest: accept, no crash — the same
// "descriptor absent" tolerance every languages_*.go descriptor reader
// already follows), which simply means no detector for that language
// fires.
func routeManifestText(language, root string) string {
	switch language {
	case "go":
		return readFileOrEmpty(filepath.Join(root, "go.mod"))
	case "java":
		return readFileOrEmpty(filepath.Join(root, "pom.xml")) +
			readFileOrEmpty(filepath.Join(root, "build.gradle")) +
			readFileOrEmpty(filepath.Join(root, "build.gradle.kts"))
	case "csharp":
		return csprojManifestText(root)
	case "python":
		return readFileOrEmpty(filepath.Join(root, "pyproject.toml")) +
			readFileOrEmpty(filepath.Join(root, "requirements.txt")) +
			readFileOrEmpty(filepath.Join(root, "Pipfile"))
	case "typescript", "tsx", "javascript":
		return readFileOrEmpty(filepath.Join(root, "package.json"))
	default:
		return ""
	}
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// csprojManifestText concatenates every *.csproj file under root (a C#
// project's csproj is commonly nested one or more directories below the
// repo root, unlike go.mod/package.json — languages_csharp.go's own
// readCSharpDescriptor only reads a root-level *.csproj for module
// identity, but a dependency signature like Microsoft.AspNetCore can live
// in ANY csproj in the tree, so this walk is intentionally broader),
// reusing ShouldSkipDir so vendor/.git/etc. are never descended into —
// bounded by the same walk discipline Discover itself uses.
func csprojManifestText(root string) string {
	var sb strings.Builder
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort manifest scan, never fails the caller
		}
		if d.IsDir() {
			if p != root && ShouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".csproj") {
			if data, err := os.ReadFile(p); err == nil {
				sb.Write(data)
				sb.WriteByte('\n')
			}
		}
		return nil
	})
	return sb.String()
}
