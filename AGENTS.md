# AGENTS.md — testgen

Go CLI tool that generates unit test scaffolding in "padiazg style".

## Project state

Not yet initialized. Only `testgen-plan.md` exists (the spec).

## Quick start (when project is created)

```bash
# Initialize
go mod init github.com/padiazg/testgen

# Dependencies
go get github.com/spf13/cobra@latest
go get golang.org/x/tools/go/packages@latest
go get github.com/stretchr/testify@latest

# Build & test
go build -o bin/testgen ./cmd/testgen
go test ./... -v -count=1

# Lint
golangci-lint run ./...
```

## Architecture

```
testgen/
├── cmd/testgen/main.go     # Cobra CLI entrypoint
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
testgen gen <pkg/path> <FuncName>
testgen gen <pkg/path> <ReceiverType.MethodName>

# Debug: inspect parsed FuncInfo
testgen inspect <pkg/path> <FuncSpec>

# Flags
--output <file>   # output file (default: stdout)
--dry-run         # print without writing
--verbose         # show parsed FuncInfo
--style <path>   # config file (default: .testgen.yaml)
```

## Style conventions (must match)

- Check function type alias: `type checkXxxFn func(*testing.T, *Receiver, *Result)`
- Check var: `var checkXxx = func(fns ...checkXxxFn) []checkXxxFn { return fns }`
- Table field `before func(*Receiver)` for mock injection
- Error check: `checkXxxError(want string) checkXxxFn`
- Test cases have `wantErr bool` when function returns error

## Config

`.testgen.yaml` search order: flag → cwd → `$HOME/.testgen.yaml` → defaults.

## Reference repos

- `github.com/padiazg/notifier` — `engine/engine_test.go`, `connector/webhook/webhook_test.go`
- `github.com/padiazg/ollama-tools` — `models/ollama/model_test.go`

## Known limitations (v1)

- No generics support (show warning)
- No gRPC/proto reflection
- Uses `go/format.Source`, falls back if `goimports` unavailable

## Stage 8 — VSCode Extension

Optional. TypeScript extension that calls `testgen` CLI as child process.

**Architecture:** extension calls `testgen gen <pkgPath> <FuncSpec> --output <file>` via `child_process.execFile`.

**Prerequisite:** `testgen` must be installed:
```bash
go install github.com/padiazg/testgen/cmd/testgen@latest
# or locally
cd testgen && make install
```

**Project location:** `testgen-vscode/` (sibling to `testgen/`).

**Key files:**
- `package.json` — manifest with `testgen.binaryPath` config
- `src/funcDetector.ts` — regex-based Go func detection at cursor
- `src/testgenCli.ts` — child process wrapper
- `src/extension.ts` — registers command + Code Action provider

**Publish:**
```bash
npm install -g @vscode/vsce
vsce package
vsce publish  # requires Microsoft account
```