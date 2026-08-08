#!/bin/bash
echo "Initializing LocalStack resources..."

# Create AWS S3 Bucket
awslocal s3 mb s3://flashsale-product-images

# Create AWS SQS Dead Letter Queue (DLQ)
awslocal sqs create-queue --queue-name flashsale-order-dlq

DLQ_ARN=$(awslocal sqs get-queue-attributes --queue-url http://localhost:4566/000000000000/flashsale-order-dlq --attribute-names QueueArn --query Attributes.QueueArn --output text)

# Create AWS SQS Main Queue with Redrive Policy pointing to DLQ
awslocal sqs create-queue --queue-name flashsale-order-queue \
  --attributes "{\"RedrivePolicy\": \"{\\\"deadLetterTargetArn\\\":\\\"$DLQ_ARN\\\",\\\"maxReceiveCount\\\":\\\"3\\\"}\"}"

echo "LocalStack S3 Bucket, SQS Main Queue, and SQS DLQ initialized successfully!"
