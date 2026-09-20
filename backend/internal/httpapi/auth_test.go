package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
)

type verifierStub struct {
	claims identity.Claims
}

func (v verifierStub) Verify(context.Context, string) (identity.Claims, error) {
	return v.claims, nil
}

type userProvisionerStub struct {
	user dbgen.User
}

func (s userProvisionerStub) UpsertUserFromIdentity(context.Context, dbgen.UpsertUserFromIdentityParams) (dbgen.User, error) {
	return s.user, nil
}

func TestAuthenticateSetsPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := [16]byte{1, 2, 3}
	router := gin.New()
	router.Use(authenticate(
		verifierStub{claims: identity.Claims{Subject: "issuer|123"}},
		userProvisionerStub{user: dbgen.User{ID: pgtype.UUID{Bytes: userID, Valid: true}}},
	))
	router.GET("/private", func(ctx *gin.Context) {
		principal, ok := identity.PrincipalFromContext(ctx.Request.Context())
		if !ok || principal.Subject != "issuer|123" {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAuthenticateRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestID(), authenticate(verifierStub{}, userProvisionerStub{}))
	router.GET("/private", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/private", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
