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
//
// This permissive fallback remains the deliberate posture for every
// caller except claudeSettingsPath's two AutoAllow steps: writeMcpEntry/
// removeMcpEntry (operating on .mcp.json and ~/.claude.json), plus every
// other agent target's config path across the other seven agents. Only
// addClaudeAllowPermission and removeClaudeAllowPermission moved to
// readJSONFileStrict (Plan 02 Task 3), because — and only because —
// claudeSettingsPath(loc) is also where Plan 01's writeHookEntry/
// removeHookEntry now merge, and those two already read it strictly. A
// single file must not have two contradictory read postures depending on
// which step reaches it first; every other file this package edits has
// exactly one writer-side step, so this divergence does not apply there.
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

// readJSONFileStrict parses path as a JSON object like readJSONFile, but
// distinguishes three outcomes instead of readJSONFile's two: absent file
// (empty map, existed=false, nil error), a genuine read failure (nil map,
// existed=false, the wrapped error), and a present-but-undecodable file —
// including one that is valid JSON but not an object, or empty (nil map,
// existed=true, an error naming path). This is deliberately the opposite
// posture from readJSONFile's documented empty-map fallback (above) — that
// fallback is correct for the MCP-entry/CLAUDE.md paths it already
// governs, and wrong for this phase's own read-error/malformed-file
// invariant (roadmap success criterion 4): a malformed or unreadable
// settings.json must make the caller write nothing, not silently proceed
// as if the file were empty. Modeled on internal/githooks/githooks.go's
// skip-and-accumulate read switch (CR-01/CR-02); readJSONFile itself is
// unmodified — every existing caller keeps its permissive fallback.
func readJSONFileStrict(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, false, nil
		}
		return nil, false, fmt.Errorf("could not read existing file: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, true, fmt.Errorf("%s: existing file is not valid JSON — fix or remove it manually: %w", path, err)
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, true, nil
}

// writeHookEntry is the array-scoped analog of writeMcpEntry for
// hooks.<event>, an array of independent {matcher, hooks[]} blocks rather
// than a single named map key (RESEARCH Pitfall 1). It reads path via
// readJSONFileStrict and returns the error unwritten if the read failed —
// a malformed or unreadable settings.json is left byte-untouched, never
// silently proceeded past. Ownership of a block is determined SOLELY by
// exact command-string match within the block's own hooks[] sub-array
// against ownCommands, never by the block's matcher value or shape: a user
// may legitimately register their own unrelated block under the same
// matcher (e.g. "startup") — RESEARCH Pitfall 1.
//
// A hand-edited codegraph-owned block therefore stops matching by command
// identity and is treated as unowned — codegraph's fresh block gets
// appended alongside it rather than overwriting it in place, producing a
// duplicate matcher entry. This is a deliberate, accepted tradeoff: a
// matcher-and-shape recovery heuristic was tried and reverted (Plan 03
// Task 3 → this revert) after security review found it let codegraph
// silently claim ownership of — and overwrite — any unrelated single-
// command hook a user placed under the same matcher name, whenever a
// codegraph manifest happened to be present at that location. Duplication
// is untidy; silently destroying content codegraph never wrote is not an
// acceptable trade to avoid it.
//
// If the owned partition already jsonDeepEquals the normalized ownBlocks,
// nothing is written (ActionUnchanged); otherwise the array is rebuilt as
// the unowned blocks in their original relative order followed by
// ownBlocks. Every unrelated event key and every unowned block under the
// same event is carried through untouched.
func writeHookEntry(path, event string, ownBlocks []any, ownCommands []string) (FileResult, error) {
	existing, existedBefore, err := readJSONFileStrict(path)
	if err != nil {
		return FileResult{}, err
	}

	hooks, _ := existing["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	events, _ := hooks[event].([]any)

	isOwned := func(block any) bool {
		obj, ok := block.(map[string]any)
		if !ok {
			return false
		}
		blockHooks, ok := obj["hooks"].([]any)
		if !ok {
			return false
		}
		for _, h := range blockHooks {
			hObj, ok := h.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := hObj["command"].(string)
			for _, own := range ownCommands {
				if cmd == own {
					return true
				}
			}
		}
		return false
	}

	var owned, unowned []any
	for _, b := range events {
		if isOwned(b) {
			owned = append(owned, b)
		} else {
			unowned = append(unowned, b)
		}
	}

	normalized, err := normalizeJSON(ownBlocks)
	if err != nil {
		return FileResult{}, err
	}
	normalizedOwn, _ := normalized.([]any)

	if jsonDeepEqual(owned, normalizedOwn) {
		return FileResult{Path: path, Action: ActionUnchanged}, nil
	}

	newEvents := append(append([]any{}, unowned...), normalizedOwn...)
	hooks[event] = newEvents
	existing["hooks"] = hooks

	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	action := ActionCreated
	if existedBefore {
		action = ActionUpdated
	}
	return FileResult{Path: path, Action: action}, nil
}

// atomicWriteExecutableFile writes content to path via fsatomic.WriteFile
// and then marks it owner-executable. fsatomic.WriteFile preserves an
// existing file's mode but defaults a brand-new file to 0644 (not
// executable), so a script written through the unmodified primitive would
// silently fail to run under Claude Code's SessionStart dispatch
// (RESEARCH Pitfall 3, NUDGE-01 regression risk). internal/fsatomic is
// not modified — this is an additive wrapper, not a new file-safety
// primitive.
func atomicWriteExecutableFile(path, content string) error {
	if err := fsatomic.WriteFile(path, content); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

// writeEmbeddedFile writes content to path — funnelled through
// atomicWriteFile or atomicWriteExecutableFile — only if the file's
// current bytes differ from content. This is what makes the non-JSON
// artifacts (SKILL.md, session-nudge.sh) raw-byte idempotent (D-07):
// re-running install against unchanged embedded content is a true no-op,
// never a rewrite with identical bytes.
//
// For an executable artifact, byte-identity alone is not enough to
// short-circuit: the content comparison says nothing about the file's
// mode, so a lost executable bit (chmod -x, a backup/restore cycle, an AV
// quarantine round-trip) would otherwise never self-heal once content
// pinned to the embedded version — the SessionStart hook would then
// silently stop running with no error anywhere (code review CR-02). When
// content matches but the executable bit is missing, chmod in place
// without a full rewrite and report ActionUpdated.
func writeEmbeddedFile(path, content string, executable bool) (FileResult, error) {
	existed := fileExists(path)
	if existed {
		current, err := os.ReadFile(path)
		if err != nil {
			return FileResult{}, err
		}
		if string(current) == content {
			if !executable {
				return FileResult{Path: path, Action: ActionUnchanged}, nil
			}
			info, statErr := os.Stat(path)
			if statErr != nil {
				return FileResult{}, statErr
			}
			if info.Mode()&0o111 != 0 {
				return FileResult{Path: path, Action: ActionUnchanged}, nil
			}
			if err := os.Chmod(path, 0o755); err != nil {
				return FileResult{}, err
			}
			return FileResult{Path: path, Action: ActionUpdated}, nil
		}
	}

	var writeErr error
	if executable {
		writeErr = atomicWriteExecutableFile(path, content)
	} else {
		writeErr = atomicWriteFile(path, content)
	}
	if writeErr != nil {
		return FileResult{}, writeErr
	}

	action := ActionCreated
	if existed {
		action = ActionUpdated
	}
	return FileResult{Path: path, Action: action}, nil
}

// removeHookEntry is the array-scoped removal analog of writeHookEntry,
// mirroring removeMcpEntry's keep-clean discipline for hooks.<event>
// (Plan 02 Task 1). It reads path via readJSONFileStrict and returns the
// error unwritten if the read failed — a malformed or unreadable
// settings.json is left byte-untouched on uninstall too, the same
// fail-loud posture writeHookEntry established for install. Ownership is
// identified purely by exact command-string match inside a block's own
// hooks[] sub-array against ownCommands, never by matcher value (T-07-04)
// — a user's own block sharing codegraph's matcher (e.g. "startup") is
// never touched.
//
// A block that mixes codegraph's own hook entry with an unrelated one in
// the same hooks[] sub-array has only codegraph's entry stripped; the
// block itself survives with the unrelated entry intact. Only when
// removing codegraph's entry would empty the sub-array does the whole
// block go. Once every block is resolved, the same keep-clean cascade
// removeMcpEntry already establishes applies here too: an emptied event
// array deletes the event key, an emptied hooks object deletes the
// top-level hooks key, and an emptied settings object deletes the file
// entirely — mirroring "the file never existed before install" for the
// uninstall direction. Reports ActionNotFound (never an error) when
// there is no hooks object, no such event, or nothing owned to remove —
// matching the pre-existing D-08 invariant.
func removeHookEntry(path, event string, ownCommands []string) (FileResult, error) {
	existing, _, err := readJSONFileStrict(path)
	if err != nil {
		return FileResult{}, err
	}

	hooks, ok := existing["hooks"].(map[string]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}
	events, ok := hooks[event].([]any)
	if !ok {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}

	isOwnCommand := func(cmd string) bool {
		for _, own := range ownCommands {
			if cmd == own {
				return true
			}
		}
		return false
	}

	var anyRemoved bool
	var kept []any
	for _, b := range events {
		obj, ok := b.(map[string]any)
		if !ok {
			kept = append(kept, b)
			continue
		}
		blockHooks, ok := obj["hooks"].([]any)
		if !ok {
			kept = append(kept, b)
			continue
		}

		var survivingHooks []any
		blockChanged := false
		for _, h := range blockHooks {
			hObj, ok := h.(map[string]any)
			if !ok {
				survivingHooks = append(survivingHooks, h)
				continue
			}
			cmd, _ := hObj["command"].(string)
			if isOwnCommand(cmd) {
				blockChanged = true
				anyRemoved = true
				continue
			}
			survivingHooks = append(survivingHooks, h)
		}

		if !blockChanged {
			kept = append(kept, b)
			continue
		}
		if len(survivingHooks) == 0 {
			// The whole block was codegraph's own — drop it entirely.
			continue
		}
		newObj := make(map[string]any, len(obj))
		for k, v := range obj {
			newObj[k] = v
		}
		newObj["hooks"] = survivingHooks
		kept = append(kept, newObj)
	}

	if !anyRemoved {
		return FileResult{Path: path, Action: ActionNotFound}, nil
	}

	if len(kept) == 0 {
		delete(hooks, event)
	} else {
		hooks[event] = kept
	}
	if len(hooks) == 0 {
		delete(existing, "hooks")
	} else {
		existing["hooks"] = hooks
	}

	if len(existing) == 0 {
		if err := os.Remove(path); err != nil {
			return FileResult{}, err
		}
		return FileResult{Path: path, Action: ActionRemoved}, nil
	}
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

// removeEmbeddedFile removes path if it exists, reporting ActionRemoved.
// Reports ActionNotFound (never an error) when path does not exist,
// matching the pre-existing D-08 invariant; any other os.Remove error is
// surfaced unwrapped for the caller to attach path context to via
// recordFile.
func removeEmbeddedFile(path string) (FileResult, error) {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return FileResult{Path: path, Action: ActionNotFound}, nil
		}
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: ActionRemoved}, nil
}

// removeSkillDirIfEmpty removes dir only when it is already empty.
// os.Remove on a directory succeeds solely under that condition and
// fails otherwise with a platform-specific "not empty" error (ENOTEMPTY
// on Unix, a different code on Windows) — rather than matching that
// value, this confirms directly by re-reading dir: if entries remain,
// the failure was exactly the expected one and is a deliberate no-op, so
// a user-authored file (or Phase 6's own verification/ subdirectory)
// keeps the directory alive. Never a recursive delete — the directory is
// codegraph-named but not codegraph-exclusive, and losing a user's file
// there is irreversible while leaving an empty directory behind is not
// (this plan's must_haves.prohibitions).
func removeSkillDirIfEmpty(dir string) error {
	err := os.Remove(dir)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	entries, readErr := os.ReadDir(dir)
	if readErr == nil && len(entries) > 0 {
		return nil
	}
	return err
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
