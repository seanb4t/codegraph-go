# 04-FANG-VERDICT — fang/v2 spike (CLI-08)

**Date:** 2026-09-17
**Candidate:** `charm.land/fang/v2 v2.0.1` (resolved via `proxy.golang.org`; source read from
`/Users/sean/go/pkg/mod/charm.land/fang/v2@v2.0.1/`, same module the milestone research pinned)

## Shipped call set (candidate, as spiked)

```go
fang.Execute(context.Background(), newRootCmd(), fang.WithoutManpage(), fang.WithVersion(versionLine()))
```

`fang.WithoutCompletions()` is **deliberately NOT called** (D-04 research correction, applied
exactly as specified). `charm.land/fang/v2@v2.0.1/fang.go:163-165`:

```go
if !opts.completions {
    root.CompletionOptions.DisableDefaultCmd = true
}
```

`opts.completions` defaults to `true` (`fang.go:113`), and `WithoutCompletions()` (`fang.go:42-46`)
is the only thing that flips it — it does not avoid a fang-owned completion command (fang has
none); it sets `cobra.CompletionOptions.DisableDefaultCmd = true`, which **removes cobra's own**
default `completion` command — the exact command GoReleaser's `generate_completions_from_executable`
step depends on. Leaving `opts.completions` at its default therefore leaves cobra's own
`completion` command fully intact; `WithoutManpage()` (`fang.go:48-53`) is confirmed correct as-is,
since fang only adds its own hidden `man` command when `opts.manpages == true` (`fang.go:142-161`).

## Criterion 1 — `serve --mcp` byte-identical (D-01)

**Verdict: PASS**

```
$ GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	49.672s
?   	github.com/seanb4t/codegraph-go/test/wireoracle/cmd/wireoracle	[no test files]
```

`test -z "$(git status --porcelain testdata/wireoracle)"` held — no transcript drift. All 38
frozen transcripts, spawning the real binary under the wrapped `Execute()`, matched.

## Criterion 2 — SBOM/govulncheck delta acceptable (D-02)

**Verdict: PASS**

Baseline (clean tree, before `go get charm.land/fang/v2@v2.0.1`):

```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 3 vulnerabilities in packages you import and 3
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
```

After (fang wired in, `go.mod`/`go.sum` modified, `root.go`/`main.go` edited):

```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 3 vulnerabilities in packages you import and 3
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
```

`diff` of the `Vulnerability #` header lines between the two runs is empty — zero new findings.

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/archtest/ -run TestCharmCgoClosure -count=1 -v
charm_cgo_test.go:104: charm.land closure audited: 11 packages, 0 with CgoFiles
--- PASS: TestCharmCgoClosure (0.28s)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	0.366s
```

## Criterion 3 — `query`/`unlock` stub contract preserved exactly once (D-03)

**Verdict: FAIL**

```
$ GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run TestRenamedStubsPrintExactlyOnce -count=1 -v
renamed_stubs_test.go:53: [query main]: stderr = "          \n   ERROR  \n          \n  \"Query\" has been renamed to \"search --full\" — run: codegraph search --full <term>                                   \n  the \"query\" stub is removed in the next minor release (v0.15.0).                                                    \n\n", want exactly "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\nthe \"query\" stub is removed in the next minor release (v0.15.0)\n" (the message printed exactly once by main.go, WR-01)
--- FAIL: TestRenamedStubsPrintExactlyOnce (0.68s)
    --- FAIL: TestRenamedStubsPrintExactlyOnce/query_bare (0.02s)
    --- FAIL: TestRenamedStubsPrintExactlyOnce/query_with_--json_flag (0.01s)
    --- FAIL: TestRenamedStubsPrintExactlyOnce/unlock_bare (0.01s)
    --- FAIL: TestRenamedStubsPrintExactlyOnce/unlock_with_nonexistent_path_arg (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/integration	4.734s
```

The message DOES print exactly once (no double-print — `main.go`'s own `fmt.Fprintln` was
correctly removed and fang's `DefaultErrorHandler` is the sole print path). The failure is a
different, more fundamental problem: **fang always renders the styled TTY box, even on a
non-TTY pipe.** Root cause, confirmed by reading `charm.land/fang/v2@v2.0.1/help.go:110-119`:

```go
func DefaultErrorHandler(w io.Writer, styles Styles, err error) {
    if w, ok := w.(term.File); ok {
        if !term.IsTerminal(w.Fd()) {
            _, _ = fmt.Fprintln(w, err.Error())
            return
        }
    }
    // styled box path — always reached if the first branch's type assertion fails
    ...
}
```

`fang.Execute` (`fang.go:173-177`) always calls this handler with
`w = colorprofile.NewWriter(root.ErrOrStderr(), os.Environ())` — a `*colorprofile.Writer`.
Confirmed by reading `colorprofile@v0.4.3/writer.go` directly: `*colorprofile.Writer` implements
only `Write`/`WriteString`/`downsample` — **it has no `Fd()` method**, so it does not satisfy
`term.File`, and the `w.(term.File)` type assertion **always fails**, regardless of whether the
underlying `root.ErrOrStderr()` is actually a terminal. The non-TTY plain-print branch
(`fmt.Fprintln(w, err.Error())`) is therefore **structurally unreachable** through
`fang.Execute`'s own wrapping — every error, on every invocation, TTY or piped, renders the
styled box (title-cased message, box-drawing whitespace, trailing period appended to
`err.Error()`). This is not a configuration mistake in this spike; it is how `fang.Execute`
composes `colorprofile.NewWriter` with `DefaultErrorHandler` in v2.0.1.

This breaks D-03 outright: `codegraph query`/`codegraph unlock`'s stub contract (Phase 3 WR-01)
requires the exact two-line message on stderr for scripts/agents that parse it, and any other
command's error output inherits the same problem. `fang.WithErrorHandler` could theoretically
override this, but doing so means hand-rolling the same TTY-detection logic today's
`present`/`cmd/codegraph/main.go` combination already gets for free — at which point fang is
buying nothing D-14's hand-rolled path doesn't already provide, while adding a process-boundary
dependency. Per D-04, this is disposition-determining on its own: **fang is declined.**

## Criterion 4 — composition with existing hidden `man` and cobra `completion` (D-04, corrected)

**Verdict: PASS** (evaluated for completeness — D-04's four criteria are conjunctive, so
Criterion 3's FAIL alone is sufficient to decline, but this criterion was run regardless per the
task's own evidence-gathering instructions)

```
$ /tmp/04-02-spike-bin completion bash | wc -c
16333
$ /tmp/04-02-spike-bin completion zsh | wc -c
7856
$ /tmp/04-02-spike-bin completion fish | wc -c
9969
$ /tmp/04-02-spike-bin man <tmpdir> && ls <tmpdir> | wc -l
29
$ /tmp/04-02-spike-bin --help | rg -c completion
1
$ /tmp/04-02-spike-bin --version
codegraph version dev (commit unknown, built unknown)
```

All three shells' completion scripts emit well over the 1000-byte non-empty floor; `man <dir>`
wrote 29 `.1` files; `--help` lists `completion` among the commands; `--version` output is
byte-identical to the pre-spike baseline (`codegraph version dev (commit unknown, built
unknown)`, captured from `/tmp/04-02-before-bin --version` before any spike edit).

## Information (not gates)

**Modules fang would add** (`go: added …`, from `GOTOOLCHAIN=go1.26.6 go get charm.land/fang/v2@v2.0.1`,
no count assertion per D-00):
```
go: added charm.land/fang/v2 v2.0.1
go: added github.com/charmbracelet/x/exp/charmtone v0.0.0-20250603201427-c31516f43444
go: added github.com/muesli/mango v0.1.0
go: added github.com/muesli/mango-cobra v1.2.0
go: added github.com/muesli/mango-pflag v0.1.0
go: added github.com/muesli/roff v0.1.0
```

**`codegraph --version` output identity:** `codegraph version dev (commit unknown, built unknown)`
before and after the wrap — byte-identical.

**Does `--color=never` (not yet existing) govern fang's help/error chrome?** No — as
04-RESEARCH.md predicted, `fang.go`/`help.go` read `os.Environ()` directly (`colorprofile.NewWriter(...,
os.Environ())` at both the help-func and error-handler call sites), not any value this repo's
own resolver would compute. Confirmed empirically:
```
NO_COLOR=1    --help ESC-line-count: 0
CLICOLOR_FORCE=1 --help ESC-line-count: 34
```
Fang's chrome does honour the real process environment directly. Plan 08's adopted branch was
expected to mirror `--color` into the process env for exactly this reason — moot now that fang
is declined, but recorded since it was empirically confirmed either way.

**`TestPlainGolden`/`TestNoColorNonTTYRegression` under the wrap:**
```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$|TestNoColorNonTTYRegression$'
ok  	github.com/seanb4t/codegraph-go/internal/cli	6.955s
```
Both stayed green under the fang wrap (unaffected by design — no renderer/resolver exists yet
that fang's wrap would interact with; this golden path is entirely inside RunE, below fang's
process-boundary wrap).

**Observed non-TTY stub error shape under fang** (`codegraph query main 2>&1 >/dev/null | cat -v`):
```
          
   ERROR  
          
  "Query" has been renamed to "search --full" M-BM- run: codegraph search --full <term>                                   
  the "query" stub is removed in the next minor release (v0.15.0).
```
This is the styled box (title case applied to the message, box-drawing padding, trailing period)
even though the pipe target is non-TTY — this is Criterion 3's failure made visible directly,
not merely inferred from the test failure.

## Key Decisions row (for phase close)

| Decision | Rationale |
|---|---|
| **fang/v2 declined** for `cli.Execute()` (04-02 spike, 2026-09-17). `fang.Execute`'s `DefaultErrorHandler`, wrapped by fang itself in a `*colorprofile.Writer`, never satisfies its own `w.(term.File)` TTY-detection type assertion (`colorprofile.Writer` has no `Fd()` method) — every error renders the styled TTY box unconditionally, on TTY or pipe alike, breaking the `query`/`unlock` stub's exact-once plain-text stderr contract (D-03, Phase 3 WR-01) and, by the same mechanism, every other command's error output. Wireoracle (D-01), govulncheck/`TestCharmCgoClosure` (D-02), and completion/man composition (D-04) all PASSED in isolation, but D-04's four criteria are conjunctive — one FAIL declines the whole candidate. Help is hand-rolled per D-14 in a later plan. |

## Verdict

**Verdict:** declined
