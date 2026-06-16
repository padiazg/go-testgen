# Complete Examples

!!! warning "Experimental"
    `gen-cases` and `.testspec.yaml` are experimental — an exploration of the approach. Schema, API, and behavior may change. Feedback welcome.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

Three worked examples covering the most common patterns: global checks with package state, cross-package check types, and state-mutation with a returning `before`.

---

## Engine.Start — global checks + package state

Tests a method that starts goroutines and reports errors via a callback. Uses package-level state to capture errors.

### Spec

```yaml
version: "1"
package: ./engine
function: Engine.Start

context:
  subject_init: |
    c := &Config{OnError: registerError}
    e := New(c)
  shared_setup: >
    Call clearErrors() before each case.
    Register each notifier from tt.notifiers via e.Register(n).
    Call e.Start(), time.Sleep(250ms), e.Stop(), time.Sleep(250ms).

package_state:
  - name: errors
    type: "[]error"
    clear_between_cases: true
    description: >
      Accumulates errors reported by the Config.OnError callback (registerError).
      Verified with hasErrors().

table_fields:
  - name: notifiers
    type: "[]model.Notifier"
    role: input
    doc: >
      Notifiers to register. DummyNotifier.Config.ConnectError controls
      whether Connect() fails.

check_types:
  - id: engine_check
    type_name: engineTestCheckFn
    signature: "func(*testing.T, *Engine)"
    composer: checkEngine
    description: Validates the engine state after Start()/Stop().

checks:
  - id: hasErrors
    for_type: engine_check
    scope: global
    signature: "hasErrors(has bool) engineTestCheckFn"
    when: >
      Verify whether the package-level `errors` variable has entries.
      Use has=true when a notifier fails in Connect().
      Use has=false when all notifiers connect correctly.
    params:
      - name: has
        type: bool
        doc: "true = expects at least one error; false = expects empty slice"

cases:
  - name: "connect-error"
    description: >
      A notifier with ConnectError configured fails to connect.
      Engine.Start() must report the error via OnError without panicking
      or blocking other notifiers.
    fields:
      notifiers: |
        []model.Notifier{
          &dummy.DummyNotifier{
            Config: &dummy.Config{
              Name:         "dummy-01",
              ConnectError: fmt.Errorf("connecting"),
            },
          },
        }
    checks:
      - hasErrors(true)

  - name: "success"
    description: >
      A notifier without ConnectError connects successfully.
      Engine.Start() must produce no errors.
    fields:
      notifiers: |
        []model.Notifier{
          &dummy.DummyNotifier{
            Config: &dummy.Config{Name: "dummy-01"},
          },
        }
    checks:
      - hasErrors(false)
```

### Generated output

```go
func TestEngine_Start(t *testing.T) {
    tests := []struct {
        name      string
        notifiers []model.Notifier
        checks    []engineTestCheckFn
    }{
        {
            name: "connect-error",
            notifiers: []model.Notifier{
                &dummy.DummyNotifier{
                    Config: &dummy.Config{
                        Name:         "dummy-01",
                        ConnectError: fmt.Errorf("connecting"),
                    },
                },
            },
            checks: checkEngine(
                hasErrors(true),
            ),
        },
        {
            name: "success",
            notifiers: []model.Notifier{
                &dummy.DummyNotifier{
                    Config: &dummy.Config{Name: "dummy-01"},
                },
            },
            checks: checkEngine(
                hasErrors(false),
            ),
        },
    }
    // ... test body (not modified by gen-cases)
}
```

---

## WebhookNotifier.Deliver — cross-package checks + field injection

Tests a method that calls external HTTP. Uses a cross-package check type and field injection to replace internal functions.

### Spec

```yaml
version: "1"
package: ./connector/webhook
function: WebhookNotifier.Deliver

context:
  subject_init: "n := New(tt.config)"

table_fields:
  - name: config
    type: "*Config"
    role: input
    doc: Config for the WebhookNotifier. nil uses defaults.
  - name: message
    type: "*model.Notification"
    role: input
    doc: Notification to deliver.

check_types:
  - id: result_check
    type_name: TestCheckResultFn
    signature: "func(*testing.T, model.Notifier, *model.Result)"
    composer: CheckResult
    package: github.com/padiazg/notifier/model
    description: Validates the *model.Result returned by Deliver.

checks:
  - id: CheckResultError
    for_type: result_check
    scope: global
    signature: "model.CheckResultError(want string) model.TestCheckResultFn"
    when: >
      Use in all Deliver cases to validate result.Error.
      want="" validates success (result.Error == nil).
      want="substring" validates that result.Error.Error() contains that substring.
    params:
      - name: want
        type: string
        doc: Expected error substring.
        sentinel_empty: "empty string asserts result.Error is nil (success)"

cases:
  - name: "json-marshal-error"
    description: >
      JSON serialization of the payload fails. Deliver must return
      a Result with an error without attempting the HTTP request.
    fields:
      config: "nil"
    before:
      description: >
        Inject into n.jsonMarshal a function that returns
        nil, fmt.Errorf("error from json.Marshal").
      mechanism: field-injection
    checks:
      - CheckResultError("error from json.Marshal")

  - name: "http-newrequest-error"
    description: >
      http.NewRequest fails. Deliver must return a Result with an error
      before executing the request.
    fields:
      config: "nil"
    before:
      description: >
        Inject into n.httpNewRequest a function that returns
        nil, fmt.Errorf("test error on http.NewRequest").
      mechanism: field-injection
    checks:
      - CheckResultError("test error on http.NewRequest")

  - name: "client-do-error"
    description: The HTTP client fails to execute the request. Deliver propagates the error.
    before:
      description: >
        Assign to n.client a mockHTTPClient whose DoFunc returns
        nil, errors.New("test http new request error").
      mechanism: field-injection
    checks:
      - CheckResultError("test http new request error")

  - name: "http-status-code-not-ok"
    description: >
      The endpoint returns HTTP 403 Forbidden. Deliver must interpret
      the non-OK status as an error and return it in the Result.
    fields:
      config: '&Config{Endpoint: "http://localhost:8080/webhook"}'
      message: '&model.Notification{Event: model.EventType("test-event"), Data: "test-data"}'
    before:
      description: >
        Assign to n.client a mockHTTPClient whose DoFunc returns
        &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(...)}
        with body string "Forbidden", and nil error.
      mechanism: field-injection
    checks:
      - CheckResultError("webhook returned non-OK status: 403")

  - name: "http-status-code-ok"
    description: >
      The endpoint returns HTTP 200 OK. Deliver completes successfully.
      The Result must not contain an error.
    fields:
      config: '&Config{Endpoint: "http://localhost:8080/webhook", Headers: map[string]string{"Header-XYZ": "xyz"}}'
      message: '&model.Notification{Event: model.EventType("test-event"), Data: "test-data"}'
    before:
      description: >
        Assign to n.client a mockHTTPClient whose DoFunc returns
        &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(...)}
        with body string "Ok", and nil error.
      mechanism: field-injection
    checks:
      - CheckResultError("")
```

### Generated output (excerpt)

```go
{
    name:   "json-marshal-error",
    config: nil,
    before: func(n *WebhookNotifier) {
        // ai-hint: field-injection
        // Inject into n.jsonMarshal a function that returns
        // nil, fmt.Errorf("error from json.Marshal").
    },
    checks: model.CheckResult(
        model.CheckResultError("error from json.Marshal"),
    ),
},
{
    name:    "http-status-code-not-ok",
    config:  &Config{Endpoint: "http://localhost:8080/webhook"},
    message: &model.Notification{Event: model.EventType("test-event"), Data: "test-data"},
    before: func(n *WebhookNotifier) {
        // ai-hint: field-injection
        // Assign to n.client a mockHTTPClient whose DoFunc returns
        // &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(...)}
        // with body string "Forbidden", and nil error.
    },
    checks: model.CheckResult(
        model.CheckResultError("webhook returned non-OK status: 403"),
    ),
},
{
    name:    "http-status-code-ok",
    config:  &Config{Endpoint: "http://localhost:8080/webhook", Headers: map[string]string{"Header-XYZ": "xyz"}},
    message: &model.Notification{Event: model.EventType("test-event"), Data: "test-data"},
    before: func(n *WebhookNotifier) {
        // ai-hint: field-injection
        // Assign to n.client a mockHTTPClient whose DoFunc returns
        // &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(...)}
        // with body string "Ok", and nil error.
    },
    checks: model.CheckResult(
        model.CheckResultError(""),
    ),
},
```

---

## DummyNotifier.Exists — state-mutation with returning `before`

Tests a method that reads from internal state. Uses `before` as a `table_field` (not the standard struct field) with a return value — the notification prepared by setup is passed directly to `Exists()`.

### Spec

```yaml
version: "1"
package: ./connector/dummy
function: DummyNotifier.Exists

context:
  subject_init: "d := New(&Config{})"

table_fields:
  - name: before
    type: "func(d *DummyNotifier) *model.Notification"
    role: state
    doc: >
      Prepares internal state of d and returns the notification to pass
      to Exists(). The signature returns *model.Notification because the
      search value depends on setup.
  - name: want
    type: bool
    role: gate
    doc: Expected result of d.Exists(n).

cases:
  - name: "found"
    description: >
      The notification is in d.in (inserted manually with lock).
      Exists() must return true.
    fields:
      want: "true"
    before:
      description: >
        Create a *model.Notification with ID="payload-01", Event="test", Data=nil.
        Acquire d.lock, append the notification to d.in, release lock.
        Return the same notification so the test can pass it to Exists().
      mechanism: state-mutation
      returns:
        type: "*model.Notification"
        used_as: "argument n passed to d.Exists(n) in the test body"

  - name: "not-found"
    description: >
      The notification is NOT in d.in — d.in is empty.
      Exists() must return false.
    fields:
      want: "false"
    before:
      description: >
        Create a *model.Notification with ID="payload-01", Event="test", Data=nil
        but do NOT append it to d.in. Return the notification so the test
        can pass it to Exists() and verify it is not found.
      mechanism: state-mutation
      returns:
        type: "*model.Notification"
        used_as: "argument n passed to d.Exists(n) in the test body"
```

### Generated output

```go
func TestDummyNotifier_Exists(t *testing.T) {
    tests := []struct {
        name   string
        before func(d *DummyNotifier) *model.Notification
        want   bool
    }{
        {
            name: "found",
            before: func(d *DummyNotifier) *model.Notification {
                // ai-hint: state-mutation
                // Create a *model.Notification with ID="payload-01", Event="test", Data=nil.
                // Acquire d.lock, append the notification to d.in, release lock.
                // Return the same notification so the test can pass it to Exists().
                return nil // ai-hint: return the value described above
            },
            want: true,
        },
        {
            name: "not-found",
            before: func(d *DummyNotifier) *model.Notification {
                // ai-hint: state-mutation
                // Create a *model.Notification with ID="payload-01", Event="test", Data=nil
                // but do NOT append it to d.in. Return the notification so the test
                // can pass it to Exists() and verify it is not found.
                return nil // ai-hint: return the value described above
            },
            want: false,
        },
    }
    // ... test body (not modified by gen-cases)
}
```

---

## WebhookNotifier.getClient — local checks + multiple `before` mechanisms

Tests a private method that lazily builds an HTTP client. Uses local checks (defined inside `TestXxx`) and combines `field-injection` and `field-reset` patterns.

### Spec (excerpt)

```yaml
version: "1"
package: ./connector/webhook
function: WebhookNotifier.getClient

context:
  subject_init: "n := New(tt.config)"

table_fields:
  - name: config
    type: "*Config"
    role: input
    doc: Config for the WebhookNotifier.
  - name: before
    type: "func(*WebhookNotifier)"
    role: state
    doc: Setup before calling getClient().

check_types:
  - id: notifier_check
    type_name: TestCheckNotifierFn
    signature: "func(*testing.T, model.Notifier)"
    composer: CheckNotifier
    package: github.com/padiazg/notifier/model

checks:
  - id: checkClientType
    for_type: notifier_check
    scope: local
    signature: "checkClientType(clientType interface{}) model.TestCheckNotifierFn"
    when: Validate that n.client is of the expected type using reflect.TypeOf.

  - id: checkClientInsecure
    for_type: notifier_check
    scope: local
    signature: "checkClientInsecure(want bool) model.TestCheckNotifierFn"
    when: >
      Validate n.client.(*http.Client).Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify.
      Only apply when checkClientType(&http.Client{}) is also present.

cases:
  - name: "existing-client"
    description: >
      If n.client is already assigned (a mock), getClient() must return it as-is
      without constructing a new one.
    before:
      description: Assign n.client = &mockHTTPClient{} before calling getClient().
      mechanism: field-injection
    checks:
      - checkClientType(&mockHTTPClient{})

  - name: "empty-client"
    description: >
      If n.client is nil, getClient() builds a standard *http.Client
      with no TLS configuration (InsecureSkipVerify=false).
    before:
      description: Force n.client = nil to ensure getClient() constructs a new one.
      mechanism: field-reset
    checks:
      - checkClientType(&http.Client{})
      - checkClientInsecure(false)

  - name: "insecure-client"
    description: >
      With Config.Insecure=true, getClient() builds a *http.Client with
      TLSClientConfig.InsecureSkipVerify=true.
    fields:
      config: "&Config{Insecure: true}"
    before:
      description: Force n.client = nil so getClient() builds the client respecting Config.Insecure.
      mechanism: field-reset
    checks:
      - checkClientType(&http.Client{})
      - checkClientInsecure(true)
```

### Generated output (excerpt)

```go
{
    name: "existing-client",
    before: func(n *WebhookNotifier) {
        // ai-hint: field-injection
        // Assign n.client = &mockHTTPClient{} before calling getClient().
    },
    checks: model.CheckNotifier(
        checkClientType(&mockHTTPClient{}),
    ),
},
{
    name: "empty-client",
    before: func(n *WebhookNotifier) {
        // ai-hint: field-reset
        // Force n.client = nil to ensure getClient() constructs a new one.
    },
    checks: model.CheckNotifier(
        checkClientType(&http.Client{}),
        checkClientInsecure(false),
    ),
},
{
    name:   "insecure-client",
    config: &Config{Insecure: true},
    before: func(n *WebhookNotifier) {
        // ai-hint: field-reset
        // Force n.client = nil so getClient() builds the client respecting Config.Insecure.
    },
    checks: model.CheckNotifier(
        checkClientType(&http.Client{}),
        checkClientInsecure(true),
    ),
},
```
