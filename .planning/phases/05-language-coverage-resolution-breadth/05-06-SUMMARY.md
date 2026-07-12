---
phase: 05-language-coverage-resolution-breadth
plan: 06
subsystem: indexing
tags: [tree-sitter, python, multi-language, extraction, resolution, cross-file-resolution, golden-parity, dotted-module-path]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 01
    provides: LanguageSpec registry, ProjectDescriptor interface, cgo.NewPythonParser + tree-sitter-python@v0.25.0 (already pinned since Phase 1)
  - phase: 05-language-coverage-resolution-breadth
    plan: 02
    provides: extension->language registry walker, per-worker language-keyed parser cache, per-language Extract dispatch
  - phase: 05-language-coverage-resolution-breadth
    plan: 03
    provides: per-language ModuleKey-keyed symbolIndex (byModuleKeyAndName), addSymbol WR-01 collision handling, resolveSelector's alias-membership boundary
  - phase: 05-language-coverage-resolution-breadth
    plan: 04
    provides: javaextract as the closest structural analog (parse-time ModuleKey override pattern is explicitly NOT reused here — see key-decisions; PascalCase/camelCase call-qualifier heuristic and external-test-package resolution_test.go pattern ARE reused)
provides:
  - internal/indexer/pyextract package — full Python structural extraction (class/function/method, module-level assignments correctly emitting no node) into the shared goextract vocabulary
  - Python LanguageSpec registration (ID "python", .py extension, cgo.NewPythonParser (pre-existing since Phase 1), pyproject.toml/setup.py Descriptor, dotted-module-path ModuleKey computed entirely at discovery time — no parse-time override, unlike Java/C#)
  - Cross-file Python resolution proven end-to-end through the real indexer.Run pipeline — from-import calls, aliased-plain-import calls, and imported-base-class inheritance all resolve into real committed edges via the dotted-module-path ModuleKey
  - TestGoldenParity_Python — a self-skipping D-12 validation harness, smoke-tested against a real litellm corpus subtree (168 files) this session
affects: [05-07, dispatch synthesis (Wave 6 — extends/implements promotion consumes the RefKindEmbeds shape this plan emits; inherited-method-call resolution explicitly deferred here, see key-decisions), docs/LANGUAGE-CAPABILITY-MATRIX.md's D-11 Python row]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Discovery-time-only ModuleKey (no parse-time override): unlike Java's `package`/C#'s `namespace` (both declared IN the source, requiring javaextract/csharpextract to override FileResult.ImportPath after parsing), Python's dotted module path is entirely directory-structure-derived — languages_python.go's LanguageSpec.ModuleKey (descriptor, relPath) already has everything it needs (the repo's resolved package root + relPath) before any file content is read. pyextract.Extract passes moduleKey straight through to FileResult.ImportPath unchanged — the first priority-4 language to NOT need the parse-time-override pattern 05-04/05-05 established."
    - "Per-file-unique moduleKey (not per-directory-shared, unlike Go/Java/C#): a Python 'module' IS the file — two files never legitimately share a moduleKey the way two Go files in the same package or two Java files in the same package do. symbolIndex's existing map[moduleKey]map[symbolName]nodeID structure needed no change; only the semantic granularity of moduleKey shifted (this generalizes cleanly because RESEARCH's Pitfall 2 already anticipated a per-language moduleKey plugin point, not a per-directory one)."
    - "Bounded, non-parsing package-root descriptor heuristic: readPythonDescriptor checks only for pyproject.toml/setup.py PRESENCE (never parses their contents — no TOML dependency needed) plus a plain os.Stat check for a top-level 'src' directory to distinguish src-layout from flat-layout. A project using neither convention (a custom build-backend package-dir override) degrades gracefully to a slightly-wrong-but-still-deterministic dotted path rather than a hard failure — self-detected via D-12's golden-parity diff, matching RESEARCH Assumptions Log A1's explicitly accepted risk."

key-files:
  created:
    - internal/indexer/pyextract/pyextract.go
    - internal/indexer/pyextract/types.go
    - internal/indexer/pyextract/pyextract_test.go
    - internal/indexer/pyextract/resolution_test.go
    - internal/indexer/languages_python.go
    - testdata/golden/parity_python_test.go
    - .planning/phases/05-language-coverage-resolution-breadth/deferred-items.md
  modified:
    - internal/indexer/discover_test.go
    - internal/indexer/languages_test.go
    - internal/query/status.go
    - testdata/golden/golden_parity_test.go

key-decisions:
  - "class_definition -> goextract.KindStruct (not a new 'class' kind) — mirrors javaextract's/csharpextract's identical decision, keeping struct/class-shaped downstream consumers (Wave 6's implements synthesis) language-agnostic."
  - "A plain `import foo.bar` (no `as` alias) populates NO Imports entry — Python's real binding semantics make the bound name only the TOP-LEVEL package ('foo'), not the full dotted path, and this extractor's call-resolution only handles a single identifier.attribute hop. Only an aliased plain import (`import foo.bar as baz`, where baz genuinely IS the full dotted path) or a from-import populates Imports. This mirrors csharpextract's documented 'a plain using directive does NOT populate Imports' gap exactly."
  - "Relative imports (`from . import x`, `from ..pkg import y`) ARE resolved, against the current file's own enclosing dotted package (derived from its own moduleKey, walking up one level per extra leading dot) — this is a genuine per-language resolution algorithm (RESEARCH 'Don't Hand-Roll'), not a documented gap, since Python's relative-import-dot semantics are well-specified and common enough in real packages to be worth implementing rather than leaving unresolved."
  - "Base-class references check Imports membership for BOTH the plain-identifier shape (`class Derived(Base):`) and the attribute-chain shape (`class Derived(pkg.Base):`) — see Deviations: the plain-identifier case initially missed this check (a Rule 1 bug caught by resolution_test.go before being committed)."
  - "Inherited-method-call resolution (a subclass calling a method it inherits, never overrides, from an imported base class) is explicitly OUT of this plan's scope, per the plan's own Task 2 action and RESEARCH Pitfall 3 — that is Wave 6's conformance-retry pass (RES-02). A higher unresolvedCount attributable to inherited-method calls on any real corpus is expected, not a regression, and is documented in TestGoldenParity_Python's own package doc comment."
  - "TestGoldenParity_Python implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency) instead of a byte/shape diff against captured TS output — no live TS CodeGraph v1.3.x CLI was available in this environment (identical finding to 05-04's Java and 05-05's C#); self-skips cleanly via CODEGRAPH_PYTHON_CORPUS when no corpus is configured, and was smoke-tested this session against a real litellm/types/ subtree (168 files, Pydantic-model-heavy — real cross-module imports and inheritance) confirming 58 calls edges and 81 embeds edges resolve, with a byte-identical second-pass rebuild."

patterns-established:
  - "Discovery-time-only ModuleKey (see tech-stack.patterns) — the shape any future directory-structure-derived-identity language (as opposed to Java/C#'s in-source-declared identity) should follow: compute the full moduleKey in LanguageSpec.ModuleKey itself, no parse-time FileResult.ImportPath override needed in Extract."
  - "Bounded, non-parsing manifest-presence descriptor heuristic (see tech-stack.patterns) — a lighter-weight alternative to javaextract's/csharpextract's structured-manifest-parsing Descriptors, appropriate when a language's project-root convention can be inferred from directory-existence checks alone without needing the manifest's actual field values."

requirements-completed: [LANG-04]

coverage:
  - id: D1
    description: "pyextract package extracts class/function/method declarations into the shared codegraph vocabulary (class->KindStruct, module-level def->KindFunction, class-body def->KindMethod, decorated definitions transparently unwrapped), with module-level/class-body assignments correctly emitting no node (mirroring goextract's/javaextract's/csharpextract's ratified skip)"
    requirement: "LANG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_NodeKinds"
        status: pass
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_NoModuleLevelAssignmentNodes"
        status: pass
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_MethodContainsEdge"
        status: pass
    human_judgment: false
  - id: D2
    description: "import_statement/import_from_statement produce RefKindImports unresolved refs (aliased-plain-import and from-import populate Imports; plain unaliased import and wildcard from-import do not, both documented gaps); relative imports resolve against the file's own enclosing dotted package; class base-class-list entries produce RefKindEmbeds refs (a plain identifier checks Imports membership, an attribute chain checks its object's Imports membership, keyword/starred args skipped)"
    requirement: "LANG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_Imports"
        status: pass
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_RelativeImports"
        status: pass
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_BaseClasses"
        status: pass
    human_judgment: false
  - id: D3
    description: "call resolution correctly disambiguates four shapes: implicit self./cls. same-class call, an imported-alias-qualified call, a same-module PascalCase-qualified call (no import needed, naming-convention heuristic mirroring javaextract/csharpextract), and a local-variable-receiver call (never mis-resolved as same-module, mirrors goextract's WR-02 fix)"
    requirement: "LANG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_Calls"
        status: pass
    human_judgment: false
  - id: D4
    description: "Python is registered in the LanguageSpec registry (resolvable by ID and .py extension), with a pyproject.toml/setup.py-presence Descriptor driving a dotted-module-path ModuleKey computed entirely at discovery time (no parse-time override, unlike Java/C#), and a path-identity fallback when the descriptor is absent"
    requirement: "LANG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/languages_test.go#TestLanguageRegistry_Python"
        status: pass
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_ModuleKeyPassedThroughUnchanged"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MixedLanguage_ExtensionRegistry"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MixedLanguage_DescriptorAbsentFallback"
        status: pass
    human_judgment: false
  - id: D5
    description: "Cross-file Python resolution works end-to-end through the real indexer.Run pipeline: from-import calls, aliased-plain-import calls, and imported-base-class inheritance each land as real committed graph edges via the dotted-module-path ModuleKey"
    requirement: "LANG-04"
    verification:
      - kind: integration
        ref: "internal/indexer/pyextract/resolution_test.go#TestResolve_CrossModuleImportedCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/pyextract/resolution_test.go#TestResolve_AliasedImportCall"
        status: pass
      - kind: integration
        ref: "internal/indexer/pyextract/resolution_test.go#TestResolve_CrossModuleInheritance"
        status: pass
    human_judgment: false
  - id: D6
    description: "The INDENT/DEDENT external-scanner parse-failure path yields FileResult.Err + nil error (per-file skip; parser.MaxSourceBytes enforced BEFORE any backend-specific parsing runs, the front-line mitigation for T-05-DoS)"
    requirement: "LANG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/pyextract/pyextract_test.go#TestExtract_OversizedFileSkippedNotFatal"
        status: pass
    human_judgment: false
  - id: D7
    description: "D-12 Python validation harness (shape-not-byte, self-skipping when no corpus is configured), smoke-tested against a real, representative, license-clean Python corpus; the full repo suite (including the pre-existing Go/Java/C# golden-parity fixtures and the weft corpus's own real .py files, now genuinely extracted) remains green under -race with Python registered"
    requirement: "LANG-04"
    verification:
      - kind: integration
        ref: "testdata/golden/parity_python_test.go#TestGoldenParity_Python (self-skips absent CODEGRAPH_PYTHON_CORPUS in the committed repo; manually smoke-tested this session against a real litellm/types/ subtree — 168 files, 2099 nodes, 2068 edges, 58 calls + 81 embeds resolved, deterministic rebuild confirmed — see Issues Encountered)"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity/status (updated to assert status.Languages == [\"go\",\"python\"] against the pinned weft corpus, which genuinely contains .py files now extracted by this plan)"
        status: pass
      - kind: other
        ref: "go build ./... && go vet ./... && go test -race ./... ./testdata/golden/... -count=1 (all 17 packages pass, including testdata/golden's pre-existing Go/Java/C# golden-parity fixtures)"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 06: Python Extraction + Cross-File Resolution (LANG-04) Summary

**Full Python structural extraction (class/function/method) via a `pyextract` package mirroring `javaextract`'s shape, registered through the multi-language seam with a discovery-time-only dotted-module-path `ModuleKey` (the first priority-4 language needing no parse-time identity override), cross-file resolution (from-import, aliased-import, imported-base-class inheritance) proven end-to-end through the real `indexer.Run` pipeline, and `TestGoldenParity_Python` smoke-tested against a real 168-file Python corpus this session.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 11 (7 created, 4 modified)

## Accomplishments
- Created `internal/indexer/pyextract`: `Extract(p, moduleKey, relPath, src) (goextract.FileResult, error)` reproducing goextract's/javaextract's/csharpextract's exact per-file skip/error contract, reusing the shared vocabulary unchanged
- Node-kind mapping: `class_definition`→`KindStruct`, a `function_definition` nested directly in a class body→`KindMethod`, a module-level `function_definition`→`KindFunction`, `decorated_definition` transparently unwrapped; module-level/class-body assignments emit no node (field-skip precedent extended to Python); `import_statement`/`import_from_statement`→`RefKindImports`; a class's base-class list→`RefKindEmbeds` (extends/implements undistinguished at parse time, Pattern 2); `call`→`RefKindCalls`
- Python's dotted module path — unlike Java's `package`/C#'s `namespace`, both declared IN the source — is entirely directory-structure-derived, so `languages_python.go`'s `ModuleKey` computes the full, final identity at DISCOVERY time; `pyextract.Extract` passes it straight through unchanged (the first priority-4 extractor that does NOT need 05-04/05-05's parse-time-override pattern)
- Import handling documents a bounded, explicit gap mirroring csharpextract's own precedent: a plain unaliased `import foo.bar` binds only the top-level name in real Python, so it populates no `Imports` entry (only the `RefKindImports` dependency ref); an aliased plain import or a from-import DOES populate `Imports`. Relative imports (`from . import x`, `from ..pkg import y`) ARE resolved against the current file's own enclosing dotted package — a genuine, implemented resolution algorithm, not a gap
- Registered Python in the `LanguageSpec` registry (`languages_python.go`): `cgo.NewPythonParser` (pre-existing since Phase 1), a bounded `pyproject.toml`/`setup.py`-presence + `src`-directory-check package-root heuristic (no TOML parsing needed), path-identity fallback when absent (D-03)
- Proved cross-file resolution end-to-end through the REAL `indexer.Run` pipeline: from-import calls, aliased-plain-import calls, and imported-base-class inheritance all land as real committed `calls`/`embeds` edges via the dotted-module-path `ModuleKey`
- `TestGoldenParity_Python` implements RESEARCH's documented D-12 fallback (source-as-specification + self-consistency), self-skipping cleanly via `CODEGRAPH_PYTHON_CORPUS` when unconfigured — and was smoke-tested this session against a real litellm checkout's `litellm/types/` subtree (168 files, mostly Pydantic model classes with heavy cross-module imports/inheritance): 2099 nodes, 2068 edges, 58 `calls` + 81 `embeds` resolved, byte-identical second-pass rebuild
- Discovered and fixed, via the pinned `../weft` corpus, that `testdata/golden/golden_parity_test.go`'s `TestGoldenParity/status` subtest hardcoded `status.Languages == ["go"]` with a comment explicitly calling out weft's then-unhandled `.py` files — now genuinely extracted, so the assertion (and `status.go`'s own rationale doc comment) needed updating to `["go","python"]`

## Task Commits

Each task was committed with a RED (test) then GREEN (feat) pair, plus fix commits for issues resolution surfaced:

1. **Task 1: pyextract package + Python LanguageSpec registration**
   - `efba177` (test) — `pyextract_test.go` alone (no implementation) confirmed RED via a genuine compile failure (`undefined: Extract`), verified by temporarily removing pyextract.go/types.go
   - `7380f7a` (feat) — pyextract.go/types.go/languages_python.go + `TestLanguageRegistry_Python`, discover_test.go's inert-spec collision removal bundled in (Rule 3 blocking fix, see Deviations), confirmed GREEN
2. **Task 2: cross-file resolution + golden parity**
   - `976fff4` (fix) — resolution_test.go caught a real Rule 1 bug in Task 1's base-class handling (see Deviations) before it was ever separately committed; fix + the test that caught it land together
   - `af2d005` (test) — `TestGoldenParity_Python`, D-12 harness, smoke-tested against a real corpus
   - `250353f` (fix) — Rule 1 fix: `status.Languages` golden assertion updated for weft's now-genuinely-extracted `.py` files

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/indexer/pyextract/pyextract.go` - Extract, extractor tree-walk, node-kind mapping, imports/calls/base-class ref emission, relative-import resolution
- `internal/indexer/pyextract/types.go` - node-kind mapping decisions + full documented rationale for Python's import/resolution boundaries (package doc comment)
- `internal/indexer/pyextract/pyextract_test.go` - table-driven node-kind mapping, imports (plain/aliased/from/relative/wildcard), base classes, calls (4 disambiguation shapes), moduleKey pass-through, oversized-file skip
- `internal/indexer/pyextract/resolution_test.go` - external test package (`pyextract_test`) driving `indexer.Run` end-to-end for 3 cross-file resolution scenarios
- `internal/indexer/languages_python.go` - Python `LanguageSpec` registration, `readPythonDescriptor` (pyproject.toml/setup.py presence + src-layout detection), `dottedModulePath`
- `testdata/golden/parity_python_test.go` - `TestGoldenParity_Python`, self-skipping D-12 harness
- `internal/indexer/discover_test.go` - removed the inert test-only "python" `LanguageSpec` (05-02); updated the two affected fallback assertions to the real pyextract Descriptor's behavior
- `internal/indexer/languages_test.go` - `TestLanguageRegistry_Python`, mirroring `TestLanguageRegistry_Java`/`_CSharp`'s shape
- `internal/query/status.go` - updated the `languages` key's rationale doc comment (no longer "Go-only until Phase 5")
- `testdata/golden/golden_parity_test.go` - `TestGoldenParity/status` now asserts `["go","python"]` against the pinned weft corpus
- `.planning/phases/05-language-coverage-resolution-breadth/deferred-items.md` - new file, logs the out-of-scope flaky `internal/daemon` `TestSoak` observation

## Decisions Made
See `key-decisions` in frontmatter for the full list. Highlights:
- Python needs NO parse-time `ModuleKey` override — the first priority-4 language where discovery-time computation is already fully authoritative, since Python's identity is directory-structure-derived rather than in-source-declared
- A plain unaliased `import foo.bar` populates no `Imports` entry (matches real Python binding semantics + this extractor's single-hop attribute-chain limit); an aliased plain import or a from-import does
- Relative imports are genuinely resolved (not just documented as a gap), since the resolution algorithm is well-specified and common in real packages
- Inherited-method-call resolution is explicitly out of scope, deferred to Wave 6 per the plan's own instruction

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Base-class plain identifier never checked Imports membership**
- **Found during:** Task 2 (writing resolution_test.go's cross-module inheritance test)
- **Issue:** `collectBaseClasses`' plain-identifier case (`class Derived(Base):`) hardcoded an empty `PkgAlias` instead of checking whether `Base` is itself an imported simple name — unlike the attribute-chain case (`class Derived(pkg.Base):`) and unlike javaextract's `emitSupertypeRef`, which both correctly check `Imports` membership. This made every cross-module `class X(ImportedBase):` inheritance ref — the single most common Python inheritance shape — resolve against the WRONG moduleKey (the subclass's own file instead of the imported base's declaring module), so the required "embeds" edge never landed.
- **Fix:** Pass the identifier's own text as its pkgAlias candidate to `emitBaseClassRef`, which already correctly checks `Imports` membership and falls through to an unqualified same-module reference otherwise.
- **Files modified:** internal/indexer/pyextract/pyextract.go
- **Verification:** `TestResolve_CrossModuleInheritance` failed before the fix (wrong edge target/no edge), passes after
- **Committed in:** `976fff4`

**2. [Rule 3 - Blocking] Resolved inert "python" LanguageSpec registry collision in discover_test.go**
- **Found during:** Task 1 (reading discover_test.go before writing languages_python.go), same collision class 05-04 already fixed for Java
- **Issue:** 05-02 registered a test-only, permanently-erroring "python" `LanguageSpec` in `discover_test.go`'s `init()`. Adding the real "python" `LanguageSpec` in `languages_python.go` this plan would register a second entry under the same registry key, silently colliding (Go's package-file `init()` order determines the winner).
- **Fix:** Removed the inert "python" registration; updated `TestDiscover_MixedLanguage_ExtensionRegistry`'s and `TestDiscover_MixedLanguage_DescriptorAbsentFallback`'s python assertions to the real `languages_python.go` spec's actual fallback behavior (bare relPath, mirroring Go/Java/C#'s own nil-descriptor convention) instead of the old inert spec's `path.Dir(relPath)` shape. Bundled into the same commit as the real registration (not a separate pre-step, unlike Java's history) because splitting it would leave an intermediate state where the extension-registry test briefly fails (app.py unregistered).
- **Files modified:** internal/indexer/discover_test.go
- **Verification:** `go test ./internal/indexer/... -run TestDiscover -v` — all pass
- **Committed in:** `7380f7a` (part of Task 1's GREEN commit)

**3. [Rule 1 - Bug] Stale `status.Languages == ["go"]` golden assertion**
- **Found during:** Post-Task-2 full-suite verification against the pinned `../weft` corpus
- **Issue:** weft's committed tree includes real `.py` files that, before this plan, were discovered but never extracted (no Python `LanguageSpec` existed) — `TestGoldenParity/status` hardcoded `status.Languages == ["go"]` with a comment explicitly naming weft's unhandled `.py` files as the reason. Now that pyextract genuinely fires on them (this plan's entire purpose), the old assertion is directly contradicted by correct new behavior, not a regression.
- **Fix:** Updated the assertion to `["go","python"]` (Languages is sorted ascending) and `status.go`'s own per-key rationale doc comment to describe the general (not "until Phase 5") behavior.
- **Files modified:** testdata/golden/golden_parity_test.go, internal/query/status.go
- **Verification:** `CODEGRAPH_WEFT_CORPUS=../weft go test ./testdata/golden/... -run TestGoldenParity -v` — all 7 subtests (including status) pass
- **Committed in:** `250353f`

---

**Total deviations:** 3 auto-fixed (1 correctness bug, 1 blocking collision, 1 stale-assertion bug). 0 scope-creep.
**Impact on plan:** All three were necessary corrections directly caused by this plan's real Python registration reaching real code (either the resolution_test.go fixture or the pinned weft corpus) — none touch files outside this plan's natural surface plus the one pre-existing golden test that genuinely observes the new behavior.

## Issues Encountered

- **No live TS CodeGraph v1.3.x CLI or curated Python golden corpus committed to this repo** — identical finding to 05-04's Java and 05-05's C#. `TestGoldenParity_Python` implements the same documented fallback and self-skips via `CODEGRAPH_PYTHON_CORPUS` when unconfigured. UNLIKE Java/C# (which only report a *manual* smoke test), this plan's environment happened to have a real, representative, license-clean Python repo (a local `litellm` fork) available as a sibling checkout — the harness was smoke-tested against its `litellm/types/` subtree (168 files) and confirmed working at real-world scale (2099 nodes, 2068 edges, 58 `calls` + 81 `embeds` edges resolved, deterministic second-pass rebuild). The test is left in its self-skipping state in the committed repo (no corpus configured by default), matching Java/C#'s precedent — selecting and committing a permanent Python validation corpus reference remains a Wave F/D-11-closeout follow-up, same as Java/C#'s equivalent open item.
- **`internal/daemon`'s `TestSoak` flaked once under `-race` + full-parallel-suite contention**, then passed 3/3 in isolation and on a full-suite retry. Confirmed unrelated to this plan (no dependency on `internal/indexer/pyextract`/`languages_python.go`) — logged to `deferred-items.md`, not fixed (out of this plan's file scope).

## User Setup Required

None - no external service configuration required. Optionally, to exercise `TestGoldenParity_Python` against a real repo: `CODEGRAPH_PYTHON_CORPUS=/path/to/a/real/python/repo go test ./testdata/golden/... -run TestGoldenParity_Python -v`.

## Next Phase Readiness
- `internal/indexer/pyextract` and Python's `LanguageSpec` registration are complete and proven through the real pipeline (including a real 168-file corpus, not just synthetic fixtures); Wave 6's dispatch synthesis (RES-02) can consume the `RefKindEmbeds` shape this plan emits for base classes to promote qualifying edges to `"implements"`, and can close the deliberately-deferred inherited-method-call resolution gap via its conformance-retry pass
- The discovery-time-only `ModuleKey` pattern (no parse-time override needed) is available for any future directory-structure-derived-identity language to reuse directly
- `go build ./...`, `go vet ./...`, and `go test -race ./... ./testdata/golden/... -count=1` all pass across the full repo (17 packages including the new `pyextract`), with the pre-existing Go/Java/C# golden-parity fixtures remaining green — Python registration does not regress Go, Java, or C#

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created files confirmed present on disk (pyextract.go, types.go, pyextract_test.go, resolution_test.go, languages_python.go, parity_python_test.go); all five commits (efba177, 7380f7a, 976fff4, af2d005, 250353f) confirmed in git log.
