package httpapi

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var openAPISpec []byte

func serveOpenAPI(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", openAPISpec)
}
