package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/apierror"
)

const maxJSONBodyBytes = 1 << 20

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

func decodeJSON(ctx *gin.Context, destination any) bool {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", "the request body is not valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(ctx, http.StatusBadRequest, "invalid_json", "the request body must contain one JSON value")
		return false
	}
	return true
}
