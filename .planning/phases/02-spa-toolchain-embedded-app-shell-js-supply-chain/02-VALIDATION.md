---
phase: 2
slug: spa-toolchain-embedded-app-shell-js-supply-chain
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-23
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (Go side — this repo's only test framework). No JS test framework: no v1 requirement asks for one, and adding `vitest` is outside this phase's locked scope. |
| **Config file** | `Taskfile.yml` (existing) gains new targets; no new Go test config needed |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/upgrade/...` |
| **Full suite command** | `task test:unit` |
| **Estimated runtime** | ~60 seconds for the quick run; `task test:unit` is the existing suite's runtime |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/uiserver/... ./internal/upgrade/...` for handler/embed/structural changes; `task web:drift` for anything touching `web/`
- **After every plan wave:** `task test:unit` plus the new `web:drift` and the extended `proto:drift`
- **Before `/gsd-verify-work`:** full suite green, **including a demonstrated-RED run of BLD-03's guard against a deliberately staled `web/build/`** (rule `84d1gfpywd` — a guard is not trusted green until it has been watched fail)
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map is keyed by requirement until plans exist.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | RPC-03 | — | A client-side route serves `index.html`; `/codegraph.ui.v1.UIService/*` reaches the Connect handler; a miss under the immutable-asset prefix returns 404, never HTML | unit (Go, `httptest`) | `go test ./internal/uiserver/... -run TestSPAFallback` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-02 | — | The embedded FS file-list equals the on-disk `web/build/` file-list, `_app/` included (criterion 1 is a file-list diff, not "the build succeeded") | unit (Go, `fs.WalkDir` + `filepath.WalkDir` set-diff) | `go test ./internal/uiserver/... -run TestEmbeddedFSMatchesOnDisk` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-03 | — | Guard reports how many source files it hashed and fails RED against a deliberately stale `web/build/` | shell (Taskfile target) | `task web:drift` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-01 | — | `pnpm install --frozen-lockfile` succeeds at the Corepack-pinned version on a clean checkout | integration (CI step) | `corepack enable && pnpm install --frozen-lockfile` (cwd `web/`) | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-05 | — | `strictDepBuilds` is in effect — asserted positively, not inferred from a passing install | shell/CI | Taskfile target reading `pnpm-workspace.yaml`'s `strictDepBuilds` key and asserting `true` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-06 | — | `pnpm audit` runs; a sibling assertion counts packages from `pnpm-lock.yaml` **without reading `pnpm audit`'s exit code** | shell/CI | Taskfile target parsing the lockfile package count independently | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-07 | — | No `node`/`npm`/`npx`/`pnpm` invocation is reachable in `.goreleaser.yaml` or `release.yml` — checked structurally, not by grep | unit (Go, structural parse) | `go test ./internal/upgrade/... -run TestReleasePathHasNoJSToolchain` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/uiserver/spa_test.go` — RPC-03: fallback vs. 404 vs. RPC dispatch
- [ ] `internal/uiserver/embed_test.go` (or similar) — BLD-02 criterion 1: embedded-vs-on-disk file-list diff
- [ ] `Taskfile.yml` `web:drift` target — BLD-03: source-hash staleness with a positive count assertion, provable RED
- [ ] `Taskfile.yml` extension of `proto:gen` / `proto:drift` — D-05/D-06/D-07: second buf template, floor moved from 3 to **4**
- [ ] `.github/workflows/ci.yml` `test` job — `actions/setup-node` pinned to **Node 24** (Corepack is unbundled from Node 25+), `corepack enable`, `pnpm install --frozen-lockfile`, the `strictDepBuilds` assertion, `pnpm audit`, and the sibling lockfile-count assertion (D-13)
- [ ] `internal/upgrade/taskfile_shape_test.go` extension — BLD-07 structural proof, following that file's existing fixture pattern

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| "A developer with only Go on `PATH` … with no JS toolchain installed anywhere on the machine" | BLD-02 (criterion 1) | The automated test proves embedded-FS ≡ on-disk tree, which is the substance. It cannot prove the *absence* of a JS toolchain on the verifying machine — that is a property of the environment, not of the code, and a test asserting it would be asserting something about its own host. | On a machine (or container) with no `node`/`pnpm` on `PATH`: `go build ./cmd/codegraph && ./codegraph ui`, open the printed URL, confirm the app renders and `GetStatus` data appears. Record which `PATH` was used. |
| The rendered app is visually coherent (D-17/D-18 shell) | — | No requirement asks for visual regression testing, and none is configured. | Open `codegraph ui`, confirm the layout renders with the four navigation slots and no console errors. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] BLD-03's guard has been **watched fail** against a staled `web/build/`, then watched pass on a clean revert
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
