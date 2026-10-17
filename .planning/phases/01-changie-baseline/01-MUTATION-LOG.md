# 01-MUTATION-LOG — Changie Baseline

**Phase:** 01-changie-baseline
**Date:** 2026-09-25
**Scope:** Four demonstration families, all appended by 01-02-PLAN.md Task 3, proving the
`check:changie` live-tool guard (Task 1) and the pre-existing `TestChangieVersionSeedsMatchChangelog`
Go guard (01-01-PLAN.md) actually catch real regressions rather than passing vacuously —
(a) CHG-03/D-08/ROADMAP SC1: a one-byte `CHANGELOG.md` mutation defeats the `check:changie`
byte-reproduction leg; (b) CHG-04/D-11: an `optional: true` addition to the `.changie.yaml` `PR`
custom field defeats the missing-PR refusal leg; (c) CHG-04: deleting `fragmentFileFormat` from
`.changie.yaml` defeats the sub-second collision leg; (d) D-06/CHG-02: the Go seed/heading
set-equality guard fails in both directions (a seed missing from disk, and a spurious extra seed).

All mutations happened in a disposable, detached scratch git worktree — never the main working
tree — created and removed within this single Task 3 execution.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
the directory-scoped checks, `git status --porcelain -- <dir>`) is asserted to exit empty/clean.
This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.

## Setup

```
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go  b85d2eea [gsd/v0.15.0-milestone]

$ S=$(mktemp -d)
$ git worktree add --detach "$S/wt" HEAD
Preparing worktree (detached HEAD b85d2eea)
HEAD is now at b85d2eea ci(01-02): wire check:changie into ci.yml and scan changie in task vuln (CHG-01, CHG-03, CHG-04)
```

No commits were made in the worktree, and the repo configures no hooks, so no hook installation
was needed.

**Positive control:**

```
$ task -d "$S/wt" check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/11] changie new with no PR refused: custom missing and prompt is disabled: custom key 'PR' — ok
check:changie: [6/11] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/11] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/11] changie new -m PR=abc refused: invalid number — ok
check:changie: [9/11] two back-to-back new -k Fixes calls raised the fragment count to 3 (sub-second fragmentFileFormat holds) — ok
check:changie: [10/11] with only Fixes fragments present, next auto = v0.14.1 (patch successor of v0.14.0) — ok
check:changie: [11/11] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

---

## Family (a) — byte-reproduction leg (ROADMAP SC1, D-08, CHG-03)

**Test/guard:** `check:changie` leg [3/11] (`changie merge --dry-run` byte-compared against
`CHANGELOG.md`).

**What are we testing, and why?** Whether the guard's byte-reproduction leg actually detects drift,
rather than passing vacuously on a `cmp` invocation that never really diverges. A header-only or
whitespace-only difference must ALSO count as drift (D-07's explicit requirement) — this is
exactly that shape: a single lowercase-vs-uppercase byte in the header line, no content removed or
added.

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- CHANGELOG.md` exited 0 (clean).

**Mutation applied:** flipped exactly one byte on line 1 (`# Changelog` -> `# changelog`) via
`sed -i.bak '1s/^# Changelog$/# changelog/' "$S/wt/CHANGELOG.md"`, then `rm -f
"$S/wt/CHANGELOG.md.bak"` (portable across BSD and GNU sed).

**Applied-confirmation:** `cmp -l <(git -C "$S/wt" show HEAD:CHANGELOG.md) "$S/wt/CHANGELOG.md" |
wc -l` printed `1`.

**RED transcript** (verbatim):

```
$ task -d "$S/wt" check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
::error::check:changie: [3/11] changie merge --dry-run does not reproduce CHANGELOG.md byte-for-byte
--- CHANGELOG.md	2026-09-25 16:17:24
+++ /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.GvmiQjLnMx/merged.md	2026-09-25 16:17:25
@@ -1,4 +1,4 @@
-# changelog
+# Changelog
 
 ## [0.14.0](https://github.com/seanb4t/codegraph-go/compare/v0.13.0...v0.14.0) (2026-09-20)
 
task: Failed to run task "check:changie": exit status 1
```

Exit code: 201 (task's wrapping of the guard's `exit 1`).

**Revert:** `git -C "$S/wt" checkout -- CHANGELOG.md`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run:**

```
$ task -d "$S/wt" check:changie
...
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

---

## Family (b) — missing-PR refusal leg (CHG-04, D-11)

**Test/guard:** `check:changie` leg [5/11] (`changie new` with no `-m PR=` must be refused on
changie's own stderr text).

**What are we testing, and why?** Whether the missing-PR refusal leg has teeth — that it would
actually catch a `.changie.yaml` regression that made `PR` optional (D-11 forbids this: a
phase-close fragment with no PR number is Phase 2's problem to solve, not an `optional:` escape
hatch here).

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- .changie.yaml` exited 0 (clean).

**Mutation applied:** inserted the line `    optional: true` directly under `    minInt: 1` in the
`PR` custom entry of `"$S/wt/.changie.yaml"` (line 48).

**Applied-confirmation:** `rg -F -o 'optional: true' "$S/wt/.changie.yaml" | wc -l` printed `1`.

**RED transcript** (verbatim):

```
$ task -d "$S/wt" check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
::error::check:changie: [5/11] changie new with no -m PR= exited 0 (expected a refusal): 
task: Failed to run task "check:changie": exit status 1
```

Exit code: 201. With `optional: true` set, changie no longer refuses a fragment missing `PR` — it
silently writes one with no PR value, exactly the D-11 hazard this leg exists to catch.

**Revert:** `git -C "$S/wt" checkout -- .changie.yaml`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run:**

```
$ task -d "$S/wt" check:changie
...
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

---

## Family (c) — collision leg (CHG-04)

**Test/guard:** `check:changie` leg [9/11] (two back-to-back `new -k Fixes` calls must raise the
fragment count by exactly 2).

**What are we testing, and why?** Whether the sub-second `fragmentFileFormat` override in
`.changie.yaml` is actually load-bearing. 01-01-RESEARCH.md measured live that changie v1.26.0's
DEFAULT (seconds-only) fragment filename template silently overwrites a same-kind fragment written
within the same second — exactly the failure this override exists to prevent. Deleting the
override should reproduce that measured collision.

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- .changie.yaml` exited 0 (clean).

**Mutation applied:** deleted the `fragmentFileFormat: '{{.Kind}}-{{.Time.Format
"20060102-150405.000000000"}}'` line (line 33) from `"$S/wt/.changie.yaml"` via `sed -i.bak
'33d'`, then `rm -f "$S/wt/.changie.yaml.bak"`.

**Applied-confirmation:** `rg -F -o 'fragmentFileFormat:' "$S/wt/.changie.yaml" | wc -l` printed
`0`.

**RED transcript** (verbatim, attempt 1 of up to 5 permitted — no re-run needed, the collision
reproduced on the first attempt):

```
$ task -d "$S/wt" check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/11] changie new with no PR refused: custom missing and prompt is disabled: custom key 'PR' — ok
check:changie: [6/11] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/11] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/11] changie new -m PR=abc refused: invalid number — ok
::error::check:changie: [9/11] fragment count = 1 after two back-to-back new calls, want 3 (collision — sub-second fragmentFileFormat not honored)
task: Failed to run task "check:changie": exit status 1
```

Exit code: 201. Without the sub-second override, changie's default (seconds-only) template
collided the two `new -k Fixes` calls into the same filename, so the count stayed at 1 instead of
rising to 3 — the exact measured hazard from 01-01-RESEARCH.md, live-reproduced.

**Revert:** `git -C "$S/wt" checkout -- .changie.yaml`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run:**

```
$ task -d "$S/wt" check:changie
...
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

---

## Family (d) — Go seed/heading set equality in both directions (D-06, CHG-02)

**Test/guard:** `internal/upgrade/changie_shape_test.go#TestChangieVersionSeedsMatchChangelog`.

**What are we testing, and why?** Whether the Go guard that ties `.changes/v*.md` seeds to
`CHANGELOG.md`'s `## [x.y.z]` headings (set equality, both directions) actually fails when the two
sets diverge, in both directions: a seed present on disk that should not be (extra), and a heading
with no matching seed (missing).

**Deviation note (documented per the executor's HARD GATE on acceptance criteria):** the plan's
literal instruction for the first half — `rm -f .changes/v0.5.1.md`, expecting the resulting
`--- FAIL` to name `v0.5.1` — collides with `changieSeedFloor = 14`: the guard checks `len(seeds)
< changieSeedFloor` BEFORE it computes the seed/heading set-mismatch, so a bare deletion (14 -> 13
seeds) short-circuits on the floor message (`"...13 seed files, want at least 14"`), which never
names the specific missing version. This is not a bug in the guard — the floor check is correct,
existing 01-01 behavior, and arguably a stronger protection than a bare set-mismatch would be. To
reach the set-mismatch branch (and literally name `v0.5.1`, as the plan's acceptance criteria
require), a filler seed (`v0.14.0-filler.md`, a byte-copy of `v0.14.0.md`) was ALSO added
temporarily to keep the seed count at the floor (14) while `v0.5.1.md` stayed deleted. Both files
(the deletion and the filler) were reverted before the second half ran.

**Pre-mutation gate:** `git -C "$S/wt" status --porcelain -- .changes` — empty (clean, no
untracked files).

### First half — missing seed (`v0.5.1`)

**Mutation applied:** `rm -f "$S/wt/.changes/v0.5.1.md"`, then `cp "$S/wt/.changes/v0.14.0.md"
"$S/wt/.changes/v0.14.0-filler.md"` (the floor-compensating filler — see deviation note above).

**Applied-confirmation:** `ls "$S/wt/.changes/" | rg -E 'v0.5.1|filler'` printed only
`v0.14.0-filler.md`; `v0.5.1.md` was gone.

**RED transcript** (verbatim):

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangieVersionSeedsMatchChangelog$' -count=1 -v
=== RUN   TestChangieVersionSeedsMatchChangelog
    changie_shape_test.go:350: ../../.changes: 14 seed files; ../../CHANGELOG.md: 14 version headings
    changie_shape_test.go:383: ../../.changes vs ../../CHANGELOG.md: set mismatch — versions only in seeds: [v0.14.0-filler], versions only in changelog headings: [v0.5.1]
--- FAIL: TestChangieVersionSeedsMatchChangelog (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.217s
FAIL
```

The FAIL names `v0.5.1` (in "versions only in changelog headings") — the literal acceptance
requirement is satisfied.

**Revert:** `rm -f "$S/wt/.changes/v0.14.0-filler.md"` (removes the planted filler); `git -C
"$S/wt" checkout -- .changes/v0.5.1.md` (restores the deleted seed).

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain -- .changes` — empty.

**GREEN re-run:**

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangieVersionSeedsMatchChangelog$' -count=1 -v
=== RUN   TestChangieVersionSeedsMatchChangelog
    changie_shape_test.go:350: ../../.changes: 14 seed files; ../../CHANGELOG.md: 14 version headings
--- PASS: TestChangieVersionSeedsMatchChangelog (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.169s
```

### Second half — spurious extra seed (`v0.1.0`)

**Pre-mutation gate:** `git -C "$S/wt" status --porcelain -- .changes` — empty (clean).

**Mutation applied:** `cp "$S/wt/.changes/v0.2.0.md" "$S/wt/.changes/v0.1.0.md"` (a planted file
whose content is a real seed's, but whose version has no matching `CHANGELOG.md` heading).

**Applied-confirmation:** `ls "$S/wt/.changes/v0.1.0.md"` succeeded (file exists).

**RED transcript** (verbatim):

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangieVersionSeedsMatchChangelog$' -count=1 -v
=== RUN   TestChangieVersionSeedsMatchChangelog
    changie_shape_test.go:350: ../../.changes: 15 seed files; ../../CHANGELOG.md: 14 version headings
    changie_shape_test.go:383: ../../.changes vs ../../CHANGELOG.md: set mismatch — versions only in seeds: [v0.1.0], versions only in changelog headings: []
--- FAIL: TestChangieVersionSeedsMatchChangelog (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.164s
FAIL
```

The FAIL names `v0.1.0` (in "versions only in seeds") — the literal acceptance requirement is
satisfied, and this direction needed no floor workaround (adding a file only ever raises the count
above the floor).

**Revert:** `rm -f "$S/wt/.changes/v0.1.0.md"`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain -- .changes` — empty.

**GREEN re-run:**

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangieVersionSeedsMatchChangelog$' -count=1 -v
=== RUN   TestChangieVersionSeedsMatchChangelog
    changie_shape_test.go:350: ../../.changes: 14 seed files; ../../CHANGELOG.md: 14 version headings
--- PASS: TestChangieVersionSeedsMatchChangelog (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.155s
```

---

## Teardown

```
$ git worktree remove --force "$S/wt"
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go  b85d2eea [gsd/v0.15.0-milestone]
```

Matches the pre-start listing exactly (one entry, the main checkout).

**Main-tree cleanliness:**

```
$ git status --porcelain -- CHANGELOG.md .changie.yaml .changes
(empty)

$ task check:changie
...
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

No mutation ever touched the main working tree — every family ran exclusively inside the detached
scratch worktree at `$S/wt`, which no longer exists.
