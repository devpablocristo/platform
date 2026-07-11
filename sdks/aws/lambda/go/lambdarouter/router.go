// Package lambdarouter es un router mínimo para AWS Lambda sobre API Gateway
// v2 (HTTP API). Matchea método + patrón de path con soporte de path params
// (":param"), normaliza el prefijo de stage y delega en handlers
// `lambdahttp.HandlerFunc`.
//
// Complementa a `lambdahttp` (helpers de request/response): platform no tenía
// un router para el sabor Lambda — los toolkits de ruteo vivían solo en
// `http/gin/go` (servidor tradicional), inutilizables en arquitecturas
// serverless. Cualquier proyecto Lambda+APIGWv2 lo reutiliza.
package lambdarouter

import (
	"context"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/devpablocristo/platform/sdks/aws/lambda/go/lambdahttp"
)

// Route define una ruta HTTP con método y patrón de path.
type Route struct {
	Method  string
	Path    string // ej: "/v1/assets/:id"
	Handler lambdahttp.HandlerFunc
}

// Router matchea método + patrón de path para APIGWv2.
type Router struct {
	routes []Route
}

// New crea un router vacío.
func New() *Router { return &Router{} }

// Add registra una ruta con método arbitrario.
func (r *Router) Add(method, path string, handler lambdahttp.HandlerFunc) {
	r.routes = append(r.routes, Route{
		Method:  strings.ToUpper(method),
		Path:    path,
		Handler: handler,
	})
}

// GET registra una ruta GET.
func (r *Router) GET(path string, h lambdahttp.HandlerFunc) { r.Add(http.MethodGet, path, h) }

// POST registra una ruta POST.
func (r *Router) POST(path string, h lambdahttp.HandlerFunc) { r.Add(http.MethodPost, path, h) }

// PUT registra una ruta PUT.
func (r *Router) PUT(path string, h lambdahttp.HandlerFunc) { r.Add(http.MethodPut, path, h) }

// PATCH registra una ruta PATCH.
func (r *Router) PATCH(path string, h lambdahttp.HandlerFunc) { r.Add(http.MethodPatch, path, h) }

// DELETE registra una ruta DELETE.
func (r *Router) DELETE(path string, h lambdahttp.HandlerFunc) { r.Add(http.MethodDelete, path, h) }

// Handler devuelve el HandlerFunc principal que rutea requests. Ante ninguna
// coincidencia responde 404 con el envelope de error canónico de lambdahttp.
func (r *Router) Handler() lambdahttp.HandlerFunc {
	return func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		method := strings.ToUpper(req.RequestContext.HTTP.Method)
		req = normalizeStagePath(req)
		path := req.RequestContext.HTTP.Path

		for _, route := range r.routes {
			if route.Method != method {
				continue
			}
			if matchPath(route.Path, path) {
				matched := req
				if params := extractPathParams(route.Path, path); len(params) > 0 {
					matched.PathParameters = params
				}
				return route.Handler(ctx, matched)
			}
		}
		return lambdahttp.Error(http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

// normalizeStagePath quita el prefijo de stage ("/stg", etc.) del path para que
// los patrones de ruta se escriban sin él. Con el stage "$default" no hay prefijo.
func normalizeStagePath(req events.APIGatewayV2HTTPRequest) events.APIGatewayV2HTTPRequest {
	stage := strings.Trim(req.RequestContext.Stage, "/")
	if stage == "" || stage == "$default" {
		return normalizeEmptyPath(req)
	}
	req.RequestContext.HTTP.Path = stripStagePrefix(req.RequestContext.HTTP.Path, stage)
	if req.RawPath != "" {
		req.RawPath = stripStagePrefix(req.RawPath, stage)
	}
	return normalizeEmptyPath(req)
}

func normalizeEmptyPath(req events.APIGatewayV2HTTPRequest) events.APIGatewayV2HTTPRequest {
	if strings.TrimSpace(req.RequestContext.HTTP.Path) == "" {
		req.RequestContext.HTTP.Path = "/"
	}
	return req
}

func stripStagePrefix(path, stage string) string {
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	prefix := "/" + stage
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

// extractPathParams rellena los path params a partir del patrón (p.ej. /v1/assets/:id).
func extractPathParams(pattern, path string) map[string]string {
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")
	if pattern == "" && path == "" {
		return nil
	}
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if len(patternParts) != len(pathParts) {
		return nil
	}
	out := make(map[string]string)
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			out[strings.TrimPrefix(part, ":")] = pathParts[i]
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// matchPath compara un patrón ("/v1/assets/:id") con un path real.
func matchPath(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			continue // wildcard
		}
		if part != pathParts[i] {
			return false
		}
	}
	return true
}
