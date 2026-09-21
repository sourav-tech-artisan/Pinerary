package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type rateLimitBucket struct {
	startedAt time.Time
	lastSeen  time.Time
	requests  int
}

type fixedWindowLimiter struct {
	mu        sync.Mutex
	max       int
	window    time.Duration
	buckets   map[string]rateLimitBucket
	now       func() time.Time
	lastSweep time.Time
}

func newFixedWindowLimiter(max int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{max: max, window: window, buckets: make(map[string]rateLimitBucket), now: time.Now}
}

func (l *fixedWindowLimiter) middleware(scope string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.ClientIP()
		if principal, ok := identity.PrincipalFromContext(ctx.Request.Context()); ok {
			key = principal.UserID.String()
		}
		allowed, retryAfter := l.allow(scope + ":" + key)
		if !allowed {
			retrySeconds := int((retryAfter + time.Second - 1) / time.Second)
			ctx.Header("Retry-After", strconv.Itoa(max(1, retrySeconds)))
			writeError(ctx, http.StatusTooManyRequests, "rate_limited", "too many requests; retry later")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func (l *fixedWindowLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.lastSweep.IsZero() || now.Sub(l.lastSweep) >= l.window {
		for bucketKey, bucket := range l.buckets {
			if now.Sub(bucket.lastSeen) >= 2*l.window {
				delete(l.buckets, bucketKey)
			}
		}
		l.lastSweep = now
	}

	bucket, exists := l.buckets[key]
	if !exists || now.Sub(bucket.startedAt) >= l.window {
		l.buckets[key] = rateLimitBucket{startedAt: now, lastSeen: now, requests: 1}
		return true, 0
	}
	bucket.lastSeen = now
	if bucket.requests >= l.max {
		l.buckets[key] = bucket
		return false, l.window - now.Sub(bucket.startedAt)
	}
	bucket.requests++
	l.buckets[key] = bucket
	return true, 0
}
