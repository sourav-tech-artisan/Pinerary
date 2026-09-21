package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func requestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetHeader(requestIDHeader)
		if id == "" {
			var value [16]byte
			if _, err := rand.Read(value[:]); err == nil {
				id = hex.EncodeToString(value[:])
			} else {
				id = "unavailable"
			}
		}

		ctx.Set(requestIDHeader, id)
		ctx.Header(requestIDHeader, id)
		ctx.Next()
	}
}

func accessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		loggedPath := ctx.Request.URL.Path
		if ctx.Request.Method == http.MethodGet && strings.HasPrefix(loggedPath, "/s/") {
			loggedPath = "/s/:token"
		}
		if ctx.Request.Method == http.MethodGet && strings.HasPrefix(loggedPath, "/api/v1/shares/") {
			loggedPath = "/api/v1/shares/:token"
		}
		logger.InfoContext(ctx.Request.Context(), "request completed",
			"request_id", ctx.GetString(requestIDHeader),
			"method", ctx.Request.Method,
			"path", loggedPath,
			"status", ctx.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	}
}

func recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx.Request.Context(), "request panicked",
					"request_id", ctx.GetString(requestIDHeader),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				writeError(ctx, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
				ctx.Abort()
			}
		}()

		ctx.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		ctx.Header("Referrer-Policy", "no-referrer")
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.Header("X-Frame-Options", "DENY")
		ctx.Next()
	}
}

func cors(allowedOrigins []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" && slices.Contains(allowedOrigins, origin) {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			ctx.Header("Vary", "Origin")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
