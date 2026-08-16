package orders

import "errors"

// Validate is the second of two identically-named top-level definitions
// that make up case (a) of the behavioral corpus — see
// accounts/validate.go's Validate. This is a free function in a different
// package sharing the exact same symbol name "Validate", so a plain
// `node Validate` query (no -f narrowing) must resolve to 2+ definitions.
func Validate(orderID string) error {
	if orderID == "" {
		return errEmptyOrder
	}
	return nil
}

var errEmptyOrder = errors.New("empty order id")
