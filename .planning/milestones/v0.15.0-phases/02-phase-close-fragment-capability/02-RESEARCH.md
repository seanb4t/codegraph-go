# Phase 2: Phase-Close Fragment Capability - Research

**Researched:** 2026-09-25
**Domain:** gsd-core 1.14.0 third-party capability mechanism (manifest, install, materialization, loop-hook dispatch); `gh`/git trailer plumbing; changie non-interactive invocation
**Confidence:** HIGH for everything tagged `[VERIFIED]` below (all obtained by running the real `gsd-tools` binary against a throw-away scratch project in this session's scratchpad, or by reading the cited source file this session). MEDIUM/LOW where noted.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Capability identity and manifest
- **D-01:** The capability id is `changie`. The skill stem is `changie-fragments`. gsd-core dispatches a `ref.skill` step as `Skill(skill="gsd-<ref.skill>")` (`references/loop-hook-dispatch.md`, `workflows/verify-work.md:562ff`, `workflows/autonomous.md:542ff`), so the host invokes the skill as **`gsd-changie-fragments`**. This is the name a user types to re-run it by hand. The id avoids the reserved `gsd-` / `gsd-core-` / `anthropic-` prefixes, and the repository name `gsd-capability-changie` is not the id. — **Reversibility:** costly. The id is recorded in the per-scope install ledger of every clone that installs it, and the skill name appears in docs and manual re-runs.
  - The researcher must confirm by a real install on a scratch project, not by reading the loader, that a project-scope install makes the skill resolvable to Claude Code as `gsd-changie-fragments`. The researcher also records where the files land (`.gsd/capabilities/changie/…` and any runtime skills directory) so that CAP-04 can gitignore every path the install writes.
- **D-02:** The manifest declares two federated config keys in its `config` slice:
  - `workflow.changie_fragments` (`boolean`, default `true`). This is the step's `when` gate, exactly as CAP-02 states it.
  - `workflow.changie_command` (`string`, default `"changie"`). This is the command prefix the skill runs changie through.

  There is exactly one `verify:post` step: `ref.skill: "changie-fragments"`, `onError: "skip"`, gated on the first key. `engines.gsd` is a range anchored on the tested 1.14.0. Use caret-style `^1.14.0` unless the validator in `capability-validator.cjs` requires another syntax; the researcher reads the validator's accepted form and the loader's behaviour on an engine mismatch. — **Reversibility:** reversible. Config keys can be added later, but removing one breaks any clone that set it.
  - **Why `workflow.changie_command` exists:** in this repository changie is **not on `PATH`**. It runs as `task changie -- …`, through `GO_TOOL_CHANGIE` and the isolated `go.tool-changie.mod` (Phase 1 D-04/D-05, `Taskfile.yml:13`, `:538`, CONTRIBUTING.md:124-128). CAP-03's literal "skip when `changie` is not on `PATH`" would therefore **always** skip here, and CAP-05 could never pass. The skill resolves the configured command instead. The "not on `PATH`" skip case becomes "the configured command's executable cannot be found or `<command> --version` fails". The default `changie` keeps the literal CAP-03 behaviour for repositories that install changie as a binary. codegraph-go sets the key to `task changie --`. The verifier must read CAP-03's first skip case as "configured changie command unavailable". Record this interpretation in the phase VERIFICATION. Do not paper over it.

#### Where the PR number comes from (ROADMAP Phase 2 Notes, hazard 4, Phase 1 D-11)
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

#### What the skill writes (CAP-03)
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

#### Proofs and repository shape (CAP-01, CAP-02, CAP-03)
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
  - A `README` covering: what the capability does, the config keys, install (`git+ssh…#<tag>`), manual re-run, skip reasons, and trailer semantics.
  - The first tag is `v0.1.0`. Tags are annotated. A repository ruleset or branch protection is not required.
  - CI in the private repository is optional (Claude's discretion). The committed test script is the proof whether or not CI runs it.
  - — **Reversibility:** costly for the tag. CAP-04 and SHIP-04 pin it, so a re-tag means re-installing and re-recording.

#### codegraph-go install (CAP-04)
- **D-11:** Install and configure in this order:
  1. `gsd_run capability install git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project` (consent granted).
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

### Claude's Discretion
- Script and test file names and layout inside the capability repository. SKILL.md wording beyond the rules above.
- Whether the capability repository gets a minimal CI (shellcheck plus `test/run.sh`).
- Exact skip-reason wording, provided each case prints a distinct, named reason.
- How the script discovers the phase directory from a phase number. Prefer `gsd_run query init.phase-op <N>` over globbing if it is available to a capability skill.

### Folded Todos
- **Adopt changie for changelog + version and replace release-please** (`.planning/todos/pending/2026-09-25-adopt-changie-replace-release-please.md`, score 0.9). This phase delivers checklist step 6 ("GSD hook. Phase close writes fragments non-interactively"). The "where this lives" question the todo left open is answered by memory `f7m4ye3rkc`: a private third-party capability. The todo stays pending for Phase 5.

### Deferred Ideas (OUT OF SCOPE)
- Porting the capability to router-hosts, engram, fovea and fzymgc-house-skills: `PORT-01`, v2.
- Proposing a first-party changie capability to `open-gsd/gsd-core`: `PORT-02`, v2, once the private capability has proven the shape on two repositories.
- A `Phase` custom field on fragments for traceability. It would change the vocabulary Phase 1 locked. Revisit only if the trailer approach in D-07 proves insufficient.
- Making the capability public: a later call, out of this milestone.

**Reviewed Todos (not folded):**
- `2026-08-14-bench-pinnedat-validates-a-checkout-by-git-rev-parse-head-alone.md` (score 0.6): matched only on keywords ("tools/phase/milestone"). The bench runner's checkout check has nothing to do with changie. Stays pending (same verdict as Phase 1).
- `2026-09-25-reply-on-gh-85-with-reshaped-contract.md` (score 0.6): matched only on keywords. The GH #85 server-mode reply belongs to the next milestone. Stays pending (same verdict as Phase 1).
</user_constraints>

## Summary

This phase installs a private third-party gsd-core capability. Every open research question in `02-CONTEXT.md` was answered by **running the real, installed `/Users/sean/.local/bin/gsd-tools` (gsd-core 1.14.0) against throw-away scratch projects** under this session's scratchpad — never by reading source alone — per the phase's own instruction. The capability manifest shape, the `verify:post` gate mechanics, the config-key install ordering, and the `changie` fragment-writing mechanics all check out and are buildable exactly as `02-CONTEXT.md` designed them.

**One finding changes the plan and must be resolved before CAP-04 is written as a task:** Claude Code's runtime descriptor in gsd-core's own capability registry declares `artifactLayout.local = [commands, agents]` — **no `skills` kind at all** — while `artifactLayout.global = [skills, agents]` carries `"skillsGlobalOnboarding": true`. This was confirmed twice: by reading the registry entry verbatim, and by running `capability install --scope project` followed by the documented materialization command (`capability set <id> --runtime claude --scope local --config-dir <dir>`) against a real scratch project — it wrote 72 first-party `.claude/commands/gsd-*.md` files and 64 `.claude/agents/gsd-*.md` files, but **zero** files under any `.claude/skills/` path, and the third-party `changie-fragments` skill appeared in **neither** output tree. A bare `capability set --runtime claude --scope global` against a non-gsd-core-hosted directory also failed outright (`Runtime Surface source is unavailable or incomplete`), because global-scope materialization requires the target `--config-dir` to already be a self-hosted, "runtime"-authority gsd-core install (which is what `~/.claude` already is for this user, since gsd-core is installed there globally). **Practical implication:** project-scope capability install (as CAP-01/CAP-04 correctly specify, for the bundle/ledger/config) does **not** by itself make `gsd-changie-fragments` dispatchable via the Skill tool. A separate, additional **global** materialization step against the user's real `~/.claude` is what actually needs to happen for the skill body to become resolvable — CAP-04's plan must include and prove this step, not assume the project-scope install alone is sufficient. This is `[VERIFIED]` from two independent angles (registry data + live command run) and is the single highest-value finding in this research pass.

Everything else is corroborating, lower-risk detail: the manifest needs one field CONTEXT.md's example didn't call out (`runtimeCompat`, required by the validator even though its absence produces a fairly generic error); the capability's install ledger (`.gsd-capabilities.json`) lands at the **project root**, not inside `.gsd/`, so it needs its own `.gitignore` line; `when` is a bare dotted config key evaluated with `Boolean()`, not an expression language; `/gsd-ship`'s `create_pr` step calls `gh pr create` **unconditionally** with no existing-PR detection anywhere in `preflight_checks`, so it **will fail**, not reuse, when a PR already exists for the branch (directly answering D-04's open question); and `changie new` does not print the fragment path it wrote — the project's own `check:changie` Taskfile target already establishes the pattern (`find .changes/unreleased -type f -name '*.yaml'`) that the skill's deterministic script must reuse.

**Primary recommendation:** Build the capability manifest and skill exactly as `02-CONTEXT.md` D-01 through D-12 specify, with the one manifest addition (`runtimeCompat`) below, and add an explicit CAP-04 sub-task that runs (and proves, with `find`/`ls` evidence) a **global-scope** Claude materialization step in addition to the project-scope capability install — do not treat "installed" and "dispatchable" as the same claim.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Capability manifest + `verify:post` step registration | gsd-core capability loader (Node/CJS, `~/.claude/gsd-core/bin/lib/*.cjs`) | — | Owns validation, config-slice schema, `when` gating, loop-point dispatch. This phase writes data the loader consumes; it never patches the loader itself (out of scope by requirement). |
| `changie-fragments` skill body (judgment half) | Claude agent (the orchestrator dispatching `Skill(gsd-changie-fragments)`) | — | Reading SUMMARY.md files and choosing kind/body is a judgment task; per `loop-hook-dispatch.md` it is dispatched via the Skill tool, not a shell script. |
| `write-fragments.sh` (deterministic half) | Bundled shell script under `.gsd/capabilities/changie/skills/changie-fragments/scripts/` | changie CLI (`task changie --`) | Preflight skips, PR lookup (`gh`), idempotency check (`git log` trailers), and the actual `changie new` / commit calls are pure, testable shell — no model needed. |
| PR-number resolution | `gh pr view` / `gh pr list --head` (GitHub, via `gh` CLI) | — | The only authoritative source for "the open PR on this branch"; never a config value or placeholder (D-03). |
| Capability install/materialization | gsd-core capability lifecycle + runtime-artifact-layout (Node/CJS) | Claude Code's own skill-discovery (reads `.claude/skills/` and `~/.claude/skills/` at session start) | Install stages the bundle under `.gsd/capabilities/<id>/`; a **separate** materialize step (global-scope, for Claude specifically) is what actually makes the skill dispatchable — see Summary. |
| Config keys (`workflow.changie_fragments`, `workflow.changie_command`) | `.planning/config.json` (tool-owned, written only via `gsd-tools config-set`) | — | Federated keys become legal only after the declaring capability is installed (confirmed, see Pitfall/D-11 section). |

## Standard Stack

This phase installs no npm/pip/cargo packages — see **Package Legitimacy Audit** below for why that gate does not apply. The "stack" here is entirely first-party tooling already present in the environment:

| Tool | Version (observed) | Purpose | Why Standard |
|------|---------------------|---------|---------------|
| `gsd-core` capability system | 1.14.0 `[VERIFIED: ~/.claude/gsd-core, this session]` | Manifest schema, `verify:post` dispatch, config federation | The only sanctioned extension surface for phase-close automation (Out of Scope table forbids editing gsd-core workflow files directly) |
| `gh` CLI | 2.101.0 `[VERIFIED: gh --version, this session]` | PR-number lookup (D-03), draft PR creation (D-04) | Already the project's standing GitHub automation tool (used throughout `ship.md`) |
| `git` | system git (used via `git log --format='%(trailers:...)'`, `git commit`) `[VERIFIED: real trailer round-trip test, this session]` | Idempotency trailers (D-07), fragment commit (D-08) | No alternative considered; this is plumbing, not a library choice |
| `changie` v1.26.0+ | pinned via `go.tool-changie.mod`, invoked as `GOWORK=off go tool -modfile=go.tool-changie.mod changie` (Taskfile `GO_TOOL_CHANGIE`, `Taskfile.yml:13`) `[VERIFIED: task changie -- --version → "changie version vdev", this session]` | Fragment authoring | Locked by Phase 1 (CHG-01..04); this phase only consumes it via the configured `workflow.changie_command` |
| `awk`/`sed` (POSIX) | system | Parse `.changie.yaml`'s `kinds:` list without `jq`/`yq` | `[VERIFIED]` — see Code Examples; D-05 explicitly forbids a `jq` dependency |

**Installation:** N/A — no package manager install step. The capability is installed with `gsd-tools capability install git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project` (D-11).

**Version verification:** `gh --version` and `task changie -- --version` were run live in this session (see Sources/Environment Availability). `gsd-core` version was cross-checked two ways: `capability-registry.cjs`'s embedded `"version": "1.14.0"` entries, and a live `engines.gsd: "^1.14.0"` install succeeding without an `engines.gsd` mismatch error (which would have named the actual running version — see Code Examples).

## Package Legitimacy Audit

**Not applicable to this phase.** CAP-01/CAP-04 install a **private git-hosted gsd-core capability** (`git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#<tag>`), not an npm/PyPI/crates package resolved through a public registry. `gsd-tools query package-legitimacy check` targets registry packages (`--ecosystem npm|pypi|crates`); a private git+ssh source has no registry entry to check, so a `SLOP`/`SUS`/`OK` verdict cannot be computed and would be meaningless here.

The equivalent trust boundary for a git-sourced capability is **install-time consent + human authorship**, not registry reputation:
- The repository is created and pushed by the same person authoring this milestone (D-10) — there is no third-party supply-chain trust decision to make.
- gsd-core's own install path **discloses** what it is about to do before writing anything: in this session's real install, the disclosure printed `"This capability ships no executable surfaces, but contributes agent instructions... these bodies are installed verbatim and are NOT content-scanned"` `[VERIFIED: real `capability install` output, this session — see Security Domain]`. That NOT-content-scanned disclosure is the actual risk surface for this phase and is carried into the Security Domain section below, not into a package-legitimacy table.
- If the capability repository later gains a `hooks[]` entry (an actual executable surface), gsd-core's own consent gate (`--yes` / the `aborted`+`requiresConsent` envelope) becomes the enforcement point — confirmed present and working via source read of `capability-command-router.cjs` (`installCapability` returns `status: "aborted"` and a non-zero exit without `--yes` when `capValidator`-visible executable surfaces are declared).

**Packages removed due to [SLOP] verdict:** none (N/A).
**Packages flagged as suspicious [SUS]:** none (N/A).

## Architecture Patterns

### System Architecture Diagram

```
                     ┌─────────────────────────────────────────┐
                     │ seanb4t/gsd-capability-changie (private) │
                     │  capability.json + skills/changie-       │
                     │  fragments/{SKILL.md, scripts/*.sh}      │
                     │  test/run.sh (proofs, D-09)               │
                     └───────────────┬───────────────────────────┘
                                     │ git+ssh://...#v0.1.0  (D-11 step 1)
                                     ▼
       gsd-tools capability install --scope project
                                     │
                                     ▼
   codegraph-go/.gsd/capabilities/changie/          codegraph-go/.gsd-capabilities.json
   (full bundle copy: capability.json,                (ledger — PROJECT ROOT, not
    skills/changie-fragments/SKILL.md,                 under .gsd/ — needs own
    skills/changie-fragments/scripts/*.sh)              .gitignore line, see Pitfall 2)
                                     │
                                     │  gsd-tools config-set (D-11 step 2, AFTER install)
                                     ▼
             codegraph-go/.planning/config.json
             workflow.changie_fragments=true
             workflow.changie_command="task changie --"
                                     │
              ┌──────────────────────┴───────────────────────────┐
              │  MISSING STEP (this research's central finding)  │
              │  capability set changie --runtime claude          │
              │  --scope global --config-dir ~/.claude             │
              │  → materializes ~/.claude/skills/gsd-changie-      │
              │    fragments/SKILL.md (+scripts) so the Skill      │
              │    tool can actually resolve it                    │
              └──────────────────────┬───────────────────────────┘
                                     ▼
   Any gsd-core workflow reaching `verify:post`
   (execute-phase.md, verify-work.md, autonomous.md, ...)
                                     │
                    gsd-tools loop render-hooks verify:post
                    → activeHooks[] includes {capId:"changie",
                      ref:{skill:"changie-fragments"},
                      when:"workflow.changie_fragments", onError:"skip"}
                                     │
                     Skill(skill="gsd-changie-fragments")  (agent dispatch)
                                     │
                 ┌───────────────────┴────────────────────┐
                 │ judgment (agent): read *-SUMMARY.md,    │
                 │ decide user-visible changes, kind+body  │
                 └───────────────────┬────────────────────┘
                                     ▼
        scripts/write-fragments.sh <kind>\t<body> per line (stdin/file)
                 │  preflight skips → PR lookup (gh) → idempotency
                 │  check (git log trailers) → CI=true task changie --
                 │  new -k <Kind> -b "<body>" -m PR=<n>  → find
                 │  .changes/unreleased -type f -name '*.yaml' (new files)
                 ▼
      git commit -m "docs(<phase>): add changelog fragments"
      (explicit pathspec; trailers Changie-Phase / Changie-Summaries)
```

### Recommended Project Structure (capability repository)
```
gsd-capability-changie/
├── capability.json              # manifest — see Code Examples
├── README.md                    # config keys, install cmd, skip reasons, trailers
├── LICENSE                      # MIT, verbatim
├── skills/
│   └── changie-fragments/
│       ├── SKILL.md             # judgment half — dispatched as gsd-changie-fragments
│       └── scripts/
│           └── write-fragments.sh   # deterministic half (D-05)
└── test/
    └── run.sh                   # D-09 proofs (CAP-01/CAP-02/CAP-03)
```

### Pattern 1: Manifest `when` is a bare dotted config key, not an expression
**What:** `step.when` (and `contribution.when`) is validated as "a string if present" (`capability-validator.cjs` `validateStep`), and resolved at render time by `_resolveActivationValue(dotKey, config, cwd, registry)` which walks the dotted segments and returns `Boolean(value)`. There is no `==`, `!=`, or boolean-combinator grammar.
**When to use:** Any single boolean config gate on a `step`/`contribution`. For anything needing equality-to-a-specific-value gating there is a **separate** field, `pointFrom`, which is not needed here.
**Example:**
```json
// Source: ~/.claude/gsd-core/bin/lib/capability-activation.cjs (read this session)
"when": "workflow.changie_fragments"
```
```js
// _resolveActivationValue — capability-activation.cjs:104-107 (verbatim, read this session)
function _resolveActivationValue(dotKey, config, cwd, registry) {
    const r = resolveConfigKey(dotKey, { config, cwd, registry });
    return r.found ? Boolean(r.value) : false;
}
```

### Pattern 2: A capability's `config` slice legally uses dotted keys
**What:** `validateFeatureBody`'s config-key loop (`capability-validator.cjs:604-618`) only rejects `''`, `__proto__`/`constructor`/`prototype`, and a missing/malformed `type`/`default`/`description` — it does **not** restrict the key's character set. `workflow.changie_fragments` and `workflow.changie_command` installed and resolved correctly in the real test.
**Example:** see the Code Examples capability.json below.

### Pattern 3: Deterministic script discovers written files by directory diff, never by parsing `changie new`'s stdout
**What:** `changie new -k <Kind> -b "<body>" -m PR=<n>` prints nothing that names the fragment file it wrote (confirmed: `changie new --help` documents only `-d/--dry-run` to print the fragment content, no flag prints the destination path). The project's own `check:changie` Taskfile target (`Taskfile.yml`, `check:changie`, leg 4/11) discovers the new fragment with `find "${work}/.changes/unreleased" -type f -name '*.yaml'` against a directory it already knows was empty (or by counting before/after).
**When to use:** `write-fragments.sh` must snapshot `.changes/unreleased/*.yaml` (or its mtime) before each `changie new` call and diff after, exactly like `check:changie` leg 4 does — this is the established, precedent-matching technique, not an invention.
**Example:**
```bash
# Source: Taskfile.yml check:changie leg 4/11 (read this session) — pattern, not verbatim copy
before_count=$(find "${dir}" -type f -name '*.yaml' | wc -l)
CI=true ${CHANGIE_CMD} new -k "${kind}" -b "${body}" -m "PR=${pr}"
frag=$(find "${dir}" -type f -name '*.yaml' -newer /tmp/marker)   # or: comm against a pre-listed set
```

### Anti-Patterns to Avoid
- **Assuming project-scope `capability install` alone makes a skill Skill-tool-dispatchable.** It does not, for the Claude runtime, in this gsd-core version — see Summary and Pitfall 1.
- **Parsing `.changie.yaml` with `jq`/`yq`.** D-05 forbids it; it is also unnecessary — the file's `kinds:`/`custom:` block structure is trivially `awk`-parseable (see Code Examples), and it is not JSON.
- **Re-quoting the SUMMARY-derived body into a shell string.** D-05 already forbids this; confirmed load-bearing by `loop-hook-dispatch.md`'s identical warning for `ref.command`/`gate.check.predicate` values — the same shell-metacharacter injection class applies to any third-party-manifest-adjacent or model-derived text reaching a shell.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Discovering the phase directory from a phase number | Globbing `.planning/phases/0*-*` and guessing | `gsd-tools init.phase-op <N> --raw` → `.phase_dir` `[VERIFIED, this session — see Code Examples]` | Already resolves padding, slug, and existence in one call; globbing risks matching the wrong phase on a renumbered/archived roadmap |
| Determining whether the changie config key is on | Hand-parsing `.planning/config.json` | `gsd-tools loop render-hooks verify:post --raw`'s `activeHooks` (or the manifest's own `when` gate, which the host already evaluates) | The host's own dispatch decision must be the single source of truth; a script computing its own answer can silently diverge from what actually got dispatched |
| PR-number lookup | A config value or placeholder number | `gh pr view --json number,state` / `gh pr list --head <branch> --state open --json number` (D-03) | Explicitly rejected by D-03's own reasoning: stale/wrong-PR risk; also the only real answer `gh` itself has |
| Idempotency tracking | A marker file in `.planning/phases/` or a new `.changie.yaml` custom field | Git trailers on the fragment commit (D-07), read with `git log --format='%(trailers:key=...,valueonly)'` `[VERIFIED, this session]` | Both alternatives are explicitly rejected in D-07 (tool-owned-tree pollution / vocabulary change); trailers are native git plumbing already used elsewhere in this milestone |

**Key insight:** every "don't hand-roll" here already has a first-party GSD or git/gh primitive; the phase's job is composition, not new infrastructure.

## Common Pitfalls

### Pitfall 1: Project-scope capability install does not materialize the skill for Claude Code
**What goes wrong:** `gsd-tools capability install <spec> --scope project --yes` succeeds (`status: "installed"`), `gsd-tools capability list` shows `"active": true, "surfaced": true`, and `gsd-tools loop render-hooks verify:post` correctly lists the step as active — everything *looks* done. But `Skill(skill="gsd-changie-fragments")` will fail to resolve, because no file was ever written under any `.claude/skills/` directory.
**Why it happens:** `capability-registry.cjs`'s `claude` runtime descriptor declares `artifactLayout.local: [commands, agents]` (no `skills` kind) and `artifactLayout.global: [skills, agents]` with `"skillsGlobalOnboarding": true` — Claude Code skills are, by this gsd-core version's own design, a **global-only** artifact, mirroring how gsd-core's own 72 first-party `/gsd-*` skills live at `~/.claude/skills/gsd-*`, never per-project. `[VERIFIED: ~/.claude/gsd-core/bin/lib/capability-registry.cjs:502-619, read this session]`
**How to avoid:** After `capability install --scope project` + the two `config-set` calls (D-11), CAP-04's plan must also run a materialization step that targets the real global Claude config home (`~/.claude`, or `$CLAUDE_CONFIG_DIR` if set) — e.g. `gsd-tools capability set changie --enable --runtime claude --scope global` (no `--config-dir` override, so it resolves the real global home) — and then **verify** with `find ~/.claude/skills -iname '*changie*'` that `gsd-changie-fragments/SKILL.md` actually landed. This is a genuinely new task the phase's Decisions section does not currently name; it must be added.
**Warning signs:** `capability list`/`render-hooks` reporting "active" is not evidence the skill is dispatchable — only a literal file-existence check under `.claude/skills/` (project or global) is.

### Pitfall 2: The install ledger is not covered by the existing `.gitignore` `.gsd/` line
**What goes wrong:** After `capability install --scope project`, `git status --porcelain` shows two new untracked paths: `.gsd/` (already ignored, `.gitignore:46`) **and** `.gsd-capabilities.json` at the **project root** — a sibling of `.gsd/`, not a child of it. `[VERIFIED: real install + `find`, this session — `.gsd-capabilities.json` landed at `<project-root>/`, confirmed against `LEDGER_FILE_NAME = '.gsd-capabilities.json'` and `path.join(runtimeDir, LEDGER_FILE_NAME)` in `capability-ledger.cjs:33,422`]`
**Why it happens:** `capabilitiesRoot(runtimeDir)` is `<runtimeDir>/.gsd/capabilities`, but the ledger writer uses `runtimeDir` directly, not the `.gsd/` subdirectory.
**How to avoid:** D-11's checkpoint ("Every other path the install writes ... is gitignored in the same commit") must add a **second** `.gitignore` line, `.gsd-capabilities.json`, alongside the existing `.gsd/` — not assume the existing line already covers it.
**Warning signs:** `git status --porcelain` after install showing anything other than `.planning/config.json` and `.gitignore` — D-11's own acceptance bar — catches this immediately if actually run, which the plan must do.

### Pitfall 3: `capability.json` needs `runtimeCompat`, which `02-CONTEXT.md`'s field list does not mention
**What goes wrong:** A manifest with exactly the fields D-01/D-02 enumerate (`id`, `role`, `title`, `description`, `tier`, `requires`, `version`, `engines.gsd`, `skills`, `agents`, `config`, `steps`, `contributions: []`, `gates: []`) fails `capability install` with `Capability validation failed: capability "changie" runtimeCompat must be an object with supported and unsupported arrays`.
**Why it happens:** `validateFeatureBody` (`capability-validator.cjs:558-561`) unconditionally calls `validateRuntimeCompat(cap.id, cap.runtimeCompat)`, and that validator requires a non-null object even though `runtimeCompat` is never mentioned in the manifest reference excerpt CONTEXT.md quotes.
**How to avoid:** Add `"runtimeCompat": { "supported": ["*"], "unsupported": [] }` to the manifest — the wildcard means "no runtime-specific exclusions," which is correct for a capability whose only surface is a `verify:post` skill step (no runtime-specific behavior). `[VERIFIED: reproduced the exact error, then fixed it, real install, this session]`
**Warning signs:** the literal error string above on a first install attempt.

### Pitfall 4: `/gsd-ship` will fail, not reuse, when a PR already exists for the branch
**What goes wrong:** D-04 opens a draft PR for `gsd/v0.15.0-milestone → main` during Phase 2. When the milestone later closes and `/gsd-ship` runs, its `create_pr` step calls `gh pr create` **unconditionally** — there is no existing-PR probe anywhere in `ship.md`'s `preflight_checks` or `create_pr` steps.
**Why it happens:** `[VERIFIED: ~/.claude/gsd-core/workflows/ship.md:360-378, read this session]` — quoting the entire `create_pr` step body:
```
<step name="create_pr">
Create the PR using the generated body. ...
gh pr create \
  --title "Phase ${PHASE_NUMBER}: ${PHASE_NAME}" \
  --body-file "${PR_BODY_FILE}" \
  --base "${BASE_BRANCH}"
```
No branch containing `gh pr view`/`gh pr list` and no conditional around this call exist in the file (`rg -n "<step name=" ship.md` lists only `initialize, preflight_checks, push_branch, generate_pr_body, create_pr, optional_review, track_shipping, ship_post_capability_dispatch, report` — `gh pr view` appears exactly once, at line 503, inside `track_shipping`, reading a PR that is assumed to already exist by number, not detecting one). `gh pr create` against a branch that already has an open PR errors with a GraphQL "A pull request already exists" message and a non-zero exit; nothing in `ship.md` catches or routes around that.
**How to avoid:** Editing `ship.md` is out of scope for this phase (explicit Out of Scope entry). The plan must instead **document** this as an operational fact for the maintainer at milestone close: either mark the existing draft PR ready (`gh pr edit <n> --ready` / `gh pr ready <n>`) directly instead of running `/gsd-ship`'s create step, or accept that `/gsd-ship` will need to be run in a mode/branch that does not already have an open PR. This directly answers D-04's open question — "reuse or fail?" — with **fail**, sourced from the workflow file itself, not assumed.
**Warning signs:** `gh pr create` stderr containing "A pull request for branch ... already exists."

## Code Examples

### Minimal valid `capability.json` (verified by a real `capability install`, this session)
```json
// Source: this session's scratch install at
// /private/tmp/.../scratchpad/gsd-capability-changie/capability.json
// — installed successfully with `gsd-tools capability install ./ --scope project`
// (exit 0, status:"installed") after adding runtimeCompat (Pitfall 3).
{
  "id": "changie",
  "role": "feature",
  "title": "Changie Fragments",
  "description": "Writes changie changelog fragments at phase close.",
  "tier": "standard",
  "requires": [],
  "version": "0.1.0",
  "engines": { "gsd": "^1.14.0" },
  "skills": ["changie-fragments"],
  "agents": [],
  "runtimeCompat": { "supported": ["*"], "unsupported": [] },
  "config": {
    "workflow.changie_fragments": {
      "type": "boolean",
      "default": true,
      "description": "Write changie changelog fragments automatically at phase close."
    },
    "workflow.changie_command": {
      "type": "string",
      "default": "changie",
      "description": "Command prefix used to invoke changie (e.g. \"task changie --\" when changie is not on PATH)."
    }
  },
  "steps": [
    {
      "point": "verify:post",
      "ref": { "skill": "changie-fragments" },
      "produces": [],
      "consumes": [],
      "when": "workflow.changie_fragments",
      "onError": "skip"
    }
  ],
  "contributions": [],
  "gates": []
}
```
Note: `ref.skill` is the **unprefixed stem** (`"changie-fragments"`, not `"gsd-changie-fragments"`) — the validator explicitly rejects a `gsd-`-prefixed value here (`capability-validator.cjs:2901-2909`, "double-prefix guard"); the host prepends `gsd-` only at dispatch time.

### Real `render-hooks` output, both states (CAP-02, this session)
```json
// gsd-tools loop render-hooks verify:post --raw, key absent (manifest default true applies)
{
  "point": "verify:post",
  "activeHooks": [
    {
      "capId": "changie",
      "kind": "step",
      "ref": { "skill": "changie-fragments" },
      "when": "workflow.changie_fragments",
      "onError": "skip"
    },
    ...
  ]
}
```
```bash
# gsd-tools config-set workflow.changie_fragments false --raw
# gsd-tools loop render-hooks verify:post --raw   → activeHooks no longer contains capId:"changie"
```

### D-11 ordering, both legs verified live
```bash
# BEFORE capability install (fresh scratch project, no changie capability installed):
$ gsd-tools config-set workflow.changie_command "task changie --" --raw
Error: Unknown config key: "workflow.changie_command". Valid keys: ...   # exit 1

# AFTER `gsd-tools capability install <spec> --scope project`:
$ gsd-tools config-set workflow.changie_command "task changie --" --raw
workflow.changie_command=task changie --   # exit 0
```

### `.changie.yaml` kinds parsing without jq (bash-only, verified against the real file)
```bash
# Source: ran against /Volumes/Code/github.com/seanb4t/codegraph-go/.changie.yaml, this session.
# Output: Breaking / Features / Fixes / Performance / Dependencies (exact match to the file's kinds:)
awk '/^kinds:/{flag=1; next} /^custom:/{flag=0} flag && /label:/{sub(/^[ \t]*-[ \t]*label:[ \t]*/,""); print}' .changie.yaml
```

### Git trailer round-trip (D-07, verified in a scratch git repo)
```bash
git commit -m "$(printf 'docs(02): add changelog fragments\n\nChangie-Phase: 2\nChangie-Summaries: 02-01,02-02\n')"
git log --format='%(trailers:key=Changie-Phase,valueonly)|%(trailers:key=Changie-Summaries,valueonly)'
# → 2|02-01,02-02   (confirmed real, this session)

# Recommended branch scope (matches ship.md's own ${BASE_BRANCH} convention, e.g. `git diff ${BASE_BRANCH}...HEAD`):
git log ${BASE_BRANCH}..HEAD --format='%(trailers:key=Changie-Summaries,valueonly)'
# two-dot (not ship.md's triple-dot diff form) is correct for `git log`: it walks only commits
# reachable from HEAD and not from the base, i.e. exactly "this branch's own commits" — verified
# semantics via a real base/HEAD divergence test, this session.
```

### `gsd-tools` phase-dir resolution (Claude's Discretion item)
```bash
$ gsd-tools init.phase-op 2 --raw
{
  ...
  "phase_dir": "/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-phase-close-fragment-capability",
  "padded_phase": "02",
  ...
}
```
**Important CLI-surface note:** the generic agent tooling docs reference `gsd_run query init.phase-op <N>` (a `query` subcommand prefix). The actual installed `gsd-tools` binary in this environment has **no `query` subcommand at all** — its top-level command list is `agent, agent-skills, ..., init, intel, capability, classify-confidence, ..., init.phase-op is not present either as a separate top-level entry; instead `init` itself takes the workflow name (`init execute-phase|plan-phase|...|phase-op`) as its first positional argument, i.e. the correct invocation is `gsd-tools init.phase-op 2 --raw`, not `gsd-tools query init.phase-op 2 --raw`. `[VERIFIED: `gsd-tools init` (no args) error output enumerating available init workflows including `phase-op`, and `gsd-tools init.phase-op 2 --raw` succeeding, both this session]`. The skill's `write-fragments.sh` must call the CLI form that actually exists in the target environment — confirm at execution time which gsd-tools build/version codegraph-go's contributors actually have, since this discrepancy could recur.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Manually hand-writing `.changes/unreleased/*.yaml` at phase close | `verify:post` capability dispatch of `gsd-changie-fragments` | This phase (CAP-01..04) | Removes the "human forgets" failure mode named in the phase description; CAP-04/D-12 keep the manual path documented as the fallback for contributors without repo access |

**Deprecated/outdated:** nothing in this phase deprecates prior codegraph-go behavior; it is additive.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | `gh auth status` exits non-zero with "You are not logged into any GitHub hosts" when unauthenticated (only the authenticated/exit-0 case was actually run this session) | D-03 skip-case design | Low — this is extremely well-established `gh` CLI behavior; if wrong, the skill's "gh not authenticated" skip case simply never fires and falls through to the "no open PR" skip case instead, which is still a safe no-prompt skip |
| A2 | `gh pr view --json number,state` / `gh pr list --head <branch> --state open --json number` exit non-zero (or print an empty/`null` result) when no PR exists for the branch, rather than hanging or prompting | D-03 | Low — documented `gh` behavior, not tested live here because doing so would require a real branch/PR round-trip against GitHub, which this research intentionally avoided per its own scope ("do not create the GitHub repo") |
| A3 | A `capability set --enable --runtime claude --scope global` (no `--config-dir` override) against the user's **real** `~/.claude` will successfully materialize `~/.claude/skills/gsd-changie-fragments/` the same way the project-scope `commands`/`agents` materialization succeeded against a scratch `--config-dir` | Pitfall 1 / Summary | **Medium-High** — this is the load-bearing fix for the central finding, but was deliberately NOT run against the real `~/.claude` in this session (mutating the researcher's live global Claude Code skill set was judged out of proportion to a research pass). CAP-04's plan MUST perform and observe this step for real, on the actual target machine, before treating the capability as "installed and dispatchable." Do not skip this checkpoint. |

**If this table is empty:** N/A — three items above need human/live confirmation at execution time, in particular A3.

## Open Questions

1. **Does `capability set changie --enable --runtime claude --scope global` (real `~/.claude`, no fake `--config-dir`) actually write `~/.claude/skills/gsd-changie-fragments/SKILL.md`?**
   - What we know: the registry declares `artifactLayout.global` includes a `skills` kind with `"skillsGlobalOnboarding": true`; the local-scope path (commands+agents) materialized correctly on the exact same code path pattern against a scratch `--config-dir`.
   - What's unclear: global-scope materialization additionally requires `resolveSourceProvider`'s `authority: 'runtime'` check to succeed, which failed against a scratch fake-home (`Runtime Surface source is unavailable or incomplete`) because that fake dir wasn't a real gsd-core install. The real `~/.claude` *is* a real gsd-core install, so this should succeed, but was not verified live (see Assumption A3).
   - Recommendation: CAP-04's plan makes this an explicit, verified task step with a `find ~/.claude/skills -iname '*changie*'` (or equivalent) assertion in its own acceptance criteria — not an assumption folded into "install the capability."

2. **What exact skip-message text should the fourth PR-lookup skip case (D-03) print, and does `gh`'s exact stderr differ between "no PR" and "not authenticated" in a way the script should distinguish?**
   - What we know: both cases must "print a named reason and return... no prompt, no block" (D-03).
   - What's unclear: this session did not exercise the "no PR exists" or "not authenticated" `gh` failure paths live (would require an unauthenticated `gh` or a branch with genuinely no PR against a real repo).
   - Recommendation: treat any non-zero exit from the PR lookup, regardless of message, as the same skip case with a generic "no PR number available (gh: <first line of stderr>)" message — simpler and does not depend on `gh`'s exact wording being stable across versions.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CAP-01 | Private repo `seanb4t/gsd-capability-changie` holds a `role:"feature"` capability whose `capability.json` validates under gsd-core 1.14.0 (`capability install ./ --scope project` succeeds on a scratch project); id not a reserved `gsd-` prefix; `engines.gsd` pinned; README, LICENSE, tagged release | Manifest shape reproduced and installed for real this session (Code Examples); `engines.gsd: "^1.14.0"` confirmed accepted; `runtimeCompat` requirement discovered and fixed (Pitfall 3); id `"changie"` passes the `KEBAB_RE`/non-reserved-prefix check (no `gsd-`/`gsd-core-`/`anthropic-` prefix) |
| CAP-02 | `changie-fragments` skill owns one `verify:post` step (`ref.skill`, `onError:"skip"`) gated on federated `workflow.changie_fragments` (boolean, default true) | `when` semantics confirmed to be a bare dotted-key boolean gate (Pattern 1); both `true`-default and `false`-set states confirmed live via `render-hooks verify:post` (Code Examples) |
| CAP-03 | Skill reads a phase's `*-SUMMARY.md`, writes one fragment per user-visible change via `CI=true changie new` using only kinds from the host's `.changie.yaml`, commits with phase conventions, skips (printed reason, never prompts/blocks) when changie unavailable / no `.changie.yaml` / no user-visible change | Real `.changie.yaml` kinds parsing verified bash-only (Code Examples); fragment-file discovery technique sourced from the project's own `check:changie` precedent (Pattern 3); `changie new --help` confirms `-b`/`-k`/`-m`/`--dry-run` flag surface, non-interactive under `CI=true` |
| CAP-04 | codegraph-go installs the capability at project scope from `git+ssh://...#<tag>`; `workflow.changie_fragments` set in `.planning/config.json`; `CONTRIBUTING.md` documents one-time install; `.gsd/` stays gitignored | git+ssh source resolution confirmed by reading `capability-source.cjs` (`git clone --depth 1 -- <url>` then optional `checkout <ref>`); D-11 install-before-config-set ordering confirmed live both directions; **Pitfall 1 (skill materialization) and Pitfall 2 (`.gsd-capabilities.json` gitignore gap) are new, load-bearing findings this requirement's task list must absorb** |
</phase_requirements>

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| `gh` CLI | D-03 PR lookup, D-04 draft PR | ✓ | 2.101.0 `[VERIFIED]` | none needed — required by design; absence is itself a documented skip case |
| `gh auth` | D-03 PR lookup | ✓ (this researcher's account) | — | codegraph-go contributors without `gh auth` hit the documented skip case, not a block |
| `git` (trailers support) | D-07 idempotency, D-08 commit | ✓ | system git, `%(trailers:...)` format confirmed working | none needed |
| `changie` | CAP-03 fragment writing | ✓, but **not on PATH directly** — only reachable via `task changie --` (`GOWORK=off go tool -modfile=go.tool-changie.mod changie`) `[VERIFIED: `changie` bare → "command not found"; `task changie -- --version` → "changie version vdev"]` | pinned per `go.tool-changie.mod` | this is exactly why `workflow.changie_command` exists (D-02) — no further fallback needed |
| gsd-core capability system | CAP-01/02/04 | ✓ | 1.14.0 | none — this phase cannot proceed without it |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** `changie` not on PATH — already solved by `workflow.changie_command` (D-02).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (codegraph-go side) | Go's standard `go test` / Taskfile `check:*` targets — but **this phase touches no Go code** (config/docs/gitignore only) |
| Framework (capability repo side) | Shell (`test/run.sh`, D-09) — the capability repository's commit trail is outside this roadmap, so its RED/GREEN evidence is **pasted into codegraph-go's own phase artifacts** (D-09), following the `01-MUTATION-LOG.md` precedent named in canonical refs |
| Config file | codegraph-go: none new. Capability repo: `test/run.sh` is itself the "config" |
| Quick run command | Capability repo: `bash test/run.sh` (proposed name per D-09; exact name is Claude's Discretion) |
| Full suite command | Same — this is a small, self-contained shell suite, not a tiered suite |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|---------------|
| CAP-01 | `capability install ./ --scope project` succeeds on a scratch project | integration (shell) | `test/run.sh` leg 1 (D-09) | ❌ Wave 0 — capability repo does not exist yet |
| CAP-02 | `render-hooks verify:post --raw` contains the step with key true, omits with key false | integration (shell), both directions observed | `test/run.sh` leg 2 (D-09) | ❌ Wave 0 |
| CAP-03 | Every skip case + the write case produce a named reason / exact fragment+trailer | integration (shell), one real skill run for the judgment half | `test/run.sh` legs 3+ (D-09); model-judgment half proven by one real skill dispatch, transcript recorded in this phase's SUMMARY | ❌ Wave 0 |
| CAP-04 | Install, config-set ordering, `.gitignore` completeness (`git status --porcelain` empty apart from `.planning/config.json`/`.gitignore`), **and skill materialization (Pitfall 1)** | manual/scripted verification inside codegraph-go itself | `git status --porcelain` + `find ~/.claude/skills -iname '*changie*'` (new — see Open Question 1) | N/A — verification commands, not a test file |

### Sampling Rate
- **Per task commit:** `bash test/run.sh` (capability repo); `git status --porcelain` (codegraph-go, after each CAP-04 sub-step)
- **Per wave merge:** full `test/run.sh` + the CAP-04 materialization check
- **Phase gate:** `test/run.sh` green, `git status --porcelain` clean apart from allowed files, and the Pitfall-1 materialization check passing, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `gsd-capability-changie/test/run.sh` — covers CAP-01, CAP-02, CAP-03 (does not exist yet — this phase creates the repo)
- [ ] `gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh` — the deterministic half under test
- [ ] A codegraph-go-side verification command for the Pitfall-1 materialization step (new, not previously scoped in `02-CONTEXT.md`)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | No | No user-facing auth surface in this phase; `gh auth` is an existing, out-of-scope dependency |
| V3 Session Management | No | N/A |
| V4 Access Control | Yes | The capability repository is **private** (D-10); access is GitHub-repo-permission-gated. `gh` auth + SSH-agent access to the private repo is the access-control boundary for who can install it — contributors without access get the documented manual fallback (D-12), not an error |
| V5 Input Validation | Yes | SUMMARY-derived fragment bodies and any manifest-declared `when`/`ref.command`/gate `predicate` values are untrusted-input-to-shell risks. D-05 already mandates passing the body as a single argv element, never through `eval` or shell re-quoting — this is `loop-hook-dispatch.md`'s own documented mitigation pattern for exactly this class of value (`[CITED: ~/.claude/gsd-core/references/loop-hook-dispatch.md]`, read this session) |
| V6 Cryptography | No (indirectly Yes via transport) | No cryptography implemented by this phase; git+ssh transport encryption is inherited, not built |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Shell/command injection via SUMMARY.md body text reaching `changie new -b "<body>"` | Tampering / Elevation of Privilege | Pass the body as a single argv element (never `eval`, never a re-quoted shell string) — D-05, mirrored by `loop-hook-dispatch.md`'s identical requirement for `ref.command`/gate `predicate` values `[VERIFIED pattern, CITED source]` |
| A malicious or compromised commit to the private capability repo rewriting `SKILL.md` to inject adversarial instructions into the agent's context | Tampering / Elevation of Privilege | gsd-core's own install disclosure is explicit that skill bodies are **"installed verbatim and are NOT content-scanned"** `[VERIFIED: real install output, this session]` — the only mitigation is repository access control (private repo, D-10) and pinning a specific tag (`#v0.1.0`, D-01/CAP-04) rather than tracking a branch, so a later malicious commit does not silently propagate to already-installed clones until a deliberate `capability update` |
| A capability declaring executable surfaces (`hooks[]`) without the installer's knowledge | Tampering / Elevation of Privilege | gsd-core's own consent gate: install without `--yes` returns `status:"aborted", requiresConsent:true` with a disclosure and a non-zero exit `[VERIFIED via source read, capability-command-router.cjs:305-321]` — but this phase's actual manifest (no `hooks[]`, only a `steps[]` skill reference) needs **no** `--yes` flag at all, confirmed by a real install succeeding without it |
| Federated config key collision or premature activation before the declaring capability is installed | Tampering (config integrity) | Confirmed fail-closed: `config-set` of an undeclared federated key is rejected with a full valid-key enumeration and exit 1 until the capability is installed (D-11, verified both directions this session) |

## Sources

### Primary (HIGH confidence — read/run this session)
- `~/.claude/gsd-core/bin/lib/capability-validator.cjs` — manifest schema, `validateStep`, `validateFeatureBody`, `validateRuntimeCompat`, semver/engines validation (read in full relevant sections)
- `~/.claude/gsd-core/bin/lib/capability-activation.cjs` — `when`/`pointFrom` resolution semantics (read in full)
- `~/.claude/gsd-core/bin/lib/capability-source.cjs` — `parseSpec`, `resolveLocal`, `resolveGit`, `stageValidated` (git+ssh install mechanics, engines.gsd mismatch error)
- `~/.claude/gsd-core/bin/lib/capability-command-router.cjs` — `capability install`/`set` CLI flag parsing, consent (`--yes`) flow
- `~/.claude/gsd-core/bin/lib/capability-lifecycle.cjs` — `installCapability`, `capabilitiesRoot`, `gsdHome: runtimeDir` binding for project scope
- `~/.claude/gsd-core/bin/lib/capability-ledger.cjs` — `.gsd-capabilities.json` ledger path (project root, not under `.gsd/`)
- `~/.claude/gsd-core/bin/lib/capability-writer.cjs`, `surface.cjs` — materialize/`applySurface` mechanics for `capability set --runtime --scope`
- `~/.claude/gsd-core/bin/lib/capability-registry.cjs` (lines 502-619) — the `claude` runtime descriptor's `artifactLayout.local`/`.global`, `skillsGlobalOnboarding: true` — the central finding
- `~/.claude/gsd-core/references/loop-hook-dispatch.md` — `step`/`gate` dispatch contract, shell-injection warning pattern
- `~/.claude/gsd-core/workflows/ship.md` (`preflight_checks`, `create_pr`, `track_shipping` steps) — confirms no existing-PR reuse path
- Real `gsd-tools` command runs this session: `capability install`, `capability list`, `capability set --runtime claude --scope local/global --config-dir`, `config-set`, `loop render-hooks verify:post`, `init.phase-op`, `skills-root claude`
- `/Volumes/Code/github.com/seanb4t/codegraph-go/.changie.yaml`, `Taskfile.yml` (`check:changie`, `GO_TOOL_CHANGIE`), `.gitignore`, `CONTRIBUTING.md` — read directly this session
- `git log --format='%(trailers:...)'`, `gh --version`, `gh auth status`, `changie new --help` — run directly this session

### Secondary (MEDIUM confidence)
- `02-CONTEXT.md` / `02-DISCUSSION-LOG.md` — user decisions, treated as locked constraints, not independently re-derived

### Tertiary (LOW confidence / Assumptions Log)
- `gh` unauthenticated/no-PR failure-path exact behavior (A1, A2) — well-documented CLI behavior, not exercised live this session
- Global-scope Claude materialization against the real `~/.claude` (A3) — the single most important unverified step; flagged as a required live checkpoint in CAP-04's own task list, not left implicit

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — nothing installed beyond first-party tooling already present and version-confirmed live
- Architecture (manifest/dispatch mechanics): HIGH — every claim in Code Examples/Patterns was reproduced by a real command this session
- Architecture (skill materialization / Pitfall 1): HIGH on the *problem* (reproduced twice, independently), MEDIUM on the *fix* (A3 — the global-scope fix path was reasoned from source + partial live testing, not fully round-tripped against the real `~/.claude`)
- Pitfalls: HIGH — all four are either reproduced errors or read-and-quoted source, not inference
- `/gsd-ship` existing-PR behavior (D-04): HIGH — read the entire relevant workflow file, quoted verbatim, confirmed no reuse path exists anywhere in it

**Research date:** 2026-09-25
**Valid until:** tied to gsd-core version — re-verify if the installed gsd-core version changes from 1.14.0 (the `capability-registry.cjs` artifact-layout shape, in particular, is exactly the kind of thing a minor gsd-core release could change)
