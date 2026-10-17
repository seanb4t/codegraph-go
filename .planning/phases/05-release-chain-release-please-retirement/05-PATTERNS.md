# Phase 5: Release Chain & release-please Retirement - Pattern Map

**Mapped:** 2026-10-14
**Files analyzed:** 22 (6 new, 13 modified, 3 deleted)
**Analogs found:** 22 / 22 (all analog paths verified git-tracked; none is a gitignored mirror)

Toolchain for every cited Go command: `GOTOOLCHAIN=go1.26.9` (go1.27 breaks cockroachdb/swiss).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.github/workflows/release-pr.yml` (new) | workflow (config) | event-driven (push to main) | `.github/workflows/release-please.yml` | exact (pretag-gate, App-token mint) |
| `.github/workflows/release-tag.yml` (new) | workflow (config) | event-driven (push, path-filtered) | `.github/workflows/release-please.yml` + `release.yml` (App-token-to-checkout) | role-match |
| `Taskfile.yml` `release:batch`, `release:tag` (new) | task target | batch / file-I/O | `Taskfile.yml` `check:changie` (:556-) | role-match |
| `Taskfile.yml` `release:goreleaser` (edit, ~:2780-2915) | task target | request-response | itself (tag derivation already at :2915) | exact |
| `Taskfile.yml` `release:dry-run` (edit, :2467-2599) | task target | batch | itself | exact |
| `Taskfile.yml` `check:changie` new legs + `[n/13]` renumber | gate target | batch | itself (:556-770) | exact |
| `Taskfile.yml` `check:fragment-policy` floor bump (:2258) | gate target | batch | itself (`floor=36`) | exact |
| `Taskfile.yml` `check:release-please-census` (new, name free) | gate target | batch | `check:fragment-policy` + `check:changie` (counts, positive control) | role-match |
| `.changie.yaml` (edit `newlines`) | config | transform | itself (`newlines:` at ~:53) | exact |
| `internal/upgrade/changie_release_shape_test.go` (new) | test | request-response | `internal/upgrade/changie_shape_test.go` + `taskfile_shape_test.go` workflow helpers | role-match |
| `internal/upgrade/changie_shape_test.go` (edit `TestChangieConfigShape`, comment :113) | test | request-response | itself (:283) | exact |
| `internal/upgrade/tag_authority_test.go` (rename/re-point + GRD-16) | test | request-response | itself (:23-114) | exact |
| `internal/upgrade/taskfile_shape_test.go` (registry :149, exceptions :1441, files :1462, SAN :2420) | test | request-response | itself | exact |
| `internal/upgrade/release_workflow_shape_test.go` (SAN fixture :889) | test | request-response | itself | exact |
| `internal/upgrade/goreleaser_shape_test.go` (:984-1009 prose; new `changelog.disable` guard) | test | request-response | itself (`TestReleaseBlockDoesNotRewriteReleaseBody`) | exact |
| `scripts/pr_template_policy.py` (:54-60, :83) | utility | transform | itself | exact |
| `scripts/fragment_policy.py` (:49 marker) | utility | transform | itself / `pr_template_policy.py` | exact |
| `scripts/test_fragment_policy.py` | test | transform | itself (`test_marker_grammar_mirrors_pr_template_exempt` :173) | exact |
| `.github/workflows/require-issue-link.yml` (:86-93) | workflow | request-response | itself | exact |
| `CONTRIBUTING.md`, `docs/RELEASE-PROCEDURES.md`, `docs/RELEASE.md`, `.github/workflows/pr-title.yml` header | docs | n/a | existing text in each file | exact |
| `.planning/phases/05-.../05-CENSUS.md` (new) | artifact | tabular verdicts | `.planning/phases/04-fragment-required-gate/04-REVIEW-DISPOSITION.md`, `04-MUTATION-LOG.md` | role-match |
| delete `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json` | deletion | n/a | n/a (last commit, after guards re-pointed) | n/a |

## Pattern Assignments

### `.github/workflows/release-pr.yml` and `release-tag.yml` (workflow, event-driven)

**Analog:** `.github/workflows/release-please.yml` (lines 16-92). Move the `pretag-gate` job verbatim; the step `Run check:cross sweep` / `run: task check:cross` is the verbatim part. Reword the stale job `name:` ("6-target"; `check:cross` runs four pairs) and the stale comment.

**pretag-gate job + pins** (release-please.yml:34-55):
```yaml
  pretag-gate:
    name: pre-tag 6-target go list sanity sweep
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Run check:cross sweep
        run: task check:cross
```

**App-token mint step** (release-please.yml:73-84; keep `app-id`, not `client-id`):
```yaml
      - name: Mint GitHub App installation token
        id: app-token
        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3.2.0
        with:
          app-id: ${{ secrets.APP_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

**Differences to apply:** top-level `permissions: contents: read` stays (release-please.yml:22-23); drop the job-level `contents/pull-requests/issues: write` block (the App token carries writes). Mint the token inside the consuming job, never as a job output (masked outputs are dropped; see `release.yml` comment ~:169-175). Neither workflow may have `name: release` (`post-release-verify.yml` keys on `workflow_run: ["release"]`). `release-tag.yml` `on:` is `push: {branches: [main], paths: [CHANGELOG.md]}`; its checkout takes `with: token: ${{ steps.app-token.outputs.token }}`; both new jobs `needs: pretag-gate`. Full proposed YAML shapes are in 05-RESEARCH.md "Code Examples" (lines 366-476); `create-pull-request` pin is `peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1` with `token:` = the mint output, `draft: false`, `body-path: .changes/<version>.md`.

**Every `run:` body must be exactly `task <target>`** (`TestWorkflowRunBodiesInvokeTask`); the `release-pr` job's only `run:` is `task release:batch`, the `tag` job's is `task release:tag`.

---

### `Taskfile.yml` `release:batch` / `release:tag` (task target, batch/file-I/O)

**Analog:** `check:changie` (Taskfile.yml:556-770) for structure; `release:goreleaser` (:2780-2915) for `desc`/preconditions/`GITHUB_OUTPUT` idiom; `GO_TOOL_CHANGIE` var at Taskfile.yml:13 (`GOWORK=off go tool -modfile=go.tool-changie.mod`), invoked as `{{.GO_TOOL_CHANGIE}} changie ...` (Taskfile.yml:549).

**Idioms to copy** (Task interprets `cmds:` with its built-in shell; use only constructs `check:changie` uses):
```yaml
    silent: true
    cmds:
      - |
        set -euo pipefail
        nseeds=$(find .changes -maxdepth 1 -type f -name 'v*.md' | wc -l | tr -d ' ')
        if [ "${nseeds}" -lt 14 ]; then
          echo "::error::check:changie: found only ${nseeds} ... — a broken enumeration must fail loud, never read as a clean pass"
          exit 1
        fi
```
Emit outputs like `release:goreleaser` does:
```sh
if [ -n "${GITHUB_OUTPUT:-}" ]; then ... >> "${GITHUB_OUTPUT}"; fi
```
Constraints: no `status:` / `platforms:` keys (`TestTaskfileGatesFailLoud`); `desc:` is a `>-` folded block; any `task <x>` mention in CONTRIBUTING.md must name a real target. Bodies proposed in 05-RESEARCH.md lines 481-531 (empty-set guard with `find .changes/unreleased -maxdepth 1 -type f -name '*.yaml' | wc -l`, version regex check, skip-ci scan, `git ls-remote --exit-code` rc-2-means-absent `case`). Prove through `task`, not `bash -c`.

---

### `Taskfile.yml` `release:goreleaser` and `release:dry-run` (edit)

**Analog:** the targets themselves. In `release:goreleaser` the final call is `"${GORELEASER_BIN}" release --clean` (~:2897) and `TAG="$(git describe --exact-match --tags HEAD)"` appears later inside the `GITHUB_OUTPUT` block (~:2915). Hoist the TAG derivation above the call, assert `[ -s ".changes/${TAG}.md" ]`, pass `--release-notes ".changes/${TAG}.md"`. `release.yml`'s `run: task release:goreleaser` is unchanged. For `release:dry-run`, replace `--snapshot` with `GORELEASER_CURRENT_TAG="$(... changie latest)"` + `--skip=publish,sign,validate` + the same flag (snapshot skips the changelog pipe; research Pitfall 2) and print a `RELEASE-NOTES-EVIDENCE` line. Keep the existing `jq` `BINARY_COUNT != "4"` and per-pair `file -b` `case` assertions (:2503-2597). `release:dry-run-signed` (:2601) changes comment only (DOCS-14).

---

### `Taskfile.yml` `check:changie` new legs; `check:fragment-policy` floor; census target

**Analog (legs):** `check:changie` (Taskfile.yml:556-770): scratch dir via `mktemp -d` + `trap`, `go build -modfile=go.tool-changie.mod -o "${scratch}/changie"`, copy `.changie.yaml`, `header.tpl.md`, `v*.md`, `.gitkeep` into `${work}`, a `passed` counter, numbered legs with `::error::check:changie: [n/13] ...` and `check:changie: [n/13] ... — ok`, before/after `cksum` snapshot proving the source tree untouched. Adding legs means updating every `[n/13]` string, "thirteen numbered checks", "must reach 13", and the Go test pin that reads the counts (consistently). Leg 3 shows the byte-compare idiom (`merge --dry-run` vs CHANGELOG.md).

**Analog (floor):** `check:fragment-policy` (Taskfile.yml:2258-2298): `floor=36`, `defined=$({ grep -oE '^    def test_[A-Za-z0-9_]+' scripts/test_fragment_policy.py || true; } | wc -l ...)`, ran-equals-defined check, `grep -x 'OK'`. Bump `floor` and the `desc` ("at least 36") in the same commit as the added Python tests.

**Analog (census):** same two targets for "positive count" discipline. Instrument per research lines 580-591: `git ls-files -co --exclude-standard | grep -vE '^(\.planning/|CHANGELOG\.md$|\.changes/v[0-9][^/]*\.md$)'`, filename check plus `git grep -iEl 'release[-_ ]?please'` on content (macOS: no `xargs -d`); a self-test phase first plants a reference and a filename in a scratch tree and must see both.

---

### `.changie.yaml` (config)

**Analog:** itself, `newlines:` block at ~:53-54 (`afterChangelogHeader: 1`). Add `beforeKind: 1`, `afterKind: 1`, `endOfVersion: 2` (Pitfall 1). Rewrite the header comment claiming it is the only newlines override, and the line-20 "release-please-authored" comment (census).

---

### `internal/upgrade/changie_release_shape_test.go` (new test)

**Analog:** `internal/upgrade/changie_shape_test.go` (package `upgrade`; imports `yaml "go.yaml.in/yaml/v3"`, `golang.org/x/mod/semver`; const paths like `changieConfigPath = "../../.changie.yaml"`) and the workflow helpers in `taskfile_shape_test.go`.

**Reusable helpers (do not rewrite):**
- `parseWorkflowJobSteps(src, jobID)` / `parseWorkflowJobsAllSteps(src)` (taskfile_shape_test.go:1285, :2652) returning `[]workflowRunStep{Name, Run, Uses}`; they error on zero jobs / missing job / zero steps (the vacuous-pass guard).
- `workflowFileYAML` / `workflowJobYAML` (:1271-1278) for decoding (extend with `needs`, `with`, `on` in the new file's own struct if needed).
- `taskTargetsInRunBody(body)` + `cosignGuardTaskCallLineRe = ^task\s+([A-Za-z0-9:_.-]+)$` (taskfile_shape_test.go ~:2662-2690), which strips comment noise via `stripRunBodyNoise` and returns invoked targets: this is the "follow `task` into Taskfile" primitive GRD-16 needs.
- `mustParseTaskBlocks(t, src)` (used at changie_shape_test.go:583) returning map target name to block text; `taskfilePath`, `ciWorkflowPath`, `postReleaseWorkflowPath`, `workflowsDir` constants.
- `parseWorkflowOnKeys`, `parseWorkflowTopLevelName` (release_workflow_shape_test.go:37, :53) for trigger/name checks.
- Pin regex idiom (raw source, not decoded, because YAML strips the comment): `cosignInstallerPinRe = (?m)uses:\s*(sigstore/cosign-installer@[0-9a-f]{40})\s+#\s*(v\d+\.\d+\.\d+)\s*$` (taskfile_shape_test.go).

**Positive-population idiom** (changie_shape_test.go:570-573, Taskfile read at :577-590):
```go
blocks := mustParseTaskBlocks(t, src)
block, ok := blocks["check:changie"]
if !ok { t.Fatalf("%s: no top-level %q task", taskfilePath, "check:changie") }
msgs, err := parsePreconditionMessages(block)
if len(msgs) == 0 { t.Fatalf("... declares preconditions: but no non-empty msg: values") }
```
Every new guard: fail loudly if the subject is not found, count a non-zero population, and be shown RED against a confirmed-applied mutation (rule `84d1gfpywd`). Header comment convention (changie_shape_test.go:1-8): state RED-first per `x1cjy9vyhq` and that `gsd-tools check tdd-red-evidence` is not used.

**GRD-16 shape:** scan all `.github/workflows/*.yml` `run:` bodies for tag-creating commands (`git tag`, `git push` with `--tags`/`refs/tags`, `gh api .../git/refs`, `gh release create`); resolve each `task <target>` to the Taskfile block and scan it; assert exactly 1 total, located in `release-tag.yml` job `tag`, whose checkout `with.token` is `steps.app-token.outputs.token` and which has no `github.token`/`secrets.GITHUB_TOKEN`; plus positive counts (workflows seen > 0, targets resolved > 0, tag sites == 1).

---

### `internal/upgrade/tag_authority_test.go` (re-point + rename)

**Analog:** itself. Keep `TestGsdTagCreationIsDisabled` logic (lines 46-72) byte-for-byte; only reword D-06R prose at :9, :40, :61, :70, :82.

**Replace** `TestReleasePleaseStaysPreMajor` (:90-114) and const `releasePleaseConfigPath` (:20). New test (new name, no "ReleasePlease"): read `../../.changie.yaml` (`changieConfigPath` exists in changie_shape_test.go; reuse it), decode with `yaml.Node` helpers (`yamlMappingValue`, `yamlMappingKeys` from changie_shape_test.go:~275-290), find the `kinds` entry with `label: Breaking`, assert `auto: minor`, `t.Fatalf` if no `Breaking` kind is found (positive assertion). Fail-message style to copy (:103, :108, :112): name the file, the observed `%T (%v)`, the decision (D-06R), and say changing it is a maintainer decision recorded in PROJECT.md first. RED mutation: set `auto: major`.

---

### `internal/upgrade/taskfile_shape_test.go` (registry edits)

**Analog:** itself.
- :149 `{Workflow: "release-please.yml", JobID: "pretag-gate"}` becomes entries for `release-pr.yml/pretag-gate`, `release-pr.yml/release-pr`, `release-tag.yml/pretag-gate`, `release-tag.yml/tag` (the latter two non-gate jobs hold `run:` steps so they belong in `inScopeJobs`, not the uses-only list).
- :1441-1445 delete the `usesOnlyJobExceptions` entry `{Workflow: "release-please.yml", JobID: "release-please", Reason: ...}` (`validateUsesOnlyJobExceptions` fails a stale entry).
- :1462 `inScopeWorkflowFiles = []string{"ci.yml", "release-please.yml", "corpora.yml", "components-drift.yml", "fragment-required.yml"}`: swap `release-please.yml` for `release-pr.yml`, `release-tag.yml` (`TestWorkflowFilePopulationMatchesDisk` requires every on-disk file in exactly one list).
- :2420 SAN negative fixture: `.../workflows/release-please.yml@refs/tags/v1.2.3` becomes `release-tag.yml`. Same swap at `release_workflow_shape_test.go:889` (`reject_renamed_workflow` subtest).

---

### `internal/upgrade/goreleaser_shape_test.go` (prose + new guard)

**Analog:** `TestReleaseBlockDoesNotRewriteReleaseBody` (:984-1009) and helper `mustGoreleaserTopLevelBlock(t, src, key)` (:248). Reword the doc comment and the line-1005 message ("release-please owns Release authorship (D-06R)") to: GoReleaser authors the body from `.changes/${TAG}.md`, so a header/footer/name_template/draft/disable key would make it differ. Forbidden-key list unchanged:
```go
for _, forbidden := range []string{"name_template", "header", "footer", "draft", "disable"} {
	if _, ok := block[forbidden]; ok { t.Errorf(...) }
}
```
**New guard `changelog.disable`** (none exists): copy the shape `data, err := os.ReadFile(goreleaserConfigPath)` then decode; assert no `changelog` block or `disable` not bool true and not string `"true"`; fatal if the config or the `release` block cannot be found. RED mutation: add `changelog: {disable: true}`.

---

### `scripts/pr_template_policy.py`, `fragment_policy.py`, `test_fragment_policy.py`, `require-issue-link.yml` (hygiene allowlist swap + WR-01)

**Analog:** the files themselves; keep the two lists a mirrored pair.

`pr_template_policy.py` :54-60 (remove these two names and their comment; add `".changes/"` to `EXEMPT_PREFIXES`, which is used with `str.startswith`):
```python
    # release-please machinery. ...
    ".release-please-manifest.json",
    "release-please-config.json",
```
WR-01 :83 (replace; reason is `.group(1).strip()`, exempt only if non-empty):
```python
EXEMPT_MARKER = re.compile(r"<!--\s*pr-template-exempt:\s*(?P<reason>[^->][^>]*?)\s*-->", re.I)
# -> re.compile(r"<!--\s*pr-template-exempt:([^>]*)-->", re.I)
```
The same fix goes into `fragment_policy.py:49` `CHANGELOG_EXEMPT_MARKER` with `changelog-exempt`. Decide and test whether a reason beginning with `-` stays rejected (old pattern's semantics differ; research Pitfall 7).

`require-issue-link.yml` :86-93 (delete the second arm and comment, add a `.changes/*` arm modeled on the first):
```sh
              .planning/*|docs/*|.github/*) ;;
              ...
              go.tool.mod|go.tool.sum|go.tool-lint.mod|go.tool-lint.sum) ;;
              .release-please-manifest.json|release-please-config.json) ;;
              *) all_exempt=false; break ;;
```

**Test idiom to extend** (`scripts/test_fragment_policy.py:173-181`; keep marker patterns identical modulo marker name):
```python
    def test_marker_grammar_mirrors_pr_template_exempt(self):
        mine = fragment_policy.CHANGELOG_EXEMPT_MARKER
        theirs = pr_template_policy.EXEMPT_MARKER
        self.assertEqual(
            mine.pattern, theirs.pattern.replace("pr-template-exempt", "changelog-exempt")
        )
        self.assertNotEqual(mine.pattern, theirs.pattern)
        self.assertEqual(mine.flags, theirs.flags)
```
Tests are stdlib `unittest`, `def test_*` at 4-space indent (the `check:fragment-policy` grep counts exactly `^    def test_`). Add: `.changes/`-only PR passes policy; parity test for both directions; 60k-character adversarial body under a time bound; raise `floor` in Taskfile.

---

### `docs/*`, `CONTRIBUTING.md`, `pr-title.yml` header (docs)

**Analog:** current text. `pr-title.yml:1-18` header (release-please parses PR title) becomes "hygiene only"; `CONTRIBUTING.md:90` and :135/:171 (`task <x>` mentions must match ``` `task\s+([A-Za-z0-9:_-]+)` ``` real targets, `TestContributingReferencesRealTaskTargets`); `Taskfile.yml:5253` `check:cross` desc; `.goreleaser.yaml:404-410`, `ci.yml:551`, `post-release-verify.yml:21-23`, `release.yml:188` comment hunks (census). Write no emojis. Never hand-edit `.changes/v*.md` or `CHANGELOG.md`; REQUIREMENTS/ROADMAP REL-11 amendment edits values inside the existing clause only.

---

### `05-CENSUS.md` (artifact)

**Analog:** `04-REVIEW-DISPOSITION.md` (YAML front matter with `findings:` list of `id / severity / disposition / title`; one verdict per item) for the 45-row D-10 verdict table, and `04-MUTATION-LOG.md` (header with phase/date/scope; "pre-mutation cleanliness gate" convention: `git diff --quiet -- <file>` before every mutation and revert; pasted transcripts per section) for the census and RED/revert transcripts. Row count of verdicts must equal 45.

## Shared Patterns

### Action pinning
**Source:** `.github/workflows/release-please.yml` lines 44, 47, 81; **Apply to:** both new workflows. Full 40-hex SHA plus trailing `# vX.Y.Z` comment on every `uses:`; local composite `./.github/actions/install-task` is the one allowed unpinned form.

### CI run bodies are `task <target>`
**Source:** `TestWorkflowRunBodiesInvokeTask` / `checkStepInvokesTask` (taskfile_shape_test.go:1338); **Apply to:** all new `run:` steps; `uses:`-only jobs need a `usesOnlyJobExceptions` entry with a reason.

### Fail loud / positive population
**Source:** `check:changie` floor-14 check (Taskfile.yml:586-592) and `parseWorkflowJobSteps` error returns; **Apply to:** every new target and Go test (rule `84d1gfpywd`: shown RED against a confirmed-applied, byte-cleanly-reverted mutation, with a non-zero count asserted).

### Tests exercise this repo only
**Apply to:** all Go/Python guards (rule `eyva2svrcz`): never assert how git, GitHub, Actions, changie, GoReleaser or `create-pull-request` behave; assert on this repo's YAML, Taskfile bodies and config.

### App token, never `GITHUB_TOKEN`, for tag and PR
**Source:** release-please.yml:9-14 rationale; **Apply to:** `release-tag.yml` tag job checkout and `create-pull-request`. Do not set `persist-credentials: false` on the tag job.

### RED-first commit convention
**Source:** changie_shape_test.go:1-8; **Apply to:** all Go guards: `test(05-PP):` commit before implementation, pasted `--- FAIL:` transcript in the SUMMARY; never `[ci skip]` in commit messages (rule `f18zrdsgx5`).

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| tag-job `release:tag` test that runs the target body against a local bare `origin` | test | file-I/O (git) | No existing test executes a Taskfile body in a temp git repo; planner uses RESEARCH.md validation map (REL-11 row) and shells `task release:tag` with a bare remote |
| GRD-16 tag-site scan that follows `task` calls into the Taskfile | test | transform | Building blocks exist (`taskTargetsInRunBody`, `mustParseTaskBlocks`) but no current guard composes them; use RESEARCH.md Pitfall 6 |
| Census self-test phase | gate target | batch | Closest is `scripts/check-workflow-output-delimiter.sh --self-test` (named in research, not read here) |

## Metadata

**Analog search scope:** `.github/workflows/`, `Taskfile.yml`, `internal/upgrade/*_test.go`, `scripts/`, `.planning/phases/04-fragment-required-gate/`
**Files scanned:** about 15 read or excerpted; all named analogs confirmed tracked (`git ls-files` over `.github/workflows`, `Taskfile.yml`, `changie_shape_test.go`, `04-PATTERNS.md`)
**Pattern extraction date:** 2026-10-14
