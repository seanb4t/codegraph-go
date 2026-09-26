---
phase: 02-phase-close-fragment-capability
reviewed: 2026-09-26T00:00:00Z
depth: deep
files_reviewed: 12
files_reviewed_list:
  - .changie.yaml
  - .gitignore
  - CONTRIBUTING.md
  - Taskfile.yml
  - internal/upgrade/changie_shape_test.go
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/capability.json
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/skills/changie-fragments/SKILL.md
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/test/run.sh
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/test/fixtures/bin/gh
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/test/fixtures/bin/fake-task
  - /Volumes/Code/github.com/seanb4t/gsd-capability-changie/README.md
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-09-26T00:00:00Z
**Depth:** deep
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Reviewed codegraph-go's Phase 2 changie-fragment-capability close-out (the
`.changie.yaml` PR-optional change, `CONTRIBUTING.md`'s new install
subsection, and the `check:changie` twelve-leg Taskfile guard) together with
the private `gsd-capability-changie` v0.1.1 source it depends on
(`write-fragments.sh`, its `SKILL.md`, `capability.json`, and the
`test/run.sh` suite).

The codegraph-go-side diffs (`.changie.yaml`, `.gitignore`,
`internal/upgrade/changie_shape_test.go`, the `CONTRIBUTING.md` subsection)
are internally consistent and match what the guard tests assert.

The capability's `write-fragments.sh` correctly avoids re-quoting or
`eval`-ing SUMMARY-derived text on its own side, and its `--list`/`--write`
skip/note taxonomy is implemented as documented. However, tracing the call
chain one hop further — from `write-fragments.sh`'s invocation of
`workflow.changie_command`, through the real value this repository actually
configures (`"task changie --"`, `.planning/config.json`), into
`Taskfile.yml`'s `changie:` task body — surfaces a command-injection path
that the capability's own security claim (and its `[argv-literal]` test)
does not actually cover, because the test's `task` stand-in does not
reproduce how the real `task` binary executes `CLI_ARGS`. See CR-01.

## Critical Issues

### CR-01: `workflow.changie_command = "task changie --"` re-shells a SUMMARY-derived fragment body, defeating the capability's own no-shell-reinterpretation guarantee

**File:** `Taskfile.yml:549` (root cause / fix location), cross-referenced against `/Volumes/Code/github.com/seanb4t/gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh:462-468` (the caller) and `/Volumes/Code/github.com/seanb4t/gsd-capability-changie/test/run.sh:853-881` + `/Volumes/Code/github.com/seanb4t/gsd-capability-changie/test/fixtures/bin/fake-task:39` (the test that gives false assurance)

**Issue:**

`write-fragments.sh` builds `NEW_ARGV=(new -k "${kind}" -b "${body}")` and
execs it as `"${CHANGIE_ARGV[@]}" "${NEW_ARGV[@]}"` — a plain `exec`-style
array expansion with no intervening shell, so `body` genuinely reaches the
immediate child process as one argv element. The script's own header
comment and the security register (`02-SECURITY.md` T-02-01) rely on this
to claim a SUMMARY-derived body can never be re-interpreted.

That guarantee breaks at the next hop. codegraph-go's actual configured
value for `workflow.changie_command` is:

```
$ grep -A1 changie_command .planning/config.json
    "changie_fragments": true,
    "changie_command": "task changie --"
```

which is exactly what `CONTRIBUTING.md`'s new subsection and the
capability's own `README.md` instruct ("set it to `task changie --` when
changie is not on `PATH`", which is codegraph-go's situation — `changie` is
built on demand from `go.tool-changie.mod`). So `CHANGIE_ARGV` becomes
`["task", "changie", "--"]`, and the child process actually invoked is the
real `task` (go-task) binary — not changie directly.

`Taskfile.yml`'s `changie:` task is:

```yaml
  changie:
    ...
    cmds:
      - "{{.GO_TOOL_CHANGIE}} changie {{.CLI_ARGS}}"
```

Per Task's own documentation, everything after `--` is captured as
`CLI_ARGS`, described explicitly as *"extra arguments as a string"* — as
opposed to `CLI_ARGS_LIST`, documented as *"a shell-parsed list"* derived
from that same string. Task also ships a dedicated `shellQuote`/`q`
template filter specifically because a raw value spliced into a `cmds:`
string is **not** shell-safe by default (Task's own template-reference
example labels an unquoted value `UNSAFE` for exactly this reason). Task's
`cmds:` entries are rendered as plain text and then executed through Task's
embedded POSIX shell (`mvdan.cc/sh`). This means the already-argv-safe
`body` string that `write-fragments.sh` carefully preserved is
re-serialized into a single line and **re-parsed by a shell a second time**
once it reaches the `changie:` task — with no escaping applied to any
special characters the body happens to contain.

A phase-close body sentence describing, say, a change to
shell-substitution handling — a completely ordinary thing for this
project's own changelog to say, e.g. *"Fixed `$(pwd)`-based path
resolution in the install script"* — would have `$(pwd)` (or any
`` ` `` / `;` / `&&` / `|` / glob) executed by Task's shell when the
`changie:` task's `cmds:` line is rendered and run, because `CLI_ARGS`
carries that text unescaped into the template. This is a real command
injection reachable from ordinary SUMMARY-derived English prose, not just
a deliberately adversarial payload — and it runs non-interactively inside
`verify:post`, i.e. automatically at phase close.

The capability's own test suite does not catch this because it doesn't
exercise the real `task` binary. `test/run.sh`'s `[argv-literal]` leg
(lines 848–881) proves the literal string
`Handles $(touch PWNED) and \`touch PWNED2\`; "quoted" * stars` survives
unexecuted — but it does so against `test/fixtures/bin/fake-task`
(line 344: `workflow.changie_command` is set to
`"${CAP_ROOT}/test/fixtures/bin/fake-task changie --"`), and `fake-task`
(fixtures/bin/fake-task:39) simply does `exec "${CHANGIE_BIN}" "$@"` —
i.e. it forwards argv with no shell involved at all. That stand-in
structurally cannot reproduce Task's `CLI_ARGS`-through-a-shell behavior,
so the test's pass gives false confidence about the one configuration
(`task changie --`) that this repository, and the capability's own
`README.md`, actually recommend.

**Fix:**

Change `Taskfile.yml`'s `changie:` task to consume the shell-parsed,
individually-quoted list instead of the raw string, e.g.:

```yaml
  changie:
    ...
    cmds:
      - "{{.GO_TOOL_CHANGIE}} changie{{range .CLI_ARGS_LIST}} {{. | shellQuote}}{{end}}"
```

(or equivalent — any form that quotes each forwarded argument before it is
handed to the shell). Then extend `test/run.sh`'s `[argv-literal]` leg (or
add a new leg) to drive the **real** `task` binary — not `fake-task` — with
`workflow.changie_command` literally set to
`"task changie --"` against a scratch Taskfile exposing the same
`changie:` shape, so the guard actually proves the production
configuration is safe rather than a structurally-different stand-in.

## Warnings

### WR-01: `--pr` validation is not anchored across the whole argument — a multi-line value passes as "a positive integer"

**File:** `/Volumes/Code/github.com/seanb4t/gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh:130-141`

**Issue:**

```bash
if [ -n "${PR_ARG}" ]; then
  case "${PR_ARG}" in
    [1-9]*)
      if ! printf '%s' "${PR_ARG}" | grep -Eq '^[1-9][0-9]*$'; then
        bad_pr "not a positive integer: ${PR_ARG}"
      fi
      ;;
    *)
      bad_pr "not a positive integer: ${PR_ARG}"
      ;;
  esac
fi
```

`grep` (without `-z`/multiline mode) applies `^...$` per line, not to the
whole buffer. A `--pr` value containing an embedded newline — e.g.
`--pr $'12\nDANGEROUS'` — starts with a digit 1-9 (passing the `case`
glob), and `grep -Eq '^[1-9][0-9]*$'` returns success because its *first
line* (`12`) matches, even though the full argument is not a clean
positive integer:

```
$ printf '12\nDANGEROUS' | grep -Eq '^[1-9][0-9]*$' && echo MATCHED
MATCHED
```

`PR="${PR_ARG}"` is then used verbatim in `-m "PR=${PR}"` (line 464),
so the un-rejected multi-line value flows straight into the `changie new`
invocation and, if `changie` accepts it, into the fragment's `custom.PR`
field and eventually `CHANGELOG.md`. This contradicts
`02-SECURITY.md`'s T-02-03, which is marked `closed` on the premise that
"`--pr` is still validated as a positive integer (`bad-pr`)". The
`gh`-lookup path is not affected (its value is passed through
`tr -d '[:space:]'` first, stripping any newline), so this only affects the
explicit `--pr` CLI argument.

**Fix:** Anchor the check to the whole string, e.g.:

```bash
case "${PR_ARG}" in
  *$'\n'*) bad_pr "not a positive integer: ${PR_ARG}" ;;
  [1-9]*[0-9]|[1-9])
    : # falls through to the case above only on a clean digit run
    ;;
esac
```

or simpler, use `printf '%s' "${PR_ARG}" | grep -Ezq '^[1-9][0-9]*$'`
(`-z` treats the input as one NUL-terminated record, so `^`/`$` anchor the
entire value) — GNU grep only; BSD/macOS `grep -z` behaves differently, so
verify on both platforms this script targets, or drop the case/grep split
entirely for a single POSIX pattern-match test:
`case "${PR_ARG}" in ''|*[!0-9]*|0*) bad_pr ... ;; esac` (rejects anything
containing a non-digit, including embedded newlines, and a leading zero).

## Info

### IN-01: `check:changie`'s "Leg N/11" comments were not updated when a 12th leg was added

**File:** `Taskfile.yml:604-858`

**Issue:** CHG-04/D-13 added a twelfth leg (`batch auto --dry-run`,
line 858) and every `echo`/`::error::` string was correctly renumbered from
`[N/11]` to `[N/12]`, and the trailing summary now reads "12 of 12" — but
the eleven `# Leg N/11: ...` *comments* above each leg block (lines 604,
630, 652, 673, 696, 731, 752, 773, 794, 813, 833) still say `/11`. A reader
skimming the comments (rather than the runtime output) sees a stale total
that no longer matches the file's own logic or its adjacent `desc:` text
("runs twelve numbered checks").

**Fix:** Update each `# Leg N/11:` comment to `# Leg N/12:` to match the
renumbered echo strings and the task's own description.

---

_Reviewed: 2026-09-26T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

---

## Orchestrator Verification (2026-09-26)

Each finding was re-checked against the live system before routing. The reviewer's text above is left unchanged.

| Finding | Verdict | Evidence |
|---------|---------|----------|
| CR-01 | **Not reproduced.** Downgraded from BLOCKER to INFO (test-coverage gap only) | See detail 1 below. |
| WR-01 | **Confirmed.** Stays WARNING, with limited reach | See detail 2 below. |
| IN-01 | **Confirmed.** Cosmetic | The `# Leg N/11` comments in `check:changie` are stale; the runtime strings already say 12. |

**1. CR-01.** The test used the real pinned go-task 3.52.0 with the configured `task changie --`, running `CI=true task changie -- new --dry-run -k Fixes -b "<payload>" -m PR=1`.

- There were 8 payloads:
  - `$(touch M)`;
  - a backticked `touch M`;
  - `; touch M`;
  - `&& touch M`;
  - mixed `'`, `"` and `$HOME`;
  - globs;
  - single-quote breakout;
  - double-quote breakout.

  One further case embedded a newline.
- Result: `executed-cases=8 exec-hits=0`. The marker file was never created, and every single-line body reached changie byte-literal.
- go-task shell-quotes `CLI_ARGS` when it renders `{{.CLI_ARGS}}`, so no re-interpretation happens.
- The real part of CR-01 is that `[argv-literal]` goes through `test/fixtures/bin/fake-task` and never through real `task`. The claim is proven only by this manual check, not by a committed guard.
- Suggested hardening, not a blocker: add a `check:changie` leg that runs `task changie -- new --dry-run -b '<$(…) payload>'` and asserts that the body is literal and no marker file was created.

**2. WR-01.**

- `write-fragments.sh --phase 2 --pr $'12\nDANGEROUS' --list` exits 0 and prints `pr 12`. `grep -Eq '^[1-9][0-9]*$'` matches line by line, so line 133 accepts a multi-line argument.
- Reach is limited:
  - only an explicit, operator-typed `--pr` is affected;
  - the automatic `gh` lookup (line 416) strips all whitespace before it validates;
  - changie's `type: int` parse still rejects a multi-line value at write time.
- Fix: capability v0.1.2 replaces the grep with a bash-3.2-safe whole-string match, for example `case "$PR_ARG" in ''|0*|*[!0-9]*) bad-pr ;; esac`, and adds a `[bad-pr:multiline]` leg that fails first.
- This needs a new tag push, which is not pre-approved.

**Net status:** 0 critical, 1 warning, 2 info. The WARNING is carried as a follow-up. It does not block the phase goal (CAP-01..04).

## Resolution (quick 260926-it7, 2026-09-26)

| Finding | Resolution | Evidence |
|---------|-----------|----------|
| WR-01 | Fixed in gsd-capability-changie `v0.1.2`. `--pr` and the looked-up PR are validated with a single whole-string `is_positive_int` helper, so an embedded or trailing newline, a sign, a leading zero or any non-digit character is refused before anything runs. `[bad-pr:multiline]` went RED first at test commit `801c5df` (19 `ok [` lines, then `expected exit 2, got 0`), then GREEN at fix commit `24abce3` (`33 of 33`). The suite passes 33 of 33 at HEAD (tag `v0.1.2`, commit `2631063`) and from a clean HTTPS clone. T-02-03's premise now holds. | `.planning/quick/260926-it7-release-gsd-capability-changie-v0-1-2-fi/260926-it7-SUMMARY.md` |
| CR-01 (residual) | Closed as test coverage. codegraph-go commit `79f13598` adds `check:changie` leg 13, which drives the real `task changie -- new --dry-run -k Fixes -b "<payload>" -m PR=1` from a scratch copy with a body carrying `$(...)`, a backticked command and a trailing `;`. The body arrives byte-literal (`body: <payload>`) and no marker file is created. The leg was proven able to fail: a confirmed-applied unquoted-splice mutation of the `changie:` task's `{{.CLI_ARGS}}` rendering, run in a disposable detached worktree, sent leg 13 RED with `re-shelled the fragment body`; the worktree was reverted byte-cleanly and re-ran GREEN. | `.planning/quick/260926-it7-release-gsd-capability-changie-v0-1-2-fi/260926-it7-SUMMARY.md` |
| IN-01 | Fixed in codegraph-go commit `79f13598`. All `# Leg N/11` and `# Leg N/12` comments, and every `[N/12]` string, in `check:changie` are renumbered to `/13` for the new 13-leg total; `desc:` and the pass-count guard were updated to match. | codegraph-go commit `79f13598` |

Both the codegraph-go project install and the maintainer's global install are repointed to `v0.1.2` through `gsd-tools capability install`/`capability set`, and `CONTRIBUTING.md` now names `#v0.1.2` in both install specs (commit `2d229c0d`). Full transcripts, including the RED/GREEN pastes and the mutation proof, are in `.planning/quick/260926-it7-release-gsd-capability-changie-v0-1-2-fi/260926-it7-SUMMARY.md`.
