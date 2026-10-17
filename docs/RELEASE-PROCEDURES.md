# Cutting a release (maintainer runbook)

This document is the maintainer's step-by-step runbook for cutting a
`codegraph-go` release — pre-tag checks, tag conventions, what the tag push
triggers, the LOCKED `verify.go` contract, post-release verification,
rollback, and the commit-signing fallback for automated pipeline commits.
It complements `docs/RELEASE.md`, which documents how a *user* verifies a
downloaded artifact after the fact; this doc is the *maintainer's* side —
how the release gets cut in the first place.

## 1. Pre-tag gate (mandatory — D-09)

Before tagging **anything**, run the following for **all 4** release
targets. This is the exact check that would have caught the v0.1
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
> `pretag-gate` job in both `.github/workflows/release-pr.yml` (on every push
> to `main`, gating the release-PR job behind it via `needs:`) and
> `.github/workflows/release-tag.yml` (on a push to `main` that changes
> `CHANGELOG.md`, gating the tag job behind it via `needs:`). The job invokes
> it as `task check:cross` (D-15) — the sweep's one definition lives in
> `Taskfile.yml`, not inlined in either workflow. Run `task check:cross`
> locally for the same check the manual command block above performs, before
> you ever push to `main` — but the enforcement point is now CI, not this
> document. No human needs to remember to run this by hand anymore.

## 2. Tag conventions

`release.yml` triggers on pushes matching `v[0-9]*`. Two distinct shapes:

| Tag shape | Example | Meaning | Triggers `release.yml`? |
|---|---|---|---|
| Stable | `v1.0.0` | Full release, no `-` suffix — becomes `codegraph upgrade`'s resolved "latest" | Yes |
| Internal milestone marker | `milestone-v1.0` | Documentation/planning marker only | **No** — deliberately does not match `v[0-9]*` |

There is no hand-pushed prerelease tag shape (D-15). Every `v[0-9]*` tag
is pushed by `release-tag.yml` when a release PR merges, and
`release:goreleaser` refuses any tag that has no `.changes/<tag>.md`. See
§4, "No hand-pushed prerelease tag (D-15)".

Never use a `milestone-v*`-style name for anything you actually want
`release.yml` to build and sign — it is intentionally shaped to never fire.
Conversely, never push a real `v[0-9]*` tag casually — every one of them
fires the full signed build/archive/checksum/sign/SBOM/publish/attest
pipeline (one job, described in full in §4 below).

## 3. Branch/tag model

1. Land all phase work on the integration branch
   (`gsd/v1.0-drop-in-parity-human-ux` for this milestone).
2. The pre-tag gate (§1) also runs automatically on every push to `main`, as
   the `pretag-gate` job in `release-pr.yml` and in `release-tag.yml` (see the
   note in §1 above) — a release PR can never be opened, and a tag never
   pushed, from a `main` state that fails it.
3. Merge the integration branch onto `main`. A squash merge is fine: the
   changelog is built from the `.changes/unreleased/` fragments the work
   carries, not from commit history, so nothing needs the individual commits
   preserved.
4. The tag is pushed by `release-tag.yml` when the release PR merges (§4) —
   never by hand.

What still holds from the earlier model: the milestone's work lands on `main`
before the tag exists, and the tag is cut at a commit on `main`, never on a
feature or integration branch.

## 4. Cutting a release (the changie release PR)

A release is cut by merging the release PR. There is no hand-run `git tag`.

The chain, end to end:

`.changes/unreleased/` fragment → release PR (release-pr.yml) → merge → App-token tag (release-tag.yml) → release.yml (unchanged)

1. A change lands on `main` carrying its fragment in `.changes/unreleased/`
   (see CONTRIBUTING.md, "Pull requests"). Merging the integration branch to
   `main` (§3) lands its fragments the same way.
2. `.github/workflows/release-pr.yml` runs on every push to `main`. Its
   `pretag-gate` job (§1's automated sweep) must pass before the release-PR
   job runs. That job runs `task release:batch`: `changie batch auto`,
   `changie merge`, `changie latest`. This renders `.changes/vX.Y.Z.md` from
   the fragments, merges it into `CHANGELOG.md`, and deletes the fragments.
   The workflow then opens, or updates, one pull request on the fixed branch
   `changie/release`, titled `chore(main): release vX.Y.Z`, whose body is
   `.changes/vX.Y.Z.md`. The PR is opened ready for review, never as a draft,
   and with the release GitHub App's token (§9), so every required check runs
   on it.
   `release:batch` fails when the rendered notes contain a GitHub skip-CI
   phrase: the PR body becomes the squash commit message on `main` (a
   repository setting), and such a phrase would suppress `release-tag.yml`.
3. Review the PR's version and notes. If the version looks wrong, change the
   kinds of the fragments on `main`; the next push regenerates the PR.
4. Merge the PR. That push to `main` changes `CHANGELOG.md`.
5. `.github/workflows/release-tag.yml` triggers on a push to `main` that
   changes `CHANGELOG.md`, so only a release-PR merge fires it. Its
   `pretag-gate` job runs first. Then `task release:tag` tags `changie latest`
   at the pushed commit and pushes that tag with the release GitHub App's
   token. A tag that already exists on origin is a successful no-op, so a
   re-run is safe. The task runs only under GitHub Actions.
6. The tag push, authored by the App token (not the default `GITHUB_TOKEN` —
   App-authored refs *do* trigger downstream workflow runs, which is exactly
   why this works), fires `release.yml`. It has exactly ONE job, `release`,
   running natively on a real Apple-Silicon macOS profile
   (`namespace-profile-macos-6x14-tahoe`) — no separate
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
   - **Publish** — GoReleaser's declarative `release:` pipe creates the
     GitHub Release for the tag and uploads into it, idempotently
     (`replace_existing_artifacts: true`). The Release body is
     `.changes/vX.Y.Z.md`, which `task release:goreleaser` hands to
     GoReleaser with `--release-notes`; it refuses a tag whose
     `.changes/<tag>.md` is missing or empty (REL-12). The prerelease flag
     follows the tag (`prerelease: auto`).
   - **Attest** — `actions/attest-build-provenance` (GitHub's first-party
     attestor) runs as the job's last step, over the SAME 8 payloads the
     checksums file covers, publishing through GitHub's Attestations API
     rather than as a release asset (D-09 — replaces the third-party SLSA
     generic generator entirely; see `docs/RELEASE.md` §1(b) for the
     `gh attestation verify` command that replaces `slsa-verifier
     verify-artifact` for releases cut by this pipeline).

**The version is derived from the fragment kinds, not chosen.** changie
computes it from the kinds of the unreleased fragments: Breaking and Features
bump the minor while the project is on a `0.x` baseline (`.changie.yaml` pins
Breaking to a minor bump until 1.0), and Fixes, Performance and Dependencies
bump the patch. `task changie -- next auto` previews the next version
locally.

> **Project directive (2026-07-29):** do **not** manufacture a version. No
> hand-edited version file and no hand-pushed tag authored for the purpose of
> producing a bump. If you believe a release needs a version the fragments
> would not compute, that is a conversation to have before the cut, not an
> edit to make during it. See `09-CONTEXT.md` D-06R.

**Expected silent no-op:** if there are no fragments in `.changes/unreleased/`,
`task release:batch` prints a notice and exits 0, and no PR is opened or
updated. This is expected, not a broken pipeline — check the `release-pr`
workflow run's log rather than assuming a failure and reaching for a manual
tag.

**No hand-pushed prerelease tag (D-15).** The earlier `rc` escape hatch is
removed. A human `git tag` contradicts the no-human-tag rule (`09-CONTEXT.md`
D-06R), and `release:goreleaser` refuses any tag that has no
`.changes/<tag>.md` (REL-12). changie can batch a prerelease with
`changie batch auto -p rc.N`, but that is not wired into `release:batch`; no
prerelease route is built.

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

> **Mandatory, not optional, for the first release cut by the changie
> chain.** Running these two commands is not optional for that release and
> for the live proof (§9) of the chain — it is the only evidence that the
> cosign identity survived the App-token rewiring, and it is the direct
> evidence REL-02's second half is verified against. Do not skip it because
> the pipeline "should" still be signing correctly; prove it.

## 7. Rollback / cleanup

**Recovery posture is PATCH FORWARD (D-07) — this section governs cleanup of
a disposable cut only (for example §9's live proof), never a published
stable (`vX.Y.Z`) release a user may have resolved as latest.** Do **not**
delete or re-push a published tag or Release to "fix" a bad cut. Deleting
and re-pushing a tag re-fires `release.yml` — a human touching tag
authority the release workflow owns (D-06R) — and, more concretely, would
destroy any un-notarized or otherwise deliberately-preserved asset a later
phase depends on as its baseline (the exact mistake this note exists to
prevent). A wrong stable release is fixed by the **next** patch release,
cut through the release PR, not by rewriting history. This section's
delete-and-retry procedure below applies **only** to a disposable cut (§9's
live proof) — never to a real `vX.Y.Z`.

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

A release cut by the changie chain leaves **three** artifacts behind:

1. the tag,
2. the GitHub Release object GoReleaser created for it, and
3. the release commit on `main` (the `CHANGELOG.md` entry plus
   `.changes/vX.Y.Z.md`, with the fragments it consumed deleted) together
   with its merged release PR.

Deleting the tag alone is not sufficient: `main` would still carry the
version's changelog entry, and the fragments that produced it would stay
gone.

**1. Delete the Release and its tag:**

```sh
gh release delete vX.Y.Z --cleanup-tag -y
```

If the tag was pushed but no Release was ever created (e.g. the build
failed before GoReleaser's `release:` pipe ran), `gh release delete` has
nothing to delete — in that case the tag itself is the only artifact left,
and it is the tag `release-tag.yml` pushed, so deleting just the tag is the
correct and only step for this part; skip straight to confirming with
`git tag -l 'vX.Y.Z'` and `git ls-remote --tags origin 'vX.Y.Z'` that
nothing remains, rather than assuming a Release exists to delete.

Nothing is published to the GitHub release until every step in the single
`release` job succeeds — a failed build produces no Sigstore
transparency-log entries and no public GitHub release, so deletion is a
no-op beyond the tag itself when the failure happened before GoReleaser's
`release:` pipe ran.

**2. Revert the release commit on `main`** through a normal pull request
(title it `revert: ...`, which the title check accepts). Reverting the merged
`chore(main): release vX.Y.Z` squash commit restores the fragments and
removes `.changes/vX.Y.Z.md` and its `CHANGELOG.md` entry. `release-tag.yml`
fires on that push too, because it changes `CHANGELOG.md`, but `changie
latest` is then the previous version, whose tag already exists, so it does
nothing. `release-pr.yml` then reopens a release PR from the restored
fragments. Never re-tag by hand.

Never reuse a tag name after deleting it for a **stable** (`vX.Y.Z`) release
— once `codegraph upgrade` has resolved a version as "latest," retagging
the same name risks a confusing mismatch between what a user already
verified and what the tag now points to.

This same two-step procedure (delete release+tag, revert the release
commit) is what tears down the disposable live-proof release described in
§9.

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
`release.yml` would simply never fire on the tag `release-tag.yml` pushes.
For the same reason, a pull request opened or updated with the default token
fires no `pull_request` workflows, so `protect-main`'s required checks would
never report on the release PR and it could never merge. A GitHub App
installation token does trigger downstream workflow runs. This is not
optional plumbing; it is the single mechanism that makes §4's automated
flow work at all.

This is a one-time, maintainer-manual setup procedure. Follow it once per
repo (or once per organization if the App is installed org-wide):

1. **Create the App** under the account's Developer settings
   (`Settings → Developer settings → GitHub Apps → New GitHub App`). Any
   name is fine; it needs no webhook, no user-facing UI, and no public
   listing.
2. **Grant it the installation permissions the chain uses:**
   - **Contents: Read and write** — needed to push the `changie/release`
     branch and the release tag.
   - **Pull requests: Read and write** — needed to open/update the
     `chore(main): release vX.Y.Z` PR.

   The previous release automation also held **Issues: Read and write**,
   because it labelled its PR and GitHub governs PR labels under the Issues
   API scope. The changie chain applies no labels and does not use that
   scope.
3. **Install the App** on `seanb4t/codegraph-go`. A single-purpose,
   per-repo App is the ideal — but reusing an existing release-automation
   App already installed on other repositories is acceptable practice as
   long as its installation permissions are no broader than step 2 lists
   (an Issues scope held over from before may remain until a maintainer
   drops it).
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
settings) authorize everything the chain does with the minted App token.
The `permissions:` blocks inside `.github/workflows/release-pr.yml` and
`.github/workflows/release-tag.yml` govern only the default `GITHUB_TOKEN`
and are a separate, unrelated scope. Configuring one correctly while
under-scoping the other does not fail at config time — it produces a runtime
authorization failure (a 403) the first time the release-PR job pushes its
branch or opens its PR, or the tag job pushes the tag, not before.

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
pull requests, and is a common blocker for workflows that open pull
requests with that token. But the release PR is authored with the **App's**
installation token (D-03), a distinct actor from `GITHUB_TOKEN`, so this
setting is not load-bearing for this pipeline. Confirm its value here for
the record (and prefer it enabled, as the safer default for this repo
generally), but a disabled value does not by itself block the release PR —
only an under-scoped App installation (step 2 above) does that.

**Branch protection note:** `main` is protected by the `protect-main`
ruleset (squash-only, linear history, required status checks). The chain
never pushes to `main`: the App pushes only the `changie/release` branch and
the release tag, and a maintainer merges the release PR after its required
checks pass. If the ruleset is ever extended to restrict creating branches or
tags, the App **must** be added to the bypass-actor list, or the release PR
branch and the tag push will be rejected.

**The live proof.** Because "does an App-token-authored tag push really fire
`release.yml` in this repo's actual Actions configuration" cannot be proven
statically, the chain's first real run is its live proof, and it is checked
before the chain is relied upon. §6's post-release verification and §7's
rollback procedure both apply to that release exactly as they would to any
other; §7 is what tears it down afterward if it must be withdrawn.

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

## 11. Recovering from a post-publish hash-recording failure (SIGN-04)

`release.yml`'s `Record final local hashes (SIGN-04, post-everything)` step
(`task release:record-final-hashes`, plan 02-07) runs AFTER the release has
already published — archive, checksum, sign, and publish have all already
happened. It exists to make the first byte-identity comparison point
(`final_local_sha256`) observable, and it hard-fails (deliberately, not
silently) if it cannot read that value from `dist/artifacts.json`.

**Because it runs post-publish, its failure is structurally different from
every other step in this job.** The release ASSETS ARE ALREADY LIVE when
this step can fail — nothing upstream of it was skipped. And because
`post-release-verify.yml` triggers on `workflow_run` gated to a `success`
conclusion (§ above), a RED `release` job conclusion means post-release
verification **never fires automatically** for that tag: the Gatekeeper job
and the notarized-suite job both silently do not run. A red release with
live assets and no automatic verification is exactly the state this runbook
must not leave unattended.

**First, always: preserve the logs.** Download both `release.yml`'s run
logs and (if it ran at all) `post-release-verify.yml`'s run logs to local
files before doing anything else. GitHub expires run logs; a failure branch
that starts writing conclusions before capturing the raw material leaves any
later post-mortem with nothing to work from.

**Then distinguish which of four states the release is actually in** — they
have different remedies, and treating one as another either under- or
over-reacts:

| State | What happened | How to tell | Remedy |
|---|---|---|---|
| (a) Hashes never recorded | The recording step failed (absent `dist/artifacts.json`, missing darwin binary, etc.) but assets are published | The `release` job's run log shows the `Record final local hashes` step red; the tag's GitHub Release has all expected assets | Manually `workflow_dispatch` `post-release-verify.yml` against the resolved tag (`gh workflow run post-release-verify.yml -f tag=<tag>`) to gather what verification IS still possible; SIGN-04's `final_local_sha256` provenance point for this tag is simply absent and must be recorded as such, not fabricated |
| (b) Attestation never ran at all | The recording step's failure means the job never reached the `Attest build provenance` step, which follows it in the same job — the build-provenance claim is ABSENT from this release, not merely unverified | `gh attestation verify` against any asset of this tag returns no attestations found (not a verification failure — a genuine absence) | Same as (a): dispatch post-release-verify manually to confirm the absence directly rather than assume it; record the absence explicitly in the phase evidence |
| (c) Manual dispatch surfaces both | A manual `workflow_dispatch` of `post-release-verify.yml` against this tag is expected to run every job — none skip — and the attestation-verification leg is expected to legitimately FAIL, since (b) means there is nothing for it to verify | The dispatched run's `verify-supply-chain` job (or equivalent) reports attestation verification failed, while other jobs (Gatekeeper, notarized-suite) may still pass if notarization and cosign both completed before the failure | This is the expected shape of (a)+(b) combined, not a new problem — do not re-run the dispatch hoping for a different result; move to (d) |
| (d) A public supply-chain claim is genuinely absent | Either the byte-identity chain (SIGN-04) or the build-provenance attestation (or both) cannot be produced for this tag, confirmed by (a)-(c) above | Confirmed by the states above, not assumed | **Patch-forward, never re-cut (D-12/D-07).** Do not delete or re-push this release or tag. Cut a new patch release through the normal release PR flow (§4); that new tag's release job will produce a complete, provable chain. Record the incomplete tag's gap explicitly in the phase's evidence artifacts (never silently omit it or backfill a value nothing produced) |

The unifying rule across all four states, restated from §7 above: **a
red release here is patched forward with a new version, never rewritten by
deleting or re-pushing the affected tag or release.**

## 12. The Homebrew tap (maintainer side)

**Where the tap lives.** `seanb4t/homebrew-tap` — public, default branch
`main`, seeded with only `LICENSE` and `README.md` (plan 03-03). GoReleaser's
`homebrew_casks:` pipe owns `Casks/codegraph.rb` in that repository
end to end: it is rendered locally as part of every `goreleaser release`
run and pushed by the SAME pipe's `Publish()` step. **Never hand-edit
`Casks/codegraph.rb` in the tap repository.** Any manual edit is
overwritten, without warning, by the next release — GoReleaser's cask
render is the sole source of truth for that file's contents, matching
`.goreleaser.yaml`'s own comment on the block ("This file was generated by
GoReleaser. DO NOT EDIT.", literally the first line of every rendered
cask).

**Which App publishes it.** A tap-scoped GitHub App — "seanb4t homebrew tap
publishing" (App id `4549710`) — installed on `seanb4t/homebrew-tap` alone,
with exactly `contents: write` + the mandatory `metadata: read` (plan
03-03). This is **deliberately a second, separate App from the
release GitHub App** documented in §9 above (`APP_ID`/`APP_PRIVATE_KEY`),
not a reuse of it: the release GitHub App is installed on
`seanb4t/codegraph-go` and authors the release-PR branch, release PR and
tag actions that repository needs; the tap App is installed on `seanb4t/homebrew-tap`
alone and can write nothing else, including `codegraph-go` itself — that
scoping is the entire content of `BREW-02`/ROADMAP criterion 5, and it only
holds if the two Apps stay separate. Its installation token is minted
**inside** the `release` job in `release.yml`, immediately before the
`Release` step — not in a separate job passing the token as a job output —
because a job output masked by GitHub Actions' secret-masking is dropped
entirely rather than passed through to a consuming job, measured directly
on a real, deletable, never-merged scratch-branch run (`03-03-SUMMARY.md`,
run `31417685002`, verdict `REDACTED`). Two repository secrets carry its
credential: `HOMEBREW_TAP_APP_ID`, `HOMEBREW_TAP_APP_PRIVATE_KEY`, both on
`seanb4t/codegraph-go` (never on the tap repository itself — a tap-scoped
App has no reason to hold its own credential inside the repo it publishes
to).

**Rotating the App's private key.** Follow the same shape as §9's
release GitHub App, applied to the tap App instead:

1. In the tap App's own settings (`Settings → Developer settings →
   GitHub Apps → seanb4t homebrew tap publishing → Private keys`), generate
   a new private key. This does not invalidate the old key immediately —
   both are valid until the old one is explicitly deleted.
2. Reseed the repository secret with the new key's full PEM:
   ```sh
   gh secret set HOMEBREW_TAP_APP_PRIVATE_KEY --repo seanb4t/codegraph-go < new-key.pem
   ```
   `HOMEBREW_TAP_APP_ID` does not change during a key rotation — the App's
   numeric id is stable across key generations; only reseed it if the App
   itself was ever recreated rather than merely re-keyed.
3. Confirm the new key mints tokens correctly before deleting the old one
   — mint a JWT with it and call `GET /app` (same identity-check pattern
   §9's troubleshooting note describes for the release GitHub App): a
   `200` response whose `id`/`slug` match the tap App confirms the new key
   is live and correctly scoped.
4. Delete the old key from the App's settings page once the new one is
   confirmed working. An old, un-deleted key remains a valid credential
   for this App indefinitely — treat key rotation as incomplete until this
   step runs.
5. Immediately shred any local copy of either key (`rm -P`, never left in
   the repository, never committed, never printed to a log or transcript)
   — the same handling `03-03-SUMMARY.md` and `03-EVIDENCE.md` both
   record for this exact key.

**When a tap push fails.** State the recovery in terms of what is already
true, not by inventing a new procedure: GoReleaser's `cask.Pipe{}` is
registered in the PUBLISH pipeline, ordered deliberately last — after
archive, checksum, sign, and SBOM have all already completed and the
release's own assets are already live (`.goreleaser.yaml`'s own comment:
"brew et al use the release URL, so, they should be last"). A failed tap
push therefore happens **after** the release itself — the GitHub Release,
its raw binaries, `.zip` archives, checksums file, cosign bundles, SBOMs,
and build-provenance attestation — is already complete and independently
re-verifiable exactly as `docs/RELEASE.md` §1 describes. Nothing about the
release itself needs to be touched, patched forward, or re-cut on account
of a failed cask push alone.

The remedy is therefore narrow: diagnose and fix whatever the tap push
failure names (an expired or misconfigured App credential, a network
failure, the tap repository being unreachable), then re-run
`goreleaser release` for the SAME tag — GoReleaser's cask publish step is
idempotent (it pushes a commit updating `Casks/codegraph.rb` to the
current render; a re-run against an unchanged render is a no-op commit,
and against a changed one is a normal update) — never delete or re-push
the release or its tag to retry a tap-only failure (§7's rule, restated
here because a tap failure is exactly the shape that tempts a "just re-cut
it" response).

**This claim rests on an argument, not an observed failure — stated
plainly rather than implied.** No real tap push has ever failed in this
milestone; this section describes what the pipeline's own ordering (cask
publish strictly after every other release artifact) implies would happen,
not something watched happen. See `03-EVIDENCE.md` "BREW-06, half one" for
the full pipeline-ordering argument this recovery procedure depends on
(citing `internal/pipe/publish/publish.go` and `internal/pipeline/pipeline.go`
by file and line, labelled throughout as an argument with no executed
evidence, per D-18R), and `03-EVIDENCE.md` "BREW-06, half two" for the
release-integrity half, which IS executed evidence — re-verified against the
real `v0.8.0` release cut by plan 03-05. ROADMAP Phase 3 criterion 4's
2026-08-10 amendment records both halves' distinct evidentiary status in the
same place a reader reaches first.
