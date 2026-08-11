---
phase: 4
slug: codegraph-upgrade-homebrew
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-11
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — `Taskfile.yml` defines every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it) |
| **Quick run command** | `go test ./internal/upgrade/...` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~5s package / ~90s full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/...`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | UPGR-01 | T-04-01, T-04-03 | A path-shape match alone never fires; an unresolvable path falls open to "not detected" rather than blocking the upgrade | unit (tracer, seam-based) | `go test ./internal/upgrade/ -run '^TestUpgradeRun_RefusesBrewManagedCask$' -v \| rg -c -e '--- PASS: TestUpgradeRun_RefusesBrewManagedCask'` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | UPGR-02 | T-04-01 | Four executing rows prove a Caskroom/Cellar-shaped path with no Homebrew receipt above it does not fire | unit (table-driven) | `test "$(go test ./internal/upgrade/ -run '^TestDetectBrewManaged$' -v 2>&1 \| rg -c -e '--- PASS: TestDetectBrewManaged/')" -ge 14` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | UPGR-03 | T-04-04 | `--check` mutates nothing and reaches no network; the reinstall-anyway option cannot override the refusal | unit (seam-based) | `test "$(go test ./internal/upgrade/ -run '^TestUpgradeRun_(CheckBrewManagedStepsAside\|ForceDoesNotOverrideBrewRefusal)$' -v 2>&1 \| rg -c -e '^--- PASS: TestUpgradeRun_')" -eq 2` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | UPGR-02 | T-04-05, T-04-06 | The install gate can no longer be satisfied by residue from a prior failed install | config (shape + validate) | `test "$(rg -c -e 'codegraph-brew-install' .goreleaser.yaml \|\| echo 0)" -eq 0 && test "$(rg -c -e 'fresh_man_pages' .goreleaser.yaml)" -ge 2 && task check:goreleaser` | ✅ | ⬜ pending |
| 04-02-02 | 02 | 1 | UPGR-02 | T-04-07 | Removing a CI gate cannot silently remove an unrelated one; the rehearsal independently re-checks freshness | config (shape) | `test "$(rg -c -e 'SENTINEL' Taskfile.yml \|\| echo 0)" -eq 0 && test "$(rg -c -e 'FRESH_MAN_PAGE_COUNT' Taskfile.yml)" -ge 3` | ✅ | ⬜ pending |
| 04-02-03 | 02 | 1 | UPGR-02 | T-04-08 | The evidence record states what was measured, with checkable citations | docs (grep set-check) | `test "$(rg -c -e 'leave the sentinel behind' .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md \|\| echo 0)" -eq 0 && test "$(rg -c -e '30 orphaned' .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md)" -ge 1` | ✅ | ⬜ pending |
| 04-03-01 | 03 | 1 | UPGR-01, UPGR-02, UPGR-03 | T-04-09, T-04-10 | The active-milestone scope still parses after the edits; no phase is dropped | docs (paired set-check + parser re-proof) | `test "$(rg --no-heading -n -e 'Cellar' .planning/ROADMAP.md \| rg -v -e 'Caskroom' \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg -n -e '^- \[.\] \*\*Phase 4:' .planning/ROADMAP.md \| rg -c -e 'Caskroom')" -eq 1` | ✅ | ⬜ pending |
| 04-03-02 | 03 | 1 | UPGR-01, UPGR-02 | T-04-11 | The falsified premise is cited so a future reader cannot re-derive it | docs (paired set-check) | `test "$(rg --no-heading -n -e 'Cellar' .planning/REQUIREMENTS.md .planning/PROJECT.md \| rg -v -e 'Caskroom' \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg -c -e '#19121' .planning/REQUIREMENTS.md)" -ge 1` | ✅ | ⬜ pending |
| 04-04-01 | 04 | 2 | UPGR-01, UPGR-03 | T-04-13, T-04-14 | Help text names both exit behaviours and offers no override | unit | `go test ./internal/cli/ -run '^TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes$' -v \| rg -c -e '--- PASS: TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes'` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | UPGR-01, UPGR-03 | T-04-12 | No published instruction contradicts the shipped mutation path | docs (paired grep) | `test "$(rg --no-heading -c -e "next phase's work" -e 'not yet shipped' -e 'undefined interaction' README.md docs/RELEASE.md \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg --no-heading -c -e 'brew upgrade codegraph' README.md docs/RELEASE.md \| wc -l \| tr -d ' ')" -eq 2` | ✅ | ⬜ pending |
| 04-05-01 | 05 | 3 | UPGR-01 | T-04-15, T-04-17 | A network-fetched binary is signature-verified before it is made executable | config (region-scoped ordering assertion) | `node -e 'const fs=require("fs");const t=fs.readFileSync("Taskfile.yml","utf8");const b=t.slice(t.indexOf("verify:self-upgrade:"),t.indexOf("verify:gatekeeper:"));const c=b.indexOf("cosign verify-blob"),m=b.indexOf("chmod +x");if(c<0\|\|m<0\|\|c>m)throw new Error("ordering");console.log("ok")'` | ✅ | ⬜ pending |
| 04-05-02 | 05 | 3 | UPGR-01 | T-04-16 | Every restatement of the release identity accepts and rejects exactly what the compiled policy does | unit (shape + vacuity self-test) | `test "$(go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy' -v 2>&1 \| rg -c -e '^--- PASS: TestCosignIdentityPolicy')" -eq 2` | ❌ W0 | ⬜ pending |
| 04-05-03 | 05 | 3 | UPGR-01 | T-04-16, T-04-18 | The drift guard is observed RED against a real semantic loosening and against an empty match set | unit (RED proof + revert) | `go test ./internal/upgrade/... && git diff --exit-code Taskfile.yml internal/upgrade/taskfile_shape_test.go` | ❌ W0 | ⬜ pending |
| 04-06-01 | 06 | 4 | UPGR-02 | T-04-19 | Detection fires against a genuinely Homebrew-authored tree, mutating nothing | acceptance (env-gated harness) | `CODEGRAPH_BREW_ACCEPTANCE_PATH="$(brew --prefix)/bin/codegraph" go test ./internal/upgrade/ -run '^TestDetectBrewManaged_RealInstall$' -v \| rg -c -e '--- PASS: TestDetectBrewManaged_RealInstall'` | ❌ W0 | ⬜ pending |
| 04-06-02 | 06 | 4 | UPGR-01, UPGR-03 | T-04-19 | The machine is provably restored to its recorded baseline on every path | acceptance (observed exit codes + restoration probes) | `test "$(brew list --cask --versions codegraph 2>/dev/null \| wc -l \| tr -d ' ')" -eq 0 -o -n "${CODEGRAPH_CASK_PREEXISTING:-}"` | ✅ | ⬜ pending |
| 04-06-03 | 06 | 4 | UPGR-01, UPGR-02, UPGR-03 | T-04-20, T-04-22 | The unproved leg is named, not claimed; every carried assumption is dispositioned | docs (structure assertion) | `test -s .planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md && test "$(rg -c -e 'UPGR-ACCEPTANCE-EVIDENCE' .planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md)" -eq 1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Sampling continuity:** no three consecutive tasks lack an `<automated>` verify — every one of
the 15 tasks above carries one. Tasks 04-06-02 and 04-06-03 additionally carry a `<human-check>`,
which is supplementary to their automated command, not a substitute for it
(`workflow.human_verify_mode` is `end-of-phase`, so no `checkpoint:human-verify` task exists in
this phase).

---

## Wave 0 Requirements

- [ ] `internal/upgrade/brew_test.go` — table-driven detection tests for UPGR-02, including the
      mandatory executing false-positive row (a non-brew binary under a path containing
      `Caskroom`/`Cellar` with no `INSTALL_RECEIPT.json` above it)
- [ ] Constructed-tree fixtures under `t.TempDir()` for `/opt/homebrew`, `/usr/local`, a custom
      prefix, and linuxbrew — **both Caskroom and Cellar shapes at the linuxbrew prefix**
      (D-04R, 2026-08-11)
- [ ] Seam assertion in `internal/upgrade/upgrade_test.go` proving `Options.download` and
      `Options.swap` are never invoked under a detected brew install (D-08)

*Existing `go test` infrastructure covers the framework itself — no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The refusal is observed against a **real** `brew tap seanb4t/tap` + `brew install codegraph` from the Phase-3 tap | UPGR-01 | The ROADMAP's Depends-on clause states a synthetic layout can only prove the mechanism, not that it matches what actually ships. Requires a real published cask and a real Homebrew install on the host. | `brew trust seanb4t/tap` (per `khb8v67c48` — the published contract does not work without it), `brew install codegraph`, then `codegraph upgrade` and `codegraph upgrade --check`; record both outputs and exit codes verbatim. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 15/15 tasks carry one
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — the six ❌ W0 rows are created by plans 04-01, 04-04, 04-05 and 04-06 themselves, each in the task that first needs them
- [x] No watch-mode flags
- [x] Feedback latency < 90s — the slowest automated command is `task test` (~90s); every per-task command is a filtered `go test` or a `rg`/`node` assertion under 10s
- [ ] `nyquist_compliant: true` set in frontmatter — set by validate-phase §6 after execution

**Approval:** plan-time map populated 2026-08-11 by `/gsd-plan-phase 4`; per-task Status column
updated during execution.
