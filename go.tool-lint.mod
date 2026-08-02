// This module is isolated from go.tool.mod (task + goreleaser) because a
// single shared tool module does NOT compile: MVS resolves one version per
// module across the whole graph, and actionlint's expected YAML API loses
// the version bid to whatever task/goreleaser drag in transitively —
// verified live in a scratch copy of go.tool.mod with the actionlint tool
// directive added: action_metadata.go:273:22: te.Errors[0].Error undefined
// (type string has no field or method Error).
//   go.tool.mod       -> task, goreleaser  (237 net-new modules if merged
//                                            into root go.mod)
//   go.tool-lint.mod  -> actionlint         (16 modules alone; incompatible
//                                            if co-located above)
//
// GOWORK=off is REQUIRED on every invocation of this modfile — go.work
// workspace mode is incompatible with -modfile.
//
// Neither Dependabot nor Renovate manages this file (both gomod managers
// target go.mod only) — version bumps here are manual. See
// .planning/phases/10-local-build-contribution-and-taskfile-yml-setup/10-CONTEXT.md
// D-03/D-05.
module github.com/seanb4t/codegraph-go/tools-lint

go 1.26.5

tool github.com/rhysd/actionlint/cmd/actionlint

require (
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/rhysd/actionlint v1.7.12 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.3 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
