---
phase: "12"
slug: "cli-reference-docs-tail"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-13"
---

# Phase 12 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — plans 12-01 through 12-03 each
carry a `<threat_model>` block. The register below is the union of all three plans' rows,
deduplicated by id; the shared `T-12-SC` id (declared identically by all three plans as "npm/pip/
cargo installs … none") merges into a single row tagged `(12-01, 12-02, 12-03)`. Verification
depth is ASVS L1 (grep/execution-level mitigation presence), which the workflow's short-circuit
rule declares sufficient for `threats_open: 0` at L1.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| live Cobra tree → `tools/clidoc` → `docs/CLI-REFERENCE.md` | in-process data only, no user input; the risk is the committed file drifting from, or being hand-edited away from, what the tree actually says | command/flag names, `Short`/`Long` text, usage strings — all already-compiled-in strings, never external input |
| `docs/CLI-REFERENCE.md` + `internal/cli/testdata/cli-reference-allowlist.txt` → `cli_reference_test.go` | two committed files read by fixed relative paths; fail-closed if either is missing or malformed; the risk is a silent omission (a flag in neither) or a stale exemption (an allowlist entry matching nothing) | generated markdown text and a tab-separated allowlist, read only, never written by the guard |
| `internal/cli` → `tools/clidoc` via the exported `NewRootCmd()` | an API widening that must return exactly the tree `Execute()` already runs, never a second, divergent tree | the constructed `*cobra.Command` tree object, in-process only |
| `docs/RELEASE.md` → a Homebrew user's shell | the instruction tells a user to switch off (or narrowly scope) a Homebrew security control before a cask's arbitrary-Ruby post-install hook runs; which form is recommended sets the blast radius of that grant (UF-2) | prose and one `sh` code block copy-pasted by a human, never executed by this project's own code |
| `Taskfile.yml` / `.github/workflows/ci.yml` → developer working tree | `docs:cli` regenerates in place (a sanctioned mutation); `docs:cli:drift` regenerates only into `mktemp -d`, never touching the working tree — the risk is a drift target that clobbers an uncommitted edit | file writes, scoped either to `-out docs/CLI-REFERENCE.md` (in-place, sanctioned) or a removed-on-EXIT scratch directory (drift check) |

**The phase's new trust surface is a generated document that must equal what the binary's own
command tree says, a guard that must see every flag the tree registers even when the generator
cannot, and one sentence of security guidance whose narrowness sets a Homebrew user's blast
radius.**

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-12-01 | Tampering | `docs/CLI-REFERENCE.md` hand-edited after generation, or left stale after a flag change | high | mitigate | `task docs:cli:drift` (12-01 Task 1) regenerates into `mktemp -d` and `cmp -s`-compares byte-for-byte, reports `compared 1 generated file` before comparing, names the file and prints a `diff -u \| head -40` excerpt on mismatch, exits 1; wired into `ci.yml`'s `test` job directly after `proto:drift`. 12-03 family (b) watched this RED live against a deleted `--no-open` Options line (count-before-compare, named file, diff excerpt) and reverted byte-clean; re-verified GREEN at phase close. | closed |
| T-12-02 | Information Disclosure (avoidance failure) | a hidden / deprecated flag absent from BOTH the generated reference and any allowlist record — the generator's structural blind spot | high | mitigate | `TestEveryRegisteredFlagIsAccountedFor` (12-01 Task 2) walks every command including hidden ones (`InitDefaultHelpFlag`/`InitDefaultCompletionCmd`/`InitDefaultVersionFlag` applied in lockstep with the generator) and fails naming `unaccounted flag: <path> --<flag> (<reason>)` unless the reference or a reason-carrying allowlist entry covers it; runs in `task test:unit`. 12-03 family (a) watched this RED live against a throwaway hidden flag registered on `ui` (`unaccounted flag: codegraph ui --zz-throwaway`, 116 flags inspected) while `task docs:cli:drift` independently stayed GREEN under the same mutation — proving the guard closes exactly the gap the drift gate structurally cannot see — and reverted byte-clean; re-verified GREEN at phase close. | closed |
| T-12-03 | Tampering | allowlist rot — an entry lingers after its flag/command is removed, or is added to hide a visible flag | medium | mitigate | every allowlist entry must match ≥ 1 inspected flag or the guard fails with `stale allowlist entry: <key> (matches no registered command or flag)`; visible flags on documented commands never consult the allowlist, so an entry for one is stale by construction. 12-03 family (c) watched this RED live against a bogus `codegraph ui --zz-bogus` entry (counts unchanged: 115 flags, 1 accepted via allowlist — the genuine `codegraph man` entry) and reverted byte-clean; re-verified GREEN at phase close. | closed |
| T-12-04 | Elevation of Privilege | exported `NewRootCmd()` widens `internal/cli`'s public API surface | low | accept | it returns `newRootCmd()` unchanged — the same tree `Execute()` runs, no new call path in the shipped binary; `cmd/codegraph/main.go` asserted unchanged (12-01 verify: `git diff --quiet ca015c4b -- go.mod go.sum cmd/codegraph/main.go`); `internal/` cannot be imported outside this module by Go's own visibility rules. | closed (accepted) |
| T-12-05 | Tampering | the generator or drift target writes outside its intended output path, or clobbers a working-tree edit | low | mitigate | `tools/clidoc` writes only to its `-out` flag's path; `docs:cli:drift` writes only under a `mktemp -d` scratch directory removed on `EXIT` via `trap`, never touching `docs/CLI-REFERENCE.md` itself; 12-01 Task 1 asserted `git status --porcelain` clean after a drift run. | closed |
| T-12-06 | Denial of Service (gate fatigue) | a date stamp or machine-dependent default makes `docs:cli:drift` permanently red, teaching developers to ignore it | medium | mitigate | `root.DisableAutoGenTag = true` on the root (propagated to every subcommand by `GenMarkdownCustom`'s own parent-walk) removes cobra's date-stamped footer entirely (zero date footers asserted in 12-01 Task 1's verify); 12-01 Task 1 ran the gate twice and required byte-identical transcripts (both GREEN runs pasted in 12-01-SUMMARY.md); 12-03's phase-close re-run reproduces the identical `compared 1 generated file` / byte-identical transcript a third time. | closed |
| T-12-07 | Repudiation | a vacuous guard — a walk over an incomplete tree (missing the completion family or `--version`) or over nothing at all reads as a passing test | medium | mitigate | both the generator and the guard call `InitDefaultCompletionCmd()` and `InitDefaultVersionFlag()` in lockstep before walking; positive floors (`cliReferenceMinCommands` ≥ 26, `cliReferenceMinFlags` ≥ 50; the real tree today walks 36 commands / 115 flags) are asserted before any pass/fail verdict is possible; the accounting identity `viaReference + viaAllowlist + len(unaccounted) == flagCount` is asserted every run; the walked-commands/inspected-flags counts line is logged on every single run, RED or GREEN (12-01, 12-03). | closed |
| T-12-08 | Elevation of Privilege (user-side; UF-2 docs-as-security-guidance) | `docs/RELEASE.md`'s untrusted-tap instructions — recommending the tap-wide grant hands every current and future cask/command in the tap arbitrary-Ruby-at-install rights on the user's machine, with no statement of the control being bypassed | high | mitigate | D-11 wording (12-02 Task 1): `brew trust --cask seanb4t/tap/codegraph` is THE command in the quoted error, the prose, and the `sh` block; one sentence names the control being opted out of (arbitrary Ruby at install time via this cask's own post-install hook) and that `--cask` scopes the opt-out to one cask; the tap-wide grant is named as existing and not recommended, and its command is never spelled anywhere in the file. Verified by one-time source assertions in 12-02's own `<verify>` block (narrow-form count ≥ 4, tap-wide spellings 0, `--tap` 0) plus a full diff review pasted in 12-02-SUMMARY.md — per D-12, no committed test or guard exists for wording, so the disposition is a **verdict**, not a gate result. | closed (verdict) |
| T-12-09 | Repudiation | the quoted Homebrew error still carrying the broad alternative, so a reader copies it "because the error said so" | medium | mitigate | the quote is trimmed to its narrow form with an ellipsis (`Run \`brew trust --cask seanb4t/tap/codegraph\` … to trust it.`), asserted exactly once in 12-02's `<verify>`; the prior "run the command the error names" phrasing is asserted absent (count 0) both before and after. | closed |
| T-12-10 | Information Disclosure | README link to a generated file that could drift from the binary's actual command tree | low | accept | 12-01's `docs:cli:drift` (CI-wired, immediately after `proto:drift`) keeps the linked target current; README itself carries no CLI facts of its own that could go stale — it only points at the generated file. | closed (accepted) |
| T-12-11 | Tampering | `STATE.md` / todo files edited into a shape their owning tools no longer parse | low | mitigate | the todo moved via `gsd_run todo complete` (tool-owned move + frontmatter stamp, not a hand edit); `STATE.md`'s change is one row moved between two existing tables, values only; heading counts asserted unchanged against `ca015c4b` in 12-02's `<verify>`. | closed |
| T-12-12 | Tampering | a mutation from 12-03 families (a)-(c) committed by accident — a hidden throwaway flag shipped in the binary, a stale reference, or a bogus allowlist entry | medium | mitigate | pre-mutation cleanliness gate (`git diff --quiet -- <file>`) run before every mutation and every revert; every family's mutation reverted via `git checkout --`, re-verified clean; `12-MUTATION-LOG.md`'s Task 1 verify asserted `git diff --quiet` on all three affected files and that the log's own commit (`d48d1f7b`) carries exactly one file; Task 2's phase-close gate re-runs both the guard and the drift gate GREEN and asserts `git status --porcelain` empty at the final commit. | closed |
| T-12-13 | Repudiation | a `threats_open: 0` written without the closing evidence backing it | medium | mitigate | this register ties `threats_open` to a literal count of `open` rows at severity `high` (0, matching the `threats_open: 0` frontmatter value exactly) and names three `high \| mitigate \| … \| closed` rows above (T-12-01, T-12-02) plus one `closed (verdict)` (T-12-08) — each citing the specific test, target, or diff-review evidence that closed it, not an unsupported assertion. | closed |
| T-12-SC | Tampering | npm/pip/cargo installs | low | accept | none across all three plans — the only touched dependency fact is that `cobra/doc` was already required via `internal/cli/man.go` before this phase began; `go.mod`/`go.sum` asserted unchanged (12-01 verify: `git diff --quiet ca015c4b -- go.mod go.sum cmd/codegraph/main.go`); no npm/pypi/crates seam applies to this phase at all — every plan is Go-and-docs-only. (12-01, 12-02, 12-03) | closed (accepted) |

*Status: open · closed · closed (accepted) · closed (verdict)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|--------------|------|
| R-12-01 | T-12-04 | `NewRootCmd()` returns `newRootCmd()` unchanged; no new call path exists in the shipped binary, and `cmd/codegraph/main.go` is asserted byte-unchanged | plan 12-01 | 2026-09-13 |
| R-12-02 | T-12-10 | `docs:cli:drift` is CI-wired and keeps the linked reference current; README carries no independent CLI facts that could go stale on their own | plan 12-02 | 2026-09-13 |
| R-12-03 | T-12-SC | No package-manager install runs in any of this phase's three plans; the only dependency touched (`cobra/doc`) was already required before this phase, with `go.mod`/`go.sum` asserted unchanged | plans 12-01, 12-02, 12-03 | 2026-09-13 |

---

## Notes

**1. The five D-13 rows, by name.** D-13 names five specific concerns this register must cover;
each is addressed by a specific row above: **UF-2 docs-as-security-guidance framing** — T-12-08
(the `brew trust` wording recommends the narrow `--cask` grant with security framing, closed by
verdict). **Hidden-flag accounting blind spot** — T-12-02 (`TestEveryRegisteredFlagIsAccountedFor`
walks hidden commands too; 12-03 family (a) proved the drift gate structurally cannot see this
class of gap while the guard can). **Generated-file tampering caught by drift** — T-12-01
(`task docs:cli:drift`; 12-03 family (b) proved it RED on a deleted line). **Allowlist rot** —
T-12-03 (the guard's rot-enforcement pass; 12-03 family (c) proved it RED on a bogus entry).
**Exported constructor widens nothing** — T-12-04 (`NewRootCmd()` returns the identical tree,
`cmd/codegraph/main.go` unchanged).

**2. Two `unclassified` edge assumptions, recorded verbatim rather than silently resolved.**
DOCS-06's guard performs a whole-document substring search (`docMentionsFlag`), not a
section-scoped match — so a flag name that recurs anywhere in `docs/CLI-REFERENCE.md` (e.g. in
another command's Synopsis prose) can, in principle, mask a missing Options line for a *different*
occurrence of that same flag name. This edge case is `unclassified` by this phase's own gates:
12-03 family (b) deliberately worked around it by choosing `--no-open` (unique to one Options line
in the whole file) rather than `--editor-url` (which also appears in `ui`'s own Synopsis prose,
and so would NOT have gone RED on the guard alone) — see `12-MUTATION-LOG.md`'s Family (b) note.
No test asserts the guard's behavior against a flag name that recurs across two *different*
commands' documentation; this remains an `unclassified` residual risk carried forward, not a
closed one. Separately, DOCS-07's wording change is `unclassified` in the same sense: per D-12, no
committed test, guard, or docs-grep verifies `docs/RELEASE.md`'s brew-trust prose on an ongoing
basis — T-12-08/T-12-09's dispositions above are one-time verdicts from a diff review at plan
time, not a regression-proof gate; a future edit to that file could silently reintroduce the
broader, unframed wording with nothing in CI to catch it.

**3. Research assumptions A1/A2 — both resolved by 12-01, not carried forward open.**
12-RESEARCH.md's Assumptions Log recorded two low-risk, plan-time-resolvable assumptions. **A1**
(GitHub's Markdown auto-anchor rules resolve the planner's `linkHandler` transform correctly) was
resolved by 12-01 Task 1's own verify assertion requiring `]\(#codegraph-githooks-status\)` to
appear at least once in the generated file — a same-leaf-name, different-depth command pair
(`codegraph status` vs `codegraph githooks status`) whose SEE ALSO anchors must resolve to
distinct headings, confirmed present in the committed `docs/CLI-REFERENCE.md`. **A2** (whether
`tools/clidoc` should also emit a section for `root` itself) was resolved by 12-01 Task 1 calling
`walk(root)`, confirmed by the 35-heading count assertion (34 would mean root was skipped). Neither
assumption is `[ASSUMED]` or open at phase close.

**4. The `--version` planner addition.** 12-01 additionally called `root.InitDefaultVersionFlag()`
in both the generator and the guard's tree-construction helper, beyond D-02's explicit
`InitDefaultCompletionCmd()` instruction — because `codegraph --version` is a real, registered root
flag only wired into the tree inside `Command.ExecuteC()`, never by a bare `newRootCmd()` call.
Without this addition, both the generated reference and the accounting guard would have silently
agreed on an incomplete tree missing one real, user-facing flag — exactly the vacuous-guard shape
T-12-07 exists to close. This raised the guard's flag floor from 114 to 115, all still accounted
for (114 via the reference, 1 via the allowlist for `codegraph man --help`).

**5. No new dependency of any kind.** All three plans in this phase are additive over an
already-vendored dependency: `github.com/spf13/cobra/doc` was already imported by
`internal/cli/man.go` before this phase began. `go.mod`/`go.sum` are asserted byte-unchanged by
12-01's own verify (`git diff --quiet ca015c4b -- go.mod go.sum cmd/codegraph/main.go`), confirmed
again at this plan's phase-close gate. No npm/pypi/crates package-legitimacy seam applies to this
phase — see T-12-SC.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-13 | 14 | 14 | 0 | plan 12-03 (ASVS L1 inline; auditor short-circuited per `threats_open:0` + `register_authored_at_plan_time:true` + `asvs_level:1`) |

**Audit note — what was checked by execution vs. by reading.** All three `high`-severity rows
(T-12-01, T-12-02, T-12-08) were checked by **execution** at phase close: `T-12-01` and `T-12-02`
by re-running `GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift` and
`GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEveryRegisteredFlagIsAccountedFor' ./internal/cli/`
live at HEAD (both GREEN, tails pasted in 12-03-SUMMARY.md), on top of `12-MUTATION-LOG.md`'s own
RED demonstrations against confirmed-applied mutations of this project's own tree, committed file,
and allowlist. `T-12-08` was checked by **reading** — per D-12's own design, no test exists for
wording; the verdict rests on 12-02's one-time source assertions (re-runnable, not committed as a
gate) plus a full diff review, both reproduced in 12-02-SUMMARY.md. The medium- and low-severity
rows disposed `mitigate` (T-12-03, T-12-05, T-12-06, T-12-07, T-12-09, T-12-11, T-12-12, T-12-13)
were checked by a mix of execution (T-12-03, T-12-06, T-12-07, T-12-12 all re-verified live at
phase close) and reading the cited source assertion or log entry (T-12-05, T-12-09, T-12-11,
T-12-13). The `low`-severity `accept` rows (T-12-04, T-12-10, T-12-SC) were checked by reading the
cited unchanged-file assertion, consistent with prior phases' (10-SECURITY.md, 11-SECURITY.md)
audit-note precedent for accepted risks that inherit an already-verified guard or assert a file is
untouched.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-13

**Outstanding, not security-blocking:**
- The two `unclassified` edge assumptions in Notes item 2 (the guard's whole-document flag-name
  match across two different commands; DOCS-07's wording having no ongoing committed gate) are
  carried forward as explicit residual risk, not resolved by this phase.
