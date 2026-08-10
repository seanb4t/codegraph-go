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

*Phase: 03-homebrew-tap-cask*
*Plan: 03-04*
*Captured: 2026-08-10*
