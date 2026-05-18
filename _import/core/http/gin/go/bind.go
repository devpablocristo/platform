package ginmw

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/devpablocristo/core/errors/go/domainerr"
)

// BindJSON parsea el body JSON, valida los campos con binding tags
// y responde automáticamente con un error humanizado si falla.
// Devuelve nil si el bind fue exitoso; error si falló (ya respondido al cliente).
func BindJSON(c *gin.Context, req any) error {
	if err := c.ShouldBindJSON(req); err != nil {
		msg := HumanizeBindError(err)
		domErr := domainerr.Validation(msg)
		Respond(c, domErr)
		return err
	}
	return nil
}

// HumanizeBindError convierte errores de validator, json.UnmarshalTypeError
// y json.SyntaxError a mensajes legibles para el cliente.
func HumanizeBindError(err error) string {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		var issues []string
		for _, fe := range validationErrs {
			field := strings.ToLower(fe.Field())
			switch fe.Tag() {
			case "required":
				issues = append(issues, field+" is required")
			case "min":
				issues = append(issues, field+" must be at least "+fe.Param())
			case "max":
				issues = append(issues, field+" must be at most "+fe.Param())
			case "oneof":
				issues = append(issues, field+" must be one of: "+fe.Param())
			default:
				issues = append(issues, field+" is invalid ("+fe.Tag()+")")
			}
		}
		if len(issues) > 0 {
			return "invalid payload: " + strings.Join(issues, "; ")
		}
	}

	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		if unmarshalTypeErr.Field != "" {
			return "invalid payload: wrong type for '" + unmarshalTypeErr.Field + "'"
		}
		return "invalid payload: wrong data type"
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return "invalid payload: malformed JSON"
	}

	return "invalid payload: " + err.Error()
}
