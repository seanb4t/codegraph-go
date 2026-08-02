# Phase 10: Local Build Tooling & CONTRIBUTING - Research

**Researched:** 2026-08-01
**Domain:** Go build-tooling / CI orchestration (go-task, isolated `go tool` modfiles, GitHub Actions rewiring)
**Confidence:** HIGH (mechanics verified live against this repo's actual Go 1.26.5 toolchain and this repo's actual workflow files; MEDIUM on Namespace-specific product details, sourced from web search only)

<user_constraints>
## User Constraints (from CONTEXT.md)

CONTEXT.md's 16 decisions (D-00..D-16) are **LOCKED** — this research does not relitigate them.
Copied verbatim below for the planner.

### Locked Decisions

#### D-00 — ROADMAP criterion 2 is already satisfied; record, do not redo

`CONTRIBUTING.md` already exists (156 lines) and its `## Building` section already
documents the CGo requirement, links `PARSER-DECISION.md`, and names **both**
`zig` (cross-builds) and `mingw-w64` (Windows vet) — the exact content criterion 2
asks for. It landed during the OSS-readiness work, outside any phase plan.

**Do not rewrite it.** The phase's CONTRIBUTING work is a pointer to the new task
targets. Record criterion 2 as pre-satisfied in `10-VERIFICATION.md` rather than
manufacturing work to "close" it.

`[user] Scope → Selected: Taskfile + CONTRIBUTING pointer (recommended)`

#### Gray area 1 — Who owns the command definitions

- **D-01:** **CI calls `task <target>` everywhere.** One definition; drift is
  structurally impossible rather than guarded against.
  Explicitly **rejected**: mirroring `ci.yml` into a Taskfile with a Go drift
  guard (the orchestrator's recommendation). The user's reasoning, recorded
  because it is the load-bearing bit: *"I'm not sure that the convention of
  building in a haphazard unplanned/organic way is a reason to keep going that
  way."* That is correct — "it is the existing convention" was not a valid
  justification, and the guard-based option was justified largely on convention.
  — **Reversibility:** costly — undoing means re-inlining commands across
  ~10 workflow files and deleting the tool modfiles; no published contract
  breaks, but every workflow is touched again.
  `[user] Authority → Selected: CI calls task everywhere`

- **D-02:** **CI calls the FINE-GRAINED targets; coarse wrappers are
  contributor-only.** `ci.yml` keeps every named step and every explanatory
  comment; only each step's `run:` body becomes a single `task <target>` call.
  Wrappers (`task test`) exist so humans do not type six commands.

  **Why this specific split matters** — `test/integration/main_test.go:1-5` says
  verbatim: *"a normal Go package (never `testdata/` — GOLDEN-01 cost Phase 2 a
  Critical when a suite silently didn't run) so `go test ./...` reaches it, PLUS
  an explicit named CI step (`.github/workflows/ci.yml`) so a future refactor of
  the filtered `go list ./...` line can never silently drop it either."*
  Collapsing the six named test steps into one coarse `task test` would discard
  that second guarantee.

  Because CI calls the fine targets directly, **a wrapper that forgets a target
  degrades a contributor's convenience but cannot weaken CI coverage.** The
  silent-drop risk is designed out rather than guarded — no set-equality test is
  needed.
  `[user] Granularity → Selected: step-for-step + coarse wrappers for humans`
  `[user] Wrapper → Selected: CI calls fine targets; wrappers local-only (recommended)`

#### Gray area 2 — Tool bootstrap (MEASURED, not argued)

- **D-03:** **Two isolated tool modfiles at repo root, invoked via
  `GOWORK=off go tool -modfile=<f> <name>`:**
  - `go.tool.mod` — `task` + `goreleaser`
  - `go.tool-lint.mod` — `actionlint`

  **Why isolated at all — measured:** adding `tool github.com/go-task/task/v3/cmd/task`
  to the root `go.mod` adds **237 net-new modules** (main module 511 → 748, +46%),
  overwhelmingly `cloud.google.com/go/*` pulled in by go-task's remote-Taskfile /
  GCS support. Those would land in the Syft SBOM, fall in scope for the **blocking**
  `govulncheck` gate (DIST-03), and have to resolve on all six `GOOS/GOARCH` legs of
  the pre-tag sweep — against a project whose stated constraint is "minimal, audited
  dependencies."

  **Why TWO modfiles and not one — measured, this is not a style choice:**
  ```
  task        ✅ 3.52.0
  goreleaser  ✅ v2.17.1
  actionlint  ❌ action_metadata.go:273:22:
                 te.Errors[0].Error undefined (type string has no field or method Error)
  ```
  MVS resolves one version per module across the graph, so co-locating merges
  unrelated tools' YAML constraints; actionlint's expected YAML API loses the bid
  to what task/goreleaser drag in. **A single shared tool module does not compile.**
  Verified independently: actionlint alone in its own modfile builds fine at
  **16 modules**. This reproduces the split documented in
  `holomush/holomush`'s `go.tool.mod` header for a different tool set.

  **`GOWORK=off` is required** — `go.work` enables workspace mode, which is
  incompatible with `-modfile`.
  — **Reversibility:** reversible — delete the modfiles and restore
  `go install …@vX` lines; nothing published depends on them.
  `[user] Bootstrap → Selected: go.mod tool directives, isolated module (option 3 + "separate module? why not?")`
  `[user] Tool mods → Selected: two root modfiles, holomush's shape (recommended)`

- **D-04:** **Build tools from the Go module proxy, NOT from GitHub Releases.**
  Rejected: `arduino/setup-task` and the `taskfile.dev/install.sh` shim.
  Evidence, not preference — `holomush/holomush`'s `install-task` composite action
  records that both broke on **CDN flake** (incidents `holomush-06vz`,
  `holomush-ar31b`). The module-proxy path is covered by the Go module/build cache.
  Cold build measured at **9.9s** wall (fast M-series; expect 2–4× on a CI runner),
  warm **0.68s**.
  `[user] Bootstrap → module proxy path implied by tool-directive selection`

- **D-05:** ⚠ **The tool modfiles will NOT be covered by automated dependency
  updates — bumps are manual.** This CORRECTS a claim made during discussion in
  favour of tool directives. `holomush`'s modfile header documents that Renovate's
  `gomod` manager is deliberately scoped to root `go.mod` only, because its
  artifact step runs `go get -modfile=… -t ./...` from the repo root and the
  `./...` sweep re-namespaces every main-module package under the tools module's
  identity. Dependabot's `gomod` manager likewise targets `go.mod`.
  **The dependency-isolation argument for D-03 stands; the automation argument does
  not.** Update posture is unchanged from today's manual `@v1.7.12` / `@v2.17.1`
  pins. Do not write docs claiming otherwise.

#### Gray area 3 — Runners and caching

- **D-06:** **Move to Namespace runners + `namespacelabs/nscloud-cache-action`
  across `ci.yml`, `bench.yml`, AND `release.yml`.** Pattern (from `holomush/holomush`
  `ci.yaml`): `runs-on: namespace-profile-linux-amd64-{2x4,4x8}`,
  `actions/setup-go` with **`cache: false`**, then
  `namespacelabs/nscloud-cache-action@<full-sha>` with `cache: go`.
  All third-party actions SHA-pinned per repo convention.

  **The orchestrator recommended limiting this to `ci.yml` + `bench.yml`;
  the user chose full scope having read that caveat.** Recorded, not re-litigated.
  — **Reversibility:** costly — the release pipeline was Phase-8-audited and
  Phase-9-verified end-to-end against a real published artifact; changing its
  runner infrastructure re-opens that audit surface and needs new threat-register
  entries.
  `[user] Runners → Selected: Namespace everywhere, re-bless perf first`
  `[user] Rel scope → Selected: everything including release.yml`

- **D-07:** 🔒 **HARD EXCEPTION — the `provenance` job CANNOT move.**
  `.github/workflows/release.yml:339` is
  `uses: slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0`.
  A **reusable workflow declares its own `runs-on`; a caller cannot override it.**
  This is a structural fact, not a trade-off. "Namespace everywhere" means
  everywhere it is *possible*, and this job is the documented exception.

- **D-08:** ⚠ **`release.yml`'s `macos-latest` leg is load-bearing and must stay a
  real macOS runner.** The matrix is 4× `ubuntu-latest` (linux amd64 native; linux
  arm64 + windows amd64/arm64 via zig) + 1× `macos-latest` building **both** darwin
  arches natively. Phase 9's D-05 protects this specifically to keep darwin off a
  zig cross-link (libresolv/DNS risk). If Namespace macOS is used, it must be a
  native darwin runner — never substituted with a cross-build.

- **D-09:** 🔢 **The perf re-bless has exactly ONE safe ordering.** `baseline.json`
  records only `{"goos":"linux","goarch":"amd64"}` and `CheckRegression` compares
  exactly those two fields — but `namespace-profile-linux-amd64-4x8` **is** linux/amd64.
  **The platform guard built to prevent the fictitious-regression bug class is
  structurally blind to a runner-class change.**

  Required order:
  1. Move `bench.yml`'s rebless job to Namespace and record a Namespace baseline
     (human-reviewed, per D-05-of-Phase-9: `-rebless` is the sole writer).
  2. Commit that baseline.
  3. Only then move `perf-regression` to Namespace.

  Moving the gate first compares Namespace throughput to an ubuntu-latest
  baseline — deliberately reproducing the failure that cost three rounds of triage.
  Moving rebless first without committing leaves the still-on-ubuntu gate failing.
  **Strongly recommended alongside:** add a `runner`/profile field to
  `baseline.json` and compare it, closing the blind spot. (Related but distinct
  from open issue #16, `Metrics.Repo` never compared.)
  — **Reversibility:** one-way in effect — a wrong-platform baseline committed to
  `baseline.json` produces a stable, entirely fictitious regression that reads as
  real and has historically survived multiple triage rounds.

#### Gray area 4 — Test scope and missing toolchains

- **D-10:** **The contributor wrapper `task test` covers host-only legs:** unit
  (materialized `go list`), golden (`./testdata/golden/...`), integration
  (`./test/integration/...`), daemon-isolated (`-count=1`), and race
  (`-race -count=1 -p 1`). All run on a clean checkout with only a C toolchain —
  satisfying ROADMAP criterion 3 for the default path.

- **D-11:** **Cross-toolchain work lives in SEPARATE targets carrying go-task
  `preconditions:` with actionable `msg:`** (e.g. `vet:windows` → mingw-w64,
  `build:cross` → zig). **Explicitly rejected: `status:`-based skipping.** A green
  local run must mean the same thing as a green CI run; a silent skip is the
  GOLDEN-01 failure class that cost Phase 2 a Critical. Nothing silently skips;
  nothing blocks a contributor's first command.
  `[user] Test scope → Selected: wrapper = host-only legs; cross targets separate + preconditions (recommended)`

- **D-12:** **`go vet ./...` becomes a real target and therefore a real CI gate.**
  Confirmed by search: plain `go vet ./...` is **not in CI today** — only two
  `GOOS=windows` package-scoped vets (`internal/graphstore`, `internal/daemon`).
  `CONTRIBUTING.md` already tells contributors to run it, but nothing enforces it.
  This is **new coverage**, not a rewiring — flag it as such; expect it to surface
  pre-existing findings on first run.

#### Gray area 5 — Dangerous and release-critical targets

- **D-13:** **No `-rebless` task target exists.** It stays reachable only through
  `bench.yml`. This matches `CONTRIBUTING.md`'s existing prohibition, which cites
  the wrong-platform baseline that produced a stable, fictitious 10.6% regression.
  Note the tension with D-09 — the Namespace re-bless is run **via the workflow**,
  not via a local target.
  `[user] Footguns → Selected: no local rebless (recommended)`

- **D-14:** **`task bench:regression` exists and must surface the platform guard's
  refusal legibly** — the gate is deliberately unrunnable on developer macOS, and
  the target should explain that rather than emit a bare non-zero exit.

- **D-15:** **`task check:cross` mirrors the 6-target `go list -mod=readonly`
  sweep, and REPLACES `release-please.yml`'s inline bash** (currently
  `.github/workflows/release-please.yml:46-51`) so the sweep has one definition.
  Note: this makes a release-critical workflow depend on the task bootstrap —
  the planner must ensure the tool build is present and cached in that job.
  — **Reversibility:** costly — `release-please.yml` gates tag creation; a
  bootstrap failure there blocks releases entirely.

#### Gray area 6 — Requirement ID

- **D-16:** **Mint a requirement ID for this phase during planning.** ROADMAP says
  `Requirements: TBD`; this is the only v1.0 phase with no requirement, and
  `nyquist_validation: true` + `security_enforcement: true` mean the downstream
  gates have nothing to key off. Suggested `DEV-01` covering
  "every CI-enforced command is invocable locally through one entry point, with
  exactly one definition." Add to `.planning/REQUIREMENTS.md` and its ownership
  table. **Confirm the ID/wording with the maintainer before writing** — the
  orchestrator flagged this as a recommendation, and the user did not object, but
  did not explicitly ratify the specific ID either.

### Claude's Discretion

- Target naming: namespaced (`test:unit`, `lint:actions`) vs flat. Namespaced is
  implied by D-02's fine granularity, but the exact names are open.
- What bare `task` does — list targets vs run a check suite. `go-task` defaults to
  listing when a `default` task is absent.
- Whether `includes:` is used to split the Taskfile into multiple files, or it stays
  one file.
- The `GO_TOOL` variable pattern for the `-modfile` invocations (holomush uses one;
  it keeps `GOWORK=off go tool -modfile=…` from being repeated per target).
- Namespace profile sizing per job (`2x4` vs `4x8`) — holomush uses `2x4` for
  lint/fast jobs and `4x8` for test jobs.
- Whether Dependabot/Renovate config gains explicit ignore entries for the tool
  modfiles, or simply stays silent about them (per D-05 they are unmanaged either way).
- Whether the `runner` field added to `baseline.json` (D-09) is a free-form string,
  the `runs-on` label, or a structured profile record.

### Deferred Ideas (OUT OF SCOPE)

- **Adding `goreleaser-check` to `main`'s required-status-check set** (ruleset
  20157557, currently 6 checks). Carried over from Phase 9's STATE notes. Should be
  done only after the job has reported at least once, or pending PRs block on a
  never-reported check. Not a Phase 10 action — but the runner move (D-06) will
  change its reported context, so sequence it after this phase.
- **Threat-register entries for the four `pull_request_target` workflows** — they
  landed after Phase 9's register was authored (advisory from that phase's security
  audit). Adjacent, not this phase.
- **`Metrics.Repo` (corpus identity) never compared by `CheckRegression`** — open
  issue #16. D-09 adds a *runner* field; the *repo* field remains open.
- **Migrating `govulncheck` off `golang/govulncheck-action`** to a pinned tool
  directive — would give version pinning, but changes the mechanism of a blocking
  security gate and needs a non-vacuous proof it still fires. Own change.
- **Open issues #13, #14, #15, #17** — daemon `getppid` race, provenance-over-checksums
  wording, `PRFILES_EOF` heredoc over fork-controlled paths, watchdog test under load.
  None are build-tooling.
- **Moving the SLSA provenance job to Namespace** — impossible today (D-07); revisit
  only if the generator gains caller-configurable runners.
</user_constraints>

<phase_requirements>
## Phase Requirements

ROADMAP.md currently lists `Requirements: TBD` for Phase 10 — the **only** v1.0
phase with no mapped requirement ID. CONTEXT.md D-16 proposes minting one during
planning (suggested `DEV-01`), explicitly **pending maintainer ratification**,
which per the phase brief is happening in parallel with this research run. This
research treats the ID as an **open input** — the table below uses the suggested
`DEV-01` wording as a placeholder so the planner has something concrete to key
verification off, but the planner MUST confirm the final ID/wording (or accept
D-16's suggestion) before writing it into `.planning/REQUIREMENTS.md`.

| ID (tentative) | Description | Research Support |
|----------------|-------------|-------------------|
| `DEV-01` (pending ratification, D-16) | Every CI-enforced command is invocable locally through one entry point (`task <target>`), with exactly one definition (no duplicate command text between `Taskfile.yml` and any `.github/workflows/*.yml` `run:` body) | Architecture Patterns (GO_TOOL var + isolated modfiles), Validation Architecture (property-based drift guard design + non-vacuity protocol), Common Pitfalls (required-status-check name preservation, `platforms:` silent-skip trap) |

ROADMAP's three stated success criteria for Phase 10 (Taskfile wraps the full
command set; `CONTRIBUTING.md` documents CGo prerequisites — pre-satisfied per
D-00; a clean checkout builds/tests/lints via task targets alone) map onto this
one requirement's three observable facets. If the maintainer prefers splitting
`DEV-01` into finer-grained requirement IDs (e.g. one for the Taskfile itself,
one for the CI single-definition property), that split does not change any
finding in this document — only the traceability table.
</phase_requirements>

## Summary

Phase 10 adds a `Taskfile.yml`-based single entry point for every build/test/lint/
release-check command in this repo, moves `.github/workflows/{ci,bench,release}.yml`
onto Namespace-hosted runners with `nscloud-cache-action`, and rewires those
workflows' `run:` bodies (plus `release-please.yml`'s inline pre-tag sweep) to call
`task <target>` instead of inlined shell. `github.com/holomush/holomush` — a public
sibling repo by this project's own maintainer — is already running this exact
pattern in production, and this research confirms its mechanics work unmodified
against this repo's live Go 1.26.5 toolchain: a root-level `go.tool.mod` sibling
file has **zero effect** on `go list ./...` (verified: package count identical
before/after adding one), `GOWORK=off go tool -modfile=go.tool.mod <name>` fails
loud (exit 1, not silent) when the modfile is absent, and the `go.tool.sum`
companion-file naming (not `go.tool.mod.sum`) matches Go's own `-modfile`
convention.

Two structural facts do the heavy lifting for the supply-chain question this
phase's tool-bootstrap design raises (D-03's "237 net-new modules" concern):
`govulncheck` (via `golang/govulncheck-action`) analyzes the build graph reachable
from **the `go.mod` its `go-version-file` input points at** — it has no reason to
ever open `go.tool.mod` unless explicitly pointed there — and the release
pipeline's per-binary SBOM step (`syft <binary>`) inspects the **compiled
artifact's embedded module info**, not the source tree, so tool-module code that
is never imported by `./cmd/codegraph` cannot appear in it even in principle.
Isolation is therefore not just a style preference; it is the mechanism that keeps
both gates honest.

Three concrete, previously-undocumented findings surfaced during this research
that the planner should treat as load-bearing:

1. **A version mismatch already exists between `ci.yml`'s `goreleaser-check` job
   (`v2.17.1`) and `release.yml`'s build matrix (`GORELEASER_VERSION: v2.17.0`)** —
   pre-existing, not introduced by this phase, but the new `go.tool.mod` pin will
   force a choice between them (recommend `v2.17.1`, matching the job this phase
   is replacing, and filing the `v2.17.0` discrepancy as a separate follow-up, not
   silently "fixing" a release-pipeline version choice that was Phase-9-audited).
2. **`release-please.yml` is NOT in D-06's Namespace-runner migration scope**
   (only `ci.yml`/`bench.yml`/`release.yml` are named) **but its `pretag-gate` job
   DOES need the `install-task` composite action wired in** once D-15 replaces its
   inline sweep with `task check:cross` — a bootstrap dependency the runner-class
   decision doesn't remove.
3. **go-task's `platforms:` field silently skips a task on a non-matching host** —
   exactly the failure class D-11 rejects by name (`status:`-based skipping). It
   must not be used to gate the windows-vet/cross-build targets; `preconditions:`
   with an actionable `msg:` is the only mechanism that fails loud, and it is a
   distinct field from `platforms:`.

**Primary recommendation:** Copy `holomush/holomush`'s `Taskfile.yaml` /
`go.tool.mod` / `.github/actions/install-task` shape near-verbatim (its own header
comments already document the CDN-flake and MVS-conflict rationale this repo
independently re-derived and measured), scope the target set to exactly what
`ci.yml`/`bench.yml`/`release-please.yml`/`release.yml` currently invoke (do not
import holomush's unrelated `web:*`/`proto:*`/`docker:*` tasks), and gate the
`goreleaser` version pin decision and the `release.yml` macOS-runner question as
explicit open items for the plan rather than silently picking one.

## Project Constraints (from CLAUDE.md)

`.claude/CLAUDE.md`'s Technology Stack section is authoritative for this repo and
directly constrains this phase:

- **"Minimal, audited dependencies"** (top-level Constraints) — the 237 net-new
  modules a naive `go.mod tool` directive would add is exactly the class of
  dependency-bloat this constraint exists to prevent. D-03's isolated-modfile
  design is the compliant implementation; a shared/root-`go.mod` tool directive
  would violate this constraint even though CONTEXT.md already rejected it on
  independent (SBOM/govulncheck-scope) grounds.
- **"Signed + attested + reproducible + SBOM'd releases"** — `goreleaser/goreleaser`
  and `anchore/syft` are the CLAUDE.md-recommended tools for exactly the jobs this
  phase's `task check:goreleaser`-equivalent target and the release SBOM step
  already use; no new tool is being introduced for these roles, only a new
  invocation path.
- **`golang.org/x/vuln/cmd/govulncheck`** is CLAUDE.md's named vuln-scanning tool
  ("call-graph-aware... lower noise than naive SCA tools") — this phase does not
  touch its invocation mechanism (deferred per CONTEXT.md's Deferred Ideas), only
  confirms via research that the isolated tool modules stay out of its scope.
- **`goreleaser/goreleaser` version**: CLAUDE.md's Installation table does not pin
  an exact version; `release.yml`'s `GORELEASER_VERSION: v2.17.0` and `ci.yml`'s
  `goreleaser@v2.17.1` are the two competing live pins this phase must reconcile
  (see Common Pitfalls).
- **No CLAUDE.md directive blocks Namespace runners, go-task, or isolated tool
  modfiles** — none of D-01 through D-16 contradicts any Technology Stack
  recommendation; this phase is purely additive tooling infrastructure.

## Architectural Responsibility Map

This is a build-tooling phase, not an application-layer phase — the standard
browser/API/DB tier table does not apply. The tiers below are this domain's
equivalent: where does a given command's *canonical definition* live, and where
does it *execute*.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|-----------------|-----------|
| Command definition (what `go build`/`go test`/`goreleaser check`/etc. actually run) | `Taskfile.yml` (repo root) | — | D-01: single source of truth; CI and humans both call into this file, never redefine a command elsewhere |
| Command invocation from automation | CI Runner (`.github/workflows/*.yml`) | — | D-02: fine-grained named steps stay in the workflow YAML (job names, comments, `if:`/`env:` context), but each step's `run:` body shrinks to `task <target>` |
| Command invocation from a human | Local Dev Environment (contributor's shell) | Taskfile.yml | D-02's coarse wrappers (`task test`, `task lint`) exist only here; CI never calls them |
| Tool version resolution (`task`, `goreleaser`, `actionlint` binaries themselves) | Tool Bootstrap Module (`go.tool.mod` / `go.tool-lint.mod`) | Go module proxy | D-03/D-04: isolated from the main module graph, fetched from the module proxy (not GitHub Releases), built on demand via `go tool -modfile=` |
| Release artifact build/sign/SBOM/publish | Release Pipeline (`release.yml` + `.goreleaser.yaml`) | Tool Bootstrap Module | Unchanged surface (goreleaser still does the work); only its invocation site (`task`-wrapped `goreleaser build`) and runner class (Namespace, except the D-07 SLSA exception) move |
| Pre-tag cross-platform sanity gate | `release-please.yml` (`pretag-gate` job) | Tool Bootstrap Module | D-15: inline bash sweep replaced by `task check:cross`; this job newly depends on the tool bootstrap despite NOT being in D-06's runner-migration scope |
| Perf baseline authority | `bench.yml` (`rebless` job, CI-only) | tools/bench/baseline.json (committed artifact) | D-13: deliberately NOT reachable from a local task target; D-09 governs the one safe re-bless ordering |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/go-task/task/v3` (cmd: `task`) | v3.52.0 `[VERIFIED: proxy.golang.org — go list -m -versions, 2026-08-01]` | Task runner / Taskfile executor | De facto standard cross-platform Make replacement for Go projects; already proven in production for this exact isolated-modfile pattern by `holomush/holomush`, the same maintainer's sibling repo |
| `github.com/goreleaser/goreleaser/v2` (cmd: `goreleaser`) | v2.17.1 `[VERIFIED: proxy.golang.org — go list -m -versions shows v2.18.0-*-nightly as the newest, v2.17.1 is the latest stable at time of research]` | Release build/config validation (`goreleaser check`, wraps `.goreleaser.yaml`) | Already the release pipeline's tool (`release.yml`); this phase only adds a `task`-wrapped invocation path for the existing `goreleaser check` CI job — **already pinned at this exact version in `ci.yml`'s current `goreleaser-check` job** (`go install github.com/goreleaser/goreleaser/v2@v2.17.1`, `.github/workflows/ci.yml:341`) |
| `github.com/rhysd/actionlint` (cmd: `actionlint`) | v1.7.12 `[VERIFIED: proxy.golang.org — go list -m -versions, matches this repo's live `ci.yml:322` pin exactly, no update available]` | GitHub Actions workflow static analysis | Already pinned and used in `ci.yml`'s `actionlint` job; this phase moves its install mechanism from `go install @pin` to the isolated tool modfile, same version |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `namespacelabs/nscloud-cache-action` | pin to commit `c5f8dab7560444c4bf8dbc64f1b203431873c547` — resolves to tag `v1.6.1` `[VERIFIED: GitHub API — /repos/namespacelabs/nscloud-cache-action/commits/{sha} and /tags, 2026-08-01]` | Go module/build cache on Namespace Cache Volumes, used alongside `actions/setup-go` with `cache: false` | Every job moved to a `namespace-profile-*` runner under D-06 (`ci.yml`, `bench.yml`, `release.yml`'s `build` matrix and `assemble` job) |

**Note on the SHA/tag comment convention:** `holomush/holomush`'s own workflow
pins this action with the trailing comment `# v1`, but this repo's existing
convention (seen throughout `ci.yml`/`release.yml`) always resolves to the
**fully-qualified** tag (e.g. `# v6.0.3`, `# v1.1.0`), never an abbreviated major
version. Use `# v1.6.1`, not `# v1`, to match this repo's own pinning discipline
— do not copy holomush's comment verbatim.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Isolated `go.tool.mod`/`go.tool-lint.mod` | Root `go.mod` `tool` directives | Rejected by measurement (D-03): 237 net-new modules, blocking-govulncheck scope creep, SBOM pollution |
| Module-proxy tool builds (`go tool -modfile=`) | `arduino/setup-task` / `taskfile.dev/install.sh` | Rejected by evidence (D-04): both broke on CDN flake in `holomush/holomush` production (incidents `holomush-06vz`, `holomush-ar31b`) |
| go-task `preconditions:` with `msg:` | go-task `status:` | Rejected (D-11): `status:` silently skips instead of failing — the exact GOLDEN-01 failure class this repo has a documented Critical-severity history with |
| go-task `preconditions:` for cross-toolchain gating | go-task `platforms:` | **New finding, this research:** `platforms:` silently skips a task on a non-matching host (confirmed via Context7 docs) — same silent-skip failure class D-11 already rejected for `status:`, but a *different* field a planner could reach for by habit. Do not use it for `vet:windows`/`build:cross`-style targets. |

**Installation:**
```bash
# One-time: create the isolated tool modules (repo root)
GOWORK=off go mod init github.com/seanb4t/codegraph-go/tools < /dev/null  # or hand-write per holomush's header-comment convention
GOWORK=off go get -tool -modfile=go.tool.mod github.com/go-task/task/v3/cmd/task@v3.52.0
GOWORK=off go get -tool -modfile=go.tool.mod github.com/goreleaser/goreleaser/v2@v2.17.1
GOWORK=off go get -tool -modfile=go.tool-lint.mod github.com/rhysd/actionlint/cmd/actionlint@v1.7.12

# Run a tool (what the Taskfile's GO_TOOL var wraps)
GOWORK=off go tool -modfile=go.tool.mod task --version
GOWORK=off go tool -modfile=go.tool-lint.mod actionlint --version
```

**Version verification:** confirmed live against this repo's Go 1.26.5 toolchain
via `GOPROXY=https://proxy.golang.org go list -m -versions <module>` for all
three tools (2026-08-01) — see table above. `go.mod`'s `require` block for each
tool module (the 237/16-module counts) is populated automatically by `go get -tool`
and does not need manual authoring.

## Package Legitimacy Audit

> The `package-legitimacy check` seam only supports `npm|pypi|crates` ecosystems —
> it errored on `--ecosystem go`. This audit is therefore a manual equivalent:
> Go module proxy existence/version-history check (an authoritative source,
> `proxy.golang.org`) plus GitHub repository reputation for each tool.

| Package | Registry | Age / Reputation | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-------------------|-----------|--------------|---------|-------------|
| `github.com/go-task/task/v3` | Go module proxy | Established, org-maintained (`go-task` GitHub org), long release history through v3.52.0 | N/A (Go modules have no download counter) | `github.com/go-task/task` | OK | Approved — already the reference implementation's (holomush) choice, and Context7's resolver rated it "High" source reputation |
| `github.com/goreleaser/goreleaser/v2` | Go module proxy | Established, already in production use in this exact repo's `release.yml` | N/A | `github.com/goreleaser/goreleaser` | OK | Approved — not a new dependency, only a new invocation path |
| `github.com/rhysd/actionlint` | Go module proxy | Established, already pinned and running in `ci.yml` today at the identical version | N/A | `github.com/rhysd/actionlint` | OK | Approved — not a new dependency |
| `namespacelabs/nscloud-cache-action` (GitHub Action, not a Go module) | GitHub Marketplace / npm-style Action registry | Active, tagged releases through v1.6.1, resolved SHA confirmed to match the tag via GitHub API | N/A | `github.com/namespacelabs/nscloud-cache-action` | OK | Approved — SHA-pin per repo convention |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** none — all four are pre-existing,
production-proven dependencies of either this repo or its sibling reference
implementation; none were discovered via an unverified web search with no
registry/reputation cross-check.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────┐
                    │   Taskfile.yml (repo root)│
                    │  single source of truth    │
                    │  for every command         │
                    └──────────┬─────────────────┘
                               │  GO_TOOL = "GOWORK=off go tool -modfile=go.tool.mod"
                               │  GO_TOOL_LINT = "GOWORK=off go tool -modfile=go.tool-lint.mod"
                               ▼
              ┌────────────────────────────────────┐
              │  go.tool.mod (task, goreleaser)      │
              │  go.tool-lint.mod (actionlint)        │◄──── Go module proxy
              │  isolated from root go.mod's graph    │      (D-04: not GitHub Releases —
              │  (D-03: MVS conflict if merged)       │       removes CDN-flake class)
              └────────────────────────────────────┘

   Invocation path A (human, local):        Invocation path B (CI):
   ┌───────────────────────┐                ┌──────────────────────────────┐
   │ contributor shell      │                │ .github/workflows/ci.yml      │
   │ `task test`  (coarse)  │                │  job: test                    │
   │ `task lint`  (coarse)  │                │   step: Checkout               │
   └──────────┬─────────────┘                │   step: setup-go(cache:false) │
              │ deps: [test:unit,             │   step: nscloud-cache-action   │
              │  test:golden, test:integration,│   step: install-task (composite)│
              │  test:daemon, test:race]       │   step: run: task build        │
              ▼                                │   step: run: task test:unit    │
   ┌───────────────────────┐                   │   step: run: task test:golden  │
   │  fine-grained targets  │◄──────────────────┤   step: run: task test:...     │
   │  test:unit, test:race, │   (D-02: CI calls  │   (comments/step names unchanged)│
   │  vet, check:cross, ... │    THESE directly, └──────────────────────────────┘
   │  lint:actions,         │    never the coarse
   │  check:goreleaser      │    wrappers)
   └───────────────────────┘

   Release-critical path (D-15):
   .github/workflows/release-please.yml
     job: pretag-gate  ──(install-task, NOT Namespace per D-06 scope)──► task check:cross
                                                                          (replaces inline
                                                                           6-target go list sweep)

   Perf-baseline path (D-09/D-13, deliberately separate):
   .github/workflows/bench.yml
     job: rebless (CI-only, human-reviewed publish, never a local task target)
       1. record candidate baseline on Namespace linux/amd64
       2. human commits it in its own PR
       3. only then: job: perf-regression moves to Namespace
```

### Recommended Project Structure
```
Taskfile.yml              # single source of truth (repo root)
go.tool.mod                # tool { task, goreleaser } — isolated module
go.tool.sum                 # companion checksum file (NOT go.tool.mod.sum)
go.tool-lint.mod            # tool { actionlint } — isolated module (MVS conflict, D-03)
go.tool-lint.sum
.github/
  actions/
    install-task/
      action.yml            # composite action: GOWORK=off go build -modfile=go.tool.mod
                             #   -o $HOME/.local/bin/task github.com/go-task/task/v3/cmd/task
  workflows/
    ci.yml                  # run: bodies -> task <target>; runners -> Namespace (D-06)
    bench.yml               # same; rebless job moves to Namespace FIRST (D-09)
    release-please.yml      # pretag-gate's inline sweep -> task check:cross (D-15)
    release.yml             # runners -> Namespace except provenance job (D-07);
                             #   darwin leg stays native (D-08)
```

### Pattern 1: The `GO_TOOL` variable indirection

**What:** A Taskfile-level `vars:` entry holding the full `GOWORK=off go tool
-modfile=go.tool.mod` prefix, referenced as `{{.GO_TOOL}} <toolname> <args>` in
every task that needs a tool from the isolated module. A second `GO_TOOL_LINT`
var does the same for `go.tool-lint.mod`.

**When to use:** Any task invoking `task`, `goreleaser`, or `actionlint` — never
hand-repeat the `-modfile=` flag per task (a rename of the modfile becomes a
one-line change instead of a repo-wide find/replace).

**Example:**
```yaml
# Source: github.com/holomush/holomush Taskfile.yaml (public repo, fetched live)
vars:
  GO_TOOL: GOWORK=off go tool -modfile=go.tool.mod
  GO_TOOL_LINT: GOWORK=off go tool -modfile=go.tool-lint.mod

tasks:
  lint:actions:
    desc: Lint GitHub Actions workflows
    cmds:
      - "{{.GO_TOOL_LINT}} actionlint .github/workflows/*"

  check:goreleaser:
    desc: Validate goreleaser config
    cmds:
      - "{{.GO_TOOL}} goreleaser check"
```

### Pattern 2: `preconditions:` with actionable `msg:` for cross-toolchain gating (D-11)

**What:** Every task requiring a toolchain not guaranteed present on a clean
checkout (`zig` for cross-builds, `mingw-w64`/`x86_64-w64-mingw32-gcc` for the
Windows vet targets) declares a `preconditions:` entry with an `sh:` check and a
`msg:` explaining exactly what to install. A failing precondition fails the task
(and any task that `deps:`-on it) with that message — it does not skip silently.

**When to use:** `build:cross`, `vet:windows` — anything D-10 does NOT put in the
host-only `task test` wrapper.

**Example:**
```yaml
# Source: taskfile.dev/docs/guide (Context7, official docs, 2026-08-01) +
# github.com/holomush/holomush Taskfile.yaml web:install task (live pattern)
tasks:
  vet:windows:
    desc: "Typecheck windows PPID watchdog (GOOS=windows go vet internal/daemon) — needs mingw-w64"
    preconditions:
      - sh: command -v x86_64-w64-mingw32-gcc
        msg: "mingw-w64 not found. Install with: apt-get install gcc-mingw-w64-x86-64 (or your platform's equivalent). See CONTRIBUTING.md#building."
    env:
      CGO_ENABLED: "1"
      CC: x86_64-w64-mingw32-gcc
    cmds:
      - GOOS=windows GOARCH=amd64 go vet ./internal/daemon/

  build:cross:
    desc: "Cross-compile to a non-host GOOS/GOARCH via zig — needs zig on PATH"
    preconditions:
      - sh: command -v zig
        msg: "zig not found. Install per CONTRIBUTING.md's cross-build prerequisite section."
    cmds:
      - "{{.GO_TOOL}} goreleaser build --single-target --clean"
```

**Do NOT use `platforms:` for this.** `platforms: [windows]` restricts a task to
run only when the *host* OS matches — on a non-matching host it is **skipped
without an error** (confirmed via Context7's official docs, 2026-08-01). That is
the silent-skip failure class D-11 explicitly names and rejects (it names
`status:`, but `platforms:` is a second, easy-to-reach-for field with the
identical failure shape). These cross-toolchain targets are not host-OS-gated —
they compile *to* a different `GOOS` *from* the current host — so `platforms:` is
categorically the wrong tool even setting the silent-skip issue aside.

### Pattern 3: The `install-task` composite action (module-proxy bootstrap)

**What:** A composite GitHub Action that builds `task` from `go.tool.mod` via
`go build -modfile=` and adds it to `$GITHUB_PATH`, instead of an install script
or a marketplace Action pulling a prebuilt binary from GitHub Releases.

**When to use:** As the very first tool-dependent step in every CI job that later
calls `task <target>` (`ci.yml`'s jobs, `bench.yml`'s jobs, `release-please.yml`'s
`pretag-gate`, `release.yml`'s `build`/`assemble` jobs that need `task`).

**Example:**
```yaml
# Source: github.com/holomush/holomush .github/actions/install-task/action.yml
# (public repo, fetched live 2026-08-01)
name: Install Task
description: Build the pinned task binary from go.tool.mod and add it to PATH.
runs:
  using: composite
  steps:
    - name: Build Task from go.tool.mod
      shell: bash
      run: |
        set -euo pipefail
        mkdir -p "$HOME/.local/bin"
        # GOWORK=off: go.work enables workspace mode, which is incompatible
        # with -modfile.
        GOWORK=off go build -modfile=go.tool.mod \
          -o "$HOME/.local/bin/task" \
          github.com/go-task/task/v3/cmd/task
        echo "$HOME/.local/bin" >> "$GITHUB_PATH"
    - name: Verify Task
      shell: bash
      run: task --version
```

### Pattern 4: `setup-go cache:false` + `nscloud-cache-action cache:go` ordering

**What:** `actions/setup-go` runs with its own caching explicitly disabled
(`cache: false`), and `namespacelabs/nscloud-cache-action` (with `cache: go`)
runs immediately after, taking over module/build caching via Namespace's
NVMe-backed Cache Volumes instead of GitHub's tarball-based cache.

**When to use:** Every job moved to a `namespace-profile-*` runner (D-06).

**Example:**
```yaml
# Source: github.com/holomush/holomush .github/workflows/ci.yaml (public repo,
# fetched live 2026-08-01) + WebSearch cross-check of namespace.so docs
steps:
  - uses: actions/checkout@<sha> # vX
  - name: Set up Go
    uses: actions/setup-go@<sha> # vX
    with:
      go-version-file: go.mod
      cache: false
  - name: Cache Go modules and build
    uses: namespacelabs/nscloud-cache-action@c5f8dab7560444c4bf8dbc64f1b203431873c547 # v1.6.1
    with:
      cache: go
  - name: Install Task
    uses: ./.github/actions/install-task
```

### Anti-Patterns to Avoid

- **Merging `go.tool.mod` and `go.tool-lint.mod` into one file:** measured to not
  compile (`action_metadata.go:273:22: te.Errors[0].Error undefined`) — MVS
  resolves one version per module across the whole graph, so co-locating
  actionlint with task/goreleaser loses the YAML-API version actionlint expects.
- **Using `status:` OR `platforms:` to gate a cross-toolchain target:** both
  silently skip instead of failing — the GOLDEN-01 failure class. Use
  `preconditions:` with `msg:` (see Pattern 2).
- **Adding a `task` bootstrap step without `GOWORK=off`:** if a `go.work` file is
  ever added to this repo later (it does not exist today, confirmed by
  `ls go.work*` returning nothing), `-modfile` silently becomes incompatible with
  workspace mode. `GOWORK=off` is cheap insurance even though it is currently a
  no-op in this specific repo.
- **Changing a rewired job's rendered `name:` field:** GitHub's required-status-check
  ruleset (id `20157557`, confirmed live via `gh api .../rulesets/20157557`) matches
  on the **rendered job name**, not the workflow key or step name. D-02 already
  preserves job names/step names/comments — but this is worth stating explicitly
  as a hard constraint, since a well-meaning rename during the `task`-ification
  pass (e.g. tidying `"reproducibility (double-build hash-diff, DIST-04)"` to
  something shorter) would silently orphan a required check.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Tool version pinning across CI + local dev | A version-string constant duplicated in `Taskfile.yml` and each workflow's `go install @vX` line | `go.tool.mod`'s `require` block (auto-populated by `go get -tool`) | One file is the single source; `go.mod`-style MVS resolution is exactly the mechanism this ecosystem already trusts for every other dependency |
| CDN-flake resilience for tool binaries | A retry/fallback wrapper around `arduino/setup-task` or the `install.sh` shim | Fetch from the Go module proxy via `go build -modfile=` | The module proxy is backed by Go's own module cache and checksum database — already the trust root every other dependency in this repo goes through; a bespoke retry wrapper around a third-party CDN doesn't remove the CDN, it just hides its flakiness |
| Cross-platform module-graph sanity checking | A hand-rolled loop reimplementing what `release-please.yml:46-51` already does inline | `task check:cross` wrapping the SAME 6-target `go list -mod=readonly ./...` loop (D-15) | The behavior doesn't change — only where it's defined. Reimplementing it "cleaner" inside Taskfile syntax risks silently changing the target list; port the loop body verbatim |
| Detecting a required GitHub Actions toolchain (mingw-w64/zig) missing locally | Bespoke shell probing scattered across multiple targets | go-task's own `preconditions:` primitive (Pattern 2) | It is the language-native mechanism for exactly this: a check + an actionable message + a hard non-zero exit; there is no reason to reimplement it with raw `if ! command -v ...; then exit 1; fi` blocks when the field exists |

**Key insight:** every piece of this phase's actual novel engineering has already
been solved once, in public, by the same maintainer's other repo. The risk here
is not "can this be built" — the mechanics are all independently verified above —
it's copying the reference implementation's *scope* rather than its *pattern*.
holomush's Taskfile is ~1,500 lines wrapping a much larger, different command
surface (web client, protobuf codegen, Docker, Ginkgo/Playwright E2E, license
headers, ADR health checks — none of which apply here). The pattern (GO_TOOL var,
isolated modfiles, install-task composite, preconditions-not-status/platforms) is
the reusable asset; the specific 1,500 lines of task bodies are not.

## Common Pitfalls

### Pitfall 1: Required-status-check names are matched by rendered job `name:`, not workflow structure

**What goes wrong:** A PR that should be mergeable stays blocked forever because
its required check never reports under the name the branch-protection ruleset
expects.
**Why it happens:** GitHub's ruleset (confirmed live: id `20157557`, 6 required
contexts — `test`, `actionlint (workflow static analysis)`,
`perf regression gate (PERF-02, INDX-06)`, `pr-title`,
`reproducibility (double-build hash-diff, DIST-04)`,
`govulncheck (DIST-03, blocking)`) matches the **rendered** `name:` string of a
job. `task`-ifying a `run:` body does not touch `name:` under D-02, but any
incidental cleanup during the pass (shortening a verbose job name while you're in
the file) silently breaks this.
**How to avoid:** Diff every job's `name:` field before/after the rewrite;
treat any change to a `name:` field as a separate, deliberate PR requiring a
ruleset update, never a byproduct of the `task`-ification pass.
**Warning signs:** A PR shows "Some checks haven't completed yet" indefinitely
after all jobs finish; the ruleset's required-context list, fetched via
`gh api repos/<owner>/<repo>/rulesets/<id>`, contains a name that no longer
appears in any workflow run.

### Pitfall 2: `platforms:` vs `preconditions:` — same silent-skip shape, different field

**What goes wrong:** A contributor on macOS runs `task vet:windows` (or it runs
implicitly via a `deps:` chain) expecting either success or a loud, actionable
failure; instead the task silently does nothing and reports success.
**Why it happens:** go-task's `platforms:` field is designed for exactly this
shape of "only run on OS X" restriction and is easy to reach for when writing an
OS-gated target — but its documented behavior is to **skip without an error**
when the host doesn't match, not to fail. D-11 named-and-rejected `status:` for
this reason but did not name `platforms:`, because at CONTEXT-gathering time this
field's silent-skip semantics were not yet independently confirmed. This
research confirms it via Context7's official docs.
**How to avoid:** Use `preconditions:` with `sh:`/`msg:` (Pattern 2) for every
cross-toolchain target — never `platforms:`, never `status:`.
**Warning signs:** A "green" `task test` or `task vet` run on a contributor's
machine that never actually exercised the windows-cross-vet or zig-cross-build
logic; compare against CI's actual step count.

### Pitfall 3: Pre-existing goreleaser version mismatch between `ci.yml` and `release.yml`

**What goes wrong:** The planner picks a goreleaser version for `go.tool.mod`
without checking both existing pins, silently changing what `goreleaser check`
validates locally versus what the release pipeline actually runs.
**Why it happens:** `ci.yml`'s `goreleaser-check` job (line 341) currently pins
`goreleaser/goreleaser/v2@v2.17.1`; `release.yml`'s build matrix (line 51) pins
`GORELEASER_VERSION: "v2.17.0"` — a one-patch-version mismatch that **already
exists in this repo today**, independent of this phase.
**How to avoid:** Pin `go.tool.mod`'s `goreleaser` to `v2.17.1` — it matches the
job this phase is directly replacing (`ci.yml`'s `goreleaser-check`). File the
`v2.17.0` vs `v2.17.1` discrepancy in `release.yml` as a separate, explicit
follow-up rather than silently bumping a release-critical pipeline's pinned
version as a side effect of a build-tooling phase (`release.yml` was
Phase-8-audited and Phase-9-verified against a real published artifact —
touching its pin outside a dedicated, reviewed change is exactly the kind of
scope creep CONTEXT.md's reversibility notes warn against for that file).
**Warning signs:** `task check:goreleaser` passes locally on a config detail that
`goreleaser build` in `release.yml` then fails on, or vice versa.

### Pitfall 4: `release-please.yml` needs the tool bootstrap but is NOT in D-06's runner-migration scope

**What goes wrong:** The planner assumes every workflow touched by this phase
also gets the Namespace-runner treatment, and either (a) moves
`release-please.yml` to Namespace when CONTEXT.md never asked for that, or (b)
forgets to add the `install-task` composite action step to `pretag-gate` because
"it's not a Namespace job so it must not need the new tooling."
**Why it happens:** D-06 names exactly three files (`ci.yml`, `bench.yml`,
`release.yml`); D-15 (a different gray area) separately makes
`release-please.yml`'s `pretag-gate` job depend on `task check:cross`. These are
independent decisions that happen to intersect on the same job.
**How to avoid:** `release-please.yml`'s `pretag-gate` job stays on
`runs-on: ubuntu-latest` (unchanged) but gains the `install-task` composite
action step (and a `setup-go` step, which it already has) so `task check:cross`
resolves. No Namespace/`nscloud-cache-action` wiring belongs there per D-06's
literal scope — though see Open Questions for a performance note about this job
running the tool build cold every time without Namespace's cache.
**Warning signs:** `pretag-gate` fails with `go: open go.tool.mod: no such file
or directory` (modfile present but no bootstrap step) or `task: command not
found` (bootstrap step missing entirely).

### Pitfall 5: The `bench:rebless`-vs-D-13 tension needs an explicit resolution, not a guess

**What goes wrong:** D-01 ("CI calls `task <target>` everywhere") and D-13 ("no
`-rebless` task target exists... reachable only through `bench.yml`") read as
contradictory if taken literally — CONTEXT.md itself flags this: *"Note the
tension with D-09 — the Namespace re-bless is run via the workflow, not via a
local target."*
**Why it happens:** These are reconcilable but the reconciliation isn't spelled
out in CONTEXT.md, so two different planners could resolve it two different ways
— one leaving the `-rebless` invocation as raw inline shell in `bench.yml`
(technically violating D-01's "everywhere"), the other creating a
`bench:rebless` task target that is technically callable by anyone on their own
laptop (undermining D-13's footgun-prevention intent, since nothing stops
`task bench:rebless` from being typed locally even if it isn't advertised).
**How to avoid (recommended resolution, not a CONTEXT.md decision — flag as an
Open Question for the plan to ratify explicitly):** create the task target (so
D-01 holds structurally) but gate it with a `preconditions:` check on a CI-only
environment signal (e.g. `test -n "${CI:-}"` with `msg: "bench:rebless only runs
in bench.yml's CI job — see CONTRIBUTING.md's rebless prohibition"`), mirroring
D-14's requirement that `bench:regression` "surface the platform guard's refusal
legibly." This closes the loophole without inventing new mechanism.
**Warning signs:** A contributor successfully runs `task bench:rebless` (or
whatever it's named) on a laptop and it silently writes a wrong-platform
candidate — the exact failure class that produced the "stable, entirely
fictitious 10.6% regression" `CONTRIBUTING.md` already warns about.

### Pitfall 6: `deps:` executes concurrently; `- task:` inside `cmds:` executes serially

**What goes wrong:** A coarse wrapper task (e.g. `task test`) is written with
`deps: [test:unit, test:golden, test:integration, test:daemon, test:race]`
expecting them to run in that order, but go-task's documented behavior is that
`deps:` entries run **concurrently** by default — order is not guaranteed.
**Why it happens:** It's an easy default to assume matches shell-script
top-to-bottom reading order; Context7's official docs confirm the opposite
("Dependencies are executed concurrently by default").
**How to avoid:** If order matters (e.g. `test:daemon` and `test:race` both pass
`-p 1` specifically to avoid cross-test contention — running them concurrently
with each other via `deps:` could reintroduce exactly the flakiness `-p 1` exists
to prevent), use `cmds: [{task: A}, {task: B}]` (serial) instead of `deps:`.
**Warning signs:** Intermittent local flakiness in a coarse wrapper that never
reproduces when the fine-grained targets are run one at a time — a signature of
accidental concurrency between targets that assume exclusive access to package
state.

## Code Examples

### The `check:cross` target replacing `release-please.yml`'s inline sweep (D-15)

```yaml
# Source: this repo's own .github/workflows/release-please.yml:46-51, ported
# verbatim per D-15 ("mirrors the 6-target go list -mod=readonly sweep").
# Target list independently cross-checked against .goreleaser.yaml's 6
# goos/goarch build entries (linux/{amd64,arm64}, windows/{amd64,arm64},
# darwin/{amd64,arm64}) — identical set, confirmed live 2026-08-01.
tasks:
  check:cross:
    desc: "6-target go list -mod=readonly sanity sweep (pre-tag gate, D-15)"
    cmds:
      - |
        set -euo pipefail
        for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
          GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null || {
            echo "::error::go list -mod=readonly ./... failed for GOOS=${pair%/*} GOARCH=${pair#*/}"
            exit 1
          }
        done
```

### The isolated tool module header comment (D-03's rationale, made durable)

```go
// Source: github.com/holomush/holomush go.tool.mod header (public repo,
// fetched live 2026-08-01) — adapt module path and tool list for this repo.
// Isolated tool-dependency module for build/lint tooling (Go 1.24+ `go tool`).
//
// Kept separate from the main go.mod so the application's dependency graph
// stays lean (D-03: a root-go.mod tool directive measured +237 net-new
// modules, dominated by cloud.google.com/go/* pulled in by go-task's
// remote-Taskfile support — those would land in the release SBOM and the
// blocking govulncheck gate's scope).
//
// task + goreleaser live here; actionlint is split into go.tool-lint.mod
// because its expected YAML-parsing API conflicts under MVS with what
// task/goreleaser pull in — verified: co-locating fails to compile with
// `action_metadata.go:273:22: te.Errors[0].Error undefined`.
//
// Tools are fetched from the Go module proxy (not GitHub Releases — removes
// the CDN-flake class that broke arduino/setup-task and the install.sh shim)
// and run via `GOWORK=off go tool -modfile=go.tool.mod <name>`. GOWORK=off is
// required because go.work enables workspace mode, which is incompatible
// with -modfile.
//
// Neither Dependabot nor Renovate is configured in this repo today — this
// modfile is unmanaged by tooling either way (D-05). If a dependency-update
// tool is added later, scope its go.mod manager to the root go.mod only.
module github.com/seanb4t/codegraph-go/tools

go 1.26

tool (
	github.com/go-task/task/v3/cmd/task
	github.com/goreleaser/goreleaser/v2
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|--------------------|----------------|--------|
| `go install <tool>@<pinned-version>` inline in each CI job (this repo's current `ci.yml` pattern for actionlint/goreleaser) | `go get -tool` writing a `tool` directive to `go.mod` (or an isolated modfile via `-modfile=`), invoked via `go tool <name>` | Go 1.24 (tool directive introduced) | Version pin lives in one file instead of duplicated `go install @vX` lines across every workflow that needs the tool; `go.sum` gives supply-chain verification the bare `go install` pattern didn't have |
| Reading CI YAML to learn how to build/test/lint locally (this repo's status quo, explicitly named in the phase's own goal statement) | A `Taskfile.yml` single entry point, with CI calling into the same definitions | This phase | Closes the exact gap ROADMAP criterion 3 names: "a clean checkout can build, test, and lint via task targets alone" |
| GitHub's own hosted runner cache (`actions/cache`, or `setup-go`'s built-in `cache: true`) | Namespace Cache Volumes via `nscloud-cache-action` + `setup-go cache: false` | D-06 (this phase) | NVMe-backed volume caching claimed lower latency than GitHub's tarball-based cache; unverified in this research beyond vendor-documented claims (tag as `[CITED: namespace.so docs]`, not independently benchmarked against this repo's own workload) |

**Deprecated/outdated:** none of this repo's existing tool-install patterns are
being *removed* as broken — `go install @pin` inline in a workflow step still
works and is not itself deprecated by Go tooling. This phase's motivation is
single-definition/CDN-flake-resilience, not obsolescence of the prior mechanism.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|-----------------|
| A1 | `goreleaser` should be pinned to `v2.17.1` in `go.tool.mod` (matching `ci.yml`'s current pin, not `release.yml`'s `v2.17.0`) | Standard Stack, Pitfall 3 | If the maintainer intended the reverse (standardize on `v2.17.0`, the release-pipeline's own pin), the plan pins the wrong version and `task check:goreleaser` validates against a config surface `goreleaser build` in `release.yml` doesn't exactly share |
| A2 | The `bench:rebless`-vs-D-13 tension should be resolved by creating the task target with a `CI`-env-var precondition, rather than leaving the `-rebless` invocation as raw inline shell in `bench.yml` | Pitfall 5 | This is this research's own recommended resolution to an explicitly-flagged-but-unresolved tension in CONTEXT.md, not a locked decision — the plan/maintainer could legitimately choose the other reading (D-01 has a narrow, documented exception for this one command) |
| A3 | `namespacelabs/nscloud-cache-action`'s NVMe-backed caching is meaningfully faster than GitHub's hosted cache for this repo's actual module/build-cache size | State of the Art | Sourced only from Namespace's own marketing/docs pages (`[CITED: namespace.so]`), not independently benchmarked; if the claimed speedup doesn't materialize for this repo's specific dependency graph, D-06's runner migration still delivers the caching-mechanism change but not necessarily the implied performance win |
| A4 | Namespace offers a real, native Apple-Silicon macOS runner profile today (not an emulated/cross-compiled one) | Open Questions | Sourced from WebSearch only (`[CITED: namespace.so/docs]`, not independently verified against a live Namespace account); if inaccurate, D-08's "must stay a real macOS runner" constraint would be violated by moving `release.yml`'s darwin leg to a Namespace profile, so this must be confirmed against Namespace's actual current product docs (or a live test) before the plan acts on it, not assumed from a single search |

## Open Questions

1. **Should `release.yml`'s `macos-latest` leg move to a Namespace macOS profile, or stay GitHub-hosted?**
   - What we know: D-06 says "Namespace everywhere... including release.yml"; D-08 says the darwin leg "must stay a real macOS runner" if moved to Namespace. WebSearch (LOW confidence, see A4) indicates Namespace does offer native Apple M4 Pro/M5 Max macOS runners today, which would satisfy D-08's constraint if true.
   - What's unclear: whether this repo's Namespace account/plan actually has macOS runner access, and whether the maintainer wants to extend Namespace there at all given D-08's own "costly reversibility" framing for this specific leg.
   - Recommendation: treat as a maintainer decision point in the plan, not something research or planning resolves unilaterally — default to leaving `macos-latest` GitHub-hosted (the conservative reading that still satisfies D-06's spirit, since the darwin leg is explicitly the one CONTEXT.md flags as highest-risk to touch) unless the maintainer confirms Namespace macOS access.

2. **Does `release-please.yml`'s `pretag-gate` job need `nscloud-cache-action` even though it's outside D-06's Namespace-runner scope?**
   - What we know: this job runs on every push to `main` (not just release pushes), currently has no explicit Go module caching configured beyond `setup-go`'s default, and will now cold-build `task` from `go.tool.mod` (237 modules) on every single run once D-15 lands.
   - What's unclear: whether the added cold-build latency (measured 9.9s wall on a fast M-series machine for `task` alone; expect more on a shared `ubuntu-latest` runner, plus `goreleaser`'s own module graph if that tool is ever needed there too — though `check:cross` only needs `go list`, not `task` itself... actually it DOES need `task` to invoke `task check:cross` in the first place) meaningfully slows down every push to `main`.
   - Recommendation: measure the actual added wall-clock time once the plan lands (GitHub Actions' own `actions/cache` for the Go module cache, keyed on `go.tool.sum`, would help without requiring Namespace/nscloud-cache-action scope creep into a file D-06 deliberately excluded — flag this as a plan-time follow-up, not a blocker).

3. **What is the final requirement ID/wording for D-16?**
   - What we know: D-16 suggests `DEV-01`, explicitly pending maintainer ratification happening in parallel with this research.
   - What's unclear: exact final ID and wording.
   - Recommendation: the planner should confirm before writing `.planning/REQUIREMENTS.md` — this research's Validation Architecture section below is written against the suggested `DEV-01` wording and should be trivially re-keyed if the ID changes.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | Everything in this phase | ✓ | go1.26.5 darwin/arm64 (matches `go.mod`'s `go 1.26.5` directive) `[VERIFIED: go version, this session]` | — |
| `go tool -modfile=` / `go get -tool` support | D-03's isolated modfile mechanism | ✓ | Confirmed working live: `go help get` shows `-tool` flag; `GOWORK=off go tool -modfile=<missing>` fails loud with exit 1 `[VERIFIED: this session]` | — |
| `go.work` file | GOWORK=off's real-world relevance | ✗ (none present) | — | `GOWORK=off` is currently a no-op protective measure, not a fix for an active conflict — still recommended per holomush's precedent for forward-safety |
| `zig` | `build:cross` precondition target | not checked this session (not required for research; CONTRIBUTING.md already documents the prerequisite) | — | Task target fails loud via `preconditions:` per Pattern 2 — no silent degrade needed |
| `mingw-w64` (`x86_64-w64-mingw32-gcc`) | `vet:windows` precondition target | not checked this session | — | Same — `preconditions:` fails loud |
| Dependabot / Renovate | D-05's "unmanaged tool modfile" concern | ✗ (neither `.github/dependabot.yml` nor `renovate.json`/`.github/renovate.json` exists in this repo) `[VERIFIED: ls, this session]` | — | **New finding:** D-05's entire premise (a hypothetical automated-dependency-update tool that would need to be told to ignore the tool modfiles) is currently moot for this specific repo — there is no such tool configured at all today. The Claude's Discretion item ("whether Dependabot/Renovate config gains explicit ignore entries") has no config file to add entries to yet; only relevant if/when one is introduced later |
| GitHub ruleset `20157557` (required status checks) | Pitfall 1's job-name-preservation constraint | ✓ | 6 required contexts confirmed live via `gh api` (this session): `test`, `actionlint (workflow static analysis)`, `perf regression gate (PERF-02, INDX-06)`, `pr-title`, `reproducibility (double-build hash-diff, DIST-04)`, `govulncheck (DIST-03, blocking)` | — |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** `zig`/`mingw-w64` absence on a given
contributor machine is handled structurally by D-11's `preconditions:` design
(fail loud with install instructions), not a research-time gap.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's standard `testing` package, using this repo's existing workflow-shape-guard idiom (`internal/upgrade/release_workflow_shape_test.go`, `internal/upgrade/pr_title_lint_test.go` — both read real `.yml` off disk and parse it with `parseX`/`mustX` helper pairs that always return a non-nil error rather than a usable zero value, per the pattern documented at `internal/upgrade/release_workflow_shape_test.go:19-24`, which reads verbatim: `"Each helper below is a pure parseX(src string) (T, error) core plus a thin mustX(t *testing.T, src string) T wrapper that fails the test on error. Every core returns a non-nil error — never a usable zero value — when its target is absent (CR-01 class defect this phase exists to stop..."` `[VERIFIED: internal/upgrade/release_workflow_shape_test.go:19-24, read this session]`) |
| Config file | none — plain `go test` |
| Quick run command | `go test ./internal/... -run TestTaskfile` (once the guard test files exist — see Wave 0 Gaps) |
| Full suite command | `go test ./...` (unchanged — new guard tests are ordinary Go test files, reached by the existing filtered `go list ./...` CI step) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|----------------------|---------------|
| `DEV-01` (tentative) | Every non-trivial `run:` body in `ci.yml`/`bench.yml`/`release-please.yml`/`release.yml` invokes `task <target>` rather than inlining shell directly | unit (workflow-shape guard, property assertion — parse each `run:` block, assert it is either a bare `task ...` invocation or one of a small documented allowlist of exceptions: `govulncheck`-action has no command form, the SLSA provenance job is a reusable-workflow `uses:`, etc.) | `go test ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask -v` | ❌ Wave 0 — new test |
| `DEV-01` (tentative) | Required-status-check job `name:` fields are unchanged across the rewrite (Pitfall 1) | unit (property: the exact 6-name set from `gh api .../rulesets/20157557`, captured as a literal fixture at plan time, is still present as a job `name:` in the post-rewrite workflow files) | `go test ./internal/upgrade/... -run TestRequiredCheckNamesPreserved -v` | ❌ Wave 0 — new test |
| `DEV-01` (tentative) | `task check:cross` and `release-please.yml`'s (former) inline sweep enumerate the identical 6 `GOOS/GOARCH` pairs, matching `.goreleaser.yaml`'s 6 build targets | unit (property: parse `Taskfile.yml`'s `check:cross` task body and `.goreleaser.yaml`'s `builds[].goos`/`goarch` entries; assert set equality — per this project's own house rule against `rg -c`/exit-status chaining for exact multi-value comparisons, per `~/.claude/rules/grepping.md`) | `go test ./internal/upgrade/... -run TestCheckCrossMatchesGoreleaserTargets -v` | ❌ Wave 0 — new test |
| `DEV-01` (tentative) | A clean checkout can run `task test`, `task build`, `task lint` (or whatever the coarse wrapper names resolve to) end-to-end with only a C toolchain (D-10) | integration / manual-verified (spawns real subprocess `task`, matching this repo's existing TEST-04 precedent of driving real binaries rather than in-process calls — see `test/integration/main_test.go:1-13`, quoted in `<user_constraints>` D-02 above) | `task test` itself, run as CI's own smoke check inside a fresh checkout (e.g. a dedicated `smoke` job in `ci.yml` that clones into a scratch dir and runs only `task test`) | ❌ Wave 0 — new CI job, or documented as human-verified at plan-check time |

### Sampling Rate

- **Per task commit:** `go test ./internal/upgrade/... -run TestWorkflow` (fast — pure-Go YAML/text parsing, no external tool invocation)
- **Per wave merge:** `go test ./...` (existing full suite, now also reaching the new workflow-shape guard tests via the same filtered `go list ./...` CI step)
- **Phase gate:** Full suite green, PLUS a real `task test`/`task build`/`task lint` run on the developer's own machine (or a scratch-checkout CI smoke job) before `/gsd-verify-work` — because the workflow-shape guards above prove the *text* of the workflows is structurally correct, not that `task` itself actually executes successfully end-to-end. Both levels are needed; neither subsumes the other (a direct application of this project's TEST-04 precedent: in-process/text-level checks structurally cannot prove reachability).

### Wave 0 Gaps

- [ ] `internal/upgrade/taskfile_shape_test.go` (or similar) — new file, covers the three `DEV-01` unit-test rows above. Follow the existing `parseX`/`mustX` pattern from `release_workflow_shape_test.go` exactly (non-nil error, never a zero value, on any parse failure).
- [ ] A fixture capturing the ruleset's 6 required-check names as of plan time (`gh api repos/seanb4t/codegraph-go/rulesets/20157557` output, or a hand-maintained literal list with a comment explaining where it came from and how to re-verify it) — needed by `TestRequiredCheckNamesPreserved`.
- [ ] Non-vacuity proof for each new guard test: run it against the **pre-phase** workflow files (current `git HEAD`, before any `task`-ification) and confirm it goes RED for the right reason (e.g. `TestWorkflowRunBodiesInvokeTask` should fail today because `ci.yml`'s `run:` bodies are currently raw shell, not `task` calls) before landing the GREEN state. This directly applies Phase-8's `CR-01`/`WR-02` lesson ("a guard that is present but never fires is not a guard") and Phase-9's `T-09-08-06` lesson ("prefer property assertions over byte-equality") — both already named in CONTEXT.md's canonical refs, restated here because Validation Architecture is exactly where they get operationalized.
- [ ] No new test framework/dependency needed — this repo's existing `go test` + workflow-shape-guard idiom covers the shape entirely; the only genuinely new *execution* surface (does `task` actually run) needs either a scratch-checkout CI smoke job or an explicit human-verify checkpoint at plan-check time, not a new library.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V1 Architecture / secure design | yes | Isolated tool-dependency module graph (D-03) is itself an architectural control limiting blast radius — a compromised transitive dependency of `go-task`/`goreleaser` cannot reach the release binary's own SBOM/govulncheck surface, because it is structurally never linked into `./cmd/codegraph` |
| V10 Malicious code / supply chain | yes | D-04's module-proxy-not-GitHub-Releases sourcing is the standard control here: the Go module proxy enforces checksum-database verification (`go.sum`) on every fetch; a GitHub-Releases-based installer (the rejected `arduino/setup-task`/`install.sh` alternatives) has no equivalent built-in integrity mechanism beyond whatever the installer script itself implements |
| V14 Configuration | yes | SHA-pinning every third-party GitHub Action (`nscloud-cache-action`, `install-task`'s own composite-action internals) — this repo's existing, unbroken convention, extended to the new Namespace-related actions without exception |
| V5 Input Validation | n/a for this phase | This phase touches no user-facing input surface — it is internal build tooling only |
| V2/V3/V4/V6 (auth/session/access-control/crypto) | n/a for this phase | Not applicable — no new authentication, session, authorization, or cryptographic surface is introduced |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Dependency confusion / typosquat on a newly-added tool module | Tampering | All three tool packages (`go-task/task`, `goreleaser/goreleaser`, `rhysd/actionlint`) are **pre-existing** dependencies of this repo or its already-verified CI pipeline — no new, unaudited package name is being introduced. Verified against the Go module proxy directly (an authoritative source) in this research, not merely "found via search" |
| CDN/registry compromise or flake substituting a malicious binary for a tool install | Tampering, Denial of Service | D-04: module-proxy sourcing with `go.sum` checksum verification, replacing GitHub-Releases-based installers that were independently observed (in `holomush/holomush` production, incidents `holomush-06vz`/`holomush-ar31b`) to have availability failures — the security property (checksum-verified fetch) is a byproduct of D-04's reliability motivation, not a separate control |
| A CI-only credential or write-capable step accidentally reachable from a `task` target a contributor can run locally | Elevation of Privilege | Directly the shape Pitfall 5 / Assumption A2 addresses for `bench:rebless` — the recommended `preconditions:` gate on a CI-only environment signal prevents a contributor from locally invoking a target whose *intent* is CI-exclusive, even though nothing in `task`'s own execution model enforces that boundary by default |
| Tool-module supply chain expanding the release binary's blocked-vuln surface (D-03's core concern) | Tampering / Repudiation (an unaudited transitive dep silently in the trust chain) | Isolation (D-03) confirmed structurally sufficient by this research: `govulncheck` scans the graph reachable from the `go.mod` its action is pointed at (the root `go.mod`, unchanged), and the release SBOM (`syft <binary>`) inspects the compiled artifact's embedded module info — code never imported by `./cmd/codegraph` cannot appear in either regardless of what `go.tool.mod` requires |

## Sources

### Primary (HIGH confidence)
- `taskfile.dev` official docs, via Context7 (`/websites/taskfile_dev`) — schema `version`, `preconditions:`, `status:`, `deps:` (concurrent-by-default), `- task:` (serial), `platforms:` (silent-skip semantics), `requires:`, `includes:`, `set:` — fetched live 2026-08-01
- This repo's own live files, read directly this session: `.github/workflows/ci.yml`, `bench.yml`, `release-please.yml`, `release.yml`, `CONTRIBUTING.md`, `go.mod`, `.goreleaser.yaml`, `release-please-config.json`, `internal/upgrade/release_workflow_shape_test.go`, `test/integration/main_test.go`
- Live command-line verification this session: `go help get` (`-tool` flag), `go help tool` (`-modfile`, `-C` flags), a scratch `go.tool.mod` sibling file's zero effect on `go list ./...` package count, `GOWORK=off go tool -modfile=<missing>` exit-code behavior
- `go.dev/doc/modules/managing-dependencies` (official Go docs, WebFetch, 2026-08-01) — `go get -tool`, `go tool <name>`, `go mod tidy` interaction with tool directives
- `github.com/holomush/holomush` (public GitHub repo, fetched live via `raw.githubusercontent.com` and the GitHub REST API, 2026-08-01) — `Taskfile.yaml` (1,491 lines), `go.tool.mod` header comment, `.github/actions/install-task/action.yml`, `.github/workflows/ci.yaml` — the reference implementation this phase is explicitly instructed to study
- `gh api repos/seanb4t/codegraph-go/rulesets/20157557` (live, this session) — the exact 6 required-status-check context strings
- `github.com/namespacelabs/nscloud-cache-action`'s GitHub API commit/tag data (live, this session) — resolved the SHA `c5f8dab7560444c4bf8dbc64f1b203431873c547` to tag `v1.6.1`
- Go module proxy (`proxy.golang.org`, via `go list -m -versions`, this session) — version currency for `go-task/task/v3`, `goreleaser/goreleaser/v2`, `rhysd/actionlint`

### Secondary (MEDIUM confidence)
- WebSearch, cross-checked against official Go docs: Go 1.24 `tool` directive overview (multiple independent blog posts converging on the same `-tool`/`-modfile` mechanics already confirmed live against this repo's toolchain)
- WebSearch: `govulncheck` scope (`golang/govulncheck-action` docs summary) — "analyzes the full build graph, including test-only and tooling dependencies declared in go.mod" — confirms the mechanism this research relies on (isolation via a *separate, unreferenced* go.mod keeps tool deps out of that graph entirely)
- WebSearch: `namespacelabs/nscloud-cache-action` + `setup-go cache:false` interaction pattern — corroborates the pattern independently observed in holomush's live `ci.yaml`

### Tertiary (LOW confidence)
- WebSearch: Namespace macOS runner (Apple M4 Pro/M5 Max) availability claim — vendor marketing/docs pages only, not independently verified against a live Namespace account or a real workflow run; flagged as Assumption A4 and Open Question 1
- WebSearch: Namespace Cache Volumes' latency/performance claims vs GitHub's hosted cache — vendor-sourced only, flagged as Assumption A3

## Metadata

**Confidence breakdown:**
- Standard stack (tool versions, module proxy availability): HIGH — every version claim independently verified against `proxy.golang.org` or this repo's own live pinned versions this session, not taken from training data
- Architecture (Taskfile/tool-modfile mechanics): HIGH — verified live against this repo's actual Go 1.26.5 toolchain (modfile sibling-file behavior, exit codes, `-tool`/`-modfile` flag existence), plus the reference implementation fetched live from a public repo
- CI rewiring surface (which files/steps change): HIGH — every workflow file was read directly this session; the "authoritative list" in CONTEXT.md's canonical_refs was independently cross-checked line-by-line against the live files, and two scope gaps were found (Pitfall 4's release-please.yml exclusion, Pitfall 3's goreleaser version mismatch) that CONTEXT.md's own text did not flag
- Pitfalls: HIGH for the mechanically-verified ones (job-name matching, `platforms:` silent-skip, `deps:` concurrency); MEDIUM for the ones synthesized from CONTEXT.md's own flagged tensions (bench:rebless resolution) since those are this research's recommendation, not a verified fact
- Namespace-specific product claims (macOS runner availability, cache performance): LOW — vendor-sourced web search only, explicitly flagged in the Assumptions Log

**Research date:** 2026-08-01
**Valid until:** 30 days for the Go-tooling mechanics (stable, unlikely to change); 7-14 days for Namespace product-specific claims (fast-moving vendor product, unverified beyond marketing docs) and for the exact tool version pins (all three tools release frequently — re-verify via `go list -m -versions` at plan-execution time, not from this document, if more than ~2 weeks have elapsed)
