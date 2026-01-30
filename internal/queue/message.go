// Package queue provides message queue abstraction layer
// Author: Done-0
// Created: 2026-01-29
package queue

import "time"

// Message represents a queue message (implementation-agnostic)
type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	Timestamp time.Time
}
