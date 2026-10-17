# 05-CENSUS — release-please retirement census and verdicts

**Phase:** 05-release-chain-release-please-retirement
**Date:** 2026-10-14
**Scope:** D-09 (the plain grep census, positive-controlled), D-10 (a recorded verdict for each of docs/RELEASE-PROCEDURES.md's 45 release-please reference lines at base commit 5a8603f5), and REL-14 (no tracked reference outside `.planning/`, `CHANGELOG.md` history, and this milestone's own notes). This file is under `.planning/`, which the census excludes, so it may quote the retired name.

## D-10 verdicts: docs/RELEASE-PROCEDURES.md (45 reference lines at 5a8603f5)

Verdict vocabulary: `rewritten` (the text now describes the changie chain), `removed` (the instruction or rationale no longer applies), `reworded-history` (the history is kept, attributed to "the previous release automation" without the name). Excerpts are verbatim from the base line, cut at 100 characters, with every `|` shown as `/`.

| ID | base line | section | excerpt | verdict | resolution |
|---|---|---|---|---|---|
| R01 | 32 | §1 | > `pretag-gate` job in `.github/workflows/release-please.yml` on every push | rewritten | Section 1 note now names the pretag-gate job in release-pr.yml and release-tag.yml. |
| R02 | 33 | §1 | > to `main`, gating the `release-please` job behind it via `needs:`. The | rewritten | Each workflow gates its own job behind pretag-gate via needs:; the note states both. |
| R03 | 62 | §3 | `main` as release-please.yml's `pretag-gate` job (see the note added to | rewritten | Section 3 item 2 names the pretag-gate job in both new workflows. |
| R04 | 63 | §3 | §1 above) — release-please can never open or merge a release PR from a | rewritten | Now: a release PR can never be opened, and a tag never pushed, from a main that fails the gate. |
| R05 | 67 | §3 | 4. The tag is created by release-please when its release PR merges (§4) — | rewritten | Section 3 item 4: the tag is pushed by release-tag.yml when the release PR merges. |
| R06 | 76 | §3 | planning-only `docs(...)` commits. Under fast-forward, release-please walks | removed | The fast-forward rationale existed so the previous automation could read commit history; fragments carry the changelog now. |
| R07 | 82 | §3 | — discarding exactly the input release-please exists to consume. | removed | Same fast-forward paragraph, dropped with it; section 3 item 3 now says a squash merge is fine. |
| R08 | 92 | §4 | ## 4. Cutting a release (now automated by release-please) | rewritten | Section 4 retitled "Cutting a release (the changie release PR)". |
| R09 | 98 | §4 | 2. `.github/workflows/release-please.yml` runs on that push: its | rewritten | Section 4 step 2 names release-pr.yml and its pretag-gate job. |
| R10 | 100 | §4 | `release-please` runs. release-please then opens or updates a | rewritten | Step 2: the release-PR job runs after pretag-gate and opens or updates the chore(main) release PR. |
| R11 | 105 | §4 | 5. release-please creates the tag **and** the GitHub Release in a single | rewritten | Steps 5 and 6: release-tag.yml pushes the tag; GoReleaser creates the Release in the Publish bullet. |
| R12 | 107 | §4 | release-please's own changelog as the release body. | rewritten | The Release body is .changes/vX.Y.Z.md, passed by release:goreleaser with --release-notes (REL-12). |
| R13 | 137 | §4 | the Release release-please already created, idempotently | rewritten | Publish bullet: GoReleaser creates the Release and uploads into it idempotently. |
| R14 | 139 | §4 | release-please's changelog body or prerelease flag. | rewritten | Publish bullet: body from the changie notes file; prerelease flag follows the tag (prerelease: auto). |
| R15 | 169 | §4 | must never move into `release-please-config.json`'s `release-as` field: a | removed | The override block and its release-as config-key warning are removed; the config file no longer exists. |
| R16 | 174 | §4 | last release, release-please opens **no PR at all**. This is expected, not | rewritten | Expected no-op paragraph: no fragments, so release:batch exits 0 and no PR is opened or updated. |
| R17 | 175 | §4 | a broken pipeline — check the `release-please.yml` workflow run's log | rewritten | Points at the release-pr workflow run log instead of the retired workflow. |
| R18 | 189 | §4 | `v[0-9]*` trigger and still fires the signed build; since no release-please | removed | Removed with the hand-pushed rc escape hatch (D-15); Publish no longer uploads into a pre-created Release. |
| R19 | 272 | §6 | > two commands is not optional for the first release-please-automated | rewritten | Reworded to the first release cut by the changie chain. |
| R20 | 285 | §7 | authority release-please owns (D-06R) — and, more concretely, would destroy | rewritten | Reworded to tag authority the release workflow owns (D-06R). |
| R21 | 288 | §7 | A wrong stable release is fixed by the **next** release-please patch | rewritten | A wrong stable release is fixed by the next patch release cut through the release PR. |
| R22 | 312 | §7 | For an `rc` cleanup, a release-please cut leaves **up to three** artifacts | rewritten | Section 7 now lists the three artifacts a changie-chain release leaves behind. |
| R23 | 318 | §7 | 3. (release-please-cut releases only) the version-bump commit on `main` | rewritten | Artifact 3 is the release commit on main with its merged release PR. |
| R24 | 319 | §7 | (the `CHANGELOG.md` entry plus the `.release-please-manifest.json` | rewritten | The commit carries the CHANGELOG.md entry plus .changes/vX.Y.Z.md; the manifest no longer exists. |
| R25 | 322 | §7 | For a release-please-cut release, deleting the tag alone is not | rewritten | Kept the point that deleting the tag alone is not sufficient, with the changie reason. |
| R26 | 323 | §7 | sufficient — release-please's next run walks `main`'s history and, seeing | removed | The manifest-already-bumped reasoning no longer applies; main still carrying the entry is the new reason. |
| R27 | 336 | §7 | release-please-authored ref, so deleting just the tag is the correct and | rewritten | The lone tag is the one release-tag.yml pushed; deleting it is still the only step for that case. |
| R28 | 347 | §7 | **2. Revert the release commit on `main`:** so release-please's next run | rewritten | Step 2 reverts the release commit through a normal PR; fragments return and release-pr.yml reopens the PR. |
| R29 | 352 | §7 | **3. Check the `autorelease` label:** release-please re-labels its release | removed | The autorelease label step is dropped: the changie chain applies no labels. |
| R30 | 355 | §7 | release-please from proposing a fresh release PR on its next run — remove | removed | Part of the autorelease label paragraph, dropped with it. |
| R31 | 356 | §7 | the `autorelease: tagged` label from the PR (or close and let release-please | removed | Part of the autorelease label paragraph, dropped with it. |
| R32 | 391 | §9 | `release.yml` would simply never fire on release-please's tag. A GitHub | rewritten | Section 9 rationale: the default token starts no release.yml run on the tag release-tag.yml pushes. |
| R33 | 408 | §9 | - **Issues: Read and write** — needed because release-please labels its | reworded-history | Kept as history in one sentence: the previous release automation held Issues write to label its PR; the chain does not use it. |
| R34 | 429 | §9 | settings) authorize everything release-please does with the minted App | rewritten | The installation permissions now authorize everything the chain does with the App token. |
| R35 | 430 | §9 | token. The `permissions:` block inside `.github/workflows/release-please.yml` | rewritten | The permissions: blocks of release-pr.yml and release-tag.yml govern only the default token. |
| R36 | 434 | §9 | time release-please actually tries to label a PR or push a commit, not | rewritten | The 403 now arises when the release-PR job pushes its branch or opens its PR, or the tag job pushes the tag. |
| R37 | 462 | §9 | pull requests. `release-please-action`'s own README calls it out as a | rewritten | The create-and-approve setting is described as a common blocker for default-token PRs, with no tool named. |
| R38 | 463 | §9 | common blocker — but D-02's whole design point is that release-please | rewritten | The release PR is authored with the App token (D-03), so the setting is not load-bearing. |
| R39 | 468 | §9 | by itself block release-please — only an under-scoped App installation | rewritten | A disabled value does not by itself block the release PR; only an under-scoped App does. |
| R40 | 473 | §9 | **must** be added to the bypass-actor list, or release-please will be | rewritten | Branch protection note now describes protect-main and says the chain never pushes to main. |
| R41 | 604 | §11 | / (d) A public supply-chain claim is genuinely absent / Either the byte-identity chain (SIGN-04) or | rewritten | Patch-forward remedy: cut a new patch release through the normal release PR flow. |
| R42 | 628 | §12 | release-please App** documented in §9 above (`APP_ID`/`APP_PRIVATE_KEY`), | rewritten | The tap App is a separate App from the release GitHub App. |
| R43 | 629 | §12 | not a reuse of it: the release-please App is installed on | rewritten | The release GitHub App is installed on this repository and authors the release PR branch, PR and tag. |
| R44 | 647 | §12 | release-please App, applied to the tap App instead: | rewritten | Key rotation follows the shape of the release GitHub App section. |
| R45 | 662 | §12 | §9's troubleshooting note describes for the release-please App): a | rewritten | The identity-check pattern is described for the release GitHub App. |

## REL-14 census instrument

Recorded verbatim, not committed as a Taskfile target or script. REL-14 is a one-time retirement proof; the instrument is
re-run from this file the way Phase 3 re-ran v0.14.0's (04-02 precedent), and a permanent CI target would be ceremony D-01
asks to avoid. It searches tracked content only (`git grep`) and tracked filenames (`git ls-files`); untracked local caches
such as a tool's `.boost/` are not repository content and would make the result machine-dependent.

```bash
census() {
  git -C "$1" grep -n -i -I -E 'release[-_ ]?please' -- . ':(exclude).planning/' ':(exclude)CHANGELOG.md' ':(exclude).changes/v*.md' ':(exclude).changes/unreleased/' 2>&1
  f=$(git -C "$1" ls-files) || { echo "census: listing tracked files failed in $1"; return; }
  printf '%s\n' "$f" | rg -i 'release[-_ ]?please' | rg -v '^\.planning/'
}
```

Exclusions, one line per REL-14 clause. Nothing else is excluded, and the filename half excludes `.planning/` only:

- `.planning/` is `':(exclude).planning/'`
- `CHANGELOG.md` history (D-09) is `':(exclude)CHANGELOG.md'` and `':(exclude).changes/v*.md'`
- this milestone's own notes (D-11) is `':(exclude).changes/unreleased/'`

D-11 is the maintainer's decision (2026-10-14) that `.changes/unreleased/` is the concrete path of REL-14's and D-09's
"this milestone's own notes" clause. Between releases that directory holds exactly the fragments `changie batch` turns into
the next release's notes, which here are v0.15.0's. Once batched they become `.changes/v0.15.0.md`, which D-09 already counts as
`CHANGELOG.md` history. Per D-11 no other `.changes/` path is excluded: `.changes/header.tpl.md` stays in scope, and the
positive control below proves the boundary in both directions.

Pass condition on a clean tree: `census <dir>` prints nothing. Exit codes are ignored (the last `rg -v` exits 1 on empty
input), so callers capture with `x=$(census <dir>; true)`. An error message is output and therefore a failure.

## Positive control

Rule 84d1gfpywd: the instrument is shown to find planted references, and to skip planted references under the excluded paths,
before its zero is trusted. Scratch worktree: a detached checkout of the post-deletion HEAD (`92a4ce15`) at a path containing
`p05-scratch-wt`. Five plants are written and staged with `git add -f`: in-scope content `docs/__census_control__.md`;
in-scope `.changes/` content, two lines prepended to the existing `.changes/header.tpl.md` (`planted Release Please reference`,
`planted RELEASEPLEASE reference`); in-scope filename `scripts/__release_please_control__.txt` (empty file); excluded
`.planning/__census_control__.md`; excluded `.changes/unreleased/__census_control__.yaml`. Excluded plants hold
`planted release-please reference`.

```
worktree HEAD: 92a4ce15
--- staged plants (git status --porcelain):
M  .changes/header.tpl.md
A  .changes/unreleased/__census_control__.yaml
A  .planning/__census_control__.md
A  docs/__census_control__.md
A  scripts/__release_please_control__.txt
--- unexcluded grep -l over the two excluded plant paths:
.changes/unreleased/__census_control__.yaml
.planning/__census_control__.md
--- census "$W":
.changes/header.tpl.md:1:planted Release Please reference
.changes/header.tpl.md:2:planted RELEASEPLEASE reference
docs/__census_control__.md:1:planted release-please reference
scripts/__release_please_control__.txt
--- revert: git reset -q --hard HEAD
--- git status --porcelain --ignored --untracked-files=all (expect empty):
[]
```

Reading: every plant landed (the staged list shows all five), and the unexcluded grep lists both excluded plants, so their
absence from the census output is a real exclusion, not a vacuous one. The instrument prints exactly four lines. The two
`.changes/header.tpl.md` lines show an in-scope `.changes/` file is searched, with case-insensitivity (`Release Please`,
`RELEASEPLEASE`) and the space and empty separators. The docs line shows the `-` separator in content; the scripts line shows the
filename half and the `_` separator. The `.changes/unreleased/` and `.planning/` plants are absent, so both exclusions hold.
After `git reset -q --hard HEAD` the status with `--ignored --untracked-files=all` is empty; `git worktree remove --force "$W"` and
`git worktree prune` followed, leaving only the main checkout and the pre-existing unrelated worktree.

## Clean-tree census

On the real tree at `92a4ce15`, which has the three retired files deleted by `ffd84c4a`:

```
$ x=$(census .; true); printf "[%s]\n" "$x"
[]
test ! -e .github/workflows/release-please.yml -> absent (ok)
test ! -e release-please-config.json -> absent (ok)
test ! -e .release-please-manifest.json -> absent (ok)
$ git grep -l -i -I -E 'release[-_ ]?please' -- CHANGELOG.md ':(glob).changes/v*.md'
.changes/v0.2.0.md
CHANGELOG.md
```

The census prints nothing. The history files still hold exactly the hits the history exclusion exists to skip
(`.changes/v0.2.0.md` and `CHANGELOG.md`), so that exclusion skips something real. Those files are history (D-09) and were
not edited. `.changes/unreleased/` holds no hit on the real tree today; the exclusion is shown to work by the planted file above.

Version source of truth: `changie latest` is the version source, and no tracked version manifest remains.

```
$ task changie -- latest
v0.14.1
$ ls .changes/v*.md | sort -V | tail -1
.changes/v0.14.1.md
```

The remaining tracked paths whose names contain "manifest" (`corpora/manifest.json`, `internal/agents/manifest.go`,
`internal/corpora/manifest.go`, `tools/bench/realcorpus/manifest.go`, `web/build/.build-manifest`, and their tests) are
unrelated to versioning.

## Phase gate

Run on the real tree after the deletion commit (`ffd84c4a`) and the mutation-log commit (`92a4ce15`). Go commands are prefixed
`GOTOOLCHAIN=go1.26.9`.

```
$ GOTOOLCHAIN=go1.26.9 task test
(exit 0; 66 "ok" lines, no FAIL line; includes test:golden, test:daemon and test:race)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.562s
ok  	github.com/seanb4t/codegraph-go/internal/daemon	64.478s
ok  	github.com/seanb4t/codegraph-go/testdata/golden	22.742s

$ task check:changie
check:changie: 14 of 14 checks passed against a scratch copy (source tree byte-unchanged)

$ task check:fragment-policy
check:fragment-policy: executed 46 of 46 defined test methods (floor 46)

$ task lint:actions
(no finding, exit 0)

$ GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.587s
```

REL-12 single-hunk check, `git diff -U0 7b12cb92c407ae7e495a62b4ac38d8685e64307c -- .github/workflows/release.yml`:
`hunks: 1`, and `non-comment changed lines: []` (the one hunk changes comment lines only).

Byte-lock against `5a8603f5` (a lock on history and `header.tpl.md`, not a census pathspec):
`git diff --quiet 5a8603f5 HEAD -- internal/upgrade/verify.go CHANGELOG.md ':(glob).changes/v*.md'` exits 0, and
`git diff --quiet 5a8603f5 -- internal/upgrade/verify.go CHANGELOG.md ':(exclude).changes/unreleased' .changes` exits 0.

No commit since `5a8603f5` carries `[skip ci]` or `[ci skip]` (checked over `git log --format=%B 5a8603f5..HEAD`). No
`p05-scratch-wt` worktree remains. `gh api repos/seanb4t/codegraph-go/git/matching-refs/tags/v0.15 --jq 'length'` prints `0`
(read-only): this phase created no tag. `git ls-remote origin refs/heads/main` still prints `7b12cb92c407ae7e495a62b4ac38d8685e64307c`,
so `main` has cut no release since the phase began and no verbatim seed is needed for the milestone merge.
