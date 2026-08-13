package wireoracle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// statFrozenTranscript is readFrozenTranscript's pure logic core, split out
// so TestEmptyTranscriptNeverMatches can assert this check FAILS (on a
// deliberately zero-byte fixture) without that expected failure
// propagating into go test's own pass/fail exit code — a nested t.Run
// whose body calls t.Fatalf always marks its ancestor tests failed too,
// regardless of what the caller does with t.Run's returned bool, so the
// "prove this guard fires" demonstration must go through a plain error
// return instead. Stats goldenPath and fails BEFORE attempting any byte
// comparison if it is missing OR zero bytes (D-07): a zero-byte frozen
// transcript trivially equaling a zero-byte capture is the single easiest
// way for a byte-exact comparator to become vacuous.
func statFrozenTranscript(goldenPath string) ([]byte, error) {
	info, err := os.Stat(goldenPath)
	if err != nil {
		return nil, fmt.Errorf("frozen transcript missing at %s: %w", goldenPath, err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("frozen transcript at %s is zero bytes — an empty frozen transcript can never be a valid comparison target (D-07)", goldenPath)
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		return nil, fmt.Errorf("read frozen transcript %s: %w", goldenPath, err)
	}
	return want, nil
}

// readFrozenTranscript is statFrozenTranscript's *testing.T-failing
// wrapper, used by every production assertion path (TestFrozenTranscriptsMatch,
// TestEmptyTranscriptNeverMatches's real-transcript case).
func readFrozenTranscript(t *testing.T, goldenPath string) []byte {
	t.Helper()
	want, err := statFrozenTranscript(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	return want
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

			want := readFrozenTranscript(t, TranscriptPath(sc.Name))

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

			_ = ledger // exercised directly by TestEveryDeclaredFiringRuleActuallyFires below
		})
	}
}

// TestScenarioCountIsExact is D-07's central non-shrinkage guard:
// len(Scenarios()) must equal the package constant ExpectedScenarioCount
// with EXACT equality, never a lower bound — a lower bound cannot detect a
// scenario silently disappearing.
func TestScenarioCountIsExact(t *testing.T) {
	got := len(Scenarios())
	if got != ExpectedScenarioCount {
		t.Fatalf("len(Scenarios()) = %d, want exactly %d (ExpectedScenarioCount) — either a scenario silently disappeared or one was added without updating the constant beside Scenarios()", got, ExpectedScenarioCount)
	}
}

// TestTranscriptSetMatchesScenarioSet is D-07's two-way set-equality guard:
// the scenario-name set and the frozen-transcript basename set under
// testdata/wireoracle/transcripts/ must be identical in BOTH directions — a
// scenario without a transcript fails, and an orphaned transcript without a
// scenario also fails, each named explicitly. Reads the directory with
// os.ReadDir rather than reconstructing filenames from scenario names, so a
// transcript that exists under an unexpected name is visible rather than
// invisible.
func TestTranscriptSetMatchesScenarioSet(t *testing.T) {
	scenarios := Scenarios()
	scenarioNames := make(map[string]bool, len(scenarios))
	for _, sc := range scenarios {
		scenarioNames[sc.Name] = true
	}

	dir := filepath.Dir(TranscriptPath("placeholder"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read transcripts dir %s: %v", dir, err)
	}
	transcriptNames := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".golden") {
			continue
		}
		transcriptNames[strings.TrimSuffix(name, ".golden")] = true
	}

	var missingTranscripts, orphanedTranscripts []string
	for name := range scenarioNames {
		if !transcriptNames[name] {
			missingTranscripts = append(missingTranscripts, name)
		}
	}
	for name := range transcriptNames {
		if !scenarioNames[name] {
			orphanedTranscripts = append(orphanedTranscripts, name)
		}
	}
	sort.Strings(missingTranscripts)
	sort.Strings(orphanedTranscripts)

	if len(missingTranscripts) > 0 || len(orphanedTranscripts) > 0 {
		t.Fatalf("scenario/transcript set mismatch: missing transcripts (scenario has no .golden file) = %v; orphaned transcripts (.golden file has no scenario) = %v", missingTranscripts, orphanedTranscripts)
	}
}

// TestEmptyTranscriptNeverMatches is D-07's central empty-transcript guard
// (VRFY-04 empty edge): an empty or zero-byte frozen transcript must never
// compare equal to a captured transcript, in either direction, and a
// frozen file that exists but is zero bytes must fail loudly naming the
// file rather than being silently treated as a trivially satisfied
// comparison.
func TestEmptyTranscriptNeverMatches(t *testing.T) {
	t.Run("captured transcript vs zero-length expected value", func(t *testing.T) {
		tr, sc := mustCaptureScenario(t, "handshake-explore")
		normalized, _ := NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})
		if len(normalized) == 0 {
			t.Fatalf("scenario %q: normalized capture unexpectedly empty (should never happen)", sc.Name)
		}
		err := compareBytesLineByLine(normalized, []byte{})
		if err == nil {
			t.Fatal("compareBytesLineByLine(realCapture, empty) unexpectedly reported equal — an empty expected value must never compare equal to a real capture")
		}
		t.Logf("guard fired as expected: %v", err)
	})

	t.Run("zero-length capture vs a real frozen transcript", func(t *testing.T) {
		want := readFrozenTranscript(t, TranscriptPath("handshake-explore"))
		err := compareBytesLineByLine([]byte{}, want)
		if err == nil {
			t.Fatal("compareBytesLineByLine(empty, realFrozenTranscript) unexpectedly reported equal — a zero-length capture must never compare equal to a real frozen transcript")
		}
		t.Logf("guard fired as expected: %v", err)
	})

	t.Run("frozen file exists but is zero bytes fails loudly naming the file", func(t *testing.T) {
		dir := t.TempDir()
		zeroPath := filepath.Join(dir, "zero-byte.golden")
		if err := os.WriteFile(zeroPath, []byte{}, 0o644); err != nil {
			t.Fatalf("write zero-byte fixture: %v", err)
		}
		_, err := statFrozenTranscript(zeroPath)
		if err == nil {
			t.Fatalf("statFrozenTranscript(%s) unexpectedly succeeded against a zero-byte file — must fail loudly naming the path", zeroPath)
		}
		if !strings.Contains(err.Error(), zeroPath) {
			t.Fatalf("statFrozenTranscript error does not name the path %s: %v", zeroPath, err)
		}
		t.Logf("guard fired as expected, naming the path: %v", err)
	})
}

// compareBytesLineByLine is assertBytesEqualLineByLine's pure logic core,
// split out for the same reason statFrozenTranscript is: it lets
// TestEmptyTranscriptNeverMatches assert this comparison FAILS on a
// deliberately empty input without that expected failure propagating into
// go test's own pass/fail exit code. Returns nil if got and want are
// byte-equal; otherwise an error naming the first differing line (or the
// length mismatch) with %q — never a summarized diff.
func compareBytesLineByLine(got, want []byte) error {
	if bytes.Equal(got, want) {
		return nil
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
			return fmt.Errorf("normalized transcript differs at line %d:\n got: %q\nwant: %q", i+1, g, w)
		}
	}
	return fmt.Errorf("normalized transcript differs in length (got %d bytes, want %d bytes) but no differing line was found", len(got), len(want))
}

// assertBytesEqualLineByLine is compareBytesLineByLine's *testing.T-failing
// wrapper, fails naming the scenario and quoting the first differing line.
func assertBytesEqualLineByLine(t *testing.T, scenario string, got, want []byte) {
	t.Helper()
	if err := compareBytesLineByLine(got, want); err != nil {
		t.Fatalf("scenario %q: %v", scenario, err)
	}
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

// TestEveryDeclaredFiringRuleActuallyFires is D-04/D-07's ledger-honesty
// guard: run a REAL capture across every scenario in Scenarios() (not a
// hand-constructed case — only a real capture can catch a rule that
// silently stopped matching, e.g. because the SDK changed key order or
// spacing around an anchor), accumulate the per-rule hit ledger
// NormalizeWithLedger returns, and require that every rule declared
// ExpectFires: true has a total hit count of at least one across the whole
// suite, and every rule declared ExpectFires: false carries a non-empty
// Why. Generalized from a single-scenario check (plan 01's tracer,
// formerly TestNormalizeRuleLedgerIsHonest) to the full scenario set so
// "representative" means the entire frozen baseline, not one hand-picked
// sample.
func TestEveryDeclaredFiringRuleActuallyFires(t *testing.T) {
	scenarios := Scenarios()
	totals := make(map[string]int, len(Rules))
	for _, sc := range scenarios {
		tr, _ := mustCaptureScenario(t, sc.Name)
		_, ledger := NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})
		for name, hits := range ledger {
			totals[name] += hits
		}
	}

	for _, rule := range Rules {
		hits := totals[rule.Name]
		if rule.ExpectFires {
			if hits < 1 {
				t.Errorf("rule %q: ExpectFires=true but the accumulated ledger recorded %d hits across all %d scenarios — rule stopped firing", rule.Name, hits, len(scenarios))
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
// every request id the scenario actually owes a response for
// (sc.expectedResponseIDs()) has exactly one response line bearing that
// id, and every request id named by sc.NoResponseRequests has EXACTLY
// ZERO — asserted positively, not merely left unchecked, so the exemption
// is a live claim rather than a silent blind spot (05-01-PLAN,
// 05-CONTEXT.md D-01: SubscriptionsListenResult is sent only on graceful
// subscription teardown, never on stdin EOF).
func assertFramingInvariant(t *testing.T, sc Scenario, stdout []byte) {
	t.Helper()

	allIDs := requestIDs(sc.Requests)
	wantIDs := sc.expectedResponseIDs()
	noResponseIDs := make(map[float64]bool, len(allIDs)-len(wantIDs))
	for id := range allIDs {
		if !wantIDs[id] {
			noResponseIDs[id] = true
		}
	}
	seen := make(map[float64]int, len(allIDs))

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
	for id := range noResponseIDs {
		if seen[id] != 0 {
			t.Fatalf("scenario %q: request id %v (Scenario.NoResponseRequests) has %d response lines, want exactly 0", sc.Name, id, seen[id])
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

// allEightToolNames is the full default surface, sorted — the eight names
// toolslist-default and toolslist-repeat advertise when CODEGRAPH_MCP_TOOLS
// is unset. Hand-authored here rather than imported from internal/mcp,
// matching this package's standing rule that its expected values are never
// read from the code under test (VRFY-01, the same rule that keeps
// legacyEraVersions and modernProtocolVersion local literals).
var allEightToolNames = []string{
	"codegraph_callees",
	"codegraph_callers",
	"codegraph_explore",
	"codegraph_files",
	"codegraph_impact",
	"codegraph_node",
	"codegraph_search",
	"codegraph_status",
}

// TestToolsListExactSets proves each tools/list variant advertises exactly
// its expected set — exact-set for the non-empty cases, exact-zero for
// toolslist-no-index and modern-listen-catalog-change's pre-init id-2 call
// — never a non-empty/length-greater-than check (PITFALLS Trap D). The two
// modern-listen-catalog-change cases prove SPEC-09's catalog-transition
// claim independently of the frozen transcript and the notification
// anchors: the id-2 tools/list (before the mid-session `codegraph init`)
// advertises exactly the empty set, and the id-3 tools/list (after it, same
// connection) advertises exactly {codegraph_explore}.
//
// The toolslist-default and toolslist-filter-empty rows are a MATCHED PAIR,
// not two independent cases: both send a byte-identical request script
// against a byte-identical fixture, and the ONLY difference between them is
// whether CODEGRAPH_MCP_TOOLS is set to the empty string or not set at all.
// An implementation reading os.Getenv instead of os.LookupEnv cannot tell
// those apart and fails exactly one of these two rows — which is what makes
// the pair a real gate on the narrowing contract rather than two spellings
// of the same assertion.
func TestToolsListExactSets(t *testing.T) {
	cases := []struct {
		name     string
		scenario string
		wantID   float64
		want     []string
	}{
		{"toolslist-default", "toolslist-default", 2, allEightToolNames},
		{"toolslist-filter-empty", "toolslist-filter-empty", 2, []string{"codegraph_explore"}},
		{"toolslist-narrowed", "toolslist-narrowed", 2, []string{"codegraph_explore", "codegraph_node", "codegraph_status"}},
		{"toolslist-no-index", "toolslist-no-index", 2, []string{}},
		{"modern-listen-catalog-change-pre-init", "modern-listen-catalog-change", 2, []string{}},
		{"modern-listen-catalog-change-post-init", "modern-listen-catalog-change", 3, []string{"codegraph_explore"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := toolNamesFromCapture(t, c.scenario, c.wantID)
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
// registered tool-name set from toolslist-repeat's capture (the default
// surface with CODEGRAPH_MCP_TOOLS unset: all seven companionNames plus
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

// resourceURIsFromCapture captures scenarioName and decodes only
// result.resources[].uri out of the response bearing wantID, sorted —
// toolNamesFromCapture's sibling for the resources method family.
func resourceURIsFromCapture(t *testing.T, scenarioName string, wantID float64) []string {
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
			Resources []struct {
				URI string `json:"uri"`
			} `json:"resources"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			t.Fatalf("scenario %q: decode resources/list result.resources[].uri: %v; raw line: %q", scenarioName, err, line)
		}
		uris := make([]string, len(res.Resources))
		for i, r := range res.Resources {
			uris[i] = r.URI
		}
		sort.Strings(uris)
		return uris
	}
	t.Fatalf("scenario %q: no resources/list response (id=%v) found in captured stdout", scenarioName, wantID)
	return nil
}

// findResourceReadRequest scans sc's static Requests (not captured output)
// for its one resources/read request (every scenario in Scenarios() carries
// at most one, per the concurrency ordering constraint documented above
// Scenarios()) and returns its request id and target URI —
// findToolCallRequest's sibling for the resources method family.
func findResourceReadRequest(sc Scenario) (id float64, uri string, ok bool) {
	for _, req := range sc.Requests {
		method, _ := req["method"].(string)
		if method != "resources/read" {
			continue
		}
		params, _ := req["params"].(map[string]any)
		u, _ := params["uri"].(string)
		reqID, idOK := idAsFloat64(req["id"])
		if !idOK {
			continue
		}
		return reqID, u, true
	}
	return 0, "", false
}

// successfulResourceRead decodes result.contents out of stdout's response
// bearing wantID, returning true only when that response exists and its
// first contents entry carries a non-empty Text and mimeType
// "text/markdown" — isSuccessfulToolCall's sibling for the resources
// method family.
func successfulResourceRead(stdout []byte, wantID float64) bool {
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
			Contents []struct {
				Text     string `json:"text"`
				MIMEType string `json:"mimeType"`
			} `json:"contents"`
		}
		if err := json.Unmarshal(frame.Result, &res); err != nil {
			return false
		}
		if len(res.Contents) == 0 {
			return false
		}
		return res.Contents[0].Text != "" && res.Contents[0].MIMEType == "text/markdown"
	}
	return false
}

// TestEveryAdvertisedResourceURIHasASuccessfulReadScenario is
// TestEveryRegisteredToolHasASuccessfulCallScenario's structural sibling for
// resources: it derives the required URI set from a LIVE capture of
// resources-list (never a hand-typed list), so an eleventh resource added in
// a future phase turns this red until it also has a wire read scenario. It
// fails immediately if that set is empty or smaller than the ten this phase
// ships, so the test cannot pass vacuously against a broken capture. It then
// walks Scenarios(), finds each scenario containing a resources/read
// request, captures it, and deletes a URI from the remaining set only when
// its response frame carries a successful non-empty text/markdown read —
// failing, naming exactly which URIs have no successful wire read, in the
// same shape TestEveryRegisteredToolHasASuccessfulCallScenario uses.
// resources-read-unknown's URI (codegraph://tools/does-not-exist) is
// deliberately not in the advertised set, so it is naturally excluded from
// satisfying anything here without any special-case code.
func TestEveryAdvertisedResourceURIHasASuccessfulReadScenario(t *testing.T) {
	advertised := resourceURIsFromCapture(t, "resources-list", 2)
	if len(advertised) < 10 {
		t.Fatalf("resources-list: advertised %d resource URIs, want at least 10: %v", len(advertised), advertised)
	}

	remaining := make(map[string]bool, len(advertised))
	for _, uri := range advertised {
		remaining[uri] = true
	}

	for _, sc := range Scenarios() {
		reqID, uri, ok := findResourceReadRequest(sc)
		if !ok || !remaining[uri] {
			continue
		}

		tr, _ := mustCaptureScenario(t, sc.Name)
		if successfulResourceRead(tr.Stdout, reqID) {
			delete(remaining, uri)
		}
	}

	if len(remaining) > 0 {
		uris := make([]string, 0, len(remaining))
		for uri := range remaining {
			uris = append(uris, uri)
		}
		sort.Strings(uris)
		t.Fatalf("no scenario provides a successful resources/read for: %v", uris)
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
