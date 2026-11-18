# Tasks

## 1. Coverage of the refusal at start (#377)

- [ ] 1.1 Record this map of scenario to test in the PR body. Verify that `go test -count=1 -run 'Refuse|Accept|IndexDirs|Confine|Outside' ./internal/query/... ./internal/cli/... ./internal/mcp/...` passes.
  - A .codegraph link that leads outside: `TestResolveIndexRefusesRootIndexLinkedOutside` (read commands) and the `index` row of `TestIndexCommandsRefuseIndexLinkedOutside` (`sync`, `index`).
  - A store link that leads outside: `TestResolveIndexRefusesRootStoreLinkedOutside` and the `store` row of `TestIndexCommandsRefuseIndexLinkedOutside`.
  - init through a link that leads outside: `TestInitRefusesIndexLinkedOutside`.
  - A dangling link is refused, not skipped: `TestResolveIndexRefusesDanglingIndexLinkOutside`.
  - A subdirectory link to an ancestor's index: `TestResolveIndexRefusesSubdirectoryIndexLinkedToAncestorIndex`.
  - The main worktree's index leads outside: `TestResolveIndexRefusesMainWorktreeIndexLinkedOutside`.
  - serve refuses to start: the new test of task 1.4.
  - A .codegraph link that stays inside: `TestResolveIndexAcceptsIndexLinkedInside` and `TestIndexCommandsAcceptIndexLinkedInside`.
  - A store link that stays inside: `TestResolveIndexAcceptsStoreLinkedInside`.
  - The text of the refusal: `TestIndexDirsRefusesDanglingLink`, extended in task 1.3.
- [x] 1.2 Turn `TestIndexCommandsRefuseStoreLinkedOutside` in `internal/cli/index_confine_test.go` into the table `TestIndexCommandsRefuseIndexLinkedOutside` with two rows: the store linked outside, and `.codegraph` linked outside. Each row runs `sync`, `index --force` and `status`, and asserts the refusal and an empty outside directory. Make `indexDirs` in `internal/cli/init.go` skip `query.IndexDirs`, and verify that the new row then fails.
- [x] 1.3 Extend `TestIndexDirsRefusesDanglingLink` in `internal/query/resolve_confine_test.go`. Also assert that the message contains the quoted resolved root and the whole phrase `remove the link or point it inside that directory`.
- [x] 1.4 Add `TestServeServerPathsRefusesIndexLinkedOutside` in `internal/cli/serve_test.go` for the scenario "serve refuses to start". It links `<repo>/.codegraph` to a temporary directory outside. It asserts that `serveServerPaths` returns an error that matches `query.ErrIndexOutsideRoot`, and that the outside directory is still empty. Make `serveServerPaths` map every error to "not initialized", and verify that the test then fails.

## 2. A watcher checks the index again before its lock and each flush (#375)

- [x] 2.1 Add `IndexChangedError{Was, Now query.IndexPaths}` and the sentinel `ErrIndexChanged` in `internal/daemon` (design D3). Cover the message with the table test `TestIndexChangedErrorText`. Rows:
  - `Dir` changed: the message holds both `Dir` values and `restart the watcher`.
  - only `Store` changed: `Dir` is equal, and the message holds both `Store` values and `restart the watcher`.
  - Each row also asserts `errors.Is(err, ErrIndexChanged)`.
- [x] 2.2 Replace `Daemon.codegraphDir` and `Daemon.storeDir` with `start query.IndexPaths`, and add `(*Daemon).checkIndex() error` (design D1). Verify that `go test -count=1 ./internal/daemon/...` still passes, `TestNewUsesResolvedIndexPaths` included.
- [x] 2.3 Cover `checkIndex` with the table test `TestCheckIndexReportsEveryChange` in `internal/daemon/confine_test.go`. Rows:
  - `.codegraph` linked outside: the error matches `query.ErrIndexOutsideRoot`.
  - the store linked outside: the error matches `query.ErrIndexOutsideRoot`.
  - `.codegraph` moved to an inside link: `*IndexChangedError` with different `Dir` values.
  - only the store moved to `<repo>/.store-real`: `*IndexChangedError` with equal `Dir` values and different `Store` values.
  - `.codegraph` replaced by a link to itself: a non-nil error that matches neither sentinel.
  - nothing changed: `nil`.
- [x] 2.4 In `Daemon.Run`, make the child context with `context.WithCancelCause`, route the watcher-closed branch through `cancel(ErrWatcherClosed)`, and remove `runErr` (design D3). Call `checkIndex` before `acquire`, and in the debounce callback before `flush`. Verify that the existing `ErrWatcherClosed` tests still pass. Cover the new stop with the table test `TestRunStopsWhenIndexChanges`, built on `newFakeDaemon` and `fakeWatcher`. Each row changes the index after the catch-up flush, then sends one event. Rows:
  - `.codegraph` linked outside, the store linked outside, and `.codegraph` linked to itself: `Run` returns the error of `checkIndex`.
  - `.codegraph` moved to an inside link `<repo>/.cg-real`: `Run` returns `*IndexChangedError`.
  - only the store moved to `<repo>/.store-real`: `Run` returns `*IndexChangedError`, and its message holds both store paths.
  - Every row asserts that `syncFn` ran exactly once, for the catch-up flush with the index unchanged. It also asserts that no `.sync-pending` file and no store files appear in the target.
- [x] 2.5 Add the table test `TestRunChecksIndexBeforeLock`. Both rows assert that the bytes of `<outside>/daemon.lock` are unchanged.
  - first call: `<outside>` holds a stale `daemon.lock` (a dead pid), and `<repo>/.codegraph` is linked to `<outside>` after `New`. `RunWithRetry` returns an error that matches `query.ErrIndexOutsideRoot` after one call to `Run`.
  - retry: `<repo>/.codegraph/daemon.lock` names this pid with `selfStartedAt()`, so the first `Run` returns `ErrLockLive`. The `onDeferred` callback links `<repo>/.codegraph` to `<outside>`, which holds a stale `daemon.lock`. `RunWithRetry` returns an error that matches `query.ErrIndexOutsideRoot` from the second `Run`.
- [x] 2.6 Add the positive control `TestRunSyncsWhenIndexUnchanged` for the scenario "The index is unchanged". It leaves `syncFn` nil, so the real `indexer.Sync` runs. It writes a Go file that declares the function `added` and sends one event. After the flush, it asserts that a search through `query.OpenAt` finds `added`, and that `Run` returns `nil` after a cancel. Verify that it passes before and after 2.4.
- [x] 2.7 Add the property `TestIndexDirsIsStableOnAnUnchangedTree` in `internal/query` with `pgregory.net/rapid` (design D6). It builds a random tree of real directories and links under `.codegraph` and `store`, inside and outside the root, and calls `IndexDirs` twice. Assert equal paths, or two errors with the same text.

## 3. A stopping watcher removes only its own lock (#375)

- [x] 3.1 Replace the deferred `release` in `Daemon.Run` with `releaseOwnLock` (design D4). Cover it with the table test `TestRunReleasesOnlyItsOwnLock`. Each row stops `Run` with a cancel. Rows:
  - `<repo>/.codegraph` linked to `<other>/.codegraph`, whose `daemon.lock` names this pid with `selfStartedAt()`: the bytes of that file are unchanged. Only the path check keeps it.
  - the lock in `<repo>/.codegraph` rewritten with another live pid, path unchanged: the file is unchanged. Only the owner check keeps it.
  - only the store linked outside: `<repo>/.codegraph/daemon.lock` is gone.
  - `.codegraph` renamed to `.cg-real` and linked: `<repo>/.cg-real/daemon.lock` is gone.
  - nothing changed: `<repo>/.codegraph/daemon.lock` is gone.

## 4. Caller output

- [x] 4.1 Add the package-level variable `runDaemonWithRetry` in `internal/cli/serve.go`, and have `serveWatchStart` call it (design D5). Add `TestServeWatchStartPrintsIndexChange` for the scenario "serve keeps serving after its watcher stops". It sets the variable to return a refusal error. Assert that stderr has the line `codegraph serve: watcher: ` followed by the error text. Also assert that `watchDone` closes and that the context the test passes in is not cancelled.
- [x] 4.2 Add the package-level variable `runDaemon` in `internal/cli/daemon.go`, and have `daemon start` call it (design D5). Add `TestDaemonStartReturnsIndexChange` for the scenario "daemon start reports the stop". It sets the variable to return a refusal error, runs `daemon start` through `execCmd`, and asserts that the returned error matches `query.ErrIndexOutsideRoot`.
- [x] 4.3 Add the script `test/integration/testdata/index_link_outside.txtar` for the path from a returned error to stderr and the exit status in `cmd/codegraph/main.go`. It links `.codegraph` to a directory outside, runs `! exec codegraph status`, and asserts with `stderr` that the output holds `remove the link or point it inside that directory`. Raise `minScripts` in `test/integration/main_test.go` from 11 to 14, the number of scripts with this one. It was out of date at 11, so 12 does not catch a lost script. Verify with `go test -count=1 -run '^TestScripts/index_link_outside$' ./test/integration`.

## 5. Review and changelog

- [ ] 5.1 Run `mise exec -- task test` and `mise exec -- task lint`. Verify that both pass with no new lint findings.
- [ ] 5.2 Run `/sb:openspec-review index-link-refusal implementation` and resolve every `changes-requested` finding.
- [ ] 5.3 Add the changelog fragment: `task changie -- new --interactive=false -k Fixes -b "A running watcher stops, and writes nothing, when a pull turns .codegraph or its store into a link that leads elsewhere."`

## Workflow follow-up

- The implementation PR says `Closes #375` and `Closes #377`, and carries the `openspec archive index-link-refusal` result.
