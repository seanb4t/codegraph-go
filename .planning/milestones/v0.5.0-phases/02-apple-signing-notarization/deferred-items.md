# Deferred Items

Out-of-scope discoveries and deliberate deferrals logged during plan execution, per the
executor's scope-boundary rule ("only auto-fix issues directly caused by the current task's
changes; log out-of-scope discoveries here rather than fixing them"), and per this phase's own
explicit deferrals. Owned by phase 02 — a prior revision of plan 02-02 wrote a Phase-2 decision
into `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/deferred-items.md`,
which weakened artifact ownership; that file now carries only a one-line backlink to this one.

## 02-02: Re-wiring `check:darwin-release-build` into `darwin-toolchain-canary.yml`

**Found during:** 02-02 Task 3 (enumerate every `goreleaser` caller and state what stops the
notarize pipe reaching it)

**Context:** Phase 1 (commit `008d51c`) unwired `check:darwin-release-build` from
`darwin-toolchain-canary.yml` because the build-scoped `binary_signs:` pipe forced every
`goreleaser build` invocation through a cosign keyless OIDC call the canary could not supply — a
`pull_request`-triggerable workflow has no OIDC token, and granting `id-token: write` to it would
have been a security regression rather than a fix.

D-18 (this plan) moved cosign to the release-scoped `signs:` pipe, which is not reachable from
`goreleaser build` at all (`sign.Pipe{}` is not registered in GoReleaser's `BuildCmdPipeline`).
That specific blocker — the cosign OIDC call — is now gone. Re-wiring `check:darwin-release-build`
back into `darwin-toolchain-canary.yml` is therefore technically unblocked, and D-18's own research
(`02-RESEARCH.md`) names it as a side benefit worth at least evaluating.

**Why deferred, not done:** `darwin-toolchain-canary.yml` is `pull_request`-triggerable. Even
though the cosign blocker is gone, `check:darwin-release-build` now reaches the `notarize:` pipe
(registered inside `BuildPipeline`, and unlike `signs:`, `--skip=notarize` is not a valid flag for
the build command — see Taskfile.yml's comment on that target). The only thing that would stop an
unprivileged fork PR run from attempting a real Apple API call is
`notarize.macos[0].enabled`'s five-term credential conjunction evaluating false — and that guard
has not yet been observed evaluating false in a real CI run, only in this plan's static backstop
test (`TestNotarizeMacosEnabledIsEnvGated`) and by source-reading `notary.MacOS`'s `Run` method.
Re-wiring a `pull_request`-triggerable workflow to depend on a not-yet-CI-observed guard, one
config typo away from a real Apple API call from a fork PR, is not a decision to make casually or
silently.

**Recommended follow-up / revisit condition:** Revisit after the first real notarized release
(plan 02-07) has demonstrated the `enabled:` guard's behavior in CI in both directions — evaluating
true with real credentials in `release.yml`'s privileged job, and (ideally) evaluating false in an
unprivileged run. At that point, re-wiring `check:darwin-release-build` into
`darwin-toolchain-canary.yml` is a reasonable follow-up, still gated on confirming that doing so
grants that workflow neither `secrets.MACOS_*` nor `id-token: write` (D-17).
