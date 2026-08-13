# Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) - Research

**Researched:** 2026-08-13
**Domain:** Go `go:embed` cross-directory constraints, Claude Code hooks/skills install surfaces, JSON-array-scoped idempotent merge, content-hash manifests
**Confidence:** HIGH (three findings below are verified against this repo's own source read this session, one against a live empirical `go:embed` experiment run this session, and one against official Claude Code docs fetched this session)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Location scope**
- **D-01:** Skill+hooks installation **follows the existing `--location`/`-l` flag** exactly like the MCP config entry and CLAUDE.md marker block do — no special-casing. Since the flag defaults to `global`, a plain `codegraph install` with no flags places SKILL.md and hooks.json under `~/.claude/...`, not the project-local `.claude/` Phase 6 dogfooded into. — **Reversibility:** costly.
- **D-02:** The consequence of a global install — the skill becoming a candidate in every Claude Code session on the machine, including unindexed repos — is **accepted as-is**, relying entirely on Phase 6's SessionStart nudge (which checks `.codegraph/` existence per-repo) and SKILL.md's own decision-procedure-first structure. No additional guard added in this phase.

**Version observability (AGENT-03)**
- **D-03:** Version is recorded in a **sidecar manifest file** written alongside SKILL.md/hooks.json, not as a field inside SKILL.md's YAML frontmatter. — **Reversibility:** costly.
- **D-04:** The manifest records **the binary version plus a content hash of every file codegraph wrote** — not version alone. Mirrors `writeMcpEntry`'s existing normalized-content-comparison idempotency pattern, extended from a single JSON key to whole files. — **Reversibility:** reversible — additive.

**Hand-edited content on reinstall**
- **D-05:** When install/upgrade is about to overwrite a file and the manifest's stored hash shows the on-disk content no longer matches what codegraph last wrote, the file is **silently overwritten** — no prompt, no warning, no new flag. — **Reversibility:** reversible.

**Upgrade auto-refresh (AGENT-03)**
- **D-06:** `codegraph upgrade` **reads the manifest(s) it finds and re-invokes each previously-configured target's `Install()`** with the new binary's embedded content, after the binary swap itself succeeds. — **Reversibility:** costly.
- **D-07:** If the auto-refresh step fails **after** the binary swap has already succeeded, `codegraph upgrade` **reports the binary swap as successful** and surfaces the refresh failure as a **separate warning** naming `codegraph install` as the command to re-run — it does not make the whole `upgrade` invocation report failure. — **Reversibility:** reversible — purely an exit-code/reporting decision.

### Claude's Discretion
- The exact hooks.json entry identity/merge strategy so uninstall removes precisely codegraph's own entries while preserving unrelated pre-existing entries in the same file. Follow the existing `writeMcpEntry`/`removeMcpEntry` idempotent-JSON-merge pattern as the closest precedent; do not invent a new file-safety primitive.
- The manifest's exact filename and JSON schema (field names, nesting) beyond D-03/D-04's content requirements.
- Whether to add a `codegraph install --status`-style reporting surface that reads the manifest — nice-to-have, planner's call.
- How `upgrade`'s auto-refresh (D-06) locates "the manifest(s) it finds" — scanning both known location roots vs. some other discovery mechanism.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. Multi-agent hooks porting (AGENT-04…07) and the PreToolUse guard hook (GUARD-HOOK-01/02) are tracked as deliberate v2 deferrals in REQUIREMENTS.md, not new ideas surfaced here.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AGENT-01 | `codegraph install` writes SKILL.md + hooks into Claude Code's actual read locations, byte-identical/idempotent across double-install | Critical Finding 1 (hooks target is `.claude/settings.json`, not a standalone `hooks.json`), Architecture Patterns → Pattern 1/2, Code Examples → manifest + merge functions |
| AGENT-02 | `codegraph uninstall` cleanly removes exactly what install wrote, preserving unrelated pre-existing entries | Architecture Patterns → Pattern 2 (array-scoped hooks merge/unmerge), Common Pitfalls → Pitfall 2 |
| AGENT-03 | Skill/hooks package versioned with the binary, refreshed by `codegraph upgrade` | Architecture Patterns → Pattern 4 (manifest schema + discovery), Don't Hand-Roll → upgrade refresh wiring |
| (roadmap criterion 4) | Read error / malformed existing file surfaces as an error, never a silent overwrite/delete, across {install, uninstall} × {read-error, malformed} | Critical Finding 3 (`readJSONFile`'s empty-map fallback is the WRONG precedent; `internal/githooks`'s skip-and-accumulate is the RIGHT one — verified against source read this session) |
</phase_requirements>

## Summary

This phase's biggest risks are not the ones named in CONTEXT.md's Claude's Discretion list — they are three structural facts this session's research uncovered by reading source and running a live experiment, none of which were visible from the phase description alone:

1. **`.claude/hooks/hooks.json` is not a file Claude Code ever reads for a plain (non-plugin) project.** This was already discovered and documented by Phase 6's own research (`06-RESEARCH.md`'s "Critical Correction") and is re-confirmed here: Claude Code reads project-scoped `SessionStart` hook registrations from `.claude/settings.json` (or `.claude/settings.local.json`), never from a bare `hooks/hooks.json` sitting in a plain project directory — that path is real, but only *inside an installed plugin package*. This repo's own `internal/agents/hookpackage_test.go:28-32` confirms this in its own doc comment: `claudeHooksFragmentPath` is explicitly "Phase 7's `go:embed` source fragment (D-04) — NOT itself read by Claude Code in this repository." **Consequence: Phase 7's install-time WRITE target for hooks content is `claudeSettingsPath(loc)` — the exact same file `addClaudeAllowPermission`/`removeClaudeAllowPermission` already read/write today — not a copied `hooks.json` file placed anywhere.** `.claude/hooks/hooks.json` remains the correct *embed source* (unaffected — D-04's embed-source location is not being reopened), it is simply never the install *target*.

2. **A brand-new `.go:embed` directive cannot reach `.claude/` from `internal/agents/`.** Go's `//go:embed` patterns may not contain `..` path elements (confirmed via `golang/go#46056` and the `embed` package's own documentation) — a pattern is resolved only relative to, or below, the directory containing the source file that carries the directive. `.claude/` and `internal/` are siblings at the repository root; no file inside `internal/agents/` can embed anything under `.claude/`. **This was verified empirically this session**, not merely asserted: a scratch Go module confirmed that (a) `..`-relative embed patterns are the failure mode to avoid, and (b) a dot-prefixed directory (`.dotdir/`) can be embedded successfully — both as an explicit file path and as a whole-directory pattern — *when the source file lives in the correct (parent) directory*. **Consequence: the `go:embed` directive must live in a new Go file placed directly in the repository root** (the only directory that is an ancestor of `.claude/`), not inside `internal/agents/` or any other existing package directory.

3. **`readJSONFile`'s existing "malformed JSON → treat as empty map" fallback is the wrong precedent for this phase's own success criterion.** `internal/agents/shared.go:41-44`'s doc comment is explicit that this fallback is a deliberate choice for the *existing* MCP-entry/CLAUDE.md files ("a corrupt/partial config on disk must never panic or block install/uninstall"). But this phase's own roadmap success criterion 4 explicitly requires the opposite for its own new artifacts: "a read error or unparseable existing `hooks.json` makes install and uninstall surface the error instead of overwriting or deleting content they could not read and parse." **The correct, already-proven-in-this-repo precedent is `internal/githooks/githooks.go`'s CR-01/CR-02 pattern** (verbatim, read this session): on a malformed marker block, "Skip this hook entirely and leave the file byte-for-byte untouched — a skipped hook is recoverable, silently eaten user content is not" (`githooks.go:262-272`); on any non-`fs.ErrNotExist` read error, "Skip this hook, accumulate the error, and leave whatever is on disk untouched" (`githooks.go:283-296`). Phase 7 needs a **new, stricter JSON reader** for the settings.json hooks-merge path and the manifest — not `readJSONFile` reused as-is.

A fourth, smaller but still load-bearing finding: the nudge script Phase 6 dogfooded (`session-nudge.sh`) is invoked via `${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh` — correct for a **project-local** hook registration, but **wrong for the `global` default location (D-01)**, since a user-level hook fires across every project on the machine and `${CLAUDE_PROJECT_DIR}` only resolves relative to whatever project happens to be active, not to where the script itself lives. A global install must reference the script via a location-appropriate absolute-or-shell-expandable path (e.g. `~/.claude/hooks/<script>.sh`), and — separately — `fsatomic.WriteFile` writes brand-new files at `0644` by default, which is **not executable**, so installing the script via the existing primitive as-is would silently ship a nudge hook that can never run.

**Primary recommendation:** Write hooks content into `claudeSettingsPath(loc)`'s `hooks.<EventName>` array using a new, array-scoped variant of the existing `writeMcpEntry`/`removeMcpEntry` pattern that identifies "codegraph's own" matcher-block by exact command-string match (not by matcher name alone, since a user may have their own unrelated block using the same matcher value). Place the `go:embed` directive in a new file at the repository root. Store the manifest at `<skillDir>/.codegraph-manifest.json`, sidecar to `SKILL.md`. Build a new strict JSON reader for this phase's artifacts, modeled on `internal/githooks`'s skip-and-accumulate precedent, not `readJSONFile`'s empty-map fallback.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Embedding SKILL.md/hooks.json/session-nudge.sh into the binary | New root-level Go package (`go:embed` source) | `internal/agents` (consumer) | The embed directive is a build-time, filesystem-tree-bound mechanism that cannot cross the `.claude/`↔`internal/` sibling boundary — it must live where `.claude/` is a descendant, i.e. the repo root, and `internal/agents` imports the resulting `embed.FS` |
| Writing SKILL.md to disk at install time | `internal/agents/claude.go` (`claudeTarget.Install`) | `internal/fsatomic` (write primitive) | Same tier that already writes the MCP entry, CLAUDE.md marker block, and settings.json permissions for this target |
| Merging hooks content into `.claude/settings.json` | `internal/agents` (new array-scoped merge helper, sibling to `writeMcpEntry`) | `internal/fsatomic` | Same file, same package, same atomic-write discipline already governing every other Claude Code config write |
| Manifest read/write (version + content hashes) | `internal/agents` (new manifest helper) | `crypto/sha256` (stdlib, already used in `internal/upgrade`) | Colocated with the artifacts it describes; no new package needed |
| Upgrade auto-refresh orchestration (D-06/D-07) | `internal/cli/upgrade.go` (RunE, after `upgradeRunFunc` succeeds) | `internal/agents` (`claudeTarget.Install`) | `internal/upgrade` has zero existing dependency on `internal/agents` and its own tests inject fakes with no agents concept; keeping refresh at the CLI layer (which already imports `internal/agents` for install/uninstall) avoids a new cross-package dependency in the security-critical swap path and makes D-07's "separate warning, not a failed upgrade" trivial to implement as sequential steps in one `RunE` |

## Standard Stack

No new Go module dependencies. Every mechanism this phase needs is already either in the Go standard library or already vendored in this repo for an unrelated purpose:

### Core
| Package/Mechanism | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `embed` (stdlib) | Go 1.16+ (repo is on 1.26.5) | Compile the SKILL.md/hooks.json/session-nudge.sh source content into the binary | Already the repo's own precedent — `internal/mcp/resources.go:18` is "the first `//go:embed` use in this repository" `[VERIFIED: internal/mcp/resources.go:14-19]` |
| `crypto/sha256` (stdlib) | — | Manifest content hashing (D-04) | Already used in this exact codebase for artifact-integrity hashing — `internal/upgrade/upgrade.go:194` (`digest := sha256.Sum256(binary)`) `[VERIFIED: internal/upgrade/upgrade.go:194]` |
| `encoding/json` (stdlib) | — | Manifest serialization, hooks-merge JSON decode/re-encode | Already the exclusive JSON mechanism in `internal/agents/shared.go` |

### Supporting
| Mechanism | Purpose | When to Use |
|---------|---------|-------------|
| `os.Chmod` (stdlib) | Mark the installed nudge script executable | `fsatomic.WriteFile` writes new files at `0644` (`internal/fsatomic/fsatomic.go:52` — "A new file gets the conventional 0644 default") `[VERIFIED: internal/fsatomic/fsatomic.go:51-52]`; a script written through the unmodified primitive would be non-executable |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| A new root-level Go file for `go:embed` | A `go:generate` step that copies `.claude/` content into an `internal/agents/embedded/` directory before build | Rejected — directly contradicts Phase 6's locked D-04 ("`.claude/` IS the canonical source, not a copy... never re-authored or copied by hand"); a generate-time copy is still a copy, and adds a staleness-detection burden (CI would need to enforce the generated copy never drifts from source) for zero benefit over the direct-embed approach that already works |
| Array-scoped command-string identity matching for hooks ownership | Matching by matcher value alone (`"startup"`/`"resume"`) | Rejected — a user (or another tool) may register their own unrelated hook under the same matcher value; matching by matcher alone would either silently delete a user's unrelated hook on uninstall or silently overwrite it on install, exactly the AGENT-02 byte-invariance failure this phase exists to prevent |
| `internal/cli/upgrade.go` orchestrating D-06/D-07 refresh | `internal/upgrade.Run` itself calling into `internal/agents` | Rejected — `internal/upgrade` currently has zero dependency on `internal/agents`, and `upgrade_test.go` exercises `Run`'s resolve→verify→swap sequence entirely through injected fakes; adding an agents dependency to the security-critical swap package for a config-refresh concern blurs a clean boundary for no correctness benefit, since the refresh can just as easily run as a second step in the CLI layer, which already imports `internal/agents` |

**Installation:** None — no package manager invocation needed.

**Version verification:** Not applicable — no new external dependency.

## Package Legitimacy Audit

**Not applicable to this phase.** No external package (Go module or otherwise) is installed by any of this phase's deliverables — every mechanism (`embed`, `crypto/sha256`, `encoding/json`, `os.Chmod`) is Go standard library, already imported elsewhere in this repository for closely analogous purposes. The Package Legitimacy Gate protocol is skipped per its own scope condition.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌───────────────────────────────────────────┐
                    │  Build time (go build)                     │
                    └───────────────────────────────────────────┘
  repo root/*.go     │
  (NEW file, e.g.    │  //go:embed .claude/skills/codegraph/SKILL.md
  claudeassets.go)   │  //go:embed .claude/hooks/hooks.json
                      │  //go:embed .claude/hooks/session-nudge.sh
                      │         │
                      │         ▼
                      │  embed.FS compiled into the codegraph binary
                      │  (moves with `codegraph upgrade` — AGENT-03)
                      └───────────────────────────────────────────┘

                    ┌───────────────────────────────────────────┐
                    │  `codegraph install` (claudeTarget.Install) │
                    └───────────────────────────────────────────┘
  loc = global|local │
  ───────────────►  │  1. Resolve skillDir(loc), settingsPath(loc),
                      │     scriptPath(loc) — mirrors claudeConfigPath's
                      │     existing global/local branch shape
                      │            │
                      │            ▼
                      │  2. Write SKILL.md → skillDir (fsatomic, 0644)
                      │     Write session-nudge.sh → hooksDir (fsatomic,
                      │     then os.Chmod 0755 — script must be executable)
                      │            │
                      │            ▼
                      │  3. Strict-read settings.json (new reader, NOT
                      │     readJSONFile — malformed/unreadable → ERROR,
                      │     file left untouched, install step fails loud)
                      │            │
                      │            ▼
                      │  4. Merge codegraph's SessionStart matcher-blocks
                      │     into hooks.SessionStart[] by command-string
                      │     identity (new array-scoped writeHookEntry) —
                      │     unrelated matcher-blocks/events preserved
                      │            │
                      │            ▼
                      │  5. Write/update manifest at
                      │     skillDir/.codegraph-manifest.json:
                      │     {version, per-file sha256 hashes}
                      └───────────────────────────────────────────┘

                    ┌───────────────────────────────────────────┐
                    │  `codegraph upgrade` (D-06/D-07)            │
                    └───────────────────────────────────────────┘
  binary swap        │
  succeeds ────────► │  6. Probe fixed manifest paths:
                      │     ~/.claude/skills/codegraph/.codegraph-
                      │     manifest.json AND ./.claude/skills/codegraph/
                      │     .codegraph-manifest.json (no directory scan —
                      │     exactly 2 known candidate locations)
                      │            │
                      │            ▼
                      │  7. For each manifest found, re-run
                      │     claudeTarget.Install(loc, opts) with the NEW
                      │     binary's embedded content (idempotent —
                      │     unchanged files are no-ops, D-07 in types.go)
                      │            │
                      │            ▼
                      │  8. Refresh error? → print as a separate warning
                      │     naming `codegraph install`; upgrade's own
                      │     exit code / "upgraded to vX" message is
                      │     UNCHANGED (D-07) — the swap already succeeded
                      └───────────────────────────────────────────┘
```

### Recommended Project Structure
```
<repo root>/
├── claudeassets.go              # NEW — the ONLY valid location for the
│                                   go:embed directive (see Critical
│                                   Finding 2); exports embed.FS + typed
│                                   accessors, imported by internal/agents
internal/agents/
├── claude.go                    # extended: Install/Uninstall gain skill,
│                                   hooks-merge, script, and manifest steps
├── shared.go                    # extended: new array-scoped hooks merge
│                                   helpers (writeHookEntry/removeHookEntry),
│                                   new strict JSON reader, manifest
│                                   read/write/hash helpers
internal/cli/
├── upgrade.go                   # extended: after upgradeRunFunc succeeds
│                                   (and not under --check), call the new
│                                   refresh step; report failures as
│                                   warnings, never as upgrade's own error
```

### Pattern 1: Root-level `go:embed` source file (the only valid placement)
**What:** A new Go file placed directly in the repository root directory (sibling to `go.mod`, `.claude/`, `internal/`, `cmd/`) carries the `//go:embed` directives and exports an `embed.FS`.
**When to use:** Any time source content under `.claude/` must be compiled into the binary — this is the only structurally valid location, verified this session.
**Example (verified via a live scratch experiment this session — not asserted from memory):**
```go
// claudeassets.go — repository root, NOT internal/agents (embed patterns
// cannot cross the .claude/↔internal/ sibling boundary via "..").
package claudeassets

import "embed"

//go:embed .claude/skills/codegraph/SKILL.md
//go:embed .claude/hooks/hooks.json
//go:embed .claude/hooks/session-nudge.sh
var FS embed.FS
```
This session's scratch experiment confirmed both halves of this pattern work: an explicit dot-prefixed path component (`.dotdir/sub/file.md`) embeds successfully via `ReadFile`, and even a bare dot-prefixed directory pattern (`.dotdir`) embeds its subtree successfully via `ReadDir` — the well-known Go embed restriction is `..` path elements and symlinks, not dot-prefixed names reached without `..` `[VERIFIED: live `go run` experiment, this session, output: "explicit-file: hello dotdir <nil>" and "dir-listing: [dr-xr-xr-x ... sub/] <nil>"]`. Deliberately NOT embedding `.claude/skills/codegraph/` as a whole directory: that would also sweep in `.claude/skills/codegraph/verification/*.md` (Phase 6's rehearsal-transcript evidence, not skill runtime content) since those filenames don't start with `.`/`_` and would not be excluded by embed's default dot/underscore filtering. Naming exact files avoids this entirely.
*Source: `golang/go#46056` ("`//go:embed <path to file in parent directory>` doesn't work") `[CITED: github.com/golang/go/issues/46056]`, cross-checked against this session's own experiment `[VERIFIED]`.*

### Pattern 2: Array-scoped hooks merge (command-string identity, not matcher identity)
**What:** Extend the `writeMcpEntry`/`removeMcpEntry` idempotent-merge shape (`internal/agents/shared.go:130-193`, read this session) to `hooks.<EventName>[]`, an array of `{matcher, hooks[]}` blocks rather than a single named map key.
**When to use:** Any write/remove into `claudeSettingsPath(loc)`'s `hooks` object.
**Identity rule:** codegraph owns exactly the matcher-blocks whose `hooks[]` array contains an entry with `"type":"command"` and `"command"` equal to codegraph's own resolved script command string for `loc` (built the same location-aware way `claudeInstructionsPath`/`claudeSettingsPath` already resolve global vs. local). Never identify ownership by matcher value alone — see Common Pitfalls → Pitfall 1.
**Example (shape, following the existing `writeMcpEntry` control flow):**
```go
// Pseudocode shape — mirrors writeMcpEntry's read→locate→compare→write
// control flow (internal/agents/shared.go:135-165), extended to search an
// array by content rather than look up a single map key.
func writeHookEntry(path, event, matcher, ownCommand string, hookEntry any) (FileResult, error) {
    existing, err := readSettingsJSONStrict(path) // NEW strict reader — see Pattern 3
    if err != nil {
        return FileResult{}, err // malformed/unreadable → caller surfaces, file untouched
    }
    hooks, _ := existing["hooks"].(map[string]any)
    if hooks == nil { hooks = map[string]any{} }
    events, _ := hooks[event].([]any)

    // Find an existing block that already contains ownCommand.
    idx := indexOfBlockContainingCommand(events, ownCommand)
    if idx >= 0 && jsonDeepEqual(events[idx], expectedBlock(matcher, hookEntry)) {
        return FileResult{Path: path, Action: ActionUnchanged}, nil // D-07 idempotency
    }
    if idx >= 0 {
        events[idx] = expectedBlock(matcher, hookEntry) // update our own block in place
    } else {
        events = append(events, expectedBlock(matcher, hookEntry)) // append alongside any unrelated blocks
    }
    hooks[event] = events
    existing["hooks"] = hooks
    // ... writeJSONFile, same as writeMcpEntry
}
```
This preserves every unrelated event key (`PreToolUse`, etc.) and every unrelated matcher-block under the SAME event untouched — the exact AGENT-02 requirement ("a `hooks.json` that already carried unrelated user-authored entries before codegraph ever touched it"). Uninstall is the mirror: locate the block by command-string, remove only codegraph's entry from its `hooks[]` sub-array, delete the whole block if that empties it, delete the event key if that empties the array, delete the top-level `hooks` key if that empties it too — mirroring `removeMcpEntry`'s existing keep-clean discipline (`internal/agents/shared.go:184-188`) `[VERIFIED: internal/agents/shared.go:171-193]`.

### Pattern 3: Strict JSON reader for artifacts this phase introduces (do not reuse `readJSONFile`'s fallback)
**What:** A new reader used for `claudeSettingsPath(loc)` (in the hooks-merge path) and the manifest file, distinguishing three outcomes instead of `readJSONFile`'s two: file absent (proceed as empty — matches `os.IsNotExist`), genuine read/parse failure (return an error, caller must NOT write), success.
**When to use:** Every read this phase performs before a hooks-merge write or a manifest read/write.
**Verified precedent — `internal/githooks/githooks.go` already implements exactly this discipline for a structurally identical problem (marker-fenced content in a file this project doesn't fully own):**
```go
// internal/githooks/githooks.go:256-297 (read this session, quoted verbatim)
existing, err := os.ReadFile(file)
switch {
case err == nil:
    base := string(existing)
    stripped, ok := stripMarkerBlock(base)
    if !ok {
        // A malformed marker block means the strip can't be trusted (CR-01).
        // ... Skip this hook entirely and leave the file byte-for-byte
        // untouched — a skipped hook is recoverable, silently eaten user
        // content is not.
        errs = append(errs, fmt.Errorf("%s: hook file has a malformed codegraph marker block — please fix or remove it manually", hook))
        continue
    }
    // ... normal path
case errors.Is(err, fs.ErrNotExist):
    content = "#!/bin/sh\n" + block + "\n"
default:
    // CR-02: any other read error ... Skip this hook, accumulate the
    // error, and leave whatever is on disk untouched — same "recoverable
    // skip beats silent data loss" contract as the malformed-marker branch.
    errs = append(errs, fmt.Errorf("%s: could not read existing hook file: %w", hook, err))
    continue
}
```
`[VERIFIED: internal/githooks/githooks.go:256-296]` — this is the in-repo precedent named by CONTEXT.md's own framing ("the invariant v1.0 Phase 5 converged on after two reproduced data-loss Criticals"); it long predates `readJSONFile`'s deliberately-more-permissive fallback and is the correct model to copy for Phase 7's JSON case, not `readJSONFile` itself. Contrast with `readJSONFile`'s documented (and, for this phase's purposes, wrong) choice: `[VERIFIED: internal/agents/shared.go:41-44]` — *"A missing file, an empty file, or unparseable content all fall back to an empty map rather than erroring... a corrupt/partial config on disk must never panic or block install/uninstall."* That is the right call for the MCP-entry/CLAUDE.md paths it already governs; it is the wrong call for this phase's own explicit success criterion.

### Pattern 4: Manifest schema and location
**What:** One manifest file per (target, location) pair, physically sidecar to `SKILL.md`.
**Location:** `<skillDir(loc)>/.codegraph-manifest.json` — i.e. `~/.claude/skills/codegraph/.codegraph-manifest.json` (global) or `./.claude/skills/codegraph/.codegraph-manifest.json` (local). A dot-prefixed filename keeps it out of any hypothetical future recursive skill-content embed/scan and signals "codegraph-internal, not skill content" the same way `.codegraph/` itself is dot-prefixed at the project root.
**Schema (recommendation — planner's call on exact field names per CONTEXT.md discretion, this is the shape to start from):**
```json
{
  "schema_version": 1,
  "codegraph_version": "v0.10.0",
  "installed_at": "2026-08-13T00:00:00Z",
  "location": "global",
  "files": {
    "skills/codegraph/SKILL.md": "sha256:<hex>",
    "hooks/session-nudge.sh": "sha256:<hex>",
    "settings.json#hooks.SessionStart": "sha256:<hex-of-the-owned-fragment-only>"
  }
}
```
The `settings.json#hooks.SessionStart` entry is a hash of **only the codegraph-owned matcher-blocks** (the exact JSON this phase writes, re-marshaled deterministically), not of the whole `settings.json` file — settings.json is shared with unrelated user content, so hashing the whole file would false-positive "drifted" on every unrelated user edit, defeating D-05's "hand-edit-detected drift" signal (which should fire only when *codegraph's own* content was hand-edited, not when a sibling permission was added). Hash `version.Info().Version` via `crypto/sha256`, hex-encode (`encoding/hex`), matching the algorithm already established for binary-integrity hashing in `internal/upgrade/upgrade.go:194` `[VERIFIED: internal/upgrade/upgrade.go:194]`.
**D-05 drift check:** before overwriting SKILL.md/session-nudge.sh/the hooks fragment, compute the on-disk content's current hash and compare to the manifest's stored hash for that entry. A mismatch means a hand-edit happened since the last codegraph write — per D-05, overwrite silently anyway (no prompt). A missing manifest (first-ever install, or a pre-Phase-7 install) means "no baseline to compare against" — treat identically to "drifted" (silently overwrite), never treat a missing manifest as "don't touch this file."

### Pattern 5: Location-appropriate hook command path (not a bare copy of Phase 6's project-relative pattern)
**What:** The `command` string Phase 7 writes into `hooks.SessionStart[].hooks[].command` must differ by `loc`, unlike Phase 6's single project-local script.
**When to use:** Building the hook entry inside Pattern 2's merge call.
**Local (`loc == LocationLocal`):** `${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh` — identical in shape to Phase 6's own dogfooded fragment (`internal/agents/hookpackage_test.go`'s `claudeHooksFragmentPath` content, read this session), since a local install genuinely places the script inside the current project's own `.claude/hooks/`.
**Global (`loc == LocationGlobal`):** an absolute, home-relative path to where Phase 7 actually writes the script — e.g. `~/.claude/hooks/session-nudge.sh` in shell form (Claude Code's official hooks docs confirm `~` expands in shell-form `command` entries — i.e. a bare `command` string with no separate `args` array — but is treated literally, unexpanded, in exec form) `[CITED: code.claude.com/docs/en/hooks, fetched this session]`. `${CLAUDE_PROJECT_DIR}` is documented as available even for a user-level hook's runtime environment, but that only helps a script that needs to know the *current* project — it does nothing to help Claude Code *locate the script itself*, which for a global install lives under the user's home directory, not under whatever project happens to be open `[CITED: code.claude.com/docs/en/hooks — "`${CLAUDE_PROJECT_DIR}`: the project root", fetched this session]`.
**Consequence for the script's own logic:** Phase 6's `session-nudge.sh` already does its own `.codegraph/`-existence check relative to `${CLAUDE_PROJECT_DIR}` (or its own cwd) at *runtime*, inside the script body — this part is location-agnostic and needs no change; only the *hook registration's path to the script file* differs by install location.

### Anti-Patterns to Avoid
- **Writing a standalone `~/.claude/hooks/hooks.json`-shaped file and calling it done:** inert, per Critical Finding 1 — Claude Code will never read it outside a plugin package.
- **Hashing all of `settings.json` for D-05 drift detection:** false-positives on every unrelated edit; hash only codegraph's owned fragment (Pattern 4).
- **Reusing `readJSONFile` unmodified for the hooks-merge path:** silently treats a malformed `settings.json` as empty, then overwrites it with only codegraph's own content on the next write — directly violates this phase's own read-error/malformed invariant (Pattern 3).
- **Identifying "codegraph's own" hooks entries by matcher value:** a matcher like `"startup"` is not unique to codegraph; use command-string identity (Pattern 2).
- **Placing the `go:embed` directive inside `internal/agents/`:** will not compile — `..` path elements are rejected (Pattern 1).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Atomic file writes for SKILL.md, the script, settings.json, the manifest | A new atomic-write helper | `internal/fsatomic.WriteFile` | Roadmap's own explicit instruction: "do not invent new file-safety primitives" — this is already the single funnel every existing agent-config write in this repo uses |
| JSON read/decode/compare for the manifest and (adapted) for the hooks merge | A hand-rolled JSON walker | `encoding/json` + the existing `jsonDeepEqual`/`normalizeJSON` helpers (`internal/agents/shared.go:85-128`, read this session) | These already implement map-order-independent structural equality, exactly what D-07 idempotency and Pattern 2's block-comparison both need |
| Content hashing for the manifest | A custom checksum or a different hash algorithm | `crypto/sha256`, hex-encoded | Already this exact codebase's established idiom for artifact-integrity hashing (`internal/upgrade/upgrade.go:194`) — reusing it avoids introducing a second hash convention into the same binary |
| Upgrade's per-target re-install loop | A bespoke "which targets were configured" scan | The manifest's own fixed, well-known paths (2 candidates: `~/.claude/skills/codegraph/.codegraph-manifest.json`, `./.claude/skills/codegraph/.codegraph-manifest.json`) | Claude-only scope (this phase) means discovery is a 2-path `os.Stat`, not a filesystem walk — building anything more general is unneeded generality for a phase explicitly scoped to one agent |

**Key insight:** Every one of this phase's genuinely new mechanisms (array-scoped merge, strict reader, manifest hashing) is a small, targeted *extension* of a pattern this repo already trusts (`writeMcpEntry`, `internal/githooks`'s skip-on-malformed discipline, `internal/upgrade`'s sha256 usage) — not a new invention. The roadmap's "do not invent new file-safety primitives" instruction is satisfiable literally, not just in spirit.

## Common Pitfalls

### Pitfall 1: Identifying "codegraph's own" hooks entries by matcher value instead of command content
**What goes wrong:** Install checks `hooks.SessionStart` for a block with `matcher == "startup"`; if the user already has their own unrelated `"startup"` block (a different script, added independently), codegraph either merges into it (corrupting a block it doesn't own) or replaces it (destroying the user's hook) — either way AGENT-02's "preserving unrelated pre-existing entries" fails.
**Why it happens:** `hooks.<Event>` is an *array* specifically because Claude Code allows multiple independent blocks per matcher value; treating matcher as a unique key is a natural but incorrect mental model carried over from `writeMcpEntry`'s single-named-key precedent (`mcpServers.codegraph`), which has no array-of-siblings-with-the-same-name equivalent.
**How to avoid:** Identify ownership by exact `command` string match within a block's `hooks[]` sub-array (Pattern 2), never by matcher alone.
**Warning signs:** A merge/removal function whose only lookup key is the matcher string, with no comparison against the actual command content.
**Phase to address:** This phase.

### Pitfall 2: Placing the `go:embed` directive somewhere it will silently fail to compile — or worse, compiles but embeds the wrong tree
**What goes wrong:** A directive inside `internal/agents/*.go` referencing `../../.claude/...` fails to build at all (`..` is rejected). A directive that embeds the whole `.claude/skills/codegraph/` directory (rather than naming `SKILL.md` explicitly) silently also embeds `.claude/skills/codegraph/verification/*.md` — Phase 6's rehearsal-transcript evidence files, which are not skill runtime content and should not ship to every install target.
**Why it happens:** `go:embed`'s directory-tree-rooted-at-the-source-file constraint is easy to overlook when the source content (`.claude/`) already exists at a fixed, familiar path; the temptation is to embed "the directory" for convenience rather than naming exact files.
**How to avoid:** New root-level Go file (Pattern 1); name `SKILL.md`, `hooks.json`, and `session-nudge.sh` as explicit file paths in the embed directive, never a bare directory pattern over `skills/codegraph/`.
**Warning signs:** A `go build ./...` failure mentioning `..`; or (much more dangerous — a silent success) an installed skill directory on a test machine unexpectedly containing a `verification/` subfolder.
**Phase to address:** This phase.

### Pitfall 3: Shipping a non-executable nudge script
**What goes wrong:** `fsatomic.WriteFile` creates new files at mode `0644` (`internal/fsatomic/fsatomic.go:51-52`, read this session). If Phase 7 writes `session-nudge.sh` through this primitive unmodified, the installed script is not executable, and Claude Code's `SessionStart` hook invocation fails silently or produces a permission-denied stderr line the user is unlikely to notice — NUDGE-01 (Phase 6) regresses for every fresh install this phase produces, with no test currently positioned to catch it (Phase 6's own hook tests exercise the *dogfooded, git-committed* copy, which git already tracks with an executable bit, not a freshly-`fsatomic`-written one).
**Why it happens:** `fsatomic.WriteFile`'s mode-preservation logic only preserves an *existing* file's mode; a brand-new file has no prior mode to preserve, so it falls back to the generic `0644` default appropriate for config/markdown files but wrong for a script.
**How to avoid:** After writing the script via `fsatomic.WriteFile`, explicitly `os.Chmod(scriptPath, 0o755)` — or add a small sibling helper in `internal/fsatomic` for executable writes, reusing the same underlying atomic mechanism (an additive extension, not a new primitive).
**Warning signs:** A fresh-install integration test that only checks file *contents* match, never file *mode*.
**Phase to address:** This phase.

### Pitfall 4: Global-scope hook registration referencing a script that doesn't exist at that path
**What goes wrong:** Global install writes the hooks fragment with `${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh` (Phase 6's project-local pattern, copied verbatim) as the command, but the script itself is written to `~/.claude/hooks/session-nudge.sh` (the global location) — the two paths never coincide except by accident (only when the currently-open project happens to also be this very repo). Every other project on the machine gets a `SessionStart` hook that references a file that doesn't exist there.
**Why it happens:** Copy-pasting Phase 6's already-working, project-local fragment without re-deriving the path for the `global` case — the two locations need genuinely different command strings, not just a different script destination.
**How to avoid:** Pattern 5 — build the command string as a function of `loc`, mirroring how `claudeConfigPath`/`claudeInstructionsPath`/`claudeSettingsPath` already branch on `loc`.
**Warning signs:** A test suite that only exercises `LocationLocal` (matching Phase 6's own dogfood) and never asserts on the `LocationGlobal` command string's actual resolved path.
**Phase to address:** This phase.

### Pitfall 5: Hashing the whole `settings.json` for D-05 drift detection
**What goes wrong:** D-05's "hand-edit-detected drift" check fires (and silently overwrites) every time the user adds an unrelated permission, a new hook for a different tool, or any other settings.json edit — because the manifest's stored hash no longer matches the whole file's current hash, even though codegraph's own owned content is untouched. This is not merely noisy: it means "silently overwrite" (D-05's declared behavior) applies far more broadly than D-05's own rationale intended, since D-05 exists specifically for "SKILL.md and hooks.json's codegraph-owned entries," not for the whole shared file.
**Why it happens:** Hashing "the file" is the naive reading of D-04's "content hash of every file codegraph wrote" when the artifact in question (hooks content) doesn't correspond to a whole physical file at all.
**How to avoid:** Hash only the codegraph-owned JSON fragment (the specific matcher-blocks this phase writes), re-marshaled deterministically, not the containing file (Pattern 4).
**Phase to address:** This phase.

## Code Examples

### Existing precedent: location-aware path resolution (the pattern new helpers must follow)
```go
// internal/agents/claude.go:76-117, read this session — this exact
// global/local branching shape is what claudeSkillDirPath(loc),
// claudeHooksScriptPath(loc), and claudeManifestPath(loc) should mirror.
func claudeConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return ".mcp.json", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}
```

### Existing precedent: the mandatory error funnel every new write/read must use
```go
// internal/agents/shared.go:12-24, read this session.
func recordFile(result *WriteResult, path string, fr FileResult, err error) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
		return
	}
	result.Files = append(result.Files, fr)
}
```
Every new call site this phase adds (skill write, script write, hooks merge, manifest write) must route through `recordFile`/`recordAction`, exactly like the ~40 existing call sites already do (CR-01's swallowed-error guarantee).

### Existing precedent: binary-integrity hashing (the manifest's hashing should match this idiom)
```go
// internal/upgrade/upgrade.go:194, read this session.
digest := sha256.Sum256(binary)
return verifyRelease(b, trustedMaterial, "sha256", digest[:], releaseWorkflowRefPattern)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Assumed `hooks.json` is directly readable by Claude Code in any project directory (`.planning/research/SUMMARY.md`'s original assumption, per Phase 6's own research) | Confirmed only `.claude/settings.json`/`.claude/settings.local.json` (or a plugin's own `hooks/hooks.json`) are read for a plain project | Corrected in Phase 6's research (2026-08-12), re-confirmed this session against this repo's own `hookpackage_test.go` doc comments | This phase's install-time write target is `.claude/settings.json`, never a copied `hooks.json` file |
| Hook events list assumed to be ~10 events (Phase 6 research's earlier pass) | 25+ documented events as of this session's live fetch (`SessionStart`, `Setup`, `SessionEnd`, `UserPromptSubmit`, `UserPromptExpansion`, `Stop`, `StopFailure`, `PreToolUse`, `PermissionRequest`, `PermissionDenied`, `PostToolUse`, `PostToolUseFailure`, `PostToolBatch`, `WorktreeCreate`, `WorktreeRemove`, `FileChanged`, `DirectoryAdded`, `CwdChanged`, `ConfigChange`, `InstructionsLoaded`, `Notification`, `MessageDisplay`, `SubagentStart`, `SubagentStop`, `TaskCreated`, `TaskCompleted`, `TeammateIdle`, `Elicitation`, `ElicitationResult`, `PreCompact`, `PostCompact`) | Confirmed via live fetch this session | Not directly load-bearing for this phase (only `SessionStart` is used), but confirms the hooks surface is actively evolving — re-verify if implementation slips |

**Deprecated/outdated:** None specific to this phase beyond the two corrections above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The manifest's exact filename (`.codegraph-manifest.json`) and its placement directly inside `skillDir` (rather than, say, a separate `.codegraph/` subdirectory under the skill dir) is this research's recommendation, not a locked decision — CONTEXT.md leaves the exact filename/schema to the planner | Pattern 4 | Low — purely cosmetic; any consistent, documented choice satisfies D-03/D-04 |
| A2 | Claude Code's own runtime behavior when re-reading a `settings.json` this phase wrote — specifically, whether it tolerates unknown top-level keys or additional matcher-blocks under an event it also manages internally — was not independently re-verified this session (the official docs fetch explicitly said "Regarding unrecognized entries: The documentation does not explicitly state whether unrecognized matcher entries or event keys are preserved on re-read. This is implementation-dependent behavior not covered in the reference.") This is a claim about *our own read-modify-write code's* behavior (which we fully control via `readJSONFile`-style generic-map round-tripping), not a claim about Claude Code's runtime — so the risk is scoped narrowly | Pattern 2 | Low for AGENT-01/02 (our own code preserves unknown keys by construction); if Claude Code's own runtime turns out to reject or ignore some shape we produce, that would surface as a live-session NUDGE-01-style regression, not a byte-invariance test failure — worth a live-session smoke check during this phase's own verification, mirroring Phase 6's NUDGE-01 live-session approach |
| A3 | `~` expands correctly in a shell-form (`command`-only, no `args`) hook entry at BOTH global and local install scope, per the official docs fetch — not independently re-verified against a live Claude Code session this research session (this repo's own Phase 6 dogfood never exercised the global/`~`-expansion path, only the project-local `${CLAUDE_PROJECT_DIR}` path) | Pattern 5 | Medium — if `~` expansion turns out not to work for some platform/shell combination, the global-scope nudge would silently never fire; recommend a live-session smoke test of the GLOBAL install path specifically (not just local, which Phase 6 already proved), or use a fully-resolved absolute path (`filepath.Join(home, ...)`) instead of a literal `~`, sidestepping the expansion question entirely — this alternative is strictly safer and costs nothing, so the planner should prefer it over relying on `~` expansion |

**Note on tag choice:** claims about this repo's own source (`shared.go`, `claude.go`, `fsatomic.go`, `githooks.go`, `hookpackage_test.go`, `upgrade.go`) are tagged `[VERIFIED: <path>:<lines>]` because they were opened and read with `Read` this session, with verbatim quotes included above. The `go:embed` `..`-restriction and dot-directory-embeddability claims are tagged `[VERIFIED]` because they were additionally proven via a live `go run` experiment this session (not merely cited from docs). Claims about Claude Code's own hooks documentation are tagged `[CITED: code.claude.com/docs/en/hooks]` because they were fetched live this session via `WebFetch`, not drawn from training memory.

## Open Questions

1. **Should the pre-existing `addClaudeAllowPermission`/`removeClaudeAllowPermission` read path (which today uses the permissive `readJSONFile`, not the new strict reader this phase introduces) be migrated onto the same strict reader for consistency?**
   - What we know: both functions read/write the exact same file (`claudeSettingsPath(loc)`) this phase's hooks-merge logic will also touch. Today, `addClaudeAllowPermission` against a malformed `settings.json` would silently treat it as empty and overwrite it with only the new permission — the same failure class this phase's own success criterion 4 explicitly forbids for the *new* artifact type.
   - What's unclear: whether fixing this pre-existing gap is in scope for AGENT-01/02/03 (which name hooks.json/SKILL.md, not the pre-existing AutoAllow permission path) or should be filed as a separate follow-up.
   - Recommendation: strongly prefer having ONE strict reader for `settings.json` used by both the pre-existing AutoAllow path and this phase's new hooks-merge path — two different read-error postures for the same file, in the same package, would itself be a maintenance hazard. If the planner descopes the AutoAllow migration to keep this phase's diff minimal, file it as an explicit follow-up rather than leaving it implicit.

2. **Does the manifest need a `location` field at all, given the manifest's own file path already encodes location (global vs. local) by construction?**
   - What we know: the two manifest paths (`~/.claude/skills/.../`, `./.claude/skills/.../`) are already location-distinguishing; a `location` field inside the JSON would be redundant with the path that led there.
   - What's unclear: whether the upgrade auto-refresh discovery step (which needs to call `Install(loc, ...)` for the right `loc`) is simpler reading the field from JSON vs. deriving `loc` from which of the two fixed candidate paths existed.
   - Recommendation: keep the field anyway — it costs nothing, makes the manifest self-describing for any future `codegraph install --status` surface (CONTEXT.md discretion item 3), and avoids a subtle bug class where a future refactor moves the discovery logic somewhere that no longer has direct access to "which candidate path matched."

## Environment Availability

Not applicable — this phase's dependencies are entirely Go standard library and content already present in this repository (`.claude/skills/codegraph/SKILL.md`, `.claude/hooks/hooks.json`, `.claude/hooks/session-nudge.sh`, all confirmed present this session via `ls`). No external tool, service, or runtime dependency is introduced.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), matching every existing test in `internal/agents` and `internal/upgrade` |
| Config file | none — plain `go test` |
| Quick run command | `go test ./internal/agents/... ./internal/cli/... ./internal/upgrade/...` |
| Full suite command | `task test` (per `Taskfile.yml`, confirmed this session — runs `test:unit`, `test:golden`, `test:integration`, `test:wireoracle`, `test:daemon`, `test:race` serially) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AGENT-01 | `install` writes SKILL.md + hooks fragment into `settings.json`, byte-identical (JSON-deep-equal, per this repo's own "byte-invariant" convention — `TestClaude_GlobalRoundTrip_ByteInvariantWithSibling` compares parsed JSON via `jsonDeepEqual`, not raw bytes `[VERIFIED: internal/agents/claude_test.go:182-211]`) across a double install | unit | `go test ./internal/agents/... -run TestClaude` | ❌ Wave 0 — new test cases needed, following the existing `TestClaude_Install_ReRunIsByteIdempotent` shape (`internal/agents/claude_test.go:275`) |
| AGENT-02 | `uninstall` preserves unrelated pre-existing hooks entries and permission entries in the same file | unit | `go test ./internal/agents/... -run TestClaude` | ❌ Wave 0 — following the existing `TestClaude_Uninstall_PreservesUnrelatedAllowEntries` shape (`internal/agents/claude_test.go:227`), extended to a hooks fixture with an unrelated matcher-block |
| AGENT-03 | Manifest records version + hashes; `upgrade` re-invokes `Install()` for previously-configured targets after a successful swap | unit | `go test ./internal/agents/... ./internal/cli/...` | ❌ Wave 0 — new manifest read/write tests, new `TestUpgradeCommand_RefreshesConfiguredTargets`-shaped test following `TestUpgradeCommand_DelegatesWithCheckAndVersion`'s fake-injection pattern (`internal/cli/upgrade_test.go:17-46`) |
| Read-error/malformed matrix | {install, uninstall} × {read-error, malformed} all surface an error, file untouched | unit | `go test ./internal/agents/...` | ❌ Wave 0 — new tests following `internal/githooks`'s own malformed-fixture test pattern (`malformedHookFixture()`, referenced in `05-REVIEW.md:155`) |
| NUDGE-01 regression risk (Pitfall 3) | Installed script is executable | unit or integration | a new assertion on `os.Stat(scriptPath).Mode()&0o111 != 0` after a real `Install()` call against a temp dir | ❌ Wave 0 — no existing test checks file *mode*, only content |
| Global-scope hook path correctness (Pitfall 4) | Global install's hook command resolves to the script's actual global-install location | unit | assertion on the exact `command` string written for `LocationGlobal`, distinct from `LocationLocal`'s | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/agents/... ./internal/cli/...`
- **Per wave merge:** `task test` (full suite — this phase touches `internal/agents`, `internal/cli`, and potentially a new root package; cheap to confirm nothing else is disturbed)
- **Phase gate:** Full suite green, plus a live-session smoke check of the GLOBAL install path specifically (Assumption A3) before `/gsd-verify-work`, given this phase's roadmap-declared "highest-risk phase of the milestone" status and the explicit call for "a deep-review pass"

### Wave 0 Gaps
- [ ] New root-level `go:embed` source file (Pattern 1) — does not exist yet
- [ ] New array-scoped hooks merge helpers (`writeHookEntry`/`removeHookEntry`, Pattern 2) — do not exist yet
- [ ] New strict JSON reader for the hooks-merge/manifest read path (Pattern 3) — does not exist yet
- [ ] Manifest read/write/hash helpers (Pattern 4) — do not exist yet
- [ ] `claudeSkillDirPath(loc)`/`claudeHooksScriptPath(loc)`/`claudeManifestPath(loc)` location-aware path helpers (mirroring `claudeConfigPath`) — do not exist yet
- [ ] Upgrade auto-refresh orchestration in `internal/cli/upgrade.go` (D-06/D-07) — does not exist yet
- [ ] Test fixtures: a `settings.json` with an unrelated `SessionStart` matcher-block AND an unrelated event key, for AGENT-02's byte-invariance proof; a malformed `settings.json` fixture, for the read-error matrix

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface added |
| V3 Session Management | no | Not applicable |
| V4 Access Control | no | No access-control surface added |
| V5 Input Validation | yes | The manifest and settings.json content this phase reads back are files this project itself previously wrote (or a third party's, in the merge case) — the strict-reader pattern (Pattern 3) IS the input-validation control: malformed/unreadable input must fail closed, never silently proceed with a guessed/empty structure that then gets written back |
| V6 Cryptography | marginal | `crypto/sha256` for content-integrity hashing (not a security/authentication boundary — the manifest's hash exists to detect hand-edits for D-05's drift signal, not to defend against a malicious actor) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Overwriting or corrupting a user's `settings.json` on a malformed-JSON read (self-inflicted, not adversarial) | Tampering | Pattern 3's strict reader — malformed/unreadable content is never silently treated as empty and overwritten; the file is left untouched and the error surfaces (matches `internal/githooks`'s existing CR-01/CR-02 precedent) |
| A hook script referenced by a non-absolute, non-anchored path being shadowed by a same-named file earlier in `$PATH` | Spoofing | Reference the script by its full resolved path (project-relative via `${CLAUDE_PROJECT_DIR}` for local, home-relative absolute for global — Pattern 5), never by bare filename; already this repo's established convention for the MCP entry (`opts.ExecPath`, the resolved running binary's absolute path, not a PATH lookup) `[VERIFIED: internal/cli/install.go:72-75, internal/agents/types.go:117-121]` |
| Installed script world-writable or otherwise over-permissioned | Elevation of Privilege (defense in depth) | `os.Chmod(scriptPath, 0o755)` — owner-executable, not group/world-writable; matches the conventional mode for a committed shell script (Phase 6's own git-tracked `session-nudge.sh`) |
| Manifest hash used as a security boundary rather than a drift-detection signal | (n/a — scope clarification, not a threat) | D-05 already establishes the manifest hash is advisory only ("silently overwrites — no prompt, no warning"), not a tamper-detection gate; no control here should be built as if a hash mismatch were a security event |

## Sources

### Primary (HIGH confidence)
- `internal/agents/types.go`, `internal/agents/claude.go`, `internal/agents/shared.go`, `internal/fsatomic/fsatomic.go`, `internal/cli/install.go`, `internal/cli/uninstall.go`, `internal/cli/upgrade.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/verify.go`, `internal/mcp/resources.go`, `internal/version/version.go`, `internal/agents/registry.go`, `internal/agents/instructions.go`, `internal/agents/hookpackage_test.go`, `internal/agents/claude_test.go`, `internal/cli/upgrade_test.go`, `internal/githooks/githooks.go` — all read directly this session, verbatim quotes used above where load-bearing
- `.claude/hooks/hooks.json`, `.claude/settings.json`, `.claude/skills/codegraph/SKILL.md` (existence/shape) — read directly this session
- `.planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-RESEARCH.md` — Phase 6's own "Critical Correction" finding, re-confirmed (not merely trusted) against this repo's own `hookpackage_test.go` source this session
- `.planning/milestones/v1.0-phases/05-git-sync-hooks/05-REVIEW.md` — located and confirmed the exact "v1.0 Phase 5's two reproduced data-loss Criticals" CONTEXT.md refers to, cross-checked against `internal/githooks/githooks.go`'s actual shipped code
- Live `go run` scratch experiment (this session, `/private/tmp/.../scratchpad/embedtest`) — empirically confirmed `go:embed`'s dot-directory and `..`-restriction behavior rather than relying on documentation alone

### Secondary (MEDIUM confidence)
- `code.claude.com/docs/en/hooks` — fetched live via `WebFetch` this session; hook JSON schema, `${CLAUDE_PROJECT_DIR}` semantics, `~` expansion in shell-form commands, full event/matcher list
- `golang/go#46056` ("`//go:embed <path to file in parent directory>` doesn't work") — fetched live via `WebSearch` this session, corroborating the empirical experiment
- Claude Code Agent Skills directory-location search results (`code.claude.com/docs/en/skills`, `agensi.io`, `getclaudeskills.com`) — cross-checked, all agree on `~/.claude/skills/<name>/SKILL.md` for personal/global scope

### Tertiary (LOW confidence)
- None — every claim in this document traces to either a source file read this session, an official doc fetched live this session, or an experiment run this session.

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — zero new dependencies, every mechanism already proven in this exact codebase
- Architecture (embed placement, hooks merge, strict reader, manifest): **HIGH** — the three critical findings were each independently verified this session (source read, live experiment, official docs fetch), not carried over unverified from the phase description
- Pitfalls: **HIGH** — all five pitfalls trace directly to a verified source-code fact or a verified experimental result, not speculation

**Research date:** 2026-08-13
**Valid until:** ~30 days for this repo's own source facts (stable until a later phase touches `internal/agents`/`internal/fsatomic`/`internal/upgrade`); ~14 days for the Claude Code hooks documentation specifically, which is an actively evolving surface (event count already grew between Phase 6's research pass on 2026-08-12 and this session on 2026-08-13's fetch confirming the same ~25+ event count) — re-verify Pattern 5's `~`-expansion claim (Assumption A3) against a live session before relying on it, given it was not independently re-tested this session beyond the official docs' own statement.
