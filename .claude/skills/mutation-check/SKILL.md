---
name: mutation-check
description: Prove a new or changed Go test catches its bug, by running it against mutants of the code.
---

# Mutation check

A test proves a fix only if it fails without the fix. Use this skill when an
OpenSpec task says "check that each new test can fail", when a fix PR claims
"I watched it fail before the fix", or when a review asks whether a guard can
fire.

## Steps

1. For each new or changed test, choose the smallest edit to the code under
   test that brings the bug back. That edit is a mutant. Give each mutant one
   job: a mutant that breaks several things proves little.
2. Write the mutants to a JSON file in your scratch directory, never in the
   repository:

   ```json
   [
     {
       "name": "M1",
       "file": "internal/daemon/stop.go",
       "old": "\t\tif !match(rec) {\n\t\t\tcontinue\n\t\t}\n",
       "new": "",
       "package": "./internal/daemon/",
       "tests": ["TestStopMatching"]
     }
   ]
   ```

   `old` must occur exactly once in `file`. `tests` names top-level tests
   (no `/`) that must catch the mutant; the script anchors them, so no other
   test runs. `new` may be empty, to delete the text.
3. Run `task mutation-check -- <path to the JSON file>`. The script first
   runs the named tests on the unmutated tree; each must report PASS.
4. Read each result:
   - `CAUGHT by <tests>`: the test fails without the fix. Done.
   - `CRASHED`: go test failed without a failing test, for example a timeout
     or a panic. The mutant is caught; read the output to see how.
   - `SURVIVED`: every named test passed. Strengthen the test, or name the
     test that should catch it.
   - `BUILD FAIL`: the mutant does not compile or vet, so it proves nothing.
     Keep the code compiling, for example with `_ = x` for a variable the
     mutant leaves unused.
   - `BAD ANCHOR`: fix `old` so it occurs exactly once, or fix `file`.
   - `BASELINE FAIL`: a named test fails, or does not run, without any
     mutant: a misspelled name, a skip or a build tag. Fix that first.
5. Repeat until the last line reads `N/N caught`. Put that line and the
   mutant list in the PR body.

## Repository notes

- The script uses `go test -overlay`, so the working tree never changes and a
  background `task test`, an editor or a reviewer can read the tree at the
  same time.
- Each mutant runs in its own `go test` call, so `internal/daemon` mutants
  stay isolated, as `task test:daemon` keeps them.
- A test with a `//go:build darwin` tag proves a mutant only on macOS.
- tree-sitter's CGo warnings on stderr are noise.
- The script mutates Go source only. To prove a guard that reads YAML, TOML or
  another data file, break that file in place, run the guard, and restore the
  file from a saved copy.
