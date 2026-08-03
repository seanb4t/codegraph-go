---
status: complete
phase: 10-local-build-contribution-and-taskfile-yml-setup
source: [10-VERIFICATION.md]
started: 2026-08-02T22:00:00Z
updated: 2026-08-03T14:24:09Z
---

## Current Test

[testing complete]

## Tests

### 1. release.yml darwin legs — goreleaser build, cosign signing, and SLSA attestation on namespace-profile-macos-6x14-tahoe

expected: Push a real `v[0-9]*` tag and watch `release.yml`'s `build` job end-to-end for the two darwin matrix legs (goos=darwin, goarch=arm64/amd64), specifically the goreleaser invocation, cosign signing, and SLSA attestation steps that run on top of the plain `go build` the canary already proves. The goreleaser step should produce both signed darwin binaries with release ldflags, the `assemble`/`provenance` steps should attest them correctly, and a `codegraph upgrade` smoke test should succeed against the resulting macOS binary on a real Mac.
result: pass
verified_by: mechanical reproduction on a native Apple-Silicon macOS host during /gsd-verify-work 10, plus a new permanent CI guard (see Notes)

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

## Notes

**Premise correction.** The test as written attributes cosign signing and SLSA
attestation to `namespace-profile-macos-6x14-tahoe`. They do not run there. The
darwin matrix legs run exactly one build command — `goreleaser build
--single-target --clean` — then rename and upload. Signing, SBOM, and provenance
all happen afterwards in `assemble` (on `namespace-profile-linux-amd64-2x4`) and
in the `provenance` job. The darwin-specific risk is therefore the goreleaser
build alone.

**Phase 10 changed only runner labels in release.yml.** `git diff
82ffd60^..HEAD -- .github/workflows/release.yml` touches nothing but the
`runs-on` values and their surrounding comments. The goreleaser invocation, the
cosign signing loop, the syft SBOM loop, and the SLSA provenance job are
byte-identical to what shipped v0.2.0.

**What was mechanically verified** (native Apple Silicon, Darwin arm64,
go1.26.5, HEAD tagged with a throwaway local tag so goreleaser resolved a real
tag context):

| Check | Result |
|---|---|
| `goreleaser build --single-target --clean`, darwin/arm64 | exit 0 |
| `goreleaser build --single-target --clean`, darwin/amd64 | exit 0 |
| Object types | `Mach-O 64-bit arm64` / `Mach-O 64-bit x86_64` |
| Release ldflags | Version, Commit, Date all resolved from tag/FullCommit/CommitDate |
| `dist/artifacts.json` lookup used by release.yml's rename step | correct path for both arches |
| syft SBOM over the darwin binary (DIST-03) | exit 0, 148 packages |
| Shape tests (`Darwin\|Provenance\|Release\|Workflow`) | ok |

**Upgrade smoke test was run for real, not simulated.** A throwaway copy of the
freshly built binary ran `codegraph upgrade v0.2.0 --force`, which downloaded the
actual released darwin/arm64 asset from GitHub, verified its cosign keyless
signature and provenance in-process, and atomically replaced itself. The
resulting binary reports `v0.2.0 (commit cce95f3…) darwin/arm64`. This proves the
per-binary cosign signing contract end-to-end for darwin assets on a real Mac.

**Residual, and how it was closed.** GoReleaser had never literally executed on
`namespace-profile-macos-6x14-tahoe` — `release.yml` is tag-push-only, so that
composition would first occur during a release already underway. Closed by
extending the D-08 canary: new `check:darwin-release-build` Taskfile target plus
a second step in `darwin-toolchain-canary.yml`. The canary's `pull_request` path
filter covers `Taskfile.yml` and `darwin-toolchain-canary.yml`, so the PR
carrying that change exercises GoReleaser on that runner class directly. The new
target was proven non-vacuous by mutation (arch assertion inverted → exit 201
with an actionable message; mutation confirmed applied via `git diff` before the
run was trusted; Taskfile restored byte-identical afterwards).

**Discovered while building the guard** (encoded as a comment in the target):
`go tool -modfile=go.tool.mod goreleaser` builds *the tool* for whatever
`GOOS/GOARCH` is in the environment, so the obvious `GOARCH=amd64 {{.GO_TOOL}}
goreleaser build …` produces an x86_64 GoReleaser under Rosetta whose child
`xcrun` cannot load the arm64-only `libxcrun.dylib`, failing as a misleading
`# runtime/cgo` architecture error. `release.yml` is immune only because
`goreleaser-action` installs a native prebuilt binary. The target builds
GoReleaser natively first, then invokes it per-arch.

**Out of scope, captured separately.** cosign signing does not satisfy macOS
Gatekeeper; the darwin binaries are `adhoc, linker-signed` (arm64) and unsigned
(amd64), with `spctl` returning `rejected`. Impact is limited to browser
downloads, since `codegraph upgrade` fetches without setting
`com.apple.quarantine`. Filed as backlog Phase 999.5.
