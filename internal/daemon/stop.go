package daemon

import (
	"errors"
	"fmt"
	"path/filepath"
)

// stopSignal is sendStop indirected behind a package-level func var — the
// same test-seam convention as registryDir/onSync*/onWatchOpen elsewhere in
// this package — so tests can inject a capturing stub instead of delivering
// a real OS signal to the test process itself.
var stopSignal = sendStop

// StopAll signals every live daemon in the global registry (DMON-02,
// `daemon stop --all`). An empty registry is a clean no-op: (nil, nil), not
// an error.
func StopAll() ([]Record, error) {
	return stopTargets(func(Record) bool { return true })
}

// StopMatching signals every live daemon in the global registry whose
// RepoRoot resolves to the same path as repoRoot (DMON-02, `daemon stop
// -p/--path`). No match is a clean no-op: (nil, nil), not an error.
func StopMatching(repoRoot string) ([]Record, error) {
	want := resolveRepoRoot(repoRoot)
	return stopTargets(func(rec Record) bool {
		return resolveRepoRoot(rec.RepoRoot) == want
	})
}

// resolveRepoRoot normalizes p via filepath.EvalSymlinks for comparison
// (mirroring WORK-03's symlink handling). Best-effort: if EvalSymlinks
// errors — e.g. the path no longer exists on disk — the original string is
// used as-is, so a genuine mismatch still falls back to plain string
// comparison instead of failing the whole match.
func resolveRepoRoot(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// stopTargets is StopAll/StopMatching's shared engine. It starts from
// List() (which already self-heals — see registry.go), then for each
// matching candidate re-derives isStale immediately before signaling: a
// record found stale in this very narrow window between List()'s scan and
// this point is skipped, never signaled — the daemon-stop safety invariant
// (T-07-04-01) that a forged/stale/reused-pid record can never cause an
// unrelated process to be SIGTERM'd. Targets are de-duplicated by pid.
// Per-target sendStop errors are aggregated via errors.Join rather than
// swallowed or short-circuited on the first failure; a no-match/empty
// candidate set returns (nil, nil), not an error.
func stopTargets(match func(Record) bool) ([]Record, error) {
	recs, err := List()
	if err != nil {
		return nil, err
	}

	var signaled []Record
	seen := make(map[int]bool)
	var errs []error
	for _, rec := range recs {
		if !match(rec) || seen[rec.PID] {
			continue
		}
		seen[rec.PID] = true

		if isStale(lockInfo{PID: rec.PID, StartedAt: rec.StartedAt}) {
			continue
		}

		if err := stopSignal(rec.PID); err != nil {
			errs = append(errs, fmt.Errorf("stopping pid %d: %w", rec.PID, err))
			continue
		}
		signaled = append(signaled, rec)
	}
	return signaled, errors.Join(errs...)
}
