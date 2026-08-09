package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisabledRepositoryRejectsUploads(t *testing.T) {
	repository := NewDisabledRepository()
	uploadURL, fileURL, err := repository.GeneratePresignedUploadURL(context.Background(), "product.jpg", "image/jpeg")

	assert.Empty(t, uploadURL)
	assert.Empty(t, fileURL)
	assert.ErrorContains(t, err, "disabled")
}
