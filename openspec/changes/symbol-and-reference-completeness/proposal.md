# Proposal
Issue: #239

## Why

CodeGraph omits several variable and field declarations across its core languages. Missing bindings also break reference paths and hide indirect calls, including the Go function-variable case in #175.

## What Changes

- Index package or module variables and constants, plus statically named fields, in Go, TypeScript, JavaScript, TSX, Python, and Java.
- Preserve declaration ownership, initializer relationships, and statically resolvable reads and writes.
- Resolve calls through uniquely established callable bindings without treating initialization as a call.
- Distinguish Go production and test binding contexts so test stubs do not hide the production path in #175.
- Define declaration kinds and bounded inherited-field lookup consistently across the supported languages.
- Preserve current identities where possible and rebuild existing indexes when new extraction becomes available.
- Define exact supported forms and exclusions through tests and the evaluation in #238.

Out of scope: unrestricted local-variable indexing, whole-program data flow, runtime reflection, general dynamic dispatch, Java overload resolution, and additional languages. Local scope analysis is limited to preventing incorrect bindings.

This change incorporates #175. [PR #237](https://github.com/seanb4t/codegraph-go/pull/237) separately proposes graph identity and storage consistency work. These additions do not expand its scope.

## Capabilities

### New Capabilities

- `data-symbols`: Discoverable variables and fields, their ownership and references, and conservative calls through callable bindings.

### Modified Capabilities

None.

## Impact

The change affects `internal/indexer` extractors and resolution, graph kind handling, rebuild metadata, query integration tests, and capability documentation.

New nodes and edges change query content. Existing CLI and MCP envelopes remain compatible. The implementation audits kind filters and generated clients through their owning tools.

The implementation of `index-invariants`, proposed in PR #237, supplies identity finalization and structural invariants. Issue #238 supplies independent acceptance evidence. No production compiler or language runtime becomes required.
