package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type userProvisioner interface {
	UpsertUserFromIdentity(context.Context, dbgen.UpsertUserFromIdentityParams) (dbgen.User, error)
}

func authenticate(verifier identity.Verifier, users userProvisioner) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if verifier == nil || users == nil {
			writeError(ctx, http.StatusServiceUnavailable, "authentication_unavailable", "authentication is unavailable")
			ctx.Abort()
			return
		}

		rawToken, ok := bearerToken(ctx.GetHeader("Authorization"))
		if !ok {
			writeError(ctx, http.StatusUnauthorized, "unauthorized", "a bearer access token is required")
			ctx.Abort()
			return
		}

		claims, err := verifier.Verify(ctx.Request.Context(), rawToken)
		if err != nil {
			writeError(ctx, http.StatusUnauthorized, "unauthorized", "the access token is invalid")
			ctx.Abort()
			return
		}

		user, err := users.UpsertUserFromIdentity(ctx.Request.Context(), dbgen.UpsertUserFromIdentityParams{
			OidcSubject: claims.Subject,
			DisplayName: claims.DisplayName,
		})
		if err != nil || !user.ID.Valid {
			writeServiceError(ctx, err)
			ctx.Abort()
			return
		}

		principal := identity.Principal{UserID: uuid.UUID(user.ID.Bytes), Subject: claims.Subject}
		ctx.Request = ctx.Request.WithContext(identity.WithPrincipal(ctx.Request.Context(), principal))
		ctx.Next()
	}
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}
