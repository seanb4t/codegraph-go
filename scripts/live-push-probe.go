//go:build ignore

// Command live-push-probe is a throwaway diagnostic harness for 06-07's
// real three-process concurrency gate (criterion 5). It is excluded from
// the shipped build by the //go:build ignore tag above — `go build ./...`
// and `go vet ./...` both skip it, and `go list ./...` never names it as
// part of any package — while `go run scripts/live-push-probe.go <cmd>`
// still compiles and runs it in full module context, with access to this
// module's own generated Connect client and its own go.mod dependencies.
// No new dependency is introduced: it uses connectrpc.com/connect (already
// required for the 13 existing rpcs) and github.com/modelcontextprotocol/
// go-sdk/mcp (already required by internal/mcp).
//
// Two subcommands, each printing exactly one JSON object to stdout on exit
// and nothing else there, so a calling shell script can parse the result
// without scraping prose:
//
//	stream --url <base> --out <file> --seconds N
//	    Opens WatchGraph with the real generated uiv1connect client against
//	    the UI's own base URL and appends one JSON line per received event
//	    to <file> as it arrives (flushed immediately, so a killed process
//	    still leaves every event it actually received on disk). Raw curl
//	    cannot do this: Connect frames each streamed message in a
//	    length-prefixed envelope, so counting messages or reading
//	    generations out of a curl response body requires decoding those
//	    envelopes. The generated client already handles framing and
//	    incremental delivery.
//
//	mcp --path <repo> --seconds N [--bin <codegraph-binary>]
//	    Spawns "codegraph serve --mcp --path <repo>" and holds its own
//	    stdin/stdout PIPES open for the whole run — this process (not a
//	    one-shot invocation) is genuinely one of the concurrent processes
//	    criterion 5 is about, for its entire duration. `serve --mcp` is
//	    stdio-only (internal/cli/serve.go:179 — "over stdio", no socket),
//	    so there is no port to poll; holding the pipes open is the only
//	    way to keep this a real, live session. After the hold period it
//	    performs one real codegraph_status tool call over the same MCP
//	    session established at startup and prints a JSON verdict carrying
//	    the tool's own response text (never a boolean the probe made up
//	    about itself) plus the CHILD process's PID (not this probe's own),
//	    since the concurrency-check script needs the real product
//	    process's PID for its own liveness bookkeeping.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "stream":
		runStream(os.Args[2:])
	case "mcp":
		runMCP(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "live-push-probe: unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `live-push-probe: a real Connect stream client and a real persistent MCP harness

Subcommands:

  stream --url <base> --out <file> --seconds N
      Open WatchGraph over the real generated uiv1connect client against
      <base> (the UI's own printed URL) and append one JSON line per
      received event to <file>, flushed as each arrives.

  mcp --path <repo> --seconds N [--bin <codegraph-binary>]
      Spawn "codegraph serve --mcp --path <repo>", hold stdin/stdout
      pipes open for N seconds, then issue one codegraph_status tool
      call over the same session and print a JSON verdict.`)
}

func printJSON(v map[string]any) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "live-push-probe: failed to encode verdict: %v\n", err)
	}
}

// runStream implements the `stream` subcommand.
func runStream(args []string) {
	fs := flag.NewFlagSet("stream", flag.ExitOnError)
	url := fs.String("url", "", "the UI's own base URL (e.g. http://127.0.0.1:PORT)")
	out := fs.String("out", "", "output file for one JSON line per received event")
	seconds := fs.Int("seconds", 30, "how long to hold the WatchGraph session open")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *url == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "stream: --url and --out are both required")
		os.Exit(2)
	}

	verdict := map[string]any{"subcommand": "stream", "url": *url}

	f, err := os.Create(*out)
	if err != nil {
		verdict["error"] = err.Error()
		printJSON(verdict)
		os.Exit(1)
	}
	defer f.Close()

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, *url)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*seconds)*time.Second)
	defer cancel()

	stream, err := client.WatchGraph(ctx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		verdict["error"] = fmt.Sprintf("WatchGraph: %v", err)
		verdict["eventsReceived"] = 0
		printJSON(verdict)
		return
	}

	count := 0
	for stream.Receive() {
		ev := stream.Msg()
		line := map[string]any{
			"generation":       ev.GetGeneration(),
			"initialized":      ev.GetInitialized(),
			"receivedAtUnixMs": time.Now().UnixMilli(),
		}
		b, marshalErr := json.Marshal(line)
		if marshalErr != nil {
			continue
		}
		f.Write(b)
		f.Write([]byte("\n"))
		f.Sync() // every line durable immediately: a killed process leaves everything it actually saw
		count++
	}

	verdict["eventsReceived"] = count
	if streamErr := stream.Err(); streamErr != nil &&
		!errors.Is(streamErr, context.DeadlineExceeded) &&
		!errors.Is(streamErr, context.Canceled) {
		verdict["error"] = streamErr.Error()
	}
	printJSON(verdict)
}

// runMCP implements the `mcp` subcommand.
func runMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	path := fs.String("path", "", "repo path passed to codegraph serve --mcp")
	seconds := fs.Int("seconds", 30, "how long to hold the MCP session open before calling a tool")
	bin := fs.String("bin", "codegraph", "path to (or name of) the codegraph binary")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *path == "" {
		fmt.Fprintln(os.Stderr, "mcp: --path is required")
		os.Exit(2)
	}

	verdict := map[string]any{"subcommand": "mcp", "path": *path}

	cmd := exec.Command(*bin, "serve", "--mcp", "--path", *path)

	// Real, persistent stdin/stdout PIPES — not a one-shot invocation.
	// This is what lets the process stay a genuine concurrent participant
	// for the whole hold period rather than a single request/response.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		verdict["error"] = fmt.Sprintf("StdoutPipe: %v", err)
		printJSON(verdict)
		os.Exit(1)
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		verdict["error"] = fmt.Sprintf("StdinPipe: %v", err)
		printJSON(verdict)
		os.Exit(1)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		verdict["error"] = fmt.Sprintf("Start: %v", err)
		printJSON(verdict)
		os.Exit(1)
	}
	verdict["serveMcpPid"] = cmd.Process.Pid

	transport := &mcp.IOTransport{Reader: stdoutPipe, Writer: stdinPipe}
	client := mcp.NewClient(&mcp.Implementation{Name: "live-push-probe", Version: "0.0.0"}, nil)

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 15*time.Second)
	session, err := client.Connect(connectCtx, transport, nil)
	cancelConnect()
	if err != nil {
		verdict["connected"] = false
		verdict["error"] = fmt.Sprintf("initialize handshake: %v", err)
		printJSON(verdict)
		_ = cmd.Process.Kill()
		return
	}
	verdict["connected"] = true

	// Hold the session — and the child process — open for the whole run.
	// This sleep IS the "concurrent for N seconds" property: the child's
	// stdin/stdout pipes stay open, and the child stays a live MCP server
	// against the shared store, for this entire duration.
	time.Sleep(time.Duration(*seconds) * time.Second)

	callCtx, cancelCall := context.WithTimeout(context.Background(), 15*time.Second)
	result, callErr := session.CallTool(callCtx, &mcp.CallToolParams{Name: "codegraph_status"})
	cancelCall()

	switch {
	case callErr != nil:
		verdict["error"] = fmt.Sprintf("CallTool(codegraph_status): %v", callErr)
		verdict["toolCallResponse"] = ""
	default:
		verdict["toolCallResponse"] = extractText(result)
		verdict["toolCallIsError"] = result.IsError
	}

	_ = session.Close()
	printJSON(verdict)
	// The child is left running deliberately: the caller manages its
	// lifecycle (liveness bookkeeping, then teardown) via the PID already
	// reported above. This probe's own exit does not, by itself, kill it.
}

// extractText concatenates every TextContent block in a CallToolResult —
// codegraph_status returns its answer as text content, and this is the
// non-empty-string proof of a real tool-call RESPONSE the concurrency
// gate's acceptance criteria require, never a boolean the probe made up
// about itself.
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}
