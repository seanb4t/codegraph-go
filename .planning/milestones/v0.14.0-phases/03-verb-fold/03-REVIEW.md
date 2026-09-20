---
phase: 03-verb-fold
reviewed: 2026-09-16T23:45:00Z
depth: deep
files_reviewed: 15
files_reviewed_list:
  - internal/cli/renamed.go
  - internal/cli/renamed_test.go
  - test/integration/renamed_stubs_test.go
  - cmd/codegraph/main.go
  - internal/cli/search.go
  - internal/cli/daemon.go
  - internal/cli/index.go
  - internal/cli/root.go
  - internal/daemon/lock.go
  - internal/cli/query_cli_test.go
  - internal/cli/daemon_test.go
  - internal/cli/index_lock_test.go
  - internal/cli/notice_test.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - docs/CLI-REFERENCE.md
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-09-16T23:45:00Z
**Depth:** deep
**Files Reviewed:** 15
**Status:** issues_found

## Summary

Iteration 2 (`--auto` re-review) after commit `fa81672c` applied WR-01 from the prior review. Scope: re-verify WR-01's fix against the real compiled binary, review the two new/changed test files the fix introduced, and re-scan the rest of the Phase 3 file set for anything the first pass missed.

**WR-01 verification (independent, from scratch):** built `./cmd/codegraph` fresh and ran the real binary directly (not `execCmd`/`root.Execute()`).

- `query main`: stdout 0 bytes; stderr is exactly two lines (`"query" has been renamed to "search --full" — run: codegraph search --full <term>` / `the "query" stub is removed in the next minor release (v0.15.0)`), 148 bytes, confirmed byte-for-byte via `wc -c`/`cat -v` (no third duplicated line); exit code 1.
- `unlock /tmp`: same pattern, 150 bytes, two lines, exit code 1.
- Repeated with `--json`/extra args on both stubs (flag parsing is disabled, so this is a no-op by design) — identical two-line output, exit 1, in every case.

This matches `renamed.go`'s current design exactly: both `RunE` bodies now return a single `fmt.Errorf` whose `Error()` text is the entire two-line D-06 message (no `fmt.Fprintln` calls left in the stub), so `cmd/codegraph/main.go`'s one `fmt.Fprintln(os.Stderr, err)` is the only place any of it is printed. WR-01 is resolved; the regression it described (a duplicated, partially-redundant third line) no longer reproduces.

**New/changed test review:**

- `internal/cli/renamed_test.go`: `TestQueryStub`/`TestUnlockStub` now assert `err.Error()` equals the full two-line text exactly (was a substring match) and `errOut` (direct `cmd.ErrOrStderr()` capture) is exactly `""`. This is a strictly stronger assertion than before and correctly encodes the new contract — it would fail immediately if a future edit reintroduced a direct `Fprintln` in either stub. The now-unused `strings` import was correctly dropped. No gaps found.
- `test/integration/renamed_stubs_test.go` (new): `TestRenamedStubsPrintExactlyOnce` spawns the real built binary (via the package's existing `runBinary`/`copyFixture`/`binPath` harness, the same pattern every other file in `test/integration` uses — `binPath` is built once in `TestMain`, `runBinary` sets `cmd.Dir` and captures stdout/stderr into separate buffers) across four cases (`query` bare/`--json`, `unlock` bare/nonexistent-path-arg) and asserts stdout is empty, stderr equals the expected two-line text with exactly one trailing newline, and `exitErr.ExitCode() == 1` via a type-asserted `*exec.ExitError`. This is exactly the missing assertion WR-01 called for (an end-to-end check of `cmd/codegraph/main.go`'s print path, which `execCmd`-based unit tests structurally can't reach). No timing dependencies, no shared mutable state across subtests, no reliance on file ordering or flakiness-prone constructs — each subtest builds its own fixture copy and runs the binary once. Ran the suite twice back-to-back locally with `-count=1`; both green.

**Re-check of the rest of the scope:** re-read `search.go`, `daemon.go`, `index.go`, `root.go`, `internal/daemon/lock.go`, and all listed test files in full. Diffed `index.go`/`internal/daemon/lock.go` against the diff base independently of the prior review — both changes are exactly the two D-08 message-text updates (`codegraph unlock` → `codegraph daemon unlock`) the prior review already described; no other changes and no new defects. Re-ran the full verification chain independently rather than trusting the fix report's claims: `go build ./...`, `go vet` on the scoped packages, `gofmt -l` (clean), the full `internal/cli` test tree (`go test ./internal/cli/... -count=1`, all green including `TestEveryRegisteredFlagIsAccountedFor`: 37 commands walked, 113 flags accounted for), the new integration test (`-run TestRenamedStub -v`, all 4 subtests green), and `task docs:cli:drift` (byte-identical, no hand-edit drift). Found nothing beyond what iteration 1 already reported.

No new Critical or Warning findings. IN-01 from iteration 1 is carried forward unchanged below — it was explicitly out of this fix's scope and nothing in this diff touches `search.go`'s `Long` field.

## Info

### IN-01: `search` has no `Long` description explaining `--full`'s two-line/JSON-envelope behavior

**File:** `internal/cli/search.go:67-70`

**Issue:** `newSearchCmd` sets only `Short` (`"Lexically search symbol names/qualified names"`) and no `Long` field. The old `query` command's replacement behavior — `--full`'s two-line human render, the `query.MarshalQueryJSON` envelope shape it selects under `--json --full` versus the plain `Location` array for default `--json` — is documented only in code comments and this phase's `03-CONTEXT.md`, not in anything a user sees via `codegraph search --help` or `docs/CLI-REFERENCE.md`. Comparable commands in the tree (`daemon`, `ui`) carry a multi-line `Long` describing exactly this kind of mode-dependent behavior. This is not a defect in the generated doc (which correctly reflects the empty `Long`) — it's a gap in the generator's input. Still present, unchanged, in the current tree; out of scope for the WR-01 fix by the orchestrator's explicit `critical_warning` fix-scope decision (see `03-REVIEW-FIX.md`).

**Fix:** Consider adding a short `Long` to `newSearchCmd` describing the `--full` flag's two-line human output and its distinct `--json` envelope shape, so `docs/CLI-REFERENCE.md` and `--help` carry the same level of detail the old `query` verb's doc comment did.

---

_Reviewed: 2026-09-16T23:45:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
