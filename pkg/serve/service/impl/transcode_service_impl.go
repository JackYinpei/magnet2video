// Package impl provides transcode service implementation
// Author: Done-0
// Created: 2026-01-26
package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"magnet2video/configs"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/db"
	"magnet2video/internal/logger"
	torrentModel "magnet2video/internal/model/torrent"
	transcodeModel "magnet2video/internal/model/transcode"
	"magnet2video/internal/queue"
	"magnet2video/internal/transcode/ffmpeg"
	"magnet2video/internal/transcode/types"
	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/vo"
)

// TranscodeServiceImpl transcode service implementation
type TranscodeServiceImpl struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	dbManager     db.DatabaseManager
	queueProducer queue.Producer
}

// NewTranscodeService creates transcode service implementation.
//
// The service used to embed an *ffmpeg.FFmpeg client to probe files inline and
// decide remux vs transcode, but the worker is the only side that actually
// owns the on-disk files in split deployment. The server now sends a job with
// Operation="" and the worker probes locally — see TranscodeHandler.resolveOperation.
//
// Since PR3 the service no longer takes a TorrentManager either: the only
// thing it used was GetDownloadDir, which now reads from config (the worker
// shares the same download_dir relative path).
func NewTranscodeService(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	queueProducer queue.Producer,
) *TranscodeServiceImpl {
	return &TranscodeServiceImpl{
		config:        config,
		loggerManager: loggerManager,
		dbManager:     dbManager,
		queueProducer: queueProducer,
	}
}

// resolveDownloadDir returns the worker's download directory as configured.
// The server never reads from this path; it's only used to compose the
// inputPath string that the worker resolves locally.
func (ts *TranscodeServiceImpl) resolveDownloadDir() string {
	if ts.config != nil {
		return ts.config.TorrentConfig.DownloadDir
	}
	return ""
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

	// In split deployment the server has neither the worker's download/ dir
	// nor an ffmpeg binary, so we MUST NOT os.Stat or Probe here. The worker's
	// TranscodeHandler.resolveOperation probes locally when Operation is empty
	// and the CloudUploadHandler stat/opens the file itself. We just dispatch
	// jobs based on filename heuristics + DB metadata.
	downloadDir := ts.resolveDownloadDir()
	var needsTranscode bool
	var totalTranscode int

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

		// inputPath is constructed but server never reads it — it is the worker's
		// view of the file, passed verbatim through the queue. Server and worker
		// are expected to share the same effective downloadDir (relative to the
		// process working dir). If they don't, that's an ops-level config issue,
		// not something this code can fix.
		inputPath := filepath.Join(downloadDir, file.Path)

		if ffmpeg.NeedsTranscoding(file.Path) {
			needsTranscode = true
			totalTranscode++

			outputPath := ffmpeg.GenerateOutputPath(inputPath)

			// Operation="" tells the worker to probe and decide remux vs transcode.
			// InputCodec / Duration are filled in by the completion event when the
			// worker is done.
			job := &transcodeModel.TranscodeJob{
				TorrentID:     torrentID,
				InfoHash:      torrentRecord.InfoHash,
				InputPath:     inputPath,
				OutputPath:    outputPath,
				FileIndex:     i,
				Status:        transcodeModel.JobStatusPending,
				OutputCodec:   "h264",
				TranscodeType: "", // worker decides
				CreatorID:     torrentRecord.CreatorID,
			}

			if err := ts.dbManager.DB().Create(job).Error; err != nil {
				ts.loggerManager.Logger().Errorf("failed to create transcode job: %v", err)
				continue
			}

			// Outbox: send first, then mark file Pending. On send failure mark the
			// job Failed and leave the file row alone so a later retry picks it up.
			msg := types.TranscodeMessage{
				JobID:      job.ID,
				TorrentID:  torrentID,
				InfoHash:   torrentRecord.InfoHash,
				FileIndex:  i,
				InputPath:  inputPath,
				OutputPath: outputPath,
				Operation:  "", // worker resolves locally
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

			ts.loggerManager.Logger().Infof("Queued transcode job: jobID=%d torrentID=%d fileIndex=%d inputPath=%s", job.ID, torrentID, i, inputPath)

			if err := ts.dbManager.DB().Model(&torrentModel.TorrentFile{}).
				Where("torrent_id = ? AND `index` = ?", torrentID, i).
				Updates(map[string]interface{}{
					"transcode_status": torrentModel.TranscodeStatusPending,
					"transcode_error":  "",
				}).Error; err != nil {
				ts.loggerManager.Logger().Errorf("failed to mark file Pending after enqueue: torrentID=%d fileIndex=%d err=%v", torrentID, i, err)
			}
			continue
		}

		// Doesn't need transcoding → straight to cloud if enabled. Use the size
		// already in DB (worker wrote it on download.completed) instead of stat.
		if ts.config.CloudStorageConfig.Enabled {
			ts.queueCloudUpload(torrentID, torrentRecord.InfoHash, i, inputPath, file.Size, false, torrentRecord.CreatorID, &torrentRecord)
		}
	}

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
//   - Default skip: TranscodeStatus == None — CheckAndQueueTranscode handles
//     fresh files itself. Pass force=true to also reset None, which is the
//     escape hatch for "the file should have been queued by now but wasn't"
//     (e.g. the prior server build silently dropped the dispatch).
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
		case torrentModel.TranscodeStatusNone:
			// Default: leave None alone — CheckAndQueueTranscode handles fresh files.
			// With force=true, include them in the reset so CheckAndQueueTranscode is
			// guaranteed to revisit them this round (the per-file Pending-skip guard
			// would otherwise short-circuit if a previous run already moved them).
			if req.Force {
				resetIDs = append(resetIDs, f.ID)
			}
		case torrentModel.TranscodeStatusCompleted:
			// skip — nothing to redo
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

	fileIndexLog := "all"
	if req.FileIndex != nil {
		fileIndexLog = fmt.Sprintf("%d", *req.FileIndex)
	}
	ts.loggerManager.Logger().Infof("RequeueTranscode: torrentID=%d resetCount=%d force=%v fileIndex=%s",
		torrentRecord.ID, len(resetIDs), req.Force, fileIndexLog)

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
