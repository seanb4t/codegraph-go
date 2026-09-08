// This module is isolated from go.tool.mod (task + goreleaser + govulncheck)
// and go.tool-lint.mod (actionlint) because co-locating buf in either one
// forces an unwanted downgrade — measured live in a scratch copy of
// go.tool.mod with buf's tool directives added and `go get` run to pin
// buf v1.72.0, protoc-gen-go v1.36.11 and protoc-gen-connect-go v1.20.0:
// all six resulting tool binaries (task, goreleaser, govulncheck, buf,
// protoc-gen-go, protoc-gen-connect-go) DID compile — this is not the
// "X undefined" hard failure go.tool-lint.mod's header documents for
// actionlint — but MVS's single-version-per-module resolution silently
// downgraded github.com/goreleaser/goreleaser/v2 from the intentionally
// pinned v2.17.1 to v2.17.0-dcf5a6b3-nightly (an unreleased nightly
// build) and github.com/sigstore/cosign/v3 from v3.1.1 to v3.0.6, both
// losing their version bid to buf's own dependency on an older
// github.com/google/go-github/v88. Silently swapping the release
// toolchain's pinned goreleaser for a nightly prerelease is a
// correctness regression for an unrelated concern (release tooling), not
// something a "just measure whether it compiles" check alone would
// catch — so this repo's own version-bid-loss precedent (go.tool-lint.mod)
// is treated as the operative standard even though the failure mode here
// is a silent downgrade rather than a build error. A third isolated
// modfile keeps buf's large, opinionated dependency tree (protovalidate,
// its own YAML handling, etc.) from ever perturbing task/goreleaser's or
// actionlint's resolved versions again.
//
//   go.tool.mod       -> task, goreleaser, govulncheck
//   go.tool-lint.mod  -> actionlint          (incompatible if co-located
//                                              with go.tool.mod)
//   go.tool-proto.mod -> buf, protoc-gen-go,  (compiles if co-located with
//                         protoc-gen-connect-go  either of the above, but
//                                              silently downgrades
//                                              goreleaser/cosign — see
//                                              above)
//
// GOWORK=off is REQUIRED on every invocation of this modfile — go.work
// workspace mode is incompatible with -modfile.
//
// Neither Dependabot nor Renovate manages this file (both gomod managers
// target go.mod only) — version bumps here are manual. See
// .planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md
// and 01-01-PLAN.md Task 2's MVS-measurement instructions.
module github.com/seanb4t/codegraph-go/tools-proto

go 1.26.5

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	github.com/bufbuild/buf/cmd/buf
	google.golang.org/protobuf/cmd/protoc-gen-go
)

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.36.11-20260626152828-968bf0468096.1 // indirect
	buf.build/gen/go/bufbuild/protodescriptor/protocolbuffers/go v1.36.11-20250109164928-1da0de137947.1 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1 // indirect
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.20.0-20260713175918-10d915f5b43b.1 // indirect
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.36.11-20260713175918-10d915f5b43b.1 // indirect
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.36.11-20241007202033-cf42259fcbfc.1 // indirect
	buf.build/go/app v0.2.1-0.20260626143626-be153867abea // indirect
	buf.build/go/bufplugin v0.10.0 // indirect
	buf.build/go/bufprivateusage v0.1.0 // indirect
	buf.build/go/interrupt v1.1.0 // indirect
	buf.build/go/protovalidate v1.2.0 // indirect
	buf.build/go/protoyaml v0.7.0 // indirect
	buf.build/go/spdx v0.2.0 // indirect
	buf.build/go/standard v0.1.1-0.20260325175353-2b287e071df5 // indirect
	cel.dev/expr v0.25.2 // indirect
	connectrpc.com/connect v1.20.0 // indirect
	connectrpc.com/otelconnect v0.9.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/bufbuild/buf v1.72.0 // indirect
	github.com/bufbuild/protocompile v0.14.2-0.20260716165721-bb5762d29672 // indirect
	github.com/bufbuild/protoplugin v0.0.0-20260414125817-25d1d281b46b // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.6.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.8 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/google/cel-go v0.29.2 // indirect
	github.com/google/go-containerregistry v0.21.7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jdx/go-netrc v1.0.0 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/moby/api v1.55.0 // indirect
	github.com/moby/moby/client v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/petermattis/goid v0.0.0-20260716134002-a9b348f0a2b9 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.60.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	github.com/tidwall/btree v1.8.1 // indirect
	go.lsp.dev/jsonrpc2 v0.10.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/protocol v0.12.0 // indirect
	go.lsp.dev/uri v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20260709172345-9ea1abe57597 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260715232425-e75dac1f907d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260715232425-e75dac1f907d // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
	pluginrpc.com/pluginrpc v0.5.0 // indirect
)
