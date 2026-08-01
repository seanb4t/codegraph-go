<!--
FIX PR — something behaved incorrectly and now does not.

Title must be Conventional-Commits-shaped and CI enforces it:
  fix(scope): subject      -> bumps the PATCH version
  fix(scope)!: subject     -> breaking change

The title becomes a permanent public changelog entry under "Bug Fixes".
-->

Fixes #

## The bug

<!-- What went wrong, in terms of observable behavior. -->

## Root cause

<!--
Why it happened — not just where the fix went. If the cause turned out to be
different from what the issue assumed, say so; that is worth more than the
patch.
-->

## The fix

<!-- What you changed, and why this approach over the alternatives. -->

## Required checklist

- [ ] References the issue above
- [ ] PR title is Conventional Commits, type `fix`
- [ ] **A regression test exists, and I watched it fail before the fix**
- [ ] `CGO_ENABLED=1 go test ./...` passes locally
- [ ] `CGO_ENABLED=1 go vet ./...` is clean

### On that regression test

Paste the failing output from **before** your fix:

```
```

<!--
This is asked for literally, not as a formality. A test that has never been
observed red is indistinguishable from one that cannot fail — and this project
has shipped at least one guard that looked correct and could never have caught
anything. Seeing it fail is the only evidence the test is wired to the bug.
-->

## Scope

- [ ] This fixes the reported bug and nothing else — unrelated cleanup is a
      separate PR
- [ ] The fix does not paper over a symptom while leaving the cause in place
- [ ] I did **not** touch `internal/upgrade/verify.go`, `release.yml`'s name /
      trigger / cosign step, or `tools/bench/baseline.json`

## Blast radius

- [ ] I checked whether the same mistake exists elsewhere in the codebase

<!--
If the bug came from a pattern rather than a typo, the same construction is
often repeated. Say what you searched for and what you found — including
"searched for X, no other instances".
-->
