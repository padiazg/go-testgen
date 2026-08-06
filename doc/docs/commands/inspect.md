# `inspect` — Debug Parsed Function Info

`inspect` runs the analyzer on a function and prints the resulting `FuncInfo` as JSON. It does not generate any test code.

## Syntax

```bash
go-testgen inspect <pkg-pattern> <FuncSpec>
```

## Example

```bash
go-testgen inspect ./internal/core/services/user Service.CreateUser
```

Output (abbreviated):

```json
{
  "Name": "CreateUser",
  "Package": "user",
  "ImportPath": "github.com/acme/app/internal/core/services/user",
  "IsMethod": true,
  "Receiver": {
    "TypeName": "Service",
    "Kind": "struct",
    "IsPointer": true,
    "Fields": [...]
  },
  "Params": [
    {
      "Name": "ctx",
      "TypeName": "context.Context",
      "IsContext": true
    },
    {
      "Name": "req",
      "TypeName": "*userDomain.UserCreateRequest",
      "IsInterface": false,
      "IsPointer": true
    }
  ],
  "Results": [
    {
      "TypeName": "*userDomain.User",
      "IsPointer": true
    },
    {
      "TypeName": "error",
      "IsError": true
    }
  ],
  "HasError": true,
  "HasContext": true
}
```

### Receiver Kind

`Receiver.Kind` indicates the receiver type category:

| Kind | Meaning | Example |
|------|---------|---------|
| `"basic"` | Basic types | `int`, `uint`, `float64`, `bool`, `string`, `byte`, `rune` |
| `"struct"` | Struct or array | `*Service`, `[100]byte`, `MyStruct` |
| `""` | Unknown (e.g. from type parameters) | — |

## When to Use

- The generated test has a wrong signature — check if the analyzer resolved the type correctly.
- A parameter is not recognized as an interface — confirm `IsInterface` is `true`.
- A mock is not being suggested — verify the `Receiver.Fields` list includes the expected interface field.
- You want to understand what `gen` sees before it produces output.
