# Phase 6: Agent Integrations & CLI Lifecycle - Pattern Map

**Mapped:** 2026-07-12
**Files analyzed:** 21 (5 new CLI commands, 12 `internal/agents` files, 4 `internal/version`/`internal/upgrade` files)
**Analogs found:** 15 / 21 (6 new-with-no-analog, all flagged in RESEARCH.md's Recommended Project Structure)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/cli/install.go` | command | request-response (delegates to registry loop) | `internal/cli/serve.go` | role-match |
| `internal/cli/uninstall.go` | command | request-response | `internal/cli/uninit.go` | exact |
| `internal/cli/upgrade.go` | command | request-response (network download) | `internal/cli/sync.go` | role-match |
| `internal/cli/version.go` | command | request-response (static/computed) | `internal/cli/init.go` (thin flag+print shape) | role-match |
| `internal/cli/telemetry.go` | command | request-response (static print) | `internal/cli/uninit.go` (no-op branch: "does not exist — nothing to do") | partial |
| `internal/cli/root.go` (MODIFIED) | route/registration | n/a | itself (`internal/cli/root.go`) | exact |
| `cmd/codegraph/main.go` (MODIFIED — ldflags seam) | config | n/a | itself | exact |
| `internal/agents/types.go` | model | n/a (interface + value types) | `internal/indexer/languages.go` (`LanguageSpec` struct + hook interfaces) | exact |
| `internal/agents/registry.go` | service (registry) | CRUD (register/lookup/list) | `internal/indexer/languages.go` (`registry`/`extToLang` map + `registerLanguage`/`lookupLanguageByID`/`RegisteredLanguageIDs`) | exact |
| `internal/agents/claude.go` | service (per-variant impl) | file-I/O (read-modify-write JSON/MD) | `internal/indexer/languages_go.go` (per-language `init()` registration + descriptor impl) | role-match |
| `internal/agents/cursor.go` | service (per-variant impl) | file-I/O | `internal/indexer/languages_go.go` | role-match |
| `internal/agents/codex.go` | service (per-variant impl) | file-I/O (TOML splice) | `internal/indexer/languages_go.go` (registration shape only — TOML splice body is new-with-no-analog) | partial |
| `internal/agents/opencode.go` | service (per-variant impl) | file-I/O (JSONC via hujson) | `internal/indexer/languages_go.go` (registration shape only — hujson body is new-with-no-analog) | partial |
| `internal/agents/gemini.go` | service (per-variant impl) | file-I/O | `internal/indexer/languages_go.go` | role-match |
| `internal/agents/antigravity.go` | service (per-variant impl) | file-I/O | `internal/indexer/languages_go.go` | role-match |
| `internal/agents/hermes.go` | service (per-variant impl) | file-I/O (YAML line-range surgery) | `internal/indexer/languages_go.go` (registration shape only — YAML surgery is new-with-no-analog) | partial |
| `internal/agents/kiro.go` | service (per-variant impl) | file-I/O | `internal/indexer/languages_go.go` | role-match |
| `internal/agents/shared.go` | utility | file-I/O (surgical JSON/marker read-modify-write) | `internal/cli/uninit.go` (`confirm()` — I/O-idiom analog only; the JSON/marker logic itself has no in-repo analog) | partial |
| `internal/agents/instructions.go` | config | n/a (static template text) | none | no-analog |
| `internal/agents/toml.go` | utility | file-I/O (text splice) | none | no-analog |
| `internal/version/version.go` | config | n/a (ldflags-injected vars) | none | no-analog (greenfield per D-09/RESEARCH) |
| `internal/upgrade/release.go` | service | request-response (HTTP + redirect-trick) | none | no-analog |
| `internal/upgrade/verify.go` | service | transform (crypto verification) | none | no-analog |
| `internal/upgrade/swap.go` | utility | file-I/O (atomic rename) | `internal/cli/init.go` (`writeGitignoreHint` — same "small scoped os.WriteFile/os.Rename under a known-safe path" idiom, but the atomic-temp-then-rename dance itself is new) | partial |
| `internal/upgrade/upgrade.go` | service (orchestrator) | request-response | `internal/cli/serve.go` (multi-step orchestration: resolve → reconcile → branch → dispatch, same "thin orchestrator wiring several sub-steps" shape) | partial |

## Pattern Assignments

### `internal/cli/install.go` / `internal/cli/uninstall.go` (command, request-response)

**Analog:** `internal/cli/uninit.go` (confirmation idiom) + `internal/cli/serve.go` (flag-resolve → delegate shape)

**Imports pattern** (`internal/cli/uninit.go` lines 1-11):
```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)
```
For `install.go`/`uninstall.go`, add `"github.com/seanb4t/codegraph-go/internal/agents"` and drop `bufio`/`strings` (those move into `agents`' own multi-select helper if D-03's interactive picker is implemented in `internal/agents` rather than `internal/cli`).

**Command skeleton pattern** (`internal/cli/uninit.go` lines 19-62 — flag declaration, `RunE` shape, confirm-then-act, clean no-op message):
```go
func newUninitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninit [path]",
		Short: "Remove .codegraph/",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			// ... resolve, act, print via cmd.OutOrStdout()
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "remove without prompting for confirmation")
	return cmd
}
```
`install.go`/`uninstall.go` follow this exact shape: `--target`/`--location` flags declared the same way `force` is; `RunE` resolves `os.Executable()` (D-04) once, builds the `[]agents.AgentTarget` slice via `agents.ResolveTargetFlag(target)`, then loops calling `.Install(loc, opts)` / `.Uninstall(loc)` per target and prints a per-agent status line to `cmd.OutOrStdout()` (mirrors D-08's `removed`/`not-configured`/`unsupported` reporting).

**Interactive-prompt idiom to extend for D-03's multi-select** (`internal/cli/uninit.go` lines 64-78):
```go
func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", prompt)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}
```
Key idiom to reuse: read via `cmd.InOrStdin()` / write via `cmd.OutOrStdout()` (never raw `os.Stdin`/`os.Stdout` — this is what makes the command testable with `cmd.SetIn`/`cmd.SetOut`), and **EOF/empty input degrades to the safe default** rather than erroring. The multi-select picker (D-03) must apply the same "no TTY → fall through to `auto`, never block" rule — detect TTY via the same pattern this file would use if it checked (it currently doesn't check TTY at all since `uninit --force` skips the prompt; `install`'s no-`--target`-and-no-TTY path needs an explicit `term.IsTerminal(int(os.Stdin.Fd()))`-style check, which has no existing in-repo precedent — flag as a small new primitive, not a full new pattern).

**Multi-step orchestration + error propagation** (`internal/cli/serve.go` lines 46-58, resolve-then-branch shape `install`/`uninstall` should mirror):
```go
start, err := resolveStartPath(path)
if err != nil {
	return err
}

repoPath := start
hasIndex := false
if dir, err := query.ResolveCodegraphDir(start); err == nil {
	hasIndex = true
	repoPath = dir
} else if !errors.Is(err, query.ErrNotInitialized) {
	return err
}
```
`install`/`uninstall` don't touch `.codegraph/` (out of scope per CONTEXT.md), but the "resolve → probe → branch on non-fatal probe error vs propagate fatal error" idiom is the template for `agents.ResolveTargetFlag`'s `auto` detection loop.

**`root.go` registration point** (lines 43-47) — add the five new commands to the existing `AddCommand(...)` call:
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd())
```
Append `newInstallCmd(), newUninstallCmd(), newUpgradeCmd(), newVersionCmd(), newTelemetryCmd()`. `SilenceUsage`/`SilenceErrors: true` are set once on `root` (line 40-41) and inherited — do not re-set on the new subcommands.

---

### `internal/agents/registry.go` (service/registry, CRUD)

**Analog:** `internal/indexer/languages.go` — this is the strongest structural match in the whole codebase: a map-keyed-by-stable-ID registry of interface implementations, populated by each variant's own `init()`, looked up by ID or a secondary key, with a sorted "list all registered" export.

**Registry pattern** (`internal/indexer/languages.go` lines 62-94):
```go
var (
	registry  = map[string]LanguageSpec{}
	extToLang = map[string]string{}
)

func registerLanguage(spec LanguageSpec) {
	registry[spec.ID] = spec
	for _, ext := range spec.Extensions {
		extToLang[ext] = spec.ID
	}
}

func lookupLanguageByID(id string) (LanguageSpec, bool) {
	spec, ok := registry[id]
	return spec, ok
}
```
Port directly: `var registry = map[TargetID]AgentTarget{}`, `func registerTarget(t AgentTarget) { registry[t.ID()] = t }`, `func GetTarget(id TargetID) (AgentTarget, bool)`.

**Sorted list-all export** (`internal/indexer/languages.go` lines 96-110):
```go
func RegisteredLanguageIDs() []string {
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
```
Port as `AllTargetIDs() []TargetID` (sorted, deterministic iteration for `--target all` and the interactive multi-select's display order — map iteration order is non-deterministic in Go, so this sort is load-bearing, not cosmetic).

**Per-variant `init()` self-registration** (`internal/indexer/languages_go.go` lines 19-47):
```go
func init() {
	registerLanguage(LanguageSpec{
		ID:         "go",
		Extensions: []string{".go"},
		NewParser: func() (parser.Parser, error) { return cgo.NewGoParser() },
		Extract:    goextract.Extract,
		ModuleKey:  func(descriptor ProjectDescriptor, relPath string) string { ... },
		Descriptor: func(root string) (ProjectDescriptor, error) { ... },
	})
}
```
Each of `claude.go`/`cursor.go`/`codex.go`/.../`kiro.go` follows this exact shape: a package-level `init()` calling `registerTarget(claudeTarget{})` (or a constructed value), one file per variant, zero cross-file coupling except through the shared `AgentTarget` interface and `internal/agents/shared.go` helpers — mirrors how `languages_go.go` only depends on `languages.go`'s interface plus `goextract`/`parser`, never on a sibling `languages_java.go`.

---

### `internal/agents/types.go` (model)

**Analog:** `internal/indexer/languages.go` lines 10-60 — the `ProjectDescriptor` interface + `LanguageSpec` struct-of-hooks shape is the direct template for `AgentTarget` + its supporting value types (`Location`, `TargetID`, `DetectionResult`, `WriteResult`, `InstallOptions` — already fully specified in RESEARCH.md's Pattern 1 code example, reproduced verbatim there). Doc-comment convention to match: every exported type/field gets a one-sentence purpose comment referencing the decision ID that motivated it (see `LanguageSpec.Descriptor` lines 56-59 referencing D-03) — `AgentTarget`'s methods should likewise reference D-02/D-03/D-07/D-08 inline.

---

### `internal/cli/version.go` (command, request-response)

**Analog:** `internal/cli/init.go` — thin flag-parse-then-print shape (`printSummary`, lines 97-110) is the closest in-repo precedent for a command whose whole job is formatting and printing a small struct to `cmd.OutOrStdout()`:
```go
func printSummary(cmd *cobra.Command, stats indexer.Stats, quiet, verbose bool) {
	if quiet {
		return
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "files=%d nodes=%d edges=%d duration=%s\n", ...)
	if verbose {
		fmt.Fprintf(out, "unresolved=%d skipped=%d\n", ...)
	}
}
```
`version.go`'s `RunE` mirrors this: read `version.Version/Commit/Date/GoVersion/OS/Arch` from the new `internal/version` package, print a single formatted line by default, and — mirroring the `--verbose` conditional-extra-output branch above — a `--json` flag switches to `json.NewEncoder(cmd.OutOrStdout()).Encode(...)` instead of the formatted line.

**`main.go` ldflags injection seam** — `cmd/codegraph/main.go` currently has zero version wiring (confirmed: 18 lines, no imports beyond `fmt`/`os`/`internal/cli`). This is the greenfield injection target D-09 names: `internal/version/version.go` declares `var (Version = "dev"; Commit = "unknown"; Date = "unknown")` and the build (`goreleaser`/`go build -ldflags`) sets them via `-X github.com/seanb4t/codegraph-go/internal/version.Version=...`. `main.go` itself needs no changes for this — only `internal/cli/version.go` imports `internal/version`.

---

### `internal/cli/telemetry.go` (command, request-response)

**Analog:** `internal/cli/uninit.go` line 34's no-op branch (`fmt.Fprintf(cmd.OutOrStdout(), "%s does not exist — nothing to do\n", codegraphDir); return nil`) — the closest existing precedent for "print a fixed informational message via `cmd.OutOrStdout()` and return nil, no side effects." `telemetry.go`'s entire `RunE` body is one `fmt.Fprintln(cmd.OutOrStdout(), telemetryStatement)` call where `telemetryStatement` is a package-level const string (D-15's exact wording lives here, not inline in the command).

---

### `internal/agents/shared.go` — JSON/marker surgical write helpers (utility, file-I/O)

**No direct in-repo analog** — this project has no prior "read-modify-write an external structured file while preserving unrelated content" code (the existing `.codegraph/` writes are pure Pebble/JSON-graph-store writes, not third-party-file edits). Use RESEARCH.md's Pattern 2/3 code examples verbatim as the implementation source (already fully specified — `writeMcpEntry`, `replaceOrAppendMarkedSection`) — they are direct mechanical ports of the TS parity-oracle's `shared.ts`, cited and reproduced in full in `06-RESEARCH.md` lines 319-509. The one in-repo idiom to carry over: `internal/cli/init.go`'s `writeGitignoreHint` (lines 88-95) demonstrates this project's convention for a small scoped `os.WriteFile` under a known-safe path with an explicit doc comment justifying why it's safe (never touching a file the tool doesn't own) — apply the same "explicit safety justification in the doc comment" discipline to every `shared.go` write helper, since these functions write to arbitrary external tool configs, a materially higher-risk surface than `.codegraph/`'s self-contained directory.

---

### `internal/upgrade/*` (service, request-response / file-I/O)

**No in-repo analog for the download/verify/swap mechanics** — this is the first network-touching, first cryptographic-verification code in the project. Structural precedent to reuse:
- **Orchestration shape** — `internal/cli/serve.go`'s `RunE` (resolve → branch on non-fatal condition → dispatch to a sub-package → propagate errors) is the template for `internal/upgrade/upgrade.go`'s `Run(currentVersion string, opts Options) error`: resolve latest → `--check`? report+return : download → verify (fatal on failure, per RESEARCH.md Pitfall 7 — never swap on an unverified bundle) → swap.
- **Atomic-write-under-known-safe-path idiom** — `internal/cli/init.go`'s `writeGitignoreHint` is the nearest existing "write a file, handle the error, nothing fancier" precedent; `internal/upgrade/swap.go`'s temp-in-same-dir + `os.Rename` dance is new territory (RESEARCH.md's D-13 fully specifies the mechanics; no `minio/selfupdate` dependency per the Package Legitimacy Audit — reference its `apply.go` only as a pattern source, not a dependency, for the Windows rename-aside branch).
- **Verification wiring** — use RESEARCH.md's `sigstore-go` code example verbatim (lines 510-533+, `verifyRelease` using `bundle.LoadJSONFromPath` / `verify.NewVerifier` / `verify.NewPolicy`) — no in-repo analog exists or should be invented here; this is exactly the "don't hand-roll crypto" case RESEARCH.md's Don't-Hand-Roll table calls out.

## Shared Patterns

### Registry-keyed-by-ID (cross-cutting: `internal/agents/registry.go`)
**Source:** `internal/indexer/languages.go` lines 62-110
**Apply to:** `internal/agents/registry.go`, and every one of the 8 per-agent files' `init()` registration
Map-of-stable-string-ID + per-variant self-registering `init()` + sorted list-all export. This is the single most load-bearing pattern in the phase — the whole `AgentTarget` fan-out mirrors the language-extractor fan-out mechanically.

### Cobra thin-command delegation (cross-cutting: all 5 new `internal/cli/*.go` files)
**Source:** `internal/cli/serve.go`, `internal/cli/uninit.go`, `internal/cli/init.go`
**Apply to:** `install.go`, `uninstall.go`, `upgrade.go`, `version.go`, `telemetry.go`
Every command: resolve flags → resolve paths (`targetRoot`/`resolveStartPath`-style helpers, or `os.Executable()` for D-04) → delegate to an `internal/<pkg>` function → print via `cmd.OutOrStdout()`/`cmd.ErrOrStderr()` (never raw `os.Stdout`/`fmt.Println`) → return the error untouched (no ad-hoc `os.Exit` inside `RunE`; `main.go` owns the exit code).

### Interactive-prompt-with-non-interactive-fallback (cross-cutting: `install.go`'s D-03 multi-select)
**Source:** `internal/cli/uninit.go` lines 64-78 (`confirm()`)
**Apply to:** the new multi-select agent picker
Read via `cmd.InOrStdin()`, write via `cmd.OutOrStdout()` (testability), and treat any unreadable/empty input as the safe default rather than erroring — extend this to "no TTY detected → skip the prompt entirely and use `auto`" per D-03.

### Doc-comment convention citing decision IDs
**Source:** every file header/type-comment sampled above (`internal/cli/root.go` lines 1-8, `internal/indexer/languages.go` lines 10-60, `internal/indexer/languages_go.go` lines 9-21)
**Apply to:** all new files in this phase
Package/type/function doc comments cite the specific `D-NN`/`AGNT-NN`/`CLI-NN` decision ID that motivates the shape, not just a generic description — this is a strong, consistent, machine-greppable convention across every file read during this mapping pass.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/agents/instructions.go` | config | n/a | Static marker-block template text (the short `codegraph explore` pointer block) — no existing "template constant" file in the codebase; use RESEARCH.md's Pattern 3 comment block as the content source, keep it a single exported `const`. |
| `internal/agents/toml.go` | utility | file-I/O | Hand-rolled TOML table-block splice (Codex only) — no TOML-editing code exists anywhere in this project; RESEARCH.md's Alternatives Considered table fully specifies the "find header, splice to next `[...]` or EOF" algorithm (mirrors TS's own `toml.ts`). |
| `internal/agents/shared.go` (JSONC/hujson body specifically) | utility | file-I/O | `hujson.Parse → Patch → Format → Pack` opencode-only workflow — no JSONC/comment-preserving-edit code exists in the repo; new third-party dependency (`github.com/tailscale/hujson`), no in-repo precedent. |
| `internal/agents/hermes.go` (YAML surgery body) | service | file-I/O | Line-range YAML list-indent-detection surgery — no YAML handling of any kind exists in this codebase today; RESEARCH.md Pitfall 5 fully specifies the indent-detection requirement, treat as new-with-no-analog. |
| `internal/version/version.go` | config | n/a | First ldflags-injected build-metadata package in the project — `cmd/codegraph/main.go` has zero version wiring today (confirmed by reading the file in full: 18 lines, no version imports). Greenfield per D-09. |
| `internal/upgrade/verify.go` | service | transform | First cryptographic-verification / first network-fetching code in the project — no analog possible; use RESEARCH.md's `sigstore-go` code example directly. |

## Metadata

**Analog search scope:** `internal/cli/` (all 20 existing command files), `internal/indexer/` (registry + per-language files), `cmd/codegraph/main.go`
**Files scanned:** `root.go`, `serve.go`, `uninit.go`, `init.go`, `sync.go`, `index.go`, `daemon.go`, `main.go`, `languages.go`, `languages_go.go` (read in full); `affected.go`/`callees.go`/`callers.go`/`files.go`/`impact.go`/`explore.go`/`node.go` (grepped for the `resolveStartPath`/`targetRoot` idiom, confirming it's project-wide, not incidental to one file)
**Pattern extraction date:** 2026-07-12

---

## PATTERN MAPPING COMPLETE

**Phase:** 6 - Agent Integrations & CLI Lifecycle
**Files classified:** 25
**Analogs found:** 15 exact/role-match, 6 partial, 6 no-analog (some files appear in both a partial-match row and a no-analog row for distinct sub-components, e.g. `hermes.go`'s registration shape vs its YAML-surgery body)

### Coverage
- Files with exact analog: 4 (`types.go`, `registry.go`, `uninstall.go`, `root.go`/`main.go` registration points)
- Files with role-match analog: 8 (the 5 uniform per-agent JSON targets, `install.go`, `upgrade.go`, `version.go`)
- Files with partial-match analog (structural shape only, novel body): 8 (`codex.go`, `opencode.go`, `hermes.go`, `shared.go`, `swap.go`, `upgrade.go` orchestrator, `telemetry.go`)
- Files with no analog: 6 (`instructions.go`, `toml.go`, hujson body, YAML-surgery body, `version.go` package, `verify.go`)

### Key Patterns Identified
- **Registry-keyed-by-ID** (`internal/indexer/languages.go`) is the load-bearing structural template for `internal/agents/registry.go` — map + per-variant self-registering `init()` + sorted list-all export, ported almost mechanically.
- **Cobra thin-command delegation** (`serve.go`/`uninit.go`/`init.go`) — resolve → delegate → print-via-cmd-writers → propagate error, applies uniformly to all 5 new commands.
- **`confirm()`'s non-interactive-safe-default idiom** (`uninit.go`) is the direct precedent for D-03's multi-select-with-auto-fallback, though the TTY-detection check itself is a new primitive with no prior instance in this codebase.
- The phase's genuinely novel surface (JSON/marker surgical writes, JSONC via hujson, YAML surgery, TOML splice, sigstore verification, atomic self-replace) has **zero in-repo analog** by design — this project has never before edited third-party external files or touched cryptography/network; RESEARCH.md's fully-specified code examples (Patterns 1-3, the sigstore wiring, the marker-upsert function) are the correct implementation source for planners, not a codebase analog.

### File Created
`/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-agent-integrations-cli-lifecycle/06-PATTERNS.md`

### Ready for Planning
Pattern mapping complete. Planner can now reference analog patterns in PLAN.md files.
