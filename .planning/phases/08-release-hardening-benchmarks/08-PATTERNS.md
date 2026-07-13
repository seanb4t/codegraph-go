# Phase 8: Release Hardening & Benchmarks - Pattern Map

**Mapped:** 2026-07-13
**Files analyzed:** ~14 new/modified files (grouped by area)
**Analogs found:** 4 strong / 14 (the CI/release/GoReleaser layer has no in-repo analog by design — see "No Analog Found")

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/bench/rss.go` (OS-level peak-RSS capture) | utility | batch/measurement | `tools/spike/corpus.go` (git history, commit `e5da8e7`) | role-match (measurement helper, not RSS-specific, but same "small pure helper feeding a benchmark" shape) |
| `internal/bench/metrics.go` (Metrics struct, JSON baseline) | model | file-I/O | `internal/version/version.go` (`VersionInfo` struct, json-tagged) | role-match (plain json-tagged struct + accessor) |
| `internal/bench/regression.go` (tolerance-band gate) | service | transform | none in-repo | no analog — RESEARCH.md Pattern 3 is authoritative |
| `tools/bench/gencorpus/main.go` (synthetic 100k+ corpus generator) | utility | batch/file-I/O | `tools/spike/main.go` (git history) — CLI-style standalone `package main` tool with corpus generation helpers (`cases.go`'s `deepNestingGo`/`randomBytes`) | role-match |
| `tools/bench/realcorpus/manifest.go` (pinned real-repo manifest) | config | file-I/O | `tools/spike/corpus.go` + `tools/spike/testdata/ATTRIBUTION.md` (git history) | exact (same pinned-commit-fixture pattern, same problem) |
| `tools/bench/runner/main.go` (head-to-head CLI: shells out to Go + TS binaries) | service | request-response/batch | `tools/spike/bench_test.go` (git history) — same "drive backend(s) through a shared measurement path, report numbers" shape, but that file used `testing.B`; this is a standalone CLI, not a `go test` benchmark | role-match |
| `internal/upgrade/verify_release_e2e_test.go` (real-signed-artifact e2e test) | test | request-response | `internal/upgrade/verify_test.go` | exact |
| `internal/version/version.go` (no code change; ldflags target) | config | — | itself (already correct — confirm symbol paths only) | exact (no changes needed, just verified as ldflags target) |
| `internal/cli/bench.go` (optional new `codegraph bench` subcommand, if added) | controller (CLI command) | request-response | `internal/cli/version.go` | exact |
| `internal/cli/root.go` (add `newBenchCmd()` to `AddCommand(...)`, if bench subcommand added) | controller (CLI wiring) | request-response | itself — existing `AddCommand` call at lines 45-50 | exact |
| `.goreleaser.yaml` | config | build | none in-repo | no analog — RESEARCH.md "Code Examples" section is authoritative |
| `.github/workflows/release.yml` | config | event-driven (CI) | none in-repo | no analog — locked by `internal/upgrade/verify.go` contract; RESEARCH.md Finding 1/2 + Code Examples authoritative |
| `.github/workflows/ci.yml` | config | event-driven (CI) | none in-repo | no analog |
| `.github/workflows/bench.yml` | config | event-driven (CI) | none in-repo | no analog |
| `docs/RELEASE.md`, `docs/BENCHMARKS.md` | doc | — | none in-repo | no analog — plain docs, no code pattern to copy |

## Pattern Assignments

### `internal/bench/rss.go` / `internal/bench/metrics.go` (utility + model, batch measurement)

**Analog:** `internal/version/version.go` (struct shape) + RESEARCH.md's own Pattern 1 code (already a concrete, ready-to-use example — reproduced here as the load-bearing excerpt planners should copy verbatim since no in-repo file does this yet).

**Struct/accessor pattern to copy** (`internal/version/version.go` lines 24-46):
```go
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func Info() VersionInfo {
	return VersionInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}
```
Apply the same shape for `bench.Metrics` (json-tagged, a pure data-holder built by a single `Capture()`-style function, no hidden state).

**RSS-capture pattern to copy verbatim (RESEARCH.md Pattern 1 — no in-repo predecessor, this literally is the pattern):**
```go
func peakRSSBytes(state *os.ProcessState) (int64, error) {
	ru, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0, fmt.Errorf("bench: platform does not expose syscall.Rusage")
	}
	switch runtime.GOOS {
	case "linux":
		return ru.Maxrss * 1024, nil // Linux: ru_maxrss is in KB
	case "darwin":
		return ru.Maxrss, nil // Darwin: ru_maxrss is already in bytes
	default:
		return 0, fmt.Errorf("bench: unsupported OS for RSS measurement: %s", runtime.GOOS)
	}
}
```
**Error handling pattern:** match `internal/version`'s doc-comment discipline — state explicitly what this package does NOT do (no network, no crypto) the way `version.go`'s package doc states its own non-surface; do the same for `bench` (e.g. "this package never shells out itself; callers own the `exec.Cmd`").

---

### `tools/bench/realcorpus/manifest.go` (pinned real-repo manifest — config, file-I/O)

**Analog:** `tools/spike/testdata/ATTRIBUTION.md` + `tools/spike/corpus.go` (recoverable via `git show e5da8e7:tools/spike/corpus.go` and `git show e5da8e7:tools/spike/testdata/ATTRIBUTION.md` — the spike directory was deliberately removed in commit `8cf8d51` after the CGo-vs-wazero decision was ratified, but its pattern is exactly what this phase's corpus manifest should reuse).

**Provenance-doc pattern to copy** (`tools/spike/testdata/ATTRIBUTION.md`, git history):
```markdown
## Go corpus — `spf13/cobra`
- Source: https://github.com/spf13/cobra
- Pinned ref: tag `v1.8.1`
- Pinned commit: `e94f6d0dd9a5e5738dca6bce03c4b1207ffbc0ec`
- License: Apache-2.0
- Selection: all top-level `*.go` files excluding `*_test.go` (14 files, ~220 KiB)
- Path: `tools/spike/testdata/go/`
```
Every corpus entry MUST record source URL, pinned tag AND commit SHA, license, and exact file-selection rule — this is the reproducibility discipline CONTEXT.md D-04 requires extended to the head-to-head PERF-01 corpus.

**Embed + load pattern to copy** (`tools/spike/corpus.go`, git history, in full above the code block header):
```go
//go:embed testdata/go/*.go
var goCorpusFS embed.FS

type corpusFile struct {
	Name   string
	Source []byte
}

func loadCorpus(fsys embed.FS, dir string) ([]corpusFile, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	files := make([]corpusFile, 0, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(fsys, dir+"/"+name)
		if err != nil {
			return nil, err
		}
		files = append(files, corpusFile{Name: name, Source: b})
	}
	return files, nil
}
```
Key discipline this demonstrates and the new corpus manifest must preserve: `go:embed` the fixture bytes directly into the binary/test binary so behavior is identical regardless of invocation `cwd` — this matters even more for `tools/bench/runner` since it will be invoked both via `go test` and via a standalone CI step.

**Note on real-repo reuse:** `testdata/golden/corpus/colbymchenry-codegraph` and `testdata/golden/corpus/weft-go` (still present in-repo, not removed) are additional pinned fixtures worth considering as PERF-01 head-to-head inputs per CONTEXT.md D-04 — check their `ATTRIBUTION`-equivalent provenance before reusing, and follow the same recorded-commit-SHA discipline.

---

### `tools/bench/gencorpus/main.go` (synthetic 100k+ file generator — utility, batch/file-I/O)

**Analog:** `tools/spike/main.go` + `tools/spike/cases.go` (git history, commit `e5da8e7`) — the closest existing "standalone `package main` tool that deterministically generates adversarial/synthetic source" precedent in this repo.

**Deterministic-generation pattern to copy** (`tools/spike/cases.go`, git history):
```go
func malformedCase(name string) ([]byte, bool) {
	switch name {
	case "truncated_go":
		return truncatedSource(mustLoadGoCorpus(), 2), true
	case "garbage":
		return randomBytes(64 * 1024), true
	case "deep_nesting_go":
		return deepNestingGo(2_000_000), true
	...
	default:
		return nil, false
	}
}
```
Apply the same "named-case dispatch over small deterministic generator functions" shape to `gencorpus`, but seed the RNG explicitly (`rand.New(rand.NewSource(fixedSeed))`, per RESEARCH.md Pattern 2 — `tools/spike/cases.go`'s `randomBytes` did NOT need a fixed seed since it was adversarial-only, not baseline-comparable; `gencorpus` output feeds a committed baseline JSON and MUST be byte-reproducible run-to-run, which is a stricter requirement than the spike's).

**Doc-comment discipline to copy:** every generator function in `tools/spike/cases.go` documents *why* that specific adversarial shape exists (e.g. `truncatedSource`'s comment on "classic truncated-transfer / partial-read adversarial shape") — apply the same standard to `gencorpus`'s cross-file-reference generation logic (RESEARCH.md Pattern 2 requires real imports/calls between generated symbols, not zero-edge files — document why each generated construct exists).

---

### `internal/upgrade/verify_release_e2e_test.go` (real-signed-artifact e2e test)

**Analog:** `internal/upgrade/verify_test.go` (exact — same package, same functions under test, extends the existing fixture-based test file's pattern to a real/CI-produced artifact instead of the embedded sigstore-js fixture).

**Fixture-loading pattern to copy** (`internal/upgrade/verify_test.go` lines 1-55):
```go
const (
	fixtureSanRegex        = "^https://github.com/sigstore/sigstore-js/"
	fixtureArtifactSHA512  = "46d4e2f7...5cd3"
	fixtureDigestAlgorithm = "sha512"
)

func loadFixtureBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "valid-bundle.json"))
	if err != nil {
		t.Fatalf("read fixture bundle: %v", err)
	}
	b, err := loadBundle(data)
	if err != nil {
		t.Fatalf("parse fixture bundle: %v", err)
	}
	return b
}

func loadFixtureTrustedRoot(t *testing.T) root.TrustedMaterial {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "trusted-root.json"))
	...
}
```
**Core assertion pattern to copy** (`verify_test.go` `TestVerifyRelease_AcceptsValidBundle`, lines 71-84):
```go
func TestVerifyRelease_AcceptsValidBundle(t *testing.T) {
	b := loadFixtureBundle(t)
	tr := loadFixtureTrustedRoot(t)
	digest := mustHexDecode(t, fixtureArtifactSHA512)
	if err := verifyRelease(b, tr, fixtureDigestAlgorithm, digest, fixtureSanRegex); err != nil {
		t.Fatalf("verifyRelease: expected nil error for a valid bundle, got: %v", err)
	}
}
```
**For the new e2e test:** replace the embedded sigstore-js fixture with a real CI-produced (or `checkpoint:human-verify`-gated manually captured) binary + `.sigstore.json` bundle downloaded from an actual `seanb4t/codegraph-go` release, and assert against the PRODUCTION identity constants (`releaseOIDCIssuer`, `releaseWorkflowRefPattern` from `internal/upgrade/verify.go` lines 41-45), not the fixture `sigstoreSanRegex`/`fixtureDigestAlgorithm` used by the existing offline tests — this is the exact seam RESEARCH.md Finding 1's action item calls for.

**Exact production constants to assert against** (`internal/upgrade/verify.go` lines 41-45):
```go
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
)
```

**`releaseAssetName`/`defaultDownload` contract the release workflow's output must satisfy** (`internal/upgrade/upgrade.go` lines 175-197):
```go
func defaultDownload(version string) (binary []byte, bundleJSON []byte, err error) {
	assetName := releaseAssetName(version)
	binary, err = downloadReleaseAsset(version, assetName)
	...
	bundleJSON, err = downloadReleaseAsset(version, assetName+".sigstore.json")
	...
}

func releaseAssetName(version string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("codegraph_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext)
}
```
`defaultVerify` (`upgrade.go` lines 153-164) confirms the digest is over the **raw downloaded binary** (`sha256.Sum256(binary)`), not any archive/checksums file — this is Finding 1's central point and the e2e test must exercise this exact call path.

---

### `internal/cli/bench.go` + `internal/cli/root.go` (optional `codegraph bench` CLI subcommand)

**Analog:** `internal/cli/version.go` (exact match — simplest existing command, same "delegate to a package function, format for humans or `--json`" shape).

**Command-builder pattern to copy** (`internal/cli/version.go` lines 15-37):
```go
func newVersionCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print build version information",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Info()
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "codegraph %s (commit %s, built %s) %s %s/%s\n",
				info.Version, info.Commit, info.Date, info.GoVersion, info.OS, info.Arch)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit build identity as JSON")
	return cmd
}
```
Apply identically for `newBenchCmd()`: delegate all logic to `internal/bench`/`tools/bench/runner`, support `--json`, no business logic in the CLI layer itself.

**Root registration pattern to copy** (`internal/cli/root.go` lines 45-50):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
	newUpgradeCmd(), newInstallCmd(), newUninstallCmd(), newMigrateCmd())
```
Add `newBenchCmd()` to this exact list (single flat `AddCommand` call, alphabetically-loose grouping by feature area) if a bench subcommand is added; `cmd/codegraph/main.go` requires no change (it only calls `cli.Execute()`).

---

## Shared Patterns

### Fail-loud / no-panic in production paths
**Source:** `internal/upgrade/verify.go` (`verifyRelease` returns errors, never panics; `fetchTrustedRoot` wraps every error with `fmt.Errorf("upgrade: ...: %w", err)`), `internal/version/version.go` (defaults to `dev`/`unknown` rather than panicking on unset ldflags).
**Apply to:** `internal/bench` (RSS capture, regression gate) and `tools/bench/*` — every measurement/comparison function should return `(T, error)`, never `panic`, so a single bad benchmark run fails that CI step loudly instead of crashing the whole gate ambiguously. (Note: `tools/spike/corpus.go`'s `mustLoadGoCorpus` DOES panic — that was acceptable for a removed spike tool with no production caller; do NOT copy the `must*`-panics pattern into `tools/bench`, which IS a production CI gate.)

### Doc-comment discipline (package + exported-symbol rationale)
**Source:** every file read in this phase (`verify.go`, `upgrade.go`, `version.go`, `cli/version.go`, `tools/spike/*`) opens with a multi-line doc comment stating not just what the code does but *why* (e.g. `verify.go`'s "never shells out to a cosign CLI", `version.go`'s "this package has no network or crypto surface").
**Apply to:** All new files in this phase, especially `.goreleaser.yaml`/workflow YAML comments — state the contract being satisfied (e.g. "asset name here MUST match `releaseAssetName` in internal/upgrade/upgrade.go") directly in the config file's comments, not only in docs/RELEASE.md.

### Embedded, pinned-commit test/bench fixtures (no network at test/CI time)
**Source:** `tools/spike/corpus.go` (`go:embed`) + `tools/spike/testdata/ATTRIBUTION.md` (git history) + `internal/upgrade/verify_test.go`'s `testdata/valid-bundle.json`/`testdata/trusted-root.json` (both fully offline fixtures).
**Apply to:** `tools/bench/realcorpus` (real OSS repos, pinned by commit SHA, embedded or committed as static files) and `internal/upgrade/verify_release_e2e_test.go` (if a real signed artifact is captured once and committed as a fixture rather than fetched live in every CI run — confirm with planner whether the e2e test fetches from a real GitHub Release each run or uses a captured fixture; either way, follow this repo's established "pin + commit, no network at test time unless explicitly testing the network path" discipline).

## No Analog Found

Files with no close in-repo match — the RESEARCH.md "Code Examples" section (Finding 1/2's YAML blocks) is the authoritative reference instead:

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `.goreleaser.yaml` | config | build | No GoReleaser config exists yet anywhere in the repo; RESEARCH.md's "Raw-binary archive + per-binary signing" and "Cross-platform CC/CXX selection" code blocks are the reference. |
| `.github/workflows/release.yml` | config | event-driven (CI) | No `.github/workflows/` directory exists yet. Contract dictated by `internal/upgrade/verify.go` (name/trigger/SAN) — treat that file as the executable spec, per CONTEXT.md's own framing. |
| `.github/workflows/ci.yml` | config | event-driven (CI) | Same — no existing workflow to pattern-match; must wire `go test ./...`, `govulncheck ./...`, the double-build gate, and the perf-regression gate. |
| `.github/workflows/bench.yml` | config | event-driven (CI) | Same — on-demand/scheduled trigger, no existing precedent in-repo. |
| `internal/bench/regression.go` | service | transform | No existing tolerance-band/baseline-diff code in this repo; RESEARCH.md Pattern 3's `checkRegression` example is the reference implementation to start from. |
| `docs/RELEASE.md`, `docs/BENCHMARKS.md` | doc | — | Plain documentation, no code pattern applicable. |

## Metadata

**Analog search scope:** `internal/upgrade/`, `internal/version/`, `internal/cli/`, `cmd/codegraph/`, `tools/spike/` (recovered via `git show`/`git log` — directory removed from working tree in commit `8cf8d51` after Phase-1 parser decision was ratified), `testdata/golden/corpus/`.
**Files scanned:** ~20 (upgrade package: 8 files; cli package: 24 files, 6 read in detail; version package: 1 file; spike package: 6 files recovered from git history at commit `e5da8e7`).
**Pattern extraction date:** 2026-07-13
**Key constraint surfaced during mapping:** `tools/spike/` no longer exists in the working tree (removed post-Phase-1) but is fully recoverable via `git show e5da8e7:tools/spike/<file>` — planner/executor should note this when writing plan actions that say "copy from tools/spike/X" (the file must be pulled from git history, not read from the working directory).
