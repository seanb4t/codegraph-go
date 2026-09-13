package cli

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// cliReferenceDocPath is the relative path from this package to DOCS-05's
// generated CLI reference (docs/CLI-REFERENCE.md), read by go test's
// package-relative working directory.
const cliReferenceDocPath = "../../docs/CLI-REFERENCE.md"

// cliReferenceAllowlistPath is the relative path to DOCS-06's committed,
// reason-carrying allowlist for flags the generated reference cannot show
// by design (hidden commands, hidden flags, deprecated flags).
const cliReferenceAllowlistPath = "testdata/cli-reference-allowlist.txt"

// cliReferenceMinCommands and cliReferenceMinFlags are D-09's positive
// floors (rule 84d1gfpywd): the real tree today walks 36 commands and 115
// flags (root + 24 visible top-level + daemon start|stop + githooks
// install|remove|status + completion + bash|zsh|fish|powershell, plus one
// --help per command and root's --version). These floors exist so a walk
// that silently finds nothing — an incomplete tree, a broken recursion —
// can never read as a passing test.
const (
	cliReferenceMinCommands = 26
	cliReferenceMinFlags    = 50
)

// cliReferenceTree builds the identical command tree tools/clidoc/main.go
// builds: the root command completed with the completion family and the
// -v/--version flag. This MUST stay in lockstep with tools/clidoc/main.go:
// any Init* call added to one is added to the other in the same commit,
// or the reference branch below stops being sound — it is only correct
// because both walks see the same command universe.
func cliReferenceTree() *cobra.Command {
	root := NewRootCmd()
	root.InitDefaultCompletionCmd()
	root.InitDefaultVersionFlag()
	return root
}

// documentedByReference reports whether cmd — and every ancestor up to
// root — is a command tools/clidoc's generator would descend into and
// render (IsAvailableCommand() && !IsAdditionalHelpTopicCommand(), the
// same predicate the generator applies at every level of its own walk).
// The generator never descends into an undocumented command, so a hidden
// command's children are undocumented too; this predicate must agree
// with that exactly, or the reference branch below is unsound.
func documentedByReference(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			return false
		}
	}
	return true
}

// inheritedFromAncestor reports whether f was declared by one of cmd's
// ancestors' PersistentFlags() and folded into cmd.Flags() as a side
// effect of InitDefaultHelpFlag()'s internal mergePersistentFlags() call
// — the LocalFlags pointer-identity idiom (cobra command.go). Such a flag
// belongs to its declaring ancestor only and must never be re-counted at
// a descendant.
func inheritedFromAncestor(cmd *cobra.Command, f *pflag.Flag) bool {
	for anc := cmd.Parent(); anc != nil; anc = anc.Parent() {
		if anc.PersistentFlags().Lookup(f.Name) == f {
			return true
		}
	}
	return false
}

// docMentionsFlag reports whether "--name" appears at a word boundary
// anywhere in doc. This is a whole-document match, deliberately NOT
// scoped to the command's own section (D-09) — task docs:cli:drift
// already guarantees the doc's content is exactly the generator's
// output. The boundary check keeps "--no" from passing on the strength
// of "--no-descriptions".
func docMentionsFlag(doc string, name string) bool {
	pattern := `(^|[^A-Za-z0-9_-])--` + regexp.QuoteMeta(name) + `([^A-Za-z0-9_-]|$)`
	return regexp.MustCompile(pattern).MatchString(doc)
}

// parseCLIReferenceAllowlist parses testdata/cli-reference-allowlist.txt:
// one entry per non-blank, non-"#" line, "<key><TAB><reason>", where key
// is "<command path>" or "<command path> --<flag>". A line missing a tab
// or carrying an empty reason is an error (fail closed — a reason is
// mandatory, D-08); a duplicate key is an error.
func parseCLIReferenceAllowlist(data []byte) (map[string]string, error) {
	entries := make(map[string]string)
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, reason, ok := strings.Cut(trimmed, "\t")
		if !ok {
			return nil, fmt.Errorf("line %d: entry %q has no tab-separated reason", i+1, trimmed)
		}
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("line %d: entry %q has an empty reason", i+1, key)
		}
		if _, dup := entries[key]; dup {
			return nil, fmt.Errorf("line %d: duplicate entry %q", i+1, key)
		}
		entries[key] = reason
	}
	return entries, nil
}

// ineligibleReason names why a non-eligible flag must be covered by the
// allowlist rather than the generated reference — used only in the
// unaccounted-flag failure message, never in the accounting logic.
func ineligibleReason(cmd *cobra.Command, f *pflag.Flag) string {
	switch {
	case !documentedByReference(cmd):
		return "flag on a hidden command"
	case f.Hidden:
		return "hidden flag"
	default:
		return "deprecated flag"
	}
}

// TestEveryRegisteredFlagIsAccountedFor is DOCS-06's guard for the one
// thing a generated reference cannot show: hidden, deprecated, and
// hidden-command flags vanish from cobra/doc output by design (D-05), so
// every registered flag must be accounted for either by the reference
// (D-01) or by a committed, reason-carrying allowlist entry (D-08). It
// tests only this project's artefacts, never cobra/doc's own rendering
// behaviour (D-09): it does not scope-match a flag to its own doc
// section, because task docs:cli:drift already guarantees the doc's
// content is exactly the generator's output.
func TestEveryRegisteredFlagIsAccountedFor(t *testing.T) {
	docBytes, err := os.ReadFile(cliReferenceDocPath)
	if err != nil {
		t.Fatalf("fail-closed: %s must exist and be readable: %v", cliReferenceDocPath, err)
	}
	if len(docBytes) == 0 {
		t.Fatalf("fail-closed: %s must exist and be readable: file is empty", cliReferenceDocPath)
	}
	docText := string(docBytes)

	allowBytes, err := os.ReadFile(cliReferenceAllowlistPath)
	if err != nil {
		t.Fatalf("fail-closed: %s must exist and be readable: %v", cliReferenceAllowlistPath, err)
	}

	allow, err := parseCLIReferenceAllowlist(allowBytes)
	if err != nil {
		t.Fatalf("fail-closed: %s is malformed: %v", cliReferenceAllowlistPath, err)
	}
	used := make(map[string]bool, len(allow))

	root := cliReferenceTree()

	var (
		cmdCount, flagCount, viaReference, viaAllowlist int
		unaccounted                                     []string
	)
	seen := make(map[string]bool)

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		cmd.InitDefaultHelpFlag() // mirrors doc.GenMarkdownCustom exactly (md_docs.go)
		cmdCount++

		visit := func(f *pflag.Flag) {
			key := cmd.CommandPath() + " --" + f.Name
			if seen[key] {
				return
			}
			seen[key] = true
			flagCount++

			eligible := documentedByReference(cmd) && !f.Hidden && f.Deprecated == ""
			if eligible {
				if docMentionsFlag(docText, f.Name) {
					viaReference++
				} else {
					unaccounted = append(unaccounted, fmt.Sprintf(
						"unaccounted flag: %s (visible flag on a documented command — missing from docs/CLI-REFERENCE.md; run task docs:cli)", key))
				}
				return
			}

			allowKeyFlag := key
			allowKeyCmd := cmd.CommandPath()
			switch {
			case hasAllowEntry(allow, allowKeyFlag):
				used[allowKeyFlag] = true
				viaAllowlist++
			case hasAllowEntry(allow, allowKeyCmd):
				used[allowKeyCmd] = true
				viaAllowlist++
			default:
				unaccounted = append(unaccounted, fmt.Sprintf(
					"unaccounted flag: %s (%s — add an allowlist entry with a reason)", key, ineligibleReason(cmd, f)))
			}
		}

		// D-07/Pitfall 5: never use NonInheritedFlags()/InheritedFlags() for
		// accounting — visit cmd.Flags() (local, post-merge) and
		// cmd.PersistentFlags() (this command's own) directly, skipping any
		// flag inheritedFromAncestor so it is attributed once, to its
		// declaring command only.
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if inheritedFromAncestor(cmd, f) {
				return
			}
			visit(f)
		})
		cmd.PersistentFlags().VisitAll(visit)

		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(root)

	var stale []string
	for key := range allow {
		if !used[key] {
			stale = append(stale, fmt.Sprintf("stale allowlist entry: %s (matches no registered command or flag)", key))
		}
	}

	t.Logf("walked %d commands (hidden included), inspected %d flags: %d accepted via %s, %d accepted via %s",
		cmdCount, flagCount, viaReference, cliReferenceDocPath, viaAllowlist, cliReferenceAllowlistPath)

	if cmdCount < cliReferenceMinCommands {
		t.Fatalf("walked only %d commands — expected at least %d (D-09 floor)", cmdCount, cliReferenceMinCommands)
	}
	if flagCount < cliReferenceMinFlags {
		t.Fatalf("inspected only %d flags — expected at least %d (D-09 floor)", flagCount, cliReferenceMinFlags)
	}
	if viaReference+viaAllowlist+len(unaccounted) != flagCount {
		t.Fatalf("accounting mismatch: %d (reference) + %d (allowlist) + %d (unaccounted) != %d (inspected) — every inspected flag must land in exactly one bucket",
			viaReference, viaAllowlist, len(unaccounted), flagCount)
	}

	if len(unaccounted) > 0 || len(stale) > 0 {
		sort.Strings(unaccounted)
		sort.Strings(stale)
		problems := make([]string, 0, len(unaccounted)+len(stale))
		problems = append(problems, unaccounted...)
		problems = append(problems, stale...)
		t.Fatalf("%d problem(s):\n%s", len(problems), strings.Join(problems, "\n"))
	}
}

// hasAllowEntry reports whether key is a registered allowlist entry.
func hasAllowEntry(allow map[string]string, key string) bool {
	_, ok := allow[key]
	return ok
}
