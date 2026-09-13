---
phase: "12"
slug: "cli-reference-docs-tail"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-13"
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `12-RESEARCH.md` § Validation Architecture; every command and file path
> below was verified against `Taskfile.yml`, `ci.yml`, and the tree during research.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib (`task test:unit` reaches `internal/cli`; only `internal/daemon` is excluded); Task drift gates (`task docs:cli:drift`, the `proto:drift` regenerate-into-temp + byte-compare shape) |
| **Config file** | `go.mod` (pins `go 1.26.6`; no new require — `github.com/spf13/cobra/doc` is already imported by `internal/cli/man.go`); `Taskfile.yml` (new `docs:cli`, `docs:cli:drift` targets) |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestEveryRegisteredFlagIsAccountedFor -v ./internal/cli/...` · `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit` + `GOTOOLCHAIN=go1.26.6 task test:golden` + `task proto:drift` + `task -s web:drift` + `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` |
| **Estimated runtime** | ~5 s guard · ~3 s drift · ~2–3 min full Go suite |

**Toolchain prefix (repo landmine).** Local Go is 1.27.1; `go.mod` pins 1.26.6 — prefix every `go build` / `go test` / `go run` / `go vet` with `GOTOOLCHAIN=go1.26.6`, including the `go run ./tools/clidoc` invocation inside the Taskfile targets.

**Byte-stability of the generated file.** `cmd.DisableAutoGenTag = true` on the root (propagated by `GenMarkdownCustom`'s own `VisitParents` walk) or the drift gate fails on the date footer every day; `root.InitDefaultCompletionCmd()` must be called by both the generator and the guard or the `completion` family (5 commands, 5 flags) is invisible to both (research finding 2 — 26 commands without it, 36 with).

**What is deliberately NOT tested.** DOCS-07 is a wording change verified by reading the diff (D-12, ROADMAP.md:121, criterion 3); nothing tests `cobra/doc`'s own behaviour; no docs-grep guard.

---

## Sampling Rate

- **After every task commit:** `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestEveryRegisteredFlagIsAccountedFor -v ./internal/cli/...` for any task touching `internal/cli`, the allowlist, or the generator; `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` for any task touching the generator, the Taskfile, or `docs/CLI-REFERENCE.md`
- **After every plan wave:** `GOTOOLCHAIN=go1.26.6 task test:unit` + `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift`
- **Before `/gsd-verify-work`:** full suite green, `GOTOOLCHAIN=go1.26.6 go vet ./...`, `task proto:drift` and `task -s web:drift` MATCH, `docs:cli:drift` green, the three RED families recorded in `12-MUTATION-LOG.md`, `git status --porcelain` empty
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map binds requirements to their automated commands so
the planner can attach them. Test names are illustrative until the planner fixes them.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | 01 | 1 | DOCS-05 | T-12 generated-file tampering | `docs/CLI-REFERENCE.md` is regenerated into a temp file by `tools/clidoc` and byte-compared against the committed file; the gate reports `compared 1 generated file` before comparing and names the file on mismatch; `DisableAutoGenTag` keeps it byte-stable across days | drift (Task) | `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` | ❌ Wave 0 (`tools/clidoc/main.go`, Taskfile targets) | ⬜ pending |
| TBD | 01 | 1 | DOCS-06 | T-12 silent hiding / allowlist rot | Walk of every command (hidden included, `InitDefaultCompletionCmd` + `InitDefaultHelpFlag` applied) over `Flags()`+`PersistentFlags()` with `VisitAll`; each flag either appears as `--name` in the generated reference (available command, not hidden/deprecated) or matches an allowlist entry with a reason; unmatched allowlist entries fail; counts asserted (commands ≥ 26, flags ≥ 50) and logged | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestEveryRegisteredFlagIsAccountedFor -v ./internal/cli/...` | ❌ Wave 0 (`internal/cli/cli_reference_test.go`, `internal/cli/testdata/cli-reference-allowlist.txt`) | ⬜ pending |
| TBD | 01 | 1 | DOCS-05 | — | `ci.yml` runs `task docs:cli:drift` in the same job and immediately after `task proto:drift` (`ci.yml:195-196`) | source assertion | `test "$(rg -o 'run: task docs:cli:drift' .github/workflows/ci.yml \| wc -l \| tr -d ' ')" = "1"` | ❌ Wave 0 | ⬜ pending |
| TBD | 02 | 2 | DOCS-07 | UF-2 docs-as-security-guidance | `docs/RELEASE.md` recommends `brew trust --cask seanb4t/tap/codegraph` with one framing sentence; the tap-wide grant is named as not recommended without its command spelled; the quoted Homebrew error is trimmed to the narrow form (README.md carries no `brew trust` text — it points at RELEASE.md; research finding 3) | manual: diff review (D-12) plus one-time source assertions in the plan | `test "$(rg -o 'brew trust --cask seanb4t/tap/codegraph' docs/RELEASE.md \| wc -l \| tr -d ' ')" -ge 1 && test "$(rg -o 'brew trust (--tap )?seanb4t/tap\b' docs/RELEASE.md \| wc -l \| tr -d ' ')" = "0"` | ✅ `docs/RELEASE.md` | ⬜ pending |
| TBD | 02 | 2 | DOCS-05 | — | README.md links to `docs/CLI-REFERENCE.md` once | source assertion | `test "$(rg -o 'docs/CLI-REFERENCE.md' README.md \| wc -l \| tr -d ' ')" = "1"` | ✅ `README.md` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/cli/root.go` — exported `NewRootCmd()` wrapper (3 test call sites use `newRootCmd()`; no rename needed) and `DisableAutoGenTag = true` on the root
- [ ] `tools/clidoc/main.go` — single-file `GenMarkdownCustom` walk in Cobra order with `InitDefaultCompletionCmd()`, banner line, `-out` flag
- [ ] `docs/CLI-REFERENCE.md` — first generated output, committed in the same commit as the generator and Taskfile targets
- [ ] `Taskfile.yml` — `docs:cli` (regenerate in place) and `docs:cli:drift` (temp + `cmp -s`, `compared 1 generated file` reported first); `.github/workflows/ci.yml` step after `task proto:drift`
- [ ] `internal/cli/cli_reference_test.go` + `internal/cli/testdata/cli-reference-allowlist.txt` (one entry: `codegraph man`, D-02 reason)
- [ ] `12-MUTATION-LOG.md` — families: (a) throwaway hidden flag on `ui` → guard RED; (b) one line deleted from the committed reference → drift RED; (c) bogus allowlist entry → guard RED on rot — each applied, RED captured, reverted byte-clean
- [ ] `12-SECURITY.md` — threat register (UF-2 framing, generated-file tampering, allowlist rot, exported constructor widens nothing)
- [ ] Framework install: none

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The brew-trust rewrite reads as intended: narrow grant first, one sentence of framing, no copy-pasteable tap-wide command | DOCS-07 | Wording quality; D-12 rules out a test | Read `git diff` for `docs/RELEASE.md` at the DOCS-07 commit |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
