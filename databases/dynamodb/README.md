# dynamodb

Adapter reusable para DynamoDB.

## Pertenece

- config DynamoDB
- client bootstrap DynamoDB
- marshaling/unmarshaling reusable
- operaciones comunes reutilizables sobre tablas DynamoDB

## No pertenece

- runtime Lambda
- wiring específico de API Gateway
- dominio de negocio

## Fuente inicial esperada

- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/databases/nosql/dynamodb`

## Bootstrap inicial creado

Implementación actual: `databases/dynamodb/go/`

Paquetes ya iniciados en esta implementación:

- package raíz `dynamodbtable`
