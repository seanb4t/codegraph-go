package goextract

import (
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// ConditionalBlockKey returns the marker of the conditional-compilation
// block around decl, or "" when decl is not under one (design D16). The marker names the block's kind and start
// byte, so two declarations under different branches of one `#if` get
// different signature keys and form an overload set, and a call that
// reaches both is ambiguous. A PHP `if` whose condition mentions
// `function_exists` is such a block. The `#ifdef __cplusplus` block that
// only opens an `extern "C"` wrapper is not.
func ConditionalBlockKey(decl *tree_sitter.Node, src []byte) string {
	prev := decl
	for p := decl.Parent(); p != nil; prev, p = p, p.Parent() {
		switch p.Kind() {
		case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif", "preproc_elifdef":
			if prev.Kind() == "linkage_specification" && isCplusplusTest(p, src) {
				continue
			}
			return "#" + p.Kind() + "@" + strconv.FormatUint(uint64(p.StartByte()), 10)
		case "if_statement":
			if c := p.ChildByFieldName("condition"); c != nil && strings.Contains(c.Utf8Text(src), "function_exists") {
				return "#if@" + strconv.FormatUint(uint64(p.StartByte()), 10)
			}
		}
	}
	return ""
}

// isCplusplusTest reports whether block is `#ifdef __cplusplus` or an
// `#if` whose condition names __cplusplus.
func isCplusplusTest(block *tree_sitter.Node, src []byte) bool {
	switch block.Kind() {
	case "preproc_ifdef":
		name := block.ChildByFieldName("name")
		return name != nil && name.Utf8Text(src) == "__cplusplus"
	case "preproc_if":
		cond := block.ChildByFieldName("condition")
		return cond != nil && strings.Contains(cond.Utf8Text(src), "__cplusplus")
	}
	return false
}

// IsPreprocBlock reports whether n is a conditional-compilation block
// whose children a top-level walk must visit.
func IsPreprocBlock(n *tree_sitter.Node) bool {
	switch n.Kind() {
	case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif", "preproc_elifdef":
		return true
	}
	return false
}
