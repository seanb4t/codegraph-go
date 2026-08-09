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
