package query

import "fmt"

// ErrNotFound and ErrInvalidArgument are the two exported classification
// sentinels every caller-reachable argument rejection and not-found error
// in this package is convertible to (ENG-01/ENG-02's typed-error work,
// plan 01-09's prerequisite). Plan 01-09's RPC layer maps a classified
// error onto a Connect code with errors.Is — never by matching on message
// text, the discipline internal/cli/serve.go's own comment records so a
// permission error can never masquerade as something else:
//
//   - ErrNotFound   -> Connect CodeNotFound. The caller's symbol/argument
//     resolved to nothing (e.g. "symbol %q not found").
//   - ErrInvalidArgument -> Connect CodeInvalidArgument. The caller
//     supplied a malformed or out-of-range argument (empty query, negative
//     limit/depth, an out-of-bound max-files, a path that escapes the repo
//     root, an unknown --kind, an unknown files format).
//
// classifiedError is the unexported carrier both sentinels' constructors
// (notFoundf/invalidArgumentf) return. Its Error() method returns its
// stored message VERBATIM — it is not a fmt.Errorf %w wrap. This is
// deliberate and load-bearing: codegraph explore's and codegraph node's
// stderr, and every message string byte the frozen golden suite pins, are
// a shipped contract this task must not move. A later contributor must
// NEVER "improve" classifiedError into a %w-prefixed wrap (e.g.
// fmt.Errorf("query: %w", cause)) — doing so would prepend bytes to every
// classified message and silently break that contract.
type classifiedError struct {
	class error
	msg   string
}

// Error returns the classified error's message verbatim — exactly the
// string the pre-classification fmt.Errorf call would have produced.
func (e *classifiedError) Error() string {
	return e.msg
}

// Is reports whether target is this error's classification sentinel,
// implementing the errors.Is protocol so a caller can classify a
// classifiedError without ever matching on its message text.
func (e *classifiedError) Is(target error) bool {
	return target == e.class
}

// ErrNotFound classifies a caller-reachable "resolved to nothing" error
// (e.g. a symbol that does not exist in the index). See the package-level
// doc comment above for the full class list and the Connect code plan
// 01-09 maps it to.
var ErrNotFound = fmt.Errorf("query: not found")

// ErrInvalidArgument classifies a caller-reachable malformed-or-out-of-
// range argument error. See the package-level doc comment above for the
// full class list and the Connect code plan 01-09 maps it to.
var ErrInvalidArgument = fmt.Errorf("query: invalid argument")

// notFoundf constructs an ErrNotFound-classified error carrying msg,
// formatted exactly like fmt.Errorf but never wrapping — see
// classifiedError's doc comment for why a %w wrap must never be
// introduced here.
func notFoundf(format string, args ...interface{}) error {
	return &classifiedError{class: ErrNotFound, msg: fmt.Sprintf(format, args...)}
}

// invalidArgumentf constructs an ErrInvalidArgument-classified error
// carrying msg, formatted exactly like fmt.Errorf but never wrapping —
// see classifiedError's doc comment for why a %w wrap must never be
// introduced here.
func invalidArgumentf(format string, args ...interface{}) error {
	return &classifiedError{class: ErrInvalidArgument, msg: fmt.Sprintf(format, args...)}
}
