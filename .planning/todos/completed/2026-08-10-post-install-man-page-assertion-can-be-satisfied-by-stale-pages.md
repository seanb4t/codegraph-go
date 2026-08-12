---
created: 2026-08-10T00:00:00.000Z
title: post-install man-page assertion can be satisfied by stale pages from a prior failed install
area: release
severity: medium
status: resolved
resolved_at: 2026-08-11T00:00:00.000Z
resolved_by_phase: 04-02
files:
  - .goreleaser.yaml:551-556
  - .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md:815-833
threat_ref: UF-5 (03-SECURITY.md), T-03-04, AR-04
---

## Resolution (2026-08-11, Phase 04-02)

Closed via the mtime-fallback fix shape (Assert on mtime), extended with a
size fallback for same-tick rewrites. `hooks.post.install` now takes a
`man_pages_before` baseline (path -> {mtime, size}) immediately before
`system_command binary, args: ["man", man_dir]` runs, and computes
`fresh_man_pages` as the subset of the post-run glob that is either newly
present, mtime-advanced, or size-differing versus that baseline. Raises when
`fresh_man_pages` is empty, distinguishing "this install wrote pages" from
"pages are present" per rule `84d1gfpywd`. `Taskfile.yml`'s
`release:rehearse-cask` independently re-confirms the same property via two
`find -newer` checks against real marker files (one baselining the first
install, one baselining the reinstall). Passes on `brew reinstall` — the
"page tree grows with every new subcommand" property this todo asked to
preserve is untouched, since freshness (not growth) is the assertion.

## Problem

The cask's first post-install assertion is:

    system_command binary, args: ["man", man_dir]
    man_pages = Dir.glob("#{man_dir}/codegraph*.1")
    if man_pages.length <= 1
      raise "..."
    end

where `man_dir` is `#{HOMEBREW_PREFIX}/share/man/man1` — the **shared** man
directory, not the Caskroom, and there is no pre-install baseline.

`03-EVIDENCE.md:815-833` documents that 30 orphaned man pages survive a failed
install's rollback: Homebrew purges the Caskroom but never runs the cask's
uninstall hook, so pages the binary already wrote stay on disk.

Consequence: a later install whose binary runs `man` **successfully but writes
nothing** would glob those 30 residual pages and pass assertion one.

## Why the exposure is narrow

`system_command`'s default `must_succeed: true` still catches a binary that
cannot run at all, which is the dominant failure this gate exists for. The gap is
specifically "ran, exited 0, wrote nothing" — plus a prior failed install having
left residue.

## Why it is still worth fixing

D-12 accepts that the RED proof does not re-fire, so these two in-hook assertions
are the **entire permanent standing guard** for the cask (AR-04). Assertion one
currently does not distinguish *"this install wrote pages"* from *"pages are
present"* — which is the property rule `84d1gfpywd` asks a guard to assert.

## Fix shape

Any of:

- Take a pre-`system_command` baseline glob and assert the count **grew**.
- Assert on mtime — require at least one page newer than the hook's start.
- Have the binary write into a fresh temp dir, assert there, then install.

Prefer whichever keeps the "page tree grows with every new subcommand" property
the current comment deliberately protects (no pinned count).
