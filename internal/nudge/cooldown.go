package nudge

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// CooldownWindow is the minimum gap between two fires for one (session,
// agent) key: fire on the first matched call, then at most once per window
// (D-05). It is the one place the window is defined.
const CooldownWindow = 60 * time.Second

// currentUID is the uid the sentinel directory and files must be owned by.
// It is a variable only so a same-package test can simulate a foreign
// owner, which a non-root test cannot otherwise produce.
var currentUID = os.Getuid

var (
	errSentinelSymlink = errors.New("nudge: sentinel is a symlink")
	errSentinelNotFile = errors.New("nudge: sentinel is not a regular file")
	errSentinelForeign = errors.New("nudge: sentinel is not owned by the current user")
)

// SessionKey builds the cooldown key for one harness session and agent
// (D-06, D-07): the main thread keys on the literal "main", each subagent
// on its own id, so a subagent's fresh context gets its own nudge. An empty
// session id yields ok=false — the nudge never fires unkeyed.
func SessionKey(sessionID, agentID string) (key string, ok bool) {
	if sessionID == "" {
		return "", false
	}
	if agentID == "" {
		agentID = "main"
	}
	return sessionID + "\x00" + agentID, true
}

// DefaultDir is the per-user sentinel directory under os.TempDir(), which
// honours TMPDIR (D-08).
func DefaultDir() string {
	return filepath.Join(os.TempDir(), "codegraph-nudge-"+strconv.Itoa(os.Getuid()))
}

// Gate decides whether the nudge fires for a key, recording each fire as
// the mtime of a zero-byte sentinel file in Dir. Now is the injectable
// clock (D-17); nil means time.Now.
type Gate struct {
	Dir string
	Now func() time.Time
}

// Due reports whether the nudge should fire for key now, and records the
// fire when it should. It returns true only when the key is outside its
// cooldown AND the fire was recorded; every failure — an unusable or
// foreign directory, a symlinked or foreign sentinel, any stat, open or
// time-setting error — resolves to false, i.e. silence (D-08). Two
// parallel callers may both see a key as due; D-08 accepts that rare
// double fire.
func (g Gate) Due(key string) bool {
	if key == "" || g.Dir == "" {
		return false
	}
	now := time.Now
	if g.Now != nil {
		now = g.Now
	}
	t := now()

	if err := os.Mkdir(g.Dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return false
	}
	info, err := os.Lstat(g.Dir)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() || !ownedByCurrentUser(info) {
		return false
	}

	path := filepath.Join(g.Dir, sentinelName(key))
	due, err := checkSentinel(path, t)
	if err != nil || !due {
		return false
	}
	return recordFire(path, t) == nil
}

// sentinelName is the lowercase hex SHA-256 of the key, so a session or
// agent id can never steer a path and never reaches the disk (T-06-13).
func sentinelName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// checkSentinel is the read layer: it inspects path without following a
// final symlink. A missing sentinel is due; a symlink, a non-regular file,
// or a file owned by another uid is an error. Otherwise the key is due
// once the sentinel's age reaches CooldownWindow, or when its mtime lies in
// the future (a clock stepped back), so a stale future stamp cannot
// silence the nudge indefinitely.
func checkSentinel(path string, now time.Time) (due bool, err error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if err := sentinelFileOK(info); err != nil {
		return false, err
	}
	age := now.Sub(info.ModTime())
	return age >= CooldownWindow || age < 0, nil
}

// recordFire is the record layer (RESEARCH Pattern 3). It opens path
// write-only with O_NOFOLLOW — so the open itself fails if the final
// component is a symlink, closing the check-then-act swap window — checks
// the OPEN descriptor's own stat, and sets the mtime through that
// descriptor. It never truncates and never writes a byte, and it never
// re-resolves the path by name: a path-based time setter would follow a
// symlink swapped in after the open.
func recordFire(path string, now time.Time) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if err := sentinelFileOK(info); err != nil {
		return err
	}
	tv := syscall.NsecToTimeval(now.UnixNano())
	return syscall.Futimes(int(f.Fd()), []syscall.Timeval{tv, tv})
}

// sentinelFileOK requires a regular, non-symlink file owned by the current
// uid.
func sentinelFileOK(info fs.FileInfo) error {
	if info.Mode()&fs.ModeSymlink != 0 {
		return errSentinelSymlink
	}
	if !info.Mode().IsRegular() {
		return errSentinelNotFile
	}
	if !ownedByCurrentUser(info) {
		return errSentinelForeign
	}
	return nil
}

// ownedByCurrentUser reports whether info's owner is currentUID(); an
// unreadable owner counts as foreign.
func ownedByCurrentUser(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(currentUID())
}
