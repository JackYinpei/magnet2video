// Package queue provides RabbitMQ queue implementation
// Author: Done-0
// Created: 2026-01-29
package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Done-0/gin-scaffold/configs"
)

// RabbitMQProducer sends messages to RabbitMQ
type RabbitMQProducer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

// NewRabbitMQProducer creates a new RabbitMQ producer
func NewRabbitMQProducer(config *configs.Config) (*RabbitMQProducer, error) {
	conn, err := amqp.Dial(config.QueueConfig.RabbitMQ.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	exchange := config.QueueConfig.RabbitMQ.Exchange
	exchangeType := config.QueueConfig.RabbitMQ.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &RabbitMQProducer{
		conn:     conn,
		channel:  ch,
		exchange: exchange,
	}, nil
}

// Send sends a message to the specified topic (routing key)
func (p *RabbitMQProducer) Send(ctx context.Context, topic string, key, value []byte) error {
	// Ensure queue exists
	_, err := p.channel.QueueDeclare(
		topic, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = p.channel.QueueBind(
		topic,      // queue name
		topic,      // routing key
		p.exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		p.exchange, // exchange
		topic,      // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         value,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		})
}

// Close closes the producer connection
func (p *RabbitMQProducer) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// RabbitMQConsumer receives messages from RabbitMQ
type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	handler  Handler
	exchange string
	prefetch int
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer(config *configs.Config, handler Handler) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(config.QueueConfig.RabbitMQ.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	exchange := config.QueueConfig.RabbitMQ.Exchange
	exchangeType := config.QueueConfig.RabbitMQ.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	prefetch := config.QueueConfig.RabbitMQ.PrefetchCount
	if prefetch <= 0 {
		prefetch = 1
	}

	err = ch.Qos(prefetch, 0, false)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RabbitMQConsumer{
		conn:     conn,
		channel:  ch,
		handler:  handler,
		exchange: exchange,
		prefetch: prefetch,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Subscribe starts consuming messages from the specified topics (queues)
func (c *RabbitMQConsumer) Subscribe(topics []string) error {
	for _, topic := range topics {
		// Declare queue
		q, err := c.channel.QueueDeclare(
			topic, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", topic, err)
		}

		// Bind queue to exchange
		err = c.channel.QueueBind(
			q.Name,     // queue name
			topic,      // routing key
			c.exchange, // exchange
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", topic, err)
		}

		// Start consuming
		msgs, err := c.channel.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		if err != nil {
			return fmt.Errorf("failed to consume from queue %s: %w", topic, err)
		}

		go c.consumeMessages(topic, msgs)
	}

	return nil
}

// consumeMessages processes messages from a delivery channel
func (c *RabbitMQConsumer) consumeMessages(topic string, msgs <-chan amqp.Delivery) {
	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				return
			}

			msg := &Message{
				Topic:     topic,
				Value:     d.Body,
				Timestamp: d.Timestamp,
			}

			if err := c.handler.Handle(c.ctx, msg); err != nil {
				log.Printf("RabbitMQ consumer handle error for topic %s: %v", topic, err)
				d.Nack(false, true) // Requeue on error
			} else {
				d.Ack(false)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Close gracefully shuts down the consumer
func (c *RabbitMQConsumer) Close() error {
	c.cancel()
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
