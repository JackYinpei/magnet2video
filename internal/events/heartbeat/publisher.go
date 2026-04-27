// Package heartbeat: worker-side periodic heartbeat loop.
// Author: magnet2video
// Created: 2026-04-20
package heartbeat

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"magnet2video/internal/events/gateway"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
)

// PublishInterval is how often the worker announces itself. Status store TTL
// must be at least 2x this to tolerate one missed beat.
const PublishInterval = 10 * time.Second

// Publisher runs a background loop publishing heartbeats via the gateway.
// Workers register in-flight jobs so the UI can show what's being processed.
type Publisher struct {
	gateway       gateway.WorkerGateway
	loggerManager logger.LoggerManager
	downloadDir   string
	version       string

	mu   sync.RWMutex
	jobs map[string]eventTypes.HeartbeatJob // keyed by "jobType:infoHash:fileName"

	stop atomic.Bool
	done chan struct{}
}

// NewPublisher constructs a heartbeat publisher.
// downloadDir is used to report free disk space to the server.
func NewPublisher(gw gateway.WorkerGateway, loggerManager logger.LoggerManager, downloadDir, version string) *Publisher {
	return &Publisher{
		gateway:       gw,
		loggerManager: loggerManager,
		downloadDir:   downloadDir,
		version:       version,
		jobs:          make(map[string]eventTypes.HeartbeatJob),
		done:          make(chan struct{}),
	}
}

// RegisterJob records an in-flight job (shown in heartbeat payload).
func (p *Publisher) RegisterJob(jobType, infoHash, fileName string) string {
	key := jobType + ":" + infoHash + ":" + fileName
	p.mu.Lock()
	p.jobs[key] = eventTypes.HeartbeatJob{
		JobType:  jobType,
		InfoHash: infoHash,
		FileName: fileName,
	}
	p.mu.Unlock()
	return key
}

// UpdateJobProgress updates the progress for a previously-registered job.
func (p *Publisher) UpdateJobProgress(key string, progress int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if j, ok := p.jobs[key]; ok {
		j.Progress = progress
		p.jobs[key] = j
	}
}

// UnregisterJob removes a finished job from the heartbeat payload.
func (p *Publisher) UnregisterJob(key string) {
	p.mu.Lock()
	delete(p.jobs, key)
	p.mu.Unlock()
}

// Start runs the heartbeat loop until the context is cancelled.
func (p *Publisher) Start(ctx context.Context) {
	// Emit an immediate heartbeat so status becomes visible without waiting for the first tick.
	p.publishOnce(ctx)

	ticker := time.NewTicker(PublishInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(p.done)
			return
		case <-ticker.C:
			p.publishOnce(ctx)
		}
	}
}

func (p *Publisher) publishOnce(ctx context.Context) {
	hb := eventTypes.Heartbeat{
		WorkerID:    p.gateway.WorkerID(),
		Timestamp:   time.Now().UnixMilli(),
		CurrentJobs: p.snapshotJobs(),
		DiskFreeGB:  p.diskFreeGB(),
		Version:     p.version,
	}
	if err := p.gateway.PublishHeartbeat(ctx, hb); err != nil {
		if p.loggerManager != nil {
			p.loggerManager.Logger().Warnf("publish heartbeat failed: %v", err)
		} else {
			log.Printf("publish heartbeat failed: %v", err)
		}
		return
	}
	if p.loggerManager != nil {
		p.loggerManager.Logger().Infof(
			"Worker heartbeat published: workerID=%s jobs=%d diskFreeGB=%d version=%s",
			hb.WorkerID,
			len(hb.CurrentJobs),
			hb.DiskFreeGB,
			hb.Version,
		)
	}
}

func (p *Publisher) snapshotJobs() []eventTypes.HeartbeatJob {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]eventTypes.HeartbeatJob, 0, len(p.jobs))
	for _, j := range p.jobs {
		out = append(out, j)
	}
	return out
}

// diskFreeGB reports free space on the download directory filesystem.
func (p *Publisher) diskFreeGB() int64 {
	if p.downloadDir == "" {
		return 0
	}
	path := p.downloadDir
	if _, err := os.Stat(path); err != nil {
		return 0
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	freeBytes := int64(stat.Bavail) * int64(stat.Bsize)
	return freeBytes / (1024 * 1024 * 1024)
}
