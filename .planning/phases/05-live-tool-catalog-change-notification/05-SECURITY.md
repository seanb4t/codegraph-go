---
phase: 05
slug: live-tool-catalog-change-notification
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: **authored at plan time**. The phase's single plan (`05-01`) carries a
parseable `<threat_model>` block, so this audit verifies that each declared mitigation
is present in the implementation — it does not retroactively scan for new threats.
`05-01-SUMMARY.md` emitted no `## Threat Flags` section; the plan register is the
complete source.

**Threat-ID scope note.** `T-05-*` here means *phase* 05. This differs from
`01-SECURITY.md`, where `T-01-*`…`T-07-*` are *plan* numbers within phase 01 — so
that file also contains a `T-05-01`, denoting an unrelated threat. Do not merge
threat IDs across phase files.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| agent client → server stdin | Untrusted JSON-RPC frames, including the `_meta` triple and the `notifications` opt-in object that decides what this session is subscribed to | Protocol metadata, subscription opt-in set |
| server → agent client stdout | Notification frames cross OUTSIDE request/response correlation — a subscriber receives bytes it did not individually ask for, on a stream held open for the session's lifetime | `notifications/tools/list_changed` frames |
| filesystem (`.codegraph/`) → server | On-disk index presence drives catalog mutation, which is what triggers notification delivery; the trigger is filesystem state, not client input | Index presence (boolean) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Denial of Service | `subscriptions/listen` stream lifetime — parked handler goroutine + session-held `toolChangeSubscriptions` entry | low | mitigate | Deadline and kill path verified present: `test/wireoracle/capture.go:240` (`context.WithTimeout(ctx, 30*time.Second)`), `:327`, `:339-350` (`Process.Kill()` + named deadline errors), `:268-270` (unconditional deferred `cmd.Wait()`). Capture error is fatal, not tolerated (`oracle_test.go:63-66`). **Measured:** scenario capture 0.69 s, full 28-scenario suite 13.5 s — neither the deadline nor the kill path was reached, so a live listen stream does not prevent exit on stdin EOF. See audit caveat 1. | closed |
| T-05-02 | Information Disclosure | `notifications/tools/list_changed` payload crossing to a subscriber | low | mitigate | `test/wireoracle/anchors.go:399-449` — `assertToolsListChangedNotification` decodes `params` into `map[string]json.RawMessage` and enforces the key set is exactly `{_meta}` (`:429`), plus exactly-one-frame (`:418`) and subscription-id correlation (`:441-448`). Registered against a **fresh** capture (`anchors.go:195-202` → `oracle_test.go:589`), never the golden bytes. Wire confirms: golden line 4 carries `_meta` only, no catalog content. | closed |
| T-05-03 | Spoofing | Misspelled `notifications` opt-in acked with a `subscriptionId`, an empty subscription set, and no error — client believes it is subscribed and is not (05-CONTEXT D-02) | low | accept | See AR-05-01. Discriminator verified non-vacuous: `assertSubscriptionAckEcho` (`test/wireoracle/anchors.go:345-388`) compares by **exact set equality** (`:368`, `:371-375`), never "present"/"non-empty", and `t.Fatalf`s on an absent ack frame (`:387`). Mutation 6 records the verbatim RED (`test/wireoracle/MUTATION-PROOF.md:437-530`). | closed |
| T-05-04 | Tampering | Catalog mutation driven by construction-time `startPath`, never a request argument | low | accept | See AR-05-02. `internal/mcp/server.go:540` — `query.ResolveCodegraphDir(startPath)` inside the per-request re-check, boundary stated at `:494` and `:528-531`. Provably unchanged this phase: `git diff --exit-code 36a3efc..HEAD -- internal/` → exit 0. | closed |
| T-05-SC | Tampering | npm/pip/cargo/go dependency installs | high | mitigate | `git diff --exit-code 36a3efc..HEAD -- go.mod go.sum` → **exit 0**, byte-unchanged across all three task commits (`924384a`, `db89e92`, `c8d7f7e`). Every task carries the criterion (`05-01-PLAN.md:279`, `:395`, `:476`); no install command appears anywhere in the plan. Phase Go diff touches only `test/wireoracle/{anchors,capture,oracle_test,scenarios}.go` and adds **zero** new external imports. VRFY-01 holds: `go list -deps ./test/wireoracle \| rg -c modelcontextprotocol/go-sdk` → 0. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-03 | go-sdk answers a misspelled `notifications` opt-in with a `subscriptionId`, an empty subscription set, and no error. Not changeable from codegraph-go, and not a correctness risk: Phase 3's `ttlMs: 0` means no client caches a catalog on the strength of a subscription. Compensating control is `assertSubscriptionAckEcho` (`test/wireoracle/anchors.go:345`) as the recorded set-equality discriminator, proven non-vacuous by mutation 6 (`MUTATION-PROOF.md:437`), so the dead-subscription shape is documented rather than latent. | seanb4t | 2026-08-06 |
| AR-05-02 | T-05-04 | Catalog mutation resolves from construction-time `startPath`, never a request argument — already closed by Phase 3 (T-03-13 / T-03-14) at `internal/mcp/server.go:540`. Recorded here only because notification delivery makes the mutation externally observable for the first time. This phase changed no production code (`git diff --exit-code 36a3efc..HEAD -- internal/` → exit 0), so the boundary is unchanged. | seanb4t | 2026-08-06 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 5 | 5 | 0 | Claude Opus 5 (`gsd-security-auditor` subagent, `/gsd-secure-phase 05`, ASVS L1) |

### Audit 2026-08-06 — notes

Audited retroactively, after the phase was verified and the milestone archived, to
close the coverage gap left by phases 02–05 shipping without a `SECURITY.md`. The
`gsd-security-auditor` subagent was spawned rather than taking the workflow's L1
short-circuit, so this file rests on measured evidence rather than grep-depth
presence checks. All verification was run against source and the built binary.

**Live-vs-dead subscription probe quality.** The wire evidence deliberately does not
rest on "an ack came back" — the milestone's own recorded gotcha is that a typo'd
opt-in is acknowledged with an empty set and no error, and that a live subscription
emits only the ack until a change occurs. Three independent discriminators were
confirmed: (1) set equality on the ack's `notifications` map, where `{}` fails
(`anchors.go:368-375`), proven by mutation 6's verbatim RED; (2) an exact frame
**count** of one `notifications/tools/list_changed` (`anchors.go:418`), where zero
fails naming the observed count; and (3) `AwaitAfterRequest[3] =
notificationToolsListChangedMethod` (`scenarios.go:1086`), which makes the capture
structurally unable to complete without an observed notification frame — mutation 5's
recorded failure is the 30 s capture deadline, so a dead subscription cannot produce
a passing capture at all.

**Caveat 1 — T-05-01's gate is weaker than its prose.** `Capture` completes when
`expectedResponseIDs()` is satisfied; the deferred `cmd.Wait()` (`capture.go:268-270`)
discards its error, and `exec.CommandContext` kills the child at the 30 s deadline. A
regression where a parked listen stream prevented exit on stdin EOF would therefore
surface as a ~30 s-per-capture wall-clock slowdown, **not** as a named test failure.
The property is true today by direct measurement (0.69 s), so the threat is closed —
but the backstop edge is a timing signal, not a loud assertion. This belongs to the
repo's documented class of gates that are weaker than they read.

**Unregistered surface.** The absence of a `## Threat Flags` section in the summary
was not treated as proof of no new surface. The phase's only new code is test-harness
surface (`Scenario.AwaitAfterRequest`, `NoResponseRequests`, `expectedResponseIDs`) in
the non-shipping `test/wireoracle` package, and production `internal/` is byte-unchanged.
No unregistered production attack surface.

**Scope note (L1).** ASVS L1 verifies that each declared mitigation is *present*. It does
not perform L2 boundary-placement analysis or L3 end-to-end trace verification.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-06
