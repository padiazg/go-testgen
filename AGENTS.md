# AGENTS.md — go-testgen

Go CLI tool that generates unit test scaffolding in "padiazg style".

## Project state

Not yet initialized. Only `go-testgen-plan.md` exists (the spec).

## Quick start (when project is created)

```bash
# Initialize
go mod init github.com/padiazg/go-testgen

# Dependencies
go get github.com/spf13/cobra@latest
go get golang.org/x/tools/go/packages@latest
go get github.com/stretchr/testify@latest

# Build & test
go build -o bin/go-testgen ./cmd/go-testgen
go test ./... -v -count=1

# Lint
golangci-lint run ./...
```

## Architecture

```
go-testgen/
├── cmd/go-testgen/main.go     # Cobra CLI entrypoint
├── internal/
│   ├── analyzer/           # Static analysis via go/packages
│   ├── generator/          # Template engine (text/template)
│   └── config/             # YAML config loader
├── Makefile
└── go.mod
```

## Commands

```bash
# Generate test scaffolding
go-testgen gen <pkg/path> <FuncName>
go-testgen gen <pkg/path> <ReceiverType.MethodName>

# Debug: inspect parsed FuncInfo
go-testgen inspect <pkg/path> <FuncSpec>

# Flags
--output <file>   # output file (default: stdout)
--dry-run         # print without writing
--verbose         # show parsed FuncInfo
--style <path>   # config file (default: .go-testgen.yaml)
```

## Style conventions (must match)

- Check function type alias: `type checkXxxFn func(*testing.T, *Receiver, *Result)`
- Check var: `var checkXxx = func(fns ...checkXxxFn) []checkXxxFn { return fns }`
- Table field `before func(*Receiver)` for mock injection
- Error check: `checkXxxError(want string) checkXxxFn`
- Test cases have `wantErr bool` when function returns error

## Config

`.go-testgen.yaml` search order: flag → cwd → `$HOME/.go-testgen.yaml` → defaults.

## Reference repos

- `github.com/padiazg/notifier` — `engine/engine_test.go`, `connector/webhook/webhook_test.go`
- `github.com/padiazg/ollama-tools` — `models/ollama/model_test.go`

## Known limitations (v1)

- No generics support (show warning)
- No gRPC/proto reflection
- Uses `go/format.Source`, falls back if `goimports` unavailable

## Future enhancements (v2)

### Interface return type detection
Functions returning interfaces should suggest `check` style (not `table`), because:
- Interfaces are hard to compare as whole values
- Check-functions allow asserting individual aspects

**Implementation:**
- Add `ReturnsInterface bool` to `FuncSummary` in `internal/analyzer/scan_types.go`
- Populate in `scanner.go:buildFuncSummary` using `types` package to detect interface types
- Update `suggest.go` to return `StyleCheck` when `ReturnsInterface` is true

### Variadic parameter display
Variadic parameters (`args ...any`) should display correctly in signatures.

**Implementation:**
- Update `typeToString` in `analyzer/analyzer.go` to handle `*ast.Ellipsis`:
  ```go
  case *ast.Ellipsis:
      return "..." + typeToString(t.Elt)
  ```
- Sync the same fix to `scanner.go:typeToString`

### Unexported functions (black/white box tests)
Include unexported functions in reports, identified by a marker. This enables:
- **Black box tests**: Test exported functions only
- **White box tests**: Test unexported functions for internal behavior

**Implementation:**
- Add `IsExported bool` field to `FuncSummary` (done)
- Scanner now includes all functions, both exported and unexported
- Future: add marker in report output to distinguish exported vs unexported

## Stage 8 — VSCode Extension

Optional. TypeScript extension that calls `go-testgen` CLI as child process.

**Architecture:** extension calls `go-testgen gen <pkgPath> <FuncSpec> --output <file>` via `child_process.execFile`.

**Prerequisite:** `go-testgen` must be installed:
```bash
go install github.com/padiazg/go-testgen/cmd/go-testgen@latest
# or locally
cd go-testgen && make install
```

**Project location:** `go-testgen-vscode/` (sibling to `go-testgen/`).

**Key files:**
- `package.json` — manifest with `go-testgen.binaryPath` config
- `src/funcDetector.ts` — regex-based Go func detection at cursor
- `src/go-testgenCli.ts` — child process wrapper
- `src/extension.ts` — registers command + Code Action provider

**Publish:**
```bash
npm install -g @vscode/vsce
vsce package
vsce publish  # requires Microsoft account
```