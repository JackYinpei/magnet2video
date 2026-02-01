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

	"github.com/Done-0/gin-scaffold/configs"
	cloudTypes "github.com/Done-0/gin-scaffold/internal/cloud/types"
	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	"github.com/Done-0/gin-scaffold/internal/queue"
	"github.com/Done-0/gin-scaffold/internal/torrent"
	"github.com/Done-0/gin-scaffold/internal/transcode/ffmpeg"
	"github.com/Done-0/gin-scaffold/internal/transcode/types"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/vo"
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
	if err := ts.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
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

			// Update file status to pending
			torrentRecord.Files[i].TranscodeStatus = torrentModel.TranscodeStatusPending

			// Send message to Kafka
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
				ts.loggerManager.Logger().Errorf("failed to send transcode message: %v", err)
				// Update job status to failed
				ts.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).Where("id = ?", job.ID).
					Update("status", transcodeModel.JobStatusFailed)
				torrentRecord.Files[i].TranscodeStatus = torrentModel.TranscodeStatusFailed
				torrentRecord.Files[i].TranscodeError = "failed to queue transcode job"
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

	// Update torrent record
	if needsTranscode {
		torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusPending
		torrentRecord.TotalTranscode = totalTranscode
		torrentRecord.TranscodeProgress = 0
	}

	return ts.dbManager.DB().Save(&torrentRecord).Error
}

// GetTranscodeStatus returns the transcode status for a torrent
func (ts *TranscodeServiceImpl) GetTranscodeStatus(c *gin.Context, torrentID int64) (*vo.TranscodeStatusResponse, error) {
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("torrent not found")
		}
		return nil, err
	}

	// Get transcode jobs
	var jobs []transcodeModel.TranscodeJob
	ts.dbManager.DB().Where("torrent_id = ?", torrentID).Order("created_at DESC").Find(&jobs)

	// Build file info
	files := make([]vo.TranscodeFileInfo, len(torrentRecord.Files))
	var transcodeFiles, completedFiles int

	for i, file := range torrentRecord.Files {
		needsTranscode := ffmpeg.NeedsTranscoding(file.Path)
		if needsTranscode {
			transcodeFiles++
		}
		if file.TranscodeStatus == torrentModel.TranscodeStatusCompleted {
			completedFiles++
		}

		files[i] = vo.TranscodeFileInfo{
			FileIndex:       i,
			FilePath:        file.Path,
			TranscodeStatus: file.TranscodeStatus,
			TranscodedPath:  file.TranscodedPath,
			TranscodeError:  file.TranscodeError,
			NeedsTranscode:  needsTranscode,
		}
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

	// Update torrent file status
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().Where("id = ?", oldJob.TorrentID).First(&torrentRecord).Error; err == nil {
		if oldJob.FileIndex >= 0 && oldJob.FileIndex < len(torrentRecord.Files) {
			torrentRecord.Files[oldJob.FileIndex].TranscodeStatus = torrentModel.TranscodeStatusPending
			torrentRecord.Files[oldJob.FileIndex].TranscodeError = ""
			torrentRecord.TranscodeStatus = torrentModel.TranscodeStatusPending
			ts.dbManager.DB().Save(&torrentRecord)
		}
	}

	// Send message to Kafka
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
		ts.loggerManager.Logger().Errorf("failed to send transcode message: %v", err)
		return nil, fmt.Errorf("failed to queue transcode job: %w", err)
	}

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
	var torrentRecord torrentModel.Torrent
	if err := ts.dbManager.DB().Where("id = ?", job.TorrentID).First(&torrentRecord).Error; err == nil {
		if job.FileIndex >= 0 && job.FileIndex < len(torrentRecord.Files) {
			torrentRecord.Files[job.FileIndex].TranscodeStatus = torrentModel.TranscodeStatusFailed
			torrentRecord.Files[job.FileIndex].TranscodeError = "canceled by user"
			ts.dbManager.DB().Save(&torrentRecord)
		}
	}

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

	// Update file cloud status to pending
	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].CloudUploadStatus = torrentModel.CloudUploadStatusPending
		torrentRecord.TotalCloudUpload++
	}

	msg := cloudTypes.CloudUploadMessage{
		TorrentID:    torrentID,
		InfoHash:     infoHash,
		FileIndex:    fileIndex,
		LocalPath:    localPath,
		CloudPath:    cloudPath,
		ContentType:  contentType,
		FileSize:     fileSize,
		IsTranscoded: isTranscoded,
		CreatorID:    creatorID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to marshal cloud upload message: %v", err)
		return
	}

	if err := ts.queueProducer.Send(context.Background(), cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		ts.loggerManager.Logger().Errorf("failed to send cloud upload message: %v", err)
	} else {
		ts.loggerManager.Logger().Infof("Queued cloud upload: torrentID=%d, fileIndex=%d, cloudPath=%s",
			torrentID, fileIndex, cloudPath)
	}
}
