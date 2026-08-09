package aws

import (
	"context"
	"errors"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
)

func loadAWSConfig(ctx context.Context, region, accessKey, secretKey string) (awssdk.Config, error) {
	options := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if accessKey != "" || secretKey != "" {
		if accessKey == "" || secretKey == "" {
			return awssdk.Config{}, errors.New("both AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
		}
		options = append(options, awsconfig.WithCredentialsProvider(
			awscredentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}
	return awsconfig.LoadDefaultConfig(ctx, options...)
}
