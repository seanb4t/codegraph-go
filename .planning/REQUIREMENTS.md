# Requirements: CodeGraph Go — v0.15.0 Changie Release Management

**Defined:** 2026-09-25
**Core Value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.

**Milestone goal:** Replace release-please with changie so the changelog is authored input — one fragment per user-visible change, written at phase close — and the version is derived from it, then prove the chain by cutting a real v0.15.0 release through it.

**ID policy:** prefixes that already exist continue their numbering (`REL-` after v1.0's REL-09; `DOCS-` after v0.14.0's DOCS-11; `GRD-` after GRD-14; `VERB-` after v0.14.0's VERB-08). New prefixes: `CHG-` (changie baseline), `GATE-` (fragment-required check), `CAP-` (phase-close gsd capability), `SHIP-` (the first release through the chain). Backlog 999.5 is promoted as `VERB-09`, keeping its ROADMAP.md Backlog entry annotated with the consuming phase (the 2026-09-08 bookkeeping decision).

**Standing rules that bind every requirement below:** a guard is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`); never `[ci skip]` in a commit message (rule `f18zrdsgx5`); `release.yml`'s `on: push: tags: v[0-9]*` trigger and `internal/upgrade/verify.go`'s cosign identity `release.yml@refs/tags/v[0-9]*` are byte-locked; every action is SHA-pinned with a version comment; `.planning/` and `CHANGELOG.md` are tool-owned; no human `git tag` (D-06R, whose tag authority this milestone transfers from release-please to the changie release workflow); `Taskfile.yml` is the single definition of every CI job body (`TestWorkflowRunBodiesInvokeTask`).

## v1 Requirements

Requirements for this milestone. Each maps to a roadmap phase.

### Changie Baseline

- [x] **CHG-01**: `.changie.yaml` exists with kinds `Breaking` (`auto: minor` until 1.0, with the flip-to-major note in the file), `Features` (`minor`), `Fixes`, `Performance`, `Dependencies` (`patch`), an optional custom `PR` field (`type: int`, `minInt: 1`, optional since Phase 2 D-13, 2026-09-26), and the version/kind/change formats from `notes/changie-release-management.md`; `changie` itself is pinned (v1.26.0 or later, one recorded install path for CI and one for contributors)
- [x] **CHG-02**: `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep` and `.changes/v0.14.0.md` exist; `CHANGELOG.md` keeps every existing entry verbatim below the changie header, byte-diffable against `main` — baseline only, no historical rewrite
- [x] **CHG-03**: `changie latest` answers `v0.14.0`; `changie next auto` with no fragments fails (the expected "nothing to release" outcome, recorded); `changie merge --dry-run` reproduces the current `CHANGELOG.md` byte-for-byte — all three captured as a test or task, not a one-off shell transcript
- [x] **CHG-04**: A fragment written with `CI=true changie new -k <Kind> -b "<sentence>" -m PR=<n>` is non-interactive and lands as `.changes/unreleased/*.yaml`; a fragment without `PR` is accepted, `PR=0` is still refused by `minInt: 1`, and a fragment using an undeclared kind is refused — proven both ways (amended by Phase 2 D-13, 2026-09-26)

### Release Chain

- [ ] **REL-10**: A new release workflow replaces `release-please.yml`: on push to `main` with a non-empty `.changes/unreleased/`, it runs `pretag-gate` (`task check:cross`, moved verbatim) and then `changie batch auto && changie merge` and opens or updates one `chore(main): release vX.Y.Z` PR carrying the rendered `.changes/vX.Y.Z.md` and `CHANGELOG.md`; a push with no unreleased fragments does nothing
- [ ] **REL-11**: When a release PR merges (detected by a new `.changes/vX.Y.Z.md` on `main`), the workflow mints a GitHub App installation token with the existing `APP_ID`/`APP_PRIVATE_KEY` secrets and pushes tag `vX.Y.Z` with it — never the default `GITHUB_TOKEN`, which cannot fire `release.yml`
- [ ] **REL-12**: `release.yml`'s trigger, its SLSA/cosign/notarization/SBOM steps and its job shape are byte-identical except for one change: `goreleaser release` receives `--release-notes .changes/${TAG}.md`; the GoReleaser config never sets `changelog.disable: true` (which would empty the body), asserted by the existing shape tests
- [ ] **REL-13**: `task release:dry-run` renders the release body from a `.changes/vX.Y.Z.md` file and the rehearsal output is captured before the first real tag — the body is non-empty and equals the file
- [ ] **REL-14**: `release-please.yml`, `release-please-config.json` and `.release-please-manifest.json` are deleted; the version source of truth is `changie latest`; a positive-controlled census reports zero `release-please` references outside `.planning/`, `CHANGELOG.md` history and this milestone's own notes

### Guards Re-pointed

- [ ] **GRD-15**: Every test that pins release-please as an authority is re-pointed at the changie chain and demonstrated RED first: `TestReleasePleaseStaysPreMajor` reads `.changie.yaml`'s `Breaking → minor` mapping instead of `bump-minor-pre-major`; `taskfile_shape_test.go`'s job registry names the new workflow's `pretag-gate`; `goreleaser_shape_test.go`'s forbidden-`release:`-keys rationale is restated for GoReleaser-authored Release bodies; `scripts/pr_template_policy.py` drops the release-please path exemptions and gains the release PR's own
- [ ] **GRD-16**: `TestGsdTagCreationIsDisabled` still passes unchanged (`git.create_tag: false`), and a new assertion proves the only `git tag`/`git push --tags` in `.github/workflows/` is the App-token step of the release workflow — RED-demonstrated by planting a second tag push

### Fragment-Required Gate

- [ ] **GATE-01**: A `fragment-required` workflow runs on every pull request and fails when the PR's changed files touch `cmd/**` or `internal/**` (excluding `*_test.go` and `testdata/`) and add no file under `.changes/unreleased/`; the path rule lives in one place shared with the gate's tests
- [ ] **GATE-02**: A PR body carrying `<!-- changelog-exempt: <reason> -->` with a non-empty reason passes the gate; the marker's grammar mirrors `pr-template-exempt` and an empty reason does not exempt
- [ ] **GATE-03**: The gate fails closed: an empty changed-file list, an API error, or an unreadable body each fail with a distinct message, never pass (the `require-issue-link` precedent)
- [ ] **GATE-04**: The gate is demonstrated in both directions against real fixtures — a PR shape with a fragment passes and the identical shape without one fails — with the run or test output recorded, and the count of executed cases asserted (rule `84d1gfpywd`)
- [ ] **GATE-05**: `fragment-required` is added to ruleset `protect-main`'s required status checks (7 contexts), the `.github/required-status-checks` fixture updated in the same change, and `TestRequiredCheckNamesPreserved` plus `scripts/check-ruleset-drift.sh` pass against the live ruleset — the ruleset edit is a repository-settings action recorded with its timestamp

### Phase-Close Capability

- [x] **CAP-01**: A private repository `seanb4t/gsd-capability-changie` exists holding a `role: "feature"` capability whose `capability.json` validates under gsd-core 1.14.0 (`gsd_run capability install ./ --scope project` on a scratch project succeeds; id is not a reserved `gsd-` prefix), with `engines.gsd` pinned, a `README`, a `LICENSE` and a tagged release
- [x] **CAP-02**: The capability owns a `changie-fragments` skill and registers one `verify:post` step (`ref.skill`, `onError: skip`) gated on a federated config key `workflow.changie_fragments` (`boolean`, default `true`), so a project with the key `false` never dispatches it
- [ ] **CAP-03**: The skill, given a phase number, reads that phase's `*-SUMMARY.md` files, writes one fragment per user-visible change via `CI=true changie new` using only the kinds declared in the host repo's `.changie.yaml`, commits them with the phase's commit conventions, and skips with a printed reason when `changie` is not on `PATH`, no `.changie.yaml` exists, or the phase produced no user-visible change — never prompting, never blocking the workflow
- [ ] **CAP-04**: codegraph-go installs the capability at project scope from `https://github.com/seanb4t/gsd-capability-changie.git#<tag>`, `workflow.changie_fragments` is set in `.planning/config.json`, and `CONTRIBUTING.md` documents the one-time per-clone install (`.gsd/` stays gitignored)
- [ ] **CAP-05**: The step is observed firing at a real phase close inside this milestone: the fragment it wrote is in `.changes/unreleased/`, its commit is on the milestone branch, and the phase's VERIFICATION or SUMMARY records the dispatch — evidence, not assertion

### Docs

- [ ] **DOCS-12**: `CONTRIBUTING.md` §Pull requests replaces "release-please derives the version and the changelog from merged PR titles" and the Release-As rules with: add a fragment per user-visible change; never hand-write `.changes/v*.md`; how to exempt; how a version is derived
- [ ] **DOCS-13**: `docs/RELEASE-PROCEDURES.md` describes the changie chain end to end (fragment → release PR → merge → App-token tag → locked `release.yml`), moves the `pretag-gate` description to the new workflow, and states that a release is cut by merging the release PR; its 45 release-please references are resolved one by one with a recorded verdict
- [ ] **DOCS-14**: `pr-title.yml`'s header comment says it is hygiene only and carries no versioning weight; `Taskfile.yml`'s `release:dry-run-signed` comment and `docs/RELEASE.md`'s pipeline paragraph name the changie workflow

### Verb Fold Close-out

- [ ] **VERB-09**: The hidden `query` and `unlock` rename stubs are removed (999.5): `internal/cli/renamed.go` and `renamed_test.go` deleted, `newQueryCmd()`/`newUnlockCmd()` dropped from `root.go`, the two `cli-reference-allowlist.txt` lines removed, `task docs:cli` produces no reference change, and the Phase 3 census instrument (v0.14.0 `03-MUTATION-LOG.md` family (c)) reports zero hits — recorded as this milestone's `Breaking` fragment

### First Release Through the Chain

- [ ] **SHIP-01**: The first changie release PR is opened by the workflow from this milestone's fragments, reviewed, and merged; `changie latest` on `main` then answers `v0.15.0` (or whatever the fragments derive — the label is a prediction)
- [ ] **SHIP-02**: The resulting tag is authored by the GitHub App, `release.yml` ran on that tag push, and the GitHub Release body equals `.changes/v0.15.0.md` byte-for-byte
- [ ] **SHIP-03**: `post-release-verify` is green against the published assets, and `codegraph upgrade` from an installed v0.14.0 verifies the new artifact against the unchanged cosign identity on darwin/arm64 and linux/amd64
- [ ] **SHIP-04**: `SEED-005` is annotated with the verified trigger (this release) so the other repos can pick up the template; the capability repo's tag used by CAP-04 is the one this release ran with

## v2 Requirements

Deferred to a future release. Tracked but not in the current roadmap.

### Porting and polish

- **PORT-01**: `.changie.yaml`, the release workflow and the fragment gate ported to router-hosts, engram, fovea and fzymgc-house-skills (SEED-005; engram's release PR also bumps `charts/engram/Chart.yaml` and `plugin.json` via changie `replacements`, which run on `merge`)
- **PORT-02**: A first-party changie capability proposed to open-gsd/gsd-core, once the private one has proven the `verify:post` shape on two repos
- **CHG-05**: Historical `.changes/vX.md` backfill regenerated from the GitHub Release notes for uniform changelog formatting

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Keeping release-please with `skip-changelog` | Leaves the title-driven bump — the failure this milestone exists to remove; `release-as` is a one-shot override, not a steady-state input |
| Tagging directly from `main` on fragment merge | No human review of the rendered notes and version before the tag exists; the release PR is the review point |
| Touching `release.yml`'s trigger, cosign identity or build/sign/attest steps | The locked contract `codegraph upgrade` verifies against; only who pushes the tag changes |
| `changelog.disable: true` in GoReleaser | Makes GoReleaser ignore `--release-notes`, producing an empty Release body |
| A `contribution` or `ref.command` step for the capability | `contribution` is not wired at `verify:post`; `ref.command` needs a `.cjs` command module (an executable surface with heavier consent). A skill step is the least-privilege sanctioned shape |
| Editing gsd-core's own workflow files for the fragment hook | Tool-owned; the capability manifest is the sanctioned extension vocabulary |
| GH #85 service mode, GRF-07, SEED-003, SEED-004, DIST-06, BREW-07, MRTR-01, NUDGE-07…10, AGENT-12, BRW-14, GH #23, GH #9 | Parked; service mode is the next milestone and ships through this chain |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CHG-01 | Phase 1 | Complete |
| CHG-02 | Phase 1 | Complete |
| CHG-03 | Phase 1 | Complete |
| CHG-04 | Phase 1 | Complete |
| REL-10 | Phase 5 | Pending |
| REL-11 | Phase 5 | Pending |
| REL-12 | Phase 5 | Pending |
| REL-13 | Phase 5 | Pending |
| REL-14 | Phase 5 | Pending |
| GRD-15 | Phase 5 | Pending |
| GRD-16 | Phase 5 | Pending |
| GATE-01 | Phase 4 | Pending |
| GATE-02 | Phase 4 | Pending |
| GATE-03 | Phase 4 | Pending |
| GATE-04 | Phase 4 | Pending |
| GATE-05 | Phase 4 | Pending |
| CAP-01 | Phase 2 | Complete |
| CAP-02 | Phase 2 | Complete |
| CAP-03 | Phase 2 | Pending |
| CAP-04 | Phase 2 | Pending |
| CAP-05 | Phase 3 | Pending |
| DOCS-12 | Phase 5 | Pending |
| DOCS-13 | Phase 5 | Pending |
| DOCS-14 | Phase 5 | Pending |
| VERB-09 | Phase 3 | Pending |
| SHIP-01 | Phase 6 | Pending |
| SHIP-02 | Phase 6 | Pending |
| SHIP-03 | Phase 6 | Pending |
| SHIP-04 | Phase 6 | Pending |

**Coverage:**

- v1 requirements: 29 total
- Mapped to phases: 29
- Unmapped: 0 ✓

---
*Requirements defined: 2026-09-25*
*Last updated: 2026-09-25 after roadmap creation*
