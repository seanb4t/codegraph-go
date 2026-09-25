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
- **go-sdk streamable HTTP handler constructor** and whether go-github v92 ships its own
  App-auth transport (else ghinstallation v2.19.0). *Unresolved — unverifiable / non-authoritative
  in the research pass; pin at plan time.*
