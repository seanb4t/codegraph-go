# API Coverage — Phase 3 (Homebrew Tap & Cask)

No external API integration: this phase configures the existing release pipeline
(a `homebrew_casks:` block in `.goreleaser.yaml`, one credential-minting step in
`release.yml`) and adds one hidden CLI subcommand — the external systems it touches
(GitHub Actions, GitHub's repository API via `gh`, and Homebrew) are consumed through
a pinned first-party action, ad-hoc read/write calls, and a package manager's own
installer respectively, none of which is an API surface this project integrates
against or exposes to its users.

## Detector outcome (recorded, not assumed)

The `api-coverage` detector was run at plan time against both the ROADMAP Phase 3
section and the concatenated `03-CONTEXT.md` + `03-RESEARCH.md` scope. Both runs
returned `{"detected": false, "signals": []}`.

This declaration is written anyway, deliberately: the seal-time gate re-runs the
detector over a scope that includes the PLAN.md bodies, which necessarily contain
release-pipeline vocabulary (`gh api`, the Attestations API, "consume", "wire"). A
reasoned declaration is the sanctioned way to record a decided non-integration, and is
preferable to a matrix invented for a capability surface that does not exist.
