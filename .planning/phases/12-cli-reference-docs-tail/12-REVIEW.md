---
phase: 12-cli-reference-docs-tail
reviewed: 2026-09-13T00:00:00Z
depth: deep
files_reviewed: 8
files_reviewed_list:
  - .github/workflows/ci.yml
  - README.md
  - Taskfile.yml
  - docs/RELEASE.md
  - internal/cli/cli_reference_test.go
  - internal/cli/root.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - tools/clidoc/main.go
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-09-13T00:00:00Z
**Depth:** deep (cross-file trace: `tools/clidoc` ↔ `internal/cli/root.go` ↔ `internal/cli/cli_reference_test.go` ↔ `Taskfile.yml` ↔ `ci.yml`, plus a direct read of `cobra/doc@v1.10.1`'s `GenMarkdownCustom`/`GenMarkdownTreeCustom` to check the generator's and guard's predicates against the library they wrap)
**Files Reviewed:** 8 (phase diff `d762d575..HEAD`, minus planning artefacts and the generated `docs/CLI-REFERENCE.md`)
**Status:** issues_found (one Warning, no Critical)

## Summary

This is a small, disciplined docs-tail phase and it reads that way: the generator (`tools/clidoc/main.go`) and the accounting guard (`internal/cli/cli_reference_test.go`) build the identical command tree (`NewRootCmd()` + `InitDefaultCompletionCmd()` + `InitDefaultVersionFlag()`), the guard's `documentedByReference`/`ineligibleReason` predicates match `cobra/doc`'s own `!IsAvailableCommand() || IsAdditionalHelpTopicCommand()` gate verbatim (checked directly against `cobra@v1.10.1/doc/md_docs.go`), and the drift gate (`task docs:cli:drift`) regenerates only into `mktemp -d`, never in place, reporting its compared-file count before comparing and failing closed on a missing/short enumeration. I ran the live guard test and the live drift task rather than trusting the SUMMARY/SECURITY narrative: both pass, matching the counts claimed (36 commands, 115 flags, 114 via the reference + 1 via the allowlist; `docs:cli:drift` reports "compared 1 generated file" and byte-identical). `go build ./...`, `go vet`, and `gofmt -l` are all clean on the three new/changed Go files. The brew-trust wording (DOCS-07) is consistent across `README.md` and `docs/RELEASE.md`: no tap-wide command is spelled out anywhere, the narrowed error quote is used consistently in both the inline block and the retrospective blockquote, and the README's new `docs/CLI-REFERENCE.md` link resolves to a real, committed file.

I traced a substring-collision risk in the guard's whole-document `docMentionsFlag` match against the actual 34 distinct flag names currently registered (`--all` vs `--all-kinds`, `--editor-url` vs `--no-editor-url`, `--path` vs `--pattern`, etc.) — the anchoring is sound and none currently collide; this matches 12-SECURITY.md's own honest disclosure of the same edge case as an accepted, unclassified residual risk, so it is not re-reported here as a new finding. One genuine gap not previously called out: the allowlist's command-level (flagless) entry format is not restricted to commands that are actually hidden, which weakens D-08's "one rule, no exemption list" design intent for future maintainers (see WR-01). The remaining items are Info-level robustness/documentation nits.

## Warnings

### WR-01: Command-level allowlist entries aren't restricted to hidden commands, undermining "no exemption list"

**File:** `internal/cli/cli_reference_test.go:195-207` (the accounting `switch`), read together with `internal/cli/testdata/cli-reference-allowlist.txt`'s documented format and `12-CONTEXT.md` D-08 ("a whole hidden command may be listed once by path to cover all its flags").

**Issue:** D-08's stated design is "One rule, no exemption list" — a flag is accounted for either by appearing in the generated reference, or by an allowlist entry that carries a specific reason. The one command-level shorthand D-08 describes (`<full command path>` with no `--flag` suffix) is explicitly framed as covering "a whole hidden command." The implementation does not enforce that framing: the `switch` at line 195 falls back to `hasAllowEntry(allow, allowKeyCmd)` (the command-path-only key) for *any* ineligible flag, regardless of whether the command itself is hidden. Today the only command-level entry (`codegraph man`) legitimately targets a hidden command, so the gap is dormant — but nothing in the code stops a future maintainer from adding a single command-path entry against a **documented, visible** command (e.g. `codegraph query\treason`) to pre-emptively cover any hidden or deprecated flag ever added to it later, without a per-flag reason and without the guard ever re-flagging that flag for review. That is precisely the "gate that can pass vacuously" shape this phase otherwise takes pains to close (T-12-02/T-12-03 in 12-SECURITY.md) — it is just not yet exercised.

**Fix:** Require `!documentedByReference(cmd)` before honoring a command-level (flagless) allowlist entry, forcing per-flag entries (`<path> --<flag>`) for any ineligible flag on a command that is otherwise publicly documented:

```go
case hasAllowEntry(allow, allowKeyCmd) && !documentedByReference(cmd):
    used[allowKeyCmd] = true
    viaAllowlist++
```

This keeps the one legitimate use (`codegraph man`) working unchanged while closing the loophole for documented commands.

## Info

### IN-01: `tools/clidoc/main.go` writes the committed reference non-atomically

**File:** `tools/clidoc/main.go:50-53`
**Issue:** `os.WriteFile(*out, buf.Bytes(), 0o644)` opens with `O_TRUNC` and writes in place; a failure partway through (disk full, permission revoked mid-write, process killed) can leave a truncated `docs/CLI-REFERENCE.md` when `task docs:cli` is run in place. This is bounded in practice — `task docs:cli:drift` (CI-wired) would immediately flag any such truncation as drift on the very next run, and `tools/graphcluster/main.go:297` uses the identical non-atomic `os.WriteFile` pattern already, so this is consistent with established project precedent rather than a new regression. Noted for completeness per the review's specific question, not because it is exploitable today.
**Fix:** Optional hardening: write to `*out + ".tmp"` and `os.Rename` over the destination, matching the temp-then-atomic-swap shape the Taskfile-level drift gates already use one level up.

### IN-02: `-out` path is accepted without validation

**File:** `tools/clidoc/main.go:32`
**Issue:** The `-out` flag value is passed straight to `os.WriteFile` with no cleaning or path-traversal check. This is a local, non-network-facing dev tool invoked only by `Taskfile.yml` with a fixed literal path (`docs/CLI-REFERENCE.md` or a `mktemp -d` scratch path), so there is no realistic attacker-controlled input here; flagged only because the review's scope explicitly asked whether the path is validated.
**Fix:** None required; if this tool ever gains an untrusted caller, validate `-out` resolves under the intended output directory first.

### IN-03: Generator's own exit-code doc comment is incomplete

**File:** `tools/clidoc/main.go:15`
**Issue:** The header comment states "Exit codes: 0 = wrote -out successfully, 1 = build or write error," but an invalid flag invocation (e.g. `-bogus`) exits via the standard library's default `flag.ExitOnError` behavior at exit code 2, not 1. Every failure path still exits non-zero, so `task docs:cli`/`docs:cli:drift`'s `set -e` correctly aborts either way — this is a documentation-accuracy nit, not a functional defect.
**Fix:** Either broaden the comment to "any non-zero exit is a failure" or note the flag-parsing exit code explicitly.

---

_Reviewed: 2026-09-13T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
