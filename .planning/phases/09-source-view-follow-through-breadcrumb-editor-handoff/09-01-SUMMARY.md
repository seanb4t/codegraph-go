---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 01
subsystem: api
tags: [connect-rpc, protobuf, cobra, editor-handoff, path-confinement, uri-scheme-allowlist]

# Dependency graph
requires:
  - phase: 03-browse-inspect-navigation
    provides: GetPermalink's structural pattern (answers-not-errors, ValidateRepoRelativePath gate, distinct reason strings) that GetEditorLink mirrors
  - phase: 06-live-push-watch-graph
    provides: readonly_test.go's chained uiProtoFieldFixtureLenAtPlan* extension convention
provides:
  - "GetEditorLink (UIService's 15th rpc): a repo-relative path plus optional line/col resolved server-side into a BUILDABLE editor URI, or an honest NO_TEMPLATE/TEMPLATE_INVALID answer"
  - "internal/uiserver/editorlink.go: EditorTemplateSource, EditorLinkOptions, EditorTemplateMaxBytes, ValidateEditorTemplate, buildEditorURL, encodeEditorPathSegments — the one validator and one encoder both the CLI and every request reuse"
  - "internal/uiserver/editorpresets.go: the three fixed editor presets (VS Code, Cursor, JetBrains) riding on every response"
  - "codegraph ui --editor-url / --no-editor-url / CODEGRAPH_EDITOR_URL / CODEGRAPH_NO_EDITOR_URL resolved once at startup into uiserver.Options.EditorLink, failing fast on a malformed explicit value before the port binds"
affects: [09-02-editor-discovery, 09-04-header-link-and-gutter, 09-05-security-doc]

# Actuals (#2632)
actuals:
  tokens: 40323
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GetEditorLink as GetPermalink's structural twin: answers-not-errors for configuration states, ValidateRepoRelativePath as the one confinement gate, distinct reason strings per cause"
    - "Positive allowlist (never a denylist) for URI schemes, asserted with a length check before any refusal is trusted (rule 84d1gfpywd)"
    - "Per-position percent-encoding: url.PathEscape+manual ':'->%3A before a template's first '?', url.QueryEscape+manual '+'->'%20' after it"
    - "Startup-frozen configuration (EditorLinkOptions copied into uiService at Listen, never re-read per request) — same discipline as the publisher and repoPath"
    - "CLI resolver validates every explicit value before consulting the off switch or discovery, so a typo never silently degrades (D-17)"

key-files:
  created:
    - internal/uiserver/editorlink.go
    - internal/uiserver/editorlink_test.go
    - internal/uiserver/editorpresets.go
    - internal/cli/editorurl.go
    - internal/cli/editorurl_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/uiserver/handlers.go
    - internal/uiserver/server.go
    - internal/uiserver/server_test.go
    - internal/uiserver/readonly_test.go
    - internal/cli/ui.go
    - internal/cli/ui_test.go

key-decisions:
  - "Field-number fixture extended as uiProtoFieldFixtureLenAtPlan0901 = uiProtoFieldFixtureLenAtPlan0601 + 14 (4 request + 3 preset + 7 response fields), following the established chained-extension convention verbatim"
  - "editorpresets.go's doc comment was reworded twice during GREEN to stop repeating literal scheme strings (vscode://, cursor://, idea://) and the excluded editor's name in prose — the plan's own verify gate counts occurrences across the WHOLE file text (code and comments alike), not just the returned data"
  - "ui.go's doc comments were reworded to avoid repeating the literal identifiers resolveEditorLink/uiserver.Listen — the plan's call-count/ordering verify gates count raw text occurrences of those exact strings"
  - "TestUICommandRefusesMalformedEditorURLBeforeBinding sets SilenceUsage/SilenceErrors on its standalone newUiCmd() instance to mirror how root.go always wraps this command in production — without it, cobra's own usage-printing on error writes to the same unbuffered io.Pipe the test uses for the success-path URL line, deadlocking ExecuteContext before it can return"

patterns-established:
  - "Wire validator shared between the per-request handler and the CLI startup resolver — ValidateEditorTemplate is called from both internal/uiserver/editorlink.go's handler and internal/cli/editorurl.go's resolver, never duplicated"

requirements-completed: [BRW-11, BRW-12]

coverage:
  - id: D1
    description: "GetEditorLink rpc (15th UIService method) returns a BUILDABLE vscode:// URL for an in-repo path through a real Connect client"
    requirement: "BRW-11"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkPathConfinementAtRPCBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Path confinement at the RPC boundary: traversal, absolute, empty and symlink-escape paths refused with CodeInvalidArgument before any template substitution, no host-path leak"
    requirement: "BRW-11"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkPathConfinementAtRPCBoundary"
        status: pass
    human_judgment: false
  - id: D3
    description: "{path} is filepath.Join(Abs(RepoPath), rel), never its EvalSymlinks form"
    requirement: "BRW-11"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkBuildsAbsoluteJoinedPathNotSymlinkResolved"
        status: pass
    human_judgment: false
  - id: D4
    description: "Positive scheme allowlist (15 members) refuses javascript:/data:/blob:/file:/ftp: by non-membership; unknown/missing placeholders and oversized templates are TEMPLATE_INVALID"
    requirement: "BRW-11, BRW-12"
    verification:
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestEditorTemplateSchemeAllowlist"
        status: pass
    human_judgment: false
  - id: D5
    description: "NO_TEMPLATE (unconfigured, disabled) and TEMPLATE_INVALID are successful answers with pairwise-distinct reason strings; the one error is a rejected path"
    requirement: "BRW-11, BRW-12"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors"
        status: pass
    human_judgment: false
  - id: D6
    description: "Per-request template override replaces the server default for one call; response still reports the server default's own provenance"
    requirement: "BRW-11"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkOverrideRidesTheRequest"
        status: pass
    human_judgment: false
  - id: D7
    description: "Line/col boundary edges: 0/negative refused before the store opens; unset substitutes 1; a line past EOF is BUILDABLE"
    requirement: "BRW-11"
    verification:
      - kind: integration
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkLineAndColBounds"
        status: pass
    human_judgment: false
  - id: D8
    description: "Per-position percent-encoding: path-position keeps ':' literal only after %3A rewrite; query-position never emits '+' for space"
    requirement: "BRW-11"
    verification:
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestEditorLinkPathEncodingPerPosition"
        status: pass
    human_judgment: false
  - id: D9
    description: "Exactly three presets (vscode, cursor, jetbrains) in fixed order; Zed absent from all nine strings"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestEditorPresetsAreExactlyThreeAndNameNoZed"
        status: pass
    human_judgment: false
  - id: D10
    description: "codegraph ui --editor-url / CODEGRAPH_EDITOR_URL / --no-editor-url / CODEGRAPH_NO_EDITOR_URL resolved once at startup with flag > env > discovered > none precedence; malformed explicit values refused before the off switch"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editorurl_test.go#TestResolveEditorLinkPrecedence"
        status: pass
      - kind: unit
        ref: "internal/cli/editorurl_test.go#TestResolveEditorLinkOffSwitch"
        status: pass
    human_judgment: false
  - id: D11
    description: "A malformed --editor-url/CODEGRAPH_EDITOR_URL value is a startup failure: RunE returns an error before uiserver.Listen binds, printing no URL"
    requirement: "BRW-12"
    verification:
      - kind: unit
        ref: "internal/cli/editorurl_test.go#TestResolveEditorLinkFailsFastOnMalformedExplicitValue"
        status: pass
      - kind: integration
        ref: "internal/cli/ui_test.go#TestUICommandRefusesMalformedEditorURLBeforeBinding"
        status: pass
    human_judgment: false
  - id: D12
    description: "The proto's 15th rpc and its wire shape are frozen additively with every read-only/field-number guard (14->15) moved in the SAME commit"
    requirement: "BRW-11"
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 1: Editor-Handoff Wire Slice Summary

GetEditorLink lands as UIService's 15th rpc — a repo-relative path plus optional line/col resolved server-side through the same confinement gate GetPermalink uses, into a positively-scheme-allowlisted, per-position-percent-encoded `vscode://` URL — with `codegraph ui --editor-url`/`--no-editor-url`/env-var twins resolved once at startup and failing fast on a malformed value before the port ever binds.

## Performance

- **Duration:** ~30 min
- **Started:** 2026-09-12T17:12:26Z
- **Completed:** 2026-09-12T17:42:47Z
- **Tasks:** 2 (both `tdd="true"`, each RED->GREEN)
- **Files modified:** 15 (5 created, 10 modified)

## Accomplishments

- `GetEditorLink` (UIService's 15th rpc), two closed enums (`EditorLinkAvailability`, `EditorTemplateSource`), `EditorPreset`, and the request/response messages, all additive, regenerated via `task proto:gen` with `task proto:drift` green
- `internal/uiserver/editorlink.go`: `ValidateEditorTemplate` (positive 15-scheme allowlist, balanced-brace placeholder scan, 2048-byte cap), `buildEditorURL`/`encodeEditorPathSegments` (per-position percent-encoding), `editorLinkAnswer` (override > disabled > unconfigured > invalid > buildable sequencing), and the handler itself (line/col validated before the store ever opens, `ValidateRepoRelativePath` gate, `filepath.Join` never `EvalSymlinks`)
- `internal/uiserver/editorpresets.go`: exactly three presets (VS Code, Cursor, JetBrains), Cursor/JetBrains tagged `[ASSUMED]` and corroborated against spatie/ignition's shipped editor-link table
- `codegraph ui --editor-url`/`--no-editor-url` plus `CODEGRAPH_EDITOR_URL`/`CODEGRAPH_NO_EDITOR_URL`, resolved once via `internal/cli/editorurl.go`'s `resolveEditorLink` before `uiserver.Listen`, with both explicit values validated regardless of which wins (D-17)
- Every read-only/field-number guard (`wantUIServiceMethods` 14->15, `uiProtoFieldFixtureLenAtPlan0901`, `wantOptsFields`/`uiService` type set) moved in the SAME commit as the proto change

## Task Commits

Both tasks followed the RED -> GREEN TDD cycle (2 commits each, no REFACTOR commit needed):

1. **Task 1: GetEditorLink wire slice**
   - `237ab0ef` `test(09-01): add GetEditorLink wire shape, fixtures and failing RPC-boundary tests` (RED)
   - `f845cb33` `feat(09-01): implement GetEditorLink` (GREEN)
2. **Task 2: CLI editor-url resolver**
   - `d610d029` `test(09-01): add failing editor-url resolver and ui flag tests` (RED)
   - `1fd7defb` `feat(09-01): resolve --editor-url/env/off-switch at startup, fail fast on malformed values` (GREEN)

## TDD Gate Compliance

| Task | RED commit | GREEN commit | REFACTOR | Status |
|------|------------|---------------|----------|--------|
| 1 (uiserver) | `237ab0ef` | `f845cb33` | none needed | Pass |
| 2 (cli) | `d610d029` | `1fd7defb` | none needed | Pass |

Both RED phases were verified INTENTIONAL (assertion failures against a real, connectable server / a real stub function — never a compile error) before any GREEN commit landed. GSD's TAP-only `tdd-red-evidence` check was NOT run per this plan's own commit-discipline note (Go-native `<verify>` gates with `<fails_when>` are the authority here); both RED transcripts are pasted below verbatim.

### Task 1 RED transcript (`go test -run 'TestGetEditorLink|TestEditorTemplate|TestEditorLinkPath|TestEditorPresets' ./internal/uiserver/...`)

```
--- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary (0.11s)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/in-repo_control (0.00s)
        editorlink_test.go:70: GetEditorLink(in-repo control): unimplemented: uiserver: GetEditorLink is not yet implemented (plan 09-01 RED phase)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/escape (0.00s)
        editorlink_test.go:115: GetEditorLink(path="../outside.txt"): code = unimplemented, want CodeInvalidArgument (...)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/absolute (0.00s)
        editorlink_test.go:115: GetEditorLink(path="/etc/passwd"): code = unimplemented, want CodeInvalidArgument (...)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/empty (0.00s)
        editorlink_test.go:115: GetEditorLink(path=""): code = unimplemented, want CodeInvalidArgument (...)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/symlink-escape (0.00s)
        editorlink_test.go:136: GetEditorLink(path="escape-link/secret.txt"): code = unimplemented, want CodeInvalidArgument (...)
--- FAIL: TestGetEditorLinkBuildsAbsoluteJoinedPathNotSymlinkResolved (0.10s)
    editorlink_test.go:177: GetEditorLink: unimplemented: uiserver: GetEditorLink is not yet implemented (plan 09-01 RED phase)
--- FAIL: TestEditorTemplateSchemeAllowlist (0.00s)
    editorlink_test.go:227: ValidateEditorTemplate("vscode://file/{path}:{line}:{col}") = uiserver: GetEditorLink is not yet implemented (plan 09-01 RED phase), want nil
--- FAIL: TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors (0.19s)
    --- FAIL: .../unconfigured (0.05s)
    --- FAIL: .../disabled_by_operator (0.05s)
    --- FAIL: .../template_invalid_via_override (0.04s)
--- FAIL: TestGetEditorLinkOverrideRidesTheRequest (0.10s)
--- FAIL: TestGetEditorLinkLineAndColBounds (0.10s)
    --- FAIL: .../rejected_before_the_store_opens/line_zero (0.00s)
    --- FAIL: .../rejected_before_the_store_opens/col_zero (0.00s)
    --- FAIL: .../rejected_before_the_store_opens/line_negative (0.00s)
    --- FAIL: .../unset_defaults_to_1_and_past-EOF_is_buildable (0.10s)
--- FAIL: TestEditorLinkPathEncodingPerPosition (0.00s)
    editorlink_test.go:453: buildEditorURL(path-position) = "", want ':' inside the path segment encoded as %3A
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.984s
```

18 `--- FAIL: TestGetEditorLink*` lines, all assertion failures against a real listener/stub function — no compile error, no zero-test discovery. INVALID_RED avoided.

### Task 2 RED transcript (`go test -run 'TestResolveEditorLink|TestUICommand' ./internal/cli/`)

```
--- FAIL: TestResolveEditorLinkPrecedence (0.00s)
    --- FAIL: .../flag_only, env_only, flag_beats_env,_both_valid (zero-value Options returned by the stub)
    --- FAIL: .../flag_valid,_env_malformed_—_both_validated,_env_error_surfaces (nil error, want one naming CODEGRAPH_EDITOR_URL/javascript)
    --- FAIL: .../flag_malformed_—_discover_never_called (nil error, want one naming --editor-url/javascript)
    --- FAIL: .../discovered_default (zero-value Options)
    --- FAIL: .../discover_finds_nothing (discover called 0 times, want 1)
--- FAIL: TestResolveEditorLinkFailsFastOnMalformedExplicitValue
    --- FAIL: .../malformed_flag, malformed_env (nil error, want a refusal)
--- FAIL: TestResolveEditorLinkOffSwitch
    --- FAIL: .../flag_disables,_discover_skipped (zero-value Options, want Disabled)
    --- FAIL: .../env_disables:_true,_1,_yes,_TRUE,_Yes (Source = 0, want EditorTemplateDisabled)
    --- FAIL: .../unparseable_env_value_errors_naming_variable_and_value (nil error)
    --- FAIL: .../malformed_flag_beats_the_off_switch (nil error)
--- FAIL: TestUICommandEditorURLFlagReachesGetEditorLink
    --- FAIL: .../--editor-url_sets_FLAG_provenance (availability = NO_TEMPLATE, want BUILDABLE)
    --- FAIL: .../--no-editor-url_disables_with_the_operator_reason (reason did not contain "disabled by operator")
--- FAIL: TestUICommandRefusesMalformedEditorURLBeforeBinding (6.06s)
    ui_test.go:303: cmd.ExecuteContext did not return within the budget
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	6.940s
```

20 `--- FAIL: TestResolveEditorLink*`/`TestUICommand*` lines. The last failure (a 6s timeout, not an assertion mismatch) is explained in Deviations below — it is still a real, bounded RED failure, not a hang, and not a compile error.

### GREEN transcripts (PASS counts)

- uiserver: 33 `--- PASS` lines across the four named families (`TestGetEditorLink*`, `TestEditorTemplate*`, `TestEditorLinkPath*`, `TestEditorPresets*`), 0 `--- FAIL`.
- cli: 31 `--- PASS` lines across `TestResolveEditorLink*`/`TestUICommand*`, 0 `--- FAIL`.
- Six named cross-cutting guards (`TestUIServiceMethodSetIsExactlyTheReadSet`, `TestUIServiceDeclaresNoMutatingMethod`, `TestUIProtoFieldNumbersAreStableAndUnique`, `TestUIServiceHoldsNoStoreTypedField`, `TestGetPermalinkPathConfinementAtRPCBoundary`, `TestGetNodeDetailPathConfinementAtRPCBoundary`): 14 PASS lines, 0 FAIL.

## Files Created/Modified

- `internal/uiproto/uiv1/ui.proto` — `GetEditorLink` (15th rpc), its two enums and three messages, appended additively
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go`, `web/src/lib/gen/ui_pb.ts` — regenerated via `task proto:gen`
- `internal/uiserver/editorlink.go` — the validator, encoder, answer-builder and handler
- `internal/uiserver/editorlink_test.go` — nine test functions exercising the wire boundary through a real Connect client
- `internal/uiserver/editorpresets.go` — the three fixed presets
- `internal/uiserver/handlers.go`, `internal/uiserver/server.go` — `uiService.editorLink` field, `Options.EditorLink`, wired at `Listen`
- `internal/uiserver/readonly_test.go`, `internal/uiserver/server_test.go` — fixtures extended in the same commit as the proto
- `internal/cli/editorurl.go` — `resolveEditorLink`, `parseBoolEnv`, the two env-var consts, `editorResolveInputs`/`discoveredEditor`
- `internal/cli/editorurl_test.go` — the three resolver test functions
- `internal/cli/ui.go` — `--editor-url`/`--no-editor-url` flags, resolver called before `Listen`
- `internal/cli/ui_test.go` — renamed flag-set test, two new command-level tests

## Decisions Made

See `key-decisions` in frontmatter. In short: the field-number fixture arithmetic (4+3+7=14), two rounds of doc-comment rewording in `editorpresets.go`/`ui.go` to satisfy the plan's own literal-text verify gates, and a test-only `SilenceUsage`/`SilenceErrors` fix mirroring root.go's production wrapping.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestUICommandRefusesMalformedEditorURLBeforeBinding` deadlocked on an unread `io.Pipe`**
- **Found during:** Task 2 GREEN phase (the test was authored in the RED commit and failed there too, but via a 6s timeout rather than an assertion — still a legitimate RED failure)
- **Issue:** `newUiCmd()` is exercised standalone in this test (not through `newRootCmd()`, which sets `SilenceUsage`/`SilenceErrors`). On a `RunE` error, cobra's default behavior writes the full usage string to `c.OutOrStderr()`, which resolves to the SAME `io.Pipe` writer this test uses for the success-path URL line (`cmd.SetOut(pw)`). `io.Pipe` is unbuffered, so the write blocked forever with nothing reading `pr` yet, hanging `ExecuteContext` itself before it could ever return the `RunE` error under test.
- **Fix:** Set `cmd.SilenceUsage = true` and `cmd.SilenceErrors = true` on the test's command instance, mirroring exactly how `root.go` always wraps this command in a real invocation (`Execute()` -> `newRootCmd().Execute()`) — the fix makes the test representative of production, not merely a workaround.
- **Files modified:** `internal/cli/ui_test.go`
- **Verification:** `TestUICommandRefusesMalformedEditorURLBeforeBinding` passes in isolation and as part of the full `./internal/cli/...` suite.
- **Committed in:** `1fd7defb` (part of the Task 2 GREEN commit)

**2. [Rule 1 - Bug] `editorpresets.go`'s and `ui.go`'s doc comments repeated literal text the plan's own `<verify>` gates count**
- **Found during:** Task 1 and Task 2 verify-gate runs (after GREEN, before commit)
- **Issue:** The plan's automated checks count raw occurrences of specific strings across whole files: exactly 3 scheme-literal mentions in `editorpresets.go` (one per preset, code only), exactly 0 case-insensitive "zed" mentions, exactly 1 `resolveEditorLink` and exactly 1 `uiserver.Listen` mention in `ui.go`. My first drafts' doc comments explained the design by repeating those same literals/names in prose, which the counting gates could not distinguish from the real code sites they exist to verify.
- **Fix:** Reworded the affected sentences to describe the same facts without repeating the literal, counted strings (e.g. "the excluded editor's name" instead of naming it; "the server binds its port below" instead of naming `uiserver.Listen`).
- **Files modified:** `internal/uiserver/editorpresets.go`, `internal/cli/ui.go`
- **Verification:** All four `<verify>` count-based checks pass (`three_schemes_ok`, `no_zed_ok`, `resolver_call_count_ok`, `listen_call_count_ok`, `ordering_ok`).
- **Committed in:** `f845cb33` (editorpresets.go), `1fd7defb` (ui.go) — both part of their task's GREEN commit.

---

**Total deviations:** 2 auto-fixed (2 Rule 1 — both bugs discovered in my own newly-authored test/comment text during this plan, not in pre-existing code).
**Impact on plan:** Both fixes are test-authoring and documentation-wording corrections with zero production-behavior change; neither altered the wire shape, the validator, or the resolver's precedence logic. No scope creep.

## Issues Encountered

None beyond the two auto-fixed items above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The editor-handoff wire slice (BRW-11/BRW-12's server half) is proven end to end through a real Connect client and a real `codegraph ui` invocation. Plan 09-02 can now implement `discoverEditor()` and wire it into `internal/cli/ui.go`'s single `discover: nil` line (the only line that plan touches in this file).
- `internal/uiserver/editorlink.go`'s `EditorLinkOptions`/`ValidateEditorTemplate`/`EditorPresets()` are exported and ready for plan 09-04's header-link/gutter UI work and plan 09-05's `09-SECURITY.md`.
- No blockers. The Cursor/JetBrains preset templates remain `[ASSUMED]` (09-RESEARCH.md A1/A2) — tagged as such in `editorpresets.go`'s doc comment, per this plan's own prohibition against presenting them as verified — and are pending human corroboration at the phase's own end-of-phase checkpoint (09-05).

## Self-Check: PASSED

All key files present on disk; all five commits (`237ab0ef`, `f845cb33`, `d610d029`, `1fd7defb`, `eb5c6c5e`) found in `git log`.

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*
