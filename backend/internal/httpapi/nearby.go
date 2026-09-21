package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/nearby"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
)

type nearbyService interface {
	Find(context.Context, uuid.UUID, routing.Coordinate, int32, routing.Mode) ([]nearby.Result, error)
}

type nearbyHandler struct {
	service nearbyService
}

func (h nearbyHandler) find(ctx *gin.Context) {
	latitude, latitudeErr := strconv.ParseFloat(ctx.Query("latitude"), 64)
	longitude, longitudeErr := strconv.ParseFloat(ctx.Query("longitude"), 64)
	limit := int64(10)
	var limitErr error
	if value := ctx.Query("limit"); value != "" {
		limit, limitErr = strconv.ParseInt(value, 10, 32)
	}
	mode := routing.Mode(ctx.DefaultQuery("mode", string(routing.ModeMotorcycle)))
	if latitudeErr != nil || longitudeErr != nil || limitErr != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_nearby_query", "latitude, longitude, limit, or mode is invalid")
		return
	}

	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	results, err := h.service.Find(ctx.Request.Context(), principal.UserID, routing.Coordinate{
		Latitude: latitude, Longitude: longitude,
	}, int32(limit), mode)
	if err != nil {
		switch {
		case errors.Is(err, nearby.ErrInvalidQuery):
			writeError(ctx, http.StatusBadRequest, "invalid_nearby_query", err.Error())
		case errors.Is(err, nearby.ErrRoutingUnavailable):
			writeError(ctx, http.StatusServiceUnavailable, "routing_unavailable", err.Error())
		default:
			writeServiceError(ctx, err)
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"sort_mode": mode,
		"items":     results,
	})
}
