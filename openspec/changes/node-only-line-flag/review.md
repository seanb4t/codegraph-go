---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-15T10:13:01+00:00
head: 2b84b870785acbfe4e82957b1893f3e3f808fe06
lenses:
  spec-lint: ok
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: not-applicable
  invariants: ok
---
# Review

## Findings

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| app-1 | approach | should-fix | in-change | The proposal rejects --line on callers, callees and impact because --file covers most of its use, but --file keeps every definition in the matching file, so it cannot select one of two overloads declared in one file, which is the purpose the requirement gives --line. The flag that covers that use is --signature. | proposal.md > Out of scope — "A `--line` flag on `callers`, `callees` or `impact`. Issue #323 lists it as the other option, and `--file` covers most o" | Restate the rejection reason in proposal.md: --signature selects an overload on callers, callees and impact, and --file narrows to a file; name the case --line alone would add (same-name definitions with the same parameter types in one file) and why it is left out. |
| fit-1 | fit | should-fix | in-change | The new scenario invents a Log.java fixture at lines 2-4 and 6-8, while the requirement's existing scenarios and its CLI test use the Logger.java overload fixture (log(String msg) at line 4, log(String msg, Throwable t) at line 6). Task 1.2 points at a narrowNodeMatches unit test over Go nodes named Dup instead of extending the requirement's CLI test, which already has a node subtest on that fixture. The proposal names neither the existing fixture nor a reason to leave it. | specs/graph-queries/spec.md > Scenario: A line selects one overload in node — "`Log.java` declares `log(String m)` on lines 2 to 4 and `log(String m, Throwable e)` on lines 6 to 8" | Write the scenario on the Logger.java shape of overloadCLIFixture (for example `codegraph node log --line 6` shows `Logger.java:6` with `(String msg, Throwable t)`). Then add a `--line` case to the node subtest of TestSignatureFlagSelectsOneOverload. If you keep the unit test, say in the proposal why the existing CLI fixture does not fit. |
| fit-2 | fit | should-fix | in-change | The delta puts the node --line rule inside the signature requirement as a way to select one overload. But narrowNodeMatches narrows any several definitions by line, overloads or not. The closest existing pattern for a selector other than --signature is a requirement of its own: "A file selects the definitions in it", which already refers to `node --file`. The proposal names neither that pattern nor a reason to depart from it. | specs/graph-queries/spec.md > Requirement: A signature selects one overload — "`node` SHALL also accept `--line <n>` for the same purpose." | State the --line rule of node as a selection over the definitions of a name, next to the --file selection, and keep the signature requirement for --signature only. Otherwise, say in the proposal why the line rule belongs under overload selection. |
| inv-1 | invariants | should-fix | in-change | The new rule that n is contained in a definition's lines is inclusive at both ends in code, but no test uses n equal to StartLine, n equal to EndLine, or one past EndLine. The subtest named by task 1.2 uses 35 in [30,40], and the new scenario uses 7 in lines 6 to 8. Both are mid-range. | specs/graph-queries/spec.md > Requirement: A signature selects one overload — "It SHALL keep the definition whose lines contain `n`, or else the one whose first line is nearest to `n`." | In task 1.2, add subtests to TestNarrowNeverEmpty for line 30 and line 40 (both select b) and for line 41 (no containment, so nearest), or change the scenario to `--line 6` or `--line 8`. |
| inv-2 | invariants | should-fix | in-change | The rule 'the one whose first line is nearest' implies one nearest definition. When two first lines are the same distance from n, the strict < keeps the definition that comes first in definitionsOf order. The spec does not state this tie-break and no test covers it. A probe with defs at 2-3 and 6-8 and line 4 kept the 6-8 def because it came first in the slice. | specs/graph-queries/spec.md > Requirement: A signature selects one overload — "or else the one whose first line is nearest to `n`" | State the tie-break in the requirement (for example, the first in definition order: non-generated files first, then lowest Id) and add an equidistant subtest to TestNarrowNeverEmpty. |
| inv-3 | invariants | should-fix | in-change | The requirement says `--line` keeps one definition 'for the same purpose' of selecting one overload. The code keeps every definition that contains n. `node` matches all same-named nodes in the repository, so two files whose `log` both span line n give two results. The probe returned [x y] for defs at 1-20 and 5-9 with line 7. No task adds the enforcement or rewords the requirement. | specs/graph-queries/spec.md > Requirement: A signature selects one overload — "`node` SHALL also accept `--line <n>` for the same purpose. It SHALL keep the definition whose lines contain `n`" | Either reword the requirement to 'the definitions whose lines contain n' (more than one can be listed when no `--file` narrows them), or add a task that reduces the containing set to one and add a subtest with two containing definitions. |
| inv-4 | invariants | should-fix | beyond-scope | The containment and nearest rule fails for n above MaxInt32. The CLI flag and the MCP input are Go int, and narrowNodeMatches truncates to int32. A probe with line 4294967298 (which wraps to 2) kept the 2-4 def, although no def contains n and the nearest first line is 6. Nothing checks the range on the CLI or the MCP path, and no task adds a check. | internal/query/node.go:287 — "line := int32(*lineHint)"; scope basis: proposal.md > Out of scope — "a line number above the range of a 32-bit integer" | Either state a range for n in the requirement and reject values outside it in the CLI and MCP handlers, or clamp before the int32 conversion. Add a subtest at math.MaxInt32 and at MaxInt32+1. |
| inv-5 | invariants | should-fix | in-change | At n = 0 the code breaks the new rule. The CLI and the MCP handler both treat 0 as no hint, so `node log --line 0` lists every overload. The requirement says the definition whose first line is nearest to n is kept. The requirement does not exclude 0, and no test covers `--line 0`. | internal/cli/node.go:39 — "if line != 0 {" | Say in the requirement that n is a 1-based line and that 0 (or an absent `line`) means no line selection, and pin that zero case with a test. Or reject 0 and negative values. |
| tes-1 | test-coverage | should-fix | in-change | Task 1.2 covers three NODE-03 scenarios but its verify step gives no runnable command. | tasks.md > 1. Spec and tests — "Verify: it passes, and it fails when the `line != 0` check in `internal/cli/node.go` is removed" | Add to task 1.2: go test ./internal/cli -run TestLineFlagNarrowsNode -count=1 passes. |
| inv-6 | invariants | note | beyond-scope | narrowNodeMatches works over an unbounded domain of line values and definition sets. Three properties fit a property test: the result is never empty when the input is not empty, every kept def contains n when any def contains n, and otherwise the result has one def at minimum first-line distance. | internal/query/node.go:313 — "return narrowed // final guard: never assign an empty working set"; scope basis: proposal.md > What Changes — "New tests pin the rule through the CLI on the existing `Logger.java` fixture, and at the first and last line of a defini" | Add a pgregory.net/rapid (go.mod v1.3.0) property test next to TestNarrowNeverEmpty, modeled on internal/indexer/sync_property_test.go. Draw n across the full int range so it also catches inv-4. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Callers of one overload | 1.4 | covered |  |
| A signature that matches no overload | 1.4 | covered |  |
| A line inside one overload | 1.2 | partial |  |
| A line inside no definition | 1.2 | partial |  |
| Line zero selects nothing | 1.2 | partial |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: The change has no design.md. The proposal states Code: none, so no decision produces the requirement; the behavior already exists at head: cli/node.go:91 registers --line (short -l) on node only, callers.go:81 and callees.go:82 bind -l to --limit, impact.go has no --line, and query/node.go:271 narrowNodeMatches keeps the containing definition or else the nearest by first line. No unaddressed finding is raised for that reason.
- approach: Proposal citations hold at head: only NodeArgs in internal/mcp/tools.go:320 carries a line field; CallersArgs, CalleesArgs and ImpactArgs carry file and signature but no line; the requirement 'A file selects the definitions in it' exists at openspec/specs/graph-queries/spec.md:184; the current requirement text at spec.md:24 does say 'They SHALL keep accepting `--line`'.
- approach: Scenario 'A line selects one overload in node' is producible: javaextract.go:450 stores the parameter list text as Signature, :467-468 set StartLine and EndLine, detail.go:278 applies the line hint and a single remaining match renders through RenderNode with '**Location:** path:line' and '**Signature:**' (render_markdown.go:147,151).
- approach: Edge outside this lens: when two same-name definitions both contain n (nested definitions), narrowNodeMatches keeps both (node.go:298-299), while the requirement says it keeps 'the definition'. Left to spec-lint and invariants.
- approach: The claim that issue #323 lists --line on the three commands as the other option was not checked: this lens has no gh access.
- fit: The change has no design.md, so the components read are the delta requirement, its new scenario and the tasks.
- fit: No other copy of the wrong --line claim was found. docs/CLI-REFERENCE.md, internal/mcp/resources/node.md, internal/mcp/tools.go and internal/cli/node.go all give --line or line for node only, with the same containing-or-nearest wording as the delta.
- fit: graph-queries pairs each CLI selector with an MCP requirement ("The MCP tools expose the same selection", "The MCP tools select by file"). The delta adds no graph-queries MCP counterpart for line, but mcp-server already states that codegraph_node takes line and that it agrees with codegraph node, so this is not raised as a finding.
- fit: The prompt named no path for repository library-preference rows. None are read, and the change adds no code that a library could replace.
- fit: --signature and --file stay registered through addSignatureFlag and addFileFlag in internal/cli/selectorflags.go. The change adds no flag, so it adds no copy of the flag registration.
- invariants: I1: node --line n keeps the definitions whose [StartLine,EndLine] contain n (inclusive; EndLine 0 counts as StartLine) \| enforced at internal/query/node.go:290-300 \| boundary test none (only mid-range 35 in [30,40]); see inv-1
- invariants: I2: when no definition contains n, node keeps exactly one, the one whose StartLine is nearest n \| enforced at internal/query/node.go:301-308 (strict <, first in definitionsOf order wins ties) \| boundary test none for the tie; see inv-2
- invariants: I3: --line selects one definition \| enforced at none (all containing definitions are kept) \| boundary test none; see inv-3
- invariants: I4: the line rule holds for every int n \| enforced at none (int32 truncation at node.go:287) \| boundary test none; see inv-4
- invariants: I5: n = 0 means no line hint (CLI and MCP) \| enforced at internal/cli/node.go:39 and internal/mcp/tools.go:507 \| boundary test none; this conflicts with the spec sentence, see inv-5
- invariants: I6: narrowing never empties a non-empty match set \| enforced at internal/query/node.go:312-315 \| boundary test internal/query/node_test.go TestNarrowNeverEmpty (file hint with no match, single match)
- invariants: I7: callers, callees and impact accept no --line; node -l is --line, callers/callees -l is --limit \| enforced by cobra flag registration and the docs/CLI-REFERENCE.md golden checked by internal/cli/cli_reference_test.go (CLI-REFERENCE.md:785 lists -l, --line for node), plus TestShortFlagsConsistent for -l=--limit \| boundary test those tests
- invariants: I8: codegraph_node is the only MCP tool input with line \| enforced at internal/mcp/tools.go:320 (NodeArgs only), pinned by internal/mcp/tools_schema_drift_test.go and the testdata/wireoracle toolslist goldens \| boundary test those goldens (not read line by line)
- invariants: Weakened check: the old requirement claimed --line for callers, callees and impact, and none of them ever registered it. Dropping the claim removes no check a test or caller relies on, so there is no weakened finding.
- invariants: Probe: the handoff probe/main.go is a verbatim copy of narrowNodeMatches (node.go:225-316 minus narrowBySignature) run with the pinned Go. Lines 6, 8 and 9 on defs 2-4 and 6-8 gave [b], which shows the inclusive ends are correct today. Lines 4294967298 and 4294967296 gave [a] (int32 wrap). Equidistant line 4 gave the first-ordered def. Two containing defs gave both.
- invariants: The change has no design.md and so no decisions. Code impact is none, as the proposal says.
- spec-lint: design.md is absent from the change root (Glob lists proposal.md, specs/graph-queries/spec.md, tasks.md and .openspec.yaml only), so there are no design decisions or open questions to check for assumed answers.
- spec-lint: The delta MODIFIED requirement 'A signature selects one overload' has three scenarios, each with WHEN and THEN lines; the THEN steps name output, a file:line and a reported message, so they are observable.
- spec-lint: No BREAKING marker appears in proposal.md or the delta spec, so the BREAKING check matches.
- spec-lint: tasks.md names no build target; its commands are go test, openspec validate and codegraph node --help. None needed a unknown-target check.
- spec-lint: Verified against the repository at head: internal/cli/node.go registers --line; no callers, callees or impact command registers --line; internal/mcp/tools.go line 320 is the only MCP input with line; internal/query/node_test.go has TestNarrowNeverEmpty with the four named subtests; openspec/specs/graph-queries/spec.md has the requirement 'A file selects the definitions in it' that proposal.md cites.
- spec-lint: Task 1.2 says the test covers 'a line inside a definition and a line inside none'. The subtest 'lineHint inside no def falls back to the single nearest' covers that case, but the delta scenario 'A line selects one overload in node' does not state the fallback. That gap is for the test-coverage lens, not a form finding.
