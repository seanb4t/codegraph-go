---
phase: 07-codex-parity
plan: 10
subsystem: docs
tags: [agent-capabilities, drift-test, codex, tdd, mutation-log]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "the scope-flipped Codex Capabilities() literal (07-05/07-07), the CODEX-01/CODEX-05/CODEX-06 live verdicts (07-04/07-09), and the append-last hook-trust/A4 subagent-cooldown findings (07-09)"
provides:
  - "docs/AGENT-CAPABILITIES.md — the published, drift-tested per-harness capability table (8 AllTargets() x 2 scopes = 16 rows) linked from README.md"
  - "internal/agents/capability_doc_test.go — TestCapabilityDoc_MirrorsCapabilities and TestCapabilityDoc_VerificationColumn, the AGENT-14 honesty guard"
  - "07-MUTATION-LOG.md Family (h1)/(h2) — both drift/honesty guards demonstrated RED against a real planted mutation, reverted byte-clean"
affects: [07-11]

# Actuals (#2632)
actuals:
  tokens: 6830
  tasks: 2
  commits: 3
plan_head_before: 881fa926

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "docs/AGENT-CAPABILITIES.md follows the docs/LANGUAGE-CAPABILITY-MATRIX.md precedent exactly: a hand-authored markdown table plus a Go test (mirroring TestMatrix_DocMirrorsDescriptor's shape) that recomputes every code-derived cell from the single source of truth (Capabilities()) and fails on any drift"
    - "repoRoot resolution via runtime.Caller(0) rather than a cwd-relative filepath.Abs('..') — required because this test t.Chdir()s into a scratch temp dir before reading the doc; a cwd-relative resolution silently breaks post-chdir (caught live in the RED run, fixed before the GREEN commit)"

key-files:
  created:
    - docs/AGENT-CAPABILITIES.md
    - internal/agents/capability_doc_test.go
  modified:
    - README.md
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "Row shape chosen (planner discretion, D-27): `| \`<id>\` | <global|local> | <mcp> | <format> | <instructions> | <skill> | <hooks> | <nudge> | <verification> |` — 9 cells, split on `|`, verified by a dedicated parseCapabilityDocRows helper that skips the header/separator rows (neither has a backtick immediately after the leading pipe)."
  - "Verification-column sourcing per row: both Codex rows -> verified 2026-09-19 (07-LIVE-SESSIONS.md); Claude local -> verified 2026-09-19 (06-LIVE-SESSIONS.md); opencode local and Antigravity global -> verified 2026-09-18 (05-LIVE-SESSIONS.md); Cursor/Gemini CLI/Kiro (both scopes) -> [ASSUMED] with their vendor doc URL and 2026-09-18 fetch date (no live session ran, per 05-LIVE-SESSIONS.md); Hermes global -> [ASSUMED] (hermes-agent.ai/blog/hermes-mcp-integration-guide, re-fetched 2026-09-19 per the plan's own instruction); Claude global -> [ASSUMED] (code.claude.com/docs/en/mcp, fetched live this session 2026-09-19) after searching every 0[5-7]-LIVE-SESSIONS.md file and finding no dedicated Claude-global install+read session (06-LIVE-SESSIONS.md only exercised Claude at local scope)."
  - "A4 sentence documented as 07-09 recorded it: per-session AND per-subagent cooldown for Codex, exactly parallel to Claude Code — explicitly NOT the narrower per-session-only fallback research had left open."
  - "The D-17 skill-description-truncation note is worded to avoid contradicting this session's own required-reading evidence: the plan's action text directs documenting truncation as an advisory, while 07-LIVE-SESSIONS.md's B7 found `codex debug prompt-input`'s raw dump untruncated. Worded both facts together: the interactive listing may truncate (advisory, no SKILL.md change) while the raw debug dump showed the full text."

patterns-established: []

requirements-completed: []  # AGENT-14 is also declared by 07-11-PLAN.md (shared requirement, per-orchestrator-note); gsd_run query requirements.ready-ids reports it NOT ready until 07-11's SUMMARY also lands — see State Updates below.

coverage:
  - id: D1
    description: "docs/AGENT-CAPABILITIES.md publishes one row per registered target per scope (16 rows) with 7 code-derived columns plus a hand-kept verification column, linked from README.md's \"Use it from an agent\" section"
    requirement: "AGENT-14"
    verification:
      - kind: unit
        ref: "internal/agents/capability_doc_test.go#TestCapabilityDoc_MirrorsCapabilities"
        status: pass
      - kind: other
        ref: "README.md \"## Use it from an agent\" section links docs/AGENT-CAPABILITIES.md (grep-verified in Task 1's <verify>)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The verification column never overclaims: verified rows name an existing live-evidence file, [ASSUMED] rows name a source and fetch date, unsupported rows read n/a, and Cursor/Gemini CLI/Kiro stay [ASSUMED] at both scopes"
    requirement: "AGENT-14"
    verification:
      - kind: unit
        ref: "internal/agents/capability_doc_test.go#TestCapabilityDoc_VerificationColumn"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both drift/honesty guards are demonstrated able to fail against a real planted mutation and revert byte-clean (Family h1/h2)"
    requirement: "AGENT-14"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (h1)/(h2) — pasted --- FAIL transcripts, byte-clean revert, green control re-run"
        status: pass
    human_judgment: false

duration: ~18min (commit-to-commit span for this plan's 3 task commits)
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 10: Published Per-Harness Capability Table (AGENT-14) Summary

**Drift-tested `docs/AGENT-CAPABILITIES.md` (16 rows, 8 targets x 2 scopes) computed from `Capabilities()` via a new Go test, with an honest hand-kept verification column and both guards demonstrated RED against real planted mutations.**

## Performance

- **Duration:** ~18 min (commit-to-commit span: 17:30:26 prior plan close to 17:48:25 this plan's last commit)
- **Started:** 2026-09-19T17:43:15-04:00 (RED commit)
- **Completed:** 2026-09-19T17:48:25-04:00 (Family (h) mutation-log commit)
- **Tasks:** 2
- **Files modified:** 4 (2 created: `docs/AGENT-CAPABILITIES.md`, `internal/agents/capability_doc_test.go`; 2 modified: `README.md`, `07-MUTATION-LOG.md`)

## Accomplishments

- Published `docs/AGENT-CAPABILITIES.md`: a framing paragraph in the `docs/LANGUAGE-CAPABILITY-MATRIX.md` style, a legend, a 16-row capability table (`AllTargets()` order — antigravity, claude, codex, cursor, gemini, hermes, kiro, opencode — global then local for each), and a Notes section covering shared paths, Codex's trust-gated project config, the opt-in/hook-trust-gated nudge, the append-last hook-trust mitigation, the A4 per-session-AND-per-subagent cooldown finding, and the D-17 skill-description-truncation advisory.
- Wrote `internal/agents/capability_doc_test.go` (RED-first, per plan): `TestCapabilityDoc_MirrorsCapabilities` recomputes every code-derived cell (scope, mcp, format, instructions, skill, hooks, nudge) straight from `Capabilities()` for all 16 rows and fails on any mismatch, missing/extra/reordered row; `TestCapabilityDoc_VerificationColumn` enforces the verification cell's two allowed forms, the Cursor/Gemini CLI/Kiro `[ASSUMED]` rule, both Codex rows reading `verified` via `07-LIVE-SESSIONS.md`, and that the doc never contains the hook-trust-bypass flag (built by string concatenation in the test so the flag name never appears literally in this file either).
- Linked the doc from `README.md`'s "Use it from an agent" section.
- Added `07-MUTATION-LOG.md` Family (h1)/(h2): a planted codex/local hooks drift (`codex-json` -> `none`) turning `TestCapabilityDoc_MirrorsCapabilities` RED naming exactly `codex/local`/`hooks`; a planted Cursor-global verification overclaim (`[ASSUMED]` -> `verified 2026-09-18 (05-LIVE-SESSIONS.md)`) turning `TestCapabilityDoc_VerificationColumn` RED naming `cursor`. Both reverted byte-clean (`git diff --quiet` exit 0 before/after, exit 1 mid-mutation), with a green full-package control after each revert.

## Task Commits

Each task was committed atomically:

1. **Task 1a: RED — the failing capability-doc drift test** - `db662e68` (test)
2. **Task 1b: GREEN — publish the per-harness capability table** - `b6edd914` (docs)
3. **Task 2: Family (h) — prove the capability-doc guard can fail** - `fb184037` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## RED Transcript (Task 1, before docs/AGENT-CAPABILITIES.md existed)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestCapabilityDoc_MirrorsCapabilities$|TestCapabilityDoc_VerificationColumn$' -v
--- FAIL: TestCapabilityDoc_MirrorsCapabilities (0.00s)
    capability_doc_test.go:188: read /Volumes/Code/github.com/seanb4t/codegraph-go/docs/AGENT-CAPABILITIES.md: open /Volumes/Code/github.com/seanb4t/codegraph-go/docs/AGENT-CAPABILITIES.md: no such file or directory
--- FAIL: TestCapabilityDoc_VerificationColumn (0.00s)
    capability_doc_test.go:254: read /Volumes/Code/github.com/seanb4t/codegraph-go/docs/AGENT-CAPABILITIES.md: open /Volumes/Code/github.com/seanb4t/codegraph-go/docs/AGENT-CAPABILITIES.md: no such file or directory
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.139s
```

This is the target tests failing for the planned reason (the doc doesn't exist yet) — a real test failure, not a build error, satisfying `#3770`'s intentional-RED requirement. (An earlier, self-caught RED run failed on a *different*, wrong line — `capabilityDocRepoRoot`'s cwd-relative path resolution broke after `t.Chdir()`; fixed via `runtime.Caller(0)` before landing this commit — see Deviations.)

## Files Created/Modified

- `docs/AGENT-CAPABILITIES.md` — the published 16-row capability table, legend, and Notes.
- `internal/agents/capability_doc_test.go` — `TestCapabilityDoc_MirrorsCapabilities`, `TestCapabilityDoc_VerificationColumn`, and their shared row-parsing/rendering helpers.
- `README.md` — one new sentence under "## Use it from an agent" linking the doc.
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (h1)/(h2) sections.

## Verification-Column Sources Chosen Per Row

| Row | Verification | Source |
|---|---|---|
| `antigravity` global | verified 2026-09-18 | 05-LIVE-SESSIONS.md (Install + positive session + GEMINI.md read) |
| `antigravity` local | n/a | not supported |
| `claude` global | [ASSUMED] | code.claude.com/docs/en/mcp, fetched 2026-09-19 (live-fetched this session; no dedicated global install+read session exists in any `0[5-7]-LIVE-SESSIONS.md`) |
| `claude` local | verified 2026-09-19 | 06-LIVE-SESSIONS.md (`install --target claude --location local --pretool-nudge`) |
| `codex` global | verified 2026-09-19 | 07-LIVE-SESSIONS.md (CODEX-01 L1 global) |
| `codex` local | verified 2026-09-19 | 07-LIVE-SESSIONS.md (CODEX-01 L1 project + CODEX-05/06 post-flip) |
| `cursor` global/local | [ASSUMED] | cursor.com/docs/skills.md, fetched 2026-09-18 (no Cursor account — 05-LIVE-SESSIONS.md) |
| `gemini` global/local | [ASSUMED] | raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md, fetched 2026-09-18 (not installed — 05-LIVE-SESSIONS.md D-10) |
| `hermes` global | [ASSUMED] | hermes-agent.ai/blog/hermes-mcp-integration-guide, re-fetched 2026-09-19 per plan instruction |
| `hermes` local | n/a | not supported |
| `kiro` global/local | [ASSUMED] | kiro.dev/docs/skills/, fetched 2026-09-18 (not installed — 05-LIVE-SESSIONS.md D-10) |
| `opencode` global | [ASSUMED] | opencode.ai/docs/skills.md, fetched 2026-09-18 (05-LIVE-SESSIONS.md only ran opencode at project/local scope, D-12) |
| `opencode` local | verified 2026-09-18 | 05-LIVE-SESSIONS.md (installed project session, `debug skill`, where-is-X) |

Searched (per the plan's own instruction before choosing `[ASSUMED]` for Claude's global row): `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-LIVE-SESSIONS.md`, `.planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md`, `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md`, and `.planning/milestones/` (no archived `*-LIVE-SESSIONS.md` files exist there — confirmed via `find`). None of these ran a dedicated Claude-global install+read session (06-LIVE-SESSIONS.md's own scaffold used `--location local` only); Claude's global row is therefore `[ASSUMED]`, live-fetched this session against `code.claude.com/docs/en/mcp` (confirmed `~/.claude.json` as the user-scope MCP config path).

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `capabilityDocRepoRoot`'s cwd-relative path resolution breaking after `t.Chdir()`**
- **Found during:** Task 1 (writing `TestCapabilityDoc_MirrorsCapabilities`, before the RED commit)
- **Issue:** The helper mirrored `internal/indexer/capability/matrix_test.go`'s `repoRoot` pattern (`filepath.Abs(filepath.Join("..", ".."))`), which resolves relative to the process's *current working directory*. `TestCapabilityDoc_MirrorsCapabilities` calls `t.Chdir(t.TempDir())` (per the plan's own behavior text) before reading the doc, so the cwd-relative `"../.."` silently resolved inside the scratch temp dir instead of the repository — the first (pre-commit) test run failed with `resolved repo root ".../T" does not contain go.mod`, not the intended "file doesn't exist" RED.
- **Fix:** Resolved the repo root via `runtime.Caller(0)` (this source file's own compile-time path) instead of a cwd-relative `filepath.Abs`, walking up three `filepath.Dir` calls (file -> `internal/agents` -> `internal` -> repo root) — independent of whatever the test has `t.Chdir()`'d into.
- **Files modified:** `internal/agents/capability_doc_test.go`
- **Verification:** Re-ran the test before committing; it now fails on the *intended* line (the doc's own read), confirmed in the pasted RED transcript above.
- **Committed in:** `db662e68` (Task 1 RED commit — fixed before landing, never committed broken)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Caught and fixed before the RED commit landed; the committed RED transcript is the intended failure. No scope creep — the fix is entirely within `capability_doc_test.go`, the file this task's own `<files>` declares.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Self-Check: PASSED

- `docs/AGENT-CAPABILITIES.md` — FOUND
- `internal/agents/capability_doc_test.go` — FOUND
- `README.md` "## Use it from an agent" section links `docs/AGENT-CAPABILITIES.md` — FOUND (grep-verified)
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` Family (h1)/(h2) — FOUND
- Commit `db662e68` (test, RED) — FOUND in `git log --oneline --all`
- Commit `b6edd914` (docs, GREEN) — FOUND in `git log --oneline --all`
- Commit `fb184037` (docs, Family (h)) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/mcp/ -count=1` — re-run at SUMMARY time: `ok` for both packages
- `docs/AGENT-CAPABILITIES.md` has exactly 16 rows matching `^\| \`(antigravity|claude|codex|cursor|gemini|hermes|kiro|opencode)\` \| (global|local) \|` — confirmed (`rg -c` = 16)
- `docs/AGENT-CAPABILITIES.md` is byte-clean after both Family (h) mutations (`git diff --quiet` exit 0) — confirmed

## Next Phase Readiness

- Task 1's must-have artifacts and Task 2's mutation-log families are both complete; the plan's `<verification>` (`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/mcp/ -count=1` green, 16 rows, README link) passes.
- `07-VALIDATION.md`'s `07-DOCS` row (table half) is satisfiable now.
- `AGENT-14` is a shared requirement with `07-11-PLAN.md` (the `instructions.go` "4 of 8" comments and the MCP `instructions` skill sentence are explicitly out of this plan's scope per its own objective). This SUMMARY does **not** mark `AGENT-14` complete in `REQUIREMENTS.md` — the shared-ID gate defers that to whichever plan's SUMMARY lands last (`gsd_run query requirements.ready-ids` will report it ready once `07-11`'s SUMMARY also exists).
- No blockers for `07-11`.

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
