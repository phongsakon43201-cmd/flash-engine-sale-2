package usecase

import (
	"context"
	"flashsale-go/internal/repository/firebase"
)

type AuthUsecase interface {
	VerifyToken(ctx context.Context, idToken string) (*firebase.TokenClaims, error)
}

type authUsecase struct {
	authClient *firebase.AuthClient
}

func NewAuthUsecase(authClient *firebase.AuthClient) AuthUsecase {
	return &authUsecase{
		authClient: authClient,
	}
}

func (u *authUsecase) VerifyToken(ctx context.Context, idToken string) (*firebase.TokenClaims, error) {
	return u.authClient.VerifyIDToken(ctx, idToken)
}
