# browser

Helpers reutilizables para persistencia y utilidades del runtime browser.

Implementación actual: `browser/ts/`

## Pertenece

- namespaces de `localStorage`/`sessionStorage`
- lectura y escritura tipada de strings o JSON
- helpers de limpieza por prefijo

## No pertenece

- tokens de auth
- i18n de un producto
- theme de un producto
- selección de workspace de una app

Los consumidores deben construir su propio estado de producto arriba de esta base.
