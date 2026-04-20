// Package gateway provides the WorkerGateway abstraction used by workers to
// report status and heartbeat to the server via message queue.
// Author: magnet2video
// Created: 2026-04-20
package gateway

import (
	"context"

	eventTypes "magnet2video/internal/events/types"
)

// WorkerGateway publishes worker lifecycle events and heartbeats.
// Worker handlers depend on this interface instead of touching the DB directly,
// which keeps workers DB-free and lets the server own all state mutations.
type WorkerGateway interface {
	// Transcode lifecycle
	TranscodeJobStarted(ctx context.Context, jobID, torrentID int64, fileIndex int) error
	TranscodeJobProgress(ctx context.Context, jobID, torrentID int64, fileIndex, progress int) error
	TranscodeJobCompleted(ctx context.Context, payload eventTypes.TranscodeJobCompletedPayload) error
	TranscodeJobFailed(ctx context.Context, payload eventTypes.TranscodeJobFailedPayload) error
	SubtitleExtracted(ctx context.Context, payload eventTypes.SubtitleExtractedPayload) error

	// Cloud upload lifecycle
	CloudUploadStarted(ctx context.Context, torrentID int64, fileIndex int) error
	CloudUploadCompleted(ctx context.Context, torrentID int64, fileIndex int, cloudPath string) error
	CloudUploadFailed(ctx context.Context, torrentID int64, fileIndex int, errMsg string) error

	// Download lifecycle
	DownloadProgress(ctx context.Context, payload eventTypes.DownloadProgressPayload) error
	DownloadCompleted(ctx context.Context, infoHash string) error
	DownloadFailed(ctx context.Context, infoHash, errMsg string) error

	// Poster candidates (images found during download/transcode)
	PosterCandidateUploaded(ctx context.Context, torrentID int64, fileIndex int, filePath, cloudPath string) error

	// Heartbeat (liveness + current job summary)
	PublishHeartbeat(ctx context.Context, hb eventTypes.Heartbeat) error

	// WorkerID returns the identifier of this worker (for logging/diagnostics)
	WorkerID() string

	Close() error
}
