// Package heartbeat handles worker heartbeat publishing (worker side) and
// status tracking (server side).
// Author: magnet2video
// Created: 2026-04-20
package heartbeat

import (
	"context"
	"encoding/json"
	"time"

	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	redisMgr "magnet2video/internal/redis"
)

// Redis key format for worker status. TTL is reset on every heartbeat.
const (
	statusKeyPrefix = "worker:status:"
	// StatusTTL is how long a heartbeat is considered fresh. Workers publish
	// every 10s; server marks worker offline after 30s of silence.
	StatusTTL = 30 * time.Second
	// registrySetKey tracks known worker IDs so the server can list them.
	registrySetKey = "worker:registry"
)

// StatusStore wraps Redis for worker-status bookkeeping.
type StatusStore struct {
	redis         redisMgr.RedisManager
	loggerManager logger.LoggerManager
}

// NewStatusStore builds a StatusStore.
func NewStatusStore(redis redisMgr.RedisManager, loggerManager logger.LoggerManager) *StatusStore {
	return &StatusStore{redis: redis, loggerManager: loggerManager}
}

// Record persists a heartbeat with the configured TTL.
func (s *StatusStore) Record(ctx context.Context, hb *eventTypes.Heartbeat) error {
	if s.redis == nil {
		return nil
	}
	client := s.redis.Client()
	if client == nil {
		return nil
	}
	data, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	if err := client.Set(ctx, statusKeyPrefix+hb.WorkerID, data, StatusTTL).Err(); err != nil {
		return err
	}
	// Add to registry (uses SADD so duplicates are fine).
	_ = client.SAdd(ctx, registrySetKey, hb.WorkerID).Err()
	return nil
}

// WorkerStatus describes one worker's live state for the API.
type WorkerStatus struct {
	WorkerID    string                    `json:"worker_id"`
	Online      bool                      `json:"online"`
	LastSeen    int64                     `json:"last_seen"`     // unix ms
	StaleFor    int64                     `json:"stale_for_sec"` // seconds since last heartbeat
	Version     string                    `json:"version"`
	DiskFreeGB  int64                     `json:"disk_free_gb"`
	DiskTotalGB int64                     `json:"disk_total_gb"`
	CurrentJobs []eventTypes.HeartbeatJob `json:"current_jobs"`
}

// DiskSummary aggregates disk space across all online workers. Used by the
// admin stats endpoint instead of statfs'ing the local server fs (which has
// no relation to where files actually live in split deployment).
type DiskSummary struct {
	TotalGB int64
	FreeGB  int64
	Workers int
}

// AggregateDisk returns the sum of disk_free / disk_total reported in the
// most recent heartbeat from each online worker.
func (s *StatusStore) AggregateDisk(ctx context.Context) DiskSummary {
	statuses, err := s.List(ctx)
	if err != nil {
		return DiskSummary{}
	}
	out := DiskSummary{}
	for _, st := range statuses {
		if !st.Online {
			continue
		}
		out.Workers++
		out.FreeGB += st.DiskFreeGB
		out.TotalGB += st.DiskTotalGB
	}
	return out
}

// List returns the status of every worker ever seen by this server.
func (s *StatusStore) List(ctx context.Context) ([]WorkerStatus, error) {
	if s.redis == nil {
		return []WorkerStatus{}, nil
	}
	client := s.redis.Client()
	if client == nil {
		return []WorkerStatus{}, nil
	}
	ids, err := client.SMembers(ctx, registrySetKey).Result()
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	result := make([]WorkerStatus, 0, len(ids))
	for _, id := range ids {
		raw, err := client.Get(ctx, statusKeyPrefix+id).Result()
		if err != nil {
			// Key expired — worker is offline. Emit a synthetic offline entry.
			result = append(result, WorkerStatus{WorkerID: id, Online: false})
			continue
		}
		var hb eventTypes.Heartbeat
		if err := json.Unmarshal([]byte(raw), &hb); err != nil {
			continue
		}
		result = append(result, WorkerStatus{
			WorkerID:    hb.WorkerID,
			Online:      true,
			LastSeen:    hb.Timestamp,
			StaleFor:    (now - hb.Timestamp) / 1000,
			Version:     hb.Version,
			DiskFreeGB:  hb.DiskFreeGB,
			DiskTotalGB: hb.DiskTotalGB,
			CurrentJobs: hb.CurrentJobs,
		})
	}
	return result, nil
}

// AnyOnline returns true if at least one worker is currently online.
func (s *StatusStore) AnyOnline(ctx context.Context) bool {
	statuses, err := s.List(ctx)
	if err != nil {
		return false
	}
	for _, st := range statuses {
		if st.Online {
			return true
		}
	}
	return false
}
