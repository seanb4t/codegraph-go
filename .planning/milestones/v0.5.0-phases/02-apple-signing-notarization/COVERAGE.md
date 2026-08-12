# Phase 2 — API Coverage Declaration

No external API integration: this phase consumes exactly one Apple notarization
submission endpoint, entirely in-process through GoReleaser's `notarize.macos`
pipe and its `github.com/goreleaser/quill` Go library — there is no capability
surface to enumerate, no client to build, and no opt-out decision to record. The
phase's other network calls (`gh release download`, `cosign verify-blob`,
`gh attestation verify`) are pre-existing verification machinery from Phase 1,
not new capability surface.

*Written at plan time, 2026-08-09, by `/gsd-plan-phase 2`.*
