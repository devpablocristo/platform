# postgres

Adapter reusable para PostgreSQL.

## Pertenece

- pool PostgreSQL
- healthcheck PostgreSQL
- config PostgreSQL por env
- helpers `pgx` o `database/sql` específicos de PostgreSQL

## No pertenece

- runtime backend genérico
- dominio de negocio
- adapters de otras bases de datos

## Fuentes iniciales esperadas

- `/home/pablo/Projects/Pablo/nexus/v2/pkgs/go-pkg/postgres`
- `/home/pablo/Projects/Pablo/nexus/v3/pkgs/go-pkg/postgres`

## Bootstrap inicial creado

Implementación actual: `databases/postgres/go/`

Paquetes ya iniciados en esta implementación:

- package raíz `postgres`

El primer recorte ya incluye:

- config por env
- apertura de pool
- healthcheck
- `MigrateUp` sobre `.sql` ordenados por scope
