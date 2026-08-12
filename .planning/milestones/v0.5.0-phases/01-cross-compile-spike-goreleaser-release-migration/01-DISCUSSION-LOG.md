# Phase 1: Cross-Compile Spike & `goreleaser release` Migration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-08
**Phase:** 1-Cross-Compile Spike & `goreleaser release` Migration
**Areas discussed:** Spike venue & evidence, Getting a real release, Job topology & gates, GoReleaser config shape

---

## Area selection

| Option | Description | Selected |
|--------|-------------|----------|
| Spike venue & evidence | Where REL-05 runs, what executes the Linux binaries, throwaway vs canary, FAIL bar | ✓ |
| Getting a real release | D-06R vs REL-08's need for published assets and Phase 2's RED baseline | ✓ |
| Job topology & gates | One job vs SLSA split, cosign SAN survival, which shape tests bend | ✓ |
| GoReleaser config shape | `binary_signs:`, `sboms:`, second `archives:` entry, zip naming and contents | ✓ |

**User's choice:** All four.

---

## Spike venue & evidence

### Q1 — Which macOS host runs the spike and the migrated pipeline?

| Option | Description | Selected |
|--------|-------------|----------|
| Namespace tahoe | `namespace-profile-macos-6x14-tahoe` — already the darwin build runner and the canary's runner; attested-native Apple Silicon, 6x14; correcting the roadmap's `macos-latest` prose becomes a scoping fix. Risk: paid third-party runner, `mlugg/setup-zig` never run on it | ✓ |
| GitHub macos-latest | What the roadmap prose literally says; free, matches upstream examples; smaller box for a serialized 4-target CGo build; drops the maintainer-attested nativeness claim | |
| Spike on both | De-confounds a FAIL from a runner-class difference; costs a second leg of spike work | |

**User's choice:** Namespace tahoe.
**Notes:** Makes the ROADMAP/PROJECT.md `macos-latest` phrasing a documented correction rather than a contradiction.

### Q2 — What executes the cross-built Linux binaries?

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub ubuntu runners | `ubuntu-latest` + `ubuntu-24.04-arm`; real hardware both arches, free for public repos, no new paid profile | |
| Namespace linux profiles | Existing `linux-amd64-4x8` plus a new Namespace linux-arm64 profile; single provider, likely faster; adds an unused runner class and a cost line | ✓ |
| Docker + QEMU on the mac | One job, no artifact handoff; but arm64-under-QEMU is emulation — the same category of weaker proof the criterion rejects for "build exited 0" | |

**User's choice:** Namespace linux profiles.
**Notes:** Surfaced during scouting that the repo has **no** arm64 execution runner today — `ci.yml`'s arm64 work is a build-only, amd64-hosted, non-blocking reproducibility diff. Provisioning the profile is new infrastructure inside this phase.

### Q3 — Throwaway or standing infrastructure?

| Option | Description | Selected |
|--------|-------------|----------|
| Permanent canary | Dispatchable workflow mirroring `darwin-toolchain-canary.yml`; answers the repo's recurring "a check that ran once is not a gate" failure mode | ✓ |
| Throwaway + recorded evidence | Run once, capture run URL/sha256s/fixture output into SUMMARY and PROJECT.md; lighter, but nothing re-proves the property when zig or the image moves | |
| Throwaway first, promote if green | Cheapest path to the verdict, invest only on the load-bearing branch; costs a second pass over the same YAML | |

**User's choice:** Permanent canary.

### Q4 — Iteration budget before declaring FAIL?

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded, enumerated up front | Write the variation list before running (zig version, glibc triple floor, static-vs-dynamic `CGO_LDFLAGS`), exhaust it, then FAIL | ✓ |
| Strict — first attempt decides | Fastest to the Pro decision; risks buying Pro over a one-line triple | |
| Open-ended | Highest confidence in a FAIL verdict; unbounded cost on a blocking phase | |

**User's choice:** Bounded, enumerated up front.

---

## Getting a real release

### Q1 — What cuts the tag?

| Option | Description | Selected |
|--------|-------------|----------|
| `feat(release):` PR title | release-please computes v0.4.0 → v0.5.0; D-06R untouched; defensible as `feat:` because the release gains a new asset shape (REL-09) | ✓ |
| `fix(release):` patch bump | More conservative; understates what shipped and desyncs from the v0.5.0 milestone label | |
| Land as `ci:`, release separately | Honest commit semantics; Phase 1 can't prove REL-08 on its own schedule and Phase 2's baseline arrives on someone else's | |
| `Release-As:` footer | Keeps release-please as tagger but forces a version — reverses PROJECT.md's D-06R "no version is ever forced" | |

**User's choice:** `feat(release):` PR title.
**Notes:** Repo is squash-only with `squash_merge_commit_title=PR_TITLE`, so the PR title is what release-please parses — a failure mode this repo has already hit once.

### Q2 — Dry run before the first real release?

| Option | Description | Selected |
|--------|-------------|----------|
| Local snapshot on macOS | New Taskfile target running `goreleaser release --snapshot --skip=publish,sign`, mirroring `check:darwin-release-build`'s existing `--snapshot` precedent | ✓ |
| Full dry run to a scratch repo | Highest fidelity end-to-end; but cosign SAN and provenance anchor to the scratch repo, so the two claims that matter aren't the ones proven | |
| No dry run | Rely on `check:goreleaser` + shape tests; fastest, leaves first evidence and first risk in the same run | |

**User's choice:** Local snapshot on macOS.

### Q3 — Recovery posture?

| Option | Description | Selected |
|--------|-------------|----------|
| Patch forward | Never delete a published release or tag; consistent with D-06R and protects Phase 2's RED baseline; a bad release is briefly public | ✓ |
| Draft-first, promote manually | Strong safety valve; puts a human back in an automated release path and a draft isn't anonymously downloadable — the population Phase 2 cares about | |
| Delete and re-cut | Cleanest history; a human effectively re-creating a tag, and it destroys the RED baseline | |

**User's choice:** Patch forward.

### Q4 — Where does the self-upgrade proof live?

| Option | Description | Selected |
|--------|-------------|----------|
| Automated post-release job | Downloads the real prior binary for darwin/arm64 + linux/amd64, runs `codegraph upgrade`, asserts sha256 matches the attested subject; re-fires every release | ✓ |
| Manual maintainer runbook step | Much less YAML; a check that ran once during a decision, which the repo has repeatedly found is not a gate | |
| Both — manual now, automated after | Unblocks by hand, lands the job in the same phase | |

**User's choice:** Automated post-release job.

---

## Job topology & gates

### Q1 — Topology after the collapse?

| Option | Description | Selected |
|--------|-------------|----------|
| One release job + SLSA job | macOS job publishes and exports `hashes`; existing `generator_generic_slsa3.yml@v2.1.0` caller unchanged; `slsa-verifier verify-artifact` keeps working verbatim | |
| Truly one job, native attestation | `actions/attest-build-provenance` replaces the reusable workflow; changes the builder identity so the documented `slsa-verifier` command would no longer pass | ✓ |
| Keep a thin Linux handoff job | Subject list derived from published assets; extra job and download | |

**User's choice:** Truly one job, native attestation.
**Notes:** User's words — *"2 seems like the right thing, less 'custom' - I'm not really worried yet about changing"*. Followed up with the consequence: five documents plus `TestProvenanceJobUsesTaggedSLSAGenerator` plus REL-08's wording. Reassuring finding surfaced at the same time: `internal/upgrade/verify.go` has no SLSA/in-toto references at all, so `codegraph upgrade` is indifferent to the attestor.

### Q1b — Scope of the switch?

| Option | Description | Selected |
|--------|-------------|----------|
| Native, verify-command-gated | Go native; if `slsa-verifier` still verifies native output REL-08 survives verbatim, otherwise the doc/test/requirement churn enters scope explicitly | |
| Native unconditionally | Commit now and carry the churn regardless of research findings | ✓ |
| Keep the SLSA generator job | Nothing about verification, docs or REL-08 changes; costs one extra job | |

**User's choice:** Native unconditionally.

### Q2 — How is the cosign SAN proven to survive?

| Option | Description | Selected |
|--------|-------------|----------|
| Live verify + a new shape test | `cosign verify-blob` on a re-downloaded asset with the unchanged identity regexp, plus a test pinning `id-token: write` to the goreleaser job | ✓ |
| Live verification only | Less test surface; the property is then only true for releases someone verified by hand | |
| Shape test only | Cheap in PR CI; says nothing about whether the real Sigstore cert carried the expected SAN | |

**User's choice:** Live verify + a new shape test.

### Q3 — Checksums / attestation subject scope?

| Option | Description | Selected |
|--------|-------------|----------|
| Binaries + zips, sidecars excluded | 8 payloads; satisfies REL-06 criterion 2 for what users download; extends provenance to the zips Phases 2–3 consume | ✓ |
| Binaries only, as today | Zero change to provenance meaning; leaves the widest-audience assets with the weakest claims | |
| Everything, sidecars included | Maximally complete; circular (signing a signature) and inflates the subject list | |

**User's choice:** Binaries + zips, sidecars excluded.

### Q4 — Disposition of `TestDarwinLegsBuildNatively`?

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite to the new invariant | Assert darwin runner + zig-cross linux build ids; demonstrate RED by flipping to ubuntu | ✓ |
| Delete as obsolete | Least code; abandons the libresolv-DNS property the test exists to protect | |
| Leave it, special-case | Smallest diff; a test accepting two shapes can pass on the shape that isn't shipping | |

**User's choice:** Rewrite to the new invariant.

---

## GoReleaser config shape

### Q1 — How is the `.sigstore.json` sidecar name held?

| Option | Description | Selected |
|--------|-------------|----------|
| Static test vs the config | PR-time test computing `releaseAssetName()+".sigstore.json"` for all four platforms against `binary_signs.signature`; dynamic side already covered by the post-release upgrade job | ✓ |
| Post-release asset set-compare | Proves against reality; only fires at release time, so a bad template ships first | |
| Both | Belt and braces on the contract whose breakage bricks every upgrade; double the test surface | |

**User's choice:** Static test vs the config.

### Q2 — What is the `.zip` called?

| Option | Description | Selected |
|--------|-------------|----------|
| Same stem + `.zip` | `codegraph_<tag>_<goos>_<goarch>.zip`; two `archives:` entries keyed by `id`; the prefix-glob hazard leaves with the deleted shell steps | ✓ |
| Distinct token in the stem | Unambiguous to any prefix matcher; diverges from GoReleaser and cask conventions | |
| Homebrew-conventional name | Optimizes Phase 3; introduces a second version-string convention alongside the v-prefixed tag | |

**User's choice:** Same stem + `.zip`.

### Q3 — Zip contents?

| Option | Description | Selected |
|--------|-------------|----------|
| Binary + LICENSE + README | GoReleaser's conventional default; completions/man stay out because BREW-03/04 generate them at cask-build time | ✓ |
| Binary only | Smallest; keeps the Phase-2 byte-identity reasoning simplest; no license alongside the binary | |
| Binary + LICENSE + README + completions + man | One-stop download; contradicts BREW-03's generate-at-cask-build-time requirement | |

**User's choice:** Binary + LICENSE + README.

### Q4 — SBOM shape?

| Option | Description | Selected |
|--------|-------------|----------|
| Per-binary, preserving today's names | `<asset>.spdx.json` unchanged; DIST-03 carries forward; retains per-platform build-constraint precision | ✓ |
| Per-artifact, binaries and zips | Most complete; a zip's SBOM duplicates its binary's | |
| One release-wide SBOM | Fewest artifacts; loses per-platform precision and changes the asset set | |

**User's choice:** Per-binary, preserving today's names.

---

## Pending todo cross-reference

| Option | Description | Selected |
|--------|-------------|----------|
| Don't fold | Wire-oracle `toolslist-repeat` flake matched at 0.6 on the generic keywords "test"/"yml" only; an MCP concern unrelated to the release pipeline | ✓ |
| Fold it in | Schedule a fix alongside the pipeline migration | |

**User's choice:** Don't fold. Recorded as a reviewed-but-deferred todo in CONTEXT.md.

---

## Claude's Discretion

None claimed — every question received an explicit answer. Four items were routed to research
rather than to the maintainer because they are facts, not preferences: Namespace linux-arm64
profile availability; `mlugg/setup-zig` on the Namespace macOS image; whether GoReleaser's
`checksum:` can exclude sidecars and `binary_signs:` supports an exact signature template; and
whether `actions/attest-build-provenance` accepts a multi-subject list and whether
`slsa-verifier` verifies its output.

## Deferred Ideas

Offered as candidate additional gray areas at the close of discussion and declined:

- DIST-04 double-build determinism when Linux binaries are zig-crossed from a macOS host, and the
  fate of `check:reproducibility:arm64`.
- Where the zig version pin lives once zig runs on the macOS host for both Linux legs, and whether
  one pin governs CI, the canary and local builds.
- Whether `check:cross`, `check:darwin-toolchain` and `check:darwin-release-build` survive as-is,
  merge, or gain a linux-cross sibling.

Also noted from prior context, out of scope here:

- `GO-2026-5932` revisit-on-evidence (STATE.md).
- The 999.3 / 999.6 backlog bookkeeping call (STATE.md Blockers/Concerns).
