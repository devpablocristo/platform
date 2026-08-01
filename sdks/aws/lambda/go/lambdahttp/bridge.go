package lambdahttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Handler adapta un `http.Handler` al contrato de API Gateway v2.
//
// Es la dirección contraria a `lambdarouter`: allá el handler canónico es
// APIGWv2-nativo y las rutas se declaran contra ese contrato; acá el handler canónico es
// stdlib —`http.ServeMux`, middlewares de `net/http`, `httptest`— y Lambda es lo que se
// adapta.
//
// La diferencia importa cuando el mismo servicio corre en Lambda y fuera de ella. Con un
// handler Lambda-nativo hay que envolverlo para servirlo local, así que desarrollo y
// producción ejecutan caminos distintos y el que se prueba no es el que se despliega. Con
// este puente hay UN handler: `cmd/api` lo sirve con `http.ListenAndServe` y `cmd/lambda`
// lo pasa por acá.
//
//	handler := wire.Build(...)        // http.Handler
//	lambda.Start(lambdahttp.Handler(handler))
func Handler(handler http.Handler) HandlerFunc {
	return func(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		request, err := toRequest(ctx, event)
		if err != nil {
			return Error(http.StatusBadRequest, "INVALID_REQUEST", "malformed request")
		}
		recorder := &responseRecorder{header: http.Header{}, status: http.StatusOK}
		handler.ServeHTTP(recorder, request)
		return recorder.response(), nil
	}
}

// StagePath devuelve el path SIN el prefijo del stage.
//
// API Gateway antepone el nombre del stage cuando el stage no es `$default`: con el stage
// `stg`, un GET a `/healthz` llega como `/stg/healthz`, y eso vale tanto para `RawPath`
// como para `RequestContext.HTTP.Path`. Las rutas de una aplicación se escriben sin stage,
// así que dejar pasar el prefijo devuelve 404 en TODA la superficie pública — y el stack se
// despliega entero y sano, con lo cual sólo se descubre probando contra AWS real.
//
// `Path` devuelve el path crudo y se conserva así para no cambiarle el comportamiento a
// quien ya lo usa; `lambdarouter` normaliza por su cuenta. Usar ésta cuando el path va a
// compararse contra rutas declaradas sin stage.
func StagePath(event events.APIGatewayV2HTTPRequest) string {
	candidate := strings.TrimSpace(event.RawPath)
	if candidate == "" {
		candidate = strings.TrimSpace(event.RequestContext.HTTP.Path)
	}
	if candidate == "" {
		return "/"
	}
	return withoutStagePrefix(candidate, event.RequestContext.Stage)
}

// withoutStagePrefix recorta el segmento del stage SÓLO si está realmente presente.
//
// Con un dominio propio y base path mapping el prefijo no viaja, así que recortar a ciegas
// mutilaría el path: un servicio en el dominio `api.ejemplo.com` con stage `prod` recibiría
// `/prod/uctos` convertido en `/uctos`.
func withoutStagePrefix(path, stage string) string {
	stage = strings.Trim(strings.TrimSpace(stage), "/")
	if stage == "" || stage == "$default" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	prefix := "/" + stage
	if path == prefix {
		return "/"
	}
	if trimmed := strings.TrimPrefix(path, prefix+"/"); trimmed != path {
		return "/" + trimmed
	}
	return path
}

// toRequest construye la request de net/http a partir del evento.
func toRequest(ctx context.Context, event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	body := []byte(event.Body)
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return nil, err
		}
		body = decoded
	}

	target := StagePath(event)
	if raw := strings.TrimSpace(event.RawQueryString); raw != "" {
		target += "?" + raw
	}

	request, err := http.NewRequestWithContext(ctx, requestMethod(event), target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// APIGWv2 entrega los valores repetidos de un header unidos por coma; se vuelven a
	// separar para que net/http los vea como el cliente los envió.
	for name, value := range event.Headers {
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				request.Header.Add(name, item)
			}
		}
	}
	for _, cookie := range event.Cookies {
		request.Header.Add("Cookie", cookie)
	}
	request.RequestURI = target
	request.ContentLength = int64(len(body))
	if host := request.Header.Get("Host"); host != "" {
		request.Host = host
	}
	return request, nil
}

func requestMethod(event events.APIGatewayV2HTTPRequest) string {
	if candidate := strings.TrimSpace(event.RequestContext.HTTP.Method); candidate != "" {
		return strings.ToUpper(candidate)
	}
	return http.MethodGet
}

// responseRecorder captura lo que el http.Handler escribe.
//
// Se implementa acá en lugar de usar `httptest.NewRecorder` para no arrastrar el paquete de
// tests a un camino de producción.
type responseRecorder struct {
	header  http.Header
	status  int
	body    bytes.Buffer
	written bool
}

func (r *responseRecorder) Header() http.Header { return r.header }

func (r *responseRecorder) WriteHeader(status int) {
	if r.written {
		return
	}
	r.status = status
	r.written = true
}

func (r *responseRecorder) Write(payload []byte) (int, error) {
	r.WriteHeader(r.status)
	return r.body.Write(payload)
}

// response traduce lo capturado al contrato de APIGWv2.
//
// El cuerpo viaja en base64 salvo que el Content-Type sea texto. Al revés —mandar bytes
// binarios como string— el runtime los corrompe silenciosamente, así que un PDF o una
// imagen llegan rotos y con status 200.
func (r *responseRecorder) response() events.APIGatewayV2HTTPResponse {
	response := events.APIGatewayV2HTTPResponse{
		StatusCode:        r.status,
		Headers:           map[string]string{},
		MultiValueHeaders: map[string][]string{},
	}
	for name, values := range r.header {
		if len(values) == 1 {
			response.Headers[name] = values[0]
			continue
		}
		response.MultiValueHeaders[name] = values
	}
	if isTextContent(r.header.Get("Content-Type")) {
		response.Body = r.body.String()
		return response
	}
	response.Body = base64.StdEncoding.EncodeToString(r.body.Bytes())
	response.IsBase64Encoded = true
	return response
}

func isTextContent(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case contentType == "":
		return true
	case strings.HasPrefix(contentType, "text/"):
		return true
	case strings.Contains(contentType, "json"):
		return true
	case strings.Contains(contentType, "xml"):
		return true
	case strings.Contains(contentType, "x-www-form-urlencoded"):
		return true
	default:
		return false
	}
}
