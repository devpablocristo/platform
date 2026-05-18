# providers

Adapters concretos atados a un proveedor externo.

La categoría existe cuando el proveedor sí define la naturaleza del módulo. Ejemplos claros:

- AWS Lambda
- AWS S3
- AWS SQS

Si el concepto principal es una categoría técnica estable como base de datos, esa capacidad vive en su categoría (`databases/`) y no bajo `providers/`.
