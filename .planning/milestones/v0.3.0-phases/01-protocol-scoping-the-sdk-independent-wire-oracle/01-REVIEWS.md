---
phase: 1
reviewers: [codex, pi]
reviewed_at: 2026-08-05T18:24:18Z
plans_reviewed: [01-01-PLAN.md, 01-02-PLAN.md, 01-03-PLAN.md, 01-04-PLAN.md, 01-05-PLAN.md, 01-06-PLAN.md, 01-07-PLAN.md]
---

# Cross-AI Plan Review — Phase 1

## Codex Review

# Cross-AI Plan Review

## Summary

The phase has a strong verification-first architecture, but the current plans contain several blockers that could make the oracle certify less than intended. The most serious are:

- Plan 01’s only `codegraph_explore` call is malformed, so it does not establish successful explore coverage.
- The proposed repo-owned protocol literal does not actually control the server’s negotiated wire version.
- Plan 02’s proxy records only the first client frame and does not capture the server’s initialize response, so it cannot measure both probes and negotiated versions.
- The normalization design violates the locked named-field-only rule for timestamps.
- Plans 04 and 05 reopen locked coverage decisions and permit narrowing one-way baselines.

The overall plan quality is high in intent and documentation, but implementation risk remains HIGH until these contradictions are resolved.

---

## Plan 01 — Tracer and Production Seam

### Strengths

- The seam correctly removes the direct SDK dependency from `internal/cli`. Today the leak is exactly the direct import at [`internal/cli/serve.go:13`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/serve.go:13) and the direct `server.ServeStdio` invocation at [`internal/cli/serve.go:252`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/serve.go:252). Routing that through `internal/mcp.NewStdioServer` satisfies SDK-02 without forcing the numerous in-process test callers to change.
- Keeping `BuildServer` concrete and adding a separate wrapper is justified by real callers. For example, the golden suite passes its result directly to `mcpclient.NewInProcessClient` at [`testdata/golden/golden_parity_test.go:1397`](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_parity_test.go:1397). A return-type change would create the ripple the plan seeks to avoid.
- The raw-wire design is grounded in a proven repository pattern. The existing integration test owns the actual stdout pipe at [`test/integration/mcp_stdout_purity_test.go:77`](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/mcp_stdout_purity_test.go:77), scans every line at [`test/integration/mcp_stdout_purity_test.go:153`](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/mcp_stdout_purity_test.go:153), and rejects malformed output at [`test/integration/mcp_stdout_purity_test.go:188`](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/mcp_stdout_purity_test.go:188).
- Adding the new test leg explicitly is appropriate. The repository already documents that `./...` silently misses `testdata/golden` at [`.github/workflows/ci.yml:86`](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:86), and the `test` wrapper’s expected legs are guarded as an exact set at [`internal/upgrade/taskfile_shape_test.go:74`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:74).

### Concerns

- **HIGH — The tracer does not make a successful explore call.** The plan sends `codegraph_explore` with empty arguments, but the production schema marks `query` required at [`internal/mcp/tools.go:80`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go:80), and the handler returns an error result when it is absent at [`internal/mcp/tools.go:105`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go:105). Consequently:

  - `handshake-explore` freezes an error, not an explore result.
  - Plan 04 incorrectly treats it as successful coverage of the eighth tool.
  - The transcript is unlikely to contain a repository path, contradicting Plan 01’s acceptance criterion requiring `<REPO>`.

- **HIGH — `ProtocolVersion` does not control the declared wire version.** Current construction calls `server.NewMCPServer("codegraph", version, ...)` at [`internal/mcp/server.go:94`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:94). That second value is the implementation version declared at [`internal/mcp/server.go:26`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:26), not the negotiated protocol revision. The plan adds `internal/mcp.ProtocolVersion` and asserts that the SDK happens to return the same value, but never wires the literal into mark3labs negotiation. Therefore VRFY-02’s truth—“the server’s declared protocol version reads from a repo-owned literal”—is not achieved.
- **MEDIUM — Capture lifecycle is underspecified.** The existing test kills and waits for the process in cleanup at [`test/integration/mcp_stdout_purity_test.go:95`](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/mcp_stdout_purity_test.go:95). The proposed standalone `Capture` has no `testing.T`, despite referring to `t.Cleanup`, and needs an explicit close-stdin/terminate/wait protocol. Otherwise subprocesses or stderr-copy goroutines can survive past capture.
- **MEDIUM — Tool count is duplicated state.** Current registrations are performed at [`internal/mcp/server.go:119`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:119). Independently calculating `toolCount` risks drifting from registration conditions. It should be incremented at the same registration sites or derived from the constructed server if supported.
- **LOW — Session-line concurrency needs a real write guarantee.** One `fmt.Fprint` call does not formally guarantee two concurrent hook callbacks will never interleave on an arbitrary `io.Writer`. If repeated initialize is supported, the writer or formatter should be mutex-protected.

### Suggestions

- Give `handshake-explore` a real fixture query and assert `IsError == false`.
- Separate “expected repo path normalization” from the first tracer if its legitimate payload contains no path; do not force `<REPO>` into an unrelated transcript.
- Either find a mark3labs option that accepts the repository literal or revise Phase 1’s VRFY-02 deliverable honestly: the literal is an asserted compatibility pin until Phase 2 provides a backend that can consume it.
- Make `Capture` use `context.WithTimeout`, close stdin after scripted requests, terminate on failure, and always call `Wait`.
- Count tools at the registration seam.

---

## Plan 02 — Agent Audit and Scoping Documents

### Strengths

- The MEASURED/UNMEASURED distinction is well designed and honors the locked decisions.
- Capturing raw client identity and capabilities without an SDK decoder is consistent with the phase’s core verification principle.
- Appending one process-start event per invocation is a useful way to detect probe spawns.
- The scoping and Team Scale document locations match the locked context.

### Concerns

- **HIGH — The shim cannot measure the negotiated version as designed.** Its observation model only parses client stdin. The negotiated version exists in the server’s initialize response on child stdout, but `Run` merely copies child stdout to the agent. Therefore the audit’s required “offered vs negotiated” columns cannot be populated from the described instrument.
- **HIGH — Capturing only the first frame loses initialize after a probe.** If the first frame is `server/discover` or another probe, the plan records that frame and never inspects the subsequent `initialize`. It cannot simultaneously record `PreInitializeProbe=true` and the initialize fields required by D-11.
- **MEDIUM — The proxy is not byte-exact.** A `bufio.Scanner` removes line terminators and the plan adds `'\n'` back. That changes CRLF input and a final frame without a newline. This contradicts the claimed byte-for-byte passthrough.
- **MEDIUM — Configuration restoration occurs too late in the failure model.** The task edits real agent configurations and restores them after launching each client. Restoration needs to be registered immediately after a verified backup, before any launch or parsing step, so every intermediate failure path restores bytes.
- **MEDIUM — The audit expects three successful measured clients but does not define noninteractive commands, timeouts, authentication handling, or proof that a tool list completed.** “A single non-interactive prompt is enough” is an assumption, not an executable procedure.
- **LOW — Raw parse errors are written to the audit log despite the stated minimization goal.** The threat model says only the first frame is logged, but malformed frames could contain unrelated data. Store a bounded/hashed representation unless raw evidence is essential.

### Suggestions

- Tee and inspect frames in both directions until the initialize request and matching response have been observed.
- Track:

  - first client method,
  - whether it preceded initialize,
  - initialize request fields,
  - initialize response protocol version,
  - process-start count.

- Use `bufio.Reader.ReadBytes('\n')` or a byte-preserving tee, not `Scanner`, for the forwarding path.
- Encapsulate each config change in a restore-on-exit helper and verify restoration in a deferred/finally path.
- Add per-client command recipes and bounded timeouts to the plan.

---

## Plan 03 — Structural Guards and Session-Line Contract

### Strengths

- Explicitly loading `github.com/seanb4t/codegraph-go/testdata/golden` is necessary and sufficient to address the `./...` blind spot, assuming the package loads successfully. The relevant SDK constant is indeed at [`testdata/golden/golden_parity_test.go:1477`](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_parity_test.go:1477).
- The six known constant sites are real, including [`test/integration/mcp_stdout_purity_test.go:120`](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/mcp_stdout_purity_test.go:120) and the golden site above.
- The direct-import SDK guard matches SDK-02’s actual boundary. It correctly avoids forbidding the legitimate transitive path through `internal/mcp`.
- The hostile session-line cases are thorough and address a real logging boundary.

### Concerns

- **HIGH — The protocol-version predicate does not match its own stated scope.** The proposed predicate accepts every external object whose name resembles `protocolVersion`, except struct fields. That includes functions, types, methods, and potentially legitimate variables, even though the goal is to forbid SDK-owned constants. It can therefore produce false positives.
- **MEDIUM — It still cannot truly forbid the semantic class.** An SDK constant named `LatestVersion`, `CurrentRevision`, or similar would evade `(?i)protocol.?version`. The plan claims zero-maintenance survival across arbitrary future SDKs, which the name heuristic cannot guarantee.
- **MEDIUM — Module ownership checking with a raw prefix should be boundary-aware.** `strings.HasPrefix(path, "github.com/seanb4t/codegraph-go/")` is generally safe for internal packages but should also explicitly admit the module root and reject lookalike module prefixes.
- **MEDIUM — The planted SDK-import self-test may fail during type checking for unrelated reasons.** Injecting an unused import or an import with no reference causes a compile error before the direct-import predicate can be evaluated. The overlay must insert a syntactically used reference.
- **LOW — “Measured wall-clock cost in a doc comment” is brittle documentation.** CI and developer hardware differ. Record the command and approximate reference measurement in the summary, not a permanent source comment that will rapidly become stale.

### Suggestions

- Restrict the predicate to `*types.Const`, plus package-level `*types.Var` only if genuinely required.
- Document the heuristic limitation rather than claiming it catches every possible spelling.
- Add tests for:

  - external const `LATEST_PROTOCOL_VERSION` → violation,
  - external function `ProtocolVersion()` → allowed,
  - external struct field `ProtocolVersion` → allowed,
  - repository-owned constant → allowed,
  - alternate SDK constant naming → explicit decision.

---

## Plan 04 — Full Tool and Error Coverage

### Strengths

- Exact-set and exact-zero assertions correctly preserve the current discipline embodied by the registration logic at [`internal/mcp/server.go:119`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:119).
- The listed tool inventory matches the actual surface: explore plus the seven companions declared at [`internal/mcp/server.go:32`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:32).
- The confinement scenario targets a real boundary at [`internal/mcp/tools.go:31`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go:31).
- Anchors are run against fresh captures rather than goldens, avoiding circular validation.

### Concerns

- **HIGH — This plan counts the malformed Plan 01 explore call as coverage.** Without correcting Plan 01, there is no successful `codegraph_explore` result in the supposedly complete eight-tool baseline.
- **HIGH — The checkpoint re-litigates locked D-05.** The user explicitly said locked decisions must not be reopened. Offering a “narrower” option can permanently weaken the baseline.
- **MEDIUM — “All four error/edge shapes” is inconsistent with the actual five listed cases.** The plan contains unknown method, unknown tool, malformed args, confinement rejection, and call-before-initialize. Acceptance text should state four errors plus one edge.
- **MEDIUM — The outside-path scenario can leak an unnormalized host path.** `confineToRepoRoot` includes the rejected path in its error at [`internal/mcp/tools.go:42`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go:42). If the scenario uses a temp-directory sibling, the transcript will contain a host-specific path that the repo-root substitution does not remove.
- **LOW — Running every scenario independently is safe but expensive.** By the end of Plan 05 the suite has roughly 23 index/capture subprocesses, not the “~17” repeatedly cited. The CI cost should be measured.

### Suggestions

- Remove the narrowing checkpoint; retain only a non-blocking opportunity to name extra cases.
- Add a dedicated successful explore scenario or repair `handshake-explore`.
- Use a stable confinement path whose wire text is portable, or add a narrowly anchored `rejectedPath` substitution explicitly approved by D-04.
- Correct all counts and measure suite wall time before making it a required PR leg.

---

## Plan 05 — Multi-Era Baseline

### Strengths

- The silent-coercion interpretation is correct for the planned pre-migration baseline: unsupported input should be frozen as observed success, not rewritten into the Phase 3 behavior desired later.
- Capturing both initialize and tools/list demonstrates that the negotiated session remains usable.
- The four supported revisions, unsupported revision, and omitted-version path form a useful compatibility matrix.
- Avoiding a fabricated error-code anchor for silent coercion is correct.

### Concerns

- **HIGH — This plan again reopens a locked decision.** D-06 already locks the four supported revisions plus an unsupported revision. Offering “five-era” versus “six-era” is acceptable only for the discretionary omitted-version addition; it must not permit dropping the locked five.
- **MEDIUM — The plan depends only on 01-04, not explicitly on the version guard in 01-03.** It later relies on Plan 03’s archtest to prove no SDK constant entered the scenarios, but the graph permits Plan 05 to run while Plan 03 remains incomplete.
- **LOW — The scenario prefix/count test depends on checkpoint outcome.** A runtime-chosen count complicates Plan 07’s exact compile-time scenario count and review reproducibility.

### Suggestions

- Make the locked five mandatory; ask only whether to add the omitted-version sixth case.
- Add `01-03` to `depends_on` if Plan 05’s acceptance requires the archtest.
- Fix the era matrix in plan text rather than varying it during execution.

---

## Plan 06 — Anti-Regeneration Guard

### Strengths

- The cross-change condition is correctly narrow: transcript plus causal server/dependency change, not a blanket ban on goldens.
- Merge-base comparison is appropriate for squash-merge workflows.
- The current CI already uses `fetch-depth: 0` where merge-base history is needed at [`.github/workflows/ci.yml:177`](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:177).
- Registering the new job in `inScopeJobs` follows the existing mechanism at [`internal/upgrade/taskfile_shape_test.go:106`](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:106).

### Concerns

- **HIGH — The rule is incompatible with Phase 3 as described unless the workflow genuinely uses separate PRs.** Phase 3 will both change `internal/mcp` and eventually update transcripts. “Regenerate in a separate PR” means that second PR contains transcript changes only, whose merge-base must already include the protocol code change. The milestone executor must enforce that repository/PR sequencing; plan prose alone does not.
- **MEDIUM — Empty changed-file lists are normal.** `ParseChangedList` treating an empty list as an error means a manual run against identical HEAD/base fails rather than reports clean, contradicting the acceptance criterion that a diff touching nothing exits zero.
- **MEDIUM — Renames are not handled explicitly.** `git diff --name-only` reports only one side depending on diff filtering. A renamed transcript directory or renamed `.golden` should remain detectable.
- **MEDIUM — Plan 01 already changes a transcript and `internal/mcp` together.** The guard lands later, so the foundational transcript is created through exactly the cross-change shape later declared unsafe. This may be acceptable for the tracer, but it must be recorded as the bootstrap exception and protected by Plan 07’s mutation proof.
- **LOW — Scratch branch mutation is operationally risky and unnecessary.** A temporary worktree or synthetic diff input can prove the wired task without switching the user’s current branch.

### Suggestions

- Define the exact two-PR Phase 3 workflow and make it a Phase 3 dependency/checkpoint.
- Treat a valid empty diff as clean; distinguish “input file missing/unreadable” from “input contains zero paths.”
- Use `git diff --name-status` and classify both sides of renames.
- Use a temporary worktree for the wired RED demonstration.

---

## Plan 07 — Non-Vacuity and Mutation Proof

### Strengths

- The plan targets the correct structural failure modes: shrinking scenarios, orphaned/missing transcripts, empty files, and over-broad normalization.
- Two-way scenario/transcript set equality is materially stronger than checking only that every scenario has a file.
- Confirming each mutation applied before trusting RED directly honors the repository’s standing rule.
- The four mutations cover distinct classes: framing, catalog loss, error mapping, and version-anchor independence.

### Concerns

- **HIGH — The normalization tests do not enforce D-04 for timestamps.** Plan 01 says every RFC3339 timestamp is replaced wherever it appears. That is not named-field substitution. Plan 07’s negative case uses an invalid date-like string; it does not test a valid RFC3339 value appearing in the wrong field. An over-broad timestamp regex would pass.
- **MEDIUM — The repo-path rule is similarly positionality-blind.** Replacing every occurrence of the path is value substitution, not necessarily named-field substitution. The negative case tests a different path, not the same path in an unrelated field.
- **MEDIUM — Mutation 3 is not guaranteed to exercise the intended handler.** The plan says to change “one handler” without binding it to a scenario that reliably reaches the altered error path. It must name the handler and request that triggers it.
- **MEDIUM — Mutation 1 may be caught by the pre-existing stdout archtest before the wire oracle runs.** That is useful corroboration but does not prove the oracle itself detected the runtime corruption unless `task test:wireoracle` is run independently and its failure recorded.
- **LOW — `git status --porcelain` being empty conflicts with creating `MUTATION-PROOF.md` unless the executor commits it before verification.** The plan should state when commits occur or limit the cleanliness check to absence of mutation residue.

### Suggestions

- Make every normalization rule field-anchored.
- Add negative cases where the exact same unstable value appears in the wrong JSON field.
- Have `Rules` carry the positive and negative matcher contract or provide rule-specific test factories, so adding a fourth rule cannot bypass testing.
- Name the exact handler and malformed request for mutation 3.
- Record separate RED results for the wireoracle task and any independent archtest.

---

## Cross-Plan Sequencing Assessment

The wave graph ensures Plan 05 follows Plan 04 and Plan 07 follows Plan 05, but it does not by itself make migration impossible before both one-way captures finish. That safety currently depends on the broader roadmap rule that Phase 2 cannot start until Phase 1 completes.

Two changes would make the constraint structural:

- Add a Phase 1 completion artifact containing the expected full scenario set and a test asserting both all-tool and multi-era subsets exist.
- Make Phase 2’s first plan depend explicitly on that artifact/test, not merely on “Phase 1 complete.”

Also, Plan 05 should depend on Plan 03 if its acceptance invokes the Plan 03 archtest.

## Overall Risk Assessment

**HIGH**

The architecture is thoughtful, source-aware, and unusually serious about non-vacuity. However, three issues directly threaten the phase goal:

1. The only explore tracer is an error response, leaving claimed all-tool coverage incomplete.
2. The repo-owned protocol literal does not control production negotiation.
3. The agent audit shim cannot observe the negotiated response and loses initialize data after a pre-initialize probe.

These are design defects rather than implementation details. Resolve them, tighten normalization to genuinely field-scoped substitution, and remove checkpoints that reopen locked one-way decisions. After those changes, the remaining risk would fall to MEDIUM, driven mainly by CI/runtime cost and the operational complexity of real-client measurement.

---

## pi Review

# Cross-AI Plan Review — v0.3.0 Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle

## Summary

This is an unusually well-evidenced plan set. I verified its load-bearing claims against the actual tree and module cache, and nearly all of them hold: the six `LATEST_PROTOCOL_VERSION` sites are exactly where the plan says (including `testdata/golden/golden_parity_test.go:1477`, which `./...` genuinely would skip), the `AddAfterInitialize` hook exists in `mark3labs/mcp-go@v0.56.0` with the exact signature quoted (`server/hooks.go:75,346`), the silent-coercion claim is real (`server/server.go:1196` — `protocolVersion()` returns `mcp.LATEST_PROTOCOL_VERSION` for unrecognized versions, never an error), and the `internal/cli/serve.go:13` import + `:252-253` `BuildServer → server.ServeStdio` pair is the only production SDK leak. The sequencing logic (VRFY-04 captures before Phase 2) is structurally sound, the non-vacuity approach attacks the harness rather than the payload as required, and the one genuine ripple risk (26 `BuildServer` call sites — the plan says "~20") is correctly designed out via the variadic-`Option` trick. Concerns are mostly edge-case and residual-risk items, not structural flaws.

## Strengths

- **Every falsifiable claim I checked is true.** Six `LATEST_PROTOCOL_VERSION` sites confirmed verbatim; `server.go:26` `const version = "0.1.0"` and `companionNames` at `:32` match; `internal/cli/serve_test.go` references `BuildServer` only in comments, so the seam change really does close the only production leak. The plan's claim that `AddAfterInitialize` hands requested + negotiated version + `clientInfo` in one callback is verified against the vendored source.
- **The `testdata/golden` blind spot is handled correctly, not just claimed.** `go list ./...` skips any directory named `testdata`; plan 01-03 passes `github.com/seanb4t/codegraph-go/testdata/golden` as an explicit second pattern AND asserts its presence in the loaded set (a load that silently resolved zero testdata packages would be caught). This is the GOLDEN-01 failure class closed by construction.
- **Sequencing/one-way doors are real.** All capture plans (01-01, 01-04, 01-05) sit in waves 1–3 *within* Phase 1, and Phase 2 (SDK removal) is a separate phase gated on Phase 1 verification — the unrecapturable baselines cannot be reordered past the migration. The two blocking `checkpoint:decision` gates in 01-04/01-05 make the one-way doors deliberate rather than accidental.
- **Silent coercion is handled honestly.** Plan 01-05 freezes the unsupported-version scenario as the *success* it is (verified: no server-side rejection path exists in v0.56.0), adds no error-code anchor, and documents it in the scoping doc. A weaker plan would have frozen an assumed `-32602`.
- **The variadic-`Option` design genuinely avoids the ripple.** `BuildServer(..., opts ...Option) *server.MCPServer` keeps all 26 verified call sites (including `testdata/golden`'s in-process-client tests that need the concrete type) compiling unchanged, while `NewStdioServer(...) mcp.Server` closes SDK-02 at the one production call site. This is the right trade and matches RESEARCH Pitfall 4's option (a).
- **Non-vacuity attacks the harness.** Empty-transcript-never-matches, exact scenario count (not lower bound), two-way `os.ReadDir` set equality, per-rule positive+negative normalization cases, and mutation-4's designed asymmetry (anchor red / transcripts green) are all structural guards, exactly per D-07.
- **The VRFY-02 predicate forbids the class.** Keying on "external package + name matches `(?i)protocol.?version` + not a struct field" survives the Phase 2 swap with zero maintenance; the struct-field exclusion keeps `req.Params.ProtocolVersion = ...` (used at all four `test/integration` sites today) legitimate.

## Concerns

- **MEDIUM — VRFY-02 archtest false-positive surface.** The `(?i)protocol.?version` name pattern will flag *any* external package-level identifier so named, not just MCP SDKs. Nothing in the current dependency closure exports such a symbol (verified the pattern targets `mcp.LATEST_PROTOCOL_VERSION`/`ValidProtocolVersions`), but a future unrelated dependency (e.g., a QUIC/TLS/Telnet library exporting `ProtocolVersion`) would trip the guard with a confusing message. This is acceptable as-is but the failure message should name the rule and the escape hatch (use a local alias), which the plan does not specify.
- **MEDIUM — D-03 trigger set is narrower than the actual diff-causing surface.** Plan 01-06 fires only on transcript + (`internal/mcp/*.go` OR MCP-dep bump). But transcript bytes also depend on `internal/query` (explore ranking output), `internal/indexer`, and the tree-sitter grammars — a query-engine change could legitimately alter `handshake-explore.golden`, and regenerating it in the same PR would pass the guard. CONTEXT D-03 locks the narrow set deliberately, so this is a *recorded* residual risk rather than a plan defect — but the plan should state in the CI failure message and the scoping doc that the narrow set is a floor, not a proof of innocence.
- **MEDIUM — Session line is "always-on" only by convention at one call site.** The hook is registered only when `cfg.sessionLog != nil` (plan 01-01, Layer 3). Today `serve.go` always passes `cmd.ErrOrStderr()`, so VRFY-03 holds — but nothing stops a future caller (or a test helper becoming production) from passing nil and silently disabling the milestone's "only available mitigation for a spec-sanctioned silent version mismatch." Suggest either making the writer mandatory in `NewStdioServer`'s signature with an explicit `io.Discard` for opt-out, or an assertion somewhere that the production path never passes nil.
- **MEDIUM — Normalizer rule 2's anchor is brittle in the fail-safe direction but adds a silent-drift risk.** The regexp anchored on `"serverInfo":{"name":"codegraph","version":"` assumes exact key order and compact serialization. If the SDK changes spacing/order, the rule stops matching and the *real version* leaks into frozen bytes — which fails the diff loudly (good) but with a misleading "transcript changed" signal rather than "normalization rule broke." Plan 01-07's per-rule positive/negative tests use hand-constructed inputs, which won't catch this. Consider a self-test that asserts each rule matched ≥1 time during a real capture (a "rule did fire" counter), so a silently dead rule is distinguishable from a byte diff.
- **LOW — Timestamp normalization rule may be vacuous today.** Nothing in the current wire output obviously emits RFC3339 timestamps; the rule exists for future-proofing. Plan 01-07's identity-when-nothing-matches test covers the harness, but a rule that never fires in any real transcript is dead weight that a future reader may widen carelessly. Worth a comment stating whether any current transcript exercises it.
- **LOW — `error-confinement-reject` scenario needs verified argument shape.** The plan instructs the executor to read `internal/mcp/tools.go` for parameter names; the confinement boundary exists (`TestConfinementAnchoredOnRepoRootNotStartPath` referenced in `server.go` comments), but whether `codegraph_node`'s `path` argument actually reaches `confineToRepoRoot` for an *outside-root* path should be confirmed during capture, not assumed — the plan's own "read every frozen transcript before trusting it" step covers this, which is good.
- **LOW — CI cost is real and unmeasured.** ~24 scenarios × (init + serve subprocess) per `task test:wireoracle` run, on every PR, plus a whole-module `packages.Load` with `Tests: true` in two new archtests. Plan 01-03 requires recording the measured load cost (good); the oracle suite itself has no such measurement requirement. Pitfall 5's advice applies to the oracle too.
- **LOW — Plan 01-02's config-restore verification is procedural, not automated.** "verified by comparing against the backup before the backup is deleted" relies on the executor; a hashed before/after comparison printed into the audit doc would make T-02-02's mitigation auditable after the fact.

## Suggestions

1. Add a "normalization rule fired ≥1× per capture" counter assertion to `oracle_test.go`, so a silently-dead rule is distinguishable from a real byte diff (addresses the brittle-anchor concern).
2. Make the session-log writer effectively mandatory: `NewStdioServer` should require a non-nil writer (panic or error on nil), with `io.Discard` as the explicit opt-out. One-line change that hardens VRFY-03's "always on" from convention to construction.
3. In the VRFY-02 archtest's violation message, include remediation text ("vendor the value into `internal/mcp.ProtocolVersion` or open a reviewed exception") — this guard *will* false-positive someday and the message should be self-explanatory.
4. Record a wall-clock measurement of `task test:wireoracle` in the phase summary, same as the archtest measurement plan 01-03 already requires.
5. In `tools/mcpaudit`'s audit task, emit SHA-256 of each agent config before/after restore into the observation log, so config restoration is verifiable from the artifact rather than from the executor's say-so.
6. In plan 01-06's `Reason` string and in `docs/MCP-2026-07-28-SCOPING.md`, state explicitly that the trigger set is deliberately a floor: changes to `internal/query`/`internal/indexer` can also legitimately move transcript bytes and still require split-PR discipline by convention.
7. Correct the "~20 existing call sites" phrasing in plan 01-01 to the verified 26 — immaterial to the design, but the plan's evidence standard deserves the accurate number.

## Risk Assessment

**LOW–MEDIUM.** The plan set achieves the phase goals with strong structural properties: the unrecapturable captures are sequenced before Phase 2 by the phase boundary plus blocking checkpoints; the oracle never uses the SDK as its own oracle (verified the purity-test precedent it generalizes); every guard carries a confirmed-applied-mutation requirement with a permanent regression encoding; and the seam design closes SDK-02 without touching 26 call sites. Residual risk concentrates in (a) the deliberately narrow D-03 trigger set, which is CONTEXT-locked and disclosed, and (b) small hardening gaps (nil-writer opt-out, dead-normalization-rule detectability) that are cheap to fix now and expensive to discover in Phase 3. No HIGH-severity defects found; nothing here blocks execution.

---

## Consensus Summary

Both reviewers had full repo access; neither carries the `REVIEWED-WITHOUT-REPO-ACCESS` marker, so
both verdicts count at full weight. Neither is a diff-only lane. Codex cited 44 `file:line`
anchors, pi cited 5 but verified against the vendored `mark3labs/mcp-go@v0.56.0` module cache.

**The two verdicts diverge sharply: Codex returned HIGH overall risk with 5 blocking findings; pi
returned LOW–MEDIUM with "no HIGH-severity defects, nothing blocks execution."** That disagreement
is the most useful output of this review, so the orchestrator independently adjudicated every
disputed HIGH claim against source rather than recording both positions. Adjudications are in
"Divergent Views" below and are marked CONFIRMED / REFUTED / PARTIAL with the evidence.

### Agreed Strengths

Both reviewers independently verified and endorsed:

- **The `testdata/` blind spot is genuinely closed, not merely claimed.** `go`'s `./...` skips any
  directory named `testdata`, and `testdata/golden/golden_parity_test.go:1477` holds one of the six
  `LATEST_PROTOCOL_VERSION` sites. Plan 01-03 passes `github.com/seanb4t/codegraph-go/testdata/golden`
  as an explicit second pattern *and* asserts its presence in the loaded set, so a load that silently
  resolved zero testdata packages is caught.
- **The variadic-`Option` seam is the right trade.** Keeping `BuildServer` concrete and adding
  `NewStdioServer(...) mcp.Server` closes the only production leak
  (`internal/cli/serve.go:13` import + `:252` `server.ServeStdio`) without forcing the in-process
  test callers — including `testdata/golden/golden_parity_test.go:1397`, which needs the concrete
  type — to change.
- **Silent coercion is handled honestly.** `protocolVersion()` really does coerce unrecognized
  versions to `LATEST_PROTOCOL_VERSION` with no error path, and plan 01-05 correctly freezes that
  scenario as the success it is, adding no fabricated error-code anchor.
- **Non-vacuity attacks the harness, not the payload**, per D-07: empty-transcript-never-matches,
  two-way scenario/transcript set equality, per-rule positive+negative normalization cases, and
  mutation-4's designed anchor-red/transcripts-green asymmetry.
- **The raw-wire design generalizes a proven in-repo pattern** rather than inventing one
  (`test/integration/mcp_stdout_purity_test.go:77,153,188`).

### Agreed Concerns

Raised independently by both reviewers — highest priority:

- **The VRFY-02 archtest predicate is over-broad.** Keying on "external package + name matches
  `(?i)protocol.?version` + not a struct field" admits functions, types, methods and package-level
  vars, not just constants. Codex rated this HIGH (false positives), pi MEDIUM (acceptable but the
  failure message needs remediation text and an escape hatch). Both are pointing at the same defect.
  Codex adds the converse: the name heuristic cannot catch an SDK constant spelled `LatestVersion`
  or `CurrentRevision`, so the plan's "zero-maintenance survival across arbitrary future SDKs" claim
  is overstated and should be documented as a heuristic, not a guarantee.
- **CI cost of the oracle suite is unmeasured.** Plan 01-03 requires recording the measured
  `packages.Load` cost; the oracle's own ~17–24 subprocess spawns per run on every PR carry no
  equivalent measurement requirement. Both flagged this; both rated it LOW.
- **Plan 01-02's config-restore is procedural, not auditable.** Both want the backup/restore proven
  from the artifact (pi: SHA-256 before/after into the observation log) rather than from the
  executor's say-so, and Codex adds that restoration must be registered immediately after a verified
  backup so every intermediate failure path restores bytes.

### Divergent Views

Each disputed claim below was checked against source by the orchestrator.

**1. "The tracer never makes a successful explore call" (Codex HIGH) — CONFIRMED. This is the most
consequential finding in the review, and pi missed it entirely.**

Verified: `exploreTool()` declares `mcp.WithString("query", mcp.Required(), ...)`
(`internal/mcp/tools.go:82`) and `exploreHandler` opens with
`req.RequireString("query")`, returning `mcp.NewToolResultError(err.Error()), nil` when it is
absent (`internal/mcp/tools.go:103-106`). Plan 01-01:332 scripts `tools/call` (id 3) naming
`codegraph_explore` **with empty `arguments`**. The frozen `handshake-explore.golden` will
therefore record an error result, not an explore result.

Two consequences follow, and the second is the serious one:
- Plan 01-01:385's acceptance criterion requires the golden to contain the literal `<REPO>`. An
  error result carries no repo path, so the `<REPO>` substitution never fires and that criterion is
  likely unsatisfiable as written.
- **Plan 01-04:176 explicitly delegates the eighth tool to it** — "One `tools/call` per remaining
  tool (`handshake-explore` already covers `codegraph_explore`)". So the "all 8 tools" bar in
  plan 01-04 is not met for `codegraph_explore`, and that is a **one-way capture**: once
  `mark3labs/mcp-go` leaves `go.mod` in Phase 2, the missing successful-explore baseline cannot be
  reconstructed. This must be fixed before 01-04's capture runs, not after.

**2. "`ProtocolVersion` does not control the declared wire version" (Codex HIGH) — CONFIRMED as a
mechanism gap, but it exposes a REQUIREMENTS-vs-ROADMAP wording conflict rather than a plan defect.
pi did not raise it.**

Verified: `BuildServer` calls `server.NewMCPServer("codegraph", version, ...)`
(`internal/mcp/server.go:94`) where `version = "0.1.0"` is documented as "this server's reported MCP
**implementation** version" (`internal/mcp/server.go:26`) — not the protocol revision. The negotiated
revision comes solely from `protocolVersion()`
(`mark3labs/mcp-go@v0.56.0/server/server.go:1196`), a private method returning only
`mcp.ValidProtocolVersions` members or `mcp.LATEST_PROTOCOL_VERSION`. **There is no
`WithProtocolVersion` option in v0.56.0** — the orchestrator enumerated every `func With…` in
`server/server.go` to confirm. A caller cannot inject a repo-owned literal while mark3labs is the
backend.

The wording conflict this exposes:
- `REQUIREMENTS.md:35` (VRFY-02) says the version is **"asserted against"** a repo-owned literal.
  The plan complies: `internal/mcp.ProtocolVersion = "2025-11-25"` currently equals
  `LATEST_PROTOCOL_VERSION`, so a dependency bump that moved the SDK value would turn the assertion
  RED — which is exactly the "must never move wire behavior silently" alarm the requirement asks for.
- `ROADMAP.md` success criterion 3 says the version **"reads from"** a repo-owned literal. The plan
  does *not* satisfy that stricter reading, and cannot in Phase 1.

Codex's recommended remedy is the correct one: keep the mechanism, and restate the Phase 1
deliverable honestly — the literal is an asserted compatibility pin until Phase 2 supplies a backend
that can consume it. The ROADMAP criterion should be reconciled with the requirement text, or the
"reads from" half explicitly deferred to Phase 2.

**3. "Plans 04 and 05 reopen locked decisions" (Codex HIGH) — PARTIAL: correct for 01-04, REFUTED
for 01-05.**

- **01-04 — CONFIRMED.** Its checkpoint offers an option `narrower` ("Capture a narrower set"),
  whose own `<cons>` reads "Permanently unrecoverable coverage gap; contradicts CONTEXT D-05, which
  already weighed and rejected this." Presenting a selectable option that the plan itself documents
  as contradicting a locked decision, on a one-way door, is a real hazard even with the warning
  label. Recommend deleting `narrower` and keeping only `full-bar` / `full-bar-plus`.
- **01-05 — REFUTED.** Codex asserts the checkpoint "must not permit dropping the locked five." It
  does not. All three options (`six-era`, `five-era`, `more`) hold the D-06 five as a floor —
  `five-era` is labelled "Exactly what CONTEXT D-06 scoped." The only variance is the discretionary
  sixth (omitted-`protocolVersion`) case, which Codex itself said was legitimate to vary. No
  locked decision is reopened here.

**4. Overall risk rating — the divergence resolves toward Codex, but for one reason, not five.**

pi's LOW–MEDIUM is not defensible given finding 1: a one-way capture that silently omits successful
coverage of the flagship tool is precisely the "seals green while measuring less than intended"
failure this phase exists to prevent. But Codex's HIGH rests on five findings, of which one is
refuted (01-05) and one is a wording conflict rather than a plan defect (VRFY-02). **The honest
reading is MEDIUM–HIGH, concentrated almost entirely in plan 01-01's explore call and its
downstream effect on plan 01-04's one-way capture.**

pi's compensating value: it caught three hardening gaps Codex did not — the session-log writer being
"always-on" only by convention (nothing stops a future caller passing nil, which would silently
disable VRFY-03's sole mitigation), the normalizer's brittle `serverInfo` anchor failing *silently*
into a misleading "transcript changed" signal, and the D-03 trigger set being narrower than the
actual diff-causing surface (`internal/query` / `internal/indexer` can legitimately move transcript
bytes). pi also counted 26 `BuildServer` call sites against the plan's "~20".

### Orchestrator finding not raised by either reviewer

**Plan 01-04:214 specifies `Scenarios()` returns "at least 17 scenarios."** A lower-bound assertion
is the exact antipattern CONTEXT D-07 and PITFALLS Trap D forbid, and plan 01-07 separately requires
an *exact* scenario-count assertion as its permanent non-vacuity guard. These two cannot both hold.
Relatedly, the scenario count is inconsistent across the plan set — 01-04 says "~17", Codex counted
~23, pi counted ~24. The same plan is also internally inconsistent on error shapes: its objective
and checkpoint say "four error/edge shapes" and it lists four golden files, while its acceptance
criteria at :214 say "the five error/edge shapes named above."
