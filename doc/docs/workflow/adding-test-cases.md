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

The number of TODO cases is controlled by `number_of_todos` in `.go-testgen.yaml` (default: `1`).

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
