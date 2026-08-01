# Google Calendar SDK para Go

Cliente HTTP low-level y agnóstico de producto para Google Calendar API.
El módulo no conoce tenants, usuarios internos, persistencia ni reglas
de scheduling. El consumer conserva y cifra tokens, decide sus retries y mapea
los recursos de Google a su propio dominio.

Módulo:

```text
github.com/devpablocristo/platform/sdks/google-calendar/go
```

## Superficie

- OAuth authorization-code: URL de consent, exchange, refresh y revoke.
- CRUD de eventos y paginación/listado.
- Create/get/update/delete de calendarios secundarios.
- Consulta `freeBusy`.
- Creación de Google Meet con `conferenceDataVersion=1`.
- Concurrencia optimista con `ETag` e `If-Match`.
- Errores tipados de validación, transporte, respuesta, OAuth y Calendar API.

No hay dependencias Go externas: el transporte usa `net/http` y se puede
inyectar con `httptest` o un `RoundTripper` fake.

## OAuth

```go
cfg := google.Config{
    ClientID:     clientID,
    ClientSecret: clientSecret,
    RedirectURL:  callbackURL,
    Scopes: []string{
        google.ScopeCalendarEvents,
        google.ScopeCalendarCalendars,
        google.ScopeCalendarFreeBusy,
    },
}

consentURL, err := google.BuildAuthURL(cfg, csrfState)
token, err := google.ExchangeCode(ctx, cfg, authorizationCode)
refreshed, err := google.Refresh(ctx, cfg, token.RefreshToken)
err = google.Revoke(ctx, cfg, token.RefreshToken)
```

El consumer genera, guarda y valida `state`; también conserva el refresh token
original cuando Google no devuelve uno nuevo durante un refresh.

## Eventos, Meet y ETag

```go
client, err := google.NewClient(google.ClientConfig{
    AccessToken: token.AccessToken,
})

event, err := client.CreateEvent(ctx, calendarID, google.EventInput{
    Summary: "Planning",
    Start: google.EventDateTime{DateTime: "2026-08-01T10:00:00Z"},
    End:   google.EventDateTime{DateTime: "2026-08-01T11:00:00Z"},
    ConferenceData: google.NewMeetConferenceData(requestID),
}, google.CreateEventOptions{SendUpdates: google.SendUpdatesAll})

if event.ConferenceData.Pending() {
    // Google todavía está aprovisionando el Meet. Volver a leer el evento.
}

updated, err := client.UpdateEvent(
    ctx,
    calendarID,
    event.ID,
    input,
    google.UpdateEventOptions{ETag: event.ETag},
)
```

Create y update envían siempre `conferenceDataVersion=1`. Para evitar
lost-updates, pasar el último `ETag`; Google responde `412` si quedó obsoleto.

```go
if google.IsPreconditionFailed(err) {
    // releer, reconciliar y reintentar con un ETag nuevo
}
if google.IsConflict(err) {
    // por ejemplo, requestId de conferencia duplicado
}
if google.IsTimeout(err) {
    // resultado incierto: reconciliar antes de repetir una escritura
}
```

`errors.As` permite acceder a `*google.APIError`, `*google.OAuthError`,
`*google.TransportError`, `*google.ResponseError` y
`*google.ValidationError`.

## FreeBusy

```go
availability, err := client.QueryFreeBusy(ctx, google.FreeBusyRequest{
    TimeMin: "2026-08-01T00:00:00Z",
    TimeMax: "2026-08-02T00:00:00Z",
    Items:   []google.FreeBusyItem{{ID: calendarID}},
})
```

Los timestamps del SDK son strings RFC3339 para conservar el contrato wire de
Google sin imponer clocks, zonas horarias ni modelos de dominio al consumer.
