# Graph Report - codegraph-go  (2026-08-03)

## Corpus Check
- 973 files · ~1,490,396 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 10334 nodes · 16355 edges · 862 communities (782 shown, 80 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 2525 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4c92d3b5`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- fakeHome
- Parser
- readLock
- main.go
- execCmd
- Extract
- Phase 8: Release Hardening & Benchmarks - Research
- Node
- Communities (60 total, 13 thin omitted)
- TargetID
- SynthesizeImplements
- File
- newRootCmd
- New
- Extract
- Extract
- BuildServer
- BuildTSIndex
- extractor
- shared_test.go
- golden_parity_test.go
- OpenAt
- Pattern Assignments
- .Install
- translate_test.go
- Feature Research — v1.0 Drop-in CLI Parity + Human TUI
- pebble_store.go
- NodeID
- WriteResult
- stringValue
- Architecture Research: v1.0 Integration into codegraph-go
- targetRoot
- extractor
- Technology Stack
- goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed
- GraphStore
- Sync
- Run
- Implementation Decisions
- extractor
- Discover
- parseFixture
- traverse.go
- Extract
- Extract
- traverse_test.go
- Implementation Decisions
- Pattern Assignments
- Phase Details
- pebbleWriter
- keys.go
- extractor
- Extract
- render_markdown.go
- Implementation Decisions
- Implementation Decisions
- Implementation Decisions
- v1 Requirements
- fileExists
- Writer
- Open
- extractor
- migrate_test.go
- Implementation Decisions
- Implementation Decisions
- Implementation Decisions
- Pattern Assignments
- Phase 5 Plan 03: Resolution Breadth — Module-Key Seam + WR-01/WR-02 Fixes Summary
- Pattern Assignments
- .Install
- hermes.go
- Generate
- Extract
- Extract
- runAndSnapshot
- search.go
- Phase 5 Plan 01: Multi-Language Seam Foundation Summary
- Pattern Assignments
- Pitfalls Research
- CGoParser
- Per-Language Gaps
- Pattern Assignments
- Phase 5 Plan 02: Multi-Language Discovery + Extraction Worker-Pool Fix Summary
- Warnings
- matrix_test.go
- resolveRefsWithIndex
- lookupLanguageByID
- extractor
- .Install
- hasEdge
- symbolIndex
- Critical Issues
- Phase 4 Plan 4: Rename/Delete/Move Prune Fixture Suite + Sync-Equals-Reindex Determinism Gate Summary
- Warnings
- Phase 5 Plan 13: Language Capability Matrix + Cross-Language Regression Closeout (LANG-06) Summary
- Phase 7: Migration Tool - Research
- Fixed Issues
- Phase 08 Plan 06: Committed-Baseline Regression Gate Summary
- UserController
- Phase 1 Plan 2: Versioned Protobuf Schema Summary
- Phase 01 Plan 04: TS CodeGraph v1.3.1 Golden Fixtures Summary
- Phase 01 Plan 06: GraphStore Interface + Pebble/v2 Implementation Summary
- Phase 3 Plan 8: Query & MCP Command Surface Summary
- Phase 3 Plan 09: Golden Parity Test Summary
- Phase 4 Plan 7: internal/daemon — Shared Watch/Index Server + Lockfile Summary
- Phase 5 Plan 05: C# Extraction + Cross-File Resolution (LANG-03) Summary
- Phase 5 Plan 11: Mainstream Grammar Layer — C/C++, Swift, Kotlin (LANG-06) Summary
- Phase 5 Plan 12: Framework-Aware Routing (LANG-07) Summary
- Phase 08 Plan 01: GoReleaser CGo Build Matrix Summary
- Phase 08 Plan 02: RSS Normalization + Metrics Core Summary
- Phase 08 Plan 03: Synthetic Corpus Generator Summary
- Phase 08 Plan 04: Release Workflow (Native Matrix + Cosign + SLSA + SBOM) Summary
- Phase 08 Plan 07: Benchmark Runner — Head-to-Head + Regression Gate Summary
- Phase 08 Plan 08: CI Gate Workflows Summary
- 1. Methodology
- newFilesCmd
- Phase 01 Plan 03: Parser Interface + CGo Tree-sitter Backend Summary
- Phase 01 Plan 05: Typed Keyspace & Key-Injection Guard Summary
- Phase 2 Plan 03: Go AST Extraction Mapper and Pass 1 Worker Pool Summary
- Phase 2 Plan 04: Pass 2 Resolve — Global Symbol Index and Deterministic Cross-File Resolution Summary
- Phase 2 Plan 05: Pipeline Orchestration and Determinism Gate Summary
- Phase 2 Plan 06: CLI Lifecycle (init/index/uninit) Summary
- Phase 3 Plan 4: Call-Graph Traversal (Callers/Callees/Impact/Affected) Summary
- Phase 3 Plan 06: Node/Explore Markdown Rendering Summary
- Phase 4 Plan 1: File-Owned Secondary Index Summary
- Phase 4 Plan 3: Incremental Sync Engine (internal/indexer.Sync) Summary
- Phase 4 Plan 5: Native fsnotify Watcher + Env-Tunable Debounce Summary
- Phase 4 Plan 8: codegraph sync/daemon/unlock Commands + serve --watch Fallback Summary
- Phase 5 Plan 04: Java Extraction + Cross-File Resolution (LANG-02) Summary
- Phase 5 Plan 06: Python Extraction + Cross-File Resolution (LANG-04) Summary
- Phase 5 Plan 07: TypeScript/JavaScript Extraction + Cross-File Resolution (LANG-05) Summary
- Phase 5 Plan 08: Mainstream Grammar Layer (LANG-06) Summary
- Phase 5 Plan 09: Interface→Implementation Dispatch Synthesis (RES-02/RES-03) Summary
- Phase 5 Plan 10: Mainstream-Tier Extraction — Rust, Ruby, PHP (LANG-06) Summary
- Phase 5 Gap Closure: Java + C# Real-Repo D-12 Validation (LANG-02, LANG-03) Summary
- Phase 6 Plan 04: install/uninstall CLI Commands Summary
- Phase 6 Plan 6: Signature-Verified Self-Update (upgrade) Summary
- Phase 7 Plan 3: Migration Reader + Translation Summary
- Phase 7 Plan 6: Migration Orchestration (migrate.Run) Summary
- Phase 08 Plan 09: Release & Benchmark Documentation Summary
- Pattern Assignments
- Edge
- detectRoutes
- swap_test.go
- Phase 01 Plan 01: Go Module & Package Skeleton Bootstrap Summary
- Phase 2 Plan 01: Node Identity and Schema Field Parity Summary
- Phase 2 Plan 02: File Discovery Seam and Shared Test Fixture Summary
- Phase 3 Plan 1: Reader Node/File Enumeration Summary
- Phase 3 Plan 2: Query Engine Foundation Summary
- Phase 3 Plan 3: Deterministic Lexical Query/Search Summary
- Warnings
- Phase 4 Plan 2: Additive Sync Schema Fields + Exported Reverse-Adjacency Summary
- Phase 4 Plan 9: Goroutine-Leak-Free Watcher/Daemon Soak (goleak) Summary
- Phase 5: Language Coverage & Resolution Breadth - Research
- Phase 6 Plan 02: JSON-Config Agent Targets (Claude, Cursor, Gemini, Kiro, Antigravity) Summary
- Phase 6 Plan 03: Non-JSON Agent Targets (Codex TOML, opencode JSONC, Hermes YAML) Summary
- Phase 6 Plan 5: CLI Lifecycle — version, --version, telemetry Summary
- Phase 7 Plan 1: Migration Tool Wave-0 Infrastructure Summary
- Phase 7 Plan 2: Migration Progress Record Summary
- Phase 7 Plan 4: Progress Cursor + Atomic Directory Swap Summary
- Phase 7 Plan 5: Migration Validation (D-09 Invariant Pass) Summary
- Phase 7 Plan 7: CLI Wiring (codegraph migrate) Summary
- recordFile
- registerLanguage
- verifyRelease
- verify_test.go
- Parser Strategy Decision (D-05 / D-05a)
- Phase 3 Plan 07: Stdio MCP Server + Tool Gating Summary
- Phase 4: Incremental Sync & File Watcher - Research
- Phase 6 Plan 01: internal/agents Foundation Summary
- Phase 6: Agent Integrations & CLI Lifecycle - Research
- Threat Verification
- Goal Achievement
- Phase 7: Code Review Report (Re-Review)
- Phase 08 Plan 05: Upgrade E2E Verification Loop-Closer Summary
- CodeGraph Go
- v1.0 Research Summary: Drop-in Parity & Human UX
- .Install
- Run
- TestPruneFixtures
- filesStatusFixture
- resolveLatestVersion
- Phase 1: Foundation — Storage, Schema & Parser Strategy - Research
- Phase 01: Code Review Report
- Phase 2: Go Indexing Pipeline - Research
- Info
- Goal Achievement
- Phase 3 Plan 05: Files & Status Query Commands Summary
- Phase 3: Query Engine & MCP Server - Research
- Fixed Issues
- Goal Achievement
- Goal Achievement
- Project State
- resolution_test.go
- init
- resolution_test.go
- resolution_test.go
- loadProgress
- NewMeta
- Graph Report - codegraph-go  (2026-07-10)
- Goal Achievement
- Phase 3: Query Engine & MCP Server - Discussion Log
- Goal Achievement
- Phase 4: Incremental Sync & File Watcher - Discussion Log
- Architecture Patterns
- Goal Achievement
- Phase 6: Agent Integrations & CLI Lifecycle - Discussion Log
- Info
- Verifying a codegraph release
- Info
- resolution_test.go
- resolution_test.go
- Registered
- walkSpringRoutes
- countingWriter
- Source
- NewGoParser
- Pointer
- atomicSwap
- Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero)
- Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log
- Phase 2: Go Indexing Pipeline - Discussion Log
- Phase 3 — Validation Strategy
- Phase 5: Language Coverage & Resolution Breadth - Discussion Log
- Phase 8: Release Hardening & Benchmarks - Discussion Log
- Fixed Issues
- Milestone: v0.1 — Initial Release
- Router
- Router
- discover.go
- languages_typescript.go
- gin_fixture.go
- pkga.go
- nodeExploreFixture
- upgrade.go
- Architecture Patterns
- Info
- Architecture Patterns
- Phase 7: Migration Tool - Discussion Log
- Common Pitfalls
- Phase 8: Release Hardening & Benchmarks Verification Report
- languages_csharp.go
- UserController
- UserController
- TestBuildTSIndex_Variants
- Phase 1 — Validation Strategy
- Phase 2 — Validation Strategy
- Architecture Patterns
- Phase 4 — Validation Strategy
- Architecture Patterns
- Phase 5 — Validation Strategy
- Common Pitfalls
- Phase 6 — Validation Strategy
- Phase 7 — Validation Strategy
- Phase 8 — Validation Strategy
- fileindex_test.go
- pebbleStore
- languages_php.go
- init
- nodeid_test.go
- UserController
- UserController
- Run
- Phase 01: Code Review Fix Report
- Code Examples
- Common Pitfalls
- Architecture Patterns
- SEED-001: local Svelte + shadcn-svelte UI for browsing/viewing/querying the graph
- Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06)
- init
- init
- init
- Common Pitfalls
- Code Examples
- Common Pitfalls
- Common Pitfalls
- 06-UAT.md
- Architecture Patterns
- Field Mapping (D-02, D-05)
- capture.sh
- package.json
- init
- import_graph_test.go
- package.json
- modernc_confinement_test.go
- Code Examples
- Validation Architecture
- Validation Architecture
- Common Pitfalls
- Validation Architecture
- Tech tracking
- Validation Architecture
- 05-05-PLAN.md
- Validation Architecture
- Deferred Items — Phase 5
- Validation Architecture
- Standard Stack
- Validation Architecture
- stubResolver
- processStartTime
- express_fixture.ts
- Foo
- app.ts
- SetConfig
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 03-01-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 04-01-PLAN.md
- 04-02-PLAN.md
- 04-03-PLAN.md
- 04-04-PLAN.md
- 04-05-PLAN.md
- 04-06-PLAN.md
- 04-07-PLAN.md
- 04-08-PLAN.md
- 04-09-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 05-01-PLAN.md
- 05-02-PLAN.md
- 05-03-PLAN.md
- 05-04-PLAN.md
- 05-06-PLAN.md
- 05-07-PLAN.md
- 05-08-PLAN.md
- 05-09-PLAN.md
- 05-10-PLAN.md
- 05-11-PLAN.md
- 05-12-PLAN.md
- 05-13-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Code Examples
- Sources
- 06-01-PLAN.md
- 06-02-PLAN.md
- 06-03-PLAN.md
- 06-04-PLAN.md
- 06-05-PLAN.md
- 06-06-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Code Examples
- Sources
- User Constraints (from CONTEXT.md)
- Sources
- 08-01-PLAN.md
- 08-02-PLAN.md
- 08-03-PLAN.md
- 08-04-PLAN.md
- 08-05-PLAN.md
- 08-06-PLAN.md
- 08-07-PLAN.md
- 08-08-PLAN.md
- 08-09-PLAN.md
- searchFakeNodeIterator
- stubDoer
- resolveCSharpCorpus
- resolveJavaCorpus
- resolvePythonCorpus
- resolveTSJSCorpus
- `baseline.json` provenance and re-bless instructions
- processStartTime
- Greeter
- CallDescribe
- Demo.csproj
- TestMain
- Milestones
- 01-01-PLAN.md
- 01-02-PLAN.md
- 01-03-PLAN.md
- 01-04-PLAN.md
- 01-05-PLAN.md
- 01-06-PLAN.md
- 01-07-PLAN.md
- Security Domain
- 02-01-PLAN.md
- 02-02-PLAN.md
- 02-03-PLAN.md
- 02-04-PLAN.md
- 02-05-PLAN.md
- 02-06-PLAN.md
- Security Domain
- 03-02-PLAN.md
- 03-03-PLAN.md
- 03-04-PLAN.md
- 03-05-PLAN.md
- 03-06-PLAN.md
- 03-07-PLAN.md
- 03-08-PLAN.md
- 03-09-PLAN.md
- Security Domain
- Security Domain
- Security Domain
- Security Domain
- 07-01-PLAN.md
- 07-02-PLAN.md
- 07-03-PLAN.md
- 07-04-PLAN.md
- 07-05-PLAN.md
- 07-06-PLAN.md
- 07-07-PLAN.md
- Security Domain
- 2026-07-14-document-release-cut-procedures-runbook.md
- Widget
- types.go
- API-SURFACE.md
- node.go
- com.example:springfixture
- djangofixture
- example.com/ginfixture
- example.com/gofixture
- example.com/gonoginfixture
- example.com/mixed
- example.com/prunefixture
- fastapifixture
- flaskfixture
- github.com/seanb4t/codegraph-go
- promptAgentMultiSelect
- stubWriter
- Discover
- status.go
- TestAspNet_DetectsHttpVerbAttributes
- Extract
- Phase 3 Plan 3: Watcher-on-MCP Default Flip Summary
- copyFixture
- companionHandler
- explore_gate_test.go
- release_workflow_shape_test.go
- Implementation Decisions
- Implementation Decisions
- Implementation Decisions
- Info
- OpenAt
- Pattern Assignments
- Pattern Assignments
- Goal Achievement
- Implementation Decisions
- execCmd
- initRepo
- Implementation Decisions
- Fixed Issues
- Implementation Decisions
- Pattern Assignments
- Pattern Assignments
- CachingDetector
- Phase 4: Output Hygiene - Context
- Phase 10 Plan 07: Close the Phase — CONTRIBUTING Pointer, Single-Definition Guard, Clean-Checkout Proof Summary
- rwr_test.go
- search_test.go
- Pattern Assignments
- Phase 9 Plan 1: release-please spine + non-vacuous drift guard Summary
- Phase 9 Plan 6: main fast-forwarded; release-please's App-token path proven end to end via release PR #2 Summary
- Pattern Assignments
- Extract
- Pattern Assignments
- Goal Achievement
- Phase 9 Plan 3: PR-title conventional-commit gate + actionlint static check Summary
- Pattern Assignments
- targetRoot
- BuildServer
- Phase 2: Code Review Report
- Pattern Assignments
- Phase 9 Plan 2: Idempotent release.yml publish step (D-04) Summary
- Phase 10 Plan 3: release.yml onto Namespace runners — darwin leg moved by maintainer decision, D-07/D-08 machine-enforced Summary
- daemon_test.go
- daemonpicker_test.go
- Phase 1 Plan 8: Java + C# D-09 Edge-Kind Extraction Summary
- Phase 1 Plan 9: Python + TS/JS D-09 Edge-Kind Extraction Summary
- Phase 1 Plan 12: Named-Symbol Seeding + Per-Overload Disambiguation Tiers (H13) Summary
- Phase 1 Plan 17: Behavioral Parity Harness — F5 Regeneration + D-02 Oracle Summary
- Pattern Assignments
- Phase 08 Plan 07: Charm Supply-Chain CGo Closure Audit Summary
- Phase 10 Plan 1: Tracer — go.tool.mod, install-task, Taskfile, and Namespace runners proven on one real CI job Summary
- Phase 10: Local Build Tooling & CONTRIBUTING - Discussion Log
- Record
- captureDiagWriter
- Phase 1 Plan 2: D-09 Edge-Kind Vocabulary Foundation Summary
- Phase 1 Plan 5: Go Reference-Language D-09 Edge-Kind Extraction Summary
- Phase 2 Plan 2: Status content — dbSizeBytes + FilesByLanguage Summary
- Phase 3 Plan 1: Watch-Policy Port Summary
- Phase 3 Plan 2: Watcher Policy Enforcement + Defer-and-Retry Convergence Summary
- Phase 5 Plan 3: internal/githooks — Install/Remove/Status Summary
- Phase 5 Plan 4: codegraph githooks CLI command tree Summary
- Phase 7 Plan 07: Daemon Picker & Command Tree Summary
- Phase 08 Plan 05: affected scripting flags (--stdin/-d/-f/-q/-j) Summary
- Phase 8 Plan 09: Release Runbook, Drop-In Gate & Caveat Retirement Summary
- Phase 9 Plan 5: GitHub App creation + secrets checkpoint Summary
- Phase 10 Plan 2: ci.yml's test job onto task targets — build/test/vet/vuln Taskfile surface, serial contributor wrappers, DEV-01's go vet gate Summary
- newFilesCmd
- Phase 1 Plan 1: Behavioral Fixture Harness + TS 1.3.1 Golden Capture Summary
- Phase 1 Plan 11: Subgraph-Construction Heuristics H10-H12 (EXPL-02) Summary
- Phase 1 Plan 13: Per-File Scoring, Hard Exclusion & Buried-Rescue (H14-H16) Summary
- Phase 1 Plan 14: File-Relevance Gate, Central-File Selection & 5-Tier Sort (H17-H19) Summary
- Phase 1 Plan 15: D-09 Force Re-Index (F4) Summary
- Phase 2: Code Review Report (iteration 2 — re-review of the fix commits)
- Phase 3 Plan 4: TEST-04 Subprocess Integration Harness Summary
- Phase 3 Plan 5: D-21 WATCH Subprocess Coverage Summary
- Phase 5 Plan 1: Extract internal/fsatomic Summary
- Phase 5 Plan 2: gitmeta IsGitRepo & HooksDir Summary
- Phase 06 Plan 01: Rendering Seam Foundation — TUI-01 Archtest + Charm Skeleton Summary
- Phase 06 Plan 02: Rendering Seam — Pretty status/files Rendering Summary
- Phase 06 Plan 03: Rendering Seam — TTY-Gated Progress Feedback (TUI-05) Summary
- Phase 07 Plan 03: PPID Reparent Watchdog Summary
- Phase 07 Plan 05: Wire Registry + PPID Watchdog into daemon.Run Summary
- Phase 8 Plan 4: Affected depth-bounded BFS with test-leaf pruning Summary
- Warnings
- Phase 8 — Validation Strategy
- Phase 9 Plan 4: RELEASE-PROCEDURES.md rewrite + recorded divergences Summary
- Phase quick-260728-rfx: Rewrite REL-02 as a Testable Property Summary
- Quick Task 260731-0a0: Fix govulncheck GO-2026-6061 Summary
- .Run
- DetectIndexMismatch
- NewProgress
- .Explore
- Phase 1 Plan 04: Node Multi-Def Enumeration + Budget/Overflow + Never-Empty Narrowing Summary
- Phase 1 Plan 6: RWR Graph-Relevance Core (EXPL-02) Summary
- Phase 1 Plan 16: Explore Pipeline Wiring (H1-H21) Summary
- Goal Achievement
- Phase 2 Plan 5: Status Renderers (CLI padded columns + MCP bolded-key bullets) Summary
- Phase 2 Plan 6: MCP Markdown Wiring + Server-Scoped Worktree Notice Summary
- Phase 2 Plan 7: CLI Wiring — Sectioned Status + Compact Worktree Notice Summary
- Info
- Info
- Phase 07 Plan 04: Daemon Stop Signaling Summary
- Phase 7 Plan 06: Bubbles Checkbox Multi-Select for Install/Uninstall Summary
- Phase 7 Plan 08: TEST-03 Byte-Invariance & Piped Never-Hang Summary
- Phase 8 Plan 1: Engine Impact-Depth Default & Short Flags Summary
- Phase 8 Plan 6: FLAG-PARITY.md matrix + drift test Summary
- Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Discussion Log
- Phase 9 — Validation Strategy
- Phase 10 Plan 04: Runner Identity, Storage-Frame Investigation & Baseline Staleness Fix Summary
- Phase 10 Plan 05: One-Definition Pre-Tag Cross-Target Sweep Summary
- Phase 10 Plan 06: Runner/ScratchFS Category-Error Gate + Reproducibility Task Targets Summary
- v1.0 Requirements
- .Install
- githooks.go
- setupIndexedFixtureWithTest
- policy.go
- Debug: perf regression gate reports ~10.6% throughput regression
- Phase 1 Plan 03: Query Tokenizers (H1 + H2) Summary
- Phase 1 Plan 07: Hybrid Gather Channels (H3-H6) Summary
- Phase 01: Code Review Report — Behavioral Parity (explore & node)
- Phase 2 Plan 4: Engine.startPath + Live WorktreeMismatch Summary
- Phase 2: status Content & Git/Worktree Awareness - Discussion Log
- Phase 2: status Content & Git/Worktree Awareness - Research
- Goal Achievement
- Phase 4 Plan 1: Pebble Logger Routing (HYG-01) Summary
- Phase 4 Plan 2: Stdout Confinement Archtest (HYG-02 Structural Guard) Summary
- Phase 5: Git Sync Hooks - Research
- Code Examples
- Phase 05: Code Review Report
- Goal Achievement
- Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Research
- Goal Achievement
- Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Research
- Phase 8: Code Review Report (iteration 3 — re-review)
- Fixed Issues
- Phase 9 — Security
- Phase 10: Local Build Tooling & CONTRIBUTING - Research
- Warnings
- statusWorktreeMismatchFixture
- Contributor Covenant Code of Conduct
- Cutting a release (maintainer runbook)
- Phase 1 Plan 10: Post-Merge Gather Rerankers (H7-H9) Summary
- Code Examples
- Phase 2 Plan 1: internal/gitmeta — Git Worktree Detection Summary
- Phase 2 Plan 3: SURF-06 Markdown Renderers Summary
- Phase 4 Plan 3: MCP Stdout Purity & Sync Noise-Absence (End-to-End) Summary
- Fixed Issues
- Fixed Issues
- Phase 4 — Validation Strategy
- Goal Achievement
- Phase 5 Plan 5: Wire HOOK-03's watcher-fallback advisory into init/uninit Summary
- Phase 5: Git Sync Hooks - Discussion Log
- Phase 7 Plan 2: Charm-Free Global Daemon Registry Summary
- Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Discussion Log
- Phase 08 Plan 02: files --dir prefix filter + -j alias Summary
- Phase 08 Plan 03: Mechanical short-flag parity + upgrade --force Summary
- Phase 9: release-please + GoReleaser - Discussion Log
- Phase 9: release-please + GoReleaser - Research
- Goal Achievement
- codegraph
- Parser
- initGitRepo
- daemonDelegate
- Phase 1: Behavioral Parity — explore & node - Research
- Fixed Issues
- Fixed Issues
- Phase 2 — Security
- Phase 04: Code Review Report
- Tests
- Fixed Issues
- Phase 5 — Validation Strategy
- Phase 6 — Validation Strategy
- Goal Achievement
- Phase 7 Plan 1: Interactive-Layer Deps & Dual-TTY Gate Summary
- Phase 7 — Validation Strategy
- Phase 08 Plan 08: Re-run Head-to-Head Benchmarks vs TS 1.3.1 (REL-03) Summary
- Phase 08 — Security
- 09-08 — COMPLETE. v0.2.0 published and verified.
- Locked Decisions
- Phase 10 — Validation Strategy
- Goal Achievement
- Roadmap: CodeGraph Go
- Phase Details
- v1.0 Standing Snapshot — Phases 1-5
- README.md
- Contributing
- main.go
- NewCachingDetector
- parser_test.go
- Phase 2 — Validation Strategy
- Phase 4: Output Hygiene - Discussion Log
- Phase 07: Code Review Report
- Phase 10 — Security
- Roadmap: CodeGraph Go
- mcp-capture.mjs
- startWatchdog
- syntheticParityEngine
- Open
- Debug Session: daemon-stale-linux-flake
- Roadmap: CodeGraph Go
- Phase 1 — Validation Strategy
- Phase 3: Watcher-on-MCP Default - Discussion Log
- Info
- Phase 4 — Security
- Phase 6: Rendering Seam & Pretty status/files - Discussion Log
- Fixed Issues
- Architecture Patterns
- Fixed Issues
- Fixed Issues
- Narrative Findings (AI reviewer)
- 09-07 — SKIPPED (not executed)
- Architecture Patterns
- Phase 09: Code Review Report
- fix.md
- PeakRSSBytes
- Milestone v1.0 — Interim Audit (after Phase 3 of 8)
- Phase 1: Behavioral Parity — explore & node - Discussion Log
- Phase 1 — Security
- Architecture Patterns
- Phase 3 — Security
- Architecture Patterns
- Phase 5 — Security
- Phase 6 — Security
- Common Pitfalls
- Phase 7 — Security
- 07-UAT.md
- 08-UAT.md
- Architecture Patterns
- Future Requirements
- enhancement.md
- feature.md
- stubStore
- closeOverServeReachableImports
- fsatomic_test.go
- TestPublishReleaseStepBranches
- Phase 1: Behavioral Parity — explore & node Fix Summary
- Phase 3: Code Review Fix Report (Iteration 2)
- Phase 05: Code Review Report (Iteration 5 — Round-5 Fix Verification / Round-6 Final Clean-Check)
- Phase 05: Code Review Fix Report
- Phase 05: Code Review Fix Report
- Phase 05: Code Review Report (Iteration 2 — Post-Fix Re-Review)
- Phase 6: Code Review Report (re-run — iteration 2, verifies WR-01..WR-04 fixes)
- Info
- Architecture Patterns
- Common Pitfalls
- Fixed Issues
- Fixed Issues
- 09-01-PLAN.md
- Common Pitfalls
- 10-UAT.md
- SEED-002: we need a homebrew installation path
- ledger.go
- UserAccountManager
- TestMigrateCmdRefusesUnrecognizedTargetWithoutForce
- TestProgressCLINonTTYReachability
- telemetry_test.go
- version_test.go
- Scope-Expansion Addendum (D-09/D-10)
- Architecture Patterns
- SURF-06: MCP Markdown Output (added 2026-07-15 — scope change, D-16)
- Validation Architecture
- Common Pitfalls
- 06-02-PLAN.md
- Validation Architecture
- 2026-07-31-rebless-perf-baseline-on-ubuntu-latest.md
- Security Policy
- recoverAccount
- loadCharmClosure
- [0.2.0](https://github.com/seanb4t/codegraph-go/compare/v0.1.0...v0.2.0) (2026-08-01)
- ⚠️ Wrong template — please pick the one for your PR type
- notice_test.go
- Common Pitfalls
- Validation Architecture
- Common Pitfalls
- Phase 04: Code Review Report
- 05-05-PLAN.md
- Validation Architecture
- Code Examples
- Validation Architecture
- Code Examples
- Validation Architecture
- 09-02-PLAN.md
- 09-03-PLAN.md
- Common Pitfalls
- API Coverage — Phase 9 (release-please + GoReleaser)
- Validation Architecture
- API Coverage — Phase 10 (local build, CONTRIBUTING, Taskfile.yml)
- 260731-0a0-PLAN.md
- release-please-config.json
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 02-01-PLAN.md
- 02-02-PLAN.md
- 02-03-PLAN.md
- 02-04-PLAN.md
- 02-05-PLAN.md
- 02-06-PLAN.md
- 02-07-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Code Examples
- Sources
- 03-01-PLAN.md
- 03-02-PLAN.md
- 03-03-PLAN.md
- 03-04-PLAN.md
- 03-05-PLAN.md
- Phase 3: Code Review Report (Round 6 — FINAL re-review, iteration 3 of --auto)
- 04-01-PLAN.md
- 04-02-PLAN.md
- 04-03-PLAN.md
- 05-01-PLAN.md
- 05-02-PLAN.md
- 05-03-PLAN.md
- 05-04-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- Archtest precedents to mirror (mechanism, not content)
- CLI seam points to modify (long-running op + summary)
- 07-01-PLAN.md
- 07-02-PLAN.md
- 07-03-PLAN.md
- 07-04-PLAN.md
- 07-05-PLAN.md
- 07-06-PLAN.md
- 07-07-PLAN.md
- 07-08-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 08-01-PLAN.md
- 08-02-PLAN.md
- 08-03-PLAN.md
- 08-04-PLAN.md
- 08-05-PLAN.md
- 08-06-PLAN.md
- 08-07-PLAN.md
- 08-08-PLAN.md
- 08-09-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Sources
- 09-04-PLAN.md
- 09-05-PLAN.md
- 09-06-PLAN.md
- 09-07-PLAN.md
- 09-08-PLAN.md
- Standard Stack
- User Constraints (from CONTEXT.md)
- Code Examples
- Sources
- Standard Stack
- Sources
- 260728-rfx-PLAN.md
- 2026-07-14-document-release-cut-procedures-runbook.md
- 2026-07-31-bisect-indexer-throughput-regression.md
- pr_template_policy.py
- synthetic-parity corpus
- Mismatch
- TestMain
- watchdog_windows.go
- GSD Debug Knowledge Base
- 01-01-PLAN.md
- 01-02-PLAN.md
- 01-03-PLAN.md
- 01-04-PLAN.md
- 01-05-PLAN.md
- 01-06-PLAN.md
- 01-07-PLAN.md
- 01-08-PLAN.md
- 01-09-PLAN.md
- 01-10-PLAN.md
- 01-11-PLAN.md
- 01-12-PLAN.md
- 01-13-PLAN.md
- 01-14-PLAN.md
- 01-15-PLAN.md
- 01-16-PLAN.md
- 01-17-PLAN.md
- Security Domain
- Security Domain
- Security Domain
- Security Domain
- Security Domain
- Security Domain
- 10-01-PLAN.md
- 10-02-PLAN.md
- 10-03-PLAN.md
- 10-04-PLAN.md
- 10-05-PLAN.md
- 10-06-PLAN.md
- 10-07-PLAN.md
- Security Domain
- Code Examples
- synthetic-parity

## God Nodes (most connected - your core abstractions)
1. `Node` - 375 edges
2. `WriteFile()` - 99 edges
3. `NodeID()` - 92 edges
4. `execCmd()` - 88 edges
5. `Edge` - 83 edges
6. `fakeHome()` - 66 edges
7. `New()` - 63 edges
8. `readFile()` - 51 edges
9. `Communities (60 total, 13 thin omitted)` - 45 edges
10. `Remove()` - 44 edges

## Surprising Connections (you probably didn't know these)
- `runOnce()` --calls--> `PeakRSSBytes()`  [INFERRED]
  tools/bench/runner/main.go → internal/bench/rss.go
- `copyFixture()` --calls--> `WriteFile()`  [INFERRED]
  test/integration/main_test.go → internal/fsatomic/fsatomic.go
- `TestLiveEditAutoSyncReachesExplore()` --calls--> `WriteFile()`  [INFERRED]
  test/integration/watch_live_sync_test.go → internal/fsatomic/fsatomic.go
- `writeCapture()` --calls--> `WriteFile()`  [INFERRED]
  testdata/golden/gocapture/main.go → internal/fsatomic/fsatomic.go
- `copyDir()` --calls--> `WriteFile()`  [INFERRED]
  testdata/golden/golden_parity_test.go → internal/fsatomic/fsatomic.go

## Import Cycles
- 1-file cycle: `internal/indexer/testdata/routesfixture/django/urls.py -> internal/indexer/testdata/routesfixture/django/urls.py`

## Communities (862 total, 80 thin omitted)

### Community 0 - "fakeHome"
Cohesion: 0.05
Nodes (105): T, TestAntigravity_DescribePaths_GlobalOnly(), TestAntigravity_GlobalOnly(), TestAntigravity_ID(), TestAntigravity_Install_EntryHasNoTypeField(), TestAntigravity_Install_NoOpForLocal(), TestAntigravity_Install_SweepsStaleLegacyEntry(), TestAntigravity_Install_UnifiedWriteFailure_PreservesLegacyEntryNoMarker() (+97 more)

### Community 1 - "Parser"
Cohesion: 0.06
Nodes (14): extractor, cDeclaratorName(), findFunctionDeclarator(), firstNamedChild(), FileResult, isLikelyTypeName(), splitQualifiedMethod(), topLevelDecls() (+6 more)

### Community 2 - "readLock"
Cohesion: 0.12
Nodes (21): DebounceDuration(), Context, Duration, Mutex, NewDebouncer(), T, TestDebounceAddAfterCancelIsNoOp(), TestDebounceCoalescesBurst() (+13 more)

### Community 3 - "main.go"
Cohesion: 0.05
Nodes (93): Metrics, CheckRegression(), platformString(), runnerString(), scratchFSString(), containsFold(), T, TestCheckRegression() (+85 more)

### Community 4 - "execCmd"
Cohesion: 0.21
Nodes (27): fakeHome(), Command, Location, T, readFileString(), readJSONMap(), TestInstall_AutoAllow_TogglesPermission(), TestInstall_ExecPathAppearsInWrittenConfig() (+19 more)

### Community 5 - "Extract"
Cohesion: 0.13
Nodes (12): extractor, countParams(), defaultPackageAlias(), FileResult, isExported(), isGoPredeclaredType(), namedTypeRef(), receiverTypeName() (+4 more)

### Community 6 - "Phase 8: Release Hardening & Benchmarks - Research"
Cohesion: 0.04
Nodes (46): Alternatives Considered, Anti-Patterns to Avoid, Applicable ASVS Categories, Architectural Responsibility Map, Architecture Patterns, Assumptions Log, Code Examples, Common Pitfalls (+38 more)

### Community 7 - "Node"
Cohesion: 0.14
Nodes (8): FileResult, isLikelyTypeName(), namespaceOverride(), phpSimpleAndFullName(), phpVariableName(), topLevelDecls(), walkDescendants(), extractor

### Community 8 - "Communities (60 total, 13 thin omitted)"
Cohesion: 0.04
Nodes (45): Communities (60 total, 13 thin omitted), Community 0 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Research", Community 10 - "CLAUDE.md", Community 11 - "Project State", Community 12 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log", Community 13 - "Architecture Patterns", Community 14 - "Project Research Summary", Community 15 - "Common Pitfalls" (+37 more)

### Community 9 - "TargetID"
Cohesion: 0.29
Nodes (21): TargetID, AllTargetIDs(), AllTargets(), DetectAll(), GetTarget(), Location, registerTarget(), ResolveTargetFlag() (+13 more)

### Community 10 - "SynthesizeImplements"
Cohesion: 0.19
Nodes (22): InterfaceSpecs, TypeMethods, ExtractedNode, FileResult, IntraEdge, MethodSpec, UnresolvedRef, candidateInterfaces() (+14 more)

### Community 11 - "File"
Cohesion: 0.05
Nodes (9): stubWriter, file_internal_schema_graph_proto_init(), file_internal_schema_graph_proto_rawDescGZIP(), init(), MessageState, File, Meta, SizeCache (+1 more)

### Community 12 - "newRootCmd"
Cohesion: 0.08
Nodes (30): collectAffectedFiles(), Command, newAffectedCmd(), Command, newCalleesCmd(), Command, newCallersCmd(), Command (+22 more)

### Community 13 - "New"
Cohesion: 0.04
Nodes (10): EdgeIterator, FileIndexIterator, FileIterator, NodeIterator, Reader, expandFakeReader, gatherFakeReader, scoringFakeReader (+2 more)

### Community 14 - "Extract"
Cohesion: 0.12
Nodes (22): Extract(), FileResult, isRequireCall(), rubyConstantName(), rubyStringContent(), findNode(), FileResult, T (+14 more)

### Community 15 - "Extract"
Cohesion: 0.20
Nodes (9): firstChildOfKind(), firstNamedChild(), FileResult, isLikelyTypeName(), kotlinDeclKind(), kotlinTypeName(), packageOverride(), walkDescendants() (+1 more)

### Community 16 - "BuildServer"
Cohesion: 0.22
Nodes (19): T, TestReconnectReconcile(), ParseAllowlist(), copyFixture(), equalStrings(), Client, Context, MCPServer (+11 more)

### Community 17 - "BuildTSIndex"
Cohesion: 0.21
Nodes (23): OpenSource(), sourceDSN(), buildNonTSDB(), buildSchemaVersionsDB(), T, hashFile(), openSourceT(), TestCountDistinctEdges() (+15 more)

### Community 18 - "extractor"
Cohesion: 0.13
Nodes (12): tsModuleKey(), isDirectlyExported(), isLikelyTypeName(), methodVisibility(), NormalizeModuleKey(), resolveModuleSpecifier(), resolvePathsAlias(), typeRefFromExpr() (+4 more)

### Community 19 - "shared_test.go"
Cohesion: 0.17
Nodes (32): atomicWriteFile(), FileResult, normalizeJSON(), readJSONFile(), removeMarkedSection(), removeMcpEntry(), replaceOrAppendMarkedSection(), T (+24 more)

### Community 20 - "golden_parity_test.go"
Cohesion: 0.08
Nodes (59): corpusCandidate, goldenCapture, locTuple, multiDefBlock, nameFileLine, assertExploreShape(), assertNameFileLineSubset(), assertNodeMultiDefShape() (+51 more)

### Community 21 - "OpenAt"
Cohesion: 0.20
Nodes (17): copyFixture(), T, indexFixture(), TestClampDepth(), TestOpenAt(), TestResolveCodegraphDir(), TestValidateDepth(), TestValidateKind() (+9 more)

### Community 22 - "Pattern Assignments"
Cohesion: 0.06
Nodes (31): Additive-only schema evolution, Error-sentinel + `%w`-wrapped guidance messages, File Classification, `internal/cli/daemon.go` (NEW — thin Cobra command), `internal/cli/root.go` (wire new commands), `internal/cli/status.go` (modify — staleness field), `internal/cli/sync.go` (NEW — thin Cobra command), `internal/cli/unlock.go` (NEW — thin Cobra command) (+23 more)

### Community 23 - ".Install"
Cohesion: 0.15
Nodes (22): codexTarget, codexConfigPath(), codexInstructionsPath(), codexTableBody(), Location, init(), readFileOrEmpty(), findTOMLTableRange() (+14 more)

### Community 24 - "translate_test.go"
Cohesion: 0.16
Nodes (29): scanAndWriteTable(), asBool(), asInt64(), asString(), edgeFromRow(), fileFromRow(), flattenMetadata(), msFromString() (+21 more)

### Community 25 - "Feature Research — v1.0 Drop-in CLI Parity + Human TUI"
Cohesion: 0.06
Nodes (30): 1. `explore` — semantic-relevance selection algorithm, 1a. Multi-word query tokenization (`extractSymbolsFromQuery`, `context/index.js:64-145`), 1b. Hybrid search — 5 ranking channels merged (`findRelevantContext`, `context/index.js:433-870`), 1c. Graph expansion beyond lexical matches — THE core "semantic relevance" mechanism, 1d. "⚠️ no covering tests" warning (`buildBlastRadiusSection`, `mcp/tools.js:2254-2304`), 1e. Rendering (file-grouping, whole-file-vs-skeleton, per-file budget) — lower priority for the algorithm spec, higher for output-shape, 2. `node` — multi-definition disambiguation, 2a. Candidate selection (`findSymbolMatches`) (+22 more)

### Community 26 - "pebble_store.go"
Cohesion: 0.08
Nodes (15): FileIndexEntry, getter, pebbleEdgeIterator, pebbleFileIndexIterator, pebbleFileIterator, pebbleNodeIterator, decodeSegment(), decodeFileIndexKey() (+7 more)

### Community 27 - "NodeID"
Cohesion: 0.24
Nodes (36): Extract(), appendSegment(), NodeID(), resolveRefs(), findEdge(), findEdgeFull(), fixtureResults(), FileResult (+28 more)

### Community 28 - "WriteResult"
Cohesion: 0.14
Nodes (11): DetectionResult, fakeTarget, FileAction, FileResult, InstallOptions, Location, WriteResult, Location (+3 more)

### Community 29 - "stringValue"
Cohesion: 0.13
Nodes (28): aspnetAttributePath(), init(), walkAspNetRoutes(), flaskFirstStringArg(), flaskRouteMethodsVerb(), init(), pythonReferenceName(), walkDjangoRoutes() (+20 more)

### Community 30 - "Architecture Research: v1.0 Integration into codegraph-go"
Cohesion: 0.07
Nodes (28): Anti-Pattern 1: Rendering inside `internal/query` or `internal/mcp`, Anti-Pattern 2: Duplicating relevance/disambiguation logic per surface, Anti-Pattern 3: Auto-installing git hooks by default, Anti-Pattern 4: A `present`-package dependency cycle back into foundational packages, Anti-Patterns, Architectural Patterns, Architecture Research: v1.0 Integration into codegraph-go, Component Responsibilities (v1.0 deltas only) (+20 more)

### Community 31 - "targetRoot"
Cohesion: 0.18
Nodes (13): Command, newIndexCmd(), Command, Stats, newInitCmd(), printSummary(), printWatchFallbackAdvisory(), writeGitignoreHint() (+5 more)

### Community 32 - "extractor"
Cohesion: 0.13
Nodes (9): FileResult, isLikelyTypeName(), ownPackageFor(), packageUp(), pythonTypeRefFromExpr(), pythonVisibility(), unwrapDecorated(), walkDescendants() (+1 more)

### Community 33 - "Technology Stack"
Cohesion: 0.07
Nodes (26): Addendum: v1.0 Milestone — Human TUI & Parity Stack, Alternatives Considered, Core Technologies, Core Technologies, Git Hooks (post-commit/merge/checkout), Git / Worktree Detection — stdlib `os/exec`, no new dependency, Installation, Installation (+18 more)

### Community 34 - "goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed"
Cohesion: 0.08
Nodes (25): Alternatives Considered, Architecture, Constraints, Conventions, Core, Core Technologies, Dev / CI tooling (not go.mod deps — install as CLI tools in CI), Developer Profile (+17 more)

### Community 35 - "GraphStore"
Cohesion: 0.25
Nodes (19): dropDanglingEdges(), findDangling(), isFileEndpoint(), nodeIDSet(), scanDangling(), buildStoreFromSource(), T, openHappySource() (+11 more)

### Community 36 - "Sync"
Cohesion: 0.16
Nodes (34): assertFileIndexEmpty(), assertFileIndexHasEdge(), assertNoOrphansOrDangling(), copyFixture(), Reader, T, TestPruneFixtures(), TestSyncIndexesCrossFileContainsEdgeUnderOwnerFile() (+26 more)

### Community 37 - "Run"
Cohesion: 0.20
Nodes (27): GraphStore, advanceCursorChecked(), checkTargetOverwrite(), countFileIndexEdges(), finishFromComplete(), getStoreMeta(), Options, Reader (+19 more)

### Community 38 - "Implementation Decisions"
Cohesion: 0.08
Nodes (24): Benchmark corpus — split by purpose (PERF-01 vs PERF-02/INDX-06), Benchmark inputs, Build → sign → provenance → SBOM wiring (DIST-02, DIST-03), Canonical References, CGo cross-compilation — the central technical risk (DIST-01), Claude's Discretion, Deferred Ideas, Established Patterns (+16 more)

### Community 39 - "extractor"
Cohesion: 0.15
Nodes (10): extractor, chainRoot(), containsMod(), csharpQualifiedParts(), csharpVisibility(), findNamespace(), FileResult, isLikelyTypeName() (+2 more)

### Community 40 - "Discover"
Cohesion: 0.20
Nodes (21): parserFactory, Discover(), T, mustWrite(), TestDiscover_Deterministic(), TestDiscover_Fixture(), TestDiscover_ImportPaths(), TestDiscover_MissingGoMod_FallsBackToPathIdentity() (+13 more)

### Community 41 - "parseFixture"
Cohesion: 0.09
Nodes (30): T, TestAspNet_DetectsHttpVerbAttributes(), TestAspNet_SignatureOptIn(), T, TestDjango_DetectsUrlpatternsEntries(), TestDjango_SignatureOptIn(), TestFastAPI_DetectsDirectVerbDecorator(), TestFastAPI_SignatureOptIn() (+22 more)

### Community 42 - "traverse.go"
Cohesion: 0.20
Nodes (21): buildContainsIndex(), BuildImplementsIndex(), BuildReverseAdjacency(), dispatchSiblingIDs(), Location, Engine, Reader, implementedInterfaces() (+13 more)

### Community 43 - "Extract"
Cohesion: 0.29
Nodes (16): Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser(), TestExtract_Calls() (+8 more)

### Community 44 - "Extract"
Cohesion: 0.12
Nodes (37): packageJSONFile, tsconfigCompilerOptions, tsconfigFile, tsProjectDescriptor, readTSDescriptor(), tsDescriptor(), T, TestTSInstantiates() (+29 more)

### Community 45 - "traverse_test.go"
Cohesion: 0.11
Nodes (30): New(), affectedDepthFixtureNodesEdges(), assertJSONArrayNotNull(), dispatchFixtureNodesEdges(), Engine, T, TestAffected(), TestAffected_DispatchTraversal() (+22 more)

### Community 46 - "Implementation Decisions"
Cohesion: 0.09
Nodes (22): Canonical References, Claude's Discretion, Daemon, Lock & unlock (SYNC-04, SYNC-05), Deferred Ideas, Established Patterns, Existing Code Insights, Implementation Decisions, Incremental Sync Engine (INDX-03) (+14 more)

### Community 47 - "Pattern Assignments"
Cohesion: 0.09
Nodes (22): Cobra thin-command delegation (cross-cutting: all 5 new `internal/cli/*.go` files), Coverage, Doc-comment convention citing decision IDs, File Classification, File Created, Interactive-prompt-with-non-interactive-fallback (cross-cutting: `install.go`'s D-03 multi-select), `internal/agents/registry.go` (service/registry, CRUD), `internal/agents/shared.go` — JSON/marker surgical write helpers (utility, file-I/O) (+14 more)

### Community 48 - "Phase Details"
Cohesion: 0.22
Nodes (9): Phase 1: Foundation — Storage, Schema & Parser Strategy, Phase 2: Go Indexing Pipeline, Phase 3: Query Engine & MCP Server, Phase 4: Incremental Sync & File Watcher, Phase 5: Language Coverage & Resolution Breadth, Phase 6: Agent Integrations & CLI Lifecycle, Phase 7: Migration Tool, Phase 8: Release Hardening & Benchmarks (+1 more)

### Community 49 - "pebbleWriter"
Cohesion: 0.09
Nodes (59): blockDeclaresKey(), blockReferencesToken(), checkStepInvokesTask(), contains(), T, mustParseCheckCrossPairs(), mustParseContributingTaskTargets(), mustParseGoreleaserCrossPairs() (+51 more)

### Community 50 - "keys.go"
Cohesion: 0.09
Nodes (26): Batch, pebbleReader, pebbleWriter, deterministicMarshal(), Message, TestFileIndexEncodingRejectsCollision(), T, TestEdgeKeyRangeScanOrdering() (+18 more)

### Community 51 - "extractor"
Cohesion: 0.16
Nodes (9): findModifiersChild(), findPackageDeclaration(), FileResult, isLikelyTypeName(), javaVisibility(), simpleTypeName(), splitJavaFQN(), walkDescendants() (+1 more)

### Community 52 - "Extract"
Cohesion: 0.22
Nodes (24): T, TestJavaInstantiates(), TestJavaReferences(), TestJavaReturns(), TestJavaTypeOf(), Extract(), findNode(), FileResult (+16 more)

### Community 53 - "render_markdown.go"
Cohesion: 0.13
Nodes (29): formatNodeRef(), joinNodeRefs(), joinSymbolKindList(), renderBlastBullet(), RenderExplore(), RenderNode(), RenderNodeMultiDef(), renderNodeSection() (+21 more)

### Community 54 - "Implementation Decisions"
Cohesion: 0.09
Nodes (21): Canonical References, Claude's Discretion, Coverage policy & documentation, Cross-file resolution (per-language, tiered), Deferred Ideas, Dynamic-dispatch synthesis & provenance (RES-02 / RES-03), Established Patterns, Existing Code Insights (+13 more)

### Community 55 - "Implementation Decisions"
Cohesion: 0.09
Nodes (21): Agent Registry & Command Surface (AGNT-01, AGNT-03), Canonical References, Claude's Discretion, CLI Ergonomics (CLI-01), Deferred Ideas, Established Patterns, Existing Code Insights, Existing Substrate (bind against — do not re-derive) (+13 more)

### Community 56 - "Implementation Decisions"
Cohesion: 0.09
Nodes (21): Canonical References, Claude's Discretion, Command Surface + Source Detection + Output (MIGR-01), Corpus (validation-target candidates), Deferred Ideas, Established Patterns, Existing Code Insights, Field Mapping — what carries vs. what drops (+13 more)

### Community 57 - "v1 Requirements"
Cohesion: 0.09
Nodes (21): Agent Integrations, Architecture Future-Proofing, Auto-Sync & Daemon, CLI & Lifecycle, Distribution & Supply Chain, Graph Resolution, Indexing Core, Language Support (+13 more)

### Community 58 - "fileExists"
Cohesion: 0.22
Nodes (13): opencodeTarget, fileExists(), FileResult, Location, init(), opencodeConfigPath(), opencodeInstructionsPath(), opencodeMcpEntry() (+5 more)

### Community 59 - "Writer"
Cohesion: 0.29
Nodes (9): Writer, exportNamespace(), pebbleStore, Message, Reader, Snapshot, Import(), importRecord() (+1 more)

### Community 60 - "Open"
Cohesion: 0.13
Nodes (27): fileIndexEntrySeen, T, TestBulkExportIsConsistentUnderConcurrentWrite(), TestBulkExportReimportsLosslessly(), collectFileIndex(), T, TestFileIndexDeleteFileSubgraphPrunesXNamespace(), TestFileIndexPointDeletes() (+19 more)

### Community 61 - "extractor"
Cohesion: 0.18
Nodes (7): FileResult, isLikelyTypeName(), joinRustPath(), rustTypeName(), rustVisibility(), walkDescendants(), extractor

### Community 62 - "migrate_test.go"
Cohesion: 0.20
Nodes (31): buildDanglingUnderFileDB(), T, mustAbs(), openTargetStore(), runTarget(), TestRun_AgedDB(), TestRun_ClosesSourceBeforeSwap(), TestRun_DanglingFailsLoud() (+23 more)

### Community 63 - "Implementation Decisions"
Cohesion: 0.10
Nodes (20): Canonical References, Claude's Discretion, Deferred Ideas, Established Patterns, Existing Code Insights, External Ground Truth, Golden Corpus & TS DDL Capture, GraphStore Interface & Concurrency (+12 more)

### Community 64 - "Implementation Decisions"
Cohesion: 0.10
Nodes (20): Canonical References, Captured TS Ground Truth (parity vocabulary + id/edge shapes), Claude's Discretion, CLI Lifecycle & Commands (INDX-01, INDX-02), Deferred Ideas, Edge Multiplicity — settling the Phase-1-deferred key-identity question, Established Patterns, Existing Code Insights (+12 more)

### Community 65 - "Implementation Decisions"
Cohesion: 0.10
Nodes (20): `affected` / `files` — no golden oracle (QRY-06, QRY-07), Canonical References, Claude's Discretion, Command Surface & CLI Wiring (QRY-01…09, MCP-01), Deferred Ideas, Established Patterns, Existing Code Insights, Golden Corpus — the parity oracle (MCP-04) (+12 more)

### Community 66 - "Pattern Assignments"
Cohesion: 0.10
Nodes (20): Archtest interface-boundary discipline, Build-once in-memory map index, `cmd.OutOrStdout()` / `cmd.SetOut` output discipline, Cobra thin-command delegation, File Classification, `internal/cli/cli_test.go` — test scaffolding to extend, `internal/cli/{query,node,search,callers,callees,impact,affected,files,status,explore}.go` — thin Cobra commands, `internal/graphstore/store.go` + `pebble_store.go` — `IterateNodes`/`IterateFiles` (D-03) (+12 more)

### Community 67 - "Phase 5 Plan 03: Resolution Breadth — Module-Key Seam + WR-01/WR-02 Fixes Summary"
Cohesion: 0.10
Nodes (20): Accomplishments, Auto-fixed Issues, Decisions Made, deferred Go fixes (D-05) — no Java/C#/Python/TS extraction or resolution, Dependency graph, Deviations from Plan, exists yet. Full per-language requirement completion happens in the, Files Created/Modified (+12 more)

### Community 68 - "Pattern Assignments"
Cohesion: 0.10
Nodes (20): Deterministic sort-then-collapse, File Classification, `internal/indexer/discover.go` (MODIFIED — generalize ext→lang walk + descriptor hooks, D-03), `internal/indexer/dispatch/implements.go` (NEW — RES-02/RES-03), `internal/indexer/extract.go` (MODIFIED — fix Pitfall 1, per-file parser+extractor selection), `internal/indexer/goextract/types.go` (shared vocabulary — extend additively, do not fork), `internal/indexer/javaextract/javaextract.go` (and csharpextract/pyextract/tsextract — same shape), `internal/indexer/languages.go` (NEW — LanguageSpec registry, D-01) (+12 more)

### Community 69 - ".Install"
Cohesion: 0.21
Nodes (13): claudeTarget, addClaudeAllowPermission(), claudeConfigPath(), claudeInstructionsPath(), claudeLegacyLocalConfigPath(), claudeSettingsPath(), FileResult, Location (+5 more)

### Community 70 - "hermes.go"
Cohesion: 0.20
Nodes (13): hermesTarget, Location, hermesAppendCliToolset(), hermesCodegraphChildBlock(), hermesConfigPath(), hermesConfigured(), hermesRemoveCliToolset(), hermesSpliceMcpServersBlock() (+5 more)

### Community 71 - "Generate"
Cohesion: 0.22
Nodes (17): Options, Stats, Rand, Generate(), generateGo(), generateJS(), generatePython(), PlannedFileCount() (+9 more)

### Community 72 - "Extract"
Cohesion: 0.28
Nodes (17): Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser(), TestExtract_Calls() (+9 more)

### Community 73 - "Extract"
Cohesion: 0.28
Nodes (20): stubOversizedParser, Extract(), languageForExt(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved() (+12 more)

### Community 74 - "runAndSnapshot"
Cohesion: 0.28
Nodes (17): T, TestDjango_RouteDetectionEndToEnd(), TestExpress_RouteDetectionEndToEnd(), TestFastAPI_RouteDetectionEndToEnd(), TestFlask_RouteDetectionEndToEnd(), TestNest_RouteDetectionEndToEnd(), T, TestAspNet_RouteDetectionEndToEnd() (+9 more)

### Community 75 - "search.go"
Cohesion: 0.15
Nodes (16): Engine, lexicalMatchTier(), MarshalQueryJSON(), renderQueryNodeJSON(), clampAffectedDepth(), clampDepth(), clampMaxFiles(), ValidateAffectedFiles() (+8 more)

### Community 76 - "Phase 5 Plan 01: Multi-Language Seam Foundation Summary"
Cohesion: 0.10
Nodes (19): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, extraction+resolution lands in later Phase 5 plans; REQUIREMENTS.md checkboxes, Files Created/Modified, frontmatter requirements field) are NOT fully satisfied by this plan — it lands (+11 more)

### Community 77 - "Pattern Assignments"
Cohesion: 0.10
Nodes (19): Additive-only namespace/schema discipline, Atomic same-filesystem temp+rename swap, Error wrapping convention, File Classification, `internal/cli/migrate.go` (cobra command), `internal/migrate/migrate.go` (orchestration), `internal/migrate/migrate_test.go` + fixture harness (test), `internal/migrate/progress.go` (progress record) (+11 more)

### Community 78 - "Pitfalls Research"
Cohesion: 0.10
Nodes (19): Critical Pitfalls, Integration Gotchas, "Looks Done But Isn't" Checklist, Performance Traps, Pitfall 1: ANSI/TUI styling bleeding into the AGENT path (breaks `codegraph_explore` and every piped consumer), Pitfall 2: Regressing proven output parity while "improving" explore relevance / node disambiguation (v0.1's golden test has a blind spot), Pitfall 3: Watcher-default flip breaks the daemon lockfile interplay / double-watches / stalls MCP startup, Pitfall 4: Worktree detection FALSE POSITIVES (submodules, nested clones, monorepo subdirs, symlinks) (+11 more)

### Community 79 - "CGoParser"
Cohesion: 0.25
Nodes (19): CGoParser, init(), T, parseFixture(), Once, newCGoParser(), NewCParser(), NewCppParser() (+11 more)

### Community 80 - "Per-Language Gaps"
Cohesion: 0.13
Nodes (15): `c`, `cpp`, `csharp`, `go`, `java`, `javascript`, `kotlin`, Per-Language Gaps (+7 more)

### Community 81 - "Pattern Assignments"
Cohesion: 0.11
Nodes (18): Additive-Only Schema Discipline, Architectural Boundary Enforcement, Close-once / Idempotent Teardown, Error Sentinels, File Classification, `internal/cli/*.go` (controller, request-response) — Cobra command tree, `internal/indexer/extract.go` (service, parallel/batch), `internal/indexer/nodeid.go` (utility, transform) (+10 more)

### Community 82 - "Phase 5 Plan 02: Multi-Language Discovery + Extraction Worker-Pool Fix Summary"
Cohesion: 0.11
Nodes (18): Accomplishments, consume this seam., Decisions Made, Dependency graph, Deviations from Plan, discovery/extraction seam generalization (Wave A per 05-RESEARCH.md's, Files Created/Modified, Full per-language requirement completion happens in the Wave B plans that (+10 more)

### Community 83 - "Warnings"
Cohesion: 0.11
Nodes (18): CR-01: Install/Uninstall silently swallow every file I/O error across all 8 agent targets, CR-02: antigravity legacy→unified migration can lose the codegraph entry entirely on a partial write failure, CR-03: Hermes YAML block matching is not CRLF-safe — breaks install idempotency and disables uninstall entirely for CRLF config files, Critical Issues, IN-01: `checkWritable` is invoked twice per upgrade run, IN-02: `hermesAppendCliToolset`'s `cli:` search assumes a fixed 2-space parent indent, Info, Phase 06: Code Review Report (+10 more)

### Community 84 - "matrix_test.go"
Cohesion: 0.22
Nodes (15): CapabilityEntry, Coverage, All(), Lookup(), T, goldenTestFuncNames(), repoRoot(), TestAll() (+7 more)

### Community 85 - "resolveRefsWithIndex"
Cohesion: 0.19
Nodes (19): edgeTriple, pendingCall, collapseEdges(), FileResult, isIntraModule(), lastPathSegment(), Resolve(), resolveNameRef() (+11 more)

### Community 86 - "lookupLanguageByID"
Cohesion: 0.26
Nodes (20): LanguageSpec, ProjectDescriptor, FileResult, lookupLanguageByExt(), lookupLanguageByID(), T, TestKindRouteAdditive(), TestLanguageRegistry() (+12 more)

### Community 87 - "extractor"
Cohesion: 0.22
Nodes (6): FileResult, isLikelyTypeName(), swiftDeclKind(), swiftSimpleTypeName(), walkDescendants(), extractor

### Community 88 - ".Install"
Cohesion: 0.23
Nodes (9): antigravityTarget, antigravityConfigPath(), antigravityEntry(), antigravityLegacyPath(), antigravityMigratedMarker(), antigravityUnifiedPath(), Location, init() (+1 more)

### Community 89 - "hasEdge"
Cohesion: 0.27
Nodes (15): exportFrame, decodeExportFrames(), encodeExportFrames(), Reader, T, hasEdge(), indexAndExport(), reportFirstDiff() (+7 more)

### Community 90 - "symbolIndex"
Cohesion: 0.21
Nodes (11): symbolIndex, FileResult, Reader, newSymbolIndex(), newSymbolIndexFromStore(), fooDeclResult(), FileResult, T (+3 more)

### Community 91 - "Critical Issues"
Cohesion: 0.12
Nodes (15): CR-01: Every parsed tree-sitter Tree leaks its underlying C memory, CR-02: CGo backend can return a "successful" Tree wrapping a nil native tree, CR-03: Import performs an unbounded allocation driven by an untrusted length prefix, CR-04: Close() races with Snapshot()/NewWriter()/Commit() panic instead of returning errors, Critical Issues, IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match, IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`, IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns (+7 more)

### Community 92 - "Phase 4 Plan 4: Rename/Delete/Move Prune Fixture Suite + Sync-Equals-Reindex Determinism Gate Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 93 - "Warnings"
Cohesion: 0.12
Nodes (16): CR-01: Daemon.Run can release the lock while a Sync is still writing (untracked flush goroutine), CR-02: `daemon/lock.go` `acquire()` has a TOCTOU race — two daemons can both "win" the lock, CR-03: Incremental `Sync` drops the `x/` file-index entry for cross-file `contains` edges — permanent dangling edges, CR-04: `pruneOwnedEdgesOnly` deletes `e/` edge records but never the matching `x/` index entries — index drift/leak on every dependent re-sync, Critical Issues, IN-01: `Watcher.root` field is dead code, IN-02: `CalleesResult`/`CallersResult`/`ImpactResult` cap logic applies `limit` before `MaxLimit`, silently reordering which truncation "wins" when both are tighter than the actual result count, Info (+8 more)

### Community 94 - "Phase 5 Plan 13: Language Capability Matrix + Cross-Language Regression Closeout (LANG-06) Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Coverage Headline, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered (+8 more)

### Community 95 - "Phase 7: Migration Tool - Research"
Cohesion: 0.12
Nodes (16): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 7: Migration Tool - Research (+8 more)

### Community 96 - "Fixed Issues"
Cohesion: 0.12
Nodes (16): CR-01: In-place migration holds the source DB open across the directory swap, Fixed Issues, Fixed Issues, IN-02: `targetPopulated` treated an unreadable target as empty (bypasses D-08 guard), Phase 7: Code Review Fix Report, Phase 7: Code Review Fix Report — Iteration 2 (re-review), Skipped Issues, Skipped Issues (+8 more)

### Community 97 - "Phase 08 Plan 06: Committed-Baseline Regression Gate Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 98 - "UserController"
Cohesion: 0.15
Nodes (10): ControllerBase, Example, HttpGet, HttpPost, User, UserController, HttpGet, HttpPost (+2 more)

### Community 99 - "Phase 1 Plan 2: Versioned Protobuf Schema Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 100 - "Phase 01 Plan 04: TS CodeGraph v1.3.1 Golden Fixtures Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 101 - "Phase 01 Plan 06: GraphStore Interface + Pebble/v2 Implementation Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 102 - "Phase 3 Plan 8: Query & MCP Command Surface Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auth Gates Encountered, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 103 - "Phase 3 Plan 09: Golden Parity Test Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Known Findings (documented in test comments, not auto-fixed — Rule scope boundary) (+7 more)

### Community 104 - "Phase 4 Plan 7: internal/daemon — Shared Watch/Index Server + Lockfile Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 105 - "Phase 5 Plan 05: C# Extraction + Cross-File Resolution (LANG-03) Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 106 - "Phase 5 Plan 11: Mainstream Grammar Layer — C/C++, Swift, Kotlin (LANG-06) Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 107 - "Phase 5 Plan 12: Framework-Aware Routing (LANG-07) Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 108 - "Phase 08 Plan 01: GoReleaser CGo Build Matrix Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 109 - "Phase 08 Plan 02: RSS Normalization + Metrics Core Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 110 - "Phase 08 Plan 03: Synthetic Corpus Generator Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 111 - "Phase 08 Plan 04: Release Workflow (Native Matrix + Cosign + SLSA + SBOM) Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Decisions Made, Deferred Real-CI Validation (READ BEFORE TRUSTING THIS RELEASE PATH), Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 112 - "Phase 08 Plan 07: Benchmark Runner — Head-to-Head + Regression Gate Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 113 - "Phase 08 Plan 08: CI Gate Workflows Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 114 - "1. Methodology"
Cohesion: 0.12
Nodes (15): 1. Methodology, 2. Raw numbers, 3. Regression gate (PERF-02, INDX-06), Benchmarks: codegraph-go vs TS CodeGraph, Comparison target, Go vs TS 1.3.1 — summary (from the medians above), median-of-5, Peak RSS is measured at the OS level, not in-process (+7 more)

### Community 115 - "newFilesCmd"
Cohesion: 0.08
Nodes (42): Builder, RenderFiles(), fixtureFileEntries(), fixtureFileTree(), T, TestRenderFiles_Empty(), TestRenderFiles_Flat(), TestRenderFiles_Tree() (+34 more)

### Community 116 - "Phase 01 Plan 03: Parser Interface + CGo Tree-sitter Backend Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 117 - "Phase 01 Plan 05: Typed Keyspace & Key-Injection Guard Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 118 - "Phase 2 Plan 03: Go AST Extraction Mapper and Pass 1 Worker Pool Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 119 - "Phase 2 Plan 04: Pass 2 Resolve — Global Symbol Index and Deterministic Cross-File Resolution Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 120 - "Phase 2 Plan 05: Pipeline Orchestration and Determinism Gate Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 121 - "Phase 2 Plan 06: CLI Lifecycle (init/index/uninit) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 122 - "Phase 3 Plan 4: Call-Graph Traversal (Callers/Callees/Impact/Affected) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 123 - "Phase 3 Plan 06: Node/Explore Markdown Rendering Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 124 - "Phase 4 Plan 1: File-Owned Secondary Index Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 125 - "Phase 4 Plan 3: Incremental Sync Engine (internal/indexer.Sync) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 126 - "Phase 4 Plan 5: Native fsnotify Watcher + Env-Tunable Debounce Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 127 - "Phase 4 Plan 8: codegraph sync/daemon/unlock Commands + serve --watch Fallback Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 128 - "Phase 5 Plan 04: Java Extraction + Cross-File Resolution (LANG-02) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 129 - "Phase 5 Plan 06: Python Extraction + Cross-File Resolution (LANG-04) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 130 - "Phase 5 Plan 07: TypeScript/JavaScript Extraction + Cross-File Resolution (LANG-05) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 131 - "Phase 5 Plan 08: Mainstream Grammar Layer (LANG-06) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 132 - "Phase 5 Plan 09: Interface→Implementation Dispatch Synthesis (RES-02/RES-03) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 133 - "Phase 5 Plan 10: Mainstream-Tier Extraction — Rust, Ruby, PHP (LANG-06) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Per-Language Coverage Assessment (for the D-11 capability matrix, Wave F / plan 05-13) (+6 more)

### Community 134 - "Phase 5 Gap Closure: Java + C# Real-Repo D-12 Validation (LANG-02, LANG-03) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 135 - "Phase 6 Plan 04: install/uninstall CLI Commands Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 136 - "Phase 6 Plan 6: Signature-Verified Self-Update (upgrade) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 137 - "Phase 7 Plan 3: Migration Reader + Translation Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 138 - "Phase 7 Plan 6: Migration Orchestration (migrate.Run) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 139 - "Phase 08 Plan 09: Release & Benchmark Documentation Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 140 - "Pattern Assignments"
Cohesion: 0.13
Nodes (14): Doc-comment discipline (package + exported-symbol rationale), Embedded, pinned-commit test/bench fixtures (no network at test/CI time), Fail-loud / no-panic in production paths, File Classification, `internal/bench/rss.go` / `internal/bench/metrics.go` (utility + model, batch measurement), `internal/cli/bench.go` + `internal/cli/root.go` (optional `codegraph bench` CLI subcommand), `internal/upgrade/verify_release_e2e_test.go` (real-signed-artifact e2e test), Metadata (+6 more)

### Community 141 - "Edge"
Cohesion: 0.08
Nodes (5): expandFakeEdgeIterator, scoringFakeEdgeIterator, seedingFakeEdgeIterator, traverseFakeEdgeIterator, Edge

### Community 142 - "detectRoutes"
Cohesion: 0.27
Nodes (11): DiscoveredFile, fileHandlerIndex, buildGlobalHandlerIndex(), csprojManifestText(), detectRoutes(), detectRoutesInFile(), FileResult, newFileHandlerIndex() (+3 more)

### Community 143 - "swap_test.go"
Cohesion: 0.37
Nodes (12): atomicSwapDir(), checkWritableDir(), siblingTempDir(), T, readMarker(), TestAtomicSwapDir_ExistingNonEmptyTarget(), TestAtomicSwapDir_FreshTarget(), TestAtomicSwapDir_RestoreOnFailure() (+4 more)

### Community 144 - "Phase 01 Plan 01: Go Module & Package Skeleton Bootstrap Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 145 - "Phase 2 Plan 01: Node Identity and Schema Field Parity Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 146 - "Phase 2 Plan 02: File Discovery Seam and Shared Test Fixture Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 147 - "Phase 3 Plan 1: Reader Node/File Enumeration Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 148 - "Phase 3 Plan 2: Query Engine Foundation Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 149 - "Phase 3 Plan 3: Deterministic Lexical Query/Search Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 150 - "Warnings"
Cohesion: 0.14
Nodes (13): CR-01: `query`/`search`/`callers`/`callees` return an unbounded result set when `--limit`/`limit` is omitted (the default case), CR-02: MCP tools accept a client-supplied `path` argument with no confinement to the server's configured repo root, Critical Issues, IN-01: `DeleteFileSubgraph`'s name does not match its documented behavior, Info, Phase 3: Code Review Report, Summary, Warnings (+5 more)

### Community 151 - "Phase 4 Plan 2: Additive Sync Schema Fields + Exported Reverse-Adjacency Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 152 - "Phase 4 Plan 9: Goroutine-Leak-Free Watcher/Daemon Soak (goleak) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+5 more)

### Community 153 - "Phase 5: Language Coverage & Resolution Breadth - Research"
Cohesion: 0.14
Nodes (13): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 5: Language Coverage & Resolution Breadth - Research (+5 more)

### Community 154 - "Phase 6 Plan 02: JSON-Config Agent Targets (Claude, Cursor, Gemini, Kiro, Antigravity) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 155 - "Phase 6 Plan 03: Non-JSON Agent Targets (Codex TOML, opencode JSONC, Hermes YAML) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 156 - "Phase 6 Plan 5: CLI Lifecycle — version, --version, telemetry Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 157 - "Phase 7 Plan 1: Migration Tool Wave-0 Infrastructure Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 158 - "Phase 7 Plan 2: Migration Progress Record Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 159 - "Phase 7 Plan 4: Progress Cursor + Atomic Directory Swap Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 160 - "Phase 7 Plan 5: Migration Validation (D-09 Invariant Pass) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 161 - "Phase 7 Plan 7: CLI Wiring (codegraph migrate) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 162 - "recordFile"
Cohesion: 0.27
Nodes (6): cursorTarget, cursorConfigPath(), cursorLegacyRulesPath(), Location, init(), recordFile()

### Community 163 - "registerLanguage"
Cohesion: 0.11
Nodes (17): cProjectDescriptor, kotlinProjectDescriptor, rubyProjectDescriptor, swiftProjectDescriptor, init(), init(), readCProjectDescriptor(), init() (+9 more)

### Community 164 - "verifyRelease"
Cohesion: 0.08
Nodes (44): Client, newLatestRedirectClient(), resolveLatestVersion(), resolveLatestVersionViaAPI(), T, TestResolveLatestVersion_APIFallbackOnBlankLocation(), TestResolveLatestVersion_RedirectTrick(), TestResolveLatestVersionViaAPI_EmptyTagIsError() (+36 more)

### Community 165 - "verify_test.go"
Cohesion: 0.04
Nodes (48): Alternatives Considered, Anti-Patterns to Avoid, Applicable ASVS Categories, Architectural Responsibility Map, Architecture Patterns, Assumptions Log, Claude's Discretion, Code Examples (+40 more)

### Community 166 - "Parser Strategy Decision (D-05 / D-05a)"
Cohesion: 0.17
Nodes (12): Crash Isolation (RESEARCH Pitfall 5), D-05b Note, Decision Rationale (against D-05a's criterion), Disposition of Spike Artifacts, DIST-05 Consequence, Full-parse throughput (entire corpus, one pass per iteration), Incremental reparse (single-keystroke append-at-EOF edit on the corpus's largest file), Measured Numbers (+4 more)

### Community 167 - "Phase 3 Plan 07: Stdio MCP Server + Tool Gating Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+4 more)

### Community 168 - "Phase 4: Incremental Sync & File Watcher - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Code Examples, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit (+4 more)

### Community 169 - "Phase 6 Plan 01: internal/agents Foundation Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+4 more)

### Community 170 - "Phase 6: Agent Integrations & CLI Lifecycle - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Corrected Per-Agent Parity Table (supersedes CONTEXT.md D-05/D-05a/D-06), Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit (+4 more)

### Community 171 - "Threat Verification"
Cohesion: 0.15
Nodes (12): 06-01 — Shared surgical-write foundation (`internal/agents/shared.go`), 06-02 — Per-agent JSON/Markdown editors, 06-03 — TOML / JSONC / YAML editors, 06-04 — Install/uninstall CLI (`internal/cli/install.go`, `uninstall.go`), 06-05 — Version / telemetry info commands, 06-06 — Signature-verified self-update (`internal/upgrade`), Accepted Risks Log, Prior-Review Fixes Confirmed Present (+4 more)

### Community 172 - "Goal Achievement"
Cohesion: 0.15
Nodes (12): 1. Live coding-agent MCP handshake, Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (Roadmap Success Criteria) (+4 more)

### Community 173 - "Phase 7: Code Review Report (Re-Review)"
Cohesion: 0.15
Nodes (12): CR-01: In-place migration holds the source DB open across the directory swap — deterministically breaks the default command on Windows, Critical Issues, IN-01: `recoverInterruptedSwap` leaks the opened partial store on `finishFromComplete`'s error paths, IN-02: `targetPopulated` treats an existing-but-unreadable target as empty, letting recovery bypass the D-08 overwrite guard, IN-03: WR-03 residual — the health probe still opens an existing store read-write during the "no changes made" refusal check, IN-04: Carry-forward — original IN-01..IN-05 remain open, Info, Phase 7: Code Review Report (Re-Review) (+4 more)

### Community 174 - "Phase 08 Plan 05: Upgrade E2E Verification Loop-Closer Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 08 Plan 05: Upgrade E2E Verification Loop-Closer Summary (+4 more)

### Community 175 - "CodeGraph Go"
Cohesion: 0.15
Nodes (12): Active, CodeGraph Go, Constraints, Context, Core Value, Current State, Evolution, Key Decisions (+4 more)

### Community 176 - "v1.0 Research Summary: Drop-in Parity & Human UX"
Cohesion: 0.15
Nodes (12): Confidence Assessment, Executive Summary, From ARCHITECTURE.md (Integration Approach), From FEATURES.md (Behavioral Parity), From PITFALLS.md (Load-Bearing Risks), From STACK.md (Technology Stack), Gaps Requiring Later Attention, Implications for Roadmap (+4 more)

### Community 177 - ".Install"
Cohesion: 0.29
Nodes (5): kiroTarget, Location, init(), kiroConfigPath(), kiroLegacySteeringPath()

### Community 178 - "Run"
Cohesion: 0.27
Nodes (10): Options, resolveFunc, Stats, Duration, readGraphCounts(), Run(), T, TestPipelineRun() (+2 more)

### Community 179 - "TestPruneFixtures"
Cohesion: 0.04
Nodes (48): Alternatives Considered, Anti-Patterns to Avoid, Applicable ASVS Categories, Architectural Responsibility Map, Architecture Patterns, Assumptions Log, Claude's Discretion, Code Examples (+40 more)

### Community 180 - "filesStatusFixture"
Cohesion: 0.33
Nodes (11): filesStatusFixture(), Engine, T, rawFilePaths(), TestDbSizeBytes(), TestFiles(), TestStatus(), T (+3 more)

### Community 181 - "resolveLatestVersion"
Cohesion: 0.07
Nodes (38): syncBuffer, buildWorktreeFixture(), containsBareNoticeGlyph(), copyFixture(), M, T, noticeGlyph(), runBinary() (+30 more)

### Community 182 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Research"
Cohesion: 0.17
Nodes (11): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 1: Foundation — Storage, Schema & Parser Strategy - Research (+3 more)

### Community 183 - "Phase 01: Code Review Report"
Cohesion: 0.17
Nodes (11): CR-05: `pebbleStore.Close()` is not idempotent — a second call panics the process, Critical Issues, IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match, IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`, IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns, Info, Phase 01: Code Review Report, Summary (+3 more)

### Community 184 - "Phase 2: Go Indexing Pipeline - Research"
Cohesion: 0.17
Nodes (11): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 2: Go Indexing Pipeline - Research (+3 more)

### Community 185 - "Info"
Cohesion: 0.17
Nodes (11): IN-01: Multi-name const/var specs stamp identical positions on every declared name, IN-02: Embedded struct field carrying a struct tag is missed, IN-03: `collapseEdges` representative tiebreak includes a `filePath` term that can never break a tie, IN-04: `readGraphCounts` failure drops the partial Stats other error paths preserve, IN-05: Dot-import (`import . "pkg"`) calls are not resolved to the dot-imported package, Info, Phase 2: Code Review Report, Summary (+3 more)

### Community 186 - "Goal Achievement"
Cohesion: 0.17
Nodes (11): Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (must_haves, merged across all 6 plans), Phase 2: Go Indexing Pipeline Verification Report (+3 more)

### Community 187 - "Phase 3 Plan 05: Files & Status Query Commands Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 3 Plan 05: Files & Status Query Commands Summary (+3 more)

### Community 188 - "Phase 3: Query Engine & MCP Server - Research"
Cohesion: 0.17
Nodes (11): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 3: Query Engine & MCP Server - Research (+3 more)

### Community 189 - "Fixed Issues"
Cohesion: 0.17
Nodes (11): CR-01: `query`/`search`/`callers`/`callees` return an unbounded result set when `--limit`/`limit` is omitted, CR-02: MCP tools accept a client-supplied `path` argument with no confinement to the server's repo root, Fixed Issues, Phase 3: Code Review Fix Report, Skipped Issues, Verification, WR-01: Zero-match results marshal `null` instead of `[]` for several JSON array fields, WR-02: Inconsistent negative-value handling across `--depth`/`--max-files`/`--limit` (+3 more)

### Community 190 - "Goal Achievement"
Cohesion: 0.17
Nodes (11): Anti-Patterns Found, Behavioral Spot-Checks / Targeted Tests, Deep Code Review Follow-Up (04-REVIEW.md), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (+3 more)

### Community 191 - "Goal Achievement"
Cohesion: 0.17
Nodes (11): Anti-Patterns Found, Behavioral Spot-Checks / Targeted Test Runs, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (Roadmap Success Criteria), Phase 7: Migration Tool Verification Report (+3 more)

### Community 192 - "Project State"
Cohesion: 0.15
Nodes (12): Accumulated Context, Blockers/Concerns, Current Position, Decisions, Deferred Items, Operator Next Steps, Pending Todos, Performance Metrics (+4 more)

### Community 193 - "resolution_test.go"
Cohesion: 0.53
Nodes (10): csharpFixtureFile, nodeKey, buildCSharpGraph(), T, hasEdge(), TestResolve_FullyQualifiedCrossNamespaceCall(), TestResolve_FullyQualifiedCrossNamespaceInheritance(), TestResolve_PartialClassBothFragmentsCallable() (+2 more)

### Community 194 - "init"
Cohesion: 0.27
Nodes (8): rustProjectDescriptor, init(), readRustDescriptor(), rustModulePath(), Base, Derived, Reader, ReadWriter

### Community 195 - "resolution_test.go"
Cohesion: 0.53
Nodes (10): buildJavaGraph(), T, hasEdge(), TestResolve_ImportedCrossPackageCall(), TestResolve_ImportedCrossPackageInheritance(), TestResolve_InheritedMethodCallResolvesViaConformanceRetry(), TestResolve_SamePackageCrossFileCall(), writeJavaFixture() (+2 more)

### Community 196 - "resolution_test.go"
Cohesion: 0.53
Nodes (10): buildTSGraph(), T, hasEdge(), TestResolve_CrossFileInheritance(), TestResolve_NamespaceImportCrossFileCall(), TestResolve_RelativeSpecifierCrossFileCall(), TestResolve_TSConfigPathsAliasedCrossFileCall(), writeTSFixture() (+2 more)

### Community 197 - "loadProgress"
Cohesion: 0.36
Nodes (9): Reader, loadProgress(), saveProgress(), T, TestLoadProgress_AbsentReportsCleanly(), TestLoadProgress_CorruptFailsLoud(), TestSaveProgressLoadProgress_RoundTrip(), TestSaveProgressLoadProgress_StatusComplete() (+1 more)

### Community 198 - "NewMeta"
Cohesion: 0.36
Nodes (6): IsCurrentSchemaVersion(), NewMeta(), T, TestMetaRoundTripsSchemaVersion(), TestNodeRoundTripsEqual(), TestSchemaRoundTripsUnknownFields()

### Community 199 - "Graph Report - codegraph-go  (2026-07-10)"
Cohesion: 0.18
Nodes (10): Community Hubs (Navigation), Corpus Check, God Nodes (most connected - your core abstractions), Graph Freshness, Graph Report - codegraph-go  (2026-07-10), Import Cycles, Knowledge Gaps, Suggested Questions (+2 more)

### Community 200 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths, Phase 1: Foundation — Storage, Schema & Parser Strategy Verification Report (+2 more)

### Community 201 - "Phase 3: Query Engine & MCP Server - Discussion Log"
Cohesion: 0.18
Nodes (10): affected / files behavior, Claude's Discretion, Deferred Ideas, explore / node markdown templates, MCP server, Parity Model, Phase 3: Query Engine & MCP Server - Discussion Log, Read Path — node enumeration (+2 more)

### Community 202 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths, Phase 3: Query Engine & MCP Server Verification Report (+2 more)

### Community 203 - "Phase 4: Incremental Sync & File Watcher - Discussion Log"
Cohesion: 0.18
Nodes (10): Claude's Discretion, Daemon, Lock & unlock (SYNC-04, SYNC-05), Deferred Ideas, Incremental Sync Engine (INDX-03), Leak-Free Verification (SYNC-06), MCP Reconnect Reconciliation (SYNC-03), Native Watcher & Debounce (SYNC-01, SYNC-02), Phase 4: Incremental Sync & File Watcher - Discussion Log (+2 more)

### Community 204 - "Architecture Patterns"
Cohesion: 0.18
Nodes (11): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Store-seeded incremental Sync() (INDX-03, the core algorithm), Pattern 2: File-owned secondary index (D-02, the pruning keyspace), Pattern 3: fsnotify recursive watch + debounce (SYNC-01), Pattern 4: Staleness banner threading (SYNC-02, D-04a), Pattern 5: MCP reconnect reconciliation (SYNC-03, D-06), Pattern 6: Daemon + lockfile + unlock (SYNC-04, SYNC-05, D-05) (+3 more)

### Community 205 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths, Phase 5: Language Coverage & Resolution Breadth Verification Report (+2 more)

### Community 206 - "Phase 6: Agent Integrations & CLI Lifecycle - Discussion Log"
Cohesion: 0.18
Nodes (10): Agent detection & target selection, Claude's Discretion, CLI ergonomics: `version` / `help` (CLI-01), Deferred Ideas, Marker fences, idempotency, user-edit preservation (AGNT-01/02), Non-TS roster agents: Hermes, Antigravity, Kiro (AGNT-01), Per-agent config matrix & quirks (AGNT-01/03), Phase 6: Agent Integrations & CLI Lifecycle - Discussion Log (+2 more)

### Community 207 - "Info"
Cohesion: 0.18
Nodes (10): IN-01: `ci.yml`'s perf-regression job comment overstates what's actually gated, IN-02: `tools/bench/runner`'s `-ts-binary` default is a macOS-Homebrew-only path, IN-03: `gencorpus`'s fixed query term assumes `goCount >= 1`, which fails for very small `-count` values, Info, Phase 8: Code Review Report, Summary, Warnings, WR-01: `.goreleaser.yaml`'s `archives:`/`checksum:` blocks are dead configuration that contradicts its own header comment (+2 more)

### Community 208 - "Verifying a codegraph release"
Cohesion: 0.22
Nodes (9): 1. Verifying a release, 2. Dependency tree (DIST-05), 3. Reproducibility posture, a) Verify the cosign keyless signature (per-binary), b) Verify SLSA provenance, c) Inspect the SBOM, `codegraph upgrade` as the automated consumer, Reproducing a build locally (+1 more)

### Community 209 - "Info"
Cohesion: 0.10
Nodes (15): main(), T, TestFlagParityDocCoversRegisteredFlags(), Execute(), Command, newRootCmd(), Command, newTelemetryCmd() (+7 more)

### Community 210 - "resolution_test.go"
Cohesion: 0.49
Nodes (9): decodeExportFrames(), encodeExportFrames(), T, indexAndExport(), TestSelfConsistency_DeterministicRebuild(), TestSelfConsistency_ExpectedStructure(), writePHPFixture(), exportFrame (+1 more)

### Community 211 - "resolution_test.go"
Cohesion: 0.53
Nodes (9): buildPyGraph(), T, hasEdge(), TestResolve_AliasedImportCall(), TestResolve_CrossModuleImportedCall(), TestResolve_CrossModuleInheritance(), writePyFixture(), nodeKey (+1 more)

### Community 212 - "Registered"
Cohesion: 0.04
Nodes (46): 10. 09-02 D4, 11. 09-02 D5, 12. 09-02 D6, 13. 09-03 D1, 14. 09-03 D2, 15. 09-03 D3, 16. 09-03 D4, 17. 09-04 D1 (+38 more)

### Community 213 - "walkSpringRoutes"
Cohesion: 0.16
Nodes (42): WriteFile(), markerBlock(), Remove(), stripMarkerBlock(), T, initRepo(), malformedHookFixture(), runGit() (+34 more)

### Community 214 - "countingWriter"
Cohesion: 0.31
Nodes (4): T, TestBatchWriter_ClosesTrailingWriter(), countingStore, countingWriter

### Community 216 - "NewGoParser"
Cohesion: 0.17
Nodes (15): goProjectDescriptor, pendingFile, importPathFor(), readModulePath(), ShouldSkipDir(), init(), NewGoParser(), T (+7 more)

### Community 218 - "atomicSwap"
Cohesion: 0.15
Nodes (12): main(), Alpha(), helper(), Run(), atomicSwap(), checkWritable(), swapWindows(), T (+4 more)

### Community 219 - "Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero)"
Cohesion: 0.20
Nodes (9): Decision (human-ratified 2026-07-10), Decisions & deviations, Dependency graph, Measured results, Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero), Requirements, Self-Check: PASSED, Tech tracking (+1 more)

### Community 220 - "Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log"
Cohesion: 0.20
Nodes (9): Claude's Discretion, Deferred Ideas, Golden-Corpus & TS DDL Capture, GraphStore Interface & Concurrency, Keyspace Layout, Parser Spike Methodology, Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log, Record Encoding & Schema Evolution (+1 more)

### Community 221 - "Phase 2: Go Indexing Pipeline - Discussion Log"
Cohesion: 0.20
Nodes (9): Claude's Discretion, CLI Lifecycle & Commands (INDX-01, INDX-02), Deferred Ideas, Edge Multiplicity (Phase-1-deferred key-identity question), Go Extraction Scope (LANG-01), Node Identity, Phase 2: Go Indexing Pipeline - Discussion Log, Schema Field Parity (+1 more)

### Community 222 - "Phase 3 — Validation Strategy"
Cohesion: 0.20
Nodes (9): Manual-Only Verifications, Per-Task Verification Map, Phase 3 — Validation Strategy, Sampling Rate, Security Domain (ASVS L1), Test Infrastructure, Validation Audit 2026-07-11, Validation Sign-Off (+1 more)

### Community 223 - "Phase 5: Language Coverage & Resolution Breadth - Discussion Log"
Cohesion: 0.20
Nodes (9): Claude's Discretion, Coverage Policy (LANG-06), Cross-File Resolution Fidelity, Deferred Ideas, Dispatch Synthesis & Provenance (RES-02 / RES-03), Extractor & Discovery Architecture, Framework-Aware Routing (LANG-07), Phase 5: Language Coverage & Resolution Breadth - Discussion Log (+1 more)

### Community 224 - "Phase 8: Release Hardening & Benchmarks - Discussion Log"
Cohesion: 0.20
Nodes (9): Benchmark Corpus Split (PERF-01 vs PERF-02/INDX-06), Build → Sign → Provenance → SBOM Wiring (DIST-02, DIST-03), CGo Cross-Compilation Toolchain (DIST-01), Claude's Discretion, Deferred Ideas, Perf Metric Methodology + Regression-Gate Policy (PERF-01/02, INDX-06), Phase 8: Release Hardening & Benchmarks - Discussion Log, Repository Layout + Workflow Structure (+1 more)

### Community 225 - "Fixed Issues"
Cohesion: 0.20
Nodes (9): Fixed Issues, IN-01: `ci.yml`'s perf-regression job comment overstates what's actually gated, IN-02: `tools/bench/runner`'s `-ts-binary` default is a macOS-Homebrew-only path, IN-03: `gencorpus`'s fixed query term assumes `goCount >= 1`, which fails for very small `-count` values, Phase 8: Code Review Fix Report, Skipped Issues, WR-01: `.goreleaser.yaml`'s `archives:`/`checksum:` blocks are dead configuration that contradicts its own header comment, WR-02: `realcorpus.Entry.Resolve()` never verifies an existing local checkout is pinned at `CommitSHA`, and no caller checks either (+1 more)

### Community 226 - "Milestone: v0.1 — Initial Release"
Cohesion: 0.12
Nodes (16): CodeGraph Go — Retrospective, Cost Observations, Cost Observations, Cross-Milestone Trends, Key Lessons, Key Lessons, Milestone: v0.1 — Initial Release, Milestone: v1.0 — Drop-in Parity & Human UX (+8 more)

### Community 227 - "Router"
Cohesion: 0.36
Nodes (3): Router, main(), setupRouter()

### Community 228 - "Router"
Cohesion: 0.36
Nodes (3): Router, main(), setupRouter()

### Community 229 - "discover.go"
Cohesion: 0.13
Nodes (41): applyCoreDirectoryBoost(), applyMultiTermReRank(), applyPostMergeRerankers(), applyTestFileDampening(), distinctiveExactMatchExemptIDs(), fileDir(), gatherChannel1(), gatherChannel2() (+33 more)

### Community 230 - "languages_typescript.go"
Cohesion: 0.05
Nodes (42): Alternatives Considered, Anti-Patterns to Avoid, Applicable ASVS Categories, Architectural Responsibility Map, Architecture Patterns, Assumptions Log, Code Examples, Common Pitfalls (+34 more)

### Community 231 - "gin_fixture.go"
Cohesion: 0.36
Nodes (3): main(), setupRouter(), Router

### Community 232 - "pkga.go"
Cohesion: 0.13
Nodes (34): bodySubstance(), corroboratedDefs(), Reader, largeOverloadSeed(), pascalCaseTypeTokens(), resolveDefsByName(), seedNamedSymbols(), seedQueryTokens() (+26 more)

### Community 233 - "nodeExploreFixture"
Cohesion: 0.11
Nodes (24): AgentTarget, Cmd, Command, Item, Location, Model, Msg, View (+16 more)

### Community 234 - "upgrade.go"
Cohesion: 0.23
Nodes (30): stubOversizedParser, Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser() (+22 more)

### Community 235 - "Architecture Patterns"
Cohesion: 0.22
Nodes (9): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Typed, range-scannable keyspace (D-03), Pattern 2: Snapshot reads + IndexedBatch writes behind `GraphStore` (D-04), Pattern 3: Additive-only protobuf schema with a versioned `meta` record (D-02/D-02a), Pattern 4: Narrow, backend-swappable `Parser` interface (D-05b), Pattern 5: Import-graph bypass test (D-04a), Recommended Project Structure (+1 more)

### Community 236 - "Info"
Cohesion: 0.22
Nodes (8): IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match, IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`, IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns, Info, Phase 01: Code Review Report, Summary, Warnings, WR-06: Residual `Commit()`-vs-`Close()` race is real but explicitly accepted/documented — track, don't block

### Community 237 - "Architecture Patterns"
Cohesion: 0.22
Nodes (9): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: File Discovery via `go/build.Context.MatchFile` (not `go/packages`), Pattern 2: Bounded Worker Pool for Pass 1 (one Parser per worker), Pattern 3: Node ID Construction (D-02/D-02a), Pattern 4: Qualified Name Construction (Go-specific), Pattern 5: Cross-Package Import Resolution, Recommended Project Structure (+1 more)

### Community 238 - "Phase 7: Migration Tool - Discussion Log"
Cohesion: 0.22
Nodes (8): Claude's Discretion, Command Surface & Source Detection (MIGR-01), Deferred Ideas, Field Mapping — carry vs. drop, Node/Edge Identity Translation, Phase 7: Migration Tool - Discussion Log, Resumability & Version-Stamping (MIGR-02), Validation & Failure Policy (MIGR-02)

### Community 239 - "Common Pitfalls"
Cohesion: 0.22
Nodes (9): Common Pitfalls, Pitfall 1: Cross-device `os.Rename` (EXDEV), Pitfall 2: Silent `Rows.Err()` after read loops, Pitfall 3: FTS5 shadow tables read as data, Pitfall 4: `file:`-source edges look dangling, Pitfall 5: NULL scan panics on aged nullable columns, Pitfall 6: `file_path` separator / relative-vs-absolute mismatch, Pitfall 7: `edge_count` has no TS source column (+1 more)

### Community 240 - "Phase 8: Release Hardening & Benchmarks Verification Report"
Cohesion: 0.22
Nodes (8): Anti-Pattern / Debt-Marker Scan, Gaps Summary, Human Verification / Action Items Before Calling v1.0 Fully Shipped, Method, Per-Requirement Verdicts, Phase 8: Release Hardening & Benchmarks Verification Report, Requirements Coverage vs REQUIREMENTS.md, Roadmap Success Criteria (the contract)

### Community 241 - "languages_csharp.go"
Cohesion: 0.36
Nodes (6): csharpProjectDescriptor, csprojProject, csprojPropertyGroup, Name, init(), readCSharpDescriptor()

### Community 242 - "UserController"
Cohesion: 0.29
Nodes (5): GetMapping, RequestMapping, RestController, User, UserController

### Community 243 - "UserController"
Cohesion: 0.29
Nodes (5): GetMapping, RequestMapping, RestController, User, UserController

### Community 244 - "TestBuildTSIndex_Variants"
Cohesion: 0.50
Nodes (7): assertCountPositive(), DB, T, TB, openReadOnly(), presentColumns(), TestBuildTSIndex_Variants()

### Community 245 - "Phase 1 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 1 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 246 - "Phase 2 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 2 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 247 - "Architecture Patterns"
Cohesion: 0.25
Nodes (8): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Additive Reader extension via range-scan (D-03), Pattern 2: Query-time reverse adjacency (D-04), Pattern 3: MCP stdio server with startup-time conditional tool registration (D-08a), Pattern 4: Nearest `.codegraph/` resolution (D-01a), Recommended Project Structure, System Architecture Diagram

### Community 248 - "Phase 4 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Requirement Verification Map, Phase 4 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 249 - "Architecture Patterns"
Cohesion: 0.25
Nodes (8): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Generic tree-walker + per-language declarative config (implementation lever, not required), Pattern 2: Extends→implements promotion at resolve time (Java/C#/TS declared implements), Pattern 3: Structural (implicit) interface satisfaction for Go — extract interface method specs as nodes, match method sets at resolve time, Pattern 4: Route detection as regex-over-comment-stripped-source, not full AST query, Recommended Project Structure, System Architecture Diagram

### Community 250 - "Phase 5 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 5 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 251 - "Common Pitfalls"
Cohesion: 0.25
Nodes (8): Common Pitfalls, Pitfall 1: Treating "5 TS-covered agents" as the parity boundary, Pitfall 2: Writing an instructions file for Cursor, Antigravity, Hermes, or Kiro, Pitfall 3: Claude Code local scope writing to the wrong file, Pitfall 4: opencode config-dir resolution special-casing Windows, Pitfall 5: Hermes YAML surgery assuming a fixed list-item indent, Pitfall 6: Antigravity's `type` field, Pitfall 7: `upgrade` swap without verifying first, or verifying-then-still-swapping-on-failure

### Community 252 - "Phase 6 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 6 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 253 - "Phase 7 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 7 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 254 - "Phase 8 — Validation Strategy"
Cohesion: 0.25
Nodes (7): Manual-Only Verifications, Per-Task Verification Map, Phase 8 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Sign-Off, Wave 0 Requirements

### Community 255 - "fileindex_test.go"
Cohesion: 0.25
Nodes (31): New(), assertNoRegistryRecord(), T, initFixture(), nodeNames(), TestDaemonCleanShutdown(), TestDaemonFlushLockRequeueGivesUpPerEpisode(), TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock() (+23 more)

### Community 256 - "pebbleStore"
Cohesion: 0.10
Nodes (23): Command, Options, newServeCmd(), serveServerPaths(), serveWatchStart(), T, TestServeKeepsStartPathDistinctFromConfinementRoot(), TestServeServerPathsNoIndex() (+15 more)

### Community 257 - "languages_php.go"
Cohesion: 0.43
Nodes (5): composerJSON, phpProjectDescriptor, init(), phpNamespaceFor(), readPHPDescriptor()

### Community 258 - "init"
Cohesion: 0.38
Nodes (5): javaProjectDescriptor, mavenPOM, Name, init(), readJavaDescriptor()

### Community 259 - "nodeid_test.go"
Cohesion: 0.48
Nodes (6): T, TestNodeID_Deterministic(), TestNodeID_Distinct(), TestNodeID_EmptyArgs(), TestNodeID_InjectionSafe(), TestNodeID_Shape()

### Community 261 - "UserController"
Cohesion: 0.29
Nodes (4): Controller, Get, Post, UserController

### Community 262 - "UserController"
Cohesion: 0.29
Nodes (4): Controller, Get, Post, UserController

### Community 263 - "Run"
Cohesion: 0.47
Nodes (9): Run(), T, TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading(), TestUpgradeRun_ForceReinstallsSameVersion(), TestUpgradeRun_ForceStillVerifiesBeforeSwap(), TestUpgradeRun_RefusesNonWritableTargetBeforeDownloading(), TestUpgradeRun_SameVersionNoOpWithoutForce(), TestUpgradeRun_TamperedDownloadNeverSwaps() (+1 more)

### Community 264 - "Phase 01: Code Review Fix Report"
Cohesion: 0.29
Nodes (6): CR-05: `pebbleStore.Close()` (and `pebbleReader.Close()`) not idempotent, Fixed Issues, Phase 01: Code Review Fix Report, Skipped Issues, Verification, WR-05: `parser.Tree.Close()` not hardened against double-close

### Community 265 - "Code Examples"
Cohesion: 0.29
Nodes (7): Code Examples, Golden `callers.json` / `callees.json` / `impact.json` — locations-only shape, Golden `explore.json` — the literal markdown contract (D-05a), Golden `node.json` — the literal markdown contract (D-05b), Golden `query.json` — full-record JSON wrapper shape, Golden `status.json` — field mapping for D-05's Go/Pebble-truthful values, mcp-go: minimal stdio server with a typed tool

### Community 266 - "Common Pitfalls"
Cohesion: 0.29
Nodes (7): Common Pitfalls, Pitfall 1: `newSymbolIndex(results)` cannot be reused unmodified for `Sync()`, Pitfall 2: Dependent files need Pass-1 re-extraction, not just Pass-2 re-resolution, Pitfall 3: `File` has no mtime/size fields — the stat pre-filter is unimplementable as-is, Pitfall 4: `buildReverseAdjacency` is unexported — a straight `internal/indexer` import won't compile, Pitfall 5: Scoping `collapseEdges` to just the reparse batch is safe — but only because node ids are globally unique per declaring file, Pitfall 6: `w.Errors` channel on `fsnotify.Watcher` must be drained, or the watcher deadlocks

### Community 267 - "Architecture Patterns"
Cohesion: 0.29
Nodes (7): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: AgentTarget interface (port of TS's `types.ts`), Pattern 2: idempotent JSON entry write (port of `writeMcpEntry` pattern used by 6 of 8 targets), Pattern 3: marker-fenced instruction block upsert (port of `shared.ts` `replaceOrAppendMarkedSection`), Recommended Project Structure, System Architecture Diagram

### Community 268 - "SEED-001: local Svelte + shadcn-svelte UI for browsing/viewing/querying the graph"
Cohesion: 0.29
Nodes (6): Breadcrumbs, Notes, Scope Estimate, SEED-001: local Svelte + shadcn-svelte UI for browsing/viewing/querying the graph, When to Surface, Why This Matters

### Community 269 - "Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06)"
Cohesion: 0.22
Nodes (8): Behavioral fixtures (Phase 1, D-01/D-03/D-06 extension), Capture provenance, Corpus (D-06a, extended by D-03), Golden Fixtures — TS CodeGraph v1.3.1 Ground Truth (D-06), Historical bug note (Pitfall 2) — edge-dedup, issue #1034, Re-running the capture, Running this suite (GOLDEN-01), Volatile fields (Pitfall 1) — stripped for byte-for-byte reproducibility

### Community 270 - "init"
Cohesion: 0.53
Nodes (4): pythonProjectDescriptor, dottedModulePath(), init(), readPythonDescriptor()

### Community 271 - "init"
Cohesion: 0.15
Nodes (29): aggregateFileGraphScore(), applyBuriedRescue(), applyHardTestExclusion(), classifyNodeTier(), computeConnectedToEntry(), computeFileScoreTiers(), Reader, isGenuinelyBuried() (+21 more)

### Community 272 - "init"
Cohesion: 0.20
Nodes (28): lockInfo, acquire(), createLockExclusive(), Time, isProcessLive(), isStale(), lockPath(), readLock() (+20 more)

### Community 273 - "Common Pitfalls"
Cohesion: 0.33
Nodes (6): Common Pitfalls, Pitfall 1: Non-deterministic golden fixtures (verified directly against the live TS index), Pitfall 2: Edge duplication without deterministic identity (a real, historical bug in the TS original), Pitfall 3: Conflating "single static binary for users" with "no CGo toolchain needed anywhere", Pitfall 4: Pebble v1/v2 import-path confusion, Pitfall 5: Treating the D-05 "crash isolation" dimension as validated by valid-input testing alone

### Community 274 - "Code Examples"
Cohesion: 0.33
Nodes (6): Bounded Worker Pool (already shown in Pattern 2, repeated here as the canonical reference for the planner), Code Examples, Import Resolution, Interface Embedding Detection (tree-sitter-go grammar, verified via Context7 this session), Method Receiver Extraction (tree-sitter-go grammar, verified via Context7 this session), Struct Embedding Detection (tree-sitter-go grammar, verified via Context7 this session)

### Community 275 - "Common Pitfalls"
Cohesion: 0.33
Nodes (6): Common Pitfalls, Pitfall 1: Edge-Dedup "Representative" Call Site Is Order-Dependent By Default, Pitfall 2: Pebble's On-Disk Bytes Are Not the Right Determinism Target, Pitfall 3: CGo Parser/Tree Lifecycle Leaks Under Concurrency, Pitfall 4: The 4 MiB `MaxSourceBytes` Ceiling Applies Per-File, Silently, Pitfall 5: `go/build.Context.MatchFile` Needs `dir` to Be the File's Actual Parent Directory

### Community 276 - "Common Pitfalls"
Cohesion: 0.33
Nodes (6): Common Pitfalls, Pitfall 1: The extract.go worker pool's "one parser per worker" design does not survive multi-language files, Pitfall 2: `symbolindex.go`'s import-path-keyed global index does not generalize across languages, Pitfall 3: Package-qualified vs. bare-name call resolution ambiguity multiplies with declared-import languages, Pitfall 4: External-scanner grammars run in-process with no crash isolation — this now applies to 8 of 10 new languages, not just Python, Pitfall 5: Java/C# multi-file compilation units and partial classes break the "one node ID scheme" assumption subtly

### Community 277 - "06-UAT.md"
Cohesion: 0.33
Nodes (5): 1. Live coding-agent MCP handshake, Current Test, Gaps, Summary, Tests

### Community 278 - "Architecture Patterns"
Cohesion: 0.33
Nodes (6): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Read-only, defensive column enumeration (D-09.4 aged-DB tolerance), Pattern 2: Nullable-column scanning with `sql.NullString`, Recommended Project Structure, System Architecture Diagram

### Community 279 - "Field Mapping (D-02, D-05)"
Cohesion: 0.33
Nodes (6): Dropped tables (D-03), `edges` → `schema.Edge`, Field Mapping (D-02, D-05), `files` → `schema.File`, `nodes` → `schema.Node`, `project_metadata` → `schema.Meta` (D-03)

### Community 280 - "capture.sh"
Cohesion: 0.57
Nodes (6): capture_behavioral(), capture_repo(), capture.sh script, strip_json(), strip_sql_timestamps(), wrap_text()

### Community 281 - "package.json"
Cohesion: 0.40
Nodes (4): express, dependencies, express, name

### Community 282 - "init"
Cohesion: 0.29
Nodes (16): Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser(), TestExtract_Calls() (+8 more)

### Community 283 - "import_graph_test.go"
Cohesion: 0.60
Nodes (4): T, isAllowedImporter(), stripTestVariant(), TestNoPackageBypassesGraphStore()

### Community 284 - "package.json"
Cohesion: 0.40
Nodes (4): dependencies, @nestjs/common, name, @nestjs/common

### Community 285 - "modernc_confinement_test.go"
Cohesion: 0.60
Nodes (4): T, isAllowedImporter(), stripTestVariant(), TestModerncSQLiteConfinedToMigrate()

### Community 286 - "Code Examples"
Cohesion: 0.40
Nodes (5): Batched writes (plain Batch, not IndexedBatch, for the bulk write path), Code Examples, Incremental tree-sitter reparse, Pebble snapshot for lock-free reads, Range-delete a file's stale subgraph

### Community 287 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 288 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 289 - "Common Pitfalls"
Cohesion: 0.40
Nodes (5): Common Pitfalls, Pitfall 1: Treating `nodeKey`'s id encoding as kind-prefix-scannable, Pitfall 2: Building the reverse-adjacency map per-command-invocation but then calling multiple commands in one process (MCP server case), Pitfall 3: Markdown template drift from the golden fixture's exact bytes, Pitfall 4: Unbounded `impact --depth` / large monorepo BFS as a resource-exhaustion vector

### Community 290 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 291 - "Tech tracking"
Cohesion: 0.40
Nodes (4): Dependency graph, Notes, Self-Check: PASSED, Tech tracking

### Community 292 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 293 - "05-05-PLAN.md"
Cohesion: 0.40
Nodes (4): Artifacts this phase produces (this plan's contribution), Plan assumption (Pitfall 5 — surfaced, not silently resolved), STRIDE Threat Register, Trust Boundaries

### Community 294 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 295 - "Deferred Items — Phase 5"
Cohesion: 0.40
Nodes (4): Deferred Items — Phase 5, From 05-06 (Python extraction + resolution), From 05-07 (TypeScript/TSX/JavaScript extraction + resolution), From 05-12 (framework-aware routing, LANG-07)

### Community 296 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 297 - "Standard Stack"
Cohesion: 0.40
Nodes (5): Alternatives Considered, Core, Installation, Standard Stack, Supporting / stdlib

### Community 298 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 299 - "stubResolver"
Cohesion: 0.20
Nodes (28): formatMB(), formatNumber(), Builder, StatusResult, RenderStatusMarkdown(), RenderStatusText(), sortedCounts(), baseStatusResult() (+20 more)

### Community 300 - "processStartTime"
Cohesion: 0.67
Nodes (3): bootTimeUnix(), Time, processStartTime()

### Community 304 - "SetConfig"
Cohesion: 0.21
Nodes (25): stubOversizedParser, Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser() (+17 more)

### Community 305 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 306 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 307 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 308 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 309 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 310 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 311 - "03-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces, STRIDE Threat Register, Trust Boundaries

### Community 312 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core (already in `go.mod` — no new pinning needed except mcp-go), Standard Stack, Supporting

### Community 313 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 314 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 315 - "04-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 316 - "04-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 317 - "04-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 318 - "04-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 319 - "04-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 320 - "04-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 321 - "04-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 322 - "04-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 323 - "04-09-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 324 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 325 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 326 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 327 - "05-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 328 - "05-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 329 - "05-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 330 - "05-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 331 - "05-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 332 - "05-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 333 - "05-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 334 - "05-09-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 335 - "05-10-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 336 - "05-11-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 337 - "05-12-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 338 - "05-13-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 339 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core (parser layer — extends the existing `internal/parser/cgo` pattern), Standard Stack, Supporting (mainstream-6, LANG-06)

### Community 340 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 341 - "Code Examples"
Cohesion: 0.50
Nodes (4): Code Examples, Language registry shape (D-01, extends the existing per-constructor parser pattern), Route node + heuristic dispatch edge (LANG-07), matching the parity target's vocabulary, Structural implements-edge synthesis (Go), bounded to avoid quadratic blowup (D-06)

### Community 342 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence — VERIFIED against authoritative source), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence — training knowledge, flagged in Assumptions Log)

### Community 343 - "06-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-01), STRIDE Threat Register, Trust Boundaries

### Community 344 - "06-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-02), STRIDE Threat Register, Trust Boundaries

### Community 345 - "06-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-03), STRIDE Threat Register, Trust Boundaries

### Community 346 - "06-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-04), STRIDE Threat Register, Trust Boundaries

### Community 347 - "06-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-05), STRIDE Threat Register, Trust Boundaries

### Community 348 - "06-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (06-06), STRIDE Threat Register, Trust Boundaries

### Community 349 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 350 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 351 - "Code Examples"
Cohesion: 0.50
Nodes (4): Code Examples, GitHub Releases "latest" resolution without hitting the rate-limited API, Marker-fenced section upsert (the shared primitive 4 of 8 targets use), sigstore-go verification wiring for `upgrade` (adapted for a GitHub-Actions-OIDC release identity)

### Community 352 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 353 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 354 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 355 - "08-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-01), STRIDE Threat Register, Trust Boundaries

### Community 356 - "08-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-02), STRIDE Threat Register, Trust Boundaries

### Community 357 - "08-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-03), STRIDE Threat Register, Trust Boundaries

### Community 358 - "08-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-04), STRIDE Threat Register, Trust Boundaries

### Community 359 - "08-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-05), STRIDE Threat Register, Trust Boundaries

### Community 360 - "08-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-06), STRIDE Threat Register, Trust Boundaries

### Community 361 - "08-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-07), STRIDE Threat Register, Trust Boundaries

### Community 362 - "08-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-08), STRIDE Threat Register, Trust Boundaries

### Community 363 - "08-09-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 08-09), STRIDE Threat Register, Trust Boundaries

### Community 364 - "searchFakeNodeIterator"
Cohesion: 0.16
Nodes (24): Deregister(), List(), recordPath(), Register(), T, TestRegistryListEmptyDir(), TestRegistryListMissingDir(), TestRegistryListPrunesStale() (+16 more)

### Community 365 - "stubDoer"
Cohesion: 0.18
Nodes (27): admitGlueCandidate(), buildExpandAdjacency(), buildTypeHierarchyIndex(), calleeCallEdges(), expandBFS(), expandGlueNodes(), expandHierarchyBudget(), expandTypeHierarchy() (+19 more)

### Community 366 - "resolveCSharpCorpus"
Cohesion: 0.83
Nodes (3): T, resolveCSharpCorpus(), TestGoldenParity_CSharp()

### Community 367 - "resolveJavaCorpus"
Cohesion: 0.83
Nodes (3): T, resolveJavaCorpus(), TestGoldenParity_Java()

### Community 368 - "resolvePythonCorpus"
Cohesion: 0.83
Nodes (3): T, resolvePythonCorpus(), TestGoldenParity_Python()

### Community 369 - "resolveTSJSCorpus"
Cohesion: 0.83
Nodes (3): T, resolveTSJSCorpus(), TestGoldenParity_TSJS()

### Community 370 - "`baseline.json` provenance and re-bless instructions"
Cohesion: 0.11
Nodes (18): 10-04-PLAN: the Namespace investigation, the tmpfs refutation, and the staleness finding, `baseline.json` provenance and re-bless instructions, Deferred: Namespace cache volumes, Historical: how the ORIGINAL darwin baseline was produced (superseded), How the CURRENT baseline was produced (2026-08-02, ubuntu-latest, disk), Measurement procedure, Re-blessing (intentional, human-invoked only), Round 1 — Namespace migration, an unexplained variance (+10 more)

### Community 378 - "Milestones"
Cohesion: 0.50
Nodes (3): Milestones, v0.1 Initial Release (Shipped: 2026-07-14), v1.0 Drop-in Parity & Human UX (Shipped: 2026-08-03)

### Community 386 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 393 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 402 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 403 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 404 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 405 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 413 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 414 - "2026-07-14-document-release-cut-procedures-runbook.md"
Cohesion: 0.07
Nodes (28): Behavioral drop-in gate (REL-04) — reuse, do not rebuild, Canonical References, Claude's Discretion, Cross-phase constraints carried forward (MUST honor), Deferred Ideas, Established Patterns, Existing Code Insights, Folded Todos (+20 more)

### Community 440 - "node.go"
Cohesion: 0.27
Nodes (5): absInt32(), Engine, isGeneratedFile(), narrowNodeMatches(), normalizeNarrowPath()

### Community 452 - "promptAgentMultiSelect"
Cohesion: 0.27
Nodes (9): Command, Location, installStatus(), newInstallCmd(), parseLocationFlag(), printAgentResults(), Command, newUninstallCmd() (+1 more)

### Community 453 - "stubWriter"
Cohesion: 0.07
Nodes (28): Canonical References, Claude's Discretion, Cross-phase constraints carried forward (MUST honor), Deferred Ideas, Docs this phase must update, Established Patterns, Existing Code Insights, External docs (fetch current versions during research — do not work from memory) (+20 more)

### Community 454 - "Discover"
Cohesion: 0.07
Nodes (28): Canonical References, Claude's Discretion, Commands being wrapped (the authoritative list), Cross-phase constraints carried forward (MUST honor), D-00 — ROADMAP criterion 2 is already satisfied; record, do not redo, Deferred Ideas, Docs this phase touches, Established Patterns (+20 more)

### Community 455 - "status.go"
Cohesion: 0.22
Nodes (11): dbSizeBytes(), Context, Mismatch, Engine, Time, MarshalStatusJSON(), newestSourceMtime(), shouldSkipStaleDir() (+3 more)

### Community 456 - "TestAspNet_DetectsHttpVerbAttributes"
Cohesion: 0.07
Nodes (27): `affected`, `callees`, `callers`, `daemon` / `daemons`, `explore`, `files`, Flag parity: TS CodeGraph 1.3.1 vs codegraph-go (SURF-05), `githooks` (Go-only, no TS command) (+19 more)

### Community 457 - "Extract"
Cohesion: 0.22
Nodes (24): T, TestPyInstantiates(), TestPyReferences(), TestPyReturns(), TestPyReturns_GenericAnnotation(), TestPyTypeOf(), TestPyUnannotatedAbsence(), Extract() (+16 more)

### Community 458 - "Phase 3 Plan 3: Watcher-on-MCP Default Flip Summary"
Cohesion: 0.07
Nodes (24): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+16 more)

### Community 459 - "copyFixture"
Cohesion: 0.15
Nodes (21): T, TestAffectedQuietSkipsControlCharacterPaths(), copyFixture(), execCmdWithInput(), T, readGraphCounts(), TestInitIndexUninit(), runGit() (+13 more)

### Community 460 - "companionHandler"
Cohesion: 0.23
Nodes (24): companionHandler(), pluralize(), Location, RenderCalleesMarkdown(), RenderCallersMarkdown(), RenderFilesMarkdown(), renderFileTreeMarkdown(), RenderImpactMarkdown() (+16 more)

### Community 461 - "explore_gate_test.go"
Cohesion: 0.21
Nodes (24): centralFileSelection(), fileRelevanceGate(), fiveTierFileSort(), containsPath(), T, TestCentralFileSelection_ExcludesFilesWithZeroTermHits(), TestCentralFileSelection_PicksTopTwoByMassWithTermHit(), TestCentralFileSelection_ReturnsFewerThanTwoWhenNotEnoughQualify() (+16 more)

### Community 462 - "release_workflow_shape_test.go"
Cohesion: 0.18
Nodes (24): T, runPRTitleLintStep(), TestPRTitleLintAcceptsAndRejects(), T, isMacOSClassRunner(), mustReleaseBuildMatrix(), mustReleaseProvenanceJob(), mustWorkflowOnKeys() (+16 more)

### Community 463 - "Implementation Decisions"
Cohesion: 0.08
Nodes (24): Behavioral fixture harness (extend, don't rebuild), Canonical References, Claude's Discretion, Current implementation (the swap/extension points), Deferred Ideas, Edge-Kind Coverage (user-decided post-research — SCOPE EXPANSION), Established Patterns, Existing Code Insights (+16 more)

### Community 464 - "Implementation Decisions"
Cohesion: 0.08
Nodes (24): Canonical References, Claude's Discretion, Current implementation (the extension points), Deferred Ideas, Established Patterns, Existing Code Insights, Golden / fixture harness (extend, don't rebuild), Implementation Decisions (+16 more)

### Community 465 - "Implementation Decisions"
Cohesion: 0.08
Nodes (24): Canonical References, Charm isolation & interactive seam (TUI-01, TUI-03, TUI-04), Claude's Discretion, Cross-phase decisions carried forward (MUST honor), Daemon command shape & lifecycle (DMON-01, DMON-02), Daemon lifecycle & lock (DMON-01..04), Deferred Ideas, Established Patterns (+16 more)

### Community 466 - "Info"
Cohesion: 0.16
Nodes (20): stdoutGuardCounts, CallExpr, T, schemaMetaGoPath(), TestStdoutGuardCatchesViolationsInTransitiveDependency(), closeOverServeReachableImports(), Package, T (+12 more)

### Community 467 - "OpenAt"
Cohesion: 0.18
Nodes (19): Closer, Context, Mismatch, Once, Engine, Reader, NewWithRoot(), OpenAt() (+11 more)

### Community 468 - "Pattern Assignments"
Cohesion: 0.08
Nodes (23): Deterministic lowest-`Id` tie-break (D-04), File Classification, Fresh-per-call, no package-level cache (RESEARCH Pitfall 2 / T-03-04-Stale), Golden fixture capture — `capture_repo` extension pattern, `internal/cli/explore.go` (modify), `internal/indexer/goextract/types.go` (extend) — model/config, CRUD, `internal/indexer/resolve.go` (extend) — service, batch/event-driven, `internal/query/explore_gate.go` (new) — service, transform (+15 more)

### Community 469 - "Pattern Assignments"
Cohesion: 0.08
Nodes (23): Best-effort degradation on missing context, Doc-comment decision table as the TS→Go remap convention, File Classification, `internal/cli/status.go` (modified) — sectioned layout (D-09), `internal/gitmeta/cache.go` (new) — `CachingDetector`, `internal/gitmeta/detect.go` (new) — `DetectIndexMismatch` 4-gate cascade, `internal/gitmeta` fixture builder (new `_test.go`, TEST-02 six layouts), `internal/gitmeta/worktree.go` (new) — `gitWorktreeRoot`/`gitCommonDir`/`realpath` (+15 more)

### Community 470 - "Goal Achievement"
Cohesion: 0.09
Nodes (22): Anti-Goal Checks (intentionally NOT gaps), Anti-Patterns Found, Behavioral Spot-Checks, Check 1: `codegraph status` — full sectioned content, Check 3: `codegraph status --json`, Check 4: CLI worktree detection (real `git worktree add`), Check 5: MCP worktree detection (real stdio JSON-RPC session), Check 6: SURF-06 markdown/JSON split (+14 more)

### Community 471 - "Implementation Decisions"
Cohesion: 0.09
Nodes (22): Archtest / ANSI-isolation guarantee (TUI-01), Archtest precedents to mirror (TUI-01), Canonical References, Charm v2 dependency scope, Claude's Discretion, Cross-phase decisions carried forward, Deferred Ideas, Established Patterns (+14 more)

### Community 472 - "execCmd"
Cohesion: 0.24
Nodes (19): execCmd(), T, setupIndexedFixture(), TestAffectedCmd(), TestCallersCalleesCmd(), TestExploreCmd(), TestFilesCmd(), TestImpactCmd() (+11 more)

### Community 473 - "initRepo"
Cohesion: 0.23
Nodes (19): T, initRepo(), newClaudeWorktreeFixture(), newLinkedWorktreeFixture(), newMonorepoSubdirFixture(), newNestedCloneFixture(), newSubmoduleFixture(), newSymlinkedFixture() (+11 more)

### Community 474 - "Implementation Decisions"
Cohesion: 0.09
Nodes (21): Canonical References, Claude's Discretion, Concurrent-session convergence (WATCH-04), Current implementation (the extension points), Deferred Ideas, Established Patterns, Existing Code Insights, Flag surface & precedence (WATCH-01) (+13 more)

### Community 475 - "Fixed Issues"
Cohesion: 0.09
Nodes (21): Fixed Issues, Fixed Issues, IN-01: Backstop Stop comment no longer overstates its coverage, IN-01: Pinned-pebble provenance claim in unix lock-classifier test comment, IN-02: serve's disabled branch gated on errors.As, symmetric with cli/daemon.go, IN-02: Time-based `TestOpenConvergesWhenHolderCloses` made event-synchronized, IN-03: Requeue give-up log off-by-one + counter never reset after exhaustion, IN-03: serve's FULL D-12 message byte sequence pinned, banner included (+13 more)

### Community 476 - "Implementation Decisions"
Cohesion: 0.09
Nodes (21): Canonical References, Claude's Discretion, Current implementation (the extension points), Deferred Ideas, Established Patterns, Existing Code Insights, Hook file semantics (HOOK-01/02), Implementation Decisions (+13 more)

### Community 477 - "Pattern Assignments"
Cohesion: 0.09
Nodes (21): Atomic file writes, Charm-free domain layer, ANSI-only-in-cli, cobra command-tree shape (`AddCommand` over a bare parent RunE), File Classification, Goroutine join discipline (`stop func()` blocks until done), `internal/cli/daemon.go` (MODIFY — controller, cobra command tree), `internal/cli/install.go` / `uninstall.go` (MODIFY — controller), `internal/cli/present/archtest/import_graph_test.go` (VERIFY — no edit expected) (+13 more)

### Community 478 - "Pattern Assignments"
Cohesion: 0.09
Nodes (21): cobra short-flag registration, Compact worktree notice + `--json` early-return ordering, `docs/FLAG-PARITY.md` (SURF-05, NEW), Docs prose/structure style, `docs/RELEASE-PROCEDURES.md` (REL-02 folded runbook, NEW), Error handling — dangling edge skip, not abort, File Classification, `internal/cli/affected.go` (SURF-04: `--stdin`/`--depth`/`--filter`/`--quiet`) (+13 more)

### Community 479 - "CachingDetector"
Cohesion: 0.15
Nodes (17): CallToolRequest, CachingDetector, Context, Mismatch, Mutex, companionTool(), confineToRepoRoot(), exploreHandler() (+9 more)

### Community 480 - "Phase 4: Output Hygiene - Context"
Cohesion: 0.10
Nodes (20): Canonical References, Claude's Discretion, Current implementation (the extension points), Deferred Ideas, Established Patterns, Existing Code Insights, Implementation Decisions, Integration Points (+12 more)

### Community 481 - "Phase 10 Plan 07: Close the Phase — CONTRIBUTING Pointer, Single-Definition Guard, Clean-Checkout Proof Summary"
Cohesion: 0.10
Nodes (20): 1. RED against the phase's base commit (82ffd60), 2. RED-then-GREEN against an injected regression, 3. RED-then-GREEN against a stale exception entry, Accomplishments, Actuals (#2632), Auto-fixed Issues, Clean-Checkout Proof (Task 3, ROADMAP criterion 3), Decisions Made (+12 more)

### Community 482 - "rwr_test.go"
Cohesion: 0.24
Nodes (18): buildRWRAdjacency(), computeGraphRelevance(), roundRWRScore(), sortRWRScores(), T, starSubgraph(), TestComputeGraphRelevance_DanglingNodeRetainsMass(), TestComputeGraphRelevance_EmptyNodeIDs() (+10 more)

### Community 483 - "search_test.go"
Cohesion: 0.18
Nodes (15): T, manyMatchingNodes(), TestQuery(), TestQueryDefaultCapAtMaxLimit(), TestQueryJSONShape(), TestQueryKindFilterOnFakeReader(), TestQueryLimitCapsAfterRanking(), TestQueryRankingTieBreak() (+7 more)

### Community 484 - "Pattern Assignments"
Cohesion: 0.10
Nodes (19): Additive-sibling renderer (SURF-06 shape, D-02), Archtest mechanism (go/packages, not regex) — D-10, File Classification, `go.mod` (modified — config), `internal/cli/files.go` (modified — CLI seam, TUI-02), `internal/cli/init.go` / `index.go` / `sync.go` (modified — progress seam, TUI-05), `internal/cli/present/archtest/import_graph_test.go` (test, archtest — TUI-01), `internal/cli/present/files.go` (component, transform) (+11 more)

### Community 485 - "Phase 9 Plan 1: release-please spine + non-vacuous drift guard Summary"
Cohesion: 0.10
Nodes (19): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, End-to-End Spine Proof (mandatory acceptance criterion, recorded verbatim), Files Created/Modified, Guard 1 — `TestReleaseWorkflowFileMatchesPattern` (drift guard) (+11 more)

### Community 486 - "Phase 9 Plan 6: main fast-forwarded; release-please's App-token path proven end to end via release PR #2 Summary"
Cohesion: 0.10
Nodes (19): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+11 more)

### Community 487 - "Pattern Assignments"
Cohesion: 0.10
Nodes (19): `CONTRIBUTING.md` (modified — pointer only, D-00), `env:` indirection instead of direct `${{ }}` interpolation in `run:`, File Classification, `.github/actions/install-task/action.yml` (new, CI composite action), `.github/workflows/bench.yml` (modified — self-analog), `.github/workflows/ci.yml` (modified — self-analog), `.github/workflows/release-please.yml` (modified — self-analog, D-15), `.github/workflows/release.yml` (modified — self-analog, D-06/D-07/D-08) (+11 more)

### Community 488 - "Extract"
Cohesion: 0.29
Nodes (16): Extract(), findNode(), FileResult, T, hasIntraEdge(), hasUnresolved(), newTestParser(), TestExtract_Calls() (+8 more)

### Community 489 - "Pattern Assignments"
Cohesion: 0.11
Nodes (18): Cancel-then-join-then-assert goroutine teardown, Env-var strict-equality parsing, Explicit named CI step for anything `go list ./...`/testdata can skip, File Classification, `internal/cli/serve.go` (modified), `internal/cli/serve_test.go` (extended), `internal/daemon/daemon.go` (modified — policy gate + `RunWithRetry`), `internal/daemon/soak_test.go` (extended, D-15) (+10 more)

### Community 490 - "Goal Achievement"
Cohesion: 0.11
Nodes (18): 1. Affected() BFS output-ordering determinism (backstop truth, 08-04), 2. ~~Signed v1.0.0 release cut (REL-02)~~ — OUT OF SCOPE (moved to Phase 9), Anti-Patterns Found, Behavioral Spot-Checks, Both arrays emptied 2026-07-28: the Affected() ordering item was RESOLVED (see, Gaps Summary, Goal Achievement, Human Verification Required (+10 more)

### Community 491 - "Phase 9 Plan 3: PR-title conventional-commit gate + actionlint static check Summary"
Cohesion: 0.11
Nodes (18): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Guard 1 — PR-title lint (Task 2 acceptance criteria), Guard 2 — `actionlint` job (Task 3 acceptance criteria) (+10 more)

### Community 492 - "Pattern Assignments"
Cohesion: 0.11
Nodes (18): Accept-and-reject (non-vacuous) test pairs for any pattern/guard, `docs/RELEASE-PROCEDURES.md` — §3/§4/§7 rewrite + new App section (MODIFY), Documented divergence over silent drift, `env:`-indirection — never interpolate `${{ }}` directly into `run:`, File Classification, `.github/workflows/ci.yml` — new PR-title conventional-commit lint job (MODIFY), `.github/workflows/release-please.yml` (NEW), `.github/workflows/release.yml` — `assemble` job, `Publish GitHub release` step (MODIFY, lines 266–284) (+10 more)

### Community 493 - "targetRoot"
Cohesion: 0.24
Nodes (15): Command, newGithooksCmd(), newGithooksInstallCmd(), newGithooksRemoveCmd(), newGithooksStatusCmd(), printHookErrors(), targetRoot(), dirNonEmpty() (+7 more)

### Community 494 - "BuildServer"
Cohesion: 0.38
Nodes (17): callTool(), deriveServeRepoPath(), CallToolResult, MCPServer, T, mcpWorktreeMismatchFixture(), resultText(), runGitM() (+9 more)

### Community 495 - "Phase 2: Code Review Report"
Cohesion: 0.11
Nodes (17): CR-01: The MCP worktree notice is unreachable through `codegraph serve --mcp` — WORK-01/WORK-02 is dead code on the MCP surface, CR-02: `TestGoldenParity/status`'s new `dbSizeBytes` assertion measures the wrong directory and fails on any clean checkout, Critical Issues, IN-01: `golden_parity_test.go`'s `WorktreeMismatch != nil` assertion is now a vestigial tautology with a stale message, IN-02: `Project:` line renders the raw `-p` value while the mismatch paths are absolute+resolved, IN-03: `TestNoticeOnWorktreeMismatch` asserts only a single-character prefix, IN-04: Markdown table cells interpolate names/paths without escaping `|`, Info (+9 more)

### Community 496 - "Pattern Assignments"
Cohesion: 0.11
Nodes (17): Atomic write (crash-safety), Best-effort / non-fatal degrade-to-message, File Classification, gitmeta exec contract (single-seam confinement), `internal/cli/githooks.go` (controller, request-response), `internal/cli/init.go` (D-07 advisory insertion), `internal/cli/uninit.go` (D-06 cleanup insertion), `internal/fsatomic/fsatomic.go` (utility, file-I/O) (+9 more)

### Community 497 - "Phase 9 Plan 2: Idempotent release.yml publish step (D-04) Summary"
Cohesion: 0.11
Nodes (17): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Extraction-guard non-vacuity (Task 1 acceptance criterion), Files Created/Modified, GREEN — Task 2, against the edited release.yml, Issues Encountered (+9 more)

### Community 498 - "Phase 10 Plan 3: release.yml onto Namespace runners — darwin leg moved by maintainer decision, D-07/D-08 machine-enforced Summary"
Cohesion: 0.11
Nodes (17): Accomplishments, Actuals (#2632), Auto-fixed Issues, Checkpoint Decision (blocking, human), Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified (+9 more)

### Community 499 - "daemon_test.go"
Cohesion: 0.28
Nodes (16): Command, T, TestDaemonBareCmd_InteractiveAllowed_CallsRunDaemonPicker(), TestDaemonBareCmd_InteractiveAllowed_EmptyRegistry_NoPicker(), TestDaemonBareCmd_NonTTY_EmptyRegistry_PrintsNoRunningDaemons(), TestDaemonBareCmd_NonTTY_PrintsSeededRecordsCurrentRepoFirst(), TestDaemonStartCmdPolicyDisabledExitsCleanly(), TestDaemonStopCmd_AggregatedError_ExitsNonZero() (+8 more)

### Community 500 - "daemonpicker_test.go"
Cohesion: 0.33
Nodes (16): dispatchDaemonAction(), aKeyMsg(), downKeyMsg(), KeyPressMsg, T, rec(), TestDaemonPickerModel_CancelSignalsNothing(), TestDaemonPickerModel_CurrentRepoFirstOrdering() (+8 more)

### Community 501 - "Phase 1 Plan 8: Java + C# D-09 Edge-Kind Extraction Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 502 - "Phase 1 Plan 9: Python + TS/JS D-09 Edge-Kind Extraction Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 503 - "Phase 1 Plan 12: Named-Symbol Seeding + Per-Overload Disambiguation Tiers (H13) Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Documented Divergences (not auto-fixes — pre-declared by the plan's own D-02/D-10 framework), Files Created/Modified, Issues Encountered (+8 more)

### Community 504 - "Phase 1 Plan 17: Behavioral Parity Harness — F5 Regeneration + D-02 Oracle Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 505 - "Pattern Assignments"
Cohesion: 0.12
Nodes (16): Archtest "assert wiring, not a replica" + mutation-proof sanity check, Diagnostics-to-stderr-never-stdout (already-stated repo-wide rule), File Classification, `go/packages`-based structural confinement (NOT regex), `internal/graphstore/archtest/stdout_confinement_test.go` (new, D-06b), `internal/graphstore/logger.go` (new), `internal/graphstore/logger_test.go` (new), `internal/graphstore/pebble_store.go` (modify, storage seam) (+8 more)

### Community 506 - "Phase 08 Plan 07: Charm Supply-Chain CGo Closure Audit Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+8 more)

### Community 507 - "Phase 10 Plan 1: Tracer — go.tool.mod, install-task, Taskfile, and Namespace runners proven on one real CI job Summary"
Cohesion: 0.12
Nodes (16): Accomplishments, Actuals (#2632), Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered (+8 more)

### Community 508 - "Phase 10: Local Build Tooling & CONTRIBUTING - Discussion Log"
Cohesion: 0.12
Nodes (16): Bench & gate footguns, CI-vs-Taskfile authority, Claude's Discretion, Corrections made during discussion, Deferred Ideas, Granularity of the CI → task call, Phase 10: Local Build Tooling & CONTRIBUTING - Discussion Log, Release-pipeline scope (+8 more)

### Community 509 - "Record"
Cohesion: 0.25
Nodes (12): Record, Command, View, newDaemonPickerModel(), printDaemonPickerResult(), resolveRepoRoot(), RunDaemonPicker(), SortRecordsCurrentFirst() (+4 more)

### Community 510 - "captureDiagWriter"
Cohesion: 0.23
Nodes (12): quietLogger, getDiagWriter(), setDiagWriter(), captureDiagWriter(), Buffer, T, TestOpenInjectsQuietLogger(), TestQuietLoggerErrorfWritesProvenance() (+4 more)

### Community 511 - "Phase 1 Plan 2: D-09 Edge-Kind Vocabulary Foundation Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+7 more)

### Community 512 - "Phase 1 Plan 5: Go Reference-Language D-09 Edge-Kind Extraction Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 513 - "Phase 2 Plan 2: Status content — dbSizeBytes + FilesByLanguage Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Final `--json` Key Set, Issues Encountered, Next Phase Readiness (+7 more)

### Community 514 - "Phase 3 Plan 1: Watch-Policy Port Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 515 - "Phase 3 Plan 2: Watcher Policy Enforcement + Defer-and-Retry Convergence Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 516 - "Phase 5 Plan 3: internal/githooks — Install/Remove/Status Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 517 - "Phase 5 Plan 4: codegraph githooks CLI command tree Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 518 - "Phase 7 Plan 07: Daemon Picker & Command Tree Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 519 - "Phase 08 Plan 05: affected scripting flags (--stdin/-d/-f/-q/-j) Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+7 more)

### Community 520 - "Phase 8 Plan 09: Release Runbook, Drop-In Gate & Caveat Retirement Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+7 more)

### Community 521 - "Phase 9 Plan 5: GitHub App creation + secrets checkpoint Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+7 more)

### Community 522 - "Phase 10 Plan 2: ci.yml's test job onto task targets — build/test/vet/vuln Taskfile surface, serial contributor wrappers, DEV-01's go vet gate Summary"
Cohesion: 0.12
Nodes (15): Accomplishments, Actuals (#2632), Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered (+7 more)

### Community 523 - "newFilesCmd"
Cohesion: 0.14
Nodes (10): Command, newFilesCmd(), printFileTree(), ChoosePresentation(), T, TestChoosePresentation(), Command, InteractiveAllowed() (+2 more)

### Community 524 - "Phase 1 Plan 1: Behavioral Fixture Harness + TS 1.3.1 Golden Capture Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 525 - "Phase 1 Plan 11: Subgraph-Construction Heuristics H10-H12 (EXPL-02) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 526 - "Phase 1 Plan 13: Per-File Scoring, Hard Exclusion & Buried-Rescue (H14-H16) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 527 - "Phase 1 Plan 14: File-Relevance Gate, Central-File Selection & 5-Tier Sort (H17-H19) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 528 - "Phase 1 Plan 15: D-09 Force Re-Index (F4) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics (+6 more)

### Community 529 - "Phase 2: Code Review Report (iteration 2 — re-review of the fix commits)"
Cohesion: 0.13
Nodes (14): CR-01 (BL-01): One cancelled MCP tool call permanently disables the worktree notice for the whole server — WR-01's fix poisons WR-02's shared cache, Critical Issues, IN-01: WR-01's CLI half is inert — `cmd.Context()` is always `context.Background()`, IN-02: golden `WorktreeMismatch` assertion is no longer a tautology, but its message is still wrong, IN-03: iteration-1 Info findings remain open (correctly out of the fixer's scope), Info, Phase 2: Code Review Report (iteration 2 — re-review of the fix commits), Summary (+6 more)

### Community 530 - "Phase 3 Plan 4: TEST-04 Subprocess Integration Harness Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 531 - "Phase 3 Plan 5: D-21 WATCH Subprocess Coverage Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 532 - "Phase 5 Plan 1: Extract internal/fsatomic Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 533 - "Phase 5 Plan 2: gitmeta IsGitRepo & HooksDir Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+6 more)

### Community 534 - "Phase 06 Plan 01: Rendering Seam Foundation — TUI-01 Archtest + Charm Skeleton Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 535 - "Phase 06 Plan 02: Rendering Seam — Pretty status/files Rendering Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+6 more)

### Community 536 - "Phase 06 Plan 03: Rendering Seam — TTY-Gated Progress Feedback (TUI-05) Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+6 more)

### Community 537 - "Phase 07 Plan 03: PPID Reparent Watchdog Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 538 - "Phase 07 Plan 05: Wire Registry + PPID Watchdog into daemon.Run Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 539 - "Phase 8 Plan 4: Affected depth-bounded BFS with test-leaf pruning Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Auto-fixed Issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 540 - "Warnings"
Cohesion: 0.13
Nodes (14): CR-01: `affected --stdin` has no cap on the number of ingested lines — unbounded-memory DoS, Critical Issues, IN-01: `downloadReleaseAsset` has no `context.Context` for early cancellation, Info, Phase 08: Code Review Report, Summary, Warnings, WR-01: `dirPrefixMatches` has no path-separator boundary check — `--dir foo` also matches sibling directory `foobar/` (+6 more)

### Community 541 - "Phase 8 — Validation Strategy"
Cohesion: 0.13
Nodes (14): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), GAP-1 — SURF-04: `--quiet` control-character skip was untested → CLOSED, GAP-2 — SURF-01: MCP tool-schema defaults were unpinned → CLOSED, GAP-3 — REL-03: `BENCHMARKS.md` not pinned to raw runs → DEFERRED, Gaps Found and Closed (2026-07-26), Manual-Only Verifications, Per-Requirement Verification Map, Phase 8 — Validation Strategy (+6 more)

### Community 542 - "Phase 9 Plan 4: RELEASE-PROCEDURES.md rewrite + recorded divergences Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, `docs/RELEASE.md` Comparison (must_haves item), Files Created/Modified, Issues Encountered, Next Phase Readiness (+6 more)

### Community 543 - "Phase quick-260728-rfx: Rewrite REL-02 as a Testable Property Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Deviations from Plan, Every REL-02 Occurrence Found in STATE.md and PROJECT.md (exhaustive, per plan's ITEM-7 report requirement), Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+6 more)

### Community 544 - "Quick Task 260731-0a0: Fix govulncheck GO-2026-6061 Summary"
Cohesion: 0.13
Nodes (14): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Govulncheck Evidence (RED -> GREEN), Issues Encountered, Next Phase Readiness (+6 more)

### Community 545 - ".Run"
Cohesion: 0.23
Nodes (9): Daemon, Option, Context, Duration, Mutex, Options, Stats, jitter() (+1 more)

### Community 546 - "DetectIndexMismatch"
Cohesion: 0.24
Nodes (11): fixtureCase, DetectIndexMismatch(), Context, Mismatch, T, resolveExpected(), TestFixtureVerdicts(), CommonDir() (+3 more)

### Community 547 - "NewProgress"
Cohesion: 0.31
Nodes (10): Mutex, NewProgress(), T, TestProgress_FramesContainLabelAndANSI(), TestProgress_NoGoroutineLeak(), TestProgress_StderrOnly(), TestProgress_StopClearsLine(), TestProgress_StopIdempotent() (+2 more)

### Community 548 - ".Explore"
Cohesion: 0.23
Nodes (10): clampExploreBudget(), computeFileTermHits(), exploreZeroResult(), getExploreOutputBudget(), Engine, Reader, groupMatchesByFile(), computeSkeletonFiles() (+2 more)

### Community 549 - "Phase 1 Plan 04: Node Multi-Def Enumeration + Budget/Overflow + Never-Empty Narrowing Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Documented, out-of-scope divergences (not bugs), Files Created/Modified, Issues Encountered, Next Phase Readiness (+5 more)

### Community 550 - "Phase 1 Plan 6: RWR Graph-Relevance Core (EXPL-02) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Metrics, Next Phase Readiness (+5 more)

### Community 551 - "Phase 1 Plan 16: Explore Pipeline Wiring (H1-H21) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Documented, in-scope design decisions (not bugs), Files Created/Modified, Issues Encountered, Next Phase Readiness (+5 more)

### Community 552 - "Goal Achievement"
Cohesion: 0.14
Nodes (13): Accepted / Future-Work Divergences (non-blocking), Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (+5 more)

### Community 553 - "Phase 2 Plan 5: Status Renderers (CLI padded columns + MCP bolded-key bullets) Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 554 - "Phase 2 Plan 6: MCP Markdown Wiring + Server-Scoped Worktree Notice Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Final Signatures, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 555 - "Phase 2 Plan 7: CLI Wiring — Sectioned Status + Compact Worktree Notice Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 556 - "Info"
Cohesion: 0.14
Nodes (13): CR-01: `IsLockHeld` matches neither of Pebble's Windows lock-failure forms — the entire round-1 CR-01 fix is inert on both shipped Windows targets, Critical Issues, IN-01: Requeue give-up log undercounts by one, and the counter never resets after exhaustion — a later contention episode gets zero requeues until one success intervenes, IN-02: Requeue-vs-shutdown TOCTOU can delay `Run`'s return by one full debounce window (2s default), IN-03: `Open`'s retry changes timing (not semantics) for every pre-existing caller — worth one line of documentation at the two probes, IN-04: `codegraph daemon`'s new disabled message is MCP-branded and is the third copy of the reason re-derivation (round-1 IN-02's pattern, now in one more place), IN-05: Round-1 Info findings IN-01 through IN-04 remain open, Info (+5 more)

### Community 557 - "Info"
Cohesion: 0.14
Nodes (13): IN-01: `locked_unix_test.go`'s provenance comment is factually wrong — pebble's cross-process fcntl form is a bare errno, not PathError-wrapped, IN-02: `TestOpenConvergesWhenHolderCloses` is time-based (150ms release vs ~400ms budget) rather than event-synchronized, IN-03: Requeue give-up log undercounts by one, and the counter never resets after exhaustion — contradicting `maxFlushLockRequeues`' own doc comment, IN-04: Requeue-vs-shutdown TOCTOU can delay `Run`'s return by up to one full debounce window (2s default), IN-05: The watch-disabled reason is re-derived at three independent sites, and `codegraph daemon`'s copy is MCP-branded, IN-06: `codegraph daemon`'s disabled branch has zero test coverage — the only shipped consumer of round-1 WR-01's fix is unexercised, IN-07: `daemon.Run` holds the lock blocked on `<-ctx.Done()` even if the watch loop exits early — silent zombie lock-holder, IN-08: CI's process substitution masks a partial `go list` failure (+5 more)

### Community 558 - "Phase 07 Plan 04: Daemon Stop Signaling Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 559 - "Phase 7 Plan 06: Bubbles Checkbox Multi-Select for Install/Uninstall Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 560 - "Phase 7 Plan 08: TEST-03 Byte-Invariance & Piped Never-Hang Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 561 - "Phase 8 Plan 1: Engine Impact-Depth Default & Short Flags Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 562 - "Phase 8 Plan 6: FLAG-PARITY.md matrix + drift test Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+5 more)

### Community 563 - "Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Discussion Log"
Cohesion: 0.14
Nodes (13): Claude's Discretion, Deferred Ideas, Folded Todos, Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Discussion Log, Phase sequencing (SURF block vs REL block), REL-01 — proving "no NEW CGo" from the Charm closure, REL-02 — release cut sequencing, REL-03 — benchmark scope (+5 more)

### Community 564 - "Phase 9 — Validation Strategy"
Cohesion: 0.14
Nodes (13): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 9 — Validation Strategy, Row 5 — retired rather than filled, Row 7 — the gap that was real, and its non-vacuity proof, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6) (+5 more)

### Community 565 - "Phase 10 Plan 04: Runner Identity, Storage-Frame Investigation & Baseline Staleness Fix Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed / maintainer-directed issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+5 more)

### Community 566 - "Phase 10 Plan 05: One-Definition Pre-Tag Cross-Target Sweep Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Known Gaps (recorded per reporting-honesty, not left silent), Next Phase Readiness, Performance (+5 more)

### Community 567 - "Phase 10 Plan 06: Runner/ScratchFS Category-Error Gate + Reproducibility Task Targets Summary"
Cohesion: 0.14
Nodes (13): Accomplishments, Auto-fixed / orchestrator-directed issues, Decisions Made, Dependency graph, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness (+5 more)

### Community 568 - "v1.0 Requirements"
Cohesion: 0.14
Nodes (14): Behavioral Parity — explore (EXPL), Behavioral Parity — node (NODE), Behavioral Parity — status content (STAT), Behavioral Parity Test Harness (TEST), Daemon Model (DMON), Git Sync Hooks (HOOK), Git / Worktree Awareness (WORK), Human TUI / Rendering (TUI) (+6 more)

### Community 569 - ".Install"
Cohesion: 0.32
Nodes (5): geminiTarget, geminiConfigPath(), geminiInstructionsPath(), Location, init()

### Community 570 - "githooks.go"
Cohesion: 0.22
Nodes (12): HookStatus, InstallResult, RemoveResult, StatusResult, Context, hasMarkerLine(), Install(), isEffectivelyEmpty() (+4 more)

### Community 571 - "setupIndexedFixtureWithTest"
Cohesion: 0.42
Nodes (12): execCmdTimeout(), Duration, T, setupIndexedFixtureWithTest(), TestAffectedEmptyStdinNoArgs(), TestAffectedFilter(), TestAffectedJSONQuiet(), TestAffectedQuietPathsOnly() (+4 more)

### Community 572 - "policy.go"
Cohesion: 0.23
Nodes (10): DetectWSL(), detectWSLUncached(), isWindowsDriveMount(), resetWSLCacheForTests(), T, TestDetectWSL(), TestWatchDisabledReason(), WatchDisabledReason() (+2 more)

### Community 573 - "Debug: perf regression gate reports ~10.6% throughput regression"
Cohesion: 0.15
Nodes (12): Constraints, Corrected Premise (evidence gathered before the session opened), Current Focus, Debug: perf regression gate reports ~10.6% throughput regression, Eliminated, Evidence, Necessary but not sufficient — why the guard alone does not make the gate valid, Original options as surfaced (retained for the record) (+4 more)

### Community 574 - "Phase 1 Plan 03: Query Tokenizers (H1 + H2) Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+4 more)

### Community 575 - "Phase 1 Plan 07: Hybrid Gather Channels (H3-H6) Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+4 more)

### Community 576 - "Phase 01: Code Review Report — Behavioral Parity (explore & node)"
Cohesion: 0.15
Nodes (12): CR-01: D-09 `overrides` synthesis collapses overloaded methods, silently dropping/misattributing RANK_EDGES `overrides` edges, CR-02: NODE-03 (file/line narrowing) is implemented and unit-tested but unreachable from any CLI/MCP entry point, Critical Issues, IN-01: `resolveNodeForDetail`'s exact-match `file` semantics diverge from TS's substring semantics without being flagged as a divergence at the call site, IN-02: `channel1BaseScore`/`channel3BaseScore` (10.0) are documented as "not verbatim, chosen for cross-channel consistency" — worth a tracking note, Info, Phase 01: Code Review Report — Behavioral Parity (explore & node), Summary (+4 more)

### Community 577 - "Phase 2 Plan 4: Engine.startPath + Live WorktreeMismatch Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+4 more)

### Community 578 - "Phase 2: status Content & Git/Worktree Awareness - Discussion Log"
Cohesion: 0.15
Nodes (12): Claude's Discretion, DB Size Semantics (STAT-01), Deferred Ideas, Detection Wiring & Caching (WORK-01/02), files-by-language (STAT-02), Live Signals (STAT-03), Notice Delivery & Verbatim Strings (WORK-02), Phase 2: status Content & Git/Worktree Awareness - Discussion Log (+4 more)

### Community 579 - "Phase 2: status Content & Git/Worktree Awareness - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Corrections to CONTEXT.md (evidence-based), Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit (+4 more)

### Community 580 - "Goal Achievement"
Cohesion: 0.15
Nodes (12): Anti-Patterns Found, Behavioral Spot-Checks / Test Execution (run by me, not sourced from SUMMARY), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Method, Observable Truths (+4 more)

### Community 581 - "Phase 4 Plan 1: Pebble Logger Routing (HYG-01) Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 4 Plan 1: Pebble Logger Routing (HYG-01) Summary (+4 more)

### Community 582 - "Phase 4 Plan 2: Stdout Confinement Archtest (HYG-02 Structural Guard) Summary"
Cohesion: 0.15
Nodes (12): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 4 Plan 2: Stdout Confinement Archtest (HYG-02 Structural Guard) Summary (+4 more)

### Community 583 - "Phase 5: Git Sync Hooks - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 5: Git Sync Hooks - Research (+4 more)

### Community 584 - "Code Examples"
Cohesion: 0.15
Nodes (13): Code Examples, DEFAULT_SYNC_HOOKS (source: `sync/git-hooks.js:60-61`), Existing CLI reachability test pattern to follow (source: `internal/cli/install_test.go:66-79`, `cli_test.go:55`, this repo), Existing Go exec-contract pattern to follow (source: `internal/gitmeta/worktree.go:35-51`, this repo), Existing real-git fixture helper to reuse (source: `internal/gitmeta/fixtures_test.go:20-54`, this repo), init advisory call sites (source: `bin/codegraph.js:544-548, 586-590`), The Go extraction target (source: `internal/agents/shared.go:312-360`, this repo — D-09), uninit cleanup call site (source: `bin/codegraph.js:629-637`) (+5 more)

### Community 585 - "Phase 05: Code Review Report"
Cohesion: 0.15
Nodes (12): CR-01: Unterminated/malformed marker block silently destroys trailing file content, Critical Issues, IN-01: Duplicate pluralization logic between uninit.go and githooks.go, IN-02: Watcher-enabled advisory test depends on ambient environment rather than asserting it, IN-03: `githooks status`/TS's `isSyncHookInstalled` report "installed" based on marker text only, not executability, Info, Phase 05: Code Review Report, Summary (+4 more)

### Community 586 - "Goal Achievement"
Cohesion: 0.15
Nodes (12): 7-Round Review Loop, Anti-Patterns Found, Behavioral Spot-Checks (live binary, real-git fixtures), Full Test Suite Run, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification (+4 more)

### Community 587 - "Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Research (+4 more)

### Community 588 - "Goal Achievement"
Cohesion: 0.15
Nodes (12): Acknowledged Gaps, AG-07-01 — Windows `daemon stop` termination and PPID-watchdog runtime semantics are unobserved, Anti-Patterns Found, Behavioral Spot-Checks, Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification (+4 more)

### Community 589 - "Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 8: Surface Reconciliation & Signed v1.0.0 Release - Research (+4 more)

### Community 590 - "Phase 8: Code Review Report (iteration 3 — re-review)"
Cohesion: 0.15
Nodes (12): Commands run and results, Critical Issues, Fix 1 — CR-01 (`333ab82`, `internal/cli/affected.go`): VERIFIED CORRECT, Fix 2 — WR-01 (`331a5a6`, `internal/cli/affected.go`): VERIFIED CORRECT, Fix 3 — WR-02 (`4feb6ff`, `internal/query/traverse.go`): VERIFIED CORRECT, IN-01: Two doc comments in `affected.go` still describe the pre-fix (and factually inverted) `bufio` semantics, IN-02: WR-01 changed the user-visible over-limit message to a double-prefixed one that leaks the internal package name, IN-03: The `--stdin` over-length error message is off by one against the behavior it reports (+4 more)

### Community 591 - "Fixed Issues"
Cohesion: 0.15
Nodes (12): CR-01: `affected --stdin` has no cap on the number of ingested lines — unbounded-memory DoS, Fixed Issues, IN-01: `downloadReleaseAsset` has no `context.Context` for early cancellation, Phase 08: Code Review Fix Report, Skipped Issues, WR-01: `dirPrefixMatches` has no path-separator boundary check, WR-02: `AffectedResult.Files` has no nil→empty-slice normalization, WR-03: `affected --quiet` writes raw, unescaped file paths (+4 more)

### Community 592 - "Phase 9 — Security"
Cohesion: 0.15
Nodes (12): Accepted Risks Log, Caveats on CLOSED verdicts, Eliminated — T-09-08-08, Guards break-tested rather than merely observed, Moot — the 09-07 group (8 threats), Phase 9 — Security, Security Audit Trail, Sign-Off (+4 more)

### Community 593 - "Phase 10: Local Build Tooling & CONTRIBUTING - Research"
Cohesion: 0.15
Nodes (12): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 10: Local Build Tooling & CONTRIBUTING - Research (+4 more)

### Community 594 - "Warnings"
Cohesion: 0.15
Nodes (12): IN-01: `test:unit`'s daemon-exclusion filter only matches the exact package path, Info, Phase 10: Code Review Report, Summary, Warnings, WR-01: `docs/RELEASE-PROCEDURES.md` still names the pre-migration runner classes, WR-02: `docs/RELEASE-PROCEDURES.md` contradicts its own correction about the SLSA provenance subject, WR-03: `docs/RELEASE-PROCEDURES.md` and the repo's own required-checks fixture disagree about whether `main` has a ruleset (+4 more)

### Community 595 - "statusWorktreeMismatchFixture"
Cohesion: 0.47
Nodes (11): noticeCase, containsBareNoticeGlyph(), T, noticeCommandCases(), noticeGlyph(), runGitC(), statusWorktreeMismatchFixture(), TestNoticeAbsentOnCleanTree() (+3 more)

### Community 596 - "Contributor Covenant Code of Conduct"
Cohesion: 0.17
Nodes (12): 1. Correction, 2. Warning, 3. Temporary Ban, 4. Permanent Ban, Attribution, Contributor Covenant Code of Conduct, Enforcement, Enforcement Guidelines (+4 more)

### Community 597 - "Cutting a release (maintainer runbook)"
Cohesion: 0.17
Nodes (11): 10. Recorded divergences (Phase 9), 1. Pre-tag gate (mandatory — D-09), 2. Tag conventions, 3. Branch/tag model, 4. Cutting a release (now automated by release-please), 5. The `verify.go` LOCKED contract, 6. Post-release verification, 7. Rollback / cleanup (+3 more)

### Community 598 - "Phase 1 Plan 10: Post-Merge Gather Rerankers (H7-H9) Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 1 Plan 10: Post-Merge Gather Rerankers (H7-H9) Summary (+3 more)

### Community 599 - "Code Examples"
Cohesion: 0.17
Nodes (12): §10. Current Go `Engine`/CLI extension points (confirmed exact bug + shape), §11. Existing Go/TS edge-kind inventory (the parity gap, exact), §1. Exact-symbol tokenizer (feeds named-symbol seeding), §2. FTS-term tokenizer + stopwords (EXPL-01's literal target), §3. RWR core (EXPL-02's load-bearing algorithm), §4. File-relevance gate (EXPL-03) — a 5-way OR, not a threshold, §5. "No covering tests" warning (EXPL-04) — exact trigger + exact string, §6. Multi-def enumeration + generated-files-last (NODE-01) (+4 more)

### Community 600 - "Phase 2 Plan 1: internal/gitmeta — Git Worktree Detection Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 2 Plan 1: internal/gitmeta — Git Worktree Detection Summary (+3 more)

### Community 601 - "Phase 2 Plan 3: SURF-06 Markdown Renderers Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Auto-fixed Issues, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance (+3 more)

### Community 602 - "Phase 4 Plan 3: MCP Stdout Purity & Sync Noise-Absence (End-to-End) Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 4 Plan 3: MCP Stdout Purity & Sync Noise-Absence (End-to-End) Summary (+3 more)

### Community 603 - "Fixed Issues"
Cohesion: 0.17
Nodes (11): Additional Fix (identified during continuation-run assessment), `bufio.Scanner` loop lacked a `scanner.Err()` check after the purity scan, CR-01: HYG-02 stdout guard does not scan any transitively-imported package, Fixed Issues, Full Suite Verification, IN-01: Stdout-confinement predicates cannot see indirect stdout writes, Phase 04: Code Review Fix Report, Skipped Issues (+3 more)

### Community 604 - "Fixed Issues"
Cohesion: 0.17
Nodes (11): Additional Fix (identified during continuation-run assessment), `bufio.Scanner` loop lacked a `scanner.Err()` check after the purity scan, CR-01: HYG-02 stdout guard does not scan any transitively-imported package, Fixed Issues, Full Suite Verification, IN-01: Stdout-confinement predicates cannot see indirect stdout writes, Phase 04: Code Review Fix Report, Skipped Issues (+3 more)

### Community 605 - "Phase 4 — Validation Strategy"
Cohesion: 0.17
Nodes (11): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 4 — Validation Strategy, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6), Test Infrastructure, Validation Audit 2026-07-16 (+3 more)

### Community 606 - "Goal Achievement"
Cohesion: 0.17
Nodes (11): 6 Post-SUMMARY Review-Fix Commits Verified Present and Correct, Anti-Patterns Found, Behavioral Spot-Checks / Mutation-Proof Checks (independently re-run, not trusting SUMMARY claims), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (+3 more)

### Community 607 - "Phase 5 Plan 5: Wire HOOK-03's watcher-fallback advisory into init/uninit Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 5 Plan 5: Wire HOOK-03's watcher-fallback advisory into init/uninit Summary (+3 more)

### Community 608 - "Phase 5: Git Sync Hooks - Discussion Log"
Cohesion: 0.17
Nodes (11): Claude's Discretion, Command surface, Deferred Ideas, fsatomic extraction scope, Git-probe placement, HOOK-03 surfacing point, Marker block bytes, Phase 5: Git Sync Hooks - Discussion Log (+3 more)

### Community 609 - "Phase 7 Plan 2: Charm-Free Global Daemon Registry Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 7 Plan 2: Charm-Free Global Daemon Registry Summary (+3 more)

### Community 610 - "Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Discussion Log"
Cohesion: 0.17
Nodes (11): Charm isolation & interactive seam (TUI-01, TUI-03, TUI-04), Claude's Discretion, Daemon command shape & lifecycle (DMON-01, DMON-02), Deferred Ideas, Global daemon registry (DMON-04), install / uninstall multi-select (TUI-03), Non-interactive fallbacks (TUI-04), Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Discussion Log (+3 more)

### Community 611 - "Phase 08 Plan 02: files --dir prefix filter + -j alias Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 08 Plan 02: files --dir prefix filter + -j alias Summary (+3 more)

### Community 612 - "Phase 08 Plan 03: Mechanical short-flag parity + upgrade --force Summary"
Cohesion: 0.17
Nodes (11): Accomplishments, Decisions Made, Deviations from Plan, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 08 Plan 03: Mechanical short-flag parity + upgrade --force Summary (+3 more)

### Community 613 - "Phase 9: release-please + GoReleaser - Discussion Log"
Cohesion: 0.17
Nodes (11): Claude's Discretion, Conventional-commit input gate, Deferred Ideas, GoReleaser's actual role — does `goreleaser release` enter the picture?, How `v1.0.0` specifically gets cut, Phase 9: release-please + GoReleaser - Discussion Log, Prerelease / rc story, Release-creation collision — `gh release create` vs. release-please's own release (+3 more)

### Community 614 - "Phase 9: release-please + GoReleaser - Research"
Cohesion: 0.17
Nodes (11): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Environment Availability, Metadata, Open Questions, Package Legitimacy Audit, Phase 9: release-please + GoReleaser - Research (+3 more)

### Community 615 - "Goal Achievement"
Cohesion: 0.17
Nodes (11): Anti-Patterns Found, Behavioral Spot-Checks, Deferred / Contextual Notes (not gaps), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (ROADMAP Success Criteria) (+3 more)

### Community 616 - "codegraph"
Cohesion: 0.17
Nodes (12): codegraph, Commands, Contributing, Install, Language support, License, Performance, Quick start (+4 more)

### Community 617 - "Parser"
Cohesion: 0.22
Nodes (5): countingParser, Once, NewTree(), Parser, Tree

### Community 618 - "initGitRepo"
Cohesion: 0.45
Nodes (10): T, initGitRepo(), TestGithooksInstall_AllHooksUnwritable_ShowsSyncFallbackMessage(), TestGithooksInstall_NonGitDirectory_SkipsCleanlyWithExitZero(), TestGithooksInstall_RealCobraTree(), TestGithooksRemove_AfterInstall_StripsMarkerFromAllHooks(), TestGithooksRemove_MalformedMarkerBlock_SurfacesWarning(), TestGithooksRemove_NothingInstalled_ShowsNothingToRemoveMessage() (+2 more)

### Community 619 - "daemonDelegate"
Cohesion: 0.24
Nodes (5): Cmd, Item, Model, Msg, daemonDelegate

### Community 620 - "Phase 1: Behavioral Parity — explore & node - Research"
Cohesion: 0.18
Nodes (10): Architectural Responsibility Map, Assumptions Log, Don't Hand-Roll, Metadata, Open Questions, Package Legitimacy Audit, Phase 1: Behavioral Parity — explore & node - Research, Phase Requirements (+2 more)

### Community 621 - "Fixed Issues"
Cohesion: 0.18
Nodes (10): CR-01: The MCP worktree notice is unreachable through `codegraph serve --mcp`, CR-02: `TestGoldenParity/status`'s `dbSizeBytes` assertion measures the wrong directory, Fixed Issues, Phase 2: Code Review Fix Report, Skipped Issues, WR-01: `Engine.WorktreeMismatch()` hardcoded `context.Background()`, WR-02: `CachingDetector`'s map grows without bound, WR-03: Gate 4 failed open on a degraded git call (+2 more)

### Community 622 - "Fixed Issues"
Cohesion: 0.18
Nodes (10): BL-01 (Critical): cancelled MCP tool call permanently disabling the worktree notice, Fixed Issues, GOLDEN-01 (raised inside WR-01's finding text): `go test ./...` never runs the golden suite, Phase 2: Code Review Fix Report (iteration 2), Skipped Issues, Verification, WR-01: `deriveServeRepoPath` did not close the gap it claimed to close, WR-02: confinement guard could not distinguish which of CR-01's two params it anchors on (+2 more)

### Community 623 - "Phase 2 — Security"
Cohesion: 0.18
Nodes (10): Accept (documented risk; behavior confirmed where code-verifiable), Accepted Risks Log, Mitigate (verified present in source), Phase 2 — Security, Re-verified historical regressions (deep-review scar tissue), Security Audit Trail, Sign-Off, Threat Register (+2 more)

### Community 624 - "Phase 04: Code Review Report"
Cohesion: 0.18
Nodes (10): CR-01: HYG-02 stdout guard does not scan any transitively-imported package, despite claiming to cover "every package that can execute during a serve --mcp session", Critical Issues, IN-01: Stdout-confinement predicates cannot see indirect stdout writes, Info, Phase 04: Code Review Report, Summary, Warnings, WR-01: Data race on `stderrBuf` in the MCP stdout-purity test's failure path (+2 more)

### Community 625 - "Tests"
Cohesion: 0.18
Nodes (10): 1. Pebble Infof chatter discarded; Errorf/Fatalf preserved via diagWriter seam (HYG-01), 2. graphstore.Open injects quietLogger at the single pebble.Open seam; mutation-proof (HYG-01), 3. go/types stdout-write detection predicates (HYG-02), 4. Six serve-reachable packages free of stdout writes; transitive-closure guard (HYG-02), 5. serve --mcp stdout is pure JSON-RPC frames end-to-end (HYG-02), 6. sync stderr carries no Pebble noise shapes (HYG-01), Current Test, Gaps (+2 more)

### Community 626 - "Fixed Issues"
Cohesion: 0.18
Nodes (10): CR-01: Unterminated/malformed marker block silently destroys trailing file content, Fixed Issues, IN-01: Duplicate pluralization logic between uninit.go and githooks.go, IN-02: Watcher-enabled advisory test depends on ambient environment rather than asserting it, IN-03: `githooks status`/TS's `isSyncHookInstalled` report "installed" based on marker text only, not executability, Phase 05: Code Review Fix Report, Skipped Issues, WR-01: Install/Remove silently swallow per-hook write/delete errors with no diagnostic (+2 more)

### Community 627 - "Phase 5 — Validation Strategy"
Cohesion: 0.18
Nodes (10): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 5 — Validation Strategy, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6), Test Infrastructure, Validation Audit 2026-07-17 (+2 more)

### Community 628 - "Phase 6 — Validation Strategy"
Cohesion: 0.18
Nodes (10): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 6 — Validation Strategy, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6), Test Infrastructure, Validation Audit 2026-07-17 (+2 more)

### Community 629 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, Code Review Cross-Check (06-REVIEW.md), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths, Phase 6: Rendering Seam & Pretty status/files Verification Report (+2 more)

### Community 630 - "Phase 7 Plan 1: Interactive-Layer Deps & Dual-TTY Gate Summary"
Cohesion: 0.18
Nodes (10): Accomplishments, Auto-fixed Issues, Decisions & Deviations, Files Created/Modified, Issues Encountered, Next Phase Readiness, Performance, Phase 7 Plan 1: Interactive-Layer Deps & Dual-TTY Gate Summary (+2 more)

### Community 631 - "Phase 7 — Validation Strategy"
Cohesion: 0.18
Nodes (10): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 7 — Validation Strategy, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6), Test Infrastructure, Validation Audit 2026-07-19 (+2 more)

### Community 632 - "Phase 08 Plan 08: Re-run Head-to-Head Benchmarks vs TS 1.3.1 (REL-03) Summary"
Cohesion: 0.18
Nodes (10): Auto-fixed / Resolved Automatically, Dependency graph, Deviations from Plan, Issues Encountered, Next Phase Readiness, Phase 08 Plan 08: Re-run Head-to-Head Benchmarks vs TS 1.3.1 (REL-03) Summary, Self-Check: PASSED, Tech tracking (+2 more)

### Community 633 - "Phase 08 — Security"
Cohesion: 0.18
Nodes (10): Accepted Risks Log, Findings Outside the Register (informational), Mitigations That Diverge From Their PLAN Description, Phase 08 — Security, Security Audit Trail, Sign-Off, Soundness Caveat on a Closed Control, Threat Register (+2 more)

### Community 634 - "09-08 — COMPLETE. v0.2.0 published and verified."
Cohesion: 0.18
Nodes (10): 09-08 — COMPLETE. v0.2.0 published and verified., Acceptance criteria — evidence, Defects found while verifying, Deviation from the plan's task structure, stated not buried, Requirement status, Task 3 verification — all five, run against real artifacts, The chain closes exactly, The upgrade test initially failed, and why that was not a defect (+2 more)

### Community 635 - "Locked Decisions"
Cohesion: 0.18
Nodes (11): Claude's Discretion, D-00 — ROADMAP criterion 2 is already satisfied; record, do not redo, Deferred Ideas (OUT OF SCOPE), Gray area 1 — Who owns the command definitions, Gray area 2 — Tool bootstrap (MEASURED, not argued), Gray area 3 — Runners and caching, Gray area 4 — Test scope and missing toolchains, Gray area 5 — Dangerous and release-critical targets (+3 more)

### Community 636 - "Phase 10 — Validation Strategy"
Cohesion: 0.18
Nodes (10): audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117), Manual-Only Verifications, Per-Task Verification Map, Phase 10 — Validation Strategy, Sampling Rate, status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6), Test Infrastructure, Validation Audit 2026-08-03 (+2 more)

### Community 637 - "Goal Achievement"
Cohesion: 0.18
Nodes (10): Anti-Patterns Found, DEV-01 Sub-Must-Haves (merged from the 7 plans' frontmatter, deduplicated against the roadmap criteria above), Gaps Summary, Goal Achievement, Human Verification Required, Key Link Verification, Observable Truths (ROADMAP Success Criteria), Phase 10: Local Build Tooling & CONTRIBUTING Verification Report (+2 more)

### Community 638 - "Roadmap: CodeGraph Go"
Cohesion: 0.18
Nodes (11): Backlog, Milestones, Overview, Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG), Phase 999.3: Vulnerability scanning for the tool modfiles (BACKLOG), Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG), Phase 999.5: macOS Gatekeeper signing and notarization for darwin release binaries (BACKLOG), Phases (+3 more)

### Community 639 - "Phase Details"
Cohesion: 0.18
Nodes (11): Phase 10: Local Build Tooling & CONTRIBUTING, Phase 1: Behavioral Parity — explore & node, Phase 2: status Content & Git/Worktree Awareness, Phase 3: Watcher-on-MCP Default, Phase 4: Output Hygiene, Phase 5: Git Sync Hooks, Phase 6: Rendering Seam & Pretty status/files, Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select (+3 more)

### Community 640 - "v1.0 Standing Snapshot — Phases 1-5"
Cohesion: 0.18
Nodes (10): Bookkeeping drift, Cosmetic, Findings, Gaps (within phases 1-5), Per-Phase Status (built phases), Requirements Coverage — Phases 1-5, Scope, Suggested next steps (+2 more)

### Community 641 - "README.md"
Cohesion: 0.24
Nodes (3): Coverage Table, Language Capability Matrix, Shared Caveats

### Community 642 - "Contributing"
Cohesion: 0.20
Nodes (10): Approval gates, Building, Code of conduct, Contributing, Issue first, Licensing, Never do these, Pull requests (+2 more)

### Community 643 - "main.go"
Cohesion: 0.42
Nodes (9): corpusSpec, goldenCapture, colbymchenryCodegraphSpec(), fatal(), main(), regenerateCorpus(), syntheticParitySpec(), weftGoSpec() (+1 more)

### Community 644 - "NewCachingDetector"
Cohesion: 0.47
Nodes (9): NewCachingDetector(), T, TestCachingDetectorBounded(), TestCachingDetectorCancelledContextNotCached(), TestCachingDetectorMemoizesNegative(), TestCachingDetectorMemoizesPositive(), TestCachingDetectorNilReceiverSafety(), TestCachingDetectorRejectsNonexistentStartPath() (+1 more)

### Community 645 - "parser_test.go"
Cohesion: 0.44
Nodes (7): T, newStubParser(), TestParserAcceptsInputUnderCeiling(), TestParserCloseIsCallable(), TestParserRejectsOversizeInput(), TestTreeCloseIsIdempotent(), stubParser

### Community 646 - "Phase 2 — Validation Strategy"
Cohesion: 0.20
Nodes (9): Manual-Only Verifications, Per-Task Verification Map, Phase 2 — Validation Strategy, Sampling Rate, Test Infrastructure, ★ The Blind Spot (drives Wave 0 ordering), Validation Audit 2026-07-16, Validation Sign-Off (+1 more)

### Community 647 - "Phase 4: Output Hygiene - Discussion Log"
Cohesion: 0.20
Nodes (9): Claude's Discretion, Debug escape hatch (HYG-01), Deferred Ideas, Error destination & seam (HYG-01), HYG-02 enforcement mechanism, Pebble logger routing shape (HYG-01), Phase 4: Output Hygiene - Discussion Log, Reviewed Todos (not folded) (+1 more)

### Community 648 - "Phase 07: Code Review Report"
Cohesion: 0.20
Nodes (9): IN-01: Daemon picker/list rendering does not dedupe records by pid, IN-02: `daemon.List()` runs its self-heal side effect on every bare `codegraph daemon` invocation, before the interactivity check, Info, Phase 07: Code Review Report, Summary, Warnings, WR-01: Interactive daemon picker gives zero confirmation of what it actually stopped, WR-02: `daemon stop --all --path <p>` silently ignores `--path` with no validation (+1 more)

### Community 649 - "Phase 10 — Security"
Cohesion: 0.20
Nodes (9): Accepted Risks Log, Advisory — Unregistered Surface, Phase 10 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries (+1 more)

### Community 650 - "Roadmap: CodeGraph Go"
Cohesion: 0.20
Nodes (10): Backlog, Milestones, Overview, Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG), Phase 999.3: Vulnerability scanning for the tool modfiles (BACKLOG), Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG), Phase 999.5: macOS Gatekeeper signing and notarization for darwin release binaries (BACKLOG), Phases (+2 more)

### Community 651 - "mcp-capture.mjs"
Cohesion: 0.22
Nodes (5): child, normalize(), pending, projectPath, wrap()

### Community 652 - "startWatchdog"
Cohesion: 0.28
Nodes (7): CancelFunc, Context, Duration, startWatchdog(), T, TestWatchdogCancelsOnReparent(), TestWatchdogJoinsOnCtxCancelWithoutFiringCancel()

### Community 653 - "syntheticParityEngine"
Cohesion: 0.47
Nodes (8): copySyntheticParityFixture(), Engine, T, syntheticParityEngine(), TestExploreMultiWord(), TestExploreRejectsEmptyQuery(), TestExploreStructuralBeatsLexical(), TestExploreTestNotTop()

### Community 654 - "Open"
Cohesion: 0.44
Nodes (6): addRecursive(), Bool, Context, Open(), watchLoop(), Watcher

### Community 655 - "Debug Session: daemon-stale-linux-flake"
Cohesion: 0.22
Nodes (8): Constraints, Current Focus, Debug Session: daemon-stale-linux-flake, Eliminated, Evidence, Leading Hypothesis (orchestrator pre-analysis — VERIFY, do not assume), Resolution, Symptoms

### Community 656 - "Roadmap: CodeGraph Go"
Cohesion: 0.22
Nodes (5): Overview, Phases, Progress, Roadmap: CodeGraph Go, Requirements Archive: v1.0 Drop-in Parity & Human UX

### Community 657 - "Phase 1 — Validation Strategy"
Cohesion: 0.22
Nodes (8): Manual-Only Verifications, Per-Requirement Verification Map, Phase 1 — Validation Strategy, Sampling Rate, Test Infrastructure, Validation Audit 2026-07-16, Validation Sign-Off, Wave 0 Requirements

### Community 658 - "Phase 3: Watcher-on-MCP Default - Discussion Log"
Cohesion: 0.22
Nodes (8): Claude's Discretion, Concurrent-session convergence (WATCH-04), Deferred Ideas, Flag surface & precedence (WATCH-01), Handshake-path budget (WATCH-02), Phase 3: Watcher-on-MCP Default - Discussion Log, Subprocess harness architecture (TEST-04), Watch-policy port shape (WATCH-03)

### Community 659 - "Info"
Cohesion: 0.22
Nodes (8): IN-01: `locked_unix_test.go`'s provenance comment is factually wrong about pebble's cross-process error shape — the fcntl form is a bare errno, not a `PathError`-wrapped one, IN-02: `TestOpenConvergesWhenHolderCloses` is time-based (150ms release vs ~400ms budget) rather than event-synchronized — small residual flake window under heavy parallel CI load, IN-03: On windows, ANY `ERROR_SHARING_VIOLATION` escaping `pebble.Open` — not just the LOCK-file collision — classifies as ErrStoreLocked, Info, Phase 3: Code Review Report (Round 3 — targeted delta review of commits 7699e9c + 1bdcd9c), Summary, Warnings, WR-01: The `GOOS=windows go vet` gate that `locked_windows_test.go` cites as guarding the windows arm does not exist in CI — the windows classifier has no automated protection against regression

### Community 660 - "Phase 4 — Security"
Cohesion: 0.22
Nodes (8): Accepted Risks Log, Phase 4 — Security, Residual Risks (documented, non-blocking), Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 661 - "Phase 6: Rendering Seam & Pretty status/files - Discussion Log"
Cohesion: 0.22
Nodes (8): Archtest structure & guarded set (TUI-01), Charm v2 dependency scope, Claude's Discretion, Deferred Ideas, Phase 6: Rendering Seam & Pretty status/files - Discussion Log, Progress feedback (TUI-05), Rendering-seam architecture (where lipgloss lives), TTY / plain-pretty switch mechanism

### Community 662 - "Fixed Issues"
Cohesion: 0.22
Nodes (8): Fixed Issues, Invariant Verification, Phase 6: Code Review Fix Report, Skipped Issues, WR-01: `present/status.go` interpolates two untrusted/host-path strings into the pretty sink without `sanitizeControl`, WR-02: `progress_test.go` still uses manual `runtime.NumGoroutine()` polling instead of the repo's `goleak` convention, WR-03: `init.go`/`index.go`/`sync.go` never install a signal handler, so `Progress.Stop()`'s line-clear/goroutine-join never runs on Ctrl-C, WR-04: The spinner-wiring block is duplicated verbatim across `init.go`, `index.go`, and `sync.go`

### Community 663 - "Architecture Patterns"
Cohesion: 0.22
Nodes (9): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: TTY-gate before `tea.NewProgram()` — gate BOTH stdin and stdout, Pattern 2: Custom checkbox `list.ItemDelegate` for multi-select, Pattern 3: Charm-free registry self-heal (generalizes the existing single-lockfile pattern to many files), Pattern 4: Build-tag platform split (mirrors `procstart_linux.go`/`procstart_other.go` and `locked_windows.go`), Pattern 5: Reparent detection with a captured baseline (not a bare `ppid == 1` check), Recommended Project Structure (+1 more)

### Community 664 - "Fixed Issues"
Cohesion: 0.22
Nodes (8): Fixed Issues, IN-01: Daemon picker/list rendering does not dedupe records by pid, IN-02: `daemon.List()` runs its self-heal side effect on every bare `codegraph daemon` invocation, before the interactivity check, Phase 07: Code Review Fix Report, Skipped Issues, WR-01: Interactive daemon picker gives zero confirmation of what it actually stopped, WR-02: `daemon stop --all --path <p>` silently ignores `--path` with no validation, WR-03: `SortRecordsCurrentFirst` uses plain string equality while the actual stop path resolves symlinks

### Community 665 - "Fixed Issues"
Cohesion: 0.22
Nodes (8): Fixed Issues, IN-01: Daemon picker/list rendering does not dedupe records by pid, IN-02: `daemon.List()` runs its self-heal side effect on every bare `codegraph daemon` invocation, before the interactivity check, Phase 07: Code Review Fix Report, Skipped Issues, WR-01: Interactive daemon picker gives zero confirmation of what it actually stopped, WR-02: `daemon stop --all --path <p>` silently ignores `--path` with no validation, WR-03: `SortRecordsCurrentFirst` uses plain string equality while the actual stop path resolves symlinks

### Community 666 - "Narrative Findings (AI reviewer)"
Cohesion: 0.22
Nodes (8): CR-01: WR-06's `scanner.Buffer` call does not enforce `affectedStdinMaxLineBytes` — the real per-line ceiling is silently 64 KiB, not 4096 bytes, Critical Issues, Narrative Findings (AI reviewer), Phase 8: Code Review Report (iteration 2 — re-review), Summary, Warnings, WR-01: `ValidateAffectedFiles` is dead code — CR-01's cap enforcement is duplicated inline instead of calling it, and has no test coverage, WR-02: `Callees` was left out of the WR-05 determinism fix — it has a different, unproven ordering contract than its three sibling functions

### Community 667 - "09-07 — SKIPPED (not executed)"
Cohesion: 0.22
Nodes (8): 09-07 — SKIPPED (not executed), Controls added instead (cheaper, and permanent), Dependency graph, Deviation record, Requirement impact, What running it would have cost, What the skip costs, stated plainly, Why this plan was skipped

### Community 668 - "Architecture Patterns"
Cohesion: 0.22
Nodes (9): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: GitHub App token as the tag-push identity, Pattern 2: Manifest seeding to skip the bootstrap PR, Pattern 3: `Release-As:` footer precedence (the v1.0.0 forcing mechanism), Pattern 4: release-please owns the Release; `release.yml` only uploads (D-04), Pattern 5: Automated pre-tag 6-target sanity gate, Recommended Project Structure (+1 more)

### Community 669 - "Phase 09: Code Review Report"
Cohesion: 0.22
Nodes (8): IN-01: `docs/RELEASE.md`/`docs/RELEASE-PROCEDURES.md` "Corrected" callouts read as later additions bolted onto un-updated prose rather than a single coherent rewrite, Info, Phase 09: Code Review Report, Summary, Warnings, WR-01: `CheckRegression` never validates corpus/subject identity, only platform — the perf gate can pass against a mismatched baseline, WR-02: SLSA-provenance "attested over the checksums file" wording was corrected in some passages but not others — three internally-contradictory copies remain, WR-03: Two `pull_request_target` workflows write `GITHUB_OUTPUT` multi-line values with a fixed heredoc delimiter over attacker-controlled file paths

### Community 670 - "fix.md"
Cohesion: 0.25
Nodes (7): Blast radius, On that regression test, Required checklist, Root cause, Scope, The bug, The fix

### Community 671 - "PeakRSSBytes"
Cohesion: 0.32
Nodes (6): normalizeMaxrss(), PeakRSSBytes(), T, TestRSSNormalize(), TestSmoke(), ProcessState

### Community 672 - "Milestone v1.0 — Interim Audit (after Phase 3 of 8)"
Cohesion: 0.25
Nodes (7): Cross-Phase Integration — 7/7 seams WIRED (integration checker, fresh run), Milestone v1.0 — Interim Audit (after Phase 3 of 8), Nyquist Compliance, Phase Verifications — 3/3 executed phases passed, Requirements Coverage — 23/47, Tech Debt (non-blocking, aggregated), Verdict

### Community 673 - "Phase 1: Behavioral Parity — explore & node - Discussion Log"
Cohesion: 0.25
Nodes (7): Claude's Discretion, Deferred Ideas, Fixture Corpus, Fixture Equivalence Oracle, Phase 1: Behavioral Parity — explore & node - Discussion Log, RWR Determinism & Tie-Breaking, TS Ground-Truth Capture

### Community 674 - "Phase 1 — Security"
Cohesion: 0.25
Nodes (7): Accepted Risks Log, Phase 1 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 675 - "Architecture Patterns"
Cohesion: 0.25
Nodes (8): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Best-effort git subprocess with bounded timeout (D-03), Pattern 2: dbSizeBytes via best-effort directory walk (D-07, mirrors existing `newestSourceMtime`), Pattern 3: MB rendering (D-07), Pattern 4: Comma-grouped number formatting (D-10, hand-rolled, no `x/text`), Recommended Project Structure, System Architecture Diagram

### Community 676 - "Phase 3 — Security"
Cohesion: 0.25
Nodes (7): Accepted Risks Log, Phase 3 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 677 - "Architecture Patterns"
Cohesion: 0.25
Nodes (8): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Marker-fenced strip-then-append (NOT in-place replacement), Pattern 2: Trimmed-line marker matching, Pattern 3: Effectively-empty detection gates deletion, not just stripping, Pattern 4: gitmeta exec contract for hook-dir/repo probes, Recommended Project Structure, System Architecture Diagram

### Community 678 - "Phase 5 — Security"
Cohesion: 0.25
Nodes (7): Accepted Risks Log, Phase 5 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 679 - "Phase 6 — Security"
Cohesion: 0.25
Nodes (7): Accepted Risks Log, Phase 6 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 680 - "Common Pitfalls"
Cohesion: 0.25
Nodes (8): Common Pitfalls, Pitfall 1: `tea.NewProgram` on a non-TTY input can misbehave instead of erroring cleanly, Pitfall 2: PPID watchdog is fundamentally incompatible with a future detached daemon, Pitfall 3: Windows has no `SIGTERM` — `daemon stop` cannot signal-deliver gracefully to an arbitrary remote pid, Pitfall 4: watchdog goroutine leak under `goleak`, Pitfall 5: `bubbles/v2/list`'s default keybindings collide with checkbox-toggle intent, Pitfall 6: `-y`/`--yes` must bypass the TTY check itself, not just skip the render, Pitfall 7: an accidental charm import from `internal/daemon` breaks the TUI-01 archtest build

### Community 681 - "Phase 7 — Security"
Cohesion: 0.25
Nodes (7): Accepted Risks Log, Phase 7 — Security, Security Audit Trail, Sign-Off, Threat Register, threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate), Trust Boundaries

### Community 682 - "07-UAT.md"
Cohesion: 0.25
Nodes (7): 1. Bubbletea daemon picker visual rendering on a real TTY, 2. Install/uninstall checkbox multi-select visual rendering on a real TTY, 3. Windows daemon stop termination + PPID watchdog on a real Windows host, Current Test, Gaps, Summary, Tests

### Community 683 - "08-UAT.md"
Cohesion: 0.25
Nodes (7): 1. Cut the signed v1.0.0 release (REL-02, maintainer-manual), 2. Affected() BFS output-ordering determinism (SURF-04 backstop), Current Test, Gaps, Readiness Recheck, Summary, Tests

### Community 684 - "Architecture Patterns"
Cohesion: 0.25
Nodes (8): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: The `GO_TOOL` variable indirection, Pattern 2: `preconditions:` with actionable `msg:` for cross-toolchain gating (D-11), Pattern 3: The `install-task` composite action (module-proxy bootstrap), Pattern 4: `setup-go cache:false` + `nscloud-cache-action cache:go` ordering, Recommended Project Structure, System Architecture Diagram

### Community 685 - "Future Requirements"
Cohesion: 0.25
Nodes (8): Daemon (beyond explicit model), Future Requirements, Human Web UI, Out of Scope, Requirements: CodeGraph Go — Milestone v1.0 (Drop-in Parity & Human UX), Team Scale / Annotations (later milestones, per PROJECT.md), Traceability, Worktree (beyond parity)

### Community 686 - "enhancement.md"
Cohesion: 0.29
Nodes (6): Behavior change, If this is a performance change, Parity, Required checklist, What it does now, What was wrong with it

### Community 687 - "feature.md"
Cohesion: 0.29
Nodes (6): Compatibility, How it behaves, Parity, Required checklist, What this adds, Why it belongs here

### Community 688 - "stubStore"
Cohesion: 0.33
Nodes (3): stubStore, Reader, TestResolve_EndToEnd()

### Community 689 - "closeOverServeReachableImports"
Cohesion: 0.52
Nodes (6): assertCharmImporterExists(), closeOverServeReachableImports(), Package, T, isModuleInternalPackage(), TestNoCharmInServeReachablePackages()

### Community 690 - "fsatomic_test.go"
Cohesion: 0.48
Nodes (6): T, TestWriteFile_ByteExactContent(), TestWriteFile_CreatesParentDirs(), TestWriteFile_ExistingFilePreservesMode(), TestWriteFile_NewFileGetsMode0644(), TestWriteFile_NoTempFileLeftoverOnSuccess()

### Community 691 - "TestPublishReleaseStepBranches"
Cohesion: 0.62
Nodes (6): argvLineContainsAll(), T, runExtractedStep(), stubGHDir(), TestPublishReleaseStepBranches(), writePublishStepFixtureAssets()

### Community 692 - "Phase 1: Behavioral Parity — explore & node Fix Summary"
Cohesion: 0.29
Nodes (6): CR-01: D-09 `overrides` synthesis collapsed overloaded methods, CR-02: NODE-03 (file/line narrowing) wired end-to-end (CLI + MCP), Deferred (non-blocking, from 01-REVIEW.md), Fixed Issues, Phase 1: Behavioral Parity — explore & node Fix Summary, Verification

### Community 693 - "Phase 3: Code Review Fix Report (Iteration 2)"
Cohesion: 0.29
Nodes (6): CR-01 + WR-01: Lock-held classification moved to the Open seam behind an exported `ErrStoreLocked` sentinel, with a build-tagged Windows arm, Fixed Issues, Phase 3: Code Review Fix Report (Iteration 2), Skipped Issues, Verification, WR-02: Direct unit tests for the classifier and Open's retry loop

### Community 694 - "Phase 05: Code Review Report (Iteration 5 — Round-5 Fix Verification / Round-6 Final Clean-Check)"
Cohesion: 0.29
Nodes (6): IN-01: `WR-02` label reused for two unrelated findings within the same file, Info, Phase 05: Code Review Report (Iteration 5 — Round-5 Fix Verification / Round-6 Final Clean-Check), Summary, Warnings, WR-01: `Remove` still surfaces zero signal for a malformed marker block, unlike `Install`'s identical-condition handling

### Community 695 - "Phase 05: Code Review Fix Report"
Cohesion: 0.29
Nodes (6): Fixed Issues, IN-01: `WR-02` label reused for two unrelated findings within the same file, Phase 05: Code Review Fix Report, Skipped Issues, Verification, WR-01: `Remove` still surfaces zero signal for a malformed marker block, unlike `Install`'s identical-condition handling

### Community 696 - "Phase 05: Code Review Fix Report"
Cohesion: 0.29
Nodes (6): CR-01: `stripMarkerBlock`'s unterminated-marker guard is defeated by `Install`'s own append-raw-content recovery — silent data loss on the *next* strip, Fixed Issues, Phase 05: Code Review Fix Report, Skipped Issues, Verification, WR-01: `uninit`'s D-06 hook cleanup still silently discards `RemoveResult.Errors`, unlike the standalone `githooks remove` command

### Community 697 - "Phase 05: Code Review Report (Iteration 2 — Post-Fix Re-Review)"
Cohesion: 0.29
Nodes (6): CR-01: `stripMarkerBlock`'s unterminated-marker guard is defeated by `Install`'s own append-raw-content recovery — silent data loss on the *next* strip, Critical Issues, Phase 05: Code Review Report (Iteration 2 — Post-Fix Re-Review), Summary, Warnings, WR-01: `uninit`'s D-06 hook cleanup still silently discards `RemoveResult.Errors`, unlike the standalone `githooks remove` command

### Community 698 - "Phase 6: Code Review Report (re-run — iteration 2, verifies WR-01..WR-04 fixes)"
Cohesion: 0.29
Nodes (6): CR-01: WR-02's signal-handling fix neutralizes Ctrl-C for the entire duration of `init`/`index`/`sync` — a new regression, not present pre-fix, Critical Issues, Phase 6: Code Review Report (re-run — iteration 2, verifies WR-01..WR-04 fixes), Summary, Warnings, WR-05: The SIGINT/SIGTERM code path added by WR-02 has zero automated test coverage

### Community 699 - "Info"
Cohesion: 0.29
Nodes (6): IN-01: Daemon picker/list rendering does not dedupe records by pid (carried forward, still holds), IN-02: `daemon.List()` runs its self-heal side effect before the interactivity check (carried forward, still holds), IN-03: `printDaemonPickerResult`'s "nothing matched" wording diverges from `printStoppedDaemons`'s callers, Info, Phase 07: Code Review Report (iteration 2 — post-fix re-review), Summary

### Community 700 - "Architecture Patterns"
Cohesion: 0.29
Nodes (7): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: Shared-engine default, single consumer (SURF-01), Pattern 2: Per-command flag maps are independent (no cross-command collision), Pattern 3: Release cut sequence (REL-02) — reuse, do not rebuild, Recommended Project Structure, System Architecture Diagram

### Community 701 - "Common Pitfalls"
Cohesion: 0.29
Nodes (7): Common Pitfalls, Pitfall 1: SURF-01 looks like a one-line change but has a hidden shared-consumer risk if not verified, Pitfall 2: SURF-04's `--depth` is not "just a flag" — `Engine.Affected` needs a real BFS, Pitfall 3: `affected`'s current `cobra.MinimumNArgs(1)` blocks `--stdin`-only invocations, Pitfall 4: `install`'s `--auto-allow` is a genuine behavioral divergence from TS, not just a naming gap, Pitfall 5: `node`'s missing TS "file mode" is a scope-sizing trap, Pitfall 6: The "not yet drop-in" caveat appears in more places than PROJECT.md's obvious summary line

### Community 702 - "Fixed Issues"
Cohesion: 0.29
Nodes (6): CR-01: WR-06's `scanner.Buffer` call did not enforce `affectedStdinMaxLineBytes` — the real ceiling was silently 64 KiB, not 4096 bytes, Fixed Issues, Phase 08: Code Review Fix Report (iteration 2), Verification, WR-01: `ValidateAffectedFiles` was dead code — CR-01's cap was duplicated inline instead of calling it, with no test coverage, WR-02: `Callees` was left out of the WR-05 determinism fix

### Community 703 - "Fixed Issues"
Cohesion: 0.29
Nodes (6): CR-01: WR-06's `scanner.Buffer` call did not enforce `affectedStdinMaxLineBytes` — the real ceiling was silently 64 KiB, not 4096 bytes, Fixed Issues, Phase 08: Code Review Fix Report (iteration 2), Verification, WR-01: `ValidateAffectedFiles` was dead code — CR-01's cap was duplicated inline instead of calling it, with no test coverage, WR-02: `Callees` was left out of the WR-05 determinism fix

### Community 704 - "09-01-PLAN.md"
Cohesion: 0.29
Nodes (6): Artifacts this phase produces (whole phase), Artifacts this plan produces, Multi-Source Coverage Audit (phase-wide), Spec-less edge-probe disposition (3 probe items, 0 silent drops), STRIDE Threat Register, Trust Boundaries

### Community 705 - "Common Pitfalls"
Cohesion: 0.29
Nodes (7): Common Pitfalls, Pitfall 1: Required-status-check names are matched by rendered job `name:`, not workflow structure, Pitfall 2: `platforms:` vs `preconditions:` — same silent-skip shape, different field, Pitfall 3: Pre-existing goreleaser version mismatch between `ci.yml` and `release.yml`, Pitfall 4: `release-please.yml` needs the tool bootstrap but is NOT in D-06's runner-migration scope, Pitfall 5: The `bench:rebless`-vs-D-13 tension needs an explicit resolution, not a guess, Pitfall 6: `deps:` executes concurrently; `- task:` inside `cmds:` executes serially

### Community 706 - "10-UAT.md"
Cohesion: 0.29
Nodes (6): 1. release.yml darwin legs — goreleaser build, cosign signing, and SLSA attestation on namespace-profile-macos-6x14-tahoe, Current Test, Gaps, Notes, Summary, Tests

### Community 707 - "SEED-002: we need a homebrew installation path"
Cohesion: 0.29
Nodes (6): Breadcrumbs, Notes, Scope Estimate, SEED-002: we need a homebrew installation path, When to Surface, Why This Matters

### Community 708 - "ledger.go"
Cohesion: 0.48
Nodes (5): applyAdjustment(), AuditEntry(), GetBalance(), PostTransaction(), ReconcileLedger()

### Community 709 - "UserAccountManager"
Cohesion: 0.40
Nodes (3): UserAccountManager, Validate(), validateBalance()

### Community 710 - "TestMigrateCmdRefusesUnrecognizedTargetWithoutForce"
Cohesion: 0.53
Nodes (5): T, TestMigrateCmdEndToEnd(), TestMigrateCmdFailsLoudOnNonTSSource(), TestMigrateCmdRefusesUnrecognizedTargetWithoutForce(), TestMigrateCmdRegisteredAndFlags()

### Community 711 - "TestProgressCLINonTTYReachability"
Cohesion: 0.67
Nodes (5): assertNoANSI(), T, TestProgressCLINonTTYReachability(), TestProgressCLIQuietSuppressesSummary(), TestProgressWiringDoesNotBreakErrorPaths()

### Community 712 - "telemetry_test.go"
Cohesion: 0.53
Nodes (5): T, TestHelpEveryCommand(), TestHelpVersionCommand(), TestTelemetryStatement(), TestTelemetryStatementIsConst()

### Community 713 - "version_test.go"
Cohesion: 0.53
Nodes (5): T, TestRootVersionFlag(), TestVersionCommandJSON(), TestVersionCommandPlain(), TestVersionInfoDefaults()

### Community 714 - "Scope-Expansion Addendum (D-09/D-10)"
Cohesion: 0.33
Nodes (6): A. Schema-bump mechanics — the load-bearing risk, and why it is smaller than feared, B. Per-language extraction for the new edge kinds (priority-4: Go, Java, C#, Python, TS/JS), C.1 — The 9 RANK_EDGES members (EXPL-02), C.2 — The full auxiliary-heuristic stack to port (D-10, in pipeline order), C. Definitive spec confirmation — the planner's D-10 checklist, Scope-Expansion Addendum (D-09/D-10)

### Community 715 - "Architecture Patterns"
Cohesion: 0.33
Nodes (6): Anti-Patterns to Avoid, Architecture Patterns, Pattern 1: RWR power iteration (EXPL-02 core), Pattern 2: Generated-files-last stable sort (NODE-01/D-07), Recommended Project Structure, System Architecture Diagram — TS 1.3.1's actual explore pipeline

### Community 716 - "SURF-06: MCP Markdown Output (added 2026-07-15 — scope change, D-16)"
Cohesion: 0.33
Nodes (6): Conflicts with this document's earlier findings, Ordering determinism, Recommended markdown shape, SURF-06: MCP Markdown Output (added 2026-07-15 — scope change, D-16), ★ The highest-risk finding: every `Marshal*JSON` helper is SHARED with the CLI `--json` path, Where the markdown renderers live: `internal/query`, not `internal/mcp`

### Community 717 - "Validation Architecture"
Cohesion: 0.33
Nodes (6): Phase Requirements → Test Map, Sampling Rate, ★ SURF-06 test-surface analysis (CONTEXT.md D-16 explicitly asks for this), Test Framework, Validation Architecture, Wave 0 Gaps

### Community 718 - "Common Pitfalls"
Cohesion: 0.33
Nodes (6): Common Pitfalls, Pitfall 1: Treating a TS-installed hook as foreign, Pitfall 2: In-place edit instead of strip-then-append, Pitfall 3: Hand-joining `.git/hooks` instead of `git rev-parse --git-path hooks`, Pitfall 4: chmod failure treated as fatal, Pitfall 5: Blocking git operations on a slow/hung `codegraph sync`

### Community 719 - "06-02-PLAN.md"
Cohesion: 0.33
Nodes (5): CLI seam points to modify, Integration harness (real non-TTY subprocess), Plain renderers to preserve byte-for-byte (D-02 — DO NOT MODIFY), STRIDE Threat Register, Trust Boundaries

### Community 720 - "Validation Architecture"
Cohesion: 0.33
Nodes (6): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps, What CAN be proven before a real release, and what CANNOT (the honest split)

### Community 721 - "2026-07-31-rebless-perf-baseline-on-ubuntu-latest.md"
Cohesion: 0.33
Nodes (5): Deviation from step 4, stated not buried, Problem, Resolution (2026-07-31) — DONE. Gate is green on main., Solution, Still open, deliberately

### Community 722 - "Security Policy"
Cohesion: 0.33
Nodes (6): Reporting a vulnerability, Scope, Security Policy, Supported versions, Threat model, Verifying what you run

### Community 723 - "recoverAccount"
Cohesion: 0.40
Nodes (4): recoverAccount(), T, TestAccountRecovery(), validateRecovery()

### Community 724 - "loadCharmClosure"
Cohesion: 0.70
Nodes (4): goListPackage, T, loadCharmClosure(), TestCharmCgoClosure()

### Community 725 - "[0.2.0](https://github.com/seanb4t/codegraph-go/compare/v0.1.0...v0.2.0) (2026-08-01)"
Cohesion: 0.40
Nodes (4): [0.2.0](https://github.com/seanb4t/codegraph-go/compare/v0.1.0...v0.2.0) (2026-08-01), Bug Fixes, Changelog, Features

### Community 726 - "⚠️ Wrong template — please pick the one for your PR type"
Cohesion: 0.40
Nodes (4): Exempt PRs, Issues are approved before PRs, Not sure which?, ⚠️ Wrong template — please pick the one for your PR type

### Community 727 - "notice_test.go"
Cohesion: 0.60
Nodes (4): T, TestMismatchNilReceiverSafety(), TestMismatchNoticeVerbatim(), TestMismatchWarningVerbatim()

### Community 728 - "Common Pitfalls"
Cohesion: 0.40
Nodes (5): Common Pitfalls, Pitfall 1: Assuming EXPL-02 is achievable as "lexical match, then RWR rerank", Pitfall 2: Edge-kind gap silently degrading RWR quality, Pitfall 3: Under-capturing new fixtures with the existing `capture.sh` defaults, Pitfall 4: Two-line node header rendered as one string

### Community 729 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 730 - "Common Pitfalls"
Cohesion: 0.40
Nodes (5): Common Pitfalls, Pitfall 1: Prepending the notice breaks the existing JSON-only contract for 5 of 7 MCP tools — ★ RESOLVED by SURF-06, retained as the reasoning trail, Pitfall 2: dbSizeBytes tripping the shared `findVolatileKeys` self-check, Pitfall 3: `resolveStartPath`'s CLI `--path` value may be relative, Pitfall 4: TS's `Journal:`/`indexState` warning branches have no Go equivalent — don't try to port them

### Community 731 - "Phase 04: Code Review Report"
Cohesion: 0.40
Nodes (4): IN-02: `diagWriterMu` only guards the `diagWriter` variable, not concurrent `Write` calls on the writer it points to, Info, Phase 04: Code Review Report, Summary

### Community 732 - "05-05-PLAN.md"
Cohesion: 0.40
Nodes (4): Artifacts this phase produces, Flagged Assumptions (surfaced, not silently decided), STRIDE Threat Register, Trust Boundaries

### Community 733 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 734 - "Code Examples"
Cohesion: 0.40
Nodes (5): Checkbox `ItemDelegate` — see Architecture Patterns #2 above (full example inline there)., Code Examples, PPID reparent watchdog (POSIX), Registry self-heal `List()` — see Architecture Patterns #3 above (full example inline there)., Windows parent liveness (build-tag sibling, not a reparent check — Windows has no `getppid` reparent semantics)

### Community 735 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 736 - "Code Examples"
Cohesion: 0.40
Nodes (5): Code Examples, Re-running the head-to-head bench (REL-03 — no new code), TS `affected`'s full command registration (SURF-04 ground truth, verified), TS `files --filter`/`--pattern` (SURF-02 ground truth), TS `impact` depth clamp (confirms SURF-01's target default + confirms D-02's deliberate max-clamp divergence)

### Community 737 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 738 - "09-02-PLAN.md"
Cohesion: 0.40
Nodes (4): Artifacts this plan produces, Identity-model delta (assumption-delta detector: `detected: true`, three `pluralization` signals), STRIDE Threat Register, Trust Boundaries

### Community 739 - "09-03-PLAN.md"
Cohesion: 0.40
Nodes (4): Artifacts this plan produces, D-08 file location: dedicated workflow instead of a job inside `ci.yml`, STRIDE Threat Register, Trust Boundaries

### Community 740 - "Common Pitfalls"
Cohesion: 0.40
Nodes (5): Common Pitfalls, Pitfall 1: PR-title lint never re-fires on a title edit, Pitfall 2: The GitHub App's own installation permissions are a different scope than the workflow `permissions:` block, Pitfall 3: `release.yml`'s header comment asserts a stronger guarantee than what D-03 would leave true, Pitfall 4: A disposable test tag resolving as `codegraph upgrade`'s "latest"

### Community 741 - "API Coverage — Phase 9 (release-please + GoReleaser)"
Cohesion: 0.40
Nodes (4): API Coverage — Phase 9 (release-please + GoReleaser), Evidence, Status, Why the detector fired

### Community 742 - "Validation Architecture"
Cohesion: 0.40
Nodes (5): Phase Requirements → Test Map, Sampling Rate, Test Framework, Validation Architecture, Wave 0 Gaps

### Community 743 - "API Coverage — Phase 10 (local build, CONTRIBUTING, Taskfile.yml)"
Cohesion: 0.40
Nodes (4): API Coverage — Phase 10 (local build, CONTRIBUTING, Taskfile.yml), Status, What this phase actually changed, Why the build-time dependencies are not an integrated API

### Community 744 - "260731-0a0-PLAN.md"
Cohesion: 0.40
Nodes (4): Reachability finding: treated as real, not dismissed, STRIDE Threat Register, Trust Boundaries, Version selection: v1.82.1, not v1.83.0

### Community 745 - "release-please-config.json"
Cohesion: 0.40
Nodes (4): include-component-in-tag, packages, release-type, $schema

### Community 746 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 747 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 748 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence — direct extraction from live TS 1.3.1 install), Secondary (repo-verified, current codebase state), Sources, Tertiary (LOW confidence)

### Community 749 - "02-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 750 - "02-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 751 - "02-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 752 - "02-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 753 - "02-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 754 - "02-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 755 - "02-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan's contribution), STRIDE Threat Register, Trust Boundaries

### Community 756 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 757 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 758 - "Code Examples"
Cohesion: 0.50
Nodes (4): Code Examples, TS CLI's own `--json` mode never calls `warn()` — the precedent for gating the CLI notice line, Verified glyph bytes (hexdump, this session), Verified TS worktree detection cascade (verbatim source read this session)

### Community 759 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 760 - "03-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register (ASVS L1, block-on: high), Trust Boundaries

### Community 761 - "03-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register (ASVS L1, block-on: high), Trust Boundaries

### Community 762 - "03-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register (ASVS L1, block-on: high), Trust Boundaries

### Community 763 - "03-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register (ASVS L1, block-on: high), Trust Boundaries

### Community 764 - "03-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register (ASVS L1, block-on: high), Trust Boundaries

### Community 765 - "Phase 3: Code Review Report (Round 6 — FINAL re-review, iteration 3 of --auto)"
Cohesion: 0.50
Nodes (3): Narrative Findings (AI reviewer), Phase 3: Code Review Report (Round 6 — FINAL re-review, iteration 3 of --auto), Summary

### Community 766 - "04-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 04-01 scope), STRIDE Threat Register (ASVS L1; block on high), Trust Boundaries

### Community 767 - "04-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 04-02 scope), STRIDE Threat Register (ASVS L1; block on high), Trust Boundaries

### Community 768 - "04-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (Plan 04-03 scope), STRIDE Threat Register (ASVS L1; block on high), Trust Boundaries

### Community 769 - "05-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces, STRIDE Threat Register, Trust Boundaries

### Community 770 - "05-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces, STRIDE Threat Register, Trust Boundaries

### Community 771 - "05-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces, STRIDE Threat Register, Trust Boundaries

### Community 772 - "05-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces, STRIDE Threat Register, Trust Boundaries

### Community 773 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 774 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 775 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 776 - "Archtest precedents to mirror (mechanism, not content)"
Cohesion: 0.50
Nodes (3): Archtest precedents to mirror (mechanism, not content), STRIDE Threat Register, Trust Boundaries

### Community 777 - "CLI seam points to modify (long-running op + summary)"
Cohesion: 0.50
Nodes (3): CLI seam points to modify (long-running op + summary), STRIDE Threat Register, Trust Boundaries

### Community 778 - "07-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 779 - "07-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 780 - "07-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 781 - "07-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 782 - "07-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 783 - "07-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 784 - "07-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 785 - "07-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 786 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 787 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 788 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 789 - "08-01-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 790 - "08-02-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 791 - "08-03-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 792 - "08-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 793 - "08-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 794 - "08-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 795 - "08-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 796 - "08-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 797 - "08-09-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this phase produces (this plan), STRIDE Threat Register, Trust Boundaries

### Community 798 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core (existing, reused), Standard Stack, SURF-03 full per-command flag map (verified against live TS 1.3.1 `--help` + current Go `cmd.Flags()` calls)

### Community 799 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 800 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence — read/run directly this session), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 801 - "09-04-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 802 - "09-05-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 803 - "09-06-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 804 - "09-07-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 805 - "09-08-PLAN.md"
Cohesion: 0.50
Nodes (3): Artifacts this plan produces, STRIDE Threat Register, Trust Boundaries

### Community 806 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 807 - "User Constraints (from CONTEXT.md)"
Cohesion: 0.50
Nodes (4): Claude's Discretion, Deferred Ideas (OUT OF SCOPE), Locked Decisions, User Constraints (from CONTEXT.md)

### Community 808 - "Code Examples"
Cohesion: 0.50
Nodes (4): Code Examples, Full `release-please.yml` (illustrative composite of Patterns 1, 2, 5), `release-please-config.json`, `.release-please-manifest.json`

### Community 809 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 810 - "Standard Stack"
Cohesion: 0.50
Nodes (4): Alternatives Considered, Core, Standard Stack, Supporting

### Community 811 - "Sources"
Cohesion: 0.50
Nodes (4): Primary (HIGH confidence), Secondary (MEDIUM confidence), Sources, Tertiary (LOW confidence)

### Community 812 - "260728-rfx-PLAN.md"
Cohesion: 0.50
Nodes (3): Phase 8 heading: KEEP "& Signed v1.0.0 Release" (do not retitle), STRIDE Threat Register, Trust Boundaries

### Community 813 - "2026-07-14-document-release-cut-procedures-runbook.md"
Cohesion: 0.50
Nodes (3): Problem, Resolution (2026-07-28, Phase 9), Solution

### Community 814 - "2026-07-31-bisect-indexer-throughput-regression.md"
Cohesion: 0.50
Nodes (3): Problem, Resolution (2026-07-31) — REFUTED. Do not run the bisect., Solution

### Community 815 - "pr_template_policy.py"
Cohesion: 1.00
Nodes (3): emit(), fail(), main()

### Community 816 - "synthetic-parity corpus"
Cohesion: 0.50
Nodes (3): D-03 case map, synthetic-parity corpus, Verifying the corpus mechanically (no algorithm required)

### Community 838 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 839 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 840 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 841 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 842 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 843 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 851 - "Security Domain"
Cohesion: 0.67
Nodes (3): Applicable ASVS Categories, Known Threat Patterns for this stack, Security Domain

### Community 852 - "Code Examples"
Cohesion: 0.67
Nodes (3): Code Examples, The `check:cross` target replacing `release-please.yml`'s inline sweep (D-15), The isolated tool module header comment (D-03's rationale, made durable)

## Knowledge Gaps
- **5014 isolated node(s):** `github.com/seanb4t/codegraph-go`, `Location`, `pebbleStore`, `Reader`, `pendingFile` (+5009 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **80 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `WriteFile()` connect `walkSpringRoutes` to `fakeHome`, `readLock`, `main.go`, `execCmd`, `main.go`, `Run`, `syntheticParityEngine`, `swap_test.go`, `init`, `BuildServer`, `BuildTSIndex`, `shared_test.go`, `golden_parity_test.go`, `OpenAt`, `targetRoot`, `.Run`, `Sync`, `Discover`, `traverse_test.go`, `fsatomic_test.go`, `TestPublishReleaseStepBranches`, `filesStatusFixture`, `render_markdown.go`, `resolveLatestVersion`, `githooks.go`, `setupIndexedFixtureWithTest`, `migrate_test.go`, `resolution_test.go`, `resolution_test.go`, `resolution_test.go`, `TestMigrateCmdRefusesUnrecognizedTargetWithoutForce`, `Generate`, `copyFixture`, `release_workflow_shape_test.go`, `resolution_test.go`, `resolution_test.go`, `.Install`, `initRepo`, `atomicSwap`, `initGitRepo`, `searchFakeNodeIterator`, `fileindex_test.go`?**
  _High betweenness centrality (0.037) - this node is a cross-community bridge._
- **Why does `Node` connect `Parser` to `Extract`, `Node`, `SynthesizeImplements`, `File`, `New`, `Extract`, `Extract`, `detectRoutes`, `extractor`, `translate_test.go`, `pebble_store.go`, `NodeID`, `stringValue`, `extractor`, `.Explore`, `extractor`, `traverse.go`, `traverse_test.go`, `keys.go`, `extractor`, `render_markdown.go`, `node.go`, `extractor`, `resolution_test.go`, `resolution_test.go`, `resolution_test.go`, `runAndSnapshot`, `search.go`, `CGoParser`, `resolution_test.go`, `resolveRefsWithIndex`, `extractor`, `Pointer`, `search_test.go`, `discover.go`, `pkga.go`?**
  _High betweenness centrality (0.036) - this node is a cross-community bridge._
- **Why does `execCmd()` connect `execCmd` to `execCmd`, `TestMigrateCmdRefusesUnrecognizedTargetWithoutForce`, `TestProgressCLINonTTYReachability`, `telemetry_test.go`, `version_test.go`, `initGitRepo`, `copyFixture`, `daemon_test.go`, `statusWorktreeMismatchFixture`, `setupIndexedFixtureWithTest`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Are the 98 inferred relationships involving `WriteFile()` (e.g. with `.Install()` and `.touchPending()`) actually correct?**
  _`WriteFile()` has 98 INFERRED edges - model-reasoned connections that need verification._
- **Are the 90 inferred relationships involving `NodeID()` (e.g. with `.collectInlineMethods()` and `.emitFunction()`) actually correct?**
  _`NodeID()` has 90 INFERRED edges - model-reasoned connections that need verification._
- **Are the 85 inferred relationships involving `execCmd()` (e.g. with `TestAffectedQuietSkipsControlCharacterPaths()` and `setupIndexedFixtureWithTest()`) actually correct?**
  _`execCmd()` has 85 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/seanb4t/codegraph-go`, `Location`, `pebbleStore` to the rest of the system?**
  _5014 weakly-connected nodes found - possible documentation gaps or missing edges._