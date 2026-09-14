---
status: complete
phase: 07-guards-that-cannot-fire
source: 07-01-SUMMARY.md, 07-02-SUMMARY.md, 07-03-SUMMARY.md, 07-04-SUMMARY.md
started: 2026-09-09T19:35:25Z
updated: 2026-09-09T20:46:44Z
---

## Current Test

[testing complete]

## Tests

### 1. CheckRegression refuses current.PeakRSSBytes=0 with an error naming PeakRSSBytes
expected: CheckRegression(baseline, current, ceiling=1) with current.PeakRSSBytes=0 and an otherwise-matching frame returns a non-nil error naming PeakRSSBytes
result: pass
source: automated
coverage_id: 07-01/D1

### 2. CheckRegression refuses current.FilesPerSec=0 as an invalid current reading
expected: CheckRegression with current.FilesPerSec=0 returns an error naming FilesPerSec as an invalid current reading, not a misattributed throughput-regression message
result: pass
source: automated
coverage_id: 07-01/D2

### 3. Degenerate-current rows were watched failing pre-fix (RED transcript in mutation log)
expected: Both degenerate-current rows were watched failing against the byte-identical committed regression.go before the fix, and that failure is pasted verbatim in 07-MUTATION-LOG.md
result: pass
source: automated
coverage_id: 07-01/D3

### 4. 07-MUTATION-LOG.md exists, is committed, and is ready for appended families
expected: 07-MUTATION-LOG.md exists, is committed, and ends with a horizontal rule ready for three appended families
result: pass
source: automated
coverage_id: 07-01/D4

### 5. Archtest fails when internal/query acquires a wire-layer dependency (transitive)
expected: A persisted archtest in internal/query/archtest fails when internal/query acquires a wire-layer dependency (internal/uiserver, internal/mcp, internal/uiproto, or connectrpc.com/connect) anywhere in its resolved transitive dependency set, not merely direct imports
result: pass
source: automated
coverage_id: 07-02/D1

### 6. Archtest fails when production internal/query depends on the internal/indexer root
expected: The same archtest fails when internal/query's production compilation unit acquires a dependency on the internal/indexer ROOT package, while continuing to allow internal/indexer/goextract and internal/indexer/nodeid
result: pass
source: automated
coverage_id: 07-02/D2

### 7. Archtest reports package count and refuses zero-load or missing production package
expected: The archtest reports the number of packages loaded and refuses to pass when that number is zero or when the production internal/query package is absent from the load
result: pass
source: automated
coverage_id: 07-02/D3

### 8. Archtest positive controls prove the walk resolved beyond one hop
expected: Positive controls: internal/graphstore AND internal/indexer/goextract present in the resolved set; internal/parser present but NOT a direct import, proving the walk resolved beyond one hop
result: pass
source: automated
coverage_id: 07-02/D4

### 9. Archtest package doc names T-01-18 and explains the production-only scoping
expected: The archtest package doc names threat T-01-18 and records why the internal/indexer-root rule is scoped to the production compilation unit, naming engine_test.go
result: pass
source: automated
coverage_id: 07-02/D5

### 10. Mutation log family (b) carries two verbatim RED transcripts with byte-clean reverts
expected: 07-MUTATION-LOG.md family (b) carries two verbatim RED transcripts, one per forbidden set, each with a pre-mutation cleanliness gate, the exact mutation, the revert, and a byte-clean proof
result: pass
source: automated
coverage_id: 07-02/D6

### 11. scripts/inject-cosign-key.sh exists, is executable, lints clean, and injects exactly one --key= line
expected: scripts/inject-cosign-key.sh exists, is tracked at mode 100755, lints clean under shellcheck, and running it against the real committed .goreleaser.yaml reports exactly one injected --key= line and exits 0
result: pass
source: automated
coverage_id: 07-03/D1

### 12. Script refuses bad arity and an unreadable first argument with exit 2
expected: The script refuses bad arity (2 args, exit 2 with a usage message) and an unreadable first argument (exit 2, naming the path)
result: pass
source: automated
coverage_id: 07-03/D2

### 13. Both release Task targets call the script; no inline injection block remains
expected: Both release:dry-run-signed and release:rehearse-notarize call the one script with three arguments each, and neither retains an inline copy of the extracted injection/diff-guard logic; the Taskfile still parses and no shape test was added
result: pass
source: automated
coverage_id: 07-03/D3

### 14. A config whose sign-blob anchor no longer matches is refused (count 0, non-zero exit)
expected: A config whose sign-blob anchor no longer matches is refused: the script reports a count of zero and exits non-zero against a re-indented copy, while the committed .goreleaser.yaml is proven byte-unchanged both before and after the demonstration
result: pass
source: automated
coverage_id: 07-03/D4

### 15. T-02-08 todo resolved and moved to todos/completed/
expected: T-02-08 pending todo is resolved and moved to .planning/todos/completed/
result: pass
source: automated
coverage_id: 07-03/D5

### 16. Test parses post-release-verify.yml jobs and fails on any non-verbatim conclusion guard
expected: A test parses post-release-verify.yml's job map, reports how many jobs it inspected (must be > 0), and fails when any single job's if: is not exactly the event-aware conclusion disjunct
result: pass
source: automated
coverage_id: 07-04/D1

### 17. Guard assertion discriminates on content (removed and inverted RED demonstrations)
expected: The assertion compares each job's parsed if: value against one verbatim expected string with no normaliser and no fixed expected-job-id list, proven by two RED demonstrations that discriminate on content: the guard removed from gatekeeper (fails naming gatekeeper with an empty value) and the guard inverted on resolve-tag (fails naming resolve-tag, quoting both the found and wanted values)
result: pass
source: automated
coverage_id: 07-04/D2

### 18. Empty-document companion test exists so a zero-job parse can never pass
expected: An empty-or-unparseable-document-is-error companion exists, mirroring TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError, so a zero-job parse can never read as a pass
result: pass
source: automated
coverage_id: 07-04/D3

### 19. Tautological tap App secret-distinctness test deleted; shared constant survives
expected: The tautological tap App secret-distinctness test is deleted (not rewritten, skipped, or stubbed), and homebrewTapCredentialNames survives because other tests in the file consume it
result: pass
source: automated
coverage_id: 07-04/D4

### 20. Mutation log is complete: four families, GRD-05 record, closing non-vacuity assertion
expected: 07-MUTATION-LOG.md carries exactly four demonstration families, each with pasted verbatim failing output and a byte-clean revert, plus the GRD-05 one-line deletion record and a closing section asserting the non-vacuity property — the log is complete
result: pass
source: automated
coverage_id: 07-04/D5

### 21. All four folded todos resolved under todos/completed/
expected: All four todos folded into this phase are resolved under .planning/todos/completed/, and .planning/todos/pending/ retains only the brew-trust todo Phase 12 owns (note: a second pending todo, 2026-09-08 graphstore-archtest-ignores-per-package-load-errors, was filed by code review after plan close and is a known, legitimate drift — not a Phase 7 gap)
result: pass
source: automated
coverage_id: 07-04/D6

### 22. Confirm automated coverage of all Phase 7 deliverables
expected: Every Phase 7 deliverable (21 across four plans) is covered by a passing automated check, and every covering check was re-executed at HEAD during this UAT session (go test on internal/bench, internal/query/archtest, internal/upgrade all ok; shellcheck + live run of scripts/inject-cosign-key.sh reports exactly one injected --key= line; Taskfile calls the script twice; 07-MUTATION-LOG.md has 4 families with 8 pasted FAIL lines; deleted tautological test has 0 references; all four folded todos are under todos/completed/). Nothing in this phase is user-facing UI, so there is no manual behaviour to exercise beyond confirming this evidence.
result: pass

## Summary

total: 22
passed: 22
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
