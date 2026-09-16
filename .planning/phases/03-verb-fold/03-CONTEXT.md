# Phase 3: Verb Fold - Context

**Gathered:** 2026-09-16
**Status:** Ready for planning

<domain>
## Phase Boundary

The CLI verb surface is streamlined — `query` folds into `search --full` on the same ranking function and `unlock` becomes `daemon unlock` — with every consumer of the old names found before and after the edit, the removed verbs exiting non-zero with a rename message for one release, and the generated reference, completions, man pages and goldens re-frozen to the new surface in a reviewed diff. Requirements: VERB-01…VERB-08. The 8-tool MCP set and the wire oracle's transcripts are unchanged by definition — this is a CLI-only fold.

The maintainer's phase-wide test carries forward from Phases 1–2: **"what are we testing, and why?"** — a rename gets no new test beyond the goldens' RED and the flag-parse test the requirement names; never test cobra's or release-please's behaviour.

</domain>

<decisions>
## Implementation Decisions

### `search --full` output (VERB-01, VERB-02)
- **D-01:** In the human branch, a `--full` hit is **two lines**: line 1 is byte-identical to default `search`'s line (`Name (Kind) FilePath:StartLine`); line 2 is four spaces of indent then `QualifiedName  Signature` (two spaces between; when `Signature` is empty — packages, files — line 2 is just the qualified name). `search --full` is therefore a **superset** of default `search`: `search --full … | rg '^\S'` reproduces default output. — **Reversibility:** reversible — a formatting change to the human branch only.
- **D-02:** The human `--full` branch prints the **same WORK-02 worktree-mismatch notice, in the same position** (top of the human branch, after the `--json` early return) as default `search` and today's `query`. JSON branches never print it.
- **D-03:** `search --full --json` emits **`query.MarshalQueryJSON(nodes)`** (the old `query --json` envelope: `[{"node": {...}}]`), unchanged in shape; default `search --json` stays **`json.Marshal(locs)`** (the `Location` array). Both shapes survive (Pitfall 4); the envelope is gated on `--full` only. Same engine path: `eng.Query` for `--full`, `eng.Search` otherwise — both already rank through `matchNodes`.
- **D-04:** `search` gains `query`'s **short flags**: `-k` (`--kind`), `-l` (`--limit`), `-j` (`--json`); `-p` (`--path`) already exists on both. Long forms are unchanged. One flag-parse test covers long and short forms on `search` and `search --full` (VERB-02) — that test is the phase's only new test besides the goldens' RED.

### Rename stubs (VERB-03, VERB-04, VERB-08)
- **D-05:** `codegraph query …` and `codegraph unlock …` become **hidden dedicated stubs** (`Hidden: true`, like `man`): they execute nothing, print to **stderr**, and return a non-nil error so `main.go` exits **1** — the tree's only error exit; no new exit code, no `os.Exit` in the stub, and **cobra's `Deprecated` field is not used**.
- **D-06:** The stderr text is the rename **plus the full replacement invocation plus the lifetime**, one message per stub:
  - `"query" has been renamed to "search --full" — run: codegraph search --full <term>` then `the "query" stub is removed in the next minor release (v0.15.0)`
  - `"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]` then `the "unlock" stub is removed in the next minor release (v0.15.0)`
  The first clause is the requirement's verbatim text; the second is what makes it actionable. Any args/flags passed to a stub are ignored (nothing is forwarded, nothing runs).
- **D-07:** `daemon unlock` is **`unlock.go`'s body moved verbatim** under the `daemon` parent (`Use: "unlock [path]"`, `MaximumNArgs(1)`, identical flags, identical behaviour); it inherits nothing new from the parent. `codegraph unlock` becomes the stub of D-05/D-06.
- **D-08:** The two live error messages that say ``run `codegraph unlock` …`` (`internal/daemon/lock.go:208` and `internal/cli/index.go:128`, plus the `index.go:126` comment) change to ``codegraph daemon unlock`` **in the same commit as the move**, with `index_lock_test.go` / daemon tests that assert the text updated in that commit (RED first). The VERB-05 census must find zero old-verb references outside the stubs — no exceptions.
- **D-09:** The stubs' removal is tracked **twice**: a ROADMAP `## Backlog` `999.x` row ("remove the `query`/`unlock` rename stubs — v0.15.0") written through the roadmap tool verb (never a hand-invented shape), and the `feat!:` commit's `BREAKING CHANGE:` footer naming the removal minor — release-please copies that footer into the CHANGELOG, which is the release note. `docs/CLI-REFERENCE.md` records the same lifetime through the allowlist reason (D-12). — **Reversibility:** reversible.

### Generated surface and goldens (VERB-05, VERB-06, VERB-07)
- **D-10:** The VERB-05 census is **positive-controlled, word-boundary, multiline** (`rg -U -w`) for `codegraph query` / `codegraph unlock` across README, `docs/`, `SKILL.md`, `internal/mcp/resources/*.md`, hook scripts, `Taskfile.yml`, `.github/workflows/`, and tests — excluding `.planning/`, `CHANGELOG.md`, `web/build/`, the unrelated `internal/query` package identifier and JSON `"query"` keys/`query.json` fixture names. A planted old-verb string must be found before the "zero" result is trusted (Phase 2 D-14's discipline). Run **before** `internal/cli/` is edited (baseline: the six consumers found at discuss — `docs/CLI-REFERENCE.md` ×6 generated, `lock.go` ×2, `index.go` ×2, `index_lock_test.go` ×2, the two verb files) and **again after**.
- **D-11:** The re-freeze is a **reviewed diff, never a blanket regenerate** (Pitfall 7): `docs/CLI-REFERENCE.md` via `task docs:cli`, shell completions and man pages via their existing generators, and any CLI golden/test the fold touches. The **RED demonstration** reintroduces a *visible* `query` command (or removes the `--full` flag) and shows `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` go red, then reverts byte-clean — recorded in `03-MUTATION-LOG.md` in the Phase 2 `02-MUTATION-LOG.md` shape.
- **D-12:** The two stubs each get **one command-level allowlist line** in `testdata/cli-reference-allowlist.txt` whose reason carries the **rename target and the removal minor**, e.g. ``query — hidden rename stub for `search --full` (VERB-03); removed in v0.15.0``.
- **D-13:** The **executor reviews the re-frozen diff and commits when the gates are green** (`task docs:cli:drift`, `TestEveryRegisteredFlagIsAccountedFor`, the flag-parse test, the census); the per-file diff summary goes in the SUMMARY and the **maintainer reviews at end of phase** — the same gate shape as Phase 2 D-12. No mid-plan checkpoint. — **Reversibility:** reversible.
- **D-14:** The 8-tool MCP set (`codegraph_explore/node/search/callers/callees/impact/files/status`) and the wire-oracle transcripts are **not touched**; `task test:wireoracle` must pass byte-identically. `search --full` does **not** change `codegraph_search`'s MCP shape.

### Commit shape (VERB-08)
- **D-15:** **One `feat!:` commit** carries the surface change — adding `--full` + short flags to `search`, moving `unlock` under `daemon`, registering the two stubs, and the D-08 message updates — as `feat!(cli): fold query into search --full and unlock into daemon unlock` with a `BREAKING CHANGE:` footer stating the removed verbs, their replacements, and that the stubs are removed in the next minor (v0.15.0). Census, reference regeneration, completions/man, goldens, the backlog row and docs are ordinary `test:`/`docs:`/`chore:` commits. Under `bump-minor-pre-major` this cuts a **minor** (Pitfall 6); the CHANGELOG's BREAKING CHANGES entry is inspected after release-please runs, not asserted in-phase. — **Reversibility:** costly — the `feat!:` commit and its footer are what the published CHANGELOG and the removal schedule hang on.

### Claude's Discretion
- Exact file layout for the stubs (a `renamed.go` holding both, or one stub per old file) and the flag-parse test's location (`query_cli_test.go` → renamed to match `search`, or a new `search_cli_test.go`).
- The census script's exact `rg` invocation and exclusion list (subject to D-10's bars); whether it lives inline in the plan's `<verify>` or as a small `scripts/` file — a one-time census does **not** need a Taskfile target.
- How completions and man pages are regenerated and whether they are committed artefacts or generated at release (follow whatever `Taskfile.yml`/`release.yml` already do; do not introduce a new committed artefact).
- The `999.x` backlog row's number and wording (via the roadmap verb).
- Ordering of the ordinary commits around the single `feat!:` commit, as long as every commit leaves `task docs:cli:drift` and `go test ./internal/cli/...` green.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase framing and prior rulings
- `.planning/ROADMAP.md` §"#### Phase 3: Verb Fold" — goal, success criteria 1–5, Notes (Pitfalls 4–7; the allowlist note; `daemon unlock` verbatim move)
- `.planning/REQUIREMENTS.md` — VERB-01…VERB-08
- `.planning/research/PITFALLS.md` §Pitfall 4 (flag superset + both JSON shapes), §5 (whole-repo word-boundary census, positive control), §6 (`feat!:` cuts a minor under `bump-minor-pre-major`), §7 (reviewed re-freeze with RED)
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-CONTEXT.md` — D-12 (executor reviews, maintainer at end of phase), D-14 (positive-controlled census discipline)
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` — the mutation-log shape for `03-MUTATION-LOG.md`
- `.planning/milestones/v0.13.0-phases/12-cli-reference-docs-tail/12-CONTEXT.md` — generated docs are build artefacts with a drift gate; a guard covers only the generator's blind spot; a wording change gets no test
- `/Users/sean/.claude/rules/planning-artifacts.md` — ROADMAP backlog rows through the tool verb only

### Code this phase edits or proves
- `internal/cli/query.go` (the command being folded; `MarshalQueryJSON` call, WORK-02 notice placement, flags `-p/-k/-l/-j`), `internal/cli/search.go` (flags `--path/-p`, `--kind`, `--limit`, `--json`; `json.Marshal(locs)`), `internal/cli/unlock.go` (body to move verbatim), `internal/cli/daemon.go` (`newDaemonStartCmd`, `newDaemonStopCmd` registration — add `newDaemonUnlockCmd`), `internal/cli/root.go` (`SilenceUsage`/`SilenceErrors`; command registration), `internal/cli/man.go` (the existing `Hidden: true` precedent), `cmd/codegraph/main.go` (error → exit 1)
- `internal/query/search.go` — `Engine.Query` / `Engine.Search` (shared `matchNodes` ranking), `MarshalQueryJSON`, `queryNodeEnvelope`/`renderQueryNodeJSON`, `Location`; `internal/schema/graph.pb.go` `Node` fields (`Name`, `Kind`, `QualifiedName`, `FilePath`, `StartLine`, `Signature`)
- `internal/daemon/lock.go:208` and `internal/cli/index.go:126-128` — the live ``codegraph unlock`` messages (D-08); `internal/cli/index_lock_test.go` (asserts the message)
- `internal/cli/query_cli_test.go` (`TestQueryCmd`, `TestSearchCmd`), `internal/cli/notice_test.go` (WORK-02 notice tests), `internal/cli/cli_reference_test.go` (`TestEveryRegisteredFlagIsAccountedFor`), `tools/clidoc/main.go`, `testdata/cli-reference-allowlist.txt`, `docs/CLI-REFERENCE.md`, `Taskfile.yml` §`docs:cli`, §`docs:cli:drift`
- `internal/mcp/tools.go` (the 8 tool names — must not change), `testdata/wireoracle/transcripts/*.golden` (must stay byte-identical), `internal/goldenspec/` (CLI golden spec — check whether any capture names the verb)
- `release-please-config.json` (`bump-minor-pre-major: true`, `release-type: go`)
- `README.md`, `docs/`, `SKILL.md`, `internal/mcp/resources/*.md`, hook scripts, `.github/workflows/` — census scope (D-10)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `query.go`'s `--json` branch (`query.MarshalQueryJSON` + `writeJSONLine`) and its human branch are the exact code that moves into `search.go` behind `--full`; `search.go`'s existing `json.Marshal(locs)` branch stays as the default
- `man.go` — the one hidden command; its `Hidden: true` rationale comment is the precedent for the stub declarations
- `unlock.go` — a self-contained command (`Use: "unlock [path]"`, `MaximumNArgs(1)`, `RunE`) that moves under `daemon` as-is
- Phase 2's census discipline (planted control, `rg -U`, EXIT-trap cleanup) and `02-MUTATION-LOG.md` families — the shapes for VERB-05 and VERB-07
- `tools/clidoc` + `task docs:cli` / `docs:cli:drift` + `TestEveryRegisteredFlagIsAccountedFor` — the generated-reference drift gate the fold must keep green

### Established Patterns
- Every command sets `SilenceUsage`/`SilenceErrors`; errors propagate to `main.go` → exit 1 (the only error exit)
- WORK-02 (D-12): the worktree notice lives strictly inside the human branch, after the `--json` early return, on every read command
- Generated docs are build artefacts under a drift gate; hidden commands need a reasoned allowlist line because they still register `help`
- Goldens are re-frozen as a reviewed diff with a RED demonstration, never a blanket regenerate (v0.13.0 Phase 12, Pitfall 7)
- Conventional commits; `[ci skip]` never (rule f18zrdsgx5); local Go gates need `GOTOOLCHAIN=go1.26.6` (`cockroachdb/swiss` is `!go1.27`)
- Agent commits may use `-c commit.gpgsign=false`; the `feat!:` commit is still an agent commit unless the maintainer asks to sign it

### Integration Points
- `internal/cli/search.go` — `--full`, `-k/-l/-j`, the `eng.Query` branch, the two-line human renderer (D-01), the envelope (D-03)
- `internal/cli/daemon.go` — `newDaemonUnlockCmd` registration; `internal/cli/root.go` — the two hidden stubs replace the `query`/`unlock` registrations
- `internal/daemon/lock.go`, `internal/cli/index.go` — message text (D-08)
- `docs/CLI-REFERENCE.md`, completions, man pages, `testdata/cli-reference-allowlist.txt` — regenerated surface (D-11/D-12)
- `.planning/ROADMAP.md` `## Backlog` — the removal row (D-09) via the roadmap verb

</code_context>

<specifics>
## Specific Ideas

- `search --full` must remain a **superset** of default `search` in the human branch: the first line of every hit is the default line. That is what keeps existing pipelines and the WORK-02 notice tests working without a second format.
- The stub message should tell the user exactly what to type next (`run: codegraph search --full <term>`) and when the stub disappears (`v0.15.0`) — a redirect that costs a lookup is half a redirect.
- The census's positive control is not optional — a "zero references" result is only evidence once the planted string was found and removed.
- One `feat!:` commit, one BREAKING CHANGES entry: the rename is a single user-facing event.

</specifics>

<deferred>
## Deferred Ideas

- Making `codegraph_search` (MCP) return full records under a flag — out of scope: the 8-tool set and its shapes are frozen for this phase.
- A distinct "renamed" exit code (e.g. 2/EX_USAGE) for stubs — declined for a one-release stub; revisit only if a caller needs to distinguish rename from failure.
- Removing the stubs (v0.15.0) — tracked as the backlog row D-09 creates.

### Reviewed Todos (not folded)
- `tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone` — matched on keywords only; it is bench-runner integrity, unrelated to the verb surface. Stays pending for a later phase.

</deferred>

---

*Phase: 03-verb-fold*
*Context gathered: 2026-09-16*
