# Feature Research

**Domain:** Local developer-tooling UI features (editor handoff, source-scroll context, index-coverage reporting, graph clustering) + TUI/CLI guard-hardening conventions
**Researched:** 2026-09-08
**Confidence:** MEDIUM (cross-checked against multiple comparable tools; two areas — Sourcegraph per-file skip reasons, tmux TUI test harnesses in gh/lazygit/k9s — have thin public documentation and are marked LOW/inferred where noted)

This is a **follow-on milestone** researching seven specific, already-scoped features against comparable tools, not a whole-domain landscape survey. Each section below stands in for one v0.13.0 feature area. Categorization (table stakes / differentiator / anti-feature) is relative to *this* feature's design space, not to the whole product.

## Feature Landscape

### 1. Editor Handoff (BRW-11)

**What comparable tools do:** VS Code's own `vscode://file/{absolute-path}:{line}:{col}` URI handler is the de facto standard other tools link against — GitHub's own "Open in VS Code" links, Confluence deep-links, and countless doc generators all target this exact shape. JetBrains ships the equivalent `jetbrains://<product>/navigate/reference?project=...&path=...`. Cursor and other VS Code forks accept the same `vscode://` scheme by convention (they register the same protocol handler family) or their own analogous `cursor://file/...` scheme. Sourcegraph's "Open in editor" button is configurable per-user precisely because there is no single winning scheme — its settings expose a URL template with `%file`/`%line` placeholders the user fills in for their own editor. This is the load-bearing precedent for BRW-11's own design: a **configurable URI template**, not a hardcoded `vscode://` assumption, because plenty of real users are on JetBrains, Zed, Sublime, or a remote/WSL/container VS Code variant whose URI shape differs (`vscode://vscode-remote/dev-container+.../path`).

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| One-click "open in editor" from a node-detail/source view | Every comparable code browser (Sourcegraph, GitHub.dev, JetBrains built-in "Open in..." action) offers this once verbatim source is shown — BRW-02 already ships the source pane this button attaches to | LOW | Pure client-side link construction from data the UI already has (repo-relative path + line, resolved to an absolute host path already known from the index) |
| Absolute-path resolution against the indexed working tree | The URI scheme needs an absolute filesystem path, not a repo-relative one — a wrong root silently opens nothing or the wrong file | LOW | The indexer already resolves and stores the working-tree root; no new backend surface needed, this is host-side path joining |
| Configurable URI template (not single hardcoded scheme) | Sourcegraph's own settings page proves this is expected once teams have JetBrains/Zed/remote users — a single `vscode://` hardcode is a support-ticket generator on day one | LOW–MEDIUM | Simple `%file`/`%line` (or Go `text/template`-style) substitution over a config value; needs a sane default (`vscode://file/{path}:{line}`) and a settings surface to override it |
| Line **and column** anchor | VS Code and JetBrains both accept a column component; omitting it still works (defaults to col 1) but a caller with a specific symbol's start_col (already in `schema.Node`) can be more precise for wide lines | LOW | `Node.start_col` is already indexed; free to include |

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Remote/WSL/container-aware path rewriting | If codegraph-go is ever run against a repo mounted in a devcontainer or over SSH, `vscode-remote://` needs an authority segment the local filesystem path doesn't carry | MEDIUM | Real differentiator vs. a naive "open file" button, but no evidence yet that codegraph-go users run this way today (single-static-binary, local-first) — defer until requested |
| Per-user (not just per-install) editor preference, remembered client-side | Sourcegraph's version is a personal setting, not an org one, because "which editor do I run" is an individual fact | LOW | `localStorage` is enough; this is a UI nicety on top of the config seam, not a backend requirement |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Server-side shell-out to launch the editor (`exec.Command("code", path)`) | Feels more "direct" than a browser URI handler | Runs arbitrary local process invocation from a server handling browser-originated requests — a real command-injection-adjacent attack surface on a tool whose whole security model (SRV-02, Origin/Host validation) exists specifically to keep the local server from being remotely triggerable. Breaks "read-only by construction." | `<a href="vscode://...">` — the browser's own protocol-handler dispatch, no server execution at all |
| Auto-detecting the user's installed editor via OS process/registry inspection | Would remove the need to configure anything | Platform-specific (registry on Windows, `defaults` on macOS, `.desktop` files on Linux), unreliable, and adds real complexity for a feature whose whole appeal is being simple | A configurable template with a sane VS Code default; let the user override once |

**Dependencies:** Builds directly on BRW-02 (node detail with verbatim source) and BRW-09's `GetPermalink` handler as the closest existing precedent — same shape of problem (turn a path+line into an external URL), same "answer, not error" pattern when a scheme isn't configured. No new Engine surface: this is a pure presentation-layer feature over data BRW-02/ENG-01 already return.

---

### 2. "Where Am I" Breadcrumb (BRW-10)

**What comparable tools do:** Two competing UX patterns exist, and the research is unambiguous about which one wins for *scrolling* context specifically:

- **VS Code's sticky scroll** (`editor.stickyScroll.enabled`) pins the nested scope chain (class → method → block) to the top of the viewport as you scroll, updating live. This is the feature that directly answers "where am I in this file" while scrolling, and it has displaced the older static breadcrumb bar as VS Code's primary answer to this problem.
- **VS Code's breadcrumb bar** (and JetBrains' equivalent structure/context bar, historically at the editor's bottom) is a **static, click-to-navigate** path bar (folder › file › symbol), useful for jump-to but not for continuous scroll feedback — long breadcrumbs truncate and don't reflect the *current* scroll position without extra logic.
- **JetBrains Rider's "Sticky Lines"** feature (its own name for the same idea, shipped after observing VS Code's version) confirms sticky-scroll-style pinned context is now the converged industry answer, not a VS Code-only quirk — multiple independent IDEs arrived at the same pattern.

For codegraph-go's read-only source pane (no editing, no nested-scope tree already built in the UI), the simplest correct implementation is a **single-line breadcrumb showing the nearest containing symbol**, computed from the node's own line-range data the Engine already returns, updated on scroll — closer to a minimal sticky-header than full VS Code sticky scroll (which needs a full scope *stack*, not just the one containing symbol).

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Single-line "you are here" symbol indicator while scrolling a long file | Both major sticky-scroll implementations (VS Code, JetBrains) exist because plain line numbers stop being useful past ~1-2 screens; this is table stakes for any source-reading pane, not a stretch feature | LOW–MEDIUM | Needs a scroll listener + a lookup of "which node's [start_line, end_line] contains the current top-of-viewport line" — this is a **client-side computation over data already fetched** (the file's symbol list — Explore/Files/FileSymbols already return per-file node ranges), no new Engine RPC |
| Click-through from the breadcrumb to the symbol's own node-detail view | VS Code's breadcrumb bar is click-to-navigate; a pure label with no action underuses data the UI already has (BRW-03's neighbor-click pattern already exists) | LOW | Reuses BRW-03's existing "click a neighbor and keep navigating" pattern verbatim |

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Nested scope chain (full sticky-scroll stack, not just one symbol) | True VS Code parity — shows enclosing class AND method, not just the innermost one | MEDIUM–HIGH | Needs a real scope tree (parent/child symbol nesting), which the flat `Node` list doesn't carry today — a bigger lift than the milestone's "contained follow-on" framing intends; the v0.13.0 scope note explicitly says "the containing symbol" (singular), so this is correctly left as a future enhancement, not v1 of this feature |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Server-computed "current symbol" via a new streaming RPC that tracks scroll position | Feels consistent with LIV-01's existing streaming infrastructure | Scroll position is purely a client-side, high-frequency UI event — routing it through the server adds network round-trips to something that must feel instantaneous, and the server has no reason to know where a user's viewport is on a read-only page | Pure client-side computation against symbol ranges already in the payload; zero new backend surface |
| A minimap (VS Code's separate code-overview strip) as part of "where am I" | Superficially related — also a scroll-context aid | Different feature entirely (visual density map, not symbol identity); scope creep relative to BRW-10's stated ask | Not in scope; the breadcrumb alone answers the stated question |

**Dependencies:** Depends on BRW-02's SourcePane (the scrollable source view BRW-10 attaches a header/overlay to) and the per-file symbol-range data the UI already fetches for BRW-04 (jump from reference to definition) and NAV-01. No new Engine method — this is confirmed by the milestone's own framing ("needs one new Engine surface" is stated only for HLT-04, not BRW-10).

---

### 3. Coverage Denominator / "Why Is My File Missing" (HLT-04)

**What comparable tools do:** This is squarely an **indexing-coverage transparency** problem, and the strongest converged pattern across code-intelligence tools is: report the discovered-vs-indexed **denominator**, and attach a **reason** to every file that didn't make it (parse error, excluded by ignore rule, unsupported language, oversized, binary/generated). Sourcegraph's own docs on "common reasons search results don't match" and its auto-indexing failure surfaces exist precisely because silent exclusion is the #1 trust-destroying failure mode in an indexed-code product — a user who can't find something assumes the tool is broken rather than that their file was skipped for a legible reason. `schema.File.errors` already exists in codegraph-go's own schema (`repeated string errors = 6`) specifically for "per-file extraction failures... so one bad file does not abort the whole index run" — this feature is substantially **surfacing data the schema already anticipated**, not inventing a new concept.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Discovered-file count vs. indexed-file count (a real denominator) | HLT-01 already shows "coverage, per-language file counts" but only for files that succeeded — a coverage number with no denominator is not a coverage number, it's just a count. This is the milestone's own stated gap ("Coverage denominator... files discovered but NOT indexed") | MEDIUM | **Needs a new Engine surface** (explicitly flagged in the milestone scope) — today's indexer walk doesn't appear to retain "files seen but excluded" as queryable state distinct from `File.errors` (which only covers files that were *attempted* and failed, not files skipped before an attempt, e.g. by `.gitignore` or unsupported extension) |
| A reason string per skipped file | Sourcegraph's failure-surfacing convention (and this repo's own `File.errors` precedent) both point the same way: a bare count ("47 files not indexed") is nearly useless without "why" per file — the milestone's own framing is literally "a reason per file" | LOW once the denominator surface exists | Reuse the `errors []string` shape already established on `File` for the attempted-but-failed case; for skipped-before-attempt files (ignored, unsupported language, binary), a small closed enum of reasons (`ignored`, `unsupported_language`, `binary`, `oversized`) is cleaner than free text, since these are classifiable at walk time |
| Surfaced in the existing Index Health view (HLT-01/HLT-02), not a separate page | HLT-02 already established the "trust verdict above raw numbers" pattern; the coverage denominator is more raw-numbers detail feeding the same verdict, not a new concern | LOW | Extends `CountTable.svelte` / the health route rather than introducing new navigation |

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Filter/search within the not-indexed list | Useful once the list is long (large monorepos), lets a user check "is *my* file in the skipped set" directly instead of scanning | LOW | Client-side filter over an already-fetched list; no backend cost beyond returning the list |
| Per-reason aggregate counts (e.g. "12 unsupported language, 3 oversized") | Answers "should I care" at a glance before drilling into individual files — mirrors HLT-01's existing per-language breakdown pattern | LOW | Simple client-side `groupBy` over the same payload |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Auto-remediation ("re-index this file now" button per skipped file) | Feels helpful — turn the diagnosis into a one-click fix | The UI is read-only by construction (a v0.12.0 standing constraint — SRV-03); a write action here is a structural violation of the whole product's security/architecture posture, not just scope creep | Show the reason and the remedy as **text** ("unsupported language — add support upstream" / "excluded by .codegraphignore — edit the ignore file"); the fix happens outside the UI, same as every other index-freshness signal today |
| Live per-file progress bar during an active index run | Superficially related to "why is my file missing" but actually a different feature (in-flight progress, not post-hoc coverage reporting) | Adds a new streaming concern to what should be a point-in-time query against `Meta`; LIV-01 already owns "watcher re-index events stream" — duplicating that plumbing for a sub-feature of HLT-04 is unwarranted scope growth | If genuinely wanted later, it's an LIV-0x extension, not part of HLT-04 |

**Dependencies:** This is the one feature in the milestone that explicitly needs new backend surface (per PROJECT.md v2-deferral text: "Needs new Engine surface; deliberately out of v1 scope"). It depends on the indexer's file-walk already distinguishing "discovered" from "indexed" internally (it must, in order to apply ignore rules and language filters at all) — the gap is that this distinction isn't retained as queryable output today. Depends on HLT-01/HLT-02's existing Index Health view as the presentation surface, and on `schema.File.errors`'s existing shape as the precedent for the per-file reason data model.

---

### 4. Community-Detection Clustering in the Graph View (GRF-06)

**What comparable tools do:** Louvain-method community detection (modularity optimization) is the standard, near-universal answer in general-purpose graph tools — Gephi ships it as a first-class Statistics-panel algorithm, and Cytoscape's ecosystem carries multiple community-detection apps built on the same modularity-maximization family. The output is conventionally rendered as a **"Modularity Class" partition attribute**, then visualized by **coloring nodes per cluster** (the standard, near-universal rendering choice) — sometimes additionally by convex-hull or bounding-region overlays per cluster, though color-coding alone is the baseline every tool ships first. codegraph-go's own schema is already forward-positioned for this: `graph.proto`'s `Node`/`Edge` messages both reserve field range `50-59` explicitly annotated "future: embedding vector, community/cluster assignment" (D-03's ARCH-01 annotation slot) — this milestone is the payoff of that reservation, not a fresh design.

For a **file/package**-granularity graph (GRF-02's existing scope — the milestone view is deliberately not whole-symbol), community detection runs over the aggregated file↔file adjacency `Engine.FileGraph()` already computes (ENG-03), producing communities of *files that cluster together by call/import density* — which for many repos will visually echo, but sometimes usefully diverge from, the directory-structural grouping GRF-02 already ships as its v1 substitute. That divergence (files in different directories that are actually tightly coupled) is precisely the signal directory grouping cannot show and is community detection's genuine differentiator here.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Louvain (or comparable modularity-based) clustering computed over the file graph | The converged standard across Gephi/Cytoscape/network-analysis tooling generally; anything hand-rolled and non-standard would be re-deriving well-trodden ground for no benefit | MEDIUM | Needs a Louvain (or similar) implementation over `Engine.FileGraph()`'s adjacency — likely a small, focused pure-Go graph algorithm (no existing dependency in the stack does this; a minimal, dependency-light Louvain implementation is the standard scope for this size of graph) |
| Per-cluster node coloring in the existing GraphCanvas view | This is the baseline rendering convention (Gephi's own default workflow), and GRF-05's existing seam ("rendering library sits behind a narrow component seam") already anticipates swapping/extending visual encodings without an architecture change | LOW–MEDIUM | Extends `graph-style.ts`/`GraphCanvas.svelte`'s existing styling seam with a cluster-id → color mapping; the annotation slot in the schema (or a computed-on-demand result, following ENG-03's own "no precomputed projection" discipline) supplies the cluster id per file |
| Toggle between directory-structural grouping (existing GRF-02 default) and community-detection clustering | Neither view subsumes the other — directory grouping is ground-truth/predictable, community detection is discovered/sometimes-surprising — a toggle lets a user compare rather than forcing one lens | LOW | UI-only state toggle over two coloring/grouping strategies already computed |

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Cluster-boundary visual grouping (convex hulls / bounding boxes), not just node color | Stronger visual legibility for "this is one community" than color alone, especially at higher node counts | MEDIUM–HIGH | Real differentiator over the color-only baseline, but adds layout-geometry work; defer past the first cut given GRF-01's hard-won lesson that this exact view already failed to converge once at scale (guava, 3,233 nodes) — anything that adds rendering cost needs the same measure-before-commit discipline GRF-01 established |
| Clicking a cluster to filter/highlight only its members | Useful drill-down once clusters exist as a first-class grouping, mirrors GRF-03's existing drill-into-a-file pattern | LOW–MEDIUM | Straightforward extension of existing click-to-filter interaction patterns already in the graph view |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Persisted/stored cluster assignments written back into the index (using the reserved schema field as a **write** target) | The schema reservation makes "just store it" look natural | Violates SRV-03 (read-only by construction) and ENG-03's own "computed fresh per call, no precomputed projection" discipline that `FileGraph()` already established — writing derived analytics into the graph store also reopens the schema-versioning discipline (D-02a) for a feature that doesn't need persistence to be useful | Compute clusters on-demand per graph-view request, same discipline as `FileGraph()` itself; the reserved field range remains available for a genuinely future team-scale/precomputed use case, not spent here |
| Force-directed whole-graph layout to "let clusters emerge visually" instead of algorithmic clustering + hierarchical layout | Sounds like a simpler way to get the same visual effect | This is the single most-flagged anti-pattern in the whole milestone's own prior research (v0.12.0's REQUIREMENTS.md Out-of-Scope table names dependency-cruiser's FAQ, the "Trimming the Hairball" MSR paper, and SourceTrail's v1→v2 retreat from force-directed as the evidence) — GRF-02 already deliberately rejected this once; GRF-06 must not reintroduce it as a side door | Keep GRF-02's existing hierarchical/directory layout as the base; overlay community membership as color/grouping metadata on that same stable layout, never switch the layout algorithm itself |

**Dependencies:** Directly builds on ENG-03 (`Engine.FileGraph()`, shipped v0.12.0 Phase 5) as its sole data input, and on GRF-02/GRF-05's existing graph view and rendering-seam as the presentation surface. Depends on the schema's reserved `50-59` field range as *prior art for the intended shape* of a cluster assignment, even if v1 computes it fresh rather than persisting it (see anti-feature above). Must respect GRF-01's measured-render-threshold lesson: any added visual complexity (hull overlays, per-cluster styling) needs the same "measure before committing a threshold" discipline the milestone's own history established, given the file-graph view has already failed to converge once at real scale.

---

### 5. tmux Real-PTY E2E Harness for the TUI (999.2)

**What comparable tools do:** Public documentation of exactly this pattern (spawning a release TUI binary inside tmux, driving it with `send-keys`, asserting on `capture-pane`) is thin for gh/lazygit/k9s/bubbletea specifically — this section is the one area of this research graded closer to LOW/inferred confidence, reconstructed from tmux's own scripting primitives and this repo's own prior incident record rather than a documented "lazygit does X" citation. What the tooling ecosystem *does* converge on, and what the milestone's own goal text already names precisely, is the specific assertion classes such a harness exists to make — because they are exactly the class of defect a piped/headless test cannot observe:

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Alt-screen entry/exit assertion | bubbletea (and Charm's ecosystem generally) offers `tea.WithAltScreen()`; whether a given picker actually enters and — critically — **restores** the primary screen buffer on quit is exactly the kind of state that only a real terminal (not a pipe) can expose. This repo's own G-07-2 incident (both pickers rendering inline without alt-screen, causing flicker) is the direct motivating precedent | MEDIUM | `capture-pane -e` (or comparing scrollback before/after) to confirm no residual escape sequences and that the main buffer is restored |
| Escape-sequence hygiene (no leaked terminal capability-probe responses) | This repo's own G-07-1 incident — bare `daemon` on an empty registry leaked raw `DECRQM` responses (`^[[?2026;2$y...`) onto a live TTY — is the textbook example of a defect class that is invisible to any non-PTY test, because a pipe never answers capability queries the way a real terminal does | MEDIUM–HIGH | Requires tmux (a real PTY, not `script`/`expect` alone) so the terminal genuinely answers DA/DECRQM-style queries the way a real terminal would, exposing whether the app leaks the raw reply into visible output |
| Keyboard interaction correctness (`space` toggles, `q`/`esc` cancels) | Standard TUI interaction-testing surface for any checkbox/picker component — this is the functional-correctness rung, distinct from the visual-hygiene rungs above | LOW–MEDIUM | `send-keys` + `capture-pane` diffing before/after each keypress; the milestone's own scope text already enumerates this exact assertion (`[x]`/`[ ]` glyphs, zero config writes on cancel) |
| A flicker proxy (stable-frame comparison across N captures) | Flicker is inherently a *rendering* property, not a state-machine property — bubbletea's own unit-testable `tea.Msg` path (used by the piped suite today) provably cannot detect it, since it never renders at all | MEDIUM | Repeated `capture-pane` at short intervals with no intervening input, asserting byte-identical (or near-identical, allowing for a legitimate spinner/clock) frames — a genuinely new assertion class this repo's test suite doesn't have today |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Screenshot/pixel-diff-based visual regression testing of the terminal | Feels like the most "real" form of visual testing | Terminal rendering varies by font, terminal emulator, and OS-level rendering — a pixel diff is far noisier and far more expensive than text-based `capture-pane` diffing, and answers a question ("does this look identical") the milestone doesn't need answered when "does this contain the right characters and no leaked escapes" already suffices | Text-mode `capture-pane` assertions, exactly as the milestone scope already specifies |
| Running the full suite unconditionally in every CI environment | Consistency ("always run the same tests everywhere") sounds appealing | tmux availability is not guaranteed on every CI runner or contributor machine, and a PTY-dependent suite that hard-fails where tmux is absent turns an optional depth check into a broken build for unrelated contributors | Exactly what the milestone already specifies: gated behind a build tag / CI job with tmux available, skipping cleanly elsewhere — the correct, already-decided answer |
| Asserting exact byte-for-byte frame content including timing-sensitive spinner glyphs | Maximizes strictness | Produces exactly the "guard that cannot fire" failure mode this milestone is otherwise dedicated to eliminating in the *other* direction — an assertion so brittle it needs constant babysitting either gets weakened into vacuity or becomes a source of unrelated CI flakes | Structural assertions (does it contain expected static text, are escape sequences absent, are N consecutive frames stable) rather than full-frame byte equality |

**Dependencies:** This item is explicitly sequenced **before** the UI follow-through work in the milestone's own plan ("lands before the UI work so that work has a real-terminal rung"), and depends on the existing bubbletea TUI surfaces from v1.0 (daemon picker, install/uninstall checkbox picker) as its subjects — it is a test-harness addition, not a product-feature addition, so it has no forward dependents inside this milestone beyond providing infrastructure. It is the direct, deliberate closure of the gap the v1.0 Phase 7 human-UAT incident (G-07-1/G-07-2) first exposed: two real user-visible bugs that both the full piped automated suite and a deep multi-agent code review missed.

---

### 6. Self-Authored CLI Reference with a Drift Guard (DOCS-05)

**What comparable tools do:** Cobra — the CLI framework this project already uses — ships a first-party `doc` package (`cobra/doc`) specifically to **generate** Markdown/Man/ReST/YAML reference docs directly from the live command tree (`cmd.Long`, `cmd.Example`, registered flags), and this is the convention every major Cobra-based CLI (kubectl, Hugo, GitHub CLI, Helm) relies on rather than hand-authoring reference prose that can drift from the actual flag set. The framework's own guidance is explicit: keep `Long`/`Example` strings good, make the root command traversable, and (notably) set `root.DisableAutoGenTag = true` for stable, reproducible output with no timestamp footer — directly relevant to making generated docs diffable/committable rather than perpetually "changed" on every regeneration.

The milestone's framing ("self-authored... with its own drift guard asserting every registered flag is documented") is deliberately choosing **hand-authored prose with a completeness guard** over **fully mechanical generation** — a reasonable middle path given this repo's prior experience: `docs/FLAG-PARITY.md` (deleted in v0.11.0) and its drift guard existed for a different purpose (parity checking against another implementation, now retired as a constraint), and DOCS-05 was explicitly declined at v0.11.0 as "separate work from deleting the comparison matrix" — this milestone is that deferred work, now scoped as genuinely new authored content rather than parity-checking.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Every registered command and flag documented | Table stakes for any CLI reference — the entire value proposition of a reference doc collapses if it's incomplete, and Cobra's own `doc` package exists because hand-tracking this by memory doesn't scale | LOW–MEDIUM | The drift guard (see below) is what makes "every flag" a provable claim rather than an aspiration |
| A drift guard asserting completeness against the live command tree | This is the milestone's own explicit requirement, and matches this repo's own established rule (`84d1gfpywd`) that a guard must be non-vacuous and demonstrated RED before being trusted — the same discipline `docs/FLAG-PARITY.md`'s guard already established as this repo's convention, now pointed at authored content instead of a parity target | LOW–MEDIUM | Walk the live `*cobra.Command` tree (root + all registered subcommands/flags via `cmd.Commands()`/`cmd.Flags()`), assert every name appears in `docs/CLI-REFERENCE.md`; this is a straightforward reflective walk since Cobra exposes its full tree programmatically — the same introspection `cobra/doc`'s own generator uses internally |
| Examples per command, not just a flag table | Cobra's own authoring guidance ("good Long and Example strings") and every comparable CLI reference (kubectl, gh) treat runnable examples as expected content, not optional polish | LOW | Authored content, not generated; no new tooling required beyond the drift guard covering presence, not prose quality |

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Auto-generated (not hand-authored) reference via `cobra/doc`, replacing hand-authored prose entirely | Zero drift by construction — the doc *is* mechanically derived, so there's nothing to guard against going stale | LOW (tooling exists off-the-shelf) | Explicitly **not** what the milestone scoped ("self-authored... with its own drift guard" implies hand-authored content that a guard checks, not generated content that cannot drift) — noted here because it is the *lower-effort* alternative a future maintainer might reasonably ask "why didn't we just do this"; the answer is that self-authored prose can carry narrative/decision context (why a flag exists, when to use it) that mechanical generation from `Long`/`Example` strings alone cannot, which is presumably the actual reason DOCS-05 is scoped as authored+guarded rather than generated |

**Anti-features:**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Reviving `docs/FLAG-PARITY.md`'s deleted guard/shape for this new purpose | It's the closest prior art in the repo, tempting to resurrect rather than rewrite | Per this project's own binding rule on planning/tool-owned artifacts (and the general principle that a retired parity concept shouldn't quietly resurface): FLAG-PARITY existed to check parity against another implementation, a constraint v0.11.0 explicitly retired — reusing its shape for a same-project completeness check conflates two different guard purposes and risks smuggling "comparison framing" back into a codebase that had a whole milestone dedicated to removing it | A fresh, purpose-built completeness guard walking the live Cobra tree against `docs/CLI-REFERENCE.md`, named and framed for what it actually checks |
| A guard that only checks flag **names** appear somewhere in the doc file (substring match) | Cheapest possible implementation | Exactly the "guard that cannot fire" shape this milestone is otherwise explicitly hunting down elsewhere (999.4, T-01-18, the awk-anchor guard) — a substring match passes if the flag name merely appears in a code example or an unrelated sentence, never verifying it's actually *documented* in a real reference entry | Structure the doc with a parseable per-command/per-flag heading convention the guard can verify by structure, not raw substring presence; demonstrate the guard RED against a deliberately-undocumented new flag before trusting it green |

**Dependencies:** Depends on nothing new architecturally — it walks the already-registered `cobra.Command` tree this project's CLI already builds. It is explicitly framed by the milestone as replacing what `docs/FLAG-PARITY.md` used to carry, so it has a direct lineage dependency on that file's deletion (v0.11.0) and the DOCS-05 deferral recorded there. It shares this milestone's own guard-hardening discipline (rule `84d1gfpywd`: demonstrate RED before green) with the five guards-that-cannot-fire items and the tmux harness — all six are instances of the same underlying pattern applied to different subjects.

---

### 7. Guard Hardening (Making Guards Self-Evidencing)

This is a cross-cutting methodology question rather than a single feature, and it is the milestone's own named recurring defect shape: **an assertion that is true but non-discriminating** — it passes whether the property being checked holds or not. The milestone names five concrete instances (`CheckRegression`'s missing current-metrics positivity check, the T-01-18 archtest that was a one-shot check never committed, `release:dry-run-signed`'s awk-anchor diff guard, `post-release-verify.yml`'s unasserted conclusion guard, and a tap-secret test comparing two in-test constants). What mature projects converge on to make such guards self-evidencing:

| Practice | Why It Works | Complexity | Notes |
|---------|--------------|------------|-------|
| **Mutation testing / deliberate RED demonstration** before trusting a guard green | This is the exact discipline mutation-testing tooling formalizes (CircleCI's own framing: "if a mutant is introduced and functionality changes, the tests should find the bug" — a suite that survives an injected fault is proven not to be checking that fault) and is already this repo's own standing rule (`84d1gfpywd`), independently re-derived and applied repeatedly across prior milestones (the golden-suite re-freeze in v0.11.0, the drift guards in v0.12.0) | LOW–MEDIUM per guard | Doesn't require adopting a mutation-testing *tool* — a hand-authored "break the thing on purpose, confirm the guard catches it, revert" commit pair (this repo's own `*-MUTATION-LOG.md` convention) is the lighter-weight, already-proven-in-this-repo version of the same idea |
| **Positive control before negative control** | A guard that has never been observed to fail is unproven — the "vacuity detector that reports a clean tree proves nothing" framing directly matches this milestone's own diagnosis (the same threat surviving three green guards in v0.12.0 Phase 5) | LOW | Every guard fix in this milestone should ship with evidence of both states: fails on the known-bad input, passes on the known-good one — not just "now it passes" |
| **Assert on the thing that actually varies, not a proxy that happens to correlate today** | `CheckRegression`'s gap is exactly this: it validates *baseline* positivity but not *current*, so a degenerate current reading (zero `PeakRSSBytes`) silently reads as "no regression" — the check was validating the wrong operand | LOW–MEDIUM per instance | Case-by-case: identify which value the guard is actually supposed to constrain and confirm the assertion touches that value, not an adjacent one that happened to be non-degenerate in every test run to date |
| **In-test constants must trace to the real artifact, not to each other** | The tap-secret test "compares two in-test constants and reads no workflow" is the textbook version of a tautology — it can never fail regardless of what the actual GitHub Actions secret configuration is, because both sides of the comparison live in the test file itself | LOW–MEDIUM | The fix is structural: read the actual workflow/config file (or a value derived from it) on at least one side of the comparison, so a real drift between test expectation and reality is representable at all |
| **An anchor/pattern guard must be demonstrated against a moving target, not just against today's file shape** | The `release:dry-run-signed` awk-anchor guard "passes vacuously when its awk anchor stops matching" — an anchor pattern silently no longer matching is functionally identical to deleting the check, but looks identical to "nothing to report" | LOW–MEDIUM | Add an explicit "the anchor matched at least once" assertion, separate from "what it matched was acceptable" — two assertions, not one, so a silently-broken anchor fails loudly instead of reading as a clean pass |

**Anti-features (what mature projects explicitly avoid when hardening guards):**

| Practice | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Full mutation-testing framework adoption (e.g., `go-mutesting`-class tooling) project-wide, as the general answer to "guards might be vacuous" | Feels like the systematic, tool-backed answer to a systematic problem | Heavyweight for five specifically-diagnosed instances with already-known root causes; mutation-testing tools also have their own noise/cost profile (long runtimes, many equivalent-mutant false positives) that this repo's lighter-weight hand-authored RED-demonstration convention already avoids, and which it has used successfully at this exact repo's scale multiple times before | Continue the repo's own established, cheaper convention: a deliberate mutation commit + revert pair per fixed guard, logged in a `*-MUTATION-LOG.md`, exactly as prior milestones already did |
| Treating "the test suite is green" as sufficient evidence a guard was fixed | Default assumption in most projects | This is precisely the failure mode this repo's own `planning-artifacts` discipline explicitly warns against ("Validator-green is not evidence... Verify the property you actually care about") and the milestone's own stated intent ("Each demonstrated RED before it is called fixed") | Every guard-hardening item needs its own explicit RED-then-GREEN demonstration recorded, not just a final green run |

**Dependencies:** This methodology applies uniformly across all five named guard instances plus the tmux harness (999.2) and the CLI-reference drift guard (DOCS-05) — it is the connective discipline binding the milestone's entire "Guards that cannot fire" theme together, not a feature with its own build dependency. It depends on nothing technically; it depends on discipline being applied consistently to each of the concretely-named instances.

---

## Feature Dependencies

```
[BRW-02 verbatim source + node detail]  (shipped v0.12.0)
    └──enables──> [BRW-11 Editor Handoff]        (pure client-side link over existing data)
    └──enables──> [BRW-10 "Where Am I" Breadcrumb]  (client-side scroll computation over existing per-file symbol ranges)

[BRW-09 GetPermalink]  (shipped v0.12.0)
    └──precedent-for──> [BRW-11 Editor Handoff]   (same "path+line → external URL, answer not error" shape)

[HLT-01/HLT-02 Index Health view]  (shipped v0.12.0)
    └──requires new Engine surface──> [HLT-04 Coverage Denominator]
                                          └──reuses shape of──> [schema.File.errors]  (existing, per-file failure reasons)

[ENG-03 Engine.FileGraph()]  (shipped v0.12.0)
    └──required-by──> [GRF-06 Community Clustering]
[GRF-01/GRF-02/GRF-05 Graph View + render seam]  (shipped v0.12.0)
    └──required-by──> [GRF-06 Community Clustering]
[schema.proto reserved 50-59]  (shipped, D-03)
    └──prior-art-for, NOT write-target-of──> [GRF-06 Community Clustering]  (computed fresh, not persisted — see anti-feature)

[v1.0 bubbletea TUI: daemon picker, install/uninstall picker]  (shipped v1.0)
    └──subject-of──> [999.2 tmux Real-PTY Harness]
[999.2 tmux Real-PTY Harness]
    └──sequenced-before, provides real-terminal rung for──> [UI Follow-Through work generally]
        (milestone's own stated ordering: harness lands before the UI items)

[Cobra command tree (existing CLI)]
    └──introspected-by──> [DOCS-05 CLI Reference drift guard]
[docs/FLAG-PARITY.md deletion]  (v0.11.0)
    └──deferred-into──> [DOCS-05 CLI Reference]  (explicitly named successor, not a revival of the old shape)

[Guard-hardening methodology (RED-before-GREEN, positive control)]
    └──applies-to (not "required by")──> [all 5 named guard instances, 999.2, DOCS-05's own guard]
```

### Dependency Notes

- **BRW-11 and BRW-10 both require zero new Engine/backend surface** — both are confirmed as pure presentation-layer additions over data the Engine already returns via BRW-02/ENG-01/ENG-02 and the existing per-file symbol data used by BRW-04/NAV-01. This is the correct reading of the milestone's own "Ordered by increasing scope" framing (BRW-11 and BRW-10 listed first, before HLT-04 and GRF-06, which do need new surface or a new algorithm).
- **HLT-04 is the one browse/health item requiring new Engine surface** — explicitly stated in both PROJECT.md's v2-deferral text and the milestone's own target-features list ("which needs one new Engine surface"). It should be planned and estimated accordingly, closer to ENG-03's original weight than to BRW-10/BRW-11's.
- **GRF-06 depends on two already-shipped v0.12.0 items (ENG-03, the graph render seam) plus a genuinely new piece of algorithmic work (community detection itself, likely Louvain)** that has no existing dependency in the stack — this is new build, not just wiring, even though the schema and data source are both already in place.
- **999.2 (tmux harness) conflicts with, in the ordering sense, nothing** — it has no feature dependents inside this milestone other than providing test infrastructure ahead of the UI work, matching the milestone's own explicit sequencing rationale.
- **Guard hardening (theme 1) and DOCS-05's drift guard share one discipline but are otherwise independent** — none of the five named guard fixes block each other or any UI follow-through item; they can be sequenced by convenience/risk rather than dependency.

## MVP Definition

Not applicable in the traditional sense — this is a scoped follow-on milestone with all seven items already committed to `v0.13.0`'s target-features list, not a from-scratch product needing MVP triage. The relevant prioritization signal from this research is **complexity ordering within the milestone**, which corroborates the milestone's own "ordered by increasing scope" framing for the UI follow-through set:

### Lower complexity, no new backend surface
- BRW-11 (Editor Handoff) — client-side link construction + a config seam
- BRW-10 (Breadcrumb) — client-side scroll computation over existing data

### Medium complexity, requires new backend/algorithmic work
- HLT-04 (Coverage Denominator) — new Engine surface for discovered-vs-indexed
- GRF-06 (Community Clustering) — new Louvain-class algorithm over existing `FileGraph()` data

### Infrastructure, sequenced first per the milestone's own stated ordering
- 999.2 (tmux Real-PTY Harness) — no product feature, but explicitly gates the UI follow-through's own testing rigor
- Guard-hardening theme (five instances) + DOCS-05's drift guard — independent of each other, share only methodology

## Feature Prioritization Matrix

| Feature | User/Maintainer Value | Implementation Cost | Notes |
|---------|------------------------|----------------------|-------|
| BRW-11 Editor Handoff | HIGH | LOW | Highest value-to-cost ratio in the UI set; directly closes a "now what" gap after every node-detail view |
| BRW-10 Breadcrumb | MEDIUM | LOW | Real but secondary UX polish on long files; not needed for small/medium files |
| HLT-04 Coverage Denominator | HIGH | MEDIUM | Directly answers a trust-destroying silent-failure question; the new Engine surface is the main cost driver |
| GRF-06 Community Clustering | MEDIUM | MEDIUM–HIGH | Genuinely novel algorithmic work; value is real but more exploratory than the other three, and must respect GRF-01's measured-threshold lesson before adding render cost |
| 999.2 tmux Harness | HIGH (defect-prevention) | MEDIUM–HIGH | Closes a real, previously-exploited gap (G-07-1/G-07-2); cost is mostly in the escape-sequence/flicker assertion classes, not the send-keys mechanics |
| Guards that cannot fire (5 instances) | HIGH (correctness of CI itself) | LOW–MEDIUM each | Individually cheap; value is disproportionate because each is a currently-false sense of security |
| DOCS-05 CLI Reference + drift guard | MEDIUM | LOW–MEDIUM | Mostly authoring effort; the guard itself is a straightforward Cobra-tree walk |

## Sources

- VS Code `vscode://file/{path}:{line}:{col}` URI scheme and remote/container variants — web, LOW confidence (community/blog sources, not a single canonical spec page; cross-checked across multiple independent write-ups converging on the same URI shape, raising it to workable MEDIUM for the shape itself)
- Sourcegraph "Open in editor" configurable URL-template setting — web, MEDIUM confidence (product behavior widely and consistently described, though not pulled from a single first-party settings-reference page in this pass)
- VS Code `editor.stickyScroll.enabled` and JetBrains Rider "Sticky Lines" — web, MEDIUM confidence (cross-checked across VS Code blog/dev.to coverage and JetBrains' own YouTrack/docs entries, independently converging on the same pattern from two unrelated vendors)
- Sourcegraph per-file indexing-failure documentation — web, LOW confidence (could not locate a page specifically enumerating per-file skip reasons; the coverage-denominator design here is grounded more in this repo's own existing `schema.File.errors` precedent than in a confirmed external Sourcegraph feature)
- Gephi Louvain/modularity community detection, Cytoscape community-detection app ecosystem — web, MEDIUM confidence (Gephi's own documented Statistics-panel workflow is well and consistently described across multiple independent sources)
- Cobra `doc` package (Markdown/Man/ReST generation from the live command tree), used by kubectl/Hugo/GitHub CLI/Helm — web, MEDIUM confidence (official `cobra.dev` documentation, directly authoritative on this point)
- lazygit/k9s/gh-cli tmux-driven TUI e2e testing conventions — web, LOW confidence (no single authoritative source found describing a documented test-harness pattern for these specific tools; the assertion-class analysis in this document is reconstructed from tmux's own scripting primitives and this repo's own G-07-1/G-07-2 incident record rather than an external documented convention)
- Mutation testing / vacuous-guard detection conventions (CircleCI explainer, general mutation-testing literature) — web, MEDIUM confidence (consistent, well-established industry framing; cross-checked against this repo's own already-independently-derived `84d1gfpywd` rule, which converges on the same practice)
- `internal/schema/graph.proto`, `internal/uiserver/permalink.go`, `.planning/ROADMAP.md` Backlog 999.2/999.4, `.planning/milestones/v0.12.0-REQUIREMENTS.md` v2 section, `web/src/lib/components/{browse,graph,health}/*` — codebase, HIGH confidence (primary source, read directly from this repository)

---
*Feature research for: CodeGraph Go v0.13.0 — Guard Hardening & UI Follow-through*
*Researched: 2026-09-08*
