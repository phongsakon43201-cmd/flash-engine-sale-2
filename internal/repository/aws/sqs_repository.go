package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"flashsale-go/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsRepository struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSRepository(region, accessKey, secretKey, customEndpoint, queueURL string) (domain.QueueRepository, error) {
	cfg, err := loadAWSConfig(context.Background(), region, accessKey, secretKey)
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
				WaitTimeSeconds:     10,
				VisibilityTimeout:   60,
			})
			if err != nil {
				log.Printf("Error receiving messages from SQS: %v. Retrying in 2 seconds...", err)
				timer := time.NewTimer(2 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case <-timer.C:
				}
				continue
			}

			var wg sync.WaitGroup
			for i := range output.Messages {
				message := output.Messages[i]
				wg.Add(1)
				go func() {
					defer wg.Done()
					if message.Body == nil || message.ReceiptHandle == nil {
						log.Printf("Received malformed SQS message metadata")
						return
					}

					var payload domain.OrderEventPayload
					if err := json.Unmarshal([]byte(*message.Body), &payload); err != nil {
						log.Printf("Failed to unmarshal SQS message body: %v", err)
						return
					}
					if err := handler(&payload); err != nil {
						log.Printf("Failed processing SQS order event [ID: %s]: %v", payload.OrderID, err)
						return
					}

					if _, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
						QueueUrl:      aws.String(r.queueURL),
						ReceiptHandle: message.ReceiptHandle,
					}); err != nil {
						log.Printf("Failed deleting SQS message: %v", err)
					}
				}()
			}
			wg.Wait()
		}
	}
}
