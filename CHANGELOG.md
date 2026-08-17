# Changelog

## [0.11.0](https://github.com/seanb4t/codegraph-go/compare/v0.10.0...v0.11.0) (2026-08-17)


### Features

* v0.11.0 — Standalone Project Identity ([#63](https://github.com/seanb4t/codegraph-go/issues/63)) ([3b981cf](https://github.com/seanb4t/codegraph-go/commit/3b981cf5283fc85579b866b90423d6e0d59b3869))

## [0.10.0](https://github.com/seanb4t/codegraph-go/compare/v0.9.0...v0.10.0) (2026-08-13)


### Features

* **agents:** ship MCP resources, Claude Code skill onboarding, and self-describing instructions (v0.10.0) ([#60](https://github.com/seanb4t/codegraph-go/issues/60)) ([963cd93](https://github.com/seanb4t/codegraph-go/commit/963cd93067d3a8d87bcce5e899f1d4ec9947b2fc))

## [0.9.0](https://github.com/seanb4t/codegraph-go/compare/v0.8.0...v0.9.0) (2026-08-12)


### Features

* **upgrade:** step aside under Homebrew-managed installs ([#55](https://github.com/seanb4t/codegraph-go/issues/55)) ([59bc137](https://github.com/seanb4t/codegraph-go/commit/59bc137e9be49c4d361a674ba83a088355b551ff))

## [0.8.0](https://github.com/seanb4t/codegraph-go/compare/v0.7.0...v0.8.0) (2026-08-10)


### Features

* **homebrew:** Homebrew tap & cask — install completions, man pages, and a fail-loud install gate ([#51](https://github.com/seanb4t/codegraph-go/issues/51)) ([296c5a6](https://github.com/seanb4t/codegraph-go/commit/296c5a6f9a94893d3c721dccdc57125c39928c0d))

## [0.7.0](https://github.com/seanb4t/codegraph-go/compare/v0.6.0...v0.7.0) (2026-08-09)


### Features

* **macos:** Developer ID signing and Apple notarization for darwin release assets ([#47](https://github.com/seanb4t/codegraph-go/issues/47)) ([de551be](https://github.com/seanb4t/codegraph-go/commit/de551beb776ee142bc5d2886a5750cf05ee41241))


### Bug Fixes

* **test:** join the subprocess before reading captured stderr in wireoracle ([#50](https://github.com/seanb4t/codegraph-go/issues/50)) ([29f6e3d](https://github.com/seanb4t/codegraph-go/commit/29f6e3d5e9e083b529d132aeaff8cf11d5a6b8b7))

## [0.6.0](https://github.com/seanb4t/codegraph-go/compare/v0.5.1...v0.6.0) (2026-08-09)


### ⚠ BREAKING CHANGES

* **mcp:** register all eight tools by default; CODEGRAPH_MCP_TOOLS now narrows ([#44](https://github.com/seanb4t/codegraph-go/issues/44))

### Features

* **mcp:** register all eight tools by default; CODEGRAPH_MCP_TOOLS now narrows ([#44](https://github.com/seanb4t/codegraph-go/issues/44)) ([4397d4b](https://github.com/seanb4t/codegraph-go/commit/4397d4b5cf589056271ae64b27720e6efa60816f))

## [0.5.1](https://github.com/seanb4t/codegraph-go/compare/v0.5.0...v0.5.1) (2026-08-08)


### Bug Fixes

* **release:** resolve goreleaser pin without a Task-eaten Go template ([#39](https://github.com/seanb4t/codegraph-go/issues/39)) ([edd2460](https://github.com/seanb4t/codegraph-go/commit/edd2460f912c4810894b1fc3b2559d839d76a359))

## [0.5.0](https://github.com/seanb4t/codegraph-go/compare/v0.4.0...v0.5.0) (2026-08-08)


### Features

* **release:** migrate to single-runner goreleaser release with zig cross-compilation ([#35](https://github.com/seanb4t/codegraph-go/issues/35)) ([ff7c357](https://github.com/seanb4t/codegraph-go/commit/ff7c357e10371f6b85281c4487137db33ce2f286))

## [0.4.0](https://github.com/seanb4t/codegraph-go/compare/v0.3.0...v0.4.0) (2026-08-07)


### ⚠ BREAKING CHANGES

* **platform:** drop native Windows support — WSL2 only ([#29](https://github.com/seanb4t/codegraph-go/issues/29))

### Features

* **platform:** drop native Windows support — WSL2 only ([#29](https://github.com/seanb4t/codegraph-go/issues/29)) ([80d2e0a](https://github.com/seanb4t/codegraph-go/commit/80d2e0a455f937eeb93d1d3f450263b6a88f23e4))

## [0.3.0](https://github.com/seanb4t/codegraph-go/compare/v0.2.0...v0.3.0) (2026-08-07)


### Features

* **mcp:** protocol currency — official go-sdk v1.7.0 and 2026-07-28 spec compliance ([#24](https://github.com/seanb4t/codegraph-go/issues/24)) ([13f2875](https://github.com/seanb4t/codegraph-go/commit/13f2875de1332bdb45666bc476b395c224677b1f))

## [0.2.0](https://github.com/seanb4t/codegraph-go/compare/v0.1.0...v0.2.0) (2026-08-01)


### Features

* **01-07:** implement gather channel 3 (H5) + H6 max-score-wins merge ([9715bc6](https://github.com/seanb4t/codegraph-go/commit/9715bc616690b928ea34871f9ce559a369138518))
* **01-07:** implement gather channels 1-2 (H3/H4) + shared isTestFile ([c2685ee](https://github.com/seanb4t/codegraph-go/commit/c2685eebdb2c952f1aa96c46d08a6e49cf5e1d51))
* **01-08:** C# Pass-1 emit for references/instantiates/type_of/returns ([f49ca91](https://github.com/seanb4t/codegraph-go/commit/f49ca91c89f180c98ca1fef609d5384b97926009))
* **01-08:** Java Pass-1 emit for references/instantiates/type_of/returns ([e163b81](https://github.com/seanb4t/codegraph-go/commit/e163b81ab3db60ef41a8e4e387a79815bb0f1a32))
* **01-09:** Python Pass-1 emit for references/instantiates/type_of/returns ([94c7547](https://github.com/seanb4t/codegraph-go/commit/94c75473ff9f522a45156d576d5f92f6c03e2a4e))
* **01-09:** TS/JS Pass-1 emit for references/instantiates/type_of/returns ([5e9590d](https://github.com/seanb4t/codegraph-go/commit/5e9590dfaa618d760ed829be1629992848c5b103))
* **01-10:** implement H7 test-file dampening + H8 core-directory boost ([be67068](https://github.com/seanb4t/codegraph-go/commit/be67068efe2b9937f9861e85248061666231a870))
* **01-10:** implement H9 multi-term co-occurrence rerank + distinctive-identifier exemption ([ea5f96d](https://github.com/seanb4t/codegraph-go/commit/ea5f96da1eee9c29a6713605d8440f03896cd6bb))
* **01-11:** implement H10 type-hierarchy expansion + H11 BFS bounds ([a9da595](https://github.com/seanb4t/codegraph-go/commit/a9da59568bb983043b724a9ca9d3bc5bffa96123))
* **01-11:** implement H12 glue-node injection with GLUE_NODE_CAP ([022b2dd](https://github.com/seanb4t/codegraph-go/commit/022b2ddf5f1020212922873ef7d3dc9333d3c603))
* **01-12:** implement H13 large-overload disambiguation tier ([9c0e4df](https://github.com/seanb4t/codegraph-go/commit/9c0e4df552d22dacc6717dd9dd25f92e58711679))
* **01-12:** implement H13 name resolution + small-overload seeding tier ([bf2e4ad](https://github.com/seanb4t/codegraph-go/commit/bf2e4adc4a1fc3ee8e96d02805363c1277875298))
* **01-13:** implement H14 per-file score tiers + fileGraphScore aggregation ([698132e](https://github.com/seanb4t/codegraph-go/commit/698132ee9465d227d3261335bc7462e88e05043b))
* **01-13:** implement H15 hard test exclusion + H16 change-surface buried-rescue ([90e2121](https://github.com/seanb4t/codegraph-go/commit/90e212157bedc4686ef3e1aafd2d8dba456a78e9))
* **01-14:** implement H17 5-way OR relevance gate + H19 central-file selection + H18 5-tier sort ([ed86161](https://github.com/seanb4t/codegraph-go/commit/ed8616181885529389298713eadf1462d3bb7ab1))
* **01-16:** accept variadic multi-word explore query (EXPL-01) ([9d8062b](https://github.com/seanb4t/codegraph-go/commit/9d8062b1a4bc73c040fa41684b82933e4204576a))
* **01-16:** add EXPL-04 no-covering-tests warning and H20 skeletonization ([d872f30](https://github.com/seanb4t/codegraph-go/commit/d872f30825aa361d7f88b67368b9c06c93ffb0e9))
* **01-16:** wire full TS-parity explore pipeline into Engine.Explore ([475503e](https://github.com/seanb4t/codegraph-go/commit/475503e9344b2e343a36b578ffed40637939eeed))
* **02-01:** add verbatim notice/warning strings + CachingDetector ([4d08261](https://github.com/seanb4t/codegraph-go/commit/4d08261b74836182acd5dcf7b346ed436e446d89))
* **02-01:** implement git primitives + 4-gate mismatch cascade (GREEN) ([3322d0b](https://github.com/seanb4t/codegraph-go/commit/3322d0bcaffbf43e20401ae9286ac264502d5db7))
* **02-02:** implement DbSizeBytes walk + FilesByLanguage aggregation (GREEN) ([f93403d](https://github.com/seanb4t/codegraph-go/commit/f93403db82670b06b58596d7cb1a80deae6652ae))
* **02-03:** implement RenderFilesMarkdown, flat + tree union branches (GREEN) ([a2d3492](https://github.com/seanb4t/codegraph-go/commit/a2d34922b50953776f15b86903e4725486a989a1))
* **02-03:** implement renderLocationTable + 4 Location-backed renderers (GREEN) ([b178757](https://github.com/seanb4t/codegraph-go/commit/b17875716f54f93e47b10e97aa0406de1a8b2524))
* **02-04:** live StatusResult.WorktreeMismatch + notice/blockquote helpers ([1dd5c39](https://github.com/seanb4t/codegraph-go/commit/1dd5c39281628a61c129d731d9392e43296d6768))
* **02-04:** retain Engine.startPath, add WorktreeMismatch()/UseDetector ([eb9528b](https://github.com/seanb4t/codegraph-go/commit/eb9528bda78eebff2fd908c46c6a44bb4f608a81))
* **02-05:** implement RenderStatusText + RenderStatusMarkdown status renderers ([26b74ae](https://github.com/seanb4t/codegraph-go/commit/26b74ae725cbb1a513fb5b376c6cdce5cde83da6))
* **02-06:** GREEN — server-scoped CachingDetector + worktree notice on 7 tools (WORK-02, D-12, D-13) ([13892fd](https://github.com/seanb4t/codegraph-go/commit/13892fdeb2740190dcf6f6e90a42e908c285d48c))
* **02-06:** GREEN — swap six MCP call sites to markdown (SURF-06, D-17) ([d6204cd](https://github.com/seanb4t/codegraph-go/commit/d6204cda647b9aba5eea7c5604e7b26ec8e3796e))
* **02-07:** GREEN — replace terse status one-liner with D-09's sectioned layout ([3fd3f38](https://github.com/seanb4t/codegraph-go/commit/3fd3f38574945c0d7c30739ac2dc0ec8c3fb0efc))
* **03-01:** implement internal/watch/policy.go (WATCH-03) ([44f83c3](https://github.com/seanb4t/codegraph-go/commit/44f83c3b78e228ab079213b1caeccac2b010f1a5))
* **03-02:** policy-gate-first Run + Probe/WithProbe + RunWithRetry/jitter ([00054b5](https://github.com/seanb4t/codegraph-go/commit/00054b5d37df0912333366dc7001b5c9f5dcdec4))
* **03-03:** flip serve --mcp watcher to default-on, defer all startup off handshake path ([75d6945](https://github.com/seanb4t/codegraph-go/commit/75d6945f2d6c251cf9923033880817ccbcd2164e))
* **04-01:** implement quietLogger + diagWriter seam ([172eed5](https://github.com/seanb4t/codegraph-go/commit/172eed577013c32261b7d8ea081fd3d53a6184e2))
* **04-01:** inject quietLogger at the pebble.Open seam ([235e97a](https://github.com/seanb4t/codegraph-go/commit/235e97aa00ae1f88e7a907fb84d99049a21886e2))
* **04-02:** implement stdout guard detection predicates (GREEN) ([2d1559f](https://github.com/seanb4t/codegraph-go/commit/2d1559f3a0581fff60d94cf16fc6767f62c6a386))
* **04-02:** scan six serve-reachable packages for stdout writes (HYG-02) ([9978655](https://github.com/seanb4t/codegraph-go/commit/9978655525c43fe4bbe3ae3c5da4fa854a4bca42))
* **05-01:** add internal/fsatomic.WriteFile ([3769e70](https://github.com/seanb4t/codegraph-go/commit/3769e707a61afd854730386382b7f16755032d91))
* **05-02:** add gitmeta.HooksDir resolution ([29d7c12](https://github.com/seanb4t/codegraph-go/commit/29d7c120f9b419096daa88434813b977c6ab02b0))
* **05-02:** add gitmeta.IsGitRepo probe ([adee75f](https://github.com/seanb4t/codegraph-go/commit/adee75fc914b5de52d2292455aacd26bc8a32270))
* **05-03:** add githooks marker constants, markerBlock, strip/empty primitives ([ff81c6b](https://github.com/seanb4t/codegraph-go/commit/ff81c6b14100ede3f2738276ba3260406be2e56e))
* **05-03:** add githooks.Install with result types ([763097d](https://github.com/seanb4t/codegraph-go/commit/763097dfac0b98f4ea7c3aba02876b428b721f62))
* **05-03:** add githooks.Remove and Status ([290b1a9](https://github.com/seanb4t/codegraph-go/commit/290b1a9e468b160d53b9876d5f80055728df016b))
* **05-04:** add codegraph githooks install/remove/status CLI command tree ([3eebfda](https://github.com/seanb4t/codegraph-go/commit/3eebfdaacefe603f5bd575bf72785cd3c5b9c545))
* **05-05:** wire D-06 best-effort hook cleanup into uninit ([aee7da1](https://github.com/seanb4t/codegraph-go/commit/aee7da166ceab1d588160793e6dcdba69ebb3509))
* **05-05:** wire D-07 watcher-fallback advisory into init success path ([be7864a](https://github.com/seanb4t/codegraph-go/commit/be7864a052a275ceff75b6d620ffce0143eb3592))
* **05:** IN-03 report exec-bit health as a distinct githooks status state ([5bc6348](https://github.com/seanb4t/codegraph-go/commit/5bc6348ca49315699d28c6a7abb8b68bfd767d72))
* **06-01:** add charm.land/lipgloss/v2 + present skeleton — archtest goes GREEN ([f5f399b](https://github.com/seanb4t/codegraph-go/commit/f5f399b5d5b70b92ca8cac51ef5e3118bb94c061))
* **06-01:** ChoosePresentation pure selector (D-04/D-05) ([97ff459](https://github.com/seanb4t/codegraph-go/commit/97ff459f3b7453b858814f8af4ffdaf82ff0bf69))
* **06-02:** present.RenderFiles — styled tree + flat (D-01/D-02) ([bf4c42f](https://github.com/seanb4t/codegraph-go/commit/bf4c42ff09918c4118eb15f5ef450864d2deca5d))
* **06-02:** present.RenderStatus — styled StatusResult (D-01/D-02) ([1c0f6a6](https://github.com/seanb4t/codegraph-go/commit/1c0f6a6e7491a3f74f905688e8d00307b2c64642))
* **06-02:** wire isTTY branch into status/files RunE (D-03/D-04/D-06) ([3e792f7](https://github.com/seanb4t/codegraph-go/commit/3e792f7ae6119060272206f4c9531ae96dcbec82))
* **06-03:** hand-rolled stderr progress writer (GREEN) ([5016d5c](https://github.com/seanb4t/codegraph-go/commit/5016d5c014a382e5be4ec7f9eed19889d0ec7ac5))
* **06-03:** wire TTY-gated progress into init/index/sync (D-07) ([b2aff19](https://github.com/seanb4t/codegraph-go/commit/b2aff19defb9d7fbddfe5c1e9e6835a16ff6d1e6))
* **07-01:** add charm.land/bubbletea/v2 + bubbles/v2 deps ([d83761e](https://github.com/seanb4t/codegraph-go/commit/d83761ee8c96d8e59f65b61349c5e038a424f3da))
* **07-01:** implement InteractiveAllowed dual-TTY gate ([845b0a0](https://github.com/seanb4t/codegraph-go/commit/845b0a08dfa5b961a4a0951a6257c46eb7d7aa2f))
* **07-02:** add daemon registry Record/Register/Deregister via fsatomic ([08e0947](https://github.com/seanb4t/codegraph-go/commit/08e09478903e3b5f510c3a19419b7274ed6981be))
* **07-02:** add registry.List() with self-heal via lock.go isStale ([2a9ff5e](https://github.com/seanb4t/codegraph-go/commit/2a9ff5e1a630b33039102e9e50e1cd7f143920fd))
* **07-03:** implement PPID reparent watchdog (POSIX) with joinable stop (GREEN) ([a6c683c](https://github.com/seanb4t/codegraph-go/commit/a6c683c26321aaedf4d529fa7a576c4460d5a41c))
* **07-03:** Windows parent-liveness watchdog sibling + x/sys direct + CI vet gate ([fddc7af](https://github.com/seanb4t/codegraph-go/commit/fddc7afe68637072f725a05b8be0baf283b187ff))
* **07-04:** implement sendStop platform split (GREEN) ([4deab40](https://github.com/seanb4t/codegraph-go/commit/4deab402cd530f969afb4462e24f8423eda69370))
* **07-04:** implement StopMatching/StopAll orchestration (GREEN) ([dc2e91e](https://github.com/seanb4t/codegraph-go/commit/dc2e91ea71852f1dca98dc7cca1f5e2eb229846e))
* **07-05:** register daemon in global registry on Run start, deregister on shutdown ([ad849c2](https://github.com/seanb4t/codegraph-go/commit/ad849c2466a375ce78bf2a0b3dbb32a4ae359298))
* **07-05:** start the PPID watchdog in Run and join it on every teardown path ([f026663](https://github.com/seanb4t/codegraph-go/commit/f026663d73ac2e9ecf16e4f836a3bc9acc05723e))
* **07-06:** implement agent checkbox picker Model (GREEN) ([06fece7](https://github.com/seanb4t/codegraph-go/commit/06fece7bfd4b924bfbb47a7d2265447ff8b0f5a8))
* **07-06:** wire -y/--yes and the bubbles picker into install/uninstall ([f893992](https://github.com/seanb4t/codegraph-go/commit/f89399225468b1c36c51cece9f3e028c4960f823))
* **07-07:** implement daemon picker Model (GREEN) ([4154877](https://github.com/seanb4t/codegraph-go/commit/4154877910b9c427035adec931191d87a1e97cb4))
* **07-07:** restructure daemon.go into bare picker/list + start + stop tree ([979e285](https://github.com/seanb4t/codegraph-go/commit/979e285d3561c639e32c5a6833b8f3ecb0363790))
* **08-01:** engine defaultDepth 5-&gt;2 — impact matches TS 1.3.1 (SURF-01, D-02) ([6fae70f](https://github.com/seanb4t/codegraph-go/commit/6fae70f1a98b4c6f213569305324e8d10ffe6c01))
* **08-01:** impact -d/-j short flags + updated depth help text (SURF-03) ([3c316e4](https://github.com/seanb4t/codegraph-go/commit/3c316e4a559136a500fe38dca0b0870b82b9bf1c))
* **08-02:** FilesOptions.Dir prefix filter in the engine (SURF-02) ([178cfbd](https://github.com/seanb4t/codegraph-go/commit/178cfbd097607cb6d599914caf5fc25704a929f4))
* **08-02:** wire files --dir and -j/--json to FilesOptions (SURF-02/03) ([80c062d](https://github.com/seanb4t/codegraph-go/commit/80c062d992500929f13789c184dfbe8505e55403))
* **08-03:** add TS-parity short flags to status/query/callers/callees/install/uninstall ([e1806ed](https://github.com/seanb4t/codegraph-go/commit/e1806ed389f5cd8047652624f40e32a9a07743ca))
* **08-03:** add upgrade --force/-f flag with verification-safe same-version guard ([ca58976](https://github.com/seanb4t/codegraph-go/commit/ca58976d2d2e2e77040a2d56f58163035d638c38))
* **08-04:** add defaultAffectedDepth/clampAffectedDepth (SURF-04) ([872d168](https://github.com/seanb4t/codegraph-go/commit/872d1684d155320f3dc51541a4ab9015164e9749))
* **08-04:** Affected as depth-bounded BFS with test-leaf pruning (SURF-04) ([da5adc4](https://github.com/seanb4t/codegraph-go/commit/da5adc45fdf16e91ffc3acc35bd3cf81f33b99de))
* **08-05:** wire affected scripting flags (--stdin/-d/-f/-q/-j) ([2b1c5e1](https://github.com/seanb4t/codegraph-go/commit/2b1c5e139f9ccc5ba3f314ddf29c067ee6e4b755))
* **09-01:** blocking 6-target pre-tag gate in release-please.yml ([ce403dc](https://github.com/seanb4t/codegraph-go/commit/ce403dcf4fb53e04eb402f50b531c12395af6895))
* **09-01:** release-please spine + non-vacuous workflow-shape drift guard ([7f60822](https://github.com/seanb4t/codegraph-go/commit/7f60822bb5f3375b1dbfa4466af32c68ffe7330e))
* **09-02:** make release.yml publish step idempotent (D-04) ([122d80c](https://github.com/seanb4t/codegraph-go/commit/122d80c144001f35033800f180297ff9f9989bea))
* **09-03:** add actionlint static gate job to ci.yml ([04cc0ec](https://github.com/seanb4t/codegraph-go/commit/04cc0ec4a5d9aced820e59fa111108093732bed4))
* **09-03:** add PR-title conventional-commit lint workflow (D-08) ([4c2cc37](https://github.com/seanb4t/codegraph-go/commit/4c2cc37ea7fc8bc5aec89b5d45732355e72ab39f))
* **bench:** median the regression gate over N independent sessions ([ebdc95d](https://github.com/seanb4t/codegraph-go/commit/ebdc95d5abcde21f984ba81abf9e869da49180ac))


### Bug Fixes

* **01-fix:** CR-01 stop overloaded overrides methods from collapsing ([b87b8cf](https://github.com/seanb4t/codegraph-go/commit/b87b8cf657e97c8ae8286d2cbffd8b13e2b9ddcf))
* **01-fix:** CR-02 wire NODE-03 file/line narrowing end-to-end ([e298e26](https://github.com/seanb4t/codegraph-go/commit/e298e2690ab5b2be8d6ee94d7576f0424ce07f05))
* **02-fix2:** BL-01 never cache a worktree-mismatch verdict computed under a cancelled context ([d5bf263](https://github.com/seanb4t/codegraph-go/commit/d5bf26357063e06526b5ddef635c5b30384ba823))
* **02-fix2:** GOLDEN-01 make CI run the golden suite explicitly (go test ./... never does) ([157319a](https://github.com/seanb4t/codegraph-go/commit/157319a7cb2e70ed74db61b50af603e1dca49ac9))
* **02-fix2:** WR-01 extract serveServerPaths so a test observes serve.go's real repoPath/startPath derivation ([ac3b16a](https://github.com/seanb4t/codegraph-go/commit/ac3b16a26de43f0f2f995738b1000e676bc367e5))
* **02-fix2:** WR-02 pin confinement anchor with startPath != repoPath (the shape production uses) ([0351901](https://github.com/seanb4t/codegraph-go/commit/035190182050645f19d75e748adc123815c279f8))
* **02-fix2:** WR-03 exclude .codegraph/ from golden fixture copies so a fixture measures only what it indexed ([8afd885](https://github.com/seanb4t/codegraph-go/commit/8afd8855066181c1857eebad1447adb603b56ac3))
* **02-fix2:** WR-04 align exploreHandler/companionHandler param order with BuildServer's own signature ([5359d41](https://github.com/seanb4t/codegraph-go/commit/5359d411fb7765463e81e6e4606546368baace9c))
* **02-fix:** CR-01 give MCP handlers the caller's start path, distinct from the confinement root ([dc6ddd5](https://github.com/seanb4t/codegraph-go/commit/dc6ddd5c912bb755ef3ae657e8c01da52b2196bf))
* **02-fix:** CR-02 measure the store buildEngineAt actually built for dbSizeBytes ([1cfd5b3](https://github.com/seanb4t/codegraph-go/commit/1cfd5b3265713fa72497724d1f60d4e394575f04))
* **02-fix:** WR-01 thread caller context through WorktreeMismatch/Status ([9cfbb4f](https://github.com/seanb4t/codegraph-go/commit/9cfbb4f86fc610de515813dfff6c6050e294dfe8))
* **02-fix:** WR-02 bound CachingDetector cache growth on client-controlled keys ([c09982d](https://github.com/seanb4t/codegraph-go/commit/c09982da6ad41234d28fedb3da4735d7fbb505c6))
* **02-fix:** WR-03 make gate 4 fail closed on a degraded git call ([ad51fd6](https://github.com/seanb4t/codegraph-go/commit/ad51fd6556eb48a657518d2fb7508b88afadadf9))
* **02-fix:** WR-04 wire the compact worktree notice into query and affected ([1dd8fb4](https://github.com/seanb4t/codegraph-go/commit/1dd8fb4daece953c8b4ac4f2fd6de8cf03869c43))
* **02-fix:** WR-05 print the worktree notice after explore/node succeed ([0410710](https://github.com/seanb4t/codegraph-go/commit/04107106ec0150d7286de92612bd2f9a46e3d4d5))
* **03:** CR-01 retry pebble lock collisions instead of failing tool calls, syncs, and serve startup ([c22ad98](https://github.com/seanb4t/codegraph-go/commit/c22ad98ea29e440e4a2bd77348437c0917e6e61a))
* **03:** CR-01+WR-01 classify lock-held at the Open seam via ErrStoreLocked sentinel with build-tagged windows arm ([7699e9c](https://github.com/seanb4t/codegraph-go/commit/7699e9cb6ad7dd62ef094e0604a622cd1edf22ed))
* **03:** IN-01 correct pinned-pebble provenance claim in unix lock-classifier test comment ([47170db](https://github.com/seanb4t/codegraph-go/commit/47170dbbd29908e6884057c0d70f495306fdcdeb))
* **03:** IN-01 document the bounded loopExited requeue window on Run's backstop Stop ([c94d394](https://github.com/seanb4t/codegraph-go/commit/c94d394ed650c8c9dc03cfb18a4312251796598a))
* **03:** IN-02 event-synchronize Open's lock-retry convergence test via an openLockRetrySleep seam ([292fb27](https://github.com/seanb4t/codegraph-go/commit/292fb27ec73595ccf6f9773f7aa1a557e40fdbdd))
* **03:** IN-02 gate serve's disabled print on errors.As, mirroring cli/daemon.go ([f454081](https://github.com/seanb4t/codegraph-go/commit/f4540812a96f96b4e5fde888a4263614a0a0ea53))
* **03:** IN-03 IN-04 IN-07 harden daemon.Run/Debouncer lifecycle seam ([aec7945](https://github.com/seanb4t/codegraph-go/commit/aec79453eae68d1e28e7c08d16d480b522af30db))
* **03:** IN-03 pin serve's full verbatim D-12 disabled message, banner included ([b4bee7e](https://github.com/seanb4t/codegraph-go/commit/b4bee7e8eae7ec547034ee45d510e71ae7b153a4))
* **03:** IN-05 carry the watch-disabled reason in a typed error instead of re-deriving it at three sites ([f0a4b70](https://github.com/seanb4t/codegraph-go/commit/f0a4b704656f3bb4b7e0aa06f729387f6cbcbf73))
* **03:** IN-06 cover codegraph daemon's policy-disabled branch with an in-process CLI test ([ce44fff](https://github.com/seanb4t/codegraph-go/commit/ce44fff86a3faa75e8179a7ba944cd3647e7cbb3))
* **03:** IN-08 make CI's go list failure visible to set -e ([e444082](https://github.com/seanb4t/codegraph-go/commit/e4440823e15ff456ccdd466e85d54a4e13ead2e2))
* **03:** IN-09 document hasIndex as a deliberate startup-time snapshot ([be8fa03](https://github.com/seanb4t/codegraph-go/commit/be8fa03147b0d2ec09731197562dc3d987b26755))
* **03:** IN-10 record Open's ~400ms lock-retry widening of the migrate probe race window ([4b1807d](https://github.com/seanb4t/codegraph-go/commit/4b1807d2b8ef72e3af20342c8dec76020ad66f45))
* **03:** WR-01 (round 3) run GOOS=windows vet on the lock classifier in CI ([3a4c2f6](https://github.com/seanb4t/codegraph-go/commit/3a4c2f6454bc65269ca20b772f92a30b7e4da241))
* **03:** WR-01 add tests for requeue give-up/reset, ErrWatcherClosed teardown, and post-cancel Add ([0a4daac](https://github.com/seanb4t/codegraph-go/commit/0a4daacdb22b95b7acdc44b0bb572f4f0bfc69a9))
* **03:** WR-01 guard the CR-01 test's onSyncStart close against straggler double-fires ([b194d3d](https://github.com/seanb4t/codegraph-go/commit/b194d3dd1106b0e56510145e29739693930855c1))
* **03:** WR-01 print friendly D-12 disabled message from codegraph daemon ([a527a71](https://github.com/seanb4t/codegraph-go/commit/a527a71f9b54fed247eabe8297361be2623340c7))
* **03:** WR-02 add direct unit tests for the lock classifier and Open retry loop ([1bdcd9c](https://github.com/seanb4t/codegraph-go/commit/1bdcd9c1d3644236526024d9e4f1dd6c8de765bf))
* **03:** WR-02 replace scheduling-race assertion with deterministic block-until-released seam test ([30b9839](https://github.com/seanb4t/codegraph-go/commit/30b983919843ca4ef62ba00ed634da0d6f2d20ec))
* **03:** WR-03 run the race detector on the concurrency packages in CI ([921ea1f](https://github.com/seanb4t/codegraph-go/commit/921ea1fa534577a222026a12e0afb8ebb64e70e6))
* **03:** WR-04 add live edit-to-explore auto-sync integration test (CR-01 regression anchor) ([1a1b07b](https://github.com/seanb4t/codegraph-go/commit/1a1b07bdd08bae1259a128d23be3f733918a287a))
* **04:** check scanner.Err after stdout purity scan loop ([8a732cc](https://github.com/seanb4t/codegraph-go/commit/8a732cc321f458aa167b0a8b9365530089b18f2c))
* **04:** CR-01 close HYG-02 stdout guard to serve-reachable import closure ([21e47b9](https://github.com/seanb4t/codegraph-go/commit/21e47b976c8b9905a2b419df8b8efd1f01bf0127))
* **04:** IN-01 document residual risk of indirect stdout writes in HYG-02 guard ([68e7a91](https://github.com/seanb4t/codegraph-go/commit/68e7a91abf454d58b24ee1b6b6443cf9e2354e39))
* **04:** WR-01 guard stderrBuf against data race in stdout-purity test ([c2fe509](https://github.com/seanb4t/codegraph-go/commit/c2fe5090cfc77b3012ebe69f4f0fa80fc0985435))
* **04:** WR-02 assert tools/call response is not a JSON-RPC error ([cbd134d](https://github.com/seanb4t/codegraph-go/commit/cbd134d058a699272be146ae9a02cbfa5e86fe2d))
* **04:** WR-03 guard diagWriter behind mutex accessor to remove data-race footgun ([9372385](https://github.com/seanb4t/codegraph-go/commit/93723854e9b03a3e738b81637bf81dd677acc031))
* **05:** CR-01 guard stripMarkerBlock against unterminated begin marker ([3be729f](https://github.com/seanb4t/codegraph-go/commit/3be729f10772f9b08536e34979a2d1b34cf91c31))
* **05:** CR-01 stop stripMarkerBlock false ok=true on nested/dangling markers and Install re-corrupting malformed hooks ([f514e57](https://github.com/seanb4t/codegraph-go/commit/f514e57aac08a3792823eb88335efd357b17389e))
* **05:** CR-02 skip unreadable existing hook file instead of overwriting it ([265cf67](https://github.com/seanb4t/codegraph-go/commit/265cf67437e28ff3e577a8fc30abeebd670de6d0))
* **05:** IN-01 de-duplicate reused WR-02 comment label ([dc5fa27](https://github.com/seanb4t/codegraph-go/commit/dc5fa27375c9f12407b029fbcb40f84ded7c514b))
* **05:** WR-01 accumulate error in Remove's malformed-marker branch ([a3118d3](https://github.com/seanb4t/codegraph-go/commit/a3118d32664e16ce7c0b003e8261868aebe8ad94))
* **05:** WR-01 accumulate per-hook write/delete errors instead of discarding ([c8ce880](https://github.com/seanb4t/codegraph-go/commit/c8ce880543655b09407a5b94d753534c10646a27))
* **05:** WR-01 make Status use exact-trimmed-line marker detection ([73aa510](https://github.com/seanb4t/codegraph-go/commit/73aa510b2a9a9e87d06b838f37214f22f959fc36))
* **05:** WR-01 surface RemoveResult.Errors as warnings during uninit hook cleanup ([f2af40a](https://github.com/seanb4t/codegraph-go/commit/f2af40a4c1cd28061518981325c1bf0df3343a8d))
* **05:** WR-02 accumulate Remove read errors instead of silently skipping ([b7c38ad](https://github.com/seanb4t/codegraph-go/commit/b7c38ad30685bed366135e66f99510cb40ba656e))
* **05:** WR-05+IN-04 make Remove report removal only when a strip actually occurred ([47b75a6](https://github.com/seanb4t/codegraph-go/commit/47b75a6b23514f93fced20c2b05788cce33a9c28))
* **06:** revert WR-02 signal handling — restore init/index/sync interruptibility (06-REVIEW iter-2 CR-01) ([e2470ea](https://github.com/seanb4t/codegraph-go/commit/e2470eaedb3d6b158c475bb45cab8cd28f6996fb))
* **06:** sanitize control chars in pretty file renderer (CR-01) ([ebaab25](https://github.com/seanb4t/codegraph-go/commit/ebaab25fc74588e84a84ead327b5dd64ff406a96))
* **06:** WR-01 sanitize control chars in status.go pretty renderer ([50bbeb9](https://github.com/seanb4t/codegraph-go/commit/50bbeb930637d686b1f219d8b8bdec6dff0d734d))
* **06:** WR-02 use goleak convention for present package goroutine-leak test ([c8b7fe9](https://github.com/seanb4t/codegraph-go/commit/c8b7fe92c17b62c6be0dc11656c33d076b7af95a))
* **06:** WR-04 de-duplicate spinner wiring across init/index/sync ([7aac3f0](https://github.com/seanb4t/codegraph-go/commit/7aac3f02fccb7fabd5f6140d4686b00635882a9f))
* **07:** G-07-1 don't open daemon picker for empty registry (leaked TTY escape probes) ([3e43b25](https://github.com/seanb4t/codegraph-go/commit/3e43b253f6bee535b3d10eaab13bad2991fac301))
* **07:** G-07-2 render pickers in alt-screen + reserve footer height (flicker/blank list on TTY) ([ad6e9cb](https://github.com/seanb4t/codegraph-go/commit/ad6e9cbdf055f87b8de7e4e6f5a52729176bf710))
* **07:** WR-01 surface stopped daemon records to the interactive picker ([e62f4d8](https://github.com/seanb4t/codegraph-go/commit/e62f4d8f09cf3f43ac917260536af2322fed01eb))
* **07:** WR-02 reject daemon stop --all --path as mutually exclusive ([3888a46](https://github.com/seanb4t/codegraph-go/commit/3888a46f38e109a7846197cdbde6d919dc132721))
* **07:** WR-03 normalize SortRecordsCurrentFirst via symlink resolution ([c7509fb](https://github.com/seanb4t/codegraph-go/commit/c7509fbb9f244792335fd9d79a2e648f69ef95ca))
* **08:** CR-01/WR-03/WR-06 harden affected --stdin input handling ([9019fe4](https://github.com/seanb4t/codegraph-go/commit/9019fe4f54af9e13c6af67646ad6e624d1af4f57))
* **08:** IN-01/02/03 correct inverted docs and off-by-one limit messages ([a30c730](https://github.com/seanb4t/codegraph-go/commit/a30c7307d61fc27f548baddcf3b706991f7e4050))
* **08:** WR-01 collectAffectedFiles calls query.ValidateAffectedFiles instead of reimplementing the bound ([331a5a6](https://github.com/seanb4t/codegraph-go/commit/331a5a6d49d89c28017d9a8bd3229e112b28e21c))
* **08:** WR-01 require a path-separator boundary in dirPrefixMatches ([ea2b889](https://github.com/seanb4t/codegraph-go/commit/ea2b8899117980ac3187f47057cb82f7bf4539ab))
* **08:** WR-02 apply sortLocations to Callees for cross-function determinism ([4feb6ff](https://github.com/seanb4t/codegraph-go/commit/4feb6ff6a87e0c6df6d797ed9f7a33c5fdcd4ed6))
* **08:** WR-02 normalize AffectedResult.Files nil to empty slice ([d887e38](https://github.com/seanb4t/codegraph-go/commit/d887e38dc61740614bc5dae30ca3396e6c7f82b7))
* **08:** WR-04 compose interface-dispatch traversal into Affected's BFS ([43c25ae](https://github.com/seanb4t/codegraph-go/commit/43c25ae51607b4e2e70ccaf3b404483a8ef6bd87))
* **08:** WR-05 sort Impact/Callers/Affected results deterministically ([d3f077c](https://github.com/seanb4t/codegraph-go/commit/d3f077cc966b2d53e31aa6f8814bf4ae9024b603))
* **08:** WR-07 tighten isTestSymbol to the _test.go suffix check only ([8a6ac3b](https://github.com/seanb4t/codegraph-go/commit/8a6ac3bc1af6d093fb6b2b447a1f914ae7dff6f7))
* **09:** gate the irreversible release checkpoints blocking-human ([#3](https://github.com/seanb4t/codegraph-go/issues/3)) ([90c32b8](https://github.com/seanb4t/codegraph-go/commit/90c32b80c024b3e74e7a3edd2f7064767393f2f5))
* **bench:** reject cross-platform baseline comparisons in CheckRegression ([0c4d550](https://github.com/seanb4t/codegraph-go/commit/0c4d550820f772270a7ebbcd57e2f4a06693e055))
* **daemon:** record the process's actual start time in locks and registry records ([326aba7](https://github.com/seanb4t/codegraph-go/commit/326aba78b40b690e328755a1cfdba3d7d773af18))
* **deps:** bump google.golang.org/grpc to v1.82.1 (GO-2026-6061) ([a0cfceb](https://github.com/seanb4t/codegraph-go/commit/a0cfcebc369c1ad6e1a88b9f3d21edbca81bbbef))
