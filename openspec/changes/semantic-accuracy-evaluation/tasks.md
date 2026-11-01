# Tasks

## 1. Evaluation inputs and comparison

- [ ] 1.1 Add authored case manifests and independent facts under `testdata/semantic/`. Include every initial language and all design matrix dimensions. Review the source-to-fact mapping and document the format beside the fixtures.
- [ ] 1.2 Add identity normalization and comparison under `tools/semantic-eval/`. Define independent identifier and construct-span mappings. Exercise multi-name declarations, Unicode, repeated names, and deduplication through `TestSemanticGo` and `TestSemanticTypeScript` product cases.
- [ ] 1.3 Add versioned JSON and text reports with raw denominators and exact differences from `TestSemanticGo`, `TestSemanticTypeScript`, `TestSemanticPython`, and `TestSemanticJava`. Report parse failures, unknown facts, unsupported and unevaluated cases, partitions, and missing inputs. Apply the spec's denominator rules. Assert nonempty required inputs inside the gate.
- [ ] 1.4 Apply exact known-gap matching to `TestSemanticGo`, `TestSemanticTypeScript`, `TestSemanticPython`, and `TestSemanticJava` results. Require issue, owner, and review date fields. Reject new, enlarged, expired, and stale exceptions inside the gate. Keep exceptions visible in accuracy totals. Do not add companion gate self-tests.

## 2. Independent reference adapters

- [ ] 2.1 Integrate pinned `scip-go` output and reviewed call annotations. Compare CodeGraph grouped declarations, build contexts, fields, aliases, and shadowing in `TestSemanticGo`.
- [ ] 2.2 Integrate pinned `scip-typescript` output for TypeScript, JavaScript, and TSX. Compare CodeGraph imports, destructuring, callable declarations, fields, and Unicode in `TestSemanticTypeScript`.
- [ ] 2.3 Add the narrow CPython AST and symbol-table bridge with reviewed dynamic cases. Compare CodeGraph module bindings, class attributes, receiver attributes, and shadowing in `TestSemanticPython`.
- [ ] 2.4 Integrate pinned `scip-javac` and aggregation through direct JDK invocation with `-proc:none`. Compare CodeGraph field lists, owners, imports, and missing classpaths in `TestSemanticJava`.
- [ ] 2.5 Document exact reference coverage, position encodings, tool pins, and annotations for unsupported reference facts. Qualify references through the four product acceptance tests. Reject missing tools inside the gate. Retain the existing-tool comparison from the design.

## 3. Public answers and language evidence

- [ ] 3.1 Add reviewed CLI and MCP cases for definitions, callers, callees, impact, and explore. Cover required facts, false matches, disambiguation, and budgets in `TestSemanticPublicQueries`.
- [ ] 3.2 Tighten `testdata/golden/behavioral_tsjs_test.go`. Assert exact language-qualified TypeScript relationships in `TestSemanticTypeScriptCorpusEdges`.
- [ ] 3.3 Add fresh-versus-incremental semantic comparisons for no-op synchronization, edits, renames, deletion, and restoration. Cover scoped graph facts and public answers in `TestSemanticLifecycle`. Match analysis contexts and report excluded categories.
- [ ] 3.4 Update `docs/LANGUAGE-CAPABILITY-MATRIX.md` with construct-level evidence and explicit limits. Trace every changed claim to a passing case or a named gap. Preserve registry-consistency tests for their existing purpose.

## 4. Reproducible execution

- [ ] 4.1 Add `task eval:semantic` for both manifest partitions and `task eval:semantic:dev` for development only. Run the four language tests and `TestSemanticPublicQueries` through these selections. Run offline after dependency installation. Document failure exits and report fields. Assert required tools inside the command without wrapper self-tests.
- [ ] 4.2 Add a small licensed TypeScript and TSX source collection through the existing corpus mechanism. Record its immutable revision and digest. Demonstrate nonempty expected TypeScript relationships through `TestSemanticTypeScriptCorpusEdges`.
- [ ] 4.3 Add development and acceptance manifest partitions, with acceptance cases under `testdata/semantic/heldout/`. Pin Go, Python, and Java inputs. Run `TestSemanticPublicQueries` with acceptance cases. Record sample sizes and unlabeled limits. Document separate source review for acceptance revisions.
- [ ] 4.4 Wire `task eval:semantic` into pull request CI and the source collections into scheduled or manual CI. Upload the semantic report with the existing workflow artifact step. Reuse existing actions and test reporting without induced workflow failures.

## 5. Integration acceptance

- [ ] 5.1 Run the complete initial evaluation and preserve its report. Account for every missing or extra fact through a fix or an exact issue-linked exception. Keep broader language gaps outside #239 visible.
- [ ] 5.2 Run the existing test targets changed by this work and the new semantic task. Record tool versions, runtime, case counts, and known gaps in the review evidence.
- [ ] 5.3 Record `<!-- changelog-exempt: developer evaluation tooling and tests only; no production behavior changes -->` in the PR body. Make sure that the final diff satisfies that exemption.
