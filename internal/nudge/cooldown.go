package nudge

import (
	"os"
	"time"
)

// CooldownWindow is the minimum gap between two fires for one key (D-05).
const CooldownWindow = 60 * time.Second

// currentUID is the uid sentinel ownership is checked against; a seam so a
// same-package test can simulate a foreign owner.
var currentUID = os.Getuid

// SessionKey placeholder (06-03 RED).
func SessionKey(sessionID, agentID string) (key string, ok bool) { return "", false }

// DefaultDir placeholder (06-03 RED).
func DefaultDir() string { return "" }

// Gate placeholder (06-03 RED).
type Gate struct {
	Dir string
	Now func() time.Time
}

// Due placeholder (06-03 RED).
func (g Gate) Due(key string) bool { return false }

func sentinelName(key string) string { return "" }

func checkSentinel(path string, now time.Time) (due bool, err error) { return false, nil }

func recordFire(path string, now time.Time) error { return nil }
