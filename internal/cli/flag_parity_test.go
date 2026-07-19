package cli

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagParityDocPath is the relative path from this package to the SURF-05
// audit artifact (docs/FLAG-PARITY.md, D-06) that this drift test keeps
// honest.
const flagParityDocPath = "../../docs/FLAG-PARITY.md"

// TestFlagParityDocCoversRegisteredFlags is the SURF-05 drift guard
// (08-06): it walks the full newRootCmd() command tree — every command
// and subcommand — and for each registered long flag name asserts the
// name appears as a literal ASCII substring somewhere in
// docs/FLAG-PARITY.md. A registered flag with no corresponding row in the
// doc fails this test loudly, so the parity claim REL-04's drop-in gate
// reads can never silently drift out of sync with the real cobra flag
// surface.
//
// Fail-closed (08-06 acceptance criteria): if the doc file itself is
// missing, this test fails loudly rather than silently skipping — a
// missing docs/FLAG-PARITY.md is itself a drift/regression, not an
// "nothing to check" state.
//
// The auto-generated "help" flag is skipped: cobra registers it on every
// command and it carries no TS-parity meaning.
//
// Self-defeat guard (verified manually during 08-06 execution, not
// checked in as a permanent test): temporarily removing a known flag
// name's occurrences from docs/FLAG-PARITY.md and re-running this test
// reproduces a real failure, confirming the assertion below is not
// vacuous.
func TestFlagParityDocCoversRegisteredFlags(t *testing.T) {
	docBytes, err := os.ReadFile(flagParityDocPath)
	if err != nil {
		t.Fatalf("fail-closed: %s must exist and be readable for the flag-parity drift guard to run: %v", flagParityDocPath, err)
	}
	docText := string(docBytes)

	root := newRootCmd()

	var missing []string
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			if !strings.Contains(docText, f.Name) {
				missing = append(missing, cmd.CommandPath()+" --"+f.Name)
			}
		})
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(root)

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%s is missing %d registered flag(s) — drift detected between the real cobra surface and the parity doc:\n%s",
			flagParityDocPath, len(missing), strings.Join(missing, "\n"))
	}
}
