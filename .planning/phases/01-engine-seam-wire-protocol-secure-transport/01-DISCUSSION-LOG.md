# Phase 1: Engine Seam, Wire Protocol & Secure Transport - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-22
**Phase:** 1-Engine Seam, Wire Protocol & Secure Transport
**Areas discussed:** Todo folding, Engine seam shape, `codegraph ui` launch behavior, Truncation contract, Degraded-state surfacing

---

## Todo Folding

`gsd-tools query todo.match-phase 1` returned `todo_count: 6` with 5 matches. The
v0.12.0 roadmapper had already ruled **0 of 6** should link (`k4857qnq7p`), but that
ruling predated the FIX-01 root-cause reading surfaced during scouting.

| Option | Description | Selected |
|--------|-------------|----------|
| Fold the toolslist-repeat flake | 2026-08-07 wire-oracle ordering flake (area `mcp`, 0.9). Same file and same race shape as FIX-01 | ✓ |
| Fold golangci-lint too | 2026-08-10 golangci-lint + gofmt + idiomatic linters (area `ci`, 0.9). Unrelated to any Phase 1 requirement | ✓ |
| Fold post-release-verify guard | 2026-08-09 event-aware conclusion guard has no regression assertion (area `ci`, 0.9). Keyword match only | ✓ |
| Fold none — keep the roadmap ruling | Honor the roadmapper's 0-of-6 decision as-is | |

**User's choice:** All three folded.
**Notes:** Orchestrator flagged once that `golangci-lint` and `post-release-verify`
have **no backing requirement** — the phase's five success criteria mention neither —
so planning must either give them tasks under an existing requirement or the roadmap
needs a new one. User proceeded. Two further matches (`dry-run-signed` additions-only
diff guard, tap App secret-distinctness test) were reviewed and not folded; both are
recorded in CONTEXT.md `<deferred>`.

---

## Engine Seam Shape

📊 Research: web search returned mostly generic Go golden-file tutorials, low value.
The one applicable finding was the **view-model pattern** (Learn Go with Tests):
render becomes a pure function of a data struct, gather stays in the domain — which
`RenderNode(node, calls, calledBy)` already is. Snapshot-testing literature's standing
warning was more useful: goldens are sensitive to *output* changes, not call-graph
changes, so the risk in a wrapper refactor is not goldens failing but goldens
**passing while the two paths silently diverge**.

Code grounding: `renderSingleDefNode` (`node.go:420`) is four steps — `fetchCalls` →
`BuildReverseAdjacency` → `fetchCalledBy` → `RenderNode(...)`. The extraction is
literally hoisting the first three.

### Q1 — Relationship between the structured variants and the string methods

| Option | Description | Selected |
|--------|-------------|----------|
| `Node()` wraps `NodeDetail()` | One gather path; three consumers cannot disagree. Cost: goldens' call graph changes, so byte-identity becomes a claim to prove | ✓ |
| `NodeDetail()` sits beside `Node()` | Goldens untouched by construction. Cost: two gather paths that can drift silently | |
| Wrap, plus a standing equivalence test | Wrapper plus a permanent assertion that the two render identically | |

**User's choice:** `Node()` wraps `NodeDetail()` (with the shown code sketch).
**Notes:** Chose the wrapper *without* the standing equivalence test, which is what
made Q4 necessary.

### Q2 — Coverage of `Node()`'s three output shapes

| Option | Description | Selected |
|--------|-------------|----------|
| All three shapes | file-only, single-def, multi-def | ✓ |
| Single-def + file-only now | Defer multi-def | |
| Single-def only | Narrowest extraction | |

**User's choice:** All three shapes.
**Notes:** ENG-01 is a blocking gate for Phases 3–6; a partial seam weakens the gate,
and overloaded symbols are common in the Java/C# corpora.

### Q3 — Where the type lives

| Option | Description | Selected |
|--------|-------------|----------|
| Go struct in `internal/query` | Engine stays wire-agnostic; RPC layer owns mapping | ✓ |
| Return generated proto types | No mapping layer, but `internal/query` takes a UI wire dependency | |
| Reuse `internal/schema` types | No new node mapping, but couples wire shape to storage shape | |

**User's choice:** Go struct in `internal/query`.

### Q4 — How byte-identity gets proven

| Option | Description | Selected |
|--------|-------------|----------|
| Mutation-proof, count asserted | Break the gather, watch all 26 go RED, record the count, revert | ✓ |
| Pre/post byte-diff | Capture 26 outputs before and after, assert zero diff | |
| Both, as an in-phase gate | Byte-diff proves the refactor; mutation proves the instrument | |

**User's choice:** Mutation-proof with the count asserted.
**Notes:** Scoring is by counting `--- PASS` lines, never exit status —
`go test -run PATTERN` exits 0 on zero matches (`5pzpmvthcc`, three phantom commands
found this way in v0.11.0).

### Q5 — Whether `codegraph status` surfaces the ENG-04 commit SHA

| Option | Description | Selected |
|--------|-------------|----------|
| UI/RPC only — CLI untouched | Keeps "CLI bytes unchanged" literally true | ✓ |
| Surface in CLI status too | Most useful staleness signal for a human, but no golden covers `status` | |
| You decide | Defer to planning | |

**User's choice:** UI/RPC only.

---

## `codegraph ui` Launch Behavior

📊 Research (mxr, Spacelift, ggui, netlify dev — consistent across all four):
loopback default with a `--port` flag where `0` means OS-assigned; universal
`--no-open`/`--no-browser`, with the mature CLIs auto-suppressing on non-TTY, `CI=1`,
or `BROWSER=none`; foreground-with-Ctrl-C the norm, with only mxr detaching (and
consequently needing a `stop` verb).

Tension named up front: SRV-03 says bind address is a seam and "neither is exposed in
v1", and a port is arguably part of the bind address.

### Q1 — Port strategy and collision behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Ephemeral `:0`, no flag | Strictly honors SRV-03; collision handling never needs writing. Cost: no stable URL | ✓ (qualified) |
| Fixed default port, no flag | Stable bookmarkable URL. Cost: explicit collision handling, hardcoded bind decision | |
| Fixed default + `--port` | Most conventional. Cost: contradicts SRV-03's "neither is exposed in v1" | |

**User's choice (free text):** *"1 but leave room for a flag in the future to
override, or a config entry"*
**Notes:** Read as option 1 with the address **seamed** — the "1, but do not preclude
4" shape SRV-03 already applies to bind address. Orchestrator then established that
the repo has **no config-file mechanism** (no viper, no rc file; `.codegraph/` holds
only store + daemon lock) and that its override convention is env vars paired with
flags (`serve.go:300`: `--watch` is "the CLI twin of `CODEGRAPH_FORCE_WATCH=1`"). The
"config entry" half was therefore re-asked as a follow-up rather than assumed.

### Q1b (follow-up) — Shape of the address seam

| Option | Description | Selected |
|--------|-------------|----------|
| Struct field, unwired in v1 | `BindAddr` defaults to `127.0.0.1:0`; nothing sets it externally | ✓ |
| Struct field + env var now | Honor `CODEGRAPH_UI_ADDR` immediately. Cost: that IS exposing bind address in v1 | |
| Add a config file | A genuinely new mechanism. Orchestrator pushed back explicitly | |

**User's choice:** Struct field, unwired in v1.

### Q2 — Browser auto-open

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, with TTY/CI suppression | Open by default, `--no-open`, plus auto-suppress on non-TTY / `CI` | ✓ |
| Yes, with `--no-open` only | Simpler; scripted runs launch a browser unless the flag is remembered | |
| Print only, `--open` to launch | Safest, no platform code. Reads as a narrower interpretation of SRV-01 | |

**User's choice:** Yes, with TTY/CI suppression.
**Notes:** Requires a cross-platform open helper; none exists in the tree today.

### Q3 — Process lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| Foreground until Ctrl-C | Matches `serve` and `daemon start`; no PID file, no orphan class | ✓ |
| Detach, with a `ui stop` | Frees the shell. Cost: rebuilds the lifecycle surface SRV-01 forbids sharing | |
| Foreground, exit when browser closes | No lingering process. Cost: needs liveness that doesn't exist, and fights Phase 6 streams | |

**User's choice:** Foreground until Ctrl-C.

---

## Truncation Contract (RPC-05)

📊 Research (`connectrpc/connect-go` `option.go` / `envelope.go` source):
`connect-go` defaults to **unlimited** message size on both client and handler, so
RPC-05 is satisfied by no default. Critically, `WithSendMaxBytes` **errors rather than
truncating** — `envelope.go` returns `CodeResourceExhausted` — which is exactly the
failure mode RPC-05 forbids. Application truncation and the transport cap are
therefore two separate things.

Repo precedent found: `internal/mcp/session_line.go:18` truncates on a UTF-8 rune
boundary with one named constant "referenced from both the truncation call and the
test", with `session_line_test.go:74,247` asserting no split runes.

### Q1 — Bounding unit and cut point

| Option | Description | Selected |
|--------|-------------|----------|
| Bytes, on a rune boundary | Reuses the existing `session_line.go` pattern exactly | |
| Lines, with a line cap | Natural for a source viewer; cuts where a human reads. Cost: a minified single-line file still needs a byte cap underneath | ✓ |
| Byte range the client pages | Bounded AND complete. Cost: real paging state in schema and Phase 3 viewer | |

**User's choice:** Lines, with a line cap.
**Notes:** Orchestrator recorded the consequence: **three** limits must now stay
mutually consistent (line cap, byte cap underneath, transport backstop above), which
raises rather than lowers the importance of the one-named-constant discipline.

### Q2 — Truncation signal

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit field on the response | `truncated` + total; client still gets usable content | ✓ |
| Connect error code | `CodeResourceExhausted`; client gets no content — contradicts RPC-05's wording | |
| You decide | Defer field naming to planning | |

**User's choice:** Explicit field on the response.

### Q3 — Transport-level backstop

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — send + read caps | Defense in depth; a missed truncation becomes a clean error, not an OOM | ✓ |
| Read cap only | Bound inbound only | |
| Neither — application only | Cost: nothing structurally prevents a future unbounded RPC | |

**User's choice:** Yes — send + read caps.

---

## Degraded-State Surfacing (SRV-04)

📊 Research (Connect protocol reference + gRPC status-code guide): `unavailable` → 503,
"currently unavailable, usually transiently; clients should back off and retry" —
textually "indexing in progress". `failed_precondition` is explicitly wrong ("the
client must explicitly fix the system state before retrying"). Connect's reference
documents the canonical code-plus-typed-detail shape
(`{"code": "unavailable", "details": [{"type": "google.rpc.RetryInfo", ...}]}`). The
partial-availability pattern does not apply to most RPCs here — a failed `Open`
leaves no partial data.

Repo grounding: `ErrStoreLocked` (`pebble_store.go:110`) is an exported, `errors.Is`-able
sentinel classified exactly once inside `Open`'s bounded retry loop, already
special-cased at `serve.go:246` and `daemon.go:297`.

Tension named up front: SRV-04 says "never as an error", but on the wire `unavailable`
*is* an error.

### Q1 — How the degraded state reaches the client

| Option | Description | Selected |
|--------|-------------|----------|
| `unavailable` + typed detail | Idiomatic Connect shape; "never as an error" read as "never surfaced to the human as an error" | ✓ |
| Typed state on every response | Most literal reading. Cost: a success carrying no data; every message must remember to set it | |
| Field on `Status` only | Smallest schema. Cost: extra round trip and a race for every other view | |

**User's choice:** `unavailable` + typed detail.

### Q2 — Additional retry above `Open`'s budget

| Option | Description | Selected |
|--------|-------------|----------|
| No — honor `Open`'s budget | That budget IS the condition SRV-04 names | ✓ |
| Short additional retry | Hides fast re-indexes. Cost: a second timing budget; harder to demonstrate | |
| You decide | Defer to planning | |

**User's choice:** No — honor `Open`'s budget.

### Q3 — Whether `Status` answers when the store is locked

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — answers from off-store facts | The health view answering "the index is busy" IS a health answer | ✓ |
| No — same as every other RPC | One uniform rule. Cost: health view goes blank when most needed | |
| You decide | Defer the derivable-field split to planning | |

**User's choice:** Yes — answers from off-store facts.
**Notes:** Phase 4's index-health verdict depends on `Status` being reachable exactly
when things are wrong.

---

## Claude's Discretion

Offered as a final round and declined by the user ("I'm ready for context"); recorded
in CONTEXT.md as planning's to settle:

- FIX-01's fix mechanics — parse for an `id` to distinguish notifications from
  responses, vs moving the counter so only response writes decrement
- BLD-04's guard shape — regenerate-into-tempdir-and-diff vs regenerate-in-place-and-
  `git diff --exit-code`; whether `task proto` becomes a required CI check
- Exact proto field names, and whether the truncation total is bytes or lines
- `connectrpc.com/connect` version pinning

---

## Deferred Ideas

- Stable/bookmarkable UI URL — blocked by D-07's ephemeral port; unblocked by wiring
  D-08's existing field if Phase 3 needs it
- Surfacing the indexed commit SHA in `codegraph status` — additive later, but would
  need its own guard since no golden covers `status`
- Byte-range paging for source responses — considered and rejected for RPC-05
- A user config file mechanism — explicitly declined as out of scope for a port question
- Two reviewed-but-not-folded todos (`dry-run-signed` diff guard, tap secret-distinctness
  test) — see CONTEXT.md `<deferred>`
