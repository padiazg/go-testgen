# Test Styles

go-testgen supports three test generation styles. Pass `--test-style <name>` to `gen` to select one.

| Style | Flag value | Default? | Description |
|-------|-----------|----------|-------------|
| [Check](check.md) | `--test-style check` | Yes | Table-driven + closure check functions |
| [Table](table.md) | `--test-style table` | No | Table-driven + `want` value fields |
| [Simple](simple.md) | `--test-style simple` | No | Standalone function, no table |

## Which Style to Use

### Use `check` (default) when:

- The function has multiple return values.
- The function returns an error that may or may not be present.
- You want to assert different things in different test cases without adding columns to every row.
- The function has interface dependencies that need mocks configured per case.
- The receiver has state you need to mutate before acting (`before` field).

`check` is the most expressive style. It requires more initial setup but scales cleanly as assertions grow.

### Use `table` when:

- The function is a pure transformation (input → output).
- A single `DeepEqual` covers all your assertions.
- You want the simplest possible test for a utility function.

### Use `simple` when:

- The function has a single execution path.
- You want a standalone test, not a loop.
- The function is internal or very small.

## Setting a Default Style

You can set a project-wide default style in `.go-testgen.yaml`:

```yaml
test_style: check   # or table, simple
```

The `--test-style` flag always overrides the config file.
