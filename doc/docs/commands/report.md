# `report` — Package Test Coverage Overview

`report` scans all exported functions and methods in a package and shows test coverage status, interface dependencies, and ready-to-run `gen` commands.

## Syntax

```bash
go-testgen report <pkg-pattern> [flags]
```

## Example

```bash
go-testgen report ./internal/core/services/user
```

Output (text format):

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
         userDomain.UserRepository   mock_userrepository_test.go  ✓
       Suggest: go-testgen gen ./internal/core/services/user Service.FindByID
```

Note: `--mock-from` is only included in the suggestion when the mock file does not yet exist.

## Suggested Style (Experimental)

!!! warning "Experimental"
    Style suggestion is experimental. It applies static heuristics to the function's signature — it has no knowledge of business logic, naming conventions, or test complexity. Always review the suggested `--style` before running `gen` and override it when it does not fit.

When the inferred style is not the default (`check`), the `Suggest:` line includes a `--style` flag. The decision follows a fixed priority chain — **first match wins**:

| Priority | Condition | Suggested style |
|----------|-----------|-----------------|
| 1 | Function has **no return values** | `simple` |
| 2 | Has a `context.Context` param **and** ≥ 2 interface dependencies | `simple` |
| 3 | Returns a **pointer** or **slice** | `check` |
| 4 | Returns an **interface** type | `check` |
| 5 | Is a **method** with ≥ 1 interface dependency | `check` |
| 6 | Has **more than 1 non-error return value** | `check` |
| 7 | *(default)* scalar returns, no interface deps | `table` |

**Reasoning behind each rule:**

- **No returns → `simple`**: nothing to assert on the result; a table loop adds no value.
- **Context + many interfaces → `simple`**: high interface count often means integration-style setup that reads better as explicit steps than as a table row.
- **Pointer/slice return → `check`**: `reflect.DeepEqual` on a whole struct or slice is brittle; check functions let you assert individual fields or elements.
- **Interface return → `check`**: interfaces cannot be compared with `==` or `DeepEqual` reliably.
- **Method + interface dep → `check`**: mock expectations vary per case and belong in `before`, which only the `check` style generates.
- **Multiple non-error returns → `check`**: hard to pack all expected values cleanly into `want` fields.
- **Everything else → `table`**: scalar types (`string`, `int`, `bool`, error-only) map cleanly to `want` fields.

**When the suggestion is wrong — common cases:**

| Situation | Override |
|-----------|----------|
| `table` or `simple` suggested but function mutates state | `--style check` |
| `check` suggested but function is a pure converter | `--style table` |
| `simple` suggested but you want multiple parameterized cases | `--style table` or `--style check` |
| Any case where the default is inconvenient | `--style <style>` |

The suggestion is a starting point, not a prescription.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text`, `table`, or `json`. |

## Output Formats

### `text` (default)

Human-readable with ✓/✗ symbols. Best for interactive use.

### `table`

Tabular layout. Good for wider terminals or when comparing multiple packages.

### `json`

Machine-readable. Useful for scripting or integrating with other tools.

```bash
go-testgen report ./internal/... --format json | jq '.[] | select(.tested == false)'
```

## Typical Workflow

1. Run `report` to see what is missing.
2. Copy the `Suggest:` lines and run them.
3. Re-run `report` to confirm coverage.

```bash
# Identify gaps
go-testgen report ./internal/core/services/...

# Generate missing tests (copy from Suggest: output)
go-testgen gen ./internal/core/services/user Service.CreateUser --mock-from userDomain.UserRepository

# Verify
go-testgen report ./internal/core/services/...
```
