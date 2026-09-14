---
quick_id: 260913-pkp
slug: fix-graphstore-archtest-to-fail-closed-on-per-package-load-errors
created: 2026-09-13
mode: quick
type: tdd
autonomous: true
files_modified:
  - internal/graphstore/archtest/import_graph_test.go
  - .planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md
  - .planning/STATE.md
files_deleted:
  - .planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md
estimate:
  tokens: 45000
  raw_tokens: 45000
  tasks: 2
  confidence: low
must_haves:
  truths:
    - "TestNoPackageBypassesGraphStore FAILS naming a non-zero package-error count when any package under the module load pattern has an unresolvable import — including one OUTSIDE internal/graphstore, where the positive control stays satisfied"
    - "At HEAD with a byte-clean tree the test PASSES (ok) under GOTOOLCHAIN=go1.26.6"
    - "The RED transcript (pre-fix vacuous PASS, post-fix FAIL with the count, byte-clean revert, GREEN re-run) is recorded verbatim in 260913-pkp-SUMMARY.md"
    - "The todo lives under .planning/todos/completed/ with completed/status stamped by `todo complete`, and STATE.md lists it under Resolved, not Pending Todos"
  artifacts:
    - "internal/graphstore/archtest/import_graph_test.go — packages.PrintErrors guard between the len(pkgs)==0 check and the importer scan"
    - ".planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md"
    - ".planning/STATE.md — Pending Todos row removed, Resolved row added"
    - ".planning/quick/260913-pkp-fix-graphstore-archtest-to-fail-closed-o/260913-pkp-SUMMARY.md — carries the transcript"
  key_links:
    - "The guard MUST sit after `len(pkgs) == 0` and before `foundGraphstoreImporter := false` — placed after the scan it would only refuse once the vacuous pass had already been computed"
    - "The RED plant MUST be in a package outside internal/graphstore (internal/query/traverse.go) — a plant inside graphstore would trip the existing positive control and prove nothing about the hole the todo names"
---

<objective>
Make `TestNoPackageBypassesGraphStore` (`internal/graphstore/archtest/import_graph_test.go`) fail closed when `go/packages` reports a per-package load error, mirroring the Phase 7 CR-01 fix in `internal/query/archtest/import_direction_test.go` (commit `a90b5457`). Today the test checks only `packages.Load`'s top-level `error` and `len(pkgs) == 0`; an unresolvable import in any package other than `internal/graphstore` leaves the positive control satisfied while the pebble-bypass scan silently skips the broken package — a guard that cannot fire.

Purpose: close the guard-vacuity hole filed as todo `2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md`, and prove the fixed guard RED against a confirmed-applied mutation per the standing rule (STATE.md: "A gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation"; rule `84d1gfpywd`).

Output: the guarded archtest, the todo moved to completed through `gsd_run todo complete`, the STATE.md row moved from Pending Todos to Resolved, and the RED/GREEN transcript in the SUMMARY.
</objective>

<execution_context>
@~/.claude/gsd-core/workflows/execute-plan.md
@~/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/STATE.md
@/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md
@/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md
@/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/archtest/import_graph_test.go

Working-tree note (verified at planning time, 2026-09-13): `.planning/debug/resolved/tty03-cold-start-poll-race.md` is ALREADY modified in the working tree before this task starts. It is not this task's file. Never stage it, never revert it, and scope every cleanliness gate below with an explicit pathspec so that pre-existing entry does not confuse the byte-clean checks.

Every `go` invocation in this plan is prefixed `GOTOOLCHAIN=go1.26.6` (verified: `go version go1.26.6 darwin/arm64`; the archtest passes at HEAD in ~0.3s).

The `gsd_run` launcher for Task 2: `gsd_run() { node /Users/sean/.claude/gsd-core/bin/gsd-tools.cjs "$@" 2>/dev/null; }`. The verb is `gsd_run todo complete <basename>` (basename only, resolved against `.planning/todos/pending/`); it supports `--dry-run` and on the real run moves the file to `.planning/todos/completed/` and upserts exactly two frontmatter fields: `completed: <today>` and `status: completed`. Do not add any other frontmatter field (planning-artifacts rule: fill values in tool-owned shapes, never invent structure).
</context>

<tasks>

<task type="tdd" tdd="true">
  <name>Task 1: Add the PrintErrors fail-closed guard and demonstrate it RED against a planted unresolvable import outside internal/graphstore</name>
  <files>internal/graphstore/archtest/import_graph_test.go</files>
  <read_first>
    - /Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/archtest/import_graph_test.go (whole file, 91 lines — the guard goes between line 45 `}` closing the `len(pkgs) == 0` check and line 47 `foundGraphstoreImporter := false`)
    - /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md ("Problem" and "Suggested fix" — the 3-line guard to add, verbatim)
    - `git show a90b5457 -- internal/query/archtest/import_direction_test.go` (the precedent: an 8-line `// packages.Load reports a per-package failure …` comment block ending in `(CR-01).` followed by the `if n := packages.PrintErrors(pkgs); n > 0 { t.Fatalf(...) }` guard, inserted immediately after the `len(pkgs) == 0` block)
    - /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md lines 115-140 ("Family (b) addendum — CR-01": the transcript shape to reproduce here — pre-mutation gate, mutation applied, FAIL output naming the count, revert, byte-clean proof; that run observed `reported 1 package error(s)` for the narrower `internal/query` load)
    - /Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/traverse.go lines 1-12 (the import block that receives the plant — the same file the precedent mutated)
  </read_first>
  <behavior>
    - Guard-contract RED (the hole, pre-fix): with an unresolvable blank import planted in `internal/query/traverse.go` (a package OUTSIDE `internal/graphstore`) and the test file UNCHANGED, `go test -run TestNoPackageBypassesGraphStore ./internal/graphstore/archtest/...` prints `ok` — the positive control is satisfied and the broken package is silently skipped. This is the defect, observed, not assumed.
    - Guard-contract GREEN (gate goes RED against the mutation, post-fix): with the SAME plant still in place and the guard added, the same command prints `--- FAIL: TestNoPackageBypassesGraphStore`, a `no required module provides package github.com/seanb4t/codegraph-go/internal/zz-does-not-exist` line from PrintErrors on stderr, and the Fatalf line `import_graph_test.go:<N>: packages.Load reported <n> package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see the errors above` with `<n>` a positive integer. Record `<n>` as observed; because this load is `Tests: true` over the whole module (not just `internal/query`), `<n>` may exceed the precedent's 1 — any positive value is the expected outcome.
    - Byte-clean revert then GREEN: `git checkout -- internal/query/traverse.go`, `git diff --quiet -- internal/query/traverse.go` exits 0, and the same command prints `ok` with the guard still present.
  </behavior>
  <action>
    Order is RED first (TDD mode); do not apply the fix before step 2 has been observed.

    1. Pre-mutation gate: run `git diff --quiet -- internal/query/traverse.go` and confirm exit 0. Then plant the mutation: insert the single line `_ "github.com/seanb4t/codegraph-go/internal/zz-does-not-exist"` as the first line inside the `import (` block of `internal/query/traverse.go` (1 insertion, nothing else touched; a nonexistent first-party path is used, as in the precedent, so no import cycle and no `go.mod` change can arise). Confirm with `git diff --stat -- internal/query/traverse.go` showing exactly 1 insertion in that one file.

    2. Observe the hole (guard-contract RED): run `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestNoPackageBypassesGraphStore ./internal/graphstore/archtest/...` with the test file still unmodified and capture the output verbatim. Expected: `ok  	github.com/seanb4t/codegraph-go/internal/graphstore/archtest`. If it instead FAILS here, stop and report — the premise in the todo would be wrong and the plan needs revisiting, not a workaround.

    3. Apply the fix to `internal/graphstore/archtest/import_graph_test.go`: immediately after the closing brace of the `if len(pkgs) == 0 { … }` block (line 45) and before the blank line preceding `foundGraphstoreImporter := false`, insert a tab-indented `//` comment block followed by the guard. The comment block mirrors the a90b5457 block, adapted to this test's context, and must contain the literal `(CR-01` once; write it as these lines, each prefixed by a tab and `// `:
       "packages.Load reports a per-package failure (an unresolvable import, a", "build error anywhere under the module pattern) in pkg.Errors, NOT in its", "top-level error return. A broken subtree is then absent from, or has an", "incomplete Imports map in, the returned graph, so the pebble-importer scan", "below looks for members that were never added. The positive control", "narrows but does not close that hole: a failure in a package other than", "internal/graphstore leaves the control satisfied while the bypass check", "silently skips the broken package. PrintErrors both surfaces each error", "on stderr and returns the count; a non-zero count means this test cannot", "verify anything about the affected package(s) and must refuse (CR-01", "sibling of the internal/query archtest fix, a90b5457)."
       Then the guard, exactly as in the todo's "Suggested fix" and the precedent: an `if n := packages.PrintErrors(pkgs); n > 0 {` line whose body is a single `t.Fatalf` with the format string `packages.Load reported %d package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see the errors above` and argument `n`, then the closing brace. Keep the Fatalf string byte-identical to the precedent's (same em dash, same wording) so the two sibling archtests read the same. No new imports are needed — `packages` is already imported. Run `GOTOOLCHAIN=go1.26.6 gofmt -l internal/graphstore/archtest/` and confirm it prints nothing.

    4. Demonstrate the gate RED against the still-applied mutation: re-run the same `go test` command and capture the output verbatim — it must FAIL with the Fatalf line naming a positive count `<n>`. Also capture the `no required module provides package …zz-does-not-exist` stderr line(s) PrintErrors emitted above it.

    5. Revert the plant byte-clean: `git checkout -- internal/query/traverse.go`, then `git diff --quiet -- internal/query/traverse.go` (exit 0) and `git status --porcelain -- internal/` which must print exactly one line, ` M internal/graphstore/archtest/import_graph_test.go`.

    6. GREEN: re-run the same `go test` command and capture `ok`. Also run the sibling once to confirm nothing regressed: `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/... ./internal/graphstore/archtest/...`.

    7. Keep every captured command + output from steps 1-6 for the SUMMARY under a heading "RED/GREEN transcript" in the same shape as 07-MUTATION-LOG.md family (b) addendum: pre-mutation gate, mutation applied, pre-fix PASS (the hole), fix applied, post-fix FAIL naming the count, revert, byte-clean proof, post-revert GREEN.

    8. Commit only `internal/graphstore/archtest/import_graph_test.go` with subject `fix(graphstore/archtest): refuse a partial go/packages load in TestNoPackageBypassesGraphStore (CR-01 sibling)` and a body that states: PrintErrors guard added after the zero-package check; demonstrated RED with an unresolvable blank import planted in internal/query/traverse.go (outside internal/graphstore, where the positive control stayed satisfied and the unfixed test passed), observed `reported <n> package error(s)`, reverted byte-clean; resolves the 2026-09-08 todo. Append the session's attribution trailers. Do not stage `.planning/debug/resolved/tty03-cold-start-poll-race.md`.
  </action>
  <acceptance_criteria>
    - `rg -o 'packages\.PrintErrors\(pkgs\)' internal/graphstore/archtest/import_graph_test.go | wc -l` prints 1
    - `rg -o 'packages\.Load reported %d package error\(s\)' internal/graphstore/archtest/import_graph_test.go | wc -l` prints 1
    - `rg -o 'CR-01' internal/graphstore/archtest/import_graph_test.go | wc -l` prints 1
    - `rg -n 'len\(pkgs\) == 0|packages\.PrintErrors\(pkgs\)|foundGraphstoreImporter := false' internal/graphstore/archtest/import_graph_test.go` prints exactly 3 lines, in that order, with strictly increasing line numbers (guard placement)
    - `GOTOOLCHAIN=go1.26.6 gofmt -l internal/graphstore/archtest/` prints nothing
    - `git diff --quiet -- internal/query/traverse.go` exits 0 after the revert
    - The transcript captured for the SUMMARY contains, verbatim, one `ok` line from the pre-fix run with the plant applied, one `--- FAIL: TestNoPackageBypassesGraphStore` block from the post-fix run containing `packages.Load reported ` followed by a positive integer and ` package error(s)`, and one `ok` line from the post-revert run
    - `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/... ./internal/graphstore/archtest/...` prints `ok` for both packages
  </acceptance_criteria>
  <verify>
    <automated>cd /Volumes/Code/github.com/seanb4t/codegraph-go && git diff --quiet -- internal/query/traverse.go && test "$(rg -o 'packages\.PrintErrors\(pkgs\)' internal/graphstore/archtest/import_graph_test.go | wc -l | tr -d ' ')" = 1 && test "$(rg -o 'packages\.Load reported %d package error\(s\)' internal/graphstore/archtest/import_graph_test.go | wc -l | tr -d ' ')" = 1 && test -z "$(GOTOOLCHAIN=go1.26.6 gofmt -l internal/graphstore/archtest/)" && GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestNoPackageBypassesGraphStore ./internal/graphstore/archtest/...</automated>
    <fails_when>the guard line is absent (count 0) or duplicated (count 2+); the Fatalf format string differs from the precedent's; gofmt reports the file; the plant was not reverted byte-clean (traverse.go dirty); or the archtest does not pass at HEAD with a clean tree. The RED half is proven by the step-4 FAIL output carried in the SUMMARY — if that transcript shows `ok` after the fix with the plant applied, the guard did not fire and the task is not done.</fails_when>
  </verify>
  <done>The guard is in place between the zero-package check and the importer scan; the transcript shows the unfixed test passing vacuously with the plant, the fixed test FAILING naming a positive package-error count with the same plant, a byte-clean revert (`git diff --quiet` exit 0), and a GREEN re-run; `internal/query/traverse.go` is untouched at HEAD; the fix is committed alone under the stated subject.</done>
</task>

<task type="auto">
  <name>Task 2: Resolve the todo through the tool and move its STATE.md row from Pending Todos to Resolved</name>
  <files>.planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md, .planning/STATE.md</files>
  <read_first>
    - `git show 910d5e61 --stat` and `git show 910d5e61 -- .planning/STATE.md` (the prior todo resolution: the tool moved the file and stamped exactly `completed:`/`status: completed`; STATE.md's Pending Todos row was deleted and one row appended to the Resolved table)
    - /Volumes/Code/github.com/seanb4t/codegraph-go/.planning/STATE.md lines 294-318 (the `### Pending Todos` table — row at line 302 begins `| 2026-09-08 | testing | — | \`internal/graphstore/archtest\` ignores per-package` — and the "Resolved and filed to `.planning/todos/completed/`" table whose last row is the 2026-09-13 brew-trust row)
  </read_first>
  <action>
    1. Preview then run the tool: `gsd_run todo complete 2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md --dry-run` (confirm `would_move` source/target and `would_set: {completed, status}`), then the same command without `--dry-run`. Confirm the file now exists under `.planning/todos/completed/`, no longer under `.planning/todos/pending/`, and its frontmatter gained exactly `completed: 2026-09-13` and `status: completed` (2 added lines in `git diff --stat` once staged as a rename). Do not hand-edit the todo file.

    2. Edit `.planning/STATE.md` with two scoped replacements (Edit tool, never a whole-file Write): (a) delete the Pending Todos row that begins `| 2026-09-08 | testing | — | \`internal/graphstore/archtest\` ignores per-package`; (b) append one row to the end of the Resolved table, directly after the `| 2026-09-13 | docs | \`brew trust\` …` row, in the table's existing three-column shape: `| 2026-09-13 | testing | \`internal/graphstore/archtest\` ignored per-package \`go/packages\` load errors — closed by quick task 260913-pkp (\`packages.PrintErrors\` guard in \`TestNoPackageBypassesGraphStore\`, demonstrated RED against an unresolvable import planted outside \`internal/graphstore\`; sibling of Phase 7 CR-01) |`. Leave the Pending Todos intro paragraph's "holds only 2 files" wording alone — it is a dated observation of a drift already flagged for a separate reconciliation pass, and this task does not perform that pass. No new headings, sections, or table columns.

    3. Commit the todo move and STATE.md together with subject `docs(todos): resolve the graphstore-archtest per-package load-error todo` and the session's attribution trailers. Pathspec the commit to `.planning/todos/` and `.planning/STATE.md` only; the pre-existing `.planning/debug/resolved/tty03-cold-start-poll-race.md` modification stays unstaged.
  </action>
  <acceptance_criteria>
    - `test ! -e .planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md` succeeds
    - `rg -o '^status: completed$' .planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md | wc -l` prints 1
    - `rg -o '^completed: 2026-09-13$' .planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md | wc -l` prints 1
    - `rg -n 'internal/graphstore/archtest' .planning/STATE.md` prints exactly one line, and that line begins with `| 2026-09-13 | testing |` (set-equality: the Pending row is gone AND the Resolved row is present — the count alone would also be 1 if the executor forgot the Resolved row but a stale mention survived elsewhere, so the prefix check is load-bearing)
    - `git status --porcelain -- internal/ .planning/todos/ .planning/STATE.md` is empty after the commit
  </acceptance_criteria>
  <verify>
    <automated>cd /Volumes/Code/github.com/seanb4t/codegraph-go && test ! -e .planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md && test "$(rg -o '^status: completed$' .planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md | wc -l | tr -d ' ')" = 1 && test "$(rg -n 'internal/graphstore/archtest' .planning/STATE.md | wc -l | tr -d ' ')" = 1 && rg -q '^\| 2026-09-13 \| testing \| `internal/graphstore/archtest` ignored' .planning/STATE.md && test -z "$(git status --porcelain -- internal/ .planning/todos/ .planning/STATE.md)"</automated>
    <fails_when>the todo still sits in pending/ (tool not run); the completed file lacks the tool's `status: completed` stamp (hand-moved instead of tool-moved); STATE.md still carries the Pending row, lacks the Resolved row, or carries both (count != 1 or prefix mismatch); or the commit left any of the intended paths dirty.</fails_when>
  </verify>
  <done>The todo file is under completed/ with the tool's two stamped fields and nothing else changed; STATE.md's Pending Todos table no longer lists the graphstore archtest and the Resolved table's last row records it as closed by 260913-pkp; both are committed together under the stated subject; the pre-existing debug-file modification is untouched.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| go/packages load → archtest assertion | The archtest trusts the loaded import graph to be complete; a per-package load error makes it silently incomplete. No new external input crosses any boundary in this task. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-260913-pkp-01 | Repudiation (guard vacuity) | `internal/graphstore/archtest.TestNoPackageBypassesGraphStore` | medium | mitigate | `packages.PrintErrors(pkgs)` non-zero count → `t.Fatalf` before the importer scan, so a partial load can no longer pass the D-04a pebble-bypass guard vacuously; demonstrated RED against a planted unresolvable import outside `internal/graphstore` (Task 1 steps 2-6). No package installs; `T-SC` not applicable. |
</threat_model>

<verification>
- Task 1's `<verify>` at HEAD with a clean tree: guard present exactly once, Fatalf string byte-identical to the precedent, gofmt clean, `internal/query/traverse.go` byte-clean, archtest `ok`.
- Task 1's RED half lives in the SUMMARY transcript: pre-fix `ok` with the plant, post-fix `--- FAIL` naming a positive count with the same plant, byte-clean revert, post-revert `ok`.
- Task 2's `<verify>`: todo moved by the tool with its two stamped fields; STATE.md carries exactly one `internal/graphstore/archtest` mention and it is the Resolved row; intended paths clean after commit.
- Both sibling archtests pass together: `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/... ./internal/graphstore/archtest/...`.
</verification>

<success_criteria>
- `TestNoPackageBypassesGraphStore` refuses on a non-zero `packages.PrintErrors` count, with the guard placed between the zero-package check and the importer scan, mirroring `a90b5457`.
- The guard has been demonstrated RED against a confirmed-applied mutation in a package outside `internal/graphstore` (the exact hole the todo names), and the mutation was reverted byte-clean before the commit.
- The todo is resolved through `gsd_run todo complete` (not a hand move) and STATE.md's Pending/Resolved tables reflect it, with no invented structure.
- Two commits: the code fix alone, then the docs move; the pre-existing `.planning/debug/resolved/tty03-cold-start-poll-race.md` modification is never staged.
</success_criteria>

<output>
Create `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/quick/260913-pkp-fix-graphstore-archtest-to-fail-closed-o/260913-pkp-SUMMARY.md` when done, carrying the full RED/GREEN transcript under a "RED/GREEN transcript" heading (pre-mutation gate, mutation applied, pre-fix PASS, fix applied, post-fix FAIL naming the count, revert, byte-clean proof, post-revert GREEN) in the shape of 07-MUTATION-LOG.md family (b) addendum.
</output>