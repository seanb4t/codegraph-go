# Tasks

## 1. Spec and tests

- [ ] 1.1 Check that the delta of `specs/graph-queries/spec.md` matches the CLI and the MCP tools. Verify: `codegraph node --help` lists `--line`, the `--help` of `callers`, `callees` and `impact` does not, and the `codegraph_node` input schema in `internal/mcp/tools.go` is the only MCP schema with `line`.
- [ ] 1.2 Add a test `TestLineFlagNarrowsNode` to `internal/cli/overloads_cli_test.go` on `overloadCLIFixture`. It covers the three scenarios of "A line narrows the definitions that node shows". It asserts that `node log --line 6` and `node log --line 7` print `Logger.java:6` with `(String msg, Throwable t)` and not `2 definitions`, and that `node log --line 0` prints `2 definitions named "log"`. Verify: `go test ./internal/cli -run TestLineFlagNarrowsNode -count=1` passes, and the test fails when the `line != 0` check in `internal/cli/node.go` is removed or when `narrowNodeMatches` ignores `lineHint`.
- [ ] 1.3 Add three subtests to `TestNarrowNeverEmpty` in `internal/query/node_test.go`. A line equal to the first line of a definition and a line equal to its last line both select that definition. Two definitions on lines 10 to 20 and 12 to 15 with the line 13 keep both definitions. Verify: `go test ./internal/query -run TestNarrowNeverEmpty -count=1` passes, and the third subtest fails when `narrowNodeMatches` keeps only the first containing definition.
- [ ] 1.4 Name the tests that cover the two scenarios that the delta repeats unchanged. Verify: `go test ./internal/query -run 'TestCallersWithSignatureSelectsOneOverload|TestCallersWithUnknownSignatureListsTheSignatures' -count=1` passes. The first test covers "Callers of one overload", and the second covers "A signature that matches no overload".
- [ ] 1.5 Run `openspec validate node-only-line-flag --strict`. Verify: it passes.
- [ ] 1.6 Record `<!-- changelog-exempt: spec correction and tests, no user-visible change -->` in the PR body.
