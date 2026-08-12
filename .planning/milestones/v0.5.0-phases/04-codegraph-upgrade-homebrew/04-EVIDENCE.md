# Phase 4 Plan 06 — Recorded Evidence

Single recorded-observation file for plan `04-06`, following `03-EVIDENCE.md`'s sectioning
and fixed-key-order conventions: a command, its verbatim output, and an explicit verdict —
never a narrative conclusion in place of the transcript.

All observations on this page were captured on **2026-08-11** on the maintainer's real Mac
(Apple Silicon, `darwin/arm64`), the same host every prior evidence file in this milestone was
captured on:

```
ProductName:  macOS
ProductVersion: 27.0
BuildVersion: 26A5388g
Darwin denver 27.0.0 Darwin Kernel Version 27.0.0: Tue Jul 14 21:43:10 PDT 2026; ... RELEASE_ARM64_T6041 arm64
Homebrew 6.0.17
```

This is a **pre-release macOS version**, as every prior evidence file in this milestone
states plainly — Homebrew warns "You are using macOS 27. We do not provide support for this
pre-release version." on the `brew install` below.

Every mutating step — `brew tap`, `brew trust`, `brew install`, and the payload
substitution — ran inside ONE trapped shell script (`RUN_ID=run-20260811T175959Z-35769`,
`TRAP_ARMED=1` written to the baseline between registering `trap restore_and_cleanup EXIT`
and the first mutating byte), whose EXIT trap restored the machine. The three run artifacts
(baseline, harness log, restore receipt) all carry that same `RUN_ID`.

## What this file proves, and what it does not

Two legs below are **executed evidence**, with different evidentiary status from each other.
A third leg is **named, not executed, and not claimed** — see Leg 3. This distinction is the
entire point of this document (T-04-20): presenting Leg 2's payload-substituted observation as
equivalent to a naturally-installed release binary would be the most consequential available
overclaim in this milestone, closing UPGR-01 on evidence that does not support it.

---

## Leg 1 — the genuine layout, no mutation. Executed evidence.

This is what a constructed fixture could not prove: a real `brew tap seanb4t/tap` +
`brew install codegraph`, producing Homebrew's own real Caskroom layout and its own real
`INSTALL_RECEIPT.json` placement, with the detector run against that tree completely unmodified.

### `brew tap seanb4t/tap`

```
$ brew tap seanb4t/tap
==> Auto-updating Homebrew...
Adjust how often this is run with `$HOMEBREW_AUTO_UPDATE_SECS` or disable with
`$HOMEBREW_NO_AUTO_UPDATE=1`. Hide these hints with `$HOMEBREW_NO_ENV_HINTS=1` (see `man brew`).
==> Auto-updated Homebrew!
==> Updated Homebrew from 4dacfe77a2 to d67b92d2de.
Updated 1 tap (dicklesworthstone/tap).
Error: Failed to import: /opt/homebrew/Library/Taps/hashicorp/homebrew-tap/Formula/vagrant.rb
hashicorp/tap/vagrant: formula requires at least a URL

==> Tapping seanb4t/tap
Cloning into '/opt/homebrew/Library/Taps/seanb4t/homebrew-tap'...
Tapped 1 cask (15 files, 17.7KB).
```

Exit `0`. (The `hashicorp/tap/vagrant` import failure is Homebrew's own unrelated
auto-update noise against a pre-existing, unrelated third-party tap already on this
machine — not this run's tap, not this run's mutation, and not retried or worked around.)

### `brew trust --tap seanb4t/tap`

```
$ brew trust --tap seanb4t/tap
Already trusted tap: seanb4t/tap
```

Exit `0`. The baseline probe (`brew trust --json v1`, taken BEFORE this command ran) had
already recorded `seanb4t/tap` present in the `taps` array — `TAP_TRUSTED_BEFORE=yes` — so
this command's "already trusted" response is consistent with the pre-run state, not a new
grant this run made.

### `brew install codegraph`

```
$ brew install codegraph
==> Would install 1 cask:
seanb4t/tap/codegraph
Warning: You are using macOS 27.
We do not provide support for this pre-release version.
==> Fetching downloads for: seanb4t/tap/codegraph
✔︎ Cask codegraph (0.8.0)
==> Installing Cask codegraph
==> Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'
🍺  codegraph was successfully installed!
```

Exit `0`.

### The genuine layout, recorded verbatim

```
$ brew list --cask --versions codegraph
codegraph 0.8.0

$ ls -l /opt/homebrew/bin/codegraph
lrwxr-xr-x@ 1 sean  admin  48 Aug 11 14:00 /opt/homebrew/bin/codegraph -> /opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
$ realpath /opt/homebrew/bin/codegraph
/opt/homebrew/Caskroom/codegraph/0.8.0/codegraph

$ ls -la /opt/homebrew/Caskroom/codegraph
total 0
drwxr-xr-x@  4 sean  admin   128 Aug 11 14:00 .
drwxrwxr-x  90 sean  admin  2880 Aug 11 14:00 ..
drwxr-xr-x@  5 sean  admin   160 Aug 11 14:00 .metadata
drwxr-xr-x@  7 sean  admin   224 Aug 11 14:00 0.8.0

$ ls -la /opt/homebrew/Caskroom/codegraph/0.8.0
total 123608
drwxr-xr-x@ 7 sean  admin       224 Aug 11 14:00 .
drwxr-xr-x@ 4 sean  admin       128 Aug 11 14:00 ..
-rw-r--r--@ 1 sean  admin       168 Aug 11 14:00 .codegraph-brew-install
-rw-r--r--@ 1 sean  admin     30933 Aug 10 15:22 CHANGELOG.md
-rwxr-xr-x@ 1 sean  admin  63231081 Aug 10 15:25 codegraph
-rw-r--r--@ 1 sean  admin      1068 Aug 10 15:22 LICENSE
-rw-r--r--@ 1 sean  admin      8387 Aug 10 15:22 README.md

$ ls -la /opt/homebrew/Caskroom/codegraph/.metadata/
total 16
drwxr-xr-x@ 5 sean  admin   160 Aug 11 14:00 .
drwxr-xr-x@ 4 sean  admin   128 Aug 11 14:00 ..
drwxr-xr-x@ 3 sean  admin    96 Aug 11 14:00 0.8.0
-rw-r--r--@ 1 sean  admin   860 Aug 11 14:00 config.json
-rw-r--r--@ 1 sean  admin  1065 Aug 11 14:00 INSTALL_RECEIPT.json

$ find /opt/homebrew/Caskroom/codegraph/.metadata/ -iname 'INSTALL_RECEIPT.json'
/opt/homebrew/Caskroom/codegraph/.metadata/INSTALL_RECEIPT.json
```

### `TestDetectBrewManaged_RealInstall` against that tree — read out of the run's own harness log

Re-running the harness after the task is impossible by construction (the merged task ends with
the cask uninstalled and `$(brew --prefix)/bin/codegraph` gone), so this is the harness's own
verbatim `go test -v` output, captured to `${TMPDIR}/codegraph-04-06-harness.log` during the run,
with the same `RUN_ID` the baseline and receipt carry:

```
RUN_ID=run-20260811T175959Z-35769
=== RUN   TestDetectBrewManaged_RealInstall
    brew_test.go:348: resolved binary: /opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
    brew_test.go:349: install dir: /opt/homebrew/Caskroom/codegraph/0.8.0
    brew_test.go:350: tree: Caskroom
    brew_test.go:351: token: codegraph
    brew_test.go:352: version: 0.8.0
    brew_test.go:353: receipt: /opt/homebrew/Caskroom/codegraph/.metadata/INSTALL_RECEIPT.json
--- PASS: TestDetectBrewManaged_RealInstall (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.164s
```

`TestDetectBrewManaged_RealInstall` aborts by name on a missing/nonexistent
`CODEGRAPH_BREW_ACCEPTANCE_PATH` and never skips once it is set (see `internal/upgrade/brew_test.go`
:331-354), so this `PASS` is a positive detection against the real tree, not an absent-test
no-op.

**Cross-check: every logged field against the on-disk listings above.**

| Logged field | Value | On-disk listing | Agrees? |
|---|---|---|---|
| resolved binary | `/opt/homebrew/Caskroom/codegraph/0.8.0/codegraph` | `realpath /opt/homebrew/bin/codegraph` above | YES |
| install dir | `/opt/homebrew/Caskroom/codegraph/0.8.0` | `ls -la .../0.8.0` above, and the symlink's resolved target's parent | YES |
| tree | `Caskroom` | `ls -la .../Caskroom/codegraph` above | YES |
| token | `codegraph` | Caskroom's immediate child directory name | YES |
| version | `0.8.0` | `brew list --cask --versions codegraph` above (`codegraph 0.8.0`) | YES |
| receipt | `/opt/homebrew/Caskroom/codegraph/.metadata/INSTALL_RECEIPT.json` | `find ... -iname 'INSTALL_RECEIPT.json'` above | YES |

The `bin/codegraph` symlink's resolved target (`/opt/homebrew/Caskroom/codegraph/0.8.0/codegraph`)
lies under the same Caskroom versioned directory the harness logged — the symlink resolution
did real work on the real layout, not a constructed fixture.

---

## Leg 2 — the observed behaviour, genuine layout with a substituted payload. Executed evidence, with a named substitution.

**Named here, in this leg's own heading, not in a footnote: the file at
`/opt/homebrew/Caskroom/codegraph/0.8.0/codegraph` was overwritten in place with a binary built
from this worktree's own source tree, immediately after Leg 1's unmutated harness run completed.
The tree, the `bin/codegraph` symlink, the `INSTALL_RECEIPT.json` receipt, and every other piece
of Homebrew's own bookkeeping were left untouched — only the payload bytes changed, which is
precisely the variable that had to change to observe this phase's own code running through the
installed symlink.** This is a genuine Caskroom layout with a substituted payload, never
presented here or anywhere else as a naturally-installed release binary.

### Preservation and substitution, with hashes

```
built binary (this worktree's ./cmd/codegraph):
  version --json: {"version":"dev","commit":"unknown","date":"unknown","go_version":"go1.26.5","os":"darwin","arch":"arm64"}
  sha256:          369162973d3cb74ed82f3dc6f5e6da39905eac552192a6c95f32bdc4cc19020c

original Caskroom payload, preserved to scratch BEFORE substitution:
  PAYLOAD_SHA256_BEFORE=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62

Caskroom payload sha256 immediately AFTER substitution (should equal the built binary's):
  369162973d3cb74ed82f3dc6f5e6da39905eac552192a6c95f32bdc4cc19020c   -> MATCH
```

### The four observations, through the installed `bin/codegraph` symlink

All four ran with networking available; none of the four showed any latency suggestive of a
network round trip — consistent with D-11's offline-path design, since detection fires and
returns before `resolveLatest` is ever reached.

```
$ /opt/homebrew/bin/codegraph upgrade
upgrade: codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
$ echo $?
1
```

```
$ /opt/homebrew/bin/codegraph upgrade --check
codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
$ echo $?
0
```

No release version number appears anywhere in the `--check` output other than the phrase
being the pointer sentence itself — brew owns the "is a newer version available" answer for a
brew-managed install, and this command does not manufacture a second one.

```
$ /opt/homebrew/bin/codegraph upgrade --force
upgrade: codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
$ echo $?
1
```

`upgrade --force`'s output is not merely a superset containing the same resolved-path pointer
sentence as bare `upgrade`'s (the acceptance bar this plan set) — it is byte-identical to it.
`--force` is powerless against the Homebrew-managed refusal.

```
$ /opt/homebrew/bin/codegraph upgrade --help
Download the target-platform binary from GitHub Releases, verify its
cosign-keyless signature/provenance in-process (never a cosign CLI), and
only then atomically replace the running binary. --check reports whether
a newer release is available without downloading anything.

A Homebrew-managed install is detected from the resolved location of
the running binary. codegraph upgrade refuses to run there and exits
non-zero, because it was asked for a mutation it declines to perform.
codegraph upgrade --check steps aside with the same pointer and exits
zero, because it only answered a question. Upgrade a Homebrew-managed
install with: brew upgrade codegraph.

Usage:
  codegraph upgrade [version] [flags]

Examples:
  codegraph upgrade --check
  codegraph upgrade
  codegraph upgrade v1.4.0
  codegraph upgrade --check  # brew-managed install: prints the pointer, exits 0

Flags:
      --check   report whether a newer release is available, without downloading
  -f, --force   reinstall even if already on the latest version
  -h, --help    help for upgrade
$ echo $?
0
```

`--help`'s output names both exit behaviours (the refusal and the `--check` step-aside) and
`brew upgrade codegraph`, observed against a real binary running from the real Caskroom path.

### Restoration — the EXIT trap's own receipt, positive floor

```
RUN_ID=run-20260811T175959Z-35769
PAYLOAD_SHA256_BEFORE=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
PAYLOAD_SHA256_AFTER=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
RESTORE_VERDICT=ok
RESTORE_INVOCATIONS=1
TAP_ACTION=untapped
TRUST_ACTION=left-trusted
```

Both payload hashes are equal — the bytes put back are the bytes taken away.
`RESTORE_INVOCATIONS=1` — the idempotency guard fired exactly once (no INT/TERM double-invoke
was exercised or needed on this run; the script completed normally and the EXIT trap alone ran
the restoration).

### Restoration — baseline vs. final, side by side (both directions)

| Probe | Baseline (before `brew tap`) | Final (after the trap) | Match? |
|---|---|---|---|
| `brew list --cask --versions codegraph` | `CASK_PREEXISTING=no` (nothing) | `(nothing)` | YES — still absent |
| `brew tap \| rg seanb4t/tap` | `TAP_PREEXISTING=no` (absent) | `(none)` | YES — untapped, as required by `TAP_PREEXISTING=no` |
| `brew trust --json v1` filtered for `seanb4t/tap` | `TAP_TRUSTED_BEFORE=yes` (present) | `"seanb4t/tap",` (still present) | YES — trust grant left in place, as required by `TAP_TRUSTED_BEFORE=yes` |
| `codegraph*.1` man page count | `MAN_PAGES_BEFORE=0` | `0` | YES |
| `ls -l $(brew --prefix)/bin/codegraph` | (did not exist before install) | `ls: ... No such file or directory` | YES — back to not existing |

The tap/trust pair is the load-bearing bidirectional case this plan's review cycles 2 and 3
exist to prove: this run added the tap (`TAP_PREEXISTING=no`) but did NOT add the trust grant
(`TAP_TRUSTED_BEFORE=yes` — the maintainer already trusted `seanb4t/tap` from a prior session),
and the trap correctly untapped while leaving the trust grant untouched — `TAP_ACTION=untapped`,
`TRUST_ACTION=left-trusted`, exactly the asymmetric outcome the recorded baseline required, not
the symmetric "clean machine" outcome an unconditional untap+untrust would have produced.

### `task test` and the working tree

```
$ task test
... (test:unit, test:golden, test:integration, test:wireoracle, test:daemon, test:race — all packages ok)
```

Exit `0`, no failures across unit, golden, integration, wireoracle, daemon, and race-detector
suites.

```
$ git status --porcelain
(no output)
```

No unexpected working-tree changes — this plan's own trapped script never touches this
repository; it only mutates Homebrew-owned paths and a scratch directory outside the repo.

```
$ git diff --exit-code go.mod go.sum
$ echo $?
0
```

No dependency added (T-04-SC).

---

## Leg 3 — the fully natural path. NOT executed, NOT claimed.

A released binary that itself carries this phase's detection code (`internal/upgrade/brew.go`,
plan 04-01), installed by `brew install` with no payload substitution whatsoever — the fully
natural path this plan's Leg 2 approximates but does not reach.

**This leg was not run. No run of it is scheduled by this plan.** The binary Leg 1 and Leg 2
installed and observed is `v0.8.0`, published before this phase's detection code existed in any
tagged release (04-01 through 04-06 all landed after `v0.8.0`); no release has yet been cut
containing this phase's code. Leg 2's substitution is exactly the accepted workaround for that
gap, named as a substitution rather than presented as this leg.

**Closing condition, stated precisely:** the next release cut after this phase's plans merge
republishes `Casks/codegraph.rb` to `seanb4t/tap` (GoReleaser's `cask.Pipe{}.Publish()`, the
same mechanism `03-EVIDENCE.md`'s BREW-02 already observed executing for real), carrying a
binary built from a tag that includes this phase's `internal/upgrade/brew.go` and
`internal/cli/upgrade.go` changes. At that point a `brew upgrade codegraph` (or a fresh
`brew install codegraph` on a virgin machine) exercises this exact detection path end to end
with nothing substituted. That release has not been cut as part of this plan and is not
scheduled by it — this is an accepted, named gap, in the same style Phase 3's own
"Scope reduction, recorded plainly" section (`03-EVIDENCE.md`) used for its own reduced-scope
observation, never a silently dropped claim.

---

## Criterion mapping

Amended 2026-08-11 (D-01, D-08, D-09, D-10, D-04R, plan 04-03) Phase-4 Success Criteria,
`.planning/ROADMAP.md` § "Phase 4: `codegraph upgrade` × Homebrew":

**Criterion 1** — After a genuine `brew tap` + `brew install codegraph` from the Phase-3 tap,
`codegraph upgrade` refuses, names `brew upgrade codegraph` in its message, and neither
`Options.download` nor `Options.swap` is ever invoked (UPGR-01). **Fully evidenced by Leg 1 +
Leg 2**: Leg 1 proves the genuine install; Leg 2's bare-`upgrade` observation shows exit `1`
with the message `upgrade: codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0).
Upgrade with: brew upgrade codegraph`. The seam-unfired half of this criterion (`download`/`swap`
never called) is not independently re-observed on the real binary here — a compiled release
binary carries no test hooks for those seams — and is instead evidenced by plan 04-01's
`TestUpgradeRun_RefusesBrewManagedCask`, a unit-level proof over the same code path this real
binary runs (`04-01-SUMMARY.md`). Combined: real-binary exit code and message (executed here) +
unit-level seam assertion (plan 04-01) together fully cover the criterion's text.

**Criterion 2** — `codegraph upgrade --check` under that install steps aside with the same
resolved-path pointer, mutates nothing, and exits 0; `brew upgrade codegraph` still succeeds
afterward (UPGR-03). **Fully evidenced by Leg 2**: the `--check` observation above shows exit
`0` and the same pointer sentence containing both the real Caskroom path and the literal
`brew upgrade codegraph`, with no second version number manufactured. The seam-unfired half is
evidenced the same way as Criterion 1, by plan 04-01's `TestUpgradeRun_CheckBrewManagedStepsAside`.
"`brew upgrade codegraph` still succeeds afterward" is NOT independently re-observed in this
plan — doing so would require cutting and publishing a second release from this tap, which is
exactly Leg 3's unexecuted, closing-condition-named gap.

**Criterion 3** — Detection fires on a resolved-symlink Caskroom or Cellar layout across
prefixes and platforms, and does NOT fire on a non-brew binary at a path merely containing the
string `Cellar` (UPGR-02). **Evidenced by unit-level proof, not this run**: this plan exercises
exactly one prefix (Apple Silicon `/opt/homebrew`) and exactly one tree (Caskroom) — the real
tap publishes only a cask, never a formula, so no real Cellar tree exists to observe here. The
full prefix × tree × false-positive matrix (Intel `/usr/local`, custom prefix, linuxbrew, Cellar,
and the false-positive row) is plan 04-01's 16-row `TestDetectBrewManaged` table
(`04-01-SUMMARY.md`), a unit-level proof this plan does not and cannot re-run against real
hardware it does not have. What this plan adds beyond that table is exactly the third
unresolved spec-less-probe row's closing condition — see "Carried assumption dispositions"
below.

**Criterion 4** — A non-brew install on a machine where `brew` is absent from `PATH` upgrades
normally, so detection never becomes a hard dependency on Homebrew being present (UPGR-02).
**Satisfied by construction, and by plan 04-01's table, not by staging a brew-absent machine
here**: `detectBrewManaged` (`internal/upgrade/brew.go`) never shells out to `brew`, never
touches the network, and returns `(brewInstall{}, false)` on any `EvalSymlinks` error or on the
absence of a matching tree/receipt — there is no code path in the detector that depends on the
`brew` executable existing anywhere. Staging a machine with Homebrew fully absent to observe
this would prove nothing additional beyond reading the source, since the property under test is
an absence of a dependency, not a behavior conditioned on one.

---

## Carried assumption dispositions

Three spec-less-probe rows carried from `04-01-PLAN.md § <flagged_assumptions>`, each named by
requirement ID, each dispositioned explicitly:

| Requirement | Disposition | Detail |
|---|---|---|
| UPGR-01 | closed | 04-01's flagged assumption was "that the seam assertion plus the message-content assertion fully discharge 'refuses, pointing at `brew upgrade codegraph`, and never modifies the install tree', closed by plan 04-06 leg 2 (the real-tap observation of the actual exit code and message)". This plan's Leg 2 IS that observation: bare `upgrade` under the real install exits `1` with the exact message text, observed above. Closed. |
| UPGR-02 | partially closed, remainder carried forward | 04-01's flagged assumption was two-part: (a) "that the four prefixes × two tree shapes × false-positive rows exhaust what 'resolves symlinks to the real install path' requires... no edge for a relative symlink chain, a bind mount, or a case-insensitive filesystem was proposed", closed by "Task 2's table plus `TestDetectBrewManaged_RealInstall` against a genuine Homebrew tree (plan 04-06 leg 1)". Leg 1's harness run against the genuine Caskroom tree closes the "genuine tree, not a fixture" half of that assumption — the relative-symlink-chain, bind-mount, and case-insensitive-filesystem edges it names are NOT exercised by this run (Homebrew's own cask install produced an absolute-path Caskroom layout on a case-sensitive APFS volume, not any of those three edges) and remain unclosed, carried forward as-is. |
| UPGR-03 | closed | 04-01's flagged assumption was "that 'still works... read-only, no mutation... and reports how to upgrade' is satisfied by a nil return, the shared pointer line, and all four seams unfired, closed by plan 04-06 leg 2 (`codegraph upgrade --check` observed exit 0 against the real install)". This plan's Leg 2 IS that observation: `upgrade --check` exits `0` with the shared pointer line, observed above. The seams-unfired half is evidenced by plan 04-01's unit test, as noted in the Criterion 2 mapping above. Closed. |

---

```
UPGR-ACCEPTANCE-EVIDENCE brew_version=6.0.17 cask_version=0.8.0 install_dir=/opt/homebrew/Caskroom/codegraph/0.8.0 upgrade_exit=1 upgrade_check_exit=0 upgrade_force_exit=1 payload_sha256_before=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62 payload_sha256_after=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62 restore_verdict=ok tap_action=untapped trust_action=left-trusted leg3_executed=no
```

---

## Full trapped-script transcript (both legs, one script, `RUN_ID=run-20260811T175959Z-35769`)

```
=== STEP 1: set -uo pipefail (active) ===
=== STEP 2: purge, mint, probe, record — all before the first mutating byte ===
RUN_ID=run-20260811T175959Z-35769
--- brew --version ---
Homebrew 6.0.17
Homebrew/homebrew-core (git revision f9658f49139; last commit 2026-08-11)
--- brew list --cask --versions codegraph (exit=1) ---

CASK_PREEXISTING derived: no
--- brew tap | rg '^seanb4t/tap$' (BEFORE brew tap) ---
TAP_PROBE_OUT=[]
TAP_PREEXISTING derived: no
--- brew trust --json v1 (BEFORE brew trust), exit=0 ---
{
  "taps": [
    "andyyyy64/whichllm",
    "antoniorodr/memo",
    "atomicjar/tap",
    "bats-core/bats-core",
    "buo/cask-upgrade",
    "controlplaneio-fluxcd/tap",
    "dagger/tap",
    "dicklesworthstone/tap",
    "fluxcd/tap",
    "fzymgc-house/tap",
    "gastownhall/beads",
    "go-task/tap",
    "hashicorp/tap",
    "int128/kubelogin",
    "jbangdev/tap",
    "nats-io/nats-tools",
    "openhue/cli",
    "oven-sh/bun",
    "pulumi/tap",
    "schpet/tap",
    "seanb4t/tap",
    "yakitrak/yakitrak",
    "zjrosen/perles"
  ],
  "formulae": [
    "dagger/tap/dagger",
    "morantron/tmux-fingers/tmux-fingers",
    "namespacelabs/namespace/nsc",
    "omnigent-ai/tap/omnigent"
  ],
  "casks": [
    "file:///private/tmp/.../03-04-mutation/mutation1/tap-src/codegraph",
    "file:///private/tmp/.../03-04-mutation/mutation2/tap-src/codegraph",
    "file:///var/folders/_b/.../tmp.3wchrgtqce/tap-src/codegraph",
    "file:///var/folders/_b/.../tmp.c9x4gsh8vf/tap-src/codegraph",
    "file:///var/folders/_b/.../tmp.sfaxyk0zrk/tap-src/codegraph",
    "file:///var/folders/_b/.../tmp.vufvrwwjwx/tap-src/codegraph"
  ],
  "commands": []
}
TRUST_PROBE_MATCH=[    "seanb4t/tap",]
TAP_TRUSTED_BEFORE derived: yes
--- man page count (BEFORE) ---
MAN_PAGES_BEFORE_COUNT=0
BREW_PREFIX=/opt/homebrew
--- baseline file written (6 lines) ---
RUN_ID=run-20260811T175959Z-35769
TAP_PREEXISTING=no
TAP_TRUSTED_BEFORE=yes
CASK_PREEXISTING=no
MAN_PAGES_BEFORE=0
BREW_PREFIX=/opt/homebrew
=== STEP 3: arm the EXIT trap NOW, before brew tap (the first mutating byte) ===
--- baseline file now (7 lines, TRAP_ARMED appended) ---
RUN_ID=run-20260811T175959Z-35769
TAP_PREEXISTING=no
TAP_TRUSTED_BEFORE=yes
CASK_PREEXISTING=no
MAN_PAGES_BEFORE=0
BREW_PREFIX=/opt/homebrew
TRAP_ARMED=1
=== STEP 4: install from the real tap (trap is now armed) ===
--- brew tap seanb4t/tap ---
==> Auto-updating Homebrew...
Adjust how often this is run with `$HOMEBREW_AUTO_UPDATE_SECS` or disable with
`$HOMEBREW_NO_AUTO_UPDATE=1`. Hide these hints with `$HOMEBREW_NO_ENV_HINTS=1` (see `man brew`).
==> Auto-updated Homebrew!
==> Updated Homebrew from 4dacfe77a2 to d67b92d2de.
Updated 1 tap (dicklesworthstone/tap).
Error: Failed to import: /opt/homebrew/Library/Taps/hashicorp/homebrew-tap/Formula/vagrant.rb
hashicorp/tap/vagrant: formula requires at least a URL

==> Tapping seanb4t/tap
Cloning into '/opt/homebrew/Library/Taps/seanb4t/homebrew-tap'...
Tapped 1 cask (15 files, 17.7KB).
brew tap exit=0
--- brew trust --tap seanb4t/tap ---
Already trusted tap: seanb4t/tap
brew trust exit=0
--- brew install codegraph ---
==> Would install 1 cask:
seanb4t/tap/codegraph
Warning: You are using macOS 27.
We do not provide support for this pre-release version.
==> Fetching downloads for: seanb4t/tap/codegraph
✔︎ Cask codegraph (0.8.0)
==> Installing Cask codegraph
==> Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'
🍺  codegraph was successfully installed!
brew install exit=0
CASK_VERSION_LINE=codegraph 0.8.0
--- bin symlink ---
lrwxr-xr-x@ 1 sean  admin  48 Aug 11 14:00 /opt/homebrew/bin/codegraph -> /opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
RESOLVED_BIN=/opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
--- Caskroom tree ---
total 0
drwxr-xr-x@  4 sean  admin   128 Aug 11 14:00 .
drwxrwxr-x  90 sean  admin  2880 Aug 11 14:00 ..
drwxr-xr-x@  5 sean  admin   160 Aug 11 14:00 .metadata
drwxr-xr-x@  7 sean  admin   224 Aug 11 14:00 0.8.0
--- Caskroom versioned subdir (best-effort dirname of resolved bin) ---
total 123608
drwxr-xr-x@ 7 sean  admin       224 Aug 11 14:00 .
drwxr-xr-x@ 4 sean  admin       128 Aug 11 14:00 ..
-rw-r--r--@ 1 sean  admin       168 Aug 11 14:00 .codegraph-brew-install
-rw-r--r--@ 1 sean  admin     30933 Aug 10 15:22 CHANGELOG.md
-rwxr-xr-x@ 1 sean  admin  63231081 Aug 10 15:25 codegraph
-rw-r--r--@ 1 sean  admin      1068 Aug 10 15:22 LICENSE
-rw-r--r--@ 1 sean  admin      8387 Aug 10 15:22 README.md
--- receipt metadata dir ---
total 16
drwxr-xr-x@ 5 sean  admin   160 Aug 11 14:00 .
drwxr-xr-x@ 4 sean  admin   128 Aug 11 14:00 ..
drwxr-xr-x@ 3 sean  admin    96 Aug 11 14:00 0.8.0
-rw-r--r--@ 1 sean  admin   860 Aug 11 14:00 config.json
-rw-r--r--@ 1 sean  admin  1065 Aug 11 14:00 INSTALL_RECEIPT.json
/opt/homebrew/Caskroom/codegraph/.metadata/INSTALL_RECEIPT.json
=== STEP 5: Leg 1 — run the harness against the real tree, no mutation ===
harness exit=0
--- harness log contents ---
RUN_ID=run-20260811T175959Z-35769
=== RUN   TestDetectBrewManaged_RealInstall
    brew_test.go:348: resolved binary: /opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
    brew_test.go:349: install dir: /opt/homebrew/Caskroom/codegraph/0.8.0
    brew_test.go:350: tree: Caskroom
    brew_test.go:351: token: codegraph
    brew_test.go:352: version: 0.8.0
    brew_test.go:353: receipt: /opt/homebrew/Caskroom/codegraph/.metadata/INSTALL_RECEIPT.json
--- PASS: TestDetectBrewManaged_RealInstall (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.164s
=== STEP 6: build this tree's binary and preserve the original payload ===
go build exit=0

--- built binary version --json ---
{"version":"dev","commit":"unknown","date":"unknown","go_version":"go1.26.5","os":"darwin","arch":"arm64"}
BUILT_BINARY_SHA256=369162973d3cb74ed82f3dc6f5e6da39905eac552192a6c95f32bdc4cc19020c
PAYLOAD_SHA256_BEFORE=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
preserve flag SET
=== STEP 7: copy freshly built binary over the resolved Caskroom payload path ===
payload substituted at /opt/homebrew/Caskroom/codegraph/0.8.0/codegraph
CURRENT_PAYLOAD_SHA (should equal built binary sha)=369162973d3cb74ed82f3dc6f5e6da39905eac552192a6c95f32bdc4cc19020c
=== STEP 8: run the four observations (leg 2) ===
--- $(brew --prefix)/bin/codegraph upgrade ---
upgrade: codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
UPGRADE_EXIT=1
--- $(brew --prefix)/bin/codegraph upgrade --check ---
codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
UPGRADE_CHECK_EXIT=0
--- $(brew --prefix)/bin/codegraph upgrade --force ---
upgrade: codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph
UPGRADE_FORCE_EXIT=1
--- $(brew --prefix)/bin/codegraph upgrade --help ---
Download the target-platform binary from GitHub Releases, verify its
cosign-keyless signature/provenance in-process (never a cosign CLI), and
only then atomically replace the running binary. --check reports whether
a newer release is available without downloading anything.

A Homebrew-managed install is detected from the resolved location of
the running binary. codegraph upgrade refuses to run there and exits
non-zero, because it was asked for a mutation it declines to perform.
codegraph upgrade --check steps aside with the same pointer and exits
zero, because it only answered a question. Upgrade a Homebrew-managed
install with: brew upgrade codegraph.

Usage:
  codegraph upgrade [version] [flags]

Examples:
  codegraph upgrade --check
  codegraph upgrade
  codegraph upgrade v1.4.0
  codegraph upgrade --check  # brew-managed install: prints the pointer, exits 0

Flags:
      --check   report whether a newer release is available, without downloading
  -f, --force   reinstall even if already on the latest version
  -h, --help    help for upgrade
UPGRADE_HELP_EXIT=0
=== STEP 9: falling off the end of the script; EXIT trap performs restoration ===
=== TRAP FIRED: restore_and_cleanup (invocation 1) ===
PAYLOAD_SHA256_BEFORE=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
PAYLOAD_SHA256_AFTER=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
--- brew uninstall --cask codegraph ---
==> Uninstalling Cask codegraph
==> Unlinking Binary '/opt/homebrew/bin/codegraph'
==> Purging files for version 0.8.0 of Cask codegraph
TAP_TRUSTED_BEFORE=yes — leaving trust grant in place.
--- brew untap seanb4t/tap (TAP_PREEXISTING=no) ---
Untapping seanb4t/tap...
Untapped 1 cask (15 files, 17.7KB).
--- receipt written ---
RUN_ID=run-20260811T175959Z-35769
PAYLOAD_SHA256_BEFORE=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
PAYLOAD_SHA256_AFTER=99763960ddf9411deabbbdee624b8da110b4edf0cc0a295cd26e2435b5b32e62
RESTORE_VERDICT=ok
RESTORE_INVOCATIONS=1
TAP_ACTION=untapped
TRUST_ACTION=left-trusted
--- final state probes ---
final: brew list --cask --versions codegraph:
final: brew tap | rg seanb4t/tap: (none)
final: brew trust --json v1 filtered:     "seanb4t/tap",
final: man page count: 0
final: ls -l bin/codegraph: ls: /opt/homebrew/bin/codegraph: No such file or directory
```

---

*Phase: 04-codegraph-upgrade-homebrew*
*Plan: 04-06*
*Captured: 2026-08-11*
