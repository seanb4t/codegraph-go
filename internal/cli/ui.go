package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/uiserver"
)

// openBrowser is browser.OpenURL indirected behind a package-level func
// var so ui_test.go can assert launch attempts (or force a failure)
// without actually spawning an OS browser process. Production behavior
// is unchanged.
var openBrowser = browser.OpenURL

// discoverEditorFn is plan 09-02's discoverEditor indirected behind a
// package-level func var, the same test-seam shape as openBrowser
// above: production behavior is unchanged, and ui_test.go can swap in a
// fake to assert the discovered-default path without depending on
// whatever editors happen to be installed on the test machine.
var discoverEditorFn = discoverEditor

// newUiCmd builds `codegraph ui` (SRV-01, SRV-03): a local, read-only web
// UI over the repository's own already-indexed graph. Foreground until
// Ctrl-C, matching `serve` and `daemon start` (D-10) — no PID file, no
// lock file, no `ui stop` verb, and no lifecycle shared with `daemon`
// (SRV-01 forbids sharing it). Registers exactly four flags:
// `--path`/`-p`, `--no-open`, `--editor-url` and `--no-editor-url`
// (D-14/D-16) — and no flag for a bind address, port, hostname, token or
// any other credential — SRV-03 requires neither the bind address nor
// auth be exposed in v1, and D-08 keeps the bind address reachable only
// as the unwired uiserver.Options.Addr field.
//
// D-17: the editor-link default is resolved immediately after
// resolveStartPath and BEFORE the server binds its port below — a
// malformed --editor-url or CODEGRAPH_EDITOR_URL value refuses to start
// before the port is ever bound and before anything is printed.
// CODEGRAPH_EDITOR_URL and CODEGRAPH_NO_EDITOR_URL are the two env vars
// this command reads.
func newUiCmd() *cobra.Command {
	var path string
	var noOpen bool
	var editorURL string
	var noEditorURL bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Run a local, read-only web UI over the repository's own index",
		Long: "codegraph ui starts a local web server over the repository's " +
			"already-indexed graph and opens it in your default browser. " +
			"It is read-only: it never writes to the graph. The server " +
			"binds an ephemeral loopback port and runs in the foreground " +
			"until interrupted (Ctrl-C) — there is no separate `ui stop` " +
			"command and no shared lifecycle with `codegraph daemon`. " +
			"An editor-link default is resolved once at startup, in order: " +
			"--editor-url, then CODEGRAPH_EDITOR_URL, then a discovered " +
			"editor, then unconfigured; --no-editor-url or " +
			"CODEGRAPH_NO_EDITOR_URL=true|1|yes disables it. A malformed " +
			"explicit value refuses to start before the port is bound.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			resolvedEditorLink, err := resolveEditorLink(editorResolveInputs{
				flagTemplate: editorURL,
				noEditorURL:  noEditorURL,
				getenv:       os.Getenv,
				// discover is plan 09-02's real editor discoverer,
				// not a stubbed-out absent source (see discoverEditorFn
				// above).
				discover: discoverEditorFn,
			})
			if err != nil {
				return err
			}

			// The bind happens here: from this point on, the port
			// srv.URL() names is real and connectable. Steps 2 (this
			// call), 3 (printing the URL below) and 4 (launching a
			// browser below) are ordered and the order is load-bearing,
			// not stylistic — with an ephemeral :0 bind there is no port
			// to publish or launch against before Listen returns, and
			// Serve (step 5) blocks, so publication and launch can only
			// happen in the window between them.
			srv, err := uiserver.Listen(uiserver.Options{RepoPath: start, EditorLink: resolvedEditorLink})
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), srv.URL())

			if shouldOpenBrowser(noOpen, cmd) {
				if err := openBrowser(srv.URL()); err != nil {
					// The URL was already printed above, so a launch
					// failure never blocks the user — it is a warning,
					// not a command failure.
					fmt.Fprintf(cmd.ErrOrStderr(), "codegraph ui: could not open a browser automatically: %v\n", err)
				}
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return srv.Serve(ctx)
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "do not open a browser automatically")
	cmd.Flags().StringVar(&editorURL, "editor-url", "",
		"editor URI template with {path}, {line} and {col} placeholders "+
			"(e.g. 'vscode://file/{path}:{line}:{col}'); overrides "+
			"CODEGRAPH_EDITOR_URL and startup editor discovery")
	cmd.Flags().BoolVar(&noEditorURL, "no-editor-url", false,
		"disable editor links entirely (also CODEGRAPH_NO_EDITOR_URL=true|1|yes); "+
			"skips editor discovery")

	return cmd
}

// shouldOpenBrowser implements D-09: the browser opens by default, with
// three independent suppressions — false when --no-open is set, false
// when the CI environment variable is set to any non-empty value, false
// when cmd's configured stdout is not a real character-device terminal —
// so scripted and CI invocations stay safe without depending on the
// caller remembering a flag. The character-device check stats the
// underlying *os.File directly; a non-*os.File stdout (a test's
// in-memory buffer, a piped writer that is not a file at all) is treated
// as not-a-TTY rather than erroring.
func shouldOpenBrowser(noOpen bool, cmd *cobra.Command) bool {
	if noOpen {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}

	f, ok := cmd.OutOrStdout().(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
