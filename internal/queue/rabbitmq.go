// Package queue provides RabbitMQ queue implementation
// Author: Done-0
// Created: 2026-01-29
package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"magnet2video/configs"
)

// notForMeRequeueDelay is how long a consumer sleeps before NACKing a message
// that wasn't targeted at it. Picks a value short enough that a redispatch
// converges quickly across a 2-3 worker cluster, long enough that we don't
// burn CPU bouncing the same message between consumers.
const notForMeRequeueDelay = 250 * time.Millisecond

// RabbitMQProducer sends messages to RabbitMQ
type RabbitMQProducer struct {
	mu           sync.Mutex
	conn         *amqp.Connection
	channel      *amqp.Channel
	url          string
	exchange     string
	exchangeType string
	closed       bool // set true by Close(); stops reconnect attempts
}

// NewRabbitMQProducer creates a new RabbitMQ producer
func NewRabbitMQProducer(config *configs.Config) (*RabbitMQProducer, error) {
	exchange := config.QueueConfig.RabbitMQ.Exchange
	exchangeType := config.QueueConfig.RabbitMQ.ExchangeType
	if exchangeType == "" {
		exchangeType = "direct"
	}

	p := &RabbitMQProducer{
		url:          config.QueueConfig.RabbitMQ.URL,
		exchange:     exchange,
		exchangeType: exchangeType,
	}
	if err := p.dial(); err != nil {
		return nil, err
	}
	return p, nil
}

// dial opens a new connection + channel and declares the exchange.
// Caller must hold p.mu (or be in the constructor where no concurrency exists).
func (p *RabbitMQProducer) dial() error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(p.exchange, p.exchangeType, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	p.conn = conn
	p.channel = ch
	return nil
}

// reconnect tears down the current channel/connection and re-dials.
// Caller must hold p.mu.
func (p *RabbitMQProducer) reconnect() error {
	if p.channel != nil {
		p.channel.Close()
		p.channel = nil
	}
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
	return p.dial()
}

// isConnAlive returns true if the underlying connection looks healthy.
// Caller must hold p.mu.
func (p *RabbitMQProducer) isConnAlive() bool {
	return p.conn != nil && !p.conn.IsClosed() && p.channel != nil
}

// publishOnce performs a single declare/bind/publish using the current channel.
// Caller must hold p.mu.
func (p *RabbitMQProducer) publishOnce(ctx context.Context, topic string, value []byte) error {
	if !p.isConnAlive() {
		return fmt.Errorf("rabbitmq channel not open")
	}

	if _, err := p.channel.QueueDeclare(topic, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := p.channel.QueueBind(topic, topic, p.exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		p.exchange, topic, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         value,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		})
}

// Send sends a message to the specified topic (routing key).
// On any failure suggesting the channel is dead, it reconnects once and retries.
func (p *RabbitMQProducer) Send(ctx context.Context, topic string, key, value []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("rabbitmq producer closed")
	}

	err := p.publishOnce(ctx, topic, value)
	if err == nil {
		return nil
	}

	// Any publish error invalidates the channel in amqp091 — retry once after a fresh dial.
	log.Printf("RabbitMQ publish to %s failed: %v — reconnecting and retrying once", topic, err)
	if reErr := p.reconnect(); reErr != nil {
		return fmt.Errorf("publish failed (%v); reconnect also failed: %w", err, reErr)
	}
	return p.publishOnce(ctx, topic, value)
}

// Close closes the producer connection
func (p *RabbitMQProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	if p.channel != nil {
		p.channel.Close()
		p.channel = nil
	}
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

// RabbitMQConsumer receives messages from RabbitMQ
type RabbitMQConsumer struct {
	mu           sync.Mutex
	conn         *amqp.Connection
	channel      *amqp.Channel
	handler      Handler
	config       *configs.Config
	exchange     string
	exchangeType string
	prefetch     int
	ctx          context.Context
	cancel       context.CancelFunc
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
		conn:         conn,
		channel:      ch,
		handler:      handler,
		config:       config,
		exchange:     exchange,
		exchangeType: exchangeType,
		prefetch:     prefetch,
		ctx:          ctx,
		cancel:       cancel,
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

// consumeMessages processes messages from a delivery channel with auto-reconnect
func (c *RabbitMQConsumer) consumeMessages(topic string, msgs <-chan amqp.Delivery) {
	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				log.Printf("RabbitMQ delivery channel closed for topic %s, attempting to reconnect...", topic)
				backoff := 5 * time.Second
				reconnected := false
				for !reconnected {
					select {
					case <-c.ctx.Done():
						log.Printf("RabbitMQ consumer for topic %s stopped during reconnect", topic)
						return
					default:
					}

					if err := c.reconnect(); err != nil {
						log.Printf("RabbitMQ reconnect failed for topic %s: %v, retrying in %v...", topic, err, backoff)
						select {
						case <-time.After(backoff):
						case <-c.ctx.Done():
							return
						}
						if backoff < 60*time.Second {
							backoff *= 2
						}
						continue
					}

					newMsgs, err := c.resubscribeTopic(topic)
					if err != nil {
						log.Printf("RabbitMQ resubscribe failed for topic %s: %v, retrying in %v...", topic, err, backoff)
						// Invalidate connection so next reconnect() actually re-dials
						c.invalidateConnection()
						select {
						case <-time.After(backoff):
						case <-c.ctx.Done():
							return
						}
						if backoff < 60*time.Second {
							backoff *= 2
						}
						continue
					}

					msgs = newMsgs
					reconnected = true
					log.Printf("RabbitMQ consumer reconnected successfully for topic %s", topic)
				}
				continue
			}

			msg := &Message{
				Topic:     topic,
				Value:     d.Body,
				Timestamp: d.Timestamp,
			}

			if err := c.handler.Handle(c.ctx, msg); err != nil {
				if errors.Is(err, ErrNotForMe) {
					// Message belongs to another consumer (e.g. targeted at a
					// specific worker by id). Requeue with a small delay so we
					// don't immediately re-grab it ourselves and hot-loop.
					// RabbitMQ has no native delayed requeue, so we sleep on
					// this consumer goroutine before NACKing — prefetch=1
					// ensures we're not blocking other in-flight work, and a
					// few-hundred-ms delay is fine for a routing miss.
					time.Sleep(notForMeRequeueDelay)
					_ = d.Nack(false, true)
				} else {
					log.Printf("RabbitMQ consumer handle error for topic %s: %v", topic, err)
					// Don't requeue on error - the handler already marked the job as failed
					// Requeuing would cause infinite retry loops for permanent failures
					d.Ack(false)
				}
			} else {
				d.Ack(false)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// reconnect re-establishes the RabbitMQ connection and channel
func (c *RabbitMQConsumer) reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if connection is already healthy (another goroutine may have reconnected)
	if c.conn != nil && !c.conn.IsClosed() && c.channel != nil {
		return nil
	}

	// Close existing resources
	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	conn, err := amqp.Dial(c.config.QueueConfig.RabbitMQ.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(c.exchange, c.exchangeType, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	c.conn = conn
	c.channel = ch
	return nil
}

// invalidateConnection forces the next reconnect() call to re-dial
func (c *RabbitMQConsumer) invalidateConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// resubscribeTopic re-declares queue, binds it, and starts consuming
func (c *RabbitMQConsumer) resubscribeTopic(topic string) (<-chan amqp.Delivery, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	q, err := c.channel.QueueDeclare(topic, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue %s: %w", topic, err)
	}

	if err := c.channel.QueueBind(q.Name, topic, c.exchange, false, nil); err != nil {
		return nil, fmt.Errorf("failed to bind queue %s: %w", topic, err)
	}

	msgs, err := c.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from queue %s: %w", topic, err)
	}

	return msgs, nil
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
