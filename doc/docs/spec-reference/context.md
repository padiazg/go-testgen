# `context` and `package_state`

!!! warning "Experimental"
    `gen-cases` and `.testspec.yaml` are experimental — an exploration of the approach. Schema, API, and behavior may change. Feedback welcome.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

These sections describe the **test body infrastructure** — how the subject under test is constructed and what shared variables the test runner needs. They are informational: `gen-cases` does not modify the test body, but an AI agent reads these sections to understand the setup when filling in `before` stubs.

## `context`

```yaml
context:
  subject_init:  <Go expression or description>
  shared_setup:  <description>
```

### `subject_init`

How the subject under test is constructed inside each test case. This expression typically appears inside the `t.Run` loop after the `before` hook is called.

**Examples:**

```yaml
# Constructor
context:
  subject_init: "e := New(tt.config)"

# From a table field
context:
  subject_init: "n := New(tt.config)"

# More complex initialization
context:
  subject_init: |
    c := &Config{OnError: registerError}
    e := New(c)
```

When `subject_init` is a multi-line string, each line is a separate statement in the setup block.

### `shared_setup`

Describes variables or infrastructure that must be set up **outside** the test case loop — before `for _, tt := range tests`. Also describes the sequence of calls the test body makes (e.g., `Start()`, `time.Sleep()`, `Stop()`).

`gen-cases` does not generate shared_setup code — this is documentation for the AI that fills in stubs, and for the human reading the spec.

**Example:**

```yaml
context:
  subject_init: |
    c := &Config{OnError: registerError}
    e := New(c)
  shared_setup: >
    Call clearErrors() before each case.
    Register each notifier from tt.notifiers via e.Register(n).
    Call e.Start(), time.Sleep(250ms), e.Stop(), time.Sleep(250ms).
```

The corresponding generated test body (written by a human or AI, not by `gen-cases`):

```go
func TestEngine_Start(t *testing.T) {
    tests := []struct{ ... }{ ... }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            clearErrors()                   // from shared_setup
            c := &Config{OnError: registerError}
            e := New(c)
            for _, n := range tt.notifiers {
                e.Register(n)
            }
            e.Start()
            time.Sleep(250 * time.Millisecond)
            e.Stop()
            time.Sleep(250 * time.Millisecond)
            for _, chk := range tt.checks {
                chk(t, e)
            }
        })
    }
}
```

---

## `package_state` {#package-state}

Describes **package-level variables** used to capture side effects between the test body and assertion checks. Common when the subject communicates errors or events via a callback.

```yaml
package_state:
  - name:                <identifier>
    type:                <Go type>
    clear_between_cases: true | false
    description:         <what it captures>
```

### Fields

| Field | Required | Description |
| - | - | - |
| `name` | yes | Go identifier for the variable (e.g. `errors`, `received`). |
| `type` | yes | Go type string (e.g. `"[]error"`, `"[]*model.Notification"`). |
| `clear_between_cases` | yes | `true` if the variable should be reset before each test case. |
| `description` | yes | What the variable captures and how it is verified. |

### Example

```yaml
package_state:
  - name: errors
    type: "[]error"
    clear_between_cases: true
    description: >
      Accumulates errors reported by the Config.OnError callback (registerError).
      Verified with hasErrors() and hasErrorsNotification().
```

Corresponding package-level Go code (written by human/AI):

```go
var errors []error

func registerError(err error) {
    errors = append(errors, err)
}

func clearErrors() {
    errors = nil
}
```

### When to use `package_state`

Use `package_state` when:

- The subject calls a **callback** to report errors or events (e.g. `Config.OnError`).
- The subject sends to a **channel** that needs draining between cases.
- The test body needs to **observe side effects** that cannot be captured through the function's return values.

Do not use it for values that can be returned from `before` — prefer `before.returns` for that pattern.

### Multiple state variables

A test can have multiple `package_state` entries:

```yaml
package_state:
  - name: errors
    type: "[]error"
    clear_between_cases: true
    description: Errors from OnError callback.

  - name: received
    type: "[]*model.Notification"
    clear_between_cases: true
    description: Notifications received by the dummy notifier.
```
