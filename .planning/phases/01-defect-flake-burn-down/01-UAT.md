---
status: complete
phase: 01-defect-flake-burn-down
source: [01-VERIFICATION.md]
started: 2026-09-15T20:37:58Z
updated: 2026-09-15T20:37:58Z
---

## Current Test

number: 1
name: 16px favicon legibility (FIX-02)
expected: |
  The 'CG' ligature in web/static/favicon.svg reads clearly in a real browser tab bar at 16px, matching the middle tile of .planning/phases/01-defect-flake-burn-down/assets/01-mark-round2-sheet.png. The executor's 32px inspection was clean but flagged the 16px render as "noticeably tighter/blurrier ... closer to a rounded/pretzel shape than a crisp CG".
awaiting: none — all tests complete

## Tests

### 1. 16px favicon legibility (FIX-02)

expected: The 'CG' ligature reads clearly at 16px in a real browser tab bar, matching the round-2 comparison sheet's middle tile (D-01 locks the mark's geometry; any remedy is a viewBox/stroke-weight judgement call on a locked asset).
result: pass — validated by maintainer 2026-09-15 (autonomous checkpoint: "All good — continue")

### 2. Live visual verification of the FIX-05 render-flow fix on both corpora

expected: In a real Chromium session on (a) this repository's own index and (b) the pinned guava corpus, /graph shows (1) a first paint of the settled ELK arrangement with no visible flash of an unlaid-out graph, (2) a legible arrangement (nodes not flung into unreadable sparseness, edges traceable, initial fit shows the whole graph), (3) correct expand/collapse of at least one directory node, and (4) correct behaviour when navigating away from /graph mid-layout on guava and back. If degraded, the remedy is a smaller/differently-targeted layout change — never an allowlist entry (D-07).
result: pass — validated by maintainer 2026-09-15 (autonomous checkpoint: "All good — continue")

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
