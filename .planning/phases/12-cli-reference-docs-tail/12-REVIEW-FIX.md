---
phase: 12-cli-reference-docs-tail
fixed_at: 2026-09-13T21:30:17Z
review_path: .planning/phases/12-cli-reference-docs-tail/12-REVIEW.md
iteration: 1
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 12: Code Review Fix Report

**Fixed at:** 2026-09-13T21:30:17Z
**Source review:** .planning/phases/12-cli-reference-docs-tail/12-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (warning): 1 (WR-01)
- Fixed: 1
- Skipped: 0

IN-01, IN-02, and IN-03 are Info-severity and out of scope per the task's
explicit instruction; left untouched, matching the review's own "optional
hardening" / "none required" dispositions for all three.

## Fixed Issues

### WR-01: Command-level allowlist entries aren't restricted to hidden commands, undermining "no exemption list"

**Files modified:** `internal/cli/cli_reference_test.go`, `internal/cli/testdata/cli-reference-allowlist.txt`
**Commit:** `666569e9`
**Applied fix:** The accounting `switch` inside `TestEveryRegisteredFlagIsAccountedFor`'s `visit` closure now requires `!documentedByReference(cmd)` before honoring a command-level (flagless) allowlist entry — exactly the review's snippet — so a hidden/deprecated flag on a command that is otherwise publicly documented can no longer be silently absorbed by a pre-existing command-path entry; it must carry its own per-flag entry (`<path> --<flag>`). The one legitimate command-level use (`codegraph man`, a genuinely hidden command) is unaffected. The `default:` unaccounted-flag message was split: when `documentedByReference(cmd)` is true, it now names the required per-flag key explicitly (`add a per-flag entry %q with a reason`) instead of the generic "add an allowlist entry" wording, so a future maintainer hitting this failure is told exactly what to add. `parseCLIReferenceAllowlist`'s doc comment and the allowlist file's own header comment were both updated to state the rule precisely: a bare `<command path>` entry covers a HIDDEN command's flags only; a flag on a documented command always needs its own per-flag entry.

**RED→GREEN mutation-test transcript** (per the review's `Fix:` instructions, applied and reverted in the working tree, never committed):

1. Pre-fix state (test file and allowlist stashed back to their original, un-fixed content), with a throwaway hidden flag registered on the documented `ui` command (`cmd.Flags().Bool("zz-throwaway", false, ""); _ = cmd.Flags().MarkHidden("zz-throwaway")`) and a command-level allowlist line `codegraph ui<TAB>test` added — demonstrating the loophole, the guard **PASSES** vacuously:
   ```
   cli_reference_test.go:236: walked 36 commands (hidden included), inspected 116 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 2 accepted via testdata/cli-reference-allowlist.txt
   --- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
   ```
2. Fix restored, same `ui` mutation and same command-level allowlist line still in place — the guard now **FAILS**, naming the specific flag:
   ```
   cli_reference_test.go:254: walked 36 commands (hidden included), inspected 116 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
       cli_reference_test.go:274: 2 problem(s):
           unaccounted flag: codegraph ui --zz-throwaway (hidden flag — a command-level allowlist entry only covers a hidden command; add a per-flag entry "codegraph ui --zz-throwaway" with a reason)
           stale allowlist entry: codegraph ui (matches no registered command or flag)
   --- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
   ```
3. Allowlist line replaced with the per-flag form `codegraph ui --zz-throwaway<TAB>test` — the guard **PASSES** again:
   ```
   cli_reference_test.go:254: walked 36 commands (hidden included), inspected 116 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 2 accepted via testdata/cli-reference-allowlist.txt
   --- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
   ```
4. Both mutations reverted (`ui.go` via `git checkout --`, the allowlist line removed by hand). Confirmed byte-identical to the fixed pre-mutation state: `diff` against the pre-mutation copies of both files showed no differences, and `git diff --quiet -- internal/cli/ui.go` was clean. No `codegraph ui` process was ever started (the mutation only registered a flag; the command was never run).

**Real allowlist confirmed unaffected:** after reverting, the guard against the real, committed one-entry allowlist (`codegraph man`) still passes with the exact counts the review cited:
```
cli_reference_test.go:254: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
```

## Skipped Issues

None in scope — WR-01 was fixed. IN-01, IN-02, and IN-03 are Info-severity and outside `critical_warning` scope per the task's explicit instruction; left untouched.

## Verification

All verification below ran in the **main working tree** (no isolated worktree — per the task's stated `Isolation: none`); the numbers are reproducible directly from this checkout.

- `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestEveryRegisteredFlagIsAccountedFor -v ./internal/cli/...` — `PASS`, logging `walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt`, matching the gate's required counts exactly.
- `GOTOOLCHAIN=go1.26.6 go vet ./internal/cli/...` — clean.
- `GOTOOLCHAIN=go1.26.6 gofmt -l internal/cli/cli_reference_test.go` — empty output.
- `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` — `compared 1 generated file` / `docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)` — MATCH; the fix did not touch the generator or the generated doc.
- `git status --porcelain` — empty after the final commit below.
- No `codegraph ui` process was left running at any point; `docs/CLI-REFERENCE.md`, `tools/clidoc`, `Taskfile.yml`, and `.github/workflows/ci.yml` were never touched.

No source files were left in a broken state; no uncommitted source changes remain outside this report and the pre-existing, previously-untracked `12-REVIEW.md` (committed alongside this report).

---

_Fixed: 2026-09-13T21:30:17Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
