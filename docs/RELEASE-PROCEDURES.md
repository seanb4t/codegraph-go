# Cutting a release (maintainer runbook)

This document is the maintainer's step-by-step runbook for cutting a
`codegraph-go` release — pre-tag checks, tag conventions, what the tag push
triggers, the LOCKED `verify.go` contract, post-release verification,
rollback, and the commit-signing fallback for automated pipeline commits.
It complements `docs/RELEASE.md`, which documents how a *user* verifies a
downloaded artifact after the fact; this doc is the *maintainer's* side —
how the release gets cut in the first place.

## 1. Pre-tag gate (mandatory — D-09)

Before tagging **anything** (rc or stable), run the following for **all 4**
release targets. This is the exact check that would have caught the v0.1
`rc.1` failure: a **linux-only** `go.sum` hash (`prometheus/procfs`) that
was invisible on a darwin-only dev machine and to green local CI.

```sh
for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null \
    && echo "OK   $pair" \
    || echo "FAIL $pair"
done
```

All 4 lines must print `OK`. A single `FAIL` means a platform-specific
`go.sum`/build-constraint issue exists that local CI (which only runs on
your dev machine's own `GOOS`/`GOARCH`) cannot see — do not tag until every
target is green.

> **Now also automated.** This exact sweep runs automatically as the
> `pretag-gate` job in `.github/workflows/release-please.yml` on every push
> to `main`, gating the `release-please` job behind it via `needs:`. The
> job invokes it as `task check:cross` (D-15) — the sweep's one definition
> lives in `Taskfile.yml`, not inlined in the workflow. Run `task
> check:cross` locally for the same check the manual command block above
> performs, before you ever push to `main` — but the enforcement point is
> now CI, not this document. No human needs to remember to run this by hand
> anymore.

## 2. Tag conventions

`release.yml` triggers on pushes matching `v[0-9]*`. Three distinct shapes:

| Tag shape | Example | Meaning | Triggers `release.yml`? |
|---|---|---|---|
| Release candidate | `v0.0.0-rc.3` | Prerelease — the `-` suffix makes GoReleaser/GitHub mark it a prerelease | Yes |
| Stable | `v1.0.0` | Full release, no `-` suffix — becomes `codegraph upgrade`'s resolved "latest" | Yes |
| Internal milestone marker | `milestone-v1.0` | Documentation/planning marker only | **No** — deliberately does not match `v[0-9]*` |

Never use a `milestone-v*`-style name for anything you actually want
`release.yml` to build and sign — it is intentionally shaped to never fire.
Conversely, never push a real `v[0-9]*` tag casually — every one of them
fires the full signed build/archive/checksum/sign/SBOM/publish/attest
pipeline (one job, described in full in §4 below).

## 3. Branch/tag model

1. Land all phase work on the integration branch
   (`gsd/v1.0-drop-in-parity-human-ux` for this milestone).
2. The pre-tag gate (§1) now also runs automatically on every push to
   `main` as release-please.yml's `pretag-gate` job (see the note added to
   §1 above) — release-please can never open or merge a release PR from a
   `main` state that fails it.
3. **Fast-forward** the integration branch onto `main`, preserving its full
   history — do **not** squash.
4. The tag is created by release-please when its release PR merges (§4) —
   never by hand.

**Why fast-forward, not squash — the evidence, not a preference:**
`main` contains **zero merge commits** (`git log --merges main` is empty)
— the v0.1 milestone landed this same way — and `main` is an ancestor of
the integration branch today, so a fast-forward is possible as-is. At the
time this section was written the branch was roughly 477 commits ahead of
`main`, of which roughly 160 are `feat`/`fix`/`perf` and roughly 217 are
planning-only `docs(...)` commits. Under fast-forward, release-please walks
all of those commits and generates a genuinely rich changelog from the 160
substantive ones; its default `changelog-sections` hide `docs:`/`chore:`/
`ci:`/`test:` automatically, so the planning commits are filtered out
without any manual curation. A squash-merge would collapse all of that
history into a single commit message and reduce the changelog to one line
— discarding exactly the input release-please exists to consume.

**This supersedes Phase 8's `08-CONTEXT.md` D-08 "squash-merge to `main`"
wording**, on repo evidence rather than preference (also independently
recorded in engram `8sa948y0g4`: "fast-forward merge not squash"). Only the
merge *mechanic* is superseded — D-08's substance (integration branch →
`main` → tag on `main`) is honored in full; the milestone's work still
lands on `main` before the tag exists, and the tag is still cut against
`main`, never a feature/integration branch.

## 4. Cutting a release (now automated by release-please)

The release-PR merge is the trigger, not a hand-run `git tag`:

1. Merge the integration branch to `main` (§3) — a normal fast-forward,
   not a PR.
2. `.github/workflows/release-please.yml` runs on that push: its
   `pretag-gate` job (§1's automated sweep) must pass before
   `release-please` runs. release-please then opens or updates a
   `chore(main): release X.Y.Z` pull request proposing the next version and
   the `CHANGELOG.md` entry.
3. Review the PR's proposed version bump and changelog for accuracy.
4. Merge the PR.
5. release-please creates the tag **and** the GitHub Release in a single
   API call — the Release object exists the instant the tag ref does, with
   release-please's own changelog as the release body.
6. The tag push, authored by the GitHub App token (not the default
   `GITHUB_TOKEN` — App-authored refs *do* trigger downstream workflow
   runs, which is exactly why this works), fires `release.yml`. It has
   exactly ONE job, `release`, running natively on a real Apple-Silicon
   macOS profile (`namespace-profile-macos-6x14-tahoe`) — no separate
   `build`/`assemble`/`provenance` jobs and no CI-artifact round trip
   between them (that three-job topology was collapsed by plan 01-03/01-04
   into a single `goreleaser release --clean` invocation, run via
   `task release:goreleaser`):
   - **Build** — compiles all 4 `(GOOS,GOARCH)` targets: darwin native
     (Xcode clang, both arches, on this same macOS runner) and BOTH linux
     legs cross-compiled via `zig cc`/`zig c++` from this one host
     (D-01/D-02, REL-05 — proven on real hardware, not just a green build
     exit code).
   - **Archive** — GoReleaser's `archives:` pipe produces both the raw
     per-platform binary `codegraph upgrade` swaps in and a `.zip` archive
     of the same binary, for browser downloads and the Homebrew tap
     (REL-09); the raw binary's bytes are byte-unchanged by the archive
     step.
   - **Checksum** — GoReleaser's `checksum:` pipe writes the shared
     `codegraph_<tag>_checksums.txt`, covering exactly the 8 raw-binary +
     `.zip` payloads (D-12) — the ONLY writer of that file (REL-07).
   - **Sign** — `binary_signs:` shells out to cosign keyless
     (`sign-blob --bundle=...`) per binary — internal/upgrade hashes the
     downloaded binary itself, not a checksums file, so per-binary signing
     is required, same trust boundary as before.
   - **SBOM** — `sboms:` shells out to `syft`, per binary, publishing an
     SPDX SBOM alongside each.
   - **Publish** — GoReleaser's declarative `release:` pipe uploads into
     the Release release-please already created, idempotently
     (`replace_existing_artifacts: true`), without rewriting
     release-please's changelog body or prerelease flag.
   - **Attest** — `actions/attest-build-provenance` (GitHub's first-party
     attestor) runs as the job's last step, over the SAME 8 payloads the
     checksums file covers, publishing through GitHub's Attestations API
     rather than as a release asset (D-09 — replaces the third-party SLSA
     generic generator entirely; see `docs/RELEASE.md` §1(b) for the
     `gh attestation verify` command that replaces `slsa-verifier
     verify-artifact` for releases cut by this pipeline).

**The version is computed, not chosen.** The bump comes from accumulated
Conventional Commits and that is the intended, default behaviour: while the
project is on a `0.x` baseline, a `feat:` bumps the minor, and reaching `1.0.0`
happens when the commits say so — not because a milestone is named "v1.0".

> **Project directive (2026-07-29):** do **not** manufacture a version.
> No `Release-As:` footer, no `release-as` config key, and no breaking-change
> marker authored for the purpose of producing a bump. If you believe a release
> needs a version the tool would not compute, that is a conversation to have
> before the cut, not a footer to add during it. See `09-CONTEXT.md` D-06R.

**The override exists, for completeness.** A one-shot `Release-As:` footer on an
empty commit forces a specific version for a single cut:

```sh
git commit --allow-empty -m "chore: release X.Y.Z" -m "Release-As: X.Y.Z"
```

Two constraints if it is ever used. It must begin a line to parse as a footer —
mentioning `Release-As:` mid-sentence in a commit body does nothing (two commits
in this repo do exactly that while documenting this section, harmlessly). And it
must never move into `release-please-config.json`'s `release-as` field: a
config-level setting is sticky and pins every subsequent release to that version
until removed.

**Expected silent no-op:** if there are no release-worthy commits since the
last release, release-please opens **no PR at all**. This is expected, not
a broken pipeline — check the `release-please.yml` workflow run's log
rather than assuming a failure and reaching for a manual tag.

**Escape hatch — `rc` prereleases only (D-10):** the hand-pushed tag is
retained, explicitly labelled, for prerelease cuts only:

```sh
git checkout main
git pull
git tag v0.0.0-rc.N
git push origin v0.0.0-rc.N
```

A `v0.0.0-rc.N` tag pushed by hand still matches `release.yml`'s
`v[0-9]*` trigger and still fires the signed build; since no release-please
Release exists yet for it, GoReleaser's declarative `release:` pipe
**creates** the Release itself (marked prerelease, per `prerelease: auto`)
rather than uploading into one that already exists. This is the only path
in this repo where a human still runs `git tag` directly — stable releases
never use it.

## 5. The `verify.go` LOCKED contract

`internal/upgrade/verify.go` pins the exact identity `codegraph upgrade`
trusts before ever swapping the installed binary. Reproduced **verbatim**
(do not paraphrase these values — copy them from `verify.go` if this
runbook and the source ever disagree, the source wins):

```go
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
)
```

`releaseWorkflowRefPattern` is a **full-match, anchored** (`^...$`) regexp —
not a prefix match. This is deliberate: an unanchored/prefix-only pattern
would authorize a signature produced by *any* workflow in this repo
(including a much weaker trust boundary, e.g. a `pull_request`-triggered CI
run), not just the intended release workflow. **Never weaken this to a
prefix match "for convenience."**

**The rule:** if you ever rename `.github/workflows/release.yml`, change
its `v[0-9]*` tag trigger, or change the cosign signing identity, you
**must** update `releaseOIDCIssuer`/`releaseRepoSlug`/
`releaseWorkflowRefPattern` in `internal/upgrade/verify.go` in the **same
commit**. Failing to do so silently breaks `codegraph upgrade` for every
existing user — their installed binary will reject every future release's
signature as untrusted, with no clear error pointing back at this cause.
This runbook never edits `verify.go`; it only documents the contract those
constants encode.

## 6. Post-release verification

After the release publishes, verify it the same way an end user would (see
`docs/RELEASE.md` §1 for the full user-facing walkthrough):

```sh
# a) cosign keyless signature (per-binary) — UNCHANGED by the attestor swap
cosign verify-blob \
  --bundle codegraph_<tag>_<goos>_<goarch>.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp \
    '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  codegraph_<tag>_<goos>_<goarch>

# b) build provenance — GitHub's first-party attestor (D-09), published
#    through GitHub's Attestations API, over all 8 payloads (4 raw
#    binaries + 4 .zip archives)
gh attestation verify codegraph_<tag>_<goos>_<goarch> -R seanb4t/codegraph-go
```

> **(b) is for releases from the migrated pipeline onward only.** Releases
> up to and including the last pre-migration tag published a separate
> `codegraph_<tag>.intoto.jsonl` (`multiple.intoto.jsonl` for v0.2.0 and
> earlier), verifiable with `slsa-verifier verify-artifact
> --provenance-path <bundle> --source-uri github.com/seanb4t/codegraph-go
> --source-tag <tag> <binary>` instead — that command cannot verify the new
> attestation format (D-10), and the new command cannot verify the old
> one. See `docs/RELEASE.md` §1(b) for the full historical note.
>
> **Corrected 2026-08-01 (applies to the pre-migration generator only).**
> (b) previously named `codegraph_<tag>_checksums.txt.intoto.jsonl` and
> passed the checksums file as the artifact. No such file was published,
> and the checksums file was not an attested subject — the platform
> binaries were, sharing one `multiple.intoto.jsonl`. The old command
> returned `FAILED: artifact hash does not match provenance subject`
> against a valid release. Found while verifying `v0.2.0`;
> `docs/RELEASE.md` §1(b) carried the same error and was corrected too.
> Retained for anyone verifying a pre-migration release.

Both commands must succeed for at least one platform's artifacts before
considering the release verified. A signature from any other issuer, any
other workflow file, or any non-tag trigger will fail (a) — as it should.

> **Mandatory, not optional, for the first automated cut.** Running these
> two commands is not optional for the first release-please-automated
> release and for the disposable live proof (§9) that precedes it — it is
> the only evidence that the cosign identity survived the App-token
> rewiring, and it is the direct evidence REL-02's second half is verified
> against. Do not skip it because the pipeline "should" still be signing
> correctly; prove it.

## 7. Rollback / cleanup

**Recovery posture is PATCH FORWARD (D-07) — this section governs
`rc`-prerelease cleanup only, never a stable (`vX.Y.Z`) release.** Do
**not** delete or re-push a published tag or Release to "fix" a bad cut.
Deleting and re-pushing a tag re-fires `release.yml` — a human touching tag
authority release-please owns (D-06R) — and, more concretely, would destroy
any un-notarized or otherwise deliberately-preserved asset a later phase
depends on as its baseline (the exact mistake this note exists to prevent).
A wrong stable release is fixed by the **next** release-please patch
release, not by rewriting history. This section's delete-and-retry
procedure below applies **only** to a disposable `rc`-shaped tag (§4's
escape hatch, §9's disposable live proof) — never to a real `vX.Y.Z`.

### 7.1 Preserved baselines — do not delete or replace (dated)

The abstract warning above has a concrete instance. Two releases are load
bearing for later phases and must survive untouched:

| Release | Why it is preserved | Recorded |
|---|---|---|
| **`v0.5.1`** | Its darwin assets (`codegraph_v0.5.1_darwin_arm64`, `codegraph_v0.5.1_darwin_amd64`) are deliberately **un-notarized** and are Phase 2's **SIGN-03 RED baseline**. Once notarization lands they cannot be recreated — a notarized build is a different artifact. Do not delete, replace, or re-upload them. The recorded RED observation against these assets lives in [`02-EVIDENCE.md`](../.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md). | 2026-08-09, plan 01-05 |
| **`v0.5.0`** | Tagged but published **zero assets** — its pipeline aborted on a Taskfile version assertion before `goreleaser release` ran. Kept per D-07 (patch forward, never delete or re-push) and marked **prerelease** so `/releases/latest` skips it and `codegraph upgrade` resolves past it. | 2026-08-09, plan 01-05 |

`v0.5.0` is **not** the SIGN-03 baseline despite being the first v0.5.x tag:
it has no assets at all. The baseline is `v0.5.1`.

Because `v0.5.0` stays in the release list permanently, any tooling that
walks releases must tolerate a zero-asset entry. `verify:self-upgrade`
already does — it drops both drafts and asset-less releases, and classifies
"stable" using the GitHub `prerelease` flag rather than the semver suffix
alone. Follow that precedent rather than rediscovering it.

For an `rc` cleanup, a release-please cut leaves **up to three** artifacts
behind (an `rc` tag pushed by hand per §4's escape hatch has no
version-bump commit, so only the first two apply there):

1. the tag,
2. the GitHub Release object created for it, and
3. (release-please-cut releases only) the version-bump commit on `main`
   (the `CHANGELOG.md` entry plus the `.release-please-manifest.json`
   bump) together with its merged release PR.

For a release-please-cut release, deleting the tag alone is not
sufficient — release-please's next run walks `main`'s history and, seeing
the manifest already bumped, would believe the version already shipped.

**1. Delete the Release and its tag (`rc` cuts only):**

```sh
gh release delete v0.0.0-rc.N --cleanup-tag -y
```

If the tag was pushed but no Release was ever created (e.g. the build
failed before GoReleaser's `release:` pipe ran), `gh release delete` has
nothing to delete — in that case the tag itself is the only artifact left,
and it is still an `rc` tag pushed by hand under §4's escape hatch, not a
release-please-authored ref, so deleting just the tag is the correct and
only step; skip straight to confirming with `git tag -l 'v0.0.0-rc.N'` and
`git ls-remote --tags origin 'v0.0.0-rc.N'` that nothing remains, rather
than assuming a Release exists to delete.

Nothing is published to the GitHub release until every step in the single
`release` job succeeds — a failed build produces no Sigstore
transparency-log entries and no public GitHub release, so deletion is a
no-op beyond the tag itself when the failure happened before GoReleaser's
`release:` pipe ran.

**2. Revert the release commit on `main`:** so release-please's next run
does not believe the version already shipped, revert the merged
`chore(main): release X.Y.Z` commit (the `CHANGELOG.md` entry plus the
manifest bump) with a normal `git revert` on `main`.

**3. Check the `autorelease` label:** release-please re-labels its release
PR `autorelease: tagged` once it believes a cut succeeded. A
partially-completed cut may leave that label in place, which can prevent
release-please from proposing a fresh release PR on its next run — remove
the `autorelease: tagged` label from the PR (or close and let release-please
re-open it) if a retry doesn't produce a new PR as expected.

Never reuse a tag name after deleting it for a **stable** (`vX.Y.Z`) release
— once `codegraph upgrade` has resolved a version as "latest," retagging
the same name risks a confusing mismatch between what a user already
verified and what the tag now points to. Prereleases (`-rc.N`) are safer to
delete/retag since they are explicitly transient by convention.

This same three-step procedure (delete release+tag, revert the release
commit, check the `autorelease` label) is what tears down the disposable
live-proof release described in §9.

## 8. Commit-signing fallback (automated pipeline commits only)

Local commit signing (e.g. 1Password's git-signing integration) has been
observed to intermittently fail during automated/pipeline commit sequences
(`failed to fill/write commit object`). Per repo rule `xmz3xknbj0`: **never
bypass signing when a human at the keyboard asks for a commit.** The
`-c commit.gpgsign=false` fallback is permitted **only** for commits made
by an automated agent/pipeline (this runbook's own release-readiness
commits, CI-driven commits, etc.) where retrying the signed commit is not
productive:

```sh
git -c commit.gpgsign=false commit -m "..."
```

Do not reach for this fallback as a first resort, and never use it on a
commit the user themselves is making interactively.

## 9. GitHub App prerequisite (D-02)

**Why an App, not the default token, is required:** a tag pushed with the
workflow's default `GITHUB_TOKEN` does not start other workflow runs — so
`release.yml` would simply never fire on release-please's tag. A GitHub
App installation token does trigger downstream workflow runs. This is not
optional plumbing; it is the single mechanism that makes §4's automated
flow work at all.

This is a one-time, maintainer-manual setup procedure. Follow it once per
repo (or once per organization if the App is installed org-wide):

1. **Create the App** under the account's Developer settings
   (`Settings → Developer settings → GitHub Apps → New GitHub App`). Any
   name is fine; it needs no webhook, no user-facing UI, and no public
   listing.
2. **Grant it exactly three installation permissions:**
   - **Contents: Read and write** — needed to push the version-bump commit
     and create the tag.
   - **Pull requests: Read and write** — needed to open/update the
     `chore(main): release X.Y.Z` PR.
   - **Issues: Read and write** — needed because release-please labels its
     PR `autorelease: pending` / `autorelease: tagged`, and GitHub governs
     PR labels under the Issues API scope in its App permission model, not
     under Pull requests.
3. **Install the App** on `seanb4t/codegraph-go`. A single-purpose,
   per-repo App is the ideal — but reusing an existing release-automation
   App already installed on other repositories is acceptable practice as
   long as its installation permissions are still exactly the three above.
   Reusing a shared App does change the blast radius of a key leak (every
   repository the App is installed on, not just this one), so record which
   shape was used (new vs. reused) when this step is actually performed —
   the `repository_selection` field on the App's own settings page (or via
   an authenticated `/apps/<slug>/installation`-style call) is the only way
   to confirm single-repo vs. all-repos scope; the unauthenticated
   `/apps/<slug>` endpoint does not expose it.
4. **Generate a private key** for the App and store the **full PEM** as the
   repository secret `APP_PRIVATE_KEY`. Store the App's numeric **App ID**
   as the repository secret `APP_ID`.

**These are two different permission systems — do not conflate them.** The
App's *installation* permissions (step 2 above, set in the App's own
settings) authorize everything release-please does with the minted App
token. The `permissions:` block inside `.github/workflows/release-please.yml`
governs only the default `GITHUB_TOKEN` and is a separate, unrelated scope.
Configuring one correctly while under-scoping the other does not fail at
config time — it produces a runtime authorization failure (a 403) the first
time release-please actually tries to label a PR or push a commit, not
before.

**Deprecated-input note:** `actions/create-github-app-token`'s *current*
required input is `client-id` (the App's Client ID string). The `app-id`
input (the numeric App ID) still works but is deprecated-but-accepted. This
runbook's secret names (`APP_ID`, `APP_PRIVATE_KEY`) wire to the `app-id`
input today. Migrating to `client-id` later means **re-seeding the secret
with the App's Client ID**, not reusing the numeric App ID value — the two
are not interchangeable strings.

**Pre-flight commands:**

```sh
# Confirm the repo's Actions settings don't have anything unusual blocking
# App-token-authored workflow runs
gh api repos/seanb4t/codegraph-go/actions/permissions

# Confirm the repo's PR-creation workflow permission (see note below)
gh api repos/seanb4t/codegraph-go/actions/permissions/workflow

# Confirm both secrets landed (lists names only, never values)
gh secret list
```

**"Allow GitHub Actions to create and approve pull requests" note:** this
repository-level Actions setting (`can_approve_pull_request_reviews` in the
API above) governs whether the default `GITHUB_TOKEN` may open/approve
pull requests. `release-please-action`'s own README calls it out as a
common blocker — but D-02's whole design point is that release-please
authors its PR with the **App's** installation token, a distinct actor
from `GITHUB_TOKEN`, so this setting is not load-bearing for this
pipeline. Confirm its value here for the record (and prefer it enabled, as
the safer default for this repo generally), but a disabled value does not
by itself block release-please — only an under-scoped App installation
(step 2 above) does that.

**Branch protection note:** the repo has no rulesets today, so this does
not currently apply. If branch protection is ever added to `main`, the App
**must** be added to the bypass-actor list, or release-please will be
unable to push its version-bump commit past the protection rule.

**The disposable live proof.** Because "does an App-token-authored tag
push really fire `release.yml` in this repo's actual Actions
configuration" cannot be proven statically, this pipeline is run once
end-to-end against a disposable, prerelease-shaped tag on a scratch branch
— never against the real `v1.0.0` as the first live test — before the real
`v1.0.0` is ever cut. §6's post-release verification and §7's rollback
procedure both apply to that disposable release exactly as they would to a
real one; §7 is what tears it down afterward.

**Troubleshooting: 401 `A JSON web token could not be decoded`.** This
error, surfaced from the "Mint GitHub App installation token" step, means
the JWT `create-github-app-token` signed with `APP_PRIVATE_KEY` was
rejected — it does **not** by itself distinguish "malformed key" from
"valid key belonging to the wrong App." A key that is well-formed PEM
(parseable, correct bit length, real newlines, no CRLF) still fails
identically if it was copied from a *different* App's key file — an easy
mistake if the maintainer has more than one App's `.pem` in the same
downloads folder. Re-pasting the same wrong key produces the same 401 on
retry, which can look like a stuck/unfixable failure.

To tell the two apart without guessing, run a local identity check against
the candidate key before re-storing the secret:

```sh
# Sign a short-lived JWT with the candidate PEM (any small script/library
# that implements RS256 App-JWT signing works) and call:
curl -s -H "Authorization: Bearer <jwt>" -H "Accept: application/vnd.github+json" \
  https://api.github.com/app
```

A key belonging to the wrong App returns `401`. The correct key returns
`200` with the App's own `slug`/`id`/`owner` — confirm those match the App
this runbook installed in step 3. This is the cheapest way to separate
"bad key material" from "wrong App," and is faster than iterating on
re-pasting/regenerating keys against the same (wrong) App.

## 10. Recorded divergences (Phase 9)

Three places where what shipped diverges from what an earlier decision
document said, each recorded with its source and reason rather than left
to drift silently.

**(a) Roadmap Phase 9 criterion 3 / D-05.** The criterion as originally
written assumed a `goreleaser release` upload model with a
`replace_existing_artifacts: true` setting and no `changelog:` block. This
repo has never run `goreleaser release` — `.goreleaser.yaml`'s `archives:`
and `checksum:` blocks are documented-inert, and signing, SBOM generation,
and publishing all live in `release.yml`'s hand-written `assemble` job
instead, because GoReleaser OSS cannot express per-binary cosign (the
`codegraph upgrade` verifier hashes the binary itself, not a checksums
file), the native darwin/linux two-OS matrix that keeps darwin off a zig
cross-link, or the SLSA generic-generator handoff. The criterion's
*intent* — idempotent artifact replacement — is satisfied instead by §4's
`gh release upload --clobber` step. Recorded as an accepted divergence in
`.planning/ROADMAP.md` (§ below), not dropped.

**(b) D-08 lint file location.** D-08 names `ci.yml` as the destination for
the PR-title conventional-commit lint. What shipped is a dedicated
`.github/workflows/pr-title.yml` instead, because `ci.yml`'s shared
`pull_request` trigger uses GitHub's default event types, which exclude
`edited` — a contributor fixing only a bad title would get no re-run
there. Widening `ci.yml`'s trigger to include `edited` would also re-run
its heavy reproducibility and perf-regression jobs on every title or
description edit, not just code changes. The lint's substance (a blocking
gate, hand-written `grep -E`, title bound via `env:`) is unchanged — only
its file location diverged.

**(c) D-03 (`workflow_dispatch` fallback) is deliberately not
implemented.** The App-token path (D-02, §9) is this phase's primary and
only implemented tag-trigger mechanism. D-03's dispatch-based fallback was
never built, and that is intentional: leaving it out keeps `release.yml`'s
header comment claim — that it triggers *only* on tag pushes — true by
construction, and `internal/upgrade/release_workflow_shape_test.go`'s
`TestReleaseWorkflowTriggerIsTagPushOnly` mechanically enforces that the
file's `on:` block has exactly one trigger. This cannot be skipped by
accident later without that test going red. If a future maintainer needs
Path B because the App becomes unavailable, the full recipe is:

1. Add `workflow_dispatch` **alongside** — never instead of —
   `release.yml`'s existing `push.tags` trigger.
2. Dispatch it **at the tag ref**
   (`gh workflow run release.yml --ref "$TAG"`), never at a branch — a
   branch-ref dispatch produces `@refs/heads/main` in the cosign SAN, which
   is unverifiable by `internal/upgrade/verify.go` and would ship silently
   broken binaries.
3. Add a guard step that hard-fails when `github.ref` does not start with
   `refs/tags/v`.
4. Add a test proving that guard actually fires on a rejecting input — a
   guard that is merely present but never exercised is not a guard (the
   Phase 8 `CR-01`/`WR-02` lesson).
5. Update `release.yml`'s header comment in the **same commit**, so it no
   longer claims a tag-push-only trigger once a second trigger type
   exists.
