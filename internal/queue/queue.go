// Package queue provides message queue abstraction layer
// Author: Done-0
// Created: 2025-08-25
package queue

import (
	"context"
	"errors"
	"fmt"

	"magnet2video/configs"
)

// ErrNotForMe is the sentinel a Handler returns when a message was delivered
// to the wrong consumer (e.g. a download/file-op job whose TargetWorkerID
// belongs to a different worker on the shared queue). Consumers MUST treat
// this as "do not ack — put the message back on the queue so a peer can take
// it" rather than "permanent failure". Implementations add a small delay
// before requeue to avoid the same consumer immediately re-grabbing it.
var ErrNotForMe = errors.New("queue: message not for this consumer")

// Producer defines the interface for message production
type Producer interface {
	Send(ctx context.Context, topic string, key, value []byte) error
	Close() error
}

// Consumer defines the interface for message consumption
type Consumer interface {
	Subscribe(topics []string) error
	Close() error
}

// Handler defines the interface for message processing.
//
// Returning nil → message was processed (ack).
// Returning ErrNotForMe → message belongs to another consumer (requeue).
// Any other non-nil error → permanent failure for this delivery (ack to
// avoid infinite retry loops; the handler is expected to have already
// recorded the failure).
type Handler interface {
	Handle(ctx context.Context, msg *Message) error
}

// NewProducer creates a producer based on configuration
func NewProducer(config *configs.Config) (Producer, error) {
	switch config.QueueConfig.Type {
	case "channel", "":
		return NewChannelProducer(), nil
	case "rabbitmq":
		return NewRabbitMQProducer(config)
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", config.QueueConfig.Type)
	}
}

// NewConsumer creates a consumer based on configuration
func NewConsumer(config *configs.Config, handler Handler) (Consumer, error) {
	switch config.QueueConfig.Type {
	case "channel", "":
		return NewChannelConsumer(handler), nil
	case "rabbitmq":
		return NewRabbitMQConsumer(config, handler)
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", config.QueueConfig.Type)
	}
}
