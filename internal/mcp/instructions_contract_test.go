package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// readmePath is the repository README, reached from internal/mcp. Mirrors
// internal/cli/flag_parity_test.go's flagParityDocPath convention: a
// relative path read directly, with a missing file treated as a loud
// failure rather than a skip — a missing README is itself a regression,
// not a "nothing to check" state.
const readmePath = "../../README.md"

// allowlistEnvName is the operator allowlist env var by name. It is
// deliberately re-typed here rather than imported from internal/cli
// (which would be an import cycle: internal/cli imports internal/mcp).
// This is the ONE literal in this file; every tool name below is derived
// from companionNames, this package's own source of truth.
const allowlistEnvName = "CODEGRAPH_MCP_TOOLS"

// instructionsMaxBytes and the no-newline rule are the constraints
// server.go's own doc comment places on the instructions constant:
// "Kept to a single paragraph with no newline characters and roughly 600
// bytes or fewer, since the value is JSON-encoded into every one of those
// transcripts and an embedded newline would become an escape sequence that
// makes each such diff harder to read." Nothing enforced them before this
// test, which matters now that the string must grow to carry the
// allowlist gate — a fix for one wire-contract defect must not silently
// introduce another.
const instructionsMaxBytes = 600

// resourcesAnchor is WIRE-03's resources half — the literal substring the
// rewritten instructions const must carry so a client is pointed at
// resources/list for tool-by-tool reference docs (D-03: generic phrasing,
// no codegraph:// URI enumerated inside the wire-budget-constrained
// const).
const resourcesAnchor = "resources/list"

// TestInstructionsNamesTheNarrowingFilter pins the wire contract against
// the behavior it describes. The instructions constant ships to every MCP
// client on every session, in both the classic initialize result and the
// server/discover result. Before this test it attributed tool visibility
// SOLELY to index state ("An empty tool list means this repository has no
// index yet... Tools appear automatically once an index exists") and never
// mentioned CODEGRAPH_MCP_TOOLS at all. An agent looking at a smaller tool
// list than it expected was therefore given exactly one explanation, and
// under the contract of the day that explanation was the one cause that
// could not produce it (MCP-03 makes a missing index produce ZERO tools,
// never one).
//
// The default has since inverted — all eight tools now register when the
// variable is unset — but this assertion did not become obsolete, it
// changed which reader it protects. It used to protect the user who saw too
// FEW tools and had no way to learn how to get more; it now protects the
// user who set the filter, forgot, and sees fewer than the eight everything
// else promises. Either way the string must name the mechanism.
//
// This is the second occurrence of the bug class
// TestMCPToolSchemaNumericClaimsMatchEngineConstants was written for
// (SURF-01: tools.go advertised "default 5" for a whole phase after the
// engine constant became 2). That test compares numeric claims in per-tool
// DESCRIPTIONS against engine constants; it structurally cannot see the
// server-level instructions string, which is where the visibility contract
// lives.
func TestInstructionsNamesTheNarrowingFilter(t *testing.T) {
	if !strings.Contains(instructions, allowlistEnvName) {
		t.Fatalf("instructions never names %s, so an agent seeing fewer than the full tool surface has no way to learn that a narrowing filter exists or that unsetting it restores all %d tools. instructions = %q",
			allowlistEnvName, len(companionNames)+1, instructions)
	}
}

// TestInstructionsDescribesEveryVisibilityMechanism is the gate that was
// actually missing. Naming CODEGRAPH_MCP_TOOLS is necessary but not
// sufficient: the original defect was a string that described ONE of the
// mechanisms governing tool visibility, confidently, while two others went
// unmentioned — and a reader given one complete-sounding explanation does
// not go looking for a second.
//
// Three mechanisms decide what a client sees, and the string must reach all
// three. Each is anchored on a token that cannot survive the deletion of
// the clause carrying it:
//
//   - the default surface        -> "default"
//   - the narrowing filter       -> CODEGRAPH_MCP_TOOLS
//   - the index gate (MCP-03)    -> "codegraph init"
//
// This is a weaker oracle than the derived one below — the anchors are
// literals, not values read from the code under test — and it is recorded
// as such rather than dressed up. There is no runtime value to derive
// "explains its own default" from; the honest alternative to three literal
// anchors is no gate at all, which is the state that produced this bug.
func TestInstructionsDescribesEveryVisibilityMechanism(t *testing.T) {
	mechanisms := []struct {
		mechanism string
		anchor    string
	}{
		{"the default tool surface", "default"},
		{"the CODEGRAPH_MCP_TOOLS narrowing filter", allowlistEnvName},
		{"the missing-index remedy (MCP-03)", "codegraph init"},
		{"the resources reference surface", resourcesAnchor},
	}

	for _, m := range mechanisms {
		if !strings.Contains(instructions, m.anchor) {
			t.Errorf("instructions never mentions %s (no %q anywhere in it). A client reading this string is given a complete-sounding account of tool visibility with one mechanism missing — the exact shape of the defect this test exists to prevent. instructions = %q",
				m.mechanism, m.anchor, instructions)
		}
	}
}

// TestInstructionsStaysWithinWireBudget enforces server.go's stated
// constraints on the instructions constant. These are boundary neighbors
// for the fix above, not incidental style checks: the string is
// JSON-encoded verbatim into 24 frozen wire-oracle transcripts, so a
// newline becomes an escape sequence in every one of them and unbounded
// growth inflates every transcript diff forever.
func TestInstructionsStaysWithinWireBudget(t *testing.T) {
	if strings.ContainsAny(instructions, "\n\r") {
		t.Errorf("instructions contains a newline; it must stay a single paragraph (it is JSON-encoded into every frozen transcript, where a newline becomes an escape sequence)")
	}
	if got := len(instructions); got > instructionsMaxBytes {
		t.Errorf("instructions is %d bytes, over the %d-byte budget server.go's doc comment sets", got, instructionsMaxBytes)
	}
	if strings.TrimSpace(instructions) == "" {
		t.Errorf("instructions is empty; every client would receive no guidance at all")
	}
}

// resourcesClaimResolves is WIRE-03's resources-half checker: it proves the
// claim carried by resourcesAnchor is not merely present in claim, but
// actually resolves against a live capability — at least one resource is
// advertised, and reading the first advertised URI returns non-empty
// content. Every input is a parameter (never package state, never a live
// session opened internally) specifically so Task 3's non-vacuity table
// test can drive it with synthetic inputs. There is no t.Skip branch
// anywhere in this checker or its caller — absence of the claim is a
// failure, not a skip (the exact vacuous-pass hole this guard exists to
// close).
func resourcesClaimResolves(claim string, uris []string, read func(string) ([]byte, error)) error {
	if !strings.Contains(claim, resourcesAnchor) {
		return fmt.Errorf("claim %q never mentions %q, so a client reading it has no way to learn resources/list exists", claim, resourcesAnchor)
	}
	if len(uris) == 0 {
		return fmt.Errorf("claim %q names %q, but resources/list advertised zero resources — the claim does not resolve", claim, resourcesAnchor)
	}
	content, err := read(uris[0])
	if err != nil {
		return fmt.Errorf("claim %q names %q, but reading advertised URI %q failed: %w", claim, resourcesAnchor, uris[0], err)
	}
	if len(content) == 0 {
		return fmt.Errorf("claim %q names %q, but reading advertised URI %q returned empty content", claim, resourcesAnchor, uris[0])
	}
	return nil
}

// TestInstructionsResourcesClaimIsResolvable is WIRE-03's resources-half
// live proof: it builds a server over an indexed fixture, opens a session
// via newTestSession, and drives resourcesClaimResolves against a REAL
// ListResources/ReadResource round-trip — never a hardcoded URI list (see
// resources_test.go's identical pattern, this package's own precedent).
func TestInstructionsResourcesClaimIsResolvable(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false)
	s := BuildServer(true, companions, dir, dir)

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	listRes, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	uris := make([]string, len(listRes.Resources))
	for i, r := range listRes.Resources {
		uris[i] = r.URI
	}

	read := func(uri string) ([]byte, error) {
		res, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
		if err != nil {
			return nil, err
		}
		if len(res.Contents) == 0 {
			return nil, fmt.Errorf("ReadResource(%q) returned no Contents elements", uri)
		}
		return []byte(res.Contents[0].Text), nil
	}

	if err := resourcesClaimResolves(instructions, uris, read); err != nil {
		t.Fatal(err)
	}
}

// TestInstructionsCarriesNoWireContractViolation is T-08-01's mitigation:
// the instructions const must stay pure ASCII (so len() and rune count
// agree, per the WIRE-01/encoding edge resolution) and must never carry an
// absolute-path token — const's own compile-time-literal nature already
// makes runtime interpolation a compile error, but a future const->var
// downgrade that admits a host path would otherwise ship silently into 38
// committed wire-oracle transcripts.
func TestInstructionsCarriesNoWireContractViolation(t *testing.T) {
	for i := 0; i < len(instructions); i++ {
		if instructions[i] >= 0x80 {
			t.Fatalf("instructions byte %d is 0x%02x, non-ASCII (>= 0x80); the WIRE-01/encoding edge requires pure ASCII so len() (bytes) and rune count always agree. instructions = %q", i, instructions[i], instructions)
		}
	}
	for _, token := range []string{"/Users/", "/home/", "/private/", `C:\`} {
		if strings.Contains(instructions, token) {
			t.Fatalf("instructions contains %q, an absolute-path token that would publish the capturing host's filesystem layout into every committed wire-oracle transcript (T-03-19). instructions = %q", token, instructions)
		}
	}
}

// TestInstructionsReachesTheWireVerbatim is the end-to-end proof that the
// instructions const actually reaches a real client rather than merely
// existing in the package: it opens a session via newTestSession and
// asserts session.InitializeResult().Instructions equals the const
// byte-for-byte.
func TestInstructionsReachesTheWireVerbatim(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false)
	s := BuildServer(true, companions, dir, dir)

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	got := session.InitializeResult().Instructions
	if got != instructions {
		t.Fatalf("session.InitializeResult().Instructions = %q, want the instructions const verbatim %q", got, instructions)
	}
}

// docNamesCompanionsWithoutTheFilter is the checker
// TestREADMEDocumentsToolVisibilityGate applies to the real README and
// TestREADMEGateCheckerIsNotVacuous applies to synthetic inputs. Extracted
// so the assertion itself is provably non-vacuous — the self-defeat
// property internal/cli's flag-parity test could only verify by hand.
//
// The rule: a document may not enumerate the companion tools without also
// naming the variable that can take them away. Naming codegraph_explore
// alone is always fine — it is the one tool no filter can remove, so a
// document mentioning only it makes no claim the filter can falsify.
//
// The rule text is unchanged across the CODEGRAPH_MCP_TOOLS inversion; its
// justification is not. It used to mean "these seven are not there unless
// you opt in" — a document naming them was overpromising. It now means
// "these seven are there unless you opted out" — a document naming them
// leaves a reader who DID opt out with no account of why their list is
// shorter. Both are the same failure: a document describing the tool
// surface as though only one configuration of it existed.
func docNamesCompanionsWithoutTheFilter(doc string) error {
	var named []string
	for _, name := range companionNames {
		if strings.Contains(doc, "codegraph_"+name) {
			named = append(named, "codegraph_"+name)
		}
	}
	if len(named) == 0 {
		return nil
	}
	if strings.Contains(doc, allowlistEnvName) {
		return nil
	}
	return fmt.Errorf("names %d filterable companion tool(s) %v but never names %s, so the document describes one configuration of the tool surface as though it were the only one",
		len(named), named, allowlistEnvName)
}

// TestREADMEDocumentsToolVisibilityGate pins the user-facing half of the
// same contract. README.md stated "Eight tools are exposed" and listed all
// eight by name, directly beneath the `codegraph install` snippet, while
// never mentioning CODEGRAPH_MCP_TOOLS anywhere in the file — a claim that
// was false for every default installation, and the origin of the reported
// "the mcp server is only showing one tool" bug.
//
// That sentence is now TRUE: the default surface really is all eight. This
// test survives the change because being true was never sufficient — the
// README still has to account for the configuration in which it is not,
// which is the half a reader with the filter set needs and the half that
// was missing.
//
// The companion names are read from companionNames, this package's own
// source of truth, rather than re-typed: adding an eighth companion tool
// automatically extends this test's reach.
func TestREADMEDocumentsToolVisibilityGate(t *testing.T) {
	doc, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read %s: %v (a missing README is itself a regression, not a skip)", readmePath, err)
	}
	if err := docNamesCompanionsWithoutTheFilter(string(doc)); err != nil {
		t.Fatalf("%s %v", readmePath, err)
	}
}

// TestREADMEGateCheckerIsNotVacuous proves the checker above actually
// discriminates, including at the boundaries of its equivalence classes:
// zero companions named (the vacuous-pass case an always-fail checker
// would get wrong), exactly one companion named (the off-by-one neighbor
// of zero), all companions named, and the filter named in each case.
// Without this, a checker that returned nil unconditionally would pass
// TestREADMEDocumentsToolVisibilityGate forever and prove nothing.
func TestREADMEGateCheckerIsNotVacuous(t *testing.T) {
	oneCompanion := "codegraph_" + companionNames[0]
	allCompanions := make([]string, 0, len(companionNames))
	for _, name := range companionNames {
		allCompanions = append(allCompanions, "codegraph_"+name)
	}
	allNamed := strings.Join(allCompanions, ", ")

	cases := []struct {
		name    string
		doc     string
		wantErr bool
	}{
		{"no tools named at all", "codegraph indexes your repo.", false},
		{"only the unfilterable tool", "Use codegraph_explore.", false},
		{"one companion, no filter named", "Use " + oneCompanion + ".", true},
		{"one companion, filter named", "Use " + oneCompanion + "; narrow with " + allowlistEnvName + ".", false},
		{"all companions, no filter named", "Eight tools are exposed: codegraph_explore, " + allNamed + ".", true},
		{"all companions, filter named", "Narrow " + allNamed + " with " + allowlistEnvName + ".", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := docNamesCompanionsWithoutTheFilter(tc.doc)
			if tc.wantErr && err == nil {
				t.Fatalf("docNamesCompanionsWithoutTheFilter(%q) = nil, want an error", tc.doc)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("docNamesCompanionsWithoutTheFilter(%q) = %v, want nil", tc.doc, err)
			}
		})
	}
}
