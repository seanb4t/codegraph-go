---
status: complete
phase: 03-verb-fold
source: [03-VERIFICATION.md]
started: 2026-09-17T00:20:00Z
updated: 2026-09-17T00:20:00Z
---

## Current Test

number: 1
name: Re-frozen generated surface reviewed by the maintainer (VERB-06/VERB-07, D-11/D-13)
expected: |
  The maintainer accepts the reviewed `docs/CLI-REFERENCE.md` diff from the feat commit (+28/−49, exactly three change groups: the `query`/`unlock` sections and index bullets removed; `search` loses "(locations only)" and gains `--full`, `-j`, `-k`, `-l`; a new `codegraph daemon unlock` section and SEE ALSO bullet) and the two reasoned allowlist lines (`codegraph query` / `codegraph unlock` — hidden rename stub, removed in v0.15.0), per D-13's deferral of the end-of-phase review to the maintainer.
awaiting: none — all tests complete

## Tests

### 1. Re-frozen generated surface reviewed by the maintainer (VERB-06/VERB-07, D-11/D-13)

expected: The maintainer accepts the reviewed `docs/CLI-REFERENCE.md` diff (+28/−49, three change groups only) and the two reasoned allowlist lines as the re-frozen surface; the executor's review (03-02-SUMMARY per-file diff table) and the verifier's independent regeneration (`task docs:cli:drift` byte-identical, two regenerations `cmp`-identical) stand.
result: pass — validated by maintainer 2026-09-16 (autonomous checkpoint: "All good — complete Phase 3")

### 2. Rename-stub stderr shape after the WR-01 review fix (VERB-03/VERB-04, D-05/D-06)

expected: The maintainer accepts that each stub now returns one error whose text is the whole two-line D-06 message (no direct `Fprintln`, no `codegraph:` prefix), so the compiled binary prints exactly the two lines once via `cmd/codegraph/main.go`'s single exit path, exits 1, and executes nothing — pinned end-to-end by `test/integration/renamed_stubs_test.go`. This supersedes 03-02's literal "two `Fprintln` + one error" wording (which printed a duplicated third line in the real binary) while honouring D-05 (no `os.Exit` in the stub, no cobra `Deprecated`, no new exit code or convention) and D-06 (the exact two lines).
result: pass — validated by maintainer 2026-09-16 (autonomous checkpoint: "All good — complete Phase 3")

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- IN-01 (Info, out of fix scope by the maintainer's choice at this checkpoint): `search` has no `Long` text describing `--full`; a `docs:` follow-up would regenerate `docs/CLI-REFERENCE.md` through `task docs:cli`.
