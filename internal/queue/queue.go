package queue

import (
	"context"
	"errors"
)

var ErrQueueClosed = errors.New("queue closed")

type Event struct {
	Type   string `json:"type"`
	CaseID int64  `json:"case_id"`
}

type Queue interface {
	Publish(ctx context.Context, event Event) error
	Consume(ctx context.Context) (Event, error)
}

type MemoryQueue struct {
	ch chan Event
}

func NewMemoryQueue(size int) *MemoryQueue {
	if size <= 0 {
		size = 128
	}
	return &MemoryQueue{ch: make(chan Event, size)}
}

func (q *MemoryQueue) Publish(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- event:
		return nil
	}
}

func (q *MemoryQueue) Consume(ctx context.Context) (Event, error) {
	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()
	case event, ok := <-q.ch:
		if !ok {
			return Event{}, ErrQueueClosed
		}
		return event, nil
	}
}
