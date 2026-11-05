// Package semanticeval compares CodeGraph's facts and query answers with
// authored expectations that independent reference tools qualify.
package semanticeval

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// ManifestVersion is the case.json format this package reads.
const ManifestVersion = 1

// Case statuses.
const (
	StatusSupported   = "supported"
	StatusKnownGap    = "known-gap"
	StatusUnsupported = "unsupported"
	StatusUnevaluated = "unevaluated"
)

// Manifest partitions.
const (
	PartitionDevelopment = "development"
	PartitionAcceptance  = "acceptance"
)

// Labeling states how much of a case's source the expectations cover.
const (
	// LabelingComplete makes an unlabeled in-scope observation a false positive.
	LabelingComplete = "complete"
	// LabelingSampled makes an unlabeled in-scope observation unknown.
	LabelingSampled = "sampled"
)

// Languages lists the evaluated languages. Each must contribute inputs.
var Languages = []string{"go", "typescript", "javascript", "tsx", "python", "java"}

// Dimensions lists the design's required inventory dimensions.
var Dimensions = []string{"declarations", "resolution", "source-boundaries", "lifecycle", "queries"}

// Position is a source location: a 1-based line and a 0-based UTF-8 byte column.
type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

// Span is a half-open source range between two positions.
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// ExpectedDeclaration is one declaration the source establishes.
type ExpectedDeclaration struct {
	// ID names the declaration inside its case. Relationships refer to it.
	ID    string `json:"id"`
	Path  string `json:"path"`
	Owner string `json:"owner"`
	Role  string `json:"role"`
	Name  string `json:"name"`
	// At locates the identifier. It must hold Name in the source.
	At Position `json:"at"`
	// SpanRule names the construct rule in testdata/semantic/README.md.
	SpanRule string `json:"spanRule"`
	Span     Span   `json:"span"`
	// Evidence is "reference" or "reviewed". Reviewed facts skip the reference check.
	Evidence string `json:"evidence,omitempty"`
}

// ExpectedRelationship is one graph-level edge between two declarations.
type ExpectedRelationship struct {
	Source string `json:"source"`
	Kind   string `json:"kind"`
	Target string `json:"target"`
	// Site locates a target occurrence in Path. Sites let the reference
	// confirm the target. The call kind itself is a reviewed annotation.
	Path     string    `json:"path,omitempty"`
	Site     *Position `json:"site,omitempty"`
	Evidence string    `json:"evidence,omitempty"`
}

// Exception lists the exact differences that a known-gap case carries.
type Exception struct {
	Issue      string   `json:"issue"`
	Owner      string   `json:"owner"`
	ReviewBy   string   `json:"reviewBy"`
	Mismatches []string `json:"mismatches"`
}

// ReferenceExclusion keeps a reference fact outside the evaluated scope.
type ReferenceExclusion struct {
	Symbol string `json:"symbol"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Scope selects the observations a case evaluates.
type Scope struct {
	Roles             []string `json:"roles"`
	RelationshipKinds []string `json:"relationshipKinds"`
}

// Case is one authored case.json.
type Case struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Language      string   `json:"language"`
	Partition     string   `json:"partition"`
	Dimension     string   `json:"dimension"`
	Construct     string   `json:"construct"`
	Status        string   `json:"status"`
	TargetSpec    string   `json:"targetSpec"`
	Evidence      string   `json:"evidence"`
	Expectation   int      `json:"expectationRevision"`
	Labeling      string   `json:"labeling"`
	BuildContext  string   `json:"buildContext,omitempty"`
	Sources       []string `json:"sources"`
	Scope         Scope    `json:"scope"`

	Declarations  []ExpectedDeclaration  `json:"declarations"`
	Relationships []ExpectedRelationship `json:"relationships"`
	// Forbidden edges count as false positives when CodeGraph emits them.
	Forbidden []ExpectedRelationship `json:"forbidden,omitempty"`
	// Unknown edges exceed reference evidence. They enter no denominator.
	Unknown            []ExpectedRelationship `json:"unknown,omitempty"`
	ReferenceExclusion []ReferenceExclusion   `json:"referenceExclusions,omitempty"`
	Queries            []QueryCase            `json:"queries,omitempty"`
	Lifecycle          []LifecycleStep        `json:"lifecycle,omitempty"`
	// ReviewedOnly says why no reference can read this case, such as source
	// that does not parse. Every fact in such a case is a reviewed annotation.
	ReviewedOnly string `json:"reviewedOnly,omitempty"`
	// Excluded maps source files to the exclusion reason CodeGraph must record.
	Excluded map[string]string `json:"excluded,omitempty"`
	// ParseFailures names source files with syntax errors.
	ParseFailures []string   `json:"parseFailures,omitempty"`
	Exception     *Exception `json:"exception,omitempty"`

	// Dir is the directory that holds case.json.
	Dir string `json:"-"`
}

// SourceDir is the directory of the case's source tree.
func (c *Case) SourceDir() string { return filepath.Join(c.Dir, "src") }

// HasRole reports whether role is in the case's scope.
func (c *Case) HasRole(role string) bool { return slices.Contains(c.Scope.Roles, role) }

// HasKind reports whether an edge kind is in the case's scope.
func (c *Case) HasKind(kind string) bool { return slices.Contains(c.Scope.RelationshipKinds, kind) }

// LoadCases reads every case.json below root that belongs to partitions.
func LoadCases(root string, partitions []string) ([]*Case, error) {
	var paths []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "case.json" {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	var cases []*Case
	seen := map[string]string{}
	var errs []error
	for _, p := range paths {
		c, err := loadCase(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			continue
		}
		if prior, dup := seen[c.ID]; dup {
			errs = append(errs, fmt.Errorf("%s: case id %s already used by %s", p, c.ID, prior))
			continue
		}
		seen[c.ID] = p
		if slices.Contains(partitions, c.Partition) {
			cases = append(cases, c)
		}
	}
	return cases, errors.Join(errs...)
}

func loadCase(path string) (*Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var c Case
	if err := dec.Decode(&c); err != nil {
		return nil, err
	}
	c.Dir = filepath.Dir(path)
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

var reviewDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (c *Case) validate() error {
	var errs []error
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if c.SchemaVersion != ManifestVersion {
		bad("schemaVersion %d, want %d", c.SchemaVersion, ManifestVersion)
	}
	if c.ID == "" || c.Construct == "" || c.TargetSpec == "" || c.Evidence == "" {
		bad("id, construct, targetSpec and evidence are required")
	}
	if c.Expectation < 1 {
		bad("expectationRevision must be at least 1")
	}
	if !slices.Contains(Languages, c.Language) {
		bad("language %q is not evaluated", c.Language)
	}
	if !slices.Contains(Dimensions, c.Dimension) {
		bad("dimension %q is not in the inventory", c.Dimension)
	}
	if c.Partition != PartitionDevelopment && c.Partition != PartitionAcceptance {
		bad("partition %q is invalid", c.Partition)
	}
	if c.Labeling != LabelingComplete && c.Labeling != LabelingSampled {
		bad("labeling %q is invalid", c.Labeling)
	}
	switch c.Status {
	case StatusSupported, StatusUnsupported, StatusUnevaluated:
		if c.Exception != nil {
			bad("only a known-gap case can carry an exception, this one is %s", c.Status)
		}
	case StatusKnownGap:
		if c.Exception == nil {
			bad("a known-gap case needs an exception")
		} else {
			if c.Exception.Issue == "" || c.Exception.Owner == "" {
				bad("an exception needs an issue and an owner")
			}
			if !reviewDate.MatchString(c.Exception.ReviewBy) {
				bad("exception reviewBy %q is not YYYY-MM-DD", c.Exception.ReviewBy)
			}
			if len(c.Exception.Mismatches) == 0 {
				bad("an exception needs at least one exact mismatch")
			}
		}
	default:
		bad("status %q is invalid", c.Status)
	}
	if len(c.Sources) == 0 {
		bad("sources must not be empty")
	}
	for _, s := range c.Sources {
		if _, err := os.Stat(filepath.Join(c.SourceDir(), s)); err != nil {
			bad("source %s: %v", s, err)
		}
	}

	if c.Dimension == "queries" && len(c.Queries) == 0 {
		bad("a queries case needs at least one query")
	}
	if c.Dimension == "lifecycle" && len(c.Lifecycle) == 0 {
		bad("a lifecycle case needs at least one step")
	}

	ids := map[string]bool{}
	for _, d := range c.Declarations {
		if d.ID == "" || ids[d.ID] {
			bad("declaration id %q is empty or repeated", d.ID)
		}
		ids[d.ID] = true
		if d.SpanRule == "" {
			bad("declaration %s has no span rule", d.ID)
		}
		if err := c.checkIdentifier(d); err != nil {
			errs = append(errs, err)
		}
	}
	for _, list := range [][]ExpectedRelationship{c.Relationships, c.Forbidden, c.Unknown} {
		for _, r := range list {
			for _, end := range []string{r.Source, r.Target} {
				if !ids[end] && !strings.HasPrefix(end, "file:") && !strings.HasPrefix(end, "external:") {
					bad("relationship endpoint %q is not a declaration", end)
				}
			}
		}
	}
	return errors.Join(errs...)
}

// checkIdentifier confirms that the authored position holds the authored name.
func (c *Case) checkIdentifier(d ExpectedDeclaration) error {
	raw, err := os.ReadFile(filepath.Join(c.SourceDir(), d.Path))
	if err != nil {
		return fmt.Errorf("declaration %s: %w", d.ID, err)
	}
	lines := strings.Split(string(raw), "\n")
	if d.At.Line < 1 || d.At.Line > len(lines) {
		return fmt.Errorf("declaration %s: line %d is outside %s", d.ID, d.At.Line, d.Path)
	}
	line := lines[d.At.Line-1]
	if d.At.Col < 0 || d.At.Col+len(d.Name) > len(line) || line[d.At.Col:d.At.Col+len(d.Name)] != d.Name {
		return fmt.Errorf("declaration %s: %s:%d:%d does not hold %q", d.ID, d.Path, d.At.Line, d.At.Col, d.Name)
	}
	return nil
}

// ExceptionExpired reports whether the exception's review date has passed.
func ExceptionExpired(e *Exception, now time.Time) bool {
	due, err := time.Parse(time.DateOnly, e.ReviewBy)
	if err != nil {
		return true
	}
	return now.After(due.AddDate(0, 0, 1))
}
