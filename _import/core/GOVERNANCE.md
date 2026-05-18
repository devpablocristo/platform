# Gobernanza del ecosistema pablo

Documento de referencia para el ecosistema multi-repo `pablo/*` (core, modules,
companion, nexus, pymes, medmory). Vive en `core/` porque es la dependencia
más universal; los demás repos lo referencian.

## 1. Estructura del ecosistema

Cuatro ejes:

- **`core/`** — primitivas técnicas reusables, sin lógica de UI ni de dominio
  (HTTP, auth, DB, observability, contratos canónicos, providers AWS).
- **`modules/`** — componentes reutilizables con dominio acotado (UI React,
  CRUD genérico, scheduling, kanban, inbox, etc.).
- **Apps** — `companion`, `nexus`, `pymes`, `medmory`. Consumen `core` y
  `modules`. Cada una tiene dominio propio.

Reglas duras:

- `core` NO contiene lógica de producto.
- `modules` NO contiene código duplicado de apps.
- Las apps NO reinventan lo que ya vive en `core` o `modules`.
- No hay `replace` directives ni paths locales en go.mod de las apps.
- No hay `file:./vendor/...` en package.json de las apps.

## 2. Promoción a `core` / `modules` — regla de los 3

Para promover código a `core` o `modules` se requieren **≥3 consumidores
actuales o concretamente planificados en próximos 30 días**. Entre tanto,
vive en la app y se duplica conscientemente (≤2 copias).

Si encontrás un patrón duplicado en 2 apps, **dejalo**. Si aparece un 3er
consumidor, ese commit es el trigger para extraer.

## 3. `core` vs `modules`

| | `core` | `modules` |
|---|---|---|
| Lenguajes | Go + TS | TS principalmente (algún Go) |
| Capa | Primitivas técnicas | Componentes/dominio acotado |
| Ejemplo | `http/go`, `authn/go`, `errors/go` | `ui-conversation-inbox`, `crud-ui`, `scheduling` |
| Dependencias | Mínimas, casi stdlib | Puede depender de `core` |

## 4. Versionado y tags

- **Semver estricto**: minor = nuevas APIs sin romper, patch = bugfix, major = breaking.
- **Tag inmediato al publicar**: prohibido publicar a npm/Go-proxy sin git tag correspondiente.
- **Convención de tag**: `<path>/v<X.Y.Z>` (e.g. `authn/ts/v0.3.0`, `ai/go/v0.3.0`).
- **VERSION = source of truth**: el archivo `VERSION` o el campo `version` de package.json debe coincidir con el último tag. CI lo valida.
- **Sin pseudo-versions en consumers**: prohibido `go.mod` con `vX.Y.Z-YYYYMMDDHHMMSS-sha`. Si necesitás cambio urgente, cortás patch del lib y consumís ese patch.

## 5. Ghost packages

Packages publicados sin consumidores tienen TTL:

- Si un package está **90 días sin un solo import nuevo**, se evalúa deprecación.
- Deprecación se hace con `npm deprecate` (TS) o anotación en README + tag deprecation (Go).
- Después de 180 días deprecado sin reactivación, se puede archivar la carpeta o borrar.

## 6. Peer dependencies (TS)

- Política actual para `modules-ui-*`: `react: "^18.0.0 || ^19.0.0"` (dual compat).
- Cuando React 20 salga: subir todos en bloque coordinado.
- Nuevos packages UI deben declarar peerDeps explícitas.

## 7. Post-release housekeeping

Si tras tagear vas a mergear docs/chores:

- **Opción A** (preferida): no mergees nada hasta que tengas el próximo bump real planeado. El housekeeping va en el mismo PR del próximo bump.
- **Opción B**: si urgen los docs, cortás patch (`vX.Y.Z+1`) inmediatamente.
- **Prohibido**: acumular >5 commits post-tag sin tagear.

## 8. Release cadence

- **Mínimo mensual** para cada package con commits, aunque sean dev-deps bumps.
- **Inmediato** al mergear feature/fix funcional.
- CI auto-publica + tagea (ver `docs/RELEASE_FLOW.md` en core y modules).

## 9. Responsabilidades

- **Dueño de `core`**: aprueba promociones, corta releases, mantiene compatibilidad. Pablo.
- **Dueño de `modules`**: idem para UI/dominios reusables. Pablo.
- **Dueños de apps**: NO copian código reusable; abren issue en core/modules si falta algo.
- **Cualquier contribuyente** que copy-paste código de core/modules a una app: el reviewer debe rechazar el PR.

## 10. Cómo se hace cumplir

- **Audit periódico** (ver `modular-swinging-hummingbird.md` plan + checklist N): regenera matrices de consumo, publicación, duplicación.
- **CI cross-repo** (roadmap MP-05): post-publish en core/modules dispara build en companion/nexus/pymes/medmory.
- **Renovate/Dependabot** (roadmap LP-01): PRs automáticos en consumers cuando core/modules publican.

## 11. Excepciones

Cualquier desviación de este documento requiere:

1. ADR (Architecture Decision Record) escrito justificando.
2. Acuerdo de los dueños de los repos afectados.
3. Plan de remediación o TTL para volver al estándar.
