package semanticeval

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, raw, 0o644)
	})
}

// copySources copies a source tree without its index.
func copySources(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".codegraph" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), raw, 0o644)
	})
}

// Digest hashes the relative paths and bytes of a source tree.
func Digest(dir string) (string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			paths = append(paths, p)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return "", err
		}
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, rel+"\x00")
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Evaluate runs one case in scratch. An error is an infrastructure failure.
// binary is the codegraph command for query cases.
func Evaluate(c *Case, binary, scratch string, now time.Time) (CaseResult, error) {
	ref := ReferenceFor(c.Language)
	res := CaseResult{
		ID: c.ID, Language: c.Language, Partition: c.Partition, Dimension: c.Dimension,
		Construct: c.Construct, Status: c.Status, TargetSpec: c.TargetSpec, Evidence: c.Evidence,
		ExpectationRevision: c.Expectation, Labeling: c.Labeling, BuildContext: c.BuildContext,
		Reference: ref.Name(),
	}
	digest, err := Digest(c.SourceDir())
	if err != nil {
		return res, err
	}
	res.SourceDigest = digest

	// javac resolves symlinks in its sourceroot, so paths must start resolved.
	scratch, err = filepath.EvalSymlinks(scratch)
	if err != nil {
		return res, err
	}
	work := filepath.Join(scratch, "src")
	if err := copyTree(c.SourceDir(), work); err != nil {
		return res, err
	}
	refDir := filepath.Join(scratch, "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		return res, err
	}
	obs, err := Observe(work, filepath.Join(scratch, "store"), c.Sources)
	if err != nil {
		return res, err
	}
	if len(obs.Unindexed) > 0 {
		res.Unindexed = obs.Unindexed
	}
	if c.Status == StatusUnsupported || c.Status == StatusUnevaluated {
		for _, d := range obs.Declarations {
			if c.HasRole(d.Role) {
				res.ObservedOutsideGate++
			}
		}
		return res, nil
	}

	if c.ReviewedOnly == "" {
		occs, err := ref.Index(work, refDir, c)
		if err != nil {
			return res, err
		}
		q, err := Qualify(c, occs)
		if err != nil {
			return res, err
		}
		res.Failures = append(res.Failures, q.Failures...)
		if len(q.Excluded) > 0 {
			res.ExcludedFacts = q.Excluded
		}
	}

	res.ParseFailures = c.ParseFailures
	var cmp *Comparison
	switch c.Dimension {
	case "queries":
		cmp, err = CompareQueries(c, binary, work, scratch)
	case "lifecycle":
		res.ExcludedCategories = LifecycleExclusions
		cmp, err = CompareLifecycle(c, binary, work, scratch)
	default:
		cmp = Compare(c, obs)
	}
	if err != nil {
		return res, err
	}
	res.Counts = cmp.Counts
	for path, want := range c.Excluded {
		if got := obs.Unindexed[path]; got != want {
			cmp.Mismatches = append(cmp.Mismatches, fmt.Sprintf("%s %s want=%s got=%s", WrongExclusion, path, want, got))
		}
	}
	for path, got := range obs.Unindexed {
		if _, ok := c.Excluded[path]; !ok {
			cmp.Mismatches = append(cmp.Mismatches, fmt.Sprintf("%s %s want=indexed got=%s", WrongExclusion, path, got))
		}
	}
	sort.Strings(cmp.Mismatches)
	res.Mismatches = cmp.Mismatches
	if c.Exception != nil {
		res.ExceptionIssue = c.Exception.Issue
	}
	res.Failures = append(res.Failures, Gate(c, cmp.Mismatches, now)...)
	return res, nil
}
