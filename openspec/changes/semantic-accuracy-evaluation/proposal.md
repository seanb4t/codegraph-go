# Proposal
Issue: #238

## Why

Graph consistency tests cannot detect declarations that the extractor never emits. The missed grouped Go variable exposed this gap, while current language tests often assert only nonempty output.

## What Changes

- Add a semantic evaluation report for Go, TypeScript, JavaScript, TSX, Python, and Java.
- Compare declarations, relationships, and public query results with independent expectations.
- Report missing facts, extra facts, unsupported constructs, unresolved references, and unevaluated cases separately.
- Add small pull request fixtures and a scheduled evaluation against pinned source collections.
- Replace broad language support claims with evidence and explicit limits.
- Reuse the graph consistency and test execution work proposed in [PR #237](https://github.com/seanb4t/codegraph-go/pull/237). Keep mutation testing in #222.

Out of scope: production extraction fixes, new languages, a general benchmark framework, release pipeline changes, and claims of complete dynamic-language resolution. Issues #239 and #240 own product changes.

## Capabilities

### New Capabilities

- `semantic-coverage`: Reproducible evaluation of graph facts and public query answers against independent evidence, with visible coverage boundaries.

### Modified Capabilities

None.

## Impact

The change adds developer tooling, authored expectations, test fixtures, and CI tasks. It tightens `testdata/golden/behavioral_tsjs_test.go` and language capability documentation.

Compiler tools remain development dependencies. Released binaries retain their current runtime requirements and public output formats. The report has its own versioned format.

Implementation reuses the relevant implementation changes proposed in PR #237 when they land. Merging that proposal alone does not supply those changes. The initial report measures known omissions without silently accepting them as correct. Issues #239 and #240 consume the same evaluation format.
