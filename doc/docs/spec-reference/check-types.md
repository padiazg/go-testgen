# Check Types and Checks

!!! warning "Experimental"
    `gen-cases` and `.testspec.yaml` are experimental — an exploration of the approach. Schema, API, and behavior may change. Feedback welcome.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

These two sections form the **assertion vocabulary** of a spec. `check_types` declares the function types used for assertions; `checks` catalogues the individual assertion functions.

---

## `check_types`

A check type is a Go function type alias that defines the shape of a single assertion closure. It also names the **composer** — the variadic helper that collects assertions per test case.

```yaml
check_types:
  - id:          <identifier>
    type_name:   <Go type name>
    signature:   <Go func signature>
    composer:    <composer function name>
    package:     <import path — omit if same package>
    description: <what this type validates>
```

### Fields

| Field | Required | Description |
| - | - | - |
| `id` | yes | Internal identifier used to link checks to this type via `for_type`. |
| `type_name` | yes | Go type name (e.g. `engineTestCheckFn`, `TestCheckResultFn`). |
| `signature` | yes | Full Go function signature (e.g. `func(*testing.T, *Engine)`). |
| `composer` | yes | Name of the variadic collector function (e.g. `checkEngine`, `CheckResult`). |
| `package` | no | Import path if the type is defined in another package. Omit for same-package types. |
| `description` | yes | What the type asserts — shown in generated `// ai-hint:` comments. |

### Same-package check type

```yaml
check_types:
  - id: engine_check
    type_name: engineTestCheckFn
    signature: "func(*testing.T, *Engine)"
    composer: checkEngine
    description: Validates the Engine state after Start()/Stop().
```

Corresponds to:

```go
type engineTestCheckFn func(*testing.T, *Engine)

var checkEngine = func(fns ...engineTestCheckFn) []engineTestCheckFn { return fns }
```

### Cross-package check type

```yaml
check_types:
  - id: result_check
    type_name: TestCheckResultFn
    signature: "func(*testing.T, model.Notifier, *model.Result)"
    composer: CheckResult
    package: github.com/padiazg/notifier/model
    description: Validates the *model.Result returned by Deliver.
```

When `package` is set, `gen-cases` qualifies both the composer call and the check function calls with the package's last path segment:

```go
checks: model.CheckResult(
    model.CheckResultError("error from json.Marshal"),
),
```

### Multiple check types in one test

A single test can have multiple check types — e.g. one for the engine state and one for notifications:

```yaml
check_types:
  - id: notification_check
    type_name: notificationCheckFn
    signature: "func(*testing.T, *Engine, *model.Notification)"
    composer: checkNotifications
    description: Validates the engine state and the notification after dispatch.

  - id: error_check
    type_name: errorCheckFn
    signature: "func(*testing.T, error)"
    composer: checkError
    description: Validates the error returned by Dispatch.
```

---

## `checks`

The checks catalogue lists each assertion function available in this test. Each entry is linked to a check type via `for_type`.

```yaml
checks:
  - id:        <function name>
    for_type:  <check_type id>
    scope:     global | local
    signature: <full signature with return type>
    when:      <when to use this check>
    params:
      - name:           <param name>
        type:           <Go type>
        doc:            <what this param controls>
        sentinel_empty: <meaning of zero/empty value, if any>
    captures: [<var1>, <var2>]
```

### Fields

| Field | Required | Description |
| - | - | - |
| `id` | yes | The Go function name. Also the prefix used in case `checks` entries. |
| `for_type` | yes | The `check_type.id` this function belongs to. |
| `scope` | yes | `global` or `local` — see [Scope](#scope) below. |
| `signature` | yes | Full Go signature including return type (e.g. `hasErrors(has bool) engineTestCheckFn`). |
| `when` | yes | Description of when to use this check and what invariant it verifies. |
| `params` | no | Documentation for each parameter — used in `// ai-hint:` comments. |
| `captures` | no | Variables from the outer closure this check reads (relevant for `scope: local`). |

### Scope

**`scope: global`** — the function is defined at package scope (in `common_test.go`, the model package, or at the top of the test file). It does not close over test-local variables.

```yaml
- id: hasErrors
  for_type: engine_check
  scope: global
  signature: "hasErrors(has bool) engineTestCheckFn"
  when: >
    Verify whether the package-level `errors` variable has entries.
    Use has=true when a notifier fails to Connect().
    Use has=false when all notifiers connect correctly.
```

Corresponding Go:

```go
func hasErrors(has bool) engineTestCheckFn {
    return func(t *testing.T, e *Engine) {
        t.Helper()
        if has {
            assert.NotEmpty(t, errors, "hasErrors: expected at least one error")
        } else {
            assert.Empty(t, errors, "hasErrors: expected no errors")
        }
    }
}
```

**`scope: local`** — the function is defined **inside** `TestXxx`, before the `tests` slice. It closes over variables from `shared_setup` (e.g. a logger buffer, a captured value).

```yaml
- id: checkClientType
  for_type: notifier_check
  scope: local
  signature: "checkClientType(clientType interface{}) model.TestCheckNotifierFn"
  when: >
    Validate that n.client is of the expected type.
    Compare with reflect.TypeOf(clientType) vs reflect.TypeOf(n.client).
  captures: []
```

Corresponding Go (defined inside `TestWebhookNotifier_getClient`):

```go
checkClientType := func(clientType interface{}) model.TestCheckNotifierFn {
    return func(t *testing.T, n model.Notifier) {
        t.Helper()
        // ai-hint: implement assertion
        // compare reflect.TypeOf(clientType) vs reflect.TypeOf(n.(*WebhookNotifier).client)
    }
}
```

### `params` fields

| Field | Required | Description |
| - | - | - |
| `name` | yes | Parameter name as it appears in the function signature. |
| `type` | yes | Go type. |
| `doc` | yes | What this parameter controls. |
| `sentinel_empty` | no | The special meaning of the zero/empty value. Common pattern: empty string means "no error expected". |

#### Sentinel empty pattern

```yaml
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
```

Corresponding check implementation:

```go
func CheckResultError(want string) TestCheckResultFn {
    return func(t *testing.T, n Notifier, r *Result) {
        t.Helper()
        if want == "" {
            assert.NoErrorf(t, r.Error, "CheckResultError: expected no error, got %v", r.Error)
            return
        }
        if assert.Errorf(t, r.Error, "CheckResultError: expected error %q", want) {
            assert.Containsf(t, r.Error.Error(), want, "CheckResultError mismatch")
        }
    }
}
```

### `captures`

Lists variables from the outer test scope that this local check closes over. This is documentation for the AI — it helps the AI understand what variables are already in scope when implementing the check.

```yaml
- id: checkLogContains
  for_type: service_check
  scope: local
  signature: "checkLogContains(want string) serviceCheckFn"
  captures: [buf]
```

Means the check reads `buf` (e.g. a `bytes.Buffer` declared in `shared_setup`):

```go
checkLogContains := func(want string) serviceCheckFn {
    return func(t *testing.T, s *Service, err error) {
        t.Helper()
        assert.Contains(t, buf.String(), want)
    }
}
```
