package ginmw

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAPIKeyFromEnv_AllowsHealthz(t *testing.T) {
	t.Setenv("X_API_KEY", "")

	router := gin.New()
	router.Use(RequireAPIKeyFromEnv("X_API_KEY"))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKeyFromEnv_AllowsReadyz(t *testing.T) {
	t.Setenv("X_API_KEY", "")

	router := gin.New()
	router.Use(RequireAPIKeyFromEnv("X_API_KEY"))
	router.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKeyFromEnv_FailsClosedWhenEnvMissing(t *testing.T) {
	t.Setenv("X_API_KEY", "")

	router := gin.New()
	router.Use(RequireAPIKeyFromEnv("X_API_KEY"))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAPIKeyFromEnv_RejectsInvalidKey(t *testing.T) {
	t.Setenv("X_API_KEY", "secret")

	router := gin.New()
	router.Use(RequireAPIKeyFromEnv("X_API_KEY"))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("X-API-KEY", "invalid")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAPIKeyFromEnv_AcceptsConfiguredKey(t *testing.T) {
	t.Setenv("X_API_KEY", "secret")

	router := gin.New()
	router.Use(RequireAPIKeyFromEnv("X_API_KEY"))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("X-API-KEY", os.Getenv("X_API_KEY"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
