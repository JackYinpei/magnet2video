// Package handler provides worker-side message handler for download-jobs.
// Author: magnet2video
// Created: 2026-04-20
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"magnet2video/configs"
	"magnet2video/internal/events/gateway"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/torrent"
)

// DownloadJobHandler executes download control commands on the worker.
// It adapts `download-jobs` messages to the local torrent client and publishes
// progress/completion events through the gateway.
type DownloadJobHandler struct {
	config         *configs.Config
	loggerManager  logger.LoggerManager
	torrentManager torrent.TorrentManager
	gateway        gateway.WorkerGateway
}

// NewDownloadJobHandler builds a DownloadJobHandler.
func NewDownloadJobHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	torrentManager torrent.TorrentManager,
	workerGateway gateway.WorkerGateway,
) *DownloadJobHandler {
	return &DownloadJobHandler{
		config:         config,
		loggerManager:  loggerManager,
		torrentManager: torrentManager,
		gateway:        workerGateway,
	}
}

// Handle dispatches a download command message.
func (h *DownloadJobHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var job eventTypes.DownloadJob
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		h.loggerManager.Logger().Errorf("unmarshal download job: %v", err)
		return nil
	}

	client := h.torrentManager.Client()
	if client == nil {
		return fmt.Errorf("torrent client unavailable")
	}

	h.loggerManager.Logger().Infof("Download job: action=%s, infoHash=%s", job.Action, job.InfoHash)

	switch job.Action {
	case eventTypes.DownloadActionStart:
		if job.MagnetURI != "" {
			// Ensure metadata is loaded so StartDownload can select files.
			if _, err := client.ParseMagnet(ctx, job.MagnetURI, job.Trackers); err != nil {
				h.loggerManager.Logger().Errorf("parse magnet failed: %v", err)
				_ = h.gateway.DownloadFailed(ctx, job.InfoHash, err.Error())
				return nil
			}
		}
		if err := client.StartDownload(ctx, job.InfoHash, job.SelectedFiles, job.Trackers); err != nil {
			h.loggerManager.Logger().Errorf("start download failed: %v", err)
			_ = h.gateway.DownloadFailed(ctx, job.InfoHash, err.Error())
			return nil
		}
	case eventTypes.DownloadActionPause:
		if err := client.PauseDownload(job.InfoHash); err != nil {
			h.loggerManager.Logger().Errorf("pause download failed: %v", err)
		}
	case eventTypes.DownloadActionResume:
		if err := client.ResumeDownload(job.InfoHash, job.SelectedFiles); err != nil {
			h.loggerManager.Logger().Errorf("resume download failed: %v", err)
		}
	case eventTypes.DownloadActionRemove:
		if err := client.RemoveTorrent(job.InfoHash, job.DeleteFiles); err != nil {
			h.loggerManager.Logger().Errorf("remove torrent failed: %v", err)
		}
	default:
		h.loggerManager.Logger().Warnf("unknown download action: %s", job.Action)
	}
	return nil
}
