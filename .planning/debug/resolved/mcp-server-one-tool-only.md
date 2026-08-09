---
status: resolved
trigger: "the mcp server is only showing one tool."
created: 2026-08-08
updated: 2026-08-09
resolved: 2026-08-09
---

## Resolution

root_cause: >-
  (AND-gate, three simultaneous contributing causes.) (1) SPEC: the accepted
  contract made codegraph_explore the only default-visible tool, gated behind
  an opt-in CODEGRAPH_MCP_TOOLS allowlist that no user-facing artifact named.
  (2) DOCUMENTATION: README.md asserted "Eight tools are exposed" directly
  beneath the install snippet — false for every default installation, and the
  origin of the reported expectation. (3) WIRE CONTRACT: the MCP instructions
  string attributed tool visibility SOLELY to index state, so the one
  explanation offered to the agent at the moment of confusion was the one
  cause that structurally cannot produce a 1-tool list (a missing index
  produces ZERO). Any one alone is latent; together they produce a user who
  concludes the server is broken. Secondary, independently confirmed:
  toolAnnotations() hardcoded readOnlyHint:false / destructiveHint:true /
  openWorldHint:true for all 8 tools, so a read-only local graph query
  advertised itself in a host's risk vocabulary as "destructive, open-world".

fix: >-
  Per user decision, the remedy inverted the spec rather than narrowing the
  docs. CODEGRAPH_MCP_TOOLS becomes an opt-out NARROWING filter: unset
  registers all 8 tools, set registers only the named companions plus the
  always-visible codegraph_explore, set-to-empty registers explore alone.
  MCP-03 (no index => zero tools) is untouched. Read via os.LookupEnv, not
  os.Getenv, because "unset" and "set to empty" must now answer differently.
  The unknown-name warning gains a consequence line naming the surviving
  surface. All 8 tools annotated readOnlyHint:true / openWorldHint:false with
  destructiveHint omitted, after a per-tool audit of the reachable call
  graph. instructions, README and `serve --help` rewritten to state all three
  visibility mechanisms, and gated by instructions_contract_test.go so the
  SURF-01 bug class (a wire claim drifting from behavior with nothing
  comparing them) cannot recur at the server level.

files_changed:
  - internal/mcp/server.go
  - internal/mcp/tools.go
  - internal/cli/serve.go
  - README.md
  - internal/mcp/instructions_contract_test.go
  - internal/mcp/tool_annotations_test.go
  - internal/mcp/server_test.go
  - test/wireoracle/scenarios.go
  - test/wireoracle/capture.go
  - test/wireoracle/oracle_test.go
  - test/wireoracle/COVERAGE-BASELINE.md
  - testdata/golden/mcp-capture.mjs
  - testdata/wireoracle/transcripts/ (25 changed, 1 added, 1 renamed)

oracle_type: >-
  derived (contract) for the tool-set and annotation assertions — expectations
  are read from companionNames/allToolNames(), this package's own source of
  truth, never re-typed. Deliberately WEAKER for
  TestInstructionsDescribesEveryVisibilityMechanism: three literal anchors
  ("default", CODEGRAPH_MCP_TOOLS, "codegraph init"), recorded as such rather
  than dressed up, because there is no runtime value to derive "explains its
  own default" from and the alternative to three literals is the no-gate
  state that produced this bug.

verification:
  guardrail_verdict: accepted
  signal_regression_test: >-
    PASS. New: TestDefaultToolVisibility (retargeted),
    TestEmptyToolFilterNarrowsToExploreOnly,
    TestEveryToolIsAnnotatedReadOnlyClosedWorld,
    TestInstructionsDescribesEveryVisibilityMechanism, wire scenarios
    toolslist-default (8 tools) and toolslist-filter-empty (1 tool).
    Boundary neighbours covered: unset vs set-empty (the os.LookupEnv
    distinction), 0/1/all companions in the doc-checker non-vacuity cases.
  signal_mutation: >-
    PASS, 5 mutations, each reverted immediately.
    (1) ResolveCompanions ignoring `present` => TestDefaultToolVisibility +
    TestEmptyToolFilterNarrowsToExploreOnly FAIL.
    (1b) the same mutation at the WIRE level => TestToolsListExactSets/
    toolslist-default FAIL with "advertised [codegraph_explore]" — the
    reported symptom reproduced verbatim by reverting the fix, which is the
    strongest available evidence the fix addresses the reported cause.
    (2) annotations restored to destructive/open-world => annotation test
    FAILS on all 8 tools.
    (3) CODEGRAPH_MCP_TOOLS removed from instructions => both instruction
    contract tests FAIL.
    (3b) the missing-index clause removed => mechanism test FAILS.
    (4) README narrowing section removed => README gate FAILS.
  signal_full_suite: >-
    PASS. `task test` green end to end, including test:golden,
    test:integration, test:wireoracle, test:daemon and the -race leg over
    daemon/watch/cli/mcp. `task lint` (go vet + actionlint) clean.
  signal_diff_shape: >-
    Not deletion-only. Adds a resolution seam (ResolveCompanions), a
    consequence-bearing warning, two new test files, one new wire scenario.
  signal_transcript_control: >-
    check:transcript-freeze fires ADVISORY as designed (it cannot gate this —
    its own remedy of splitting into two PRs is structurally unavailable
    here). The mandated control was applied instead: capture before and
    after, classify mechanically per transcript, name every cause in the
    commit message. 25 changed / 1 new / 3 unchanged, and the 3 that did NOT
    move are themselves evidence the three causes are scoped as claimed.
  signal_live_binary: >-
    PASS. Freshly built binary driven over real stdio against THIS repo (the
    one that produced the report): unset => 8 tools; "node,status" => 3;
    empty => 1; all-typo value => 1 plus the new stderr consequence line
    "0 of 7 companion tools will register ... Unset the variable entirely to
    register all 8 tools". Session line reads tools=8. Every tool advertises
    {"idempotentHint":false,"openWorldHint":false,"readOnlyHint":true}.

# Debug: MCP server exposes only one tool

## Symptoms

**Expected behavior**
The `codegraph` MCP server exposes its full agent-facing tool surface. The
codebase defines 8 tools total: the always-visible `codegraph_explore` plus 7
companions (`node`, `search`, `callers`, `callees`, `impact`, `files`,
`status` — see `internal/mcp/server.go:58` `companionNames`). The server's own
MCP `instructions` string — the text shipped to clients and visible in this
Claude Code session — tells agents: *"An empty tool list means this repository
has no index yet, so run codegraph init to fix it, not that the server is
broken. Tools appear automatically once an index exists, with no client restart
required."* That sentence promises **index-gated** visibility.

**Actual behavior**
Claude Code's `/mcp` tool listing for the `codegraph` server shows:

```
Tools for codegraph
1 tool

  1. codegraph_explore   destructive, open-world
```

Only `codegraph_explore`. This repo HAS an index (`.codegraph/store/` present,
`daemon.lock` live as of 2026-08-08 21:02), so the "no index yet" explanation
in the instructions does not apply.

Secondary observation from the same screenshot: `codegraph_explore` is
annotated **`destructive, open-world`**. A read-only graph query should
plausibly advertise `readOnlyHint: true` / `destructiveHint: false`. Worth
confirming whether the annotations are deliberate or a missing-annotation
default.

**Error messages**
None. No crash, no stderr surfaced to the client. Silent under-registration.

**Timeline**
Unknown whether this ever showed >1 tool in this client. The MCP surface last
changed in `13f2875 feat(mcp): protocol currency — official go-sdk v1.7.0 and
2026-07-28 spec compliance (#24)`.

**Reproduction**
Open `/mcp` in Claude Code with the `codegraph` server connected against this
repo and view the tool list.

## Evidence

- timestamp: 2026-08-08 (orchestrator pre-flight, unverified by the debugger)
  finding: `internal/cli/serve.go:23-26` — `CODEGRAPH_MCP_TOOLS` is documented
  as "the operator allowlist env var (D-08a, MCP-02): a comma-separated list of
  companion tool names to register alongside the always-visible
  codegraph_explore."
- timestamp: 2026-08-08
  finding: `internal/cli/serve.go:240` — `allowlist, unknown :=
  mcp.ParseAllowlist(os.Getenv(codegraphMCPToolsEnv))`. Env unset ⇒ empty
  allowlist ⇒ zero companions registered. **One tool is therefore the designed
  default, not a failure path.**
- timestamp: 2026-08-08
  finding: `docs/MCP-8-AGENT-AUDIT.md:70` — "`CODEGRAPH_MCP_TOOLS` was not set
  for this audit, matching how each client's [default registration works]".
  Corroborates that shipped-default == 1 tool.
- timestamp: 2026-08-08
  finding: `internal/mcp/server_test.go:135` expects exactly
  `[codegraph_explore]`; `:165` expects `[codegraph_explore, codegraph_node,
  codegraph_status]` under an allowlist. The tests PIN the env-gated behavior,
  so any fix that changes default visibility must also move these expectations.
- timestamp: 2026-08-08
  finding: The repo's `.mcp.json` (untracked, project scope) registers only
  `gsd-workflow` and `gsd-browser` — **no `codegraph` entry**. The `codegraph`
  server is therefore registered at user scope (`~/.claude.json`) and its
  registration is the place an `env` block would have to live.

## Eliminated

- hypothesis: "No index exists, so tools are correctly hidden."
  eliminated_because: `.codegraph/store/` exists with a live `daemon.lock`;
  `codegraph_explore` itself works, which requires a resolvable index.

## Evidence (continued — this session)

- timestamp: 2026-08-08
  checked: `jq '.mcpServers.codegraph' ~/.claude.json`
  found: `{"args":["serve","--mcp"],"command":"codegraph","type":"stdio"}` —
  no `env` block. No project-scope override exists either.
  implication: `CODEGRAPH_MCP_TOOLS` is genuinely unset at server startup.
- timestamp: 2026-08-08
  checked: `.planning/milestones/v0.1-REQUIREMENTS.md:45-47`
  found: **MCP-01**: "`codegraph_explore` as the only default-visible tool"
  (marked `[x]` Complete). **MCP-02**: companions via `CODEGRAPH_MCP_TOOLS`.
  **MCP-03**: zero tools with no `.codegraph/`.
  implication: **DECISIVE.** Explore-only-by-default is the accepted,
  specified contract. `serve.go:240` + `registerTools` implement it exactly.
  There is no code defect in the gating, and `install` omitting an `env`
  block is correct-by-design, not a bug.
- timestamp: 2026-08-08
  checked: `README.md:91-93`
  found: "Eight tools are exposed: `codegraph_explore`, `codegraph_node`,
  `codegraph_search`, `codegraph_callers`, `codegraph_callees`,
  `codegraph_impact`, `codegraph_files`, `codegraph_status`." — placed
  directly beneath the `codegraph install` snippet. `CODEGRAPH_MCP_TOOLS`
  appears NOWHERE in README.md.
  implication: The README is flatly false for every default installation and
  is the direct origin of the reported expectation.
- timestamp: 2026-08-08
  checked: `rg 'CODEGRAPH_MCP_TOOLS'` across all user-facing surfaces
  found: Zero hits in README.md, docs/FLAG-PARITY.md, or any `--help` text
  (`serve` has `Short` only — no `Long`, no `Example`, no env-var note).
  Its only appearances are a Go code comment, `docs/MCP-8-AGENT-AUDIT.md`
  (internal audit), and test fixtures.
  implication: A user who sees one tool has no documented remedy anywhere.
- timestamp: 2026-08-08
  checked: `internal/agents/instructions.go:17-18`
  found: The marker block install writes "explicitly defers full tool
  guidance to the MCP initialize response (Phase 3)".
  implication: A documented hand-off that was never completed — the
  initialize response's `instructions` never covered tool guidance, so both
  ends of the hand-off point at each other and neither names the allowlist.
- timestamp: 2026-08-08
  checked: `internal/mcp/tools.go:79-96` `toolAnnotations()`
  found: `ReadOnlyHint:false, DestructiveHint:true, OpenWorldHint:true`
  hardcoded for ALL 8 tools, with a comment stating these are inherited
  mark3labs zero-values, "arguably wrong for read-only query tools", and
  deliberately deferred to Phase 3.
  implication: The "destructive, open-world" display is DELIBERATE deferred
  debt with a paper trail — not a fresh defect. Phase 3 (v0.3.0) shipped
  without doing it. Report; do not fold into this fix.
- timestamp: 2026-08-08
  checked: `internal/mcp/tools_schema_drift_test.go:34-41`
  found: SURF-01's prior escape — "Phase 8 changed defaultDepth 5→2 and
  updated CLI help and docs/FLAG-PARITY.md, but internal/mcp/tools.go kept
  advertising 'default 5', so MCP agent clients were told the wrong default
  for a whole phase."
  implication: **Same bug class, second occurrence.** A wire-contract claim
  drifting from actual behavior with no gate comparing them. The existing
  test guards numeric claims in tool *descriptions*; nothing guards the
  visibility claim in the `instructions` string or in README.
- timestamp: 2026-08-08
  checked: `Taskfile.yml:1344` `check:transcript-freeze` + 24 of 28 golden
  transcripts contain the `instructions` string verbatim.
  implication: Changing `instructions` churns 24 goldens. The freeze guard
  is ADVISORY (never fails the build); the mandated control is a
  reviewed-diff pass with every cause named in the commit message.

- timestamp: 2026-08-08 (LIVE CONFIRMATION — orchestrator client, empirical)
  checked: Set `CODEGRAPH_MCP_TOOLS` in the live client's `codegraph` server
  registration and observed the tool list WITHOUT restarting the server.
  found: All 7 companion tools appeared immediately; no client restart and no
  server restart were required.
  implication: **Root cause confirmed empirically, not just by code reading.**
  Two independent facts established: (a) the env var is the sole gate on
  companion visibility — the 1-tool list was the allowlist being empty, nothing
  else; (b) the `notifications/tools/list_changed` path demonstrably works, so
  a dynamic change to the registered tool set does reach a connected client.
  (b) matters for the fix: the instructions string's claim that tools appear
  "with no client restart required" is TRUE as a mechanism — it was only
  attached to the wrong trigger (index state instead of the allowlist).

## Decision — default inverted (supersedes MCP-01)

decided: 2026-08-08 (user)
change: >-
  Default (CODEGRAPH_MCP_TOOLS unset) registers ALL 8 tools, provided an index
  exists. CODEGRAPH_MCP_TOOLS inverts from an opt-in allowlist to an opt-out
  NARROWING filter: when set, only the named companions register, plus the
  always-visible codegraph_explore. MCP-03 is UNCHANGED — no `.codegraph/`
  index still means zero tools.
supersedes: >-
  MCP-01 ("codegraph_explore as the only default-visible tool"). That
  requirement is marked [x] Complete in `.planning/milestones/
  v0.1-REQUIREMENTS.md`, an ARCHIVED milestone, and it accurately records what
  shipped at the time.
artifact_constraint: >-
  MCP-01's explore-only default is superseded by this change; the requirement
  text lives in an archived milestone and must be re-stated in the ACTIVE
  milestone via GSD verbs, not hand-edited here. Do not retroactively edit the
  archived requirement — rewriting it would falsify the historical record and
  violates the repo's planning-artifact rule against hand-authoring structure
  into GSD-owned generated files. Routing this re-statement is the
  orchestrator's job, not this debug session's.

## Eliminated (continued)

- hypothesis: "`codegraph install` writes a defective registration because it
  omits an `env` block setting `CODEGRAPH_MCP_TOOLS`."
  evidence: MCP-01 specifies explore-only as the *default*. An install that
  wrote an allowlist env block would violate the accepted requirement. The
  omission is correct-by-design.
  timestamp: 2026-08-08
- hypothesis: "A code defect in `registerTools`/`ParseAllowlist` under-registers
  companion tools."
  evidence: `registerTools` (server.go:362-377) registers explore
  unconditionally then each allowlisted companion — exactly MCP-01/MCP-02.
  `TestDefaultToolVisibility` and `TestAllowlist` both pass and pin this.
  timestamp: 2026-08-08

## Current Focus

reasoning_checkpoint:
  hypothesis: >-
    "One tool" is the CORRECT, specified default (MCP-01). The defect is a
    documentation/wire-contract failure: no artifact the user or their agent
    can read states the actual contract, and two state the opposite —
    README.md claims all eight tools are exposed, and the MCP `instructions`
    string attributes visibility solely to index state, never to the
    allowlist.
  confirming_evidence:
    - "v0.1-REQUIREMENTS.md:45 MCP-01 marked [x] Complete: codegraph_explore
       is the ONLY default-visible tool."
    - "README.md:91 states 'Eight tools are exposed' and names all 8, with
       zero mentions of CODEGRAPH_MCP_TOOLS anywhere in the file."
    - "server.go:56 instructions: 'An empty tool list means this repository
       has no index yet... Tools appear automatically once an index exists' —
       the only explanation offered, and it is inapplicable to a 1-tool list."
    - "rg finds CODEGRAPH_MCP_TOOLS in zero user-facing surfaces."
  falsification_test: >-
    Find any user-facing artifact (README, --help, docs/) that states the
    default is explore-only or names CODEGRAPH_MCP_TOOLS. One such hit would
    demote this from 'undocumented contract' to 'user missed the docs'.
    Result: zero hits — hypothesis survives.
  fix_rationale: >-
    Addresses the root cause, not the symptom. The symptom-level fix
    (register all 8 by default) would violate the accepted MCP-01 contract
    and break TestDefaultToolVisibility, toolslist-default.golden, and the
    wire-oracle corpus. The real failure is that the contract is true in code
    and false in every document describing it — so the fix corrects the
    documents and adds a gate that keeps them correct.
  blind_spots:
    - "Not yet tested: whether growing the instructions string stays inside
       its documented ~600-byte / single-paragraph / no-newline budget."
    - "Not yet tested: whether the 24 golden transcripts regenerate cleanly."
    - "Untested claim: that MCP-01's explore-only default is still the RIGHT
       product decision. This fix assumes it is (it is an accepted, shipped
       requirement) and only makes it honest. Revisiting it is a requirements
       change, not a debug fix."
  candidate_causes:
    - "code: registerTools under-registers — ELIMINATED, matches MCP-01/02."
    - "config: install writes no env block — ELIMINATED, correct-by-design."
    - "documentation/wire-contract: README + instructions both misdescribe the
       gate — CONFIRMED."
    - "environment: index missing/stale — ELIMINATED in prior session."
  and_gate: >-
    YES, two contributing conditions are simultaneously required. (a) README's
    'Eight tools are exposed' creates the expectation — without it the user
    would never have expected 8. (b) The instructions string's index-only
    explanation denies the agent any way to explain or remedy the 1-tool list
    at the moment of confusion — and actively misdirects, since the single
    cause it names (no index) is false here. Either alone is a latent doc bug;
    together they produce a user who concludes the server is broken.

reasoning_checkpoint_v2 (supersedes the fix_rationale above after DECISION 1):
  status: >-
    The root cause above is UNCHANGED and confirmed empirically. What changed
    is the chosen REMEDY. The user rejected "make the documents honest about
    an explore-only default" in favour of "make the default what the
    documents already promised, and re-point the env var at the opposite
    job." This converts a docs-only fix into a spec change, so the fix now
    has four legs, not one.
  new_contract:
    default: "CODEGRAPH_MCP_TOOLS unset + index present => all 8 tools."
    narrowed: "CODEGRAPH_MCP_TOOLS set => only the named companions, plus the
      always-visible codegraph_explore."
    escape_hatch: "CODEGRAPH_MCP_TOOLS set to the EMPTY string => explore
      only. This is the old default, still reachable, now by explicit
      operator action."
    unchanged: "MCP-03 — no .codegraph/ index still means zero tools."
  load_bearing_mechanism: >-
    os.LookupEnv, not os.Getenv. Under narrowing semantics "unset" and "set
    to empty" MUST answer differently (all 8 vs 1), and os.Getenv collapses
    them into the same empty string. This is the single most fragile point of
    the inversion and the reason ResolveCompanions takes an explicit
    `present bool` rather than testing `value != ""`.
  new_failure_mode_analysis: >-
    Under the OLD opt-in semantics a typo cost exactly the mistyped tool —
    blast radius 1, and the operator was adding, so the loss was visible as
    "the thing I asked for is missing." Under NARROWING semantics a typo in
    an otherwise-valid list still narrows to the valid subset (blast radius
    unchanged for that name), but a list whose names are ALL typos narrows to
    explore-only — blast radius 7, presenting as the exact symptom that
    opened this debug session. The per-name stderr warning is therefore
    NECESSARY BUT NOT SUFFICIENT: it names what was ignored but never states
    the resulting surface, and MCP stdio stderr is frequently invisible to
    the agent user. VERDICT: keep the warning non-fatal (an unknown name must
    never fail startup — MCP-02), but add ONE consequence line stating how
    many companions actually survived the filter and how to get all 8 back.
    Do not make it fatal; do not silently fall back to default-all on an
    all-unknown list (that would make a typo indistinguishable from intent).
  fix_rationale: >-
    Addresses the root cause at the layer the user chose: the gate itself.
    The instructions string and README stop being wrong by becoming true
    rather than by being narrowed, and the env var keeps a real job
    (shrinking the context surface) instead of being deleted.
  blind_spots:
    - "modern-listen-catalog-change's frozen one-notification property was
       derived from exactly one AddTool call landing in changeAndNotify's
       10ms debounce window. Under default-all that becomes 8 AddTool calls.
       Mitigated structurally by pinning that scenario to the explore-only
       filter rather than trusting the debounce to coalesce."
    - "Wire-oracle captures inherit the developer's real environment. Now
       that the DEFAULT depends on the variable's PRESENCE, a developer with
       CODEGRAPH_MCP_TOOLS exported would capture different bytes. Mitigated
       by making Capture strip the variable from the inherited environment
       unless the scenario declares it."

test: TDD — write RED contract tests before touching any prose. DONE.

tdd_checkpoint:
  test_file: "internal/mcp/instructions_contract_test.go"
  status: "red"
  tests:
    - name: "TestInstructionsDocumentsAllowlistGate"
      status: "RED (target)"
      failure: "instructions never names CODEGRAPH_MCP_TOOLS"
    - name: "TestREADMEDocumentsToolVisibilityGate"
      status: "RED (target)"
      failure: "README names 7 gated companions, never names CODEGRAPH_MCP_TOOLS"
    - name: "TestInstructionsStaysWithinWireBudget"
      status: "PASS (boundary neighbor — must STAY green through the fix)"
    - name: "TestREADMEGateCheckerIsNotVacuous"
      status: "PASS (6 sub-cases: 0/1/all companions x gate present/absent)"
  oracle_type: "derived (contract) — each artifact's claim is checked against
    companionNames, this package's own source of truth, not a re-typed literal"

fix_feasibility:
  instructions_current_bytes: 511
  instructions_drafted_bytes: 585
  budget: 600
  verdict: "fits with 15 bytes headroom, single paragraph, no newline"

blast_radius_flagged:
  golden_transcripts: "24 of 28 testdata/wireoracle/transcripts/*.golden embed
    the instructions string verbatim and must be regenerated"
  freeze_guard: "Taskfile check:transcript-freeze is ADVISORY (never fails the
    build); the mandated control is a reviewed-diff pass with every cause named
    in the commit message"
  pinned_tests_NOT_moved: "TestDefaultToolVisibility (:135) and TestAllowlist
    (:165) stay exactly as they are — this fix does not change gating behavior,
    only the documents describing it"

fix_feasibility_v2:
  instructions_new_bytes: 580
  budget: 600
  headroom: 20
  verdict: >-
    The three-mechanism string fits WITHOUT widening the budget. Measured,
    not estimated. Single paragraph, no newline.

commits:
  - "970af89 feat(mcp)!: register all eight tools by default; CODEGRAPH_MCP_TOOLS now narrows"
  - "349e805 test(wireoracle): re-derive the scenario matrix for narrowing semantics"
  - "edd3aa5 fix(mcp): annotate all eight tools read-only and closed-world"
  - "45ba5d6 docs(mcp): document the tool-visibility contract and gate it in CI"
  - "29a30b1 test(wireoracle): re-freeze the transcript corpus (25 changed, 1 new, 3 unchanged)"

commit_sequence_deviation: >-
  The suggested sequence put the internal/mcp unit-test moves in commit 2.
  They landed in commit 1 instead: removing ParseAllowlist breaks the
  internal/mcp test package's COMPILATION, and a commit that does not build
  is worse than the split. Commit 2 remains the wire-oracle matrix, which is
  a separate package and a separate logical change. Nothing else moved.

human_verification: >-
  CONFIRMED 2026-08-09 by the maintainer in the reporting client (Claude Code),
  which reconnected to the codegraph MCP server against a build of HEAD. This
  closes the checkpoint the session was held at.

  CORRECTION to the note this field previously carried: it claimed the live
  client's CODEGRAPH_MCP_TOOLS env block "must be removed to see the default".
  That was WRONG for this maintainer's actual registration, which names all
  seven companions (node,search,callers,callees,impact,files,status) — under
  narrowing semantics that yields 8 tools, verified by direct probe. Removing
  the block is still RECOMMENDED, but for a different reason: it now PINS the
  set to today's seven names, so a companion tool added later would be silently
  hidden. That is the same failure mode that got the envAllowlistAllCompanions
  wire scenario deleted rather than renamed.

maintainer_ruling: >-
  Raised during PR review and settled 2026-08-09: TS CodeGraph v1.3.x parity is
  a FUNCTIONALITY BASELINE, not a binding constraint — deliberate divergence is
  acceptable. This matters because testdata/golden/README.md records that TS
  v1.3.1 ALSO gates codegraph_node off by default (its capture harness must set
  CODEGRAPH_MCP_TOOLS=explore,node), so CODEGRAPH_MCP_TOOLS is an opt-in
  allowlist UPSTREAM TOO. This port did not invent the confusing default; it
  inherited it without inheriting any documentation of it, then contradicted it
  in README.md. Inverting to opt-out is therefore a real divergence, surfaced
  and signed off rather than absorbed as a bug fix. Note that .claude/CLAUDE.md
  still phrases parity more strictly than this ruling.

shipped_as: >-
  PR #44 (issue #43), branch fix/mcp-tool-visibility-default, 6 commits.
  Title feat(mcp)!: with bump-minor-pre-major at 0.5.1, so release-please will
  propose v0.6.0 on merge. All 12 CI checks green, mergeStateStatus CLEAN.
  Note that transcript-freeze passing is NOT independent confirmation the 25
  regenerated goldens are correct — that check is advisory and never fails the
  build; the reviewed diff was and remains the actual control.

next_action: >-
  None — session resolved. Two follow-ups live outside this file: MCP-01's
  supersession still needs re-stating in the ACTIVE milestone via GSD verbs
  (the archived v0.1 requirement was deliberately left unedited), and
  .planning/todos/pending/2026-08-08-author-a-codegraph-usage-skill-for-agents.md
  captures the codegraph-usage skill this investigation showed was missing.
