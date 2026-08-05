package wireoracle

import (
	"bytes"
	"encoding/json"
	"regexp"
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
