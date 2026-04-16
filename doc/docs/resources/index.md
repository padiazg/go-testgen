# Resources

Supplementary techniques and tools that complement go-testgen. Each resource focuses on a specific aspect of the testing workflow and can be adopted independently.

---

## [Test Coverage](test-coverage.md)

Go's built-in coverage tools measure which lines and functions are executed during tests. This resource covers two techniques:

- **HTML report** (`go tool cover -html`) — visual line-by-line coverage in a browser, colour-coded by covered/uncovered status. Fast way to spot untested branches and error paths.
- **Spreadsheet report with GitHub links** — transforms `go tool cover -func` output into a tab-separated table where each row links directly to the function in the repository. Useful for triaging untested functions across a large codebase during a PR review or test sprint.

**When to use it:** after running `go-testgen report` on individual packages, use the coverage script for a full-module audit or to share a triage list with a team.

---

## [Mutation Testing](mutation-test.md)

Coverage tells you which lines ran. Mutation testing tells you whether your tests actually *verify* what those lines do.

A mutation tool ([Gremlins](https://gremlins.dev)) makes small automated changes to the source — flipping a `<` to `<=`, removing a return value, negating a condition — and re-runs the tests. If the tests still pass despite the broken code, the mutant **survived**: your tests cover the line but do not assert its behavior.

This resource covers:

- Installing and running Gremlins
- Understanding the six mutation statuses (`KILLED`, `LIVED`, `NOT COVERED`, `TIMED OUT`, `NOT VIABLE`, `RUNNABLE`)
- Transforming Gremlins output into a spreadsheet with clickable GitHub links
- A workflow that connects surviving mutants directly to `go-testgen gen` and new check functions

**When to use it:** after generating and filling in test scaffolding, run mutation testing to find cases where coverage passes but behavior is under-verified. Surviving mutants are the most precise input for writing new `checkXxxFn` functions.
