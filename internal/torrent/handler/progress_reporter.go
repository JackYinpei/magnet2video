// Package handler: progress reporter runs on the worker, periodically polling
// the local torrent client for active downloads and publishing progress events.
// Author: magnet2video
// Created: 2026-04-20
package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"magnet2video/internal/events/gateway"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/torrent"
	torrentInternal "magnet2video/internal/torrent/internal"
)

// ProgressReportInterval is how often the reporter publishes progress events.
// Matches the user's preference of ~every 2 seconds.
const ProgressReportInterval = 2 * time.Second

// ProgressLogInterval controls visible diagnostic logs while keeping progress
// events frequent enough for Redis/UI updates.
const ProgressLogInterval = 10 * time.Second

// ProgressReporter periodically polls the local torrent client for active
// torrents and emits `download.progress` / `download.completed` events.
type ProgressReporter struct {
	torrentManager torrent.TorrentManager
	gateway        gateway.WorkerGateway
	loggerManager  logger.LoggerManager

	// tracks the last reported status per infoHash so we emit "completed"
	// exactly once.
	mu         sync.Mutex
	lastStatus map[string]string
	lastLogAt  map[string]time.Time
}

// NewProgressReporter constructs a reporter.
func NewProgressReporter(
	torrentManager torrent.TorrentManager,
	workerGateway gateway.WorkerGateway,
	loggerManager logger.LoggerManager,
) *ProgressReporter {
	return &ProgressReporter{
		torrentManager: torrentManager,
		gateway:        workerGateway,
		loggerManager:  loggerManager,
		lastStatus:     make(map[string]string),
		lastLogAt:      make(map[string]time.Time),
	}
}

// TrackTorrent adds a torrent to the reporter's watch list. Called when the
// download handler accepts a new start command.
func (r *ProgressReporter) TrackTorrent(infoHash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lastStatus[infoHash]; !ok {
		r.lastStatus[infoHash] = "pending"
	}
	if r.loggerManager != nil {
		r.loggerManager.Logger().Infof("Download progress tracking started: infoHash=%s", infoHash)
	}
}

// UntrackTorrent removes a torrent from the watch list.
func (r *ProgressReporter) UntrackTorrent(infoHash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lastStatus, infoHash)
	delete(r.lastLogAt, infoHash)
	if r.loggerManager != nil {
		r.loggerManager.Logger().Infof("Download progress tracking stopped: infoHash=%s", infoHash)
	}
}

// Start runs the report loop until the context is cancelled.
func (r *ProgressReporter) Start(ctx context.Context) {
	if r.loggerManager != nil {
		r.loggerManager.Logger().Infof("Download progress reporter started: interval=%s, log_interval=%s", ProgressReportInterval, ProgressLogInterval)
	}
	ticker := time.NewTicker(ProgressReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reportOnce(ctx)
		}
	}
}

func (r *ProgressReporter) reportOnce(ctx context.Context) {
	client := r.torrentManager.Client()
	if client == nil {
		return
	}
	r.mu.Lock()
	hashes := make([]string, 0, len(r.lastStatus))
	for h := range r.lastStatus {
		hashes = append(hashes, h)
	}
	r.mu.Unlock()

	for _, infoHash := range hashes {
		progress, err := client.GetProgress(infoHash)
		if err != nil {
			// Torrent was removed.
			r.UntrackTorrent(infoHash)
			continue
		}
		r.publish(ctx, progress)
	}
}

func (r *ProgressReporter) publish(ctx context.Context, pr *torrentInternal.DownloadProgress) {
	err := r.gateway.DownloadProgress(ctx, eventTypes.DownloadProgressPayload{
		InfoHash:       pr.InfoHash,
		Name:           pr.Name,
		Progress:       pr.Progress,
		Status:         pr.Status,
		DownloadedSize: pr.DownloadedSize,
		TotalSize:      pr.TotalSize,
		Peers:          pr.Peers,
		Seeds:          pr.Seeds,
		DownloadSpeed:  pr.DownloadSpeed,
	})
	if r.shouldLog(pr.InfoHash) && r.loggerManager != nil {
		if err != nil {
			r.loggerManager.Logger().Warnf(
				"Download progress publish failed: infoHash=%s status=%s progress=%.2f%% speed=%s peers=%d seeds=%d err=%v",
				pr.InfoHash,
				pr.Status,
				pr.Progress,
				formatSpeed(pr.DownloadSpeed),
				pr.Peers,
				pr.Seeds,
				err,
			)
		} else {
			r.loggerManager.Logger().Infof(
				"Download progress published: infoHash=%s status=%s progress=%.2f%% speed=%s peers=%d seeds=%d downloaded=%s/%s",
				pr.InfoHash,
				pr.Status,
				pr.Progress,
				formatSpeed(pr.DownloadSpeed),
				pr.Peers,
				pr.Seeds,
				formatBytes(pr.DownloadedSize),
				formatBytes(pr.TotalSize),
			)
		}
	}

	r.mu.Lock()
	prev := r.lastStatus[pr.InfoHash]
	r.lastStatus[pr.InfoHash] = pr.Status
	r.mu.Unlock()

	// "seeding" implies the download finished and the client is now uploading;
	// manager.go skips the "completed" status entirely once t.Seeding() is true,
	// so we must treat both as the completion trigger.
	isComplete := pr.Status == "completed" || pr.Status == "seeding"
	wasComplete := prev == "completed" || prev == "seeding"
	if isComplete && !wasComplete {
		_ = r.gateway.DownloadCompleted(ctx, pr.InfoHash)
	}
}

func (r *ProgressReporter) shouldLog(infoHash string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	last, ok := r.lastLogAt[infoHash]
	if ok && now.Sub(last) < ProgressLogInterval {
		return false
	}
	r.lastLogAt[infoHash] = now
	return true
}

func formatSpeed(bytesPerSecond int64) string {
	return fmt.Sprintf("%s/s", formatBytes(bytesPerSecond))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
