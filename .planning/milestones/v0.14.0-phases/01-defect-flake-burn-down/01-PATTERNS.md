# Phase 1: Defect & Flake Burn-down - Pattern Map

**Mapped:** 2026-09-14
**Files analyzed:** 13 (5 new, 8 modified)
**Analogs found:** 13 / 13 (one, the icon assets, has no *code* analog — see `## No Analog Found`)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `web/scripts/graph-console-check.mjs` (new) | test (live-browser script) | event-driven (page/console events) | `web/scripts/graph-expand-check.mjs` (record/write shape) + `web/scripts/breadcrumb-check.mjs` (binary-boot shape) + `web/scripts/graph-live-update-check.mjs` (corpus resolution) | exact (composite of 3 sibling scripts, same directory/convention family) |
| `internal/cli/index_lock_test.go` (new, name per CONTEXT.md's own suggestion) | test | request-response (CLI command invocation) | `internal/graphstore/open_lock_test.go` (lock-hold shape) + `internal/cli/cli_test.go`'s `execCmd`/`execCmdWithInput` (in-process cobra invocation) | exact |
| `scripts/write-multiline-output.sh` (new, name at Claude's Discretion per D-17) + its Taskfile test target | utility / test | transform (shell heredoc write) | `scripts/inject-cosign-key.sh` (extract-from-workflow-into-script-plus-Taskfile-target precedent) | role-match (no exact prior shell-test-of-workflow-logic precedent exists) |
| `corpora/graph-console-check.json` (new, committed observation) | config (committed data) | batch (one-shot verdict write) | `corpora/graph-expand-check.json` (JS check script's own write shape) + `corpora/graph-cluster-observations.json` (v0.13.0 GRF-09 schema precedent) | exact |
| `web/static/favicon.svg`, `web/static/favicon-32.png`, `web/static/apple-touch-icon.png` (new static assets) | config (static asset) | file-I/O (static serve) | `web/static/robots.txt` (only existing file in `web/static/`) — no code analog exists | no code analog (see below) |
| `internal/cli/index.go` (modified: `priorCoverageGeneration`, `newIndexCmd` RunE) | controller (CLI command) | CRUD (store lifecycle) | `internal/daemon/lock.go`'s `acquire()` (message-parity precedent) + `internal/graphstore/pebble_store.go` (sentinel definitions) | exact (message shape) / role-match (control flow) |
| `internal/daemon/watchdog.go`, `watchdog_posix.go`, `daemon.go` (modified: per-instance `getppid`/ticker seam) | service (background goroutine + test seam) | event-driven (ticker-driven poll) | `internal/daemon/daemon.go`'s existing `onSync`/`onSyncStart`/`syncFn`/`onWatchOpen` unexported-field seam convention, and `WithProbe` (the one exported `Option`) | exact |
| `internal/daemon/daemon_test.go` / `watchdog_test.go` (modified: seam injection, signal-based assertion) | test | event-driven | `TestRunWatchdogCancelsRunOnSimulatedReparent` (current shape, being rewritten) + `internal/daemon/testbudget_test.go`'s `joinDaemonRun` | exact |
| `internal/bench/regression.go` (modified: `Repo` guard) | service (comparison gate) | transform (struct comparison) | The existing `ScratchFS` guard in the same file (`:91-110`) | exact |
| `internal/bench/regression_test.go` (modified: `Repo` subtests) | test | transform | The existing `scratch_fs mismatch` table-driven subtests (`:433-524`) | exact |
| `.github/workflows/require-issue-link.yml` + `.github/workflows/pr-template-format.yml` (modified: delimiter) | config (CI workflow) | event-driven (`pull_request_target`) | Each other (byte-identical vulnerable block) + `scripts/inject-cosign-key.sh` (extraction-into-script precedent) | exact |
| `web/src/lib/components/graph/GraphCanvas.svelte` (modified: FIX-04 teardown guard) | component | event-driven (cytoscape lifecycle) | Its own existing `layoutGeneration`/`activeLayoutRun` in-flight-token pattern (`:264-375`) | exact (self-analog — extend the pattern already in the file) |
| `web/src/lib/components/graph/graph-style.ts` (modified: FIX-05 `text-valign: 'right'`) | config (cytoscape stylesheet) | transform (style rule) | Sibling selectors in the same file (`:99-111`, valid `text-valign`/`text-halign` pairs) | exact |
| `web/src/routes/+layout.svelte` (modified: favicon links) | component (SvelteKit layout) | request-response (static asset link) | Its own current `<svelte:head>` block (`:7`, `:91`) | exact (self-analog) |

## Pattern Assignments

### `web/scripts/graph-console-check.mjs` (new test, event-driven)

**Analogs:** `web/scripts/graph-expand-check.mjs`, `web/scripts/breadcrumb-check.mjs`, `web/scripts/graph-live-update-check.mjs` — all in `web/scripts/`, all real-Chromium checks with the identical `repoRoot()`/`pollUntil()`/diagnostic-JSON-on-every-exit-path convention this repo's own `06-PATTERNS.md` already documents.

**Imports pattern** (`graph-expand-check.mjs:35-40`):
```js
import { chromium } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

import { readProtocol } from './graph-measure.mjs';
```
`breadcrumb-check.mjs` additionally imports `{ spawn }` from `node:child_process` (needed here too, to boot the real binary) and `graph-live-update-check.mjs` additionally imports `node:crypto` (needed here too, to resolve the pinned guava corpus directory).

**Boot-the-real-binary pattern** (`breadcrumb-check.mjs:66-120`, verbatim):
```js
function startCodegraphUi(binaryPath, repoPath, editorUrl) {
	return new Promise((resolve, reject) => {
		const cliArgs = ['ui', '--no-open', '--path', repoPath];
		if (editorUrl) {
			cliArgs.push('--editor-url', editorUrl);
		}
		const child = spawn(binaryPath, cliArgs, { stdio: ['ignore', 'pipe', 'pipe'] });
		let stdout = '';
		let settled = false;
		const timer = setTimeout(() => {
			if (!settled) {
				settled = true;
				reject(new Error('breadcrumb-check: codegraph ui printed no URL within 15s'));
			}
		}, 15000);
		child.stdout.on('data', (chunk) => {
			stdout += chunk.toString();
			const m = stdout.match(/https?:\/\/\S+/);
			if (m && !settled) {
				settled = true;
				clearTimeout(timer);
				resolve({ child, url: m[0].trim() });
			}
		});
		child.on('exit', (code) => {
			if (!settled) {
				settled = true;
				clearTimeout(timer);
				reject(new Error(`breadcrumb-check: codegraph ui exited early (code ${code})`));
			}
		});
		child.on('error', (err) => {
			if (!settled) { settled = true; clearTimeout(timer); reject(err); }
		});
	});
}
```
Pair with `stopChild(child)` (`breadcrumb-check.mjs:122-135`) for teardown: `SIGTERM`, then `SIGKILL` after a 3s grace window if the child hasn't exited.

**Corpus resolution pattern** (guava, pinned repo/sha) (`graph-live-update-check.mjs:57-90`, verbatim):
```js
function corpusRoot() {
	if (process.env.CODEGRAPH_CORPUS_DIR) return process.env.CODEGRAPH_CORPUS_DIR;
	if (process.env.XDG_CACHE_HOME) return path.join(process.env.XDG_CACHE_HOME, 'codegraph', 'corpora');
	const home = process.env.HOME;
	if (!home) throw new Error('...: HOME is not set and no cache override is configured');
	return path.join(home, '.cache', 'codegraph', 'corpora');
}

function corpusDir(repo, sha) {
	const slug = repo.replace(/\//g, '-');
	const digest = crypto.createHash('sha256').update(repo).digest('hex').slice(0, 8);
	return path.join(corpusRoot(), `${slug}-${digest}@${sha}`);
}
```
This repo's own graph (no corpus resolution needed) is the second target: point `--path` at `repoRoot()` itself, exactly as `graph-live-update-check.mjs` does for its own-repo control run.

**`pageerror` capture pattern** (`graph-expand-check.mjs:359-365`, verbatim — registered BEFORE navigation, every existing check does this):
```js
const page = await context.newPage();
page.on('pageerror', (err) => {
	pageErrors.push(err.message);
});
await page.goto(args.url, { waitUntil: 'load', timeout: protocol.navigationTimeoutMs });
```

**Gap — `console.warn`/`console.error` capture has NO existing precedent in this codebase.** None of the five sibling scripts register `page.on('console', ...)` — they only ever collect `pageerror`. `graph-console-check.mjs` is the first script to need this; use Playwright's own `page.on('console', msg => ...)` API (`msg.type()` is `'warning'`/`'error'`, `msg.text()` carries the message, e.g. `Edge \`<id>\` has invalid endpoints...` per 01-RESEARCH.md Investigation 2) — there is no in-repo pattern to copy for this half, only the Playwright API itself.

**Verdict-write pattern** (`graph-expand-check.mjs:475-493`, verbatim shape to copy):
```js
const record = {
	success: true,
	/* ...check-specific fields... */,
	pageErrorCount: pageErrors.length,
	pageErrors,
	browserIdentity: { name: 'chromium', version: browserVersion }
};
await fs.promises.mkdir(path.dirname(args.out), { recursive: true });
fs.writeFileSync(args.out, JSON.stringify(record, null, 2) + '\n');
process.stdout.write(`...: wrote ${args.out} (...)\n`);
```
Output path convention: `path.join(repoRoot(), 'corpora', '<script-name>.json')` (every sibling script's default `--out`, e.g. `graph-live-update-check.mjs:340`).

**Entry-point pattern** (identical across all five, `graph-expand-check.mjs:513-519`):
```js
const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`...: fatal — ${err instanceof Error ? err.message : String(err)}`);
		process.exit(1);
	});
}
```

**Taskfile wiring — no existing precedent for these five scripts.** Contrary to the phase brief's assumption, `Taskfile.yml` has **zero** references to any of `graph-expand-check.mjs`, `breadcrumb-check.mjs`, `graph-collapse-affordance-check.mjs`, `graph-live-update-check.mjs`, or `live-push-multitab-check.mjs` — they are run standalone (`node web/scripts/<name>.mjs --url ... --out ...`), not wired into any Task target today. The one Node-script Task target that exists is `check:no-force-layout` (`Taskfile.yml:1796-1814`, a static scan, not a live-browser check):
```yaml
check:no-force-layout:
  preconditions:
    - sh: command -v node
      msg: "node not found — ..."
  cmds:
    - node web/scripts/check-no-force-layout.mjs --self-test
    - node web/scripts/check-no-force-layout.mjs
```
D-09 still requires a Task target for the new script (it is the one thing this phase asks for that has no existing sibling to copy exactly) — model it on `check:no-force-layout`'s precondition/cmds shape, but add a precondition for the built `codegraph` binary existing (mirrors `breadcrumb-check.mjs:220`'s own runtime check: `binary not found at ${binaryPath} — run \`task build:release\` first`) and pass `--url`/`--out` or let the script boot the binary itself per D-09's "boots the real binary" wording.

---

### `internal/cli/index_lock_test.go` (new test, request-response)

**Analog:** `internal/graphstore/open_lock_test.go` (`TestOpenSecondOpenInProcessReturnsErrStoreLocked`, lines 11-46) for the lock-hold shape; `internal/cli/cli_test.go` (`execCmd`/`execCmdWithInput`, lines 55-70) for in-process CLI invocation; `internal/cli/index_test.go` (`readCoverageGeneration`, lines 16-35; `TestIndexForceRebuildBumpsCoverageGeneration`, line 53+) for the sibling CLI-level test's own helper/fixture conventions.

**Lock-hold pattern to copy** (`open_lock_test.go:23-46`, verbatim):
```go
func TestOpenSecondOpenInProcessReturnsErrStoreLocked(t *testing.T) {
	dir := t.TempDir()

	holder, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer holder.Close()

	start := time.Now()
	second, err := Open(dir)
	elapsed := time.Since(start)

	if err == nil {
		second.Close()
		t.Fatal("second Open succeeded while the first store was still open; want lock-held failure")
	}
	if !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("second Open error = %v; want errors.Is(err, ErrStoreLocked)", err)
	}
	...
}
```

**CLI-level equivalent shape (composite of the two analogs above):** hold a `graphstore.Open(storeDir)` handle in the test (same `.codegraph/store` path `index.go:89` computes), then invoke `codegraph index --force` in-process via `execCmd("index", dir, "--force")` (`cli_test.go:55-70`'s exact pattern — `newRootCmd()`, `SetArgs`, `Execute()`), and assert (a) `err` is non-nil, (b) `errors.Is`-style text or a sentinel wrap names `graphstore.ErrStoreLocked`, and (c) the held store's on-disk contents are untouched (e.g. `readGraphCounts`/`readCoverageGeneration`, `index_test.go:16-35`, read the same values before and after the refused call) — proving `os.RemoveAll` never ran.

**Imports pattern** (`index_test.go:1-11`, verbatim):
```go
package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
)
```

**RED-before-GREEN requirement (rule `84d1gfpywd`, D-12):** this test must be run once against the current, pre-fix `priorCoverageGeneration`/`RunE` shape (which collapses every error to floor-0 and proceeds to `RemoveAll` regardless) and observed to fail, with that transcript captured in the plan's verify gates — mirroring how `TestOpenSecondOpenInProcessReturnsErrStoreLocked` itself pins a fix that was already reviewed for exactly this discipline (`03-REVIEW-2.md WR-02`, cited in its own doc comment).

---

### `scripts/write-multiline-output.sh` (new utility) + Taskfile test target (D-17)

**Analog:** `scripts/inject-cosign-key.sh` — the one existing precedent in this repo for "extract a piece of shell logic out of a workflow/build step into a standalone script specifically so it can be exercised via a `task` target."

**Script header/contract pattern to copy** (`inject-cosign-key.sh:1-43`, verbatim shape):
```sh
#!/usr/bin/env bash
# scripts/<name>.sh
#
# <what this does, one paragraph>
#
# Usage: <name>.sh <arg1> <arg2> ...
#   $1  ...
#   $2  ...
#
# This script writes only to $2. It does not read, write, or reference
# <anything outside its own explicit args> ...

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $(basename "$0") <arg1> <arg2> <arg3>" >&2
  exit 2
fi

ARG1="$1"
ARG2="$2"
ARG3="$3"

if [ ! -r "${ARG1}" ]; then
  echo "::error::<name>.sh: cannot read ...: ${ARG1}" >&2
  exit 2
fi
```

**Taskfile invocation pattern** (`Taskfile.yml:2156`, `:2672` — same script called from two different rehearsal targets, exactly the "both workflows call the same script" structure D-17 needs):
```yaml
        GENERATED_CONFIG="${SIGNTEST_DIR}/.goreleaser.signtest.yaml"
        ./scripts/inject-cosign-key.sh .goreleaser.yaml "${GENERATED_CONFIG}" "${SIGNTEST_DIR}/cosign.key"
```
Use a `mktemp -d` scratch dir + `trap cleanup EXIT` (`Taskfile.yml:2130-2134`) for the test target: stub `gh` as a tiny script placed earlier on `PATH` (a shell function/script named `gh` that echoes a fixed file list including one `PRFILES_EOF`-named path), run the extracted script against that stub, then assert the resulting `$GITHUB_OUTPUT` file parses back to the exact, uncorrupted list.

**Both workflow call sites to replace** (byte-identical block in both files, verbatim):
```sh
{
  echo "list<<PRFILES_EOF"        # or "files<<PRFILES_EOF" in pr-template-format.yml
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "PRFILES_EOF"
} >> "$GITHUB_OUTPUT"
```
Exact locations: `.github/workflows/require-issue-link.yml:39-50` (`echo "list<<PRFILES_EOF"`) and `.github/workflows/pr-template-format.yml:39-46` (`echo "files<<PRFILES_EOF"`) — the output key name (`list` vs `files`) differs, so the shared script needs the key name as a parameter.

**No existing bats/shellcheck framework** — do not introduce one (see `## Don't Hand-Roll` carried over from RESEARCH.md); a plain script + Task target is the sanctioned shape.

---

### `corpora/graph-console-check.json` (new committed observation)

**Analog:** the JS-side write is `graph-expand-check.mjs`'s own `record`/`fs.writeFileSync` pattern (already quoted above under `graph-console-check.mjs`). The schema-precedent analog is `corpora/graph-cluster-observations.json` (v0.13.0 GRF-09):
```json
{
  "schemaVersion": 1,
  "thresholdRef": "corpora/graph-cluster-threshold.json",
  "thresholdDigest": "sha256:...",
  "corpus": { "repo": "google/guava", "sha": "94f39958baf7ad51ddf9c70e406ed6b188194daa" },
  ...
}
```
Written from Go via (`tools/graphcluster/main.go:288-298`):
```go
func writeObservation(path string, obs observation) error {
	data, err := json.MarshalIndent(obs, "", "  ")
	...
	if err := os.WriteFile(path, data, 0o644); err != nil {
	...
}
```
`graph-console-check.mjs` is a JS script, not a Go tool — follow `graph-expand-check.mjs`'s `fs.writeFileSync(args.out, JSON.stringify(record, null, 2) + '\n')` shape (already established in this directory), not the Go `writeObservation` shape; the `graph-cluster-observations.json` precedent is cited for its "schema versioned, corpus-identified, committed under `corpora/`" *policy*, not its Go implementation.

---

### `web/static/favicon.svg`, `favicon-32.png`, `apple-touch-icon.png` (new static assets) + `web/src/routes/+layout.svelte` (modified)

**No code analog** — `web/static/` today holds only `robots.txt`, a static text file with no accompanying serve logic (SvelteKit's adapter-static serves everything under `static/` at the site root automatically; there is no `internal/uiserver` route registered for it, confirmed no `robots.txt`-specific handler exists). The icon files are the same kind of asset: drop-in static files, no server code needed.

**Current `+layout.svelte` shape to remove** (`+layout.svelte:7`, `:91`, verbatim):
```svelte
import favicon from '$lib/assets/favicon.svg';
...
<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>
```
**New shape** (D-04): delete the `import favicon from '$lib/assets/favicon.svg'` line and `web/src/lib/assets/favicon.svg` itself; replace the `<svelte:head>` block with three static `<link>` tags pointing at `/favicon.svg`, `/favicon-32.png` (`sizes="32x32" type="image/png"`), and `/apple-touch-icon.png` (`rel="apple-touch-icon"`) — plain root-relative paths, the same way `web/static/robots.txt` is reachable at `/robots.txt` with zero routing code.

**CSP — do not touch:** `internal/uiserver/spa.go:99`'s `spaCSPBaseDirectives` (`"default-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"`) and `spa_test.go:439`'s exact `default-src 'self'` assertion are both byte-identical before and after this fix (D-05) — static same-origin files need no CSP change; this is a negative-space pattern (prove nothing here moved), not something to copy.

---

### `internal/cli/index.go` (modified: `priorCoverageGeneration`, `newIndexCmd` RunE) — FIX-06

**Current shape to replace** (`index.go:30-48`, verbatim, quoted in full in 01-RESEARCH.md Investigation 4):
```go
func priorCoverageGeneration(storeDir string) int64 {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return 0
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		return 0
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		return 0
	}
	return meta.GetCoverageGeneration()
}
```
Call site to change (`index.go:101-107`):
```go
coverageGenerationFloor := priorCoverageGeneration(storeDir)
...
if err := os.RemoveAll(storeDir); err != nil {
	return err
}
```

**Message-parity pattern to mirror** (`internal/daemon/lock.go:208`, verbatim — this is the exact string-shape D-10 wants echoed):
```go
return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
```

**Sentinels to branch on** (`internal/graphstore/pebble_store.go:21`, `:110`):
```go
var ErrNotFound = errors.New("graphstore: not found")
var ErrStoreLocked = errors.New("graphstore: store lock held")
```

**New branching shape** (RESEARCH.md's own derived template, ready to adapt):
```go
coverageGenerationFloor, err := priorCoverageGeneration(storeDir) // signature changes to return error
switch {
case errors.Is(err, graphstore.ErrNotFound):
	// never indexed — floor stays 0, no message (D-11)
case errors.Is(err, graphstore.ErrStoreLocked):
	return fmt.Errorf("%w: another process holds the store — stop it first (`codegraph daemon stop`) "+
		"or run `codegraph unlock` once it has exited", graphstore.ErrStoreLocked) // D-10, before RemoveAll
case err != nil:
	fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not read the prior coverage generation (%v) — "+
		"rebuilding from floor 0\n", err) // D-11, corrupt/unreadable store
}
```
Keep this refuse branch strictly BEFORE the `os.RemoveAll(storeDir)` call (`index.go:107`) — that ordering is the entire fix.

---

### `internal/daemon/watchdog.go`, `watchdog_posix.go`, `daemon.go` (modified: per-instance seam) — FIX-07/FIX-08

**Analog:** the existing unexported-field test-seam convention already used four times in the same struct (`daemon.go:97-137`), NOT the one exported `Option` (`WithProbe`) — RESEARCH.md's own Investigation 3 explicitly recommends following the dominant (unexported-field) convention since this seam, like the other four, is test-only:

**Convention to copy** (`daemon.go:97-137`, representative excerpt — `onSyncStart`, structurally identical to what `getppid`/ticker need):
```go
// onSyncStart, when non-nil, is invoked at the very start of flush,
// before touchPending or indexer.Sync run. It is a test-only
// control seam (mirrors onSync) that lets daemon_test.go
// deterministically hold a flush "in flight" ... Production callers leave it nil.
onSyncStart func()
```
Direct-assignment-from-test-file pattern (`daemon_test.go:306-316`, current — to be migrated from the package-level `var` onto the new unexported field):
```go
origGetppid := getppid
t.Cleanup(func() { getppid = origGetppid })
const original = 424242
var current int32 = original
getppid = func() int { return int(atomic.LoadInt32(&current)) }
```

**Current package-level seam being removed** (`watchdog.go:1-13`, `21`, verbatim):
```go
const watchdogInterval = 1 * time.Second

// getppid is an unexported package-level test seam: ...
var getppid = os.Getppid
```
```go
func startWatchdog(ctx context.Context, cancel context.CancelFunc, interval time.Duration) (stop func()) {
	original := getppid()
	...
```
`watchdog_posix.go:14-16`:
```go
func parentChanged(original int) bool {
	return getppid() != original
}
```
Call site inside `Run` (`daemon.go:261-265`):
```go
ctx, cancel := context.WithCancel(ctx)
stop := startWatchdog(ctx, cancel, watchdogInterval)
```

**Fix shape (per D-13/D-14, evidence-backed in RESEARCH.md Investigation 3):** thread `getppid func() int` and a ticker source (an injectable `<-chan time.Time`, defaulting to `time.NewTicker(watchdogInterval).C` in production) as parameters into `startWatchdog`, stored as new unexported `Daemon` fields set directly by `daemon_test.go`/`watchdog_test.go` (no new exported `Option`, no exported setter — matches `onSync`/`onSyncStart`/`syncFn`/`onWatchOpen`'s own precedent exactly). `parentChanged` takes the injected func or becomes a method on whatever owns it. **No `watchdog_windows.go` exists in this tree** — do not create or look for one.

**Confirmed no data races currently reproduce** (`GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...`, live-run 2026-09-14) — this is a design-defect removal (D-13), not a currently-observed-race fix; do not skip it because nothing reproduces today (Pitfall in RESEARCH.md).

---

### `internal/daemon/daemon_test.go` / `watchdog_test.go` (modified) — FIX-07 signal-based assertion

**Analog:** `TestRunWatchdogCancelsRunOnSimulatedReparent` itself (`daemon_test.go:302-359`, quoted above under the seam section) is both the file being fixed and its own closest analog — the test's *shape* (seam override → trigger → assert) stays, only the mechanism changes from a real 1s ticker + `time.After(testBudget(10*time.Second))` deadline race to a directly-signaled tick channel the test controls.

**Join-discipline helper to keep using** (`testbudget_test.go:170-181`, verbatim):
```go
func joinDaemonRun(t *testing.T, cancel context.CancelFunc, runErr <-chan error) {
	t.Helper()
	t.Cleanup(func() {
		cancel()
		budget := testBudget(joinTimeoutBase)
		select {
		case <-runErr:
		case <-time.After(budget):
			t.Errorf("joinDaemonRun: spawned goroutine did not return within %s of ctx cancellation — goroutine leak (MAINT-01/MAINT-02): ...", budget)
		}
	})
}
```
Ordering contract to preserve (from the same file's doc comment): `joinDaemonRun` must be registered AFTER any `t.Cleanup` that restores a per-instance test seam, since `t.Cleanup` runs LIFO.

**D-14's own constraint:** do not add a second timeout bump to `TestRunWatchdogCancelsRunOnSimulatedReparent`'s existing `testBudget(10 * time.Second)` deadline (`daemon_test.go:344`) — replace the race against a real ticker with a directly-sent tick on the injected channel instead.

---

### `internal/bench/regression.go` (modified: `Repo` guard) — FIX-09

**Analog:** the `ScratchFS` guard in the same file, structurally identical to what `Repo` needs (`regression.go:91-110`, verbatim):
```go
if baseline.ScratchFS != current.ScratchFS {
    return fmt.Errorf(
        "bench: scratch filesystem mismatch: baseline was measured on scratch_fs %s but "+
            "this run is %s; the scratch filesystem class is as load-bearing to a throughput "+
            "comparison as the runner class or GOOS/GOARCH are (tools/bench/BASELINE.md), so "+
            "this comparison would be meaningless. An empty scratch_fs value means it "+
            "predates scratch_fs recording, which is not a wildcard match against a recorded "+
            "one — re-bless the baseline on this scratch_fs class instead of comparing across "+
            "them",
        scratchFSString(baseline.ScratchFS), scratchFSString(current.ScratchFS),
    )
}
```
**Degrade-empty helper to mirror** (`regression.go:172-183`, verbatim — add a `repoString` sibling):
```go
func scratchFSString(scratchFS string) string {
	if scratchFS == "" {
		return "(not recorded)"
	}
	return scratchFS
}
```
Place the new `Repo` guard alongside the other three (GOOS/GOARCH at `:56`, Runner at `:79`, ScratchFS at `:91`), before the `FilesPerSec <= 0`/`PeakRSSBytes <= 0` validity checks — same file, same ordering convention.

**Field already exists, no schema change needed:** `Metrics.Repo` (`metrics.go:9`) is already populated at every write site (`tools/bench/runner/main.go:370,691,759`) with a stable synthesized id — the guard is purely additive to `CheckRegression`, nothing upstream changes.

---

### `internal/bench/regression_test.go` (modified: `Repo` subtests) — FIX-09

**Analog:** the existing `scratch_fs mismatch` subtests in the same table (`regression_test.go:433-524`, verbatim structure to copy for `Repo`):
```go
{
	name: "scratch_fs mismatch between baseline and current fails even when runner and GOOS/GOARCH match",
	baseline: Metrics{ GOOS: "linux", GOARCH: "amd64", Runner: "ubuntu-latest", ScratchFS: "disk", FilesPerSec: 100.0, PeakRSSBytes: 500_000_000 },
	current:  Metrics{ GOOS: "linux", GOARCH: "amd64", Runner: "ubuntu-latest", ScratchFS: "tmpfs", FilesPerSec: 100.0, PeakRSSBytes: 500_000_000 },
	ceiling: ceiling,
	wantErr: true,
	errHint: "scratch",
},
{
	name: "matching scratch_fs on both sides passes",
	baseline: Metrics{ ...ScratchFS: "tmpfs"... },
	current:  Metrics{ ...ScratchFS: "tmpfs"... },
	ceiling: ceiling,
	wantErr: false,
},
{
	// Empty scratch_fs is "never recorded", not a wildcard — same treatment as an empty runner.
	name: "empty baseline scratch_fs against non-empty current scratch_fs fails",
	...
},
```
Add the mirror-image `Repo` set (mismatch fails, match passes, empty-vs-non-empty fails either direction, both-empty passes) into the same `t.Run(tt.name, ...)` table loop at `regression_test.go:541`.

---

### `.github/workflows/require-issue-link.yml` + `.github/workflows/pr-template-format.yml` (modified) — FIX-10

**Both files' vulnerable block, byte-identical in shape:**
`require-issue-link.yml:39-50`:
```yaml
      - name: Collect changed files
        id: files
        env:
          GH_TOKEN: ${{ github.token }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
        run: |
          set -euo pipefail
          {
            echo "list<<PRFILES_EOF"
            gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
            echo "PRFILES_EOF"
          } >> "$GITHUB_OUTPUT"
```
`pr-template-format.yml:34-46` (same shape, output key `files` not `list`).

**Fix shape (D-17):**
```sh
DELIM="PRFILES_$(openssl rand -hex 16)"
{
  echo "list<<${DELIM}"
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "${DELIM}"
} >> "$GITHUB_OUTPUT"
```
Both workflows must change together (Pitfall in RESEARCH.md: fixing only the one GH #15 names by title and missing the other is the named failure mode) — extracting into the shared `scripts/write-multiline-output.sh` (see above) makes "fix only one" structurally hard, since both `run:` blocks then call the same script.

**`permissions:` blocks — do not touch:** both files already carry minimal `contents: read` / `issues: write` / `pull-requests: read` with no `contents: write` (confirmed present, unchanged by this fix).

---

### `web/src/lib/components/graph/GraphCanvas.svelte` (modified) — FIX-04

**Self-analog: the file's own existing in-flight-layout guard** (`GraphCanvas.svelte:264-375`) is the pattern to extend, not a different file's pattern. Current shape:
```ts
let layoutGeneration = 0;
let activeLayoutRun: any;

function runLayout(layoutRunStartedAt, fit, resizeAfter = false, survivorPositions?) {
	layoutGeneration += 1;
	const myGeneration = layoutGeneration;
	// Best-effort cleanup of the superseded run — NOT the correctness mechanism.
	if (activeLayoutRun && typeof activeLayoutRun.stop === 'function') {
		activeLayoutRun.stop();
	}
	opts.cy.one('layoutstop', () => {
		if (myGeneration !== layoutGeneration) return;   // <-- the existing token check
		...
	});
	activeLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit }).run();
}
```
**Teardown site to guard** (`GraphCanvas.svelte:895-899`, the mount `$effect`'s cleanup, verbatim):
```ts
return () => {
	resizeObserver?.disconnect();
	renderer = undefined;
	cy.destroy();
};
```
**Fix direction (per RESEARCH.md Investigation 1, D-06-compliant — our code only, no `pnpm patch`):** the existing `layoutGeneration`/`activeLayoutRun` token protects only the `layoutstop` callback GraphCanvas itself registers; it cannot protect cytoscape core's own internal `endBatch()` → `renderer.notify()` call, which fires from inside `layoutPositions()`'s `nodes.positions()` write, BEFORE `layoutstop` is even emitted. Since `cytoscape-elk`'s `Layout.stop()`/`destroy()` are no-ops against the pending elkjs promise, cancellation must happen on the GraphCanvas side: track an "ELK layout in flight" boolean (set true right before `.run()`, cleared when that generation's `layoutstop` fires), and in the cleanup above, gate the `cy.destroy()` call on it — do not call `cy.destroy()` while a layout for the current (or any not-yet-superseded) generation is still in flight; detach event handlers/DOM instead and let the pending promise resolve harmlessly against a still-technically-alive `cy`. Use `cy.destroyed()` (already exposed by cytoscape, the same predicate `notify()` itself checks) rather than inventing a parallel liveness tracker.

---

### `web/src/lib/components/graph/graph-style.ts` (modified) — FIX-05 (`text-valign: 'right'`)

**Bug site, verbatim** (`graph-style.ts:128-141`):
```ts
selector: 'node[?isSymbol]',
style: {
	shape: 'rectangle',
	width: 8,
	height: 8,
	'background-color': '#737373',
	label: 'data(label)',
	'font-size': 7,
	color: '#404040',
	'text-valign': 'right',   // INVALID — 'right' is not a text-valign value (top/center/bottom only)
	'text-halign': 'right',   // valid — this is the one that actually means "label to the right"
	'text-margin-x': 4
}
```
**Sibling selectors in the same file show the correct pairing** (`graph-style.ts:99-111`, `:83-90` region — verbatim valid usage to copy the shape of):
```ts
selector: 'node[!isDirectory]',
style: {
	...
	'text-valign': 'bottom',
	'text-margin-y': 4
}
```
Fix: drop the invalid `'text-valign': 'right'` line (or set it to a real value such as `'center'`) and keep `'text-halign': 'right'` plus `'text-margin-x': 4` — this is the exact selector the guava-scale "invalid endpoints" root-cause investigation (RESEARCH.md Investigation 2) does NOT explain (that's a separate, ELK-spacing mechanism) — this file's fix is purely the stylesheet typo D-08 names directly.

## Shared Patterns

### Test-seam convention (unexported field, no exported setter)
**Source:** `internal/daemon/daemon.go:97-137` (four existing examples: `onSync`, `onSyncStart`, `syncFn`, `onWatchOpen`)
**Apply to:** `getppid`/ticker injection (FIX-07/FIX-08) — do NOT introduce a new exported `Option` for this; `WithProbe` (`daemon.go:157-161`) is the one exception because it is also used by production code (`serve --mcp`).

### Error-message parity ("pid=N — stop it first, or run X")
**Source:** `internal/daemon/lock.go:208`
```go
return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
```
**Apply to:** `internal/cli/index.go`'s new FIX-06 refuse message — same shape, substituting `codegraph daemon stop` / `codegraph unlock` per the phase brief's own pointer.

### Category-error guard template (empty-vs-empty passes, non-empty mismatch refuses, never a wildcard)
**Source:** `internal/bench/regression.go:56-110` (three existing guards: GOOS/GOARCH, Runner, ScratchFS)
**Apply to:** the new `Repo` guard (FIX-09) — identical shape, no normalization logic (confirmed unnecessary by RESEARCH.md Investigation 5's write-site audit).

### Live-Chromium check script convention (`repoRoot()`, `pollUntil()`, diagnostic-JSON-on-every-exit-path, `pageerror` registered before navigation)
**Source:** `web/scripts/graph-expand-check.mjs`, `web/scripts/breadcrumb-check.mjs`, `web/scripts/graph-live-update-check.mjs`, `web/scripts/graph-collapse-affordance-check.mjs`, `web/scripts/live-push-multitab-check.mjs` — all five share this convention (see this repo's own `06-PATTERNS.md` for the original documentation of it).
**Apply to:** `graph-console-check.mjs` (FIX-04/FIX-05 gate) — genuinely new territory only for `page.on('console', ...)` capture (no existing sibling does this).

### CSP untouched (negative-space pattern)
**Source:** `internal/uiserver/spa.go:99` (`spaCSPBaseDirectives`), `internal/uiserver/spa_test.go:439` (`default-src 'self'` assertion)
**Apply to:** FIX-02's favicon fix — prove these two are byte-identical before/after, never widen `img-src`/add `data:`.

### Extract-workflow-shell-into-script-plus-Taskfile-target
**Source:** `scripts/inject-cosign-key.sh` + its two Taskfile call sites (`Taskfile.yml:2156`, `:2672`)
**Apply to:** FIX-10's shared delimiter-write script — the only existing precedent in this repo for turning inline workflow shell into a testable, Task-invocable standalone script.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `web/static/favicon.svg`, `web/static/favicon-32.png`, `web/static/apple-touch-icon.png` | config (static asset) | file-I/O | `web/static/` currently holds only `robots.txt`, a plain text file with no accompanying serve/build logic — there is no prior "binary static asset" precedent in this repo to copy code from; SvelteKit's adapter-static serves the directory automatically, so no new code is needed, only the files themselves plus the `+layout.svelte` link-tag edit (which does have a self-analog, documented above) |

## Metadata

**Analog search scope:** `web/scripts/`, `web/src/lib/components/graph/`, `web/src/routes/`, `web/static/`, `internal/cli/`, `internal/daemon/`, `internal/graphstore/`, `internal/bench/`, `internal/uiserver/`, `tools/graphcluster/`, `corpora/`, `scripts/`, `.github/workflows/`, `Taskfile.yml`
**Files scanned:** ~35 (read in full or via targeted offset/limit reads); 17 analog paths verified git-tracked via `git ls-files`
**Pattern extraction date:** 2026-09-14
