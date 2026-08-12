---
id: SEED-002
status: implemented
planted: 2026-08-01
planted_during: v1.0 Phase 09 (release-please + GoReleaser)
consumed_by: v0.5.0 — macOS Distribution & Homebrew (Phases 1–4)
consumed_on: 2026-08-11
trigger_when: when relevant
scope: unknown
---

# SEED-002: we need a homebrew installation path

## Status: implemented — consumed by v0.5.0

This seed was consumed by milestone **v0.5.0 — macOS Distribution & Homebrew**.
`ROADMAP.md:16` and `PROJECT.md:17` both record the milestone as consuming SEED-002;
this file is the seed-side half of that record, set so the milestone-close audit stops
listing it as pending work.

Every breadcrumb below was resolved:

| Breadcrumb | Resolution |
|---|---|
| No `brews:` block; `release.yml` only runs `goreleaser build --single-target`, so archives/checksum are dead config | Phase 1 migrated to a single `goreleaser release` invocation; `archives:`, `checksum:`, `signs:` and `sboms:` are all live |
| GoReleaser's Homebrew support runs during `release`, so a tap is not just a config block | Phase 3 shipped the `homebrew_casks:` block and the `seanb4t/homebrew-tap` repository |
| Binaries publish raw, not tarballed; formulae expect tarballs | Phase 1 (REL-09) added a `zip` archive alongside the byte-unchanged raw binary; the cask consumes `ids: [zip]` |
| `codegraph upgrade` self-updates in place, conflicting with a brew-managed install | Phase 4 (UPGR-01…03) made `upgrade` detect a brew-managed install and step aside with a pointer to `brew upgrade codegraph` |
| Repo went public 2026-08-01 — prerequisite for a public tap | Satisfied before the milestone began |

**Caveat recorded at close:** Phase 4's implementation was still unmerged to `main` when this
seed was marked implemented (see the v0.5.0 milestone close). The seed is closed against the
milestone's completed work, not against a released binary.

**Tooling gap, noted rather than worked around:** GSD has no writer verb that marks a seed
implemented — `bin/lib/audit.cjs:305` only *reads* `status`, treating anything outside
`{dormant, active, triggered}` as implemented. This frontmatter was therefore edited directly.
If a `/gsd-capture --seed --close` verb is added later, prefer it.

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
