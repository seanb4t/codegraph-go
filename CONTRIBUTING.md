# Contributing

Thanks for considering a contribution. This document is the contract — the
triage process and the CI gates both read it literally.

## Issue first

**Open an issue before opening a pull request.** Every PR must reference an
issue it resolves.

This is not bureaucracy for its own sake: behavior is pinned by the project's
specs in `openspec/specs/` and its golden-corpus suite, and a change that looks
like an obvious improvement can still contradict a deliberate decision. The
issue is where we find that out before you've written the code.

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
   changes do not need a label. An issue labelled **`spec-needed`** also needs
   its OpenSpec proposal merged before the code (see
   [Specs and change proposals](#specs-and-change-proposals)).
2. All nine required checks pass: `test`,
   `actionlint (workflow static analysis)`, `govulncheck (DIST-03, blocking)`,
   `perf regression gate (PERF-02, INDX-06)`, `pr-title`,
   `reproducibility (double-build hash-diff, DIST-04)`,
   `goreleaser check (config validation, DIST-01)`,
   `tmux e2e (real-pty harness, TTY-01..TTY-07)`, `fragment-required`.
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
  day. `task test` covers every host-only leg — unit, golden-corpus
  behavioral, subprocess integration, isolated daemon, and race — and needs
  nothing beyond the C toolchain above.
- The cross-toolchain checks are deliberately separate targets, not part of
  `task test`: `task check:reproducibility:arm64` needs `zig`. It fails with
  an install instruction rather than skipping, so a green run means the same
  thing here and in CI.
- `task check:cross` is the same sweep the `pretag-gate` jobs in
  `release-pr.yml` and `release-tag.yml` run before a release PR is opened or
  a tag is pushed.
- `task release:dry-run` proves the full `goreleaser release` composition —
  all four release targets in one invocation — on a native darwin host,
  without publishing or signing anything. Needs `zig`, `syft`, and `cosign`
  on `PATH` in addition to the C toolchain above.
- `task release:dry-run-signed` is the same composition but does NOT skip
  `sign` — it exercises the `binary_signs:`/`sboms:` pipes against a
  throwaway local cosign key, so it needs no OIDC token, and asserts four
  distinctly-named published signature and SBOM artifacts from
  `dist/artifacts.json`.
- `task release:goreleaser` is the single definition of a REAL release —
  the same `goreleaser release --clean` composition, but it actually
  publishes to GitHub Releases, signs with cosign, and emits SBOMs. It is
  what release.yml's `release` job calls on a tag push; use `release:dry-run`
  above, not this one, to exercise the composition locally.
- `task check:linux-cross-export` and `task check:linux-cross-exec` are the
  REL-05 real-Linux-execution proof: the first resolves the two zig-crossed
  linux binaries `release:dry-run` just produced; the second runs one of
  them (via `CODEGRAPH_BIN`) on a **real Linux host** and asserts it indexes
  a real tree to a non-zero graph. Both are driven end-to-end by the
  permanent, dispatchable `linux-cross-canary` workflow — see
  `.github/workflows/linux-cross-canary.yml` — rather than run standalone
  locally.
- `task verify:release-assets` and `task verify:self-upgrade` re-prove
  REL-06/REL-07/REL-08 against a PUBLISHED release's RE-DOWNLOADED assets —
  never a local `dist/` copy. They need `TAG`/`REPO`/`GH_TOKEN` (plus
  `GOOS`/`GOARCH` for the self-upgrade target) in the environment and are
  driven automatically by the permanent, `workflow_run`-triggered
  `post-release-verify` workflow — see
  `.github/workflows/post-release-verify.yml` — every time `release.yml`
  completes, rather than run standalone locally.
- CI calls these same fine-grained targets directly — a contributor and CI
  run identical command bodies, never a divergent local approximation.
- `task`, `goreleaser`, `actionlint`, and `changie` build on demand from
  `go.tool.mod`, `go.tool-lint.mod`, and `go.tool-changie.mod` — there is
  nothing to install first, only Go and whatever toolchain the target
  itself needs. changie runs as `task changie`, for example
  `task changie -- latest`. Version bumps for those three files are
  manual: neither Dependabot nor Renovate is configured for this
  repository at all, so nothing updates them automatically.

## Pull requests

**Add a changelog fragment for every user-visible change.** Run
`task changie -- new -k <Kind> -b "<one sentence>"`, optionally with
`-m PR=<n>`. The kinds are declared in `.changie.yaml`: Breaking, Features,
Fixes, Performance and Dependencies. The command writes a file under
`.changes/unreleased/`; commit it with the change. The `fragment-required`
check fails a PR that changes `cmd/**` or `internal/**` (outside tests and
testdata) without one.

A change with nothing user-visible carries
`<!-- changelog-exempt: <reason> -->` in the PR body. The reason must not be
empty; `fragment-required` enforces that.

**The version is derived from the fragment kinds.** changie computes the next
version from the unreleased fragments: Breaking and Features bump the minor
version while the project is below 1.0, and Fixes, Performance and
Dependencies bump the patch. On every push to `main`, `release-pr.yml` batches
the fragments into one `chore(main): release vX.Y.Z` pull request. Merging that
PR cuts the release: `release-tag.yml` pushes the tag and `release.yml` builds
it. `task changie -- next auto` previews the next version locally. Never
hand-write or edit `.changes/v*.md` or `CHANGELOG.md`; the release PR generates
both.

**The PR title stays Conventional-Commits-shaped.** CI enforces it as hygiene,
so squash-merge commit subjects stay readable. It no longer decides the
version or the changelog.

```
^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\(scope\))?!?: subject
```

Scopes are lowercase, digits, `_` and `-` only. Choose deliberately: `feat:` on
an internal refactor advertises a feature that users cannot see.

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

CI/tooling, dependency, doc-only and spec-only (`openspec/`) PRs are detected
and exempted automatically. For anything else that genuinely does not fit, note
`<!-- pr-template-exempt: <reason> -->` in the body.

### Never do these

- **Do not bump versions or write release files by hand.** No hand-written or
  hand-edited `.changes/v*.md` or `CHANGELOG.md` (changie generates both in the
  release PR), and no hand-pushed `v*` tag. If a computed version looks wrong,
  the honest lever is the kinds of the fragments.
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

The suite is a mix of unit tests, golden-corpus behavioral fixtures frozen
from codegraph-go's own output against pinned third-party corpora, and
workflow-shape guards that parse the real `.yml` files.

## Specs and change proposals

The project specifies its behavior with [OpenSpec](https://github.com/Fission-AI/OpenSpec).

- `openspec/specs/` holds one spec per capability. Specs are added as changes
  touch each capability. A capability without a spec yet is still pinned by its
  tests and goldens.
- A change to specified behavior starts as an OpenSpec change in
  `openspec/changes/<id>/`. It holds a `proposal.md`, delta specs, an optional
  `design.md` and a `tasks.md`. The line after the title of `proposal.md` is
  `Issue: #<n>`, the issue the change resolves.
- Fixes and chores that change no specified behavior skip the proposal.
- `openspec/config.yaml` holds the project context and the rules every change
  follows.

With an agent, use `/opsx:propose`, `/opsx:apply` and `/opsx:archive` in Claude
Code, or the `openspec-*` skills in Codex. Without one, use the `openspec` CLI:
`openspec new change <id>`, `openspec validate <id> --strict` and
`openspec archive <id>`. Do not edit the generated `/opsx:*` commands or
`openspec-*` skills; `openspec update` rewrites them.

A pre-commit hook runs `openspec validate --all --strict` when a commit stages a
file under `openspec/`. CI does not run this check. The hook is optional. To
install it with [mise](https://mise.jdx.dev), run `mise trust`, `mise install`
and `hk install --mise`. mise installs the `hk` and `openspec` versions that
`mise.toml` pins. `HK=0 git commit` skips the hook for one commit.

A proposal can land on its own. A PR that touches only `openspec/` needs no
template, and `Refs #<n>` is enough. The implementation PR says `Closes #<n>`
and carries the code, the changelog fragment, and the `openspec archive` result,
so the spec changes in the same commit as the behavior.

The planning record from before the move to OpenSpec is archived at the
[`planning-archive-v0.15.0`](https://github.com/seanb4t/codegraph-go/tree/planning-archive-v0.15.0/.planning)
tag. Requirement and decision IDs in code comments, such as `SYNC-02` or
`D-06R`, resolve there. Its decision log is also kept in
[`docs/2026-10-17-gsd-era-decisions.md`](docs/2026-10-17-gsd-era-decisions.md).

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Licensing

Contributions are accepted under the [MIT License](LICENSE). By submitting a
pull request you agree your contribution may be distributed under those terms.
