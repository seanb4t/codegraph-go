# Phase 2: Apple Signing & Notarization - Pattern Map

**Mapped:** 2026-08-09
**Files analyzed:** 7 (config, tests, Taskfile, test harness, CI workflows x2, docs)
**Analogs found:** 7 / 7 (all as in-file/in-repo self-analogs — this phase mostly extends existing blocks rather than creating new file kinds)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.goreleaser.yaml` (`notarize:` new block; `binary_signs:` → `signs:` rename) | config | batch (build pipeline) | Same file's `sboms:` (233-278) and `archives:` (128-166) blocks | exact (in-file) |
| `internal/upgrade/goreleaser_shape_test.go` (retarget `TestBinarySignsSidecarMatchesUpgradeContract`; add analogous `signs:`-block test) | test | transform (config→resolved-template assertion) | `TestSbomsArePerBinaryWithSpdxNames` (579), `TestChecksumCoversRawAndZipIdsOnly` (338), `TestRawArchiveEntryStaysBinaryFormat` (281) — same file | exact |
| `Taskfile.yml` (new guarded notarize-rehearsal target; new `verify:gatekeeper`-style target) | config/utility | request-response (local shell invocation with preconditions) | `release:dry-run-signed` (488), `check:darwin-release-build` (270), `verify:release-assets` (924) | exact |
| `test/integration/main_test.go` `TestMain` (add `CODEGRAPH_TEST_BIN`/similar env override) | test | file-I/O (binary resolution seam) | `internal/upgrade/verify_release_e2e_test.go:92-105` (`e2eArtifactPaths`, `CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE`) | exact (directly transplantable) |
| `test/wireoracle/main_test.go` `TestMain` (same override, if in scope) | test | file-I/O | same as above; also `test/integration/main_test.go:39-57`'s own `TestMain` as the sibling to mirror | exact |
| `.github/workflows/post-release-verify.yml` (new gatekeeper-verification job + integration-suite-against-real-binary job) | route/workflow | event-driven (workflow_run) | `verify-supply-chain` (209-235) and `self-upgrade` (245-269) jobs, same file | exact |
| `.github/workflows/release.yml` (Apple secrets → env on the `release` job) | config | CRUD (secrets→env passthrough) | Same job's existing `env:`-free steps + `permissions:` block (85-100) | exact |
| `docs/RELEASE.md` (criterion-5 guarantee statement + `xattr`/`spctl` reproduction commands) | docs | — | Existing §1 verification structure (§a cosign, §b provenance, §c SBOM) | role-match |

## Pattern Assignments

### `.goreleaser.yaml` — `signs:` block (renamed from `binary_signs:`) + new `notarize:` block

**Analog:** same file, `sboms:` block (233-278) for comment-rationale style; `archives:` block (128-166) for "byte-frozen runtime contract" marking style.

**Load-bearing comment style to copy** (`archives:` id: raw, current lines ~140-148):
```yaml
    # UNCHANGED character-for-character (D-02/Finding 1): this is a
    # runtime contract with internal/upgrade.releaseAssetName(), pinned
    # by TestReleaseAssetNameMatchesGoReleaser
    # (verify_release_e2e_test.go) and TestRawArchiveEntryStaysBinaryFormat
    # (goreleaser_shape_test.go). `codegraph upgrade` downloads and swaps
    # in this exact byte stream with no extraction step — changing
    # `formats:` away from `binary`, or changing name_template by a
    # single character, bricks every user's next upgrade.
```
Apply the same "this is a runtime contract, pinned by TestX, breaking it bricks Y" phrasing to the new `signs:` block's `signature:` template and to `notarize:`'s `ids:` list (Pitfall 2's silent-skip risk).

**Current `binary_signs:` block being renamed** (`.goreleaser.yaml:223-231`, exact text to replace per D-18 LOCKED ruling):
```yaml
binary_signs:
  - cmd: cosign
    signature: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: binary
```

**Required replacement shape (D-18 diff, verbatim):**
```diff
-binary_signs:
-  - cmd: cosign
+signs:
+  - cmd: cosign
+    ids: [raw]
+    artifacts: binary
     signature: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"
     args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]
-    artifacts: binary
```
`signature:` template MUST stay byte-identical — line 275 of `sign.go` rebinds `env["artifact"]` to `art.Name` for the publish-naming pass, so the Path-vs-Name hazard documented in the `sboms:` comment block (below) still applies unchanged.

**`sboms:` comment block to mirror for the new `notarize:`/`signs:` rationale** (`.goreleaser.yaml:233-256`):
```yaml
# `documents:` is NAME-derived (`{{ .ArtifactName }}`), NOT PATH-derived
# (`${artifact}`) — this is cycle-3 review HIGH-B and is a correctness
# requirement, not a style choice. GoReleaser's sboms: pipe, with
# artifacts: binary, dedupes by PATH preferring the archives: pipe's
# renamed UploadableBinary artifact ...
# ... Held by TestSbomsArePerBinaryWithSpdxNames,
# which RESOLVES this template for all four platforms and asserts the
# results are four DISTINCT strings, not a literal string match.
sboms:
  - id: binary-sbom
    artifacts: binary
    cmd: syft
    args:
      - "$artifact"
      - "--output"
      - "spdx-json=$document"
    documents:
      - "{{ .ArtifactName }}.spdx.json"
```

**The now-FALSE comment that MUST be rewritten (Phase 1's D-14 rationale, A4 in RESEARCH.md confirms it is wrong):**
```yaml
# `binary_signs:` (not `signs:`) is used because it is format-independent
# — once `.zip` archives coexist with the `raw` binaries (REL-09), the
# `signs:` pipe's `artifacts: binary` mode (which requires
# `archives.formats: binary` project-wide) would no longer cleanly apply.
```
This comment sits directly above the `binary_signs:` block being replaced. It must be deleted/replaced, not left in place — leaving a known-false rationale in load-bearing config comments is explicitly flagged in RESEARCH.md D-18 as "the fifth instance of this repo's recurring 'following upstream docs faithfully was the trap' failure class."

**`notarize:` block to add — copy this shape (already vetted against pinned source, RESEARCH.md Pattern 1):**
```yaml
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'   # explicit — do not omit
      ids:
        - codegraph-darwin-amd64
        - codegraph-darwin-arm64
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
        # entitlements: intentionally OMITTED — D-03's working hypothesis
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"
        wait: true
        timeout: 20m
```
Position: `notarize:` runs inside `BuildPipeline` (before `archives:`/`checksum:`/`signs:`/`sboms:`), so no explicit reordering is needed in the YAML — D-04's ordering requirement is satisfied by moving cosign off `binary_signs:` (which also runs in `BuildPipeline`, pre-notarize) onto `signs:` (which runs post-`BuildPipeline`, i.e. after notarize). Do not attempt to reorder blocks in the YAML file itself — GoReleaser's pipe order is a hardcoded Go slice, not driven by block order (see RESEARCH.md Architecture Patterns Pattern 2).

---

### `internal/upgrade/goreleaser_shape_test.go` — retarget `TestBinarySignsSidecarMatchesUpgradeContract`

**Analog:** same file — `TestSbomsArePerBinaryWithSpdxNames` (579-632) is the closest sibling (already asserts a per-platform resolved-template property against a release-scoped block).

**Current test to retarget** (lines 507-568) — the type/parse plumbing to change (`goreleaserBinarySign`/`mustGoreleaserBinarySigns` → a `signs:`-block equivalent, e.g. `goreleaserSign`/`mustGoreleaserSigns`) plus the assertions:
```go
func TestBinarySignsSidecarMatchesUpgradeContract(t *testing.T) {
	data, err := os.ReadFile(goreleaserConfigPath)
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserConfigPath, err)
	}
	src := string(data)

	signs := mustGoreleaserBinarySigns(t, src)
	if len(signs) != 1 {
		t.Fatalf("binary_signs: has %d entries, want exactly 1", len(signs))
	}
	bs := signs[0]

	if bs.Cmd != "cosign" {
		t.Errorf("binary_signs[0].cmd = %q, want %q", bs.Cmd, "cosign")
	}
	if bs.Artifacts != "binary" {
		t.Errorf("binary_signs[0].artifacts = %q, want %q", bs.Artifacts, "binary")
	}
	wantArgs := []string{"sign-blob", "--bundle=${signature}", "${artifact}", "--yes"}
	// ... args equality loop ...

	seen := map[string]bool{}
	hostMatched := false
	for _, p := range releasePairs {
		got, err := resolveGoreleaserFieldTemplate(bs.Signature, map[string]any{
			"ProjectName": "codegraph",
			"Tag":         pinnedReleaseTag,
			"Os":          p.goos,
			"Arch":        p.goarch,
		})
		// ...
		want := goReleaserAssetName(pinnedReleaseTag, p.goos, p.goarch) + ".sigstore.json"
		if got != want {
			t.Errorf(...)
		}
		if seen[got] {
			t.Errorf("... signature resolved to a NON-DISTINCT name %q ...", got, p.goos, p.goarch)
		}
		seen[got] = true
		// host cross-check against releaseAssetName(tag) omitted here for brevity, keep verbatim
	}
	if !hostMatched {
		t.Fatalf(...)
	}
}
```
**MANDATORY pattern for the retarget** (RESEARCH.md D-18, "must move deliberately, not be deleted"): keep the exact same structure — parse the (now `signs:`) block, assert `cmd == "cosign"`, assert `ids: [raw]` (NEW field to assert — this is the D-18 addition), assert `artifacts == "binary"`, assert `args` equality, then RESOLVE `signature:` for all 4 `releasePairs` and assert (a) equals `goReleaserAssetName(...) + ".sigstore.json"`, (b) all 4 are DISTINCT (`seen` map), (c) the host pair matches `releaseAssetName(tag)`. **Never** collapse this to a literal-string match on the YAML — that is the exact anti-pattern memory `wewn0wp1n1` warns against.

**`TestSbomsArePerBinaryWithSpdxNames` (579-632)** — read in full for the resolve-and-assert-distinctness idiom to copy for the new `ids: [raw]` field check and the `documents:`-style resolution loop; same `seen[got]` distinctness-map pattern.

**`TestChecksumCoversRawAndZipIdsOnly` (338-358)** and **`TestRawArchiveEntryStaysBinaryFormat` (281-296)** — simpler analogs for any smaller new static-shape assertions this phase needs (e.g. asserting `notarize.macos[].ids` contains exactly the two darwin build ids, mirroring `checksum.ids`'s exact-set assertion via `sortedJoin`).

**`TestReleaseAssetNameMatchesGoReleaser`** (`internal/upgrade/verify_release_e2e_test.go:30-79`) — the release-matrix pinning pattern (`pairs := []struct{goos, goarch, want string}{...}` with independently-pinned literals, never derived from the functions under test) to reuse if a new asset-name-shape test is needed for notarize.

---

### `Taskfile.yml` — new guarded notarize-rehearsal target (D-08/D-09)

**Analog:** `release:dry-run-signed` (488-…) for the "hard-fail by name before any network round trip" precondition idiom AND for the temp-dir/trap/throwaway-credential rehearsal shape; `check:darwin-release-build` (270-309) for the `command -v cosign` precondition-after-hang lesson.

**Precondition idiom to copy exactly** (`release:dry-run-signed:499-507`):
```yaml
    preconditions:
      - sh: '[ "$(go env GOHOSTOS)" = "darwin" ]'
        msg: "release:dry-run-signed must run on a native darwin host — that is the runner class the migrated single-job release.yml (plan 01-03) uses (D-01). Run it on macOS locally, or dispatch the linux-cross-canary workflow, which runs this same target on namespace-profile-macos-6x14-tahoe."
      - sh: command -v zig
        msg: "zig not found. ..."
      - sh: command -v syft
        msg: "syft not found. ..."
      - sh: command -v cosign
        msg: "cosign not found. ..."
```
For the new notarize-rehearsal target, add preconditions that hard-fail BY NAME (not via a network round-trip) on:
```yaml
      - sh: '[ -n "${MACOS_SIGN_P12:-}" ]'
        msg: "MACOS_SIGN_P12 is not set. This target requires a local Developer ID Application certificate (base64 .p12) — see docs/RELEASE.md. Do not attempt this rehearsal without a real Apple Developer ID; it is maintainer-only (D-09)."
      - sh: '[ -n "${MACOS_NOTARY_KEY:-}" ]'
        msg: "MACOS_NOTARY_KEY is not set. This target requires an App Store Connect API key (base64 .p8) — see docs/RELEASE.md."
```
Mirror the exact same message shape: name the missing variable, name the doc that explains it, state why the target is gated (maintainer-only).

**The `check:darwin-release-build` gotcha to surface verbatim** (270-308 comment):
```
LOCAL-ONLY SINCE 2026-08-08 — this target is NO LONGER RUN IN CI, and it
now requires an interactive Sigstore session. ... `goreleaser build` has no
`--skip=sign` ... so this target now always reaches a cosign KEYLESS
signing call, which needs an OIDC token.
```
```yaml
    preconditions:
      - sh: '[ "$(go env GOHOSTOS)" = "darwin" ]'
        msg: "check:darwin-release-build must run on a native darwin host ..."
      - sh: command -v clang
        msg: "clang not found. Install Xcode or the Command Line Tools: xcode-select --install"
```
This is the precondition-check idiom that was added AFTER the 5-minute `expired_token` hang (`command -v cosign` in `release:dry-run-signed:506-507`) — apply the identical style: `sh: command -v <tool>` + a `msg:` that names the tool, the install command, and WHY it's needed, never a bare failure.

**CRITICAL Taskfile-templating gotcha to document inline in the new target** (already known, RESEARCH.md/CONTEXT.md D-09 reference `e2cnrbt6ph`): `task` renders every `cmds:` string through Go `text/template` BEFORE the shell sees it — so literal `{{ }}` in a Taskfile `cmds:` entry is eaten by `task`, not passed to the shell. `release:dry-run-signed`'s `cmds:` block (509 onward) avoids this entirely by using an `awk`/heredoc-free shell script with no `{{ }}` syntax anywhere — copy that avoidance strategy (plain shell, no Go-template-looking braces) rather than trying to escape braces inline.

**`verify:release-assets` (924-…)** — analog for the "$TAG/$REPO precondition + `gh`/`jq`/`cosign` tool preconditions + bounded-retry polling of `gh release view --json assets`" shape, reusable for any Taskfile target this phase adds that re-downloads published assets (e.g. a `verify:gatekeeper` target):
```yaml
    preconditions:
      - sh: '[ -n "${TAG:-}" ]'
        msg: "TAG is not set. ..."
      - sh: '[ -n "${REPO:-}" ]'
        msg: "REPO is not set. ..."
      - sh: '[ -n "${GH_TOKEN:-}" ]'
        msg: "GH_TOKEN is not set. ..."
      - sh: command -v gh
        msg: "gh CLI not found. ..."
      - sh: command -v jq
        msg: "jq not found. ..."
      - sh: command -v cosign
        msg: "cosign not found. ..."
```

---

### `test/integration/main_test.go` `TestMain` — add binary-override seam

**Analog (directly transplantable, per RESEARCH.md):** `internal/upgrade/verify_release_e2e_test.go:92-105`

**Precedent to copy verbatim in spirit:**
```go
// e2eArtifactPaths resolves the real signed artifact TestVerifyReleaseE2E
// needs, in priority order: (1) the committed testdata fixture pair, (2) the
// CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE env vars for a live-artifact run
// (e.g. against a real tag's release from CI). ok is false when neither
// source is present — the caller must t.Skip, never fail, in that case.
func e2eArtifactPaths() (binaryPath, bundlePath string, ok bool) {
	if fileExists(e2eFixtureBinaryPath) && fileExists(e2eFixtureBundlePath) {
		return e2eFixtureBinaryPath, e2eFixtureBundlePath, true
	}
	if envBinary, envBundle := os.Getenv("CODEGRAPH_E2E_BINARY"), os.Getenv("CODEGRAPH_E2E_BUNDLE"); envBinary != "" && envBundle != "" {
		return envBinary, envBundle, true
	}
	return "", "", false
}
```

**Current `TestMain` to modify** (`test/integration/main_test.go:39-57`, hardcodes the build):
```go
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "codegraph-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "integration: TestMain: go build github.com/seanb4t/codegraph-go/cmd/codegraph failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}
```
**Required edit (mirroring the RESEARCH.md Code Examples snippet):** check an env var (e.g. `CODEGRAPH_TEST_BIN`) FIRST, before the `MkdirTemp`/`go build` fallback path, and set `binPath` from it directly, skipping the build entirely:
```go
func TestMain(m *testing.M) {
	if envBin := os.Getenv("CODEGRAPH_TEST_BIN"); envBin != "" {
		binPath = envBin
		os.Exit(m.Run())
	}
	// ...existing MkdirTemp/go build fallback unchanged...
}
```
`test/wireoracle/main_test.go`'s `TestMain` should receive the identical edit if wireoracle is in scope for Criterion 4 — same env var name for consistency across both harnesses (planner's discretion whether both are in scope; CONTEXT.md only names `test/integration` explicitly).

---

### `.github/workflows/post-release-verify.yml` — new gatekeeper + integration-suite jobs

**Analog:** `verify-supply-chain` (209-235) and `self-upgrade` (245-269) jobs, same file — both are the established shape for "new job, needs: resolve-tag, event-aware guard, runs on a real runner, calls a Taskfile target with TAG/REPO/GH_TOKEN env."

**Event-aware guard — MUST copy verbatim onto every new job:**
```yaml
    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
```
This exact line appears at 103, 212, 248. The header comment (18-58) explains why: `github.event.workflow_run` is null on `workflow_dispatch`, so a bare `... .conclusion == 'success'` test (without the `event_name != 'workflow_run' ||` prefix) makes every job skip silently and report green (`h9348wvthq`). Any new job added for this phase's gatekeeper check or integration-suite run inherits this trap and MUST include the identical guard.

**`verify-supply-chain` job full shape to mirror** (209-235):
```yaml
  verify-supply-chain:
    name: verify supply-chain claims against the published release
    needs: resolve-tag
    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
    runs-on: namespace-profile-linux-amd64-4x8
    steps:
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
          cache: false
      - name: Install cosign
        uses: sigstore/cosign-installer@6f9f17788090df1f26f669e9d70d6ae9567deba6 # v4.1.2
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Verify published release assets
        env:
          TAG: ${{ needs.resolve-tag.outputs.tag }}
          REPO: ${{ github.repository }}
          GH_TOKEN: ${{ github.token }}
        run: task verify:release-assets
```
For the new gatekeeper job: same skeleton, `runs-on: namespace-profile-macos-6x14-tahoe` (needs real `spctl`/`syspolicy_check`/`xattr`, unlike the linux-hosted `verify-supply-chain`), no `needs: verify-supply-chain` — mirror `self-upgrade`'s deliberate independence comment (241-244: "Deliberately does NOT declare needs: verify-supply-chain: the two claims are independent evidence and both should report on a bad release, so a broken sidecar name and a broken self-upgrade are distinguishable rather than one masking the other").

**`self-upgrade` job's matrix shape** (245-269) — analog for parameterizing the gatekeeper job or integration-suite job over darwin/arm64 (and darwin/amd64 if both are checked):
```yaml
  self-upgrade:
    name: "self-upgrade proof (${{ matrix.goos }}/${{ matrix.goarch }})"
    needs: resolve-tag
    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
    strategy:
      fail-fast: false
      matrix:
        include:
          - goos: darwin
            goarch: arm64
            runner: namespace-profile-macos-6x14-tahoe
          - goos: linux
            goarch: amd64
            runner: namespace-profile-linux-amd64-4x8
    runs-on: ${{ matrix.runner }}
```

**Zero-asset-release tolerance:** required by D-11; `verify:release-assets`'s classification comment (Taskfile.yml:955-962) documents the discipline to replicate — never a fixed total, `ALLOWED_ASSETS` starts empty and additions are deliberate edits. Any new release-walking logic this phase adds (e.g. for the gatekeeper job to pick which asset to download) must tolerate `v0.5.0`'s permanently-zero-asset entry, matching `verify:self-upgrade`'s prior hardening (Taskfile.yml:1121, "hardened in Phase 1 to require both no-semver-suffix and prerelease: false, and to skip zero-asset releases" per CONTEXT.md's Reusable Assets note).

---

### `.github/workflows/release.yml` — Apple secrets wiring

**Analog:** the same `release` job's existing `permissions:` block (85-100) and steps for tool installation (`Install cosign` 131-135, `Install syft` 137-141).

**The single `id-token: write` job — DO NOT duplicate this permission elsewhere (D-14/D-17):**
```yaml
  release:
    name: build, sign, SBOM, and publish release
    runs-on: namespace-profile-macos-6x14-tahoe
    permissions:
      contents: write # publishing release assets
      id-token: write # cosign keyless OIDC signing (Finding 1) — this
      # job's OIDC token is what produces the cert SAN
      # releaseWorkflowRefPattern anchors on: this exact
      # workflow file, triggered by this exact tag ref. It is
      # the ONLY job in this file that can mint one (T-01-11, D-11:
      # exactly one job in this file may hold id-token: write —
      # machine-checked by TestOIDCWriteScopedToSingleGoreleaserJob,
      # no allowance).
      attestations: write # actions/attest-build-provenance (D-09)
```
Apple secrets belong in this SAME job's `env:` on the `Release` step (see Code Examples snippet in RESEARCH.md, already verified against quill's field bindings):
```yaml
      - name: Release
        id: release
        run: task release:goreleaser
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
          MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}
          MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
          MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
          MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
```
D-17 constraint: verify neither `darwin-toolchain-canary.yml` nor `linux-cross-canary.yml` (both `pull_request`-triggerable per the `check:darwin-release-build`/`release:dry-run-signed` comments referencing them) ever gains these `secrets.MACOS_*` references or `id-token: write`. Grep both canary workflow files for `secrets.MACOS_` and `id-token` as a verification step before considering this file's edit complete.

---

### `docs/RELEASE.md` — criterion-5 guarantee statement + reproduction commands

**Analog:** existing §1 structure (§a cosign, §b provenance, §c SBOM per CONTEXT.md's canonical_refs) and `docs/RELEASE-PROCEDURES.md` §7.1 "Preserved baselines" (~line 286) — the section added in Phase 1 for a similar "this fact must survive as documented state, not just as a passing test" need.

**Required content (exact three-part guarantee phrase, per CONTEXT.md Specific Ideas and D-16):**
```
notarized, online-verified, not stapled
```
Must name offline first launch as a KNOWN LIMITATION (not omit it) — this is the direct consequence of D-16 declining stapling.

**Reproduction snippet — copy this shape verbatim from RESEARCH.md's Code Examples** (already the canonical form for this phase):
```sh
curl -LO https://github.com/seanb4t/codegraph-go/releases/download/<tag>/codegraph_<tag>_darwin_arm64
xattr -w com.apple.quarantine "0081;$(printf '%x' "$(date +%s)");Safari;$(uuidgen)" codegraph_<tag>_darwin_arm64
xattr -p com.apple.quarantine codegraph_<tag>_darwin_arm64   # confirm BEFORE assessing
spctl -a -vv -t exec codegraph_<tag>_darwin_arm64
syspolicy_check distribution codegraph_<tag>_darwin_arm64
```
The `xattr -p` confirmation step BEFORE `spctl` is non-negotiable in the docs, not just the gate — CONTEXT.md's Specific Ideas section explicitly requires this ("A reader who runs spctl on a never-quarantined file gets a misleading pass").

## Shared Patterns

### "Assert the property, not the literal string" (test discipline)
**Source:** `internal/upgrade/goreleaser_shape_test.go` — every `TestXxxMatchesUpgradeContract`/`TestXxxArePerBinaryWith...` test resolves Go templates for all 4 release pairs and asserts (a) equality to an independently-derived expected name, (b) 4-way distinctness via a `seen` map.
**Apply to:** the retargeted `TestBinarySignsSidecarMatchesUpgradeContract`, any new static-shape test for `notarize:`'s `ids:`/`enabled:` fields, and any Taskfile-rehearsal-time assertion (`dist/artifacts.json` inspection) this phase adds.

### "Preconditions halt by name, not by timeout" (Taskfile discipline)
**Source:** `Taskfile.yml` — `release:dry-run-signed:499-507`, `check:darwin-release-build:302-308`, `check:darwin-toolchain:254-258`, `verify:release-assets:934-946`.
**Apply to:** every new Taskfile target this phase adds (notarize-rehearsal, gatekeeper-verify). Each missing tool/credential gets its own `sh:`/`msg:` pair naming the variable/tool, the fix command, and the reason.

### Event-aware `workflow_run` guard
**Source:** `.github/workflows/post-release-verify.yml:103,212,248` — `if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'`
**Apply to:** every new job added to `post-release-verify.yml` in this phase.

### Zero-asset-release / permanently-un-notarized-baseline tolerance
**Source:** `Taskfile.yml:955-962` (`verify:release-assets` classification comment) and the D-06/`v0.5.1`-baseline / D-07/`v0.5.0`-zero-asset carve-outs in CONTEXT.md.
**Apply to:** any release-walking logic in the new gatekeeper job or Taskfile target — must not choke on `v0.5.0` (zero assets) and must not accidentally target `v0.5.1` as anything other than the intentional RED baseline.

## No Analog Found

None — every file this phase touches has a strong in-repo analog (mostly in the same file being modified), reflecting that Phase 2 extends Phase 1's already-established GoReleaser/Taskfile/CI conventions rather than introducing a new architectural layer.

## Metadata

**Analog search scope:** `.goreleaser.yaml`, `internal/upgrade/*_test.go`, `Taskfile.yml`, `test/integration/main_test.go`, `test/wireoracle/main_test.go`, `.github/workflows/{release,post-release-verify}.yml`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`
**Files scanned:** 9
**Pattern extraction date:** 2026-08-09
