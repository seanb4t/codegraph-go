# Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-19
**Phase:** 8-Surface Reconciliation & Signed v1.0.0 Release
**Mode:** --auto --chain (all gray areas auto-selected; recommended option chosen for each)
**Areas discussed:** Phase sequencing, impact depth locus, files directory-filter name, short-flag collisions, affected scripting flags, flag-audit artifact, no-new-CGo proof, release sequencing, benchmark scope

---

## Phase sequencing (SURF block vs REL block)

| Option | Description | Selected |
|--------|-------------|----------|
| SURF fully green → then REL → tag last | Reconcile all flags, verify, then run the release/audit; signed tag is the last action | ✓ |
| Interleave SURF and REL work | Mix flag edits with release prep | |

**Choice:** SURF-01..05 green first, then REL-01..04, tag last (D-01).
**Notes:** REL-04's drop-in gate re-runs the SURF-05 flag audit, so flags must be final before the tag is cut.

---

## SURF-01 — impact depth default (change locus)

| Option | Description | Selected |
|--------|-------------|----------|
| Change shared-engine default (0→2) | CLI + MCP `impact`/BFS surfaces inherit depth-2 in one commit | ✓ |
| Change only the CLI flag default | MCP surface would keep the old default — divergent | |

**Choice:** Change the engine's default-when-0 from 5 to 2 (D-02).
**Notes:** Matches the v1.0 pattern — behavioral defaults live in `internal/query` so CLI==MCP. Keep max-50 clamp; regen golden if impact appears.

---

## SURF-02 — new `files` directory filter (flag name)

| Option | Description | Selected |
|--------|-------------|----------|
| `--dir <glob>` (keep `--filter`=language) | Distinct self-describing name; language filter untouched; divergence documented | ✓ |
| Reuse `--filter` for directory (TS spelling) | Would break the existing Go language-filter surface | |
| `--path-filter` / other name | Wordier; same intent | |

**Choice:** `--dir <glob>` for the directory filter; `--filter` stays language (D-03).
**Notes:** The locked "keep ours + add TS's" divergence (SURF-02 requirement) — recorded in `docs/FLAG-PARITY.md`.

---

## SURF-03 — short-flag alias collisions

| Option | Description | Selected |
|--------|-------------|----------|
| TS letter where free; keep Go binding + document where taken | No silent remap of established shorts (`-l`=line, etc.); long flag always parity | ✓ |
| Force exact TS short letters everywhere | Would collide with / remap existing Go shorts | |

**Choice:** Adopt TS shorts where free; document divergences where a Go binding exists (D-04).
**Notes:** Respect `-p/-q/-v/-y/-f/-l`. Per-command letter map lands in the SURF-05 audit; no intra-command cobra collision (build catches it).

---

## SURF-04 — `affected` scripting flags

| Option | Description | Selected |
|--------|-------------|----------|
| Add `--stdin`/`--depth`/`--filter <glob>`/`--quiet`, TS semantics, plain output | Full git-hook/CI scripting parity, plain when piped | ✓ |
| Add a subset | Incomplete vs TS | |

**Choice:** All four flags, TS-parity semantics, machine-readable plain output (D-05).
**Notes:** `--stdin --quiet` must emit a clean parseable path list (respects Phase-6 rendering seam).

---

## SURF-05 — flag audit + divergence doc

| Option | Description | Selected |
|--------|-------------|----------|
| `docs/FLAG-PARITY.md` matrix + optional tree-walk drift test | Single source REL-04 reads; optional test prevents drift | ✓ |
| Doc only, no test | Can silently drift from the real flag surface | |

**Choice:** `docs/FLAG-PARITY.md` per-command matrix; drift test is discretion (D-06).
**Notes:** Records the `--filter`/`--dir` divergence, short-flag divergences, `search` (Go-only), `migrate` (accepted divergence).

---

## REL-01 — proving "no NEW CGo" from the Charm closure

| Option | Description | Selected |
|--------|-------------|----------|
| Dependency-closure diff over `charm.land/*` (zero cgo pkgs) | Correct: binary already links CGo tree-sitter, so a clean build proves nothing about Charm | ✓ |
| `CGO_ENABLED=0` clean-build assertion | Would FAIL on pre-existing tree-sitter CGo — false signal | |

**Choice:** Closure diff asserting the Charm deps add zero cgo packages, then govulncheck + SBOM regen + reproducible double-build (D-07).
**Notes:** GOTCHA carried from Phase 7 — Windows cross-`go vet` needs `gcc-mingw-w64` + `CGO_ENABLED=1` (not zig) because `internal/daemon` transitively imports CGo tree-sitter.

---

## REL-02 — release cut sequencing

| Option | Description | Selected |
|--------|-------------|----------|
| integration-branch → squash-merge main → tag `v1.0.0` on main (tag push manual) | Milestone branch model; merge signs; phase produces readiness, maintainer pushes tag | ✓ |
| Tag directly on the integration branch | Bypasses the main squash-merge signing model | |

**Choice:** Squash-merge to main, tag `v1.0.0` on main; the tag push is the maintainer's runbook action (D-08, D-09).
**Notes:** Stable `vX.Y.Z` (no suffix) = full release + `upgrade` latest; `milestone-v*` markers never fire. LOCKED `release.yml`/`verify.go` contract. Mandatory pre-tag 6-target `go list -mod=readonly` sweep.

---

## REL-03 — benchmark scope

| Option | Description | Selected |
|--------|-------------|----------|
| Re-run existing `bench.yml`, refresh `docs/BENCHMARKS.md` + release notes | Reuse proven v0.1 methodology; comparable numbers | ✓ |
| Author a new bench framework | Redundant; breaks comparability | |

**Choice:** Re-run the existing harness head-to-head vs TS 1.3.1, refresh published numbers (D-10).
**Notes:** Median-of-3, same corpora as v0.1 for comparability.

---

## Claude's Discretion

- Exact `--dir` glob-match semantics (prefix vs full-glob vs doublestar) — match TS on the golden corpus.
- The precise per-command short-flag letter map and which divergences land.
- Whether SURF-05's audit gets a tree-walk drift test or stays doc-only.
- Whether `affected --depth` shares the `impact` clamp helper or a local copy.
- SBOM tool choice if the goreleaser Syft default needs supplementing (cyclonedx-gomod app-mode).
- Splitting the phase into SURF + REL waves vs interleaving (D-01 only mandates SURF-green-before-tag).

## Folded Todos

- **"Document release procedures (maintainer runbook)"** (score 0.6) — folded into REL-02 as `docs/RELEASE-PROCEDURES.md`: pre-tag 6-target check, tag conventions (rc/stable/marker), `verify.go` LOCKED contract, post-release verification (`cosign verify-blob` + `slsa-verifier`), rollback, `-c commit.gpgsign=false` fallback for automated commits only (repo rule xmz3xknbj0).

## Deferred Ideas

- DMON-FUT-01 (detached per-project daemons + unix-socket sharing) — later milestone.
- Backlog 999.1 (Taskfile.yml + CONTRIBUTING.md) — pairs with the runbook but is its own item.
- Backlog 999.2 (tmux real-PTY e2e/UAT harness) — separate backlog.
- Team-scale / central-server / annotations / Svelte web UI (SEED-001) — post-v1.0.
