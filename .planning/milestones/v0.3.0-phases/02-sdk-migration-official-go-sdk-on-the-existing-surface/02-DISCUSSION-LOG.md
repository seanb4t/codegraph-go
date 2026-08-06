# Phase 2: SDK Migration — official go-sdk on the existing surface - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-05
**Phase:** 2-sdk-migration-official-go-sdk-on-the-existing-surface
**Areas discussed:** Divergence adjudication, Negotiated protocol version, Schema declaration style, ONE-WAY window

---

## Course correction (before any area was resolved)

The first question offered three ways to *govern* transcript divergence —
pre-authored expected diffs, observe-then-adjudicate, or compensate to
byte-identity. The maintainer rejected the framing:

> "I think we're being _way_ too careful here for an unshipped system. Do we
> really need all this oracle stuff?"

Response, on the record:

- The proposed governance process **was** over-engineered for a solo-maintained
  repo, and was dropped.
- The oracle itself is sunk cost and cheap to run (23 transcripts, ~17s), so
  keeping it was not the question worth asking.
- One premise was corrected factually: `v0.2.0` is a real released tag with a
  goreleaser/cosign/SLSA pipeline and a live `codegraph upgrade` path, so the
  system is not unshipped. Actual usage volume is the maintainer's knowledge,
  not something claimed here.
- The question was then reduced to the one thing that changes the build: the
  **bar**, not the process.

This exchange is why CONTEXT.md carries an explicit calibration note in
`<domain>`.

---

## The verification bar

| Option | Description | Selected |
|--------|-------------|----------|
| Relax to reviewed diff | No semantic change; the full oracle diff gets one human read where every changed line is explainable. Requires editing ROADMAP criterion 1 and REQUIREMENTS SDK-01. | ✓ |
| Keep byte-identity | Compensate in code until all 23 transcripts pass unchanged. Strongest gate, but likely requires forcing a protocol version that may not be injectable. | |
| Drop the oracle from Phase 2 | Swap the SDK, run the normal suite, ship. Fastest, but accepts that a silent tools-not-advertised regression reaches a client before anyone notices. | |

**User's choice:** Relax to reviewed diff.
**Notes:** Byte-identity cannot separate a cosmetic `omitempty` change from a
`protocolVersion` change; a human reading the diff can. The edit also forced
restating criterion 2, whose "never regenerated, relaxed, or re-baselined"
wording directly contradicts the new bar. Both criteria plus REQUIREMENTS
`SDK-01` were edited in-session and verified structurally intact
(`phase_count: 5` unchanged).

---

## Negotiated protocol version

| Option | Description | Selected |
|--------|-------------|----------|
| Whatever the SDK negotiates | Take five-era negotiation as inherited, per ROADMAP's own goal wording. Zero code. Means advertising 2026-07-28 before Phase 3 implements that revision's obligations. | ✓ |
| Pin back to 2025-11-25 | Keep Phase 2 a pure backend swap; let Phase 3 move deliberately. May not be possible — go-sdk's initialize handler calls a package-level `negotiatedVersion()` with no visible `ServerOptions` hook. | |
| Decide after research | Defer to the researcher to confirm whether any server-side version control exists. | |

**User's choice:** Whatever the SDK negotiates.
**Notes:** Consistent with ROADMAP's existing goal text. Carries the known
consequence that `legacy-unsupported-2026-07-28.golden` moves to `2026-07-28`
and the scenario needs renaming. Research is still required — CONTEXT.md D-06
requires source confirmation that no injection point exists, to this repo's own
standard (Phase 1 enumerated every `func With…` before concluding absence).

---

## Schema declaration style

| Option | Description | Selected |
|--------|-------------|----------|
| Struct tags + reflection | One tagged input struct per tool; `mcp.AddTool` infers the schema. Idiomatic, deletes the builder boilerplate, gives handlers typed arguments. Schema becomes a derived artifact. | ✓ |
| Hand-authored jsonschema | Explicit `*jsonschema.Schema` passed as `Tool.InputSchema` (supported — the field is typed `any`). Direct control, readable in one place, but duplicated against argument extraction. | |
| Tags now, patch where needed | Infer, then mutate `in.Properties[...]` before `AddTool`. Documented SDK path; more moving parts. | |

**User's choice:** Struct tags + reflection.
**Notes:** Viable specifically *because* the byte-identity bar was relaxed
first — hand-authoring's main argument was byte control. Scouting also found
zero enum constraints in `tools.go` today, so SDK-05's headline risk ("enum
constraints dropped by reflection") does not apply as written.

---

## ONE-WAY window

Not put to a formal question. With the byte-identity bar relaxed, the value of
extending the frozen set before `mark3labs` leaves `go.mod` dropped
substantially, and the maintainer had just called for less ceremony. Recorded
in CONTEXT.md `<deferred>` as a **deliberate, irreversible skip** flagged for
the maintainer to reverse if desired before the swap PR lands, rather than
silently dropped.

---

## Claude's Discretion

- Migration shape — sibling implementation behind the existing `Server`
  interface vs. in-place replacement (leaning: sibling; the seam's own doc
  comment anticipates it).
- `BuildServer`'s return type — whether to keep returning an SDK concrete type
  or close the last leak, given 17 test call sites.
- Whether to correct the `annotations` hints (`destructiveHint:true` on
  read-only query tools looks wrong) — allowed, but it is a semantic change and
  must be surfaced in the diff review.

## Deferred Ideas

- Extending the frozen set to cover handler-level required-argument validation
  before the ONE-WAY window closes — declined, reversible only until the swap.
- Correcting the `annotations` hints, if not taken here — belongs with Phase 3's
  deliberate wire changes.
- Documenting client-side `tools/list` caching bugs as a known confound
  (`anthropics/claude-code#41123`, `#40025`, `#50515`) — ships alongside the
  milestone, not a Phase 2 build task.
