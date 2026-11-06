package semanticeval

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// QueryFacts is the fact-kind key of public query counts.
const QueryFacts = "query"

// Query interfaces.
const (
	InterfaceCLI = "cli"
	InterfaceMCP = "mcp"
)

// QueryCase is one reviewed public query. Required and Forbidden hold names
// the answer must or must not show as whole words.
type QueryCase struct {
	ID string `json:"id"`
	// Tool is node, callers, callees, impact or explore.
	Tool      string   `json:"tool"`
	Symbol    string   `json:"symbol"`
	File      string   `json:"file,omitempty"`
	Required  []string `json:"required"`
	Forbidden []string `json:"forbidden,omitempty"`
	// MaxFiles is explore's output budget. Zero means the tool default.
	MaxFiles int `json:"maxFiles,omitempty"`
}

// queryEnv keeps per-user state of the binary inside scratch.
func queryEnv(scratch string) []string {
	return append(os.Environ(),
		"HOME="+scratch,
		"XDG_STATE_HOME="+filepath.Join(scratch, "state"),
		"XDG_CONFIG_HOME="+filepath.Join(scratch, "config"),
	)
}

func cliAnswer(binary, dir, scratch string, q QueryCase, maxFiles int) (string, error) {
	args := []string{q.Tool}
	switch q.Tool {
	case "node":
		args = append(args, q.Symbol)
		if q.File != "" {
			args = append(args, "--file", q.File)
		}
	case "explore":
		args = append(args, q.Symbol)
		if maxFiles > 0 {
			args = append(args, "--max-files", strconv.Itoa(maxFiles))
		}
	default:
		args = append(args, q.Symbol, "--json")
	}
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = queryEnv(scratch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codegraph %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

func mcpArguments(q QueryCase, maxFiles int) map[string]any {
	switch q.Tool {
	case "explore":
		args := map[string]any{"query": q.Symbol}
		if maxFiles > 0 {
			args["max_files"] = maxFiles
		}
		return args
	case "node":
		args := map[string]any{"symbol": q.Symbol}
		if q.File != "" {
			args["file"] = q.File
		}
		return args
	}
	return map[string]any{"symbol": q.Symbol}
}

func mcpAnswer(ctx context.Context, s *mcp.ClientSession, q QueryCase, maxFiles int) (string, error) {
	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: "codegraph_" + q.Tool, Arguments: mcpArguments(q, maxFiles)})
	if err != nil {
		return "", fmt.Errorf("codegraph_%s: %w", q.Tool, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	if res.IsError {
		return "", fmt.Errorf("codegraph_%s: %s", q.Tool, b.String())
	}
	return b.String(), nil
}

func shows(answer, fact string) bool {
	return regexp.MustCompile(`(^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(fact) + `($|[^\p{L}\p{N}_])`).MatchString(answer)
}

// scoreAnswer checks one interface's answer. unbounded is consulted only when
// a budget applies, to tell truncation from a missing graph fact.
func scoreAnswer(res *Comparison, label string, q QueryCase, answer string, unbounded func() (string, error)) error {
	n := res.Counts[QueryFacts]
	for _, fact := range q.Required {
		if shows(answer, fact) {
			n.TruePositives++
			continue
		}
		n.FalseNegatives++
		kind := MissingQueryFact
		if q.MaxFiles > 0 {
			full, err := unbounded()
			if err != nil {
				return err
			}
			if shows(full, fact) {
				kind = TruncatedQueryFact
			}
		}
		res.Mismatches = append(res.Mismatches, fmt.Sprintf("%s %s %s", kind, label, fact))
	}
	for _, fact := range q.Forbidden {
		if shows(answer, fact) {
			n.FalsePositives++
			res.Mismatches = append(res.Mismatches, fmt.Sprintf("%s %s %s", ForbiddenQueryFact, label, fact))
		}
	}
	res.Counts[QueryFacts] = n
	return nil
}

// CompareQueries runs a case's queries through the CLI and the MCP server.
// Each interface is scored alone, so agreement between them proves nothing.
func CompareQueries(c *Case, binary, dir, scratch string) (*Comparison, error) {
	res := &Comparison{Counts: map[string]Counts{}}
	init := exec.Command(binary, "init", "--quiet")
	init.Dir, init.Env = dir, queryEnv(scratch)
	if out, err := init.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("codegraph init: %w: %s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "serve", "--mcp")
	cmd.Dir, cmd.Env = dir, queryEnv(scratch)
	client := mcp.NewClient(&mcp.Implementation{Name: "semantic-eval", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to codegraph serve --mcp: %w", err)
	}
	defer session.Close()

	for _, q := range c.Queries {
		answer, err := cliAnswer(binary, dir, scratch, q, q.MaxFiles)
		if err != nil {
			return nil, err
		}
		err = scoreAnswer(res, q.ID+"/"+InterfaceCLI, q, answer, func() (string, error) {
			return cliAnswer(binary, dir, scratch, q, 0)
		})
		if err != nil {
			return nil, err
		}
		answer, err = mcpAnswer(ctx, session, q, q.MaxFiles)
		if err != nil {
			return nil, err
		}
		err = scoreAnswer(res, q.ID+"/"+InterfaceMCP, q, answer, func() (string, error) {
			return mcpAnswer(ctx, session, q, 0)
		})
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
