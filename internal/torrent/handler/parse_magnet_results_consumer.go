// Package handler: server-side consumer that routes parse-magnet-results
// messages back into the in-memory ParseMagnetBus so the originating HTTP
// request can return a response.
//
// Author: magnet2video
// Created: 2026-05-01
package handler

import (
	"context"
	"encoding/json"

	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/torrent/replybus"
	torrentTypes "magnet2video/internal/torrent/types"
)

// ParseMagnetResultsConsumer consumes parse-magnet-results messages and
// hands them to the bus. It runs only on hosts that issue parse requests
// (mode=all, mode=server) — the worker doesn't need it.
type ParseMagnetResultsConsumer struct {
	loggerManager logger.LoggerManager
	bus           *replybus.ParseMagnetBus
}

// NewParseMagnetResultsConsumer builds a results consumer.
func NewParseMagnetResultsConsumer(loggerManager logger.LoggerManager, bus *replybus.ParseMagnetBus) *ParseMagnetResultsConsumer {
	return &ParseMagnetResultsConsumer{
		loggerManager: loggerManager,
		bus:           bus,
	}
}

// Handle routes one result.
func (c *ParseMagnetResultsConsumer) Handle(_ context.Context, msg *queue.Message) error {
	var result torrentTypes.ParseMagnetResult
	if err := json.Unmarshal(msg.Value, &result); err != nil {
		c.loggerManager.Logger().Errorf("unmarshal parse-magnet result: %v", err)
		return nil
	}
	if !c.bus.Deliver(&result) {
		c.loggerManager.Logger().Warnf("parse-magnet result %s had no waiter (timed out?)", result.RequestID)
	}
	return nil
}
