// Package gateway provides MQ-backed WorkerGateway implementation
// Author: magnet2video
// Created: 2026-04-20
package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
)

// mqGateway publishes worker events through a message queue producer.
type mqGateway struct {
	producer      queue.Producer
	loggerManager logger.LoggerManager
	workerID      string
}

// NewMQGateway builds a WorkerGateway backed by the provided queue producer.
// The same implementation works in all deployment modes (all/server/worker)
// because the producer abstracts the GoChannel / RabbitMQ transport.
func NewMQGateway(producer queue.Producer, loggerManager logger.LoggerManager, workerID string) WorkerGateway {
	return &mqGateway{
		producer:      producer,
		loggerManager: loggerManager,
		workerID:      workerID,
	}
}

func (g *mqGateway) WorkerID() string { return g.workerID }

func (g *mqGateway) Close() error {
	// Producer lifecycle is managed by wire/cmd, don't close it here.
	return nil
}

func (g *mqGateway) publishEvent(ctx context.Context, eventType string, payload any) error {
	event, err := eventTypes.NewEvent(eventType, g.workerID, payload)
	if err != nil {
		return fmt.Errorf("build event: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := g.producer.Send(ctx, eventTypes.TopicWorkerEvents, nil, data); err != nil {
		if g.loggerManager != nil {
			g.loggerManager.Logger().Errorf("publish worker event %s failed: %v", eventType, err)
		}
		return err
	}
	return nil
}

// ---- Transcode ----

func (g *mqGateway) TranscodeJobStarted(ctx context.Context, jobID, torrentID int64, fileIndex int) error {
	return g.publishEvent(ctx, eventTypes.EventTypeTranscodeJobStarted, eventTypes.TranscodeJobStartedPayload{
		JobID:     jobID,
		TorrentID: torrentID,
		FileIndex: fileIndex,
	})
}

func (g *mqGateway) TranscodeJobProgress(ctx context.Context, jobID, torrentID int64, fileIndex, progress int) error {
	return g.publishEvent(ctx, eventTypes.EventTypeTranscodeJobProgress, eventTypes.TranscodeJobProgressPayload{
		JobID:     jobID,
		TorrentID: torrentID,
		FileIndex: fileIndex,
		Progress:  progress,
	})
}

func (g *mqGateway) TranscodeJobCompleted(ctx context.Context, payload eventTypes.TranscodeJobCompletedPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeTranscodeJobCompleted, payload)
}

func (g *mqGateway) TranscodeJobFailed(ctx context.Context, payload eventTypes.TranscodeJobFailedPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeTranscodeJobFailed, payload)
}

func (g *mqGateway) SubtitleExtracted(ctx context.Context, payload eventTypes.SubtitleExtractedPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeSubtitleExtracted, payload)
}

// ---- Cloud upload ----

func (g *mqGateway) CloudUploadStarted(ctx context.Context, torrentID int64, fileIndex int) error {
	return g.publishEvent(ctx, eventTypes.EventTypeCloudUploadStarted, eventTypes.CloudUploadStartedPayload{
		TorrentID: torrentID,
		FileIndex: fileIndex,
	})
}

func (g *mqGateway) CloudUploadCompleted(ctx context.Context, torrentID int64, fileIndex int, cloudPath string) error {
	return g.publishEvent(ctx, eventTypes.EventTypeCloudUploadCompleted, eventTypes.CloudUploadCompletedPayload{
		TorrentID: torrentID,
		FileIndex: fileIndex,
		CloudPath: cloudPath,
	})
}

func (g *mqGateway) CloudUploadFailed(ctx context.Context, torrentID int64, fileIndex int, errMsg string) error {
	return g.publishEvent(ctx, eventTypes.EventTypeCloudUploadFailed, eventTypes.CloudUploadFailedPayload{
		TorrentID: torrentID,
		FileIndex: fileIndex,
		ErrorMsg:  errMsg,
	})
}

// ---- Download ----

func (g *mqGateway) DownloadProgress(ctx context.Context, payload eventTypes.DownloadProgressPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeDownloadProgress, payload)
}

func (g *mqGateway) DownloadCompleted(ctx context.Context, infoHash string) error {
	return g.publishEvent(ctx, eventTypes.EventTypeDownloadCompleted, eventTypes.DownloadCompletedPayload{
		InfoHash: infoHash,
	})
}

func (g *mqGateway) DownloadFailed(ctx context.Context, infoHash, errMsg string) error {
	return g.publishEvent(ctx, eventTypes.EventTypeDownloadFailed, eventTypes.DownloadFailedPayload{
		InfoHash: infoHash,
		ErrorMsg: errMsg,
	})
}

// ---- Poster candidates ----

func (g *mqGateway) PosterCandidateUploaded(ctx context.Context, torrentID int64, fileIndex int, filePath, cloudPath string) error {
	return g.publishEvent(ctx, eventTypes.EventTypePosterCandidateUploaded, eventTypes.PosterCandidateUploadedPayload{
		TorrentID: torrentID,
		FileIndex: fileIndex,
		FilePath:  filePath,
		CloudPath: cloudPath,
	})
}

// ---- File ops ----

func (g *mqGateway) FileOpCompleted(ctx context.Context, payload eventTypes.FileOpResultPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeFileOpCompleted, payload)
}

func (g *mqGateway) FileOpFailed(ctx context.Context, payload eventTypes.FileOpResultPayload) error {
	return g.publishEvent(ctx, eventTypes.EventTypeFileOpFailed, payload)
}

// ---- Heartbeat ----

func (g *mqGateway) PublishHeartbeat(ctx context.Context, hb eventTypes.Heartbeat) error {
	data, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	return g.producer.Send(ctx, eventTypes.TopicWorkerHeartbeat, nil, data)
}
