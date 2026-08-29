package wireoracle

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"time"
)

// Substitutions carries the capture-time values normalization rules match
// against — today, only the fixture's absolute repo directory.
type Substitutions struct {
	// RepoDir is the capture-time absolute path of the copied fixture
	// tree (Transcript.RepoDir) — the value the repoDir rule replaces.
	RepoDir string
}

// Rule documents one normalization rule in the fixed allowlist below. Name
// keys the per-capture hit ledger NormalizeWithLedger returns; ExpectFires
// records whether the tracer capture actually observed this rule fire, so
// a rule that silently stops matching is distinguishable from a genuine
// byte diff (TestNormalizeRuleLedgerIsHonest enforces this). A rule with
// ExpectFires: false must carry a non-empty Why explaining why it is
// retained anyway.
type Rule struct {
	Name        string
	Placeholder string
	ExpectFires bool
	Why         string
}

// Rules is the complete, documented normalization allowlist (D-04).
// Normalization is named-field placeholder substitution only: every rule
// below is anchored on the JSON field name that owns the value, never on
// the value alone — a bare value-replacement rule would silently erase the
// same string appearing in a field that must stay byte-verbatim. Bytes not
// matched by any rule here are compared byte-for-byte; the transcript is
// never decoded and re-encoded through encoding/json (a round trip erases
// field presence/absence and key ordering, which is the only place an
// omitempty-on-bare-bool regression is visible).
var Rules = []Rule{
	{
		Name:        "repoDir",
		Placeholder: "<REPO>",
		// The handshake-explore scenario's codegraph_explore result is a
		// successful markdown TEXT blob (result.content[0].text) — the
		// wire-level JSON around it carries no "path"/"file"/"root"/
		// "repoPath" field for this scenario, so this rule does not fire
		// against today's frozen transcript. Retained for forward
		// compatibility: a later scenario (e.g. codegraph_node's file-mode
		// read) is expected to carry one of these fields verbatim.
		ExpectFires: false,
		Why:         "handshake-explore's tools/call result carries no path/file/root/repoPath JSON field; retained for later scenarios that do (e.g. codegraph_node file-mode reads)",
	},
	{
		Name:        "serverVersion",
		Placeholder: "<VERSION>",
		// Every initialize response carries serverInfo.version.
		ExpectFires: true,
		Why:         "",
	},
	{
		Name:        "timestamp",
		Placeholder: "<TS>",
		// No response in the handshake-explore scenario carries a
		// timestamp/time/ts field today.
		ExpectFires: false,
		Why:         "no response in the handshake-explore scenario carries a timestamp/time/ts field; retained for a future scenario that does",
	},
}

// repoDirFieldNames are the JSON field names the repoDir rule anchors on
// (D-04): a "path"-family field only, never a bare value match.
var repoDirFieldNames = []string{"path", "file", "root", "repoPath"}

// serverInfoVersionRe anchors on "serverInfo", then a bounded,
// whitespace-tolerant "version" key within the same object (the
// non-greedy [^{}]*? never crosses a nested brace), and captures the
// three pieces needed to replace only the value.
var serverInfoVersionRe = regexp.MustCompile(`("serverInfo"\s*:\s*\{[^{}]*?"version"\s*:\s*")([^"]*)(")`)

// timestampFieldRe anchors on a timestamp-named field ("timestamp",
// "time", or "ts") and captures its key and raw string value; the value is
// only replaced if it parses as RFC3339/RFC3339Nano.
var timestampFieldRe = regexp.MustCompile(`"(timestamp|time|ts)"\s*:\s*"([^"]*)"`)

// NormalizeWithLedger applies every rule in Rules, in order, to raw and
// returns the normalized bytes alongside a per-rule hit count — the
// mechanism TestNormalizeRuleLedgerIsHonest checks against each rule's
// declared ExpectFires.
func NormalizeWithLedger(raw []byte, subs Substitutions) ([]byte, map[string]int) {
	ledger := make(map[string]int, len(Rules))

	out, hits := normalizeRepoDir(raw, subs.RepoDir)
	ledger["repoDir"] = hits

	out, hits = normalizeServerVersion(out)
	ledger["serverVersion"] = hits

	out, hits = normalizeTimestamp(out)
	ledger["timestamp"] = hits

	return out, ledger
}

// Normalize applies NormalizeWithLedger and discards the ledger.
func Normalize(raw []byte, subs Substitutions) []byte {
	out, _ := NormalizeWithLedger(raw, subs)
	return out
}

// normalizeRepoDir replaces the DIRECTORY PORTION of a "path"-family
// field's value with <REPO> whenever that value begins with repoDir — the
// remainder of the value (a subpath) is left verbatim. Handles both the
// raw and JSON-escaped forms of repoDir as anchored alternatives.
func normalizeRepoDir(raw []byte, repoDir string) ([]byte, int) {
	if repoDir == "" {
		return raw, 0
	}
	hits := 0
	out := raw
	for _, variant := range repoDirVariants(repoDir) {
		for _, field := range repoDirFieldNames {
			anchor := []byte(`"` + field + `":"`)
			target := append(append([]byte{}, anchor...), variant...)
			n := bytes.Count(out, target)
			if n == 0 {
				continue
			}
			replacement := append(append([]byte{}, anchor...), []byte("<REPO>")...)
			out = bytes.ReplaceAll(out, target, replacement)
			hits += n
		}
	}
	return out, hits
}

// repoDirVariants returns the raw byte form of repoDir plus its
// JSON-escaped form (deduplicated when they are identical, which is the
// common case on a host whose temp path needs no JSON escaping).
func repoDirVariants(repoDir string) [][]byte {
	variants := [][]byte{[]byte(repoDir)}
	if escaped, err := json.Marshal(repoDir); err == nil && len(escaped) >= 2 {
		unquoted := string(escaped[1 : len(escaped)-1])
		if unquoted != repoDir {
			variants = append(variants, []byte(unquoted))
		}
	}
	return variants
}

// normalizeServerVersion replaces every serverInfo.version value with
// <VERSION>, anchored on the field name (D-04) rather than the fully
// compact literal form — a compact-serialization-shaped anchor would
// silently stop matching if key order or spacing ever changed.
func normalizeServerVersion(raw []byte) ([]byte, int) {
	hits := 0
	out := serverInfoVersionRe.ReplaceAllFunc(raw, func(m []byte) []byte {
		sub := serverInfoVersionRe.FindSubmatch(m)
		hits++
		return append(append(append([]byte{}, sub[1]...), []byte("<VERSION>")...), sub[3]...)
	})
	return out, hits
}

// normalizeTimestamp replaces the value of a timestamp/time/ts field with
// <TS>, but only when that value actually parses as RFC3339 or
// RFC3339Nano — an RFC3339-shaped string in any other field, or a
// non-RFC3339 value in one of these fields, is left verbatim.
func normalizeTimestamp(raw []byte) ([]byte, int) {
	hits := 0
	out := timestampFieldRe.ReplaceAllFunc(raw, func(m []byte) []byte {
		sub := timestampFieldRe.FindSubmatch(m)
		key := string(sub[1])
		val := string(sub[2])
		if !looksLikeRFC3339(val) {
			return m
		}
		hits++
		return []byte(`"` + key + `":"<TS>"`)
	})
	return out, hits
}

func looksLikeRFC3339(val string) bool {
	if _, err := time.Parse(time.RFC3339Nano, val); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, val)
	return err == nil
}

// isResponseLine reports whether raw is a JSON-RPC RESPONSE line — id
// present AND method absent (IN-07). responseID alone (id present) is
// NOT sufficient: per JSON-RPC 2.0, a numeric id appears on requests too,
// not only responses, and every server->client REQUEST this protocol
// defines (sampling/createMessage, roots/list, elicitation/create)
// carries both a "method" and an "id". No current scenario emits one, so
// misclassifying it here is latent — but were one added, that request
// frame would join the sorted-by-id set below and could be swapped with
// an unrelated response line, masking or fabricating an ordering
// discrepancy in the frozen transcript. frameMethod already exists next
// door (capture.go) for exactly this distinction.
func isResponseLine(raw []byte) bool {
	if _, ok := responseID(raw); !ok {
		return false
	}
	_, hasMethod := frameMethod(raw)
	return !hasMethod
}

// CanonicalizeResponseOrder is 03-03-PLAN.md Task 3's R2 resolution
// (03-03-EVIDENCE.md, VERDICT: SERVER-EMITTED-OUT-OF-ORDER): the frozen
// transcript oracle was freezing response ARRIVAL order, a property
// github.com/modelcontextprotocol/go-sdk@v1.7.0 explicitly does not
// guarantee for pipelined non-initialize calls — mcp/server.go's
// ServerSession.handle calls jsonrpc2.Async(ctx) unconditionally for
// every call except "initialize" (modelcontextprotocol/go-sdk#26), and
// internal/jsonrpc2/conn.go's handleAsync dequeues requests sequentially
// but only blocks until Async() fires or the handler completes, so two
// consecutive same-method calls run in independently scheduled goroutines
// with no ordering guarantee between them.
//
// CanonicalizeResponseOrder narrows what the oracle freezes to response
// CONTENT, never touching a byte within a line: it identifies every line
// position that holds a JSON-RPC RESPONSE (id present AND method absent,
// per isResponseLine's classification above — built from responseID and
// frameMethod, capture.go's existing extractors, reused rather than
// re-derived, "no second copy of a rule") and reassigns those SAME
// positions the response bytes sorted ascending by id, via a stable sort.
// Every other line — notifications, blank lines, anything without a
// numeric id — is left completely untouched, at its original position,
// with its original content. On input already in ascending-id order (the
// common case for every scenario in this package, and for every existing
// frozen transcript — verified this session against
// testdata/wireoracle/transcripts/toolslist-repeat.golden, whose ids
// already read 1,2,3) this is a true no-op: zero hits, byte-identical
// output.
//
// Deliberately NOT folded into Rules/NormalizeWithLedger above: those
// rules are single-value, named-field placeholder SUBSTITUTIONS (D-04);
// this reorders whole LINES and substitutes nothing, so it is a
// structurally different operation with its own name, applied as an
// explicit separate step by TestFrozenTranscriptsMatch, on both the
// captured and the frozen side, immediately before the byte comparison —
// never silently decoding/re-encoding through encoding/json, matching
// this file's own documented "never a round trip" discipline.
//
// hits reports how many response positions actually changed — 0 when the
// input was already canonical.
func CanonicalizeResponseOrder(raw []byte) ([]byte, int) {
	if len(raw) == 0 {
		return raw, 0
	}
	hadTrailingNewline := bytes.HasSuffix(raw, []byte("\n"))
	lines := bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n"))

	var positions []int
	var respLines [][]byte
	for i, line := range lines {
		if isResponseLine(line) {
			positions = append(positions, i)
			respLines = append(respLines, line)
		}
	}
	if len(positions) < 2 {
		return raw, 0
	}

	sorted := make([][]byte, len(respLines))
	copy(sorted, respLines)
	sort.SliceStable(sorted, func(i, j int) bool {
		idI, _ := responseID(sorted[i])
		idJ, _ := responseID(sorted[j])
		return idI < idJ
	})

	hits := 0
	for i, pos := range positions {
		if !bytes.Equal(lines[pos], sorted[i]) {
			hits++
		}
		lines[pos] = sorted[i]
	}

	out := bytes.Join(lines, []byte("\n"))
	if hadTrailingNewline {
		out = append(out, '\n')
	}
	return out, hits
}
