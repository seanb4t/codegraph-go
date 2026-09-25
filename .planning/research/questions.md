# Open research questions

Appended by `/gsd-explore`; consumed by phase research. Each entry carries its origin and
the disposition it left the exploration with.

## 2026-09-25 — GH #85 service mode (see notes/gh-85-service-mode-design-map.md)

- **Keycloak 26.x and Client ID Metadata Documents.** MCP 2026-07-28 deprecates DCR in favour of
  CIMD (admitted: modelcontextprotocol.io spec). Claude Code's native OAuth uses CIMD. Does the
  realm at `id.fzymgc.house/realms/fzymgc` accept a URL `client_id`? If not, humans on Claude
  Code use static tokens (allowed under D7). *Unresolved — not checked.*
- **Policy engine choice.** cedar-go v1.8.0 (Apache-2.0, AWS-maintained) admitted as a fit for
  principal→repo entitlements. OPA-as-library / casbin / OpenFGA comparison *unresolved —
  untagged in the research pass*. Decide on license, maintenance, and whether policies must be
  operator-editable at runtime.
- **Base-vs-head diff-impact semantics (#81).** Both graphs share deterministic node ids
  (`nodeid` hashes file path + symbol). Define the diff as set-difference restricted to changed
  files plus reverse closure; specify what "gained/lost edges" means for heuristic edge kinds
  (implements, routes). *Design question, no source needed.*
- **Head-graph TTL vs base retention.** Interaction of "newest N bases per repo, never evict
  recently queried" with idle-TTL heads and a per-repo storage budget. *Design question.*
- **Manifest store.** bbolt vs a Pebble "meta" DB for graph status/retention/last-queried;
  whether per-graph `Export` is the transport for CI-distributed indexes (SEED-004).
  *Design question.*
- **Symbol history / blame as a mirror-backed RPC, not index data.** codegraph records one
  `Meta.commit_sha` per graph and nothing temporal on nodes/edges/files (no author, no blame,
  no per-symbol commit); `internal/gitmeta` only does worktree/remote introspection. With a
  bare mirror per repo (D2), "who last changed this symbol" is
  `git log -L <start>,<end>:<path> <sha>` against the mirror — on demand, correct by
  construction, nothing new to keep in sync in Pebble. Candidate RPC
  `SymbolHistory(repo, sha, symbol) → commits[]` (author, date, subject, sha). Decide: is this
  in the fovea cutover scope (reviewer routing / context), and does it need a cache keyed by
  (sha, path, range) since `git log -L` walks history? *Design question, no source needed.*
- **go-sdk streamable HTTP handler constructor** and whether go-github v92 ships its own
  App-auth transport (else ghinstallation v2.19.0). *Unresolved — unverifiable / non-authoritative
  in the research pass; pin at plan time.*

## 2026-09-25 — changie adoption (see notes/changie-release-management.md)

- **Where does the phase-close `changie new` step live?** Decision D3 has GSD's verify/close
  write one fragment per user-visible change. Does gsd-core expose a capability/hook at phase
  verify or close (the way `execute:wave:post` and `ship:post` exist) where a project-local
  step can run `CI=true changie new -k … -b … -m PR=…`? If not, this is an upstream feature
  request to open-gsd/gsd-core — per the planning-artifacts rule, never invent a local step
  inside a tool-owned workflow file. *Unresolved — check gsd-core capabilities registry.*
- **Fragment-required gate implementation.** changie has no check subcommand (research pass,
  abstain). Extend `scripts/pr_template_policy.py` (already path-aware, fails closed) or add a
  sibling script; decide whether the exemption is a body marker (`changelog-exempt`) only or
  also a label. *Design question.*
