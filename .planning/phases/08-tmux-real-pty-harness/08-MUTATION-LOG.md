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
