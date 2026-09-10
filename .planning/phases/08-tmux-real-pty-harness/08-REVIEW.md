---
status: findings
phase: 08-tmux-real-pty-harness
depth: deep
files_reviewed: 13
files_reviewed_list:
  - .github/workflows/ci.yml
  - Taskfile.yml
  - internal/upgrade/taskfile_shape_test.go
  - test/tmux/capture.go
  - test/tmux/confighash.go
  - test/tmux/daemon_empty_test.go
  - test/tmux/daemon_picker_test.go
  - test/tmux/daemon_seed.go
  - test/tmux/frame_stability_test.go
  - test/tmux/install_cancel_test.go
  - test/tmux/main_test.go
  - test/tmux/session.go
  - test/tmux/skip_contract_test.go
critical: 0
warning: 5
info: 4
reviewed_at: 2026-09-10T17:07:18Z
---

# Phase 08: tmux Real-PTY Harness — Code Review Report

**Reviewed:** 2026-09-10T17:07:18Z
**Depth:** deep (cross-file, plus targeted empirical reproduction of two failure scenarios)
**Files Reviewed:** 13
**Status:** issues_found (no Critical, 5 Warning, 4 Info)

## Summary

This is a well-constructed test-infrastructure phase. The three highest-value assertions
(`TestDaemonEmptyRegistryLeaksNoModeQueryBytes`, `TestDaemonPickerEntersAltScreenAndRestoresMainBuffer`,
`TestInstallPickerCancelWritesNoConfig`) each carry a genuine positive control in the *same*
capture/measurement as their negative assertion, satisfying rule 84d1gfpywd — I read them
looking for the vacuous-guard shape the phase exists to close and did not find it. The
mutation log's honest reporting of family (d)'s non-reproduction is a real strength, not a
gap to pad over.

The defects found are all in the harness's edges, not its core assertions: a misleading
comment left behind by a one-time (not durably enforced) grep-dodge, a second independent
instance of the same fixed-wait-then-assert shape the phase's own doc comment claims doesn't
exist, an empirically-confirmed diagnostic-masking bug in the Taskfile's `jq` pipeline under a
plausible toolchain-resolution failure, an unbounded subprocess `Wait()` that breaks with the
repo's own established never-hang convention, and a hash-construction boundary ambiguity in
the config-tree oracle that is not currently exploitable but is undocumented as a reuse risk.
None of these are vacuous gates and none allow a defect to ship silently — they degrade
diagnostics, maintainability, or robustness at the margins.

## Assessment of the three orchestrator-flagged findings

**1. `test/tmux/capture.go:113`** — Confirmed as described: `<-time.After(stabilityPollInterval)`
outside a `select` is a plain blocking pause, functionally identical to `time.Sleep(d)`, and
the trailing comment ("interval pacing via a channel wait, no blocking pause call") is false —
it *is* a blocking pause call. Worse than the orchestrator's framing suggests: I checked
whether the `rg -o 'time\.Sleep' test/tmux | wc -l == 0` gate this respelling exists to satisfy
is wired into anything durable (Taskfile.yml, `ci.yml`, a Go test) — it is not; it was a
one-time `08-01-PLAN.md` verify command, run once during plan execution, never re-checked
again. So today this comment protects nothing at all going forward, while actively lying
about what the line does. See **WR-01**.

I also found a second, unflagged instance of the same underlying shape: `frame_stability_test.go`'s
idle-stability loop performs its own `<-time.After(stabilityPollInterval)` immediately before a
single `capturePane` + direct equality check, entirely bypassing `pollUntilStable`. This
directly contradicts `capture.go`'s own doc-comment claim that `pollUntilStable` is "the
package's ONLY wait primitive" and that "nothing in this package may pause for a fixed
duration and then assert without going through this poll." The design is reasonable (sample
N times after convergence to prove *idleness*, which is a different property than *converging*),
but the stated invariant is factually false about the shipped code. See **WR-02**.

On the orchestrator's specific question — is "every content assertion routes through
`pollUntilStable`" the right replacement gate? Partially: it's right for every test file
(`daemon_empty_test.go`, `daemon_picker_test.go`, `install_cancel_test.go` all do route
through it), but it would incorrectly flag `frame_stability_test.go`'s loop, which is a
deliberate, bounded, N-count sampling loop — not the "wait once and hope" shape TTY-02 bans.
The real property is narrower: *any fixed-duration pause must be part of a bounded,
multi-sample loop that can fail (convergence or stability), never a single wait-then-assume-ready*.
That property is not mechanically grep-able across arbitrary future code; recommend dropping
the literal-substring gate (it protects nothing durable today anyway) and instead landing a
committed, always-running (no build tag) Go test — following the repo's own
`taskfile_shape_test.go` convention — that scans `test/tmux/*.go` for `time.Sleep(` and fails
loudly, since that at least gives durable, mechanical enforcement of the literal-token
property the plan actually verified, even if it can't enforce the deeper semantic one.

**2. `test/tmux/confighash.go`** — Read the current doc comment in full. "rather than shelling
out to a coreutils checksum binary" still communicates the load-bearing property (pure Go, no
subprocess, portable to darwin) exactly as clearly as naming `sha256sum` would have. No
clarity was lost. Not a defect.

**3. `.github/workflows/ci.yml`** — Read the current job header comment in full. "Nothing on
this job or any of its steps can turn a failure soft or make a step conditional" is, if
anything, clearer than the terse "No continue-on-error and no if:" it replaced, since it
states the *property* rather than naming two YAML keys a reader has to already know the
significance of. No clarity was lost. Not a defect.

Both #2 and #3 are instances of the same root cause as #1 — a whole-file literal-substring
count used as a proxy for a structural property, tripped by prose rather than the actual
construct it polices — but in both cases the rewording itself lost nothing, and neither gate
is durably wired into CI either (both were one-time plan-execution verify commands). If this
gate shape is kept for future phases, scope it to the actual YAML key position
(`^\s*continue-on-error:`) or an AST/line-anchor check, per the orchestrator's own suggestion,
rather than a whole-file substring count that prose can trip.

**A fourth instance, different in kind:** see **WR-03** below — not a prose-dodge, but a place
where the Taskfile's own documented invariant (D-03: "always prints both an executed and a
skipped count") is empirically broken under a specific, plausible failure mode.

## Warnings

### WR-01: `pollUntilStable`'s interval-wait comment is false, and the gate it exists to satisfy is not durably enforced

**File:** `test/tmux/capture.go:113`
**Issue:** `<-time.After(stabilityPollInterval) // interval pacing via a channel wait, no blocking pause call` — the code on this line blocks the goroutine for `stabilityPollInterval` before proceeding, identically to `time.Sleep(stabilityPollInterval)` in this non-`select` context. The comment asserts the opposite. The `rg -o 'time\.Sleep' test/tmux | wc -l == 0` check this line's spelling exists to satisfy was run once, during `08-01-PLAN.md`'s execution, and is not present anywhere in `Taskfile.yml`, `ci.yml`, or any committed Go test (verified: `rg -n 'time\.Sleep|time\.After' Taskfile.yml .github/workflows/ci.yml internal/upgrade/taskfile_shape_test.go` returns nothing) — so today the comment misleads a future reader with zero compensating enforcement.
**Failure scenario:** A future contributor reads the comment, believes some mechanism prevents a literal `time.Sleep` from ever landing in this package, and does not think to check — then adds a `time.Sleep(2*time.Second)` directly in a new test's assertion path. Nothing catches it; the misleading comment actively discouraged the check that would have.
**Fix:**
```go
// interval pacing: a plain blocking pause, equivalent to time.Sleep —
// spelled as <-time.After to avoid the literal substring "time.Sleep"
// (see this package's own history), not because it behaves differently.
<-time.After(stabilityPollInterval)
```
Additionally, land a committed, always-running (no `tmux` build tag) Go test under `internal/` or `test/tmux` itself that scans `test/tmux/*.go` for the literal `time.Sleep(` and fails — this gives durable enforcement of the one property the original one-time grep actually checked, instead of leaving that property permanently unverified after the plan that introduced it closed.

### WR-02: `frame_stability_test.go`'s idle loop bypasses `pollUntilStable`, contradicting the package's stated "only wait primitive" invariant

**File:** `test/tmux/frame_stability_test.go:52-58`
**Issue:** The loop performs `<-time.After(stabilityPollInterval)` then a single `capturePane(t, session)` call compared directly against `settled`, entirely outside `pollUntilStable`. `capture.go`'s doc comment states `pollUntilStable` is "the package's ONLY wait primitive" and "nothing in this package may pause for a fixed duration and then assert without going through this poll" — both are false as written, since this loop does exactly that (deliberately, for a legitimate different purpose: sampling stability across N further captures after convergence, not seeking a new convergence point).
**Failure scenario:** A future reader trusts `capture.go`'s stated invariant at face value when reviewing a new test, assumes any content assertion in this package is safe from the classic "sleep-then-assert" flake shape because "the package's only wait primitive" handles it — and doesn't notice that `frame_stability_test.go`'s pattern is a second, independent implementation that a new test could copy without the bounded-and-fails-loudly discipline `pollUntilStable` itself carries (e.g., a future variant of this loop that doesn't bound its interval count, or that silently ignores a divergence instead of `t.Fatalf`-ing).
**Fix:** Narrow the claim in `capture.go`'s doc comment, e.g.: "`pollUntilStable` is the package's only *convergence* primitive. `frame_stability_test.go`'s own idle-sampling loop is a deliberate, distinct, bounded pattern for a different property (idle stability, not first-convergence) — see that file's doc comment; it is not a violation of this invariant, but it is not covered by it either." This costs nothing and prevents the doc comment from over-promising.

### WR-03: `task test:tmux`'s `jq` pipeline crashes with an opaque error — and skips the target's own promised diagnostics — when `go test -json` fails before producing valid JSON

**File:** `Taskfile.yml:239-243` (the `test:tmux` target's inline shell)
**Issue:** `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -json ./test/tmux/... 2>&1 | tee "${JSON_OUT}" || STATUS=$?` merges stderr into the same file that `jq -s` subsequently slurps as a JSON stream. When `go test` fails before it starts producing test2json-wrapped output — the concrete, reproduced case is a `GOTOOLCHAIN` resolution failure — the failure line is plain text, not JSON. `jq -s` then aborts with a parse error, and under this script's `set -euo pipefail`, the whole target dies right there: before the `echo "test:tmux: executed=..."` line, and before the target's own designed `::error::test:tmux: go test exited ${STATUS}...` diagnostic. This directly contradicts the target's own documented D-03 contract: "Always prints both an executed and a skipped count."
**Failure scenario (empirically reproduced, not hypothetical):**
```
$ cd /tmp/goprobe && set -euo pipefail; JSON_OUT="$(mktemp)"; STATUS=0
GOPROXY=off GOTOOLCHAIN=go1.99.9 go test -json ./... 2>&1 | tee "$JSON_OUT" || STATUS=$?
echo "STATUS=$STATUS"
jq -s '[.[] | select(.Action=="pass")] | length' "$JSON_OUT"
```
Output:
```
go: download go1.99.9 for darwin/arm64: toolchain not available
STATUS=1
jq: parse error: Invalid numeric literal at line 1, column 3
```
`STATUS` is captured correctly (1), but the script never reaches the code that would report it clearly — `jq`'s own nonzero exit under `set -e` terminates the target first, with a message that gives a CI triager zero indication this was a toolchain problem rather than a jq/tooling bug. The job still fails overall (this is not a silent pass), but the diagnostic the target explicitly promises to always print never appears.
**Fix:** Don't merge stderr into the file `jq` parses. Capture stdout and stderr separately:
```bash
GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -json ./test/tmux/... >"${JSON_OUT}" 2>"${ERR_OUT}" || STATUS=$?
...
if [ "${STATUS}" -ne 0 ]; then
  echo "::error::test:tmux: go test exited ${STATUS}"
  cat "${ERR_OUT}" >&2
  exit 1
fi
```
so `jq -s` only ever sees valid JSON, and a pre-test failure surfaces via the target's own clear message plus the real stderr, not a jq parse error.

### WR-04: Seeded daemon subprocess teardown has no bounded timeout, unlike the repo's own established never-hang convention

**File:** `test/tmux/daemon_seed.go:49-52`
**Issue:**
```go
t.Cleanup(func() {
	_ = daemonCmd.Process.Signal(syscall.SIGTERM)
	_ = daemonCmd.Wait()
})
```
`Wait()` has no bound. `08-CONTEXT.md`'s own canonical refs name `test/integration/piped_never_hang_test.go` as "the bounded never-hang pattern" this harness inherits — that file races a blocking call against `time.After` in a `select` specifically so a hang fails the test loudly instead of blocking `go test` itself. This cleanup does not follow that pattern.
**Failure scenario:** If the daemon subprocess does not exit promptly after SIGTERM (a signal-handling regression, a blocked syscall, a deadlock reached only under CI's particular timing), `Wait()` blocks indefinitely. The test binary is eventually rescued only by Go's own default test timeout (10 minutes), which then reports a generic "panic: test timed out" with a full goroutine dump — burning CI time and producing a far less actionable diagnostic than the rest of this harness aims for (compare `pollUntilStable`'s own named, bounded, `t.Fatalf`-with-both-captures failure).
**Fix:**
```go
t.Cleanup(func() {
	_ = daemonCmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- daemonCmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("seedRunningDaemon cleanup: daemon pid %d did not exit within 5s of SIGTERM", daemonCmd.Process.Pid)
		_ = daemonCmd.Process.Kill()
	}
})
```

### WR-05: `hashConfigTree`'s path/content concatenation has no separator, an undocumented boundary-ambiguity risk for future reuse

**File:** `test/tmux/confighash.go:62-74`
**Issue:**
```go
for _, rel := range relPaths {
	io.WriteString(h, rel)
	f, _ := os.Open(...)
	io.Copy(h, f)
	...
}
```
Each file's relative path is written directly into the running hash immediately followed by its bytes, with no delimiter between path and content or between successive file entries. This is a classic hash-construction ambiguity: `Write("foo") + Write("bar")` produces the same hash input as `Write("foob") + Write("ar")` — two structurally different trees can, in principle, hash identically.
**Failure scenario:** Not currently exploitable in `TestInstallPickerCancelWritesNoConfig` — the "before" baseline there is always the hash of a genuinely empty tree (the fixed SHA-256-of-empty-input constant `e3b0c442...`), and producing any nonempty write that reconstructs that exact digest requires a SHA-256 preimage, which is computationally infeasible. But the doc comment already flags one reuse caveat (`LocationLocal`/`cursorConfigPath`) without flagging this one, and a future test comparing two *nonempty* tree states (e.g., "does `install --agent X` touch only X's files, leaving Y's alone") would inherit a real, in-principle collision risk with no warning.
**Fix:** Add a delimiter that cannot appear in a valid relative path, e.g.:
```go
io.WriteString(h, rel)
h.Write([]byte{0}) // NUL separator: disambiguates path/content and file/file boundaries
```

## Info

### IN-01: `<-time.After(d)` used as a bare blocking pause instead of idiomatic `time.Sleep(d)`

**File:** `test/tmux/capture.go:113`, `test/tmux/frame_stability_test.go:53`
**Issue:** Both call sites use `<-time.After(d)` outside a `select`. `time.After` earns its keep inside a `select` racing another channel (exactly how `test/integration/piped_never_hang_test.go` uses it); used bare, it is a strictly worse spelling of `time.Sleep(d)` (extra timer allocation, no functional difference here since the channel is drained immediately). Both instances trace to the same literal-substring-grep-dodging origin as WR-01.
**Fix:** Once WR-01's durable-enforcement question is resolved, prefer `time.Sleep(d)` at both sites for its clearer idiom, or keep `time.After` only if a `select` is genuinely anticipated later.

### IN-02: `GOTOOLCHAIN=go1.26.6` is hardcoded and duplicated with no cross-check against `go.mod`

**File:** `Taskfile.yml:239`, `test/tmux/main_test.go:28`
**Issue:** `Taskfile.yml:239` is the only place in the entire file that pins a toolchain version this way (`rg -n "GOTOOLCHAIN=go1" Taskfile.yml` returns exactly this one line); the same literal is repeated in a doc comment in `main_test.go`. `go.mod` currently pins `go 1.26.6`, so the two agree today, but nothing checks that they continue to.
**Fix:** Low priority given this exists specifically to route around a documented ambient-toolchain landmine rather than for correctness — but if `go.mod`'s pin is ever bumped, both copies need a manual, unenforced update. Consider deriving the value at task-run time (e.g., `GOTOOLCHAIN=go$(go mod edit -json | jq -r .Go)`), or at minimum a one-line comment at both sites cross-referencing the other.

### IN-03: `filepath.Walk` used in one diagnostic-only path where the package otherwise uses `filepath.WalkDir`

**File:** `test/tmux/install_cancel_test.go:81`
**Issue:** The failure-message path lists files via `filepath.Walk`, while `confighash.go`'s `hashConfigTree` (the same package's primary tree-walking helper) uses the more efficient `filepath.WalkDir`. No functional impact — this code only runs once the test has already failed and is building a diagnostic message — but it's a minor style inconsistency within the same package.
**Fix:** Use `filepath.WalkDir` for consistency, or reuse a shared listing helper.

### IN-04: Two of the three orchestrator-flagged rewordings cost no real clarity

See "Assessment of the three orchestrator-flagged findings" above for `confighash.go` and `ci.yml` — recorded here for completeness since the review contract asks for a file:line-anchored entry per assessed item, but no code change is recommended for either.

---

_Reviewed: 2026-09-10T17:07:18Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
