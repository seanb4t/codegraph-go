---
phase: 1
slug: engine-seam-wire-protocol-secure-transport
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-22
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded by plan-phase from `01-RESEARCH.md` §Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` (`go test`) — no third-party test framework anywhere in this repo |
| **Config file** | none — `Taskfile.yml` defines the invocation surface (`task test:unit`, `task test:golden`, `task test:race`) |
| **Quick run command** | `go test ./internal/query/... ./internal/mcp/... -run <TestName> -v` |
| **Full suite command** | `task test:unit && task test:golden` |
| **Estimated runtime** | ~TBD — measure during Wave 0 and replace this value |

> **Golden-suite tooling quirk (do not "fix"):** the golden suite is excluded from
> `go list ./...`, per `testdata/golden/README.md`'s own documented note. It must be
> invoked explicitly (`task test:golden` / `go test ./testdata/golden/...`); a
> `./...` run does NOT cover it. A validation step that only runs `./...` and reports
> green has not exercised a single golden.

---

## Sampling Rate

- **After every task commit:** the specific new/touched test file(s) from the map below
- **After every plan wave:** `task test:unit && task test:golden`
- **Before `/gsd-verify-work`:** full suite green, **plus** the D-04 mutation-proof
  procedure run once with its PASS/FAIL counts recorded in the phase's evidence artifact
- **Max feedback latency:** TBD — measure in Wave 0

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this table is seeded per requirement and must be
re-keyed to real task IDs once PLAN.md files exist.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 0 | ENG-01 | — | N/A | golden byte-identity (NEW) | `go test -v -run TestNodeGoldensMatchEngine ./testdata/golden/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | 0 | ENG-02 | — | N/A | golden byte-identity (NEW) | `go test -v -run TestExploreGoldensMatchEngine ./testdata/golden/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | ENG-04 | — | N/A | unit | `go test ./internal/schema/... -run TestMeta -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RPC-01 | — | N/A | integration | `go test ./internal/uiserver/... -run TestUIService -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RPC-02 | — | N/A | integration | `go test ./internal/uiserver/... -run TestUIService -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | RPC-05 | TBD | bounded response, `truncated` set, rune-safe cut | unit | `go test ./internal/uiserver/... -run TestTruncate -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SRV-01 | — | N/A | integration | `go test ./internal/uiserver/... -run TestUIServe -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SRV-02 | TBD | foreign `Host`/`Origin` rejected **before any handler runs**; `localhost`, `127.0.0.1`, `[::1]` each admitted by EXACT match | unit — **demonstrated RED first** | `go test ./internal/uiserver/... -run TestOriginHostGuard -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SRV-03 | TBD | no v1 flag exposes bind address or auth; no RPC mutates the index | unit | `go test ./internal/uiserver/... -run TestReadOnlyAndNoBindFlag -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | SRV-04 | — | locked store renders "indexing in progress", never an error | integration | `go test ./internal/uiserver/... -run TestStatusDegradesOnLock -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | BLD-04 | — | N/A | shape + drift task | `task proto:drift` (new) + `go test ./internal/upgrade/... -run TestProtoTaskExists -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-01 | — | in-flight response not lost when a notification is written concurrently | unit — **demonstrated RED first** | `go test ./internal/mcp/... -run TestPendingWriter -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Scoring rule (rule `84d1gfpywd`, and `5pzpmvthcc`'s dominant v0.11.0 finding):**
> `go test -run PATTERN` **exits 0 when the pattern matches nothing.** Three
> plan-declared commands in v0.11.0 matched zero tests and read green, one of them
> the named guard for a CRITICAL threat. Score every command in this table by
> counting `--- PASS` lines, **never** by exit status.

---

## Wave 0 Requirements

- [ ] **A byte-identity comparison test over the 26 golden `(corpus, filename)` pairs.**
      **This does not exist today** — `TestReFrozenGoldensValid`
      (`testdata/golden/golden_test.go:243-297`) checks only file existence,
      non-emptiness, a leading `{`, `goldenCapture` parseability, and a non-empty
      `Output` field; it never re-invokes `Engine.Node`/`Explore`. And
      `TestCorpusBehavior_Go` (`testdata/golden/behavioral_test.go:685-691`) states in
      its own doc comment that it asserts named behavioral properties of live output,
      "**not byte-diffs against a frozen golden**" — the capture path was retired in
      FIXT-04. **Highest-priority Wave 0 item:** ENG-01/ENG-02's success criterion
      cannot be proven without it, and D-04's mutation-proof has nothing to turn RED.
- [ ] `internal/uiserver` package and its test files — entirely new
- [ ] A concurrency reproduction isolating the `toolslist-repeat` flake **from any
      notification traffic**, to keep it separable from FIX-01 (see the falsified
      hypothesis recorded in `01-CONTEXT.md`)
- [ ] `internal/schema` test asserting `Meta` field 8 round-trips and degrades
      gracefully when absent (the `has_file_index = 7` precedent)
- [ ] A FIX-01 reproduction that **demonstrably shows the response loss before the
      fix** — success criterion 5 requires the reproduction, not just the fix

*Measure and record the full-suite runtime during Wave 0; the two TBD latency values
above are not to be left as TBD at sign-off.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Browser auto-open launches the default browser | SRV-01 / D-09 | Launching a real browser is not assertable in CI; the TTY/CI suppression path IS automatable and must be, so only the positive launch is manual | On a TTY with `CI` unset, run `codegraph ui`; confirm the printed loopback URL opens in the default browser. Then confirm `--no-open` and a piped (non-TTY) invocation each print the URL without launching. |
| The SPA's own same-origin requests are admitted by the SRV-02 guard | SRV-02 | Phase 1 ships no SPA (that is Phase 2), so the real browser-origin shape cannot be exercised end-to-end yet | Deferred to Phase 2 integration. Phase 1 MUST still automate the `Origin`-absent-on-GET case flagged in research, since over-rejection is the predicted failure mode. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency measured and recorded (currently TBD)
- [ ] Every command in the verification map scored by `--- PASS` count, not exit status
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
