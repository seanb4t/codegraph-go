package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
)

// brewReceiptFilename is the file Homebrew writes on a successful install,
// for both the formula (Cellar) and cask (Caskroom) trees. Its existence,
// not its contents, is the detection signal (D-03) — the file's internal
// schema is documented as Homebrew's own private API (RESEARCH.md
// Pitfall 2), so it is only ever stat'd, never parsed.
const brewReceiptFilename = "INSTALL_RECEIPT.json"

// brewInstall describes a Homebrew-managed install located by
// detectBrewManaged. Every field is derived from the resolved filesystem
// path around targetPath — nothing here depends on `brew` being on PATH,
// on any prefix constant, or on the receipt's contents (D-03, D-12).
type brewInstall struct {
	// ResolvedBinary is targetPath after filepath.EvalSymlinks — the real
	// file Homebrew's cask/formula hook staged, not the symlink it may
	// have been invoked through.
	ResolvedBinary string
	// InstallDir is the versioned install directory
	// (<Tree>/<Token>/<Version>), always absolute when ResolvedBinary is
	// absolute. Built only from filepath.Dir ancestors retained during the
	// walk — never by splitting and rejoining path segments, which would
	// silently drop a leading absolute-path separator (review cycle 1,
	// HIGH).
	InstallDir string
	// Tree is "Caskroom" or "Cellar" — the segment name that identified
	// this as a Homebrew-managed tree.
	Tree string
	// Token is the cask token or formula name.
	Token string
	// Version is the version directory name.
	Version string
	// Receipt is the absolute path to the INSTALL_RECEIPT.json that
	// confirmed this is a real Homebrew install, not merely a
	// similarly-named directory (D-03, T-04-01).
	Receipt string
}

// detectBrewManaged reports whether targetPath resolves into a
// Homebrew-managed Caskroom or Cellar tree. Detection is structural and
// prefix-agnostic (D-01, D-03, D-04R): it locates a Caskroom or Cellar
// ancestor directory anywhere above the resolved binary and requires a
// Homebrew-authored INSTALL_RECEIPT.json at the tree-specific location —
// path shape alone is spoofable (T-04-01), so shape without a receipt is
// never treated as detected.
//
// Detection never shells out to `brew`, never touches the network, and
// never returns an error: an unresolvable path (a dangling symlink or a
// symlink loop) or the absence of a receipt at every candidate simply
// reports "not detected" (T-04-03), so a legitimate non-brew upgrade is
// never blocked by this check.
func detectBrewManaged(targetPath string) (brewInstall, bool) {
	resolved, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		// An unresolvable path is not a brew-managed-install signal — it
		// is not-detected immediately, with no fallback to the unresolved
		// path and no continued shape scan (review cycle 1, MEDIUM; the
		// single EvalSymlinks-error contract T-04-03 relies on).
		return brewInstall{}, false
	}

	// Collect the ancestry from the binary's immediate parent up to the
	// filesystem root. Every entry is a value returned by filepath.Dir, so
	// each retains its root/volume prefix by construction — no string is
	// ever split and rejoined (review cycle 1, HIGH).
	var chain []string
	dir := filepath.Dir(resolved)
	for {
		chain = append(chain, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// A candidate exists when some ancestor is literally named Caskroom or
	// Cellar and has at least two descendant components back down to
	// resolved (a token directory, then a version directory) — i.e. the
	// ancestor is at chain index >= 2, with chain[i-1] the token directory
	// and chain[i-2] the version directory.
	for i, treeDir := range chain {
		base := filepath.Base(treeDir)
		if (base != "Caskroom" && base != "Cellar") || i < 2 {
			continue
		}

		tokenDir := chain[i-1]
		versionDir := chain[i-2]

		var receipt string
		if base == "Caskroom" {
			receipt = filepath.Join(tokenDir, ".metadata", brewReceiptFilename)
		} else {
			receipt = filepath.Join(versionDir, brewReceiptFilename)
		}

		if _, statErr := os.Stat(receipt); statErr != nil {
			// No receipt at the location this tree implies — shape alone
			// never suffices (D-03, T-04-01). Keep walking in case a
			// further ancestor is also named Caskroom/Cellar.
			continue
		}

		return brewInstall{
			ResolvedBinary: resolved,
			InstallDir:     versionDir,
			Tree:           base,
			Token:          filepath.Base(tokenDir),
			Version:        filepath.Base(versionDir),
			Receipt:        receipt,
		}, true
	}

	return brewInstall{}, false
}

// brewPointerMessage builds the one sentence both the bare-upgrade refusal
// (D-05, D-07) and the --check step-aside (D-09) print, so the two
// surfaces can never drift apart (must_haves.key_links).
func brewPointerMessage(inst brewInstall) string {
	return fmt.Sprintf("codegraph is managed by Homebrew (%s). Upgrade with: brew upgrade codegraph", inst.InstallDir)
}
