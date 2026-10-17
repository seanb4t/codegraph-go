---
phase: 03-rename-stub-removal
verified: 2026-09-26T21:30:00Z
status: passed
score: 13/13 must-haves verified
covered_files:
  - ".changes/unreleased/Breaking-20260926-165849.599611000.yaml"
  - ".planning/phases/03-rename-stub-removal/03-01-PLAN.md"
  - ".planning/phases/03-rename-stub-removal/03-01-SUMMARY.md"
  - ".planning/phases/03-rename-stub-removal/03-CONTEXT.md"
  - ".planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md"
  - ".planning/phases/03-rename-stub-removal/03-RESEARCH.md"
  - ".planning/phases/03-rename-stub-removal/03-REVIEW.md"
  - ".planning/phases/03-rename-stub-removal/03-SECURITY.md"
  - ".planning/phases/03-rename-stub-removal/03-VALIDATION.md"
  - "internal/cli/root.go"
  - "internal/cli/testdata/cli-reference-allowlist.txt"
  - "test/integration/removed_verbs_test.go"
covered_digest: "v1:sha256:1bac802ca8a96b90fef38f36f34a1e9cf08e8aa26e3ca6d6f33aabea3fc68e4e"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 3: Rename-Stub Removal Verification Report

**Phase Goal:** The hidden `query` and `unlock` rename stubs v0.14.0 promised to remove in the following minor are gone, and this phase's close is the first observed firing of the phase-close capability in this repository — the milestone's `Breaking` fragment is the one the capability wrote.
**Verified:** 2026-09-26T21:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | VERB-09 SC1 (D-01/D-02): the real spawned binary answers all four removed-verb invocations with cobra's exact unknown-command stderr, empty stdout, exit 1, and an executed-count guard of 4 | ✓ VERIFIED | Ran `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v` myself: `--- PASS: TestRemovedVerbsAreUnknownCommands` with all 4 subtests PASS and `removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases`. Source at `test/integration/removed_verbs_test.go` contains `func TestRemovedVerbsAreUnknownCommands(`, `len(cases) != 4`, `unknown command %q for %q`, and `removed-verb guard: executed %d of %d cases`, with zero occurrences of the old rename text. |
| 2 | The stub files, their unit test, and their registration/allowlist entries no longer exist | ✓ VERIFIED | `internal/cli/renamed.go`, `internal/cli/renamed_test.go`, `test/integration/renamed_stubs_test.go` confirmed absent on disk. `rg -o -w -e 'newQueryCmd' -e 'newUnlockCmd' --glob '*.go' .` returns 0 matches. `internal/cli/testdata/cli-reference-allowlist.txt` verified to hold exactly 6 header lines plus 3 entries (`man`, `hook`, `hook pretooluse`), read directly. |
| 3 | RED-first (rule x1cjy9vyhq, D-02, D-04): exactly one `test(03-01):` commit precedes exactly one `feat(cli)!:` commit touching internal/cli or test/integration, with the RED transcript pasted in the log | ✓ VERIFIED | `git log --reverse --format=%s 84778a82..HEAD -- internal/cli test/integration` returns exactly `test(03-01): expect the removed query and unlock verbs to be unknown commands (VERB-09)` then `feat(cli)!: remove the query and unlock rename stubs`. `git diff-tree --name-only 90dc419a` = exactly `renamed.go` (D), `renamed_test.go` (D), `root.go` (M), `cli-reference-allowlist.txt` (M). Commit body carries `BREAKING CHANGE: … search --full … daemon unlock …`, no CI-skip marker. 03-MUTATION-LOG.md `## RED-first transcript` section holds the failing `--- FAIL:` transcript with the stub's rename text as the got value. |
| 4 | VERB-09 SC2 (D-03): no other generated artifact changed; docs:cli, drift and flag-accounting/grouping tests are green; full suite green | ✓ VERIFIED | Ran `task docs:cli:drift` myself: `docs:cli:drift: compared 1 generated file` / `docs/CLI-REFERENCE.md byte-identical to a fresh regeneration`. Ran `go test ./internal/cli/ -run '^(TestEveryRegisteredFlagIsAccountedFor|TestEveryCommandHasGroupID)$'`: both PASS (`walked 37 commands …, inspected 117 flags`; `inspected 24 visible commands across 4 groups`). Ran `git diff --quiet 84778a82 HEAD -- CHANGELOG.md .changes/v0.14.0.md .changes/header.tpl.md docs/CLI-REFERENCE.md internal/cli/cli_reference_test.go` — exit 0 (untouched). Ran `GOTOOLCHAIN=go1.26.6 task test:unit` myself — all packages `ok`/cached, no FAIL. |
| 5 | Both guards bite (rule 84d1gfpywd): the removed-verb guard goes RED against a revived alias and a revived SuggestFor hint; the allowlist guard goes RED against re-added entries; all mutations confirmed applied and byte-clean reverted in a scratch worktree that never touched the main checkout | ✓ VERIFIED | 03-MUTATION-LOG.md Family (a) (line 223) and Family (b) (line 315) contain full transcripts: alias mutation → 2 query subtests FAIL with real search output, 2 unlock subtests PASS; SuggestFor mutation → 2 query subtests FAIL with `Did you mean this?\n\tsearch\n`; allowlist mutation → `TestEveryRegisteredFlagIsAccountedFor` FAILs with exactly 2 `stale allowlist entry:` lines. Each mutation records numstat confirmation, revert, and a `git diff --quiet` byte-clean proof. Independently confirmed the main checkout is currently clean (`git status --porcelain` empty) and no scratch worktree remains registered (`git worktree list` shows only the real checkout and an unrelated herdr worktree). |
| 6 | VERB-09 SC2 census (D-06): the v0.14.0 instrument, byte-identical, finds the positive control first, then reports exactly 4 lines across 2 files before removal and 0 after | ✓ VERIFIED | 03-MUTATION-LOG.md Family (c)/(d) show the control found on `:1:`/`:3:` each time, `before: 4 lines across 2 files` (`renamed.go:10-11`, allowlist `:8-9`), `after: 0 lines across 0 files`, `rg` exit 1. Independently re-ran the exact instrument (`rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .`) at current HEAD myself: no output, exit 1 — zero hits, matching the after-state (and even confined to none, since the fragment body never spells the literal `codegraph query`/`codegraph unlock` sequence). |
| 7 | D-05 and its D-06 consequence: root.go's comments record the v0.15.0 removal, name no deleted identifiers/files, and no tracked file outside .planning/ pairs the binary name directly with a removed verb | ✓ VERIFIED | Read `internal/cli/root.go` directly: package doc, group comment, commandGroups comment and newRootCmd doc all now read "the one-release `query`/`unlock` rename stubs were removed in v0.15.0 (VERB-09)" with no mention of `renamed.go`, `newQueryCmd`, or `newUnlockCmd`. `rg -o -F 'VERB-09' internal/cli/root.go` finds 2+ occurrences; `rg -o -F 'renamed.go' --hidden --glob '!.planning/**' --glob '!.git/**' .` finds 0. |
| 8 | D-09: 03-01-SUMMARY.md states the change in user terms, naming only `search --full <term>` and `daemon unlock [path]` as replacements | ✓ VERIFIED | Read 03-01-SUMMARY.md: substantive one-liner and first Accomplishments bullet both state the commands are removed, invoking either is an unknown-command error exiting 1, and name exactly `search --full <term>` / `daemon unlock [path]` as replacements. No other replacement command is named. |
| 9 | CAP-05/SC3 (D-08): the changie-fragments capability fired at this phase's real close, writing the Breaking fragment via its own commit with the correct trailers, on the milestone branch, with no earlier commit pre-writing it | ✓ VERIFIED | `.changes/unreleased/Breaking-20260926-165849.599611000.yaml` exists: `kind: Breaking`, body "Removed the hidden `query` and `unlock` commands, so running either now fails as an unknown command; run `search --full <term>` or `daemon unlock [path]` instead.", `custom: PR: "88"`. Its commit `d9892bebe59831841f313f9c0f074417a7a44b8c` has subject `docs(03): add changelog fragments` and trailers `Changie-Phase: 3` / `Changie-Summaries: 03-01` (read via `git log -1 --format`). `git merge-base --is-ancestor d9892beb gsd/v0.15.0-milestone` succeeds; `git branch --contains d9892beb` shows only `gsd/v0.15.0-milestone`. `git log 84778a82..d9892beb -- .changes/unreleased` lists only the fragment commit itself — no earlier commit in the phase range touched the directory. Fragment body accurate per D-09 (Breaking kind, correct replacements) — no D-10 correction needed. |
| 10 | VERB-09 SC4 (D-12): with the fragment present, `task changie -- next auto` on the milestone branch prints `v0.15.0` | ✓ VERIFIED | Ran `task changie -- next auto` myself on `gsd/v0.15.0-milestone`: output `v0.15.0`. |
| 11 | D-07: a census re-run after the close hits only inside the capability-written fragment (or nothing); the instrument is never edited | ✓ VERIFIED | Re-ran the unmodified D-06 instrument at current HEAD (fragment present): no output, exit 1. Zero hits overall — even the fragment file does not literally contain the `codegraph\s+(query|unlock)` sequence, so there is no hit anywhere, which satisfies D-07's bar (a hit confined to the fragment is expected; here there is no hit at all). |
| 12 | D-10 conditional (fragment inaccuracy) | N/A (not triggered) | Fragment kind is `Breaking` and body names exactly `search --full` and `daemon unlock` — accurate per D-09. No correction commit exists or is needed. |
| 13 | D-11 conditional (capability did not fire) | N/A (not triggered) | The capability fired successfully (see truth 9); no hand-written fallback fragment exists, and REQUIREMENTS.md correctly keeps CAP-05 as `[x]` Complete. |

**Score:** 13/13 truths verified (11 affirmatively verified, 2 conditional truths correctly not triggered)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `test/integration/removed_verbs_test.go` | VERB-09 guard: spawned binary, four removed-verb invocations, exact unknown-command stderr, exit 1, executed-case count 4 | ✓ VERIFIED | Exists, contains `func TestRemovedVerbsAreUnknownCommands`, ran and passed 4/4 with executed-count line |
| `internal/cli/root.go` | Command tree without the rename stubs; comments record the v0.15.0 removal | ✓ VERIFIED | `AddCommand` list has no `newQueryCmd()`/`newUnlockCmd()`; comments name VERB-09 twice |
| `internal/cli/testdata/cli-reference-allowlist.txt` | Allowlist holding only the man, hook and hook pretooluse entries | ✓ VERIFIED | 6 header lines + exactly 3 entries, read directly |
| `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` | RED/GREEN transcripts, regression gates, Families (a)-(d), commit-history audit | ✓ VERIFIED | 514 lines; all 13 required section headings present (`## RED-first transcript`, `## GREEN after the removal`, `## Regression gates`, `## Family (a)` through `(d)`, `## Commit-history audit`, `## Verdict`) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `internal/cli/root.go` AddCommand list | `cmd/codegraph/main.go` single error seam | no `query`/`unlock` child → cobra unknown-command error → main prints once, exits 1 | ✓ WIRED | Confirmed empirically: spawned-binary test produces exactly `unknown command "query" for "codegraph"\n` / `unknown command "unlock" for "codegraph"\n` on stderr, empty stdout, exit 1 |
| `test/integration/removed_verbs_test.go` | `test/integration/main_test.go` TestMain/runBinary | go build of cmd/codegraph, exec with fixture dir | ✓ WIRED | Test ran successfully building and exec'ing the real binary |
| `internal/cli/testdata/cli-reference-allowlist.txt` | `internal/cli/cli_reference_test.go` TestEveryRegisteredFlagIsAccountedFor | unmatched entry fails the guard | ✓ WIRED | Test passes with the trimmed allowlist; Family (b) proves the reverse (re-added entries fail it) |
| feat(cli)! commit + 03-01-SUMMARY.md | `.changes/unreleased` Breaking fragment | verify:post changie-fragments capability, `!` subject + SUMMARY user-terms text | ✓ WIRED | Fragment committed with `Changie-Phase: 3` trailer, correct kind and body |
| `.changes/unreleased` Breaking fragment | `task changie -- next auto` | `.changie.yaml` Breaking kind, auto: minor | ✓ WIRED | Command printed `v0.15.0` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Removed verbs are unknown commands, 4/4, executed-count guard | `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v` | `--- PASS` all 4 subtests; `executed 4 of 4 cases` | ✓ PASS |
| Flag-accounting and grouping guards green post-removal | `go test ./internal/cli/ -run '^(TestEveryRegisteredFlagIsAccountedFor\|TestEveryCommandHasGroupID)$' -v` | both `--- PASS` | ✓ PASS |
| Generated CLI reference has no drift | `task docs:cli:drift` | `compared 1 generated file`, byte-identical | ✓ PASS |
| Census instrument reports zero surviving references | `rg -nU -w --hidden 'codegraph\s+(query\|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .` | no output, exit 1 | ✓ PASS |
| `changie next auto` predicts the correct minor bump | `task changie -- next auto` | `v0.15.0` | ✓ PASS |
| Full unit suite green | `GOTOOLCHAIN=go1.26.6 task test:unit` | all packages `ok`/cached, no FAIL lines | ✓ PASS |
| Working tree and worktrees clean after phase | `git status --porcelain`; `git worktree list` | clean; no scratch/census worktree registered | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| VERB-09 | 03-01 | Stubs removed, allowlist edited, docs:cli no change, census zero | ✓ SATISFIED | Truths 1, 2, 4, 6, 7, 11; REQUIREMENTS.md line 60 already `[x]` |
| CAP-05 | 03-01 | Capability firing observed at a real phase close, fragment on milestone branch, dispatch recorded | ✓ SATISFIED | Truth 9; REQUIREMENTS.md line 50 already `[x]`, correctly not reset to pending (D-11 path not taken — the capability fired) |

No orphaned requirements: REQUIREMENTS.md's `Phase 3` mapping (lines 119, 123) lists exactly CAP-05 and VERB-09, both declared in 03-01-PLAN.md frontmatter `requirements: [VERB-09, CAP-05]`.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | none found | — | Scanned `test/integration/removed_verbs_test.go`, `internal/cli/root.go`, `internal/cli/testdata/cli-reference-allowlist.txt`, `03-MUTATION-LOG.md` for TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER — zero matches |

03-REVIEW.md (deep, 4 files reviewed) independently found 0 critical, 0 warning, 1 info (a minor test-table redundancy in `removed_verbs_test.go`, not a defect). 03-SECURITY.md records `threats_open: 0` across 7 registered threats, all closed or accepted with rationale.

### Prohibitions Checked

| Statement | Requirement | Status | Evidence |
|-----------|-------------|--------|----------|
| MUST NOT give removed verbs any afterlife (alias/SuggestFor/hidden/hint) | VERB-09 | ✓ Held | `rg -o -w -e 'newQueryCmd' -e 'newUnlockCmd' --glob '*.go' .` = 0; live binary test shows no suggestion text; Family (a) proves both revival forms are caught |
| MUST NOT pre-write the fragment or present hand-written as capability firing | CAP-05 | ✓ Held | No commit before `d9892beb` touched `.changes/unreleased/`; capability's own trailers present |
| MUST NOT edit tool-owned CHANGELOG.md/.changes/v*.md or regenerate/hand-edit docs/CLI-REFERENCE.md or a golden | VERB-09 | ✓ Held | `git diff --quiet 84778a82 HEAD -- CHANGELOG.md .changes/v0.14.0.md .changes/header.tpl.md docs/CLI-REFERENCE.md internal/cli/cli_reference_test.go` exits 0 |
| MUST NOT alter the census instrument, including excluding .changes/ | VERB-09 | ✓ Held | Instrument re-run byte-identical to CONTEXT D-06's text, confirmed matching `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` per plan acceptance criteria |
| MUST NOT edit the changie capability or amend/rebase its commit | CAP-05 | ✓ Held | `git diff --quiet 84778a82 HEAD -- .gsd/capabilities/changie` exits 0; fragment commit `d9892beb` is a plain object in git history, not amended |

## Human Verification Required

None. All must-haves resolved to VERIFIED or correctly-not-triggered conditionals with direct codebase/command evidence.

## Gaps Summary

No gaps found. Every ROADMAP Phase 3 success criterion (SC1-SC4) and every VERB-09/CAP-05 must-have truth and prohibition was independently re-verified against the live codebase and git history, not merely read from SUMMARY.md's claims:

- SC1/SC2: re-ran the exact test commands and diff-quiet checks myself.
- SC3 (CAP-05): independently inspected the fragment file, its commit, trailers, and branch ancestry, and confirmed no earlier commit pre-wrote it.
- SC4: re-ran `task changie -- next auto` myself and got `v0.15.0`.
- The census (D-06/D-07) was re-run verbatim at current HEAD and confirmed zero hits.
- The commit-history shape (D-04) was independently reconstructed from `git log`, matching exactly one `test(03-01):` commit followed by exactly one `feat(cli)!:` commit with the required `BREAKING CHANGE:` footer.

---

*Verified: 2026-09-26T21:30:00Z*
*Verifier: Claude (gsd-verifier)*
