package lambdarouter

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/devpablocristo/platform/sdks/aws/lambda/go/lambdahttp"
)

func req(method, path, stage string) events.APIGatewayV2HTTPRequest {
	var r events.APIGatewayV2HTTPRequest
	r.RequestContext.HTTP.Method = method
	r.RequestContext.HTTP.Path = path
	r.RawPath = path
	r.RequestContext.Stage = stage
	return r
}

func TestStaticRouteAndMethod(t *testing.T) {
	r := New()
	r.GET("/healthz", func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return lambdahttp.OK(map[string]string{"status": "ok"})
	})
	h := r.Handler()

	resp, err := h(context.Background(), req(http.MethodGet, "/healthz", "$default"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz = %d", resp.StatusCode)
	}
	// método incorrecto → 404
	resp, _ = h(context.Background(), req(http.MethodPost, "/healthz", "$default"))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("POST /healthz = %d, want 404", resp.StatusCode)
	}
}

func TestPathParams(t *testing.T) {
	r := New()
	var gotID string
	r.GET("/v1/assets/:id", func(_ context.Context, in events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		gotID = lambdahttp.PathParam(in, "id")
		return lambdahttp.NoContent()
	})
	resp, err := r.Handler()(context.Background(), req(http.MethodGet, "/v1/assets/abc-123", "$default"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if gotID != "abc-123" {
		t.Fatalf("path param id = %q, want abc-123", gotID)
	}
}

func TestStagePrefixStripped(t *testing.T) {
	r := New()
	hit := false
	r.GET("/v1/ping", func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		hit = true
		return lambdahttp.NoContent()
	})
	// APIGW manda el path con el prefijo del stage: /stg/v1/ping
	resp, _ := r.Handler()(context.Background(), req(http.MethodGet, "/stg/v1/ping", "stg"))
	if !hit || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ruta con stage no matcheó: hit=%v status=%d", hit, resp.StatusCode)
	}
}

func TestNotFound(t *testing.T) {
	r := New()
	resp, err := r.Handler()(context.Background(), req(http.MethodGet, "/nope", "$default"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
