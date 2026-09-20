# Pitfalls Research

**Domain:** Adding tmux-driven TUI e2e testing, graph community detection, editor URI handoff, Svelte 5 scroll-tracking, a discovery/skip-reason surface, guard-hardening, Go import-direction archtests, and CLI-reference drift guards to an existing Go + Svelte 5 + ConnectRPC + Pebble system (`codegraph-go`, v0.13.0 "Guard Hardening & UI Follow-through")
**Researched:** 2026-09-08
**Confidence:** MEDIUM-HIGH (grounded directly in this repo's existing code, `PROJECT.md`, and `RETROSPECTIVE.md`; cross-checked against current external sources for the parts that are genuinely ecosystem-level — tmux/DECRQM behavior, Louvain/Leiden determinism, Svelte 5 `$effect`, Cobra hidden-flag doc generation)

This research targets four theme-groups the milestone already names, and maps every pitfall to one:

- **G — Guards that cannot fire** (999.4 `CheckRegression`, T-01-18 archtest, `dry-run-signed` additions-only diff, `post-release-verify.yml` conclusion guard, tap-secret test)
- **T — tmux real-PTY e2e harness (999.2)** — lands *before* UI work by design
- **U — UI follow-through** (BRW-11 editor handoff, BRW-10 breadcrumb, HLT-04 discovery/skip reasons, GRF-06 community detection) — ordered by increasing scope
- **D — Docs tail** (DOCS-05 CLI reference + drift guard, brew-trust framing)

## Critical Pitfalls

### Pitfall 1: A community-detection guard that asserts "clustering ran" instead of "clustering is stable"

**What goes wrong:**
Louvain/Leiden/label-propagation are non-deterministic by construction unless every source of randomness is pinned. The most likely test written first — "does `Engine` return a non-empty set of clusters for a known graph?" — passes whether or not two consecutive calls (or two runs on identical input) return the *same* partition. That is exactly this repo's own guard shape (rule `84d1gfpywd`): an assertion that is true regardless of whether the property under test holds.

**Why it happens:**
Standard Louvain/Leiden implementations iterate nodes/edges in whatever order the underlying map or adjacency-list gives them, break modularity-gain ties arbitrarily, and (for Leiden specifically) use randomness in the refinement phase — all confirmed non-deterministic behavior in the current literature: different node orderings or initialization states produce different partitions across runs, and Leiden's own randomness requires an explicit seed/theta control to tame. Label propagation is worse — it is explicitly order-dependent by algorithm design, not incidentally. This repo already computes analogous whole-graph structures (`Engine.FileGraph()`, `BuildReverseAdjacency`) "fresh-per-call full-scan" with no persisted projection — the natural default for community detection is to do the same, which means every UI page load / every RPC call re-runs the algorithm, and any non-determinism becomes user-visible cluster reshuffling, not just a test-suite risk.

**How to avoid:**
- Iterate nodes and edges in a fixed, explicit order (sorted by node ID, not map iteration order) at every step the algorithm touches — Go map iteration order is randomized per-process specifically to prevent code from depending on it, so any code path that ranges over a `map[NodeID]...` without first collecting and sorting keys is non-deterministic by default, not by mistake.
- Fix every source of randomness to a **constant seed** (not time-based, not `crypto/rand`), and make tie-breaking a deterministic rule (e.g., "prefer the lower community ID, then the lower node ID") rather than "first one found."
- Write the test as: run the algorithm N times (N ≥ 3) on the identical input and assert the outputs are byte-identical (or set-identical up to a canonical community-ID relabeling — Louvain/Leiden don't guarantee stable *labels* across runs even when the *partition* is stable, so the guard must canonicalize labels, e.g. by lowest-member-node-ID, before comparing) — never assert "ran without error" or "returned len > 0."
- Persist the result into the `Node` schema's already-reserved field range (`internal/schema/graph.proto` reserves 50–59 on `Node` explicitly for "community/cluster assignment"). Persisting at index time, keyed to the same deterministic algorithm run once, sidesteps most of the query-time-determinism problem entirely — the UI reads a stored value, not a live recomputation — and is consistent with this project's D-02a additive-only schema discipline. **This is the strongest single mitigation for this pitfall** and should be the default design, not the fallback.
- If computed on demand instead (`FileGraph()`-style, to avoid re-indexing on schema change), the freshness contract must be explicit: same graph state in ⇒ same clustering out, verified by the repeated-run test above, on every code path that can trigger it (including the incremental-resync path in Pitfall 2).

**Warning signs:**
- The only test for the clustering endpoint checks `len(clusters) > 0` or `err == nil`.
- Any `for k, v := range someMap` in the clustering code with no prior `sort.Slice`.
- A seed parameter that defaults to `time.Now().UnixNano()` or is absent entirely.
- Manual QA notices cluster colors/groupings visually "jump" between two loads of the same file/package graph with no underlying data change.

**Phase to address:** U (GRF-06) — the guard must exist in the same phase/commit that introduces clustering, not retrofitted after; this is this repo's own stated discipline (rule `84d1gfpywd`, and the T-01-18 lesson that a mitigation proven once and never committed as a test is not closed).

---

### Pitfall 2: Community assignments that go stale — or silently reshuffle — across incremental re-index

**What goes wrong:**
`codegraph sync` is incremental: it reparses changed files and recomputes only the affected subgraph (`internal/indexer/pipeline.go`'s `FilesReparsed`/`FilesPruned`/`DependentsRecomputed` machinery). Community detection algorithms are global — a single edge addition/removal can, in principle, change the optimal partition anywhere in the graph. Two failure modes follow: (a) if clusters are persisted per-node and only recomputed for the touched subgraph, stale cluster IDs linger on untouched nodes whose "true" community has actually shifted, and the UI silently shows a wrong grouping with no signal that it's stale; (b) if clusters are fully recomputed on every sync (even a one-file change), the deterministic-seed discipline from Pitfall 1 can still produce a *materially different* partition for the whole graph on a trivial edit, which reads to a user as "my one-line change reorganized the entire graph view" — technically correct given global optimization, but a UX regression against the "third consumer, never a second implementation, stable/deep-linkable navigation model" bar this milestone inherits from v0.12.0.

**Why it happens:**
Nothing in this codebase's sync path has ever had to reconcile a *global* derived property against a *local* diff before — `FileGraph()` and `BuildReverseAdjacency` are both stateless recomputations from edges that already exist, with no cross-call consistency requirement. Community detection is the first feature in this project's history where "recompute correctly" and "recompute stably" are two different, both-necessary properties.

**How to avoid:**
- Decide explicitly, before writing code, whether clusters recompute on every sync or only on a full re-index — and write that decision into `Meta` (additive field, following the `HasFileIndex`/`commit_sha` precedent) as a generation/version stamp so the UI can detect "clusters are from an older graph generation" rather than silently rendering a mismatch.
- If recomputing on every sync: seed the algorithm with the *previous* partition as its initialization where the underlying algorithm supports it (Louvain/Leiden both support warm-starting from an existing partition), which tends to minimize gratuitous relabeling for small diffs — this is a real mitigation, not just a determinism fix, and should be evaluated against the "reshuffle" complaint above specifically.
- Test this with a scenario the sync test suite doesn't yet have: index a real multi-file corpus, capture the cluster assignment, touch one file with a trivial change (e.g., a comment), re-sync, and assert the resulting clustering differs from the baseline only within an explicitly bounded region (not "differs nowhere" — that's too strong given global optimization is real — but "differs nowhere unrelated").

**Warning signs:**
- No `Meta`-level generation/version field ties a served clustering to the graph state it was computed from.
- The sync path recomputes clusters but the incremental-resync test suite (`internal/indexer` sync tests) has no test that touches clustering at all.
- Manual UAT: edit one file, re-sync, watch the graph view — if the whole layout reorganizes for an unrelated one-line change, that's the warm-start gap manifesting.

**Phase to address:** U (GRF-06).

---

### Pitfall 3: Community detection blows through the same rendering wall GRF-01 already hit once

**What goes wrong:**
v0.12.0's `GRF-01` render threshold genuinely FAILED at 3,233 nodes (guava's file view never became interactive past 10.1 minutes) before a remedy was chosen and re-measured. Community detection sits *downstream* of that same rendering pipeline — it adds a clustering pass on top of a node/edge set the milestone context says already failed to converge once — and GRF-07 (whole-symbol graph) was explicitly parked for this exact reason ("symbols multiply that; revisit only with a measured budget"). A first cut of GRF-06 that runs clustering server-side on the same aggregated file/package graph and then feeds an unbounded result into the same client renderer risks reproducing GRF-01's failure with an extra O(V+E) or worse clustering pass bolted onto it, and — worse — risks *masking* it: a clustering algorithm that itself takes non-trivial time on a 3,000+ node graph could make a slow render look like a slow *algorithm*, misdirecting debugging effort.

**Why it happens:**
Louvain/Leiden are near-linear in practice but not free, and most reference implementations are single-threaded, unbudgeted Go/Python/C loops with no cancellation point — exactly the profile that turned GRF-01's render into a >10-minute hang once node count crossed a threshold this project has already measured as real, not hypothetical.

**How to avoid:**
- Reuse GRF-01's methodology exactly: commit a pass/fail threshold *before* measuring (this repo's own strongest evidence-shape per the retrospective), measure clustering time and render time *separately* on the same guava-scale corpus already used for GRF-01, and only then decide whether clustering runs server-side (cacheable, one-time cost per generation) vs. client-side (repeated cost per view).
- Given clustering is a *global* graph property (unlike `FileGraph()`'s local aggregation), strongly prefer server-side computation persisted to the schema (Pitfall 1) over client-side recomputation — this also sidesteps shipping a clustering algorithm to the browser bundle at all.
- Do not let GRF-06 quietly expand scope toward GRF-07's parked whole-symbol graph; keep clustering scoped to the same file/package granularity `FileGraph()` already operates at.

**Warning signs:**
- No committed pass/fail threshold exists before the first clustering-time measurement is taken.
- A benchmark exists for render time but not for clustering-computation time in isolation.
- The clustering implementation runs in the same request/goroutine as the render-blocking RPC handler with no timeout or cancellation.

**Phase to address:** U (GRF-06), sequenced after Pitfall 1's determinism work but before UI wiring.

---

### Pitfall 4: The `internal/query` archtest for T-01-18 is written as a one-shot check again, not a committed guard — or is committed vacuously

**What goes wrong:**
T-01-18's own stated defect is "empirically true, but its mitigation was a one-shot check never committed as a test." The two failure modes for the *fix* are: (a) repeating the exact same mistake — running `go list`/`packages.Load` once by hand, confirming the direction holds, and not leaving behind a `_test.go` that runs in CI; (b) committing a test that *looks* like this repo's own established pattern (`internal/graphstore/archtest/import_graph_test.go`, `internal/cli/present/archtest/import_graph_test.go`) but omits the positive-control sanity check those two already carry — both existing archtests explicitly fail loudly if the expected importer disappears from the loaded graph ("no package ... was found importing X — this test cannot verify enforcement"), because without that check, a refactor that deletes the only real import the rule is protecting would make the guard pass **for the wrong reason** — the forbidden import simply isn't there to find, not because the rule held.

**Why it happens:**
`golang.org/x/tools/go/packages.Load` genuinely can resolve fewer packages than expected (silently, on a moved/renamed package — `internal/cli/present/archtest/import_graph_test.go` already guards against this explicitly with a count check), and `packages.Load` without `Tests: true` is blind to test-only imports (both existing archtests set it deliberately, with an inline comment explaining why). A T-01-18 test copy-pasted without reading why those two lines exist reproduces the exact "guard that cannot fire" defect class this whole milestone exists to burn down.

**How to avoid:**
- Copy the *structure* of `internal/graphstore/archtest/import_graph_test.go` exactly: `packages.Load` with `Tests: true`, a `len(pkgs) == 0` sanity check, and — critically — a "did we actually find at least one real (illegitimate-if-misplaced) importer to verify against" positive control before trusting a clean pass.
- Demonstrate the new `internal/query` archtest RED before merging: temporarily add a forbidden import in a scratch commit, confirm the test fails with the expected message, then revert — this repo's own standing acceptance bar (`git diff -S` audit trail expected, not just a green run once).
- Name the exact direction being enforced in the test's doc comment the way the two existing tests do (`internal/graphstore/archtest/import_graph_test.go`'s header names D-04a explicitly) — a future reader needs to know which direction is forbidden without archaeology.

**Warning signs:**
- The new test has no `Tests: true` and there exists (or could exist) a test-only import that would violate the direction rule invisibly.
- No assertion that the loaded package count matches an expected guarded-package count.
- No assertion that at least one real (or planted, in a mutation-proof CI check) violator is detectable — i.e., the test would also pass if the forbidden dependency direction were structurally impossible to create, meaning it verifies nothing.
- The test exists only as a one-off `go run` invocation in a plan's evidence log, not as a `_test.go` file that runs under `go test ./...` and therefore CI.

**Phase to address:** G — this is explicitly named in the milestone's guard-hardening set.

---

### Pitfall 5: `go/packages.Load` cold-cache latency in CI turns a correctness guard into a flaky/slow one

**What goes wrong:**
`packages.Load` invokes `go list` under the hood, which on a cold module cache (a fresh GitHub Actions runner, or `actions/cache` miss) can take tens of seconds for a project this size (~209k lines of Go across many `internal/indexer/*extract` packages) — multiplied across every one of this project's now-five `archtest` packages (`graphstore`, `cli`, `cli/present`, `mcp`, plus the new `internal/query` one) if each does its own independent `packages.Load(cfg, ".../...")` call rather than sharing one load. This isn't a correctness pitfall but it is exactly the kind of thing that turns a good guard into one people learn to `-short`-skip or that intermittently times out in CI, which is a slow path to the same "guard that cannot fire" outcome (a guard nobody trusts is a guard nobody investigates when it goes red).

**Why it happens:**
Each `import_graph_test.go` today calls `packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")` independently with its own `Config` — reasonable for isolation, but it means CI pays the full module-graph resolution cost N times, once per archtest package, on every run.

**How to avoid:**
- Confirm CI already caches `$GOMODCACHE`/`$GOPATH/pkg/mod` and the build cache (`actions/setup-go`'s built-in caching, or an explicit `actions/cache` step) before assuming this is a problem — check the existing `ci.yml`/`Taskfile.yml` job that runs the current four archtest suites for its actual measured wall-clock time first, per this project's own "measure before deciding" discipline (GRF-01's pattern).
- If it is measurably slow, prefer a single shared `packages.Load` call (loading the whole module graph once) with each archtest package asserting its own rule against the shared result, over N independent loads — this is a refactor of the *existing* four archtests as much as it is new work for the fifth, so scope it explicitly rather than only solving it for `internal/query`.
- Do not silently drop `Tests: true` or narrow the load pattern to "just the packages I think matter" to save time — that reintroduces Pitfall 4's blind spot for a performance win.

**Warning signs:**
- Archtest suite wall-clock time grows noticeably (measure, don't guess) after adding the fifth `import_graph_test.go`.
- CI flakiness reports specifically on archtest jobs, correlated with cache-miss runs (first run after a `go.mod` change, or a scheduled job on a fresh runner).

**Phase to address:** G, as a measured check alongside the T-01-18 work — not a blocking prerequisite, but worth a single wall-clock measurement before the fifth archtest ships.

---

### Pitfall 6: tmux capture-pane races the TUI's own render — flakiness that looks like a bug in the TUI

**What goes wrong:**
`tmux capture-pane` returns whatever is in the pane buffer *at the instant it's called*, with no built-in signal for "the application has finished rendering this frame." A test that sends keys and immediately captures will intermittently see a partially-rendered frame, a still-blank pane, or (worse) a frame from *before* the keypress was processed — producing exactly the class of flake this milestone's own list already anticipates ("timing... capture-pane timing vs render"). This is a generic tmux-testing pitfall, not specific to bubbletea, but bubbletea's render loop (batched Msg → Update → View, rendered on its own goroutine's schedule) makes the race window variable rather than fixed, so a fixed `sleep 200ms` that "works" locally is exactly the kind of guard that passes on a fast dev machine and flakes on a loaded CI runner.

**Why it happens:**
There is no synchronous "render complete" signal available to an external tmux client by default — the mode-2026 synchronized-output escape sequence (`DECSET 2026`) is designed to solve a *different* problem (preventing the terminal from painting a half-written frame to the *display*), not to give an external harness a completion signal it can poll for.

**How to avoid:**
- Never use a fixed `sleep` as the sync primitive. Poll `capture-pane` on an interval with a bounded total timeout, and assert on *content stability* — e.g., capture twice N milliseconds apart and require the two captures to be byte-identical before trusting the content, which is a real synchronization primitive rather than a guess at render latency.
- Where the TUI itself can be made to emit a detectable, deterministic marker on settle (e.g., a status line that includes a monotonically increasing frame counter, gated behind a test-only env var/build tag so it never ships in the release binary), prefer that over pure polling — it turns "probably done" into "provably done." Confirm this doesn't leak into production output before committing to it (this project already has a build-enforced ANSI-isolation seam precedent from v1.0 worth reusing as the template for "test-only instrumentation never reaches the agent/production output path").
- Budget generous, CI-aware timeouts (CI runners are frequently 2-4x slower than a dev laptop for terminal/PTY-heavy work) and fail with the captured pane content in the error message, not just "timeout" — a flaky test that dumps no diagnostic is expensive to debug the second time it fails.

**Warning signs:**
- Any `time.Sleep(fixedDuration)` between a `send-keys` and a `capture-pane` in the new harness.
- Tests that pass locally and flake only in CI, correlated with CI-runner load rather than a specific assertion.
- No stability-check (capture-twice-and-compare) anywhere in the harness's polling helper.

**Phase to address:** T (999.2), and this is the single highest-flake-risk item in that phase — prototype the polling/stability primitive first, before writing the first real scenario test against it.

---

### Pitfall 7: DECRQM/mode-2026/2027 probe responses leak into captured pane content when the terminal running tmux doesn't answer them

**What goes wrong:**
The milestone context calls this out explicitly: `\e[?2026$p` (synchronized-output support query) and `\e[?2027$p` (grapheme-cluster-mode query) are DECRQM report-mode requests. If the TUI (or a dependency — bubbletea/lipgloss/termenv all probe terminal capabilities) sends one of these and the terminal running inside the tmux pane doesn't answer with a `DECRPM` response in the expected window, the raw escape sequence itself can end up sitting in the pane's scrollback or being echoed back and re-captured — producing literal escape-sequence garbage (`\e[?2026;2$p`-shaped bytes) in a `capture-pane` result that a naive test then string-matches against and either false-fails (garbage where clean output was expected) or — worse — false-passes (the garbage happens to not intersect the substring being checked, silently hiding a real capability-detection bug).

**Why it happens:**
tmux itself only gained DECRQM handling for mode 2026 recently (August 2026, per current tmux development), and even where tmux answers correctly, the layer *outside* tmux (the CI runner's own pty/terminal emulation, or `TERM`/`COLORTERM` env vars that don't match what tmux is told to advertise) can still fail to answer a probe tmux forwards, or answer with different capability bits than a real interactive terminal would (iTerm2, Alacritty, kitty, foot are the terminals with confirmed 3.4+ support — a bare `xterm`/`screen` `TERM` value in CI is not guaranteed to be one of these).

**How to avoid:**
- Pin `TERM` and `COLORTERM` explicitly in the CI job (don't inherit whatever the runner image defaults to) to a value this project has verified tmux answers correctly for — and record that value, and the tmux version it was verified against, directly in the workflow file as a comment, since this is exactly the kind of environment-sensitivity that silently breaks on a runner-image bump.
- Assert pane content with a filter that strips *all* recognized escape sequences before substring-matching, not a filter that only strips SGR color codes — a garbage DECRPM response is not an SGR sequence and a color-stripping regex will not catch it.
- Add a dedicated, minimal harness test whose only job is "launch the TUI, capture the pane, assert zero raw `\e[?` DECRPM-shaped bytes appear anywhere in the captured content" — this is the guard that would have caught the milestone context's named failure mode directly, and it should exist as its own test, not be inferred from other tests passing.
- Prefer disabling capability *probing* in the test harness's invocation where the TUI library supports an explicit override (e.g., forcing a known profile via `termenv`/`lipgloss` env vars) over trying to make every CI terminal answer every probe correctly — fewer variables to get right.

**Warning signs:**
- Captured pane assertions use a plain string `strings.Contains` check with no ANSI-stripping, or a color-only stripping regex.
- No test exists whose sole purpose is asserting the *absence* of raw escape-probe bytes.
- CI failures that show a captured string containing `?2026` or `?2027` substrings embedded in otherwise-normal-looking output.

**Phase to address:** T (999.2).

---

### Pitfall 8: tmux version / OS drift between ubuntu-latest and macOS runners silently narrows what the harness actually proves

**What goes wrong:**
`ubuntu-latest` and macOS GitHub-hosted runners ship different tmux versions (and macOS's is frequently older, sourced from Homebrew at image-build time rather than tracking Ubuntu's package cadence) — a harness written and tuned against whichever runner is developed on first can pass there and silently no-op, skip, or behave differently on the other, especially around DECSET 2026 support (only tmux 3.4+) and pane-sizing behavior. This project's own release matrix is linux/{amd64,arm64} + darwin/{amd64,arm64} (v0.4.0), so a harness that only meaningfully exercises one OS family is a guard with a real, silent scope gap — the same shape as this repo's other "correct enumeration, wrong population" defect class, just applied to CI matrix coverage instead of code coverage.

**Why it happens:**
tmux isn't preinstalled identically across GitHub-hosted runner images and isn't pinned by this project today; "gated behind a build tag / CI job with tmux available; skips cleanly elsewhere" (as scoped) is correct for *portability* but creates a real risk that "skips cleanly" quietly becomes "always skips on the runner that matters" if the availability check is wrong for one OS.

**How to avoid:**
- Pin an explicit minimum tmux version check (not just "tmux is on PATH") inside the harness's skip logic, and log the detected version on both skip and run so a reviewer can see from CI output which runners actually executed the suite versus silently skipped.
- Run the tmux job on both OS families in the CI matrix explicitly (not just linux, where tmux is cheaper to install/pin via apt), and if macOS's tmux is meaningfully behind on DECSET 2026 support, install a pinned newer tmux via Homebrew in that job rather than accepting the image default — this project already does exactly this kind of explicit pinning for its zig-cross toolchain and GoReleaser version, so there's an established pattern to follow, not a new one to invent.
- Add a canary-style always-run job (this project's own pattern from `release.yml`'s path-scoped canary, established specifically because "its bad outcomes are quiet") that fails loudly and immediately if the tmux availability check silently degrades from "ran" to "skipped" on a runner it previously ran on — a bare skip count trending toward 100% over time is the leading indicator, and nothing currently watches for that trend by default.

**Warning signs:**
- The harness has ever been observed to skip on CI without anyone noticing until much later.
- No version-pinning or version-logging in the tmux-availability check.
- The harness was developed and is habitually run/verified locally on only one OS.

**Phase to address:** T (999.2).

---

### Pitfall 9: The daemon picker's alt-screen enter/restore is asserted by absence, which is exactly this repo's own recurring vacuous-guard shape

**What goes wrong:**
"Enters the alternate screen and restores the main buffer on quit" is naturally tested by capturing the pane before entry, during the picker, and after quit, and asserting the "after" state matches the "before" state (the main buffer was restored). A weaker, easier-to-write version of this test only checks that the picker's *own* content is gone after quit — which passes identically whether the alt-screen was properly entered-and-exited, or whether the picker simply cleared the (still-primary) screen buffer directly and never used the alt-screen mechanism (`\e[?1049h`/`\e[?1049l`) at all. That's a materially different, and worse, behavior (scrollback pollution, no restore of prior terminal content) that the weak test cannot distinguish from correct behavior.

**Why it happens:**
"Content looks right after quit" is the intuitive thing to check and is *necessary* but not *sufficient* — it's the same "enumerate what exists, not what's missing" trap this project's own v0.12.0 retrospective named as a new defect class (a correct-looking check whose blind spot is structural, not a bug in the check itself).

**How to avoid:**
- Assert the actual escape sequences: capture raw pane output (not the rendered/interpreted view) around entry and exit and confirm `\e[?1049h` appears on entry and `\e[?1049l` on exit — this directly tests the mechanism, not just an inferred side effect of it.
- Additionally assert content that was on the primary screen *before* entering the picker (e.g., a sentinel string printed to the terminal before launching the TUI) is still present and unscrolled after quitting — this is the actual user-facing property "restores the main buffer" promises, and it's a property the escape-sequence check alone doesn't fully cover either (a library could emit the right codes but still corrupt content around them).

**Warning signs:**
- The test only checks "picker UI text is gone" post-quit, with no check for the alt-screen escape codes or pre-existing terminal content survival.

**Phase to address:** T (999.2).

---

### Pitfall 10: Editor URI handoff treats CSP as the security boundary when it isn't one

**What goes wrong:**
The milestone context explicitly asks about "CSP interaction" for `vscode://`/`idea://`/`cursor://`/`zed://` links, and the SPA's actual CSP (`default-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'` — read directly from `internal/uiserver/spa.go`) has no directive that governs top-level navigation to a non-http(s) custom scheme at all. `navigate-to` was a CSP Level 3 draft directive intended to cover exactly this case and was never shipped by any browser — it does not exist as an enforceable control today. A reviewer or implementer who reasons "the CSP already locks this down" is trusting a boundary that structurally cannot apply here, and will under-scope the actual mitigation work.

**Why it happens:**
CSP genuinely does govern `fetch`/`connect-src`/script/style loading on this page (and correctly protects those), so it's a natural but wrong inference that it also governs `<a href="vscode://...">` or `window.location = 'vscode://...'` — those are top-level browsing-context navigations, a different security model entirely, governed by the browser's own external-protocol-handler policy, not CSP.

**How to avoid:**
- Treat the real boundary as: (a) the browser's native "Open [App]?" confirmation prompt (present in Chrome/Edge/Firefox for unregistered/less-trusted schemes, sometimes with a "always allow" checkbox that, once checked, removes the prompt entirely for that origin+scheme pair going forward — a one-time click that permanently downgrades the UX to silent-launch), and (b) whatever validation this project does on the *path/line* values it interpolates into the URI before generating the link.
- Validate and encode the file path defensively before building the URI: use the abs path exactly as `codegraph`'s own index already stores it (already normalized), percent-encode per the URI scheme's actual argument grammar (VS Code's `vscode://file/<path>:<line>:<col>` is *not* a generic URL — colons in the path itself, spaces, and `#`/`?` characters need scheme-specific handling, and a naive `url.URL{Scheme: "vscode", ...}.String()` will not automatically produce a URI these editors' handlers parse correctly cross-platform), and never accept a client-controlled override of the *base* path — only the line/column should be client-influenced, since the server already knows the true file path for any node/symbol.
- Do not conflate "the editor scheme is configurable" (a legitimate settings feature named in BRW-11) with "the target path is configurable" — keep the scheme prefix (`vscode://`, `cursor://`, etc.) as the only user-editable part of the template, with the path/line values always server-derived from the already-validated index, closing off path-injection via a crafted symbol name or file path that somehow reached the client.
- Document, in the feature's own text, that this is a genuinely different trust boundary than the rest of the read-only UI — clicking a link in a *read-only* browsing tool causing a *local editor to open a file* is new capability this milestone adds, not a variant of anything the read-only guarantee (`mutatingVerbs`) already covers, since it's an OS-level side effect the RPC layer's read-only invariant says nothing about.

**Warning signs:**
- Any code or doc comment asserting the CSP "prevents" or "restricts" the vscode:// navigation.
- The URI-building code uses generic URL escaping (`net/url`) without a scheme-specific format function, or accepts a raw path string from a request parameter rather than deriving it from the already-resolved node/symbol on the server side.
- No design note addressing what happens when a user has previously checked "always allow" for the scheme — silent-launch is the steady-state behavior to design for, not the one-time-prompt behavior.

**Phase to address:** U (BRW-11).

---

### Pitfall 11: Editor-link prompt fatigue drives users to "always allow," which then makes a same-origin XSS (however unlikely) a one-click local file-open primitive

**What goes wrong:**
A file/graph view with many navigable symbols will surface an editor-open affordance repeatedly, and per-scheme "always allow" is a real per-browser feature exactly because per-click prompts don't scale to that usage pattern — so the realistic steady state, once a user has used the feature a handful of times, is silent auto-launch with no confirmation at all. At that point, the CSP-hardened, `mutatingVerbs`-pinned read-only surface's *only* remaining protection against "a compromised/injected asset opens arbitrary local files in the user's editor" is whatever validation this feature does server-side (Pitfall 10) — there is no browser-side friction left once "always allow" is set, and this project's own CSP explicitly cannot add any (Pitfall 10 again). This raises the actual stakes of Pitfall 10's server-side validation from "nice to have" to "the only remaining control."

**Why it happens:**
Prompt-fatigue-driven "always allow" is standard, expected browser UX, not user error — the feature's own success (being useful enough to click often) creates the condition that removes the friction.

**How to avoid:**
- Treat Pitfall 10's server-side path validation as load-bearing security work, not polish — thread it through this milestone's `SECURITY.md` threat model explicitly (this repo's own process already requires `<threat_model>` blocks and a per-phase SECURITY.md; v0.12.0's Phase 1 shipping without one was called out as the only such gap in project history — don't repeat it for a phase that adds a *new* trust boundary).
- Since the read-only RPC layer cannot be the enforcement point for this (it's a client-side navigation, not an RPC call), the enforcement has to live in what values the server ever puts into a rendered `href` — audit that no code path can cause the server to embed a path outside what the index itself resolved (e.g., a malformed deep-link URL parameter should never flow through to the editor-link `href` unvalidated).

**Warning signs:**
- The threat model for this phase treats the browser prompt as sufficient mitigation with no server-side validation named.
- A deep-link route parameter (already a feature per v0.12.0's "deep-linkable navigation model") flows into the editor-link path without being re-validated against the actual indexed file set.

**Phase to address:** U (BRW-11) — specifically its `SECURITY.md`.

---

### Pitfall 12: HLT-04's "reason" field lies by construction if it's inferred after the fact instead of recorded at the point of decision

**What goes wrong:**
The milestone context names this exact failure mode: "reasons that are lies (e.g. 'unsupported language' when the truth is 'exceeded MaxSourceBytes')." Today, `internal/indexer/pipeline.go`'s `Stats.Skipped` is a bare integer — no per-file reason is captured anywhere in the current pipeline; the doc comment literally says skips happen for "`parser.ErrSourceTooLarge` or a read failure" but the *specific* cause per file is discarded once it's folded into the count. A natural-looking implementation of HLT-04 that reconstructs reasons *after indexing*, by re-deriving them from file extension + current registered-extractor set (e.g., "no `.rs` extractor registered ⇒ report 'unsupported language'"), will get it wrong for exactly the files where the truth is a different, real skip cause (oversized, unreadable, malformed) that happened to also lack — or happened to have — an extractor. That's not a hypothetical edge case; it's the default failure mode of any inference that doesn't consult the actual decision point.

**Why it happens:**
The temptation named directly in the milestone context — "the temptation to re-walk the filesystem at query time" — is real because it's *cheaper* to implement: no new schema field, no new pipeline plumbing, just a `filepath.Walk` plus a language-support lookup at query time. But that re-walk has no access to the actual reason a file was skipped during indexing (a since-fixed permissions issue, a transient read error, a size ceiling this specific version enforces) — it can only ever *guess* a plausible-sounding reason from static, present-tense file properties, which is structurally the same shape as this repo's own "reason that is a lie" warning.

**How to avoid:**
- Capture the real reason at the exact point `internal/indexer/extract.go`/`goextract.go` (and siblings) already decide to skip a file — the code comments already reference `parser.ErrSourceTooLarge` and "a read failure" as the two known real causes; a third-file-skip type check (no extractor registered for the extension at all, distinct from "extractor registered but failed to parse") should be added as its own explicit reason, not conflated with either.
- Define a small, closed, additive reason enum (mirroring D-02a's additive-only schema discipline) rather than a free-text string — a closed set is easier to keep honest than prose that can drift, and easier to test exhaustively ("every skip path sets exactly one of these reasons, and every reason has at least one test that trips it for real").
- Persist per-file skip reasons through the same `x/` file-owned secondary index this project already uses for rename/delete/move pruning (Phase 4, v1.0) rather than inventing a second discovery-tracking mechanism — this keeps the additive-schema and single-source-of-truth disciplines this project already enforces, and avoids the query-time re-walk trap entirely: the UI reads what indexing actually recorded, not a live re-derivation.
- Test each reason with a **positive control**: for every reason value, construct a real file that trips exactly that path (an actually-oversized file for the size-ceiling reason, an actually-unreadable file via a permissions fixture for the read-failure reason, an actual unregistered-extension file for the no-extractor reason) and assert the recorded reason matches — this is the same discipline v0.11.0's golden-corpus work already established ("new goldens must be demonstrated RED against a mutation before being trusted").

**Warning signs:**
- HLT-04's implementation re-walks the filesystem and infers reasons from static file properties at query/serve time, rather than reading from something written during indexing.
- The reason is a free-text string built with `fmt.Sprintf` at the point it's *displayed*, rather than a fixed value set at the point it's *decided*.
- No test constructs a real file per reason category and asserts the recorded reason — only a synthetic/mocked skip event is tested.
- The existing `parser.MaxSourceBytes` skip path and the "no extractor for this language" path share a single generic reason string in the implementation (a strong sign they were conflated rather than distinguished).

**Phase to address:** U (HLT-04).

---

### Pitfall 13: HLT-04's coverage denominator becomes a second, silently divergent "what counts as a file" definition

**What goes wrong:**
"Coverage denominator" implies a ratio: indexed / discovered. This project already has at least one notion of "discovered" (whatever the indexing walk enumerates before extraction) and the query-serving side already has its own file listing (`Files` on `Engine`). If HLT-04 introduces a *third* definition of "discovered" (e.g., a fresh walk at query time, per Pitfall 12's temptation) that uses different ignore rules, symlink handling, or `.gitignore`/binary-file heuristics than the indexing walk actually used, the denominator itself becomes wrong independent of whether individual reasons are honest — the ratio can read "87% coverage" when the true figure, using the walk that actually ran, is different, and nothing in the system would notice the two counts have quietly diverged.

**Why it happens:**
Discovery logic (what counts as a candidate source file at all — extension allowlist, `.gitignore` respect, binary detection, symlink policy) tends to live close to the indexing walk and isn't obviously reusable from a query-serving package without a deliberate refactor; it's easy for a new surface to reimplement "walk the tree and count files" from scratch using `filepath.WalkDir` + a plausible-looking filter, rather than calling the actual function the indexer used.

**How to avoid:**
- Reuse the indexer's actual discovery function (not a reimplementation) as the source of the denominator, or — better — record the discovered-count *at index time* alongside the skip reasons from Pitfall 12, so the denominator is a stored fact from the run that actually happened, not a recomputation that can drift from it.
- Add a test that specifically exercises `.gitignore`-excluded files, symlinks, and binary files, and asserts HLT-04's denominator agrees with the indexer's own discovery count on the same corpus — a disagreement here is the exact "silently divergent definition" failure mode, and it needs an explicit cross-check, not just each side's own tests passing independently.

**Warning signs:**
- HLT-04's discovery logic is a new `filepath.WalkDir` call rather than a call into the indexer's existing discovery path.
- No test cross-checks the HLT-04 denominator against the indexer's own file-count stat on the same corpus.

**Phase to address:** U (HLT-04).

---

### Pitfall 14: `$effect` double-invocation recurs a third time because the root cause (reading reactive state inside the effect that the effect itself schedules writes to) isn't named as a pattern yet

**What goes wrong:**
v0.12.0 Phase 6 already hit two `$effect` double-invocation bugs, and BRW-10's scroll-tracking breadcrumb is a canonical trigger for exactly this class: an `IntersectionObserver` callback that updates reactive state (`currentSymbol = ...`) inside an `$effect` that itself reads reactive state to decide whether/how to (re)create the observer creates a feedback loop Svelte 5's fine-grained reactivity will re-run more than once per logical scroll event — the documented root cause (an effect's dependency set is re-recorded from scratch on every run, and a workaround as blunt as "read the value once outside the loop" is the community's own reported fix, which suggests this is a live, unresolved rough edge in the framework, not solely an application bug).

**Why it happens:**
Nothing in this codebase currently records "don't read-and-write the same piece of state inside one `$effect`'s reactive closure" as an established local rule — the two v0.12.0 occurrences were each caught by *live browser UAT*, not by a written pattern anyone could check code against in review, meaning the same shape can recur a third time by the same mechanism (an author not knowing to look for it) that produced the first two.

**How to avoid:**
- Structure the breadcrumb specifically to avoid the trap: create the `IntersectionObserver` once (e.g., in `$effect` with an empty/stable dependency set — observer *construction* should not depend on the scroll position it's meant to report), and have the observer's callback (not the `$effect` body) perform the reactive-state write — the callback runs outside Svelte's effect-tracking context by default, which is the structural fix, not a `sleep`/read-once workaround.
- Explicitly clean up the observer in the `$effect`'s teardown (return a cleanup function) so navigating away from a file view doesn't leave a stale observer whose callback still fires and mutates state for an unmounted view — a second known Svelte 5 effect-cleanup gap distinct from the double-invocation one, and worth its own explicit test given this is a long-scrolling, deep-linkable file view where mount/unmount churn is routine.
- Given this is now a recurring category (2 occurrences in v0.12.0 alone), write the pattern down somewhere durable this repo already trusts for cross-cutting frontend conventions (this milestone should be the one that promotes it from "caught twice in live UAT" to "documented rule"), not just fix BRW-10's instance in isolation.
- Do not rely on jsdom-based unit tests to catch this — v0.12.0's retrospective is explicit that "jsdom verifies the model; it never verifies the rendering," and both prior `$effect` bugs were live-browser-UAT findings specifically because headless tests structurally couldn't see them. Budget a live browser UAT pass for BRW-10 as a required gate, not optional polish.

**Warning signs:**
- The `IntersectionObserver` is constructed inside an `$effect` whose own dependency array/closure reads the same state variable the observer's callback writes.
- No cleanup function is returned from the `$effect` that creates the observer.
- The only tests for the breadcrumb are Vitest/jsdom component tests with no live-browser pass.

**Phase to address:** U (BRW-10).

---

### Pitfall 15: Layout thrash from breadcrumb updates racing scroll, invisible to any headless test

**What goes wrong:**
A scroll-tracking breadcrumb that reads DOM geometry (element `getBoundingClientRect()`/intersection ratios) on every scroll tick and writes it straight back into reactive state that triggers a re-render can force synchronous layout recalculation on every frame (classic forced-reflow/layout-thrash), producing visibly janky scrolling on long files — a real, user-perceptible defect class that (per Pitfall 14's closing point) is invisible to jsdom, which has no real layout engine at all.

**Why it happens:**
`IntersectionObserver` itself is *designed* to avoid this (it doesn't poll geometry synchronously on scroll), so choosing it (as the milestone context already directs) avoids the naive scroll-handler version of this problem by construction — but a poorly tuned `threshold`/`rootMargin` (too many thresholds, or a threshold array producing many callback firings per scroll distance) or additional geometry reads inside the callback itself (e.g., calling `getBoundingClientRect()` again inside the callback "just to be sure") can reintroduce the same class of cost even while nominally using the right API.

**How to avoid:**
- Use a small, deliberate `threshold` set (not a dense array) and set `rootMargin` to define the "currently containing symbol" boundary precisely rather than firing on every fractional-visibility change — tune this against the same guava-scale corpus already used for GRF-01/GRF-06's rendering-budget work, since a long real file (not a toy fixture) is the actual stress case.
- Don't call `getBoundingClientRect()` or any other synchronous layout-forcing API from inside the observer callback — trust the `IntersectionObserverEntry` the callback already receives.
- Verify with a live browser pass under Chrome DevTools' Performance panel (or equivalent) on a genuinely long file, watching for forced-reflow warnings — this is exactly the kind of check jsdom cannot perform and that this milestone's own precedent (live UAT finding what headless tests structurally cannot) says must be a real gate, not a nice-to-have.

**Warning signs:**
- Dense `threshold` arrays (e.g., `Array.from({length: 100}, (_, i) => i / 100)`) on the observer.
- Any `getBoundingClientRect()` call inside the observer callback.
- No live-browser performance check exists for the breadcrumb feature specifically.

**Phase to address:** U (BRW-10).

---

### Pitfall 16: `CheckRegression`'s positivity guard gets "fixed" by adding a floor that also passes vacuously

**What goes wrong:**
999.4's stated defect is precise: "a zero `PeakRSSBytes` reading passes both the relative and the absolute INDX-06 check today." The milestone context names the exact fix-pass anti-pattern to watch for: the repair introduces a *new* vacuous guard shaped like the old one — e.g., "fix" it with `if peakRSS <= 0 { t.Skip(...) }` (silently downgrades a real failure to a skip, which is arguably worse — a skip doesn't even show as failing in a summary the way a loud error would) or with a floor check (`peakRSS > 0`) that's *itself* satisfiable by a different measurement bug (e.g., a metric that's always `1` due to integer-division-to-zero-then-plus-one, or a stub/mock code path left wired in from testing that returns a constant).

**Why it happens:**
This repo's own retrospective names this exact recurring shape across every milestone reviewed here — "a fix-composition Critical" where the correction to one guard interacts badly with something else nearby, or is itself a second guard shaped the same way as the first. A "positivity guard" fix that only adds `> 0` treats the *symptom* (zero passes) rather than the *mechanism* (why does the metric ever legitimately read zero, and how would a genuine regression to zero be distinguished from a genuine, currently-broken zero-reading instrument?).

**How to avoid:**
- Root-cause *why* `PeakRSSBytes` can read zero in the first place before writing the guard — this project's own history already has the answer pattern (`getrusage`/`ru_maxrss` must be read from the OS on the *child process*, never in-process, per this repo's own v0.1 lesson) — if the zero is coming from an in-process read or a platform where `ru_maxrss` genuinely isn't populated the way the code assumes, fixing *that* is the real fix; a `> 0` assertion on top of a broken reader just moves the vacuous-pass one layer deeper.
- Demonstrate the new guard RED against the actual historical bug: reproduce (or synthetically inject) the exact zero-reading condition that let INDX-06 pass vacuously before, and confirm the new guard fails on it, then confirm it passes on a known-good measurement — this is the mutation-proof discipline this project already applies to goldens and archtests; apply it here too, since a "guard that cannot fire" fix is precisely the class this whole milestone is meant to close, and closing it with another vacuous guard would be the single worst possible outcome for this specific phase.
- Prefer asserting a *plausible range* (e.g., "greater than some floor that a real Go process's peak RSS could never be below, given the binary's own known minimum footprint") over a bare `> 0`, since `> 0` is trivially satisfied by almost any bug in the reader, including ones that produce numbers with no real relationship to actual peak RSS.

**Warning signs:**
- The fix is a single added comparison (`> 0`) with no change to *how* the metric is captured.
- No test reproduces the historical zero-reading condition and confirms the new guard catches it.
- The guard's fix commit doesn't reference or link back to the original INDX-06 finding it's closing.

**Phase to address:** G — flagged directly by the milestone as the canonical example of this whole theme; treat it as the template the other four "guards that cannot fire" items in this set should be fixed the same way (root-cause, then RED-demonstrate, then close).

---

### Pitfall 17: `release:dry-run-signed`'s additions-only diff guard is "fixed" by widening the awk pattern instead of re-deriving it from the actual diff semantics

**What goes wrong:**
The stated defect — "passes vacuously when its awk anchor stops matching" — describes an `awk` (or similar line-anchored) pattern matching against `git diff` output that silently stops matching (e.g., after a diff-format change, a locale/encoding difference, or a refactor of the surrounding release script) and therefore reports "no removals found" not because none exist, but because the tool can no longer see removals *at all*. The naive fix — tweak the awk pattern until it matches the *current* diff output shape again — repeats the exact defect if it isn't paired with a positive control, since the pattern can drift out of sync with the diff format again on the next incidental change, and nothing would notice.

**Why it happens:**
Line-oriented text tools (`awk`, `grep`, ad hoc `sed`) over `git diff` output are exactly this repo's own recurring "query that could not find" failure class (v0.5.0's retrospective names four instances in one session of exactly this shape: `rg -c` matching version strings, `task` echoing unexpanded vars, etc.) — the diff's *textual* shape (context-line count, `\ No newline at end of file` markers, rename-detection headers) is not a stable contract, and a guard built directly against it inherits that instability.

**How to avoid:**
- Prefer a structured diff API over line-anchored text parsing where feasible — `git diff --numstat` or `git diff --name-status` give a stable, parseable machine format explicitly designed for tooling, rather than the human-readable unified-diff format an `awk` pattern is reaching into.
- Whatever the mechanism, add a positive control identical in spirit to the archtest pattern already established in this repo (Pitfall 4): construct a synthetic commit that *does* remove a line the guard is meant to catch, confirm the guard fires, revert. This is the direct, mechanical fix for "passes vacuously when its anchor stops matching" — a guard proven to fire on a real removal cannot silently stop matching without the positive-control test also failing.

**Warning signs:**
- The fix changes only the awk/grep pattern text, with no accompanying test that feeds it a synthetic diff containing a real removal.
- The guard's implementation parses `git diff`'s default unified-diff text output rather than a `--numstat`/`--name-status`/porcelain form.

**Phase to address:** G.

---

### Pitfall 18: `post-release-verify.yml`'s conclusion guard and the tap-secret test get "fixed" by asserting the artifact instead of the property

**What goes wrong:**
Two related items: (a) "no test asserts [the event-aware conclusion guard], so a regression is silent" — the fix needs to actually assert the *behavior* (does the workflow correctly gate/skip/run based on the triggering event, per this project's own established `workflow_run` + validated-`tag`-input pattern from v0.5.0's retrospective, which was chosen specifically because `release: [published]` fires too early); a fix that merely documents the intended behavior in a comment, or asserts the YAML *contains* a `workflow_run` trigger key (structure) without asserting it actually *gates on the right condition* (behavior), is a guard that passes on the presence of a keyword rather than the presence of correct logic. (b) the tap-secret test "compares two in-test constants and reads no workflow" — the textbook version of a guard that cannot fail: it's testing that a value equals itself, restated once, never touching the real artifact (the actual GitHub Actions secret / workflow file) it's supposed to be verifying.

**Why it happens:**
Both are variants of the same root failure this repo's retrospective names repeatedly across milestones: a test that asserts something *about its own fixture* rather than about the *real system* it's meant to guard (v0.11.0's supersede-memory schema misread, v0.12.0's "10 of 10" edits that made 0 real changes) — cheap to write, looks green, verifies nothing.

**How to avoid:**
- For the conclusion guard: write a test that actually parses (or, better, invokes with a mocked/fixture event payload) the real workflow YAML and asserts its *decision* for at least two contrasting event inputs (one that should gate/skip, one that should proceed) — asserting a decision under two contrasting inputs is what makes the guard capable of failing; a single-input smoke test cannot distinguish "correctly gates" from "always proceeds" or "always skips."
- For the tap-secret test: change it to read the actual value it claims to verify — the real workflow file's secret reference, or (if secrets literally cannot be read in a test context) at minimum a fixture that is demonstrably sourced *from* the real workflow (e.g., extracted by the test itself via a parse step) rather than a second hand-typed constant living beside the first. Two hand-typed constants can drift from each other trivially and independently of whether the real system is even correct.
- Demonstrate both RED: for the conclusion guard, feed it the event payload that should be rejected and confirm it currently isn't (or would drift silently); for the tap-secret test, temporarily change the real secret name/value and confirm the "fixed" test notices — if it doesn't notice, it still reads no workflow.

**Warning signs:**
- The tap-secret test file contains two string/const literals and one `==`/`Equal` comparison between them, with no I/O, no file read, no HTTP/workflow-dispatch call.
- The conclusion-guard test asserts on YAML structure/keys present, not on a computed decision for a given input.
- Neither test has ever been observed to fail.

**Phase to address:** G.

---

### Pitfall 19: DOCS-05's CLI-reference drift guard is built on Cobra's own doc generator, which deliberately omits hidden flags — making "every registered flag is documented" false by construction

**What goes wrong:**
Cobra's built-in markdown doc generator (`doc.GenMarkdownTree`) intentionally does not include hidden flags in its output — this is documented, current Cobra behavior, not a bug. If DOCS-05's drift guard is built as "diff the generated docs against the committed `CLI-REFERENCE.md`," it will by construction never notice a hidden flag going undocumented, because the source of truth it's diffing against (the generator's own output) already excludes them — the guard can be perfectly green while the actual invariant it's meant to assert ("every registered flag is documented") silently narrows to "every *visible* registered flag is documented," a materially weaker claim than the one named in the milestone context.

**Why it happens:**
It's the natural, minimum-effort implementation — Cobra ships the generator, so "diff against what Cobra generates" looks like reuse rather than reinvention, but it inherits Cobra's own scoping choice (hidden flags are hidden from *users*, which the generator faithfully reflects) into a drift guard whose job is explicitly the opposite — auditing the *registration* surface, not the *user-facing* surface.

**How to avoid:**
- Enumerate the actual registered-flag surface directly via `cmd.Flags().VisitAll` / `cmd.PersistentFlags().VisitAll` (which sees hidden flags — `Flag.Hidden` is just a display bit, not an exclusion from the `FlagSet`) rather than via the generated-docs output, for the "what must be documented" side of the guard.
- Decide, explicitly and in writing in the guard's own doc comment, whether hidden flags are (a) required to appear in `docs/CLI-REFERENCE.md` too (even if marked "internal/undocumented-for-users" inline) or (b) deliberately excluded with a named allowlist of exceptions — either is defensible, but "silently excluded because the generator excludes them" is not a decision anyone made, it's an inherited default.
- Cover the other three named blind spots explicitly in the guard, since each is a distinct way "every flag is documented" goes wrong in a different direction from hidden flags: **persistent flags inherited from parents** — a subcommand's *effective* flag set includes everything `InheritedFlags()` reports, not just `Flags()` (a guard that walks only each command's own `Flags()` under-reports every subcommand's true surface); **flags registered conditionally at runtime** (e.g., behind a build tag or feature flag, or added in an `init()`/`PersistentPreRun` rather than at command construction) — a static enumeration that only walks the command tree as constructed at `main()` entry, before any conditional registration runs, will miss these entirely, so the guard must run against the *fully wired* command tree, not a partially-constructed one; **deprecated flags** — Cobra's `Flag.Deprecated` marks a flag as still-functional-but-discouraged, and the guard needs an explicit policy (documented-with-a-deprecation-note vs. excluded) rather than falling into whichever behavior the reused generator happens to have.
- Demonstrate RED: register a new hidden flag, a new conditionally-registered flag, and a new deprecated flag in a scratch commit; confirm the guard flags all three as undocumented (or correctly recognizes their explicit exception, per whichever policy was chosen); revert.

**Warning signs:**
- The guard's implementation calls into `doc.GenMarkdownTree` (or diffs against its output) as its source of "what flags exist," rather than walking `FlagSet`s directly.
- The guard walks `cmd.Flags()` only, never `cmd.InheritedFlags()`.
- No test in the guard's own suite plants a hidden, a conditionally-registered, and a deprecated flag and confirms detection of all three.
- `docs/FLAG-PARITY.md`'s deletion (v0.11.0) removed the prior drift guard (`internal/cli/flag_parity_test.go`) entirely — confirm DOCS-05's new guard isn't a near-verbatim resurrection of that deleted file without addressing whatever gap motivated deleting it as "framing to retire" rather than "a check to keep and rename."

**Phase to address:** D (DOCS-05).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Recompute community clusters fresh on every UI request instead of persisting into the schema | No schema change, no re-index-time cost | Query-time non-determinism becomes user-visible reshuffling (Pitfall 1); harder to make byte-identical across two browser tabs | Only for a throwaway spike measuring clustering cost in isolation (Pitfall 3) — never ship this as the default |
| Reconstruct HLT-04's skip reasons by re-walking the filesystem at query time | Zero pipeline/schema plumbing | Reasons become plausible-sounding guesses, not facts — the exact "lie" the milestone context warns about | Never — this is explicitly named as the wrong path in the milestone brief itself |
| Use a fixed `sleep` between tmux `send-keys` and `capture-pane` to "make CI green" | Fast to write, often passes locally | The single largest recurring source of CI flake this feature will produce; erodes trust in the whole harness | Never, even as a stopgap — poll-with-stability-check costs one function, not a redesign |
| Diff Cobra's generated docs output for the CLI-reference drift guard, reusing Cobra's own generator | Least code to write | Silently narrows "every flag documented" to "every visible flag documented" (Pitfall 19) | Only as one *input* alongside a direct `FlagSet.VisitAll` walk — never as the sole source |
| "Fix" a vacuous guard by adding a bare `> 0` / non-empty check on top of the existing computation | Smallest possible diff, quick to ship | Reintroduces the exact defect class this milestone exists to close, one layer deeper (Pitfall 16) | Never — root-cause the zero/empty condition first |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|--------------|------------------|-------------------|
| tmux (external process, versioned independently of this repo) | Assume DECSET 2026/DECRQM behavior is uniform across tmux versions and OSes | Pin and log the tmux version per CI job; verify mode-2026 handling explicitly rather than assuming 3.4+ everywhere (Pitfall 8) |
| `golang.org/x/tools/go/packages` (already in tree via 4 existing archtests) | Write a fifth independent `packages.Load` call for `internal/query` without reusing the existing pattern's `Tests: true` + count-sanity + positive-control shape | Copy the established structure exactly (Pitfall 4); consider sharing one load across all five archtests if CI time is measured as a real cost (Pitfall 5) |
| Browser custom-protocol handlers (`vscode://`, `cursor://`, `zed://`, `idea://`) | Treat the page's CSP as governing whether the navigation is allowed | Recognize CSP has no jurisdiction over top-level custom-scheme navigation; the real boundary is the browser's native prompt (or its absence, once "always allow" is set) plus server-side path validation (Pitfall 10, 11) |
| `IntersectionObserver` + Svelte 5 `$effect` | Construct the observer inside an effect whose closure also reads the state the observer's callback writes | Construct once with a stable dependency set; write state only from the callback, outside the effect-tracking context (Pitfall 14) |
| Cobra's `doc.GenMarkdownTree` | Use its output as the enumeration source for a drift guard | Enumerate via `FlagSet.VisitAll`/`InheritedFlags()` directly on the fully-wired command tree (Pitfall 19) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Server-side community detection on the full file/package graph, unbounded | Slow RPC responses on large repos; possibly masks/compounds GRF-01's already-measured render wall | Measure clustering time separately from render time against the guava-scale corpus before committing a design (Pitfall 3) | Already measured as real at 3,233 nodes for rendering alone (GRF-01); clustering adds cost on top |
| `packages.Load` invoked independently by 5 separate archtest packages | Archtest CI job wall-clock time grows with each new archtest added | Measure current wall-clock time first; share one load across archtests if the cost is real (Pitfall 5) | Cold-cache CI runners, or as more archtests accumulate over future milestones |
| Dense `IntersectionObserver` threshold arrays / geometry reads inside the callback | Janky scroll on long files, invisible to jsdom | Small deliberate threshold set; no synchronous layout reads inside the callback; verify with a live-browser profiler pass (Pitfall 15) | Long real files (hundreds+ of lines/symbols), not toy fixtures |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Trusting CSP to constrain `vscode://`-style navigation | False sense of a mitigated boundary; the actual risk (arbitrary local file open via a compromised/injected asset) is unmitigated | Recognize the real boundary (browser prompt + server-side path validation); document explicitly as a new trust boundary in the phase's SECURITY.md (Pitfall 10, 11) |
| Accepting a client-supplied path/file value into the editor-link `href` unvalidated | A crafted deep-link parameter could cause the served page to open an arbitrary local path via the user's editor once "always allow" is set | Only the scheme prefix is client-configurable; path/line are always server-derived from the already-indexed, already-validated node/symbol (Pitfall 10) |
| Skipping a per-phase SECURITY.md for BRW-11 because it "just adds a link" | Repeats v0.12.0 Phase 1's exact, already-called-out gap (shipped with `security_enforcement: true` and no `SECURITY.md`, the only such omission in project history) for a phase that genuinely opens a new OS-level side-effect boundary | Write the threat model explicitly — this is new capability (local process launch from a read-only web UI), not a trivial addition |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|--------------|-------------------|
| Editor-link prompts on every click, forever | Users train themselves to click through/dismiss without reading — classic prompt fatigue, and it degrades trust in the whole feature | Accept that "always allow" is the expected steady state and design server-side validation accordingly (Pitfall 11), rather than treating the prompt as a durable safeguard |
| Community-detection clusters visually reshuffle on a trivial one-line edit | Users lose their mental map of the graph after an unrelated change; erodes trust in the "stable, deep-linkable navigation model" v0.12.0 established | Warm-start incremental reclustering from the prior partition where the algorithm supports it (Pitfall 2) |
| Coverage denominator that silently drifts from what was actually indexed | "Why is my file missing" answers a subtly wrong question, defeating the feature's whole purpose | Reuse the indexer's own discovery count, don't reimplement discovery at query time (Pitfall 13) |

## "Looks Done But Isn't" Checklist

- [ ] **tmux e2e harness:** Often missing a stability-check (capture-twice-compare) primitive — verify no test relies on a fixed `sleep` between `send-keys` and `capture-pane` (Pitfall 6)
- [ ] **Community detection:** Often missing a determinism test — verify a test runs the algorithm N≥3 times on identical input and asserts label-canonicalized output equality, not just "returns non-empty" (Pitfall 1)
- [ ] **Editor URI handoff:** Often missing server-side path re-validation — verify the `href`-building code derives path/line only from server-resolved index state, never a raw client parameter (Pitfall 10)
- [ ] **Scroll breadcrumb:** Often missing observer cleanup — verify the `$effect` that constructs the `IntersectionObserver` returns a teardown function, and that a live-browser UAT pass (not just jsdom) exercised it (Pitfall 14, 15)
- [ ] **Discovery/skip-reason surface:** Often missing positive-control tests per reason — verify each reason category has a real fixture file that trips exactly that path, not just a mocked skip event (Pitfall 12)
- [ ] **"Fixed" guards (999.4, T-01-18, dry-run-signed, post-release-verify, tap-secret):** Often missing a RED-then-revert demonstration against the *actual historical failure condition* — verify each fix's PR/commit shows the guard failing on a reproduction of the original bug, not just passing on the current good state (Pitfall 4, 16, 17, 18)
- [ ] **CLI-reference drift guard:** Often missing hidden/inherited/conditional/deprecated flag coverage — verify the guard's own test suite plants one of each and confirms detection (Pitfall 19)

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|----------------|------------------|
| Non-deterministic community detection shipped and later noticed | MEDIUM | Add fixed seed + sorted iteration + canonical relabeling; re-freeze any golden/snapshot fixtures; add the repeated-run determinism test retroactively and demonstrate it would have failed against the pre-fix code (keep the failing run in the PR description, per this project's own evidence-preservation discipline) |
| tmux harness flaking in CI after merge | LOW-MEDIUM | Replace fixed sleeps with poll+stability-check; add version pinning/logging; if unfixable within a reasonable window, gate the job `continue-on-error` temporarily with a tracked todo — do not delete the coverage silently |
| HLT-04 reasons discovered to be inferred/wrong post-ship | MEDIUM-HIGH | Requires pipeline plumbing to capture real reasons at decision time (not a query-time patch) — treat as the "fix that needs work, not time" this repo's own retrospective distinguishes; do not attempt a query-time reason-inference patch as the fix, since that's the original defect restated |
| A "fixed" guard turns out still vacuous | LOW (to detect) / MEDIUM (to actually fix) | Apply the RED-demonstration protocol retroactively: reproduce the original bug condition, confirm the "fixed" guard still passes, then root-cause properly — this is cheap to check and should be a standing post-merge audit step for every item in the guard-hardening set, not just at authoring time |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| 1. Community-detection guard asserts "ran" not "stable" | U (GRF-06) | Repeated-run determinism test, label-canonicalized, N≥3 |
| 2. Cluster staleness/reshuffle across incremental sync | U (GRF-06) | Sync-then-recluster scenario test on a real multi-file corpus with a bounded-diff assertion |
| 3. Clustering compounds GRF-01's rendering wall | U (GRF-06) | Pre-committed threshold + isolated clustering-time benchmark on the guava-scale corpus |
| 4. T-01-18 archtest repeats the one-shot-check mistake | G | RED-then-revert demonstration against a planted forbidden import; positive-control check present |
| 5. `packages.Load` cold-cache CI cost | G | Measured wall-clock time before/after the fifth archtest ships |
| 6. tmux capture-pane races render | T | Poll+stability-check primitive exists and is used by every scenario test, no fixed sleeps |
| 7. DECRQM/mode-2027 probe leakage into captured output | T | Dedicated test asserting zero raw `\e[?` bytes in captured content |
| 8. tmux version/OS drift ubuntu vs macOS | T | Version pinning + logging in CI; both OS families actually exercised, not just gated-and-skipped |
| 9. Alt-screen restore asserted by absence only | T | Raw escape-sequence assertion (`\e[?1049h`/`l`) plus pre-existing-content survival check |
| 10. CSP mistaken for the security boundary on editor links | U (BRW-11) | SECURITY.md names the actual boundary; server-side path validation test exists |
| 11. Prompt fatigue makes "always allow" the steady state | U (BRW-11) | Threat model explicitly designs for silent-launch as the default case, not the exception |
| 12. HLT-04 reasons inferred instead of recorded | U (HLT-04) | Per-reason positive-control fixture test; reason captured at pipeline decision point, not query time |
| 13. Coverage denominator drifts from real discovery | U (HLT-04) | Cross-check test against the indexer's own discovery count on the same corpus |
| 14. `$effect` double-invocation, third occurrence | U (BRW-10) | Live browser UAT pass required as a gate, not optional; observer constructed with stable deps, state written only in callback |
| 15. Layout thrash from breadcrumb updates | U (BRW-10) | Live-browser performance profiling pass on a real long file |
| 16. `CheckRegression` positivity guard "fixed" vacuously | G | Root-cause the zero-reading condition; RED-demonstrate against the historical bug |
| 17. `dry-run-signed` diff guard "fixed" by widening the pattern | G | Positive-control synthetic-removal commit; prefer `--numstat`/`--name-status` over unified-diff text parsing |
| 18. `post-release-verify` conclusion guard and tap-secret test assert fixtures, not reality | G | Conclusion guard tested against ≥2 contrasting event inputs; tap-secret test reads the real workflow/secret reference, not a second hand-typed constant |
| 19. CLI-reference guard inherits Cobra doc-gen's hidden-flag blind spot | D (DOCS-05) | Guard's own suite plants a hidden, an inherited, a conditionally-registered, and a deprecated flag and confirms detection of all four |

## Sources

- `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md` — Current Milestone scope, Constraints, Key Decisions, Current State (v0.12.0 GRF-01 failure/remeasure, schema `Meta`/D-02a discipline) — HIGH confidence, primary source
- `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/RETROSPECTIVE.md` — cross-milestone "gates that could not fail" and "query that could not find" trend analysis, rule `84d1gfpywd`, fix-composition Critical pattern — HIGH confidence, primary source
- `internal/graphstore/archtest/import_graph_test.go`, `internal/cli/present/archtest/import_graph_test.go` (read directly) — established `go/packages`-based archtest pattern this repo already uses, including its positive-control and count-sanity checks — HIGH confidence, direct code inspection
- `internal/uiserver/spa.go` (read directly) — actual served CSP string, confirming `default-src 'self'` etc. has no jurisdiction over custom-scheme navigation — HIGH confidence, direct code inspection
- `internal/indexer/pipeline.go`, `internal/parser/parser.go`, `internal/indexer/goextract/goextract.go` (read directly) — confirms `Stats.Skipped` is currently a bare count with no persisted per-file reason, and that `parser.ErrSourceTooLarge` is the one structured skip cause that exists today — HIGH confidence, direct code inspection
- `internal/schema/graph.proto` (read directly) — confirms field range 50–59 is reserved on `Node`/`Edge` explicitly for "community/cluster assignment," supporting the persist-at-index-time recommendation — HIGH confidence, direct code inspection
- Web search: tmux DECSET 2026/DECRQM support, tmux 3.4+ terminal support (iTerm2/Alacritty/kitty/foot), August-2026 tmux DECRQM implementation for mode 2026 — MEDIUM confidence, current web search, not independently verified against tmux's own changelog
- Web search: Louvain/Leiden non-determinism (node-order and tie-break sensitivity, Leiden's theta parameter, consensus-clustering as a stabilization technique) — MEDIUM confidence, current web search across multiple sources (PuppyGraph, Memgraph docs, academic PDF), broadly consistent
- Web search: Svelte 5 `$effect` double-invocation issues (`sveltejs/svelte` #10505, `sveltejs/kit` #11460) — MEDIUM confidence, GitHub issue tracker, actively-discussed and consistent with this project's own v0.12.0 Phase 6 findings
- Web search: Cobra `doc.GenMarkdownTree` hidden-flag exclusion (`spf13/cobra` issue #1663) — MEDIUM confidence, GitHub issue tracker, directly on-point
- Web search: `golang.org/x/tools/go/packages` usage patterns — LOW-MEDIUM confidence, general package documentation, no direct CI-latency benchmark found; treat the cold-cache latency claim (Pitfall 5) as a reasoned inference from general `go list`/module-resolution behavior, not a measured figure — **flagged for phase-time measurement, not taken as fact**
