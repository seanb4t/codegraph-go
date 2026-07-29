# Cutting a release (maintainer runbook)

This document is the maintainer's step-by-step runbook for cutting a
`codegraph-go` release — pre-tag checks, tag conventions, what the tag push
triggers, the LOCKED `verify.go` contract, post-release verification,
rollback, and the commit-signing fallback for automated pipeline commits.
It complements `docs/RELEASE.md`, which documents how a *user* verifies a
downloaded artifact after the fact; this doc is the *maintainer's* side —
how the release gets cut in the first place.

## 1. Pre-tag gate (mandatory — D-09)

Before tagging **anything** (rc or stable), run the following for **all 6**
release targets. This is the exact check that would have caught the v0.1
`rc.1` failure: a **linux-only** `go.sum` hash (`prometheus/procfs`) that
was invisible on a darwin-only dev machine and to green local CI.

```sh
for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null \
    && echo "OK   $pair" \
    || echo "FAIL $pair"
done
```

All 6 lines must print `OK`. A single `FAIL` means a platform-specific
`go.sum`/build-constraint issue exists that local CI (which only runs on
your dev machine's own `GOOS`/`GOARCH`) cannot see — do not tag until every
target is green.

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
fires the full signed build/assemble/provenance pipeline.

## 3. Branch/tag model

1. Land all phase work on the integration branch
   (`gsd/v1.0-drop-in-parity-human-ux` for this milestone).
2. Run the pre-tag gate (§1) on the integration branch's tip.
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
   runs, which is exactly why this works), fires `release.yml`. Its three
   jobs are unchanged from the pipeline this section has always described:
   - **`build`** — compiles all 6 `(GOOS,GOARCH)` targets (native darwin
     matrix via `macos-latest`/Xcode clang; zig cross-compilation for
     `linux/arm64` and both `windows` targets from `ubuntu-latest`),
     uploading each as a CI artifact.
   - **`assemble`** — downloads all 6 build artifacts, signs each binary
     **individually** with cosign keyless (`cosign sign-blob
     --bundle="${f}.sigstore.json"` — internal/upgrade hashes the
     downloaded binary itself, not a checksums file, so per-binary signing
     is required), generates a per-binary SPDX SBOM via `syft`, computes
     the shared `codegraph_<tag>_checksums.txt`, and publishes into the
     release.
   - **`provenance`** — runs SLSA3 build provenance
     (`slsa-framework/slsa-github-generator`'s generic generator) over the
     checksums file, producing an `.intoto.jsonl` attestation.
7. `assemble`'s publish step finds the Release release-please already
   created and uploads the signed binaries into it with `gh release upload
   --clobber`, leaving release-please's changelog body and prerelease flag
   untouched (see the `assemble` job's `Publish GitHub release` step).

**Forcing a specific version:** the bump comes from accumulated
Conventional Commits, but can be forced for a one-off cut with a one-shot
`Release-As:` footer on an empty commit, e.g.:

```sh
git commit --allow-empty -m "chore: release 1.0.0" -m "Release-As: 1.0.0"
```

Never move this into `release-please-config.json`'s `release-as` field
instead — a config-level setting is sticky and pins every subsequent
release to that version until removed.

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
Release exists yet for it, the `assemble` job's publish step takes its
*create* branch (`gh release create … --generate-notes --prerelease`)
rather than its upload branch. This is the only path in this repo where a
human still runs `git tag` directly — stable releases never use it.

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
# a) cosign keyless signature (per-binary)
cosign verify-blob \
  --bundle codegraph_<tag>_<goos>_<goarch>.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp \
    '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  codegraph_<tag>_<goos>_<goarch>

# b) SLSA provenance
slsa-verifier verify-artifact \
  --provenance-path codegraph_<tag>_checksums.txt.intoto.jsonl \
  --source-uri github.com/seanb4t/codegraph-go \
  --source-tag <tag> \
  codegraph_<tag>_checksums.txt
```

Both commands must succeed for at least one platform's artifacts before
considering the release verified. A signature from any other issuer, any
other workflow file, or any non-tag trigger will fail (a) — as it should.

## 7. Rollback / cleanup

A release-please cut leaves **three** artifacts behind, and rolling back
means undoing all three — not just deleting a tag:

1. the tag,
2. the GitHub Release object release-please created for it, and
3. the version-bump commit on `main` (the `CHANGELOG.md` entry plus the
   `.release-please-manifest.json` bump) together with its merged release
   PR.

Deleting the tag alone is no longer sufficient — release-please's next run
walks `main`'s history and, seeing the manifest already bumped, would
believe the version already shipped.

**1. Delete the Release and its tag:**

```sh
gh release delete v0.0.0-rc.N --cleanup-tag -y

# If the tag was pushed but the release step never ran (e.g. build failed
# before assemble), just delete the tag directly:
git push --delete origin v0.0.0-rc.N
git tag -d v0.0.0-rc.N
```

Nothing is published to the GitHub release until the `build`/`assemble`
gates pass — a failed build produces no Sigstore transparency-log entries
and no public GitHub release, so this step is a no-op (beyond the tag
itself) when the failure happened before `assemble`.

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
