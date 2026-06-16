# `before` and `after` Patterns

!!! warning "Experimental"
    `gen-cases` and `.testspec.yaml` are experimental — an exploration of the approach. Schema, API, and behavior may change. Feedback welcome.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

`before` and `after` describe the setup and teardown of each test case. Their concrete implementation varies widely depending on the subject under test. The spec captures the **intent** in natural language and a **mechanism** keyword; `gen-cases` generates a typed stub with `// ai-hint:` comments; an AI completes the body.

---

## `before`

```yaml
before:
  description: <what setup to perform — precise enough for AI to generate code>
  mechanism:   <mechanism keyword>
  returns:
    type:    <Go return type>
    used_as: <how the returned value is used in the test body>
```

### Fields

| Field | Required | Description |
| - | - | - |
| `description` | yes | Natural-language description of what the setup does. Name specific fields, mock methods, and return values. |
| `mechanism` | yes | Keyword identifying the pattern (see below). |
| `returns` | no | Present only when `before` returns a value the test body uses. |

### `before.returns`

| Field | Required | Description |
| - | - | - |
| `type` | yes | Go return type (e.g. `"*model.Notification"`, `"context.CancelFunc"`). |
| `used_as` | yes | How the test body uses the returned value (e.g. passed as an argument, stored for cleanup). |

---

## `before` Mechanisms

### `testify-mock`

The subject's dependency is a testify mock. `before` sets expectations on the mock using `.On()`, `.Return()`, and optionally `.Once()`, `.Maybe()`, `.Run()`.

```yaml
before:
  description: >
    Configure the mock UserRepository: CreateUser returns
    &userDomain.User{ID: "uuid-1", Name: "alice"} and nil error.
  mechanism: testify-mock
```

Generated stub:

```go
before: func(s *Service) {
    // ai-hint: testify-mock
    // Configure the mock UserRepository: CreateUser returns
    // &userDomain.User{ID: "uuid-1", Name: "alice"} and nil error.
},
```

AI fills in:

```go
before: func(s *Service) {
    s.repo.(*mockUserRepository).
        On("CreateUser", mock.Anything, mock.Anything).
        Return(&userDomain.User{ID: "uuid-1", Name: "alice"}, nil)
},
```

**When to use:** The subject has interface dependencies injected at construction. The mock is stored as a field on the receiver.

**Modifiers to mention in `description`:**

- `.Once()` — call expected exactly once (use when call order matters)
- `.Maybe()` — call may or may not happen (use for goroutine loops)
- `.Times(n)` — call expected exactly n times
- `.Run(func(args mock.Arguments){...})` — side effect that mutates a caller's buffer or channel

---

### `field-injection`

The subject's internal field (a function or a struct implementing an interface) is replaced with a test double directly on the receiver instance.

```yaml
before:
  description: >
    Inject into n.jsonMarshal a function that returns
    nil, fmt.Errorf("error from json.Marshal").
  mechanism: field-injection
```

Generated stub:

```go
before: func(n *WebhookNotifier) {
    // ai-hint: field-injection
    // Inject into n.jsonMarshal a function that returns
    // nil, fmt.Errorf("error from json.Marshal").
},
```

AI fills in:

```go
before: func(n *WebhookNotifier) {
    n.jsonMarshal = func(_ any) ([]byte, error) {
        return nil, fmt.Errorf("error from json.Marshal")
    }
},
```

**Another example — injecting a struct:**

```yaml
before:
  description: >
    Assign to n.client a mockHTTPClient whose DoFunc returns
    nil, errors.New("test http new request error").
  mechanism: field-injection
```

```go
before: func(n *WebhookNotifier) {
    n.client = &mockHTTPClient{
        DoFunc: func(req *http.Request) (*http.Response, error) {
            return nil, errors.New("test http new request error")
        },
    }
},
```

**When to use:** The subject stores its dependencies as struct fields (function fields or interface fields), not as constructor parameters with testify mocks.

---

### `field-reset`

A field on the receiver is set to `nil` or its zero value to force the subject into a specific code path.

```yaml
before:
  description: Force n.client = nil to ensure getClient() builds a new one.
  mechanism: field-reset
```

Generated stub:

```go
before: func(n *WebhookNotifier) {
    // ai-hint: field-reset
    // Force n.client = nil to ensure getClient() builds a new one.
},
```

AI fills in:

```go
before: func(n *WebhookNotifier) {
    n.client = nil
},
```

**When to use:** The subject lazily initializes a field; you want to test the initialization branch. Or you want to test nil-handling behaviour by removing a dependency.

---

### `state-mutation`

The subject's **internal state** is mutated directly — usually by acquiring a lock, modifying a slice or map, and then releasing the lock. The `before` function often **returns a value** that the test body passes to the function under test.

```yaml
before:
  description: >
    Create a *model.Notification with ID="payload-01", Event="test", Data=nil.
    Acquire d.lock, append the notification to d.in, release lock.
    Return the notification for the test body to pass to Exists().
  mechanism: state-mutation
  returns:
    type: "*model.Notification"
    used_as: "argument n passed to d.Exists(n) in the test body"
```

Generated stub:

```go
before: func(d *DummyNotifier) *model.Notification {
    // ai-hint: state-mutation
    // Create a *model.Notification with ID="payload-01", Event="test", Data=nil.
    // Acquire d.lock, append the notification to d.in, release lock.
    // Return the notification for the test body to pass to Exists().
    return nil // ai-hint: return the value described above
},
```

AI fills in:

```go
before: func(d *DummyNotifier) *model.Notification {
    n := &model.Notification{ID: "payload-01", Event: "test", Data: nil}
    d.lock.Lock()
    defer d.lock.Unlock()
    d.in = append(d.in, n)
    return n
},
```

**When to use:** The function under test reads from internal state that cannot be set through the constructor or public API. Common for `Exists()`, `Get()`, and similar read methods.

---

### `mixed`

The setup requires more than one pattern — e.g. both a field injection and a state mutation, or a mock setup combined with a direct field assignment.

```yaml
before:
  description: >
    Set n.client = &mockHTTPClient{} (field-injection).
    Also set n.retryCount = 3 to trigger the retry loop (field-injection).
    Configure the mock to fail twice then succeed (testify-mock).
  mechanism: mixed
```

Generated stub:

```go
before: func(n *WebhookNotifier) {
    // ai-hint: mixed
    // Set n.client = &mockHTTPClient{} (field-injection).
    // Also set n.retryCount = 3 to trigger the retry loop (field-injection).
    // Configure the mock to fail twice then succeed (testify-mock).
},
```

**When to use:** When no single mechanism covers the full setup. Document each sub-step clearly in `description` so the AI knows the complete picture.

---

### `none`

No setup needed. The test runs with the subject constructed as-is by `subject_init`.

```yaml
before:
  description: No setup needed — subject uses default construction.
  mechanism: none
```

When `mechanism: none` or `before` is absent from the case entirely, `gen-cases` **omits the `before` field** from the generated struct literal (or emits `before: nil` if the struct field type is a pointer).

**When to use:** Simple functions with no dependencies, constructor tests with nil config, or cases where the default state is the test condition.

---

## `before.returns`

When `before` returns a value that the test body passes to the function under test:

```yaml
before:
  description: >
    Create notification with ID="payload-01". Add to d.in with lock.
    Return the notification.
  mechanism: state-mutation
  returns:
    type: "*model.Notification"
    used_as: "argument n passed to d.Exists(n)"
```

`gen-cases` inspects the `before` field type from the existing AST to determine the full function signature. If the AST says `func(d *DummyNotifier) *model.Notification`, the generated stub is:

```go
before: func(d *DummyNotifier) *model.Notification {
    // ai-hint: state-mutation
    // Create notification with ID="payload-01". Add to d.in with lock. Return the notification.
    return nil // ai-hint: return the value described above
},
```

---

## `after`

```yaml
after:
  description: <what teardown to perform>
  mechanism:   stop-method | cancel-context | close-channel | mixed
```

`after` is present only when the function under test starts goroutines, holds channels, or acquires external state that must be released.

### `stop-method`

Call a `Stop()` method on the subject to terminate background goroutines.

```yaml
after:
  description: Call e.Stop() to terminate the engine's goroutines.
  mechanism: stop-method
```

Generated stub:

```go
after: func(e *Engine, cancel context.CancelFunc) {
    // ai-hint: stop-method
    // Call e.Stop() to terminate the engine's goroutines.
},
```

AI fills in:

```go
after: func(e *Engine, cancel context.CancelFunc) {
    e.Stop()
},
```

---

### `cancel-context`

Cancel a context to signal shutdown to goroutines that select on `ctx.Done()`.

```yaml
after:
  description: Cancel the context to stop the reader goroutine.
  mechanism: cancel-context
```

Generated:

```go
after: func(z *ZH07i, cancel context.CancelFunc) {
    // ai-hint: cancel-context
    // Cancel the context to stop the reader goroutine.
},
```

AI fills in:

```go
after: func(z *ZH07i, cancel context.CancelFunc) {
    cancel()
},
```

---

### `close-channel`

Close a channel to signal completion. The goroutine sending to the channel typically runs in the test body (via `go func(){}`), not in `after`.

```yaml
after:
  description: >
    Send a notification to n.Channel then close it
    to unblock the listener goroutine.
  mechanism: close-channel
```

Generated:

```go
after: func(n *Notifier) {
    // ai-hint: close-channel
    // Send a notification to n.Channel then close it to unblock the listener goroutine.
},
```

---

### `mixed`

Teardown requires multiple actions (e.g. cancel context **and** call Stop).

```yaml
after:
  description: Cancel the context, then call s.Stop() to drain the queue.
  mechanism: mixed
```

---

## Omitting `before` and `after`

If a case does not need setup or teardown, omit `before` and `after` entirely:

```yaml
cases:
  - name: "nil-config"
    description: New(nil) returns an engine with default values.
    fields:
      config: "nil"
    checks:
      - hasOnError(false)
      - hasNotifiers(0)
```

`gen-cases` omits the `before` and `after` fields from the generated struct literal.
