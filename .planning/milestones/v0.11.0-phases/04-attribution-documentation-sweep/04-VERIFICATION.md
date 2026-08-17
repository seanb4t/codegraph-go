---
phase: 04-attribution-documentation-sweep
verified: 2026-08-15T00:00:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps: []
deferred: []
behavior_unverified_items: []
human_verification: []
---

# Phase 4: Attribution & Documentation Sweep Verification Report

**Phase Goal:** A reader of this project's own documentation finds the origin acknowledged exactly once — legally, and in the past tense — and encounters no comparison framing anywhere else.
**Verified:** 2026-08-15
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | ATTR-01: NOTICE contains MIT copyright transcription + one origin sentence, no drop-in/ported/flag-parity/benchmark argument | ✓ VERIFIED | `rg -c 'Copyright (c) 2026 Colby Mchenry' NOTICE` = 1 (verbatim transcription); `begin as a ground-up Go rewrite` = 1 (origin sentence); `Third-party dependencies` = 1; `Do not move any of this` = 1; `drop-in\|\bported\b\|FLAG-PARITY\|BENCHMARKS` = 0. One-sentence origin clause present. |
| 2 | ATTR-02: README has no `## Relationship to the original`; only origin mention is one past-tense clause in `## License` linking to NOTICE | ✓ VERIFIED | `rg -c 'Relationship to the original' README.md` = 0; `rg -c 'began as a Go rewrite of CodeGraph'` = 1 (exact License clause at line 208); `the TS original\|it ports\|flattering\|parity\|divergence` = 0; `'original'` appears only in the License clause (line 208); Status claim rewritten on own terms; Performance label now "(TypeScript)". |
| 3 | ATTR-03: LICENSE is verbatim MIT and `gh api repos/seanb4t/codegraph-go/license` returns MIT | ✓ VERIFIED | `git diff --exit-code HEAD -- LICENSE` clean (byte-identical); MIT markers present; live `gh api .../license --jq '.license.spdx_id'` = `MIT` (License name "MIT License"). |
| 4 | DOCS-01/03/04: no comparison framing in README/CONTRIBUTING/SECURITY/CODE_OF_CONDUCT/PARSER-DECISION/RELEASE/RELEASE-PROCEDURES/MCP-SCOPING/MCP-AUDIT; capability matrix on its own terms | ✓ VERIFIED | Word-boundary framing sweep per file (parity/upstream/drop-in/head-to-head/the original/vs TS/based on/ported) = 0 except recorded keeps: README:208 (mandated License clause), RELEASE-PROCEDURES:60 (`gsd/v1.0-drop-in-parity-human-ux` historical branch), RELEASE-PROCEDURES:581 (pipeline "upstream"), matrix 181/186 (`original fwcd recommendation` historical research ref, kept with reason per D-04). Capability matrix describes codegraph-go's own per-language tiering with no reference to another implementation's coverage. |
| 5 | DOCS-02: `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go` deleted, `go test ./...` passes, nothing references either | ✓ VERIFIED | Both files absent (`test -f` -> NO); full-tree reference sweep `rg -in 'FLAG-PARITY|flag_parity|flag-parity|flagParity|TestFlagParityDocCoversRegisteredFlags' --glob '!.planning/**' --glob '!graphify-out/**' --glob '!.claude/worktrees/**'` = 0; `go build ./...` exit 0; `CGO_ENABLED=1 go test -count=1 ./...` exit 0 (all packages ok, daemon included). |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ---------| ------ | ------- |
| `NOTICE` | MIT transcription + one origin sentence; no comparison argument | ✓ VERIFIED | Arg removed; LICENSE warning block + Third-party deps kept (D-01 legal pair) |
| `README.md` | No Relationship section; one License clause; Performance/Status framing swept | ✓ VERIFIED | `## License` line 208 exact clause; Status on own terms; Performance label "(TypeScript)" |
| `LICENSE` | Verbatim MIT, byte-identical to HEAD | ✓ VERIFIED | `git diff --exit-code HEAD -- LICENSE` clean; live SPDX `MIT` |
| `docs/FLAG-PARITY.md` | Deleted | ✓ VERIFIED | File absent |
| `internal/cli/flag_parity_test.go` | Deleted | ✓ VERIFIED | File absent |
| `CONTRIBUTING.md` | Comparison framing swept | ✓ VERIFIED | issue-first paragraph reworded on own terms, no FLAG-PARITY link |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md` | Capability on codegraph-go's own terms | ✓ VERIFIED | golden-parity wording replaced with behavioral; matrix:181 kept with reason |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| README `## License` | `NOTICE` | link `[NOTICE](NOTICE)` in the one origin clause | ✓ WIRED | Line 208 exact clause links to NOTICE |
| README `## License` | `LICENSE` | link `[LICENSE](LICENSE)` | ✓ WIRED | Line 208 |
| `.github` templates | `.planning/` | re-pointed links (FLAG-PARITY removed) | ✓ WIRED | `rg -in 'FLAG-PARITY' .github/` = 0; re-pointed to `.planning/` in pull_request_template.md, auto-close-unsolicited-prs.yml, enhancement.yml, feature.md |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Repo builds after deletion | `go build ./...` | exit 0 | ✓ PASS |
| Test suite still passes | `CGO_ENABLED=1 go test -count=1 ./...` | exit 0, all packages ok | ✓ PASS |
| Daemon package (documented flake class) | `go test -count=1 ./internal/daemon/` | exit 0 (64.79s) | ✓ PASS |
| Live license detection | `gh api repos/seanb4t/codegraph-go/license --jq '.license.spdx_id'` | `MIT` | ✓ PASS |
| LICENSE byte-identity | `git diff --exit-code HEAD -- LICENSE` | clean | ✓ PASS |

### Requirements Coverage

| Requirement | Description | Status | Evidence |
| ----------- | ----------- | ------ | -------- |
| ATTR-01 | NOTICE trimmed to MIT transcription + one origin sentence, argument removed | ✓ SATISFIED | NOTICE file content verified |
| ATTR-02 | README Relationship section gone; one origin clause in License | ✓ SATISFIED | README file content verified |
| ATTR-03 | LICENSE verbatim MIT; live detection MIT | ✓ SATISFIED | git byte-identical + gh api MIT |
| DOCS-01 | No comparison framing in README/CONTRIBUTING/SECURITY/CoC/PARSER-DECISION | ✓ SATISFIED | framing sweep clean |
| DOCS-02 | FLAG-PARITY doc + guard deleted, no references | ✓ SATISFIED | files absent, 0 references, go test green |
| DOCS-03 | Capability matrix on own terms | ✓ SATISFIED | golden-parity wording replaced; matrix:181 recorded keep |
| DOCS-04 | Remaining docs/* no retired framing | ✓ SATISFIED | only recorded keeps (branch name, pipeline upstream) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | none (no TBD/FIXME/XXX/PLACEHOLDER in any touched file) | — | — |

### Product Truth Intact (D-05)

- `tsextract` package present and its tests ran ok (`internal/indexer/tsextract`, 2.563s).
- `codegraph migrate` present and tested (`internal/migrate`, 25.233s ok).
- TypeScript-as-indexed-language intact: README Full `Go, Java, C#, TypeScript, TSX` row present; migrate row `import an existing TypeScript CodeGraph index` present.
- `began as a Go rewrite` clause present in NOTICE (ground-up) and README License.

### D-08 Boundary Untouched

`git diff --exit-code HEAD` clean for `tools/bench/realcorpus`, `bench.yml`, `tools/bench/headtohead-*.json`, `docs/BENCHMARKS.md`, `ts-schema*`, `ts-version.txt`. The README Performance numbers table is intentionally retained for Phase 6 BENCH (D-05), matching the phase scope.

### Human Verification Required

None. All criteria verified with direct file, live API, and behavioral (build/test) evidence.

### Gaps Summary

No gaps. All 5 success criteria and all 7 requirements (ATTR-01/02/03, DOCS-01/02/03/04) verified. Working tree clean; phase commits present.

---

_Verified: 2026-08-15_
_Verifier: Claude (gsd-verifier)_