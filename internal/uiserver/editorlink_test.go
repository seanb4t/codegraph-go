package uiserver

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// int32Ptr is a small address-of helper for GetEditorLinkRequest's
// optional int32 fields (proto3 presence), mirroring the local
// `line := int32(N); &line` shape permalink_test.go already uses,
// collapsed into one helper since this file constructs many more of
// them.
func int32Ptr(v int32) *int32 { return &v }

// startedServerWith mirrors handlers_test.go's startedServer but passes
// the given Options to Listen, so a test can construct a server whose
// Options.EditorLink is a specific value under test — the shared helper
// signature this plan's read_first names rather than changing
// startedServer's own signature.
func startedServerWith(t *testing.T, opts Options) *Server {
	t.Helper()
	srv, err := Listen(opts)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	go func() { _ = srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))
	return srv
}

// TestGetEditorLinkPathConfinementAtRPCBoundary drives GetEditorLink
// through a real uiv1connect client against a real listener — never by
// calling the handler struct directly, because the point of this test is
// the boundary, not the function (mirroring
// TestGetPermalinkPathConfinementAtRPCBoundary and
// TestGetNodeDetailPathConfinementAtRPCBoundary).
//
// The positive control is asserted FIRST: a service broken for every
// input must fail there, not read as successful refusals below it.
func TestGetEditorLinkPathConfinementAtRPCBoundary(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServerWith(t, Options{
		RepoPath:   dir,
		EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag},
	})
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	t.Run("in-repo control", func(t *testing.T) {
		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{
			Path: "main.go",
			Line: int32Ptr(7),
			Col:  int32Ptr(3),
		}))
		if err != nil {
			t.Fatalf("GetEditorLink(in-repo control): %v", err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_BUILDABLE {
			t.Fatalf("availability = %v, want BUILDABLE", got)
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			t.Fatalf("filepath.Abs(%q): %v", dir, err)
		}
		wantURL := "vscode://file/" + encodeEditorPathSegments(filepath.Join(absDir, "main.go"), false) + ":7:3"
		if got := resp.Msg.GetUrl(); got != wantURL {
			t.Fatalf("url = %q, want %q", got, wantURL)
		}
		if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_FLAG {
			t.Fatalf("default_source = %v, want FLAG", got)
		}
		if got := len(resp.Msg.GetPresets()); got != 3 {
			t.Fatalf("len(presets) = %d, want 3", got)
		}
	})

	type refusalCase struct {
		name string
		path string
	}
	cases := []refusalCase{
		{name: "escape", path: "../outside.txt"},
		{name: "absolute", path: "/etc/passwd"},
		{name: "empty", path: ""},
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", dir, err)
	}
	resolvedAbsDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q): %v", absDir, err)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: c.path}))
			if err == nil {
				t.Fatalf("GetEditorLink(path=%q) succeeded, want a refusal", c.path)
			}
			if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
				t.Fatalf("GetEditorLink(path=%q): code = %v, want CodeInvalidArgument (err=%q)", c.path, code, err.Error())
			}
			if strings.Contains(err.Error(), absDir) || strings.Contains(err.Error(), resolvedAbsDir) {
				t.Fatalf("GetEditorLink(path=%q): error %q leaks the absolute host path", c.path, err.Error())
			}
		})
	}

	// The symlink case gets its own t.Run rather than joining the table
	// above: setupSymlinkEscape may call t.Skipf, and that must localize
	// to this one subtest, not abort the whole parent test function.
	t.Run("symlink-escape", func(t *testing.T) {
		relPath, ok := setupSymlinkEscape(t, dir)
		if !ok {
			return // setupSymlinkEscape already called t.Skipf or t.Fatalf
		}
		_, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: relPath}))
		if err == nil {
			t.Fatalf("GetEditorLink(path=%q) succeeded, want a refusal", relPath)
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("GetEditorLink(path=%q): code = %v, want CodeInvalidArgument (err=%q)", relPath, code, err.Error())
		}
		if strings.Contains(err.Error(), absDir) || strings.Contains(err.Error(), resolvedAbsDir) {
			t.Fatalf("GetEditorLink(path=%q): error %q leaks the absolute host path", relPath, err.Error())
		}
	})
}

// TestGetEditorLinkBuildsAbsoluteJoinedPathNotSymlinkResolved proves D-08:
// {path} is filepath.Join(filepath.Abs(RepoPath), rel), never its
// EvalSymlinks form. On macOS, t.TempDir() lives under /var/folders/...,
// itself a symlink to /private/var/folders/... — exactly the "different
// workspace to VS Code" case D-08's doc comment names. On a platform
// where the two already coincide (e.g. many Linux CI runners), the
// positive half still runs and the test logs that only it applied,
// rather than silently skipping (rule 84d1gfpywd).
func TestGetEditorLinkBuildsAbsoluteJoinedPathNotSymlinkResolved(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServerWith(t, Options{
		RepoPath:   dir,
		EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag},
	})
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", dir, err)
	}
	resolvedAbsDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q): %v", absDir, err)
	}

	resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{
		Path: "main.go",
		Line: int32Ptr(1),
		Col:  int32Ptr(1),
	}))
	if err != nil {
		t.Fatalf("GetEditorLink: %v", err)
	}
	url := resp.Msg.GetUrl()

	wantAbsSegment := encodeEditorPathSegments(filepath.Join(absDir, "main.go"), false)
	if !strings.Contains(url, wantAbsSegment) {
		t.Fatalf("url = %q, want it to contain the Abs-joined path %q", url, wantAbsSegment)
	}

	if resolvedAbsDir == absDir {
		t.Logf("filepath.Abs(dir) (%q) already equals its EvalSymlinks form on this platform — only the positive half of this test applies here", absDir)
		return
	}
	resolvedSegment := encodeEditorPathSegments(filepath.Join(resolvedAbsDir, "main.go"), false)
	if resolvedSegment != wantAbsSegment && strings.Contains(url, resolvedSegment) {
		t.Fatalf("url = %q, contains the SYMLINK-RESOLVED path %q — D-08 forbids resolving the joined path", url, resolvedSegment)
	}
}

// TestEditorTemplateSchemeAllowlist proves D-13: membership is a
// POSITIVE check, javascript/data/blob/file are refused by
// non-membership (never a denylist), and the allowlist's own size is
// asserted before any refusal is trusted (rule 84d1gfpywd).
func TestEditorTemplateSchemeAllowlist(t *testing.T) {
	if got := len(editorURLSchemes); got != 15 {
		t.Fatalf("len(editorURLSchemes) = %d, want exactly 15", got)
	}
	for _, p := range EditorPresets() {
		colon := strings.IndexByte(p.Template, ':')
		if colon < 0 {
			t.Fatalf("preset %q's template %q has no scheme", p.ID, p.Template)
		}
		scheme := strings.ToLower(p.Template[:colon])
		if _, ok := editorURLSchemes[scheme]; !ok {
			t.Fatalf("preset %q's scheme %q is not a member of editorURLSchemes", p.ID, scheme)
		}
	}

	validCases := []string{
		"vscode://file/{path}:{line}:{col}",
		"cursor://file/{path}:{line}",
		"idea://open?file={path}&line={line}",
		"vscode-insiders://file/{path}",
		"vscodium://file/{path}",
		"goland://open?file={path}",
		"http://localhost:63342/api/file/{path}:{line}",
		"https://x/{path}",
	}
	for _, tmpl := range validCases {
		if err := ValidateEditorTemplate(tmpl); err != nil {
			t.Fatalf("ValidateEditorTemplate(%q) = %v, want nil", tmpl, err)
		}
	}

	invalidSchemeCases := []struct {
		tmpl   string
		scheme string
	}{
		{"javascript:alert({path})", "javascript"},
		{"JAVASCRIPT:alert({path})", "javascript"},
		{"data:text/html,{path}", "data"},
		{"blob:http://x/{path}", "blob"},
		{"file:///{path}", "file"},
		{"ftp://x/{path}", "ftp"},
	}
	for _, c := range invalidSchemeCases {
		t.Run(c.scheme, func(t *testing.T) {
			err := ValidateEditorTemplate(c.tmpl)
			if err == nil {
				t.Fatalf("ValidateEditorTemplate(%q) = nil, want an error naming the scheme", c.tmpl)
			}
			if !strings.Contains(strings.ToLower(err.Error()), c.scheme) {
				t.Fatalf("ValidateEditorTemplate(%q) error = %q, want it to name the scheme %q", c.tmpl, err.Error(), c.scheme)
			}
		})
	}

	otherInvalidCases := []struct {
		name       string
		tmpl       string
		wantSubstr string
	}{
		{"no path placeholder", "vscode://file/{line}:{col}", "{path}"},
		{"unknown placeholder", "vscode://file/{foo}", "foo"},
		{"unbalanced brace", "vscode://file/{path", ""},
		{"space", "vscode://file/{path} extra", ""},
		{"control byte", "vscode://file/{path}\x01", ""},
		{"too long", "vscode://file/" + strings.Repeat("a", 2049) + "{path}", ""},
	}
	for _, c := range otherInvalidCases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateEditorTemplate(c.tmpl)
			if err == nil {
				t.Fatalf("ValidateEditorTemplate(%q) = nil, want an error", c.tmpl)
			}
			if c.wantSubstr != "" && !strings.Contains(err.Error(), c.wantSubstr) {
				t.Fatalf("ValidateEditorTemplate(%q) error = %q, want it to contain %q", c.tmpl, err.Error(), c.wantSubstr)
			}
		})
	}
}

// TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors proves D-07:
// NO_TEMPLATE (both causes) and TEMPLATE_INVALID are SUCCESSFUL
// responses with pairwise-distinct reason strings — never errors.
func TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	if noTemplateConfiguredReason == disabledByOperatorReason {
		t.Fatal("noTemplateConfiguredReason and disabledByOperatorReason must not share text")
	}

	t.Run("unconfigured", func(t *testing.T) {
		srv := startedServerWith(t, Options{RepoPath: dir})
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go"}))
		if err != nil {
			t.Fatalf("GetEditorLink: %v", err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_NO_TEMPLATE {
			t.Fatalf("availability = %v, want NO_TEMPLATE", got)
		}
		if got := resp.Msg.GetReason(); got != noTemplateConfiguredReason {
			t.Fatalf("reason = %q, want %q", got, noTemplateConfiguredReason)
		}
		if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_NONE {
			t.Fatalf("default_source = %v, want NONE", got)
		}
	})

	t.Run("disabled by operator", func(t *testing.T) {
		srv := startedServerWith(t, Options{RepoPath: dir, EditorLink: EditorLinkOptions{Source: EditorTemplateDisabled}})
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go"}))
		if err != nil {
			t.Fatalf("GetEditorLink: %v", err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_NO_TEMPLATE {
			t.Fatalf("availability = %v, want NO_TEMPLATE", got)
		}
		if got := resp.Msg.GetReason(); !strings.Contains(got, "disabled by operator") {
			t.Fatalf("reason = %q, want it to contain %q", got, "disabled by operator")
		}
		if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_DISABLED {
			t.Fatalf("default_source = %v, want DISABLED", got)
		}
	})

	t.Run("template invalid via override", func(t *testing.T) {
		srv := startedServerWith(t, Options{RepoPath: dir, EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag}})
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		override := "javascript:{path}"
		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go", Template: &override}))
		if err != nil {
			t.Fatalf("GetEditorLink: %v", err)
		}
		if got := resp.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_TEMPLATE_INVALID {
			t.Fatalf("availability = %v, want TEMPLATE_INVALID", got)
		}
		if !resp.Msg.GetOverrideApplied() {
			t.Fatal("override_applied = false, want true")
		}
		if got := resp.Msg.GetReason(); !strings.Contains(strings.ToLower(got), "javascript") {
			t.Fatalf("reason = %q, want it to name %q", got, "javascript")
		}
		if resp.Msg.GetReason() == noTemplateConfiguredReason || resp.Msg.GetReason() == disabledByOperatorReason {
			t.Fatalf("TEMPLATE_INVALID reason %q collides with a NO_TEMPLATE reason", resp.Msg.GetReason())
		}
	})
}

// TestGetEditorLinkOverrideRidesTheRequest proves D-06: a per-request
// template replaces the server default for that one call, but the
// response still reports the SERVER DEFAULT's own provenance.
func TestGetEditorLinkOverrideRidesTheRequest(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServerWith(t, Options{RepoPath: dir, EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag}})
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	override := "cursor://file/{path}:{line}"
	resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go", Template: &override}))
	if err != nil {
		t.Fatalf("GetEditorLink: %v", err)
	}
	if got := resp.Msg.GetUrl(); !strings.HasPrefix(got, "cursor://file/") {
		t.Fatalf("url = %q, want prefix %q", got, "cursor://file/")
	}
	if !resp.Msg.GetOverrideApplied() {
		t.Fatal("override_applied = false, want true")
	}
	if got := resp.Msg.GetDefaultSource(); got != uiv1.EditorTemplateSource_EDITOR_TEMPLATE_SOURCE_FLAG {
		t.Fatalf("default_source = %v, want FLAG (the server default's own provenance, unaffected by the override)", got)
	}
}

// TestGetEditorLinkLineAndColBounds proves the BRW-11 boundary edge:
// line 0 / col 0 / negative refuse before the store ever opens; an
// unset line/col substitutes 1; a line past the end of the file is
// BUILDABLE (GetEditorLink never reads the file).
func TestGetEditorLinkLineAndColBounds(t *testing.T) {
	t.Run("rejected before the store opens", func(t *testing.T) {
		dir := copyGofixture(t) // deliberately NOT indexed
		srv := startedServerWith(t, Options{RepoPath: dir, EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag}})
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		badCases := []struct {
			name string
			line *int32
			col  *int32
		}{
			{"line zero", int32Ptr(0), nil},
			{"col zero", nil, int32Ptr(0)},
			{"line negative", int32Ptr(-1), nil},
		}
		for _, c := range badCases {
			t.Run(c.name, func(t *testing.T) {
				_, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go", Line: c.line, Col: c.col}))
				if err == nil {
					t.Fatalf("GetEditorLink(%s) succeeded, want a refusal", c.name)
				}
				if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
					t.Fatalf("GetEditorLink(%s): code = %v, want CodeInvalidArgument (err=%q)", c.name, code, err.Error())
				}
				if !strings.Contains(err.Error(), "must be >= 1") {
					t.Fatalf("GetEditorLink(%s): error = %q, want it to name the offending value", c.name, err.Error())
				}
			})
		}
	})

	t.Run("unset defaults to 1 and past-EOF is buildable", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServerWith(t, Options{RepoPath: dir, EditorLink: EditorLinkOptions{Template: "vscode://file/{path}:{line}:{col}", Source: EditorTemplateFlag}})
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go"}))
		if err != nil {
			t.Fatalf("GetEditorLink: %v", err)
		}
		if !strings.HasSuffix(resp.Msg.GetUrl(), ":1:1") {
			t.Fatalf("url = %q, want suffix %q (unset line/col default to 1)", resp.Msg.GetUrl(), ":1:1")
		}

		resp2, err := client.GetEditorLink(context.Background(), connect.NewRequest(&uiv1.GetEditorLinkRequest{Path: "main.go", Line: int32Ptr(999999)}))
		if err != nil {
			t.Fatalf("GetEditorLink(line past EOF): %v", err)
		}
		if got := resp2.Msg.GetAvailability(); got != uiv1.EditorLinkAvailability_EDITOR_LINK_AVAILABILITY_BUILDABLE {
			t.Fatalf("availability = %v, want BUILDABLE (GetEditorLink never reads the file)", got)
		}
	})
}

// TestEditorLinkPathEncodingPerPosition proves D-08's per-position
// percent-encoding: {path} in a path position uses url.PathEscape per
// segment plus ':' rewritten to %3A; {path} after the template's first
// '?' uses url.QueryEscape per segment with '+' rewritten to %20;
// segments are rejoined with a literal '/' in both cases.
func TestEditorLinkPathEncodingPerPosition(t *testing.T) {
	const path = "/r/a b#c?d:e&f.go"

	pathURL := buildEditorURL("vscode://file/{path}:{line}", path, 1, 1)
	afterFile := strings.TrimPrefix(pathURL, "vscode://file/")
	for _, raw := range []string{" ", "#", "?"} {
		if strings.Contains(afterFile, raw) {
			t.Fatalf("buildEditorURL(path-position) = %q, contains raw %q after 'file/'", pathURL, raw)
		}
	}
	if !strings.Contains(pathURL, "%3A") {
		t.Fatalf("buildEditorURL(path-position) = %q, want ':' inside the path segment encoded as %%3A", pathURL)
	}
	if !strings.HasSuffix(pathURL, ":1") {
		t.Fatalf("buildEditorURL(path-position) = %q, want the trailing template ':1' kept literal", pathURL)
	}
	if !strings.Contains(pathURL, "/r/") {
		t.Fatalf("buildEditorURL(path-position) = %q, want '/' path separators kept literal", pathURL)
	}

	queryURL := buildEditorURL("idea://open?file={path}&line={line}", path, 1, 1)
	if !strings.Contains(queryURL, "/r/") {
		t.Fatalf("buildEditorURL(query-position) = %q, want '/' path separators kept literal", queryURL)
	}
	if strings.Contains(queryURL, "+") {
		t.Fatalf("buildEditorURL(query-position) = %q, must never encode space as '+'", queryURL)
	}
	if !strings.Contains(queryURL, "%20") {
		t.Fatalf("buildEditorURL(query-position) = %q, want the space encoded as %%20", queryURL)
	}
	// The filename's own '#', '?' and ':' bytes must not appear raw
	// inside the substituted {path} value — isolate that value by
	// slicing between "/r/" (its first literal segment) and the
	// template's own literal "&line=" separator, which follows it.
	pathValueStart := strings.Index(queryURL, "/r/")
	if pathValueStart < 0 {
		t.Fatalf("buildEditorURL(query-position) = %q, could not locate the substituted path value", queryURL)
	}
	substituted := strings.SplitN(queryURL[pathValueStart:], "&line=", 2)[0]
	for _, raw := range []string{"#", "?", ":"} {
		if strings.Contains(substituted, raw) {
			t.Fatalf("buildEditorURL(query-position) substituted path %q contains raw %q", substituted, raw)
		}
	}
}

// TestEditorPresetsAreExactlyThreeAndNameNoZed proves D-18/A4: exactly
// three presets, in the fixed order vscode/cursor/jetbrains, and Zed
// appears in none of their nine strings (id, name, template x 3).
func TestEditorPresetsAreExactlyThreeAndNameNoZed(t *testing.T) {
	presets := EditorPresets()
	if len(presets) != 3 {
		t.Fatalf("EditorPresets() has %d entries, want exactly 3", len(presets))
	}
	wantIDs := []string{"vscode", "cursor", "jetbrains"}
	for i, want := range wantIDs {
		if presets[i].ID != want {
			t.Fatalf("EditorPresets()[%d].ID = %q, want %q (order matters)", i, presets[i].ID, want)
		}
	}

	inspected := 0
	for _, p := range presets {
		for _, s := range []string{p.ID, p.Name, p.Template} {
			inspected++
			if strings.Contains(strings.ToLower(s), "zed") {
				t.Fatalf("preset %q contains %q, which mentions Zed — Zed is deliberately absent (D-18/A4)", p.ID, s)
			}
		}
	}
	if inspected != 9 {
		t.Fatalf("inspected %d preset strings, want exactly 9 (3 presets x 3 fields)", inspected)
	}
}
