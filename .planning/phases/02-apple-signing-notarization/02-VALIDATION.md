---
phase: 2
slug: apple-signing-notarization
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-09
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go stdlib testing) + Taskfile targets |
| **Config file** | Taskfile.yml |
| **Quick run command** | `task test:unit` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~60 seconds (unit); full suite longer — `test:integration` / `test:wireoracle` / `test:daemon` each shell out to `go build` |

---

## Sampling Rate

- **After every task commit:** Run `task test:unit`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | SIGN-01 | — | TBD | TBD | TBD | TBD | ⬜ pending |
| TBD | TBD | TBD | SIGN-02 | — | TBD | TBD | TBD | TBD | ⬜ pending |
| TBD | TBD | TBD | SIGN-03 | — | TBD | TBD | TBD | TBD | ⬜ pending |
| TBD | TBD | TBD | SIGN-04 | — | TBD | TBD | TBD | TBD | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] TBD — populated by the planner; RESEARCH.md flags that neither `test/integration/main_test.go:39-57` nor `test/wireoracle/main_test.go` has a seam to point `TestMain` at an already-downloaded binary (both hardcode `go build`), which success criterion 4 requires. Precedent to transplant: `internal/upgrade/verify_release_e2e_test.go:97-105` (`CODEGRAPH_E2E_BINARY` / `CODEGRAPH_E2E_BUNDLE`).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Local notarization rehearsal against Apple's service | SIGN-01 | D-08 locks rehearsal to the maintainer's Mac; requires a Developer ID Application certificate and App Store Connect API key that CI-triggerable workflows must not be able to reach (a `pull_request`-reachable credential is a security regression, not a fix) | Run the guarded maintainer-only Taskfile target the planner adds; it must hard-fail BY NAME on a missing cert or API key before any network round-trip (D-09) |
| `spctl -a -vv -t exec` RED baseline on the v0.5.1 published darwin asset | SIGN-03 | Requires real macOS with Gatekeeper; `spctl` has no CI-usable equivalent, and the quarantine xattr must be confirmed present via `xattr -p` first or the result is not evidence | Download the v0.5.1 darwin asset, set `com.apple.quarantine`, confirm with `xattr -p`, run `spctl`, record `rejected` before any green run |
| `syspolicy_check distribution` on the notarized asset | SIGN-02 | Real-hardware Apple tooling; RESEARCH.md flags its exact input/output semantics as an unresolved open question | Per the procedure the planner derives; treat the first run as discovery, not as a pass/fail gate |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
