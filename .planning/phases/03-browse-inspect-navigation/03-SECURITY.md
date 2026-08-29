---
phase: 03
slug: browse-inspect-navigation
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
created: 2026-08-29
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all 10 phase plans carry a
`<threat_model>` block. Verification depth is ASVS L1 (grep-level mitigation presence),
which the workflow's short-circuit rule declares sufficient for `threats_open: 0` at L1.
Every negative (zero-count) check below is paired with a positive control per rule `84d1gfpywd`.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| npm registry → `web/pnpm-lock.yaml` → committed `web/build/` → signed binary | Third-party JS enters an artifact committed to git and embedded in the release | Executable JS |
| dependency lifecycle scripts → developer / CI machine | Install-time scripts execute with the invoking user's privileges | Arbitrary code |
| shadcn-svelte registry (network) → committed `.svelte` source | Registry files are committed and shipped, never entering `pnpm-lock.yaml` | Component source |
| browser → Connect RPC on loopback | Unauthenticated, client-steerable `GetNodeDetailRequest.file` / `GetPermalinkRequest.path` — the only client-steerable filesystem paths on the service | Repo-relative paths |
| indexed repository bytes → `SourceBlob.content` → DOM | Occasionally adversarial third-party source is rendered as markup in a browser | File contents |
| `.git/config` remote URL → RPC response body | Repository config, which commonly carries embedded credentials, crosses into a loopback-readable body | Remote URL / userinfo |
| `git` subprocess → server | An external binary's output is parsed into a URL a user will click | Subprocess stdout |
| shared URL → `depth` / `limit` on traversal RPCs | A link authored by someone else controls how much graph traversal the server performs | Integers |
| server error text → browser | Anything a handler puts in a Connect error is readable by any process reaching the loopback port | Error strings |
| MCP client stdin → `internal/mcp` → stdout | JSON-RPC framing across a process boundary; response identity carried only by `id` | JSON-RPC frames |
| frozen transcript → CI merge gate | The transcript is the sole oracle for wire-protocol currency | Golden transcripts |
| lint / vuln gate verdict → merge decision | A gate that cannot fail lets everything through while appearing to guard | CI verdicts |
| displayed identifier → system clipboard | A value from the indexed repository enters a cross-application buffer | Identifiers |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-SC | Tampering | `pnpm add` of test/highlight deps (03-01) | high | mitigate | Blocking human legitimacy checkpoint; `@testing-library/jest-dom` approval recorded in 03-01-SUMMARY key-decisions | closed |
| T-03-10 | Tampering | dependency lifecycle scripts | high | mitigate | `strictDepBuilds: true` (web/pnpm-workspace.yaml:22); `allowBuilds` populated by `pnpm approve-builds`; `task web:deps:strict` present | closed |
| T-03-11 | Tampering | `web/pnpm-lock.yaml` integrity | medium | mitigate | `task web:lockfile` present — asserts lockfileVersion 9.0, integrity-bearing resolutions, zero file:/git/HTTP sources | closed |
| T-03-12 | Denial of Service | CI runtime of the new JS suite | low | accept | Runs inside the existing `test` job; seconds, not minutes | closed (accepted) |
| T-03-01 | Tampering / Info Disclosure | `GetNodeDetailRequest.file` → `resolveSourcePath` | high | mitigate | `internal/uiserver/confinement_test.go` (2 tests) proves the gate at the RPC boundary with paired positive controls | closed |
| T-03-06 | Information Disclosure | Connect error text from the confinement gate | medium | mitigate | `TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath` present | closed |
| T-03-13 | Spoofing | DNS rebinding against `GetNodeDetail` | high | accept | Closed by `originHostGuard` wrapping the whole mux; no new mux entry added | closed (accepted) |
| T-03-14 | Tampering | frozen transcript re-baselining | high | mitigate | **0** commits touched `testdata/wireoracle` across phase 3 (first commit e47d162a..HEAD); path proven non-empty first — 46 tracked files (inoculation per `ad0edgpf94`). `check:transcript-freeze` in Taskfile + ci.yml | closed |
| T-03-15 | Repudiation | a normalizer change that blinds the oracle | high | mitigate | `TestFrozenTranscriptComparisonDetectsContentMutation` (normalize_test.go:294) with a planted `9.9.9-mutation-probe` | closed |
| T-03-16 | Denial of Service | intermittently red required PR leg | medium | mitigate | Root cause captured in 03-03-EVIDENCE.md before any change; subtest never skipped/retried/excluded | closed |
| T-03-02 | Tampering / EoP | `{@html}` of highlighted source | high | mitigate | All unescaped markup routes through `highlightSource`, whose only two branches are `hljs.highlight().value` (v11 escapes by default) and `escapeHtml()` (highlight.ts:149,170,175). `highlightAuto` count **0** | closed — see Note 1 |
| T-03-17 | Tampering / Info Disclosure | URL `file` param reaching `GetNodeDetail` | high | accept | Server-side confinement only; a client-side filter would be a second implementation that can drift | closed (accepted) |
| T-03-18 | Denial of Service | very large source file in the browser | medium | accept | `internal/uiserver/truncate.go` line-then-byte cap bounds every `SourceBlob` (RPC-05) | closed (accepted) |
| T-03-19 | Information Disclosure | Connect error text rendered in the UI | medium | accept | `mapEngineError` default arm scrubs to a fixed path-free message | closed (accepted) |
| T-03-01b | Tampering / Info Disclosure | `GetPermalinkRequest.path` | high | mitigate | `ValidateRepoRelativePath` (6 references) delegates to the existing gate; `TestGetPermalinkPathConfinementAtRPCBoundary` | closed |
| T-03-03 | Information Disclosure | remote-URL userinfo in `url` / `reason` | high | mitigate | `u.Hostname()` deliberately excludes userinfo (permalink.go:179); `TestRemoteGitHubRepo_CredentialsStripped` | closed |
| T-03-20 | Spoofing | remote host matching | high | mitigate | Exact `host != "github.com"` (permalink.go:106); `TestRemoteGitHubRepo_LookalikeHostRejected` | closed |
| T-03-21 | Tampering | URL assembly with URL-significant chars | medium | mitigate | Per-segment `percentEncodeRepoPath`; `TestGetPermalink_PercentEncodesURLSignificantCharacters` + `TestBuildGitHubBlobURL_PercentEncodesOwnerAndRepo` | closed |
| T-03-22 | Tampering | `GetPermalink` network I/O or mutation | high | mitigate | Only `git ls-remote --get-url` (documented **non-network** URL-expansion form) and `git branch -r --contains` (local). Both `context.WithTimeout` bounded with `cmd.Stdin = nil` (permalink.go:69,74,243,261) | closed |
| T-03-23 | Spoofing | DNS rebinding against the new method | high | accept | Inherits `originHostGuard` by construction; no new mux entry | closed (accepted) |
| T-03-24 | Repudiation | unverified link presented as verified | high | mitigate | Three-valued availability; `TestRemotePresenceResponse_KnownValues`, `..._UnrecognizedValueDegradesToUnknown`, `TestCommitOnRemoteTrackingBranch_UnknownIsNotNotObserved` | closed |
| T-03-24b | Information Disclosure | absolute host path in since-deleted-file refusal | high | mitigate | `TestGetPermalinkRefusesSinceDeletedFile` present | closed |
| T-03-07 | Tampering | vendored shadcn-svelte source | high | mitigate | Blocking human checkpoint, commit `205da685` "human-approved"; 36 files read in full; fetch/XHR/WebSocket/require/process./dynamic-import/fs/eval/`new Function` scans all zero, positive-controlled (79 `script` hits across 31/36 files) | closed |
| T-03-SC-b | Tampering | transitively added npm packages | high | mitigate | Same blocking checkpoint; `web:deps:strict` / `web:lockfile` / `web:audit` all present and re-run post-add | closed |
| T-03-25 | Denial of Service | unbounded RPC dispatch from keystrokes | medium | mitigate | `SEARCH_DEBOUNCE_MS = 150` + two-char minimum + abort-on-supersede (search.ts:46,106,114) | closed |
| T-03-26 | Tampering | rendering symbol names / paths | medium | accept | Svelte text interpolation escapes by construction | closed (accepted) |
| T-03-27 | Denial of Service | crafted link with extreme `depth` / `limit` | high | mitigate | Server-side: `MaxLimit = 1000` refuses, `MaxDepth = 50` clamps (validate.go:22,26; traverse.go:345). No client-side second copy | closed |
| T-03-17b | Tampering / Info Disclosure | `file` from neighbour clicks | high | accept | Same server-side gate, re-validated on the way back in | closed (accepted) |
| T-03-28 | Spoofing | shared link reconstructing a different view | medium | mitigate | One parser/one serializer with `describe('browse-url: round-trip')` asserting canonical-order and unknown-param round-trips | closed |
| T-03-02b | Tampering / EoP | click-target decoration of source | high | mitigate | DOM-API-only decoration; `innerHTML\|outerHTML\|insertAdjacentHTML\|createContextualFragment` count **0** across web/src, positive-controlled (8 `document.` hits) | closed |
| T-03-29 | Tampering | server-supplied permalink as clickable target | medium | mitigate | URL assembled server-side with per-segment encoding; rendered as a link, never as markup; unavailable state renders no link | closed |
| T-03-30 | Information Disclosure | identifier → system clipboard | low | accept | User-initiated, local single-user tool, value already on screen | closed (accepted) |
| T-03-31 | Repudiation | unverified permalink presented as working | high | mitigate | Three availability states render distinctly; six `TestGetPermalink_*` no-link/unverified cases. **Observed live during UAT**: the "no recorded commit SHA" state rendered its own distinct message | closed |
| T-03-32 | Tampering | committed `web/build/` diverging from source | high | mitigate | `task web:drift` compares independent source-tree and output-tree digests (10 `WEB_HASH_LIB`/`web_source_files` references) | closed |
| T-03-33 | Repudiation | stale/absent index presented as trustworthy | high | mitigate | Every degraded verdict renders its own banner; `degrade-states.test.ts` asserts texts are **pairwise** different | closed |
| T-03-34 | Denial of Service | status gate polling the server | medium | mitigate | `setInterval\|setTimeout` count **0** in status.ts, positive-controlled (182 lines, 3 exports) | closed |
| T-03-35 | Information Disclosure | server error text in a named in-view state | medium | accept | Server already scrubs its unclassified arm | closed (accepted) |
| T-03-36 | Tampering | a lint gate that cannot fail | high | mitigate | `lint:go` (Taskfile.yml:4300) invoked as `run: task lint:go` (ci.yml:90) using the linter's own exit status, no capture, no pipeline | closed |
| T-03-37 | Tampering | new CI job drifting from single-definition | medium | mitigate | `task lint:go` lives **inside the required `test:` job** — IN-04's fix, since a new job cannot block merge — and `{ci.yml, test}` is in the `inScopeJobs` fixture (taskfile_shape_test.go:153) | closed |
| T-03-SC-d | Tampering | guard blind to the artifact it guards | high | mitigate | `isolatedModfilePaths` + `TestToolModfilesPopulationMatchesDisk` replace the hardcoded slice — closes and inoculates `v4zqxrz6b3` | closed |
| T-03-SC-e | Tampering | large tool tree entering CI unscanned | high | mitigate | `vuln` target now builds **8** binaries across all four modfiles; `go.tool-proto.mod` (5 refs) and `go.tool-golangci.mod` (3 refs) both in scope; desc asserts no allowlist, no exclusion | closed |
| T-03-SC-c | Tampering | the linter's own dependency tree | medium | mitigate | Exact module path `golangci-lint/v2` pinned in its own `go.tool-golangci.mod` with committed `.sum`, `GOWORK=off`, in scope for `vuln` | closed |
| T-03-38 | Repudiation | suppressing findings to reach green | high | mitigate | 4 real `//nolint` directives repo-wide, each individually justified inline (nilerr, gosec/noctx, errorlint, staticcheck-for-GO-2026-5932); no blanket exclusions | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-03-01 | T-03-12 | JS suite runs in the existing `test` job on a provisioned runner; seconds, not minutes | plan 03-01 | 2026-08-28 |
| R-03-02 | T-03-13 | `originHostGuard` already wraps the whole mux; no new mux entry, so no new rebinding surface | plan 03-02 | 2026-08-28 |
| R-03-03 | T-03-17 | Confinement is server-side only by design; a client-side path filter would be a second implementation free to drift from the real one | plan 03-04 | 2026-08-28 |
| R-03-04 | T-03-18 | `truncate.go`'s line-then-byte cap already bounds every `SourceBlob` leaving the server (RPC-05) | plan 03-04 | 2026-08-28 |
| R-03-05 | T-03-19 | `mapEngineError`'s default arm scrubs to a fixed path-free message; classified arms carry only the caller's own request values | plan 03-04 | 2026-08-28 |
| R-03-06 | T-03-23 | A method registered on the same Connect handler inherits `originHostGuard` by construction | plan 03-05 | 2026-08-28 |
| R-03-07 | T-03-26 | Symbol names and paths render via Svelte text interpolation, which escapes by construction | plan 03-06 | 2026-08-28 |
| R-03-08 | T-03-17b | Neighbour targets originate from the server's own `Location` results and are re-validated by the same gate | plan 03-07 | 2026-08-28 |
| R-03-09 | T-03-30 | Clipboard value is one the user is already looking at, in a local single-user tool; affordance is explicit and user-initiated | plan 03-08 | 2026-08-29 |
| R-03-10 | T-03-35 | Server already scrubs its unclassified arm; classified arms carry only caller-supplied values | plan 03-09 | 2026-08-29 |

---

## Notes

**Note 1 — T-03-02 criterion drift (recorded, non-blocking).** 03-04-PLAN.md's acceptance
criterion asserts "exactly one `{@html}` site exists in all of `web/src`". There are now
**two** (`SourcePane.svelte:203` and `:227`), both added when 03-08 extended the pane; a third
grep hit is prose inside a `call-targets.ts` comment. The *security property* the criterion
exists to protect is intact — both sites call `highlightSource(...)`, the single
markup-producing helper, whose only branches escape. The literal count is stale, not the
mitigation. If 03-04's criterion is ever re-run verbatim it will fail; the criterion should be
restated as "every `{@html}` site's expression is `highlightSource(...)`", which is the
property actually wanted.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-29 | 43 | 43 | 0 | /gsd-secure-phase (orchestrator, ASVS L1 inline; auditor short-circuited per threats_open:0 + register_authored_at_plan_time:true + asvs_level:1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-29
