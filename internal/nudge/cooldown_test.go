package nudge

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// gateT is the fixed instant every gate test's injected clock reads (D-17):
// the cooldown is driven by the clock and by mtimes the test plants, never
// by waiting.
var gateT = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// mustKey returns SessionKey's key and fails the test when it is not ok.
func mustKey(t *testing.T, session, agent string) string {
	t.Helper()
	k, ok := SessionKey(session, agent)
	if !ok {
		t.Fatalf("SessionKey(%q, %q) ok = false, want true", session, agent)
	}
	return k
}

// plantMtime sets a sentinel's mtime from the test (D-17). Only test code
// uses the path-based setter; the gate itself times the open descriptor.
func plantMtime(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("plant mtime on %s: %v", path, err)
	}
}

func TestSessionKey(t *testing.T) {
	cases := []struct {
		session, agent string
		wantKey        string
		wantOK         bool
	}{
		{"s", "", "s\x00main", true},
		{"s", "a1", "s\x00a1", true},
		{"", "a1", "", false},
		{"", "", "", false},
	}
	for _, tc := range cases {
		key, ok := SessionKey(tc.session, tc.agent)
		if key != tc.wantKey || ok != tc.wantOK {
			t.Errorf("SessionKey(%q, %q) = (%q, %v), want (%q, %v)", tc.session, tc.agent, key, ok, tc.wantKey, tc.wantOK)
		}
	}
	main, _ := SessionKey("s", "")
	sub, _ := SessionKey("s", "a1")
	if main == sub {
		t.Errorf("SessionKey(s, \"\") == SessionKey(s, a1) == %q; the main thread and a subagent must not share a key (D-06)", main)
	}
}

func TestGate_FirstCallFiresThenCooldown(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	k := mustKey(t, "s", "")

	g := Gate{Dir: dir, Now: fixedClock(gateT)}
	if !g.Due(k) {
		t.Fatal("first call: Due = false, want true (fire on the first matched call, D-05)")
	}
	if g.Due(k) {
		t.Error("second call at the same instant: Due = true, want false")
	}
	if (Gate{Dir: dir, Now: fixedClock(gateT.Add(59 * time.Second))}).Due(k) {
		t.Error("call at T+59s: Due = true, want false (inside the cooldown)")
	}
}

func TestGate_CooldownBoundary(t *testing.T) {
	cases := []struct {
		name  string
		mtime time.Time
		want  bool
	}{
		{"age_59s_inside", gateT.Add(-59 * time.Second), false},
		{"age_60s_exactly_due", gateT.Add(-CooldownWindow), true},
		{"age_61s_due", gateT.Add(-61 * time.Second), true},
		{"future_mtime_due", gateT.Add(time.Hour), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "nudge")
			k := mustKey(t, "s", "")
			g := Gate{Dir: dir, Now: fixedClock(gateT)}
			if !g.Due(k) {
				t.Fatal("planting fire: Due = false, want true")
			}
			path := filepath.Join(dir, sentinelName(k))
			plantMtime(t, path, tc.mtime)

			if got := g.Due(k); got != tc.want {
				t.Fatalf("Due with sentinel age %v = %v, want %v", gateT.Sub(tc.mtime), got, tc.want)
			}
			if tc.want {
				info, err := os.Lstat(path)
				if err != nil {
					t.Fatalf("lstat sentinel after fire: %v", err)
				}
				if !info.ModTime().Equal(gateT) {
					t.Errorf("sentinel mtime after fire = %v, want the clock %v (fire re-recorded)", info.ModTime(), gateT)
				}
			}
		})
	}
}

func TestGate_SeparateKeysForMainAndSubagent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	mainKey := mustKey(t, "s", "")
	subKey := mustKey(t, "s", "sub-1")

	at := func(d time.Duration) Gate { return Gate{Dir: dir, Now: fixedClock(gateT.Add(d))} }

	if !at(0).Due(mainKey) {
		t.Fatal("main at T: Due = false, want true")
	}
	if !at(0).Due(subKey) {
		t.Fatal("subagent at T: Due = false, want true (a subagent has its own cooldown, D-06)")
	}
	if at(30 * time.Second).Due(mainKey) {
		t.Error("main at T+30s: Due = true, want false")
	}
	if at(30 * time.Second).Due(subKey) {
		t.Error("subagent at T+30s: Due = true, want false")
	}
	if !at(60 * time.Second).Due(mainKey) {
		t.Error("main at T+60s: Due = false, want true")
	}
	if !at(60 * time.Second).Due(subKey) {
		t.Error("subagent at T+60s: Due = false, want true")
	}
}

func TestGate_DirCreated0700(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	if !(Gate{Dir: dir, Now: fixedClock(gateT)}).Due(mustKey(t, "s", "")) {
		t.Fatal("first call: Due = false, want true")
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatalf("lstat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%s mode = %v, want a directory", dir, info.Mode())
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("%s perm = %o, want 700 (D-08)", dir, perm)
	}
}

// TestGate_LoosePermissionsIsSilent is WR-01's regression test
// (06-REVIEW.md): a PRE-EXISTING sentinel directory with permission bits
// looser than 0700 (e.g. left behind by an older binary before this Gate
// existed, or widened by a misconfigured umask) must be refused exactly
// like a symlinked or foreign-owned one, since ordinary directory
// permission bits — not just ownership — govern who else on a shared
// multi-user machine can list, create, or delete entries inside it.
func TestGate_LoosePermissionsIsSilent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Positive control: an existing 0700 directory still fires.
	if !(Gate{Dir: dir, Now: fixedClock(gateT)}).Due(mustKey(t, "control", "")) {
		t.Fatal("control: Due = false for a pre-existing 0700 dir, want true")
	}

	loose := filepath.Join(t.TempDir(), "nudge")
	if err := os.Mkdir(loose, 0o755); err != nil {
		t.Fatal(err)
	}
	if (Gate{Dir: loose, Now: fixedClock(gateT)}).Due(mustKey(t, "s", "")) {
		t.Error("Due in a pre-existing 0755 dir = true, want false (WR-01: enforce the 0700 invariant on a pre-existing sentinel dir)")
	}
	entries, err := os.ReadDir(loose)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("loosely-permissioned dir gained %d entries, want none", len(entries))
	}
}

func TestSentinel_ReadRefusesSymlink(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	plantMtime(t, target, gateT.Add(-time.Hour))
	link := filepath.Join(d, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	// Positive control: the target itself is an owned regular file an hour
	// old, so it reads as due — only the link must be refused.
	if due, err := checkSentinel(target, gateT); err != nil || !due {
		t.Fatalf("checkSentinel(target) = (%v, %v), want (true, nil)", due, err)
	}
	if due, err := checkSentinel(link, gateT); err == nil {
		t.Fatalf("checkSentinel(symlink) = (%v, nil), want an error (D-08: refuse a symlinked sentinel)", due)
	}
}

func TestSentinel_RecordRefusesSymlink(t *testing.T) {
	d := t.TempDir()
	victim := filepath.Join(d, "victim")
	if err := os.WriteFile(victim, []byte("victim"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := gateT.Add(-24 * time.Hour)
	plantMtime(t, victim, old)
	link := filepath.Join(d, "link")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}

	if err := recordFire(link, gateT); err == nil {
		t.Error("recordFire(symlink) = nil, want an error (D-08: never write through a symlink)")
	}
	info, err := os.Lstat(victim)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Errorf("victim mtime = %v, want unchanged %v", info.ModTime(), old)
	}
	if b, _ := os.ReadFile(victim); string(b) != "victim" {
		t.Errorf("victim content = %q, want unchanged", b)
	}

	// Positive control: a plain path is recorded, with the clock's mtime.
	plain := filepath.Join(d, "plain")
	if err := recordFire(plain, gateT); err != nil {
		t.Fatalf("recordFire(plain) = %v, want nil", err)
	}
	if info, err := os.Lstat(plain); err != nil || !info.ModTime().Equal(gateT) {
		t.Fatalf("plain sentinel = (%v, %v), want mtime %v", info, err, gateT)
	}
}

func TestGate_SymlinkedDirIsSilent(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "real")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(d, "nudge")
	if err := os.Symlink(target, dir); err != nil {
		t.Fatal(err)
	}

	if (Gate{Dir: dir, Now: fixedClock(gateT)}).Due(mustKey(t, "s", "")) {
		t.Error("Due through a symlinked Dir = true, want false (D-08)")
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("symlink target gained %d entries, want none", len(entries))
	}
}

func TestGate_ForeignOwnedIsSilent(t *testing.T) {
	k := mustKey(t, "s", "")

	t.Run("existing_dir", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nudge")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		// Positive control before the uid is faked: the same Dir fires.
		if !(Gate{Dir: dir, Now: fixedClock(gateT)}).Due(mustKey(t, "control", "")) {
			t.Fatal("control: Due = false with the real uid, want true")
		}
		fakeForeignUID(t)
		if (Gate{Dir: dir, Now: fixedClock(gateT)}).Due(k) {
			t.Error("Due in a Dir owned by another uid = true, want false (D-08)")
		}
	})

	t.Run("existing_sentinel", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nudge")
		g := Gate{Dir: dir, Now: fixedClock(gateT)}
		if !g.Due(k) {
			t.Fatal("planting fire: Due = false, want true")
		}
		path := filepath.Join(dir, sentinelName(k))
		plantMtime(t, path, gateT.Add(-time.Hour))
		fakeForeignUID(t)
		if g.Due(k) {
			t.Error("Due with a foreign-owned Dir and sentinel = true, want false (D-08)")
		}
		if _, err := checkSentinel(path, gateT); err == nil {
			t.Error("checkSentinel on a foreign-owned sentinel = nil error, want an error (D-08)")
		}
		if err := recordFire(path, gateT); err == nil {
			t.Error("recordFire on a foreign-owned sentinel = nil, want an error (D-08)")
		}
	})
}

// fakeForeignUID makes the gate believe it runs as another uid, so every
// file this test created reads as foreign-owned; restored on cleanup.
func fakeForeignUID(t *testing.T) {
	t.Helper()
	orig := currentUID
	currentUID = func() int { return os.Getuid() + 1 }
	t.Cleanup(func() { currentUID = orig })
}

func TestGate_UnwritableBaseIsSilent(t *testing.T) {
	file := filepath.Join(t.TempDir(), "regular-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// A path under a regular file fails with ENOTDIR, even when run as root.
	dir := filepath.Join(file, "nudge")
	if (Gate{Dir: dir, Now: fixedClock(gateT)}).Due(mustKey(t, "s", "")) {
		t.Error("Due under a regular-file base = true, want false (D-08: any failure is silent)")
	}
}

func TestGate_ParallelRuns(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	k := mustKey(t, "s", "")
	g := Gate{Dir: dir, Now: fixedClock(gateT)}

	const n = 16
	results := make([]bool, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = g.Due(k)
		}()
	}
	wg.Wait()

	fired := 0
	for _, r := range results {
		if r {
			fired++
		}
	}
	if fired < 1 {
		t.Errorf("parallel runs fired %d times, want at least 1", fired)
	}
	if _, err := os.Lstat(filepath.Join(dir, sentinelName(k))); err != nil {
		t.Errorf("sentinel after parallel runs: %v, want it to exist", err)
	}
}

func TestGate_SentinelHoldsNoSessionContent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nudge")
	g := Gate{Dir: dir, Now: fixedClock(gateT)}
	for _, k := range []string{mustKey(t, "session-SECRET", ""), mustKey(t, "session-SECRET", "agent-SECRET")} {
		if !g.Due(k) {
			t.Fatalf("Due(%q) = false, want true", k)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("Dir holds %d entries, want 2 (one per key)", len(entries))
	}
	hexName := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, e := range entries {
		if !hexName.MatchString(e.Name()) || strings.Contains(e.Name(), "SECRET") {
			t.Errorf("sentinel name %q, want a 64-char lowercase hex digest", e.Name())
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 0 {
			t.Errorf("sentinel %s size = %d, want 0", e.Name(), info.Size())
		}
		if b, _ := os.ReadFile(path); strings.Contains(string(b), "SECRET") {
			t.Errorf("sentinel %s holds session content", e.Name())
		}
	}
}

func TestDefaultDirHonoursTMPDIR(t *testing.T) {
	d := t.TempDir()
	t.Setenv("TMPDIR", d)
	want := filepath.Join(d, "codegraph-nudge-"+strconv.Itoa(os.Getuid()))
	if got := DefaultDir(); got != want {
		t.Errorf("DefaultDir() = %q, want %q", got, want)
	}
}
