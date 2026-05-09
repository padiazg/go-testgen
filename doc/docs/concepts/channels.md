# Channel Type Support

go-testgen detects and correctly handles all Go channel types — `chan`, `chan<-`, `<-chan` — in both function parameters and return values.

## Detection

The analyzer inspects AST `*ast.ChanType` nodes and the `types.Chan` type from `go/packages`, setting:

- `IsChannel` — whether the parameter/result is a channel
- `ChanDir` — direction: `0` (bidirectional), `1` (send-only), `2` (receive-only)

## Generated Output

Channel types are handled in three places:

### `typeToString`

Preserves channel direction in string representation:

```
chan int       → "chan int"
chan<- string  → "chan<- string"
<-chan []byte  → "<-chan []byte"
```

### `qualifiedTypeName`

Correctly qualifies channel types with package prefixes:

```
chan *user.User     → "chan *userDomain.User"
<-chan []string     → "<-chan cache.Pkg.StringList"
```

### `placeholderValue`

Returns `nil` for all channel types — the only valid zero value.

## Example

```go
// Source function
func ChannelRecvReturn(ctx context.Context) <-chan *SomeType {
    ch := make(chan *SomeType)
    go func() {
        defer close(ch)
        ch <- &SomeType{ID: "1", Name: "one"}
    }()
    return ch
}
```

```bash
go-testgen gen ./internal/pkg ChannelRecvReturn
```

The generated test captures the `<-chan *SomeType` return correctly with a `nil` placeholder for the channel parameter.
