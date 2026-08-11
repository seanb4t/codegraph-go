---
phase: 4
slug: codegraph-upgrade-homebrew
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-11
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — `Taskfile.yml` defines every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it) |
| **Quick run command** | `go test ./internal/upgrade/...` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~5s package / ~90s full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/...`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | UPGR-01 | T-04-01, T-04-03 | A path-shape match alone never fires; an unresolvable path falls open to "not detected" rather than blocking the upgrade | unit (tracer, seam-based) | `go test ./internal/upgrade/ -run '^TestUpgradeRun_RefusesBrewManagedCask$' -v \| rg -c -e '--- PASS: TestUpgradeRun_RefusesBrewManagedCask'` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | UPGR-02 | T-04-01, T-04-03 | Seven executing not-detected rows prove a Caskroom/Cellar-shaped path with no Homebrew receipt above it does not fire, and that a dangling symlink or a symlink loop yields not-detected immediately rather than falling through to a shape scan | unit (table-driven) | `test "$(go test ./internal/upgrade/ -run '^TestDetectBrewManaged$' -v 2>&1 \| rg -c -e '--- PASS: TestDetectBrewManaged/')" -ge 16` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | UPGR-03 | T-04-04 | `--check` mutates nothing and reaches no network; the reinstall-anyway option cannot override the refusal | unit (seam-based) | `test "$(go test ./internal/upgrade/ -run '^TestUpgradeRun_(CheckBrewManagedStepsAside\|ForceDoesNotOverrideBrewRefusal)$' -v 2>&1 \| rg -c -e '^--- PASS: TestUpgradeRun_')" -eq 2` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | UPGR-02 | T-04-05, T-04-06 | The install gate can no longer be satisfied by residue from a prior failed install; the freshness snapshot records size as well as mtime, so its size fallback is implementable rather than promised | config (shape + validate) | `test "$(rg -c -e 'codegraph-brew-install' .goreleaser.yaml \|\| echo 0)" -eq 0 && test "$(rg -c -e 'atomic_write' .goreleaser.yaml \|\| echo 0)" -eq 0 && test "$(rg -c -e 'system_command binary, args: \["man", man_dir\]' .goreleaser.yaml)" -eq 1 && test "$(rg -c -e 'fresh_man_pages' .goreleaser.yaml)" -ge 2 && test "$(rg -c -e 'File\.size' .goreleaser.yaml)" -ge 2 && task check:goreleaser` | ✅ | ⬜ pending |
| 04-02-02 | 02 | 1 | UPGR-02 | T-04-07 | Removing a CI gate cannot silently remove an unrelated one; the rehearsal independently re-checks freshness; both evidence echoes agree on one schema; the shape tests are proved to have RUN via named PASS counts, never a package-level `ok` match | config (shape + named-PASS counts) | `test "$(rg -c -e 'SENTINEL' Taskfile.yml \|\| echo 0)" -eq 0 && test "$(rg -c -e 'FRESH_MAN_PAGE_COUNT' Taskfile.yml)" -ge 3 && test "$(rg -c -e 'CASK-REHEARSE-EVIDENCE schema=3' Taskfile.yml)" -eq 2 && test "$(go test ./internal/upgrade/ -run '^TestTaskfile' -v 2>&1 \| rg -c -e '^--- PASS: TestTaskfile')" -ge 5 && test "$(go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -v 2>&1 \| rg -c -e '^--- PASS: TestWorkflowRunBodiesInvokeTask')" -eq 1` | ✅ | ⬜ pending |
| 04-02-03 | 02 | 1 | UPGR-02 | T-04-08 | The evidence record states what was measured, with checkable citations; each folded todo is asserted BY NAME — absent from `pending/`, present exactly once in `completed/` — rather than by a directory total that every unrelated `/gsd-capture` perturbs | docs (grep set-check + per-name membership) | `test "$(rg -c -e 'leave the sentinel behind' .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md \|\| echo 0)" -eq 0 && test "$(rg -c -e '30 orphaned' .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md)" -ge 1 && for t in 2026-08-10-03-evidence-falsely-claims-a-failed-install-can-strand-the-phase-4-sentinel.md 2026-08-10-post-install-man-page-assertion-can-be-satisfied-by-stale-pages.md; do test ! -e ".planning/todos/pending/$t" \|\| { echo "todo not folded — still present in pending/: $t" >&2; exit 1; }; test "$(ls .planning/todos/completed/ \| rg -c -F -e "$t" \|\| echo 0)" -eq 1 \|\| { echo "todo not filed exactly once in completed/: $t" >&2; exit 1; }; done` | ✅ | ⬜ pending |
| 04-03-01 | 03 | 1 | UPGR-01, UPGR-02, UPGR-03 | T-04-09, T-04-10 | The active-milestone scope still parses after the edits; no phase is dropped; the `-ge 4` leg is the vacuity guard that stops the first leg passing by deleting every `Cellar` mention | docs (paired set-check + parser re-proof) | `test "$(rg --no-heading -n -e 'Cellar' .planning/ROADMAP.md \| rg -v -e 'Caskroom' \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg --no-heading -n -e 'Cellar' .planning/ROADMAP.md \| wc -l \| tr -d ' ')" -ge 4 && test "$(rg -n -e '^- \[.\] \*\*Phase 4:' .planning/ROADMAP.md \| rg -c -e 'Caskroom')" -eq 1` | ✅ | ⬜ pending |
| 04-03-02 | 03 | 1 | UPGR-01, UPGR-02 | T-04-11 | The falsified premise is cited so a future reader cannot re-derive it | docs (paired set-check) | `test "$(rg --no-heading -n -e 'Cellar' .planning/REQUIREMENTS.md .planning/PROJECT.md \| rg -v -e 'Caskroom' \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg -c -e '#19121' .planning/REQUIREMENTS.md)" -ge 1` | ✅ | ⬜ pending |
| 04-04-01 | 04 | 2 | UPGR-01, UPGR-03 | T-04-13, T-04-14 | Help text names both exit behaviours and offers no override | unit | `go test ./internal/cli/ -run '^TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes$' -v \| rg -c -e '--- PASS: TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes'` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | UPGR-01, UPGR-03 | T-04-12 | No published instruction contradicts the shipped mutation path, and the `brew trust` instructions the published contract depends on survive the rewrite | docs (paired grep) | `test "$(rg --no-heading -c -e "next phase's work" -e 'not yet shipped' -e 'undefined interaction' README.md docs/RELEASE.md \| wc -l \| tr -d ' ')" -eq 0 && test "$(rg --no-heading -c -e 'brew upgrade codegraph' README.md docs/RELEASE.md \| wc -l \| tr -d ' ')" -eq 2 && test "$(rg -c -e 'brew trust' docs/RELEASE.md)" -ge 2` | ✅ | ⬜ pending |
| 04-05-01 | 05 | 3 | UPGR-01 | T-04-15, T-04-17 | A network-fetched binary is signature-verified before it is made executable; the invocation floor is anchored on `^\s*` and calibrated against the measured baseline of 4 real invocations (10 matching lines) | config (invocation floor + region-scoped ordering assertion) | `test "$(rg -c -e '^\s*cosign verify-blob' Taskfile.yml)" -ge 5 && node -e 'const fs=require("fs");const t=fs.readFileSync("Taskfile.yml","utf8");const b=t.slice(t.indexOf("verify:self-upgrade:"),t.indexOf("verify:gatekeeper:"));const c=b.indexOf("cosign verify-blob"),m=b.indexOf("chmod +x");if(c<0\|\|m<0\|\|c>m)throw new Error("ordering");console.log("ok")'` | ✅ | ⬜ pending |
| 04-05-02 | 05 | 3 | UPGR-01 | T-04-16 | All six identity restatements across five files exhibit selected boundary-case parity with the compiled policy; per-file set membership and a `verify:self-upgrade`-region requirement make the guard RED if this plan's own literal is dropped | unit (shape + vacuity self-test) | `test "$(go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy' -v 2>&1 \| rg -c -e '^--- PASS: TestCosignIdentityPolicy')" -eq 2` | ❌ W0 | ⬜ pending |
| 04-05-03 | 05 | 3 | UPGR-01 | T-04-16, T-04-18 | The drift guard is observed RED three times: a real semantic loosening, an empty match set (total floor), and a total-preserving relocation of Task 1's own literal out of `verify:self-upgrade` (region-scoped check); the named-PASS count is executable rather than prose, so a `-run` pattern matching nothing cannot pass; the folded todo is asserted BY NAME — absent from `pending/`, present exactly once in `completed/` — rather than by a directory total that every unrelated `/gsd-capture` perturbs | unit (RED proof ×3 + named-PASS count + revert + per-name membership) | `go test ./internal/upgrade/... >/dev/null && test "$(go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy' -v 2>&1 \| rg -c -e '^--- PASS: TestCosignIdentityPolicy')" -eq 2 && git diff --exit-code Taskfile.yml internal/upgrade/taskfile_shape_test.go && for t in 2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md; do test ! -e ".planning/todos/pending/$t" \|\| { echo "todo not folded — still present in pending/: $t" >&2; exit 1; }; test "$(ls .planning/todos/completed/ \| rg -c -F -e "$t" \|\| echo 0)" -eq 1 \|\| { echo "todo not filed exactly once in completed/: $t" >&2; exit 1; }; done` | ❌ W0 | ⬜ pending |
| 04-06-01 | 06 | 4 | UPGR-01, UPGR-02, UPGR-03 | T-04-19 | EVERY mutation — `brew tap`, `brew trust`, `brew install`, payload substitution — runs inside ONE script whose EXIT trap was armed before the first of them, so none can outlive an interruption across a task boundary — asserted positively by a `TRAP_ARMED=1` marker the arming step appends to the baseline between registering the trap and `brew tap`, so a run that reached the first mutation unarmed fails rather than resting on a transcript read; detection fires against a genuinely Homebrew-authored tree; restoration is proved POSITIVELY by a receipt carrying exactly ONE well-formed `PAYLOAD_SHA256_BEFORE` line AND exactly ONE well-formed `PAYLOAD_SHA256_AFTER` line whose values are equal (cardinality asserted PER KEY alongside distinctness — `sort -u` also reports one distinct value when the after-hash is absent or malformed, and a single `(BEFORE\|AFTER)` alternation counted to 2 is set membership wearing a count's clothing: two BEFORE lines and no AFTER at all satisfied it, measured against fixtures), `RESTORE_INVOCATIONS=1`, a `TAP_ACTION` agreeing with `TAP_PREEXISTING` and a `TRUST_ACTION` agreeing with `TAP_TRUSTED_BEFORE` — never by an absence check that passes when `brew` is missing, and never by untapping or untrusting unconditionally; the three run artifacts are bound to one another by a shared `RUN_ID` so a stale harness log cannot supply leg 1's PASS | acceptance (single trapped script: harness log + positive restoration receipt + bidirectional baseline match + run-ID binding) | `R="${TMPDIR:-/tmp}/codegraph-04-06-restore-receipt.txt"; B="${TMPDIR:-/tmp}/codegraph-04-06-baseline.env"; H="${TMPDIR:-/tmp}/codegraph-04-06-harness.log"; command -v brew >/dev/null \|\| { echo "brew not on PATH — restoration is UNVERIFIED, not clean" >&2; exit 1; }; for f in "$B" "$H" "$R"; do test -s "$f" \|\| { echo "missing or empty acceptance artifact: $f — the trapped script did not run to completion" >&2; exit 1; }; done; test "$(rg -c -e '^--- PASS: TestDetectBrewManaged_RealInstall \([0-9.]+s\)$' "$H")" -eq 1 && test "$(rg -c -e '^TRAP_ARMED=1$' "$B")" -eq 1 && test "$(rg -c -e '^RESTORE_VERDICT=ok$' "$R")" -eq 1 && test "$(rg -c -e '^RESTORE_INVOCATIONS=1$' "$R")" -eq 1 && test "$(rg -c -e '^PAYLOAD_SHA256_BEFORE=[0-9a-f]{64}$' "$R")" -eq 1 && test "$(rg -c -e '^PAYLOAD_SHA256_AFTER=[0-9a-f]{64}$' "$R")" -eq 1 && test "$(rg -o -e '^PAYLOAD_SHA256_(BEFORE\|AFTER)=[0-9a-f]{64}$' "$R" \| rg -o -e '[0-9a-f]{64}$' \| sort -u \| wc -l \| tr -d ' ')" -eq 1 && test "$(brew list --cask --versions codegraph \| wc -l \| tr -d ' ')" -eq 0 && B="$B" R="$R" H="$H" node -e 'const fs=require("fs");const b=fs.readFileSync(process.env.B,"utf8"),r=fs.readFileSync(process.env.R,"utf8"),h=fs.readFileSync(process.env.H,"utf8");const g=(s,re,what)=>{const m=re.exec(s);if(!m)throw new Error("missing "+what);return m[1]};const id=g(b,/^RUN_ID=(\S+)$/m,"RUN_ID in baseline");if(g(r,/^RUN_ID=(\S+)$/m,"RUN_ID in receipt")!==id)throw new Error("receipt RUN_ID does not match baseline — artifacts come from different runs");if(g(h,/^RUN_ID=(\S+)$/m,"RUN_ID in harness log")!==id)throw new Error("harness log RUN_ID does not match baseline — stale log standing in for this run");const pre=g(b,/^TAP_PREEXISTING=(yes\|no)$/m,"TAP_PREEXISTING in baseline"),act=g(r,/^TAP_ACTION=(untapped\|left-in-place)$/m,"TAP_ACTION in receipt"),wantTap=pre==="yes"?"left-in-place":"untapped";if(act!==wantTap)throw new Error("tap restoration wrong: TAP_PREEXISTING="+pre+" requires TAP_ACTION="+wantTap+", receipt says "+act);const tb=g(b,/^TAP_TRUSTED_BEFORE=(yes\|no)$/m,"TAP_TRUSTED_BEFORE in baseline"),ta=g(r,/^TRUST_ACTION=(untrusted\|left-trusted)$/m,"TRUST_ACTION in receipt"),wantTrust=tb==="yes"?"left-trusted":"untrusted";if(ta!==wantTrust)throw new Error("trust restoration wrong: TAP_TRUSTED_BEFORE="+tb+" requires TRUST_ACTION="+wantTrust+", receipt says "+ta);console.log("run binding OK: "+id+"; tap "+pre+" -> "+act+"; trust "+tb+" -> "+ta)' && test "$(brew tap \| rg -c -e '^seanb4t/tap$' \|\| echo 0)" -eq "$(rg -c -e '^TAP_PREEXISTING=yes$' "$B" \|\| echo 0)" && test "$(brew trust --json v1 \| rg -c -e '^\s*"seanb4t/tap",?$' \|\| echo 0)" -eq "$(rg -c -e '^TAP_TRUSTED_BEFORE=yes$' "$B" \|\| echo 0)" && test "$(ls "$(brew --prefix)/share/man/man1/" \| rg -c -e '^codegraph' \|\| echo 0)" -eq 0` | ❌ W0 | ⬜ pending |
| 04-06-02 | 06 | 4 | UPGR-01, UPGR-02, UPGR-03 | T-04-20, T-04-22 | The unproved leg is named, not claimed; every carried assumption is dispositioned, with the ID↔disposition association enforced per ID rather than by a line count, and each Leg token and each of the four criterion mappings asserted independently rather than by a combined threshold | docs (structure assertion, one independent check per token) | `E=.planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md; test -s "$E" && test "$(rg -c -e 'UPGR-ACCEPTANCE-EVIDENCE' "$E")" -eq 1 && for t in 'Leg 1' 'Leg 2' 'Leg 3'; do test "$(rg -c -F -e "$t" "$E")" -ge 1 \|\| { echo "missing heading token: $t" >&2; exit 1; }; done && for r in UPGR-01 UPGR-02 UPGR-03; do test "$(rg -c -F -e "$r" "$E")" -ge 1 \|\| { echo "missing requirement ID: $r" >&2; exit 1; }; done && for r in UPGR-01 UPGR-02 UPGR-03; do test "$(rg -c -e "^\\\| $r \\\|" "$E")" -eq 1 \|\| { echo "no disposition-table row for $r" >&2; exit 1; }; done && for c in 1 2 3 4; do test "$(rg -c -F -e "Criterion $c" "$E")" -ge 1 \|\| { echo "missing criterion mapping: Criterion $c" >&2; exit 1; }; done` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Table-escaping convention (read before copy-pasting any command above):** a literal `|` inside
a table cell is written `\|`, which is markdown's escape, not shell syntax. Strip the backslashes
before running — `\|\|` is `||` and a single `\|` is a pipe. Every command in the Automated
Command column is otherwise byte-identical to the `<automated>` block of the task it maps to; see
the synchronization note at the end of this document.

**Sampling continuity:** no three consecutive tasks lack an `<automated>` verify — every one of
the 15 tasks above carries one. Tasks 04-06-01 and 04-06-02 additionally carry a `<human-check>`,
which is supplementary to their automated command, not a substitute for it
(`workflow.human_verify_mode` is `end-of-phase`, so no `checkpoint:human-verify` task exists in
this phase).

---

## Wave 0 Requirements

- [ ] `internal/upgrade/brew_test.go` — table-driven detection tests for UPGR-02, including the
      mandatory executing false-positive row (a non-brew binary under a path containing
      `Caskroom`/`Cellar` with no `INSTALL_RECEIPT.json` above it)
- [ ] Constructed-tree fixtures under `t.TempDir()` for `/opt/homebrew`, `/usr/local`, a custom
      prefix, and linuxbrew — **both Caskroom and Cellar shapes at the linuxbrew prefix**
      (D-04R, 2026-08-11)
- [ ] Dangling-symlink and symlink-loop fixture modes, pinning the single `EvalSymlinks` error
      contract (error ⇒ not detected, returned immediately) — added in review cycle 1; the table
      is **16 rows / 7 not-detected**, and both anti-vacuity floors equal those exact numbers
- [ ] Seam assertion in `internal/upgrade/upgrade_test.go` proving `Options.download` and
      `Options.swap` are never invoked under a detected brew install (D-08)

*Existing `go test` infrastructure covers the framework itself — no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The refusal is observed against a **real** `brew tap seanb4t/tap` + `brew install codegraph` from the Phase-3 tap | UPGR-01 | The ROADMAP's Depends-on clause states a synthetic layout can only prove the mechanism, not that it matches what actually ships. Requires a real published cask and a real Homebrew install on the host. | **Three blocking preconditions, all HALT the run.** (1) Review cycle 1, HIGH: HALT if a codegraph cask is already installed — an arbitrary prior cask version is not reliably restorable from a tap that publishes only its current cask, matching the rehearsal's own refusal at `Taskfile.yml:1671`. (2) Review cycle 2, HIGH: HALT if any `codegraph*.1` man page already exists under `$(brew --prefix)/share/man/man1/` — the cask's uninstall hook globs that SHARED directory unconditionally (`.goreleaser.yaml:621`) and would delete pages the run never created. (3) Review cycle 3, LOW: HALT unless the installed Homebrew can both report and revoke tap trust (`brew untrust --help` exits 0 and names `--tap`; `brew trust --json v1` exits 0) — otherwise the run is about to make a persistent trust grant with no confirmed way to revoke it. Then `brew trust seanb4t/tap` (per `khb8v67c48` — the published contract does not work without it), `brew install codegraph`, then `codegraph upgrade` and `codegraph upgrade --check`; record both outputs and exit codes verbatim. **All mutating steps — `brew tap`, `brew trust`, `brew install` and the payload substitution — run inside ONE script whose EXIT trap is armed before the first of them**, with INT/TERM legs that disarm the traps, run the idempotent cleanup once and exit. This claim was FALSE through cycles 1 and 2, which armed the trap in a later task while the tap/trust/install ran in an earlier one; review cycle 3 (HIGH) closed it by merging those two tasks into the single task `04-06-01`, so it is now true by structure rather than by assertion. **The tap is untapped only if the run added it** (review cycle 2, HIGH) and **the trust grant is revoked only if the run made it** (review cycle 3, LOW), both decided from values the baseline probes recorded BEFORE tapping and BEFORE trusting. The trap emits a restore receipt that the automated gate asserts positively, including `TAP_ACTION` and `TRUST_ACTION`. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 15/15 tasks carry one
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — the eight ❌ W0 rows are created by plans 04-01, 04-04, 04-05 and 04-06 themselves, each in the task that first needs them (the count read "six" from plan time through cycle 2; the live table has carried eight since cycle 1 added the two `TestCosignIdentityPolicy` rows, and the merge in cycle 3 did not change it — corrected here rather than left to mislead)
- [x] No watch-mode flags
- [x] Feedback latency < 90s — the slowest automated command is `task test` (~90s); every per-task command is a filtered `go test` or a `rg`/`node` assertion under 10s
- [ ] `nyquist_compliant: true` set in frontmatter — set by validate-phase §6 after execution

**Approval:** plan-time map populated 2026-08-11 by `/gsd-plan-phase 4`; per-task Status column
updated during execution.

**Revision 2026-08-11 — cross-AI review cycle 1 (`/gsd-plan-phase --reviews`).** Task count is
unchanged at **16**; no task was added, removed, or renumbered, so every row above still maps to
the same plan task. Six rows' automated commands changed and are updated in place:

| Row | What changed | Why |
|---|---|---|
| `04-01-02` | `-ge 14` → `-ge 16` | Two new not-detected rows (dangling symlink, symlink loop) pin the reconciled `EvalSymlinks` contract; the floor equals the exact inventory so deleting any row turns it RED |
| `04-02-02` | `\| rg -q 'ok\|PASS'` → two independent count-asserted named-PASS gates | The old leg matched the literal `ok` in `ok … [no tests to run]` and could not distinguish "tests ran and passed" from "no tests ran" — confirmed empirically against this package |
| `04-05-01` | added `rg -c -e '^\s*cosign verify-blob' Taskfile.yml` ≥ 5 | The old file-wide `-ge 3` floor was pre-satisfied (10 matching lines / 4 real invocations today); the anchored floor is RED before the task |
| `04-05-02` | Secure-Behavior text only | "accepts and rejects exactly the same" recalibrated to selected boundary-case parity; the `-run` pattern and count are unchanged because both test names keep the `TestCosignIdentityPolicy` prefix |
| `04-06-02` | all-negative check → positive restore-receipt assertion | The old gate passed when `brew` was absent from PATH and was short-circuitable by an env var, with no positive floor proving restoration happened |
| `04-06-03` | combined `rg -c` threshold → one independent assertion per requirement ID, anchored on the disposition-table row shape | `rg -c` counts lines, so the old threshold was satisfiable by three lines all naming `UPGR-01`, and it did not enforce the ID↔disposition association the prose requires |

**Revision 2026-08-11 — cross-AI review cycle 2 (`/gsd-plan-phase --reviews`).** Task count is
again unchanged at **16**; no task was added, removed, or renumbered. Cycle 2's MEDIUM finding was
that five rows had drifted into being weaker SUBSETS of the task commands they map to — each
dropping at least one leg, and in the `04-03-01` case dropping a deliberately-added vacuity guard,
so the row could pass on a ROADMAP with every `Cellar` mention deleted. **All five are now
synchronized byte-for-byte with their task commands** (modulo the `\|` table escaping documented
under the map), and two further rows were resynchronized because the cycle-2 revision changed the
task command itself:

| Row | What changed | Why |
|---|---|---|
| `04-02-01` | added `atomic_write == 0` and `File.size >= 2` legs | The task command gained both: the `:592` `atomic_write` mention is now explicitly owned by Edit 5, and the freshness snapshot now records size alongside mtime so its promised size fallback is implementable |
| `04-02-02` | added the dropped `CASK-REHEARSE-EVIDENCE schema=3` leg | Cycle 2 MEDIUM — the row was a strict subset; the schema-agreement assertion existed only in the plan |
| `04-03-01` | added the dropped `-ge 4` `Cellar` vacuity guard | Cycle 2 MEDIUM, the sharpest of the five — without it the row passes on a ROADMAP with every `Cellar` mention deleted, which is exactly what the paired positive count exists to prevent |
| `04-04-02` | added the dropped `brew trust` preservation leg | Cycle 2 MEDIUM — the published contract does not work without those instructions, and the row did not check they survived the rewrite |
| `04-05-03` | replaced with the revised task command | Cycle 2 MEDIUM ×2 — the `\| rg -q '^ok'` vacuity shape is gone (it is the cycle-1 HIGH #6 shape, reintroduced), the named-PASS count is promoted from prose into the executable command, and the dropped `todos/pending == 7` leg is restored |
| `04-06-02` | replaced with the revised task command | Cycle 2 HIGH ×2 + MEDIUM — the gate now asserts `RESTORE_INVOCATIONS=1` (idempotent cleanup), agreement between the receipt's `TAP_ACTION` and Task 1's recorded `TAP_PREEXISTING`, the machine's real final tap state, and a zero final man-page count |
| `04-06-03` | added the dropped `Leg 1/2/3` presence loop and the `-F` per-ID presence loop | Cycle 2 MEDIUM — the row was a strict subset of the task command |

**Manual-Only Verification row updated** in the same pass: the acceptance run now carries two
blocking preconditions, not one — no pre-existing cask AND no pre-existing `codegraph*.1` man
page — and the tap is untapped only when the run added it.

**Revision 2026-08-11 — cross-AI review cycle 3 (`/gsd-plan-phase --reviews`, maintainer-authorized
extra cycle).** Task count changes for the first time: **16 → 15**, because cycle 3's HIGH was closed
by MERGING plan `04-06`'s Task 1 and Task 2 into one trapped task. Cycles 1 and 2 armed the
restoration trap at the top of Task 2's script while `brew tap`, `brew trust --tap` and
`brew install` ran in Task 1 — so three mutations to a real workstation preceded every trap, across
a task boundary no shell trap can span, while three artifacts (including this row's predecessor)
asserted the opposite. Narrowing the claims was rejected: the residue is on a real machine, so the
mechanism had to change. Row changes:

| Row | What changed | Why |
|---|---|---|
| `04-06-01` | **Merged** — absorbs the former `04-06-02`. Requirements `UPGR-02` → `UPGR-01, UPGR-02, UPGR-03`; the command now asserts the harness log, the restoration receipt, the tap and trust agreements, and the shared `RUN_ID` binding | Cycle 3 HIGH: the two tasks became one task, so the two rows become one row. Leg 1's PASS is now read out of the run's own harness log rather than re-run, because the merged task ends with the cask uninstalled and `$(brew --prefix)/bin/codegraph` gone — the `RUN_ID` binding across baseline, log and receipt is what makes reading a fixed-path log safe |
| `04-06-02` | **Renumbered** from `04-06-03`; command and Secure Behavior byte-unchanged | The evidence-document task is now Task 2 of a 2-task plan |
| (removed) | the former `04-06-02` row | Its assertions are all present, verbatim, inside the merged `04-06-01` command, plus the new trust and `RUN_ID` legs |

**No numeric floor moved in this cycle in any VALIDATION row** — the merged `04-06-01` command
preserves every comparison from the two rows it replaces (`-eq 1` ×3, `-eq 0` ×1, plus the two
machine re-probe equalities) and adds `-eq 1` for the harness PASS and `-eq` for the trust re-probe.
The one threshold that moved anywhere in cycle 3 is in `04-02-PLAN.md`'s acceptance criteria
(`MAN_BASELINE_MARKER` reference floor 4 → 3, forced by splitting one marker into two); it is not
carried in any row of this table, and it is named explicitly in that plan's
`<review_dispositions>`.

**Revision 2026-08-11 — cross-AI review cycle 5 (`/gsd-plan-phase --reviews`, maintainer-authorized
TARGETED pass).** Task count is unchanged at **15**; no task was added, removed, or renumbered, and
only rows `04-06-01` and `04-06-02` changed. Cycle 4's merge is confirmed correct and was not
disturbed. Four gate-text fixes, each landing in `04-06-PLAN.md` and mirrored here:

| Row | What changed | Why |
|---|---|---|
| `04-06-01` | leg-1 PASS pattern `'^--- PASS: TestDetectBrewManaged_RealInstall$'` → `'^--- PASS: TestDetectBrewManaged_RealInstall \([0-9.]+s\)$'` | **BLOCKING.** `go test -v` always appends ` (N.NNs)`, so cycle 4's trailing `$` matched zero lines and made this row permanently RED — it would have blocked execution outright. Measured against real `go test -v` output on `./internal/upgrade/`: the cycle-4 pattern matches 0, the corrected pattern matches 1. Anchoring on the emitted duration keeps the check exact (`…_RealInstallExtra` still cannot satisfy it) without being unsatisfiable |
| `04-06-01` | added `test "$(rg -c -e '^PAYLOAD_SHA256_(BEFORE\|AFTER)=[0-9a-f]{64}$' "$R")" -eq 2` before the existing `sort -u … -eq 1` leg | Hash equality proved one distinct VALUE, not two agreeing LINES. `sort -u` also reports 1 for a receipt carrying only `_BEFORE`, or one whose `_AFTER` is malformed enough to be dropped by the 64-hex filter — so the gate passed for a receipt that never recorded an after-hash. Deduplication hides absence; the cardinality leg forbids it. The distinctness leg is unchanged |
| `04-06-01` | added `test "$(rg -c -e '^TRAP_ARMED=1$' "$B")" -eq 1` | The trap-armed-before-mutation property — the one this whole convergence exists to establish, and the subject of cycle 3's HIGH — had no automated assertion. The arming step now appends `TRAP_ARMED=1` to the baseline between registering the trap and `brew tap`, so the guard carries a positive assertion that it did its work (repo rule `84d1gfpywd`). The baseline is seven lines rather than six as a result. The marker proves the arming step completed; textual ordering stays with the acceptance criterion and the `<human-check>` |
| `04-06-02` | added the per-criterion mapping loop `for c in 1 2 3 4; …` | The check was claimed in `04-06-PLAN.md`'s cycle-1 disposition table and again in Task 2's acceptance criteria, and implemented in neither — the command asserted the evidence line, the Leg tokens and the requirement IDs and no criterion token at all. Now executed, `-F` so no token is read as a regex, naming the missing criterion on failure |

**No numeric floor or comparison operator was raised or lowered in this cycle.** Fixes 2 and 3 ADD
assertions (`-eq 2`, `-ge 1` ×4), fix 4 ADDs one (`-eq 1`), and fix 1 corrects a pattern; every
pre-existing comparison in both rows is byte-unchanged. The Manual-Only Verification row is
unchanged — its claims were already reconciled by cycle 3 and remain true.

**Revision 2026-08-11 — cross-AI review cycle 6 (`/gsd-plan-phase --reviews`, FINAL pass).** Cycle 5
returned `current_high=0`; the finding trajectory across the convergence is 19 → 11 → 3 → 2. Task
count is unchanged at **15**; no task was added, removed, or renumbered; the phase stays at 6 plans /
4 waves / 15 tasks / 15 rows. Exactly two MEDIUMs, landing in three rows:

| Row | What changed | Why |
|---|---|---|
| `04-06-01` | the cycle-5 cardinality leg `test "$(rg -c -e '^PAYLOAD_SHA256_(BEFORE\|AFTER)=[0-9a-f]{64}$' "$R")" -eq 2` is SPLIT into two per-key legs, `…BEFORE… -eq 1` and `…AFTER… -eq 1`; the `sort -u … -eq 1` distinctness leg is untouched | **MEDIUM.** An alternation counted is set MEMBERSHIP wearing a count's clothing: the leg counts lines matching EITHER key, so a receipt carrying two `PAYLOAD_SHA256_BEFORE` lines and no `_AFTER` line at all yields cardinality 2 and distinctness 1 — **both legs passed while the after-hash was never recorded**, which is exactly the guarantee the gate exists to provide, and it is reachable through the trap double-invocation defect `04-06-PLAN.md` documents. Third recurrence of one family (cycle 1's multi-`-e` `-ge N`, cycle 4's `sort -u`-hides-absence, now this). **Measured RED/GREEN against four fixtures on 2026-08-11:** (a) two BEFORE / no AFTER — old GREEN, new RED; (b) one BEFORE / 63-hex malformed AFTER — old RED, new RED; (c) one BEFORE / one equal AFTER — old GREEN, new GREEN; (d) one BEFORE / one differing AFTER — old RED, new RED. The cycle-5 row above records what cycle 5 wrote and is retained as history, not as the current gate |
| `04-02-03` | `test "$(ls .planning/todos/pending/ \| wc -l \| tr -d ' ')" -eq 8` → a per-name loop asserting each of the two folded todos is ABSENT from `pending/` and PRESENT exactly once in `completed/`, naming the offending file on failure. (This row had also DROPPED the total leg entirely — a cycle-2-style subset drift — so the mirror both restores the check and corrects its quantity) | **MEDIUM.** The totals were NOT broken: wave ordering makes 10 → 8 → 7 correct, and a report that these gates were guaranteed-RED compared a post-state gate against the current pre-state and was withdrawn. The defect is the wrong QUANTITY — a directory total couples both gates to every unrelated `/gsd-capture` between planning and execution, and commit `ffad9ee` on this branch is exactly that event, so the gate flips RED for reasons unrelated to the task's work. Totals are demoted to a recorded SUMMARY observation. This is the cycle-1 `rg -c` set-membership lesson applied to a directory listing |
| `04-05-03` | `test "$(ls .planning/todos/pending/ \| wc -l \| tr -d ' ')" -eq 7` → the same per-name loop for the one todo this plan folds | Same finding, same fix; the substring `rg -c -e 'no-signature-check'` form is replaced by a full-filename `-F` check in the same pass |

**No numeric floor or comparison operator was raised or lowered in this cycle.** The `04-06-01` fix
turns one `-eq 2` over a two-key alternation into two `-eq 1`s over one key each — the same
strictness expressed per key, and strictly stronger against the input that defeated it. The
`04-02-03` / `04-05-03` fix replaces a count with named set membership, which is not a threshold at
all. Every do-not-regress floor is byte-unchanged: the shell-gate cosign floor `-ge 5`, the in-test
identity floor of 7 literals across five files (asserted inside `TestCosignIdentityPolicy`, not as a
shell `-ge`), named-PASS 5/1/5/2, detection-table `-ge 16`, `File.size` `-ge 2`, `fresh_man_pages` `-ge 2`,
`FRESH_MAN_PAGE_COUNT` `-ge 3`, `schema=3` `-eq 2`, the marker reference floors `-ge 3` ×2, and
`04-06-01`'s `TRAP_ARMED=1`, `RESTORE_VERDICT=ok`, `RESTORE_INVOCATIONS=1`, leg-1 PASS and man-page
`-eq 0` legs. The Manual-Only Verification row is unchanged.
