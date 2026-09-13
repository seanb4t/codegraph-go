# 12-MUTATION-LOG — CLI Reference & Docs Tail

**Phase:** 12-cli-reference-docs-tail
**Date:** 2026-09-13
**Scope:** Three RED demonstrations, one per instrument-discriminating claim this phase's two
new gates make: (a) DOCS-06's accounting guard, perturbed by a throwaway hidden flag registered
on `codegraph ui` (`--zz-throwaway`) — shows the guard catching a hidden flag the drift gate
cannot see; (b) DOCS-05's drift gate, perturbed by deleting the `--no-open` Options line from the
committed `docs/CLI-REFERENCE.md` — shows the drift gate catching hand-edited/stale generated
content, with the guard independently confirming the same gap; (c) DOCS-06's accounting guard
again, perturbed by appending a bogus allowlist entry (`codegraph ui --zz-bogus`) for a flag that
does not exist — shows the rot-enforcement half of the guard. No family exercises `cobra/doc`'s
own behaviour: (a) perturbs this project's `internal/cli/ui.go` Cobra tree, (b) perturbs this
project's committed `docs/CLI-REFERENCE.md`, (c) perturbs this project's own
`internal/cli/testdata/cli-reference-allowlist.txt` — `cobra/doc.GenMarkdownCustom` itself is
never touched, stubbed, or asserted against in any family (D-09).

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is
asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and
no revert was a destructive blind checkout of someone else's in-flight work (11-MUTATION-LOG.md
convention, itself following the 10-MUTATION-LOG.md precedent).

```
$ git diff --quiet -- internal/cli/ui.go docs/CLI-REFERENCE.md internal/cli/testdata/cli-reference-allowlist.txt; echo $?
0
```

---

## Family (a) — DOCS-06: throwaway hidden flag registered on `ui`

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEveryRegisteredFlagIsAccountedFor' ./internal/cli/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/cli/ui.go; echo $?
0
```

**Mutation applied.** After the `--no-editor-url` registration in `newUiCmd()`, a throwaway
hidden flag is added — a flag that exists in the live Cobra tree but nowhere in the generated
reference or the allowlist:

```diff
--- a/internal/cli/ui.go
+++ b/internal/cli/ui.go
@@ -122,6 +122,8 @@ func newUiCmd() *cobra.Command {
 	cmd.Flags().BoolVar(&noEditorURL, "no-editor-url", false,
 		"disable editor links entirely (also CODEGRAPH_NO_EDITOR_URL=true|1|yes); "+
 			"skips editor discovery")
+	cmd.Flags().Bool("zz-throwaway", false, "MUTATION 12-03 family (a) — reverted")
+	_ = cmd.Flags().MarkHidden("zz-throwaway")
 
 	return cmd
 }
```

**Confirmed applied:**
```
$ rg -n 'zz-throwaway' internal/cli/ui.go
125:	cmd.Flags().Bool("zz-throwaway", false, "MUTATION 12-03 family (a) — reverted")
126:	_ = cmd.Flags().MarkHidden("zz-throwaway")
$ GOTOOLCHAIN=go1.26.6 go build ./internal/cli/
(no output — success)
```

**Instrument 1 (the guard) — RED, pasted verbatim (exit 1):**
```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 116 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        unaccounted flag: codegraph ui --zz-throwaway (hidden flag — add an allowlist entry with a reason)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.452s
FAIL
```

The counts line reports 116 flags inspected — one more than the baseline 115 — with the new hidden
flag correctly bucketed as neither reference-eligible (hidden flags are never reference-eligible)
nor allowlist-matched, landing in `unaccounted` and naming the exact command path and flag.

**Instrument 2 (the drift gate), run with the mutation still applied — GREEN, pasted verbatim
(exit 0):**
```
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
```

**Why this pairing is the point of DOCS-06.** A hidden flag never reaches `cobra/doc`'s generated
output — hidden flags are, by Cobra's own contract, invisible to `GenMarkdownCustom`'s walk — so
`docs/CLI-REFERENCE.md` regenerates byte-identical whether or not `--zz-throwaway` exists, and the
drift gate stays green throughout. This is exactly the generator's blind spot DOCS-06 exists to
close: the drift gate proves the committed file matches what the generator produces, but says
nothing about whether the generator's *input* — the live Cobra tree — grew a flag nobody can see.
Only the guard's independent walk-and-account discipline, which inspects the live tree directly
rather than trusting the generated output, catches a hidden flag added without a matching
allowlist entry.

**Revert:**
```
$ git checkout -- internal/cli/ui.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/cli/ui.go; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.428s
```

---

## Family (b) — DOCS-05: one Options line deleted from the committed reference

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- docs/CLI-REFERENCE.md; echo $?
0
```

**Mutation applied.** The single Options line carrying `--no-open` under `## codegraph ui` is
deleted — nothing else in the file changes:

```diff
--- a/docs/CLI-REFERENCE.md
+++ b/docs/CLI-REFERENCE.md
@@ -786,7 +786,6 @@ codegraph ui [flags]
       --editor-url string   editor URI template with {path}, {line} and {col} placeholders (e.g. 'vscode://file/{path}:{line}:{col}'); overrides CODEGRAPH_EDITOR_URL and startup editor discovery
   -h, --help                help for ui
       --no-editor-url       disable editor links entirely (also CODEGRAPH_NO_EDITOR_URL=true|1|yes); skips editor discovery
-      --no-open             do not open a browser automatically
   -p, --path string         repo path (default: cwd)
 ```
```

**Confirmed applied:**
```
$ rg -o -- '--no-open' docs/CLI-REFERENCE.md | wc -l | tr -d ' '
0
```

**Instrument 1 (the drift gate) — RED, pasted verbatim (exit 201, `task -s` wrapping the
target's own exit 1; the count line prints BEFORE the comparison, per D-14):**
```
docs:cli:drift: compared 1 generated file
::error::docs:cli:drift: docs/CLI-REFERENCE.md differs from a fresh regeneration by the pinned toolchain — run `task docs:cli` and commit the result
--- docs/CLI-REFERENCE.md	2026-09-13 17:04:28
+++ /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.ouBldAjNyo/CLI-REFERENCE.md	2026-09-13 17:04:36
@@ -786,6 +786,7 @@
       --editor-url string   editor URI template with {path}, {line} and {col} placeholders (e.g. 'vscode://file/{path}:{line}:{col}'); overrides CODEGRAPH_EDITOR_URL and startup editor discovery
   -h, --help                help for ui
       --no-editor-url       disable editor links entirely (also CODEGRAPH_NO_EDITOR_URL=true|1|yes); skips editor discovery
+      --no-open             do not open a browser automatically
   -p, --path string         repo path (default: cwd)
 ```
 
task: Failed to run task "docs:cli:drift": exit status 1
```

The count line (`compared 1 generated file`) is printed before the `::error::` line and before the
`diff -u … | head -40` excerpt — the enumeration always reports its floor before comparing
anything (rule 84d1gfpywd), so a broken enumeration can never masquerade as a clean pass. The
`diff -u` excerpt shows the freshly-regenerated file's `+      --no-open` line — the exact content
the committed file is now missing.

**Instrument 2 (the guard) — RED, pasted verbatim (exit 1):**
```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 113 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        unaccounted flag: codegraph ui --no-open (visible flag on a documented command — missing from docs/CLI-REFERENCE.md; run task docs:cli)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.433s
FAIL
```

**The `--editor-url` note (from the plan's own context).** Deleting the `--editor-url` line
instead would NOT have gone RED on the whole-document guard check: `ui`'s own `Long` (Synopsis)
text repeats the literal string `--editor-url` in its own prose, so a naive whole-document
substring search would still find the name elsewhere in the file and stay green even with the
Options line gone. `--no-open` was deliberately chosen instead because it appears ONLY in the
Options line — nowhere else in `ui`'s Synopsis or any other section — so both instruments fire on
its deletion. This is the recorded DOCS-06 assumption from 12-01's must_haves: the guard's
flag-accounting check is a per-flag documentedByReference/docMentionsFlag search over the whole
document text, not a section-scoped match, so a flag name that recurs anywhere in the file
(including unrelated prose) can mask a missing Options line — the drift gate is the actual
content-completeness check here (D-09), and the guard's coverage of this specific case depends on
the flag name being unique in the document.

**Revert:**
```
$ git checkout -- docs/CLI-REFERENCE.md
```

**Post-revert gate:**
```
$ git diff --quiet -- docs/CLI-REFERENCE.md; echo $?
0
```

**GREEN — both instruments re-run at HEAD, pasted verbatim (exit 0):**
```
$ GOTOOLCHAIN=go1.26.6 task -s docs:cli:drift
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)

$ GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEveryRegisteredFlagIsAccountedFor' ./internal/cli/
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.432s
```

---

## Family (c) — DOCS-06: allowlist rot (a bogus entry matching nothing)

**Instrument:**
```
GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEveryRegisteredFlagIsAccountedFor' ./internal/cli/
```

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/cli/testdata/cli-reference-allowlist.txt; echo $?
0
```

**Mutation applied.** A line is appended to the committed allowlist naming a flag that does not
exist anywhere in the live tree (a real tab separates the key from the reason, matching the
allowlist's own documented format):

```diff
--- a/internal/cli/testdata/cli-reference-allowlist.txt
+++ b/internal/cli/testdata/cli-reference-allowlist.txt
@@ -2,3 +2,4 @@
 # (hidden commands, hidden flags, deprecated flags). One entry per line:
 # <full command path>[ --flag]<TAB><reason>. An entry matching nothing FAILS the guard.
 codegraph man	hidden by v0.5.0 D-02 — the Homebrew cask post-install hook's man-page generator, never an interactive command; its only flag is cobra's --help (12-CONTEXT D-05)
+codegraph ui --zz-bogus	MUTATION 12-03 family (c) — bogus entry for a flag that does not exist, reverted
```

**Confirmed applied:**
```
$ rg -o 'zz-bogus' internal/cli/testdata/cli-reference-allowlist.txt | wc -l | tr -d ' '
1
```

**Instrument — RED, pasted verbatim (exit 1):**
```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        stale allowlist entry: codegraph ui --zz-bogus (matches no registered command or flag)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.432s
FAIL
```

The counts line is unchanged from the clean baseline — still 115 flags inspected, still 1 flag
accepted via the allowlist (the genuine `codegraph man` entry; the bogus `--zz-bogus` entry
accepted nothing, because no flag in the live tree matches it). The failure comes entirely from
the rot-enforcement pass, which walks every allowlist entry after the tree walk completes and
fails on any entry never `used` — proving the guard treats a stale allowlist entry as a defect in
its own right, not merely as a harmless no-op.

**Revert:**
```
$ git checkout -- internal/cli/testdata/cli-reference-allowlist.txt
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/cli/testdata/cli-reference-allowlist.txt; echo $?
0
```

**GREEN — re-run at HEAD, pasted verbatim (exit 0):**
```
cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.425s
```

---

## Closing

### The three instruments, RED and GREEN

| Family | Requirement | Instrument(s) | Mutated file | RED | GREEN (post-revert) |
|--------|-------------|----------------|--------------|-----|----------------------|
| (a) | DOCS-06 | `TestEveryRegisteredFlagIsAccountedFor` (RED) + `task -s docs:cli:drift` (stays GREEN under the same mutation) | `internal/cli/ui.go` | guard: `--- FAIL`, `unaccounted flag: codegraph ui --zz-throwaway`, 116 flags inspected | guard: `--- PASS`, 115 flags |
| (b) | DOCS-05 | `task -s docs:cli:drift` (RED) + `TestEveryRegisteredFlagIsAccountedFor` (RED) | `docs/CLI-REFERENCE.md` | drift: `compared 1 generated file` then `::error::…differs…` then `diff -u` excerpt, exit 201; guard: `unaccounted flag: codegraph ui --no-open`, 113 via reference | drift: `compared 1 generated file`, byte-identical, exit 0; guard: `--- PASS`, 114 via reference |
| (c) | DOCS-06 | `TestEveryRegisteredFlagIsAccountedFor` | `internal/cli/testdata/cli-reference-allowlist.txt` | `--- FAIL`, `stale allowlist entry: codegraph ui --zz-bogus`, counts unchanged (115 flags, 1 via allowlist) | `--- PASS`, 115 flags, 1 via allowlist |

### Non-vacuity assertion

All three families were watched fail on the specific assertion each exists for, none rewritten to
force a pass. (a) registered a real hidden flag on `ui`'s own Cobra command and watched the guard
name it exactly (`codegraph ui --zz-throwaway`) while independently confirming the drift gate
cannot see it — the generator/guard pair discriminating exactly as DOCS-06 requires. (b) deleted a
real line from the committed generated file and watched BOTH the drift gate (count-before-compare,
named file, diff excerpt) and the guard (named flag) reject it, while recording the one case
(`--editor-url`) that would NOT have gone RED on the guard's whole-document match. (c) appended a
bogus allowlist entry for a flag that does not exist and watched the rot-enforcement pass reject
it by name, with the accounting counts unchanged — proving the rejection is specific to the rot,
not a side effect of a changed flag count. Every RED transcript above names the exact mutated
item; the guard's counts line (`walked N commands…, inspected M flags…`) printed on every single
run, RED or GREEN, so no run's outcome could be read as a silent or vacuous result. No family
exercises `cobra/doc`'s own generation behaviour — every mutation is to this project's own tree
(`ui.go`), this project's own committed output (`docs/CLI-REFERENCE.md`), or this project's own
allowlist (`cli-reference-allowlist.txt`).

### Byte-clean proof

```
$ git diff --quiet -- internal/cli/ui.go docs/CLI-REFERENCE.md internal/cli/testdata/cli-reference-allowlist.txt; echo $?
0
$ rg -o 'zz-throwaway|zz-bogus' internal/cli/ui.go internal/cli/testdata/cli-reference-allowlist.txt | wc -l | tr -d ' '
0
$ git status --porcelain
?? .planning/phases/12-cli-reference-docs-tail/12-MUTATION-LOG.md
```

No mutation was committed. Every family's mutation was reverted via `git checkout --` and
re-verified clean before the next family began; only this new log file is untracked at the time
of this commit.
