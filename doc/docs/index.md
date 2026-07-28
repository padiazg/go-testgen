# go-testgen

**go-testgen** is a CLI tool that generates unit test scaffolding for Go projects. It produces closure-based check functions, a `before` hook for mock setup, and table-driven tests that compose cleanly as test suites grow.

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
| `check` (default) | `--style check` | Methods, constructors, anything with multiple assertions |
| `table` | `--style table` | Pure transformation functions |
| `simple` | `--style simple` | Single-path utilities |

→ [Compare Test Styles](test-styles/index.md)

### Channel Types

go-testgen correctly handles `chan`, `chan<-`, and `<-chan` in parameters and results — detecting direction, qualifying types, and using `nil` placeholders.

→ [Channel Type Support](concepts/channels.md)

### Spec-Driven Case Generation

Author a `.testspec.yaml` describing scenarios in domain terms, then run `gen-cases` to materialize the struct literals into the existing test file with `// ai-hint:` stubs for a generative AI to complete.

```bash
go-testgen gen-cases ./engine/engine_start.testspec.yaml
```

→ [gen-cases command reference](commands/gen-cases.md)

### Smart Merge

If the target `_test.go` already exists, go-testgen appends the new test and injects missing imports. It never overwrites an existing test function.

## Quick Start

```bash
# Install
curl -fsSL https://padiazg.github.io/go-testgen/install.sh | sh

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
curl -fsSL https://padiazg.github.io/go-testgen/install.sh | sh
```

→ [Installation Guide](getting-started/installation.md)
