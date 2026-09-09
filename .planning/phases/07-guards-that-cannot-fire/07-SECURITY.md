---
phase: "07"
slug: "guards-that-cannot-fire"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-09"
---

# Phase 07 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| benchmark harness → `CheckRegression` | An untrusted numeric measurement (possibly degenerate: 0, negative, or from a broken probe) crosses into a gate that decides pass/fail for the PERF-02 and INDX-06 budgets | Numeric metrics (files/s, peak RSS bytes); low sensitivity, high integrity requirement |
| RED transcript → mutation log | Text produced by a test run crosses into a committed evidence artifact that later readers treat as proof | Test names, error strings, byte counts; local-run output |
| module import graph → archtest | A developer-editable dependency structure crosses into an assertion that gates the read-side/wire-side separation and, downstream, where Phase 10's helper may live | Package import paths |
| `go/packages` loader → assertion loop | A load that partially or wholly fails to resolve produces an empty or truncated package set that an unguarded loop would read as compliance | Resolved package graph, per-package load errors |
| committed `.goreleaser.yaml` → generated signtest config | A developer-editable release config crosses into a generated copy that a real signing pipeline consumes; a drift here changes what is rehearsed versus what ships | Release configuration YAML |
| script argv → the file it inspects | Caller-supplied paths determine which file the count check runs against; the wrong path makes every assertion true of a file nobody cares about | File paths (config in, config out, cosign key path) |
| generated config → cosign subprocess | A config lacking the injected key silently redirects cosign to the keyless OIDC flow the calling job is denied | `--key=` flag carrying a cosign private-key path |
| `post-release-verify.yml` → the verification workflow's own execution | A developer-editable `if:` expression decides whether release verification runs against a release whose producing run failed | GitHub Actions job conditions |
| workflow YAML → the shape test's parsed view | A parse that yields zero jobs, or a struct field that silently fails to decode, produces an assertion loop with nothing in it | Parsed workflow document |
| in-test constant lists → a claimed property about real workflow files | A test comparing two constants it wrote itself makes a claim about files it never opened | None (the defect is that nothing crosses) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01 | Tampering | `CheckRegression` current-metrics path | high | mitigate | `internal/bench/regression.go`: `current.FilesPerSec <= 0` and `current.PeakRSSBytes <= 0` each return their own named `invalid current: <Field>` error, placed after the baseline checks and before the delta arithmetic. Two table rows in `regression_test.go` exercise both fields | closed |
| T-07-02 | Repudiation | GRD-01 fix that is itself vacuous | high | mitigate | `07-MUTATION-LOG.md` family (a) pastes the RED run: exactly two failing subtests (`degenerate_current_PeakRSSBytes_…` and `degenerate_current_FilesPerSec_…`) against the byte-identical pre-fix `regression.go`, with the no-mutation shape explicitly justified | closed |
| T-07-03 | Information disclosure | mutation-log transcript | low | accept | Transcript contains only test names, error strings, package paths and byte counts from a local run; `mktemp` paths are the only host-specific values. Accepted; see Accepted Risks Log | closed |
| T-07-04 | Tampering | errHint too weak to discriminate | high | mitigate | Both new rows use the multi-word prefix `invalid current: PeakRSSBytes` / `invalid current: FilesPerSec`; the in-test comment records why the bare word `current` is forbidden (pre-fix throughput message contains `current=0.00 files/s`) | closed |
| T-07-05 | Tampering | archtest loader returning zero packages | high | mitigate | `import_direction_test.go`: `len(pkgs) == 0` is fatal; absence of production `internal/query` is separately fatal; loaded count is logged. Strengthened post-review by CR-01: `packages.PrintErrors(pkgs) > 0` is fatal, refusing a partial load (RED transcript in mutation log, family (b) addendum) | closed |
| T-07-06 | Elevation of privilege | `internal/query` acquiring a wire-layer dependency | high | mitigate | Four-entry `forbiddenWireLayerImports` checked over `transitiveDeps` of every loaded query-rooted package including test variants; proven RED by demonstration b1 (`connectrpc.com/connect`) | closed |
| T-07-07 | Elevation of privilege | `internal/query` acquiring the indexer pipeline root | high | mitigate | Exact-path `forbiddenProductionOnlyImports` entry checked over the production compilation unit only (test variants `continue`); proven RED by demonstration b2. Allowed leaves are asserted live in the production dependency set so the exact-path check cannot go vacuous | closed |
| T-07-08 | Tampering | single-hop check masquerading as a transitive one | high | mitigate | `internal/parser` asserted present in `productionDeps` AND absent from `productionQueryPkg.Imports`; a single-hop walk cannot satisfy both | closed |
| T-07-09 | Spoofing | a RED demonstration that fails at compile time rather than through the guard | medium | mitigate | Mutation log b1 records that `internal/uiserver`/`internal/mcp` both import `internal/query` and would fail as an import cycle; the external `connectrpc.com/connect` was chosen so the failure passes through the guard's own assertion (pasted `import_direction_test.go:127` lines) | closed |
| T-07-10 | Tampering | additions-only diff guard passing on an empty diff | high | mitigate | `scripts/inject-cosign-key.sh`: `KEY_LINE_COUNT` is printed and any value other than `1` exits 1; zero (unmatched anchor) is named in the error. Proven RED by family (c) re-indent mutation on a temp copy | closed |
| T-07-11 | Spoofing | the count check running against the wrong file | high | mitigate | Strict `$# -ne 3` arity check with usage on stderr and exit 2; `[ ! -r "$COMMITTED_CONFIG" ]` exits 2 naming the unreadable path. Strengthened post-review by WR-01: a `"` or `\` in the key path is refused with exit 2 before any file is written | closed |
| T-07-12 | Tampering | a duplicated sign-blob block injecting two keys | medium | mitigate | Same exactly-1 count assertion; the error message names "two or more means a duplicated sign-blob block" separately from the zero case | closed |
| T-07-13 | Denial of service | cosign falling through to the denied keyless OIDC flow and hanging | high | mitigate | The script's header comment records the causal chain (missing `--key=` → keyless Fulcio/OIDC → denied → hang); both `Taskfile.yml` call sites invoke the script before `goreleaser` is run, so the refusal precedes any cosign invocation | closed |
| T-07-14 | Tampering | the RED demonstration editing the committed release config | high | mitigate | Family (c) perturbs a `sed`-written copy in a `mktemp -d` directory; `git diff --quiet -- .goreleaser.yaml` recorded clean before and after. Re-verified clean at this audit | closed |
| T-07-15 | Tampering | conclusion guard removed from one job | high | mitigate | `TestPostReleaseJobsDeclareConclusionGuard` compares every job in `doc.Jobs` to the verbatim `postReleaseConclusionGuard` constant; proven RED by demonstration d1 | closed |
| T-07-16 | Tampering | conclusion guard inverted rather than removed | high | mitigate | `job.If != postReleaseConclusionGuard` is exact string equality with no normaliser; proven RED by demonstration d2 | closed |
| T-07-17 | Spoofing | a zero-job parse reading as compliance | high | mitigate | `decodeFullWorkflowDoc` returns an error on zero jobs; the test fatals on an empty map and on `inspected == 0`; the count is logged; `…_EmptyDocIsError` companion pins the helper's property | closed |
| T-07-18 | Repudiation | a guard that passes by comparing constants it wrote itself | high | mitigate | `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` deleted outright (D-09, commit 3dbdcc33); the new test's expected value is compared only against the parsed workflow file | closed |
| T-07-19 | Tampering | a fixed expected-job-id list going stale as jobs are added | medium | mitigate | No job-id list exists; the test iterates whatever `doc.Jobs` declares, so a new unguarded job fails on its own | closed |
| T-07-20 | Tampering | a RED mutation of a release-critical workflow reaching a commit | high | mitigate | Family (d) records `git diff --quiet -- .github/workflows/post-release-verify.yml` clean before each mutation, `git checkout --` revert, and clean after. Re-verified clean at this audit | closed |
| T-07-SC | Tampering | npm/pip/cargo installs (declared in all four plans) | high | mitigate | No package-manager install occurred anywhere in the phase; `git diff --quiet main..HEAD -- go.mod go.sum` is clean. `shellcheck` and `task` are pre-existing developer tooling | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-07-01 | T-07-03 | The committed mutation-log transcripts contain only test names, Go error strings, module import paths, byte counts and `mktemp`-scoped temp paths from local runs. No credentials, environment values, or paths outside the repository or the OS temp root are present. Redaction would remove the evidentiary value the log exists to carry | Phase 07 plan (07-01-PLAN.md threat register) | 2026-09-08 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-09 | 21 | 21 | 0 | /gsd-secure-phase (orchestrator, ASVS L1 grep-depth; auditor short-circuited per plan-time register rule) |

Audit notes (2026-09-09): register origin is plan-time (`<threat_model>` present in all four PLAN files); no SUMMARY carried a `## Threat Flags` section. Verification at HEAD `0d5d188a`: `go test ./internal/bench/ ./internal/query/archtest/ ./internal/upgrade/` green; `shellcheck scripts/inject-cosign-key.sh` clean; `.goreleaser.yaml`, `post-release-verify.yml`, `go.mod`, `go.sum` unchanged. Out of scope for this register: the sibling finding that `internal/graphstore/archtest` ignores per-package load errors is tracked as a pending todo (commit 1879aea1), not a Phase 07 threat.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-09
