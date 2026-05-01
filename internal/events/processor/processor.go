// Package processor consumes worker events on the server side and applies them
// to the database, queueing follow-up jobs (e.g. cloud uploads) as needed.
// Author: magnet2video
// Created: 2026-04-20
package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"magnet2video/configs"
	"magnet2video/internal/cache"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/db"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	torrentModel "magnet2video/internal/model/torrent"
	transcodeModel "magnet2video/internal/model/transcode"
	"magnet2video/internal/queue"
	redisMgr "magnet2video/internal/redis"
	"magnet2video/pkg/serve/service"
)

const (
	// idempotencyTTL is how long a processed event id is remembered in Redis.
	idempotencyTTL = 5 * time.Minute
	// idempotencyKeyPrefix is the redis key prefix for the SETNX dedup.
	idempotencyKeyPrefix = "worker:event:seen:"
	// downloadProgressTTL keeps high-frequency download stats out of MySQL.
	downloadProgressTTL = 30 * time.Second
)

// WorkerEventProcessor translates worker events into database mutations and
// follow-up queue messages. Implements queue.Handler.
type WorkerEventProcessor struct {
	config           *configs.Config
	loggerManager    logger.LoggerManager
	dbManager        db.DatabaseManager
	redisManager     redisMgr.RedisManager
	queueProducer    queue.Producer
	transcodeChecker service.TranscodeChecker

	progressLogMu sync.Mutex
	progressLogAt map[string]time.Time
}

// NewWorkerEventProcessor constructs a processor.
func NewWorkerEventProcessor(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	redisManager redisMgr.RedisManager,
	queueProducer queue.Producer,
) *WorkerEventProcessor {
	return &WorkerEventProcessor{
		config:        config,
		loggerManager: loggerManager,
		dbManager:     dbManager,
		redisManager:  redisManager,
		queueProducer: queueProducer,
		progressLogAt: make(map[string]time.Time),
	}
}

// SetTranscodeChecker wires the transcode service for post-download triggering.
// Called after all services are initialised to avoid cyclic construction.
func (p *WorkerEventProcessor) SetTranscodeChecker(checker service.TranscodeChecker) {
	p.transcodeChecker = checker
}

// Handle implements queue.Handler.
func (p *WorkerEventProcessor) Handle(ctx context.Context, msg *queue.Message) error {
	var event eventTypes.WorkerEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		p.loggerManager.Logger().Errorf("failed to unmarshal worker event: %v", err)
		return nil // drop malformed events rather than re-queuing
	}

	// Idempotency: same event-id within TTL is processed once.
	if p.isDuplicate(ctx, event.EventID) {
		return nil
	}

	switch event.EventType {
	case eventTypes.EventTypeTranscodeJobStarted:
		return p.handleTranscodeStarted(ctx, &event)
	case eventTypes.EventTypeTranscodeJobProgress:
		return p.handleTranscodeProgress(ctx, &event)
	case eventTypes.EventTypeTranscodeJobCompleted:
		return p.handleTranscodeCompleted(ctx, &event)
	case eventTypes.EventTypeTranscodeJobFailed:
		return p.handleTranscodeFailed(ctx, &event)
	case eventTypes.EventTypeSubtitleExtracted:
		return p.handleSubtitleExtracted(ctx, &event)
	case eventTypes.EventTypeCloudUploadStarted:
		return p.handleCloudUploadStarted(ctx, &event)
	case eventTypes.EventTypeCloudUploadCompleted:
		return p.handleCloudUploadCompleted(ctx, &event)
	case eventTypes.EventTypeCloudUploadFailed:
		return p.handleCloudUploadFailed(ctx, &event)
	case eventTypes.EventTypeDownloadProgress:
		return p.handleDownloadProgress(ctx, &event)
	case eventTypes.EventTypeDownloadCompleted:
		return p.handleDownloadCompleted(ctx, &event)
	case eventTypes.EventTypeDownloadFailed:
		return p.handleDownloadFailed(ctx, &event)
	case eventTypes.EventTypePosterCandidateUploaded:
		return p.handlePosterCandidate(ctx, &event)
	default:
		p.loggerManager.Logger().Warnf("unknown worker event type: %s", event.EventType)
		return nil
	}
}

// isDuplicate uses Redis SETNX to dedupe events by EventID within idempotencyTTL.
// Returns true if this event has already been processed.
func (p *WorkerEventProcessor) isDuplicate(ctx context.Context, eventID string) bool {
	if eventID == "" || p.redisManager == nil {
		return false
	}
	client := p.redisManager.Client()
	if client == nil {
		return false
	}
	key := idempotencyKeyPrefix + eventID
	ok, err := client.SetNX(ctx, key, 1, idempotencyTTL).Result()
	if err != nil {
		p.loggerManager.Logger().Warnf("redis setnx failed for event dedup: %v", err)
		return false
	}
	return !ok // SetNX returns true when the key was set (= first time); duplicate when false
}

// ---- Transcode handlers ----

func (p *WorkerEventProcessor) handleTranscodeStarted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.TranscodeJobStartedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	db := p.dbManager.DB()
	if err := db.Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", payload.JobID).
		Updates(map[string]any{
			"status":     transcodeModel.JobStatusProcessing,
			"progress":   0,
			"started_at": time.Now().Unix(),
		}).Error; err != nil {
		return err
	}
	db.Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Update("transcode_status", torrentModel.TranscodeStatusProcessing)
	return p.recomputeTranscodeStatus(payload.TorrentID)
}

func (p *WorkerEventProcessor) handleTranscodeProgress(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.TranscodeJobProgressPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	db := p.dbManager.DB()
	db.Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", payload.JobID).
		Update("progress", payload.Progress)
	// Aggregate torrent transcode_progress using the completed-count + current-progress heuristic.
	return p.recomputeTranscodeProgress(payload.TorrentID, payload.Progress)
}

func (p *WorkerEventProcessor) handleTranscodeCompleted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.TranscodeJobCompletedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	log := p.loggerManager.Logger()
	log.Infof("Transcode completed: jobID=%d torrentID=%d fileIndex=%d outputPath=%s outputSize=%d",
		payload.JobID, payload.TorrentID, payload.FileIndex, payload.OutputPath, payload.OutputSize)

	db := p.dbManager.DB()

	// 1. Update job record
	if err := db.Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", payload.JobID).
		Updates(map[string]any{
			"status":       transcodeModel.JobStatusCompleted,
			"progress":     100,
			"output_path":  payload.OutputPath,
			"completed_at": time.Now().Unix(),
		}).Error; err != nil {
		log.Errorf("Failed to update transcode job record: jobID=%d err=%v", payload.JobID, err)
		return err
	}

	// 2. Update source file transcode status
	var sourceFile torrentModel.TorrentFile
	if err := db.Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).First(&sourceFile).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("Failed to query source file: torrentID=%d fileIndex=%d err=%v", payload.TorrentID, payload.FileIndex, err)
			return err
		}
		log.Warnf("Source file not found for transcode completed event: torrentID=%d fileIndex=%d", payload.TorrentID, payload.FileIndex)
	} else {
		if err := db.Model(&sourceFile).Updates(map[string]any{
			"transcode_status": torrentModel.TranscodeStatusCompleted,
			"transcoded_path":  payload.OutputPath,
			"transcode_error":  "",
		}).Error; err != nil {
			log.Errorf("Failed to update source file transcode status: torrentID=%d fileIndex=%d err=%v", payload.TorrentID, payload.FileIndex, err)
			return err
		}
	}

	// 3. Create transcoded file entry. We can't rely on Count + Create being safe
	// alone — under MySQL REPEATABLE READ two concurrent completions can read the
	// same Count snapshot. The DB-level unique index on (torrent_id, index)
	// (installed in migration.go) is the real guard; here we retry on conflict
	// using MAX(index)+1 instead of Count, so a clash just bumps to the next free
	// slot. Bounded retries cap the worst case.
	parentPath := ""
	if sourceFile.ID != 0 {
		parentPath = sourceFile.Path
	}
	var newFile torrentModel.TorrentFile
	if err := createTranscodedFileWithRetry(db, &newFile, payload.TorrentID, payload.OutputPath, payload.OutputSize, parentPath); err != nil {
		log.Errorf("Failed to create transcoded file record: torrentID=%d outputPath=%s err=%v", payload.TorrentID, payload.OutputPath, err)
		return err
	}
	log.Infof("Transcoded file record created: torrentID=%d newIndex=%d path=%s", payload.TorrentID, newFile.Index, newFile.Path)

	// 4. Queue cloud upload for the transcoded file (if cloud storage is on).
	if p.config.CloudStorageConfig.Enabled && p.queueProducer != nil {
		log.Infof("Queuing cloud upload: torrentID=%d fileIndex=%d path=%s", payload.TorrentID, newFile.Index, payload.OutputPath)
		p.queueCloudUpload(ctx, payload.TorrentID, payload.InfoHash, newFile.Index, payload.OutputPath, payload.OutputSize, true, payload.CreatorID)
	} else {
		log.Infof("Cloud upload skipped (disabled or no producer): torrentID=%d", payload.TorrentID)
	}

	return p.recomputeTranscodeStatus(payload.TorrentID)
}

func (p *WorkerEventProcessor) handleTranscodeFailed(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.TranscodeJobFailedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	db := p.dbManager.DB()
	db.Model(&transcodeModel.TranscodeJob{}).
		Where("id = ?", payload.JobID).
		Updates(map[string]any{
			"status":        transcodeModel.JobStatusFailed,
			"error_message": payload.ErrorMsg,
		})
	db.Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Updates(map[string]any{
			"transcode_status": torrentModel.TranscodeStatusFailed,
			"transcode_error":  payload.ErrorMsg,
		})
	return p.recomputeTranscodeStatus(payload.TorrentID)
}

func (p *WorkerEventProcessor) handleSubtitleExtracted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.SubtitleExtractedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	db := p.dbManager.DB()

	var parent torrentModel.TorrentFile
	parentPath := ""
	if err := db.Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.ParentFileIndex).First(&parent).Error; err == nil {
		parentPath = parent.Path
	}

	var count int64
	db.Model(&torrentModel.TorrentFile{}).Where("torrent_id = ?", payload.TorrentID).Count(&count)
	newIndex := int(count)

	newFile := torrentModel.TorrentFile{
		TorrentID:     payload.TorrentID,
		Index:         newIndex,
		Path:          payload.FilePath,
		Size:          payload.FileSize,
		IsSelected:    true,
		IsShareable:   false,
		IsStreamable:  false,
		Type:          "subtitle",
		Source:        "extracted",
		ParentPath:    parentPath,
		StreamIndex:   payload.StreamIndex,
		Language:      payload.Language,
		LanguageName:  payload.LanguageName,
		Title:         payload.Title,
		Format:        payload.Format,
		OriginalCodec: payload.OriginalCodec,
	}
	if err := db.Create(&newFile).Error; err != nil {
		return err
	}

	if p.config.CloudStorageConfig.Enabled && p.queueProducer != nil {
		p.queueCloudUpload(ctx, payload.TorrentID, payload.InfoHash, newIndex, payload.FilePath, payload.FileSize, false, payload.CreatorID)
	}
	return nil
}

// ---- Cloud upload handlers ----

func (p *WorkerEventProcessor) handleCloudUploadStarted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.CloudUploadStartedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Update("cloud_upload_status", torrentModel.CloudUploadStatusUploading)
	return p.recomputeCloudStatus(payload.TorrentID)
}

func (p *WorkerEventProcessor) handleCloudUploadCompleted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.CloudUploadCompletedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Updates(map[string]any{
			"cloud_upload_status": torrentModel.CloudUploadStatusCompleted,
			"cloud_path":          payload.CloudPath,
			"cloud_upload_error":  "",
		})
	return p.recomputeCloudStatus(payload.TorrentID)
}

func (p *WorkerEventProcessor) handleCloudUploadFailed(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.CloudUploadFailedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Updates(map[string]any{
			"cloud_upload_status": torrentModel.CloudUploadStatusFailed,
			"cloud_upload_error":  payload.ErrorMsg,
		})
	return p.recomputeCloudStatus(payload.TorrentID)
}

// ---- Download handlers ----

func (p *WorkerEventProcessor) handleDownloadProgress(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.DownloadProgressPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	if p.shouldLogDownloadProgress(payload.InfoHash) {
		p.loggerManager.Logger().Infof(
			"Download progress event received: workerID=%s infoHash=%s status=%s progress=%.2f%% speed=%dB/s peers=%d seeds=%d",
			event.WorkerID,
			payload.InfoHash,
			payload.Status,
			payload.Progress,
			payload.DownloadSpeed,
			payload.Peers,
			payload.Seeds,
		)
	}
	if p.redisManager == nil || p.redisManager.Client() == nil {
		if p.shouldLogProgress("redis-unavailable:" + payload.InfoHash) {
			p.loggerManager.Logger().Warnf("Download progress event skipped: Redis unavailable, infoHash=%s", payload.InfoHash)
		}
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := p.redisManager.Client().Set(ctx, cache.TorrentProgressKey(payload.InfoHash), data, downloadProgressTTL).Err(); err != nil {
		p.loggerManager.Logger().Warnf("failed to cache download progress: %v", err)
	} else if p.shouldLogDownloadProgressCache(payload.InfoHash) {
		p.loggerManager.Logger().Infof("Download progress cached in Redis: key=%s ttl=%s", cache.TorrentProgressKey(payload.InfoHash), downloadProgressTTL)
	}
	return nil
}

func (p *WorkerEventProcessor) shouldLogDownloadProgress(infoHash string) bool {
	return p.shouldLogProgress("received:" + infoHash)
}

func (p *WorkerEventProcessor) shouldLogDownloadProgressCache(infoHash string) bool {
	return p.shouldLogProgress("cached:" + infoHash)
}

func (p *WorkerEventProcessor) shouldLogProgress(key string) bool {
	p.progressLogMu.Lock()
	defer p.progressLogMu.Unlock()
	now := time.Now()
	last, ok := p.progressLogAt[key]
	if ok && now.Sub(last) < 10*time.Second {
		return false
	}
	p.progressLogAt[key] = now
	return true
}

func (p *WorkerEventProcessor) handleDownloadCompleted(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.DownloadCompletedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	var t torrentModel.Torrent
	if err := p.dbManager.DB().Where("info_hash = ?", payload.InfoHash).First(&t).Error; err != nil {
		return err
	}
	p.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", t.ID).
		Updates(map[string]any{
			"status":   torrentModel.StatusCompleted,
			"progress": 100,
		})
	if p.redisManager != nil && p.redisManager.Client() != nil {
		_ = p.redisManager.Client().Del(ctx, cache.TorrentProgressKey(payload.InfoHash)).Err()
	}
	if p.transcodeChecker != nil {
		go p.transcodeChecker.TriggerTranscodeCheck(t.ID)
	}
	return nil
}

func (p *WorkerEventProcessor) handleDownloadFailed(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.DownloadFailedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	p.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", payload.InfoHash).
		Update("status", torrentModel.StatusFailed)
	if p.redisManager != nil && p.redisManager.Client() != nil {
		_ = p.redisManager.Client().Del(ctx, cache.TorrentProgressKey(payload.InfoHash)).Err()
	}
	return nil
}

func (p *WorkerEventProcessor) handlePosterCandidate(ctx context.Context, event *eventTypes.WorkerEvent) error {
	var payload eventTypes.PosterCandidateUploadedPayload
	if err := event.DecodePayload(&payload); err != nil {
		return err
	}
	// Treat as a regular cloud-upload completion for the image file.
	p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", payload.TorrentID, payload.FileIndex).
		Updates(map[string]any{
			"cloud_upload_status": torrentModel.CloudUploadStatusCompleted,
			"cloud_path":          payload.CloudPath,
			"cloud_upload_error":  "",
		})
	return p.recomputeCloudStatus(payload.TorrentID)
}

// ---- helpers ----

// queueCloudUpload enqueues a cloud-upload job message.
//
// Outbox-style ordering: send the message FIRST; only on send-success write
// Pending to the file row. If sending fails the file's cloud status is left
// untouched (stays None / Failed / whatever it was), which lets a later retry
// pick it up rather than stranding it in Pending forever.
//
// total_cloud_upload is NOT incremented here — that aggregate is now derived
// in recomputeCloudStatus from per-file rows, so it can't drift on retries.
func (p *WorkerEventProcessor) queueCloudUpload(ctx context.Context, torrentID int64, infoHash string, fileIndex int, localPath string, fileSize int64, isTranscoded bool, creatorID int64) {
	log := p.loggerManager.Logger()
	pathPrefix := p.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	fileName := filepath.Base(localPath)
	cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, infoHash, fileName)

	contentType := guessContentType(localPath)

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
		log.Errorf("marshal cloud upload message: torrentID=%d fileIndex=%d err=%v", torrentID, fileIndex, err)
		return
	}

	if err := p.queueProducer.Send(ctx, cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		log.Errorf("send cloud upload message failed (status untouched, retry safe): torrentID=%d fileIndex=%d err=%v",
			torrentID, fileIndex, err)
		return
	}

	// Send succeeded → mark file Pending and recompute the aggregate.
	if err := p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(map[string]any{
			"cloud_upload_status": torrentModel.CloudUploadStatusPending,
			"cloud_upload_error":  "",
		}).Error; err != nil {
		log.Errorf("update file cloud_upload_status=Pending after enqueue: torrentID=%d fileIndex=%d err=%v",
			torrentID, fileIndex, err)
		// Message is already in the queue; aggregate will catch up via recompute.
	}

	if err := p.recomputeCloudStatus(torrentID); err != nil {
		log.Warnf("recomputeCloudStatus after enqueue failed: torrentID=%d err=%v", torrentID, err)
	}

	log.Infof("queued cloud upload: torrentID=%d fileIndex=%d cloudPath=%s", torrentID, fileIndex, cloudPath)
}

// createTranscodedFileWithRetry inserts a new TorrentFile row for a transcoded
// output. The Index is computed as MAX(index)+1 inside a transaction; if the
// DB-level unique constraint rejects the insert (because a concurrent completion
// just claimed the same index), we re-query MAX and try again. Capped retries.
func createTranscodedFileWithRetry(db *gorm.DB, newFile *torrentModel.TorrentFile, torrentID int64, outputPath string, outputSize int64, parentPath string) error {
	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := db.Transaction(func(tx *gorm.DB) error {
			var maxIdx struct{ V *int }
			if err := tx.Raw("SELECT MAX(`index`) AS v FROM torrent_files WHERE torrent_id = ?", torrentID).Scan(&maxIdx).Error; err != nil {
				return fmt.Errorf("max(index): %w", err)
			}
			next := 0
			if maxIdx.V != nil {
				next = *maxIdx.V + 1
			}
			*newFile = torrentModel.TorrentFile{
				TorrentID:    torrentID,
				Index:        next,
				Path:         outputPath,
				Size:         outputSize,
				IsSelected:   true,
				IsShareable:  false,
				IsStreamable: true,
				Type:         "video",
				Source:       "transcoded",
				ParentPath:   parentPath,
			}
			return tx.Create(newFile).Error
		})
		if err == nil {
			return nil
		}
		lastErr = err
		// Unique-violation error strings vary across drivers; match the common
		// substrings and retry. Anything else is a real failure.
		es := err.Error()
		if !strings.Contains(es, "Duplicate") && !strings.Contains(es, "UNIQUE") && !strings.Contains(es, "uniq_torrent_file_torrent_id_index") {
			return err
		}
	}
	return fmt.Errorf("createTranscodedFileWithRetry exhausted %d attempts: %w", maxAttempts, lastErr)
}

// recomputeTranscodeStatus aggregates per-file transcode status into the torrent record.
func (p *WorkerEventProcessor) recomputeTranscodeStatus(torrentID int64) error {
	type Row struct {
		Status int
		Count  int
	}
	var rows []Row
	if err := p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("transcode_status as status, count(*) as count").
		Where("torrent_id = ? AND (source = '' OR source = 'original')", torrentID).
		Group("transcode_status").
		Scan(&rows).Error; err != nil {
		return err
	}
	var pending, processing, completed, failed int
	for _, r := range rows {
		switch r.Status {
		case torrentModel.TranscodeStatusPending:
			pending += r.Count
		case torrentModel.TranscodeStatusProcessing:
			processing += r.Count
		case torrentModel.TranscodeStatusCompleted:
			completed += r.Count
		case torrentModel.TranscodeStatusFailed:
			failed += r.Count
		}
	}
	updates := map[string]any{
		"transcode_status":   torrentModel.TranscodeStatusNone,
		"transcode_progress": 0,
		"transcoded_count":   completed,
	}
	switch {
	case processing > 0:
		updates["transcode_status"] = torrentModel.TranscodeStatusProcessing
	case pending > 0:
		updates["transcode_status"] = torrentModel.TranscodeStatusPending
	case failed > 0 && completed == 0:
		updates["transcode_status"] = torrentModel.TranscodeStatusFailed
	case completed > 0 || (pending == 0 && processing == 0 && failed == 0):
		updates["transcode_status"] = torrentModel.TranscodeStatusCompleted
		updates["transcode_progress"] = 100
	}
	return p.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		Updates(updates).Error
}

// recomputeTranscodeProgress mixes current-file progress into the aggregate.
func (p *WorkerEventProcessor) recomputeTranscodeProgress(torrentID int64, currentProgress int) error {
	type Row struct {
		Status int
		Count  int
	}
	var rows []Row
	if err := p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("transcode_status as status, count(*) as count").
		Where("torrent_id = ? AND (source = '' OR source = 'original')", torrentID).
		Group("transcode_status").
		Scan(&rows).Error; err != nil {
		return err
	}
	var totalWeighted, totalCount int
	for _, r := range rows {
		switch r.Status {
		case torrentModel.TranscodeStatusCompleted:
			totalWeighted += 100 * r.Count
			totalCount += r.Count
		case torrentModel.TranscodeStatusProcessing, torrentModel.TranscodeStatusPending:
			totalCount += r.Count
		}
	}
	overall := 0
	if totalCount > 0 {
		overall = (totalWeighted + currentProgress) / totalCount
	}
	return p.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		Update("transcode_progress", overall).Error
}

// recomputeCloudStatus aggregates per-file cloud_upload_status into the torrent record.
func (p *WorkerEventProcessor) recomputeCloudStatus(torrentID int64) error {
	type Row struct {
		Status int
		Count  int
	}
	var rows []Row
	if err := p.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("cloud_upload_status as status, count(*) as count").
		Where("torrent_id = ?", torrentID).
		Group("cloud_upload_status").
		Scan(&rows).Error; err != nil {
		return err
	}
	var pending, uploading, completed, failed int
	var total int
	for _, r := range rows {
		if r.Status != torrentModel.CloudUploadStatusNone {
			total += r.Count
		}
		switch r.Status {
		case torrentModel.CloudUploadStatusPending:
			pending += r.Count
		case torrentModel.CloudUploadStatusUploading:
			uploading += r.Count
		case torrentModel.CloudUploadStatusCompleted:
			completed += r.Count
		case torrentModel.CloudUploadStatusFailed:
			failed += r.Count
		}
	}
	updates := map[string]any{
		"total_cloud_upload":    total,
		"cloud_uploaded_count":  completed,
		"cloud_upload_progress": 0,
		"cloud_upload_status":   torrentModel.CloudUploadStatusNone,
	}
	if total > 0 {
		updates["cloud_upload_progress"] = int(float64(completed) * 100 / float64(total))
	}
	switch {
	case uploading > 0:
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusUploading
	case pending > 0:
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusPending
	case failed > 0 && completed == 0:
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusFailed
	case completed > 0:
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusCompleted
	}
	return p.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentID).
		Updates(updates).Error
}

func guessContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".srt":
		return "application/x-subrip"
	case ".ass":
		return "text/x-ssa"
	case ".vtt":
		return "text/vtt"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func downloadStatusFromString(status string) int {
	switch status {
	case "downloading", "fetching_metadata":
		return torrentModel.StatusDownloading
	case "completed", "seeding":
		return torrentModel.StatusCompleted
	case "paused":
		return torrentModel.StatusPaused
	case "failed":
		return torrentModel.StatusFailed
	default:
		return torrentModel.StatusPending
	}
}
