---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 02
subsystem: cli
tags: [editor-discovery, exec.LookPath, os.Stat, uri-scheme-allowlist, startup-config]

# Dependency graph
requires:
  - phase: 09-source-view-follow-through-breadcrumb-editor-handoff
    provides: "plan 09-01's editorResolveInputs.discover seam, uiserver.EditorPresets(), and uiserver.ValidateEditorTemplate that this plan's discovered templates must pass"
provides:
  - "internal/cli/editordiscovery.go: discoverEditor() — startup-time, probe-only editor detection (exec.LookPath then platform app directories), in the committed order code, cursor, idea, goland, webstorm, pycharm, rider, clion, phpstorm, rubymine"
  - "codegraph ui with no --editor-url flag and no CODEGRAPH_EDITOR_URL env now resolves a real, installed editor at startup instead of always falling through to unconfigured"
affects: [09-04-header-link-and-gutter, 09-05-security-doc]

# Actuals (#2632)
actuals:
  tokens: 6606
  tasks: 2
  commits: 3
plan_head_before: 599450ce7d83b01a4f17ec90b00b87ee7ebf5770

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "editorProbes{lookPath, stat, goos, getenv} — full dependency injection of every OS/filesystem touch, mirroring the discovery function's need to be exercised against fakes rather than the real host"
    - "PATH probe before app-directory probe, per launcher, with an immediate return on the first hit — search order is popularity order (editorLaunchers), not alphabetical or platform-grouped"
    - "discoverEditorFn (internal/cli/ui.go) is the same package-level func-var test seam openBrowser already established"

key-files:
  created:
    - internal/cli/editordiscovery.go
    - internal/cli/editordiscovery_test.go
  modified:
    - internal/cli/ui.go
    - internal/cli/ui_test.go

key-decisions:
  - "gofmt's struct-literal column alignment padded 'discover:' with multiple spaces (aligned against the longer 'flagTemplate'/'noEditorURL' keys in the same composite literal), which defeated the plan's own single-space literal verify grep (rg -o 'discover: discoverEditorFn'). Fixed by inserting an explanatory comment immediately above the discover field — this breaks gofmt's alignment group for that field alone, producing a single space, satisfying both gofmt and the literal-text gate without touching the check itself (same class of fix 09-01-SUMMARY.md recorded twice for editorpresets.go/ui.go doc comments)."
  - "templateForLauncher never restates the vscode/cursor template strings — it looks them up from uiserver.EditorPresets() by ID (presetTemplate helper), so the two presets have exactly one source of truth across both plans."
  - "TestUICommandDiscoveredDefaultHasProvenance's off-switch subtest records discoverEditorFn calls via atomic.LoadInt32 rather than calling t.Fatal inside the fake — the fake runs on cmd.ExecuteContext's own goroutine, and the testing package documents FailNow/Fatal as unsafe off the test's own goroutine."

patterns-established:
  - "appDirCandidates(launcher, goos, getenv) centralizes the platform-specific app-directory candidate list (macOS bundles under /Applications and $HOME/Applications; Linux JetBrains-only under /opt and the Toolbox apps dir) so discoverEditorWith's loop body never branches on goos itself."

requirements-completed: [BRW-12]

coverage:
  - id: D1
    description: "discoverEditorWith probes PATH before app directories, in the committed ten-launcher order, stopping at the first hit"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editordiscovery_test.go#TestDiscoverEditorOrderIsCommitted"
        status: pass
      - kind: unit
        ref: "internal/cli/editordiscovery_test.go#TestDiscoverEditorProbesPathBeforeAppDirs"
        status: pass
    human_judgment: false
  - id: D2
    description: "Discovery is probe-only (exec.LookPath/os.Stat only) — zero process-spawning calls in editordiscovery.go, a positive exec.LookPath count, and at least 10 launcher-name literals"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editordiscovery_test.go#TestDiscoverEditorNeverExecutes"
        status: pass
      - kind: other
        ref: "rg -o 'exec\\.Command\\(|exec\\.CommandContext\\(|os\\.StartProcess\\(' internal/cli/editordiscovery.go (0 matches); rg -o 'exec\\.LookPath' (2 matches); launcher-literal count (25 matches)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A discovered launcher's template is always allowlist-valid: code/cursor read from uiserver.EditorPresets(), every JetBrains launcher composes its own open?file= form"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editordiscovery_test.go#TestTemplateForLauncherEmitsAllowlistedTemplates"
        status: pass
    human_judgment: false
  - id: D4
    description: "Nothing installed is never fatal — discoverEditorWith reports (zero value, false) with no error, and the real codegraph ui command still starts and answers NO_TEMPLATE"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editordiscovery_test.go#TestDiscoverEditorNotFoundIsNotFatal"
        status: pass
      - kind: integration
        ref: "internal/cli/ui_test.go#TestUICommandDiscoveredDefaultHasProvenance/not_found_is_never_fatal"
        status: pass
    human_judgment: false
  - id: D5
    description: "codegraph ui rewired to the real discoverer: DISCOVERED provenance and the launcher name reach GetEditorLink's wire response through a real running command"
    requirement: "BRW-12"
    verification:
      - kind: integration
        ref: "internal/cli/ui_test.go#TestUICommandDiscoveredDefaultHasProvenance/discovered"
        status: pass
    human_judgment: false
  - id: D6
    description: "The --no-editor-url off switch skips discovery entirely — discoverEditorFn is never called"
    requirement: "BRW-12"
    verification:
      - kind: integration
        ref: "internal/cli/ui_test.go#TestUICommandDiscoveredDefaultHasProvenance/off_switch_skips_discovery_entirely"
        status: pass
    human_judgment: false
  - id: D7
    description: "On a machine with a real editor installed, the discovered default is usable end to end: the picker shows discovered provenance and a gutter line-number click lands the editor on that line"
    verification: []
    human_judgment: true
    rationale: "This is 09-02's own plan text: an end-of-phase, harvested human-check (09-RESEARCH.md A1/A2's Cursor/JetBrains URI-form confirmation), deliberately not a mid-flight checkpoint for this plan. No SourcePane UI exists yet in this phase's own scope (that lands in a later plan) to click a gutter number through; the fake-backed automated tests above already prove the wiring. Deferred to the phase's end-of-phase UAT (09-05) alongside 09-01's own still-[ASSUMED] Cursor/JetBrains templates."

duration: 14min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 2: Startup Editor Discovery Summary

`codegraph ui` now finds an installed editor on its own — probing PATH then platform application directories in a committed VS Code → Cursor → JetBrains order — and wires that discovery into the server's effective default with DISCOVERED provenance, with no flag or env var required.

## Performance

- **Duration:** ~14 min
- **Started:** 2026-09-12T17:45:40Z
- **Completed:** 2026-09-12T17:58:33Z
- **Tasks:** 2 (Task 1 `tdd="true"`, RED->GREEN; Task 2 `type="auto"`)
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `internal/cli/editordiscovery.go`: `editorLaunchers` (the committed ten-launcher probe order), `jetbrainsLaunchers`, `macAppBundles`, `linuxAppDirs`, `editorProbes`, `appDirCandidates`, `templateForLauncher`, `discoverEditorWith`, and `discoverEditor` — PATH via `exec.LookPath` first, then platform app directories, stopping at the first hit; probe-only by construction (zero process-spawning calls), never fatal when nothing is found
- `templateForLauncher` reads the vscode/cursor templates from `uiserver.EditorPresets()` (never restates them) and derives every JetBrains launcher's `<launcher>://open?file={path}&line={line}` form
- `internal/cli/ui.go`: `discoverEditorFn` (the `openBrowser` test-seam pattern) now replaces `discoverEditor` for `discover: nil` — `codegraph ui` with no flag and no env discovers an installed editor at startup
- `TestUICommandDiscoveredDefaultHasProvenance` proves all three outcomes through the real command: discovered (DISCOVERED provenance, `goland` launcher, `BUILDABLE` url), not-found (server still starts, `NO_TEMPLATE`/`NONE`), and the off switch (discovery never invoked)

## Task Commits

1. **Task 1: Startup editor discovery** (`tdd="true"`, RED -> GREEN)
   - `fe37e3ce` `test(09-02): add failing editor discovery tests` (RED)
   - `c78f0a71` `feat(09-02): discover an installed editor at startup, probe-only, in committed order` (GREEN)
2. **Task 2: Rewire `codegraph ui` to the real discoverer**
   - `cc48f44f` `feat(09-02): wire startup editor discovery into codegraph ui`

## TDD Gate Compliance

| Task | RED commit | GREEN commit | REFACTOR | Status |
|------|------------|---------------|----------|--------|
| 1 (editordiscovery) | `fe37e3ce` | `c78f0a71` | none needed | Pass |

The RED phase was verified INTENTIONAL (assertion failures against stubbed `discoverEditorWith`/`templateForLauncher`, never a compile error) before the GREEN commit. Following 09-01's own precedent, GSD's TAP-only `tdd-red-evidence` check was not run — this plan's Go-native `<verify>` gates with `<fails_when>` are the authority. The RED transcript is pasted below verbatim.

### RED transcript (`go test -run 'TestDiscoverEditor|TestTemplateForLauncher' ./internal/cli/`)

```
--- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs (0.00s)
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/cursor_found_on_PATH,_stat_never_succeeds (0.00s)
        editordiscovery_test.go:52: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/code_and_cursor_both_on_PATH:_code_wins,_lookPath_called_exactly_once (0.00s)
        editordiscovery_test.go:77: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/darwin_app_bundle:_GoLand_under_/Applications (0.00s)
        editordiscovery_test.go:102: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/darwin_app_bundle_under_$HOME/Applications (0.00s)
        editordiscovery_test.go:129: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/linux_JetBrains_Toolbox_under_XDG_DATA_HOME-derived_path (0.00s)
        editordiscovery_test.go:160: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/linux_/opt_install (0.00s)
        editordiscovery_test.go:187: discoverEditorWith: got false, want true
    --- FAIL: TestDiscoverEditorProbesPathBeforeAppDirs/code's_PATH_probe_happens_before_any_stat_of_a_code_bundle (0.00s)
        editordiscovery_test.go:212: no probes were recorded
--- FAIL: TestDiscoverEditorNeverExecutes (0.00s)
    editordiscovery_test.go:244: lookPath was never called
--- FAIL: TestTemplateForLauncherEmitsAllowlistedTemplates (0.00s)
    editordiscovery_test.go:280: templateForLauncher("code") = ("", ""), want non-empty
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.460s
```

10 `--- FAIL: Test(DiscoverEditor|TemplateForLauncher)` lines, all assertion failures against a real, compiling stub function — no compile error, no zero-test discovery. `TestDiscoverEditorOrderIsCommitted` and `TestDiscoverEditorNotFoundIsNotFatal` passed even in this RED state (they assert data/degenerate-input properties the stub already satisfies by construction — a static literal comparison and a not-found-with-no-error contract respectively), which is expected and does not weaken the RED demonstration: every test whose assertion depends on the probe LOGIC itself failed.

### GREEN transcript (PASS counts)

- Task 1 (`editordiscovery.go`): 12 `--- PASS` lines across `TestDiscoverEditor*`/`TestTemplateForLauncher*`, 0 `--- FAIL`, `inspected 10 launchers` logged.
- Task 2 (`ui.go`/`ui_test.go`): 10 `--- PASS: TestUICommand*` lines (including `TestUICommandDiscoveredDefaultHasProvenance`'s 3 subtests), 0 `--- FAIL`.
- Full `./internal/cli/...` and `./internal/uiserver/...` suites: green (`go test -count=1`).
- `go vet ./...` and `go build ./...`: clean.

## Files Created/Modified

- `internal/cli/editordiscovery.go` — the discoverer: probe tables, `discoverEditorWith`, `discoverEditor`, `templateForLauncher`
- `internal/cli/editordiscovery_test.go` — the five named test functions (with two internal subtests bundled: `TestDiscoverEditorProbesPathBeforeAppDirs` covers 7 of the plan's 8 `<behavior>` scenarios in table-test form)
- `internal/cli/ui.go` — `discoverEditorFn` package-level func var; `discover: nil` -> `discover: discoverEditorFn`
- `internal/cli/ui_test.go` — `TestUICommandDiscoveredDefaultHasProvenance` (3 subtests)

## Decisions Made

See `key-decisions` in frontmatter. In short: `presetTemplate` centralizes the vscode/cursor lookup so no template string is duplicated; the off-switch subtest uses an atomic counter instead of calling `t.Fatal` from a non-test goroutine; and a one-line explanatory comment above the `discover:` struct field was needed to stop gofmt's column-alignment from defeating the plan's own single-space literal verify grep — resolved without touching the grep itself, by shaping the code the grep already expected.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] gofmt's struct-literal alignment defeated the plan's own literal verify grep**
- **Found during:** Task 2, running the plan's own `<verify>` gate for `internal/cli/ui.go` after gofmt
- **Issue:** `resolveEditorLink(editorResolveInputs{...})`'s four fields (`flagTemplate`, `noEditorURL`, `getenv`, `discover`) form one gofmt column-alignment group; gofmt pads every shorter key with spaces so all values start at the same column, producing `discover:     discoverEditorFn` (five spaces) rather than the single space the plan's `rg -o 'discover: discoverEditorFn'` check requires. Reformatting is not optional (gofmt is enforced project-wide), so the check as literally written could not pass against correctly-formatted code.
- **Fix:** Inserted a short explanatory comment directly above the `discover:` field. Comments break gofmt's alignment group, so `discover:` became its own single-field group and gofmt emitted exactly one space after its colon — satisfying both gofmt and the literal grep, with no change to the check and no unidiomatic formatting override.
- **Files modified:** `internal/cli/ui.go`
- **Verification:** `gofmt -l internal/cli/ui.go` reports clean; `rg -o 'discover: discoverEditorFn' internal/cli/ui.go | wc -l` = 1; `rg -o 'discover: nil' internal/cli/ui.go | wc -l` = 0.
- **Committed in:** `cc48f44f` (part of the Task 2 commit).

---

**Total deviations:** 1 auto-fixed (1 Rule 3 — a formatting/verify-gate friction point, not a behavior change).
**Impact on plan:** Zero production-behavior change; the fix is a code-comment addition that happens to also satisfy a literal text-matching check. No scope creep.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The editor-handoff wire slice (09-01) and startup discovery (this plan) together complete BRW-11/BRW-12's server-side precedence chain: flag -> env -> discovered -> unconfigured, with a committed, tested, probe-only discovery mechanism as the third rung.
- `discoverEditorFn` in `internal/cli/ui.go` is the seam future plans (or a future picker feature) can swap for other purposes without touching `discoverEditor` itself.
- **Deferred human-check (D7 above, not a blocker for this plan):** the plan's own end-of-phase UAT step — verifying a real installed editor's discovered default reaches the browser picker and that clicking a gutter line number actually opens the editor at that line — is explicitly harvested to the phase's end (09-05), not evaluated mid-flight here, per the plan's own text. It also depends on SourcePane UI work not yet built in this phase. Record alongside 09-01's still-`[ASSUMED]` Cursor/JetBrains preset templates (09-RESEARCH.md A1/A2) for that same end-of-phase pass.
- No blockers.

## Self-Check: PASSED

All key files present on disk; all three commits (`fe37e3ce`, `c78f0a71`, `cc48f44f`) found in `git log`.

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*
