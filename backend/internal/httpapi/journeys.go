package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/journeys"
)

type journeyService interface {
	Complete(context.Context, uuid.UUID, uuid.UUID, int32) (journeys.Journey, error)
	ConvertToTrip(context.Context, uuid.UUID, uuid.UUID, int32) (journeys.Journey, error)
	Create(context.Context, journeys.CreateInput) (journeys.Journey, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (journeys.Journey, error)
	List(context.Context, uuid.UUID, *time.Time, int32) ([]journeys.Journey, error)
	Rename(context.Context, uuid.UUID, uuid.UUID, int32, string) (journeys.Journey, error)
}

type journeyHandler struct {
	service journeyService
}

type createJourneyRequest struct {
	ClientRequestID uuid.UUID     `json:"client_request_id"`
	Kind            journeys.Kind `json:"kind"`
	Label           string        `json:"label"`
}

type updateJourneyRequest struct {
	Label   string `json:"label"`
	Version int32  `json:"version"`
}

type versionRequest struct {
	Version int32 `json:"version"`
}

type journeyResponse struct {
	ID        uuid.UUID       `json:"id"`
	Kind      journeys.Kind   `json:"kind"`
	Label     string          `json:"label"`
	Status    journeys.Status `json:"status"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   *time.Time      `json:"ended_at"`
	Version   int32           `json:"version"`
}

func (h journeyHandler) create(ctx *gin.Context) {
	var request createJourneyRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	journey, err := h.service.Create(ctx.Request.Context(), journeys.CreateInput{
		OwnerID: principal.UserID, ClientRequestID: request.ClientRequestID,
		Kind: request.Kind, Label: request.Label,
	})
	if err != nil {
		writeJourneyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, toJourneyResponse(journey))
}

func (h journeyHandler) get(ctx *gin.Context) {
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	journey, err := h.service.Get(ctx.Request.Context(), principal.UserID, journeyID)
	if err != nil {
		writeJourneyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toJourneyResponse(journey))
}

func (h journeyHandler) list(ctx *gin.Context) {
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	pageSize := int32(25)
	if value := ctx.Query("page_size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(ctx, http.StatusBadRequest, "invalid_page_size", "page_size must be between 1 and 100")
			return
		}
		pageSize = int32(parsed)
	}

	var before *time.Time
	if value := ctx.Query("before"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(ctx, http.StatusBadRequest, "invalid_cursor", "before must be an RFC3339 timestamp")
			return
		}
		before = &parsed
	}

	result, err := h.service.List(ctx.Request.Context(), principal.UserID, before, pageSize)
	if err != nil {
		writeJourneyError(ctx, err)
		return
	}
	items := make([]journeyResponse, 0, len(result))
	for _, journey := range result {
		items = append(items, toJourneyResponse(journey))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h journeyHandler) rename(ctx *gin.Context) {
	var request updateJourneyRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	journey, err := h.service.Rename(ctx.Request.Context(), principal.UserID, journeyID, request.Version, request.Label)
	if err != nil {
		writeJourneyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toJourneyResponse(journey))
}

func (h journeyHandler) complete(ctx *gin.Context) {
	h.transition(ctx, h.service.Complete)
}

func (h journeyHandler) convertToTrip(ctx *gin.Context) {
	h.transition(ctx, h.service.ConvertToTrip)
}

func (h journeyHandler) transition(
	ctx *gin.Context,
	transition func(context.Context, uuid.UUID, uuid.UUID, int32) (journeys.Journey, error),
) {
	var request versionRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	journey, err := transition(ctx.Request.Context(), principal.UserID, journeyID, request.Version)
	if err != nil {
		writeJourneyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toJourneyResponse(journey))
}

func pathUUID(ctx *gin.Context, parameter string) (uuid.UUID, bool) {
	value, err := uuid.Parse(ctx.Param(parameter))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_id", parameter+" must be a UUID")
		return uuid.Nil, false
	}
	return value, true
}

func writeJourneyError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, journeys.ErrNotFound):
		notFound(ctx)
	case errors.Is(err, journeys.ErrActiveJourneyExists), errors.Is(err, journeys.ErrConflict), errors.Is(err, journeys.ErrInvalidTransition):
		writeError(ctx, http.StatusConflict, "journey_conflict", err.Error())
	case errors.Is(err, journeys.ErrInvalidKind), errors.Is(err, journeys.ErrInvalidLabel):
		writeError(ctx, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		writeServiceError(ctx, err)
	}
}

func toJourneyResponse(journey journeys.Journey) journeyResponse {
	return journeyResponse{
		ID: journey.ID, Kind: journey.Kind, Label: journey.Label, Status: journey.Status,
		StartedAt: journey.StartedAt, EndedAt: journey.EndedAt, Version: journey.Version,
	}
}
