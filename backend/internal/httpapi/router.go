package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type RouterConfig struct {
	AllowedOrigins   []string
	Logger           *slog.Logger
	Ready            func(context.Context) error
	UserProvisioner  userProvisioner
	ProfileStore     profileStore
	JourneyService   journeyService
	PlaceService     placeService
	GeocodingService geocodingService
	TrackingService  trackingService
	Verifier         identity.Verifier
}

func NewRouter(config RouterConfig) *gin.Engine {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		requestID(),
		securityHeaders(),
		cors(config.AllowedOrigins),
		accessLog(logger),
		recovery(logger),
	)
	router.NoRoute(notFound)
	router.NoMethod(methodNotAllowed)
	router.GET("/openapi.yaml", serveOpenAPI)

	api := router.Group("/api/v1")
	api.Use(authenticate(config.Verifier, config.UserProvisioner))
	profile := profileHandler{store: config.ProfileStore}
	api.GET("/me", profile.getMe)
	api.PATCH("/me", profile.updateMe)
	api.POST("/devices", profile.registerDevice)
	api.GET("/devices", profile.listDevices)
	api.DELETE("/devices/:deviceId", profile.deleteDevice)

	journey := journeyHandler{service: config.JourneyService}
	api.POST("/journeys", journey.create)
	api.GET("/journeys", journey.list)
	api.GET("/journeys/:journeyId", journey.get)
	api.PATCH("/journeys/:journeyId", journey.rename)
	api.POST("/journeys/:journeyId/end", journey.complete)
	api.POST("/journeys/:journeyId/convert-to-trip", journey.convertToTrip)

	place := placeHandler{service: config.PlaceService}
	api.POST("/places", place.save)
	api.GET("/places", place.list)
	api.PATCH("/places/:placeId", place.update)
	api.DELETE("/places/:placeId", place.delete)
	api.POST("/journeys/:journeyId/stops", place.pinStop)
	api.GET("/journeys/:journeyId/stops", place.listStops)
	api.PATCH("/journeys/:journeyId/stops/:stopId", place.updateStop)

	geocoder := geocodingHandler{service: config.GeocodingService}
	api.GET("/places/reverse-geocode", geocoder.reverse)

	tracking := trackingHandler{service: config.TrackingService}
	api.POST("/journeys/:journeyId/locations/batch", tracking.ingest)
	api.GET("/journeys/:journeyId/route", tracking.route)

	router.GET("/health/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health/ready", func(ctx *gin.Context) {
		if config.Ready != nil {
			if err := config.Ready(ctx.Request.Context()); err != nil {
				writeError(ctx, http.StatusServiceUnavailable, "not_ready", "the service is not ready")
				return
			}
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
