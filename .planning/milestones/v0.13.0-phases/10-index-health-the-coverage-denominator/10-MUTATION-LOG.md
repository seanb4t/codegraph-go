# 10-MUTATION-LOG — Index Health: The Coverage Denominator

**Phase:** 10-index-health-the-coverage-denominator
**Date:** 2026-09-12
**Scope:** Three RED demonstrations, one per guard family this phase introduces: (a) D-15's unset-`has_coverage` short-circuit, read as zero instead of unknown; (b) D-16's `GetCoverage` read-only method-set guard, renamed to the `GetIndexCoverage` decoy; (c) D-14's query-time reason reconstruction, a present-tense disk filter added to the exclusion-row loop.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a destructive blind checkout of someone else's in-flight work. Every family entry below records this gate's result at the point it was checked (09-MUTATION-LOG.md convention).

```
$ git diff --quiet -- internal/query/coverage.go internal/uiserver/readonly_test.go; echo $?
0
```

---

## Family (a) — D-15: unset `has_coverage` read as zero instead of unknown

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestCoverageSummaryOnOldGraphIsUnknown|TestCoverageRowsUnknownGraphAndSingleRow' ./internal/query/
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse' ./internal/uiserver/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/query/coverage.go; echo $?
0
```

**Mutation applied.** Both `CoverageSummary` and `CoverageRows`' unset/false `has_coverage` short-circuit removed — an old graph falls through to counting whatever `IterateFiles`/`IterateExcludedFiles` actually returns instead of reporting `Known: false` with zero counts. This is D-15's exact "treat unset as zero" lie: the final `Known: true` return at the bottom of each function is unconditional, so removing the guard silently promotes an unknown graph to "known, with real counts":

```diff
 func (e *Engine) CoverageSummary() (CoverageSummary, error) {
+	// MUTATION (10-06 family (a), D-15 RED demonstration): the
+	// unset/false has_coverage short-circuit was removed here, so an
+	// old graph falls through to counting instead of reporting
+	// Known:false — the exact "treat unset as zero" lie D-15 forbids.
+	// Reverted via `git checkout --` after the RED is observed.
 	meta, err := e.IndexMeta()
 	if err != nil {
 		return CoverageSummary{}, err
 	}
-	if meta == nil || !meta.GetHasCoverage() {
-		return CoverageSummary{Known: false}, nil
-	}
+	_ = meta
 
 	fit, err := e.reader.IterateFiles()
@@
 func (e *Engine) CoverageRows(opts CoverageRowsOptions) (CoveragePage, error) {
+	// MUTATION (10-06 family (a), D-15 RED demonstration): same
+	// short-circuit removal as CoverageSummary above.
 	meta, err := e.IndexMeta()
 	if err != nil {
 		return CoveragePage{}, err
 	}
-	if meta == nil || !meta.GetHasCoverage() {
-		return CoveragePage{Known: false}, nil
-	}
+	_ = meta
 
 	pageSize := opts.PageSize
```

**Confirmed applied, build clean:**
```
$ rg -c 'GetHasCoverage' internal/query/coverage.go
0
$ GOTOOLCHAIN=go1.26.6 go build ./internal/query/... ./internal/uiserver/...
(no output — success)
```

**RED — pasted verbatim (`internal/query`, exit 1):**
```
coverage_test.go:143: Known = true, want false
    coverage_test.go:143: counts = {Known:true Discovered:1 Indexed:1 Excluded:0 ExtractionFailed:0 ExcludedByReason:map[]}, want all zero
    coverage_test.go:143: ExcludedByReason = map[], want nil
    coverage_test.go:163: Known = true, want false
    coverage_test.go:163: ExcludedByReason = map[], want nil
--- FAIL: TestCoverageSummaryOnOldGraphIsUnknown (0.08s)
    --- FAIL: TestCoverageSummaryOnOldGraphIsUnknown/meta_present_but_has_coverage_unset (0.05s)
    --- FAIL: TestCoverageSummaryOnOldGraphIsUnknown/no_meta_record_at_all (0.03s)
    coverage_test.go:202: Known = true, want false
--- FAIL: TestCoverageRowsUnknownGraphAndSingleRow (0.12s)
    --- FAIL: TestCoverageRowsUnknownGraphAndSingleRow/old_graph (0.03s)
    --- PASS: TestCoverageRowsUnknownGraphAndSingleRow/tagged_repo_two_rows (0.08s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/query	0.507s
FAIL
```

**RED — pasted verbatim (`internal/uiserver`, exit 1):**
```
coverage_test.go:583: GetHealth coverage.known = true, want false
--- FAIL: TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse (0.19s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.504s
FAIL
```

Both instruments fail on exactly the clause D-15 exists to prevent: an old-graph read reports `Known = true` with non-zero real counts (1 indexed file, in the query-package "meta present but has_coverage unset" subtest) instead of `Known: false` with every count zero. `TestCoverageRowsUnknownGraphAndSingleRow/tagged_repo_two_rows` correctly stays PASS — that subtest indexes a real fixture with `has_coverage` genuinely true, so it was never exercising the short-circuit this mutation removed; the failure is isolated to the "old graph" cases, proving the discriminator, not a broken test harness.

**Revert:**
```
$ git checkout -- internal/query/coverage.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/query/coverage.go; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
--- PASS: TestCoverageSummaryOnOldGraphIsUnknown (0.08s)
    --- PASS: TestCoverageSummaryOnOldGraphIsUnknown/meta_present_but_has_coverage_unset (0.04s)
    --- PASS: TestCoverageSummaryOnOldGraphIsUnknown/no_meta_record_at_all (0.03s)
--- PASS: TestCoverageRowsUnknownGraphAndSingleRow (0.11s)
    --- PASS: TestCoverageRowsUnknownGraphAndSingleRow/old_graph (0.03s)
    --- PASS: TestCoverageRowsUnknownGraphAndSingleRow/tagged_repo_two_rows (0.08s)
ok  	github.com/seanb4t/codegraph-go/internal/query	0.478s
```
```
--- PASS: TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse (0.21s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.536s
```

---

## Family (b) — D-16: `wantUIServiceMethods`'s `GetCoverage` entry renamed to the `GetIndexCoverage` decoy

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestUIServiceMethodSetIsExactlyTheReadSet|TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected' ./internal/uiserver/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/uiserver/readonly_test.go; echo $?
0
```

**Mutation applied.** `"GetCoverage": {}` replaced with `"GetIndexCoverage": {}` in `wantUIServiceMethods`:

```diff
 	"GetEditorLink": {},
-	"GetCoverage":   {},
+	"GetIndexCoverage": {},
 }
```

**Confirmed applied (grep, before running):**
```
$ rg -n '"GetIndexCoverage"' internal/uiserver/readonly_test.go
95:	"GetIndexCoverage": {},
```

**RED — pasted verbatim (exit 1):**
```
readonly_test.go:124: uiv1connect.UIServiceHandler is missing expected read method "GetIndexCoverage" — a method was REMOVED from the service without updating this fixture
--- FAIL: TestUIServiceMethodSetIsExactlyTheReadSet (0.00s)
    readonly_test.go:722: "GetCoverage" is not a member of wantUIServiceMethods — the chosen name was never registered in the read-only fixture
--- FAIL: TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.338s
FAIL
```

**Both directions of the set-equality guard, recorded honestly.** `TestUIServiceMethodSetIsExactlyTheReadSet` iterates `for name := range wantUIServiceMethods { ... }` first (checking every expected name is present in the real, reflected method set) and calls `t.Fatalf` the instant `"GetIndexCoverage"` is not found in `got` — this is the direction that fired, halting the test function before its second loop (`for name := range got { ... }`, which would have reported the real `"GetCoverage"` method as an unexpected addition) ever runs. The OTHER direction is not silently skipped, though: `len(got)` is still the real 16 (nothing was removed from the actual `UIServiceHandler` interface) and `len(got) == len(wantUIServiceMethods)` (both maps have exactly 16 entries — the decoy renamed one entry, it did not add or drop one), so the length guards pass and only the membership check fires — meaning the "GetCoverage now unexpected" direction, while not directly asserted by this Fatalf, is trivially true by construction (`got` still contains `"GetCoverage"`, `wantUIServiceMethods` no longer does). `TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected` independently proves that same fact directly: its closing check `wantUIServiceMethods[wantCoverageRPCName]` (the const `"GetCoverage"`, unchanged) fails to find the key, firing `"GetCoverage" is not a member of wantUIServiceMethods"` — the "table test must fail on the clean name no longer being a member" clause this family's instrument requires.

**Revert:**
```
$ git checkout -- internal/uiserver/readonly_test.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/uiserver/readonly_test.go; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
--- PASS: TestUIServiceMethodSetIsExactlyTheReadSet (0.00s)
--- PASS: TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.303s
```

---

## Family (c) — D-14: rows filtered by present-tense disk state (a query-time reconstruction)

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestCoverageSourceNeverWalksDisk|TestCoverageRowsSurviveDiskMutationWithoutReindex' ./internal/query/
GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/query/coverage.go; echo $?
0
```

**Mutation applied.** `coverage.go`'s exclusion-row loop in `CoverageRows` gained a present-tense disk filter: an `os.Stat` of `filepath.Join(e.repoRoot, path)` skips the row entirely when the excluded path no longer exists on disk — exactly the query-time reconstruction D-14 forbids (a deleted-but-still-recorded exclusion "disappears" instead of staying stale):

```diff
 import (
 	"encoding/base64"
+	"os"
+	"path/filepath"
 	"strings"
 	"unicode/utf8"
 
 	"github.com/seanb4t/codegraph-go/internal/schema"
 )
@@
 			path := x.GetPath()
+			// MUTATION (10-06 family (c), D-14 RED demonstration): a
+			// present-tense disk filter — a query-time reconstruction
+			// D-14 forbids. Skips a row when the excluded path no
+			// longer exists on disk, so a deleted-but-still-recorded
+			// exclusion "disappears" instead of staying stale.
+			// Reverted via `git checkout --` after the RED is observed.
+			if _, statErr := os.Stat(filepath.Join(e.repoRoot, path)); statErr != nil {
+				continue
+			}
 			if skipping {
 				if path == cursorPath {
 					skipping = false
 				}
 				continue
 			}
```

**Confirmed applied, build clean:**
```
$ rg -n 'os\.Stat\(' internal/query/coverage.go
262:			if _, statErr := os.Stat(filepath.Join(e.repoRoot, path)); statErr != nil {
$ GOTOOLCHAIN=go1.26.6 go build ./internal/query/...
(no output — success)
```

**RED — pasted verbatim (`internal/query`, exit 1):**
```
coverage_test.go:274: coverage.go contains "os.Stat(" — D-14 forbids any filesystem call: reasons are read back through graphstore, never reconstructed by a query-time walk
--- FAIL: TestCoverageSourceNeverWalksDisk (0.00s)
    coverage_test.go:950: Rows = [{Path:broken.py Kind:2 Reason:EXCLUSION_REASON_UNSPECIFIED Detail:indexer: reading ./broken.py: open ./broken.py: no such file or directory} {Path:go.mod Kind:1 Reason:EXCLUSION_REASON_UNSUPPORTED_EXTENSION Detail:.mod} {Path:vendor Kind:1 Reason:EXCLUSION_REASON_DIR_VENDOR Detail:vendor} {Path:.hidden Kind:1 Reason:EXCLUSION_REASON_DIR_DOTPREFIX Detail:.hidden} {Path:huge.go Kind:1 Reason:EXCLUSION_REASON_SIZE_LIMIT Detail:4194305 bytes > 4194304} {Path:tagged.go Kind:1 Reason:EXCLUSION_REASON_BUILD_TAG Detail:darwin/arm64}], want a stale notes.md row despite deletion from disk (D-14a)
--- FAIL: TestCoverageRowsSurviveDiskMutationWithoutReindex (0.09s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/query	0.373s
FAIL
```

The structural scan (`TestCoverageSourceNeverWalksDisk`) fails first, on the forbidden `os.Stat(` literal — exactly the guard D-14b exists to catch. The behavioural test (`TestCoverageRowsSurviveDiskMutationWithoutReindex`) independently fails on the missing `notes.md` row: after the fixture's `notes.md` is deleted from disk without re-indexing, the mutated code stats the recorded path, finds it absent, and drops the row — so `page.Rows` shows the other five stale exclusions (`broken.py`, `go.mod`, `vendor`, `.hidden`, `huge.go`, `tagged.go`) but not `notes.md`. `tagged.go`'s row survives this specific mutation (its content was rewritten, not deleted — the file still exists on disk, so the stat succeeds) and is reported unaffected by this test, confirming the mutation's effect is isolated to genuinely deleted paths rather than a broader breakage.

**Archtest instrument, pasted verbatim (exit 0 — stays GREEN under the mutation):**
```
import_direction_test.go:186: loaded 7 packages; production internal/query resolved 392 transitive dependencies
--- PASS: TestQueryImportsNoWireLayerOrIndexerRoot (0.08s)
ok  	github.com/seanb4t/codegraph-go/internal/query/archtest	0.150s
```

**Recorded explicitly, per the plan's own instruction:** `internal/query/archtest` stays green under this mutation, and that is expected, not a gap. `TestQueryImportsNoWireLayerOrIndexerRoot`'s `forbiddenProductionOnlyImports` rule forbids `internal/indexer`'s root import (the discovery/pipeline package) resolving transitively into `internal/query`'s production compilation unit — it says nothing about the standard-library `os`/`path/filepath` packages the mutation used. The package-wide archtest is a boundary guard (query must not depend on the indexer's discovery internals), not a disk-access guard; D-14b's own file-scoped, positive-controlled source scan (`TestCoverageSourceNeverWalksDisk`, reading `coverage.go`'s literal bytes for `os.Stat(`/`os.ReadFile(`/`filepath.WalkDir(`/etc.) is the guard that exists specifically to catch a disk call smuggled in via a normal stdlib import that no import-graph analysis would ever flag. This is exactly why D-14b's source-text scan was chosen over relying on the archtest alone — the mutation used `os`, not the indexer root, and it fired the correct guard.

**Revert:**
```
$ git checkout -- internal/query/coverage.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/query/coverage.go; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
    coverage_test.go:287: inspected coverage.go: 0 forbidden filesystem calls, 5 positive-control occurrences
--- PASS: TestCoverageSourceNeverWalksDisk (0.00s)
--- PASS: TestCoverageRowsSurviveDiskMutationWithoutReindex (0.09s)
ok  	github.com/seanb4t/codegraph-go/internal/query	0.356s
```

---

## Closing

### The three instruments, RED and GREEN

| Family | Requirement | Instrument | Mutated file(s) | RED | GREEN (post-revert) |
|--------|-------------|------------|------------------|-----|----------------------|
| (a) | D-15 | `TestCoverageSummaryOnOldGraphIsUnknown`, `TestCoverageRowsUnknownGraphAndSingleRow`, `TestGetCoverageAndGetHealthOnOldGraphAreKnownFalse` | `internal/query/coverage.go` | `--- FAIL` on all three (2 subtests + 1 whole test) | `--- PASS` on all three |
| (b) | D-16 | `TestUIServiceMethodSetIsExactlyTheReadSet`, `TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected` | `internal/uiserver/readonly_test.go` | `--- FAIL` on both | `--- PASS` on both |
| (c) | D-14 | `TestCoverageSourceNeverWalksDisk`, `TestCoverageRowsSurviveDiskMutationWithoutReindex` (+ `internal/query/archtest` control, stays green) | `internal/query/coverage.go` | `--- FAIL` on both structural and behavioural | `--- PASS` on both |

### Non-vacuity assertion

All three families were watched fail on the specific assertion each exists for, none rewritten to force a pass. (a) removed the unknown-graph short-circuit and watched real counts leak through as `Known: true` on an old graph, at both the Engine level (query package, two functions) and the real-listener level (uiserver). (b) renamed the fixture entry and watched the set-equality guard fail on the missing expected name (the direction that fires first, given map/loop order) while the second test independently proved the other direction's fact (the real name no longer registered) via its own closing membership check — both directions are provably exercised, per the plan's own instruction to note whichever fires first. (c) added a present-tense disk filter and watched BOTH the file-scoped structural scan (forbidden `os.Stat(` literal) and the behavioural disk-mutation test (the missing `notes.md` row) fail, while confirming `internal/query/archtest`'s package-wide import-boundary guard correctly stays green — it is not the guard for this specific mutation shape, D-14b's own source-text scan is.

### Byte-clean proof

```
$ git status --porcelain
(empty)
$ git diff --quiet -- internal/query/coverage.go internal/uiserver/readonly_test.go; echo $?
0
```

No mutation was committed. Every family's mutation was reverted via `git checkout --` and re-verified clean before the next family began.
