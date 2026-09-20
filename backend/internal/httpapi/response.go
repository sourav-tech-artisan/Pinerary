package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/apierror"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorBody{
		Code:      code,
		Message:   message,
		RequestID: ctx.GetString(requestIDHeader),
	}})
}

func writeServiceError(ctx *gin.Context, err error) {
	publicErr := apierror.Public(err)
	writeError(ctx, publicErr.Status, publicErr.Code, publicErr.Message)
}

func notFound(ctx *gin.Context) {
	writeError(ctx, http.StatusNotFound, "not_found", "the requested resource was not found")
}

func methodNotAllowed(ctx *gin.Context) {
	writeError(ctx, http.StatusMethodNotAllowed, "method_not_allowed", "the HTTP method is not allowed")
}
