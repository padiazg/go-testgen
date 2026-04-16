# Test Coverage Resources

Useful techniques for measuring and navigating test coverage in Go projects — complementing the scaffolding that go-testgen produces.

---

## Coverage Report with Clickable GitHub Links

### What it does

`go tool cover -func` produces a flat list of every function and its line coverage percentage. The technique below transforms that output into a tab-separated table where the last column is a direct GitHub link to the exact line in the source file.

Paste the result into a spreadsheet (Google Sheets, Excel, LibreOffice Calc) and each GitHub URL becomes a clickable link — useful for triaging untested functions during a PR review or a test sprint.

**Input** (raw `go tool cover` output):

```
github.com/acme/app/internal/core/services/user/service.go:35:   CreateUser   0.0%
github.com/acme/app/internal/core/services/user/service.go:67:   FindByID     72.3%
github.com/acme/app/pkg/utils/parse.go:5:                        ParseInt     100.0%
```

**Output** (tab-separated, ready for a spreadsheet):

```
internal/core/services/user/service.go   CreateUser   0.0%    github.com/acme/app/blob/main/internal/core/services/user/service.go#L35
internal/core/services/user/service.go   FindByID     72.3%   github.com/acme/app/blob/main/internal/core/services/user/service.go#L67
pkg/utils/parse.go                       ParseInt     100.0%  github.com/acme/app/blob/main/pkg/utils/parse.go#L5
```

### Setup

**Step 1 — Export your module path as a regex pattern.**

The value comes from the `module` line in your `go.mod`. Dots (`.`) must be escaped as `\.` and slashes (`/`) as `\/` because the value is used inside a `sed` regular expression.

```bash
# go.mod: module github.com/acme/app
export PROJECT_REPO_URL='github\.com\/acme\/app\/'
```

!!! warning "Regex escaping required"
    `PROJECT_REPO_URL` is **not** a plain URL — it is a regex fragment used inside `sed -E`.
    Forgetting to escape `.` will cause the pattern to match any character, producing wrong output.

**Step 2 — Set your default branch name.**

The generated links point to `blob/<branch>/...`. Adjust if your repo uses `master`:

```bash
export PROJECT_BRANCH='main'   # or 'master'
```

**Step 3 — Run tests with coverage and transform the output.**

```bash
go test -coverprofile=coverage.out -cover ./... > /dev/null 2>&1 && \
go tool cover -func=coverage.out | \
sed -E 's/(\t)+/\t/g' | \
sed -E "s/($PROJECT_REPO_URL)(.*\.go):([0-9]+):(.*)/\2\4\t\1blob\/$PROJECT_BRANCH\/\2#L\3/g"
```

What each step does:

| Step | Command | Purpose |
|------|---------|---------|
| 1 | `go test -coverprofile=coverage.out` | Run tests and write coverage data |
| 2 | `go tool cover -func=coverage.out` | Expand coverage data into per-function lines |
| 3 | `sed -E 's/(\t)+/\t/g'` | Normalise multiple tabs to a single tab |
| 4 | `sed -E "s/($PROJECT_REPO_URL)...` | Strip module prefix, reformat as TSV, append GitHub URL |

**Step 4 — Paste into a spreadsheet.**

Copy the terminal output and paste into Google Sheets or Excel. Tabs become column separators; the last column is a GitHub permalink that opens the exact function in the browser.

### Filtering untested functions only

Pipe through `grep` to show only functions with 0% coverage:

```bash
go test -coverprofile=coverage.out -cover ./... > /dev/null 2>&1 && \
go tool cover -func=coverage.out | \
sed -E 's/(\t)+/\t/g' | \
sed -E "s/($PROJECT_REPO_URL)(.*\.go):([0-9]+):(.*)/\2\4\t\1blob\/$PROJECT_BRANCH\/\2#L\3/g" | \
grep $'\t0\.0%\t'
```

### Using with `go-testgen report`

`go-testgen report` works at the package level and is faster for interactive use. The coverage script above works at the binary level across all packages and is better for a full-project audit or a spreadsheet triage session.

A typical combined workflow:

```bash
# 1. Full audit — identify all zero-coverage functions
go test -coverprofile=coverage.out -cover ./... > /dev/null 2>&1 && \
go tool cover -func=coverage.out | \
sed -E 's/(\t)+/\t/g' | \
sed -E "s/($PROJECT_REPO_URL)(.*\.go):([0-9]+):(.*)/\2\4\t\1blob\/$PROJECT_BRANCH\/\2#L\3/g" | \
grep $'\t0\.0%\t'

# 2. Drill into a specific package
go-testgen report ./internal/core/services/user

# 3. Generate the scaffolding
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# 4. Re-run coverage to confirm improvement
go test -coverprofile=coverage.out -cover ./... > /dev/null 2>&1 && \
go tool cover -func=coverage.out | grep 'service.go'
```

---

## HTML Coverage Report

For visual exploration, Go's built-in HTML report highlights covered and uncovered lines directly in the source:

```bash
go test -coverprofile=coverage.out -cover ./...
go tool cover -html=coverage.out
```

Opens a browser with each file colour-coded: green = covered, red = not covered. Useful for spotting which branches or error paths are missing tests, which then feeds directly into which `gen` commands to run next.

To save the HTML to a file instead of opening a browser:

```bash
go tool cover -html=coverage.out -o coverage.html
```
