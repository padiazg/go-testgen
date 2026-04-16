# `simple` Style

The `simple` style generates a standalone test function without a table. Each test case becomes its own `t.Run` block, or there is a single assertion block for single-path functions.

## When to Use

- The function has a single obvious execution path.
- You want a quick test without the overhead of a table.
- Internal or very small helper functions.

## Generated Structure

For a function `func Add(a, b int) int`:

```go
func TestAdd(t *testing.T) {
    t.Run("TODO: case 1", func(t *testing.T) {
        got := Add(0, 0)
        assert.Equalf(t, 0, got, "Add(0, 0)")
    })
}
```

For a function returning `error`:

```go
func TestConnect(t *testing.T) {
    t.Run("TODO: success", func(t *testing.T) {
        err := Connect("localhost:5432")
        assert.NoErrorf(t, err, "Connect: unexpected error")
    })

    t.Run("TODO: error", func(t *testing.T) {
        err := Connect("")
        assert.Errorf(t, err, "Connect: expected error for empty addr")
    })
}
```

## When Not to Use

- Functions with multiple inputs — a table is cleaner.
- Methods with interface dependencies — the `check` style's `before` field handles mock setup more clearly.
- Anything where you need to assert different things in different cases.
