# Phase 2: Phase-Close Fragment Capability - Pattern Map

**Mapped:** 2026-09-25
**Files analyzed:** 12 (new repo: 7 — capability.json, SKILL.md, write-fragments.sh, test/run.sh, README.md, LICENSE, optional CI; codegraph-go: 4 modified — `.gitignore`, `CONTRIBUTING.md`, `.planning/config.json` [tool-verb only], plus the GitHub-side draft PR which has no file analog; capability-repo test-log evidence pasted into a new codegraph-go phase artifact)
**Analogs found:** 11 / 12 (one item — the draft-PR checkpoint — has no code analog; it's a `gh` operational step, listed under No Analog Found)

**Read-only-source note:** the new repository `seanb4t/gsd-capability-changie` does not exist yet, so it has no git-tracked analog of its own. Every analog below for its files is either (a) a git-tracked file in **this** repo (`codegraph-go`, verified with `git ls-files`), or (b) explicitly-flagged read-only reference material outside any tracked repo in scope (`~/.claude/gsd-core/...`, the researcher's live-verified `capability.json` in `02-RESEARCH.md`) — cited for shape only, never as a copy-verbatim source, and never presented as a tracked analog.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `gsd-capability-changie/capability.json` | config (manifest) | request-response (declarative, read by gsd-core loader) | `02-RESEARCH.md` Code Examples "Minimal valid `capability.json`" (live-verified this phase's own research pass) + `~/.claude/gsd-core/bin/lib/capability-validator.cjs` (schema, read-only) | exact (values), no in-repo structural analog (new artifact class) |
| `gsd-capability-changie/skills/changie-fragments/SKILL.md` | controller (judgment half — dispatched via `Skill(gsd-changie-fragments)`) | request-response | `~/.claude/skills/gh-stack/SKILL.md` (read-only, outside any tracked repo in this scope — frontmatter + `gh`-driven non-interactive skill shape) | role-match (external reference only; no in-repo skill exists) |
| `gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh` | utility (deterministic half: preflight skips, PR lookup, idempotency check, `changie new`, commit) | transform / event-driven (reacts to phase-close dispatch) | `scripts/check-ruleset-drift.sh` (codegraph-go, git-tracked) — header-comment discipline, `set -euo pipefail`, named `::error::` exits, `command -v` preconditions, counted-before-compared assertions | role-match (same discipline, different domain) |
| `gsd-capability-changie/test/run.sh` | test (integration/proof script, D-09) | batch (drives every skip + write case, asserts named reasons) | `Taskfile.yml` `check:changie` target (codegraph-go, git-tracked, lines 551-849) — scratch-dir build, numbered legs, `passed` counter, zero-fails-loud assertion (rule `84d1gfpywd`) | exact (this is CONTEXT.md's own stated model, "shaped like `docs:cli:drift`/`check:changie`") |
| `gsd-capability-changie/README.md` | config (doc) | — | `CONTRIBUTING.md` (codegraph-go, git-tracked) tone/structure for tool-usage docs; no direct analog for a capability README | role-match |
| `gsd-capability-changie/LICENSE` | config (legal, verbatim MIT) | — | `LICENSE` (codegraph-go, git-tracked) — D-10 says "matching codegraph-go" explicitly | exact |
| `gsd-capability-changie/test/*` CI (optional, Claude's discretion) | config (CI) | request-response | `.github/workflows/ci.yml` `test` job's `docs:cli:drift`/`check:changie` step ordering (codegraph-go, git-tracked) | role-match |
| `.gitignore` (codegraph-go, modified) | config | file-I/O | itself — existing `.gsd/` line at `.gitignore:46` (git-tracked); add sibling line for `.gsd-capabilities.json` (Pitfall 2) | exact (in-place edit target) |
| `CONTRIBUTING.md` §What `.planning/` is (codegraph-go, modified) | config (doc) | transform | itself — the existing "What `.planning/` is" section (git-tracked, ~line 196) and the changie tool bullet (lines 122-128) for voice/structure | exact (in-place edit target) |
| `.planning/config.json` `workflow.changie_fragments` / `workflow.changie_command` (codegraph-go, tool-owned) | config (tool-owned, generated) | request-response | itself — written only via `gsd_run query config-set` (per planning-artifacts rule: never hand-edited) | exact (tool-verb only; **do not** hand-edit this file, do not invent structure in it) |
| Phase test-log evidence (new codegraph-go phase artifact, e.g. `02-TEST-LOG.md` or similar, per D-09's "pasted into codegraph-go's phase artifacts") | test (evidence record) | batch | `01-MUTATION-LOG.md` (codegraph-go, git-tracked) — the precedent named explicitly in `02-CONTEXT.md`'s Established Patterns ("A phase directory can carry extra evidence files… for proofs that run outside the repository") | exact (named precedent) |
| Draft PR `gsd/v0.15.0-milestone → main` (D-04, GitHub-side) | — (operational checkpoint, not a file) | — | none — a `gh pr create --draft` invocation, maintainer-approved checkpoint, not a code pattern | no analog (operational step; see No Analog Found) |

## Pattern Assignments

### `gsd-capability-changie/capability.json` (config manifest, request-response)

**Analog:** `02-RESEARCH.md` Code Examples "Minimal valid `capability.json`" — this is a **live-verified** artifact from this phase's own research pass (a real `capability install ./ --scope project` succeeded against it), not a generic template. Treat it as the locked starting shape; do not re-derive fields from scratch.

**Full shape to copy from** (`02-RESEARCH.md` lines 343-380):
```json
{
  "id": "changie",
  "role": "feature",
  "title": "Changie Fragments",
  "description": "Writes changie changelog fragments at phase close.",
  "tier": "standard",
  "requires": [],
  "version": "0.1.0",
  "engines": { "gsd": "^1.14.0" },
  "skills": ["changie-fragments"],
  "agents": [],
  "runtimeCompat": { "supported": ["*"], "unsupported": [] },
  "config": {
    "workflow.changie_fragments": {
      "type": "boolean",
      "default": true,
      "description": "Write changie changelog fragments automatically at phase close."
    },
    "workflow.changie_command": {
      "type": "string",
      "default": "changie",
      "description": "Command prefix used to invoke changie (e.g. \"task changie --\" when changie is not on PATH)."
    }
  },
  "steps": [
    {
      "point": "verify:post",
      "ref": { "skill": "changie-fragments" },
      "produces": [],
      "consumes": [],
      "when": "workflow.changie_fragments",
      "onError": "skip"
    }
  ],
  "contributions": [],
  "gates": []
}
```

**Load-bearing details the plan must not drop** (all `[VERIFIED]` in `02-RESEARCH.md`):
- `runtimeCompat` is **required** even though `02-CONTEXT.md`'s field list omits it (Pitfall 3) — omitting it produces `Capability validation failed: capability "changie" runtimeCompat must be an object with supported and unsupported arrays`.
- `ref.skill` is the **unprefixed stem** (`"changie-fragments"`), never `"gsd-changie-fragments"` — the validator rejects a `gsd-`-prefixed value here (double-prefix guard, `capability-validator.cjs:2901-2909`); the host prepends `gsd-` only at dispatch time.
- `id: "changie"` avoids reserved `gsd-`/`gsd-core-`/`anthropic-` prefixes (D-01).
- Config keys use dotted names (`workflow.changie_fragments`) — legal per `capability-validator.cjs:604-618` (Pattern 2 in RESEARCH.md); only `''`, `__proto__`/`constructor`/`prototype` and malformed type/default/description are rejected.

**Error handling pattern:** none in the manifest itself — validation errors surface at `capability install` time from the gsd-core loader, read-only and out of scope to patch (Out of Scope table forbids editing gsd-core).

---

### `gsd-capability-changie/skills/changie-fragments/SKILL.md` (controller, request-response)

**Analog:** `~/.claude/skills/gh-stack/SKILL.md` (read-only, external reference — not a tracked analog in any repo this phase modifies; cited for frontmatter and prose shape only).

**Frontmatter pattern** (external reference, `~/.claude/skills/gh-stack/SKILL.md` lines 1-9):
```yaml
---
name: gh-stack
description: >
  Manages stacked PRs and splits multi-part work into reviewable branches with gh-stack.
  Use for stack creation, viewing, edits, push, submit, sync, rebase, merge, or checkout;
  ...
metadata:
  author: github
  version: "0.1.0"
---
```
Adapt for `changie-fragments`: `name: changie-fragments`, a `description` naming the phase-close trigger and the manual re-run form (`/gsd-changie-fragments <phase> --pr <n>`), `metadata.version: "0.1.0"` matching the repo's first tag (D-10).

**Core pattern (per D-05):** SKILL.md is the **judgment half only** — it reads every `*-SUMMARY.md` in the phase directory (discovered via `gsd_run query init.phase-op <N>` per Claude's Discretion, with the CLI-surface caveat in `02-RESEARCH.md`'s "gsd-tools phase-dir resolution" note — confirm the installed binary's actual subcommand form, `init.phase-op` vs `init <workflow>`, at execution time), classifies each change per D-06 (user-visible vs not, kind from `.changie.yaml`'s parsed kinds, one plain-sentence body), and hands `kind<TAB>body` lines to `write-fragments.sh` via stdin or a file. It must **never** inline shell logic that touches SUMMARY-derived text directly — that's the deterministic script's job (Security Domain, V5).

**Error handling pattern:** SKILL.md documents (but does not itself implement) the four skip cases from D-03/CAP-03 — the actual skip logic and message text live in `write-fragments.sh`.

---

### `gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh` (utility, transform/event-driven)

**Analog:** `codegraph-go/scripts/check-ruleset-drift.sh` (git-tracked; verified `git ls-files`).

**Header-comment convention** (`scripts/check-ruleset-drift.sh` lines 1-38):
```bash
#!/usr/bin/env bash
# scripts/check-ruleset-drift.sh
#
# <what this does and why, one paragraph>
#
# Why this exists (<REQ-ID>, ...): <the specific hazard this guards against>
#
# This script is <scope note> — every failure path (<enumerate them>)
# is a named `::error::` and a non-zero exit — never a skip that silently
# passes.
#
# Usage: bash scripts/<name>.sh [--flag <value>]
#
# Environment:
#   VAR   description (default: ...)
```
Adapt: enumerate write-fragments.sh's real failure/skip paths in the header (disabled key, missing/broken configured command, no `.changie.yaml`, no PR, nothing new/already-recorded) — same enumeration discipline, but note these are **named skips that return cleanly**, not `::error::` failures (D-05: "no prompt, no block").

**Argv parsing pattern** (lines 40-59):
```bash
set -euo pipefail

FIXTURE_PATH=".github/required-status-checks.txt"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --fixture)
      if [ "$#" -lt 2 ]; then
        echo "usage: $(basename "$0") [--fixture <path>]" >&2
        exit 2
      fi
      FIXTURE_PATH="$2"
      shift 2
      ;;
    *)
      echo "usage: $(basename "$0") [--fixture <path>]" >&2
      exit 2
      ;;
  esac
done

for bin in curl jq; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "::error::ruleset-drift: required tool '${bin}' not found on PATH" >&2
    exit 1
  fi
done
```
Adapt for the `--pr <n>` argument (D-03: "an explicit argument wins over the lookup") using the same `case`/`shift 2` shape. Adapt the tool-precondition loop for `gh` and `git` (D-05: "gh and git are its only external tools besides changie") — but as **skip cases with named reasons**, not `::error::` exits, since a missing `gh` is D-03's documented skip path, not a hard failure.

**Command-invocation and non-interactive discipline** (`Taskfile.yml` `check:changie` target, git-tracked, lines 674-692 and 815-826 — pattern, not verbatim, per `02-RESEARCH.md` Pattern 3):
```bash
set +e
out=$(cd "${work}" && CI=true "${scratch}/changie" new -k Fixes -b "check:changie probe fragment" -m PR=1 2>&1)
status=$?
set -e
if [ "${status}" -ne 0 ]; then
  echo "::error::check:changie: [4/11] changie new (valid) exited ${status}: ${out}"
  exit 1
fi
count=$(fragment_count)
```
Adapt directly for `write-fragments.sh`'s core loop: one `CI=true <changie_command> new -k <Kind> -b "<body>" -m PR=<n>` per stdin/file entry, body passed as a **single argv element** (never re-quoted or `eval`'d — D-05, Security Domain V5), with a directory-diff/count-based fragment-discovery step before/after each call (see Pattern 3 below), since `changie new` prints nothing that names the file it wrote.

**Fragment-file discovery pattern** (`02-RESEARCH.md` Pattern 3, sourced from `check:changie` leg 4's `fragment_count()` helper, `Taskfile.yml` line 599-601, git-tracked):
```bash
fragment_count() {
  find "${work}/.changes/unreleased" -type f -name '*.yaml' | wc -l | tr -d ' '
}
```
`write-fragments.sh` snapshots this count (or a `find ... -newer <marker>` diff) before and after each `changie new` call to identify the exact fragment file written — never parse `changie new`'s stdout for a path.

**Idempotency pattern (D-07):** `git log <base>..HEAD --format='%(trailers:key=Changie-Summaries,valueonly)'` — verified round-trip in `02-RESEARCH.md` Code Examples ("Git trailer round-trip"):
```bash
git commit -m "$(printf 'docs(02): add changelog fragments\n\nChangie-Phase: 2\nChangie-Summaries: 02-01,02-02\n')"
git log --format='%(trailers:key=Changie-Phase,valueonly)|%(trailers:key=Changie-Summaries,valueonly)'
# → 2|02-01,02-02

git log ${BASE_BRANCH}..HEAD --format='%(trailers:key=Changie-Summaries,valueonly)'
```
Two-dot `git log A..HEAD` (not `ship.md`'s triple-dot diff form) is the verified-correct form for "this branch's own commits."

**`.changie.yaml` kinds-parsing pattern (no `jq`, D-05)** — `02-RESEARCH.md` Code Examples, verified against the real file:
```bash
awk '/^kinds:/{flag=1; next} /^custom:/{flag=0} flag && /label:/{sub(/^[ \t]*-[ \t]*label:[ \t]*/,""); print}' .changie.yaml
```

**Commit pattern (D-08):** explicit pathspec, never `git add -A`/`git add .`:
```bash
git add -- ".changes/unreleased/<file1>.yaml" ".changes/unreleased/<file2>.yaml"
git commit -m "$(printf 'docs(%s): add changelog fragments\n\nChangie-Phase: %s\nChangie-Summaries: %s\n' "${padded_phase}" "${phase}" "${plan_ids}")"
```

**Error handling pattern:** every skip case prints one distinct, named reason to stdout (or stderr — Claude's Discretion, but be consistent) and returns 0 — modeled on `check:changie`'s per-leg named-assertion style but inverted (a skip is a clean, documented non-write, not a test failure).

---

### `gsd-capability-changie/test/run.sh` (test, batch)

**Analog:** `Taskfile.yml` `check:changie` target (git-tracked, `Taskfile.yml:551-849`) — CONTEXT.md D-09 explicitly names this shape ("shaped like `docs:cli:drift`").

**Setup / scratch-dir pattern** (`Taskfile.yml` lines 574-601):
```bash
set -euo pipefail
scratch=$(mktemp -d)
trap 'rm -rf "${scratch}"' EXIT
work="${scratch}/work"

nseeds=$(find .changes -maxdepth 1 -type f -name 'v*.md' | wc -l | tr -d ' ')
echo "check:changie: found ${nseeds} seeded version files under .changes/ (floor 14)"
if [ "${nseeds}" -lt 14 ]; then
  echo "::error::check:changie: found only ${nseeds} seeded version files under .changes/ (floor 14) — a broken enumeration must fail loud, never read as a clean pass"
  exit 1
fi
```
Adapt: build a throw-away git repo with a fixture `.changie.yaml`, fixture SUMMARY files, and a stub `gh` on PATH (D-09 leg 1), then run `gsd_run capability install ./ --scope project` inside it.

**Numbered-leg / counted-assertion pattern** (`Taskfile.yml` lines 611-627, and the `set +e`/`set -e`/status-capture idiom repeated at every leg, e.g. lines 673-693):
```bash
passed=0
...
# Leg N/M: <case name>
set +e
out=$(cd "${work}" && <command> 2>&1)
status=$?
set -e
if [ "${status}" -ne <expected> ]; then
  echo "::error::check:changie: [N/M] <what> exited ${status}: ${out}"
  exit 1
fi
if ! printf '%s' "${out}" | grep -qF "<expected reason text>"; then
  echo "::error::check:changie: [N/M] <case> — output does not contain the expected reason: ${out}"
  exit 1
fi
echo "check:changie: [N/M] <case> — ok"
passed=$((passed + 1))
```
Adapt for every D-09 leg: CAP-01 install, CAP-02 both `render-hooks` states, and each of CAP-03's named skip cases (disabled key, missing/broken `changie_command`, no `.changie.yaml`, no PR, nothing-new/already-recorded) plus the positive write case — asserting the exact fragment file and trailer content, not a bare exit code (per D-09 and rule `84d1gfpywd`).

**Final zero-fails-loud assertion** (`Taskfile.yml` lines 851-857):
```bash
if [ "${passed}" -ne 11 ]; then
  echo "::error::check:changie: only ${passed} of 11 checks executed"
  exit 1
fi
echo "check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)"
```
Adapt the literal count to however many legs `test/run.sh` ends up with; the rule (`84d1gfpywd`) is: print the executed-case count and fail loud on zero, never let a broken enumeration read as a silent pass.

**CAP-02 assertion source** (`02-RESEARCH.md` Code Examples, "Real `render-hooks` output, both states" — live-verified this session):
```bash
gsd-tools loop render-hooks verify:post --raw
# key absent/true → activeHooks[] contains {capId:"changie", ref:{skill:"changie-fragments"}, when:"workflow.changie_fragments", onError:"skip"}
gsd-tools config-set workflow.changie_fragments false --raw
gsd-tools loop render-hooks verify:post --raw
# → activeHooks no longer contains capId:"changie"
```

---

### `gsd-capability-changie/README.md` and `LICENSE` (config/doc)

**LICENSE analog:** `codegraph-go/LICENSE` (git-tracked) — D-10 states "matching codegraph-go" explicitly; copy verbatim, MIT text only, no appended attribution paragraph (this repo's own CLAUDE.md notes an appended paragraph downgrades GitHub's license detection — same rule applies to the new repo).

**README.md structure:** no direct in-repo analog; content is dictated by D-10's enumerated list (what the capability does, config keys, install command with `git+ssh…#<tag>`, manual re-run, skip reasons, trailer semantics) — use `CONTRIBUTING.md`'s prose voice (direct, second-person-avoidant, technical) as the tone reference, not its structure.

---

### `.gitignore` (codegraph-go, modified)

**Analog:** itself — existing `.gsd/` line (`git ls-files` confirms tracked, `.gitignore:46`):
```
# GSD run-scoped dispatch-isolation sentinel (ephemeral session state)
.gsd/
```
**Required addition (Pitfall 2, `[VERIFIED]` in `02-RESEARCH.md`):** a **second**, sibling line for `.gsd-capabilities.json` — the install ledger lands at the **project root**, not inside `.gsd/`, so the existing line does not cover it:
```
# gsd-core capability install ledger (project scope) — project root, not
# under .gsd/ (capability-ledger.cjs: LEDGER_FILE_NAME joined against
# runtimeDir directly, not the .gsd/ subdirectory)
.gsd-capabilities.json
```
After D-11's install, `git status --porcelain` must show nothing but `.planning/config.json` and `.gitignore` — verify this literally, per D-11's own acceptance bar.

---

### `CONTRIBUTING.md` §What `.planning/` is (codegraph-go, modified)

**Analog:** itself — the existing section (git-tracked, confirmed at `CONTRIBUTING.md` around line 196):
```markdown
## What `.planning/` is

Roughly half the tracked files live in `.planning/`. It is the project's
planning and decision record — phase plans, execution summaries, debug sessions,
and the reasoning behind decisions that are otherwise invisible in the diff.

It is published deliberately. If you want to know *why* something is the way it
is, the answer is usually there, and it is usually more candid than a commit
message. You are not expected to add to it, and PRs are not judged on it.
```
**Voice reference for the new subsection** — the existing changie-tool bullet (git-tracked, `CONTRIBUTING.md` lines ~122-128):
```markdown
- `task`, `goreleaser`, `actionlint`, and `changie` build on demand from
  `go.tool.mod`, `go.tool-lint.mod`, and `go.tool-changie.mod` — there is
  nothing to install first, only Go and whatever toolchain the target
  itself needs. changie runs as `task changie`, for example
  `task changie -- latest`.
```
Adapt: add a short subsection under (not replacing) "What `.planning/` is" per D-12, covering the five enumerated points (one-time install command + tag, private-repo optionality, manual fallback `task changie -- new …`, Phase 4 gate as backstop, manual re-run). Do **not** place this under §Pull requests (reserved for `DOCS-12`, Phase 5, per D-12).

---

### `.planning/config.json` (tool-owned, codegraph-go)

**Pattern:** **never hand-edit.** Per the planning-artifacts rule and D-11, both keys are written exclusively through `gsd_run query config-set` (research confirms the actual installed binary form may be `gsd-tools config-set ... --raw` without a `query` prefix — verify the real CLI surface at execution time, per `02-RESEARCH.md`'s CLI-surface note). Verified ordering (`02-RESEARCH.md` "D-11 ordering, both legs verified live"):
```bash
# BEFORE capability install: rejected (key not yet federated)
$ gsd-tools config-set workflow.changie_command "task changie --" --raw
Error: Unknown config key: "workflow.changie_command". Valid keys: ...   # exit 1

# AFTER capability install --scope project: accepted
$ gsd-tools config-set workflow.changie_command "task changie --" --raw
workflow.changie_command=task changie --   # exit 0
```
Install must run **before** either `config-set` call.

---

### Phase evidence artifact (new codegraph-go file, e.g. `02-TEST-LOG.md`)

**Analog:** `01-MUTATION-LOG.md` (git-tracked, `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md`) — explicitly named in `02-CONTEXT.md`'s Established Patterns as the precedent for "a phase directory can carry extra evidence files for proofs that run outside the repository."

**Structure to copy** (`01-MUTATION-LOG.md` lines 1-30):
```markdown
# 0N-MUTATION-LOG — <Phase Name>

**Phase:** NN-phase-slug
**Date:** YYYY-MM-DD
**Scope:** <what is being proven and why>

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>`
(or `git status --porcelain -- <dir>`) is asserted to exit empty/clean...

## Setup

```
$ git worktree add --detach "$S/wt" HEAD
...
```
```
Adapt for D-09's requirement: paste `test/run.sh`'s real run transcript, the RED demonstration (mutate a skip/write guard, confirm applied, revert byte-cleanly — same worktree-isolation discipline as `01-MUTATION-LOG.md`), and the one real skill-dispatch transcript proving the model-judgment half (D-06) against fixture SUMMARYs with one user-visible and one planning-only change.

## Shared Patterns

### Positive-assertion / never-pass-vacuously discipline (rule `84d1gfpywd`)
**Source:** `scripts/check-ruleset-drift.sh` and `Taskfile.yml` `check:changie`/`docs:cli:drift` (all git-tracked)
**Apply to:** `write-fragments.sh`'s skip-case reporting, `test/run.sh`'s leg counter, and the CAP-04 materialization verification (`find ~/.claude/skills -iname '*changie*'`).
```bash
echo "check:changie: found ${nseeds} seeded version files under .changes/ (floor 14)"
if [ "${nseeds}" -lt 14 ]; then
  echo "::error::check:changie: found only ${nseeds} seeded version files ... — a broken enumeration must fail loud, never read as a clean pass"
  exit 1
fi
```
Every count is reported *before* it's used in a comparison; every enumeration is bounds-checked; a zero-count path is always a named `::error::`, never a silent pass.

### Drift/live-tool guard shape (scratch dir, byte-isolation, named errors)
**Source:** `Taskfile.yml` `docs:cli:drift` (lines 494-535) and `check:changie` (lines 551-857), both git-tracked
**Apply to:** `test/run.sh` (D-09's own stated model)
```bash
scratch=$(mktemp -d)
trap 'rm -rf "${scratch}"' EXIT
before=$(find .changie.yaml .changes CHANGELOG.md -type f -exec cksum {} + | LC_ALL=C sort)
# ... run everything against ${scratch}/work, never the source tree ...
after=$(find .changie.yaml .changes CHANGELOG.md -type f -exec cksum {} + | LC_ALL=C sort)
if [ "${before}" != "${after}" ]; then
  echo "::error::check:changie: the source tree changed during the run"
  exit 1
fi
```

### Untrusted-shell-input discipline (Security Domain V5)
**Source:** D-05 (CONTEXT.md), corroborated by `~/.claude/gsd-core/references/loop-hook-dispatch.md`'s identical warning for `ref.command`/gate `predicate` values (read-only, cited in `02-RESEARCH.md` Security Domain)
**Apply to:** every point where SUMMARY-derived text reaches a shell in `write-fragments.sh`.
- Pass the body as a **single argv element**: `"${changie_cmd[@]}" new -k "${kind}" -b "${body}" -m "PR=${pr}"` — never `eval "${body}"`, never re-interpolate into a re-quoted string.
- Same discipline for any `when`/`ref` value a manifest declares — never build a shell command by string concatenation of model- or manifest-derived text.

### Tool-owned-file discipline (planning-artifacts rule)
**Source:** this session's user CLAUDE.md rule, corroborated by D-11's own "never by hand-editing `.planning/config.json`"
**Apply to:** `.planning/config.json` (config-set only), `.gsd-capabilities.json` (install ledger, never hand-written), `CHANGELOG.md`/`.changes/v*.md` (changie-owned, per `.changie.yaml`'s own header comment, git-tracked).

### `git status --porcelain` clean-apart-from-allowlist checkpoint
**Source:** D-11's own acceptance bar
**Apply to:** every CAP-04 install sub-step.
```bash
git status --porcelain
# must show nothing but .planning/config.json and .gitignore after install + config-set
```

## No Analog Found

| File / Step | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Draft PR `gsd/v0.15.0-milestone → main` (D-04) | — (GitHub operational step) | — | Not a code artifact; a `gh pr create --draft` invocation performed as a maintainer-approved checkpoint. `02-RESEARCH.md` Pitfall 4 (read `~/.claude/gsd-core/workflows/ship.md:360-378`, read-only) confirms `/gsd-ship`'s `create_pr` step calls `gh pr create` unconditionally with no existing-PR probe — document this as an operational fact, do not attempt a code fix (editing `ship.md` is out of scope). |
| Global-scope Claude skill materialization (`capability set changie --enable --runtime claude --scope global`, Pitfall 1) | — (operational verification step, CAP-04) | — | Not a file to write; a required-but-unverified-live CLI step (`02-RESEARCH.md` Assumption A3) plus a `find ~/.claude/skills -iname '*changie*'` assertion. No code pattern to copy; the plan must add this as an explicit, separately-verified task, not fold it into "install the capability." |

## Metadata

**Analog search scope:** `codegraph-go/Taskfile.yml`, `codegraph-go/scripts/`, `codegraph-go/.gitignore`, `codegraph-go/CONTRIBUTING.md`, `codegraph-go/.changie.yaml`, `codegraph-go/.planning/phases/01-changie-baseline/` (all git-tracked, verified via `git ls-files`); `~/.claude/gsd-core/*` and `~/.claude/skills/gh-stack/SKILL.md` (read-only external reference material, outside any repo this phase modifies, cited for shape only per the phase's own explicit permission — never presented as a tracked analog).
**Files scanned:** ~15 (Taskfile.yml targets, scripts/*.sh, .gitignore, CONTRIBUTING.md, .changie.yaml, .planning/config.json, 01-PATTERNS.md, 01-MUTATION-LOG.md, gh-stack SKILL.md, gsd-core capability-validator.cjs excerpts already quoted in 02-RESEARCH.md).
**Pattern extraction date:** 2026-09-25
