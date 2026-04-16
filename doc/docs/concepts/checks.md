# The Check Function Pattern

The check function pattern is the central idea behind go-testgen's default `check` style. Understanding it makes the generated code immediately readable and shows why it is more flexible than `want` fields.

## The Problem With `want` Fields

Traditional table-driven tests compare results through `want` fields:

```go
tests := []struct {
    name    string
    input   string
    want    *User
    wantErr bool
}{
    {name: "valid", input: "alice", want: &User{Name: "alice"}, wantErr: false},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := CreateUser(tt.input)
        if (err != nil) != tt.wantErr {
            t.Fatalf("unexpected error: %v", err)
        }
        if !reflect.DeepEqual(got, tt.want) {
            t.Errorf("got %+v, want %+v", got, tt.want)
        }
    })
}
```

This works for simple cases but has limits:

- One assertion style fits all cases — you cannot easily check different things per test case.
- Adding a new field to check means adding a new column to every row.
- The assertion logic is buried in the loop, not named.

## The Check Function Solution

A **check function** (`checkXxxFn`) is a typed closure that asserts one specific thing about the function's output. Multiple check functions compose per test case via the `checks []checkXxxFn` field.

### The Type Alias

```go
type checkServiceCreateUserFn func(*testing.T, *userDomain.User, error)
```

The signature mirrors the function under test's return list, plus `*testing.T` as the first parameter.

### The Collector

```go
var checkServiceCreateUser = func(fns ...checkServiceCreateUserFn) []checkServiceCreateUserFn {
    return fns
}
```

This is just a variadic identity function. Its purpose is to provide a named, type-safe way to build the `checks` slice in the table — `checkServiceCreateUser(fn1, fn2)` reads naturally at the call site.

### Individual Check Functions

Each check function tests one aspect:

```go
// checkCreateUserNoError verifies no error was returned.
func checkCreateUserNoError() checkServiceCreateUserFn {
    return func(t *testing.T, _ *userDomain.User, err error) {
        t.Helper()
        assert.NoErrorf(t, err, "checkCreateUserNoError: unexpected error")
    }
}

// checkCreateUserName verifies the returned user has the expected name.
func checkCreateUserName(want string) checkServiceCreateUserFn {
    return func(t *testing.T, u *userDomain.User, _ error) {
        t.Helper()
        assert.Equalf(t, want, u.Name, "checkCreateUserName mismatch")
    }
}

// checkCreateUserError verifies an error containing want was returned.
func checkCreateUserError(want string) checkServiceCreateUserFn {
    return func(t *testing.T, _ *userDomain.User, err error) {
        t.Helper()
        if want == "" {
            assert.NoErrorf(t, err, "checkCreateUserError: expected no error, got %v", err)
            return
        }
        if assert.Errorf(t, err, "checkCreateUserError: expected error %q", want) {
            assert.Containsf(t, err.Error(), want, "checkCreateUserError mismatch")
        }
    }
}
```

Key properties:

- `t.Helper()` makes failures point to the test case, not the check function body.
- Each function has a single responsibility and a descriptive name.
- Parameterized checks (like `checkCreateUserName("alice")`) return a closure with the parameter captured.

### The Test Table

```go
func TestService_CreateUser(t *testing.T) {
    tests := []struct {
        name   string
        req    *userDomain.UserCreateRequest
        before func(*Service)
        checks []checkServiceCreateUserFn
    }{
        {
            name: "success",
            req:  &userDomain.UserCreateRequest{Name: "alice"},
            before: func(s *Service) {
                s.repo.(*mockUserRepository).
                    On("CreateUser", mock.Anything, mock.Anything).
                    Return(&userDomain.User{Name: "alice"}, nil)
            },
            checks: checkServiceCreateUser(
                checkCreateUserNoError(),
                checkCreateUserName("alice"),
            ),
        },
        {
            name: "repo error",
            req:  &userDomain.UserCreateRequest{Name: "alice"},
            before: func(s *Service) {
                s.repo.(*mockUserRepository).
                    On("CreateUser", mock.Anything, mock.Anything).
                    Return(nil, errors.New("db unavailable"))
            },
            checks: checkServiceCreateUser(
                checkCreateUserError("db unavailable"),
            ),
        },
    }
    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            s := New(nil)
            if tt.before != nil {
                tt.before(s)
            }
            r, err := s.CreateUser(context.Background(), tt.req)
            for _, c := range tt.checks {
                c(t, r, err)
            }
        })
    }
}
```

### The `before` Field

`before func(*ReceiverType)` is called between setup and act. It exists to:

- Configure mock expectations per test case (each case can expect different calls).
- Mutate internal state of the receiver (e.g., set a field to nil to trigger an error path).
- Keep the test table clean — mock setup does not pollute the input fields.

`before` is always `nil`-guarded:

```go
if tt.before != nil {
     tt.before(s) 
}
```

## Why Checks Beat `want` Fields

| Property | `want` fields | Check functions |
|---|---|---|
| Per-case assertions | Fixed columns | Any combination per case |
| Adding a new assertion | New column in every row | New check function, add to relevant cases |
| Assertion failure messages | Generic (`got X, want Y`) | Named (`checkCreateUserName mismatch`) |
| Reuse across test files | Repeated inline code | Exported check functions from a `testutil` package |
| Partial assertions | Hard (all fields must be set) | Natural (assert only what matters) |

## Generated vs. Hand-Written Checks

go-testgen generates:
- The `checkXxxFn` type alias
- The `checkXxx` collector
- A `checkXxxError` function when the function returns `error`

Everything else — domain-specific check functions like `checkCreateUserName` — is written by you. The generated skeleton is a starting point; you fill in what matters for each function.

## Composing Checks in Practice

A single test case can mix and match any number of checks:

```go
{
    name: "success with all validations",
    req:  &userDomain.UserCreateRequest{Name: "alice", Email: "alice@example.com"},
    checks: checkServiceCreateUser(
        checkCreateUserNoError(),
        checkCreateUserName("alice"),
        checkCreateUserEmail("alice@example.com"),
        checkCreateUserIDNotEmpty(),
    ),
},
```

An error case only needs the error check — no need to populate unused `want` fields:

```go
{
    name:   "missing name",
    req:    &userDomain.UserCreateRequest{},
    checks: checkServiceCreateUser(
        checkCreateUserError("name is required"),
    ),
},
```
