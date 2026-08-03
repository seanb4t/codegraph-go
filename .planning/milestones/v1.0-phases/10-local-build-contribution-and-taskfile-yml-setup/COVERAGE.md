# API Coverage — Phase 10 (local build, CONTRIBUTING, Taskfile.yml)

No external API integration: this phase adds build tooling and rewires existing
CI `run:` bodies — it consumes no external API/SDK surface and adds no capability
the codebase calls out to.

## What this phase actually changed

`Taskfile.yml`, two isolated `go tool` modfiles (`go.tool.mod`,
`go.tool-lint.mod`), a composite GitHub Action (`.github/actions/install-task`),
and the `run:` bodies of existing CI workflows, which now invoke `task <target>`
instead of inline shell. No product source file gained a network call.

## Why the build-time dependencies are not an integrated API

The GitHub Actions marketplace actions and the Go module proxy this phase uses
are build-time infrastructure, not an integrated API whose capability surface
could be partially covered. Nothing in the shipped binary calls them; they exist
only to assemble it. Enumerating INTEGRATE/OPT-OUT rows for them would assert
coverage decisions that were never made — the precise failure the gate exists to
prevent.

The product's one real GitHub API integration (`internal/upgrade`, release
resolution and artifact download for `codegraph upgrade`) predates this phase and
is untouched by it, so its capability surface is not this phase's to enumerate.

## Status

Recorded per the `api-coverage` capability's documented path for a
no-integration phase: a reasoned declaration, not a fabricated matrix.

Reversible: if a later phase adds real API integration to product code, delete
this declaration and produce a real matrix.

*Declared during `/gsd-verify-work 10`.*
