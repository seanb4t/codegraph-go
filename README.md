# codegraph

**A pre-indexed code knowledge graph for coding agents.** One static binary — no
Node runtime, no `node_modules`, no install-time network fetch.

Coding agents burn most of their context rediscovering the same thing: where a
symbol is defined, what calls it, and what breaks if it changes. `codegraph`
answers those from a local graph it maintains incrementally, so an agent can ask
one question instead of grepping its way to an answer.

```console
$ codegraph explore "release verification"
```

Returns the relevant symbols' verbatim source plus the call paths between them —
ranked by graph relevance, not string similarity.

---

## Supported platforms

| Platform | Architectures | Status |
|----------|---------------|--------|
| Linux | `amd64`, `arm64` | Supported |
| macOS | `amd64`, `arm64` | Supported |
| Windows | — | **Not supported natively — use WSL2** |

On Windows, run codegraph inside [WSL2](https://learn.microsoft.com/windows/wsl/install)
and install the `linux` binary for your architecture. Everything works there,
including `codegraph upgrade`, because WSL2 is Linux.

One caveat worth knowing: if your project lives on a `/mnt/<drive>` Windows
mount rather than inside the WSL2 filesystem, codegraph turns the file watcher
off automatically — recursive watching across that mount is too slow to be
reliable. Keep repositories on the WSL2 side (`~/...`, not `/mnt/c/...`) for
full functionality, or pass `--watch` to force it on anyway.

## Install

Download a binary from [Releases](https://github.com/seanb4t/codegraph-go/releases/latest):

```sh
TAG=v0.2.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

curl -fLo codegraph \
  "https://github.com/seanb4t/codegraph-go/releases/download/${TAG}/codegraph_${TAG}_${OS}_${ARCH}"
chmod +x codegraph
```

Every binary is signed. **Verify before you run it** — the full walkthrough is in
[`docs/RELEASE.md`](docs/RELEASE.md):

```sh
cosign verify-blob \
  --bundle "codegraph_${TAG}_${OS}_${ARCH}.sigstore.json" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp \
    '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  "codegraph_${TAG}_${OS}_${ARCH}"
```

Subsequent upgrades are self-service: `codegraph upgrade` resolves the latest
release, verifies the signature against the identity compiled into the binary,
and swaps itself.

## Quick start

```sh
cd your-project
codegraph init          # build the graph
codegraph explore "authentication middleware"
codegraph callers ParseToken
codegraph impact internal/store/pebble.go
```

`init` indexes the repo and writes to `.codegraph/`. From then on the graph is
kept fresh automatically — by an in-process watcher, or by git hooks
(`codegraph githooks install`) where a watcher isn't viable.

## Use it from an agent

`codegraph serve --mcp` speaks MCP over stdio, with auto-sync on by default:

```sh
codegraph install       # register with detected agents
codegraph serve --mcp   # or run it directly
```

Eight tools are exposed: `codegraph_explore`, `codegraph_node`,
`codegraph_search`, `codegraph_callers`, `codegraph_callees`, `codegraph_impact`,
`codegraph_files`, `codegraph_status`.

`codegraph_explore` answers most questions in one call.

## Commands

| | |
|---|---|
| `init` · `index` · `sync` · `uninit` | build and maintain the graph |
| `explore` · `search` · `node` | find code |
| `callers` · `callees` · `impact` · `affected` · `files` | traverse it |
| `serve --mcp` | MCP server for agents |
| `install` · `uninstall` · `githooks` | agent and git integration |
| `daemon` · `status` · `unlock` | lifecycle |
| `migrate` | import an existing TypeScript CodeGraph index |
| `upgrade` · `version` | self-update |

Every command has `--help`. Output is colorized on a TTY and byte-identical
plain when piped — agents and pipelines never see escape codes.

## Language support

Fourteen languages are registered. They are **not** equally supported, and the
differences are documented rather than averaged away — see
[`docs/LANGUAGE-CAPABILITY-MATRIX.md`](docs/LANGUAGE-CAPABILITY-MATRIX.md).

| Tier | Languages | Extraction | Cross-file resolution |
|---|---|---|---|
| Full | Go, Java, C#, TypeScript, TSX | full | full |
| Full extraction, no interface dispatch | Python, JavaScript | full | full |
| Partial | Rust, Ruby, PHP, C, C++, Swift, Kotlin | full | partial |

## Performance

Measured against the TypeScript implementation it ports — median-of-3, on three
real repositories, on the same machine:

| Repo | Indexing | Query latency | Peak RSS | Cold start |
|---|---|---|---|---|
| `weft` (~84 files) | 8.1× faster | 12.9× lower | 2.8× lighter | 8.4× faster |
| `codegraph` (the TS original) | 4.3× faster | 11.5× lower | 4.5× lighter | 7.6× faster |
| `cockroachdb/pebble` (largest) | 21.2× faster | 7.9× lower | 3.0× lighter | 8.2× faster |

All three repos, every metric. Indexing spread is wide — **4.3×–21.2×** — and
depends heavily on the language mix, so treat any single number with suspicion,
including the flattering one. Full methodology, pinned corpus commits, and raw
per-run figures are in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

## Supply chain

The reason this is one static binary rather than a bundled runtime:

- **Signed** — cosign keyless, per binary, with the signing identity anchored to
  this repository's release workflow and a tag ref. The identity is compiled
  into the binary, so `codegraph upgrade` refuses anything signed elsewhere.
- **Provenance** — GitHub-native build provenance via
  `actions/attest-build-provenance`, published through GitHub's Attestations
  API, verifiable with `gh attestation verify`.
- **SBOM** — SPDX per platform, published as a release asset.
- **Reproducible** — CI rebuilds every release binary and fails on a hash diff.
- **Scanned** — `govulncheck` gates every merge.

## Status

**v0.2.0 — pre-1.0.** The core is in daily use and the release pipeline is
signed and attested end to end, but the API and CLI surface may still move
before 1.0. Behavioral parity with TypeScript CodeGraph v1.3.x is covered by a
fixture harness that diffs against frozen goldens; known divergences are logged,
not hidden.

## Relationship to the original

This is an independent Go reimplementation of
[CodeGraph](https://github.com/colbymchenry/codegraph) (TypeScript, MIT). It
reproduces that project's CLI semantics, MCP tools, and ranking behavior closely
enough to be a drop-in replacement, and `codegraph migrate` imports an existing
TypeScript index.

The original is the reason this project has a specification to hit at all. See
[NOTICE](NOTICE) for attribution.

## Contributing

Building from source requires a C toolchain — tree-sitter is CGo, and the
reasoning behind that choice is in [`PARSER-DECISION.md`](PARSER-DECISION.md).

```sh
CGO_ENABLED=1 go build ./cmd/codegraph
CGO_ENABLED=1 go test ./...
```

Issues and pull requests are welcome. Please read
[SECURITY.md](SECURITY.md) before reporting anything security-related — don't
open a public issue for a vulnerability.

## License

MIT — see [LICENSE](LICENSE).
