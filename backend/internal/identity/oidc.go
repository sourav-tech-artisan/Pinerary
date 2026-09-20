package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCVerifier(ctx context.Context, issuerURL, audience string) (*OIDCVerifier, error) {
	if issuerURL == "" || audience == "" {
		return nil, fmt.Errorf("OIDC issuer and audience are required")
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}

	return &OIDCVerifier{verifier: provider.Verifier(&oidc.Config{ClientID: audience})}, nil
}

func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	var tokenClaims struct {
		Subject string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
	}
	if err := token.Claims(&tokenClaims); err != nil {
		return Claims{}, fmt.Errorf("%w: decode claims", ErrInvalidToken)
	}
	if strings.TrimSpace(tokenClaims.Subject) == "" {
		return Claims{}, fmt.Errorf("%w: subject is missing", ErrInvalidToken)
	}

	displayName := strings.TrimSpace(tokenClaims.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(tokenClaims.Email)
	}
	return Claims{Subject: tokenClaims.Subject, DisplayName: displayName}, nil
}

type DevelopmentVerifier struct{}

func (DevelopmentVerifier) Verify(_ context.Context, rawToken string) (Claims, error) {
	subject := strings.TrimSpace(rawToken)
	if subject == "" {
		return Claims{}, ErrInvalidToken
	}
	return Claims{Subject: "development:" + subject, DisplayName: subject}, nil
}
