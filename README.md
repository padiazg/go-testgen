# testgen

Go CLI tool that generates unit test scaffolding in "padiazg style".

## Installation

```bash
go install github.com/padiazg/testgen/cmd/testgen@latest
```

Or build from source:

```bash
make build
make install
```

## Usage

```bash
# Generate test scaffolding for a constructor
testgen gen ./pkg/path New

# Generate test scaffolding for a method
testgen gen ./pkg/path ReceiverType.MethodName

# Debug: inspect parsed FuncInfo
testgen inspect ./pkg/path FuncName

# Flags
--output <file>   # output file (default: stdout)
--dry-run         # print without writing
--verbose        # show parsed FuncInfo
--style <path>   # config file (default: .testgen.yaml)
```

## Output style

Generates tests in this style:

```go
type checkXxxFn func(*testing.T, *Receiver, *Result)
var checkXxx = func(fns ...checkXxxFn) []checkXxxFn { return fns }

func TestReceiver_Method(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
        before  func(*Receiver)
        checks  []checkXxxFn
    }{
        {
            name: "TODO: success case",
            checks: checkXxx(),
        },
    }
    // ...
}
```

## Configuration

Create `.testgen.yaml` in your project root:

```yaml
receiver_var_name: "e"
result_var_name: "got"
error_var_name: "err"
use_testify: true
add_todo_cases: true
number_of_todos: 2
```