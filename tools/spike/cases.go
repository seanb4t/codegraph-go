package main

import (
	"strings"

	"github.com/seanb4t/codegraph-go/internal/parser"
)

// malformedCase builds one deliberately adversarial input for the D-05
// crash-isolation dimension (RESEARCH Pitfall 5: crash isolation only
// manifests on malformed/adversarial input, not valid code). Every case
// here is a genuinely broken or pathological construct — never a real
// program from the benchmark corpus.
func malformedCase(name string) ([]byte, bool) {
	switch name {
	case "truncated_go":
		return truncatedSource(mustLoadGoCorpus(), 2), true
	case "truncated_python":
		return truncatedSource(mustLoadPythonCorpus(), 2), true
	case "garbage":
		return randomBytes(64 * 1024), true
	case "deep_nesting_go":
		return deepNestingGo(2_000_000), true
	case "deep_nesting_python":
		return deepNestingPython(2_800), true
	case "oversized":
		return oversizedSource(), true
	default:
		return nil, false
	}
}

// truncatedSource cuts a real corpus file off mid-stream at roughly the
// halfway point — likely mid-token/mid-string/mid-multi-byte-rune — which
// is a classic truncated-transfer / partial-read adversarial shape rather
// than syntactically-invalid-but-well-formed source.
func truncatedSource(files []corpusFile, idx int) []byte {
	if len(files) == 0 {
		return []byte{}
	}
	f := files[idx%len(files)]
	cut := len(f.Source) / 2
	if cut == 0 {
		cut = 1
	}
	return f.Source[:cut]
}

// deepNestingGo builds a pathologically deep parenthesis nest — the classic
// recursive-descent-parser stack-exhaustion shape — inside an otherwise
// syntactically-plausible Go expression.
func deepNestingGo(depth int) []byte {
	var b strings.Builder
	b.WriteString("package p\nfunc f() int {\n\treturn ")
	for i := 0; i < depth; i++ {
		b.WriteString("(")
	}
	b.WriteString("1")
	for i := 0; i < depth; i++ {
		b.WriteString(")")
	}
	b.WriteString("\n}\n")
	return []byte(b.String())
}

// deepNestingPython builds a pathologically deep indentation nest, which
// specifically targets tree-sitter Python's external INDENT/DEDENT C
// scanner (RESEARCH: the reason Python was chosen as the spike's
// external-scanner partner language for D-05's crash-isolation dimension).
// Indentation grows by a single space per level (not a realistic 4-space
// style) specifically to keep total byte size roughly O(depth^2 / 2) rather
// than O(depth^2 * 4) — deep enough to stress the scanner's internal
// indent stack across thousands of levels while staying under
// parser.MaxSourceBytes, so this case exercises the scanner itself rather
// than tripping the (separately-tested) size ceiling.
func deepNestingPython(depth int) []byte {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString(strings.Repeat(" ", i))
		b.WriteString("if True:\n")
	}
	b.WriteString(strings.Repeat(" ", depth))
	b.WriteString("pass\n")
	return []byte(b.String())
}

// oversizedSource builds an input one byte larger than parser.MaxSourceBytes
// to confirm BOTH arms reject it with parser.ErrSourceTooLarge before ever
// invoking the underlying C/WASM parser (Security Domain V5 / threat
// T-01-03) — the ceiling is meant to make this case a non-event for both
// backends, and that is exactly what this case verifies.
func oversizedSource() []byte {
	return []byte(strings.Repeat("a", parser.MaxSourceBytes+1))
}
