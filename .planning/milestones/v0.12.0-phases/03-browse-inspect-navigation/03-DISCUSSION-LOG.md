# Phase 3: Browse, Inspect & Navigation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-28
**Phase:** 3-Browse, Inspect & Navigation
**Areas discussed:** Confinement/error states (SRV-05, NAV-04), GitHub permalink plumbing (BRW-09), URL state grammar (NAV-01/02), Search surface (BRW-01/BRW-08), Source rendering & click-to-def (BRW-04/05/06)

**Mode:** default interactive; `workflow.research_before_questions = true`
(web research run for the permalink, URL-state, search-UX and highlighter areas).
All five offered areas were selected.

---

## Confinement: refuse vs. degrade (SRV-05, NAV-04)

### Q1 — New endpoint, or bind SRV-05 to the existing GetNodeDetail?

| Option | Description | Selected |
|--------|-------------|----------|
| No new endpoint — test GetNodeDetail | Treat "the new endpoint" as describing what was new at roadmap-writing time; add a uiserver regression test only | |
| Add a dedicated GetSource RPC | Matches criterion 5's wording exactly, but risks the duplicate confinement implementation SRV-05 warns against | |
| Decide from what the views actually need | Enumerate every source fetch Browse performs; add an endpoint only if a call site is poorly served | ✓ |

**User's choice:** Decide from what the views actually need.
**Notes:** Driven by a tree finding — `GetNodeDetail(file: "../outside.txt")` is
*already* refused (`internal/query/errors_test.go:139`), and the `file` field is
the only client-steerable path input on the service. The criterion-5 wording
tension is recorded explicitly in CONTEXT.md so the planner cannot silently pick
either reading.

### Q2 — Source pane behaviour when a single-def node's source can't be read

| Option | Description | Selected |
|--------|-------------|----------|
| Split out stale only | Keep CR-02's collapse for the two unactionable causes; use the on-the-wire `GetStatusResponse.stale` for the actionable one | ✓ |
| Add a reason enum to SourceBlob | Most precise, but permanent under D-02a and directly contradicts CR-02's recorded reasoning | |
| Keep the collapse as-is | Simplest, fully consistent with CR-02, but a stale-index user sees a blank pane with no hint | |

**User's choice:** Split out stale only.
**Notes:** CR-02 collapses three causes to nil. Its reasoning ("a taxonomy of
read errors it cannot act on") holds for two of them but not for *file removed
since indexing* — the one case a user can act on, and the one NAV-04 names by
hand. Resolved with **no proto change**: `stale` is field 6 already.

### Q3 — Where should NAV-04 state handling live?

| Option | Description | Selected |
|--------|-------------|----------|
| Shared layer: status gate + error mapper | Two shared modules in `web/src/lib/`, reused by Phases 4-6 | ✓ |
| Handle inside the Browse view only | Avoids abstracting against one use case, but the local implementation becomes the de-facto contract | |
| Shared error mapper only, no status gate | More flexible per-view, but nothing guarantees the actionable stale case is surfaced | |

**User's choice:** Shared layer: status gate + error mapper.
**Notes:** Motivated by an asymmetry in `degrade.go` — `GetStatus` is the only
RPC that answers when degraded; every other handler *throws*
(`CodeUnavailable` + typed `IndexingInProgress`). So NAV-04's three states arrive
by two different mechanisms, and a client handling only empty responses renders a
blank pane in exactly the two cases NAV-04 exists to prevent.

### Q4 — How fresh should the status gate be?

| Option | Description | Selected |
|--------|-------------|----------|
| Fetch on load + on navigation | No timer; a deliberately small seam for Phase 6 to replace | ✓ |
| Add a periodic poll | Closer to live-push feel, but throwaway work and drifts toward a Phase 6 anti-feature | |
| On load + manual refresh control | No background work, explicit user control, but likely made redundant by Phase 6 | |

**User's choice:** Fetch on load + on navigation.

---

## GitHub permalink plumbing (BRW-09)

**Research applied:** forge URL shapes are not interchangeable (GitHub `/blob/`,
GitLab `/-/blob/`, Bitbucket `/src/`, Codeberg `/src/commit/`); remote URLs come
in three shapes needing normalization plus `url.<base>.insteadOf` rewrites; and —
the sharp one, called out by `vscode-gitweblinks` and `git-link.nvim` — a
commit-pinned permalink **404s when the commit was never pushed**.

### Q1 — What goes on the wire?

| Option | Description | Selected |
|--------|-------------|----------|
| Server sends a ready-made URL template | One additive string field; client substitutes | |
| Send structured host/owner/repo + forge kind | Client assembles; moves forge knowledge into TypeScript | |
| Send the raw remote URL only | Smallest, but genuinely insufficient — cannot tell the client which URL shape to emit | |
| **Other (free text)** | *"hmm, why not have the client send the details to the server, so that it can return the right url?"* | ✓ |

**User's choice:** Free-text — a `GetPermalink(path, line)` RPC.
**Notes:** Maintainer redirected away from all three offered shapes. Analysed and
adopted: it is a strict superset of the template option, and the **only** one
able to express the unpushed-commit case, since a template is a pure formatter.
Confirmed the plumbing precedent exists (`internal/gitmeta/`, and
`internal/indexer/commit.go:60-68`'s git-may-be-absent degradation contract).
Two caveats surfaced before adopting: the pushed check is sound in one direction
only, and cost is about *where* it is called, not the loopback round trip.

### Q2 — Is the unpushed-commit hint in scope?

**User's choice (free text):** "pushed in" — hint is in scope.
**Notes:** Recorded as **three-valued** `availability`
(linkable / linkable-but-possibly-unpushed / no-link), never a boolean, because
`git branch -r --contains` proves a positive but cannot prove a negative without
fetching — which a read-only tool must not do. A boolean would cry wolf on any
repo that hasn't fetched recently.

### Q3 — How many forges?

| Option | Description | Selected |
|--------|-------------|----------|
| GitHub only, others report no-link | Honors BRW-09's literal text; extension point is a switch arm plus a test | ✓ |
| GitHub + GitLab | Small diff, but self-hosted GitLab needs a config knob this project has no mechanism for | |
| Forge-agnostic table from the start | Most useful, clearly beyond BRW-09's text, and every extra shape can rot untested | |

**User's choice:** GitHub only, others report no-link.
**Notes:** Turns on Phase 1 D-08 having explicitly declined a config file — there
is nowhere to put a self-hosted host mapping.

### Q4 — Line range or single line?

| Option | Description | Selected |
|--------|-------------|----------|
| Range when known, single line otherwise | Uses `Node.end_line` (field 8) when opened; `Location` has start only | ✓ |
| Always a single line | One code path, but a 200-line function permalinks to its signature | |
| Range everywhere — add end_line to Location | Consistent, but spends a permanent field on five RPCs' shared message | |

**User's choice:** Range when known, single line otherwise.

---

## URL state grammar (NAV-01/NAV-02)

**Research applied:** SvelteKit exposes URL state via `page.url.searchParams` and
mutates it with `goto(url, {replaceState})` plus `pushState`/`replaceState` from
`$app/navigation` for shallow routing; community consensus is to treat the URL as
the single source of truth rather than mirroring into a store, so deep links and
history stay correct by construction.

### Q1 — How to encode target/depth/limit?

| Option | Description | Selected |
|--------|-------------|----------|
| Query params mirroring the RPC request | `?symbol=&file=&line=&depth=&limit=` — URL→RPC mapping is identity | ✓ |
| Target in the path, params in the query | Prettier, but re-opens the dotted-path ambiguity Phase 2's D-10 closed | |
| Single opaque state param | Trivially extensible, but the URL stops being readable or hand-editable | |

**User's choice:** Query params mirroring the RPC request.
**Notes:** View is *already* a path segment from Phase 2's D-18, so only
target/depth/limit remained. Phase 2's D-10 rejected the dot-heuristic
specifically anticipating this phase's deep links.

### Q2 — What earns a history entry?

| Option | Description | Selected |
|--------|-------------|----------|
| Navigation pushes, refinement replaces | Opening/clicking pushes; typing and depth/limit replace | ✓ |
| Everything pushes | Consistent, but back becomes unusable — the thing NAV-02 asks for | |
| Only view changes push | Predictable, but breaks "click through five callers, return to the third" | |

**User's choice:** Navigation pushes, refinement replaces.

### Q3 — Who validates a hand-edited URL?

| Option | Description | Selected |
|--------|-------------|----------|
| Parse shape, pass values through | Server's existing bounds refuse; error mapper renders it | ✓ |
| Clamp client-side | User never sees an error, but duplicates bounds in TS and silently alters a shared link | |
| Reject unknown params strictly | Catches typos, but breaks forward compat with later phases' params | |

**User's choice:** Parse shape, pass values through.
**Notes:** Matches the discipline stated on nearly every request message in
`ui.proto` — "no second copy of that rule that could drift from the Engine's own."

### Q4 — How much grammar should Phase 3 build?

| Option | Description | Selected |
|--------|-------------|----------|
| One shared parser, params Phase 3 needs | Settles the shape without speculating about unwritten Phase 4/5 params | ✓ |
| Design the full grammar now for all four views | Maximum consistency, but speculative against unwritten requirements | |
| Keep it local to /browse | Avoids premature abstraction, but this is the one area the roadmap says settles here | |

**User's choice:** One shared parser, params Phase 3 needs.

---

## Search surface (BRW-01/BRW-08)

**Research applied:** debounce 150-300ms (lower when data is local), 2-3 character
minimum, and `AbortController` cancellation — the last being essential and
independent of the delay, since debounce alone does not prevent a slow `"car"`
response landing after `"card"`.

### Q1 — How should one box drive three RPCs?

| Option | Description | Selected |
|--------|-------------|----------|
| Cheap RPCs live, Explore on submit | Search+Files debounced; Explore on Enter, rendered alongside | ✓ |
| All three debounced together | "Alongside" literally true always, but reads 5-20 files' source per pause in typing | |
| Explicit mode toggle | No wasted calls, but makes the two alternatives rather than companions | |

**User's choice:** Cheap RPCs live, Explore on submit.
**Notes:** Settled by the proto — `ExploreGroup` carries a `SourceBlob` *per
matched file* (field 4) for up to `defaultMaxFiles = 5`, so `Explore` performs
disk reads, while `Search` returns bare `Location`s.

### Q2 — How should symbols and files appear together?

| Option | Description | Selected |
|--------|-------------|----------|
| Two labelled sections, symbols first | Each keeps its own server ordering; no invented ranking | ✓ |
| One merged, cross-ranked list | Command-palette feel, but no comparable score exists on the wire | |
| Symbols only, files via a prefix | Focused default, but BRW-01's text has no mode in it | |

**User's choice:** Two labelled sections, symbols first.

### Q3 — Debounce / min-chars / cancellation

| Option | Description | Selected |
|--------|-------------|----------|
| ~150ms, 2 chars, AbortController | Low end of the range because loopback removes network latency | ✓ |
| ~300ms, 3 chars, AbortController | Conservative; costs responsiveness on a local tool | |
| You decide — tune during implementation | Defer numbers to measurement against a real corpus | |

**User's choice:** ~150ms, 2 chars, AbortController.

### Q4 — Focus shortcut (NAV-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Both / and Cmd+K | Serves both the GitHub-conditioned and command-palette-conditioned user | ✓ |
| Cmd+K only | One shortcut, no special-casing, but `/` is muscle memory for this audience | |
| / only | Matches GitHub/Sourcegraph, but Cmd+K users try it first and find nothing | |

**User's choice:** Both / and Cmd+K.

---

## Source rendering & click-to-def (BRW-04/05/06)

**Research applied:** Prism ~2KB core + 0.3-0.5KB per language (~9KB for 12);
highlight.js larger per language but selectively registrable; starry-night 185KB
plus a WASM binary before grammars; Shiki already ruled out at roadmap time.

### Q1 — Which identifiers are clickable?

| Option | Description | Selected |
|--------|-------------|----------|
| Only names in the node's calls list | Uses data already on the wire; locals stay inert | ✓ |
| Every identifier token clickable | Never wrongly excludes, but most clicks lead nowhere | |
| Nothing clickable — probe on hover | No false affordances, but hides the feature entirely | |

**User's choice:** Only names in the node's `calls` list.
**Notes:** Server-side resolved spans were checked and found **structurally
impossible** this phase — `graph.proto`'s `Edge` comment states the Pebble edge
key omits line/col, so multiple call sites collapse to one stored edge. This is
also why BRW-05's picker ships in the same phase.

### Q2 — Which highlighter?

| Option | Description | Selected |
|--------|-------------|----------|
| Prism, 12 languages registered | Smallest (~9KB), but v1 is in maintenance and v2 long in alpha | |
| highlight.js with selective registration | Larger per language, but the stronger maintenance story | ✓ |
| Let research decide against measured bundle size | Mirrors how GRF-01's renderer choice is being handled | |

**User's choice:** highlight.js with selective registration.
**Notes:** Chosen over the smaller option on maintenance grounds. Bundle size
still matters because `web/build/` is committed to git *and* embedded in the
signed binary.

### Q3 — What should a truncated file offer?

| Option | Description | Selected |
|--------|-------------|----------|
| Say it plainly, offer the GitHub permalink | Bounded local view + unbounded remote view complete each other | ✓ |
| Add a ranged source fetch | Most complete, but a second bounded-read path SRV-05 warns against | |
| Just state the truncation | Honest but leaves no route to the rest of the file | |

**User's choice:** Say it plainly, offer the GitHub permalink.

### Q4 — How should BRW-05's picker use `detail_gathered`?

| Option | Description | Selected |
|--------|-------------|----------|
| List all M, mark the ungathered ones | Uses exactly the distinction Phase 1 built the flag for | ✓ |
| Show only the gathered candidates | Every row complete, but hides candidates the user may want | |
| Auto-pick the best, offer the rest | One click shorter, but BRW-05 says "picker instead of guessing" and nothing ranks | |

**User's choice:** List all M, mark the ungathered ones.

### Q5 — The vendored-component supply-chain gap

| Option | Description | Selected |
|--------|-------------|----------|
| Inherit it, record it, keep components minimal | Constrain blast radius; file a todo for a registry-pinning guard | ✓ |
| Add the registry drift guard now | Uniform with `web/build/` and generated TS, but reverses Phase 2's ruling | |
| Write the components by hand instead | Closes the gap fully, but undoes D-17's stated purpose | |

**User's choice:** Inherit it, record it, keep components minimal.
**Notes:** Phase 2 recorded this gap knowing it goes live "the moment Phase 3
adds one" — `web/src/lib/` has no `components/ui/` directory today.

---

## Todo Cross-Reference

Six pending todos matched Phase 3 by area keyword. Two folded.

| Todo | Area | Score | Folded |
|---|---|---|---|
| Wire oracle `toolslist-repeat` response ordering flake | mcp | 0.9 | ✓ |
| `post-release-verify.yml` conclusion guard has no test | ci | 0.9 | |
| Add golangci-lint with gofmt and idiomatic Go linters | ci | 0.7 | ✓ |
| `release:dry-run-signed` additions-only diff guard vacuous | release | 0.6 | |
| tap App secret-distinctness test is tautological | testing | 0.6 | |
| brew trust instructions recommend broader `--tap` grant | docs | 0.2 | |

**Notes:** The recommendation was to fold none — the matches are keyword-driven
(`ci`/`mcp`/`release`) rather than scope-driven, and golangci-lint had already
been folded into Phase 1 and Phase 2 without being implemented either time. That
concern was stated once; the maintainer reaffirmed and selected both. Recorded in
CONTEXT.md as a **third** fold, with the carried-forward history attached and an
explicit requirement that it land as a verifiable task rather than a mention.
Verified before writing: no `.golangci.yml`/`.golangci.yaml` in tree (positive
control: `.goreleaser.yaml` exists), and no pending todo carries `resolves_phase`
(positive control: a completed todo does, proving the pattern matches).

## Claude's Discretion

- Browse view panel layout (how source, callers, callees and blast radius share
  the screen).
- BRW-07's copy affordance and which identifiers are copyable.
- How Impact/Affected depth is surfaced, within the `depth` param.
- Precise `AbortController` wiring and any request-id backstop.
- Whether the status banner is dismissible.
- Exact `GetPermalink` field names and whether `availability` is an enum or
  string, provided it stays three-valued.
- Which shadcn-svelte components are added, subject to minimality.

## Deferred Ideas

- Server-provided syntax-highlight spans / resolved reference ranges (blocked by
  the Pebble edge key omitting line/col).
- A ranged/paginated source fetch for truncated files.
- `end_line` on `Location`.
- Registry-version pinning with a committed-source match assertion for
  shadcn-svelte components.
- Additional forges for BRW-09 and the host-mapping config they would need.
- A vulnerable-JS-fixture red proof for BLD-06 (carried from Phase 2 D-15).
