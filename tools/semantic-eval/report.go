package semanticeval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReportVersion is the format of the JSON report.
const ReportVersion = 1

// CaseResult is one evaluated case.
type CaseResult struct {
	ID                  string            `json:"id"`
	Language            string            `json:"language"`
	Partition           string            `json:"partition"`
	Dimension           string            `json:"dimension"`
	Construct           string            `json:"construct"`
	Status              string            `json:"status"`
	TargetSpec          string            `json:"targetSpec"`
	Evidence            string            `json:"evidence"`
	ExpectationRevision int               `json:"expectationRevision"`
	Labeling            string            `json:"labeling"`
	BuildContext        string            `json:"buildContext,omitempty"`
	Reference           string            `json:"reference"`
	SourceDigest        string            `json:"sourceDigest"`
	Counts              map[string]Counts `json:"counts,omitempty"`
	Mismatches          []string          `json:"mismatches,omitempty"`
	ExceptionIssue      string            `json:"exceptionIssue,omitempty"`
	Unindexed           map[string]string `json:"unindexed,omitempty"`
	ParseFailures       []string          `json:"parseFailures,omitempty"`
	ExcludedCategories  []string          `json:"excludedCategories,omitempty"`
	ExcludedFacts       map[string]int    `json:"excludedFacts,omitempty"`
	ObservedOutsideGate int               `json:"observedOutsideGate,omitempty"`
	Failures            []string          `json:"failures,omitempty"`
}

// Total is the accuracy of one language and fact kind.
type Total struct {
	Language string `json:"language"`
	Fact     string `json:"fact"`
	Counts
	PrecisionDenominator int `json:"precisionDenominator"`
	RecallDenominator    int `json:"recallDenominator"`
	// Precision and Recall are null when their denominator is zero.
	Precision *float64 `json:"precision"`
	Recall    *float64 `json:"recall"`
}

// Report is the versioned evaluation report.
type Report struct {
	SchemaVersion  int                       `json:"schemaVersion"`
	Partitions     []string                  `json:"partitions"`
	Tools          map[string]string         `json:"tools"`
	Totals         []Total                   `json:"totals"`
	StatusCounts   map[string]map[string]int `json:"statusCounts"`
	PartitionSize  map[string]int            `json:"partitionSizes"`
	Infrastructure []string                  `json:"infrastructureFailures,omitempty"`
	MissingInputs  []string                  `json:"missingInputs,omitempty"`
	Cases          []CaseResult              `json:"cases"`
}

func ratio(n, d int) *float64 {
	if d == 0 {
		return nil
	}
	v := float64(n) / float64(d)
	return &v
}

// Build assembles a report. Unsupported and unevaluated cases stay outside
// the totals.
func Build(results []CaseResult, tools map[string]string, partitions, infrastructure, missing []string) *Report {
	r := &Report{
		SchemaVersion: ReportVersion, Partitions: partitions, Tools: tools,
		StatusCounts: map[string]map[string]int{}, PartitionSize: map[string]int{},
		Infrastructure: infrastructure, MissingInputs: missing, Cases: results,
	}
	type key struct{ lang, fact string }
	sums := map[key]Counts{}
	for _, c := range results {
		if r.StatusCounts[c.Language] == nil {
			r.StatusCounts[c.Language] = map[string]int{}
		}
		r.StatusCounts[c.Language][c.Status]++
		r.PartitionSize[c.Partition]++
		if c.Status != StatusSupported && c.Status != StatusKnownGap {
			continue
		}
		for fact, n := range c.Counts {
			k := key{c.Language, fact}
			sum := sums[k]
			sum.add(n)
			sums[k] = sum
		}
	}
	for k, n := range sums {
		pd, rd := n.TruePositives+n.FalsePositives, n.TruePositives+n.FalseNegatives
		r.Totals = append(r.Totals, Total{
			Language: k.lang, Fact: k.fact, Counts: n,
			PrecisionDenominator: pd, RecallDenominator: rd,
			Precision: ratio(n.TruePositives, pd), Recall: ratio(n.TruePositives, rd),
		})
	}
	sort.Slice(r.Totals, func(i, j int) bool {
		if r.Totals[i].Language != r.Totals[j].Language {
			return r.Totals[i].Language < r.Totals[j].Language
		}
		return r.Totals[i].Fact < r.Totals[j].Fact
	})
	sort.Slice(r.Cases, func(i, j int) bool { return r.Cases[i].ID < r.Cases[j].ID })
	return r
}

// Failed reports whether the gate fails.
func (r *Report) Failed() bool {
	if len(r.Infrastructure) > 0 || len(r.MissingInputs) > 0 {
		return true
	}
	for _, c := range r.Cases {
		if len(c.Failures) > 0 {
			return true
		}
	}
	return false
}

func percent(v *float64) string {
	if v == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%.1f%%", *v*100)
}

// WriteText renders the short text form.
func (r *Report) WriteText(w io.Writer) {
	fmt.Fprintf(w, "semantic evaluation report v%d, partitions: %s\n", r.SchemaVersion, strings.Join(r.Partitions, ", "))
	tools := make([]string, 0, len(r.Tools))
	for name := range r.Tools {
		tools = append(tools, name)
	}
	sort.Strings(tools)
	for _, name := range tools {
		fmt.Fprintf(w, "tool %s: %s\n", name, r.Tools[name])
	}
	fmt.Fprintln(w, "\nlanguage fact precision recall tp fp fn unknown wrong-span")
	for _, t := range r.Totals {
		fmt.Fprintf(w, "%s %s %s (%d) %s (%d) %d %d %d %d %d\n", t.Language, t.Fact,
			percent(t.Precision), t.PrecisionDenominator, percent(t.Recall), t.RecallDenominator,
			t.TruePositives, t.FalsePositives, t.FalseNegatives, t.Unknown, t.WrongSpans)
	}
	fmt.Fprintln(w, "\nlanguage statuses")
	langs := make([]string, 0, len(r.StatusCounts))
	for lang := range r.StatusCounts {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		s := r.StatusCounts[lang]
		fmt.Fprintf(w, "%s supported=%d known-gap=%d unsupported=%d unevaluated=%d\n", lang,
			s[StatusSupported], s[StatusKnownGap], s[StatusUnsupported], s[StatusUnevaluated])
	}
	for _, m := range r.MissingInputs {
		fmt.Fprintf(w, "\nMISSING INPUT %s\n", m)
	}
	for _, f := range r.Infrastructure {
		fmt.Fprintf(w, "\nINFRASTRUCTURE FAILURE %s\n", f)
	}
	for _, c := range r.Cases {
		if len(c.Mismatches)+len(c.Failures)+len(c.Unindexed)+len(c.ParseFailures) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s [%s, %s, revision %d]\n", c.ID, c.Status, c.Partition, c.ExpectationRevision)
		for _, m := range c.Mismatches {
			fmt.Fprintf(w, "  mismatch %s\n", m)
		}
		for p, why := range c.Unindexed {
			fmt.Fprintf(w, "  unindexed %s: %s\n", p, why)
		}
		for _, p := range c.ParseFailures {
			fmt.Fprintf(w, "  parse failure %s\n", p)
		}
		for _, f := range c.Failures {
			fmt.Fprintf(w, "  FAIL %s\n", f)
		}
	}
}

// Write stores the JSON and text forms in dir.
func (r *Report) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "semantic-report.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "semantic-report.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	r.WriteText(f)
	return nil
}
