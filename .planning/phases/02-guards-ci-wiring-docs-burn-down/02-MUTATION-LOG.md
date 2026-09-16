# 02-MUTATION-LOG — Guards, CI Wiring & Docs Burn-down

**Phase:** 02-guards-ci-wiring-docs-burn-down
**Date:** 2026-09-15
**Scope:** Three demonstration families across this phase's guard work — (a) GRD-09: `web:drift`
replayed against its real historical incident, `98cd41dd`, appended here by plan 02-01; (b)
GRD-11: `check:gonum` / `check:no-force-layout` wired into `ci.yml`'s `test` job, appended by
plan 02-02; (c) GRD-12: the `protect-main` ruleset-drift comparison, appended by plan 02-04. Each
family proves its guard fails against the real condition it exists to catch before the guard (or,
for family (a), the pre-existing unchanged guard) is trusted.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
a replay against a historical commit with no mutation at all, `git status --porcelain` inside the
scratch worktree) is asserted to exit empty/clean. This proves no pre-existing tracked edit was
overwritten by the mutation, and no revert (or, for a replay, no worktree teardown) was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.

---

## Family (a) — GRD-09: web:drift OUTPUT-half mismatch replay against 98cd41dd

**Test/guard:** `task web:drift` (`Taskfile.yml`), replayed against commit `98cd41dd`'s exact tree
via `git worktree add --detach <scratch> 98cd41dd` — no code mutation, per the RESEARCH-verified
Pattern 1: the RED condition is the historical git tree's own shape, not an injected defect. This
is the same shape as Phase 7's Family (a) (`GRD-01`): the guard was never edited to produce this
failure, and there is nothing to revert in the guard itself.

**What are we testing, and why?** Whether the committed gate, run exactly as CI runs it (a clean
checkout, `find web/build` enumerating the git tree), goes RED on the one real incident that ever
fooled a developer — commit `98cd41dd`, which staged `web/build` as a bare pathspec and silently
missed 9 brand-new output files while `web:drift` still reported PASS in that developer's own
working tree at the time. We are deliberately NOT testing whether a `find`-vs-`git ls-files`
paired set assertion would also catch it: D-01 established that such an assertion would only
detect that the developer forgot to `git add` what they built — a local staging-hygiene misread
that costs one CI round trip, not a genuine gate blind spot, since CI itself always runs on a
clean checkout where `find` and the git tree already agree.

**Pre-mutation gate:** `git status --porcelain` inside the fresh worktree at `98cd41dd` — empty
(genuinely clean checkout, exactly the CI shape, not a developer's dirty tree with the extra
untracked files that fooled `web:drift` locally in the original incident).

**Mutation applied:** None — the RED condition is the commit's own historical shape (`98cd41dd`'s
tree), not an injected mutation of currently-correct code. `Taskfile.yml` was never touched; this
plan's `files_modified` excludes it by design (D-01: the gate is not changed).

**Observed failure** (pasted verbatim, `task web:drift` inside `git worktree add --detach
/tmp/02-01-98cd41dd 98cd41dd`):

```
web:drift: hashed 108 source files
web:drift: manifested 22 output files
web:drift: source half MATCH (108 files, 1e0bff2fb48d58bd943e8bb842dff6d81f3c608105ed3d2b31aa6585a6b8f5c3)
::error::web:drift: OUTPUT-half mismatch — the committed web/build/ bytes are NOT the ones `task web:build` produced: someone edited, added, or removed a file inside the committed build output (marker: 32 files / 8658e6fe64bb02282e008557d39baba453d3e2765d6020071c8dc58d7ca9c432; recomputed: 22 files / a76add11cadd4bd4cb0322de2a0e62246187397236860cd0b9eb3e1e57bf39e2). Run `task web:build` to rebuild — but an unexplained output-half mismatch on a tree nobody rebuilt should be INVESTIGATED, not rebuilt away.
task: Failed to run task "web:drift": exit status 1
```

Exit code: **201** (`task`'s own wrapper exit for the target's `exit status 1`).

The count subtlety RESEARCH already flagged: the manifest recorded 32 output files, the replay
recomputed 22 on disk — a difference of 10, not the 9 files commit `fad3b39c` later staged to fix
the incident. The tenth is `web/build/.build-manifest` itself, which `web_output_files()`
explicitly `! -name '.build-manifest'`-excludes from the `find` enumeration but which the marker's
own recorded `output-files: 32` count includes (the manifest counts itself as part of the tree it
describes). So: 32 manifested − 22 found-on-disk = 10 missing from the replay's `find`, of which 1
is the manifest's own self-exclusion and the remaining 9 are exactly the brand-new chunk files
`fad3b39c` later staged. This is stated carefully here so a reader does not conclude the 32-vs-22
gap contradicts the "9 files" fix commit — it does not; the arithmetic reconciles exactly.

**Revert:** `git worktree remove --force /tmp/02-01-98cd41dd`, followed by `git worktree prune` —
the main repo was never checked out to, stashed, or reset; only the scratch worktree existed.

**Byte-clean proof:** No tracked file in the main repo was touched by this replay.
`git diff --quiet -- Taskfile.yml` exits 0 at this plan's final commit (the gate itself is
unedited, per D-01), and `git status --porcelain -- Taskfile.yml web/` is empty in the main tree
both before and after the replay.

**Green control** (the discriminator, probe GRD-09/empty — proves the RED above is a property of
`98cd41dd`'s tree, not of running the gate inside a worktree at all): `task web:drift` inside
`git worktree add --detach /tmp/02-01-head HEAD`:

```
web:drift: hashed 119 source files
web:drift: manifested 36 output files
web:drift: source half MATCH (119 files, 3f9a0e7c4def36e008d820f45650546d096eb2ee099780c9cad76558bd41ccad)
web:drift: output half MATCH (36 files, e2b98b5a6e18ccb728a5c7456f6eea9c1d3e5087ea72869c373f0bb3cef4fc97)
web:drift: PASS — hashed 119 source files, manifested 36 output files, committed web/build/ matches both digests
```

Exit code: **0**. Both counts are non-zero (119 ≥ the `SRC_FLOOR=8` guard, 36 ≥ the `OUT_FLOOR=3`
guard), so this green path reports a genuine, non-trivial set size on both halves rather than
passing on an empty enumeration — the RED above discriminates against a real defect rather than
failing on everything indiscriminately. Cleanup: `git worktree remove --force
/tmp/02-01-head` then `git worktree prune`; `git worktree list` afterward shows only the main
working tree.

**Verdict:** CI's clean-checkout `find` enumeration already fails on this incident shape — the
gate was never vacuous against the one real incident that ever fooled a developer, and `Taskfile.yml`
required no change to prove it. The ledger's suggested `find`-vs-`git ls-files` paired set
assertion (WINDOWS #29's "SUGGESTED FIX") is **DECLINED per D-01**: it would test whether the
developer staged what they built — git hygiene, not the gate — and buys at most one saved CI round
trip against a defect class CI's own clean-checkout semantics already catch. WINDOWS #29 is closed
as **FIXED** by plan 02-07 (`gsd-tools windows fixed 29`), citing this entry's transcript as the
recorded cause.

---

## Family (b) — GRD-11: planted forbidden layout literal makes the wired check:no-force-layout step fail

**Test/guard:** the `ci.yml` `test`-job step `No-force-layout guard (GRD-11/GRF-06)` →
`task check:no-force-layout` (`web/scripts/check-no-force-layout.mjs`, run with `--self-test`
first then the real scan, exactly as the wired CI step runs it).

**What are we testing, and why?** Whether the newly-wired step's exact command can fail on a real
forbidden-layout literal, or whether the wiring is vacuous (D-04, rule `84d1gfpywd`). We are NOT
re-testing the scanner's own logic — its `--self-test` (run unconditionally, before every real
scan) is the target's own positive control; this family plants a violation on real disk and
proves the SAME command the CI step invokes goes RED.

**Pre-mutation gate:** `git status --porcelain web/src` — empty.

**Mutation applied:** created an UNTRACKED file `web/src/lib/__planted-no-force-layout__.ts`,
one line: `const layout = { name: 'cose' };` (the forbidden-name-at-`name:`-position shape the
scanner exists to catch — the same literal shape the script's own `selfTest()` injects
in-memory). Never `git add`ed.

**Observed failure** (verbatim, `task check:no-force-layout`, planted file present):

```
task: [check:no-force-layout] node web/scripts/check-no-force-layout.mjs --self-test
check-no-force-layout self-test (symlink traversal): PASS — forbidden match found through linkdir -> planted
check-no-force-layout self-test (WR-03b): PASS — unresolved layout name detected at <self-test>/unresolved.ts:1
check-no-force-layout self-test: FAIL — the scan did not detect the injected layout name
{"filesScanned":109,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[{"file":"web/src/lib/__planted-no-force-layout__.ts","line":1,"match":"name: 'cose'"},{"file":"<self-test>/injected.ts","line":1,"match":"name: 'cose'"}],"unresolvedLayoutNames":[{"file":"web/src/lib/components/graph/GraphCanvas.svelte","line":396,"snippet":"const thisLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit });"},{"file":"web/src/lib/components/graph/GraphCanvas.svelte","line":999,"snippet":"// then `cy.layout(layoutOpts).run()`) — when no `layout` option"},{"file":"<self-test>/unresolved.ts","line":1,"snippet":"cy.layout({ name: layoutName });"}],"verdict":"FAIL"}
check-no-force-layout: scanned 109 files; elk layout refs 2; elk import refs 3; forbidden matches 2; unresolved layout names (advisory) 3; verdict FAIL
  forbidden: web/src/lib/__planted-no-force-layout__.ts:1 — name: 'cose'
  forbidden: <self-test>/injected.ts:1 — name: 'cose'
  unresolved (advisory, does not fail verdict): web/src/lib/components/graph/GraphCanvas.svelte:396 — const thisLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit });
  unresolved (advisory, does not fail verdict): web/src/lib/components/graph/GraphCanvas.svelte:999 — // then `cy.layout(layoutOpts).run()`) — when no `layout` option
  unresolved (advisory, does not fail verdict): <self-test>/unresolved.ts:1 — cy.layout({ name: layoutName });
task: Failed to run task "check:no-force-layout": exit status 1
exit=201
```

**Note on the self-test's own verdict (a stronger result than the plan anticipated):**
`selfTest()` in `check-no-force-layout.mjs` does not scan an isolated fixture tree — it calls
`runScan({ extraFiles: [injected, injectedUnresolved] })`, which walks the REAL `web/src` tree
and appends its two in-memory injected sources, then asserts `forbiddenMatches.length === 1`
(exactly the injected file, proving the real tree "stayed clean"). With the planted file present
in `web/src`, that assertion sees TWO forbidden matches (the planted file plus the self-test's
own injection) and correctly reports `self-test: FAIL`. This means the planted violation was
caught by BOTH the self-test's own real-tree contamination check AND the real scan below it —
a stronger demonstration than a self-test that stayed silent while only the real scan went red.
(Plan 02-02's authored `<verify>` automated check for this task asserted `self-test: PASS` would
still appear in this transcript; that assumption did not hold given the self-test's real-tree
scan, and the plan's diff-unscoped `continue-on-error` grep in Task 1's third `<verify>` command
had the analogous "whole-file, not diff-scoped" issue — both are recorded here as plan-authoring
assumption gaps, not implementation defects; the task's real acceptance criteria — exits non-zero
with the planted path named, and exits 0 after a byte-clean revert — are met exactly as written.)

**Revert:** `rm -f web/src/lib/__planted-no-force-layout__.ts`.

**Byte-clean proof:** `git status --porcelain web/src` — empty immediately after removal.

**Green re-run** (verbatim, `task check:no-force-layout`, planted file removed):

```
task: [check:no-force-layout] node web/scripts/check-no-force-layout.mjs --self-test
check-no-force-layout self-test (symlink traversal): PASS — forbidden match found through linkdir -> planted
check-no-force-layout self-test (WR-03b): PASS — unresolved layout name detected at <self-test>/unresolved.ts:1
check-no-force-layout self-test: PASS — injected 'cose' detected at <self-test>/injected.ts:1
task: [check:no-force-layout] node web/scripts/check-no-force-layout.mjs
{"filesScanned":106,"elkLayoutRefs":2,"elkImportRefs":3,"forbiddenMatches":[],"unresolvedLayoutNames":[{"file":"web/src/lib/components/graph/GraphCanvas.svelte","line":396,"snippet":"const thisLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit });"},{"file":"web/src/lib/components/graph/GraphCanvas.svelte","line":999,"snippet":"// then `cy.layout(layoutOpts).run()`) — when no `layout` option"}],"verdict":"PASS"}
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 0; unresolved layout names (advisory) 2; verdict PASS
```

Exit code: **0**.

**check:gonum — no planted mutation (built-in positive controls only).** `check:gonum` was not
mutated: a planted violation would require a `go.mod`/dependency change, out of scope for a
CI-wiring plan. Its three halves carry built-in positive controls that fired in the clean-tree
run captured for Task 1 (verbatim):

```
check:gonum: SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)
check:gonum: cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 3 with cgo (want >= 1)
check:gonum: PASS
```

**Verdict:** the `No-force-layout guard (GRD-11/GRF-06)` CI step's exact command is proven to fail
loudly on a real planted violation (naming the offending file and line), and to return cleanly to
PASS after a byte-clean revert — the wiring is not vacuous. `check:gonum`'s own positive controls
(quoted above) are the guard for that target, per D-04/Phase 7 D-07.

---

*Family (c) — GRD-12 (`protect-main` ruleset-drift comparison) is appended to this file by plan
02-04.*
