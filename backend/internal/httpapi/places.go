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
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/places"
)

type placeService interface {
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	List(context.Context, uuid.UUID, int32) ([]places.Place, error)
	ListStops(context.Context, uuid.UUID, uuid.UUID) ([]places.Stop, error)
	Pin(context.Context, places.PinInput) (places.Stop, error)
	Save(context.Context, places.SaveInput) (places.Place, error)
	Update(context.Context, uuid.UUID, uuid.UUID, string, string) (places.Place, error)
	UpdateStop(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string) (places.Stop, error)
}

type placeHandler struct {
	service placeService
}

type savePlaceRequest struct {
	ClientRequestID uuid.UUID `json:"client_request_id"`
	Name            string    `json:"name"`
	Notes           string    `json:"notes"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
}

type updatePlaceRequest struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

type pinStopRequest struct {
	ClientRequestID uuid.UUID `json:"client_request_id"`
	Name            string    `json:"name"`
	Note            string    `json:"note"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	CapturedAt      time.Time `json:"captured_at"`
}

type updateStopRequest struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type placeResponse struct {
	ID              uuid.UUID `json:"id"`
	ClientRequestID uuid.UUID `json:"client_request_id"`
	Name            string    `json:"name"`
	Notes           string    `json:"notes"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type stopResponse struct {
	ID              uuid.UUID `json:"id"`
	JourneyID       uuid.UUID `json:"journey_id"`
	PlaceID         uuid.UUID `json:"place_id"`
	ClientRequestID uuid.UUID `json:"client_request_id"`
	SequenceNumber  int64     `json:"sequence_number"`
	CapturedAt      time.Time `json:"captured_at"`
	Name            string    `json:"name"`
	Note            string    `json:"note"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
}

func (h placeHandler) save(ctx *gin.Context) {
	var request savePlaceRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	place, err := h.service.Save(ctx.Request.Context(), places.SaveInput{
		OwnerID: principal.UserID, ClientRequestID: request.ClientRequestID,
		Name: request.Name, Notes: request.Notes,
		Coordinates: places.Coordinates{Latitude: request.Latitude, Longitude: request.Longitude},
	})
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, toPlaceResponse(place))
}

func (h placeHandler) list(ctx *gin.Context) {
	limit := int32(100)
	if value := ctx.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 500 {
			writeError(ctx, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = int32(parsed)
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	result, err := h.service.List(ctx.Request.Context(), principal.UserID, limit)
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	items := make([]placeResponse, 0, len(result))
	for _, place := range result {
		items = append(items, toPlaceResponse(place))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h placeHandler) update(ctx *gin.Context) {
	var request updatePlaceRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	placeID, ok := pathUUID(ctx, "placeId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	place, err := h.service.Update(ctx.Request.Context(), principal.UserID, placeID, request.Name, request.Notes)
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toPlaceResponse(place))
}

func (h placeHandler) delete(ctx *gin.Context) {
	placeID, ok := pathUUID(ctx, "placeId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	if err := h.service.Delete(ctx.Request.Context(), principal.UserID, placeID); err != nil {
		writePlaceError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h placeHandler) pinStop(ctx *gin.Context) {
	var request pinStopRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	stop, err := h.service.Pin(ctx.Request.Context(), places.PinInput{
		OwnerID: principal.UserID, JourneyID: journeyID, ClientRequestID: request.ClientRequestID,
		Name: request.Name, Note: request.Note, CapturedAt: request.CapturedAt,
		Coordinates: places.Coordinates{Latitude: request.Latitude, Longitude: request.Longitude},
	})
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, toStopResponse(stop))
}

func (h placeHandler) listStops(ctx *gin.Context) {
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	result, err := h.service.ListStops(ctx.Request.Context(), principal.UserID, journeyID)
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	items := make([]stopResponse, 0, len(result))
	for _, stop := range result {
		items = append(items, toStopResponse(stop))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h placeHandler) updateStop(ctx *gin.Context) {
	var request updateStopRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	stopID, ok := pathUUID(ctx, "stopId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	stop, err := h.service.UpdateStop(ctx.Request.Context(), principal.UserID, journeyID, stopID, request.Name, request.Note)
	if err != nil {
		writePlaceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toStopResponse(stop))
}

func writePlaceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, places.ErrNotFound):
		notFound(ctx)
	case errors.Is(err, places.ErrJourneyNotActive):
		writeError(ctx, http.StatusConflict, "journey_not_active", err.Error())
	case errors.Is(err, places.ErrInvalidName), errors.Is(err, places.ErrInvalidCoordinates):
		writeError(ctx, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	default:
		writeServiceError(ctx, err)
	}
}

func toPlaceResponse(place places.Place) placeResponse {
	return placeResponse{
		ID: place.ID, ClientRequestID: place.ClientRequestID, Name: place.Name, Notes: place.Notes,
		Latitude: place.Coordinates.Latitude, Longitude: place.Coordinates.Longitude,
		CreatedAt: place.CreatedAt, UpdatedAt: place.UpdatedAt,
	}
}

func toStopResponse(stop places.Stop) stopResponse {
	return stopResponse{
		ID: stop.ID, JourneyID: stop.JourneyID, PlaceID: stop.PlaceID,
		ClientRequestID: stop.ClientRequestID, SequenceNumber: stop.SequenceNumber,
		CapturedAt: stop.CapturedAt, Name: stop.DisplayName, Note: stop.Note,
		Latitude: stop.Coordinates.Latitude, Longitude: stop.Coordinates.Longitude,
	}
}
