# core/activity/go

**Estado: experimental — no consumir desde código productivo.**

Sin tag estable. La API puede cambiar sin previo aviso hasta que se publique
`activity/go/v0.1.0`. Para proponer promoción a estable, abrir issue en el repo.

## Qué es

Primitivas para registrar y consultar actividad/auditoría tipada en dominios SaaS.
Subpaquetes:

- `audit/` — modelo + repository + usecases para asentar eventos de auditoría
- `kernel/` — usecases núcleo compartidos
- `timeline/` — proyección de eventos como timeline consultable
