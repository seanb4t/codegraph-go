# Design

## Context

See [proposal.md](proposal.md) for motivation. The [language registry](../../../internal/indexer/languages.go) has parser and extractor seams but no shell entry.

[Discovery](../../../internal/indexer/discoverexclusion.go) selects languages by extension. It prunes dot-prefixed directories through rules shared with file watching. That excludes repository-local `.config/fish` sources.

The implementation must retain one released binary. CGo remains limited to permitted parser and platform integrations. Shell source must never execute during indexing.

## Goals / Non-Goals

Goals: Reuse parser, extractor, graph, and synchronization seams with explicit dialect boundaries. Make parse failures and heuristic path resolution visible.

Non-goals: Emulate shell execution, discover the user's live shell environment, or index every hidden directory. Do not add embedded shell extraction in this change.

## Decisions

Use maintained Tree-sitter grammars through the existing parser approach. Keep dialect adapters separate even when they share traversal helpers. Do not use regular expressions as the declaration parser.

The initial candidates are pinned source snapshots, not proven support claims:

| Dialect | Candidate | Qualification boundary |
| --- | --- | --- |
| POSIX shell and Bash | [tree-sitter-bash](https://github.com/tree-sitter/tree-sitter-bash/tree/a06c2e4415e9bc0346c6b86d401879ffb44058f7) | Separate the POSIX feature set from Bash extensions. A shared grammar does not establish POSIX conformance. |
| Zsh | [georgeharker/tree-sitter-zsh](https://github.com/georgeharker/tree-sitter-zsh/tree/7a593401efb5418ffdedbe3c0e4c61c6d240166d) | Exercise Zsh declarations, parameter expansion, arrays, and glob qualifiers. Record unsupported forms explicitly. |
| Fish | [ram02z/tree-sitter-fish](https://github.com/ram02z/tree-sitter-fish/tree/b7f1d682941e0c62dfcd6bf9ef481638351bab16) | The inspected repository has Node and Rust bindings, but no Go binding. Qualify a minimal adapter and scanner integration. |

The native Go [mvdan/sh parser](https://github.com/mvdan/sh/blob/master/syntax/parser.go) is an alternative for POSIX shell and Bash. Its inspected Zsh variant explicitly describes experimental, incomplete support. It does not remove the Fish integration requirement. Prefer the existing Tree-sitter seam to avoid two production AST models in this change.

Use upstream Go bindings where available. For Fish, prefer a maintained upstream binding contribution. If unavailable, use a small authored adapter outside generated parser sources. Pin and preserve upstream generated files unchanged. Record licenses and source attribution in `NOTICE`.

Before registration, exercise CodeGraph's extraction contract on macOS and Linux. Include static linking, scanner lifecycle, source limits, malformed input, and repeated parsing. Do not add a duplicate test suite for upstream grammar behavior.

If a candidate cannot meet a required case, keep that dialect unregistered and the corresponding tasks incomplete. Report the precise upstream gap. Do not downgrade required cases to claim all four dialects complete. Parser qualification remains an explicit implementation risk, not completed research evidence.

Register language IDs `shell`, `bash`, `zsh`, and `fish`. The `shell` ID means the documented POSIX subset. Use dialect-specific module keys based on repository-relative paths.

Add bounded source classification before parser creation. Read only the shebang prefix after the existing size and file-type safeguards. Accept direct interpreter paths and supported `/usr/bin/env` forms, including literal `-S` interpreter selection. Do not expand environment variables or arbitrary command expressions.

Recognize exactly the interpreter basenames `sh`, `bash`, `zsh`, and `fish`, mapped to the corresponding dialect IDs. Other explicit interpreters, including `dash`, `ash`, and `ksh`, are mismatches only for shell extensions or startup names. This bounded list does not claim that their languages lack POSIX-compatible syntax.

Apply shell classification only to shell extensions, recognized startup names, and otherwise eligible extensionless files. Registered non-shell extensions retain their existing classification regardless of shebang. Within shell candidates, use this precedence: recognized interpreter, explicit unsupported-interpreter exclusion, startup filename, then extension.

An eligible extensionless input is a regular repository file within the size limit that passes existing directory rules. Select a shell dialect only when its interpreter is recognized. Otherwise, preserve `EXCLUSION_REASON_UNSUPPORTED_EXTENSION`, including for extensionless Python, Node, or Perl scripts.

`.sh` without an explicit interpreter selects `shell`. `.bash`, `.zsh`, and `.fish` select their named dialects. An explicit unsupported interpreter on a shell extension or startup name uses `EXCLUSION_REASON_INTERPRETER_MISMATCH`, with the interpreter in diagnostic details.

Add that enum value to the authored graph and UI protocol sources. Regenerate their clients through the owning generators. Update exclusion renderers and tests for the additive reason without changing response envelopes.

Recognize `.profile` as POSIX shell. Recognize `.bashrc`, `.bash_profile`, `.bash_login`, and `.bash_logout` as Bash. Recognize `.zshenv`, `.zprofile`, `.zshrc`, `.zlogin`, and `.zlogout` as Zsh. Explicit shebangs take precedence.

Allow only `<repo>/.config/fish/config.fish`, `<repo>/.config/fish/functions/**/*.fish`, and `<repo>/.config/fish/conf.d/**/*.fish`. Anchor this exception at the repository root, not any nested `.config/fish` suffix. Traverse only the required ancestors. Keep other dot-prefixed directories excluded, including `home/.config/fish` and hidden parents. Do not expand this exception to the user home directory.

Replace the basename-only prune decision with a shared root-relative inclusion decision where necessary. Apply it to discovery, fsnotify recursive registration, new-directory events, and the macOS FSEvents path filter in `watcher_darwin.go`. Apply it also to the no-daemon staleness walk in `internal/query/status.go`. Place shared path policy below indexer and query packages to avoid an import cycle. Preserve exclusion accounting and keep route discovery within its existing scope.

Do not follow file or directory symlinks outside the repository. Apply the same canonical-path boundary to sourced files. Respect source-size limits before shebang reads and extraction.

Keep one adapter per dialect and one common graph result shape. Reuse identity finalization after the `index-invariants` implementation proposed in PR #237 lands. Collect bindings before graph emission to consolidate repeated assignments.

Use these extraction boundaries:

| Dialect | Bindings and syntax | Resolution limits |
| --- | --- | --- |
| POSIX shell | Named functions, global assignment words, parameter reads, literal function commands, literal dot commands | Command-scoped environment assignments do not create persistent globals. Recognized constructs outside this subset reject the file with a named indexing error. |
| Bash | POSIX forms plus literal `source`, named arrays, `local`, `declare`, and `typeset` declarations | Dynamic scope, namerefs, indirect expansion, and computed command names do not receive definitive targets. |
| Zsh | Named functions, static `typeset` or assignment bindings, parameter reads, literal function commands, and literal `.` or `source` commands | Autoload, runtime option changes, and expansion-dependent binding remain unresolved. Rejected grammar forms fail the whole file. |
| Fish | Functions, `set` declarations, explicit local/global scope, parameter reads, literal function commands, and literal `source` commands | Universal variables and environment-dependent bindings remain unresolved. Do not infer autoload from a filename alone. |

Record direct calls in pipelines, subshells, and command substitutions when a target is statically established. Respect process and scope boundaries for writes. Do not turn words inside quoted strings or literal heredocs into declarations. Interpolated expansions receive only their supported syntactic references.

Use existing function and variable kinds plus `contains`, `references`, `calls`, and `imports`. Reuse the imports edge kind, not the Go-only import resolver. Add a shell path resolver that targets existing indexed file nodes. Do not route shell paths through `isIntraModule` or create synthetic Go packages. Reuse #239's access metadata convention independently of core-language extractor rollout.

External commands remain unresolved names. A literal command matches a repository function only when lexical and source evidence establish a unique candidate. Mark inferred imported targets with their source-path assumption. Ambiguous definitions do not resolve by traversal order.

Resolve literal relative source paths against the importing file's directory as a static candidate convention. Record `resolution=script_relative_candidate` in relationship metadata. This is not a claim about the runtime working directory. Only eligible repository files can become targets.

Paths with expansion, command substitution, runtime search through `PATH`, or unknown roots stay unresolved. Do not execute `cd` statements to infer working directories. Literal absolute paths are eligible only when their canonical target stays within the repository.

Use cycle detection for sourced-file traversal. Do not create duplicate nodes when several scripts source the same file. Persist resolved and unresolved literal source candidates as normalized repository-relative lookup paths on importing file records through the schema source and generator.

Add path-based invalidation to synchronization rather than relying on symbol-name waiting lists. Adding, restoring, deleting, renaming, or changing a candidate's eligibility must re-resolve its unchanged importers and their transitive dependents. Reapply canonical-path boundaries on each resolution. Commit path dependencies and graph updates together, with cycle detection. Source edits must also invalidate imported function resolution.

Use an explicit all-or-nothing shell parse policy. Any Tree-sitter error or missing node rejects extraction for that file. Keep the file record with indexing errors, but emit no declarations or relationships from it. Count it as failed, retain independently expected facts in evaluation denominators, and remove stale facts during synchronization. Other languages retain their existing behavior.

The POSIX adapter also rejects recognized syntax outside its declared subset, even when the shared Bash grammar accepts it. Qualification must include arrays, Bash `function` declarations, `[[` conditions, process substitution, and literal `local`, `declare`, or `typeset` declaration commands. The indexing error names the rejected construct and uses the same whole-file rejection behavior. This is a supported-subset boundary, not a full POSIX conformance claim.

A sourced file keeps its independently selected dialect. A Bash importer does not promote a shebang-less `.sh` file to Bash. Such a library needs a recognized Bash shebang or `.bash` extension for Bash extraction.

Add dialect-specific fixtures to #238's evaluation format. Independent hand-reviewed expectations define source facts. Add `TestSemanticShell` to `task eval:semantic` and `task eval:semantic:dev` with the existing partition selection. Include CLI and MCP query cases plus fresh-versus-incremental comparisons. Do not use an installed shell as a production oracle.

Land classification and shared inclusion changes first. Add dialect adapters behind registration readiness. Enable each dialect only after its contract passes. Existing `Sync` calls `DiscoverAll` and adds files absent from storage. Use that path to discover unchanged, newly supported shell files without a new rebuild marker. Test migration from an index created before shell registration.

Rollback uses the prior binary and a fresh compatible index. Keep the release pipeline intact and use its existing build and dependency audit steps.

## Risks / Trade-offs

- Fish lacks an inspected upstream Go binding. Keep adapter qualification explicit and do not modify generated grammar structure.
- Zsh grammars have syntax limits. Require representative extraction cases and visible parse failures before support claims.
- Relative source paths depend on runtime working directories. Label static candidates and avoid unconditional runtime claims.
- Hidden-directory exceptions can diverge from file watching. Use one path decision and integration cases for newly created Fish directories.
- Shell scope is dynamic in several dialects. Prefer unresolved references when static ownership is ambiguous.
- New parser dependencies affect portability and binary size. Measure both through the existing release build targets before registration.
