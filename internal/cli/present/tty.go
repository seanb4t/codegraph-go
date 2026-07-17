package present

// ChoosePresentation reports whether the pretty (lipgloss) branch should
// render, per D-04/D-05: isTTY must be true AND NO_COLOR must be
// unset/empty. It is pure and side-effect-free — it must NOT read
// os.Getenv or call term.IsTerminal itself. Real fd/env values are read
// only at the RunE call sites (internal/cli/status.go, files.go, init.go,
// index.go, sync.go — D-03), which makes this the one shared,
// unit-testable branch selector every call site wires identically.
func ChoosePresentation(isTTY bool, noColor string) bool {
	return isTTY && noColor == ""
}
