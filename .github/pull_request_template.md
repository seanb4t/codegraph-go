<!--
This is the DEFAULT template, applied to every PR.

There are more specific ones. Append to this page's URL to switch:

  ?template=feature.md       new capability            -> feat:
  ?template=enhancement.md   existing thing, better    -> feat: / perf:
  ?template=fix.md           something was wrong       -> fix:

Use them where they fit — they ask for the evidence review will want anyway.
-->

Resolves #

## What changed

<!-- And why. The diff shows what; this should explain why it was worth doing. -->

## Checklist

- [ ] References an issue above — see [CONTRIBUTING.md](../blob/main/CONTRIBUTING.md);
      trivially mechanical fixes (typo, dead link) may skip this, say so if that's the case
- [ ] PR title is Conventional Commits and the type is honest about user-visible
      impact — it becomes a permanent public changelog entry
- [ ] `CGO_ENABLED=1 go test ./...` passes locally
- [ ] `CGO_ENABLED=1 go vet ./...` is clean
- [ ] Behavior changes have tests; bug fixes have a test observed failing first
- [ ] Docs updated if a documented surface changed

## Not touched

- [ ] `internal/upgrade/verify.go`, or `release.yml`'s name / trigger / cosign step
      — shipped binaries hard-code that identity
- [ ] `tools/bench/baseline.json` — written only by `-rebless` on the CI runner class
- [ ] `.release-please-manifest.json`, or any version-forcing footer or config key
      — the version is computed, never chosen

<!--
If you did need to touch one of those, that's not automatically wrong — but say
why here, and expect a slower review. Each one has a compatibility obligation to
binaries or releases already in the wild.
-->
