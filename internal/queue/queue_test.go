// Package queue provides message queue tests
// Author: Done-0
// Created: 2026-01-31
package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := Message{
		Topic:     "test-topic",
		Key:       []byte("key"),
		Value:     []byte("value"),
		Timestamp: now,
	}

	if msg.Topic != "test-topic" {
		t.Errorf("Topic = %s, want test-topic", msg.Topic)
	}
	if string(msg.Key) != "key" {
		t.Errorf("Key = %s, want key", string(msg.Key))
	}
	if string(msg.Value) != "value" {
		t.Errorf("Value = %s, want value", string(msg.Value))
	}
	if msg.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
}

func TestChannelProducer_Send(t *testing.T) {
	producer := NewChannelProducer()
	defer producer.Close()

	ctx := context.Background()
	topic := "test-send-" + time.Now().Format("150405")

	err := producer.Send(ctx, topic, []byte("key"), []byte("value"))
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestChannelProducer_SendWithCancelledContext(t *testing.T) {
	producer := NewChannelProducer()
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	topic := "test-cancelled-" + time.Now().Format("150405")

	// Fill the buffer first
	bgCtx := context.Background()
	for i := 0; i < defaultBufferSize; i++ {
		producer.Send(bgCtx, topic, nil, []byte("fill"))
	}

	// Now send with cancelled context should fail
	err := producer.Send(ctx, topic, nil, []byte("should-fail"))
	if err == nil {
		t.Error("Send() with cancelled context should return error when buffer is full")
	}
}

func TestChannelProducer_Close(t *testing.T) {
	producer := NewChannelProducer()
	err := producer.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// MockHandler implements Handler interface for testing
type MockHandler struct {
	mu       sync.Mutex
	messages []*Message
	errors   []error
}

func (h *MockHandler) Handle(ctx context.Context, msg *Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msg)
	return nil
}

func (h *MockHandler) GetMessages() []*Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.messages
}

func TestChannelConsumer_Subscribe(t *testing.T) {
	handler := &MockHandler{}
	consumer := NewChannelConsumer(handler)
	defer consumer.Close()

	topic := "test-subscribe-" + time.Now().Format("150405")

	err := consumer.Subscribe([]string{topic})
	if err != nil {
		t.Errorf("Subscribe() error = %v", err)
	}
}

func TestChannelConsumer_ReceiveMessage(t *testing.T) {
	handler := &MockHandler{}
	consumer := NewChannelConsumer(handler)
	defer consumer.Close()

	topic := "test-receive-" + time.Now().Format("150405")

	err := consumer.Subscribe([]string{topic})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Give the consumer goroutine time to start
	time.Sleep(10 * time.Millisecond)

	producer := NewChannelProducer()
	defer producer.Close()

	err = producer.Send(context.Background(), topic, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Wait for message to be processed
	time.Sleep(50 * time.Millisecond)

	messages := handler.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
		return
	}

	if string(messages[0].Value) != "value1" {
		t.Errorf("Message value = %s, want value1", string(messages[0].Value))
	}
}

func TestChannelConsumer_MultipleTopics(t *testing.T) {
	handler := &MockHandler{}
	consumer := NewChannelConsumer(handler)
	defer consumer.Close()

	topics := []string{
		"test-multi-1-" + time.Now().Format("150405"),
		"test-multi-2-" + time.Now().Format("150405"),
	}

	err := consumer.Subscribe(topics)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	producer := NewChannelProducer()
	defer producer.Close()

	for _, topic := range topics {
		producer.Send(context.Background(), topic, nil, []byte(topic))
	}

	time.Sleep(50 * time.Millisecond)

	messages := handler.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestChannelConsumer_Close(t *testing.T) {
	handler := &MockHandler{}
	consumer := NewChannelConsumer(handler)

	topic := "test-close-" + time.Now().Format("150405")
	consumer.Subscribe([]string{topic})

	err := consumer.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestGetChannelBroker_Singleton(t *testing.T) {
	broker1 := getChannelBroker()
	broker2 := getChannelBroker()

	if broker1 != broker2 {
		t.Error("getChannelBroker() should return the same instance")
	}
}

func TestChannelBroker_GetOrCreateChannel(t *testing.T) {
	broker := getChannelBroker()

	topic := "test-getorcreate-" + time.Now().Format("150405")

	ch1 := broker.getOrCreateChannel(topic)
	ch2 := broker.getOrCreateChannel(topic)

	if ch1 != ch2 {
		t.Error("getOrCreateChannel() should return the same channel for the same topic")
	}
}

func TestChannelBroker_ConcurrentAccess(t *testing.T) {
	broker := getChannelBroker()
	topic := "test-concurrent-" + time.Now().Format("150405")

	var wg sync.WaitGroup
	channels := make([]chan *Message, 10)

	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			channels[idx] = broker.getOrCreateChannel(topic)
		}(i)
	}

	wg.Wait()

	// All channels should be the same
	for i := 1; i < 10; i++ {
		if channels[i] != channels[0] {
			t.Error("Concurrent getOrCreateChannel() returned different channels")
		}
	}
}

func BenchmarkChannelProducer_Send(b *testing.B) {
	producer := NewChannelProducer()
	defer producer.Close()

	ctx := context.Background()
	topic := "benchmark-send"

	// Create a consumer to drain the channel
	handler := &MockHandler{}
	consumer := NewChannelConsumer(handler)
	defer consumer.Close()
	consumer.Subscribe([]string{topic})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		producer.Send(ctx, topic, nil, []byte("benchmark"))
	}
}
