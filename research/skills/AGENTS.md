# go-testgen — AI agent instructions

## Workflow

```
go-testgen gen <package> <FuncSpec>   →   _test.go scaffolding
IA lee los skills + código fuente     →   completa los test cases
go test ./...                         →   verde
```

No hay paso intermedio. No hay archivos de spec. La IA genera los cases
directamente a partir del código fuente aplicando los skills.

## Skills disponibles

Están en `skills/` en la raíz del repositorio.
Lee ambos antes de generar cualquier test.

| Skill | Cuándo leerlo |
|---|---|
| `skills/closure-check-tests/SKILL.md` | Siempre — enseña la mecánica del patrón |
| `skills/gen-test-cases/SKILL.md` | Siempre — enseña qué casos generar |

## Qué genera `go-testgen gen`

El scaffolding ya incluye:
- El tipo `checkXxxFn` con los parámetros correctos según la firma de la función
- El composer `var checkXxx = func(fns ...checkXxxFn) []checkXxxFn { return fns }`
- El struct del slice `tests` con los campos `name`, `before`, `after` (si aplica), `checks`
- El test runner completo (el `for` loop con `t.Run`, la construcción del sujeto, la llamada a la función, la ejecución de checks)
- El slice `tests` vacío

**No modificar** el test runner. Solo agregar entries al slice `tests` y
los check functions y fixtures que van encima.

## Cómo pedir a la IA que genere los cases

Prompt mínimo:
```
Lee skills/closure-check-tests/SKILL.md y skills/gen-test-cases/SKILL.md.
Luego lee [archivo fuente] y [archivo _test.go].
Genera los test cases para TestXxx_Yyy.
```

Con contexto DDD:
```
Lee skills/closure-check-tests/SKILL.md y skills/gen-test-cases/SKILL.md.
Contexto del dominio: [descripción de entidades, invariantes, puertos].
Lee [archivo fuente] y [_test.go].
Genera los test cases para TestXxx_Yyy cubriendo los casos de uso del dominio.
```

## En el pipeline hexago

Cuando hexago genera la estructura del proyecto, go-testgen se usa como sidekick:

```
hexago generate [prompt DDD]
  └─► estructura hexagonal + interfaces
      └─► go-testgen gen [cada adapter/usecase/entity]
          └─► IA aplica skills + contexto DDD → test cases completos
```

La IA que genera los cases puede recibir el prompt DDD original como contexto
para nombrar los casos en términos del dominio.
