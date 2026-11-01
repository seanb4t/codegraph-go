# Design

## Context

See [proposal.md](proposal.md) for motivation. The [language registry](../../../internal/indexer/languages.go) has parser and extractor seams but no shell entry.

[Discovery](../../../internal/indexer/discoverexclusion.go) selects languages by extension. It prunes dot-prefixed directories through rules shared with file watching. That excludes repository-local `.config/fish` sources.

The implementation must retain one released binary. CGo remains limited to permitted parser and platform integrations. Shell source must never execute during indexing.

## Goals / Non-Goals

Goals: Reuse parser, extractor, graph, and synchronization seams with explicit dialect boundaries. Make partial parsing and heuristic path resolution visible.

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

Use this precedence: recognized interpreter, explicit unsupported-interpreter exclusion, startup filename, then extension. `.sh` without a recognized interpreter selects `shell`. `.bash`, `.zsh`, and `.fish` select their named dialects. An unidentified extensionless file stays excluded.

Recognize `.profile` as POSIX shell. Recognize `.bashrc`, `.bash_profile`, `.bash_login`, and `.bash_logout` as Bash. Recognize `.zshenv`, `.zprofile`, `.zshrc`, `.zlogin`, and `.zlogout` as Zsh. Explicit shebangs take precedence.

Allow `.config/fish/config.fish`, `.config/fish/functions/**/*.fish`, and `.config/fish/conf.d/**/*.fish` within the project root. Traverse only the ancestors needed for these paths. Keep other dot-prefixed directories excluded. Do not expand this exception to the user home directory.

Replace the basename-only prune decision with a shared root-relative inclusion decision where necessary. Initial discovery, recursive watcher registration, and new-directory events must use it. Preserve exclusion accounting when a directory has permitted descendants.

Do not follow file or directory symlinks outside the repository. Apply the same canonical-path boundary to sourced files. Respect source-size limits before shebang reads and extraction.

Keep one adapter per dialect and one common graph result shape. Reuse identity finalization after the `index-invariants` implementation proposed in PR #237 lands. Collect bindings before graph emission to consolidate repeated assignments.

Use these extraction boundaries:

| Dialect | Bindings and syntax | Resolution limits |
| --- | --- | --- |
| POSIX shell | Named functions, global assignment words, parameter reads, literal function commands, literal dot commands | Command-scoped environment assignments do not create persistent globals. Unsupported extensions remain visible. |
| Bash | POSIX forms plus named arrays, `local`, `declare`, and `typeset` declarations | Dynamic scope, namerefs, indirect expansion, and computed command names do not receive definitive targets. |
| Zsh | Named functions and statically named `typeset` or assignment bindings | Autoload, runtime option changes, and expansion-dependent binding remain unresolved. Unsupported grammar forms produce partial results. |
| Fish | Functions, `set` declarations, explicit local/global scope, parameter reads, and literal `source` commands | Universal variables and environment-dependent bindings remain unresolved. Do not infer autoload from a filename alone. |

Record direct calls in pipelines, subshells, and command substitutions when a target is statically established. Respect process and scope boundaries for writes. Do not turn words inside quoted strings or literal heredocs into declarations. Interpolated expansions receive only their supported syntactic references.

Use existing function and variable kinds plus `contains`, `references`, `calls`, and the existing import relationship convention. Reuse #239's access metadata convention without requiring every core-language extractor to land first. Add graph vocabulary only if an existing relationship cannot express the required distinction.

External commands remain unresolved names. A literal command matches a repository function only when lexical and source evidence establish a unique candidate. Mark inferred imported targets with their source-path assumption. Ambiguous definitions do not resolve by traversal order.

Resolve literal relative source paths against the importing file's directory as a static candidate convention. Record `resolution=script_relative_candidate` in relationship metadata. This is not a claim about the runtime working directory. Only eligible repository files can become targets.

Paths with expansion, command substitution, runtime search through `PATH`, or unknown roots stay unresolved. Do not execute `cd` statements to infer working directories. Literal absolute paths are eligible only when their canonical target stays within the repository.

Use cycle detection for sourced-file traversal. Do not create duplicate nodes when several scripts source the same file. Editing a sourced file must invalidate dependent resolution through the existing synchronization mechanism.

Report Tree-sitter error and missing nodes through existing indexing diagnostics. A nonempty result cannot hide rejected syntax. Keep recovered declarations only under the current partial-result policy, with failure evidence visible in #238.

Add dialect-specific fixtures to #238's evaluation format. Independent hand-reviewed expectations define source facts. Include CLI and MCP query cases plus fresh-versus-incremental comparisons. Do not use an installed shell as a production oracle.

Land classification and shared inclusion changes first. Add dialect adapters behind registration readiness. Enable each dialect only after its contract passes. Use an additive rebuild marker so existing indexes discover newly supported files.

Rollback uses the prior binary and a fresh compatible index. Keep the release pipeline intact and use its existing build and dependency audit steps.

## Risks / Trade-offs

- Fish lacks an inspected upstream Go binding. Keep adapter qualification explicit and do not modify generated grammar structure.
- Zsh grammars have syntax limits. Require representative extraction cases and visible parse failures before support claims.
- Relative source paths depend on runtime working directories. Label static candidates and avoid unconditional runtime claims.
- Hidden-directory exceptions can diverge from file watching. Use one path decision and integration cases for newly created Fish directories.
- Shell scope is dynamic in several dialects. Prefer unresolved references when static ownership is ambiguous.
- New parser dependencies affect portability and binary size. Measure both through the existing release build targets before registration.
