package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/upgrade"
	"github.com/seanb4t/codegraph-go/internal/version"
)

// updatePlainGoldens is the ONLY sanctioned way to (re)write a golden file
// under testdata/plain/ — a golden is never hand-edited (D-16, CLI-05).
// Usage: GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestPlainGolden$' -update-plain-goldens -count=1
var updatePlainGoldens = flag.Bool("update-plain-goldens", false, "rewrite internal/cli/testdata/plain/*.golden from the current plain output")

// plainRun is what one plainCase resolves to at run time: the args/env to
// execute through execCmd's non-TTY bytes.Buffer harness (never a real
// TTY), whether to capture stderr instead of stdout, an optional run
// override for cases that don't fit the simple args/env shape (the `ui`
// case, whose lifecycle needs an io.Pipe and a cancellable context), and a
// per-case output normalizer.
type plainRun struct {
	args      []string
	env       []string
	stderr    bool
	run       func(t *testing.T) string
	normalize func(string) string
}

// plainCase is one named table entry. setup defers any fixture/env
// construction until the subtest actually runs, so every case gets its
// own fresh temp dir / fake home / seams, exactly like every other test in
// this package.
type plainCase struct {
	name  string
	setup func(t *testing.T) plainRun
}

// dbSizeRegexp/durationRegexp/uiPortRegexp are the three CLI-04/D-16
// normalizers beyond the literal fixture/home/project path substitutions:
// pebble's on-disk size, indexer wall-clock duration, and the ui command's
// ephemeral loopback port are none of them byte-stable across runs, but
// nothing else on any of these pages is rewritten (D-16's "normalization
// is explicit and minimal").
var (
	dbSizeRegexp   = regexp.MustCompile(`DB Size:(\s+)[0-9.]+ MB`)
	durationRegexp = regexp.MustCompile(`duration=\S+`)
	uiPortRegexp   = regexp.MustCompile(`http://127\.0\.0\.1:\d+`)
)

// normalizeDir returns a normalizer that replaces every literal occurrence
// of dir with placeholder — the fixture/home/project path substitution
// every case that touches a temp directory needs.
func normalizeDir(dir, placeholder string) func(string) string {
	return func(s string) string {
		return strings.ReplaceAll(s, dir, placeholder)
	}
}

// composeNormalize chains normalizers left to right.
func composeNormalize(fns ...func(string) string) func(string) string {
	return func(s string) string {
		for _, fn := range fns {
			s = fn(s)
		}
		return s
	}
}

// normalizeVersionLine replaces the ENTIRE formatted version line (never
// individual tokens — Commit and Date are both "unknown" in a dev build
// and would collide) with the placeholder line.
func normalizeVersionLine(s string) string {
	info := version.Info()
	literal := fmt.Sprintf("codegraph %s (commit %s, built %s) %s %s/%s\n",
		info.Version, info.Commit, info.Date, info.GoVersion, info.OS, info.Arch)
	return strings.Replace(s, literal, "codegraph <VERSION> (commit <COMMIT>, built <DATE>) <GO> <OS>/<ARCH>\n", 1)
}

// normalizePlain applies run.normalize when set, else returns out
// unchanged (the identity default every case without a normalizer gets).
func normalizePlain(run plainRun, out string) string {
	if run.normalize == nil {
		return out
	}
	return run.normalize(out)
}

// runPlainCase executes one plainCase's resolved plainRun and returns its
// captured, normalized output. Default execution: t.Setenv each KEY=VALUE
// in env, then execCmdWithInput("", args...); capture stdout, or stderr
// when stderr is true. When run.run is non-nil, call it instead.
func runPlainCase(t *testing.T, run plainRun) string {
	t.Helper()

	if run.run != nil {
		return normalizePlain(run, run.run(t))
	}

	for _, kv := range run.env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}

	stdout, stderr, err := execCmdWithInput("", run.args...)
	if err != nil {
		t.Fatalf("%v: unexpected error: %v (stdout=%q stderr=%q)", run.args, err, stdout, stderr)
	}

	out := stdout
	if run.stderr {
		out = stderr
	}
	return normalizePlain(run, out)
}

// runUICase reproduces TestUICommandPrintsConnectableURLBeforeServing's
// io.Pipe/cancel shape (ui_test.go) for the `ui` plain-golden case: it
// does not fit execCmd's simple args/env model since the command blocks
// serving until its context is cancelled.
func runUICase(t *testing.T, dir string) string {
	t.Helper()

	pr, pw := io.Pipe()
	cmd := newUiCmd()
	cmd.SetOut(pw)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--path", dir, "--no-open"})

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- cmd.ExecuteContext(ctx) }()

	scanner := bufio.NewScanner(pr)
	if !scanner.Scan() {
		t.Fatalf("ui: no URL line printed on stdout: %v", scanner.Err())
	}
	line := scanner.Text()

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("ui: cmd.ExecuteContext returned %v after cancellation, want nil", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("ui: cmd.ExecuteContext did not return within the shutdown budget after context cancellation")
	}

	return line + "\n"
}

// serveMCPStderrOnce/serveMCPStderrResult back runServeMCPStderrOnce
// below: a real `serve --mcp` invocation must run at most ONCE across this
// entire test binary.
var (
	serveMCPStderrOnce   sync.Once
	serveMCPStderrResult string
)

// runServeMCPStderrOnce executes `serve --mcp` exactly once for the whole
// test binary and caches its (already fixture-normalized) stderr, reusing
// the cached value on every later call. This works around a genuine,
// pre-existing internal/mcp resource-lifecycle property this plan is
// forbidden from touching (D-16's prohibitions): goSDKServer.ServeStdio
// reads from the process-global os.Stdin directly — never cmd.InOrStdin(),
// so execCmd's configured strings.Reader has no effect on it — and its
// stdinLingerReader.Close() closes that SAME *os.File when the session
// ends. A second in-process `serve --mcp` invocation anywhere in this
// binary therefore always fails with "read /dev/stdin: file already
// closed" (confirmed empirically). The disabled-watcher banner this case
// captures is unstyled, NO_COLOR-invariant stderr text — no present
// renderer exists on this path — so caching one real invocation's result
// and reusing it for TestPlainGolden and both TestNoColorNonTTYRegression
// variants still genuinely exercises the real command once, and asserts
// the exact byte-identity property the regression test checks everywhere
// else, without re-triggering the SDK's os.Stdin close.
func runServeMCPStderrOnce(t *testing.T) string {
	t.Helper()
	serveMCPStderrOnce.Do(func() {
		dir := setupIndexedFixture(t)
		fakeHome(t)
		t.Setenv("CODEGRAPH_NO_WATCH", "1")
		_, stderr, err := execCmdWithInput("", "serve", "--mcp", "-p", dir)
		if err != nil {
			t.Fatalf("serve --mcp -p %s: unexpected error: %v (stderr=%q)", dir, err, stderr)
		}
		serveMCPStderrResult = strings.ReplaceAll(stderr, dir, "<FIXTURE>")
	})
	return serveMCPStderrResult
}

// plainCases returns the full plain-golden table: the twelve query-verb
// cases (status, search, search --full, callers, callees, impact,
// affected, files flat/tree, explore, node symbol/file mode) plus the
// lifecycle/agent/one-line verbs D-16 also names (version, telemetry,
// init, index --force, sync, uninit --force, githooks install/status/
// remove, install, uninstall, daemon list/stop/unlock, upgrade, ui,
// serve --mcp) plus 05-01's read-only --print-config-style pair
// (print-config-style, print-config-style-local) — 31 cases in total,
// above the >= 28 floor TestPlainGolden asserts (D-16's own enumeration
// counted to 29; the two added cases are not a discrepancy).
func plainCases(_ *testing.T) []plainCase {
	return []plainCase{
		{name: "status", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{
				args: []string{"status", "-p", dir},
				normalize: composeNormalize(
					normalizeDir(dir, "<FIXTURE>"),
					func(s string) string { return dbSizeRegexp.ReplaceAllString(s, "DB Size:${1}<SIZE> MB") },
				),
			}
		}},
		{name: "search", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"search", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "search-full", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"search", "Alpha", "-p", dir, "--full"}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "callers", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"callers", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "callees", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"callees", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "impact", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"impact", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "affected-empty", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"affected", "pkga/pkga.go", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "files-flat", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"files", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "files-tree", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"files", "-p", dir, "--format", "tree"}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "explore", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"explore", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "node-symbol", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"node", "Alpha", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "node-file", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"node", "-p", dir, "-f", "pkga/pkga.go"}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},

		{name: "version", setup: func(t *testing.T) plainRun {
			return plainRun{args: []string{"version"}, normalize: normalizeVersionLine}
		}},
		{name: "telemetry", setup: func(t *testing.T) plainRun {
			return plainRun{args: []string{"telemetry"}}
		}},
		{name: "init", setup: func(t *testing.T) plainRun {
			// init is the command under test here, so this case cannot go
			// through setupIndexedFixture — it still needs the same
			// platform-portable corpus, or its files= count differs
			// between darwin and linux.
			dir := copyFixture(t)
			pruneGOOSSuffixedFiles(t, dir)
			return plainRun{
				args: []string{"init", dir},
				env:  []string{"CODEGRAPH_NO_WATCH=1"},
				normalize: composeNormalize(
					normalizeDir(dir, "<FIXTURE>"),
					func(s string) string { return durationRegexp.ReplaceAllString(s, "duration=<DURATION>") },
				),
			}
		}},
		{name: "index-force", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{
				args: []string{"index", dir, "--force"},
				normalize: composeNormalize(
					normalizeDir(dir, "<FIXTURE>"),
					func(s string) string { return durationRegexp.ReplaceAllString(s, "duration=<DURATION>") },
				),
			}
		}},
		{name: "sync", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{
				args: []string{"sync", dir},
				normalize: composeNormalize(
					normalizeDir(dir, "<FIXTURE>"),
					func(s string) string { return durationRegexp.ReplaceAllString(s, "duration=<DURATION>") },
				),
			}
		}},
		{name: "uninit-force", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"uninit", dir, "--force"}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "githooks-install", setup: func(t *testing.T) plainRun {
			dir := initGitRepo(t, t.TempDir())
			return plainRun{args: []string{"githooks", "install", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "githooks-status", setup: func(t *testing.T) plainRun {
			dir := initGitRepo(t, t.TempDir())
			if _, _, err := execCmd("githooks", "install", dir); err != nil {
				t.Fatalf("githooks install (setup): %v", err)
			}
			return plainRun{args: []string{"githooks", "status", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "githooks-remove", setup: func(t *testing.T) plainRun {
			dir := initGitRepo(t, t.TempDir())
			if _, _, err := execCmd("githooks", "install", dir); err != nil {
				t.Fatalf("githooks install (setup): %v", err)
			}
			return plainRun{args: []string{"githooks", "remove", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "install-local", setup: func(t *testing.T) plainRun {
			home := fakeHome(t)
			project, err := os.Getwd()
			if err != nil {
				t.Fatalf("Getwd: %v", err)
			}
			return plainRun{
				args: []string{"install", "--target", "claude", "--location", "local"},
				normalize: composeNormalize(
					normalizeDir(home, "<HOME>"),
					normalizeDir(project, "<PROJECT>"),
				),
			}
		}},
		{name: "uninstall-local", setup: func(t *testing.T) plainRun {
			home := fakeHome(t)
			project, err := os.Getwd()
			if err != nil {
				t.Fatalf("Getwd: %v", err)
			}
			if _, _, err := execCmd("install", "--target", "claude", "--location", "local"); err != nil {
				t.Fatalf("install (setup): %v", err)
			}
			return plainRun{
				args: []string{"uninstall", "--target", "claude", "--location", "local"},
				normalize: composeNormalize(
					normalizeDir(home, "<HOME>"),
					normalizeDir(project, "<PROJECT>"),
				),
			}
		}},
		{name: "print-config-style", setup: func(t *testing.T) plainRun {
			home := fakeHome(t)
			return plainRun{
				args:      []string{"install", "--print-config-style"},
				normalize: normalizeDir(home, "<HOME>"),
			}
		}},
		{name: "print-config-style-local", setup: func(t *testing.T) plainRun {
			home := fakeHome(t)
			project, err := os.Getwd()
			if err != nil {
				t.Fatalf("Getwd: %v", err)
			}
			return plainRun{
				args: []string{"install", "--print-config-style", "--location", "local"},
				normalize: composeNormalize(
					normalizeDir(home, "<HOME>"),
					normalizeDir(project, "<PROJECT>"),
				),
			}
		}},
		{name: "daemon-list-empty", setup: func(t *testing.T) plainRun {
			fakeHome(t)
			return plainRun{args: []string{"daemon"}}
		}},
		{name: "daemon-stop-nomatch", setup: func(t *testing.T) plainRun {
			// setupIndexedFixture resolves the shared gofixture via a
			// package-relative path (copyFixture) — it must run BEFORE
			// fakeHome's t.Chdir, or the relative resolution breaks.
			dir := setupIndexedFixture(t)
			fakeHome(t)
			return plainRun{args: []string{"daemon", "stop", "-p", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "daemon-unlock", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{args: []string{"daemon", "unlock", dir}, normalize: normalizeDir(dir, "<FIXTURE>")}
		}},
		{name: "upgrade-refresh-warning", setup: func(t *testing.T) plainRun {
			origRun := upgradeRunFunc
			upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error { return nil }
			t.Cleanup(func() { upgradeRunFunc = origRun })

			origRefresh := refreshInstalledSkillsFunc
			refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
				return errors.New("simulated refresh failure")
			}
			t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

			return plainRun{args: []string{"upgrade"}}
		}},
		{name: "ui-url", setup: func(t *testing.T) plainRun {
			dir := setupIndexedFixture(t)
			return plainRun{
				run:       func(t *testing.T) string { return runUICase(t, dir) },
				normalize: func(s string) string { return uiPortRegexp.ReplaceAllString(s, "http://127.0.0.1:<PORT>") },
			}
		}},
		{name: "serve-mcp-stderr", setup: func(t *testing.T) plainRun {
			return plainRun{run: runServeMCPStderrOnce}
		}},
	}
}

// plainGoldenFloor is the positive floor (rule 84d1gfpywd): a table that
// silently shrinks — or a walk that finds nothing — must never read as
// green.
const plainGoldenFloor = 28

// diffLines reports the first differing line number between want and got,
// for a readable failure message instead of a giant blob diff.
func diffLines(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}
	for i := 0; i < max; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("first differing line %d:\n  want: %q\n  got:  %q", i+1, w, g)
		}
	}
	return "(no line-level difference found; lengths differ)"
}

// TestPlainGolden freezes (or, under -update-plain-goldens, writes) the
// plain, non-TTY, NO_COLOR-unset output of every case in plainCases to
// testdata/plain/<name>.golden, then asserts byte-equality and zero ESC
// (0x1b) bytes — the bar-before-measurement regression D-16 exists for.
func TestPlainGolden(t *testing.T) {
	cases := plainCases(t)
	if len(cases) < plainGoldenFloor {
		t.Fatalf("plainCases returned %d cases, want at least %d", len(cases), plainGoldenFloor)
	}

	caseNames := make(map[string]bool, len(cases))

	for _, c := range cases {
		c := c
		caseNames[c.name] = true
		t.Run(c.name, func(t *testing.T) {
			// Resolve the golden path to an ABSOLUTE path before calling
			// c.setup(t): several cases' setup (e.g. fakeHome) t.Chdir()s
			// into a fresh temp directory, and a relative "testdata/plain/…"
			// path computed after that would resolve against the WRONG
			// directory and silently write/read nowhere near this package.
			goldenPath, err := filepath.Abs(filepath.Join("testdata", "plain", c.name+".golden"))
			if err != nil {
				t.Fatalf("resolve golden path for %s: %v", c.name, err)
			}

			t.Setenv("NO_COLOR", "")
			run := c.setup(t)
			got := runPlainCase(t, run)

			if *updatePlainGoldens {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", filepath.Dir(goldenPath), err)
				}
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				t.Logf("wrote golden %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("fail-closed: %s must exist — run with -update-plain-goldens to create it: %v", goldenPath, err)
			}
			// Both assertions run unconditionally — never short-circuited via
			// Fatalf — so a mutation tripping both (e.g. a planted ESC byte,
			// which is ALSO a byte mismatch against the golden) is reported
			// with both messages in the same failure, not whichever check
			// happened to run first (Family (a), 04-MUTATION-LOG.md).
			if got != string(want) {
				t.Errorf("output does not match golden %s\n%s", goldenPath, diffLines(string(want), got))
			}
			if strings.Contains(got, "\x1b") {
				t.Errorf("output contains an ESC byte (0x1b): %q", got)
			}
		})
	}

	if *updatePlainGoldens {
		return
	}

	goldenFiles, err := filepath.Glob(filepath.Join("testdata", "plain", "*.golden"))
	if err != nil {
		t.Fatalf("glob testdata/plain: %v", err)
	}
	goldenBasenames := make(map[string]bool, len(goldenFiles))
	for _, f := range goldenFiles {
		goldenBasenames[strings.TrimSuffix(filepath.Base(f), ".golden")] = true
	}

	var missingGolden, orphanGolden []string
	for name := range caseNames {
		if !goldenBasenames[name] {
			missingGolden = append(missingGolden, name)
		}
	}
	for base := range goldenBasenames {
		if !caseNames[base] {
			orphanGolden = append(orphanGolden, base)
		}
	}
	sort.Strings(missingGolden)
	sort.Strings(orphanGolden)
	if len(missingGolden) > 0 {
		t.Fatalf("case(s) with no golden file: %v", missingGolden)
	}
	if len(orphanGolden) > 0 {
		t.Fatalf("orphan golden file(s) with no matching case: %v", orphanGolden)
	}
}

// TestNoColorNonTTYRegression re-runs the same table under NO_COLOR=1 and
// NO_COLOR=banana and asserts byte-equality to the SAME goldens plus zero
// ESC bytes — the gh #13335 lesson pinned end-to-end, and (for banana) the
// D-09 research correction pinned before the resolver exists: today
// ChoosePresentation treats any non-empty NO_COLOR as off, and this test
// pins that this stays true after the glow-up too.
func TestNoColorNonTTYRegression(t *testing.T) {
	for _, envVal := range []string{"1", "banana"} {
		envVal := envVal
		t.Run("NO_COLOR="+envVal, func(t *testing.T) {
			for _, c := range plainCases(t) {
				c := c
				t.Run(c.name, func(t *testing.T) {
					// Same absolute-path requirement as TestPlainGolden —
					// resolve BEFORE c.setup(t) may t.Chdir() away.
					goldenPath, err := filepath.Abs(filepath.Join("testdata", "plain", c.name+".golden"))
					if err != nil {
						t.Fatalf("resolve golden path for %s: %v", c.name, err)
					}

					t.Setenv("NO_COLOR", envVal)
					run := c.setup(t)
					got := runPlainCase(t, run)

					want, err := os.ReadFile(goldenPath)
					if err != nil {
						t.Fatalf("fail-closed: %s must exist: %v", goldenPath, err)
					}
					// Both assertions run unconditionally, same rationale as
					// TestPlainGolden above.
					if got != string(want) {
						t.Errorf("NO_COLOR=%s: output does not match golden %s\n%s", envVal, goldenPath, diffLines(string(want), got))
					}
					if strings.Contains(got, "\x1b") {
						t.Errorf("NO_COLOR=%s: output contains an ESC byte (0x1b): %q", envVal, got)
					}
				})
			}
		})
	}
}
