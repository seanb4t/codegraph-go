# Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-04
**Phase:** 1-protocol-scoping-the-sdk-independent-wire-oracle
**Areas discussed:** Oracle form & expected values, Oracle coverage bar, 8-agent audit method, Negotiated-version stderr line

Four gray areas were offered; the user selected all four.

---

## Oracle form & expected values

### Q1 — How should the wire oracle be built?

| Option | Description | Selected |
|--------|-------------|----------|
| Capture tool + frozen transcripts | Go package with a thin runnable entrypoint that spawns ANY binary path, drives a scripted request list over real stdio, emits a normalized transcript. Tests invoke it and diff against frozen transcripts. Same tool runs pre-migration, post-migration, and against a released asset. Cost: new internal surface + normalization rules to get right. | ✓ |
| Go integration tests only | Extend `test/integration/` with a helper and table-driven cases, following `mcp_stdout_purity_test.go`. Zero new surface. Cost: comparing pre/post-migration bytes means checking out the old commit — no artifact to hold side by side. | |
| Tests now, extract tool if Phase 2 needs it | Defer the surface decision until there is evidence. Cost: the extraction lands mid-migration, when you least want to refactor your oracle. | |

**User's choice:** Capture tool + frozen transcripts (the recommended option).

### Q2 — Where do the oracle's expected values come from?

| Option | Description | Selected |
|--------|-------------|----------|
| Captured + frozen, plus hand-authored spec anchors | Bulk transcripts frozen as captured; on top, a small hand-authored set of literal spec assertions independent of the capture (`-32601`, the `protocolVersion` string, framing invariants). | ✓ |
| Captured + frozen only | Pure snapshot of today's behavior. Cost: encodes today's bugs as correct — anything mark3labs gets wrong that the official SDK gets right would read as a regression. | |
| Hand-authored expected shapes only | Every golden is a claim, not a photograph. Cost: prohibitive across the full tool surface, and markdown tool output has no spec to derive from. | |

**User's choice:** Captured + frozen, plus hand-authored spec anchors (the recommended option).

### Q3 — How is "don't regenerate the golden alongside the change it should catch" enforced?

Framing supplied before the question: transcripts *must* legitimately change in Phase 3 (`resultType`, `_meta.serverInfo`, `ttlMs` are deliberate wire additions), so the rule cannot be "never regenerate."

| Option | Description | Selected |
|--------|-------------|----------|
| CI cross-change guard + no auto-regenerate flag | No `UPDATE_TRANSCRIPTS=1` path exists; plus a CI check failing when one PR touches both a frozen transcript and `go.mod`'s MCP dep or `internal/mcp/*.go`. Both survive squash-merge. | ✓ |
| CI cross-change guard only | Keep a convenient regenerate path for Phase 3's legitimate churn. Cost: the flag is one keystroke away when someone is trying to get a red build green. | |
| No regenerate path only | Rely on transcript diffs being conspicuous in review. Cost: convention is not a mechanism — the exact objection the blocking-anti-pattern protocol asks about. | |

**User's choice:** CI cross-change guard + no auto-regenerate flag (the recommended option).

### Q4 — How much does the capture tool normalize before freezing a transcript?

Framing supplied before the question: any normalization that round-trips through `encoding/json` destroys the field presence/ordering evidence that PITFALLS Pitfall 7's shipped go-sdk v1.7.0 `omitempty` regression is only visible in.

| Option | Description | Selected |
|--------|-------------|----------|
| Named-field placeholder substitution only | Verbatim bytes except an explicit documented allowlist (`<REPO>`, `<VERSION>`, `<TS>`); everything else byte-for-byte. Cost: a new unstable field shows up as a flake before it shows up as a rule. | ✓ |
| Canonicalize via JSON re-serialization | Decode, drop volatile fields, re-encode with sorted keys. Immune to cosmetic churn. Cost: erases the omitempty/ordering signal. | |
| No normalization — make the input deterministic instead | Pin version, fixed absolute path, raw bytes. Purest oracle. Cost: fights the existing `copyFixture` + `t.TempDir()` harness and CI's checkout location. | |

**User's choice:** Named-field placeholder substitution only (the recommended option).

**Not asked in this area** (offered as a "more questions" branch, user chose to move on): where the tool lives in the tree, one transcript file per scenario vs. combined, whether the tool ships in the release binary.

---

## Oracle coverage bar

Two scout findings were surfaced before the questions: the existing golden parity corpus drives the server through `mcpclient.NewInProcessClient` (PITFALLS Trap A) and therefore covers zero wire bytes; and every handler in `internal/mcp/tools.go` returns `mcp.NewToolResultError(err.Error()), nil` rather than `return nil, err`, making Pitfall 6's error-mapping risk lower than the research assumed.

### Q1 — How much of the wire surface does the oracle lock down?

| Option | Description | Selected |
|--------|-------------|----------|
| All 8 tools + envelope + error/edge shapes | ~18 scenarios: handshake, three `tools/list` variants, a `tools/call` per tool, unknown-method / unknown-tool / malformed-args / confinement-reject. Matches Phase 2's literal criterion with no interpretation gap. | ✓ |
| Envelope + one tool per output-shape class | ~9 scenarios; argues tools 4–8 re-prove the identical `NewToolResultText` envelope. Cost: reads Phase 2's "every tool" as "every tool shape." | |
| Envelope and protocol only | No tool content on the wire. Cost: the golden corpus never touches stdio, so nothing would assert how tool text is JSON-escaped and wrapped. | |

**User's choice:** All 8 tools + envelope + error/edge shapes (the recommended option).

### Q2 — Does the Phase 1 oracle script older-protocol-version handshakes against today's server?

Framing supplied: VRFY-04's anti-circularity logic applies with most force to Legacy negotiation, which REQUIREMENTS.md names the single highest-consequence mistake available in this milestone.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — freeze a Legacy baseline now | Handshake at each revision mark3labs supports plus an unsupported one, frozen. Cost: pulls a slice of SPEC-06's assertion work into Phase 1. | ✓ |
| No — one version, leave multi-era to Phase 3 | Clean phase boundary. Cost: that coverage would then be authored against the post-swap SDK — the exact circularity VRFY-04 exists to prevent. | |
| Handshake per era, freeze pass/fail not full bytes | Cheaper, avoids five near-duplicate transcripts. Cost: a per-era response-shape change would pass. | |

**User's choice:** Yes — freeze a Legacy baseline now (the recommended option).

### Q3 — What has to go RED before the oracle is trusted?

Framing supplied: a byte-exact comparator is near-impossible to make vacuous on content, so the real vacuity risks are structural — a scenario silently stops running, a normalization rule over-matches, or empty output compares equal.

| Option | Description | Selected |
|--------|-------------|----------|
| One-time mutation matrix + permanent self-defeat guards | One-time: stray non-JSON stdout line, dropped tool, changed error code, changed `protocolVersion`. Permanent: scenario-count assertion, empty-is-never-a-match, normalization self-defeat per placeholder rule. | ✓ |
| One-time mutation matrix only | Matches existing security-audit mutation practice. Cost: nothing stops the scenario list shrinking or a rule widening later — the failure mode this repo has hit three times. | |
| Permanent self-defeat guards only | Leaner and durable. Cost: no recorded proof the oracle detected a real wire-shape change at the moment it was built, which is the claim Phase 2 leans on. | |

**User's choice:** One-time mutation matrix + permanent self-defeat guards (the recommended option).

### Q4 — What index does the oracle capture against?

Framing supplied: a shared fixture means an unrelated fixture edit invalidates every transcript at once, manufacturing exactly the bulk-regenerate pressure the Q3 guard resists.

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated frozen oracle fixture | Purpose-built tree, frozen once transcripts exist. Any diff is unambiguously a server change. Cost: a second fixture, rich enough for all 8 tools. | ✓ |
| Reuse `test/integration`'s `copyFixture` | Zero new fixture. Cost: shared, so unrelated needs cascade into bulk regeneration. | |
| Reuse it, plus declare it frozen | One fixture with the stability. Cost: constrains every future integration test, via a doc rather than a mechanism. | |

**User's choice:** Dedicated frozen oracle fixture (the recommended option).

**Not asked in this area** (user chose to move on): whether the oracle runs on every PR, whether it runs under `-race`, whether SDK-02's seam gets its own archtest.

---

## 8-agent audit method

Framing supplied: "re-run immediately before the migration lands" implies an instrument, not a procedure — agents self-update on their own cadence, so the answer has a shelf life measured in weeks.

### Q1 — What actually measures each client's negotiated protocol revision?

| Option | Description | Selected |
|--------|-------------|----------|
| Proxying capture shim swapped into each agent's config | Records the raw `initialize` frame verbatim, then proxies stdio to the real server so the agent keeps working. Independent of stderr surfacing; captures capabilities and observes probe spawns. | ✓ |
| Read the VRFY-03 stderr line from each agent's MCP log | Zero new surface; doubles as a real-world reachability test for the diagnostic. Cost: stderr handling varies per agent; no capabilities or probe behavior. | |
| Documented manual wire-capture procedure | No code to maintain. Cost: "re-run before migration" becomes eight manual sessions; documentary controls are not re-verifiable. | |

**User's choice:** Proxying capture shim (the recommended option).

### Q2 — What is the completeness bar when a roster agent isn't installable?

| Option | Description | Selected |
|--------|-------------|----------|
| Measure what you can; UNMEASURED rows structurally distinct | MEASURED (frame + date) or UNMEASURED (blocking reason); a doc-sourced value may appear only in a separately labelled unverified column. | ✓ |
| All 8 measured — install whatever it takes | Strongest evidence; also re-proves `codegraph install` against all 8 current client versions. Cost: an account wall or platform gap stalls the phase and blocks Phase 2. | |
| One client per underlying MCP SDK family | Far less work for arguably the same wire coverage. Cost: the family mapping is itself a doc-sourced claim — reintroduces the inference the requirement exists to eliminate. | |

**User's choice:** Measure what you can; UNMEASURED rows structurally distinct (the recommended option).

### Q3 — What does each audited row record?

Framing supplied: PITFALLS Pitfall 8 — a client defaulting its stdio transport to `'auto'` era-negotiation spawns a throwaway probe process plus the real session process on every launch, and stalls for the full probe timeout against a server that cannot answer `server/discover` yet.

| Option | Description | Selected |
|--------|-------------|----------|
| Version + negotiated revision + declared capabilities + probe behavior | Probe column is Pitfall 8's early warning, free to collect while the shim is in place; informs whether Phase 3's `server/discover` work is latency-urgent. | ✓ |
| Version + negotiated revision only | The literal VRFY-05 minimum. Cost: leaves Pitfall 8 unmeasured going into Phase 3. | |
| All of the above plus tool-cache behavior across reconnect | Would make Pitfall 5 triage a lookup. Cost: materially more work per agent for a client defect codegraph cannot fix — arguably a troubleshooting doc, not an audit column. | |

**User's choice:** Version + negotiated revision + declared capabilities + probe behavior (the recommended option).

### Q4 — Where do the three scoping artifacts live?

| Option | Description | Selected |
|--------|-------------|----------|
| SEP table + agent audit in `docs/`, Team Scale read-out in `.planning/` | Follows the `docs/FLAG-PARITY.md` / `docs/RELEASE-PROCEDURES.md` precedent for artifacts with ongoing operational value; keeps future-milestone strategy where future planning looks. | ✓ |
| All three in `docs/` | One home, all discoverable. Cost: a read-out for an unscoped future milestone reads oddly in shipped user docs and will go stale unowned. | |
| All three in `.planning/phases/01-*/` | Nothing lands in `docs/` until it has earned it. Cost: the agent audit is what a user hitting "my tools disappeared" would most want to find. | |

**User's choice:** SEP table + agent audit in `docs/`, Team Scale read-out in `.planning/` (the recommended option).

**Not asked in this area** (user chose to move on): whether the shim is a hidden subcommand / env var / separate binary, whether it self-destructs after one capture, whether the audit is a Phase 2 gate or merely precedes it.

---

## Negotiated-version stderr line

Framing supplied: VRFY-03 exists to close the "no tools" ambiguity, and logging the protocol version alone only closes half of it because codegraph's catalog is already dynamic on `hasIndex` and returns zero tools with no error.

### Q1 — What does the always-on session line report?

| Option | Description | Selected |
|--------|-------------|----------|
| Negotiated + requested version, client identity, advertised tool count | The tool count collapses the ambiguity in one read; requested-vs-negotiated makes a silent downgrade visible. Costs nothing extra to emit. | ✓ |
| Negotiated version + client identity | Exactly what Pitfall 4 recommends and VRFY-03 requires. Cost: leaves the index side to a separate `codegraph status` run; hides a downgrade. | |
| Negotiated version only | Narrowest surface. Cost: cannot tell which client or build produced it. | |

**User's choice:** Negotiated + requested version, client identity, advertised tool count (the recommended option).

### Q2 — What shape is the line?

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed prefix + `key=value` pairs | Greppable by prefix, parseable without a JSON decoder, readable when pasted from an agent log, consistent with existing plain-stderr diagnostics. | ✓ |
| JSON object, one line | Machine-first, non-breaking field additions. Cost: introduces a structured-logging convention the repo has nowhere else; worse to read as a single pasted line. | |
| Human prose sentence | Most readable. Cost: shim and oracle must parse prose, making the wording load-bearing. | |

**User's choice:** Fixed prefix + `key=value` pairs (the recommended option).

### Q3 — How does the oracle treat stderr?

Framing supplied: stderr also carries watcher diagnostics, Pebble's `Errorf` path, and timing-dependent output; freezing it makes each of those a normalization rule or a flake, and a flaky oracle gets relaxed.

| Option | Description | Selected |
|--------|-------------|----------|
| stdout byte-frozen; stderr asserted by targeted check only | One targeted per-scenario assertion: the session line is present exactly once, parses, carries expected keys. | ✓ |
| Freeze both streams | Would catch a new stderr-noise regression (the HYG-01 failure class). Cost: the first flake creates pressure to widen rules until they swallow real signal. | |
| stdout frozen; stderr captured but not asserted | Forensic value without flakiness. Cost: an unasserted capture is not a gate, and VRFY-03's point is that the line is reliably there. | |

**User's choice:** stdout byte-frozen; stderr asserted by targeted check only (the recommended option).

### Q4 — Is the session line a stability contract?

| Option | Description | Selected |
|--------|-------------|----------|
| Additive-only: prefix and existing keys frozen, new keys may be added | Documented and enforced by a format test; Phase 3 can add `discover=` / `cacheable=` without a contract break. | ✓ |
| Fully frozen — exact line shape is the contract | Strongest guarantee, simplest assertion. Cost: forces a second line or a contract break exactly when Phase 3 wants the diagnostic to say more. | |
| Internal diagnostic — no contract | Zero maintenance burden. Cost: it becomes a de facto interface the moment users are asked to paste it. | |

**User's choice:** Additive-only (the recommended option).

---

## Claude's Discretion

The user explicitly delegated these at the close-out prompt, with the recorded leanings as starting positions:

- **VRFY-02** — the repo-owned protocol-version constant's pre-migration value (leaning `"2025-11-25"`) and the guard mechanism (leaning a `go/packages` archtest with a self-defeat guard, over a CI grep, per v1.0 Phase 4/6 precedent and this repo's inverted-`rg -qv` history). The guard must forbid the *class* of SDK-owned protocol constant so it survives Phase 2's swap.
- **SDK-02** — the shape of the `internal/mcp.Server` seam. `BuildServer`'s `*server.MCPServer` return type is the leak; the seam must own the serve call. Concrete type vs. interface is the planner's call.
- **Tool location and CI run scope** — where the capture tool lives, and whether all ~18 scenarios gate every PR or a narrower set gates while the full sweep runs on a schedule.

## Deferred Ideas

- **Per-agent tool-cache-across-reconnect observation** (PITFALLS Pitfall 5) — explicitly offered as a third option in the audit-row question and not chosen. Belongs in a troubleshooting note carrying the "rename the server entry" discriminator, not as an audit column. Revisit if a real "tools missing" report lands during v0.3.0.
- **Freezing stderr into the transcripts** — rejected on flake risk; revisit as a separately-scoped stderr transcript if a stderr-noise regression ever ships.
- **Extending the oracle to the interactive TUI surface** — already tracked as backlog 999.2; same failure shape, different surface and harness.

No scope creep was raised during the discussion — every area stayed inside the phase boundary.
