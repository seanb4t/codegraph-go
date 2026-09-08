---
phase: 03-browse-inspect-navigation
plan: 05
subsystem: api
tags: [connectrpc, protobuf, git, github, permalink, confinement, tdd]

# Dependency graph
requires:
  - phase: 03-browse-inspect-navigation (plan 03-02)
    provides: the resolveSourcePath confinement gate GetNodeDetail already proved at the RPC boundary — GetPermalink reuses it via ValidateRepoRelativePath, never a second implementation
  - phase: 03-browse-inspect-navigation (plan 03-04)
    provides: the rpc-errors.ts classification the frontend routes GetPermalink's Connect errors through (serialization edge only — no code dependency)
provides:
  - "UIService.GetPermalink: a tenth, additive, read-only rpc turning a repo-relative path + optional line/end_line into a GitHub blob URL pinned to the INDEXED commit"
  - "internal/gitmeta.RemoteGitHubRepo / RemoteURL / CommitOnRemoteTrackingBranch: reusable git remote-derivation and pushed-commit-check primitives, degrade-to-a-value, never to a bare bool or a swallowed error"
  - "(*query.Engine).ValidateRepoRelativePath: an exported delegating wrapper any future caller can use to confine a path without a second read/parse"
affects: [04-graph-view, 06-live-push]

# Actuals (#2632)
actuals:
  tokens: 27091
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tri-state degrade (RemotePresence{Unknown,Observed,NotObserved}) instead of a bool or (bool, error) whenever 'could not check' and 'checked, found nothing' must stay distinguishable to the caller"
    - "Structured no-answer struct (GitHubRemote{Owner,Repo,Host,Reason}) instead of a bare pair, so every failure path can name WHICH cause held instead of collapsing to one undifferentiated refusal"
    - "Reclassifying one already-refused confinement case for PRESENTATION only, in the RPC handler, by threading a connect.Error around withEngine's own automatic mapEngineError re-wrap via an outer captured variable — never returning a *connect.Error directly from withEngine's closure, which gets scrubbed to CodeInternal a second time"

key-files:
  created:
    - internal/gitmeta/permalink.go
    - internal/gitmeta/permalink_test.go
    - internal/uiserver/permalink.go
    - internal/uiserver/permalink_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/query/node.go
    - internal/uiserver/readonly_test.go

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human): wire shape frozen exactly as sketched — GetPermalinkRequest{path=1,line=2 optional,end_line=3 optional}, GetPermalinkResponse{url=1,availability=2,reason=3}, PermalinkAvailability enum {UNSPECIFIED,LINKABLE,LINKABLE_UNVERIFIED,NO_LINK} — CHOSEN as an enum over an open string for compiler-checked exhaustiveness."
  - "Task 1 checkpoint: tri-state git-failure contract CHOSEN (yes) — RemotePresence{Unknown,Observed,NotObserved} plus structured GitHubRemote{Owner,Repo,Host,Reason}, resolving the cycle-1 HIGH finding that a swallowed check failure would render a transient git error as a confident NO_LINK."
  - "Task 1 checkpoint, point 5: deleted-classify-in-handler CHOSEN (NOT the deleted-accept-internal default). Maintainer rationale recorded verbatim: shared permalinks outlive the files they point at, so a since-deleted file is a NORMAL outcome for this feature, and ROADMAP criterion 5 requires an explicit named state rather than the opaque CodeInternal an unhandled fs.ErrNotExist would otherwise scrub to (bypassing 03-09's source-absent messaging, which only covers an absent SourceBlob on a SUCCESSFUL call). Implemented in internal/uiserver/permalink.go ONLY, matched via errors.Is(err, fs.ErrNotExist), message built from the caller's own repo-relative path, never the underlying *fs.PathError text (which carries the absolute host checkout path). TestGetPermalinkRefusesSinceDeletedFile pins CodeInvalidArgument with a paired positive (repo-relative path present) and negative (absolute host path absent) containment check."
  - "Deviation: TestUIServiceMethodSetIsExactlyTheReadSet's hardcoded 'len(got) != 9' / 'want exactly 9' literal was updated to 10 alongside the wantUIServiceMethods map entry, even though the plan text said 'change nothing else in that file'. This was mechanically required — the hardcoded literal makes the test permanently unpassable once a tenth method exists, contradicting the plan's own acceptance criterion that this exact test must go GREEN after the fixture edit. Structure (two-fold check: hardcoded count + fixture-size match, full bidirectional membership loop) is unchanged; only the two numeric/string occurrences of the old count moved from 9 to 10."

requirements-completed: [BRW-09, SRV-05]

coverage:
  - id: D1
    description: "GetPermalink returns a GitHub blob URL pinned to the indexed commit (never HEAD), with a single-line anchor, a range anchor, or no anchor at all depending on the request's line/end_line"
    requirement: BRW-09
    verification:
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_LinkableSingleLineAnchor"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_LinkableRangeAnchor"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_LinkableNoLineAnchorAtAll"
        status: pass
    human_judgment: false
  - id: D2
    description: "availability is a three-valued, never-boolean classification: LINKABLE (commit observed on a remote-tracking branch), LINKABLE_UNVERIFIED (not observed, or the check could not run at all — both cases share the wire value but reason differs), NO_LINK (no GitHub remote, or no indexed commit_sha)"
    requirement: BRW-09
    verification:
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_LinkableUnverifiedWhenCommitNotObserved"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_NoLinkNonGitHubRemote"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_NoLinkNoRemote"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalink_NoLinkNoCommitSHA"
        status: pass
      - kind: unit
        ref: "internal/gitmeta/permalink_test.go#TestCommitOnRemoteTrackingBranch_UnknownIsNotNotObserved"
        status: pass
    human_judgment: false
  - id: D3
    description: "GetPermalinkRequest.path is confined by the same gate GetNodeDetail uses (ValidateRepoRelativePath -> resolveSourcePath), proven at the RPC boundary with a passing positive control alongside escape/absolute refusals; the since-deleted-file case is reclassified to an actionable CodeInvalidArgument per the maintainer's checkpoint decision, without leaking the absolute host path"
    requirement: SRV-05
    verification:
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalinkPathConfinementAtRPCBoundary"
        status: pass
      - kind: integration
        ref: "internal/uiserver/permalink_test.go#TestGetPermalinkRefusesSinceDeletedFile"
        status: pass
    human_judgment: false
  - id: D4
    description: "Following the returned URL in a browser lands on the same file and line the UI was showing (plan's own backstop truth, marked verification: backstop)"
    verification: []
    human_judgment: true
    rationale: "Requires an actual GitHub-hosted repository and a browser to click through — no automated proof is possible from this sandboxed dev environment against a synthetic fixture. The URL-assembly logic itself (owner/repo/sha/path/anchor composition and percent-encoding) is unit- and integration-tested exhaustively; only the live end-to-end browser click is unverified."

duration: ~40min active work (excludes wait time for the Task 1 human checkpoint decision)
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 5: GitHub Permalinks (GetPermalink) Summary

**Added `UIService.GetPermalink` — a tenth, additive rpc that turns a repo-relative path and an optional line/end_line into a GitHub blob URL pinned to the indexed commit, with a three-valued `availability` (LINKABLE/LINKABLE_UNVERIFIED/NO_LINK) that never collapses "could not check" into a false claim of no-link, and a since-deleted-file case reclassified to an actionable `CodeInvalidArgument` per the maintainer's own checkpoint decision.**

## Performance

- **Duration:** ~40 min active work (excludes the Task 1 human checkpoint wait)
- **Tasks:** 3 (1 checkpoint:decision, 2 TDD)
- **Files:** 4 created, 6 modified
- **Commits:** 5

## Accomplishments

- `internal/gitmeta/permalink.go`: `RemoteGitHubRepo`, `RemoteURL`, and `CommitOnRemoteTrackingBranch` — remote-URL derivation across https/scp-like/ssh transport shapes with `url.<base>.insteadOf` support, exact-match GitHub host detection (no `Contains`/`HasPrefix`/`HasSuffix` anywhere in the file), credential stripping, and a `RemotePresence` tri-state that keeps "checked and not observed" distinct from "could not check" all the way through. 16 named tests, all passing.
- `internal/uiproto/uiv1/ui.proto` gains `GetPermalink` additively (0 line deletions, verified via `git diff --numstat`), regenerated through the pinned toolchain with `task proto:drift` staying green at the same 4-file floor.
- `(*query.Engine).ValidateRepoRelativePath`: a one-line delegating wrapper exposing the existing confinement gate to the wire layer, with no path logic of its own.
- `internal/uiserver/permalink.go`: the handler, going through `withEngine` for the indexed commit SHA and the confinement wrapper, answering honestly (never an error) for every git-introspection outcome, and reclassifying the since-deleted-file case in this one handler only.
- Extended both of `internal/uiserver/readonly_test.go`'s cross-plan security fixtures (`wantUIServiceMethods`, `uiProtoFieldNumbers`) to cover the tenth method and its six new fields, keeping their bidirectional-equality guards non-vacuous.

## Task Commits

Each task was committed atomically, following RED-GREEN TDD discipline for Tasks 2 and 3:

1. **Task 1: Freeze the GetPermalink wire shape** — no commit (checkpoint:decision, human-answered; the decision is recorded above and consumed by Tasks 2/3, not itself a code change)
2. **Task 2: Git remote derivation, exact GitHub detection, tri-state pushed check**
   - `3fc1477` — `test(03-05): add failing tests for git remote derivation and pushed-commit check` (RED: build fails, functions undefined)
   - `46fe7f2` — `feat(03-05): implement git remote derivation and tri-state pushed-commit check` (GREEN: 16/16 passing)
3. **Task 3: The additive proto method, the handler, and its confinement proof**
   - `49b357e9` — `feat(03-05): add GetPermalink to the wire schema and expose confinement wrapper` (proto + regen + `ValidateRepoRelativePath`)
   - `1dff272` — `test(03-05): add failing tests for GetPermalink and extend the read-set/field fixtures` (RED: package fails to build, `*uiService` missing `GetPermalink`)
   - `52dadeb` — `feat(03-05): implement GetPermalink handler with confinement and since-deleted-file reclassification` (GREEN: 13/13 passing)

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/gitmeta/permalink.go` — remote-URL derivation, GitHub host detection, pushed-commit tri-state check
- `internal/gitmeta/permalink_test.go` — 16 named tests covering every transport shape and degrade path
- `internal/uiserver/permalink.go` — the `GetPermalink` RPC handler
- `internal/uiserver/permalink_test.go` — RPC-boundary tests: line/range/no-anchor, all three availability states, percent-encoding, confinement + positive control, the pinned since-deleted-file test
- `internal/uiproto/uiv1/ui.proto` — `GetPermalink` rpc, `GetPermalinkRequest`/`GetPermalinkResponse`/`PermalinkAvailability`, additive-only
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go`, `web/src/lib/gen/ui_pb.ts` — regenerated via `task proto:gen`
- `internal/query/node.go` — `ValidateRepoRelativePath` confinement wrapper
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` and `uiProtoFieldNumbers` extended for the tenth rpc

## Decisions Made

See `key-decisions` in frontmatter for the full text of the three checkpoint decisions (wire shape, tri-state git-failure contract, since-deleted-file disposition) plus the one auto-fix deviation (the `readonly_test.go` hardcoded method-count literal).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Threaded the since-deleted-file `connect.Error` around `withEngine`'s automatic re-wrap**

- **Found during:** Task 3, writing `TestGetPermalinkRefusesSinceDeletedFile`
- **Issue:** Returning `connect.NewError(connect.CodeInvalidArgument, ...)` directly from `withEngine`'s closure gets passed through `withEngine`'s own unconditional `mapEngineError(err)` call a second time. `mapEngineError` does not recognize an already-built `*connect.Error` as any of its classified sentinels, so it fell to the default arm and re-scrubbed the response to an opaque `CodeInternal` — observed live as the first RED run for this test (`code = internal, want CodeInvalidArgument`).
- **Fix:** Introduced an outer `classifiedErr` variable captured by the closure; the since-deleted-file branch sets it and returns `nil` from the closure (so `withEngine` itself returns `nil`), and `GetPermalink` checks `classifiedErr` after `withEngine` returns, before building the success response.
- **Files modified:** `internal/uiserver/permalink.go`
- **Verification:** `TestGetPermalinkRefusesSinceDeletedFile` passes; the escape/absolute-path refusal path (which legitimately flows through `mapEngineError`) is unaffected and still passes.
- **Committed in:** `52dadeb`

**2. [Rule 3 - Blocking] Updated `TestUIServiceMethodSetIsExactlyTheReadSet`'s hardcoded method-count literal**

- **Found during:** Task 3, step (a1)
- **Issue:** The plan text said to add exactly one map entry to `wantUIServiceMethods` and "change nothing else in that file — not the length assertion". The test ALSO contains a separate hardcoded `if len(got) != 9` check (distinct from the fixture-size comparison). Left at 9, the test can never pass again once a tenth method exists, contradicting the plan's own acceptance criterion that this exact test go GREEN after the fixture edit.
- **Fix:** Updated the two occurrences of the literal `9` (the `!=` comparison and its error-message text) to `10`, and the trailing error message's "nine-name" to "ten-name". No other structure changed: still a two-fold check (hardcoded count + fixture-size agreement) and a full bidirectional membership loop.
- **Files modified:** `internal/uiserver/readonly_test.go`
- **Verification:** RED observed first (`has 10 methods, want exactly 9`) before the fixture edit, then GREEN after both the map entry and the literal were updated.
- **Committed in:** `1dff272`

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues preventing a stated acceptance criterion from ever being satisfiable). **Impact:** Both fixes were necessary for internal consistency between the plan's own text and its own acceptance criteria; neither expands scope beyond what Task 3 already specified.

## Issues Encountered

None beyond the two deviations above (which were caught, fixed, and verified within the same task rather than escalated).

## RED Proofs (recorded per acceptance criteria)

- **Task 2, git remote derivation:** `internal/gitmeta/permalink_test.go` failed to compile (`undefined: RemoteGitHubRepo`, etc.) before `internal/gitmeta/permalink.go` existed — RED confirmed via `go test ./internal/gitmeta/ -run 'Permalink|Remote' -count=1 -v` exiting non-zero with `undefined` errors.
- **Task 2, host-exactness and credential-stripping:** covered by `TestRemoteGitHubRepo_LookalikeHostRejected` and `TestRemoteGitHubRepo_CredentialsStripped`, both passing GREEN in the same run that proved the other 14 cases (16/16 total).
- **Task 3, read-set fixture:** `go test ./internal/uiserver/ -run '^TestUIServiceMethodSetIsExactlyTheReadSet$' -count=1 -v` failed RED with `has 10 methods, want exactly 9: map[... GetPermalink:{} ...]` before the fixture edit; GREEN after.
- **Task 3, confinement/since-deleted-file:** `TestGetPermalinkRefusesSinceDeletedFile` failed RED with `code = internal, want CodeInvalidArgument` (the `withEngine` double-wrap deviation above) before the `classifiedErr` fix; GREEN after.
- **Task 3, package build (interface satisfaction):** confirmed by temporarily moving `permalink.go` aside — `go build ./internal/uiserver/...` failed with `*uiService does not implement uiv1connect.UIServiceHandler: missing method GetPermalink`; restored and GREEN.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `GetPermalink` is live on the wire, additively, with `task proto:drift` still reporting the same 4-file floor.
- Ready for `03-06` (next wave), which per its own frontmatter now depends on `03-05` for serialization ordering on the shared generated-client file.
- One coverage item (D4, the live browser click-through end-to-end) is unverified from this sandboxed environment and flagged `human_judgment: true` for manual UAT.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

All 4 created source files and the SUMMARY itself verified present on disk via `[ -f ]`; all 5 task commit hashes (`3fc1477`, `46fe7f2`, `49b357e9`, `1dff272`, `52dadeb`) verified present via `git log --oneline --all`.
