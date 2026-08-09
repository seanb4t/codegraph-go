# Phase 2 — Recorded Evidence

Single recorded-observation file for phase `02-apple-signing-notarization`,
referenced by later plans rather than duplicated. Produced by plan 02-01
(Tasks 1–3), against the real, published `v0.5.1` darwin release assets —
never a local `dist/` copy, never a locally rebuilt binary.

All observations on this page were captured on **2026-08-09** on the
maintainer's real Mac (Apple Silicon, `darwin/arm64`) — the same host
`02-RESEARCH.md`'s D-19 ruling records its own measurements against
(`macOS 27.0`, per that document). `gh` was authenticated as `seanb4t` for
`seanb4t/codegraph-go`; `gh --version` reported `2.97.0`.

## Evidence-line schema

Defined once, for the whole phase, by plan 02-01 Task 1 (`Taskfile.yml`'s
`verify:gatekeeper`). Every evidence line in this phase, in every plan,
follows it:

- One line, beginning with the prefix, then space-separated `key=value` pairs.
- Keys are fixed per prefix and always all present, in a fixed order. A value
  that could not be obtained is never omitted — it is the literal sentinel
  `not-found` (the thing does not exist) or `unknown` (it exists but could
  not be read).
- Values are shell-safe: no spaces, no quotes, no newlines. Any value that
  would contain whitespace is emitted on its own separate labelled line
  immediately above, and the evidence field carries a short token instead.
- The first field is always a literal `schema` marker set to `1`, so a later
  consumer can detect a format change rather than mis-parse one.

For `verify:gatekeeper` the prefix is `GATEKEEPER-EVIDENCE`, and it always carries `schema=1` as its first field; its fixed field order is: `schema`, `tag`, `goos`, `goarch`, `sha256`, `gh_digest`, `digest_match`, `expect`, `observed`, `exit`, `xattr_present`, `source_assertion`, `syspolicy_exit`.

## SIGN-03 — RED baseline (v0.5.1 darwin assets)

Baseline release: `v0.5.1` (D-06 — never `v0.5.0`, which published zero
assets and cannot baseline anything). Both `codegraph_v0.5.1_darwin_arm64`
and `codegraph_v0.5.1_darwin_amd64` are deliberately un-notarized
(`docs/RELEASE-PROCEDURES.md` §7.1) and are preserved untouched by this
observation — `verify:gatekeeper` only ever reads them via
`gh release download` into a `mktemp -d`.

### Pre-xattr control (NON-EVIDENCE)

An assessment of a file that was never quarantined cannot stand in for the
real verdict, so it is recorded here, separately and explicitly labelled, so
it can never be mistaken for the evidence below.

**darwin/arm64**, before any xattr was applied:

```
GATEKEEPER-PRE-XATTR-CONTROL (NON-EVIDENCE — file was never quarantined at this point) exit=3
/var/folders/.../tmp.Vx5wCe4ock/codegraph_v0.5.1_darwin_arm64: rejected
```

**darwin/amd64**, before any xattr was applied: same shape (`exit=3`,
`rejected`), captured in the same run per the checkpoint transcript.

### darwin/arm64 RED observation

Run: `TAG=v0.5.1 REPO=seanb4t/codegraph-go GOOS=darwin GOARCH=arm64 GATEKEEPER_EXPECT=rejected GH_TOKEN=$(gh auth token) task verify:gatekeeper`

- Asset: `codegraph_v0.5.1_darwin_arm64`
- sha256 (recomputed from the downloaded asset): `69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d`
- GitHub's recorded digest (`gh release view v0.5.1 --json assets`): `sha256:69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d`
- Digest match: **true** — the two 64-hex-character values above are identical.
- SYNTHETIC `com.apple.quarantine` value written: `0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851`
- `xattr -p com.apple.quarantine` readback, confirmed present BEFORE assessment: `0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851` (matches the written value)
- `spctl -a -vv -t install` verbatim output and exit status:
  ```
  GATEKEEPER-SPCTL exit=3
  /var/folders/.../tmp.Vx5wCe4ock/codegraph_v0.5.1_darwin_arm64: rejected
  ```
- `syspolicy_check distribution` — **NON-GATING observation**, verbatim output and exit status:
  ```
  GATEKEEPER-SYSPOLICY-CHECK (NON-GATING) exit=70
    "Failed to find total signatures count in file:///.../codegraph_v0.5.1_darwin_arm64"
    "App has failed one or more pre-distribution checks."
    Adhoc Signed App — Severity: Warning — "This app is adhoc signed. While it may run locally, adhoc signed apps are not suitable for distribution." Type: Distribution Error
    Notary Ticket Missing — Severity: Fatal — "A Notarization ticket is not stapled to this application." Type: Distribution Error
  ```
  The target did NOT fail because of this — exit 70 / Fatal / Notary Ticket
  Missing is the CORRECT, expected, non-gating observation for an unstapled
  binary (D-16/DIST-06 puts stapling permanently out of scope).
- Full evidence line:
  ```
  GATEKEEPER-EVIDENCE schema=1 tag=v0.5.1 goos=darwin goarch=arm64 sha256=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d gh_digest=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d digest_match=true expect=rejected observed=rejected exit=3 xattr_present=true source_assertion=fail syspolicy_exit=70
  ```
- Target's own exit status: `0` (observed matched expect — `verify:gatekeeper: PASS`).

### darwin/amd64 RED observation

Run: `TAG=v0.5.1 REPO=seanb4t/codegraph-go GOOS=darwin GOARCH=amd64 GATEKEEPER_EXPECT=rejected GH_TOKEN=$(gh auth token) task verify:gatekeeper`

- Asset: `codegraph_v0.5.1_darwin_amd64`
- sha256 (recomputed from the downloaded asset): `943102ba763661a10700fb79d9fe56bc4ca7a70c3f75cd2f4380df51af76d8a1`
- GitHub's recorded digest: `sha256:943102ba763661a10700fb79d9fe56bc4ca7a70c3f75cd2f4380df51af76d8a1`
- Digest match: **true**.
- `spctl -a -vv -t install` exit status: `3` (`rejected`).
- `syspolicy_check distribution` — **NON-GATING observation**: same shape as
  the arm64 run (Adhoc Signed App / Severity Warning, Notary Ticket Missing /
  Severity Fatal), exit `70`; the checkpoint transcript recorded this pair
  matched the arm64 run rather than reproducing the full text a second time.
- Full evidence line:
  ```
  GATEKEEPER-EVIDENCE schema=1 tag=v0.5.1 goos=darwin goarch=amd64 sha256=943102ba763661a10700fb79d9fe56bc4ca7a70c3f75cd2f4380df51af76d8a1 gh_digest=943102ba763661a10700fb79d9fe56bc4ca7a70c3f75cd2f4380df51af76d8a1 digest_match=true expect=rejected observed=rejected exit=3 xattr_present=true source_assertion=fail syspolicy_exit=70
  ```
- Target's own exit status: `0`.

### Proof the gate can fail (not just report)

Deliberate mismatch: the arm64 asset re-assessed with
`GATEKEEPER_EXPECT=accepted`.

```
::error::observed verdict 'rejected' does not match GATEKEEPER_EXPECT='accepted'
GATEKEEPER-EVIDENCE schema=1 tag=v0.5.1 goos=darwin goarch=arm64 sha256=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d gh_digest=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d digest_match=true expect=accepted observed=rejected exit=3 xattr_present=true source_assertion=fail syspolicy_exit=70
task: Failed to run task "verify:gatekeeper": exit status 1
```

The evidence line was still printed on the fail path, as designed (step 12).
Shell script exit `1`; `task`'s own wrapper exit `201`.

### Proof input validation halts (before any network round-trip)

Run with `GATEKEEPER_EXPECT=yes` (a value outside the two allowed strings):

```
task: GATEKEEPER_EXPECT is set to a value other than 'accepted' or 'rejected'. An unvalidated value would fail safe but confusingly, turning every observation into a mismatch regardless of what spctl actually reports — value must be exactly one of the two allowed strings.
task: Failed to run task "verify:gatekeeper": task: precondition not met
```

`task` exit `201`. No download and no evidence line was produced — correct:
the precondition halts before any network round-trip, so there is nothing
yet to attest.

### GitHub digest availability (step 3a)

`gh release view v0.5.1 --repo seanb4t/codegraph-go --json assets` (gh
`v2.97.0`) exposes a non-null `digest` field for both darwin raw-binary
assets:

| Asset | GitHub digest | size |
|---|---|---|
| `codegraph_v0.5.1_darwin_amd64` | `sha256:943102ba763661a10700fb79d9fe56bc4ca7a70c3f75cd2f4380df51af76d8a1` | 64519472 |
| `codegraph_v0.5.1_darwin_arm64` | `sha256:69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d` | 62467714 |

Both matched the recomputed sha256 exactly, so `digest_match=true` on both
arches in every run above. See **Observations**, OBS-1 below: the
digest-ABSENT sentinel/hard-fail branches (`verify:gatekeeper`'s step 4) did
NOT fire against this baseline, because this `gh` version and this release
both carry the digest.

## D-19 — the oracle, and the gate that could never have passed

**The four checks that feel like verification and are not** (PROJECT.md Key
Decisions), each explicitly recorded as insufficient, for a self-contained
record:

1. **A green CI step.** This entire phase exists because green CI never
   proved Gatekeeper trust — nothing in `release.yml` or `ci.yml` has ever
   run `spctl` against a real, quarantined, published asset before this
   plan.
2. **`codesign -dvv` passing.** Measured (Phase 1 background, ROADMAP.md
   Phase 2 Notes) to already pass TODAY on the un-notarized, adhoc-signed
   darwin/arm64 binary that this plan's own RED observation (below) shows
   `spctl -a -vv -t install` rejects — it answers "is this signed," not
   "does Gatekeeper trust this."
3. **`notarytool history` showing Accepted.** Reports on Apple's notary
   service accepting a submission, not on what a locally-quarantined
   Gatekeeper install-time check does with the resulting binary — the two
   are different questions, and this plan's oracle answers the second one.
4. **`spctl` on a file that was never quarantined.** Recorded explicitly in
   this document as the "Pre-xattr control (NON-EVIDENCE)" subsection above,
   for both darwin arches — the observation that makes this insufficiency
   visible rather than silently conflated with the real verdict.

**Positive controls** (a notarized subject accepted, exit 0) — TWO
independent vendors, both re-measured against this repo's exact SYNTHETIC
quarantine value (`0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851`),
reproducing D-19's original ruling on the maintainer's own machine rather
than resting on that single prior observation:

| Subject | xattr readback | `spctl -a -vv -t install` | source | exit |
|---|---|---|---|---|
| `/tmp/pc-docker` (copy of `/usr/local/bin/docker`, Developer ID: Docker Inc, 9BNSXJN65R) | `0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851` | `accepted` | `Developer ID Application: Docker Inc (9BNSXJN65R)`, `source=Notarized Developer ID` | 0 |
| `/tmp/pc-codex` (copy of `/opt/homebrew/bin/codex`, Developer ID: OpenAI OpCo LLC, 2DC432GLL2) | `0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851` | `accepted` | `Developer ID Application: OpenAI OpCo, LLC (2DC432GLL2)`, `source=Notarized Developer ID` | 0 |

**Negative control** (this repo's shape rejected, exit 3): both RED
observations above — `codegraph_v0.5.1_darwin_arm64` and
`codegraph_v0.5.1_darwin_amd64`, adhoc/un-notarized, rejected under the
IDENTICAL synthetic quarantine string that the two vendors above were
accepted under.

The identical synthetic quarantine string discriminates correctly on
notarization status alone — `accepted`/exit 0 on a notarized binary,
`rejected`/exit 3 on the un-notarized codegraph asset — which is the
property that makes it a usable rig, not merely "a string that makes `spctl`
unhappy."

**What the old oracle (`spctl -a -vv -t exec` + gating `syspolicy_check`)
would have done.** Under the pre-D-19 criteria, this plan's RED baseline
would have gone red today for the RIGHT reason (adhoc, un-notarized) —
`-t exec` also rejects a bare Mach-O today, so the RED demonstration alone
would not have exposed the defect. The defect surfaces only after
notarization: `-t exec` rejects a bare Mach-O on SHAPE before notarization is
even considered (a CLI binary is not an "app"), so it would have STAYED red
after notarization landed, for a COMPLETELY DIFFERENT reason than the RED
baseline's — and `syspolicy_check distribution`'s Fatal/unstapled verdict
would have made that half of the old criteria a gate that could never pass
at all, since stapling is permanently out of scope (D-16/DIST-06). Discovery
would have landed only after plan 02-07's irreversible publish. This is why
the D-19 ruling replaced `-t exec` with `-t install` and demoted
`syspolicy_check` to a recorded, non-gating observation.

**Assumption A1 — CLOSED, not carried.** `02-RESEARCH.md`'s Assumptions Log
carried A1 as open (`syspolicy_check distribution`'s bare-Mach-O acceptance
and output shape were unverified). D-19's measurement, reproduced again in
this plan's Task 2 checkpoint on both darwin arches, settles it: the tool
DOES accept a bare Mach-O, its verdict for an un-notarized/unstapled binary
is `Notary Ticket Missing`, Severity Fatal, exit `70`, and it is demoted to a
non-gating observation for exactly that reason. A1 is CLOSED as of this
plan, not carried forward.

## Assumption A2 — synthetic vs genuine quarantine xattr

**Verdict: CONFIRMED.** The synthetic rig produces the same `spctl -a -vv -t
install` verdict as a genuine browser download, on a byte-identical file —
despite the two NOT sharing an identical quarantine attribute value.

| | Synthetic (`verify:gatekeeper`) | Genuine (Safari download) |
|---|---|---|
| sha256 of the asset | `69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d` | `69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d` (byte-identical) |
| `com.apple.quarantine` value | `0081;6a78a6d5;Safari;399D5E8C-4C3A-4491-9A41-725818DF9851` | `0083;6a78a93d;Safari;7EC9AF84-7772-4185-BA05-2F3EC4CEBE8A` |
| `spctl -a -vv -t install` verdict | `rejected` | `rejected` |
| exit status | 3 | 3 |

The quarantine flag byte itself DIFFERS (`0081` synthetic vs `0083`
genuine), the timestamp and event UUID differ (as expected — they encode
when/how the download happened), and the genuine copy additionally carries
`com.apple.metadata:kMDItemWhereFroms` (a bplist naming the real
`release-assets.githubusercontent.com` URL and
`https://github.com/seanb4t/codegraph-go/releases/tag/v0.5.1`) and
`com.apple.lastuseddate#PS`, neither of which the synthetic rig writes.
Despite those two observable differences, the `spctl -t install` verdict is
invariant on this byte-identical file: `rejected`/exit 3 in both cases.

**Conclusion, stated at the scope this evidence actually supports:** the
verdict does not hinge on faithfully reproducing the quarantine attribute's
exact flag byte, timestamp, or UUID, nor on the presence of
`kMDItemWhereFroms`. The synthetic rig `verify:gatekeeper` writes is adequate
for THIS gate (`spctl -a -vv -t install` on a bare Mach-O), for a stated,
measured reason — not by coincidence, and not by assumption. This finding is
NOT generalized beyond that scope: it was not tested against other `spctl`
assessment types, against an `.app` bundle, or against `.zip` archives.

**Consequence for `Taskfile.yml`.** Because A2 is CONFIRMED, no change to
`verify:gatekeeper` was needed or made in this task — the target's existing
comments (which state the xattr is a simulation "until Task 2's checkpoint
confirms it behaves like an actual browser download") describe exactly this
outcome and required no correction.
`TestVerifyGatekeeperDeclaresNamedPreconditions` still passes unchanged.

## Observations from checkpoint execution (not defects)

Recorded plainly, per the maintainer's direction, as observations — not as
work items for this plan.

**OBS-1 — the digest-ABSENT branch is unexercised.** `gh v2.97.0` exposes a
non-null `digest` on every `v0.5.1` asset (see "GitHub digest availability"
above), so `verify:gatekeeper`'s two digest-missing branches — the RED-path
sentinel-and-continue (`digest_match=unknown`, warn, proceed) and the
GREEN-path hard failure — never fired against this baseline. This is a real,
currently-untested code path inside a gate whose whole purpose is closing
the "check that cannot fire" failure family, and it should not later be
counted as covered by this checkpoint. Whether and how to force-exercise it
(e.g. by stubbing the digest lookup, or by finding an older release whose API
predates per-asset digests) is a follow-up decision, out of scope for this
plan.

**OBS-2 — `task`'s own exit code collapses distinct failure modes.** A
precondition halt (the invalid-value proof) and a verdict mismatch (the
deliberate-mismatch proof) both surfaced as `task` exit `201`, even though
the underlying shell script itself exits `1` for a verdict mismatch. This is
`task`'s own wrapper behavior, not a defect in `verify:gatekeeper`. The
evidence line (`GATEKEEPER-EVIDENCE schema=1 ...`) disambiguates the two
cleanly — a precondition halt produces no evidence line at all (nothing was assessed yet), while a
verdict mismatch always produces one with `expect=` and `observed=`
disagreeing — so the machine-parseable contract holds. Any downstream
consumer (plan 02-06's CI job included) must key off the evidence line's
presence and its `expect=`/`observed=` fields, never off `task`'s process
exit code alone, to distinguish these failure modes.

## SIGN-01 — local notarize rehearsal (D-08)

**This is a rehearsal observation, not the phase's GREEN criterion.** The
GREEN criterion (ROADMAP Phase 2 criterion 2, D-05's byte-identity chain)
requires the re-downloaded, published release asset — that does not exist
until plan 02-07's real, irreversible release. Everything recorded in this
section is a local build, signed and notarized with the maintainer's real
Developer ID Application certificate against Apple's real notary service,
but never published and never re-downloaded, and is therefore evidence for
the mechanism, not itself sufficient to close the criterion.

Captured **2026-08-09** on the maintainer's real Mac, macOS 27.0,
`darwin/arm64` host (Apple Silicon). Certificate:
`Developer ID Application: Sean Brandt (8D762W58T4)`, issued by Apple's
Developer ID Certification Authority, `notBefore=Aug 9 18:10:51 2026 GMT`,
`notAfter=Feb 1 22:12:15 2027 GMT`. The five `MACOS_*` credential values
were supplied via `op run --env-file=.env` as base64, deliberately the same
input form CI will use in plan 02-06.

### D-09 precondition proof (credential-free host)

With all five `MACOS_*` variables unset:

```
task: MACOS_SIGN_P12 is not set. release:rehearse-notarize is MAINTAINER-ONLY and
requires a real Developer ID Application certificate (base64-encoded .p12, or a
filesystem path — quill accepts either). See docs/RELEASE.md.
task: Failed to run task "release:rehearse-notarize": task: precondition not met
```

Exit `201`. Elapsed, measured under `time`: **0.036 s total** (0.02s user,
0.01s system). Halted BY NAME on the first missing credential
(`MACOS_SIGN_P12`), before any build work and before any network
round-trip — D-09 is satisfied and this is its first demonstration on a
genuinely credential-absent host.

### The real rehearsal (final run, shipped configuration, both darwin arches)

Baseline determinism — measured, not assumed; both notarize-disabled
double-builds byte-identical:

```
BASELINE-DETERMINISM-OK platform=darwin/arm64 sha256=40fd5bf03d3ee08e25370a524c8a8b5396a53c8b18ff548dccaf4d410ca289da
BASELINE-DETERMINISM-OK platform=darwin/amd64 sha256=856646e4c486da7f8ae065ca317e62f731a59fb8ba188690d3de06ef7708a4aa
```

`codesign -dvv`, both darwin arches — Team Identifier present in both:

```
REHEARSAL-CODESIGN-DVV platform=darwin/arm64
Format=Mach-O thin (arm64)
CodeDirectory v=20500 size=489781 flags=0x10000(runtime) hashes=15300+2 location=embedded
Authority=Developer ID Application: Sean Brandt (8D762W58T4)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
Timestamp=Aug 9, 2026 at 3:24:59 PM
TeamIdentifier=8D762W58T4
Runtime Version=12.1.0

REHEARSAL-CODESIGN-DVV platform=darwin/amd64
Format=Mach-O thin (x86_64)
CodeDirectory v=20500 size=505653 flags=0x10000(runtime) hashes=15796+2 location=embedded
Authority=Developer ID Application: Sean Brandt (8D762W58T4)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
Timestamp=Aug 9, 2026 at 3:24:35 PM
TeamIdentifier=8D762W58T4
Runtime Version=12.1.0
```

`spctl -a -vv -t install` (D-19's oracle, verdict from exit status, never a
substring search) against a synthetic-quarantine copy of each locally
notarized binary:

```
REHEARSAL-SPCTL platform=darwin/arm64 exit=0
  <tmpdir>/spctl-assess-darwin-arm64: accepted
  source=Notarized Developer ID
  origin=Developer ID Application: Sean Brandt (8D762W58T4)
REHEARSAL-SPCTL platform=darwin/amd64 exit=0
  <tmpdir>/spctl-assess-darwin-amd64: accepted
  source=Notarized Developer ID
  origin=Developer ID Application: Sean Brandt (8D762W58T4)
```

`NOTARIZE-EVIDENCE`/`SIGN04-EVIDENCE`, shipped configuration (`mutate=0`),
both darwin arches:

```
NOTARIZE-EVIDENCE schema=1 goos=darwin goarch=arm64 presign_sha256=40fd5bf03d3ee08e25370a524c8a8b5396a53c8b18ff548dccaf4d410ca289da final_sha256=016db76bc32c79ee1ecf089917ae2c685728fb27885c0c62be19211db8714c2e sha_changed=true team_id_present=true spctl_exit=0 spctl_verdict=accepted apple_status=notarized
SIGN04-EVIDENCE schema=1 goos=darwin goarch=arm64 final_sha256=016db76bc32c79ee1ecf089917ae2c685728fb27885c0c62be19211db8714c2e zip_content_sha256=016db76bc32c79ee1ecf089917ae2c685728fb27885c0c62be19211db8714c2e checksums_file_sha256=016db76bc32c79ee1ecf089917ae2c685728fb27885c0c62be19211db8714c2e cosign_verifies_presign=false cosign_verifies_final=true cosign_subject=final verdict=pass mutate=0 config_sha256=ce90b0c03ceedd04da2bbf2e151c102ed75a1ac9b1cbab4f599cb4e844d2b99f
NOTARIZE-EVIDENCE schema=1 goos=darwin goarch=amd64 presign_sha256=856646e4c486da7f8ae065ca317e62f731a59fb8ba188690d3de06ef7708a4aa final_sha256=b85e2c0befea3ad1a36a8f34ea3002dd44fbb91dbbdb4ce83c5e87ef91934443 sha_changed=true team_id_present=true spctl_exit=0 spctl_verdict=accepted apple_status=notarized
SIGN04-EVIDENCE schema=1 goos=darwin goarch=amd64 final_sha256=b85e2c0befea3ad1a36a8f34ea3002dd44fbb91dbbdb4ce83c5e87ef91934443 zip_content_sha256=b85e2c0befea3ad1a36a8f34ea3002dd44fbb91dbbdb4ce83c5e87ef91934443 checksums_file_sha256=b85e2c0befea3ad1a36a8f34ea3002dd44fbb91dbbdb4ce83c5e87ef91934443 cosign_verifies_presign=false cosign_verifies_final=true cosign_subject=final verdict=pass mutate=0 config_sha256=ce90b0c03ceedd04da2bbf2e151c102ed75a1ac9b1cbab4f599cb4e844d2b99f
release:rehearse-notarize: PASS (mutate=0)   [exit 0]
```

Committed `.goreleaser.yaml` sha256, from the `config_sha256` field above —
unchanged across every run in this document, including the mutation below:
`ce90b0c03ceedd04da2bbf2e151c102ed75a1ac9b1cbab4f599cb4e844d2b99f`.

### Snapshot-mode asset naming shape (expected, not a defect)

Under `--snapshot`, the tag segment resolves empty, so the four signature
sidecar names carry a doubled separator where the tag would sit. This is
the **expected** shape for a snapshot rehearsal, recorded here so it is
never later misread as a violation of the tagged-release naming contract;
it does not describe the real tagged release plan 02-07 produces.

### Apple submission timing — sequential, measured

The two notarize submissions inside the single `notarize.macos[0]` entry
run **sequentially** — `notary.MacOS{}` parallelizes ACROSS config entries
via an error group, never within one entry's per-binary loop. This was
already documented in `.goreleaser.yaml` (lines 236-243) from reading
`notary.MacOS{}` at source; this rehearsal is the first time it is
corroborated by measurement rather than only by reading the pipe's
implementation. Observed wall-clock notarize durations, goreleaser-reported,
across the runs performed this session: **1m16s and 49s** on the first full
run, **48s** on the final run recorded above. These are the basis for
`.goreleaser.yaml`'s configured 20-minute per-binary `timeout:` — comfortably
above every observed value.

**Apple pending/timeout behavior: OPEN, not measured by this rehearsal.** No
submission returned `pending` and none timed out in any of the four runs
performed this session (the real rehearsal twice, the mutation, and the
secret-leak spot check, which failed locally before reaching Apple).
`.goreleaser.yaml`'s comment states, from reading `notary.MacOS{}` at
source, that a timeout status and a pending status are both logged and the
pipe continues past rather than fails. That claim is a source-reading
claim, not a measured one — this rehearsal never exercised the code path it
describes, so the question is recorded here as still OPEN rather than
restated as settled. Carried to plan 02-06/02-07: if a real submission ever
returns pending or times out, that observation should be recorded here as
measured.

### D-03 — entitlements hypothesis: HOLDS

No `entitlements:` key was configured (per `.goreleaser.yaml`'s `sign:`
block), Apple notarized both darwin binaries under that omission, and
`spctl -a -vv -t install` accepted both. This proves Apple **accepted** the
binary under D-03's working hypothesis that this repository's CGo
tree-sitter grammars (compiled at build time, not runtime) need no
JIT/unsigned-executable-memory entitlement — it does **not** prove the
binary **runs correctly**; ROADMAP criterion 4 (the full CLI/MCP suite
against the real notarized asset) is what converts that remaining half from
"accepted" to "works."

### Legacy-RC2 PKCS#12 concern — not a problem for quill

The concern that Keychain Access P12 exports use legacy RC2 encryption
(requiring OpenSSL 3.6.3's `-legacy` flag to read) does not apply here:
`quill` read the certificate without incident.

## SIGN-04 — ordering, measured (D-07)

The converged and divergent hash sets, side by side, per darwin platform:

| Platform | Config | pre-sign sha256 | final sha256 | cosign verifies pre-sign | cosign verifies final | cosign subject |
|---|---|---|---|---|---|---|
| darwin/arm64 | shipped (`mutate=0`) | `40fd5bf03d3ee08e25370a524c8a8b5396a53c8b18ff548dccaf4d410ca289da` | `016db76bc32c79ee1ecf089917ae2c685728fb27885c0c62be19211db8714c2e` | false | true | final |
| darwin/amd64 | shipped (`mutate=0`) | `856646e4c486da7f8ae065ca317e62f731a59fb8ba188690d3de06ef7708a4aa` | `b85e2c0befea3ad1a36a8f34ea3002dd44fbb91dbbdb4ce83c5e87ef91934443` | false | true | final |
| darwin/arm64 | pre-D-18 mutation (`mutate=1`) | `40fd5bf03d3ee08e25370a524c8a8b5396a53c8b18ff548dccaf4d410ca289da` | `338b806608226297daebb979440b4dcefb40c3427a078419d508049d1b54c431` | true | false | presign |

**The relationship inverted, which is the whole point:**

```
shipped config  (mutate=0): verifies_presign=false verifies_final=true  subject=final
pre-D-18 config (mutate=1): verifies_presign=true  verifies_final=false subject=presign
```

This is D-07's one-time recorded mutation, discharged. SIGN-04 is now a
MEASUREMENT, not an argument from reading GoReleaser's source. **No
permanent ordering regression test was added** — D-07 declines one — and
the residual risk is accepted exactly as D-07 named it: a future refactor
could silently reorder the pipes with nothing firing.

The mutation diff, verbatim:

```
MUTATION-DIFF (the two-edit signs: block mutation, D-07):
324c324
< signs:
---
> binary_signs:
326d325
<     ids: [raw]
MUTATION-CONFIG-INVALID check: PASSED (structural validation completed before any Apple round-trip)
```

A separate, earlier single-arch narrowing (`254d253 < - codegraph-darwin-amd64`,
restricting notarize to arm64 only for this run) is not one of the two D-07
edits — it is the Apple-submission-budget narrowing the plan calls for,
applied independently.

`SIGN04-EVIDENCE`, mutation (`mutate=1`, arm64 only, per the
Apple-submission budget — the mechanism the mutation proves is per-pipe,
not per-architecture, so a second submission buys no extra evidence):

```
SIGN04-EVIDENCE schema=1 goos=darwin goarch=arm64 final_sha256=338b806608226297daebb979440b4dcefb40c3427a078419d508049d1b54c431 zip_content_sha256=unknown checksums_file_sha256=unknown cosign_verifies_presign=true cosign_verifies_final=false cosign_subject=presign verdict=pass mutate=1 config_sha256=ce90b0c03ceedd04da2bbf2e151c102ed75a1ac9b1cbab4f599cb4e844d2b99f
release:rehearse-notarize: PASS (mutate=1)   [exit 0]
```

`zip_content_sha256`/`checksums_file_sha256` are `unknown` under the
mutation by design: the mutated build-scoped `binary_signs:` shape signs the
raw build artifact directly and never runs the release-scoped archive and
checksum pipes the shipped configuration does, so there is no archived or
checksummed subject to compare for this run.

### Assumption A3 — verdict: SUPPORTED, with an honest scope limit

A3 asked whether switching cosign from `binary_signs:` to
`signs: {artifacts: binary}` is a complete fix, with no other
GoReleaser-internal pipe between the sign pipe and publish reintroducing a
mismatch. **Verdict: SUPPORTED for every pipe this rehearsal actually
exercised** — the archived (`.zip`), checksummed, and cosign-verified
subjects all equal the final notarized binary on both darwin arches under
the shipped configuration (the table above). **Not fully closed:** this
rehearsal runs `--skip=publish`, so it never reaches the `publish:` pipe or
any release-scoped pipe registered after `sign.Pipe{}` and before publish.
No such pipe is currently configured in this repository's
`.goreleaser.yaml` (per `02-RESEARCH.md`'s Open Question 4), so there is
nothing today that could reintroduce a mismatch — but that is a fact about
the current config, not a property this rehearsal measured past the sign
pipe. A3 is carried, in this narrower form, to plan 02-07 — the first run
that reaches `publish:` for real.

### The non-reproducible-signature finding (measured, not previously anticipated)

Across the baseline-vs-notarize comparisons performed this session, the
PRE-SIGN hashes were byte-identical run-to-run (arm64 `40fd5bf0...`, amd64
`856646e4...`), but the FINAL (post-notarize) hashes DIFFERED between the
first full run and the final run recorded above: arm64 `fb7ba58e...` (first
run) vs `016db76b...` (final run, recorded above); amd64 `ebc7b80d...`
(first run) vs `b85e2c0b...` (final run, recorded above).

**Cause:** `codesign -dvv`'s `Timestamp=` line shows an embedded,
Apple-issued trusted timestamp inside the code signature. The signature —
and therefore the final notarized binary — is NOT byte-reproducible across
separate signing operations, even though the pre-sign build that feeds it
is (per the `BASELINE-DETERMINISM-OK` measurement above).

**Consequence for any future darwin reproducibility leg:** it must compare
PRE-SIGN artifacts, never final signed/notarized ones, or it will report a
false regression on every single run — the signature timestamp alone
guarantees a hash difference between any two signing operations of
otherwise identical bytes.

### Reconciliation against code comments

No comment in `Taskfile.yml`'s `release:rehearse-notarize` target, and no
comment in `.goreleaser.yaml`'s `notarize:`/`signs:` blocks, was found to be
refuted by this run. The one candidate — `.goreleaser.yaml`'s existing
claim (lines 236-249) that the two notarize submissions run sequentially
and that timeout/pending both log-and-continue — is corroborated for its
sequential half (measured above) and left explicitly OPEN for its
pending/timeout half (also above), rather than "corrected" on no evidence.
Nothing in `Taskfile.yml` or `.goreleaser.yaml` was changed by this task.

## Assumption A5 — secret material in error paths

**Verdict: RESOLVED, no leak observed.** `release:rehearse-notarize` step 2
was re-run with a deliberately WRONG `MACOS_SIGN_PASSWORD`. Verbatim failure
output:

```
⨯ release failed after 1s     error=unable to decode p12 file: pkcs12: decryption password incorrect
task: Failed to run task "release:rehearse-notarize": exit status 1
```

The full failure output was scanned for: PEM headers (`BEGIN ... PRIVATE
KEY` / `BEGIN CERTIFICATE`) — zero found; base64 runs of 200+ characters
(the shape a P12 or API key blob would take) — zero found; the supplied
(wrong) password itself — zero occurrences. The failure occurred in **1
second**, before any Apple network contact — `quill`'s P12-decode step
fails locally, well before the notarize submission. No certificate or key
material appears anywhere in the error path exercised by this test.

Because the verdict is negative (no leak), **no masking requirement is
carried into plan 02-06's CI wiring** — this closes the open item in
T-02-11's mitigation plan rather than adding new scope to it. If a future
change to the signing library's error wrapping is made, this spot check
should be re-run rather than assumed to still hold.

### Cleanliness, confirmed after every run

`git diff --stat -- .goreleaser.yaml` was EMPTY after all four runs
performed this session, including the mutation — the generated-copy-only
design held. `git status` additionally showed only the maintainer's own
unrelated `.gitignore`/`.envrc` 1Password work, plus the `Taskfile.yml`
and-list fix below, since committed as `775d4df`.

### Defect found by this rehearsal, already fixed (`775d4df`)

The FIRST real run exited `201` AFTER emitting a complete set of passing
evidence lines, with no `::error::` printed. Root cause: `[ cond ] && VAR=x`
as the LAST statement of a loop body. go-task runs `cmds:` through its own
embedded shell (`mvdan.cc/sh`), not the system shell, and under `set -e`
that interpreter treats a FALSE and-list in final position as fatal. The
verdict-pass path is exactly the path where that condition is false — so
the target aborted ON SUCCESS, silently, while the failure path printed its
errors and also exited non-zero. The two outcomes were indistinguishable by
exit code.

Characterised empirically through `task` itself: and-list mid-block SAFE;
and-list at top level SAFE; and-list last-in-loop FATAL. A plain `sh` run of
the identical script exits `0` in all three positions, so the defect was
invisible outside `task`. This is the SECOND instance in this repository of
a bug that lives in the Taskfile wrapper and cannot be seen by running the
same script in a normal shell — the first ate a Go template
(`go list -f '{{.Version}}'`) and shipped v0.5.0 with zero assets.

Fix (`775d4df`): the fatal site plus five latent siblings converted to
`if`/`fi`, with the mechanism recorded in a `Taskfile.yml` comment.
Re-verified after the fix: `PASS (mutate=0) exit 0` and
`PASS (mutate=1) exit 0`.

### Acknowledgement (checkpoint step 6)

The maintainer explicitly read and accepted: the SLSA attestation leg of
ROADMAP criterion 3 is produced by a GitHub Action that only runs in CI, so
no local rehearsal can ever include the attested subject, and the
five-point chain therefore closes for the first time only on the
irreversible release in plan 02-07.
