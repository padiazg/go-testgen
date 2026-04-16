# go-testgen

**go-testgen** is a CLI tool that generates unit test scaffolding for Go projects in "padiazg style": closure-based check functions, a `before` hook for mock setup, and table-driven tests that compose cleanly as test suites grow.

## What It Does

Point it at a function or method — it reads the AST, infers the right scaffolding, and writes a complete `_test.go` ready for you to fill in:

```bash
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository
```

Generated output includes:

- A `checkServiceCreateUserFn` type alias matching the function's return signature
- A `checkServiceCreateUser` variadic collector for composing assertions
- A `checkServiceCreateUserError` closure for error assertions
- A testify mock for `UserRepository`
- A `TestService_CreateUser` table with placeholder `TODO` cases

## Key Concepts

### Check Functions

The central idea. Instead of `want` fields, each assertion is a named closure (`checkXxxFn`). Multiple checks compose per test case — different cases assert different things without adding columns to every row.

→ [Learn about the Check Function Pattern](concepts/checks.md)

### Three Test Styles

| Style | Flag | Best for |
|-------|------|---------|
| `check` (default) | `--test-style check` | Methods, constructors, anything with multiple assertions |
| `table` | `--test-style table` | Pure transformation functions |
| `simple` | `--test-style simple` | Single-path utilities |

→ [Compare Test Styles](test-styles/index.md)

### Smart Merge

If the target `_test.go` already exists, go-testgen appends the new test and injects missing imports. It never overwrites an existing test function.

## Quick Start

```bash
# Install
go install github.com/padiazg/go-testgen@latest

# See what needs tests
go-testgen report ./internal/core/services/user

# Generate tests + mock
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# Preview without writing (stdout)
go-testgen gen ./internal/core/services/user Service.CreateUser -o -
```

→ [Full Quick Start](getting-started/quickstart.md)

## Installation

```bash
go install github.com/padiazg/go-testgen@latest
```

→ [Installation Guide](getting-started/installation.md)
