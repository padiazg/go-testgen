# Mocks

go-testgen generates testify mocks for Go interfaces. This page explains when to use them, how they are generated, and how they integrate with the check function pattern.

## When Mocks Are Generated

Pass `--mock-from qualifier.InterfaceName` to `gen`:

```bash
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository
```

go-testgen resolves the interface in the named package and generates a complete testify mock written to `mock_<interfacename>_test.go` in the same directory as the test file.

The flag is repeatable — pass multiple `--mock-from` flags for functions with multiple interface dependencies:

```bash
go-testgen gen ./internal/core/services/order Service.PlaceOrder \
  --mock-from orderDomain.OrderRepository \
  --mock-from cache.Cache
```

Existing mock files are **never overwritten**.

## Generated Mock Structure

For an interface:

```go
// userDomain package
type UserRepository interface {
    CreateUser(ctx context.Context, req *UserCreateRequest) (*User, error)
    FindByID(ctx context.Context, id string) (*User, error)
    Delete(ctx context.Context, id string) error
}
```

go-testgen generates:

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

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}
```

Key properties:

- All interface methods implemented.  
- Pointer and slice returns use comma-ok type assertion (`r, _ := args.Get(0).(*T)`) — safe when the mock returns `nil`.  
- `args.Error(n)` handles `error` returns correctly, including `nil`.  

## Using Mocks in the `before` Field

Mock expectations are set per test case in the `before` field, not globally:

```go
{
    name: "repo returns error",
    req:  &userDomain.UserCreateRequest{Name: "alice"},
    before: func(s *Service) {
        s.repo.(*mockUserRepository).
            On("CreateUser", mock.Anything, mock.Anything).
            Return(nil, errors.New("connection refused"))
    },
    checks: checkServiceCreateUser(
        checkCreateUserError("connection refused"),
    ),
},
```

This keeps each test case self-contained: the mock is configured, the function is called, checks run.

## Skipping Mock Generation

If a mock file already exists (from a previous `gen` call or written by hand), go-testgen will not overwrite it. Omit `--mock-from` for subsequent functions in the same package:

```bash
# First call — creates the mock
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# Subsequent calls — mock already exists, skip --mock-from
go-testgen gen ./internal/core/services/user Service.FindByID
go-testgen gen ./internal/core/services/user Service.Delete
```

The `report` command shows `--mock-from` flags only for interfaces whose mock file does not yet exist, so you can copy-paste its suggested commands directly.

## Manual Mocks

You can write mocks by hand or use another tool (e.g., `mockery`). go-testgen does not require its own generated mocks — it only generates them when `--mock-from` is passed.
