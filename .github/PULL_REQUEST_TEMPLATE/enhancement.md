<!--
ENHANCEMENT PR — something that already worked, now works better.

Title must be Conventional-Commits-shaped and CI enforces it. Pick the type
that tells the truth about what changed:
  feat(scope):     new user-visible capability      -> MINOR
  perf(scope):     measurably faster / lighter      -> PATCH
  refactor(scope): no behavior change               -> hidden from changelog

If nothing a user can observe changed, it is not a feat.
-->

Resolves #

## What was wrong with it

<!-- The existing behavior, and why it fell short. -->

## What it does now

<!-- Before / after. Real output beats description. -->

**Before**
```console
$
```

**After**
```console
$
```

## Required checklist

- [ ] References an **approved** issue above
- [ ] PR title type is honest about the user-visible impact
- [ ] The improvement is covered by a test that would fail without this change
- [ ] `CGO_ENABLED=1 go test ./...` passes locally
- [ ] `CGO_ENABLED=1 go vet ./...` is clean

## Behavior change

- [ ] Existing output shapes are unchanged — **or** the change is described below
      and I understand agents consume these

<!-- Describe any changed output, flag default, or ranking behavior: -->

## If this is a performance change

- [ ] I measured it, and the numbers are below
- [ ] I measured on a **fixed** platform, varying only the code — a comparison
      across machines or operating systems measures the machines, not the change

<!--
Numbers, with the environment stated:

  before:
  after:
  environment:
  method:  (median of N, not a single sample)
-->

- [ ] I did **not** modify `tools/bench/baseline.json`

## Parity

- [ ] This does not diverge from TypeScript CodeGraph v1.3.1 — or the divergence
      is deliberate, documented, and explained above
