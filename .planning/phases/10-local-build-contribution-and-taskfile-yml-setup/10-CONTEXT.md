# Phase 10: Local Build Tooling & CONTRIBUTING - Context

**Gathered:** 2026-08-01
**Status:** Ready for planning
**Mode:** interactive (`discuss`), no flags

<domain>
## Phase Boundary

Give the repo a **single local entry point for every build/test/lint/release-check
command**, and make CI consume that same entry point so the commands have exactly
one definition.

Today the repo has **no `Makefile`, no `Taskfile`, no `justfile`** (`scripts/`
holds one Python file, `pr_template_policy.py`). Every invocation lives only
inside `.github/workflows/`, so a contributor must reverse-engineer CI YAML to
build or test.

**In scope:**
- `Taskfile.yml` at repo root — fine-grained targets mirroring each named CI step,
  plus coarse wrapper targets for humans
- Two isolated tool modfiles at repo root: `go.tool.mod` + `go.tool-lint.mod`
  (+ their `.sum` files)
- Rewiring `.github/workflows/*.yml` so each step's `run:` body becomes a
  `task <target>` call, and moving CI onto Namespace runners + `nscloud-cache-action`
- A **pointer-sized** `CONTRIBUTING.md` amendment directing contributors at the
  task targets
- Minting a requirement ID for this phase (ROADMAP currently says `Requirements: TBD`)

**Not in this phase (explicit out-of-scope):**
- **Rewriting `CONTRIBUTING.md`'s toolchain prose.** ROADMAP criterion 2 is
  **already satisfied** — see D-00. Only a pointer is added.
- Changing what any command *does*. This phase changes *where commands are
  defined* and *what invokes them*, never their semantics.
- Any CLI/MCP behavioral surface change.
- Target naming conventions, `task`-with-no-args behavior, and Dependabot/Renovate
  config for the new modfiles — Claude's discretion (see below).
- Adding `goreleaser-check` to `main`'s required-status-check set — deferred.

</domain>

<decisions>
## Implementation Decisions

### D-00 — ROADMAP criterion 2 is already satisfied; record, do not redo

`CONTRIBUTING.md` already exists (156 lines) and its `## Building` section already
documents the CGo requirement, links `PARSER-DECISION.md`, and names **both**
`zig` (cross-builds) and `mingw-w64` (Windows vet) — the exact content criterion 2
asks for. It landed during the OSS-readiness work, outside any phase plan.

**Do not rewrite it.** The phase's CONTRIBUTING work is a pointer to the new task
targets. Record criterion 2 as pre-satisfied in `10-VERIFICATION.md` rather than
manufacturing work to "close" it.

`[user] Scope → Selected: Taskfile + CONTRIBUTING pointer (recommended)`

### Gray area 1 — Who owns the command definitions

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

### Gray area 2 — Tool bootstrap (MEASURED, not argued)

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

### Gray area 3 — Runners and caching

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

### Gray area 4 — Test scope and missing toolchains

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

### Gray area 5 — Dangerous and release-critical targets

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

### Gray area 6 — Requirement ID

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

### Folded Todos
None.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope
- `.planning/ROADMAP.md` §"Phase 10: Local Build Tooling & CONTRIBUTING" — goal,
  3 success criteria, and the Notes paragraph recording the v1.0-vs-v1.1 fit
  caveat. **Criterion 2 is pre-satisfied — see D-00.**
- `.planning/REQUIREMENTS.md` — currently has **no** entry for Phase 10
  (`Requirements: TBD` in ROADMAP). D-16 mints one.
- `.planning/STATE.md` §"Operator Next Steps" — records that a scope decision was
  the gate for this phase; that decision is D-00.

### The reference implementation (read this before designing anything)
- `github.com/holomush/holomush` (PUBLIC) — a sibling Go repo already running this
  exact pattern in production. Specifically:
  - `.github/workflows/ci.yaml` — Namespace profiles, `setup-go cache: false` +
    `nscloud-cache-action cache: go`, and CI invoking `task <target>` throughout
  - `.github/actions/install-task/action.yml` — the composite action that builds
    `task` from `go.tool.mod` via `-modfile`, with `GOWORK=off`, and the CDN-flake
    rationale for rejecting `arduino/setup-task`
  - `go.tool.mod` header comment — the definitive explanation of why the tools
    module is split in two, and why Renovate cannot manage it

### Commands being wrapped (the authoritative list)
- `.github/workflows/ci.yml` — the `test` job's **six deliberately separate steps**
  (lines 60–146), each with a comment explaining a past failure. `govulncheck`
  (line 148, via `golang/govulncheck-action@v1.1.0` — an action, not a runnable
  command, so it has no direct task equivalent), `reproducibility` (line 165),
  `perf-regression` (line 253), `actionlint` (line 306, `@v1.7.12`),
  `goreleaser-check` (line 325, `@v2.17.1`).
- `.github/workflows/bench.yml` — `go run ./tools/bench/runner -mode {headtohead,regression}`,
  `-rebless`, `-trials`. Lines 101, 158, 229.
- `.github/workflows/release-please.yml:46-51` — the 6-target
  `go list -mod=readonly` sweep that D-15 replaces.
- `.github/workflows/release.yml:66-95` — the native 2-OS build matrix (D-08);
  line 339 — the SLSA reusable workflow (D-07).

### Guards and patterns that constrain this phase
- `test/integration/main_test.go:1-13` — states in-file *why* the named CI step
  exists. **Read before touching `ci.yml`'s test job.**
- `internal/upgrade/release_workflow_shape_test.go` — the `parseX`/`mustX`
  guard pattern (every parser returns a non-nil error, never a usable zero value).
  Not required by D-02, but the template if any guard is added.
- `internal/upgrade/pr_title_lint_test.go` — second instance of the same pattern.
- `tools/bench/baseline.json` + `tools/bench/BASELINE.md` — the perf baseline and
  its provenance rules. **Never hand-edited** (Phase-9 D-05).
- `tools/bench/runner/main.go` — `CheckRegression`'s platform comparison
  (`runtime.GOOS`/`GOARCH` only — the D-09 blind spot).

### Docs this phase touches
- `CONTRIBUTING.md` §"Building" (lines 57–75) — **already correct**, gets a pointer
  to the task targets only. §"Never do these" (lines 112–128) — the `-rebless`
  prohibition D-13 preserves.
- `docs/RELEASE-PROCEDURES.md` §1 — the 6-target sweep's prose home; update if
  D-15 changes the invocation.

### Cross-phase constraints carried forward (MUST honor)
- Repo rule `xmz3xknbj0` (engram) — `-c commit.gpgsign=false` is allowed for
  agent/pipeline commits ONLY; sign when the user is at the keyboard and asks.
- **Pin every third-party Action to a full commit SHA** with the resolved tag in a
  trailing comment. `nscloud-cache-action` and any Namespace action inherit this.
- **Never interpolate `${{ }}` directly into a `run:` script** — pass via `env:`.
- Phase-8 lesson (`CR-01`, `WR-02`): a guard that is present but never fires is not
  a guard. Any new gate must be proven non-vacuous against a rejecting input.
- Phase-9 lesson (`T-09-08-06`): **point-in-time diff assertions decay; durable
  guards do not.** Prefer property assertions over byte-equality.

### External docs (fetch current versions during research — do not work from memory)
- `go-task` — `/llmstxt/taskfile_dev_llms-full_txt` or `/go-task/task`.
  Confirmed relevant: `preconditions:` (with `msg:`), `status:`, `deps:`,
  `includes:`. Current version measured: **v3.52.0**.
- `namespacelabs/nscloud-cache-action` — inputs, `cache: go` semantics, and how it
  interacts with `actions/setup-go`'s `cache: false`.
- Namespace runner profile labels and macOS availability (needed for D-08).
- Go `tool` directives + `go tool -modfile` / `-C` (Go 1.24+; repo is on **1.26.5**).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`.github/workflows/ci.yml`** — already an unusually well-commented, decomposed
  set of gates. Each of the six test steps encodes a past failure in its comment.
  These comments live in the workflow and **stay there** under D-02; only the
  `run:` bodies move.
- **`internal/upgrade/*_shape_test.go`** — a proven, idiomatic pattern for tests
  that read real `.yml` off disk and fail on drift. Two instances.
- **`tools/bench/runner`** — already has a platform guard and median-of-N
  (`-trials`); `task bench:*` wraps rather than reimplements.
- **`CONTRIBUTING.md`** — 156 lines, already accurate on toolchain prerequisites.
- **`github.com/holomush/holomush`** — the whole pattern, already working, same
  maintainer. The single highest-value reference for this phase.

### Established Patterns
- **Version-pinned `go install …@vX`** for CI tools — the pattern D-03 replaces
  with modfiles (`actionlint@v1.7.12`, `goreleaser@v2.17.1`).
- **SHA-pin every third-party Action**, resolved tag in a trailing comment. The
  documented sole exception is the SLSA generator, which `slsa-verifier` requires
  be referenced by a `@vX.Y.Z` tag.
- **Hand-written, auditable CI shell over opaque actions** where a contract must be
  exact. D-01 moves that shell into a Taskfile — it does not make it opaque.
- **Fail loud, never silently skip.** GOLDEN-01 (Phase 2 Critical) is the named
  precedent; D-11 is the direct application.
- **CI job names encode requirement IDs** so a red run is self-describing from the
  job list. D-02 preserves this.

### Integration Points
- **New** `Taskfile.yml`, `go.tool.mod`, `go.tool.sum`, `go.tool-lint.mod`,
  `go.tool-lint.sum` at repo root.
- **New** `.github/actions/install-task/` composite action (holomush's shape) —
  every job that calls `task` needs it.
- **`ci.yml`** → every `run:` becomes `task <target>`; runners → Namespace;
  `setup-go cache: false` + `nscloud-cache-action`.
- **`bench.yml`** → same; rebless job moves FIRST (D-09).
- **`release-please.yml:46`** → inline sweep becomes `task check:cross` (D-15).
- **`release.yml`** → runners move except the SLSA job (D-07); darwin leg stays
  native (D-08).
- **`tools/bench/baseline.json`** → re-blessed on Namespace; gains a runner field
  (D-09).
- **`CONTRIBUTING.md`** → pointer paragraph only.

### Landmines found during scout
- ⚠ **`tools/` is NOT available for the tools module.** `tools/bench/{gencorpus,
  realcorpus,runner}` are part of the **main module** and `bench.yml` runs
  `go run ./tools/bench/runner` from the repo root three times. A `go.mod` at
  `tools/` would silently excise all three from `go list ./...` and break those
  invocations. D-03's root-level modfiles avoid this entirely.
- ⚠ `govulncheck` runs via an **action**, not a command — there is no CI command
  for a task target to reuse. Wrapping it means choosing an invocation, which
  changes the mechanism of a **blocking** gate (DIST-03). Deliberately left alone.

</code_context>

<specifics>
## Specific Ideas

- Measured, and load-bearing for the plan — do not re-derive:
  - Root `go.mod` + go-task tool directive = **237 net-new modules** (511 → 748, +46%),
    dominated by `cloud.google.com/go/*`.
  - `task` + `goreleaser` + `actionlint` in one tool module **does not compile**:
    `action_metadata.go:273:22: te.Errors[0].Error undefined (type string has no
    field or method Error)`.
  - `actionlint` alone in its own modfile: **16 modules**, builds clean at v1.7.12.
  - `go tool task` cold-cache build **9.9s**, warm **0.68s** (fast M-series;
    CI runners slower).
  - Nested `go.mod` exclusion from a parent's `./...` confirmed by experiment.
  - `go tool` accepts both `-modfile=` and `-C`; both invocations verified working.
- The `install-task` composite action should carry the CDN-flake rationale as an
  in-file comment, the way holomush's does — otherwise a future contributor
  "simplifies" it back to `arduino/setup-task`.
- Each tool modfile needs a header comment explaining **why the split exists**
  (the actionlint MVS conflict). Without it the two files look like an accident and
  someone will merge them — and get a compile error they cannot explain.
- Phase 9's `T-09-08-06` lesson applies directly: if any guard is added here, assert
  a **property**, never bytes. A byte-diff assertion over `ci.yml` would go red on
  every unrelated edit.

</specifics>

<deferred>
## Deferred Ideas

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

### Reviewed Todos (not folded)
Three todos matched Phase 10 by keyword and **none were folded** — all three are
already closed, and only *look* pending because of directory placement:
- `2026-07-14-document-release-cut-procedures-runbook.md` — `status: resolved`
  (folded into Phase 9 as an update to `docs/RELEASE-PROCEDURES.md`)
- `2026-07-31-bisect-indexer-throughput-regression.md` — `status: refuted`
  (there was never a regression; measurement-frame error)
- `2026-07-31-rebless-perf-baseline-on-ubuntu-latest.md` — `status: resolved`

⚠ All three sit in `.planning/todos/pending/` with non-pending frontmatter.
`todo.match-phase` reports **directory membership, not status** — a known gotcha.
Worth a cleanup pass, but not this phase's work. **Note:** D-09 makes the third one
recur in a new form — a Namespace baseline will need recording, which is a *new*
todo, not a reopening of the resolved one.

</deferred>

---

*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Context gathered: 2026-08-01*
