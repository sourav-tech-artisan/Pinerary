package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/media"
)

type mediaService interface {
	Complete(context.Context, uuid.UUID, uuid.UUID, string) (media.CompleteResult, error)
	ListStopPhotos(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]media.PhotoView, error)
	Reserve(context.Context, media.ReserveInput) (media.UploadIntent, error)
}

func (h mediaHandler) listStopPhotos(ctx *gin.Context) {
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	stopID, ok := pathUUID(ctx, "stopId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	photos, err := h.service.ListStopPhotos(ctx.Request.Context(), principal.UserID, journeyID, stopID)
	if err != nil {
		writeMediaError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": photos})
}

type mediaHandler struct {
	service mediaService
}

type reservePhotoRequest struct {
	JourneyID       uuid.UUID  `json:"journey_id"`
	StopID          *uuid.UUID `json:"stop_id"`
	ClientRequestID uuid.UUID  `json:"client_request_id"`
	ContentType     string     `json:"content_type"`
	CapturedAt      *time.Time `json:"captured_at"`
}

type completePhotoRequest struct {
	ChecksumSHA256 string `json:"checksum_sha256"`
}

func (h mediaHandler) reserve(ctx *gin.Context) {
	var request reservePhotoRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	intent, err := h.service.Reserve(ctx.Request.Context(), media.ReserveInput{
		OwnerID: principal.UserID, JourneyID: request.JourneyID, StopID: request.StopID,
		ClientRequestID: request.ClientRequestID, ContentType: request.ContentType,
		CapturedAt: request.CapturedAt,
	})
	if err != nil {
		writeMediaError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, intent)
}

func (h mediaHandler) complete(ctx *gin.Context) {
	var request completePhotoRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	photoID, ok := pathUUID(ctx, "photoId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	result, err := h.service.Complete(ctx.Request.Context(), principal.UserID, photoID, request.ChecksumSHA256)
	if err != nil {
		writeMediaError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, result)
}

func writeMediaError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, media.ErrNotFound):
		notFound(ctx)
	case errors.Is(err, media.ErrInvalidPhoto), errors.Is(err, media.ErrUploadInvalid):
		writeError(ctx, http.StatusUnprocessableEntity, "invalid_photo", err.Error())
	case errors.Is(err, media.ErrUploadState):
		writeError(ctx, http.StatusConflict, "photo_state_conflict", err.Error())
	default:
		writeServiceError(ctx, err)
	}
}
