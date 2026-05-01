// Package replybus provides an in-memory request/reply correlation map for
// MQ-based RPC patterns (e.g. parse-magnet). The server registers a request
// id and waits on a channel; a dedicated consumer for the result topic
// delivers messages back to the registered channel.
//
// Author: magnet2video
// Created: 2026-05-01
package replybus

import (
	"context"
	"errors"
	"sync"

	torrentTypes "magnet2video/internal/torrent/types"
)

// ErrTimeout is returned when Wait expires before a reply lands.
var ErrTimeout = errors.New("reply bus: timeout waiting for response")

// ParseMagnetBus correlates parse-magnet-jobs requests with results.
//
// The bus is intentionally single-purpose. Generalizing it to "any reply"
// would buy us nothing right now and obscures the type-safety we get from
// returning a concrete struct.
type ParseMagnetBus struct {
	mu       sync.Mutex
	pending  map[string]chan *torrentTypes.ParseMagnetResult
}

// NewParseMagnetBus builds an empty bus.
func NewParseMagnetBus() *ParseMagnetBus {
	return &ParseMagnetBus{
		pending: make(map[string]chan *torrentTypes.ParseMagnetResult),
	}
}

// Register reserves a slot for the given request id and returns a channel
// that will receive the reply (or never fire if the request times out, in
// which case the caller should call Cancel).
//
// The returned channel is buffered (size 1) so Deliver never blocks even if
// the waiter has already given up.
func (b *ParseMagnetBus) Register(requestID string) <-chan *torrentTypes.ParseMagnetResult {
	ch := make(chan *torrentTypes.ParseMagnetResult, 1)
	b.mu.Lock()
	b.pending[requestID] = ch
	b.mu.Unlock()
	return ch
}

// Cancel removes a pending registration. Idempotent.
func (b *ParseMagnetBus) Cancel(requestID string) {
	b.mu.Lock()
	delete(b.pending, requestID)
	b.mu.Unlock()
}

// Deliver hands a result to whoever registered for that request id. If
// nobody is waiting (already cancelled / timed out) the result is dropped.
func (b *ParseMagnetBus) Deliver(result *torrentTypes.ParseMagnetResult) bool {
	if result == nil || result.RequestID == "" {
		return false
	}
	b.mu.Lock()
	ch, ok := b.pending[result.RequestID]
	if ok {
		delete(b.pending, result.RequestID)
	}
	b.mu.Unlock()
	if !ok {
		return false
	}
	// Non-blocking — channel has buffer 1 and can only receive one message.
	select {
	case ch <- result:
	default:
	}
	return true
}

// Wait blocks until either the result lands or the context expires.
// On context cancellation the request is removed from the pending map so
// late deliveries don't leak.
func (b *ParseMagnetBus) Wait(ctx context.Context, requestID string, ch <-chan *torrentTypes.ParseMagnetResult) (*torrentTypes.ParseMagnetResult, error) {
	select {
	case res := <-ch:
		return res, nil
	case <-ctx.Done():
		b.Cancel(requestID)
		return nil, ErrTimeout
	}
}
