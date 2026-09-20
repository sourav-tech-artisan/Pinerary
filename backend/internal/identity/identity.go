package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Claims struct {
	Subject     string
	DisplayName string
}

type Verifier interface {
	Verify(context.Context, string) (Claims, error)
}

type Principal struct {
	UserID  uuid.UUID
	Subject string
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

var ErrInvalidToken = errors.New("invalid access token")
