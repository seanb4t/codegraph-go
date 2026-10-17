# Phase 1: Changie Baseline - Research

**Researched:** 2026-09-25
**Domain:** Go build-tooling pin (isolated `-modfile`), changelog-tool migration (release-please → changie), config-shape + live-tool CI guards
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Every existing release block in `CHANGELOG.md` (fourteen `## [x.y.z]` sections, `0.2.0` through `0.14.0`) becomes its own verbatim version file `.changes/vX.Y.Z.md`, byte-sliced from today's file. `.changes/v0.14.0.md` is one of them, not a blob carrying the others. — **Reversibility:** reversible — plain files; a different placement is a move, no contract depends on the layout.
  - Why: changie's config reference (https://changie.dev/config/) has `headerPath` for the merged changelog and only per-batch `versionHeaderPath` / `versionFooterPath` / `footerFormat`; there is no merged-changelog footer. `changie merge` therefore emits header + every `.changes/v*.md` and nothing else, so any history outside a version file is dropped on the first merge. Placing history in `header.tpl.md` is rejected: the header renders above the versions, inverting the order.
  - "Baseline only, no historical rewrite" is satisfied because the content of each seed is verbatim; only its location changes. Seed content is NOT re-rendered through `versionFormat` / `changeFormat` — those apply to batches from this phase on.
- **D-02:** The byte-reproduction question is settled by a real `changie merge --dry-run`, not by reading docs (ROADMAP Phase 1 Notes). The researcher runs it against the seeded layout and reports which `newlines` settings (`beforeChangelogVersion`, `afterChangelogVersion`, `afterChangelogHeader`, `endOfVersion`) reproduce the boundaries. If no `newlines` combination reproduces the file byte-for-byte, the fallback is to carry the boundary blank lines inside the seed files themselves (content still verbatim, only whitespace moved) and record that in the plan. Trailing byte of today's file is a single `\n` after the last entry.
- **D-03:** `changie latest` answering `v0.14.0` follows from the seed filenames (changie derives it from the highest semver among `.changes/v*.md`); the `v` prefix comes from the filename, matching the tag scheme. Historical headings inside the seeds keep their release-please form `## [0.14.0](compare…) (date)`; new entries from this phase forward use the `versionFormat` from the design note (`## [vX.Y.Z](releases/tag/…) — date`). That visible format change for *new* entries is accepted; it is locked by CHG-01, not re-litigated here.
- **D-04:** changie is pinned as a Go `tool` directive in a NEW isolated modfile `go.tool-changie.mod` (`tool github.com/miniscruff/changie`), v1.26.0 or the newest tag at plan time, and invoked through a Taskfile variable in the existing `GO_TOOL_*` style (`GOWORK=off go tool -modfile=go.tool-changie.mod changie …`). — **Reversibility:** costly — every invocation site names this path; moving it later touches all of them.
  - Why a modfile at all: `.github/actions/install-task/action.yml` requires every CI build tool to be built from the checksum-verified module proxy against a pinned `-modfile`, never a GitHub Releases download, marketplace action, or install script.
  - Why a separate modfile rather than `go.tool.mod`: co-locating in `go.tool.mod` is permitted only if the executor measures it live (compiles, no lost bids, negligible module growth) and records the measurement — the default is the separate file.
  - The new modfile MUST be registered in `isolatedModfilePaths` (`internal/upgrade/taskfile_shape_test.go`) and MUST carry a header comment stating the isolation rationale, or `TestToolModfilesRemainIsolated` / `TestToolModfilesPopulationMatchesDisk` fail. `GOWORK=off` on every invocation.
- **D-05:** "One recorded install path for CI and one for contributors" (CHG-01) is discharged by: the `go.tool-changie.mod` header, a `changie` wrapper target in `Taskfile.yml` whose `desc` names the path, and adding changie to the existing tool bullet in CONTRIBUTING.md (line 124). That bullet edit is a value fill; the §Pull requests prose rewrite stays with `DOCS-12` in Phase 5. Neither Dependabot nor Renovate manages tool modfiles — the bump is manual, say so in the header like the siblings do.
- **D-06:** Proofs are split by nature. Static config-shape assertions are a Go test in `internal/upgrade`, beside `taskfile_shape_test.go`: `.changie.yaml` declares exactly the five kinds `Breaking`/`Features`/`Fixes`/`Performance`/`Dependencies` with `auto` = minor/minor/patch/patch/patch, the flip-to-major-at-1.0 note is present in the file, the `PR` custom is `type: int`, `minInt: 1`, and NOT `optional`, the three format strings equal the design note's, `header.tpl.md` and `unreleased/.gitkeep` exist, and the set of `.changes/v*.md` files equals the set of `## [x.y.z]` headings in `CHANGELOG.md` (set equality, both directions). This test lands RED-first per rule `x1cjy9vyhq` (test commit before implementation, `--- FAIL` transcript pasted in the SUMMARY; never gate on `gsd-tools check tdd-red-evidence`).
- **D-07:** Live-tool proofs are one Taskfile target `check:changie` in the drift-guard shape of `docs:cli:drift`: print a positive count line BEFORE comparing (number of seeded version files, floor 14), run in a scratch copy so the source tree is never mutated, and fail with a named `::error::` on every path. It runs, in order: `changie latest` equals `v0.14.0`; `changie next auto` with an empty `unreleased/` exits non-zero AND its stderr contains changie's own "nothing to release"-class message; `changie merge --dry-run` piped to `cmp` against `CHANGELOG.md`; `CI=true changie new -k Fixes -b "…" -m PR=1` writes exactly one `.changes/unreleased/*.yaml` in the scratch copy; `changie new` with no `-m PR=` is refused with the field named in stderr; `changie new -k Undeclared …` is refused with the kind named in stderr. Each refusal asserts on the message. — **Reversibility:** reversible.
- **D-08:** `check:changie` is wired into `ci.yml`'s `test` job as `run: task check:changie` immediately after `docs:cli:drift`, so the existing `TestWorkflowRunBodiesInvokeTask` and `TestGateStancesStated` guards cover it with no exception entry. The RED demonstration for the byte-reproduction check is a one-byte mutation of `CHANGELOG.md` in a scratch copy, shown failing, reverted byte-cleanly, and pasted into the plan's SUMMARY — never a committed mutation.
- **D-09:** `.changes/header.tpl.md` renders exactly what today's file starts with: `# Changelog` followed by the same blank-line spacing, and nothing else. Rejected: a "generated by changie, do not edit" preamble or HTML comment in the header (it would render or change bytes at the top of the file).
- **D-10:** Kinds, `auto` bumps, the required `PR` field and the three format strings are locked by CHG-01 and the design note; this phase copies them, it does not redesign them. The `changeFormat` conditional `{{if .Custom.PR}}` from the note is kept as written.
- **D-11:** The known hazard that a phase-close fragment has no PR number is NOT solved by making `PR` optional here — CHG-04 requires the missing-`PR` refusal. Phase 1 must leave `optional` unset on `PR`.

### Claude's Discretion

- Fragment filename format (`fragmentFileFormat`) — default is fine unless a collision surfaces.
- Exact wording of Taskfile `desc` blocks and modfile header, following the sibling files' voice.
- Whether the Go test parses `.changie.yaml` with the YAML library already in the module graph (`go.yaml.in/yaml/v3`) or with a minimal line scan — choose whichever keeps the assertion exact.
- Order of the six checks inside `check:changie` beyond the dependency that `merge --dry-run` runs before any `changie new` in the scratch copy.

### Deferred Ideas (OUT OF SCOPE)

- Supplying a PR number to fragments written at phase close before a PR exists — Phase 2 (`CAP-*`), see D-11.
- Adding `.changes/**` to the `require-issue-link` / `pr_template_policy.py` exemption lists for the release PR — Phase 4/5.
- Regenerating historical entries uniformly from GitHub Release notes — `CHG-05`, v2.
- Reconciling release-please and changie both writing `CHANGELOG.md` if a release were cut mid-milestone — Phase 5 retires release-please; until then, no release.
- The release workflow, the fragment-required gate, release-please retirement, the phase-close capability, doc prose rewrites (`DOCS-12`), and any regeneration of historical entries (`CHG-05`, v2) are all out of THIS phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CHG-01 | `.changie.yaml` exists with kinds `Breaking`/`Features`/`Fixes`/`Performance`/`Dependencies`, correct `auto` bumps, a required custom `PR` field (`type: int`, `minInt: 1`), the version/kind/change formats from the design note; `changie` pinned (v1.26.0+) with one recorded install path for CI and contributors | Standard Stack, Architecture Pattern 2 (isolated modfile), Package Legitimacy Audit confirm v1.26.0 resolves live; Common Pitfall 3 (forbiddenToolPackages) |
| CHG-02 | `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep`, `.changes/v0.14.0.md` exist; `CHANGELOG.md` keeps every entry verbatim below the header, byte-diffable against `main` | Architecture Pattern 1 (byte-sliced seeding), Code Examples (`changie merge --dry-run` byte-identity proof) |
| CHG-03 | `changie latest` answers `v0.14.0`; `changie next auto` with no fragments fails with the expected outcome; `changie merge --dry-run` reproduces `CHANGELOG.md` byte-for-byte — captured as a test/task, not a one-off transcript | Code Examples (all three commands run live with exact output captured); Validation Architecture (`check:changie` target + RED/GREEN mutation cycle) |
| CHG-04 | `CI=true changie new -k <Kind> -b "<sentence>" -m PR=<n>` lands as `.changes/unreleased/*.yaml`; a fragment missing `PR` or using an undeclared kind is refused — proven both ways | Code Examples (all three outcomes run live with exact stderr text captured); Common Pitfall 4 (fragment YAML serialization) |
</phase_requirements>

## Summary

This phase is almost entirely de-risked by direct experimentation rather than documentation reading, and every one of D-02's open questions has now been answered by a real `changie v1.26.0` run in a scratch copy (never the repo working tree). The single headline finding: **`changie merge --dry-run` reproduces `CHANGELOG.md` byte-for-byte** with the byte-sliced 14-version seed layout described in CONTEXT.md D-01, using **exactly one** non-default `newlines` setting — `afterChangelogHeader: 1` — and no other `newlines` key. This was verified with `cmp` reporting zero difference, confirmed with a RED demonstration (a one-byte header mutation, `cmp` failing at char 4, byte-cleanly reverted, `cmp` passing again).

`changie latest` answers `v0.14.0` from the seed filenames with no extra configuration. `changie next auto` against an empty (`.gitkeep`-only) `unreleased/` exits 1 with stderr `Error: no unreleased changes found for automatic bumping` — this repo's `.gitkeep` is confirmed ignored by changie's fragment glob (a real `Fixes-*.yaml` fragment coexisting with `.gitkeep` computed `v0.14.1` cleanly). `CI=true changie new -k Fixes -b "..." -m PR=1` writes exactly one `.changes/unreleased/*.yaml` non-interactively. A missing `PR` is refused with stderr `Error: custom missing and prompt is disabled: custom key 'PR'`; an undeclared kind is refused with stderr `Error: invalid kind: Undeclared`; both direction-of-failure texts are needed verbatim in the check:changie assertions since a bare non-zero exit is not proof (rule `84d1gfpywd`).

On the Go-tooling side, `go.tool-changie.mod` (an isolated tool modfile, mirroring `go.tool-lint.mod`'s header/registration protocol) was built and measured live: standalone it resolves 64 modules total (63 net-new plus the tools-changie module itself — smaller than task/goreleaser's graph, larger than actionlint's 16, dominated by Cobra + a Charm/Bubbletea prompt library + Masterminds/sprig templating that changie pulls in for its interactive mode). A **second, unplanned measurement** was also taken per D-04's own escape-hatch text ("permitted only if the executor measures it live... and records the measurement"): co-locating changie's tool directive directly into a scratch copy of the real `go.tool.mod` cost only **+4 net-new modules** (1002 → 1006) with **zero lost MVS bids** — `goreleaser/v2` stayed pinned at `v2.17.1` and `sigstore/cosign/v3` stayed at `v3.1.1`, and all three tools (`task`, `goreleaser`, `changie`) built successfully from the co-located modfile. This is a materially different result from `go.tool-proto.mod`'s documented buf-vs-goreleaser silent-downgrade experience, and from the root `go.mod`'s 237-module cost — changie's own dependency graph turns out to overlap heavily with what task/goreleaser already resolve. D-04 still locks the separate-modfile path as the phase's default deliverable; this measurement is recorded here so the plan (or a future executor) does not have to re-derive it if it ever revisits the co-location escape hatch.

**Primary recommendation:** Seed `.changes/v0.2.0.md` … `.changes/v0.14.0.md` as exact byte slices of today's `## [x.y.z]` blocks (including each block's own trailing blank-line separator, which the release-please format already carries), write `.changes/header.tpl.md` as exactly `# Changelog\n`, set `newlines.afterChangelogHeader: 1` in `.changie.yaml` and no other `newlines` key, pin changie v1.26.0 in a new `go.tool-changie.mod` + `go.tool-changie.sum` pair registered in `isolatedModfilePaths` and `forbiddenToolPackages`, and build `check:changie` as a `docs:cli:drift`-shaped Taskfile target that copies `.changie.yaml` + `.changes/` into a `mktemp -d` scratch directory before running any of the six checks.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Changelog content authority (`.changie.yaml`, seed files, header) | Build/Release tooling (repo root, tool-owned files) | — | `CHANGELOG.md` is generated output; the config and seeds are the only hand-authored inputs, same tier as `.goreleaser.yaml` |
| `changie` binary pin | Go tool-modfile tier (`go.tool-changie.mod`) | CI workflow tier (`ci.yml` invocation) | Mirrors the four existing isolated tool modfiles exactly — a build-tool binary, never a runtime import of the main module |
| Config-shape assertions (CHG-01/02 static checks) | Go test tier (`internal/upgrade` package) | — | Same package/tier as `taskfile_shape_test.go`'s existing D-01/D-03/D-07 guards; no new package |
| Live-tool proofs (CHG-03/CHG-04) | Taskfile tier (`check:changie` target) | CI workflow tier (`ci.yml` `test` job step) | Drift-guard shape (`docs:cli:drift`) already lives at this tier; CI just invokes the Taskfile target per D-01/D-02 single-definition rule |
| Fragment writing (`changie new`) | CLI/local + CI tier | — | Out of this phase's scope for automation (Phase 2's capability); this phase only proves the command shape works |

## Package Legitimacy Audit

`gsd-tools query package-legitimacy check` supports only `npm`/`pypi`/`crates` ecosystems — it does not cover Go modules, so this audit was performed manually against the Go module proxy and GitHub, per the protocol's ecosystem-specific fallback.

| Package | Registry | Age | Downloads/Activity | Source Repo | Verdict | Disposition |
|---------|----------|-----|---------------------|--------------|---------|-------------|
| `github.com/miniscruff/changie` | Go module proxy (`proxy.golang.org`) | Repo created 2020-12-05 (~5.8 yr) `[VERIFIED: api.github.com/repos/miniscruff/changie, fetched this session]` | 910 GitHub stars, `pushed_at: 2026-09-19` (6 days before this research), `open_issues_count: 11`, not archived `[VERIFIED: api.github.com/repos/miniscruff/changie]` | `github.com/miniscruff/changie`, MIT license `[VERIFIED: api.github.com/repos/miniscruff/changie license.spdx_id=MIT]` | OK | Approved |

**Registry resolution proof:** `GOBIN=<scratch>/bin GOFLAGS=-mod=mod GOWORK=off go install github.com/miniscruff/changie@v1.26.0` succeeded (exit 0) against the real module proxy; `go version -m` on the resulting binary confirms `mod github.com/miniscruff/changie v1.26.0 h1:4Tlgh4r7XrldjJ6+jsaU+PMQCa2p1E3N++doe8KYR2I=` `[VERIFIED: live go install + go version -m, this session]`. The GitHub Releases API independently confirms `tag_name: v1.26.0`, `published_at: 2026-08-20T21:58:36Z` `[VERIFIED: api.github.com/repos/miniscruff/changie/releases/latest]` — matching the design note's `notes/changie-release-management.md` admitted claim exactly.

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

**Note on package-name provenance:** the package name `github.com/miniscruff/changie` originates from `notes/changie-release-management.md` (a prior `/gsd-explore` research session), not from this session's independent discovery — per the package-name provenance rule this is `[ASSUMED]` in origin, but its existence, version, license and maintenance activity are now `[VERIFIED]` in this session via the module proxy and GitHub API directly (not merely "passes `npm view`"-equivalent).

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/miniscruff/changie` | v1.26.0 `[VERIFIED: go install + go version -m + GitHub Releases API, this session]` | Changelog fragment tool: authors changelog entries as committed YAML fragments, derives version bumps from declared `kinds[].auto` | Locked by CONTEXT D-04/CHG-01; only actively-maintained fragment-based changelog tool that already has a documented, working `--release-notes`-file integration with GoReleaser (design note, `[CITED: goreleaser.com/customization/publish/scm]`) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.yaml.in/yaml/v3` | already in module graph (used by `web:deps:strict`, confirmed present as a transitive dep of `go.tool-lint.mod`/`go.tool-golangci.mod`) | YAML decode for the D-06 Go test's `.changie.yaml` shape assertion, if the test parses YAML rather than line-scanning | Only if the plan chooses the "parse with the YAML library" branch of Claude's Discretion; a minimal line-scan (matching `taskfile_shape_test.go`'s existing `parseTaskBlocks`-style regex parsers) is equally valid and keeps this test independent of the YAML library's own compatibility surface |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `miniscruff/changie` pinned via isolated Go tool-modfile | `miniscruff/changie-action` (GitHub Action) | Rejected outright by D-04/`.github/actions/install-task/action.yml`'s own rule: every CI build tool comes from the checksum-verified module proxy against a pinned `-modfile`, never a marketplace action |
| `miniscruff/changie-action` / `brew install changie` | pinned local Go tool-modfile build | Rejected: violates "contributor and CI run identical command bodies" (CONTRIBUTING.md:122) and D-04's action.yml precedent |

**Installation:**
```bash
# Contributor and CI both run this exact body (no separate install step —
# `go tool` builds on demand, same as task/goreleaser/actionlint):
GOWORK=off go tool -modfile=go.tool-changie.mod changie <subcommand>
```

**Version verification:** `go install github.com/miniscruff/changie@v1.26.0` resolved successfully against the live module proxy this session (see Package Legitimacy Audit above) — v1.26.0 is confirmed current and correctly a real, resolvable tag, not a hallucinated version string.

## Architecture Patterns

### System Architecture Diagram

```
Contributor / CI invocation
        │
        ▼
Taskfile.yml "changie" wrapper target
        │  GOWORK=off go tool -modfile=go.tool-changie.mod changie <args>
        ▼
go.tool-changie.mod  ──(isolated MVS graph, 64 modules)──▶  changie v1.26.0 binary
        │
        ├─ reads .changie.yaml (kinds, auto bumps, PR custom field, 3 format strings)
        │
        ├─ `changie latest`   ──▶ scans .changes/v*.md filenames ──▶ stdout: v0.14.0
        │
        ├─ `changie next auto` ──▶ scans .changes/unreleased/*.yaml (NOT .gitkeep)
        │        │                       │
        │        │                 empty ─▶ exit 1, stderr "no unreleased changes found..."
        │        │                 non-empty ─▶ stdout: computed next version
        │
        ├─ `changie new -k <Kind> -b "<sentence>" -m PR=<n>` (CI=true, non-interactive)
        │        │
        │        ├─ kind not in .changie.yaml kinds[] ──▶ exit 1, stderr "invalid kind: <Kind>"
        │        ├─ PR missing (required custom, no `optional`) ──▶ exit 1, stderr
        │        │      "custom missing and prompt is disabled: custom key 'PR'"
        │        ├─ PR < minInt:1 ──▶ exit 1, stderr "input below minimum: <n> < 1"
        │        └─ else ──▶ writes .changes/unreleased/<Kind>-<timestamp>.yaml
        │
        └─ `changie merge --dry-run` (read-only, no file writes)
                 │
                 renders: header.tpl.md + newlines.afterChangelogHeader blank line(s)
                          + v0.14.0.md + v0.13.0.md + ... + v0.2.0.md (descending, verbatim)
                 │
                 ▼
        stdout ──▶ `cmp - CHANGELOG.md` ──▶ byte-identical (verified this session)

check:changie Taskfile target (D-07):
        mktemp -d scratch; cp .changie.yaml, .changes/{header.tpl.md,v*.md,unreleased/.gitkeep} → scratch
        cd scratch; run the six checks above in order; every failure emits a named ::error::
        (never mutates the real working tree — same discipline as proto:drift / docs:cli:drift)
```

### Recommended Project Structure
```
.changie.yaml                    # config: kinds, auto bumps, PR custom, 3 format strings, newlines.afterChangelogHeader: 1
.changes/
├── header.tpl.md                # exactly "# Changelog\n" — nothing else
├── unreleased/
│   └── .gitkeep                 # ignored by changie's *.yaml glob (confirmed live)
├── v0.14.0.md                   # verbatim byte slice of CHANGELOG.md's "## [0.14.0]..." block
├── v0.13.0.md                   # ... through v0.2.0.md, 14 files total
└── v0.2.0.md
go.tool-changie.mod              # new isolated tool modfile (5th one), `tool github.com/miniscruff/changie`
go.tool-changie.sum              # companion sum file, produced by `go mod tidy -modfile=go.tool-changie.mod`
```

### Pattern 1: Byte-sliced version-file seeding (D-01)
**What:** Each pre-existing `## [x.y.z]` block in `CHANGELOG.md` becomes its own `.changes/vX.Y.Z.md` file, sliced at the exact line range from that heading to (but not including) the next heading line — which means each seed file **already carries its own trailing blank-line separator** as part of its content (verified: `v0.14.0.md` ends `...)\n\n` — two newlines — while `v0.13.0.md` begins directly with `## [0.13.0]`, no leading blank).
**When to use:** Any historical-entry migration into a fragment-based changelog tool where the old format already embeds inter-entry spacing inside each entry (release-please's default format does this).
**Example (verified reproduction):**
```yaml
# .changie.yaml — the ONLY newlines override needed
newlines:
  afterChangelogHeader: 1
```
```bash
# Source: this session's live experiment, scratch copy only
$ changie merge --dry-run | cmp - CHANGELOG.md
# (zero output, exit 0 — byte-identical)
```

### Pattern 2: Isolated tool-modfile registration (D-04)
**What:** A 5th `go.tool-*.mod` file following the exact header/registration protocol of the existing four (`go.tool.mod`, `go.tool-lint.mod`, `go.tool-proto.mod`, `go.tool-golangci.mod`).
**When to use:** Pinning any new CI build tool per this repo's D-03/D-04 convention.
**Example:**
```go
// Source: internal/upgrade/taskfile_shape_test.go's own header-comment
// contract (mustToolModfileHeaderComment requires the word "isolat" to
// appear) — modeled on go.tool-lint.mod's header shape verbatim.
module github.com/seanb4t/codegraph-go/tools-changie

go 1.26.5

tool github.com/miniscruff/changie
```
Register in `internal/upgrade/taskfile_shape_test.go`:
```go
changieModfilePath = "../../go.tool-changie.mod" // add to the const block
var isolatedModfilePaths = []string{toolModfilePath, lintModfilePath, protoModfilePath, golangciModfilePath, changieModfilePath}
var forbiddenToolPackages = []string{
    "github.com/go-task/task",
    "github.com/goreleaser/goreleaser",
    "github.com/rhysd/actionlint",
    "github.com/golangci/golangci-lint",
    "github.com/bufbuild/buf",
    "github.com/miniscruff/changie", // NEW — changie has no runtime import anywhere in this module
}
```
`[VERIFIED: internal/upgrade/taskfile_shape_test.go:29-45,118-134,1057, read this session]` — the constants block, `forbiddenToolPackages` var, and `isolatedModfilePaths` var are exactly as quoted; `changie` must be added to **both** collections or `TestToolModfilesRemainIsolated`/`TestToolModfilesPopulationMatchesDisk` will not cover it (the same "absent from the set, inspected by nothing" gap that var's own doc comment says `go.tool-proto.mod` fell into for two phases before `03-10-PLAN.md` fixed it).

### Anti-Patterns to Avoid
- **Putting the `# Changelog` string, or any preamble, in a place other than `header.tpl.md`:** D-09 already rejects a "generated by changie, do not edit" comment in the header because it would add bytes that break the `cmp` byte-identity proof. Verified: `header.tpl.md` containing exactly `# Changelog\n` plus `newlines.afterChangelogHeader: 1` reproduces the real file's blank line; adding any extra text to the header file changes the diff.
- **Trusting a green exit code from `changie next auto` or `changie new` as proof of anything:** every failure path in this tool prints its reason to **both** stdout and stderr (`Error: <msg>` on stderr, bare `<msg>` on stdout) — CHG-04's proof must assert on the message text, not merely the exit code (rule `84d1gfpywd`; this project's own standing decision: "A guard must carry a positive assertion that it did its work").
- **Running any `changie new` invocation against the real repo working tree during CI verification:** D-07 requires a scratch copy; `changie new` mutates `.changes/unreleased/` on success, and a `check:changie` run that leaves stray fragments in the real tree would corrupt the next `changie latest`/`next auto` computation for actual contributors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| "Is this changelog byte-reproducible?" verification | A custom diff/normalize script that tries to account for whitespace differences | `cmp -s` (exact byte comparison) against changie's real `merge --dry-run` stdout | `docs:cli:drift`/`proto:drift` already establish this exact pattern in this repo; a "close enough" normalizer would hide the very regression class (a header-only or whitespace-only change) this repo's own conventions call "drift" |
| A changie "check" subcommand for the fragment-required gate | A hoped-for `changie check` or `changie validate` CLI verb | Custom diff-based gate (out of Phase 1 scope — Phase 4/GATE-01) | Confirmed in the design note's research ledger: "no changie subcommand or documented action pattern" exists for this; changie's `new`/`latest`/`next`/`merge` verbs are the complete public surface relevant here |

**Key insight:** Every "does X work" question in this phase has a cheap, real answer via `go install ... && ./changie <verb>` in a scratch directory — there is no unknowable behavior here that requires reading source, and no config-shape claim in this phase should ship un-verified when the verification costs under a minute.

## Common Pitfalls

### Pitfall 1: Assuming `newlines` needs multiple keys tuned
**What goes wrong:** CONTEXT D-02 lists four candidate `newlines` sub-keys (`beforeChangelogVersion`, `afterChangelogVersion`, `afterChangelogHeader`, `endOfVersion`) as if all four might need tuning to reproduce the file byte-for-byte.
**Why it happens:** The seed files already embed their own inter-version blank-line separator (each block, as byte-sliced from release-please's output, ends with a trailing blank line except the very last one) — so `beforeChangelogVersion`/`afterChangelogVersion`/`endOfVersion` all need to stay at their **defaults** (effectively 0 extra), and only the header→first-version boundary (which the seed files cannot supply, since the header is a separate template) needs an explicit setting.
**How to avoid:** Set only `newlines.afterChangelogHeader: 1`. Verified this session: adding this single key produced a byte-identical `cmp` result; leaving it unset produced a diff of exactly one missing blank line (`1a2 > ` in `diff` output).
**Warning signs:** A `diff` showing an extra or missing blank line ONLY at the header/first-version boundary, with every other version-to-version boundary already correct, is the fingerprint of this exact gap — don't reach for `beforeChangelogVersion` first.

### Pitfall 2: Treating `.gitkeep` as a possible source of a "phantom fragment"
**What goes wrong:** Worrying that changie's `unreleased/` scan might pick up `.gitkeep` as a malformed fragment and either error or silently miscount.
**Why it happens:** Untested assumption about glob behavior.
**How to avoid:** Confirmed live: `.changes/unreleased/` containing only `.gitkeep` makes `changie next auto` behave exactly as "zero fragments" (the "nothing to release" error); `.changes/unreleased/` containing `.gitkeep` **plus** one real `Fixes-*.yaml` fragment computes the bump correctly (`v0.14.1`) with `.gitkeep` invisible to the tool. changie's fragment discovery only matches `*.yaml`/`*.yml`.
**Warning signs:** N/A — verified clean; only a risk if a future changie version changes its glob (pin discipline in `go.tool-changie.mod` already guards against silent upgrades).

### Pitfall 3: Forgetting `github.com/miniscruff/changie` in `forbiddenToolPackages`
**What goes wrong:** The Go test `TestToolModfilesRemainIsolated` only fails a root-`go.mod` accidental promotion of a build tool if that tool's import path is listed in `forbiddenToolPackages`. Adding `go.tool-changie.mod` to `isolatedModfilePaths` alone does NOT protect against someone later adding `require github.com/miniscruff/changie` directly to the root `go.mod` — that guard is a separate check, keyed off the separate `forbiddenToolPackages` slice.
**Why it happens:** The two lists look similar but serve different assertions (`isolatedModfilePaths` proves N distinct modfiles exist with rationale headers; `forbiddenToolPackages` proves the root module never requires any of them directly), and it's easy to update only one when adding a 5th tool.
**How to avoid:** Update both `isolatedModfilePaths` and `forbiddenToolPackages` in the same edit to `taskfile_shape_test.go`.
**Warning signs:** `TestToolModfilesRemainIsolated` staying green after a hypothetical `go get github.com/miniscruff/changie` were added to the root `go.mod` — the exact "new subject passes because it is absent from the set" failure class this file's own comments describe happening to `go.tool-proto.mod` for two phases (`internal/upgrade/taskfile_shape_test.go:1046-1057`, read this session).

### Pitfall 4: Reading `CI=true changie new`'s custom-field YAML output as a schema violation
**What goes wrong:** The written fragment stores `custom: {PR: "1"}` — a quoted string — even though `.changie.yaml` declares `PR` as `type: int`. A shape test that strictly expects an unquoted YAML integer in a **written fragment** would misfire.
**Why it happens:** changie validates the `-m PR=<n>` CLI argument as an int at creation time (confirmed: `PR=0` was refused with `input below minimum: 0 < 1`) but serializes the custom-field map back to the fragment YAML as a string value regardless of declared type.
**How to avoid:** D-06's Go test only inspects `.changie.yaml` (the config), never a written fragment's YAML shape — this pitfall only matters if a future phase (e.g. Phase 2's capability, or Phase 4's gate) parses fragment YAML directly; note it there.
**Warning signs:** A test asserting `custom.PR` is a YAML integer type against a real fragment file will fail even though the config and the CLI-level validation are both correct.

## Code Examples

Verified patterns from this session's live experiments (scratch copy `/private/tmp/.../scratchpad/changie-experiment`, never the repo working tree):

### `changie latest` — the CHG-03 version-source-of-truth proof
```bash
# Source: this session, live run against the 14-file seed layout
$ changie latest
v0.14.0
```

### `changie next auto` — both outcomes, exact text
```bash
# Empty unreleased/ (only .gitkeep present)
$ changie next auto
Error: no unreleased changes found for automatic bumping   # stderr
no unreleased changes found for automatic bumping           # stdout
$ echo $?
1

# With one real Fixes fragment present
$ changie next auto
v0.14.1
$ echo $?
0
```

### `changie merge --dry-run` — the CHG-02/CHG-03 byte-reproduction proof
```bash
# Source: this session, live run
$ changie merge --dry-run | cmp - CHANGELOG.md
# (no output — byte identical, exit 0)

# RED demonstration (one-byte mutation of CHANGELOG.md's header text,
# scratch copy only, reverted immediately after):
$ sed -i '' 's/# Changelog/# CHANGELOG/' CHANGELOG.md   # scratch copy
$ changie merge --dry-run | cmp - CHANGELOG.md
stdin CHANGELOG.md differ: char 4, line 1
$ echo $?
1
# reverted from backup; re-ran cmp — byte-identical again (confirmed)
```

### `CI=true changie new` — all three outcomes, exact stderr text
```bash
# Valid: writes exactly one fragment
$ CI=true changie new -k Fixes -b "probe" -m PR=1
$ ls .changes/unreleased/
.gitkeep  Fixes-20260925-143406.yaml
$ cat .changes/unreleased/Fixes-20260925-143406.yaml
kind: Fixes
body: probe
time: 2026-09-25T14:34:06.114023-04:00
custom:
    PR: "1"

# Missing required PR field
$ CI=true changie new -k Fixes -b "probe-no-pr"
Error: custom missing and prompt is disabled: custom key 'PR'
$ echo $?
1

# Undeclared kind
$ CI=true changie new -k Undeclared -b "probe-bad-kind" -m PR=1
Error: invalid kind: Undeclared
$ echo $?
1

# Bonus (not in CHG-04's required proof set, but confirms minInt:1 works):
$ CI=true changie new -k Fixes -b "probe-pr-zero" -m PR=0
Error: input below minimum: 0 < 1
$ echo $?
1
```

### `go.tool-changie.mod` viability + measured module counts
```bash
# Source: this session, scratch copy with a placeholder root go.mod
$ GOWORK=off go mod tidy -modfile=go.tool-changie.mod
$ GOWORK=off go tool -modfile=go.tool-changie.mod changie --version
changie version vdev   # no ldflags version-stamping configured upstream; harmless
$ GOWORK=off go list -m -modfile=go.tool-changie.mod all | wc -l
64   # standalone: changie's own isolated graph (mirrors "16 modules alone" in go.tool-lint.mod's header)

# Co-location measurement (D-04's escape-hatch text: "permitted only if the
# executor measures it live... and records the measurement"). Scratch copy
# of the REAL go.tool.mod + go.mod + go.sum, tool directive added by hand:
$ GOWORK=off go mod tidy -modfile=go.tool.mod   # in the co-location scratch copy
$ GOWORK=off go list -m -modfile=go.tool.mod all | wc -l
1006   # baseline (unmodified go.tool.mod) measured at 1002 — delta: +4
$ GOWORK=off go list -m -modfile=go.tool.mod all | grep goreleaser/v2
github.com/goreleaser/goreleaser/v2 v2.17.1   # unchanged from baseline — no lost MVS bid
$ GOWORK=off go list -m -modfile=go.tool.mod all | grep sigstore/cosign
github.com/sigstore/cosign/v3 v3.1.1          # unchanged from baseline — no lost MVS bid
$ GOWORK=off go build -modfile=go.tool.mod -o /tmp/task-coloc-test github.com/go-task/task/v3/cmd/task
$ GOWORK=off go build -modfile=go.tool.mod -o /tmp/changie-coloc-test github.com/miniscruff/changie
$ GOWORK=off go build -modfile=go.tool.mod -o /tmp/goreleaser-coloc-test github.com/goreleaser/goreleaser/v2
# all three exit 0 — task, changie AND goreleaser all compile from the co-located modfile
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| release-please derives changelog + version from squash-merge PR titles | changie derives changelog + version from committed fragment files (`.changes/unreleased/*.yaml`) | This milestone (v0.15.0), Phase 1 baselines the config | The changelog becomes authored input rather than a proxy of PR titles; version becomes `changie latest`/`changie next auto`, not release-please's manifest file (which stays untouched until Phase 5) |

**Deprecated/outdated:** None within this phase's scope — `release-please.yml`, `release-please-config.json`, and `.release-please-manifest.json` are explicitly untouched until Phase 5 (REL-14); this phase adds a second, dormant configuration alongside the still-live release-please pipeline.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The package name `github.com/miniscruff/changie` (and the general shape of the migration) originated from a prior `/gsd-explore` research pass, not this session's independent discovery | Package Legitimacy Audit | Low — the package's existence, version and maintenance activity were independently re-verified this session via the module proxy and the GitHub API directly, not merely inherited on faith |
| A2 | `changie version vdev` (rather than a stamped `v1.26.0` string) is a harmless upstream ldflags-stamping gap, not a build misconfiguration on this repo's side | Code Examples | Low — `go version -m` on the binary independently confirms the correct pinned module version (`v1.26.0`) regardless of what `--version` prints; if a future guard asserts on `changie --version`'s output text specifically, it would need a different check (`go version -m` style) instead |

**If this table is empty:** N/A — two low-risk assumptions recorded above; neither affects a locked decision or a compliance/security surface.

## Open Questions

None outstanding for this phase. D-02's stated research question (which `newlines` settings reproduce the file byte-for-byte) is fully answered: `afterChangelogHeader: 1` alone, verified with `cmp` and a RED/GREEN mutation cycle. D-04's escape-hatch measurement (co-location cost) is answered as a bonus finding, though D-04 itself already locks the default (separate modfile) as this phase's deliverable regardless of that measurement.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building changie via `go tool -modfile=...`, running `go mod tidy` | Yes | go1.26.6 (root go.mod), go1.27.1 (this session's ambient toolchain, auto-selected by `go install`'s own `go 1.27.1` toolchain directive on the changie module) `[VERIFIED: go version -m output, this session]` | None needed |
| Network access to `proxy.golang.org` | `go install`/`go mod tidy` resolving `github.com/miniscruff/changie@v1.26.0` | Yes (this session) | — | CI already relies on the same proxy for every other tool modfile; no new dependency class introduced |
| `cmp`, `diff` (BSD variants, macOS) | `check:changie`'s byte-comparison step; this session's manual verification | Yes | macOS BSD `cmp`/`diff` (note: BSD `cat` has no `-A` flag, unlike GNU — use `od -c` for byte inspection instead if writing local verification scripts) | `docs:cli:drift`/`proto:drift` already use plain `cmp -s`, portable across CI's Linux runners and local macOS dev |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — every dependency this phase needs is already present in this repo's existing build-tool conventions.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (`go test ./internal/upgrade/...`), following `taskfile_shape_test.go`'s existing pure-parser + `mustX`/`parseX` idiom |
| Config file | none — plain `go test`, no framework config |
| Quick run command | `go test ./internal/upgrade/... -run TestChangieConfigShape` (name illustrative; plan names the actual test function) |
| Full suite command | `task test:unit` (existing wrapper leg; `internal/upgrade` is part of the unit-test tree already) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| CHG-01 | `.changie.yaml` declares exactly the 5 kinds with correct `auto` bumps, the flip-to-major note text, the `PR` custom field (`type: int`, `minInt: 1`, no `optional`), and the 3 format strings match the design note verbatim | unit (Go, RED-first per D-06) | `go test ./internal/upgrade/... -run TestChangieConfigShape -v` | ❌ Wave 0 — new test function in `internal/upgrade` (beside `taskfile_shape_test.go`, per D-06) |
| CHG-01 (changie pin) | `go.tool-changie.mod` exists, is registered in `isolatedModfilePaths` + `forbiddenToolPackages`, has an "isolat"-mentioning header | unit (Go) | `go test ./internal/upgrade/... -run TestToolModfilesRemainIsolated -v` and `-run TestToolModfilesPopulationMatchesDisk` | ✅ existing test, extended with the new registration |
| CHG-02 | `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep`, all 14 `.changes/v*.md` exist; set-equality against `CHANGELOG.md`'s `## [x.y.z]` headings, both directions | unit (Go, RED-first) | `go test ./internal/upgrade/... -run TestChangieConfigShape -v` (same function as CHG-01's static half, per D-06's single-test design) | ❌ Wave 0 — same new test |
| CHG-03 | `changie latest` = `v0.14.0`; `changie next auto` (empty) fails with the exact stderr text; `changie merge --dry-run` byte-reproduces `CHANGELOG.md` | live-tool (Taskfile target, shells to the real changie binary) | `task check:changie` | ❌ Wave 0 — new Taskfile target, `docs:cli:drift`-shaped |
| CHG-04 | `CI=true changie new -k <Kind> -b "..." -m PR=<n>` writes one fragment; missing `PR` refused; undeclared kind refused — both refusal texts asserted verbatim | live-tool (same Taskfile target) | `task check:changie` | ❌ Wave 0 — same new target |
| CHG-03 RED demonstration | one-byte `CHANGELOG.md` mutation (scratch copy) makes the byte-reproduction leg of `check:changie` fail, byte-cleanly reverted | manual/recorded transcript (not a committed test — a mutation-log entry per rule `84d1gfpywd`) | `cmp` failure text pasted into the plan's SUMMARY, mirroring this session's own `stdin CHANGELOG.md differ: char 4, line 1` transcript | N/A — evidence artifact, not a test file |

### Sampling Rate
- **Per task commit:** `go test ./internal/upgrade/... -run TestChangieConfigShape -v` (fast, seconds) plus a manual `task check:changie` run once the Taskfile target exists (shells to a real changie binary build, ~5-10s for `go build` cold, near-instant warm)
- **Per wave merge:** `task test:unit` (covers `internal/upgrade`'s full suite, including the pre-existing `taskfile_shape_test.go` guards this phase extends) + `task check:changie`
- **Phase gate:** Both green, plus the RED-then-revert mutation-log transcript for the byte-reproduction check pasted into the phase's SUMMARY, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] New Go test function (name TBD by the plan) in `internal/upgrade/` — the D-06 config-shape assertions (CHG-01 static half + CHG-02). Must land RED-first per rule `x1cjy9vyhq`: commit the test before `.changie.yaml`/`.changes/` exist, paste the `--- FAIL` transcript in the SUMMARY.
- [ ] `check:changie` Taskfile target — the D-07 live-tool proof (CHG-03 + CHG-04), `docs:cli:drift`-shaped (scratch dir, count-before-compare, named `::error::` failures).
- [ ] `ci.yml` `test` job step `run: task check:changie`, placed immediately after the existing `docs:cli:drift` step (ci.yml:203) — no new `runBodyExceptions` entry needed since `ci.yml`/`test` is already in `inScopeJobs`, provided the step's `run:` body is exactly `task check:changie`.
- [ ] `go.tool-changie.mod` + `go.tool-changie.sum` — new files, `go mod tidy -modfile=go.tool-changie.mod` generates the `.sum` companion automatically (confirmed this session).

*(Framework install: none — `go test` is already the framework; no new test tooling needed.)*

## Security Domain

`security_enforcement` is enabled (`.planning/config.json` → `workflow.security_enforcement: true`, `security_asvs_level: 1`) `[VERIFIED: .planning/config.json, read this session]`. This phase's surface is build/release tooling configuration, not a user-facing input or auth surface — most ASVS categories do not apply. The two real considerations are supply-chain (a new external Go tool) and data-handling of contributor-authored fragment content.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No auth surface in this phase |
| V3 Session Management | No | No session surface in this phase |
| V4 Access Control | No | No access-control surface; fragment writing is a local/CI CLI operation, not a network-exposed capability |
| V5 Input Validation | Partial | changie's own `kinds[]` allowlist + required `PR` `minInt: 1` custom field IS an input-validation control (verified live: undeclared kinds and missing/sub-minimum `PR` values are both refused) — but this validation lives inside the pinned, checksum-verified `changie` binary, not in this repo's own code |
| V6 Cryptography | No | No cryptographic operation introduced; module fetch integrity is covered by Go's existing `go.sum` checksum verification, same as every other tool modfile |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Supply-chain compromise of a new external build-tool dependency | Tampering | Already the existing repo pattern: pin the exact version, resolve only via the checksum-verified module proxy (`go.sum`), never a marketplace action or curl-pipe install (D-04, `.github/actions/install-task/action.yml`'s documented rationale) — the Package Legitimacy Audit above confirms `changie` v1.26.0 is a mature (5.8yr), actively-maintained, non-archived, MIT-licensed project, not a fresh or hijacked package |
| Malicious content in a contributor-authored fragment body/PR field rendered into `CHANGELOG.md` | Tampering / Information Disclosure (low severity — output is a markdown file, not executed code) | Fragment files are `.yaml` written by `changie new` under the same PR-review gate as any other repo change (this phase doesn't add a write path outside normal git commits); `changeFormat`'s `{{.Body}}`/`{{.Custom.PR}}` template variables are rendered as plain text data, not executed as templates themselves — no template-injection vector was found in this session's testing (Body content is inserted as a literal string, never re-parsed as Go template syntax) |

