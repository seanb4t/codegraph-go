# 11-MUTATION-LOG — Graph View: Community Clustering

**Phase:** 11-graph-view-community-clustering
**Date:** 2026-09-13
**Scope:** Four RED demonstrations, one per guard family this phase introduces: (a) GRF-08's determinism proof, perturbed by a per-call seed counter; (b) GRF-09's threshold-tamper proof, perturbed by widening `max` after measurement; (c) GRF-06's force-layout scan, perturbed by planting a real layout-name literal in a tracked `web/src` file; (d) GRF-10's cgo positive control, perturbed by swapping it for a pure-Go decoy.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a destructive blind checkout of someone else's in-flight work (10-MUTATION-LOG.md convention).

```
$ git diff --quiet -- internal/query/community.go corpora/graph-cluster-threshold.json web/src/lib/components/graph/graph-style.ts Taskfile.yml; echo $?
0
```

---

## Family (a) — GRF-08: per-call seed counter flips the tie fixture

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestAssignCommunitiesDeterministic' ./internal/query/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/query/community.go; echo $?
0
```

**Mutation applied.** `AssignCommunities` no longer calls `assignCommunitiesWith` with the fixed `communityDefaults` — a package-level `communityCallCounter` is added and incremented on every call, offsetting `communitySeed1` by the running count. Run 1 (offset 0), run 2 (offset 1), run 3 (offset 2), etc. — a different seed on every single call, inside and across both sub-tests of the same test process:

```diff
 var communityDefaults = communityOptions{seed1: communitySeed1, seed2: communitySeed2}
 
+// MUTATION (11-05 family (a), GRF-08 RED demonstration): a per-call
+// counter perturbing the seed on every call — this is NOT production
+// code, it is reverted via `git checkout --` immediately after the RED
+// is observed.
+var communityCallCounter uint64
+
 // AssignCommunities computes a 1-based canonical community id for every
 // node in nodes, from the undirected weighted adjacency edges encodes
 // (GRF-06). It is exported so tools/graphcluster (11-02) can time it in
@@ -76,7 +82,9 @@ var communityDefaults = communityOptions{seed1: communitySeed1, seed2: community
 // this function runs; a node absent from the returned map means it was
 // never passed in, not "not computed".
 func AssignCommunities(nodes []FileGraphNode, edges []FileGraphEdge) map[string]int {
-	return assignCommunitiesWith(nodes, edges, communityDefaults)
+	offset := communityCallCounter
+	communityCallCounter++
+	return assignCommunitiesWith(nodes, edges, communityOptions{seed1: communitySeed1 + offset, seed2: communitySeed2})
 }
```

This is exactly the "flipping offset 11-01 Task 3 logged" perturbation, applied continuously rather than once: 11-01's `TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture` established that `communitySeed1 + 1` flips the tie fixture relative to `communitySeed1 + 0`. This mutation guarantees every call after the first uses a strictly increasing, never-repeating offset, so any two runs of the same fixture in the same process are extremely unlikely to land on the same tie-break.

**Confirmed applied, build clean:**
```
$ rg -n 'communityCallCounter' internal/query/community.go
75:var communityCallCounter uint64
85:	offset := communityCallCounter
86:	communityCallCounter++
$ GOTOOLCHAIN=go1.26.6 go build ./internal/query/...
(no output — success)
```

**RED — pasted verbatim (exit 1):**
```
community_test.go:108: compared 5 runs
    community_test.go:121: run 2: AssignCommunities = map[a.go:1 b.go:2 c.go:2 d.go:1], want run 0's result map[a.go:1 b.go:1 c.go:2 d.go:2]
--- FAIL: TestAssignCommunitiesDeterministic (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/structured (0.00s)
    --- FAIL: TestAssignCommunitiesDeterministic/tie (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/query	0.322s
FAIL
```

**The `structured` sub-test staying PASS is expected, and recorded here as the seed-invariance of a well-separated graph.** `TestAssignCommunitiesDeterministic` runs `t.Run("structured", …)` FIRST — consuming call-counter offsets 0 through 4 — then `t.Run("tie", …)` SECOND, consuming offsets 5 through 9. The `structured` fixture is the well-separated graph (two dense triangles plus a thin bridge, an isolated singleton, and a separate 4-cycle) that 11-01's `TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture` independently proved is seed-invariant under the exact offset that flips the tie fixture: "structured fixture invariant under the flipping offset" (11-01-SUMMARY.md). Modularity optimization over that fixture has one clear, unambiguous answer regardless of which seed drives Louvain's move ordering — there is no tie for the seed to break. The `tie` fixture (a standalone 4-cycle with two perfectly symmetric 2+2 modularity-tying partitions) has no such single answer; which of the two partitions wins is decided entirely by seed-dependent tie-break order, which is precisely why varying the seed on every call turns it RED while `structured` stays GREEN. This is not a gap in the mutation — it is independent confirmation, under a continuously-varying seed rather than one single offset, of the exact distinction 11-01's own hardening test already established.

**Revert:**
```
$ git checkout -- internal/query/community.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/query/community.go; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
community_test.go:108: compared 5 runs
    community_test.go:139: compared 5 runs
--- PASS: TestAssignCommunitiesDeterministic (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/structured (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/tie (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/query	0.318s
```

---

## Family (b) — GRF-09: threshold widened after measurement

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestClusterThresholdDigestMatchesCommittedObservation' ./tools/graphcluster/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- corpora/graph-cluster-threshold.json; echo $?
0
$ git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l
1
```

**Mutation applied.** `metrics.clusteringTimeMs.max` widened from `500` to `5000` — exactly the widening-after-measurement the threshold's own `prohibitions` array and `purpose` string forbid, applied in the WORKING TREE ONLY:

```diff
   "metrics": {
     "clusteringTimeMs": {
-      "max": 500,
+      "max": 5000,
       "unit": "ms",
       "statistic": "median",
       "runs": 3,
```

**Confirmed applied:**
```
$ rg -n '"max": 5000' corpora/graph-cluster-threshold.json
9:      "max": 5000,
$ git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l
1
```
(The commit count stayed at 1 — the mutation is an uncommitted working-tree edit, never staged, never committed.)

**RED — pasted verbatim (exit 1):**
```
ancestry_test.go:129: observation.thresholdDigest = "sha256:6adc6722ff0148b5000e5653c77b4737fb698d27e6f632983baee474e766b971", want "sha256:0e61d57399736d479ab9a77334ce83d5b9d06441d79e218939f5931d0d16536f" (committed threshold digest) — the threshold was edited after measurement, or the observation is stale
--- FAIL: TestClusterThresholdDigestMatchesCommittedObservation (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/tools/graphcluster	0.175s
FAIL
```

Both digests are named: `sha256:6adc6722ff…` is the digest recorded in the committed `corpora/graph-cluster-observations.json` (computed from the threshold's real, unwidened bytes at measurement time); `sha256:0e61d573…` is the digest the test computes fresh from the mutated working-tree file. The mismatch is the exact tamper signal GRF-09's ancestry/digest proof exists to catch — a widened bar would forge a PASS by comparing the harness's next median against a looser number than the one actually measured against.

**Revert:**
```
$ git checkout -- corpora/graph-cluster-threshold.json
```

**Post-revert gate:**
```
$ git diff --quiet -- corpora/graph-cluster-threshold.json; echo $?
0
$ git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l
1
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
ancestry_test.go:150: threshold digest sha256:6adc6722ff0148b5000e5653c77b4737fb698d27e6f632983baee474e766b971 matches; verdict PASS; median 106 ms
--- PASS: TestClusterThresholdDigestMatchesCommittedObservation (0.01s)
ok  	github.com/seanb4t/codegraph-go/tools/graphcluster	0.159s
```

---

## Family (c) — GRF-06: force-layout name planted in a tracked `web/src` file

**Instrument:**
```
node web/scripts/check-no-force-layout.mjs; echo "exit=$?"
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- web/src/lib/components/graph/graph-style.ts; echo $?
0
```

**Mutation applied.** A real layout-name-position literal appended to the end of a tracked `web/src` file (not the untracked scratch file 11-04's own rehearsal used):

```diff
 		}
 	}
 ];
+
+const plantedLayout = { name: 'cose' };
```

**Confirmed applied:**
```
$ rg -n "name: 'cose'" web/src/lib/components/graph/graph-style.ts
187:const plantedLayout = { name: 'cose' };
```

**RED — pasted verbatim (exit 1):**
```
{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[{"file":"web/src/lib/components/graph/graph-style.ts","line":187,"match":"name: 'cose'"}],"verdict":"FAIL"}
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 1; verdict FAIL
  forbidden: web/src/lib/components/graph/graph-style.ts:187 — name: 'cose'
exit=1
```

The scan's `LAYOUT_NAME_RE` (anchored to a `name: '<x>'` option position, never a bare-word search) correctly matched the planted literal, named the exact file and line, and still reported the `elk` positive control unaffected (2 layout refs, 3 import refs) — proving the scan is discriminating, not vacuous.

**Revert:**
```
$ git checkout -- web/src/lib/components/graph/graph-style.ts
```

**Post-revert gate:**
```
$ git diff --quiet -- web/src/lib/components/graph/graph-style.ts; echo $?
0
$ git status --porcelain -- web/src
(empty)
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"verdict":"PASS"}
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 0; verdict PASS
exit=0
```

---

## Family (d) — GRF-10: cgo positive control neutered

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 task -s check:gonum; echo "exit=$?"
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- Taskfile.yml; echo $?
0
```

**Mutation applied.** `check:gonum`'s cgo-closure half's positive control swapped from `github.com/tree-sitter/go-tree-sitter` (a known-cgo package) to `github.com/spf13/cobra` (pure Go) — working tree only:

```diff
-        go list -deps -f '{{"{{"}}.ImportPath{{"}}"}} {{"{{"}}len .CgoFiles{{"}}"}}' github.com/tree-sitter/go-tree-sitter > "${scratch}/control.txt"
+        go list -deps -f '{{"{{"}}.ImportPath{{"}}"}} {{"{{"}}len .CgoFiles{{"}}"}}' github.com/spf13/cobra > "${scratch}/control.txt"
         cnon=$(awk '$2 != "0"' "${scratch}/control.txt" | wc -l | tr -d ' ')
```

**Confirmed applied:**
```
$ rg -n "github.com/spf13/cobra' > " Taskfile.yml
1715:        go list -deps -f '{{"{{"}}.ImportPath{{"}}"}} {{"{{"}}len .CgoFiles{{"}}"}}' github.com/spf13/cobra > "${scratch}/control.txt"
```

**RED — pasted verbatim (target exit 1; `task -s` wraps it as exit 201):**
```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 4 vulnerabilities in packages you import and 3
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.
check:gonum: govulncheck (source mode, main module) clean — 26 gonum packages in the scanned set
check:gonum: SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)
check:gonum: cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 0 with cgo (want >= 1)
::error::check:gonum: positive control found no cgo in github.com/tree-sitter/go-tree-sitter — the scan cannot detect cgo, its clean verdict is meaningless
task: Failed to run task "check:gonum": exit status 1
exit=201
```

The target reached the third half before failing: Half 1 (govulncheck) reported its own clean line, Half 2 (SBOM) reported its own presence line, and only Half 3 (cgo closure) fired — the report line still names `github.com/tree-sitter/go-tree-sitter` (the label baked into the echo string, unaffected by the mutation) but reports `0 with cgo` for it, because the actual `go list -deps` call underneath now targets the pure-Go `github.com/spf13/cobra` decoy instead. The named `::error::` line is exact: `positive control found no cgo in github.com/tree-sitter/go-tree-sitter — the scan cannot detect cgo, its clean verdict is meaningless`.

**Revert:**
```
$ git checkout -- Taskfile.yml
```

**Post-revert gate:**
```
$ git diff --quiet -- Taskfile.yml; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim tail (exit 0):**
```
check:gonum: govulncheck (source mode, main module) clean — 26 gonum packages in the scanned set
check:gonum: SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)
check:gonum: cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 3 with cgo (want >= 1)
check:gonum: PASS
```

---

## Closing

### The four instruments, RED and GREEN

| Family | Requirement | Instrument | Mutated file(s) | RED | GREEN (post-revert) |
|--------|-------------|------------|------------------|-----|----------------------|
| (a) | GRF-08 | `TestAssignCommunitiesDeterministic` | `internal/query/community.go` | `--- FAIL` on `tie` sub-test; `structured` sub-test stays `--- PASS` (expected, seed-invariant) | `--- PASS` on both sub-tests |
| (b) | GRF-09 | `TestClusterThresholdDigestMatchesCommittedObservation` | `corpora/graph-cluster-threshold.json` (working tree only) | `--- FAIL`, digest mismatch naming both digests | `--- PASS` |
| (c) | GRF-06 | `node web/scripts/check-no-force-layout.mjs` | `web/src/lib/components/graph/graph-style.ts` | exit 1, `forbidden matches 1`, file:line named | exit 0, `verdict PASS` |
| (d) | GRF-10 | `task -s check:gonum` | `Taskfile.yml` (working tree only) | exit 1 (task-wrapped 201), `positive control found no cgo`, first two halves' lines printed first | exit 0, `check:gonum: PASS` |

### Non-vacuity assertion

All four families were watched fail on the specific assertion each exists for, none rewritten to force a pass. (a) perturbed the seed on every call and watched the modularity-tying `tie` sub-test disagree run-to-run, while independently confirming `structured` stays invariant — the exact discriminator 11-01's own hardening test established. (b) widened the committed bar in the working tree only and watched the digest proof reject it by name, with the threshold's single-commit invariant unbroken throughout. (c) planted a real layout-name literal in a tracked source file and watched the positive-controlled scan name the exact file and line while the `elk` control stayed unaffected. (d) swapped the cgo scan's own known-cgo control for a pure-Go decoy and watched the guard refuse to call itself clean, reaching all the way to the third half before firing — proving the earlier two halves are not masking the same failure.

### Byte-clean proof

```
$ git status --porcelain
(empty, except this new log file once staged)
$ git diff --quiet -- internal/query/community.go corpora/graph-cluster-threshold.json web/src/lib/components/graph/graph-style.ts Taskfile.yml; echo $?
0
$ git log --format=%H -- corpora/graph-cluster-threshold.json | wc -l
1
```

No mutation was committed. Every family's mutation was reverted via `git checkout --` and re-verified clean before the next family began.
