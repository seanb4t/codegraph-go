# Phase 7: Guards That Cannot Fire - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-08
**Phase:** 7-Guards That Cannot Fire
**Areas discussed:** Archtest boundary (GRD-02), GRD-03 proof path, Workflow test shape (GRD-04/05), GRD-01 refusal semantics

---

## Archtest boundary (GRD-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Forbid indexer root, allow leaves | Forbid `internal/indexer`; allow `indexer/goextract`, `indexer/nodeid`. Green today; enforces HLT-05's persisted-never-re-walked ruling structurally | ✓ |
| Wire-layer only | uiserver/connect/mcp/uiproto only; leaves query→indexer open | |
| Forbid all internal/indexer/... | RED today (expand.go uses goextract constants); needs a refactor outside scope | |

| Option | Description | Selected |
|--------|-------------|----------|
| Union of both lists, transitive | uiserver, mcp, uiproto, connect over resolved deps (NeedDeps) | ✓ |
| Union of both lists, direct only | Same set, import statements only | |
| Roadmap's three only, transitive | Drops uiproto | |

| Option | Description | Selected |
|--------|-------------|----------|
| Assert graphstore + goextract present | Both must appear in the resolved set; report package count | ✓ |
| Assert google.golang.org/protobuf present | Indirect external module via schema; brittle to a schema refactor | |
| Package-count floor only | Proves the loader ran, not that query's graph was loaded | |

| Option | Description | Selected |
|--------|-------------|----------|
| internal/query/archtest/ | Sibling package beside the code it constrains, like graphstore/archtest | ✓ |
| internal/graphstore/archtest/ alongside | One module-wide archtest package; doc says D-04a only | |
| New internal/archtest/ | Would mean moving the existing test too | |

**User's choice:** all four recommended options.
**Notes:** None.

---

## GRD-03 proof path

User first asked what is being tested and why "code we don't own" is under test. Clarified: everything under test is the repo's own Taskfile shell body (awk injection + additions-only diff guard); awk, goreleaser and cosign are not under test.

| Option | Description | Selected |
|--------|-------------|----------|
| Extract to a script, Task calls it | inject + diff guard + positive assertion in one script under scripts/; both Task targets call it; RED demo runs on any host | ✓ |
| Keep inline in both Task bodies | Add assertion in place; RED demo needs darwin + zig/syft/cosign | |

| Option | Description | Selected |
|--------|-------------|----------|
| Exactly one --key= line added | Count reported; 0 = anchor miss, 2+ = duplicated block | ✓ |
| At least one --key= line added | The todo's wording; doubled injection passes | |

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, in taskfile_shape_test.go | Shape test asserting both targets call the script | |
| No, script plus RED demo is enough | Script assertion + mutation log is the guard | ✓ |

**User's choice:** extract to script; exactly one; no shape test.
**Notes:** User: "why are you proposing tests for tests?" — recorded as a standing preference.

---

## Workflow test shape (GRD-04/05)

| Option | Description | Selected |
|--------|-------------|----------|
| Verbatim string, every job in file | Exact disjunct on each job's if:, count reported, empty-doc-is-error twin | ✓ |
| Normalized expression, every job in file | Strip `${{ }}`/whitespace before comparing | |
| Verbatim string, fixed expected job-id list | Set-equality against a hardcoded five ids | |

User then asked what the tap-secret test is and why it matters. Clarified: D-16 two-distinct-GitHub-Apps property from v0.5.0; the existing test compares two in-file constants and reads no workflow. User: "Why the hell are we testing this? Why do we care?"

| Option | Description | Selected |
|--------|-------------|----------|
| Fix it, files only | Read release.yml + release-please.yml, non-empty, disjoint | |
| Delete the test, drop GRD-05 | Remove the tautological test; de-scope GRD-05; mutation log becomes four demos | ✓ |

**User's choice:** verbatim per job; delete the tap test and drop GRD-05.
**Notes:** Scope change applied to REQUIREMENTS.md, ROADMAP.md, PROJECT.md in this session. The 2026-08-10 todo is resolved by the deletion.

---

## GRD-01 refusal semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Right after the baseline checks, one error per field | Mirrors the two existing baseline lines; each error names its field | ✓ |
| Before the platform checks | Numbers first, attribution second; diverges from existing order | |

**User's choice:** after the baseline checks, one error per field.
**Notes:** None.

---

## Claude's Discretion

- Script name/args under scripts/ (D-05)
- `Tests: true` on the GRD-02 loader
- GRD-04 as an extension of the existing test or a sibling
- Error wording, test names, commit granularity

## Deferred Ideas

- GRD-05 file-reading rewrite → REQUIREMENTS.md v2
- brew-trust todo → Phase 12 (already routed by ROADMAP)
