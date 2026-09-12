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
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/query"
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

// schemeRE matches a well-formed URI scheme (RFC 3986 §3.1): a letter
// followed by any number of letters, digits, '+', '.' or '-'.
var schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

// ValidateEditorTemplate reports whether template is a well-formed,
// allowlisted editor URI template (D-13). nil means valid; a non-nil
// error's text is used verbatim as GetEditorLink's TEMPLATE_INVALID
// reason and as the CLI's startup-failure message (D-17) — ONE
// validator, exercised from both call sites, never a second
// implementation. Each distinct cause has its own message text; none
// shares wording with another (permalink.go:33-52's distinct-reason
// discipline).
func ValidateEditorTemplate(template string) error {
	if template == "" {
		return errors.New("template is empty")
	}
	if len(template) > EditorTemplateMaxBytes {
		return fmt.Errorf("template is %d bytes, which exceeds the %d-byte limit", len(template), EditorTemplateMaxBytes)
	}
	for i := 0; i < len(template); i++ {
		if b := template[i]; b < 0x21 || b == 0x7f {
			return errors.New("template contains whitespace or a control character")
		}
	}

	colon := strings.IndexByte(template, ':')
	if colon <= 0 {
		return errors.New("template has no scheme (expected a leading `scheme:`)")
	}
	scheme := template[:colon]
	if !schemeRE.MatchString(scheme) {
		return fmt.Errorf("template scheme %q is not well-formed", scheme)
	}
	if _, ok := editorURLSchemes[strings.ToLower(scheme)]; !ok {
		return fmt.Errorf("template scheme %q is not in the allowlist", scheme)
	}

	hasPath := false
	for i := 0; i < len(template); {
		switch template[i] {
		case '{':
			end := strings.IndexByte(template[i:], '}')
			if end == -1 {
				return errors.New("template has an unbalanced '{' with no matching '}'")
			}
			token := template[i+1 : i+end]
			if strings.ContainsAny(token, "{}") {
				return errors.New("template has an unbalanced brace")
			}
			if _, ok := editorPlaceholders[token]; !ok {
				return fmt.Errorf("template has an unknown placeholder {%s}", token)
			}
			if token == "path" {
				hasPath = true
			}
			i += end + 1
		case '}':
			return errors.New("template has an unbalanced '}' with no matching '{'")
		default:
			i++
		}
	}
	if !hasPath {
		return errors.New("template has no {path} placeholder")
	}
	return nil
}

// buildEditorURL substitutes template's {path}/{line}/{col} placeholders
// with absPath (percent-encoded per its position — before or after the
// template's first '?') and line/col (decimal). template is assumed
// already validated by ValidateEditorTemplate.
func buildEditorURL(template, absPath string, line, col int32) string {
	queryStart := strings.IndexByte(template, '?')

	var b strings.Builder
	for i := 0; i < len(template); {
		if template[i] != '{' {
			b.WriteByte(template[i])
			i++
			continue
		}
		end := strings.IndexByte(template[i:], '}')
		token := template[i+1 : i+end]
		switch token {
		case "path":
			inQuery := queryStart != -1 && i > queryStart
			b.WriteString(encodeEditorPathSegments(absPath, inQuery))
		case "line":
			b.WriteString(strconv.Itoa(int(line)))
		case "col":
			b.WriteString(strconv.Itoa(int(col)))
		}
		i += end + 1
	}
	return b.String()
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
func encodeEditorPathSegments(absPath string, inQuery bool) string {
	segments := strings.Split(absPath, "/")
	for i, seg := range segments {
		if inQuery {
			seg = url.QueryEscape(seg)
			seg = strings.ReplaceAll(seg, "+", "%20")
		} else {
			seg = url.PathEscape(seg)
			seg = strings.ReplaceAll(seg, ":", "%3A")
		}
		segments[i] = seg
	}
	return strings.Join(segments, "/")
}

// editorTemplateSourceToProto maps an EditorTemplateSource onto its wire
// projection — a named mapper, never an inline literal at the response-
// building call site (statusToProto's mapper discipline). default
// covers EditorTemplateNone and any future member, degrading to the
// honest NONE wording rather than a positive claim about a source that
// does not exist.
func editorTemplateSourceToProto(s EditorTemplateSource) uiv1.EditorTemplateSource {
	switch s {
	case EditorTemplateFlag:
		return uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_FLAG
	case EditorTemplateEnv:
		return uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_ENV
	case EditorTemplateDiscovered:
		return uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_DISCOVERED
	case EditorTemplateDisabled:
		return uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_DISABLED
	default: // EditorTemplateNone, and any future member
		return uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_NONE
	}
}

// editorLinkAnswer builds a GetEditorLinkResponse from the server's
// frozen options, an optional per-request override template, and the
// already-confined, already-joined absolute path (D-06/D-07). Extracted
// from GetEditorLink so its branches are independently unit-testable —
// the same rationale permalink.go's remotePresenceResponse extraction
// documents.
//
// default_source/default_editor and presets are populated on EVERY
// answer (D-07): the response always reports the SERVER DEFAULT's own
// provenance, whether or not this call's answer used an override.
func editorLinkAnswer(opts EditorLinkOptions, override *string, abs string, line, col int32) *uiv1.GetEditorLinkResponse {
	protoPresets := make([]*uiv1.EditorPreset, 0, 3)
	for _, p := range EditorPresets() {
		protoPresets = append(protoPresets, editorPresetToProto(p))
	}

	resp := &uiv1.GetEditorLinkResponse{
		DefaultSource: editorTemplateSourceToProto(opts.Source),
		DefaultEditor: opts.Editor,
		Presets:       protoPresets,
	}

	var effectiveTemplate string
	if override != nil && *override != "" {
		effectiveTemplate = *override
		resp.OverrideApplied = true
	} else {
		// D-16 first: an operator-disabled default is never silently
		// replaced by discovery or a stale value. Every other source
		// (None, Flag, Env, Discovered, and any future member) shares
		// the same "configured or not" check via this default arm —
		// the honest degrade permalink.go's own safe-default switch
		// documents.
		switch opts.Source {
		case EditorTemplateDisabled:
			resp.Availability = uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_NO_TEMPLATE
			resp.Reason = disabledByOperatorReason
			return resp
		default:
			if opts.Template == "" {
				resp.Availability = uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_NO_TEMPLATE
				resp.Reason = noTemplateConfiguredReason
				return resp
			}
			effectiveTemplate = opts.Template
		}
	}

	if err := ValidateEditorTemplate(effectiveTemplate); err != nil {
		resp.Availability = uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_TEMPLATE_INVALID
		resp.Reason = err.Error()
		return resp
	}

	resp.Availability = uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_BUILDABLE
	resp.Url = buildEditorURL(effectiveTemplate, abs, line, col)
	return resp
}

// GetEditorLink answers BRW-11/BRW-12 over the wire. line/col are
// validated BEFORE withEngine, exactly like permalink.go:100-111 — a
// present value below 1 refuses before the store is ever opened. path is
// confined by (*query.Engine).ValidateRepoRelativePath — the SAME gate
// GetNodeDetail, GetPermalink and FileSymbols already share (SRV-05);
// this handler adds no second confinement implementation. The one
// reclassification below (errors.Is(verr, fs.ErrNotExist)) mirrors
// GetPermalink's own since-deleted-file handling, built from the
// caller's own repo-relative path — never the absolute host path.
func (s *uiService) GetEditorLink(ctx context.Context, req *connect.Request[uiv1.GetEditorLinkRequest]) (*connect.Response[uiv1.GetEditorLinkResponse], error) {
	if line := req.Msg.Line; line != nil && *line < 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("line %d must be >= 1", *line))
	}
	if col := req.Msg.Col; col != nil && *col < 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("col %d must be >= 1", *col))
	}

	var resp *uiv1.GetEditorLinkResponse
	// classifiedErr carries the since-deleted-file reclassification
	// OUTSIDE withEngine's own error-mapping path — see permalink.go's
	// identical variable for the full rationale (IN-02: typed
	// *connect.Error rather than the broader `error`).
	var classifiedErr *connect.Error
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		path := req.Msg.GetPath()
		if verr := eng.ValidateRepoRelativePath(path); verr != nil {
			if errors.Is(verr, fs.ErrNotExist) {
				classifiedErr = connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
					"path %q does not exist in the working tree — it may have existed at the indexed commit and been deleted or renamed since; re-index or check the commit history",
					path,
				))
				return nil
			}
			return verr
		}

		root, err := filepath.Abs(s.repoPath)
		if err != nil {
			return err
		}
		// D-08: Join, never EvalSymlinks — editors key on the
		// workspace folder the user opened, and the resolved form was
		// only ever needed to PROVE confinement, above.
		abs := filepath.Join(root, filepath.Clean(path))

		line := int32(1)
		if req.Msg.Line != nil {
			line = *req.Msg.Line
		}
		col := int32(1)
		if req.Msg.Col != nil {
			col = *req.Msg.Col
		}

		resp = editorLinkAnswer(s.editorLink, req.Msg.Template, abs, line, col)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if classifiedErr != nil {
		return nil, classifiedErr
	}
	return connect.NewResponse(resp), nil
}
