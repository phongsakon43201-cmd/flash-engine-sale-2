package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"flashsale-go/internal/domain"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type sqsRepository struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSRepository(region, accessKey, secretKey, customEndpoint, queueURL string) (domain.QueueRepository, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if customEndpoint != "" {
			o.BaseEndpoint = aws.String(customEndpoint)
		}
	})

	return &sqsRepository{
		client:   client,
		queueURL: queueURL,
	}, nil
}

func (r *sqsRepository) PublishOrderEvent(ctx context.Context, event *domain.OrderEventPayload) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	_, err = r.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(r.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("failed to publish SQS message: %w", err)
	}

	return nil
}

func (r *sqsRepository) ReceiveOrderEvents(ctx context.Context, handler func(event *domain.OrderEventPayload) error) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping SQS Consumer worker...")
			return nil
		default:
			output, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(r.queueURL),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     5, // Long polling
			})
			if err != nil {
				log.Printf("Error receiving messages from SQS: %v. Retrying in 2 seconds...", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for _, message := range output.Messages {
				var payload domain.OrderEventPayload
				if err := json.Unmarshal([]byte(*message.Body), &payload); err != nil {
					log.Printf("Failed to unmarshal SQS message body: %v", err)
					continue
				}

				if err := handler(&payload); err != nil {
					log.Printf("Failed processing SQS order event [ID: %s]: %v", payload.OrderID, err)
					continue
				}

				// Delete message after successful processing
				_, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(r.queueURL),
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Printf("Failed deleting message from SQS [ReceiptHandle: %s]: %v", *message.ReceiptHandle, err)
				}
			}
		}
	}
}
