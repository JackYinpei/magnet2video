// Package handler provides message handler for transcoding jobs
// Author: Done-0
// Created: 2026-01-26
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"magnet2video/configs"
	"magnet2video/internal/events/gateway"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/transcode/ffmpeg"
	"magnet2video/internal/transcode/types"
)

// progressThrottleInterval is the minimum gap between progress events per job.
// User asked for ~every 2 seconds.
const progressThrottleInterval = 2 * time.Second

// TranscodeHandler handles transcode job messages. It does NOT touch the
// database directly — all state updates are published as worker events via
// WorkerGateway, and the server-side event processor applies them to the DB.
type TranscodeHandler struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	gateway       gateway.WorkerGateway
	ffmpeg        *ffmpeg.FFmpeg
}

// NewTranscodeHandler creates a new transcode handler.
func NewTranscodeHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	workerGateway gateway.WorkerGateway,
) *TranscodeHandler {
	return &TranscodeHandler{
		config:        config,
		loggerManager: loggerManager,
		gateway:       workerGateway,
		ffmpeg: ffmpeg.New(
			config.TranscodeConfig.FFmpegPath,
			config.TranscodeConfig.FFprobePath,
		),
	}
}

// Handle processes a transcode job message
func (h *TranscodeHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var transcodeMsg types.TranscodeMessage
	if err := json.Unmarshal(msg.Value, &transcodeMsg); err != nil {
		h.loggerManager.Logger().Errorf("failed to unmarshal transcode message: %v", err)
		return err
	}

	h.loggerManager.Logger().Infof("Processing transcode job: jobID=%d, infoHash=%s, fileIndex=%d, operation=%s",
		transcodeMsg.JobID, transcodeMsg.InfoHash, transcodeMsg.FileIndex, transcodeMsg.Operation)

	startTime := time.Now()
	if err := h.gateway.TranscodeJobStarted(ctx, transcodeMsg.JobID, transcodeMsg.TorrentID, transcodeMsg.FileIndex); err != nil {
		h.loggerManager.Logger().Warnf("failed to publish job started event: %v", err)
	}

	if _, err := os.Stat(transcodeMsg.InputPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("input file not found: %s", transcodeMsg.InputPath)
		h.failJob(ctx, transcodeMsg, errMsg)
		return fmt.Errorf("input file not found: %s", transcodeMsg.InputPath)
	}

	operation := h.resolveOperation(ctx, transcodeMsg)

	subtitleResults := h.extractSubtitles(ctx, transcodeMsg)

	throttle := newProgressThrottle(progressThrottleInterval)
	progressCallback := func(progress float64) {
		if !throttle.allow() {
			return
		}
		_ = h.gateway.TranscodeJobProgress(ctx, transcodeMsg.JobID, transcodeMsg.TorrentID, transcodeMsg.FileIndex, int(progress))
	}

	var err error
	switch operation {
	case types.OperationRemux:
		err = h.ffmpeg.Remux(ctx, transcodeMsg.InputPath, transcodeMsg.OutputPath, progressCallback)
	case types.OperationTranscode:
		preset := transcodeMsg.Preset
		if preset == "" {
			preset = h.config.TranscodeConfig.DefaultPreset
		}
		crf := transcodeMsg.CRF
		if crf == 0 {
			crf = h.config.TranscodeConfig.DefaultCRF
		}
		err = h.ffmpeg.Transcode(ctx, transcodeMsg.InputPath, transcodeMsg.OutputPath, preset, crf, progressCallback)
	default:
		err = fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		h.failJob(ctx, transcodeMsg, err.Error())
		return err
	}

	outputInfo, _ := os.Stat(transcodeMsg.OutputPath)
	var outputSize int64
	if outputInfo != nil {
		outputSize = outputInfo.Size()
	}
	duration := time.Since(startTime).Milliseconds()

	if err := h.gateway.TranscodeJobCompleted(ctx, eventTypes.TranscodeJobCompletedPayload{
		JobID:      transcodeMsg.JobID,
		TorrentID:  transcodeMsg.TorrentID,
		InfoHash:   transcodeMsg.InfoHash,
		FileIndex:  transcodeMsg.FileIndex,
		OutputPath: transcodeMsg.OutputPath,
		OutputSize: outputSize,
		Duration:   duration,
		CreatorID:  transcodeMsg.CreatorID,
	}); err != nil {
		h.loggerManager.Logger().Errorf("failed to publish transcode completed event: %v", err)
	}

	for _, r := range subtitleResults {
		if err := h.gateway.SubtitleExtracted(ctx, eventTypes.SubtitleExtractedPayload{
			TorrentID:       transcodeMsg.TorrentID,
			InfoHash:        transcodeMsg.InfoHash,
			ParentFileIndex: transcodeMsg.FileIndex,
			CreatorID:       transcodeMsg.CreatorID,
			FilePath:        r.FilePath,
			FileSize:        r.FileSize,
			StreamIndex:     r.StreamIndex,
			Language:        r.Language,
			LanguageName:    r.LanguageName,
			Title:           r.Title,
			Format:          r.Format,
			OriginalCodec:   r.OriginalCodec,
		}); err != nil {
			h.loggerManager.Logger().Warnf("failed to publish subtitle extracted event: %v", err)
		}
	}

	h.loggerManager.Logger().Infof("Transcode job completed: jobID=%d, duration=%dms, outputSize=%d",
		transcodeMsg.JobID, duration, outputSize)
	return nil
}

// failJob publishes a failure event for the transcode job.
func (h *TranscodeHandler) failJob(ctx context.Context, msg types.TranscodeMessage, errMsg string) {
	h.loggerManager.Logger().Errorf("Transcode job failed: jobID=%d, error=%s", msg.JobID, errMsg)
	_ = h.gateway.TranscodeJobFailed(ctx, eventTypes.TranscodeJobFailedPayload{
		JobID:     msg.JobID,
		TorrentID: msg.TorrentID,
		FileIndex: msg.FileIndex,
		ErrorMsg:  errMsg,
	})
}

// resolveOperation picks the transcode operation. If the message specifies
// a concrete operation we trust it; otherwise (or when the worker wants to
// second-guess the server), we probe the file locally to decide remux vs
// transcode. This lets the server queue jobs without needing to read the
// source file — the worker handles the decision.
func (h *TranscodeHandler) resolveOperation(ctx context.Context, msg types.TranscodeMessage) string {
	if msg.Operation == types.OperationRemux || msg.Operation == types.OperationTranscode {
		return msg.Operation
	}
	probeCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	info, err := h.ffmpeg.Probe(probeCtx, msg.InputPath)
	if err != nil {
		h.loggerManager.Logger().Warnf("probe failed, default to transcode: %v", err)
		return types.OperationTranscode
	}
	if h.ffmpeg.DetermineTranscodeType(info, msg.InputPath) == ffmpeg.TranscodeTypeRemux {
		return types.OperationRemux
	}
	return types.OperationTranscode
}

// extractSubtitles extracts subtitle streams from the input video file
func (h *TranscodeHandler) extractSubtitles(ctx context.Context, msg types.TranscodeMessage) []ffmpeg.SubtitleExtractResult {
	outputDir := filepath.Dir(msg.OutputPath)
	baseName := strings.TrimSuffix(filepath.Base(msg.InputPath), filepath.Ext(msg.InputPath))

	results, err := h.ffmpeg.ExtractSubtitles(ctx, msg.InputPath, outputDir, baseName)
	if err != nil {
		h.loggerManager.Logger().Warnf("Failed to extract subtitles for jobID=%d: %v", msg.JobID, err)
		return nil
	}

	if len(results) > 0 {
		h.loggerManager.Logger().Infof("Extracted %d subtitle(s) for jobID=%d", len(results), msg.JobID)
	}

	return results
}

// progressThrottle limits the rate of progress callbacks.
type progressThrottle struct {
	mu       sync.Mutex
	last     time.Time
	interval time.Duration
}

func newProgressThrottle(interval time.Duration) *progressThrottle {
	return &progressThrottle{interval: interval}
}

// allow returns true at most once per interval.
func (t *progressThrottle) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if now.Sub(t.last) < t.interval {
		return false
	}
	t.last = now
	return true
}
