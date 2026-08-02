// Command runner is the codegraph benchmark harness: it drives the
// freshly-built Go `codegraph` binary — and, in headtohead mode, the
// installed TS `codegraph@1.3.1` binary too — through a shared, external
// measurement path and reports internal/bench.Metrics.
//
// Two modes (-mode):
//
//   - headtohead: for each tools/bench/realcorpus entry, runs BOTH the
//     Go and TS binaries over the real, pinned repo and prints raw
//     per-repo/per-subject Metrics as JSON to stdout (PERF-01). This is
//     a published comparison, not a gate — it never fails the process
//     on its own.
//
//   - regression: materializes the deterministic, network-free
//     synthetic corpus (tools/bench/gencorpus) and runs ONLY the Go
//     binary over it, then checks the result against the committed
//     baseline (-baseline) via internal/bench.CheckRegression
//     (PERF-02 + the INDX-06 absolute peak-RSS ceiling, -ceiling-bytes).
//     Exits non-zero when CheckRegression returns an error — this is
//     the blocking CI gate. -rebless is the ONLY flag that causes this
//     mode to overwrite -baseline instead of gating against it.
//
// Every measured command is run 5 times (medianRuns) and each metric's
// own median is reported, per CONTEXT.md D-05. Regression mode adds a
// second, outer level of repetition on top of that: -trials N (default
// defaultTrials) repeats the whole materialize+init+measure session N
// times and medians each metric across sessions, because median-of-5
// inside one session does not touch session-to-session variance — see
// defaultTrials. Peak RSS is ALWAYS read
// from the completed child process via bench.PeakRSSBytes — never via
// this process's own in-process memory statistics, which cannot be
// compared fairly against the TS Node process. Only fixed, pinned repo
// paths or the generated corpus's own scratch directory are ever passed
// as subprocess arguments (V5) — this tool never forwards
// attacker-controlled input to either shelled-out binary.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/bench"
	"github.com/seanb4t/codegraph-go/tools/bench/realcorpus"
)

// medianRuns is the fixed repetition count per measured command (D-05's
// "median-of-5"). Not exposed as a flag: CONTEXT.md fixed this as a
// starting point, not an operator knob.
const medianRuns = 5

// defaultTrials is how many INDEPENDENT measurement sessions regression
// mode runs before taking each metric's median (-trials). A session is a
// full corpus-materialize + init + measure cycle; medianRuns above is the
// separate, inner repetition of each measured command WITHIN one session.
//
// Why this exists at all: medianRuns only collapses jitter inside a
// single session. It does not touch session-to-session variance, and the
// perf-gate-throughput-regress debug session measured that variance
// directly — three back-to-back sessions in one fixed container, after
// median-of-5, still spread 1.2%-3.0% (38504/38558/38970 and
// 37738/38837/38894 files/s). CI's own three observations spread 2.4%
// (11374.57/11642.18/11361.67). So a one-session number, however
// internally medianed, is a single draw from a distribution that is
// wide enough to flip a verdict near the threshold — and, on the
// -rebless path, wide enough to bake a tail value in as the new normal.
//
// Why 3 specifically:
//   - It is the repo's already-ratified convention for its other perf
//     mechanism (PERF-01 head-to-head is median-of-3); one number for
//     both, rather than inventing a second.
//   - It is the smallest N with a true median, so a single outlier
//     session in either direction is rejected rather than averaged in.
//   - Against a 10% budget and a measured ~1-3% session spread, rejecting
//     one outlier is sufficient; N=5 would cost ~1.7x the CI time for a
//     marginal precision gain at this variance level.
const defaultTrials = 3

// defaultCeilingBytes is the starting INDX-06 absolute peak-RSS budget
// for the synthetic regression corpus. It is a documented starting
// point (Task 3 records the actual measured baseline peak RSS
// alongside it) — tune via -ceiling-bytes once real CI hardware
// numbers are known.
const defaultCeilingBytes int64 = 4 * 1024 * 1024 * 1024 // 4 GiB

// regressionFileCount is the default synthetic-corpus size regression
// mode materializes when -count isn't overridden. This mirrors
// tools/bench/gencorpus's own ProductionFileCount literal — duplicated
// here (rather than imported) because gencorpus is a standalone
// `package main` and cannot be imported by another Go package; see
// buildGencorpus below, which shells out to it instead.
const regressionFileCount = 120000

// regressionQueryTerm is the fixed query-latency probe for regression
// mode: gencorpus's Go population always writes this exact symbol as
// its very first generated file (pkg0000/file0000.go), for any
// FileCount >= 1 — see tools/bench/gencorpus/gen.go's generateGo.
const regressionQueryTerm = "Fn0000_0000"

// macOSHomebrewTSBinary is the conventional Apple-Silicon Homebrew
// install path for the TS codegraph@1.3.1 CLI. It is used only as a
// last-resort fallback when -ts-binary isn't set and "codegraph" isn't
// on PATH (IN-02, Phase 8 re-review) — never as the flag's own default,
// since that default previously broke silently on Linux and Intel
// macOS. bench.yml always passes -ts-binary explicitly and is
// unaffected either way.
const macOSHomebrewTSBinary = "/opt/homebrew/bin/codegraph"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "runner: %v\n", err)
		os.Exit(1)
	}
}

// config holds every flag this tool accepts.
type config struct {
	mode         string
	goBinary     string
	tsBinary     string
	baselinePath string
	ceilingBytes int64
	rebless      bool
	scratchDir   string
	seed         int64
	count        int
	trials       int
	runner       string
}

func run(args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	if cfg.goBinary == "" {
		built, cleanup, err := buildGoBinary()
		if err != nil {
			return fmt.Errorf("build Go codegraph binary: %w", err)
		}
		defer cleanup()
		cfg.goBinary = built
	}

	if cfg.mode == "headtohead" && cfg.tsBinary == "" {
		cfg.tsBinary = resolveTSBinary()
	}

	switch cfg.mode {
	case "headtohead":
		return runHeadToHead(cfg)
	case "regression":
		return runRegression(cfg)
	default:
		return fmt.Errorf("unknown -mode %q (want headtohead or regression)", cfg.mode)
	}
}

// parseFlags defines and parses every flag the runner accepts. -mode,
// -rebless, and -ceiling-bytes are always listed in usage output (Task
// 2 acceptance criterion) regardless of which mode is ultimately used.
func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("runner", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.mode, "mode", "", "benchmark mode: headtohead or regression (required)")
	fs.StringVar(&cfg.goBinary, "go-binary", "", "path to the Go codegraph binary; if empty, the runner builds one fresh from ./cmd/codegraph")
	fs.StringVar(&cfg.tsBinary, "ts-binary", "", "path to the installed TS codegraph@1.3.1 binary (headtohead mode only); if empty, resolved via PATH lookup, falling back to the macOS Homebrew default")
	fs.StringVar(&cfg.baselinePath, "baseline", filepath.Join("tools", "bench", "baseline.json"), "path to the committed regression baseline JSON (regression mode only)")
	fs.Int64Var(&cfg.ceilingBytes, "ceiling-bytes", defaultCeilingBytes, "absolute peak-RSS ceiling in bytes (INDX-06, regression mode only); 0 disables the ceiling check")
	fs.BoolVar(&cfg.rebless, "rebless", false, "regression mode only: overwrite -baseline with the freshly-measured metrics instead of gating against it — the ONLY path that writes baseline.json")
	fs.StringVar(&cfg.scratchDir, "scratch-dir", "", "regression mode only: directory to materialize the synthetic corpus into (default: a fresh temp dir, cleaned up on exit)")
	fs.Int64Var(&cfg.seed, "seed", 42, "regression mode only: gencorpus RNG seed (same seed -> byte-identical corpus)")
	fs.IntVar(&cfg.count, "count", regressionFileCount, "regression mode only: number of synthetic files to generate")
	fs.IntVar(&cfg.trials, "trials", defaultTrials, "regression mode only: number of independent measurement sessions to take each metric's median over; 1 means single-sample (not recommended for gating or -rebless)")
	fs.StringVar(&cfg.runner, "runner", os.Getenv("CODEGRAPH_BENCH_RUNNER"), "runner identity label recorded alongside goos/goarch (e.g. a Namespace runs-on profile such as namespace-profile-linux-amd64-4x8); defaults to $CODEGRAPH_BENCH_RUNNER, and an explicit flag wins over the env var. Empty is fine outside CI — measurement never requires it")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if cfg.mode == "" {
		fs.Usage()
		return config{}, fmt.Errorf("-mode is required")
	}
	if cfg.trials < 1 {
		return config{}, fmt.Errorf("-trials must be >= 1, got %d", cfg.trials)
	}
	return cfg, nil
}

// resolveTSBinary finds the installed TS codegraph@1.3.1 binary when
// -ts-binary wasn't set explicitly (IN-02, Phase 8 re-review): first via
// a PATH lookup (portable across Linux/macOS/Windows and any install
// method), falling back to the conventional Apple-Silicon Homebrew path
// if that exists. If neither resolves, it returns "" and
// measureSubject's existing "no binary configured for subject %q" error
// path reports the failure clearly rather than silently trying an
// unrunnable macOS-only default.
func resolveTSBinary() string {
	if p, err := exec.LookPath("codegraph"); err == nil {
		return p
	}
	if info, err := os.Stat(macOSHomebrewTSBinary); err == nil && !info.IsDir() {
		return macOSHomebrewTSBinary
	}
	return ""
}

// ---------------------------------------------------------------------
// Shared measurement primitive
// ---------------------------------------------------------------------

// measuredRun is one external, OS-level measurement of a single
// subprocess invocation.
type measuredRun struct {
	durationMS float64
	peakRSS    int64
}

// runOnce executes name with args in directory dir, waits for it to
// exit, and returns its wall-clock duration and OS-level peak RSS via
// bench.PeakRSSBytes. It never inspects this (the runner's own) process
// memory — only the completed child's ProcessState.
func runOnce(name string, args []string, dir string) (measuredRun, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	if err != nil {
		return measuredRun{}, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}

	rss, err := bench.PeakRSSBytes(cmd.ProcessState)
	if err != nil {
		return measuredRun{}, fmt.Errorf("peak RSS for %s %s: %w", name, strings.Join(args, " "), err)
	}

	return measuredRun{
		durationMS: float64(elapsed.Microseconds()) / 1000.0,
		peakRSS:    rss,
	}, nil
}

// medianOfN runs fn n times and returns the median duration and the
// median peak RSS, each computed independently over its own sorted
// sample (D-05: "median-of-5" per metric, not "the run whose duration
// happened to be the median").
func medianOfN(n int, fn func() (measuredRun, error)) (measuredRun, error) {
	if n <= 0 {
		return measuredRun{}, fmt.Errorf("medianOfN: n must be > 0, got %d", n)
	}
	durations := make([]float64, 0, n)
	rsses := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		m, err := fn()
		if err != nil {
			return measuredRun{}, err
		}
		durations = append(durations, m.durationMS)
		rsses = append(rsses, m.peakRSS)
	}
	sort.Float64s(durations)
	sort.Slice(rsses, func(i, j int) bool { return rsses[i] < rsses[j] })
	return measuredRun{durationMS: medianFloat64(durations), peakRSS: medianInt64(rsses)}, nil
}

func medianFloat64(v []float64) float64 {
	n := len(v)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

func medianInt64(v []int64) int64 {
	n := len(v)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

// ---------------------------------------------------------------------
// headtohead mode (PERF-01)
// ---------------------------------------------------------------------

func runHeadToHead(cfg config) error {
	scratchRoot, err := os.MkdirTemp("", "codegraph-bench-h2h-")
	if err != nil {
		return fmt.Errorf("create scratch root: %w", err)
	}
	defer os.RemoveAll(scratchRoot)

	var results []bench.Metrics
	for _, entry := range realcorpus.Corpora() {
		srcDir, err := resolveOrClone(entry, scratchRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "runner: skipping %s: %v\n", entry.Name, err)
			continue
		}

		for _, subject := range []struct {
			name   string
			binary string
		}{
			{"go", cfg.goBinary},
			{"ts", cfg.tsBinary},
		} {
			m, err := measureSubject(subject.name, subject.binary, entry, srcDir, scratchRoot, cfg.runner)
			if err != nil {
				fmt.Fprintf(os.Stderr, "runner: %s/%s: %v\n", entry.Name, subject.name, err)
				continue
			}
			results = append(results, m)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// measureSubject runs one binary (identified by subjectName, "go" or
// "ts") over one realcorpus entry's source tree and returns its
// bench.Metrics. It copies srcDir into an isolated per-subject work
// directory first so the Go binary's Pebble store and the TS binary's
// SQLite store never collide, and so neither binary's .codegraph/ is
// ever written into the resolved source checkout itself (which, for a
// sibling-checkout entry, is a real directory the operator owns).
func measureSubject(subjectName, binary string, entry realcorpus.Entry, srcDir, scratchRoot, runner string) (bench.Metrics, error) {
	if binary == "" {
		return bench.Metrics{}, fmt.Errorf("no binary configured for subject %q", subjectName)
	}
	if _, err := exec.LookPath(binary); err != nil {
		if _, statErr := os.Stat(binary); statErr != nil {
			return bench.Metrics{}, fmt.Errorf("subject %q binary %q not runnable: %w", subjectName, binary, err)
		}
	}

	workDir := filepath.Join(scratchRoot, entry.Name, subjectName)
	if err := copyTree(srcDir, workDir); err != nil {
		return bench.Metrics{}, fmt.Errorf("copy corpus into work dir: %w", err)
	}

	fileCount, byteCount, err := countTree(workDir)
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("count corpus size: %w", err)
	}

	// Unmeasured warmup: both CLIs' `init` creates .codegraph/ and
	// does an initial build. The Go CLI's `index --force` refuses to
	// run without .codegraph/ already existing (ErrNotInitialized);
	// running init first keeps both subjects on the same setup step
	// before the measured, genuinely-from-scratch `index --force`
	// rebuild below.
	// Note: --quiet is intentionally NOT passed here — the TS CLI's
	// `init` (unlike its `index`) has no --quiet flag, and runOnce
	// already discards the child's stdout/stderr (nil Cmd.Stdout/
	// Stderr -> /dev/null), so suppressing output is unnecessary here.
	if _, err := runOnce(binary, []string{"init", workDir}, workDir); err != nil {
		return bench.Metrics{}, fmt.Errorf("warmup init: %w", err)
	}

	indexResult, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(binary, []string{"index", workDir, "--force", "--quiet"}, workDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure index --force: %w", err)
	}

	queryTerm := regressionQueryTerm
	if len(entry.QueryTerms) > 0 {
		queryTerm = entry.QueryTerms[0]
	}
	queryResult, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(binary, []string{"query", queryTerm, "-p", workDir}, workDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure query: %w", err)
	}

	coldStart, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(binary, []string{"--version"}, workDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure cold start: %w", err)
	}

	seconds := indexResult.durationMS / 1000.0
	var filesPerSec, bytesPerSec float64
	if seconds > 0 {
		filesPerSec = float64(fileCount) / seconds
		bytesPerSec = float64(byteCount) / seconds
	}

	return bench.Metrics{
		Subject: subjectName,
		Repo:    entry.Name,
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
		Runner:  runner,
		// headtohead measures each (repo, subject) pair in exactly one
		// session — it publishes raw numbers and never gates, so the
		// multi-session median regression mode needs is not applied here.
		// Recorded as 1 rather than left at 0 so published JSON states
		// its own provenance instead of implying "unknown".
		MedianOfTrials:       1,
		FilesPerSec:          filesPerSec,
		BytesPerSec:          bytesPerSec,
		QueryLatencyMedianMS: queryResult.durationMS,
		PeakRSSBytes:         indexResult.peakRSS,
		ColdStartMS:          coldStart.durationMS,
	}, nil
}

// resolveOrClone resolves entry to a local source path, shallow-cloning
// it at exactly entry.CommitSHA into a cache directory under
// scratchRoot when no local checkout is already available. This is the
// one place in the runner that touches the network — headtohead mode
// only, never regression mode (D-04) — and it always pins to the exact
// commit, never HEAD (V5).
func resolveOrClone(entry realcorpus.Entry, scratchRoot string) (string, error) {
	if p, err := entry.Resolve(); err == nil {
		head := pinnedAt(p)
		if head == entry.CommitSHA {
			return p, nil
		}
		fmt.Fprintf(os.Stderr, "runner: %s checkout at %s is not pinned at %s (HEAD=%s); cloning fresh\n", entry.Name, p, entry.CommitSHA, head)
	}

	dest := filepath.Join(scratchRoot, "clone-"+entry.Name)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", fmt.Errorf("create clone dir: %w", err)
	}
	if err := runGit(dest, "clone", "--quiet", "--depth", "1", entry.SourceURL, "."); err != nil {
		return "", fmt.Errorf("clone %s: %w", entry.SourceURL, err)
	}
	if err := runGit(dest, "fetch", "--quiet", "--depth", "1", "origin", entry.CommitSHA); err != nil {
		return "", fmt.Errorf("fetch pinned commit %s: %w", entry.CommitSHA, err)
	}
	if err := runGit(dest, "checkout", "--quiet", entry.CommitSHA); err != nil {
		return "", fmt.Errorf("checkout pinned commit %s: %w", entry.CommitSHA, err)
	}
	return dest, nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(out), err)
	}
	return nil
}

// pinnedAt returns dir's current HEAD commit SHA, or "" if dir is not a
// git checkout or the rev-parse fails. Used by resolveOrClone (WR-02,
// Phase 8 re-review) to verify an existing local checkout found via
// realcorpus.Entry.Resolve is actually pinned at the expected commit
// before trusting it, rather than silently benchmarking a stale sibling
// checkout that has since moved.
func pinnedAt(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ---------------------------------------------------------------------
// regression mode (PERF-02 + INDX-06)
// ---------------------------------------------------------------------

// scratchFSTmpfs and scratchFSDisk are internal/bench.Metrics.ScratchFS's
// two possible values (10-04-PLAN). Not an enum type: Metrics is a plain
// json-tagged struct with no methods (mirrors GOOS/GOARCH/Runner's own
// plain-string discipline).
const (
	scratchFSTmpfs = "tmpfs"
	scratchFSDisk  = "disk"
)

// tmpfsScratchRoot is the tmpfs mount this repo's Linux CI runners expose
// (measured: 4.0GiB free on namespace-profile-linux-amd64-4x8, 7.9GiB
// free on ubuntu-latest — both comfortably hold the ~773MB the regression
// corpus + Pebble store actually occupy on disk once small-file block
// overhead is accounted for). Not checked on non-Linux hosts (macOS has
// no /dev/shm) — resolveRegressionScratchDir degrades to disk there,
// which is unaffected by this change (D-04: never gated on a specific
// host).
const tmpfsScratchRoot = "/dev/shm"

// resolveRegressionScratchDir resolves regression mode's scratch
// directory when -scratch-dir isn't set, preferring tmpfsRoot (RAM-backed)
// over disk. tmpfsRoot is a parameter — rather than resolveRegressionScratchDir
// hardcoding tmpfsScratchRoot itself — specifically so tests can substitute
// a t.TempDir() standing in for "tmpfs" and exercise the preference logic
// without depending on a real tmpfs mount existing in the test
// environment; the production call site (defaultRegressionScratchDir)
// always passes tmpfsScratchRoot.
//
// Why prefer tmpfs at all: PERF-02's synthetic corpus is ~18MB of raw
// content but ~773MB of actual on-disk footprint (120k files at ~4KB/
// block minimum each dominates), making this workload IOPS/metadata-bound
// rather than bandwidth-bound — exactly where overlayfs's layer-stack
// lookups and copy-up-on-write cost more AND vary more under co-tenancy
// than a real block device or tmpfs does (measured: 4.36% Namespace
// session-to-session throughput variance against a rock-stable 0.56%
// peak-RSS variance — same work, same memory, different wall-clock is the
// textbook signature of storage-latency jitter, not CPU or memory
// pressure). Returns the scratch dir path and which filesystem class it
// landed on, so the caller can record that frame descriptor in the
// measured Metrics (see internal/bench.Metrics.ScratchFS) — the storage
// frame isn't the only thing that can silently change what "no
// regression" means; the runner class already gets this same treatment
// via Metrics.Runner.
func resolveRegressionScratchDir(tmpfsRoot string) (dir string, fsClass string, err error) {
	if info, statErr := os.Stat(tmpfsRoot); statErr == nil && info.IsDir() {
		if d, mkErr := os.MkdirTemp(tmpfsRoot, "codegraph-bench-regression-"); mkErr == nil {
			return d, scratchFSTmpfs, nil
		}
		// Fall through to disk on any tmpfs MkdirTemp failure (e.g.
		// permissions, tmpfs full) rather than hard-failing the whole
		// run over a storage-frame optimization.
	}
	d, mkErr := os.MkdirTemp("", "codegraph-bench-regression-")
	if mkErr != nil {
		return "", "", mkErr
	}
	return d, scratchFSDisk, nil
}

// defaultRegressionScratchDir is resolveRegressionScratchDir bound to the
// production tmpfs root. The only production call site.
func defaultRegressionScratchDir() (dir string, fsClass string, err error) {
	return resolveRegressionScratchDir(tmpfsScratchRoot)
}

func runRegression(cfg config) error {
	scratchDir := cfg.scratchDir
	// scratchFS records which filesystem class trials in THIS run were
	// measured on. Left "" when the caller supplied an explicit
	// -scratch-dir: they made the choice, and this runner doesn't probe
	// an arbitrary caller-supplied path's filesystem type — only the
	// path it resolves itself is characterized.
	var scratchFS string
	if scratchDir == "" {
		dir, fsClass, err := defaultRegressionScratchDir()
		if err != nil {
			return fmt.Errorf("create scratch dir: %w", err)
		}
		defer os.RemoveAll(dir)
		scratchDir = dir
		scratchFS = fsClass
	}

	// Each trial is a fully independent measurement session: its own
	// freshly-materialized corpus, its own init, its own median-of-5
	// command runs. Sharing a corpus across trials would leave trials
	// 2..N running against a warm page cache and a store directory the
	// previous trial already touched, which biases the median toward the
	// warm case rather than sampling the distribution the gate actually
	// lives in. See defaultTrials for why this level of independence is
	// the one that matters.
	trials := make([]bench.Metrics, 0, cfg.trials)
	for i := 0; i < cfg.trials; i++ {
		m, err := measureRegressionTrial(cfg, scratchDir, i)
		if err != nil {
			return fmt.Errorf("trial %d/%d: %w", i+1, cfg.trials, err)
		}
		m.ScratchFS = scratchFS
		fmt.Fprintf(os.Stderr, "runner: trial %d/%d: %.2f files/s, peak RSS %d bytes\n",
			i+1, cfg.trials, m.FilesPerSec, m.PeakRSSBytes)
		trials = append(trials, m)
	}

	current, err := medianMetrics(trials)
	if err != nil {
		return fmt.Errorf("aggregate trials: %w", err)
	}

	if cfg.rebless {
		return writeBaseline(cfg.baselinePath, current)
	}

	baseline, err := readBaseline(cfg.baselinePath)
	if err != nil {
		return fmt.Errorf("load baseline %s: %w", cfg.baselinePath, err)
	}

	if err := bench.CheckRegression(baseline, current, cfg.ceilingBytes); err != nil {
		printMetrics(os.Stderr, current)
		return fmt.Errorf("regression gate failed: %w", err)
	}

	fmt.Fprintln(os.Stdout, "regression gate passed")
	printMetrics(os.Stdout, current)
	return nil
}

// measureRegressionTrial runs ONE independent measurement session and
// returns its Metrics (MedianOfTrials left unset — that is the aggregate's
// property, not a single trial's). trialIdx only names the scratch
// subdirectory so concurrent trials never share corpus state.
//
// The trial's corpus is removed on the way out: a 120k-file corpus is
// ~18MB but ~120k inodes, and holding cfg.trials of them simultaneously
// buys nothing once the trial's numbers are recorded.
func measureRegressionTrial(cfg config, scratchDir string, trialIdx int) (bench.Metrics, error) {
	corpusDir := filepath.Join(scratchDir, fmt.Sprintf("corpus-trial%d", trialIdx))
	if err := generateSyntheticCorpus(cfg.seed, cfg.count, corpusDir); err != nil {
		return bench.Metrics{}, fmt.Errorf("generate synthetic corpus: %w", err)
	}
	defer os.RemoveAll(corpusDir)

	fileCount, byteCount, err := countTree(corpusDir)
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("count corpus size: %w", err)
	}

	if _, err := runOnce(cfg.goBinary, []string{"init", corpusDir}, corpusDir); err != nil {
		return bench.Metrics{}, fmt.Errorf("warmup init: %w", err)
	}

	indexResult, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(cfg.goBinary, []string{"index", corpusDir, "--force", "--quiet"}, corpusDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure index --force: %w", err)
	}

	queryResult, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(cfg.goBinary, []string{"query", regressionQueryTerm, "-p", corpusDir}, corpusDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure query: %w", err)
	}

	coldStart, err := medianOfN(medianRuns, func() (measuredRun, error) {
		return runOnce(cfg.goBinary, []string{"--version"}, corpusDir)
	})
	if err != nil {
		return bench.Metrics{}, fmt.Errorf("measure cold start: %w", err)
	}

	seconds := indexResult.durationMS / 1000.0
	var filesPerSec, bytesPerSec float64
	if seconds > 0 {
		filesPerSec = float64(fileCount) / seconds
		bytesPerSec = float64(byteCount) / seconds
	}

	return bench.Metrics{
		Subject:              "go",
		Repo:                 fmt.Sprintf("synthetic-seed%d-count%d", cfg.seed, cfg.count),
		GOOS:                 runtime.GOOS,
		GOARCH:               runtime.GOARCH,
		Runner:               cfg.runner,
		FilesPerSec:          filesPerSec,
		BytesPerSec:          bytesPerSec,
		QueryLatencyMedianMS: queryResult.durationMS,
		PeakRSSBytes:         indexResult.peakRSS,
		ColdStartMS:          coldStart.durationMS,
	}, nil
}

// medianMetrics reduces per-trial Metrics to one aggregate, taking EACH
// numeric metric's own median over its own sorted sample — the same
// discipline medianOfN applies within a trial (D-05: not "the trial whose
// throughput happened to be the median"). Identity fields (Subject, Repo,
// GOOS, GOARCH, Runner) are identical across trials by construction and
// are carried from the first.
//
// Runner is checked for consistency across trials before being carried:
// a mixed-runner trial set (e.g. some trials measured on ubuntu-latest,
// others on a Namespace profile mid-migration) is a category error of
// exactly the kind internal/bench.CheckRegression already refuses for a
// GOOS/GOARCH mismatch — silently picking the first value would let that
// masquerade as one coherent measurement, so it is refused with both
// values named instead.
//
// MedianOfTrials is set to the sample size so the number's provenance
// travels with it into baseline.json and into the gate's own output.
func medianMetrics(trials []bench.Metrics) (bench.Metrics, error) {
	if len(trials) == 0 {
		return bench.Metrics{}, nil
	}

	for _, t := range trials[1:] {
		if t.Runner != trials[0].Runner {
			return bench.Metrics{}, fmt.Errorf(
				"medianMetrics: mixed runner identities across trials: %q vs %q — a mixed-runner trial set cannot be aggregated into one measurement",
				trials[0].Runner, t.Runner,
			)
		}
	}

	filesPerSec := make([]float64, 0, len(trials))
	bytesPerSec := make([]float64, 0, len(trials))
	queryMS := make([]float64, 0, len(trials))
	coldStartMS := make([]float64, 0, len(trials))
	peakRSS := make([]int64, 0, len(trials))
	for _, t := range trials {
		filesPerSec = append(filesPerSec, t.FilesPerSec)
		bytesPerSec = append(bytesPerSec, t.BytesPerSec)
		queryMS = append(queryMS, t.QueryLatencyMedianMS)
		coldStartMS = append(coldStartMS, t.ColdStartMS)
		peakRSS = append(peakRSS, t.PeakRSSBytes)
	}
	sort.Float64s(filesPerSec)
	sort.Float64s(bytesPerSec)
	sort.Float64s(queryMS)
	sort.Float64s(coldStartMS)
	sort.Slice(peakRSS, func(i, j int) bool { return peakRSS[i] < peakRSS[j] })

	// ScratchFS gets no separate mixed-trial guard (unlike Runner):
	// runRegression resolves the scratch filesystem class ONCE per run
	// and assigns that same value to every trial, so it cannot diverge
	// across trials[i] by construction — a mismatch guard here would be
	// dead code, not a real category-error check.
	return bench.Metrics{
		Subject:              trials[0].Subject,
		Repo:                 trials[0].Repo,
		GOOS:                 trials[0].GOOS,
		GOARCH:               trials[0].GOARCH,
		Runner:               trials[0].Runner,
		ScratchFS:            trials[0].ScratchFS,
		MedianOfTrials:       len(trials),
		FilesPerSec:          medianFloat64(filesPerSec),
		BytesPerSec:          medianFloat64(bytesPerSec),
		QueryLatencyMedianMS: medianFloat64(queryMS),
		PeakRSSBytes:         medianInt64(peakRSS),
		ColdStartMS:          medianFloat64(coldStartMS),
	}, nil
}

func printMetrics(w io.Writer, m bench.Metrics) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(m)
}

// readBaseline loads the committed baseline metrics from path.
func readBaseline(path string) (bench.Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return bench.Metrics{}, err
	}
	var m bench.Metrics
	if err := json.Unmarshal(data, &m); err != nil {
		return bench.Metrics{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// writeBaseline overwrites path with m. This is the ONLY function in
// the runner that writes baseline.json, and it is only ever called
// when -rebless is explicitly set (see runRegression) — never as a
// side effect of a normal gating run.
func writeBaseline(path string, m bench.Metrics) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "reblessed %s\n", path)
	printMetrics(os.Stdout, m)
	return nil
}

// generateSyntheticCorpus materializes the deterministic, network-free
// PERF-02/INDX-06 corpus by building and running tools/bench/gencorpus
// as a subprocess (gencorpus is package main and cannot be imported
// directly). No repo is cloned and no network is touched by this
// function or by gencorpus itself (D-04).
func generateSyntheticCorpus(seed int64, count int, outDir string) error {
	root, err := repoRootDir()
	if err != nil {
		return fmt.Errorf("locate repo root: %w", err)
	}

	genBinDir, err := os.MkdirTemp("", "codegraph-gencorpus-bin-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(genBinDir)

	genBin := filepath.Join(genBinDir, "gencorpus")
	buildCmd := exec.Command("go", "build", "-o", genBin, "./tools/bench/gencorpus")
	buildCmd.Dir = root
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build gencorpus: %s: %w", string(out), err)
	}

	genCmd := exec.Command(genBin,
		"-seed", fmt.Sprintf("%d", seed),
		"-out", outDir,
		"-count", fmt.Sprintf("%d", count),
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run gencorpus: %s: %w", string(out), err)
	}
	return nil
}

// ---------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------

// buildGoBinary compiles ./cmd/codegraph into a fresh temp file and
// returns its path plus a cleanup func removing the temp directory.
func buildGoBinary() (string, func(), error) {
	root, err := repoRootDir()
	if err != nil {
		return "", func() {}, fmt.Errorf("locate repo root: %w", err)
	}

	dir, err := os.MkdirTemp("", "codegraph-bench-gobin-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	binPath := filepath.Join(dir, "codegraph")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/codegraph")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("go build ./cmd/codegraph: %s: %w", string(out), err)
	}
	return binPath, cleanup, nil
}

// repoRootDir returns this checkout's top-level directory via `git
// rev-parse --show-toplevel`, run from the current working directory —
// local git metadata only, no network.
func repoRootDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// skipDirs names directories copyTree/countTree never descend into:
// version control metadata and any prior codegraph index, so a
// benchmark work directory always starts from a clean, indexable
// source tree.
var skipDirs = map[string]bool{
	".git":       true,
	".codegraph": true,
}

// copyTree copies src's directory tree into dst (created if needed),
// skipping skipDirs and symlinks (a symlinked directory reports
// IsDir()==false from WalkDir's Lstat-based fs.DirEntry, which would
// otherwise make copyFile try — and fail — to open it as a regular
// file; skipping is simpler and safer than resolving link targets that
// may point outside src). Used to give each (entry, subject) pair its
// own isolated work directory so the Go binary's Pebble store and the
// TS binary's SQLite store never collide, and so neither writes into a
// resolved sibling-checkout source directory the operator owns.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A single unreadable entry (permission-denied, broken
			// symlink, etc.) shouldn't abort copying an entire
			// real-world repo; skip it and keep going.
			fmt.Fprintf(os.Stderr, "runner: copyTree: skipping %s: %v\n", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// countTree returns the number of files and total byte size under dir,
// skipping skipDirs, so throughput (files/sec, bytes/sec) is always
// computed from a self-measured, subject-agnostic count rather than
// trusting either binary's own self-reported stats.
func countTree(dir string) (files int, bytes int64, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() != filepath.Base(dir) && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}
