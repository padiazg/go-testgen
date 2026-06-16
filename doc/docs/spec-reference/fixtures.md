# Fixtures

!!! warning "Experimental"
    `gen-cases` and `.testspec.yaml` are experimental — an exploration of the approach. Schema, API, and behavior may change. Feedback welcome.
    Alternatively, install the [AI agent skills](../getting-started/installation.md#ai-agent-skills) and let an AI generate cases directly from source code — no spec file required.

Fixtures are **shared data values** declared once at file scope, before the `TestXxx` function. They represent payloads, protocol constants, or any reusable Go value that multiple test cases reference.

## Schema

```yaml
fixtures:
  - name:        <identifier>
    type:        <Go type>
    description: <what this fixture represents>
    value: |
      <Go literal — may be multiline>
```

### Fields

| Field | Required | Description |
| - | - | - |
| `name` | yes | Go identifier for the `var` declaration (e.g. `samplePayload`, `validConfig`). |
| `type` | yes | Go type string (e.g. `"[]byte"`, `"*model.Notification"`, `"*Config"`). |
| `description` | yes | Human-readable description of what the fixture represents. |
| `value` | no | Go literal to use as the initializer. If omitted, `gen-cases` emits an `// ai-hint:` placeholder. |

## Generated Output

### With a value

```yaml
fixtures:
  - name: sampleInitiativePayload
    type: "[]byte"
    description: Complete, valid ZH07B sensor frame in initiative-upload mode.
    value: |
      []byte{
          0x42, 0x4D, // Start bytes
          0x00, 0x1C, // Frame length
          0x00, 0x01, // PM1.0
          0x00, 0x05, // PM2.5
          0x00, 0x09, // PM10
      }
```

Generated:

```go
// Complete, valid ZH07B sensor frame in initiative-upload mode.
var sampleInitiativePayload = []byte{
    []byte{
        0x42, 0x4D, // Start bytes
        0x00, 0x1C, // Frame length
        0x00, 0x01, // PM1.0
        0x00, 0x05, // PM2.5
        0x00, 0x09, // PM10
    }
}
```

### Without a value (AI fills in)

```yaml
fixtures:
  - name: validWebhookConfig
    type: "*Config"
    description: >
      A WebhookNotifier config pointing to a local test server at
      http://localhost:8080/webhook, with a custom Header-XYZ header.
```

Generated:

```go
// A WebhookNotifier config pointing to a local test server at
// http://localhost:8080/webhook, with a custom Header-XYZ header.
var validWebhookConfig = *Config{
    // ai-hint: fill with the value described above
    // A WebhookNotifier config pointing to a local test server at
    // http://localhost:8080/webhook, with a custom Header-XYZ header.
}
```

## Insertion Position

Fixtures are inserted **before** the `TestXxx` function, after any existing `var`/`const` declarations in the file.

Multiple fixtures are grouped together in the order they appear in the spec.

## Idempotency

If a `var` with the same `name` already exists in the file (detected by AST inspection of all `GenDecl` nodes at file scope), the fixture is **skipped**. Use `--force` to replace it.

## When to Use Fixtures

Use fixtures when:

- A payload or config struct is **shared across multiple test cases** — define it once and reference it by name.
- The value is **large** (e.g. a binary frame, a long JSON blob) and repeating it inline would clutter the case table.
- The value is a **constant** in the domain (e.g. a protocol magic byte sequence).

Do not use fixtures for values that differ per case — put those in `case.fields` instead.

## Multiple Fixtures

```yaml
fixtures:
  - name: minimalConfig
    type: "*Config"
    description: Config with only required fields set.
    value: "&Config{Endpoint: \"http://localhost:8080\"}"

  - name: fullConfig
    type: "*Config"
    description: Config with all optional fields populated.
    value: |
      &Config{
          Endpoint: "http://localhost:8080/webhook",
          Headers:  map[string]string{"X-API-Key": "test-key"},
          Insecure: false,
      }

  - name: testNotification
    type: "*model.Notification"
    description: Standard notification used in happy-path cases.
    value: |
      &model.Notification{
          ID:    "msg-01",
          Event: model.EventType("test-event"),
          Data:  "test-data",
      }
```
