---
phase: 01-cross-compile-spike-goreleaser-release-migration
plan: 02
subsystem: infra
tags: [goreleaser, cosign, syft, sbom, signing, release-config, yaml]

# Dependency graph
requires:
  - phase: 01-01
    provides: "zig-cross-compiling .goreleaser.yaml build matrix; goreleaser_shape_test.go's parseGoreleaserBuildEnv/mustGoreleaserBuildEnv helper pair and yaml.Unmarshal-based parser convention; syft/cosign installed as release:dry-run preconditions"
provides:
  - "archives: split into id: raw (formats: [binary], byte-unchanged name_template — the internal/upgrade.releaseAssetName() runtime contract) and id: zip (formats: [zip], same stem, GoReleaser's actual default file set: binary+LICENSE*+README*+CHANGELOG*)"
  - "checksum.ids: [raw, zip] scoping the checksums file to exactly the 8 downloadable payloads"
  - "binary_signs: block reproducing today's hand-rolled cosign sign-blob loop declaratively, with a corrected Go-template-FIELD-based signature: template (a $artifact-env-var form, matching RESEARCH.md's Pattern 2 and GoReleaser's own documented default, was found during this task to collide to one name for all 4 platforms)"
  - "sboms: block reproducing today's hand-rolled syft loop declaratively, with the NAME-derived documents: template cycle-3 review required (HIGH-B), empirically confirmed via a real task release:dry-run producing four distinct .spdx.json names"
  - "release: block pinning replace_existing_artifacts: true and prerelease: auto, replacing gh release upload --clobber's idempotency and release.yml's prerelease case"
  - "internal/upgrade/goreleaser_shape_test.go: parseGoreleaserArchives/mustGoreleaserArchives, parseGoreleaserTopLevelBlock/mustGoreleaserTopLevelBlock, parseGoreleaserBinarySigns/mustGoreleaserBinarySigns, parseGoreleaserSBOMs/mustGoreleaserSBOMs, resolveGoreleaserFieldTemplate, and 7 new tests — all 8 mutation-RED demonstrations recorded below"
affects: ["01-03 (deletes release.yml's hand-rolled checksum/sign/sbom/upload steps this plan's config blocks replace)", "01-05 (observes release: for real, and closes the single-writer checksum invariant)", "01-06 (release:dry-run-signed exercises binary_signs:/sboms: against a real pipe)"]

# Actuals (#2632)
actuals:
  tokens: 9177
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Go-template-FIELD-based (.ProjectName/.Tag/.Os/.Arch or .ArtifactName) name templates for GoReleaser sidecar-naming config, never $artifact/${artifact} env-var substitution, whenever the published artifact's name must vary per platform — the env var resolves to the artifact's un-renamed .Name on the publish-naming pass for BOTH binary_signs: and (via a different mechanism) would for a naively-path-derived sboms: template."
    - "resolveGoreleaserFieldTemplate (text/template execution against caller-supplied fields) to test what a .goreleaser.yaml template ACTUALLY resolves to for a given artifact, rather than string-matching its literal YAML source — required whenever a test's real property is 'N distinct outputs', which a literal-text assertion cannot express or protect."
    - "Read the pinned GoReleaser module's actual source (via `go mod download`-into-a-throwaway-module, since goreleaser is a CLI tool dependency of the repo, not a Go import) before writing a name-template config block whose collision behavior is safety-critical — RESEARCH.md's vetted pattern text is not sufficient evidence on its own when two structurally different code paths (signs vs sboms) can silently differ."

key-files:
  created: []
  modified:
    - .goreleaser.yaml
    - internal/upgrade/goreleaser_shape_test.go

key-decisions:
  - "binary_signs.signature uses \"{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json\" instead of RESEARCH.md Pattern 2's \"${artifact}.sigstore.json\" — confirmed against the pinned goreleaser/v2@v2.17.1 module that binary_signs: signs the raw, un-renamed Binary-type artifact (sign_binary.go:80's artifact.ByType(artifact.Binary) filter) and computes the PUBLISHED signature name from env[\"artifact\"] = art.Name (sign.go's signone(), second/publish-naming pass), which is this project's literal `binary: codegraph` value for every platform. The $artifact form would publish one codegraph.sigstore.json for all 4 platforms."
  - "sboms.documents uses \"{{ .ArtifactName }}.spdx.json\" per the plan's cycle-3-review-mandated correction (HIGH-B), confirmed against artifact.go:734's ByBinaryLikeArtifacts (dedupes by path preferring the archives: pipe's renamed UploadableBinary artifact) and empirically verified via a real `task release:dry-run`: dist/artifacts.json shows four distinct codegraph_<tag>_<goos>_<goarch>.spdx.json SBOM records."
  - "Zip archive's files: is left at GoReleaser's default (not overridden) and the config comment documents the real v2.17.1 default set (binary + LICENSE* + README* + CHANGELOG*), correcting RESEARCH.md/D-16's incomplete 'binary + LICENSE + README' description (cycle-3 review MEDIUM) — D-16's binding requirement (no completions/man pages) is unaffected."
  - "Restructured execution into 3 explicit RED/GREEN commit pairs (6 commits total) rather than writing all three tasks' config+tests in one shot, to satisfy the executor's per-task TDD commit protocol precisely — each task's tests were independently confirmed to fail against the PRE-task config before that task's config edit landed."

patterns-established:
  - "Any future .goreleaser.yaml top-level MAPPING block (checksum:, release:, etc.) decodes via parseGoreleaserTopLevelBlock(src, key) into map[string]any; any top-level LIST block (archives:, binary_signs:, sboms:, etc.) gets its own typed struct + parseX/mustX pair, per the parseGoreleaserArchives/parseGoreleaserBinarySigns/parseGoreleaserSBOMs precedent set here."

requirements-completed: [REL-06, REL-09]

coverage:
  - id: D1
    description: "archives: split into id: raw (formats: [binary], name_template byte-unchanged) and id: zip (formats: [zip], same stem) — REL-09's dual-asset shape, with the raw entry's runtime contract with internal/upgrade.releaseAssetName() provably unchanged"
    requirement: "REL-09"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestRawArchiveEntryStaysBinaryFormat"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestZipArchiveSharesRawAssetStem"
        status: pass
      - kind: unit
        ref: "internal/upgrade/verify_release_e2e_test.go#TestReleaseAssetNameMatchesGoReleaser"
        status: pass
      - kind: other
        ref: "task release:dry-run (real goreleaser release --snapshot --skip=publish,sign run: 4 raw binaries file(1)-verified correct object types, 4 .zip archives produced)"
        status: pass
    human_judgment: false
  - id: D2
    description: "checksum.ids: [raw, zip] scopes the checksums file to exactly the 8 downloadable payloads (D-12)"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestChecksumCoversRawAndZipIdsOnly"
        status: pass
    human_judgment: false
  - id: D3
    description: "binary_signs: reproduces today's hand-rolled cosign sign-blob loop declaratively, with a corrected per-platform-distinct signature: template matching internal/upgrade's download contract (assetName + \".sigstore.json\")"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestBinarySignsSidecarMatchesUpgradeContract"
        status: pass
    human_judgment: true
    rationale: "This test proves the CONFIG resolves correctly for all 4 platforms via a standalone text/template execution (necessary, matching the plan's own acceptance criteria) — it does not run the real binary_signs: pipe against a real cosign invocation, which requires plan 01-06's release:dry-run-signed leg (wave 3, not yet executed). The config-only proof is exactly what this plan's <done> criterion asks for; full pipe-level proof is explicitly routed to 01-06."
  - id: D4
    description: "sboms: reproduces today's hand-rolled syft loop declaratively with artifacts: binary explicit and a NAME-derived documents: template producing four distinct <asset>.spdx.json names (D-17, cycle-3 review HIGH-B)"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestSbomsArePerBinaryWithSpdxNames"
        status: pass
      - kind: other
        ref: "task release:dry-run — dist/artifacts.json SBOM-type entries: codegraph_v0.4.0_{linux,darwin}_{amd64,arm64}.spdx.json, four distinct names, confirmed via jq"
        status: pass
    human_judgment: false
  - id: D5
    description: "release: block pins replace_existing_artifacts: true and prerelease: auto, with no name_template/header/footer/draft/disable key (D-06R) — rerun idempotency and prerelease handling as config, not GoReleaser defaults (review HIGH-2)"
    requirement: "REL-06"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestReleaseBlockIsRerunIdempotent"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestReleaseBlockDoesNotRewriteReleaseBody"
        status: pass
    human_judgment: true
    rationale: "The plan itself states this pipe is unexercisable by any dry run — --skip=publish,sign cannot reach the release pipe, and there is no local mode that publishes to a real GitHub Release without publishing to one. The config-pinning is fully proven (static test + task check:goreleaser); the pipe's real behavior is first observed during plan 01-05's actual release, per this plan's own design."

duration: ~45min
completed: 2026-08-08
status: complete
---

# Phase 1 Plan 2: `.goreleaser.yaml` Activation — Archives, Checksum, Signing, SBOMs, Release Summary

**Split `archives:` into `raw`+`zip` id-keyed entries, scoped `checksum:` to both, and activated `binary_signs:`/`sboms:`/`release:` — with a previously-undiscovered per-platform-name-collision bug found and fixed in `binary_signs.signature` along the way, sibling to the one cycle-3 review already caught in `sboms.documents`.**

## Performance

- **Duration:** ~45 min
- **Started:** ~2026-08-08T19:00Z (approximate — session start time was not captured via the record_start_time step)
- **Completed:** 2026-08-08T19:44:00Z
- **Tasks:** 3 of 3 completed
- **Files modified:** 2 (`.goreleaser.yaml`, `internal/upgrade/goreleaser_shape_test.go`)

## Accomplishments

- `archives:` now has two `id:`-keyed entries: `raw` (`formats: [binary]`, `name_template` byte-unchanged — the `internal/upgrade.releaseAssetName()` runtime contract `TestReleaseAssetNameMatchesGoReleaser` already pins) and `zip` (`formats: [zip]`, same stem — REL-09/D-15). `checksum.ids: [raw, zip]` scopes the checksums file to exactly the 8 downloadable payloads (D-12).
- `binary_signs:` and `sboms:` are both live, reproducing today's hand-rolled `cosign sign-blob`/`syft` shell loops (`release.yml`'s assemble job, deleted by plan 01-03) declaratively.
- **Found and fixed a previously-undiscovered HIGH-severity bug**, not caught by any prior cycle-1 or cycle-3 review of this plan: the plan's own literal instruction for `binary_signs.signature` (`"${artifact}.sigstore.json"`, matching RESEARCH.md's Pattern 2 and GoReleaser's own documented usage pattern) publishes **one** `codegraph.sigstore.json` signature asset for **all four platforms**, not four distinct ones — the same collision-and-silent-clobber failure mode cycle-3 review caught in `sboms.documents` (HIGH-B), but in the sign pipe instead of the SBOM pipe, and specific to this project's config (a literal, unparameterized `binary: codegraph` build name shared by all four `builds:` entries). Traced end-to-end against the pinned `goreleaser/v2@v2.17.1` module's `internal/pipe/sign/sign_binary.go` and `sign.go` source, confirmed with a standalone `text/template` probe before writing the fix. See "Deviations from Plan" below.
- `release:` pins `replace_existing_artifacts: true` and `prerelease: auto`, replacing the idempotency and prerelease behaviors the deleted `gh release upload --clobber` step provides today (review HIGH-2), with no `name_template`/`header`/`footer`/`draft`/`disable` key (D-06R — release-please keeps Release authorship).
- All 8 required mutation-RED demonstrations recorded below, redone against each task's actual committed config state (not just the final combined state) after restructuring execution into per-task RED→GREEN commit pairs.
- `task release:dry-run` (real `goreleaser release --snapshot --skip=publish,sign` run) independently confirmed, via `dist/artifacts.json`, that the `sboms:` pipe publishes four distinct `.spdx.json` names — not just a static-config assertion.

## Task Commits

Each task's TDD RED/GREEN phases were committed separately:

1. **Task 1: Dual archives keyed by id — raw stays binary, zip is added alongside (REL-09, D-12, D-15, D-16)**
   - RED: `3410c63` (test)
   - GREEN: `47df9b1` (feat)
2. **Task 2: `binary_signs:` and `sboms:` reproduce today's sidecar names declaratively (D-14, D-17)**
   - RED: `029af36` (test)
   - GREEN: `09b110a` (feat)
3. **Task 3: Pin the `release:` pipe explicitly — rerun idempotency and prerelease handling are config, not GoReleaser defaults (review HIGH-2)**
   - RED: `effe310` (test)
   - GREEN: `90d1ef3` (feat)

## Files Created/Modified

- `.goreleaser.yaml` — `archives:` split into `raw`/`zip`; `checksum.ids: [raw, zip]`; new `binary_signs:` and `sboms:` blocks; new `release:` block; header contract comments (a)/(c) rewritten to describe the now-live pipes
- `internal/upgrade/goreleaser_shape_test.go` — `goreleaserArchive`/`parseGoreleaserArchives`/`mustGoreleaserArchives`; `parseGoreleaserTopLevelBlock`/`mustGoreleaserTopLevelBlock` (generic top-level-mapping decoder, reused across all 3 tasks); `goreleaserBinarySign`/`parseGoreleaserBinarySigns`/`mustGoreleaserBinarySigns`; `goreleaserSBOM`/`parseGoreleaserSBOMs`/`mustGoreleaserSBOMs`; `resolveGoreleaserFieldTemplate` (text/template-based template resolution helper); 7 new tests (`TestRawArchiveEntryStaysBinaryFormat`, `TestZipArchiveSharesRawAssetStem`, `TestChecksumCoversRawAndZipIdsOnly`, `TestParseGoreleaserArchives_NoArchivesBlockIsError`, `TestBinarySignsSidecarMatchesUpgradeContract`, `TestSbomsArePerBinaryWithSpdxNames`, `TestReleaseBlockIsRerunIdempotent`, `TestReleaseBlockDoesNotRewriteReleaseBody`)

## Decisions Made

- **`binary_signs.signature`: `"{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"`, not RESEARCH.md's `"${artifact}.sigstore.json"`.** See "Deviations from Plan" — this is a Rule 1 bug fix against the plan's own literal instruction, discovered and fixed during Task 2 execution, before any commit landed with the broken form.
- **`sboms.documents`: `"{{ .ArtifactName }}.spdx.json"`**, exactly as the plan's action prescribed (cycle-3 review HIGH-B), and additionally verified empirically via a real `task release:dry-run` (not just the static shape test) — `dist/artifacts.json` shows four distinct SBOM records.
- **Zip archive `files:` left at GoReleaser's default**, with the config comment corrected to name the real v2.17.1 default set (`binary + LICENSE* + README* + CHANGELOG*`), per the plan's cycle-3-review-driven correction of D-16's description.
- **Execution restructured into 6 commits (3 RED/GREEN pairs)** rather than one shot per task, to give the executor's TDD protocol a faithful audit trail: each task's tests were independently re-confirmed to fail against the pre-task config before that task's GREEN commit landed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `binary_signs.signature`'s plan-prescribed `${artifact}.sigstore.json` template collides to one signature name for all 4 platforms**

- **Found during:** Task 2, before writing the config (caught by tracing the pinned `goreleaser/v2@v2.17.1` module's source per the plan's own `<read_first>` instruction to verify the SBOM `documents:` template against pinned source — the same scrutiny, applied to `binary_signs:`, surfaced a sibling bug the plan did not anticipate)
- **Issue:** The plan's Task 2 action, and RESEARCH.md's Pattern 2 it was copied from, specify `signature: "${artifact}.sigstore.json"` for `binary_signs:` — GoReleaser's own documented usage pattern. Unlike `sboms:` (which filters `artifacts: binary` via `ByBinaryLikeArtifacts`, preferring the `archives:` pipe's RENAMED `UploadableBinary` artifact), `binary_signs:` filters strictly on `artifact.ByType(artifact.Binary)` (`internal/pipe/sign/sign_binary.go:80`) — the RAW, un-renamed build-output artifact. For the PUBLISHED signature artifact's name, `internal/pipe/sign/sign.go`'s `signone()` resets `env["artifact"] = art.Name` (not `art.Path`) for the second, publish-naming template-resolution pass (confirmed by reading the function line-by-line: the `${artifact}` env var is `art.Path` for the FIRST pass — used only to pick cosign's on-disk write location — and `art.Name` for the SECOND pass, which determines the GitHub release asset filename via `internal/client/github.go`'s `UploadOptions{Name: artifact.Name}`). A raw `Binary`-type artifact's `.Name` is this project's literal, unparameterized `binary: codegraph` build config value for every one of the 4 platforms (`internal/builders/golang/build.go`'s `buildOptionsForTarget`: `bin := tmpl.Apply(build.Binary)`, and `build.Binary` is the literal string `"codegraph"` with no template placeholders in `.goreleaser.yaml`). A `signature: "${artifact}.sigstore.json"` template therefore resolves to the identical string `codegraph.sigstore.json` for all four platforms — confirmed empirically with a standalone Go `text/template`/`os.Expand` probe reproducing GoReleaser's exact two-pass substitution logic before writing the fix. With `release.replace_existing_artifacts: true` (this plan's own Task 3), a real release run would silently CLOBBER 3 of 4 signature sidecars, leaving `codegraph upgrade`'s signature verification broken for 3 of 4 platforms while the release reports success — the exact severity and failure shape of cycle-3 review's HIGH-B finding against `sboms.documents`, but undiscovered by any prior review cycle of this plan (cycle-1, cycle-3) because neither examined the sign pipe's source with the same rigor applied to the SBOM pipe.
- **Fix:** Changed `binary_signs.signature` from `"${artifact}.sigstore.json"` to `"{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"` — a Go-template-FIELD-based template (mirroring the `raw` archive entry's `name_template` character-for-character, plus `.sigstore.json`) instead of the `$artifact` env-var form. `.Os`/`.Arch` are bound from the SAME raw `Binary` artifact's own `Goos`/`Goarch` fields (which ARE correct per-platform, confirmed via `internal/tmpl/tmpl.go`'s `WithArtifact`), so this resolves to four distinct, correct names on both the on-disk-write pass and the publish-naming pass. Verified this resolves byte-identically to `goReleaserAssetName(tag, goos, goarch) + ".sigstore.json"` — the exact string `internal/upgrade` downloads — for all four platforms, and cross-checked against `releaseAssetName()` for the current host's platform, mirroring `TestReleaseAssetNameMatchesGoReleaser`'s own discipline.
- **Files modified:** `.goreleaser.yaml` (the `binary_signs.signature` value and its extensive explanatory config comment, which documents the finding, its evidence, and the fix rationale so a future reader does not reintroduce the collision by "simplifying" the template back to `${artifact}`-based form); `internal/upgrade/goreleaser_shape_test.go` (`TestBinarySignsSidecarMatchesUpgradeContract` asserts the property — four distinct resolved names matching the download contract — via `resolveGoreleaserFieldTemplate`, not a literal-string match against `"${artifact}.sigstore.json"` as a naively-transcribed plan test would have)
- **Verification:** `TestBinarySignsSidecarMatchesUpgradeContract` passes with the fix, and was demonstrated RED against the fix reverted to the broken `${artifact}.sigstore.json` form (see Mutation-RED Demonstrations below — this specific RED run reproduces the exact single-name collision this fix prevents). `task check:goreleaser` and `task release:dry-run` both pass with the fixed config.
- **Committed in:** `09b110a` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug, Rule 1)
**Impact on plan:** The fix is directly load-bearing for REL-06/D-14's own stated top-severity property ("the one contract whose breakage bricks every user's `codegraph upgrade`" — this plan's own threat register, T-01-06). Following the plan's literal text here would have shipped a real regression discovered only at plan 01-06's or 01-05's later gates, or in production. No scope creep: the fix is entirely inside `binary_signs.signature`, this plan's own Task 2 deliverable, and does not touch anything outside this plan's declared file scope.

## Mutation-RED Demonstrations (all 8 required by the plan)

Each recorded live during execution, against that task's actual committed config state, and reverted before the next step:

**Task 1 (3 mutations, demonstrated against commit `47df9b1`):**

1. Raw entry `formats: [binary]` → `[tar.gz]`:
   ```
   === RUN   TestRawArchiveEntryStaysBinaryFormat
       goreleaser_shape_test.go:292: archives[id=raw].formats = [tar.gz], want exactly ["binary"]
   --- FAIL: TestRawArchiveEntryStaysBinaryFormat (0.00s)
   ```
2. Zip entry `name_template` stem changed (`{{ .ProjectName }}` → `{{ .ProjectName }}-archive`):
   ```
   === RUN   TestZipArchiveSharesRawAssetStem
       goreleaser_shape_test.go:319: archives[id=raw].name_template = "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}", archives[id=zip].name_template = "{{ .ProjectName }}-archive_{{ .Tag }}_{{ .Os }}_{{ .Arch }}", want byte-identical
   --- FAIL: TestZipArchiveSharesRawAssetStem (0.00s)
   ```
3. `checksum.ids: [raw, zip]` → `[raw]`:
   ```
   === RUN   TestChecksumCoversRawAndZipIdsOnly
       goreleaser_shape_test.go:354: checksum.ids = [raw], want exactly [raw zip]
   --- FAIL: TestChecksumCoversRawAndZipIdsOnly (0.00s)
   ```

**Task 2 (3 mutations, demonstrated against commit `09b110a`):**

4. `binary_signs.signature` → `"${artifact}.bundle.json"` (the exact collision this task's deviation fix prevents):
   ```
   === RUN   TestBinarySignsSidecarMatchesUpgradeContract
       goreleaser_shape_test.go:550: binary_signs[0].signature resolved for linux/amd64 = "${artifact}.bundle.json", want "codegraph_v1.2.3_linux_amd64.sigstore.json"
       goreleaser_shape_test.go:553: binary_signs[0].signature resolved to a NON-DISTINCT name "${artifact}.bundle.json" for linux/arm64 — this is D-14's collision failure mode
       (... repeated for darwin/amd64, darwin/arm64, and the host cross-check ...)
   --- FAIL: TestBinarySignsSidecarMatchesUpgradeContract (0.00s)
   ```
5. `sboms.artifacts: binary` line deleted:
   ```
   === RUN   TestSbomsArePerBinaryWithSpdxNames
       goreleaser_shape_test.go:596: sboms[0].artifacts = "", want "binary" (absent defaults to "archive", which would break DIST-03)
   --- FAIL: TestSbomsArePerBinaryWithSpdxNames (0.00s)
   ```
6. `sboms.documents` reverted to the path-derived form `"${artifact}.spdx.json"`:
   ```
   === RUN   TestSbomsArePerBinaryWithSpdxNames
       goreleaser_shape_test.go:625: sboms[0].documents[0] resolved for linux/amd64 = "${artifact}.spdx.json", want "codegraph_v1.2.3_linux_amd64.spdx.json"
       goreleaser_shape_test.go:628: sboms[0].documents[0] resolved to a NON-DISTINCT name "${artifact}.spdx.json" for linux/arm64 — this is HIGH-B's collision failure mode
       (... repeated for darwin/amd64, darwin/arm64 ...)
   --- FAIL: TestSbomsArePerBinaryWithSpdxNames (0.00s)
   ```

**Task 3 (2 mutations, demonstrated against commit `90d1ef3`):**

7. `replace_existing_artifacts: true` → `false`:
   ```
   === RUN   TestReleaseBlockIsRerunIdempotent
       goreleaser_shape_test.go:658: release.replace_existing_artifacts = false (bool), want true
   --- FAIL: TestReleaseBlockIsRerunIdempotent (0.00s)
   ```
8. `name_template: "Release {{ .Tag }}"` added to `release:`:
   ```
   === RUN   TestReleaseBlockDoesNotRewriteReleaseBody
       goreleaser_shape_test.go:686: release: block declares forbidden key "name_template" — release-please owns Release authorship (D-06R)
   --- FAIL: TestReleaseBlockDoesNotRewriteReleaseBody (0.00s)
   ```

All 8 reverted immediately after capture; final committed state carries none of these mutations.

## Issues Encountered

- Pre-existing, unrelated test failure observed in a whole-repo `go test ./...` run: `test/wireoracle`'s `TestFrozenTranscriptsMatch/error-unknown-method` fails (`stderr must contain exactly one "codegraph: mcp-session" line, found 0`). This is entirely outside this plan's file scope (`.goreleaser.yaml`, `internal/upgrade/goreleaser_shape_test.go`) and outside `internal/upgrade` — not touched, not caused by this plan's changes, and not auto-fixed per the executor's scope-boundary rule. Logged here for visibility; not fixed.

## User Setup Required

None. No external service configuration required.

## Next Phase Readiness

- `.goreleaser.yaml` is now the single declarative definition of the raw+zip archive pair, checksum scope, per-binary signing, per-binary SBOMs, and rerun/prerelease release handling — everything downstream of compilation that plan 01-03 (concurrent, same wave, `.github/workflows/release.yml`/`Taskfile.yml`/`CONTRIBUTING.md`) needs to delete the hand-rolled shell equivalent of.
- The `binary_signs.signature` fix is a hard prerequisite for plan 01-06's `release:dry-run-signed` leg (wave 3) to produce a meaningful result — had the collision shipped, 01-06 would have caught it there (per its own design as the dynamic-proof gate for exactly this defect class), but at a later, more expensive point in the pipeline than this plan's own static shape test.
- Plan 01-05's Task 2 checkpoint is where `release:`'s rerun-idempotency and prerelease-handling config is observed against a REAL GitHub Release for the first time — this plan proves only that the config is present, correctly shaped, and passes `goreleaser check`.
- No blockers for 01-03, 01-05, or 01-06.

---
*Phase: 01-cross-compile-spike-goreleaser-release-migration*
*Completed: 2026-08-08*

## Self-Check: PASSED

- FOUND: `.goreleaser.yaml`
- FOUND: `internal/upgrade/goreleaser_shape_test.go`
- FOUND: `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-02-SUMMARY.md`
- FOUND commits: `3410c63`, `47df9b1`, `029af36`, `09b110a`, `effe310`, `90d1ef3` (all present in `git log --oneline`)
