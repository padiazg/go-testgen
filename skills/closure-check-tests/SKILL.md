---
name: closure-check-tests
description: >
  Write Go unit tests in the closure-check style used by go-testgen. Apply when
  filling in test cases in a _test.go that was scaffolded by `go-testgen gen`,
  or when writing new tests from scratch that follow this pattern. The style uses
  typed check functions composed into table-driven tests with before/after hooks.
---

# Go closure-check test pattern

The closure-check pattern structures table-driven tests so that assertions are
expressed as composable typed closures, not inline `if` statements. This keeps
the test table declarative and makes each case's intent readable at a glance.

`go-testgen gen` produces the scaffolding. Your job is to fill in the cases.

---

## Anatomy of a scaffolded test

```go
// 1. The check function type — parameters match the function's return values
type checkZH07iReadFn func(*testing.T, *domain.ReadingEvent, error)

// 2. The composer — collects checks into a slice
var checkZH07iRead = func(fns ...checkZH07iReadFn) []checkZH07iReadFn { return fns }

func TestZH07i_Read(t *testing.T) {
    // 3. Check functions are defined here (scope: local) or at package level (scope: global)

    tests := []struct {
        name   string
        before func(*ZH07i)
        checks []checkZH07iReadFn
    }{
        // 4. Cases go here — this is what you fill in
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            z := newZH07i(&Config{Transport: new(mockTransportProvider)})
            if tt.before != nil {
                tt.before(z)
            }
            re, err := z.Read(context.Background())
            for _, check := range tt.checks {
                check(t, re, err)
            }
        })
    }
}
```

---

## Check function patterns

### Error check (sentinel empty string)

The most common pattern. Empty `want` means success; non-empty means error containing that substring.

```go
checkZH07iReadError := func(want string) checkZH07iReadFn {
    return func(t *testing.T, re *domain.ReadingEvent, err error) {
        t.Helper()
        if want == "" {
            assert.NoErrorf(t, err, "checkZH07iReadError: expected no error, got %v", err)
            return
        }
        if assert.Errorf(t, err, "checkZH07iReadError: expected error containing %q", want) {
            assert.Containsf(t, err.Error(), want, "checkZH07iReadError mismatch")
        }
    }
}
```

### Value check

```go
checkReadValues := func(pm1, pm25, pm10 float32) checkZH07iReadFn {
    return func(t *testing.T, re *domain.ReadingEvent, err error) {
        t.Helper()
        if assert.NotNilf(t, re.Reading, "checkReadValues: reading is nil") {
            assert.InDeltaf(t, pm1,  re.Reading.NumberPM1,  0.01, "PM1 mismatch")
            assert.InDeltaf(t, pm25, re.Reading.NumberPM25, 0.01, "PM2.5 mismatch")
            assert.InDeltaf(t, pm10, re.Reading.NumberPM10, 0.01, "PM10 mismatch")
        }
    }
}
```

### Boolean/state check (captures outer variable)

```go
// errors is a package-level var populated by the OnError callback
hasErrors := func(has bool) engineTestCheckFn {
    return func(t *testing.T, e *Engine) {
        t.Helper()
        if has {
            assert.NotEmptyf(t, errors, "hasErrors: expected errors, got none")
        } else {
            assert.Emptyf(t, errors, "hasErrors: expected no errors, got %+v", errors)
        }
    }
}
```

### Cross-package check

When the check type is defined in another package (e.g. `model.TestCheckResultFn`),
it's used directly — no local redefinition needed:

```go
checks: model.CheckResult(
    model.CheckResultError(""),
    model.CheckResultSuccess(true),
),
```

---

## Before hook patterns

The `before` field signature varies based on what the subject under test needs.
Read the existing struct in the scaffolded test to know the exact signature.

### testify-mock — multi-step protocol reads

Use `.Once()` when order matters (protocol with sequential reads).
Use `.Maybe()` for goroutine loops (called zero or more times).
Use `.Run()` to mutate the caller's buffer ([]byte fills).

```go
before: func(z *ZH07i) {
    mk := z.transport.(*mockTransportProvider)
    mk.On("Read", mock.Anything, mock.Anything).
        Run(func(args mock.Arguments) {
            copy(args.Get(0).([]byte), samplePayload[0:1])
        }).
        Return(1, nil).Once()
    mk.On("Read", mock.Anything, mock.Anything).
        Run(func(args mock.Arguments) {
            copy(args.Get(0).([]byte), samplePayload[1:4])
        }).
        Return(3, nil).Once()
    mk.On("Read", mock.Anything, mock.Anything).
        Run(func(args mock.Arguments) {
            copy(args.Get(0).([]byte), samplePayload[4:])
        }).
        Return(28, nil).Once()
},
```

### field-injection — replace a dependency with a fake

```go
before: func(n *WebhookNotifier) {
    n.client = &mockHTTPClient{
        DoFunc: func(req *http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: http.StatusForbidden,
                Body:       io.NopCloser(bytes.NewBufferString("Forbidden")),
            }, nil
        },
    }
},
```

Inject a function field to force a specific error path:

```go
before: func(n *WebhookNotifier) {
    n.jsonMarshal = func(_ any) ([]byte, error) {
        return nil, fmt.Errorf("error from json.Marshal")
    }
},
```

### field-reset — force a branch by zeroing a field

```go
before: func(n *WebhookNotifier) {
    n.client = nil  // forces getClient() to construct a new one
},
```

### state-mutation with return value

When `before` needs to both set up state AND return a value the test body uses:

```go
before: func(d *DummyNotifier) *model.Notification {
    n := &model.Notification{ID: "payload-01", Event: "test"}
    d.lock.Lock()
    defer d.lock.Unlock()
    d.in = append(d.in, n)
    return n  // returned value is passed to d.Exists(n) in the test body
},
```

### no before

Simply omit the field or set it to `nil`. The test runner checks `if tt.before != nil`.

---

## After hook patterns

Only needed when the function starts goroutines or channels.

```go
// stop-method
after: func(z *ZH07q) { z.Stop() },

// cancel-context
after: func(_ *T, cancel context.CancelFunc) { cancel() },
```

---

## Naming conventions

| Pattern | Example |
|---|---|
| Happy path | `"success"` |
| Specific success | `"valid measurement"` |
| Error — infrastructure | `"fail - transport read error"` |
| Error — validation | `"fail - invalid frame signature"` |
| Error — missing resource | `"fail - missing target channel"` |
| Edge case | `"empty notification id"`, `"nil message"` |

Name from the **domain perspective**, not the implementation:
- ✅ `"fail - data not ready after retries"`
- ❌ `"branch where retry counter reaches max"`

---

## Complete filled-in example

```go
var samplePayload = []byte{
    0x42, 0x4D, 0x00, 0x1C,
    0x00, 0x54, 0x00, 0x6E, 0x00, 0x7C, // PM1=84, PM2.5=110, PM10=124
    // ... rest of frame
    0x03, 0x27, // checksum
}

func TestZH07i_Read(t *testing.T) {
    checkZH07iReadError := func(want string) checkZH07iReadFn {
        return func(t *testing.T, re *domain.ReadingEvent, err error) {
            t.Helper()
            if want == "" {
                assert.NoErrorf(t, err, "expected no error")
                return
            }
            if assert.Error(t, err) {
                assert.Contains(t, err.Error(), want)
            }
        }
    }

    checkReadValues := func(pm1, pm25, pm10 float32) checkZH07iReadFn {
        return func(t *testing.T, re *domain.ReadingEvent, err error) {
            t.Helper()
            if assert.NotNil(t, re.Reading) {
                assert.InDeltaf(t, pm1,  re.Reading.NumberPM1,  0.01, "PM1")
                assert.InDeltaf(t, pm25, re.Reading.NumberPM25, 0.01, "PM2.5")
                assert.InDeltaf(t, pm10, re.Reading.NumberPM10, 0.01, "PM10")
            }
        }
    }

    tests := []struct {
        name   string
        before func(*ZH07i)
        checks []checkZH07iReadFn
    }{
        {
            name: "fail - start character read error",
            before: func(z *ZH07i) {
                z.transport.(*mockTransportProvider).
                    On("Read", mock.Anything, mock.Anything).
                    Return(0, fmt.Errorf("read-error"))
            },
            checks: checkZH07iRead(
                checkZH07iReadError("read-error"),
            ),
        },
        {
            name: "fail - incorrect start character",
            before: func(z *ZH07i) {
                z.transport.(*mockTransportProvider).
                    On("Read", mock.Anything, mock.Anything).
                    Run(func(args mock.Arguments) {
                        args.Get(0).([]byte)[0] = 0x00
                    }).
                    Return(1, nil)
            },
            checks: checkZH07iRead(
                checkZH07iReadError("expected 0x42"),
            ),
        },
        {
            name: "success",
            before: func(z *ZH07i) {
                mk := z.transport.(*mockTransportProvider)
                mk.On("Read", mock.Anything, mock.Anything).
                    Run(func(args mock.Arguments) { copy(args.Get(0).([]byte), samplePayload[0:1]) }).
                    Return(1, nil).Once()
                mk.On("Read", mock.Anything, mock.Anything).
                    Run(func(args mock.Arguments) { copy(args.Get(0).([]byte), samplePayload[1:4]) }).
                    Return(3, nil).Once()
                mk.On("Read", mock.Anything, mock.Anything).
                    Run(func(args mock.Arguments) { copy(args.Get(0).([]byte), samplePayload[4:]) }).
                    Return(28, nil).Once()
            },
            checks: checkZH07iRead(
                checkZH07iReadError(""),
                checkReadValues(84, 110, 124),
            ),
        },
    }
    // ... test runner (don't modify)
}
```
