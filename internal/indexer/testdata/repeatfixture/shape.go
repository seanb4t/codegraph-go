// This file does not compile: it declares Shape.Area twice, as generated
// or half-edited code can. Each copy still gets its own node, and each
// node keeps its own arity.
package repeat

type Shape struct{}

func (s *Shape) Area(a int) int { return a }

func (s *Shape) Area(a, b int) int { return a * b }
