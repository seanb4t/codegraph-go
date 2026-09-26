# Phase 3: Rename-Stub Removal - Discussion Log

> **Audit trail only.** Do not use this log as input to planning, research or execution agents.
> Decisions are captured in CONTEXT.md. This log keeps the alternatives that were considered.

**Date:** 2026-09-26
**Phase:** 03-rename-stub-removal
**Mode:** `--auto`. Every gray area was auto-selected, and each question was answered with the recommended option.
**Areas discussed:** Unknown-command surface, Removal guard, Commit shape, Census and the fragment, Capability evidence location, Inaccurate fragment handling

---

## Unknown-command surface

| Option | Description | Selected |
|--------|-------------|----------|
| Plain cobra unknown-command (recommended) | `legacyArgs` error, exit 1, no hint | ✓ |
| `SuggestFor` hint on `search` / `daemon` | "Did you mean …" line pointing at the replacement | |

**Choice:** [auto] Plain cobra unknown-command (recommended default).
**Notes:** ROADMAP SC1 says "rather than printing a rename message". A hint that names the replacement verb is still a rename message.

## Removal guard

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite the integration test as a binary-level unknown-command guard (recommended) | Keeps the four invocations and asserts exit 1 and the exact stderr. Positive executed-case count. RED-first. | ✓ |
| Delete both tests, no replacement | SC1 checked only by a manual run | |
| Unit-only guard through `execCmd` | Does not exercise `main.go`'s exit path | |

**Choice:** [auto] Rewrite the integration test (recommended default).
**Notes:** `test/integration/renamed_stubs_test.go` is missing from the roadmap's deletion list, but it would go RED on removal. Found during the codebase scout.

## Commit shape

| Option | Description | Selected |
|--------|-------------|----------|
| `test(03-NN)` RED commit, then `feat(cli)!:` with a `BREAKING CHANGE:` footer (recommended) | The `!` feeds the capability's Breaking-kind rule | ✓ |
| Plain `feat` / `refactor` | Loses the Breaking signal | |

**Choice:** [auto] RED commit, then a `feat(cli)!` commit (recommended default).

## Census and the fragment

| Option | Description | Selected |
|--------|-------------|----------|
| Census run in the plan before the close, instrument unchanged. A later hit only in the capability's fragment is expected. (recommended) | Keeps the v0.14.0 "reused verbatim" contract | ✓ |
| Edit the instrument to exclude `.changes/` | Breaks the verbatim-reuse contract | |

**Choice:** [auto] Census in the plan, instrument unchanged (recommended default).
**Notes:** Running the census on the current tree gave exactly four hits (`renamed.go:10-11` and the allowlist lines `:8-9`). New text must not spell the two-token phrase.

## Capability evidence location

| Option | Description | Selected |
|--------|-------------|----------|
| The verifier records the fragment, SHA and trailers in VERIFICATION.md (recommended) | `execute-phase` dispatches `verify:post` in `aggregate_results`, before `verify_phase_goal` | ✓ |
| A separate addendum file after the close | Adds a new structure. SC3 names VERIFICATION or SUMMARY. | |

**Choice:** [auto] The verifier records the evidence in VERIFICATION.md (recommended default).

## Inaccurate fragment handling

| Option | Description | Selected |
|--------|-------------|----------|
| Correct it in a separate commit and record both SHAs. CAP-05 is still met. (recommended) | No amend or rewrite of the capability's commit | ✓ |
| Leave it inaccurate | A wrong changelog line ships | |

**Choice:** [auto] Correct it in a separate commit (recommended default).

## Claude's Discretion

- Number of plans, the guard's file and function names, the wording of the `root.go` comments, and whether to leave the empty 999.5 backlog directory in place (default: leave it).

## Deferred Ideas

- A `SuggestFor` hint for the removed verbs belongs in a future CLI-ergonomics phase.
- Capability judgment defects go into a later capability release.
- Three keyword-only todo matches were reviewed and not folded: bench `pinnedAt`, the GH #85 reply, and the Go toolchain pin.
