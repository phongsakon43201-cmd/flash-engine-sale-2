package queue

import (
	"context"
	"log"

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
	select {
	case r.ch <- event:
		return nil
	default:
		log.Printf("[MemoryQueue Warning] Queue buffer full, order dropped: %s", event.OrderID)
		return nil
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
			if err := handler(event); err != nil {
				log.Printf("[MemoryQueue Error] Failed processing order %s: %v", event.OrderID, err)
			}
		}
	}
}
