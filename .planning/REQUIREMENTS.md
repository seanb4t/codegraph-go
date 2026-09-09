# Requirements: CodeGraph Go — v0.13.0 Guard Hardening & UI Follow-through

**Defined:** 2026-09-08
**Core Value:** CodeGraph Go gives coding agents a pre-indexed code knowledge graph — fast symbol/call-path/impact queries served from a single static, verifiably-built binary, with no bundled runtime to install or manage.

**Milestone goal:** Burn down the deferral backlog in two coherent sets — close every known guard that cannot fire, then finish the contained UI follow-ons v0.12.0 deliberately left — with a self-authored CLI reference as the documentation tail.

**Consumes:** Backlog 999.2 (tmux TTY harness) and 999.4 (`CheckRegression` positivity guard); three pending todos (dry-run-signed guard, post-release-verify guard, tap-secret test); the T-01-18 archtest todo; DOCS-05 (deferred at v0.11.0); v0.12.0's v2 deferrals BRW-10, BRW-11, HLT-04, GRF-06; the brew-trust docs todo.

**Scope decisions recorded at definition (maintainer, 2026-09-08):**

- GRF-06 clustering is computed **fresh per call, deterministically**, inside `FileGraph()`; index-time persistence into the reserved 50-59 field range is the documented fallback only if GRF-09's measurement fails.
- BRW-11's editor URI template is set by a **server flag with a per-browser UI override**.
- HLT-04's pre-extraction exclusion reasons are **recorded at discovery time and persisted additively**, never inferred by a query-time re-walk.
- The two further guard-class open threads (`requiredCheckNames` vs live ruleset; root `SECURITY.md`'s govulncheck claim) stay in open threads — **declined for this milestone**, not forgotten.

## v1 Requirements

### Guard Hardening

- [x] **GRD-01**: `CheckRegression` refuses a non-positive *current* throughput or peak-RSS reading with an error naming the degenerate field, mirroring the existing baseline check — demonstrated RED with `current.PeakRSSBytes = 0` on an otherwise-matching frame before the fix lands (999.4)
- [x] **GRD-02**: A persisted archtest asserts `internal/query` imports no wire-layer package (`internal/uiserver`, `connectrpc.com/connect`, `internal/mcp`), following `internal/graphstore/archtest`'s `go/packages` pattern with a package-count sanity check and a positive control that fails when the expected importer disappears (T-01-18)
- [ ] **GRD-03**: `release:dry-run-signed`'s additions-only diff guard carries a positive assertion that the awk anchor matched and the `--key=` injection occurred, failing when the anchor stops matching rather than passing vacuously
- [ ] **GRD-04**: A test parses `post-release-verify.yml` and asserts every job carries the event-aware conclusion guard in its expected shape, failing when the guard is removed or inverted on any job
- [ ] **GRD-06**: Each of GRD-01..04 is recorded in a committed mutation log with pasted failing output from its RED demonstration and a byte-clean revert, following `03-MUTATION-LOG.md`'s precedent

### tmux Real-PTY Harness

- [ ] **TTY-01**: A tmux-driven harness builds the binary, spawns it in a tmux pane, sends keys and captures the pane, gated behind a build tag; without tmux on `PATH` the suite reports a skip with a reason, never a silent pass (999.2)
- [ ] **TTY-02**: Frame capture polls until two consecutive captures are identical before asserting; no fixed sleeps in the assertion path
- [ ] **TTY-03**: Bare `codegraph daemon` on a TTY with an empty registry renders only the `no running daemons` line, with zero DECRQM/mode-query response bytes in the pane or scrollback (closes the G-07-1 class)
- [ ] **TTY-04**: The daemon picker enters the alternate screen, renders `Running daemons` plus a seeded record, and on quit restores the main buffer with no residual escape sequences in scrollback (closes the G-07-2 class)
- [ ] **TTY-05**: The install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, and `q`/`esc` cancel with zero config-file writes, asserted by hashing the config tree before and after
- [ ] **TTY-06**: A flicker proxy asserts frame stability across N captures on an idle picker
- [ ] **TTY-07**: A CI job installs a pinned tmux version, runs the harness suite, and asserts a positive count of executed (not skipped) test cases

### Browse & Inspect

- [ ] **BRW-10**: While scrolling a long file's source, a single-line breadcrumb shows the innermost containing symbol, computed client-side from `FileSymbols` line ranges; verified in a live browser against a real index, not only jsdom
- [ ] **BRW-11**: Node detail and source views offer an "open in editor" link built from a URI template with `{path}`, `{line}` and `{col}` placeholders, resolved server-side so the absolute path never weakens repo-root confinement at the RPC boundary
- [ ] **BRW-12**: The template default is set by `codegraph ui --editor-url <template>` and can be overridden per browser in the UI; presets exist for VS Code, Cursor and JetBrains; Zed is neither a preset nor a claimed target
- [ ] **BRW-13**: Editor handoff ships with a threat model naming the browser's external-protocol prompt, not CSP, as the security boundary, and covering validation of the template's inputs

### Index Health

- [ ] **HLT-04**: Index health shows the coverage denominator — files discovered but not indexed — with a per-file reason, distinguishing extraction failures (already persisted in `File.errors`) from pre-extraction exclusions
- [ ] **HLT-05**: Pre-extraction exclusion reasons (vendor/dot-dir, unsupported extension, build tag, size limit) are recorded at the discovery decision point and persisted additively within SchemaVersion 1, never inferred by a query-time re-walk
- [ ] **HLT-06**: The coverage surface extends `GetHealthResponse` additively, or adds an rpc whose name clears every `mutatingVerbs` substring including "Index", with `wantUIServiceMethods` updated by set-equality in both directions

### Graph View

- [ ] **GRF-06**: The file/package graph colors nodes by community, computed fresh inside `FileGraph()` following the `CycleID` precedent, on the existing layered layout — never a switch to force-directed
- [ ] **GRF-08**: Community assignment is deterministic (sorted iteration, fixed seed, canonical relabeling), proven by a test running the algorithm at least three times on identical input and asserting label-canonicalized equality
- [ ] **GRF-09**: Clustering time in isolation is measured against the guava corpus with the pass threshold committed before measurement, following GRF-01's precedent; a failing measurement triggers the documented fallback (index-time persistence into the reserved 50-59 range), not a raised bar
- [ ] **GRF-10**: The new `gonum.org/v1/gonum` dependency passes govulncheck, appears in the SBOM, and its import closure is verified to contain no cgo

### Documentation

- [ ] **DOCS-05**: A self-authored `docs/CLI-REFERENCE.md` documents every command and flag in the Cobra tree, replacing what `docs/FLAG-PARITY.md` used to carry
- [ ] **DOCS-06**: A drift guard walks the live Cobra tree — including hidden, inherited persistent and deprecated flags — and fails when any registered flag is missing from the reference; demonstrated RED by registering a throwaway flag
- [ ] **DOCS-07**: The brew-trust instructions recommend the narrow grant with security framing rather than the broader `--tap` grant

## v2 Requirements

Deferred to a future release. Tracked but not in the current roadmap.

### Graph View

- **GRF-07**: Opt-in whole-symbol graph scoped within a single already-drilled-into package — parked on the v0.12.0 Phase 5 evidence (3,233-node file view never converged); revisit only with a measured budget

### Browse & Inspect

- **BRW-14**: Nested scope-stack breadcrumb (every enclosing symbol, not only the innermost) — the deferred enhancement over BRW-10

### Guard Hardening

- **GRD-07**: `requiredCheckNames` fixture compared against the live `protect-main` ruleset via `gh api`, skip-clean offline — declined for this milestone by maintainer
- **GRD-08**: Root `SECURITY.md`'s govulncheck-gates-every-merge claim corrected with the advisory caveat and a drift assertion — declined for this milestone by maintainer
- **GRD-05**: The tap App secret-distinctness test reads secret names from the real release-please and tap workflow files rather than two in-test constants — declined for this milestone by maintainer (2026-09-08, Phase 7 discussion); the tautological test is deleted in Phase 7 rather than rewritten, and D-16's two-App distinctness rests on documentation and the v0.5.0 one-time proof

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| PreToolUse guard hook (GUARD-HOOK-01/02) and multi-agent skill/hooks porting (AGENT-04…07) | Gated on evidence that skill+resources+nudge are insufficient, which nobody has measured; porting is blocked on unverified per-agent hook-schema differences |
| homebrew-core submission (BREW-07) | External review queue and notability criteria we cannot schedule |
| Stapled offline-safe container (DIST-06) | Waits on evidence of real users hitting the offline first-launch case |
| Team Scale (central server, CI-distributed indexes) | Its own milestone, not a deferral |
| SEED-003 markdown in the index | No "why" written yet; enrich the seed first |
| Server-side shell-out to launch an editor | Breaks read-only-by-construction (SRV-03); the browser's own URI-handler dispatch is the mechanism |
| "Re-index this file" auto-remediation on the coverage view | Same SRV-03 violation; show reason and remedy as text only |
| Force-directed graph layout for clustering | Repeatedly rejected in v0.12.0; clustering is rendered as coloring on the existing layered layout |
| Persisted cluster assignments as the default | Violates ENG-03's fresh-per-call discipline; retained only as GRF-09's documented fallback |
| Project-wide mutation-testing framework | Heavyweight for five diagnosed instances; the hand-authored RED-demonstration convention already works at this scale |
| A Go tmux client library | No viable one exists; `os/exec` over the stable tmux CLI matches the repo's existing git/brew interop style |
| Cobra-generated CLI reference as the source of truth | `cobra/doc` silently excludes hidden flags by design; the reference is hand-authored and guarded by a live-tree walk |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| GRD-01 | Phase 7 | Complete |
| GRD-02 | Phase 7 | Complete |
| GRD-03 | Phase 7 | Pending |
| GRD-04 | Phase 7 | Pending |
| GRD-06 | Phase 7 | Pending |
| TTY-01 | Phase 8 | Pending |
| TTY-02 | Phase 8 | Pending |
| TTY-03 | Phase 8 | Pending |
| TTY-04 | Phase 8 | Pending |
| TTY-05 | Phase 8 | Pending |
| TTY-06 | Phase 8 | Pending |
| TTY-07 | Phase 8 | Pending |
| BRW-10 | Phase 9 | Pending |
| BRW-11 | Phase 9 | Pending |
| BRW-12 | Phase 9 | Pending |
| BRW-13 | Phase 9 | Pending |
| HLT-04 | Phase 10 | Pending |
| HLT-05 | Phase 10 | Pending |
| HLT-06 | Phase 10 | Pending |
| GRF-06 | Phase 11 | Pending |
| GRF-08 | Phase 11 | Pending |
| GRF-09 | Phase 11 | Pending |
| GRF-10 | Phase 11 | Pending |
| DOCS-05 | Phase 12 | Pending |
| DOCS-06 | Phase 12 | Pending |
| DOCS-07 | Phase 12 | Pending |

**Coverage:**

- v1 requirements: 27 total
- Mapped to phases: 27 ✓
- Unmapped: 0 ✓
- Duplicated across phases: 0 ✓ (every requirement maps to exactly one phase)

Per-phase totals — Phase 7: 5 (GRD-01…04, GRD-06) · Phase 8: 7 (TTY-01…07) · Phase 9: 4 (BRW-10…13) · Phase 10: 3 (HLT-04…06) · Phase 11: 4 (GRF-06, GRF-08, GRF-09, GRF-10) · Phase 12: 3 (DOCS-05…07). 5 + 7 + 4 + 3 + 4 + 3 = 26.

The v2 requirements (GRF-07, BRW-14, GRD-05, GRD-07, GRD-08) are deliberately unmapped and stay in
`ROADMAP.md` → Milestones → **Later**; GRD-07 and GRD-08 were *declined for this milestone*
by maintainer decision (2026-09-08), not deferred by omission; GRD-05 was declined at the Phase 7 discussion the same day.

---
*Requirements defined: 2026-09-08*
*Last updated: 2026-09-08 after roadmap creation (Phases 7–12; 27/27 mapped)*
