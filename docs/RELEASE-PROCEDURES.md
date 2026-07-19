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
3. Squash-merge the integration branch to `main`.
4. Tag `main` (never a feature/integration branch) with the stable tag.

The squash-merge is where the milestone's work lands as a single reviewable
commit on `main`; the tag is a pure ref on top of that commit — it carries
no code changes of its own.

## 4. Cutting the tag (maintainer-manual action)

```sh
git checkout main
git pull
git tag v1.0.0
git push origin v1.0.0
```

This is a **manual, maintainer-only** action — no automation in this repo
ever creates or pushes a `v[0-9]*` tag on your behalf. Pushing it triggers
`release.yml`'s three jobs:

- **`build`** — compiles all 6 `(GOOS,GOARCH)` targets (native darwin
  matrix via `macos-latest`/Xcode clang; zig cross-compilation for
  `linux/arm64` and both `windows` targets from `ubuntu-latest`), uploading
  each as a CI artifact.
- **`assemble`** — downloads all 6 build artifacts, signs each binary
  **individually** with cosign keyless (`cosign sign-blob
  --bundle="${f}.sigstore.json"` — internal/upgrade hashes the downloaded
  binary itself, not a checksums file, so per-binary signing is required),
  generates a per-binary SPDX SBOM via `syft`, computes the shared
  `codegraph_<tag>_checksums.txt`, and publishes the GitHub release.
- **`provenance`** — runs SLSA3 build provenance
  (`slsa-framework/slsa-github-generator`'s generic generator) over the
  checksums file, producing an `.intoto.jsonl` attestation.

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

## 7. Rollback / cleanup of a failed rc tag

Nothing is published to the GitHub release until the `build`/`assemble`
gates pass — a failed build produces no Sigstore transparency-log entries
and no public GitHub release. If a prerelease tag (`v0.0.0-rc.N`) was
pushed and the workflow failed, or the resulting prerelease needs to be
retracted:

```sh
# Delete the GitHub release (if one was published) and its tag
gh release delete v0.0.0-rc.N --cleanup-tag -y

# If the tag was pushed but the release step never ran (e.g. build failed
# before assemble), just delete the tag directly:
git push --delete origin v0.0.0-rc.N
git tag -d v0.0.0-rc.N
```

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
