package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type profileStore interface {
	DeleteDevice(context.Context, dbgen.DeleteDeviceParams) (int64, error)
	GetUserByID(context.Context, pgtype.UUID) (dbgen.User, error)
	ListDevices(context.Context, pgtype.UUID) ([]dbgen.Device, error)
	UpdateUserPreferences(context.Context, dbgen.UpdateUserPreferencesParams) (dbgen.User, error)
	UpsertDevice(context.Context, dbgen.UpsertDeviceParams) (dbgen.Device, error)
}

type profileHandler struct {
	store profileStore
}

type userResponse struct {
	ID                   uuid.UUID `json:"id"`
	Subject              string    `json:"subject"`
	DisplayName          string    `json:"display_name"`
	DefaultTransportMode string    `json:"default_transport_mode"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type updateProfileRequest struct {
	DisplayName          string `json:"display_name"`
	DefaultTransportMode string `json:"default_transport_mode"`
}

type registerDeviceRequest struct {
	InstallationID   string          `json:"installation_id"`
	Platform         string          `json:"platform"`
	PushSubscription json.RawMessage `json:"push_subscription"`
}

type deviceResponse struct {
	ID             uuid.UUID `json:"id"`
	InstallationID string    `json:"installation_id"`
	Platform       string    `json:"platform"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

func (h profileHandler) getMe(ctx *gin.Context) {
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	user, err := h.store.GetUserByID(ctx.Request.Context(), pgUUID(principal.UserID))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toUserResponse(user))
}

func (h profileHandler) updateMe(ctx *gin.Context) {
	var request updateProfileRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	if len(request.DisplayName) > 160 || !validTransportMode(request.DefaultTransportMode) {
		writeError(ctx, http.StatusUnprocessableEntity, "validation_failed", "profile values are invalid")
		return
	}

	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	user, err := h.store.UpdateUserPreferences(ctx.Request.Context(), dbgen.UpdateUserPreferencesParams{
		ID:                   pgUUID(principal.UserID),
		DisplayName:          request.DisplayName,
		DefaultTransportMode: request.DefaultTransportMode,
	})
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toUserResponse(user))
}

func (h profileHandler) registerDevice(ctx *gin.Context) {
	var request registerDeviceRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	if request.InstallationID == "" || len(request.InstallationID) > 255 ||
		(request.Platform != "web" && request.Platform != "android") {
		writeError(ctx, http.StatusUnprocessableEntity, "validation_failed", "device values are invalid")
		return
	}

	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	device, err := h.store.UpsertDevice(ctx.Request.Context(), dbgen.UpsertDeviceParams{
		UserID:           pgUUID(principal.UserID),
		InstallationID:   request.InstallationID,
		Platform:         request.Platform,
		PushSubscription: request.PushSubscription,
	})
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, toDeviceResponse(device))
}

func (h profileHandler) listDevices(ctx *gin.Context) {
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	devices, err := h.store.ListDevices(ctx.Request.Context(), pgUUID(principal.UserID))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}

	items := make([]deviceResponse, 0, len(devices))
	for _, device := range devices {
		items = append(items, toDeviceResponse(device))
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (h profileHandler) deleteDevice(ctx *gin.Context) {
	deviceID, err := uuid.Parse(ctx.Param("deviceId"))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_id", "device ID must be a UUID")
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	deleted, err := h.store.DeleteDevice(ctx.Request.Context(), dbgen.DeleteDeviceParams{
		ID:     pgUUID(deviceID),
		UserID: pgUUID(principal.UserID),
	})
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	if deleted == 0 {
		notFound(ctx)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func toUserResponse(user dbgen.User) userResponse {
	return userResponse{
		ID:                   uuid.UUID(user.ID.Bytes),
		Subject:              user.OidcSubject,
		DisplayName:          user.DisplayName,
		DefaultTransportMode: user.DefaultTransportMode,
		CreatedAt:            user.CreatedAt.Time,
		UpdatedAt:            user.UpdatedAt.Time,
	}
}

func toDeviceResponse(device dbgen.Device) deviceResponse {
	return deviceResponse{
		ID:             uuid.UUID(device.ID.Bytes),
		InstallationID: device.InstallationID,
		Platform:       device.Platform,
		LastSeenAt:     device.LastSeenAt.Time,
	}
}

func validTransportMode(mode string) bool {
	return mode == "motorcycle" || mode == "car" || mode == "walking"
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}
