//go:build !windows

package wtindex

import (
	"os"
	"syscall"
)

// sameDevice reports whether paths a and b are on one filesystem, where a
// checkpoint can hard-link instead of copying. A test can replace it.
var sameDevice = func(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return ai.Sys().(*syscall.Stat_t).Dev == bi.Sys().(*syscall.Stat_t).Dev, nil
}
