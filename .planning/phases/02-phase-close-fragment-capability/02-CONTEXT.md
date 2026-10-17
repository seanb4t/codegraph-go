# Phase 2: Phase-Close Fragment Capability - Context

**Gathered:** 2026-09-25
**Status:** Ready for planning
**Mode:** auto (`/gsd-discuss-phase 2 --auto`, `.planning/config.json` `mode: yolo`). Every choice below is the recommended option. The audit trail is in `02-DISCUSSION-LOG.md`.

<domain>
## Phase Boundary

This phase delivers two things in two repositories.

1. **A new private repository, `seanb4t/gsd-capability-changie`.** It holds a gsd-core 1.14.0 `role: "feature"` capability. The capability owns the skill `changie-fragments` and registers one `verify:post` step. That step is gated on the federated config key `workflow.changie_fragments`. When the step runs, the skill turns a closed phase's `*-SUMMARY.md` files into changie fragments and commits them. The repository also has a README, a LICENSE, a tagged release, and its own scratch-project proofs for CAP-01, CAP-02 and CAP-03.
2. **codegraph-go work.** Install that tag at project scope. Set the config keys through the tool's own verb. Keep `.gsd/` and any other install output gitignored. Document the one-time install for each clone in `CONTRIBUTING.md` (CAP-04). Before Phase 3 closes, make sure the skill can get a real PR number on the milestone branch (see D-03).

Out of this phase:
- The capability's first firing inside this repository. That is `CAP-05`, proven when Phase 3 closes.
- The fragment-required gate (Phase 4).
- The release workflow (Phase 5).
- Edits to gsd-core workflow files (out of scope by requirement).
- A `contribution` or `ref.command` step (Out of Scope table).
- Porting to other repositories and upstreaming to gsd-core (`PORT-01`/`PORT-02`, v2).

</domain>

<decisions>
## Implementation Decisions

### Capability identity and manifest
- **D-01:** The capability id is `changie`. The skill stem is `changie-fragments`. gsd-core dispatches a `ref.skill` step as `Skill(skill="gsd-<ref.skill>")` (`references/loop-hook-dispatch.md`, `workflows/verify-work.md:562ff`, `workflows/autonomous.md:542ff`), so the host invokes the skill as **`gsd-changie-fragments`**. This is the name a user types to re-run it by hand. The id avoids the reserved `gsd-` / `gsd-core-` / `anthropic-` prefixes, and the repository name `gsd-capability-changie` is not the id. — **Reversibility:** costly. The id is recorded in the per-scope install ledger of every clone that installs it, and the skill name appears in docs and manual re-runs.
  - The researcher must confirm by a real install on a scratch project, not by reading the loader, that a project-scope install makes the skill resolvable to Claude Code as `gsd-changie-fragments`. The researcher also records where the files land (`.gsd/capabilities/changie/…` and any runtime skills directory) so that CAP-04 can gitignore every path the install writes.
- **D-02:** The manifest declares two federated config keys in its `config` slice:
  - `workflow.changie_fragments` (`boolean`, default `true`). This is the step's `when` gate, exactly as CAP-02 states it.
  - `workflow.changie_command` (`string`, default `"changie"`). This is the command prefix the skill runs changie through.

  There is exactly one `verify:post` step: `ref.skill: "changie-fragments"`, `onError: "skip"`, gated on the first key. `engines.gsd` is a range anchored on the tested 1.14.0. Use caret-style `^1.14.0` unless the validator in `capability-validator.cjs` requires another syntax; the researcher reads the validator's accepted form and the loader's behaviour on an engine mismatch. — **Reversibility:** reversible. Config keys can be added later, but removing one breaks any clone that set it.
  - **Why `workflow.changie_command` exists:** in this repository changie is **not on `PATH`**. It runs as `task changie -- …`, through `GO_TOOL_CHANGIE` and the isolated `go.tool-changie.mod` (Phase 1 D-04/D-05, `Taskfile.yml:13`, `:538`, CONTRIBUTING.md:124-128). CAP-03's literal "skip when `changie` is not on `PATH`" would therefore **always** skip here, and CAP-05 could never pass. The skill resolves the configured command instead. The "not on `PATH`" skip case becomes "the configured command's executable cannot be found or `<command> --version` fails". The default `changie` keeps the literal CAP-03 behaviour for repositories that install changie as a binary. codegraph-go sets the key to `task changie --`. The verifier must read CAP-03's first skip case as "configured changie command unavailable". Record this interpretation in the phase VERIFICATION. Do not paper over it.

### Where the PR number comes from (ROADMAP Phase 2 Notes, hazard 4, Phase 1 D-11)
- **D-03:** The skill finds `<n>` by asking GitHub for the open PR (draft or ready) whose head is the current branch: `gh pr view --json number,state` or `gh pr list --head <branch> --state open --json number`. Rules:
  - The skill **never** writes a placeholder number and **never** makes `PR` optional. `.changie.yaml` keeps `PR` required (CHG-01/CHG-04).
  - If there is no open PR, `gh` is missing, or `gh` is not authenticated, the skill prints a named reason and returns. This is a fourth skip case, same shape as CAP-03's three: no prompt, no block.
  - The skill takes an optional `--pr <n>` argument for manual re-runs (`/gsd-changie-fragments 3 --pr 88`). An explicit argument wins over the lookup.
  - — **Reversibility:** reversible. The lookup lives in the skill only.
- **D-04:** In codegraph-go, the milestone PR `gsd/v0.15.0-milestone → main` is opened as a **draft PR during this phase**, before Phase 3 closes. That PR is the `<n>` every fragment of this milestone carries, which is accurate because every change ships in it. Opening the PR is outward-facing, so the plan makes it a **checkpoint the maintainer approves**; it is not an autonomous step.
  - The researcher confirms how `/gsd-ship` behaves when a PR for the branch already exists: does it reuse or update the PR, or fail? The plan records the answer so the milestone close does not open a second PR.
  - The draft PR runs CI on every push. Checks that fail on a draft body, such as PR-template policy or issue link, are expected and do not block this phase.
  - — **Reversibility:** reversible. A draft PR can be closed, and fragments can be edited before batch.
  - Rejected: a config value holding the PR number. It is stale on the next milestone and silently wrong on any other branch.
  - Rejected: a placeholder such as `PR=1`. The fragment would link to the wrong PR, and an invented number presented as real breaks the "evidence, not assertion" rule.

### What the skill writes (CAP-03)
- **D-05:** The skill splits the work into a judgment half and a deterministic half.
  - **Judgment (the model):** read every `*-SUMMARY.md` in the phase directory, decide which changes are user-visible, and choose each change's kind and sentence.
  - **Deterministic (a bundled script in the skill directory, e.g. `scripts/write-fragments.sh`):** everything else — the preflight skips (disabled key, missing command, no `.changie.yaml`, no PR), the PR lookup, the idempotency check (D-07), one `CI=true <changie_command> new -k <Kind> -b <body> -m PR=<n>` per entry, and the commit.
  - The script reads entries from a file or stdin, one `kind<TAB>body` per line. It passes each body as a **single argv element**, never through `eval` or a re-quoted shell string. SUMMARY text is untrusted input to a shell (threat model item). The script must be POSIX-ish bash with no `jq` dependency; `gh` and `git` are its only external tools besides changie.
  - This split makes the skip cases and the write path testable without a model. That is what CAP-03's scratch-project proof needs.
  - The researcher confirms that capability install copies a skill's supporting files (not only `SKILL.md`). If it does not, the script body goes inline in `SKILL.md` as fenced blocks, with the same argv discipline.
  - — **Reversibility:** reversible.
- **D-06:** **A user-visible change** is anything a user of the released artifact can observe:
  - a CLI command, flag or output
  - MCP tool behaviour
  - install, upgrade or verification behaviour
  - index format or compatibility
  - a measurable performance change
  - a shipped dependency bump

  Planning artifacts, CI and workflow changes, tests, internal refactors and contributor tooling are **not** user-visible, even when a SUMMARY describes them at length. Rules:
  - **One fragment per change, not per plan.** Merge duplicates across plans.
  - **Kind:** chosen only from the kinds parsed out of the host's `.changie.yaml` at run time, never from a hard-coded list. `Breaking` applies when a command, flag, tool, output field or on-disk format is removed or renamed incompatibly, or when a commit in the phase carries `!` / `BREAKING CHANGE`. The phase's conventional-commit types (`feat`→Features, `fix`→Fixes, `perf`→Performance, `deps`/`build(deps)`→Dependencies) are hints that inform the SUMMARY reading; they do not replace it.
  - **Body:** one plain sentence addressed to users. No requirement IDs, decision IDs, plan numbers or phase jargon. Example: "Removed the hidden `query` and `unlock` rename stubs; run `codegraph explore` instead."
  - — **Reversibility:** reversible. Fragments are editable until batch, and the Phase 4 gate is the hard backstop.
- **D-07:** **Idempotency.** `verify:post` is dispatched by more than one host workflow: `execute-phase.md` at phase completion, `verify-work.md` on a pass, `autonomous.md` after verification, and others that match only their own skill. So one phase close can dispatch the skill two or more times. Rules:
  - The fragment commit carries git trailers `Changie-Phase: <N>` and `Changie-Summaries: <comma-separated plan ids covered>`.
  - Before writing, the script reads the trailers of the current branch's commits and processes **only SUMMARY files not already covered**.
  - A second dispatch with nothing new prints "already recorded" and returns.
  - Gap-closure plans that add new SUMMARYs later still get their fragments.
  - A run that decides "no user-visible change" writes no commit. A re-run re-evaluates and reaches the same skip, which is acceptable.
  - — **Reversibility:** reversible.
  - Rejected: a marker file in `.planning/phases/`, which adds structure to a tool-owned tree. Rejected: a `Phase` custom field in `.changie.yaml`, which changes the vocabulary Phase 1 locked.
- **D-08:** **Commit convention.**
  - Subject: `docs(<padded-phase>): add changelog fragments`. Body: the trailers from D-07.
  - The commit runs through plain `git commit` with an **explicit pathspec** listing only the fragment files this run wrote. Never `git add -A` or `git add .`. Pre-existing uncommitted fragments in `.changes/unreleased/` stay untouched.
  - Not through `gsd_run query commit`, which adds no trailers (memory `gv8ppm3cce`) and is scoped to planning docs.
  - The commit is created regardless of `commit_docs`, because the fragments are product changelog input, not planning docs.
  - The skill never pushes. In the pipeline, commit signing follows rule `xmz3xknbj0`. The subject never contains `[ci skip]` / `[skip ci]` (rule `f18zrdsgx5`).
  - — **Reversibility:** reversible.

### Proofs and repository shape (CAP-01, CAP-02, CAP-03)
- **D-09:** The capability repository's proofs are one committed test script, for example `test/run.sh`. It works as follows:
  - It builds a throw-away git repository with a fixture `.changie.yaml`, fixture SUMMARY files and a stub `gh` on `PATH`. It then runs `gsd_run capability install ./ --scope project` (CAP-01).
  - It asserts that `gsd_run loop render-hooks verify:post --raw` **contains** the `changie-fragments` step with the key `true` and **does not** with the key `false`, both observed (CAP-02).
  - It drives the deterministic script through the write path and through **every** skip case: key disabled, no command, no `.changie.yaml`, no PR, nothing new or already recorded. It asserts the named reason each time.
  - It prints an executed-case count and fails on zero (rule `84d1gfpywd`). The positive write case is asserted on the exact fragment file and trailer, not on a green exit.
  - The model-judgment half (D-06) is proven by one real skill run against the fixture SUMMARYs, which must contain one user-visible change and one planning-only change. Its transcript is recorded in this phase's SUMMARY.
  - Test runs, RED transcripts and mutation evidence are pasted into **codegraph-go's** phase artifacts, because the capability repository's commit trail is outside this roadmap (ROADMAP Phase 2 Notes).
  - The RED demonstration mutates the skip or write guard, is confirmed applied, and is reverted byte-cleanly.
  - — **Reversibility:** reversible.
- **D-10:** Repository metadata.
  - Private visibility, created with `gh repo create seanb4t/gsd-capability-changie --private`. It is named by CAP-01 and so is authorized scope. Visibility stays private.
  - MIT `LICENSE`, verbatim text, matching codegraph-go.
  - A `README` covering: what the capability does, the config keys, install (`https://…git#<tag>`; the `v0.1.0` README still shows the `git+ssh` form, which also works), manual re-run, skip reasons, and trailer semantics.
  - The first tag is `v0.1.0`. Tags are annotated. A repository ruleset or branch protection is not required.
  - CI in the private repository is optional (Claude's discretion). The committed test script is the proof whether or not CI runs it.
  - — **Reversibility:** costly for the tag. CAP-04 and SHIP-04 pin it, so a re-tag means re-installing and re-recording.

### codegraph-go install (CAP-04)
- **D-11:** Install and configure in this order:
  - **Amended 2026-09-26 (maintainer, during 02-03):** the install spec is HTTPS, `https://github.com/seanb4t/gsd-capability-changie.git#v0.1.0`, not `git+ssh://git@github.com/...`. Why: installs must not block on SSH-agent access (a locked 1Password agent halted 02-03 once). The `gh` credential helper authenticates HTTPS for the private repo. This was verified on a scratch project with SSH disabled (`GIT_SSH_COMMAND=/usr/bin/false`): the install exited 0, and the `capability.json` it installed is byte-identical to `v0.1.0`'s. CAP-04's text in REQUIREMENTS.md and ROADMAP.md is updated to match. The capability repo's `v0.1.0` README still shows the `git+ssh` form, which also works; the next capability release updates it.
  1. `gsd_run capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project` (consent granted).
  2. `workflow.changie_fragments=true` and `workflow.changie_command="task changie --"` written through `gsd_run query config-set`, never by hand-editing `.planning/config.json`.

  Install comes first because the federated keys may be rejected by the config schema until the capability that declares them is installed. The researcher verifies this.
  - `.gitignore` already has `.gsd/` (line 46). Every other path the install writes (D-01 research) is gitignored in the same commit or proven not to exist.
  - After install, `git status --porcelain` must be empty apart from `.planning/config.json` and `.gitignore`.
  - — **Reversibility:** reversible.
- **D-12:** `CONTRIBUTING.md` gains a short subsection under `## What .planning/ is` (line 196), not under §Pull requests (that prose belongs to `DOCS-12`, Phase 5). The subsection says:
  - the one-time install command for each clone and the tag
  - that the repository is private, so the capability is optional for contributors without access
  - that without it, fragments are written by hand with `task changie -- new …`
  - that the Phase 4 `fragment-required` gate is the backstop either way
  - how to re-run the skill by hand

  — **Reversibility:** reversible.

### PR field made optional (maintainer decision, 2026-09-26, during 02-04)
- **D-13:** The `PR` custom field becomes **optional**. This supersedes D-03's "never make `PR` optional" and Phase 1 D-11's "leave `optional` unset".
  - **Why:** under the milestone-PR workflow, every fragment in a milestone carries the same PR number, so the per-line link is low-value. A required PR forced a draft milestone PR to be opened ahead of time at every milestone start, and that PR then conflicts with `/gsd-ship`'s create step at close.
  - **Scope, delivered by new plan 02-06, which runs before 02-05:**
    1. **codegraph-go.**
       - `.changie.yaml`: `PR` gets `optional: true`, keeping `type: int` and `minInt: 1`. Its header comment is updated. `changeFormat`'s `{{if .Custom.PR}}` already renders an entry with no PR cleanly.
       - REQUIREMENTS: CHG-01 and CHG-04 are amended in place. CHG-04's "missing `PR` is refused" becomes "a fragment without `PR` is accepted, and `PR=0` is still refused by `minInt: 1`".
       - Guards: Phase 1's shape test (`internal/upgrade/changie_shape_test.go`) and the `check:changie` refusal leg are re-pointed. Each re-pointed guard is demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`). The Go test change lands RED-first (rule `x1cjy9vyhq`). `check:changie`'s pass-count line is updated.
       - Phase 1's CHANGELOG byte-reproduction must stay green.
    2. **Capability `v0.1.1`.**
       - PR resolution becomes: explicit `--pr <n>`, else the open PR for the current branch, else **no PR**, meaning `changie new` runs without `-m PR=`. It prints a note, never a skip.
       - `gh` missing, `gh` auth failure and `no-open-pr` stop being skip reasons. When `gh` is present but errors, the run notes it and continues without a PR.
       - `--pr 0` is still refused.
       - `test/run.sh` legs are updated: RED first, then GREEN, with an executed-leg count.
       - The README's install line switches to HTTPS (D-11 amendment), and a CHANGELOG or release note is added in the capability repo if one exists.
    3. **Publish.** Push `main` and annotated tag `v0.1.1` to the PRIVATE repo, **pre-approved by the maintainer** (no checkpoint). Prove the tag from a clean HTTPS clone. Never re-point `v0.1.0`; never change visibility.
    4. **Upgrade.** Upgrade codegraph-go's project-scope install AND the global install (added by 02-04's `global-install` answer) to `v0.1.1` through gsd-core verbs. Re-verify: the `capability.json` byte identity, `render-hooks verify:post`, and `~/.claude/skills/gsd-changie-fragments/` materialization.
  - **Kept:** draft PR #88 stays open as the eventual milestone PR. This milestone's fragments may still carry `#88` via the open-PR lookup. The Phase 4 `fragment-required` gate remains the backstop that a fragment exists.
  - **Reversibility:** reversible. Re-adding the requirement is a one-line config change plus guard re-points.
- **D-14:** 02-04 Task 2 was answered **`global-install`** by the maintainer. `/gsd-changie-fragments` is materialized from a global-scope install, and this machine's global `~/.gsd` and `~/.claude/skills` gain the capability. 02-05's CONTRIBUTING text documents this per-machine step.

### Claude's Discretion
- Script and test file names and layout inside the capability repository. SKILL.md wording beyond the rules above.
- Whether the capability repository gets a minimal CI (shellcheck plus `test/run.sh`).
- Exact skip-reason wording, provided each case prints a distinct, named reason.
- How the script discovers the phase directory from a phase number. Prefer `gsd_run query init.phase-op <N>` over globbing if it is available to a capability skill.

### Folded Todos
- **Adopt changie for changelog + version and replace release-please** (`.planning/todos/pending/2026-09-25-adopt-changie-replace-release-please.md`, score 0.9). This phase delivers checklist step 6 ("GSD hook. Phase close writes fragments non-interactively"). The "where this lives" question the todo left open is answered by memory `f7m4ye3rkc`: a private third-party capability. The todo stays pending for Phase 5.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and design
- `.planning/REQUIREMENTS.md` §Phase-Close Capability: CAP-01..CAP-05 acceptance text. §Out of Scope: no `contribution` / `ref.command` step, no gsd-core workflow edits.
- `.planning/ROADMAP.md` §Phase 2: success criteria 1-4 and the Notes paragraph (PR-number hazard, tool-owned `config.json`, the scratch-project proofs are the capability's own tests). §Phase 3: CAP-05 is proven at that phase's close.
- `.planning/notes/changie-release-management.md`: the adopted design and the flow `phase close → changie new -k <kind> -b … -m PR=<n>` (line 53).
- `.planning/phases/01-changie-baseline/01-CONTEXT.md`: D-04/D-05 (changie runs as `task changie --` from `go.tool-changie.mod`), D-10/D-11 (vocabulary locked, `PR` stays required).
- `.changie.yaml`: the host kinds and the required `PR` custom field that the skill must parse and obey. The header comment names Phase 2 as the owner of the PR-number problem.

### gsd-core 1.14.0 capability mechanism (tool-owned; read, never edit)
- `~/.claude/gsd-core/references/loop-hook-dispatch.md`: `step` dispatch (`Skill(gsd-<ref.skill>)`), advisory semantics, `onError`.
- `~/.claude/gsd-core/workflows/execute-phase.md:1105`, `verify-work.md:562`, `autonomous.md:542`, `secure-phase.md:32`, `validate-phase.md:32`, `audit-milestone.md:162`: every `verify:post` call site. This is why D-07 exists.
- `~/.claude/gsd-core/bin/lib/capability-validator.cjs`: manifest schema, config-slice validation (`type`/`default`/`description`), reserved ids, `activationKey`, `engines`.
- `~/.claude/gsd-core/bin/lib/capability-loader.cjs`, `capability-writer.cjs`, `capability-source.cjs`, `capability-ledger.cjs`, `capability-consent.cjs`: install spec kinds (`./`, `https://…git#ref`, `git+ssh://…#ref`), skill materialization, the ledger, consent.
- Upstream `docs/reference/capability-manifest.md` in `open-gsd/gsd-core` (1.14.0 tag): the manifest reference that memory `f7m4ye3rkc` was verified against.

### codegraph-go integration points
- `Taskfile.yml:13` (`GO_TOOL_CHANGIE`) and `:538` (`changie` wrapper target): the command that `workflow.changie_command` points at.
- `.gitignore:46` (`.gsd/`).
- `CONTRIBUTING.md` §What `.planning/` is (line 196) for the D-12 subsection. Lines 124-128 are the changie tool bullet, already present from Phase 1.

### Standing rules and memory (engram)
- Rules `84d1gfpywd` (positive assertion), `x1cjy9vyhq` (Go RED evidence; this phase's proofs are shell, so RED is shown as a mutation transcript), `f18zrdsgx5` (no `[ci skip]`), `xmz3xknbj0` (pipeline commit signing).
- Memories `f7m4ye3rkc` (capability decision and mechanism facts), `r19fvsz4tf` (hazard 4: no PR number at phase close), `gv8ppm3cce` (`gsd_run query commit` adds no trailers), `sj7a76sd5d` (changie v1.26.0 gotchas: sub-second `fragmentFileFormat`, silent overwrite; two fragments of the same kind written in one run need the sub-second format that is already set).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The `task changie` wrapper (`Taskfile.yml:538`) is the invocation the skill uses in this repository through `workflow.changie_command`.
- `check:changie` (Phase 1, 01-02) already proves `CI=true changie new … -m PR=<n>` is non-interactive and that a missing `PR` or an unknown kind is refused. The skill relies on those refusals as a second line of defence behind its own kind parsing.
- The drift-guard shape (count before compare, named `::error::`, scratch copy) from `docs:cli:drift` / `check:changie` is the model for the capability's `test/run.sh`.

### Established Patterns
- Tool-owned files are changed only through the tool's verbs (`gsd_run query config-set`, `gsd_run capability install`). `.planning/config.json` is never hand-edited (planning-artifacts rule).
- Every guard prints a positive count and fails loud on zero. Every RED demonstration is a confirmed-applied, byte-cleanly-reverted mutation with a pasted transcript.
- A phase directory can carry extra evidence files (`01-MUTATION-LOG.md` precedent) for proofs that run outside the repository.

### Integration Points
- `.planning/config.json` `workflow` block gains `changie_fragments` and `changie_command` (via config-set).
- `.gitignore` gains any install-output path besides `.gsd/` (D-01 research).
- `CONTRIBUTING.md` gains one subsection (D-12).
- GitHub: the private repository `seanb4t/gsd-capability-changie` (created) and a draft PR for `gsd/v0.15.0-milestone` (opened at a maintainer checkpoint, D-04).

</code_context>

<specifics>
## Specific Ideas

- The first real input to the capability is Phase 3's stub removal. Its expected fragment is `Breaking`, with a body like "Removed the hidden `query` and `unlock` rename stubs…", and it carries the draft milestone PR's number.
- The skip path gets a natural second observation at Phase 4's close, because the gate is CI-only and has no user-visible change (ROADMAP Phase 3 Notes). The planner should not treat that as a failure.
- `sj7a76sd5d`: the sub-second `fragmentFileFormat` in `.changie.yaml` is what keeps two same-kind fragments written in one skill run from silently overwriting each other. The test's write case should write two fragments of the same kind and assert both files exist.

</specifics>

<deferred>
## Deferred Ideas

- Porting the capability to router-hosts, engram, fovea and fzymgc-house-skills: `PORT-01`, v2.
- Proposing a first-party changie capability to `open-gsd/gsd-core`: `PORT-02`, v2, once the private capability has proven the shape on two repositories.
- A `Phase` custom field on fragments for traceability. It would change the vocabulary Phase 1 locked. Revisit only if the trailer approach in D-07 proves insufficient.
- Making the capability public: a later call, out of this milestone.

### Reviewed Todos (not folded)
- `2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md` (score 0.6): matched only on keywords ("tools/phase/milestone"). The bench runner's checkout check has nothing to do with changie. Stays pending (same verdict as Phase 1).
- `2026-09-25-reply-on-gh-85-with-reshaped-contract.md` (score 0.6): matched only on keywords. The GH #85 server-mode reply belongs to the next milestone. Stays pending (same verdict as Phase 1).

</deferred>

---

*Phase: 02-phase-close-fragment-capability*
*Context gathered: 2026-09-25 via /gsd-discuss-phase 2 --auto*
