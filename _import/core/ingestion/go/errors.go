package ingestion

import "fmt"

// ExtractError error de negocio del contrato de extracción (código estable).
type ExtractError struct {
	Code    string
	Message string
}

func (e ExtractError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Code
}

// ErrorBody es el cuerpo JSON de error del servicio HTTP.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
