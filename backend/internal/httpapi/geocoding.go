package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/geocoding"
)

type geocodingService interface {
	Reverse(context.Context, float64, float64) (geocoding.Result, error)
}

type geocodingHandler struct {
	service geocodingService
}

func (h geocodingHandler) reverse(ctx *gin.Context) {
	latitude, latitudeErr := strconv.ParseFloat(ctx.Query("latitude"), 64)
	longitude, longitudeErr := strconv.ParseFloat(ctx.Query("longitude"), 64)
	if latitudeErr != nil || longitudeErr != nil || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		writeError(ctx, http.StatusBadRequest, "invalid_coordinates", "valid latitude and longitude are required")
		return
	}

	result, err := h.service.Reverse(ctx.Request.Context(), latitude, longitude)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"display_name": result.DisplayName})
}
