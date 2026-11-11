package fsatomic

import "golang.org/x/sys/unix"

func exchangeOS(a, b string) error {
	return unix.RenamexNp(a, b, unix.RENAME_SWAP)
}
