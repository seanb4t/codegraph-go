# Semantic evaluation fixtures

These fixtures measure what CodeGraph says about source code. A person authors the expected facts in each `case.json`. Reference tools then qualify those facts. The reference tools never write the expectation.

## Run the evaluation

Run the gate over both partitions:

```bash
mise exec -- task eval:semantic
```

Run only the development partition while you work:

```bash
mise exec -- task eval:semantic:dev
```

Both tasks write `test-results/semantic/semantic-report.json` and `semantic-report.txt`. The first run downloads the pinned CPython through `uv`. After that download and `mise install`, the tasks need no network.

The task exits with 0 when every selected case passes. It exits with a nonzero status for each of these failures:

| Failure | Where the report shows it |
|---|---|
| A case has an unexpected difference | `cases[].failures`, `cases[].mismatches` |
| An exception is new, enlarged, stale or past its review date | `cases[].failures` |
| A reference disagrees with an authored fact | `cases[].failures` |
| A reference tool or a required input is missing | `missingInputs` |
| A required dimension has no case | `missingInputs` |
| A tool or a case cannot run | `infrastructureFailures` |

## Report fields

| Field | Meaning |
|---|---|
| `schemaVersion` | The report format version. |
| `partitions` | The partitions that this run evaluated. |
| `tools` | The exact version of each reference tool. |
| `totals` | Raw counts, denominators, precision and recall, per language and fact kind. |
| `statusCounts` | Cases per language and status. |
| `partitionSizes` | Cases per partition. |
| `cases` | One result per case, with its source digest and exact differences. |

`precision` and `recall` are `null` in JSON and `unavailable` in text when the denominator is zero. A wrong span counts as one false positive, one false negative and one wrong span. Exceptions stay in the totals, so a known gap lowers the score that the report shows.

## Case layout

A case is a directory, `<partition>/<language>/<case>/`, with a `case.json` file and a `src/` directory. The `development/` directory holds the development partition. The `heldout/` directory holds the acceptance partition.

| Field | Meaning |
|---|---|
| `id` | `<language>/<case>`, unique across both partitions. |
| `dimension` | `declarations`, `resolution`, `source-boundaries`, `lifecycle` or `queries`. |
| `status` | See "Status" below. |
| `targetSpec` | The specification file and the revision that defines the expected behavior. |
| `expectationRevision` | Increase this number when you change a reviewed expectation. |
| `labeling` | `complete` or `sampled`. In `complete`, an unlabeled fact is a false positive. In `sampled`, an unlabeled fact is an unknown fact, and it enters no denominator. |
| `declarations` | Expected declarations: path, owner, role, name, identifier position and span. |
| `relationships` | Expected edges, by declaration `id`. |
| `forbidden` | Edges that must not exist. |
| `unknown` | Edges that exceed the reference evidence. |
| `referenceExclusions` | Reference facts that this case does not expect, each with a reason. |
| `queries` | Public CLI and MCP questions with required facts, forbidden facts and limits. |
| `lifecycle` | Edit steps that compare incremental sync with a fresh index. |
| `excluded` | Files that CodeGraph must leave unindexed, with the reason. |
| `parseFailures` | Files with syntax errors. Their expected declarations stay in the recall denominator. |
| `reviewedOnly` | The reason that no reference can read this case. |
| `exception` | The exact accepted differences of a `known-gap` case. |

Unknown fields fail the case load. The identifier at `at` must equal `name` in the source.

## Status

| Status | Meaning |
|---|---|
| `supported` | The case must match exactly. |
| `known-gap` | The behavior is inside the contract and is missing. The case needs `exception` with an issue, an owner, a review date and the exact mismatches. |
| `unsupported` | The construct is outside the target specification. The report counts what the graph emits for it. |
| `unevaluated` | No independent evaluation exists yet. |

When a fix removes a difference, the exception goes stale and the gate fails. Remove the difference from the exception in the same change.

## Span rules

A span rule names the source construct that a declaration span covers. Lines start at 1. Columns start at 0 and count UTF-8 bytes. The end position is exclusive.

| Rule | Span |
|---|---|
| `go-func-decl` | The whole function or method declaration. |
| `go-type-spec` | The type spec, without the `type` keyword. |
| `go-value-spec` | The value spec, without `var` or `const`. Names in one spec share this span. |
| `go-field-decl` | The field declaration. Names in one declaration share this span. |
| `java-type-declaration` | The whole type declaration, with modifiers. |
| `java-method-declaration` | The whole method declaration, with modifiers. |
| `java-field-declaration` | The whole field declaration, with modifiers and the semicolon. Names in one declaration share this span. |
| `py-function-definition` | The `def` statement and its body. |
| `py-class-definition` | The `class` statement and its body. |
| `py-assignment` | The assignment or annotation statement that binds the name. Names in one statement share this span. |
| `py-receiver-assignment` | The assignment statement that binds `self.<name>`. |
| `ts-function-declaration` | The whole function declaration, without `export`. |
| `ts-class-declaration` | The whole class declaration, without `export`. |
| `ts-interface-declaration` | The whole interface declaration, without `export`. |
| `ts-type-alias` | The whole type alias, without `export`. |
| `ts-method-definition` | The whole method definition. |
| `ts-field-definition` | The class field definition with its modifiers, without the semicolon. |
| `ts-lexical-declarator` | The declarator that binds the name, without the keyword and the semicolon. Names in one pattern share this span. |
| `ts-callable-declarator` | The declarator with its function or arrow function value, without the keyword and the semicolon. |

## References

`mise.toml` pins each reference tool, and `PythonVersion` in `tools/semantic-eval/adapters.go` pins CPython. The report records the exact versions. A missing tool fails the run. It never skips a case.

| Language | Reference | Establishes | Column unit | Does not establish |
|---|---|---|---|---|
| Go | `scip-go` | Definitions, enclosing ranges, reference occurrences | UTF-8 bytes | Call kinds, runtime dispatch |
| TypeScript, TSX, JavaScript | `scip-typescript` | Definitions, reference occurrences | UTF-16 code units | Enclosing ranges, call kinds |
| Java | `scip-javac` compiler plugin with the pinned JDK and `-proc:none` | Definitions, enclosing ranges, reference occurrences | UTF-16 code units | Call kinds, facts that need a missing classpath entry |
| Python | CPython `ast` and `symtable` | Declarations, lexical bindings | UTF-8 bytes | Dynamic attributes, call targets |

Each reference qualifies the authored facts in both directions. An authored fact that the reference does not hold fails. A reference fact with no label fails, unless `referenceExclusions` names it with a reason.

A reference occurrence does not prove a call. Every `calls` edge is a reviewed annotation with `evidence: reviewed`. A `site` lets the reference confirm that the target name at that position is the expected symbol.

The `scip` CLI converts each index to JSON, so the module needs no SCIP dependency. The Java adapter reads one index per source file, so it needs no aggregation step. `CODEGRAPH_SCIP_JAVAC_CLASSPATH` lists the compiler plugin and its three dependencies, which `mise.toml` pins as raw jars. The task sets this variable.

### Comparison with existing tools

| Tool | Decision | Reason |
|---|---|---|
| `scip-go`, `scip-typescript`, `scip-javac` | Used | Each tool uses the compiler front end of its language and gives definitions with source ranges. |
| `scip-python` | Not used | It adds Pyright inference. The CPython `ast` and `symtable` interfaces are narrower and have no inference to qualify. |
| A custom compiler walker | Not used | An existing indexer costs less to keep and is independent of this repository. The small CPython bridge is the one exception, because `ast` and `symtable` are the reference. |
| Maven or Gradle for Java | Not used | The evaluation must not execute project code, processors or build logic. |

## Corpus samples

`TestSemanticCorpusSamples` and `TestSemanticTypeScriptCorpusEdges` in `testdata/golden/behavioral_tsjs_test.go` read pinned repositories from `corpora/manifest.json`. `TestSemanticCorpusSamples` is a declared sample. Its reviewed edges and its known gaps are the labeled denominator. Every other edge is unlabeled, and the test logs the sample size and the unlabeled count. Run the test with `-v` to read them:

```bash
mise exec -- task corpora:fetch
mise exec -- go test -count=1 -v -run 'TestSemanticCorpusSamples|TestSemanticTypeScriptCorpusEdges' ./testdata/golden/
```

Unlabeled edges are unknown. They give no precision and no recall. Do not claim accuracy for a whole repository from a sample. `TestSemanticTypeScriptCorpusEdges` labels every cross-file TypeScript `calls` edge under `src/` of its corpus, so an extra edge fails there.

## Acceptance review

The acceptance partition exists so that a change cannot fit the score by tuning to a known case.

1. Author each acceptance case from the source and the target specification, before you look at CodeGraph output for that case.
2. Do not reuse a development fixture as an acceptance fixture.
3. Have a second reviewer read each new acceptance revision against its `src/` directory. The review covers the facts, not the CodeGraph result.
4. If an acceptance case fails, fix the product or file an issue. Do not change the expectation to match CodeGraph output. If the expectation is wrong, change it, increase `expectationRevision`, and record the reason in the pull request.
