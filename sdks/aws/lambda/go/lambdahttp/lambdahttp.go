package lambdahttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// HandlerFunc representa un handler HTTP sobre API Gateway v2.
type HandlerFunc func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

// APIError representa el envelope de error HTTP canónico.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

// Wrap envuelve un handler con logging y fallback genérico de error.
func Wrap(handler HandlerFunc, logger *slog.Logger) HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		logger.InfoContext(ctx,
			"processing lambda http request",
			"method", Method(request),
			"path", Path(request),
			"request_id", request.RequestContext.RequestID,
		)

		response, err := handler(ctx, request)
		if err != nil {
			logger.ErrorContext(ctx,
				"lambda http handler failed",
				"method", Method(request),
				"path", Path(request),
				"request_id", request.RequestContext.RequestID,
				"error", err.Error(),
			)
			return Error(http.StatusInternalServerError, "INTERNAL", "internal server error")
		}
		return response, nil
	}
}

// DecodeJSONBody decodifica JSON y rechaza campos desconocidos o payload extra.
func DecodeJSONBody(body string, dst any) error {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return errors.New("unexpected trailing data")
}

// JSON construye una respuesta JSON con el status indicado.
func JSON(statusCode int, payload any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

// Error construye el envelope de error HTTP canónico.
func Error(statusCode int, code, message string) (events.APIGatewayV2HTTPResponse, error) {
	return JSON(statusCode, errorEnvelope{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	})
}

// OK devuelve 200 con payload JSON.
func OK(payload any) (events.APIGatewayV2HTTPResponse, error) {
	return JSON(http.StatusOK, payload)
}

// Created devuelve 201 con payload JSON.
func Created(payload any) (events.APIGatewayV2HTTPResponse, error) {
	return JSON(http.StatusCreated, payload)
}

// NoContent devuelve 204 sin body.
func NoContent() (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNoContent}, nil
}

// Path devuelve el path efectivo de la request.
func Path(request events.APIGatewayV2HTTPRequest) string {
	if request.RawPath != "" {
		return request.RawPath
	}
	return request.RequestContext.HTTP.Path
}

// Method devuelve el método HTTP efectivo de la request.
func Method(request events.APIGatewayV2HTTPRequest) string {
	if request.RequestContext.HTTP.Method != "" {
		return request.RequestContext.HTTP.Method
	}
	return request.RouteKey
}

// PathParam obtiene un path parameter por nombre.
func PathParam(request events.APIGatewayV2HTTPRequest, key string) string {
	if request.PathParameters == nil {
		return ""
	}
	return request.PathParameters[key]
}

// QueryParam obtiene un query param por nombre.
func QueryParam(request events.APIGatewayV2HTTPRequest, key string) string {
	if request.QueryStringParameters == nil {
		return ""
	}
	return request.QueryStringParameters[key]
}

// IntQueryParam obtiene un query param entero con default explícito.
func IntQueryParam(request events.APIGatewayV2HTTPRequest, key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(QueryParam(request, key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}
