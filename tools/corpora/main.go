// Command corpora is the single resolution path both the Taskfile fetch
// targets and (in a later plan) CI read for the corpora manifest: given
// -mode, it either prints the resolved out-of-tree corpus root (root) or
// a pre-validated JSON array of fetchable entries (entries). Neither mode
// ever emits a value that has not already passed internal/corpora's
// strict allowlist — the Taskfile's bash re-validates independently at
// the interpolation point anyway, but this program's output is never
// itself the unvalidated source of a manifest field.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/seanb4t/codegraph-go/internal/corpora"
)

// manifestPath is the sole pin authority this program reads (D-09),
// relative to the repo root every Taskfile target invokes it from.
const manifestPath = "corpora/manifest.json"

// entryOutput is one entries-mode array element: the fields the bash
// fetch loop needs (repo, sha) plus the two derived, already-computed
// values (slug, dir) so the Taskfile never re-derives a destination path
// on its own.
type entryOutput struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
	Slug string `json:"slug"`
	Dir  string `json:"dir"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main's testable core: it never calls os.Exit itself.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("corpora", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mode := fs.String("mode", "", `output mode: "root" or "entries" (required)`)
	locked := fs.Bool("locked", false, "entries mode: restrict output to locked entries only")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	switch *mode {
	case "root":
		return runRoot(stdout, stderr)
	case "entries":
		return runEntries(stdout, stderr, *locked)
	default:
		fmt.Fprintf(stderr, "tools/corpora: -mode must be \"root\" or \"entries\" (got %q)\n", *mode)
		return 2
	}
}

// runRoot prints the resolved corpus root and nothing else.
func runRoot(stdout, stderr io.Writer) int {
	root, err := corpora.CorpusRoot()
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: resolve corpus root: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, root)
	return 0
}

// runEntries loads and validates the manifest, then prints a JSON array
// of entryOutput — all entries, or only locked ones when lockedOnly is
// set. A validation failure exits non-zero with a message on stderr,
// never a partial or unvalidated listing.
func runEntries(stdout, stderr io.Writer, lockedOnly bool) int {
	m, err := corpora.Load(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: %v\n", err)
		return 1
	}
	root, err := corpora.CorpusRoot()
	if err != nil {
		fmt.Fprintf(stderr, "tools/corpora: resolve corpus root: %v\n", err)
		return 1
	}

	entries := m.Corpora
	if lockedOnly {
		entries = corpora.LockedEntries(m)
	}

	out := make([]entryOutput, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryOutput{
			Repo: e.Repo,
			SHA:  e.SHA,
			Slug: e.Slug(),
			Dir:  e.Dir(root),
		})
	}

	enc := json.NewEncoder(stdout)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(stderr, "tools/corpora: encode entries: %v\n", err)
		return 1
	}
	return 0
}
