# Phase 2: Apple Signing & Notarization - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-09
**Phase:** 2-Apple Signing & Notarization
**Areas discussed:** Signing mechanism, Notarization scope & pipe ordering, Pre-tag rehearsal venue, Suite-against-notarized-binary gate, Entitlements, SIGN-04 proof method, Rehearsal shape, Todo cross-reference

---

## Signing & Notarization Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| GoReleaser native `notarize:` (quill) | Built-in `notarize.macos` block; quill already an indirect dep at `go.tool.mod:236`, so no new dependency and no shell glue. Cost: signing is a pure-Go reimplementation of `codesign`, so hardened-runtime/entitlement differences surface only at exec time | ✓ |
| Hand-rolled `codesign` + `notarytool` hooks | Apple's real tooling via post-build hooks on the macOS runner. Authoritative, but puts signing in shell glue inside the unrehearsable path and outside GoReleaser's artifact bookkeeping | |
| Native block, real-tool verification | Native signing, Apple-native verification | |
| Let research decide | Defer to the phase researcher | |

**User's choice:** GoReleaser native `notarize:` (quill)
**Notes:** Apple-native verification is not actually forfeited by this choice — SIGN-02 already mandates `spctl -a -vv -t exec` + `syspolicy_check distribution` as the oracle, so quill's own success report is never trusted. Recorded as D-02 in CONTEXT.md.

---

## Notarization Scope & Pipe Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Raw binaries; notarize first | Notarize the raw Mach-O before archive/checksum/sign/sbom, so `.zip`, checksums, cosign subject and SLSA subject all describe post-notarization bytes by construction | ✓ |
| Notarize both raw and `.zip` | Broader coverage, but the `.zip` is built from the raw binary — redundant work, doubled Apple round-trip | |
| Notarize the `.zip` only | Rejected direction — `codegraph upgrade` consumes the raw binary, which would then ship un-notarized | |

**User's choice:** Raw binaries; notarize first
**Notes:** Rated **one-way once published** in CONTEXT.md (D-04) — the ordering determines what cosign signed and SLSA attested for a given tag, and D-07 forbids deleting a release to correct it.

---

## Pre-Tag Rehearsal Venue

| Option | Description | Selected |
|--------|-------------|----------|
| `workflow_dispatch` rehearsal | Manually-dispatched, main-branch-only job running the full notarize pipe against real credentials, output discarded. Secrets never reachable from a PR trigger. Only registers after the workflow lands on main | |
| Local rehearsal on the maintainer's Mac | Guarded Taskfile target run locally before any tag push. Fastest loop, no CI secret exposure; does not exercise the CI environment | ✓ |
| Accept tag-push-only | No rehearsal; rely on D-07 patch-forward. The posture that produced an empty `v0.5.0` | |
| Dispatch rehearsal + local target | Both | |

**User's choice:** Local rehearsal on the maintainer's Mac
**Notes:** The accepted gap is recorded explicitly in CONTEXT.md D-08 — first CI execution remains genuinely first execution. The declined `workflow_dispatch` option is preserved in Deferred Ideas as the revisit path if the first notarized release fails for an environment-specific reason.

---

## Suite-Against-Notarized-Binary Gate (criterion 4)

| Option | Description | Selected |
|--------|-------------|----------|
| Post-release verify job | Extend the existing post-release-verify workflow to download the published notarized asset and run the suite against it. Reuses proven machinery; tests exactly the bytes a user gets; failure handled patch-forward | ✓ |
| Pre-publish gate in the release job | Blocks a bad release from publishing at all. Strongest guarantee, but lengthens the single unrehearsable path | |
| Both — smoke pre-publish, full post-release | Layered; two places to maintain | |

**User's choice:** Post-release verify job
**Notes:** CONTEXT.md D-11 records the two traps this inherits — the event-aware guard (`github.event.workflow_run` is null under `workflow_dispatch`, so a bare `conclusion` test skips silently and reports green) and zero-asset-release tolerance (`v0.5.0` is permanently present).

---

## Entitlements & Hardened Runtime

| Option | Description | Selected |
|--------|-------------|----------|
| Hardened runtime, zero entitlements | Grant nothing; CGo tree-sitter compiles C at build time, not runtime | |
| Hardened runtime, minimal entitlements if forced | Start at zero, add only on a measured exec failure, each addition recorded | |
| Let research determine the set | Researcher establishes quill's entitlement support and what a CGo Go binary requires under hardened runtime before the planner commits | ✓ |

**User's choice:** Let research determine the set
**Notes:** Captured in CONTEXT.md D-03 as an explicit open research question with a stated working hypothesis to be confirmed or refuted — deliberately not locked as a decision.

---

## SIGN-04 Proof Method

| Option | Description | Selected |
|--------|-------------|----------|
| One-time recorded mutation | Temporarily mis-order the pipe during execution, record divergent sha256 values as phase evidence, revert. Proves the check can fire; no permanent test surface | ✓ |
| Permanent test pinning the ordering | Standing regression guard, demonstrated RED first. Risk: this repo has been bitten by a test that pinned a broken template and resisted correction | |
| Both | Recorded mutation plus property-based guard | |

**User's choice:** One-time recorded mutation
**Notes:** The trade is stated plainly in CONTEXT.md D-07 — the proof is an observation, not a guard, so a future refactor could silently reorder the pipe without a test firing. Rated reversible; the guard can be added later.

---

## Rehearsal Target Shape

| Option | Description | Selected |
|--------|-------------|----------|
| Guarded maintainer-only Taskfile target | Hard-fails by name when the cert or App Store Connect key is absent, mirroring the `command -v cosign` precondition added after the 5-minute `expired_token` hang | ✓ |
| Guarded target plus credential-free dry mode | Adds a contributor-runnable mode stopping short of Apple submission. More surface to build and keep honest | |
| Ad-hoc local run, no committed target | Lowest build cost; nothing repeatable next release | |

**User's choice:** Guarded maintainer-only Taskfile target
**Notes:** CONTEXT.md D-09. The credential-free contributor mode is preserved in Deferred Ideas as surface without a current consumer.

---

## Todo Cross-Reference

| Option | Description | Selected |
|--------|-------------|----------|
| Fold neither | Both matched on generic keywords, not domain | ✓ |
| Fold the wire-oracle flake | MCP wire-protocol ordering flake (score 0.9) | |
| Fold the usage-skill todo | Agent tooling (score 0.6) | |

**User's choice:** Fold neither
**Notes:** Both recorded as reviewed-not-folded in CONTEXT.md `<deferred>` so future phases know they were considered. The 0.9 score on the wire-oracle todo came from keywords `ordering`/`test`/`github` plus area `mcp` — a false-positive match against a macOS distribution phase.

---

## Claude's Discretion

- Concrete CI secret shape (base64 P12 + password vs. keychain import; App Store Connect `issuer_id`/`key_id`/`key` naming and encoding) — follow GoReleaser's `notarize.macos` schema
- Taskfile target naming, consistent with existing `check:*` / `verify:*` / `release:*` conventions
- Format of the recorded SIGN-04 mutation evidence in phase artifacts
- Whether the `docs/RELEASE.md` reproduction commands form a new section or extend §1

## Deferred Ideas

- CI rehearsal of the notarize pipe via `workflow_dispatch` — declined here; revisit if the first notarized release fails for an environment-specific reason
- Permanent property-based regression test pinning the notarize pipe's position — additive later
- Credential-free contributor dry mode for the rehearsal target — no current consumer
- DIST-06, stapled offline-safe first launch via hand-rolled `pkgbuild`/`productbuild`/`productsign`/`xcrun stapler` — out of scope per REQUIREMENTS.md:57
