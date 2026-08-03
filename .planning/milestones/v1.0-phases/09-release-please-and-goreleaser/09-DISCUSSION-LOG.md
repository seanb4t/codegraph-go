# Phase 9: release-please + GoReleaser - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-28
**Phase:** 9-release-please-and-goreleaser
**Mode:** `--auto --chain` — no user prompts; the recommended option was auto-selected for every question and logged here plus inline in `09-CONTEXT.md`.
**Areas discussed:** Tag-trigger mechanism, Release-creation collision, GoReleaser's actual role, How v1.0.0 gets cut, Version source of truth, Conventional-commit input gate, v1.0 merge model, Prerelease/rc story

---

## Tag-trigger mechanism — how does release-please's tag fire `release.yml`?

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub App token | release-please authenticates via `actions/create-github-app-token`; App-created tags DO trigger downstream workflows, so `release.yml` fires unchanged. Roadmap-preferred. | ✓ |
| Fine-grained PAT | Same triggering behavior, but a long-lived credential in repo secrets — key-custody risk the project's supply-chain posture explicitly avoids elsewhere (keyless cosign). | |
| `workflow_dispatch` at the tag ref | release-please job dispatches `release.yml --ref <tag>`. SAN-safe (the pattern constrains the ref, not the event) but requires editing the LOCKED file to add a trigger. | fallback (D-03) |
| Collapse into one workflow (engram's model) | Signs from `@refs/heads/main` → SAN mismatch → silently breaks `codegraph upgrade` for every shipped binary. | rejected |

**Selected:** GitHub App token (D-02), with `workflow_dispatch`-at-tag-ref recorded as the documented Path-B fallback (D-03).
**Notes:** The default `GITHUB_TOKEN` cannot be used — tags it pushes do not trigger other workflows, which is exactly why engram collapses everything into one job. Repo currently has zero secrets, so App creation/installation is a blocking maintainer-manual prerequisite. If Path B is ever used, its ref-shape guard must be proven non-vacuous (Phase-8 `CR-01`/`WR-02` lesson: a guard that never fires is not a guard).

---

## Release-creation collision — `gh release create` vs. release-please's own release

| Option | Description | Selected |
|--------|-------------|----------|
| Create-if-absent, else `upload --clobber` | release-please owns the release + changelog body; `release.yml` only uploads assets idempotently. Manual rc tags still take the create path. | ✓ |
| `skip-github-release: true` on release-please | release-please tags only; `release.yml` keeps creating the release — but then the changelog never reaches the release body. | |
| Force `gh release create --clobber`-style overwrite | Would overwrite release-please's changelog with `--generate-notes` output. | |

**Selected:** create-if-absent-else-upload-clobber (D-04).
**Notes:** This is the phase's highest-risk single edit. release-please creates a GitHub Release automatically on release-PR merge (its own design docs), so today's `release.yml:279` `gh release create "$TAG" … --generate-notes` would fail outright on an existing release, and if forced would clobber the changelog. `--clobber` also delivers the idempotency roadmap criterion 3 asks for. `--generate-notes` must NOT appear on the upload path.

---

## GoReleaser's actual role — does `goreleaser release` enter the picture?

| Option | Description | Selected |
|--------|-------------|----------|
| Keep `goreleaser build --single-target` (build-only) | Matches what the repo actually does today; signing/SBOM/publish stay in the hand-written `assemble` job. Roadmap criterion 3 recorded as an accepted divergence. | ✓ |
| Migrate to `goreleaser release` with `signs:`/`sboms:`/`replace_existing_artifacts` | Matches the roadmap's engram-modeled criterion 3 literally, but needs GoReleaser Pro `release --split`/`--merge` and cannot express per-binary cosign or the native darwin matrix. | |

**Selected:** build-only preserved; roadmap criterion 3 amended rather than silently dropped (D-05).
**Notes:** Decisive evidence is in-repo: `.goreleaser.yaml` lines 129–146 already document its own `archives:`/`checksum:` blocks as dead config, and its header explains there is deliberately no `signs:`/`sbom:` block. `internal/upgrade`'s `defaultVerify` hashes the binary itself, not a checksums file — per-binary signing is a hard requirement GoReleaser OSS cannot satisfy. Amendment is recorded the same way Phase 8's goal amendment was, rather than quietly narrowing the criterion.

---

## How `v1.0.0` specifically gets cut

| Option | Description | Selected |
|--------|-------------|----------|
| Seed manifest at `0.1.0` + one-shot `Release-As: 1.0.0` footer | Deterministic and auditable; `Release-As` takes precedence at the top of release-please's `buildNewVersion`. One-shot, then gone. | ✓ |
| `release-as` pinned in `release-please-config.json` | Sticky — would force 1.0.0 on every subsequent run until removed. | |
| Rely on `feat!:` / `BREAKING CHANGE` | Yields 1.0.0 only via pre-major rules, and misrepresents a milestone as a breaking change. | |

**Selected:** manifest seeded `0.1.0` + one-shot `Release-As: 1.0.0` (D-06).
**Notes:** Newest tag is `v0.1.0`; default `0.x` behavior would compute `0.2.0` from the accumulated `feat:` commits, never `1.0.0`.

---

## Version source of truth

| Option | Description | Selected |
|--------|-------------|----------|
| ldflags only — no `extra-files` | `.goreleaser.yaml` already injects `-X …version.Version={{ .Tag }}`; the tag IS the version. `Version = "dev"` stays the non-release sentinel. | ✓ |
| Add `internal/version/version.go` to release-please `extra-files` | A second source release-please rewrites — invites the exact doc-vs-code drift this repo has been bitten by twice. | |

**Selected:** ldflags only (D-07).
**Notes:** `release-type: go` manages `CHANGELOG.md` + the manifest and needs no Go source file.

---

## Conventional-commit input gate

| Option | Description | Selected |
|--------|-------------|----------|
| PR-title conventional-commit check in `ci.yml` | Gates the text that actually becomes the commit under squash-merge. Codifies existing practice (30/30 recent commits conform). | ✓ |
| Lint every branch commit | Gates text that never reaches `main` under a squash-merge model. | |
| Rely on convention, add nothing | release-please's entire output is a function of commit messages; leaving it unenforced makes the changelog silently degradable. | |

**Selected:** PR-title check in `ci.yml` (D-08).
**Notes:** Repo currently enforces nothing — no commitlint, no PR-title lint.

---

## v1.0 merge model ⚠ (documented model vs. actual practice)

| Option | Description | Selected |
|--------|-------------|----------|
| Fast-forward, preserving full history | Matches what the repo actually did for v0.1. release-please sees all 477 commits and builds a rich `v1.0.0` changelog from the 160 `feat`/`fix`/`perf` entries; its default `changelog-sections` hide the ~217 `docs:` planning commits automatically. | ✓ |
| Squash-merge (Phase-8 D-08's literal wording) | Collapses 477 commits into one message; `v1.0.0`'s changelog becomes one line, discarding the input release-please exists to consume. | |
| `--no-ff` merge commit | Also preserves history, but introduces the first merge commit `main` has ever had — no benefit over fast-forward here since `main` is already an ancestor of HEAD. | |

**Selected:** fast-forward (D-09).
**Notes:** ⚠ **Correction made during discussion.** The recommendation was initially written as "honor Phase-8 D-08's squash-merge", then corrected after checking the repo. Evidence: `git log --merges main` is **empty** (zero merge commits ever), `main` **is an ancestor of HEAD** (fast-forwardable today), the branch is **477 commits ahead** with **160 `feat`/`fix`/`perf`**. Engram `8sa948y0g4` independently records "fast-forward merge not squash" for the v0.1 milestone. D-08's *substance* (integration branch → `main` → tag on `main`) is honored in full; only its merge-mechanic wording is superseded, on evidence. Rated one-way: merge shape is fixed the moment it lands on `main`. **If a true squash was actually intended, override before planning.**

---

## Prerelease / rc story

| Option | Description | Selected |
|--------|-------------|----------|
| Stable-only automation; manual rc tags retained | rc tags still match `v[0-9]*` and still fire the signed build — they simply take D-04's create branch. Keeps the automated path narrow. | ✓ |
| Configure release-please `prerelease: true` + `prerelease-type` | More automation surface than REL-02 needs right now. | |

**Selected:** stable-only (D-10).
**Notes:** This is *why* D-04 must handle both create and upload paths rather than assuming a release always pre-exists.

---

## Claude's Discretion

- SHA-pinning vs. tag-pinning the release-please action (prefer SHA, matching `release.yml`'s convention for every other third-party action).
- Exact `release-please-config.json` shape: `changelog-sections`, `include-component-in-tag`, `bump-minor-pre-major`.
- PR-title lint implementation: off-the-shelf action vs. a ~10-line `grep -E` step (the latter avoids a new pinned dependency).
- Create-vs-upload branch detection: `gh release view "$TAG"` exit status vs. an explicit workflow input.
- Where the GitHub App setup procedure lives: `docs/RELEASE-PROCEDURES.md` vs. a sibling `docs/RELEASE-AUTOMATION.md`.
- Whether to delete `.goreleaser.yaml`'s dead `archives:`/`checksum:` blocks or keep them as documented intent — either is defensible, but do not change them silently.

## Deferred Ideas

- Migrating to `goreleaser release` (needs GoReleaser Pro; would re-derive the per-binary cosign and native-darwin guarantees Phase 8 audited).
- Changing `releaseWorkflowRefPattern` to allow a consolidated release-please+ship workflow — needs a migration story for already-shipped binaries first.
- release-please prerelease automation for automated rc cuts.
- Making the repo public (moots the SLSA generator's `private-repository: true` opt-in).
- Phase 10 — `Taskfile.yml` + `CONTRIBUTING.md`, deliberately sequenced after this phase.
- Backlog 999.2 — tmux real-PTY e2e/UAT harness.
- Team-scale / central-server / annotations / local Svelte web UI (SEED-001) — post-v1.0.
