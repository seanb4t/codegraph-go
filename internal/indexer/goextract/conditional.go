package goextract

import (
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// ConditionalBlockKey returns the marker of the conditional-compilation
// block around decl, or "" when decl is not under one (design D16 of the
// index-invariants change). The marker names the block's kind and start
// byte, so two declarations under different branches of one `#if` get
// different signature keys and form an overload set, and a call that
// reaches both is ambiguous. A PHP `if` whose condition mentions
// `function_exists` is such a block.
func ConditionalBlockKey(decl *tree_sitter.Node, src []byte) string {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif", "preproc_elifdef":
			return "#" + p.Kind() + "@" + strconv.FormatUint(uint64(p.StartByte()), 10)
		case "if_statement":
			if c := p.ChildByFieldName("condition"); c != nil && strings.Contains(c.Utf8Text(src), "function_exists") {
				return "#if@" + strconv.FormatUint(uint64(p.StartByte()), 10)
			}
		}
	}
	return ""
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
