# Changelog

All notable changes to go-testgen are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The full `CHANGELOG.md` is also available at the [repository root](https://github.com/padiazg/go-testgen/blob/main/CHANGELOG.md).

## v0.1.0 - [unreleased]

### Added

#### Core Pipeline: analyze → generate → write

- **Three-stage pipeline**: AST analysis via `go/packages` → template-based scaffolding → file write with smart merge
- **`gen` command**: generate test scaffolding for a function or method
    - `FuncSpec` accepts plain function name (`New`, `CreateUser`) or receiver-qualified method (`Service.CreateUser`)
    - Auto-detects output file (`<source>_test.go`) when `--output` is omitted
    - `--dry-run` flag: prints generated code to stdout without writing
    - `--verbose` flag: dumps parsed `FuncInfo` JSON to stderr before generating
    - `--mock-from qualifier.InterfaceName`: generates a testify mock for the named interface (repeatable)
    - `-o -` writes to stdout explicitly
- **`inspect` command**: dumps `FuncInfo` JSON for a function — useful for diagnosing wrong signatures or missing types
- **`report` command**: scans all exported functions and methods in a package and shows test coverage status
    - `--format text|table|json` output modes
    - Shows interface dependencies and mock file existence
    - Prints exact `go-testgen gen` command for each untested function

#### Test Generation Styles

- **`check` style** (default): table-driven tests with closure-based check functions
    - Generates a `checkXxxFn` type alias and `checkXxx` variadic collector
    - `before func(*ReceiverType)` field for per-case mock setup and state mutation
    - Context parameters injected automatically, not exposed in the table
- **`table` style**: table-driven tests with `want` value comparison fields
- **`simple` style**: standalone test functions without a table

#### Mock Generation

- `--mock-from qualifier.InterfaceName` generates a complete testify mock
- All interface methods implemented with correct comma-ok type assertions
- Existing mock files are never overwritten

#### Smart Merge

- If target `_test.go` exists but does not contain the test function: appends and injects missing imports
- Never overwrites an existing test function

#### Static Analyzer (`internal/analyzer`)

- Loads packages via `golang.org/x/tools/go/packages`
- Extracts `FuncInfo` including receiver, params, results, `HasError`, `HasContext`
- Resolves interface params and struct fields from receiver

#### Generator (`internal/generator`)

- Template-based code generation via `text/template`
- Output formatted with `go/format.Source`; `goimports` applied when available
- `suggest` module: infers `go-testgen gen` commands for untested functions
- `factory` module: selects generator by style

#### Configuration (`internal/config`)

- Viper-based `.go-testgen.yaml` config
- Search order: `--config` flag → cwd → `$HOME` → defaults

#### CLI (`cmd/`)

- Cobra-based with `gen`, `inspect`, `report`, `version` subcommands

#### Build & Release

- GoReleaser multi-platform builds (Linux x86_64/arm64, macOS x86_64/arm64)
- GitHub Actions CI/CD workflow
- Static binaries (`CGO_ENABLED=0`)
