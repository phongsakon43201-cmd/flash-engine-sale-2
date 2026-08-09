package firebase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	adminfirebase "firebase.google.com/go/v4"
	adminauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type TokenClaims struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type AuthClient struct {
	devMode bool
	client  *adminauth.Client
}

func NewAuthClient(ctx context.Context, devMode bool, credentialsPath string) (*AuthClient, error) {
	if devMode {
		return &AuthClient{devMode: true}, nil
	}

	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	app, err := adminfirebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase app: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase auth client: %w", err)
	}

	return &AuthClient{client: client}, nil
}

func (c *AuthClient) VerifyIDToken(ctx context.Context, idToken string) (*TokenClaims, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, errors.New("empty authorization token")
	}

	// Local-only mock tokens. Prefix admin tokens with "admin:".
	if c.devMode {
		uid := idToken
		role := "customer"
		if strings.HasPrefix(uid, "admin:") {
			role = "admin"
			uid = strings.TrimPrefix(uid, "admin:")
		}
		if strings.TrimSpace(uid) == "" {
			return nil, errors.New("empty development user ID")
		}
		return &TokenClaims{
			UID:   uid,
			Email: uid + "@example.com",
			Name:  "Test User " + uid,
			Role:  role,
		}, nil
	}

	if c.client == nil {
		return nil, errors.New("Firebase auth client is not configured")
	}

	token, err := c.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("verify Firebase ID token: %w", err)
	}

	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)
	role, _ := token.Claims["role"].(string)
	if role != "admin" && role != "seller" {
		role = "customer"
	}

	return &TokenClaims{
		UID:   token.UID,
		Email: email,
		Name:  name,
		Role:  role,
	}, nil
}
