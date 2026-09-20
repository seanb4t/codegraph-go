# Phase 2: Guards, CI Wiring & Docs Burn-down - Research

**Researched:** 2026-09-15
**Domain:** CI/CD workflow wiring (GitHub Actions + Taskfile), guard-vacuity proof methodology, GitHub branch-protection ruleset drift detection, documentation-claim census, tool-owned planning-artifact bookkeeping
**Confidence:** HIGH — every load-bearing claim below was verified live against this repository, its git history, its GitHub Actions run history, or the GitHub REST API this session; nothing rests on training-data recall of shadcn-svelte/GoReleaser/GitHub Actions internals

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**web:drift and the 98cd41dd incident (GRD-09)**
- **D-01:** The `web:drift` gate is **not changed**. The maintainer reset the requirement after the discussion established the facts: CI runs `web:drift` on a clean checkout, where `find web/build` *is* the git tree, so `98cd41dd`'s shape (9 brand-new chunk files on disk, never staged) hashes to a different output digest than its committed `.build-manifest` and **fails in CI**. The local `find` enumeration is deliberate — `//go:embed all:build` ships whatever is on disk, tracked or not (Taskfile comment on `web_output_files`). A `find`-vs-`git ls-files` set assertion would test whether the developer staged what they built — git hygiene, not the gate — and buys only one saved CI round trip. Reversible.
- **D-02:** GRD-09's deliverable is the **proof, not a fix**: replay `98cd41dd`'s tree in a scratch clone, run `task web:drift`, record the non-zero exit and the digest mismatch in `02-MUTATION-LOG.md` (the `07-MUTATION-LOG.md` shape). Close WINDOWS #29 via `gsd-tools windows fixed 29` with that evidence as the cause: the CI path was never vacuous; the ledger's suggested fix is declined.
- **D-03:** `REQUIREMENTS.md`'s GRD-09 text and ROADMAP success criterion 1 are **reworded**: "`web:drift` is demonstrated RED against commit `98cd41dd`'s exact incident shape on a clean checkout, closing WINDOWS #29 with the recorded cause; the local `find` enumeration is retained by design." No "paired assertion" language survives.

**CI wiring (GRD-11)**
- **D-04:** `check:gonum` and `check:no-force-layout` run as **steps inside the existing `test` job** in `ci.yml`, next to the pnpm-audit and `web:drift` steps. `test` is already a required context, so both block merges the moment they land — no ruleset edit, no fixture change. Each step calls the Taskfile target; the targets' own positive controls are the guard.

**Ruleset drift (GRD-12)**
- **D-05:** The comparison is **exact set equality**: fixture context set == live `protect-main` ruleset's `required_status_checks[].context` set, both directions. A context required live but absent from the fixture under-asserts; a context in the fixture but not required live asserts a requirement that does not exist. Both are drift.
- **D-06:** The comparison runs as a **CI-only step** (in the `test` job) — no local Taskfile target, no Go test. Fetches `GET /repos/seanb4t/codegraph-go/rulesets/20157557` (Metadata:read; default `GITHUB_TOKEN` suffices, unauthenticated works on this public repo), extracts required contexts, prints both counts, exits non-zero on any diff. **An API failure (non-2xx, empty body, unparseable JSON) is a hard failure naming the cause — never a skip.** Reversible.
- **D-07:** The context list moves out of `taskfile_shape_test.go`'s `requiredCheckNames` Go literal into a **data file read by both** the Go test (keeps its job-name assertion) and the CI step — one list, no bash parsing of Go source, no duplication. Location/format at Claude's discretion (newline-delimited text file under `.github/` is the obvious shape); the Go test must fail loudly if the file is missing or empty.
- **D-08:** The maintainer **adds two contexts to the live ruleset during this phase**: `goreleaser check (config validation, DIST-01)` and `tmux e2e (real-pty harness, TTY-01..TTY-07)`. The fixture therefore grows to **8**. This is a `user_setup` step with a `blocking-human` precondition on the plan that lands the drift step: the executor hands the maintainer the exact context strings and a reviewable `gh api` PUT/PATCH body, then verifies the live set before the fixture is edited. Costly to reverse.

**Vendored component drift (GRD-10)**
- **D-09:** The isolation is **already done by evidence**: the `components-drift.yml` scheduled run of 2026-09-14 (run 34820878640) ran under Corepack-resolved pnpm v11.23.0 and reported drift. The local-toolchain hypothesis (pnpm 12 vs pinned 11.23.0) in WINDOWS #31 is **refuted**; the registry moved under a pinned CLI version. A fresh `workflow_dispatch` run is still triggered to capture the current diff for the re-vendor.
- **D-10:** The decision rule for vendored components that drift is **re-vendor with a reviewed diff, never waive**: regenerate from the registry, review the diff (visual and API changes called out per file), run every gate, commit as a human-approved re-vendor in the lineage of `205da685`.
- **D-11:** Re-vendor **all eight families to one registry snapshot** — one upstream point in time for every vendored component, one reviewed diff, one commit; families that regenerate byte-identically show no change. Never a tree with three components at "new registry" and five at "old".
- **D-12:** The **executor reviews and commits** when all gates are green (`pnpm check` exit 0 with zero errors; `cd web && pnpm vitest run` exit 0 with zero "unhandled errors" lines; `task web:build && task web:drift`; the live graph/browse check scripts; `task web:components:drift` green under Corepack). No mid-plan checkpoint; the per-file diff summary (with any visual change named) goes in the SUMMARY and the maintainer reviews at end of phase.

**Doc claims (DOCS-08, DOCS-09, DOCS-10)**
- **D-13:** `docs/RELEASE.md`'s dependency paragraph **drops the raw counts** and keeps the shape: wide-but-shallow, crediting `modelcontextprotocol/go-sdk` (not `mark3labs/mcp-go`) and `cockroachdb/pebble/v2`, pointing at `go list -m all` / `go mod graph` for live numbers. **No drift test** (a wording change gets no test).
- **D-14:** DOCS-09 is a **whole-repo, positive-controlled census**: word-boundary, multiline `rg` for checksums-file wording inside provenance/SLSA/attest sentences across `README.md`, `docs/`, `.github/workflows/`, `SECURITY.md` and `internal/upgrade` comments — excluding `CHANGELOG.md` and `.planning/`. A planted phrase must be found by the census before it is trusted. Every hit is reworded to: provenance is attested over each platform binary; the checksums file is the *transport* for the subject list, not a subject. GH #14 is closed citing the census output; its stale line numbers are not relied on.
- **D-15:** Root `SECURITY.md` gains **one sentence** for DOCS-10: govulncheck blocks merges on the main module graph; the isolated tool modfiles (`go.tool*.mod`) are scanned by the advisory `tool-vuln` job, which reports and never fails the build. **No named exposure**, no drift assertion. The existing two-scanner disjoint-scope paragraph and pnpm audit's blocking description stay as written.

### Claude's Discretion
- **GRD-14 / DOCS-11 bookkeeping.** `gsd-tools windows` offers `status | append | waive | fixed` — there is no "annotate as record-only" verb and no such status. Therefore: #16 and #33 are closed with `gsd-tools windows fixed 16` / `fixed 33` and the verification evidence is recorded in the plan SUMMARY (and, if the verb accepts a note, in the row); #20, #21 and #34 **stay `open`**, their record-only status is stated in the SUMMARY and in STATE.md's Blockers list, and the missing annotate verb is **reported upstream**. STATE.md's Pending Todos table (17 rows) versus an empty `.planning/todos/pending/` is reconciled through `gsd-tools todo` / `state` verbs where one exists; where none does, the SEED-002 precedent applies (frontmatter values only). SEED-001's frontmatter records its consumption by v0.12.0 in the same fields SEED-002 uses.
- Data-file location/format for the shared required-contexts list (D-07); mutation-log file name (`02-MUTATION-LOG.md`, following `07-MUTATION-LOG.md`).
- Exact wording of the reworded requirement/criterion (D-03), the RELEASE.md paragraph (D-13), the census pattern (D-14) and the SECURITY.md sentence (D-15) — subject to the bars above.
- Whether the ruleset PUT is applied by the maintainer through the UI or through the executor-prepared `gh api` call (D-08) — the maintainer's call at the checkpoint.

### Deferred Ideas (OUT OF SCOPE)
- A local pre-push honesty check for `web/build` (`git status --porcelain web/build` must be empty before `web:drift` reports PASS) — declined for this phase as git hygiene rather than a gate property.
- Pinning the shadcn-svelte registry snapshot (so the drift monitor compares against a recorded upstream point rather than "latest") — a change to what the monitor asks, not in scope.
- Gating the three `window.__codegraphFileGraph*` debug globals behind a build flag — not this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| GRD-09 | `web:drift` demonstrated RED against `98cd41dd`'s exact incident shape on a clean checkout, closing WINDOWS #29 (reworded per D-03: proof, not a code change) | Live replay reproduced this session (Pattern 1, Code Examples) — exact digest/count mismatch captured for `02-MUTATION-LOG.md` |
| GRD-10 | Vendored component drift isolated (already done, D-09) and all 8 families re-vendored to one registry snapshot (D-10/D-11), executor-reviewed and committed (D-12) | Real current drift fetched from CI run 34820878640 (24/50 files, 7/8 families) — see Code Examples; regeneration mechanics documented in Architecture Patterns |
| GRD-11 | `check:gonum` / `check:no-force-layout` run in `ci.yml` on every PR, as steps inside the existing `test` job (D-04) | Exact `test`-job step ordering and preconditions verified (Pitfalls 2-3); missing `syft` install step identified as the one net-new CI dependency |
| GRD-12 | `requiredCheckNames` compared against the live `protect-main` ruleset, fails on divergence (D-05/D-06), shared data file (D-07), maintainer adds 2 contexts (D-08) | Live ruleset fetched (6 contexts today, unauthenticated); exact existing-drift direction found (fixture already asserts `goreleaser check`, which is not yet live); CI step design in Pattern 3 |
| GRD-14 | WINDOWS #16/#33 closed via `gsd-tools windows fixed`; #20/#21/#34 stay open, record-only | CLI verb source read and confirmed — no note parameter, no annotate verb (Pitfall 5); do not invoke during planning, only execution |
| DOCS-08 | `docs/RELEASE.md` dependency paragraph reshaped, drops raw counts, credits `modelcontextprotocol/go-sdk` (D-13) | Actual current `go.mod` counts extracted (34 direct / 116 indirect, not the documented 27/107) — Code Examples |
| DOCS-09 | SLSA provenance described as attested over binaries everywhere a census finds the claim (D-14); GH #14 closed citing the census | Whole-repo positive-controlled census run this session — zero live incorrect claims found; GH #14's three original sites already fixed (Pitfall 6) |
| DOCS-10 | Root `SECURITY.md` gains one sentence on govulncheck vs. `tool-vuln` advisory scope (D-15) | Exact `tool-vuln` job name/wording verified (Code Examples) |
| DOCS-11 | STATE.md Pending Todos reconciled with `.planning/todos/`; SEED-001 frontmatter records consumption by v0.12.0 | `.planning/todos/pending/` confirmed empty (contradicts STATE.md's own claim); tool's bullet-render shape vs. hand-authored table mismatch discovered (Pitfall 4); SEED-002's exact frontmatter fields identified for SEED-001 |
</phase_requirements>

## Summary

This phase touches no application code and installs no new dependency — it is entirely CI-wiring, Taskfile-target reuse, a documentation census, and tool-owned bookkeeping. The single largest research finding is that **the phase-wide test ("what are we testing, and why?") has already collapsed most of the apparent work into verification-only tasks**: GRD-09's fix already exists and is proven RED against the exact historical incident by simply checking out `98cd41dd` (verified this session — see Code Examples); GRD-12's live ruleset already diverges from the local fixture in the *opposite* direction CONTEXT assumes at first glance (the fixture already asserts a context — `goreleaser check`  — that is NOT required live); and DOCS-09's originally-cited defect sites have *already been fixed* by the D-09/D-10 provenance migration that shipped since GH #14 was filed. The remaining real work is: (1) two new `test`-job CI steps for `check:gonum`/`check:no-force-layout` (needs a new `syft` install step — not currently in `ci.yml`'s `test` job); (2) one new CI-only ruleset-comparison step plus a shared data file; (3) re-vendoring all eight shadcn-svelte families against a materially larger diff than CONTEXT's summary states (24 of 50 files differ today, not 3); (4) three doc-paragraph rewrites; and (5) WINDOWS/STATE.md/seed bookkeeping through tool verbs that this session discovered have narrower shapes than CONTEXT assumed in one case (STATE.md's "Pending Todos table" has no tool-native table renderer — the tool only emits a bullet list).

**Primary recommendation:** Do the RED replay and the ruleset/census probes as read-only verification first (all reproduced safely this session); wire the two `check:*` steps and the ruleset-drift step into `ci.yml`'s `test` job in the exact slot next to the existing JS gates; re-vendor all eight component families in one `pnpm dlx shadcn-svelte@1.5.1 add <8 families> -y -o` pass against the real `web/` tree (not just the drift target's scratch copy); rewrite the three doc paragraphs to the exact current numbers/wording found below; and close only the two WINDOWS rows that already have terminable evidence via `gsd-tools windows fixed <id>` (a verb this session confirmed by reading its source takes **no note parameter** — evidence goes in the plan SUMMARY only).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `web:drift` incident replay/proof | CI / Build tooling (Taskfile) | — | Pure shell guard over the git-tree + filesystem; no application tier involved |
| `check:gonum` / `check:no-force-layout` CI wiring | CI / Build tooling (GitHub Actions `test` job) | Go module graph, JS static scan | Pre-existing local guards; the only new work is invocation from CI, not guard logic |
| Ruleset-drift comparison | CI / Build tooling (GitHub Actions, one-off step) + GitHub REST API (external service boundary) | `internal/upgrade` (Go test, shared fixture) | Compares two independently-owned sources of truth (repo-local fixture vs. GitHub's ruleset API); correctly lives outside the local Taskfile per D-06 |
| Vendored component re-vendor | Frontend build tooling (`web/`, shadcn-svelte CLI) | — | Pure source regeneration against an external registry; no runtime/server code touched |
| Doc-paragraph rewrites | Documentation | — | No code or CI surface; text-only, explicitly no drift test per D-13/D-15 |
| WINDOWS/STATE.md/seed bookkeeping | Planning tooling (`gsd-tools`) | — | Tool-owned generated files; values only, via tool verbs — see Common Pitfalls |

## Package Legitimacy Audit

**No new external packages (npm, PyPI, crates, or Go module) are installed by this phase.** `check:gonum` and `check:no-force-layout` are pre-existing Taskfile targets (already present, already passing locally per STATE.md's Phase-11 blockers note); wiring them into `ci.yml` invokes `task <target>`, adding no new `require` line to `go.mod` or `web/package.json`. The vendored-component re-vendor (GRD-10) regenerates existing `.svelte` source files in place at the same pinned CLI version (`shadcn-svelte@1.5.1`) already used by the prior vendoring commits — no new npm dependency is added (per `web:components:drift`'s own doc comment, the CLI's dependency-install step bumps a `package.json` devDependency caret range as a side effect, which the drift comparison deliberately ignores; the re-vendor plan should apply the same exclusion and not commit that incidental `package.json` churn unless it is a genuine, reviewed version bump).

One net-new CI dependency is required for GRD-11's wiring: **`syft`** is not installed on any GitHub-hosted runner by default and is not currently present anywhere in `ci.yml`'s `test` job (verified: `rg -n syft .github/workflows/*.yml` finds it only in `release.yml` and `linux-cross-canary.yml`, both of which install it via `anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0.24.0` before use). `check:gonum`'s SBOM half (`Taskfile.yml:1645+`) shells out to `syft` directly and will fail the `test` job with "command not found" unless the identical install step is added before the `check:gonum` step.

| Package/Action | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---|---|---|---|---|---|---|
| `anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610` | GitHub Actions | Already in use elsewhere in this repo (`release.yml`, `linux-cross-canary.yml`), same pinned SHA | N/A (Action, not a package registry entry) | github.com/anchore/sbom-action | `OK` — already vetted and pinned identically elsewhere in this repo | Approved — reuse the existing pin, do not re-derive a new SHA |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────┐
                         │   PR opened / push to main   │
                         └───────────────┬──────────────┘
                                         │
                                         ▼
                         ┌─────────────────────────────┐
                         │   ci.yml  job: test          │
                         │  (namespace-profile runner)   │
                         │                               │
                         │  checkout → go setup → task   │
                         │  build/vet/lint/test:unit …   │
                         │                               │
                         │  ── existing JS gate chain ── │
                         │  Set up Node → web:deps →     │
                         │  web:test → web:deps:strict → │
                         │  web:audit → web:build:verify │
                         │  → web:drift → proto:drift    │
                         │        ▲                      │
                         │        │  NEW: insert here     │
                         │        │  (D-04)                │
                         │  [Install syft]  ← NEW          │
                         │  [check:gonum]   ← NEW          │
                         │  [check:no-force-layout] ← NEW  │
                         │        │                        │
                         │        ▼                        │
                         │  [ruleset-drift step] ← NEW,     │
                         │  CI-only (D-06), reads shared     │
                         │  data file, calls GitHub REST API │
                         └───────────────┬────────────────┘
                                         │ pass/fail = required check
                                         ▼
                    ┌─────────────────────────────────────┐
                    │ GitHub ruleset 20157557 (protect-main)│
                    │  compared against, never written to   │
                    │  by this step (read-only GET)          │
                    └─────────────────────────────────────┘

  Separate, schedule/dispatch-only path (never a merge gate):
                    ┌─────────────────────────────┐
                    │ components-drift.yml         │
                    │  (Mon 08:00 UTC / dispatch)   │
                    │  → task web:components:drift  │
                    │  → regenerate 8 families into  │
                    │    scratch tree, cmp -s against │
                    │    web/src/lib/components/ui/**  │
                    └───────────────┬──────────────┘
                                    │ RED today (24/50 files differ)
                                    ▼
                    ┌─────────────────────────────┐
                    │ Executor re-vendors ALL 8      │
                    │ families in web/ directly       │
                    │ (pnpm dlx shadcn-svelte add ...) │
                    │ reviews diff, runs all gates,     │
                    │ commits (D-10..D-12)               │
                    └─────────────────────────────┘
```

### Recommended Project Structure (files touched, no new directories)
```
.github/
├── workflows/ci.yml              # test job: 3 new steps (syft install, check:gonum, check:no-force-layout, ruleset-drift)
├── required-status-checks.txt    # NEW — shared data file (D-07), newline-delimited context list
internal/upgrade/
├── taskfile_shape_test.go        # requiredCheckNames var replaced by a loader of the data file above
web/src/lib/components/ui/
├── {button,command,dialog,input,input-group,table,tabs,textarea}/  # re-vendored in place
docs/RELEASE.md                   # dependency paragraph reworded (D-13); provenance wording checked by census (D-14)
SECURITY.md                       # one sentence added (D-15)
.planning/WINDOWS.md              # #16, #33 -> fixed via gsd-tools verb; #20/#21/#34 stay open, annotated in SUMMARY/STATE.md prose
.planning/STATE.md                # Pending Todos section — see Common Pitfalls (tool-shape mismatch)
.planning/seeds/SEED-001-*.md     # frontmatter: status/consumed_by/consumed_on added (SEED-002 shape)
```

### Pattern 1: Replaying a historical incident via `git worktree` (GRD-09's proof mechanic)
**What:** Check out the exact pre-fix commit into an isolated worktree and run the real guard against it — no code mutation needed because the RED condition is the git tree's own historical shape.
**When to use:** When a guard's vacuity claim concerns *whether it fires against a historical incident*, not a hypothetical injected defect (contrast with the `07-MUTATION-LOG.md` families, which mutate a currently-correct file).
**Example (reproduced this session, exit code 201, fully clean afterward):**
```bash
git worktree add --detach /path/to/scratch 98cd41dd
cd /path/to/scratch
git status --porcelain   # empty — confirms a genuinely clean checkout, not the
                          # developer's original working directory that had
                          # extra untracked files present
task web:drift
# web:drift: hashed 108 source files
# web:drift: manifested 22 output files
# web:drift: source half MATCH (108 files, 1e0bff2f...)
# ::error::web:drift: OUTPUT-half mismatch — the committed web/build/ bytes are
#   NOT the ones `task web:build` produced ... (marker: 32 files / 8658e6fe...;
#   recomputed: 22 files / a76add11...)
# task: Failed to run task "web:drift": exit status 1
cd /Volumes/Code/github.com/seanb4t/codegraph-go
git worktree remove --force /path/to/scratch   # cleanup, main repo untouched
```
This is the exact non-zero output the mutation log should paste for D-02: **output-files 22 vs manifest 32, digest `a76add11cadd4bd4cb0322de2a0e62246187397236860cd0b9eb3e1e57bf39e2` vs `8658e6fe64bb02282e008557d39baba453d3e2765d6020071c8dc58d7ca9c432`**, SOURCE half matches (108 files both sides — confirming the defect is isolated to the output half, exactly as WINDOWS #29 describes). `[VERIFIED: live `task web:drift` run this session against `git worktree add --detach <scratch> 98cd41dd`]`.

### Pattern 2: CI-only network-dependent comparison, never a local Task target (D-06's shape)
**What:** A step that calls an external, repo-independent service (here, the GitHub REST API for a live ruleset) belongs directly in the workflow YAML, not behind a Taskfile target that a contributor might invoke offline.
**When to use:** Any comparison whose "ground truth" side lives outside the git tree (contrast with `web:components:drift`, which is *also* network-dependent but is deliberately kept in a Taskfile target because it is schedule/dispatch-only, never a required check — see `components-drift.yml`'s own header comment on why it avoids `ci.yml` entirely).
**Example (verified live this session, unauthenticated, 200 OK, on this public repo):**
```bash
curl -s "https://api.github.com/repos/seanb4t/codegraph-go/rulesets/20157557"
# -> 200, full ruleset JSON, no Authorization header needed on this public repo
```
The live `required_status_checks[].context` array today is exactly:
```json
["test", "actionlint (workflow static analysis)", "perf regression gate (PERF-02, INDX-06)",
 "pr-title", "reproducibility (double-build hash-diff, DIST-04)", "govulncheck (DIST-03, blocking)"]
```
`[VERIFIED: gh api repos/seanb4t/codegraph-go/rulesets/20157557, run this session]` — **6 contexts today**, confirming CONTEXT's "current live set is 6" claim. Note this does **not** include `goreleaser check (config validation, DIST-01)`, even though that name already sits in `internal/upgrade/taskfile_shape_test.go`'s `requiredCheckNames` fixture — i.e., **the fixture and the live ruleset are already drifted today**, in the direction "fixture asserts a requirement that isn't live," which is exactly the drift shape D-05 says must also fail (not just the reverse direction). GRD-12's step will be RED from the moment it is written, until D-08's ruleset PUT adds both `goreleaser check` and `tmux e2e`, bringing live to 8 and matching the fixture (7 existing + `tmux e2e`).

### Pattern 3: Shared required-checks data file consumed by both a Go test and a CI step (D-07)
**What:** Extract `requiredCheckNames` from a Go `[]string` literal (`internal/upgrade/taskfile_shape_test.go:77-85`) into `.github/required-status-checks.txt` (recommended path/format — no existing precedent for this exact shape in the repo, so this is Claude's-discretion territory, not tool-owned), one context string per line, no comments (keep the parser trivial on both the Go and shell side). The Go test's existing assertion (`TestRequiredCheckNamesPreserved` — **not** `TestRequiredCheckJobsExist`, see Common Pitfalls) reads the file via `os.ReadFile` instead of a package-level `var`; a missing or empty file must `t.Fatalf`, not silently produce zero required names (which would make the test vacuously pass — rule `84d1gfpywd`).
**Example CI step shape:**
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
No `GITHUB_TOKEN`/`GH_TOKEN` header is required — verified live, unauthenticated, 200 — but adding `-H "Authorization: Bearer ${{ secrets.GITHUB_TOKEN }}"` is harmless and future-proofs against the repo going private; either is acceptable, note it as Claude's discretion since D-06 explicitly says "the default GITHUB_TOKEN suffices, unauthenticated works." Do **not** use `gh api` for this step unless `GH_TOKEN`/`GITHUB_TOKEN` is exported into the step's env — `gh` (unlike a bare `curl`) refuses to run unauthenticated even against a public endpoint that itself needs no auth.

### Anti-Patterns to Avoid
- **Re-deriving `check:gonum`/`check:no-force-layout`'s guard logic in the CI step.** Per D-04, CI only calls `task check:gonum` / `task check:no-force-layout` — the positive-controlled assertions already live in the Taskfile target. Do not add a second, CI-only assertion of the same property (Phase 7's own D-07 precedent: no shape test mirroring a script that already asserts the property).
- **Skipping the ruleset-drift step on API failure.** GRD-07 was declined at v0.13.0 specifically for a "skip-clean offline" clause. Every non-2xx, empty-body, or unparseable-JSON outcome is a hard failure with a named cause (D-06).
- **Trusting CONTEXT.md's "three drifting files" characterization of GRD-10 without re-checking.** The actual, current (2026-09-14) drift is **24 of 50 files across 7 of 8 families** (see Code Examples) — CONTEXT's own text undercounts it; this doesn't change the already-locked decision (re-vendor all 8 regardless — D-11), but the plan/SUMMARY should record the real scope for effort estimation, not the stale 3-file figure.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Comparing a fixture set against a live GitHub ruleset | A custom diffing library or Go program | `jq` + `sort` + shell set-comparison (see Pattern 3) | `jq` and POSIX sort are already assumed-available on GitHub-hosted runners (`post-release-verify.yml` already defensively checks for `jq`); a hand-rolled comparator is unnecessary surface area for a single-purpose CI step |
| Re-vendoring shadcn-svelte components | Hand-editing the eight `.svelte` files to match the registry | `pnpm dlx shadcn-svelte@1.5.1 add <8 families> -y -o` directly against `web/` | This is literally what `web:components:drift`'s own regeneration step does (Taskfile.yml:1487) — the CLI is the single source of truth for "correct" vendored source; hand-editing risks introducing exactly the kind of undetectable drift the guard exists to catch |
| Detecting the web/build staleness incident | A new guard mechanism | The existing `web:drift` two-half hash comparison, unmodified (D-01) | Already proven to fire correctly on a clean checkout; the "vacuous" appearance was a misreading of a developer's dirty local working tree, not a real gate defect |

**Key insight:** Every "guard that cannot fire" claim in this phase turned out, on replay, to be either (a) a guard that fires correctly and was misjudged from a contaminated local reproduction (GRD-09), or (b) a guard/fixture pair that is *already* drifted in a direction nobody had checked (GRD-12's `goreleaser check` context). The phase's actual leverage is in the replay/census work, not in writing new detection logic.

## Common Pitfalls

### Pitfall 1: `TestRequiredCheckJobsExist` does not exist — the real test name is `TestRequiredCheckNamesPreserved`
**What goes wrong:** CONTEXT.md's canonical-refs section and the phase notes refer to "`TestRequiredCheckJobsExist` keeps its job-name assertion." No such function exists in `internal/upgrade/taskfile_shape_test.go`.
**Why it happens:** Likely a paraphrase drift between the discuss-phase conversation and the actual symbol name.
**How to avoid:** The actual test is `TestRequiredCheckNamesPreserved` (and its edge-case sibling `TestRequiredCheckNamesPreserved_ZeroJobsIsError`), at `internal/upgrade/taskfile_shape_test.go:749` and `:793`. D-07's refactor target is this function's `requiredCheckNames` package var (`:77-85`), not a function named `TestRequiredCheckJobsExist`.
**Warning signs:** `grep`/`rg` for the literal name in the plan returns nothing before editing — verify against the file directly (`[VERIFIED: internal/upgrade/taskfile_shape_test.go:749,793]`).

### Pitfall 2: `syft` is not on the `test` job's runner — `check:gonum` will fail with "command not found" if wired without an install step
**What goes wrong:** `check:gonum`'s SBOM half (`Taskfile.yml:1645+`) has a hard `preconditions:` check for `syft` and will hard-fail the precondition (not silently skip) if it's missing.
**Why it happens:** `syft` is currently installed only in `release.yml` and `linux-cross-canary.yml`, both via the pinned `anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610` Action — this Action does not appear anywhere in `ci.yml` today.
**How to avoid:** Add the identical `- name: Install syft` / `uses: anchore/sbom-action/download-syft@e22c389904149dbc22b58101806040fa8d37a610 # v0.24.0` step to `ci.yml`'s `test` job, placed before the new `check:gonum` step and after `Set up Node` (check:gonum also needs `node` on PATH for its SBOM-JSON-parsing half).
**Warning signs:** First CI run after wiring fails at `check:gonum`'s precondition line with "syft not found."

### Pitfall 3: `check:gonum` and `check:no-force-layout` both require `node` on PATH, which is only set up mid-job
**What goes wrong:** If either step is placed before `- name: Set up Node` (`ci.yml:131`), `check:gonum`'s `command -v node` precondition and `check:no-force-layout`'s `command -v node` precondition both fail.
**Why it happens:** `check:no-force-layout` (`web/scripts/check-no-force-layout.mjs`) uses only Node builtins (`node:fs`, `node:os`, `node:path`, `node:url` — verified, no npm import) so it does **not** need `task web:deps`/`pnpm install` to have run first, only `node` itself — meaning it can be placed anywhere after `Set up Node`, it does not need to wait for the full JS install chain.
**How to avoid:** Insert both new steps after `- name: Set up Node` (`ci.yml:131`), matching D-04's "next to the pnpm-audit and web:drift steps" guidance; exact order between the two new steps and the existing JS gate chain is Claude's discretion, but neither can precede Node setup.
**Warning signs:** `command not found: node` in the precondition failure message.

### Pitfall 4: STATE.md's "Pending Todos table" has no tool-native table renderer — the tool only emits a bullet list
**What goes wrong:** DOCS-11 asks that "STATE.md's Pending Todos table matches `.planning/todos/`." The current STATE.md section is a hand-authored markdown **table** (`| Created | Area | Severity | Title |`). `gsd-core`'s own renderer for this section, `renderPendingTodosMarkdown` (`init.cjs:2024`, consumed as the `pending_todos_markdown` field from `gsd-tools init todos`, verified live this session), emits **one bullet per todo**, not a table row — "One bullet per todo, each capped at PENDING_TODO_BULLET_MAX_CHARS." No table-header string (`| Created | Area | Severity | Title |`) exists anywhere in `gsd-core`.
**Why it happens:** The table shape predates this bullet-rendering convention (or was hand-authored contrary to it) and was never reconciled.
**How to avoid:** Per the planning-artifacts rule, adopting the tool's own bullet shape (replacing the table with the rendered bullet list) is filling in the tool's canonical structure, not inventing one — this is the correct fix, not a gap. Live evidence this session (`gsd-tools init todos`, read-only): `.planning/todos/pending/` **does not exist at all** (`pending_dir_exists: false`, `todo_count: 0`, `pending_todos_markdown: "None yet."`) — directly contradicting STATE.md's own claim that it "holds only 2 files (brew-trust, graphstore archtest)." Both of those files are actually already in `.planning/todos/completed/` (`[VERIFIED: ls .planning/todos/completed/, this session]`). The tool-correct STATE.md Pending Todos body, right now, is **"None yet."** — all 4 rows currently in STATE.md's hand-written table are stale:
  - "Wire oracle toolslist-repeat" — **already resolved**, completed file exists (`.planning/todos/completed/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`) and STATE.md's own Phase-03 decision log records the root cause fixed via R2.
  - "Add golangci-lint" — **already resolved**, completed file exists, and Phase 3's decisions confirm golangci-lint was added in an isolated tool modfile.
  - "CR-01 pendingWriter" — **already resolved** as v0.12.0 Phase 1's `FIX-01` (shipped, per STATE.md's own decisions log and `ROADMAP.md`).
  - "`tools/bench/runner/main.go:482` `pinnedAt()` validates by `git rev-parse HEAD` alone" — **still genuinely open** (`[VERIFIED: tools/bench/runner/main.go:435-441, read this session — unchanged, still `git -C dir rev-parse HEAD` alone]`), but **has no corresponding file** in `.planning/todos/pending/` (never filed as one) and no `gsd-tools` verb exists to create a new pending-todo file from a CLI command (`rg` for a "todos/pending" write site across `gsd-core`'s `commands.cjs` and its workflow markdown found none). This is a genuine gap: either file it properly first (e.g., via the `/gsd-capture`-style `add-todo` workflow, which is agent-authored, not a CLI verb) so the bullet-render picks it up, or record it as an explicit STATE.md Blockers-section prose line instead of inventing a table row — never re-invent the table shape.
**Warning signs:** Any plan that proposes hand-editing STATE.md's Pending Todos table rows directly, rather than regenerating the section via `gsd-tools init todos`'s `pending_todos_markdown` field, is inventing structure in a tool-owned file.

### Pitfall 5: `windows fixed <id>` accepts no note/reason parameter — confirmed by reading the source, not by guessing
**What goes wrong:** CONTEXT's discretion text hedges with "if the verb accepts a note, in the row." It does not.
**Why it happens:** `markFixed(ledger, id, opts)` (`gsd-core/bin/lib/broken-windows.cjs:286-293`) takes only `(ledger, id, opts)` with `opts` limited to `{ now }`; the CLI dispatcher `cmdWindowsMarkFixed` (`:1068-1088`) parses args via `parseArgs(args, { flags: [], required: [], positionals: 1 })` — exactly one positional (the id), zero flags of any kind. `[VERIFIED: gsd-core/bin/lib/broken-windows.cjs:286-293,1068-1088]`. Only `markWaived` accepts a `reason` string.
**How to avoid:** Verification evidence for #16/#33 goes in the plan SUMMARY only, never attempted as a CLI argument to `windows fixed`.
**Warning signs:** Passing a second positional to `windows fixed <id> "some note"` — this session accidentally confirmed the CLI ignores/rejects extra args (it does not error on the note; it simply reported "already fixed" on the second call because the first call with just the id had already succeeded and mutated `.planning/WINDOWS.md` — **do not invoke `windows fixed` during research/exploration; it is a real mutation, confirmed the hard way this session and reverted via `git checkout -- .planning/WINDOWS.md`**).

### Pitfall 6: GH #14's three originally-cited defect sites are already fixed
**What goes wrong:** A plan that re-derives fixes for `release.yml:327`, `docs/RELEASE.md:26-28`, and `docs/RELEASE-PROCEDURES.md:120-122` (GH #14's cited locations) will find nothing to fix there — those exact lines were already corrected by the `actions/attest-build-provenance` migration (D-09/D-10 in `release.yml`'s own history) that shipped after #14 was filed.
**Why it happens:** GH #14 is stale (CONTEXT already flags "the issue's line numbers are stale"), but this session's live census (word-boundary, multiline, positive-controlled — see Code Examples) confirms it more strongly than CONTEXT states: **zero live incorrect claims remain anywhere in the repo.** The only remaining hits for the "provenance ... over the checksums file" pattern are two **historical correction callouts** (`docs/RELEASE.md:145`, `docs/RELEASE-PROCEDURES.md:260`, both explicitly framed as "Corrected 2026-08-01 ... This section previously instructed ... Both were wrong") — these are deliberately retained as historical record, not live claims, and should not be reworded.
**How to avoid:** Run the census fresh (do not trust GH #14's line numbers or CONTEXT's summary), confirm zero live hits, and close GH #14 citing the census output as evidence the claim no longer exists anywhere — DOCS-09 becomes a verification task, not a rewrite task.
**Warning signs:** A plan task titled "fix release.yml:327" — that line number is stale and the file at that location today reads correctly.

### Pitfall 7: Taskfile `cmds:` strings pass through Go `text/template` before the shell ever sees them
**What goes wrong:** A literal `{{` written directly in a Taskfile `cmds:` block (including inside a shell comment) is consumed by Task's own templating engine, not passed through — this is already a documented, previously-hit pitfall in this exact Taskfile (`check:gonum`'s own `go list -f` line uses `{{"{{"}}.ImportPath{{"}}"}}`  to escape it).
**Why it happens:** `version: "3"` Taskfile semantics render every `cmds:` string as a Go template before shelling out.
**How to avoid:** Any new Taskfile content this phase might touch (unlikely, since GRD-11's work is CI-YAML-only, not Taskfile-target-authoring) must apply the same `{{"{{"}}...{{"}}"}}` escape for a literal template-looking sequence.
**Warning signs:** A `task: template: executing` or `reflect: call of reflect.Value.Type on zero Value` error.

### Pitfall 8: `web/build/**` must be rebuilt and committed in the same commit as any `web/src` change
**What goes wrong:** The GRD-10 re-vendor touches `web/src/lib/components/ui/**`. If `task web:build` is not re-run and `web/build/**` re-committed in the same commit, `task web:drift`'s SOURCE half will legitimately go RED (source changed, output didn't) — this is D-01's `web:drift` working exactly as designed, not a new incident.
**How to avoid:** After the re-vendor's diff is reviewed and approved, run `task web:build` then `task web:drift` (both must be GREEN) before committing, exactly as D-12's gate list requires.

## Code Examples

### GRD-09: the exact RED transcript to paste into `02-MUTATION-LOG.md`
```
# Source: live `task web:drift` run this session, `git worktree add --detach <scratch> 98cd41dd`
web:drift: hashed 108 source files
web:drift: manifested 22 output files
web:drift: source half MATCH (108 files, 1e0bff2fb48d58bd943e8bb842dff6d81f3c608105ed3d2b31aa6585a6b8f5c3)
::error::web:drift: OUTPUT-half mismatch — the committed web/build/ bytes are NOT the ones `task web:build` produced: someone edited, added, or removed a file inside the committed build output (marker: 32 files / 8658e6fe64bb02282e008557d39baba453d3e2765d6020071c8dc58d7ca9c432; recomputed: 22 files / a76add11cadd4bd4cb0322de2a0e62246187397236860cd0b9eb3e1e57bf39e2). Run `task web:build` to rebuild — but an unexplained output-half mismatch on a tree nobody rebuilt should be INVESTIGATED, not rebuilt away.
task: Failed to run task "web:drift": exit status 1  # exit code 201 at the shell level
```
Supporting facts, all verified live this session:
- `git status --porcelain` in the fresh worktree was **empty** — this is a genuinely clean checkout, matching D-01/D-02's "clean checkout" framing exactly.
- `98cd41dd` has **zero GitHub check-runs and zero commit statuses** (`gh api repos/seanb4t/codegraph-go/commits/98cd41dd/check-runs` → `total_count: 0`; `.../status` → `total_count: 0`) — confirms "98cd41dd never ran in CI" `[VERIFIED: GitHub REST API, this session]`.
- `98cd41dd` and its fix `fad3b39c` are still present as loose git objects (`git cat-file -t 98cd41dd` → `commit`) but are **not reachable from any current ref** (`git merge-base --is-ancestor 98cd41dd HEAD` → exit 1; `git branch --all --contains 98cd41dd` → empty) — history was squashed/rewritten after the fact, but the commits are still fetchable by SHA both locally and via `gh api` (confirmed the commit exists on GitHub).
- `fad3b39c`'s own commit message independently confirms the mechanism: "9 brand-new output files (3 chunks, 2 entry files, 4 nodes)" were never staged by `98cd41dd`'s bare `git commit web/build` — matching this session's file-count delta exactly (32 manifested − 22 actually-checked-out = wait, 9 missing chunks in fad3b39c's own count vs. the 10-file gap observed here; the discrepancy is that fad3b39c's diff stat shows +9 files, and this replay's manifest-vs-disk gap is 32−22=10 — the tenth is `.build-manifest` itself, which is `find`-excluded from `OUT_N`'s count but was still correctly written to disk at 98cd41dd; not a contradiction, just a reminder to state the count carefully in the mutation log).

### GRD-12: live ruleset JSON extraction pattern
```bash
$ curl -s "https://api.github.com/repos/seanb4t/codegraph-go/rulesets/20157557" \
  | jq -r '.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context'
test
actionlint (workflow static analysis)
perf regression gate (PERF-02, INDX-06)
pr-title
reproducibility (double-build hash-diff, DIST-04)
govulncheck (DIST-03, blocking)
```
`[VERIFIED: gh api / curl, this session]` — 6 contexts, unauthenticated works (200, no Authorization header). Compare against the CURRENT `requiredCheckNames` fixture (`internal/upgrade/taskfile_shape_test.go:77-85`), which has **7** entries including `"goreleaser check (config validation, DIST-01)"` and `"pr-title"` — i.e., the fixture and live set are **already mismatched today** (fixture has `goreleaser check`, live does not). This is exactly the drift GRD-12 exists to catch, and it will legitimately fire RED the moment the CI step is written, until D-08's ruleset PUT lands both new contexts.

Also verified this session directly against a real recent CI run's own reported grant (`gh run view 34820878640 --log`, unrelated job but same repo/token shape): `GITHUB_TOKEN Permissions: Contents: read / Metadata: read` — confirming `contents: read` at the workflow level (`ci.yml:42`) is sufficient; `Metadata: read` is granted automatically and needs no explicit `permissions:` addition.

### GRD-10: the real current drift (2026-09-14, run 34820878640, Corepack-pinned pnpm v11.23.0)
```
web:components:drift: compared 50 vendored component files across 8 components
Done in 4.2s using pnpm v11.23.0    # confirms Corepack correctly resolved the pinned version
##[error] ... 24 files differ (full list below), exit code 201
```
Full differing-file list `[VERIFIED: gh run view 34820878640 --log, this session]`:
```
button/button.svelte
command/command.svelte, command-group.svelte, command-input.svelte, command-item.svelte,
  command-link-item.svelte, command-separator.svelte, command-shortcut.svelte
dialog/dialog-content.svelte, dialog-description.svelte, dialog-overlay.svelte
input/input.svelte
input-group/input-group.svelte, input-group-addon.svelte, input-group-button.svelte, input-group-text.svelte
table/table-caption.svelte, table-footer.svelte, table-head.svelte, table-row.svelte
tabs/tabs.svelte, tabs-list.svelte, tabs-trigger.svelte
textarea/textarea.svelte
```
This is **24 of 50 files across 7 of the 8 families** (only `command`'s `command-group.svelte` etc. partially drifted — `command.svelte` itself also differs; no family shows zero drift except possibly one not listed above). CONTEXT.md's own text ("button.svelte, command-group.svelte and command-input.svelte") undercounts this significantly — it appears to describe an earlier, partial reading of the same run's annotations rather than the full log. This is the **latest available run**; `gh run list --workflow=components-drift.yml` shows no run since 2026-09-14, so D-09's "a fresh workflow_dispatch run is still triggered" has not yet happened as of this research session — the plan should decide whether to trigger a fresh run (an outward-facing CI action) or proceed directly to re-vendoring against this ~1-day-old evidence, which is very unlikely to have changed given the registry-side, not-local-toolchain nature of the drift (D-09 already independently confirmed).

### DOCS-09: positive-controlled multiline census (reproducible pattern)
```bash
$ rg -n -U -i 'provenance[^.]{0,80}(generated|attested)[^.]{0,40}over[^.]{0,40}checksums file' \
  README.md SECURITY.md docs/*.md .github/workflows/*.yml internal/upgrade/*.go
docs/RELEASE.md:145:> and stated that provenance was generated over the checksums file rather than
```
With a planted control phrase in a scratch file added to the same `rg` invocation, the pattern matched **both** the planted phrase and the one real historical-correction hit above — proving the census actually looks (rule `84d1gfpywd`'s positive-control discipline). The one real hit is inside a block explicitly headed `> **Corrected 2026-08-01** ... This section previously instructed ... Both were wrong` — i.e., **already-corrected historical narration, not a live incorrect claim**. `docs/RELEASE-PROCEDURES.md:260` carries the identical historical-correction framing. Live, current claims found elsewhere (README.md:178-179, SECURITY.md:45-46, docs/RELEASE.md:101, `.github/workflows/release.yml`'s Attest step comment) all **already** correctly state "subjects are the binaries," "over every published binary and `.zip` archive" — no rewrite needed.

### DOCS-08: current (stale) vs. actual go.mod counts
```
docs/RELEASE.md:338: "Of the 27 direct requires ... 14 are tree-sitter grammar modules"
docs/RELEASE.md:353: "The remaining 13 direct requires ... mark3labs/mcp-go"
```
Actual, verified this session (`awk '/^require \(/{f=1; next} /^\)/{f=0} f' go.mod`):
- **34 direct requires**, **116 indirect** (150 total) — not 27/13/134.
- `github.com/modelcontextprotocol/go-sdk v1.7.0` is the actual direct require (line 15) — `mark3labs/mcp-go` does not appear in `go.mod` at all.
- `gonum.org/v1/gonum v0.17.0` (GRF-10's new direct require) and `github.com/cockroachdb/pebble/v2 v2.1.6` are both present.
- 14 tree-sitter-related direct requires still holds structurally (13 grammar modules + 1 `go-tree-sitter` binding), but the "remaining 13" is now actually **20** (34 − 14). This confirms D-13's "drop raw counts entirely" decision is correct — the docs would need re-deriving on every dependency bump otherwise, exactly the drift WINDOWS #13 already flagged once at a different (32-direct) snapshot.

### DOCS-10: exact `tool-vuln` job identity for the one new SECURITY.md sentence
```yaml
# .github/workflows/ci.yml
tool-vuln:
  name: tool-vuln (VULN-01/02/03, advisory)
  ...
  - name: Tool-modfile vulnerability scan (advisory — reports, never fails the build)
    run: task vuln
```
`[VERIFIED: .github/workflows/ci.yml:311-330]`. The existing `SECURITY.md` paragraph (lines ~60-72) already correctly frames `govulncheck` vs. `pnpm audit` as disjoint blocking scanners; it says nothing about the isolated tool modfiles (`go.tool.mod`, `go.tool-lint.mod`, `go.tool-proto.mod`, `go.tool-golangci.mod`) being scanned separately and only advisorily. D-15's one sentence should name the job (`tool-vuln`) and state plainly that it reports, never blocks — matching the job's own name and comment verbatim rather than paraphrasing a new claim.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Third-party SLSA generic generator (`slsa-framework/slsa-github-generator`) + `slsa-verifier verify-artifact` | GitHub-native `actions/attest-build-provenance` + `gh attestation verify` | D-09/D-10, before this phase | The originally-cited GH #14 defect sites were rewritten as part of this migration; DOCS-09's census confirms no live incorrect claim survived it |
| `mark3labs/mcp-go` | `github.com/modelcontextprotocol/go-sdk` | Some point before this session (exact commit not traced; go.mod already shows the SDK) | `docs/RELEASE.md`'s credit line is stale and must be corrected per D-13 |

**Deprecated/outdated:** the raw dependency counts in `docs/RELEASE.md` (27/14/13/134/107) are all stale as of this session's `go.mod` read (34/14/20/150/116) — D-13's decision to drop counts entirely, rather than patch them to the current numbers, is the only approach that doesn't recreate the same staleness on the next dependency bump.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `.github/required-status-checks.txt` (newline-delimited, no comments) is the recommended data-file shape for D-07 | Architecture Patterns, Pattern 3 | Low — explicitly Claude's discretion per CONTEXT; any equally simple, parseable-by-both-Go-and-shell shape works equally well |
| A2 | The ruleset-drift CI step should use `curl` directly rather than `gh api`, to avoid `gh`'s own unauthenticated-refusal precondition | Architecture Patterns, Pattern 3 | Low — `gh api` would also work if `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` is exported into the step's env; either is a reasonable implementation choice, verified both directions this session |
| A3 | Placing the two new `check:*` steps and the ruleset-drift step in the exact order shown in the System Architecture Diagram (after `web:drift`/`proto:drift`, i.e., near the end of the JS gate chain rather than immediately after `Set up Node`) is acceptable | Architecture Patterns | Low — D-04 only requires "next to the pnpm-audit and web:drift steps," not an exact position; any placement after `Set up Node` (Pitfall 3) satisfies the stated constraint |
| A4 | Re-triggering `components-drift.yml` via `workflow_dispatch` before re-vendoring (rather than proceeding directly on the 2026-09-14 evidence) is the plan's call, not predetermined by research | Code Examples, GRD-10 section | Low — D-09 already independently refutes the local-toolchain hypothesis; a fresh run would very likely reproduce the same 24-file diff, but re-running costs one CI job and removes all doubt |

**If this table is empty:** N/A — see above; all four are low-risk implementation-detail choices explicitly left to Claude's discretion by CONTEXT.md, not domain claims needing user confirmation.

## Open Questions

1. **Should the plan trigger a fresh `components-drift.yml` `workflow_dispatch` run before re-vendoring, or proceed on the 2026-09-14 (run 34820878640) evidence?**
   - What we know: D-09 says "a fresh workflow_dispatch run is still triggered to capture the current diff for the re-vendor," implying the plan is expected to do this. No fresher run exists as of this research session.
   - What's unclear: Whether "still triggered" means research should have done it (out of scope for a read-only research pass — triggering CI is an outward-facing action) or the executing plan should.
   - Recommendation: The plan's first task should trigger the dispatch, wait for completion, and re-fetch the log — cheap (~1-2 min job) and removes any staleness doubt before committing to the 24-file re-vendor scope.

2. **How should the still-open `bench pinnedAt` todo (STATE.md's 4th Pending Todos row) be represented, given it has no `.planning/todos/pending/*.md` file and no CLI verb creates one?**
   - What we know: The underlying defect is real and unfixed (`tools/bench/runner/main.go:435-441`, verified). No `gsd-tools` verb writes a new pending-todo file; the `/gsd-capture`-family workflow is agent-authored, not CLI-driven.
   - What's unclear: Whether DOCS-11's scope includes authoring a proper todo file (arguably a new "capture" action, not "bookkeeping reconciliation") or whether it's acceptable to fold it into STATE.md's Blockers prose instead, leaving the Pending Todos *section* itself truthfully "None yet."
   - Recommendation: Fold it into STATE.md's Blockers/Concerns prose (a shape STATE.md already uses extensively for exactly this kind of standing, file-less item) rather than inventing a todo file under research-driven judgment; flag this choice for the maintainer to confirm at plan review, since it's a policy call about DOCS-11's scope boundary, not a technical one.

## Environment Availability

| Dependency | Required By | Available (CI runner) | Available (this dev machine) | Fallback |
|------------|------------|:---:|:---:|---|
| `syft` | `check:gonum`'s SBOM half | ✗ (must be added — see Pitfall 2) | — | Add the same `anchore/sbom-action/download-syft` step already used in `release.yml`/`linux-cross-canary.yml` |
| `node` (v24) | `check:no-force-layout`, `check:gonum`'s JSON parsing | ✓ (already set up at `ci.yml:131`) | ✓ (v26.8.2, no impact — target scripts use only Node builtins) | — |
| `jq` | ruleset-drift step | ✓ (GitHub-hosted runners ship jq; `post-release-verify.yml` already relies on it defensively) | ✓ | Defensive `command -v jq` check with a named `::error::` exit, matching `post-release-verify.yml`'s existing pattern |
| `corepack` (pinned pnpm 11.23.0) | GRD-10 re-vendor fidelity | ✓ (`components-drift.yml` already resolves it) | ✗ (not installed; bare `pnpm` on this machine resolves to v12.4.1) | **Do not attempt a local re-vendor on this development machine** — it would reproduce the exact "local-toolchain artifact" confound WINDOWS #31 already ruled out (D-09); re-vendor only under Corepack-pinned pnpm 11.23.0, either in `components-drift.yml`'s own runner or a machine with Corepack installed |
| `gh` CLI | GRD-12 verification, GRD-10 log fetching | ✓ (pre-installed on GitHub-hosted runners) | ✓ (used throughout this research session) | — |

**Missing dependencies with no fallback:** none — `syft`'s only "fallback" is the identical existing install step, which is the correct fix, not a workaround.

**Missing dependencies with fallback:** Corepack/pinned-pnpm on the local dev machine — do the re-vendor via CI (`components-drift.yml`'s runner, or a scratch branch + `workflow_dispatch`), never locally.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (standard library), `internal/upgrade/taskfile_shape_test.go` is the relevant shape-test file |
| Framework (JS) | Vitest (`web/package.json` `"test": "vitest run"`), `svelte-check` for `pnpm check` |
| Config file | `Taskfile.yml` (single source of truth for every CI job body — `TestWorkflowRunBodiesInvokeTask` enforces this project-wide invariant) |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/...` (Go shape tests); `task web:components:drift` (component drift, network-dependent, Corepack-only) |
| Full suite command | `task test:unit && task test:integration` (Go); `cd web && pnpm check && pnpm vitest run` (JS) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|--------------------|-------------|
| GRD-09 | `web:drift` fires RED against `98cd41dd`'s exact incident shape | manual replay + mutation-log | `git worktree add --detach <scratch> 98cd41dd && (cd <scratch> && task web:drift)` | ✅ (Taskfile target exists; replay reproduced this session) |
| GRD-10 | All 8 vendored families re-vendored to one snapshot, zero drift | shell/CI | `task web:components:drift` (must go GREEN post-re-vendor) | ✅ |
| GRD-11 | `check:gonum` / `check:no-force-layout` run in `ci.yml` on every PR | CI wiring | `task check:gonum`, `task check:no-force-layout` (already exist; new work is the `ci.yml` step) | ✅ (targets exist; CI step is new — Wave 0 gap below) |
| GRD-12 | `requiredCheckNames` compared against the live ruleset, fails on divergence | CI-only shell step | new step (see Pattern 3) | ❌ — Wave 0 gap |
| GRD-14 | #16/#33 closed via `gsd-tools windows fixed`; #20/#21/#34 stay open, record-only | CLI verb + SUMMARY prose | `gsd-tools windows fixed 16`, `gsd-tools windows fixed 33` | ✅ (verb exists; do not invoke during planning/research, only during execution) |
| DOCS-08 | `docs/RELEASE.md` dependency paragraph reflects current shape, credits `modelcontextprotocol/go-sdk` | manual doc edit, no test (D-13) | — | N/A by design |
| DOCS-09 | Zero live "provenance over checksums file" claims remain | census script | `rg -n -U -i '<pattern from Code Examples>' <scope>` with a planted positive control | ❌ — Wave 0 gap (the census command itself, not yet a committed script) |
| DOCS-10 | `SECURITY.md` states govulncheck/tool-vuln disjoint, advisory scope | manual doc edit, no test (D-15) | — | N/A by design |
| DOCS-11 | STATE.md Pending Todos reconciled; SEED-001 frontmatter records consumption | CLI verb + manual frontmatter edit | `gsd-tools init todos` (read) + manual STATE.md/SEED-001.md edits | ✅ (read verb exists; no automated drift test — matches DOCS-11's own text: "reconciled ... where none does, the SEED-002 precedent applies") |

### Sampling Rate
- **Per task commit:** run the specific target/step just added (`task check:gonum`, `task check:no-force-layout`, the ruleset-drift shell block extracted to a local script for iteration, `task web:components:drift`).
- **Per wave merge:** `task test:unit` (Go shape tests) + `cd web && pnpm check && pnpm vitest run` (JS, if GRD-10's re-vendor touched `web/`).
- **Phase gate:** full `ci.yml` `test` job equivalent locally where feasible (`task build && task vet && task lint:go && task test:unit`), plus a real PR to observe the new `check:gonum`/`check:no-force-layout`/ruleset-drift steps fire for real (the ruleset-drift step cannot be meaningfully unit-tested locally — it depends on the live GitHub API and the maintainer's D-08 ruleset edit).

### Wave 0 Gaps
- [ ] `.github/required-status-checks.txt` — does not exist yet; must be created in the same commit that refactors `taskfile_shape_test.go`'s `requiredCheckNames`.
- [ ] The ruleset-drift CI step itself — no existing script/target to adapt; author fresh per Pattern 3.
- [ ] The DOCS-09 census command — no existing committed script; author the `rg` invocation fresh (with its positive control) per Code Examples, and decide whether it becomes a one-off verification (run once, record output, no persisted script) or a committed reusable check — CONTEXT's D-14 implies one-off ("no drift test" isn't stated for DOCS-09 explicitly the way it is for D-13/D-15, but DOCS-09's own census is inherently a point-in-time claim about current text, not an ongoing guard — Claude's discretion on whether to commit the script).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V1 Architecture, Design and Threat Modeling | yes | The new ruleset-drift step reads a GitHub API endpoint at CI time; scope the request to `GET` only, never write, and never echo the response body verbatim into logs beyond the extracted context list (avoid leaking `bypass_actors`/`node_id` internal identifiers unnecessarily, though none are secret) |
| V5 Input Validation | yes | The shared data file (`.github/required-status-checks.txt`) is read by both a Go test and a shell step — treat blank lines and trailing whitespace defensively in both parsers (`sort`/`jq` on the shell side, `strings.TrimSpace` + skip-empty on the Go side) so a stray blank line doesn't silently widen or narrow the required set |
| V7 Error Handling and Logging | yes | D-06's hard-fail-never-skip requirement on the ruleset-drift step is itself a security-relevant control: a silently-skipping required check is a bypassable gate (mirrors GRD-07's declined "skip-clean offline" clause, rule `84d1gfpywd`) |
| V14 Configuration | yes | No secrets are needed for the new GitHub API read (public repo, unauthenticated works); do not add a `permissions:` write scope to accommodate it — `contents: read` already grants the required `Metadata: read` |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| A required-status-check comparison step that silently passes on API error, letting a stale/tampered ruleset go undetected | Tampering / Repudiation | Hard-fail on any non-2xx, empty body, or unparseable JSON (D-06); never a soft-fail or `continue-on-error` |
| Command injection via an unsanitized `components` variable in `web:components:drift`'s `pnpm dlx shadcn-svelte@1.5.1 add ${components} -y -o` | Tampering | Not new to this phase (pre-existing target), but the re-vendor plan should note it: `${components}` is derived from `git ls-files` output filtered through `awk`, not user input — low risk, already the existing pattern, no change needed |
| A malformed or truncated `.github/required-status-checks.txt` silently narrowing the required-check set (e.g., a bad merge leaves it empty) | Tampering / DoS on the guard itself | The Go test (`TestRequiredCheckNamesPreserved`-successor) must `t.Fatalf` on an empty/missing file rather than iterating zero times and passing vacuously — same rule `84d1gfpywd` discipline already enforced elsewhere in this file |

## Sources

### Primary (HIGH confidence — verified live this session)
- `git worktree add --detach <scratch> 98cd41dd` + `task web:drift` — direct reproduction of the GRD-09 incident, exit code 201, exact digest/count mismatch captured
- `gh api repos/seanb4t/codegraph-go/rulesets/20157557` and `curl` (unauthenticated) — live ruleset contents, 6 required contexts
- `gh api repos/seanb4t/codegraph-go/commits/98cd41dd/check-runs` and `/status` — confirms zero CI runs against that commit
- `gh run view 34820878640 --log` (components-drift.yml, 2026-09-14) — full 24-file drift list under Corepack-pinned pnpm v11.23.0
- `gh run list --workflow=components-drift.yml` — confirms no fresher run exists
- Direct `Read`/`rg` of `Taskfile.yml`, `.github/workflows/ci.yml`, `internal/upgrade/taskfile_shape_test.go`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `SECURITY.md`, `go.mod`, `web/scripts/check-no-force-layout.mjs`, `.github/workflows/components-drift.yml` — all line numbers and quoted text cited above were read this session
- Direct `Read` of `gsd-core/bin/lib/broken-windows.cjs` and `gsd-core/bin/gsd-tools.cjs` (routeWindows, cmdWindowsMarkFixed, markFixed) and `gsd-core/bin/lib/init.cjs` (renderPendingTodosMarkdown, cmdInitTodos) — confirms exact CLI verb shapes and the STATE.md table/bullet mismatch
- `gsd-tools init todos` (read-only) — confirms `.planning/todos/pending/` does not exist, contradicting STATE.md's own stale claim
- `gh issue view 14` — full original issue text and cited (now-stale) line numbers

### Secondary (MEDIUM confidence)
- `.planning/milestones/v0.13.0-phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` — the mutation-log shape/format precedent for `02-MUTATION-LOG.md`
- `.planning/STATE.md`, `.planning/WINDOWS.md`, `.planning/REQUIREMENTS.md`, `.planning/PROJECT.md` — project history and decision record, read this session for context and cross-checked against live code/CI evidence rather than trusted at face value (multiple staleness findings resulted from this cross-check)

### Tertiary (LOW confidence)
- None — every claim above was either verified against a live system this session or is explicitly marked `[ASSUMED]`/flagged in the Assumptions Log or Open Questions.

## Metadata

**Confidence breakdown:**
- CI wiring mechanics (GRD-11/GRD-12): HIGH — verified against live `ci.yml`, live GitHub ruleset API, and this session's own reproduced Taskfile runs
- GRD-09 replay: HIGH — directly reproduced, exact byte-for-byte transcript captured
- GRD-10 scope: HIGH — real CI log fetched, exact file list confirmed (materially corrects CONTEXT's undercount)
- Doc-census findings (DOCS-08/09/10): HIGH — direct `go.mod`/file reads, live census run with positive control
- Bookkeeping (GRD-14/DOCS-11): HIGH on the verb shapes (source-code-verified); MEDIUM on the *policy* question of how to represent the file-less `bench pinnedAt` todo (an Open Question, not resolved by research)

**Research date:** 2026-09-15
**Valid until:** Short — this research pins several live, mutable external states (the GitHub ruleset's current 6 contexts, the components-drift 24-file diff, `go.mod`'s exact counts). Re-verify the ruleset and component-drift numbers immediately before executing the plan if more than a few days elapse; the CLI-verb and file-content findings (Pitfalls 1-8) are stable until the next `gsd-core` upgrade or the next commit to the cited files.
