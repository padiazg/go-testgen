# Adding Test Cases

go-testgen generates placeholder `TODO` cases. This page explains how to turn them into real tests.

## What Gets Generated

After running `gen`, the test table contains one or more placeholder rows:

```go
tests := []struct {
    name   string
    req    *userDomain.UserCreateRequest
    before func(*Service)
    checks []checkServiceCreateUserFn
}{
    {
        name:   "TODO: success case",
        checks: checkServiceCreateUser(),
    },
    {
        name:   "TODO: error case",
        checks: checkServiceCreateUser(
            checkServiceCreateUserError("TODO: expected error message"),
        ),
    },
}
```

The number of TODO cases is controlled by `number_of_todos` in `.go-testgen.yaml` (default: `2`).

## Good Practices Before You Start

go-testgen scaffolds the structure. What makes tests valuable is what you put inside them. A few principles worth keeping in mind as you fill in the TODO cases.

### Test behavior, not implementation

A unit test should answer: *"given these inputs and conditions, does the unit behave correctly?"* — not *"does the code follow these internal steps?"*

**Wrong approach — testing implementation:**
```go
// Asserts that a specific internal method was called with specific args.
// This breaks whenever you refactor, even if behavior stays correct.
{
    name: "calls repo.CreateUser once",
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(&userDomain.User{}, nil).
            Times(1)  // ← testing internal call count, not outcome
    },
    checks: checkServiceCreateUser(),
},
```

**Right approach — testing behavior:**
```go
// Asserts what the caller cares about: the returned user has the right name.
{
    name: "returns user with the requested name",
    req:  &userDomain.UserCreateRequest{Name: "alice"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(&userDomain.User{Name: "alice"}, nil)
    },
    checks: checkServiceCreateUser(
        checkServiceCreateUserError(""),
        checkUserName("alice"),
    ),
},
```

If you can refactor the internals without changing any test, the tests are testing behavior. If a pure internal rename breaks a test, the test is too tightly coupled to the implementation.

### Don't write tests to hit a coverage number

Code coverage is a useful signal, not a goal. A function with 100% coverage and no meaningful assertions tells you nothing about correctness. A function with 70% coverage but well-designed behavioral cases is far more valuable.

Ask: *"if this code had a bug, would this test catch it?"* If the answer is no, the test case is not adding safety — only noise.

Situations where coverage-driven tests mislead:
- Asserting only that no panic occurred, without checking the output.
- Duplicating cases that exercise the same code path with different variable names.
- Testing error paths with `wantErr: true` but never checking the error message.

### One scenario per test case

Each row in the table should represent a single, distinct scenario. When a case tries to verify multiple unrelated behaviors at once, failures become hard to diagnose.

**Hard to diagnose:**
```go
{
    name: "various validations",
    // Tests nil input, missing name, AND repo error all in one case.
}
```

**Clear:**
```go
{name: "nil request is rejected"},
{name: "missing name is rejected"},
{name: "repo unavailable propagates error"},
```

### Test names are documentation

The test name is what appears in `go test -v` output and in CI failure logs. Write it as a sentence that describes the scenario, not the code path:

| Avoid | Prefer |
|-------|--------|
| `"success"` | `"returns created user when request is valid"` |
| `"error"` | `"returns error when repository is unavailable"` |
| `"nil"` | `"returns validation error when request is nil"` |
| `"test case 1"` | `"creates user with minimal required fields"` |

A good name makes the failure self-explanatory without opening the source file.

### Cover the boundaries, not just the happy path

The happy path (valid input → expected output) is usually the first case written and the least likely to reveal bugs. Invest equally in:

- **Zero/nil/empty inputs**: `nil` request, empty string, zero int.
- **Boundary values**: minimum and maximum allowed values, exactly-at-limit inputs.
- **Dependency failures**: what happens when the repository, cache, or HTTP client returns an error.
- **Partial failures**: first call succeeds, second fails.
- **Idempotency**: calling the same operation twice produces the correct result both times.

---

## Step 1: Name Your Cases

Replace `TODO: success case` with a description of what the case tests:

```go
name: "creates user with valid request",
```

Good names answer "what scenario does this case cover?" — not just "success" or "error".

## Step 2: Set Input Fields

Fill in the input fields of the table struct. Context parameters are already injected in the runner; only non-context inputs appear as fields:

```go
{
    name: "creates user with valid request",
    req: &userDomain.UserCreateRequest{
        Name:  "alice",
        Email: "alice@example.com",
    },
},
```

## Step 3: Configure the `before` Hook

For methods with interface dependencies, set up mock expectations in `before`:

```go
{
    name: "creates user with valid request",
    req:  &userDomain.UserCreateRequest{Name: "alice"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(&userDomain.User{ID: "uuid-1", Name: "alice"}, nil)
    },
},
```

Use `before` to:
- Set mock return values per case.
- Set a field to `nil` to trigger nil-pointer paths.
- Replace a dependency with a test double.

## Step 4: Add Checks

Populate the `checks` field with assertions. The generated `checkXxxError` handles error cases; write domain-specific checks for everything else:

```go
checks: checkServiceCreateUser(
    checkServiceCreateUserError(""),   // no error expected
    checkUserName("alice"),
    checkUserIDNotEmpty(),
),
```

For an error case:

```go
{
    name: "repo unavailable",
    req:  &userDomain.UserCreateRequest{Name: "alice"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(nil, errors.New("connection refused"))
    },
    checks: checkServiceCreateUser(
        checkServiceCreateUserError("connection refused"),
    ),
},
```

## Step 5: Run Tests

```bash
go test ./internal/core/services/user/... -v -run TestService_CreateUser
```

## Patterns for Common Scenarios

### Testing nil input

```go
{
    name:   "nil request returns error",
    req:    nil,
    checks: checkServiceCreateUser(
        checkServiceCreateUserError("request must not be nil"),
    ),
},
```

### Testing state after the call

If your check function receives the receiver, you can assert post-call state:

```go
func checkServiceHasUser(id string) checkServiceCreateUserFn {
    return func(t *testing.T, _ *userDomain.User, _ error) {
        // assert via the service's state if accessible
        t.Helper()
        // ...
    }
}
```

### Testing multiple scenarios quickly

Duplicate existing cases and modify only what changes:

```go
{
    name: "creates user — minimal fields",
    req:  &userDomain.UserCreateRequest{Name: "bob"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(&userDomain.User{ID: "uuid-2", Name: "bob"}, nil)
    },
    checks: checkServiceCreateUser(
        checkServiceCreateUserError(""),
        checkUserName("bob"),
    ),
},
```
