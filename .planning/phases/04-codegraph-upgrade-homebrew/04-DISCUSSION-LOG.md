# Phase 4: `codegraph upgrade` × Homebrew - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-10
**Phase:** 4-`codegraph upgrade` × Homebrew
**Areas discussed:** Detection signal & precedence, Refusal semantics & `--force`, What `--check` reports under brew, Where the refusal fires in `Run()`

---

## Detection signal & precedence

### Q1 — Caskroom vs Cellar criteria mismatch

Raised by the orchestrator before the first question, after measuring Phase 3's own
evidence: the shipped `homebrew_casks:` block installs into
`/opt/homebrew/Caskroom/codegraph/<version>/`, while ROADMAP criteria 1–3 say `Cellar`
five times.

| Option | Description | Selected |
|--------|-------------|----------|
| Amend to Caskroom; detect both trees | Amend criteria + UPGR-02 to Caskroom, detector still recognizes a Cellar-shaped install; false-positive test probes both literals | ✓ |
| Amend to Caskroom only | Narrowest, matches shipped reality exactly; silently blind if a formula path ever appears | |
| Keep Cellar wording, detect both | Leave roadmap text alone, note the discrepancy | |

**User's choice:** Amend to Caskroom; detect both trees
**Notes:** Recorded as the fifth falsified scoping assumption in this milestone, same class
as Phase 3's D-09.

### Q2 — Which signal decides "brew-managed"

| Option | Description | Selected |
|--------|-------------|----------|
| Structural OR sentinel — either fires | Two independent positive signals; sentinel enriches the message | |
| Structural only — ignore the sentinel | One mechanism, exactly what criterion 3 tests; sentinel becomes dead weight | ✓ (amended) |
| Sentinel primary, structural fallback | Exact and cheapest, but degenerates the constructed-tree test | |

**User's choice:** "structural, remove the sentinel"
**Notes:** Went past the option as offered — not merely ignoring the sentinel but deleting
it from the cask. The orchestrator confirmed removal is safe (the sentinel lives inside the
Caskroom versioned dir, which Homebrew purges wholesale) and enumerated the four places it
touches. BREW-05's install gate is unaffected.

### Q3 — What structural evidence fires detection

| Option | Description | Selected |
|--------|-------------|----------|
| Name shape + Homebrew receipt | Segment shape AND Homebrew-authored `INSTALL_RECEIPT.json` at the matching ancestor | ✓ |
| Name shape only | Simplest; still a name guess, false-positive leg rests on segment-count pedantry | |
| Ask brew directly | `brew --prefix` / `brew list`; authoritative but breaks criterion 4 and tests Homebrew's CLI | |

**User's choice:** Name shape + Homebrew receipt
**Notes:** Receipt layout measured on a real Homebrew install before the question was asked
(cask: `Caskroom/<token>/.metadata/INSTALL_RECEIPT.json`; formula:
`Cellar/<name>/<version>/INSTALL_RECEIPT.json` plus `.brew/`).

### Q4 — Linuxbrew scope in criterion 3

| Option | Description | Selected |
|--------|-------------|----------|
| Keep, Cellar shape only, note why | Constructed-tree test under a linuxbrew prefix for the formula shape; record that casks are unreachable there | ✓ |
| Keep both shapes under linuxbrew | Criterion stays verbatim; half of it asserts a layout Homebrew on Linux cannot produce | |
| Drop linuxbrew from criterion 3 | Named scope reduction; loses the cheap guard against a `/opt/homebrew` hardcode | |

**User's choice:** Keep, Cellar shape only, note why
**Notes:** The orchestrator flagged in the question itself that "Homebrew on Linux has no
casks" is asserted, not measured, and must be confirmed at research time before being locked.

---

## Refusal semantics & `--force`

### Q1 — Exit code

| Option | Description | Selected |
|--------|-------------|----------|
| Non-zero error | Matches `checkWritable`'s precedent; a script notices | ✓ |
| Zero with an informational line | Friendliest interactively; exit 0 asserts an upgrade that did not happen | |
| Non-zero, distinct exit code | Most scriptable; introduces an exit-code vocabulary this CLI lacks | |

**User's choice:** Non-zero error

### Q2 — Can `--force` override

| Option | Description | Selected |
|--------|-------------|----------|
| No — `--force` is powerless here | Refusal unconditional; `--force` keeps its narrow documented meaning | ✓ |
| Yes, with a loud warning | Preserves agency; produces a Caskroom that disagrees with brew's receipt | |
| Separate explicit flag | `--ignore-homebrew`; same lying-Caskroom end state, plus a new public flag | |

**User's choice:** No — `--force` is powerless here
**Notes:** No escape hatch at all. `brew uninstall --cask codegraph` is the honest path.

### Q3 — Refusal message content

| Option | Description | Selected |
|--------|-------------|----------|
| Resolved path + `brew upgrade` | Names the symlink-resolved match, so a misfire is self-diagnosing | ✓ |
| Path + `brew update && brew upgrade` | Guards against `HOMEBREW_NO_AUTO_UPDATE=1`; redundant for the majority | |
| Command only, no path | Cleanest output; no diagnostic value when detection misfires | |

**User's choice:** Resolved path + `brew upgrade`

### Q4 — Where criterion 1's "unchanged" proof lives

| Option | Description | Selected |
|--------|-------------|----------|
| Automated sha256/mtime test + real-tap run | Standing regression guard plus one-time acceptance | |
| Real-tap one-time recording only | Follows the D-12/D-17 house pattern; no standing guard | |
| Automated sha256/mtime test only | Fails the roadmap's real-tap dependency clause | |

**User's choice:** "we're trying to test homebrew again :/"
**Notes:** All three options rejected. Fifth sighting of maintainer preference
`za9ms2mvjh`. The orchestrator had again answered only two of the ownership test's three
questions, skipping *would the next natural run notice it anyway?* — the same omission that
sank Phase 3's G1. Corrected proposal accepted below.

### Q4b — Corrected: seam-based proof

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — assert the seams are never reached | Assert `Options.download` and `Options.swap` are never invoked, via the existing func-vars; keep the real-tap run to observe the refusal | ✓ |
| Yes, and drop the real-tap run too | Would abandon the roadmap's own Depends-on clause | |
| Keep sha256/mtime as written | Indirect proxy; pulls Homebrew's filesystem behavior into a test of our branch | |

**User's choice:** Yes — assert the seams are never reached

---

## What `--check` reports under brew

### Q1 — Which version does it report

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub release, worded honestly | No new network path; wording admits the tap may lag | |
| Read the tap's cask version | Most accurate; a parser for a GoReleaser-generated file | |
| Report both | Most informative; two failure modes in a quick read-only command | |

**User's choice:** "don't check, this is what brew is for"
**Notes:** All three rejected. `--check` steps aside under brew rather than answering a
question `brew outdated` owns. The orchestrator confirmed this matches REQUIREMENTS.md
UPGR-03 better than the ROADMAP criterion does — UPGR-03 says "reports how to upgrade" and
never says "reports the available version", so this is a criterion amendment, not a
requirement change. It also collapsed the ordering question in the next area: the refusal
fires before `resolveLatest` for both forms.

### Q2 — `--check` exit code

| Option | Description | Selected |
|--------|-------------|----------|
| Zero — it's a query, nothing failed | Bare `upgrade` stays non-zero; two behaviors on one command need documenting | ✓ |
| Non-zero, same as bare `upgrade` | One rule; makes `if codegraph upgrade --check` unusable | |

**User's choice:** Zero — it's a query, nothing failed

---

## Where the refusal fires in `Run()`

The ordering half of this area was resolved by the `--check` decision above: detection
fires first, before `resolveLatest`, needing zero network. Only placement remained.

### Q1 — Package placement

| Option | Description | Selected |
|--------|-------------|----------|
| `internal/upgrade` (new `brew.go`) | Sits with the package that already owns install-shape knowledge and the seams | ✓ |
| New `internal/brew` package | Reusable by a future `doctor` command; a package for one predicate | |
| `internal/cli/upgrade.go` | Argued against — leaves `Run()` willing to swap over a Caskroom when called directly | |

**User's choice:** `internal/upgrade` (new `brew.go`)

---

## Todo cross-reference

| Option | Description | Selected |
|--------|-------------|----------|
| `03-EVIDENCE` sentinel-stranding claim | Subject evaporates once D-02 deletes the sentinel | ✓ |
| Stale man-page assertion (UF-5) | Co-located in the hook block D-02 already opens | ✓ |
| `verify:self-upgrade` signature gap | Our target executing our binary without checking our own signature | ✓ |
| None — keep the phase narrow | Record all ten as deferred | |

**User's choice:** All three folded — "again, let's make sure we're only
testing/fixing/addressing things we own"
**Notes:** The orchestrator applied the ownership filter to each individually and recorded
the boundary per todo in CONTEXT.md (D-13): the man-page fix must not become an assertion
about Homebrew's rollback behavior, and the `verify:self-upgrade` fix must not become a
re-test of Sigstore. Seven todos reviewed and not folded, itemized in CONTEXT.md's deferred
section.

---

## Claude's Discretion

- Test-fixture construction (`t.TempDir()` layout, receipt contents, how the four prefixes
  are simulated) — constrained by criterion 3's requirement that the false-positive case be
  an executing test, not a comment.
- The exported shape of the detection API in `brew.go`.
- Exact wording of the `--check` step-aside line.
- Whether the receipt check parses any field or only asserts existence.
- How `docs/` and `--help` record the two exit behaviors.

## Deferred Ideas

- A `codegraph doctor` / install-provenance command — surfaced while weighing whether
  detection deserves its own package; nothing scoped today.
- Detecting other managed install channels (future formula, distro package, `go install`) —
  the property generalizes, but nothing ships those channels. Cellar/formula detection is
  the only piece kept speculatively, and only because it is free.
