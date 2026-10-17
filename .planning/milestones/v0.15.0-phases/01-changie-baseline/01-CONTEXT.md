# Phase 1: Changie Baseline - Context

**Gathered:** 2026-09-25
**Status:** Ready for planning
**Mode:** auto (`.planning/config.json` `mode: yolo`) — every choice below is the recommended option, logged for audit in `01-DISCUSSION-LOG.md`.

<domain>
## Phase Boundary

The repository gains a changie configuration and baseline that answers `v0.14.0` as the latest version and reproduces today's `CHANGELOG.md` exactly, so every later phase writes fragments against a fixed vocabulary and no history is rewritten. Deliverables: `.changie.yaml`, `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep`, the version seed files under `.changes/`, a pinned changie with one recorded invocation path for CI and contributors, and committed proofs for CHG-03 and CHG-04.

Out of this phase: the release workflow, the fragment-required gate, release-please retirement, the phase-close capability, doc prose rewrites (`DOCS-12`), and any regeneration of historical entries (`CHG-05`, v2).

</domain>

<decisions>
## Implementation Decisions

### History placement (how pre-v0.14.0 entries survive `changie merge`)
- **D-01:** Every existing release block in `CHANGELOG.md` (fourteen `## [x.y.z]` sections, `0.2.0` through `0.14.0`) becomes its own verbatim version file `.changes/vX.Y.Z.md`, byte-sliced from today's file. `.changes/v0.14.0.md` is one of them, not a blob carrying the others. — **Reversibility:** reversible — plain files; a different placement is a move, no contract depends on the layout.
  - Why: changie's config reference (https://changie.dev/config/) has `headerPath` for the merged changelog and only per-batch `versionHeaderPath` / `versionFooterPath` / `footerFormat`; there is no merged-changelog footer. `changie merge` therefore emits header + every `.changes/v*.md` and nothing else, so any history outside a version file is dropped on the first merge. Placing history in `header.tpl.md` is rejected: the header renders above the versions, inverting the order.
  - "Baseline only, no historical rewrite" is satisfied because the content of each seed is verbatim; only its location changes. Seed content is NOT re-rendered through `versionFormat` / `changeFormat` — those apply to batches from this phase on.
- **D-02:** The byte-reproduction question is settled by a real `changie merge --dry-run`, not by reading docs (ROADMAP Phase 1 Notes). The researcher runs it against the seeded layout and reports which `newlines` settings (`beforeChangelogVersion`, `afterChangelogVersion`, `afterChangelogHeader`, `endOfVersion`) reproduce the boundaries. If no `newlines` combination reproduces the file byte-for-byte, the fallback is to carry the boundary blank lines inside the seed files themselves (content still verbatim, only whitespace moved) and record that in the plan. Trailing byte of today's file is a single `\n` after the last entry.
- **D-03:** `changie latest` answering `v0.14.0` follows from the seed filenames (changie derives it from the highest semver among `.changes/v*.md`); the `v` prefix comes from the filename, matching the tag scheme. Historical headings inside the seeds keep their release-please form `## [0.14.0](compare…) (date)`; new entries from this phase forward use the `versionFormat` from the design note (`## [vX.Y.Z](releases/tag/…) — date`). That visible format change for *new* entries is accepted; it is locked by CHG-01, not re-litigated here.

### Changie pinning and invocation
- **D-04:** changie is pinned as a Go `tool` directive in a NEW isolated modfile `go.tool-changie.mod` (`tool github.com/miniscruff/changie`), v1.26.0 or the newest tag at plan time, and invoked through a Taskfile variable in the existing `GO_TOOL_*` style (`GOWORK=off go tool -modfile=go.tool-changie.mod changie …`). — **Reversibility:** costly — every invocation site (Taskfile targets, ci.yml step bodies, Phase 5's release workflow, the Phase 2 capability's `changie new` call) names this path; moving it later touches all of them.
  - Why a modfile at all: `.github/actions/install-task/action.yml` (D-04) requires every CI build tool to be built from the checksum-verified module proxy against a pinned `-modfile`, never a GitHub Releases download, marketplace action, or install script. `miniscruff/changie-action` and `brew install changie` are therefore rejected for CI; contributors get the identical command body, satisfying "contributor and CI run identical command bodies" (CONTRIBUTING.md:122).
  - Why a separate modfile rather than `go.tool.mod`: `go.tool.mod` already carries goreleaser's graph (237 net-new modules); actionlint lost a version bid to that graph and had to be isolated (`go.tool-lint.mod` header). A new modfile costs one file plus one registration and cannot lose an MVS bid to goreleaser. Co-locating in `go.tool.mod` is permitted only if the executor measures it live (compiles, no lost bids, negligible module growth) and records the measurement — the default is the separate file.
  - The new modfile MUST be registered in `isolatedModfilePaths` (`internal/upgrade/taskfile_shape_test.go`) and MUST carry a header comment stating the isolation rationale, or `TestToolModfilesRemainIsolated` / `TestToolModfilesPopulationMatchesDisk` fail. `GOWORK=off` on every invocation.
- **D-05:** "One recorded install path for CI and one for contributors" (CHG-01) is discharged in this phase by: the `go.tool-changie.mod` header, a `changie` wrapper target in `Taskfile.yml` whose `desc` names the path, and adding changie to the existing tool bullet in CONTRIBUTING.md (line 124: "`task`, `goreleaser`, and `actionlint` build on demand…"). That bullet edit is a value fill; the §Pull requests prose rewrite stays with `DOCS-12` in Phase 5. Neither Dependabot nor Renovate manages tool modfiles — the bump is manual, say so in the header like the siblings do.

### Proof harness (CHG-03 and CHG-04)
- **D-06:** Proofs are split by nature. Static config-shape assertions are a Go test in the repo-shape guard package (`internal/upgrade`, beside `taskfile_shape_test.go`): `.changie.yaml` declares exactly the five kinds `Breaking`/`Features`/`Fixes`/`Performance`/`Dependencies` with `auto` = minor/minor/patch/patch/patch, the flip-to-major-at-1.0 note is present in the file, the `PR` custom is `type: int`, `minInt: 1`, and NOT `optional`, the three format strings equal the design note's, `header.tpl.md` and `unreleased/.gitkeep` exist, and the set of `.changes/v*.md` files equals the set of `## [x.y.z]` headings in `CHANGELOG.md` (set equality, both directions). This test lands RED-first per rule `x1cjy9vyhq` (test commit before implementation, `--- FAIL` transcript pasted in the SUMMARY; never gate on `gsd-tools check tdd-red-evidence`).
- **D-07:** Live-tool proofs are one Taskfile target `check:changie` in the drift-guard shape of `docs:cli:drift` (Taskfile.yml:494): print a positive count line BEFORE comparing (number of seeded version files, floor 14), run in a scratch copy so the source tree is never mutated, and fail with a named `::error::` on every path. It runs, in order: `changie latest` equals `v0.14.0`; `changie next auto` with an empty `unreleased/` exits non-zero AND its stderr contains changie's own "nothing to release"-class message (the exact text captured at research time — a bare non-zero exit is not proof); `changie merge --dry-run` piped to `cmp` against `CHANGELOG.md`; `CI=true changie new -k Fixes -b "…" -m PR=1` writes exactly one `.changes/unreleased/*.yaml` in the scratch copy; `changie new` with no `-m PR=` is refused with the field named in stderr; `changie new -k Undeclared …` is refused with the kind named in stderr. Each refusal asserts on the message, so it is confirmed to have executed rather than inferred from a green exit (CHG-04, rule `84d1gfpywd`). — **Reversibility:** reversible.
- **D-08:** `check:changie` is wired into `ci.yml`'s `test` job as `run: task check:changie` immediately after `docs:cli:drift`, so the existing `TestWorkflowRunBodiesInvokeTask` and `TestGateStancesStated` guards cover it with no exception entry. The RED demonstration for the byte-reproduction check (success criterion 1) is a one-byte mutation of `CHANGELOG.md` in a scratch copy, shown failing, reverted byte-cleanly, and pasted into the plan's SUMMARY — never a committed mutation.

### Changelog header
- **D-09:** `.changes/header.tpl.md` renders exactly what today's file starts with: `# Changelog` followed by the same blank-line spacing, and nothing else. Result: after this phase `CHANGELOG.md` is byte-identical to `main` as a whole file, which is the strongest available form of "no historical rewrite" and makes the `cmp` in D-07 unambiguous. — **Reversibility:** reversible.
  - Rejected: a "generated by changie, do not edit" preamble or HTML comment in the header (it would render or change bytes at the top of the file). The tool-ownership note lives as comments in `.changie.yaml` now and in CONTRIBUTING prose in Phase 5.

### Fragment vocabulary and validation (locked, recorded for downstream)
- **D-10:** Kinds, `auto` bumps, the required `PR` field and the three format strings are locked by CHG-01 and the design note; this phase copies them, it does not redesign them. The `changeFormat` conditional `{{if .Custom.PR}}` from the note is kept as written — with `PR` required it is inert, and it costs nothing.
- **D-11:** The known hazard that a phase-close fragment has no PR number (ROADMAP Notes hazard 4; memory `r19fvsz4tf`) is NOT solved by making `PR` optional here — CHG-04 requires the missing-`PR` refusal. How Phase 2's capability obtains a number (for example, a draft milestone PR opened at milestone start) is Phase 2's decision. Phase 1 must leave `optional` unset on `PR`.

### Claude's Discretion
- Fragment filename format (`fragmentFileFormat`) — default is fine unless a collision surfaces.
- Exact wording of Taskfile `desc` blocks and modfile header, following the sibling files' voice.
- Whether the Go test parses `.changie.yaml` with the YAML library already in the module graph (`go.yaml.in/yaml/v3` is used by `web:deps:strict`) or with a minimal line scan — choose whichever keeps the assertion exact.
- Order of the six checks inside `check:changie` beyond the dependency that `merge --dry-run` runs before any `changie new` in the scratch copy.

### Folded Todos
- **Adopt changie for changelog + version and replace release-please with a changie release-PR workflow** (`.planning/todos/pending/2026-09-25-adopt-changie-replace-release-please.md`, score 0.9, `resolves_phase: 5`). This phase delivers its checklist step 1 ("Config + baseline") exactly as written; the todo stays pending for Phase 5, which closes it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design and requirements
- `.planning/notes/changie-release-management.md` — the adopted design: `.changie.yaml` sketch (kinds, `auto` map, `PR` custom, three format strings), research ledger (what is admitted vs corrected), open questions. §"`.changie.yaml` sketch" is the source the config is copied from.
- `.planning/REQUIREMENTS.md` §Changie Baseline — CHG-01..CHG-04, the acceptance text this phase is verified against.
- `.planning/ROADMAP.md` §Phase 1 — success criteria 1–4 and the Notes paragraph naming the merge-regeneration research question.
- `.planning/todos/pending/2026-09-25-adopt-changie-replace-release-please.md` — checklist step 1 and the two guard reminders.
- https://changie.dev/config/ — configuration reference (external; fetched 2026-09-25). Relevant facts: no merged-changelog footer key; `newlines` sub-keys; `custom[].optional`; `kinds[].auto` semantics ("only none changes is not a valid bump and will fail to batch").

### Baseline oracle
- `CHANGELOG.md` — the byte oracle. Fourteen `## [x.y.z]` blocks (0.2.0 → 0.14.0), file ends with a single `\n`. Never hand-edited.
- `.release-please-manifest.json` / `release-please-config.json` — stay untouched in this phase (retired in Phase 5, `REL-*`); read only to confirm `0.14.0` is the current manifest version.

### Repo conventions the deliverables must follow
- `go.tool.mod` and `go.tool-lint.mod` header comments — the isolation rationale and `GOWORK=off` requirement every tool modfile must restate.
- `.github/actions/install-task/action.yml` — D-04: build tools come from the module proxy against a pinned modfile, never a download; the rule D-04 in this context inherits.
- `internal/upgrade/taskfile_shape_test.go` — `isolatedModfilePaths` (register the new modfile), `TestToolModfilesRemainIsolated`, `TestToolModfilesPopulationMatchesDisk`, `TestWorkflowRunBodiesInvokeTask`, `TestGateStancesStated`; the Go test from D-06 lives beside these.
- `Taskfile.yml` — `GO_TOOL*` vars (lines 9–12) for the invocation style; `docs:cli:drift` (line 494) for the drift-guard shape D-07 copies; `check:goreleaser` (line 3931) for the "check class" desc convention.
- `.github/workflows/ci.yml` — `test` job, the `docs:cli:drift` step (line 203) that `check:changie` follows.
- `CONTRIBUTING.md` lines 122–128 — the tool bullet that gains changie.

### Standing rules (engram, `rule:repo:github.com/seanb4t/codegraph-go`)
- `84d1gfpywd` — every guard carries a positive assertion that it did its work.
- `x1cjy9vyhq` — Go RED evidence is the test commit before the implementation plus the pasted `--- FAIL` transcript.
- `f18zrdsgx5` — never `[ci skip]` in a commit message.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `docs:cli:drift` / `proto:drift` Taskfile targets: the scratch-dir, count-before-compare, named-error drift-guard shape that `check:changie` copies.
- `GO_TOOL`, `GO_TOOL_LINT`, `GO_TOOL_PROTO`, `GO_TOOL_GOLANGCI` Taskfile vars: add `GO_TOOL_CHANGIE` in the same form.
- `internal/upgrade/taskfile_shape_test.go` helpers (`mustToolModfileHeaderComment`, `parseGoModToolPackages`, workflow step parsers): reuse for the modfile and ci.yml assertions rather than re-parsing.

### Established Patterns
- Four isolated tool modfiles already exist (`go.tool.mod`, `go.tool-lint.mod`, `go.tool-proto.mod`, `go.tool-golangci.mod`); a fifth follows the same header + registration protocol, enforced by population-vs-disk set equality.
- CI steps must invoke `task <target>`; an inline `go` invocation in a workflow body fails `TestWorkflowRunBodiesInvokeTask` unless listed with a reason.
- Guards report a count before asserting and fail loud on zero (`docs:cli:drift`, `check-ruleset-drift.sh`).
- `CHANGELOG.md` is exempt from `require-issue-link` and the PR-template policy as a tool-owned file; the new `.changes/` tree is not yet exempt anywhere (Phase 4/5 concern for the release PR, not this phase).

### Integration Points
- `ci.yml` `test` job gains one step after `docs:cli:drift`.
- `Taskfile.yml` gains `changie` (wrapper) and `check:changie` targets.
- `CONTRIBUTING.md` tool bullet gains changie (value fill only).
- `.github/workflows/release-please.yml` keeps running unchanged through this phase; it will continue to manage `CHANGELOG.md` until Phase 5 retires it, so no release may be cut between Phase 1 and Phase 5 without reconciling the two writers — flag this in the plan's verification notes.

</code_context>

<specifics>
## Specific Ideas

- The phase's proof that history survived is whole-file byte identity of `CHANGELOG.md` against `main` after seeding, plus `changie merge --dry-run | cmp - CHANGELOG.md` succeeding; the RED case is a one-byte mutation in a scratch copy.
- The refusal proofs read changie's stderr for the named field/kind; the exact strings are captured by the researcher from a real run of v1.26.0 and asserted verbatim.
- `changie next auto` with an empty `unreleased/` directory (only `.gitkeep`) is the expected-failure baseline; the researcher confirms `.gitkeep` is ignored by changie's `*.yaml` glob.

</specifics>

<deferred>
## Deferred Ideas

- Supplying a PR number to fragments written at phase close before a PR exists — Phase 2 (`CAP-*`), see D-11.
- Adding `.changes/**` to the `require-issue-link` / `pr_template_policy.py` exemption lists for the release PR — Phase 4/5.
- Regenerating historical entries uniformly from GitHub Release notes — `CHG-05`, v2.
- Reconciling release-please and changie both writing `CHANGELOG.md` if a release were cut mid-milestone — Phase 5 retires release-please; until then, no release.

### Reviewed Todos (not folded)
- `2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md` (score 0.6) — keyword match on "phase/check/main" only; bench runner integrity is unrelated to changie. Stays pending.
- `2026-09-25-reply-on-gh-85-with-reshaped-contract.md` (score 0.6) — keyword match on "phase/plan/requirements"; GH #85 server-mode reply belongs to the next milestone. Stays pending.

</deferred>

---

*Phase: 01-changie-baseline*
*Context gathered: 2026-09-25 via /gsd-discuss-phase 1 (auto mode)*
