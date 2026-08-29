package wireoracle

import (
	"bytes"
	"sort"
	"testing"
)

// normalizationRuleCase is one hand-constructed positive-or-negative case
// for a rule declared in Rules (D-04, D-07). Constructed by hand as a raw
// byte literal rather than via a real capture — the point here is
// exercising the rule's field anchoring, not the server (TestFrozenTranscriptsMatch
// and TestEveryDeclaredFiringRuleActuallyFires already cover the real-capture
// side of this package's non-vacuity story).
type normalizationRuleCase struct {
	rule string
	name string
	subs Substitutions
	in   string
	want string
}

// normalizationRuleCases is the single source of truth for both
// TestNormalizationRulesMatchOnlyTheirOwnField (which runs every case) and
// TestRuleTestCoverageScalesWithRules (which derives the exercised
// rule-name set from this same table) — one positive case plus two
// negative cases per rule in Rules. The second negative case in each
// triple is the one that actually tests D-04/D-07's field-anchoring
// requirement: it places the EXACT SAME unstable value used by the
// positive case into a field the rule must NOT own. A negative case that
// merely uses a different value would pass even against a naive
// value-wide substitution rule and would prove nothing about anchoring.
func normalizationRuleCases() []normalizationRuleCase {
	const (
		repoDirValue   = "/tmp/wireoracle-fixture-abc123"
		otherPathValue = "/tmp/some-other-unrelated-path-xyz789"

		versionValue      = "9.9.9-mutation-probe"
		otherVersionValue = "1.0.0-unrelated"

		rfc3339Value    = "2026-08-05T12:00:00Z"
		nonRFC3339Value = "2026-08-05 (not RFC3339)"
	)

	return []normalizationRuleCase{
		// --- repoDir: anchored on a "path"-family field name (D-04) ---
		{
			rule: "repoDir",
			name: "positive: path field carrying the capture-time repoDir is replaced",
			subs: Substitutions{RepoDir: repoDirValue},
			in:   `{"result":{"path":"` + repoDirValue + `/pkga/pkga.go"}}`,
			want: `{"result":{"path":"<REPO>/pkga/pkga.go"}}`,
		},
		{
			rule: "repoDir",
			name: "negative A: a DIFFERENT absolute path in a path field is left verbatim",
			subs: Substitutions{RepoDir: repoDirValue},
			in:   `{"result":{"path":"` + otherPathValue + `/pkga/pkga.go"}}`,
			want: `{"result":{"path":"` + otherPathValue + `/pkga/pkga.go"}}`,
		},
		{
			rule: "repoDir",
			name: "negative B: the SAME repoDir value inside a text field (not a path-family field) is left verbatim",
			subs: Substitutions{RepoDir: repoDirValue},
			in:   `{"result":{"content":[{"type":"text","text":"see ` + repoDirValue + `/pkga/pkga.go for details"}]}}`,
			want: `{"result":{"content":[{"type":"text","text":"see ` + repoDirValue + `/pkga/pkga.go for details"}]}}`,
		},

		// --- serverVersion: anchored on serverInfo.version (D-04) ---
		{
			rule: "serverVersion",
			name: "positive: serverInfo.version is replaced",
			subs: Substitutions{},
			in:   `{"result":{"serverInfo":{"name":"codegraph","version":"` + versionValue + `"}}}`,
			want: `{"result":{"serverInfo":{"name":"codegraph","version":"<VERSION>"}}}`,
		},
		{
			rule: "serverVersion",
			name: "negative A: a DIFFERENT version value outside serverInfo is left verbatim",
			subs: Substitutions{},
			in:   `{"result":{"clientInfo":{"name":"codegraph-wire-oracle","version":"` + otherVersionValue + `"}}}`,
			want: `{"result":{"clientInfo":{"name":"codegraph-wire-oracle","version":"` + otherVersionValue + `"}}}`,
		},
		{
			rule: "serverVersion",
			name: "negative B: the SAME version value as a clientInfo.version key (not serverInfo) is left verbatim",
			subs: Substitutions{},
			in:   `{"result":{"clientInfo":{"name":"codegraph-wire-oracle","version":"` + versionValue + `"}}}`,
			want: `{"result":{"clientInfo":{"name":"codegraph-wire-oracle","version":"` + versionValue + `"}}}`,
		},

		// --- timestamp: anchored on a timestamp/time/ts field name, and
		// only when the value actually parses as RFC3339 (D-04) ---
		{
			rule: "timestamp",
			name: "positive: an RFC3339 value in a timestamp-named field is replaced",
			subs: Substitutions{},
			in:   `{"result":{"timestamp":"` + rfc3339Value + `"}}`,
			want: `{"result":{"timestamp":"<TS>"}}`,
		},
		{
			rule: "timestamp",
			name: "negative A: a non-RFC3339 date-like string in a timestamp field is left verbatim",
			subs: Substitutions{},
			in:   `{"result":{"timestamp":"` + nonRFC3339Value + `"}}`,
			want: `{"result":{"timestamp":"` + nonRFC3339Value + `"}}`,
		},
		{
			rule: "timestamp",
			name: "negative B: the SAME valid RFC3339 string as the value of a non-timestamp field (text) is left verbatim",
			subs: Substitutions{},
			in:   `{"result":{"text":"` + rfc3339Value + `"}}`,
			want: `{"result":{"text":"` + rfc3339Value + `"}}`,
		},
	}
}

// TestNormalizationRulesMatchOnlyTheirOwnField is D-04/D-07's per-rule
// field-anchoring guard: one positive case and TWO negative cases per rule
// declared in Rules. The negative half is the load-bearing half — a rule
// that over-matches quietly erases exactly the evidence the transcript
// exists to preserve.
func TestNormalizationRulesMatchOnlyTheirOwnField(t *testing.T) {
	for _, c := range normalizationRuleCases() {
		t.Run(c.rule+"/"+c.name, func(t *testing.T) {
			got := Normalize([]byte(c.in), c.subs)
			if !bytes.Equal(got, []byte(c.want)) {
				t.Fatalf("rule %q case %q:\n got:  %s\nwant: %s", c.rule, c.name, got, c.want)
			}
		})
	}
}

// TestRuleTestCoverageScalesWithRules is D-07's coverage-tracking guard:
// the number of distinct rule names exercised by normalizationRuleCases()
// (and therefore by TestNormalizationRulesMatchOnlyTheirOwnField, which
// runs every entry in that same table) must equal len(Rules), so a fourth
// rule added to Rules without matching cases here fails loudly instead of
// being silently untested.
func TestRuleTestCoverageScalesWithRules(t *testing.T) {
	exercised := make(map[string]bool)
	for _, c := range normalizationRuleCases() {
		exercised[c.rule] = true
	}

	if len(exercised) != len(Rules) {
		names := make([]string, 0, len(exercised))
		for name := range exercised {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Fatalf("normalizationRuleCases() exercises %d distinct rule(s) (%v), want exactly %d (len(Rules)) — a rule was added to Rules without matching test cases here", len(exercised), names, len(Rules))
	}

	for _, rule := range Rules {
		if !exercised[rule.Name] {
			t.Errorf("rule %q is declared in Rules but has no case in normalizationRuleCases()", rule.Name)
		}
	}
}

// TestNormalizeIsIdentityWhenNothingMatches is D-04's no-match identity
// property: for input containing none of the three rules' anchors,
// Normalize returns bytes byte-identical to its input, including key
// ordering, whitespace, and the presence or absence of every field —
// normalization is never a lossy rewrite of the untouched majority.
func TestNormalizeIsIdentityWhenNothingMatches(t *testing.T) {
	// Deliberately unusual whitespace, field ordering, and a multi-line
	// input, containing none of the three anchors: no "path"/"file"/
	// "root"/"repoPath" field, no "serverInfo" object, no "timestamp"/
	// "time"/"ts" field.
	in := []byte("{  \"jsonrpc\" : \"2.0\",  \"id\":42,\"result\":{\"tools\":[{\"name\":\"codegraph_explore\",\"description\":\"x\"}],\"unrelated\":  true }}\n" +
		"{\"jsonrpc\":\"2.0\",\"id\":43,\"error\":{\"code\":-32601,\"message\":\"Method foo not found\"}}")

	got := Normalize(in, Substitutions{RepoDir: "/tmp/should-not-matter-here"})
	if !bytes.Equal(got, in) {
		t.Fatalf("Normalize on non-matching input is not byte-identical:\n got:  %q\nwant: %q", got, in)
	}
}

// TestNormalizeHelpersFailLoudly mirrors
// internal/upgrade/release_workflow_shape_test.go's FailLoudly table
// convention: normalize.go's one classification helper, looksLikeRFC3339,
// must return a documented false — never a false-positive true — on empty
// and malformed input, since a false positive here would make
// normalizeTimestamp replace a value it must leave verbatim.
// repoDirVariants (the other helper normalize.go introduces that takes
// untrusted-shaped input) must not panic on an empty repoDir either.
func TestNormalizeHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"empty string", ""},
		{"malformed: date only, no time", "2026-08-05"},
		{"malformed: missing timezone", "2026-08-05T12:00:00"},
		{"malformed: not a date at all", "not-a-timestamp"},
		{"malformed: RFC3339-shaped but invalid month", "2026-13-05T12:00:00Z"},
	}
	for _, c := range cases {
		t.Run("looksLikeRFC3339/"+c.name, func(t *testing.T) {
			if looksLikeRFC3339(c.val) {
				t.Fatalf("looksLikeRFC3339(%q) = true, want false", c.val)
			}
		})
	}

	t.Run("repoDirVariants empty repoDir does not panic", func(t *testing.T) {
		got := repoDirVariants("")
		if len(got) != 1 || len(got[0]) != 0 {
			t.Fatalf("repoDirVariants(\"\") = %v, want a single empty-byte-slice variant", got)
		}
	})
}

// TestToolsListRepeatOrderingResolution is 03-03-PLAN.md Task 3's R2
// resolution test (03-03-EVIDENCE.md, VERDICT: SERVER-EMITTED-OUT-OF-
// ORDER). It feeds CanonicalizeResponseOrder a synthetic transcript whose
// responses arrive out of request-id order — id-3 before id-2, the exact
// shape this session reproduced live (03-03-EVIDENCE.md: line 2 of the
// captured transcript held the id-3 response where id-2 belonged) — and
// asserts the canonical output reorders responses ascending by id while
// leaving every byte of each response line, and every notification's
// position, untouched.
func TestToolsListRepeatOrderingResolution(t *testing.T) {
	in := []byte(
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}` + "\n" +
			`{"jsonrpc":"2.0","id":3,"result":{"tools":["c"]}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{"tools":["c"]}}` + "\n" +
			`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n" +
			`{"jsonrpc":"2.0","id":5,"result":"e"}` + "\n",
	)
	want := []byte(
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{"tools":["c"]}}` + "\n" +
			`{"jsonrpc":"2.0","id":3,"result":{"tools":["c"]}}` + "\n" +
			`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}` + "\n" +
			`{"jsonrpc":"2.0","id":5,"result":"e"}` + "\n",
	)

	got, hits := CanonicalizeResponseOrder(in)
	if !bytes.Equal(got, want) {
		t.Fatalf("CanonicalizeResponseOrder did not resequence out-of-order responses:\n got:  %s\nwant: %s", got, want)
	}
	if hits != 2 {
		t.Fatalf("CanonicalizeResponseOrder reported %d changed positions, want 2 (id-2 and id-3 both moved)", hits)
	}

	// Already-in-order input (every OTHER scenario's normal case, and
	// every existing frozen transcript in this repo — verified this
	// session against testdata/wireoracle/transcripts/toolslist-repeat.golden,
	// whose ids already appear 1,2,3) must be a true no-op: zero hits,
	// byte-identical output.
	alreadyOrdered := []byte(
		`{"jsonrpc":"2.0","id":1,"result":"a"}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":"b"}` + "\n",
	)
	got2, hits2 := CanonicalizeResponseOrder(alreadyOrdered)
	if !bytes.Equal(got2, alreadyOrdered) || hits2 != 0 {
		t.Fatalf("CanonicalizeResponseOrder must be a no-op on already-ordered input; got hits=%d, got=%s", hits2, got2)
	}
}

// TestCanonicalizeResponseOrderIgnoresRequestFrames reproduces IN-07: per
// JSON-RPC 2.0, a numeric "id" appears on REQUESTS too, not only
// responses — every server->client request this protocol defines
// (sampling/createMessage, roots/list, elicitation/create) carries both
// "method" and "id". A request frame sitting BETWEEN two genuine
// responses, carrying an id numerically between theirs, must be left at
// its ORIGINAL position — never folded into the ascending-id sort
// alongside the genuine responses on either side of it.
func TestCanonicalizeResponseOrderIgnoresRequestFrames(t *testing.T) {
	// A server->client "roots/list" REQUEST carrying id:3, sitting between
	// the id:1 and id:2 responses. Before this fix (bare responseID
	// check, id present only): all three lines classify as "response
	// positions", and a stable ascending-id sort (1, 2, 3) would swap the
	// request (id:3) with the id:2 response, producing VISIBLE reordering
	// of a line that carries a method — the exact defect IN-07 describes.
	in := []byte(
		`{"jsonrpc":"2.0","id":1,"result":"a"}` + "\n" +
			`{"jsonrpc":"2.0","method":"roots/list","id":3}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":"b"}` + "\n",
	)

	got, hits := CanonicalizeResponseOrder(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("CanonicalizeResponseOrder must leave a request frame (method present, id present) untouched at its original position:\n got:  %s\nwant: %s (unchanged)", got, in)
	}
	if hits != 0 {
		t.Fatalf("CanonicalizeResponseOrder reported %d changed positions, want 0 — the request frame must never be classified as a response or reordered against real responses", hits)
	}
}

// TestFrozenTranscriptComparisonDetectsContentMutation is 03-03-PLAN.md
// Task 3's automated discrimination proof (rule 84d1gfpywd): it proves R2's
// canonicalization narrows response ORDER only, never CONTENT. A planted
// one-character mutation inside a tool description, in an otherwise
// out-of-order (and therefore canonicalized) transcript, must still be
// caught by the SAME comparison path TestFrozenTranscriptsMatch uses
// (CanonicalizeResponseOrder, then compareBytesLineByLine) — paired with
// the unmutated case, which must compare equal, so the positive detection
// is proven against a comparison that can also say "same".
func TestFrozenTranscriptComparisonDetectsContentMutation(t *testing.T) {
	wantTranscript := []byte(
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"codegraph_callers","description":"List a symbol's reverse callers"}]}}` + "\n" +
			`{"jsonrpc":"2.0","id":3,"result":{"tools":[{"name":"codegraph_callers","description":"List a symbol's reverse callers"}]}}` + "\n",
	)

	// gotUnmutated simulates a captured run that raced (id-3 arrived
	// before id-2) but carries IDENTICAL content -- the exact shape R2
	// exists to stop failing.
	gotUnmutated := []byte(
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}` + "\n" +
			`{"jsonrpc":"2.0","id":3,"result":{"tools":[{"name":"codegraph_callers","description":"List a symbol's reverse callers"}]}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"codegraph_callers","description":"List a symbol's reverse callers"}]}}` + "\n",
	)

	gotCanon, _ := CanonicalizeResponseOrder(gotUnmutated)
	wantCanon, _ := CanonicalizeResponseOrder(wantTranscript)
	if err := compareBytesLineByLine(gotCanon, wantCanon); err != nil {
		t.Fatalf("canonicalized reorder-only transcript must compare EQUAL (no content changed): %v", err)
	}

	// Plant a ONE-CHARACTER mutation inside the tool description of the
	// id-3 response (still racing ahead of id-2, so canonicalization
	// still reorders it) — a content change, not an ordering change.
	gotMutated := bytes.Replace(gotUnmutated,
		[]byte("List a symbol's reverse callers"),
		[]byte("List a symbol's reverse callerz"), // one character changed: s -> z
		1)
	if bytes.Equal(gotMutated, gotUnmutated) {
		t.Fatal("test bug: the planted mutation did not change gotUnmutated")
	}

	gotMutatedCanon, _ := CanonicalizeResponseOrder(gotMutated)
	err := compareBytesLineByLine(gotMutatedCanon, wantCanon)
	if err == nil {
		t.Fatal("canonicalization must NOT hide a planted one-character content mutation, but the comparison reported no difference")
	}
	t.Logf("planted mutation correctly caught: %v", err)
}
