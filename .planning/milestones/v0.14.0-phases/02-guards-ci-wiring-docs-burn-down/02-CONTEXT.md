# Phase 2: Guards, CI Wiring & Docs Burn-down - Context

**Gathered:** 2026-09-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Every guard in the ledger that could pass vacuously is shown to fail against its recorded incident shape, the checks that today exist only as local Taskfile targets run on every pull request, and every documentation and planning claim — dependency shape, provenance scope, scanner coverage, window status, todo table, seed status — states what is true today. Requirements: GRD-09, GRD-10, GRD-11, GRD-12, GRD-14, DOCS-08, DOCS-09, DOCS-10, DOCS-11. GRD-13 rides with Phase 4 by roadmap decision.

The maintainer's framing for this phase, applied to every item: **"what are we testing, and why?"** A guard exists only for a blind spot that has actually fooled someone; a wording change gets no test; never test git, the filesystem, or a third-party tool's behaviour (v0.13.0 Phase 12 ruling carried forward).

</domain>

<decisions>
## Implementation Decisions

### web:drift and the 98cd41dd incident (GRD-09)
- **D-01:** The `web:drift` gate is **not changed**. The maintainer reset the requirement after the discussion established the facts: CI runs `web:drift` on a clean checkout, where `find web/build` *is* the git tree, so `98cd41dd`'s shape (9 brand-new chunk files on disk, never staged) hashes to a different output digest than its committed `.build-manifest` and **fails in CI**. The local `find` enumeration is deliberate — `//go:embed all:build` ships whatever is on disk, tracked or not (Taskfile comment on `web_output_files`). A `find`-vs-`git ls-files` set assertion would test whether the developer staged what they built — git hygiene, not the gate — and buys only one saved CI round trip. — **Reversibility:** reversible — the paired assertion can be added later if a *second* incident shows the local misread costs more than a CI cycle.
- **D-02:** GRD-09's deliverable is the **proof, not a fix**: replay `98cd41dd`'s tree in a scratch clone (clean checkout of that commit, or a scratch repo reconstructing "manifest + index.html reference chunks that are absent from the tree"), run `task web:drift`, record the non-zero exit and the digest mismatch in `02-MUTATION-LOG.md` (the `07-MUTATION-LOG.md` shape: pre-mutation cleanliness gate, mutation applied, guard observed RED, byte-clean revert). Close WINDOWS #29 via `gsd-tools windows fixed 29` with that evidence as the cause: the CI path was never vacuous; the ledger's suggested fix is declined.
- **D-03:** `REQUIREMENTS.md`'s GRD-09 text and ROADMAP success criterion 1 are **reworded** to match (a value change in an existing shape): "`web:drift` is demonstrated RED against commit `98cd41dd`'s exact incident shape on a clean checkout, closing WINDOWS #29 with the recorded cause; the local `find` enumeration is retained by design." No "paired assertion" language survives.

### CI wiring (GRD-11)
- **D-04:** `check:gonum` and `check:no-force-layout` run as **steps inside the existing `test` job** in `ci.yml`, next to the pnpm-audit and `web:drift` steps. `test` is already a required context, so both block merges the moment they land — no ruleset edit, no fixture change. Each step calls the Taskfile target; the targets' own positive controls are the guard (Phase 7 D-07: no shape test asserting that a Task target calls a script).

### Ruleset drift (GRD-12)
- **D-05:** The comparison is **exact set equality**: the fixture's context set == the live `protect-main` ruleset's `required_status_checks[].context` set, both directions. A context required live but absent from the fixture under-asserts; a context in the fixture but not required live (today's `goreleaser check (config validation, DIST-01)`) asserts a requirement that does not exist. Both are drift.
- **D-06:** The comparison runs as a **CI-only step** (in the `test` job) — no local Taskfile target, no Go test. It fetches `GET /repos/seanb4t/codegraph-go/rulesets/20157557` (Metadata:read; the default `GITHUB_TOKEN` suffices, unauthenticated works on this public repo), extracts the required contexts, prints both counts, and exits non-zero on any diff. **An API failure (non-2xx, empty body, unparseable JSON) is a hard failure naming the cause — never a skip.** GRD-07 was declined at v0.13.0 partly for its "skip-clean offline" clause; a skip is a vacuous pass (rule `84d1gfpywd`). — **Reversibility:** reversible — a local target can wrap the same step later.
- **D-07:** The context list moves out of `taskfile_shape_test.go`'s `requiredCheckNames` Go literal into a **data file read by both** the Go test (`TestRequiredCheckJobsExist` keeps its job-name assertion) and the CI step — one list, no bash parsing of Go source, no duplication. Location and format at Claude's discretion (a newline-delimited text file under `.github/` is the obvious shape); the Go test must fail loudly if the file is missing or empty.
- **D-08:** The maintainer **adds two contexts to the live ruleset during this phase**: `goreleaser check (config validation, DIST-01)` and `tmux e2e (real-pty harness, TTY-01..TTY-07)` (the Phase 08-03 `user_setup` item still open in STATE.md). The fixture therefore grows to **8**. This is a `user_setup` step with a `blocking-human` precondition on the plan that lands the drift step: the executor hands the maintainer the exact context strings and a reviewable `gh api` PUT/`--method PATCH` body, then verifies the live set before the fixture is edited. Once `tmux e2e` is required, every PR runs the tmux job — that is the maintainer's informed choice. — **Reversibility:** costly — removing a required context later is a repo-settings change plus a fixture edit plus this test going red until both agree.

### Vendored component drift (GRD-10)
- **D-09:** The isolation is **already done by evidence**: the `components-drift.yml` scheduled run of 2026-09-14 (run 34820878640) ran under Corepack-resolved `pnpm v11.23.0` and reported `button.svelte`, `command-group.svelte` and `command-input.svelte` all differing from `shadcn-svelte@1.5.1`'s regeneration. The local-toolchain hypothesis (pnpm 12 vs pinned 11.23.0) in WINDOWS #31 is **refuted**; the registry moved under a pinned CLI version — which is what this monitor exists to detect. A fresh `workflow_dispatch` run is still triggered to capture the current diff for the re-vendor.
- **D-10:** The decision rule for vendored components that drift is **re-vendor with a reviewed diff**, never waive: regenerate from the registry, review the diff (visual and API changes called out per file), run every gate, commit as a human-approved re-vendor in the lineage of `205da685`. Waiving would be a growing lie as more families drift.
- **D-11:** Re-vendor **all eight families to one registry snapshot** — one upstream point in time for every vendored component, one reviewed diff, one commit; families that regenerate byte-identically show no change. Never a tree with three components at "new registry" and five at "old".
- **D-12:** The **executor reviews and commits** when all gates are green (`pnpm check` exit 0 with zero errors; `cd web && pnpm vitest run` exit 0 with zero "unhandled errors" lines; `task web:build && task web:drift`; the live graph/browse check scripts; `task web:components:drift` green under Corepack). No mid-plan checkpoint; the per-file diff summary (with any visual change named) goes in the SUMMARY and the maintainer reviews at end of phase.

### Doc claims (DOCS-08, DOCS-09, DOCS-10)
- **D-13:** `docs/RELEASE.md`'s dependency paragraph **drops the raw counts** ("27 direct", "134 indirect", "14 grammar modules", "remaining 13") and keeps the shape: wide-but-shallow — one grammar module per supported language plus a short, named list of deliberately chosen direct requires — crediting `modelcontextprotocol/go-sdk` (not `mark3labs/mcp-go`) and `cockroachdb/pebble/v2`, and pointing at `go list -m all` / `go mod graph` for live numbers. **No drift test** (Phase 12: a wording change gets no test; nothing is left to drift).
- **D-14:** DOCS-09 is a **whole-repo, positive-controlled census**: word-boundary, multiline `rg` for checksums-file wording inside provenance/SLSA/attest sentences across `README.md`, `docs/`, `.github/workflows/`, `SECURITY.md` and `internal/upgrade` comments — excluding `CHANGELOG.md` and `.planning/`. A planted phrase must be found by the census before it is trusted (the Phase 3 census discipline). Every hit is reworded to: provenance is attested over each platform binary; the checksums file is the *transport* for the subject list, not a subject. GH #14 is closed citing the census output; its stale line numbers are not relied on.
- **D-15:** Root `SECURITY.md` gains **one sentence** for DOCS-10: govulncheck blocks merges on the main module graph; the isolated tool modfiles (`go.tool*.mod`) are scanned by the advisory `tool-vuln` job, which reports and never fails the build. **No named exposure**, no drift assertion. The existing two-scanner disjoint-scope paragraph and pnpm audit's blocking description stay as written.

### Claude's Discretion
- **GRD-14 / DOCS-11 bookkeeping.** `gsd-tools windows` offers `status | append | waive | fixed` — there is no "annotate as record-only" verb and no such status. Therefore: #16 and #33 are closed with `gsd-tools windows fixed 16` / `fixed 33` and the verification evidence is recorded in the plan SUMMARY (and, if the verb accepts a note, in the row); #20, #21 and #34 **stay `open`**, their record-only status is stated in the SUMMARY and in STATE.md's Blockers list, and the missing annotate verb is **reported upstream** (planning-artifacts rule: never invent a shape in a tool-owned file). STATE.md's Pending Todos table (17 rows) versus an empty `.planning/todos/pending/` is reconciled through `gsd-tools todo` / `state` verbs where one exists; where none does, the SEED-002 precedent applies (frontmatter values only). SEED-001's frontmatter records its consumption by v0.12.0 in the same fields SEED-002 uses.
- Data-file location/format for the shared required-contexts list (D-07); mutation-log file name (`02-MUTATION-LOG.md`, following `07-MUTATION-LOG.md`).
- Exact wording of the reworded requirement/criterion (D-03), the RELEASE.md paragraph (D-13), the census pattern (D-14) and the SECURITY.md sentence (D-15) — subject to the bars above.
- Whether the ruleset PUT is applied by the maintainer through the UI or through the executor-prepared `gh api` call (D-08) — the maintainer's call at the checkpoint.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase framing and prior rulings
- `.planning/ROADMAP.md` §"Phase 2: Guards, CI Wiring & Docs Burn-down" — goal, success criteria (criterion 1 to be reworded per D-03), notes on pitfall 13, GRD-07/08 promotion, DOCS-11 bounds
- `.planning/REQUIREMENTS.md` — GRD-09…GRD-14, DOCS-08…DOCS-11 (GRD-09 text to be reworded per D-03)
- `.planning/research/PITFALLS.md` §"Pitfall 13" — the paired-assertion argument the maintainer declined after establishing CI already fails on the incident (record why, do not re-litigate)
- `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-CONTEXT.md` — D-05 (extract guard logic to `scripts/`), D-07 (no shape test mirroring a script), D-11 (mutation-log shape)
- `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` — the mutation-log format to follow for `02-MUTATION-LOG.md`
- `.planning/milestones/v0.13.0-REQUIREMENTS.md` lines 86-87 — GRD-07/GRD-08 as declined (the "skip-clean offline" clause is what D-06 removes)
- `.planning/phases/01-defect-flake-burn-down/01-CONTEXT.md` — D-07 (fix, never waive), D-17 (per-run delimiter precedent for CI-step scripting)
- `/Users/sean/.claude/rules/planning-artifacts.md` — tool-owned generated files: values in existing shapes only, tool verbs first, report upstream gaps

### Incident and ledger evidence
- `.planning/WINDOWS.md` rows #13, #16, #20, #21, #29, #31, #33, #34 — the ledger entries this phase closes, reconciles, or leaves record-only
- Commit `98cd41dd` (`feat(05): rebuild the committed bundle …`) and its fix `fad3b39c` — the exact incident shape for D-02
- `Taskfile.yml` §`web:drift`, §`web_output_files`, §`web_hash_tree`, §`check:gonum`, §`check:no-force-layout`, §`web:components:drift` — the targets this phase wires or proves
- `.github/workflows/ci.yml` — the `test` job (BLD-01 `web:build:verify` then BLD-03 `web:drift` ordering; pnpm audit gate), the `govulncheck` (blocking) and `tool-vuln` (advisory) jobs
- `.github/workflows/components-drift.yml` and run `34820878640` (2026-09-14, pnpm v11.23.0, three files differing) — D-09's evidence
- `internal/upgrade/taskfile_shape_test.go` — `requiredCheckNames` (lines ~70-85) and `TestRequiredCheckJobsExist`; ruleset id 20157557
- GitHub REST: `GET /repos/{owner}/{repo}/rulesets/{ruleset_id}` (docs.github.com/en/rest/repos/rules) — Metadata:read; unauthenticated on public repos; `bypass_actors` withheld without write access

### Doc targets
- `docs/RELEASE.md` — dependency paragraph (~lines 330-360) for D-13; provenance wording for D-14
- `docs/RELEASE-PROCEDURES.md`, `.github/workflows/release.yml`, `README.md`, `SECURITY.md` — census scope for D-14; `SECURITY.md` two-scanner paragraph (~lines 60-72) for D-15
- GitHub issue #14 — the three originally cited sites (line numbers stale); closed by the census
- `.planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md`, `.planning/seeds/SEED-002-homebrew-installation-path.md` — the frontmatter precedent for DOCS-11

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `scripts/check-workflow-output-delimiter.sh` (Phase 1, FIX-10): the `--self-test`-first, positive-controlled check-script shape; `check:workflow-output-delimiter` shows how a Taskfile target wraps it
- `web/scripts/graph-console-check.mjs`: same discipline for live-browser checks; its `--self-test` prints counts before asserting
- `.github/workflows/components-drift.yml`: already resolves pnpm through Corepack from `web/package.json`'s `packageManager`; the re-vendor of D-10/D-11 uses the same regeneration path the workflow uses
- `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md`: the format for D-02's replay entry

### Established Patterns
- Guards print the count they inspected **before** asserting on it (every `check:*` target); a zero count is a failure, never a pass
- `[ci skip]`/`[skip ci]` never appear in a commit message (rule `f18zrdsgx5`); `protect-main` requires 6 contexts today, 8 after D-08
- Taskfile `cmds:` strings pass through Go `text/template` — a literal `{{` needs `{{"{{"}}` (gotcha `npwg333n9h`); prefer a `scripts/` file for anything template-hostile
- Local Go gates need `GOTOOLCHAIN=go1.26.6` (ambient Homebrew go 1.27.1 breaks `cockroachdb/swiss`); CI is unaffected via `go-version-file`
- Vitest is judged by exit code 0 **and** zero "unhandled errors" lines, never by the pass count (Phase 1 WR-01); piped `svelte-check` is MACHINE format (uppercase ERRORS)
- `web/build/**` is committed generated output under the `web:drift` gate — any `web/src` change is rebuilt and committed in the same commit

### Integration Points
- `ci.yml` `test` job: two new `task check:*` steps (D-04) and one ruleset-drift step (D-06) join the existing pnpm-audit/web:drift/proto:drift steps
- `internal/upgrade/taskfile_shape_test.go`: `requiredCheckNames` becomes a loader of the shared data file (D-07); the job-name assertion is unchanged
- `web/src/lib/components/ui/**`: eight vendored families regenerated to one snapshot (D-11); `web/build/**` rebuilt in the same commit
- `.planning/WINDOWS.md`, `STATE.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `seeds/`: value-only edits through tool verbs where they exist

</code_context>

<specifics>
## Specific Ideas

- The phase-wide test: "what are we testing and why?" — the maintainer applied it to GRD-09 and it removed the code change entirely. Planners should apply it to every remaining item before adding a mechanism, and record the answer in the plan's objective.
- A guard proven RED once against the real incident (D-02) is worth more than a new assertion that has never failed.
- The ruleset change (D-08) is the maintainer's action; the executor's job is to make it a one-glance review (exact strings, exact API body) and to verify the live set afterwards, not to apply it silently.

</specifics>

<deferred>
## Deferred Ideas

- A local pre-push honesty check for `web/build` (`git status --porcelain web/build` must be empty before `web:drift` reports PASS) — declined for this phase as git hygiene rather than a gate property; revisit only if a second `98cd41dd`-shaped incident costs more than one CI cycle.
- Pinning the shadcn-svelte registry snapshot (so the drift monitor compares against a recorded upstream point rather than "latest") — a change to what the monitor asks, not in scope; the monitor's current shape (D-16, T-04-27) is deliberate.
- Gating the three `window.__codegraphFileGraph*` debug globals behind a build flag (Phase 1 SECURITY R-01-14 follow-up) — not this phase.

</deferred>

---

*Phase: 02-guards-ci-wiring-docs-burn-down*
*Context gathered: 2026-09-15*
