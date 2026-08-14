// record.go declares the two independent typed documents Plan 01-05
// builds on top of manifest.go's Entry/Manifest: Observations (GENERATED,
// upserted by task corpora:measure, keyed by repo@sha, never hand-edited)
// and Selection (CURATED, hand-authored policy — thresholds, threshold
// rationale, the locked set, the rejected-candidate ledger and any
// synthetic-coverage declarations — which this package's own tooling
// never writes). See 01-05-PLAN.md's "Why the record is TWO files, not
// one": a merge of the two into one generator-owned file is code that can
// be wrong, whereas "the generator has no write path to this file"
// cannot be.
package corpora

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// ---------------------------------------------------------------------
// Volatile-key stripping
// ---------------------------------------------------------------------

// goldenVolatileKeys mirrors testdata/golden/golden_test.go's
// volatileKeys exactly — the golden suite's own base policy for JSON
// keys that are non-deterministic across reindex runs.
var goldenVolatileKeys = map[string]bool{
	"score":       true,
	"lastIndexed": true,
	"dbSizeBytes": true,
}

// statusVolatileKeys are the four ADDITIONAL host-or-timing-specific
// query.StatusResult keys this package strips on top of
// goldenVolatileKeys. StripVolatile is a superset of the golden suite's
// policy — a strictly broader policy, not the same policy applied twice:
//
//   - projectPath, indexPath — host-local absolute filesystem paths; a
//     published, committed record must not leak a developer's or CI
//     runner's directory layout.
//   - worktreeMismatch — carries those same two absolute paths inside a
//     nested object whenever a mismatch is detected.
//   - stale — a point-in-time watcher/daemon signal with no meaning once
//     the measurement is committed to git.
//
// A reader who expects only the golden suite's three keys to disappear
// will be surprised these four vanish too, which is exactly why this is
// spelled out here rather than left implicit.
var statusVolatileKeys = map[string]bool{
	"projectPath":      true,
	"indexPath":        true,
	"worktreeMismatch": true,
	"stale":            true,
}

// isVolatileKey reports whether StripVolatile removes key: every name in
// goldenVolatileKeys or statusVolatileKeys, plus (mirroring the golden
// suite's own suffix rule) any key ending in "_at" or "At".
func isVolatileKey(key string) bool {
	if goldenVolatileKeys[key] || statusVolatileKeys[key] {
		return true
	}
	return strings.HasSuffix(key, "_at") || strings.HasSuffix(key, "At")
}

// StripVolatile returns a NEW map with every volatile key (isVolatileKey)
// removed, recursing into nested objects so a volatile key nested under
// "index" is removed too. m itself is never mutated. pendingChanges
// deliberately SURVIVES the strip: it is a deterministic all-zero
// placeholder today (query.PendingChanges), so removing it would shrink
// the record's schema for no reproducibility benefit — every measurement
// keeps the identical shape whether or not sync tracking is ever wired
// up.
func StripVolatile(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isVolatileKey(k) {
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			out[k] = StripVolatile(nested)
		} else {
			out[k] = v
		}
	}
	return out
}

// ---------------------------------------------------------------------
// Priority-4 languages
// ---------------------------------------------------------------------

// LanguageGroup names one FIXT-01 priority-4 coverage target and the
// concrete query.StatusResult.FilesByLanguage keys that count toward it.
// Four groups map to exactly one indexer language ID; TS/JS is the one
// exception, spanning THREE real per-file language IDs
// (internal/indexer/languages_typescript.go registers "typescript" and
// "javascript" as separate LanguageSpecs, and TSX source lands under
// "tsx" — see testdata/golden/parity_tsjs_test.go's identical
// tsjsLanguages grouping) because REQUIREMENTS.md counts them as one
// coverage target ("TS/JS").
type LanguageGroup struct {
	Name string
	Keys []string
}

// FileCount sums filesByLanguage's counts across g's constituent keys.
func (g LanguageGroup) FileCount(filesByLanguage map[string]int64) int64 {
	var total int64
	for _, k := range g.Keys {
		total += filesByLanguage[k]
	}
	return total
}

// PriorityLanguages is the five priority-4 language groups FIXT-01
// requires the locked corpus set to cover, declared once so
// ComputeThresholds/SelectLockedSet and the prose renderer (tools/corpora
// /prose.go) never hand-restate the group boundaries.
var PriorityLanguages = []LanguageGroup{
	{Name: "go", Keys: []string{"go"}},
	{Name: "java", Keys: []string{"java"}},
	{Name: "csharp", Keys: []string{"csharp"}},
	{Name: "python", Keys: []string{"python"}},
	{Name: "tsjs", Keys: []string{"typescript", "javascript", "tsx"}},
}

// ---------------------------------------------------------------------
// Generated side: Observation / Observations
// ---------------------------------------------------------------------

// Observation is one candidate's measured evidence: everything task
// corpora:measure recorded about repo@sha from a single indexing run. It
// is GENERATED — never hand-edited — and lives only inside Observations.
type Observation struct {
	// Repo is the GitHub org/name slug, matching a corpora/manifest.json
	// entry's Repo (manifest.go).
	Repo string `json:"repo"`
	// SHA is the pinned commit this observation measured, matching the
	// manifest entry's SHA.
	SHA string `json:"sha"`
	// License is copied from the manifest entry at measurement time, so
	// a reader of the observation alone can see the candidate's SPDX
	// identifier without cross-referencing the manifest.
	License string `json:"license"`
	// Language is the manifest entry's nominated Language — the
	// coverage gap this candidate was fetched to help close.
	Language string `json:"language"`
	// TrackedFiles is `git ls-files | wc -l` at the pinned SHA: the
	// tracked-file count of the fetched tree. NEVER the GitHub API
	// repository "size" field, which reports full-history packed size
	// and does not describe what a shallow `--depth 1` fetch at one
	// commit yields (01-RESEARCH.md Pitfall 6).
	TrackedFiles int64 `json:"trackedFiles"`
	// Status is `codegraph status --json --all-kinds`'s decoded output
	// for this corpus, after query.DenseEdgesByKind and then
	// StripVolatile: dense edgesByKind (every ranked kind with an
	// explicit value, plus any unranked kind carrying a positive count),
	// filesByLanguage, languages, the three top-level counts
	// (fileCount/nodeCount/edgeCount), backend, version and index
	// survive; every key StripVolatile names is gone.
	Status map[string]any `json:"status"`
}

// toInt64 coerces a decoded JSON number (float64, once round-tripped
// through LoadObservations) or a directly-constructed Go integer to
// int64. Any other value (nil, string, ...) reads as 0 rather than
// panicking — a malformed or absent field degrades to "not measured"
// instead of crashing the caller.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

// numericSubmap reads status[section] as a map and coerces every value
// to int64 via toInt64, tolerating either a freshly-constructed
// map[string]int64-shaped value or a JSON-decoded map[string]any of
// float64s.
func numericSubmap(status map[string]any, section string) map[string]int64 {
	raw, _ := status[section].(map[string]any)
	out := make(map[string]int64, len(raw))
	for k, v := range raw {
		out[k] = toInt64(v)
	}
	return out
}

// EdgeCount returns o's measured count for kind (a query.RankEdges
// member or an unranked kind), read from its stripped
// status.edgesByKind map. Absent kinds read as 0, matching "not
// measured" — callers that need to distinguish "measured zero" from
// "absent" should inspect o.Status directly.
func (o Observation) EdgeCount(kind string) int64 {
	return numericSubmap(o.Status, "edgesByKind")[kind]
}

// LanguageFileCount sums o's measured file counts across g's constituent
// filesByLanguage keys.
func (o Observation) LanguageFileCount(g LanguageGroup) int64 {
	return g.FileCount(numericSubmap(o.Status, "filesByLanguage"))
}

// ObservationKey joins repo and sha the same way Observations keys its
// map: repo, an at-sign, sha — matching the manifest's own repo@SHA
// identity form (D-09).
func ObservationKey(repo, sha string) string {
	return repo + "@" + sha
}

// Observations is the full corpora/observations.json document: every
// candidate ever measured, keyed by ObservationKey. Generated and
// UPSERTED by task corpora:measure — entries within a run's scope are
// replaced, entries outside it are left untouched, and nothing is
// deleted without an explicit -prune. Never hand-edited, never fully
// reconstructed.
type Observations struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ManifestPath  string                 `json:"manifestPath"`
	Observations  map[string]Observation `json:"observations"`
}

// ErrDuplicateObservationKey is returned by NewObservations when two
// input entries share an ObservationKey.
var ErrDuplicateObservationKey = errors.New("corpora: duplicate observation key")

// NewObservations builds an Observations from a slice, computing each
// entry's key via ObservationKey and returning ErrDuplicateObservationKey
// (naming the key) when two entries collide. A Go map literal cannot
// represent that collision on its own, so this constructor is what
// surfaces it for any caller assembling one Observation at a time — the
// shape both task corpora:measure's upsert loop and this package's own
// tests use.
func NewObservations(schemaVersion int, manifestPath string, obs []Observation) (Observations, error) {
	m := make(map[string]Observation, len(obs))
	for _, o := range obs {
		key := ObservationKey(o.Repo, o.SHA)
		if _, exists := m[key]; exists {
			return Observations{}, fmt.Errorf("%w: %q", ErrDuplicateObservationKey, key)
		}
		m[key] = o
	}
	return Observations{SchemaVersion: schemaVersion, ManifestPath: manifestPath, Observations: m}, nil
}

// LoadObservations reads path, decodes it as JSON into an Observations,
// and returns a non-nil error naming path when the file is absent or
// malformed — never an empty document a downstream guard would then
// vacuously pass over.
func LoadObservations(path string) (Observations, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Observations{}, fmt.Errorf("corpora: read %s: %w", path, err)
	}
	var o Observations
	if err := json.Unmarshal(data, &o); err != nil {
		return Observations{}, fmt.Errorf("corpora: decode %s: %w", path, err)
	}
	if o.Observations == nil {
		o.Observations = map[string]Observation{}
	}
	return o, nil
}

// ---------------------------------------------------------------------
// Curated side: Selection
// ---------------------------------------------------------------------

// RejectedCandidate is one entry in Selection's rejected-candidate
// ledger (D-17): a measured candidate that did NOT make the locked set,
// recorded with why — never simply deleted from the record.
type RejectedCandidate struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

// Selection is the full corpora/selection.json document: hand-authored
// CURATED policy — per-kind thresholds, the threshold rationale, the
// locked set, the rejected-candidate ledger and any synthetic-coverage
// declarations. This package's own tooling never writes this file; a
// human authors it (Plan 01-06).
type Selection struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	MinEdgesPerKind    map[string]int64    `json:"minEdgesPerKind"`
	ThresholdRationale string              `json:"thresholdRationale"`
	LockedSet          []string            `json:"lockedSet"`
	Rejected           []RejectedCandidate `json:"rejected"`
	SyntheticKinds     []string            `json:"syntheticKinds"`
}

// LoadSelection reads path, decodes it as JSON into a Selection, and
// validates it before returning: a non-nil error naming path when the
// file is absent or malformed, and a validation error when
// ThresholdRationale is empty or whitespace-only (D-15 makes the
// recorded rationale mandatory — an unexplained threshold is invalid
// input, not a lint warning) or when any Rejected entry carries an empty
// Reason.
func LoadSelection(path string) (Selection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Selection{}, fmt.Errorf("corpora: read %s: %w", path, err)
	}
	var s Selection
	if err := json.Unmarshal(data, &s); err != nil {
		return Selection{}, fmt.Errorf("corpora: decode %s: %w", path, err)
	}
	if strings.TrimSpace(s.ThresholdRationale) == "" {
		return Selection{}, fmt.Errorf("corpora: %s: thresholdRationale must be non-empty (D-15)", path)
	}
	for _, r := range s.Rejected {
		if strings.TrimSpace(r.Reason) == "" {
			return Selection{}, fmt.Errorf("corpora: %s: rejected candidate %q has an empty reason", path, r.Repo)
		}
	}
	return s, nil
}

// ---------------------------------------------------------------------
// The deterministic selection algorithm
// ---------------------------------------------------------------------

// ComputeThresholds derives a per-kind minimum edge-count threshold from
// obs, sourced from ALL of eligible — the full measured universe, never
// the locked subset. Sourcing best from the locked set would make
// thresholds and selection each depend on the other, and multiple fixed
// points could exist; sourcing it from the full measured universe breaks
// the cycle. For each query.RankEdges kind, best is the highest
// Observation.EdgeCount for that kind across eligible; the returned
// threshold is min(max(2, best/2), best) using integer division. The
// outer min is what makes every threshold SATISFIABLE BY CONSTRUCTION:
// without it, a kind whose best is 1 would derive max(2, 0) = 2, a bar
// nothing measured can clear. With the clamp, best == 1 yields 1 and
// best == 0 yields 0, so the record can say plainly that a kind is
// arithmetically uncoverable rather than inventing a bar nothing can
// pass.
func ComputeThresholds(obs Observations, eligible []string) map[string]int64 {
	out := make(map[string]int64, len(query.RankEdges))
	for kind := range query.RankEdges {
		var best int64
		for _, key := range eligible {
			o, ok := obs.Observations[key]
			if !ok {
				continue
			}
			if c := o.EdgeCount(kind); c > best {
				best = c
			}
		}
		threshold := best / 2
		if threshold < 2 {
			threshold = 2
		}
		if threshold > best {
			threshold = best
		}
		out[kind] = threshold
	}
	return out
}

// ErrNoQualifyingSubset is returned by SelectLockedSet, wrapped with the
// unsatisfiable kinds and languages, when no subset of eligible clears
// every threshold and gives every PriorityLanguages member a non-zero
// summed file count.
var ErrNoQualifyingSubset = errors.New("corpora: no qualifying subset")

// SelectLockedSet enumerates subsets of eligible in increasing
// cardinality and returns the FIRST that satisfies every kind in th
// (each kind's threshold is compared against the MAX Observation.EdgeCount
// among the subset's observations — the single named supplier the
// coverage claim attributes each kind to) and gives every
// PriorityLanguages member a non-zero summed file count across the
// subset. Within a cardinality, subsets are ordered by total
// TrackedFiles ascending, then by their sorted repository-name list
// lexicographically, so the result is fully determined. Brute-force
// enumeration is correct and cheap here: D-17 bounds the candidate count
// at ten, so at most 1024 subsets are ever considered. Returns
// ErrNoQualifyingSubset, naming the unsatisfiable kinds and languages
// (evaluated across the FULL eligible set), when no subset qualifies.
func SelectLockedSet(obs Observations, th map[string]int64, eligible []string) ([]string, error) {
	keys := append([]string(nil), eligible...)
	sort.Strings(keys)
	n := len(keys)

	for size := 0; size <= n; size++ {
		masks := masksOfSize(n, size)
		sort.Slice(masks, func(i, j int) bool {
			si, sj := subsetFromMask(keys, masks[i]), subsetFromMask(keys, masks[j])
			ti, tj := totalTrackedFiles(obs, si), totalTrackedFiles(obs, sj)
			if ti != tj {
				return ti < tj
			}
			return lessRepoNames(repoNames(obs, si), repoNames(obs, sj))
		})
		for _, mask := range masks {
			subset := subsetFromMask(keys, mask)
			if satisfiesThresholds(obs, th, subset) && satisfiesPriorityLanguages(obs, subset) {
				out := append([]string(nil), subset...)
				sort.Strings(out)
				return out, nil
			}
		}
	}

	return nil, unsatisfiableError(obs, th, keys)
}

func masksOfSize(n, size int) []int {
	var out []int
	for mask := 0; mask < (1 << n); mask++ {
		if bits.OnesCount(uint(mask)) == size {
			out = append(out, mask)
		}
	}
	return out
}

func subsetFromMask(keys []string, mask int) []string {
	var out []string
	for i, k := range keys {
		if mask&(1<<i) != 0 {
			out = append(out, k)
		}
	}
	return out
}

func totalTrackedFiles(obs Observations, keys []string) int64 {
	var total int64
	for _, k := range keys {
		total += obs.Observations[k].TrackedFiles
	}
	return total
}

func repoNames(obs Observations, keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, obs.Observations[k].Repo)
	}
	sort.Strings(out)
	return out
}

func lessRepoNames(a, b []string) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func satisfiesThresholds(obs Observations, th map[string]int64, subset []string) bool {
	for kind, threshold := range th {
		var best int64
		for _, k := range subset {
			if c := obs.Observations[k].EdgeCount(kind); c > best {
				best = c
			}
		}
		if best < threshold {
			return false
		}
	}
	return true
}

func satisfiesPriorityLanguages(obs Observations, subset []string) bool {
	for _, group := range PriorityLanguages {
		var total int64
		for _, k := range subset {
			total += obs.Observations[k].LanguageFileCount(group)
		}
		if total == 0 {
			return false
		}
	}
	return true
}

func unsatisfiableError(obs Observations, th map[string]int64, eligible []string) error {
	var failedKinds []string
	for kind, threshold := range th {
		var best int64
		for _, k := range eligible {
			if c := obs.Observations[k].EdgeCount(kind); c > best {
				best = c
			}
		}
		if best < threshold {
			failedKinds = append(failedKinds, kind)
		}
	}
	var failedLanguages []string
	for _, group := range PriorityLanguages {
		var total int64
		for _, k := range eligible {
			total += obs.Observations[k].LanguageFileCount(group)
		}
		if total == 0 {
			failedLanguages = append(failedLanguages, group.Name)
		}
	}
	sort.Strings(failedKinds)
	sort.Strings(failedLanguages)
	return fmt.Errorf("%w: kinds %v unsatisfiable, languages %v unsatisfiable, across %d eligible candidates",
		ErrNoQualifyingSubset, failedKinds, failedLanguages, len(eligible))
}
