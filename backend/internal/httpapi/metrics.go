package httpapi

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requests      atomic.Uint64
	errors        atomic.Uint64
	inflight      atomic.Int64
	durationNanos atomic.Uint64
}

func metricsMiddleware(metrics *Metrics) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		metrics.inflight.Add(1)
		defer metrics.inflight.Add(-1)
		ctx.Next()

		metrics.requests.Add(1)
		metrics.durationNanos.Add(uint64(time.Since(startedAt)))
		if ctx.Writer.Status() >= http.StatusInternalServerError {
			metrics.errors.Add(1)
		}
	}
}

func (m *Metrics) serve(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	ctx.String(http.StatusOK, fmt.Sprintf(
		"# TYPE pinerary_http_requests_total counter\n"+
			"pinerary_http_requests_total %d\n"+
			"# TYPE pinerary_http_errors_total counter\n"+
			"pinerary_http_errors_total %d\n"+
			"# TYPE pinerary_http_inflight_requests gauge\n"+
			"pinerary_http_inflight_requests %d\n"+
			"# TYPE pinerary_http_request_duration_seconds_total counter\n"+
			"pinerary_http_request_duration_seconds_total %.6f\n",
		m.requests.Load(), m.errors.Load(), m.inflight.Load(),
		float64(m.durationNanos.Load())/float64(time.Second),
	))
}
