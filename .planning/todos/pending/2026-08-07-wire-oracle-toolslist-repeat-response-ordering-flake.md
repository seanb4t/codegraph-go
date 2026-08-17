---
created: 2026-08-07T17:53:31.671Z
title: Wire oracle toolslist-repeat response ordering flake
area: mcp
severity: major
files:

  - test/wireoracle/scenarios.go:559-573
  - test/wireoracle/oracle_test.go:130
  - test/wireoracle/oracle_test.go:291
  - .github/workflows/ci.yml:104-110

audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
---

## Problem

`TestFrozenTranscriptsMatch/toolslist-repeat` failed once on Linux CI under
parallel load, then **passed on a re-run of the identical commit**. That
re-run is the decisive control: the same tree both failing and passing means
the failure is nondeterministic, not introduced by the change under test. The
flake is now latent on `main`.

**Provenance.** Observed 2026-08-07 on PR #29's first CI run — run
`31202833332`, job `92946682482`, at the commit that merged as `80d2e0a`.
Full evidence in the PR #29 conversation.

**Symptom.** The two transcripts are byte-identical except for one field:

```
got:  {"jsonrpc":"2.0","id":3,"result":{...tools...}}
want: {"jsonrpc":"2.0","id":2,"result":{...tools...}}
```

The tool payloads match exactly. Line 2 of the normalized transcript held the
**id-3** response where the id-2 response belonged — so the id-2 response was
either dropped or overtaken by id-3.

**Why this contradicts a documented invariant.** The scenario
(`test/wireoracle/scenarios.go:564`) sends exactly three requests:

```go
initializeRequest(1),
toolsListRequest(2),
toolsListRequest(3),
```

and its own comment at :559-563 asserts this cannot happen:

> Both are "tools/list", not "tools/call", so both are handled synchronously
> in request order — no worker-pool race.

The evidence contradicts that comment. Either the claim is wrong, or it is
true of the server but not of the transcript reader.

**Load- and platform-dependence** (the useful diagnostic signal):

| Condition | Result |
|-----------|--------|
| darwin, isolated, `-count=15` | 0 failures |
| Linux CI, inside `task test:unit` (all packages in parallel) | 1 failure in 2 runs |

`task test:wireoracle` alone was green locally. It only failed where it runs
under contention from every sibling package.

**Why this matters more than an ordinary flake.** `ci.yml:104-110` documents
the wire oracle as "a required PR leg from the moment it lands, never relaxed
or re-baselined" (VRFY-01/VRFY-04). It is the one gate the v0.3.0 SDK
migration was verified against. An intermittent failure there is corrosive
twice over: it blocks merges at random, and it teaches people that a red
blocking gate means "re-run CI" — which is exactly how a real regression gets
waved through later.

## Solution

TBD — establish the root cause before changing anything.

**Leading hypothesis, to VERIFY not assume:** request dispatch in
`modelcontextprotocol/go-sdk` v1.7.0 (adopted in v0.3.0 Phase 2) may not
guarantee in-order response emission for pipelined same-method requests,
which would make the scenario comment's synchronous-ordering claim
SDK-dependent rather than guaranteed.

Suggested order of work:

1. **Reproduce deterministically.** The failure is contention-driven, so
   reproduce under load rather than with `-count`: run the wireoracle package
   concurrently with the rest of the suite, on Linux, ideally in a container
   matching the runner. A repro that only appears once in two CI runs is not
   yet a repro.

2. **Split server from harness.** Determine whether the server emitted
   responses out of order, or the transcript reader consumed them out of
   order. These need different fixes and the current evidence does not
   distinguish them. Capturing raw stdio bytes with arrival timestamps,
   before normalization, separates the two.

3. **Settle the ordering question at the source.** Read what the go-sdk
   actually guarantees for concurrent request handling. If it does not
   promise in-order responses for pipelined requests, the scenario's comment
   is the defect and the invariant must be re-stated (or enforced) rather
   than assumed.

4. **Fix, then prove it RED first.** Per this repo's recurring lesson, any
   fix must be demonstrated against a test that fails before it and passes
   after — not merely wired up.

**Do NOT re-baseline the frozen transcript to make this pass.** The transcript
is the oracle; regenerating it to match observed output would destroy the
only evidence that an invariant was violated, and `check:transcript-freeze`
exists to prevent exactly that.
