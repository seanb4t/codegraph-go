# Proposal
Issue: #238

## Why

Graph consistency tests cannot detect declarations that the extractor never emits. The missed grouped Go variable exposed this gap, while current language tests often assert only nonempty output.

## What Changes

- Add a semantic evaluation report for Go, TypeScript, JavaScript, TSX, Python, and Java.
- Compare declarations, relationships, and public query results with independent expectations.
- Reuse SCIP indexers for Go, TypeScript, JavaScript, TSX, and Java, with a narrow CPython bridge for lexical facts.
- Report missing facts, extra facts, unsupported constructs, unresolved references, and unevaluated cases separately.
- Add small pull request fixtures and a scheduled evaluation against pinned source collections.
- Separate exact supported cases from issue-linked known gaps and preserve held-out acceptance cases.
- Replace broad language support claims with evidence and explicit limits.
- Reuse the graph consistency work from [PR #250](https://github.com/seanb4t/codegraph-go/pull/250) and the test execution work from [PR #242](https://github.com/seanb4t/codegraph-go/pull/242). Keep mutation testing in #222.

Out of scope: production extraction fixes, new languages, a general benchmark framework, release pipeline changes, and claims of complete dynamic-language resolution. Issues #239 and #240 own product changes.

## Capabilities

### New Capabilities

- `semantic-coverage`: Reproducible evaluation of graph facts and public query answers against independent evidence, with visible coverage boundaries.

### Modified Capabilities

None.

## Impact

The change adds developer tooling, authored expectations, test fixtures, and CI tasks. It tightens `testdata/golden/behavioral_tsjs_test.go` and language capability documentation.

Compiler tools remain development dependencies. Released binaries retain their current runtime requirements and public output formats. The report has its own versioned format.

Implementation reuses the changes that PR #250 and PR #242 merged. The initial report measures known omissions without silently accepting them as correct. Issues #239 and #240 consume the same evaluation format.
