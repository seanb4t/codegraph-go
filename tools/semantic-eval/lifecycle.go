package semanticeval

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// LifecycleFacts is the fact-kind key of fresh-versus-incremental counts.
const LifecycleFacts = "lifecycle"

// DriftedFact is the mismatch kind of a fact only one path has.
const DriftedFact = "lifecycle-drift"

// LifecycleExclusions names the categories outside the lifecycle comparison.
var LifecycleExclusions = []string{"route nodes", "route edges", "edge provenance and call-site positions", "internal node ids"}

// LifecycleStep changes the source tree before one synchronization.
type LifecycleStep struct {
	Name   string            `json:"name"`
	Write  map[string]string `json:"write,omitempty"`
	Rename map[string]string `json:"rename,omitempty"`
	Delete []string          `json:"delete,omitempty"`
}

func (s LifecycleStep) apply(dir string) error {
	for from, to := range s.Rename {
		if err := os.Rename(filepath.Join(dir, from), filepath.Join(dir, to)); err != nil {
			return err
		}
	}
	for _, p := range s.Delete {
		if err := os.Remove(filepath.Join(dir, p)); err != nil {
			return err
		}
	}
	for p, text := range s.Write {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// semanticFacts renders observations without internal ids, so a fresh and an
// incremental store compare by meaning.
func semanticFacts(o *Observed, keep func(string) bool) []string {
	label := map[string]string{}
	var facts []string
	for _, d := range o.Declarations {
		key := fmt.Sprintf("%s/%s/%s/%s", d.Path, d.Owner, d.Role, d.Name)
		label[d.NodeID] = key
		facts = append(facts, fmt.Sprintf("declaration %s %d:%d-%d:%d", key, d.Span.Start.Line, d.Span.Start.Col, d.Span.End.Line, d.Span.End.Col))
	}
	for _, r := range o.Relationships {
		s, sok := label[r.Source]
		t, tok := label[r.Target]
		if sok && tok && keep(r.Kind) {
			facts = append(facts, fmt.Sprintf("relationship %s %s %s", s, r.Kind, t))
		}
	}
	sort.Strings(facts)
	return slices.Compact(facts)
}

// CompareLifecycle applies each step and compares an incremental store with a
// fresh index of the same tree after every synchronization.
//
// Each step also compares the case's queries through the CLI, because agreement
// between the stores does not show that the answers built from them agree.
func CompareLifecycle(c *Case, binary, work, scratch string) (*Comparison, error) {
	res := &Comparison{Counts: map[string]Counts{}}
	if err := runBinary(binary, work, scratch, "init", "--quiet"); err != nil {
		return nil, err
	}
	incremental := filepath.Join(scratch, "incremental")
	if _, err := indexer.Run(work, incremental, indexer.Options{Quiet: true}); err != nil {
		return nil, err
	}
	keep := c.HasKind
	for i, step := range c.Lifecycle {
		if err := step.apply(work); err != nil {
			return nil, fmt.Errorf("step %s: %w", step.Name, err)
		}
		if _, err := indexer.Sync(work, incremental, indexer.Options{Quiet: true}); err != nil {
			return nil, fmt.Errorf("step %s: sync: %w", step.Name, err)
		}
		fresh := filepath.Join(scratch, fmt.Sprintf("fresh-%d", i))
		if _, err := indexer.Run(work, fresh, indexer.Options{Quiet: true}); err != nil {
			return nil, fmt.Errorf("step %s: fresh: %w", step.Name, err)
		}
		a, err := ReadStore(incremental, nil)
		if err != nil {
			return nil, err
		}
		b, err := ReadStore(fresh, nil)
		if err != nil {
			return nil, err
		}
		inc, full := semanticFacts(a, keep), semanticFacts(b, keep)
		if len(full) == 0 {
			return nil, fmt.Errorf("step %s: the fresh index has no evaluated fact", step.Name)
		}
		n := res.Counts[LifecycleFacts]
		for _, f := range full {
			if slices.Contains(inc, f) {
				n.TruePositives++
			} else {
				n.FalseNegatives++
				res.Mismatches = append(res.Mismatches, fmt.Sprintf("%s %s only-fresh %s", DriftedFact, step.Name, f))
			}
		}
		for _, f := range inc {
			if !slices.Contains(full, f) {
				n.FalsePositives++
				res.Mismatches = append(res.Mismatches, fmt.Sprintf("%s %s only-incremental %s", DriftedFact, step.Name, f))
			}
		}
		res.Counts[LifecycleFacts] = n
		if err := compareAnswers(res, c, step, binary, work, filepath.Join(scratch, fmt.Sprintf("answers-%d", i)), scratch); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func runBinary(binary, dir, scratch string, args ...string) error {
	cmd := exec.Command(binary, args...)
	cmd.Dir, cmd.Env = dir, queryEnv(scratch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("codegraph %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return nil
}

// compareAnswers synchronizes work and asks a fresh copy of the tree the same
// queries. The copy leaves .codegraph behind, so init indexes it from scratch.
func compareAnswers(res *Comparison, c *Case, step LifecycleStep, binary, work, freshDir, scratch string) error {
	if err := runBinary(binary, work, scratch, "sync", "--quiet"); err != nil {
		return err
	}
	if err := copySources(work, freshDir); err != nil {
		return err
	}
	if err := runBinary(binary, freshDir, scratch, "init", "--quiet"); err != nil {
		return err
	}
	n := res.Counts[LifecycleFacts]
	for _, q := range c.Queries {
		inc, err := cliAnswer(binary, work, scratch, q, q.MaxFiles)
		if err != nil {
			return err
		}
		fresh, err := cliAnswer(binary, freshDir, scratch, q, q.MaxFiles)
		if err != nil {
			return err
		}
		if inc == fresh {
			n.TruePositives++
			continue
		}
		n.FalsePositives++
		n.FalseNegatives++
		res.Mismatches = append(res.Mismatches, fmt.Sprintf("%s %s answer %s incremental=%q fresh=%q", DriftedFact, step.Name, q.ID, inc, fresh))
	}
	res.Counts[LifecycleFacts] = n
	return nil
}
