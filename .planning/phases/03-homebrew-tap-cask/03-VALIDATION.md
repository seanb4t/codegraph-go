---
phase: 3
slug: homebrew-tap-cask
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-09
validated: 2026-08-10
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

**Phase-gate caveat (D-13, amended 2026-08-10):** this phase's gate cannot close on the full suite alone — a cold `brew install` against a published release is the only proof of criterion 1, and no local command shortens that.

> **Amended 2026-08-10 (validate-phase 03).** This caveat previously read "D-13 requires **two real releases inside the phase**, with criterion 1 re-verified cold against the *second*, GoReleaser-regenerated cask." The maintainer reduced the phase's scope to **one** release, and ROADMAP criterion 1 was amended to record why: a second release would have exercised GoReleaser's tap-push UPDATE path and `brew upgrade` — code this project does not own, cannot patch, and which surfaces on the next natural release regardless. See `03-EVIDENCE.md` "Scope reduction, recorded plainly". Corrected here rather than left standing because VALIDATION.md is a live contract future runs execute, per the ruling recorded for `02-VALIDATION.md`; completed `PLAN.md` files carrying the old wording are deliberately left alone as a record of plan-time intent.

---

## Per-Task Verification Map

Task IDs are assigned at planning time; this table is seeded by requirement and is filled in per task by the planner. **Statuses below were resolved 2026-08-10 by EXECUTING each declared command verbatim**, never by reading it — the method that caught `task lint:actionlint` (a target that has never existed) in phase 2's audit.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01 T2 | 03-01 | 1 | BREW-04, BREW-05 | T-03-01, T-03-03 | Tracer: a GoReleaser-rendered cask installs via real Homebrew and its hook executes the installed binary | integration (local rehearsal) | `task release:rehearse-cask` | ✅ target exists | ✅ green (recorded) |
| 03-01 T3 | 03-01 | 1 | BREW-04 | — | `newManCmd()` is `Hidden: true`, registered on root, one positional arg, creates its output dir, errors on an unwritable path, emits the full tree | unit | `go test ./internal/cli/... -run TestManCmd` | ✅ exists | ✅ green — 5/5 |
| 03-02 T3 | 03-02 | 2 | BREW-02 | — | `homebrew_casks:` selects only the `zip` archive id, and declares no `url` key | unit (shape) | `go test ./internal/upgrade/... -run TestHomebrewCask` | ✅ exists | ✅ green — 5/5 |
| 03-02 T3 | 03-02 | 2 | BREW-03 | T-03-06 | `generate_completions_from_executable` targets exactly `[bash, zsh, fish]` with `shell_parameter_format: cobra` | unit (shape) | `go test ./internal/upgrade/... -run TestHomebrewCask` | ✅ exists | ✅ green — 5/5 |
| 03-02 T1 | 03-02 | 2 | BREW-05 | T-03-04, T-03-05 | Post-install hook asserts man output non-empty AND binary version equals cask version; sentinel written; uninstall symmetric | integration (local rehearsal) + shape | `task release:rehearse-cask` | ✅ target exists | ✅ green (shape 5/5; rehearsal recorded) |
| 03-03 T3 | 03-03 | 2 | BREW-02 | T-03-07, T-03-09, T-03-10 | Tap secrets scoped to one job at step level; `id-token: write` still singly held; release halts on a missing or non-distinct tap credential | unit (shape) | `go test ./internal/upgrade/... -run TestHomebrewTap` | ✅ exists | ⚠️ **partial — 2/2 green, but this row is a 3-leg conjunction and only legs 1–2 are covered; see gaps G1/G2** |
| 03-04 T1 | 03-04 | 3 | BREW-05 (amended) | T-03-11 | A deliberately broken binary fails `brew install` non-zero and leaves nothing behind | manual, one-time recorded RED (D-12) | none — deliberately not a permanent re-fire | ❌ by decision | ✅ proven once (D-12) |
| 03-04 T2 | 03-04 | 3 | BREW-02 | T-03-12 | Tap credential wrote the tap (positive control) and was refused a write to `seanb4t/codegraph-go` | manual, one-time recorded (D-17) | none — explicitly not a permanent CI assertion | ❌ by decision | ✅ proven once (D-17) |
| 03-05 T2 | 03-05 | 4 | BREW-01 | T-03-14, T-03-15 | Cold `brew tap` + `brew install` + a real command, against the published release's cask (**amended 2026-08-10:** previously "the SECOND release's regenerated cask" — scope reduced to one release) | manual, real release (D-13, amended) | none possible locally | ❌ by decision | ✅ proven once vs v0.8.0 |
| 03-05 T3 | 03-05 | 4 | BREW-06 (integrity half) | — | Release assets, cosign bundles, SBOMs, provenance re-verified against re-downloaded artifacts; no duplicated or orphaned asset | automated, existing | `task verify:release-assets` | ✅ exists | ✅ green — **re-run this session vs v0.8.0, exit 0** |
| 03-05 T1 | 03-05 | 4 | BREW-06 (mechanism half) | T-03-13 | A failed tap push cannot corrupt the release | **structural argument only (D-18R)** | **none — no executed evidence by maintainer decision** | ❌ | ⬜ accepted-limitation |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky/partial*

### Commands executed this session (verbatim)

| Command | Result |
|---------|--------|
| `go test ./internal/cli/... -run TestManCmd` | exit 0 — **5 tests ran**, all PASS |
| `go test ./internal/upgrade/... -run TestHomebrewCask` | exit 0 — **5 tests ran**, all PASS |
| `go test ./internal/upgrade/... -run TestHomebrewTap` | exit 0 — **2 tests ran**, all PASS |
| `task check:goreleaser` | 1 configuration file validated |
| `task test` (full suite) | **exit 0, 59 packages ok, 0 FAIL** |
| `TAG=v0.8.0 REPO=seanb4t/codegraph-go task verify:release-assets` | **exit 0 — PASS**, checksums + cosign + attestation re-verified against re-downloaded published assets |

**Test counts were taken, not inferred.** `go test -run PATTERN` exits 0 when the pattern matches *nothing*, so a bare exit-0 is indistinguishable from a pattern that selected no tests — the same shape as this repo's `rg -h` incident. Each count above comes from `-v` and `--- PASS` lines.

**All four declared `task` targets were confirmed to exist** (`release:rehearse-cask`, `verify:release-assets`, `check:goreleaser`, `test`). No `lint:actionlint`-class phantom target in this phase.

**`task release:rehearse-cask` was deliberately NOT executed this session.** It installs and uninstalls a real cask in the maintainer's real Homebrew prefix — outside any sandbox, which is precisely why T-03-02 exists — and it requires 1Password-loaded Apple credentials. Its evidence is recorded from the phase's own runs (`03-01` D3, `03-02` D1–D5). Recorded as green-by-recorded-execution, not re-run.

**`verify:release-assets`'s three preconditions fired in order** (`TAG`, then `REPO`, then `GH_TOKEN`), each with an actionable message and exit 201, before any network work — a free positive signal that the guard chain is ordered correctly.

---

## Wave 0 Requirements

- [x] Shape test(s) in `internal/upgrade` for the `homebrew_casks:` block, following `TestRawArchiveEntryStaysBinaryFormat` / `TestChecksumCoversRawAndZipIdsOnly`:
  - [x] assert `ids` is exactly the set `[zip]` — `TestHomebrewCaskIDsIsExactlyZipArchiveSet`
  - [x] assert `generate_completions_from_executable.shells` is exactly `[bash, zsh, fish]` — `TestHomebrewCaskGeneratedCompletionsShellsIsExactSet`
  - [x] distinguish *absent* from *present-but-wrong* for `url.template` — `TestHomebrewCaskHasNoURLKey`
- [x] Go unit test for `newManCmd()` — 5 tests, covering `Hidden: true`, root registration, single positional arg, directory creation incl. missing parents, and an unwritable-path error naming the path
- [x] Framework install: **none**. `go test` and Taskfile are already fully wired.

**Wave 0 complete** — verified 2026-08-10 by executing both suites (5/5 and 5/5).

**Assert properties, never literal template strings.** This repo has a recorded case where a shape test *pinned a broken template* demonstrated-RED and thereby **resisted correction** — a test pinning a broken invariant is worse than no test. Where a template is involved, assert the resolved *property* (e.g. "resolves to N distinct names") and read `dist/artifacts.json` `name` fields, never filesystem basenames — basenames were the blind oracle that hid the Path-vs-Name collision through two review cycles.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cold `brew tap seanb4t/tap && brew install codegraph` succeeds and runs a real command | BREW-01, BREW-02 | Requires a published release and a real Homebrew install; no local test can stand in | On a machine with no prior codegraph: `brew tap seanb4t/tap && brew trust --cask seanb4t/tap/codegraph && brew install codegraph && codegraph version`. **Amended 2026-08-10:** the `brew trust` step is required — Homebrew 6.0.16 refuses casks from untrusted third-party taps, and every local rehearsal missed this because a `file://` tap never crosses that boundary. The instruction previously read "Per D-13, repeat against the **second** release's regenerated cask"; scope was reduced to one release (see the phase-gate caveat above). |
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

4. **G1 — the two `release:goreleaser` tap-credential preconditions have no standing automated guard.** Row `03-03 T3` states a three-leg conjunction; its declared command (`go test -run TestHomebrewTap`) parses **workflow YAML** and covers legs 1–2 only. Leg 3 — "release halts on a missing or non-distinct tap credential" — lives in `Taskfile.yml:765,767`, and `HOMEBREW_TAP_TOKEN` appears in **no** Taskfile test (`taskfile_shape_test.go` is the correct surface — it carries 39 `preconditions` assertions and a `TestTaskfileGatesFailLoud` — it simply does not cover these two). Per `03-SECURITY.md` AR-08 the guards were demonstrated once by running their text verbatim under `sh -c`, never through `task`.

   **Deliberately not closed, for two independent reasons.** (a) *Ownership:* the failure this defends against is GoReleaser's `client.NewIfToken` silently falling back to the release token — code this project does not own and cannot patch. Our precondition is ours, but a shape test over it would assert that two lines of YAML we wrote still say what we wrote; it would not prove a release halts. (b) *Natural detection:* if the guard were removed and the token went missing, the release ships without a cask push, which surfaces immediately as `brew upgrade` not offering the new version against a visibly stale tap — recoverable and patch-forward-able, not silent and not irreversible. (c) *It would buy nothing:* even with this guard built, gaps 1–3 and 5 leave the phase non-compliant regardless, so the work would not change the phase's status. Recorded rather than built.

5. **G2 — `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` is tautological.** It is row `03-03 T3`'s leg-1 coverage on paper. It compares the in-test constant `homebrewTapCredentialNames[:2]` against a hardcoded literal `{"APP_ID","APP_PRIVATE_KEY"}` and reads **no workflow file**, so it cannot fail for any reason other than someone editing the test itself. It runs green and does not bind reality. Recorded as `03-SECURITY.md` AR-01 with a todo filed (`2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md`). Unlike G1 this one **is** worth fixing eventually — it is the only *standing* assertion of the two-App distinctness that ROADMAP criterion 5 rests on, since D-17's `403` is one-time.

---

## Validation Audit 2026-08-10

| Metric | Count |
|--------|-------|
| Map rows | 11 |
| Green via a command executed this session | 6 |
| Proven once, no standing command **by explicit decision** (D-12, D-17, D-13) | 3 |
| Accepted limitation, no evidence by decision (D-18R) | 1 |
| Partial (leg-level gap) | 1 |
| Gaps found | 2 (G1 MISSING, G2 PARTIAL) |
| Resolved | 0 |
| Recorded as accepted | 2 |

**Verdict: VALIDATED (PARTIAL).** `status: validated`, `nyquist_compliant: false`.

### Why `nyquist_compliant: false`

Two sign-off criteria fail, and both failures are **recorded decisions rather than oversights**:

1. **Four requirements have no automated command at all.** D-12 (the RED proof deliberately does not re-fire), D-17 (the `403` is one-time, not a standing CI assertion), D-13 (needs a real published release), D-18R (structural argument, no executed evidence by maintainer decision). Every one of the four *was* proven — once, on purpose, with the evidence recorded in `03-EVIDENCE.md`.

2. **Sampling continuity is violated.** Rows `03-04 T1`, `03-04 T2` and `03-05 T2` are **three consecutive tasks with no automated verify**, spanning waves 3→4. The criterion says no 3 consecutive. Contrast phase 10, where the two unautomated tasks were non-adjacent and continuity held.

Note that **closing G1 would not have changed this verdict** — the four decisions above and the continuity violation stand regardless. The gap is recorded, not built.

### The honest framing

This phase's central behaviors are a cold `brew install`, a credential refused across a repository boundary, and a broken binary failing an install gate. None can be sampled in 15 seconds by construction, which this document's own opening paragraph anticipated at plan time. `nyquist_compliant: false` here means *"four proofs were deliberately made one-time"*, not *"something is untested"*. Reading it as the latter would misrepresent the phase.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies, **or** appear in Manual-Only / Named Coverage Gaps above — 6 automated, 5 in Manual-Only / Named Coverage Gaps
- [ ] **Sampling continuity: no 3 consecutive tasks without automated verify** — FAILS: rows `03-04 T1`, `03-04 T2`, `03-05 T2` are three consecutive (D-12, D-17, D-13). Accepted.
- [x] Wave 0 covers all MISSING references — Wave 0 complete, both suites green (5/5, 5/5)
- [x] No watch-mode flags
- [x] Feedback latency < 15s for automatable legs — quick command runs in ~1s cached, well inside budget
- [ ] **`nyquist_compliant: true` set in frontmatter** — deliberately `false`; see "Why" above

**Approval:** validated (PARTIAL) 2026-08-10 — maintainer-approved with the four one-time proofs and G1 recorded as accepted rather than closed.
