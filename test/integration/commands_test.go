package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	cgmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// runWithin is the scripts' within command. It runs a program with the
// caller's stdio and fails if the program has not exited after the
// duration, so a hang fails its script instead of the whole run.
//
//	within duration program [args...]
func runWithin() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: within duration program [args...]")
		os.Exit(2)
	}
	limit, err := time.ParseDuration(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "within:", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[2], os.Args[3:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		fmt.Fprintf(os.Stderr, "within: %v did not exit within %s\n", os.Args[2:], limit)
		os.Exit(1)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "within:", err)
		os.Exit(1)
	}
}

// runMCPClient is the scripts' mcpclient command. Each subcommand starts
// `codegraph serve --mcp` in the current directory with the current
// environment, keeps its stdin open for the whole exchange, and stops it
// at the end.
//
//	mcpclient tools
//	    List the tools, one name per line.
//	mcpclient explore query
//	    Print the codegraph_explore text for query.
//	mcpclient stderr text
//	    Wait until the server's stderr contains text, then print that stderr.
//	mcpclient live-edit -from dir -to dir -touch file -query q -want text
//	    Copy the files of -from into -to while the session is live, then
//	    call codegraph_explore until its text contains -want, and print it.
//	mcpclient raw tool
//	    Send initialize and a tools/call for tool as hand-written JSON-RPC
//	    lines, and print each stdout line until both responses arrive.
func runMCPClient() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mcpclient tools|explore|stderr|live-edit|raw [args...]")
		os.Exit(2)
	}
	sub, args := os.Args[1], os.Args[2:]
	var err error
	switch {
	case sub == "tools" && len(args) == 0:
		err = mcpTools()
	case sub == "explore" && len(args) == 1:
		err = mcpExplore(args[0])
	case sub == "stderr" && len(args) == 1:
		err = mcpStderr(args[0])
	case sub == "live-edit":
		err = mcpLiveEdit(args)
	case sub == "raw" && len(args) == 1:
		err = mcpRaw(args[0])
	default:
		err = fmt.Errorf("unknown subcommand or wrong arguments: %q", os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpclient:", err)
		os.Exit(1)
	}
}

func connect(ctx context.Context, stderr io.Writer) (*mcp.ClientSession, error) {
	cmd := exec.CommandContext(ctx, "codegraph", "serve", "--mcp")
	cmd.Stderr = stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-integration-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to codegraph serve --mcp: %w", err)
	}
	return session, nil
}

func mcpTools() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session, err := connect(ctx, os.Stderr)
	if err != nil {
		return err
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("ListTools: %w", err)
	}
	for _, tool := range result.Tools {
		fmt.Println(tool.Name)
	}
	return nil
}

func mcpExplore(query string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	session, err := connect(ctx, os.Stderr)
	if err != nil {
		return err
	}
	defer session.Close()

	text, err := explore(ctx, session, query)
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}

// explore returns the first text block of a codegraph_explore call. A
// tool error is an error.
func explore(ctx context.Context, session *mcp.ClientSession, query string) (string, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_explore",
		Arguments: map[string]any{"query": query},
	})
	if err != nil {
		return "", fmt.Errorf("codegraph_explore transport error: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("codegraph_explore returned a tool error: %+v", result)
	}
	if len(result.Content) == 0 {
		return "", errors.New("codegraph_explore returned no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("codegraph_explore content[0] is not text: %+v", result.Content[0])
	}
	return text.Text, nil
}

// notifyBuffer is a concurrency-safe buffer that signals each write. The
// os/exec goroutine that copies the server's stderr writes to it.
type notifyBuffer struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	written chan struct{}
}

func (b *notifyBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case b.written <- struct{}{}:
	default:
	}
	return b.buf.Write(p)
}

func (b *notifyBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// mcpStderr waits for text on the server's stderr. The watcher writes its
// messages off the handshake path (D-06), so they can arrive after
// initialize completes.
func mcpStderr(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stderr := &notifyBuffer{written: make(chan struct{}, 1)}
	session, err := connect(ctx, stderr)
	if err != nil {
		return fmt.Errorf("%w; stderr: %q", err, stderr.String())
	}
	defer session.Close()

	limit := time.After(10 * time.Second)
	for !strings.Contains(stderr.String(), text) {
		select {
		case <-stderr.written:
		case <-limit:
			return fmt.Errorf("server stderr does not contain %q after 10s: %q", text, stderr.String())
		}
	}
	fmt.Print(stderr.String())
	return nil
}

// mcpLiveEdit is WATCH-01's live-edit client. The anchor file named by
// -touch is rewritten every 500ms with the same content: the watcher
// starts off the handshake path (D-06) and can miss the first burst, and
// the rewrite interval is longer than the debounce window the script sets,
// so the rewrites cannot hold off the flush. Every explore in the loop
// must succeed: a store-lock collision between an explore and the flush
// (CR-01) must never surface as an error.
func mcpLiveEdit(args []string) error {
	fs := flag.NewFlagSet("live-edit", flag.ContinueOnError)
	from := fs.String("from", "", "directory holding the new files")
	to := fs.String("to", "", "directory to copy them into")
	touch := fs.String("touch", "", "file of -from to rewrite until -want appears")
	query := fs.String("query", "", "codegraph_explore query")
	want := fs.String("want", "", "text the explore payload must contain")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" || *touch == "" || *query == "" || *want == "" {
		return errors.New("live-edit needs -from, -to, -touch, -query and -want")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	session, err := connect(ctx, os.Stderr)
	if err != nil {
		return err
	}
	defer session.Close()

	entries, err := os.ReadDir(*from)
	if err != nil {
		return err
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(*from, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*to, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	anchor := filepath.Join(*to, *touch)
	anchorData, err := os.ReadFile(anchor)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(90 * time.Second)
	lastTouch := time.Now()
	var payload string
	for time.Now().Before(deadline) {
		if payload, err = explore(ctx, session, *query); err != nil {
			return fmt.Errorf("during a live watch session: %w", err)
		}
		if strings.Contains(payload, *want) {
			fmt.Println(payload)
			return nil
		}
		if time.Since(lastTouch) >= 500*time.Millisecond {
			if err := os.WriteFile(anchor, anchorData, 0o644); err != nil {
				return err
			}
			lastTouch = time.Now()
		}
	}
	return fmt.Errorf("codegraph_explore %q never contained %q within 90s; last payload:\n%s", *query, *want, payload)
}

// mcpRaw is HYG-02's stdout-purity client. It owns the server's stdout
// itself, because the SDK's stdio transport skips any line that is not
// JSON-RPC, so a client built on it cannot see a violation. Every stdout
// line up to both responses must be a JSON-RPC frame, and no response may
// be a JSON-RPC error (WR-02: an error for the tools/call would mean the
// store-open path was never exercised).
func mcpRaw(tool string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "codegraph", "serve", "--mcp")
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	enc := json.NewEncoder(stdin)
	for _, req := range []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				// The repo-owned constant: a dependency bump must not move
				// this value silently (VRFY-02).
				"protocolVersion": cgmcp.ProtocolVersion,
				"capabilities":    map[string]any{},
				"clientInfo":      map[string]any{"name": "codegraph-purity-test", "version": "0.0.0"},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params":  map[string]any{"name": tool, "arguments": map[string]any{}},
		},
	} {
		if err := enc.Encode(req); err != nil {
			return fmt.Errorf("write request: %w", err)
		}
	}

	type line struct {
		raw []byte
		err error
	}
	lines := make(chan line)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			lines <- line{raw: append([]byte(nil), scanner.Bytes()...)}
		}
		lines <- line{err: fmt.Errorf("stdout closed before both responses (scanner error: %v)", scanner.Err())}
	}()

	seen := map[float64]bool{}
	for !seen[1] || !seen[2] {
		select {
		case ln := <-lines:
			if ln.err != nil {
				return ln.err
			}
			var frame struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      float64         `json:"id"`
				Error   json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(ln.raw, &frame); err != nil || frame.JSONRPC == "" {
				return fmt.Errorf("non-JSON-RPC line on stdout: %q", ln.raw)
			}
			if len(frame.Error) > 0 {
				return fmt.Errorf("response %v is a JSON-RPC error: %s", frame.ID, frame.Error)
			}
			fmt.Printf("%s\n", ln.raw)
			seen[frame.ID] = true
		case <-ctx.Done():
			return fmt.Errorf("no responses for both requests after 30s (initialize=%v tools/call=%v)", seen[1], seen[2])
		}
	}
	return nil
}
