package ginmw

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewBodySizeLimit crea un middleware que limita el tamaño del body.
// maxBytes default: 64KB.
func NewBodySizeLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 64 << 10
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// IsBodyTooLarge verifica si el error es de body demasiado grande.
func IsBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
