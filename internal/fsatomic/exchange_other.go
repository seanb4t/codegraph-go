//go:build !linux && !darwin

package fsatomic

import "errors"

func exchangeOS(a, b string) error {
	return errors.ErrUnsupported
}
