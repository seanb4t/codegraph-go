# Tasks

## 1. Evaluation inputs and comparison

- [ ] 1.1 Add authored case manifests and independent facts under `testdata/semantic/`. Include every initial language and all design matrix dimensions. Review the source-to-fact mapping and document the format beside the fixtures.
- [ ] 1.2 Add semantic identity normalization and graph comparison under `tools/semantic-eval/`. Cover omissions, extra edges, Unicode, repeated names, and deduplication in `TestSemanticComparison`.
- [ ] 1.3 Add versioned JSON and text reports with raw denominators and exact differences. Cover empty samples, parse failures, unknown facts, and missing inputs in `TestSemanticReport`.
- [ ] 1.4 Add exact known-failure matching with issue, owner, and review date fields. Cover new, enlarged, expired, and stale exceptions in `TestSemanticKnownFailures`. Document that permitted exceptions still reduce accuracy.

## 2. Independent reference adapters

- [ ] 2.1 Add the Go reference adapter through compiler definitions, uses, and selections. Cover grouped declarations, build context, fields, aliases, and shadowing in `TestSemanticGoReference`.
- [ ] 2.2 Add the pinned TypeScript reference adapter for TypeScript, JavaScript, and TSX. Cover import resolution, destructuring, callable constants, fields, and Unicode in `TestSemanticTypeScriptReference`.
- [ ] 2.3 Add pinned Python AST and symbol-table extraction with reviewed dynamic cases. Cover module bindings, class attributes, receiver attributes, and shadowing in `TestSemanticPythonReference`.
- [ ] 2.4 Add pinned Java compiler analysis with annotation processing disabled. Cover field lists, owners, imports, and missing classpaths in `TestSemanticJavaReference`.
- [ ] 2.5 Document each adapter's exact fact coverage and tool pin. Run all four adapter tests against source-reviewed expectations. Make sure that missing tools produce explicit failures.

## 3. Public answers and language evidence

- [ ] 3.1 Add reviewed CLI and MCP cases for definitions, callers, callees, impact, and explore. Cover required facts, false matches, disambiguation, and budgets in `TestSemanticPublicQueries`.
- [ ] 3.2 Tighten `testdata/golden/behavioral_tsjs_test.go` to assert exact expected relationships with language-qualified endpoints. Cover the unrelated-Go-edges failure in `TestSemanticTypeScriptCorpusEdges`.
- [ ] 3.3 Add fresh-versus-incremental semantic comparisons for edits, renames, deletion, and restoration. Cover graph content and public answers in `TestSemanticLifecycle`.
- [ ] 3.4 Update `docs/LANGUAGE-CAPABILITY-MATRIX.md` with construct-level evidence and explicit limits. Trace every changed claim to a passing case or a named gap. Preserve registry-consistency tests for their existing purpose.

## 4. Reproducible execution

- [ ] 4.1 Add `task eval:semantic` through the existing Taskfile and pinned development tools. Cover success, semantic failure, and unavailable tools in `TestSemanticCommand`. Run the small suite offline after dependency installation. Document the command and report fields.
- [ ] 4.2 Add a small licensed TypeScript and TSX source collection through the existing corpus mechanism. Record its immutable revision and digest. Demonstrate nonempty expected TypeScript relationships through `TestSemanticTypeScriptCorpusEdges`.
- [ ] 4.3 Add reviewed held-out query cases and pin the existing Go, Python, and Java evaluation inputs. Run `TestSemanticPublicQueries` with the held-out selection. Record sample sizes and unlabeled source limits.
- [ ] 4.4 Wire the small suite into pull request CI and the source collections into scheduled or manual CI. Demonstrate report retention on semantic and infrastructure failures. Reuse existing workflow actions and test reporting.

## 5. Integration acceptance

- [ ] 5.1 Run the complete initial evaluation and preserve its report. Account for every missing or extra fact through a fix or an exact issue-linked exception. Keep broader language gaps outside #239 visible.
- [ ] 5.2 Run the existing test targets changed by this work and the new semantic task. Record tool versions, runtime, case counts, and known gaps in the review evidence.
- [ ] 5.3 Record `<!-- changelog-exempt: developer evaluation tooling and tests only; no production behavior changes -->` in the PR body. Make sure that the final diff satisfies that exemption.
