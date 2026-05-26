# Table Fields

`table_fields` declares the columns in the anonymous test struct beyond the standard `name`, `before`, `after`, and `checks` fields. Each field has a **role** that controls how `gen-cases` uses its value when generating a case entry.

## Schema

```yaml
table_fields:
  - name: <field name>
    type: <Go type>
    role: input | gate | state
    doc:  <description>
```

### Fields

| Field | Required | Description |
| - | - | - |
| `name` | yes | Go field name as it appears in the test struct. |
| `type` | yes | Go type string. |
| `role` | yes | `input`, `gate`, or `state` — see [Roles](#roles) below. |
| `doc` | yes | Description of what this field controls. |

---

## Roles

### `role: input`

The value is **passed to the function under test** or to its constructor. It becomes an argument or an input to `subject_init`.

```yaml
table_fields:
  - name: config
    type: "*Config"
    role: input
    doc: Config for the WebhookNotifier. nil uses defaults.

  - name: message
    type: "*model.Notification"
    role: input
    doc: Notification to deliver. nil is valid (some cases do not need it).
```

Test struct:

```go
tests := []struct {
    name    string
    config  *Config
    message *model.Notification
    checks  []checkFn
}{...}
```

Case entry when `fields.config` and `fields.message` are provided:

```go
{
    name:    "http-status-code-ok",
    config:  &Config{Endpoint: "http://localhost:8080/webhook"},
    message: &model.Notification{Event: model.EventType("test-event"), Data: "test-data"},
    checks:  model.CheckResult(model.CheckResultError("")),
},
```

Case entry when `fields.config` is provided but `fields.message` is absent:

```go
{
    name:    "json-marshal-error",
    config:  nil,
    message: nil, // ai-hint: set message for this case
    checks:  model.CheckResult(model.CheckResultError("error from json.Marshal")),
},
```

---

### `role: gate`

The value **controls conditional logic in the test body**. Gate fields are checked with `if tt.wantPanic`, `if tt.wantValue`, etc. rather than through closure checks.

```yaml
table_fields:
  - name: wantLog
    type: string
    role: gate
    doc: Expected substring in the log output. Empty means no log expected.

  - name: wantPanic
    type: bool
    role: gate
    doc: Whether the function is expected to panic.
```

Gate values are sourced from `case.gates`:

```yaml
cases:
  - name: "non-ok-status"
    gates:
      wantLog: "webhook returned non-OK status: 403"
      wantPanic: false
```

Generated:

```go
{
    name:      "non-ok-status",
    wantLog:   "webhook returned non-OK status: 403",
    wantPanic: false,
},
```

!!! note "Fields vs Gates"
    `case.fields` and `case.gates` both produce the same kind of assignment in the generated struct literal. The distinction is semantic — it tells the AI and the spec reader whether the value is an input to the function or a control signal for the test body.

    Some specs put gate values in `case.fields` rather than `case.gates`. Both work.

---

### `role: state`

The value is **passed to `before` or `after`** to set up or tear down test state. State fields appear in the table but their values feed into the setup hook rather than the function under test directly.

```yaml
table_fields:
  - name: before
    type: "func(d *DummyNotifier) *model.Notification"
    role: state
    doc: >
      Prepares internal state of d and returns the notification to pass to Exists().
      The signature returns *model.Notification because the search value depends on setup.
```

!!! note "before as a table field"
    When `before` appears in `table_fields` with `role: state`, it means the `before` function **varies per case** and is declared as a table column rather than a standard struct field. This pattern is used when `before` has a non-standard signature (e.g. it returns a value used by the test body) and the value differs meaningfully per case.

    In the standard closure-check pattern, `before` is **not** in `table_fields` — it is a struct field with a fixed signature and its stub is generated from the AST.

---

## Complete Example

```yaml
table_fields:
  - name: notifiers
    type: "[]model.Notifier"
    role: input
    doc: Notifiers to register in the engine before the test.

  - name: message
    type: "*model.Notification"
    role: input
    doc: Notification to dispatch. nil is a valid case.

  - name: wantErr
    type: bool
    role: gate
    doc: Whether an error is expected from Dispatch.
```

Generates a test struct:

```go
tests := []struct {
    name      string
    notifiers []model.Notifier
    message   *model.Notification
    wantErr   bool
    checks    []notificationCheckFn
}{...}
```

And a case:

```go
{
    name: "fail-missing-channel",
    notifiers: []model.Notifier{
        dummy.New(&dummy.Config{Name: "dummy-01"}),
        dummy.New(&dummy.Config{Name: "dummy-02"}),
    },
    message: &model.Notification{
        ID: "msg-01", Event: model.EventType("test"),
        Channels: []string{"dummy-03"},
    },
    wantErr: true,
    checks:  checkNotifications(hasErrorsNotification(true)),
},
```

---

## When to Omit Table Fields

Not all tests need extra table fields. The minimal test struct has only `name`, `before`, and `checks`. Omit `table_fields` entirely if:

- The function under test takes no user-controlled inputs (e.g. it reads from `package_state` or fixed config).
- All inputs come from `before` (field injection or mock setup) rather than from the table.
- The function has a single, fixed call pattern per test case.
