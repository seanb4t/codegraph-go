---
status: complete
phase: 04-codegraph-upgrade-homebrew
source: [04-01-SUMMARY.md, 04-02-SUMMARY.md, 04-03-SUMMARY.md, 04-04-SUMMARY.md, 04-05-SUMMARY.md, 04-06-SUMMARY.md]
started: 2026-08-11T18:55:00Z
updated: 2026-08-11T19:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Non-brew install still upgrades normally
expected: `codegraph upgrade --check` against `~/.local/bin/codegraph` (a non-brew install) behaves as before — resolves a version, reports upgrade availability, never mentions Homebrew, never refuses. Detection must not have turned `codegraph upgrade` into a hard dependency on Homebrew existing (ROADMAP criterion 4, UPGR-02).
result: pass
source: orchestrator-executed
observed: |
  $ ~/.local/bin/codegraph upgrade --check
  codegraph v0.8.0 is available (current: v0.5.1-8-g29a30b1-dirty)

  $ env PATH=/usr/bin:/bin:/usr/sbin:/sbin ~/.local/bin/codegraph upgrade --check
  codegraph v0.8.0 is available (current: v0.5.1-8-g29a30b1-dirty)   # exit 0
  Homebrew mentions in output: 0
  Resolved binary under a Caskroom/Cellar tree: no
note: Deterministic — executed rather than asked. The brew-stripped-PATH run is a stronger proof of criterion 4 than the declared test, which only required that a non-brew install upgrade normally.
plan: 04-01

### 2. The refusal message is actually actionable
expected: The refusal text a brew user sees names `brew upgrade codegraph` as the thing to run, states plainly that the binary is Homebrew-managed, and reads as a helpful redirect rather than an error.
result: pass
source: human (shown inline, no objection raised)
observed: |
  codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
  bare `upgrade` exit 1 | `--check` exit 0, same pointer line | `--force` exit 1
note: Single line, names the tree it detected and the exact command. Presented to the maintainer inline rather than asked as an errand.
plan: 04-01

### 3. `release:rehearse-cask` against a real Homebrew prefix
expected: `CASK_REHEARSE=1 task release:rehearse-cask` installs and uninstalls a REAL cask into your real Homebrew prefix, with the sentinel/marker assertions removed and two independent `find -newer` freshness re-checks in their place (Step 5 pre-install, Step 5d pre-uninstall). Deliverable D3 of plan 04-02, flagged `human_judgment` because it mutates the real prefix and is explicitly opt-in.
result: skipped
reason: "Maintainer decision 2026-08-11: the real-install path was already exercised end-to-end by plan 04-06 against the published tap, with the machine independently verified restored on 8 dimensions afterward. Rehearsing mutates the real Homebrew prefix a second time to prove a target whose assertions were already verified statically during the security audit (sentinel count 0, FRESH_MAN_PAGE_COUNT >= 3, two independent find -newer re-checks present, task check:goreleaser exit 0). Not a gap."
source: human
plan: 04-02
coverage_id: D3

### 4. Drift-guard RED demonstration was not watched to completion
expected: Plan 04-05's D3 claims the certificate-identity drift guard was demonstrated RED against three confirmed-applied mutations (semantic loosening, emptied file list, total-preserving relocation). Its own rationale records that the session's `task test` run "was not watched to completion" — `test:daemon` was still in progress. The orchestrator has since run the full suite four times at the merged tree (6 steps, 59 packages, 0 FAIL each) and re-ran `TestCosignIdentityPolicy` (2/2 PASS). Confirm you accept that as closing D3, or say so if you want the three RED mutations re-demonstrated live.
result: pass
reason: "Maintainer decision 2026-08-11: accepted as closed. The three-check drift guard was independently verified during the security audit (T-04-16) — total floor >= 7 calibrated against a measured pre-task baseline of 6, per-file set membership across all five restatement files, and a verify:self-upgrade region-scoped requirement, with the scan set confirmed COMPLETE (the only other file containing --certificate-identity-regexp is the test itself). TestCosignIdentityPolicy re-run 2/2 PASS; full task test suite run 4x at the merged tree (6 steps, 59 packages, 0 FAIL each). The unwatched background run named in D3's rationale was test:daemon, unrelated to this guard."
source: human
plan: 04-05
coverage_id: D3

### 5. Cask hooks no longer write or remove a marker file
expected: Cask hooks no longer write or remove a `.codegraph-brew-install` marker file; the falsified line-451 comment is corrected.
result: pass
source: automated
coverage_id: D1
plan: 04-02

### 6. Man-page assertion proves the current install produced fresh pages
expected: `hooks.post.install`'s man-page assertion proves the current install produced fresh pages via a before/after mtime+size baseline, rather than being satisfied by residue from a prior failed install.
result: pass
source: automated
coverage_id: D2
plan: 04-02

### 7. 03-EVIDENCE.md's false sentinel-stranding passages corrected
expected: `03-EVIDENCE.md`'s two false "a failed install can strand the sentinel" passages are corrected in place with checkable citations; both folded todos moved from `pending/` to `completed/`.
result: pass
source: automated
coverage_id: D4
plan: 04-02

### 8. `--help` documents the refusal and both exit behaviours
expected: `codegraph upgrade --help` names the Homebrew refusal, the `brew upgrade codegraph` pointer, and both exit behaviours.
result: pass
source: automated
coverage_id: D1
plan: 04-04

### 9. Published docs describe the shipped mechanism
expected: `README.md` and `docs/RELEASE.md` describe the shipped Homebrew-refusal mechanism instead of announcing it as future work.
result: pass
source: automated
coverage_id: D2
plan: 04-04

### 10. Signature verified before the prior binary is executed
expected: `verify:self-upgrade` downloads the prior release's cosign bundle into its own `SIG_DIR` and verifies it before `chmod +x`, with a named hard-fail if the bundle matched nothing.
result: pass
source: automated
coverage_id: D1
plan: 04-05

### 11. Every certificate-identity restatement agrees with the compiled pattern
expected: Every `--certificate-identity-regexp` restatement across five files exhibits boundary-case parity with the compiled `releaseWorkflowRefPattern`; 7 literals across 5 files (pre-task baseline 6).
result: pass
source: automated
coverage_id: D2
plan: 04-05

## Summary

total: 11
passed: 10
issues: 0
pending: 0
skipped: 1
blocked: 0

## Coverage Classification Notes

- **04-01 and 04-06 are `mode: legacy`** — neither SUMMARY carries a `coverage:` block, so no
  deliverable from either could be deterministically auto-passed. Tests 1 and 2 are derived
  from prose. These are the phase's two most consequential plans (the detection predicate and
  the real-install acceptance run). Combined with security finding UF-3 (04-01, 04-03, 04-04
  lack a `## Threat Flags` section), **04-01's SUMMARY is missing both structured sections**
  while shipping the most security-relevant code in the phase.
- **04-03 reports `all_auto_covered: true` with `total: 0`** — the `coverage: []` case. Plan
  04-03 amended planning prose only and shipped no user-observable behaviour, so it
  contributes no checkpoint.
- **04-05 D3 carries a malformed coverage entry** — `invalid_kind` at `verification[2].kind`
  (must be one of unit, integration, e2e, automated_ui, manual_procedural, other). Per the
  fail-safe rule a malformed entry is never dropped; D3 was independently classified `present`
  (`validation_failed`), so it is Test 4 either way. Worth fixing in the SUMMARY so the
  classifier stops erroring.
- **No cold-start smoke test injected** — this phase modified no `server.*`, `app.*`,
  `index.*`, `main.*`, `database/*`, `db/*`, `migrations/*`, `Dockerfile*` or
  `docker-compose*` path.

## Gaps

[none yet]
