# saas

Capacidad reusable para tenancy, identidad y billing multi-tenant.

Implementación actual: `saas/go/`

`saas-core` ya quedó absorbido dentro de esta implementación. El repo viejo puede pasar a estado obsoleto una vez que los consumidores migren imports a `core/saas/go`.

## Pertenece

- admin de tenant y lifecycle multi-tenant
- orgs
- users
- identity
- billing
- entitlements
- usage metering
- middleware de auth que resuelve principal, tenant, role y scopes
- webhooks de identity provider cuando sincronizan users, orgs y memberships SaaS

## No pertenece

- onboarding específico de un producto
- UI/admin de una app específica
- copy comercial o dominio específico de un producto
- transporte de notificaciones (`smtp`, `ses`, `noop`)
- helpers HTTP genéricos
- context keys, errores de dominio y métricas genéricas
- verificación `jwks`/`oidc` reusable fuera del dominio SaaS

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/saas-core`

## Estado de absorción

- absorbido: `org`, `users`, `identity`, `billing`, `clerkwebhook`, `admin`, `usagemetering`, `migrations`
- ampliado: `tenant/`, `kernel/usecases/domain/`, `handler/dto/`, `repository/models/`, runtime de billing y tests por contexto
- pendiente fuera de este módulo: migrar consumers y recién después eliminar código duplicado del repo `saas-core`
- extraído a módulos raíz: `authn/`, `authz/`, `http/`, `observability/`, `config/`, `security/`, `validate/`, `errors/`, `utils/`, `concurrency/` y `notifications/`

## Estructura actual

Este módulo ya aplica la convención Go de contexto + `usecases/domain` y quedó sin shims técnicos de compatibilidad.

Paquetes activos en `saas/go/`:

- `kernel/usecases/domain/` como fuente de verdad de tipos compartidos
- `identity/` con `usecases.go` y `usecases/domain/`
- `org/`
- `users/`
- `billing/`
- `admin/`
- `entitlement/` con `usecases.go` y `usecases/domain/`
- `tenant/` con `usecases.go` y `usecases/domain/`
- `usagemetering/`
- `middleware/` con auth middleware reusable
- `session/` — `GET /session` como eco JSON del `Principal` del kernel (sin lógica de producto)
- `handler/dto/` en los contexts de aplicación
- `repository/models/` en los contexts con puertos de persistencia o lookup
- `notifications/` como contrato SaaS de intención de notificación

## Fronteras

La regla para `saas` es simple: se queda acá todo lo que entiende de verdad el modelo multi-tenant.

Se quedan en `saas/go/`:

- `admin/`, porque maneja `tenant settings`, `plan`, lifecycle y límites
- `users/`, porque modela users, members y relaciones con orgs dentro de la plataforma
- `middleware/`, porque este middleware no es HTTP genérico; resuelve `Principal`, `TenantID`, `Role` y `Scopes`
- `clerkwebhook/`, porque sincroniza users, orgs y memberships de la plataforma

No se quedan en `saas/go/` las implementaciones técnicas genéricas:

- `notifications/go/` contiene transporte reusable (`noop`, `smtp`, `ses`)
- `saas/go/notifications/` contiene solo el puerto de dominio: la intención de notificar algo del tenant
- `authn/go/` contiene `jwks` y `oidc`
- `authz/go/` contiene roles/scopes reutilizables
- `http/`, `observability/`, `config/`, `security/`, `validate/`, `errors/`, `utils/` y `concurrency/` contienen la infraestructura reusable que ya no pertenece al dominio SaaS

La diferencia no es “usa auth” o “usa email”. La diferencia es: si la pieza necesita entender tenant, org, membership, plan o lifecycle, pertenece a `saas`. Si solo provee infraestructura reusable, sale a un módulo raíz.
