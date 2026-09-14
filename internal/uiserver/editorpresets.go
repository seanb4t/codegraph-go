package uiserver

import uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"

// EditorPreset is the Go-side twin of uiv1.EditorPreset (D-18): one of
// the three fixed editor choices GetEditorLink offers on every response
// so the browser's picker never holds a second copy of a template it
// constructs itself.
type EditorPreset struct {
	ID       string
	Name     string
	Template string
}

// EditorPresets returns a FRESH slice of exactly three presets, in this
// order: VS Code, Cursor, JetBrains (D-18). VS Code's template is
// confirmed against code.visualstudio.com's own command-line
// documentation (`code --goto {file}:{line}:{col}` maps onto the URI
// handler this template uses).
//
// The Cursor and JetBrains templates are [ASSUMED] (09-RESEARCH.md
// A1/A2) — neither vendor documents its URI scheme as a stable public
// contract at time of writing. They are corroborated by spatie/ignition,
// a shipped, widely-used error-page editor-link table (`editor_options`),
// whose two entries use the SAME absolute-path `{path}`/`{line}` shape
// D-08 produces — which is why the IDE-side `open?file=` form was chosen
// here over the JetBrains Toolbox `navigate/reference?project=…` form,
// which wants a project-relative path GetEditorLink does not have. Other
// JetBrains IDEs (GoLand, PyCharm, WebStorm, RubyMine, CLion, PhpStorm,
// Rider) each use their own URI scheme; plan 09-02's discovery selects
// the matching one automatically, and a custom template can name any of
// them by hand via the picker's custom field (D-18).
//
// Per 09-CONTEXT.md D-18/A4, exactly these three editors are presets —
// no fourth editor lacking a stable public URI scheme is named here as
// either a preset or a discovery target (enforced by a repo-wide check
// this plan's own tests run against this file's own text).
func EditorPresets() []EditorPreset {
	return []EditorPreset{
		{ID: "vscode", Name: "VS Code", Template: "vscode://file/{path}:{line}:{col}"},
		{ID: "cursor", Name: "Cursor", Template: "cursor://file/{path}:{line}"},
		{ID: "jetbrains", Name: "JetBrains (IntelliJ IDEA)", Template: "idea://open?file={path}&line={line}"},
	}
}

// editorPresetToProto maps one EditorPreset onto its wire projection —
// a named mapper, never an inline literal at the response-building call
// site, mirroring this package's existing mapper discipline
// (statusToProto, handlers.go).
func editorPresetToProto(p EditorPreset) *uiv1.EditorPreset {
	return &uiv1.EditorPreset{
		Id:       p.ID,
		Name:     p.Name,
		Template: p.Template,
	}
}
