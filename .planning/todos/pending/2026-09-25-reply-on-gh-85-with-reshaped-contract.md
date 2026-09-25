---
created: 2026-09-25T00:00:00.000Z
title: Reply on GH #85 / #82 / #80 with the reshaped contract from the 2026-09-25 exploration so fovea's Phase 15 plan updates
area: server-mode
severity: major
files:
  - .planning/notes/gh-85-service-mode-design-map.md
---

## Problem

fovea's Phase 15 plan (seanb4t/fovea, requirements CG-01..04) was written against #85 as
filed: codegraph as a Temporal worker (#82), fovea forwarding tarballs and `.fovea.yml`
excludes per request (#80). The exploration changed both. Until the issues say so, fovea's
planning will build against the wrong contract.

## What to post (draft — do not send without Sean's go-ahead)

**On #82 (Temporal):** codegraph will not embed the Temporal SDK or run a worker. It exposes an
idempotent start-or-join `EnsureGraph(repo, revision)` with single-flight dedupe, persisted
per-graph status (absent/building/ready/failed), and wait-with-deadline (server-streaming
progress). fovea's Temporal activity is the durable retrier; a retry after a codegraph restart
re-requests and rebuilds. The "published job contract" becomes part of the #79 API package.
Suggest retitling #82 to "idempotent start-or-join index requests with persisted status".

**On #80 (snapshots):** codegraph keeps a bare git mirror per repository and materializes
trees itself via its own GitHub App; fovea does not upload tarballs. Repo-level excludes come
from a repo-owned config file honoured by both server and CLI, not from a per-request field.
fovea's `.fovea.yml` excludes can be dropped or migrated.

**On #85 (tracking):** the milestone is the Team Scale central server (source-driven builds
from App installations, OIDC + static-token authz, MCP over streamable HTTP as a pure OAuth
resource server), with fovea as first client. Link the design map note once it is on `main`.

## Also

- Ask fovea to remove its "fovea passes excludes per request" and "codegraph as Temporal
  worker" assumptions from its design map / CG-01..04.
- Include the required AI authorship byline on each comment.
