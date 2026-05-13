# `version` — Show Build Version

`version` prints the build version of go-testgen.

## Syntax

```bash
go-testgen version [flags]
```

## Example

```bash
go-testgen version
```

Output:

```
go-testgen v0.1.3
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-s`, `--simple` | false | Print only the version string (script-friendly). |

## Simple Output

```bash
go-testgen version --simple
# or
go-testgen version -s
```

Output:

```
v0.1.3
```

Use `-s` in scripts or CI pipelines where only the version string is needed.
