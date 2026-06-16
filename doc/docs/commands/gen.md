# `gen` — Generate Test Scaffolding

`gen` analyzes a function or method and produces a complete test scaffold. It can also generate mocks standalone without a consuming package.

## Syntax

```bash
go-testgen gen <pkg-pattern> <FuncSpec> [flags]                          # normal mode
go-testgen gen --mock-from <spec> --pkg <name> --output <path> [flags]   # standalone mode
```

`pkg-pattern` is the Go package path (e.g., `./internal/core/services/user`, `github.com/acme/app/pkg/auth`).

`FuncSpec` is either:

- A plain function name: `New`, `CreateUser`, `RandomID`
- A receiver-qualified method name: `Service.CreateUser`, `Engine.Dispatch`

## Examples

```bash
# Constructor
go-testgen gen ./internal/core/services/user New

# Method
go-testgen gen ./internal/core/services/user Service.CreateUser

# Write to stdout (preview without touching the filesystem)
go-testgen gen ./internal/core/services/user Service.CreateUser -o -

# Write to a specific file
go-testgen gen ./internal/core/services/user Service.CreateUser \
  -o mytest_test.go

# Generate with a mock for an interface
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --mock-from userDomain.UserRepository

# Multiple mocks
go-testgen gen ./internal/core/services/order Service.PlaceOrder \
  --mock-from orderDomain.OrderRepository \
  --mock-from cache.Cache

# Choose a test style
go-testgen gen ./internal/core/services/user Service.CreateUser \
  --style table
go-testgen gen ./pkg/utils Sanitize --style simple

# Standalone: generate mocks for stdlib/external interfaces
go-testgen gen --mock-from "io.Writer" --pkg mypkg -o -
go-testgen gen --mock-from "io.Writer" --mock-from "io.Reader" \
  --pkg mypkg --output ./mocks/
go-testgen gen --mock-from "net/http.Handler" --pkg mypkg -o -
```

## Flags

| Flag | Default | Description |
| - | - | - |
| `-o`, `--output` | auto | Output file. Omit to auto-detect (`<source>_test.go`). Use `-` for stdout (preview). Required in standalone mode. |
| `-v`, `--verbose` | false | Print parsed `FuncInfo` JSON to stderr before generating. |
| `--style` | from config / `check` | Test style: `check`, `table`, or `simple`. Overrides `test_style` in config. |
| `--config` | auto | Path to `.go-testgen.yaml` config file. |
| `--pkg` | — | Package name for generated mock files. Required in standalone mode. |
| `--mock-from` | — | Generate a testify mock (repeatable). Formats: `qualifier.IfaceName` for cross-package, `path/to.IfaceName` for full import path (e.g. `io/fs.FS`), `.IfaceName` for same-package, or bare `IfaceName` (same package). |

## Output Files

When `--output` is omitted, go-testgen writes to `<source_file>_test.go` next to the source file.

Mock files are written to `mock_<interfacename>_test.go` in the same directory.

## Smart Merge

If the target `_test.go` already exists but does not contain the test function, go-testgen **appends** the new test and injects any missing imports. It never overwrites an existing test function.

If the target `_test.go` exists **and** the test function already exists, go-testgen prompts for overwrite (Spanish: [S]obrescribir / [C]ancelar).

## Standalone Mode

When no positional args are provided but `--mock-from` is present, go-testgen enters standalone
mock mode. This is useful for generating mocks for stdlib interfaces (`io.Writer`, `net/http.Handler`)
or interfaces from external modules without needing a consuming package in scope.

```bash
go-testgen gen --mock-from "io.Writer" --pkg mypkg --output ./
```

Required in standalone mode:
| Flag | Description |
| `-o`, `--output` | Output file or directory |
| `--pkg` | Package name for generated files |

The `--mock-from` spec supports full import paths (containing `/`) and single-segment stdlib names.
When the spec contains `/`, it is resolved directly. When it does not (e.g. `io.Writer`),
go-testgen falls back to treating the qualifier as a direct import path.

## Factory Function Instantiation

When generating tests for methods, go-testgen looks for a factory function (e.g., `NewClient`, `NewService`) that returns a pointer to the receiver type. If found, the test instantiates the receiver using the factory function with placeholder values matching each parameter's type:

- `string` → `"value"`
- `int`/`int64` → `0`
- `bool` → `false`
- Other types → `nil`

This produces idiomatic initialization instead of `&Type{}` or `Type{}`. Replace placeholders with actual test values before running tests.

## Test Styles

| Style | Description | When to use |
| - | - | - |
| `check` | Table-driven + closure check functions (default) | Methods, constructors, anything with multiple assertions or mock dependencies |
| `table` | Table-driven + `want` value fields | Simple functions where `DeepEqual` is sufficient |
| `simple` | Standalone test function, no table | Pure utilities, single-path functions |

See [Test Styles](../test-styles/index.md) for full details.

## Channel Types

go-testgen detects and handles `chan`, `chan<-`, and `<-chan` types in parameters and results. Channel placeholders use `nil` as the zero value.

See [Channel Type Support](../concepts/channels.md) for details.
