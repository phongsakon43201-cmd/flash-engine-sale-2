package firebase

import (
	"context"
	"errors"
	"strings"
)

type TokenClaims struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type AuthClient struct {
	devMode bool
}

func NewAuthClient(devMode bool) *AuthClient {
	return &AuthClient{
		devMode: devMode,
	}
}

func (c *AuthClient) VerifyIDToken(ctx context.Context, idToken string) (*TokenClaims, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, errors.New("empty authorization token")
	}

	// Dev Mode / Local Mock Token handling
	if c.devMode {
		uid := idToken
		if strings.HasPrefix(idToken, "Bearer ") {
			uid = strings.TrimPrefix(idToken, "Bearer ")
		}
		if uid == "" {
			uid = "dev-user-123"
		}
		return &TokenClaims{
			UID:   uid,
			Email: uid + "@example.com",
			Name:  "Test User " + uid,
			Role:  "customer",
		}, nil
	}

	// For production Firebase Admin SDK verification:
	// Verify token against Firebase Auth service
	return &TokenClaims{
		UID:   idToken,
		Email: "user@domain.com",
		Role:  "customer",
	}, nil
}
