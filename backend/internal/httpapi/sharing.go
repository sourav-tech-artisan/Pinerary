package httpapi

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/sharing"
)

type sharingService interface {
	Create(context.Context, sharing.CreateInput) (sharing.CreatedShare, error)
	Resolve(context.Context, string) (sharing.PublicSnapshot, error)
	Revoke(context.Context, uuid.UUID, uuid.UUID) error
}

type sharingHandler struct {
	service sharingService
}

type createShareRequest struct {
	Title     string              `json:"title"`
	Items     []sharing.ItemInput `json:"items"`
	ExpiresAt *time.Time          `json:"expires_at"`
}

//go:embed share_page.html
var sharePageHTML string

var sharePageTemplate = template.Must(template.New("share").Funcs(template.FuncMap{
	"formatDate": func(value time.Time) string { return value.Format("2 Jan 2006") },
	"formatTime": func(value time.Time) string { return value.Format("3:04 PM") },
	"inc":        func(value int) int { return value + 1 },
	"mapURL": func(latitude, longitude float64) string {
		return fmt.Sprintf("https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=16/%.6f/%.6f", latitude, longitude, latitude, longitude)
	},
	"svgPoints": shareSVGPoints,
}).Parse(sharePageHTML))

func (h sharingHandler) create(ctx *gin.Context) {
	var request createShareRequest
	if !decodeJSON(ctx, &request) {
		return
	}
	journeyID, ok := pathUUID(ctx, "journeyId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	created, err := h.service.Create(ctx.Request.Context(), sharing.CreateInput{
		OwnerID: principal.UserID, JourneyID: journeyID, Title: request.Title,
		Items: request.Items, ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		writeSharingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, created)
}

func (h sharingHandler) resolveJSON(ctx *gin.Context) {
	snapshot, err := h.service.Resolve(ctx.Request.Context(), ctx.Param("token"))
	if err != nil {
		writeSharingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, snapshot)
}

func (h sharingHandler) resolvePage(ctx *gin.Context) {
	snapshot, err := h.service.Resolve(ctx.Request.Context(), ctx.Param("token"))
	if err != nil {
		writeSharingError(ctx, err)
		return
	}
	ctx.Header("Content-Security-Policy", "default-src 'none'; img-src https: http:; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	ctx.Header("Referrer-Policy", "no-referrer")
	ctx.Header("X-Robots-Tag", "noindex, nofollow")
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Status(http.StatusOK)
	if err := sharePageTemplate.Execute(ctx.Writer, snapshot); err != nil {
		ctx.Error(err) //nolint:errcheck // response has already started
	}
}

func (h sharingHandler) revoke(ctx *gin.Context) {
	shareID, ok := pathUUID(ctx, "shareId")
	if !ok {
		return
	}
	principal, _ := identity.PrincipalFromContext(ctx.Request.Context())
	if err := h.service.Revoke(ctx.Request.Context(), principal.UserID, shareID); err != nil {
		writeSharingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func writeSharingError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, sharing.ErrNotFound):
		notFound(ctx)
	case errors.Is(err, sharing.ErrInvalidShare):
		writeError(ctx, http.StatusUnprocessableEntity, "invalid_share", err.Error())
	default:
		writeServiceError(ctx, err)
	}
}

func shareSVGPoints(items []sharing.PublicItem) string {
	if len(items) == 0 {
		return ""
	}
	minLatitude, maxLatitude := items[0].Latitude, items[0].Latitude
	minLongitude, maxLongitude := items[0].Longitude, items[0].Longitude
	for _, item := range items[1:] {
		minLatitude, maxLatitude = math.Min(minLatitude, item.Latitude), math.Max(maxLatitude, item.Latitude)
		minLongitude, maxLongitude = math.Min(minLongitude, item.Longitude), math.Max(maxLongitude, item.Longitude)
	}
	latitudeSpan := math.Max(maxLatitude-minLatitude, 0.001)
	longitudeSpan := math.Max(maxLongitude-minLongitude, 0.001)
	points := make([]string, 0, len(items))
	for _, item := range items {
		x := 30 + ((item.Longitude-minLongitude)/longitudeSpan)*540
		y := 190 - ((item.Latitude-minLatitude)/latitudeSpan)*160
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}
	return strings.Join(points, " ")
}
