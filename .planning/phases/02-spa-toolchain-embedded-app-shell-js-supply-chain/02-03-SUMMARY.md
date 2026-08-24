---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
plan: 03
subsystem: ui
tags: [protobuf, buf, connect-es, typescript, codegen-drift-guard, pnpm]

requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport
    provides: "internal/uiproto/uiv1/ui.proto — the UIService schema this plan's TypeScript client is generated from"
  - phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
    provides: "02-01's web/ pnpm toolchain (web/node_modules, pnpm-lock.yaml, strictDepBuilds) that the protoc-gen-es plugin installs into"
provides:
  - "buf.gen.ts.yaml — second buf template, protoc-gen-es only, scoped to internal/uiproto/uiv1/ui.proto via the --path CLI fallback (NOT inputs: directory: — see key-decisions)"
  - "web/src/lib/gen/ui_pb.ts — committed, drift-guarded generated TypeScript exporting UIService and every message schema"
  - "web/src/lib/client.ts — transport and uiClient, the browser's typed Connect JSON client factory"
  - "task proto:gen's extension: a hard precondition on web/node_modules/.bin/protoc-gen-es plus the second buf generate + relocate step"
  - "task proto:drift's extension: floor moved 3 -> 4, fourth glob added, second scratch regeneration + relocate, demonstrated RED twice"
affects: [02-05, 02-06, 02-07]

actuals:
  tokens: 18531
  tasks: 3
  commits: 3

tech-stack:
  added: ["@bufbuild/protoc-gen-es 2.14.0 (dev)", "@bufbuild/protobuf 2.14.0", "@connectrpc/connect 2.1.2", "@connectrpc/connect-web 2.1.2"]
  patterns:
    - "buf's `inputs: directory:` override is INCOMPATIBLE with a subdirectory already covered by a configured buf.yaml module (v1.72.0) — confirmed identically in the main checkout and a scratch worktree. The working mechanism is the `--path <absolute-proto-path>` CLI flag against the default module, exactly as RESEARCH.md's Alternatives Considered predicted as the fallback."
    - "Because `--path` scoping names output relative to the WHOLE module root (not the scoped subdirectory), a plugin with no paths=source_relative-equivalent option (protoc-gen-es has none) lands its output nested under the proto's full repo-relative path. Both proto:gen and proto:drift relocate the single emitted file (mv + rm -rf the now-empty intermediate dirs) immediately after each `buf generate --path` call so the committed path stays exactly web/src/lib/gen/ui_pb.ts."
    - "Vite/SvelteKit build output is not byte-reproducible across runs on identical source (confirmed empirically, matching RESEARCH.md Pitfall 1) — running `pnpm build` for this plan's own verification produced new content-hash filenames under web/build/_app/immutable/ with no source content change; those were reverted (git checkout -- web/build/ + rm the leftover untracked hashed files) rather than committed, since this plan's file list does not include web/build/**."

key-files:
  created:
    - buf.gen.ts.yaml
    - web/src/lib/gen/ui_pb.ts
    - web/src/lib/client.ts
  modified:
    - Taskfile.yml
    - web/package.json
    - web/pnpm-lock.yaml

key-decisions:
  - "Adopted the `--path` CLI fallback, not the `inputs: directory:` override RESEARCH.md's Pattern 1 recommended as primary. Empirically tested both in the main checkout AND a scratch git worktree (git worktree add --detach /tmp/gsd-buf-worktree-test HEAD): both environments produce the byte-identical error `failed to build input \"internal/uiproto/uiv1\" because it is contained by module at path \".\" ... you must provide the workspace or module as the input, and filter to this path using --path`. This is a real buf v1.72.0 behavioral constraint (a module-configured subdirectory cannot also be declared as an independent `inputs:` entry), not the worktree-specific upward-search misidentification the existing proto:drift desc: documents for a different reason — the two are related but distinct failure modes, and this plan's evidence is the former."
  - "protoc-gen-es has no output-path-flattening option (verified against its own bundled README: target, import_extension, js_import_style, keep_empty_files, ts_nocheck, elide_plugin_version, json_types, valid_types, erasable_syntax — no paths=source_relative equivalent). Both proto:gen and proto:drift therefore relocate the nested output (web/src/lib/gen/internal/uiproto/uiv1/ui_pb.ts) to the committed flat path (web/src/lib/gen/ui_pb.ts) via mv + rm -rf immediately after the buf generate call, rather than accepting the nested path or hand-rolling a path-stripping plugin wrapper."
  - "useBinaryFormat: false is set explicitly in web/src/lib/client.ts even though @connectrpc/connect-web@2.1.2's own type definitions already default to JSON (\"By default, connect-web clients use the JSON format.\" — node_modules/@connectrpc/connect-web/dist/esm/connect-transport.d.ts:23-26, read directly per RESEARCH.md Assumption A1). Resolves A1 as: default is JSON, set explicitly anyway so a future dependency bump cannot silently flip it."

requirements-completed: [BLD-01, BLD-02]  # Both declared by multiple sibling plans in this phase (BLD-01: 02-01, 02-03, 02-06, 02-07; BLD-02: 02-01, 02-02, 02-03, 02-05). Marked complete only if requirements.ready-ids confirms every declaring plan has a SUMMARY at this plan's completion (shared-ID gate, #2388) — see Next Phase Readiness for the actual readiness result recorded by the executor.

coverage:
  - id: D1
    description: "task proto:gen regenerates the browser's TypeScript Connect client from ui.proto alone, emits exactly one committed file (web/src/lib/gen/ui_pb.ts), and fails loudly with a named remedy when its pnpm-installed plugin is absent"
    requirement: BLD-01
    verification:
      - kind: other
        ref: "task proto:gen (clean tree after) — git ls-files web/src/lib/gen count 1, file is web/src/lib/gen/ui_pb.ts; buf.gen.yaml byte-unchanged (git diff --exit-code); scoping check (test -s + rg positive/negative) exits 0 scoped, verified to exit 1 when protoc-gen-es was temporarily appended to buf.gen.yaml and reverted"
        status: pass
      - kind: other
        ref: "task proto:gen with web/node_modules moved aside — exit 201, precondition message names web/node_modules/.bin/protoc-gen-es and `pnpm install --frozen-lockfile`; restored and re-ran green"
        status: pass
    human_judgment: false
  - id: D2
    description: "task proto:drift now covers four generated files across two languages, reports the count before comparing, hard-fails below the correct floor of 4, and has been watched fail against both a staled committed file and a narrowed three-glob enumeration"
    requirement: BLD-01
    verification:
      - kind: other
        ref: "task proto:drift — printed 'compared 4 generated files' before comparing, exited 0, git diff --exit-code -- web/src/lib/gen internal/schema internal/uiproto clean afterward"
        status: pass
      - kind: other
        ref: "RED proof: one-character append to committed ui_pb.ts, ran task proto:drift — printed 'compared 4 generated files' then named web/src/lib/gen/ui_pb.ts as differing, exit 201; git checkout -- web/src/lib/gen/ui_pb.ts reverted byte-clean (diff -q against pre-edit backup), re-ran green"
        status: pass
      - kind: other
        ref: "Floor-fires proof: narrowed the git ls-files enumeration to three globs (Taskfile.yml edit), ran task proto:drift — printed 'compared 3 generated files' then the new floor-4 error message, exit 201; restored Taskfile.yml, diff against the intended edited state confirmed clean"
        status: pass
    human_judgment: false
  - id: D3
    description: "The app can construct a typed UIService client over Connect's JSON wire against the same origin that serves it, and the whole web/ tree still type-checks and builds"
    requirement: BLD-02
    verification:
      - kind: other
        ref: "cd web && pnpm exec svelte-check --threshold error --output human — 'svelte-check found 0 errors and 0 warnings', log non-empty"
        status: pass
      - kind: other
        ref: "cd web && pnpm build — succeeded, build/index.html present and non-empty (verified against committed build/ tree; the fresh non-reproducible rebuild's new content hashes were reverted per key-decisions, not committed)"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-08-24
status: complete
---

# Phase 2 Plan 3: SPA Toolchain, Embedded App Shell & JS Supply Chain — TypeScript Connect Client Summary

**Generated and committed the browser's TypeScript Connect client from `ui.proto` alone via a `--path`-scoped `buf generate` fallback (not the `inputs: directory:` override this repo's own buf.yaml module topology rejects), extended `task proto:drift` to a correctly-derived floor of 4, and wired the same-origin JSON-wire client factory `02-05` will call.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-24 (session continuation from 02-02)
- **Completed:** 2026-08-24T23:15:57Z
- **Tasks:** 3
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- Installed `@bufbuild/protoc-gen-es` (dev), `@bufbuild/protobuf`, `@connectrpc/connect`, `@connectrpc/connect-web` — all four at the exact versions RESEARCH.md verified live against the npm registry (2.14.0 / 2.14.0 / 2.1.2 / 2.1.2), reconfirmed live again this session (`npm view` on all four matched exactly). No `strictDepBuilds` approval was needed — zero lifecycle/build scripts in this dependency set.
- Discovered, by direct empirical testing rather than trusting RESEARCH.md's primary recommendation, that buf v1.72.0's `inputs: directory:` override cannot scope a subdirectory that a `buf.yaml` module already covers — this repo's single root module at `.` covers `internal/uiproto/uiv1`, so the override fails with an explicit `--path` redirect message. Tested identically in the main checkout and a scratch `git worktree add --detach` clone; same byte-identical error in both. Adopted the CLI `--path` fallback RESEARCH.md's Alternatives Considered already named as viable.
- Because `--path` scoping names output relative to the whole module root and `protoc-gen-es` has no `paths=source_relative` equivalent, both `task proto:gen` and `task proto:drift` now include a deterministic relocate step (`mv` the nested output + `rm -rf` the emptied intermediate directories) so the committed artifact lands exactly at `web/src/lib/gen/ui_pb.ts`.
- Extended `task proto:gen`: added a hard precondition on `web/node_modules/.bin/protoc-gen-es` (never a skip-when-missing branch, D-05) and a second `buf generate --template buf.gen.ts.yaml --path ...` invocation. Verified both directions: with `web/node_modules` moved aside, the target fails with a named remedy (exit 201); restored, it regenerates cleanly with `git diff --exit-code` clean afterward.
- Extended `task proto:drift`: fourth `git ls-files` glob (`web/src/lib/gen/*.ts`), floor moved from 3 to 4 (never 5 — `target=ts` makes `protoc-gen-es` emit exactly one file, RESEARCH Pitfall 3), a second scratch regeneration into the same `${scratch}/gen` root, and the identical hard plugin precondition. Watched it fail twice: a one-character edit to the committed `ui_pb.ts` (printed `compared 4 generated files`, then named the differing file, exit 201; reverted byte-clean, confirmed with `diff -q` against a pre-edit backup), and a narrowed three-glob enumeration (printed `compared 3 generated files`, then the new floor-4 error, exit 201; Taskfile.yml restored).
- Verified the leak-detection scoping check in both directions by hand: appended `protoc-gen-es` to `buf.gen.yaml` (leak) — the negated `! rg -q 'protoc-gen-es' buf.gen.yaml` check exits non-zero as designed; reverted — exits zero again. `buf.gen.yaml` stayed byte-unchanged throughout (`git diff --exit-code buf.gen.yaml` clean).
- Wrote `web/src/lib/client.ts`: `transport` (`createConnectTransport`, `baseUrl: "/"`, `useBinaryFormat: false` set explicitly) and `uiClient` (`createClient(UIService, transport)`). Read `@connectrpc/connect-web@2.1.2`'s installed type definitions directly to resolve RESEARCH.md Assumption A1: the package's own doc comment states JSON is already the default ("By default, connect-web clients use the JSON format."); the explicit `false` is set anyway so a future dependency bump cannot silently flip it.
- `pnpm exec svelte-check --threshold error` reports `found 0 errors and 0 warnings` — the generated module's types resolve and the client construction type-checks. `pnpm build` succeeds; `web/build/index.html` is present and non-empty. Vite/SvelteKit's build is not byte-reproducible (RESEARCH Pitfall 1, confirmed empirically this session — new content-hash filenames under `_app/immutable/` on an unchanged source tree); the fresh rebuild's output was reverted rather than committed, matching this plan's file scope (`web/build/**` is not in this plan's `files_modified`).

## Task Commits

1. **Task 1: Scoped second buf template and a `proto:gen` that fails loudly without its plugin** — `2527aac` `feat(02-03): generate scoped TypeScript Connect client from ui.proto`
2. **Task 2: Extend `proto:drift` to four files and demonstrate it RED** — `39e9218` `feat(02-03): extend proto:drift to four generated files across two languages`
3. **Task 3: The browser client factory over Connect's JSON wire** — `d6e96e3` `feat(02-03): browser Connect client factory over JSON wire`

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `buf.gen.ts.yaml` — new buf v2 template, `protoc-gen-es` only, `opt: target=ts`; header comment documents the empirically-decided scoping mechanism (`--path` fallback, not `inputs: directory:`) and the relocate consequence
- `web/src/lib/gen/ui_pb.ts` — generated, committed, drift-guarded; exports `UIService` (`GenService`) and every message schema from `ui.proto`
- `web/src/lib/client.ts` — `transport` and `uiClient`, the browser's typed Connect JSON client factory
- `Taskfile.yml` — `proto:gen` gains the plugin precondition and second `buf generate` + relocate; `proto:drift` gains the fourth glob, floor 4, second scratch regeneration + relocate, and the same precondition
- `web/package.json`, `web/pnpm-lock.yaml` — four new dependency entries at RESEARCH-verified pinned versions

## Decisions Made

See `key-decisions` in frontmatter. Summary: the primary scoping mechanism RESEARCH.md recommended does not work in this repo's actual buf module topology — this was discovered by direct empirical testing (including a scratch worktree, per the plan's own instruction), not assumed from the research document, and the plan's own documented Alternatives-Considered fallback was adopted with a companion relocate step neither RESEARCH.md nor the plan explicitly specified (a consequence of `protoc-gen-es` lacking a `paths=source_relative` option).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `inputs: directory:` scoping (RESEARCH.md Pattern 1, the plan's primary mechanism) fails outright in this repo — required switching to the plan's own documented fallback plus an undocumented relocate step**
- **Found during:** Task 1, step (c) — validating the input-scoping mechanism per the plan's explicit instruction
- **Issue:** `buf generate --template buf.gen.ts.yaml` with `inputs: [{directory: internal/uiproto/uiv1}]` fails with `failed to build input "internal/uiproto/uiv1" because it is contained by module at path "." specified in your configuration, you must provide the workspace or module as the input, and filter to this path using --path` — reproduced identically in the main checkout and in a scratch `git worktree add --detach /tmp/gsd-buf-worktree-test HEAD`. This is buf v1.72.0 refusing to treat a subdirectory of an already-configured module as an independent input, not the worktree-specific upward-search misidentification `proto:drift`'s pre-existing `desc:` documents (a related but distinct failure this plan's own read_first list flagged as the thing to check for).
- **Fix:** Adopted the `--path <absolute-path>` CLI fallback the plan's own step (c) instructions and RESEARCH.md's Alternatives Considered explicitly permit. Because that mechanism resolves output paths relative to the whole module root rather than the scoped subdirectory, and `protoc-gen-es` has no `paths=source_relative`-equivalent option (confirmed against its own bundled README — no such option exists), both `task proto:gen` and `task proto:drift` now include an `mv` + `rm -rf` relocate step immediately after each `buf generate --path` call, so the committed file lands exactly at `web/src/lib/gen/ui_pb.ts` as the acceptance criteria require.
- **Files modified:** `buf.gen.ts.yaml`, `Taskfile.yml`
- **Verification:** `task proto:gen` and `task proto:drift` both produce/compare exactly `web/src/lib/gen/ui_pb.ts` (no nested `internal/` subtree left behind — `rm -rf` confirmed via `git status --short` showing no stray directories); both plan-level `<verify>` blocks pass at HEAD.
- **Committed in:** `2527aac` (Task 1), `39e9218` (Task 2)

---

**Total deviations:** 1 auto-fixed (Rule 3 — a blocking mechanism failure discovered exactly where the plan instructed checking for it, resolved with the plan's own documented fallback plus a necessary companion step).
**Impact on plan:** No architecture change and no scope creep — the plan's own step (c) anticipated this exact possibility ("If it does not, switch to the fallback D-07's discretion clause permits") and the relocate step is mechanically required by the fallback's own output-naming behavior, not a new design decision. The committed artifact set, floor arithmetic (4, never 5), and every acceptance criterion are unaffected.

## Issues Encountered

None beyond the deviation above.

## Threat Flags

None — this plan's new surface (the npm-registry-sourced `protoc-gen-es` code-generating binary; committed generated code entering the repo) is exactly what this plan's own `<threat_model>` already registers (T-02-03-SC, T-02-03-02, T-02-03-03, T-02-03-04, T-02-03-05), all disposed `mitigate` and covered by this plan's own work (the hard preconditions, the floor-4 drift guard, the scoped template, the pinned matching-minor versions).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `web/src/lib/client.ts`'s `uiClient` is ready for `02-05` to import and call `getStatus({})` — the plan's own scope boundary (no error handling/retry/cancellation here) is intentional; `02-05` owns that.
- `task proto:gen` and `task proto:drift` both cover all four generated files (three Go, one TypeScript) at HEAD, tree clean, both green.
- `buf.gen.ts.yaml`'s header comment and this SUMMARY are the two places documenting the `inputs: directory:` failure and the `--path` + relocate fallback — a future plan touching either Taskfile target or buf.gen.ts.yaml should read one of these first rather than reintroducing the `inputs:` override RESEARCH.md's Pattern 1 describes as primary (it is not viable in this repo).
- **Requirement readiness:** `BLD-01` is declared by 02-01, 02-03, 02-06, 02-07; `BLD-02` is declared by 02-01, 02-02, 02-03, 02-05. Neither is expected to mark complete in `REQUIREMENTS.md` yet — sibling plans in both sets (02-05/02-06/02-07) have not yet produced SUMMARYs. The shared-ID gate (`requirements.ready-ids`) is invoked in this plan's state-update step; see the executor's actual STATE.md/REQUIREMENTS.md changes for the computed result at this plan's completion.

## Self-Check: PASSED

Confirmed on disk: `buf.gen.ts.yaml`, `web/src/lib/gen/ui_pb.ts`, `web/src/lib/client.ts`, `Taskfile.yml`, `web/package.json`, `web/pnpm-lock.yaml`, this SUMMARY.md.
Confirmed in `git log --oneline --all`: `2527aac`, `39e9218`, `d6e96e3`.
Re-ran the plan's exact `<verification>` block post-commit at HEAD: `task proto:gen` clean tree after; `task proto:drift` prints `compared 4 generated files` and exits 0; `git ls-files web/src/lib/gen` reports exactly 1; `pnpm exec svelte-check --threshold error` prints `found 0 errors`; `pnpm build` succeeds with a non-empty `build/index.html`; `buf.gen.yaml` byte-unchanged and the scoping check proven to exit non-zero on a temporary leak, zero after revert.

---
*Phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain*
*Plan: 03*
*Completed: 2026-08-24*
