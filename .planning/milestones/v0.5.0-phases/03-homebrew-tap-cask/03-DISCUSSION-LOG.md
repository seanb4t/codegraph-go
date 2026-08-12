# Phase 3: Homebrew Tap & Cask - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-09
**Phase:** 3-homebrew-tap-cask
**Areas discussed:** Man page source, Cask delivery shape, Broken-cask gate, Tap token & recovery

---

## Area selection

| Option | Description | Selected |
|--------|-------------|----------|
| Man page source | The binary can't generate man pages at all today; adding cobra/doc pulls go-md2man + blackfriday into the shipped binary | ✓ |
| Cask delivery shape | `generate_completions_from_executable` solves BREW-03 natively, but `manpages:` takes archive-internal paths, colliding with D-16 | ✓ |
| Broken-cask gate | BREW-05 asks for a cask `test:` block — casks have no such stanza | ✓ |
| Tap token & recovery | Criterion 5's negative proof and criterion 4's deliberately-failed tap push | ✓ |

**User's choice:** all four.

**Pre-discussion scouting findings that shaped the areas:**
1. `codegraph completion bash|zsh|fish` already works — verified against the real `dist/` binary, not inferred from docs. Cobra 1.10.2 auto-registers it; `root.go` never disables it.
2. No man-page capability exists anywhere — no `spf13/cobra/doc` import, no `man` subcommand.
3. Homebrew Casks have no `test do` stanza (formula-only), and GoReleaser's `homebrew_casks:` reference exposes no `test` field.
4. `seanb4t/homebrew-tap` does not exist (`gh repo view` → could not resolve).

---

## Man page source

### Q1: Where should the man page roff come from?

| Option | Description | Selected |
|--------|-------------|----------|
| `codegraph man` in binary | spf13/cobra/doc + a man subcommand; go-md2man + blackfriday become compiled deps of the shipped binary | ✓ |
| tools/-only generator | A `tools/mangen` main package writes roff at release time; shipped binary's dep tree unchanged | |
| Hand-rolled roff writer | Walk the Cobra tree ourselves; zero new deps, but we own a roff formatter | |

**User's choice:** `codegraph man` in binary.

### Q2: What shape should the `man` command take on the CLI surface?

| Option | Description | Selected |
|--------|-------------|----------|
| Hidden, writes to a dir | `Hidden: true`, takes an output directory; minimal FLAG-PARITY divergence footprint | ✓ |
| Visible, documented divergence | Normal visible command recorded in docs/FLAG-PARITY.md, the `githooks` treatment | |
| Hidden, roff to stdout | Single concatenated document to stdout | |

**User's choice:** Hidden, writes to a dir.

### Q3: How should the two new compiled dependencies be admitted?

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit recorded audit | The v0.1 Phase 5 `[SUS]` grammar treatment — written supply-chain review before the import lands | |
| Ordinary deps + gate check | Import normally, but record the govulncheck/SBOM delta as a phase criterion | |
| Just import them | Ubiquitous, pure Go, already in the module graph; existing govulncheck + SBOM pipeline covers them | ✓ |

**User's choice:** Just import them.
**Notes:** Both appear in `go.sum` today only as `/go.mod` hashes — importing `cobra/doc` promotes them to compiled deps and into the SBOM/govulncheck reachable set for the first time.

### Q4: How many man pages should `codegraph man` emit?

| Option | Description | Selected |
|--------|-------------|----------|
| Full tree (cobra/doc default) | `GenManTree` — ~27 pages, one per command; `man codegraph-explore` works | ✓ |
| Root page only | A single codegraph.1; simpler cask wiring, no per-subcommand pages | |

**User's choice:** Full tree.

---

## Cask delivery shape

### Q1: How should the ~27 man pages reach the user's machine?

| Option | Description | Selected |
|--------|-------------|----------|
| Into the zip, amend D-16 | `before:` hook generates them, zip gains a `files:` entry, cask lists them under `manpages:`; brew tracks and removes them | |
| post-install hook | D-16 stays literally intact; hook runs the installed binary to generate into the prefix; brew does not track the files | ✓ |
| Both: zip + hook fallback | Redundant by construction — the "two writers" shape resolved by deletion during the checksums collision | |

**User's choice:** post-install hook.
**Notes:** Constraint established during discussion — `manpages:` takes paths inside the downloaded archive, and a cask downloads exactly one artifact per platform, so a separate man-only archive is not attachable. The hook is the only mechanism consistent with D-16.

### Q2: How should shell completions be delivered?

| Option | Description | Selected |
|--------|-------------|----------|
| generate_completions_from_executable | Native GoReleaser mechanism, `shell_parameter_format: cobra`, shells [bash, zsh, fish]; runs the installed binary | ✓ |
| Ship completion files in the zip | Symmetric with man-pages-in-zip; brew tracks the files, but they become release-time artifacts | |

**User's choice:** generate_completions_from_executable.

### Q3: How should uninstall clean up files brew doesn't track?

| Option | Description | Selected |
|--------|-------------|----------|
| `uninstall: delete:` | Declarative stanza; runs on every uninstall, not just `--zap` | |
| `hooks.post.uninstall` | Symmetric `system_command` rm — one matched pair with post.install | ✓ |
| `zap: trash:` only | Homebrew's convention for user data, not for artifacts we generated | |

**User's choice:** `hooks.post.uninstall`.

### Q4: Should Phase 3's hook also write a brew-install sentinel for Phase 4?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, write the sentinel now | Explicitly named in the ROADMAP's Phase 3 Notes; the hook is being written anyway | ✓ |
| No, Phase 4 owns detection | Phase 4's criteria demand linuxbrew and custom-prefix coverage a sentinel can't give | |
| Sentinel + path detection | Fast path plus fallback — but two detection paths can independently rot | |

**User's choice:** Yes, write the sentinel now.

---

## Broken-cask gate

### Q1: What should serve as BREW-05's gate, given casks have no `test:` block?

| Option | Description | Selected |
|--------|-------------|----------|
| Post-install hook is the gate | The hook already executes the installed binary, so a broken binary fails `brew install` for the first user | ✓ |
| Post-release CI install job | macOS CI job doing a real tap + install + run | |
| Both — hook + CI job | Different moments (user's machine vs our verification); neither makes the other vacuous | |
| `brew audit --cask` | Never executes the binary, so cannot catch the failure BREW-05 names | |

**User's choice:** Post-install hook is the gate.

### Q2: How should the deliberately-broken binary be constructed to prove RED?

| Option | Description | Selected |
|--------|-------------|----------|
| Corrupt binary in a local cask | Locally-hosted archive with a truncated/non-executable binary; repeatable, so it could be permanent | |
| Wrong-arch binary | linux/amd64 on darwin — closer to the real risk (a bad `ids:` selection) | |
| One-time recorded mutation | The D-07/SIGN-03 pattern: single recorded run, evidence committed, never automated | ✓ |

**User's choice:** One-time recorded mutation.
**Notes:** Tradeoff was stated in the option and accepted — no regression protection, so the hook's positive assertions become the only permanent guard.

### Q3: What positive assertion should the post-install hook carry?

| Option | Description | Selected |
|--------|-------------|----------|
| Assert man files produced | Man output directory non-empty; inverts the "exit 0 with zero output" vacuous pass | |
| Assert version match | `codegraph version` equals the cask version; catches the wrong binary, not just a dead one | |
| Both assertions | Covers "didn't run" and "ran but is the wrong artifact" — two different failures | ✓ |

**User's choice:** Both assertions.
**Notes:** Discharges repo rule `84d1gfpywd`.

### Q4: How should the phase handle criterion 1 needing two releases?

| Option | Description | Selected |
|--------|-------------|----------|
| Block on second release | Most literal reading; the phase's tail waits on release cadence | |
| Automated re-verify, don't block | Permanent post-release cold-install job; the property becomes a standing guarantee | |
| Two releases inside the phase | Cut both within the phase boundary, no external cadence dependency | ✓ |

**User's choice:** Two releases inside the phase.
**Notes:** release-please won't cut from zero releasable commits — the second release needs a real `feat:`/`fix:` commit, identified at plan time.

---

## Tap token & recovery

### Q1: What shape should the tap-writing credential take?

| Option | Description | Selected |
|--------|-------------|----------|
| Fine-grained PAT, tap only | Contents: write on the tap alone; 404 for out-of-scope repos makes the negative proof clean; carries an expiry | |
| GitHub App token | Installed on the tap repo only, short-lived installation token per release; scope set by installation | ✓ |
| Classic PAT with repo scope | Fails criterion 5 outright — it can write seanb4t/codegraph-go | |

**User's choice:** GitHub App token.
**Notes:** Raised and left for research — `release.yml:79-117` records that exactly one job may hold `id-token: write` and that every additional Action inside it is within the OIDC token's reach. Minting inside that job places a new Action next to the cosign identity; minting elsewhere means passing a live write-token between jobs.

### Q2: How should seanb4t/homebrew-tap be bootstrapped?

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal, GoReleaser owns it | Public repo, README + LICENSE; GoReleaser creates Casks/codegraph.rb on first release | ✓ |
| README documents install | Same plus install instructions in the tap README — a second place they can drift | |
| Pre-seed a placeholder cask | Recreates the "second, staler source" problem D-16 exists to prevent | |

**User's choice:** Minimal, GoReleaser owns it.

### Q3: How should criterion 5's negative proof be produced?

| Option | Description | Selected |
|--------|-------------|----------|
| One-time recorded run | Attempt the write, record status and response as committed evidence | ✓ |
| Permanent CI assertion | Runs every release; turns an observation into a standing invariant, but attempts an unauthorized write each time | |
| Assert App installation scope | Positive assertion about the control rather than a probe of its effect; proves configuration, not behavior | |

**User's choice:** One-time recorded run.

### Q4: How should the deliberately-failed tap push be induced?

| Option | Description | Selected |
|--------|-------------|----------|
| Withhold the token | Real release with the App token absent; most realistic failure mode, fails at the last pipe | |
| Point at a nonexistent repo | Deterministic, no credential manipulation, but a config mutation that must be reverted | |
| Local dry-run only | `--snapshot`/local; zero risk, but cannot demonstrate criterion 4's published-release observation | ✓ |

**User's choice:** Local dry-run only.

### Q5 (follow-up): How should criterion 4 be reconciled with proving the failure locally?

Raised because criterion 4 as written requires observing the published Release, cosign bundles, SBOMs and SLSA provenance intact — which a local dry-run cannot produce.

| Option | Description | Selected |
|--------|-------------|----------|
| Split the criterion | Local dry-run proves failure-and-recovery; the existing permanent post-release verification proves release integrity; criterion wording amended | ✓ |
| Amend criterion 4 down | Drop the published-release-intact clause — removes a property rather than relocating it | |
| Keep it, do it live after all | Induce the failure on a real release; risk bounded by `replace_existing_artifacts: true` and patch-forward recovery | |

**User's choice:** Split the criterion.

---

## Claude's Discretion

- Sentinel filename, location and content format (subject to Phase 4 being able to read it, and post-uninstall removing it).
- Man page output directory within `HOMEBREW_PREFIX`, and `/opt/homebrew` vs `/usr/local` portability.
- Cask metadata not discussed: `desc`, `homepage`, `livecheck`, `auto_updates`, `depends_on`.
- Whether the cask covers darwin only or also linuxbrew.
- Hook idempotency on reinstall/upgrade.

## Deferred Ideas

- homebrew-core submission (already BREW-07).
- Stapling / offline-safe first launch (already DIST-06).
- A permanent automated cold-install verification job — considered for criterion 1, not taken.
- A permanent CI assertion that the tap credential cannot write codegraph-go — considered, not taken.
- Phase 4's detection strategy (sentinel vs resolved-symlink vs both) — Phase 4's call.

## Reviewed Todos (not folded)

- `release:dry-run-signed` additions-only diff guard passes vacuously (release, 0.9) — adjacent to D-18's local dry-run; not folded, but the planner should know it is unfixed.
- `verify:self-upgrade` downloads and executes without signature verification (release, 0.9) — accepted as T-02-24 in Phase 2.
- `post-release-verify.yml` event-aware conclusion guard has no test (ci, 0.6) — relevant only via D-19's split.
- Wire oracle `toolslist-repeat` ordering flake (mcp, 0.6) — unrelated.
