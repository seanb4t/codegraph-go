package agents

import (
	"fmt"
	"slices"
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

// SkippedTarget is a target that --target all or --target auto left out
// because its Detect reported an error. Err says why.
type SkippedTarget struct {
	ID  TargetID
	Err error
}

// ResolveTargets resolves the --target flag's value at loc into a concrete,
// TargetID-sorted list of AgentTarget, and lists the targets it left out
// because Detect(loc).Err was set. A target is never both selected and
// skipped.
//   - "all": every registered target whose Detect reports no error
//   - "none": no targets
//   - "auto": the targets whose Detect(loc).Installed is true and whose Detect
//     reports no error. When that selects none, it falls back to the Claude
//     target alone, unless Claude itself was skipped, rather than installing
//     nothing, so a clean environment still gets a working install.
//   - anything else: a comma-separated list of TargetIDs, resolved to
//     exactly those targets, whatever Detect says, so that an explicit
//     request reports the error itself. An unknown id in the list is an
//     error.
func ResolveTargets(spec string, loc Location) ([]AgentTarget, []SkippedTarget, error) {
	switch spec {
	case "none":
		return nil, nil, nil
	case "all", "auto":
		var selected []AgentTarget
		var skipped []SkippedTarget
		for _, id := range AllTargetIDs() {
			t := registry[id]
			d := t.Detect(loc)
			switch {
			case d.Err != nil:
				skipped = append(skipped, SkippedTarget{ID: id, Err: d.Err})
			case spec == "all" || d.Installed:
				selected = append(selected, t)
			}
		}
		if spec == "auto" && len(selected) == 0 {
			claudeSkipped := slices.ContainsFunc(skipped, func(s SkippedTarget) bool { return s.ID == Claude })
			if claude, ok := GetTarget(Claude); ok && !claudeSkipped {
				selected = []AgentTarget{claude}
			}
		}
		return selected, skipped, nil
	default:
		parts := strings.Split(spec, ",")
		out := make([]AgentTarget, 0, len(parts))
		for _, raw := range parts {
			id := TargetID(strings.TrimSpace(raw))
			t, ok := GetTarget(id)
			if !ok {
				return nil, nil, fmt.Errorf("unknown agent target %q", id)
			}
			out = append(out, t)
		}
		return out, nil, nil
	}
}
