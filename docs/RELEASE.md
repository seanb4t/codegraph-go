# Verifying a codegraph release

This document tells you how to independently verify that a `codegraph`
release binary is authentic and untampered, what this project's dependency
tree actually looks like, and how reproducible the release build is. Every
command below only references artifacts `.github/workflows/release.yml`
actually publishes — if a command here doesn't work against a real release,
that is a bug in this doc (or the workflow), please report it.

> **Status note:** these are the commands a real, tagged release publishes
> artifacts for. As of this writing no `v*` tag has been pushed yet, so no
> live release exists to run these commands against — they are documented
> now so the first real release (and every one after it) is verifiable from
> day one. `codegraph upgrade` runs the equivalent of steps 1 below
> automatically, in-process, before ever swapping the installed binary.

## 1. Verifying a release

Every tagged release (`v[0-9]*`) publishes, per platform
(`darwin`/`linux` × `amd64`/`arm64`):

- a raw binary: `codegraph_<tag>_<goos>_<goarch>` — the asset
  `codegraph upgrade` downloads and self-replaces with; unchanged by this
  section's other additions
- a `.zip` archive of the same binary: `codegraph_<tag>_<goos>_<goarch>.zip`
  — for browser downloads from the GitHub Releases page and for the
  Homebrew tap; not consumed by `codegraph upgrade`
- a per-binary cosign bundle: `<binary>.sigstore.json`
- a per-binary SPDX SBOM: `<binary>.spdx.json`
- one shared checksums file: `codegraph_<tag>_checksums.txt`, covering all
  8 of the above payloads (4 raw binaries + 4 `.zip` archives)
- GitHub build-provenance attestation over those same 8 payloads, published
  through GitHub's Attestations API (`actions/attest-build-provenance`) —
  not a downloadable release asset, unlike everything else in this list

All three verification steps below use only these published assets.

### a) Verify the cosign keyless signature (per-binary)

This is the same check `codegraph upgrade` performs in-process before it
ever swaps the installed binary (`internal/upgrade/verify.go`). Download the
binary and its `.sigstore.json` bundle for your platform, then:

```sh
cosign verify-blob \
  --bundle codegraph_<tag>_<goos>_<goarch>.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp \
    '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  codegraph_<tag>_<goos>_<goarch>
```

This asserts two things, matching exactly what `internal/upgrade/verify.go`
enforces:

- **Issuer:** the certificate was minted by GitHub Actions' Sigstore
  public-good OIDC issuer (`https://token.actions.githubusercontent.com`),
  not some other CI system.
- **Identity (SAN):** the certificate's Subject Alternative Name matches
  `.github/workflows/release.yml` in `seanb4t/codegraph-go`, triggered by a
  `refs/tags/v*` ref — i.e. it was signed by *this specific release
  workflow*, running because of *a version tag push*, not any other
  workflow or trigger in this repository (a pull-request-triggered CI run,
  for example, would never satisfy this pattern).

A signature from any other issuer, any other workflow file, or any
non-tag trigger will fail this check — as it should. The release pipeline
migration that added `.zip` archives and native build-provenance attestation
(below) collapsed the former multi-job pipeline into a single
`goreleaser release` invocation, but did not touch this workflow's filename
or its `refs/tags/v[0-9]*` trigger — the SAN this command checks still
resolves to the same workflow file and tag ref it always has.

### b) Verify build provenance

Releases from **`<first-migrated-release-tag>`** onward (this section will
name the exact tag once plan 01-05 has cut it) are attested by GitHub's
first-party `actions/attest-build-provenance`, published through GitHub's
Attestations API — not as a downloadable `.intoto.jsonl` release asset.
Verify with the `gh` CLI, already installed if you use GitHub at all:

```sh
gh attestation verify codegraph_<tag>_<goos>_<goarch> -R seanb4t/codegraph-go
```

Run this against whichever published asset you intend to trust — a raw
binary or a `.zip` archive; both are covered by the same attestation, over
all 8 published payloads (4 raw binaries + 4 `.zip` archives).

A successful run looks like (exact wording may vary across `gh` versions):

```
Loaded digest sha256:<digest> for file://codegraph_<tag>_<goos>_<goarch>
Loading attestations for sha256:<digest>

The following policy criteria will be enforced:
- Predicate type must match:................ https://slsa.dev/provenance/v1
- Source Repository URI must match:.......... https://github.com/seanb4t/codegraph-go
- Subject Alternative Name must match regex:.. (?i)^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$

✓ Verification succeeded!
```

> **Releases up to and including the last pre-migration tag** published
> `codegraph_<tag>.intoto.jsonl` (`multiple.intoto.jsonl` for v0.2.0 and
> earlier) — a separate downloadable SLSA3 provenance bundle generated by
> `slsa-framework/slsa-github-generator`'s generic generator, verifiable
> with `slsa-verifier verify-artifact`. That command CANNOT verify releases
> from the migrated pipeline: `actions/attest-build-provenance` output is a
> structurally different attestation format, published to a different
> location (GitHub's Attestations API, not a release asset). If you are
> verifying an older release, use the pre-migration instructions this
> section previously documented; if you are verifying a current one, use
> `gh attestation verify` above.

> **Corrected 2026-08-01 (applies to the pre-migration generator only).**
> This section previously instructed verifying `codegraph_<tag>_checksums.txt`
> against a provenance file named `codegraph_<tag>_checksums.txt.intoto.jsonl`,
> and stated that provenance was generated over the checksums file rather than
> over each binary. Both were wrong: no such file was published, and the
> checksums file was not an attested subject. Following the old instructions
> produced `FAILED: artifact hash does not match provenance subject` on a
> release whose provenance was entirely valid — found while verifying
> `v0.2.0`. Attesting the binaries directly was the stronger arrangement;
> only the documentation was wrong. This historical note is retained for
> anyone verifying a pre-migration release; it does not apply to
> `gh attestation verify` above.

### c) Inspect the SBOM

Each binary ships an SPDX-format SBOM generated by `syft`:

```sh
cat codegraph_<tag>_<goos>_<goarch>.spdx.json | jq '.packages[] | {name, versionInfo, licenseConcluded}'
```

This is the authoritative, machine-readable package inventory for that
specific binary — see the dependency-tree narrative below for how to read
it.

## 2. Dependency tree (DIST-05)

**The dependency tree is minimal and audited. CGo, via tree-sitter, is the
sole documented exception to the project's pure-Go, no-CGo default.**

Read literally, `go.mod`'s `require` blocks list 134 total module
requirements. That number alone is misleading without context: 107 of those
134 are **indirect** (transitive) dependencies pulled in by the project's
actual direct choices — none were individually selected or reviewed on
their own merits, they are the closure of what `pebble`, `mcp-go`,
`sigstore-go`, `cobra`, and the tree-sitter bindings themselves require.

Of the **27 direct** requires this project deliberately added, **14 are
tree-sitter grammar modules** — one per supported language
(`tree-sitter-go`, `tree-sitter-java`, `tree-sitter-c-sharp`,
`tree-sitter-python`, `tree-sitter-typescript`, `tree-sitter-rust`,
`tree-sitter-ruby`, `tree-sitter-php`, `tree-sitter-c`, `tree-sitter-cpp`,
`tree-sitter-kotlin`, `tree-sitter-swift`, plus the shared
`tree-sitter/go-tree-sitter` binding and `tree-sitter-javascript`, which also
covers TS/JSX parsing). This is a **wide-but-shallow** tree by deliberate
design, not a small flat one: each grammar module is a self-contained,
independently-versioned parser for exactly one language, pulled in only
because that language is a supported extraction target (see
`docs/LANGUAGE-CAPABILITY-MATRIX.md`). It is not incidental bloat — it *is*
the multi-language parsing story, made explicit rather than hidden behind a
single monolithic "parser" dependency.

The remaining 13 direct requires are the actual supply-chain surface worth
auditing individually: the storage engine (`cockroachdb/pebble/v2`), the MCP
server (`mark3labs/mcp-go`), the CLI framework (`spf13/cobra`), the file
watcher (`fsnotify/fsnotify`), the release-verification client
(`sigstore/sigstore-go`), a JSONC parser for agent-config editing
(`tailscale/hujson`), a pure-Go SQLite reader confined to the one-shot
migration tool (`modernc.org/sqlite`), protobuf codegen support
(`google/protobuf`), a goroutine-leak test harness
(`go.uber.org/goleak`), and a handful of `golang.org/x/*` toolchain
packages.

**CGo is the sole documented exception.** `CGO_ENABLED=1` is required
because `tree-sitter/go-tree-sitter` wraps the tree-sitter C runtime and
every grammar's (mostly-generated) C parser tables — no pure-Go tree-sitter
alternative exists today with comparable language coverage or performance
(see `PARSER-DECISION.md` for the full benchmarked spike that made this
call). No other component in this project introduces CGo.

The **SBOM published with every release binary** (§1c above) is the
authoritative, per-binary, machine-readable inventory of exactly what
shipped in that artifact — use it, not this document, as the source of
truth for any specific release; this section is the narrative context for
reading it.

## 3. Reproducibility posture

Builds are made reproducible via:

- `-trimpath` (strips local filesystem paths from the binary)
- `-ldflags "-buildid="` (clears the non-deterministic Go build ID)
- `SOURCE_DATE_EPOCH` pinned to the release commit's own commit date (not
  build wall-clock time), used to compute the `-X ...Date=` ldflag
  deterministically
- pinned Go and `zig` toolchain versions (`.github/workflows/release.yml`'s
  `GORELEASER_VERSION` env, plus a pinned `zig` version via
  `mlugg/setup-zig`) — `zig cc` is itself a deterministic C compiler
  driver, which is a real asset here compared to a system-installed
  clang/gcc whose exact patch version can silently drift between CI runs

**The reproducibility guarantee is scoped, not blanket, and states this
explicitly rather than hiding cross-target drift behind one green check:**

- **`linux/amd64` is the canonical, blocking target.** `.github/workflows/ci.yml`'s
  `reproducibility` job builds the binary twice, back to back, with
  identical flags and environment, and fails the job if the two resulting
  binaries' SHA-256 digests differ.
- **All other targets (`linux/arm64`, both `darwin` arches) are
  best-effort and reported, not blocking.** CGo cross-linked artifacts
  (via `zig cc`, or via native `macos-latest`/Xcode toolchain for darwin)
  are inherently harder to guarantee bit-identical across toolchain
  versions and host environments than a native `linux/amd64` build. Today
  only the `linux/arm64` leg has an automated double-build check (also in
  `ci.yml`, via `continue-on-error: true`, so a mismatch is visible as a
  warning without failing the run); the `darwin` legs are not yet
  independently double-built in CI. Being explicit about which target is
  the hard guarantee is more honest than a passing check that silently
  doesn't cover the other three.

### Reproducing a build locally

To reproduce the `linux/amd64` build yourself against a specific release
commit:

```sh
git checkout <tag>
COMMIT=$(git rev-parse HEAD)
EPOCH=$(git log -1 --format=%ct)
DATE=$(date -u -d "@${EPOCH}" +%Y-%m-%dT%H:%M:%SZ)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -buildid= -X github.com/seanb4t/codegraph-go/internal/version.Version=${TAG} -X github.com/seanb4t/codegraph-go/internal/version.Commit=${COMMIT} -X github.com/seanb4t/codegraph-go/internal/version.Date=${DATE}" \
  -o codegraph_local ./cmd/codegraph
sha256sum codegraph_local
```

Compare the resulting SHA-256 against the released
`codegraph_<tag>_linux_amd64`'s own digest in `codegraph_<tag>_checksums.txt`
(after verifying that file per §1b above). A match proves the published
`linux/amd64` binary was built exactly from the source at that tag, with no
undisclosed modification.

## `codegraph upgrade` as the automated consumer

`codegraph upgrade` performs the equivalent of §1a above automatically,
in-process (never shelling out to a `cosign` binary), before ever replacing
the currently-installed binary. It fetches `<asset>` and
`<asset>.sigstore.json` for the running platform, verifies the bundle
against the exact issuer + SAN pattern described in §1a, and hashes the
downloaded binary itself (`sha256.Sum256`) to bind the signature to the
actual bytes about to be installed. Verification failure is fatal — `upgrade`
never proceeds to swap the binary on a rejected signature. This document's
manual commands exist so you can independently audit that automated path,
not replace it.
