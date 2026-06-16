# `gen-cases` — Materialize Test Cases from a Spec

`gen-cases` reads a `.testspec.yaml` file and inserts the described test case entries into the `tests` slice of an existing `_test.go`. The test scaffolding must already exist — run [`gen`](gen.md) first.


!!! warning "Experimental"
    This command is experimental. See the [Spec Reference](../spec-reference/index.md) for format details and limitations.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

## Syntax

```bash
go-testgen gen-cases <spec-file> [flags]
```

`spec-file` is the path to a `.testspec.yaml` file.

## Examples

```bash
# Preview output without writing (dry-run)
go-testgen gen-cases --dry-run ./engine/engine_start.testspec.yaml

# Write cases to the resolved test file
go-testgen gen-cases ./engine/engine_start.testspec.yaml

# Override output path
go-testgen gen-cases -o ./engine/engine_test.go ./engine/engine_start.testspec.yaml

# Replace existing entries (re-sync from an updated spec)
go-testgen gen-cases --force ./engine/engine_start.testspec.yaml

# Omit ai-hint comments (plain stubs only)
go-testgen gen-cases --no-hints ./engine/engine_start.testspec.yaml

# Show what was generated vs skipped
go-testgen gen-cases --verbose ./engine/engine_start.testspec.yaml
```

## Flags

| Flag | Default | Description |
| - | - | - |
| `--dry-run` | false | Print generated code to stdout without modifying any file. |
| `-o`, `--output` | auto | Override the output `_test.go` path. |
| `--force` | false | Replace existing entries instead of skipping them. |
| `--no-hints` | false | Omit `// ai-hint:` comments from output. |
| `-v`, `--verbose` | false | Print a summary of generated/skipped entries. |

## The `.testspec.yaml` Format

A spec file describes **what to test** in domain terms — without writing code. Example:

```yaml
version: "1"
package: ./engine
function: Engine.Start

context:
  subject_init: |
    c := &Config{OnError: registerError}
    e := New(c)

check_types:
  - id: engine_check
    type_name: engineTestCheckFn
    composer: checkEngine

table_fields:
  - name: notifiers
    type: "[]model.Notifier"
    role: input

cases:
  - name: "connect-error"
    description: >
      A notifier with ConnectError configured fails to connect.
      Engine.Start() should report the error via OnError.
    fields:
      notifiers: |
        []model.Notifier{
          &dummy.DummyNotifier{Config: &dummy.Config{
            Name: "dummy-01", ConnectError: fmt.Errorf("connecting"),
          }},
        }
    checks:
      - hasErrors(true)

  - name: "success"
    description: A notifier without ConnectError connects successfully.
    fields:
      notifiers: |
        []model.Notifier{
          &dummy.DummyNotifier{Config: &dummy.Config{Name: "dummy-01"}},
        }
    checks:
      - hasErrors(false)
```

## What Gets Generated

For each case in the spec, `gen-cases` inserts a struct literal into the `tests` slice:

```go
{
    name: "connect-error",
    notifiers: []model.Notifier{
        &dummy.DummyNotifier{Config: &dummy.Config{
            Name:         "dummy-01",
            ConnectError: fmt.Errorf("connecting"),
        }},
    },
    checks: checkEngine(
        hasErrors(true),
    ),
},
```

For cases with a `before` hook, it generates a function stub:

```go
{
    name: "json-marshal-error",
    before: func(n *WebhookNotifier) {
        // ai-hint: field-injection
        // Inject into n.jsonMarshal a function that returns
        // fmt.Errorf("error from json.Marshal").
    },
    checks: model.CheckResult(
        model.CheckResultError("error from json.Marshal"),
    ),
},
```

The `// ai-hint:` comments tell a generative AI exactly what to implement. Use `--no-hints` to omit them.

## Target File Resolution

The `_test.go` to modify is resolved in this order:

1. `--output` flag, if provided.
2. `test_file` field in the spec, if present.
3. Derived automatically: loads the package with `go/packages`, converts the receiver name to snake_case, and appends `_test.go`.

Examples:

| `function` | Derived filename |
| - | - |
| `Engine.Start` | `engine_test.go` |
| `WebhookNotifier.Deliver` | `webhook_notifier_test.go` |
| `DummyNotifier.Exists` | `dummy_notifier_test.go` |
| `NewEngine` | `engine_test.go` |

If the file does not exist, `gen-cases` returns an error — run `gen` first to create the scaffold.

## Idempotency

`gen-cases` never inserts duplicate entries. Before inserting each case, it checks whether a struct literal with that `name` key already exists in the `tests` slice.

- Without `--force`: existing entries are skipped with a warning (`--verbose` shows which ones).
- With `--force`: existing entries are replaced with the regenerated version from the spec.

Running `gen-cases` twice on the same spec and file produces identical output.

## Pipeline Position

```
go-testgen gen          → _test.go scaffold (struct + TODO cases)
  ↓
author .testspec.yaml   → domain scenarios in YAML
  ↓
go-testgen gen-cases    → struct literals inserted with ai-hint stubs
  ↓
AI fills in before/values reading spec + source
  ↓
go test ./...
```
