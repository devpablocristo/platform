# `@devpablocristo/platform-http`

Cliente HTTP TypeScript agnóstico para los productos del ecosistema. La API
recomendada crea clientes aislados por instancia; el paquete no lee API keys,
organizaciones ni credenciales desde estado global.

## Cliente por instancia

```ts
import { createHttpClient } from "@devpablocristo/platform-http";

const api = createHttpClient({
  baseURL: "https://api.example.com",
  fetch: window.fetch.bind(window),
  resolveHeaders: async () => ({
    Authorization: `Bearer ${await getToken()}`,
  }),
});

const session = await api.request<Session>("/api/v1/session", {
  signal: abortController.signal,
});
```

`resolveHeaders` se ejecuta en cada solicitud y puede devolver headers de forma
síncrona o asíncrona. Los headers de la solicitud tienen precedencia sobre los
resueltos por la instancia. `AbortSignal` se propaga tanto al resolver como a
`fetch`.

El `baseURL` pertenece al cliente y no puede reemplazarse por request. Las URLs
absolutas se conservan, lo que permite destinos prefirmados explícitos.

## Compatibilidad

Las funciones de módulo `request` y `requestResponse` siguen disponibles y
usan `globalThis.fetch`. Conservan `baseURLs` para los consumidores existentes.
El cliente creado con `createHttpClient` no usa esa opción legacy.

Los errores HTTP no exitosos se exponen como `HttpError`; `request<T>` devuelve
JSON, texto o `undefined` para respuestas `204`.
