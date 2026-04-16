# `check` Style

The `check` style is the default and the most expressive. It combines a table-driven structure with closure-based check functions for composable, named assertions.

See [The Check Function Pattern](../concepts/checks.md) for a deep-dive into how and why this pattern works.

## When to Use

- Methods on structs with dependencies.
- Constructors (`New`).
- Functions returning `error` where some cases expect errors and others do not.
- Anything requiring different assertions per test case.

## Generated Structure

```go
// 1. Type alias — mirrors the function's return signature + *testing.T
type checkServiceCreateUserFn func(*testing.T, *userDomain.User, error)

// 2. Collector — type-safe variadic builder
var checkServiceCreateUser = func(fns ...checkServiceCreateUserFn) []checkServiceCreateUserFn {
    return fns
}

// 3. Error check (generated when the function returns error)
func checkServiceCreateUserError(want string) checkServiceCreateUserFn {
    return func(t *testing.T, _ *userDomain.User, err error) {
        t.Helper()
        if want == "" {
            assert.NoErrorf(t, err, "checkServiceCreateUserError: expected no error, got %v", err)
            return
        }
        if assert.Errorf(t, err, "checkServiceCreateUserError: expected error %q", want) {
            assert.Containsf(t, err.Error(), want, "checkServiceCreateUserError mismatch")
        }
    }
}

// 4. Test function
func TestService_CreateUser(t *testing.T) {
    tests := []struct {
        name   string
        req    *userDomain.UserCreateRequest
        before func(*Service)
        checks []checkServiceCreateUserFn
    }{
        {
            name:   "success case",
            checks: checkServiceCreateUser(
                checkServiceCreateUserError(""),
            ),
        },
        {
            name:   "fail case",
            checks: checkServiceCreateUser(
                checkServiceCreateUserError("expected error message"),
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

## What Is Generated

| Element | Always? | Description |
|---------|---------|-------------|
| `checkXxxFn` type | Yes | Signature: `*testing.T` + function's return types |
| `checkXxx` collector | Yes | Variadic identity — builds the checks slice |
| `checkXxxError` | When `HasError == true` | Checks error presence and message content |
| `before` field | When `IsMethod == true` | Per-case mock setup / state mutation |
| Context injection | When first param is `context.Context` | Injected as `context.Background()`, not in table |
| TODO cases | Configurable | Placeholder rows controlled by `number_of_todos` |

## Adding Your Own Check Functions

The generated `checkXxxError` is a starting point. Add domain-specific checks by hand:

```go
func checkUserName(want string) checkServiceCreateUserFn {
    return func(t *testing.T, u *userDomain.User, _ error) {
        t.Helper()
        assert.Equalf(t, want, u.Name, "checkUserName: got %q, want %q", u.Name, want)
    }
}

func checkUserIDNotEmpty(t *testing.T, u *userDomain.User, _ error) {
    t.Helper()
    assert.NotEmptyf(t, u.ID, "checkUserIDNotEmpty: ID should not be empty")
}
```

Then compose them in the table:

```go
{
    name: "creates user with correct fields",
    req:  &userDomain.UserCreateRequest{Name: "alice"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(&userDomain.User{ID: "123", Name: "alice"}, nil)
    },
    checks: checkServiceCreateUser(
        checkServiceCreateUserError(""),
        checkUserName("alice"),
        checkUserIDNotEmpty,
    ),
},
```
