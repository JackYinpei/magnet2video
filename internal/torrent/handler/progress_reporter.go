// Package handler: progress reporter runs on the worker, periodically polling
// the local torrent client for active downloads and publishing progress events.
// Author: magnet2video
// Created: 2026-04-20
package handler

import (
	"context"
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
}

// UntrackTorrent removes a torrent from the watch list.
func (r *ProgressReporter) UntrackTorrent(infoHash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lastStatus, infoHash)
}

// Start runs the report loop until the context is cancelled.
func (r *ProgressReporter) Start(ctx context.Context) {
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
	_ = r.gateway.DownloadProgress(ctx, eventTypes.DownloadProgressPayload{
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

	r.mu.Lock()
	prev := r.lastStatus[pr.InfoHash]
	r.lastStatus[pr.InfoHash] = pr.Status
	r.mu.Unlock()

	if pr.Status == "completed" && prev != "completed" {
		_ = r.gateway.DownloadCompleted(ctx, pr.InfoHash)
	}
}
