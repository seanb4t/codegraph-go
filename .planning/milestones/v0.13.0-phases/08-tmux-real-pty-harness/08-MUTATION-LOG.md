# 08-MUTATION-LOG — tmux Real-PTY Harness

**Phase:** 08-tmux-real-pty-harness
**Date:** 2026-09-10
**Scope:** Four RED demonstrations, one per assertion class this phase's harness introduced (TTY-03 through TTY-06). D-06 deliberately extends coverage past the two families the ROADMAP success criteria name (family (a)/TTY-03 and family (b)/TTY-04): TTY-05 and TTY-06 are new assertion classes (families (c) and (d)) that would otherwise ship having never been watched fail — exactly the untrusted-gate shape this milestone exists to close. Family (a)'s recipe was corrected by 08-RESEARCH.md's empirical work before this plan was written (see Pitfall 1 below) — the single-location mutation D-06 originally named does not reach the historical defect.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a destructive blind checkout of someone else's in-flight work. Every family entry below records this gate's result at the point it was checked.

One addition this log needs that Phase 7's did not: two families touch `internal/cli/tui/daemonpicker.go` (families (a) and (b)) and two touch `internal/cli/tui/agentpicker.go` (families (c) and (d)). Mutations are applied and reverted **strictly one at a time, never overlapped** — a `git checkout --` on a shared file while a sibling mutation is still live would discard that sibling and make the byte-clean proof meaningless. Each family below re-checks the cleanliness gate for its shared file fresh, rather than inheriting the previous family's result.

---

## Family (a) — TTY-03: the G-07-1 DECRQM leak

**Test name:** `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` (`test/tmux/daemon_empty_test.go`).

**Pre-mutation gate:**
- `git diff --quiet -- internal/cli/daemon.go` → exit 0 (clean).
- `git diff --quiet -- internal/cli/tui/daemonpicker.go` → exit 0 (clean).

**Mutation applied — BOTH guards, together, in the same window.** This is the corrected recipe and the single most important thing in this log (08-RESEARCH.md Pitfall 1, 08-CONTEXT.md D-06 CORRECTED note):

1. `internal/cli/daemon.go`'s caller-side picker guard: dropped the `&& len(records) > 0` conjunct, so the condition gates on interactivity alone.
   ```diff
   -			if interactiveAllowed(cmd) && len(records) > 0 {
   +			if interactiveAllowed(cmd) {
   ```
2. `internal/cli/tui/daemonpicker.go`'s `RunDaemonPicker` defense-in-depth early return: changed the empty-set comparison to one that can never hold — a length is never negative, so the early return becomes unreachable — keeping the guard's body intact so the diff stays single-token.
   ```diff
   -	if len(records) == 0 {
   +	if len(records) < 0 {
   ```

**Why both are needed.** `RunDaemonPicker` carries its own independent, defense-in-depth empty-registry short-circuit that returns before ever constructing a `tea.Program` — its own doc comment states it exists "so this function can never be the source of that leak regardless of caller." Mutating only the caller-side guard in `daemon.go` was verified empirically (08-RESEARCH.md) to still print exactly `no running daemons` with no leak: `RunDaemonPicker` silently absorbed the caller's mistake. Reaching the historical G-07-1 behavior requires defeating both guards at once. This is worth stating plainly: the product is defended in two places, which is a real property of the shipped code, not an artifact of this demonstration.

**Confirmed applied** (grep, before running):
```
$ rg -n 'if interactiveAllowed\(cmd\) \{' internal/cli/daemon.go
82:			if interactiveAllowed(cmd) {
$ rg -n 'if len\(records\) < 0 \{' internal/cli/tui/daemonpicker.go
318:	if len(records) < 0 {
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -v -run TestDaemonEmptyRegistryLeaksNoModeQueryBytes ./test/tmux/...`, `CODEGRAPH_TEST_BIN` unset, exit 1):**

```
=== RUN   TestDaemonEmptyRegistryLeaksNoModeQueryBytes
    daemon_empty_test.go:50: TTY-03: stabilized capture leaks DECRQM mode-query response marker "^[[?2026;2$y" — empty-registry escape hygiene violated:
        env HOME=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonEmptyRegistryLeaksNoModeQueryByt
        es3591483503/001 USERPROFILE=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonEmptyRegistr
        yLeaksNoModeQueryBytes3591483503/001 /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/codegraph-tmux
        -1352458759/codegraph daemon
        sean@denver tmux % env HOME=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonEmptyRegistry
        LeaksNoModeQueryBytes3591483503/001 USERPROFILE=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/Tes
        tDaemonEmptyRegistryLeaksNoModeQueryBytes3591483503/001 /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000
        gn/T/codegraph-tmux-1352458759/codegraph daemon
        no running daemons
        ^[[?2026;2$y^[[?2027;0$y\033[1;7m%\033[0m
        sean@denver tmux %
        /2027;0$y_




--- FAIL: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (2.89s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/tmux	6.602s
FAIL
```

The failing assertion names the mode-query response marker found (`^[[?2026;2$y`) and the surrounding captured pane text — the exact historical G-07-1 shape: `no running daemons` prints correctly (the caller-side guard defeated cleanly falls through to the picker path, then `RunDaemonPicker`'s own defeated guard falls through to constructing a `tea.Program` that immediately quits on the empty set), and the leaked DECRQM capability-probe bytes (`^[[?2026;2$y^[[?2027;0$y`) surface on the next reader, the shell's own line editor, matching 08-RESEARCH.md's Code Examples transcript.

**Revert:**
```
$ git checkout -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go
```

**Post-revert gate:**
- `git diff --quiet -- internal/cli/daemon.go` → exit 0 (clean).
- `git diff --quiet -- internal/cli/tui/daemonpicker.go` → exit 0 (clean).

**Green re-run (pasted verbatim, same command, exit 0):**
```
--- PASS: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (4.69s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	9.869s
```

---

## Family (b) — TTY-04: the G-07-2 alt-screen regression

**Test name:** `TestDaemonPickerEntersAltScreenAndRestoresMainBuffer` (`test/tmux/daemon_picker_test.go`).

**Pre-mutation gate (freshly re-checked here, after family (a)'s revert, not assumed from it):**
- `git diff --quiet -- internal/cli/tui/daemonpicker.go` → exit 0 (clean).

**Mutation applied.** `daemonPickerModel.View()`'s alternate-screen field disabled:
```diff
-	v.AltScreen = true
+	v.AltScreen = false
```

**Confirmed applied** (grep, before running):
```
$ rg -n 'v.AltScreen = false' internal/cli/tui/daemonpicker.go
241:	v.AltScreen = false
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -v -run TestDaemonPickerEntersAltScreenAndRestoresMainBuffer ./test/tmux/...`, `CODEGRAPH_TEST_BIN` unset, exit 1):**

```
    daemon_picker_test.go:40: TTY-04: alternateOn is false while the daemon picker should be open:
        env HOME=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonPickerEntersAltScreenAndRestores
        MainBuffer3062078927/001 USERPROFILE=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonPick
        erEntersAltScreenAndRestoresMainBuffer3062078927/001 /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/
        T/codegraph-tmux-1744152664/codegraph daemon
        sean@denver tmux % env HOME=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestDaemonPickerEntersA
        ltScreenAndRestoresMainBuffer3062078927/001 USERPROFILE=/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000
        gn/T/TestDaemonPickerEntersAltScreenAndRestoresMainBuffer3062078927/001 /var/folders/_b/3hyf5qvs62q0
        wh2vyh856z580000gn/T/codegraph-tmux-1744152664/codegraph daemon

        > 002 (pid 63961, up 2s)




















        enter: stop selected  a: stop all  q/esc: cancel
--- FAIL: TestDaemonPickerEntersAltScreenAndRestoresMainBuffer (4.44s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/tmux	7.953s
FAIL
```

**Which of D-13's two signals fired.** The `alternateOn` probe (`#{alternate_on}`) reported `false` while the picker was open — this is the loud, unambiguous failure, and it is what actually failed the test, at the first assertion checked (`daemon_picker_test.go:40`), before the content assertion was ever reached.

The content assertion's behavior under this mutation is worth recording alongside it, since it did not fail here and that is the practical value of D-13's two-signal design: the pasted capture above shows the seeded record's row (`> 002 (pid 63961, up 2s)`) and the help footer (`enter: stop selected  a: stop all  q/esc: cancel`) rendered plainly and legibly in the plain (non-`-a`) capture, exactly as they would be with `AltScreen` correctly `true` — because with the mutation applied the picker now renders **inline**, and a plain capture-pane sees whatever is currently displayed regardless of which screen buffer it lives in. A content-only assertion (`strings.Contains(capture, "Running daemons")`) would have passed against this exact mutation, proving nothing wrong. The `alternateOn`/`#{alternate_on}` signal is the one that actually discriminates "did the picker enter the alternate screen," and it is the one that failed here — confirming D-13's design choice to gate content assertions on the alt-mode signal rather than relying on content alone.

**Revert:**
```
$ git checkout -- internal/cli/tui/daemonpicker.go
```

**Post-revert gate:**
- `git diff --quiet -- internal/cli/tui/daemonpicker.go` → exit 0 (clean).

**Green re-run (pasted verbatim, same command, exit 0):**
```
--- PASS: TestDaemonPickerEntersAltScreenAndRestoresMainBuffer (5.50s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	8.958s
```

---

## Family (c) — TTY-05: a cancel that writes config

**Test name:** `TestInstallPickerCancelWritesNoConfig` (`test/tmux/install_cancel_test.go`).

**Pre-mutation gate:**
- `git diff --quiet -- internal/cli/tui/agentpicker.go` → exit 0 (clean).

**Mutation applied.** The cancel branch of `Update`'s key switch set to the same confirmed flag the enter branch sets:
```diff
 		case "q", "esc", "ctrl+c":
-			m.confirmed = false
+			m.confirmed = true
 			return m, tea.Quit
```

That single token makes `resolvedTargets` return the checked set on cancel instead of `nil`, `install`'s `RunE` then hands that set to `printAgentResults`, and real config files land under the throwaway `$HOME` — exactly what TTY-05's before/after tree-hash exists to catch.

**Confirmed applied** (diff, before running):
```
$ git diff -- internal/cli/tui/agentpicker.go
--- a/internal/cli/tui/agentpicker.go
+++ b/internal/cli/tui/agentpicker.go
@@ -129,7 +129,7 @@ func (m agentPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
 		case "q", "esc", "ctrl+c":
-			m.confirmed = false
+			m.confirmed = true
 			return m, tea.Quit
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -v -run TestInstallPickerCancelWritesNoConfig ./test/tmux/...`, `CODEGRAPH_TEST_BIN` unset, exit 1):**

```
=== RUN   TestInstallPickerCancelWritesNoConfig
    install_cancel_test.go:87: TTY-05: config-tree hash changed after cancel — before=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 after=c6de07d31c3a1ff9123053a7d543c7b05fa2a3519bec464cf68214533a755e5c; paths present under /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestInstallPickerCancelWritesNoConfig3785278073/001: [/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestInstallPickerCancelWritesNoConfig3785278073/001/.gemini/GEMINI.md /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestInstallPickerCancelWritesNoConfig3785278073/001/.gemini/config/.migrated /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestInstallPickerCancelWritesNoConfig3785278073/001/.gemini/config/mcp_config.json /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestInstallPickerCancelWritesNoConfig3785278073/001/.gemini/settings.json]
--- FAIL: TestInstallPickerCancelWritesNoConfig (14.70s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/tmux	18.234s
FAIL
```

Both tree digests differ (`before=e3b0c44298...` — the SHA-256 of an empty tree — vs `after=c6de07d3...`), and four real config file paths under the throwaway `$HOME`'s `.gemini/` directory are named: `GEMINI.md`, `config/.migrated`, `config/mcp_config.json`, `settings.json`. This is a genuinely-mutated cancel writing real files, not a hash artifact.

**Why the `space` toggle before each cancel is load-bearing.** On a throwaway `$HOME`, no agent is detected, so nothing starts checked. Without the `space` keypress the test sends before each cancel path, `resolvedTargets` — even with the cancel branch wrongly resolving to `confirmed=true` — would still map an all-unchecked `checked` map to an empty target slice, `printAgentResults` would iterate zero targets, and the mutation would produce a false GREEN: the demonstration would prove the opposite of what it claims. The toggle is what gives the wrongly-confirming cancel path something real to be wrong about.

**Revert:**
```
$ git checkout -- internal/cli/tui/agentpicker.go
```

**Post-revert gate:**
- `git diff --quiet -- internal/cli/tui/agentpicker.go` → exit 0 (clean).

**Green re-run (pasted verbatim, same command, exit 0):**
```
--- PASS: TestInstallPickerCancelWritesNoConfig (11.89s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	17.036s
```

---

## Family (d) — TTY-06: an inline render that flickers

**Test name:** `TestInstallPickerFrameStableWhileIdle` (`test/tmux/frame_stability_test.go`).

**Pre-mutation gate (freshly re-checked here, after family (c)'s revert, not assumed from it):**
- `git diff --quiet -- internal/cli/tui/agentpicker.go` → exit 0 (clean).

**Mutation applied.** The checkbox picker's `View()` alternate-screen field disabled — the same token family (b) mutated in the other picker:
```diff
 	v.AltScreen = false
 	return v
 }
```
(i.e. `v.AltScreen = true` → `v.AltScreen = false`, at `internal/cli/tui/agentpicker.go:147`.)

**Confirmed applied** (diff, before running, and re-confirmed immediately before the second run below):
```
$ git diff -- internal/cli/tui/agentpicker.go
--- a/internal/cli/tui/agentpicker.go
+++ b/internal/cli/tui/agentpicker.go
@@ -144,7 +144,7 @@ func (m agentPickerModel) View() tea.View {
 	// Alt-screen (bubbletea v2 per-View field) — see daemonpicker.go's
 	// View() for the full rationale (07-UAT test 1): prevents the inline
 	// full-height render from scrolling/flickering the main buffer.
-	v.AltScreen = true
+	v.AltScreen = false
 	return v
 }
```

**Why the 12-row pane is load-bearing regardless of this outcome.** `frameStabilityRows` (12) is deliberately shorter than the package's default 30-row session height specifically so the 8-agent list plus its help footer cannot fit inline — a picker rendering inline in a 12-row pane genuinely overflows during its settling transient, where a 30-row pane might not. That geometry is what makes the *settling-phase* divergence between alt-screen and inline rendering observable at all; it is not what this specific idle-stability assertion measures (see below), but it remains the correct, load-bearing choice for the test's own stated purpose (TTY-06: does an already-settled picker stay stable) and for any future assertion that measures the settling transient itself.

**Observed result: the mutation did NOT produce a failure, run twice, back to back, both against a freshly rebuilt binary (`CODEGRAPH_TEST_BIN` unset, so `TestMain` rebuilds from the mutated working tree every run — confirmed, not assumed).** Per this plan's own instruction, this is reported as observed rather than written up as a RED demonstration that did not happen.

**Run 1 (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -v -run TestInstallPickerFrameStableWhileIdle ./test/tmux/...`, exit 0):**
```
    frame_stability_test.go:61: TTY-06: frame-stable across N=5 captures (pane 100x12, install picker idle)
--- PASS: TestInstallPickerFrameStableWhileIdle (9.31s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	14.043s
```

**Run 2, mutation re-confirmed still applied via `git diff` immediately before (pasted verbatim, same command, exit 0):**
```
    frame_stability_test.go:61: TTY-06: frame-stable across N=5 captures (pane 100x12, install picker idle)
--- PASS: TestInstallPickerFrameStableWhileIdle (7.88s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	11.355s
```

**What this implies about the assertion's discriminating power.** `TestInstallPickerFrameStableWhileIdle` converges once via `pollUntilStable` to a settled first frame, then asserts byte-identical captures across `frameStabilityCaptures` (5) further polls with **no further input sent**. `agentPickerModel.Init()` returns no `tea.Cmd` — there is no periodic tick, and bubbletea only re-renders `View()` in response to a `tea.Msg` (a keypress, a resize, a command's result). With no input arriving after the initial `Enter`, the Program simply does not re-render at all once settled, regardless of whether `AltScreen` is `true` or `false` — there is nothing to be unstable. The scrolling/flicker behaviour the `AltScreen` field's doc comment describes ("a full-height list that doesn't fit the remaining space scrolls the main buffer every frame") is a property of the *transient settling phase* — the sequence of renders between the Program starting and reaching its first stable frame — not of a genuinely idle, already-settled Program. `pollUntilStable` itself absorbs that transient before the stability loop's five captures ever begin, so this test's design (converge first, then assert idle stability) structurally cannot observe a defect that only manifests during convergence.

This is a real, reproducible finding about this specific assertion's discriminating power against this specific mutation, not a flake: the mutation was confirmed applied to the tracked file before each of the two runs, `TestMain` rebuilt the binary from the mutated working tree both times (verified via `CODEGRAPH_TEST_BIN` being unset), and both runs produced the identical PASS outcome. TTY-06 as currently specified — "an idle install picker... holds byte-identical across 5 further captures after its first settled frame" — is a real, meaningful, non-vacuous assertion about idle stability (proven non-trivial by D-14's own stability-poll design elsewhere in this harness), but the single-token `v.AltScreen = false` mutation D-06 named for family (d) does not reach it, because that mutation's effect is confined to the settling transient the test deliberately polls past before measurement begins.

**Revert:**
```
$ git checkout -- internal/cli/tui/agentpicker.go
```

**Post-revert gate:**
- `git diff --quiet -- internal/cli/tui/agentpicker.go` → exit 0 (clean).

**Green re-run (pasted verbatim, same command, exit 0 — identical output to both RED-attempt runs above, since the mutation was never observable):**
```
    frame_stability_test.go:61: TTY-06: frame-stable across N=5 captures (pane 100x12, install picker idle)
--- PASS: TestInstallPickerFrameStableWhileIdle (7.90s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	11.306s
```

---

## Closing

### Phase-wide byte-clean proof

Per-family gates (above) prove each revert individually; this proves that no mutation from any family survived into the phase's final state, which is the property that actually matters for shipping.

**Porcelain status across the whole touched product tree (pasted verbatim):**
```
$ git status --porcelain -- internal/cli internal/cli/tui
$
```
Empty output — no modified, staged, or untracked file survives under `internal/cli`/`internal/cli/tui`.

**Diff against the phase's branch point, ranged form, across all three touched production files (pasted verbatim, exit 0):**
```
$ git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go
$ echo $?
0
```
This is the ranged form deliberately: a bare working-tree diff would report clean for a mutation that was accidentally *committed*, and this phase changes no product behaviour (08-CONTEXT.md), so any diff against the branch point is a surviving mutation. All three restored guard tokens (`len(records) > 0` in `daemon.go`; `len(records) == 0` and `v.AltScreen = true` in `daemonpicker.go`; `m.confirmed = false` and `v.AltScreen = true` in `agentpicker.go`) are present exactly once each, verified per-family above.

### Non-vacuity assertion

These four demonstrations establish that three of the four assertion classes this phase built — TTY-03, TTY-04, and TTY-05 — can actually fail, against the real historical defect behaviour or a real, newly-possible-but-never-exercised defect, observed through a real pty driving a really-built binary. That is the property this milestone exists to guarantee, and neither a checked-in bad-pane fixture nor a fault-injection switch would have established it (D-05): a fixture proves only that a matcher can match a frozen string, never that the harness observes a defect from a real terminal; a fault-injection switch ships a deliberately-broken code path that itself then needs a guard.

**Family (d) is the one demonstration in this log that did not reproduce a failure, and that fact is not minimized here.** The `v.AltScreen = false` mutation D-06 named for TTY-06 was confirmed applied to the tracked file, confirmed to rebuild into the binary under test (`CODEGRAPH_TEST_BIN` unset, verified), and run twice — both runs passed. The reason is structural, not a flake: `TestInstallPickerFrameStableWhileIdle` measures stability *after* `pollUntilStable` has already converged past the picker's settling transient, and the alt-screen-vs-inline scrolling difference this mutation targets is confined to exactly that transient — a Program that receives no further input after its first settled frame does not re-render at all, regardless of `AltScreen`'s value, so there is nothing for this specific idle-stability assertion to observe. TTY-06's assertion itself is real and non-vacuous (an idle picker genuinely could fail to hold a stable frame, and D-14's stability-poll design elsewhere in this harness proves that kind of instability is a real, catchable failure mode) — what is NOT established is that *this specific single-token mutation* is a defect TTY-06's *idle-stability* assertion can detect. That is a narrower, more honest claim than "TTY-06 has been watched fail," and this log states it as such rather than writing up family (d) as a success it was not.

**What none of these four demonstrations establish, even where they succeeded:** they do not prove the assertions catch every possible regression in these surfaces — each is one mutation, not an exhaustive fault model — and they do not prove the harness itself is free of flake. Both are out of scope for a mutation log and always will be.

**Declining a second guard (Phase 7 D-07 precedent, 07-CONTEXT.md).** Family (d)'s finding could be read as motivating a new test — one that observes the settling transient directly, closing the gap this log just described. That is deliberately **not** added here: per 07-CONTEXT.md D-07 ("why are you proposing tests for tests?"), a new assertion class is architectural scope this plan was not given, not a mutation-log deliverable, and the honest, recorded finding above is itself the correct output of a RED demonstration that did not reproduce — not a mandate to expand scope to force one. The finding is left as a candidate for a future phase or todo, not silently absorbed into this plan's file set.

### Family-to-requirement table

| Family | Requirement | Test function | Mutated file(s) | ROADMAP criterion |
|--------|-------------|----------------|------------------|--------------------|
| (a) | TTY-03 | `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` | `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go` | Criterion 3 (first half) — watched fail against the historical G-07-1 behaviour; **discharged** |
| (b) | TTY-04 | `TestDaemonPickerEntersAltScreenAndRestoresMainBuffer` | `internal/cli/tui/daemonpicker.go` | Criterion 3 (second half) — watched fail against the historical G-07-2 behaviour; **discharged** |
| (c) | TTY-05 | `TestInstallPickerCancelWritesNoConfig` | `internal/cli/tui/agentpicker.go` | Criterion 4 describes the functional behaviour only (glyphs/toggle/hash), verified by 08-02; D-06's *extended* watched-fail coverage beyond Criterion 4 — **discharged** |
| (d) | TTY-06 | `TestInstallPickerFrameStableWhileIdle` | `internal/cli/tui/agentpicker.go` | Criterion 4 describes the functional behaviour only (N-capture stability), verified by 08-02; D-06's *extended* watched-fail coverage beyond Criterion 4 — **NOT discharged via the specified mutation** (see Non-vacuity assertion above) |

Criterion 3 (TTY-03/TTY-04, the ROADMAP's own explicit "watched fail" ask) is fully discharged by families (a) and (b). Criterion 4 (TTY-05/TTY-06)'s functional behaviour is a separate property verified by 08-02-SUMMARY.md's own tests, not this plan's job to re-prove; D-06's *additional*, self-imposed ambition to also watch TTY-05 and TTY-06 fail is met for TTY-05 (family (c)) and not met for TTY-06 (family (d)) via the single mutation D-06 specified — recorded honestly above rather than papered over.
