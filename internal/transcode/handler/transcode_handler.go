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
	"time"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/configs"
	cloudTypes "github.com/Done-0/gin-scaffold/internal/cloud/types"
	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	"github.com/Done-0/gin-scaffold/internal/queue"
	"github.com/Done-0/gin-scaffold/internal/transcode/ffmpeg"
	"github.com/Done-0/gin-scaffold/internal/transcode/types"
)

// TranscodeHandler handles transcode job messages from Kafka
type TranscodeHandler struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	dbManager     db.DatabaseManager
	ffmpeg        *ffmpeg.FFmpeg
	queueProducer queue.Producer
}

// NewTranscodeHandler creates a new transcode handler
func NewTranscodeHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	queueProducer queue.Producer,
) *TranscodeHandler {
	return &TranscodeHandler{
		config:        config,
		loggerManager: loggerManager,
		dbManager:     dbManager,
		queueProducer: queueProducer,
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
		return fmt.Errorf("input file not found: %s", transcodeMsg.InputPath)
	}

	// Extract subtitles before transcoding
	subtitleResults := h.extractSubtitles(ctx, transcodeMsg)

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

	// Update torrent file with transcoded path and append transcoded file entry
	transcodedIndex, err := h.updateTorrentFileTranscoded(transcodeMsg.TorrentID, transcodeMsg.FileIndex, transcodeMsg.OutputPath, outputSize)
	if err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent file: %v", err)
	}

	// Save subtitle info to database and queue cloud uploads
	if len(subtitleResults) > 0 {
		h.saveSubtitleInfo(transcodeMsg, subtitleResults)
	}

	// Update overall torrent transcode status
	if err := h.checkAndUpdateTorrentTranscodeStatus(transcodeMsg.TorrentID); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent transcode status: %v", err)
	}

	h.loggerManager.Logger().Infof("Transcode job completed: jobID=%d, duration=%dms, outputSize=%d",
		transcodeMsg.JobID, duration, outputSize)

	// Trigger cloud upload if enabled
	if h.config.CloudStorageConfig.Enabled && outputInfo != nil && transcodedIndex >= 0 {
		h.queueCloudUpload(ctx, transcodeMsg.TorrentID, transcodeMsg.InfoHash, transcodedIndex,
			transcodeMsg.OutputPath, outputInfo.Size(), true, transcodeMsg.CreatorID)
	}

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
	// Update file transcode status directly
	err := h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Update("transcode_status", torrentModel.TranscodeStatusProcessing).Error
	if err != nil {
		return err
	}

	// Calculate overall progress using aggregation
	type Result struct {
		Status int
		Count  int
	}
	var results []Result
	if err := h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("transcode_status as status, count(*) as count").
		Where("torrent_id = ? AND (source = '' OR source = 'original')", torrentID).
		Group("transcode_status").
		Scan(&results).Error; err != nil {
		return err
	}

	var totalProgress int // Estimate: completed=100, processing=progress, pending=0
	var count int

	for _, r := range results {
		c := r.Count
		s := r.Status
		if s == torrentModel.TranscodeStatusProcessing || s == torrentModel.TranscodeStatusPending {
			count += c
			// For processing, we use the passed progress for the CURRENT file, 
			// but we don't know progress of other processing files. 
			// Original code: "if processing... count++". 
			// It only added 'progress' (the arg) ONCE to totalProgress, and 100*completed.
			// It averaged (totalProgress + currentFileProgress) / processingCount.
			// This logic seems slightly flawed if multiple files are processing (concurrency).
			// But assuming single worker per file or simple approximation:
			
			// Actually, the original logic was:
			// totalProgress += 100 * completedCount
			// count = processing + pending + completed
			// avg = (totalProgress + progress) / count.
			// This assumes 'progress' is the SUM of progress of all processing files? No, it's just 'progress' argument.
			// Meaning it only accounts for the CURRENT file's progress. 
			// This suggests the progress bar only tracks the *currently reporting* file plus completed ones.
			// Let's stick to simple approximation or just replicate original logic.
		}
		if s == torrentModel.TranscodeStatusCompleted {
			totalProgress += 100 * c
			count += c
		}
	}

	var overallProgress int
	if count > 0 {
		overallProgress = (totalProgress + progress) / count
	}

	return h.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		Update("transcode_progress", overallProgress).Error
}

// updateTorrentFileTranscoded updates the original torrent file and appends a transcoded file entry.
// Returns the index of the new transcoded file entry.
func (h *TranscodeHandler) updateTorrentFileTranscoded(torrentID int64, fileIndex int, transcodedPath string, outputSize int64) (int, error) {
	// 1. Get source file
	var sourceFile torrentModel.TorrentFile
	if err := h.dbManager.DB().Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).First(&sourceFile).Error; err != nil {
		return -1, err
	}

	// 2. Update source file
	if err := h.dbManager.DB().Model(&sourceFile).Updates(map[string]interface{}{
		"transcode_status": torrentModel.TranscodeStatusCompleted,
		"transcoded_path":  transcodedPath,
		"transcode_error":  "",
	}).Error; err != nil {
		return -1, err
	}

	// 3. Determine new index
	var count int64
	h.dbManager.DB().Model(&torrentModel.TorrentFile{}).Where("torrent_id = ?", torrentID).Count(&count)
	newIndex := int(count)

	// 4. Create new file
	newFile := torrentModel.TorrentFile{
		TorrentID:    torrentID,
		Index:        newIndex,
		Path:         transcodedPath,
		Size:         outputSize,
		IsSelected:   true,
		IsShareable:  false,
		IsStreamable: true,
		Type:         "video",
		Source:       "transcoded",
		ParentPath:   sourceFile.Path,
	}

	if err := h.dbManager.DB().Create(&newFile).Error; err != nil {
		return -1, err
	}

	return newIndex, nil
}

// updateTorrentFileStatus updates the torrent file transcode status
func (h *TranscodeHandler) updateTorrentFileStatus(torrentID int64, fileIndex int, status int, errorMsg string) error {
	updates := map[string]interface{}{
		"transcode_status": status,
	}
	if errorMsg != "" {
		updates["transcode_error"] = errorMsg
	}

	return h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(updates).Error
}

// checkAndUpdateTorrentTranscodeStatus checks and updates overall torrent transcode status
func (h *TranscodeHandler) checkAndUpdateTorrentTranscodeStatus(torrentID int64) error {
	type Result struct {
		Status int
		Count  int
	}
	var results []Result
	// Count files by status (excluding derived files for status checking? Original code filtered source != "original" for counting)
	if err := h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("transcode_status as status, count(*) as count").
		Where("torrent_id = ? AND (source = '' OR source = 'original')", torrentID).
		Group("transcode_status").
		Scan(&results).Error; err != nil {
		return err
	}

	var pending, processing, completed, failed int
	for _, r := range results {
		s := r.Status
		c := r.Count
		switch s {
		case torrentModel.TranscodeStatusPending:
			pending += c
		case torrentModel.TranscodeStatusProcessing:
			processing += c
		case torrentModel.TranscodeStatusCompleted:
			completed += c
		case torrentModel.TranscodeStatusFailed:
			failed += c
		}
	}

	updates := map[string]interface{}{
		"transcode_status": torrentModel.TranscodeStatusNone,
		"transcode_progress": 0,
	}
	
	if completed > 0 {
		updates["transcoded_count"] = completed
	}
	
	// Determine overall status
	if processing > 0 {
		updates["transcode_status"] = torrentModel.TranscodeStatusProcessing
	} else if pending > 0 {
		updates["transcode_status"] = torrentModel.TranscodeStatusPending
	} else if failed > 0 && completed == 0 {
		updates["transcode_status"] = torrentModel.TranscodeStatusFailed
	} else if completed > 0 || (completed == 0 && pending == 0 && processing == 0 && failed == 0) {
		updates["transcode_status"] = torrentModel.TranscodeStatusCompleted
		updates["transcode_progress"] = 100
	}

	return h.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		Updates(updates).Error
}

// queueCloudUpload sends a cloud upload job to the queue
func (h *TranscodeHandler) queueCloudUpload(ctx context.Context, torrentID int64, infoHash string, fileIndex int, localPath string, fileSize int64, isTranscoded bool, creatorID int64) {
	if h.queueProducer == nil {
		return
	}

	// Build cloud path
	pathPrefix := h.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	fileName := filepath.Base(localPath)
	cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, infoHash, fileName)

	// Determine content type
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(localPath))
	if ext == ".mp4" {
		contentType = "video/mp4"
	}

	// Update file cloud status to pending
	h.updateTorrentFileCloudPending(torrentID, fileIndex)

	msg := cloudTypes.CloudUploadMessage{
		TorrentID:     torrentID,
		InfoHash:      infoHash,
		FileIndex:     fileIndex,
		SubtitleIndex: -1, // Not a subtitle
		LocalPath:     localPath,
		CloudPath:     cloudPath,
		ContentType:   contentType,
		FileSize:      fileSize,
		IsTranscoded:  isTranscoded,
		CreatorID:     creatorID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.loggerManager.Logger().Errorf("failed to marshal cloud upload message: %v", err)
		return
	}

	if err := h.queueProducer.Send(ctx, cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		h.loggerManager.Logger().Errorf("failed to send cloud upload message: %v", err)
	} else {
		h.loggerManager.Logger().Infof("Queued cloud upload: torrentID=%d, fileIndex=%d, cloudPath=%s",
			torrentID, fileIndex, cloudPath)
	}
}

// updateTorrentFileCloudPending marks a file's cloud upload status as pending
func (h *TranscodeHandler) updateTorrentFileCloudPending(torrentID int64, fileIndex int) {
	// Increment total cloud upload count on torrent
	h.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		UpdateColumn("total_cloud_upload", gorm.Expr("total_cloud_upload + ?", 1))

	// Update file status
	h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Update("cloud_upload_status", torrentModel.CloudUploadStatusPending)
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

// saveSubtitleInfo saves extracted subtitle info as flat file entries and queues cloud uploads
func (h *TranscodeHandler) saveSubtitleInfo(msg types.TranscodeMessage, results []ffmpeg.SubtitleExtractResult) {
	// 1. Get source file for parent path
	var sourceFile torrentModel.TorrentFile
	if err := h.dbManager.DB().Where("torrent_id = ? AND `index` = ?", msg.TorrentID, msg.FileIndex).First(&sourceFile).Error; err != nil {
		h.loggerManager.Logger().Errorf("failed to load source file for subtitle update: %v", err)
		return
	}
	parentPath := sourceFile.Path

	// 2. Get next index
	var count int64
	h.dbManager.DB().Model(&torrentModel.TorrentFile{}).Where("torrent_id = ?", msg.TorrentID).Count(&count)
	nextIndex := int(count)

	type queuedSubtitle struct {
		index  int
		result ffmpeg.SubtitleExtractResult
	}
	var queued []queuedSubtitle

	for _, r := range results {
		newFile := torrentModel.TorrentFile{
			TorrentID:     msg.TorrentID,
			Index:         nextIndex,
			Path:          r.FilePath,
			Size:          r.FileSize,
			IsSelected:    true,
			IsShareable:   false,
			IsStreamable:  false,
			Type:          "subtitle",
			Source:        "extracted",
			ParentPath:    parentPath,
			StreamIndex:   r.StreamIndex,
			Language:      r.Language,
			LanguageName:  r.LanguageName,
			Title:         r.Title,
			Format:        r.Format,
			OriginalCodec: r.OriginalCodec,
		}
		
		if err := h.dbManager.DB().Create(&newFile).Error; err != nil {
			h.loggerManager.Logger().Errorf("failed to save subtitle info: %v", err)
			continue
		}
		
		queued = append(queued, queuedSubtitle{
			index:  nextIndex,
			result: r,
		})
		nextIndex++
	}

	for _, q := range queued {
		if h.config.CloudStorageConfig.Enabled && h.queueProducer != nil {
			h.queueSubtitleCloudUpload(context.Background(), msg, q.result, q.index)
		}
	}
}

// queueSubtitleCloudUpload sends a cloud upload job for a subtitle file
func (h *TranscodeHandler) queueSubtitleCloudUpload(ctx context.Context, msg types.TranscodeMessage, result ffmpeg.SubtitleExtractResult, fileIndex int) {
	pathPrefix := h.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	fileName := filepath.Base(result.FilePath)
	cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, msg.InfoHash, fileName)

	contentType := "text/plain"
	switch result.Format {
	case "srt":
		contentType = "application/x-subrip"
	case "ass":
		contentType = "text/x-ssa"
	case "vtt":
		contentType = "text/vtt"
	}

	// Mark subtitle file as pending
	h.updateTorrentFileCloudPending(msg.TorrentID, fileIndex)

	uploadMsg := cloudTypes.CloudUploadMessage{
		TorrentID:     msg.TorrentID,
		InfoHash:      msg.InfoHash,
		FileIndex:     fileIndex,
		SubtitleIndex: -1,
		LocalPath:     result.FilePath,
		CloudPath:     cloudPath,
		ContentType:   contentType,
		FileSize:      result.FileSize,
		IsTranscoded:  false,
		CreatorID:     msg.CreatorID,
	}

	msgBytes, err := json.Marshal(uploadMsg)
	if err != nil {
		h.loggerManager.Logger().Errorf("failed to marshal subtitle cloud upload message: %v", err)
		return
	}

	if err := h.queueProducer.Send(ctx, cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		h.loggerManager.Logger().Errorf("failed to send subtitle cloud upload message: %v", err)
	} else {
		h.loggerManager.Logger().Infof("Queued subtitle cloud upload: %s", cloudPath)
	}
}
