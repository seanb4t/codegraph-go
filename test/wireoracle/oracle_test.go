package wireoracle

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	internalmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// codegraphSessionLinePrefix is VRFY-03/D-16's published, additive-only
// stderr prefix — internal/mcp.sessionLinePrefix is unexported, so this is
// a deliberate, documented duplicate of the published contract, not a
// guess.
const codegraphSessionLinePrefix = "codegraph: mcp-session"

// sanitizedRequestedVersion mirrors internal/mcp.sanitizeClientField's
// documented empty-string behavior
// (internal/mcp/session_line.go: "an empty or all-stripped result becomes
// the literal \"<unknown>\"") -- a deliberate, documented duplicate of the
// published contract, like codegraphSessionLinePrefix above, since
// sanitizeClientField itself is unexported. Only the empty-string case
// matters here: legacy-omitted-version is the one era scenario whose
// EraOfferedVersion is "" (its initialize params carry no protocolVersion
// key at all), so its session line reports requested=<unknown>, not
// requested= -- every other era scenario's offered version is already
// free of spaces and control characters, so this is a no-op for them.
func sanitizedRequestedVersion(offered string) string {
	if offered == "" {
		return "<unknown>"
	}
	return offered
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "testdata", "wireoracle", "fixture"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}
	return abs
}

// mustCaptureScenario runs Capture for the named scenario against a fresh
// t.TempDir() fixture copy, in its own subprocess (VRFY-01 concurrency
// edge: every scenario capture is independent, so repeated runs are
// byte-identical).
func mustCaptureScenario(t *testing.T, name string) (Transcript, Scenario) {
	t.Helper()
	sc, ok := ScenarioByName(name)
	if !ok {
		t.Fatalf("unknown scenario %q", name)
	}
	workDir := t.TempDir()
	tr, err := Capture(context.Background(), binPath, fixtureDir(t), workDir, sc)
	if err != nil {
		t.Fatalf("capture scenario %q: %v", name, err)
	}
	return tr, sc
}

// TestFrozenTranscriptsMatch is the oracle's central assertion (VRFY-01,
// VRFY-04): for every scenario, capture against the real binary, normalize,
// and require byte equality with the frozen transcript. Before the golden
// file exists, this fails naming the missing file — it must never skip.
func TestFrozenTranscriptsMatch(t *testing.T) {
	scenarios := Scenarios()
	if len(scenarios) == 0 {
		t.Fatal("Scenarios() returned zero scenarios")
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			tr, sc := mustCaptureScenario(t, sc.Name)

			normalized, ledger := NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})

			goldenPath := TranscriptPath(sc.Name)
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("scenario %q: frozen transcript missing at %s: %v", sc.Name, goldenPath, err)
			}

			if len(normalized) == 0 {
				t.Fatalf("scenario %q: normalized transcript is empty — an empty transcript is never a match", sc.Name)
			}

			assertBytesEqualLineByLine(t, sc.Name, normalized, want)
			if sc.NoInitialize {
				// No initialize means the VRFY-03 AddAfterInitialize hook
				// never fires (no session line to expect) and there is no
				// initialize result.protocolVersion field to anchor
				// against (edge-call-before-initialize, 01-04-PLAN Task 2).
				assertNoSessionLine(t, sc, tr.Stderr)
			} else if sc.EraScenario {
				// The six-era Legacy baseline (01-05-PLAN Task 1
				// checkpoint: six-era) deliberately offers a protocol
				// revision OTHER than internal/mcp.ProtocolVersion for
				// five of its six scenarios -- the independent D-02 spec
				// anchor compares against each scenario's own recorded
				// offered/negotiated pair instead of the single fixed
				// literal every other scenario shares.
				assertSessionLine(t, sc, tr.Stderr, sanitizedRequestedVersion(sc.EraOfferedVersion), sc.EraNegotiatedVersion)
				assertProtocolVersionAnchor(t, tr.Stdout, sc.EraNegotiatedVersion)
			} else {
				assertSessionLine(t, sc, tr.Stderr, internalmcp.ProtocolVersion, internalmcp.ProtocolVersion)
				assertProtocolVersionAnchor(t, tr.Stdout, internalmcp.ProtocolVersion)
			}

			_ = ledger // exercised directly by TestNormalizeRuleLedgerIsHonest below
		})
	}
}

// assertBytesEqualLineByLine fails naming the scenario and quoting the
// first differing line from each side with %q — never a summarized diff.
func assertBytesEqualLineByLine(t *testing.T, scenario string, got, want []byte) {
	t.Helper()
	if bytes.Equal(got, want) {
		return
	}
	gotLines := bytes.Split(got, []byte("\n"))
	wantLines := bytes.Split(want, []byte("\n"))
	max := len(gotLines)
	if len(wantLines) > max {
		max = len(wantLines)
	}
	for i := 0; i < max; i++ {
		var g, w []byte
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if !bytes.Equal(g, w) {
			t.Fatalf("scenario %q: normalized transcript differs at line %d:\n got: %q\nwant: %q", scenario, i+1, g, w)
		}
	}
	t.Fatalf("scenario %q: normalized transcript differs in length (got %d bytes, want %d bytes) but no differing line was found", scenario, len(got), len(want))
}

// assertSessionLine checks the D-13/D-14 stderr contract: exactly one line
// beginning with the published prefix, parsing into the four keys in
// fixed order, with requested/negotiated compared against wantRequested/
// wantNegotiated (internal/mcp.ProtocolVersion for every scenario except
// the six-era Legacy baseline, whose own EraOfferedVersion/
// EraNegotiatedVersion the caller passes instead) and client/tools
// matching what this scenario's own script and ExpectTools declare.
func assertSessionLine(t *testing.T, sc Scenario, stderr string, wantRequested, wantNegotiated string) {
	t.Helper()

	var sessionLines []string
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, codegraphSessionLinePrefix) {
			sessionLines = append(sessionLines, l)
		}
	}
	if len(sessionLines) != 1 {
		t.Fatalf("scenario %q: stderr must contain exactly one %q line, found %d: %v", sc.Name, codegraphSessionLinePrefix, len(sessionLines), sessionLines)
	}

	line := sessionLines[0]
	rest := strings.TrimSpace(strings.TrimPrefix(line, codegraphSessionLinePrefix))
	fields := strings.Fields(rest)
	wantKeys := []string{"requested", "negotiated", "client", "tools"}
	if len(fields) != len(wantKeys) {
		t.Fatalf("scenario %q: session line has %d fields, want %d (%v): %q", sc.Name, len(fields), len(wantKeys), wantKeys, line)
	}

	values := make(map[string]string, len(fields))
	for i, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("scenario %q: session line field %q is not key=value: %q", sc.Name, f, line)
		}
		if parts[0] != wantKeys[i] {
			t.Fatalf("scenario %q: session line field %d key = %q, want %q (fixed order requested,negotiated,client,tools): %q", sc.Name, i, parts[0], wantKeys[i], line)
		}
		values[parts[0]] = parts[1]
	}

	if values["requested"] != wantRequested {
		t.Fatalf("scenario %q: session line requested=%q, want %q", sc.Name, values["requested"], wantRequested)
	}
	if values["negotiated"] != wantNegotiated {
		t.Fatalf("scenario %q: session line negotiated=%q, want %q", sc.Name, values["negotiated"], wantNegotiated)
	}
	wantClient := handshakeExploreClientName + "/" + handshakeExploreClientVersion
	if values["client"] != wantClient {
		t.Fatalf("scenario %q: session line client=%q, want %q", sc.Name, values["client"], wantClient)
	}
	wantTools := strconv.Itoa(sc.ExpectTools)
	if values["tools"] != wantTools {
		t.Fatalf("scenario %q: session line tools=%q, want %q", sc.Name, values["tools"], wantTools)
	}
}

// assertNoSessionLine checks the NoInitialize counterpart of
// assertSessionLine: a scenario that never sends "initialize" must never
// produce the VRFY-03 stderr session line at all, since its
// AddAfterInitialize hook has nothing to hook (edge-call-before-initialize,
// RESEARCH Pitfall 2).
func assertNoSessionLine(t *testing.T, sc Scenario, stderr string) {
	t.Helper()
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, codegraphSessionLinePrefix) {
			t.Fatalf("scenario %q: NoInitialize=true but stderr contains a session line: %q", sc.Name, l)
		}
	}
}

// assertProtocolVersionAnchor is D-02's hand-authored spec anchor,
// independent of the capture: the initialize response's
// result.protocolVersion must equal want (internal/mcp.ProtocolVersion for
// every scenario except the six-era Legacy baseline, whose own
// EraNegotiatedVersion the caller passes instead), decoded from the RAW
// captured line by field name only — never through an SDK type.
func assertProtocolVersionAnchor(t *testing.T, stdout []byte, want string) {
	t.Helper()
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != 1 || len(frame.Result) == 0 {
			continue
		}
		var res struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("decode initialize result.protocolVersion: %v; raw line: %q", err, line)
		}
		if res.ProtocolVersion != want {
			t.Fatalf("initialize response protocolVersion = %q, want %q: %q", res.ProtocolVersion, want, line)
		}
		return
	}
	t.Fatal("no initialize response (id=1, non-empty result) found in captured stdout")
}

// TestTracerExploreCallSucceeds is VRFY-04's flagship-tool assertion: the
// tools/call (id 3) response for codegraph_explore decodes to
// result.isError false and a non-empty result.content[0].text — a real
// explore result, not a rejected-argument error.
func TestTracerExploreCallSucceeds(t *testing.T) {
	tr, _ := mustCaptureScenario(t, "handshake-explore")

	for _, line := range bytes.Split(tr.Stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != 3 || len(frame.Result) == 0 {
			continue
		}
		var res struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("decode tools/call result: %v; raw line: %q", err, line)
		}
		if res.IsError {
			t.Fatalf("codegraph_explore tools/call returned isError=true: %q", line)
		}
		if len(res.Content) == 0 || res.Content[0].Text == "" {
			t.Fatalf("codegraph_explore tools/call returned empty result.content[0].text: %q", line)
		}
		return
	}
	t.Fatal("no tools/call response (id=3, non-empty result) found in captured stdout")
}

// TestNormalizeRuleLedgerIsHonest is D-04/D-07's ledger-honesty check: a
// rule declared ExpectFires: true must have actually fired at least once
// against the tracer capture; a rule declared ExpectFires: false must
// carry a non-empty Why.
func TestNormalizeRuleLedgerIsHonest(t *testing.T) {
	tr, _ := mustCaptureScenario(t, "handshake-explore")
	_, ledger := NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})

	for _, rule := range Rules {
		hits := ledger[rule.Name]
		if rule.ExpectFires {
			if hits < 1 {
				t.Errorf("rule %q: ExpectFires=true but the ledger recorded %d hits against the tracer capture — rule stopped firing", rule.Name, hits)
			}
			continue
		}
		if rule.Why == "" {
			t.Errorf("rule %q: ExpectFires=false but Why is empty", rule.Name)
		}
	}
}

// TestCaptureIsDeterminstic proves running the scenario twice in one test
// process produces byte-identical normalized transcripts and identical
// ledgers (VRFY-01 concurrency edge).
func TestCaptureIsDeterministic(t *testing.T) {
	sc, ok := ScenarioByName("handshake-explore")
	if !ok {
		t.Fatal("scenario handshake-explore not found")
	}

	var normalized [2][]byte
	var ledgers [2]map[string]int
	for i := range normalized {
		workDir := t.TempDir()
		tr, err := Capture(context.Background(), binPath, fixtureDir(t), workDir, sc)
		if err != nil {
			t.Fatalf("run %d: capture: %v", i, err)
		}
		normalized[i], ledgers[i] = NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})
	}

	if !bytes.Equal(normalized[0], normalized[1]) {
		t.Fatalf("normalized transcript differs between two runs in the same process:\nrun0: %q\nrun1: %q", normalized[0], normalized[1])
	}
	if !reflect.DeepEqual(ledgers[0], ledgers[1]) {
		t.Fatalf("ledger differs between two runs: %v vs %v", ledgers[0], ledgers[1])
	}
}

// assertFramingInvariant is D-02's framing invariant, applied to every
// scenario (not just the three named Anchors()): every non-empty stdout
// line parses as a JSON object carrying a "jsonrpc" field equal to "2.0",
// and every request id sc.Requests sent has exactly one response line
// bearing that id.
func assertFramingInvariant(t *testing.T, sc Scenario, stdout []byte) {
	t.Helper()

	wantIDs := requestIDs(sc.Requests)
	seen := make(map[float64]int, len(wantIDs))

	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			t.Fatalf("scenario %q: stdout line is not valid JSON: %v: %q", sc.Name, err, line)
		}
		if frame.JSONRPC != "2.0" {
			t.Fatalf("scenario %q: stdout line jsonrpc = %q, want \"2.0\": %q", sc.Name, frame.JSONRPC, line)
		}
		if id, ok := idAsFloat64(frame.ID); ok {
			seen[id]++
		}
	}

	for id := range wantIDs {
		if seen[id] != 1 {
			t.Fatalf("scenario %q: request id %v has %d response lines, want exactly 1", sc.Name, id, seen[id])
		}
	}
}

// TestSpecAnchorsHold is D-02's central assertion: every hand-authored
// anchor fails independently of the frozen transcripts, checked against
// freshly captured stdout (never the golden file — that would be
// circular), plus the framing invariant applied uniformly across every
// scenario in Scenarios(). Each scenario is captured exactly once here
// (not once per matching anchor) so this test's own subprocess-spawn cost
// stays at 17, not 17-plus-anchor-count.
func TestSpecAnchorsHold(t *testing.T) {
	anchorsByScenario := make(map[string][]Anchor)
	for _, a := range Anchors() {
		anchorsByScenario[a.Scenario] = append(anchorsByScenario[a.Scenario], a)
	}

	for _, sc := range Scenarios() {
		t.Run(sc.Name, func(t *testing.T) {
			tr, _ := mustCaptureScenario(t, sc.Name)

			assertFramingInvariant(t, sc, tr.Stdout)

			if sc.Name == "handshake-explore" {
				t.Run("protocolVersion", func(t *testing.T) {
					assertProtocolVersionAnchor(t, tr.Stdout, internalmcp.ProtocolVersion)
				})
			}

			for _, a := range anchorsByScenario[sc.Name] {
				t.Run(a.Name, func(t *testing.T) {
					a.Assert(t, tr.Stdout)
				})
			}
		})
	}
}

// TestToolsListOrderIsDeterministic proves toolslist-repeat's two
// consecutive tools/list calls (ids 2 and 3) return byte-identical
// result.tools arrays — not merely "both non-empty".
func TestToolsListOrderIsDeterministic(t *testing.T) {
	tr, _ := mustCaptureScenario(t, "toolslist-repeat")

	var toolsArrays [][]byte
	for _, line := range bytes.Split(tr.Stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || (idf != 2 && idf != 3) {
			continue
		}
		var res struct {
			Tools json.RawMessage `json:"tools"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("decode tools/list result.tools (id=%v): %v; raw line: %q", idf, err, line)
		}
		toolsArrays = append(toolsArrays, res.Tools)
	}

	if len(toolsArrays) != 2 {
		t.Fatalf("toolslist-repeat: found %d tools/list responses (ids 2,3), want 2", len(toolsArrays))
	}
	if !bytes.Equal(toolsArrays[0], toolsArrays[1]) {
		t.Fatalf("toolslist-repeat: result.tools differs between the two consecutive tools/list calls:\nfirst:  %s\nsecond: %s", toolsArrays[0], toolsArrays[1])
	}
}

// toolNamesFromCapture captures scenarioName and decodes only
// result.tools[].name out of the response bearing wantID, sorted.
func toolNamesFromCapture(t *testing.T, scenarioName string, wantID float64) []string {
	t.Helper()
	tr, _ := mustCaptureScenario(t, scenarioName)

	for _, line := range bytes.Split(tr.Stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != wantID {
			continue
		}
		var res struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("scenario %q: decode tools/list result.tools[].name: %v; raw line: %q", scenarioName, err, line)
		}
		names := make([]string, len(res.Tools))
		for i, tool := range res.Tools {
			names[i] = tool.Name
		}
		sort.Strings(names)
		return names
	}
	t.Fatalf("scenario %q: no tools/list response (id=%v) found in captured stdout", scenarioName, wantID)
	return nil
}

// equalStrings mirrors internal/mcp/server_test.go's helper of the same
// name exactly (PITFALLS Trap D's positive pattern to preserve): a
// positional comparison over two already-sorted slices, never a
// length-greater-than-zero check.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestToolsListExactSets proves each tools/list variant advertises exactly
// its expected set — exact-set for the non-empty cases, exact-zero for
// toolslist-no-index — never a non-empty/length-greater-than check
// (PITFALLS Trap D).
func TestToolsListExactSets(t *testing.T) {
	cases := []struct {
		scenario string
		want     []string
	}{
		{"toolslist-default", []string{"codegraph_explore"}},
		{"toolslist-allowlist", []string{"codegraph_explore", "codegraph_node", "codegraph_status"}},
		{"toolslist-no-index", []string{}},
	}
	for _, c := range cases {
		t.Run(c.scenario, func(t *testing.T) {
			got := toolNamesFromCapture(t, c.scenario, 2)
			if !equalStrings(got, c.want) {
				t.Fatalf("scenario %q: tools/list advertised %v, want exactly %v", c.scenario, got, c.want)
			}
		})
	}
}

// findToolCallRequest scans sc's static Requests (not captured output) for
// its one tools/call request (every scenario in Scenarios() carries at most
// one, per the concurrency ordering constraint documented above
// Scenarios()) and returns its request id and target tool name.
func findToolCallRequest(sc Scenario) (id float64, name string, ok bool) {
	for _, req := range sc.Requests {
		method, _ := req["method"].(string)
		if method != "tools/call" {
			continue
		}
		params, _ := req["params"].(map[string]any)
		toolName, _ := params["name"].(string)
		reqID, idOK := idAsFloat64(req["id"])
		if !idOK {
			continue
		}
		return reqID, toolName, true
	}
	return 0, "", false
}

// isSuccessfulToolCall decodes only result.isError out of stdout's response
// bearing wantID, returning true when that response exists, is not an
// error-level JSON-RPC response, and decodes to isError=false.
func isSuccessfulToolCall(stdout []byte, wantID float64) bool {
	for _, line := range bytes.Split(stdout, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != wantID || len(frame.Result) == 0 {
			continue
		}
		var res struct {
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			return false
		}
		return !res.IsError
	}
	return false
}

// TestEveryRegisteredToolHasASuccessfulCallScenario derives the full
// registered tool-name set from toolslist-repeat's capture (all seven
// companionNames allowlisted, per internal/mcp/server.go:35, plus
// codegraph_explore — 8 names) and requires that for EACH of those 8 names
// there exists a scenario whose Requests contains a tools/call naming it
// AND whose captured response for that call decodes to result.isError
// false. This is the structural replacement for "handshake-explore already
// covers codegraph_explore" prose-delegation: a scenario that silently
// degrades into an error result cannot pass as coverage.
func TestEveryRegisteredToolHasASuccessfulCallScenario(t *testing.T) {
	registered := toolNamesFromCapture(t, "toolslist-repeat", 2)
	if len(registered) != 8 {
		t.Fatalf("toolslist-repeat: registered %d tools, want 8: %v", len(registered), registered)
	}

	remaining := make(map[string]bool, len(registered))
	for _, name := range registered {
		remaining[name] = true
	}

	for _, sc := range Scenarios() {
		reqID, toolName, ok := findToolCallRequest(sc)
		if !ok || !remaining[toolName] {
			continue
		}

		tr, _ := mustCaptureScenario(t, sc.Name)
		if isSuccessfulToolCall(tr.Stdout, reqID) {
			delete(remaining, toolName)
		}
	}

	if len(remaining) > 0 {
		names := make([]string, 0, len(remaining))
		for name := range remaining {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Fatalf("no scenario provides a successful tools/call for: %v", names)
	}
}

// legacyEraExpectedCount is the size of the multi-era Legacy baseline
// approved at 01-05-PLAN's Task 1 blocking checkpoint (selection:
// six-era) -- the four protocol revisions today's server recognizes, plus
// one unsupported revision, plus one omitted-version request. This brings
// the suite from 17 to 23 scenarios total; plan 07 independently sets its
// own ExpectedScenarioCount to that resulting total (D-07) -- this
// constant is NOT that one, and does not create it.
const legacyEraExpectedCount = 6

// TestLegacyEraBaselineIsDocumented is a positive assertion over the
// already-frozen six-era baseline (VRFY-04, D-06): it reads golden files
// from disk, never re-captures, proving the frozen evidence itself is
// complete and internally consistent rather than re-verifying wire
// behavior TestFrozenTranscriptsMatch already covers byte-for-byte.
func TestLegacyEraBaselineIsDocumented(t *testing.T) {
	var eraScenarios []Scenario
	for _, sc := range Scenarios() {
		if strings.HasPrefix(sc.Name, legacyEraPrefix) {
			eraScenarios = append(eraScenarios, sc)
		}
	}

	if len(eraScenarios) != legacyEraExpectedCount {
		t.Fatalf("era scenario count (name prefix %q) = %d, want exactly %d (01-05-PLAN Task 1 six-era selection)", legacyEraPrefix, len(eraScenarios), legacyEraExpectedCount)
	}

	for _, sc := range eraScenarios {
		t.Run(sc.Name, func(t *testing.T) {
			goldenPath := TranscriptPath(sc.Name)
			raw, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("scenario %q: frozen transcript missing at %s: %v", sc.Name, goldenPath, err)
			}
			if len(bytes.TrimSpace(raw)) == 0 {
				t.Fatalf("scenario %q: frozen transcript at %s is empty", sc.Name, goldenPath)
			}

			if sc.EraOfferedVersion == "" || sc.EraOfferedVersion != sc.EraNegotiatedVersion {
				// The unsupported and omitted-version scenarios negotiate
				// a DIFFERENT value than offered by design (silent
				// coercion, RESEARCH Pitfall 1) -- only the four
				// supported revisions assert offered==negotiated here.
				return
			}

			negotiated := decodeFrozenInitializeProtocolVersion(t, sc.Name, raw)
			if negotiated != sc.EraOfferedVersion {
				t.Fatalf("scenario %q: frozen transcript negotiated protocolVersion = %q, want %q (offered == negotiated for a supported revision)", sc.Name, negotiated, sc.EraOfferedVersion)
			}
		})
	}
}

// decodeFrozenInitializeProtocolVersion decodes only the initialize
// response's (id=1) result.protocolVersion field out of a frozen golden
// file's raw bytes, field-by-field -- never through an SDK type, and
// never by re-capturing.
func decodeFrozenInitializeProtocolVersion(t *testing.T, scenario string, raw []byte) string {
	t.Helper()
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			continue
		}
		idf, ok := idAsFloat64(frame.ID)
		if !ok || idf != 1 || len(frame.Result) == 0 {
			continue
		}
		var res struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("scenario %q: decode frozen initialize result.protocolVersion: %v", scenario, err)
		}
		return res.ProtocolVersion
	}
	t.Fatalf("scenario %q: no initialize response (id=1, non-empty result) found in frozen transcript", scenario)
	return ""
}
