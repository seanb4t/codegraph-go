// Command transcriptfreeze implements CONTEXT.md D-03's anti-regeneration
// cross-change rule as a CI-invokable guard: a pull request must never both
// change a frozen wire-oracle transcript AND change something that could
// have caused the diff (internal/mcp/*.go, or the MCP dependency line in
// go.mod). This file holds the pure, I/O-free classifier; main.go is the CI
// entrypoint that reads the merge-base diff and drives it.
//
// The rule is deliberately narrow. Touching only a frozen transcript, or
// only internal/mcp, or only go.mod's MCP dependency line, is NOT a
// violation — regenerating transcripts in their own pull request is the
// sanctioned Phase 3 path. See Classify's doc comment and buildReason for
// the "floor, not a proof of innocence" disclosure this narrowness requires.
//
// One shape of the rule's collision is exempted rather than forbidden:
// 02-CONTEXT.md's D-01/D-02/D-03 supersede D-03's original "Phase 2 =
// frozen, no exceptions" clause with a single self-expiring waiver for
// SDK-01's one-time mark3labs-to-go-sdk transition. See sdkSwapExemption
// and buildSwapExemptionNotice for that narrow exception.
//
// As of 03-02: a detected collision is advisory, not blocking. The guard's
// own "split into two pull requests" remedy is structurally unavailable in
// this repository (03-CONTEXT.md D-06 — a server-change-only PR fails the
// required wire-oracle leg, and a transcripts-only PR has no mergeable
// base), so main.go's run() reports Verdict.Reason in full but no longer
// fails the build on it. This does not change Classify's predicate or
// Verdict.Violation's meaning — see buildReason for what did change.
package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// transcriptDirPrefix is the directory under which every frozen wire-oracle
// transcript lives. A changed-file path with this prefix is what this guard
// treats as "a frozen transcript changed."
const transcriptDirPrefix = "testdata/wireoracle/transcripts/"

// serverDirPrefix is the directory whose Go source shapes what the wire
// oracle observes over stdio. A changed-file path with this prefix is what
// this guard treats as "the server that produced the transcript changed."
const serverDirPrefix = "internal/mcp/"

// mcpSDKModulePrefixes names the MCP SDK module paths whose go.mod diff
// line counts as "the MCP dependency changed." The second entry is
// deliberately forward-declared for Phase 2's SDK swap, exactly as the
// SDK-02 archtest's own list is — a guard naming only today's dependency
// would go vacuous at the moment the swap makes it matter most.
var mcpSDKModulePrefixes = []string{
	"github.com/mark3labs/mcp-go",
	"github.com/modelcontextprotocol/go-sdk",
}

// ErrNoInput is the sentinel main.go wraps when an input FILE could not be
// read at all (missing, permission error, etc.) — the "unusable input"
// case. It is never returned for a successfully-read, merely empty or
// malformed, changed-file list; ParseChangedList reports those two cases
// through its own return values instead (see its doc comment).
var ErrNoInput = errors.New("transcriptfreeze: input could not be read")

// diffStatusRe matches a git diff --name-status status field: a single
// status letter optionally followed by a similarity-percentage number (as
// in "R100" or "C87"). A status field that does not match this shape is
// treated as unparseable input, not silently accepted.
var diffStatusRe = regexp.MustCompile(`^[ACDMRTUXB][0-9]*$`)

// Verdict is Classify's result: whether the changed set forms D-03's
// forbidden shape, which paths and dependency change contributed, and a
// human-readable Reason naming both sides of the collision for a CI failure
// message.
type Verdict struct {
	Violation         bool
	Transcripts       []string
	ServerChanges     []string
	DependencyChanged bool
	SwapExemption     bool
	Reason            string
}

// isTranscriptPath reports whether path names a frozen wire-oracle
// transcript: it lives under transcriptDirPrefix and ends in ".golden".
func isTranscriptPath(path string) bool {
	return strings.HasPrefix(path, transcriptDirPrefix) && strings.HasSuffix(path, ".golden")
}

// isServerChangePath reports whether path names Go source under
// serverDirPrefix that could have shaped a transcript's bytes.
func isServerChangePath(path string) bool {
	return strings.HasPrefix(path, serverDirPrefix) && strings.HasSuffix(path, ".go")
}

// ParseChangedList parses `git diff --name-status -M -C` output: it splits
// raw on newlines, drops blank lines, and for each remaining record splits
// on tabs. A record whose status begins with R or C (a rename or copy)
// carries TWO paths and contributes both to the result — `--name-status`
// with -M/-C reports both sides of a rename record, which is what makes a
// renamed transcript (or a renamed transcripts directory) still detectable.
// Every other record contributes its single path.
//
// Input that is empty, or contains only blank lines, is a CLEAN diff — it
// returns an empty, non-nil slice and a nil error. That is a normal result,
// not a failure: a pull request whose merge-base diff touches nothing must
// never be misreported as unusable. A record that is non-empty but
// unparseable (no tab, or a status field that is not a recognized git
// diff --name-status status letter) returns a non-nil error naming the
// offending line, so malformed input still fails loudly rather than being
// silently dropped.
//
// ParseChangedList never returns ErrNoInput — that sentinel is reserved for
// main.go's own file-read failure, a categorically different case ("the
// caller could not read the input at all") from anything ParseChangedList
// can observe once it has been handed a string.
func ParseChangedList(raw string) ([]string, error) {
	paths := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			return nil, fmt.Errorf("transcriptfreeze: unparseable diff record (no tab): %q", line)
		}
		status := fields[0]
		if !diffStatusRe.MatchString(status) {
			return nil, fmt.Errorf("transcriptfreeze: unparseable diff record (unrecognized status %q): %q", status, line)
		}
		if status[0] == 'R' || status[0] == 'C' {
			if len(fields) < 3 {
				return nil, fmt.Errorf("transcriptfreeze: rename/copy record missing its second path: %q", line)
			}
			paths = append(paths, fields[1], fields[2])
			continue
		}
		paths = append(paths, fields[1])
	}
	return paths, nil
}

// MCPDependencyTouched reports whether goModDiff's added or removed lines
// (lines beginning with a single "+" or "-", excluding the "+++"/"---" file
// headers) name one of mcpSDKModulePrefixes. A context line that merely
// mentions the dependency (no leading +/-) does not trigger it — only an
// actual add or remove counts as "touched."
func MCPDependencyTouched(goModDiff string) bool {
	for _, line := range strings.Split(goModDiff, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
			continue
		}
		for _, prefix := range mcpSDKModulePrefixes {
			if strings.Contains(line, prefix) {
				return true
			}
		}
	}
	return false
}

// sdkSwapExemption reports whether goModDiff shows SDK-01's one-time
// mark3labs-to-go-sdk transition: some line begins with a single "-"
// (excluding the "---" file header) and contains
// mcpSDKModulePrefixes[0] (github.com/mark3labs/mcp-go), AND some line
// begins with a single "+" (excluding "+++") and contains
// mcpSDKModulePrefixes[1] (github.com/modelcontextprotocol/go-sdk). Reusing
// mcpSDKModulePrefixes' own entries by index — rather than retyping the
// module paths — guarantees this predicate can never name a module
// MCPDependencyTouched itself does not already consider an MCP SDK.
//
// This is a diff SHAPE, not a runtime switch: it reads no environment
// variable, CLI switch, or author-addable file — the go.mod diff string is
// its only input. Once github.com/mark3labs/mcp-go is absent from go.mod,
// no future diff can reproduce a removal line naming it, so this predicate
// is self-expiring by construction — it can fire at most once in this
// repository's history.
func sdkSwapExemption(goModDiff string) bool {
	removedMark3labs := false
	addedGoSDK := false
	for _, line := range strings.Split(goModDiff, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "-") && strings.Contains(line, mcpSDKModulePrefixes[0]):
			removedMark3labs = true
		case strings.HasPrefix(line, "+") && strings.Contains(line, mcpSDKModulePrefixes[1]):
			addedGoSDK = true
		}
	}
	return removedMark3labs && addedGoSDK
}

// Classify collects transcript paths and server-change paths from changed,
// computes MCPDependencyTouched(goModDiff), and reports Violation true if
// and only if at least one transcript path is present AND (at least one
// server-change path is present OR the MCP dependency was touched) AND the
// collision is not the one shape sdkSwapExemption waives — SDK-01's
// one-time mark3labs-to-go-sdk transition (02-CONTEXT.md D-01/D-02/D-03).
// When the exemption fires, Violation stays false and SwapExemption records
// that it did, with Reason carrying the exemption notice rather than a
// failure message.
//
// This is D-03's narrow, deliberately-locked trigger set, not a blanket ban
// on golden changes: transcript-only and internal/mcp-only changes are both
// clean. See buildReason for the floor disclosure a violation's Reason
// carries, and buildSwapExemptionNotice for the exemption's own notice.
func Classify(changed []string, goModDiff string) Verdict {
	var transcripts, serverChanges []string
	for _, path := range changed {
		if isTranscriptPath(path) {
			transcripts = append(transcripts, path)
		}
		if isServerChangePath(path) {
			serverChanges = append(serverChanges, path)
		}
	}
	depTouched := MCPDependencyTouched(goModDiff)

	v := Verdict{
		Transcripts:       transcripts,
		ServerChanges:     serverChanges,
		DependencyChanged: depTouched,
	}
	if len(transcripts) > 0 && (len(serverChanges) > 0 || depTouched) {
		if sdkSwapExemption(goModDiff) {
			v.SwapExemption = true
			v.Reason = buildSwapExemptionNotice(transcripts, serverChanges, depTouched)
			return v
		}
		v.Violation = true
		v.Reason = buildReason(transcripts, serverChanges, depTouched)
	}
	return v
}

// buildReason names both offending sides of a D-03 collision in full,
// names the control that now applies (advisory since 03-02 — see this
// file's package doc comment), and states — per this phase's scoping doc
// (docs/MCP-2026-07-28-SCOPING.md, "The wire-oracle anti-regeneration
// trigger set is a floor, not a proof of innocence") — that this trigger
// set is a deliberate floor, not proof the regeneration was safe:
// transcript bytes also legitimately depend on internal/query and
// internal/indexer, which this guard does not watch. Do NOT widen the
// trigger set to compensate — CONTEXT.md D-03 locks it on purpose; the
// disclosure IS the mitigation.
//
// This reason no longer instructs the author to split the pull request:
// that remedy is structurally unavailable in this repository
// (03-CONTEXT.md D-06 — a server-change-only PR fails the required
// wire-oracle leg, and a transcripts-only PR has no mergeable base), and
// telling a reviewer to do the impossible is worse than telling them
// nothing.
func buildReason(transcripts, serverChanges []string, depTouched bool) string {
	var sb strings.Builder
	sb.WriteString("D-03 anti-regeneration collision (advisory, not blocking, since 03-02): this pull request changes frozen transcript(s) [")
	sb.WriteString(strings.Join(transcripts, ", "))
	sb.WriteString("] together with ")

	var causes []string
	if len(serverChanges) > 0 {
		causes = append(causes, fmt.Sprintf("internal/mcp source file(s) [%s]", strings.Join(serverChanges, ", ")))
	}
	if depTouched {
		causes = append(causes, "the MCP dependency line in go.mod")
	}
	sb.WriteString(strings.Join(causes, " and "))
	sb.WriteString(". This guard's own remedy — splitting the change into two pull requests — is structurally unavailable in this repository (03-CONTEXT.md D-06: a server-change-only PR fails the required wire-oracle leg, and a transcripts-only PR has no mergeable base), so this collision no longer fails the build. The control that applies is the reviewed-diff pass D-06 mandates: capture the wire oracle before and after, read every changed line, and name every cause in the commit message. ")
	sb.WriteString("This trigger set is a deliberate floor, not a proof of innocence: transcript bytes also legitimately depend on internal/query and internal/indexer (and the tree-sitter grammars), which this guard does not watch — a clean pass means this PR did not commit the one shape the guard can detect mechanically, not that this regeneration was safe.")
	return sb.String()
}

// buildSwapExemptionNotice names both sides of the collision sdkSwapExemption
// waived and states the three facts a reviewer needs, mirroring buildReason's
// structure for the one shape 02-CONTEXT.md's D-01/D-02/D-03 declared
// exempt rather than forbidden. main.go prints this to stderr even on a
// clean (exit 0) run — an exemption that fired must never be a silent pass.
func buildSwapExemptionNotice(transcripts, serverChanges []string, depTouched bool) string {
	var sb strings.Builder
	sb.WriteString("D-03 anti-regeneration EXEMPTED (SDK-01 swap): this pull request changes frozen transcript(s) [")
	sb.WriteString(strings.Join(transcripts, ", "))
	sb.WriteString("] together with ")

	var causes []string
	if len(serverChanges) > 0 {
		causes = append(causes, fmt.Sprintf("internal/mcp source file(s) [%s]", strings.Join(serverChanges, ", ")))
	}
	if depTouched {
		causes = append(causes, "the MCP dependency line in go.mod")
	}
	sb.WriteString(strings.Join(causes, " and "))
	sb.WriteString(fmt.Sprintf(". This is SDK-01's one-time %s -> %s transition. ", mcpSDKModulePrefixes[0], mcpSDKModulePrefixes[1]))
	sb.WriteString("02-CONTEXT.md D-01/D-02/D-03 replaced this guard's byte-identity bar with semantic equivalence plus one human diff read, so these transcripts moving is expected, not a violation. ")
	sb.WriteString(fmt.Sprintf("This exemption is self-expiring: once %s is absent from go.mod, no future diff can reproduce a removal line for it, so this exact waiver can fire at most once in this repository's history.", mcpSDKModulePrefixes[0]))
	return sb.String()
}
