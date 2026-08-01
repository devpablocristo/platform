// El puente es código de PRODUCCIÓN que sólo se ejecuta dentro de Lambda: quien sirve el
// mismo http.Handler fuera de Lambda —local, tests, un contenedor— nunca pasa por acá. Sin
// estos tests, todo este camino queda sin ejecutar hasta el primer request real en AWS.
package lambdahttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func event(method, rawPath, rawQuery, body string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{
		RawPath:        rawPath,
		RawQueryString: rawQuery,
		Body:           body,
		Headers:        map[string]string{},
	}
	request.RequestContext.HTTP.Method = method
	request.RequestContext.HTTP.Path = rawPath
	return request
}

func TestHandlerTranslatesMethodPathQueryAndBody(t *testing.T) {
	var seen *http.Request
	var payload []byte
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		payload, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	response, err := handler(context.Background(),
		event(http.MethodPost, "/v1/documents/upload-url", "lifecycle=active&disposition=inline", `{"file_name":"a.txt"}`))
	if err != nil {
		t.Fatalf("the adapter must not return an error to Lambda: %v", err)
	}

	if seen.Method != http.MethodPost {
		t.Errorf("method = %q", seen.Method)
	}
	if seen.URL.Path != "/v1/documents/upload-url" {
		t.Errorf("path = %q", seen.URL.Path)
	}
	if seen.URL.Query().Get("lifecycle") != "active" || seen.URL.Query().Get("disposition") != "inline" {
		t.Errorf("query = %v", seen.URL.Query())
	}
	if string(payload) != `{"file_name":"a.txt"}` {
		t.Errorf("body = %q", payload)
	}
	if seen.ContentLength != int64(len(`{"file_name":"a.txt"}`)) {
		t.Errorf("content length = %d", seen.ContentLength)
	}
	if response.StatusCode != http.StatusCreated || response.Body != `{"ok":true}` {
		t.Errorf("unexpected response %+v", response)
	}
	if response.IsBase64Encoded {
		t.Error("a JSON response must not be base64 encoded")
	}
}

func TestHandlerRoutesThroughAServeMuxWithPathValues(t *testing.T) {
	// Lo que realmente importa: los patrones método+path de Go 1.22 y r.PathValue
	// tienen que funcionar detrás del bridge, porque así están escritos todos los
	// handlers.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/documents/{document_id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("document_id")))
	})
	mux.HandleFunc("POST /v1/documents/{document_id}/finalize", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("finalize:" + r.PathValue("document_id")))
	})
	handler := Handler(mux)

	response, err := handler(context.Background(), event(http.MethodGet, "/v1/documents/doc-42", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Body != "doc-42" {
		t.Fatalf("path value did not survive the bridge: %+v", response)
	}

	response, err = handler(context.Background(), event(http.MethodPost, "/v1/documents/doc-42/finalize", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Body != "finalize:doc-42" {
		t.Fatalf("nested route did not match: %+v", response)
	}

	// Y un método no soportado lo rechaza el mux, como en local.
	response, err = handler(context.Background(), event(http.MethodDelete, "/v1/documents/doc-42", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 from the mux, got %d", response.StatusCode)
	}
}

func TestHandlerSplitsCommaJoinedHeaders(t *testing.T) {
	// APIGWv2 une los valores repetidos de un header con coma; net/http tiene que
	// verlos como el cliente los mandó.
	var seen http.Header
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = r.Header }))

	request := event(http.MethodGet, "/v1/documents", "", "")
	request.Headers["X-Medmory-Internal-Token"] = "secret"
	request.Headers["Accept"] = "application/json, text/plain"
	request.Cookies = []string{"a=1", "b=2"}

	if _, err := handler(context.Background(), request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen.Get("X-Medmory-Internal-Token") != "secret" {
		t.Errorf("the internal token must survive: %v", seen)
	}
	if values := seen.Values("Accept"); len(values) != 2 {
		t.Errorf("expected the joined header to be split, got %v", values)
	}
	if values := seen.Values("Cookie"); len(values) != 2 {
		t.Errorf("expected both cookies, got %v", values)
	}
}

func TestHandlerDecodesABase64EncodedRequestBody(t *testing.T) {
	var payload []byte
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ = io.ReadAll(r.Body)
	}))

	binary := []byte{0x00, 0x01, 0xff, 0xfe}
	request := event(http.MethodPost, "/v1/documents", "", base64.StdEncoding.EncodeToString(binary))
	request.IsBase64Encoded = true

	if _, err := handler(context.Background(), request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(payload, binary) {
		t.Fatalf("binary body was not decoded, got %v", payload)
	}
}

func TestHandlerRejectsAMalformedBase64BodyWithoutFailingTheInvocation(t *testing.T) {
	reached := false
	handler := Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	request := event(http.MethodPost, "/v1/documents", "", "!!!not base64!!!")
	request.IsBase64Encoded = true

	response, err := handler(context.Background(), request)
	// Devolver un error haría que Lambda reintente un request que nunca va a
	// mejorar; se responde 400 y listo.
	if err != nil {
		t.Fatalf("a malformed body must not fail the invocation: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
	if reached {
		t.Fatal("the handler must not see a malformed request")
	}
}

func TestHandlerBase64EncodesABinaryResponse(t *testing.T) {
	binary := []byte{0x89, 0x50, 0x4e, 0x47}
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(binary)
	}))

	response, err := handler(context.Background(), event(http.MethodGet, "/v1/documents", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !response.IsBase64Encoded {
		t.Fatal("a binary response must be base64 encoded")
	}
	decoded, err := base64.StdEncoding.DecodeString(response.Body)
	if err != nil || !bytes.Equal(decoded, binary) {
		t.Fatalf("the binary response did not round-trip: %v", err)
	}
}

func TestHandlerPreservesAStatusWrittenWithoutABody(t *testing.T) {
	// /healthz y todas las respuestas 204 escriben sólo el status.
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response, err := handler(context.Background(), event(http.MethodGet, "/healthz", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusNoContent || response.Body != "" {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestHandlerDefaultsAStatusWhenTheHandlerOnlyWrites(t *testing.T) {
	// Un handler que escribe sin llamar WriteHeader implica 200, igual que en
	// net/http.
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	response, err := handler(context.Background(), event(http.MethodGet, "/", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Body != "ok" {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestHandlerKeepsTheFirstStatusWritten(t *testing.T) {
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("conflict"))
	}))
	response, err := handler(context.Background(), event(http.MethodGet, "/", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("the first status must win, got %d", response.StatusCode)
	}
}

func TestHandlerSplitsRepeatedResponseHeaders(t *testing.T) {
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	response, err := handler(context.Background(), event(http.MethodGet, "/", "", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Headers["Content-Type"] != "application/json" {
		t.Errorf("single-valued headers go in Headers, got %v", response.Headers)
	}
	if len(response.MultiValueHeaders["Set-Cookie"]) != 2 {
		t.Errorf("repeated headers go in MultiValueHeaders, got %v", response.MultiValueHeaders)
	}
}

func TestHandlerFallsBackWhenRawPathAndMethodAreAbsent(t *testing.T) {
	var seen *http.Request
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = r }))

	var request events.APIGatewayV2HTTPRequest
	request.Headers = map[string]string{}
	request.RequestContext.HTTP.Path = "/v1/diagnosis"
	request.RequestContext.HTTP.Method = "post"

	if _, err := handler(context.Background(), request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen.URL.Path != "/v1/diagnosis" {
		t.Errorf("path fallback failed: %q", seen.URL.Path)
	}
	// El método llega en minúscula en algunos payloads; net/http compara exacto.
	if seen.Method != http.MethodPost {
		t.Errorf("method must be upcased, got %q", seen.Method)
	}

	request = events.APIGatewayV2HTTPRequest{Headers: map[string]string{}}
	if _, err := handler(context.Background(), request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen.URL.Path != "/" || seen.Method != http.MethodGet {
		t.Errorf("an empty event must degrade to GET /, got %s %s", seen.Method, seen.URL.Path)
	}
}

func TestHandlerPropagatesTheInvocationContext(t *testing.T) {
	type key struct{}
	var found any
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		found = r.Context().Value(key{})
	}))

	ctx := context.WithValue(context.Background(), key{}, "carried")
	if _, err := handler(ctx, event(http.MethodGet, "/", "", "")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != "carried" {
		t.Fatalf("the invocation context must reach the handler, got %v", found)
	}
}

func TestIsTextContentClassifiesResponses(t *testing.T) {
	for contentType, isText := range map[string]bool{
		"":                                  true,
		"application/json; charset=utf-8":   true,
		"text/plain":                        true,
		"application/xml":                   true,
		"application/x-www-form-urlencoded": true,
		"image/png":                         false,
		"application/pdf":                   false,
		"application/octet-stream":          false,
	} {
		if got := isTextContent(contentType); got != isText {
			t.Errorf("isTextContent(%q) = %v, want %v", contentType, got, isText)
		}
	}
}

func TestRequestTargetIncludesTheQueryString(t *testing.T) {
	var seen *http.Request
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = r }))
	if _, err := handler(context.Background(), event(http.MethodGet, "/v1/documents", "lifecycle=archived", "")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(seen.RequestURI, "lifecycle=archived") {
		t.Fatalf("RequestURI must carry the query, got %q", seen.RequestURI)
	}
}

func TestHandlerStripsTheStagePrefixApiGatewayPrepends(t *testing.T) {
	// El defecto que ningún test cubría y que apareció recién contra AWS: con el
	// stage `stg`, API Gateway entrega `/stg/healthz`, y el mux —que registra
	// `/healthz`— devolvía 404 en TODA la superficie pública.
	var seenDocument string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /v1/documents/{document_id}", func(w http.ResponseWriter, r *http.Request) {
		seenDocument = r.PathValue("document_id")
		w.WriteHeader(http.StatusOK)
	})
	handler := Handler(mux)

	staged := event(http.MethodGet, "/stg/healthz", "", "")
	staged.RequestContext.Stage = "stg"
	response, err := handler(context.Background(), staged)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("un stage con nombre tiene que llegar al mux sin prefijo, status = %d", response.StatusCode)
	}

	// Y los path values se extraen del path real, no del que trae el stage pegado.
	staged = event(http.MethodGet, "/stg/v1/documents/doc-7", "", "")
	staged.RequestContext.Stage = "stg"
	if response, err = handler(context.Background(), staged); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusOK || seenDocument != "doc-7" {
		t.Fatalf("status = %d, document_id = %q", response.StatusCode, seenDocument)
	}
}

func TestWithoutStagePrefixOnlyStripsWhatTheStageAdded(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		stage string
		want  string
	}{
		{"stage con nombre", "/stg/healthz", "stg", "/healthz"},
		{"el stage solo queda en la raíz", "/stg", "stg", "/"},
		{"$default no antepone nada", "/healthz", "$default", "/healthz"},
		{"una Function URL no manda stage", "/internal/uploads/reap", "", "/internal/uploads/reap"},
		// Con dominio propio y base path mapping el prefijo NO viaja: recortar a
		// ciegas mutilaría el path.
		{"prefijo ausente", "/healthz", "stg", "/healthz"},
		// Un segmento que sólo EMPIEZA con el nombre del stage no es el stage.
		{"prefijo parcial", "/stgx/healthz", "stg", "/stgx/healthz"},
		// API Gateway antepone SIEMPRE, así que una ruta que de verdad empezara con el
		// nombre del stage llegaría duplicada y se recorta una sola vez.
		{"ruta homónima del stage", "/stg/stg/healthz", "stg", "/stg/healthz"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := withoutStagePrefix(testCase.path, testCase.stage); got != testCase.want {
				t.Fatalf("withoutStagePrefix(%q, %q) = %q, want %q",
					testCase.path, testCase.stage, got, testCase.want)
			}
		})
	}
}
