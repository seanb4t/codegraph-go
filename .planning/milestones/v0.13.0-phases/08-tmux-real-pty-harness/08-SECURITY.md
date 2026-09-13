---
phase: "08"
slug: "tmux-real-pty-harness"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-10"
---

# Phase 08 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Developer/CI home ↔ test process | Every pane command and every spawned subprocess redirects `HOME`/`USERPROFILE` to a per-test `t.TempDir()`, so the harness never reads or writes the real `~/.codegraph` registry or real agent configs | Daemon registry records, agent config files (`.gemini/` et al.) |
| Test process ↔ real tmux server | `runTmux` drives the tmux CLI via `exec.Command("tmux", args...)` — variadic separate argv, never a shell string | Session names, pane geometry, key sequences, capture-pane output |
| Test process ↔ spawned `codegraph` binary | `TestMain` builds the binary from the working tree into a temp dir; `seedRunningDaemon` starts a real background `codegraph daemon start` | Build artifact path, daemon lifecycle signals, registry state |
| Repo working tree ↔ deliberate product mutations (08-04) | Plan 08-04 mutates shipped guards to prove assertions can fail, bracketed by cleanliness gates on both sides | Source of `internal/cli/daemon.go`, `internal/cli/tui/{daemonpicker,agentpicker}.go` |
| GitHub-hosted runner ↔ distro package archive | `test:tmux:install` installs tmux via `sudo apt-get` on an ephemeral runner | tmux binary from the runner image's trusted archive |
| In-repo fixture ↔ external GitHub ruleset 20157557 | `requiredCheckNames` mirrors a live branch-protection ruleset that only a human can change | Required-status-check context strings |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-08-01 | Tampering | `runTmux` argv construction, `test/tmux/session.go:31` | low | mitigate | `exec.Command("tmux", args...)` passes every element as a separate argv entry; zero `sh -c`/`bash -c` anywhere in `test/tmux` (verified: 0 occurrences). Every value is a fixed literal or a `t.TempDir()`/`binPath`-derived path | closed |
| T-08-02 | Information Disclosure | every spawned process, all of `test/tmux` | high | mitigate | All 5 pane commands use `env HOME=%s USERPROFILE=%s` with a `t.TempDir()` home (`daemon_empty_test.go:39`, `daemon_picker_test.go:34`, `install_cancel_test.go:34,67`, `frame_stability_test.go:44`). `daemon_seed.go:35` builds the isolated env and assigns it to BOTH subprocesses — `initCmd.Env` (`:38`) and `daemonCmd.Env` (`:44`), individually verified. `main_test.go:136`'s `go build` deliberately does not isolate `HOME` (it would break `GOMODCACHE` resolution); it builds rather than runs the product against a registry | closed |
| T-08-03 | Denial of Service | `seedRunningDaemon`'s background `*exec.Cmd` | medium | mitigate | SIGTERM-then-`Wait()` `t.Cleanup` registered at `daemon_seed.go:49`, immediately after `Start()` succeeds; the only `t.Fatalf` calls before it occur when no daemon exists yet. Verified at HEAD: `pgrep -fl 'codegraph daemon'` → 0 matches, positive-controlled against `pgrep -fl bash` → 101 matches, so the zero is a real absence and not a broken instrument | closed |
| T-08-04 | Denial of Service | tmux sessions, including the 12-row short pane | low | mitigate | `newSessionSized` (`session.go:87`) inherits the `kill-session` `t.Cleanup` at `session.go:98`, which discards tmux's exit-1-on-already-gone | closed |
| T-08-05 | Tampering | `internal/cli/daemon.go`, `internal/cli/tui/{daemonpicker,agentpicker}.go` | high | mitigate | Every mutation bracketed by `git diff --quiet -- <file>` before and after, both recorded per log entry; shared-file families applied strictly one at a time. Verified independently at HEAD: `git diff --quiet fb346814..HEAD -- internal/cli internal/cli/tui` exits 0 (byte-identical), no 08-04 commit touches any `.go` file, and both `v.AltScreen = true` tokens are restored exactly once each | closed |
| T-08-06 | Repudiation | `task test:tmux`, the `tmux-e2e` CI status check | high | mitigate | D-01/D-02 executed-count equality; the job carries neither `continue-on-error` nor `if:` (verified absent from the job body), so it cannot skip its own assertion. Verified: `executed=5 skipped=0 expected=5`, exit 0. **Caveat — see WR-03 in `08-REVIEW.md`:** the register's claim that the counts print "on every run" is FALSE under one reproduced failure mode (a `GOTOOLCHAIN` resolution error merges plain text into the JSON stream, `jq -s` aborts, and `set -euo pipefail` kills the target before both the count line and the designed `::error::` diagnostic). The threat nonetheless stays closed because that path exits **5**, not 0 — positive-controlled against a healthy run that exits 0 and prints both — so a run that asserted nothing can never present as a green check. What WR-03 costs is diagnosability, not attestation integrity | closed |
| T-08-07 | Elevation of Privilege | `test:tmux:install`'s `sudo apt-get` | low | accept | Accepted risk — see Accepted Risks Log (R-08-01) | closed |
| T-08-08 | Tampering | `hashConfigTree`, `test/tmux/confighash.go` | low | mitigate | `filepath.WalkDir` over the whole throwaway tree (`:42`) rather than an enumerated path list, so a newly registered agent target cannot slip a write past it; `sort.Strings(relPaths)` (`:60`) removes walk-order as a false-difference source. Collision resistance is not relied on — this is change detection, not a boundary | closed |
| T-08-09 | Spoofing | the pinned tmux version string | medium | mitigate | D-11 install-then-assert: the CI branch asserts `tmux -V` equals `TMUX_EXPECTED_VERSION` and fails printing BOTH values on mismatch. The committed value is the declared bootstrap sentinel `UNPINNED-BOOTSTRAP` (`Taskfile.yml:214`) — deliberately unreportable by any real tmux, so the first real runner supplies the true string via its own failure output rather than anyone guessing it | closed |
| T-08-10 | Tampering | `requiredCheckNames` vs GitHub ruleset 20157557 | medium | mitigate | The fixture is deliberately left unedited (verified byte-unchanged since `fb346814`) until the external ruleset actually carries the context, so a fixture claiming a gate that does not exist is never committed. Residual drift is in the SAFE direction (under-reporting, not over-claiming). The outstanding human action is recorded at `STATE.md:290` | closed |
| T-08-11 | Tampering | third-party actions in the `tmux-e2e` job | low | mitigate | Both external `uses:` refs are full-SHA pinned — `actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10`, `actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16`; the third is a local path (`./.github/actions/install-task`). `setup-go` is invoked DIRECTLY with only `go-version-file: go.mod` and no sibling `go-version:` key, so the `govulncheck-action` defaulted-version-forwarding hazard does not apply here | closed |
| T-08-12 | Tampering | family (a)'s two-location mutation | high | mitigate | 08-RESEARCH.md Pitfall 1 established empirically that the single-location recipe does not reproduce the leak, correcting D-06. The requirement to REPORT a non-failing RED rather than write it up was exercised in practice: family (d)'s mutation did not fail, and was recorded as an observed negative result with root cause instead of being presented as a demonstration | closed |
| T-08-13 | Repudiation | the pasted RED transcripts in `08-MUTATION-LOG.md` | medium | mitigate | Each family entry names the exact command producing its output and pastes it verbatim; the log's convention forbids paraphrase, and a wrong entry is replaced by re-running rather than edited | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-08-01 | T-08-07 | `test:tmux:install` runs `sudo apt-get install tmux` on an ephemeral GitHub-hosted runner. The package comes from the distro archive the runner image already trusts, and the runner is destroyed after the job. Accepted rather than mitigated because the only alternative — a third-party install action — enlarges the supply-chain surface instead of shrinking it | maintainer (plan-time disposition, 08-01 and 08-03 threat models) | 2026-09-10 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-10 | 13 | 13 | 0 | orchestrator (ASVS L1 short-circuit; see note) |

**Audit note — why the L1 short-circuit was taken, and what corroborated it.**
`threats_open: 0` with `register_authored_at_plan_time: true` and `asvs_level: 1` permits skipping the
auditor. It was taken deliberately, on this evidence: all four PLANs carry parseable `<threat_model>`
blocks (19 rows, 13 unique threats); no threat is `critical`; nothing in this phase publishes or
irreversibly mutates anything outside the working tree; and every one of the 13 mitigations was checked
at file:line at HEAD by execution rather than by reading the register's own claims — with a positive
control wherever an absence was asserted (`pgrep` controlled against a known-present process; heading
counts controlled before trusting a zero `## Threat Flags` result). No SUMMARY carries a
`## Threat Flags` section, verified as a real absence rather than a pattern miss.

The short-circuit skipped the auditor but NOT independent scrutiny: a `deep` `gsd-code-reviewer` pass
over all 13 changed files ran concurrently and returned 0 Critical / 5 Warning / 4 Info
(`08-REVIEW.md`). That pass is what surfaced WR-03, which falsified the "prints on every run" half of
T-08-06's plan-time mitigation text. Had the L1 pass stood alone, that sentence would have entered this
record as verified. The threat classification is unchanged — the failure path exits 5, so an
assert-nothing run cannot read as green — but the evidence behind it is now accurate rather than
transcribed. Recorded here because a security record whose entries were copied from a plan and never
executed is exactly the vacuous-attestation shape repo rule 84d1gfpywd exists to reject.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-10

**Outstanding, not security-blocking:**
- WR-01 … WR-05 in `08-REVIEW.md` — code-quality findings, 0 Critical. WR-03 is referenced in T-08-06 above.
- GitHub ruleset 20157557 must gain the `tmux e2e (real-pty harness, TTY-01..TTY-07)` context, after which the same string is added to `requiredCheckNames` (T-08-10; `STATE.md:290`).
