---
created: 2026-08-10T00:00:00.000Z
title: 03-EVIDENCE.md falsely claims a failed install can strand the Phase-4 sentinel
area: docs
severity: low
files:
  - .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md:829-831
  - .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md:1039
  - .goreleaser.yaml:551-587
threat_ref: UF-3 (03-SECURITY.md)
---

## Problem

`03-EVIDENCE.md` states in two places that a failed `brew install` "can leave the
sentinel behind with no cask installed". **That is false.**

The post-install hook writes the sentinel at `.goreleaser.yaml:577-587`, which is
**after both raises**:

- assertion one (man pages) raises at `:553-555`
- assertion two (version equality) raises at `:566-568`
- sentinel write at `:577-587`

A gate failure raises before the sentinel exists, so the sentinel cannot be
stranded by a failed install. Only man pages leak — they are written by the
binary into the shared `share/man/man1` before assertion one evaluates.

The evidence file contradicts its own claim two hundred lines earlier: after plan
03-04's Mutation 2 it records 30 orphaned man pages (`:815`) while
`find … -iname "*.codegraph-brew-install*"` returned nothing (`:807-808`).

## Why it matters

Phase 4 (`codegraph upgrade` × Homebrew) reads this sentinel to detect a
brew-managed install. If Phase 4 designs around a possibly-stranded sentinel it
will carry defensive complexity for a state that cannot occur — and, worse, may
weaken its detection to tolerate a "false" sentinel that is in fact always
truthful.

**Phase 4 must not design around a stranded sentinel.**

A successful uninstall is fully symmetric (`hooks.post.uninstall` removes it).

## Fix shape

Correct both passages to state that the sentinel is written after both raises and
therefore cannot be stranded, and that only man pages leak on a failed install.
Cite the line numbers above so the claim is checkable.
