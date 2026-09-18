# 05-MUTATION-LOG — Agent Reach — Capability Model & Skill in Every Harness

**Phase:** 05-agent-reach-capability-model-skill-in-every-harness
**Date:** 2026-09-18
**Scope:** Family (a) — D-03's capability-table guard (plan 05-01): (a1) a
hand-written extra path appended to Cursor's `DescribePaths` turns
`TestCapabilitiesTableDrivesDerivations/cursor/…` RED; (a2) deleting
`Instructions` from Claude's `Capabilities()` literal turns
`TestCapabilitiesMatchInstallWrites/claude/…` RED.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>`
is asserted to exit clean. This proves no pre-existing tracked edit was overwritten by the
mutation, and no revert was a destructive blind checkout of someone else's in-flight work.
Every family entry below records this gate's result at the point it was checked.

---

## Family (a1) — D-03: an extra path in Cursor's DescribePaths turns TestCapabilitiesTableDrivesDerivations RED

**Test/guard:** `TestCapabilitiesTableDrivesDerivations`
(`internal/agents/capabilities_test.go`) — the D-03 guard asserting
`DescribePaths(loc)` is set-equal to a path set built independently from
`Capabilities()` alone, for every registered target x {global, local}.

**What are we testing, and why?** Whether the set-equality assertion
actually catches a target's `DescribePaths` diverging from what its own
`Capabilities()` literal declares — i.e. that a hand-written extra (or
missing) path would be caught — before any later plan in this phase relies
on this guard to keep every target's paths in the table and nowhere else
(rule `84d1gfpywd`).

**Pre-mutation gate:** `git diff --quiet -- internal/agents/cursor.go` — clean.

**Mutation applied:** via `perl -0pi`, appended a hand-written extra path
(Cursor's legacy, self-heal-deleted rules file — never declared by the
table) onto `DescribePaths`'s return value in `internal/agents/cursor.go`:

```diff
--- a/internal/agents/cursor.go
+++ b/internal/agents/cursor.go
@@ -137,5 +137,5 @@ func (cursorTarget) Uninstall(loc Location) WriteResult {

 // DescribePaths is a derivation of the capability table (D-02, D-03).
 func (t cursorTarget) DescribePaths(loc Location) []string {
-	return describeDeclaredPaths(t, loc)
+	return append(describeDeclaredPaths(t, loc), filepath.Join(".cursor", "rules", "codegraph.mdc"))
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
-count=1 -run 'TestCapabilitiesTableDrivesDerivations$' -v`, exit code appended):

```
    capabilities_test.go:329: DescribePaths(global) = [/var/folders/.../.cursor/mcp.json .cursor/rules/codegraph.mdc], want set-equal to [/var/folders/.../.cursor/mcp.json]
    capabilities_test.go:329: DescribePaths(local) = [.cursor/mcp.json .cursor/rules/codegraph.mdc], want set-equal to [.cursor/mcp.json]
--- FAIL: TestCapabilitiesTableDrivesDerivations (0.01s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/antigravity/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/antigravity/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/claude/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/claude/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/codex/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/codex/local (0.00s)
    --- FAIL: TestCapabilitiesTableDrivesDerivations/cursor/global (0.00s)
    --- FAIL: TestCapabilitiesTableDrivesDerivations/cursor/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/gemini/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/gemini/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/hermes/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/hermes/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/kiro/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/kiro/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/opencode/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/opencode/local (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.139s
FAIL
exit=1
```

The failure names exactly the two subtests that diverge —
`TestCapabilitiesTableDrivesDerivations/cursor/global` and `.../cursor/local`
— while every other target's subtest (14 of 16) stays PASS, proving the
guard is scoped to the mutated target and not a whole-suite false positive.

**Revert:** `git checkout -- internal/agents/cursor.go`.

**Byte-clean proof:** `git diff --quiet -- internal/agents/cursor.go` — holds.

**Green re-run** (verbatim, exit code appended):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestCapabilitiesTableDrivesDerivations$' -v
--- PASS: TestCapabilitiesTableDrivesDerivations (0.01s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/antigravity/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/antigravity/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/claude/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/claude/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/codex/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/codex/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/cursor/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/cursor/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/gemini/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/gemini/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/hermes/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/hermes/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/kiro/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/kiro/local (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/opencode/global (0.00s)
    --- PASS: TestCapabilitiesTableDrivesDerivations/opencode/local (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/agents	0.110s
exit=0
```

**Verdict:** The set-equality guard is live and discriminating — a
hand-written path appended alongside `describeDeclaredPaths`'s own output
(the "a path in DescribePaths the table does not hold" direction D-03
names) was caught at both locations for the mutated target only, and the
mutation reverted byte-clean.

---

## Family (a2) — D-03: deleting Claude's Instructions declaration turns TestCapabilitiesMatchInstallWrites RED

**Test/guard:** `TestCapabilitiesMatchInstallWrites`
(`internal/agents/capabilities_test.go`) — the D-03 guard's other
direction: after a real `Install` call, every created/updated/unchanged
file `Install` reports must be declared by the target's `DescribePaths`.

**What are we testing, and why?** Whether this guard actually catches the
opposite divergence from Family (a1) — a path the table stops declaring
while `Install` keeps writing it (the "a path... vice versa" direction
D-03 names) — not merely that it returns green on today's tree, which a
walk that silently inspects the wrong direction would also do (rule
`84d1gfpywd`).

**Pre-mutation gate:** `git diff --quiet -- internal/agents/claude.go` — clean.

**Mutation applied:** via `perl -0pi`, deleted the `Instructions:
claudeInstructionsPath,` field from Claude's `Capabilities()` literal in
`internal/agents/claude.go` — the table stops declaring
`~/.claude/CLAUDE.md` / `.claude/CLAUDE.md`, but `Install` (untouched by
this mutation) still writes it via `upsertInstructionsEntry`:

```diff
--- a/internal/agents/claude.go
+++ b/internal/agents/claude.go
@@ -46,7 +46,6 @@ func (claudeTarget) Capabilities() Capabilities {
 		ConfigFormat: ConfigFormatJSON,
 		Hooks:        HooksClaudeJSON,
 		MCPConfig:    claudeConfigPath,
-		Instructions: claudeInstructionsPath,
 		SkillDirs: func(loc Location) ([]string, error) {
 			dir, err := claudeSkillDirPath(loc)
 			if err != nil {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
-count=1 -run 'TestCapabilitiesMatchInstallWrites$' -v`, exit code appended):

```
    capabilities_test.go:410: Install wrote/kept "/var/folders/.../.claude/CLAUDE.md", which the table does not declare: declared=[/var/folders/.../.claude.json /var/folders/.../.claude/settings.json /var/folders/.../.claude/hooks/session-nudge.sh /var/folders/.../.claude/skills/codegraph/SKILL.md /var/folders/.../.claude/skills/codegraph/.codegraph-manifest.json]
    capabilities_test.go:410: Install wrote/kept ".claude/CLAUDE.md", which the table does not declare: declared=[.mcp.json .claude/settings.json .claude/hooks/session-nudge.sh .claude/skills/codegraph/SKILL.md .claude/skills/codegraph/.codegraph-manifest.json]
--- FAIL: TestCapabilitiesMatchInstallWrites (0.02s)
    --- PASS: TestCapabilitiesMatchInstallWrites/antigravity/global (0.00s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/global (0.00s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/codex/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/cursor/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/cursor/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/gemini/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/gemini/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/hermes/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/kiro/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/kiro/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/opencode/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/opencode/local (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.124s
FAIL
exit=1
```

The failure names the exact undeclared-but-written path (`.claude/CLAUDE.md`
at both the resolved-home global path and the relative local path) for
`claude/global` and `claude/local` only, while the other 11 of 13 leaf
subtests (the ones this plan's grid runs for supported locations) stay
PASS.

**Revert:** `git checkout -- internal/agents/claude.go`.

**Byte-clean proof:** `git diff --quiet -- internal/agents/claude.go` — holds.

**Green re-run** (verbatim, exit code appended):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestCapabilitiesMatchInstallWrites$' -v
--- PASS: TestCapabilitiesMatchInstallWrites (0.01s)
    --- PASS: TestCapabilitiesMatchInstallWrites/antigravity/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/claude/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/claude/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/codex/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/cursor/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/cursor/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/gemini/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/gemini/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/hermes/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/kiro/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/kiro/local (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/opencode/global (0.00s)
    --- PASS: TestCapabilitiesMatchInstallWrites/opencode/local (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/agents	0.065s
exit=0
```

**Verdict:** The written-files-vs-declared-paths guard is live and
discriminating in the direction Family (a1) does not cover — a table
literal that stops declaring a path `Install` still writes was caught at
both of Claude's supported locations, named precisely, and the mutation
reverted byte-clean.

---

## Family (b1) — D-13: shape-based skill-dir ownership recovery turns TestOwnershipExactIdentity RED

**Test/guard:** `TestOwnershipExactIdentity` (`internal/agents/ownership_test.go`)
— the D-13 32-leaf table's foreign-codegraph-dir variant, which plants a
manifest-less SKILL.md in every `newSkillDirs` root and asserts it is
reported `kept (foreign)` and left byte-identical (D-14: ownership of a
skill directory is manifest-file PRESENCE only, never a SKILL.md's name or
content).

**What are we testing, and why?** Whether `skillDirIsForeign`'s
manifest-presence check actually catches an ownership-recovery heuristic
that widens "is this ours?" to include SKILL.md's mere presence — the same
class of authorization differential commit `242ec0a` closed for Claude's
SessionStart hooks, now applied to the shared skill-directory writer
(rule `84d1gfpywd`).

**Pre-mutation gate:** `git diff --quiet -- internal/agents/skillshared.go` — clean.

**Mutation applied:** via `perl -0pi`, widened `skillDirIsForeign`'s
manifest check in `internal/agents/skillshared.go` to also treat a bare
SKILL.md's presence as proof of ownership:

```diff
--- a/internal/agents/skillshared.go
+++ b/internal/agents/skillshared.go
@@ -85,7 +85,7 @@ func skillDirIsForeign(dir string) (bool, error) {
 	if !fileExists(dir) {
 		return false, nil
 	}
-	if fileExists(skillManifestPath(dir)) {
+	if fileExists(skillManifestPath(dir)) || fileExists(filepath.Join(dir, skillFileName)) {
 		return false, nil
 	}
 	entries, err := os.ReadDir(dir)
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
-count=1 -run 'TestOwnershipExactIdentity$' -v`, exit code appended):

```
    ownership_test.go:527: expected {.../.agents/skills/codegraph, "kept (foreign)"} in Install result, got [{Path:.../.cursor/mcp.json Action:created} {Path:.../.agents/skills/codegraph/SKILL.md Action:updated} {Path:.../.agents/skills/codegraph/.codegraph-manifest.json Action:created}]
--- FAIL: TestOwnershipExactIdentity/cursor/global/foreign-codegraph-dir (0.00s)
    ownership_test.go:527: expected {.../.agents/skills/codegraph, "kept (foreign)"} in Install result, got [{Path:.../.cursor/mcp.json Action:created} {Path:.../.agents/skills/codegraph/SKILL.md Action:updated} {Path:.../.agents/skills/codegraph/.codegraph-manifest.json Action:created}]
--- FAIL: TestOwnershipExactIdentity/cursor/local/foreign-codegraph-dir (0.00s)
    ownership_test.go:527: expected {.../.agents/skills/codegraph, "kept (foreign)"} in Install result, got [{Path:.../opencode.jsonc Action:updated} {Path:.../AGENTS.md Action:updated} {Path:.../.agents/skills/codegraph/SKILL.md Action:updated} {Path:.../.agents/skills/codegraph/.codegraph-manifest.json Action:created}]
--- FAIL: TestOwnershipExactIdentity/opencode/global/foreign-codegraph-dir (0.00s)
    ownership_test.go:527: expected {.../.agents/skills/codegraph, "kept (foreign)"} in Install result, got [{Path:.../opencode.jsonc Action:updated} {Path:.../AGENTS.md Action:updated} {Path:.../.agents/skills/codegraph/SKILL.md Action:updated} {Path:.../.agents/skills/codegraph/.codegraph-manifest.json Action:created}]
--- FAIL: TestOwnershipExactIdentity/opencode/local/foreign-codegraph-dir (0.00s)
--- FAIL: TestOwnershipExactIdentity (0.43s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.153s
FAIL
exit=1
```

The failure names exactly the four foreign-codegraph-dir leaves that share
the widened dir (`cursor/global`, `cursor/local`, `opencode/global`,
`opencode/local` — both targets write the SAME shared directory, so both
are affected by construction) while the other 28 of 32 leaves stay PASS —
each shows the foreign SKILL.md silently overwritten (`Action:updated`)
and a manifest created where none should exist, exactly the "SKILL.md
content claimed as ours" shape.

**Revert:** `git checkout -- internal/agents/skillshared.go`.

**Byte-clean proof:** `git diff --quiet -- internal/agents/skillshared.go` — holds.

**Green re-run** (verbatim, exit code appended):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestOwnershipExactIdentity$'
ok  	github.com/seanb4t/codegraph-go/internal/agents	0.153s
exit=0
```

**Verdict:** The manifest-presence-only ownership check is live and
discriminating — widening it to also accept SKILL.md's mere presence was
caught at every affected leaf, named precisely, and the mutation reverted
byte-clean.

---

## Family (b2) — D-13/D-15: the literal 242ec0a hook-ownership-by-matcher reintroduction turns TestOwnershipExactIdentity RED

**Test/guard:** `TestOwnershipExactIdentity` (`internal/agents/ownership_test.go`)
— the claude/* leaves, which reproduce commit `242ec0a`'s exact
precondition (install once so a manifest exists, hand-edit the
SessionStart `"startup"` matcher slot to hold an unrelated command,
re-install, uninstall) and assert the unrelated command survives
byte-for-byte and codegraph's own command is gone after uninstall.

**What are we testing, and why?** Whether reintroducing the LITERAL
authorization differential `242ec0a` reverted — ownership of a SessionStart
block granted by matcher name/shape rather than exact command-string
identity — is still caught by this phase's own guard, not just by the
pre-existing Claude-specific regression test. This is the D-15 "cited in
review" requirement's own positive control: the guard must fail against
the exact historical shape, not merely a shape resembling it.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/shared.go` — clean.

**Mutation applied:** via `perl -0pi`, reintroduced `writeHookEntry`'s
`isOwned` closure treating any block under the `"startup"` matcher as
owned, regardless of its own command string — the exact heuristic
`242ec0a` reverted:

```diff
--- a/internal/agents/shared.go
+++ b/internal/agents/shared.go
@@ -213,6 +213,9 @@ func writeHookEntry(path, event string, ownBlocks []any, ownCommands []string)
 		if !ok {
 			return false
 		}
+		if m, _ := obj["matcher"].(string); m == "startup" {
+			return true
+		}
 		blockHooks, ok := obj["hooks"].([]any)
 		if !ok {
 			return false
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
-count=1 -run 'TestOwnershipExactIdentity$' -v`, exit code appended):

```
=== RUN   TestOwnershipExactIdentity/claude/global/clean
    ownership_test.go:527: settings.json missing after uninstall — the unrelated startup hook should have survived
=== RUN   TestOwnershipExactIdentity/claude/global/foreign-codegraph-dir
    ownership_test.go:527: settings.json missing after uninstall — the unrelated startup hook should have survived
=== RUN   TestOwnershipExactIdentity/claude/local/clean
    ownership_test.go:527: settings.json missing after uninstall — the unrelated startup hook should have survived
=== RUN   TestOwnershipExactIdentity/claude/local/foreign-codegraph-dir
    ownership_test.go:527: settings.json missing after uninstall — the unrelated startup hook should have survived
--- FAIL: TestOwnershipExactIdentity/claude/global/clean (0.04s)
--- FAIL: TestOwnershipExactIdentity/claude/global/foreign-codegraph-dir (0.06s)
--- FAIL: TestOwnershipExactIdentity/claude/local/clean (0.00s)
--- FAIL: TestOwnershipExactIdentity/claude/local/foreign-codegraph-dir (0.01s)
--- FAIL: TestOwnershipExactIdentity (0.43s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.153s
FAIL
exit=1
```

The failure names exactly the four `claude/*` leaves — the mutation
widened `isOwned` so the unrelated `"startup"` block was claimed and
overwritten by codegraph's fresh registration, and once codegraph's own
entry was later removed on uninstall, nothing remained in `settings.json`
at all (the unrelated hook, which should have survived untouched, was
silently destroyed) — while the other 28 of 32 leaves stay PASS.

**Revert:** `git checkout -- internal/agents/shared.go`.

**Byte-clean proof:** `git diff --quiet -- internal/agents/shared.go` — holds.

**Green re-run** (verbatim, exit code appended):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestOwnershipExactIdentity$'
ok  	github.com/seanb4t/codegraph-go/internal/agents	0.153s
exit=0
```

**Verdict:** The exact-command-string hook ownership check is live and
discriminating against the LITERAL historical vulnerability shape — every
`claude/*` leaf caught the widened `isOwned` closure, and the mutation
reverted byte-clean.

---

## Summary

Every guard demonstrated in this log was confirmed RED against a
confirmed-applied, byte-cleanly-reverted mutation before being trusted:

| Family | Guard | Mutation | RED confirmed | Reverted clean |
|---|---|---|---|---|
| (a1) | `TestCapabilitiesTableDrivesDerivations` | extra hand-written path appended to `cursor.go`'s `DescribePaths` | yes (names `cursor/global`, `cursor/local`) | yes |
| (a2) | `TestCapabilitiesMatchInstallWrites` | `Instructions` field deleted from Claude's `Capabilities()` literal | yes (names `.claude/CLAUDE.md` undeclared, both locations) | yes |
| (b1) | `TestOwnershipExactIdentity` (foreign-codegraph-dir) | `skillDirIsForeign` widened to also accept a bare SKILL.md as proof of ownership | yes (names `cursor/{global,local}`, `opencode/{global,local}`) | yes |
| (b2) | `TestOwnershipExactIdentity` (claude/*) | `writeHookEntry`'s `isOwned` widened to claim any `"startup"`-matcher block (the literal 242ec0a shape) | yes (names all four `claude/*` leaves) | yes |

`git status --porcelain internal/agents/` is empty at the end of every
plant above — no production source file was left modified by any
mutation; each was reverted byte-clean before the commit that records it
in this log.
