---
status: complete
phase: 03-homebrew-tap-cask
source:
  - 03-01-SUMMARY.md
  - 03-02-SUMMARY.md
  - 03-03-SUMMARY.md
  - 03-04-SUMMARY.md
  - 03-05-SUMMARY.md
started: 2026-08-10
updated: 2026-08-10
---

## Current Test

[testing complete]

## Tests

<!-- 03-01: man command + minimal cask block -->

### 1. Hidden `codegraph man <dir>` command generating the full man-page tree
expected: Hidden Cobra command registered on newRootCmd(), creates its target directory including missing parents, and returns a non-nil error naming the offending path on write failure. 30 pages measured.
result: pass
source: automated
coverage_id: D1
covering_tests: go build ./...; go run ./cmd/codegraph man <dir> (30 files); go test ./internal/cli/... -run TestManCmd -v (5/5)

### 2. Minimal `homebrew_casks:` block in .goreleaser.yaml
expected: ids [zip], no url: override, hooks.post.install invoking codegraph man.
result: pass
source: automated
coverage_id: D2
covering_tests: task check:goreleaser; task release:dry-run; task release:dry-run-signed

### 3. release:rehearse-cask end-to-end proof
expected: A real `brew install --cask` of the GoReleaser-rendered cask, post-install hook executing the installed binary, codegraph.1 landing on disk.
result: pass
source: automated
coverage_id: D3
covering_tests: op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask (real Developer ID credentials; 30 man pages; diff exactly one line)

<!-- 03-02: the install gate, sentinel, completions -->

### 4. hooks.post.install carries two positive assertions
expected: Man-page count > 1 and version equality (v-stripped exact match, never containment) — both raise and roll back a real brew install on failure. BREW-05's amended gate.
result: pass
source: automated
coverage_id: D1
covering_tests: release:rehearse-cask (real signed+notarized binaries); goreleaser_shape_test.go#TestHomebrewCaskHooksHaveStructuralProperties

### 5. Version-mismatch perturbation proves the assertion fires
expected: brew install exits non-zero with a message quoting BOTH the real reported version and the mutated declared version; restored and re-ran clean.
result: pass
source: automated
coverage_id: D2
covering_tests: rehearse-cask Step 7 — observed exit 1, message quoting 0.7.0 vs the mutated value; restore observed exit 0

### 6. Phase-4 sentinel written beside the symlink-resolved binary
expected: `.codegraph-brew-install` present with all 6 keys after install, absent after uninstall. Located via Pathname#realpath, never a path-prefix match.
result: pass
source: automated
coverage_id: D3
covering_tests: release:rehearse-cask — 6/6 keys present after install, absent after uninstall

### 7. Shell completions generated from the installed binary
expected: bash, zsh and fish completion files all present and non-empty after install, generated at install time from the binary (BREW-03).
result: pass
source: automated
coverage_id: D4
covering_tests: release:rehearse-cask (completion_count=3); goreleaser_shape_test.go#TestHomebrewCaskGeneratedCompletionsShellsIsExactSet

### 8. Completions cannot substitute for the install gate
expected: A deliberately wrong completions executable name produces exit 0 with only a warning — measured, not assumed (RESEARCH.md Pitfall 1).
result: pass
source: automated
coverage_id: D5
covering_tests: rehearse-cask Step 7b — observed exit 0, "Failed to generate ... exited with 127"

### 9. Seven shape tests hold the cask block's non-obvious fields
expected: Each demonstrated RED against a real, confirmed-applied, byte-cleanly-reverted mutation.
result: pass
source: automated
coverage_id: D6
covering_tests: go test ./internal/upgrade/... -run 'TestHomebrewCask|TestParseGoreleaserCask|TestParseGoreleaserArchivesRaw' -v; task test:unit

<!-- 03-03: legacy mode — no coverage: block, so these are human checkpoints -->

### 10. Tap-scoped GitHub App — configuration and boundary
expected: A SEPARATE GitHub App (id 4549710) from release-please. Only permission is contents:write (plus mandatory metadata:read), webhook inactive (events: []), installed on exactly one repository — seanb4t/homebrew-tap. Proven differentially with one minted token: codegraph-go/collaborators -> 403, homebrew-tap/collaborators -> 200. This is ROADMAP criterion 5.
result: pass

### 11. Token mint placement inside the OIDC-bearing release job
expected: A real measurement (run 31417685002) showed GitHub Actions REDACTS a secret-masked job output across a job boundary — the consumer received an empty string. So the mint moved INSIDE the release job rather than into a separate one. This widens the OIDC-bearing job's surface by one Action, pinned to the same SHA release-please.yml already carries, with inputs scoped to step level. The tradeoff is recorded in-comment at release.yml:164-184.
result: pass

### 12. A release refuses to proceed on a missing or collapsed tap credential
expected: Two release:goreleaser preconditions. Missing token -> halt. Token byte-identical to GITHUB_TOKEN -> halt. The `-n` form fails closed on unset AND empty, which matters because goreleaser's client.NewIfToken falls back SILENTLY to the release token when the variable is set-but-empty. Both demonstrated failing in isolation.
result: pass

<!-- 03-04: RED demonstrations + credential boundary -->

### 13. Install gate demonstrated RED for a binary that cannot execute
expected: Hand-corrupted real signed+notarized zip; exit 1 with "terminated by uncaught signal KILL" inside assertion one; post-failure check shows no codegraph remained.
result: pass
source: automated
coverage_id: D1
covering_tests: brew install --cask local-rehearsal/cask-mutation1/codegraph — confirmed-applied via sha256 pairs

### 14. Install gate demonstrated RED from the BINARY side
expected: A second real signed+notarized build under a temporary local tag reports the wrong version; exit 1 quoting both versions. Drives the mismatch from the binary rather than the cask, which 03-02 already covered.
result: pass
source: automated
coverage_id: D2
covering_tests: brew install --cask local-rehearsal/cask-mutation2/codegraph

### 15. Credential boundary with a positive control
expected: A real, REVERTED write to homebrew-tap first (201 create, 204 delete), then the same token refused a write to codegraph-go with a resource-scope 403 — not a credential-validity 401. A refusal without a positive control would not have been evidence.
result: pass
source: automated
coverage_id: D3
covering_tests: POST homebrew-tap/git/refs -> 201, DELETE -> 204, repo confirmed restored; POST codegraph-go/git/refs -> 403

### 16. BREW-05 and ROADMAP criterion 3 amended after the proof
expected: Amendment names hooks.post.install's two assertions instead of a cask `test:` block Homebrew Casks do not have. Only criterion 3's line changed; criterion 4 byte-unchanged; no heading altered; phase filter identical before and after.
result: pass
source: automated
coverage_id: D4
covering_tests: git diff .planning/ROADMAP.md; getMilestonePhaseFilter() identical before/after

<!-- 03-05: publication and cold install -->

### 17. BREW-06 half one written as an ARGUMENT, never as evidence
expected: Structural argument with file:line citations against pinned GoReleaser v2.17.1 source, labelled throughout as having no executed evidence, with no claim word (demonstrated/proven/verified/tested) applied to it.
result: pass
source: automated
coverage_id: D1
covering_tests: awk-scoped rg over 03-EVIDENCE.md's half-one section — only the section's own meta-statement and a negated sentence match

### 18. Cold install against the published v0.8.0 release
expected: brew tap resolves, brew install completes (after closing over Homebrew's untrusted-tap refusal), the binary reports v0.8.0 and indexes a real fixture, and bash/zsh/fish completion each confirmed as a SEPARATE verdict via genuine interactive shells.
result: pass
source: automated
coverage_id: D2
covering_tests: brew tap && brew trust --tap && brew install on a torn-down machine; codegraph init/status (files=1 nodes=3 edges=3); tmux-driven Tab completion in all three shells

### 19. Tap publication written once, by the App alone
expected: Casks/codegraph.rb has exactly one commit, authored by goreleaserbot (not a human), and its declared sha256 values match the real downloaded v0.8.0 assets byte-for-byte.
result: pass
source: automated
coverage_id: D3
covering_tests: gh api repos/seanb4t/homebrew-tap/commits?path=Casks/codegraph.rb -> 1 commit, author goreleaserbot; sha256 cross-check exact

### 20. BREW-06 half two as executed evidence
expected: post-release-verify.yml green on all 7 jobs against v0.8.0; verify:release-assets PASS against re-downloaded assets; v0.7.0 and v0.8.0 asset lists identical in shape (17 entries each), no duplication or orphaning.
result: pass
source: automated
coverage_id: D4
covering_tests: gh run view 31424108520 conclusion=success 7/7; "PASS — checksums, cosign, and attestation all verified"; gh release view both --json assets length 17

### 21. ROADMAP criteria 1 and 4 amended, REQUIREMENTS marked complete
expected: Criteria 1 and 4 amended dated 2026-08-10, BREW-01/02/06 marked complete, milestone phase filter unchanged before and after.
result: pass
source: automated
coverage_id: D5
covering_tests: git diff shows no heading altered; getMilestonePhaseFilter() identical before/after

## Summary

total: 21
passed: 21
issues: 0
pending: 0
skipped: 0

Of the 21: 18 auto-passed deterministically from structured `coverage:` blocks
(03-01, 03-02, 03-04, 03-05), 3 were human checkpoints derived from
03-03-SUMMARY.md, which is `mode: legacy` — it carries no `coverage:` block at
all, so its deliverables could not be auto-classified and fell through to
prose extraction. All three confirmed by the maintainer.

## Gaps

[none yet]
