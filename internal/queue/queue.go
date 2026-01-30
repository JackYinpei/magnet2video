// Package queue provides message queue abstraction layer
// Author: Done-0
// Created: 2025-08-25
package queue

import (
	"context"
	"fmt"

	"github.com/Done-0/gin-scaffold/configs"
)

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

// Handler defines the interface for message processing
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
