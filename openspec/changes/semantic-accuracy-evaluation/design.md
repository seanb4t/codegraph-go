# Design

## Context

See [proposal.md](proposal.md) for motivation. This design addresses the evidence gap outside [PR #237](https://github.com/seanb4t/codegraph-go/pull/237).

The current [TypeScript behavioral test](../../../testdata/golden/behavioral_tsjs_test.go) counts edges across the Hugo graph. That total cannot establish TypeScript relationship correctness.

[Capability tests](../../../internal/indexer/capability/matrix_test.go) establish registry consistency and basic extraction. They do not establish coverage of every declaration form. [File coverage](../../../internal/query/coverage.go) measures successful indexing, not semantic accuracy.

Frozen goldens remain useful regression evidence. Expected output captured through CodeGraph is not an independent reference for facts that CodeGraph omits.

## Goals / Non-Goals

Goals: Detect omitted facts and invented relationships with inspectable evidence. Make each support claim traceable to source, expectations, and an acceptance result.

Non-goals: Replace the test runner, implement production compilers, or prove arbitrary runtime behavior. Structural invariants remain in #237. Mutation testing remains in #222.

## Decisions

Use a small developer entry point, `task eval:semantic`, with authored fixtures under `testdata/semantic/`. Keep reference integration under `tools/semantic-eval/`. Use native Go tests and the existing test runner. Avoid a separate test framework or generated expectation language.

`task eval:semantic:dev` selects only development cases for local iteration. `task eval:semantic` runs both development and acceptance partitions and is the pull request gate. Both commands use the same product tests with an explicit manifest selection.

Assert nonempty inputs, required tools, exact differences, and exception validity inside the acceptance gate. Do not add companion gate self-tests or simulated workflow failures. Exercise normalization through tests of CodeGraph extraction and public queries against independent facts.

The report is the new developer-facing behavior. It has a versioned JSON format and a short text rendering. This format does not alter production CLI or MCP responses.

Each case records language, construct, source digest, configuration, expected facts, evidence, status, and the target specification revision. Authored expectations remain separate from generated observations. The report writer owns generated report structure.

The evaluated contract is the cited target specification, including planned forms before implementation. Initially, missing #239 forms are known-gap cases linked to #239. Explicitly excluded forms are unsupported. Later language changes, including #240, extend the evaluated contract when they register independently labeled suites.

Use these reference sources within their actual limits:

| Language | Independent reference | Boundary |
| --- | --- | --- |
| Go | Pinned `scip-go` definitions and reference occurrences | Match build tags, platform, package, and production/test contexts. Do not infer runtime dispatch from a reference. |
| TypeScript, JavaScript, TSX | Pinned `scip-typescript` definitions and reference occurrences | Match module resolution and compiler configuration. Keep JS and TSX results separate. |
| Python | Pinned CPython `ast` and `symtable`, plus reviewed expectations | Establish declarations and lexical bindings. Hand-review dynamic attributes and calls rather than claiming compiler-proven targets. |
| Java | Pinned `scip-javac` compiler plugin and SCIP aggregation | Invoke the pinned JDK directly with `-proc:none` and an explicit classpath. Do not execute source, processors, Maven, or Gradle. |

Prefer existing indexers over custom compiler walkers. [scip-go](https://github.com/scip-code/scip-go) and [scip-typescript](https://github.com/sourcegraph/scip-typescript) supply independent definitions and references. [scip-java manual integration](https://github.com/scip-code/scip-java/blob/main/docs/manual-configuration.md) supports direct compiler-plugin invocation without a project build tool.

[SCIP](https://github.com/scip-code/scip/blob/main/scip.proto) stores symbol occurrences and source ranges. An ordinary reference occurrence does not establish a call edge. Use reviewed annotations for call kinds, ownership, or other facts that the selected indexer cannot establish. Record these boundaries per adapter.

Python documents separate [AST](https://docs.python.org/3/library/ast.html) and [symbol table](https://docs.python.org/3/library/symtable.html) interfaces. Neither establishes arbitrary dynamic call targets. [scip-python](https://github.com/sourcegraph/scip-python/tree/scip) uses Pyright inference. Keep a narrow CPython bridge because this contract measures lexical bindings separately from inferred types. Review dynamic attributes and calls as source annotations.

Java's [compiler documentation](https://github.com/openjdk/jdk25u-dev/blob/master/src/jdk.compiler/share/man/javac.md) defines `-proc:none`. Record missing dependencies as reference limitations. Require a qualification case for each chosen indexer before accepting its facts.

Pin adapter tools through the repository's development tool mechanism. Record exact versions in reports. Reuse the repository's TypeScript pin when its compiler API meets the contract. Never depend on an unpinned upstream default branch.

Normalize declarations to language, relative path, owner, role, name, and the independently located identifier occurrence. Convert each reference format's declared position encoding to UTF-8 byte offsets. Do not compare CodeGraph hashes with compiler-generated identifiers.

Define a source-reviewed span rule for each declaration construct. CodeGraph can span a complete declaration while the reference spans its identifier. Match each identifier once by owner, role, name, and its independently derived declaration envelope. Require the CodeGraph span to match that construct's span rule. Containment alone cannot establish a match.

For Go multi-name specs, map each named binding to its own identifier within the shared declaration envelope. Preserve distinct occurrences for repeated same-line names. Report incorrect spans separately from missing identities. Include Unicode and nested equal names in product acceptance cases without injecting faults into observed output.

Normalize relationships to source declaration, kind, and target declaration. CodeGraph deduplicates repeated relationships with those endpoints. Keep call-site counts separate. Normalize callable constants consistently so existing arrow-function identities do not create false duplicates.

Report precision as correct emitted facts divided by evaluated emitted facts. Report recall as correct emitted facts divided by expected facts. Show both denominators, exact differences, and unavailable categories.

Expected declarations missing from a failed or omitted file remain false negatives in declaration recall. For incompletely labeled source, report unlabeled observations as unknown. Exclude unknown facts from both precision and recall denominators and report their raw counts separately. Unknown targets cannot become true positives. Do not claim whole-repository precision from a reviewed sample.

When the reference cannot establish a dynamic target, classify any emitted target edge as unknown and outside reference scope. When reviewed source explicitly forbids an edge, count an emitted match as a false positive. These are separate expectation labels, not interchangeable uses of unresolved.

The first inventory covers the following dimensions:

| Dimension | Required cases |
| --- | --- |
| Declarations | Grouped bindings, destructuring, constants, fields, methods, nested owners, repeated names, exported and private symbols |
| Resolution | Imports, aliases, cross-file references, initializer references, receivers, shadowing, unresolved and ambiguous targets |
| Source boundaries | Unicode, comments, literals, malformed input, excluded files, generated files, platform and build context |
| Lifecycle | Fresh indexing, no-op synchronization, edits, renames, deletion, restored files |
| Queries | Definitions, callers, callees, impact, explore, disambiguation, required context, forbidden facts, truncation |

Inventory decorators, Java record components, enum-specific declarations, general overload resolution, interface dispatch, and dynamic attributes. Label each construct unsupported while it remains outside the target specification. Reserve known-gap for missing facts within that contract. Other languages remain unevaluated until their separate changes register independent suites. Track future scope separately instead of expanding #239 automatically.

Run small reviewed fixtures on pull requests. Run pinned source collections on a schedule and on demand. Reuse `internal/corpora` and existing corpus tasks. Add genuine TypeScript and TSX input because Hugo's JavaScript files do not establish TypeScript coverage.

Choose a small licensed TypeScript collection during implementation and record its exact commit and digest before the first accepted run. Existing Java and Python collections remain useful. Each sample needs a declared denominator and independent labels.

Declare `development` and `acceptance` partitions in authored manifests. Keep acceptance cases under `testdata/semantic/heldout/` and exclude them from routine tuning commands. Run the acceptance partition in pull request CI and record its size and expectation revision.

Review acceptance changes separately from extractor changes, with a language-contract reason and source evidence. This is a review policy, not another CI guard. The public corpus is not a secret or statistically unbiased benchmark. Never regenerate expectations from CodeGraph to make a run pass.

Use distinct case statuses: `supported`, `known-gap`, `unsupported`, and `unevaluated`. Supported cases require zero missing or extra facts. Known-gap cases list exact differences, an owner, an issue, and a review date. They remain failures in accuracy totals even when temporarily permitted by the regression gate. Expired, enlarged, or stale exceptions fail the gate.

Remove each repaired mismatch from its exception. Keep the case known-gap while any listed mismatch remains. Change it to supported only when no differences remain and the exception is removed.

Unsupported cases are inventoried constructs outside the declared product contract. Unevaluated cases lack independent evaluation. Report their case counts and any observed fact counts separately, outside both accuracy denominators and the regression gate. They never establish support. Manifest labels control this scope, not whatever facts a reference tool happens to emit.

Compare fresh and incremental semantic facts and public answers under the same configuration and analysis context. Include no-op synchronization, edits, renames, deletion, and restoration. Name the evaluated fact subset and retain excluded categories in reports. Do not imply route-graph equality beyond the boundary defined by `index-invariants` in PR #237.

Install the report and known-failure inventory first. Then enable the small acceptance gate. Product fixes in #239 remove exceptions as their cases pass. This order avoids weakening expectations to land the measurement change.

Publish the capability matrix with links to evidence and explicit limits. Do not repurpose runtime `pendingRefs` or indexed-file percentages as semantic recall. Rollback removes the evaluation gate without changing production indexes.

## Risks / Trade-offs

- Reference adapters can share normalization mistakes with the product. Use small source-reviewed facts alongside compiler output and show exact mismatches.
- Compiler tools add CI cost. Keep pull request fixtures small and reuse pinned installations. Reserve large collections for scheduled runs.
- Dynamic targets cannot always be established. Keep unknown and heuristic categories visible and separate from exact facts.
- Known failures can become permanent exceptions. Require owners, issues, review dates, and exact mismatch matching.
- A single aggregate score can hide regressions. Gate each supported language and construct category separately.
