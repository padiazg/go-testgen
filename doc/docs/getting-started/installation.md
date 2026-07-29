# Installation

## Prerequisites

- Go 1.21 or later
- `goimports` (optional but recommended — used to sort imports in generated files)

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

## Install via curl (Recommended)

Install the pre-built binary with version info stamped via ldflags. Prefer this over `go install` which rebuilds from source and loses version/commit/buildDate stamping.

```bash
curl -fsSL https://padiazg.github.io/go-testgen/install.sh | sh
```

Or install a specific version:

```bash
curl -fsSL https://padiazg.github.io/go-testgen/install.sh | sh -s -- -v v0.2.1
```

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Install via `go install`

```bash
go install github.com/padiazg/go-testgen/cmd/go-testgen@latest
```

> **Note:** `go install` rebuilds the binary from source. The version will show as `v0.0.0 unknown unknown` because ldflags are not applied during `go install`. Use the curl installer above for release binaries with proper version info.

The binary is placed in `$GOPATH/bin` (or `$GOBIN` if set). Make sure that directory is in your `PATH`.

## Build from Source

```bash
git clone https://github.com/padiazg/go-testgen.git
cd go-testgen
make build    # outputs to go-testgen
make install  # installs to $GOPATH/bin
```

## Install via Homebrew (macOS and Linux)

```bash
brew tap padiazg/go-testgen
brew install go-testgen
```

Homebrew places the binary in its own prefix and adds it to `PATH` automatically. To upgrade later:

```bash
brew upgrade go-testgen
```

The tap repository is at [github.com/padiazg/homebrew-go-testgen](https://github.com/padiazg/homebrew-go-testgen).

## Verify Installation

```bash
go-testgen version
```

## AI Agent Skills

Install go-testgen AI agent skills to guide coding assistants in generating test cases:

```bash
curl -fsSL https://padiazg.github.io/go-testgen/skills.sh | bash
```

Installs `closure-check-tests` and `gen-test-cases` skills into `~/.agents/skills/`. See [AI Agent Skills](../workflow/adding-test-cases.md#option-a-use-ai-agent-skills-recommended) for usage details.

## Optional: Project Configuration File

Create `.go-testgen.yaml` in your project root to control generation behavior. See [Configuration](../configuration/index.md) for all options.

```yaml
receiver_var_name: "s"
result_var_name: "r"
use_testify: true
add_todo_cases: true
number_of_todos: 2
```
