// Package heartbeat: server-side consumer that writes heartbeats to Redis.
// Author: magnet2video
// Created: 2026-04-20
package heartbeat

import (
	"context"
	"encoding/json"

	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
)

// Consumer is a queue.Handler that records incoming worker heartbeats.
type Consumer struct {
	store         *StatusStore
	loggerManager logger.LoggerManager
}

// NewConsumer wires a heartbeat consumer.
func NewConsumer(store *StatusStore, loggerManager logger.LoggerManager) *Consumer {
	return &Consumer{store: store, loggerManager: loggerManager}
}

// Handle implements queue.Handler.
func (c *Consumer) Handle(ctx context.Context, msg *queue.Message) error {
	var hb eventTypes.Heartbeat
	if err := json.Unmarshal(msg.Value, &hb); err != nil {
		c.loggerManager.Logger().Warnf("heartbeat unmarshal failed: %v", err)
		return nil
	}
	if err := c.store.Record(ctx, &hb); err != nil {
		c.loggerManager.Logger().Warnf("heartbeat record failed: %v", err)
	}
	return nil
}
