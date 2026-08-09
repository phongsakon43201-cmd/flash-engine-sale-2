package storage

import (
	"context"
	"errors"

	"flashsale-go/internal/domain"
)

var errDisabled = errors.New("object storage is disabled for this deployment")

type disabledRepository struct{}

func NewDisabledRepository() domain.StorageRepository {
	return &disabledRepository{}
}

func (r *disabledRepository) GeneratePresignedUploadURL(context.Context, string, string) (string, string, error) {
	return "", "", errDisabled
}
