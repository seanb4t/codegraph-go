---
created: 2026-09-08T00:00:00.000Z
title: graphstore archtest ignores per-package go/packages load errors, so a partial load passes vacuously
area: testing
severity: medium
files:

  - internal/graphstore/archtest/import_graph_test.go:33-46

threat_ref: CR-01 (07-REVIEW.md), sibling of the internal/query archtest fix in Phase 7
---

## Problem

`TestNoPackageBypassesGraphStore` calls `packages.Load` and checks only the
top-level `error` return plus `len(pkgs) == 0`. `go/packages` reports a
per-package failure (an unresolvable import, a build error reachable from the
loaded pattern) in `pkg.Errors`, not in that top-level error. A broken subtree
is then absent from, or has an incomplete `Imports` map in, the returned graph,
so the pebble-importer scan looks for members that were never added.

The positive control (`foundGraphstoreImporter`) narrows the hole but does not
close it: a load failure in a package *other than* `internal/graphstore` leaves
the control satisfied while the bypass check silently skips the broken package.

## Evidence

Phase 7 code review (deep) found the identical defect in the new
`internal/query/archtest/import_direction_test.go` (CR-01) and reproduced it
with an unresolvable blank import: the test reported `--- PASS`. That test was
fixed in commit `a90b5457` and demonstrated RED in `07-MUTATION-LOG.md`
family (b) addendum. The graphstore archtest is the precedent the query archtest
was modelled on and carries the same shape; it was outside Phase 7's file set.

## Suggested fix

Mirror `a90b5457`: after the zero-package check, add

```go
if n := packages.PrintErrors(pkgs); n > 0 {
	t.Fatalf("packages.Load reported %d package error(s) — the import graph did not fully resolve, so this test cannot verify anything about the affected package(s); see the errors above", n)
}
```

Prove RED per the standing rule: add an unresolvable blank import to any
package under the load pattern, watch the test fail naming the count, revert
byte-clean, re-run green, paste the transcript.
