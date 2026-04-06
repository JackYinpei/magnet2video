// Package sse provides Server-Sent Events streaming utilities tests
// Author: Done-0
// Created: 2026-01-31
package sse

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	internalsse "magnet2video/internal/sse"
)

type mockManager struct {
	events []*internalsse.Event
}

func (m *mockManager) StreamToClient(c *gin.Context, events <-chan *internalsse.Event) error {
	for event := range events {
		m.events = append(m.events, event)
	}
	return nil
}

type payload struct {
	Message string `json:"message"`
}

func TestSend_SerializesData(t *testing.T) {
	ch := make(chan *internalsse.Event, 1)
	if err := Send(ch, "test", payload{Message: "hello"}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	event := <-ch
	if event.Event != "test" {
		t.Fatalf("Send() event type = %q, want %q", event.Event, "test")
	}
	if event.Data != "{\"message\":\"hello\"}" {
		t.Fatalf("Send() event data = %q, want JSON payload", event.Data)
	}
}

func TestSend_ErrorOnUnsupportedType(t *testing.T) {
	ch := make(chan *internalsse.Event, 1)
	if err := Send(ch, "test", make(chan int)); err == nil {
		t.Fatal("Send() expected error for unsupported type, got nil")
	}

	select {
	case <-ch:
		t.Fatal("Send() should not emit event on error")
	default:
	}
}

func TestStream_InvokesHandlerAndStreams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	manager := &mockManager{}
	messageHandler := func(ctx context.Context, ch chan<- *internalsse.Event) {
		_ = Send(ch, "first", payload{Message: "one"})
		_ = Send(ch, "second", payload{Message: "two"})
	}

	if err := Stream(c, messageHandler, manager); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if len(manager.events) != 2 {
		t.Fatalf("Stream() events = %d, want 2", len(manager.events))
	}
	if manager.events[0].Event != "first" || manager.events[1].Event != "second" {
		t.Fatalf("Stream() event order = %v", []string{manager.events[0].Event, manager.events[1].Event})
	}
}
