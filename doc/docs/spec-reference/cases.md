# Cases

`cases` is the primary content of a spec. Each entry describes one scenario and produces one struct literal in the `tests` slice.

## Schema

```yaml
cases:
  - name:        <test case name>
    description: <natural language scenario description>
    fields:
      <field_name>: <Go-syntax value>
    before:
      description: <setup description>
      mechanism:   <mechanism keyword>
      returns:
        type:    <Go return type>
        used_as: <how the returned value is used>
    after:
      description: <teardown description>
      mechanism:   <mechanism keyword>
    checks:
      - <check_id>(<args>)
    gates:
      <gate_field>: <value>
    todo: false | true
```

### Top-level case fields

| Field | Required | Description |
| - | - | - |
| `name` | yes | The string passed to `t.Run()`. Appears as `name: "..."` in the struct literal. |
| `description` | yes | Human-readable scenario description. Precise enough for an AI to generate the correct `before` code. |
| `fields` | no | Values for `table_fields` with role `input` or `state`. Keys are field names; values are Go-syntax strings. |
| `before` | no | Setup hook. See [`before`/`after` patterns](before-after.md). |
| `after` | no | Teardown hook. See [`before`/`after` patterns](before-after.md). |
| `checks` | no | List of check function call expressions. |
| `gates` | no | Values for `table_fields` with role `gate`. |
| `todo` | no | `true` emits a comment placeholder instead of a struct literal. |

---

## `name`

The case name becomes the first field in the struct literal and is the string `t.Run()` uses.

```yaml
- name: "connect-error"
```

Generated:

```go
{
    name: "connect-error",
    ...
},
```

**Conventions:**

- Use kebab-case: `"json-marshal-error"`, `"http-status-code-ok"`
- Describe the scenario, not the code path: `"returns-user-when-request-is-valid"` over `"success"`
- Keep it short enough to read in `go test -v` output

---

## `description`

A natural-language paragraph explaining exactly what this scenario tests. The description is used in two ways:

1. As `// ai-hint:` comments inside generated stubs — telling the AI what to implement.
2. As documentation for the human reading the spec.

```yaml
- name: "json-marshal-error"
  description: >
    JSON serialization of the payload fails. Deliver must return
    a Result with an error without attempting the HTTP request.
```

Be precise: name the method that fails, the error message returned, and the expected behavior.

---

## `fields`

Values for `table_fields` with role `input` or `state`. The key is the field name; the value is a Go expression string used verbatim in the generated struct literal.

```yaml
- name: "http-status-code-ok"
  fields:
    config:  '&Config{Endpoint: "http://localhost:8080/webhook", Headers: map[string]string{"Header-XYZ": "xyz"}}'
    message: '&model.Notification{Event: model.EventType("test-event"), Data: "test-data"}'
```

Generated:

```go
{
    name:    "http-status-code-ok",
    config:  &Config{Endpoint: "http://localhost:8080/webhook", Headers: map[string]string{"Header-XYZ": "xyz"}},
    message: &model.Notification{Event: model.EventType("test-event"), Data: "test-data"},
    ...
},
```

For multi-line values, use YAML block scalars:

```yaml
fields:
  notifiers: |
    []model.Notifier{
      dummy.New(&dummy.Config{Name: "dummy-01"}),
      dummy.New(&dummy.Config{Name: "dummy-02"}),
    }
```

Generated (after `go/format`):

```go
notifiers: []model.Notifier{
    dummy.New(&dummy.Config{Name: "dummy-01"}),
    dummy.New(&dummy.Config{Name: "dummy-02"}),
},
```

### Nil values

```yaml
fields:
  config: "nil"
  message: "nil"
```

---

## `checks`

A list of check function call expressions. Each string is a Go function call that gets inserted as an argument to the composer.

```yaml
checks:
  - hasErrors(true)
  - hasNotifiers(0)
```

Generated:

```go
checks: checkEngine(
    hasErrors(true),
    hasNotifiers(0),
),
```

For cross-package check types, `gen-cases` automatically adds the package qualifier:

```yaml
checks:
  - CheckResultError("error from json.Marshal")
```

Generated (with `package: github.com/padiazg/notifier/model`):

```go
checks: model.CheckResult(
    model.CheckResultError("error from json.Marshal"),
),
```

### Empty checks (AI fills in)

If `checks` is absent or empty, `gen-cases` emits an `// ai-hint:` comment inside the composer call:

```go
checks: checkEngine(
    // ai-hint: add checks for case "success"
    // All notifiers connect without error. Engine.Start() must
    // produce no errors in the `errors` package variable.
),
```

Use `--no-hints` to omit these comments entirely.

---

## `gates`

Values for `table_fields` with role `gate`. Gate fields control conditional logic in the test body (`if tt.wantPanic`, `if tt.wantLog != ""`).

```yaml
- name: "non-ok-status"
  gates:
    wantLog:   "webhook returned non-OK status: 403"
    wantPanic: false
    wantValue: true
```

Generated:

```go
{
    name:      "non-ok-status",
    wantLog:   "webhook returned non-OK status: 403",
    wantPanic: false,
    wantValue: true,
},
```

!!! tip "fields vs gates"
    Both `fields` and `gates` produce identical struct field assignments. The distinction is semantic. Use `fields` for function inputs and `gates` for test-body control signals. If a gate value lives naturally in `fields`, that works too — `gen-cases` checks both maps.

---

## `todo`

When `todo: true`, `gen-cases` emits a comment block instead of a compilable struct literal.

```yaml
- name: "partial-failure"
  todo: true
  description: >
    First notifier connects successfully, second fails with a network error.
    Engine.Start() must report the second error via OnError and continue.
  checks:
    - hasErrors(true)
```

Generated:

```go
// TODO: implement case "partial-failure"
// First notifier connects successfully, second fails with a network error.
// Engine.Start() must report the second error via OnError and continue.
// Suggested checks: hasErrors(true)
```

Use `todo: true` to:

- Reserve a slot for a case you know you need but haven't designed yet.
- Mark cases that require complex setup that will be implemented later.
- Generate a readable TODO list from the spec without breaking compilation.

---

## Minimal Case

The simplest possible case — no `before`, no extra fields, no checks yet:

```yaml
- name: "default"
  description: New(nil) creates an Engine with default values.
  fields:
    config: "nil"
  checks:
    - hasOnError(false)
    - hasNotifiers(0)
```

Generated:

```go
{
    name:   "default",
    config: nil,
    checks: checkEngine(
        hasOnError(false),
        hasNotifiers(0),
    ),
},
```

---

## Case with All Fields

```yaml
- name: "json-marshal-error"
  description: >
    JSON serialization of the payload fails. Deliver must return
    a Result with an error without attempting the HTTP request.
  fields:
    config: "nil"
  before:
    description: >
      Inject into n.jsonMarshal a function that returns
      fmt.Errorf("error from json.Marshal").
    mechanism: field-injection
  checks:
    - CheckResultError("error from json.Marshal")
```

Generated:

```go
{
    name:   "json-marshal-error",
    config: nil,
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
