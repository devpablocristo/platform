# sdks/aws/lambda/go

## `lambdahttp.Handler` — servir un `http.Handler` desde Lambda (v0.3.0)

Es la dirección contraria a `lambdarouter`: allá el handler canónico es APIGWv2-nativo; acá
es stdlib y Lambda es lo que se adapta.

```go
handler := wire.Build(...)        // http.Handler
lambda.Start(lambdahttp.Handler(handler))
```

La diferencia importa cuando el mismo servicio corre dentro y fuera de Lambda. Con un
handler Lambda-nativo hay que envolverlo para servirlo local, así que desarrollo y
producción ejecutan caminos distintos y el que se prueba no es el que se despliega.

`StagePath` recorta el prefijo que API Gateway antepone cuando el stage no es `$default`
(con stage `stg`, `/healthz` llega como `/stg/healthz`). Dejarlo pasar devuelve **404 en
toda la superficie pública**, con el stack desplegado y sano — sólo se descubre probando
contra AWS real. `Path` sigue devolviendo el path crudo, para no cambiarle el
comportamiento a quien ya lo usa.
