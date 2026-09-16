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

*Families (b) — GRD-11 (`check:gonum` / `check:no-force-layout` CI wiring) and (c) — GRD-12
(`protect-main` ruleset-drift comparison) are appended to this file by plans 02-02 and 02-04
respectively.*
