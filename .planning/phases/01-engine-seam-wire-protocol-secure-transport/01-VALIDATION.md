---
phase: 1
slug: engine-seam-wire-protocol-secure-transport
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-22
validated: 2026-09-07
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded by plan-phase from `01-RESEARCH.md` §Validation Architecture.
> **Reconciled against the executed phase 2026-09-07** — every `TBD` below is now a real
> plan id, a real test name, and a counted `--- PASS` result.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` (`go test`) — no third-party test framework anywhere in this repo |
| **Config file** | none — `Taskfile.yml` defines the invocation surface (`task test:unit`, `task test:golden`, `task test:race`) |
| **Quick run command** | `GOTOOLCHAIN=go1.26.5 go test -count=1 -v -run <TestName> ./internal/uiserver/` |
| **Full suite command** | `task test:unit && task test:golden` |
| **Measured runtime** | **`task test:unit` 10s · `task test:golden` 27s · ~37s combined** (measured 2026-09-07) |

> **Golden-suite tooling quirk (do not "fix"):** the golden suite is excluded from
> `go list ./...`, per `testdata/golden/README.md`'s own documented note. It must be
> invoked explicitly (`task test:golden` / `go test ./testdata/golden/...`); a
> `./...` run does NOT cover it. A validation step that only runs `./...` and reports
> green has not exercised a single golden.

> **Environment (measured, not assumed):** every local Go command needs
> `GOTOOLCHAIN=go1.26.5` — go1.27 breaks the `cockroachdb/swiss` build. `go.mod` says
> 1.26.5 but `GOTOOLCHAIN=auto` resolves *up*. CI is unaffected (`go-version-file` pins
> 1.26). Also: bare `go test ./...` fails on a pre-existing `internal/daemon` watchdog
> flake under parallel load, which is why `task test:unit` excludes it deliberately
> (`Taskfile.yml:131`).

---

## Sampling Rate

- **After every task commit:** the specific new/touched test file(s) from the map below
- **After every plan wave:** `task test:unit && task test:golden`
- **Before `/gsd-verify-work`:** full suite green, **plus** the D-04 mutation-proof
  procedure run once with its PASS/FAIL counts recorded in the phase's evidence artifact
- **Max feedback latency:** **~37s** for the full suite; **<1s** for a single-package
  targeted run (measured 2026-09-07)

---

## Per-Task Verification Map

Re-keyed to real plan ids and real test names 2026-09-07. Every command below was executed
fresh with `-count=1` and scored by counting `--- PASS` lines.

| Requirement | Plan | Wave | Threat Ref | Secure Behavior | Test Type | Automated Command | PASS | Status |
|-------------|------|------|------------|-----------------|-----------|-------------------|------|--------|
| ENG-01 | 01-02, 01-04, 01-05 | 1–3 | T-01-12, T-01-16 | N/A | golden byte-identity | `go test -count=1 -v -run 'TestGoldensMatchLiveEngineOutput$' ./testdata/golden/` | **27** | ✅ green |
| ENG-02 | 01-02, 01-05 | 1, 3 | T-01-11 | N/A | golden byte-identity | (same command — one oracle covers both) | **27** | ✅ green |
| — | — | — | T-01-12 | the oracle cannot pass vacuously | mutation proof | `go test -count=1 -v -run 'TestGoldensMatchLiveEngineOutputIsNonVacuous$' ./testdata/golden/` | **3** | ✅ green |
| ENG-04 | 01-06, 01-08 | 2, 4 | T-01-30 | `Meta` field 8 round-trips; absent degrades | unit | `go test -count=1 -v -run 'TestMetaCommitSHA\|TestKnownMetaFieldNumbersAreStable\|TestIndexMeta' ./internal/schema/ ./internal/query/` | **5** | ✅ green |
| RPC-01 | 01-01, 01-08, 01-09 | 1, 4, 5 | T-01-04 | read-only method set | integration | `go test -count=1 -v -run 'TestUIService' ./internal/uiserver/` | **56** | ✅ green |
| RPC-02 | 01-01, 01-08, 01-09 | 1, 4, 5 | T-01-23 | errors classified by sentinel, never text | integration | (same command) | **56** | ✅ green |
| RPC-05 | 01-10 | 6 | T-01-05, T-01-37 | bounded response, `truncated` set, rune-safe cut | unit | `go test -count=1 -v -run 'TestTruncateSource\|TestTransportBackstop' ./internal/uiserver/` | **15** | ✅ green |
| SRV-01 | 01-01 | 1 | T-01-03 | loopback bind, ephemeral port | integration | `go test -count=1 -v -run 'TestListen\|TestUIServer\|TestUICommand' ./internal/uiserver/ ./internal/cli/` | **11** | ✅ green |
| SRV-02 | 01-01 | 1 | T-01-01, T-01-02 | foreign `Host`/`Origin` rejected **before any handler runs**; `localhost`, `127.0.0.1`, `[::1]` each admitted by EXACT match | unit — **demonstrated RED first** | `go test -count=1 -v -run 'TestOriginHostGuard' ./internal/uiserver/` | **21** | ✅ green |
| SRV-03 | 01-01, 01-09 | 1, 5 | T-01-04 | no v1 flag exposes bind address or auth; no RPC mutates the index | unit | `go test -count=1 -v -run 'TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIServiceDeclaresNoMutatingMethod\|TestUICommandFlagSetIsExactlyPathAndNoOpen' ./internal/uiserver/ ./internal/cli/` | **3** | ✅ green |
| SRV-04 | 01-11 | 7 | T-01-10, T-01-39 | locked store renders "indexing in progress", never an error | integration | `go test -count=1 -v -run 'TestDegrade' ./internal/uiserver/` | **2** | ✅ green |
| BLD-04 | 01-07 | 3 | T-01-08, T-01-32, T-01-33 | generated code matches its `.proto`, header included | drift task | `task proto:drift` | exit 0, 4 files byte-identical | ✅ green |
| FIX-01 | 01-03 | 1 | T-01-09, T-01-27 | in-flight response not lost when a notification is written concurrently | unit — **demonstrated RED first** | `go test -count=1 -v -run 'TestPendingWriter' ./internal/mcp/` | **18** | ✅ green |

**Totals: 12/12 requirements automated · 161 `--- PASS` · 0 `--- FAIL`.**

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Scoring rule (rule `84d1gfpywd`, and `5pzpmvthcc`'s dominant v0.11.0 finding):**
> `go test -run PATTERN` **exits 0 when the pattern matches nothing.** Three
> plan-declared commands in v0.11.0 matched zero tests and read green, one of them
> the named guard for a CRITICAL threat. Score every command in this table by
> counting `--- PASS` lines, **never** by exit status.
>
> **Positive control run 2026-09-07:** `go test -count=1 -v -run 'TestThisDoesNotExistAnywhere'
> ./internal/uiserver/` returned **PASS=0** while exiting 0 — confirming the counting
> method discriminates and that the 161 above are real, not an artifact of a dead pattern.

---

## Wave 0 Requirements — all complete

- [x] **A byte-identity comparison test over the golden `(corpus, filename)` pairs.**
      Delivered as `TestGoldensMatchLiveEngineOutput`
      (`testdata/golden/byte_identity_test.go:268`), which re-invokes the live Engine and
      compares bytes — the thing `TestReFrozenGoldensValid` never did. Its non-vacuity
      proof, `TestGoldensMatchLiveEngineOutputIsNonVacuous` (`:404`), plants two mutations
      (appended byte, flipped interior byte) and requires both to be caught. Verified live
      2026-09-07: 26/26 goldens matched, both planted mutations caught.
- [x] `internal/uiserver` package and its test files — exists, 56 `TestUIService*` PASS
- [x] A concurrency reproduction isolating the `toolslist-repeat` flake from notification
      traffic — `TestPendingWriter*` (7 declarations, 18 PASS including subtests), covering
      `NeverGoesNegative`, `DecrementsOnlyResponses`, `SerializesWriteAndClassification`
- [x] `internal/schema` test asserting `Meta` field 8 round-trips and degrades gracefully
      when absent — `TestMetaCommitSHARoundTrips` / `TestMetaCommitSHAAbsentDegrades`
      (`internal/schema/meta_commit_test.go:12,45`)
- [x] A FIX-01 reproduction that **demonstrably shows the response loss before the fix** —
      `TestPendingWriterDrainsEarlyWithoutFix` (`internal/mcp/pending_writer_test.go:44`),
      named for exactly that property

*Full-suite runtime measured 2026-09-07 and recorded above; no TBD values remain.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| A real browser process actually launches on `codegraph ui` | SRV-01 / D-09 | Launching a real browser cannot be asserted in CI. **The automatable half is now automated** — `TestShouldOpenBrowserSuppression` (`internal/cli/ui_test.go:43`) covers all four decision branches (`--no-open` suppresses, `CI` set suppresses, non-character-device stdout suppresses, none-of-those opens), 4/4 PASS. Only the positive launch remains manual. | On a TTY with `CI` unset, run `codegraph ui`; confirm the printed loopback URL opens in the default browser. |

**Row closed since seeding:** *"The SPA's own same-origin requests are admitted by the SRV-02
guard"* was deferred to Phase 2 because Phase 1 shipped no SPA. Phase 2 delivered it —
`TestSPAInheritsOriginHostGuard` (`internal/uiserver/spa_test.go:666`), PASS. The
`Origin`-absent-on-GET case Phase 1 was told it "MUST still automate" is likewise automated as
`TestOriginHostGuardAdmitsOriginlessGET` (`originguard_test.go:189`), PASS. No longer
manual-only.

---

## Validation Audit 2026-09-07

| Metric | Count |
|--------|-------|
| Requirements in scope | 12 |
| Gaps found (MISSING) | **0** |
| Gaps found (PARTIAL) | 0 |
| Already COVERED | 12 |
| Resolved this audit | 0 (none needed) |
| Escalated to manual-only | 0 |
| Manual-only rows closed by later automation | 1 |

**Finding:** this file was a plan-time seed that execution never reconciled — every row read
`TBD` / `❌ W0` / `⬜ pending` because it was written before the phase ran, not because
anything was untested. All 12 requirements already had automated, passing verification; the
audit's work was reconciliation, not gap-filling. No tests were generated.

**Method:** each requirement's command was executed fresh with `-count=1` and scored by
counting `--- PASS` lines rather than exit status, with a deliberate negative pattern run as a
positive control to prove the counting discriminates.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — **12/12 automated, so continuity holds trivially**
- [x] Wave 0 covers all MISSING references — all five Wave 0 items delivered and verified
- [x] No watch-mode flags
- [x] Feedback latency measured and recorded (10s unit · 27s golden · ~37s combined)
- [x] Every command in the verification map scored by `--- PASS` count, not exit status, with a positive control
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-07
