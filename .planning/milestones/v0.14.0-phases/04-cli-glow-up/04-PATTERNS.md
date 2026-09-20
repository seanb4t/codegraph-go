# Phase 4: CLI Glow-up - Pattern Map

**Mapped:** 2026-09-17
**Files analyzed:** 27 (new + modified)
**Analogs found:** 24 / 27

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/cli/colorflag.go` (new) | middleware | request-response | `internal/cli/status.go` (inline resolver call site, lines 86-88) + `internal/cli/progress_cli.go` (lines 30-37) | role-match (pattern to centralize, not a pre-existing resolver file) |
| `internal/cli/colorflag_test.go` (new) | test | request-response | `internal/cli/present/tty_test.go` | exact (pure-function table test shape) |
| `internal/cli/present/palette.go` (new) | utility/config | transform | `internal/cli/present/styles.go` | exact (D-05: the 3 vars this file declares fold directly into the new struct) |
| `internal/cli/present/line.go` (new) | utility | transform | `internal/cli/present/status.go` (`writeStatLine`, lines 101-107) | role-match |
| `internal/cli/present/help.go` (new, only if fang declined) | component | transform | `internal/cli/present/status.go` (`RenderStatus`) | role-match |
| `internal/cli/present/explore.go` (new) | component | transform | `internal/cli/present/status.go` (`RenderStatus`, full file) | exact (struct-driven, multi-section, same package) |
| `internal/cli/present/node.go` (new) | component | transform | `internal/cli/present/status.go` (`RenderStatus`, full file) | exact |
| `internal/cli/present/search.go` (new) | component | transform | `internal/cli/present/files.go` (`RenderFiles`) | exact (flat list-of-entries shape) |
| `internal/cli/present/callers.go` (new) | component | transform | `internal/cli/present/files.go` (`RenderFiles`) | role-match |
| `internal/cli/present/callees.go` (new) | component | transform | `internal/cli/present/files.go` (`RenderFiles`) | role-match |
| `internal/cli/present/impact.go` (new) | component | transform | `internal/cli/present/files.go` (`RenderFiles`) | role-match |
| `internal/cli/present/affected.go` (new) | component | transform | `internal/cli/present/files.go` (`RenderFiles`) | role-match |
| `internal/cli/present/install.go` (new, `Render<Verb>` branch inside `printAgentResults`) | component | transform | `internal/cli/install.go` (`printAgentResults`, lines 139-165) | exact |
| `internal/cli/present/styles.go` (modified) | utility/config | transform | itself (pre-image) | exact — D-05 removes the 3 vars, replaced by `palette.go` |
| `internal/cli/present/archtest/import_graph_test.go` (modified) | test | transform | itself (pre-image) | exact — widen `forbiddenImportPaths` to prefix match |
| `internal/cli/root.go` (modified) | route/controller | request-response | itself (pre-image) | exact |
| `cmd/codegraph/main.go` (modified, only if fang adopted) | controller | request-response | itself (pre-image) | exact |
| `internal/cli/status.go` (modified) | controller | request-response | itself (pre-image, lines 78-96 is the idiom every other RunE below copies) | exact |
| `internal/cli/search.go` (modified) | controller | request-response | `internal/cli/status.go` (lines 78-96) | exact |
| `internal/cli/node.go` (modified) | controller | request-response | `internal/cli/explore.go` (structurally identical today — no `--json`, no styled branch) | exact |
| `internal/cli/explore.go` (modified) | controller | request-response | `internal/cli/status.go` (lines 78-96, adapted — no `--json` branch here) | role-match |
| `internal/cli/callers.go`, `callees.go`, `impact.go`, `affected.go`, `files.go` (modified) | controller | request-response | `internal/cli/status.go` (lines 78-96) | exact |
| `internal/cli/install.go`, `uninstall.go` (modified) | controller | request-response | `internal/cli/install.go` (`printAgentResults`, lines 139-165) | exact |
| `internal/cli/version.go`, `uninit.go`, `githooks.go`, `upgrade.go`, `serve.go`, `ui.go`, `daemon.go`, `telemetry.go` (modified, one-line verbs) | controller | request-response | `internal/cli/telemetry.go` (single `fmt.Fprintln`) | role-match |
| `internal/cli/testdata/plain/<verb>.golden` (new fixtures) | test fixture | file-I/O | `test/wireoracle/testdata/wireoracle/transcripts/*.golden` (frozen-transcript convention) | role-match |
| `internal/cli/plain_golden_test.go` (new) | test | file-I/O | `test/wireoracle/oracle_test.go` (`compareBytesLineByLine`, `TestTranscriptSetMatchesScenarioSet`) + `test/integration/status_files_plain_test.go` (`TestStatusFilesPlainByteIdentity`) | exact (the latter already tests exactly D-16's property, just via dual-invocation instead of a frozen fixture) |
| `internal/cli/short_flags_test.go` (new) | test | transform | `internal/cli/cli_reference_test.go` (`TestEveryRegisteredFlagIsAccountedFor`, walk-the-tree shape) | exact |
| `internal/cli/cli_reference_test.go` (extended, GroupID guard) | test | transform | itself (pre-image, same walk-the-tree idiom) | exact |
| `docs/CLI-REFERENCE.md` (regenerated) | config (build artifact) | batch | itself (pre-image, via `task docs:cli`) | exact — never hand-edited |
| `go.mod` (modified: promote `colorprofile`; add `fang/v2` if adopted) | config | batch | itself (pre-image) | exact |

## Pattern Assignments

### `internal/cli/colorflag.go` (middleware, request-response)

**Analog:** `internal/cli/status.go` lines 78-96 (the inline resolver call site to centralize) and `internal/cli/progress_cli.go` lines 30-37 (the second inline call site, on stderr's fd instead of stdout's).

**Current inline pattern being centralized** (`internal/cli/status.go:86-88`):
```go
if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
    return present.RenderStatus(result, start, cmd.OutOrStdout())
}
```

**Second call site, same shape but stderr's fd** (`internal/cli/progress_cli.go:30-37`):
```go
func startProgress(quiet bool, label string) func() {
    if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
        return func() {}
    }
    prog := present.NewProgress(os.Stderr)
    prog.Start(label)
    return prog.Stop
}
```
`progress_cli.go`'s own stderr-fd resolver is explicitly OUT OF SCOPE (04-CONTEXT.md `<domain>`) — do not unify it with the new stdout resolver; leave it exactly as-is.

**Verified target signatures to build against** (cited from RESEARCH, do not re-derive):
```go
// github.com/charmbracelet/colorprofile@v0.4.3/env.go
func Detect(output io.Writer, env []string) Profile
// github.com/charmbracelet/colorprofile@v0.4.3/writer.go
func NewWriter(w io.Writer, environ []string) *Writer
type Writer struct { Forward io.Writer; Profile Profile }
```

**Core pattern (illustrative shape from RESEARCH, exact signature is Claude's Discretion per D-09):**
```go
func resolveColorProfile(cmd *cobra.Command, colorFlag string) colorprofile.Profile {
    out := cmd.OutOrStdout()
    environ := os.Environ()
    switch colorFlag {
    case "always":
        environ = withoutEnv(environ, "NO_COLOR", "CLICOLOR")
        environ = append(environ, "CLICOLOR_FORCE=1")
    case "never":
        environ = append(environ, "NO_COLOR=1")
    case "auto":
        // pass through untouched
    }
    return colorprofile.Detect(out, environ)
}
```

**Error handling pattern:** an unknown `--color` value is a usage error (exit 1), matching cobra's own flag-validation idiom already used by `internal/cli/install.go`'s `parseLocationFlag` (lines 29-36):
```go
func parseLocationFlag(raw string) (agents.Location, error) {
    switch agents.Location(raw) {
    case agents.LocationGlobal, agents.LocationLocal:
        return agents.Location(raw), nil
    default:
        return "", fmt.Errorf("--location must be \"global\" or \"local\" (got %q)", raw)
    }
}
```

**Code-comment citation discipline (D-00/D-09):** the resolver's comment must cite `colorprofile@v0.4.3`'s NO_COLOR/CLICOLOR precedence (see RESEARCH "Code Examples" verbatim block) and the `NO_COLOR` ParseBool-gating divergence (Pitfall 1) — never re-test that behavior, only cite it.

---

### `internal/cli/colorflag_test.go` (test)

**Analog:** `internal/cli/present/tty_test.go` (full file, above) — pure-function table test over a small enum × bool matrix, exactly the CLI-03 D-09 matrix shape (`--color ∈ {auto, always, never, invalid} × TTY ∈ {yes, no}`).

**Pattern to copy:** table-driven `tests := []struct{name string; ...; want ...}{}`, `for _, tt := range tests { t.Run(tt.name, func(t *testing.T) {...}) }`. Assert the rewritten environ slice directly (not colorprofile's `Detect` output) per D-00.

---

### `internal/cli/present/palette.go` (utility/config, transform)

**Analog:** `internal/cli/present/styles.go` (full file, above).

**Imports pattern** (lines 1-12):
```go
package present

import lipgloss "charm.land/lipgloss/v2"
```

**Core pattern to replace** (lines 19-28 — the three vars folding into the new struct):
```go
var (
    headerStyle  = lipgloss.NewStyle().Bold(true)
    labelStyle   = lipgloss.NewStyle().Faint(true)
    sectionStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)
```

**Verified target API** (RESEARCH, cite with version, do not re-test):
```go
// charm.land/lipgloss/v2@v2.0.5/color.go
type LightDarkFunc func(light, dark color.Color) color.Color
func LightDark(isDark bool) LightDarkFunc
```

**D-05 shape:** `present.Palette` struct with exactly seven fields (header, label, value, path, count, warning, error), built by `NewPalette(dark bool)` using one `lipgloss.LightDark(dark)` closure per role. No mutable package-level vars, no setter — every `Render*` takes `palette Palette` as an explicit parameter (breaking change to every existing `Render*` signature: `RenderStatus`, `RenderFiles` gain a `palette Palette` parameter).

---

### `internal/cli/present/explore.go`, `internal/cli/present/node.go` (component, transform)

**Analog:** `internal/cli/present/status.go` (`RenderStatus`, full file, above) — the only existing struct-driven, multi-section `Render*` in the package.

**Imports pattern** (lines 1-12): `io`, `strings`, this package's own `internal/query` import for the result struct, no new import needed beyond what `query.ExploreDetail`/`query.NodeDetail` already require.

**Core pattern** (`RenderStatus`, lines 153-192): build via `strings.Builder`, walk sections in the exact order the plain-text renderer uses, apply `headerStyle`/`labelStyle`/`sectionStyle` (now palette-field-based) as structural chrome only, `io.WriteString(w, b.String())` at the end. D-07 requires the styled branch calls `eng.ExploreDetail`/`eng.NodeDetail` — confirmed present at `internal/query/detail.go:681` / `:250` — never re-derives content from the markdown string `eng.Explore`/`eng.Node` already return.

**Sanitization pattern (CR-01), must copy verbatim discipline** (lines 158-159, 168-171):
```go
fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Project:"), sanitizeControl(projectPath))
```
Every user-supplied string reaching the styled render (symbol names, file paths, search terms) must pass through `sanitizeControl` first — this is the established CR-01 precedent, not new work per file.

**Error handling pattern:** `RenderStatus` returns only the `io.WriteString` error; no other error path exists (present functions never fail on their own data — they only fail on the write itself). Mirror this: no new validation happens inside `present`, all validation already happened in the engine call.

---

### `internal/cli/present/search.go`, `callers.go`, `callees.go`, `impact.go`, `affected.go` (component, transform)

**Analog:** `internal/cli/present/files.go` (`RenderFiles`, full file, above) — flat list-of-entries shape, matching these five verbs' plain output (`"%s (%s) %s:%d\n"` per-entry lines, no nested sections).

**Core pattern** (lines 37-50):
```go
func RenderFiles(r query.FilesResult, w io.Writer) error {
    var b strings.Builder
    if r.Format == "tree" {
        writeFileTree(&b, r.Tree, "")
    } else {
        for _, f := range r.Files {
            fmt.Fprintf(&b, "%s (%s)\n", sanitizeControl(f.Path), labelStyle.Render(f.Language))
        }
    }
    _, err := io.WriteString(w, b.String())
    return err
}
```
Adapt this loop shape for each verb's own result slice (`query.Location` for search/callers/callees, `query.ImpactResult.Affected`/`AffectedResult`), sanitizing `Name`/`FilePath` (filesystem/symbol-derived, potentially adversarial) exactly as `f.Path`/`n.Name` are sanitized here.

---

### `internal/cli/present/line.go` (utility, transform)

**Analog:** `internal/cli/present/status.go`'s `writeStatLine` (lines 101-107):
```go
func writeStatLine(b *strings.Builder, label, value string) {
    fmt.Fprintf(b, "  %s%s\n", labelStyle.Render(fmt.Sprintf("%-*s", statLabelWidth, label+":")), value)
}
```
D-08's `present.Line`/`present.Lines` is this same "apply one palette role's style to one line, write it" idiom generalized to a standalone exported helper consumed by the one-line verbs (`version`, `uninit`, `githooks`, `upgrade`, `serve`/`ui` banners, `daemon` status lines, `telemetry`).

---

### `internal/cli/present/help.go` (component, transform — only if fang declined, D-14)

**Analog:** `internal/cli/present/status.go` (`RenderStatus`) for the styling-application idiom; the layout itself walks cobra's own command/group/flag metadata (new territory — no existing renderer in this package consumes a `*cobra.Command` tree).

**Verified cobra signatures to build against** (RESEARCH "Pattern 4", cite with version v1.10.2, do not re-test cobra's own rendering):
```go
type Group struct { ID string; Title string }
// Command struct field, command.go:77
GroupID string
func (c *Command) AddGroup(groups ...*Group)
func (c *Command) SetHelpFunc(f func(*Command, []string))
func (c *Command) SetHelpCommandGroupID(groupID string)
func (c *Command) SetCompletionCommandGroupID(groupID string)
```
Wire via `root.SetHelpFunc(...)` in `root.go`, gated by the same D-09 resolver so `<verb> --help` inherits root's styling (D-14). Layout order must render the four D-13 groups in that order — otherwise Claude's Discretion.

---

### `internal/cli/present/install.go` (component, transform — the `printAgentResults` styling seam)

**Analog:** `internal/cli/install.go`'s `printAgentResults` (lines 139-165, full excerpt above).

**Core pattern to branch on:**
```go
func printAgentResults(cmd *cobra.Command, targets []agents.AgentTarget, loc agents.Location, do func(agents.AgentTarget) agents.WriteResult, statusOf func(agents.WriteResult) string) error {
    out := cmd.OutOrStdout()
    if len(targets) == 0 {
        fmt.Fprintln(out, "no agents selected")
        return nil
    }
    var errs []error
    for _, t := range targets {
        if !t.SupportsLocation(loc) {
            fmt.Fprintf(out, "%s: unsupported (%s not supported)\n", t.DisplayName(), loc)
            continue
        }
        result := do(t)
        fmt.Fprintf(out, "%s: %s\n", t.DisplayName(), statusOf(result))
        for _, f := range result.Files {
            fmt.Fprintf(out, "  %s: %s\n", f.Action, f.Path)
        }
        for _, note := range result.Notes {
            fmt.Fprintf(out, "  note: %s\n", note)
        }
        for _, e := range result.Errors {
            fmt.Fprintf(out, "  error: %v\n", e)
            errs = append(errs, fmt.Errorf("%s: %w", t.DisplayName(), e))
        }
    }
    return errors.Join(errs...)
}
```
D-08's plan: branch inside this function over `agents.WriteResult` — apply palette roles (label for action verbs, warning for `f.Action != Unchanged`, error for the `errs` loop) — rather than adding a parallel `Render<Verb>` file, since the data this function walks is not itself a query-engine struct.

**Error handling pattern:** `errors.Join(errs...)` — a hard write failure must never look identical to "unchanged" (CR-01). Preserve this exactly; only the printed line's styling changes.

---

### `internal/cli/status.go`, `search.go`, `callers.go`, `callees.go`, `impact.go`, `affected.go`, `files.go` (controller, request-response — RunE extension)

**Analog:** `internal/cli/status.go` lines 48-104 (full `newStatusCmd`, above) — the three-call-site RunE idiom every one of these files already partially follows.

**Core pattern (the exact idiom being extended, `status.go:78-96`):**
```go
if jsonOut {
    data, err := query.MarshalStatusJSON(result)
    if err != nil {
        return err
    }
    return writeJSONLine(cmd, data)
}

if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
    return present.RenderStatus(result, start, cmd.OutOrStdout())
}

fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
return nil
```
**What changes per D-09/D-10:** replace the inline `term.IsTerminal(...)`/`os.Getenv(...)` pair with a call to the new `colorflag` resolver, and wrap `cmd.OutOrStdout()` in a `colorprofile.Writer` before calling `present.Render*` — `present.Render*`'s own signature/body needs no change for this (D-10). The `--json` early return and the plain-text fallback are structurally UNCHANGED (this is exactly what keeps CLI-05's byte-identity guarantee true).

**Short-flag pattern already correct (D-12), do not add new ones** (`search.go:135-139`):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().StringVarP(&kind, "kind", "k", "", "restrict to one node kind")
cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
```

---

### `internal/cli/node.go`, `internal/cli/explore.go` (controller, request-response — gains a styled branch that does not exist today)

**Analog:** each other, since both are today's identical "no `--json`, no styled branch" shape (full files, above) — confirmed by RESEARCH: neither calls anything but the markdown-string `eng.Explore`/`eng.Node`.

**Current pattern being extended** (`explore.go:32-68`, full RunE body above): resolve path → open engine → call `eng.Explore(exploreQuery, maxFiles)` → print `query.WorktreeNotice(...)` then the markdown string verbatim.

**D-07's addition:** insert a styled branch between the worktree-notice print and the markdown print, calling `eng.ExploreDetail`/`eng.NodeDetail` (verified present, `internal/query/detail.go:681`/`:250`) — same section order/wording as the markdown, hue instead of `#`/`**`/backtick syntax. The plain markdown branch is UNCHANGED.

---

### `cmd/codegraph/main.go` (controller, request-response — only if fang adopted, D-03)

**Analog:** itself, pre-image (full file, above):
```go
func main() {
    if err := cli.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**Verified fang mechanics forcing the change** (RESEARCH "Pattern 5", `fang.go`/`help.go` v2.0.1, cite with version, do not re-test):
```go
func Execute(ctx context.Context, root *cobra.Command, options ...Option) error {
    ...
    if err := root.ExecuteContext(ctx); err != nil {
        w := colorprofile.NewWriter(root.ErrOrStderr(), os.Environ())
        opts.errHandler(w, makeStyles(mustColorscheme(opts.colorscheme)), err) // prints err.Error() EXACTLY ONCE
        return err
    }
    return nil
}
```

**Required change if adopted:**
```go
func main() {
    if err := cli.Execute(); err != nil { // cli.Execute() now calls fang.Execute internally,
        os.Exit(1)                         // which already printed err.Error() exactly once
    }
}
```

**Regression this guards against (D-03):** `internal/cli/renamed_test.go`'s `TestQueryStub`/`TestUnlockStub` and `test/integration/renamed_stubs_test.go`'s `TestRenamedStubsPrintExactlyOnce` (both read above) assert the two-line rename message prints to stderr EXACTLY ONCE — a double-print (main.go's own `fmt.Fprintln` plus fang's `DefaultErrorHandler`) breaks both. If fang cannot honor this, D-03 says decline fang (D-14 hand-rolled help instead) — `main.go` stays as the pre-image shown above.

---

### `internal/cli/root.go` (route/controller, request-response)

**Analog:** itself, pre-image (full file, above).

**Core pattern to extend** (`newRootCmd`, lines 54-73): the `AddCommand(...)` list stays; add `root.AddGroup(...)` calls and set `cmd.GroupID` at each subcommand's own construction site (not centrally in `root.go` — each `new<Verb>Cmd()` sets its own `GroupID` per D-13's four-group table) plus the persistent `--color` flag (default `"auto"`, D-11) and, only if fang is adopted per the D-01…D-04 verdict, wrap `Execute()`:
```go
func Execute() error {
    return newRootCmd().Execute()
}
```
becomes (if adopted):
```go
func Execute() error {
    return fang.Execute(context.Background(), newRootCmd(), fang.WithoutManpage(), fang.WithVersion(versionLine()))
}
```
`WithoutCompletions()` is deliberately NOT called — see Common Pitfall #3 in RESEARCH: it disables cobra's own `completion` command, which GoReleaser depends on. `NewRootCmd()` (the clidoc entry point, lines 82-90) stays byte-identical either way.

**D-13 group table (verbatim from CONTEXT.md), to apply as each verb's `GroupID` field:**
| Group ID (Claude's Discretion for the literal string) | Title | Verbs |
|---|---|---|
| `query` | "Query the graph:" | explore, search, node, callers, callees, impact, affected, files, status |
| `build` | "Build the index:" | init, index, sync, daemon, githooks, uninit |
| `agents` | "Agents & serving:" | serve, ui, install, uninstall |
| `maintenance` | "Maintenance:" | version, upgrade, telemetry, help, completion |

Hidden `man`/`query`/`unlock` (the `renamed.go` stubs) stay groupless — do not assign a `GroupID` to `newManCmd()`, `newQueryCmd()`, or `newUnlockCmd()`.

---

### `internal/cli/plain_golden_test.go` + `internal/cli/testdata/plain/<verb>.golden` (test / test fixture, file-I/O)

**Analog:** `test/integration/status_files_plain_test.go`'s `TestStatusFilesPlainByteIdentity` (full file, above) — already tests exactly D-16's property (non-TTY output has zero ESC bytes, `NO_COLOR` toggling doesn't change bytes) via dual-invocation comparison rather than a frozen fixture; and `test/wireoracle/oracle_test.go`'s frozen-transcript convention (`TestTranscriptSetMatchesScenarioSet`, `compareBytesLineByLine`) for the freeze-then-compare shape D-16 actually asks for.

**Core pattern to copy from `status_files_plain_test.go`:**
```go
if strings.Contains(plainOut, "\x1b[") {
    t.Errorf("%v: non-TTY output contains an ANSI escape sequence:\n%s", tc.args, plainOut)
}
```
D-16 additionally requires: capture this corpus to `testdata/plain/<verb>.golden` files BEFORE any renderer lands (the phase's first test commit), then after the glow-up assert byte-equality against the frozen file AND zero ESC bytes — mirror `test/wireoracle`'s `TranscriptPath`/frozen-file-read idiom (`scenarios.go:1439`, `filepath.Join("..", "..", "testdata", "wireoracle", "transcripts", name+".golden")`) for the path convention, adapted to `internal/cli/testdata/plain/<verb>.golden`.

---

### `internal/cli/short_flags_test.go` (test, transform)

**Analog:** `internal/cli/cli_reference_test.go`'s `TestEveryRegisteredFlagIsAccountedFor` (full file, above) — the walk-the-tree-via-cobra idiom (`cmd.Flags().VisitAll`, recursive `walk(sub)` over `cmd.Commands()`).

**Core pattern to copy** (lines 176-244, the walk shape):
```go
var walk func(cmd *cobra.Command)
walk = func(cmd *cobra.Command) {
    cmd.Flags().VisitAll(func(f *pflag.Flag) { ... })
    for _, sub := range cmd.Commands() {
        walk(sub)
    }
}
walk(root)
```
D-12's table test: for each of the four long names (`--json`, `--limit`, `--kind`, `--path`) present on a query verb, assert its short (`-j`/`-l`/`-k`/`-p`) also exists via `pflag.Flag.Shorthand`. Positive control: plant a long-only flag on a throwaway command and assert it goes RED (rule `84d1gfpywd`).

---

### `internal/cli/cli_reference_test.go` (extended: GroupID guard, CLI-06)

**Analog:** itself, pre-image (full file, above) — same walk-the-tree idiom `TestEveryRegisteredFlagIsAccountedFor` already establishes; extend with a sibling `TestEveryCommandHasGroupID` walking `cmd.Commands()` recursively and asserting every visible (`documentedByReference`-eligible) command's `GroupID` is non-empty and one of the four D-13 IDs. Reuse `documentedByReference` (lines 57-64) to exclude the hidden `man`/`query`/`unlock` stubs from the requirement.

---

### `internal/cli/present/archtest/import_graph_test.go` (test, transform — D-15)

**Analog:** itself, pre-image (full file, above).

**Exact edit:** `forbiddenImportPaths` (lines 51-55) changes from an exact-match literal list to prefix matching on two roots:
```go
// before:
var forbiddenImportPaths = []string{
    "charm.land/lipgloss/v2",
    "charm.land/bubbletea/v2",
    "charm.land/bubbles/v2",
}
// after (illustrative — exact helper shape is implementation's choice):
var forbiddenImportPathPrefixes = []string{
    "charm.land/",
    "github.com/charmbracelet/",
}
```
The match loop (`TestNoCharmInServeReachablePackages`, lines 187-199) changes from `pkg.Imports[forbidden]` exact lookup to a `strings.HasPrefix` walk over `pkg.Imports` keys. `charmImporterProbePath` (line 61) and the self-defeat guard `assertCharmImporterExists` stay unchanged — they remain the positive control. Prove RED via a planted import from each newly-required module (`colorprofile`; `fang/v2` if adopted) in `04-MUTATION-LOG.md`, mirroring the `03-MUTATION-LOG.md` shape (see Shared Patterns below).

**Explicitly NOT touched:** `charm_cgo_test.go`'s `charmClosurePathPrefix = "charm.land"` (a separate, narrower guard — D-02 only requires it stay green, not be widened).

---

## Shared Patterns

### Auth/env-boundary discipline: `present` stays env-blind
**Source:** `internal/cli/present/tty.go` doc comment + `internal/cli/present/status.go` doc comment (lines 7-9, 1-9)
**Apply to:** every new `present.Render*`/`present.Line` file and `colorflag.go`
```go
// present must NOT read os.Getenv or call term.IsTerminal itself — real
// fd/env values are read only at the RunE call sites in internal/cli
```
This boundary is unchanged by the phase (D-03/D-09/D-10): `present` never reads env/fd; only `internal/cli` RunE sites (via the new `colorflag.go` resolver) do.

### Sanitization before styled render (CR-01)
**Source:** `internal/cli/present/sanitize.go` (full file, above), used at `present/status.go:159,169-170` and `present/files.go:22,25,45`
**Apply to:** every new `present.Render*` file that renders a filesystem path, symbol name, or search term
```go
fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Project:"), sanitizeControl(projectPath))
```
Any user-supplied or filesystem-derived string reaching a styled render must pass through `sanitizeControl` first. This guards the pretty path only — the plain/piped renderer in `internal/query` is frozen and untouched (TUI-02).

### The three-call-site RunE idiom (unchanged skeleton, only the third step's content changes)
**Source:** `internal/cli/status.go:57-96` (full RunE, above)
**Apply to:** every verb file listed under "controller" above
```
resolve start path → open engine → --json early return → styled/plain branch
```
The `--json` early return and plain fallback NEVER move; only the styled branch's internals change (resolver + `colorprofile.Writer` wrap + `present.Render*`). This mechanical invariant is what makes CLI-05's byte-identity guarantee true by construction, not merely by test.

### Error handling: single main.go print, SilenceUsage/SilenceErrors everywhere
**Source:** `internal/cli/root.go` (`SilenceUsage: true, SilenceErrors: true`, lines 62-63) + `cmd/codegraph/main.go` (pre-image, above)
**Apply to:** every command in the tree (unchanged), `main.go` (conditionally changed per D-03 if fang adopted)
Every command sets `SilenceUsage`/`SilenceErrors`; `main.go` prints the returned error once and exits 1 — the only error exit path. If fang is adopted, `main.go` must stop printing (fang's `DefaultErrorHandler` already does, exactly once) — see the `cmd/codegraph/main.go` pattern assignment above.

### Injectable package-level func-var seam (for anything needing a test double)
**Source:** `internal/cli/install.go:22-23`
```go
var interactiveAllowed = tui.InteractiveAllowed
var runAgentPicker = tui.RunAgentPicker
```
Apply this idiom if the fang spike or resolver needs a stubbable seam (e.g. a fake `colorprofile.Detect` or `lipgloss.HasDarkBackground` call for a unit test) — mirrors this project's established convention (`upgradeRunFunc`, `refreshInstalledSkillsFunc` in `upgrade.go`) rather than inventing a new DI mechanism.

### Mutation-log shape for RED proofs (D-15, D-16, D-12)
**Source:** `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` (excerpt above) — pre-mutation cleanliness gate (`git status --porcelain` / `git diff --quiet`), a plainly-named "What are we testing, and why?" section, verbatim observed output, explicit revert + byte-clean proof.
**Apply to:** `04-MUTATION-LOG.md` for every planted-RED proof this phase needs (D-15's archtest widening, D-16's golden-freeze regression, D-12's short-flag pinning test, D-03's stub double-print check).

## No Analog Found

Files with no close match (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `04-FANG-VERDICT.md` | config/doc (phase artifact) | batch | No prior `*-VERDICT.md` exists in this repo; D-04 only cites `03-MUTATION-LOG.md`'s general phase-artifact *shape* (evidence-carrying, dated, phase-directory-local) as precedent, not a literal template — use RESEARCH.md's "Code Examples" Pattern 5 block plus the four D-01…D-04 criteria as the content outline. |
| `internal/cli/present/help.go` | component | transform | No existing `present` renderer consumes a `*cobra.Command` tree (every existing `Render*` consumes a plain `internal/query` result struct) — this is genuinely new territory; build against the verified cobra `Group`/`SetHelpFunc` signatures cited above, only if D-14's fang-declined branch is taken. |
| `internal/cli/colorflag.go`'s exact resolver signature | middleware | request-response | No pre-existing single-environment-read resolver exists to copy verbatim (today's three call sites each read env/fd inline) — RESEARCH's illustrative `resolveColorProfile` shape is the closest available reference, explicitly marked Claude's Discretion in D-09. |

## Metadata

**Analog search scope:** `internal/cli/`, `internal/cli/present/`, `internal/cli/present/archtest/`, `internal/query/` (signatures only, no changes), `test/wireoracle/`, `test/integration/`, `cmd/codegraph/`, `.planning/phases/03-verb-fold/` (mutation-log convention)
**Files scanned:** ~30 (all of `internal/cli/*.go` top-level command files, all of `internal/cli/present/*.go`, both `archtest/*.go` files, `root.go`, `cli_reference_test.go`, `renamed.go`/`renamed_test.go`, `cmd/codegraph/main.go`, `test/integration/renamed_stubs_test.go` and `status_files_plain_test.go`, `test/wireoracle/oracle_test.go` and `scenarios.go`, `03-MUTATION-LOG.md`)
**Pattern extraction date:** 2026-09-17
**Tracked-source gate:** every analog path listed above was confirmed via `git ls-files` to be tracked source in this repository (not a gitignored mirror) — this repo has no `.gsd/capabilities/` install-mirror structure; all analogs are the repo's own primary source tree.
