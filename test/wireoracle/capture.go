// Package wireoracle is the phase's wire-level regression oracle (D-01,
// VRFY-01): it spawns the REAL codegraph binary via os/exec, drives a
// scripted JSON-RPC session over real stdio, and returns the raw captured
// bytes. It never imports any github.com/mark3labs/mcp-go package and
// never decodes a line into an SDK type — the entire point is proving wire
// behavior independent of whichever SDK happens to be serving it.
//
// This is a normal Go package under test/, never testdata/ (GOLDEN-01's own
// lesson: `go list ./...` silently skips any directory literally named
// testdata).
package wireoracle

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Scenario describes one scripted MCP session the oracle drives against a
// real spawned binary.
type Scenario struct {
	// Name identifies the scenario and its frozen transcript file
	// (testdata/wireoracle/transcripts/<Name>.golden).
	Name string
	// Env holds extra "KEY=VALUE" entries appended to the subprocess's
	// environment, alongside the always-on CODEGRAPH_NO_WATCH=1.
	Env []string
	// Index, when true, runs `codegraph init <workDir>` to completion
	// before starting `serve --mcp`.
	Index bool
	// Requests is the ordered list of JSON-RPC requests written to the
	// subprocess's stdin, one per line.
	Requests []map[string]any
	// ExpectTools is the tool count this scenario's session is expected
	// to advertise via VRFY-03's stderr session line.
	ExpectTools int
}

// Transcript is one capture's raw output.
type Transcript struct {
	// Stdout is every captured stdout line, newline-joined, RAW (not yet
	// normalized).
	Stdout []byte
	// Stderr is the subprocess's complete captured stderr output.
	Stderr string
	// RepoDir is the workDir the fixture was copied into and the
	// subprocess ran against — callers use it to build Substitutions
	// without recomputing the path.
	RepoDir string
}

// syncBuffer is a concurrency-safe io.Writer wrapping a bytes.Buffer.
// Copied from test/integration/mcp_stdout_purity_test.go's syncBuffer
// (WR-01): os/exec starts a background goroutine to copy a subprocess's
// stderr pipe into cmd.Stderr whenever that field is a non-*os.File
// io.Writer, and that goroutine is only joined by cmd.Wait() — a plain
// bytes.Buffer would race under that goroutine's concurrent writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// Capture copies fixtureSrc into workDir, optionally runs `binPath init
// workDir` (sc.Index), spawns `binPath serve --mcp` with cmd.Dir=workDir,
// writes sc.Requests to its stdin (one JSON-marshaled line per request,
// then closes stdin so the server sees EOF and exits on its own in the
// common case), and collects every stdout line until one response per
// request id has been seen, the scanner reaches EOF, or a 30-second
// deadline fires.
//
// Capture takes no *testing.T and therefore owns its own lifecycle
// explicitly: it never relies on t.Cleanup, because its second caller
// (test/wireoracle/cmd/wireoracle) has no *testing.T at all. On every error
// path and on the deadline path the subprocess is killed; cmd.Wait() always
// runs before Capture returns, via a deferred call registered immediately
// after cmd.Start() succeeds, so no subprocess and no stderr-copy goroutine
// ever outlives this call.
//
// Every captured byte is treated as untrusted display data: Capture never
// executes or shell-interpolates anything it reads back from the
// subprocess.
func Capture(ctx context.Context, binPath, fixtureSrc, workDir string, sc Scenario) (Transcript, error) {
	if err := copyTree(fixtureSrc, workDir); err != nil {
		return Transcript{}, fmt.Errorf("wireoracle: copy fixture %s into %s: %w", fixtureSrc, workDir, err)
	}

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if sc.Index {
		out, err := exec.CommandContext(runCtx, binPath, "init", workDir).CombinedOutput()
		if err != nil {
			return Transcript{}, fmt.Errorf("wireoracle: scenario %q: init %s: %w: %s", sc.Name, workDir, err, out)
		}
	}

	cmd := exec.CommandContext(runCtx, binPath, "serve", "--mcp")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), append([]string{"CODEGRAPH_NO_WATCH=1"}, sc.Env...)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Transcript{}, fmt.Errorf("wireoracle: scenario %q: stdin pipe: %w", sc.Name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Transcript{}, fmt.Errorf("wireoracle: scenario %q: stdout pipe: %w", sc.Name, err)
	}
	stderrBuf := &syncBuffer{}
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return Transcript{}, fmt.Errorf("wireoracle: scenario %q: start %s: %w", sc.Name, binPath, err)
	}
	defer func() {
		_ = cmd.Wait()
	}()

	for _, req := range sc.Requests {
		b, marshalErr := json.Marshal(req)
		if marshalErr != nil {
			_ = cmd.Process.Kill()
			return Transcript{}, fmt.Errorf("wireoracle: scenario %q: marshal request: %w", sc.Name, marshalErr)
		}
		if _, writeErr := stdin.Write(append(b, '\n')); writeErr != nil {
			_ = cmd.Process.Kill()
			return Transcript{}, fmt.Errorf("wireoracle: scenario %q: write request: %w", sc.Name, writeErr)
		}
	}
	if closeErr := stdin.Close(); closeErr != nil {
		_ = cmd.Process.Kill()
		return Transcript{}, fmt.Errorf("wireoracle: scenario %q: close stdin: %w", sc.Name, closeErr)
	}

	wantIDs := requestIDs(sc.Requests)
	seen := make(map[float64]bool, len(wantIDs))

	type scannedLine struct{ raw []byte }
	lines := make(chan scannedLine)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			lines <- scannedLine{raw: append([]byte(nil), scanner.Bytes()...)}
		}
	}()

	var out bytes.Buffer
	for len(seen) < len(wantIDs) {
		select {
		case ln, ok := <-lines:
			if !ok {
				_ = cmd.Process.Kill()
				return Transcript{}, fmt.Errorf("wireoracle: scenario %q: stdout closed after %d/%d responses; stderr:\n%s",
					sc.Name, len(seen), len(wantIDs), stderrBuf.String())
			}
			out.Write(ln.raw)
			out.WriteByte('\n')
			if id, ok := responseID(ln.raw); ok {
				seen[id] = true
			}
		case <-runCtx.Done():
			_ = cmd.Process.Kill()
			return Transcript{}, fmt.Errorf("wireoracle: scenario %q: capture deadline exceeded after %d/%d responses; stderr:\n%s",
				sc.Name, len(seen), len(wantIDs), stderrBuf.String())
		}
	}

	return Transcript{
		Stdout:  out.Bytes(),
		Stderr:  stderrBuf.String(),
		RepoDir: workDir,
	}, nil
}

// copyTree copies src into dst, preserving directory structure. Lifted
// from test/integration/main_test.go's copyFixture mechanics, pointed at
// whatever fixture the caller supplies.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// requestIDs extracts the set of JSON-RPC request ids from reqs.
func requestIDs(reqs []map[string]any) map[float64]bool {
	ids := make(map[float64]bool, len(reqs))
	for _, r := range reqs {
		if raw, ok := r["id"]; ok {
			if f, ok := idAsFloat64(raw); ok {
				ids[f] = true
			}
		}
	}
	return ids
}

// idAsFloat64 normalizes a JSON-RPC id (which may arrive as a Go int
// literal from a Scenario or as a float64 from json.Unmarshal) to a single
// comparable type.
func idAsFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

// responseID extracts the "id" field from one raw captured line, returning
// ok=false for a line that is not valid JSON or carries no id (e.g. a
// notification).
func responseID(raw []byte) (float64, bool) {
	var frame struct {
		ID any `json:"id"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		return 0, false
	}
	return idAsFloat64(frame.ID)
}
