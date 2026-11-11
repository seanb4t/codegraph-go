package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/extract"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/routes"
	"github.com/seanb4t/codegraph-go/internal/indexer/spill"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
	"github.com/seanb4t/codegraph-go/internal/scratchdir"
)

const (
	// scratchSuffix follows the name of the store directory in the prefix
	// of the scratch directories of that store.
	scratchSuffix = ".scratch-"
	// stagingName is the name of the staging store inside a scratch
	// directory, and spillName that of the spill file.
	stagingName = "store"
	spillName   = "spill"
)

// runState is the state of a full index that lives until the run ends.
// Each part grows with the repository, but none holds the extraction
// result of a file: those are in the spill file, and in memory only while
// a worker holds the file.
type runState struct {
	// spill holds the extraction result of each record, at offsets[i]. A
	// record is a file, or the route result, which is the last record
	// when the run has one.
	spill   *spill.File
	offsets []int64
	// decls holds the declarations of each record.
	decls []symbols.Declarations

	// routeNodes and routeEdges hold what route detection found in each
	// file, by file index.
	routeNodes [][]*schema.Node
	routeEdges [][]*schema.Edge

	// structural holds the structural edges of each record, from pass 2.
	structural [][]*schema.Edge

	// units is the channel from pass 3 to the writer.
	units chan stagedUnit
	// noFile holds the edges of each record that belong to the no-file
	// unit, which the writer collapses once after pass 3.
	noFile []resolve.NoFileEdges
}

// stagedUnit is the finished unit of one record, on its way to the writer.
type stagedUnit struct {
	unit resolve.Unit
	// release gives back the token of the record. The writer calls it
	// once it staged the unit.
	release func()
}

// run is a full index: it builds a new store beside the live store at
// storeDir and puts it in the place of the live store in one atomic step.
// A run that fails or stops halfway leaves the live store as it was.
//
// The steps, each on forEachFile with the same number of workers:
//
//  1. Pass 1 extracts each file and writes the result to the spill file.
//  2. Route detection runs over the files of the languages whose detector
//     fired.
//  3. Barrier 1 builds the symbol view from the declarations of every
//     record, in file order.
//  4. Pass 2 resolves the types of each record and keeps its structural
//     edges.
//  5. Barrier 2 builds the hierarchy and runs the passes that need every
//     file (resolve.NewGlobal).
//  6. Pass 3 resolves each record in full, and sends its finished unit to
//     the writer.
//
// One writer goroutine stages the units into bounded batches. A record
// holds a token from the moment a worker claims it until the writer staged
// its unit, so at most Workers records hold working state at any time.
//
// run holds the writer lock of the store from its start to its end, so no
// sync commits to the live store during the build. A query still reads the
// live store until the exchange.
func run(ctx context.Context, repoRoot, storeDir string, opts Options) (stats Stats, err error) {
	start := time.Now()
	defer func() { stats.Duration = time.Since(start) }()
	// ENG-04/D-05: HEAD is resolved once for the run, before the work.
	headCommitSHA := resolveHeadCommitSHA(repoRoot)

	discovery, err := discover.DiscoverAll(repoRoot)
	if err != nil {
		return Stats{}, err
	}
	files := discovery.Files
	stats.Files = len(files)
	workers := opts.Workers
	if workers <= 0 {
		workers = DefaultWorkers()
	}

	// The lock files and the scratch directory live in the parent of the
	// store, so that parent must exist before any of them.
	storeDir = filepath.Clean(storeDir)
	parent := filepath.Dir(storeDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return stats, err
	}
	writerLock, err := graphstore.LockWriter(storeDir)
	if err != nil {
		return stats, err
	}
	defer writerLock.Close()

	// The three outcomes of the read of the prior coverage generation. A
	// held store fails the run. A store that was never indexed gives
	// generation 0. Any other error also gives generation 0, and the run
	// goes on with the error in its stats, because a full index is the
	// way to repair a store that cannot be read.
	var priorGeneration int64
	switch meta, metaErr := readStoreMeta(storeDir); {
	case metaErr == nil:
		priorGeneration = meta.GetCoverageGeneration()
	case errors.Is(metaErr, graphstore.ErrStoreLocked):
		return stats, metaErr
	case errors.Is(metaErr, graphstore.ErrNotFound):
	default:
		stats.PriorGenerationErr = metaErr
	}

	scratchPrefix := filepath.Base(storeDir) + scratchSuffix
	scratchdir.Sweep(parent, scratchPrefix)
	scratch, err := scratchdir.New(parent, scratchPrefix, false)
	if err != nil {
		return stats, err
	}
	// The scratch directory goes away on every return path. Before the
	// exchange it holds the new store, and after it the old one. Once the
	// new store is in place the run succeeded, so an error of the removal
	// then goes into the stats and does not fail the run.
	replaced := false
	defer func() {
		removeErr := scratch.Remove()
		if replaced {
			stats.AfterReplaceErr = errors.Join(stats.AfterReplaceErr, removeErr)
			return
		}
		err = errors.Join(err, removeErr)
	}()

	spillFile, err := spill.Create(filepath.Join(scratch.Path(), spillName))
	if err != nil {
		return stats, err
	}
	defer spillFile.Close()

	stagingDir := filepath.Join(scratch.Path(), stagingName)
	staging, err := graphstore.OpenStaging(stagingDir)
	if err != nil {
		return stats, err
	}
	built, buildErr := buildStore(ctx, staging, spillFile, repoRoot, discovery, workers, headCommitSHA, priorGeneration+1)
	// The Close of a staging store flushes it, so it comes before the
	// exchange, and its error fails the run.
	if err := errors.Join(buildErr, staging.Close()); err != nil {
		return stats, err
	}
	stats.Nodes, stats.Edges = built.nodes, built.edges
	stats.Unresolved, stats.Skipped = built.unresolved, built.skipped

	if err := graphstore.ReplaceStore(stagingDir, storeDir); err != nil {
		if !errors.Is(err, graphstore.ErrAfterReplace) {
			return stats, err
		}
		stats.AfterReplaceErr = err
	}
	replaced = true
	if onExchange != nil {
		onExchange()
	}
	return stats, nil
}

// readStoreMeta returns the metadata of the store at storeDir. It never
// creates a store: a missing store, and a store without a metadata record,
// both give graphstore.ErrNotFound.
func readStoreMeta(storeDir string) (*schema.Meta, error) {
	store, err := graphstore.OpenExisting(storeDir)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.GetMeta()
}

// built is what buildStore reports for the stats of the run.
type built struct {
	nodes, edges, unresolved, skipped int
}

// buildStore runs the passes and the writer of a full index over the files
// of discovery, and writes the graph into store. Pass 1, route detection
// and pass 2 run one after the other, each in the errgroup of its own
// forEachFile call. Pass 3 and the writer share one more errgroup. In each
// group the first error cancels the others, and buildStore returns once
// every goroutine ended.
func buildStore(
	ctx context.Context, store graphstore.GraphStore, spillFile *spill.File,
	repoRoot string, discovery discover.Discovery, workers int, commitSHA string, generation int64,
) (built, error) {
	files := discovery.Files
	st := &runState{
		spill:      spillFile,
		offsets:    make([]int64, len(files)),
		decls:      make([]symbols.Declarations, len(files)),
		routeNodes: make([][]*schema.Node, len(files)),
		routeEdges: make([][]*schema.Edge, len(files)),
	}
	var out built
	var skipped, unresolved atomic.Int64
	setup := parserSetup(defaultParserFactory)
	pass := func(step string) {
		if onPass != nil {
			onPass(step, st)
		}
	}

	// Pass 1, extract.
	pass(stepExtract)
	err := forEachFile(ctx, claimLoop{step: stepExtract, state: st}, len(files), workers, setup,
		func(_ context.Context, parsers *parserCache, i int, release func()) error {
			defer release()
			r, err := extractOne(parsers, files[i])
			if err != nil {
				return err
			}
			if r.Err != nil {
				// An internal error is a defect of the extractor, and it
				// fails the run before the record is written.
				if errors.Is(r.Err, goextract.ErrInternal) {
					return fmt.Errorf("extract %s: %w", r.RelPath, r.Err)
				}
				skipped.Add(1)
			}
			if st.offsets[i], err = spillFile.Append(&r); err != nil {
				return err
			}
			st.decls[i] = declarationOf(r)
			return nil
		})
	if err != nil {
		return out, err
	}
	out.skipped = int(skipped.Load())

	// Route detection.
	if err := detectRoutes(ctx, st, repoRoot, files, workers, setup); err != nil {
		return out, err
	}
	records := len(st.offsets)

	// Barrier 1: the declarations go into the builder in record order,
	// because the builder keeps candidates in the order it gets them.
	builder := symbols.NewBuilder()
	builder.AddResults(st.decls)
	view := builder.Freeze()
	// The view holds the nodes now. The rest of the declarations, such as
	// the contains edges and the arity maps, has no reader after this.
	st.decls = nil

	// Pass 2, structure.
	pass(stepStructure)
	st.structural = make([][]*schema.Edge, records)
	err = forEachFile(ctx, claimLoop{step: stepStructure, state: st}, records, workers, noSetup,
		func(_ context.Context, _ struct{}, i int, release func()) error {
			defer release()
			r, err := spillFile.Read(st.offsets[i])
			if err != nil {
				return err
			}
			st.structural[i] = resolve.StructuralEdges(resolve.Types(view, discovery.ModulePath, r).Edges())
			return nil
		})
	if err != nil {
		return out, err
	}

	// Barrier 2: the hierarchy, and the passes over the whole graph.
	var structural []*schema.Edge
	for _, edges := range st.structural {
		structural = append(structural, edges...)
	}
	st.structural = nil
	h := symbols.NewHierarchy(view, structural)
	global := resolve.NewGlobal(view, h, structural)
	if err := checkUnitsHaveRecords(global, files); err != nil {
		return out, err
	}

	// Pass 3, resolve and stage. The channel holds as many units as there
	// are tokens, so a worker that has its token never waits on its send:
	// a worker waits for a token, and the writer sets the pace.
	st.units = make(chan stagedUnit, min(workers, records))
	st.noFile = make([]resolve.NoFileEdges, records)
	tail := func(w graphstore.Writer, staged *stagedCounts) error {
		return stageTail(w, staged, view, h, global, st.noFile, discovery.Excluded, commitSHA, generation)
	}
	var counts stagedCounts
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		counts, err = writeUnits(gctx, store, st.units, global, tail)
		return err
	})
	g.Go(func() error {
		pass(stepResolve)
		err := forEachFile(gctx, claimLoop{step: stepResolve, state: st}, records, workers, noSetup,
			func(ctx context.Context, _ struct{}, i int, release func()) error {
				r, err := spillFile.Read(st.offsets[i])
				if err != nil {
					return err
				}
				t := resolve.Types(view, discovery.ModulePath, r)
				u, err := global.Assemble(t, resolve.Calls(view, h, r))
				if err != nil {
					return err
				}
				st.noFile[i] = u.NoFile
				unresolved.Add(int64(u.Unresolved))
				// The token goes back when the writer staged the unit,
				// so the body does not call release itself.
				select {
				case st.units <- stagedUnit{unit: u, release: release}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		if err != nil {
			// The channel stays open: the writer leaves through the
			// cancelled context, and must not take a closed channel for
			// the end of a good pass.
			return err
		}
		close(st.units)
		return nil
	})
	if err := g.Wait(); err != nil {
		return out, err
	}
	out.nodes, out.edges = counts.nodes, counts.edges
	out.unresolved = int(unresolved.Load())
	return out, nil
}

// checkUnitsHaveRecords returns an error if a whole-graph pass made an
// edge for a file that is not among files. Pass 3 writes the whole-graph
// edges of a file with the unit of that file, so such an edge would never
// reach the store. In a full index every file has a record, so this is a
// defect of the resolver, and the error wraps goextract.ErrInternal.
func checkUnitsHaveRecords(global *resolve.Global, files []discover.DiscoveredFile) error {
	paths := make(map[string]bool, len(files))
	for _, f := range files {
		paths[f.RelPath] = true
	}
	rest := global.Rest(func(path string) bool { return paths[path] })
	if len(rest) == 0 {
		return nil
	}
	e := rest[0]
	return fmt.Errorf("%w: %d edges belong to a file with no record; the first is %s -%s-> %s",
		goextract.ErrInternal, len(rest), e.Source, e.Kind, e.Target)
}

// noSetup is the setup of a forEachFile loop whose workers need no state.
func noSetup() (struct{}, func(), error) {
	return struct{}{}, func() {}, nil
}

// declarationOf returns the declarations of r, with the default visibility
// of its language.
func declarationOf(r goextract.FileResult) symbols.Declarations {
	spec, _ := extract.ByID(r.Language)
	return symbols.DeclarationsOf(r, spec.DefaultVisibility)
}

// detectRoutes runs the route detectors that fired over the files of their
// languages, and adds the route result to the state as one more record:
// its bytes go into the spill and its declarations into the list, like
// those of a file.
func detectRoutes(
	ctx context.Context, st *runState, repoRoot string, files []discover.DiscoveredFile, workers int,
	setup func() (*parserCache, func(), error),
) error {
	active := activeRouteDetectors(repoRoot, files)
	if len(active) == 0 {
		return nil
	}
	globalByLang := make(map[string]map[string]string, len(active))
	for lang := range active {
		globalByLang[lang] = buildGlobalHandlerIndex(st.decls, lang)
	}
	var targets []int
	for i, f := range files {
		if len(active[f.Language]) > 0 && st.decls[i].Err == nil {
			targets = append(targets, i)
		}
	}

	if onPass != nil {
		onPass(stepRoutes, st)
	}
	err := forEachFile(ctx, claimLoop{step: stepRoutes, state: st, files: targets}, len(targets), workers, setup,
		func(_ context.Context, parsers *parserCache, position int, release func()) error {
			defer release()
			i := targets[position]
			f := files[i]
			src, err := os.ReadFile(f.AbsPath)
			if err != nil {
				return nil
			}
			res, err := st.spill.Read(st.offsets[i])
			if err != nil {
				return err
			}
			p, err := parsers.get(f.Language)
			if err != nil {
				return err
			}
			st.routeNodes[i], st.routeEdges[i], err = detectRoutesInFile(p, f, src, &res, globalByLang[f.Language], active[f.Language])
			return err
		})
	if err != nil {
		return err
	}

	var nodes []*schema.Node
	var edges []*schema.Edge
	for i := range files {
		nodes = append(nodes, st.routeNodes[i]...)
		edges = append(edges, st.routeEdges[i]...)
	}
	// The route result below holds the nodes and edges now, so the lists
	// by file have no reader left.
	st.routeNodes, st.routeEdges = nil, nil
	if len(nodes) == 0 && len(edges) == 0 {
		return nil
	}
	routeResult := routeResultFrom(nodes, edges)
	off, err := st.spill.Append(&routeResult)
	if err != nil {
		return err
	}
	st.offsets = append(st.offsets, off)
	st.decls = append(st.decls, declarationOf(routeResult))
	return nil
}

// activeRouteDetectors returns, for each language among files, the route
// detectors whose signature the manifest of the repository matches.
func activeRouteDetectors(repoRoot string, files []discover.DiscoveredFile) map[string][]routes.Detector {
	languagesPresent := make(map[string]bool)
	for _, f := range files {
		languagesPresent[f.Language] = true
	}
	// The manifest of a language is read once, and each detector of that
	// language tests the cached text.
	manifestCache := make(map[string]string)
	active := make(map[string][]routes.Detector)
	for _, d := range routes.Registered() {
		if !languagesPresent[d.Language] {
			continue
		}
		spec, ok := extract.ByID(d.Language)
		if !ok || spec.NewParser == nil {
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
	return active
}

// stagedCounts are the record counts that the writer keeps for the
// metadata record.
type stagedCounts struct {
	nodes, edges int
	// packageNodes holds the id of each package node that a unit staged.
	// Several files name one package, and the metadata counts it once.
	packageNodes map[string]struct{}
}

// writeUnits is the body of the writer goroutine of a full index. It
// stages each unit from units into a bounded Writer of store, and calls
// the release of the unit once the unit is staged. When units closes, it
// stages the tail and commits. It takes the store as an interface, so a
// test can call it with a stub.
//
// writeUnits closes the Writer on every return path, so a run that fails
// releases its open batch and commits nothing more.
func writeUnits(
	ctx context.Context, store graphstore.GraphStore, units <-chan stagedUnit, global *resolve.Global,
	tail func(w graphstore.Writer, staged *stagedCounts) error,
) (stagedCounts, error) {
	counts := stagedCounts{packageNodes: make(map[string]struct{})}
	w, err := store.NewBoundedWriter(batchCap, onCommit)
	if err != nil {
		return counts, err
	}
	defer w.Close()
	for {
		select {
		case <-ctx.Done():
			return counts, ctx.Err()
		case u, ok := <-units:
			if !ok {
				if err := tail(w, &counts); err != nil {
					return counts, err
				}
				return counts, w.Commit()
			}
			g := resolve.Graph{Nodes: u.unit.Nodes, PackageNodes: u.unit.PackageNodes, Edges: u.unit.Edges}
			if u.unit.File != nil {
				g.Files = []*schema.File{u.unit.File}
			}
			if err := stage(w, g, global.OwnerPath); err != nil {
				return counts, err
			}
			counts.edges += len(u.unit.Edges)
			for _, pn := range u.unit.PackageNodes {
				counts.packageNodes[pn.Id] = struct{}{}
			}
			u.release()
		}
	}
}

// stageTail stages what follows the units of pass 3: the overload set
// nodes, the edges of the no-file unit, the excluded-file records and,
// last, the metadata record. The metadata is the last record of the run,
// so it is in the last batch, and a store that holds it holds every other
// record.
func stageTail(
	w graphstore.Writer, staged *stagedCounts, view symbols.View, h symbols.Hierarchy, global *resolve.Global,
	noFile []resolve.NoFileEdges, excluded []*schema.ExcludedFile, commitSHA string, generation int64,
) error {
	sets := h.SetNodes()
	noFileEdges := global.CollapseNoFile(noFile)
	if err := stage(w, resolve.Graph{Nodes: sets, Edges: noFileEdges}, global.OwnerPath); err != nil {
		return err
	}
	staged.edges += len(noFileEdges)

	sortedExcluded := append([]*schema.ExcludedFile(nil), excluded...)
	sort.Slice(sortedExcluded, func(i, j int) bool { return sortedExcluded[i].GetPath() < sortedExcluded[j].GetPath() })
	for _, x := range sortedExcluded {
		if err := w.PutExcludedFile(x); err != nil {
			return err
		}
	}

	// The node count is the number of distinct node ids: the nodes of the
	// view, the package nodes and the overload set nodes. Their ids come
	// from distinct kinds, so the three sets do not overlap. A count per
	// unit would count a node that several files emit once per file.
	viewNodes := 0
	for range view.Nodes() {
		viewNodes++
	}
	staged.nodes = viewNodes + len(staged.packageNodes) + len(sets)

	meta := schema.NewMeta()
	meta.NodeCount = int64(staged.nodes)
	meta.EdgeCount = int64(staged.edges)
	stampCapabilities(meta)
	// A full index writes the excluded-file records itself, so the store
	// has its coverage data (D-14).
	meta.HasCoverage = true
	// The generation rises by one with each commit of coverage data, so a
	// page token of an earlier generation can tell that the data changed
	// (D-13).
	meta.CoverageGeneration = generation
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	meta.CommitSha = commitSHA
	return w.PutMeta(meta)
}
