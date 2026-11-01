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

Use a small developer entry point, `task eval:semantic`, with authored fixtures under `testdata/semantic/`. Keep compiler adapters under `tools/semantic-eval/`. Use native Go tests for integration. Avoid a separate test framework or generated expectation language.

The report is the new developer-facing behavior. It has a versioned JSON format and a short text rendering. This format does not alter production CLI or MCP responses.

Each case records language, construct, source digest, configuration, expected facts, evidence, and status. Authored expectations remain separate from generated observations. The report writer owns generated report structure.

Use these reference sources within their actual limits:

| Language | Independent reference | Boundary |
| --- | --- | --- |
| Go | `go/parser`, `go/ast`, and `go/types` definitions, uses, and selections | Match build tags, platform, and package context. Do not equate interface method shape with concrete runtime dispatch. |
| TypeScript, JavaScript, TSX | Pinned TypeScript compiler program and symbol APIs | Match module resolution and compiler configuration. Keep JS and TSX results separate. |
| Python | Pinned CPython `ast` and `symtable`, plus reviewed expectations | Establish declarations and lexical bindings. Hand-review dynamic attributes and calls rather than claiming compiler-proven targets. |
| Java | Pinned JDK `JavacTask` and `Trees.getElement` | Parse and analyze with `-proc:none`. Record missing dependencies. Do not execute source, processors, Maven, or Gradle. |

The [Go type API](https://pkg.go.dev/go/types) exposes definitions and uses. The [TypeScript compiler guide](https://github.com/microsoft/TypeScript/wiki/Using-the-Compiler-API) describes programs and type analysis.

Python documents separate [AST](https://docs.python.org/3/library/ast.html) and [symbol table](https://docs.python.org/3/library/symtable.html) interfaces. Neither establishes arbitrary dynamic call targets.

Java provides [parse and analysis operations](https://docs.oracle.com/en/java/javase/25/docs/api/jdk.compiler/com/sun/source/util/JavacTask.html) and [tree-to-element lookup](https://docs.oracle.com/en/java/javase/25/docs/api/jdk.compiler/com/sun/source/util/Trees.html). Its [compiler documentation](https://github.com/openjdk/jdk25u-dev/blob/master/src/jdk.compiler/share/man/javac.md) defines `-proc:none`.

Pin adapter tools through the repository's development tool mechanism. Record exact versions in reports. Reuse the repository's TypeScript pin when its compiler API meets the contract. Never depend on an unpinned upstream default branch.

Normalize declarations to language, relative path, owner, role, name, and source span. Convert positions to UTF-8 byte offsets. Test Unicode and repeated names explicitly. Do not compare CodeGraph hashes with compiler-generated identifiers.

Normalize relationships to source declaration, kind, and target declaration. CodeGraph deduplicates repeated relationships with those endpoints. Keep call-site counts separate. Normalize callable constants consistently so existing arrow-function identities do not create false duplicates.

Report precision as correct emitted facts divided by evaluated emitted facts. Report recall as correct emitted facts divided by expected facts. Show both denominators, exact differences, and unavailable categories.

Retain omitted files in source coverage when expectations exist. For incompletely labeled source, report unlabeled observations as unknown. Do not claim whole-repository precision from a reviewed sample.

The first inventory covers the following dimensions:

| Dimension | Required cases |
| --- | --- |
| Declarations | Grouped bindings, destructuring, constants, fields, methods, nested owners, repeated names, exported and private symbols |
| Resolution | Imports, aliases, cross-file references, initializer references, receivers, shadowing, unresolved and ambiguous targets |
| Source boundaries | Unicode, comments, literals, malformed input, excluded files, generated files, platform and build context |
| Lifecycle | Fresh indexing, no-op synchronization, edits, renames, deletion, restored files, rebuild migration |
| Queries | Definitions, callers, callees, impact, explore, disambiguation, required context, forbidden facts, truncation |

Record decorators, Java records and enums, overloads, interface dispatch, and dynamic attributes even when the initial result is a known gap. Other registered languages remain explicitly unevaluated by this change. Add issues for gaps beyond #239 instead of expanding its scope automatically.

Run small reviewed fixtures on pull requests. Run pinned source collections on a schedule and on demand. Reuse `internal/corpora` and existing corpus tasks. Add genuine TypeScript and TSX input because Hugo's JavaScript files do not establish TypeScript coverage.

Choose a small licensed TypeScript collection during implementation and record its exact commit and digest before the first accepted run. Existing Java and Python collections remain useful. Each sample needs a declared denominator and independent labels.

Hold a reviewed subset apart from routine fixture tuning. Keep its inventory visible. Acceptance changes require a stated language-contract reason and review of the source evidence. Never regenerate expectations from CodeGraph to make a run pass.

Supported fixture cases require zero unexpected missing or extra facts. Initial known failures list exact differences, an owner, an issue, and a review date. They remain failures in accuracy totals even when temporarily permitted by the regression gate. Expired, enlarged, or stale exceptions fail the gate.

Install the report and known-failure inventory first. Then enable the small acceptance gate. Product fixes in #239 remove exceptions as their cases pass. This order avoids weakening expectations to land the measurement change.

Publish the capability matrix with links to evidence and explicit limits. Do not repurpose runtime `pendingRefs` or indexed-file percentages as semantic recall. Rollback removes the evaluation gate without changing production indexes.

## Risks / Trade-offs

- Reference adapters can share normalization mistakes with the product. Use small source-reviewed facts alongside compiler output and show exact mismatches.
- Compiler tools add CI cost. Keep pull request fixtures small and reuse pinned installations. Reserve large collections for scheduled runs.
- Dynamic targets cannot always be established. Keep unknown and heuristic categories visible and separate from exact facts.
- Known failures can become permanent exceptions. Require owners, issues, review dates, and exact mismatch matching.
- A single aggregate score can hide regressions. Gate each supported language and construct category separately.
