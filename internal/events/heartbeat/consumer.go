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

// FreshBootHook is invoked once per heartbeat that has FreshBoot=true.
// Servers register this to re-dispatch download/transcode work that was
// active when the worker restarted. The hook runs in its own goroutine
// so it does not block heartbeat record latency.
type FreshBootHook func(ctx context.Context, workerID string)

// Consumer is a queue.Handler that records incoming worker heartbeats.
type Consumer struct {
	store         *StatusStore
	loggerManager logger.LoggerManager
	freshBootHook FreshBootHook
}

// NewConsumer wires a heartbeat consumer.
func NewConsumer(store *StatusStore, loggerManager logger.LoggerManager) *Consumer {
	return &Consumer{store: store, loggerManager: loggerManager}
}

// SetFreshBootHook installs the recovery callback. Calling with nil disables
// the hook. Safe to set after construction; reads are not concurrent with
// Handle (the hook is set during bootstrap before the consumer subscribes).
func (c *Consumer) SetFreshBootHook(hook FreshBootHook) {
	c.freshBootHook = hook
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
		return nil
	}
	c.loggerManager.Logger().Infof(
		"Worker heartbeat received: workerID=%s jobs=%d diskFreeGB=%d freshBoot=%v version=%s",
		hb.WorkerID,
		len(hb.CurrentJobs),
		hb.DiskFreeGB,
		hb.FreshBoot,
		hb.Version,
	)
	// FreshBoot=true means this is the first heartbeat after the worker
	// (re)started — drive recovery off-thread so any DB / queue work the
	// hook does cannot stall heartbeat throughput.
	if hb.FreshBoot && c.freshBootHook != nil {
		hook := c.freshBootHook
		workerID := hb.WorkerID
		go func() {
			recoveryCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			hook(recoveryCtx, workerID)
		}()
	}
	return nil
}
