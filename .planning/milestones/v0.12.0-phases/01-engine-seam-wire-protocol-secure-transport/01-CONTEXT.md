# Phase 1: Engine Seam, Wire Protocol & Secure Transport - Context

**Gathered:** 2026-08-22
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase ships **no pixels**. It delivers the seam and the wire that every later
view sits on:

- `codegraph ui` as its own process, serving typed, bounded, **read-only** RPCs over
  a loopback listener that refuses a rebinding request
- Structured `NodeDetail` / `ExploreResult` variants extracted from `internal/query`
- A commit-aware `schema.Meta`
- A protobuf codegen drift guard covering **both** proto surfaces
- The `internal/mcp` `pendingWriter` fix

**Requirements (12):** ENG-01, ENG-02, ENG-04, RPC-01, RPC-02, RPC-05,
SRV-01, SRV-02, SRV-03, SRV-04, BLD-04, FIX-01

**Hard constraint carried from ROADMAP:** every existing CLI and MCP byte stays
unchanged, and every frozen golden covering them passes unchanged.

**Blocking order (from ROADMAP):** no RPC handler ships before SRV-02's Origin/Host
control exists; no UI view work is planned before ENG-01/ENG-02 land and are
independently proven golden-clean.

</domain>

<decisions>
## Implementation Decisions

### Engine Seam (ENG-01, ENG-02)

- **D-01:** `Node()` and `Explore()` become **thin wrappers over the new structured
  variants** — not parallel implementations. `renderSingleDefNode` keeps its exact
  output but is reduced to `RenderNode(d.Node, d.Calls, d.CalledBy)` over a
  `NodeDetail` produced by a single extracted gather. There is exactly ONE gather
  path, so CLI, MCP and UI cannot disagree about what a node *is*.
  — **Reversibility:** costly — undoing means re-inlining the gather into
  `renderSingleDefNode` and `renderMultiDefNode`, then re-proving all 26 goldens
  byte-identical a second time; the three consumers would silently re-acquire the
  ability to diverge.

- **D-02:** `NodeDetail` covers **all three** shapes `Node()` can return: file-only
  source (`renderNumberedSource`), single-def (`RenderNode`), and multi-def
  (`RenderNodeMultiDef`). Partial coverage was rejected because ENG-01 is a blocking
  gate for Phases 3–6 — a half-extracted seam would let Phase 3 discover a gap in
  something it was told was finished. Overloaded symbols are common in the Java and
  C# corpora, so the UI hits multi-def immediately.

- **D-03:** `NodeDetail` / `ExploreResult` are **plain Go structs in
  `internal/query`**, not generated protobuf types. The RPC layer owns the mapping
  to `uiv1` messages. `internal/query` — shared by CLI and MCP — stays wire-agnostic,
  exactly as it is today.
  — **Reversibility:** costly — reversing this makes `internal/query` depend on the
  UI wire format, which drags D-02a's additive-only discipline into every future
  engine refactor and couples two consumers that have no stake in the UI schema.

- **D-04:** The "goldens stay byte-identical" claim is proven by **mutation-proof
  with the count asserted**: deliberately break the extracted gather (drop
  `calledBy`), observe all 26 scenarios go RED, **record the number that failed**,
  revert. A passing suite alone cannot distinguish "nothing changed" from "nothing
  could be detected." Scoring is by counting `--- PASS` lines, **never** by exit
  status — `go test -run PATTERN` exits 0 when the pattern matches nothing
  (three phantom commands were found this way in v0.11.0).
  This satisfies rule `84d1gfpywd`: the guard carries a positive assertion that it
  did its work.

  > **⚠ CORRECTED — the "never by exit status" clause above is WRONG. Read this
  > before writing any `<verify>` gate.**
  > The sentence "Scoring is by counting `--- PASS` lines, **never** by exit status"
  > is an over-compression introduced by the orchestrator when phrasing this
  > decision, not a maintainer ruling. The maintainer selected this option for its
  > substance — mutation-proof with the count asserted. The v0.11.0 finding it cites
  > (`5pzpmvthcc`) says exit status ALONE is insufficient because
  > `go test -run PATTERN` exits 0 when the pattern matches nothing. It does **not**
  > say the status should be discarded.
  > **The correct rule is the CONJUNCTION:**
  > - the test command's exit status MUST be honored — capture it before any pipe,
  >   or use `set -o pipefail` / `${PIPESTATUS[0]}`; **and**
  > - the `--- PASS` count MUST meet its floor.
  >
  > Exit status alone is vacuous when the pattern matches nothing. The count alone is
  > vacuous when something fails — in a pipeline `$?` belongs to the LAST command, so
  > `go test … | rg -o -e '--- PASS' | wc -l` yields `wc`'s status and N passing
  > subtests plus one `--- FAIL` satisfies the gate. Cycle 2 of the plan-review
  > convergence loop found this shape in 26 gates, all authored from the uncorrected
  > sentence above.
  > **The one place the original wording holds:** while OBSERVING a deliberate
  > mutation, a non-zero exit IS the expected evidence — requiring status 0 there
  > would make this very mutation-proof unperformable. Score those by the failure
  > count. Everywhere else, use the conjunction.
  > A plan-level prohibition against the bare count-only form is carried in all 11
  > plans' `must_haves.prohibitions`.

  > **⚠ CORRECTED BY RESEARCH (01-RESEARCH.md) — read before planning D-04.**
  > This decision was taken believing the 26 frozen goldens already act as an
  > output-regression net. **They do not.** Verified in source:
  > `TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243-297`) checks only
  > that each golden file exists, is non-empty, begins with `{`, parses as
  > `goldenCapture`, and has a non-empty `Output` field — it never re-invokes
  > `Engine.Node`/`Explore`. And `TestCorpusBehavior_Go`'s doc comment
  > (`testdata/golden/behavioral_test.go:685-691`) states it asserts "named
  > behavioral properties of live engine output, **not byte-diffs against a frozen
  > golden**", the TS-era capture path having been retired in FIXT-04.
  > **Nothing byte-diffs live output against the frozen goldens.**
  > Consequences the planner MUST absorb:
  > 1. D-04's mutation-proof cannot go RED against a comparison that does not exist.
  >    **A new live-output-vs-frozen-golden comparison test is Wave-0, must-add
  >    scope** — write it, prove it RED, and only then is D-04 performable.
  > 2. ROADMAP success criterion 1's "every frozen golden covering them passes
  >    unchanged" is, as written, satisfiable WITHOUT the output being unchanged —
  >    a vacuous criterion of exactly the shape rule `84d1gfpywd` names.
  > 3. What DOES exist and is worth preserving: `behavioral_test.go` invokes
  >    `eng.Node(...)` / `eng.Explore(...)` live (`:874`, `:909`, `:931`, `:961`,
  >    `:1043`, `:1051`, `:1282`, `:1326`, `:1378`) and asserts named properties.
  >    That is a property net, not a byte net; treat it as complementary, not
  >    a substitute.

### Commit-Aware Meta (ENG-04)

- **D-05:** The indexed commit SHA is added to `schema.Meta` as **field 8**,
  following the `has_file_index = 7` precedent — absent means a pre-upgrade graph
  and degrades gracefully. Fields 1–7 are occupied; there is no drift between
  `graph.proto` and `graph.pb.go` today.
  — **Reversibility:** one-way — under D-02a, once field 8 ships its number can
  never be renumbered or reused; retiring it requires adding 8 to a `reserved`
  clause, not deleting it. The on-disk format is a published contract.

- **D-06:** The SHA is surfaced **only via the `Status` RPC — `codegraph status`
  output is untouched.** `status.go` already projects `Meta` selectively (its own
  doc notes `edgesByKind` is "deliberately NOT stored in Meta"), so adding a field
  does not leak into CLI output on its own. Keeping it out matters because **no
  golden covers `status`** — all 26 scenarios are node/explore — so a CLI change
  there would be silent. Surfacing it in the CLI later is additive.

### Process & Launch (SRV-01, SRV-03)

- **D-07:** `codegraph ui` binds **ephemeral `127.0.0.1:0`** and prints the resulting
  URL. There is **no `--port` flag and no `--host` flag in v1** — literally honoring
  SRV-03's "neither is exposed in v1." A second `codegraph ui` simply gets a
  different port, so port-collision handling is code that never needs writing.
  Accepted cost: no stable/bookmarkable URL, and Phase 3 deep links live only as
  long as the process.

- **D-08:** The bind address lives as a **field on a server options struct**
  (defaulting to `127.0.0.1:0`) that **nothing wires from outside in v1** — no flag,
  no env var, no config file. Adding `--port` or `CODEGRAPH_UI_ADDR` later is wiring
  one existing field. This is SRV-03's "1, but do not preclude 4" applied to the
  port as well as the host.
  **Explicitly rejected:** a config file. This repo has no user-config mechanism —
  no viper, no rc file; `.codegraph/` holds only the store and the daemon lock. Its
  established override convention is env vars paired with flags (`serve.go:300`
  documents `--watch` as "the CLI twin of `CODEGRAPH_FORCE_WATCH=1`"). Introducing a
  config file is a milestone-sized decision and must not arrive as a side effect of
  a port question.

- **D-09:** The browser **opens by default**, with `--no-open`, **and** automatic
  suppression when stdout is not a TTY or `CI` is set. SRV-01 requires opening; the
  suppression keeps scripted and CI invocations safe without depending on the caller
  remembering a flag. Needs a small cross-platform open helper
  (`open` / `xdg-open` / `rundll32`) — **none exists in the tree today.**

- **D-10:** The process runs **foreground until Ctrl-C**, matching `serve` and
  `daemon start`. No PID file, no `ui stop` verb, no orphan-process class of bug.
  Detaching was rejected specifically because it would rebuild the lifecycle surface
  `daemon` already owns — which SRV-01 says must not be shared.

### Bounded Responses (RPC-05)

- **D-11:** A verbatim source response is bounded by a **line cap as the primary
  limit**, with a **byte cap underneath it** — a pathological minified single-line
  file blows any byte budget regardless, so the byte cap is not optional.
  **Consequence to plan for:** three limits now have to stay mutually consistent
  (line cap, byte cap, transport backstop). Follow `internal/mcp/session_line.go:18`'s
  discipline — each limit is one named constant "referenced from both the truncation
  call and the test." Byte truncation MUST land on a UTF-8 rune boundary; source is
  UTF-8 and `session_line_test.go:74,247` already assert this property for the
  existing truncation path.

- **D-12:** Truncation is signalled by an **explicit field on the response**
  (`truncated` plus the true total), never by an error. The client still receives
  usable content and can render "showing first N of M" — which is what RPC-05's
  "comes back bounded and explicitly marked truncated" literally asks for. Returning
  `CodeResourceExhausted` was rejected: it prevents an unbounded response but does
  not deliver a bounded one.
  — **Reversibility:** one-way for the field numbers (D-02a); costly for the
  semantics, since every Phase 3–6 source view reads them.

- **D-13:** **Both `WithSendMaxBytes` and `WithReadMaxBytes` are set** as a transport
  backstop above application truncation. `connect-go` defaults to **unlimited** on
  both sides ("Both clients and handlers default to allowing any message size"), so
  RPC-05 is satisfied by no default. If application truncation is ever missed, the
  failure is a clean `resource_exhausted` rather than an unbounded write or an OOM.
  The backstop MUST sit above the application limit or it masks correct behavior as
  an error.

### Degraded State (SRV-04)

- **D-14:** A store that stays locked past `graphstore.Open`'s retry budget is
  reported as **`CodeUnavailable` carrying a typed `IndexingInProgress` detail
  message**; the client special-cases the code and renders the state. This is the
  code+detail shape Connect's protocol reference documents, and `unavailable` is
  defined as "transient, back off and retry" — `failed_precondition` is explicitly
  wrong here because the user fixes nothing; it resolves itself.
  SRV-04's "never as an error" is read as **never surfaced to the human as an
  error**, which is the property the requirement protects.
  — **Reversibility:** costly — every Phase 3–6 view special-cases this code;
  switching later to a state field on every message means touching every view.

- **D-15:** **No retry beyond `Open`'s existing bounded budget.** `Open`'s loop
  (`pebble_store.go:132-141`) is already the bounded, never-block answer, and SRV-04
  describes precisely the case where a re-index *outlasts* that budget. A second
  retry layer would change the very condition the requirement names and make it
  harder to demonstrate. The UI polls, or from Phase 6 is pushed to.

- **D-16:** **`Status` still answers when the store cannot be opened**, reporting
  what is derivable without opening it (repo path, store-exists, lock-held,
  "indexing in progress") and degrading the graph-derived counts. The health view
  answering "the index is busy" *is* a health answer, and Phase 4's index-health
  verdict depends on `Status` being reachable exactly when things are wrong. This is
  the one place the partial-availability pattern genuinely applies — everywhere else
  a failed `Open` leaves no partial data to return.

### Claude's Discretion

Planning may settle without returning to the user:

- The exact fix mechanics for FIX-01 — whether `pendingWriter` learns to distinguish
  server-initiated notifications from client responses, or the counter moves so only
  response writes decrement. The bug itself is confirmed: `pendingWriter.Write`
  (`internal/mcp/server.go:339`) calls `p.pending.Add(-1)` on **every** write, while
  only client-initiated request lines increment it (`stdinLingerReader:276`).
- The BLD-04 guard's shape — regenerate-into-tempdir-and-diff vs
  regenerate-in-place-and-`git diff --exit-code`, and whether a new `task proto`
  becomes a required CI check. **Constraint:** it must report **how many generated
  files it actually compared** and be watched fail against a deliberately stale
  checked-in file (success criterion 4, and rule `84d1gfpywd`).
- Exact field names, and whether the truncation total is expressed in bytes or lines.
- `connectrpc.com/connect` version pinning. It is **not** currently a dependency;
  `google.golang.org/protobuf v1.36.11` already is, directly.

### Folded Todos

Three pending todos were folded into this phase. **Note for planning:** only the
first maps onto an existing requirement; the other two have **no requirement ID**
and will need either a task under an existing requirement or a roadmap addition.

1. **Wire oracle `toolslist-repeat` response ordering flake**
   (`todos/pending/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`,
   area `mcp`, score 0.9) — two recorded sightings; the diff was `"id":3` vs `"id":2`
   with payloads otherwise identical (arrival order, not content), and a re-run of
   the identical commit went green. It freezes JSON-RPC arrival order, which the
   protocol does not guarantee. Folded on the hypothesis that it shared FIX-01's root
   cause, with an obligation to **prove it either way**.

   > **⚠ HYPOTHESIS FALSIFIED BY RESEARCH (01-RESEARCH.md). The obligation is
   > discharged — as a DISPROOF. Do not plan these as one fix.**
   > Verified against `github.com/modelcontextprotocol/go-sdk@v1.7.0` in the local
   > module cache: every JSON-RPC call except `initialize` has `jsonrpc2.Async(ctx)`
   > invoked on it (`mcp/server.go:1908-1913`, citing upstream `go-sdk#26`). Two
   > pipelined `tools/list` calls therefore run in **separate goroutines with no
   > response-ordering guarantee**. That is a different defect from `pendingWriter`
   > decrementing a counter it never incremented (`internal/mcp/server.go:339-343`
   > vs `:276`).
   > **Planner guidance:** FIX-01 remains scoped to `pendingWriter` alone and is the
   > only half backed by a requirement. The flake is a *separate* finding whose fix
   > belongs in the **wire oracle**, which freezes an arrival order the protocol
   > never promised — not in the server. Size it as its own small task or hand it
   > back to its todo; either is defensible, but it must not be folded INTO FIX-01,
   > and closing FIX-01 must not be reported as closing the flake.

2. **Add golangci-lint with gofmt and idiomatic Go linters**
   (`todos/pending/2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md`,
   area `ci`, score 0.9) — folded on the reasoning that Phase 1 adds a new package
   (connect handlers) and a new codegen path. **No backing requirement.**

3. **`post-release-verify.yml`'s event-aware conclusion guard has no regression
   assertion**
   (`todos/pending/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md`,
   area `ci`, score 0.9) — a silent-regression risk in the release pipeline.
   **No backing requirement**, and this phase does not otherwise touch the release
   path.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope and requirements
- `.planning/ROADMAP.md` §"Phase 1" — goal, 5 success criteria, blocking order, and
  the Notes clarifying that ENG-01/02 are extractions, not rewrites
- `.planning/ROADMAP.md` lines 91–96 — "Ordering is load-bearing, not stylistic",
  including the warning that **two distinct findings are both named CR-01**: this
  milestone's is `internal/mcp/server.go`'s `pendingWriter` (open, scoped as FIX-01);
  the CR-01 cited throughout `internal/graphstore/pebble_store.go` is a v0.5.0
  Phase-3 Pebble-lock finding that is **already fixed** — the bounded retry loop *is*
  that fix. Never conflate them.
- `.planning/REQUIREMENTS.md` lines 15–18, 23–27, 31–34, 88, 95 — the 12 requirement
  texts for this phase
- `.planning/PROJECT.md` — core value, constraints, the retired Compatibility
  constraint

### Engine seam (ENG-01, ENG-02)
- `internal/query/node.go` §316–460 — `Node()`'s three output shapes and the
  `renderSingleDefNode` / `renderMultiDefNode` gather→render boundary that D-01 splits
- `internal/query/explore.go` §240 — `Explore()`, the ENG-02 twin
- `internal/query/render_markdown.go` §127, §206, §360 — `RenderNode`,
  `RenderNodeMultiDef`, `RenderExplore`: the pure render functions the extraction
  must stop before
- `internal/query/gather.go` — existing candidate-limiting precedent (§187–189)

### Golden suite (the ENG-01/ENG-02 safety net)
- `testdata/golden/golden_test.go` — the harness; §255–258 special-cases the
  behavioral slug
- `testdata/golden/README.md` — suite conventions
- `testdata/golden/corpus/` — 24 scenarios: 4 corpora (guava, hugo, requests,
  serilog) × 6 slugs (`go-node`, `go-node-multi`, `go-node-mcp`, `go-explore`,
  `go-explore-multi`, `go-explore-mcp`)
- `corpus/behavioral/` — 2 further scenarios (`go-node-multi`, `go-explore-multi`)
- **Measured fact:** 24 + 2 = **26**, and **every** scenario covers node or explore.
  The ENG-01/ENG-02 blast radius is 100% of the golden suite. The `-mcp` slugs mean
  the suite already covers two consumers, so the UI is genuinely the third.

### Schema and codegen (ENG-04, RPC-01, BLD-04)
- `internal/schema/graph.proto` §137–151 — the `Meta` message; fields 1–7 occupied,
  `has_file_index = 7` is the additive precedent ENG-04 follows
- `internal/schema/meta.go` — `SchemaVersion`, `NewMeta()`, `IsCurrentSchemaVersion()`,
  and the D-02a additive-only discipline stated in full in the package comment
- `internal/schema/graph.pb.go` — the committed codegen BLD-04 must prove current
  (verified in sync with `graph.proto` at the time of this discussion)
- `internal/schema/roundtrip_test.go` — tests **serialization, not codegen currency**;
  it is not the drift guard BLD-04 needs
- `Taskfile.yml` — **has no proto regeneration task today.** BLD-04 closes this
  pre-existing gap.

### Store lifecycle and degraded state (SRV-04)
- `internal/graphstore/pebble_store.go` §67–160 — `openLockRetryAttempts`,
  `openLockRetryBackoff`, `ErrStoreLocked` (§110), `classifyOpenError` (§112), and
  `Open`'s bounded retry loop (§141). Classification happens **exactly once**, inside
  the loop.
- `internal/cli/serve.go` §241–246 — existing `errors.Is(err, graphstore.ErrStoreLocked)`
  handling and its comment on what a broader check would paper over
- `internal/daemon/daemon.go` §297, §431 — the second existing `ErrStoreLocked`
  consumer
- `internal/query/status.go` §30–45, §137–143 — how `Status` projects `Meta`
  selectively, `computeStale`, and the note that `edgesByKind` is deliberately not
  stored in `Meta`. Relevant to both D-06 and D-16.

### Bounding and truncation (RPC-05)
- `internal/mcp/session_line.go` §18–50 — the established truncation pattern: one
  named constant referenced from both the call and the test, cutting on a UTF-8 rune
  boundary
- `internal/mcp/session_line_test.go` §74, §247 — the assertions that truncation never
  splits a rune
- `internal/query/node.go` §81–84 — `readSourceFile`, repo-root-confined
  (`T-03-06-Path`); the UI source path must go through it, not a new read path

### MCP fix (FIX-01)
- `internal/mcp/server.go` §214–349 — `pendingWriter` (§334–349), `stdinLingerReader`
  (§261–300), and the design comments explaining the intended invariant. The bug:
  `Write` decrements on every write (§341) while only client-initiated request lines
  increment (§276).
- `todos/pending/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`
  — the folded todo, two sightings recorded

### Wire protocol (RPC-01, RPC-02)
- `connectrpc.com/connect` `option.go` — `WithSendMaxBytes` / `WithReadMaxBytes`;
  **both default to unlimited**
- `connectrpc.com/connect` `envelope.go` — the cap returns
  `CodeResourceExhausted`, it does **not** truncate. This is why D-11/D-12
  (application truncation) and D-13 (transport backstop) are two separate things.
- Connect protocol reference, error-code table — `unavailable` → 503, "transient,
  clients should back off and retry"; and the `{"code": ..., "details": [...]}`
  typed-detail shape D-14 uses
- `internal/cli/serve.go` §179–301 and `internal/cli/daemon.go` §51–240 — command
  structure, flag conventions (`--path/-p`), and the foreground-serving precedent
  D-10 follows

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`RenderNode` / `RenderNodeMultiDef` / `RenderExplore`** are already pure functions
  of their arguments. The "view model" the extraction needs is the argument tuple
  they already take — `NodeDetail` is `(node, calls, calledBy)`, nothing invented.
- **`graphstore.ErrStoreLocked`** is an exported, `errors.Is`-able sentinel with two
  existing consumers. The classification seam D-14 needs already exists; the RPC layer
  is a third consumer of it, not a new mechanism.
- **`internal/mcp/session_line.go`'s truncation pattern** — named-constant limit,
  rune-boundary cut, test asserting no split runes. Directly transferable to RPC-05.
- **`readSourceFile`** (`node.go:84`) is repo-root-confined and already the safety gate
  both `Node()`'s file mode and multi-def use. The UI must reuse it.
- **`google.golang.org/protobuf v1.36.11`** is already a direct dependency; protobuf
  tooling is in-tree, so the UI schema is a **second** proto surface, not a greenfield one.

### Established Patterns
- **Additive-only proto evolution (D-02a)** is stated in `internal/schema/meta.go`'s
  package comment and enforced by convention: field numbers are never renumbered or
  reused; retiring a field means a `reserved` clause. The UI schema inherits this.
- **Flags paired with env twins** — `serve.go:300` documents `--watch` as "the CLI
  twin of `CODEGRAPH_FORCE_WATCH=1`". There is **no config-file mechanism** anywhere
  in the repo.
- **Foreground long-running commands** — both `serve` and `daemon start` run in the
  foreground; only `daemon` carries PID/lock lifecycle, and SRV-01 forbids sharing it.
- **`--path/-p` on every repo-scoped command** — `codegraph ui` should follow.
- **Guards carry positive assertions** (rule `84d1gfpywd`) — the repo has 49 such
  non-vacuity guards, 19 in `taskfile_shape_test.go` alone. D-04 and BLD-04 both
  inherit this standard.

### Integration Points
- `internal/cli/` — a new `ui.go` alongside `serve.go` / `daemon.go`, registered on
  the same Cobra root (`root.go`)
- `internal/query/` — the extraction point; new structured types live here, wire-agnostic
- `internal/schema/graph.proto` — `Meta` field 8
- A new UI proto package + generated `connect` handlers, mounted on `net/http`
  alongside the (Phase 2) `go:embed`'d SPA
- `internal/mcp/server.go` — the FIX-01 change, isolated from everything else here
- `Taskfile.yml` — a new proto regeneration task, which does not exist today

</code_context>

<specifics>
## Specific Ideas

- **"Ephemeral now, but leave room."** The maintainer's exact framing for the port
  decision: pick ephemeral for v1, but shape the code so a later `--port` flag or
  config entry is an override of something that already exists rather than a
  restructuring. Recorded as D-07 + D-08. The "config entry" half was answered with
  the options-struct field rather than a config file, after the repo was shown to have
  no config mechanism.
- **The golden suite must be shown able to fail, not merely shown passing** (D-04) —
  continuing the v0.11.0 precedent where re-baselined goldens were re-proven
  non-vacuous rather than trusted because they were green.
- **`Status` should still answer when things are broken** (D-16) — the health view
  going blank exactly when the user needs it was the deciding argument.

</specifics>

<deferred>
## Deferred Ideas

- **Stable/bookmarkable UI URL.** D-07's ephemeral port means Phase 3's deep links
  live only as long as the process. If Phase 3 finds that unacceptable in practice,
  the fix is wiring D-08's existing options field — a change, not a rewrite.
- **Surfacing the indexed commit SHA in `codegraph status`.** Deliberately excluded
  from v1 by D-06 to keep the "CLI bytes unchanged" criterion unarguable. Additive
  later, but it would need its own guard — no golden covers `status`.
- **Byte-range paging for source responses.** Considered and rejected for RPC-05 in
  favor of line-capped truncation; it would have changed what Phase 3's viewer must
  build. Revisit only if truncation proves insufficient in a real view.
- **A user config file mechanism.** Explicitly declined as out of scope for a port
  question (see D-08). If it is ever wanted, it is a milestone-sized decision.

### Known limitation carried forward — the sixth prohibition's generality (H-9-1)

**Status: accepted, not fixed. No gate in Phase 1 exploits it. Recorded for whichever phase
next authors gates.**

The sixth (semantic) prohibition carried in all 11 Phase-1 plans still does not entail its own
headline — *"Never write a gate whose floor can be met without the deliverable it names existing."*
Two independent reviewers (the cycle-9 review agent and Codex, without contact) constructed the same
counterexample. The diagnosis is structural:

- **Half (a) OWNERSHIP AND CAUSATION is causal but NOT exhaustive** — it requires a no-op replacement
  to turn the leg RED, but names only *one* artifact under test.
- **Half (b) HOMING is exhaustive but NOT causal** — it quantifies over every behavior bullet, but
  requires a bullet to *name* a counted test, never to be *asserted* by it.

Neither half is both, so their conjunction is neither, while the headline requires both. A task with a
**plural** deliverable therefore satisfies every clause literally by naming one peripheral artifact for
(a) and nominally homing the rest to inert subtests. The `STATUS -eq 0` conjunction does not rescue it:
a subtest asserting nothing load-bearing cannot fail, so it contributes its `--- PASS` line and its
name while staying causally inert.

**Why it was not fixed here.** This would have been the *third* textual amendment in three cycles —
cycle 6 added the prohibition, cycle 8 made half (a) causal, and each closed its predecessor's
**instance** without closing the **pattern**. A fourth amendment has a poor prior. Maintainer ruling:
accept and record.

**The proposal to adopt instead, from the cycle-9 reviewer.** Replace the textual rule with a
per-task **no-op matrix** — a bounded planning-time artifact, one row per deliverable, one column
naming the test that goes RED when that deliverable is replaced by a no-op. It is structural rather
than textual, so a naming trick cannot satisfy it: a table that must enumerate every deliverable
cannot be satisfied by naming one. If a later phase adopts this, it supersedes the sixth prohibition
rather than amending it again.

**What the phase's own gates do today**, verified: the two multi-artifact form-A1 derivations actually
written name several artifacts each and trace the no-op consequence for all of them. The gap is in what
the rule *permits* a future author to write, not in what this phase wrote.

Full text, the exact admitting gate, and the clause-by-clause admission argument are in
`01-REVIEWS.md`'s cycle-9 section (commit `4fc5176`).

### Reviewed Todos (not folded)

- **`release:dry-run-signed`'s additions-only diff guard passes vacuously when the
  awk anchor stops matching**
  (`todos/pending/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md`,
  area `release`, score 0.6) — matched on keyword overlap; concerns the release
  pipeline, which this phase does not touch. A genuine instance of rule `84d1gfpywd`,
  but not this phase's.
- **Tap App secret-distinctness test is tautological — it compares two in-test
  constants and reads no workflow**
  (`todos/pending/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md`,
  area `testing`, score 0.6) — concerns the Homebrew tap workflow; out of scope.

</deferred>

---

*Phase: 1-Engine Seam, Wire Protocol & Secure Transport*
*Context gathered: 2026-08-22*
