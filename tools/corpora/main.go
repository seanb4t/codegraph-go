// Command corpora is the single resolution path both the Taskfile fetch
// targets and (in a later plan) CI read for the corpora manifest, plus
// the measurement instrument Plan 01-05 builds: given -mode, it prints
// the resolved out-of-tree corpus root (root), a pre-validated JSON array
// of fetchable entries (entries), the sorted query.RankEdges members
// (kinds), the deterministic threshold/locked-set proposal derived from
// an observations file (select), or drives this repository's own
// indexer + query engine in-process to produce a measured Observation
// per corpus and upsert it into corpora/observations.json (measure).
//
// Neither root nor entries ever emits a value that has not already
// passed internal/corpora's strict allowlist — the Taskfile's bash
// re-validates independently at the interpolation point anyway, but this
// program's output is never itself the unvalidated source of a manifest
// field.
//
// measure mode has NO write path to corpora/selection.json: that file is
// hand-authored curated policy (thresholds, threshold rationale, the
// locked set, the rejected-candidate ledger), and this package's own
// tooling must never be able to overwrite it — see
// internal/corpora/record.go's package doc.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// manifestPath is the sole pin authority this program reads (D-09),
// relative to the repo root every Taskfile target invokes it from. A
// package-level var, not a const, so measure_test.go can point it at a
// temporary fixture manifest without depending on process cwd.
var manifestPath = "corpora/manifest.json"

// defaultObservationsPath is measure mode's default read-modify-write
// target and select mode's default read-only source. A non-default -out
// is what the smoke path uses so a scratch run never touches the
// committed file.
const defaultObservationsPath = "corpora/observations.json"

// entryOutput is one entries-mode array element: the fields the bash
// fetch loop needs (repo, sha) plus the two derived, already-computed
// values (slug, dir) so the Taskfile never re-derives a destination path
// on its own.
type entryOutput struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
	Slug string `json:"slug"`
	Dir  string `json:"dir"`
}

// selectOutput is -mode select's sole stdout contract: exactly these
// three keys, both arrays sorted for byte-stable comparison.
type selectOutput struct {
	MinEdgesPerKind map[string]int64 `json:"minEdgesPerKind"`
	LockedSet       []string         `json:"lockedSet"`
	Eligible        []string         `json:"eligible"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main's testable core: it never calls os.Exit itself.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("corpora", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mode := fs.String("mode", "", `output mode: "root", "entries", "measure", "select", or "kinds" (required)`)
	locked := fs.Bool("locked", false, "entries mode: restrict output to locked entries only")
	repos := fs.String("repos", "", "measure mode: comma-separated org/name list to measure exactly (overrides -scope)")
	scope := fs.String("scope", "locked", `measure mode: "locked" (default) or "all" — ignored when -repos is set`)
	out := fs.String("out", defaultObservationsPath, "measure mode: observations file to read-modify-write")
	in := fs.String("in", defaultObservationsPath, "select mode: observations file to read; never written")
	prune := fs.Bool("prune", false, "measure mode: drop observations for entries no longer named in the manifest (default off; corpora:drift never passes this)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	switch *mode {
	case "root":
		return runRoot(stdout, stderr)
	case "entries":
		return runEntries(stdout, stderr, *locked)
	case "measure":
		return runMeasure(stdout, stderr, *repos, *scope, *out, *prune)
	case "select":
		return runSelect(stdout, stderr, *in)
	case "kinds":
		return runKinds(stdout)
	default:
		fmt.Fprintf(stderr, "tools/corpora: -mode must be \"root\", \"entries\", \"measure\", \"select\", or \"kinds\" (got %q)\n", *mode)
		return 2
	}
}

// runRoot prints the resolved corpus root and nothing else.
func runRoot(stdout, stderr io.Writer) int {
	root, err := corpora.CorpusRoot()
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: resolve corpus root: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, root)
	return 0
}

// runEntries loads and validates the manifest, then prints a JSON array
// of entryOutput — all entries, or only locked ones when lockedOnly is
// set. A validation failure exits non-zero with a message on stderr,
// never a partial or unvalidated listing.
func runEntries(stdout, stderr io.Writer, lockedOnly bool) int {
	m, err := corpora.Load(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}
	root, err := corpora.CorpusRoot()
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: resolve corpus root: %v\n", err)
		return 1
	}

	entries := m.Corpora
	if lockedOnly {
		entries = corpora.LockedEntries(m)
	}

	out := make([]entryOutput, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryOutput{
			Repo: e.Repo,
			SHA:  e.SHA,
			Slug: e.Slug(),
			Dir:  e.Dir(root),
		})
	}

	enc := json.NewEncoder(stdout)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(stderr, "tools/corpora: encode entries: %v\n", err)
		return 1
	}
	return 0
}

// runKinds prints the members of query.RankEdges, one per line, sorted —
// the sole authority every shell/Python verify in this phase derives the
// ranked-kind set from, rather than restating the nine strings.
func runKinds(stdout io.Writer) int {
	kinds := make([]string, 0, len(query.RankEdges))
	for k := range query.RankEdges {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintln(stdout, k)
	}
	return 0
}

// runMeasure resolves the requested scope, measures each entry by
// driving this repository's own indexer and query engine in-process
// against its already-fetched, integrity-checked corpus directory, and
// UPSERTS the results into the observations file named by out: entries
// within scope are replaced, entries outside it are left untouched, and
// nothing is deleted unless prune is set. A missing or integrity-failing
// corpus is a loud, named, non-zero failure for the whole run — never a
// silently omitted observation.
func runMeasure(stdout, stderr io.Writer, reposFlag, scope, out string, prune bool) int {
	m, err := corpora.Load(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}

	entries, err := resolveMeasureScope(m, reposFlag, scope)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}

	root, err := corpora.CorpusRoot()
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: resolve corpus root: %v\n", err)
		return 1
	}

	measured := make(map[string]corpora.Observation, len(entries))
	for _, e := range entries {
		dir := e.Dir(root)
		trackedFiles, err := verifyAndCountTrackedFiles(dir, e.SHA)
		if err != nil {
			fmt.Fprintf(stderr, "tools/corpora: measure %s: %v\n", e.Repo, err)
			return 1
		}
		obs, err := measureOne(e, dir, trackedFiles)
		if err != nil {
			fmt.Fprintf(stderr, "tools/corpora: measure %s: %v\n", e.Repo, err)
			return 1
		}
		measured[corpora.ObservationKey(e.Repo, e.SHA)] = obs
	}

	existing, err := loadOrEmptyObservations(out)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}
	for k, v := range measured {
		existing.Observations[k] = v
	}
	if prune {
		keep := make(map[string]bool, len(m.Corpora))
		for _, e := range m.Corpora {
			keep[corpora.ObservationKey(e.Repo, e.SHA)] = true
		}
		for k := range existing.Observations {
			if !keep[k] {
				delete(existing.Observations, k)
			}
		}
	}
	existing.ManifestPath = manifestPath
	if existing.SchemaVersion == 0 {
		existing.SchemaVersion = 1
	}

	if err := writeObservations(out, existing); err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}

	// Render the human-readable document — only when -out is at its
	// default, so the smoke path with -out /tmp/... writes no committed
	// document. This is what keeps the smoke verifies in this plan from
	// regenerating the committed document from a one-entry scratch file.
	if out == defaultObservationsPath {
		sel, selErr := corpora.LoadSelection("corpora/selection.json")
		if selErr != nil && !errors.Is(selErr, os.ErrNotExist) {
			// A real read error (permission, malformed JSON) is a failure;
			// a missing selection file is expected during this plan and
			// early Plan 01-06 — proceed with a zero-value Selection.
			fmt.Fprintf(stderr, "tools/corpora: load selection for prose: %v\n", selErr)
			return 1
		}
		prose := renderMeasurementProse(existing, sel)
		docPath := "docs/CORPUS-MEASUREMENT.md"
		if err := os.MkdirAll("docs", 0o755); err != nil {
			fmt.Fprintf(stderr, "tools/corpora: mkdir docs: %v\n", err)
			return 1
		}
		if err := os.WriteFile(docPath, []byte(prose), 0o644); err != nil {
			fmt.Fprintf(stderr, "tools/corpora: write %s: %v\n", docPath, err)
			return 1
		}
	}

	fmt.Fprintf(stdout, "tools/corpora: measured %d entries, wrote %s\n", len(entries), out)
	return 0
}

// resolveMeasureScope resolves which manifest entries runMeasure
// measures, in precedence order: -repos names entries exactly (an
// unknown name exits loudly, naming it — this is what makes D-17's
// bounded, one-candidate-at-a-time search possible, since before any
// selection exists the locked set is empty and "all" would demand every
// candidate be present); otherwise -scope "locked" (the default), which
// refuses loudly on an empty locked set rather than silently no-op'ing;
// otherwise -scope "all".
func resolveMeasureScope(m corpora.Manifest, reposFlag, scope string) ([]corpora.Entry, error) {
	if reposFlag != "" {
		byRepo := make(map[string]corpora.Entry, len(m.Corpora))
		for _, e := range m.Corpora {
			byRepo[e.Repo] = e
		}
		names := strings.Split(reposFlag, ",")
		out := make([]corpora.Entry, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			e, ok := byRepo[name]
			if !ok {
				return nil, fmt.Errorf("-repos names %q, which has no entry in %s", name, manifestPath)
			}
			out = append(out, e)
		}
		return out, nil
	}

	switch scope {
	case "", "locked":
		lockedEntries := corpora.LockedEntries(m)
		if len(lockedEntries) == 0 {
			return nil, fmt.Errorf("the locked corpus set in %s is empty — nothing to measure; pass -repos or -scope all", manifestPath)
		}
		return lockedEntries, nil
	case "all":
		if len(m.Corpora) == 0 {
			return nil, fmt.Errorf("%s has no entries to measure", manifestPath)
		}
		return m.Corpora, nil
	default:
		return nil, fmt.Errorf("-scope must be \"locked\" or \"all\" (got %q)", scope)
	}
}

// verifyAndCountTrackedFiles confirms dir exists and its checked-out
// HEAD equals the pinned sha — the loud-failure gate a missing or
// wrong-commit corpus must hit before this program ever attempts to
// index it — then returns `git ls-files | wc -l`'s count: the
// tracked-file count of the fetched tree, NEVER the GitHub API
// repository size field (01-RESEARCH.md Pitfall 6).
func verifyAndCountTrackedFiles(dir, sha string) (int64, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return 0, fmt.Errorf("corpus directory %s is absent — run `task corpora:fetch-one` first", dir)
	}
	head, err := runGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return 0, fmt.Errorf("integrity check at %s: %w", dir, err)
	}
	if strings.TrimSpace(head) != sha {
		return 0, fmt.Errorf("integrity check at %s: HEAD %q does not match the pinned sha %q", dir, strings.TrimSpace(head), sha)
	}
	tracked, err := runGit(dir, "ls-files")
	if err != nil {
		return 0, fmt.Errorf("git ls-files at %s: %w", dir, err)
	}
	return countTrackedFiles(tracked), nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func countTrackedFiles(lsFilesOutput string) int64 {
	trimmed := strings.TrimRight(lsFilesOutput, "\n")
	if trimmed == "" {
		return 0
	}
	return int64(strings.Count(trimmed, "\n") + 1)
}

// measureOne indexes dir fresh into a throwaway temporary store — never
// touching dir's own checkout — then drives the SAME query.Engine.Status
// instrument `codegraph status --json --all-kinds` uses: densify the
// edge-count map (D-04), marshal it through the identical --json
// encoding path, decode, and strip volatile keys (corpora.StripVolatile)
// before this measurement is ever eligible to be committed.
func measureOne(e corpora.Entry, dir string, trackedFiles int64) (corpora.Observation, error) {
	storeDir, err := os.MkdirTemp("", "corpora-measure-*")
	if err != nil {
		return corpora.Observation{}, fmt.Errorf("MkdirTemp: %w", err)
	}
	defer os.RemoveAll(storeDir)

	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		return corpora.Observation{}, fmt.Errorf("indexer.Run(%s): %w", dir, err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		return corpora.Observation{}, fmt.Errorf("graphstore.Open: %w", err)
	}
	defer store.Close()

	reader, err := store.Snapshot()
	if err != nil {
		return corpora.Observation{}, fmt.Errorf("store.Snapshot: %w", err)
	}
	defer reader.Close()

	eng := query.NewWithRoot(reader, dir)
	result, err := eng.Status(context.Background())
	if err != nil {
		return corpora.Observation{}, fmt.Errorf("Status(%s): %w", dir, err)
	}
	result.EdgesByKind = query.DenseEdgesByKind(result)

	data, err := query.MarshalStatusJSON(result)
	if err != nil {
		return corpora.Observation{}, fmt.Errorf("marshal status for %s: %w", dir, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return corpora.Observation{}, fmt.Errorf("decode status JSON: %w", err)
	}

	return corpora.Observation{
		Repo:         e.Repo,
		SHA:          e.SHA,
		License:      e.License,
		Language:     e.Language,
		TrackedFiles: trackedFiles,
		Status:       corpora.StripVolatile(raw),
	}, nil
}

// loadOrEmptyObservations returns the observations already at path, or a
// fresh empty document when path does not exist yet. A path that exists
// but fails to decode is a loud error, never silently treated as absent.
func loadOrEmptyObservations(path string) (corpora.Observations, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return corpora.Observations{SchemaVersion: 1, ManifestPath: manifestPath, Observations: map[string]corpora.Observation{}}, nil
		}
		return corpora.Observations{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return corpora.LoadObservations(path)
}

// writeObservations marshals obs with stable indentation — encoding/json
// sorts every map[string]T's keys, so two runs over identical inputs
// produce byte-identical files — and writes it to path with a trailing
// newline.
func writeObservations(path string, obs corpora.Observations) error {
	data, err := json.MarshalIndent(obs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal observations: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// runSelect reads the observations file named by in (never writing it or
// any other file), derives the eligible universe, computes thresholds
// and the locked-set proposal, and prints exactly one JSON object to
// stdout.
func runSelect(stdout, stderr io.Writer, in string) int {
	obs, err := corpora.LoadObservations(in)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}

	eligible := eligibleObservationKeys(obs)
	th := corpora.ComputeThresholds(obs, eligible)
	lockedSet, err := corpora.SelectLockedSet(obs, th, eligible)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	if err := enc.Encode(selectOutput{MinEdgesPerKind: th, LockedSet: lockedSet, Eligible: eligible}); err != nil {
		fmt.Fprintf(stderr, "tools/corpora: encode select output: %v\n", err)
		return 1
	}
	return 0
}

// eligibleObservationKeys derives select's ELIGIBLE universe: every
// observation whose own repo/sha/license/language fields would
// themselves pass corpora.Validate if they were a manifest entry. This
// is independent of whether that repo@sha happens to already be present
// in the live corpora/manifest.json, which is what makes select's
// contract testable against a committed, self-contained fixture
// (testdata/corpora/observations.fixture.json) rather than requiring
// coordination with the manifest's current, still-changing content.
func eligibleObservationKeys(obs corpora.Observations) []string {
	var eligible []string
	for key, o := range obs.Observations {
		candidate := corpora.Manifest{Corpora: []corpora.Entry{{
			Repo:     o.Repo,
			SHA:      o.SHA,
			License:  o.License,
			Language: o.Language,
		}}}
		if corpora.Validate(candidate) == nil {
			eligible = append(eligible, key)
		}
	}
	sort.Strings(eligible)
	return eligible
}
