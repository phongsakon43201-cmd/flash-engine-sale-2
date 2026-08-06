#!/bin/bash
echo "Initializing LocalStack resources..."

# Create AWS S3 Bucket
awslocal s3 mb s3://flashsale-product-images

# Create AWS SQS Queue
awslocal sqs create-queue --queue-name flashsale-order-queue

echo "LocalStack resources initialized successfully!"
