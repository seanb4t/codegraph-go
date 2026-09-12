// editorlink.go implements GetEditorLink (D-05/D-06/D-07/D-08): it turns
// a repo-relative path plus an optional line/col into an editor URI
// built from a {path}/{line}/{col} template.
//
// Every configuration state this handler answers with is an ANSWER,
// never an error: no template configured, a template disabled by the
// operator, and a template that fails validation all render as a
// SUCCESSFUL response carrying EDITOR_LINK_AVAILABILITY_NO_TEMPLATE or
// EDITOR_LINK_AVAILABILITY_TEMPLATE_INVALID with a populated reason —
// the same "empty is a successful response" discipline GetPermalink
// already follows (D-07). The ONE case that IS an error is a rejected
// path argument, handled by (*query.Engine).ValidateRepoRelativePath
// (SRV-05) below.
//
// The absolute path substituted for {path} is filepath.Join(repo root,
// relative path) — computed AFTER ValidateRepoRelativePath has accepted
// the relative path (which already applies both the string-level and
// the EvalSymlinks confinement gate) — and is NEVER the symlink-resolved
// form: editors key on the workspace folder the user opened, and
// /tmp/x and /private/tmp/x are different workspaces to VS Code (D-08).
// Paths are POSIX only; native Windows support was dropped at v0.4.0
// (WSL2 only, STATE.md), so under WSL2 the POSIX rules below apply.
package uiserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// EditorTemplateSource records where a server's EFFECTIVE DEFAULT editor
// template came from (D-14/D-07). The zero value, EditorTemplateNone,
// means no source configured a template at all — distinct from
// EditorTemplateDisabled, which means the operator explicitly turned
// editor links off (D-16).
type EditorTemplateSource int

const (
	// EditorTemplateNone is the zero value: no flag, env var, or
	// discovery ever configured a template.
	EditorTemplateNone EditorTemplateSource = iota
	// EditorTemplateFlag means `--editor-url` set the template.
	EditorTemplateFlag
	// EditorTemplateEnv means CODEGRAPH_EDITOR_URL set the template.
	EditorTemplateEnv
	// EditorTemplateDiscovered means startup-time editor discovery
	// (plan 09-02) found an installed editor and selected its preset.
	EditorTemplateDiscovered
	// EditorTemplateDisabled means the operator explicitly disabled
	// editor links (`--no-editor-url` or CODEGRAPH_NO_EDITOR_URL,
	// D-16) — discovery is skipped entirely in this state.
	EditorTemplateDisabled
)

// EditorLinkOptions is the server's frozen-at-startup editor-link
// configuration (D-14/D-15/D-16), written ONLY by internal/cli's startup
// resolver (flag, env, discovery) and copied into uiService at Listen —
// never re-read per request, the same startup-frozen discipline the
// publisher and repoPath already follow.
type EditorLinkOptions struct {
	// Template is the effective default editor URI template. Empty
	// when Source is EditorTemplateNone or EditorTemplateDisabled.
	Template string
	// Source records where Template came from (or that none was
	// configured / editor links are disabled).
	Source EditorTemplateSource
	// Editor is the discovered launcher/preset id (e.g. "code",
	// "cursor", "idea") when Source is EditorTemplateDiscovered, and
	// empty otherwise.
	Editor string
}

// EditorTemplateMaxBytes bounds an editor URI template's length (T-09-10,
// denial of service via a pathological template). Both the CLI's
// startup resolver and this handler's per-request validation enforce
// the SAME limit through the SAME validator.
const EditorTemplateMaxBytes = 2048

// editorURLSchemes is D-13's POSITIVE scheme allowlist: exactly the
// schemes the three shipped presets and their siblings use, plus the two
// web-IDE schemes (http/https, for JetBrains' localhost REST endpoint
// and any browser-based editor). Membership is checked on the parsed,
// lower-cased scheme — javascript, data, blob and file are refused by
// NON-membership, never enumerated in a denylist (rule 84d1gfpywd: a
// denylist is a negative-only guard that a new dangerous scheme could
// silently slip past).
var editorURLSchemes = map[string]struct{}{
	"vscode":          {},
	"vscode-insiders": {},
	"vscodium":        {},
	"cursor":          {},
	"idea":            {},
	"phpstorm":        {},
	"pycharm":         {},
	"webstorm":        {},
	"goland":          {},
	"clion":           {},
	"rider":           {},
	"rubymine":        {},
	"jetbrains":       {},
	"http":            {},
	"https":           {},
}

// editorPlaceholders is the set of placeholder names ValidateEditorTemplate
// accepts inside a template's `{...}` tokens. Any other token name is
// TEMPLATE_INVALID, naming the unknown placeholder.
var editorPlaceholders = map[string]struct{}{
	"path": {},
	"line": {},
	"col":  {},
}

// noTemplateConfiguredReason is GetEditorLink's NO_TEMPLATE reason when
// no source configured a template at all (EditorTemplateNone). Names
// all three routes an operator or user has to configure one (D-14).
const noTemplateConfiguredReason = "no editor template is configured — set --editor-url, set CODEGRAPH_EDITOR_URL, or choose a preset in the browser"

// disabledByOperatorReason is GetEditorLink's NO_TEMPLATE reason when
// the operator explicitly disabled editor links (D-16). The exact
// phrase "disabled by operator" is a stable, tested substring.
const disabledByOperatorReason = "editor links are disabled by operator (--no-editor-url or CODEGRAPH_NO_EDITOR_URL)"

// ValidateEditorTemplate reports whether template is a well-formed,
// allowlisted editor URI template (D-13). nil means valid; a non-nil
// error's text is used verbatim as GetEditorLink's TEMPLATE_INVALID
// reason and as the CLI's startup-failure message (D-17) — ONE
// validator, exercised from both call sites, never a second
// implementation.
//
// STUB (RED phase, plan 09-01 Task 1): always returns
// connect.NewError(CodeUnimplemented, ...) so
// TestEditorTemplateSchemeAllowlist and friends fail on their own
// assertions rather than on a missing symbol. Replaced with the real
// validator in the GREEN commit.
func ValidateEditorTemplate(template string) error {
	return errStubNotYetImplemented
}

// buildEditorURL substitutes template's {path}/{line}/{col} placeholders
// with absPath (percent-encoded per its position — before or after the
// template's first '?') and line/col (decimal). template is assumed
// already validated by ValidateEditorTemplate.
//
// STUB (RED phase): returns an empty string.
func buildEditorURL(template, absPath string, line, col int32) string {
	return ""
}

// encodeEditorPathSegments percent-encodes each "/"-separated segment of
// absPath independently and rejoins with a literal "/" (the same
// split-escape-rejoin shape percentEncodeRepoPath (permalink.go) uses,
// extended for position): inQuery false applies url.PathEscape per
// segment then rewrites ':' to '%3A' (PathEscape's pchar allowance
// leaves ':' unescaped, which a URI path position does not permit
// unambiguously); inQuery true applies url.QueryEscape per segment then
// rewrites '+' to '%20' (QueryEscape's default space encoding, which an
// editor's file-path query parameter must not receive literally).
//
// STUB (RED phase): returns absPath unchanged.
func encodeEditorPathSegments(absPath string, inQuery bool) string {
	return absPath
}

// editorLinkAnswer builds a GetEditorLinkResponse from the server's
// frozen options, an optional per-request override template, and the
// already-confined, already-joined absolute path (D-06/D-07). Extracted
// from GetEditorLink so its branches are independently unit-testable —
// the same rationale permalink.go's remotePresenceResponse extraction
// documents.
//
// STUB (RED phase): returns a zero-value response.
func editorLinkAnswer(opts EditorLinkOptions, override *string, abs string, line, col int32) *uiv1.GetEditorLinkResponse {
	return &uiv1.GetEditorLinkResponse{}
}

// errStubNotYetImplemented marks a RED-phase declaration whose real body
// lands in this plan's GREEN commit (test(09-01) -> feat(09-01)).
var errStubNotYetImplemented = errors.New("uiserver: GetEditorLink is not yet implemented (plan 09-01 RED phase)")

// GetEditorLink answers BRW-11/BRW-12 over the wire.
//
// STUB (RED phase): always returns connect.CodeUnimplemented so the
// suite compiles and every named test fails on its own assertion
// against a real, connectable server rather than on a missing symbol.
// Replaced with the real handler body in this plan's GREEN commit.
func (s *uiService) GetEditorLink(ctx context.Context, req *connect.Request[uiv1.GetEditorLinkRequest]) (*connect.Response[uiv1.GetEditorLinkResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errStubNotYetImplemented)
}
