package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/fsatomic"
)

// recordFile appends fr to result.Files, or — if err is non-nil — wraps
// err with path context and appends it to result.Errors instead. This is
// the single write-outcome funnel every AgentTarget.Install/Uninstall call
// site uses so a returned I/O error can never be silently swallowed
// (CR-01): every one of the ~40 `if fr, err := ...; err == nil { append }`
// sites this package used to have is now `recordFile(&result, path, fr, err)`.
func recordFile(result *WriteResult, path string, fr FileResult, err error) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
		return
	}
	result.Files = append(result.Files, fr)
}

// recordAction is recordFile's analog for helpers (removeMarkedSection,
// stripTOMLTable-style callers) that report a bare FileAction rather than
// a full FileResult.
func recordAction(result *WriteResult, path string, action FileAction, err error) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
		return
	}
	result.Files = append(result.Files, FileResult{Path: path, Action: action})
}

// readJSONFile parses path as a JSON object and returns it as a generic
// map. A missing file, an empty file, or unparseable content all fall
// back to an empty map rather than erroring — every agent config this
// package edits is a file it does not own, written by user-editable
// external tools, so a corrupt/partial config on disk must never panic
// or block install/uninstall (V5, T-06-01). A genuine I/O error other
// than "file does not exist" (e.g. a permission failure) is surfaced to
// the caller alongside the empty-map fallback.
func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return map[string]any{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		// Malformed existing config — defensive empty-map fallback per
		// V5; the caller proceeds as if the file were empty rather than
		// failing the whole install/uninstall run.
		return map[string]any{}, nil
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// writeJSONFile marshals data as 2-space-indented JSON with a trailing
// newline (matching the shape every agent's own config tooling produces)
// and writes it atomically (V12).
func writeJSONFile(path string, data map[string]any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, string(out)+"\n")
}

// normalizeJSON round-trips v through JSON marshal/unmarshal so entries
// built from concrete Go types (structs, []string, map[string]string)
// compare correctly against entries readJSONFile already decoded into
// generic map[string]any/[]any/float64/string/bool/nil — jsonDeepEqual
// only needs to reason about one type shape.
func normalizeJSON(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// jsonDeepEqual compares two already-JSON-decoded values for equality,
// independent of map key iteration order (Go maps have none) — used to
// decide whether writeMcpEntry needs to write at all (D-07 idempotency).
func jsonDeepEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvv, ok := bv[k]
			if !ok || !jsonDeepEqual(v, bvv) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonDeepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// writeMcpEntry reads path's existing JSON, sets mcpServers.codegraph to
// buildEntry()'s (normalized) result, and writes back only if it
// differs from what's already there — every sibling key under both the
// top level and mcpServers is preserved untouched (D-07). Reports
// created/updated/unchanged accordingly.
func writeMcpEntry(path string, buildEntry func() any) (FileResult, error) {
	existing, err := readJSONFile(path)
	if err != nil {
		return FileResult{}, err
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	before := mcpServers["codegraph"]

	after, err := normalizeJSON(buildEntry())
	if err != nil {
		return FileResult{}, err
	}

	if jsonDeepEqual(before, after) {
		return FileResult{Path: path, Action: ActionUnchanged}, nil
	}

	action := ActionCreated
	if before != nil {
		action = ActionUpdated
	}
	mcpServers["codegraph"] = after
	existing["mcpServers"] = mcpServers
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: action}, nil
}

// removeMcpEntry deletes mcpServers.codegraph from path's JSON, if
// present. If that empties mcpServers, the now-empty mcpServers object
// is removed too rather than left as clutter (D-07 keep-clean). Never
// errors when there's nothing to remove — reports ActionNotFound.
func removeMcpEntry(path string) (FileResult, error) {
	existing, err := readJSONFile(path)
	if err != nil {
		return FileResult{}, err
	}
	mcpServers, ok := existing["mcpServers"].(map[string]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	if _, present := mcpServers["codegraph"]; !present {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	delete(mcpServers, "codegraph")
	if len(mcpServers) == 0 {
		delete(existing, "mcpServers")
	} else {
		existing["mcpServers"] = mcpServers
	}
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

// replaceOrAppendMarkedSection upserts a marker-fenced body (body must
// already include startMarker/endMarker) into filePath:
//   - missing file: create it, body only
//   - markers present, identical body: no-op (ActionUnchanged, byte-for-
//     byte idempotent re-run, D-07)
//   - markers present, different body: replace only the marked span,
//     everything outside it preserved verbatim (T-06-02)
//   - no markers present: append after existing content with a
//     separating blank line (or no separator if the file was empty)
func replaceOrAppendMarkedSection(filePath, body, startMarker, endMarker string) (FileAction, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := atomicWriteFile(filePath, body+"\n"); err != nil {
			return "", err
		}
		return ActionCreated, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	s := string(content)
	startIdx := strings.Index(s, startMarker)
	endIdx := strings.Index(s, endMarker)

	if startIdx != -1 && endIdx > startIdx {
		existingBlock := s[startIdx : endIdx+len(endMarker)]
		if existingBlock == body {
			return ActionUnchanged, nil
		}
		newContent := s[:startIdx] + body + s[endIdx+len(endMarker):]
		if err := atomicWriteFile(filePath, newContent); err != nil {
			return "", err
		}
		return ActionUpdated, nil
	}

	// No markers — append, preserving existing content.
	trimmed := strings.TrimRight(s, "\n \t")
	sep := ""
	if len(trimmed) > 0 {
		sep = "\n\n"
	}
	if err := atomicWriteFile(filePath, trimmed+sep+body+"\n"); err != nil {
		return "", err
	}
	return ActionUpdated, nil
}

// removeMarkedSection removes only the marker-fenced span from filePath
// (plus the blank-line separator replaceOrAppendMarkedSection's append
// path introduces before it), restoring the file to its exact pre-insert
// bytes — the inverse of replaceOrAppendMarkedSection (T-06-02). A file
// with no markers is left completely untouched (ActionKept); a missing
// file reports ActionNotFound. If removing the span empties the file
// entirely, the file itself is deleted (mirrors "the file never existed
// before install" for the missing-file insert case).
func removeMarkedSection(filePath, startMarker, endMarker string) (FileAction, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ActionNotFound, nil
		}
		return "", err
	}
	s := string(content)
	startIdx := strings.Index(s, startMarker)
	endIdx := strings.Index(s, endMarker)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return ActionKept, nil
	}
	spanEnd := endIdx + len(endMarker)

	before := strings.TrimRight(s[:startIdx], "\n")
	after := strings.TrimLeft(s[spanEnd:], "\n")

	var newContent string
	switch {
	case before == "" && after == "":
		newContent = ""
	case before == "":
		newContent = after
		if !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
	case after == "":
		newContent = before + "\n"
	default:
		newContent = before + "\n\n" + after
	}

	if newContent == "" {
		if err := os.Remove(filePath); err != nil {
			return "", err
		}
		return ActionRemoved, nil
	}
	if err := atomicWriteFile(filePath, newContent); err != nil {
		return "", err
	}
	return ActionRemoved, nil
}

// upsertInstructionsEntry binds a marker-fenced instructions block
// (startMarker + content + endMarker) to replaceOrAppendMarkedSection,
// for the 4 of 8 agent targets that get an instructions file (Claude,
// Codex, opencode, Gemini). Marker text and block content are passed in
// by the caller rather than imported from instructions.go, keeping this
// helper agnostic of the specific codegraph marker constants.
func upsertInstructionsEntry(filePath, startMarker, endMarker, content string) (FileResult, error) {
	body := startMarker + "\n" + content + "\n" + endMarker
	action, err := replaceOrAppendMarkedSection(filePath, body, startMarker, endMarker)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: filePath, Action: action}, nil
}

// atomicWriteFile writes content to path via a temp file created in the
// same directory followed by os.Rename, so a crash or interrupt mid-
// write never leaves a truncated or corrupted external agent config on
// disk (V12, T-06-03). Every write this package performs funnels through
// this one function — these targets are a materially higher-risk
// surface than codegraph's own self-contained store directory, since
// they are arbitrary third-party tool configs this project does not own.
//
// The implementation lives in internal/fsatomic (D-09 extraction), shared
// with internal/githooks; this is a thin delegation so every existing
// call site in this package is unaffected.
func atomicWriteFile(path, content string) error {
	return fsatomic.WriteFile(path, content)
}
