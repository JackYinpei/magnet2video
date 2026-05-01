// Package handler: worker-side handler for parse-magnet-jobs.
// Author: magnet2video
// Created: 2026-05-01
package handler

import (
	"context"
	"encoding/json"
	"time"

	"magnet2video/internal/events/gateway"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/torrent"
	torrentTypes "magnet2video/internal/torrent/types"
)

// parseMagnetTimeout caps how long the worker spends fetching torrent
// metadata before giving up. The server-side waiter has its own deadline
// (~120s) so this needs to fit inside that.
const parseMagnetTimeout = 90 * time.Second

// ParseMagnetHandler runs on the worker. It listens on parse-magnet-jobs,
// resolves the magnet URI through the local torrent client, and publishes
// the result back on parse-magnet-results.
type ParseMagnetHandler struct {
	loggerManager  logger.LoggerManager
	torrentManager torrent.TorrentManager
	gateway        gateway.WorkerGateway
	queueProducer  queue.Producer
}

// NewParseMagnetHandler builds a ParseMagnetHandler.
func NewParseMagnetHandler(
	loggerManager logger.LoggerManager,
	torrentManager torrent.TorrentManager,
	workerGateway gateway.WorkerGateway,
	queueProducer queue.Producer,
) *ParseMagnetHandler {
	return &ParseMagnetHandler{
		loggerManager:  loggerManager,
		torrentManager: torrentManager,
		gateway:        workerGateway,
		queueProducer:  queueProducer,
	}
}

// Handle resolves one parse-magnet request.
func (h *ParseMagnetHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var req torrentTypes.ParseMagnetRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		h.loggerManager.Logger().Errorf("unmarshal parse-magnet request: %v", err)
		return nil
	}
	if req.RequestID == "" {
		h.loggerManager.Logger().Warn("parse-magnet request missing request_id, dropping")
		return nil
	}

	result := torrentTypes.ParseMagnetResult{
		RequestID: req.RequestID,
		WorkerID:  h.gateway.WorkerID(),
	}

	parseCtx, cancel := context.WithTimeout(ctx, parseMagnetTimeout)
	defer cancel()

	client := h.torrentManager.Client()
	if client == nil {
		result.ErrorMsg = "worker torrent client unavailable"
		h.publishResult(ctx, result)
		return nil
	}

	info, err := client.ParseMagnet(parseCtx, req.MagnetURI, req.Trackers)
	if err != nil {
		h.loggerManager.Logger().Errorf("parse magnet %s failed: %v", req.RequestID, err)
		result.ErrorMsg = err.Error()
		h.publishResult(ctx, result)
		return nil
	}

	result.InfoHash = info.InfoHash
	result.Name = info.Name
	result.TotalSize = info.TotalSize
	result.Files = make([]torrentTypes.ParseMagnetFile, len(info.Files))
	for i, f := range info.Files {
		result.Files[i] = torrentTypes.ParseMagnetFile{
			Path:         f.Path,
			Size:         f.Size,
			IsStreamable: f.IsStreamable,
		}
	}
	h.publishResult(ctx, result)
	return nil
}

func (h *ParseMagnetHandler) publishResult(ctx context.Context, result torrentTypes.ParseMagnetResult) {
	data, err := json.Marshal(result)
	if err != nil {
		h.loggerManager.Logger().Errorf("marshal parse-magnet result %s: %v", result.RequestID, err)
		return
	}
	if err := h.queueProducer.Send(ctx, torrentTypes.TopicParseMagnetResults, nil, data); err != nil {
		h.loggerManager.Logger().Errorf("publish parse-magnet result %s: %v", result.RequestID, err)
	}
}
