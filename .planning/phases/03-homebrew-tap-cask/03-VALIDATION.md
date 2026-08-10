---
phase: 3
slug: homebrew-tap-cask
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-09
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

> **Read this first.** This phase's outputs are overwhelmingly **configuration** (`.goreleaser.yaml`'s `homebrew_casks:` block, generated Ruby, workflow YAML) plus one small Go surface (`codegraph man`). Its central behaviors — a cold `brew install`, completions and man pages appearing, a broken binary failing the install — **cannot be proven by a fast local test**. Sampling a config file with a unit test is precisely the shape this repo keeps finding cannot fire. The map below therefore states, per requirement, *which layer actually owns the proof* and names the legs that have **no** automated command, so the planner sequences them into the right wave rather than pretending a shape test covers them.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` — shape tests in `internal/upgrade` mirroring `goreleaser_shape_test.go` / `taskfile_shape_test.go` / `release_workflow_shape_test.go`. **No Ruby test framework is introduced**; Homebrew's own `brew audit` / `brew style` are external tools invoked through Taskfile targets, not a project-owned suite. |
| **Config file** | `.goreleaser.yaml` (new `homebrew_casks:` block). No new Go test config — new tests belong in the existing packages per this repo's shape-test convention. |
| **Quick run command** | `go test ./internal/upgrade/... -run TestHomebrewCask && go test ./internal/cli/... -run TestManCmd` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~10 seconds quick; full suite per existing baseline |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/upgrade/... -run TestHomebrewCask && go test ./internal/cli/... -run TestManCmd`, plus `task check:goreleaser` (schema-level config validation — cheap, catches YAML/field-name typos in the new block; it does **not** semantically validate hook bodies and runs no pipe)
- **After every plan wave:** `task test`
- **Before `/gsd-verify-work`:** Full suite green
- **Max feedback latency:** ~15 seconds for the automatable legs

**Phase-gate caveat (D-13):** this phase's gate cannot close on the full suite alone. D-13 requires **two real releases inside the phase**, with criterion 1 re-verified cold against the *second*, GoReleaser-regenerated cask. No local command shortens that.

---

## Per-Task Verification Map

Task IDs are assigned at planning time; this table is seeded by requirement and is filled in per task by the planner.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 0 | BREW-02 | — | `homebrew_casks:` selects only the `zip` archive id | unit (shape) | `go test ./internal/upgrade/... -run TestHomebrewCaskShape` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | BREW-03 | — | `generate_completions_from_executable` targets exactly `[bash, zsh, fish]` with `shell_parameter_format: cobra` | unit (shape) | `go test ./internal/upgrade/... -run TestHomebrewCaskCompletions` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | BREW-04 | — | `newManCmd()` is `Hidden: true`, registered on root, takes one positional arg | unit | `go test ./internal/cli/... -run TestManCmd` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | BREW-04 | — | `codegraph man <dir>` emits the full command tree | unit/integration | `go test ./internal/cli/... -run TestManCmdEmitsTree` | ❌ W0 | ⬜ pending |
| TBD | TBD | 1+ | BREW-05 (amended) | T-03-xx | Post-install hook asserts man output non-empty AND version matches cask | manual, one-time recorded RED (D-12) | none — deliberately not a permanent re-fire | ❌ | ⬜ pending |
| TBD | TBD | 1+ | BREW-02 | T-03-xx | Tap credential refused a write to `seanb4t/codegraph-go` | manual, one-time recorded (D-17) | none — explicitly not a permanent CI assertion | ❌ | ⬜ pending |
| TBD | TBD | last | BREW-01 | — | Cold `brew tap` + `brew install` + a real command | manual, real release (D-13) | none possible locally | ❌ | ⬜ pending |
| TBD | TBD | last | BREW-06 (integrity half) | — | Release assets, cosign bundles, SBOMs, SLSA provenance re-verified against re-downloaded artifacts | automated, existing | `task verify:release-assets` | ✅ exists | ⬜ pending |
| TBD | TBD | — | BREW-06 (mechanism half) | — | A failed tap push cannot corrupt the release | **structural argument only (D-18R)** | **none — no executed evidence by maintainer decision** | ❌ | ⬜ accepted-limitation |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Shape test(s) in `internal/upgrade` for the `homebrew_casks:` block, following `TestRawArchiveEntryStaysBinaryFormat` / `TestChecksumCoversRawAndZipIdsOnly`:
  - assert `ids` is exactly the set `[zip]` — **not** `raw`, **not** both (a both-match throws GoReleaser's own `ErrMultipleArchivesSameOS`)
  - assert `generate_completions_from_executable.shells` is exactly `[bash, zsh, fish]` — **not** Homebrew's 4-shell default that includes `pwsh`
  - distinguish *absent* from *present-but-wrong* for `url.template` — they are different failure modes
- [ ] Go unit test for `newManCmd()` — `Hidden: true`, registered on root, single positional arg
- [ ] Framework install: **none**. `go test` and Taskfile are already fully wired.

**Assert properties, never literal template strings.** This repo has a recorded case where a shape test *pinned a broken template* demonstrated-RED and thereby **resisted correction** — a test pinning a broken invariant is worse than no test. Where a template is involved, assert the resolved *property* (e.g. "resolves to N distinct names") and read `dist/artifacts.json` `name` fields, never filesystem basenames — basenames were the blind oracle that hid the Path-vs-Name collision through two review cycles.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cold `brew tap seanb4t/tap && brew install codegraph` succeeds and runs a real command | BREW-01, BREW-02 | Requires a published release and a real Homebrew install; no local test can stand in | On a machine with no prior codegraph: `brew tap seanb4t/tap && brew install codegraph && codegraph version`. Per D-13, repeat against the **second** release's regenerated cask. |
| Completions work in bash, zsh **and** fish after install | BREW-03 | `generate_completions_from_executable` runs at install time inside Homebrew | In each shell, confirm subcommand completion for `codegraph <TAB>` |
| `man codegraph` renders | BREW-04 | Depends on where the postflight hook writes and on `MANPATH` resolution for the active brew prefix | `man codegraph`; confirm on `/opt/homebrew` and note the `/usr/local` case |
| A deliberately broken binary fails `brew install` | BREW-05 (amended) | The gate is Ruby inside a generated cask, executed by Homebrew — not reachable from Go tests | One-time recorded RED mutation (D-12). Capture evidence durably in an `-EVIDENCE.md` file, following the `02-EVIDENCE.md` precedent — not merely asserted in checkpoint text. |
| Tap credential refused a write to `seanb4t/codegraph-go` | BREW-02 (criterion 5) | One-time by decision (D-17); deliberately not a standing CI assertion | Attempt a benign write with the App installation token; record status and response verbatim |

---

## Named Coverage Gaps (accepted, not oversights)

These are **decisions**, recorded so the verifier does not manufacture a substitute that passes vacuously.

1. **BREW-06 mechanism half has no executed evidence at all (D-18R).** `--snapshot` sets `skips.Publish` and `cask.Pipe{}` lives inside the Publish pipeline, so a local dry-run structurally cannot reach a tap-push failure. The maintainer accepted a **structural argument only**, resting on `internal/pipe/publish/publish.go`'s ordering (`cask.Pipe{}` after `release.Pipe{}`, under the source comment *"brew et al use the release URL, so, they should be last"*). This leaves a check that has never been observed to fire — the failure mode this repo names as its recurring one. **Report it as an accepted limitation, never as demonstrated.**
   *(Supersedes RESEARCH.md's BREW-06 row, which still describes a scratch-repo mechanism.)*
2. **D-12's RED proof does not re-fire.** One-time recorded mutation by decision. D-11's two positive assertions inside the generated hook are therefore the *only* permanent protection against the gate becoming a no-op — which is exactly what rule `84d1gfpywd` demands of them.
3. **D-17's negative proof does not re-fire.** If the App's installation scope is later widened, nothing notices.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies, **or** appear in Manual-Only / Named Coverage Gaps above
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s for automatable legs
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
