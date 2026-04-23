# Changelog

All notable changes to go-testgen will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.2.0 - [unreleased]

### Fixed

- Factory function instantiation now uses proper placeholder values instead of hardcoded `nil`

### Changed

- Fixed repo name in GitLab CI settings

### Added

- Gitea CI configuration files
- Homebrew install method documentation in docs

---

## v0.1.0 - 2026-04-16

### Added

#### Core Pipeline: analyze → generate → write

- **Three-stage pipeline**: AST analysis via `go/packages` → template-based scaffolding → file write with smart merge
- **`gen` command**: generate test scaffolding for a function or method
  - `FuncSpec` accepts plain function name (`New`, `CreateUser`) or receiver-qualified method (`Service.CreateUser`)
  - Auto-detects output file (`<source>_test.go`) when `--output` is omitted
  - `--dry-run` flag: prints generated code to stdout without writing
  - `--verbose` flag: dumps parsed `FuncInfo` JSON to stderr before generating
  - `--mock-from qualifier.InterfaceName`: generates a testify mock for the named interface (repeatable flag)
  - `-o -` writes to stdout explicitly
- **`inspect` command**: dumps `FuncInfo` JSON for a function — useful for diagnosing wrong signatures or missing types before running `gen`
- **`report` command**: scans all exported functions and methods in a package and shows test coverage status
  - `--format text|table|json` output modes
  - Shows interface dependencies and whether mock files exist
  - Prints exact `go-testgen gen` command to run for each untested function

#### Test Generation Styles

- **`check` style** (default): table-driven tests with closure-based check functions
  - Generates a `checkXxxFn` type alias and `checkXxx` variadic collector
  - Each assertion is an independent named closure composable per test case
  - `before func(*ReceiverType)` field in the table for mock setup and state mutation
  - Context parameters (`context.Context`) are injected automatically, not exposed in the table
- **`table` style**: table-driven tests with `want` value comparison fields
- **`simple` style**: standalone test functions without a table

#### Mock Generation

- `--mock-from qualifier.InterfaceName` generates a complete testify mock
- Written to `mock_<interfacename>_test.go` next to the test file
- All interface methods implemented; pointer/slice returns use comma-ok type assertion (`r, _ := args.Get(0).(*T)`) — safe for nil returns
- Existing mock files are never overwritten

#### Smart Merge

- If target `_test.go` exists but does not contain the test function: appends the new test and injects missing imports
- Never overwrites an existing test function

#### Static Analyzer (`internal/analyzer`)

- Loads packages via `golang.org/x/tools/go/packages`
- Extracts `FuncInfo`: name, package, import path, receiver, params, results, `HasError`, `HasContext`
- Resolves interface params — enables mock injection suggestions
- Extracts struct fields from receiver — used for constructor (`New`) check generation
- Resolves required imports for the generated test file

#### Generator (`internal/generator`)

- Template-based code generation using `text/template`
- Output formatted with `go/format.Source`; `goimports` applied when available in `PATH`
- `suggest` module: infers `go-testgen gen` commands for untested functions
- `factory` module: selects generator implementation based on style

#### Configuration (`internal/config`)

- Viper-based config from `.go-testgen.yaml`
- Search order: explicit `--style` flag → cwd → `$HOME` → hardcoded defaults
- Configurable: variable names, testify usage, TODO placeholders, check type suffix, mock/check generation

#### CLI (`cmd/`)

- Cobra-based CLI with `gen`, `inspect`, `report`, `version` subcommands
- `version` command reports build version

#### Build & Release

- GoReleaser configuration for multi-platform releases
- GitHub Actions CI/CD workflow
- Platform targets: Linux x86_64/arm64, macOS x86_64/arm64
- Static binaries (`CGO_ENABLED=0`)

---

## How to Install

```bash
go install github.com/padiazg/go-testgen@latest
```

Or build from source:

```bash
make build
make install
```
