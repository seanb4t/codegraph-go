// coverage.go is the FIXT-01 coverage guard: it DERIVES the coverage claim
// from three committed documents rather than validating a precomputed
// summary. See 01-07-PLAN.md's objective — "The guard derives the claim; it
// does not validate a summary".
//
// An earlier draft of this guard read a stored `coverage[kind]` map from a
// record and compared each entry against a threshold. Nothing in that design
// required the claimed supplier to be locked, to have actually been measured,
// or to match its own raw measurement — a summary naming a repository that
// was never locked, or citing a count higher than that repository's
// observation, would have passed. That is rule `84d1gfpywd` in its subtler
// form: not a guard that checks nothing, but a guard that checks the wrong
// artifact.
//
// CheckCoverage therefore takes all three already-decoded documents and
// reconstructs the claim from first principles. Loading — and the failure
// modes of loading — belong to the CALLER, not to this function: CheckCoverage
// has no I/O. (An earlier draft's must_haves attributed loading to it, which
// its own declared signature contradicted.)
package corpora

import (
	"sort"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// KindCoverage is CheckCoverage's per-kind derived view: the threshold the
// selection imposes, the best measured count across the locked observations,
// and the repository that supplied that count. There is NO stored supplier
// field behind this — Repo is computed from the observations here, which is
// exactly why it cannot disagree with the evidence.
type KindCoverage struct {
	// Threshold is sel.MinEdgesPerKind[kind], the bar the selection sets.
	Threshold int64
	// Count is the DERIVED best measured edge count for kind across the
	// locked observations. If no locked observation measured kind, Count
	// is zero and Repo is empty.
	Count int64
	// Repo is the manifest entry whose observation supplied Count — the
	// derived best supplier, never a stored field.
	Repo string
}

// CoverageResult is CheckCoverage's whole output.
type CoverageResult struct {
	// CheckedKinds is the number of kinds swept: always len(query.RankEdges),
	// DERIVED from the ranked-kind set so there is no hand-maintained
	// expected-count constant to go stale beside it.
	CheckedKinds int
	// CheckedCorpora is the number of corpora swept: always
	// len(LockedEntries(m)), DERIVED from the manifest (the sole pin
	// authority, D-09). A run over an empty locked set reports a failure,
	// never a green pass.
	CheckedCorpora int
	// Kinds is a per-kind view keyed by every query.RankEdges kind.
	Kinds map[string]KindCoverage
	// Failures is a deterministic (sorted) list of every violation found.
	Failures []string
}

// CheckCoverage derives the coverage claim from m, obs and sel together and
// reports every failure. It performs NO I/O — the caller loads the three
// documents (corpora.Load, corpora.LoadObservations, corpora.LoadSelection)
// and fails with the path named when any is missing or malformed; this
// function works only on the decoded values.
//
// Derivation order:
//
//  1. derive the locked repo@sha identities from the manifest's locked
//     entries (LockedEntries);
//  2. require EXACT set equality with sel.LockedSet — in both directions, so
//     a manifest that locks a repository the selection omits fails just as a
//     selection that names an unlocked repository does;
//  3. require every locked identity to have an observation in obs;
//  4. compute, per query.RankEdges kind, the best count across those locked
//     observations and which repository supplied it — ONLY locked identities
//     contribute, so an unlocked (or rejected) candidate's observation can
//     never supply a kind;
//  5. apply sel's thresholds to the DERIVED counts (at-least comparison: a
//     kind measuring exactly its threshold PASSES — this is the boundary
//     convention Plan 01-06 recorded in the selection);
//  6. require sel.MinEdgesPerKind to cover every query.RankEdges kind and to
//     carry no kind outside it;
//  7. require every PriorityLanguages group to have a non-zero summed file
//     count across the locked observations;
//  8. require syntheticKinds to be empty.
//
// The positive-count discipline is preserved WITHOUT a hand-maintained
// constant. The wire oracle's ExpectedScenarioCount (test/wireoracle/
// scenarios.go) has no derivable source, so a constant is the only way to
// assert its count positively; the locked-corpus count DOES derive — from the
// manifest, which D-09 makes the sole authority — so this guard derives
// CheckedKinds from query.RankEdges and CheckedCorpora from
// len(LockedEntries(m)) instead of restating a constant beside the manifest.
// A hand-maintained constant would be a second authority requiring an edit on
// every manifest change and capable of disagreeing with it. This is the
// deliberate deviation from the ExpectedScenarioCount pattern.
func CheckCoverage(m Manifest, obs Observations, sel Selection) CoverageResult {
	locked := LockedEntries(m)
	checkedCorpora := len(locked)

	// Step 1: derive the locked identities from the manifest.
	lockedIDs := make(map[string]Entry, len(locked))
	for _, e := range locked {
		lockedIDs[ObservationKey(e.Repo, e.SHA)] = e
	}

	// Step 2: exact set equality with sel.LockedSet, both directions.
	selLocked := make(map[string]bool, len(sel.LockedSet))
	for _, id := range sel.LockedSet {
		selLocked[id] = true
	}

	var failures []string

	for id := range lockedIDs {
		if !selLocked[id] {
			failures = append(failures, "manifest locks "+id+" but the selection's locked set omits it")
		}
	}
	for id := range selLocked {
		if _, ok := lockedIDs[id]; !ok {
			failures = append(failures, "selection locks "+id+" but the manifest does not")
		}
	}

	// Step 3: every locked identity must have an observation.
	observed := make(map[string]Observation, len(obs.Observations))
	for id, o := range obs.Observations {
		observed[id] = o
	}
	for id := range lockedIDs {
		if _, ok := observed[id]; !ok {
			failures = append(failures, "locked corpus "+id+" has no observation")
		}
	}

	// Steps 4-5 together: per-kind derived best count + supplier across
	// LOCKED observations only, then thresholds applied to the derived
	// counts. Because we iterate over the manifest's locked identities, an
	// unlocked or rejected candidate's observation can never supply a kind —
	// it is structurally excluded.
	kinds := make(map[string]KindCoverage, len(query.RankEdges))
	best := make(map[string]int64, len(query.RankEdges))
	bestRepo := make(map[string]string, len(query.RankEdges))
	for kind := range query.RankEdges {
		var b int64
		var repo string
		for _, e := range locked {
			o, ok := observed[ObservationKey(e.Repo, e.SHA)]
			if !ok {
				continue // missing observation already reported in step 3
			}
			if c := o.EdgeCount(kind); c > b {
				b = c
				repo = e.Repo
			}
		}
		best[kind] = b
		bestRepo[kind] = repo
	}

	// Step 6: the selection's threshold map must cover every RANK_EDGES kind
	// and carry no kind outside it.
	for kind := range query.RankEdges {
		if _, ok := sel.MinEdgesPerKind[kind]; !ok {
			failures = append(failures, "selection omits a threshold for ranked kind "+kind)
		}
	}
	for kind := range sel.MinEdgesPerKind {
		if !query.RankEdges[kind] {
			failures = append(failures, "selection carries a threshold for kind "+kind+" outside the ranked set")
		}
	}

	// Step 5: thresholds applied to DERIVED counts, at-least comparison — a
	// derived count exactly equal to its threshold PASSES.
	for kind := range query.RankEdges {
		threshold, hasTh := sel.MinEdgesPerKind[kind]
		count := best[kind]
		if hasTh && count < threshold {
			failures = append(failures, "kind "+kind+" derived count "+fmtInt(count)+" below threshold "+fmtInt(threshold))
		}
		kinds[kind] = KindCoverage{Threshold: threshold, Count: count, Repo: bestRepo[kind]}
	}

	// Step 7: every priority language has a non-zero summed file count across
	// the locked observations.
	for _, group := range PriorityLanguages {
		var total int64
		for _, e := range locked {
			if o, ok := observed[ObservationKey(e.Repo, e.SHA)]; ok {
				total += o.LanguageFileCount(group)
			}
		}
		if total == 0 {
			failures = append(failures, "priority language "+group.Name+" has zero files across the locked observations")
		}
	}

	// Step 8: synthetic coverage is not accepted for FIXT-01 in this phase.
	if len(sel.SyntheticKinds) > 0 {
		names := make([]string, 0, len(sel.SyntheticKinds))
		for _, k := range sel.SyntheticKinds {
			names = append(names, k)
		}
		sort.Strings(names)
		failures = append(failures, "synthetic coverage is not accepted: "+strings.Join(names, ", "))
	}

	// The checked-corpus positive assertion: a run over an empty locked set
	// fails, never reports green vacuously (rule 84d1gfpywd).
	if checkedCorpora == 0 {
		failures = append(failures, "locked corpus set is empty — nothing to verify")
	}

	sort.Strings(failures)

	return CoverageResult{
		CheckedKinds:   len(query.RankEdges),
		CheckedCorpora: checkedCorpora,
		Kinds:          kinds,
		Failures:       failures,
	}
}

func fmtInt(v int64) string {
	return strconv.FormatInt(v, 10)
}