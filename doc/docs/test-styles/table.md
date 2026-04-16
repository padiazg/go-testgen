# `table` Style

The `table` style generates a standard table-driven test with `want` fields for value comparison. No check function closures are involved.

## When to Use

- Pure transformation functions (input → output, no side effects).
- A single `assert.Equal` or `reflect.DeepEqual` covers all assertions.
- Simple utility functions where the `check` style would be over-engineered.

## Generated Structure

For a function `func Sanitize(s string) string`:

```go
func TestSanitize(t *testing.T) {
    tests := []struct {
        name string
        s    string
        want string
    }{
        {
            name: "TODO: case 1",
            s:    "",
            want: "",
        },
        {
            name: "TODO: case 2",
            s:    "",
            want: "",
        },
    }
    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got := Sanitize(tt.s)
            assert.Equalf(t, tt.want, got, "Sanitize(%q)", tt.s)
        })
    }
}
```

For a function returning `error`:

```go
func TestParseDate(t *testing.T) {
    tests := []struct {
        name    string
        s       string
        want    time.Time
        wantErr bool
    }{
        {name: "TODO: valid", s: ""},
        {name: "TODO: invalid", s: "", wantErr: true},
    }
    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseDate(tt.s)
            if tt.wantErr {
                assert.Errorf(t, err, "ParseDate(%q): expected error", tt.s)
                return
            }
            assert.NoErrorf(t, err, "ParseDate(%q): unexpected error", tt.s)
            assert.Equalf(t, tt.want, got, "ParseDate(%q)", tt.s)
        })
    }
}
```

## Limitations vs. `check`

- All test cases share the same assertion logic in the loop body — you cannot assert different things per case without modifying the loop.
- Adding a new assertion requires a new column in every row.
- Error messages are less specific than named check functions.

For functions where these limitations matter, consider the [`check` style](check.md) instead.
