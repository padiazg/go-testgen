# go-testgen

**go-testgen** is a CLI tool that generates unit test scaffolding for Go projects. It produces closure-based check functions, a `before` hook for mock setup, and table-driven tests that compose cleanly as test suites grow.

## Installation

```bash
go install github.com/padiazg/go-testgen/cmd/go-testgen@latest
```

Or build from source:

```bash
make build
make install
```

## Commands

### `gen` — generate test scaffolding

```bash
go-testgen gen <pkg-pattern> <FuncSpec> [flags]
```

`FuncSpec` is either a plain function name (`New`, `CreateUser`) or a receiver-qualified method name (`Service.CreateUser`).

```bash
# Constructor
go-testgen gen ./internal/core/services/user New

# Method
go-testgen gen ./internal/core/services/user Service.CreateUser

# Write to stdout instead of file
go-testgen gen ./internal/core/services/user Service.CreateUser -o -

# Write to a specific file
go-testgen gen ./internal/core/services/user Service.CreateUser -o user_test.go

# Also generate a testify mock for an interface used by the function
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# Multiple mocks
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository \
  --mock-from cache.Cache
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | auto | Output file. Omit to auto-detect (`<source>_test.go`). Use `-` for stdout. |
| `-v`, `--verbose` | false | Print parsed `FuncInfo` JSON to stderr before generating |
| `--style` | — | Path to `.go-testgen.yaml` config file |
| `--mock-from` | — | Generate a testify mock for `qualifier.InterfaceName` (repeatable) |

**Smart merge:** if the target `_test.go` already exists but doesn't contain the test function, go-testgen appends the new test and injects any missing imports. It never overwrites an existing test function without prompting.

### `inspect` — debug parsed function info

```bash
go-testgen inspect <pkg-pattern> <FuncSpec>
```

Prints the `FuncInfo` JSON that `gen` uses internally. Useful for diagnosing wrong signatures or missing types.

### `report` — package test coverage overview

```bash
go-testgen report <pkg-pattern> [--format text|table|json]
```

Scans all exported functions and methods in a package and shows:
- Whether a test function exists
- Interface dependencies inferred from struct fields
- Mock file existence
- Exact `go-testgen gen` command to run for each untested function

**Example output (text format):**

```
Package: github.com/padiazg/user-manager/internal/core/services/user
Source:  /path/to/internal/core/services/user

  ✓  TestService_New
       New(cfg *Config) *Service

  ✗  TestService_CreateUser
       Service.CreateUser(ctx context.Context, req *userDomain.UserCreateRequest) (*userDomain.User, error)
       Interface deps:
         userDomain.UserRepository   mock_userrepository_test.go  ✓
       Suggest: go-testgen gen ./internal/core/services/user Service.CreateUser

  ✗  TestService_FindByID
       Service.FindByID(ctx context.Context, id string) (*userDomain.User, error)
       Interface deps:
         userDomain.UserRepository   mock_userrepository_test.go  ✓
       Suggest: go-testgen gen ./internal/core/services/user Service.FindByID
```

`--mock-from` flags are only suggested for interfaces whose mock file does not yet exist.

**Formats:**

| Flag | Description |
|------|-------------|
| `--format text` | Default. Human-readable with ✓/✗ |
| `--format table` | Tabular layout (uses go-pretty) |
| `--format json` | Machine-readable JSON |

## Generated test style

```go
type checkServiceCreateUserFn func(*testing.T, *userDomain.User, error)

var checkServiceCreateUser = func(fns ...checkServiceCreateUserFn) []checkServiceCreateUserFn {
    return fns
}

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

func TestService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        req     *userDomain.UserCreateRequest
        before  func(*Service)
        checks  []checkServiceCreateUserFn
    }{
        {
            name:   "TODO: success case",
            checks: checkServiceCreateUser(),
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

Key properties:
- Check functions are closures — each assertion is a separate `checkXxxFn`, composable via `checkXxx(fn1, fn2, ...)`.
- The `before` hook sets up mock expectations per test case.
- The check function signature mirrors the function's full return list (including `error`).
- Context parameters are injected automatically (`context.Background()`), not exposed in the table.

## Generated mock style

`--mock-from qualifier.InterfaceName` generates a complete testify mock for the named interface, written to `mock_<interfacename>_test.go` in the same directory as the test file.

```go
type mockUserRepository struct {
    mock.Mock
}

func (m *mockUserRepository) CreateUser(ctx context.Context, req *userDomain.UserCreateRequest) (*userDomain.User, error) {
    args := m.Called(ctx, req)
    r, _ := args.Get(0).(*userDomain.User)
    return r, args.Error(1)
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*userDomain.User, error) {
    args := m.Called(ctx, id)
    r, _ := args.Get(0).(*userDomain.User)
    return r, args.Error(1)
}
```

- All interface methods are implemented.
- Pointer/slice returns use comma-ok type assertion (`r, _ := args.Get(0).(*T)`) — safe when mock returns `nil`.
- Existing mock files are never overwritten.

## Configuration

Create `.go-testgen.yaml` in your project root:

```yaml
receiver_var_name: "s"      # variable name for the receiver in tests
result_var_name: "r"        # variable name for non-error return values
error_var_name: "err"       # variable name for the error return
use_testify: true           # use testify assert helpers
add_todo_cases: true        # add placeholder TODO test cases
number_of_todos: 1          # how many TODO cases to add
check_type_suffix: "Fn"     # suffix for check type names (e.g. checkCreateUserFn)
generate_mocks: true        # generate mock files when --mock-from is used
generate_checks: true       # generate checkXxx helper functions
```

## Typical workflow

```bash
# 1. See what needs tests
go-testgen report ./internal/core/services/user

# 2. Generate tests + mocks for each untested function
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# 3. Subsequent functions reuse the existing mock (no --mock-from needed)
go-testgen gen ./internal/core/services/user Service.FindByID

# 4. Fill in test cases, run tests
go test ./internal/core/services/user/...
```
