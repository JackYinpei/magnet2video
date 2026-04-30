// Package impl provides transcode service implementation
// Author: Done-0
// Created: 2026-01-26
package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"magnet2video/configs"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/db"
	"magnet2video/internal/logger"
	torrentModel "magnet2video/internal/model/torrent"
	transcodeModel "magnet2video/internal/model/transcode"
	"magnet2video/internal/queue"
	"magnet2video/internal/torrent"
	"magnet2video/internal/transcode/ffmpeg"
	"magnet2video/internal/transcode/types"
	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/vo"
)

// TranscodeServiceImpl transcode service implementation
type TranscodeServiceImpl struct {
	config         *configs.Config
	loggerManager  logger.LoggerManager
	dbManager      db.DatabaseManager
	torrentManager torrent.TorrentManager
	queueProducer  queue.Producer
	ffmpeg         *ffmpeg.FFmpeg
}

// NewTranscodeService creates transcode service implementation
func NewTranscodeService(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	torrentManager torrent.TorrentManager,
	queueProducer queue.Producer,
) *TranscodeServiceImpl {
	return &TranscodeServiceImpl{
		config:         config,
		loggerManager:  loggerManager,
		dbManager:      dbManager,
		torrentManager: torrentManager,
		queueProducer:  queueProducer,
		ffmpeg: ffmpeg.New(
			config.TranscodeConfig.FFmpegPath,
			config.TranscodeConfig.FFprobePath,
		),
	}
}

// CheckAndQueueTranscode checks a torrent for files that need transcoding and queues jobs
func (ts *TranscodeServiceImpl) CheckAndQueueTranscode(c *gin.Context, torrentID int64) error {
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("id = ?", torrentID).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("torrent not found")
		}
		return err
	}

	// Only process completed downloads
	if torrentRecord.Status != torrentModel.StatusCompleted {
		return nil
	}

	downloadDir := ts.torrentManager.Client().GetDownloadDir()
	var needsTranscode bool
	var totalTranscode int

	// Check each file for transcoding needs
	for i, file := range torrentRecord.Files {
		if !file.IsSelected {
			continue
		}
		if file.Source != "" && file.Source != "original" {
			continue
		}

		// Skip if already transcoded or in progress
		if file.TranscodeStatus == torrentModel.TranscodeStatusCompleted ||
			file.TranscodeStatus == torrentModel.TranscodeStatusProcessing ||
			file.TranscodeStatus == torrentModel.TranscodeStatusPending {
			continue
		}

		// Check if file needs transcoding
		if ffmpeg.NeedsTranscoding(file.Path) {
			// Get full path - file.Path might not include torrent directory
			// Try both: with and without torrent name directory
			inputPath := filepath.Join(downloadDir, file.Path)

			// Check if file exists at direct path
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				// Try with torrent name as directory prefix
				inputPath = filepath.Join(downloadDir, torrentRecord.Name, filepath.Base(file.Path))
				if _, err := os.Stat(inputPath); os.IsNotExist(err) {
					ts.loggerManager.Logger().Warnf("file not found, skipping transcode check: %s", file.Path)
					continue
				}
			}

			needsTranscode = true
			totalTranscode++

			outputPath := ffmpeg.GenerateOutputPath(inputPath)

			// Probe file to determine transcode type (2 minute timeout for large files)
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			videoInfo, err := ts.ffmpeg.Probe(ctx, inputPath)
			cancel()

			if err != nil {
				ts.loggerManager.Logger().Warnf("failed to probe file %s: %v", inputPath, err)
				continue
			}

			// Determine operation type
			transcodeType := ts.ffmpeg.DetermineTranscodeType(videoInfo, inputPath)
			operation := types.OperationTranscode
			if transcodeType == ffmpeg.TranscodeTypeRemux {
				operation = types.OperationRemux
			}

			// Create transcode job in database
			job := &transcodeModel.TranscodeJob{
				TorrentID:     torrentID,
				InfoHash:      torrentRecord.InfoHash,
				InputPath:     inputPath,
				OutputPath:    outputPath,
				FileIndex:     i,
				Status:        transcodeModel.JobStatusPending,
				InputCodec:    videoInfo.Codec,
				OutputCodec:   "h264",
				TranscodeType: operation,
				Duration:      int64(videoInfo.Duration * 1000),
				CreatorID:     torrentRecord.CreatorID,
			}

			if err := ts.dbManager.DB().Create(job).Error; err != nil {
				ts.loggerManager.Logger().Errorf("failed to create transcode job: %v", err)
				continue
			}

			// Outbox-style: send the message FIRST. Only on success do we mark the
			// file Pending. If sending fails we leave the file row alone (so a later
			// retry can pick it up) and mark the job Failed so it doesn't dangle.
			msg := types.TranscodeMessage{
				JobID:      job.ID,
				TorrentID:  torrentID,
				InfoHash:   torrentRecord.InfoHash,
				FileIndex:  i,
				InputPath:  inputPath,
				OutputPath: outputPath,
				InputCodec: videoInfo.Codec,
				Operation:  operation,
				Priority:   5,
				CreatorID:  torrentRecord.CreatorID,
				Preset:     ts.config.TranscodeConfig.DefaultPreset,
				CRF:        ts.config.TranscodeConfig.DefaultCRF,
			}

			msgBytes, _ := json.Marshal(msg)
			if err := ts.queueProducer.Send(context.Background(), types.TopicTranscodeJobs, nil, msgBytes); err != nil {
				ts.loggerManager.Logger().Errorf("failed to send transcode message (file status untouched, retry safe): jobID=%d fileIndex=%d err=%v", job.ID, i, err)
				ts.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).Where("id = ?", job.ID).
					Updates(map[string]interface{}{
						"status":        transcodeModel.JobStatusFailed,
						"error_message": "failed to queue transcode job",
					})
				continue
			}

			// Send succeeded → mark file Pending.
			if err := ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
				Where("torrent_id = ? AND `index` = ?", torrentID, i).
				Updates(map[string]interface{}{
					"transcode_status": torrentModel.TranscodeStatusPending,
					"transcode_error":  "",
				}).Error; err != nil {
				ts.loggerManager.Logger().Errorf("failed to mark file Pending after enqueue: torrentID=%d fileIndex=%d err=%v", torrentID, i, err)
			}
		} else {
			// File doesn't need transcoding, queue for cloud upload directly if enabled
			if ts.config.CloudStorageConfig.Enabled {
				inputPath := filepath.Join(downloadDir, file.Path)
				// Check if file exists at direct path
				if _, err := os.Stat(inputPath); os.IsNotExist(err) {
					// Try with torrent name as directory prefix
					inputPath = filepath.Join(downloadDir, torrentRecord.Name, filepath.Base(file.Path))
				}

				fileInfo, err := os.Stat(inputPath)
				if err == nil {
					ts.queueCloudUpload(torrentID, torrentRecord.InfoHash, i, inputPath, fileInfo.Size(), false, torrentRecord.CreatorID, &torrentRecord)
				}
			}
		}
	}

	// Update torrent record aggregates
	if needsTranscode {
		ts.dbManager.DB().Model(&torrentRecord).Updates(map[string]interface{}{
			"transcode_status":   torrentModel.TranscodeStatusPending,
			"total_transcode":    totalTranscode,
			"transcode_progress": 0,
		})
	}

	return nil
}

// GetTranscodeStatus returns the transcode status for a torrent
func (ts *TranscodeServiceImpl) GetTranscodeStatus(c *gin.Context, torrentID int64) (*vo.TranscodeStatusResponse, error) {
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("id = ?", torrentID).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("torrent not found")
		}
		return nil, err
	}

	// Get transcode jobs
	var jobs []transcodeModel.TranscodeJob
	ts.dbManager.DB().Where("torrent_id = ?", torrentID).Order("created_at DESC").Find(&jobs)

	// Build file info (original files only)
	files := make([]vo.TranscodeFileInfo, 0, len(torrentRecord.Files))
	var transcodeFiles, completedFiles int

	for i, file := range torrentRecord.Files {
		if file.Source != "" && file.Source != "original" {
			continue
		}
		needsTranscode := ffmpeg.NeedsTranscoding(file.Path)
		if needsTranscode {
			transcodeFiles++
		}
		if file.TranscodeStatus == torrentModel.TranscodeStatusCompleted {
			completedFiles++
		}

		files = append(files, vo.TranscodeFileInfo{
			FileIndex:       i,
			FilePath:        file.Path,
			TranscodeStatus: file.TranscodeStatus,
			TranscodedPath:  file.TranscodedPath,
			TranscodeError:  file.TranscodeError,
			NeedsTranscode:  needsTranscode,
		})
	}

	// Build job info
	jobInfos := make([]vo.TranscodeJobInfo, len(jobs))
	for i, job := range jobs {
		jobInfos[i] = vo.TranscodeJobInfo{
			ID:            job.ID,
			TorrentID:     job.TorrentID,
			InfoHash:      job.InfoHash,
			FileIndex:     job.FileIndex,
			InputPath:     job.InputPath,
			OutputPath:    job.OutputPath,
			Status:        job.Status,
			Progress:      job.Progress,
			InputCodec:    job.InputCodec,
			OutputCodec:   job.OutputCodec,
			TranscodeType: job.TranscodeType,
			ErrorMessage:  job.ErrorMessage,
			StartedAt:     job.StartedAt,
			CompletedAt:   job.CompletedAt,
			CreatedAt:     job.CreatedAt,
		}
	}

	return &vo.TranscodeStatusResponse{
		TorrentID:       torrentID,
		InfoHash:        torrentRecord.InfoHash,
		OverallStatus:   torrentRecord.TranscodeStatus,
		OverallProgress: torrentRecord.TranscodeProgress,
		TotalFiles:      len(torrentRecord.Files),
		TranscodeFiles:  transcodeFiles,
		CompletedFiles:  completedFiles,
		Files:           files,
		Jobs:            jobInfos,
	}, nil
}

// RetryTranscode retries a failed transcode job
func (ts *TranscodeServiceImpl) RetryTranscode(c *gin.Context, req *dto.RetryTranscodeRequest) (*vo.RetryTranscodeResponse, error) {
	var oldJob transcodeModel.TranscodeJob
	if err := ts.dbManager.DB().Where("id = ?", req.JobID).First(&oldJob).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, err
	}

	if oldJob.Status != transcodeModel.JobStatusFailed {
		return nil, errors.New("can only retry failed jobs")
	}

	// Create new job
	newJob := &transcodeModel.TranscodeJob{
		TorrentID:     oldJob.TorrentID,
		InfoHash:      oldJob.InfoHash,
		InputPath:     oldJob.InputPath,
		OutputPath:    oldJob.OutputPath,
		FileIndex:     oldJob.FileIndex,
		Status:        transcodeModel.JobStatusPending,
		InputCodec:    oldJob.InputCodec,
		OutputCodec:   oldJob.OutputCodec,
		TranscodeType: oldJob.TranscodeType,
		Duration:      oldJob.Duration,
		CreatorID:     oldJob.CreatorID,
	}

	if err := ts.dbManager.DB().Create(newJob).Error; err != nil {
		return nil, err
	}

	// Outbox-style: send first; only on success update DB statuses.
	msg := types.TranscodeMessage{
		JobID:      newJob.ID,
		TorrentID:  newJob.TorrentID,
		InfoHash:   newJob.InfoHash,
		FileIndex:  newJob.FileIndex,
		InputPath:  newJob.InputPath,
		OutputPath: newJob.OutputPath,
		InputCodec: newJob.InputCodec,
		Operation:  newJob.TranscodeType,
		Priority:   5,
		CreatorID:  newJob.CreatorID,
		Preset:     ts.config.TranscodeConfig.DefaultPreset,
		CRF:        ts.config.TranscodeConfig.DefaultCRF,
	}

	msgBytes, _ := json.Marshal(msg)
	if err := ts.queueProducer.Send(context.Background(), types.TopicTranscodeJobs, nil, msgBytes); err != nil {
		ts.loggerManager.Logger().Errorf("failed to send transcode message: jobID=%d err=%v", newJob.ID, err)
		// Roll back the new job so it doesn't dangle in Pending.
		ts.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).Where("id = ?", newJob.ID).
			Updates(map[string]interface{}{
				"status":        transcodeModel.JobStatusFailed,
				"error_message": "failed to queue transcode job",
			})
		return nil, fmt.Errorf("failed to queue transcode job: %w", err)
	}

	// Send succeeded → mark file + torrent Pending.
	ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", oldJob.TorrentID, oldJob.FileIndex).
		Updates(map[string]interface{}{
			"transcode_status": torrentModel.TranscodeStatusPending,
			"transcode_error":  "",
		})

	ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", oldJob.TorrentID).
		Update("transcode_status", torrentModel.TranscodeStatusPending)

	return &vo.RetryTranscodeResponse{
		JobID:   newJob.ID,
		Message: "Transcode job queued for retry",
	}, nil
}

// CancelTranscode cancels a pending or processing transcode job
func (ts *TranscodeServiceImpl) CancelTranscode(c *gin.Context, jobID int64) (*vo.CancelTranscodeResponse, error) {
	var job transcodeModel.TranscodeJob
	if err := ts.dbManager.DB().Where("id = ?", jobID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("job not found")
		}
		return nil, err
	}

	if job.Status != transcodeModel.JobStatusPending && job.Status != transcodeModel.JobStatusProcessing {
		return nil, errors.New("can only cancel pending or processing jobs")
	}

	// Update job status to failed (canceled)
	if err := ts.dbManager.DB().Model(&job).Updates(map[string]interface{}{
		"status":        transcodeModel.JobStatusFailed,
		"error_message": "canceled by user",
	}).Error; err != nil {
		return nil, err
	}

	// Update torrent file status
	ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", job.TorrentID, job.FileIndex).
		Updates(map[string]interface{}{
			"transcode_status": torrentModel.TranscodeStatusFailed,
			"transcode_error":  "canceled by user",
		})

	return &vo.CancelTranscodeResponse{
		JobID:   jobID,
		Message: "Transcode job canceled",
	}, nil
}

// TriggerTranscodeCheck triggers transcode check for a torrent (called asynchronously after download completes)
func (ts *TranscodeServiceImpl) TriggerTranscodeCheck(torrentID int64) {
	ts.loggerManager.Logger().Infof("Triggering transcode check for torrent ID: %d", torrentID)

	// Use a background context since this is called asynchronously
	if err := ts.CheckAndQueueTranscode(&gin.Context{}, torrentID); err != nil {
		ts.loggerManager.Logger().Errorf("Failed to check and queue transcode: %v", err)
	}
}

// RequeueTranscode resets transcode state for an eligible set of original files
// and re-runs CheckAndQueueTranscode so the worker picks them up again.
//
// Eligibility (per file):
//   - Always reset: TranscodeStatus == Failed
//   - Default skip: TranscodeStatus == Pending or Processing (worker may still
//     be doing it). Pass force=true to override.
//   - Always skip: TranscodeStatus == Completed (no point — nothing to redo)
//   - Always skip: TranscodeStatus == None — nothing to reset; CheckAndQueueTranscode
//     handles fresh files itself.
//
// Verifies caller owns the torrent.
func (ts *TranscodeServiceImpl) RequeueTranscode(c *gin.Context, req *dto.RequeueTranscodeRequest, callerUserID int64) (*vo.RequeueTranscodeResponse, error) {
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND deleted = ?", req.InfoHash, false).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("torrent not found")
		}
		return nil, err
	}

	if torrentRecord.CreatorID != callerUserID {
		return nil, errors.New("not the torrent creator")
	}

	// Build the set of file indexes to reset.
	var resetIDs []int64
	for _, f := range torrentRecord.Files {
		// Only original files participate — derived (transcoded/extracted) rows
		// are byproducts, not source material.
		if f.Source != "" && f.Source != "original" {
			continue
		}
		if req.FileIndex != nil && f.Index != *req.FileIndex {
			continue
		}

		switch f.TranscodeStatus {
		case torrentModel.TranscodeStatusFailed:
			resetIDs = append(resetIDs, f.ID)
		case torrentModel.TranscodeStatusPending, torrentModel.TranscodeStatusProcessing:
			if req.Force {
				resetIDs = append(resetIDs, f.ID)
			}
		case torrentModel.TranscodeStatusCompleted, torrentModel.TranscodeStatusNone:
			// skip
		}
	}

	if len(resetIDs) > 0 {
		if err := ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
			Where("id IN ?", resetIDs).
			Updates(map[string]any{
				"transcode_status": torrentModel.TranscodeStatusNone,
				"transcoded_path":  "",
				"transcode_error":  "",
			}).Error; err != nil {
			return nil, fmt.Errorf("reset file transcode state: %w", err)
		}
	}

	// Reset torrent-level aggregate so CheckAndQueueTranscode can rebuild it.
	if err := ts.dbManager.DB().Model(&torrentRecord).Updates(map[string]any{
		"transcode_status":   torrentModel.TranscodeStatusNone,
		"transcode_progress": 0,
	}).Error; err != nil {
		return nil, fmt.Errorf("reset torrent transcode state: %w", err)
	}

	// Make sure the torrent is marked completed so CheckAndQueueTranscode
	// doesn't bail at its early "only completed downloads" guard.
	if torrentRecord.Status == torrentModel.StatusCompleted {
		if err := ts.CheckAndQueueTranscode(c, torrentRecord.ID); err != nil {
			return nil, fmt.Errorf("re-queue transcode: %w", err)
		}
	} else {
		ts.loggerManager.Logger().Warnf("RequeueTranscode skipped CheckAndQueue because torrent status != completed: torrentID=%d status=%d",
			torrentRecord.ID, torrentRecord.Status)
	}

	ts.loggerManager.Logger().Infof("RequeueTranscode: torrentID=%d resetCount=%d force=%v fileIndex=%v",
		torrentRecord.ID, len(resetIDs), req.Force, req.FileIndex)

	return &vo.RequeueTranscodeResponse{
		InfoHash:      req.InfoHash,
		RequeuedFiles: len(resetIDs),
		Message:       fmt.Sprintf("Reset %d file(s) and re-ran transcode check", len(resetIDs)),
	}, nil
}

// queueCloudUpload sends a cloud upload job to the queue
func (ts *TranscodeServiceImpl) queueCloudUpload(torrentID int64, infoHash string, fileIndex int, localPath string, fileSize int64, isTranscoded bool, creatorID int64, torrentRecord *torrentModel.Torrent) {
	// Build cloud path
	pathPrefix := ts.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	fileName := filepath.Base(localPath)
	cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, infoHash, fileName)

	// Determine content type
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(localPath))
	contentTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".mp3":  "audio/mpeg",
		".flac": "audio/flac",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
	}
	if ct, ok := contentTypes[ext]; ok {
		contentType = ct
	}

	msg := cloudTypes.CloudUploadMessage{
		TorrentID:     torrentID,
		InfoHash:      infoHash,
		FileIndex:     fileIndex,
		SubtitleIndex: -1,
		LocalPath:     localPath,
		CloudPath:     cloudPath,
		ContentType:   contentType,
		FileSize:      fileSize,
		IsTranscoded:  isTranscoded,
		CreatorID:     creatorID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to marshal cloud upload message: torrentID=%d fileIndex=%d err=%v", torrentID, fileIndex, err)
		return
	}

	// Outbox-style: send first; only on success mark Pending. total_cloud_upload
	// is no longer incremented here — it's recomputed from per-file rows by the
	// event processor's recomputeCloudStatus, so retries don't double-count.
	if err := ts.queueProducer.Send(context.Background(), cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		ts.loggerManager.Logger().Errorf("failed to send cloud upload message (status untouched, retry safe): torrentID=%d fileIndex=%d err=%v", torrentID, fileIndex, err)
		return
	}

	if err := ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(map[string]interface{}{
			"cloud_upload_status": torrentModel.CloudUploadStatusPending,
			"cloud_upload_error":  "",
		}).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to mark file cloud Pending after enqueue: torrentID=%d fileIndex=%d err=%v", torrentID, fileIndex, err)
	}

	ts.loggerManager.Logger().Infof("Queued cloud upload: torrentID=%d fileIndex=%d cloudPath=%s", torrentID, fileIndex, cloudPath)
}
