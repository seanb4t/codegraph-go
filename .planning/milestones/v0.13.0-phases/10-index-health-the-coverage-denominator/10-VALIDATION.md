---
phase: "10"
slug: "index-health-the-coverage-denominator"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-12"
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `10-RESEARCH.md` § Validation Architecture; every command and file path
> below was verified against `Taskfile.yml`, `web/package.json`, and the tree during research.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib (`task test:unit`, excludes `internal/daemon`; `task test:golden` for the frozen goldens); `vitest` via `pnpm -C web test` wrapped by `task web:test` (numTotalTests floor); committed Playwright `.mjs` scripts under `web/scripts/` for live-browser gates (`startCodegraphUi` + `pollUntil` conventions from `breadcrumb-check.mjs`) |
| **Config file** | `go.mod` (pins `go 1.26.6`); `web/vite.config.ts` (vitest); no Playwright config — scripts launch chromium directly |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test ./internal/indexer/... ./internal/graphstore/... ./internal/query/... ./internal/uiserver/... ./internal/schema/...` · `pnpm -C web test -- health` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 task test:unit` + `GOTOOLCHAIN=go1.26.6 task test:golden` + `task web:test` + `task proto:drift` + `task -s web:drift` |
| **Estimated runtime** | ~60 s targeted Go · ~15 s vitest · ~2–3 min full Go suite |

**Toolchain prefix (repo landmine).** Local Go is 1.27.1; `go.mod` pins 1.26.6 and `cockroachdb/swiss` breaks on newer toolchains. Prefix every local `go build` / `go test` / `go vet` with `GOTOOLCHAIN=go1.26.6`.

**Type-check gate (Phase 9 lesson, 09-06).** Any task touching `web/src` must re-run `pnpm -C web check` and gate on `0 ERRORS` case-insensitively with exit 0 — piped `svelte-check` prints its machine format in uppercase, so a literal `'0 errors'` grep never matches. Re-run it after every fix pass, not only the test suites.

**Proto regeneration.** `task proto:gen` regenerates BOTH the schema proto (`internal/schema/graph.proto`) and the UI proto (`internal/uiproto/uiv1/ui.proto`) in one invocation; `task proto:drift` proves the committed generated code matches. Schema and wire edits ship with their regenerated output and the `readonly_test.go` fixture update in the SAME commit (the 09-01 recipe).

---

## Sampling Rate

- **After every task commit:** the quick Go command scoped to the touched packages, plus `pnpm -C web test -- health` and `pnpm -C web check` for any web-touching task
- **After every plan wave:** `GOTOOLCHAIN=go1.26.6 task test:unit` + `task test:golden` + `task web:test` + `task proto:drift`
- **Before `/gsd-verify-work`:** full suite green, `GOTOOLCHAIN=go1.26.6 go vet ./...`, `task -s web:drift` MATCH, and the fixture-repo demonstration (criterion 1) run with its counts captured
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map binds requirements to their automated commands so
the planner can attach them. Test names are illustrative until the planner fixes them.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-02/T1 | 02 | 2 | HLT-05 | T-10-01 (reason integrity) | All four exclusion reasons are recorded at `Discover`'s decision points (dir-skip, extension miss, build-tag miss, stat-based size pre-check), never inferred later | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestDiscoverAll_RecordsBuildTagExclusion|TestDirExclusionReasonAgreesWithShouldSkipDir|TestDiscoverAll_ExactSizeLimitFileIsNotExcluded' ./internal/indexer/...` | ✅ | ✅ green |
| 10-02/T2 | 02 | 2 | HLT-05 | T-10-07 (reason integrity) | Reasons survive a process restart with NO re-walk: index the fixture, mutate disk without re-indexing, read-back still reports `BUILD_TAG` / `UNSUPPORTED_EXTENSION` (D-14a) | integration | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestCoverageReasonsSurviveDiskMutationWithoutReindex' ./internal/indexer/...` | ✅ | ✅ green |
| 10-01/T1 | 01 | 1 | HLT-05 | T-10-07 (archtest) | `internal/query`'s coverage reader never walks disk — file-scoped positive-controlled source scan (D-14b) + the existing GRD-02 archtest still green | unit (source scan) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestCoverageSourceNeverWalksDisk' ./internal/query/...` && `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/...` | ✅ | ✅ green |
| 10-01/T1 | 01 | 1 | HLT-05 | T-10-15 | Additive within `SchemaVersion 1`: a graph with `Meta.has_coverage` unset opens and reports coverage `known == false` with no counts — never `0/0`, never an error (D-06/D-15) | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestCoverageSummaryOnOldGraphIsUnknown|TestKnownMetaFieldNumbersAreStable' ./internal/query/... ./internal/schema/...` | ✅ | ✅ green |
| 10-01/T1 | 01 | 1 | HLT-06 | T-10-09 (read-only guard) | `GetCoverage` clears every `mutatingVerbs` substring; decoy `GetIndexCoverage` is REJECTED (positive control); `wantUIServiceMethods` 15→16 set-equal both directions, count from both sides (D-16) | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestUIServiceMethodSetIsExactlyTheReadSet|TestCoverageRPCNameClearsMutatingVerbsWhileDecoyIsRejected' ./internal/uiserver/...` | ✅ (extended `readonly_test.go`) | ✅ green |
| 10-01/T1 | 01 | 1 | HLT-06 | — | `GetHealthResponse.coverage = 17` and `GetCoverage` round-trip through the real listener; `task proto:drift` MATCH | integration | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestGetCoverage|TestGetHealth' ./internal/uiserver/...` && `task proto:drift` | ✅ | ✅ green |
| 10-02/T2 | 02 | 2 | HLT-04 | — | Against the D-13 fixture: exact discovered / indexed / excluded / extraction-failed counts and the per-file reasons, extraction failures distinguished from exclusions — counts reported, not implied | integration | `GOTOOLCHAIN=go1.26.6 go test -count=1 -run 'TestCoverageFixtureCountsAndReasons' ./internal/indexer/...` | ✅ | ✅ green |
| 10-04/T2 | 04 | 2 | HLT-04 | T-10-01, T-10-06, T-10-08, T-10-15 | `/health` renders the Coverage section: counts line, reason groups with counts expanding to rows, "extraction failed: <error>" rows distinct, "Coverage unknown — re-index to record it" for an old graph, remedy text only (no action control) | unit (vitest) | `pnpm -C web test -- health` | ✅ (extended `web/tests/health-page.test.ts` / `health-view.test.ts`) | ✅ green |
| n/a | 04 | 2 | HLT-04 | — | Same, observed in a live Chromium against the real fixture index — Open Question 3 decided vitest-level (10-04-SUMMARY.md key-decisions); no Playwright gate added | live-browser | `node web/scripts/coverage-check.mjs` | ❌ not built (deliberate) | ⬜ n/a — Open Question 3: vitest-level (10-04) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/indexer/testdata/coverage/` — the D-13 fixture module: `go.mod`, `vendor/x.go`, `.hidden/y.go`, `notes.md`, `tagged.go` (`//go:build ignore`), one extraction-failing file; the oversize `.go` is generated at test time (> `parser.MaxSourceBytes`), never committed
- [x] `internal/indexer/discoverexclusion_test.go` / `internal/indexer/coverage_fixture_test.go` — the four decision points + the fixture counts test + the D-14a mutate-without-reindex test
- [x] `internal/query/coverage_test.go` — Engine coverage read-back, `has_coverage` unknown-state test, and the file-scoped D-14b source scan with a positive control
- [x] `internal/schema` — `Meta.has_coverage = 9` and `ExcludedFile` / `ExclusionReason` round-trip + field-number stability entries (mirror `TestKnownMetaFieldNumbersAreStable`)
- [x] `internal/uiserver/readonly_test.go` — method set 15→16 both directions, decoy-rejection table test; `uiProtoFieldFixture` extended for `GetHealthResponse.coverage = 17` and the new messages
- [x] `internal/uiserver/coverage_test.go` — `GetCoverage` paging + `GetHealth` coverage summary through the real listener
- [x] `web/tests/health-page.test.ts` / `health-view.test.ts` — extended for the Coverage section and the unknown state
- [x] `10-MUTATION-LOG.md` — families: unset-`has_coverage` read as zero (D-15), rpc renamed to the `GetIndexCoverage` decoy (D-16), reason reconstructed by a walk (D-14) — each RED, byte-cleanly reverted
- [x] Framework install: none — Go stdlib testing, vitest, and Playwright are all present and already used for these shapes
- [ ] Optional live-Chromium coverage gate (`web/scripts/coverage-check.mjs`) — not built; Open Question 3 decided vitest-level only (10-04-SUMMARY.md key-decisions)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| — | — | — | — |

*All phase behaviors have automated verification; the optional live-Chromium gate (Open Question 3) is automated if adopted.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
