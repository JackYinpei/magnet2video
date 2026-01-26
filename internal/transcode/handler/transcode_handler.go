// Package handler provides Kafka message handler for transcoding jobs
// Author: Done-0
// Created: 2026-01-26
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	"github.com/Done-0/gin-scaffold/internal/transcode/ffmpeg"
	"github.com/Done-0/gin-scaffold/internal/transcode/types"
)

// TranscodeHandler handles transcode job messages from Kafka
type TranscodeHandler struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	dbManager     db.DatabaseManager
	ffmpeg        *ffmpeg.FFmpeg
}

// NewTranscodeHandler creates a new transcode handler
func NewTranscodeHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
) *TranscodeHandler {
	return &TranscodeHandler{
		config:        config,
		loggerManager: loggerManager,
		dbManager:     dbManager,
		ffmpeg: ffmpeg.New(
			config.TranscodeConfig.FFmpegPath,
			config.TranscodeConfig.FFprobePath,
		),
	}
}

// Handle processes a transcode job message
func (h *TranscodeHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var transcodeMsg types.TranscodeMessage
	if err := json.Unmarshal(msg.Value, &transcodeMsg); err != nil {
		h.loggerManager.Logger().Errorf("failed to unmarshal transcode message: %v", err)
		return err
	}

	h.loggerManager.Logger().Infof("Processing transcode job: jobID=%d, infoHash=%s, fileIndex=%d, operation=%s",
		transcodeMsg.JobID, transcodeMsg.InfoHash, transcodeMsg.FileIndex, transcodeMsg.Operation)

	// Update job status to processing
	startTime := time.Now()
	if err := h.updateJobStatus(transcodeMsg.JobID, transcodeModel.JobStatusProcessing, 0, ""); err != nil {
		h.loggerManager.Logger().Errorf("failed to update job status: %v", err)
		return err
	}

	// Check if input file exists
	if _, err := os.Stat(transcodeMsg.InputPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("input file not found: %s", transcodeMsg.InputPath)
		h.handleJobFailure(transcodeMsg, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Progress callback to update database
	progressCallback := func(progress float64) {
		h.updateJobProgress(transcodeMsg.JobID, int(progress))
		h.updateTorrentProgress(transcodeMsg.TorrentID, transcodeMsg.FileIndex, int(progress))
	}

	// Execute transcoding based on operation type
	var err error
	switch transcodeMsg.Operation {
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
		err = fmt.Errorf("unknown operation: %s", transcodeMsg.Operation)
	}

	if err != nil {
		h.handleJobFailure(transcodeMsg, err.Error())
		return err
	}

	// Get output file info
	outputInfo, _ := os.Stat(transcodeMsg.OutputPath)
	var outputSize int64
	if outputInfo != nil {
		outputSize = outputInfo.Size()
	}

	duration := time.Since(startTime).Milliseconds()

	// Update job as completed
	if err := h.updateJobCompleted(transcodeMsg.JobID, transcodeMsg.OutputPath, outputSize, duration); err != nil {
		h.loggerManager.Logger().Errorf("failed to update job completion: %v", err)
	}

	// Update torrent file with transcoded path
	if err := h.updateTorrentFileTranscoded(transcodeMsg.TorrentID, transcodeMsg.FileIndex, transcodeMsg.OutputPath); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent file: %v", err)
	}

	// Update overall torrent transcode status
	if err := h.checkAndUpdateTorrentTranscodeStatus(transcodeMsg.TorrentID); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent transcode status: %v", err)
	}

	h.loggerManager.Logger().Infof("Transcode job completed: jobID=%d, duration=%dms, outputSize=%d",
		transcodeMsg.JobID, duration, outputSize)

	return nil
}

// handleJobFailure handles job failure by updating status and logging
func (h *TranscodeHandler) handleJobFailure(msg types.TranscodeMessage, errMsg string) {
	h.loggerManager.Logger().Errorf("Transcode job failed: jobID=%d, error=%s", msg.JobID, errMsg)

	// Update job status to failed
	h.updateJobStatus(msg.JobID, transcodeModel.JobStatusFailed, 0, errMsg)

	// Update torrent file status to failed
	h.updateTorrentFileStatus(msg.TorrentID, msg.FileIndex, torrentModel.TranscodeStatusFailed, errMsg)
}

// updateJobStatus updates the transcode job status in database
func (h *TranscodeHandler) updateJobStatus(jobID int64, status int, progress int, errorMsg string) error {
	updates := map[string]interface{}{
		"status":   status,
		"progress": progress,
	}

	if status == transcodeModel.JobStatusProcessing {
		updates["started_at"] = time.Now().Unix()
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return h.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", jobID).
		Updates(updates).Error
}

// updateJobProgress updates the transcode job progress
func (h *TranscodeHandler) updateJobProgress(jobID int64, progress int) error {
	return h.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", jobID).
		Update("progress", progress).Error
}

// updateJobCompleted marks the job as completed
func (h *TranscodeHandler) updateJobCompleted(jobID int64, outputPath string, outputSize int64, duration int64) error {
	return h.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":       transcodeModel.JobStatusCompleted,
			"progress":     100,
			"output_path":  outputPath,
			"completed_at": time.Now().Unix(),
		}).Error
}

// updateTorrentProgress updates the torrent file transcode progress
func (h *TranscodeHandler) updateTorrentProgress(torrentID int64, fileIndex int, progress int) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	// Update file transcode status
	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].TranscodeStatus = torrentModel.TranscodeStatusProcessing
	}

	// Calculate overall progress
	var totalProgress int
	var count int
	for _, file := range torrentRecord.Files {
		if file.TranscodeStatus == torrentModel.TranscodeStatusProcessing ||
			file.TranscodeStatus == torrentModel.TranscodeStatusPending {
			count++
		}
		if file.TranscodeStatus == torrentModel.TranscodeStatusCompleted {
			totalProgress += 100
			count++
		}
	}

	if count > 0 {
		torrentRecord.TranscodeProgress = (totalProgress + progress) / count
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}

// updateTorrentFileTranscoded updates the torrent file with transcoded path
func (h *TranscodeHandler) updateTorrentFileTranscoded(torrentID int64, fileIndex int, transcodedPath string) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].TranscodeStatus = torrentModel.TranscodeStatusCompleted
		torrentRecord.Files[fileIndex].TranscodedPath = transcodedPath
		torrentRecord.Files[fileIndex].TranscodeError = ""
		torrentRecord.TranscodedCount++
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}

// updateTorrentFileStatus updates the torrent file transcode status
func (h *TranscodeHandler) updateTorrentFileStatus(torrentID int64, fileIndex int, status int, errorMsg string) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].TranscodeStatus = status
		if errorMsg != "" {
			torrentRecord.Files[fileIndex].TranscodeError = errorMsg
		}
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}

// checkAndUpdateTorrentTranscodeStatus checks and updates overall torrent transcode status
func (h *TranscodeHandler) checkAndUpdateTorrentTranscodeStatus(torrentID int64) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	var pending, processing, completed, failed int
	for _, file := range torrentRecord.Files {
		switch file.TranscodeStatus {
		case torrentModel.TranscodeStatusPending:
			pending++
		case torrentModel.TranscodeStatusProcessing:
			processing++
		case torrentModel.TranscodeStatusCompleted:
			completed++
		case torrentModel.TranscodeStatusFailed:
			failed++
		}
	}

	// Determine overall status
	if processing > 0 {
		torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusProcessing
	} else if pending > 0 {
		torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusPending
	} else if failed > 0 && completed == 0 {
		torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusFailed
	} else if completed > 0 || (completed == 0 && pending == 0 && processing == 0 && failed == 0) {
		torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusCompleted
		torrentRecord.TranscodeProgress = 100
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}
