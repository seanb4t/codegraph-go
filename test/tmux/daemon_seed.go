//go:build tmux

package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// seedRunningDaemon starts a REAL codegraph daemon (`daemon start`) as a
// background subprocess against home+repo, so TTY-04's picker has a
// genuinely registered record to show. `daemon start` refuses an
// uninitialized repo, so this first runs `codegraph init repo` synchronously
// (08-RESEARCH.md Pattern 2, verified end to end). The registry itself is
// never hand-written: internal/daemon/lock.go's isStale liveness+clock
// corroboration is unexported and platform-gated, and reimplementing it in
// this black-box package would assert against a fixture rather than the
// real registration path (08-CONTEXT.md's settled note).
//
// A t.Cleanup that gracefully SIGTERMs the daemon and Wait()s for it is
// registered immediately after Start() succeeds and BEFORE anything that
// could t.Fatal, so a panicking or failing test can never leave a live
// daemon holding a registry record (T-08-03). SIGTERM triggers the
// daemon's own signal handler, which deregisters via a deferred call before
// exiting; a bare Kill would leave the record behind. The launched
// process's own pid IS the registered pid — there is no fork or
// double-fork — so no pid discovery is needed.
func seedRunningDaemon(t *testing.T, home, repo string) *exec.Cmd {
	t.Helper()

	env := append(os.Environ(), "HOME="+home, "USERPROFILE="+home)

	initCmd := exec.Command(binPath, "init", repo)
	initCmd.Env = env
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("seedRunningDaemon: codegraph init %s failed: %v: %s", repo, err, out)
	}

	daemonCmd := exec.Command(binPath, "daemon", "start", "--path", repo, "--quiet")
	daemonCmd.Env = env
	if err := daemonCmd.Start(); err != nil {
		t.Fatalf("seedRunningDaemon: codegraph daemon start --path %s failed to start: %v", repo, err)
	}

	t.Cleanup(func() {
		_ = daemonCmd.Process.Signal(syscall.SIGTERM)
		_ = daemonCmd.Wait()
	})

	return daemonCmd
}

// waitForRegistryRecord polls home's daemon registry directory
// (~/.codegraph/daemons, internal/daemon/registry.go's registryDir) on
// stabilityPollInterval until at least one record file is present. This is
// a bounded poll with a convergence condition, not a fixed pause — the same
// discipline D-14 imposes on frame capture applies equally to process
// readiness. On deadline it calls t.Fatalf naming the directory it watched
// and what it found there; there is no code path where it returns after a
// timeout.
func waitForRegistryRecord(t *testing.T, home string) {
	t.Helper()

	dir := filepath.Join(home, ".codegraph", "daemons")
	deadline := time.Now().Add(stabilityPollDeadline)
	for {
		entries, err := os.ReadDir(dir)
		if err == nil && len(entries) > 0 {
			return
		}
		if time.Now().After(deadline) {
			var found []string
			for _, e := range entries {
				found = append(found, e.Name())
			}
			t.Fatalf("waitForRegistryRecord: no registry record appeared under %s within %s (readdir err: %v, found: %v)",
				dir, stabilityPollDeadline, err, found)
		}
		<-time.After(stabilityPollInterval)
	}
}
