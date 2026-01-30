// Package queue provides in-memory channel queue implementation
// Author: Done-0
// Created: 2026-01-29
package queue

import (
	"context"
	"log"
	"sync"
	"time"
)

const defaultBufferSize = 100

// channelBroker manages in-memory message channels
type channelBroker struct {
	mu       sync.RWMutex
	channels map[string]chan *Message
	bufSize  int
}

var (
	globalBroker *channelBroker
	brokerOnce   sync.Once
)

// getChannelBroker returns the singleton broker instance
func getChannelBroker() *channelBroker {
	brokerOnce.Do(func() {
		globalBroker = &channelBroker{
			channels: make(map[string]chan *Message),
			bufSize:  defaultBufferSize,
		}
	})
	return globalBroker
}

// getOrCreateChannel returns the channel for a topic, creating it if needed
func (b *channelBroker) getOrCreateChannel(topic string) chan *Message {
	b.mu.RLock()
	ch, exists := b.channels[topic]
	b.mu.RUnlock()

	if exists {
		return ch
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check after acquiring write lock
	if ch, exists = b.channels[topic]; exists {
		return ch
	}

	ch = make(chan *Message, b.bufSize)
	b.channels[topic] = ch
	return ch
}

// ChannelProducer sends messages to in-memory channels
type ChannelProducer struct {
	broker *channelBroker
}

// NewChannelProducer creates a new channel producer
func NewChannelProducer() *ChannelProducer {
	return &ChannelProducer{broker: getChannelBroker()}
}

// Send sends a message to the specified topic
func (p *ChannelProducer) Send(ctx context.Context, topic string, key, value []byte) error {
	ch := p.broker.getOrCreateChannel(topic)

	msg := &Message{
		Topic:     topic,
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}

	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the producer (no-op for channel implementation)
func (p *ChannelProducer) Close() error {
	return nil
}

// ChannelConsumer receives messages from in-memory channels
type ChannelConsumer struct {
	broker  *channelBroker
	handler Handler
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewChannelConsumer creates a new channel consumer
func NewChannelConsumer(handler Handler) *ChannelConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &ChannelConsumer{
		broker:  getChannelBroker(),
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Subscribe starts consuming messages from the specified topics
func (c *ChannelConsumer) Subscribe(topics []string) error {
	for _, topic := range topics {
		go c.consumeTopic(topic)
	}
	return nil
}

// consumeTopic continuously reads from a topic channel
func (c *ChannelConsumer) consumeTopic(topic string) {
	ch := c.broker.getOrCreateChannel(topic)

	for {
		select {
		case msg := <-ch:
			if err := c.handler.Handle(c.ctx, msg); err != nil {
				log.Printf("Channel consumer handle error for topic %s: %v", topic, err)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Close gracefully shuts down the consumer
func (c *ChannelConsumer) Close() error {
	c.cancel()
	return nil
}
