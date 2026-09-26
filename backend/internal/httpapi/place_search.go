package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/placesearch"
)

type placeSearchService interface {
	Search(context.Context, placesearch.SuggestInput) ([]placesearch.Suggestion, error)
}

type placeSearchHandler struct {
	service placeSearchService
}

func (h placeSearchHandler) search(ctx *gin.Context) {
	limit := int64(5)
	var limitErr error
	if value := ctx.Query("limit"); value != "" {
		limit, limitErr = strconv.ParseInt(value, 10, 32)
	}
	if limitErr != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_search_query", "query or limit is invalid")
		return
	}

	results, err := h.service.Search(ctx.Request.Context(), placesearch.SuggestInput{
		Query: ctx.Query("q"), Limit: int32(limit), AcceptLanguage: ctx.GetHeader("Accept-Language"),
	})
	if err != nil {
		switch {
		case errors.Is(err, placesearch.ErrInvalidQuery):
			writeError(ctx, http.StatusBadRequest, "invalid_search_query", "query must contain 3 to 200 characters and limit must be between 1 and 10")
		case errors.Is(err, placesearch.ErrQuotaExceeded):
			ctx.Header("Retry-After", strconv.Itoa(placesearch.SecondsUntilUTCReset(time.Now())))
			writeError(ctx, http.StatusTooManyRequests, "place_search_quota_exhausted", "place suggestions are unavailable until the daily allowance resets")
		case errors.Is(err, placesearch.ErrProviderUnavailable):
			writeError(ctx, http.StatusServiceUnavailable, "place_search_unavailable", "place suggestions are temporarily unavailable")
		default:
			writeServiceError(ctx, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items": results,
		"attribution": gin.H{
			"provider": "Powered by Geoapify", "provider_url": "https://www.geoapify.com/",
			"data": "© OpenStreetMap contributors", "data_url": "https://www.openstreetmap.org/copyright",
		},
	})
}
