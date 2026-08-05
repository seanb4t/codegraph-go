package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// shimBinPath is the absolute path to the real mcpaudit binary, built
// once per `go test` run by TestMain — the CLI-level tests below drive
// this exact binary rather than only the in-process Run function,
// mirroring test/integration/main_test.go's TestMain pattern.
var shimBinPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "mcpaudit-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcpaudit: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	shimBinPath = filepath.Join(tmpDir, "mcpaudit")
	buildCmd := exec.Command("go", "build", "-o", shimBinPath, "github.com/seanb4t/codegraph-go/tools/mcpaudit")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "mcpaudit: TestMain: go build failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func mustJSONLine(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(b, '\n')
}

func TestParseClientFrameRecordsInitializeFields(t *testing.T) {
	line := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "claude-code", "version": "2.1.222"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		},
	})

	cf, err := ParseClientFrame(line)
	if err != nil {
		t.Fatalf("ParseClientFrame: %v", err)
	}
	if cf.Method != "initialize" || !cf.IsInitialize() {
		t.Fatalf("Method = %q, IsInitialize() = %v, want initialize/true", cf.Method, cf.IsInitialize())
	}
	if cf.ProtocolVersion != "2025-11-25" {
		t.Fatalf("ProtocolVersion = %q, want 2025-11-25", cf.ProtocolVersion)
	}
	if cf.ClientName != "claude-code" || cf.ClientVersion != "2.1.222" {
		t.Fatalf("ClientName/Version = %q/%q, want claude-code/2.1.222", cf.ClientName, cf.ClientVersion)
	}
	if len(cf.Capabilities) == 0 {
		t.Fatalf("Capabilities was not captured")
	}
	if string(cf.ID) != "1" {
		t.Fatalf("ID = %s, want 1", cf.ID)
	}
}

func TestParseServerFrameRecordsNegotiatedVersion(t *testing.T) {
	line := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo":      map[string]any{"name": "codegraph", "version": "0.3.0"},
		},
	})

	sf, err := ParseServerFrame(line)
	if err != nil {
		t.Fatalf("ParseServerFrame: %v", err)
	}
	if !sf.IsResult {
		t.Fatalf("IsResult = false, want true")
	}
	if sf.ProtocolVersion != "2025-06-18" {
		t.Fatalf("ProtocolVersion = %q, want 2025-06-18 (the NEGOTIATED value, from result, not offered)", sf.ProtocolVersion)
	}
	if sf.ServerName != "codegraph" || sf.ServerVersion != "0.3.0" {
		t.Fatalf("ServerName/Version = %q/%q, want codegraph/0.3.0", sf.ServerName, sf.ServerVersion)
	}
	if string(sf.ID) != "1" {
		t.Fatalf("ID = %s, want 1", sf.ID)
	}
}

// TestProbeThenInitializeRecordsBoth is D-11's central assertion: a
// session whose first client frame is a probe and whose second is
// initialize must yield ONE observation carrying both facts together,
// never trading one off against the other.
func TestProbeThenInitializeRecordsBoth(t *testing.T) {
	obs := newSessionObserver(Observation{PID: 1})

	probe := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0", "id": 0, "method": "server/discover",
	})
	obs.observeClientLine(probe)

	initReq := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "opencode", "version": "1.18.10"},
			"capabilities":    map[string]any{},
		},
	})
	obs.observeClientLine(initReq)

	resp := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]any{
			"protocolVersion": "2025-11-25",
			"serverInfo":      map[string]any{"name": "codegraph", "version": "0.3.0"},
		},
	})
	obs.observeServerLine(resp)

	got := obs.snapshot()
	if !got.PreInitializeProbe {
		t.Fatalf("PreInitializeProbe = false, want true (session started with server/discover, not initialize)")
	}
	if got.FirstMethod != "server/discover" {
		t.Fatalf("FirstMethod = %q, want server/discover", got.FirstMethod)
	}
	if got.OfferedProtocolVersion != "2025-11-25" || got.ClientName != "opencode" || got.ClientVersion != "1.18.10" {
		t.Fatalf("initialize fields not recorded: offered=%q client=%q/%q", got.OfferedProtocolVersion, got.ClientName, got.ClientVersion)
	}
	if got.NegotiatedProtocolVersion != "2025-11-25" || got.ServerName != "codegraph" {
		t.Fatalf("negotiated fields not recorded: negotiated=%q server=%q", got.NegotiatedProtocolVersion, got.ServerName)
	}
}

func TestParseClientFrame_NonInitializeMethodIsRecordedAsProbe(t *testing.T) {
	obs := newSessionObserver(Observation{PID: 1})
	line := mustJSONLine(t, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/list"})
	obs.observeClientLine(line)

	got := obs.snapshot()
	if got.FirstMethod != "tools/list" {
		t.Fatalf("FirstMethod = %q, want tools/list", got.FirstMethod)
	}
	if !got.PreInitializeProbe {
		t.Fatalf("PreInitializeProbe = false, want true for a first frame that is not initialize")
	}
}

func TestParseFrameHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func([]byte) error
		in   []byte
	}{
		{"ParseClientFrame: empty input", func(b []byte) error { _, err := ParseClientFrame(b); return err }, []byte("")},
		{"ParseClientFrame: non-JSON line", func(b []byte) error { _, err := ParseClientFrame(b); return err }, []byte("not json at all")},
		{"ParseClientFrame: no method and no result/error", func(b []byte) error { _, err := ParseClientFrame(b); return err }, []byte(`{"jsonrpc":"2.0","id":1}`)},
		{"ParseServerFrame: empty input", func(b []byte) error { _, err := ParseServerFrame(b); return err }, []byte("")},
		{"ParseServerFrame: non-JSON line", func(b []byte) error { _, err := ParseServerFrame(b); return err }, []byte("{not valid")},
		{"ParseServerFrame: no method and no result/error", func(b []byte) error { _, err := ParseServerFrame(b); return err }, []byte(`{"jsonrpc":"2.0","id":1}`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(c.in); err == nil {
				t.Fatalf("%s: expected a non-nil error, got nil", c.name)
			}
		})
	}
}

// TestMalformedFrameIsDigestedNotStored is T-02-01's repudiation guard: a
// malformed frame must never appear in full anywhere in the resulting
// Observation — only a SHA-256 digest and a short escaped prefix.
func TestMalformedFrameIsDigestedNotStored(t *testing.T) {
	obs := newSessionObserver(Observation{PID: 1})
	// Deliberately longer than the 64-byte prefix cap, so a passing test
	// actually proves truncation rather than accidentally fitting whole.
	malformed := []byte(strings.Repeat("x", 200) + " this is not valid JSON-RPC at all")
	obs.observeClientLine(malformed)

	got := obs.snapshot()
	if got.ParseError == "" {
		t.Fatalf("ParseError is empty, want a recorded parse error")
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(got.FrameDigest) {
		t.Fatalf("FrameDigest = %q, want a 64-character hex string", got.FrameDigest)
	}
	if strings.Contains(got.ParseError, strings.Repeat("x", 200)) {
		t.Fatalf("ParseError contains the malformed frame in full: %q", got.ParseError)
	}
}

func TestProxyIsByteExactInBothDirections(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found on PATH")
	}
	input := []byte("line one\nline two\nline three\n")
	var out bytes.Buffer
	cfg := Config{
		RealBin: catPath,
		LogPath: filepath.Join(t.TempDir(), "obs.jsonl"),
		In:      bytes.NewReader(input),
		Out:     &out,
		ErrOut:  io.Discard,
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Equal(out.Bytes(), input) {
		t.Fatalf("output bytes = %q, want %q (byte-exact passthrough)", out.Bytes(), input)
	}
}

func TestProxyPreservesCRLFAndUnterminatedFinalFrame(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found on PATH")
	}
	input := []byte("crlf line one\r\ncrlf line two\r\nunterminated final frame, no trailing newline")
	var out bytes.Buffer
	cfg := Config{
		RealBin: catPath,
		LogPath: filepath.Join(t.TempDir(), "obs.jsonl"),
		In:      bytes.NewReader(input),
		Out:     &out,
		ErrOut:  io.Discard,
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !bytes.Equal(out.Bytes(), input) {
		t.Fatalf("output bytes = %q, want %q (CRLF and unterminated final frame preserved byte-exact)", out.Bytes(), input)
	}
}

// TestRunAppendsObservationWhenCanceledBeforeStdinCloses is a regression
// test for a real bug found running the Task 2 audit against live Claude
// Code: it terminates the mcpaudit process directly rather than closing
// its stdin. Without Config.Ctx wired to a caught signal, Run would block
// forever in wg.Wait() on the caller->child forwarding goroutine — which
// reads from a pipe the (now-gone) external caller never closes — so the
// post-exit appendObservation call was never reached, leaving only the
// process-start record behind. cfg.Ctx cancellation is the deterministic,
// signal-free way to exercise that same "the caller is gone but never
// closed our stdin" condition in a test.
func TestRunAppendsObservationWhenCanceledBeforeStdinCloses(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found on PATH")
	}
	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())
	logPath := filepath.Join(t.TempDir(), "obs.jsonl")

	cfg := Config{
		RealBin: catPath,
		LogPath: logPath,
		In:      pr,
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Ctx:     ctx,
	}

	runDone := make(chan error, 1)
	go func() { runDone <- Run(cfg) }()

	// Give Run a moment to start the child and block reading from `in`
	// (deliberately never closed here), then cancel — simulating a
	// caught termination signal.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run returned an error after cfg.Ctx was canceled: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Run did not return within 5s of ctx cancellation — still blocked on the caller's stdin, which never closes (the exact bug this test guards against)")
	}

	lines := readLogLines(t, logPath)
	if len(lines) != 2 {
		t.Fatalf("log has %d lines, want 2 (process-start record + one observation) even though Run was interrupted mid-exchange", len(lines))
	}
}

// readLogLines reads every non-blank line from an observation log.
func readLogLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			lines = append(lines, sc.Text())
		}
	}
	return lines
}

// TestCLIProxiesAndAppendsProcessStartAndObservation drives the real
// built mcpaudit binary (not just the in-process Run function) against
// `cat` as a stand-in real server: it must echo the exact frame back on
// stdout, append a process-start record plus one observation to the log,
// and running the same command a second time against the same -log path
// must double the record count (append-only, never truncated).
func TestCLIProxiesAndAppendsProcessStartAndObservation(t *testing.T) {
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found on PATH")
	}
	logPath := filepath.Join(t.TempDir(), "obs.jsonl")

	initReq := mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "test-client", "version": "0.0.1"},
			"capabilities":    map[string]any{},
		},
	})

	run := func() {
		cmd := exec.Command(shimBinPath, "-real", catPath, "-log", logPath, "-args", "")
		cmd.Stdin = bytes.NewReader(initReq)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("mcpaudit -real cat: %v", err)
		}
		if !bytes.Equal(stdout.Bytes(), initReq) {
			t.Fatalf("stdout = %q, want the exact initialize frame echoed back: %q", stdout.Bytes(), initReq)
		}
	}

	run()
	lines := readLogLines(t, logPath)
	if len(lines) != 2 {
		t.Fatalf("log has %d lines after one invocation, want 2 (process-start record + one observation)", len(lines))
	}

	run()
	lines = readLogLines(t, logPath)
	if len(lines) != 4 {
		t.Fatalf("log has %d lines after a second invocation, want 4 (append-only doubles the record count)", len(lines))
	}
}

// TestMainRequiresRealAndLogFlags is main.go's fail-loudly guard: a
// silently-misconfigured shim would produce an empty log indistinguishable
// from "the client sent nothing".
func TestMainRequiresRealAndLogFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"missing both flags", nil},
		{"missing -log", []string{"-real", "/bin/cat"}},
		{"missing -real", []string{"-log", "/tmp/does-not-matter.jsonl"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(c.args, bytes.NewReader(nil), io.Discard, &stderr)
			if code == 0 {
				t.Fatalf("run(%v) = 0, want non-zero exit for a missing required flag", c.args)
			}
			if stderr.Len() == 0 {
				t.Fatalf("run(%v): stderr is empty, want a message naming the missing flag(s)", c.args)
			}
		})
	}
}
