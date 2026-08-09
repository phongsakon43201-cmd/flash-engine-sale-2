package aws

import (
	"context"
	"fmt"
	"time"

	"flashsale-go/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Repository struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewS3Repository(region, accessKey, secretKey, customEndpoint, bucketName string) (domain.StorageRepository, error) {
	cfg, err := loadAWSConfig(context.Background(), region, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if customEndpoint != "" {
			o.BaseEndpoint = aws.String(customEndpoint)
			o.UsePathStyle = true
		}
	})

	presignClient := s3.NewPresignClient(client)

	return &s3Repository{
		client:        client,
		presignClient: presignClient,
		bucketName:    bucketName,
	}, nil
}

func (r *s3Repository) GeneratePresignedUploadURL(ctx context.Context, filename string, contentType string) (string, string, error) {
	objectKey := fmt.Sprintf("products/%d-%s", time.Now().UnixNano(), filename)

	req, err := r.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return "", "", fmt.Errorf("failed to generate S3 presigned URL: %w", err)
	}

	fileURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", r.bucketName, objectKey)
	return req.URL, fileURL, nil
}
