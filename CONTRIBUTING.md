# Contributing

Thanks for considering a contribution. This document is the contract — the
triage process and the CI gates both read it literally.

## Issue first

**Open an issue before opening a pull request.** Every PR must reference an
issue it resolves.

This is not bureaucracy for its own sake: this project ports observable behavior
from another implementation, and a change that looks like an obvious improvement
is sometimes a deliberate parity decision recorded in `docs/FLAG-PARITY.md` or
`.planning/`. The issue is where we find that out before you've written the code.

The exception is trivially mechanical fixes — a typo, a broken link, a dead
reference. Open the PR directly and say so.

Issues use templates. Pick the one that fits:

| Template | For |
|---|---|
| `bug_report.yml` | Something behaves incorrectly |
| `feature_request.yml` | New capability that does not exist |
| `enhancement.yml` | Existing capability should work better |
| `chore.yml` | Maintenance, dependencies, tooling, docs |

Submissions missing required fields may be closed during triage with a note
saying which fields were missing. Reopen with a complete submission — that is
not a rejection of the idea.

**Security vulnerabilities do not go in issues.** See [SECURITY.md](SECURITY.md).

## Approval gates

A change lands only when all of these hold:

1. It references an approved issue. "Approved" is a label, not a vibe:
   **`approved-feature`** for features, **`approved-enhancement`** for
   enhancements. An open issue is not an approved one. Bug fixes and mechanical
   changes do not need a label.
2. All six required checks pass: `test`, `actionlint`,
   `govulncheck (DIST-03, blocking)`, `perf regression gate (PERF-02, INDX-06)`,
   `pr-title`, `reproducibility (double-build hash-diff, DIST-04)`.
3. Review conversations are resolved.
4. A maintainer merges it. `main` is protected; merges are **squash-only** and
   linear history is enforced.

Only items 2–4 are machine-enforced. Item 1 is a maintainer reading the PR —
worth knowing, so that a PR passing CI is not mistaken for a PR that is
approved.

For anything touching the release pipeline, signing identity, or the graph
schema, expect a slower conversation. Those have compatibility obligations to
binaries already in the wild.

## Building

You need a **C toolchain** — tree-sitter is CGo. The reasoning is in
[`PARSER-DECISION.md`](PARSER-DECISION.md), including why the pure-Go
alternatives were rejected.

```sh
CGO_ENABLED=1 go build ./cmd/codegraph
CGO_ENABLED=1 go test ./...
CGO_ENABLED=1 go vet ./...
```

Cross-compiling to other platforms needs `zig`. If you are only changing Go
code for your own platform, you do not need it.

codegraph targets **Linux and macOS** (amd64 and arm64). Native Windows is
not supported — Windows contributors should work inside WSL2, where the
Linux toolchain and the Linux build are what you exercise.

Workflow changes should pass `actionlint` locally before you push.

Every command CI runs is defined exactly once, as a `task` target — see
`Taskfile.yml` for the full list and each target's `desc:`.

- `task` with no arguments lists everything available.
- `task build`, `task test`, and `task lint` are the three you need day to
  day. `task test` covers every host-only leg — unit, golden parity,
  subprocess integration, isolated daemon, and race — and needs nothing
  beyond the C toolchain above.
- The cross-toolchain checks are deliberately separate targets, not part of
  `task test`: `task check:reproducibility:arm64` needs `zig`. It fails with
  an install instruction rather than skipping, so a green run means the same
  thing here and in CI.
- `task check:cross` is the same pre-tag sweep `release-please.yml`'s
  `pretag-gate` job runs before a tag can be created.
- CI calls these same fine-grained targets directly — a contributor and CI
  run identical command bodies, never a divergent local approximation.
- `task`, `goreleaser`, and `actionlint` build on demand from `go.tool.mod`
  and `go.tool-lint.mod` — there is nothing to install first, only Go and
  whatever toolchain the target itself needs. Version bumps for those two
  files are manual: neither Dependabot nor Renovate is configured for this
  repository at all, so nothing updates them automatically.

## Pull requests

**The PR title must be Conventional-Commits-shaped.** This is enforced by CI,
and it is load-bearing — `release-please` derives the version and the changelog
from merged PR titles, so a sloppy title becomes a permanent public release
note.

```
^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\(scope\))?!?: subject
```

Scopes are lowercase, digits, `_` and `-` only. `feat` bumps the minor version;
`fix` and `perf` bump the patch; `docs`, `chore`, `ci`, `test`, `build`,
`refactor` are hidden from the changelog. A `!` marks a breaking change.

Choose deliberately: `feat:` on an internal refactor advertises a feature that
users cannot see.

Every PR opens with a default template whose only job is to send you to a typed
one — GitHub does not offer them in its UI, so the default carries the links:

```
?template=PULL_REQUEST_TEMPLATE/fix.md
?template=PULL_REQUEST_TEMPLATE/enhancement.md
?template=PULL_REQUEST_TEMPLATE/feature.md
```

Switch **before** writing; selecting a template replaces the body.

They are worth the extra step. Each asks for the evidence review will want
anyway — `fix.md` in particular asks for the regression test's failing output,
which is the only thing distinguishing a test that catches the bug from one that
cannot fail.

CI/tooling, dependency, and doc-only PRs are exempt; note
`<!-- pr-template-exempt: <reason> -->` in the body.

### Never do these

- **Do not bump versions by hand.** No editing `.release-please-manifest.json`,
  no `Release-As:` commit footer, no `release-as` config key, no manufactured
  breaking-change marker. The version is *computed* from Conventional Commits.
  If a computed version looks wrong, the honest lever is the commits that
  produce it.
- **Do not edit `tools/bench/baseline.json`.** It is written only by
  `runner -mode regression -rebless`, run on the CI runner class that spends it.
  A baseline recorded anywhere else is not comparable — a wrong-platform
  baseline once produced a stable, entirely fictitious "10.6% regression" that
  survived three rounds of triage.
- **Do not change the release identity** — `internal/upgrade/verify.go`'s
  constants, or `release.yml`'s name, trigger, or cosign step. Binaries already
  shipped hard-code that identity; changing it breaks their upgrade path
  permanently. Guard tests will fail you, which is the point.

## Tests

New behavior needs a test. Bug fixes need a test that fails before the fix and
passes after — please say in the PR that you observed it fail, because a
regression test never seen red is indistinguishable from one that cannot fail.

The suite is a mix of unit tests, golden-corpus behavioral fixtures diffed
against frozen TypeScript CodeGraph v1.3.1 output, and workflow-shape guards
that parse the real `.yml` files.

## What `.planning/` is

Roughly half the tracked files live in `.planning/`. It is the project's
planning and decision record — phase plans, execution summaries, debug sessions,
and the reasoning behind decisions that are otherwise invisible in the diff.

It is published deliberately. If you want to know *why* something is the way it
is, the answer is usually there, and it is usually more candid than a commit
message. You are not expected to add to it, and PRs are not judged on it.

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Licensing

Contributions are accepted under the [MIT License](LICENSE). By submitting a
pull request you agree your contribution may be distributed under those terms.
