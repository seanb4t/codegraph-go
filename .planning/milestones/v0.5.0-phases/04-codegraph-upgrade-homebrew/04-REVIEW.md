---
phase: 04-codegraph-upgrade-homebrew
reviewed: 2026-08-11T18:21:02Z
depth: deep
files_reviewed: 11
files_reviewed_list:
  - .goreleaser.yaml
  - README.md
  - Taskfile.yml
  - docs/RELEASE.md
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
  - internal/upgrade/brew.go
  - internal/upgrade/brew_test.go
  - internal/upgrade/taskfile_shape_test.go
  - internal/upgrade/upgrade.go
  - internal/upgrade/upgrade_test.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-08-11T18:21:02Z
**Depth:** deep
**Files Reviewed:** 11
**Status:** issues_found

## Summary

This phase teaches `codegraph upgrade` to detect a Homebrew-managed install
structurally (resolved-symlink path shape + Homebrew's own
`INSTALL_RECEIPT.json`) and refuse to self-replace it. I read every file in
scope, traced `detectBrewManaged`'s ancestor-walk arithmetic by hand against
both the Cellar and Caskroom layouts, and cross-checked it against a **real,
live Homebrew installation on this machine** (`/opt/homebrew`, both a cask —
`1password-cli` — and several Cellar formulae) rather than trusting the
fixture-only test suite. The receipt locations the code computes
(`<tokenDir>/.metadata/INSTALL_RECEIPT.json` for casks,
`<versionDir>/INSTALL_RECEIPT.json` for Cellar) match the real, on-disk
layout exactly. I also traced every path through `Run()` by hand to confirm
the brew-refusal branch precedes all network I/O and that `--force` cannot
reach it, built and `go vet`'d the changed packages, and ran the full
relevant test suite (`go test ./internal/upgrade/... ./internal/cli/...`) —
everything passes, including a floor assertion
(`TestCosignIdentityPolicyBoundaryParityWithCompiledPattern`) that is
currently satisfied by exactly 7 literals with a floor of 7 — i.e. genuinely
tight, not pre-satisfied slack.

I did not find a bypass of the brew refusal, a false-positive/false-negative
in the detection predicate against any of the four documented prefixes, or a
vacuous test in the reviewed files. `internal/upgrade/brew_test.go`'s two
row-count guards (16 total rows, 7 not-detected rows) are exact floors
against the current fixture count, not loose ones. Documentation
(`README.md`, `docs/RELEASE.md`, the cobra `Long` text) accurately and
consistently describes the refusal, the `--check` step-aside, the absence of
a `--force` override, and the exit codes — I traced each claim back to the
`Run()` code path it describes and found no drift.

One real (non-blocking) gap: `Taskfile.yml`'s `verify:self-upgrade` target
applies the "`gh release download` can exit 0 having matched nothing"
hard-fail-by-name check to the cosign signature bundle but not to the
release binary itself, despite the same risk applying to both downloads
identically. See WR-01 below — it is not exploitable (a missing binary still
fails loudly, just less legibly, because `cosign verify-blob` errors on a
nonexistent target) but is worth fixing for consistency with the pattern the
surrounding code otherwise applies uniformly.

## Warnings

### WR-01: Asymmetric "download matched nothing" guard in verify:self-upgrade

**File:** `Taskfile.yml:2731-2747`
**Issue:** The target explicitly guards against `gh release download`
silently matching zero files (a real, cited regression — "this repository's
own v0.5.1 post-release break") for the cosign signature bundle:

```sh
gh release download "${PRIOR_TAG}" --repo "${REPO}" --dir "${SIG_DIR}" \
  --pattern "${PRIOR_SIG_NAME}"
PRIOR_SIG="${SIG_DIR}/${PRIOR_SIG_NAME}"
if [ ! -f "${PRIOR_SIG}" ]; then
  echo "::error::cosign bundle ${PRIOR_SIG_NAME} was not downloaded — cannot verify the prior release binary before it is executed"
  exit 1
fi
```

but the identical risk on the immediately preceding binary download is left
unguarded:

```sh
gh release download "${PRIOR_TAG}" --repo "${REPO}" --dir "${DL_DIR}" \
  --pattern "${PRIOR_ASSET}"
PRIOR_BIN="${DL_DIR}/${PRIOR_ASSET}"
```

If `gh release download` matches zero files for `PRIOR_ASSET` (e.g. an
asset-naming drift or a bad `PRIOR_TAG` resolution), `PRIOR_BIN` never
exists. This is not silently masked — the subsequent `cosign verify-blob
... "${PRIOR_BIN}"` call fails to open the missing file and errors under
`set -euo pipefail`, so the script still aborts. It is fail-closed, not a
security bypass. But the failure surfaces as a generic cosign error instead
of the named, actionable `::error::` this target uses everywhere else for
exactly this class of problem, which is a real (if minor) loss of
diagnostic clarity for whoever debugs a red CI run.
**Fix:** Add the same explicit existence check immediately after the binary
download, mirroring the sig-bundle check:
```sh
gh release download "${PRIOR_TAG}" --repo "${REPO}" --dir "${DL_DIR}" \
  --pattern "${PRIOR_ASSET}"
PRIOR_BIN="${DL_DIR}/${PRIOR_ASSET}"
if [ ! -f "${PRIOR_BIN}" ]; then
  echo "::error::${PRIOR_ASSET} was not downloaded — gh release download matched nothing for ${PRIOR_TAG}"
  exit 1
fi
```

## Info

### IN-01: TOCTOU window between detection and swap is structural, not a defect

**File:** `internal/upgrade/upgrade.go:98-105`, `internal/upgrade/swap.go`
**Issue:** `detectBrewManaged(targetPath)` inspects the resolved path once,
before `resolveLatest`/`download`/`verify` run; `atomicSwap` later acts on
the original (un-resolved) `targetPath` again via `os.Stat`/`os.Rename`, not
on the `ResolvedBinary` the detector computed. In principle a symlink at
`targetPath` could be redirected between the detection check and the final
rename. This is not something to fix in this phase: exploiting it requires
an attacker who already has write access to the running binary's directory,
at which point they could simply overwrite the binary directly — the
brew-detection check was never a security boundary against a co-resident
attacker, only a footgun-prevention measure for the legitimate operator.
Noting this only because review priority #1 asked for explicit TOCTOU
scrutiny; no fix is proposed and none is warranted here.

### IN-02: Detection predicate verified correct against a real, live Homebrew install

**File:** `internal/upgrade/brew.go:83-119`
**Issue:** Not a defect — recorded so the review is falsifiable. I cross-checked
the receipt-path arithmetic (`chain[i-1]`/`chain[i-2]` relative to the
`Caskroom`/`Cellar` ancestor) against a real `/opt/homebrew` install on this
machine:
- Cask (`1password-cli`): binary resolves directly into
  `Caskroom/<token>/<version>/op` (no intermediate `bin/`), receipt lives at
  `Caskroom/<token>/.metadata/INSTALL_RECEIPT.json` — exactly where
  `detectBrewManaged` looks (`tokenDir/.metadata/INSTALL_RECEIPT.json`).
- Formula (`terraform-lsp`, `tokei`, `himalaya`, `rumdl`, ...): symlinked
  through `bin/`, so the binary's immediate parent is `Cellar/<f>/<v>/bin`,
  and the receipt lives at `Cellar/<f>/<v>/INSTALL_RECEIPT.json` — exactly
  where `detectBrewManaged` looks (`versionDir/INSTALL_RECEIPT.json`).

Both match the code's assumptions exactly. This is the kind of assumption a
fixture-only test suite cannot validate on its own (the fixtures encode the
same assumption the production code makes, so agreement between them proves
internal consistency, not real-world correctness) — worth stating plainly
since it is the single highest-priority correctness question this phase
raises.

---

_Reviewed: 2026-08-11T18:21:02Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
