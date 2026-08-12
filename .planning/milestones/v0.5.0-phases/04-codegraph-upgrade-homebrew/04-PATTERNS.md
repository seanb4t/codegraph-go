# Phase 4: `codegraph upgrade` × Homebrew - Pattern Map

**Mapped:** 2026-08-11
**Files analyzed:** 7 (2 new, 5 modified)
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/upgrade/brew.go` (new) | service/utility (filesystem predicate) | request-response (pure function, no I/O side effects beyond stat) | `internal/upgrade/swap.go` (`checkWritable`) | exact — same package, same "early filesystem-probe guard" shape |
| `internal/upgrade/brew_test.go` (new) | test | CRUD/table-driven | `internal/upgrade/upgrade_test.go` | role-match — same package's test idiom, constructed-tree fixtures |
| `internal/upgrade/upgrade.go` (modified — new branch in `Run`) | service/controller (orchestrator) | request-response | itself (`Run`'s existing `checkWritable` call site) | exact — self-modification, precedent already in file |
| `internal/upgrade/upgrade_test.go` (modified — add D-08/D-09 assertions) | test | request-response, seam-based | itself (`TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading`, `TestUpgradeRun_TamperedDownloadNeverSwaps`) | exact |
| `.goreleaser.yaml` (modified — remove sentinel write/rm, fix stale comment) | config | event-driven (release-time hook) | itself (`hooks.post.install` / `hooks.post.uninstall` blocks) | exact — surgical removal within existing structure |
| `Taskfile.yml` (modified — remove sentinel assertions, add man-page baseline) | config/CI script | batch (release rehearsal script) | itself (`release:rehearse-cask` Step 5b and surrounding steps) | exact |
| `internal/upgrade/verify.go` (read-only reuse, folded todo 3 touches `Taskfile.yml`'s `verify:self-upgrade` target) | service (crypto verification) | request-response | itself (`releaseWorkflowRefPattern`) | exact — reused, not modified |

## Pattern Assignments

### `internal/upgrade/brew.go` (new file)

**Analog:** `internal/upgrade/swap.go` (`checkWritable`, lines 18-28) for the "early filesystem-probe, error-return" shape, plus `upgrade.go`'s `Run` (lines 82-121) for exactly where/how it gets called.

**Package/imports pattern** (from `swap.go` lines 1-7):
```go
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
)
```
`brew.go` will additionally need nothing beyond stdlib (`os`, `path/filepath`) per RESEARCH.md — no third-party dependency.

**Core predicate-function pattern** (analog: `checkWritable`, `swap.go:18-28`):
```go
func checkWritable(targetPath string) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".codegraph-upgrade-writable-check-*")
	if err != nil {
		return fmt.Errorf("upgrade: %s is not writable by the current user; refusing to upgrade (no changes made): %w", targetPath, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}
```
`brew.go`'s detection function follows the same shape: a single function taking `targetPath string`, doing filesystem probes (`filepath.EvalSymlinks`, segment-shape match, `os.Stat` on `INSTALL_RECEIPT.json`), returning either a descriptive result or an error/bool — never panicking, never touching the network (RESEARCH.md Pattern 3: fail-open to "not detected" on `EvalSymlinks` error, do not bubble a blocking error).

**Doc-comment convention** (every exported/unexported function in this package carries a "why", not just a "what" — see `checkWritable`'s and `atomicSwap`'s comments in `swap.go:9-17` and `swap.go:30-39`). Follow this density for `brew.go`'s detection function and any exported result type.

**Call-site integration pattern** (analog: `upgrade.go:117-121`, the `checkWritable` call):
```go
// D-13: refuse a non-writable target BEFORE downloading anything — no
// wasted download for an upgrade that can't be installed anyway.
if err := checkWritable(targetPath); err != nil {
	return err
}
```
D-11 requires the new brew-detection call to be inserted **above** `resolveLatest` (`upgrade.go:93`), not merely above `checkWritable`. The new branch must fork on `opts.Check` (matching the shape of the existing `if opts.Check { ... }` block at `upgrade.go:98-105`) rather than being a single unconditional error return, since the brew-managed path has two different outcomes (step-aside exit 0 vs. refusal exit non-zero) — see Error Handling below.

---

### `internal/upgrade/brew_test.go` (new file)

**Analog:** `internal/upgrade/upgrade_test.go` (whole-file idiom, table-driven where applicable), plus `TestUpgradeRun_TamperedDownloadNeverSwaps` (lines 54-60+) for `t.TempDir()`-seeded fixture construction.

**Imports pattern** (from `upgrade_test.go:1-10`):
```go
package upgrade

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)
```

**Fixture-construction pattern** (analog: `upgrade_test.go:54-60`):
```go
dir := t.TempDir()
target := filepath.Join(dir, "codegraph")
if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
	t.Fatalf("seed target: %v", err)
}
```
`brew_test.go`'s constructed-tree fixtures extend this idiom: build `t.TempDir()/Caskroom/<token>/<version>/codegraph` (or `Cellar/<name>/<version>/codegraph`), write `INSTALL_RECEIPT.json` at the matching ancestor, and — per Pitfall 1 in RESEARCH.md — model the real `bin/<name>` → `Caskroom/.../<name>` **symlink** (`os.Symlink`), not just a plain file, or the fixture will pass even with a `filepath.EvalSymlinks` bug present. Table-driven form covers all four prefixes (`/opt/homebrew`-shaped, `/usr/local`-shaped, custom, linuxbrew) plus the mandatory-executing false-positive case (CONTEXT.md's explicit constraint: not a comment).

**Seam-based assertion pattern** (analog: `upgrade_test.go:15-48`, full text):
```go
func TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading(t *testing.T) {
	var downloadCalled, verifyCalled, swapCalled bool

	var out bytes.Buffer
	opts := Options{
		Check: true,
		Out:   &out,
		resolveLatest: func(repoSlug string) (string, error) {
			return "v1.2.3", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			downloadCalled = true
			return nil, nil, nil
		},
		verify: func(binary, bundleJSON []byte) error {
			verifyCalled = true
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	if err := Run("v1.0.0", "/does/not/matter", opts); err != nil {
		t.Fatalf("Run(--check): %v", err)
	}
	if downloadCalled || verifyCalled || swapCalled {
		t.Fatalf("Run(--check) invoked download=%v verify=%v swap=%v, want all false", downloadCalled, verifyCalled, swapCalled)
	}
	if !strings.Contains(out.String(), "v1.2.3") {
		t.Errorf("Run(--check) output = %q, want it to mention v1.2.3", out.String())
	}
}
```
D-08's refusal proof (in `upgrade_test.go`, not `brew_test.go`, per the file-split below) is this exact shape: seed `targetPath` as a constructed Caskroom/Cellar tree with a real receipt, assert `Run` returns a non-nil error whose message contains `brew upgrade codegraph`, and assert `downloadCalled`/`swapCalled` are both still false. `brew_test.go` itself only needs to test the detection function directly (`TestDetectBrewManaged`, table-driven per RESEARCH.md's Validation Architecture) — it does not need `Run`'s func-var seams at all, since the detector is a pure filesystem predicate.

---

### `internal/upgrade/upgrade.go` (modified)

**Analog:** itself — the file already contains the exact structural precedent for this change.

**Insertion point** (`Run`, lines 82-96, current code before modification):
```go
func Run(currentVersion, targetPath string, opts Options) error {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}

	resolveLatest := opts.resolveLatest
	if resolveLatest == nil {
		resolveLatest = defaultResolveLatest
	}

	latest, err := resolveLatest(releaseRepoSlug)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	if opts.Check {
		...
```
D-11: the new brew-detection call must be inserted between the `out := opts.Out` block and the `resolveLatest := opts.resolveLatest` block (or immediately after, but strictly before the `resolveLatest(...)` call at line 93) — this is what gives both the refusal and the `--check` step-aside zero-network behavior.

**Error-handling / exit-code pattern to mirror** (D-05's precedent, `upgrade.go:119-121` — `checkWritable`'s call site returns the raw error unwrapped, letting the CLI layer's cobra `RunE` propagate it as non-zero exit): the new bare-`upgrade` refusal returns an error the same way. The `--check` step-aside (D-09/D-10) instead follows the `opts.Check` block's own pattern (`fmt.Fprintf(out, ...); return nil`, lines 98-104) — same function, same `out` writer, `return nil` for the zero-exit case.

**Doc-comment convention** (`Run`'s own doc comment, lines 75-81) — extend rather than replace; document the new early branch in the same doc-comment block describing `Run`'s sequence, matching the existing "resolve latest → (Check? report and return...) → refuse a non-writable targetPath..." narrative style.

---

### `internal/upgrade/upgrade_test.go` (modified — add tests)

**Analog:** itself, `TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading` (full excerpt above) and `TestUpgradeRun_TamperedDownloadNeverSwaps` fixture-seeding style (`upgrade_test.go:54-60`).

Two new tests belong here (not in `brew_test.go`, per RESEARCH.md's Wave-0 Gaps list): `TestUpgradeRun_RefusesBrewManaged` (D-08, asserts error + `brew upgrade codegraph` substring + `downloadCalled`/`swapCalled` both false) and `TestUpgradeRun_CheckBrewManaged` (D-09/D-10, asserts `nil` error, exit-0-equivalent, step-aside message, and all three seams still false). Both follow the `bytes.Buffer` + `strings.Contains` assertion idiom already used at lines 18, 45-47.

---

### `.goreleaser.yaml` (modified — sentinel removal)

**Analog:** itself — the sentinel write and its removal-counterpart already exist side by side; this is a subtraction within an established block, not a new pattern.

**Site 1 — stale comment to correct** (line 451, inside the block starting at line 424):
```
#     Phase 4 teaches it to refuse when brew-managed (D-08's sentinel).
```
D-02 falsifies "sentinel" — correct to describe structural detection (D-03), not delete the surrounding context about `auto_updates` being unsettable.

**Site 2 — the write to delete** (`hooks.post.install`, lines 570-588):
```ruby
# The Phase-4 sentinel (D-08). Located by RESOLVING #{binary}
# THROUGH SYMLINKS (Pathname#realpath) — never by matching a
# path prefix — so detection is correct under /opt/homebrew,
# /usr/local, a custom prefix, and linuxbrew alike. This is a
# cross-phase contract: internal/upgrade reads this file's
# location and format in Phase 4; changing either leaves
# already-installed users undetectable until they reinstall.
real_bin_dir = Pathname.new(binary).realpath.dirname
sentinel = real_bin_dir/".codegraph-brew-install"
sentinel.atomic_write(<<~SENTINEL)
  schema=1
  cask_token=#{token}
  cask_version=#{version}
  homebrew_prefix=#{HOMEBREW_PREFIX}
  man_dir=#{man_dir}
  man_page_count=#{man_pages.length}
  installed_at=#{Time.now.utc.iso8601}
SENTINEL
```
Delete this block wholesale. **BREW-05's gate (lines 542-568, the two `raise` assertions) is untouched** — do not touch anything above line 570.

**Site 3 — the removal to delete** (`hooks.post.uninstall`, lines 623-624):
```ruby
sentinel = Pathname.new(staged_path.to_s)/".codegraph-brew-install"
FileUtils.rm_f(sentinel)
```
Delete these two lines; the surrounding `man_dir`/`Dir.glob(...).each { rm_f }` lines (620-621) stay — they are folded-todo-2's target (man-page cleanup), not the sentinel.

---

### `Taskfile.yml` (modified — sentinel assertion removal + man-page baseline)

**Analog:** itself — `release:rehearse-cask` Step 5b (lines ~1995-2029) and the idempotency/post-uninstall checks that reference `SENTINEL_PATH`/`SENTINEL_VERDICT` downstream (lines 2072, 2091-2095, 2211-2215).

**Assertion pattern to remove** (Step 5b, lines 1995-2029 area):
```bash
SENTINEL_PATH="${REAL_BIN_DIR}/.codegraph-brew-install"
...
SENTINEL_VERDICT="absent"
if [ ! -f "${SENTINEL_PATH}" ]; then
  echo "::error::${SENTINEL_PATH} does not exist after brew install --cask — the post-install hook did not write the Phase-4 sentinel (D-08)"
  ...
else
  SENTINEL_VERDICT="present"
  SENTINEL_FIRST_LINE="$(head -n1 "${SENTINEL_PATH}")"
  if [ "${SENTINEL_FIRST_LINE}" != "schema=1" ]; then
    ...
  fi
  for key in ...; do
    if ! grep -q "^${key}=" "${SENTINEL_PATH}"; then
      ...
    fi
  done
  echo "sentinel contents:"
  cat "${SENTINEL_PATH}"
fi
```
Delete Step 5b in full. Downstream references at line 2072 (`[ ! -f "${SENTINEL_PATH}" ]`), lines 2091-2095 (post-uninstall symmetry check), and line 2215's `sentinel=${SENTINEL_VERDICT}` interpolation in the `CASK-REHEARSE-EVIDENCE` line must all be updated in the same change — either removed or replaced with the folded-todo-2 man-page pre-install-baseline check (`03-SECURITY.md:75`'s UF-5 fix), which reuses this same `if [ ! -f ... ]` / echo-error idiom for "man pages present before AND absent after a fresh install" rather than trusting a single post-install snapshot.

**Style precedent to carry forward:** every check in this block uses `echo "::error::..."` (GitHub Actions annotation format) followed by an explicit non-zero exit, never a bare `exit 1` with no message — keep this convention for the new man-page-baseline check and for whatever `SENTINEL_VERDICT`-shaped variable, if any, replaces the removed one in the evidence line.

**Note:** per RESEARCH.md's Assumption A3, `rg 'sentinel|codegraph-brew-install'` returned zero matches against `goreleaser_shape_test.go`/`taskfile_shape_test.go` at research time — confirmed again this session (see grep output below) — but re-run the same search immediately before editing, per the general "search before editing" rule.

---

### Folded todo 3 (`verify:self-upgrade`, `Taskfile.yml`)

**Analog:** `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` (referenced, not modified) and `internal/upgrade/upgrade.go:161-176` (`defaultVerify`, showing how the pattern is consumed today):
```go
func defaultVerify(binary, bundleJSON []byte) error {
	b, err := loadBundle(bundleJSON)
	if err != nil {
		return err
	}
	trustedMaterial, err := fetchTrustedRoot()
	if err != nil {
		return err
	}
	digest := sha256.Sum256(binary)
	return verifyRelease(b, trustedMaterial, "sha256", digest[:], releaseWorkflowRefPattern)
}
```
The `verify:self-upgrade` Taskfile target's fix must call the equivalent verification path (in-process, via this package's existing `verifyRelease`/`releaseWorkflowRefPattern`), not hand-roll a second cosign-SAN policy — reuse, per D-13 and the folded-todo's own planner note.

## Shared Patterns

### Early-refusal-before-mutation guard
**Source:** `internal/upgrade/swap.go:18-28` (`checkWritable`) and its call site `internal/upgrade/upgrade.go:119-121`
**Apply to:** `brew.go`'s detection function and its call site in `Run`
```go
if err := checkWritable(targetPath); err != nil {
	return err
}
```
Same idiom: probe first, return a descriptive `fmt.Errorf`-wrapped error, never mutate anything before the probe succeeds.

### Seam-based (not filesystem-side-effect) test proof
**Source:** `internal/upgrade/upgrade_test.go:15-48`
**Apply to:** all new `Run`-level tests (D-08's refusal proof, D-09's `--check` step-aside proof)
Inject fakes for `resolveLatest`/`download`/`verify`/`swap` via `Options`' unexported func fields; assert booleans recording invocation, never assert on real filesystem hashes/mtimes (explicitly rejected by D-08's maintainer ruling).

### Injectable func-var CLI seam
**Source:** `internal/cli/upgrade.go:17` (`var upgradeRunFunc = upgrade.Run`)
**Apply to:** no new CLI-layer change is needed for this phase (D-12 places everything in `internal/upgrade`), but any `--help`/exit-code documentation work (D-10) touches `newUpgradeCmd()` in this same file — its existing `Long`/`Example` string-building style (lines 33-37) is the pattern for adding the two-exit-behavior note.

### GitHub Actions annotation + explicit exit on Taskfile assertions
**Source:** `Taskfile.yml` `release:rehearse-cask` (lines ~2013, 2019, 2024, 2073, 2092)
**Apply to:** the new man-page pre-install-baseline check (folded todo 2) and any replacement for the removed sentinel evidence field
```bash
echo "::error::<description of what failed and why>"
exit 1
```

## No Analog Found

None — every file in scope for this phase is either a new file inside `internal/upgrade` (which already contains a directly-applicable analog for both the production code and the test idiom) or a surgical edit within an existing, already-read file.

## Metadata

**Analog search scope:** `internal/upgrade/` (all `.go` and `_test.go` files), `internal/cli/upgrade.go`, `.goreleaser.yaml` (homebrew_casks block), `Taskfile.yml` (`release:rehearse-cask` target)
**Files scanned:** `internal/upgrade/upgrade.go`, `internal/upgrade/swap.go`, `internal/upgrade/upgrade_test.go` (partial — first 60 lines + grep confirmation of remaining tests' presence), `internal/cli/upgrade.go`, `.goreleaser.yaml` lines 440-635, `Taskfile.yml` sentinel references (grep), `goreleaser_shape_test.go`/`taskfile_shape_test.go` (grep only — zero sentinel matches, confirming RESEARCH.md's Assumption A3)
**Pattern extraction date:** 2026-08-11
