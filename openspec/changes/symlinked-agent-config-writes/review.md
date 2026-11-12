---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-12T22:14:30+00:00
head: a12602e6505f0b0752771a84cbcbf3658b01c87d
lenses:
  spec-lint: ok
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: not-applicable
  invariants: ok
---
# Review

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | The design's list of user-file edits that bypass recordFile is not complete at head. The OpenCode Install also edits a user's config in opencodeSweepStaleAppData with removeOpencodeEntry and records nothing. No decision or task routes it through resolveUserFile, so at global scope a linked stale opencode.jsonc is still replaced by a regular file, against the global symlink requirement. | design.md > Context — "Some steps append to `result.Files` and `result.Errors` directly. They are the configuration edits in `codex.go` and `he" | Name the stale APPDATA sweep in the Context and in D6 as a user-file edit. In task 3.3, resolve stalePath with resolveUserFile at global scope before removeOpencodeEntry. Otherwise, exclude the best-effort sweep explicitly under Non-Goals. |
| app-2 | approach | should-fix | The proposal says #292 already added the parse rule in code for two writers. At head, both writers still fall back to an empty object on unparseable content and overwrite the file. The design itself says #303 is not on main yet. | proposal.md > What Changes — "#292 added this rule in code for two writers." | Reword the proposal bullet: #303, the fix for #292, adds the rule for the two writers. This change records it in the spec and depends on #303. |
| app-3 | approach | should-fix | The Context says a JSON parse error carries no path. At head, the one JSON parse error a helper returns, from readJSONFileStrict (used by the hook and permission writers), already names the path. D4's rule of adding the real location for a linked path then puts the real location in the line twice for those writers. | design.md > Context — "The TOML conflict error and a JSON parse error carry none." | State which JSON parse error carries no path, meaning the one from #303 or readJSONFileStrict. In D4, say whether the funnel adds the real location when the helper error already names it. |
| fit-1 | fit | should-fix | After the change the repository holds two resolvers for a path that does not exist yet: realpath.Resolve and resolveExistingPrefix in internal/mcp. The proposal promises one, and the design keeps the second with 'its one caller' as the only reason. | internal/mcp/tools.go:111 — "func resolveExistingPrefix(abs string) (string, error) {" | Point confineToRepoRoot at realpath.Resolve and delete resolveExistingPrefix, or state in D2 the behavior difference that forces two resolvers and name a ticket that collapses them. |
| fit-2 | fit | should-fix | The change creates realpath.Within as the shared 'inside this directory' test but leaves two inline copies of the same filepath.Rel check in resolveSourcePath; the stated reason is scope, not a difference in behavior. | internal/query/node.go:59 — "if rel == ".." \|\| strings.HasPrefix(rel, ".."+string(filepath.Separator)) {" | Call realpath.Within at both sites of resolveSourcePath in task 2.5, or name a ticket that does it. |
| fit-3 | fit | should-fix | Task 3.3 wraps four identical upsertInstructionsEntry steps in recordUserFile one by one, while D7 collapses only the uninstall side into removeSharedInstructions. | internal/agents/claude.go:481 — "fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())" | Add an install counterpart beside removeSharedInstructions that routes upsertInstructionsEntry through recordUserFile, and call it from the four Install methods in task 3.3. |
| fit-4 | fit | should-fix | The Claude and Antigravity legacy steps both drop a legacy MCP entry and record it only on ActionRemoved, yet the design routes one through the funnel as an edit function and the other through a bare resolveUserFile call. | internal/agents/claude.go:462 — "if fr, err := removeMcpEntry(legacyPath); err != nil {" | Give both legacy-entry removals one helper that resolves through the funnel and records only a removal, and name it in D4 and tasks 3.3 and 3.4. |
| fit-5 | fit | should-fix | recordUserFile always ends in recordFile, but three of the direct-record steps D4 turns into edit functions append a result only on ActionRemoved; D4 names neither that pattern nor how the funnel keeps it. | design.md > D4. One funnel records a user's file under the configured path — "It replaces the `FileResult.Path` of the helper with the configured path, and ends in `recordFile`." | State in D4 how an edit function asks the funnel to record nothing for a step that changed nothing, such as a skip result, or state that these steps now report not-found and list them. |
| inv-1 | invariants | should-fix | Three requirements say a refresh exits non-zero, but the refresh after codegraph upgrade reports failures as a warning and exits zero (D-07), and no task changes that. | specs/agent-install/spec.md > Requirement: A local install refuses a file outside the project — "an install or a refresh SHALL report an error" | Either scope the exit-code sentence of the three requirements (outside-project refusal, loop, unparseable MCP file) to an install and an uninstall, and state that the refresh after upgrade reports the error as a warning, as the existing ephemeral-path requirement does; or add a task that changes upgrade's D-07 rule and its test. |
| inv-2 | invariants | should-fix | D4 says a non-symlink path keeps today's error text, but D1 hands helpers the resolved absolute path, and helpers embed that path in their errors. | design.md > D4. One funnel records a user's file under the configured path — "For a path that is not a symlink, the error text is the same as today." | Restate the D4 claim as 'the error line starts with the configured path' (a local .mcp.json parse error will now name the absolute real path too), or make TestRecordUserFile_UnlinkedLocalErrorKeepsItsText drive writeMcpEntry on an unparseable relative .mcp.json and compare against the text at head instead of a stub edit. |
| inv-3 | invariants | should-fix | Task 3.4 runs the Antigravity legacy removal and migration write on resolved paths outside the funnel, so their FileResults name the real location, not the configured path. | specs/agent-install/spec.md > Requirement: A global agent file is edited through a symlink — "The result SHALL name the configured path." | In D4 and task 3.4, record both migration steps under the configured unified and legacy paths (overwrite fr.Path, or route each step through recordUserFile as an edit function), and make the two new migration tests assert the reported path is the configured link path. |
| inv-4 | invariants | should-fix | The global opencode Install's stale APPDATA sweep writes removeOpencodeEntry through fsatomic at an unresolved path, so a symlink there is replaced; no task routes it. | specs/agent-install/spec.md > Requirement: A global agent file is edited through a symlink — "The symlink SHALL stay as it was." | Either name opencodeSweepStaleAppData in task 3.3 and resolve stalePath with resolveUserFile(stalePath, LocationGlobal) before the call, or list it in the proposal's out-of-scope section with the reason (APPDATA is unset on macOS and Linux). |
| inv-5 | invariants | should-fix | The cursor and kiro legacy instruction files are deleted with os.Remove on the configured path, so at local scope a link or a linked directory leading outside is followed or removed. | specs/agent-install/spec.md > Requirement: A global agent file is edited through a symlink — "An agent file is an agent configuration file or an instruction file." | Decide in D6 whether .cursor/rules/codegraph.mdc and .kiro/steering/codegraph.md are owned files; if so, list them in the proposal's owned-file list and narrow the spec's 'agent file' definition to exclude them; if not, route the two removals through resolveUserFile with the uninstall rule (keep and note when outside). |
| inv-6 | invariants | should-fix | D3 refuses the file when localProjectRoot fails, but no planned test passes that null case. | design.md > D3. `resolveUserFile` applies the scope rule — "If `os.Getwd` or `filepath.EvalSymlinks` fails, `localProjectRoot` returns the error, and `resolveUserFile` refuses the " | Add TestResolveUserFile_LocalRootErrorRefuses to task 3.1: chdir into a temp dir, remove it so os.Getwd fails, call resolveUserFile at local scope, and assert an error and that no file was written; or inject localProjectRoot through a package variable. |
| inv-7 | invariants | should-fix | D7 compares both sides by real location, but both planned tests put the link on the uninstalling agent's own path; no test puts it on the other agent's path. | design.md > D7. Two agents that share one file are found by real location — "`instructionsRequestedElsewhere` compares paths with `sameRealPath`." | Add TestUninstall_Codex_SectionKeptForGeminiThroughLink to task 3.5: GEMINI.md links to AGENTS.md, local codex and gemini installed, uninstall --target codex; assert AGENTS.md still holds the section and the result reports kept. |
| inv-8 | invariants | should-fix | The CR-02 rule (keep the legacy entry and write no marker unless the unified write succeeded) has no planned test for the new failure mode where resolving the unified path fails. | internal/agents/antigravity.go:232 — "if migrationOK {" | In task 3.4, state that a resolve error on either path sets migrationOK to false before the legacy removal. Add TestAntigravity_MigrationUnifiedLoopKeepsLegacyEntryAndNoMarker: the unified path is a two-link loop and the legacy file holds the entry. Assert an error that names the unified path, a legacy entry that is unchanged, and no .migrated marker. |
| spe-1 | spec-lint | should-fix | The requirement on an MCP configuration file that does not parse matches tasks.md only through identifier stems, with no task line naming two of its distinctive words. | specs/agent-install/spec.md > Requirement: An MCP configuration file that does not parse is not written — "An MCP configuration file that does not parse is not written" | Name the requirement's distinctive words in the task 4.4 line, for example 'MCP configuration file that does not parse', or cite the requirement by its heading. |
| inv-9 | invariants | note | An install followed by an uninstall through a link should leave the target's bytes as they were and keep the link, for any sibling content. | specs/agent-install/spec.md > Requirement: An uninstall keeps a symlink — "An uninstall SHALL edit the file that a symlink at an agent file leads to. The symlink SHALL stay as it was." | Consider a pgregory.net/rapid property over random sibling MCP servers and instruction text: install then uninstall at global scope through a link, and assert the target's bytes are unchanged and the link stays. Model it on internal/watch/watcher_property_test.go. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A configuration file kept in a dotfile manager | 4.1 | covered |  |
| A chain of two links | 4.1 | covered |  |
| A link to a file that does not exist yet | 4.1 | covered |  |
| A relative link under a linked directory | 4.1 | covered |  |
| The file keeps its permission bits | 4.1 | covered |  |
| Uninstall keeps the link | 4.1 | covered |  |
| Uninstall that empties the file keeps the link | 4.1 | covered |  |
| Uninstall of one of two agents that share a linked file | 3.5 | covered |  |
| One instruction file linked to another in the project | 4.2 | covered |  |
| A file inside the project whose name starts with two dots | 4.2 | covered |  |
| An instruction file linked outside the project | 4.2 | covered |  |
| A configuration file linked outside the project | 4.2 | covered |  |
| A configuration file linked outside that already holds the entry | 4.2 | covered |  |
| A link that leaves the project through a linked directory | 4.2 | covered |  |
| A link whose target climbs out of a linked directory | 4.2 | covered |  |
| A file under a directory that is linked outside the project | 4.2 | covered |  |
| Uninstall leaves a link that leads outside the project | 4.2, 4.5 | covered |  |
| Two links that point at each other on install | 4.3 | covered |  |
| Two links that point at each other on uninstall | 4.3 | covered |  |
| A file with a syntax error | 1.1 | covered |  |
| An OpenCode file with a syntax error | 1.1 | covered |  |
| An empty file | 1.1 | covered |  |
| A link to a file with a syntax error | 4.4 | covered |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: D3 says cursorTarget.Install keeps its present fallback to the unresolved directory on an error. At head, a failed os.Getwd falls back to ${workspaceFolder} (cursor.go:113-121), not to a directory. With localProjectRoot returning only (string, error), the design does not say how cursor gets the unresolved cwd when EvalSymlinks fails.
- approach: Task 4.4 may change writeOpencodeEntry for an empty or whitespace file, which D8 does not cover ('This change adds only the scenario through a link'). At head, writeOpencodeEntry already treats whitespace-only content as {} (opencode.go:135-137), so the conditional is likely a no-op.
- approach: Verified at head: fsatomic.WriteFile temp-file-plus-rename and os.Stat mode copy; os.Remove sites shared.go:490, shared.go:750 and codex.go:314; fileExists is os.Stat; the direct-append sites at claude.go:460-467 and 636-647, codex.go:302, hermes.go:407 and antigravity.go:210-250; codex.go:332 and opencode.go:346 ActionKept duplication; claude.go:711 and gemini.go:158 without the shared-instructions gate; resolveSkillDir bound 8 with textual join; six sameSkillDir test calls; isWithin and resolveExistingPrefix in mcp/tools.go; inline Rel in query/node.go; uninstall defaults to all targets; rapid v1.3.0 in go.mod; locktest and watcher_property_test exist.
- approach: Verified in Go 1.26.11 source: filepath.EvalSymlinks gives up after 255 links (path/filepath/symlink.go:100), and os.Root forbids absolute symlinks (os/root.go:43), so the os.Root rejection in D3 holds.
- approach: The three #303 tests that task 1.1 names do not exist at head. That is expected, since #303 is not on main.
- fit: openspec/config.yaml context holds no library-preference rows for this repository; only the generic rows apply.
- fit: The design's count of 'inside' test copies (mcp and query) matches the repository; internal/uiserver delegates to query.ValidateRepoRelativePath.
- fit: D2's new test-helper package realpathtest follows internal/graphstore/locktest, and its generator is not a copy of the watcher generator, which draws directories only.
- fit: D4's errOutsideProject follows the wrapped-sentinel pattern of errTOMLTableConflict, and recordUserAction mirrors recordAction.
- fit: D8 has no component to compare: #303 is not on main at head, so writeMcpEntry's parse rule could not be read.
- fit: The os.Remove calls at cursor.go:106 and kiro.go:126 delete codegraph-owned legacy files (codegraph.mdc, codegraph.md), consistent with D6 leaving them out of the funnel.
- invariants: I1: A global agent file behind a symlink is edited at its real location and the link stays \| enforced at none at head (fsatomic.WriteFile renames over the link, fsatomic.go:65); plan D1 plus recordUserFile, task 3.3 \| boundary test none at head; plan 4.1 (chain, missing target, relative link under linked dir, mode 0640)
- invariants: I2: The real file keeps its permission bits \| enforced at fsatomic.go:58 os.Stat(path) follows the resolved path \| boundary test planned TestInstall_GlobalConfigSymlinkKeepsTargetMode (0640)
- invariants: I3: An uninstall that empties a linked file deletes the target and keeps the link \| enforced at shared.go:490, shared.go:750, codex.go:314 once given the resolved path (D1) \| boundary test planned 4.1, the last three uninstall tests
- invariants: I4: At local scope no user file is written whose real location is outside the project \| enforced at planned resolveUserFile with realpath.Within \| boundary tests planned 4.2 (linked dir, climbs out, .claude linked) and property 3.2; gaps are inv-4, inv-5 and inv-6
- invariants: I5: Within(root, p) is false exactly for '..' or a '../' prefix, so '..notes.md' is inside \| enforced at internal/mcp/tools.go:102 isWithin \| boundary tests planned 2.2 (..notes.md, ../notes.md, root itself). Probe under the handoff dir confirmed '..notes.md' inside, '../notes.md' outside, root inside, '/pq/x' outside of '/p', and '/P/x' outside of '/p' (case-sensitive text compare)
- invariants: I6: Resolve agrees with the OS, including '..' after a link \| enforced at planned realpath.Resolve \| boundary tests planned 2.2 and property 2.4. Probe: head's resolveSkillDir maps AGENTS.md -> sub/../x.md (sub -> outside/a) to proj/x.md, while the OS reaches outside/x.md. This confirms the D2 example and shows head's resolver cannot be reused
- invariants: I7: Resolve counts at most 8 link hops in the walk \| enforced at planned Resolve; head's maxSymlinkResolveDepth=8 also counts missing parents (probe: 10 missing dirs fails at head) \| boundary tests planned 8 hops resolve, 9 fail, 10 missing dirs resolve. No existing test relies on the old counting (no depth or cycle test in skillshared_test.go)
- invariants: I8: A loop or unresolvable chain writes nothing and exits non-zero \| enforced at planned Resolve error through the funnel \| boundary tests planned 4.3 (claude install, codex and hermes uninstall). The exit-code part does not hold for the refresh (inv-1)
- invariants: I9: The outside-project check runs before any read, so the unchanged short-circuit cannot skip it \| enforced at planned recordUserFile (D5) \| boundary test planned TestInstall_LocalConfigSymlinkOutsideWithTheEntryIsRefused
- invariants: I10: A local uninstall leaves an outside file, reports kept, and exits zero \| enforced at planned recordUserFile opUninstall \| boundary tests planned 3.1, 4.2 and 4.5
- invariants: I11: An MCP JSON/JSONC file that does not parse is not written; an empty or whitespace-only file is written \| enforced at none on head (readJSONFile shared.go:70-74 falls back to an empty map); on branch fix/install-unparseable-config by readJSONFileForWrite and writeOpencodeEntry. Task 1.1 names the guard \| boundary tests on that branch plus planned 4.4 (empty and whitespace opencode)
- invariants: I12: Two agents that share a real instruction file keep the section on uninstall of one \| enforced at shared.go:802 instructionsRequestedElsewhere (text compare at head), planned sameRealPath \| boundary tests planned 3.5 with the link only on self's side (inv-7)
- invariants: I13: The Antigravity migration never drops the legacy entry or writes the marker unless the unified write succeeded (CR-02) \| enforced at antigravity.go:208-256 migrationOK \| boundary test TestAntigravity_Install_UnifiedWriteFailure_PreservesLegacyEntryNoMarker; none for a resolve failure (inv-8)
- invariants: Note: the resolve bound differs by target existence. EvalSymlinks follows up to 255 links, the walk 8, so a 9-link chain resolves when its last target exists and fails when it is missing. The spec sentence 'If that file does not exist, the install SHALL create it' names no bound, and D2 fixes 8 for the missing case.
- invariants: Note: realpath.Within compares text. On a case-insensitive macOS volume, an absolute link target that names the project in another case is refused (fail-closed).
- invariants: Note: the inside-project rule treats .git/ as inside. A committed AGENTS.md -> .git/config link gets the codegraph section appended at local scope. The text is fixed codegraph content, but it can corrupt the file. This belongs to the approach lens; it is recorded here only as the invariant boundary.
- invariants: Note: the Claude legacy ./.claude.json step (claude.go:464) and the Antigravity legacy removal (antigravity.go:236) record only ActionRemoved today. D4's funnel ends in recordFile, which records every action, so an edit function must keep that filter or the output gains a not-found line. For claude the install-local golden in internal/cli/plain_golden_test.go:341 guards this; the Antigravity step has no golden.
- invariants: Note: refresh after upgrade runs Install at LocationLocal with the upgrade's working directory as the project (internal/cli/upgrade.go:95, manifest.go:314). The new refusal there surfaces as a warning that says to run codegraph install, which will fail the same way.
- spec-lint: Every requirement in specs/agent-install/spec.md has at least one scenario, and every scenario has WHEN and THEN lines and a GIVEN line.
- spec-lint: All THEN steps name an observable (a file state, a link target, an output error, an exit code). No WHEN or GIVEN step was found without a concrete input or state.
- spec-lint: No BREAKING marker appears in proposal.md or in the delta spec, so no breaking-mismatch. The Breaking kind in task 6.2 is a changie fragment kind.
- spec-lint: Build targets named in tasks.md exist in Taskfile.yml: test, lint, docs:cli:drift and changie.
- spec-lint: design.md has no open-questions section, so no assumed-answer check applies.
- spec-lint: A directory-wide search of the change root also matched text in review.md. I did not read that file and did not use that text in any finding.
- test-coverage: Task 4.2 lacks a comma between TestInstall_LocalSymlinkToDotDotPrefixedNameIsWrittenThrough and TestUninstall_LocalSymlinkOutsideProjectIsKeptWithANote; cosmetic.
- test-coverage: The non-zero exit of the refusal and loop scenarios is asserted at CLI level only for one local-outside case (4.5); other scenarios assert the recorded error in internal/agents. Check at implementation that the agent-level tests assert the error that drives the exit code.
- test-coverage: design.md was not read; the matrix rests on spec.md and tasks.md.
