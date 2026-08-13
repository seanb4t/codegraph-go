# Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 7-`codegraph install` Skill + Hooks Distribution (Claude Code)
**Areas discussed:** Location scope default, Version observability, Hand-edited content on reinstall, Upgrade auto-refresh

---

## Location scope default

| Option | Description | Selected |
|--------|-------------|----------|
| Always local, ignore --location for this artifact | Matches Phase 6's dogfood location; avoids surfacing the skill in unindexed repos | |
| Follow --location like MCP config does | Consistent with existing flag semantics (defaults global) | ✓ |
| Install to both scopes unconditionally | Maximizes discoverability, doubles the merge surface | |

**User's choice:** Follow --location like MCP config does
**Notes:** Chose consistency with existing flag semantics over the recommended local-only option.

### Follow-up: global-trigger reliance

| Option | Description | Selected |
|--------|-------------|----------|
| Acceptable as-is | Trust Phase 6's SessionStart nudge + SKILL.md decision table | ✓ |
| Add an explicit index-existence check to SKILL.md's decision table | Belt-and-suspenders for the global-install case | |

**User's choice:** Acceptable as-is

---

## Version observability

| Option | Description | Selected |
|--------|-------------|----------|
| Sidecar manifest file | Matches Anthropic's own plugin.json precedent; keeps SKILL.md frontmatter portable | ✓ |
| Version field in SKILL.md frontmatter | Simpler but nonstandard, and doesn't cover hooks.json | |

**User's choice:** Sidecar manifest file
**Notes:** Informed by web research on Claude Code's plugin.json versioning convention.

### Follow-up: manifest contents

| Option | Description | Selected |
|--------|-------------|----------|
| Version + per-file hashes | One mechanism serves both AGENT-03 and hand-edit drift detection | ✓ |
| Version only | Simpler, but hand-edit detection would need a second mechanism | |

**User's choice:** Version + per-file hashes

---

## Hand-edited content on reinstall

| Option | Description | Selected |
|--------|-------------|----------|
| Silently overwrite | Matches CLAUDE.md marker-fence posture; keeps install non-interactive | ✓ |
| Warn to stderr, then overwrite | Signal without blocking | |
| Refuse and require --force | Fail-closed, stricter than any existing AgentTarget behavior | |

**User's choice:** Silently overwrite

---

## Upgrade auto-refresh

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-refresh via manifest | Upgrade reads the manifest and re-invokes Install() for previously-configured targets | ✓ |
| Manual re-install required | Simpler, but leaves installs silently stale after a binary upgrade | |

**User's choice:** Auto-refresh via manifest

### Follow-up: refresh-failure isolation

| Option | Description | Selected |
|--------|-------------|----------|
| Binary swap succeeds, refresh warning surfaced separately | Binary swap is independently load-bearing and already verified | ✓ |
| Whole upgrade reports failure | Simpler mental model, but conflates two independently-atomic operations | |

**User's choice:** Binary swap succeeds, refresh warning surfaced separately

---

## Claude's Discretion

- Exact hooks.json entry identity/merge strategy for uninstall precision (follow `writeMcpEntry`/`removeMcpEntry` precedent)
- Manifest's exact filename and JSON schema beyond version + per-file hashes
- Whether to add a `codegraph install --status`-style reporting surface
- How upgrade's auto-refresh locates "the manifest(s) it finds"

## Deferred Ideas

None — discussion stayed within phase scope.

**Todos reviewed but not folded** (all judged false-positive keyword matches from `todo.match-phase`, no semantic overlap with skill+hooks distribution):
- `2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md`
- `2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md`
- `2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md`
- `2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md`
- `2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md`
