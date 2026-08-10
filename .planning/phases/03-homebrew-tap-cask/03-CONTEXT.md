# Phase 3: Homebrew Tap & Cask - Context

**Gathered:** 2026-08-09
**Status:** Ready for planning

<domain>
## Phase Boundary

A macOS user installs codegraph the way they install everything else — `brew tap seanb4t/tap && brew install codegraph` — and it keeps working across releases rather than only on the day it was hand-checked. The cask is published by GoReleaser's `homebrew_casks:` block on every release, authenticated by a credential that can write the tap repository and nothing else, and it delivers shell completions and man pages generated from the installed binary rather than from committed files.

**In scope:** the `homebrew_casks:` block in `.goreleaser.yaml`; a man-page generation capability in the binary; the cask's install/uninstall hooks; the tap repository and its write credential; the evidence that a broken cask fails before a user hits it and that a failed tap push is recoverable.

**Out of scope:** homebrew-core submission (deferred as BREW-07); stapling / offline-safe first launch (deferred as DIST-06); `codegraph upgrade`'s brew detection and refusal (Phase 4 — this phase only lays the sentinel the ROADMAP names).

</domain>

<decisions>
## Implementation Decisions

### Man page generation (BREW-04)

- **D-01:** Man pages are generated **by the binary itself** — add `github.com/spf13/cobra/doc` and a `codegraph man <dir>` subcommand. Rejected: a `tools/`-only generator (would make man pages a release-time artifact, which conflicts with the install-time delivery chosen in D-05) and a hand-rolled roff writer (owning a roff formatter for no gain). — **Reversibility:** costly — once a release publishes man pages and a cask hook invokes `codegraph man`, every installed cask depends on that command existing; removing it breaks reinstall/upgrade for users on published casks until the tap regenerates.
- **D-02:** The command is **hidden** (`Hidden: true`) and takes an output **directory** argument. Not visible in `--help`, so the FLAG-PARITY divergence footprint is one documented hidden command rather than a new public surface. Rejected: a visible command recorded as a Go-only divergence (the `githooks` treatment), and roff-to-stdout (a multi-page set wants separate files).
- **D-03:** `go-md2man/v2` and `blackfriday/v2` are admitted as **ordinary dependencies**, covered by the existing `govulncheck` + SBOM pipeline like every other dep. No separate written supply-chain audit (the v0.1 Phase 5 `[SUS]` grammar treatment was **not** applied here). Note for the planner: both currently appear in `go.sum` only as `/go.mod` hashes — importing `cobra/doc` promotes them to compiled deps of the shipped binary, entering the SBOM and govulncheck's reachable set for the first time.
- **D-04:** Emit the **full command tree** (`GenManTree`) — `codegraph.1` plus one page per subcommand, ~27 files. Satisfies criterion 2's "a new subcommand appears without anyone editing a committed file" most literally. Rejected: a single root page.

### Cask delivery shape (BREW-03, BREW-04)

- **D-05:** Man pages reach the user via **`homebrew_casks.hooks.post.install`**, which runs the *installed binary* to generate them into the Homebrew prefix. **D-16 in `.goreleaser.yaml:158-168` stays intact as written** — nothing is added to the `zip` archive's file set, and no `files:` override is introduced. Rejected: shipping man pages in the zip and listing them under `manpages:` (would have required amending D-16), and doing both (the two-writers shape this repo resolved by deletion during the checksums collision).
  - **Known consequence, deliberately accepted:** brew does **not** track files a hook writes, so they must be removed explicitly (D-07).
  - **Constraint for research:** `homebrew_casks.manpages:` takes paths *inside the downloaded archive*, and a cask downloads exactly one artifact per platform — so a separate man-only archive is not attachable. The hook is the only mechanism consistent with D-16.
- **D-06:** Shell completions use GoReleaser's native **`generate_completions_from_executable`** with `shell_parameter_format: cobra` and shells `[bash, zsh, fish]`. **Verified during discussion against the real `dist/` binary:** `codegraph completion bash` already works — Cobra 1.10.2 auto-registers the `completion` command and `internal/cli/root.go` never sets `CompletionOptions.DisableDefaultCmd`. No new CLI work is needed for BREW-03. Rejected: shipping completion files in the zip (gives up the structural form of the "no committed completion file" guarantee).
- **D-07:** Generated artifacts are removed by **`hooks.post.uninstall`** (a `system_command` rm), symmetric with how they were created — one matched pair in the config. Rejected: the declarative `uninstall: delete:` stanza and `zap: trash:` (the latter is Homebrew's convention for *user* data, not for artifacts we generated).
- **D-08:** The post-install hook **also writes the Phase-4 brew-install sentinel**, as the ROADMAP's Phase 3 Notes explicitly names it ("the most robust signal Phase 4 can key on") — scoped work, not creep. The post-uninstall hook removes it. — **Reversibility:** costly — once users have installs carrying a sentinel of a given shape, changing its location or format leaves Phase 4's detection blind for those users until they reinstall.
  - **Open for Phase 4, not decided here:** whether Phase 4 keys on the sentinel *instead of* or *alongside* resolved-symlink Cellar detection. Phase 4's own criteria demand detection across `/opt/homebrew`, `/usr/local`, a custom prefix **and linuxbrew**, which a sentinel written by this cask cannot cover on its own.

### The broken-cask gate (BREW-05)

- **D-09:** **BREW-05 as written is not implementable.** `test do` is a **formula** stanza; the Homebrew Cask DSL has `preflight`/`postflight`/`uninstall`/`zap`/`caveats` and no `test`, `brew test` operates only on formulae, and GoReleaser's `homebrew_casks:` reference exposes no `test` field. This is residue from the pre-correction scoping that assumed `brews:` — the fourth falsified scoping assumption in this milestone. **BREW-05's wording in `REQUIREMENTS.md` and ROADMAP criterion 3 must be amended to name the mechanism actually available.**
- **D-10:** The **post-install hook is the gate.** Because it executes the installed binary (`codegraph man`, plus the version check in D-11), a binary that cannot run — wrong arch, failed notarization, corrupt download — makes `brew install` fail loudly for the very first user, before they ever run a command. That is BREW-05's stated property, delivered structurally rather than by a separate check. Rejected: a post-release CI `brew install` job, both together, and `brew audit --cask` (which never executes the binary and therefore cannot catch the failure BREW-05 names).
- **D-11:** The hook carries **both positive assertions** — the man output directory is non-empty (e.g. `codegraph.1` present) **and** the binary's reported version equals the cask version. These cover two genuinely different failures: "didn't run" and "ran but is the wrong artifact". **This directly discharges repo rule `84d1gfpywd`** (a guard MUST carry a positive assertion that it did its work): a hook that exits 0 having produced nothing would otherwise pass vacuously, which is this repo's named recurring failure.
- **D-12:** The gate is demonstrated **RED by a one-time recorded mutation** against a deliberately broken binary, following the D-07/SIGN-03 precedent. **Accepted cost, stated explicitly:** there is no automated RED re-fire, so D-11's positive assertions are the *only* permanent protection against the hook silently becoming a no-op.
- **D-13:** Criterion 1 ("verified cold, at least one release later, against a cask GoReleaser regenerated rather than the one hand-checked at first publish") is satisfied by cutting **two releases inside the phase** rather than blocking on external cadence or standing up a permanent re-verify job.
  - **Constraint for the planner:** release-please computes versions from Conventional Commits and will not cut a release from zero releasable commits. The second release needs at least one real `feat:`/`fix:` commit — identify what it is at plan time rather than discovering it at the tail. See memory `n2tdt2nb3s`: release-cutting phases invert the gsd-ship ordering, because publishing is a prerequisite of the phase's own final verification.

### Tap, credential, and recovery (BREW-01, BREW-02, BREW-06)

- **D-14:** The tap is **`seanb4t/homebrew-tap`**, cask name **`codegraph`**, so the published contract is `brew tap seanb4t/tap && brew install codegraph`. **The repository does not exist yet** (confirmed 2026-08-09: `gh repo view seanb4t/homebrew-tap` → "Could not resolve to a Repository") — creating it is a prerequisite, not implementation work. — **Reversibility:** one-way — `brew tap seanb4t/tap` is a published user-facing contract the moment the first release ships; renaming the tap or the cask orphans every installed user, who must manually untap and reinstall.
- **D-15:** The tap repo is **bootstrapped minimally** — public, README + LICENSE only. **GoReleaser owns `Casks/codegraph.rb`** and creates it on first release. Nothing hand-written that could drift from what the tool generates. Rejected: a README carrying install instructions (a second place they can drift from the main README) and pre-seeding a placeholder cask (the "second, staler source" problem D-16 exists to prevent, one level up).
- **D-16:** The tap-writing credential is a **GitHub App** installed on `seanb4t/homebrew-tap` alone, minting a short-lived installation token per release. No PAT expiry to babysit; scope is set by installation rather than by a checkbox that can later be widened. Rejected: a fine-grained PAT scoped to the tap (workable, but carries an expiry that becomes an operational item like the Developer ID cert's 2027-02-01 expiry) and a classic `repo`-scoped PAT (fails criterion 5 outright — it *can* write `seanb4t/codegraph-go`).
  - **Flagged for research, deliberately not decided here:** *where* the App token is minted. `release.yml:79-117` records that **exactly one job in the file may hold `id-token: write`**, and that every additional Action inside that job sits within the OIDC token's reach — the token that produces the cosign certificate SAN. Minting via `actions/create-github-app-token` inside that job places a new Action next to the signing identity; minting it in a separate job means passing a live write-token between jobs. Resolve on evidence.
  - **Secret placement:** the App's private key and ID must be **repository** secrets, not environment secrets. Memory `q5yhyebw5k` records the exact failure this avoids: environment-scoped Apple secrets would have shipped an un-notarized release **with a green log**. A missing tap credential is the same silent-degradation class.
- **D-17:** Criterion 5's negative proof — the tap credential **refused** a write to `seanb4t/codegraph-go` — is a **one-time recorded run** (attempt the write, record status and response as committed evidence), consistent with SIGN-03's RED baseline and the D-07 mis-order mutation. Rejected: a permanent CI assertion (a deliberate unauthorized-write attempt on every release) and asserting App installation scope via the API (proves configuration rather than behavior).
- **D-18:** ~~The deliberately-failed tap push is reproduced by a local dry-run (`--snapshot`).~~ **FALSIFIED AND REPLACED 2026-08-09, before planning was drawn over it.**

  **Why it was falsified — verified against the pinned module's own source, not documentation:**
  - `goreleaser/v2@v2.17.1` `cmd/release.go:161-162` — `if ctx.Snapshot { skips.Set(ctx, skips.Publish, skips.Announce, skips.Validate) }`
  - `internal/pipe/publish/publish.go` — `cask.Pipe{}` is a member of the **Publish** pipeline, listed after `release.Pipe{}` under the literal source comment *"brew et al use the release URL, so, they should be last"*

  A `--snapshot` run therefore **structurally cannot reach the cask-push path at all**. `HomebrewCask.SkipUpload` (`config.go:226`) exists but *prevents* the push rather than failing it, so it is no substitute.

  **D-18R (replacement, maintainer decision 2026-08-09): STRUCTURAL ARGUMENT ONLY.** The failure-and-recovery mechanism is **not** demonstrated by an executed run. It rests on the source-level ordering fact above: because `cask.Pipe{}` runs last in Publish, a tap-push failure cannot corrupt the Release, its cosign bundles, SBOMs or SLSA provenance, all of which are complete before the cask pipe starts. Rejected: a real `goreleaser release` against a throwaway scratch repo with a bad tap token (the house pattern from 01-06's throwaway cosign key and 02-04's guarded notarize rehearsal), and withholding the token on the real release.

  **ACCEPTED RISK, NAMED EXPLICITLY AND NOT SILENT.** The orchestrator raised, and the maintainer accepted, that this leaves BREW-06's mechanism half as **a check that has never been observed to fire** — the failure mode this repo names as its own recurring one (`rez50fp4hp`, `yctys69cke`, `vep9bdqkw9`, rule `84d1gfpywd`). Downstream agents MUST carry this forward as a stated limitation rather than reporting BREW-06 as demonstrated. The verifier should expect **no executed evidence** for this leg and must not manufacture a substitute that passes vacuously.
- **D-19:** **Criterion 4 is split**, and its wording in ROADMAP.md must be amended to say so:
  1. **Failure-and-recovery mechanism** — **argued structurally per D-18R, not executed.** Recorded as an accepted limitation, citing the `publish.go` pipeline ordering as the evidence.
  2. **Release integrity** — proven by the **existing permanent post-release verification**, which already re-verifies cosign bundles, SLSA provenance and the asset set against re-downloaded artifacts on every release. This half IS executed evidence.

  Rejected: amending criterion 4 down without saying where the property went, and inducing the failure on a real release.

### Claude's Discretion

- The sentinel's exact filename, location and content format (subject to D-08's constraint that Phase 4 must be able to read it, and that it is removed by post-uninstall).
- The man page output directory the hook targets within `HOMEBREW_PREFIX`, and portability across `/opt/homebrew` vs `/usr/local`.
- Cask metadata not discussed: `desc`, `homepage`, `livecheck`, `auto_updates`, `depends_on`.
- Whether the cask covers darwin only or also linuxbrew.
- Hook idempotency on reinstall/upgrade.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` § "Phase 3: Homebrew Tap & Cask" — goal, dependencies, the five success criteria, and the Notes block naming `homebrew_casks:`, the cross-repo token constraint, and the `hooks.post.install` sentinel. **Criteria 3 and 4 need amendment per D-09 and D-19.**
- `.planning/REQUIREMENTS.md` — BREW-01 … BREW-06 (lines 31-36); DIST-06 and BREW-07 as explicit deferrals (lines 50-51); the "Falsified assumptions" table (lines 57-59). **BREW-05's wording needs amendment per D-09.**
- `.planning/PROJECT.md` § "Key Decisions" — the `homebrew_casks:`-not-`brews:` correction, the own-tap-not-homebrew-core decision, archives-alongside-raw-binaries, notarize-but-do-not-staple, and the `codegraph upgrade` refusal that Phase 4 implements.

### The release pipeline this phase extends
- `.goreleaser.yaml` — `archives:` (lines 133-170), including the byte-frozen `raw` entry (D-02/Finding 1) and **D-16's binding constraint at lines 158-168** that completions and man pages stay out of the `zip`; `checksum:` (line 173, D-12's 8-payload scope); `notarize:` (line 250); `signs:` (line 325); `sboms:` (line 365); `release:` (line 389, `replace_existing_artifacts: true` for patch-forward recovery). The `homebrew_casks:` block does not exist yet.
- `.github/workflows/release.yml` — lines 79-117 record that exactly one job may hold `id-token: write` and why; lines 195-200 show the current secret surface (`GITHUB_TOKEN` plus the five `MACOS_*`). Line 168 notes GoReleaser's release pipe reads `GITHUB_TOKEN`, not `GH_TOKEN`.
- `Taskfile.yml` — the single definition of every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it). Line 1755 already anticipates "Phase 3 Homebrew cask artifacts". Any new gate belongs in a task target, not inline in a workflow — see memory `yctys69cke`: grepping a workflow for enforcement that actually lives in the Taskfile returns zero hits and reads as absence.
- `docs/RELEASE.md` and `docs/RELEASE-PROCEDURES.md` — the published guarantee and the maintainer runbook; both will need the brew install path added.

### The CLI surface being extended
- `internal/cli/root.go` — `newRootCmd()` and the 25-command `AddCommand` list; where `newManCmd()` attaches. Note it does **not** set `CompletionOptions.DisableDefaultCmd`, which is why `completion` already works.
- `internal/cli/flag_parity_test.go` and `docs/FLAG-PARITY.md` — the surface-parity audit a new command must be recorded against.
- `internal/cli/githooks.go` — the precedent for a documented Go-only command TS CodeGraph does not have.

### Phase 4's consumer
- `internal/upgrade/` — where the brew detection and refusal land. `verify.go`'s `releaseWorkflowRefPattern` anchors the cosign SAN to `.github/workflows/release.yml@refs/tags/v[0-9]*`; renaming the workflow file or changing the tag trigger breaks every user's upgrade.

### External documentation (fetched 2026-08-09, Context7 `/websites/goreleaser`)
- `https://goreleaser.com/customization/homebrew` — the `homebrew_casks:` reference: `manpages:`, `completions:`, `generate_completions_from_executable` (with `shell_parameter_format: cobra`), `hooks.pre/post.install/uninstall`, `uninstall:`, `zap:`, `url:`, `custom_block:`. **Confirmed: there is no `test` field.**
- `https://docs.brew.sh/Cask-Cookbook` — the Cask DSL stanza list, the authority for D-09.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Cobra's built-in `completion` command** — already live and verified against `./dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph`; `completion bash` emits a V2 script and exits 0. BREW-03 needs no CLI work.
- **`tools/bench`, `tools/mcpaudit`, `tools/transcriptfreeze`** — the established shape for a build-time generator, had D-01 gone the other way. Left unused by this phase's decisions, but the pattern exists.
- **`internal/cli/githooks.go`** — the closest analog for adding a command TS CodeGraph lacks, including how the divergence was documented.
- **The existing post-release verification job** — D-19 assigns criterion 4's release-integrity half to it, so it is a load-bearing dependency of this phase's evidence, not just adjacent CI.

### Established Patterns
- **`Taskfile.yml` is the single definition of every CI job body**, enforced by `TestWorkflowRunBodiesInvokeTask`. New verification goes in a task target that the workflow invokes.
- **Gates are demonstrated RED before being trusted green** — D-12 and D-17 both follow it; D-19's split follows it by refusing to let a local run stand in for a published-artifact observation.
- **Repo rule `84d1gfpywd`: a guard MUST carry a positive assertion that it did its work.** D-11 is this phase's application.
- **`.goreleaser.yaml` carries its rationale inline** as comments naming the decision id and the test that holds it (see the `raw` archive entry, `checksum:`, `release:`). The `homebrew_casks:` block should match that density — it is the file's convention, not decoration.
- **Shape tests pin config invariants** — `TestRawArchiveEntryStaysBinaryFormat`, `TestChecksumCoversRawAndZipIdsOnly`, `TestReleaseBlockIsRerunIdempotent`, `TestReleaseBlockDoesNotRewriteReleaseBody` all live in `goreleaser_shape_test.go`. A `homebrew_casks:` block wants equivalent coverage.

### Integration Points
- `.goreleaser.yaml` — a new top-level `homebrew_casks:` block consuming the existing `zip` archive id.
- `.github/workflows/release.yml` — the App-token mint and its placement relative to the single `id-token: write` job (D-16, flagged).
- `internal/cli/root.go` — `newManCmd()` registration.
- `go.mod` / `go.sum` — `go-md2man/v2` and `blackfriday/v2` promoted from module-graph-only to compiled deps.
- The Phase-4 sentinel written by the cask hook (D-08) → read by `internal/upgrade`.

</code_context>

<specifics>
## Specific Ideas

- The published contract is exactly `brew tap seanb4t/tap && brew install codegraph`.
- "A broken cask fails before a user hits it" is delivered by making the *install itself* execute the binary — not by a separate check that runs somewhere else and could drift out of the path.
- The one-time-recorded-mutation pattern used for SIGN-03 and the D-07 mis-order mutation is the accepted house style for proving a gate can fire; D-12 and D-17 both adopt it deliberately, with the regression-protection cost stated rather than glossed.

</specifics>

<deferred>
## Deferred Ideas

- **homebrew-core submission** — already recorded as BREW-07 in REQUIREMENTS.md; unchanged by this discussion.
- **Stapling / offline-safe first launch** — already recorded as DIST-06; unchanged.
- **A permanent automated cold-install verification job** — considered and not taken for criterion 1 (D-13 cuts two releases instead). Worth revisiting if the phase's second release proves awkward to arrange, or as ongoing release hygiene after the milestone.
- **A permanent CI assertion that the tap credential cannot write `codegraph-go`** — considered and not taken (D-17 records once instead). The gap it leaves is real: if the App's installation scope is later widened, nothing notices.
- **Phase 4's detection strategy** (sentinel vs resolved-symlink Cellar detection vs both) — this phase only writes the sentinel; the choice belongs to Phase 4, whose criteria demand linuxbrew and custom-prefix coverage a sentinel alone cannot give.

### Reviewed Todos (not folded)

Surfaced by `todo.match-phase 3` and reviewed; none folded into this phase's scope.

- **`release:dry-run-signed`'s additions-only diff guard passes vacuously when the awk anchor stops matching** (release, score 0.9) — genuinely adjacent, since D-18 uses a local dry-run and this phase adds a new pipe to `.goreleaser.yaml`. Not folded: it is a defect in an existing Phase-2 guard, not work this phase's requirements cover. **The planner should be aware it is unfixed if it leans on `release:dry-run-signed` for D-18's evidence.**
- **`verify:self-upgrade` downloads and executes a prior release binary with no signature verification** (release, score 0.9) — pre-existing exposure accepted as T-02-24 in Phase 2. Out of scope.
- **`post-release-verify.yml`'s event-aware conclusion guard has no test asserting it** (ci, score 0.6) — Phase 2 residue. Relevant only because D-19 assigns criterion 4's release-integrity half to post-release verification; not folded.
- **Wire oracle `toolslist-repeat` response ordering flake** (mcp, score 0.6) — unrelated to this phase.

</deferred>

---

*Phase: 3-Homebrew Tap & Cask*
*Context gathered: 2026-08-09*
