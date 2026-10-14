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
