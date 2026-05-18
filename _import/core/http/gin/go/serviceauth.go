package ginmw

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

)

// NewInternalServiceAuth valida el header X-Internal-Service-Token con constant-time comparison.
func NewInternalServiceAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(token) == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, SimpleErrorResponse{Error: "internal service auth not configured"})
			return
		}
		provided := strings.TrimSpace(c.GetHeader("X-Internal-Service-Token"))
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, SimpleErrorResponse{Error: "unauthorized"})
			return
		}
		c.Next()
	}
}
