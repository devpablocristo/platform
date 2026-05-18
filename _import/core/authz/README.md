# authz

Capacidad reusable para autorización basada en roles y scopes.

Implementación actual: `authz/go/`

## Pertenece

- constantes de scopes
- checks de scopes
- checks de roles
- adapters livianos para exponer autorización reusable

## No pertenece

- resolución de identidad
- storage de sesión del browser
- tenancy, billing o entitlements
- middleware específico de una app

## Fuente inicial esperada

- `/home/pablo/Projects/Pablo/saas-core/shared/authz`
