# Phase 9: release-please + GoReleaser - Context

**Gathered:** 2026-07-28
**Status:** Ready for planning
**Mode:** `--auto --chain` (all gray areas auto-selected; recommended option chosen and logged inline / in `09-DISCUSSION-LOG.md`; auto-advancing to plan → execute)

<domain>
## Phase Boundary

Replace the **maintainer-manual `git tag`** step with automated release management:
release-please owns version bumps, `CHANGELOG.md`, and tag/release creation from
Conventional Commits. The existing signed-build pipeline
(`.github/workflows/release.yml` + `.goreleaser.yaml`) is **wired to it, not
rewritten**.

**The single requirement: REL-02** — "releases are cut by release-please from
Conventional Commits — version bump, `CHANGELOG.md`, and tag creation all happen
without a human running `git tag` — and the resulting signed artifacts still
satisfy `internal/upgrade/verify.go`'s cosign identity
(`releaseWorkflowRefPattern`, SAN anchored to `release.yml@refs/tags/v[0-9]*`),
so `codegraph upgrade` keeps working for already-shipped binaries."

Both halves of that sentence are load-bearing. The second half is a **LOCKED
contract**, not a preference: every already-shipped binary (`v0.0.0-rc.3`,
`v0.1.0`) hard-codes the SAN regex it will accept. Break it and `codegraph
upgrade` rejects every future release, **silently**, for every existing user.

**In scope:**
- `release-please-config.json` (`release-type: go`) + `.release-please-manifest.json`
- A new `.github/workflows/release-please.yml` on `push: branches: [main]`
- Making release-please's tag actually *fire* the existing `release.yml`
- Making `release.yml`'s publish step idempotent against a release-please-created release
- Producing the real signed **`v1.0.0`** through the new path
- Updating `docs/RELEASE-PROCEDURES.md`, whose §3/§4/§7 this phase invalidates

**Not in this phase (explicit out-of-scope):**
- **Rebuilding the build/sign/SBOM/SLSA pipeline.** The 2-OS native matrix,
  per-binary cosign, and SLSA generic-generator handoff are Phase-8-audited and
  stay as-is. This phase changes *what triggers them* and *where artifacts land*.
- **Changing `release.yml`'s name, its `v[0-9]*` tag trigger, or the cosign
  identity** — see D-01. Any such change requires a lockstep `verify.go` edit
  plus a migration story for shipped binaries, which is a different phase.
- **Migrating to `goreleaser release`** (see D-04) — rejected, with the roadmap
  divergence recorded.
- Contributor `Taskfile.yml` / `CONTRIBUTING.md` — that is Phase 10.
- Any CLI/MCP behavioral surface change. Phase 9 touches CI, config, and docs only.

</domain>

<decisions>
## Implementation Decisions

### The locked contract (constrains everything below)

- **D-01:** `internal/upgrade/verify.go`'s
  `releaseWorkflowRefPattern` = ``^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$``
  is **untouchable in this phase**. Concretely this forbids: renaming
  `release.yml`, removing its `push.tags: v[0-9]*` trigger, moving the
  `id-token: write` cosign step into any other workflow file, and signing from
  any ref that is not `refs/tags/v…`. The roadmap's stated alternative
  ("collapse release-please + ship into one workflow like engram, change
  `releaseWorkflowRefPattern` in lockstep") is **rejected for this phase** — it
  breaks `codegraph upgrade` for everyone already on `v0.1.0`/`v0.0.0-rc.3` and
  needs a migration story first.
  — **Reversibility:** one-way — the SAN is compiled into every published binary;
  changing it cannot be un-shipped, and the only remedy is telling users to
  manually re-download.
  `[auto] Locked contract → Selected: SAN/trigger/filename frozen; engram-style single-workflow collapse rejected (recommended)`

### Gray area 1 — How release-please's tag actually fires `release.yml`

- **D-02:** **Path A (primary): a GitHub App token.** The release-please workflow
  authenticates via `actions/create-github-app-token`, and release-please uses that
  token to open/merge its release PR and create the tag. This is required because
  **tags pushed with the default `GITHUB_TOKEN` do not trigger other workflows** —
  which is precisely why engram collapses everything into one job, the thing D-01
  forbids here. App-token-created refs *do* trigger downstream workflows, so
  `release.yml` fires on the tag push exactly as it does today, with an unchanged
  SAN.
  Requires two new repo secrets (`APP_ID`, `APP_PRIVATE_KEY`) — **the repo
  currently has zero secrets configured** (`gh secret list` is empty), so App
  creation + installation is a **maintainer-manual prerequisite**, in the same
  category as Phase 8's manual tag. Plan for it as a blocking human checkpoint
  with a documented setup procedure, not as an executor task.
  — **Reversibility:** reversible — swapping the token source is a few lines in
  one workflow file; nothing is published from it.
  `[auto] Tag-trigger mechanism → Selected: GitHub App token (roadmap-preferred) (recommended)`

- **D-03:** **Path B (documented fallback, only if the App is unavailable):** add
  `workflow_dispatch` alongside — never instead of — `release.yml`'s existing
  `push.tags` trigger, and have the release-please job dispatch it **at the tag
  ref** (`gh workflow run release.yml --ref "$TAG"`). This is SAN-safe because
  the pattern constrains the **ref**, not the event: a dispatch at `refs/tags/v1.0.0`
  still yields `release.yml@refs/tags/v1.0.0`. It is **not** safe if dispatched at
  a branch — that produces `@refs/heads/main` and unverifiable binaries. If Path B
  is implemented, it MUST carry a guard step that hard-fails when
  `github.ref` does not start with `refs/tags/v`, plus a test proving the guard
  fires on a rejecting input (Phase-8 lesson: `CR-01`/`WR-02` were guards that were
  present but never fired). Also update `release.yml`'s header comment, which
  currently asserts "MUST trigger ONLY on tag pushes".
  — **Reversibility:** costly — adding a second trigger to the LOCKED file widens
  the surface that can emit a mis-signed artifact; the guard becomes permanently
  load-bearing.
  `[auto] App-token fallback → Selected: workflow_dispatch at tag ref + non-vacuous ref-shape guard, documented as Path B only (recommended)`

### Gray area 2 — The `gh release create` collision (the highest-risk detail)

- **D-04:** **release-please owns the GitHub Release and its notes; `release.yml`
  only uploads assets into it.** release-please creates a GitHub Release
  automatically when its release PR merges (its own design docs: "release-please
  automatically creates a GitHub release after a release pull request is merged").
  Today `release.yml:279` runs `gh release create "$TAG" … --generate-notes
  codegraph_*`, which will (a) **fail outright** because the release already
  exists and (b) if forced, **clobber release-please's changelog body** with
  auto-generated notes.
  Replace it with a **create-if-absent-else-upload-clobber** step:
  - release exists (the release-please path) → `gh release upload "$TAG" --clobber codegraph_*`,
    leaving the release-please-authored body and prerelease flag untouched;
  - release absent (a manually-pushed `rc` tag, see D-09) → keep today's
    `gh release create … --generate-notes` with the existing `-`-suffix
    `--prerelease` logic.
  This is what roadmap criterion 3's `replace_existing_artifacts: true` means for
  *this* repo's pipeline; `--clobber` also makes re-runs idempotent, which
  criterion 3 explicitly asks for.
  — **Reversibility:** reversible — a contained edit to one shell step in the
  `assemble` job; no published contract depends on it.
  `[auto] Release-creation collision → Selected: create-if-absent-else-upload-clobber; release-please owns the notes (recommended)`

### Gray area 3 — GoReleaser's actual role (roadmap divergence, recorded)

- **D-05:** **Keep the `goreleaser build --single-target` model. Do NOT introduce
  `goreleaser release`.** The roadmap's criterion 3 ("GoReleaser uploads artifacts
  … `replace_existing_artifacts: true`, no `changelog:` block") is modeled on
  engram and **does not transfer**: this repo has never run `goreleaser release`.
  `.goreleaser.yaml`'s `archives:` and `checksum:` blocks are already documented
  in-file as **dead configuration** (lines 129–146), and signing/SBOM/publishing
  live in `release.yml`'s hand-written `assemble` job precisely because
  GoReleaser OSS cannot express (i) per-binary cosign — `internal/upgrade`'s
  `defaultVerify` hashes the **binary itself**, not a checksums file — (ii) the
  native 2-OS matrix that keeps darwin off a zig cross-link (the libresolv/DNS
  risk), or (iii) the SLSA generic-generator handoff. Migrating would require
  GoReleaser Pro's `release --split`/`--merge`, which Phase 8 research explicitly
  ruled out.
  **Record this as an accepted divergence from roadmap criterion 3**, satisfied
  instead by D-04's `--clobber` idempotency. Do not silently drop the criterion —
  amend it the way Phase 8's goal amendment was recorded.
  — **Reversibility:** costly — reverting to a `goreleaser release` model would
  mean re-deriving the per-binary signing and native-darwin guarantees that Phase
  8 audited and that `verify.go` depends on.
  `[auto] GoReleaser role → Selected: build-only preserved; criterion 3 recorded as accepted divergence (recommended)`

### Gray area 4 — What version the first automated cut produces

> **⚠ D-06 REVERSED 2026-07-29 by maintainer directive.** The original decision
> (preserved verbatim below) forced `v1.0.0` via a one-shot `Release-As:` footer.
> The maintainer rejected it: *"We are **not** going to jump to 1.0, stop pushing
> it. We'll follow things as release-please and conventional commits requires."*
> This is consistent with — and was foreshadowed by — the 2026-07-28
> recategorization that rewrote REL-02 from an *event* ("a signed `v1.0.0` tag
> exists") into a *property* ("releases are cut by release-please from
> Conventional Commits… and still satisfy the cosign identity"). **REL-02 does
> not name a version, and never did.** The forced version was reintroduced during
> this phase's discuss step and is now removed. D-06 is superseded by D-06R.

- **D-06R (supersedes D-06):** **Seed `.release-please-manifest.json` with the
  real current version `0.1.0` and let release-please compute every subsequent
  version from Conventional Commits. Force nothing.** No `Release-As:` footer, no
  `release-as` key in `release-please-config.json`, no breaking-change marker
  authored to manufacture a major bump. The manifest seed stays at `0.1.0`
  because `v0.1.0` is the real last release — that is a truthful baseline, not a
  version target.
  **Expected first cut: `0.2.0`** — for a `0.x` baseline release-please bumps the
  minor on `feat:`, and reaching `1.0.0` would require exactly the kind of
  override this decision forbids. Maintainer confirmed `0.2.0` is correct
  (2026-07-29). Verified empirically by `release-please release-pr --dry-run`
  against the real branch history, not asserted.
  **Guard against silent reintroduction:** no commit reaching `main` may carry a
  line beginning `Release-As:`. Two commit *bodies* on this branch mention the
  footer in prose while documenting the runbook (`7f60822`, `62916bc`); neither
  starts a line with the keyword, so neither parses as a footer — confirmed by
  `git log main..HEAD --format='%B' | rg "^Release-As:"` returning nothing.
  — **Reversibility:** reversible — a future release *may* legitimately reach
  `1.0.0` when Conventional Commits say so. What is forbidden is manufacturing it.
  `[maintainer 2026-07-29] version derivation → Selected: no forcing; release-please computes it (0.2.0 next)`

<details>
<summary>Superseded original D-06 (kept for audit; do not act on)</summary>

- **D-06:** **Seed `.release-please-manifest.json` with the real current version
  `0.1.0`, then force the first automated cut with an explicit
  `Release-As: 1.0.0` footer** on an empty `chore: release 1.0.0` commit.
  Rationale: the newest tag is `v0.1.0`, and release-please's default `0.x`
  behavior would compute `0.2.0` from the accumulated `feat:` commits, never
  `1.0.0`. `Release-As:` takes precedence over the versioning strategy at the top
  of `buildNewVersion`, so it is deterministic and auditable. Seeding the manifest
  at `0.1.0` (rather than `0.0.0`) keeps release-please's baseline honest against
  the tag that actually exists.
  Rejected alternatives: `release-as` pinned in `release-please-config.json`
  (sticky — it would force `1.0.0` on *every* subsequent run until removed);
  relying on a `feat!:`/`BREAKING CHANGE` marker (yields `1.0.0` only by accident
  of the pre-major rules and reads as a lie — v1.0 is a milestone, not a breaking
  change).
  — **Reversibility:** one-way once published — `v1.0.0` becomes what `codegraph
  upgrade` resolves as "latest"; a wrong version number can only be superseded,
  never withdrawn.
  `[auto] v1.0.0 derivation → Selected: manifest seeded 0.1.0 + one-shot Release-As: 1.0.0 footer (recommended)`

</details>

### Gray area 5 — Version source of truth

- **D-07:** **Leave ldflags as the sole build-time version source; do NOT add
  `internal/version/version.go` to release-please `extra-files`.** `.goreleaser.yaml`
  already injects `-X …/internal/version.Version={{ .Tag }}`, so the tag *is* the
  version at build time; `version.go`'s `Version = "dev"` default is the
  deliberate non-release sentinel. Adding a second source that release-please
  rewrites invites exactly the doc-vs-code drift this project has been bitten by
  twice (the MCP `default 5 vs 2` schema escape; PROJECT.md's "v1.0.0 shipped"
  claim with no tag). release-please's `release-type: go` manages `CHANGELOG.md`
  and the manifest and requires no Go source file.
  — **Reversibility:** reversible — `extra-files` can be added later without
  touching anything published.
  `[auto] Version source → Selected: ldflags only, no extra-files (recommended)`

### Gray area 6 — Conventional-commit input gate

- **D-08:** **Add a lightweight PR-title conventional-commit check to `ci.yml`.**
  release-please's entire output is a function of commit messages, and the repo
  currently enforces nothing (no commitlint, no PR-title lint in `ci.yml`). The
  last 30 commits are 30/30 conformant, so this codifies existing practice rather
  than imposing a new one. Gate the **PR title** specifically, because the
  squash-merge model (D-09) makes the PR title the commit message release-please
  parses — linting individual branch commits would gate text that never reaches
  `main`.
  — **Reversibility:** reversible — a single CI job; removable with no artifact impact.
  `[auto] Commit-message gate → Selected: PR-title conventional-commit check in ci.yml (recommended)`

### Gray area 7 — The v1.0 merge model (⚠ documented model vs. actual practice)

- **D-09:** **Fast-forward the v1.0 integration branch onto `main`, preserving
  full history — do NOT squash.** Verified against the repo, not inferred:
  - `main` contains **zero merge commits** (`git log --merges main` is empty) —
    the v0.1 milestone landed by fast-forward.
  - `main` **is an ancestor of HEAD** today, so
    `gsd/v1.0-drop-in-parity-human-ux` is fast-forwardable as-is.
  - The branch is **477 commits ahead**, of which **160 are `feat`/`fix`/`perf`**
    and ~217 are planning-only `docs(...)` commits.
  Under fast-forward, release-please sees all 477 commits and generates a
  genuinely rich `v1.0.0` changelog from those 160 entries, while its default
  `changelog-sections` hide `docs:`/`chore:`/`ci:`/`test:` — so the planning
  commits are filtered out automatically, not manually.
  **⚠ Conflict recorded, not silently resolved:** Phase-8's `08-CONTEXT.md` D-08
  says "squash-merge to `main`". That wording contradicts what the repo actually
  did for v0.1 (fast-forward — also recorded in engram `8sa948y0g4`,
  "fast-forward merge not squash"). D-08's *substance* — integration branch →
  `main` → tag on `main` — is honored here in full; only its merge-mechanic
  wording is superseded, and only because the evidence on disk contradicts it.
  A squash would collapse all 477 commits into one message and reduce `v1.0.0`'s
  changelog to a single line, discarding the input release-please exists to
  consume. **If the maintainer intended a true squash, say so before planning —
  this is the one decision here worth overriding.**
  — **Reversibility:** one-way — merge shape is fixed the moment it lands on
  `main`; a squashed history cannot be un-squashed to regenerate the changelog.
  `[auto] Merge model → Selected: fast-forward preserving history (matches actual v0.1 practice + main has zero merge commits); Phase-8 D-08 squash wording superseded on evidence (recommended)`

### Gray area 8 — Prerelease / rc story

- **D-10:** **Automate stable releases only. Keep manual `rc` tags as a documented
  escape hatch.** Do not configure a release-please prerelease strategy in this
  phase. `v0.0.0-rc.N`-style tags still match `release.yml`'s `v[0-9]*` trigger
  and still fire the signed build; they simply take D-04's *create* branch instead
  of its *upload* branch. This keeps the automated path narrow and is why D-04
  must handle both paths rather than assuming a release always pre-exists.
  — **Reversibility:** reversible — `prerelease: true` + `prerelease-type` can be
  added to the config later.
  `[auto] Prerelease story → Selected: stable-only automation, manual rc path retained (recommended)`

### Claude's Discretion
- Whether the release-please action is pinned by SHA (matching `release.yml`'s
  convention for every third-party action) or by tag — prefer SHA for consistency.
- Exact `release-please-config.json` shape: `changelog-sections` customization,
  `include-component-in-tag: false`, `bump-minor-pre-major` (moot once at 1.x).
- Whether the PR-title lint (D-08) uses an off-the-shelf action or a ~10-line
  `grep -E` step — the latter avoids a new pinned dependency and matches the
  repo's preference for hand-written, auditable CI shell.
- Whether the `assemble` job's create-vs-upload branch is decided by
  `gh release view "$TAG"` exit status or by an explicit workflow input.
- Whether the GitHub App setup procedure lives in `docs/RELEASE-PROCEDURES.md` or
  a sibling `docs/RELEASE-AUTOMATION.md`.
- Whether to keep or delete `.goreleaser.yaml`'s dead `archives:`/`checksum:`
  blocks — they are documented-inert; deleting removes a footgun, keeping
  preserves stated intent. Either is defensible; do not change them *silently*.

### Folded Todos
- **"Document release procedures (maintainer runbook)"**
  (`.planning/todos/2026-07-14-document-release-cut-procedures-runbook.md`,
  `area: docs`, `resolves_phase: 8`, matched Phase 9 at score 0.6 — **folded, but
  as an update, not a new deliverable**). The runbook already exists — Phase 8
  shipped `docs/RELEASE-PROCEDURES.md` with all 8 sections. Phase 9
  **invalidates** three of them and MUST rewrite them:
  - **§3 Branch/tag model** and **§4 "Cutting the tag (maintainer-manual action)"**
    — the manual `git tag` is exactly what this phase removes.
  - **§7 Rollback / cleanup of a failed rc tag** — with release-please owning
    tags, recovery now also means reverting the manifest bump and the release PR,
    not just `git push --delete`.
  Add a new section covering the **GitHub App prerequisite** (D-02) and the
  release-PR-merge flow. §1 (pre-tag 6-target `go list` gate), §2, §5 (the
  `verify.go` LOCKED contract), §6 (post-release cosign/SLSA verification), and §8
  (`-c commit.gpgsign=false` for pipeline commits only, per repo rule
  `xmz3xknbj0`) all survive unchanged — §1 in particular should become an
  automated gate in the release-please workflow now that no human runs it by hand.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope (locked)
- `.planning/ROADMAP.md` §"Phase 9: release-please + GoReleaser" — goal, 4 success
  criteria, and the Notes paragraph stating the LOCKED-contract constraint and the
  two resolution paths. **Criterion 3 is amended by D-05** (see that decision).
- `.planning/REQUIREMENTS.md` — **REL-02** (line 89) is the sole requirement; it
  was rewritten from a project event into a testable property on 2026-07-28.
  Ownership table line 178 assigns it to Phase 9.
- `.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-CONTEXT.md`
  §D-08 (branch/tag model + the LOCKED contract), §D-09 (mandatory pre-tag
  6-target `go list -mod=readonly` sweep) — both carry forward into Phase 9.

### The LOCKED release identity (read before touching any CI file)
- `internal/upgrade/verify.go` lines 20–44 — `releaseOIDCIssuer`,
  `releaseRepoSlug`, `releaseWorkflowRefPattern`. The full-match SAN pattern is
  the contract D-01 freezes.
- `internal/upgrade/verify_test.go` lines 118–136 — the two tests asserting the
  pattern **accepts** `release.yml@refs/tags/v1.2.3` and **rejects**
  `ci.yml@refs/heads/main`. Any new trigger (D-03) needs an equivalently
  non-vacuous test.
- `internal/upgrade/upgrade.go` lines 164–210 — `defaultVerify` hashing the
  downloaded binary, and `releaseAssetName()`'s
  `codegraph_<tag>_<goos>_<goarch>[.exe]` contract.

### The pipeline being wired (not rewritten)
- `.github/workflows/release.yml` — the whole file. Header comment lines 1–36
  states the locked contract in situ. **Line 279's `gh release create …
  --generate-notes` is the exact line D-04 replaces.** Line 274–277 holds the
  `-`-suffix `--prerelease` logic D-10 preserves. Lines 61–100 are the native
  2-OS matrix D-05 protects; lines 226–241 the per-binary cosign; lines 293–311
  the SLSA generic generator.
- `.goreleaser.yaml` — header comment (a)/(b)/(c) lines 1–34 explain why
  `archives:`/`checksum:` (lines 129–146) are **dead config** and why there is
  deliberately no `signs:`/`sbom:` block. This is the primary evidence for D-05.
- `.github/workflows/ci.yml` — where D-08's PR-title lint lands; also holds the
  govulncheck (line 149) and reproducible double-build (line 166) gates.
- `internal/version/version.go` — the ldflags-injected `Version`/`Commit`/`Date`
  vars and the `"dev"` sentinel D-07 preserves.

### Docs this phase must update
- `docs/RELEASE-PROCEDURES.md` — §3, §4, §7 are invalidated by this phase; §1, §2,
  §5, §6, §8 survive. See Folded Todos.
- `.planning/todos/2026-07-14-document-release-cut-procedures-runbook.md` — the
  folded todo; mark resolved when the runbook is updated.
- `docs/RELEASE.md` — the *user*-facing verification doc. Should not need changes
  (the verification commands are unchanged), but confirm rather than assume.

### External docs (fetch current versions during research — do not work from memory)
- release-please: `/googleapis/release-please` — `release-type: go`, manifest
  bootstrapping/`initial-version`, the `Release-As:` footer precedence in
  `buildNewVersion`, and the "release-please automatically creates a GitHub
  release after a release pull request is merged" behavior that D-04 hinges on.
- release-please-action: `/googleapis/release-please-action` — inputs, required
  permissions (`contents: write`, `pull-requests: write`), token wiring.
- `actions/create-github-app-token` — App-token minting for D-02.

### Cross-phase constraints carried forward (MUST honor)
- Repo rule `xmz3xknbj0` (engram) — `-c commit.gpgsign=false` is allowed for
  agent/pipeline commits ONLY; sign when the user is at the keyboard and asks.
- Phase-8 lesson (`CR-01`, `WR-02`): **a guard that is present but never fires is
  not a guard.** Any new CI guard in this phase must be proven non-vacuous by
  feeding it a rejecting input and observing failure.
- Phase-8 lesson: never close a fix loop on iteration 1; a test proves nothing
  unless it fails without its fix.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`.github/workflows/release.yml`** — a complete, proven, tag-triggered
  build→sign→SBOM→publish→provenance pipeline that has shipped two real releases
  (`v0.0.0-rc.3`, `v0.1.0`). Phase 9 changes **one shell step** in it (D-04) and
  otherwise leaves it alone.
- **`.goreleaser.yaml`** — six explicit build entries with reproducibility flags
  (`-trimpath`, `-buildid=`, `mod_timestamp`) already wired to `{{ .Tag }}`.
  release-please supplies the tag; nothing in this file needs to change.
- **`internal/version`** — already tag-driven via ldflags, so no version file for
  release-please to rewrite (D-07).
- **`docs/RELEASE-PROCEDURES.md`** — 8 sections; 5 survive verbatim, so the doc
  work is a targeted amendment, not a rewrite.
- **Conventional-commit discipline already in practice** — 30/30 of the last 30
  commits conform, so release-please gets clean input from day one.

### Established Patterns
- **Pin every third-party Action to a full commit SHA**, with the resolved tag in
  a trailing comment — the sole documented exception is the SLSA generic
  generator, which `slsa-verifier` requires be referenced by a `@vX.Y.Z` tag. New
  actions in this phase must follow the SHA convention.
- **Never interpolate `${{ }}` directly into a `run:` script** — pass workflow
  context via `env:` and reference `$VAR`, so a crafted tag name cannot inject
  shell (see `release.yml` lines 136–144). Any new step handling `$TAG` inherits this.
- **Hand-written, auditable CI shell over opaque actions** where a contract must
  be exact (asset naming, checksums, per-binary signing). D-05 is the
  phase-scale expression of this same preference.
- **Documented divergence over silent drift** — Phase 8's `docs/FLAG-PARITY.md`
  pattern. D-05's roadmap-criterion divergence gets the same explicit treatment.

### Integration Points
- **New** `.github/workflows/release-please.yml` (`push: branches: [main]`) →
  mints an App token → runs release-please → creates tag + GitHub Release.
- **That tag push** → fires the existing `release.yml` (unchanged trigger, unchanged SAN).
- **`release.yml` `assemble` job, the `Publish GitHub release` step (line 266)** →
  becomes create-if-absent-else-`upload --clobber` (D-04). This is the single
  highest-risk edit in the phase.
- **New** `release-please-config.json` + `.release-please-manifest.json` at repo root.
- **`ci.yml`** → gains the PR-title conventional-commit job (D-08).
- **`docs/RELEASE-PROCEDURES.md`** §3/§4/§7 → rewritten; new GitHub App section added.
- **Repo settings (maintainer-manual)** → create + install the GitHub App, add
  `APP_ID` / `APP_PRIVATE_KEY` secrets. The repo currently has **no secrets and no
  rulesets**; if branch protection on `main` is added later, the App must be on
  the bypass-actor list.

</code_context>

<specifics>
## Specific Ideas

- The repo is **private**, default branch `main`, and has exactly three tags:
  `milestone-v0.1`, `v0.0.0-rc.3`, `v0.1.0`. `.release-please-manifest.json` seeds
  from **`0.1.0`** (D-06). `milestone-v*` is deliberately non-matching so it never
  fires `release.yml` — keep it that way.
- ~~The `Release-As: 1.0.0` footer goes on an **empty** commit and is a
  **one-shot**.~~ **Struck 2026-07-29 (D-06R).** No `Release-As:` footer is
  authored by this phase. release-please computes the version from Conventional
  Commits; the next cut is **`0.2.0`**, confirmed by a live `release-pr
  --dry-run` against the real branch (`title: chore(...): release 0.2.0`, baseline
  resolved from the real `v0.1.0`). The mechanism remains documented in
  `docs/RELEASE-PROCEDURES.md` §4 as a tool that *exists*, explicitly not as a
  step this phase performs. `release-as` must still never go in the config file —
  it is sticky and would pin every subsequent release.
- **Do not** add `--generate-notes` to the upload path. release-please's changelog
  IS the release body; auto-generated notes would overwrite it (D-04).
- Adding `workflow_dispatch` to `release.yml` (D-03, fallback only) is SAN-safe
  **only** when dispatched with `--ref <tag>`. A dispatch at a branch produces
  `@refs/heads/main` and silently unverifiable binaries. Guard it and prove the
  guard fires.
- `docs/RELEASE-PROCEDURES.md` §1's pre-tag 6-target
  `GOOS=… GOARCH=… go list -mod=readonly ./...` sweep exists because v0.1's
  `rc.1` failed on a **linux-only** `go.sum` hash invisible from darwin. Now that
  no human runs it manually, it should become an automated gate before the tag is
  created — otherwise D-09's lesson is silently dropped.
- **Whatever version this phase publishes (`0.2.0`) becomes what `codegraph
  upgrade` resolves as "latest"** for every
  existing user the moment it publishes. Post-release verification (§6:
  `cosign verify-blob` + `slsa-verifier verify-artifact`) is not optional for the
  first automated cut — it is the only proof the SAN survived the rewiring, and it
  is the direct evidence REL-02 needs.

</specifics>

<deferred>
## Deferred Ideas

- **Migrating to `goreleaser release`** with `signs:`/`sboms:` blocks and
  `replace_existing_artifacts: true` — would need GoReleaser Pro (`release
  --split`/`--merge`) and a re-derivation of the per-binary cosign and
  native-darwin guarantees. Revisit only if the hand-written `assemble` job
  becomes a maintenance burden. Not v1.0.
- **Changing `releaseWorkflowRefPattern`** to allow a consolidated
  release-please+ship workflow (engram's model) — needs a migration story for
  already-shipped binaries first. Own phase, post-v1.0.
- **release-please prerelease automation** (`prerelease: true` + `prerelease-type`)
  for automated `rc` cuts — D-10 keeps rc manual for now.
- **Making the repo public** — the SLSA generator's `private-repository: true`
  opt-in and the "destined to be public OSS" note in `release.yml` both become
  moot. Not a Phase 9 action.
- **Phase 10** — `Taskfile.yml` + `CONTRIBUTING.md`, deliberately sequenced after
  this phase so the task wrappers (`goreleaser check`, cross-`GOOS` `go list`,
  `govulncheck`, `actionlint`) are written once against the final release setup.
- **Backlog 999.2** — tmux real-PTY e2e/UAT harness for the interactive TUI.
- **Team-scale / central-server / CI-distributed indexes / annotations / local
  Svelte web UI (SEED-001)** — post-v1.0 milestone, per PROJECT.md.

### Reviewed Todos (not folded)
None — the single matching todo (release maintainer runbook, score 0.6) was
folded as an **update** to the existing `docs/RELEASE-PROCEDURES.md` rather than
as a new deliverable (see Folded Todos).

</deferred>

---

*Phase: 9-release-please-and-goreleaser*
*Context gathered: 2026-07-28*
