package semanticeval

import (
	"fmt"
	"slices"
	"time"
)

// Gate returns why a case fails its status. Accuracy totals keep every
// mismatch whatever this returns.
func Gate(c *Case, mismatches []string, now time.Time) []string {
	var fails []string
	switch c.Status {
	case StatusSupported:
		for _, m := range mismatches {
			fails = append(fails, "supported case has "+m)
		}
	case StatusKnownGap:
		e := c.Exception
		if ExceptionExpired(e, now) {
			fails = append(fails, fmt.Sprintf("exception for %s expired on %s", e.Issue, e.ReviewBy))
		}
		for _, m := range mismatches {
			if !slices.Contains(e.Mismatches, m) {
				fails = append(fails, "new mismatch outside the exception: "+m)
			}
		}
		for _, m := range e.Mismatches {
			if !slices.Contains(mismatches, m) {
				fails = append(fails, "stale exception, remove the repaired mismatch: "+m)
			}
		}
		if len(mismatches) == 0 {
			fails = append(fails, "no mismatch remains: remove the exception and mark the case supported")
		}
	}
	return fails
}
