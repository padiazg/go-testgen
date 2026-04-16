# Installation

## Prerequisites

- Go 1.21 or later
- `goimports` (optional but recommended — used to sort imports in generated files)

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

## Install via `go install`

```bash
go install github.com/padiazg/go-testgen@latest
```

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Build from Source

```bash
git clone https://github.com/padiazg/go-testgen.git
cd go-testgen
make build    # outputs to bin/go-testgen
make install  # installs to $GOPATH/bin
```

## Verify Installation

```bash
go-testgen version
```

## Optional: Project Configuration File

Create `.go-testgen.yaml` in your project root to control generation behavior. See [Configuration](../configuration/index.md) for all options.

```yaml
receiver_var_name: "s"
result_var_name: "r"
use_testify: true
add_todo_cases: true
number_of_todos: 1
```
