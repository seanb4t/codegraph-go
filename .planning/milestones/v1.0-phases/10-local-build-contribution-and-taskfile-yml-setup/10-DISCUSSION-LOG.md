# Phase 10: Local Build Tooling & CONTRIBUTING - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-01
**Phase:** 10-local-build-contribution-and-taskfile-yml-setup
**Areas discussed:** Scope/fit, CI-vs-Taskfile authority, Tool bootstrap, Runners & caching, Test scope & missing toolchains, Bench & gate footguns

---

## Scope / phase fit

| Option | Description | Selected |
|--------|-------------|----------|
| Taskfile + CONTRIBUTING pointer | Criterion 2 recorded as already-satisfied; phase = Taskfile + short pointer. Stays in v1.0. | ✓ |
| Full phase as originally scoped | CONTRIBUTING gets an expansion pass (per-platform matrix, troubleshooting) alongside the Taskfile | |
| Defer whole phase to v1.1 | No CONTEXT.md; `/gsd-complete-milestone` closes v1.0 at 9 phases | |

**User's choice:** Taskfile + CONTRIBUTING pointer
**Notes:** Scout established `CONTRIBUTING.md` (156 lines) already documents the CGo
requirement, links `PARSER-DECISION.md`, and names both `zig` and `mingw-w64` — the
exact content ROADMAP criterion 2 asks for. It landed during OSS-readiness work,
outside any phase plan. Orchestrator also flagged `Requirements: TBD` (the only v1.0
phase with no requirement ID) and recommended minting one; user did not object.

---

## CI-vs-Taskfile authority

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror + Go drift guard | Taskfile as second definition; `taskfile_shape_test.go` reads both files off disk and goes red on divergence. Zero CI churn, no new CI supply-chain surface. *(Orchestrator's recommendation)* | |
| CI calls `task <target>` everywhere | One definition, drift structurally impossible. Cost: pinned go-task install in every workflow, new CI supply-chain surface, matrix/env logic doesn't move | ✓ |
| Hybrid — CI calls task for leaf commands only | Self-contained commands become single-definition; complex jobs mirrored | |
| Mirror only, best-effort | Header comment admits it may lag; nothing enforces agreement | |

**User's choice:** CI calls `task <target>` everywhere *(asked twice — same answer both times)*
**Notes:** User rejected the orchestrator's recommendation and challenged its
justification directly: *"I'm not sure that the convention of building in a haphazard
unplanned/organic way is a reason to keep going that way."* Orchestrator conceded the
point — "it is the existing convention" is not a valid justification, and the
guard-based option leaned on it. The named-CI-step guard has an independent *mechanism*
argument (GOLDEN-01: a suite silently stopped running, cost a Critical), but that
argument doesn't privilege CI step names as the home for the guarantee; a task
dependency graph is a better home.

---

## Granularity of the CI → task call

| Option | Description | Selected |
|--------|-------------|----------|
| Step-for-step: keep names, swap `run:` bodies | ci.yml keeps named steps and comments; each `run:` becomes one `task <target>`. Fine-grained Taskfile. *(Orchestrator's recommendation)* | ✓ |
| Coarse: one task per CI job | Smallest Taskfile, best contributor UX; collapses the named steps | |
| Coarse in CI, fine targets still available | CI calls coarse; fine targets exist locally | |

**User's choice:** Step-for-step **plus** a wrapper set of coarse tasks
**Notes:** User's exact framing: *"1 + a wrapper set of coarse tasks so engineers don't
have to call 1200 tasks."* A superset of the offered option. Orchestrator surfaced the
derived risk — the wrapper's completeness becomes the new silent-drop surface — which
led directly to the next question.

---

## Wrapper completeness

| Option | Description | Selected |
|--------|-------------|----------|
| CI calls fine targets; wrappers local-only | Wrapper omission degrades convenience but cannot weaken CI. Risk designed out, no guard needed. *(Recommended)* | ✓ |
| Set-equality guard in Go | `taskfile_shape_test.go` asserts `{test:*}` == `{wrapper deps}` — "no more AND no fewer" | |
| Both | Belt and braces | |

**User's choice:** CI calls fine-grained targets; wrappers are local-only

---

## Tool bootstrap

| Option | Description | Selected |
|--------|-------------|----------|
| Pinned `go install …@vX.Y.Z` | Matches ci.yml's existing actionlint/goreleaser pattern. No go.mod pollution. *(Orchestrator's recommendation)* | |
| Pinned `arduino/setup-task` action (SHA) | Fastest in CI (prebuilt binary); one more third-party action | |
| go.mod `tool` directive + `go tool` | Version pinned in a manifest; cost: transitive deps enter the module graph → SBOM, govulncheck, 6-target sweep | ✓ *(modified)* |

**User's choice:** Option 3 — *"but with a separate module? why not? if we can't do the
separate mod, then go install, but I'm pretty sure the 'tool' module makes sense"*
**Notes:** Orchestrator tested rather than assumed. Measured: root `go.mod` + go-task
tool directive = **237 net-new modules** (511 → 748, +46%), dominated by
`cloud.google.com/go/*`. Nested-module exclusion from a parent's `./...` confirmed by
experiment. Both `go -C <dir> tool` and `go tool -modfile=` verified working (task
v3.52.0). Cold build 9.9s / warm 0.68s. User's instinct confirmed empirically.

---

## Tool module location (first pass)

| Option | Description | Selected |
|--------|-------------|----------|
| `tools/toolchain/` | Nested under existing tools/; excluded from parent `./...`; tools/bench untouched *(Recommended)* | ✓ *(later superseded)* |
| `.tools/` | Dot-prefix means Go skips it even before the nested-module rule | |
| `build/tools/` | New top-level namespace separating build tooling from project source | |

**User's choice:** `tools/toolchain/`
**Notes:** Landmine found during scout and stated in the question: `tools/` itself is
unavailable — `tools/bench/{gencorpus,realcorpus,runner}` are part of the **main
module** and `bench.yml` runs `go run ./tools/bench/runner` from repo root three times.
A `go.mod` at `tools/` would silently excise all three. **Superseded** by the two-modfile
answer below, which uses repo-root modfiles and no directory at all.

---

## Tool set in the module (first pass — VOIDED)

| Option | Description | Selected |
|--------|-------------|----------|
| task + actionlint + goreleaser | One manifest, Dependabot covers all three *(Recommended)* | ✓ *(does not compile)* |
| All four — also replace govulncheck-action | Maximal single-source; changes a blocking security gate's mechanism | |
| Just `task` | Smallest blast radius | |

**User's choice:** task + actionlint + goreleaser
**Notes:** ⚠ **This answer was voided by measurement.** Orchestrator tested the three
tools in one module rather than inheriting holomush's conclusion:
`task` ✅ 3.52.0, `goreleaser` ✅ v2.17.1, `actionlint` ❌
`action_metadata.go:273:22: te.Errors[0].Error undefined (type string has no field or
method Error)` — a YAML-library MVS conflict. actionlint alone in its own modfile: 16
modules, builds clean. Re-asked below.

---

## Tool module layout (revised, after the compile failure)

| Option | Description | Selected |
|--------|-------------|----------|
| Two root modfiles, holomush's shape | `go.tool.mod` (task + goreleaser) + `go.tool-lint.mod` (actionlint), `GOWORK=off go tool -modfile=<f>`. No directory carve-out. *(Recommended)* | ✓ |
| One `go.tool.mod`; actionlint stays as `go install` | Single new manifest; actionlint's version stays in ci.yml | |
| Keep `tools/toolchain/`, two modfiles inside | Preserves the earlier location answer | |

**User's choice:** Two root modfiles, holomush's shape

---

## Runners & caching

| Option | Description | Selected |
|--------|-------------|----------|
| Namespace for test/lint; perf pinned to ubuntu-latest | Plus a `runner` field in baseline.json that CheckRegression compares *(Orchestrator's recommendation)* | |
| Namespace everywhere, re-bless perf first | Requires running the rebless job on Namespace and committing that baseline before the gate moves | ✓ |
| No Namespace | setup-go's built-in cache; avoids fork-PR quota consumption on a public repo | |
| Namespace for ci.yml only; release.yml untouched | Release pipeline keeps GitHub-hosted runners | |

**User's choice:** Namespace everywhere, re-bless perf first
**Notes:** User raised this mid-turn, citing `github.com/holomush/holomush`. Investigation
found the full production pattern: `namespace-profile-linux-amd64-{2x4,4x8}`,
`actions/setup-go` with `cache: false`, `namespacelabs/nscloud-cache-action` with
`cache: go`, and a local `install-task` composite action that builds from the Go module
proxy — explicitly because `arduino/setup-task@v2` and `taskfile.dev/install.sh` both
broke on CDN flake (incidents `holomush-06vz`, `holomush-ar31b`).

Orchestrator surfaced the sharp edge: `baseline.json` records only
`{"goos":"linux","goarch":"amd64"}` and `CheckRegression` compares exactly those two
fields — but `namespace-profile-linux-amd64-4x8` **is** linux/amd64, so the guard built
to prevent the fictitious-regression bug class is structurally blind to a runner-class
change. There is exactly one safe re-bless ordering (see CONTEXT D-09).

---

## Release-pipeline scope

| Option | Description | Selected |
|--------|-------------|----------|
| ci.yml + bench.yml only; release pipeline stays GitHub-hosted | Release pipeline was Phase-8-audited and Phase-9-verified end-to-end *(Orchestrator's recommendation)* | |
| Everything including release.yml | Re-opens the Phase-8/9 release audit inside a build-tooling phase | ✓ |
| ci.yml + bench.yml now, release pipeline as a follow-up phase | Same immediate scope, recorded as intended future work | |

**User's choice:** Everything including release.yml
**Notes:** Chosen with the caveat visible in the option text. Recorded, not re-litigated.
Orchestrator then verified constraints rather than handing the planner a preference that
might be impossible: the `provenance` job is
`uses: slsa-framework/…/generator_generic_slsa3.yml@v2.1.0` — a **reusable workflow**,
whose `runs-on` a caller cannot override. It **cannot** move. Separately, the
`macos-latest` leg builds both darwin arches natively and is load-bearing (Phase-9 D-05
keeps darwin off a zig cross-link).

---

## Test scope & missing toolchains

| Option | Description | Selected |
|--------|-------------|----------|
| Wrapper = host-only legs; cross targets separate + `preconditions:` | Satisfies criterion 3 on a clean checkout; nothing silently skips, nothing blocks a first run *(Recommended)* | ✓ |
| Wrapper covers everything, hard-fails without toolchain | Maximum CI fidelity; hostile to onboarding | |
| Wrapper covers everything; cross legs skip with a notice | Nobody blocked; green locally ≠ green in CI — the GOLDEN-01 silent-skip class | |

**User's choice:** Wrapper = host-only legs; cross-toolchain targets separate with preconditions
**Notes:** Scout also found plain `go vet ./...` is **not in CI today** — only two
`GOOS=windows` package-scoped vets. `CONTRIBUTING.md` tells contributors to run it but
nothing enforces it, so `task vet` represents new gate coverage.

---

## Bench & gate footguns

| Option | Description | Selected |
|--------|-------------|----------|
| No local rebless; `task check:cross` mirrors and replaces release-please's inline sweep | Matches CONTRIBUTING's existing prohibition *(Recommended)* | ✓ |
| Same, but release-please.yml keeps its inline sweep | Release-critical workflow avoids a task-bootstrap dependency; sweep exists twice | |
| Expose rebless behind a CI-only env guard | Makes the Namespace re-bless runnable through the same interface; loaded gun in a contributor-facing file | |

**User's choice:** No local rebless; sweep target mirrors and replaces the inline version
**Notes:** Tension worth flagging for planning — D-13 forbids a local rebless target
while D-09 requires a Namespace re-bless. Resolution: the re-bless runs via `bench.yml`,
not via a local target. D-15 makes a release-critical workflow depend on the task
bootstrap, which the planner must account for.

---

## Claude's Discretion

- Target naming: namespaced (`test:unit`, `lint:actions`) vs flat
- What bare `task` does — list targets vs run a check suite
- Whether `includes:` splits the Taskfile across files
- The `GO_TOOL` variable pattern for `-modfile` invocations
- Namespace profile sizing per job (`2x4` vs `4x8`)
- Whether Dependabot/Renovate config gains explicit ignores for the tool modfiles
- Shape of the `runner` field added to `baseline.json`

## Deferred Ideas

- Adding `goreleaser-check` to `main`'s required-status-check set (ruleset 20157557) —
  sequence after this phase, since the runner move changes its reported context
- Threat-register entries for the four `pull_request_target` workflows
- `Metrics.Repo` (corpus identity) never compared by `CheckRegression` — issue #16
- Migrating `govulncheck` off `golang/govulncheck-action` to a pinned tool directive
- Open issues #13, #14, #15, #17 — none are build-tooling
- Moving the SLSA provenance job to Namespace — impossible today; revisit only if the
  generator gains caller-configurable runners

## Corrections made during discussion

- **Dependabot coverage was oversold.** The orchestrator justified tool directives partly
  on automated-update visibility. holomush's `go.tool.mod` header documents the opposite:
  Renovate's `gomod` manager is deliberately scoped to root `go.mod` only, because its
  artifact step runs `go get -modfile=… -t ./...` and re-namespaces every main-module
  package under the tools module's identity. Dependabot's `gomod` manager likewise targets
  `go.mod`. The dependency-isolation argument stands; the automation argument does not.
- **"Convention" was not a valid justification** for preserving named CI steps — conceded
  to the user. The independent mechanism argument (GOLDEN-01) survives; the appeal to
  existing practice does not.
- **Phase 9's context said the 6-target sweep "should become an automated gate"** — that
  intent has since been implemented at `release-please.yml:46`. The orchestrator's initial
  reading of it as still-outstanding was stale.
