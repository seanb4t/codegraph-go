package wireoracle

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
			} else {
				assertSessionLine(t, sc, tr.Stderr)
				assertProtocolVersionAnchor(t, tr.Stdout)
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
// fixed order, with requested/negotiated pinned to internal/mcp.ProtocolVersion
// and client/tools matching what this scenario's own script and
// ExpectTools declare.
func assertSessionLine(t *testing.T, sc Scenario, stderr string) {
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

	if values["requested"] != internalmcp.ProtocolVersion {
		t.Fatalf("scenario %q: session line requested=%q, want %q (internal/mcp.ProtocolVersion)", sc.Name, values["requested"], internalmcp.ProtocolVersion)
	}
	if values["negotiated"] != internalmcp.ProtocolVersion {
		t.Fatalf("scenario %q: session line negotiated=%q, want %q (internal/mcp.ProtocolVersion)", sc.Name, values["negotiated"], internalmcp.ProtocolVersion)
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
// result.protocolVersion must equal internal/mcp.ProtocolVersion, decoded
// from the RAW captured line by field name only — never through an SDK
// type.
func assertProtocolVersionAnchor(t *testing.T, stdout []byte) {
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
		if res.ProtocolVersion != internalmcp.ProtocolVersion {
			t.Fatalf("initialize response protocolVersion = %q, want %q (internal/mcp.ProtocolVersion): %q", res.ProtocolVersion, internalmcp.ProtocolVersion, line)
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
