# `.testspec.yaml` — Spec Reference

A `.testspec.yaml` file describes **what to test** in domain terms, without writing Go code. `go-testgen gen-cases` reads it and materializes the described cases into the `tests` slice of an existing `_test.go`.

## Design Principle

> The concrete values in `before`, `after`, and checks are **circumstantial** — they depend on how the subject under test is built, what dependencies it has, and how injection works. The spec describes them in natural language. A generative AI translates them to Go code by reading the spec alongside the source and the generated `_test.go`.

A spec author works in the **domain space** (invariants, scenarios, business rules). An AI completes the **implementation space** (field names, mock patterns, exact literals).

## Full Schema

```yaml
version: "1"                          # required, always "1"

# ── Identification ────────────────────────────────────────────────────────────
package:   <go package pattern>        # required — e.g. ./connector/webhook
function:  <FuncSpec>                  # required — e.g. WebhookNotifier.Deliver
test_file: <path>                      # optional — override target _test.go

# ── Test context ──────────────────────────────────────────────────────────────
context:
  subject_init:  <Go expression or description>
  shared_setup:  <description>

package_state:
  - name:                <identifier>
    type:                <Go type>
    clear_between_cases: true | false
    description:         <what it captures>

# ── Fixtures ──────────────────────────────────────────────────────────────────
fixtures:
  - name:        <identifier>
    type:        <Go type>
    description: <what it represents>
    value: |
      <Go literal — may be multiline>

# ── Check types ───────────────────────────────────────────────────────────────
check_types:
  - id:          <identifier used in cases>
    type_name:   <Go type name>
    signature:   <func signature>
    composer:    <composer function name>
    package:     <import path — omit if same package>
    description: <what this type validates>

# ── Checks ────────────────────────────────────────────────────────────────────
checks:
  - id:        <function name>
    for_type:  <check_type id>
    scope:     global | local
    signature: <full signature with return type>
    when:      <when to use this check>
    params:
      - name:           <param name>
        type:           <Go type>
        doc:            <what this param represents>
        sentinel_empty: <meaning of zero/empty value, if any>
    captures: [<var1>, <var2>]

# ── Table fields ──────────────────────────────────────────────────────────────
table_fields:
  - name: <field name>
    type: <Go type>
    role: input | gate | state
    doc:  <description>

# ── Cases ─────────────────────────────────────────────────────────────────────
cases:
  - name:        <test case name — appears in t.Run()>
    description: <natural language scenario description>
    fields:
      <field_name>: <Go-syntax value>
    before:
      description: <what setup to perform>
      mechanism:   testify-mock | field-injection | field-reset | state-mutation | mixed | none
      returns:
        type:    <Go return type>
        used_as: <how the returned value is used>
    after:
      description: <what teardown to perform>
      mechanism:   stop-method | cancel-context | close-channel | mixed
    checks:
      - <check_id>(<args>)
    gates:
      <gate_field>: <value>
    todo: false | true
```

## Top-Level Fields

| Field | Required | Description |
| - | - | - |
| `version` | yes | Always `"1"`. |
| `package` | yes | Go package pattern passed to `go/packages`. E.g. `./engine`, `./connector/webhook`. |
| `function` | yes | `Receiver.Method` or plain `FuncName`. Controls the `TestXxx` function name and target file derivation. |
| `test_file` | no | Explicit path to the `_test.go` to modify. Overrides automatic derivation. |

### `function` format

| Value | Target test function | Derived filename |
| - | - | - |
| `Engine.Start` | `TestEngine_Start` | `engine_test.go` |
| `WebhookNotifier.Deliver` | `TestWebhookNotifier_Deliver` | `webhook_notifier_test.go` |
| `DummyNotifier.Exists` | `TestDummyNotifier_Exists` | `dummy_notifier_test.go` |
| `NewEngine` | `TestNewEngine` | `engine_test.go` |
| `ZH07i.Read` | `TestZH07i_Read` | `zh07i_test.go` |

Receiver is converted to snake_case. For constructors (`NewXxx`), `New` is stripped before conversion.

## Section Reference

| Section | Purpose |
| - | - |
| [`context`](context.md) | How the subject is constructed and what shared state the test body needs |
| [`package_state`](context.md#package-state) | Package-level variables used to capture side effects across test cases |
| [`fixtures`](fixtures.md) | Shared data (payloads, constants) declared once before the test function |
| [`check_types`](check-types.md) | The closure-check function types used in this test |
| [`checks`](check-types.md#checks) | Catalogue of assertion functions, with scope and parameter docs |
| [`table_fields`](table-fields.md) | Extra columns in the test table beyond `name`, `before`, `after`, `checks` |
| [`cases`](cases.md) | The test scenarios — the primary content of a spec |
| `before` / `after` | [Setup and teardown patterns](before-after.md) |
