---
phase: 02
slug: golden-harness-re-authoring-re-freeze
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 5
threats_closed: 5
threats_mitigated: 4
threats_accepted: 1
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. All 4 PLAN files carried a `<threat_model>` block, so
`register_authored_at_plan_time: true`. With `asvs_level: 1` and `threats_open: 0`, the L1
short-circuit applies; mitigations were verified at grep depth against the tree at commit `3598bb2`.

**Register shape note.** Unlike Phase 1 (which numbered threats per plan), Phase 2 uses a **shared
threat-ID namespace**: the same five IDs recur across all four plans, each re-asserted at the
severity and disposition appropriate to that plan. `T-02-01`, for instance, is `high / mitigate` in
02-01 and 02-02 (which perform sweeps) and `low / accept` in 02-03 and 02-04 (which sweep nothing).
The register below rolls each threat up to its **highest** severity across plans — the conservative
reading — and records the per-plan variation in the Notes column.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| repo root → `tools/bench` + `NOTICE` | The rename/delete sweep must not touch the retained bench corpus pin or the MIT attribution (licence, not framing) | Identifier text |
| `corpus/behavioral` (committed authored input) → tests | Committed test input is loaded as trusted data; a case map naming a wrong symbol silently weakens the assertion | Case map, source fixtures |
| fetched locked corpus (out-of-tree) → gocapture / tests | Untrusted third-party tree, read **only after** the four-part integrity check | Third-party source |
| capture write → committed golden path | A capture failure could truncate or misplace a golden if written in place | Golden transcript bytes |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Tampering | sweep scope vs `tools/bench` + `NOTICE` | **high** | mitigate | Sweep scoped to `testdata/golden/` plus four named forced dependents; verify block asserts no path under `tools/bench/`, `bench.yml` or `NOTICE` appears in the diff. **Verified:** `NOTICE` retains 2 `colbymchenry` attribution hits, `tools/bench/realcorpus` retains 6 `weft-go` pin hits, `bench.yml` present | closed |
| T-02-02 | Spoofing / supply-chain | capture path | **high** | mitigate | The TS/network capture path is deleted and gocapture is the sole capture authority; the MCP-surface capture uses the **in-process** `BuildServer` + in-tree go-sdk client, never a live external server. **Verified:** `capture.sh` and `mcp-capture.mjs` absent (0), zero `.mjs`/`.ts` under `testdata/golden/`, 3 `BuildServer` references in `gocapture/main.go` | closed |
| T-02-03 | Tampering (integrity) | golden bytes / golden write path | **high** | mitigate | `writeCapture` is capture-to-**temp**-then-move: `os.CreateTemp` → non-empty assertion (*"refusing to rename onto committed path"*) → `{` envelope-marker assertion → `os.Rename`. Rename never runs on a failed capture. Backed by `TestReFrozenGoldensValid`, which reads every enumerated golden off disk. **Verified:** `gocapture/main.go:337-395`; test executed → **26/26 goldens verified** | closed |
| T-02-04 | Tampering | locked-tree resolution | **high** | mitigate | gocapture and the hermetic tests resolve corpora **only** via `internal/corpora`; `golden:regen` runs `task corpora:assert` (four-part: real `.git`, `HEAD == sha`, clean tree, `HEAD^{tree}`) **before** any locked tree is read. **Verified:** 1 `internal/corpora` import, **0** raw `CORPUS_*` env resolution in gocapture, `corpora:assert` target present and invoked by `golden:regen` | closed |
| T-02-SC | Tampering | `go.mod` / installs | low | accept | No dependency added or removed in any of the four plans; gocapture depends only on in-repo packages — see Accepted Risks | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*

**Per-plan severity variation (rolled up above):**

| Threat | 02-01 | 02-02 | 02-03 | 02-04 |
|---|---|---|---|---|
| T-02-01 | high / mitigate | high / mitigate | low / accept | low / accept |
| T-02-02 | medium / mitigate | high / mitigate | high / mitigate | medium / mitigate |
| T-02-03 | high / mitigate | medium / mitigate | high / mitigate | high / mitigate |
| T-02-04 | low / accept | low / accept | high / mitigate | high / mitigate |
| T-02-SC | low / accept | low / accept | low / accept | low / accept |

*The `low / accept` cells are "not active in this plan, carried for completeness" — the plans said so
explicitly rather than omitting the row, which is why the rollup is unambiguous.*

**Totals:** 5 threats — 4 mitigated, 1 accepted, **0 open**. High-severity: 4, all mitigated and verified.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-02-01 | T-02-SC | No package is added or removed across any of the four plans — gocapture depends only on in-repo packages — so the npm/pip/cargo Package Legitimacy Gate is not triggered. Recorded rather than omitted. | plan author (02-01…02-04) | 2026-08-14 |
| R-02-02 | T-02-01 (in 02-03, 02-04) | The sweep-boundary threat is inactive in the tooling and re-baseline plans, which perform no file sweep. Carried forward at `low / accept` for register completeness rather than dropped, so the ID does not silently vanish between plans. | plan author (02-03, 02-04) | 2026-08-14 |
| R-02-03 | T-02-04 (in 02-01, 02-02) | Corpus-resolution integrity is inactive in the identifier-rename and delete plans, which read no locked corpora. Owned by 02-03/02-04 via `task corpora:assert`. | plan author (02-01, 02-02) | 2026-08-14 |

---

## Threat Flags from Execution

`02-02-SUMMARY.md` is the only Phase 2 summary carrying a `## Threat Flags` section; it records
**"None identified."** Its verification block reports `rg -i "weft|colbymchenry|mcp-capture|capture\.sh"
testdata/golden/` = 0 hits and `rg "parity" testdata/golden/` = 0 hits, with the remaining
`synthetic-parity` matches confined to `.go` doc comments in corpus documentation — exempt per the
framing-gate rule. No threat flag was raised by any other plan.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 5 | 5 | 0 | `/gsd-secure-phase 2` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep depth against the working tree at commit `3598bb2`, plus one
executed test. Every high-severity `mitigate` threat was confirmed by locating its named control in
source — `writeCapture`'s temp→assert→rename sequence was read line by line rather than accepted from
the plan's description of it — and `TestReFrozenGoldensValid` was **run** (26/26) rather than cited.
The sweep-boundary threat (T-02-01) was verified by confirming the things that must *survive* are
still present, not by confirming the things that must vanish are gone; a sweep-scope threat is only
disproved by the former.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
