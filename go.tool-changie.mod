// This module is isolated from go.tool.mod (task, goreleaser, govulncheck),
// go.tool-lint.mod (actionlint), go.tool-proto.mod (buf, protoc-gen-go,
// protoc-gen-connect-go) and go.tool-golangci.mod (golangci-lint) for the
// same reason those four are isolated from each other and from the root
// go.mod: MVS resolves one version per module across the whole graph, and
// go.tool-lint.mod's own header already records a LIVE-VERIFIED collision
// from co-locating a new tool with an existing one.
//
// Standalone, changie's own isolated graph resolves 64 modules (measured
// live: `GOWORK=off go list -m -modfile=go.tool-changie.mod all | wc -l`),
// dominated by Cobra plus a Charm/Bubbletea prompt library and
// Masterminds/sprig templating changie pulls in for its interactive mode —
// smaller than task/goreleaser's graph, larger than actionlint's 16.
//
// A SECOND, unplanned measurement was also taken (01-RESEARCH.md, D-04's
// own escape-hatch text: "permitted only if the executor measures it
// live... and records the measurement"): co-locating changie's tool
// directive directly into go.tool.mod cost only +4 net-new modules
// (1002 -> 1006) with ZERO lost MVS bids — goreleaser/v2 stayed pinned at
// v2.17.1 and sigstore/cosign/v3 stayed at v3.1.1, and all three tools
// (task, goreleaser, changie) built successfully from the co-located
// modfile. This is a materially different result from go.tool-proto.mod's
// documented buf-vs-goreleaser silent-downgrade experience. D-04 still
// locks THIS separate-modfile path as the phase's default deliverable
// regardless of that measurement; it is recorded here so a future
// executor revisiting the co-location escape hatch does not have to
// re-derive it.
//
//   go.tool.mod         -> task, goreleaser, govulncheck
//   go.tool-lint.mod     -> actionlint            (incompatible if
//                                                   co-located with
//                                                   go.tool.mod)
//   go.tool-proto.mod    -> buf, protoc-gen-go,    (compiles if co-located
//                            protoc-gen-connect-go  with either of the
//                                                   above, but silently
//                                                   downgrades goreleaser/
//                                                   cosign — see its own
//                                                   header)
//   go.tool-golangci.mod -> golangci-lint          (large, opinionated
//                                                   dependency tree kept
//                                                   isolated from all
//                                                   three)
//   go.tool-changie.mod  -> changie                (this file — 64 modules
//                                                   alone; +4/zero-lost-bids
//                                                   if ever co-located with
//                                                   go.tool.mod, per the
//                                                   measurement above)
//
// GOWORK=off is REQUIRED on every invocation of this modfile — go.work
// workspace mode is incompatible with -modfile.
//
// Neither Dependabot nor Renovate manages this file (both gomod managers
// target go.mod only) — version bumps here are manual.
//
// Contributors and CI run the identical command body through
// `task changie -- <args>` (CHG-01's recorded install path, D-05) — see
// Taskfile.yml's `changie` target. See
// .planning/phases/01-changie-baseline/01-CONTEXT.md D-04/D-05.
module github.com/seanb4t/codegraph-go/tools-changie

go 1.26.5

tool github.com/miniscruff/changie

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/charmbracelet/bubbles v0.16.1 // indirect
	github.com/charmbracelet/bubbletea v0.24.2 // indirect
	github.com/charmbracelet/lipgloss v0.9.1 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/cqroot/multichoose v0.1.1 // indirect
	github.com/cqroot/prompt v0.9.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.14.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/miniscruff/changie v1.26.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20211018074035-2e021307bc4b // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
