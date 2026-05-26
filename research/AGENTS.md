# Unit Test Generation — go-testgen closure-check style

Este proyecto usa `go-testgen` para scaffolding de tests con el patrón closure-check.
Las especificaciones de test viven en archivos `.testspec.yaml` junto al código fuente.

---

## Qué es un `.testspec.yaml`

Describe **qué probar** en términos del dominio, sin código Go.
Lo puede escribir una IA a partir del análisis DDD o un humano durante TDD.
`go-testgen gen-cases` lo lee y genera las entradas del slice `tests` en el `_test.go`.

Campos clave:

| Campo | Qué describe |
|---|---|
| `context` | Cómo se construye el sujeto, variables compartidas, estado de package |
| `fixtures` | Payloads de datos compartidos entre casos |
| `check_types` | Los tipos de check function (`engineTestCheckFn`, `notificationCheckFn`, etc.) |
| `checks` | Catálogo de funciones check con su firma y cuándo usarlas |
| `table_fields` | Campos extra del struct del test (inputs, gates, state) |
| `cases` | Casos de uso con `before`, `after`, `fields`, `gates`, `checks` |

---

## Cómo generar un `.testspec.yaml`

Cuando se te pide generar un spec a partir de un diseño DDD o análisis del dominio:

1. **Lee el código fuente** del componente a testear: tipos, dependencias, qué retorna.
2. **Lee el `_test.go` generado** por go-testgen: tipos de check, composer, patrones de setup.
3. **Identifica** los casos de uso a partir de las invariantes del dominio:
   - Happy path(s)
   - Fallos de validación (inputs inválidos)
   - Fallos de infraestructura (errores de IO, mocks que fallan)
   - Casos borde (nil, vacío, colecciones con un elemento, etc.)
4. **Determina el mecanismo de before** mirando las dependencias del sujeto:
   - ¿Usa testify mock? → `mechanism: testify-mock`
   - ¿Inyecta funciones/clientes? → `mechanism: field-injection`
   - ¿Resetea campos a nil? → `mechanism: field-reset`
   - ¿Muta estado interno? → `mechanism: state-mutation`
   - ¿Combinación? → `mechanism: mixed`
   - ¿Nada? → omitir `before` o `mechanism: none`
5. **Nombra los casos** en términos del dominio, no de la implementación:
   - ✅ `"notification delivered to all channels"`
   - ✅ `"fail - missing target channel"`
   - ❌ `"test case 3"`
   - ❌ `"nil pointer branch"`

---

## Cómo implementar tests desde un spec

Cuando se te pide completar el código de un `_test.go` a partir de un `.testspec.yaml`:

### Paso 1 — Leer el contexto

```
spec.context.subject_init     → cómo construir el sujeto por caso
spec.context.shared_setup     → qué declarar fuera del for loop
spec.context.package_state    → variables de package (errors, counters)
```

### Paso 2 — Mapear check types

```
spec.check_types[*].type_name → nombre del tipo Go
spec.check_types[*].composer  → nombre del composer var
spec.check_types[*].package   → import si es cross-package
```

### Paso 3 — Generar cada case

Para cada `spec.cases[i]`:

```
name    → tt.name en el struct literal
fields  → valores de spec.table_fields para este caso
before  → función Go según spec.cases[i].before.mechanism (ver abajo)
after   → función Go según spec.cases[i].after.mechanism (ver abajo)
checks  → composer(spec.cases[i].checks...) 
          ó []CheckType{spec.cases[i].checks...}
gates   → campos wantLog/wantPanic/wantValue/wantErr para este caso
```

### Paso 4 — Elegir el patrón de before según mechanism

**`testify-mock`**: Construir el mock, llamar `.On()` con matchers, `.Return()`.
Agregar `.Once()` cuando el orden importa (protocolo multi-step).
Agregar `.Maybe()` para goroutine loops.
Agregar `.Run(func(args mock.Arguments){...})` para mutar buffers del caller.

```go
before: func(z *ZH07i) {
    mk := z.transport.(*mockTransportProvider)
    mk.On("Read", mock.Anything, mock.Anything).
        Run(func(args mock.Arguments) {
            in := args.Get(0).([]byte)
            copy(in, samplePayload[0:1])
        }).
        Return(1, nil).Once()
},
```

**`field-injection`**: Asignar funciones, structs, o interfaces directamente.

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

**`field-reset`**: Setear campo a nil o zero value.

```go
before: func(n *WebhookNotifier) {
    n.client = nil
},
```

**`state-mutation`** (con retorno): Mutar estado interno, retornar el valor de prueba.

```go
before: func(d *DummyNotifier) *model.Notification {
    n := &model.Notification{ID: "payload-01", Event: "test"}
    d.lock.Lock()
    defer d.lock.Unlock()
    d.in = append(d.in, n)
    return n
},
```

### Paso 5 — Patrón de after según mechanism

**`stop-method`**:
```go
after: func(z *ZH07q, _ context.CancelFunc) { z.Stop() },
```

**`cancel-context`**:
```go
after: func(_ *T, cancel context.CancelFunc) { cancel() },
```

**`close-channel`** (va en el test body, no en la tabla):
```go
go func() {
    n.Channel <- &model.Notification{Data: "Test message"}
    close(n.Channel)
}()
time.Sleep(10 * time.Millisecond)
```

---

## Convenciones del patrón closure-check

### Tipo y composer
```go
type check<Name>Fn func(*testing.T, <return_types...>)

var check<Name> = func(fns ...check<Name>Fn) []check<Name>Fn { return fns }
```

### Check de error con sentinel vacío
```go
func checkFooError(want string) checkFooFn {
    return func(t *testing.T, result ReturnType, err error) {
        t.Helper()
        if want == "" {
            assert.NoErrorf(t, err, "checkFooError: expected no error, got %v", err)
            return
        }
        if assert.Errorf(t, err, "checkFooError: expected error containing %q", want) {
            assert.Containsf(t, err.Error(), want, "checkFooError: error mismatch")
        }
    }
}
```

### Check de valor numérico
```go
func checkPM25(want, delta float32) checkReadFn {
    return func(t *testing.T, r *domain.Reading, err error) {
        t.Helper()
        assert.InDeltaf(t, want, r.PM25, float64(delta), "checkPM25 mismatch")
    }
}
```

### Check de estado booleano (slice vacío vs no vacío)
```go
func hasErrors(has bool) engineTestCheckFn {
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

---

## Generando specs desde el dominio (pipeline DDD → Tests)

Cuando recibes un prompt de diseño DDD (entidades, ports, adapters, invariantes):

```
1. Por cada puerto/adaptador/use-case, identificar las funciones públicas a testear.
2. Para cada función, crear un .testspec.yaml con:
   a. context: cómo se construye el sujeto
   b. check_types + checks: qué se puede validar
   c. cases: cada invariante del dominio como un caso de test
3. Usar nombres de casos que lean como requisitos:
      "order placed successfully"
      "fail - inventory not available"
      "fail - payment gateway timeout"
4. En before.description, describir el estado del mundo necesario
   para ese caso, no los detalles de implementación del mock.
5. En checks, listar los comportamientos observables a verificar.
```

El código concreto de before/after lo genera la IA en un segundo paso,
leyendo el spec + el código fuente + el `_test.go` scaffolded.
