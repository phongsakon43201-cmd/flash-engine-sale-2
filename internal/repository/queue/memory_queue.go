package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"flashsale-go/internal/domain"
)

type memoryQueueRepository struct {
	ch chan *domain.OrderEventPayload
}

// NewMemoryQueueRepository returns an in-memory queue implementation for local development
func NewMemoryQueueRepository(bufferSize int) domain.QueueRepository {
	if bufferSize <= 0 {
		bufferSize = 10000
	}
	return &memoryQueueRepository{
		ch: make(chan *domain.OrderEventPayload, bufferSize),
	}
}

func (r *memoryQueueRepository) PublishOrderEvent(ctx context.Context, event *domain.OrderEventPayload) error {
	if event == nil {
		return errors.New("order event is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.ch <- event:
		return nil
	default:
		log.Printf("[MemoryQueue Warning] Queue buffer full, order dropped: %s", event.OrderID)
		return fmt.Errorf("memory queue is full for order %s", event.OrderID)
	}
}

func (r *memoryQueueRepository) ReceiveOrderEvents(ctx context.Context, handler func(event *domain.OrderEventPayload) error) error {
	log.Println("[MemoryQueue] Started listening for order events in-memory...")
	for {
		select {
		case <-ctx.Done():
			log.Println("[MemoryQueue] Stopping worker listener...")
			return nil
		case event, ok := <-r.ch:
			if !ok {
				return nil
			}
			var err error
			for attempt := 1; attempt <= 3; attempt++ {
				err = handler(event)
				if err == nil {
					break
				}
				if attempt < 3 {
					time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				}
			}
			if err != nil {
				log.Printf("[MemoryQueue Error] Failed processing order %s after retries: %v", event.OrderID, err)
			}
		}
	}
}
