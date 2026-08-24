---
phase: 2
reviewers: [codex]
reviewed_at: 2026-08-24T00:59:41Z
plans_reviewed: [02-01-PLAN.md, 02-02-PLAN.md, 02-03-PLAN.md, 02-04-PLAN.md, 02-05-PLAN.md, 02-06-PLAN.md, 02-07-PLAN.md]
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 2

## Codex Review

# Cross-AI Plan Review

## 1. Summary

The seven plans are unusually thorough, source-aware, and correctly ordered, but the phase is not ready to execute unchanged. Three high-risk gaps undermine the phase’s core supply-chain claims:

1. `web:drift` binds a source digest to a marker but never binds or regenerates the committed `web/build/` bytes, so direct modification of the shipped SPA passes.
2. BLD-07 scans only the immediate YAML text, although the release workflow executes Taskfile targets and local actions transitively.
3. Several verification commands can pass despite failure, including a command that prints `LEAK` and exits successfully and a full-suite command masked by `tail`.

There is also a concrete schema mismatch: plan 02-05 says it renders all eight `GetStatusResponse` fields, but the message has nine fields, including `indexing_in_progress`. The dependency graph is acyclic, all declared task counts and `files_modified` counts are correct, and all 19 locked decisions have corresponding work, subject to the implementation defects below.

## 2. Strengths

- The seven-plan dependency graph is acyclic and correctly wave-ordered:

  - Wave 1: 02-01
  - Wave 2: 02-02, 02-03, 02-04 depend on 02-01
  - Wave 3: 02-05 depends on 02-01 and 02-03
  - Wave 4: 02-06 depends on 02-03 and 02-05
  - Wave 5: 02-07 depends on 02-05 and 02-06

  No plan depends on a same-wave or later-wave plan.

- All declared task counts are arithmetically correct:

  | Plan | Declared | Actual |
  |---|---:|---:|
  | 02-01 | 3 | 3 |
  | 02-02 | 2 | 2 |
  | 02-03 | 3 | 3 |
  | 02-04 | 2 | 2 |
  | 02-05 | 3 | 3 |
  | 02-06 | 3 | 3 |
  | 02-07 | 3 | 3 |

  Total: 19 tasks, including 02-01’s blocking checkpoint.

- All `files_modified` counts are correct: 19, 2, 6, 1, 14, 3, and 4 respectively.

- The plans correctly distinguish existing and newly created paths. The entire `web/` tree, `buf.gen.ts.yaml`, `internal/uiserver/spa.go`, and `internal/uiserver/spa_test.go` do not exist yet; the plans explicitly identify them as phase-created artifacts. Existing sources confirm the current state has only the Connect handler mounted, followed by the outer origin guard at [internal/uiserver/server.go:111](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:111), [internal/uiserver/server.go:119](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:119), and [internal/uiserver/server.go:125](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:125).

- 02-01 correctly places the embed directive inside `web/`. The repository already documents that embed patterns cannot traverse `..` at [claudeassets.go:8](/Volumes/Code/github.com/seanb4t/codegraph-go/claudeassets.go:8), while the existing in-package embed precedent appears at [internal/mcp/resources.go:18](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/resources.go:18).

- 02-01 correctly revisits nested ignore behavior. The existing root rule is only `/dist/` at [.gitignore:4](/Volumes/Code/github.com/seanb4t/codegraph-go/.gitignore:4); it does not itself exclude `web/build/`.

- 02-01’s embedded-versus-disk set comparison is genuinely positive: it requires a non-empty tree and at least one `_app/immutable/` entry. That closes the two-empty-sets failure mode and makes omission of `all:` observable.

- 02-02’s routing architecture matches the source. `originHostGuard` currently wraps the entire mux, and the existing package documentation explicitly anticipates Phase 2’s static routes at [internal/uiserver/server.go:82](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:82). Registering `"/"` before the existing wrapping preserves that boundary.

- 02-03 correctly identifies the current codegen arithmetic. The repository has exactly three committed generated Go files, and `proto:drift` currently prints the enumerated count and enforces a floor of three at [Taskfile.yml:224](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:224)–[Taskfile.yml:232](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:232). Adding exactly one `target=ts` output makes the correct new floor four.

- 02-03 also correctly avoids modifying the module-wide Go template. The current `buf.gen.yaml` applies both existing plugins across the root module at [buf.gen.yaml:10](/Volumes/Code/github.com/seanb4t/codegraph-go/buf.gen.yaml:10)–[buf.gen.yaml:17](/Volumes/Code/github.com/seanb4t/codegraph-go/buf.gen.yaml:17), and the root module contains both protos at [buf.yaml:1](/Volumes/Code/github.com/seanb4t/codegraph-go/buf.yaml:1)–[buf.yaml:11](/Volumes/Code/github.com/seanb4t/codegraph-go/buf.yaml:11).

- Folding JS checks into the existing `test` job is source-supported. `test` is a required context at [internal/upgrade/taskfile_shape_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:43), and that job is already inside the single-definition guard at [internal/upgrade/taskfile_shape_test.go:109](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:109).

- 02-04’s plan to parse YAML and fail on missing/empty files is materially better than a raw grep. The repository already uses explicit zero guards in this area; for example, the required-context scanner fails when it reads zero workflows at [internal/upgrade/taskfile_shape_test.go:704](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:704)–[internal/upgrade/taskfile_shape_test.go:729](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:729).

- 02-07 correctly updates an incomplete security statement. `SECURITY.md` currently says only that `govulncheck` gates dependency vulnerabilities at [SECURITY.md:61](/Volumes/Code/github.com/seanb4t/codegraph-go/SECURITY.md:61)–[SECURITY.md:63](/Volumes/Code/github.com/seanb4t/codegraph-go/SECURITY.md:63), which will be incomplete once the JS tree lands.

## 3. Concerns

- **HIGH — 02-06 — The proposed drift guard does not prove `web/build/` matches source.**  
  The marker hashes only source/configuration inputs. A contributor can directly alter `web/build/_app/immutable/*.js`, leave `.source-sha256` untouched, and `task web:drift` will pass because neither the altered asset nor any build-output digest participates. This violates BLD-03’s claimed mechanism and leaves the exact bytes embedded into the binary unbound to reviewed source. The existing proto guard actually regenerates and compares every generated artifact at [Taskfile.yml:179](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:179)–[Taskfile.yml:199](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:199); the SPA proposal deliberately does neither.

- **HIGH — 02-06 — CI never rebuilds the SPA despite claiming a clean checkout rebuild is verified.**  
  The planned CI steps are setup-node, `task web:deps`, and `task web:drift`; `task web:build` is not run. Therefore CI proves only that the committed marker matches current source—not that the pinned toolchain can reproduce a functioning build or that `pnpm build` still succeeds. This contradicts the plan’s truth that a clean checkout “rebuilds a `web/build/` the drift guard finds matching.”

- **HIGH — 02-04 — BLD-07’s scanner is not transitive enough to prove “reachable anywhere in the signed release path.”**  
  The release workflow invokes `task release:goreleaser` and `task release:record-final-hashes` at [.github/workflows/release.yml:228](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:228) and [.github/workflows/release.yml:274](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:274). It also executes the local composite action at [.github/workflows/release.yml:162](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:162), whose implementation shells out at [.github/actions/install-task/action.yml:17](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/actions/install-task/action.yml:17)–[.github/actions/install-task/action.yml:28](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/actions/install-task/action.yml:28). Scanning only the two YAML documents’ immediate scalars would miss Node/pnpm introduced into either Taskfile target or a referenced local action. The claimed “reachable anywhere” property is therefore stronger than the scanner’s reachability model.

- **HIGH — 02-03 — The scoped-codegen verification passes even when it detects a leak.**  
  Exact command:

  ```sh
  rg -q 'target=ts' buf.gen.ts.yaml && rg -q 'protoc-gen-es' buf.gen.yaml && echo LEAK || echo SCOPED
  ```

  Both `echo LEAK` and `echo SCOPED` exit zero. Thus the automated verification passes when `protoc-gen-es` incorrectly appears in the module-wide `buf.gen.yaml`. It lacks `test "$result" = SCOPED` or an explicit non-zero exit on `LEAK`.

- **HIGH — 02-07 — The full-suite verification masks `go test` failures.**  
  Exact command:

  ```sh
  go test ./... 2>&1 | tail -30
  ```

  Without `set -o pipefail`, the pipeline returns `tail`’s status. A failing test suite can therefore produce a green verification. It lacks either `set -o pipefail`, an explicit captured test status, or a positive assertion from `go test -json`.

- **HIGH — 02-05 — The GetStatus field count and degraded-state model are wrong.**  
  The plan repeatedly requires rendering “all eight” fields. The actual proto has nine fields: `initialized`, `version`, three counts, `stale`, `commit_sha`, `store_exists`, and `indexing_in_progress` at [internal/uiproto/uiv1/ui.proto:116](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:116)–[internal/uiproto/uiv1/ui.proto:177](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:177). Omitting field 9 weakens the promised “indexing in progress” rendering and forces the UI to infer it from `store_exists && !initialized`, despite the schema providing a direct signal.

- **MEDIUM — 02-04 — The forbidden action list is narrower than the stated property.**  
  It catches `actions/setup-node` and `pnpm/action-setup`, but a release job could install Node through a different action, a local action, a container image, `brew`, `curl`, or another Taskfile target. The plan should describe this as a curated-denylist guard unless it adds structural transitive analysis.

- **MEDIUM — 02-01/02-05 — Executable tooling is not immutable enough.**  
  The plans run `pnpm dlx sv@latest` and `pnpm dlx shadcn-svelte@latest`. Those commands fetch and execute whatever is current at execution time, after the package-legitimacy review was written. Neither CLI is locked by `pnpm-lock.yaml`, and the shadcn CLI is intentionally absent from `package.json`. This weakens provenance and makes later reproduction of the scaffold/init impossible.

- **MEDIUM — 02-01/02-06 — The pnpm manager pin lacks an integrity hash or independent provenance assertion.**  
  `"packageManager": "pnpm@11.23.0"` pins a version but not the bytes of the package-manager artifact. The plans verify the resolved version and frozen dependency lock, but do not verify Corepack’s package-manager integrity metadata or otherwise attest the pnpm binary used to create committed assets.

- **MEDIUM — 02-06 — Lockfile integrity is only indirectly checked.**  
  `--frozen-lockfile` prevents rewriting, but no planned check asserts that every registry-backed package entry carries an integrity checksum, that the lockfile version is expected, or that no `file:`, mutable Git branch, HTTP tarball, or workspace-escaping dependency source has entered. For a phase explicitly centered on JS supply-chain integrity, this is a meaningful omission.

- **MEDIUM — 02-06 — CI cache poisoning is addressed only by omission, not enforced.**  
  The plan deliberately adds no cache, which is sound, but no structural test prevents a later `actions/cache` or setup-node `cache: pnpm` addition to the JS install path. The release workflow already documents why mutable caches are unacceptable inside its signing boundary at [.github/workflows/release.yml:115](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:115)–[.github/workflows/release.yml:120](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:120). The JS build should receive an equivalent invariant.

- **MEDIUM — 02-02 — CSP is absent from the embedded-SPA serving policy.**  
  `nosniff` and cache policy are useful, but there is no Content-Security-Policy. A local application rendering repository-derived data and executing committed JS should at least constrain `default-src`, `script-src`, `style-src`, `connect-src`, `object-src`, `base-uri`, and framing. Origin/Host validation protects the server boundary; it does not constrain what a compromised or injected SPA asset may load.

- **MEDIUM — 02-07 — The `pnpm audit` detector itself remains unproved.**  
  The lockfile count proves real input but, as the plan admits, does not prove that advisory results are parsed correctly. That limitation is acceptable only if the phase’s success language is narrowed. Current wording says the audit gate distinguishes clean, vulnerable, and scan-failed outcomes, but only clean and scan-error paths are demonstrated.

- **MEDIUM — 02-06/02-07 — Several multi-test verifications assert only that one test passed.**  
  Both plans use:

  ```sh
  ... | tee ... && grep -q -- '--- PASS' ...
  ```

  The named `-run` expressions contain three tests, but one `PASS` line satisfies the command. This is not fully vacuous, but it does not positively assert that every requested guard test ran.

- **LOW — 02-01 — The first tracer verification does not assert an exact match count.**  
  `grep -c 'PASS: TestSPAServesEmbeddedIndexAtRoot'` succeeds with one or more matches. It is adequate for non-vacuity but weaker than the exact-count discipline used elsewhere.

- **LOW — 02-02 — The absence-only static checks violate rule `84d1gfpywd` when considered independently.**  
  Examples include the planned absence checks for `http.FileServer` and hardcoded RPC service strings. They lack a command-local assertion that the subject file existed and was scanned. The Go tests provide contextual protection, but the exact negative commands remain vacuous if copied or run independently.

- **LOW — 02-05 — The “no vendored component” guard is negative-only.**  
  Exact guard:

  ```sh
  git ls-files 'web/src/lib/components/ui/*'
  ```

  expecting zero entries. It lacks a positive assertion that the intended `web/` project and shadcn configuration were present when scanned. Other plan checks partly compensate, but the command itself is vacuous under rule `84d1gfpywd`.

## Guard/Verification Non-Vacuity Inventory

The four milestone guards named by the context are mostly designed correctly:

| Guard | Verdict |
|---|---|
| Extended `proto:drift` | Non-vacuous in design: enumerated count printed before comparison, floor 4, stale-file and narrowed-enumeration RED proofs. Existing positive pattern is visible at [Taskfile.yml:224](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:224)–[Taskfile.yml:232](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:232). |
| `web:drift` | Enumeration itself is non-vacuous: printed count, derived floor, stale-source/floor/missing-marker RED proofs. Mechanistically insufficient because it does not bind output bytes. |
| `web:deps:strict` | Non-vacuous in design: parsed boolean assertion, approval and denial counts, absent-map failure, RED demonstrations. |
| `web:audit` | Non-vacuous as an input scan: independent lockfile package count, floor, and scan-error RED proof. Detector-firing ability remains deliberately unproved. |

Additional vacuous or weakened verification commands introduced by the plans:

- 02-03 scoped-template check: prints `LEAK` but exits zero; lacks a failing assertion on `LEAK`.
- 02-07 full Go suite: pipeline status comes from `tail`; lacks `pipefail` or captured `go test` status.
- 02-06 and 02-07 multi-test commands: assert only one `PASS`; lack exact per-test assertions.
- 02-02 negative `rg` checks: lack a command-local assertion that `spa.go`/`spa_test.go` existed and were scanned.
- 02-05 zero-component check: lacks a positive assertion that the configured component root/project existed.

No `verify:` command runs against a plan-created file before the plan/task that creates it. I found no wrong-wave verification ordering requiring a HIGH finding under that rule.

## Locked Decision Coverage

All 19 decisions have corresponding tasks:

| Decisions | Implemented by |
|---|---|
| D-01–D-04 | 02-01 |
| D-05–D-08 | 02-03 |
| D-09–D-12 | 02-01 and 02-02 |
| D-13 | 02-06 and 02-07 |
| D-14 | 02-01 and 02-07 |
| D-15 | 02-07 |
| D-16 | 02-01 and 02-06 |
| D-17–D-18 | 02-05 |
| D-19 | 02-05 |

D-19 is present but not fully honored because the page specification omits the ninth response field, `indexing_in_progress`.

## 4. Suggestions

- Replace the source-only marker with a two-part attestation:

  1. a source digest covering all build inputs; and
  2. a deterministic manifest of committed output paths and byte digests.

  `web:drift` must validate both. This catches direct output tampering even if a fresh rebuild cannot be byte-compared reliably.

- Add a separate CI buildability check that runs `pnpm build` into an uncommitted temporary/output directory and validates structural invariants: successful build, `index.html`, non-empty immutable JS/CSS assets, and expected embedded-entry classes. Do not confuse this with byte reproducibility.

- Make BLD-07 transitive. Parse the release workflow, follow local `uses:` actions, resolve every `task <target>` invocation, and scan the referenced Taskfile command bodies. At minimum, explicitly include:

  - `release:goreleaser`
  - `release:record-final-hashes`
  - `.github/actions/install-task/action.yml`
  - executable GoReleaser hooks

- Replace the faulty 02-03 command with:

  ```sh
  rg -q 'target=ts' buf.gen.ts.yaml &&
  ! rg -q 'protoc-gen-es' buf.gen.yaml
  ```

  Add `test -s buf.gen.yaml` and `test -s buf.gen.ts.yaml` first for command-local positivity.

- Replace the full-suite pipeline with:

  ```sh
  set -o pipefail
  go test ./... 2>&1 | tee /tmp/go-test.log | tail -30
  ```

- Update 02-05 to render and test all nine GetStatus fields, using `indexing_in_progress` directly for the locked-store state.

- Pin scaffolding tools in a committed, lockfile-covered tool package or use exact versions plus verified integrity. Avoid `@latest` for executable project-generation tooling.

- Add a lockfile-shape guard that parses `pnpm-lock.yaml`, asserts the expected lockfile version, counts integrity-bearing registry resolutions, and rejects unapproved mutable or non-registry dependency sources.

- Enforce the no-cache decision structurally for the JS install/build steps, including setup-node’s `cache` input and `actions/cache`.

- Add a strict CSP appropriate to the generated Svelte app. Keep `connect-src 'self'` so Connect requests continue to work.

- Convert every test-pattern verification into exact named-test assertions rather than a generic “at least one PASS.”

## 5. Risk Assessment

**Overall risk: HIGH**

The plan set is well organized and covers the intended feature surface, but its most important security claim—committed SPA output matching reviewed source—is not actually established. The release-purity scanner is also shallower than the release execution graph, and two explicit verification commands can return success after detecting or encountering failure. Those are guard-mechanism defects, not documentation polish, and should be corrected before execution.

---

## Consensus Summary

Only one external reviewer ran this cycle (Codex, `gpt-5.6-sol` at `reasoning=low`), so there is no
multi-reviewer consensus to compute. In its place, the orchestrator independently verified Codex's
load-bearing claims against the worktree and ran its own pass over the same seven plans. Findings
below are marked **[codex]**, **[orchestrator]**, or **[both]**. Everything marked
**[codex, verified]** was re-checked against the repository by the orchestrator and confirmed.

The internal `gsd-plan-checker`'s earlier PASS is **not** treated as prior evidence in this cycle —
three of its evidence claims had already been falsified (a miscounted task total, a claimed-absent
section that exists at a known line, and a shell check run against a directory that does not exist
yet). Task counts, `files_modified` counts, the wave DAG, and the D-01..D-19 coverage map were all
re-derived from scratch here and are correct.

### Agreed Strengths

- **Wave DAG is acyclic and correctly ordered** [both, verified]. 02-01 → {02-02, 02-03, 02-04} →
  02-05 → 02-06 → 02-07. No plan depends on a same-wave or later-wave plan.
- **Task counts are arithmetically correct** [both, verified]. 3/2/3/2/3/3/3 = 19 tasks including
  02-01's blocking-human checkpoint. Declared `tasks:` matches the `<task>` count in every plan.
- **All 19 locked decisions D-01..D-19 have corresponding work** [both, verified].
- **Requirement coverage is complete** [orchestrator, verified]. ROADMAP Phase 2 names RPC-03,
  BLD-01, BLD-02, BLD-03, BLD-05, BLD-06, BLD-07; every one is claimed by at least one plan.
  BLD-04 is correctly absent — `.planning/REQUIREMENTS.md:182` assigns it to Phase 1, Complete.
- **The plans correctly distinguish existing from phase-created paths** [both, verified]. `web/`,
  `buf.gen.ts.yaml`, `internal/uiserver/spa.go` and `internal/uiserver/spa_test.go` do not exist
  yet and are named as phase-created. **No `verify:` command runs against a plan-created path
  before the task that creates it** — the wrong-wave ordering hazard was searched for and not found.
- **02-01's embed set-diff is genuinely non-vacuous** [both]. `TestEmbeddedBuildTreeIsNonTrivial`
  requires a non-empty walk plus at least one `_app/immutable/` entry, closing the two-empty-sets
  failure mode that would otherwise let a missing `all:` prefix pass.
- **02-01's `git check-ignore` guard carries a working positive control** [orchestrator, verified].
  `git check-ignore -v dist/artifacts.json` exits 0 today against `.gitignore:4` (`/dist/`), and
  `git check-ignore -v web/build/index.html` exits 1. The control is real, not hypothetical.
- **The `proto:drift` floor arithmetic is right** [both, verified]. Three committed generated Go
  files today with the floor at 3 (`Taskfile.yml:224`-`:232`); exactly one `target=ts` output makes
  4 the correct new floor, and 02-03 explicitly refuses the plausible-but-wrong 5.
- **02-07's `web:audit` sibling assertion is well designed** [both]. It runs before `pnpm audit` is
  invoked at all, counts lockfile packages, and hard-fails below a derived floor of 50 — genuinely
  independent of the audit exit code, exactly as D-15 requires.

### Agreed Concerns

**HIGH**

1. **02-06 — `web:drift` never binds the committed `web/build/` bytes to anything** [codex,
   verified]. The marker hashes only `web/src`, `web/static` and seven named config files
   (`02-06-PLAN.md:155-157`). A contributor can edit `web/build/_app/immutable/*.js` directly,
   leave `.source-sha256` untouched, and the guard passes. Those are the exact bytes embedded into
   the shipped binary, and BLD-03's claimed mechanism does not cover them. Contrast `proto:drift`,
   which regenerates and byte-compares every artifact (`Taskfile.yml:179-199`).

2. **02-06 — CI never runs `task web:build`, so the "clean checkout rebuilds" truth is unproven**
   [codex, verified]. The planned CI steps are setup-node + `task web:deps` + `task web:drift`
   (`02-06-PLAN.md:29`). CI therefore proves the marker matches source, never that the pinned
   toolchain still produces a working build. ROADMAP criterion 3 asks for the rebuild.

3. **02-04 — BLD-07's scanner is shallower than "reachable anywhere in the signed release path"**
   [codex, verified]. `release.yml:228` runs `task release:goreleaser`, `:274` runs
   `task release:record-final-hashes`, and `:162` runs the local composite action
   `.github/actions/install-task` which shells out at `action.yml:17-28`. Scanning only the two
   YAML documents' immediate scalars cannot see Node/pnpm introduced into a Taskfile target, a
   local action, or a GoReleaser hook script. Either make the scan transitive over those three
   edges or narrow the plan's own claim to "these two documents".

4. **02-03 — the scoped-codegen verify passes even when it detects the leak** [both, verified].
   `rg -q 'target=ts' buf.gen.ts.yaml && rg -q 'protoc-gen-es' buf.gen.yaml && echo LEAK || echo SCOPED`
   exits 0 on both branches. The acceptance criterion says a human should see `SCOPED`, but the
   command sits in an `<automated>` slot where only the exit code is read. This is precisely the
   vacuous shape rule `84d1gfpywd` exists to catch. Replace with
   `test -s buf.gen.yaml && test -s buf.gen.ts.yaml && rg -q 'target=ts' buf.gen.ts.yaml && ! rg -q 'protoc-gen-es' buf.gen.yaml`.

5. **02-07 — the full-suite verify masks `go test` failures** [both, verified]. `go test ./... 2>&1 | tail -30`
   returns `tail`'s status without `set -o pipefail`. A red suite verifies green. Needs `pipefail`
   or a captured `go test` status.

6. **02-05 — `GetStatusResponse` has NINE fields, not eight** [codex, verified].
   `internal/uiproto/uiv1/ui.proto` declares `initialized=1 … store_exists=8, indexing_in_progress=9`.
   `02-05-PLAN.md:327` requires rendering "all eight" and lists eight labels. Field 9 is the direct
   signal for the indexing state; omitting it forces the shell to infer it from
   `store_exists && !initialized`, which is D-19's honesty requirement implemented on worse data.

7. **02-07 edits a hashed-source-set file but never refreshes the drift marker** [orchestrator].
   `02-06-PLAN.md:157` puts `web/pnpm-workspace.yaml` inside `web:drift`'s hashed enumeration.
   02-07 (wave 5, i.e. after 02-06) lists `web/pnpm-workspace.yaml` in `files_modified` and its
   must_haves call it "the final committed `strictDepBuilds` and `allowBuilds` state" — but 02-07
   never runs `task web:build` to rewrite `web/build/.source-sha256`, never lists
   `web/build/.source-sha256` in `files_modified`, and never runs `task web:drift` in any verify
   block. If any byte of that file changes in 02-07, the phase's own CI gate (added one wave
   earlier) is RED on the phase's final commit and nothing in 02-07 would notice. Either add
   `task web:build` + the marker to 02-07, add `task web:drift` to 02-07's final verify, or drop
   `web/pnpm-workspace.yaml` from `files_modified` and state explicitly that 02-07 leaves it
   byte-identical.

**MEDIUM**

8. **02-03 / 02-05 — `grep -qE '0 errors'` false-passes on counts ending in zero**
   [orchestrator]. Four `svelte-check` verifies use it (`02-03-PLAN.md:298`, `02-05-PLAN.md:190`,
   `:247`, `:321`). `svelte-check found 10 errors` contains the substring `0 errors`, so the guard
   goes green on 10, 20, 30 … errors. Anchor it: `grep -qE 'found 0 errors'`. The same lines also
   use `;` rather than `&&` before the grep, so a `pnpm exec` crash is not itself a failure.

9. **02-05 does not declare `depends_on: 02-02`** [orchestrator]. `02-05-PLAN.md:6` lists only
   `["02-01","02-03"]`, but its Task 3 verify asserts `-ge 9` `--- PASS` lines from
   `TestEmbeddedFSMatchesOnDiskBuildTree|TestEmbeddedBuildTreeIsNonTrivial|TestSPA` — seven of those
   nine are the `TestSPA*` tests 02-02 authors. Wave gating happens to cover it, but an executor
   that schedules on `depends_on` rather than on wave would run 02-05 against a two-PASS tree.

10. **02-02 — no Content-Security-Policy anywhere in the phase** [both, verified]. `CSP` /
    `Content-Security-Policy` appears zero times in all seven plans, in `02-CONTEXT.md`, and in
    `02-RESEARCH.md`; `rg` over `internal/uiserver/` finds no security headers today. This phase
    turns a JSON-only server into an HTML-and-JS-serving one. `nosniff` and the origin/Host guard
    protect different things: neither constrains what a compromised or injected embedded asset may
    load. It is not in the Deferred Ideas list either — so it is currently invisible to
    `/gsd-execute-phase`. Either add a `default-src 'self'` / `connect-src 'self'` policy to
    02-02's header set, or record an explicit deferral with rationale.

11. **02-01 / 02-05 — scaffolding tooling is executed from `@latest`** [codex, verified].
    `pnpm dlx sv@latest` (`02-01-PLAN.md:232`) and `pnpm dlx shadcn-svelte@latest init`
    (`02-05-PLAN.md:162`) fetch and execute whatever is current at run time, after the
    package-legitimacy checkpoint was written, and neither is covered by `pnpm-lock.yaml`
    (`02-05-PLAN.md:198` makes the exclusion deliberate). For a phase whose subject is JS supply
    chain, this is the one code path with no provenance at all. Pin exact versions.

12. **02-06 — lockfile integrity is only indirectly checked** [codex]. `--frozen-lockfile` proves
    the lockfile was not rewritten; nothing asserts that every registry-backed entry carries an
    integrity checksum, that the lockfile version is the expected one, or that no `file:`, mutable
    git-branch, or HTTP-tarball source has entered the tree.

13. **02-06 — the no-cache decision is unenforced** [codex, verified]. Adding no cache is right,
    but no structural test prevents a later `actions/cache` or `setup-node: cache: pnpm` on the JS
    install path. `release.yml:115-120` already documents why mutable caches are unacceptable
    inside the signing boundary; the JS path deserves the same invariant, and 02-04's scanner is
    the natural home for it.

14. **02-04 — the forbidden-action list is a curated denylist, not a structural property** [codex].
    It catches `actions/setup-node` and `pnpm/action-setup`; a container image, `brew`, `curl`, or
    a different marketplace action would pass. Either say so in the plan or widen the model.

15. **02-06 / 02-07 — several multi-test verifies assert only that ONE test passed** [both,
    verified]. `... && grep -q -- '--- PASS' ...` after a `-run` expression naming three tests
    (`02-06-PLAN.md:318`, `02-07-PLAN.md:304`). Other plans use exact `-eq N` counts; these two
    should too, especially since `go test -run` exits 0 when the pattern matches nothing.

16. **02-07 — the `pnpm audit` detector's ability to fire remains unproved** [codex]. Accepted by
    D-15 and explicitly deferred, but 02-07's own must_have says the gate "distinguishes 'scanned
    and clean' from 'the scan itself failed'" while only the clean and scan-error paths are
    demonstrated — the vulnerable path is never exercised. Narrow the claim or exercise it.

**LOW**

17. **02-01 — the tracer verify does not assert an exact match count** [codex, verified].
    `grep -c 'PASS: TestSPAServesEmbeddedIndexAtRoot'` succeeds on one or more matches, weaker than
    the `-eq N` discipline used in the sibling task.

18. **02-01 — `git check-ignore` conflates "not ignored" with "error"** [orchestrator].
    `git check-ignore -v web/build/index.html; test $? -ne 0` is true for exit 1 (not ignored,
    desired) and for exit 128 (git error). The `dist/artifacts.json` positive control mitigates
    this, but the first leg alone cannot tell the two apart.

19. **02-02 / 02-05 — absence-only static checks are locally vacuous** [codex]. The planned
    `http.FileServer` absence check, the hardcoded-RPC-string absence check, and
    `git ls-files 'web/src/lib/components/ui/*'`-expecting-zero all lack a command-local assertion
    that the subject existed and was scanned. Surrounding Go tests compensate contextually, but the
    commands themselves are the shape rule `84d1gfpywd` names.

20. **ROADMAP still says `dist/`, the plans say `web/build/`** [orchestrator]. ROADMAP Phase 2
    criterion 3 and its Notes say "rebuilds a `dist/`" and "`//go:embed all:dist`"; D-02 moved the
    tree to `web/build/` and the plans use `//go:embed all:build`. The substance is unchanged and
    the deviation is intentional, but the ROADMAP acceptance text now names a path the phase never
    creates.

### Divergent Views

No divergence to report — a single reviewer ran. Where the orchestrator's independent pass
disagreed with Codex it was by addition, not contradiction: Codex concluded "no wrong-wave
verification ordering" (confirmed) and "the dependency graph is acyclic and correctly ordered"
(confirmed), but did not check whether a plan's *asserted test count* implies a dependency it does
not declare (finding 9) or whether a later wave invalidates an earlier wave's committed digest
(finding 7). Codex also did not catch the `'0 errors'` substring false-pass (finding 8).

**Reviewer-count caveat.** One grounded reviewer is not adversarial review. Everything above marked
**[codex]** without **verified** rests on a single model at `reasoning=low`; the HIGH items were all
re-checked by hand and stand, but the MEDIUM/LOW set has had no second opinion. Consider a second
lane before treating this cycle as a clean gate.

### Overall Risk

**HIGH** — concurring with Codex. The plan set's structure, ordering, arithmetic and decision
coverage are all sound, and the guard *designs* are unusually disciplined about rule `84d1gfpywd`.
But the phase's central security claim — that the SPA bytes shipped inside the binary match reviewed
source — is not established by the guard that is supposed to establish it (finding 1), the release
purity scanner is shallower than the release execution graph (finding 3), two verification commands
return success after detecting or encountering failure (findings 4 and 5), and a wave-5 edit can
leave the phase's own CI gate red with nothing watching (finding 7). These are guard-mechanism
defects, not documentation polish.
