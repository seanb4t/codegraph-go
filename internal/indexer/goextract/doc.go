// Package goextract implements Pass 1's Go-specific tree-walk (LANG-01):
// mapping a parsed Go file's tree-sitter concrete syntax tree onto the
// codegraph node/edge vocabulary (D-06) — function/method/struct/
// interface/type_alias/constant/variable nodes, intra-file contains
// edges, and unresolved cross-file references (calls, imports, struct/
// interface embedding) for Pass 2 (resolve, a later plan) to settle.
//
// This package takes only primitives (a parser.Parser, plain strings and
// bytes) and returns plain data — it does not import internal/indexer, so
// internal/indexer can depend on it without an import cycle.
package goextract
