# Quick Start

This guide walks through generating your first test in under two minutes.

## 1. See What Needs Tests

Run `report` against a package to get a coverage overview and ready-to-run `gen` commands:

```bash
go-testgen report ./internal/core/services/user
```

Example output:

```
Package: github.com/acme/app/internal/core/services/user
Source:  /path/to/internal/core/services/user

  ✓  TestService_New
       New(cfg *Config) *Service

  ✗  TestService_CreateUser
       Service.CreateUser(ctx context.Context, req *userDomain.UserCreateRequest) (*userDomain.User, error)
       Interface deps:
         userDomain.UserRepository   mock_userrepository_test.go  ✗
       Suggest: go-testgen gen ./internal/core/services/user Service.CreateUser --mock-from userDomain.UserRepository

  ✗  TestService_FindByID
       Service.FindByID(ctx context.Context, id string) (*userDomain.User, error)
       Interface deps:
         userDomain.UserRepository   mock_userrepository_test.go  ✗
       Suggest: go-testgen gen ./internal/core/services/user Service.FindByID
```

## 2. Generate Tests + Mocks

Copy the suggested command from the report output and run it:

```bash
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository
```

This creates two files:
- `internal/core/services/user/service_test.go` — the test scaffold
- `internal/core/services/user/mock_userrepository_test.go` — the testify mock

## 3. Generate Subsequent Tests (Mock Already Exists)

For functions that share the same interface dependency, omit `--mock-from`:

```bash
go-testgen gen ./internal/core/services/user Service.FindByID
```

go-testgen detects the existing mock and does not regenerate it.

## 4. Fill In Test Cases

Open the generated file. You'll find placeholder `TODO` cases:

```go
{
    name:   "TODO: success case",
    checks: checkServiceCreateUser(),
},
```

Replace with real cases. See [Adding Test Cases](../workflow/adding-test-cases.md) for guidance.

## 5. Run Tests

```bash
go test ./internal/core/services/user/...
```

## Preview Without Writing

Use `-o -` to print to stdout without touching the filesystem:

```bash
go-testgen gen ./internal/core/services/user Service.CreateUser -o -
```

## Debug the Analyzer

If the generated code looks wrong, dump the parsed function info:

```bash
go-testgen inspect ./internal/core/services/user Service.CreateUser
```

This prints the `FuncInfo` JSON that `gen` uses internally — useful for diagnosing wrong signatures or missing types.
