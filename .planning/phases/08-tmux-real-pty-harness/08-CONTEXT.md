# Phase 8: tmux Real-PTY Harness - Context

**Gathered:** 2026-09-09
**Status:** Ready for planning

<domain>
## Phase Boundary

A build-tagged Go test harness that builds the real `codegraph` binary, spawns it inside a
real tmux pane, drives it with `send-keys`, and asserts on `capture-pane` output. It gives
the terminal UI the missing rung between the piped, TTY-blind integration suite and manual
human UAT, and it closes the G-07-1 / G-07-2 defect classes that v1.0 Phase 7's human UAT
caught after both the full piped suite and a deep multi-agent code review had missed them.

This is **test infrastructure**, not product surface. No product behavior changes in this
phase. The only edits to production files are the temporary, byte-cleanly-reverted mutations
that the RED demonstrations require.

Requirements: TTY-01 … TTY-07.

</domain>

<decisions>
## Implementation Decisions

### Anti-vacuity: proving the suite executed (TTY-01, TTY-07)

- **D-01:** CI proves execution by **parsing `go test -json`** from the Task target and
  counting test-granularity `pass` events — never by an in-test mechanism. This is the only
  option that survives a misspelled build tag: a mistyped `-tags` value compiles zero test
  files and `go test` exits 0, and any env-var or counter mechanism living inside the test
  code cannot fire because no test code runs. — **Reversibility:** reversible.
- **D-02:** The assertion is **executed count equals a committed constant**, not a
  greater-than-zero floor. A floor is satisfied by one case running while the rest skip,
  which is the same vacuity shape one level up. Adding, deleting, or silently skipping a case
  fails the job until the constant is updated in the same commit. This follows the v0.12.0
  Phase 5 lesson where a floor-counting guard absorbed a missing call site, and the fix was
  enumerate-then-assert-set-equality. — **Reversibility:** reversible.
- **D-03:** **One** `task test:tmux` target, not two. It always prints both an executed and a
  skipped count. It enforces the expected-count equality only when `CI` is set, so a
  contributor without tmux gets a clean skip report and exit 0, while CI fails hard. Keeps
  the Taskfile as the single definition of the job body, which `TestWorkflowRunBodiesInvokeTask`
  already enforces.
- **D-04:** The tmux-absent path reports a **skip with a reason and a skipped-case count** —
  never a silent pass. Follows the repo's existing `t.Skipf`-with-reason convention
  (`internal/mcp/markdown_test.go`, `internal/githooks/githooks_test.go` both do this for a
  missing `git`).

### RED demonstrations (TTY-03 … TTY-06, and the GRD-06 discipline carried forward)

- **D-05:** Assertions are proven to fire by the **Phase 7 mutation-log shape**, not by
  fault-injection in product code and not by a checked-in bad-pane fixture. For each family:
  a pre-mutation cleanliness gate (`git diff --quiet -- <file>`), the mutation applied, the
  binary rebuilt, the pasted verbatim failing output, the revert, and a byte-clean proof.
  A fault-injection switch was rejected because it ships a deliberately-broken code path that
  itself then needs a guard; a fixture was rejected because it proves only that the matcher
  can match a frozen string, never that the harness observes the defect from a real pty.
- **D-06:** **Four families**, covering TTY-03 through TTY-06 — not only the two the ROADMAP
  success criteria name. TTY-05 and TTY-06 are new assertion classes and would otherwise ship
  having never been watched fail, which is exactly the untrusted-gate shape this milestone
  exists to close. All four mutations are single-token and already located:

  | Family | Req | File | Mutation |
  |---|---|---|---|
  | (a) | TTY-03 | `internal/cli/daemon.go:82` | drop `&& len(records) > 0` from the picker guard |
  | (b) | TTY-04 | `internal/cli/tui/daemonpicker.go:241` | `v.AltScreen = false` |
  | (c) | TTY-05 | `internal/cli/tui/agentpicker.go` cancel path | make cancel write config |
  | (d) | TTY-06 | `internal/cli/tui/agentpicker.go:147` | `v.AltScreen = false` (inline render ⇒ flicker) |

  Log file: `.planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md`, following
  `07-MUTATION-LOG.md`'s shape.

### Location, build tag, and binary provenance (TTY-01)

- **D-07:** The harness lives at **`test/tmux/`**, a top-level sibling of `test/integration/`
  and `test/wireoracle/`. **This overrides `.planning/research/ARCHITECTURE.md` §5**, which
  recommends `internal/tmuxtest/` on the stated premise that "there is currently no top-level
  `test/` directory". That premise is stale — both sibling packages exist today, each with its
  own Taskfile target and its own named CI step. `test/tmux/` follows the convention rather
  than breaking it. — **Reversibility:** reversible.
- **D-08:** The build tag is **`tmux`** (`//go:build tmux`). This is the repo's **first
  feature build tag** — every existing constraint is a GOOS gate or `//go:build ignore` — so
  it sets the convention. The tag names the external dependency that gates the suite, so the
  failure mode is self-describing. — **Reversibility:** costly — a later rename touches every
  file in the package, the Taskfile target, and the CI job together.
- **D-09:** `test/tmux` writes its **own** `TestMain` binary resolver following
  `test/integration/main_test.go`'s contract — honor an env override, abort by name when the
  override is invalid, otherwise `go build` locally — rather than extracting a shared package.
  Duplicates ~30 lines but touches no working harness that a required check depends on, and
  the RED demonstrations need a forced local build of mutated source, which the existing
  resolver has no notion of. **The no-silent-fallback rule is mandatory** and must be stated
  in the new resolver's own doc comment.
- **D-10:** The CI job runs on **`ubuntu-latest`** — a documented, publicly-inspectable image,
  which matters when the job's entire purpose is to prove a real pty behaves correctly. Two
  jobs (`govulncheck`, `perf-regression`) already use it; the rest use Namespace profiles.
- **D-11:** TTY-07's "pinned tmux version" is satisfied by **install-then-assert**: install
  tmux from the distro, then fail loudly when `tmux -V` does not equal a committed expected
  string. A hard apt version pin was rejected as brittle — `apt-get install tmux=<exact>`
  starts failing the moment the archive snapshot rolls, turning an unrelated event into a red
  build. Install-then-assert gives the same know-exactly-what-ran property, and drift becomes
  a visible one-line decision rather than a silent behavior change. This directly applies the
  `govulncheck` `go-version-input: stable` lesson, where an unpinned resolution meant every
  historical green attested to the wrong toolchain.
  **REQUIREMENTS.md TTY-07 was amended in this session** to match, with a dated note — the
  yardstick was not left disagreeing with the work.

### Capture instrument and frame stability (TTY-02, TTY-03, TTY-04, TTY-06)

- **D-12:** Escape-hygiene assertions capture with **`capture-pane -p -e -C -S -`**.
  All three flags are load-bearing, verified against the tmux 3.7c man page:
  - **`-e`** includes escape sequences; the default capture strips them.
  - **`-C`** renders non-printable bytes as octal `\xxx`, which is what makes a leaked ESC
    byte matchable *as text*.
  - **`-S -`** starts at the beginning of history; the default captures only the visible pane,
    and TTY-04 asserts on scrollback.

  **Without `-e -C` the TTY-03 assertion cannot fire at all** — it would pass against every
  capture, fireable or not. That is the milestone's own vacuity shape hiding inside the
  harness built to catch it. Family (a)'s RED demonstration is what proves the instrument is
  adequate; if the mutation produces no observable difference under this capture, escalate to
  `pipe-pane` (raw byte stream, pre-emulator) and **record why in the mutation log** rather
  than weakening the assertion.
- **D-13:** Alt-screen entry and exit are asserted by **`capture-pane -a` exit status**, a
  two-sided positive probe, not by inferring from buffer contents. `-a` targets the alternate
  screen and errors when none exists. During the picker it must exit 0 and contain
  `Running daemons`; after quit it must exit non-zero. Under mutation (b) it is already
  non-zero during the picker, so the assertion fails loudly and unambiguously.
- **D-14:** The stability poll is a **bounded poll whose non-convergence is a FAILURE**, never
  a skip and never a whole-case retry. Capture on a short interval until two consecutive
  captures are byte-identical, up to a deadline; blowing the deadline fails the test naming
  the last two differing captures. A frame that never settles is either real flicker or a
  broken app, and both earn a red test. Skipping on non-convergence would convert the
  milestone's highest-risk assertion into one that can quietly stop running.
  **Note for the planner:** TTY-02's "no fixed sleeps in the assertion path" forbids an
  *unconditional* sleep-then-assert. A bounded poll with a convergence condition satisfies it;
  the interval between captures is not the prohibited thing.
- **D-15:** Captures are compared **raw, with no normalization** — no masked regions, no `-J`.
  Default capture already trims trailing whitespace, which removes the main source of false
  differences. Any normalization added later must justify what it is allowed to hide. If a
  picker turns out to animate while idle, add a declared masked region and state what it hides;
  do not add one pre-emptively.

### Claude's Discretion

- Poll interval and deadline values; TTY-06's `N` and how it is reported.
- The expected-count constant's home (Taskfile variable vs a committed file the suite reads).
- tmux session/window naming, pane geometry, and teardown; helper and test function names.
- Whether TTY-05's config-tree hash walks the tree in Go or shells out.
- How the daemon registry is seeded for TTY-04's "seeded record" — `daemon.Register` vs
  writing the registry file directly.
- The `jq` precondition message on the Task target (precedent: `Taskfile.yml:3512`).
- Commit granularity and exact error wording.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` § "Phase 8: tmux Real-PTY Harness" — goal, the five success criteria, and the Notes paragraph flagging that external precedent for tmux-driven TUI e2e testing is thin to absent
- `.planning/REQUIREMENTS.md` lines 29-35 — TTY-01 … TTY-07 verbatim (TTY-07 amended 2026-09-09, see D-11)

### Milestone research — read with the correction in D-07
- `.planning/research/FEATURES.md` § 5 "tmux Real-PTY E2E Harness for the TUI (999.2)" — the four assertion classes and, importantly, the **anti-features** table: no pixel-diff visual regression, no unconditional execution in every CI environment, no byte-for-byte frame equality including timing-sensitive glyphs
- `.planning/research/ARCHITECTURE.md` § 5 — useful on why the existing suite structurally cannot cover this. **Its "where it should live" recommendation is superseded by D-07**, and its claim that no test builds the binary via `go build` is false: `test/integration/main_test.go` does exactly that

### Prior-phase discipline this phase inherits
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` — the mutation-log shape D-05/D-06 follow, including the two documented deviations for cases where the tracked file must not be edited
- `.planning/phases/07-guards-that-cannot-fire/07-CONTEXT.md` § D-07 — the precedent for declining a second guard when one assertion plus its RED demonstration already proves the property

### Code under test and reusable substrate
- `test/integration/main_test.go` — the binary-resolution contract D-09 follows; read its `resolveTestBinPath` doc comment for the no-silent-fallback reasoning, and its package doc for the GOLDEN-01 "a suite that silently didn't run" warning
- `test/integration/piped_never_hang_test.go` — the throwaway-`HOME` convention and the bounded never-hang pattern
- `internal/cli/daemon.go` — the G-07-1 fix (empty-registry short-circuit) with its rationale comment
- `internal/cli/tui/daemonpicker.go` — the G-07-2 fix; note the bubbletea v2 comment explaining that alt-screen is a per-`View` field, not a `Program` option
- `internal/cli/tui/agentpicker.go` — the checkbox picker: `[x]`/`[ ]` glyph rendering, `space`/`enter`/`q`/`esc` key handling, and its own `AltScreen`
- `internal/cli/tui/tty.go` — `InteractiveAllowed`, the stdin+stdout TTY gate the harness must satisfy for the pickers to open at all
- `Taskfile.yml` lines 3548-3557 — the existing "requires a non-zero count, not a green exit code" gate; D-01/D-02's assertion is the same shape

### External tool surface
- `man tmux` § `capture-pane` — the authority for D-12/D-13. Flag semantics were read from tmux 3.7c on the dev machine, not recalled

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`test/integration/` package shape**: `TestMain` that builds the binary once, a package-level `binPath`, a `runBinary` helper, `CODEGRAPH_TEST_BIN` override with a strict resolution contract, throwaway-`HOME` env for tests that would otherwise touch the developer's real `~/.codegraph`. `test/tmux` mirrors this rather than importing it (D-09).
- **`Taskfile.yml` positive-count gate** (~3548-3557): reads a JSON field through `jq` and fails with `::error::` when the count is zero, explicitly saying a non-zero count is required rather than a green exit. Directly transferable to D-01/D-02, including the `jq` precondition pattern at ~3512.
- **Skip-with-reason convention**: `t.Skipf` carrying the underlying error, used for a missing `git` in `internal/mcp/markdown_test.go:116` and `internal/githooks/githooks_test.go:31`.
- **Phase 7 mutation-log**: a committed, four-family worked example of the exact RED discipline D-05 requires.

### Established Patterns
- **Standing rule 84d1gfpywd**: a guard must carry a positive assertion that it did its work. D-02 (expected count) and D-13 (`-a` exit status) are the applications here.
- **A gate is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation.** D-05/D-06.
- **`Taskfile.yml` is the single definition of every CI job body**, enforced by `TestWorkflowRunBodiesInvokeTask`. The tmux CI job must be a thin caller of `task test:tmux` (D-03).
- **`go test -run PATTERN` exits 0 when the pattern matches nothing** — a standing repo gotcha, and the direct ancestor of D-01's reasoning about a misspelled build tag.
- **bubbletea v2 (`charm.land/bubbletea/v2`)**: alt-screen is a per-`View` field (`v.AltScreen = true`), not a `Program` option. Both pickers set it; both are mutation targets.
- **No feature build tag exists yet** — only GOOS gates and one `//go:build ignore`. D-08 establishes the first.

### Integration Points
- **New**: `test/tmux/` package; a `test:tmux` Taskfile target; a `tmux-e2e` job in `.github/workflows/ci.yml` on `ubuntu-latest`; `08-MUTATION-LOG.md`.
- **Touched temporarily and reverted**: `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/agentpicker.go` — mutations only, byte-clean at phase close.
- **Amended**: `.planning/REQUIREMENTS.md` TTY-07 (D-11).
- **Not touched**: `test/integration/`, `test/wireoracle/`, and every production code path in its shipped state.

</code_context>

<specifics>
## Specific Ideas

- The maintainer chose the recommended option in every question, including where the
  recommendation argued *against* the literal requirement text (D-11) and *against* a prior
  phase's economy precedent (D-06 extends RED coverage past what the success criteria demand).
  The consistent through-line: prefer the mechanism that cannot be bypassed over the one that
  is simpler to read.
- `tmux 3.7c` is installed on the development machine, so the suite executes rather than skips
  locally. The CI expected-version string in D-11 must be whatever `ubuntu-latest` actually
  ships, determined empirically at execution time — **not** guessed from the dev machine's
  version.
- Two pending todos matched Phase 8 by keyword and **neither was folded** — see Deferred.

</specifics>

<deferred>
## Deferred Ideas

- **Extracting a shared binary-resolution package** for `test/integration` and `test/tmux`
  (D-09 chose duplication). Worth revisiting if a third harness appears, at which point the
  drift risk outweighs the cost of refactoring a required-check harness.
- **Running the tmux suite on macOS as well as Linux** (D-10 chose Linux only). Terminal
  behavior is exactly where OS differences show up and the project ships a darwin binary, but
  doubling the job also doubles the flake surface in the milestone's highest-risk area.
- **A `pipe-pane` raw-byte-stream capture path** as a general instrument, if D-12's
  `capture-pane -e -C` proves insufficient during family (a)'s RED demonstration.

### Reviewed Todos (not folded)
- **`graphstore archtest ignores per-package go/packages load errors`**
  (`.planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md`,
  matched at 0.9) — **not folded**. It matched on generic keywords (test, review, query, phase)
  and is the sibling of the blindness fixed in Phase 7's CR-01, with no tmux or TTY surface
  whatsoever. Folding it would widen this phase off-domain.
- **`brew trust instructions recommend the broader tap grant with no security framing`**
  (`.planning/todos/pending/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md`,
  matched at 0.6) — **not folded**. ROADMAP.md explicitly assigns it to Phase 12 as DOCS-07,
  deliberately, so that one phase owns the docs tree.

</deferred>

---

*Phase: 8-tmux-real-pty-harness*
*Context gathered: 2026-09-09*
