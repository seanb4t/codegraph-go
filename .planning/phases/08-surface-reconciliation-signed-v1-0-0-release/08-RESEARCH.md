# Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Research

**Researched:** 2026-07-19
**Domain:** cobra CLI flag-surface reconciliation (Go vs. live TS CodeGraph 1.3.1) + existing signed-release/supply-chain pipeline (GoReleaser/cosign/SLSA/govulncheck)
**Confidence:** HIGH

## Summary

This phase has two independent bodies of work, and nearly everything needed to
plan both was obtained by **direct inspection of running code**, not
secondary sources: (1) the actual TS CodeGraph 1.3.1 CLI was installed
locally (`npm install @colbymchenry/codegraph@1.3.1` into a throwaway
prefix — the exact pinned version this project targets) and its real
`--help` output plus its unpacked `bin/codegraph.js` source were read
directly, giving an authoritative, verified flag surface (names, shorts,
defaults, and — critically — the *exact* matching semantics for `files
--filter`/`--pattern` and `affected --filter`/`--depth`); and (2) every
`internal/cli/*.go` command file, the `internal/query` depth-clamp
machinery, and every release/CI/bench workflow file were read to establish
the Go side and the existing supply-chain infrastructure precisely.

**Body A (surface reconciliation)** is smaller than "add 4 short flags"
might suggest. The TS-vs-Go diff surfaces several findings not spelled out
in CONTEXT.md that the planner must account for: `affected` in TS is not a
single-hop lookup — it is a depth-bounded BFS with test-files-as-leaves
semantics, so SURF-04 requires extending `Engine.Affected` from its current
one-hop reverse-adjacency walk into a real bounded BFS, not just wiring a
flag; `impact`'s shared `defaultDepth` constant is used by nothing else in
the codebase today, so flipping it 5→2 is safe, but `affected`'s new
`--depth` needs its **own** default (5, not 2 — TS's own two commands use
different defaults) and therefore cannot reuse `clampDepth`'s baked-in
constant as-is; TS's `files --filter <dir>` is a plain path-prefix
`startsWith` match (not a glob) despite the flag's `<dir>`-suffixed help
text, which directly resolves CONTEXT's open "glob semantics" discretion
item; and there are at least four **additional** undocumented-in-CONTEXT
divergences the SURF-05 audit must record (`upgrade` is missing `-f/--force`
entirely; `files --format` defaults to `"flat"` in Go vs `"tree"` in TS;
`files --depth` is TS's `--max-depth`; `node` has no TS-parity "file mode"
at all — `--offset`/`--limit`/`--symbols-only` don't exist in Go).

**Body B (signed release)** requires almost no new engineering — every gate
already exists and passed once for `v0.0.0-rc.3`. This phase's job is to
*run* them again post-Charm, cut the first **stable** tag, and write the
runbook. The `charm.land/*` "no new CGo" claim was verified in this
research session with a runnable `go list -deps -json` + `jq` one-liner
(§Package Legitimacy Audit) — it returns zero CGo packages in the closure,
confirmed empirically, not just asserted.

**Primary recommendation:** Sequence exactly as CONTEXT D-01 locks (SURF
green → REL green → tag last). For SURF-04, budget real engine work
(`Engine.Affected` BFS extension), not just flag wiring. For REL-01, reuse
the exact `go list -deps -json` + `jq` command below rather than inventing
a new closure-inspection method.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `impact`/`affected` depth default & BFS bound | API/Backend (`internal/query.Engine`) | CLI (`internal/cli`) | CONTEXT D-02: the shared-engine-default pattern established in every prior v1.0 phase — CLI is a thin flag-to-call adapter, MCP inherits automatically where applicable |
| `files --dir`/`--filter` matching semantics | API/Backend (`internal/query.Files`) | CLI (`internal/cli/files.go`) | `FilesOptions` struct already centralizes all filter logic; CLI only parses flags |
| Short-flag aliases (SURF-03) | CLI (`internal/cli/*.go`) | — | Pure cobra `Flags()` registration; no engine change |
| `docs/FLAG-PARITY.md` audit artifact | Docs/build-time test | CLI (`internal/cli/root.go` tree-walk) | A static doc plus an optional archtest-style drift check over `cmd.Commands()`/`Flags().VisitAll` |
| Charm dependency-closure CGo audit | Build/CI (`go list -deps`, CI job) | — | Static analysis over `go.mod`'s module graph; no runtime component |
| Signed release cut | CI/CD (`.goreleaser.yaml`, `release.yml`) | Release/Storage (GitHub Releases, Sigstore transparency log) | Existing, proven pipeline; this phase only triggers and documents it |
| Benchmark re-run/publish | CI/CD (`bench.yml`) | Docs (`docs/BENCHMARKS.md`) | Existing `tools/bench/runner -mode headtohead`; a publish action, not new code |
| Drop-in validation gate | Test/CI (`testdata/golden/...`, `docs/FLAG-PARITY.md`) | Docs (`PROJECT.md` caveat retirement) | Consumes two existing artifacts; the retirement edit is a docs-tier action |

## User Constraints

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 (sequencing):** All SURF-01..05 work lands green FIRST, then all
  REL-01..04 work. REL-04's drop-in gate re-runs the SURF-05 flag audit, so
  flags must be final before the release is cut. The signed tag is the
  **last** action of the phase (and of v1.0).
- **D-02 (impact depth locus):** Change the default in the **shared engine**
  (the `depth == 0 → 5` clamp becomes `→ 2`), not just the CLI flag default,
  so CLI and MCP `impact`/`callers`/`callees` surfaces inherit depth-2
  together. Keep the max-50 clamp. Verify golden/MCP corpus reflects the new
  default (regen if `impact` appears).
- **D-03 (`files` directory filter name):** Add the directory-path filter as
  **`--dir <glob>`**, keeping the existing **`--filter <language>`**
  untouched. This is the locked "keep ours + add TS's" divergence — TS
  spells the directory filter `--filter`, but that name is already ours for
  language, so the new flag takes a distinct name and the divergence is
  recorded in the SURF-05 audit doc.
- **D-04 (short-flag collision policy):** Adopt TS's short letters
  per-command wherever the letter is free; where a letter already has an
  established Go binding, keep the Go binding and record the divergence.
  Known live bindings to respect: `-p`=path, `-q`=quiet, `-v`=verbose,
  `-y`=yes, `-f`=file/force, `-l`=line (node). Produce the exact per-command
  letter map as part of the SURF-05 audit; never introduce a cobra
  short-flag collision within one command.
- **D-05 (`affected` scripting flags):** Add `--stdin`, `--depth` (reuse the
  engine BFS-depth knob, same clamp as `impact`), `--filter <glob>`, and
  `--quiet` to `affected`. Match TS names/semantics; `--quiet` reuses the
  established `-q` short. `--stdin` + `--quiet` together must emit a clean,
  parseable path list with no styling (Phase-6 rendering seam).
- **D-06 (flag audit artifact):** The systematic audit produces
  **`docs/FLAG-PARITY.md`** — a per-command matrix (TS flag+default → Go
  flag+default → status: present/divergence(reason)/Go-only). Records the
  dual `--filter`/`--dir` divergence, every short-flag divergence, `search`
  as Go-only, `migrate` as accepted divergence. A lightweight tree-walk
  drift test is preferred but Claude's discretion if heavy.
- **D-07 (no-new-CGo proof):** Prove via a **dependency-closure inspection**
  (`go list -deps` over the `charm.land/*` closure asserting zero cgo/
  `import "C"` packages), **NOT** a blanket `CGO_ENABLED=0` build (the
  binary already links CGo tree-sitter). Then run existing gates:
  `govulncheck ./...` clean, regenerate SBOM (goreleaser Syft block),
  confirm reproducible double-build still matches. **Carry-forward gotcha:**
  `internal/daemon` transitively imports CGo tree-sitter, so Windows
  cross-`go vet` needs `gcc-mingw-w64` + `CGO_ENABLED=1` (NOT zig) — must
  not reintroduce a `CGO_ENABLED=0` expectation anywhere in CI.
- **D-08 (release sequencing & branch/tag model):** Land Phase-8 work on
  `gsd/v1.0-drop-in-parity-human-ux` → squash-merge to `main` → tag `v1.0.0`
  on `main` (stable, no suffix → full release + `codegraph upgrade`
  "latest"). The actual `git tag`/push is the **maintainer's manual action**
  documented in the runbook — this phase produces release-*readiness*.
  Never use a `milestone-v*` marker (non-matching → won't fire release).
  **LOCKED contract:** do not rename `release.yml`, change its `v[0-9]*`
  trigger, or the cosign identity without editing `internal/upgrade/verify.go`'s
  `releaseWorkflowRefPattern`/`releaseOIDCIssuer`/`releaseRepoSlug` in
  lockstep.
- **D-09 (pre-tag gate):** Run `GOOS=<os> GOARCH=<arch> go list -mod=readonly
  ./...` for all 6 release targets before tagging (the v0.1 `rc.1`
  linux-only `go.sum` lesson). Add to the runbook and wire as a CI/pre-tag
  check if not already present.
- **D-10 (benchmark re-run):** Re-run the existing `bench.yml` harness
  head-to-head vs. pinned TS 1.3.1, refresh `docs/BENCHMARKS.md`, surface in
  `v1.0.0` release notes. Reuse v0.1 methodology (median-of-3, same
  corpora); do not author a new bench framework.
- **D-11 (drop-in gate):** Phase-1 TEST-01 behavioral harness green (against
  real TS 1.3.1) AND `docs/FLAG-PARITY.md` audit green (0 undocumented
  divergences). Only then edit `.planning/PROJECT.md` to retire the "not yet
  drop-in" caveat — sweep ALL occurrences. REL-04 consumes the harness; does
  not modify it.
- **Folded todo:** Produce `docs/RELEASE-PROCEDURES.md` (or a "Cutting a
  release" section) as the REL-02 maintainer runbook, covering: pre-tag
  6-target `go list -mod=readonly` check, tag conventions, what the tag push
  triggers, the `verify.go` LOCKED contract, post-release verification
  (`cosign verify-blob`/`slsa-verifier`), rollback/cleanup of a failed rc
  tag, and the `-c commit.gpgsign=false` fallback for automated pipeline
  commits only.

### Claude's Discretion

- Exact `--dir` glob-match semantics (prefix vs full-glob vs `doublestar`) —
  **RESOLVED by this research**: TS's own `--filter <dir>` implementation
  is a plain path-prefix `startsWith` match, not a glob at all (see
  §Code Examples, `files --filter`/`--dir`). Recommend matching this
  literally: `strings.HasPrefix(path, dir) || strings.HasPrefix(path,
  "./"+dir)`, reusing no new dependency.
- The precise per-command short-flag letter map (D-04) — see
  §Standard Stack / SURF-03 flag map table below; driven by what letters
  are actually free per command (verified empirically).
- Whether SURF-05's audit gets a tree-walk drift **test** or stays doc-only
  — recommend a lightweight test (see §Architecture Patterns).
- Whether `--depth` on `affected` shares the exact `impact` clamp helper or
  a local copy — **RESOLVED by this research**: it must NOT share
  `clampDepth`/`defaultDepth` unmodified, because TS's two defaults differ
  (impact=2, affected=5) and Go's `defaultDepth` constant is being changed
  to 2 for impact. Recommend a local `defaultAffectedDepth = 5` constant
  reusing the same `MaxDepth=50` ceiling and the same clamp *shape* (see
  §Common Pitfalls).
- SBOM tool choice for REL-01 regen — the existing goreleaser Syft block is
  sufficient; no evidence in this session that supplementing with
  cyclonedx-gomod is needed (no build-constraint-sensitive dependency
  divergence observed across the 6 targets).
- Splitting the phase into a SURF wave and a REL wave vs. interleaving — D-01
  only mandates SURF-green-before-tag, not a hard wave barrier; recommend a
  SURF wave then a REL wave for a clean gate, matching D-01's spirit.

### Deferred Ideas (OUT OF SCOPE)

- **DMON-FUT-01** — true detached/double-forked per-project daemons + unix-
  socket sharing. Later milestone.
- **Backlog 999.1** — contributor `Taskfile.yml` + `CONTRIBUTING.md`. Only
  pull in if REL work naturally touches it.
- **Backlog 999.2** — tmux real-PTY e2e/UAT harness. Separate backlog.
- Team-scale/central-server/CI-distributed indexes/annotations/local Svelte
  web UI — post-v1.0 milestone.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SURF-01 | `impact` default depth 2 (from 5) | §Common Pitfalls Pitfall 1; exact clamp site `internal/query/validate.go:34,44-55`, sole consumer `internal/query/traverse.go:399` — safe, isolated change. TS confirmed default 2, clamp [1,10] (Go keeps [1,50] per D-02). |
| SURF-02 | `files` gains directory filter as NEW `--dir`, language `--filter` retained | §Code Examples "TS `files --filter`"; resolves glob-semantics discretion (prefix match, not glob) |
| SURF-03 | Missing short-flag aliases (`-l`/`-k`/`-j`/`-d`, etc.) added | §Standard Stack SURF-03 table — full per-command letter map, verified free/taken against real cobra `Flags()` registrations |
| SURF-04 | `affected` gains `--stdin`/`--depth`/`--filter <glob>`/`--quiet` | §Common Pitfalls Pitfall 2 — requires extending `Engine.Affected` to a real depth-bounded BFS, not just flag wiring; exact TS BFS/leaf-pruning/quiet/json semantics captured in §Code Examples |
| SURF-05 | Systematic per-command flag audit; `search`/`migrate` documented | §Standard Stack full table (every command); §Common Pitfalls flags 4 additional undocumented divergences (`upgrade` missing `--force`, `files --format` default mismatch, `files --depth` vs `--max-depth` naming, `node` missing TS file-mode) |
| REL-01 | Charm dependency closure audited: no new CGo, govulncheck clean, SBOM regen, double-build passes | §Package Legitimacy Audit — runnable, already-executed `go list -deps -json`+`jq` command returning 0 cgo packages in the charm.land closure; existing govulncheck/SBOM/double-build gates identified verbatim |
| REL-02 | Signed `v1.0.0` cut, closing DIST-02 | §Architecture Patterns "Release cut sequence"; exact LOCKED constants from `internal/upgrade/verify.go` reproduced verbatim |
| REL-03 | Benchmarks re-run + published, closing PERF-01 | §Code Examples "Re-running the head-to-head bench"; existing `bench.yml`/`docs/BENCHMARKS.md` methodology reproduced |
| REL-04 | Drop-in parity validated against real TS CLI; caveat retired | §Common Pitfalls Pitfall 6; exact PROJECT.md caveat locations enumerated below |
</phase_requirements>

## Standard Stack

No new libraries are introduced by this phase. Body A is pure `spf13/cobra`
flag registration + `internal/query` engine edits. Body B reuses existing,
already-proven infrastructure (GoReleaser v2, cosign v3, syft, SLSA generic
generator, govulncheck) with zero new dependencies.

### Core (existing, reused)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 (`go.mod`, `[VERIFIED: go.mod]`) | CLI framework | Already the project's CLI framework since v0.1; SURF-01..05 are additive edits to existing `Flags()` blocks |
| `charm.land/lipgloss/v2` | v2.0.5 (`go.mod`) | TTY styling (Phase 6/7) | Being AUDITED, not extended, in REL-01 |
| `charm.land/bubbletea/v2` | v2.0.8 (`go.mod`) | Interactive TUI (Phase 7) | Same |
| `charm.land/bubbles/v2` | v2.1.1 (`go.mod`) | TUI components (Phase 7) | Same |
| `golang.org/x/vuln/cmd/govulncheck` | v1.6.0 (`[VERIFIED: govulncheck --version]`, run locally this session) | Call-graph-aware vuln scan | Already wired in `ci.yml`'s `govulncheck` job |
| GoReleaser | v2.17.0 (pinned in `release.yml`'s `GORELEASER_VERSION`) | Build config | Already the release build tool |
| `sigstore/cosign` | v3 CLI (installed via `sigstore/cosign-installer` action, pinned SHA) | Keyless signing | Already wired in `release.yml`'s `assemble` job |
| `slsa-framework/slsa-github-generator` | `generator_generic_slsa3.yml@v2.1.0` | SLSA3 provenance | Already wired in `release.yml`'s `provenance` job |
| `anchore/syft` (via `sbom-action/download-syft`) | pinned action SHA | SBOM (SPDX) | Already wired per-binary in `assemble` job |

### Alternatives Considered

Not applicable — no new dependency decisions in this phase. The one
discretion item (SBOM tool for REL-01) is resolved above: reuse the existing
Syft block; no evidence a supplement is needed.

**Installation:** None — no `go get`/`npm install` required for this
phase's own deliverables. (The TS reference CLI, `@colbymchenry/codegraph@1.3.1`,
is already installed by `bench.yml`'s `headtohead` job via `npm install -g`;
this research additionally verified that installing it locally into a
throwaway prefix — `npm install @colbymchenry/codegraph@1.3.1 --no-save`
— is a safe, reproducible way to obtain the live `--help` oracle for the
SURF-05 audit without polluting the dev machine's global npm scope.)

### SURF-03 full per-command flag map (verified against live TS 1.3.1 `--help` + current Go `cmd.Flags()` calls)

`[VERIFIED: live TS 1.3.1 install --help output]` for the TS column;
`[VERIFIED: internal/cli/*.go source read this session]` for the Go column.

| Command | TS flags (name/short/default) | Go flags today (name/short/default) | SURF-03/05 action |
|---|---|---|---|
| `init` | `-i/--index`(deprecated) `-f/--force`(home/root guard) `-v/--verbose` | `-q/--quiet` `-v/--verbose` `--workers` | No TS `--force` equivalent exists in Go at all (different semantic than Go's `index -f`/`uninit -f` "skip confirm" — TS's is a home-dir/fs-root safety guard). **Document as accepted divergence** — adding a new safety-guard behavior is out of SURF's "flag name/default" scope. `-i` deprecated flag: Go-only omission, document as accepted (TS itself calls it deprecated). |
| `uninit` | `-f/--force`(skip confirm) | `-f/--force`(skip confirm) | Already matches — same letter, same semantic. No action. |
| `index` | `-f/--force`(home/root guard) `-q/--quiet` `-v/--verbose` | `-f/--force`(rebuild-without-prompt) `-q/--quiet` `-v/--verbose` `--workers` | Letter matches but **semantic diverges** (Go's `-f` = skip-rebuild-confirmation; TS's `-f` = bypass home/root safety check). Document as accepted divergence, same reasoning as `init`. |
| `sync` | `-q/--quiet` | `-q/--quiet` `-v/--verbose` `--workers` | Matches; Go's extra `-v`/`--workers` are Go-only additions (document as Go extensions, not divergences — additive, no TS name taken). |
| `status` | `-j/--json` | `--json`(no short) | **Add `-j` short.** Free (only `-p` used). |
| `query` | `-p` `-l/--limit`(10) `-k/--kind` `-j/--json` | `-p` `--kind`(no short) `--limit`(no short,0) `--json`(no short) | **Add `-l`, `-k`, `-j` shorts.** All free. Note default value gap too: Go's `--limit` default is `0` (meaning "engine applies its own default downstream"); confirm the effective default matches TS's 10 after clamping — flag for the plan's verification step. |
| `search` (Go-only, no TS command) | — | `-p` `--kind`(no short) `--limit`(no short) `--json`(no short) | No TS parity target — document in `docs/FLAG-PARITY.md` as "Go-only extension" per D-06/CONTEXT. Optionally add `-k`/`-l`/`-j` for internal consistency with `query` (Claude's discretion, not required by SURF-03). |
| `explore` | `-p` `--max-files`(no short) | `-p` `--max-files`(no short, default 0→5) | Matches — no short-flag gap. |
| `node` | `-p` `-f/--file` `--offset`(no short, file-mode) `--limit`(no short, file-mode) `--symbols-only`(no short, file-mode) | `-p` `-f/--file` `-l/--line`(Go-only NODE-03, no short collision since different command) | **No TS file-mode exists in Go at all** (reading a file with line numbers + dependents when `name` is omitted). This is bigger than a flag/default gap — see Pitfall 5. Recommend documenting as an accepted divergence for v1.0 rather than implementing file-mode in Phase 8 (out of stated SURF scope), unless the planner chooses to size it in. |
| `files` | `-p` `--filter <dir>`(prefix match) `--pattern <glob>` `--format`(default `"tree"`) `--max-depth` `--no-metadata` `-j/--json` | `-p` `--pattern` `--filter`(=language) `--depth`(default 0→unlimited) `--format`(default `""`→flat) `--json`(no short) | **Add `-j` short** (free). SURF-02 adds `--dir <glob>` per D-03 (prefix-match semantics, see Code Examples). Document 3 additional divergences: (a) Go's `--depth` vs. TS's `--max-depth` naming — recommend keeping Go's established `--depth` name and documenting the naming divergence rather than a breaking rename; (b) Go's `--format` default `"flat"` vs. TS's default `"tree"` — flag for explicit decision (silently changing the default could surprise existing Go users; recommend documenting as accepted divergence, not changing); (c) TS's `--no-metadata` has no Go equivalent — document as a TS-only flag not yet ported (out of this phase's explicit scope; SURF-05's job is to record it, not necessarily implement it). |
| `daemon`/`daemons` | (no flags besides `-h`) | `-p` on the bare/list path; `start`/`stop` subcommands have their own `-q`/`-v`/`--workers`/`--all` | TS's `daemon` is flag-less (interactive picker only). Go's richer subcommand surface (DMON-01..04, already shipped in Phase 7) is a **documented Go-only extension**, not a divergence to reconcile — no TS flag to match against. |
| `unlock` | (no flags) | (no flags) | Matches. |
| `callers` | `-p` `-l/--limit`(20) `-j/--json` | `-p` `--limit`(no short,0) `--json`(no short) | **Add `-l`, `-j` shorts.** Both free. |
| `callees` | `-p` `-l/--limit`(20) `-j/--json` | `-p` `--limit`(no short,0) `--json`(no short) | **Add `-l`, `-j` shorts.** Both free. |
| `impact` | `-p` `-d/--depth`(2, clamp [1,10]) `-j/--json` | `-p` `--depth`(no short, engine default 5→2 per D-02, clamp [1,50] per D-02) | **Add `-d`, `-j` shorts.** Both free. Depth default fixed by SURF-01 (engine-level). Max-clamp intentionally stays 50, not TS's 10 (D-02 explicit). |
| `affected` | `-p` `--stdin` `-d/--depth`(5) `-f/--filter <glob>` `-j/--json` `-q/--quiet` | `-p` `--json`(no short); NO `--stdin`/`--depth`/`--filter`/`--quiet` exist yet | **Whole SURF-04 surface to add.** All 4 new flags' shorts are free in this command (`-d`,`-f`,`-j`,`-q` all unused today) — no collision to document, straightforward additions once the engine BFS extension (Pitfall 2) lands. Also: current `cobra.MinimumNArgs(1)` must relax to allow 0 positional args when `--stdin` is set. |
| `install` | `-t/--target` `-l/--location` `-y/--yes` `--no-permissions` `--print-config <id>` | `--target`(no short) `--location`(no short) `--auto-allow`(no short, Go-only) `-y/--yes` `--print-config`(check exact name) | **Add `-t`, `-l` shorts.** Both free (only `-y` used today). Note: TS has NO separate `--auto-allow` flag — permissions are written by default unless `--no-permissions` is passed; Go instead requires an explicit opt-in `--auto-allow`. This is a real **behavioral** divergence beyond naming (see Pitfall 4) — flag for explicit decision, do not silently flip Go's security-conservative default. |
| `uninstall` | `-t/--target` `-l/--location` `-y/--yes` | `--target`(no short) `--location`(no short) `-y/--yes` | **Add `-t`, `-l` shorts.** Both free. |
| `telemetry` | (no flags) | (no flags) | Matches. |
| `upgrade` | `--check` `-f/--force` | `--check`(no short) — **no `--force` at all** | **Add missing `-f/--force` flag entirely** (not just a short-alias gap — the whole flag is absent in Go). Free letter (only `--check` bound). |
| `version` | (no flags; `-v/--version` at root) | `--json`(no short, Go-only) | Go's `--json` is an accepted Go-only extension (documented). |
| `serve` (hidden in both) | `-p` `--mcp` `--no-watch` | `-p` `--mcp` `--watch`(Go-only force-on) `--no-watch` | Matches; Go's `--watch` is a documented Go-only addition (Phase 3). |
| `migrate` (Go-only, no TS command) | — | `--from` `--to` `-f/--force` `--drop-dangling` | Document in `docs/FLAG-PARITY.md` as "accepted divergence" per D-06/CONTEXT (TS has no migration tool). |
| `githooks` (Go-only, no TS command) | — | (subcommands, no top-level flags observed) | Document as Go-only extension (Phase 5) — no TS equivalent to reconcile against. |

## Package Legitimacy Audit

No new external packages are introduced by this phase. REL-01's job is to
**audit an existing closure**, not add one — but D-07 requires proving the
"no new CGo" claim with a runnable check, so this section documents the
exact command (run and verified during this research session) rather than
a package table.

**Verified command (`[VERIFIED: ran locally this session]`), reusable
as-is in a plan/CI step:**

```sh
go list -deps -json ./internal/cli/... | jq -s '
  [.[] | select(.ImportPath | startswith("charm.land"))] as $charm
  | ($charm | map(.Deps // []) | flatten | unique) as $closure
  | {
      charm_pkg_count: ($charm | length),
      charm_direct_cgo: [$charm[] | select(.CgoFiles != null) | .ImportPath],
      closure_size: ($closure | length),
      cgo_in_closure: [ .[] | select(.CgoFiles != null and (.ImportPath as $p | $closure | index($p) != null)) | .ImportPath ]
    }
'
```

**Actual output when run against this repo's current HEAD:**

```json
{
  "charm_pkg_count": 10,
  "charm_direct_cgo": [],
  "closure_size": 123,
  "cgo_in_closure": []
}
```

10 `charm.land/*` packages are reachable from `internal/cli` (lipgloss,
bubbletea, bubbles + their `key`/`help`/`paginator`/`spinner`/`cursor`/
`internal/runeutil`/`textinput`/`list` subpackages), with a 123-package
transitive closure, and **zero** of them have `CgoFiles` — proving D-07's
"no new CGo" claim empirically rather than by assertion. Compare this
against the codebase's pre-existing, already-documented CGo exception
(tree-sitter, reachable via `internal/indexer`/`internal/daemon`, NOT via
`internal/cli`'s charm imports) — the two closures are disjoint at the
package level, which is exactly what D-07's "closure diff, not a clean
build" framing predicts.

**govulncheck:** already installed locally (`govulncheck@v1.6.0`,
`[VERIFIED: govulncheck --version]`) and already wired as its own blocking
CI job (`ci.yml`'s `govulncheck` job, `golang/govulncheck-action@032d...`
pinned SHA `v1.1.0`). REL-01 is a **re-run**, not new wiring — `go run
golang.org/x/vuln/cmd/govulncheck@latest ./...` (or the pinned local
binary) is sufficient locally; CI already gates on every push/PR.

**SBOM:** already wired per-binary in `release.yml`'s `assemble` job via
`anchore/sbom-action/download-syft` + `syft "$f" -o spdx-json=...`. REL-01
is a re-run at tag time, not new config.

**Reproducible double-build:** already wired and blocking in `ci.yml`'s
`reproducibility` job (linux/amd64 hard-blocks; linux/arm64 reported only,
non-blocking; windows/darwin not yet independently double-built, per
`docs/RELEASE.md` §3's explicit scoping). REL-01 does not need to widen this
scope — it only needs to confirm the existing gate still passes post-Charm.

**Packages removed due to [SLOP] verdict:** none — no new packages.
**Packages flagged as suspicious [SUS]:** none — no new packages.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │  Live TS CodeGraph 1.3.1 (npm-installed) │
                    │  --help output + bin/codegraph.js source │
                    └───────────────────┬───────────────────────┘
                                        │ (oracle — read-only, this research)
                                        ▼
   ┌────────────────────────────────────────────────────────────────┐
   │  docs/FLAG-PARITY.md  (SURF-05 audit artifact, D-06)             │
   │  per-command: TS flag+default → Go flag+default → status        │
   └───────────────┬───────────────────────────────┬────────────────┘
                    │ drives                        │ read by (gate)
                    ▼                                ▼
   ┌─────────────────────────────┐      ┌─────────────────────────────┐
   │ internal/cli/*.go            │      │ REL-04 drop-in gate          │
   │ (SURF-01..04 flag/behavior    │      │ (re-runs, does not rebuild): │
   │  edits: impact -d/-j, files   │      │  - testdata/golden/... green │
   │  --dir, affected --stdin/     │      │  - docs/FLAG-PARITY.md green │
   │  --depth/--filter/--quiet,    │      │  - PROJECT.md caveat retired │
   │  short-flag adds across all)  │      └──────────────┬──────────────┘
   └───────────────┬───────────────┘                     │
                    │ calls (unchanged signatures          │
                    │ except Engine.Affected extension)    │
                    ▼                                       │
   ┌─────────────────────────────┐                          │
   │ internal/query.Engine         │                          │
   │ - clampDepth/defaultDepth=2   │                          │
   │   (SURF-01, sole consumer:    │                          │
   │    Impact())                  │                          │
   │ - NEW: Affected() BFS w/      │                          │
   │   depth param + own default=5 │                          │
   └───────────────────────────────┘                          │
                                                               │
   ┌──────────────────────────────────────────────────────────┘
   │
   ▼
┌───────────────────────────────┐   ┌──────────────────────────────────┐
│ REL-01 audit                    │   │ REL-02/03 release cut              │
│ go list -deps -json | jq        │──▶│ squash-merge → tag v1.0.0 (manual) │
│ (charm.land closure, 0 CGo) +   │   │ → release.yml build/sign/SBOM/     │
│ govulncheck + SBOM + double-    │   │   SLSA (existing, unchanged) →     │
│ build (all pre-existing gates)  │   │ bench.yml re-run → BENCHMARKS.md   │
└───────────────────────────────┘   └──────────────────────────────────┘
```

### Recommended Project Structure

No new packages/directories. Changes land in-place:

```
internal/cli/          # impact.go, files.go, affected.go, upgrade.go, +
                        # every other command file (short-flag adds)
internal/query/         # validate.go (defaultDepth 5->2), traverse.go
                        # (Affected() BFS extension + new depth param)
docs/
├── FLAG-PARITY.md      # NEW — SURF-05 audit matrix (D-06)
├── RELEASE-PROCEDURES.md  # NEW — REL-02 maintainer runbook (folded todo)
└── BENCHMARKS.md       # REFRESHED — REL-03 re-run numbers
.planning/PROJECT.md    # EDITED — REL-04 caveat retirement (3 sites, see below)
```

### Pattern 1: Shared-engine default, single consumer (SURF-01)

**What:** `defaultDepth` in `internal/query/validate.go` is used by exactly
one function, `clampDepth`, which is used by exactly one caller, `Impact()`.
No MCP-only or CLI-only path bypasses it — `internal/mcp`'s `codegraph_impact`
tool calls the same `Engine.Impact`.
**When to use:** Any behavioral default that must move together across
CLI+MCP (the established v1.0 pattern from every prior phase).
**Example:**
```go
// Source: internal/query/validate.go:34,44-55 [VERIFIED: read this session]
// defaultDepth is applied when a caller passes a non-positive depth.
const defaultDepth = 5   // change to 2 for SURF-01

func clampDepth(n int) int {
	if n <= 0 {
		n = defaultDepth
	}
	if n > MaxDepth { // MaxDepth = 50, unchanged per D-02
		return MaxDepth
	}
	return n
}
```
Changing `defaultDepth` here is a one-line, isolated fix — confirmed via
`rg -n "clampDepth\(" internal/query/*.go`, which returns only
`traverse.go:399` (the real call site) and `engine_test.go` (unit tests of
`clampDepth` itself).

### Pattern 2: Per-command flag maps are independent (no cross-command collision)

**What:** cobra `Flags()` are scoped per `*cobra.Command`; `node`'s `-l`
(=`--line`, Go-only NODE-03) and `query`/`callers`/`callees`'s new `-l`
(=`--limit`, TS parity) never collide because they are different `Command`
objects with independent `FlagSet`s.
**When to use:** Whenever evaluating whether adding a TS short letter to one
command is blocked by that letter's use in a *different* command — it
never is. Collisions only matter **within** a single command's own flag
set.
**Example:** Verified empirically — every one of the 8 commands needing new
shorts in the SURF-03 table above has that letter completely unused in its
own `Flags()` block today (confirmed via `rg -n 'Flags\(\)\.' internal/cli/<cmd>.go`
per command, cross-checked against the live TS `--help`).

### Pattern 3: Release cut sequence (REL-02) — reuse, do not rebuild

**What:** the exact sequence, reproduced verbatim from D-08/D-09 plus the
folded runbook todo:

1. Land all SURF-01..05 + REL-01/03/04 work on
   `gsd/v1.0-drop-in-parity-human-ux`.
2. **Pre-tag gate (mandatory, D-09):** for all 6 release targets —
   `linux/{amd64,arm64}`, `windows/{amd64,arm64}`, `darwin/{amd64,arm64}` —
   run:
   ```sh
   GOOS=<os> GOARCH=<arch> go list -mod=readonly ./...
   ```
   This is the exact check that would have caught the v0.1 `rc.1` failure
   (a linux-only `go.sum` hash for `prometheus/procfs`, invisible on a
   darwin-only dev machine).
3. Squash-merge the integration branch to `main`.
4. **Maintainer-manual action:** `git tag v1.0.0 && git push origin v1.0.0`
   (stable tag, no `-` suffix). This triggers `release.yml`'s `on: push:
   tags: v[0-9]*` matcher.
5. `release.yml`'s `build` job compiles all 6 targets natively/via zig,
   `assemble` job signs each binary individually with cosign keyless
   (`cosign sign-blob --bundle="${f}.sigstore.json"`), generates a per-binary
   SPDX SBOM via syft, publishes the GitHub release; `provenance` job runs
   SLSA3 over the checksums file.
6. **Post-release verification (from `docs/RELEASE.md`, already written)**:
   ```sh
   cosign verify-blob \
     --bundle codegraph_v1.0.0_<goos>_<goarch>.sigstore.json \
     --certificate-oidc-issuer https://token.actions.githubusercontent.com \
     --certificate-identity-regexp \
       '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
     codegraph_v1.0.0_<goos>_<goarch>
   ```

**LOCKED constants (verbatim from `internal/upgrade/verify.go:42-45`,
`[VERIFIED: read this session]`)** — the runbook MUST cite these exactly:

```go
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
)
```

Any rename of `.github/workflows/release.yml`, change to its `v[0-9]*`
trigger, or change of signing identity **must** update these three
constants in the same commit, or `codegraph upgrade` breaks for every user.

### Anti-Patterns to Avoid

- **Asserting `CGO_ENABLED=0` builds clean as the REL-01 "no new CGo"
  proof:** this is explicitly wrong per D-07 — the binary already links
  CGo tree-sitter (`internal/indexer`/`internal/daemon`), so a clean-0
  build fails on pre-existing code and proves nothing about the Charm
  closure specifically. Use the dependency-closure diff instead (see
  §Package Legitimacy Audit).
- **Reusing `clampDepth`/`defaultDepth` unmodified for `affected`'s new
  `--depth`:** TS's own two commands (`impact` default 2, `affected`
  default 5) prove the defaults are NOT meant to be shared. A naive "just
  call `clampDepth`" implementation for `affected` will silently apply the
  wrong default (2 instead of 5) once SURF-01 lands.
- **Treating `files --filter <dir>`'s help text (`<dir>`, docs say
  "directory") as license to implement a real glob:** the live TS source
  (`bin/codegraph.js:1349-1351`) is a plain `startsWith` prefix check, not
  `picomatch`/any glob library, despite `--pattern` (a genuinely different
  flag) using a hand-rolled glob-to-regex conversion. Implementing `--dir`
  as a full glob would silently diverge from TS's actual (simpler)
  behavior, defeating the parity goal.
- **Renaming `release.yml` or its trigger "for clarity" during the Charm
  audit:** this is the single highest-blast-radius action available in
  this phase — it breaks `codegraph upgrade` for every existing user unless
  `verify.go`'s three constants change in lockstep (D-08 LOCKED contract).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Directory-prefix filtering (`--dir`) | A new glob-matching abstraction or `doublestar` dependency | `strings.HasPrefix` (matching TS's actual `bin/codegraph.js` implementation exactly) | TS's own implementation is a prefix check, not a glob — matching it precisely both satisfies parity AND avoids a new dependency; the repo has zero `doublestar`/glob-library dependencies today (`rg doublestar go.mod` → no hits) |
| CGo-closure detection | A custom AST/import scanner | `go list -deps -json` + `jq` (verified command in §Package Legitimacy Audit) | `go list` already computes the exact, authoritative transitive import graph including `CgoFiles`; this is the standard Go toolchain mechanism for exactly this question |
| Release signing/SBOM/provenance | Any new signing or attestation flow | The existing `release.yml` `assemble`/`provenance` jobs (cosign v3, syft, slsa-github-generator) | Already proven end-to-end on `v0.0.0-rc.3`; REL-01/02 only re-run them |
| Benchmark harness | A new bench runner or methodology | `tools/bench/runner -mode headtohead` (existing, D-10 locked) | Already produces comparable, medianed, provenance-tracked numbers against the same pinned TS 1.3.1 + real-repo corpora |
| TS flag-surface ground truth | Reading TS's GitHub source blindly (may lag the installed npm package, or require guessing the exact installed version) | `npm install @colbymchenry/codegraph@1.3.1 --no-save` into a throwaway prefix, then `<bin> <cmd> --help` and reading the unpacked `bin/codegraph.js` | This project's own `bench.yml` already does exactly this (`npm install -g @colbymchenry/codegraph@1.3.1`) — reusing the identical pinned version guarantees the SURF-05 audit and the benchmark numbers are checked against the *same* TS build |

**Key insight:** Every "don't hand-roll" item in this phase resolves to
*"the tool/pipeline you need already exists in this repo or the Go
toolchain — the work is invocation and documentation, not construction."*
This is a mechanical-reconciliation-plus-release-cut phase, not a
greenfield one.

## Common Pitfalls

### Pitfall 1: SURF-01 looks like a one-line change but has a hidden shared-consumer risk if not verified
**What goes wrong:** A less careful audit might assume `defaultDepth` is
consumed by multiple query paths (query/search/callers/callees also expose
depth-like flags) and hesitate to change it, or conversely change it without
checking for other consumers and later discover a silent behavior shift
elsewhere.
**Why it happens:** The constant's name (`defaultDepth`) doesn't scope
itself to `Impact` in the type system.
**How to avoid:** This research already ran `rg -n "clampDepth\("
internal/query/*.go` and confirmed the ONLY non-test call site is
`traverse.go:399` inside `Impact()`. Re-verify this grep at execution time
(one command, near-zero cost) before changing the constant, as a guard
against drift between research and execution.
**Warning signs:** A new function added between now and execution that
calls `clampDepth` without the author realizing it now defaults to 2.

### Pitfall 2: SURF-04's `--depth` is not "just a flag" — `Engine.Affected` needs a real BFS
**What goes wrong:** `internal/query/traverse.go`'s current `Affected(files
[]string)` does exactly ONE hop of reverse-adjacency lookup per seed symbol
(`for _, edge := range rev[id]`) with no depth loop at all — unlike
`Impact`, which already has a `for d := 0; d < depth && ...` BFS loop. A
plan that treats SURF-04 as "add 4 cobra flags to affected.go" will
under-scope the work; `--depth` genuinely changes `Affected`'s traversal
shape, not just its CLI surface.
**Why it happens:** `affected.go`'s current single-hop behavior was
sufficient for v0.1's scope (D-07's "no persisted test-coverage edge,
derive at query time") and was never revisited against TS's actual
multi-hop semantics.
**How to avoid:** Port TS's exact semantics (verified from
`bin/codegraph.js:1900-1960`, `[VERIFIED: TS 1.3.1 dist]`):
```javascript
// Source: bin/codegraph.js:~1934-1955 [VERIFIED: TS 1.3.1 dist]
// BFS through dependents; TEST FILES ARE LEAVES — a file classified as a
// test file is added to affectedTests and NOT expanded further; only
// non-test dependents get queued for the next depth level.
const queue = [{ file, depth: 0 }];
const visited = new Set([file]);
while (queue.length > 0) {
  const current = queue.shift();
  if (current.depth >= maxDepth) continue;
  const dependents = cg.getFileDependents(current.file);
  for (const dep of dependents) {
    if (visited.has(dep)) continue;
    visited.add(dep);
    if (isTestFile(dep)) {
      affectedTests.add(dep);            // leaf — stop here
    } else {
      queue.push({ file: dep, depth: current.depth + 1 }); // keep expanding
    }
  }
}
```
Go's `Impact()` already has the right BFS *shape* (frontier/next-frontier
loop) to crib from — `Affected` needs the same shape plus the
test-symbol-is-a-leaf pruning rule (do not expand past a node once
`isTestSymbol` is true), which `Impact` does not need since it has no
"leaf" concept.
**Divergence note (flag for SURF-05, not a bug to fix):** TS's `isTestFile`
default heuristic is FILE-PATH based (`.spec.`, `.test.`, `/__tests__/`,
`/tests?/`, `/e2e/`, `/spec/` — all JS/TS ecosystem conventions) while Go's
existing `isTestSymbol` (Phase-8-adjacent, already shipped) is
SYMBOL/FILE-SUFFIX based (`_test.go`, `Test*`/`Benchmark*` name prefix) —
appropriate for Go's own conventions. Recommend keeping Go's existing
`isTestSymbol` rather than porting TS's JS-specific path patterns; document
this as an accepted, sensible per-language divergence in
`docs/FLAG-PARITY.md`, not a gap to close.
**`--quiet`/`--stdin`/exit-code semantics (verified, port these exactly):**
zero input files (no positional args, no/empty stdin) prints `"No files
provided..."` and exits 0 (not an error) unless `--quiet` is set, in which
case nothing prints; zero affected tests found (but files given) prints
`"No test files affected..."` in non-quiet/non-json mode, prints nothing in
`--quiet` mode; `--json` always emits the full JSON object regardless of
`--quiet`.
**Warning signs:** A plan task titled only "add --depth/--stdin/--filter/
--quiet flags to affected.go" with no corresponding `internal/query`
engine-file task.

### Pitfall 3: `affected`'s current `cobra.MinimumNArgs(1)` blocks `--stdin`-only invocations
**What goes wrong:** `internal/cli/affected.go:25` currently has
`Args: cobra.MinimumNArgs(1)` — TS's `affected [files...] --stdin` (with
zero positional args, reading `git diff --name-only | codegraph affected
--stdin`) would be REJECTED by cobra before `RunE` ever runs.
**Why it happens:** The `Args` validator was written when `affected` only
supported positional file arguments (v0.1/pre-Phase-8 scope).
**How to avoid:** Relax to `cobra.ArbitraryArgs` (or a custom validator
that requires `len(args) > 0 || stdinFlag`) and add the actual zero-input
handling (print advisory, exit 0) inside `RunE`.
**Warning signs:** `codegraph affected --stdin` erroring with a "requires
at least 1 arg" cobra usage message during manual verification.

### Pitfall 4: `install`'s `--auto-allow` is a genuine behavioral divergence from TS, not just a naming gap
**What goes wrong:** TS's `install` has NO separate `--auto-allow` flag —
permissions-list writing happens BY DEFAULT and is only suppressed via
`--no-permissions`. Go's `install` instead requires an explicit opt-in
`--auto-allow` (default `false`). A SURF-05 audit that only checks "is
there a flag with a similar name" will mark this "present" when the actual
default behavior is inverted.
**Why it happens:** Go's `--auto-allow` was very likely a deliberate,
security-conservative choice made during Phase 6 (auto-writing permissions
without being asked is a meaningfully more consequential default than
TS's). This session did not find a CONTEXT/PROJECT.md record explicitly
re-litigating this for Phase 8.
**How to avoid:** Do not silently flip Go's default to match TS in this
phase — CONTEXT.md never asked for it, and it's a security-relevant
default, not a mechanical flag/name issue. Document it explicitly in
`docs/FLAG-PARITY.md` as an accepted, deliberate divergence (with a one-line
rationale), and surface it as an Open Question for the planner/user to
confirm rather than resolving unilaterally.
**Warning signs:** A plan task that changes `--auto-allow`'s default to
`true` "to match TS" without a corresponding CONTEXT decision authorizing
a behavioral (not just cosmetic) change.

### Pitfall 5: `node`'s missing TS "file mode" is a scope-sizing trap
**What goes wrong:** TS's `node [name]` command supports a whole second
mode — when `name` is omitted, it reads a FILE with line numbers plus
dependents (`--offset`/`--limit`/`--symbols-only` govern this mode). Go's
`node.go` has no such mode at all (`--file` only disambiguates a symbol
match; there is no path where `symbol==""` triggers a file read). A
literal reading of SURF-05 ("every TS flag name + default is present or a
documented divergence") could be read as requiring this feature be BUILT,
which is a nontrivial net-new capability, not a flag/default reconciliation.
**Why it happens:** The phase description bundles "flag audit" and "flag
addition" language; file-mode is a genuine command-behavior gap that
predates this phase and was never called out in CONTEXT.md.
**How to avoid:** Treat this the same way `search`/`migrate` are already
treated (per CONTEXT's own precedent) — document `node`'s missing file
mode as an accepted, explicitly-recorded divergence in
`docs/FLAG-PARITY.md` rather than implementing it in Phase 8, UNLESS the
user/planner explicitly decides to size it in (it would be a genuinely new
capability, comparable in scope to a small feature, not "add a flag").
**Warning signs:** A plan task attempting to implement file-mode reading
as a sub-bullet of a "SURF-03 short flags" task, without its own
sizing/verification.

### Pitfall 6: The "not yet drop-in" caveat appears in more places than PROJECT.md's obvious summary line
**What goes wrong:** REL-04 says "sweep all of them" — a quick single-match
edit will miss occurrences.
**Why it happens:** The caveat text was written at v0.1 close and echoed
into multiple narrative sections as the project evolved.
**How to avoid:** This research located **3 confirmed sites** via
`rg -n "not yet drop-in|drop-in parity" .planning/PROJECT.md` (verify this
grep again at execution time in case PROJECT.md changes before Phase 8
executes):
1. Line ~13 — the milestone goal line ("...then cut the first real signed
   `v1.0.0` release — retiring the "not yet drop-in" caveat v0.1 shipped
   with.")
2. Line ~72 — the repo-state paragraph ("**Not yet drop-in parity** — the
   CLI command surface diverges from TS CodeGraph; closing that gap is the
   remaining bar for a 1.0.")
3. Line ~88 — the Key Decisions table row ("⚠️ Partial — v0.1 shipped the
   core capabilities + a signed release, but the CLI command surface
   diverges from TS, so it is NOT yet a drop-in parity replacement...")

Also check line ~95 (a related but distinct decision-log row: "Milestone
v1.0 = drop-in parity + human UX" — status "In progress") which should be
updated to "Complete"/"✓" alongside the caveat retirement, for narrative
consistency, though it does not contain the literal caveat phrase.
**Warning signs:** `rg -n "not yet drop-in" .planning/PROJECT.md` returning
any hits after the REL-04 edit is considered done.

## Code Examples

### TS `files --filter`/`--pattern` (SURF-02 ground truth)
```javascript
// Source: bin/codegraph.js:1348-1354 [VERIFIED: TS 1.3.1 dist, this session]
// Filter by path prefix (the "--filter <dir>" flag — NOT a glob despite
// the <dir> placeholder name)
if (options.filter) {
    const filter = options.filter;
    files = files.filter(f => f.path.startsWith(filter) || f.path.startsWith('./' + filter));
}
// Filter by glob pattern (the SEPARATE "--pattern <glob>" flag)
if (options.pattern) {
    const regex = globToRegex(options.pattern);
    files = files.filter(f => regex.test(f.path));
}

// globToRegex, used ONLY by --pattern, never by --filter:
function globToRegex(pattern) {
    const escaped = pattern
        .replace(/[.+^${}()|[\]\\]/g, '\\$&')
        .replace(/\*\*/g, '{{GLOBSTAR}}')
        .replace(/\*/g, '[^/]*')
        .replace(/\?/g, '[^/]')
        .replace(/\{\{GLOBSTAR\}\}/g, '.*');
    return new RegExp(escaped);
}
```
Recommended Go port for the new `--dir` flag (D-03's directory filter):
```go
// matches TS's files --filter (bin/codegraph.js:1349-1351) exactly —
// plain prefix match, NOT filepath.Match/glob.
func matchesDir(path, dir string) bool {
	if dir == "" {
		return true
	}
	return strings.HasPrefix(path, dir) || strings.HasPrefix(path, "./"+dir)
}
```

### TS `impact` depth clamp (confirms SURF-01's target default + confirms D-02's deliberate max-clamp divergence)
```javascript
// Source: bin/codegraph.js:~1824 [VERIFIED: TS 1.3.1 dist]
const depth = Math.min(Math.max(parseInt(options.depth || '2', 10), 1), 10);
```
Go (post-SURF-01, per D-02 — keep MaxDepth=50, only change the *default*):
```go
// internal/query/validate.go
const defaultDepth = 2 // was 5 — SURF-01/D-02
// MaxDepth stays 50 (D-02 explicit: intentionally wider than TS's 10)
```

### TS `affected`'s full command registration (SURF-04 ground truth, verified)
```javascript
// Source: bin/codegraph.js:1902-1911 [VERIFIED: TS 1.3.1 dist]
program
    .command('affected [files...]')
    .description('Find test files affected by changed source files')
    .option('-p, --path <path>', 'Project path')
    .option('--stdin', 'Read file list from stdin (one per line)')
    .option('-d, --depth <number>', 'Max dependency traversal depth', '5')
    .option('-f, --filter <glob>', 'Custom glob filter for test files (e.g. "e2e/*.spec.ts")')
    .option('-j, --json', 'Output as JSON')
    .option('-q, --quiet', 'Only output file paths, no decoration')
```

### Re-running the head-to-head bench (REL-03 — no new code)
```sh
# Exactly what bench.yml's workflow_dispatch/schedule trigger runs:
npm install -g @colbymchenry/codegraph@1.3.1
go build -o codegraph-bench ./cmd/codegraph
go run ./tools/bench/runner -mode headtohead \
  -go-binary "$(pwd)/codegraph-bench" \
  -ts-binary "$(command -v codegraph)" \
  | tee headtohead-results.json
```
This is `docs/BENCHMARKS.md`'s own documented regeneration command
(§"Regenerating the numbers") — REL-03 triggers `bench.yml` (manually via
`workflow_dispatch`, or waits for the weekly schedule) and transcribes the
CI run's numbers into `docs/BENCHMARKS.md`, replacing the current
local-machine-captured table (explicitly flagged in that doc as
provisional: "trigger [a CI run] once the release is cut, and replace this
table's absolute figures with the CI run's output").

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `impact` engine default depth 5 | 2 | This phase (SURF-01/D-02) | Matches TS exactly; MCP `codegraph_impact` inherits automatically (shared engine) |
| `affected` single-hop reverse lookup | Depth-bounded BFS with test-file-as-leaf pruning, default depth 5 | This phase (SURF-04) | New engine capability, not just a flag; must not reuse `impact`'s `clampDepth`/`defaultDepth=2` |
| No `docs/FLAG-PARITY.md` | A committed, per-command TS↔Go flag/default/status matrix | This phase (SURF-05/D-06) | Becomes the canonical parity oracle REL-04 gates on |
| v0.1 shipped only prerelease tags (`v0.0.0-rc.3`) | First stable `v1.0.0` tag, full GitHub release, `codegraph upgrade`'s "latest" | This phase (REL-02/D-08) | Closes DIST-02; the LOCKED `release.yml`/`verify.go` contract does not change shape, only the tag matched |
| Local-machine-only (darwin/arm64) head-to-head numbers in `docs/BENCHMARKS.md`, explicitly flagged provisional | CI-run (`bench.yml`, ubuntu-latest) numbers | This phase (REL-03/D-10) | Numbers become reproducible on standardized hardware; ratios expected to hold, absolutes will shift |

**Deprecated/outdated:** none — this phase doesn't deprecate any existing
mechanism; it exercises and finalizes what Phases 1-7 already built.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go's `search`/`migrate`/`githooks` commands should be documented as "Go-only extensions" rather than reconciled toward any TS equivalent, since TS genuinely has no such commands | Standard Stack table, multiple rows | Low — directly confirmed by reading TS's real `--help` command list this session; no TS command named `search`, `migrate`, or `githooks` exists |
| A2 | `install`'s `--auto-allow` default-off behavior should NOT be silently flipped to match TS's default-on (via `--no-permissions`) | Pitfall 4 | Medium — if the user actually wants Go to match TS's default-on auto-permissions behavior, this needs an explicit decision (security-relevant default), not a unilateral SURF-03 short-flag change |
| A3 | `files --format` defaulting to `"flat"` in Go vs. `"tree"` in TS, and `files --depth` vs. TS's `--max-depth` naming, should be documented divergences rather than silently changed | SURF-03 table, `files` row | Medium — an existing Go user's scripts/muscle memory around `files`'s current flat-default output could break if the default silently flips to match TS |
| A4 | `node`'s missing TS "file mode" (`--offset`/`--limit`/`--symbols-only`) is out of Phase 8's scope and should be a documented divergence, not new-built in this phase | Pitfall 5 | Medium — if the user actually wants file-mode read parity, this is a real feature-sized task that needs its own sizing, not a bullet inside a "flag audit" plan item |
| A5 | Reusing the pinned `@colbymchenry/codegraph@1.3.1` npm package (rather than the GitHub source tree at a specific commit) as the SURF-05 ground-truth oracle is the correct approach | Standard Stack / Don't Hand-Roll | Low — this is the exact same reference version `bench.yml` and `docs/BENCHMARKS.md` already use; consistency across REL-03 and SURF-05 is a feature, not a risk, but flagging since it's a methodology choice not explicitly dictated by CONTEXT.md |

**If this table is empty:** N/A — see entries above; all are LOW-MEDIUM
risk methodology/scope judgment calls made where CONTEXT.md was silent,
not compliance/security-critical unknowns.

## Open Questions

1. **Should `install --auto-allow`'s default be reconciled toward TS's
   default-on behavior, or stay Go's explicit opt-in?**
   - What we know: TS writes permissions by default (opt-out via
     `--no-permissions`); Go requires explicit opt-in (`--auto-allow`,
     default false).
   - What's unclear: whether this was a deliberate Phase-6 security
     decision or simply never flagged as a divergence until now.
   - Recommendation: keep Go's current behavior, document as an accepted
     divergence in `docs/FLAG-PARITY.md` with a one-line security
     rationale; do not change default behavior in a "mechanical
     reconciliation" phase without an explicit user decision.

2. **Should `files --format` change its default from `"flat"` to `"tree"`
   to match TS, given SURF-02 is already touching this command's options
   struct?**
   - What we know: TS defaults to `"tree"`; Go defaults to `"flat"`
     (confirmed via `internal/query/files.go:37`).
   - What's unclear: whether any existing golden fixture or documented Go
     behavior depends on the current flat default.
   - Recommendation: document as accepted divergence unless the user opts
     to align it; a default-value change is a user-visible behavior change
     that should not ride along silently inside the `--dir` flag addition.

3. **Should `node`'s missing TS file-mode be implemented in Phase 8, or
   deferred as a documented divergence (like `search`/`migrate`)?**
   - What we know: it's a real, nontrivial TS capability with 3 dedicated
     flags (`--offset`/`--limit`/`--symbols-only`) that Go has zero
     support for today.
   - What's unclear: whether this was considered and implicitly excluded
     from CONTEXT.md's SURF-01..05 decisions, or simply not noticed during
     discuss-phase.
   - Recommendation: document as an accepted, explicitly-recorded
     divergence for v1.0; revisit as its own future-milestone item if
     wanted (comparable scope to a small phase, not a flag tweak).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `npm`/`node` (for the TS 1.3.1 reference CLI) | SURF-05 audit ground truth, `bench.yml`'s REL-03 | ✓ (verified this session — installed `@colbymchenry/codegraph@1.3.1` into a throwaway prefix) | node/npm present locally; TS package resolved to 1.3.1 exactly | If unavailable at execution time: the exact per-command flag tables in this document (verified from a real install) serve as the frozen oracle, same posture as `testdata/golden/README.md`'s own "if capture.sh can no longer run" fallback |
| `go list -deps -json` + `jq` | REL-01's CGo-closure audit | ✓ (verified — ran the exact command, got the exact output documented above) | Go 1.26.5 (`go version` this session), `jq` present | None needed — both are already project/toolchain baseline requirements |
| `govulncheck` | REL-01 | ✓ (v1.6.0, verified via `govulncheck --version`) | v1.6.0 | CI's `golang/govulncheck-action` is the canonical gate regardless of local availability |
| `zig`, `gcc-mingw-w64-x86-64` | Release cross-builds (REL-02), Windows cross-vet gate (REL-01 gotcha) | Not verified locally (CI-only concern — these run inside `release.yml`/`ci.yml`'s runners, already pinned) | zig `0.15.1` (pinned in both workflows) | None needed — CI already provisions these; this phase does not change that provisioning |
| `cosign`, `slsa-verifier` | Post-release verification (REL-02 runbook) | Not installed locally this session (not needed for research — `docs/RELEASE.md` already documents exact install/usage) | cosign v3 (installed via pinned Action in CI) | The runbook (folded todo) should note these as maintainer-machine prerequisites for POST-release verification, not build-time requirements |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** none blocking — all identified
gaps have either a working fallback or are already covered by existing CI
provisioning untouched by this phase.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (existing project-wide convention) |
| Config file | none — see `go.mod`/`ci.yml` for the existing, unmodified test-invocation shape |
| Quick run command | `go test ./internal/query/... ./internal/cli/...` |
| Full suite command | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` (three invocations — `go list ./...` silently skips `testdata/`, per existing `ci.yml`/`testdata/golden/README.md` documentation) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|--------------------|--------------|
| SURF-01 | `impact` default depth is 2, MCP+CLI together | unit + golden | `go test ./internal/query/... -run TestImpact` + `go test ./testdata/golden/... -run TestGoldenParity` (regen `impact.json` fixtures if depth appears in golden output) | ✅ `internal/query/traverse_test.go` (existing `TestImpact`); ⚠️ golden regen may need a fixture update — Wave 0 gap if so |
| SURF-02 | `files --dir <glob>` new flag matches TS prefix semantics, `--filter`=language retained | unit | `go test ./internal/query/... -run TestFiles` | ❌ new subtest needed — Wave 0 gap |
| SURF-03 | Missing short-flag aliases added, no cobra collisions | build-time + unit | `go build ./...` (cobra panics on a real short-flag collision at command construction) + `go test ./internal/cli/... -run TestFlag` (new) | ❌ new test needed — Wave 0 gap; cobra's own panic-on-duplicate-shorthand is a free correctness check at `go build`/first invocation |
| SURF-04 | `affected --stdin/--depth/--filter/--quiet` match TS BFS/leaf/exit-code semantics | unit | `go test ./internal/query/... -run TestAffected` (extend existing `TestAffected` in `traverse_test.go`) + `go test ./internal/cli/... -run TestAffectedCmd` (new) | ⚠️ `internal/query/traverse_test.go` has `TestAffected` today (single-hop) — needs BFS-depth subtests; CLI-level test is new — Wave 0 gap |
| SURF-05 | `docs/FLAG-PARITY.md` complete, 0 undocumented divergences | doc completeness check (optional tree-walk test, Claude's discretion per D-06) | A new test walking `newRootCmd().Commands()` + `cmd.Flags().VisitAll(...)`, asserting every long-flag name appears as a substring somewhere in `docs/FLAG-PARITY.md` | ❌ new test needed if the tree-walk-drift option is taken — Wave 0 gap; doc-only alternative needs no test |
| REL-01 | Charm closure has 0 CGo pkgs; govulncheck clean; SBOM regen; double-build passes | CLI + CI | `go list -deps -json ./internal/cli/... \| jq -s '...'` (verified command above, exit via jq assertion `cgo_in_closure == []`) + `govulncheck ./...` + re-run `ci.yml`'s `reproducibility` job | ✅ command verified working this session; CI jobs already exist unmodified |
| REL-02 | Signed `v1.0.0` cut | **procedural/human-gated** — not automatable | `git tag v1.0.0 && git push origin v1.0.0` (maintainer-manual, per D-08) then `cosign verify-blob ...` (see Pattern 3) | N/A — this is the phase's one genuinely manual, backstop-needed step |
| REL-03 | Benchmarks re-run + published | **procedural/human-gated** trigger, automated measurement | `gh workflow run bench.yml` (manual dispatch) or wait for weekly schedule; then transcribe `docs/BENCHMARKS.md` by hand from the CI job summary | ✅ `bench.yml` exists and runs unmodified; the *publish/transcribe* step is manual |
| REL-04 | Drop-in gate: TEST-01 harness green + FLAG-PARITY.md green; PROJECT.md caveat retired | integration + doc sweep | `go test ./testdata/golden/...` (re-run, do not modify) + `rg -n "not yet drop-in" .planning/PROJECT.md` returning zero hits after the edit | ✅ harness exists (Phase 1); the `rg` zero-hits check is a trivial, scriptable completeness gate for the doc edit |

### Sampling Rate

- **Per task commit:** `go test ./internal/query/... ./internal/cli/...` (fast, targeted)
- **Per wave merge:** the full three-invocation suite above, plus
  `go test ./test/integration/...`
- **Phase gate:** full suite green (including `testdata/golden`) BEFORE
  `/gsd-verify-work`, and BEFORE the REL-02 tag push (D-01's SURF-green-
  before-REL barrier is exactly this gate).

### Wave 0 Gaps

- [ ] `internal/query/traverse_test.go` — extend `TestAffected` with
  depth-bounded/multi-hop/leaf-pruning subtests (covers SURF-04's engine
  change)
- [ ] `internal/cli/affected_test.go` (does not exist yet — check for a
  `cli_test.go`-embedded case first) — covers `--stdin`/`--quiet`/exit-code
  behavior at the CLI layer
- [ ] `internal/query/files_test.go` (or wherever `TestFiles` lives) — new
  `--dir` prefix-match subtest (covers SURF-02)
- [ ] Optional: a new `internal/cli/flag_parity_test.go` walking
  `newRootCmd().Commands()` if the D-06 tree-walk-drift test is chosen
  over doc-only (Claude's discretion)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V5 Input Validation | yes | Reuse the existing `validateDepth`/`validateLimit`/`validateMaxFiles` pattern (`internal/query/validate.go`) for `affected`'s new `--depth`/`--filter` — reject negative depth, cap at the same `MaxDepth=50` ceiling as `impact` (do not introduce a second, inconsistent ceiling) |
| V14 Configuration (dependency/supply-chain) | yes | REL-01's whole job: `go list -deps` closure audit, `govulncheck`, SBOM — all pre-existing, standard controls, being re-run not re-designed |
| V10 Malicious/Untrusted Input (glob patterns as ReDoS vector) | yes | `affected --filter <glob>`'s glob-to-regex conversion (whether ported from TS's hand-rolled version or Go's `filepath.Match`) must not accept attacker-controlled patterns capable of catastrophic backtracking. TS's own conversion (`bin/codegraph.js:1448-1456`) uses only `.*`/`[^/]*`/`[^/]` — no nested quantifiers, so it is not ReDoS-prone as written; a Go port should preserve this same non-nested-quantifier shape rather than naively wrapping user input into a more expressive regex engine |
| V2/V3 Auth/Session | no | This phase touches no authentication/session surface |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Release artifact tampering / supply-chain substitution | Tampering | Already fully mitigated: per-binary cosign keyless signing bound to the exact `release.yml` workflow + tag-ref identity (`verify.go`'s LOCKED constants), verified by `codegraph upgrade` before every binary swap. REL-02 does not change this control, only exercises it for the first stable tag |
| Malicious/typo-squatted new dependency introduced during a "quick Charm-related fix" | Supply chain / Spoofing | Not applicable this phase — REL-01 is an audit of already-added (Phase 6/7) dependencies; no new `go get` is part of this phase's own deliverables. If a plan does end up adding any dependency (e.g. a doublestar-style glob library, which this research recommends AGAINST — see Don't Hand-Roll), it must go through the standard `package-legitimacy check` gate before being accepted |
| ReDoS via user-supplied `affected --filter <glob>`/`files --pattern <glob>` | Denial of Service | Keep the existing non-backtracking-prone conversion shape (linear `[^/]*`/`.*` substitutions only, no nested quantifiers or lookaheads) for any new glob-to-regex logic; this is a local CLI flag (not network-exposed), so blast radius is low, but the mitigation costs nothing to preserve |
| Unauthorized workflow-identity spoofing during release verification | Spoofing | Already mitigated by the anchored, full-match `releaseWorkflowRefPattern` regex (`^...$`, not a prefix match) — a differently-triggered workflow in the same repo cannot satisfy the SAN pattern. This phase's runbook must not weaken this to a prefix match "for convenience" |

## Sources

### Primary (HIGH confidence — read/run directly this session)

- Live install `npm install @colbymchenry/codegraph@1.3.1 --no-save` (into
  a throwaway scratch prefix) — `--help` output for every command,
  `bin/codegraph.js` source for `files`/`affected`/`impact` — the
  authoritative TS 1.3.1 flag-surface and semantics oracle for this
  research
- `internal/cli/*.go` (all 30+ command files) — read directly, current Go
  flag surface
- `internal/query/validate.go`, `internal/query/traverse.go` — depth clamp
  and `Affected`/`Impact` implementations, read directly
- `internal/upgrade/verify.go` — LOCKED release-identity constants, read
  verbatim
- `.goreleaser.yaml`, `.github/workflows/release.yml`,
  `.github/workflows/ci.yml`, `.github/workflows/bench.yml`,
  `docs/RELEASE.md`, `docs/BENCHMARKS.md` — all read in full
- `.planning/todos/pending/2026-07-14-document-release-cut-procedures-runbook.md`
  — folded runbook todo, read in full
- `go list -deps -json ./internal/cli/... | jq ...` — ran locally this
  session, output captured verbatim (§Package Legitimacy Audit)
- `govulncheck --version` — ran locally, confirmed v1.6.0 available
- `testdata/golden/README.md` — read in full (drop-in gate mechanics,
  `weft`/`colbymchenry-codegraph`/`synthetic-parity` corpora)

### Secondary (MEDIUM confidence)

- `.planning/PROJECT.md`, `.planning/STATE.md`, `.planning/ROADMAP.md`,
  `.planning/REQUIREMENTS.md`, `.planning/phases/08-.../08-CONTEXT.md` —
  project-internal planning docs, read in full for scope/constraint
  grounding

### Tertiary (LOW confidence)

- None — no ungrounded web-search-only claims were needed for this phase;
  every factual claim above traces to a file read or a command run in
  this session.

## Metadata

**Confidence breakdown:**
- Standard Stack (flag surface): HIGH — verified against a real, pinned
  TS 1.3.1 install's `--help` output and source, cross-checked against
  every Go command file read directly
- Architecture (release pipeline reuse): HIGH — every workflow file read
  in full; the CGo-closure claim was independently verified by running
  the actual command, not just described
- Pitfalls: HIGH — Pitfalls 1-3 verified via direct source reads and a
  live command run (`rg`); Pitfalls 4-6 are judgment calls flagged
  explicitly as Open Questions/Assumptions rather than asserted as fact

**Research date:** 2026-07-19
**Valid until:** ~30 days for the release-pipeline findings (stable
infrastructure, low churn); the TS 1.3.1 flag-surface findings are valid
indefinitely UNLESS the pinned reference version changes (this project
targets exactly 1.3.1; TS has since released through 1.4.1 per `npm view
@colbymchenry/codegraph versions`, but this project's parity target
remains 1.3.1 per every existing artifact — `ts-version.txt`,
`docs/BENCHMARKS.md`, `bench.yml`)
