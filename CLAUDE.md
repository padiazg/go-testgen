# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build      # build to bin/go-testgen
make test       # go test ./... -v -count=1
make lint       # golangci-lint run ./...
make install    # install to $GOPATH/bin
make clean      # remove bin/

go test ./internal/analyzer/... -run TestName  # single test
```

## Architecture

Three-stage pipeline: **analyze → generate → write**

```
cmd/go-testgen/main.go          # Cobra CLI: gen + inspect commands
internal/analyzer/           # AST analysis via golang.org/x/tools/go/packages
  analyzer.go                # Load() entry, AST traversal → FuncInfo
  funcinfo.go                # FuncInfo struct (params, results, receiver)
  merge.go                   # Merge generated tests into existing test files
internal/generator/          # Text/template test scaffolding
  generator.go               # FuncInfo + config → test code
internal/config/             # Viper-based YAML/JSON config
  config.go                  # Loads .go-testgen.yaml with defaults
go-testgen.go                   # Stub Engine/Config types (root package)
```

**gen flow**: config load → analyze package/func → generate → write/merge  
**inspect flow**: analyze only → output FuncInfo JSON (for debugging)

## Generated Test Pattern

Tests use a closure-based checker pattern (not standard `want` fields):

```go
type checkFooFn func(*testing.T, *ReceiverType, *ResultType)
var checkFoo = func(fns ...checkFooFn) []checkFooFn { return fns }

func TestReceiver_Method(t *testing.T) {
    tests := []struct {
        name    string
        before  func(*ReceiverType)
        checks  []checkFooFn
    }{...}
}
```

## Config (.go-testgen.yaml)

Key options affecting generation behavior:

```yaml
receiver_var_name: "e"
result_var_name: "got"
use_testify: true
add_todo_cases: true
number_of_todos: 2
check_type_suffix: "CheckFn"
generate_mocks: true
generate_checks: true
```

## CLI Usage

```bash
go-testgen gen ./pkg/path FuncName                    # function test
go-testgen gen ./pkg/path ReceiverType.MethodName     # method test
go-testgen gen ./pkg/path New                         # constructor test
go-testgen inspect ./pkg/path FuncName                # dump FuncInfo JSON
go-testgen gen ./pkg/path FuncName --dry-run          # preview only
```

## Key Files for Context

- `go-testgen-plan.md` — detailed design decisions and implementation notes
- `AGENTS.md` — agent/tool integration guidance
