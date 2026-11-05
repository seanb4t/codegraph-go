//go:build semantic

package semanticeval

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	partitionsEnv = "CODEGRAPH_SEMANTIC_PARTITIONS"
	reportDirEnv  = "CODEGRAPH_SEMANTIC_REPORT_DIR"
	casesRoot     = "../../testdata/semantic"
)

var (
	mu             sync.Mutex
	results        []CaseResult
	tools          = map[string]string{}
	infrastructure []string
	missing        []string
)

func partitions() []string {
	v := os.Getenv(partitionsEnv)
	if v == "" {
		return []string{PartitionDevelopment, PartitionAcceptance}
	}
	return strings.Split(v, ",")
}

var selectedCases = sync.OnceValues(func() ([]*Case, error) { return LoadCases(casesRoot, partitions()) })

func TestMain(m *testing.M) {
	status := m.Run()
	all, err := LoadCases(casesRoot, []string{PartitionDevelopment, PartitionAcceptance})
	if err != nil {
		infrastructure = append(infrastructure, "load cases: "+err.Error())
	}
	for _, dim := range Dimensions {
		if !slices.ContainsFunc(all, func(c *Case) bool { return c.Dimension == dim }) {
			missing = append(missing, "dimension "+dim+" has no case")
		}
	}
	report := Build(results, tools, partitions(), infrastructure, missing)
	report.WriteText(os.Stdout)
	if dir := os.Getenv(reportDirEnv); dir != "" {
		if err := report.Write(dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
	if report.Failed() {
		status = 1
	}
	os.Exit(status)
}

func infrastructureFailure(t *testing.T, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	mu.Lock()
	infrastructure = append(infrastructure, msg)
	mu.Unlock()
	t.Error(msg)
}

// runLanguage evaluates the selected cases of languages. Every language must
// contribute at least one selected case that the gate evaluates.
func runLanguage(t *testing.T, languages ...string) {
	t.Helper()
	runCases(t, languages, func(c *Case) bool { return c.Dimension != "queries" && c.Dimension != "lifecycle" }, "")
}

// runCases evaluates the selected cases that keep accepts.
func runCases(t *testing.T, languages []string, keep func(*Case) bool, binary string) {
	t.Helper()
	cases, err := selectedCases()
	if err != nil {
		t.Errorf("load cases: %v", err)
		return
	}
	now := time.Now()
	for _, lang := range languages {
		ref := ReferenceFor(lang)
		version, err := ref.Version()
		if err != nil {
			infrastructureFailure(t, "%s: %v", ref.Name(), err)
			continue
		}
		mu.Lock()
		tools[ref.Name()] = version
		mu.Unlock()

		selected, gated := 0, 0
		for _, c := range cases {
			if c.Language != lang || !keep(c) {
				continue
			}
			selected++
			if c.Status == StatusSupported || c.Status == StatusKnownGap {
				gated++
			}
			t.Run(c.ID, func(t *testing.T) {
				res, err := Evaluate(c, binary, t.TempDir(), now)
				if err != nil {
					infrastructureFailure(t, "%s: %v", c.ID, err)
					return
				}
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
				for _, f := range res.Failures {
					t.Error(f)
				}
			})
		}
		if selected == 0 || gated == 0 {
			msg := fmt.Sprintf("language %s: %d selected cases, %d gated, in partitions %v", lang, selected, gated, partitions())
			mu.Lock()
			missing = append(missing, msg)
			mu.Unlock()
			t.Error(msg)
		}
	}
}

func TestSemanticGo(t *testing.T)         { runLanguage(t, "go") }
func TestSemanticTypeScript(t *testing.T) { runLanguage(t, "typescript", "javascript", "tsx") }
func TestSemanticPython(t *testing.T)     { runLanguage(t, "python") }
func TestSemanticJava(t *testing.T)       { runLanguage(t, "java") }

func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "codegraph")
	build := exec.Command("go", "build", "-o", binary, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := build.CombinedOutput(); err != nil {
		infrastructureFailure(t, "build codegraph: %v: %s", err, out)
	}
	return binary
}

func TestSemanticLifecycle(t *testing.T) {
	runCases(t, []string{"go", "typescript", "python", "java"}, func(c *Case) bool { return c.Dimension == "lifecycle" }, buildBinary(t))
}

func TestSemanticPublicQueries(t *testing.T) {
	runCases(t, []string{"go", "typescript", "python", "java"}, func(c *Case) bool { return c.Dimension == "queries" }, buildBinary(t))
}
