# Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Context

**Gathered:** 2026-07-19
**Status:** Ready for planning
**Mode:** --auto --chain (all gray areas auto-selected; recommended options chosen and logged inline / in 08-DISCUSSION-LOG.md; auto-advancing to plan → execute)

<domain>
## Phase Boundary

The **final v1.0 phase**: close the *surface* gap (flag names + defaults) against
TS CodeGraph 1.3.1, audit the new Charm dependency closure, then cut the first
real signed **`v1.0.0`** and retire v0.1's "not yet drop-in" caveat. Two distinct
bodies of work:

**A. Surface reconciliation (mechanical, ships first) — SURF-01..05**
- **SURF-01** — `impact` default depth **2** (from 5), matching TS.
- **SURF-02** — `files` gains a **directory-path filter** as a NEW flag; the
  existing language `--filter` is **retained** (locked documented divergence:
  keep ours + add TS's).
- **SURF-03** — add missing **short-flag aliases** (`-l`, `-k`, `-j`, `-d`, …)
  across commands to match TS, resolving collisions with existing Go bindings.
- **SURF-04** — `affected` gains `--stdin`, `--depth`, `--filter <glob>`,
  `--quiet` for git-hook/CI scripting.
- **SURF-05** — a **systematic per-command flag audit**: every TS flag name +
  default is present or a documented divergence; `search` documented as a Go-only
  extension, `migrate` documented as an accepted divergence.

**B. Signed release + supply-chain audit (ships after A is green) — REL-01..04**
- **REL-01** — audit the new **Charm/TUI dependency closure**: no *new* CGo,
  `govulncheck` clean, SBOM regenerated, reproducible double-build still passes.
- **REL-02** — cut a real signed **`v1.0.0`** (per-binary cosign keyless + SLSA
  provenance + SBOM), closing v0.1's pending DIST-02.
- **REL-03** — re-run + **publish** head-to-head benchmarks vs TS 1.3.1, closing
  v0.1's pending PERF-01.
- **REL-04** — validate the "drop-in parity" claim against the real TS CLI
  (re-run Phase-1 TEST-01 behavioral harness + SURF-05 flag audit green), then
  **retire the PROJECT.md "not yet drop-in" caveat**.

**Not in this phase (explicit out-of-scope):**
- **Rebuilding** the release pipeline, bench harness, or behavioral harness —
  all three already exist (`.goreleaser.yaml`, `release.yml`, `bench.yml`,
  Phase-1 TEST-01 golden harness). REL-02/03/04 *consume* them; they do not
  rebuild them. REL-04 re-runs the harness green; it does not own it.
- **New behavioral parity** or any `explore`/`node`/`status` algorithm change
  (that was Phases 1–2). This phase is surface (flag) parity only.
- **New interactive/Charm features** beyond Phases 6–7 — the Charm work is
  *audited* here (REL-01), not extended.
- Team-scale / central-server / annotation features (later milestone).

</domain>

<decisions>
## Implementation Decisions

### Order of work (A before B)
- **D-01:** Do **all surface reconciliation (SURF-01..05) first, green, then the
  release (REL-01..04)**. REL-04's drop-in gate re-runs the SURF-05 flag audit,
  so the flags must be final before the release is cut. The signed tag is the
  **last** action of the phase (and of v1.0).
  `[auto] Phase sequencing → Selected: SURF block fully green → then REL block → tag last (recommended)`

### SURF-01 — impact depth default (change locus)
- **D-02:** Change the **default in the shared engine** (the `depth == 0 → 5`
  clamp becomes `→ 2`), NOT just the CLI flag default, so **both the CLI and the
  MCP `impact`/`callers`/`callees` surfaces** inherit depth-2 in the same commit
  — consistent with how every v1.0 behavioral default landed in the shared
  `internal/query` engine (CLI==MCP). Keep the max-50 clamp. Verify the
  golden/MCP corpus reflects the new default (regen if impact appears).
  `[auto] impact depth locus → Selected: change engine default-when-0 (0→2), CLI+MCP together (recommended)`

### SURF-02 — the new `files` directory filter (flag name)
- **D-03:** Add the directory-path filter as **`--dir <glob>`** (directory/path
  glob), keeping the existing **`--filter <language>`** untouched. This is the
  locked "keep ours + add TS's" divergence: TS spells the directory filter
  `--filter`, but that name is already ours for language, so the new flag takes a
  distinct, self-describing name and the divergence is recorded in the SURF-05
  audit doc. Wire `--dir` into the existing `files` options struct (glob match
  against the file's directory), sibling to `--pattern`/`--filter`/`--depth`.
  `[auto] files directory-filter name → Selected: --dir <glob>, keep --filter=language, document divergence (recommended)`

### SURF-03 — short-flag alias policy (collision handling)
- **D-04:** Adopt **TS's short letters per-command wherever the letter is free**;
  where a letter already has an established Go binding, **keep the Go binding and
  record the divergence** rather than silently remapping (a drop-in user's muscle
  memory for the *long* flag always works; short-flag divergences are the
  documented tail). Known live bindings to respect: `-p`=path, `-q`=quiet,
  `-v`=verbose, `-y`=yes, `-f`=file/force, `-l`=line (node). Produce the exact
  per-command letter map as part of the SURF-05 audit; never introduce a
  cobra short-flag collision within one command (build/`go test` catches it).
  `[auto] short-flag collisions → Selected: TS letter where free, keep Go binding + document divergence where taken (recommended)`

### SURF-04 — `affected` scripting flags
- **D-05:** Add `--stdin` (read changed paths from stdin), `--depth` (reuse the
  engine BFS-depth knob, same clamp as `impact`), `--filter <glob>` (glob over
  affected paths), and `--quiet` (suppress the human summary; machine-readable
  path list only) to `affected`. Match TS names/semantics; `--quiet` reuses the
  established `-q` short. Purpose is git-hook/CI scripting, so `--stdin` +
  `--quiet` together must emit a clean, parseable path list with no styling
  (respects the Phase-6 rendering seam — plain when piped).
  `[auto] affected flags → Selected: add --stdin/--depth/--filter/--quiet, TS-parity semantics, plain machine output (recommended)`

### SURF-05 — flag audit + divergence doc
- **D-06:** The systematic audit produces **`docs/FLAG-PARITY.md`** — a
  per-command matrix (TS flag + default → Go flag + default → status:
  *present* / *divergence(reason)* / *Go-only*). This is the single artifact
  REL-04's drop-in gate reads to declare surface parity. It explicitly records:
  the dual `--filter`/`--dir` divergence (D-03), every short-flag divergence
  (D-04), `search` as a Go-only extension, and `migrate` as an accepted
  divergence. Prefer to back it with a **lightweight test** that walks the cobra
  command tree and asserts every registered flag appears in the matrix (so the
  doc can't silently drift) — test enforcement is Claude's discretion if it
  proves heavy.
  `[auto] flag audit artifact → Selected: docs/FLAG-PARITY.md matrix + optional tree-walk drift test (recommended)`

### REL-01 — Charm-closure CGo/vuln/SBOM audit (how to prove "no NEW CGo")
- **D-07:** Prove **no *new* CGo** via a **dependency-closure inspection**
  (`go list -deps`/module graph over the `charm.land/*` closure asserting it
  pulls **zero** cgo/`import "C"` packages), **NOT** a blanket `CGO_ENABLED=0`
  build — the binary already links CGo tree-sitter, so a clean-build assertion
  would fail on *pre-existing* code and prove nothing about the Charm deps. Then
  run the existing gates: `govulncheck ./...` clean, regenerate the SBOM
  (goreleaser Syft block), and confirm the reproducible **double-build** still
  matches. **GOTCHA (carry forward from Phase 7):** `internal/daemon`
  transitively imports CGo tree-sitter, so any Windows cross-`go vet` needs
  `gcc-mingw-w64` + `CGO_ENABLED=1` (NOT zig) — the Charm audit must not
  reintroduce a `CGO_ENABLED=0` expectation anywhere in CI.
  `[auto] no-new-CGo proof → Selected: charm-closure dependency diff (not clean-build), then govulncheck+SBOM+double-build (recommended)`

### REL-02 — release cut sequencing & branch/tag model
- **D-08:** Land all Phase-8 work on the integration branch
  **`gsd/v1.0-drop-in-parity-human-ux`** → **squash-merge to `main`** (the merge
  is where signing happens, per the milestone branch model) → **tag `v1.0.0` on
  `main`** (stable tag, no `-` suffix → full GitHub release + becomes `codegraph
  upgrade`'s "latest"). The **actual `git tag` / push is the maintainer's manual
  action** documented in the runbook (D-11) — the phase produces
  release-*readiness* (green gates + runbook + retired caveat), and the tag push
  triggers the signed `release.yml` build. Never use a `milestone-v*` marker
  name (non-matching → won't fire release). **LOCKED contract:** do not rename
  `release.yml`, change its `v[0-9]*` trigger, or the cosign identity without
  editing `internal/upgrade/verify.go`'s
  `releaseWorkflowRefPattern`/`releaseOIDCIssuer`/`releaseRepoSlug` in lockstep
  (breaks `codegraph upgrade` for every user).
  `[auto] release sequencing → Selected: integration-branch → squash-merge main → tag v1.0.0 on main; tag push is manual/runbook (recommended)`
- **D-09:** **Pre-tag cross-platform gate is mandatory** — run
  `GOOS=<os> GOARCH=<arch> go list -mod=readonly ./...` for **all 6 release
  targets** before tagging (v0.1 `rc.1` failed on a linux-only `go.sum` hash
  invisible on darwin). Add it to the runbook and, if not already, wire it as a
  CI/pre-tag check.

### REL-03 — benchmark re-run (reuse, don't rebuild)
- **D-10:** **Re-run the existing `bench.yml` harness** head-to-head vs the
  pinned TS 1.3.1 checkout, refresh the numbers in **`docs/BENCHMARKS.md`**, and
  surface them in the `v1.0.0` release notes. Reuse the v0.1 methodology
  (median-of-3, same corpora) so the numbers are comparable; do not author a new
  bench framework.
  `[auto] benchmark scope → Selected: re-run existing bench.yml, refresh docs/BENCHMARKS.md + release notes (recommended)`

### REL-04 — drop-in validation + caveat retirement
- **D-11:** The drop-in gate is: **Phase-1 TEST-01 behavioral harness green
  (against real TS 1.3.1) AND `docs/FLAG-PARITY.md` audit green (0 undocumented
  divergences)**. Only when both pass, edit **`.planning/PROJECT.md`** to retire
  the "not yet drop-in" caveat (the caveat text appears in multiple places —
  sweep all of them: milestone goal line, repo-state paragraph, decision-log
  rows). REL-04 consumes the harness; it does not modify it.

### Folded Todos
- **"Document release procedures (maintainer runbook)"** (`docs/RELEASE.md`,
  `.github/workflows/release.yml`, `internal/upgrade/verify.go`; matched Phase 8
  at score 0.6 — folded). Produce **`docs/RELEASE-PROCEDURES.md`** (or a "Cutting
  a release" section) as the REL-02 maintainer runbook, covering: the **pre-tag
  6-target `go list -mod=readonly` check** (D-09), **tag conventions** (rc
  `v0.0.0-rc.N` = prerelease / stable `vX.Y.Z` = full / `milestone-v*` = never
  fires), what the tag push triggers, the **`verify.go` LOCKED contract** (D-08),
  **post-release verification** (`cosign verify-blob --bundle …` +
  `slsa-verifier verify-artifact`), rollback/cleanup of a failed rc tag, and the
  `-c commit.gpgsign=false` fallback for **automated pipeline commits only** (per
  repo rule `xmz3xknbj0` — never bypass signing when a human explicitly asks for
  a commit). This runbook is a REL-02 deliverable.

### Claude's Discretion
- Exact `--dir` glob-match semantics (prefix vs full-glob vs `doublestar`) —
  match TS's directory-filter behavior on the golden corpus.
- The precise per-command short-flag letter map (D-04) and which divergences
  land — driven by what letters are actually free per command.
- Whether SURF-05's audit gets a tree-walk drift **test** or stays doc-only.
- Whether `--depth` on `affected` shares the exact `impact` clamp helper or a
  local copy.
- SBOM tool choice for the REL-01 regen if the goreleaser Syft default needs
  supplementing (cyclonedx-gomod app-mode is the documented higher-fidelity
  option in the stack doc).
- Splitting the phase into a SURF wave and a REL wave vs interleaving (D-01 only
  mandates SURF-green-before-tag, not a hard wave barrier).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope (locked)
- `.planning/ROADMAP.md` §"Phase 8: Surface Reconciliation & Signed v1.0.0
  Release" — goal, 5 success criteria, Notes (mechanical reconciliation first,
  then Charm-dep audit + signed tag; REL-04 consumes the Phase-1 harness).
- `.planning/REQUIREMENTS.md` — SURF-01, SURF-02, SURF-03, SURF-04, SURF-05,
  REL-01, REL-02, REL-03, REL-04 (SURF-06 already Complete in Phase 2; note the
  SURF-02 "keep ours + add TS's `--filter`" divergence decision and the
  `search`/`migrate` divergence notes).
- `.planning/PROJECT.md` — the milestone goal + the **"not yet drop-in" caveat
  text REL-04 retires** (appears at lines ~13, ~72, ~88; sweep all sites).

### Surface reconciliation (SURF-01..05) — command surface
- `internal/cli/impact.go` — `--depth` flag (`IntVar depth,"depth",0`); the
  0→engine-default path SURF-01 changes.
- `internal/cli/files.go` — `--filter`(language), `--pattern`, `--depth`; SURF-02
  adds the new `--dir` directory filter here (options struct `Filter`/`Depth`).
- `internal/cli/affected.go` — the command SURF-04 extends with
  `--stdin`/`--depth`/`--filter`/`--quiet`.
- `internal/cli/*.go` (all commands) — the full flag surface SURF-03 adds short
  aliases to and SURF-05 audits. Existing short bindings to respect: `-p`(path),
  `-q`(quiet), `-v`(verbose), `-y`(yes), `-f`(file/force), `-l`(line, node.go).
- `internal/cli/root.go` — command registration (the tree the SURF-05 audit
  walks; where a drift test would hook).
- `internal/query/` — the shared engine holding the `impact`/BFS depth clamp
  (SURF-01 D-02 changes the default here so CLI+MCP move together); `internal/mcp`
  tool wiring inherits it.
- `internal/cli/search.go`, `internal/cli/migrate.go` — the two commands SURF-05
  documents as Go-only extension / accepted divergence.

### Behavioral drop-in gate (REL-04) — reuse, do not rebuild
- Phase-1 TEST-01 behavioral fixture harness (golden corpus + CLI==MCP parity
  diff vs TS 1.3.1) under `testdata/golden/` — REL-04 re-runs it green.
- `.planning/phases/01-behavioral-parity-explore-node/01-CONTEXT.md` — the
  harness contract REL-04 depends on (do not modify the harness).

### Release + supply chain (REL-01..03) — existing infra
- `.goreleaser.yaml` — the build/sign/SBOM config (Syft block, per-binary cosign,
  reproducible build) REL-01/REL-02 exercise.
- `.github/workflows/release.yml` — the `v[0-9]*`-triggered signed build.
  **LOCKED contract** (see `internal/upgrade/verify.go`).
- `.github/workflows/ci.yml` — where govulncheck / cross-`go vet` gates live
  (Windows cross-vet needs mingw+CGO=1, not zig — REL-01 gotcha).
- `.github/workflows/bench.yml` — the head-to-head bench harness REL-03 re-runs.
- `internal/upgrade/verify.go` — `releaseWorkflowRefPattern` /
  `releaseOIDCIssuer` / `releaseRepoSlug`: the pinned identity that MUST change
  in lockstep with any `release.yml` rename/trigger change (D-08).
- `docs/RELEASE.md` — the *user* verification doc; the *maintainer* runbook
  (`docs/RELEASE-PROCEDURES.md`, folded todo) is the new sibling.
- `docs/BENCHMARKS.md` — the published-numbers artifact REL-03 refreshes.

### Cross-phase constraints carried forward (MUST honor)
- `.planning/phases/06-rendering-seam-pretty-status-files/06-CONTEXT.md` +
  `internal/cli/present/archtest/import_graph_test.go` — the ANSI-isolation
  archtest stays green; `affected --stdin --quiet` machine output must be plain
  (no styling reaches piped/scripted consumers).
- Repo rule `xmz3xknbj0` (engram) — `-c commit.gpgsign=false` allowed for
  agent/pipeline commits ONLY; sign when the user is at the keyboard and asks.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The whole `internal/cli/*.go` cobra tree with an established flag idiom
  (`StringVarP`/`IntVar`/`BoolVarP`, `-p`/`-q`/`-v`/`-y`/`-f` shorts) — SURF-01..04
  are additive edits into existing `cmd.Flags()` blocks, no new command scaffolding.
- `internal/query` shared engine (CLI==MCP) — SURF-01's depth default changes in
  one place and both surfaces inherit it (the v1.0 shared-engine pattern).
- `.goreleaser.yaml` + `release.yml` + `bench.yml` — REL-01/02/03 are *runs* of
  existing, proven pipelines (v0.1 shipped `v0.0.0-rc.3` through them), not new builds.
- Phase-1 golden/behavioral harness under `testdata/golden/` — REL-04's drop-in
  gate re-runs it; `go test ./testdata/golden/...` must be invoked explicitly
  (the go tool skips `testdata/`).
- The Phase-7 release runbook todo — pre-captured pre-tag gotchas (rc.1 linux-only
  go.sum failure, tag conventions, verify.go LOCKED contract) feed the folded runbook.

### Established Patterns
- **Shared-engine defaults (CLI==MCP)** — behavioral defaults live in
  `internal/query`, not per-surface flag defaults (SURF-01).
- **Documented-divergence over silent-remap** — SURF-02/03/05 record every
  intentional TS gap in one matrix rather than forcing exact parity that would
  break an existing Go surface (`--filter`=language, `search`, `migrate`).
- **Plain-when-piped rendering seam** (Phase 6) — `affected` scripting output
  (SURF-04) must respect it; no ANSI in `--stdin --quiet` output.
- **LOCKED release contract** — `release.yml` shape pinned by `verify.go`; change
  in lockstep or break `codegraph upgrade`.
- **Pre-tag 6-target `go list -mod=readonly` sweep** — the v0.1 rc.1 lesson,
  mandatory before any tag (D-09).

### Integration Points
- `internal/cli/impact.go` + `internal/query` depth clamp → SURF-01 default 5→2.
- `internal/cli/files.go` options struct → SURF-02 new `--dir` glob filter.
- `internal/cli/affected.go` `RunE` + flags → SURF-04 `--stdin`/`--depth`/`--filter`/`--quiet`.
- Every `internal/cli/*.go` `cmd.Flags()` block → SURF-03 short aliases.
- `internal/cli/root.go` command tree → SURF-05 audit walk + `docs/FLAG-PARITY.md`.
- `.planning/PROJECT.md` caveat sites → REL-04 retirement edit.
- `.goreleaser.yaml`/`release.yml`/`bench.yml` + `docs/BENCHMARKS.md` +
  `docs/RELEASE-PROCEDURES.md` → REL-01/02/03 + folded runbook.

</code_context>

<specifics>
## Specific Ideas

- Stable release tag is **`v1.0.0`** (no `-` suffix → full GitHub release + the
  `codegraph upgrade` "latest"); rc tags are `v0.0.0-rc.N` (prerelease); internal
  markers use a **non-matching** `milestone-v*` name so they never fire `release.yml`.
- The dual `files` filter: **`--filter` = language (ours, kept)** +
  **`--dir <glob>` = directory (new, = TS's `--filter` semantics)** — the locked
  "keep ours + add TS's" divergence, recorded in `docs/FLAG-PARITY.md`.
- REL-01's "no new CGo" is a **dependency-closure claim about the `charm.land/*`
  packages**, NOT a `CGO_ENABLED=0` clean build (the binary already links CGo
  tree-sitter — a clean-build check would fail on pre-existing code).
- Windows cross-`go vet` needs **`gcc-mingw-w64` + `CGO_ENABLED=1`** (not zig),
  because `internal/daemon` transitively imports CGo tree-sitter (Phase-7 lesson).
- The signed tag is the **very last** action of Phase 8 and of the v1.0
  milestone; everything else (flags, audit, benchmarks, caveat retirement) is
  release-*readiness* and lands before it.

</specifics>

<deferred>
## Deferred Ideas

- **DMON-FUT-01** — true detached / double-forked per-project daemons +
  unix-socket sharing (full TS auto-spawn parity). Later milestone.
- **Backlog 999.1** — contributor `Taskfile.yml` + `CONTRIBUTING.md` (local
  build/test/lint wrappers). Pairs with the folded release runbook but is its own
  backlog item; only pull in if REL work naturally touches it.
- **Backlog 999.2** — tmux real-PTY e2e/UAT harness for the interactive TUI.
  Not a Phase-8 (release) concern; separate backlog.
- **Team-scale / central-server / CI-distributed indexes / annotations
  (embeddings, communities, export) / local Svelte web UI (SEED-001)** — post-v1.0
  milestone, per PROJECT.md.

### Reviewed Todos (not folded)
None — the one matching todo (release maintainer runbook, score 0.6) was folded
into REL-02 (see Folded Todos above).

</deferred>

---

*Phase: 8-Surface Reconciliation & Signed v1.0.0 Release*
*Context gathered: 2026-07-19*
