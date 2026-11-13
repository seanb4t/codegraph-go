---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-13T01:23:06+00:00
head: f1b3a26e314c8d12aaa42f791309f234211f5dcb
lenses:
  spec-lint: not-applicable
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: ok
  invariants: ok
---
# Review

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | D3 step 1 says resolveUserFile calls realpath.Resolve. At head it calls realpath.ResolveFile, which D2 also names, so D3 contradicts both D2 and the code. | design.md > D3. `resolveUserFile` applies the scope rule — "1. Call `realpath.Resolve(path)`. If it fails, return the error. A loop fails here." | Change D3 step 1 to call realpath.ResolveFile, and say that a trailing separator that names a missing directory fails there too. |
| app-2 | approach | should-fix | The D4 paragraph says the Antigravity migration reuses its resolved legacy path for the removeMcpEntry call. At head that call goes through recordUserFile with the configured legacy path, which resolves the path again. Only the later settled bullet describes this. | design.md > D4. One funnel records a user's file under the configured path — "The Antigravity migration also resolves its legacy path with `resolveUserFile`, and uses the result for its existence te" | Change that D4 sentence so the resolved legacy path is used for the existence test and the read, and the removal goes through the funnel with the configured path, as the settled bullet says. Or delete the sentence and keep only the bullet. |
| app-3 | approach | should-fix | realpathtest also exports TempDir and Symlink, and tests in graphstore, watch and wtindex now import it. Task 2.3 gives the package one generator, and neither the design nor the proposal Impact names those three packages. | tasks.md > 2. Shared path package — "It holds one `rapid` generator of a random graph of directories, files and links" | Record in design.md D2 (or as a deviation note) that realpathtest also holds TempDir and Symlink, and that the graphstore, watch and wtindex tests use them. Otherwise move that adoption to a separate chore PR. |
| fit-1 | fit | should-fix | The change adds realpath.Resolve, a second resolver that starts at the longest existing part of a path, beside resolveExistingPrefix in internal/mcp. The reason in D4 for keeping both contradicts itself: it says a missing rest has no links, then says a dangling link is an entry that exists. filepath.EvalSymlinks reports a dangling link as missing, so resolveExistingPrefix joins the link name on as text, which is exactly the case realpath.Resolve was written to follow. | internal/mcp/tools.go:104 — "func resolveExistingPrefix(abs string) (string, error) {" | Have confineToRepoRoot call realpath.Resolve and delete resolveExistingPrefix. If the #299 review must stay untouched, give D4 a reason that holds for a dangling link in the path, plus a tracked ticket for the collapse. |
| fit-2 | fit | should-fix | TestResolveUserFile_LocalNeverLeavesTheProject runs the inside-the-project test two ways. Its refusal branch spells the test out inline with strings.HasPrefix, and its accept branch calls realpath.Within, the shared form that this change creates. | internal/agents/userfile_test.go:230 — "if evalErr == nil && (reached == project \|\| strings.HasPrefix(reached, project+string(filepath.Separator))) {" | Use one oracle in both branches. Either call realpath.Within at line 230, or, if the oracle has to be independent of the code under test, use the inline form at line 251 as well. |
| inv-1 | invariants | should-fix | For a link with an empty target (macOS accepts os.Symlink("", link)), ResolveFile returns the directory that holds the link with no error, and the operating system refuses to make a file through it (ENOENT). The generator never draws an empty target. The property test uses filepath.EvalSymlinks as its oracle for a path that exists, and that oracle gives the same wrong answer. A probe showed the effect: for .codex/config.toml -> "", the Codex write makes its temporary file in the project root and then fails to rename it over .codex with 'file exists'. Nothing is written, but the error does not name the cause. | internal/realpath/realpath.go:72 — "real, err = filepath.EvalSymlinks(path)" | Make ResolveFile return an error when the location it resolves is an existing directory, since a user's file is never a directory. Add TestResolveFile_EmptyLinkTargetIsAnError, which skips where os.Symlink("", ...) fails (Linux) and on macOS asserts an error and that os.OpenFile with O_CREATE also fails. |
| ctl-not-run | test-coverage | should-fix | covered row 'An instruction file linked outside the project' has no verification run | review > not-run — "internal/agents/symlink_install_test.go: TestInstall_LocalInstructionSymlinkOutsideProjectIsRefused; internal/cli/instal" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Uninstall leaves a link that leads outside the project' has no verification run | review > not-run — "internal/agents/symlink_install_test.go: TestUninstall_LocalSymlinkOutsideProjectIsKeptWithANote; internal/cli/install_t" | run the covering test and record it in verification-runs.txt |
| ctl-not-run | test-coverage | should-fix | covered row 'Two links that point at each other on install' has no verification run | review > not-run — "internal/agents/symlink_install_test.go: TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing; internal/cli/install_tes" | run the covering test and record it in verification-runs.txt |
| tes-1 | test-coverage | should-fix | The 'An empty file' scenario runs writeMcpEntry directly; no test runs gemini install on an empty settings.json. | internal/agents/shared_test.go:255 — "result, err := writeMcpEntry(path, func() any { return map[string]any{"command": "codegraph"} })" | Add TestGemini_Install_EmptySettingsIsWritten in internal/agents/gemini_test.go: empty ~/.gemini/settings.json, geminiTarget{}.Install(LocationGlobal, ...), assert no errors and the file holds the codegraph entry. |
| fit-3 | fit | note | Inside the funnel, the edit closures of the Codex and Hermes installs share one shape. Each records whether the file existed, reads it, transforms it, returns unchanged if nothing changed, writes atomically, then returns created or updated. Only the transform differs, and the Codex closure also runs a conflict check first. | internal/agents/codex.go:196 — "existed := fileExists(real)" | Optionally add a parameter func(existing string) (string, error) to one splice-and-write helper that both closures call. |
| inv-2 | invariants | note | Within decides every local-scope refusal and the MCP confinement, over an unbounded domain of paths. Only four examples test it. The design's claim that filepath.IsLocal on the relative path gives the same answer is not tested. | internal/realpath/realpath.go:23 — "return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))" | Add a pgregory.net/rapid property test in internal/realpath/realpath_test.go, modeled on TestResolve_AgreesWithTheOperatingSystem in the same file. It draws relative paths from the elements a, .., ..a, a.. and . and asserts that Within(root, filepath.Join(root, rel)) equals (rel cleans to "." or filepath.IsLocal(rel)). |
| tes-2 | test-coverage | note | The 'OpenCode file with a syntax error' scenario is proved at helper level (writeOpencodeEntry), not through opencodeTarget.Install; the scenario names the install command and non-zero exit. | internal/agents/opencode_test.go:88 — "_, err := writeOpencodeEntry(path, "/usr/local/bin/codegraph")" | Optionally add an opencodeTarget{}.Install(LocationGlobal) case asserting result.Errors names the file and bytes are unchanged. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A configuration file kept in a dotfile manager | internal/agents/symlink_install_test.go: TestInstall_GlobalConfigSymlinkIsWrittenThrough | covered | pass |
| A chain of two links | internal/agents/symlink_install_test.go: TestInstall_GlobalConfigSymlinkChain | covered | pass |
| A link to a file that does not exist yet | internal/agents/symlink_install_test.go: TestInstall_GlobalConfigSymlinkToMissingTarget | covered | pass |
| A relative link under a linked directory | internal/agents/symlink_install_test.go: TestInstall_GlobalRelativeLinkUnderLinkedDirectory | covered | pass |
| The file keeps its permission bits | internal/agents/symlink_install_test.go: TestInstall_GlobalConfigSymlinkKeepsTargetMode | covered | pass |
| Uninstall keeps the link | internal/agents/symlink_install_test.go: TestUninstall_GlobalConfigSymlinkIsKept | covered | pass |
| Uninstall that empties the file keeps the link | internal/agents/symlink_install_test.go: TestUninstall_GlobalInstructionSymlinkWithOnlyTheSectionKeepsLink | covered | pass |
| Uninstall of one of two agents that share a linked file | internal/agents/symlink_install_test.go: TestUninstall_Gemini_LinkedInstructionFileKeepsTheSectionCodexUses | covered | pass |
| One instruction file linked to another in the project | internal/agents/symlink_install_test.go: TestInstall_LocalInstructionSymlinkInsideProjectIsWrittenThrough | covered | pass |
| A file inside the project whose name starts with two dots | internal/agents/symlink_install_test.go: TestInstall_LocalSymlinkToDotDotPrefixedNameIsWrittenThrough | covered | pass |
| An instruction file linked outside the project | internal/agents/symlink_install_test.go: TestInstall_LocalInstructionSymlinkOutsideProjectIsRefused; internal/cli/install_test.go: TestInstall_LocalSymlinkOutsideProjectExitsNonZero | covered | not-run |
| A configuration file linked outside the project | internal/agents/symlink_install_test.go: TestInstall_LocalConfigSymlinkOutsideProjectIsRefused | covered | pass |
| A configuration file linked outside that already holds the entry | internal/agents/symlink_install_test.go: TestInstall_LocalConfigSymlinkOutsideWithTheEntryIsRefused | covered | pass |
| A link that leaves the project through a linked directory | internal/agents/symlink_install_test.go: TestInstall_LocalSymlinkOutsideThroughLinkedDirectoryIsRefused | covered | pass |
| A link whose target climbs out of a linked directory | internal/agents/symlink_install_test.go: TestInstall_LocalSymlinkThatClimbsOutOfALinkedDirectoryIsRefused | covered | pass |
| A file under a directory that is linked outside the project | internal/agents/symlink_install_test.go: TestInstall_LocalUserFilesUnderDirectoryLinkedOutsideAreRefused | covered | pass |
| Uninstall leaves a link that leads outside the project | internal/agents/symlink_install_test.go: TestUninstall_LocalSymlinkOutsideProjectIsKeptWithANote; internal/cli/install_test.go: TestUninstall_LocalSymlinkOutsideProjectExitsZeroWithANote | covered | not-run |
| Two links that point at each other on install | internal/agents/symlink_install_test.go: TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing; internal/cli/install_test.go: TestInstallAndUninstall_SymlinkLoopExitsNonZero | covered | not-run |
| Two links that point at each other on uninstall | internal/agents/symlink_install_test.go: TestUninstall_SymlinkLoopReportsAnError | covered | pass |
| A file with a syntax error | internal/agents/gemini_test.go: TestGemini_Install_UnparseableSettingsReportsAnErrorAndKeepsTheFile | covered | pass |
| An OpenCode file with a syntax error | internal/agents/opencode_test.go: TestOpencode_UnparseableConfigIsRefusedAndLeftAlone | covered | pass |
| An empty file | internal/agents/shared_test.go: TestMcpEntry_EmptyFileIsWritten | partial | pass |
| A link to a file with a syntax error | internal/agents/symlink_install_test.go: TestInstall_GlobalConfigSymlinkToUnparseableTargetIsUntouched | covered | pass |

### Not run

- An instruction file linked outside the project: no verification run names this row
- Uninstall leaves a link that leads outside the project: no verification run names this row
- Two links that point at each other on install: no verification run names this row

## Verification runs

- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalConfigSymlinkIsWrittenThrough$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalConfigSymlinkChain$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalConfigSymlinkToMissingTarget$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalRelativeLinkUnderLinkedDirectory$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalConfigSymlinkKeepsTargetMode$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestUninstall_GlobalConfigSymlinkIsKept$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestUninstall_GlobalInstructionSymlinkWithOnlyTheSectionKeepsLink$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestUninstall_Gemini_LinkedInstructionFileKeepsTheSectionCodexUses$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalInstructionSymlinkInsideProjectIsWrittenThrough$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalSymlinkToDotDotPrefixedNameIsWrittenThrough$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalInstructionSymlinkOutsideProjectIsRefused$ -count=1 -v`
- exit 0: `go test ./internal/cli/ -run ^TestInstall_LocalSymlinkOutsideProjectExitsNonZero$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalConfigSymlinkOutsideProjectIsRefused$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalConfigSymlinkOutsideWithTheEntryIsRefused$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalSymlinkOutsideThroughLinkedDirectoryIsRefused$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalSymlinkThatClimbsOutOfALinkedDirectoryIsRefused$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_LocalUserFilesUnderDirectoryLinkedOutsideAreRefused$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestUninstall_LocalSymlinkOutsideProjectIsKeptWithANote$ -count=1 -v`
- exit 0: `go test ./internal/cli/ -run ^TestUninstall_LocalSymlinkOutsideProjectExitsZeroWithANote$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing$ -count=1 -v`
- exit 0: `go test ./internal/cli/ -run ^TestInstallAndUninstall_SymlinkLoopExitsNonZero$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestUninstall_SymlinkLoopReportsAnError$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestGemini_Install_UnparseableSettingsReportsAnErrorAndKeepsTheFile$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestOpencode_UnparseableConfigIsRefusedAndLeftAlone$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestMcpEntry_EmptyFileIsWritten$ -count=1 -v`
- exit 0: `go test ./internal/agents/ -run ^TestInstall_GlobalConfigSymlinkToUnparseableTargetIsUntouched$ -count=1 -v`

## Dropped

None.

## Lens notes

- approach: All seven requirements trace to decisions at head: global edit through a link (D1, D4), uninstall keeps a link (D1, D7), local inside (D3), local refusal and uninstall keep (D3, D5), loop (D2, D3), and parse rule (D8 with the helpers that #303 added).
- approach: Each user-file helper call in claude, codex, codex_pretooluse, cursor, gemini, hermes, kiro, opencode and antigravity goes through recordUserFile, recordUserAction, installMcpEntry, installInstructions or removeSharedInstructions. The owned-file writers keep recordFile, as D6 says.
- approach: D3 rejects os.Root because it refuses absolute links. The Go 1.26.11 src/os/root.go line 43 says 'Symbolic links must not be absolute.', so the reason holds.
- approach: D2 says filepath.EvalSymlinks stops after 255 links. Go 1.26.11 path/filepath/symlink.go line 100 confirms it.
- approach: D4 says 'The implementation settled four points', but 13 bullets follow. The count is wrong, but it cites no code.
- approach: Codex install in Keep mode reads hooks.json with hasOwnHookBlock on the configured path (codex.go:262). On a read error, including a symlink loop, it does nothing and reports no error. That matches the posture before this change. It is not counted as a gap in the loop requirement, because Keep does not edit the file then.
- approach: recordUserFile sends the configured path to edit only when filepath.Abs(path) equals the real location. A relative local path whose working directory sits under a link (for example macOS /tmp) therefore sends edit the real absolute path, and a helper error then names that absolute path instead of the relative one. That is within D4's 'no link in the path' wording, but the wording is loose.
- approach: Context line numbers are pinned to a12602e6 by design.md and were not checked against head.
- fit: The prompt names no path for the repository rows of the library-preference table, so only the generic rows were applied.
- fit: resolveSourcePath in internal/query/node.go keeps three inline copies of the inside test (lines 45, 59 and 74). The change moves isWithin rather than adding a copy, and D4 records that the user cut the query move from scope. No ticket tracks the collapse.
- fit: realpathtest.TempDir replaces the resolved-temp-dir idiom in the graphstore, watch, wtindex, agents and cli tests. realpathtest follows the locktest helper-package pattern, as task 2.3 names.
- fit: The instructionsKeptNote and instructionsRequestedElsewhere doc comments in shared.go still describe codexTarget.Uninstall and opencodeTarget.Uninstall as the callers. removeSharedInstructions is the caller now. This is comment drift, not a fit category.
- invariants: I1: Within(root, path) is true only for root itself or a descendant, and a name such as ..notes.md is inside \| enforced at internal/realpath/realpath.go:21 Within \| boundary test TestWithin_NameWithDotDotPrefixIsInside, TestWithin_ParentIsOutside, TestWithin_SiblingWithSharedPrefixIsOutside, TestWithin_RootItselfIsInside. A probe confirmed Within(/,/x)=true, Within(/a,/a/..b)=true and Within(/a,a/x)=false.
- invariants: I2: ResolveFile returns the location where the operating system makes a file for the path, or an error \| enforced at internal/realpath/realpath.go:60 resolve \| boundary test TestResolve_AgreesWithTheOperatingSystem (property) plus the example tests. Edge gap: an empty link target, inv-1.
- invariants: I3: A trailing separator or '.' that names a missing location is refused for a file \| enforced at internal/realpath/realpath.go:52 dirOnly \| boundary test TestResolve_LinkWithTrailingSlashToAMissingTarget, TestResolveFile_TrailingDotAfterAMissingElementIsAnError. A probe of link -> existing-file/ got ENOTDIR from ResolveFile and from the operating system alike.
- invariants: I4: The walk reads at most 8 links, and a missing directory does not count \| enforced at internal/realpath/realpath.go:131 links > maxLinks \| boundary test TestResolve_EightHopsThroughMissingTargetsResolve and TestResolve_NineHopsThroughMissingTargetsFail (both sides), TestResolve_MissingTargetUnderTenMissingDirectoriesResolves. A probe confirmed the asymmetry that design D2 accepts: 9 links to an existing file resolve, 9 links to a missing file are refused, and the operating system makes the file through both.
- invariants: I5: '..' after a missing element is an error, and '..' after a link names the parent of the link's real target \| enforced at internal/realpath/realpath.go:106 \| boundary test TestResolve_DotDotAfterAMissingElementIsAnError, TestResolve_RelativeDotDotTargetUnderALinkedDirectory, TestResolve_MissingTargetWithDotDotAfterALinkedDirectory.
- invariants: I6: A symlink loop is an error and never 'no file' \| enforced at filepath.EvalSymlinks (too many links) and the maxLinks bound, before any fileExists in the edit functions (codex.go:305, hermes.go:409) \| boundary test TestResolve_LoopIsAnError, TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing, TestUninstall_SymlinkLoopReportsAnError, TestUninstall_Hermes_SymlinkLoopReportsAnError, TestAntigravity_MigrationSymlinkLoopAtTheUnifiedPathReportsAnError, TestInstallAndUninstall_SymlinkLoopExitsNonZero.
- invariants: I7: At local scope, the real location of a user's file is always inside the real working directory, or the file is refused \| enforced at internal/agents/shared.go:73 resolveUserFile \| boundary test TestResolveUserFile_LocalNeverLeavesTheProject (property, asserts both outcomes drawn), TestResolveUserFile_LocalWithNoWorkingDirectoryIsRefused.
- invariants: I8: Every read, write and delete of a user's file in Install and Uninstall gets the resolved path \| enforced at recordUserFile/recordUserAction at every helper call site (rg of writeMcpEntry, removeMcpEntry, writeOpencodeEntry, removeOpencodeEntry, writeHookEntry, removeHookEntry, upsertInstructionsEntry, removeMarkedSection and the TOML/YAML edits found no call outside the funnel except the APPDATA sweep, which calls resolveUserFile first) \| boundary test TestInstall_LocalConfigSymlinkOutsideProjectIsRefusedByEachTarget plus the per-target scenario tests.
- invariants: I9: An install refuses a local file outside the project, and an uninstall records kept with a note and returns nil \| enforced at internal/agents/shared.go:99-105 \| boundary test TestRecordUserFile_InstallOutsideRecordsAnError, TestRecordUserFile_UninstallOutsideRecordsKeptAndANote, TestRecordUserAction_UninstallOutsideRecordsKeptAndANote, TestInstall_LocalSymlinkOutsideProjectExitsNonZero, TestUninstall_LocalSymlinkOutsideProjectExitsZeroWithANote.
- invariants: I10: The check runs before any read of the file, so the unchanged short-circuit cannot skip it \| enforced at recordUserFile, which resolves before it calls edit, and in antigravity.go:214-224, where readMcpEntry runs only when legacyErr == nil \| boundary test TestInstall_LocalConfigSymlinkOutsideWithTheEntryIsRefused.
- invariants: I11: A result names the configured path, and an error for a linked path names the real location once \| enforced at internal/agents/shared.go:112 and :121 \| boundary test TestRecordUserFile_ResultNamesTheConfiguredPath, TestRecordUserFile_HelperErrorNamesTheRealLocation, TestRecordUserFile_UnlinkedLocalErrorKeepsItsText.
- invariants: I12: The file a link leads to keeps its permission bits \| enforced at internal/fsatomic/fsatomic.go:58 os.Stat(path) on the real path \| boundary test TestInstall_GlobalConfigSymlinkKeepsTargetMode (0640).
- invariants: I13: An MCP JSON or JSONC file that does not parse is never written, and an empty or whitespace-only file is written as new \| enforced at shared.go:306 readJSONFileForWrite and opencode.go:136-146 \| boundary test TestMcpEntry_EmptyFileIsWritten and TestOpencode_EmptyAndWhitespaceConfigIsWritten (both with "" and whitespace), TestInstall_GlobalConfigSymlinkToUnparseableTargetIsUntouched.
- invariants: I14: An uninstall keeps a section that another configured agent uses through the same real file \| enforced at shared.go:963 sameRealPath in instructionsRequestedElsewhere, called by removeSharedInstructions for claude, gemini, codex and opencode \| boundary test TestUninstall_Gemini_LinkedInstructionFileKeepsTheSectionCodexUses, TestUninstall_Claude_LinkedInstructionFileKeepsTheSectionCodexUses.
- invariants: I15: The APPDATA sweep never edits the live OpenCode file \| enforced at opencode.go:275 realpath.Within(liveDir, stalePath), which replaces the textual APPDATA == cfgDir test \| boundary test opencode_test.go:255 (APPDATA equals the XDG directory) and TestOpencode_AppDataSweep_SkipsALinkToTheLiveConfig. Not weakened.
- invariants: I16: The MCP path confinement is unchanged \| enforced at internal/mcp/tools.go, which calls realpath.Within with the same body as the removed isWithin (diff checked line by line) \| boundary test the existing MCP confinement tests. Not weakened.
- invariants: Probes ran on a copy of internal/realpath/realpath.go in a separate module under the handoff directory (probe/). No repository test was run.
- test-coverage: Tasks 6.1 and 6.2 are unchecked (full test run, lint, validate, changelog fragments); not a scenario matter.
- test-coverage: The 'refresh after an upgrade reports a warning' clause has no scenario; not graded.
- test-honesty: Read symlink_install_test.go, userfile_test.go, the property test in realpath_test.go and the two CLI tests; no new skip, no assertion-free test, no tautology found.
- test-honesty: Minor, not a finding: the property test TestResolveUserFile_LocalNeverLeavesTheProject continues silently on non-outside errors and does not check that a refusal for a not-yet-existing path is right.
- test-honesty: Did not read the individual Resolve unit tests, skillshared_test.go renames, the graphstore, watch and wtindex test edits, or the other CLI tests.

## Plan review (2026-11-12)

Verdict approve-with-notes at head a12602e6505f0b0752771a84cbcbf3658b01c87d.

### Findings

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

### Coverage matrix

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
