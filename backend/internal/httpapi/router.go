package httpapi

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	AllowedOrigins []string
	Logger         *slog.Logger
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

	router.GET("/health/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
