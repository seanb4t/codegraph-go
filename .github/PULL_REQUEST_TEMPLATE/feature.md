<!--
FEATURE PR — a capability that did not exist before.

Title must be Conventional-Commits-shaped and CI enforces it:
  feat(scope): subject          -> bumps the MINOR version
  feat(scope)!: subject         -> breaking change

The title becomes a permanent public changelog entry. Write it for someone
reading release notes, not for the diff.
-->

Resolves #

## What this adds

<!-- The capability, in terms of what a user can now do. -->

## Why it belongs here

<!--
Link the approved issue discussion. If a maintainer agreed it should be built,
say where. PRs without an approved issue may be asked to back up a step.
-->

## How it behaves

<!-- Show it. Command + output, or the MCP tool call + result. -->

```console
$
```

## Required checklist

- [ ] References an **approved** issue above
- [ ] PR title is Conventional Commits and `feat` is the honest type
      (not a refactor dressed as a feature)
- [ ] New behavior has tests
- [ ] `CGO_ENABLED=1 go test ./...` passes locally
- [ ] `CGO_ENABLED=1 go vet ./...` is clean
- [ ] Docs updated if this changes a documented surface
      (`docs/FLAG-PARITY.md`, `docs/LANGUAGE-CAPABILITY-MATRIX.md`, README)

## Parity

- [ ] I checked whether TypeScript CodeGraph v1.3.1 has this behavior
- [ ] If it does, this matches it — or the divergence is documented and explained below

<!-- Divergence, if any: -->

## Compatibility

- [ ] No change to the graph schema — or migration is handled and tested
- [ ] No change to MCP output shapes that agents already consume — or called out below
- [ ] I did **not** touch `internal/upgrade/verify.go`, `release.yml`'s name /
      trigger / cosign step, or `tools/bench/baseline.json`

<!--
If you did need to touch any of those, stop and say why in the issue first.
Shipped binaries hard-code the release identity; changing it breaks their
upgrade path permanently.
-->
