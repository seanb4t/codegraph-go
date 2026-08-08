# Feature Research: macOS Gatekeeper Notarization + Homebrew Distribution

**Domain:** macOS distribution features for an open-source Go CLI (self-updating, single static binary)
**Researched:** 2026-08-07
**Confidence:** HIGH overall (every load-bearing claim below is corroborated by Apple's own docs, GoReleaser's own docs/source, or a real shipping project's config/source — LOW-confidence items are called out individually)

---

## 1. Concrete user-facing flows (before / after / degraded)

### 1a. Browser download from GitHub Releases

| State | What the user sees | Source |
|---|---|---|
| **Today** (unsigned raw binary, downloaded via browser → `com.apple.quarantine` set) | Double-click in Finder, or double-click after unzip: **"'codegraph' cannot be opened because the developer cannot be verified."** Only option offered is "Move to Trash" — the user must go to System Settings → Privacy & Security → find the blocked notice → "Open Anyway" (macOS Ventura+), or run `xattr -d com.apple.quarantine`. Running from an **already-open Terminal** (`./codegraph`) can instead be outright **`zsh: killed`** with zero dialog — that's `com.apple.provenance` (attached by macOS 13+ once quarantine clears once, or by some download paths directly), enforced by the kernel, not Gatekeeper's UI layer. Confirmed independently by a real fix in `NJannasch/vibecockpit` (`xattr -c`, not just `-d com.apple.quarantine`, was needed to stop the SIGKILL) and by `donatstudios.com`'s writeup of the same symptom. HIGH confidence. | Apple Developer Forums (r. 706379, 723397), donatstudios.com, `NJannasch/vibecockpit#24` |
| **After notarize (no staple)** | If the Mac is online at first launch: Gatekeeper silently fetches the ticket from Apple, no dialog, binary runs. **This is the actual goreleaser/goreleaser project's own shipped behavior** — its own `.goreleaser.yaml` runs `notarize.macos` (quill) on the raw binary and publishes via `homebrew_casks`, with no `.dmg`/`.pkg`/staple step at all. HIGH confidence (verified against `goreleaser/goreleaser`'s live `.goreleaser.yaml`). | github.com/goreleaser/goreleaser `.goreleaser.yaml` |
| **After notarize, offline machine** | Gatekeeper cannot reach Apple to fetch the ticket → falls back to the same **"cannot be opened because the developer cannot be verified"** dialog as the unsigned case, until the machine is next online. This is Apple's own documented behavior, not a bug: *"If they are online they will download and cache the ticket... If there is no internet connection, this will fail and you will get a nasty Gatekeeper alert."* A **bare Mach-O CLI binary cannot be stapled at all** — Apple's own docs state it plainly: *"tickets are created for standalone binaries, but it's not currently possible to staple tickets to them."* Neither can a `.zip` be stapled directly (only `.dmg`, `.pkg`, and `.app` bundles are staple-able containers). This is the single most load-bearing, unanimous finding across every source consulted (Apple Developer Docs, Apple Developer Forums threads r.651808/r.689337, scriptingosx.com, octet-stream.net, StackOverflow #78850298) — **HIGH confidence, no dissent found anywhere.** | developer.apple.com/documentation/security/customizing-the-notarization-workflow; Apple Developer Forums r.651808 |
| **Finder double-click quirk (separate from all of the above)** | Even a *correctly* signed+notarized CLI binary is unconditionally blocked by Gatekeeper if a user **double-clicks it in Finder** — Finder hands it to Terminal as a "document," which always fails Gatekeeper's document-open check regardless of signing (documented Apple bug, r.58097824, will not be fixed). Running the exact same binary by typing `./codegraph` in an already-open Terminal works fine once notarized. This is irrelevant to `brew install` (brew never launches via Finder) but is directly relevant to the raw-binary/zip download path's UX, since a typical non-technical user's instinct is to double-click. | Apple Developer Forums r.706379, r.689337 |

**Bottom line for Q1a:** signing + notarizing the raw binary (no staple) converts "always broken for browser downloads" into "broken only when offline at first launch" — a real, large improvement, cheaply achieved, and it's exactly what GoReleaser's own release does for itself. Chasing full staple-ability (offline-safe from first launch) requires wrapping in a `.pkg` or `.dmg`, which is a materially bigger lift for a CLI tool that ships no GUI (see §4).

### 1b. `brew tap seanb4t/tap && brew install codegraph`

Once GoReleaser's `homebrew_casks:` block is wired (see §3), the flow is: `brew tap` clones the tap repo; `brew install codegraph` resolves the auto-generated `Casks/codegraph.rb`, downloads the release asset for the running architecture, and (if the cask has no `post_install` xattr-strip hook) marks it `quarantine true` implicitly like any cask — meaning **an unsigned cask-installed binary hits the exact same "developer cannot be verified" dialog as a raw download**, a real, well-documented regression Homebrew introduced when GoReleaser switched from formulas to casks (see §3). Signing + notarizing (this milestone's other deliverable) removes the need for the workaround hack (`xattr -dr com.apple.quarantine` in a `post_install` hook) that unsigned cask taps are forced to ship — this is the direct synergy between the two halves of this milestone. `brew info codegraph` shows the cask's declared version, homepage, and caveats (if any); `brew upgrade` (or bare `brew upgrade`, which sweeps all outdated formulae/casks) pulls the new cask definition GoReleaser pushed on the next tagged release and re-downloads.

### 1c. `codegraph upgrade` under a brew-managed install — what a good refusal looks like

No single canonical Go CLI implementation was found (self-updaters that ship *alongside* a brew formula/cask are the rarer combination — see §2), but the shape of a good refusal is well established by cross-ecosystem precedent:
- **`gh` (GitHub CLI):** doesn't refuse — it *prints* `A new release of gh is available: X → Y` and `To upgrade, run: brew upgrade gh`, and its actual update-notification code is compiled out entirely for brew-built binaries via a build-time ldflag the Homebrew formula sets (`-X main.updaterEnabled=cli/cli`, only present when Homebrew's own formula compiles `gh` from source — the ldflag is homebrew-core's own build recipe, not something `gh` decides at runtime). No accidental Cellar mutation is possible because the self-updater code path plainly isn't linked into a brew build.
- **`lazygit`:** same pattern, inverted framing — the in-app updater is gated behind a `checkForUpdates`-style ldflag that is *absent* by default and only set by the officially-distributed binary release build; packagers (including brew, when the formula builds from source) simply never set it. Maintainer's own words on why: *"Updating mechanism built in is pointless for software being packaged."*
- **`gemini-cli` (Node, not Go, but instructive for the failure mode):** originally used a naive brew-path heuristic that produced **false positives** when both `npm` and `brew` installs coexisted on one machine — it incorrectly told npm users to `brew upgrade` because it detected *a* homebrew path rather than confirming *this* running binary's path resolves inside it. The real fix (PR #14727): resolve `brew --prefix`, then `fs.realpathSync` the running executable, and only claim "brew-managed" if the *resolved* real path is inside that *queried* (not hardcoded) prefix.

**codegraph-go's target shape**, synthesizing the above: `codegraph upgrade` under a brew-managed install should print something like `codegraph was installed via Homebrew — run 'brew upgrade codegraph' instead` and exit non-zero *before* touching the filesystem, using a detection method that is provably robust (see §5), not a guess.

---

## 2. Table stakes vs differentiators vs anti-features

### Gatekeeper / notarization

| Feature | Category | Why | Complexity | Notes |
|---|---|---|---|---|
| Developer ID codesign + `notarytool` submission of the raw binary (GoReleaser's OSS `notarize.macos` block, quill-backed) | **Table stakes** | Every comparable shipping Go CLI that publishes native macOS binaries does this at minimum (`goreleaser/goreleaser` itself, `anchore/syft`/`grype` per the quill author's own blog post). Without it, every browser-downloaded/cask-installed binary hits the "developer cannot be verified" dialog. | LOW–MEDIUM | Cross-platform: quill is a Go library baked into GoReleaser, so this step does **not** require a macOS runner — it can run on the same runner as the linux builds. This directly loosens the "must all run on `macos-latest`" framing in the milestone's key-context; only the darwin *compile* step needs a native macOS toolchain (per the recorded libresolv/zig-cross finding), not the *sign/notarize* step. Needs: Developer ID Application cert (`.p12` + password), App Store Connect API key (issuer ID, key ID, `.p8`) — maintainer already has the Program membership. |
| `spctl -a -vv -t exec` returning `accepted` on the actual shipped artifact, gated in CI, not assumed | **Table stakes** | This is the project's own stated bar and its own standing rule ("a gate is not trusted until demonstrated RED against a confirmed-applied mutation"). Notarization submission succeeding is not proof the binary passes Gatekeeper; only running `spctl`/`syspolicy_check` against a genuinely quarantined copy is. | LOW (once notarization exists) | `syspolicy_check distribution <path>` is the newer, more informative Apple tool for exactly this ("Notary Ticket Missing" etc.) — worth using alongside `spctl`. |
| Stapling (offline-safe from first launch) | **Differentiator, not table stakes** | Real-world precedent is strongly against building this for a CLI: **GoReleaser's own release of itself does not staple** — it ships a signed+notarized raw binary through a brew cask and accepts the online-check fallback. Every source found agrees a bare Mach-O and a bare `.zip` categorically cannot be stapled (see §1a) — the only staple-able containers are `.dmg`, `.pkg`, and `.app` bundles, none of which are the natural shape for a CLI tool. | HIGH if pursued | Would require either (a) GoReleaser **Pro**'s `notarize.macos_native` block (dmg/pkg-only, `macos-latest`-only, licensed feature) or (b) hand-rolling `pkgbuild`/`productbuild`/`productsign`/`xcrun stapler` as custom pipeline steps calling Apple's own CLTs directly (no GoReleaser Pro purchase required for this path — it's just shell commands, well documented by `octet-stream.net`'s and `scriptingosx.com`'s from-scratch guides). Either way this is a second container format (`.pkg`) most Go CLI users neither expect nor want for a tool they intend to put on `$PATH` and drive from scripts. Recommend: **defer**; ship signed+notarized, unstapled, and treat "offline machine, first launch" as an accepted, documented limitation — exactly the goreleaser project's own choice. |
| `.pkg` installer that writes files outside a controlled prefix | **Anti-feature** | See §6. |
| Ad-hoc self-signing tricks (`codesign -s -`) as a *substitute* for real notarization | **Anti-feature** | Only removes the dialog's severity slightly; still triggers Gatekeeper warnings and does nothing to fix the "damaged and can't be opened" failure mode `homebrew/brew`'s own maintainers explicitly call "hacky" and recommend against (PR #20291 discussion). Real signing is not meaningfully harder given the maintainer already holds the certificate. |

### Homebrew tap

| Feature | Category | Why | Complexity | Notes |
|---|---|---|---|---|
| Own tap (`seanb4t/homebrew-tap`), formula/cask auto-published by GoReleaser on every tag | **Table stakes** | Already decided (PROJECT.md). Every named comparable (`k9s`→`derailed/homebrew-k9s`, `lazysql`, `rampart`/`vanity`→`wdm0006/homebrew-tap`, `goreleaser` itself→`goreleaser/homebrew-tap`) does exactly this. | LOW | GoReleaser fully automates this (`repository:`/`token:` blocks); needs one `HOMEBREW_TAP_TOKEN` PAT (fine-grained, `Contents: Read and Write`, scoped to the tap repo only) as a CI secret. |
| **`homebrew_casks:` — not `brews:`** | **Table stakes** (this is a correction to a common assumption, see §3) | GoReleaser deprecated `brews:` (formula generation) in **v2.10** in favor of `homebrew_casks:`. Homebrew's own maintainers, in a 2025-07-22 doc PR discussion (Homebrew/brew#20291), state directly: *"If you have binaries that are precompiled by GoReleaser, the best experience in Homebrew will be with casks."* `goreleaser/goreleaser` dogfoods `homebrew_casks:` on its own tap. | LOW (GoReleaser config swap) | See full analysis in §3. |
| Shell/zsh/fish/bash **completions** + **man pages** installed by the cask | **Differentiator** (cheap, expected polish) | `goreleaser/goreleaser`'s own cask installs `manpages` and `completions` blocks; `homebrew_casks` has first-class `generate_completions_from_executable` support (since GoReleaser v2.15) that can invoke `codegraph completion <shell>` at cask-build time rather than committing static files. | LOW | Only worth it if `codegraph` has a `completion` subcommand (cobra provides this for free per the existing stack). |
| `brew test` block exercising a real command (`system "#{bin}/codegraph", "--version"` style) | **Differentiator** | Every real tap example found (`k9s`, `rampart`) includes a `test:` block; catches a broken cask before users hit it. | LOW | |
| Auto-submitting to `homebrew-core` | **Anti-feature** | See §6. |
| A second, parallel `brews:` (formula) definition kept "just in case" | **Anti-feature** | See §6; also directly contradicts the deprecation — GoReleaser's own docs mark the whole `homebrew_formulas` page "(deprecated)." |

### Self-updating binary × package manager interaction

| Feature | Category | Why | Complexity | Notes |
|---|---|---|---|---|
| `codegraph upgrade` detects a brew-managed install and refuses, pointing at `brew upgrade codegraph` | **Table stakes** | Already decided (PROJECT.md). Precedent: `gh` and `lazygit` both solve this at compile time (see §2 walkthrough and §5) rather than leaving a runtime self-updater free to fight brew for control of the same file. Not doing this produces exactly the failure mode Homebrew's own FAQ documents and works around defensively (*"An app's own updater can make Homebrew's installation record older than the app that is actually installed... blindly replacing the app based on that record could downgrade it"*) — i.e. this is a known, named class of bug, not a hypothetical. | LOW–MEDIUM | Detection approach matters — see §5 for what's robust vs what isn't. |
| `codegraph upgrade` prints the brew-upgrade instruction and exits non-zero, never touching the Cellar | **Table stakes** | Same as above; "never touching the Cellar" is the actual acceptance bar, not just "prints a suggestion." | LOW | |
| Fully disabling upgrade-*checking* (not just upgrade-*applying*) under brew | **Differentiator, situational** | `atuin` (Rust, not Go, but instructive) ships a `update_check` boolean specifically so users can turn off *any* self-network-call, independent of install method — good practice for the "zero passive phone-home" telemetry stance this project already holds (`codegraph telemetry` already documents `upgrade` as the sole network path). Consider allowing `--check` to still run (read-only, no network write) even under brew, since it's informational, while `upgrade` (mutating) refuses. | LOW | |
| Build-time flag baked into the binary by the formula, à la `gh`'s `-X main.updaterEnabled` | **Not directly portable — GoReleaser casks are precompiled, not source builds** | `gh`'s trick only works because homebrew-core *compiles gh from source* per-install; a cask just unpacks a prebuilt tarball, so there is no per-install compile step to inject an ldflag into. The cask-compatible analogue is a **post-install hook sentinel file** (see §5) — functionally equivalent, achievable with `homebrew_casks.hooks.post.install`. | LOW–MEDIUM | |

---

## 3. Formula vs cask — Homebrew's current guidance, and why the milestone's implicit assumption needs correcting

**This is the single biggest correction this research surfaces relative to the milestone's framing** (PROJECT.md and SEED-002 both talk in terms of a generic "formula published by GoReleaser's `brews:` block" — that path is deprecated and actively discouraged for this exact use case).

**Timeline, dated:**
- GoReleaser **v2.10**: `brews:` config marked deprecated; its docs page is now literally titled *"Homebrew Formulas (deprecated)"* and reads *"Users are encouraged to utilize Homebrew Casks as the preferred alternative for distributing software."*
- `homebrew_casks:` was added in GoReleaser PR #5780 ("feat: homebrew casks", caarlos0) specifically to be the pre-compiled-binary path; the PR's own description frames it as generating "casks instead of formulae."
- **Homebrew's own documentation PR #20291** (merged 2025-07-22, `gibfahn`) exists specifically to codify this after GoReleaser's switch caused confusion. Direct quotes from Homebrew maintainers in that thread (HIGH confidence — this is Homebrew's own team, not a third party):
  - *"If you have binaries that are precompiled by GoReleaser, the best experience in Homebrew will be with casks."*
  - *"For 3rd-party taps, it's the tap owner's call"* — i.e. nothing stops a formula from working, but it's not the recommended/idiomatic path any more.
  - *"Cask binaries are quarantined, Formula binaries are temp signed by homebrew... this breaks many taps... [or] causes people to run xattr in a post-install task. This is pretty hacky."* — Homebrew's rebuttal: *"Disabling a macOS security feature should feel 'pretty hacky.' People should sign these binaries to avoid these hacks."* — i.e. the correct fix for the cask-quarantine regression is signing, which is exactly what this milestone's notarization half delivers.
- **Real, dated migration precedent**: `dash0hq/dash0-cli` migrated its own tap from formula to cask in June 2026, citing this exact GoReleaser deprecation as the reason, alongside an unrelated Homebrew 6.0 (2026-06-11) tap-trust change.

**What formula-from-tarball vs cask actually look like** (both auto-generated by GoReleaser, both confirmed against real, current examples):

- **Formula (deprecated path)** — a Ruby class (`Formula`) with `on_macos`/`on_linux` blocks pointing at release URLs + SHA256, an `install` block that's effectively just `bin.install "codegraph"`. Real examples still on this path: `derailed/homebrew-k9s` (predates the deprecation), `jorgerojas26/homebrew-lazysql`, `wdm0006/homebrew-tap`. These exist because they were built before GoReleaser v2.10, not because formula is currently the recommended choice for a precompiled Go binary.
- **Cask (current path)** — a Ruby `Cask` block (`url`, `sha256`, `binary "codegraph"` or similar artifact stanza) that Homebrew treats as "pre-compiled binary built and signed by upstream." Real, dated (2026) example doing exactly this milestone's shape — Go CLI, own tap, unsigned-at-time-of-writing, working around it explicitly: `devenjarvis/lathe`'s `.goreleaser.yaml`, whose own inline comment reads: *"Casks are GoReleaser's supported path for distributing pre-built binaries (the deprecated `brews:`/formula path is meant for build-from-source)."* — and its `hooks.post.install` strips quarantine because the binary is unsigned. **`goreleaser/goreleaser`'s own tap** is the highest-confidence example of all: it runs `notarize.macos` *and* publishes via `homebrew_casks` together — i.e. it is the concrete, already-shipping instance of exactly this milestone's target shape.

**Recommendation for codegraph-go: `homebrew_casks:`, not `brews:`.** Given the binary will be Developer-ID-signed and notarized (this milestone's other half), the cask does **not** need the `xattr -dr com.apple.quarantine` post-install hack that unsigned taps like `lathe`'s are forced to carry — signing removes the reason casks have a worse reputation than formulas on this specific point. This closes the loop the maintainer's own key decision already opened ("a bare Mach-O can be notarized but not stapled... brew conventionally wants a container") with a materially better answer than was assumed: the "container" a cask wants is just the same `.zip`/`.tar.gz` archive already being produced for the raw-binary contract (D-02), not a `.pkg`/`.dmg`.

---

## 4. Container format for the browser-download path

| Format | Staplable? | What users expect for a CLI | Pipeline cost | Verdict |
|---|---|---|---|---|
| **Raw binary** (current, unchanged per D-02/Finding 1) | No — bare Mach-O binaries categorically cannot be stapled (§1a) | Fine for scripted/`upgrade`-path consumers; a browser user gets no README/LICENSE and a bare executable named oddly (`codegraph_v0.5.0_darwin_arm64`) | Zero — already shipping | Keep publishing unchanged; this is `internal/upgrade`'s contract, not the human-download UX. |
| **`.zip`** | **No** (confirmed unanimously — Apple's own docs: *"While you can notarize a ZIP archive, you can't staple to it directly"*; multiple independent Apple Forums threads corroborate for CLI-specifically, not just app bundles) | This is the de facto standard for CLI tool archives on macOS/Windows (GoReleaser's own default `format_overrides` pattern uses `zip` for Windows, `tar.gz` for Unix — but macOS commonly gets `zip` too when a human-facing archive is wanted, since macOS's Archive Utility handles `.zip` natively without a terminal) | LOW — this is exactly the `archives:` block already dead-configured in `.goreleaser.yaml` and about to go live under `goreleaser release` | **Recommended** for the browser-download archive. Gets the README/LICENSE/completions bundling humans expect; accept that it can only be notarized, not stapled — same tradeoff GoReleaser's own project accepts. |
| **`.tar.gz`** | No (same underlying constraint as zip — not a staple-able container) | Standard for Unix CLI tools generally (this is GoReleaser's own project-wide default), but on macOS a `.tar.gz` from a browser needs `tar` — normal for a dev audience, mildly worse than `.zip`'s native double-click-to-extract for less technical users | LOW — same block as zip | Reasonable alternative to zip; either is fine given neither staples. Pick one, don't ship both, to avoid asset-count bloat this milestone already accepts is doubling (raw + archive). |
| **`.dmg`** | **Yes** | Not a CLI convention at all — DMGs signal "drag this .app to Applications," which codegraph is not; every source on notarizing CLI tools specifically (scriptingosx.com, octet-stream.net, the `#78850298` StackOverflow thread) converges on **`.pkg`, not `.dmg`**, as the correct staple-able container *when* a CLI project decides staple-ability is worth the cost | MEDIUM–HIGH — requires GoReleaser Pro's `notarize.macos_native` (dmg-only by default) or hand-building via `hdiutil`; also needs a `create-dmg`-style layout step that makes no sense without a `.app` | **Not recommended** — wrong metaphor for this product, real added cost, no comparable CLI in this research used it. |
| **`.pkg`** | **Yes** | The correct staple-able container *if* stapling is pursued — installs to `/usr/local/bin` (or a configurable prefix) via `pkgbuild`/`productbuild`, works from Finder double-click with no Gatekeeper document-open bug (unlike a bare binary — see §1a's Finder-quirk note) | MEDIUM — GoReleaser Pro's `notarize.macos_native` (`use: pkg`) exists, or a hand-rolled `pkgbuild → productsign → notarytool submit --wait → xcrun stapler staple` sequence (well documented, no Pro license needed) | **Deferred, not this milestone** — this is the "differentiator, not table stakes" call from §2. Revisit only if the online-Gatekeeper-check limitation proves to matter in practice (e.g., CI runners, air-gapped machines) — track as a documented, accepted limitation for now, exactly matching this repo's stated practice of naming tradeoffs instead of hiding them. |

**Recommendation:** archives alongside raw binaries as `.zip` (macOS + Windows-style convention, human-friendly, no extra tooling); do not build `.dmg` or `.pkg` this milestone; document the offline-first-launch limitation explicitly rather than silently accepting it (matches this repo's existing "gate is not trusted until demonstrated RED" ethos — the limitation should be demonstrated and written down, not merely assumed away).

---

## 5. Detecting a brew-managed install from inside the binary

| Approach | Robust? | Why / why not | Real precedent |
|---|---|---|---|
| **Hardcoded path-prefix guess** (e.g. `strings.HasPrefix(exePath, "/opt/homebrew/")` or `"/usr/local/Cellar/"`) | **No — this is exactly what the project has already decided to avoid** | Two hardcoded prefixes (`/opt/homebrew` Apple Silicon, `/usr/local` Intel, plus `/home/linuxbrew/.linuxbrew` on Linux) is already fragile; a symlink-following user, a non-default `--prefix` install (Homebrew's own FAQ explicitly warns against these but they exist), or simply comparing against the wrong constant produces silent false negatives/positives. `koizuka/why`'s own detection table literally labels this method "Cellar path pattern" — i.e. it's a known, named, low-rigor technique, not a robust one. | `koizuka/why` (openly a heuristic tool, not a self-updater's gate) |
| **Naive brew-path substring check without symlink resolution** | **No — proven to false-positive in production** | `google-gemini/gemini-cli` shipped exactly this and had to fix it (PR #14727) after it told npm-installed users to run `brew upgrade` because *some* homebrew path existed on the machine, regardless of whether the *running* binary actually came from it. | `google-gemini/gemini-cli#14727` (real, merged fix — the "before" state is the anti-pattern) |
| **Query `brew --prefix` (or `brew --cellar`) live, then resolve the running executable's real path and check it falls inside that *queried* prefix** | **Yes — this is the corrected, shipping approach** | This is literally the fix gemini-cli landed: *"uses `brew --prefix` and `fs.realpathSync` to verify the actual installation path... verifying that the running script's path is actually located within the Homebrew installation directory."* In Go: `os.Executable()` → `filepath.EvalSymlinks()` → compare against `exec.Command("brew", "--prefix", "codegraph").Output()` (or `brew --cellar`), not a compiled-in constant. Degrades safely if `brew` isn't on `PATH` (treat as "not brew-managed," never guess yes). | `google-gemini/gemini-cli#14727` (post-fix code) |
| **Post-install hook writes a sentinel file next to the binary** (cask-compatible analogue of `gh`'s compiled-in ldflag) | **Yes — the most robust option, and directly buildable with GoReleaser's own `homebrew_casks.hooks.post.install`** | Since a cask is a precompiled artifact (no per-install build step), there's no ldflag-injection point the way homebrew-core's *source* build of `gh` has. But `homebrew_casks` hooks run arbitrary Ruby (`system_command`) at install time — exactly what `dash0`/`lathe`'s existing quarantine-strip hooks already prove works in practice — so the same hook mechanism can instead (or additionally) write a marker file (`.brew-managed` sentinel, or write a value into a fixed relative path) that `codegraph upgrade` checks for at runtime. This has zero ambiguity: the file's mere existence, written by Homebrew itself at install time, *is* the ground truth, no guessing about paths at all. | Mechanism proven by `dash0hq`/`devenjarvis/lathe`'s existing GoReleaser `hooks.post.install` blocks (different payload, same mechanism) |
| **Compiled-in ldflag set by the formula's own build recipe** (`gh`, `lazygit`'s pattern) | **Robust, but only applies to source-built formulas, not precompiled casks** | Doesn't transfer directly to codegraph-go's cask-based plan (§3) since there's no per-install `go build` step for Homebrew to inject an `-X` flag into. Included for completeness / to explain why `gh`'s well-known trick isn't directly reusable here. | `Homebrew/homebrew-core` `gh.rb`, `jesseduffield/lazygit` `checkForUpdates` ldflag |

**Recommendation:** combine the two robust options — `brew --prefix`/`EvalSymlinks` as the primary, always-available check (works even if the sentinel-file mechanism is added later or the user manually relocated a cask install), plus a sentinel file written by the cask's `post_install` hook as a secondary, unambiguous signal. Either one alone is defensible; together they cover the "must be tested against a real brew-managed layout, not a path-prefix guess" bar the project has already set for itself, with the sentinel file being the part that can be asserted in a test without actually shelling out to `brew`.

---

## 6. What NOT to build (anti-features)

| Anti-feature | Why it looks appealing | Why it's a problem here | Do instead |
|---|---|---|---|
| Auto-submitting the formula/cask to `homebrew-core` | "More official," wider discoverability without users needing to `brew tap` first | `homebrew-core` requires build-from-source (a *formula*, not the cask this milestone is building), an external review queue with notability criteria the maintainer doesn't control the timeline of, and — per Homebrew's own maintainers in the PR#20291 thread — precompiled GoReleaser binaries specifically *are not* what `homebrew-core` wants (*"never going to happen"* for GoReleaser-style precompiled binaries into core). Already correctly rejected in PROJECT.md's Key Decisions. | Own tap now; revisit `homebrew-core` only once adoption independently justifies the review-queue cost, and only as a from-source formula, not a repackaged cask. |
| A `.pkg` installer that writes files outside Homebrew's own managed prefix, or outside a location the user chose | Feels "more like a real installer," matches what non-technical users may expect from other macOS software | Directly conflicts with `codegraph upgrade`'s own atomic-swap contract and with Homebrew's entire model (kegs, opt-prefix symlinks, uninstallability) — a `.pkg` that scatters files means neither `brew uninstall` nor `codegraph uninstall` can find everything it touched. This is exactly the shape of gate this repo's own rules explicitly warn against inventing structure for. | If a `.pkg` is ever built (deferred per §4), keep its install root scoped to a single relocatable prefix (e.g. `/usr/local/codegraph` or matching brew's own `opt_prefix` convention), never scatter into arbitrary system locations. |
| A GUI installer / app-bundle wrapper (`.app`, drag-to-Applications) | "Feels native," works around the Finder-double-click Gatekeeper document-open bug (§1a) | codegraph is not a GUI application and has no menu-bar/dock presence to launch from; every comparable tool in this research (`gh`, `k9s`, `lazygit`, `fzf`, `goreleaser` itself) ships as a bare CLI binary with no `.app` wrapper. Building one would be pure surface area with no corresponding user value — this product's entire audience invokes it from a terminal or as an MCP server subprocess. | None needed — the Finder-double-click bug is irrelevant to `brew install` and to script/`curl`-driven downloads; only matters for a browser user who double-clicks a raw binary, which the `.zip` archive with a README already discourages. |
| A self-updater that silently mutates the Cellar (i.e. `codegraph upgrade` swapping the binary in place even when brew-managed, without refusing) | Feels friendlier ("just works" regardless of install method) | This is the exact failure mode Homebrew's own FAQ documents as a known problem class (*"An app's own updater can make Homebrew's installation record older than the app that is actually installed... blindly replacing the app based on that record could downgrade it"*), and it's the specific mistake `gh` and `lazygit` both engineered around at build time. A user's next `brew upgrade` (or `brew bundle`, or a colleague following brew-based setup instructions) would silently disagree with what's actually on disk. Already correctly rejected in PROJECT.md's Key Decisions. | Detect (§5) and refuse with a clear pointer to `brew upgrade codegraph`, as already decided. |
| Keeping a parallel `brews:` (formula) definition "just in case," alongside the new `homebrew_casks:` | Hedges against the cask choice being wrong; "more options for users" | GoReleaser's own docs mark the formula path deprecated as of v2.10; shipping both means two install paths that can silently diverge (a user with the old formula tapped won't automatically get cask updates, and vice versa) — this is precisely the confusion Homebrew's own doc-clarification PR (#20291) exists to head off, and precisely what `dash0`'s real migration guide had to write a whole doc page to unwind for its users (`brew uninstall` the formula, `brew untap`, then reinstall as a qualified cask — "not automatic"). | Ship `homebrew_casks:` only; if the project ever *did* have an old formula tap in the wild, use Homebrew's own `tap_migrations.json` mechanism to redirect users automatically rather than running both indefinitely. (Not applicable here — codegraph-go has never published a Homebrew formula before, so there's no legacy migration to manage.) |
| Building GoReleaser Pro's `notarize.macos_native` / `dmg`/`pkg` support purely to get stapling | Feels like "doing it properly," closes the offline-first-launch gap completely | Real cost: either a paid GoReleaser Pro license plus a `.dmg`/`.pkg` build step, or a meaningful hand-rolled `pkgbuild`/`productbuild`/stapler pipeline — for a gap that GoReleaser's own project (dogfooding this exact milestone's shape) has decided is not worth closing for a CLI tool. | Ship signed+notarized+unstapled; document the offline-first-launch limitation; revisit only on real evidence it matters (matches this repo's stated evidence-over-assumption practice). |

---

## Feature Dependencies

```
[goreleaser release migration]  (enabling change — already scoped as its own milestone item)
    └──requires──> [notarize.macos block wired + Developer ID cert + ASC API key secrets]
                       └──enables──> [archives: block goes live (.zip)]
                                         └──requires──> [homebrew_casks: block, pointing url_template at the .zip archive asset]
                                                            └──enables──> [brew tap seanb4t/tap && brew install codegraph works]

[Developer ID signing + notarization]  ──removes the need for──> [xattr -dr com.apple.quarantine post_install hook that unsigned casks require]

[codegraph upgrade brew-managed detection]  ──requires──> [homebrew_casks.hooks.post.install sentinel-file write] (secondary signal)
                                             ──requires──> [brew --prefix / EvalSymlinks check in internal/upgrade] (primary signal)
                                             ──conflicts with──> [any self-updater path that swaps the binary unconditionally]

[Stapling / .pkg or .dmg container]  ──deferred, not required by── [any table-stakes item above]
```

### Dependency Notes

- **`goreleaser release` requires notarization to be wired before `homebrew_casks` is worth publishing without the xattr hack**: technically GoReleaser will happily publish an unsigned cask (with the workaround hook), but doing so ships the exact "hacky" pattern Homebrew's own maintainers criticize — sequence notarization first, or accept the temporary workaround hook as an explicit interim state if phasing requires it.
- **The `.zip` archive is a genuine shared dependency of both the browser-download UX (§4) and the cask's `url_template` (§3)** — GoReleaser's cask points at a release asset URL, so whatever container format is chosen for humans is also what the cask downloads; this is why §3 and §4's recommendations converge on the same `.zip`.
- **Stapling conflicts with "ship this milestone"**: it's the one item in this research with a real, well-evidenced complexity/value mismatch for a CLI tool — every other item above is either already decided, cheap, or has a direct, cited precedent from a comparable shipping project.

---

## MVP Definition

### Launch With (v0.5.0)

- [ ] `notarize.macos` (quill, OSS GoReleaser) wired for both darwin build IDs — table stakes, cheap, cross-platform-runnable
- [ ] `spctl -a -vv -t exec` / `syspolicy_check` proven `accepted` against a genuinely quarantined artifact in CI, not assumed from a green notarization submission
- [ ] `archives:` block live under `goreleaser release`, producing `.zip` for darwin (and likely all platforms, for consistency) alongside the unchanged raw binaries
- [ ] `homebrew_casks:` (not `brews:`) block, own tap, pointing at the `.zip` archive
- [ ] `codegraph upgrade` brew-managed detection (`brew --prefix` + `EvalSymlinks`, optionally + post-install sentinel file) and refusal, tested against a real brew-managed layout

### Add After Validation (v0.5.x)

- [ ] Shell completions + man pages generated into the cask (`generate_completions_from_executable`) — cheap polish, no reason to rush it into the first cut if it adds review surface
- [ ] `brew test` block exercising `codegraph --version` or similar

### Future Consideration (later milestone, on evidence only)

- [ ] `.pkg`-based stapled distribution — only if the offline-first-launch limitation is reported as a real problem by real users, not preemptively
- [ ] `homebrew-core` submission — only once adoption independently justifies the external review-queue cost, and as a from-source formula, not the cask this milestone builds

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---|---|---|---|
| Sign + notarize raw binary (unstapled) | HIGH | LOW | P1 |
| `homebrew_casks` tap (not `brews`) | HIGH | LOW | P1 |
| `codegraph upgrade` brew-detection + refusal | HIGH | LOW–MEDIUM | P1 |
| `.zip` archive alongside raw binaries | MEDIUM | LOW | P1 |
| Completions/man pages in cask | LOW–MEDIUM | LOW | P2 |
| `brew test` block | LOW | LOW | P2 |
| `.pkg` + full stapling | LOW (evidence-free) | HIGH | P3 |
| `homebrew-core` submission | LOW (this early) | HIGH (external process) | P3 |

---

## Named Real-World Example Analysis

| Tool | Notarizes? | Formula or cask? | Ships own self-updater alongside brew? | How it reconciles the conflict |
|---|---|---|---|---|
| **goreleaser/goreleaser** | Yes — `notarize.macos` (quill), unstapled | **Cask** (`homebrew_casks`, own tap `goreleaser/homebrew-tap`) | No (it's a build tool, not a self-updating CLI) | N/A — but is the single best template for this milestone's sign+notarize+cask combo, since it's the same tool dogfooding itself |
| **gh (GitHub CLI)** | Ships signed/notarized macOS binaries via its own release process (separate from the brew path) | **Formula**, homebrew-core, **built from source** (not a GoReleaser precompiled binary) | Yes | Self-updater's code path is compiled out entirely via a build-time ldflag (`-X main.updaterEnabled=cli/cli`) that only homebrew-core's own build recipe sets — brew-built `gh` never has the updater linked in at all |
| **lazygit** | Not verified in this research (no GoReleaser notarize block found in scope of what was checked) | Both exist in the wild via various taps/homebrew-core-adjacent channels; upstream itself just publishes GitHub release archives | Yes, but gated | In-app updater requires a `buildBinary`/`checkForUpdates`-style ldflag present only in the officially-distributed release build; disabled on Windows entirely; maintainer's own stated rationale: *"Updating mechanism built in is pointless for software being packaged"* |
| **k9s** | Not verified in scope | **Formula** (`derailed/homebrew-k9s`) — predates GoReleaser's cask deprecation | No self-updater found | N/A |
| **atuin** (Rust, included for the self-update-vs-package-manager pattern even though off-ecosystem) | N/A (not this research's ecosystem) | brew formula (via its own installer/package-manager-first philosophy) | Notification-only (`update_check`), never self-replaces | Explicitly does **not** implement a self-replacing updater specifically *because* of package-manager conflicts — direct quote from Atuin's own community forum: *"We tried for a really long time to try and use the system package manager exclusively... something like `atuin update` can't work"* when a package manager owns the install. Strongest cross-ecosystem validation that self-replace-under-a-package-manager is a real, well-known trap, not a theoretical one. |
| **fzf, jj (jujutsu)** | Not verified in scope of this research pass | Both ship via `homebrew-core` formulas | fzf: no persistent self-updater found in scope; jj: no persistent self-updater found in scope | Neither was found to combine a self-updater with brew — consistent with the broader pattern that **shipping both is genuinely rare**, and every case that does exists solves it at build/compile time, never at runtime with a path-prefix guess |
| **dash0hq/dash0-cli** | Not yet, at time of its formula→cask migration (explicitly noted as a gap) | Migrated formula → **cask** in 2026-06, citing GoReleaser's deprecation directly | Not verified in scope | Real, dated (June 2026), well-documented precedent for exactly the formula-to-cask move this milestone should make from the start (codegraph-go has the advantage of never having shipped a formula, so there's no migration to manage) |
| **devenjarvis/lathe** | No (unsigned at time of research) | **Cask**, own tap, with explicit `.goreleaser.yaml` comment citing the same deprecation | Not verified in scope | Ships the `xattr -dr com.apple.quarantine` post-install workaround this milestone's signing avoids needing |

---

## Sources

**HIGH confidence (official docs / primary shipping source read directly):**
- `developer.apple.com/documentation/security/customizing-the-notarization-workflow` and `.../notarizing-macos-software-before-distribution` — staple-ability constraints (zip/binary cannot be stapled; dmg/pkg/app can)
- `goreleaser.com/customization/sign/notarize/` (Context7 + direct fetch, checked 2026-08-07) — `notarize.macos` (quill, OSS, cross-platform) vs `notarize.macos_native` (Pro-only, dmg/pkg, macOS-runner-only)
- `goreleaser.com/customization/publish/homebrew_casks/` and `.../homebrew_formulas/` (Context7 + direct fetch) — cask is current path, `brews:` marked deprecated since v2.10
- `github.com/goreleaser/goreleaser/.goreleaser.yaml` (live, main branch) — the project's own dogfooded notarize+cask configuration, no staple step
- `github.com/Homebrew/brew/pull/20291` — Homebrew maintainers' own direct statements on formula-vs-cask for precompiled binaries
- `docs.brew.sh/Formula-Cookbook`, `docs.brew.sh/FAQ`, `docs.brew.sh/Manpage` — Cellar/prefix/keg terminology, self-updating-app upgrade-skip behavior
- `github.com/goreleaser/goreleaser/pull/5780` — the PR that introduced `homebrew_casks` and deprecated `brews`

**HIGH confidence (real, named shipping project source/config read directly):**
- `github.com/devenjarvis/lathe/.goreleaser.yaml` — cask + quarantine-strip hook, with explanatory inline comments
- `github.com/cli/cli` (issues #6949, #10242, #2141; PR #70784, #4247; `docs/install_macos.md`) — gh's homebrew-core formula, build-time updater-disable ldflag
- `github.com/jesseduffield/lazygit` (`pkg/updates/updates.go`, `pkg/config/user_config.go`, PR #189) — build-time updater-gate ldflag, explicit packaging rationale
- `github.com/google-gemini/gemini-cli` PR #14727 — real false-positive brew-detection bug and its fix (`brew --prefix` + `realpathSync`)
- `docs.atuin.sh`, `forum.atuin.sh` — explicit rejection of self-replacing updater under a package manager
- `deepwiki.com/derailed/k9s/7.4-release-and-packaging` — k9s's formula-based tap (predates cask deprecation)
- `www.dash0.com/docs/dash0/miscellaneous/tooling/dash0-cli/brew-tap-migration-2026-06` — dated, real formula→cask migration citing the same deprecation

**MEDIUM confidence (community writeups, cross-checked against 2+ independent sources each):**
- `donatstudios.com/mac-terminal-run-unsigned-binaries`, `ctxloom.dev/getting-started/binary-trust`, `github.com/NJannasch/vibecockpit/pull/24` — concrete "cannot be opened"/SIGKILL user-facing symptoms for unsigned CLI downloads, cross-checked against each other and against Apple Forums threads
- `octet-stream.net/b/scb/guide-to-signing-notarising-single-cli-binary-mac.html`, `scriptingosx.com/2019/09/notarize-a-command-line-tool` — hand-rolled `.pkg` staple pipeline (no GoReleaser Pro required), cross-checked against Apple's own docs and Apple Developer Forums threads r.651808/r.689337
- Apple Developer Forums threads (r.706379, r.689337, r.651808, r.723397) — Finder-double-click Gatekeeper bug, `com.apple.provenance` behavior; forum posts, not official docs, but directly from Apple DTS engineers and internally consistent across threads
- `mcginniscommawill.com`, `bindplane.com`, `dev.to/40percentironman`, `engineered.at` — GoReleaser+Homebrew tap setup walkthroughs, used only for confirming mechanical config shape (token setup, workflow YAML), not for the formula-vs-cask judgment call (which rests on the HIGH-confidence sources above)
- `koizuka/why` — cited only as a *negative* example (a tool that intentionally does naive Cellar-path pattern matching, for a different, lower-stakes use case than a self-updater's gate)
