# Verifying a codegraph release

This document tells you how to independently verify that a `codegraph`
release binary is authentic and untampered, what this project's dependency
tree actually looks like, and how reproducible the release build is. Every
command below only references artifacts `.github/workflows/release.yml`
actually publishes — if a command here doesn't work against a real release,
that is a bug in this doc (or the workflow), please report it.

> **Status note:** tagged releases exist now and these commands are live,
> not aspirational. `v0.5.1` and every release since publish real assets
> and are verifiable today via §1a (cosign), §1b (provenance) and §1c
> (SBOM) — all three run clean against them. §1d (Gatekeeper) now applies
> starting with **`v0.7.0`**, the latest release as of this writing and the
> first to go through real Apple notarization — confirmed with a GREEN
> `spctl -a -vv -t install` verdict on both darwin architectures, plus an
> independent, unproxied confirmation on the maintainer's own Mac via a
> genuine Safari download (`02-EVIDENCE.md` § "SIGN-02 — GREEN Gatekeeper
> verdict on the published release"). `v0.5.1`'s darwin binaries remain
> *deliberately* un-notarized — they are this project's own preserved RED
> baseline for that gate (`02-EVIDENCE.md`) and must not be deleted or
> replaced (`docs/RELEASE-PROCEDURES.md` §7.1). See §1d's applicability
> table for exactly which releases each section covers.
> `codegraph upgrade` runs the equivalent of §1a automatically, in-process,
> before ever swapping the installed binary.

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
- (once notarized — see §1d's applicability table) Apple notarization of
  the two darwin raw binaries above; this adds no new file to the list —
  see §1d for exactly which releases this applies to and how to check it
  yourself

All four verification steps below use only these published assets.

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

Releases from **`v0.5.1`** onward are attested by GitHub's first-party
`actions/attest-build-provenance`, published through GitHub's Attestations
API — not as a downloadable `.intoto.jsonl` release asset.

> `v0.5.1`, not `v0.5.0`, is the first migrated release. `v0.5.0` was tagged
> but published **zero assets**: its pipeline aborted on a Taskfile version
> assertion before `goreleaser release` ran. Per D-07 the tag and release were
> left in place (patch forward, never delete or re-push) and `v0.5.0` is
> marked prerelease so `/releases/latest` skips it. There is nothing to verify
> at `v0.5.0` — it has no assets.

Note the attestation's **subjects are the binaries**, not the checksums file.
`subject-checksums:` feeds `actions/attest-build-provenance` the list of
subjects to attest; the file transporting that list is never itself a subject.
Verifying `codegraph_<tag>_checksums.txt` therefore returns HTTP 404 — that is
correct behavior, not a missing attestation.
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

### d) Verify Gatekeeper trust (notarization)

**Applicability — read this table first.** This project's darwin release
assets have not always meant the same thing with respect to macOS
Gatekeeper, and the guarantee below only applies to some of them:

| Releases | What they published | Which sections apply |
|---|---|---|
| Every release through the last one published before Apple notarization lands (as of this writing: at least `v0.5.1` and `v0.6.0`) | Signed-or-adhoc darwin binaries, cosign bundles, SBOMs, build provenance — **not** Apple-notarized | §a, §b, §c only. §d does not apply: the Gatekeeper install-time assessment below rejects these darwin binaries by design (exit 3) — this is `v0.5.1`'s own recorded SIGN-03 RED baseline, `02-EVIDENCE.md` |
| The first notarized release — **`v0.7.0`**, confirmed GREEN on both darwin architectures (`02-EVIDENCE.md` § "SIGN-02 — GREEN Gatekeeper verdict on the published release") | The same artifacts, plus real Apple notarization of the darwin binaries | §a, §b, §c, §d all apply |
| Every release after the first notarized one | Same as above | §a, §b, §c, §d all apply |

From the first notarized release onward (see the table above), this
project's guarantee is exactly this, quotable verbatim:
**notarized, online-verified, not stapled**.

What each part means, operationally:

- **Notarized** — the darwin binaries carry a real Apple Developer ID
  signature and were accepted by Apple's notarization service.
- **Online-verified** — the acceptance record lives with Apple, not with
  the file. Gatekeeper resolves it over the network the first time the
  binary runs.
- **Not stapled** — nothing is attached to the binary itself. There is no
  local ticket to fall back on if that first-launch network lookup fails.

**Known limitation: offline first launch fails.** A browser-downloaded
binary's *first* launch, on a machine with no network reachability to
Apple, cannot complete the online ticket lookup above and is blocked
exactly as an un-notarized binary would be. This is a deliberate scope
decision, not an oversight — DIST-06 is the deferred requirement that
would close it, and stapling remains out of scope for this
milestone.[^staple-why]

**Known limitation: a browser download loses the execute bit.** Unlike
`gh release download` or `curl`, downloading a raw binary through a
browser (e.g. Safari, from the GitHub Releases page) does not preserve
the Unix executable permission — the file arrives as a plain, non-executable
Mach-O. Trying to run it directly fails with a shell-specific message; for
example, in `fish`:

```
fish: Unknown command. 'codegraph_<tag>_darwin_<arch>' exists but is not an executable file.
```

Restore the bit before running it:

```sh
chmod +x codegraph_<tag>_<goos>_<goarch>
```

(Found during `v0.7.0`'s browser-download verification — `02-EVIDENCE.md`
§ "Post-release verification — manual-dispatch guard check and the
maintainer's own machine".)

[^staple-why]: Apple's `stapler` tool attaches tickets only to
`.app`/`.pkg`/`.dmg` containers, never to a bare Mach-O executable or an
archive, and this project's notarization backend (Quill) has no staple
command at all — see `PROJECT.md`'s macOS Distribution key decisions for
the full rationale.

**Reproducing the check yourself.** These are the same steps
`verify:gatekeeper` in `Taskfile.yml` runs — this is the manual, one-off
version; that target is the automated, repeatable one plan 02-06's CI job
calls. Assumption A2 (`02-EVIDENCE.md`) confirmed that a synthetic
quarantine attribute produces the identical `spctl` verdict as a genuine
browser download on a byte-identical file, so the synthetic form below is
measured, not assumed, to reproduce a real download.

```sh
# 1. Download the raw per-platform binary — never an archive; this is the
# file spctl actually assesses in practice. (The automated target also
# cross-checks GitHub's recorded digest before downloading; omitted here
# for brevity — see Taskfile.yml's verify:gatekeeper for that version.)
gh release download <tag> --repo seanb4t/codegraph-go \
  --pattern 'codegraph_<tag>_darwin_<arch>'

# 2. Apply a synthetic com.apple.quarantine attribute in Apple's documented
# flags;timestamp;agent;event-uuid form. 0081 is "downloaded, not yet
# launched" — the same flag value Safari itself writes, and the same value
# this project's own verify:gatekeeper target uses.
xattr -w com.apple.quarantine "0081;$(printf '%x' "$(date +%s)");Safari;$(uuidgen)" \
  "codegraph_<tag>_darwin_<arch>"

# 3. Read the attribute back and confirm it is actually present. This step
# is not optional: an assessment run against a file that was never
# quarantined produces a misleading pass that has nothing to do with
# notarization, and step 4 below must never run without this confirmation
# succeeding first.
xattr -p com.apple.quarantine "codegraph_<tag>_darwin_<arch>"

# 4. Run the actual Gatekeeper install-time assessment. Read the EXIT
# STATUS, never the text — see "read the exit status" below for why.
spctl -a -vv -t install "codegraph_<tag>_darwin_<arch>"
echo "exit status: $?"
```

**What a pass and a fail look like.** This asserts two independent things,
mirroring §a's shape above: the exit status is the verdict itself, and the
`source=` line is a second, separate confirmation of *why*.

A pass, once a release is notarized:

```
codegraph_<tag>_darwin_<arch>: accepted
source=Notarized Developer ID
exit status: 0
```

A fail, on every release published before notarization (including every
release in the table's first row today):

```
codegraph_<tag>_darwin_<arch>: rejected
exit status: 3
```

**Read the exit status, never a text search.** `spctl`'s full verbose
output can legitimately contain the substring "accepted" inside a
*rejected* verdict's own explanatory text (e.g. "rejected (the code is
valid but does not seem to be an app)"), and inside unrelated `origin=`
lines. A `grep -q accepted` over that output would silently match the
wrong thing. The exit status (`0` = accepted, `3` = rejected) is the only
reliable signal, and it is what both this document's commands and
`verify:gatekeeper` key off.

<details>
<summary>What does <em>not</em> count as verification</summary>

None of the following demonstrate that Gatekeeper trusts a release, even
though each one is easy to mistake for doing so:

1. **A green CI step.** Nothing in this project's CI ever ran `spctl`
   against a real, quarantined, published asset before Phase 2 — a green
   pipeline never asked this question at all.
2. **`codesign -dvv` reporting a valid signature.** This already passes
   today on the adhoc-signed, un-notarized darwin binary that `spctl`
   rejects (see the applicability table above). It answers "is this
   signed," not "does Gatekeeper trust this."
3. **`notarytool history` showing an Accepted submission.** That reports on
   Apple's notary service accepting the *submission*; it says nothing about
   what a locally-quarantined Gatekeeper install-time check does with the
   resulting binary.
4. **An assessment run on a file that was never quarantined.** See step 3
   above — the read-back exists specifically because this produces a
   misleading pass.
5. **`spctl -a -vv -t exec`.** This assessment type rejects any bare
   Mach-O executable on shape alone — a CLI binary is not an `.app` —
   before notarization is even considered. Using it here would report a
   broken release even when notarization is working exactly as intended.
6. **`syspolicy_check distribution` reporting success.** Its `Notary
   Ticket Missing` verdict, Severity Fatal, exit 70, is the *correct,
   permanent, expected* result for every release this project ships,
   because stapling is permanently out of scope (DIST-06). If you run it,
   expect that Fatal line on every release, forever — it does not mean
   anything is broken.

</details>

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
- **The darwin binaries carry a real code signature, confirmed as of
  `v0.7.0` (§1d), and it is NOT bit-for-bit reproducible by anyone —
  including the certificate holder.** This corrects an earlier, unmeasured
  version of this claim. `02-EVIDENCE.md`'s SIGN-04 rehearsal measured that
  the *pre-sign* build is byte-reproducible (`BASELINE-DETERMINISM-OK`,
  identical hashes across repeated builds of the same commit), but the
  *final*, signed-and-notarized binary is **not**: two separate signing
  operations of the identical pre-sign bytes produced different final
  hashes, because Apple's notarization service embeds a trusted timestamp
  inside the code signature that varies per signing operation
  (`02-EVIDENCE.md` § "The non-reproducible-signature finding"). Holding
  this project's actual Developer ID Application certificate lets you
  produce a *validly signed* binary — it does not let you reproduce the
  exact published bytes. Anyone verifying a darwin release should compare
  the **pre-sign** build against a source-only rebuild, never the final
  signed artifact, or every single comparison will report a false
  regression. This does not change or weaken the `linux/amd64` canonical
  guarantee above, which carries no signature at all and remains
  reproducible end-to-end, bit-for-bit, by anyone.

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

## 4. Installing via Homebrew (macOS)

The canonical install command lives in a single place — [`README.md`](../README.md)'s
"Homebrew (macOS)" section — not repeated here or in the tap repository's own
README. This section states the shipped guarantee precisely, in the same
voice §1d states notarization's guarantee.

**What the cask installs.** The `codegraph` binary, shell completions for
bash, zsh, and fish, and man pages — but not the way a typical Homebrew
formula ships completions and man pages (as files bundled inside the
download). Completions and man pages are **generated from the installed
binary at install time**: `generate_completions_from_executable` runs
`codegraph completion <shell>` against the just-installed binary, and a
post-install hook runs `codegraph man` against it. Neither is a static file
this project authored — both are the exact output of the exact binary the
cask installed, so a new subcommand or flag shows up without anyone editing
a committed completion file.

**What the install does on your behalf.** The post-install hook that
generates man pages does more than generate them — it is the cask's install
gate (BREW-05). It runs the installed binary and refuses to complete the
install (rolling back everything already staged) if either: the binary
cannot produce more than one man page, or the binary reports a version that
does not match the cask's own declared version. This is why a corrupted or
mismatched download fails loudly at `brew install` time, for you, rather
than silently succeeding and failing quietly the first time you actually run
`codegraph`. See `03-EVIDENCE.md` (this milestone's Phase 3 plan 4) for both
failure modes demonstrated against real, deliberately broken artifacts —
this is not an argued property, it has been watched fail both ways.

**The man-path dependency — measured, not assumed.** Man pages install
under the Homebrew prefix's own man directory
(`$(brew --prefix)/share/man/man1`). Whether `man codegraph` resolves
depends on whether that directory is on your shell's search path, and that
in turn depends on whether Homebrew's own post-install shell setup has ever
been sourced in your profile — Homebrew's own standard instruction after
first install (`eval "$(brew shellenv)"` in `~/.zprofile`/`~/.bash_profile`,
or via a login shell that runs `/usr/libexec/path_helper`). Measured
directly on a real Apple Silicon Mac (Darwin 27.0.0): `/etc/manpaths.d/` is
empty and `path_helper -s`'s own `MANPATH` output does not include
`/opt/homebrew/share/man` — this prefix is genuinely **absent from the
system-level man path configuration**, not merely "sometimes missing." In a
shell that never sourced Homebrew's environment, `man codegraph` returns
`No manual entry for codegraph` even with the page correctly installed
on disk. If that happens to you, this invocation bypasses man-path
resolution entirely and reads the page directly — measured working on this
same machine, in an environment with no Homebrew shell setup sourced at
all:

```sh
man "$(brew --prefix)/share/man/man1/codegraph.1"
```

**The bash-completion dependency — measured, not assumed.** zsh and fish
each pick up their generated completion automatically (zsh via `compinit`,
fish via its own vendor-completions auto-load) once Homebrew's own shell
integration has been sourced. Bash is different: the Cobra-generated
completion script's own fallback path still calls a helper function
(`_get_comp_words_by_ref`) that only exists once the `bash-completion`
formula (v1 or v2) is installed and sourced — without it, `codegraph <TAB>`
silently falls back to ordinary filename completion instead of offering
subcommands, with no error. If bash completion isn't offering subcommands,
`brew install bash-completion` (or `bash-completion@2`) and make sure your
`~/.bash_profile`/`~/.bashrc` sources `$(brew --prefix)/etc/profile.d/bash_completion.sh`
— the same shell-setup step the man-path caveat above depends on.

**If `brew install codegraph` refuses with "untrusted tap."** The first
install from any newly-tapped, non-official tap on recent Homebrew versions
requires an explicit trust step — this is a general Homebrew mechanism, not
specific to this cask:

```
Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap seanb4t/tap.
Run `brew trust --cask seanb4t/tap/codegraph` or `brew trust seanb4t/tap` to trust it.
```

Run the command the error names, then re-run `brew install codegraph`:

```sh
brew trust --tap seanb4t/tap
brew install codegraph
```

**Upgrading.** A brew-managed install is upgraded with `brew upgrade
codegraph`, not `codegraph upgrade`. `codegraph upgrade` detects a
brew-managed install by resolving the running binary through symlinks and
requiring Homebrew's own `INSTALL_RECEIPT.json` at the matching
Caskroom/Cellar ancestor — correct under any install prefix, and it never
shells out to `brew`. Under that detection, bare `codegraph upgrade`
refuses with a pointer naming `brew upgrade codegraph` and exits non-zero,
because it was asked for a mutation it declines to perform; `codegraph
upgrade --check` reports the same pointer and exits zero, because it only
answered a question it could answer. There is no override flag and no
environment escape hatch: `brew uninstall --cask codegraph` is the way to
leave Homebrew's management, and that is deliberate — a forced self-swap
would leave `brew list --cask --versions` reporting a version that is no
longer on disk.
>
> **Amended 2026-08-11 (phase 4, `UPGR-01`/`UPGR-02`/`UPGR-03`).** This
> paragraph previously stated that detection and refusal did not exist and
> that the interaction with Homebrew's own Caskroom bookkeeping was
> undefined. All three claims are now closed: the mechanism above is what
> ships as of this amendment.

> **Verified 2026-08-10 (plan 03-05), against the published `v0.8.0`
> release.** The three claims this section previously left pending are now
> each closed, with a citation to real, executed evidence rather than a
> rehearsal:
>
> 1. **The tap resolves** — `brew tap seanb4t/tap` was run cold against the
>    real, published `seanb4t/homebrew-tap` repository. See `03-EVIDENCE.md`
>    "BREW-01 — the cold install."
> 2. **A cold install succeeds** — `brew install codegraph` completed on a
>    machine torn down of any prior `codegraph` cask, tap, binary, sentinel,
>    or man pages, against the cask GoReleaser rendered and pushed for the
>    real, tagged `v0.8.0` release. The starting state was a torn-down
>    machine, not a genuinely never-had-codegraph one — recorded plainly as
>    the weaker of the two, in `03-EVIDENCE.md`.
> 3. **Completion works in all three shells** — bash, zsh, and fish each
>    offered real subcommand names with descriptions, driven through a
>    genuine interactive shell (`tmux send-keys`/`capture-pane`), against the
>    real brew-installed binary. `03-EVIDENCE.md` "BREW-01 — the cold
>    install" records all three as separate verdicts.
>
> **One documentation gap this verification found, closed above and
> repeated here for visibility.** The literal published two-command line
> does not succeed unmodified on Homebrew 6.0.16: the first `brew install`
> of any cask from a newly-tapped, non-official tap is refused
> (`Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap
> seanb4t/tap.`) until the tap is explicitly trusted
> (`brew trust --tap seanb4t/tap` or `brew trust --cask
> seanb4t/tap/codegraph`) — a real, general Homebrew mechanism, not a defect
> in this cask or tap, but one this document did not previously mention. If
> `brew install codegraph` refuses with that message, run the `brew trust`
> command it names, then re-run `brew install codegraph`.
>
> **One release only was cut for this verification, not two.** GoReleaser's
> tap-push *update* path (writing a second commit to an already-existing
> `Casks/codegraph.rb`) and `brew upgrade codegraph` consuming a
> regenerated cask both remain unexercised — an accepted gap, not a silently
> dropped claim, closed on the next natural release. See `03-EVIDENCE.md`
> "Scope reduction, recorded plainly" and ROADMAP criterion 1's 2026-08-10
> amendment.

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

One distinction worth being explicit about, since the two are easy to
conflate: the signature `codegraph upgrade` checks above is a *detached*
Sigstore/cosign bundle — this project's own signing mechanism, checked
in-process before ever swapping the installed binary. That is a completely
different mechanism from the *embedded* Apple Developer ID signature
notarization adds directly to the darwin Mach-O binary (§1d), and it does
nothing for Gatekeeper: `codegraph upgrade` successfully verifying a cosign
bundle has no bearing on whether a browser-downloaded copy of the same
binary passes `spctl`. The two checks protect against different threats and
neither substitutes for the other.
