// Package web embeds this repository's committed SvelteKit build output
// (D-01, D-02, D-03) — the compiled app shell `codegraph ui` serves at
// `GET /` — so the running binary needs no JS toolchain and no separate
// asset directory on disk at runtime (BLD-02).
//
// This file MUST live inside web/, not under internal/uiserver/. Go's
// //go:embed patterns may not contain ".." path elements
// (golang/go#46056) and are resolved only relative to, or below, the
// directory containing the source file that carries the directive.
// internal/uiserver/ and web/build/ are SIBLING directories off the
// repository root — internal/uiserver/ can only reach web/build/ via
// ../../web/build, which is not a legal embed pattern — so the directive
// cannot live beside the SPA handler in internal/uiserver/spa.go, even
// though D-12 puts the handler itself there. This repository already hit
// and documented the identical constraint once, for claudeassets.go
// embedding .claude/ from the repository root (see claudeassets.go:7-17
// for the general form of this rule); web/ is simply the ancestor
// directory here instead of the repository root, because web/ is already
// an ancestor of web/build/ and internal/uiserver/ is not.
//
// The "all:" prefix on the embed pattern is mandatory (D-03): Go's
// default (non-"all:") embed walk silently excludes files and
// directories beginning with "." or "_", and SvelteKit's compiled output
// puts its entire hashed-asset tree under an underscore-prefixed
// directory (build/_app/). Without "all:", `go build` still succeeds —
// the omission produces no compile error — but every asset under
// build/_app/ silently vanishes from BuildFS. internal/uiserver/spa_test.go's
// TestEmbeddedFSMatchesOnDiskBuildTree exists specifically to catch this
// class of regression as a file-list diff, not merely "the build
// succeeded".
package web

import "embed"

// BuildFS is the embedded contents of web/build/, rooted at "build" (so
// every path inside it reads as e.g. "build/index.html", "build/_app/...").
// internal/uiserver/spa.go strips that one "build/" prefix exactly once
// via fs.Sub before serving.
//
//go:embed all:build
var BuildFS embed.FS
