# 02-MUTATION-LOG — Phase-Close Fragment Capability (D-13 PR-optional)

**Phase:** 02-phase-close-fragment-capability
**Date:** 2026-09-26
**Scope:** RED-first evidence and mutation-family demonstrations for the re-pointed
codegraph-go guards that D-13 (the changie `PR` custom field becomes optional)
touches: `internal/upgrade/changie_shape_test.go#TestChangieConfigShape` (the
`custom[0]` key set) and `check:changie`'s legs 5, 7 and 12 in `Taskfile.yml`.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, and before every revert, `git diff --quiet --
<file>` (or, for a directory-scoped check, `git status --porcelain -- <dir>`) is
asserted to exit empty/clean. This proves no pre-existing tracked edit was
overwritten by the mutation, and no revert was a destructive blind checkout of
someone else's in-flight work. Every family entry below records this gate's
result at the point it was checked. This log follows `01-MUTATION-LOG.md`'s
convention exactly (pre-mutation gate, applied confirmation, verbatim RED
transcript, byte-clean revert, GREEN re-run).

## RED-first transcripts (Task 1)

### Go RED — `TestChangieConfigShape`, committed before `.changie.yaml` changed

Command: `GOWORK=off go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`,
run immediately after the test-file-only commit `test(02-06): expect the changie
PR field to be optional (D-13)` and before `.changie.yaml` gained `optional: true`.

```
--- FAIL: TestChangieConfigShape (0.00s)
changie_shape_test.go:260: ../../.changie.yaml: custom[0] key set = [key minInt type], want exactly [key minInt optional type] (an absent optional key, or any other key, must fail — 02-CONTEXT D-13, which supersedes D-11)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.186s
FAIL
```

This is a genuine assertion failure naming the `custom[0]` key set (the real,
pre-change `.changie.yaml` has no `optional` key yet) — not a build failure or a
harness crash. Per the file's own header comment, this test is never gated on
`gsd-tools check tdd-red-evidence` (it parses TAP only and always reports
`INVALID_RED` for `go test`); the RED evidence is this commit's position on
`main..HEAD` plus this pasted `--- FAIL` transcript.

### `check:changie` leg 5 RED — re-pointed leg against the real, pre-change `.changie.yaml`

Command: `task check:changie`, run after `Taskfile.yml`'s leg 5 (and legs 6-9's
count expectations) were re-pointed to expect acceptance, but before
`.changie.yaml` gained `optional: true` — so the real host config still
refuses a no-PR fragment.

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
::error::check:changie: [5/11] changie new with no -m PR= exited 1 (expected acceptance, D-13): Error: custom missing and prompt is disabled: custom key 'PR'
custom missing and prompt is disabled: custom key 'PR'
task: Failed to run task "check:changie": exit status 1
```

Exit code 201 (task's wrapping of the guard's `exit 1`). This is a genuine
assertion failure on the leg's newly-planned behavior (the fragment must now
be *accepted* without `-m PR=`), not a harness crash — the real, unedited
`.changie.yaml` still declares `PR` without `optional: true`, so changie's own
refusal fires exactly as it did before this plan.

### GREEN dry-run confirmation — real host config accepts a no-PR fragment

Command: `CI=true task changie -- new --dry-run -k Fixes -b "no-PR tracer probe"`,
run against the real, now-edited `.changie.yaml` (after `optional: true` was
added), from the plan's own `<verify>` block.

```
kind: Fixes
body: no-PR tracer probe
time: 2026-09-26T11:01:29.764608-04:00
```

No `PR:` line, no `custom:` block. `git status --porcelain -- .changes` was
empty afterward — the `--dry-run` flag wrote nothing.

## Setup (Task 2)

```
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                        8c4d3f7a [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]

$ S=$(mktemp -d)
$ git worktree add --detach "$S/wt" HEAD
Preparing worktree (detached HEAD 8c4d3f7a)
HEAD is now at 8c4d3f7a feat(02-06): prove no-PR fragments render without a link in check:changie (CHG-04, D-13)
```

No commits were made in the worktree, and the repo configures no hooks, so no
hook installation was needed. `worktree-brave-river-a240` is a pre-existing,
unrelated worktree from another active session — present at both setup and
teardown, untouched throughout.

**Positive control:**

```
$ task -d "$S/wt" check:changie
...
check:changie: [11/12] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: [12/12] changie batch auto --dry-run renders the no-PR fragment without a link and a PR fragment with its link (dry run wrote nothing) — ok
check:changie: 12 of 12 checks passed against a scratch copy (source tree byte-unchanged)
```

---

## Family (a) — PR-optional key guard (D-13, rule 84d1gfpywd)

**What is tested, and why?** Whether the re-pointed `check:changie` leg 5 and
`TestChangieConfigShape`'s custom-key-set assertion actually have teeth — that
removing the `optional: true` key really does restore the pre-D-13 refusal,
rather than the guards passing vacuously.

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- .changie.yaml` exited 0
(clean).

**Mutation applied:** deleted the `    optional: true` line from
`"$S/wt/.changie.yaml"` via `sed -i.bak '/^    optional: true$/d'`, then `rm -f
"$S/wt/.changie.yaml.bak"`.

**Applied confirmation:** `grep -F -o 'optional: true' "$S/wt/.changie.yaml" |
wc -l` printed `0`.

**RED transcript — `task -d "$S/wt" check:changie`:**

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/12] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/12] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/12] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/12] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/12] changie new -k Fixes writes exactly one fragment with kind/body set — ok
::error::check:changie: [5/12] changie new with no -m PR= exited 1 (expected acceptance, D-13): Error: custom missing and prompt is disabled: custom key 'PR'
custom missing and prompt is disabled: custom key 'PR'
task: Failed to run task "check:changie": exit status 1
```

Fired exactly at the predicted leg, `[5/12]`.

**RED transcript — `GOWORK=off go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:**

```
changie_shape_test.go:260: ../../.changie.yaml: custom[0] key set = [key minInt type], want exactly [key minInt optional type] (an absent optional key, or any other key, must fail — 02-CONTEXT D-13, which supersedes D-11)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.230s
FAIL
```

Names the `custom[0]` key set, exactly as predicted.

**Revert:** `git -C "$S/wt" checkout -- .changie.yaml`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run (both):** `task -d "$S/wt" check:changie` ends "12 of 12 checks
passed against a scratch copy (source tree byte-unchanged)"; the Go test prints
`--- PASS: TestChangieConfigShape (0.00s)`.

---

## Family (b) — changeFormat PR-conditional guard (CHG-04, D-13, rule 84d1gfpywd)

**What is tested, and why?** Whether leg 12 (`batch auto --dry-run` renders a
no-PR fragment without a link) and `TestChangieConfigShape`'s `changeFormat`
assertion have teeth — that removing the `{{if .Custom.PR}}...{{end}}`
conditional really does break the no-link rendering, rather than the guards
passing vacuously because the two fragment kinds happen to render identically
either way.

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- .changie.yaml` exited 0
(clean).

**Mutation applied:** replaced line 35 of `"$S/wt/.changie.yaml"` —
`changeFormat: '- {{.Body}}{{if .Custom.PR}} ([#{{.Custom.PR}}](...)){{end}}'`
— with the unconditional form `changeFormat: '- {{.Body}} ([#{{.Custom.PR}}]
(https://github.com/seanb4t/codegraph-go/pull/{{.Custom.PR}}))'` (keeps the
link text, drops the `{{if .Custom.PR}}` / `{{end}}` wrappers), via `sed
-i.bak` (a `|`-delimited substitution, since the replacement text itself
contains `#`), then `rm -f "$S/wt/.changie.yaml.bak"`.

**Applied confirmation:** `grep -F -o '{{if .Custom.PR}}' "$S/wt/.changie.yaml"
| wc -l` printed `0`.

**RED transcript — `task -d "$S/wt" check:changie`:**

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/12] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/12] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/12] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/12] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/12] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/12] changie new with no PR accepted: fragment written without a PR field — ok
check:changie: [6/12] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/12] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/12] changie new -m PR=abc refused: invalid number — ok
check:changie: [9/12] two back-to-back new -k Fixes calls raised the fragment count to 4 (sub-second fragmentFileFormat holds) — ok
check:changie: [10/12] with only Fixes fragments present, next auto = v0.14.1 (patch successor of v0.14.0) — ok
check:changie: [11/12] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
::error::check:changie: [12/12] batch auto --dry-run missing the exact no-PR line
## [v0.15.0](https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0) — 2026-09-26
### Breaking
- check:changie breaking probe ([#4](https://github.com/seanb4t/codegraph-go/pull/4))
### Fixes
- check:changie probe fragment ([#1](https://github.com/seanb4t/codegraph-go/pull/1))
- check:changie probe without PR ([#<no value>](https://github.com/seanb4t/codegraph-go/pull/<no value>))
- check:changie collision probe one ([#2](https://github.com/seanb4t/codegraph-go/pull/2))
- check:changie collision probe two ([#3](https://github.com/seanb4t/codegraph-go/pull/3))task: Failed to run task "check:changie": exit status 1
```

Fired exactly at the predicted leg, `[12/12]` — the unconditional template
renders `[#<no value>]` for the no-PR fragment instead of omitting the link.

**RED transcript — `GOWORK=off go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:**

```
changie_shape_test.go:211: ../../.changie.yaml: changeFormat = "- {{.Body}} ([#{{.Custom.PR}}](https://github.com/seanb4t/codegraph-go/pull/{{.Custom.PR}}))", want "- {{.Body}}{{if .Custom.PR}} ([#{{.Custom.PR}}](https://github.com/seanb4t/codegraph-go/pull/{{.Custom.PR}})){{end}}"
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.172s
FAIL
```

Fails on the `changeFormat` mismatch, exactly as predicted.

**Revert:** `git -C "$S/wt" checkout -- .changie.yaml`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run (both):** `task -d "$S/wt" check:changie` ends "12 of 12 checks
passed against a scratch copy (source tree byte-unchanged)"; the Go test prints
`--- PASS: TestChangieConfigShape (0.00s)`.

---

## Family (c) — minInt PR=0 refusal guard (CHG-04, D-13, rule 84d1gfpywd)

**What is tested, and why?** Whether leg 7 (`PR=0` still refused) and
`TestChangieConfigShape`'s `minInt` assertion have teeth under the now-optional
config — proving the amended CHG-04's "PR=0 is still refused by minInt: 1"
half actually holds, rather than `optional: true` having silently loosened
`minInt` too.

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- .changie.yaml` exited 0
(clean).

**Mutation applied:** deleted the `    minInt: 1` line from
`"$S/wt/.changie.yaml"` via `sed -i.bak '/^    minInt: 1$/d'`, then `rm -f
"$S/wt/.changie.yaml.bak"`.

**Applied confirmation:** `grep -F -o 'minInt:' "$S/wt/.changie.yaml" | wc -l`
printed `0`.

**RED transcript — `task -d "$S/wt" check:changie`:**

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/12] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/12] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/12] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/12] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/12] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/12] changie new with no PR accepted: fragment written without a PR field — ok
check:changie: [6/12] changie new -k Undeclared refused: invalid kind: Undeclared — ok
::error::check:changie: [7/12] changie new -m PR=0 exited 0 (expected a refusal): 
task: Failed to run task "check:changie": exit status 1
```

Fired exactly at the predicted leg, `[7/12]` — PR=0 is now silently accepted.

**RED transcript — `GOWORK=off go test ./internal/upgrade/ -run '^TestChangieConfigShape$' -count=1 -v`:**

```
changie_shape_test.go:260: ../../.changie.yaml: custom[0] key set = [key optional type], want exactly [key minInt optional type] (an absent optional key, or any other key, must fail — 02-CONTEXT D-13, which supersedes D-11)
--- FAIL: TestChangieConfigShape (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.159s
FAIL
```

Fails on the key set, exactly as predicted.

**Revert:** `git -C "$S/wt" checkout -- .changie.yaml`.

**Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` — empty.

**GREEN re-run (both):** `task -d "$S/wt" check:changie` ends "12 of 12 checks
passed against a scratch copy (source tree byte-unchanged)"; the Go test prints
`--- PASS: TestChangieConfigShape (0.00s)`.

---

## Teardown (Task 2)

```
$ git worktree remove --force "$S/wt"
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                        8c4d3f7a [gsd/v0.15.0-milestone]
/Volumes/Code/herdr-worktrees/codegraph-go/worktree-brave-river-a240 2df78be0 [chore/octopusignore]

$ git status --porcelain -- .changie.yaml .changes CHANGELOG.md
(empty)
```

Matches the setup listing exactly (main checkout plus the pre-existing,
unrelated `worktree-brave-river-a240`), and the main tree is clean.

**3 of 3 mutation families went RED on the named guard and back to GREEN.**

<!-- gsd:write-continue -->
