package ginmw

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

)

// ReadinessCheck verifica que un servicio esté listo.
type ReadinessCheck func(context.Context) error

// RegisterHealthEndpoints registra /healthz y /readyz en un gin.Engine o gin.RouterGroup.
func RegisterHealthEndpoints(r gin.IRoutes, ready ReadinessCheck) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if ready != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			if err := ready(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "not_ready"})
				return
			}
		}
		c.JSON(http.StatusOK, HealthResponse{Status: "ready"})
	})
}
