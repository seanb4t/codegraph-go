---
quick_id: 260913-pkp
slug: fix-graphstore-archtest-to-fail-closed-on-per-package-load-errors
status: complete
completed: 2026-09-13
files_modified:
  - internal/graphstore/archtest/import_graph_test.go
  - .planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md
  - .planning/STATE.md
files_deleted:
  - .planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md
commits:
  - 140e42ae
  - cf411a98
---

# Fix graphstore archtest to fail closed on per-package `go/packages` load errors — Summary

Closed the guard-vacuity hole in `TestNoPackageBypassesGraphStore`: a per-package `go/packages` load error (unresolvable import, build error) anywhere under the module pattern used to leave the positive control satisfied while the pebble-bypass scan silently skipped the broken package. Mirrored the Phase 7 CR-01 fix (`a90b5457`) and demonstrated the guard RED against a confirmed-applied mutation planted outside `internal/graphstore`, per the standing rule (STATE.md rule `84d1gfpywd`).

## What shipped

**Task 1 (RED/GREEN):** `internal/graphstore/archtest/import_graph_test.go` gained a `packages.PrintErrors(pkgs)` guard immediately after the `len(pkgs) == 0` check and before the `foundGraphstoreImporter` scan. A non-zero count now `t.Fatalf`s with the byte-identical message used by the `internal/query/archtest` sibling, naming the count and pointing at the errors PrintErrors already surfaced on stderr. Comment block cites `CR-01` and `a90b5457` explicitly.

**Task 2:** Resolved the 2026-09-08 todo through `gsd_run todo complete` (never hand-moved) — the file now lives at `.planning/todos/completed/` with `completed: 2026-09-13` and `status: completed` stamped by the tool. `.planning/STATE.md`'s Pending Todos row was removed and one row appended to the Resolved table recording closure by this quick task.

## RED/GREEN transcript

**Pre-mutation gate** (`git diff --quiet -- internal/query/traverse.go`): exit 0 (clean).

**Mutation applied:** inserted `_ "github.com/seanb4t/codegraph-go/internal/zz-does-not-exist"` as the first line of `internal/query/traverse.go`'s import block (1 insertion, confirmed via `git diff --stat`).

**Pre-fix PASS (the hole), `GOTOOLCHAIN=go1.26.6 go test -count=1 -run TestNoPackageBypassesGraphStore ./internal/graphstore/archtest/...`, test file unmodified:**

```
ok  	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	0.344s
```

The positive control (`foundGraphstoreImporter`) is satisfied by `internal/graphstore`'s own pebble import while the broken `internal/query` subtree is silently absent from the loaded graph — exactly the hole the todo named, observed rather than assumed.

**Fix applied** to `import_graph_test.go` (guard + comment block, `CR-01` cited, Fatalf string byte-identical to `a90b5457`'s). `GOTOOLCHAIN=go1.26.6 gofmt -l internal/graphstore/archtest/` printed nothing.

**Post-fix FAIL naming the count (same plant still applied), same command with `-v`:**

```
../../query/traverse.go:4:2: no required module provides package github.com/seanb4t/codegraph-go/internal/zz-does-not-exist; to add it:
	go get github.com/seanb4t/codegraph-go/internal/zz-does-not-exist
    import_graph_test.go:58: packages.Load reported 1 package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see the errors above
--- FAIL: TestNoPackageBypassesGraphStore (0.27s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	0.383s
FAIL
```

`n=1` — this whole-module `Tests: true` load happened to match the precedent's own count for the identical single-package plant; the plan flagged any positive value as the expected outcome, not specifically 1.

**Revert:** `git checkout -- internal/query/traverse.go`.

**Byte-clean proof:** `git diff --quiet -- internal/query/traverse.go` → exit 0. `git status --porcelain -- internal/` printed exactly one line: ` M internal/graphstore/archtest/import_graph_test.go`.

**Post-revert GREEN, same command:**

```
ok  	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	0.296s
```

**Sibling regression check**, `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/archtest/... ./internal/graphstore/archtest/...`:

```
ok  	github.com/seanb4t/codegraph-go/internal/query/archtest	0.204s
ok  	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	7.122s
```

## Acceptance criteria (verbatim results)

- `rg -o 'packages\.PrintErrors\(pkgs\)' import_graph_test.go | wc -l` = 1
- `rg -o 'packages\.Load reported %d package error\(s\)' import_graph_test.go | wc -l` = 1
- `rg -o 'CR-01' import_graph_test.go | wc -l` = 1
- Guard placement order (`len(pkgs) == 0` line 43 → `packages.PrintErrors(pkgs)` line 57 → `foundGraphstoreImporter := false` line 61) — strictly increasing
- `gofmt -l internal/graphstore/archtest/` — empty
- `git diff --quiet -- internal/query/traverse.go` after revert — exit 0
- Task 2: pending file absent, completed file carries both stamped fields, exactly one `internal/graphstore/archtest` mention in STATE.md and it is the Resolved row, `git status --porcelain` on intended paths empty after commit

All PASS.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — the change only tightens an existing test's failure mode (T-260913-pkp-01, disposition `mitigate`, per the plan's threat register); no new surface, no package installs.

## Self-Check: PASSED

- FOUND: `internal/graphstore/archtest/import_graph_test.go` (modified)
- FOUND: `.planning/todos/completed/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md`
- MISSING (expected): `.planning/todos/pending/2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md` (moved, not deleted-in-place)
- FOUND commit `140e42ae` (`git log --oneline --all | grep 140e42ae`)
- FOUND commit `cf411a98` (`git log --oneline --all | grep cf411a98`)
