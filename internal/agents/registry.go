package agents

import (
	"fmt"
	"sort"
	"strings"
)

// registry is the package-level TargetID -> AgentTarget map, populated by
// each per-agent target file's own init() calling registerTarget —
// mirrors the project's established registry-keyed-by-ID shape used for
// language-extractor registration (D-02, D-03). No real per-agent files
// are registered yet in 06-01; they land in 06-02/06-03.
var registry = map[TargetID]AgentTarget{}

// registerTarget indexes t by its own ID(). Called from each agent
// target's own init(), e.g. claude.go's init() { registerTarget(claudeTarget{}) }.
func registerTarget(t AgentTarget) {
	registry[t.ID()] = t
}

// GetTarget returns the registered AgentTarget for id, if any.
func GetTarget(id TargetID) (AgentTarget, bool) {
	t, ok := registry[id]
	return t, ok
}

// AllTargetIDs returns every registered TargetID, sorted ascending. Map
// iteration order is non-deterministic in Go, so this sort is
// load-bearing — not cosmetic — for --target all and the interactive
// multi-select's display order (D-03).
func AllTargetIDs() []TargetID {
	ids := make([]TargetID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// AllTargets returns every registered AgentTarget, in the same sorted
// order as AllTargetIDs.
func AllTargets() []AgentTarget {
	ids := AllTargetIDs()
	out := make([]AgentTarget, 0, len(ids))
	for _, id := range ids {
		out = append(out, registry[id])
	}
	return out
}

// DetectAll runs Detect(loc) for every registered target, keyed by
// TargetID — the source data behind --target auto's detected-only
// default and the interactive multi-select's prefilled selection (D-03).
func DetectAll(loc Location) map[TargetID]DetectionResult {
	out := make(map[TargetID]DetectionResult, len(registry))
	for id, t := range registry {
		out[id] = t.Detect(loc)
	}
	return out
}

// ResolveTargetFlag resolves the --target flag's value at loc into a
// concrete, TargetID-sorted list of AgentTarget (D-03):
//   - "all": every registered target
//   - "none": no targets
//   - "auto": only targets whose Detect(loc).Installed is true; if that
//     detects zero targets, falls back to just the Claude target rather
//     than installing nothing — least-surprise for a clean environment
//     (RESEARCH.md recommendation, not a hard requirement)
//   - anything else: a comma-separated list of TargetIDs, resolved to
//     exactly those targets; an unknown id in the list is an error
func ResolveTargetFlag(spec string, loc Location) ([]AgentTarget, error) {
	switch spec {
	case "all":
		return AllTargets(), nil
	case "none":
		return nil, nil
	case "auto":
		var detected []AgentTarget
		for _, id := range AllTargetIDs() {
			t := registry[id]
			if t.Detect(loc).Installed {
				detected = append(detected, t)
			}
		}
		if len(detected) == 0 {
			if claude, ok := GetTarget(Claude); ok {
				return []AgentTarget{claude}, nil
			}
			return nil, nil
		}
		return detected, nil
	default:
		parts := strings.Split(spec, ",")
		out := make([]AgentTarget, 0, len(parts))
		for _, raw := range parts {
			id := TargetID(strings.TrimSpace(raw))
			t, ok := GetTarget(id)
			if !ok {
				return nil, fmt.Errorf("unknown agent target %q", id)
			}
			out = append(out, t)
		}
		return out, nil
	}
}
