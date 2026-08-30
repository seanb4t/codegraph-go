package query

import "sort"

// filegraph_cycles.go computes strongly-connected-component membership
// over a file-level adjacency (GRF-04, D-06). It does NOT read the
// store, does NOT know about edge kinds, and never runs client-side — it
// takes an adjacency and returns component membership, nothing more.
// Computing cycles server-side, over the aggregated file graph
// Engine.FileGraph() already builds, is what keeps GRF-04's correctness
// independent of whichever renderer wins GRF-01 (D-06): a renderer swap
// never has to reimplement this.

// scccFrame is one entry on stronglyConnectedCycles' explicit work
// stack — the iterative form's replacement for a recursive call's own
// stack frame. neighborIdx tracks how far into v's (sorted) successor
// list this frame has advanced, so the outer loop can resume a
// partially-visited vertex exactly where it left off.
type scccFrame struct {
	v           string
	neighborIdx int
}

// stronglyConnectedCycles implements Tarjan's strongly-connected-
// components algorithm in its ITERATIVE form, with an explicit work
// stack (scccFrame) standing in for the call stack a recursive
// implementation would use. This is a hard requirement, not a style
// preference: component depth on a monorepo is bounded by the
// dependency chain, not by anything this code controls, and a deep
// dependency chain must not grow the goroutine stack proportionally to
// graph depth. No function in this file calls itself.
//
// adj maps a file path to the file paths it has an aggregated edge
// toward (FileGraph's rollup, kind-agnostic). The returned map holds an
// entry ONLY for members of a component of size 2 or more — a
// single-node component (no self-loop, since FileGraph already excludes
// self-edges) is never reported as a cycle, matching FileGraphResult's
// contract. Component ids are 1-based and assigned in strict, sorted-
// input-derived order: the adjacency's keys are iterated in sorted
// order, and each key's successor list is iterated in sorted order, so
// two calls over the same adjacency agree element for element rather
// than depending on Go's randomized map iteration.
func stronglyConnectedCycles(adj map[string][]string) map[string]int {
	// Collect every vertex reachable from either side of an edge — a
	// pure successor (appears only as a target) still needs a stable
	// place in the sorted start order, even though it can never itself
	// start a strongconnect call that discovers something new.
	verticesSet := make(map[string]struct{})
	sortedAdj := make(map[string][]string, len(adj))
	for v, neighbors := range adj {
		verticesSet[v] = struct{}{}
		cp := make([]string, len(neighbors))
		copy(cp, neighbors)
		sort.Strings(cp)
		sortedAdj[v] = cp
		for _, w := range neighbors {
			verticesSet[w] = struct{}{}
		}
	}
	vertices := make([]string, 0, len(verticesSet))
	for v := range verticesSet {
		vertices = append(vertices, v)
	}
	sort.Strings(vertices)

	var (
		nextIndex   int
		indices     = make(map[string]int, len(vertices))
		lowlink     = make(map[string]int, len(vertices))
		onStack     = make(map[string]bool, len(vertices))
		stack       []string
		componentID int
		result      = make(map[string]int)
	)

	for _, start := range vertices {
		if _, seen := indices[start]; seen {
			continue
		}

		// Iterative strongconnect(start): an explicit work stack of
		// scccFrame replaces the recursive call stack.
		work := []*scccFrame{{v: start}}
		indices[start] = nextIndex
		lowlink[start] = nextIndex
		nextIndex++
		stack = append(stack, start)
		onStack[start] = true

		for len(work) > 0 {
			top := work[len(work)-1]
			v := top.v
			neighbors := sortedAdj[v]

			if top.neighborIdx < len(neighbors) {
				w := neighbors[top.neighborIdx]
				top.neighborIdx++

				if _, seen := indices[w]; !seen {
					indices[w] = nextIndex
					lowlink[w] = nextIndex
					nextIndex++
					stack = append(stack, w)
					onStack[w] = true
					work = append(work, &scccFrame{v: w})
					continue
				}
				if onStack[w] && indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
				continue
			}

			// Every successor of v has been visited: v's frame is done.
			work = work[:len(work)-1]
			if len(work) > 0 {
				parent := work[len(work)-1]
				if lowlink[v] < lowlink[parent.v] {
					lowlink[parent.v] = lowlink[v]
				}
			}

			if lowlink[v] == indices[v] {
				var component []string
				for {
					w := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					onStack[w] = false
					component = append(component, w)
					if w == v {
						break
					}
				}
				if len(component) >= 2 {
					componentID++
					for _, w := range component {
						result[w] = componentID
					}
				}
			}
		}
	}

	return result
}
