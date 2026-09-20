package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type RouterConfig struct {
	AllowedOrigins  []string
	Logger          *slog.Logger
	Ready           func(context.Context) error
	UserProvisioner userProvisioner
	Verifier        identity.Verifier
}

func NewRouter(config RouterConfig) *gin.Engine {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		requestID(),
		securityHeaders(),
		cors(config.AllowedOrigins),
		accessLog(logger),
		recovery(logger),
	)
	router.NoRoute(notFound)
	router.NoMethod(methodNotAllowed)
	router.GET("/openapi.yaml", serveOpenAPI)

	api := router.Group("/api/v1")
	api.Use(authenticate(config.Verifier, config.UserProvisioner))

	router.GET("/health/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health/ready", func(ctx *gin.Context) {
		if config.Ready != nil {
			if err := config.Ready(ctx.Request.Context()); err != nil {
				writeError(ctx, http.StatusServiceUnavailable, "not_ready", "the service is not ready")
				return
			}
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
