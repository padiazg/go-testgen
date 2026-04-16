# Mutation Testing

## What Is Mutation Testing?

Code coverage tells you which lines were executed during tests. Mutation testing answers the harder question: *do your tests actually verify what those lines do?*

A mutation testing tool makes small, targeted changes to the source code — called **mutants** — and re-runs the test suite for each one. If the tests fail, the mutant was **killed** (good). If the tests still pass despite the code being wrong, the mutant **survived** (bad — your tests are not asserting what they appear to assert).

### Why It Matters Alongside Coverage

A function can have 100% line coverage and still allow mutants to survive:

```go
// Original
func IsAdmin(role string) bool {
    return role == "admin"
}

// Mutant: operator changed from == to !=
func IsAdmin(role string) bool {
    return role != "admin"  // ← mutant
}
```

If your test only calls `IsAdmin("admin")` and checks that it returns `true` — but never calls it with a non-admin value — the mutant survives. The line is covered; the behavior is not verified.

Mutation testing surfaces exactly these gaps. Surviving mutants are the most valuable input to go-testgen: they tell you precisely which functions need stronger test cases.

---

## Tool: Gremlins

[Gremlins](https://gremlins.dev) is a mutation testing tool for Go. It integrates with the standard `go test` toolchain and requires no changes to your code or test files.

### Installation

```bash
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
```

Verify:

```bash
gremlins --version
```

### Basic Usage

```bash
# Run mutation testing on the whole module
gremlins unleash

# Run on a specific package
gremlins unleash --package ./internal/core/services/user
```

Gremlins outputs one line per mutant:

```
KILLED       ConditionalsBoundary  at internal/core/services/user/service.go:45:12
LIVED        ConditionalsBoundary  at internal/core/services/user/service.go:67:8
NOT COVERED  ArithmeticBase        at pkg/utils/parse.go:5:4
TIMED OUT    InvertNegatives       at internal/core/services/user/service.go:89:3
```

### Mutation Statuses

| Status | Meaning | Action |
|--------|---------|--------|
| `KILLED` | Tests caught the mutation — good. | No action needed. |
| `LIVED` | Mutation survived — tests don't verify this behavior. | Add or strengthen test cases. |
| `NOT COVERED` | No test executes this line at all. | Generate test scaffolding with `go-testgen gen`. |
| `TIMED OUT` | Tests timed out on this mutant. | Usually indicates an infinite loop; inspect the mutation. |
| `NOT VIABLE` | Mutation produces uncompilable code; skipped. | No action needed. |

**Focus on `LIVED` and `NOT COVERED`** — those are the two signals that translate directly into missing or weak tests.

---

## Coverage Report with Clickable GitHub Links

The same spreadsheet technique from the [Test Coverage](test-coverage.md) page applies here. The command below transforms Gremlins output into a tab-separated table where the last column is a direct GitHub link to the mutated line.

### Setup

**Step 1 — Set your GitHub repository URL (plain, no escaping needed here).**

```bash
export GITHUB_REPO_URL='https://github.com/acme/app'
export PROJECT_BRANCH='main'   # or 'master'
```

**Step 2 — Run Gremlins and transform the output.**

```bash
gremlins unleash | \
sed -E "s/(RUNNABLE|NOT COVERED|LIVED|KILLED|TIMED OUT|NOT VIABLE)[[:space:]]+([a-zA-Z_]+)[[:space:]]+at[[:space:]]+([a-zA-Z0-9\.\/_\-]+):([0-9]+):[0-9]+/\1\t\2\t${GITHUB_REPO_URL}\/blob\/${PROJECT_BRANCH}\/\3#L\4/g;t;d"
```

**Input:**

```
LIVED        ConditionalsBoundary  at internal/core/services/user/service.go:67:8
NOT COVERED  ArithmeticBase        at pkg/utils/parse.go:5:4
KILLED       ConditionalsBoundary  at internal/core/services/user/service.go:45:12
```

**Output (tab-separated):**

```
LIVED        ConditionalsBoundary    https://github.com/acme/app/blob/main/internal/core/services/user/service.go#L67
NOT COVERED  ArithmeticBase          https://github.com/acme/app/blob/main/pkg/utils/parse.go#L5
KILLED       ConditionalsBoundary    https://github.com/acme/app/blob/main/internal/core/services/user/service.go#L45
```

Paste into a spreadsheet — tabs become columns, URLs become clickable links.

### Filtering for actionable mutants only

Show only `LIVED` and `NOT COVERED` — the ones that need attention:

```bash
gremlins unleash | \
sed -E "s/(RUNNABLE|NOT COVERED|LIVED|KILLED|TIMED OUT|NOT VIABLE)[[:space:]]+([a-zA-Z_]+)[[:space:]]+at[[:space:]]+([a-zA-Z0-9\.\/_\-]+):([0-9]+):[0-9]+/\1\t\2\t${GITHUB_REPO_URL}\/blob\/${PROJECT_BRANCH}\/\3#L\4/g;t;d" | \
grep -E '^(LIVED|NOT COVERED)'
```

---

## Workflow: Mutation Testing + go-testgen

Mutation results feed directly into go-testgen:

```bash
# 1. Run mutation testing, filter survivors
gremlins unleash --package ./internal/core/services/user | \
grep -E '^(LIVED|NOT COVERED)'

# Example output:
# LIVED        ConditionalsBoundary  at internal/core/services/user/service.go:67:8
# NOT COVERED  ArithmeticBase        at internal/core/services/user/service.go:89:4

# 2. Use go-testgen report to get the exact gen command for that package
go-testgen report ./internal/core/services/user

# 3. Generate scaffolding for the untested or weakly-tested function
go-testgen gen ./internal/core/services/user Service.FindByID

# 4. Open the linked file (from the mutation output) to understand what the
#    surviving mutant changed, then write a check function that would catch it

# 5. Re-run mutation testing to confirm the new tests kill the survivors
gremlins unleash --package ./internal/core/services/user | \
grep -E '^(LIVED|NOT COVERED)'
```

### Reading a surviving mutant

When `LIVED ConditionalsBoundary at service.go:67` appears, open line 67 in the source. A `ConditionalsBoundary` mutant changes `<` to `<=`, `>` to `>=`, etc. The surviving mutant means no test case exercises the boundary between those two operators.

Write a check function that asserts behavior at exactly that boundary:

```go
// Boundary case: ID at the exact minimum allowed length
{
    name: "rejects ID shorter than minimum length",
    req:  &userDomain.UserFindRequest{ID: "ab"},  // boundary value
    checks: checkServiceFindByID(
        checkServiceFindByIDError("ID must be at least 3 characters"),
    ),
},
{
    name: "accepts ID at minimum length",
    req:  &userDomain.UserFindRequest{ID: "abc"},  // boundary value + 1
    checks: checkServiceFindByID(
        checkServiceFindByIDError(""),
    ),
},
```

This is the full value of the check function pattern combined with mutation testing: surviving mutants point you to the exact behavior that needs a new, focused `checkXxxFn`.
