# Configuration

go-testgen is configured via a YAML file. Place `.go-testgen.yaml` in your project root.

## Config File Search Order

1. Explicit path via `--style` flag
2. `.go-testgen.yaml` in the current working directory
3. `.go-testgen.yaml` in `$HOME`
4. Hardcoded defaults

## Full Reference

```yaml
# Variable names in generated tests
receiver_var_name: "s"       # Variable name for the receiver. Default: first letter of the type, lowercased.
result_var_name: "r"         # Variable name for non-error return values. Default: "r".
error_var_name: "err"        # Variable name for the error return. Default: "err".

# Assertion library
use_testify: true             # Use github.com/stretchr/testify assert helpers. Default: true.
use_require: false            # Use testify require (stops on first failure) instead of assert. Default: false.

# Placeholder cases
add_todo_cases: true          # Add TODO placeholder rows to the generated table. Default: true.
number_of_todos: 2            # How many TODO rows to add. Default: 2.

# Check type naming
check_type_suffix: "CheckFn"  # Suffix for the generated check type name.
                              # e.g. "CheckFn" → checkServiceCreateUserCheckFn
                              # Default: "CheckFn".
check_type_prefix: ""         # Prefix for the check type name. Default: "" (inferred from type).
mock_prefix: "mock"           # Prefix for generated mock type names. Default: "mock".

# Generation flags
generate_mocks: true          # Generate mock files when --mock-from is passed. Default: true.
generate_checks: true         # Generate checkXxxError and check type definitions. Default: true.

# Default test style
test_style: "check"           # Default style: check, table, or simple. Default: "check".
```

## Minimal Config

Most projects only need a few options:

```yaml
receiver_var_name: "s"
use_testify: true
number_of_todos: 2
```

## Per-Project vs. Global

- Per-project: `.go-testgen.yaml` in the repo root — checked into version control so all contributors share the same style.
- Global: `~/.go-testgen.yaml` — personal preferences that apply to all projects.

Per-project takes precedence over global.

## Option Details

### `receiver_var_name`

Controls the variable name used for the receiver in the test setup:

```yaml
receiver_var_name: "e"
```

Generates:

```go
e := New(nil)
if tt.before != nil {
    tt.before(e)
}
```

Default is the first letter of the receiver type, lowercased (e.g., `Service` → `s`).

### `check_type_suffix`

Controls the suffix of the generated check type name:

```yaml
check_type_suffix: "CheckFn"   # default
```

Generates:

```go
type checkServiceCreateUserCheckFn func(*testing.T, *userDomain.User, error)
```

### `number_of_todos`

How many TODO placeholder rows to generate. Default is `2` — one success case and one error case:

```yaml
number_of_todos: 2
```

### `use_require`

When `true`, uses `testify/require` instead of `testify/assert`. `require` stops the test on the first failure; `assert` continues. For check functions, `assert` is almost always preferable so all checks in a case run to completion.
