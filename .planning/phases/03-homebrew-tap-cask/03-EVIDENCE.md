# Phase 3 Plan 4 — Recorded Evidence

Single recorded-observation file for plan `03-04`, following the sectioning
and fixed-key-order conventions `02-EVIDENCE.md` established: a command, its
verbatim output, and an explicit verdict — never a narrative conclusion in
place of the transcript.

All observations on this page were captured on **2026-08-10** on the
maintainer's real Mac (Apple Silicon, `darwin/arm64`), the same host every
prior evidence file in this milestone was captured on:

```
ProductName:  macOS
ProductVersion: 27.0
BuildVersion: 26A5388g
Darwin denver 27.0.0 Darwin Kernel Version 27.0.0 ... RELEASE_ARM64_T6041 arm64
Homebrew 6.0.16-2-g007333f
```

This is a **pre-release macOS version** — Homebrew itself warns "You are
using macOS 27. We do not provide support for this pre-release version." on
every `brew install` below. Stated plainly, as every prior evidence file in
this phase states it, not glossed over.

## What this file proves, and what it does not

Both observations below are **one-time recorded mutations**, not standing
regression tests (D-12/D-17). Neither re-fires on its own. This is not a
footnote — it is stated beside each result below, because it is the reason
the two positive assertions inside `hooks.post.install` (D-11) — man-page
presence, version equality — are the gate's *only* permanent protection, and
the reason `hooks.post.install`'s cross-repository write scope is likewise
protected only by the App's installation list, not by a running check.

---

## BREW-05 — the install gate demonstrated RED, twice, for two different reasons

### Setup common to both mutations

A goreleaser binary was built the same way `release:rehearse-cask` builds
one (`GOWORK=off go build -modfile=go.tool.mod -o <tmp>/goreleaser
github.com/goreleaser/goreleaser/v2`), then invoked directly — never through
`Taskfile.yml`'s `release:rehearse-cask` target, since this plan's own
mutations happen outside that target's scripted install/uninstall cycle and
`Taskfile.yml`/`.goreleaser.yaml` are not in this plan's `files_modified`.

**Run A — the correct, paired baseline** (real signed + notarized, tag
`v0.7.0`, no mutation):

```
$ op run --env-file=.env -- env CASK_REHEARSE=1 <goreleaser> release --snapshot --skip=publish,sign --clean
  • sign & notarize macOS binaries
    • notarizing and waiting - this might take a while  binary=dist/codegraph-darwin-amd64_darwin_amd64_v1/codegraph
    • notarized                                          binary=dist/codegraph-darwin-amd64_darwin_amd64_v1/codegraph
    • notarizing and waiting - this might take a while  binary=dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph
    • notarized                                          binary=dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph
      • took: 49s
  • release succeeded after 1m5s
```

- Local darwin/arm64 raw build, `version --json`: `{"version":"v0.7.0","commit":"dca3e8b3aeb01dba2aba537cb190dedbb7714bd6","date":"2026-08-10T18:16:23Z","go_version":"go1.26.5","os":"darwin","arch":"arm64"}`
- Rendered `dist/homebrew/Casks/codegraph.rb` declares `version
  "0.7.0-SNAPSHOT-dca3e8b"` (snapshot-mode pseudoversion — the same fidelity
  gap plan 03-02 already found and rewrites around).
- Darwin/arm64 zip, real signed+notarized bytes: sha256
  `a60e185e1b837b2319caa3663f3b2189efcdb5c69615fc4d616852616a75761a`.
- Rehearsal copy (`rehearsal.rb`): the `on_arm` `url` line rewritten to
  `file://.../runA/zip-original.zip`, and the top `version "..."` line
  rewritten to `0.7.0` — the same two-line rewrite `release:rehearse-cask`
  performs, reproduced by hand here since this plan does not invoke that
  target. `sha256 "a60e18...5761a"` (the `on_arm` checksum line) is left
  UNCHANGED, since Run A's zip is genuinely unmutated at this point — this
  is the paired-good render both mutations below are measured against.

**Plan 03-02's own clean-install result, placed beside these for the same
target producing all three verdicts:** `CASK-REHEARSE-EVIDENCE schema=2
snapshot_version=0.7.0-SNAPSHOT-91b6df6 cask_sha256=<...> url_mechanism=file
man_page_count=30 sentinel=present completion_count=3
reported_version=codegraph v0.7.0 (...) outcome=pass` (`03-02-SUMMARY.md`,
Task Commits §, D1–D6 coverage). Same target, same host, same mechanism —
a genuine `brew install --cask` of the unmutated render exits 0.

### Mutation 1 — a binary that cannot execute

**What was changed.** Run A's real zip was extracted; its `codegraph` entry
(a genuine Mach-O, Developer-ID signed and notarized) was replaced with the
first 1024 bytes of itself — a truncated, non-loadable file, closer to a
real corrupted/partial download than a wrong-architecture rebuild — then
re-zipped with the three untouched sibling files (`CHANGELOG.md`, `LICENSE`,
`README.md`).

**Confirmed applied** — mutated artifact's sha256 beside the original's:

| | codegraph binary sha256 | zip sha256 |
|---|---|---|
| Original (Run A, signed+notarized) | `eb2a929148559389c886404987746587fb010ea555d34c8f372f5e204db73284` | `a60e185e1b837b2319caa3663f3b2189efcdb5c69615fc4d616852616a75761a` |
| Mutated (truncated to 1024 bytes) | `89a6108cd92c0335b73024e12dbb62140dfb87ed94d09c54c777046aa030333e` | `5060025d0fc061c6f4ef3b84a2bfe53ff7c2148fb4c99547db34e9ffeecd88e7` |

`file codegraph` on the mutated entry still reported `Mach-O 64-bit
executable arm64` — the magic bytes and early load commands survive a
1024-byte truncation of a 63MB binary — but the file is not loadable; this
is the shape of failure the mutation targets, not a mislabeled file.

The cask's `on_arm` `sha256`/`url` pair were rewritten to point at this
mutated zip (`file://.../mutation1/mutated.zip`, sha256
`5060025d...88e7`) — Homebrew's own download-checksum verification is
therefore satisfied (it matches what was actually downloaded), so the
failure below is the post-install hook, not the downloader, exactly as this
plan requires.

**Install — verbatim, real `brew install --cask`:**

```
$ brew install --cask local-rehearsal/cask-mutation1/codegraph
==> Trusted cask local-rehearsal/cask-mutation1/codegraph
==> Would install 1 cask:
local-rehearsal/cask-mutation1/codegraph
Warning: You are using macOS 27.
We do not provide support for this pre-release version.
==> Fetching downloads for: local-rehearsal/cask-mutation1/codegraph
✔︎ Cask codegraph (0.7.0)
==> Installing Cask codegraph
==> Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'
Warning: An exception occurred within a child process:
  RuntimeError: Failed to generate bash completions from /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph: Failure while executing; `{"SHELL" => "bash"} /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph completion bash` was terminated by uncaught signal KILL. Here's the output:

Failed to generate zsh completions from /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph: Failure while executing; `{"SHELL" => "zsh"} /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph completion zsh` was terminated by uncaught signal KILL. Here's the output:

Failed to generate fish completions from /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph: Failure while executing; `{"SHELL" => "fish"} /opt/homebrew/Caskroom/codegraph/0.7.0/codegraph completion fish` was terminated by uncaught signal KILL. Here's the output:

Error: Failure while executing; `/usr/bin/env /opt/homebrew/bin/codegraph man /opt/homebrew/share/man/man1` was terminated by uncaught signal KILL.
==> Unlinking Binary '/opt/homebrew/bin/codegraph'
==> Purging files for version 0.7.0 of Cask codegraph
```

- **Exit status:** `1` (non-zero, confirmed via `$?` on the bare `brew
  install` invocation, never through a pipeline).
- **The raise comes from assertion one's own call site.** The failing
  command named in the `Error:` line is exactly assertion one's
  `system_command binary, args: ["man", man_dir]` — the man-page-generation
  step — not assertion two's version check, which the process never
  reached. `generate_completions_from_executable`'s three warnings (bash,
  zsh, fish) fired first and were correctly swallowed as warnings, not
  raises, per D-11's own comment on that artifact — reproducing, on a
  genuinely truncated download this time rather than a bad executable name,
  the same non-gating behavior 03-02's completions perturbation already
  measured.
- **Mechanism:** `SIGKILL` (exit "terminated by uncaught signal KILL"), the
  same Gatekeeper enforcement 03-01's own finding named for an
  ad-hoc-signed binary — here fired against a *truncated* signed binary
  instead, because the retained Mach-O header still carries load commands
  that reference a code-signature region the truncated file no longer has
  the bytes to satisfy, and Homebrew Cask unconditionally quarantines the
  download regardless of the original binary's real signing status.
- **Post-failure bin-path check:**
  ```
  $ ls -la /opt/homebrew/bin/codegraph
  ls: /opt/homebrew/bin/codegraph: No such file or directory
  $ ls -la /opt/homebrew/Caskroom/codegraph
  ls: /opt/homebrew/Caskroom/codegraph: No such file or directory
  $ brew list --cask codegraph; echo "exit=$?"
  exit=1
  ```
  No `codegraph` remains on the Homebrew prefix's bin path, and no Caskroom
  directory survives — Homebrew's own rollback (`Unlinking Binary` /
  `Purging files for version 0.7.0`) is independently reconfirmed here, not
  merely inferred from the log lines above.
- **Cleanup:** `brew untap local-rehearsal/cask-mutation1` — exit 0,
  "Untapped 1 cask (13 files, 15.4KB)."

### Mutation 2 — a binary that runs and is the wrong artifact (driven from the binary side)

Plan 03-02 already perturbed this relationship from the **cask** side
(editing the declared `version "..."` line against a real installed
binary). This mutation drives it from the **binary** side — the shape a
real mismatched download produces — leaving the cask's declared version
untouched and instead shipping a real, working, correctly-signed binary
that reports a *different* version.

**What was changed.** A temporary, local-only, never-pushed git tag was
created at `HEAD`:

```
$ git tag v0.0.0-cask-mutation2-binary-version
$ git describe --tags --abbrev=0
v0.0.0-cask-mutation2-binary-version
```

**Confirmed applied** — the tag resolved as `HEAD`'s nearest tag before the
build ran, so GoReleaser's `.Tag`-derived ldflags (the same
`-X .../internal/version.Version={{ .Tag }}` mechanism 03-02 traced) picked
it up. A second real, credentialed `goreleaser release --snapshot
--skip=publish,sign --clean` run (Run B) then genuinely signed and
notarized a raw darwin/arm64 binary under this tag:

```
  • snapshotting
    • building snapshot...  version=0.0.0-cask-mutation2-binary-version-SNAPSHOT-dca3e8b
  • sign & notarize macOS binaries
    • notarizing and waiting - this might take a while  binary=dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph
    • notarized                                          binary=dist/codegraph-darwin-arm64_darwin_arm64_v8.0/codegraph
      • took: 49s
  • release succeeded after 1m5s
```

Local darwin/arm64 raw build, `version --json`:
`{"version":"v0.0.0-cask-mutation2-binary-version","commit":"dca3e8b3aeb01dba2aba537cb190dedbb7714bd6","date":"2026-08-10T18:16:23Z","go_version":"go1.26.5","os":"darwin","arch":"arm64"}`
— a real, Developer-ID-signed, Apple-notarized binary; the only thing wrong
with it is which version string it reports.

**The tag was deleted immediately after the build captured what it needed**
(before any install was attempted):

```
$ git tag -d v0.0.0-cask-mutation2-binary-version
Deleted tag 'v0.0.0-cask-mutation2-binary-version' (was dca3e8b)
$ git tag -l | grep -c mutation2
0
$ git describe --tags --abbrev=0
v0.7.0
```

Byte-clean revert, confirmed by both an absence check and `describe`
resolving back to the real tag.

**The mutated cask:** a copy of Run A's `rehearsal.rb` (declaring `version
"0.7.0"`, unchanged) with only its `on_arm` `sha256`/`url` pair repointed at
Run B's zip:

```diff
-      sha256 "a60e185e1b837b2319caa3663f3b2189efcdb5c69615fc4d616852616a75761a"
-      url "file://.../runA/zip-original.zip"
+      sha256 "f27ab714dcf7388cdeafdcda48ef09393a9973c15bb88a314101c121dc78f168"
+      url "file://.../runB/zip-wrongversion.zip"
```

The declared `version "0.7.0"` line — the one field this mutation must
leave alone — is unchanged from Run A's correct rewrite; only the
downloadable bytes (Run B's real binary, reporting a different version)
were substituted.

**Install — verbatim, real `brew install --cask`:**

```
$ brew install --cask local-rehearsal/cask-mutation2/codegraph
==> Trusted cask local-rehearsal/cask-mutation2/codegraph
==> Would install 1 cask:
local-rehearsal/cask-mutation2/codegraph
Warning: You are using macOS 27.
We do not provide support for this pre-release version.
==> Fetching downloads for: local-rehearsal/cask-mutation2/codegraph
✔︎ Cask codegraph (0.7.0)
==> Installing Cask codegraph
==> Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'
Error: codegraph cask post-install: installed binary reports version "0.0.0-cask-mutation2-binary-version", cask declares "0.7.0"
==> Unlinking Binary '/opt/homebrew/bin/codegraph'
==> Purging files for version 0.7.0 of Cask codegraph
```

- **Exit status:** `1`.
- **The raise comes from assertion two.** Assertion one passed — the real
  signed/notarized binary DID run `man` successfully (no completions
  warnings, no man-generation error; the process reached and executed the
  version comparison) — and the message is exactly D-11's version-equality
  raise, quoting **both** values: the real reported version
  (`"0.0.0-cask-mutation2-binary-version"`) and the cask's declared version
  (`"0.7.0"`).
- **Post-failure bin-path check:**
  ```
  $ ls -la /opt/homebrew/bin/codegraph
  ls: /opt/homebrew/bin/codegraph: No such file or directory
  $ ls -la /opt/homebrew/Caskroom/codegraph
  ls: /opt/homebrew/Caskroom/codegraph: No such file or directory
  $ brew list --cask codegraph >/dev/null 2>&1; echo "exit=$?"
  exit=1
  ```
  No `codegraph` remains on the bin path afterward.
- **Cleanup:** `brew untap local-rehearsal/cask-mutation2` — exit 0,
  "Untapped 1 cask (13 files, 15.4KB)."

### Working tree and `dist/` restored byte-clean

```
$ git status --short
 M .gitignore
?? .envrc
```
(both pre-existing, unrelated to this plan — see the executor's sequential-
execution instructions).

```
$ git tag -l | grep -c mutation
0
```

`dist/` was never mutated in place — every corrupted or repointed cask/zip
was constructed in a separate scratch directory, never inside `dist/`
itself. Confirmed by re-hashing `dist/`'s own contents after both
mutations, against the values recorded at the moment each run produced
them:

```
$ sha256sum dist/homebrew/Casks/codegraph.rb
d883189515ad83d3bc2011098be1f8f8295b933ea7851a89a10fb3698e738fe7
$ sha256sum dist/codegraph_v0.0.0-cask-mutation2-binary-version_darwin_arm64.zip
f27ab714dcf7388cdeafdcda48ef09393a9973c15bb88a314101c121dc78f168   (matches Run B's own recorded value exactly)
$ brew list --cask 2>&1 | grep -i codegraph || echo none
none
$ brew tap 2>&1 | grep -i local-rehearsal || echo "no local-rehearsal taps remain"
no local-rehearsal taps remain
```

No brew-managed cask, no stray tap, and no mutated file leaked into
`dist/`.

### Accepted limitation — stated beside the result, not only in a footnote

**Neither observation above re-fires.** D-12 declined a standing regression
test for the install gate, on purpose. Both mutations in this section are
one-time, hand-driven, confirmed-applied-and-reverted observations — the
next person who changes `hooks.post.install` gets no automatic signal if
either assertion becomes a no-op (an anchor that stops matching, a
`system_command` call whose failure stops propagating). The compensating
control is not a test suite; it is D-11's decision to specify **two**
positive assertions covering two structurally different failure classes
(a binary that cannot run at all, a binary that runs and is wrong) rather
than one, and 03-02's decision to make the version comparison an equality
rather than a containment test. Those two design choices are what is left
protecting BREW-05 once this plan's own recorded mutations stop being
current.

---

## BREW-02 — the tap credential's write scope, refused against the release repository

### Token minted the same way the release path mints it

Per plan 03-03 (`release.yml`'s in-job mint step), the tap-scoped GitHub
App's installation token comes from `actions/create-github-app-token`,
which itself performs exactly the App-JWT → installation-access-token
exchange the GitHub Apps REST API defines. This observation reproduces
that same two-step exchange directly against the API (no `create-github-
app-token` Action available outside a workflow run), using App id `4549710`
(`seanb4t homebrew tap publishing`, non-secret, already recorded in
`03-03-SUMMARY.md`) and installation id `152719025` (also already
recorded).

The App's private key was retrieved via `op document get
t3rawp5moh7pfybhidfj2myr3m --out-file <path outside the repo>`, used only
from that path, and destroyed (`rm -P`) immediately after the installation
token was minted — never printed, never committed, never transcribed. The
minted installation token itself was likewise never printed to this
document or any transcript; only its HTTP effects are recorded below.

```
$ curl -s -H "Authorization: Bearer <JWT>" https://api.github.com/app
HTTP_STATUS=200
{"id":4549710,"slug":"seanb4t-homebrew-tap-publishing", ...
 "permissions":{"contents":"write","metadata":"read"}, ...}

$ curl -s -H "Authorization: Bearer <JWT>" https://api.github.com/app/installations
HTTP_STATUS=200
[{"id":152719025,"account":{"login":"seanb4t"},"repository_selection":"selected"}]

$ curl -s -X POST -H "Authorization: Bearer <JWT>" \
    https://api.github.com/app/installations/152719025/access_tokens
HTTP_STATUS=201
{token_len=383 expires_at=2026-08-10T19:30:18Z
 permissions={'contents': 'write', 'metadata': 'read'} repository_selection=selected}
```

**App's installation repository-access list**, as recorded in
`03-03-SUMMARY.md` and reconfirmed by the `installations` call above:
installed on `seanb4t/homebrew-tap` alone (`repository_selection:
"selected"`, `installation_count=1`). This is the configuration the two
results below are a consequence of.

### Positive control — a write against the tap that SUCCEEDED

Without this, a refusal against `codegraph-go` proves nothing: a token that
cannot write anything at all produces the identical refusal to a correctly
scoped one — the false-positive shape this repository has been bitten by
before (a verifier that never downloaded the artifact it then read). The
write chosen is a throwaway git ref, created and then deleted — benign and
fully reversible.

```
$ curl -s -X POST -H "Authorization: token <TOKEN>" \
    https://api.github.com/repos/seanb4t/homebrew-tap/git/refs \
    -d '{"ref":"refs/heads/scratch-03-04-boundary-probe","sha":"dcb477996c312bcc3c2acd840e13c1088758b0bd"}'
HTTP_STATUS=201
{"ref":"refs/heads/scratch-03-04-boundary-probe", ...
 "object":{"sha":"dcb477996c312bcc3c2acd840e13c1088758b0bd","type":"commit"}}
```

**Reverted:**

```
$ curl -s -X DELETE -H "Authorization: token <TOKEN>" \
    https://api.github.com/repos/seanb4t/homebrew-tap/git/refs/heads/scratch-03-04-boundary-probe
HTTP_STATUS=204
```

**Tap repository confirmed to still contain only what plan 03-03 seeded,**
after the revert:

```
$ curl -s https://api.github.com/repos/seanb4t/homebrew-tap/branches
HTTP_STATUS=200
[{"name":"main", ...}]          # exactly one branch, the scratch ref is gone

$ curl -s https://api.github.com/repos/seanb4t/homebrew-tap/contents/
HTTP_STATUS=200
[{"name":"LICENSE", ...}, {"name":"README.md", ...}]   # exactly what 03-03 seeded
```

### Negative proof — the same kind of write, refused against `seanb4t/codegraph-go`

A **read** against the release repository (`GET .../git/ref/heads/main`)
returned `200` — expected and not evidence of anything, since
`codegraph-go` is a public repository and reads of public repository data
are not the boundary criterion 5 asserts (the same distinction
`03-03-SUMMARY.md`'s "Methodological note" already recorded: a public
repo's readability is not a scope signal). The disambiguating call is the
**write**:

```
$ curl -s -X POST -H "Authorization: token <TOKEN>" \
    https://api.github.com/repos/seanb4t/codegraph-go/git/refs \
    -d '{"ref":"refs/heads/scratch-03-04-boundary-probe-should-be-refused","sha":"7b34ab15bfeebf2713819ab2e0e5e87d3074567d"}'
HTTP_STATUS=403
{
  "message": "Resource not accessible by integration",
  "documentation_url": "https://docs.github.com/rest/git/refs#create-a-reference",
  "status": "403"
}
```

**Verdict: REFUSED, by resource scope, not by credential validity.** `403`
with `"Resource not accessible by integration"` is GitHub's
resource-access-scope refusal — the shape this document's own read section
above needed a positive-write control to distinguish from a malformed- or
expired-credential failure. A malformed or invalid token would have failed
differently: `401` with `"Bad credentials"`, at every call above including
the ones against the tap that succeeded. Since the SAME token succeeded
(`201`) against the tap and was refused (`403`, resource-scoped) against
`codegraph-go`, the refusal is unambiguous on its own — no second
disambiguating call was needed.

### Accepted limitation — stated plainly, beside the result

**This observation does not re-fire.** D-17 declined a standing CI
assertion of this boundary. If the App's installation is later widened to
include `seanb4t/codegraph-go` (accidentally or otherwise), nothing here
notices — the next signal would be the App's own installation-repository
list changing, which nothing currently watches. What would notice, if this
were ever made standing: a scheduled job re-minting an installation token
and attempting the same refused write on a cadence, asserting the `403`
persists. That job does not exist; this is a one-time recorded observation,
by the same design choice D-12 made for the install gate above.

---

## BREW-06, half one — the failure-and-recovery mechanism: A STRUCTURAL ARGUMENT, NOT EXECUTED EVIDENCE

**Read this section's own limitation subsection before its claim.** Everything
above this line in this file is a recorded observation — a command run, its
verbatim output, a verdict. This section is different in kind: it is an
argument from the pinned GoReleaser module's own source, and it carries **no
executed run** anywhere in it. D-18R (maintainer decision, 2026-08-09) chose
this shape deliberately, after the originally-planned reproduction
(`--snapshot`) was falsified against the source before any plan was drawn
over it. Nothing below should be read, quoted, or summarized elsewhere as
"demonstrated," "proven," "verified," or "tested" — those words describe the
observations elsewhere in this file, not this section.

### The claim

A failed tap push leaves an otherwise-good release intact: the GitHub
Release, its raw binaries, `.zip` archives, checksums file, cosign bundles,
SBOMs, and build-provenance attestation are already complete by the time a
tap push is even attempted, so a push failure has nothing left to corrupt.

### The evidence — from the pinned module's own source, not documentation

All citations below are against `github.com/goreleaser/goreleaser/v2@v2.17.1`,
the version this repository pins in `go.tool.mod` and `release.yml`'s
`GORELEASER_VERSION`, read directly from the module cache
(`$(go env GOMODCACHE)/github.com/goreleaser/goreleaser/v2@v2.17.1`).

**1. `cask.Pipe{}` is a member of the MAIN run pipeline, where it RENDERS the
cask file.** `internal/pipeline/pipeline.go:155`:

```go
var Pipeline = append(
    BuildPipeline,
    ...
    // homebrew formula
    brew.Pipe{},
    // homebrew cask
    cask.Pipe{},          // line 155
    ...
    // publishes artifacts
    publish.New(),         // line 170
    ...
)
```

`cask.Pipe{}` here runs via its `Run(ctx)` method
(`internal/pipe/cask/cask.go:96-103`), which calls `client.NewReleaseClient`
and `runAll` — `doRun` writes `dist/homebrew/Casks/codegraph.rb` to local
disk. Nothing is pushed to any repository at this point; this is the render
step, and it runs whether or not `--snapshot` is set.

**2. `cask.Pipe{}` is ALSO a member of the PUBLISH pipeline, where it PUSHES
the rendered file to the tap — and the pipe's own source comment states
why it is ordered where it is.** `internal/pipe/publish/publish.go:44-64`:

```go
// New publish pipeline.
func New() Pipe {
    return Pipe{
        pipeline: []Publisher{
            blob.Pipe{},
            upload.Pipe{},
            artifactory.Pipe{},
            docker.Pipe{},
            docker.ManifestPipe{},
            dockerv2.Publish{},
            dockerdigest.Pipe{},
            ko.Pipe{},
            sign.DockerPipe{},
            snapcraft.Pipe{},
            // This should be one of the last steps
            release.Pipe{},                              // line 59
            // brew et al use the release URL, so, they should be last
            nix.New(),
            winget.Pipe{},
            brew.Pipe{},
            cask.Pipe{},                                  // line 64
            aur.Pipe{},
            ...
        },
    }
}
```

The comment at line 60 — quoted verbatim above, `// brew et al use the
release URL, so, they should be last` — is the pipeline author's own stated
reason `cask.Pipe{}` sits after `release.Pipe{}` (line 59) in this list.
Here it runs via a **different** method on the same type,
`Publish(ctx)` (`internal/pipe/cask/cask.go:106-112`), which calls
`client.New(ctx)` — the release's own `GITHUB_TOKEN`-or-configured-token
client, template-resolved per `.goreleaser.yaml`'s
`homebrew_casks[0].repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` — and
`publishAll`/`doPublish` push the commit to `seanb4t/homebrew-tap`.

`internal/pipe/publish/publish.go:85-103`'s `Run` method executes this
`pipeline` slice **in order**, propagating the first hard error
(`cask.Pipe{}` also implements `Continuable`/`ContinueOnError() bool { return
true }`, so a cask-publish failure is memoized and does not abort sibling
publishers, but it is still evaluated strictly after `release.Pipe{}` has
already run to completion in the same ordered loop). By the time
`cask.Pipe{}.Publish()` is reached, `release.Pipe{}.Publish()` — the step
that uploads every release asset — has already returned without error.

**Same type, two interfaces, two different moments.** `cask.Pipe{}` is one
Go value implementing both `pipeline.Piper` (`Run`, called from the render
step, line 155) and `publish.Publisher` (`Publish`, called from the publish
step, line 64) — an argument citing only one of the two list memberships
would describe half the mechanism. The render always happens (nothing in
`--snapshot` skips it); only the push is what a `--snapshot` run removes,
per the next point.

**3. Why the originally-specified reproduction (`--snapshot`) cannot reach
the push at all.** `cmd/release.go:161-163`:

```go
if ctx.Snapshot {
    skips.Set(ctx, skips.Publish, skips.Announce, skips.Validate)
}
```

and `internal/pipe/publish/publish.go:82-83`:

```go
func (Pipe) String() string                 { return "publishing" }
func (Pipe) Skip(ctx *context.Context) bool { return skips.Any(ctx, skips.Publish) }
```

A `--snapshot` run sets `skips.Publish`, and `publish.Pipe{}.Skip()` checks
exactly that flag — so the **entire** publish pipeline (`publish.New()`'s
`pipeline` slice, all 22 publishers including `cask.Pipe{}.Publish()`) is
skipped as a unit under `skip.Maybe` in the outer pipeline runner. There is
no partial-skip mode that runs `release.Pipe{}.Publish()` while still
reaching `cask.Pipe{}.Publish()`; the render (list membership 1, above)
still runs under `--snapshot`, which is exactly what
`Taskfile.yml`'s `release:rehearse-cask` target exercises and all of this
plan's predecessor plans' local rehearsals relied on — but the push never
executes locally, by construction, regardless of any other flag.

**4. `HomebrewCask.SkipUpload` exists but PREVENTS the push rather than
FAILING it — not a substitute for observing a failure.**
`pkg/config/config.go:226` (the `HomebrewCask` struct, confirmed by reading
the struct itself rather than assuming field parity with the deprecated
`Homebrew` formula struct one screen above it, which has a same-named but
independently-declared field):

```go
type HomebrewCask struct {
    ...
    SkipUpload            string    `yaml:"skip_upload,omitempty" ...`
    ...
}
```

consumed in `internal/pipe/cask/cask.go:141-149`'s `doPublish`:

```go
func doPublish(ctx *context.Context, cask *artifact.Artifact, cl client.Client) error {
    brew := artifact.MustExtra[config.HomebrewCask](*cask, brewConfigExtra)
    if strings.TrimSpace(brew.SkipUpload) == "true" {
        return pipe.Skip("brew.skip_upload is set")
    }
    if strings.TrimSpace(brew.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
        return pipe.Skip("prerelease detected with 'auto' upload, skipping homebrew publish")
    }
    ...
```

`pipe.Skip(...)` is a **skip** sentinel, handled by `publishAll`
(`cask.go:124-139`) as `pipe.IsSkip(err)` and memorized into a non-fatal
`SkipMemento` rather than surfaced as a publish failure. Setting
`skip_upload` therefore reproduces "the push did not happen" — the render
still occurs, so `dist/homebrew/Casks/codegraph.rb` is still inspectable —
but it cannot reproduce "the push was attempted and failed," which is the
shape of event this claim is actually about. `.goreleaser.yaml` sets no
`skip_upload` key on this project's `homebrew_casks:` block; it is named
here only to rule it out as an alternate reproduction path, not because
this project uses it.

### The limitation — stated here, in this section's own body, not a footnote

**This half has NO executed evidence, and none is planned.** It has never
been observed to fire, in this project or (so far as this evidence file's
author can determine) in any rehearsal against this pinned module version.
The maintainer raised, considered, and explicitly rejected a
scratch-repository rehearsal (a real `goreleaser release` against a
throwaway tap with a deliberately bad or withheld tap token) as the
alternative to this argument, accepting the cost that follows from that
choice rather than paying for the rehearsal (D-18R, `03-CONTEXT.md`).

An argument from source-code ordering is not a demonstration, and nothing in
this repository will notice if that ordering changes in a future GoReleaser
version — a hypothetical v2.18 that reordered `publish.New()`'s pipeline
slice to run `cask.Pipe{}` before `release.Pipe{}`, for example, would
silently invalidate everything cited above, and this project's test suite,
CI, and release pipeline would all stay green through that change. Two
things would close this gap, neither of which this phase builds:

1. **A shape test asserting the ordering against the pinned module** — for
   example, reading `publish.New()`'s returned `pipeline` slice via
   reflection or a small harness importing `internal/pipe/publish`, and
   asserting `release.Pipe{}`'s index precedes `cask.Pipe{}`'s index. This
   would turn a future GoReleaser version bump that reorders the slice into
   a red test at `go.tool.mod` bump time, rather than a silent invalidation
   discovered only if a tap push ever actually fails badly.
2. **A rehearsal against a disposable, throwaway repository** — the
   alternative D-18R considered and rejected for this phase, still available
   to a future phase that wants executed evidence instead of an argument.

Both remedies are named so the gap is recorded with what would close it, not
left as an unlabelled dead end.

### Scope note carried from this plan's maintainer-directed reduction

The original plan (`03-05-PLAN.md`) proposed cutting a second release
specifically to exercise more of GoReleaser's tap-push mechanics on a
regenerated cask. The maintainer reduced that scope before execution
(recorded in the executor's own task context, not reproduced in full here):
a second release inside this phase would exercise GoReleaser's tap-push
**UPDATE** path and `brew upgrade`'s consumption of it — both of which are
code this project does not own and cannot patch, and which surface on the
next release regardless. This section's argument is unaffected by that
reduction: the ordering claim above depends only on `publish.New()`'s pipe
list, not on how many releases have been cut. **Named plainly, not
buried:** the tap-push UPDATE path (a second write to an already-existing
`Casks/codegraph.rb`) and `brew upgrade codegraph` consuming a
GoReleaser-regenerated cask both remain unexercised by this phase and
accepted as a gap until the next natural release exercises them.

---

## BREW-06, half two — release integrity: EXECUTED EVIDENCE (post-release verification, `v0.8.0`)

**Labelled here, immediately after half one above and in deliberate
contrast to it, as executed evidence — not an argument.** A reader scanning
this file's headings sees "STRUCTURAL ARGUMENT, NOT EXECUTED EVIDENCE"
directly followed by "EXECUTED EVIDENCE" for the same requirement's two
halves, without needing to read either in full.

`post-release-verify.yml` run
[`31424108520`](https://github.com/seanb4t/codegraph-go/actions/runs/31424108520)
(`workflow_run`, triggered by `release.yml` run `31423733320`'s completion)
— `completed`/`success`, all 7 jobs green:

```
resolve and validate the tag under verification .......... success
self-upgrade proof (darwin/arm64) ......................... success
self-upgrade proof (linux/amd64) .......................... success
Gatekeeper verdict (darwin/amd64) ......................... success
Gatekeeper verdict (darwin/arm64) ......................... success
verify supply-chain claims against the published release .. success
notarized suite proof (darwin/arm64) ...................... success
```

The verify-supply-chain job's own recorded verdict, from its log:

```
verify:release-assets: TAG=v0.8.0 REPO=seanb4t/codegraph-go
verify:release-assets: all required assets visible after 1 attempt(s)
verify:release-assets: OBSERVED 17 total published assets
verify:release-assets: cosign verify-blob: Verified OK
verify:release-assets: PASS — checksums, cosign, and attestation all
  verified against re-downloaded published assets for v0.8.0
```

Every claim was re-checked against **assets re-downloaded from the published
release** — never a local `dist/` copy — exactly as `verify:release-assets`
is designed to do.

**Asset-list comparison, `v0.7.0` vs `v0.8.0`:**

```
$ gh release view v0.8.0 --json assets --jq '[.assets[].name] | sort' | wc -l
17
$ gh release view v0.7.0 --json assets --jq '[.assets[].name] | sort' | wc -l
17
```

Both lists contain exactly the same 17-entry shape (1 checksums file + 4 raw
binaries + 4 `.zip` archives + 4 `.sigstore.json` bundles + 4 `.spdx.json`
SBOMs), differing **only** in the version string embedded in each asset
name. **Verdict: no duplicated and no orphaned assets, and the difference
between the two releases' asset lists is exactly the intended one — the
version bump, nothing else.** The tap push added zero entries to this list
in either release: `Casks/codegraph.rb` lives in `seanb4t/homebrew-tap`, a
separate repository, never as a release asset on `codegraph-go` — confirmed
structurally by the fact that neither list contains anything cask-shaped.

---

## BREW-01 — the cold install: EXECUTED EVIDENCE (against the published `v0.8.0` release)

All observations below were captured 2026-08-10 on the maintainer's real Mac
(Apple Silicon, `darwin/arm64`), the same host every prior evidence file in
this milestone was captured on:

```
ProductName:  macOS
ProductVersion: 27.0
BuildVersion: 26A5388g
Homebrew 6.0.16-2-g007333f
```

This is the same **pre-release macOS version** every prior evidence file in
this milestone states plainly — Homebrew warns "You are using macOS 27. We
do not provide support for this pre-release version." on every `brew
install` below.

### The release this evidence is against

One release was cut for this plan (the scope reduction below explains why
only one): `v0.8.0`, published from tag `v0.8.0`
(`0798c751feb188b8ea30baf2a46cd63a209e6692`), via `release.yml` run
[`31423733320`](https://github.com/seanb4t/codegraph-go/actions/runs/31423733320) —
`completed`/`success`. The tap's `Casks/codegraph.rb` was written **once**,
by this one release, by the App and by nothing else — see "BREW-02 — the tap
publication" below for the commit record.

**The install verified below used the cask GoReleaser rendered and pushed
for THIS release — not a cask hand-checked at authoring time.** No human
edited `Casks/codegraph.rb`; the file this install resolved was generated by
`goreleaser release`'s `cask.Pipe{}.Run()` (render) and pushed by its
`Publish()` (the same pipe, two methods — see the BREW-06 argument above)
inside `release.yml` run `31423733320`, then fetched fresh by `brew tap` on
this machine in a separate process with no shared state. That is the
property ROADMAP criterion 1 asks for (see the criterion 1 amendment below
for why "at least one release later" is no longer the right phrase for it).

### Starting state — torn down, not never-installed (stated plainly, per the plan's own honesty requirement)

This machine is the maintainer's daily-use development Mac, not a
never-had-codegraph machine — it carries residue from every prior plan's
local rehearsals in this phase. **The starting state for this section's
install is a torn-down machine, a weaker form of evidence than a genuinely
virgin one, and this is named rather than smoothed over:**

```
$ brew list --cask 2>&1 | grep -i codegraph || echo "no codegraph cask installed"
no codegraph cask installed
$ brew tap 2>&1 | grep -i seanb4t || echo "no seanb4t tap"
no seanb4t tap
$ ls "$(brew --prefix)/share/man/man1/" | grep -c '^codegraph'
0
$ find "$(brew --prefix)" -iname "*.codegraph-brew-install*"
(no output)
$ ls -la /opt/homebrew/bin/codegraph
ls: /opt/homebrew/bin/codegraph: No such file or directory
```

**Getting to this state required an explicit cleanup this plan performed and
is recording, because it is itself a finding.** Before the above was true,
`$(brew --prefix)/share/man/man1/` held **30 orphaned `codegraph*.1` man
pages**, dated from earlier in this same day — residue from plan 03-04's
Mutation 2 (`brew install --cask local-rehearsal/cask-mutation2/codegraph`),
whose post-install hook's assertion one (man-page generation) genuinely
succeeded before assertion two raised. **Homebrew's own install-failure
rollback (`Cask::Installer`'s `rescue => e; ensure purge_versioned_files;
raise e`) purges the Caskroom's versioned directory but never invokes the
cask's `uninstall` hook — the hook that is the only thing that knows to
remove files written outside the Caskroom, like the shared man directory.**
D-07's symmetric-uninstall design assumes `brew uninstall` runs; a *failed
install's own rollback* is a third path that runs neither hook's uninstall
half. This is a real, previously-unrecorded asymmetry in the cask's cleanup
story: **files a hook writes outside the Caskroom survive a failed install's
rollback, and are only removed by a subsequent successful `brew uninstall`
(confirmed below) — never by the failure path itself.** Phase 4, which reads
the sentinel this same hook writes, should know that a failed-then-abandoned
install can leave the sentinel behind with no cask installed to explain it.
The 30 stray pages were removed by hand (`rm`) before this section's install,
not by any cask mechanism.

### `brew tap seanb4t/tap` — resolves

```
$ brew tap seanb4t/tap
==> Auto-updating Homebrew...
==> Tapping seanb4t/tap
Cloning into '/opt/homebrew/Library/Taps/seanb4t/homebrew-tap'...
Tapped 1 cask (15 files, 17.7KB).
```

Exit `0`.

### `brew install codegraph` — a real, previously-unrecorded gate: untrusted-tap refusal

The literal, verbatim, published two-command line (`brew tap seanb4t/tap &&
brew install codegraph`) does **not** succeed unmodified on this Homebrew
version. The second command refuses on a genuinely new machine the first
time it is run:

```
$ brew install codegraph
Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap seanb4t/tap.
Run `brew trust --cask seanb4t/tap/codegraph` or `brew trust seanb4t/tap` to trust it.
```

Exit `1`. This is Homebrew 6.0.16's own tap-trust mechanism (`brew trust
--help`: "Trust non-official tap formulae, casks or commands so Homebrew may
load them" — a local, per-user trust store at
`${XDG_CONFIG_HOME:-~/.homebrew}/trust.json`), not a defect in this cask or
this tap — it is the same refusal any third-party tap's first install would
hit on this Homebrew version. It was not anticipated by this phase's earlier
plans (03-01 through 03-04's local-rehearsal installs pre-date this refusal
being hit against a real, non-local tap, or ran on a Homebrew version/session
where the tap was already trusted). **This is a real documentation gap**,
closed below (`README.md`/`docs/RELEASE.md`).

Following the refusal's own remedy, exactly as a real user reading the error
would:

```
$ brew trust --tap seanb4t/tap
Trusted tap: seanb4t/tap
$ brew install codegraph
==> Would install 1 cask:
seanb4t/tap/codegraph
Warning: You are using macOS 27.
We do not provide support for this pre-release version.
==> Fetching downloads for: seanb4t/tap/codegraph
✔︎ Cask codegraph (0.8.0)
==> Installing Cask codegraph
==> Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'
🍺  codegraph was successfully installed!
```

Exit `0`. The post-install gate (BREW-05, D-11's two assertions) ran and
passed silently — no raise, no rollback — confirming both that man-page
generation produced output and that the installed binary's reported version
equals the cask's declared version, against a real published artifact rather
than a rehearsal copy.

### The installed binary runs a real command, reporting the published version

```
$ /opt/homebrew/bin/codegraph version --json
{"version":"v0.8.0","commit":"0798c751feb188b8ea30baf2a46cd63a209e6692", ...}
```

matches the tag and the commit `release.yml` run `31423733320` built from.
Not `--version` alone — a real operation, indexing a fresh fixture
repository:

```
$ codegraph init   # inside a scratch git repo with one Go file
files=1 nodes=3 edges=3 duration=44ms
$ codegraph status
Index Statistics:
  Files:     1
  Nodes:     3
  Edges:     3
  Backend:   pebble
Nodes by Kind:
  function        2
  file            1
Files by Language:
  go              1
Index is up to date.
```

### Completion, in bash, zsh AND fish — three separate verdicts, each driven through a real interactive shell (tmux `send-keys`/`capture-pane`, never a synthetic function call)

**bash — OFFERED, conditional on `bash-completion` being installed and
sourced.** Under a bare `/bin/bash --noprofile --norc` with only
`codegraph`'s own completion file sourced (no `bash-completion` package),
`codegraph <TAB>` falls back to filename completion and prints
`_get_comp_words_by_ref: command not found` — Cobra's generated script's
own internal fallback (`__codegraph_init_completion`) still calls that
helper, so it is not truly self-contained. With Homebrew's `bash-completion`
(v1, `1.3_3`, already installed on this machine) sourced first — the state a
user gets from `brew install bash-completion` plus Homebrew's own shell
setup — real subcommand completion with descriptions is offered:

```
$ codegraph <TAB>
affected    (List test symbols impacted by changes to the given files)
callees     (List a symbol's forward call targets)
...
version     (Print build version information)
```

**This mirrors the man-path caveat already in `docs/RELEASE.md`: a
dependency on Homebrew's own shell integration having been sourced, not a
defect in the generated completion itself** — recorded here as a second
instance of the same class of caveat, and closed with the same kind of
documentation fix below.

**zsh — OFFERED**, under `/bin/zsh --no-rcs` with `compinit` run and
Homebrew's `shellenv` sourced (the standard zsh completion bootstrap):

```
$ codegraph <TAB>
status      -- Report index health and counts
sync        -- Incrementally update the graph from changed files
...
```

**fish — OFFERED**, via fish's own vendor-completions auto-load
(`/opt/homebrew/share/fish/vendor_completions.d/codegraph.fish`, populated
by Homebrew's fish integration once `fish_complete_path` includes the vendor
directory — the default for a normal, non-`--no-config` fish session):

```
$ codegraph <TAB>
affected    (List test symbols impacted by changes to the given files)
...
version    (Print build version information)
```

All three verdicts are **subcommand names with real descriptions offered**,
not merely "three files exist under `/opt/homebrew/etc`/`share`" — the check
this plan's own `must_haves` explicitly distinguishes.

### `man codegraph` — both invocations, the distinct results already documented, reproduced against this real install

```
$ man codegraph
CODEGRAPH(1)                                                      CODEGRAPH(1)
NAME
     codegraph - Pre-indexed code knowledge graph for coding agents
...
```

Exit `0` — in this shell, which already has Homebrew's environment sourced.
Reproducing `docs/RELEASE.md`'s own measured caveat in a genuinely clean
environment (`env -i`, no Homebrew shellenv):

```
$ env -i HOME="$HOME" PATH="/usr/bin:/bin" man codegraph
No manual entry for codegraph
$ env -i HOME="$HOME" PATH="/usr/bin:/bin" man "/opt/homebrew/share/man/man1/codegraph.1"
CODEGRAPH(1)                                                      CODEGRAPH(1)
...
```

The difference between the two is exactly the caveat `docs/RELEASE.md`
already documents from 03-04's evidence, now reproduced against a cold
install of the published, tap-resolved cask rather than a rehearsal.

### `codegraph upgrade --check` under the brew-managed install

```
$ codegraph upgrade --check
```

Behaves as the direct-download path does — Phase 4 (`UPGR-01/02/03`) is what
teaches `codegraph upgrade` to recognize a brew-managed install and refuse;
this plan does not add that recognition. Recorded as the actual, current
behavior, not a defect: today `codegraph upgrade --check` under a
brew-managed install reports availability exactly as it would for any other
install, with no brew-aware refusal or pointer to `brew upgrade codegraph`
yet. This is Phase 4's starting observation, not this phase's gap.

### Post-uninstall — the OTHER half of the asymmetry found above: the SUCCESS path is symmetric

Where the failed-install rollback (above) leaves hook-written files behind,
a genuinely successful `brew uninstall` — the intended, documented removal
path — is confirmed symmetric, exactly as D-07 designed it:

```
$ brew uninstall --cask codegraph
==> Uninstalling Cask codegraph
==> Unlinking Binary '/opt/homebrew/bin/codegraph'
==> Purging files for version 0.8.0 of Cask codegraph
$ ls "$(brew --prefix)/share/man/man1/" | grep -c '^codegraph'
0
$ find /opt/homebrew -iname ".codegraph-brew-install"
(no output)
$ ls -la /opt/homebrew/bin/codegraph
ls: /opt/homebrew/bin/codegraph: No such file or directory
```

All 30 man pages and the sentinel are gone; the binary is gone. **The
asymmetry is specifically that a FAILED install's rollback does not reach
the uninstall hook — a SUCCESSFUL uninstall does, correctly, every time.**
Both halves are now recorded, for Phase 4's benefit: the failure path can
leave the sentinel behind with nothing installed to explain it; the success
path never does.

```
$ brew untap seanb4t/tap
Untapping seanb4t/tap...
Untapped 1 cask (15 files, 17.7KB).
```

The machine was left in the state this section found it: no codegraph cask,
tap, binary, sentinel, or man pages. (`seanb4t/tap` remains in this user's
local `trust.json` after untapping — trust is a standing local preference,
separate from tap presence, and untapping does not and should not revoke
it; this is Homebrew's own design, not something this plan reverts.)

---

## BREW-02 — the tap publication: EXECUTED EVIDENCE (the App wrote it, and only the App)

```
$ gh api repos/seanb4t/homebrew-tap/contents/Casks --jq '.[] | {name,size}'
{"name":"codegraph.rb","size":6816}

$ gh api "repos/seanb4t/homebrew-tap/commits?path=Casks/codegraph.rb" \
    --jq '.[] | {sha: .sha[0:7], author: .commit.author.name, message: .commit.message}'
{"sha":"7425f9b","author":"goreleaserbot","message":"Brew cask update for codegraph version v0.8.0"}

$ gh api "repos/seanb4t/homebrew-tap/commits?path=Casks/codegraph.rb" --jq 'length'
1
```

Exactly one commit on that path, authored by `goreleaserbot`
(`bot@goreleaser.com`) — never a human — matching the App's identity, and
exactly one file in `Casks/`. **Because this plan cuts one release only
(scope reduction, below), one commit is the correct and complete count** —
not evidence of an UPDATE path never having been exercised, which is named
as an accepted gap in its own section below.

The rendered cask's declared version and URLs:

```
version "0.8.0"
on_macos do
  on_intel do
    sha256 "5498b1804dc91af3bf70a82a76c6bf61678c65e28f5b2c0ffb0fc2ca3abe447b"
    url "https://github.com/seanb4t/codegraph-go/releases/download/v#{version}/codegraph_v#{version}_darwin_amd64.zip"
  end
  on_arm do
    sha256 "bf633a755b07a0015127a7c4989a791a7017f5498c6411be92959d8bc2859c6b"
    url ".../codegraph_v#{version}_darwin_arm64.zip"
  end
end
```

**Cross-checked against the real, published release assets** (not merely
against the cask's own declared checksums — a cask that declares a
consistent-with-itself but wrong checksum would pass a self-check and fail a
real download):

```
codegraph_v0.8.0_darwin_amd64.zip
   declared in tap cask : 5498b1804dc91af3bf70a82a76c6bf61678c65e28f5b2c0ffb0fc2ca3abe447b
   sha256 of real asset : 5498b1804dc91af3bf70a82a76c6bf61678c65e28f5b2c0ffb0fc2ca3abe447b  -> MATCH
codegraph_v0.8.0_darwin_arm64.zip
   declared in tap cask : bf633a755b07a0015127a7c4989a791a7017f5498c6411be92959d8bc2859c6b
   sha256 of real asset : bf633a755b07a0015127a7c4989a791a7017f5498c6411be92959d8bc2859c6b  -> MATCH
```

**This match is the actual positive proof `HOMEBREW_TAP_TOKEN` was
non-empty and authenticated cross-repo, not a silent fallback.**
`internal/client/client.go`'s `NewIfToken` falls back to the release's own
client on an empty template — a fallback that CANNOT write
`seanb4t/homebrew-tap` (BREW-02's own boundary, refused and recorded in
03-04's evidence). If the tap token had been empty, this push would either
have failed outright or (in a differently-configured pipeline) silently used
the wrong credential; a green `release.yml` log alone would not have shown
which. The App-authored commit plus the byte-matching checksums together are
what a green log alone cannot provide.

**A methodological note worth recording, because it is this repository's own
recurring failure family (rule `84d1gfpywd`) and it fired twice in the
course of gathering this evidence:** an early sha256 comparison script
grepped the cask's raw text for the literal filename
`codegraph_v0.8.0_darwin_arm64.zip` — a string that does not appear anywhere
in the file, because the URL template uses `#{version}` interpolation
(`codegraph_v#{version}_darwin_arm64.zip`). The grep matched nothing,
compared against an empty string, and printed a confident `!!! MISMATCH` for
both platforms — a **false alarm from a broken anchor**, not a vacuous pass.
The corrected check parses the `sha256`/`url` pairs structurally rather than
grepping for a literal that the template never produces. Worth stating
plainly: an assertion whose anchor stops matching can produce a confident
**wrong** verdict, not only a silently-vacuous one — the inverse failure
mode from the one rule `84d1gfpywd` is usually invoked against, and just as
capable of misleading a reader.

Cross-reference: 03-04's evidence already recorded the tap credential's
write scope refused against `seanb4t/codegraph-go` with a positive control
proving the same token could write the tap. Nothing here re-verifies that
boundary on an ongoing basis — the accepted limitation recorded there
(D-17: a one-time observation, not a standing check) stands unchanged.

---

## Scope reduction, recorded plainly: one release, not two

The plan this phase's Phase 3 wave 4 was written against (`03-05-PLAN.md`)
specified cutting **two** releases inside this phase, so that criterion 1
could be checked "against a cask GoReleaser regenerated rather than the one
hand-checked at first publish." Before execution, the maintainer reduced
this to **one** release. The reasoning, transcribed rather than
paraphrased:

- The second release would almost exclusively have exercised code this
  project does not own — GoReleaser's tap-push **UPDATE** path (a second
  write to an already-existing `Casks/codegraph.rb`) and Homebrew's `brew
  upgrade`. Neither is preventable or patchable from this repository; a
  regression in either surfaces on the next natural release and is
  patch-forward-able exactly as every other release-pipeline defect in this
  project already is.
- Criterion 1's original worry — a cask "hand-checked at first publish" —
  does not describe this pipeline's actual shape: GoReleaser renders the
  cask **and** pushes it within the same automated CI run
  (`release.yml`'s single `release` job), so there is no hand-check step
  the first release's cask could be dependent on in the first place. The
  criterion, as originally worded, was guarding against a curated-tap
  workflow this project never had.

**What one release genuinely proves, and what it does not:**

Proven by this section's evidence: the tap-token mint executing for real in
CI (not a set-but-empty degradation — `HOMEBREW_TAP_TOKEN` was non-empty and
the write it authenticated is checksum-matched above); the published cask
installing, cold, from a real GitHub release asset over the real network;
and criterion 4's release-integrity half, executed against this release.

**Named as an accepted gap, not implied to be covered:** GoReleaser's
tap-push **UPDATE** path (writing a *second* commit to an already-existing
`Casks/codegraph.rb`) and `brew upgrade codegraph`'s consumption of a
GoReleaser-regenerated cask both remain unexercised by this phase. Both
surface on the next natural release, at which point they become observable
without any special arrangement — this gap is not permanent, only deferred
past this phase's boundary.

---

*Phase: 03-homebrew-tap-cask*
*Plans: 03-04, 03-05*
*Captured: 2026-08-10*
