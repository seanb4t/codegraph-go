---
id: SEED-002
status: dormant
planted: 2026-08-01
planted_during: v1.0 Phase 09 (release-please + GoReleaser)
trigger_when: when relevant
scope: unknown
---

# SEED-002: we need a homebrew installation path

## Why This Matters

_To be filled in. Run `/gsd-capture --seed --enrich SEED-002` to add context._

## When to Surface

**Trigger:** when relevant

This seed will surface during `/gsd-new-milestone` when the milestone scope matches.

## Scope Estimate

**Unknown** — run `/gsd-capture --seed --enrich SEED-002` to estimate effort.

## Breadcrumbs

- `.goreleaser.yaml` has **no `brews:` block**. It has `builds:` (39),
  `archives:` (138), `checksum:` (143) — but `release.yml` only ever runs
  `goreleaser build --single-target`, never `goreleaser release`, so the
  archives/checksum blocks are dead config today. GoReleaser's native Homebrew
  support runs during `release`, so a tap is not just a config block here.
- Binaries publish **raw, not tarballed** (D-02 — `internal/upgrade` hashes the
  binary itself). Homebrew formulae usually expect tarballs.
- **`codegraph upgrade` self-updates in place.** That conflicts with a
  brew-managed install, where users expect `brew upgrade` and the Cellar is not
  meant to be mutated by the binary itself.
- Prior art in-repo: `tools/bench/runner/main.go` already hardcodes
  `macOSHomebrewTSBinary = "/opt/il/bin/codegraph"` for the TS reference binary.
- The repo went **public 2026-08-01**, which is the prerequisite for a public tap.

## Notes

_Captured via one-shot seed capture. Enrich with trigger, why, and scope at your convenience._
