package cli

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// newAffectedCmd builds `codegraph affected [files...]` (QRY-06's
// affected sibling, D-07): the test symbols reachable from the given
// changed files via the D-04 reverse-adjacency map, filtered to
// isTestSymbol matches (query-time derivation, not a persisted
// test-coverage edge — see D-07's documented divergence). --json emits
// this plan's own AffectedResult shape (query.MarshalAffectedJSON) — no
// golden oracle exists for this command (D-07a).
//
// SURF-04/08-05 (CONTEXT D-05): also wires the scripting surface —
// --stdin (union'd with positional args), -d/--depth, -f/--filter
// <glob>, -q/--quiet, -j/--json — so `git diff --name-only | codegraph
// affected --stdin --quiet` works as a clean git-hook/CI pipeline stage.
// Args is relaxed from MinimumNArgs(1) to ArbitraryArgs so a
// zero-positional --stdin invocation isn't rejected by cobra before
// RunE ever runs.
func newAffectedCmd() *cobra.Command {
	var path string
	var jsonOut bool
	var stdinFlag bool
	var depth int
	var filter string
	var quiet bool

	cmd := &cobra.Command{
		Use:   "affected [files...]",
		Short: "List test symbols impacted by changes to the given files",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := collectAffectedFiles(cmd, args, stdinFlag)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				// SURF-04: zero input (no positional args, no/empty stdin) is
				// an advisory, not an error — a git hook piping `git diff
				// --name-only` on a no-op diff must exit 0 cleanly. Handled
				// before query.OpenAt so this path never requires an index.
				if !quiet {
					fmt.Fprintln(cmd.OutOrStdout(), "no files provided (pass files as positional args or use --stdin)")
				}
				return nil
			}

			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			eng, closer, err := query.OpenAt(start)
			if err != nil {
				return err
			}
			defer closer.Close()

			// SURF-04/08-04/08-05: engine's Affected takes a depth-bounded
			// BFS parameter; 0 uses defaultAffectedDepth (5), negative is
			// rejected by the engine's own validateDepth.
			result, err := eng.Affected(files, depth)
			if err != nil {
				return err
			}

			if filter != "" {
				filtered := make([]query.Location, 0, len(result.AffectedTests))
				for _, l := range result.AffectedTests {
					ok, err := filepath.Match(filter, l.FilePath)
					if err != nil {
						return fmt.Errorf("affected: invalid --filter glob %q: %w", filter, err)
					}
					if ok {
						filtered = append(filtered, l)
					}
				}
				result.AffectedTests = filtered
			}

			if jsonOut {
				data, err := query.MarshalAffectedJSON(result)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}

			out := cmd.OutOrStdout()

			if quiet {
				// SURF-04/T-08-05-01/T-08-05-04: plain, unstyled, one path
				// per line — no present.RenderFiles, no WorktreeNotice. Safe
				// to pipe straight into another command.
				//
				// WR-03: a FilePath containing an embedded \n or \r is
				// skipped rather than emitted verbatim — POSIX filesystems
				// permit any byte except NUL and '/' in a filename,
				// including a literal newline, and this output's whole
				// contract is one-path-per-line machine-readable text safe
				// to pipe into another command
				// (`for line in $(codegraph affected --quiet); do ...`). A
				// path with an embedded newline would otherwise inject an
				// attacker-controlled extra "line" into that stream.
				seen := make(map[string]bool, len(result.AffectedTests))
				for _, l := range result.AffectedTests {
					if seen[l.FilePath] {
						continue
					}
					seen[l.FilePath] = true
					if strings.ContainsAny(l.FilePath, "\n\r") {
						continue
					}
					fmt.Fprintf(out, "%s\n", l.FilePath)
				}
				return nil
			}

			// Compact worktree notice (WORK-02, D-12): lives strictly inside
			// the human-output branch, AFTER the --json and --quiet early
			// returns above — see explore.go's call site for the full
			// rationale. WR-04: this command was previously omitted (all 7
			// other read commands had it) with no documented reason — an
			// oversight, now closed.
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			if len(result.AffectedTests) == 0 {
				fmt.Fprintln(out, "no test files affected")
				return nil
			}
			fmt.Fprintf(out, "%d affected test(s):\n", len(result.AffectedTests))
			for _, l := range result.AffectedTests {
				fmt.Fprintf(out, "  %s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
	cmd.Flags().BoolVar(&stdinFlag, "stdin", false, "read changed file paths from stdin, one per line (union'd with positional args)")
	cmd.Flags().IntVarP(&depth, "depth", "d", 0, "BFS depth (default 5, max 50)")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "glob to narrow affected test paths (filepath.Match syntax)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "emit only affected test file paths, one per line (no summary, no worktree notice)")

	return cmd
}

// affectedStdinMaxLineBytes bounds a single --stdin line (WR-06): a file
// path legitimately never needs to exceed this — 4096 is the common
// PATH_MAX — so a single pathological/malicious "line" far longer than
// any real path is treated as malformed input (an explicit error) rather
// than tripping bufio.Scanner's much smaller default 64KB
// bufio.MaxScanTokenSize ceiling and silently truncating every line after
// it.
const affectedStdinMaxLineBytes = 4096

// collectAffectedFiles assembles the file list from positional args plus,
// when stdinFlag is set, newline-delimited paths read from cmd.InOrStdin()
// via bufio.NewScanner — never hangs on a piped/closed/empty stream since
// Scan() returns false on EOF (T-08-05-03). Each stdin line is
// whitespace-trimmed (also strips a trailing \r from CRLF input) and blank
// lines are skipped. Positional args and stdin paths are unioned,
// deduplicated, and order-preserved; stdin lines are used purely as
// index-lookup keys (T-08-05-01) — never shell-exec'd or path-joined here.
//
// CR-01: the combined file count is capped at query.MaxAffectedFiles —
// checked as each stdin line is added — so a hostile or merely oversized
// input source (`yes | codegraph affected --stdin`, a compromised git
// hook, an attacker-influenced CI diff) cannot grow the seen/files sets
// (and, downstream, Engine.Affected's fileSet) without limit before the
// engine's own bounded BFS ever runs. This rejects outright with a clear
// error, matching validateMaxFiles/validateLimit's "reject absurd input,
// don't silently truncate" convention elsewhere in this package.
//
// WR-06: scanner.Buffer raises the per-line token ceiling to
// affectedStdinMaxLineBytes (well above any real path, well below
// unbounded) so a legitimately long-but-valid path is never mistaken for
// "too long" — and a genuine bufio.ErrTooLong (which, unlike ordinary
// EOF, indicates malformed input) is surfaced as an explicit error
// instead of silently stopping the scan and dropping every line after it.
func collectAffectedFiles(cmd *cobra.Command, args []string, stdinFlag bool) ([]string, error) {
	seen := make(map[string]bool, len(args))
	files := make([]string, 0, len(args))

	add := func(f string) error {
		if f == "" || seen[f] {
			return nil
		}
		if err := query.ValidateAffectedFiles(len(files) + 1); err != nil {
			return fmt.Errorf("affected: %w", err)
		}
		seen[f] = true
		files = append(files, f)
		return nil
	}

	for _, a := range args {
		if err := add(a); err != nil {
			return nil, err
		}
	}

	if stdinFlag {
		scanner := bufio.NewScanner(cmd.InOrStdin())
		// Initial buffer capacity must not exceed affectedStdinMaxLineBytes:
		// bufio.Scanner.Buffer's effective ceiling is max(maxArg, cap(buf)),
		// so a larger initial cap (e.g. the old 64*1024) silently defeats
		// the intended 4096-byte cap.
		scanner.Buffer(make([]byte, 0, affectedStdinMaxLineBytes), affectedStdinMaxLineBytes)
		for scanner.Scan() {
			if err := add(strings.TrimSpace(scanner.Text())); err != nil {
				return nil, err
			}
		}
		if err := scanner.Err(); err != nil {
			// WR-06: bufio.ErrTooLong specifically indicates a malformed
			// (pathologically long) line — surface it rather than
			// swallowing it. Other scanner errors (rare: genuine I/O
			// errors on the underlying reader, not "no more input") are
			// still deliberately swallowed — a git-hook pipeline should
			// degrade to "no files from stdin" rather than fail the
			// whole command; the empty-input path above still resolves
			// the same "no files provided" advisory.
			if errors.Is(err, bufio.ErrTooLong) {
				return nil, fmt.Errorf("affected: --stdin line exceeds maximum %d bytes: %w", affectedStdinMaxLineBytes, err)
			}
		}
	}

	return files, nil
}
