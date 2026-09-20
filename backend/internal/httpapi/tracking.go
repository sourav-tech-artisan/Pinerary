package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/tracking"
)

type trackingService interface {
	Ingest(context.Context, uuid.UUID, uuid.UUID, []tracking.Point) (tracking.IngestResult, error)
	Route(context.Context, uuid.UUID, uuid.UUID) ([]tracking.RouteSegment, error)
}

func (h trackingHandler) route(ctx *gin.Context) {
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	segments, err := h.service.Route(ctx.Request.Context(), principal.UserID, journeyID)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"segments": segments})
}

type trackingHandler struct {
	service trackingService
}

type ingestLocationsRequest struct {
	Points []tracking.Point `json:"points"`
}

func (h trackingHandler) ingest(ctx *gin.Context) {
	var request ingestLocationsRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	result, err := h.service.Ingest(ctx.Request.Context(), principal.UserID, journeyID, request.Points)
	if err != nil {
		switch {
		case errors.Is(err, tracking.ErrInvalidBatch):
			writeError(ctx, http.StatusUnprocessableEntity, "invalid_location_batch", err.Error())
		case errors.Is(err, tracking.ErrJourneyNotActive):
			writeError(ctx, http.StatusConflict, "journey_not_active", err.Error())
		default:
			writeServiceError(ctx, err)
		}
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}
