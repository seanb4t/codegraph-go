# Phase 2: Guards, CI Wiring & Docs Burn-down - Pattern Map

**Mapped:** 2026-09-15
**Files analyzed:** 11 (no new directories; every item is a create-or-edit inside an existing file's own established shape)
**Analogs found:** 11 / 11 — this phase creates no genuinely new architectural shape; every file's closest analog is another file of the identical role sitting a few lines away in the same file, or its own direct predecessor phase artifact.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.github/workflows/ci.yml` (`test` job: 3 new steps — syft install, `check:gonum`, `check:no-force-layout`) | config (CI workflow step) | event-driven (PR-triggered) | same file, `test` job's existing `pnpm audit gate (BLD-06)` / `SPA build drift guard (BLD-03)` steps (`ci.yml:158-186`) | exact |
| `.github/workflows/ci.yml` (`test` job: ruleset-drift CI-only step, GRD-12) | config (CI workflow step) | request-response (external GitHub REST API call) | `.github/workflows/post-release-verify.yml`'s defensive `jq`-availability shell pattern (named in RESEARCH Don't-Hand-Roll; cite for the `set -euo pipefail` / named `::error::` shape) — no in-repo step already calls an external read-only API from `ci.yml` itself, so this is the closest available shape, not a byte-for-byte precedent | role-match |
| `.github/required-status-checks.txt` (NEW data file, D-07) | config (data file) | batch (newline-delimited list read by two independent consumers) | no existing precedent of this exact shape in the repo (RESEARCH A1); closest sibling is any newline-delimited fixture consumed by both Go and shell — none found, so this is genuinely new-shape, format free at Claude's discretion | no analog (see below) |
| `internal/upgrade/taskfile_shape_test.go` (`requiredCheckNames` var -> data-file loader, D-07) | test (fixture loader) | file-I/O (transform: `os.ReadFile` -> `[]string`) | same file, `TestGoreleaserPinParity`'s `os.ReadFile` + `t.Fatalf`-on-error pattern (`:808-819`); and the file's own `TestRequiredCheckNamesPreserved` test body it must keep passing (`:749-786`) | exact |
| `web/src/lib/components/ui/{button,command,dialog,input,input-group,table,tabs,textarea}/*.svelte` (re-vendor, D-10/D-11) | component | transform (regenerate-from-registry, not hand-authored) | `web/src/lib/components/ui/button/button.svelte`'s own most recent vendoring commit shape (`205da685`, "feat(03-06) vendor shadcn-svelte Command component, human-approved" per WINDOWS #31's own history note) — regenerate via the identical CLI invocation `web:components:drift`'s own Taskfile target already uses, never hand-edit | exact |
| `docs/RELEASE.md` (dependency paragraph, D-13) | config (doc, prose) | transform (value-only rewording, no drift test) | same file's own "Corrected 2026-08-01" historical-correction block (`docs/RELEASE.md:145` per RESEARCH Pitfall 6) — the established in-file convention for framing a wording fix without inventing a new heading shape | exact |
| `docs/RELEASE.md` (provenance wording, D-14 — census target, likely zero live hits) | config (doc, prose) | transform | same file's already-correct provenance sentence at `docs/RELEASE.md:101` (per RESEARCH Code Examples, "subjects are the binaries") — the wording standard every census hit must be rewritten to match | exact |
| `SECURITY.md` (one sentence, D-15) | config (doc, prose) | transform | same file's existing two-scanner disjoint-scope paragraph (`SECURITY.md:61-73`) — extend in place, same paragraph, same voice | exact |
| `.planning/WINDOWS.md` (#16, #33 -> `fixed`; #20/#21/#34 stay `open`, annotated in SUMMARY/STATE.md) | config (tool-owned generated ledger) | CRUD (tool-verb-mediated row update) | same file, any prior `fixed` row with a `resolved_at` timestamp (e.g. row 36, `id 36`, `.planning/WINDOWS.md:53`) — **never hand-edit; only `gsd-tools windows fixed <id>` may touch this file** (planning-artifacts rule) | exact |
| `.planning/STATE.md` (Pending Todos section, DOCS-11) | config (tool-owned generated section) | transform (regenerate from tool's own renderer) | `gsd-tools init todos`'s `pending_todos_markdown` field (bullet-list shape, `gsd-core/bin/lib/init.cjs:2024`) — **the hand-authored table (`.planning/STATE.md:319-329`) must be replaced by the tool's own bullet rendering, not hand-edited into a corrected table** (RESEARCH Pitfall 4) | exact (tool-shape correction) |
| `.planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md` (frontmatter: status/consumed_by/consumed_on) | config (frontmatter, values-only) | transform | `.planning/seeds/SEED-002-homebrew-installation-path.md`'s frontmatter (`id`, `status: implemented`, `consumed_by`, `consumed_on`) — copy the field set verbatim, fill in SEED-001's own values | exact |
| `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` (NEW, GRD-09 replay evidence) | test (mutation-log artifact) | batch (one-shot replay transcript, not a repeatable script) | `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` — the exact file shape to follow (pre-mutation cleanliness gate / mutation applied / observed failure / revert / byte-clean proof sections) | exact |

## Pattern Assignments

### `.github/workflows/ci.yml` — new `test`-job steps for `check:gonum` / `check:no-force-layout` (GRD-11, D-04)

**Analog:** same file, `test` job, existing JS-gate step chain (`ci.yml:131-203`)

**Placement pattern** (verified live, lines 131-203): every gate step in this job is a `- name:` block with a multi-line `#`-prefixed rationale comment immediately above it, explaining *why* the step exists and *why* it sits where it does (never a bare `run:` with no comment). Copy this convention exactly — a plain `run: task check:gonum` with no rationale comment would be out of house style.

**Ordering constraint (Pitfall 2/3, hard requirement):** both new steps must be placed **after** `- name: Set up Node` (`ci.yml:131`) — both `check:gonum` and `check:no-force-layout` precondition on `command -v node`. `check:gonum`'s SBOM half additionally preconditions on `syft`, which is **not installed anywhere in `ci.yml`'s `test` job today** — a new `- name: Install syft` step (copy verbatim from `release.yml` / `linux-cross-canary.yml`'s pinned `anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0.24.0`) must precede the `check:gonum` step specifically. Exact position within the JS-gate chain is Claude's discretion (D-04 only requires "next to the pnpm-audit and web:drift steps"); simplest is appending after `proto:drift`/`docs:cli:drift` and before the Go test steps, since those are already the tail of the "drift-guard cluster."

**Core step shape to copy:**
```yaml
      - name: SPA build drift guard (BLD-03)
        run: task web:drift
```
becomes (illustrative, same one-line `run: task <target>` shape, no shell scripting needed — the guard logic lives entirely in the Taskfile target per Phase 7's own precedent of never mirroring a script's assertion in the caller):
```yaml
      - name: Install syft
        uses: anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0.24.0

      - name: Gonum dependency-shape guard (GRD-11/GRF-10)
        run: task check:gonum

      - name: No-force-layout guard (GRD-11/GRF-06)
        run: task check:no-force-layout
```

**Error handling pattern:** none needed at the CI-step level — each Taskfile target owns its own precondition failures and non-zero exit; per the Anti-Patterns list in RESEARCH, do **not** add a second CI-only assertion of the same property the Taskfile target already checks.

---

### `.github/workflows/ci.yml` — ruleset-drift CI-only step (GRD-12, D-06)

**Analog:** no exact in-repo precedent for "CI step reads a live external read-only API and fails hard on any non-2xx/empty/unparseable response" — nearest shape is the defensive `command -v jq` pattern already used elsewhere in this project's CI scripting per RESEARCH's Don't-Hand-Roll table (`post-release-verify.yml`). Build fresh per the RESEARCH-verified transcript (already a fully-worked, live-tested shell block — copy it directly rather than re-deriving):

```yaml
      - name: Ruleset drift check (GRD-12)
        run: |
          set -euo pipefail
          resp="$(curl -sS -w '\n%{http_code}' \
            -H 'Accept: application/vnd.github+json' \
            "https://api.github.com/repos/${{ github.repository }}/rulesets/20157557")"
          code="$(printf '%s' "$resp" | tail -n1)"
          body="$(printf '%s' "$resp" | sed '$d')"
          if [ "$code" != "200" ] || [ -z "$body" ]; then
            echo "::error::ruleset-drift: GET rulesets/20157557 returned HTTP ${code} (or an empty body) — hard failure, never a skip" >&2
            exit 1
          fi
          live="$(printf '%s' "$body" | jq -r '
            .rules[] | select(.type=="required_status_checks")
            | .parameters.required_status_checks[].context' | LC_ALL=C sort)" \
            || { echo "::error::ruleset-drift: jq could not parse the ruleset response" >&2; exit 1; }
          fixture="$(LC_ALL=C sort .github/required-status-checks.txt)"
          echo "ruleset-drift: live has $(printf '%s\n' "$live" | wc -l) contexts, fixture has $(printf '%s\n' "$fixture" | wc -l)"
          if [ "$live" != "$fixture" ]; then
            echo "::error::ruleset-drift: live ruleset and .github/required-status-checks.txt disagree" >&2
            diff <(printf '%s\n' "$live") <(printf '%s\n' "$fixture") >&2 || true
            exit 1
          fi
```
Verified live this session (RESEARCH Pattern 3 / Code Examples): unauthenticated `curl` against this public repo's ruleset returns 200 with no `Authorization` header; `contents: read` (already granted at `ci.yml:42`) is sufficient for the automatically-granted `Metadata: read` scope — no `permissions:` block change needed.

**Error handling pattern (hard rule, D-06):** every failure path (`code != 200`, empty body, unparseable JSON) is a **named, non-zero exit** — never `continue-on-error: true`, never a conditional skip. This is the load-bearing anti-pattern this step exists to avoid (GRD-07's declined "skip-clean offline" clause).

---

### `.github/required-status-checks.txt` (NEW, D-07)

**No analog** — genuinely new shape (see "No Analog Found" below). Format: newline-delimited, one context string per line, no comments, no trailing blank-line ambiguity (both the Go `os.ReadFile` + line-split side and the shell `sort` side must treat blank lines defensively per RESEARCH's Security Domain V5 note). Seed its initial 8-line content from the **fixture's current 7 entries** (`internal/upgrade/taskfile_shape_test.go:77-85`) plus the D-08 addition once the maintainer's ruleset PUT lands:
```
test
govulncheck (DIST-03, blocking)
reproducibility (double-build hash-diff, DIST-04)
perf regression gate (PERF-02, INDX-06)
actionlint (workflow static analysis)
goreleaser check (config validation, DIST-01)
pr-title
tmux e2e (real-pty harness, TTY-01..TTY-07)
```

---

### `internal/upgrade/taskfile_shape_test.go` — `requiredCheckNames` becomes a data-file loader (D-07)

**Analog:** same file, `TestGoreleaserPinParity`'s `os.ReadFile` + `t.Fatalf` pattern (lines 808-819) — the established in-file idiom for "read a sibling file, fail loudly if unreadable."

**Imports pattern:** no new imports — `os`, `strings`, `sort` are already imported in this file (used by `TestRequiredCheckNamesPreserved` itself, `:749-786`).

**Core loader pattern to write** (illustrative, matching the file's existing var-then-loader idiom — replace the `var requiredCheckNames = []string{...}` literal at `:77-85` with a function the test calls, or a package `init()`-populated var read from the file; either satisfies D-07 as long as a missing/empty file is a hard `t.Fatalf`, never a silent empty slice):
```go
// requiredCheckNames loads the shared context list (D-07) — one list, no
// bash parsing of Go source, no duplication with the CI ruleset-drift step.
func requiredCheckNames(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../../.github/required-status-checks.txt")
	if err != nil {
		t.Fatalf("read required-status-checks.txt: %v", err)
	}
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, line)
	}
	if len(names) == 0 {
		t.Fatalf("required-status-checks.txt: found zero non-blank lines — a missing or emptied fixture must fail loudly, never pass vacuously")
	}
	return names
}
```
**Validation pattern:** the zero-length-after-trim guard mirrors the file's own `TestRequiredCheckNamesPreserved_ZeroJobsIsError` discipline (`:788-798`) and the project-wide rule `84d1gfpywd` (never let an empty collection read as "nothing required" = pass).

**Test-name correction (Pitfall 1):** the test to update is `TestRequiredCheckNamesPreserved` (`:749`), **not** `TestRequiredCheckJobsExist` — that name does not exist in this file.

---

### `web/src/lib/components/ui/{8 families}/*.svelte` (re-vendor, GRD-10, D-10/D-11)

**Analog:** the vendored components' own most recent human-approved vendoring commit (`205da685`) and `web:components:drift`'s own regeneration mechanics.

**Core regeneration pattern** (per Don't-Hand-Roll table and `components-drift.yml:90`): run the identical CLI invocation the drift target itself uses, against the real `web/` tree, under Corepack-pinned pnpm (never this dev machine's bare pnpm — Environment Availability table confirms Corepack is not installed locally and would reproduce WINDOWS #31's already-refuted local-toolchain confound):
```
pnpm dlx shadcn-svelte@1.5.1 add button command dialog input input-group table tabs textarea -y -o
```
**Post-regen gate sequence (D-12, hard order):** `pnpm check` (0 errors) -> `cd web && pnpm vitest run` (0 exit, zero "unhandled errors" lines) -> `task web:build && task web:drift` (both green — Pitfall 8: `web/build/**` must be rebuilt and committed in the same commit as the `web/src` change, or `web:drift`'s SOURCE half legitimately goes RED) -> live graph/browse check scripts -> `task web:components:drift` (green, under Corepack).

**Exclusion pattern:** do not commit the CLI's incidental `web/package.json` devDependency caret-range bump unless it is a genuine, reviewed version change — `web:components:drift`'s own comparison deliberately ignores this (Package Legitimacy Audit).

---

### `docs/RELEASE.md` — dependency paragraph reword (DOCS-08, D-13)

**Analog:** same file's own "Corrected 2026-08-01" historical-correction block pattern (`:145`) for framing a value-only prose fix without inventing new heading structure.

**Current stale text to replace** (`:326-361`, read this session):
```
Read literally, `go.mod`'s `require` blocks list 134 total module
requirements. ... Of the **27 direct** requires this project deliberately
added, **14 are tree-sitter grammar modules** ...
The remaining 13 direct requires are the actual supply-chain surface worth
auditing individually: the storage engine (`cockroachdb/pebble/v2`), the MCP
server (`mark3labs/mcp-go`), ...
```
**Target shape (D-13):** drop all raw counts (27/134/14/13/107 — all stale per DOCS-08's verified `go.mod` read of 34 direct / 116 indirect), keep "wide-but-shallow" framing (one grammar module per supported language + a short named list of deliberately chosen direct requires), correct the MCP credit to `github.com/modelcontextprotocol/go-sdk` (not `mark3labs/mcp-go` — confirmed absent from `go.mod` entirely), keep `cockroachdb/pebble/v2`, and point at `go list -m all` / `go mod graph` for live numbers instead of a hardcoded count. **No drift test** — a wording change gets no test (Phase 12 ruling, carried forward).

---

### `docs/RELEASE.md` / provenance census targets (DOCS-09, D-14)

**Analog:** the file's own already-correct provenance sentence at `:101` ("subjects are the binaries" framing) — every live hit the census finds gets reworded to match this exact voice.

**Census pattern to run** (verified live this session, positive-controlled):
```bash
rg -n -U -i 'provenance[^.]{0,80}(generated|attested)[^.]{0,40}over[^.]{0,40}checksums file' \
  README.md SECURITY.md docs/*.md .github/workflows/*.yml internal/upgrade/*.go
```
Scope per D-14: `README.md`, `docs/`, `.github/workflows/`, `SECURITY.md`, `internal/upgrade` comments; **excludes** `CHANGELOG.md` and `.planning/`. A planted control phrase must be found before the census output is trusted (rule `84d1gfpywd`'s positive-control discipline — RESEARCH already ran and confirmed this control fires). **Do not reword** the two "Corrected 2026-08-01"-framed historical-correction blocks at `docs/RELEASE.md:145` and `docs/RELEASE-PROCEDURES.md:260` — those are deliberate historical record, not live claims (Pitfall 6).

---

### `SECURITY.md` — one sentence on `tool-vuln` (DOCS-10, D-15)

**Analog:** same file, existing two-scanner disjoint-scope paragraph (`:61-73`, quoted in full above under Read evidence) — extend this exact paragraph, same voice, same bullet structure.

**Exact job identity to name** (verified live, `.github/workflows/ci.yml:311-330`):
```yaml
tool-vuln:
  name: tool-vuln (VULN-01/02/03, advisory)
  ...
  - name: Tool-modfile vulnerability scan (advisory — reports, never fails the build)
    run: task vuln
```
**Target sentence shape:** name the job (`tool-vuln`) and state plainly it scans the isolated tool modfiles (`go.tool*.mod`) and reports-but-never-fails, matching the job's own comment verbatim rather than paraphrasing a new claim. **No named exposure, no drift assertion** — the existing paragraph's disjoint-scanner framing stays as written.

---

### `.planning/WINDOWS.md` (GRD-14, tool-owned)

**Analog:** same file's own `fixed`-row shape (any row with a populated `resolved_at`, e.g. row 36 at `:53`).

**Mandatory pattern — never hand-edit this file.** Close #16 and #33 exclusively via:
```
gsd-tools windows fixed 16
gsd-tools windows fixed 33
```
Verified this session (RESEARCH Pitfall 5, source-read confirmed): `markFixed`/`cmdWindowsMarkFixed` accept **only the id, zero flags, no note parameter** — verification evidence goes in the plan SUMMARY only, never as a second CLI argument. #20, #21, #34 **stay `open`**; their record-only status is stated in the plan SUMMARY and in STATE.md's Blockers prose (which already carries this exact shape — see `.planning/STATE.md:358` for the precedent of a Blockers-section bullet recording a standing, file-less item). The missing "annotate as record-only" verb is reported upstream, not worked around locally (planning-artifacts rule).

---

### `.planning/STATE.md` — Pending Todos section (DOCS-11)

**Analog:** `gsd-tools init todos`'s own `pending_todos_markdown` renderer output (bullet-list shape, confirmed this session to currently render `"None yet."` since `.planning/todos/pending/` does not exist).

**Mandatory pattern:** replace the hand-authored table (`:319-329`, `| Created | Area | Severity | Title |`) with the tool's rendered bullet output — this is filling in the tool's own canonical shape, not inventing a new one (Pitfall 4 is explicit that no table-header string exists anywhere in `gsd-core`). The `bench pinnedAt` todo (still genuinely open, no pending-todo file exists, no CLI verb creates one) is folded into the Blockers/Concerns prose instead of the Pending Todos section, following the exact bullet shape already used at `.planning/STATE.md:358` ("[Phase 08-03] user_setup NOT completed: ...").

---

### `.planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md` (DOCS-11)

**Analog:** `.planning/seeds/SEED-002-homebrew-installation-path.md`'s frontmatter, verbatim field set:
```yaml
---
id: SEED-002
status: implemented
planted: 2026-08-01
planted_during: v1.0 Phase 09 (release-please + GoReleaser)
consumed_by: v0.5.0 — macOS Distribution & Homebrew (Phases 1–4)
consumed_on: 2026-08-11
trigger_when: when relevant
scope: unknown
---
```
Copy this exact field set into SEED-001's frontmatter, filling `consumed_by`/`consumed_on` with SEED-001's own consuming milestone/date and `status: implemented` (or the correct current status per SEED-001's actual disposition) — values only, same shape, no new fields invented.

---

### `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` (NEW, GRD-09, D-02)

**Analog:** `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` — copy this file's section shape exactly: `## Pre-mutation cleanliness gate` preamble, then one `## Family (a) — <requirement>` block per guard proven, with `**Pre-mutation gate:**` / `**Mutation applied:**` / `**Observed failure (pasted, ...)**` / `**Revert:**` / `**Byte-clean proof:**` subsections.

**GRD-09's own family entry — content already fully captured live this session (RESEARCH Pattern 1 / Code Examples), paste directly:**
```
## Family (a) — GRD-09: web:drift OUTPUT-half mismatch replay against 98cd41dd

**Test/guard:** `task web:drift` (Taskfile.yml), replayed against commit 98cd41dd's exact tree via `git worktree add --detach <scratch> 98cd41dd` — no code mutation, per Pattern 1 (the RED condition is the historical git tree's own shape).

**Pre-mutation gate:** `git status --porcelain` in the fresh worktree — empty (genuinely clean checkout).

**Mutation applied:** None — see Pattern 1: the defect is the commit's own historical shape, not an injected mutation.

**Observed failure:**
web:drift: hashed 108 source files
web:drift: manifested 22 output files
web:drift: source half MATCH (108 files, 1e0bff2fb48d58bd943e8bb842dff6d81f3c608105ed3d2b31aa6585a6b8f5c3)
::error::web:drift: OUTPUT-half mismatch — the committed web/build/ bytes are NOT the ones `task web:build` produced (marker: 32 files / 8658e6fe64bb02282e008557d39baba453d3e2765d6020071c8dc58d7ca9c432; recomputed: 22 files / a76add11cadd4bd4cb0322de2a0e62246187397236860cd0b9eb3e1e57bf39e2)
task: Failed to run task "web:drift": exit status 1   # exit code 201

**Revert:** `git worktree remove --force <scratch>` — main repo untouched throughout.

**Byte-clean proof:** N/A — no tracked file in the main repo was ever touched (scratch worktree only).
```
This closes WINDOWS #29 via `gsd-tools windows fixed 29`, citing this exact transcript as the cause: CI (clean-checkout `find`) already fails on this incident shape; the ledger's suggested `find`-vs-`git ls-files` paired assertion is declined per D-01.

## Shared Patterns

### Every Taskfile-backed CI step is a bare one-line `run: task <target>`, never re-derived logic
**Source:** `.github/workflows/ci.yml:131-219` (every existing gate step in the `test` job)
**Apply to:** the two new `check:*` steps and any future Taskfile-target wiring
```yaml
      - name: <Guard name> (<REQ-ID>)
        run: task <target>
```
The guard's positive-controlled assertion lives entirely inside the Taskfile target (or the script it calls); the CI step is invocation only. Phase 7's own D-07 precedent (cited in RESEARCH Anti-Patterns) forbids a CI-only shape test that mirrors a script's own assertion.

### Hard-fail, never skip, on any external-dependency failure
**Source:** GRD-07's declined "skip-clean offline" clause (`.planning/milestones/v0.13.0-REQUIREMENTS.md:86-87`); D-06's explicit restatement
**Apply to:** the ruleset-drift step (GRD-12) — any non-2xx, empty body, or unparseable JSON is a **named** `exit 1` via `::error::`, never `continue-on-error: true` and never a conditional early-return that reads as PASS.

### Positive-count-before-assertion, never a silent zero-pass
**Source:** rule `84d1gfpywd`, already enforced throughout this file (`TestRequiredCheckNamesPreserved_ZeroJobsIsError`, `check:gonum`'s own SBOM-count precondition, the census's positive control)
**Apply to:** the `requiredCheckNames` data-file loader (empty-file `t.Fatalf`), the DOCS-09 census (planted control phrase required before trusting a zero-hit result), the ruleset-drift step (`echo "... live has $(wc -l) contexts, fixture has $(wc -l) contexts"` printed **before** the equality assertion, matching every other guard in this Taskfile/CI surface that "prints the count it inspected before asserting on it," per RESEARCH's Established Patterns).

### Tool-owned generated files are edited only through their owning tool's verb
**Source:** `/Users/sean/.claude/rules/planning-artifacts.md`; RESEARCH Pitfalls 4-5
**Apply to:** `.planning/WINDOWS.md` (`gsd-tools windows fixed <id>` only), `.planning/STATE.md`'s Pending Todos section (`gsd-tools init todos`'s rendered output only) — never a hand-authored table row or an invented "record-only" status/heading in either file. Where the tool has no verb for a needed shape (annotate-as-record-only, file a new pending-todo from a CLI command), the gap is reported upstream and the fact recorded in plan-owned prose (SUMMARY, STATE.md Blockers), never papered over inside the tool-owned file.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `.github/required-status-checks.txt` | config (data file) | batch | No existing newline-delimited fixture in this repo is shared between a Go test and a shell/CI step today — genuinely new shape, format explicitly left to Claude's discretion by CONTEXT.md D-07. Use the plain-list format shown above (one context per line, no comments, defensive blank-line handling on both readers) since it is the simplest shape both a `sort`+`jq` shell pipeline and a Go `os.ReadFile`+`strings.Split` loader can parse identically. |
| `.github/workflows/ci.yml` ruleset-drift step | CI step (external API read) | request-response | No existing `ci.yml` step calls an external, repo-independent read-only service; nearest sibling (`components-drift.yml`) is deliberately *not* in `ci.yml` at all (D-06's whole point is that this one, unlike that one, IS a required check). Build fresh from the RESEARCH-verified transcript (already live-tested) rather than searching further for an analog that structurally cannot exist yet. |

## Metadata

**Analog search scope:** `.github/workflows/*.yml`, `internal/upgrade/taskfile_shape_test.go`, `Taskfile.yml` (`web:components:drift`, `check:gonum`, `check:no-force-layout` targets), `web/src/lib/components/ui/**`, `docs/RELEASE.md`, `SECURITY.md`, `.planning/WINDOWS.md`, `.planning/STATE.md`, `.planning/seeds/*.md`, `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md`
**Files scanned:** 11 target files + 6 analog source files read in full or by targeted range this session (no re-reads of any already-loaded range)
**Pattern extraction date:** 2026-09-15
**Tracked-source gate:** `git ls-files` confirmed every analog path cited above is tracked (no `.gsd/capabilities` or other gitignored mirror involved anywhere in this phase)
