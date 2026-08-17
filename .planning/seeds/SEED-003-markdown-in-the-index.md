---
id: SEED-003
status: dormant
planted: 2026-08-10
planted_during: v0.5.0 / phase 03-homebrew-tap-cask
trigger_when: when relevant
scope: unknown
audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
  status: dormant
---

# SEED-003: Consider how to add markdown to the index

## Why This Matters

_To be filled in. Run `/gsd-capture --seed --enrich SEED-003` to add context._

## When to Surface

**Trigger:** when relevant

This seed will surface during `/gsd-new-milestone` when the milestone scope matches.

## Scope Estimate

**Unknown** — run `/gsd-capture --seed --enrich SEED-003` to estimate effort.

## Breadcrumbs

Collected at plant time (2026-08-10), unverified beyond the greps noted:

- **Language registration pattern:** one file per language at
  `internal/indexer/languages_<lang>.go`. Twelve exist today: `c`, `cpp`, `csharp`,
  `go`, `java`, `kotlin`, `php`, `python`, `ruby`, `rust`, `swift`, `typescript`
  (plus a `languages_test.go`). A markdown entry would presumably follow the same shape.

- **`.md` is currently EXCLUDED BY ASSERTION, not merely unhandled.** This is the
  non-obvious part. `internal/indexer/discover_test.go:162` documents that "an
  unsupported extension (.md, .json) is never discovered", and line 191 checks
  `relPath == "README.md" || relPath == "config.json"`. So indexing markdown is **not a
  purely additive change** — it turns an existing green test red, and that test is
  asserting current behaviour deliberately. Whoever picks this up has to decide whether
  that assertion narrows to `.json` or is replaced outright, and say why.

- **No markdown grammar is vendored.** `go.mod` pins eleven `tree-sitter-*` grammar
  modules; there is no `tree-sitter-markdown`. Adding one means a new dependency in the
  CGo parser path, which lands in the SBOM and `govulncheck` reachable set — the same
  cost `go-md2man`/`blackfriday` incurred in phase 03-01. Note also that the upstream
  markdown grammar is split (block + inline parsers), which does not match the
  one-grammar-per-language assumption the current registry appears to make.

- **Open design question the above implies:** what a markdown "symbol" even *is*.
  The existing extractors emit functions/types/calls; headings, links and code fences
  are a different shape of node, and it is not obvious they belong in the same graph
  vocabulary. Worth settling before any grammar work.

Related surfaces: `internal/indexer/capability/matrix.go`, `internal/parser/cgo/parser_cgo.go`.

## Notes

_Captured via one-shot seed capture. Enrich with trigger, why, and scope at your convenience._
