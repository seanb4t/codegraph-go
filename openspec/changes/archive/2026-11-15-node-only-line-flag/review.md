---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-15T12:48:30+00:00
head: 035b54d085ef191825c7dea8417255a5b319f463
lenses:
  spec-lint: not-applicable
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: ok
  invariants: ok
---
# Review

## Findings

| id | lens | severity | scope | claim | evidence | action |
|---|---|---|---|---|---|---|
| inv-1 | invariants | should-fix | in-change | The new requirement says callers, callees and impact take no --line. Only the missing flag registration enforces this. No test runs callers/callees/impact with --line or checks Lookup("line") on them, and the short-flags test skips a missing flag. Task 1.1 checked this by hand only. | specs/graph-queries/spec.md > Requirement: A line narrows the definitions that node shows (NODE-03) — "`callers`, `callees` and `impact` select by `--signature` and `--file` instead." | Add a CLI test that asserts cmd.Flags().Lookup("line") is nil for callers, callees and impact, and non-nil for node, so a later --line on those commands fails the suite. |
| inv-2 | invariants | should-fix | in-change | The MCP codegraph_node handler has its own line-zero guard, separate from the CLI one. No MCP test passes line, either 0 or a narrowing line, so the guard could be removed or inverted without any test failing. | internal/mcp/tools.go:507 — "if args.Line != 0 {" | Add a test in internal/mcp/overloads_test.go that calls codegraph_node on the Logger.java fixture with line 6, which keeps the one overload, and with line 0 or the line omitted, which keeps both definitions. |
| inv-3 | invariants | should-fix | in-change | No test passes the line one past a definition's last line where the nearest-start fallback would pick a different definition. An off-by-one such as end+1 >= line passes every test. The CLI line 7 and the unit lines 22, 30, 35 and 40 give the same result either way. | internal/query/node.go:294 — "if n.StartLine <= line && end >= line {" | Add a TestNarrowNeverEmpty subtest with line 21, one past a's end at 20. It must select b by nearest start, \|30-21\|=9 against \|10-21\|=11, and not a. |
| inv-4 | invariants | should-fix | in-change | narrowNodeMatches treats EndLine 0 as a one-line definition at StartLine. This is the null case of the containment rule. No test builds a node with EndLine 0. | internal/query/node.go:291 — "if end == 0 {" | Add a subtest with one node at StartLine 30, EndLine 0. Line 30 keeps it as containing, and line 31 does not count as containment. |
| inv-5 | invariants | should-fix | in-change | The requirement defines n as a 1-based line number with 0 meaning no line. Nothing rejects a negative line on the CLI or in MCP. A value like --line -5 reaches narrowNodeMatches and silently picks the definition with the nearest first line. | specs/graph-queries/spec.md > Requirement: A line narrows the definitions that node shows (NODE-03) — "where `n` is a 1-based line number and `0` means no line" | Do one of two things. Either reject n < 0 with a usage error in node.go and the MCP node handler, and test -1, or state in the requirement that a negative line is treated like any other line, or list it under Out of scope next to the 32-bit case. |
| ctl-not-run | test-coverage | should-fix | in-change | covered row 'A line inside one overload' has no verification run | review > not-run — "internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 6" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | in-change | covered row 'A line inside no definition' has no verification run | review > not-run — "internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 7" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | in-change | covered row 'Line zero selects nothing' has no verification run | review > not-run — "internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 0" | run the covering test and record it in verification-runs.txt |
| fit-1 | fit | note | in-change | The two new subtests for the first and the last line of a definition differ only in the line number (30 versus 40) and the subtest name; they are one table case varied by the parameter line. | internal/query/node_test.go:243 — "t.Run("lineHint on a def's first line narrows to that def", func(t *testing.T) {" | Fold the two subtests into one loop over line in {30, 40} that expects [b], as TestLineFlagNarrowsNode already does for lines 6 and 7; the parameter is line. |
| inv-6 | invariants | note | in-change | Two properties hold over any line and any non-empty set of definitions, which is an unbounded domain. The result is never empty, and it is a subset of the input in input order. Today they are pinned only at hand-picked lines. | specs/graph-queries/spec.md > Requirement: A line narrows the definitions that node shows (NODE-03) — "The line SHALL never leave `node` with no definition." | Add a pgregory.net/rapid (go.mod v1.3.0) property test for narrowNodeMatches over random start/end ranges and lines, modeled on internal/indexer/spill_roundtrip_test.go. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Callers of one overload | internal/query/overloads_test.go: TestCallersWithSignatureSelectsOneOverload | covered | pass |
| A signature that matches no overload | internal/query/overloads_test.go: TestCallersWithUnknownSignatureListsTheSignatures | covered | pass |
| A line inside one overload | internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 6 | covered | not-run |
| A line inside no definition | internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 7 | covered | not-run |
| Line zero selects nothing | internal/cli/overloads_cli_test.go: TestLineFlagNarrowsNode/line 0 | covered | not-run |

### Not run

- A line inside one overload: no verification run names this row
- A line inside no definition: no verification run names this row
- Line zero selects nothing: no verification run names this row

## Verification runs

- exit 0: `go test ./internal/query -run '^TestCallersWithSignatureSelectsOneOverload$' -count=1 -v`
- exit 0: `go test ./internal/query -run '^TestCallersWithUnknownSignatureListsTheSignatures$' -count=1 -v`
- exit 0: `go test ./internal/cli -run '^TestLineFlagNarrowsNode$/^line_6$' -count=1 -v`
- exit 0: `go test ./internal/cli -run '^TestLineFlagNarrowsNode$/^line_7$' -count=1 -v`
- exit 0: `go test ./internal/cli -run '^TestLineFlagNarrowsNode$/^line_0$' -count=1 -v`

## Dropped

None.

## Lens notes

- approach: The change has no design.md. The decisions were read from proposal.md (What Changes, Impact, Out of scope) and tasks.md, and the change touches only tests, so the behavior comes from code that already exists at head.
- approach: Requirement "A line narrows the definitions that node shows (NODE-03)" is produced by narrowNodeMatches in internal/query/node.go:271-314 (containing definitions kept, else the single nearest first line, never empty), by the line != 0 check in internal/cli/node.go:39 and internal/mcp/tools.go:507, and by buildNodeDetail in internal/query/detail.go:278.
- approach: Requirement "A signature selects one overload" with the --line sentence dropped matches head: --line is registered only in internal/cli/node.go:91, callers and callees bind -l to --limit (callers.go:81, callees.go:81), impact has no --line, and NodeArgs is the only MCP schema with line (internal/mcp/tools.go:320).
- approach: Cited facts hold at head: overloadCLIFixture puts log(String msg) on line 4 and log(String msg, Throwable t) on line 6 of Logger.java, so line 7 is nearest to line 6; the multi-definition header is "%d definitions named %q" (internal/query/render_results.go:139); TestNarrowNeverEmpty, TestCallersWithSignatureSelectsOneOverload and TestCallersWithUnknownSignatureListsTheSignatures exist; the requirement "A file selects the definitions in it" exists in openspec/specs/graph-queries/spec.md:184.
- approach: The out-of-scope claim about line numbers above the 32-bit range matches the int32(*lineHint) conversion at internal/query/node.go:287.
- approach: No code under cmd/ or internal/ other than tests is in diff-files.txt, so no undocumented deviation is possible. The proposal states no risks, open questions or migration plan.
- fit: The change has no design.md, so it names no decisions to read against existing patterns.
- fit: The diff touches tests and tasks.md only; no production component is created or changed.
- fit: TestLineFlagNarrowsNode reuses the existing overloadCLIFixture and the execCmd plus strings.Contains shape of TestSignatureFlagSelectsOneOverload in the same file; no new fixture or helper is added.
- fit: The new requirement carries the id NODE-03, which matches the existing tag on narrowNodeMatches (internal/query/node.go:260), the node --line flag help (internal/cli/node.go:91) and TestNarrowNeverEmpty's doc comment.
- fit: Pre-existing, not introduced by this change: the 0-means-no-line conversion to a nil *int is written twice, in internal/cli/node.go:38-41 and internal/mcp/tools.go:506-509. The change does not add a third copy.
- fit: The proposal names no next change, so no forward-misfit was assessed.
- invariants: I1: node --line 0 means no line \| enforced at internal/cli/node.go:39 \| boundary test internal/cli/overloads_cli_test.go TestLineFlagNarrowsNode/line 0
- invariants: I2: codegraph_node line 0 means no line \| enforced at internal/mcp/tools.go:507 \| boundary test none (inv-2)
- invariants: I3: a line never leaves node with no definition \| enforced at internal/query/node.go:298-315 (nearest fallback and final guard) \| boundary test internal/query/node_test.go TestNarrowNeverEmpty lineHint inside no def; property-candidate inv-6
- invariants: I4: containment is inclusive at both StartLine and EndLine \| enforced at internal/query/node.go:294 \| boundary test TestNarrowNeverEmpty first-line (30) and last-line (40) subtests; one-past-end edge missing (inv-3)
- invariants: I5: every containing definition is kept, not only the first \| enforced at internal/query/node.go:295 \| boundary test TestNarrowNeverEmpty lineHint inside two defs keeps both
- invariants: I6: EndLine 0 counts as a one-line definition \| enforced at internal/query/node.go:291 \| boundary test none (inv-4)
- invariants: I7: line narrowing applies only when more than one definition remains \| enforced at internal/query/node.go:272 and 286 \| boundary test TestNarrowNeverEmpty single match subtest
- invariants: I8: callers, callees and impact have no --line, and only codegraph_node has a line property \| enforced at the absence of the flag and of struct fields (internal/mcp/tools.go CallersArgs/CalleesArgs/ImpactArgs) \| boundary test none (inv-1)
- invariants: I9: n is 1-based (no negative) \| enforced at none \| boundary test none (inv-5)
- invariants: Signature narrowing runs before line narrowing (internal/query/detail.go:268-278), so a line cannot select a definition that --signature already excluded.
- invariants: The int32 cast at internal/query/node.go:287 truncates a line above 2^31-1. The proposal puts this out of scope, so it gets no finding.
- invariants: The diff removes no check and weakens no requirement. The MODIFIED requirement drops only the '--line' sentence, which no flag ever enforced.
- invariants: I ran no enforcement script: the enforcement points are Go functions and not scripts.
- test-coverage: Tests were not run, per lens rules.
- test-coverage: Task 1.1 (help output and MCP schema check) has no test; it is a manual verification task, not a scenario.
- test-honesty: TestLineFlagNarrowsNode and the three new TestNarrowNeverEmpty subtests assert the spec's THEN (Logger.java:6, signature, no '2 definitions', '2 definitions named "log"' for line 0; both containing defs kept). Expected values are literals independent of the code under test. No disabled tests, no missing assertions.

## Plan review (2026-11-15)

Verdict approve-with-notes at head 2b84b870785acbfe4e82957b1893f3e3f808fe06.

### Findings

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

### Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Callers of one overload | 1.4 | covered |  |
| A signature that matches no overload | 1.4 | covered |  |
| A line inside one overload | 1.2 | partial |  |
| A line inside no definition | 1.2 | partial |  |
| Line zero selects nothing | 1.2 | partial |  |

### Not run

Plan phase: no verification runs.
