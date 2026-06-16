---
name: gen-test-cases
description: >
  Identify and generate test cases for a Go function. Apply when given source
  code (and optionally a DDD/domain description) and asked to fill in test cases
  in a _test.go scaffolded by go-testgen. Works together with the
  closure-check-tests skill which handles the mechanics of writing each case.
---

# Generating test cases from source code and domain context

Read this skill to understand *what* cases to generate. Read `closure-check-tests`
to understand *how* to write each case.

---

## Step 1 — Read before writing

Before generating any case, read:

1. **The function under test** — its signature, what it returns, what errors it
   can produce, what dependencies it calls. Every error return path is a test case.

2. **The `_test.go` scaffolding** — the struct fields in `tests`, the check function
   type signature, what parameters `before` receives, whether `after` exists.
   Do not add fields the struct doesn't have. Do not change the test runner.

3. **The mock(s)** — what methods they expose. The mock interface determines what
   behaviors you can simulate.

4. **The domain context** (if provided) — invariants, validation rules, port
   contracts. These become the names and intent of your cases.

---

## Step 2 — Identify cases systematically

For every function under test, generate cases in this order:

### A. Infrastructure failures (one per external call)

Every call to an injected dependency (transport, HTTP client, database, marshal
function) that can return an error is a case. Read the function body — each
`if err != nil { return ... }` that wraps an external call is a candidate.

```go
// This function body → 3 infrastructure failure cases
func (z *ZH07i) Read(...) {
    if _, err = z.transport.Read(b0, false); err != nil { // case: transport fails on first read
        return ...
    }
    if _, err = z.transport.Read(b1, false); err != nil { // case: transport fails on second read
        return ...
    }
    if _, err = z.transport.Read(data, true); err != nil { // case: transport fails on third read
        return ...
    }
}
```

### B. Validation failures (one per validation check)

Every `if condition { return error }` that validates state — format, range,
protocol, business rule — is a case.

```go
if b0[0] != 0x42 { return error }      // case: wrong start byte
if b1[0] != 0x4d { return error }      // case: wrong frame signature
if length != 28  { return error }       // case: wrong frame length
```

### C. Happy path(s)

At minimum one case where all dependencies succeed and the function returns the
expected value. If there are multiple valid configurations (e.g. with/without
optional fields, different input shapes), add one case per distinct success path.

### D. Edge cases

- `nil` inputs where the function explicitly handles them
- Empty collections
- Zero values that trigger different behavior
- Boundary values (empty string, 0, max int) if the function branches on them

### E. Cases from domain context

If given a DDD description, add cases for:
- Each stated invariant ("an order cannot be placed if inventory is zero")
- Each port contract ("the payment gateway must be called exactly once per order")
- Each explicit validation rule from the domain model

---

## Step 3 — Determine the before mechanism

Read the function under test and the mock to choose:

| What the function uses | Mechanism |
|---|---|
| Interface field accessed as mock (`.transport`, `.client`) | `testify-mock` |
| Function field on the struct (`n.jsonMarshal`, `n.httpNewRequest`) | `field-injection` |
| Nil check on a field to choose a branch | `field-reset` |
| Internal slice/map state read by the function | `state-mutation` |
| Only config passed at construction time | none — no `before` needed |
| Combination of the above | `mixed` |

For `testify-mock` with sequential reads of the same method, count exactly how
many times the method is called and use `.Once()` for each call in order.

---

## Step 4 — Determine shared fixtures

If multiple cases use the same binary payload, struct literal, or large constant,
declare it once as a `var` or `const` above the test function:

```go
var samplePayload = []byte{ 0x42, 0x4D, ... }
```

Only create a fixture if at least two cases reference it. Single-use data goes
inline in the `before`.

---

## Step 5 — Order the cases

Write cases in this order inside the `tests` slice:
1. Infrastructure failures (in the order they appear in the function body)
2. Validation failures (in the order they appear in the function body)
3. Happy path(s)
4. Edge cases

This order makes the test output diagnostic: the first failure points to the
first broken code path.

---

## Applying this to a DDD project (hexago pipeline)

When generating tests as part of a hexago-scaffolded project:

**For domain entities and value objects:**
- Happy path: valid construction
- Validation failures: one case per invariant (nil ID, empty name, negative amount)
- Each method: follow Steps A–D above

**For use cases / application services:**
- Each port interaction is an infrastructure failure case
- Each business rule check is a validation failure case
- The happy path exercises the full flow end-to-end

**For adapters (repositories, HTTP handlers, message consumers):**
- Focus on infrastructure failures (DB errors, network errors, decode errors)
- Happy path with representative real data
- Edge cases specific to the protocol (empty response body, unexpected status code)

**Naming from the domain:**
Use the language of the domain in case names, not the language of the implementation.
The test suite is documentation for future developers.

```
✅ "fail - payment gateway unavailable"
✅ "fail - order already confirmed"
✅ "success - order placed with split shipment"

❌ "test case 3"
❌ "err != nil branch in processPayment"
❌ "when the http client returns 503"
```

---

## Checklist before submitting

- [ ] Every `if err != nil` wrapping an external call has a failure case
- [ ] Every validation `if condition { return error }` has a failure case
- [ ] At least one happy path case
- [ ] Cases are in execution order (failures first, then success)
- [ ] `before` uses the correct mechanism for each case
- [ ] No field is added to the struct that wasn't already there
- [ ] Shared data used by 2+ cases is declared as a fixture var
- [ ] Case names read as domain scenarios, not code paths
- [ ] The file compiles: `go build ./...`
- [ ] The tests pass: `go test ./...`
