# Phase 4: `codegraph upgrade` × Homebrew - Context

**Gathered:** 2026-08-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Neither install path lies about what is installed. `codegraph upgrade` recognizes a
Homebrew-managed install from the resolved location of its own binary, refuses with an
actionable pointer at `brew upgrade codegraph`, and never reaches the code that could
mutate the install behind brew's back. `--check` steps aside the same way rather than
answering a question brew owns.

**In scope:** brew-managed detection in `internal/upgrade`; the refusal branch in
`upgrade.Run()` and its exit semantics; the `--check` behavior under a brew install; the
constructed-tree tests that prove detection fires and does not false-fire; removal of the
Phase-3 sentinel and its assertions; amendment of ROADMAP criteria 1–3 and REQUIREMENTS
UPGR-02 wording to name the layout that actually ships; acceptance against the real
Phase-3 tap.

**Out of scope:** anything about the cask's own content or publication (Phase 3, shipped);
homebrew-core submission (BREW-07); stapling / offline-safe first launch (DIST-06);
teaching `codegraph upgrade` any new install channel.

</domain>

<decisions>
## Implementation Decisions

### Detection: what the criteria should say (UPGR-02)

- **D-01:** **ROADMAP criteria 1–3 and REQUIREMENTS UPGR-02 are amended from `Cellar` to
  `Caskroom`**, and the detector recognizes **both** trees. Phase 3 shipped a
  `homebrew_casks:` block, and a cask stages into
  `$HOMEBREW_PREFIX/Caskroom/<token>/<version>/` with `$HOMEBREW_PREFIX/bin/<name>`
  symlinked into it — `Cellar/` is the *formula* tree and `/opt/homebrew/Cellar/codegraph`
  has never existed for this project (measured: `03-02-SUMMARY.md:128` records the
  sentinel resolving to `/opt/homebrew/Caskroom/codegraph/<version>`; `03-EVIDENCE.md:160`
  records `ls /opt/homebrew/Caskroom/codegraph`).

  **This is the fifth falsified scoping assumption in this milestone**, same class as
  Phase 3's D-09 (`test do` is a formula stanza with no cask equivalent) — the Phase-4
  criteria were written before `homebrew_casks:`-not-`brews:` was corrected. It is
  load-bearing: a detector matching `Cellar` literally would never fire on our own
  install, and criterion 3's false-positive leg would be probing a string that never
  appears — precisely the vacuous-gate shape repo rule `84d1gfpywd` exists for.

  Cellar/formula detection is kept even though this project ships no formula, so a future
  formula path or a hand-built brew install is not silently invisible.

- **D-02:** **Detection is structural only, and the Phase-3 sentinel is REMOVED** rather
  than left as dead weight. Phase 3's D-08 deliberately left "sentinel vs structural vs
  both" to Phase 4; the answer is structural, so the artifact that existed solely to serve
  Phase 4 goes with it. Rejected: `structural OR sentinel` (a second signal whose only
  effect is to hide a structural bug behind it on every real install) and `sentinel
  primary` (degenerates criterion 3's constructed-tree test into planting a file).

  **Removal is safe, no stranding:** the sentinel lives *inside*
  `Caskroom/codegraph/<version>/`, which Homebrew purges wholesale on both successful
  uninstall and failed-install rollback (`03-05-SUMMARY.md:49,195`). Nothing survives on a
  user's machine that the `rm_f` was needed to clean.

  **Removal touches four places** (the planner should treat this as one atomic change):
  - `.goreleaser.yaml:570-588` — the `hooks.post.install` sentinel write
  - `.goreleaser.yaml:623-624` — the `hooks.post.uninstall` `FileUtils.rm_f`
  - `Taskfile.yml:1996-2028` — `release:rehearse-cask` Step 5b, which positively asserts
    the sentinel's `schema=1` line and all six keys
  - any `goreleaser_shape_test.go` / `taskfile_shape_test.go` assertion pinning those
    blocks — search before editing, do not assume none exist

  **BREW-05's install gate is untouched.** D-11's two positive assertions (man output
  non-empty, reported version equals declared version) never involved the sentinel, and
  they remain the mechanism that makes `brew install` fail loudly for the first user.

  — **Reversibility:** costly — re-introducing a sentinel later means every already-installed
  user is undetectable-by-sentinel until they reinstall, which is the exact hazard Phase 3's
  D-08 recorded. Structural detection has no such dependency, which is part of why it wins.

- **D-03:** Detection = **path-shape match AND a Homebrew-authored install receipt.**
  Resolve `os.Executable()` through symlinks, require a `Caskroom/<token>/<version>/` or
  `Cellar/<name>/<version>/` segment shape under *any* prefix, **and** require the
  Homebrew-written `INSTALL_RECEIPT.json` at the matching ancestor.

  **Measured on this machine, not assumed:**
  - cask: `<prefix>/Caskroom/<token>/.metadata/INSTALL_RECEIPT.json` (plus
    `.metadata/<version>/<timestamp>/Casks/<token>.json`); the versioned dir holds only
    the payload
  - formula: `<prefix>/Cellar/<name>/<version>/INSTALL_RECEIPT.json` and a `.brew/` dir

  This is what defeats criterion 3's false-positive leg **with evidence rather than a
  cleverer string test** — a non-brew binary under a directory literally named `Caskroom`
  has no receipt above it. It also discharges repo rule `84d1gfpywd` at the detector level:
  the guard carries a positive assertion that it matched a real Homebrew install, not only
  the absence of a bad shape. Prefix-agnostic and offline; costs one extra `stat`.

  Rejected: name-shape only (still a guess, and the false-positive leg would rest on
  segment-count pedantry) and shelling out to `brew --prefix` / `brew list` (breaks
  criterion 4's brew-absent machine, adds a subprocess to every upgrade, and tests
  Homebrew's CLI rather than our wiring).

- **D-04:** ~~**Linuxbrew stays in criterion 3, scoped to the Cellar shape only,** with an
  inline note recording why. Homebrew on Linux does not support casks, so the Caskroom leg
  is unreachable there by construction.~~ **PREMISE FALSIFIED BY RESEARCH AND REPLACED
  2026-08-11, before planning was drawn over it.**

  The rationale above rested on "Homebrew on Linux does not support casks", which CONTEXT.md
  itself flagged as *asserted, not measured*. Research confirmed it is **false for the
  current Homebrew release line** — and it is the **sixth falsified scoping assumption in
  this milestone**:
  - Homebrew PR #19121 ("feat: allow linux binaries in casks") allows casks containing only
    `binary` or `zap` stanzas to install on Linux.
  - Homebrew 6.0.0 (2026-06-11 release notes) shipped four further Linux-cask items —
    explicit Linux cask requirements (#21909), Caskroom using the user's primary group on
    Linux (#22202), Linux checksum variations for casks (#22632), AppImage support.
  - `brew --version` on this machine is `6.0.16-2` — the same major line.
  - codegraph's own cask (`.goreleaser.yaml:495-523`) declares `binaries: [codegraph]` with
    **no `app`/`pkg`/`zap` stanza** — exactly the shape PR #19121 made Linux-installable.

  The belief was not mistaken when formed — a 2022 Homebrew maintainer statement did say
  "Casks are only for macOS" (Homebrew discussion #3999). It **rotted**. Flagging it as
  unmeasured is what caught it.

  **D-04R (replacement, maintainer decision 2026-08-11): the linuxbrew constructed-tree test
  covers BOTH the Caskroom and Cellar shapes,** and the ROADMAP/REQUIREMENTS amendment drops
  the "unreachable by construction" claim rather than carrying it forward with a footnote.
  Nearly free — detection is already prefix-agnostic (D-03), so this is one more table row,
  not new detection logic. Rejected: keeping Cellar-only with a corrected-but-weaker
  rationale (the test would then cover less than what Homebrew can actually produce for our
  own cask), and measuring real Linuxbrew installability first (new scope, and it tests
  Homebrew's Linux cask support plus Phase 3's publication surface — a direct D-13 violation).

  - **Open, deliberately not closed (research assumption A2):** whether codegraph's cask —
    with its `hooks.post.install`/`post.uninstall` Ruby blocks and
    `generate_completions_from_executable`, beyond the bare `binary` stanza #19121 names —
    actually installs end-to-end on a real Linuxbrew host. **Not measured, and deliberately
    not scoped:** the detector's correctness does not depend on the answer, and measuring it
    is D-13-prohibited. Recorded so a later reader does not mistake silence for evidence.

### Refusal semantics (UPGR-01)

- **D-05:** Bare `codegraph upgrade` under a brew-managed install **returns an error and
  exits non-zero.** Matches `checkWritable`'s precedent at `upgrade.go:119` — "you asked
  for a mutation I will not perform". A provisioning script or CI step that shells out to
  `codegraph upgrade` notices instead of believing it upgraded. Rejected: exit 0 with an
  informational line (matching the same-version no-op at `upgrade.go:112`) because exit 0
  asserts the upgrade happened; and a dedicated exit code, which would introduce an
  exit-code vocabulary this CLI does not have anywhere.

- **D-06:** **`--force` is powerless against the refusal.** It keeps its narrow documented
  meaning — "reinstall even if already on the latest version" (`upgrade.go:57-62`) — and
  gains no new power. Rejected: `--force` overriding with a warning, and a separate
  `--ignore-homebrew` flag; both produce the same end state, a Caskroom whose contents
  disagree with brew's own receipt, so `brew list --cask --versions` lies and the next
  `brew upgrade` silently overwrites the user's forced install. There is deliberately **no
  escape hatch**: `brew uninstall --cask codegraph` is the honest path.

- **D-07:** The refusal message carries **the symlink-resolved install path plus the exact
  command**, e.g.
  `codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph`.
  Naming the resolved path makes a misfire self-diagnosing — a user who is *not* on brew
  can see immediately what we matched, and so can we from a bug report. That is the whole
  concern of criterion 3. Rejected: a terse command-only message (no diagnostic value when
  detection is wrong) and prefixing `brew update &&` (redundant — brew auto-updates on
  `upgrade` by default).

- **D-08:** **Criterion 1's proof is amended from a sha256/mtime comparison to a
  seam-based assertion.** The standing test asserts the refusal fires, names
  `brew upgrade codegraph`, and **neither `Options.download` nor `Options.swap` is ever
  invoked** — driven through the injectable func-vars already in `upgrade.go:38-43`.

  **Maintainer ruling, fifth sighting of `za9ms2mvjh` this milestone.** The orchestrator
  proposed hashing the Caskroom binary before and after; the maintainer's response was
  "we're trying to test homebrew again". The proposal failed the ownership test's third
  question — *would the next natural run notice it anyway?* A sha256 comparison is an
  indirect proxy for "did our code call `swap`", and it drags Homebrew's filesystem and
  rollback semantics into a test of our own branch. The seam assertion tests our control
  flow directly and re-fires forever.

  The real-tap acceptance run is retained (see D-09), but it now only has to **observe the
  refusal** — the part that genuinely requires a real install.

### Read-only path (UPGR-03)

- **D-09:** **`--check` under a brew-managed install does not check.** It steps aside with
  the same pointer rather than resolving a version — "this is what brew is for" (maintainer,
  2026-08-10). `brew outdated codegraph` owns version knowledge for brew installs, and
  duplicating it creates two sources that can disagree.

  **This is a criterion amendment, not a requirement change.** REQUIREMENTS.md UPGR-03 says
  `--check` *"still works under a brew-managed install — read-only, no mutation — and
  reports how to upgrade"*; it never says "reports the available version". The ROADMAP
  criterion added that. Amend ROADMAP criterion 2 to match UPGR-03's actual wording.

  Rejected: reporting the GitHub Release version (honest wording could not stop it from
  answering a question brew owns), and reading the tap's `Casks/codegraph.rb` (a parser for
  a file GoReleaser generates — testing GoReleaser's rendering, the `za9ms2mvjh` objection,
  and it breaks silently when the template changes).

- **D-10:** **`--check` exits zero.** It is a query and nothing failed; bare `upgrade`
  stays non-zero because it asked for a mutation we declined. The two exit behaviors on one
  command must be documented in `--help` and `docs/`. Rejected: uniform non-zero, which
  makes `if codegraph upgrade --check` unusable and reports failure for successfully
  answering a question.

### Placement and ordering

- **D-11:** **Detection fires first in `Run()`, before `resolveLatest`.** This falls out of
  D-09: if neither `--check` nor bare `upgrade` needs a version under brew, the refusal
  needs zero network and works offline. It also keeps criterion 4 trivially satisfied — a
  brew-absent machine is unaffected because detection never shells out to brew (D-03),
  it only stats the filesystem.

- **D-12:** Detection and the refusal live in **`internal/upgrade`, as a new
  `brew.go` / `brew_test.go` pair.** That package already owns install-shape knowledge
  (`checkWritable`, `atomicSwap`, `releaseAssetName`), `Run()` gains one early branch, and
  the seam-based tests sit next to the seams they drive. Rejected: a new `internal/brew`
  package (a package boundary for one predicate, splitting install-shape knowledge across
  two places) and — argued against explicitly — `internal/cli/upgrade.go`, which would leave
  `upgrade.Run()` willing to download and swap over a Caskroom when called directly, putting
  the guarantee outside the seam that enforces everything else and out of reach of D-08's test.

### Standing constraint on all folded work

- **D-13:** **Only test, fix, or address things we own** (maintainer, restated 2026-08-10 —
  fifth statement this milestone, `za9ms2mvjh`). This binds the folded todos below
  individually, not just the phase in aggregate. Before proposing any verification, answer
  all three questions: who owns the code this exercises; could we fix it or only notice it;
  **and would the next natural run notice it anyway** — the third is the one that sank
  Phase 3's G1 and this phase's sha256 proposal.

### Claude's Discretion

- The exact construction of the test fixtures — `t.TempDir()` layout, receipt file contents,
  how the four prefixes (`/opt/homebrew`, `/usr/local`, custom, linuxbrew) are simulated.
  Constraint: the false-positive case must be an **executing test**, not a comment
  (criterion 3 says so explicitly).
- The exported/unexported shape of the detection API in `brew.go` and whether it returns a
  struct (path, token, version) or a bool plus path.
- Exact wording of the `--check` step-aside line, subject to D-07's resolved-path rule and
  UPGR-03's "reports how to upgrade".
- Whether the receipt check reads any field out of `INSTALL_RECEIPT.json` or only asserts
  its existence. Existence is sufficient for D-03; parsing it is optional and adds a
  dependency on a format Homebrew owns.
- How `docs/` and `--help` record the two exit behaviors (D-10).

### Folded Todos

All three folded with D-13 applied individually — each fix is scoped to our side of the
boundary, and the planner must not let any of them drift into testing Homebrew or Sigstore.

- **`03-EVIDENCE.md` falsely claims a failed install can strand the Phase-4 sentinel**
  (docs, `.planning/todos/2026-08-10-03-evidence-falsely-claims-a-failed-install-can-strand-the-phase-4-sentinel.md`).
  D-02 deletes the sentinel outright, so the todo's subject evaporates — the false claim is
  corrected or retired as part of the removal rather than left describing an artifact that
  no longer exists. **We own:** our doc, our artifact, our false statement. Nothing to
  scope-guard.

- **Post-install man-page assertion can be satisfied by stale pages from a prior failed
  install** (release, UF-5 from `03-SECURITY.md:75`). Lives in the exact `hooks.post.install`
  block D-02 already opens, so it is one extra edit in a file being touched anyway.
  **We own:** the assertion body in our cask hook and our `release:rehearse-cask` target.
  **Must not drift into:** asserting Homebrew's rollback behavior. The fix is a pre-install
  baseline (or writing where we control cleanup) on our side — not a test of what brew purges.

- **`verify:self-upgrade` downloads and executes a prior release binary with no signature
  verification** (release, accepted as T-02-24 in Phase 2). **We own:** our Taskfile target,
  executing our own release binary without checking our own cosign signature.
  **Must not drift into:** re-testing Sigstore. This is verifying our signing wiring, using
  the verification `internal/upgrade` already performs.
  **Planner note:** `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` anchors the
  cosign SAN to `.github/workflows/release.yml@refs/tags/v[0-9]*` — reuse it, do not
  hand-roll a second policy that can drift.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements — all three need amendment
- `.planning/ROADMAP.md` § "Phase 4: `codegraph upgrade` × Homebrew" — goal, the
  Depends-on clause (acceptance requires the real Phase-3 tap; a synthetic layout proves
  the mechanism, not that it matches what ships), and the four success criteria.
  **Criteria 1–3 require amendment per D-01 (Cellar→Caskroom), D-08 (seam proof, not
  sha256/mtime) and D-09 (`--check` does not report a version).**
- `.planning/REQUIREMENTS.md:40-42` — UPGR-01, UPGR-02, UPGR-03. **UPGR-02's wording
  requires amendment per D-01.** Note UPGR-03 is already correct as written and is what
  ROADMAP criterion 2 should be amended *toward*, not away from.
- `.planning/PROJECT.md` § "Key Decisions" — "`codegraph upgrade` refuses under a
  brew-managed install, detected by resolving symlinks to the real Cellar path — never a
  path-prefix guess." The mechanism stands; the word `Cellar` is subject to D-01.

### The Phase-3 contract this phase consumes and dismantles
- `.planning/phases/03-homebrew-tap-cask/03-CONTEXT.md` — D-08 (the sentinel and the
  explicit deferral of Phase 4's detection strategy), D-10/D-11 (BREW-05's install gate,
  which must survive D-02 untouched), D-07 (hook symmetry).
- `.goreleaser.yaml:412-457` — the `homebrew_casks:` block's discretionary-metadata
  comments, including "Phase 4 teaches it to refuse when brew-managed (D-08's sentinel)"
  at line 451, which D-02 falsifies and which must be corrected in the same change.
- `.goreleaser.yaml:524-632` — `hooks.post.install` / `hooks.post.uninstall`. Lines
  570-588 write the sentinel; 623-624 remove it; 542-568 are BREW-05's gate and stay.
- `Taskfile.yml:1996-2028` — `release:rehearse-cask` Step 5b, the sentinel assertions to
  be removed. Note the comment there reads "schema=2" while the hook writes and the
  assertion checks `schema=1` — a pre-existing inconsistency, moot once removed.
- `.planning/phases/03-homebrew-tap-cask/03-05-SUMMARY.md:49,195-196` — the measured
  asymmetry: Homebrew's install-failure rollback purges the Caskroom versioned directory
  but never invokes the cask's own uninstall hook, so hook-written files *outside* the
  Caskroom survive. This is the evidence D-02's no-stranding claim rests on.
- `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md:160-165,273-274,801-833` — the
  recorded Caskroom paths (D-01's evidence) and the stale-man-page finding (folded todo 2).
  **Contains the false sentinel-stranding claim to be corrected.**
- `.planning/phases/03-homebrew-tap-cask/03-SECURITY.md:75` — UF-5, the stale-man-page
  assertion in full.

### The code being extended
- `internal/upgrade/upgrade.go` — `Run()` at :82 (the resolve→check?→writable→download→
  verify→swap sequence D-11 branches at the head of); the four injectable func-vars at
  :38-43 that D-08's proof drives; `Options.Force`'s documented meaning at :57-62 (D-06);
  `checkWritable`'s refusal precedent at :119 (D-05's model).
- `internal/cli/upgrade.go` — `newUpgradeCmd()`, the `--check` / `--force` flag wiring and
  the `upgradeRunFunc` package-level seam. Where D-10's exit-code documentation lands.
- `internal/upgrade/upgrade_test.go` — the existing fake-driven `Run()` tests; D-08's
  seam assertion belongs alongside them, in the same idiom.
- `internal/upgrade/verify.go` — `releaseWorkflowRefPattern`; reused by folded todo 3.
  Renaming `release.yml` or changing its tag trigger breaks every user's upgrade.

### External documentation to verify at research time
- Homebrew's own docs on Linux cask support — **D-04 depends on "linuxbrew has no casks"
  and it is currently asserted, not measured.**
- Homebrew's `INSTALL_RECEIPT.json` location and stability for both casks and formulae —
  D-03 rests on the measured layout above; confirm it is not an implementation detail that
  moves between Homebrew versions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`Options.download` / `Options.swap` func-vars** (`internal/upgrade/upgrade.go:38-43`) —
  already built for exactly this kind of assertion. D-08's entire proof is one fake that
  records whether it was called. No new test infrastructure.
- **`checkWritable`** (`upgrade.go:119`) — the in-package precedent for an early,
  error-returning refusal inside `Run()`. D-05 and D-11 both follow its shape.
- **`upgradeRunFunc`** (`internal/cli/upgrade.go:17`) — the CLI-level seam for asserting
  flag wiring without touching the network.
- **`Check: true` early return** (`upgrade.go:98-105`) — already structurally read-only, so
  UPGR-03's "no mutation" needs no new mechanism, only a branch placed above it.

### Established Patterns
- **Repo rule `84d1gfpywd`** — a guard must carry a positive assertion it did its work.
  D-03's receipt requirement is this phase's application at the detector level; D-08's
  seam assertion is its application at the test level.
- **Measured, not assumed** — every layout claim in this document was probed on a real
  Homebrew installation or read out of Phase 3's recorded evidence. Continue that: the two
  research items in canonical refs are the outstanding unmeasured claims.
- **`Taskfile.yml` is the single definition of every CI job body**
  (`TestWorkflowRunBodiesInvokeTask`). Any new gate goes in a task target, not inline in a
  workflow — and any *removed* gate must be removed from the Taskfile, not just the workflow.
- **Shape tests pin config invariants** (`goreleaser_shape_test.go`,
  `taskfile_shape_test.go`). D-02's removal must check for and update assertions there
  rather than leaving a shape test asserting a block that no longer exists.
- **Amendments must be scoped to every place a claim appears, not just the normative one**
  — Phase 3's UF-1 (`xkbc8m36hm`): the ROADMAP index bullet stayed stale because amendments
  landed only on the criteria. D-01/D-08/D-09 amend criteria; check the Phase-4 index
  bullet and the Notes block too.

### Integration Points
- `internal/upgrade/brew.go` (new) → called from `Run()`'s head (D-11/D-12).
- `.goreleaser.yaml` cask hooks → sentinel write/remove deleted (D-02); the comment at
  :451 corrected.
- `Taskfile.yml` `release:rehearse-cask` → Step 5b deleted (D-02); the man-page baseline
  added (folded todo 2).
- `docs/` + `--help` → the two exit behaviors (D-10) and the brew-install upgrade path.

</code_context>

<specifics>
## Specific Ideas

- The refusal message names the **resolved** path, not the invoked one — the diagnostic
  value is entirely in showing what detection actually matched.
- "This is what brew is for" is the governing instinct for the whole phase: where Homebrew
  already owns an answer (what version is available, whether an upgrade exists), codegraph
  points at it rather than computing a second answer that can disagree.
- The ownership test is three questions, not two. The third — *would the next natural run
  notice it anyway?* — is the one that has now killed two proposed verifications in this
  milestone.

</specifics>

<deferred>
## Deferred Ideas

- **A `codegraph doctor` / install-provenance command** — surfaced while weighing whether
  detection deserves its own `internal/brew` package. A standalone predicate would be
  reusable by such a command; there is no such command and none is scoped. Revisit if one
  is ever proposed, at which point extracting the predicate is a cheap refactor.
- **Detecting other managed install channels** (a future formula, a distro package, `go
  install`) — the same "don't lie about what's installed" property generalizes, but nothing
  ships those channels today. Cellar/formula detection (D-01) is the only piece kept
  speculatively, and only because it is free.

### Reviewed Todos (not folded)

Surfaced by `todo.match-phase 4` (10 matches, all score ≤ 0.6) and reviewed. Three were
folded (see Folded Todos above); the rest were not:

- **`brew trust` instructions recommend the broader `--tap` grant with no security framing**
  (docs, 0.6) — genuinely adjacent (it is the published brew install path) but it is Phase
  3's documentation defect, not detection or refusal work. Unfixed; the planner should know
  the published contract still instructs users to opt out of a security control.
- **Tap App secret-distinctness test is tautological** (testing, 0.6) — Phase 3's AR-01;
  compares two in-test constants and reads no workflow. Out of scope, and Phase 3's own
  validation already filed it.
- **`release:dry-run-signed`'s additions-only diff guard passes vacuously** (release, 0.6)
  — Phase 2 residue, already reviewed-and-not-folded in `03-CONTEXT.md`. Still unfixed.
- **`post-release-verify.yml`'s event-aware conclusion guard has no regression assertion**
  (ci, 0.6) — Phase 2 residue. Not touched by this phase.
- **Wire oracle `toolslist-repeat` response ordering flake** (mcp, 0.6) — unrelated.
- **Author a codegraph usage skill for agents** (agents, 0.6) — unrelated; keyword match only.
- **Add golangci-lint with gofmt and idiomatic Go linters** (ci, 0.4) — real and repo-wide
  (memory `skn731h5qj`: this repo has **no gofmt gate anywhere** — `task lint` is vet +
  lint:actions, no `.golangci.yml`, no workflow references gofmt). It is its own change,
  not Phase 4's; folding it would put a repo-wide lint rollout inside a three-requirement
  phase.

</deferred>

---

*Phase: 4-`codegraph upgrade` × Homebrew*
*Context gathered: 2026-08-10*
