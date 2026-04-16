# Examples

Practical examples of how to write effective unit tests for common scenarios using go-testgen's generated scaffolding.

!!! note
    These examples are being added incrementally. Check back for updates.

## Planned Examples

The following examples will be added here to demonstrate how to handle common testing scenarios:

### Database Clients

Testing functions that interact with a database — using generated testify mocks for repository interfaces, setting up mock expectations per case, and asserting database-related side effects.

### HTTP Clients

Testing functions that make HTTP requests — using `mockHTTPClient` for `http.Client` injection, `DryRunTransport` for `http.RoundTripper`-based injection, and `httptest.Server` for integration-style tests.

### Standard Output (`os.Stdout`)

Testing functions that write to stdout — capturing output by redirecting `os.Stdout` to a pipe and reading the result in a check function.

### Goroutines and Channels

Testing concurrent code — asserting values sent to channels, handling `time.Sleep` + `select` patterns for async operations, and using `panic`/`recover` for channel close scenarios.

### `io.Reader` / `io.ReadCloser`

Testing functions that read from an `io.Reader` — using `errorReader` to simulate read failures and `io.NopCloser(strings.NewReader(...))` for happy-path response bodies.

### Constructors (`New`)

Testing constructor functions — asserting that returned structs have the correct initial field values using per-field check functions.

### Functions Returning Functions (Iterators)

Testing generator functions that return a `func() T` iterator — calling the iterator in a loop inside a check function and asserting the sequence of yielded values.

---

*Examples will be added with full runnable code. If you have a scenario you want covered, open an issue at [github.com/padiazg/go-testgen](https://github.com/padiazg/go-testgen/issues).*
