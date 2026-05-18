package lambdahttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestDecodeJSONBodyRejectsTrailingData(t *testing.T) {
	t.Parallel()

	var body struct {
		Name string `json:"name"`
	}
	if err := DecodeJSONBody(`{"name":"a"}{"name":"b"}`, &body); err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestErrorBuildsCanonicalEnvelope(t *testing.T) {
	t.Parallel()

	resp, err := Error(http.StatusUnauthorized, "UNAUTHORIZED", "valid api key required")
	if err != nil {
		t.Fatalf("Error returned err: %v", err)
	}
	if got, want := resp.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("unexpected status: got=%d want=%d", got, want)
	}
	if got, want := resp.Body, `{"error":{"code":"UNAUTHORIZED","message":"valid api key required"}}`; got != want {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestWrapReturnsInternalErrorOnHandlerFailure(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Wrap(func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return events.APIGatewayV2HTTPResponse{}, context.Canceled
	}, logger)

	resp, err := handler(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/resources",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-1",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
				Path:   "/v1/resources",
			},
		},
	})
	if err != nil {
		t.Fatalf("handler returned err: %v", err)
	}
	if got, want := resp.StatusCode, http.StatusInternalServerError; got != want {
		t.Fatalf("unexpected status: got=%d want=%d", got, want)
	}
}

func TestPathAndQueryHelpers(t *testing.T) {
	t.Parallel()

	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/resources/123",
		PathParameters: map[string]string{
			"id": "123",
		},
		QueryStringParameters: map[string]string{
			"limit": "25",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
				Path:   "/v1/resources/123",
			},
		},
	}

	if got, want := Path(req), "/v1/resources/123"; got != want {
		t.Fatalf("unexpected path: got=%q want=%q", got, want)
	}
	if got, want := Method(req), http.MethodGet; got != want {
		t.Fatalf("unexpected method: got=%q want=%q", got, want)
	}
	if got, want := PathParam(req, "id"), "123"; got != want {
		t.Fatalf("unexpected path param: got=%q want=%q", got, want)
	}
	if got, want := QueryParam(req, "limit"), "25"; got != want {
		t.Fatalf("unexpected query param: got=%q want=%q", got, want)
	}

	limit, err := IntQueryParam(req, "limit", 50)
	if err != nil {
		t.Fatalf("IntQueryParam returned err: %v", err)
	}
	if got, want := limit, 25; got != want {
		t.Fatalf("unexpected limit: got=%d want=%d", got, want)
	}
}

func TestIntQueryParamUsesDefault(t *testing.T) {
	t.Parallel()

	value, err := IntQueryParam(events.APIGatewayV2HTTPRequest{}, "limit", 50)
	if err != nil {
		t.Fatalf("IntQueryParam returned err: %v", err)
	}
	if got, want := value, 50; got != want {
		t.Fatalf("unexpected value: got=%d want=%d", got, want)
	}
}
