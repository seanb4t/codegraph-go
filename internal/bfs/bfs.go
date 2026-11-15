// Package bfs holds the breadth-first walk by depth that the resolver,
// the sync and the queries share.
package bfs

// Levels walks the graph that next gives, breadth-first from start. It
// calls visit once for each depth, from depth 1, with the nodes whose
// least depth that is. Each node comes once, start never, and the nodes
// of one depth keep the order of next. The walk stops when visit returns
// true or no node is left, and Levels reports whether visit stopped it.
// visit must not change the slice it gets.
func Levels(start string, next func(node string) []string, visit func(level []string) bool) bool {
	seen := map[string]bool{start: true}
	level := []string{start}
	for {
		var deeper []string
		for _, n := range level {
			for _, m := range next(n) {
				if !seen[m] {
					seen[m] = true
					deeper = append(deeper, m)
				}
			}
		}
		if len(deeper) == 0 {
			return false
		}
		if visit(deeper) {
			return true
		}
		level = deeper
	}
}
