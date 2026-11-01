# Tasks

## 1. Parser qualification

- [ ] 1.1 Add independently reviewed fixtures for all four dialects under #238's evaluation format. Document required forms and expected parse limitations beside the fixtures.
- [ ] 1.2 Pin the Bash, Zsh, and Fish parser sources and record their licenses in `NOTICE`. Preserve upstream generated files unchanged. Record the dependency review and exact revisions.
- [ ] 1.3 Add minimal parser adapters through the existing parser seam. Cover scanner lifecycle, malformed input, source limits, and repeated extraction in `TestShellParserIntegration`.
- [ ] 1.4 Exercise the adapters with the existing macOS and Linux build targets. Record linking results and binary-size changes. Keep any failing dialect unregistered and its tasks incomplete.

## 2. Discovery and file watching

- [ ] 2.1 Add bounded shell classification and `EXCLUSION_REASON_INTERPRETER_MISMATCH` to both authored protocol sources. Regenerate clients and update exclusion rendering. Cover precedence, non-shell extension compatibility, `env -S`, unknown interpreters, and extensionless files in `TestShellDiscovery`.
- [ ] 2.2 Add shared root-relative inclusion rules for repository-local Fish configuration. Cover allowed paths, excluded siblings, source limits, and symlink escapes in `TestShellDiscoveryBoundaries`.
- [ ] 2.3 Share root-anchored inclusion across discovery, fsnotify recursive registration and directory events, and macOS FSEvents filtering. Cover new Fish directories, nested-path exclusions, and changed shebangs in `TestShellWatchDiscoveryParity` on both platforms.
- [ ] 2.4 Document recognized filenames, shebang forms, precedence, and hidden-directory exceptions. Make sure that every documented form appears in `TestShellDiscovery` or `TestShellDiscoveryBoundaries`.

## 3. Dialect extraction

- [ ] 3.1 Add POSIX shell function, binding, read, and direct-call extraction. Cover command-scoped assignments, pipelines, subshells, literals, and heredocs in `TestPOSIXShellExtraction`. Document the supported subset.
- [ ] 3.2 Add Bash arrays and explicit local declarations through its dialect adapter. Cover shadowing, substitutions, indirect names, and dynamic scope in `TestBashExtraction`. Document unresolved forms.
- [ ] 3.3 Add Zsh named functions and static bindings through its dialect adapter. Cover arrays, parameter expansion, glob syntax, and parse failures in `TestZshExtraction`. Document exact parser limits.
- [ ] 3.4 Add Fish functions and explicit local/global `set` bindings. Cover repeated assignments, scope, substitutions, and universal-variable limits in `TestFishExtraction`. Document unresolved forms.
- [ ] 3.5 Reject shell extraction on any Tree-sitter error or missing node. Preserve file errors and remove stale facts during synchronization. Cover valid syntax beside rejected syntax, dependent references, and failed-file accounting in `TestShellParseFailures`.

## 4. Source relationships and reference resolution

- [ ] 4.1 Add a shell-specific source-path resolver to existing file nodes with imports edges and path-assumption metadata. Bypass Go module-prefix resolution. Cover relative candidates, absolute paths, escapes, expansion, and absence of synthetic packages in `TestShellSourceResolution`.
- [ ] 4.2 Add cycle-safe source traversal and conservative imported function resolution. Cover cycles, repeated imports, ambiguous names, and dynamic commands in `TestShellReferenceResolution`.
- [ ] 4.3 Add read/write metadata and stable binding identities through the shared graph contract. Cover repeated assignments and owner separation in `TestShellBindingIdentity`.
- [ ] 4.4 Document source-path assumptions and unresolved runtime behavior. Make sure that `TestShellNoExecution` indexes side-effecting source text without executing it.

## 5. Registration and public behavior

- [ ] 5.1 Register each qualified dialect and extend language filters. Cover distinct IDs and exact declarations in `TestShellLanguageRegistration`. Do not mark the change complete until all four dialects qualify.
- [ ] 5.2 Add CLI and MCP definitions, callers, callees, impact, and explore acceptance. Cover expected facts, false matches, and output compatibility in `TestShellPublicQueries`.
- [ ] 5.3 Use normal `Sync` discovery for unchanged shell files absent from older indexes. Cover the upgrade in `TestShellSyncDiscoversExistingFiles`. Document that shell registration alone needs no rebuild marker.
- [ ] 5.4 Add semantic synchronization cases for source edits, rename, deletion, restoration, and dialect changes. Compare fresh and updated graphs in `TestShellSyncEquivalence`.
- [ ] 5.5 Publish per-dialect capability evidence and limitations. Trace each support claim to the dialect extraction tests and #238's independent report.

## 6. Integration acceptance

- [ ] 6.1 Run structural invariants, shell semantic acceptance, and affected watcher tests on macOS and Linux. Record the commands, exact parser revisions, and remaining limitations.
- [ ] 6.2 Run the existing release build and dependency audit targets. Make sure that released artifacts need no shell interpreter and preserve existing signing, attestation, and SBOM workflows.
- [ ] 6.3 Add the release note with `task changie -- new --interactive=false -k Features -b "Index POSIX shell, Bash, Zsh, and Fish sources with explicit dialect and resolution limits."`. Make sure that the generated fragment matches the qualified dialects.
