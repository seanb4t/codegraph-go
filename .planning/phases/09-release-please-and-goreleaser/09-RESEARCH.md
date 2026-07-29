# Phase 9: release-please + GoReleaser - Research

**Researched:** 2026-07-28
**Domain:** GitHub Actions release automation (release-please + GitHub App auth + existing signed-build pipeline wiring)
**Confidence:** MEDIUM — every mechanism below is confirmed against release-please/release-please-action/create-github-app-token source (Context7, MEDIUM tier) and/or official GitHub docs, and cross-checked against this repo's actual live state via `gh api` (VERIFIED). The one thing nothing can verify pre-release is whether a real App-token-authored tag push fires `release.yml` in *this specific* repo's Actions settings — see Validation Architecture.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 (untouchable):** `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` = ``^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`` is frozen this phase. Forbidden: renaming `release.yml`, removing its `push.tags: v[0-9]*` trigger, moving `id-token: write` cosign signing into another workflow file, signing from a non-tag ref. The "collapse into one engram-style workflow + change the regex" alternative is rejected for this phase.
- **D-02 (primary tag-trigger mechanism):** A **GitHub App token** (via `actions/create-github-app-token`) authenticates the new release-please workflow so its tag push fires `release.yml` (default `GITHUB_TOKEN` pushes do not trigger other workflows). Requires two new repo secrets (`APP_ID`, `APP_PRIVATE_KEY`) — the repo currently has **zero secrets configured**. App creation/installation is a maintainer-manual blocking checkpoint, not an executor task.
- **D-03 (documented fallback only, not primary):** If the App is unavailable, add `workflow_dispatch` alongside (never instead of) `release.yml`'s `push.tags` trigger, dispatched at the tag ref (`gh workflow run release.yml --ref "$TAG"`). SAN-safe only when dispatched at a tag ref, not a branch. MUST carry a guard that hard-fails when `github.ref` doesn't start with `refs/tags/v`, plus a test proving the guard fires on a rejecting input.
- **D-04 (highest-risk edit):** release-please owns the GitHub Release and its notes. `release.yml`'s `Publish GitHub release` step becomes **create-if-absent-else-upload-clobber**: release exists (release-please path) → `gh release upload "$TAG" --clobber codegraph_*` (leave body/prerelease flag untouched); release absent (manual `rc` tag, D-10) → keep today's `gh release create … --generate-notes` + `-`-suffix `--prerelease` logic.
- **D-05 (roadmap divergence, recorded):** Keep `goreleaser build --single-target`. Do **NOT** introduce `goreleaser release`. `.goreleaser.yaml`'s `archives:`/`checksum:` blocks stay documented-dead. Roadmap criterion 3's `replace_existing_artifacts: true` is satisfied instead by D-04's `--clobber` idempotency — an accepted, recorded divergence, not a silent drop.
- **D-06:** Seed `.release-please-manifest.json` with `0.1.0` (the real current tag), then force the first cut with a one-shot `Release-As: 1.0.0` footer on an empty `chore: release 1.0.0` commit. Do NOT put `release-as` in `release-please-config.json` (sticky — pins every future release to 1.0.0).
- **D-07:** ldflags stay the sole build-time version source. Do NOT add `internal/version/version.go` to release-please `extra-files`.
- **D-08:** Add a lightweight PR-title conventional-commit check to `ci.yml`, gating the **PR title** (squash-merge model makes the title the commit message release-please parses).
- **D-09:** Fast-forward the v1.0 integration branch onto `main` (NOT squash) — `main` has zero merge commits and is fast-forwardable today; this supersedes Phase-8 `08-CONTEXT.md`'s squash wording on repo evidence. A squash would collapse 477 commits (160 `feat`/`fix`/`perf`) into one line, starving release-please's changelog of its actual input.
- **D-10:** Automate stable releases only. Keep manual `rc` tags as a documented escape hatch — no release-please prerelease strategy this phase. `rc` tags still fire `release.yml` via D-04's *create* branch, not its *upload* branch.

### Claude's Discretion

- SHA-pin vs tag-pin for the release-please action — prefer SHA for consistency with the repo's existing convention.
- Exact `release-please-config.json` shape: `changelog-sections` customization, `include-component-in-tag: false`, `bump-minor-pre-major` (moot once at 1.x).
- PR-title lint: off-the-shelf action vs. ~10-line `grep -E` step (repo prefers hand-written, auditable CI shell).
- `assemble` job's create-vs-upload branch decided by `gh release view "$TAG"` exit status vs. an explicit workflow input.
- GitHub App setup procedure: lives in `docs/RELEASE-PROCEDURES.md` or a sibling `docs/RELEASE-AUTOMATION.md`.
- Keep or delete `.goreleaser.yaml`'s dead `archives:`/`checksum:` blocks — either defensible, must not change silently.

### Deferred Ideas (OUT OF SCOPE)

- Migrating to `goreleaser release` with `signs:`/`sboms:` blocks (needs GoReleaser Pro `release --split`/`--merge`).
- Changing `releaseWorkflowRefPattern` to allow an engram-style consolidated workflow (needs a migration story for shipped binaries first — its own phase, post-v1.0).
- release-please prerelease automation (`prerelease: true` + `prerelease-type`) for automated `rc` cuts.
- Making the repo public (SLSA `private-repository: true` opt-in becomes moot, not a Phase 9 action).
- Phase 10 (`Taskfile.yml` + `CONTRIBUTING.md`) — deliberately sequenced after this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-02 | Releases are cut by release-please from Conventional Commits — version bump, `CHANGELOG.md`, and tag creation all happen without a human running `git tag` — and the resulting signed artifacts still satisfy `internal/upgrade/verify.go`'s cosign identity (`releaseWorkflowRefPattern`), so `codegraph upgrade` keeps working for already-shipped binaries | Architecture Patterns (App-token wiring, manifest seeding, Release-As precedence), Common Pitfalls (workflow-trigger asymmetry, GH-release collision, PR-title-lint trigger gap), Validation Architecture (mechanical guards + the one live-only proof, and the cheapest honest way to close that loop pre-v1.0.0) |
</phase_requirements>

## Summary

release-please's manifest-driven mode needs exactly three things to safely take over version/changelog/tag ownership of this repo without a bootstrap PR: a `release-please-config.json` with `release-type: go` and `packages: {".": {}}`, a `.release-please-manifest.json` seeded to the real current version (`{".": "0.1.0"}`), and a workflow that runs the action on every push to `main`. Because a GitHub Release already exists for tag `v0.1.0` (verified live against this repo, SHA `5bce6bf8…`), release-please's manifest-mode "find the previous release" fallback will locate it automatically — no `bootstrap-sha` or `last-release-sha` needed. `Release-As: 1.0.0` on a one-shot empty commit is confirmed, from the source, to take precedence over every other version-derivation path (`buildNewVersion` checks it before the accumulated-commits bump).

The single highest-risk mechanical fact in this phase is confirmed straight from GitHub's own docs: the default `GITHUB_TOKEN` does not trigger downstream workflows on the refs it pushes, but a **GitHub App installation token does** — this is exactly what makes D-02's App-token path work and is *why* projects like engram collapse release-please and the build into one job (this repo cannot do that, per D-01). release-please-action's current major is **v5.0.0** (not v4, which is what most cached examples still show) — a Node-runtime bump only, not a breaking-behavior change relevant here. `actions/create-github-app-token` has quietly moved its required input from `app-id` to `client-id` (`app-id` is deprecated-but-accepted) — a documentation-drift trap worth flagging explicitly for the plan.

The second highest-risk fact is the exact shape of `release.yml`'s `Publish GitHub release` step (line 279) after D-04: release-please creates the GitHub Release automatically when its PR merges (confirmed in `googleapis/release-please`'s own design doc), so `release.yml` must branch on release existence rather than always calling `gh release create`. The safest existence check is `gh release view "$TAG"`'s exit code, not `gh release create || upload` (the latter masks real API failures behind the same fallback path used for "already exists").

Finally: nothing about "release-please really cuts a tag from a merged PR and that tag really fires `release.yml`" can be proven inside this session or by any static check — it requires a live push event against this repo's real Actions configuration. The honest, cheap way to prove it before publishing the real `v1.0.0` is a **throwaway prerelease cycle on a scratch branch** (release-please's own `--target-branch` input), not treating the actual `v1.0.0` cut as the first live test — see Validation Architecture.

**Primary recommendation:** Wire release-please via a GitHub App token exactly as D-02 specifies; seed the manifest to `0.1.0` and rely on the existing `v0.1.0` GitHub Release for automatic SHA discovery (no `bootstrap-sha`); rewrite `release.yml:279` as an existence-checked create-vs-upload branch; and run the entire wiring once against a disposable scratch branch/tag before ever cutting the real `v1.0.0`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Version bump / SemVer decision | release-please (new `release-please.yml`) | — | Parses Conventional Commits on every push to `main`; owns `buildNewVersion` including the one-shot `Release-As:` override (D-06) |
| `CHANGELOG.md` generation | release-please | — | `release-type: go` strategy; no Go source file needed (D-07) |
| Tag creation | release-please (via GitHub App token) | — | Fires on release-PR merge; MUST use the App token so the push event isn't self-suppressed (D-02) |
| GitHub Release creation + notes body | release-please | `release.yml` `assemble` job (upload-only) | release-please owns the Release object and its changelog-derived body; `release.yml` only attaches binaries (D-04) |
| Downstream build trigger | `release.yml` (tag-push trigger, LOCKED) | — | D-01: must stay tag-triggered; the App-token tag push is what makes this fire at all |
| Binary compilation (6-target matrix) | `release.yml` `build` job + `.goreleaser.yaml` (`build --single-target`) | — | Untouched by this phase (D-05) |
| Per-binary cosign signing | `release.yml` `assemble` job | — | Untouched; still the OIDC identity `verify.go` anchors on |
| SBOM generation | `release.yml` `assemble` job (syft) | — | Untouched |
| SLSA provenance | `release.yml` `provenance` job (slsa-github-generator) | — | Untouched |
| Asset upload into the release | `release.yml` `assemble` job | — | New: branches create-vs-`upload --clobber` depending on whether release-please already created the Release (D-04) |
| Commit-message quality gate | `ci.yml` (new PR-title job) | — | Must gate PR **title**, not branch commits (squash-merge model, D-08) — needs `pull_request` event types widened to include `edited` (see Common Pitfalls) |
| Pre-tag cross-platform build sanity (6-target `go list`) | New automated gate in the release-please workflow (gating the release-please job itself) | `docs/RELEASE-PROCEDURES.md` §1 (documentation only, no longer the enforcement point) | No human runs this by hand anymore once release-please owns the tag — must become a CI gate, not just a runbook step (canonical_refs, Folded Todos) |
| Client-side trust verification | `internal/upgrade/verify.go` (unchanged) | — | Everything above exists to keep this file's SAN pattern satisfied |

## Standard Stack

### Core

| Action / Tool | Version (pinned) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `googleapis/release-please-action` | `v5.0.0` — SHA `8b8fd2cc23b2e18957157a9d923d75aa0c6f6ad5` [VERIFIED: GitHub API `gh api repos/googleapis/release-please-action/releases/latest`] | Runs release-please inside GitHub Actions: opens/updates the release PR, and on merge creates the tag + GitHub Release | The official, Google-maintained action; `v5.0.0`'s only breaking change vs `v4.4.1` is a Node 24 runtime bump (verified via `gh api .../releases`), no behavioral change relevant to this phase [VERIFIED: GitHub API release body] |
| `actions/create-github-app-token` | `v3.2.0` — SHA `bcd2ba49218906704ab6c1aa796996da409d3eb1` [VERIFIED: GitHub API `gh api repos/actions/create-github-app-token/releases/latest`] | Mints a short-lived GitHub App installation token inside the workflow, used as release-please's `token:` input (D-02) | Official GitHub action for exactly this purpose; required because App-token-authored pushes trigger downstream workflows and `GITHUB_TOKEN`-authored ones don't [CITED: docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow] |

### Supporting

| Tool | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `rhysd/actionlint` | latest (`go install github.com/rhysd/actionlint/cmd/actionlint@latest`) [CITED: github.com/rhysd/actionlint] | Static checker for GitHub Actions workflow YAML — schema validation, expression type-checking, embedded shellcheck, script-injection detection | Pre-merge/CI gate on the new `release-please.yml` and the modified `release.yml`/`ci.yml`; installable via `go install`, no new node dependency, matching this project's pure-Go CI tooling preference |
| `goreleaser check` | Already-pinned `v2.17.0` (no version change needed) [CITED: goreleaser docs, "Check GoReleaser Configuration"] | Validates `.goreleaser.yaml` syntax/structure without building | Re-run as a config-drift regression check — this phase does not modify `.goreleaser.yaml` (D-05), so this proves that fact rather than testing anything new |
| `release-please` npm CLI (`npx release-please@latest`) | latest [CITED: googleapis/release-please docs/troubleshooting.md] | `--dry-run --debug` local/CI simulation of what the next release PR would contain, without opening a PR or creating anything | The **only** pre-real-release mechanism that exercises release-please's actual version/changelog/manifest-resolution logic against this repo's real commit history — see Validation Architecture |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-written `grep -E` PR-title lint (D-08 discretion, recommended) | `amannn/action-semantic-pull-requests` | Off-the-shelf action is less code to maintain but adds a new pinned third-party dependency for a ~10-line regex check; this repo's established pattern (per `08-CONTEXT.md` "Established Patterns") favors auditable hand-written shell for exact-contract checks. `amannn/action-semantic-pull-requests` has no discoverable GitHub Releases via `gh api` (tag-only releases), making SHA-pin resolution and update-tracking slightly more manual than release-please-action's clean release history. |
| `gh release view "$TAG"` exit-status branch (D-04 discretion, recommended) | Explicit workflow input/output threading `release_created` from the release-please job into `release.yml` | `gh release view` is self-contained inside `release.yml` and works identically for both the release-please path and the manual-`rc`-tag path (D-10) without requiring cross-workflow data plumbing (`release.yml` is triggered by the tag push itself, not invoked with inputs from the release-please job) — the exit-status check is the only option that doesn't need new cross-workflow wiring |
| App-token tag push (D-02, primary/locked) | `workflow_dispatch` fallback at the tag ref (D-03) | Fallback only per explicit decision; widens the LOCKED `release.yml`'s trigger surface and requires a proven-non-vacuous ref-shape guard — costlier to maintain, not the default path |

**Installation:**
```bash
# actionlint (CI tooling, not a go.mod dependency)
go install github.com/rhysd/actionlint/cmd/actionlint@latest

# No go.mod/npm changes required for release-please itself — it runs
# entirely inside the GitHub-hosted Actions runner via the pinned
# googleapis/release-please-action, never as a project dependency.
```

**Version verification:** Both pinned actions above were verified live against the GitHub API (`gh api repos/<org>/<repo>/releases/latest`), not training data — training data's cached "v4" release-please-action examples are one major version behind the real current release.

## Package Legitimacy Audit

This phase installs **no npm/PyPI/crates packages** into the project's own dependency tree (`go.mod` is untouched). The only new external artifacts are two GitHub Actions, verified directly against the GitHub API rather than a package registry:

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `googleapis/release-please-action@v5.0.0` | GitHub Actions (not npm/PyPI) | Org repo, multi-year history, `v5.0.0` published 2026-04-22 [VERIFIED: `gh api`] | N/A (Actions Marketplace, not download-counted) | `github.com/googleapis/release-please-action` (Google's own org) | OK | Approved — pin by full commit SHA per repo convention |
| `actions/create-github-app-token@v3.2.0` | GitHub Actions | Official `actions` org, `v3.2.0` published 2026-05-12 [VERIFIED: `gh api`] | N/A | `github.com/actions/create-github-app-token` (GitHub's own org) | OK | Approved — pin by full commit SHA per repo convention |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** none.

*The `gsd-tools query package-legitimacy check` seam targets npm/PyPI/crates ecosystems; it does not apply to GitHub Actions. The verification substitute used here — resolving each action's latest release and commit SHA directly via `gh api`, and confirming the owning org is the action's canonical publisher (`googleapis`, `actions`) — is the ecosystem-appropriate equivalent and is `[VERIFIED]`, not `[ASSUMED]`.*

## Architecture Patterns

### System Architecture Diagram

```
Conventional-commit PRs
        │
        ▼
  ci.yml (pull_request)
   ├─ test / govulncheck / reproducibility / perf-regression   (unchanged)
   └─ NEW: PR-title conventional-commit lint (D-08)  ───────────► blocks merge on non-conformant title
        │  (squash-merge)
        ▼
      main
        │  push
        ▼
  NEW release-please.yml (push: branches: [main])
   ├─ NEW: pre-tag 6-target `go list -mod=readonly` gate  ──────► fails job, no PR/tag proceeds
   ├─ actions/create-github-app-token  ───────────────────────────► mints App installation token
   └─ googleapis/release-please-action (token: <App token>)
        ├─ no open release PR yet → opens/updates "chore: release X.Y.Z" PR
        │    (Release-As: 1.0.0 footer honored here, one-shot, D-06)
        └─ release PR merged      → creates tag vX.Y.Z (App-token push!)
                                     + creates GitHub Release (body = CHANGELOG excerpt)
                                          │
                                          │  tag push (App-token-authored ⇒ DOES trigger)
                                          ▼
                              release.yml  (UNCHANGED trigger: push.tags v[0-9]*, LOCKED D-01)
                                ├─ build job (6-target matrix)         — unchanged (D-05)
                                ├─ assemble job
                                │    ├─ sign each binary (cosign)      — unchanged
                                │    ├─ per-binary SBOM (syft)         — unchanged
                                │    └─ NEW: Publish GitHub release step (D-04)
                                │         if `gh release view "$TAG"` succeeds:
                                │             release-please already made it → `gh release upload --clobber`
                                │         else (manual rc tag, D-10):
                                │             `gh release create … --generate-notes` (unchanged)
                                └─ provenance job (SLSA)                — unchanged
                                          │
                                          ▼
                        Published, signed v1.0.0 artifacts
                                          │
                                          ▼
                    codegraph upgrade (already-shipped binaries)
                    verifies cosign SAN against releaseWorkflowRefPattern
                    (internal/upgrade/verify.go — UNCHANGED, D-01)
```

### Recommended Project Structure

```
.github/workflows/
├── ci.yml                     # existing; gains PR-title lint job (D-08)
├── release.yml                # existing; ONE shell step changed (D-04); trigger/name/identity LOCKED
└── release-please.yml         # NEW: push:branches:[main] → App token → release-please-action
release-please-config.json     # NEW: release-type=go, packages={".":{}}
.release-please-manifest.json  # NEW: {".": "0.1.0"}
docs/
├── RELEASE-PROCEDURES.md      # §3/§4/§7 rewritten, new App-prerequisite section added
└── RELEASE.md                 # confirm unchanged (verification commands are unaffected)
```

### Pattern 1: GitHub App token as the tag-push identity

**What:** Mint a short-lived installation token from a GitHub App and pass it as release-please-action's `token:` input, instead of the default `GITHUB_TOKEN`.
**When to use:** Whenever an automated actor's action (here: release-please's tag push) needs to trigger a *separate* downstream workflow (`release.yml`) via a standard `on:` trigger.
**Why it's required, not optional:** GitHub's own docs are unambiguous: *"When you use the repository's `GITHUB_TOKEN` to perform tasks, events triggered by the `GITHUB_TOKEN` will not create a new workflow run"* — with narrow exceptions that do not include a tag push. *"If you do want to trigger a workflow from within a workflow run, you can use a GitHub App installation access token or a personal access token instead of `GITHUB_TOKEN`."* [CITED: docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow]
**Example:**
```yaml
# Source: actions/create-github-app-token README + googleapis/release-please-action README (Context7)
name: release-please
on:
  push:
    branches: [main]
permissions:
  contents: write
  issues: write
  pull-requests: write
jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - name: Mint GitHub App installation token
        id: app-token
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
      - uses: googleapis/release-please-action@8b8fd2cc23b2e18957157a9d923d75aa0c6f6ad5 # v5.0.0
        id: release
        with:
          token: ${{ steps.app-token.outputs.token }}
          release-type: go
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```
**Pitfall flagged for the plan:** `create-github-app-token`'s *current* required input is `client-id` (App's Client ID), with `app-id` (the numeric App ID) kept only as a deprecated back-compat alias. Both work today, but `app-id` carries a `deprecationMessage` in the action's own metadata [VERIFIED: Context7 excerpt of `action.yml`]. D-02's own text says "`APP_ID`, `APP_PRIVATE_KEY`" secrets — keep the secret **names** if desired for readability, but confirm at implementation time whether to wire them to the `app-id` (deprecated) or `client-id` (current) input; either works, `client-id` is the forward-compatible choice.

### Pattern 2: Manifest seeding to skip the bootstrap PR

**What:** Manually write `.release-please-manifest.json` with the real current version instead of letting `release-please bootstrap` open a bootstrap PR.
**When to use:** Any repo (like this one) that already has real tags/releases and wants release-please to start from that point, not from `0.0.0`.
**Example:**
```json
// Source: googleapis/release-please docs/manifest-releaser.md (Context7)
{
  ".": "0.1.0"
}
```
```json
// release-please-config.json — Source: googleapis/release-please docs (Context7),
// shape confirmed for a single (non-monorepo) root package
{
  "release-type": "go",
  "include-component-in-tag": false,
  "packages": {
    ".": {}
  }
}
```
**Why no `bootstrap-sha` is needed here [VERIFIED against live repo state]:** In manifest mode, release-please "determines the previous release by reading the latest version from `.release-please-manifest.json`... [and] attempts to find the commit SHA for that release by examining recent releases and falling back to the expected tag name" [CITED: googleapis/release-please docs/troubleshooting.md]. This repo already has a real **GitHub Release** object for `v0.1.0` at commit `5bce6bf8c690d9b5afcc05f0b8fcceab3147555f` (confirmed live via `gh release list` and `gh api .../git/refs/tags/v0.1.0`), so release-please's built-in discovery will find it directly — `bootstrap-sha`/`last-release-sha` exist for repos *without* a matching Release/tag to fall back on, which is not this repo's situation.
**`include-component-in-tag: false` recommendation:** Source confirms `getComponent()` returns `''` unconditionally when this flag is `false`, guaranteeing the tag stays exactly `v<version>` (matching the existing `v0.1.0`/`v0.0.0-rc.3` shape) regardless of any default-component-name derivation for the `go` strategy [VERIFIED: Context7 excerpt of `src/strategies/base.ts`]. Docs also note that for genuinely single-package repos, "a component configuration is not needed, and versions will be simple (`v1.2.3`)" even without this flag — but setting it explicitly removes the ambiguity rather than relying on an implicit default, which is the belt-and-suspenders choice this repo's conventions favor.

### Pattern 3: `Release-As:` footer precedence (the v1.0.0 forcing mechanism)

**What:** An empty commit with a `Release-As: 1.0.0` trailer, forcing the next release-please PR to compute exactly `1.0.0` regardless of the accumulated commit history's own bump logic.
**Example:**
```sh
# Source: googleapis/release-please README (Context7)
git commit --allow-empty -m "chore: release 1.0.0" -m "Release-As: 1.0.0"
```
**Verified precedence, straight from the source [VERIFIED: Context7 excerpt of `src/strategies/base.ts` `buildNewVersion`]:**
```typescript
protected async buildNewVersion(conventionalCommits, latestRelease) {
  if (this.releaseAs) {                       // (1) config-level release-as — NOT used here (sticky)
    return Version.parse(this.releaseAs);
  }
  const releaseAsCommit = conventionalCommits.find(c =>
    c.notes.find(note => note.title === 'RELEASE AS'));
  if (releaseAsCommit) {                       // (2) the Release-As: footer — THIS is D-06's mechanism
    return Version.parse(note.text);
  }
  if (latestRelease) {                         // (3) normal versioning strategy bump
    return await this.versioningStrategy.bump(latestRelease.tag.version, conventionalCommits);
  }
  return this.initialReleaseVersion();          // (4) fallback for a genuinely first-ever release
}
```
This confirms D-06's design exactly: the footer (2) is checked before the normal bump path (3), so it deterministically overrides whatever `0.2.0`-from-`0.1.0` the accumulated `feat:` commits would otherwise produce — without needing the sticky config-level `release-as` (1), which D-06 explicitly and correctly rejects.

### Pattern 4: release-please owns the Release; `release.yml` only uploads (D-04)

**What:** Replace `release.yml:279`'s unconditional `gh release create … --generate-notes` with an existence-checked branch.
**Confirmed premise [CITED: googleapis/release-please docs/design.md]:** *"`release-please` automatically creates a GitHub release after a release pull request is merged into the release branch."*
**Recommended shell shape** (extends the existing env-indirection convention already used at this exact line):
```sh
# Source: this project's release.yml convention (env-indirection, set -euo pipefail)
# + gh CLI docs / community-verified existence-check idiom
set -euo pipefail
PRERELEASE_FLAG=""
case "$TAG" in
*-*) PRERELEASE_FLAG="--prerelease" ;;
esac

if gh release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
  # release-please already created this release (body/prerelease flag are
  # release-please's, do NOT pass --generate-notes or re-set --prerelease here)
  gh release upload "$TAG" --repo "$REPO" --clobber codegraph_*
else
  # manual rc tag path (D-10) — unchanged from today
  # shellcheck disable=SC2086
  gh release create "$TAG" --repo "$REPO" --title "$TAG" --generate-notes \
    $PRERELEASE_FLAG codegraph_*
fi
```
**Why `gh release view`'s exit code, not `||` chaining `create`:** a WebSearch-sourced GitHub-issue thread on `gh release upload --clobber` explicitly recommends the `if ! gh release view … ; then` idiom over an `||`-chained fallback, because `||` conflates "release doesn't exist" with any other transient `gh`/API failure (rate limit, network) — both would silently fall through to the same branch. [LOW confidence — community source, not official gh docs; the underlying exit-code semantics of `gh release view` are stable CLI behavior and low-risk to rely on, but re-confirm against the pinned `gh` CLI version in the runner image at implementation time.]
**`--clobber` semantics:** overwrites/replaces existing same-named assets rather than erroring — this is what makes `assemble` idempotent across CI re-runs against an already-published release, directly satisfying what roadmap criterion 3's `replace_existing_artifacts: true` would have meant under a `goreleaser release` model (D-05's accepted substitute).

### Pattern 5: Automated pre-tag 6-target sanity gate

**What:** `docs/RELEASE-PROCEDURES.md` §1's manual maintainer sweep (`for pair in linux/amd64 linux/arm64 … ; do GOOS=… GOARCH=… go list -mod=readonly ./... ; done`) has no human left to run it once release-please owns the tag. Port it into a blocking job in the new `release-please.yml`, gating the `release-please` job itself via `needs:`.
```yaml
# Illustrative shape — this project's own §1 script, wired as a CI job precondition
jobs:
  pretag-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@... # pinned per repo convention
      - name: 6-target go list -mod=readonly sanity sweep
        run: |
          set -euo pipefail
          for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
            GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null
          done
  release-please:
    needs: pretag-gate
    runs-on: ubuntu-latest
    steps: [...]
```
This runs on every push to `main` (not just at release time) — slightly broader than the original manual trigger point, but it means release-please can never process a `main` state that fails this sweep, closing the exact gap D-09/RESEARCH notes called out ("no human runs it manually, it should become an automated gate").

### Anti-Patterns to Avoid

- **Using `release-as` in `release-please-config.json` for the v1.0.0 forcing.** Sticky — pins every subsequent release to `1.0.0` until manually removed (explicitly rejected by D-06; confirmed as config-level precedence tier (1) in Pattern 3 above, above even the one-shot footer).
- **`gh release create … --generate-notes` unconditionally after this phase ships.** Will hard-fail once release-please has created the Release first (the API rejects creating a release for a tag that already has one), or — if someone "fixes" it with `--force`-style flags instead of branching — silently clobbers release-please's changelog-derived body with generic auto-notes. This is exactly the D-04 collision the roadmap calls "the highest-risk detail."
- **Collapsing release-please + build into one workflow (engram's model).** Rejected by D-01: the resulting tag/signature would carry `@refs/heads/main` instead of `@refs/tags/v…`, and every already-shipped binary's `verify.go` would reject it — silently, with no actionable error for the user.
- **Treating `workflow_dispatch` (D-03) as interchangeable with the App-token path.** It's the fallback only, and only SAN-safe when dispatched with `--ref <tag>`; dispatching against a branch produces an unverifiable artifact. If ever implemented, it needs its own non-vacuous rejecting-input test (Phase-8 lesson, canonical_refs).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Conventional-commit parsing → SemVer bump → CHANGELOG diff | A custom commit-parsing/version-bump script | `release-type: go` in release-please | release-please's `default` versioning strategy, breaking-change detection, and `DEFAULT_CHANGELOG_SECTIONS` (hiding `chore`/`docs`/`style`/`refactor`/`test`/`build`/`ci`, showing `feat`/`fix`/`perf`/`revert`) are exactly what D-09's 160-`feat`/`fix`/`perf`-out-of-477-commits changelog depends on — reimplementing this is high-effort and easy to get subtly wrong (e.g. `revert` commits fall through to a default patch bump, not an explicit rule; missing this reimplementing from scratch is an easy trap) [VERIFIED: Context7 excerpt of `src/versioning-strategies/default.ts`] |
| Minting a scoped, short-lived App token in a workflow | A hand-rolled JWT-signing + GitHub API token-exchange step | `actions/create-github-app-token` | This is precisely the mechanism GitHub ships for this use case; reimplementing JWT signing for App auth in shell/Python inside a workflow reintroduces a private-key-handling security surface for zero benefit |
| GitHub Actions workflow YAML correctness (schema, expression types, injection safety) | Manual review only | `actionlint` | Given this repo already treats "guard present but never fires" as a documented past failure mode (Phase-8 CR-01/WR-02), a static linter that catches malformed `on:`/`permissions:`/`${{ }}` injection risks *before* merge is strictly better than eyeballing YAML — especially for a phase that edits three workflow files |

**Key insight:** every "Don't Hand-Roll" here maps to a piece of this phase's LOCKED contract (D-01) or highest-risk edit (D-04) — the entire point of using release-please + the standard App-token pattern is to avoid the class of subtle bugs (wrong SemVer bump, wrong tag ref shape, leaked/mishandled App credentials) that a from-scratch implementation would risk reintroducing into a codebase that has already been bitten twice by doc-vs-code drift (per D-07's own rationale).

## Common Pitfalls

### Pitfall 1: PR-title lint never re-fires on a title edit

**What goes wrong:** D-08's PR-title conventional-commit check is added as a job inside `ci.yml`, which currently triggers on `pull_request:` with GitHub's **default** event types (`opened`, `synchronize`, `reopened`). If a contributor opens a PR with a bad title, gets flagged, then edits *only the title* (no new commits), the workflow does **not** re-run — `edited` is not in the default type set — so the check appears to still be failing (stale run) or, worse, is never re-evaluated and the stale "pass" from before the title change is what merge protection sees.
**Why it happens:** GitHub Actions' `pull_request` trigger requires explicit `types:` to include title-edit-driven re-evaluation; this is a well-known, easy-to-miss gap for exactly this class of check (PR-title/label linters).
**How to avoid:** Add `types: [opened, edited, synchronize, reopened]` — either on `ci.yml`'s single shared `pull_request:` trigger (which then re-runs the *entire* CI suite, including the heavier test/govulncheck/reproducibility/perf jobs, on every title edit — a cost, not a correctness bug) or, cleaner, split the PR-title lint into its own lightweight sibling workflow file with its own narrowly-scoped `pull_request: types: [...]` trigger, decoupled from `ci.yml`'s heavy jobs. Flag this as a decision point for the plan — CONTEXT.md's D-08 says "in `ci.yml`" but doesn't resolve the shared-trigger-widening tradeoff; a separate file may be the better implementation even though the decision names `ci.yml`.
**Warning signs:** A PR title is edited after initial CI has already run green, and no new check run appears — the merge button doesn't reflect the corrected title.

### Pitfall 2: The GitHub App's own installation permissions are a different scope than the workflow `permissions:` block

**What goes wrong:** `release-please-action`'s README states the **workflow-level `permissions:` block** needs `contents: write`, `issues: write`, `pull-requests: write` — but that statement describes the block governing the default `GITHUB_TOKEN`. Once D-02 swaps in an App token via the `token:` input, the API calls release-please-action makes are authorized by the **App's own installation permissions** (configured at App-creation time in GitHub's UI), not by the workflow's `permissions:` block. Conflating the two is an easy mistake: setting the workflow-level `permissions:` correctly but under-scoping the App itself (e.g. forgetting `Issues: Read & write`, needed because release-please labels its PRs `autorelease: pending`/`autorelease: tagged`, and PR labels are governed by the Issues API scope in GitHub's App permission model) produces a runtime 403 on first real use, not a config-time error.
**Why it happens:** GitHub Actions has two independent permission systems (workflow-declared `permissions:` for the ephemeral `GITHUB_TOKEN`, and an App's own installation-time permission grant) that look similar but are configured in completely different places.
**How to avoid:** When creating the App (maintainer-manual step, D-02), grant at minimum: **Contents: Read & write**, **Pull requests: Read & write**, **Issues: Read & write**. `[ASSUMED — extrapolated from release-please-action's stated workflow-permission requirements, not independently confirmed against the App-permission-specific docs this session; verify the App's exact required scopes against `googleapis/release-please-action`'s "Authenticating with GitHub App"-style docs if it publishes one, or accept the cheap failure mode: an under-scoped App produces an immediately visible, easily-diagnosed 403 on the first real run, not a silent one.]`
**Warning signs:** release-please-action's job fails with a 403/permission error the first time it tries to label a PR or create a release, despite the workflow's own `permissions:` block looking correct.

### Pitfall 3: `release.yml`'s header comment asserts a stronger guarantee than what D-03 would leave true

**What goes wrong:** `release.yml`'s header currently says (line 8-9): *"MUST trigger ONLY on tag pushes matching `v[0-9]*`"*. If D-03's `workflow_dispatch` fallback is ever implemented, this comment becomes false (the file now has two trigger types) and nothing forces it to be updated — a stale doc-vs-code claim, the exact failure class this project has been bitten by twice already (per D-07's own rationale).
**Why it happens:** Comments don't get exercised by tests; only the actual `on:` block does.
**How to avoid:** If D-03 is implemented, update the header comment in the same commit, AND add the non-vacuous ref-shape guard test D-03 already mandates — the test proves the *runtime* behavior; the comment update keeps the *documentation* honest. Since D-03 is fallback-only and not this phase's primary path, the safest default is to **not** implement it unless the App path (D-02) is confirmed unavailable — leaving the comment true by construction.
**Warning signs:** `grep -c "on:" release.yml` shows more trigger blocks than the header comment describes.

### Pitfall 4: A disposable test tag resolving as `codegraph upgrade`'s "latest"

**What goes wrong:** If the live end-to-end proof recommended in Validation Architecture (a throwaway prerelease cycle via release-please's `target-branch` input) produces a tag that is **not** prerelease-shaped (no `-` suffix), it becomes indistinguishable from a real stable release to `internal/upgrade`'s `resolveLatestVersion` — a user running `codegraph upgrade` during the test window could be upgraded to a throwaway test build.
**Why it happens:** `release.yml`'s `--prerelease` flag is derived purely from the tag string shape (`*-*` → prerelease); a test tag computed by release-please's normal versioning strategy against a scratch branch would just be the next real SemVer bump (e.g. `0.2.0`), not obviously a test artifact.
**How to avoid:** Force any throwaway/live-proof tag through a `Release-As:` footer with an explicit prerelease-shaped version string (e.g. `0.0.0-rc.99-releasetest`), or push the disposable test as a raw `git tag`/`gh release create` with a `-`-suffixed name rather than letting release-please's own bump logic pick it. Verify `Version.parse` in the pinned release-please version accepts a prerelease-suffixed string before relying on this — not independently confirmed this session `[ASSUMED]`. Delete the tag and Release immediately after the test per `docs/RELEASE-PROCEDURES.md` §7's existing rollback procedure.
**Warning signs:** A tag appears in `gh release list` without a "Pre-release" label that wasn't an intentional stable cut.

## Code Examples

### Full `release-please.yml` (illustrative composite of Patterns 1, 2, 5)
```yaml
# Source: composite of googleapis/release-please-action README,
# actions/create-github-app-token README (both Context7), and this
# project's own release.yml conventions (SHA pins, env-indirection)
name: release-please

on:
  push:
    branches: [main]

permissions:
  contents: write
  issues: write
  pull-requests: write

jobs:
  pretag-gate:
    name: pre-tag 6-target go list sanity sweep
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
      - run: |
          set -euo pipefail
          for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
            GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null
          done

  release-please:
    name: release-please
    needs: pretag-gate
    runs-on: ubuntu-latest
    steps:
      - name: Mint GitHub App installation token
        id: app-token
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}

      - uses: googleapis/release-please-action@8b8fd2cc23b2e18957157a9d923d75aa0c6f6ad5 # v5.0.0
        with:
          token: ${{ steps.app-token.outputs.token }}
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

### `release-please-config.json`
```json
// Source: googleapis/release-please docs/manifest-releaser.md (Context7)
{
  "release-type": "go",
  "include-component-in-tag": false,
  "packages": {
    ".": {}
  }
}
```

### `.release-please-manifest.json`
```json
// Source: googleapis/release-please docs/manifest-releaser.md (Context7),
// seeded to this repo's real current tag per D-06
{
  ".": "0.1.0"
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `actions/create-github-app-token` required `app_id`/`private_key` (snake_case) | `app-id`/`private-key` (kebab-case) required; `app-id` itself now deprecated in favor of `client-id` | `v2.0.0` removed the snake_case aliases; `client-id` deprecation of `app-id` is the current (`v3.x`) state [VERIFIED: Context7 excerpt of `action.yml`] | Cached/training-data examples using `app_id` will fail outright; examples using `app-id` still work but carry a deprecation warning |
| `release-please-action@v4` | `release-please-action@v5.0.0` | 2026-04-22 [VERIFIED: `gh api`] | Node 24 runtime bump only — no input/output/behavior change relevant to this phase, but any cached "v4" example (including some still-current Context7 snippets) is one major version stale |

**Deprecated/outdated:**
- `release-please bootstrap` CLI/PR flow: not needed here — manually seeding `.release-please-manifest.json` (Pattern 2) achieves the same effect for a repo with existing tags, without opening a throwaway bootstrap PR.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The GitHub App needs `Contents: Read & write`, `Pull requests: Read & write`, `Issues: Read & write` installation permissions (extrapolated from release-please-action's stated *workflow-token* permission list, not independently confirmed against App-specific permission docs) | Common Pitfalls #2 | Under-scoping produces an immediately visible 403 on first real run — cheap to diagnose and fix, not a silent failure, but costs one maintainer round-trip during App setup |
| A2 | Release-please's `Version.parse` accepts a prerelease-suffixed string (e.g. `0.0.0-rc.99-releasetest`) for a `Release-As:` footer, making a safely-throwaway live-proof tag possible without hand-pushing a raw tag | Common Pitfalls #4, Validation Architecture | If wrong, the recommended scratch-branch live-proof method needs to fall back to a hand-pushed `-`-suffixed tag instead of a `Release-As:`-forced one; low risk, alternate path already documented |
| A3 | `gh release view "$TAG"`'s exit-status idiom (vs. `||`-chained `create`) is the more robust existence check | Pattern 4 | Sourced from a community WebSearch result (LOW tier), not official `gh` docs; if the exit-code semantics differ from expectation, D-04's create-vs-upload branch could misfire — mitigated by this being exactly the kind of logic the Validation Architecture recommends unit-testing with a stubbed `gh` before relying on it in production |

**If this table is empty:** N/A — see entries above; all three are flagged precisely so the plan can either verify them cheaply (A1, A3 via a stubbed test; A2 via checking the pinned release-please version's semver parser) or accept the documented low-risk fallback.

## Open Questions

1. **Does this specific repo's GitHub Actions settings allow the App-token-authored tag push to trigger `release.yml` without any additional org/repo-level "Allow Actions to create workflow runs" toggle?**
   - What we know: GitHub's own docs confirm App-token pushes trigger workflows *in general* [CITED]; this repo currently has default Actions settings (no rulesets, no branch protection, confirmed live via `gh api`).
   - What's unclear: Whether any repo-level Actions permission setting (e.g. "Allow GitHub Actions to create and approve pull requests", explicitly called out in release-please-action's own README as sometimes needing to be enabled for org-hosted repos) is already correctly configured for `seanb4t/codegraph-go`.
   - Recommendation: Check `gh api repos/seanb4t/codegraph-go/actions/permissions` (workflow permissions + PR-creation toggle) during App setup, as a cheap pre-flight before relying on it in the live-proof run.

2. **Should the pre-tag 6-target gate (Pattern 5) block `release-please.yml` on every push to `main`, or only when a release PR is about to merge?**
   - What we know: The phase's own notes say this must become "an automated gate... now that no human runs it by hand" (canonical_refs).
   - What's unclear: Gating every push to `main` (my recommendation) runs the sweep more often than the original manual trigger point (only before tagging), costing a small amount of extra CI time on non-release pushes, but with zero risk of a bad state slipping into a release PR unnoticed.
   - Recommendation: Gate every push to `main` (via `needs:`) — the cost is low (a handful of `go list` invocations) and it strictly dominates a "gate right before tag" design that would require detecting "is this push about to trigger a release" in advance, which release-please itself doesn't expose as a pre-check hook.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `gh` CLI | `release.yml` D-04 edit, live-proof verification, manual App/secret setup | ✓ (used throughout this research session) | — | — |
| GitHub App creation UI access | D-02 (blocking maintainer-manual prerequisite) | Unknown — requires org/repo admin access not exercisable from this session | — | D-03 `workflow_dispatch` fallback, documented but not primary |
| Two new repo secrets (`APP_ID`/`APP_PRIVATE_KEY` or equivalent) | D-02 | ✗ — confirmed live: `gh secret list` returns empty for this repo | — | None — this is the phase's one hard maintainer-manual blocking checkpoint |
| `actionlint` | Recommended pre-merge CI check (new) | Not yet installed in this environment; installable via `go install` in CI | latest | Skip and rely on `goreleaser check` + manual review (weaker) |

**Missing dependencies with no fallback:**
- The GitHub App itself (D-02's blocking prerequisite) — no automation in this repo can create a GitHub App; this must be a human action before the release-please workflow can function via its primary path.

**Missing dependencies with fallback:**
- `actionlint` — optional but recommended; its absence doesn't block the phase, just weakens the pre-merge YAML-correctness guarantee.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (existing `internal/upgrade` package convention) + shell/bats-style scripts for CI-logic-only pieces + `actionlint`/`goreleaser check` as static gates |
| Config file | none new — extends `internal/upgrade/verify_test.go`'s existing pattern |
| Quick run command | `go test ./internal/upgrade/...` (existing) |
| Full suite command | `go test ./...` + `actionlint` + `goreleaser check` + a new workflow-shape-drift test |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REL-02 (half A: automation) | `release-please-config.json`/`.release-please-manifest.json` are well-formed and resolve to the expected baseline (`v0.1.0` found via existing GitHub Release, no bootstrap PR needed) | integration (dry-run) | `npx -y release-please@latest release-pr --dry-run --debug --token=$GITHUB_TOKEN --repo-url=seanb4t/codegraph-go --config-file=release-please-config.json --manifest-file=.release-please-manifest.json` | ❌ Wave 0 — no such check exists today; this is a new, run-locally-or-in-CI verification step |
| REL-02 (half A: automation) | `release.yml`'s trigger shape (filename, `on.push.tags` pattern) still matches `releaseWorkflowRefPattern` after every edit in this phase | unit (non-vacuous drift guard) | `go test ./internal/upgrade/... -run TestReleaseWorkflowFileMatchesPattern` (NEW) | ❌ Wave 0 — must be written; parses the literal YAML `name:`/`on.push.tags` and asserts it against the compiled regex, so it fails immediately if trigger/filename ever drifts |
| REL-02 (half A: automation) | `release.yml`'s D-04 create-vs-upload branch calls `gh release create` when no release exists and `gh release upload --clobber` when one does | unit (stubbed `gh`) | A small bats/shell test stubbing `gh` on `PATH` and asserting the correct subcommand fires for both branches (NEW) | ❌ Wave 0 — the single highest-risk edit in the phase; must be proven both ways, not just "looks right" |
| REL-02 (half A: automation) | PR-title conventional-commit lint rejects a non-conformant title and accepts a conformant one, and re-fires on a title-only edit | unit + manual verification | The lint's own regex tested directly (`grep -E` against a table of valid/invalid sample titles); trigger-type coverage (Pitfall 1) verified by manual PR-title-edit smoke test | ❌ Wave 0 |
| REL-02 (half A: automation) | New/modified workflow YAML is schema-valid, has no expression-injection risk, and its `permissions:` blocks are as intended | static | `actionlint .github/workflows/release-please.yml .github/workflows/release.yml .github/workflows/ci.yml` | ❌ Wave 0 — tool not yet installed |
| REL-02 (half A: automation) | `.goreleaser.yaml` remains valid and unchanged (D-05 — no regression) | static | `goreleaser check` | ✓ (tool already pinned in repo, command just needs re-running as a gate) |
| REL-02 (half B: signed-artifact trust) | `releaseWorkflowRefPattern` still accepts a real `release.yml@refs/tags/v…` identity and rejects a non-release-workflow one | unit (existing) | `go test ./internal/upgrade/... -run TestReleaseWorkflowRefPattern` | ✓ already exists (`verify_test.go` lines 118-136) — unaffected by this phase, re-run to confirm no regression |
| REL-02 (half B: signed-artifact trust) | The actual OIDC cert SAN produced by a release-please-triggered `release.yml` run matches the pattern | **manual/live only** | `cosign verify-blob ...` per `docs/RELEASE-PROCEDURES.md` §6, against artifacts from a real (even if disposable/prerelease-shaped) tag push | N/A — cannot be proven statically; see below |

### Sampling Rate
- **Per task commit:** `go test ./internal/upgrade/...` (existing + new drift-guard test) + `actionlint` on any touched workflow file
- **Per wave merge:** full `go test ./...` + `actionlint .github/workflows/*.yml` + `goreleaser check`
- **Phase gate:** all of the above green, PLUS the live scratch-branch proof described below, before the real `v1.0.0` tag is ever cut

### Wave 0 Gaps
- [ ] `internal/upgrade/verify_release_e2e_test.go` (or a new sibling file) — `TestReleaseWorkflowFileMatchesPattern`, parsing `release.yml`'s literal `name:`/`on.push.tags` and asserting against `releaseWorkflowRefPattern` — covers REL-02 half B's static half
- [ ] A shell/bats test for the D-04 create-vs-upload branch logic, with a stubbed `gh` — covers REL-02's highest-risk single edit
- [ ] `actionlint` install step (CI or local `go install`) — no config file needed, defaults are sufficient
- [ ] PR-title lint regex + its own small table-driven test (valid/invalid sample titles) — covers D-08

### What CAN be proven before a real release, and what CANNOT (the honest split)

**Verifiable pre-release (mechanical, no live release needed):**
- Config/manifest shape and resolved baseline version — via `release-please --dry-run` against real repo history (read-only GitHub API calls, no write).
- `release.yml`'s trigger/filename never drifted from what `verify.go` expects — a Go test, not a comment.
- The D-04 create-vs-upload branch logic in isolation — a stubbed-`gh` shell test.
- Workflow YAML correctness/injection-safety — `actionlint`.
- `.goreleaser.yaml` config validity (unaffected by this phase, D-05) — `goreleaser check`.
- PR-title lint's own regex correctness — direct unit test against sample titles.

**Only observable on a real (even if disposable) release event — cannot be mechanically proven:**
- That release-please's merged-PR tag push, authenticated by the App token, actually fires `release.yml` in *this repo's* real Actions configuration (Open Question 1 — org/repo Actions settings can only be confirmed by a live event, not read purely from `gh api repos/.../actions/permissions` with full confidence that nothing else intervenes).
- That the App's installation permissions (Assumption A1) are sufficient — a real run either succeeds or produces an immediately diagnosable 403.
- That the resulting cosign OIDC certificate's SAN, as *actually issued by GitHub's Sigstore-federated identity for that specific run*, matches the pattern — the static test above proves the workflow *file* didn't drift, not that GitHub's runtime SAN construction behaves as documented for an App-token-triggered run specifically (vs. a human-triggered one).
- That an already-shipped `v0.1.0` binary's `codegraph upgrade` genuinely succeeds against a release-please-cut `v1.0.0` — true end-to-end proof needs a real prior binary and a real new release.

**Recommended live-proof path (the cheapest honest way to close the loop — avoids repeating Phase 8's "the only test was the real thing" mistake):** Run the entire pipeline once against a **disposable scratch branch**, using release-please-action's `target-branch` input, before ever cutting the real `v1.0.0`:
1. Create a short-lived branch off `main` with one trivial `fix:` commit.
2. Run `release-please.yml`'s logic against that branch (either by temporarily pointing a manual `workflow_dispatch` test run's `target-branch` input at it, or via the CLI's own `--target-branch` flag locally) to produce a real release PR.
3. Merge that PR — release-please creates a **real** tag (force it prerelease-shaped per Common Pitfalls #4/Assumption A2, so it's never mistaken for "latest" by `codegraph upgrade`) using the App token.
4. Confirm the tag push actually fires `release.yml` (closes Open Question 1 and the App-permission risk for real).
5. Run `cosign verify-blob`/`slsa-verifier` against the resulting disposable artifacts per `docs/RELEASE-PROCEDURES.md` §6 — this is the one live signal that genuinely proves REL-02 half B end-to-end.
6. Delete the disposable tag/release/branch per `docs/RELEASE-PROCEDURES.md` §7's existing rollback procedure.
7. Only then cut the real, permanent `v1.0.0` via the same now-proven path.

This makes the real `v1.0.0` the *second* time the pipeline has run end-to-end, not the first — directly addressing the phase's own stated goal of not repeating "REL-02 unverifiable, blocked Phase 8's UAT twice."

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | GitHub App installation token (short-lived, scoped) replacing/augmenting `GITHUB_TOKEN`; private key stored only as a repo secret, never in workflow YAML |
| V3 Session Management | no | N/A — no user sessions in this phase's scope |
| V4 Access Control | yes | App installation permissions scoped to exactly Contents/PRs/Issues write (Pitfall 2); workflow `permissions:` blocks kept minimal (`contents: read` default on `release.yml`, explicit elevated blocks only where needed, matching existing repo convention) |
| V5 Input Validation | yes | `$TAG`/`$REPO` passed via `env:` indirection, never interpolated directly into `run:` shell (existing repo convention, extended to the new `release-please.yml` and the modified `assemble` step) — this is the project's own documented mitigation against a crafted-ref shell-injection class of bug |
| V6 Cryptography | yes (unchanged) | cosign keyless signing (unchanged by this phase) — the entire point of D-01's LOCKED contract is that this phase must not weaken it |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Workflow-YAML script injection via an untrusted `${{ }}` expression interpolated into `run:` | Tampering | Env-indirection (`env: TAG: ${{ github.ref_name }}` then reference `$TAG`), already this project's convention — extend it to every new `run:` step touching `$TAG`/`$REPO` |
| Over-scoped GitHub App installation permissions (broader than Contents/PRs/Issues write) becoming a lateral-movement vector if the App's private key ever leaks | Elevation of Privilege | Scope the App to the minimum needed (Pitfall 2); rotate `APP_PRIVATE_KEY` if ever exposed; App tokens are short-lived by design, limiting a leaked *token's* blast radius even if the *key* isn't immediately rotated |
| A weaker-trust-boundary workflow's signature being accepted by `codegraph upgrade` (the exact bug WR-08's existing anchored-regex test guards against) | Spoofing | Already mitigated — `releaseWorkflowRefPattern` is a full-match anchored regex (D-01, unchanged); this phase's obligation is only to not reintroduce an unanchored/looser pattern, which nothing in this phase's design proposes |
| A disposable live-proof tag (Validation Architecture) being mistaken for "latest" by a real user's `codegraph upgrade` mid-test | Tampering (of user trust, not cryptographic) | Force the test tag prerelease-shaped (Common Pitfalls #4); delete it promptly per the existing §7 rollback procedure |

## Sources

### Primary (HIGH confidence)
None — no findings this session reached the HIGH tier (that tier is reserved for provider results independently cross-verified per the `classify-confidence --verified` seam; this phase's Context7 findings are MEDIUM, and live `gh api` checks against this repo's own state are the closest to independently-verified ground truth, called out inline as `[VERIFIED: ...]`).

### Secondary (MEDIUM confidence)
- `/googleapis/release-please-action` (Context7) — action inputs, permissions, App-token wiring, outputs, v5.0.0 breaking-change note
- `/googleapis/release-please` (Context7) — `release-type: go`, manifest bootstrapping/seeding, `bootstrap-sha`/`last-release-sha`, `Release-As:` footer precedence (`buildNewVersion` source), automatic GitHub Release creation (`docs/design.md`), default versioning strategy and changelog sections
- `/actions/create-github-app-token` (Context7) — required inputs (`client-id`/`app-id`), permission-* inputs, v2.0.0/v3.x deprecation history
- `/goreleaser/goreleaser` (Context7) — `goreleaser check` command, unaffected by this phase
- docs.github.com "Trigger a workflow" (WebFetch, official GitHub docs) — `GITHUB_TOKEN` vs. App-token/PAT workflow-triggering behavior, the load-bearing fact behind D-02
- `gh api` calls against `seanb4t/codegraph-go` and `googleapis/release-please-action`/`actions/create-github-app-token` (live, VERIFIED tier per this repo's own state: no secrets, no rulesets, no branch protection, real `v0.1.0`/`v0.0.0-rc.3` GitHub Releases with known commit SHAs, current action release versions/SHAs)

### Tertiary (LOW confidence)
- WebSearch: `gh release upload --clobber` / `gh release view` existence-check idiom (community sources, not official `gh` CLI docs — flagged as Assumption A3)
- WebSearch: `actionlint` installation/usage summary (project README-derived, not independently cross-verified against a second source)

## Metadata

**Confidence breakdown:**
- Standard stack (release-please-action/create-github-app-token versions, SHAs, input shapes): MEDIUM — Context7-sourced, cross-checked live against `gh api` for version/SHA accuracy specifically
- Architecture (App-token wiring, manifest seeding, Release-As precedence, D-04 create-vs-upload branch): MEDIUM-HIGH — the mechanisms are confirmed directly from release-please's own source excerpts (Context7), and the repo-specific facts they depend on (existing `v0.1.0` Release, zero secrets, no branch protection) are VERIFIED live
- Pitfalls (PR-title trigger-type gap, App-permission-scope ambiguity, header-comment drift, disposable-tag "latest" risk): MEDIUM — derived from correct first-principles reasoning about documented GitHub Actions mechanics, not independently reproduced in a live test this session
- Validation Architecture's live-only/pre-release split: HIGH confidence in the *categorization* (what is and isn't provable without a real release event is a structural fact about GitHub Actions + Sigstore, not a researched claim) — the specific recommended scratch-branch procedure is a design recommendation for the plan, not itself independently tested

**Research date:** 2026-07-28
**Valid until:** 30 days (release-please-action/create-github-app-token both release somewhat frequently; re-verify pinned SHAs at implementation time regardless of this window)
