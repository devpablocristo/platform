# GPT.md

Este repo usa `AGENTS.md` como fuente de verdad para GPT/Codex y agentes equivalentes.

Aplican sin excepción las reglas de `AGENTS.md`.

## Resumen crítico

- Este repo es un monorepo de capacidades reutilizables, no de productos
- Los módulos raíz son: `saas`, `browser`, `http`, `observability`, `config`, `security`, `validate`, `errors`, `utils`, `concurrency`, `authn`, `authz`, `notifications`, `databases`, `providers`, `eventing`, `governance`, `artifact`, `webhook`, `activity`, `ai`
- Dentro de este repo NO se usan sufijos `-core`
- No crear `common/`, `shared/`, `utils/` en la raíz
- Lenguaje distinto no implica repo distinto
- Cada capacidad se separa por lenguaje desde el inicio: `saas/go`, `http/go`, `authn/ts`, `ai/python`, etc.
- Prohibido crear código o manifests en la raíz de una capacidad; la raíz solo organiza
- Prohibido crear Go fuera de `go/`, Python fuera de `python/`, Rust fuera de `rust/` o TypeScript fuera de `ts/`
- Prohibido usar una sola versión global del repo; cada implementación tiene su propio `VERSION`
- Los tags de release son por subdirectorio: `{modulo}/{runtime}/vX.Y.Z`
- No meter dominio específico de Nexus, Pymes, Ponti, Fixguard o KMA/AlphaCoding
- Hexagonal se evalúa por dirección de dependencias y aislamiento del dominio, no por el nombre de las carpetas
- La estructura `usecases/domain`, `handler/dto` y `repository/models` es convención obligatoria para módulos Go con dominio real
- Rastrear siempre consumidores y productores afectados antes de afirmar que un cambio es seguro
- No decir "listo" sin verificación
