# Phase 8: tmux Real-PTY Harness - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-09
**Phase:** 8-tmux-real-pty-harness
**Areas discussed:** Todo cross-reference, Anti-vacuity mechanism, RED demonstrations, Location and build tag, Frame stability and capture shape

---

## Todo Cross-Reference

| Option | Description | Selected |
|--------|-------------|----------|
| Neither | Both matches are off-domain: the graphstore archtest matched on generic keywords with no TTY surface, and brew-trust is explicitly Phase 12's | ✓ |
| graphstore archtest | Fold the go/packages per-package-load-error blindness in as extra guard hardening | |
| brew-trust docs | Pull the brew trust security-framing todo forward from Phase 12 | |

**User's choice:** Neither
**Notes:** `todo.match-phase 8` scored the graphstore item at 0.9 purely on the words test, review, query and phase. Both are recorded in CONTEXT.md → Deferred → Reviewed Todos so a later phase knows they were considered and why they were passed over.

---

## Anti-Vacuity Mechanism (TTY-01, TTY-07)

| Option | Description | Selected |
|--------|-------------|----------|
| Parse `go test -json` | Task target counts test-granularity pass events; the only option that survives a misspelled build tag, since no in-test mechanism runs when zero test files compile | ✓ |
| Env var flips skip to fatal | `CODEGRAPH_TMUX_REQUIRED` makes a missing tmux fatal in CI; simpler to read but blind to a build-tag typo | |
| Both mechanisms | Belt and braces; Phase 7's D-07 precedent argues against a second guard when one suffices | |

**User's choice:** Parse `go test -json`
**Notes:** The deciding argument was the build-tag typo case. `go test -tags tmuxx ./test/tmux/...` compiles zero test files and exits 0, so any guard living inside the test code cannot fire.

### Follow-up: floor or exact count?

| Option | Description | Selected |
|--------|-------------|----------|
| Committed expected count | Assert executed count equals a committed constant; a single silently-skipping case fails the job | ✓ |
| Assert the executed set | Compare sorted test names against a committed list; names the missing case but breaks on renames | |
| Floor of zero only | Simple non-zero check; leaves a partially-skipping suite green | |

**User's choice:** Committed expected count
**Notes:** A floor is satisfied by one case running while eleven skip, which is the same vacuity shape one level up. Echoes the v0.12.0 Phase 5 convergence lesson where a floor-counting guard absorbed a missing call site.

### Follow-up: local skip path

| Option | Description | Selected |
|--------|-------------|----------|
| One target, CI strictness via env | Single `test:tmux` always prints executed and skipped counts; enforces equality only under CI | ✓ |
| Two separate targets | `test:tmux` lenient, `test:tmux:ci` strict; explicit but two bodies that can drift | |

**User's choice:** One target, CI strictness via env
**Notes:** Keeps the Taskfile as the single definition of the job body, which `TestWorkflowRunBodiesInvokeTask` already enforces.

---

## RED Demonstrations (TTY-03 … TTY-06)

| Option | Description | Selected |
|--------|-------------|----------|
| Mutation log, Phase 7 shape | One-token mutation, watched fail, pasted transcript, byte-clean revert | ✓ |
| Fault-injection switch in product code | Repeatable forever, but ships a deliberately-broken path that itself needs a guard | |
| Captured bad-pane fixture | Cheap and hermetic, but tests the matcher against a frozen string rather than a real pty | |

**User's choice:** Mutation log, Phase 7 shape
**Notes:** Both historical fixes survive as single-token edits, which is what makes this cheap: `&& len(records) > 0` in `internal/cli/daemon.go:82` and `v.AltScreen = true` in `internal/cli/tui/daemonpicker.go:241`.

### Follow-up: how many families?

| Option | Description | Selected |
|--------|-------------|----------|
| All four, TTY-03 … TTY-06 | Every new assertion class watched failing once | ✓ |
| Only the two the criteria name | TTY-03 and TTY-04 only; TTY-05 and TTY-06 rely on their own positive assertions | |
| Three, skipping the flicker proxy | Omit TTY-06 on the grounds that making frames differ proves little about catching real flicker | |

**User's choice:** All four
**Notes:** Chosen over the narrower reading of the success criteria, which name a watched-fail only for TTY-03 and TTY-04. The reasoning was that TTY-05 and TTY-06 would otherwise ship never having been seen fail.

---

## Location, Build Tag, and Binary Provenance (TTY-01)

| Option | Description | Selected |
|--------|-------------|----------|
| `test/tmux/` | Follows the top-level convention that exists today alongside `test/integration/` and `test/wireoracle/` | ✓ |
| `internal/tmuxtest/` | What the milestone research recommends, on the stale premise that no top-level test directory exists | |

**User's choice:** `test/tmux/`
**Notes:** `.planning/research/ARCHITECTURE.md` §5 is factually stale on this point. Recorded in CONTEXT.md as an explicit override so the planner does not follow the research over the decision.

| Option | Description | Selected |
|--------|-------------|----------|
| `tmux` | Names the external dependency that gates the suite | ✓ |
| `ptye2e` | Names the capability, leaving room for a non-tmux pty harness later | |
| `e2e` | Broadest; risks becoming a bucket where one missing tool disables unrelated suites | |

**User's choice:** `tmux`
**Notes:** This is the repo's first feature build tag, so the choice sets the convention.

| Option | Description | Selected |
|--------|-------------|----------|
| Own resolver, same contract | ~30 duplicated lines, touches no working harness, and forces a local build for mutations | ✓ |
| Extract a shared resolver | One contract in one place, but refactors a harness a required check depends on | |
| Always build locally | Simplest, but loses the ability to point the suite at a notarized release binary | |

**User's choice:** Own resolver, same contract
**Notes:** The no-silent-fallback rule must be restated in the new resolver's own doc comment, not merely inherited by convention.

### Follow-up: CI job shape

| Option | Description | Selected |
|--------|-------------|----------|
| Install, then assert the version | Fails loudly on drift; reads the requirement's intent as knowing the version | ✓ |
| Hard apt version pin | Literally what the requirement said; breaks when the archive snapshot rolls | |
| Build tmux from source at a tag | Fully deterministic; adds build time and three build dependencies per run | |

**User's choice:** Install, then assert the version
**Notes:** Applies the `govulncheck` `go-version-input: stable` lesson, where an unpinned resolution meant every historical green attested to the wrong toolchain.

| Option | Description | Selected |
|--------|-------------|----------|
| `ubuntu-latest` | Documented, publicly-inspectable image; matches two existing jobs | ✓ |
| Namespace runner | Matches the majority of jobs; image contents not visible from this repo | |
| Both Linux and macOS | Covers the darwin binary, but doubles the flake surface in the highest-risk area | |

**User's choice:** `ubuntu-latest`

### Follow-up: requirement wording

| Option | Description | Selected |
|--------|-------------|----------|
| Amend with a dated note | Update TTY-07 so the yardstick and the work agree before planning | ✓ |
| Leave the text, note it in CONTEXT.md | Less churn, but a later verifier reading TTY-07 literally could flag a mismatch | |
| Change the decision to match the text | Revert to a hard apt pin | |

**User's choice:** Amend with a dated note
**Notes:** Applied to `REQUIREMENTS.md` TTY-07 and, on the same reasoning, to the matching ROADMAP success criterion 5 — the recorded goal-drift lesson is that criteria and goal text are separate edit surfaces and both must be checked.

---

## Frame Stability and Capture Shape (TTY-02, TTY-03, TTY-04, TTY-06)

| Option | Description | Selected |
|--------|-------------|----------|
| `capture-pane -e -C -S -` | Escapes and non-printables as text, over pane plus scrollback | ✓ |
| `pipe-pane` raw byte stream | Highest fidelity, pre-emulator; a second instrument to maintain | |
| Decide during the red demo | Let the RED result pick the instrument | |

**User's choice:** `capture-pane -e -C -S -`
**Notes:** Flag semantics were read from the tmux 3.7c man page on the development machine, not recalled. The decisive fact is that a default capture strips escape sequences, so the TTY-03 assertion would pass against every capture without `-e -C`. The escalation path to `pipe-pane` is preserved in CONTEXT.md D-12 as a documented fallback if family (a)'s mutation produces no observable difference.

| Option | Description | Selected |
|--------|-------------|----------|
| `capture-pane -a` exit status | Two-sided positive probe of real terminal state | ✓ |
| Scrollback content comparison | Matches TTY-04's wording; catches restoration failures | |
| Both probes | Covers two genuinely different properties | |

**User's choice:** `capture-pane -a` exit status
**Notes:** `-a` targets the alternate screen and errors when none exists, which makes it a direct probe rather than an inference from buffer contents.

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded poll, non-convergence fails | A frame that never settles is real flicker or a broken app; both earn a red test | ✓ |
| Bounded poll, non-convergence skips | Safer against CI flakes, but the highest-risk assertion can quietly stop running | |
| Retry the whole case | Absorbs runner hiccups; masks intermittent real flicker | |

**User's choice:** Bounded poll, non-convergence fails

| Option | Description | Selected |
|--------|-------------|----------|
| Compare raw captures | Default capture already trims trailing whitespace; nothing hidden by construction | ✓ |
| Strip a declared animating region | Only needed if a picker animates while idle; each mask can hide a real defect | |
| Join wrapped lines with `-J` | Insensitive to reflow, but preserves trailing spaces and reintroduces spurious diffs | |

**User's choice:** Compare raw captures

---

## Claude's Discretion

- Poll interval and deadline values; TTY-06's `N` and how it is reported.
- Where the expected-count constant lives (Taskfile variable or a committed file the suite also reads).
- tmux session and window naming, pane geometry, and teardown.
- Helper and test function names; commit granularity; exact error wording.
- Whether TTY-05's config-tree hash walks the tree in Go or shells out.
- How the daemon registry is seeded for TTY-04's seeded record.
- The `jq` precondition message on the Task target.

## Deferred Ideas

- Extracting a shared binary-resolution package once a third harness exists.
- Running the tmux suite on macOS in addition to Linux.
- A general `pipe-pane` raw-capture instrument, if `capture-pane -e -C` proves insufficient.
- Two reviewed-but-unfolded todos: the graphstore archtest load-error blindness, and the brew-trust docs item that Phase 12 owns.
