// Package impl provides torrent service implementation
// Author: Done-0
// Created: 2026-01-22
package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"magnet2video/configs"
	"magnet2video/internal/cache"
	"magnet2video/internal/db"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/middleware/auth"
	torrentModel "magnet2video/internal/model/torrent"
	"magnet2video/internal/queue"
	"magnet2video/internal/torrent"
	"magnet2video/internal/torrent/replybus"
	torrentTypes "magnet2video/internal/torrent/types"
	"magnet2video/internal/types/consts"
	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/serve/service"
	"magnet2video/pkg/vo"
)

// TorrentServiceImpl torrent service implementation
type TorrentServiceImpl struct {
	config           *configs.Config
	loggerManager    logger.LoggerManager
	dbManager        db.DatabaseManager
	torrentManager   torrent.TorrentManager
	cacheManager     cache.CacheManager
	queueProducer    queue.Producer
	parseMagnetBus   *replybus.ParseMagnetBus
	transcodeChecker service.TranscodeChecker // Lazy-loaded to avoid circular dependency
}

// NewTorrentService creates torrent service implementation
func NewTorrentService(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	torrentManager torrent.TorrentManager,
	cacheManager cache.CacheManager,
	queueProducer queue.Producer,
	parseMagnetBus *replybus.ParseMagnetBus,
) *TorrentServiceImpl {
	instance := &TorrentServiceImpl{
		config:         config,
		loggerManager:  loggerManager,
		dbManager:      dbManager,
		torrentManager: torrentManager,
		cacheManager:   cacheManager,
		queueProducer:  queueProducer,
		parseMagnetBus: parseMagnetBus,
	}

	// In server mode the download engine runs on the worker; skip restore here.
	if config.AppConfig.Mode != configs.ModeServer {
		go instance.restoreTorrents()
	}

	return instance
}

// resolveDownloadDir returns the directory used for relative-path
// rewriting. Picks the live torrent client when available (mode=all),
// otherwise falls back to the configured value (mode=server has no
// torrent client at all).
func (ts *TorrentServiceImpl) resolveDownloadDir() string {
	if ts.torrentManager != nil {
		if cl := ts.torrentManager.Client(); cl != nil {
			if d := cl.GetDownloadDir(); d != "" {
				return d
			}
		}
	}
	if ts.config != nil {
		return ts.config.TorrentConfig.DownloadDir
	}
	return ""
}

// publishDownloadJob publishes a download control message to the download-jobs topic.
func (ts *TorrentServiceImpl) publishDownloadJob(ctx context.Context, job eventTypes.DownloadJob) error {
	if ts.queueProducer == nil {
		return errors.New("queue producer unavailable")
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return ts.queueProducer.Send(ctx, eventTypes.TopicDownloadJobs, nil, data)
}

// lookupWorkerID returns the worker that owns the on-disk state for the given
// info hash. Returns empty string when the torrent has not been claimed yet
// (newly created records, mode=all single-process deployments before the
// first progress event arrives, or rows pre-dating the worker_id column).
//
// Empty values are treated as "any worker may execute" by the worker filter.
// Non-empty values cause the worker handler to NACK + requeue any message
// that doesn't target it, so peer workers can pick it up.
func (ts *TorrentServiceImpl) lookupWorkerID(infoHash string) string {
	if ts.dbManager == nil || ts.dbManager.DB() == nil {
		return ""
	}
	var t torrentModel.Torrent
	if err := ts.dbManager.DB().
		Select("worker_id").
		Where("info_hash = ?", infoHash).
		First(&t).Error; err != nil {
		return ""
	}
	return t.WorkerID
}

// magnetURIFor reconstructs a magnet URI from infoHash + trackers.
func magnetURIFor(infoHash string, trackers []string) string {
	u := fmt.Sprintf("magnet:?xt=urn:btih:%s", infoHash)
	for _, t := range trackers {
		if t != "" {
			u += "&tr=" + t
		}
	}
	return u
}

// SetTranscodeChecker sets the transcode checker (called after all services are created)
func (ts *TorrentServiceImpl) SetTranscodeChecker(checker service.TranscodeChecker) {
	ts.transcodeChecker = checker

	// Check completed torrents that haven't been checked for transcoding yet
	go ts.checkPendingTranscodes()
}

// checkPendingTranscodes checks all completed torrents that need transcode detection
func (ts *TorrentServiceImpl) checkPendingTranscodes() {
	// Wait for system to stabilize
	time.Sleep(5 * time.Second)

	if ts.transcodeChecker == nil {
		return
	}

	ts.loggerManager.Logger().Info("Checking completed torrents for pending transcode...")

	var torrents []torrentModel.Torrent
	// Find completed torrents with transcode_status = 0 (not checked yet)
	err := ts.dbManager.DB().Where("deleted = ? AND status = ? AND transcode_status = ?",
		false, torrentModel.StatusCompleted, 0).Find(&torrents).Error

	if err != nil {
		ts.loggerManager.Logger().Errorf("Failed to find pending transcode torrents: %v", err)
		return
	}

	if len(torrents) == 0 {
		ts.loggerManager.Logger().Info("No pending transcode torrents found")
		return
	}

	ts.loggerManager.Logger().Infof("Found %d torrents pending transcode check", len(torrents))

	for _, t := range torrents {
		ts.loggerManager.Logger().Infof("Triggering transcode check for: %s", t.Name)
		ts.transcodeChecker.TriggerTranscodeCheck(t.ID)
		// Small delay between checks to avoid overwhelming the system
		time.Sleep(2 * time.Second)
	}
}

// ParseMagnet parses a magnet URI and returns available files.
//
// Server-side this never touches the local torrent client. Even in mode=all
// where the worker is co-located in-process, we route through the MQ so the
// code path is identical to mode=server. The worker's ParseMagnetHandler
// resolves the magnet and publishes the result on parse-magnet-results;
// this goroutine waits on the bus channel until the response or context
// deadline arrives.
func (ts *TorrentServiceImpl) ParseMagnet(c *gin.Context, req *dto.ParseMagnetRequest) (*vo.ParseMagnetResponse, error) {
	result, err := ts.parseViaWorker(c.Request.Context(), req.MagnetURI, req.Trackers)
	if err != nil {
		return nil, err
	}

	files := make([]vo.TorrentFileInfo, len(result.Files))
	for i, f := range result.Files {
		files[i] = vo.TorrentFileInfo{
			Index:        i,
			Path:         f.Path,
			Size:         f.Size,
			SizeReadable: formatSize(f.Size),
			IsStreamable: f.IsStreamable,
		}
	}

	return &vo.ParseMagnetResponse{
		InfoHash:  result.InfoHash,
		Name:      result.Name,
		TotalSize: result.TotalSize,
		Files:     files,
	}, nil
}

// parseViaWorker dispatches a parse-magnet job to the worker pool and waits
// for the reply. On success the result is cached so StartDownload can avoid
// a second round-trip.
func (ts *TorrentServiceImpl) parseViaWorker(parentCtx context.Context, magnetURI string, trackers []string) (*torrentTypes.ParseMagnetResult, error) {
	if ts.parseMagnetBus == nil || ts.queueProducer == nil {
		return nil, errors.New("parse-magnet bus not configured")
	}

	// 120s overall budget, leaving the worker its own 90s parse timeout.
	ctx, cancel := context.WithTimeout(parentCtx, 120*time.Second)
	defer cancel()

	requestID := eventTypes.GenerateEventID()
	resultCh := ts.parseMagnetBus.Register(requestID)

	jobBytes, err := json.Marshal(torrentTypes.ParseMagnetRequest{
		RequestID: requestID,
		MagnetURI: magnetURI,
		Trackers:  trackers,
	})
	if err != nil {
		ts.parseMagnetBus.Cancel(requestID)
		return nil, fmt.Errorf("marshal parse-magnet request: %w", err)
	}
	if err := ts.queueProducer.Send(ctx, torrentTypes.TopicParseMagnetJobs, nil, jobBytes); err != nil {
		ts.parseMagnetBus.Cancel(requestID)
		ts.loggerManager.Logger().Errorf("publish parse-magnet job: %v", err)
		return nil, fmt.Errorf("publish parse-magnet job: %w", err)
	}

	result, err := ts.parseMagnetBus.Wait(ctx, requestID, resultCh)
	if err != nil {
		ts.loggerManager.Logger().Warnf("parse-magnet timeout: requestID=%s magnet=%s", requestID, magnetURI)
		return nil, fmt.Errorf("parse magnet: %w", err)
	}
	if result.ErrorMsg != "" {
		ts.loggerManager.Logger().Errorf("parse-magnet worker error: requestID=%s err=%s", requestID, result.ErrorMsg)
		return nil, errors.New(result.ErrorMsg)
	}

	// Cache the result so StartDownload (and other parse-then-act flows)
	// can avoid bouncing the worker again. Best-effort.
	if ts.cacheManager != nil && result.InfoHash != "" {
		cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cacheCancel()
		if cacheErr := ts.cacheManager.Set(cacheCtx, cache.ParsedMagnetKey(result.InfoHash), result, cache.TTLParsedMagnet); cacheErr != nil {
			ts.loggerManager.Logger().Warnf("cache parse-magnet result: %v", cacheErr)
		}
	}

	return result, nil
}

// fetchTorrentMetadata returns the cached parse-magnet result for an
// infoHash, falling back to a fresh worker round-trip if the cache has
// expired.
func (ts *TorrentServiceImpl) fetchTorrentMetadata(ctx context.Context, infoHash, magnetURI string, trackers []string) (*torrentTypes.ParseMagnetResult, error) {
	if ts.cacheManager != nil && infoHash != "" {
		var cached torrentTypes.ParseMagnetResult
		if err := ts.cacheManager.Get(ctx, cache.ParsedMagnetKey(infoHash), &cached); err == nil {
			return &cached, nil
		}
	}
	if magnetURI == "" {
		magnetURI = magnetURIFor(infoHash, trackers)
	}
	return ts.parseViaWorker(ctx, magnetURI, trackers)
}

// StartDownload kicks off a BT download. Since PR2 the server NEVER drives
// the local torrent client directly even in mode=all — it always publishes
// a download-jobs message and the worker (in-process for mode=all, remote
// for mode=server) handles it. File metadata comes from the parse cache or,
// on a miss, a fresh parse round-trip.
func (ts *TorrentServiceImpl) StartDownload(c *gin.Context, req *dto.StartDownloadRequest) (*vo.StartDownloadResponse, error) {
	ctx := c.Request.Context()

	// Get the torrent metadata (file list, sizes, name) — from cache when
	// possible, falling back to a worker round-trip. We don't hit the local
	// torrent client because in mode=server it doesn't have the metadata
	// loaded, and in mode=all the worker is the source of truth anyway.
	torrentInfo, err := ts.fetchTorrentMetadata(ctx, req.InfoHash, "", req.Trackers)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to fetch torrent metadata: %v", err)
		return nil, err
	}

	// Build selected files map
	selectedMap := make(map[int]bool)
	for _, idx := range req.SelectedFiles {
		selectedMap[idx] = true
	}

	// Convert to model files with selection status
	files := make([]torrentModel.TorrentFile, len(torrentInfo.Files))
	for i, f := range torrentInfo.Files {
		fileType := torrentModel.DetectFileType(f.Path)
		files[i] = torrentModel.TorrentFile{
			Index:        i,
			Path:         f.Path,
			Size:         f.Size,
			IsSelected:   selectedMap[i],
			IsShareable:  false,
			IsStreamable: f.IsStreamable,
			Type:         fileType,
			Source:       "original",
			ParentPath:   "",
		}
	}

	// Check if torrent already exists in database
	var existingTorrent torrentModel.Torrent
	result := ts.dbManager.DB().Where("info_hash = ?", req.InfoHash).First(&existingTorrent)

	// Get current user ID from context
	userID := auth.GetUserID(c)

	// Resolve a download path string to persist on the torrent record. In
	// mode=all the local torrent client is the source of truth; in
	// mode=server we fall back to whatever the config says. This is purely
	// for display / "where does this torrent live" — actual writes happen
	// on the worker.
	downloadPath := ts.resolveDownloadDir()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new torrent record
		newTorrent := &torrentModel.Torrent{
			InfoHash:     req.InfoHash,
			Name:         torrentInfo.Name,
			TotalSize:    torrentInfo.TotalSize,
			Files:        files,
			DownloadPath: downloadPath,
			Status:       torrentModel.StatusDownloading,
			Progress:     0,
			Trackers:     torrentModel.StringSlice(req.Trackers),
			CreatorID:    userID,
			Visibility:   torrentModel.VisibilityPrivate,
		}

		if err := ts.dbManager.DB().Create(newTorrent).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to create torrent record: %v", err)
			return nil, err
		}
	} else if result.Error == nil {
		// Update existing record - may be a soft-deleted record being re-added
		// Update the record: reset deleted flag, update status, files and user
		updates := map[string]any{
			"status":                torrentModel.StatusDownloading,
			"progress":              0,
			"deleted":               false,
			"name":                  torrentInfo.Name,
			"total_size":            torrentInfo.TotalSize,
			"transcode_status":      torrentModel.TranscodeStatusNone,
			"transcode_progress":    0,
			"transcoded_count":      0,
			"total_transcode":       0,
			"cloud_upload_status":   torrentModel.CloudUploadStatusNone,
			"cloud_upload_progress": 0,
			"cloud_uploaded_count":  0,
			"total_cloud_upload":    0,
		}

		// Update creator_id only if it was 0 or if this user is adding it
		if existingTorrent.CreatorID == 0 {
			updates["creator_id"] = userID
		}

		if err := ts.dbManager.DB().Model(&existingTorrent).Updates(updates).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to update torrent record: %v", err)
			return nil, err
		}

		// Delete old files and insert new ones because torrent_id cannot be null
		if err := ts.dbManager.DB().Where("torrent_id = ?", existingTorrent.ID).Delete(&torrentModel.TorrentFile{}).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to delete old torrent files: %v", err)
			return nil, err
		}

		for i := range files {
			files[i].TorrentID = existingTorrent.ID
		}
		if err := ts.dbManager.DB().Create(&files).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to create new torrent files: %v", err)
			return nil, err
		}
	}

	// 数据库更新成功后失效缓存
	ts.invalidateTorrentListCache(userID)

	// Always dispatch the download command via MQ. In mode=all the worker is
	// in the same process so the GoChannel queue delivers it locally; in
	// mode=server it goes out over RabbitMQ. Either way the server never
	// drives the torrent client directly.
	if err := ts.publishDownloadJob(ctx, eventTypes.DownloadJob{
		Action:        eventTypes.DownloadActionStart,
		InfoHash:      req.InfoHash,
		MagnetURI:     magnetURIFor(req.InfoHash, req.Trackers),
		SelectedFiles: req.SelectedFiles,
		Trackers:      req.Trackers,
	}); err != nil {
		ts.loggerManager.Logger().Errorf("failed to publish download job: %v", err)
		return nil, err
	}

	return &vo.StartDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download started successfully",
	}, nil
}

// GetProgress returns download progress for a torrent. Always reads from
// the heartbeat-fed cache + DB — the worker is the source of truth and the
// server should never query the local torrent client even in mode=all.
func (ts *TorrentServiceImpl) GetProgress(c *gin.Context, req *dto.GetProgressRequest) (*vo.DownloadProgressResponse, error) {
	if progress, ok := ts.cachedDownloadProgress(c.Request.Context(), req.InfoHash); ok {
		return &vo.DownloadProgressResponse{
			InfoHash:              progress.InfoHash,
			Name:                  progress.Name,
			TotalSize:             progress.TotalSize,
			DownloadedSize:        progress.DownloadedSize,
			Progress:              progress.Progress,
			Status:                progress.Status,
			Peers:                 progress.Peers,
			Seeds:                 progress.Seeds,
			DownloadSpeed:         progress.DownloadSpeed,
			DownloadSpeedReadable: formatSpeed(progress.DownloadSpeed),
		}, nil
	}

	var t torrentModel.Torrent
	if err := ts.dbManager.DB().Where("info_hash = ? AND deleted = ?", req.InfoHash, false).First(&t).Error; err != nil {
		return nil, err
	}
	return &vo.DownloadProgressResponse{
		InfoHash:              t.InfoHash,
		Name:                  t.Name,
		TotalSize:             t.TotalSize,
		Progress:              t.Progress,
		Status:                statusStringFromModel(t.Status),
		DownloadSpeed:         0,
		DownloadSpeedReadable: formatSpeed(0),
	}, nil
}

// PauseDownload pauses a torrent download. Always dispatched via MQ — the
// server never drives the torrent client directly.
func (ts *TorrentServiceImpl) PauseDownload(c *gin.Context, req *dto.PauseDownloadRequest) (*vo.PauseDownloadResponse, error) {
	if err := ts.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:         eventTypes.DownloadActionPause,
		InfoHash:       req.InfoHash,
		TargetWorkerID: ts.lookupWorkerID(req.InfoHash),
	}); err != nil {
		ts.loggerManager.Logger().Errorf("publish pause job: %v", err)
	}

	// Update database
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusPaused).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}
	ts.deleteCachedDownloadProgress(c.Request.Context(), req.InfoHash)
	ts.invalidateTorrentListCache(auth.GetUserID(c))

	return &vo.PauseDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download paused successfully",
	}, nil
}

// ResumeDownload resumes a paused torrent download. Always dispatched via MQ.
func (ts *TorrentServiceImpl) ResumeDownload(c *gin.Context, req *dto.ResumeDownloadRequest) (*vo.ResumeDownloadResponse, error) {
	if err := ts.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:         eventTypes.DownloadActionResume,
		InfoHash:       req.InfoHash,
		SelectedFiles:  req.SelectedFiles,
		TargetWorkerID: ts.lookupWorkerID(req.InfoHash),
	}); err != nil {
		ts.loggerManager.Logger().Errorf("publish resume job: %v", err)
	}

	// Update database
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusDownloading).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}
	ts.invalidateTorrentListCache(auth.GetUserID(c))

	return &vo.ResumeDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download resumed successfully",
	}, nil
}

// RemoveTorrent removes a torrent from the system. Always dispatched via MQ.
func (ts *TorrentServiceImpl) RemoveTorrent(c *gin.Context, req *dto.RemoveTorrentRequest) (*vo.RemoveTorrentResponse, error) {
	if err := ts.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:         eventTypes.DownloadActionRemove,
		InfoHash:       req.InfoHash,
		DeleteFiles:    req.DeleteFiles,
		TargetWorkerID: ts.lookupWorkerID(req.InfoHash),
	}); err != nil {
		ts.loggerManager.Logger().Errorf("publish remove job: %v", err)
	}

	// Remove from database (soft delete)
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("deleted", true).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to delete torrent from database: %v", err)
	}
	ts.deleteCachedDownloadProgress(c.Request.Context(), req.InfoHash)

	// 删除后失效缓存
	userID := auth.GetUserID(c)
	ts.invalidateTorrentListCache(userID)

	return &vo.RemoveTorrentResponse{
		InfoHash: req.InfoHash,
		Message:  "Torrent removed successfully",
	}, nil
}

// StopSeed drops the torrent from the swarm on the worker side but keeps the
// local files on disk. The DB row stays with status=StatusSeedingStopped, which
// fresh-boot recovery (RedispatchActiveTorrents) skips by design.
func (ts *TorrentServiceImpl) StopSeed(c *gin.Context, req *dto.StopSeedRequest) (*vo.StopSeedResponse, error) {
	if err := ts.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:         eventTypes.DownloadActionStopSeed,
		InfoHash:       req.InfoHash,
		TargetWorkerID: ts.lookupWorkerID(req.InfoHash),
	}); err != nil {
		ts.loggerManager.Logger().Errorf("publish stop_seed job: %v", err)
	}

	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusSeedingStopped).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}
	ts.deleteCachedDownloadProgress(c.Request.Context(), req.InfoHash)
	ts.invalidateTorrentListCache(auth.GetUserID(c))

	return &vo.StopSeedResponse{
		InfoHash: req.InfoHash,
		Message:  "Seeding stopped successfully",
	}, nil
}

// ResumeSeed re-adds the torrent to the worker's swarm. Selected files come
// from the DB so we don't depend on the caller remembering them.
func (ts *TorrentServiceImpl) ResumeSeed(c *gin.Context, req *dto.ResumeSeedRequest) (*vo.ResumeSeedResponse, error) {
	var t torrentModel.Torrent
	if err := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND deleted = ?", req.InfoHash, false).
		First(&t).Error; err != nil {
		return nil, err
	}

	var selectedFiles []int
	for _, f := range t.Files {
		if f.IsSelected {
			selectedFiles = append(selectedFiles, f.Index)
		}
	}

	if err := ts.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:         eventTypes.DownloadActionResumeSeed,
		InfoHash:       req.InfoHash,
		SelectedFiles:  selectedFiles,
		Trackers:       t.Trackers,
		TargetWorkerID: t.WorkerID,
	}); err != nil {
		ts.loggerManager.Logger().Errorf("publish resume_seed job: %v", err)
	}

	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusCompleted).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}
	ts.invalidateTorrentListCache(auth.GetUserID(c))

	return &vo.ResumeSeedResponse{
		InfoHash: req.InfoHash,
		Message:  "Seeding resumed successfully",
	}, nil
}

// ListTorrents 获取当前用户的 torrent 列表
// 如果用户已认证，只返回该用户的 torrents
// 如果未认证，返回空列表
// 使用 Redis 缓存数据库查询，实时统计数据始终是最新的
func (ts *TorrentServiceImpl) ListTorrents(c *gin.Context) (*vo.TorrentListResponse, error) {
	userID := auth.GetUserID(c)

	// 未认证，返回空列表
	if userID == 0 {
		return &vo.TorrentListResponse{
			Torrents: []vo.TorrentListItem{},
			Total:    0,
		}, nil
	}

	ctx := c.Request.Context()
	cacheKey := cache.TorrentListKey(userID)

	// 优先从缓存获取
	var cachedTorrents []torrentModel.Torrent
	err := ts.cacheManager.Get(ctx, cacheKey, &cachedTorrents)

	if err != nil {
		if err != cache.ErrCacheMiss {
			ts.loggerManager.Logger().Warnf("Cache get error: %v", err)
		}

		// 缓存未命中或出错，从数据库加载
		query := ts.dbManager.DB().
			Preload("Files", func(db *gorm.DB) *gorm.DB {
				return db.Select("id, torrent_id, `index`, path, cloud_upload_status, cloud_upload_error").Order("`index` asc")
			}).
			Where("deleted = ? AND creator_id = ?", false, userID)
		if err := query.Order("created_at DESC").Find(&cachedTorrents).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to list torrents: %v", err)
			return nil, err
		}

		// 异步存储到缓存
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if setErr := ts.cacheManager.Set(cacheCtx, cacheKey, cachedTorrents, cache.TTLTorrentList); setErr != nil {
				ts.loggerManager.Logger().Warnf("Failed to cache torrent list: %v", setErr)
			}
		}()
	}

	// 转换为列表项，包含实时统计数据（始终是最新的，不缓存）
	items := ts.torrentListToItems(ctx, cachedTorrents)

	return &vo.TorrentListResponse{
		Torrents: items,
		Total:    len(items),
	}, nil
}

// ListPublicTorrents 获取所有公开的 torrent 列表
// 未登录：只返回 visibility=2 的种子
// 已登录：返回 visibility IN (1, 2) 的种子
// 使用 Redis 缓存数据库查询，实时统计数据始终是最新的
func (ts *TorrentServiceImpl) ListPublicTorrents(c *gin.Context) (*vo.TorrentListResponse, error) {
	ctx := c.Request.Context()
	userID := auth.GetUserID(c)
	isLoggedIn := userID > 0

	// Use different cache keys for logged-in vs anonymous
	var cacheKey string
	if isLoggedIn {
		cacheKey = cache.PublicTorrentListKey() + ":internal"
	} else {
		cacheKey = cache.PublicTorrentListKey()
	}

	// 优先从缓存获取
	var cachedTorrents []torrentModel.Torrent
	err := ts.cacheManager.Get(ctx, cacheKey, &cachedTorrents)

	if err != nil {
		if err != cache.ErrCacheMiss {
			ts.loggerManager.Logger().Warnf("Cache get error: %v", err)
		}

		// 缓存未命中或出错，从数据库加载
		query := ts.dbManager.DB().
			Preload("Files", func(db *gorm.DB) *gorm.DB {
				return db.Select("id, torrent_id, `index`, path, cloud_upload_status, cloud_upload_error").Order("`index` asc")
			}).
			Where("deleted = ?", false)
		if isLoggedIn {
			query = query.Where("visibility IN ?", []int{torrentModel.VisibilityInternal, torrentModel.VisibilityPublic})
		} else {
			query = query.Where("visibility = ?", torrentModel.VisibilityPublic)
		}

		if err := query.Order("created_at DESC").Find(&cachedTorrents).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to list public torrents: %v", err)
			return nil, err
		}

		// 异步存储到缓存
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if setErr := ts.cacheManager.Set(cacheCtx, cacheKey, cachedTorrents, cache.TTLPublicList); setErr != nil {
				ts.loggerManager.Logger().Warnf("Failed to cache public torrent list: %v", setErr)
			}
		}()
	}

	// 转换为列表项，包含实时统计数据（始终是最新的，不缓存）
	items := ts.torrentListToItems(ctx, cachedTorrents)

	return &vo.TorrentListResponse{
		Torrents: items,
		Total:    len(items),
	}, nil
}

// torrentListToItems converts a list of torrent models to list items with real-time stats
func (ts *TorrentServiceImpl) torrentListToItems(ctx context.Context, torrents []torrentModel.Torrent) []vo.TorrentListItem {
	items := make([]vo.TorrentListItem, len(torrents))
	for i, t := range torrents {
		items[i] = vo.TorrentListItem{
			InfoHash:           t.InfoHash,
			Name:               t.Name,
			TotalSize:          t.TotalSize,
			Progress:           t.Progress,
			Status:             t.Status,
			PosterPath:         t.PosterPath,
			ImdbID:             t.ImdbID,
			CreatedAt:          t.CreatedAt,
			IsPublic:           t.Visibility >= torrentModel.VisibilityPublic,
			Visibility:         t.Visibility,
			TranscodeStatus:    t.TranscodeStatus,
			TranscodeProgress:  t.TranscodeProgress,
			TranscodedCount:    t.TranscodedCount,
			TotalTranscode:     t.TotalTranscode,
			CloudUploadStatus:  t.CloudUploadStatus,
			CloudUploadedCount: t.CloudUploadedCount,
			TotalCloudUpload:   t.TotalCloudUpload,
			LocalDeleted:       t.LocalDeleted,
		}

		// Populate per-file cloud upload info when there are cloud uploads
		if t.TotalCloudUpload > 0 && len(t.Files) > 0 {
			cloudFiles := make([]vo.CloudFileInfo, 0, len(t.Files))
			for _, f := range t.Files {
				if f.CloudUploadStatus != torrentModel.CloudUploadStatusNone {
					cloudFiles = append(cloudFiles, vo.CloudFileInfo{
						FileIndex:         f.Index,
						FileName:          filepath.Base(f.Path),
						CloudUploadStatus: f.CloudUploadStatus,
						CloudUploadError:  f.CloudUploadError,
					})
				}
			}
			items[i].CloudFiles = cloudFiles
		}

		if progress, ok := ts.cachedDownloadProgress(ctx, t.InfoHash); ok {
			applyCachedProgress(&items[i], progress)
			continue
		}
		// No cached progress: fall back to whatever DB already says. The
		// server intentionally does not query the local torrent client even
		// in mode=all — the worker publishes progress via the heartbeat /
		// download-progress topics every ~2s, which is fresh enough.
	}
	return items
}

func (ts *TorrentServiceImpl) cachedDownloadProgress(ctx context.Context, infoHash string) (eventTypes.DownloadProgressPayload, bool) {
	var progress eventTypes.DownloadProgressPayload
	if ts.cacheManager == nil || infoHash == "" {
		return progress, false
	}
	if err := ts.cacheManager.Get(ctx, cache.TorrentProgressKey(infoHash), &progress); err != nil {
		return progress, false
	}
	return progress, true
}

func (ts *TorrentServiceImpl) deleteCachedDownloadProgress(ctx context.Context, infoHash string) {
	if ts.cacheManager == nil || infoHash == "" {
		return
	}
	if err := ts.cacheManager.Delete(ctx, cache.TorrentProgressKey(infoHash)); err != nil {
		ts.loggerManager.Logger().Warnf("failed to delete cached download progress: %v", err)
	}
}

func applyCachedProgress(item *vo.TorrentListItem, progress eventTypes.DownloadProgressPayload) {
	item.Progress = progress.Progress
	item.Status = getStatusFromString(progress.Status)
	item.DownloadSpeed = progress.DownloadSpeed
	item.DownloadSpeedReadable = formatSpeed(progress.DownloadSpeed)
	if item.TotalSize == 0 {
		item.TotalSize = progress.TotalSize
	}
}

// GetTorrentDetail gets detailed information about a torrent
// Returns a flat file list stored in the torrent record
func (ts *TorrentServiceImpl) GetTorrentDetail(c *gin.Context, infoHash string) (*vo.TorrentDetailResponse, error) {
	var t torrentModel.Torrent

	if err := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND deleted = ?", infoHash, false).
		First(&t).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to get torrent detail: %v", err)
		return nil, err
	}

	downloadDir := ts.resolveDownloadDir()

	// Build flat file list directly from stored files
	allFiles := make([]vo.TorrentFileInfo, len(t.Files))
	pathToIndex := make(map[string]int)
	for i, f := range t.Files {
		fileType := f.Type
		if fileType == "" {
			fileType = torrentModel.DetectFileType(f.Path)
		}
		source := f.Source
		if source == "" {
			source = "original"
		}
		relPath := toRelativePath(f.Path, downloadDir)
		parentPath := f.ParentPath
		if parentPath != "" {
			parentPath = toRelativePath(parentPath, downloadDir)
		}
		allFiles[i] = vo.TorrentFileInfo{
			Index:           i,
			Path:            relPath,
			Size:            f.Size,
			SizeReadable:    formatSize(f.Size),
			Type:            fileType,
			Source:          source,
			ParentPath:      parentPath,
			OriginalIndex:   -1,
			IsStreamable:    f.IsStreamable,
			TranscodeStatus: f.TranscodeStatus,
			Language:        f.Language,
			LanguageName:    f.LanguageName,
			Title:           f.Title,
			CloudPath:       f.CloudPath,
			CloudStatus:     f.CloudUploadStatus,
		}
		if source == "original" || f.ParentPath == "" {
			pathToIndex[f.Path] = i
		}
	}
	for i := range allFiles {
		if t.Files[i].ParentPath == "" {
			continue
		}
		if idx, ok := pathToIndex[t.Files[i].ParentPath]; ok {
			allFiles[i].OriginalIndex = idx
		}
	}

	return &vo.TorrentDetailResponse{
		InfoHash:     t.InfoHash,
		Name:         t.Name,
		TotalSize:    t.TotalSize,
		Files:        allFiles,
		PosterPath:   t.PosterPath,
		ImdbID:       t.ImdbID,
		DownloadPath: t.DownloadPath,
		Status:       t.Status,
		Progress:     t.Progress,
		CreatedAt:    t.CreatedAt,
		IsPublic:     t.Visibility >= torrentModel.VisibilityPublic,
		Visibility:   t.Visibility,
	}, nil
}

// GetFilePath returns the file path for serving. Only valid in mode=all
// where the torrent client is in-process; mode=server returns an error.
func (ts *TorrentServiceImpl) GetFilePath(c *gin.Context, infoHash string, filePath string) (string, error) {
	if ts.torrentManager == nil {
		return "", errors.New("torrent client not available in this mode")
	}
	client := ts.torrentManager.Client()
	if client == nil {
		return "", errors.New("torrent client unavailable")
	}
	return client.GetFilePath(infoHash, filePath)
}

// GetFileStream returns the file stream for serving. Only valid in mode=all.
func (ts *TorrentServiceImpl) GetFileStream(c *gin.Context, infoHash string, filePath string) (io.ReadSeeker, *vo.TorrentFileInfo, error) {
	if ts.torrentManager == nil {
		return nil, nil, errors.New("torrent client not available in this mode")
	}
	client := ts.torrentManager.Client()
	if client == nil {
		return nil, nil, errors.New("torrent client unavailable")
	}
	reader, info, err := client.GetFileReader(infoHash, filePath)
	if err != nil {
		return nil, nil, err
	}

	fileInfo := &vo.TorrentFileInfo{
		Path:         info.Path,
		Size:         info.Size,
		SizeReadable: formatSize(info.Size),
		IsStreamable: info.IsStreamable,
	}

	return reader, fileInfo, nil
}

// GetDownloadDir returns the download directory path. Falls back to the
// configured value when the torrent client is not in this process.
func (ts *TorrentServiceImpl) GetDownloadDir() string {
	return ts.resolveDownloadDir()
}

// SetPosterFromFile sets poster from an existing torrent file
func (ts *TorrentServiceImpl) SetPosterFromFile(c *gin.Context, req *dto.SetPosterRequest) (*vo.PosterResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}
	if req == nil || req.InfoHash == "" || req.FileIndex == nil {
		return nil, errors.New("invalid request")
	}
	fileIndex := *req.FileIndex
	if fileIndex < 0 {
		return nil, errors.New("file_index out of range")
	}

	var torrentRecord torrentModel.Torrent
	result := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", req.InfoHash, userID, false).
		First(&torrentRecord)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("torrent not found or not owned by you")
	}
	if result.Error != nil {
		ts.loggerManager.Logger().Errorf("failed to find torrent: %v", result.Error)
		return nil, result.Error
	}

	if fileIndex >= len(torrentRecord.Files) {
		return nil, errors.New("file_index out of range")
	}

	file := torrentRecord.Files[fileIndex]
	if !isPosterImage(file.Path) {
		return nil, errors.New("poster file must be an image")
	}

	downloadDir := ts.resolveDownloadDir()
	relPath := toRelativePath(file.Path, downloadDir)
	if relPath == "" {
		return nil, errors.New("invalid poster file path")
	}

	posterPath := consts.PosterPathLocalPrefix + relPath
	if file.CloudUploadStatus == torrentModel.CloudUploadStatusCompleted && file.CloudPath != "" {
		posterPath = consts.PosterPathCloudPrefix + file.CloudPath
	}
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentRecord.ID).
		Update("poster_path", posterPath).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to update poster path: %v", err)
		return nil, err
	}

	ts.invalidateTorrentListCache(userID)

	return &vo.PosterResponse{
		InfoHash:   req.InfoHash,
		PosterPath: posterPath,
		Message:    "Poster updated successfully",
	}, nil
}

// UpdatePosterPath updates poster path directly (for uploads)
func (ts *TorrentServiceImpl) UpdatePosterPath(c *gin.Context, infoHash string, posterPath string) (*vo.PosterResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}
	if infoHash == "" || posterPath == "" {
		return nil, errors.New("invalid request")
	}

	result := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", infoHash, userID, false).
		Update("poster_path", posterPath)
	if result.Error != nil {
		ts.loggerManager.Logger().Errorf("failed to update poster path: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("torrent not found or not owned by you")
	}

	ts.invalidateTorrentListCache(userID)

	return &vo.PosterResponse{
		InfoHash:   infoHash,
		PosterPath: posterPath,
		Message:    "Poster updated successfully",
	}, nil
}

// BindIMDB binds an IMDB ID to a torrent
func (ts *TorrentServiceImpl) BindIMDB(c *gin.Context, req *dto.BindIMDBRequest) (*vo.BindIMDBResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}
	if req == nil || req.InfoHash == "" || req.ImdbID == "" {
		return nil, errors.New("invalid request")
	}

	result := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", req.InfoHash, userID, false).
		Update("imdb_id", req.ImdbID)
	if result.Error != nil {
		ts.loggerManager.Logger().Errorf("failed to bind IMDB ID: %v", result.Error)
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("torrent not found or not owned by you")
	}

	ts.invalidateTorrentListCache(userID)

	return &vo.BindIMDBResponse{
		InfoHash: req.InfoHash,
		ImdbID:   req.ImdbID,
		Message:  "IMDB ID bound successfully",
	}, nil
}

// Helper functions

// getFileSize returns the file size in bytes, or 0 if the file does not exist
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatSpeed(bytesPerSec int64) string {
	return formatSize(bytesPerSec) + "/s"
}

func isPosterImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

// toRelativePath converts an absolute path to a relative path based on the download directory.
// DB stores absolute paths (for cloud upload etc.), but API returns relative paths for URL construction.
func toRelativePath(absPath, downloadDir string) string {
	if absPath == "" {
		return ""
	}

	cleanAbs := filepath.Clean(absPath)
	cleanDownload := filepath.Clean(downloadDir)

	// Use strings.HasPrefix with separator to respect path boundaries
	prefix := cleanDownload + string(filepath.Separator)
	if strings.HasPrefix(cleanAbs, prefix) {
		return cleanAbs[len(prefix):]
	}
	// Exact match (path IS the download dir)
	if cleanAbs == cleanDownload {
		return "."
	}

	return absPath
}

func getStatusFromString(status string) int {
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

func statusStringFromModel(status int) string {
	switch status {
	case torrentModel.StatusDownloading:
		return "downloading"
	case torrentModel.StatusCompleted:
		return "completed"
	case torrentModel.StatusPaused:
		return "paused"
	case torrentModel.StatusFailed:
		return "failed"
	default:
		return "pending"
	}
}

// restoreTorrents is the startup-time wrapper around RedispatchActiveTorrents.
// Sleeps briefly so consumers have time to subscribe in mode=all, then
// performs the full re-dispatch.
func (ts *TorrentServiceImpl) restoreTorrents() {
	// Give the server some time to start up
	time.Sleep(2 * time.Second)
	ts.loggerManager.Logger().Info("Startup torrent restoration: re-dispatching active torrents")
	ts.RedispatchActiveTorrents(context.Background(), "startup")
}

// RedispatchActiveTorrents re-issues download-jobs(start) commands for every
// torrent whose DB status is Downloading or Completed (so seeding resumes
// after a worker restart). Intended for two callers:
//
//   - Startup: ensure newly-spawned workers pick up unfinished work even if
//     no event was missed (mode=all on first boot). reason="startup".
//   - Worker fresh-boot: invoked from the heartbeat consumer when a worker
//     publishes its first heartbeat. reason="fresh-boot:<workerID>". Only
//     torrents owned by that worker (or unclaimed ones) are redispatched —
//     without this filter, worker A rebooting would steal worker B's
//     in-flight torrents off the shared queue.
//
// Torrents with LocalDeleted=true are paused instead of restored; the user
// explicitly removed their disk copy and we don't want to silently re-grab
// terabytes of data.
//
// Each dispatched message carries TargetWorkerID = torrent.WorkerID so the
// worker filter on the consumer side enforces routing even if the queue
// briefly hands the message to the wrong consumer.
func (ts *TorrentServiceImpl) RedispatchActiveTorrents(ctx context.Context, reason string) {
	const freshBootPrefix = "fresh-boot:"
	bootingWorker := ""
	if strings.HasPrefix(reason, freshBootPrefix) {
		bootingWorker = strings.TrimPrefix(reason, freshBootPrefix)
	}

	q := ts.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("deleted = ? AND status IN ?", false, []int{
			torrentModel.StatusDownloading,
			torrentModel.StatusCompleted,
		})
	// Multi-worker safety: when a specific worker is reporting fresh-boot,
	// only redispatch its own torrents (plus the unclaimed ones, which any
	// worker may pick up). Without this, peer workers' active torrents get
	// re-issued and another consumer can race-pick them up.
	if bootingWorker != "" {
		q = q.Where("worker_id = ? OR worker_id = ''", bootingWorker)
	}

	var torrents []torrentModel.Torrent
	if err := q.Find(&torrents).Error; err != nil {
		ts.loggerManager.Logger().Errorf("redispatch (%s): load torrents failed: %v", reason, err)
		return
	}

	if len(torrents) == 0 {
		ts.loggerManager.Logger().Infof("redispatch (%s): nothing to do", reason)
		return
	}

	ts.loggerManager.Logger().Infof("redispatch (%s): %d torrent(s) to (re)dispatch", reason, len(torrents))

	dispatchCtx := ctx
	if dispatchCtx == nil {
		dispatchCtx = context.Background()
	}

	for _, t := range torrents {
		// Skip torrents whose local files have been deleted
		if t.LocalDeleted {
			ts.loggerManager.Logger().Infof("redispatch (%s): skip %s (local files deleted)", reason, t.Name)
			ts.dbManager.DB().Model(&torrentModel.Torrent{}).
				Where("id = ?", t.ID).
				Update("status", torrentModel.StatusPaused)
			continue
		}

		var selectedFiles []int
		for _, f := range t.Files {
			if f.IsSelected {
				selectedFiles = append(selectedFiles, f.Index)
			}
		}

		// Target the owning worker. Unclaimed torrents (worker_id empty)
		// fall through to "any worker" routing, which is the right answer
		// for fresh records or pre-migration data.
		targetWorkerID := t.WorkerID
		// During a fresh-boot redispatch, force unclaimed torrents to the
		// booting worker so they don't get scattered across peers.
		if targetWorkerID == "" && bootingWorker != "" {
			targetWorkerID = bootingWorker
		}

		if err := ts.publishDownloadJob(dispatchCtx, eventTypes.DownloadJob{
			Action:         eventTypes.DownloadActionStart,
			InfoHash:       t.InfoHash,
			MagnetURI:      magnetURIFor(t.InfoHash, t.Trackers),
			SelectedFiles:  selectedFiles,
			Trackers:       t.Trackers,
			TargetWorkerID: targetWorkerID,
		}); err != nil {
			ts.loggerManager.Logger().Errorf("redispatch (%s): publish failed for %s: %v", reason, t.InfoHash, err)
			continue
		}
		ts.loggerManager.Logger().Infof("redispatch (%s): dispatched %s (target=%s)", reason, t.Name, targetWorkerID)
	}
}

// invalidateTorrentListCache 在数据变更后失效 torrent 列表缓存
// 异步调用，不阻塞响应
func (ts *TorrentServiceImpl) invalidateTorrentListCache(userID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var keysToDelete []string

		// 失效用户的列表缓存
		if userID > 0 {
			keysToDelete = append(keysToDelete, cache.TorrentListKey(userID))
		}

		// 失效公共列表缓存（以防 torrent 可见性变更）
		keysToDelete = append(keysToDelete, cache.PublicTorrentListKey())
		keysToDelete = append(keysToDelete, cache.PublicTorrentListKey()+":internal")

		if err := ts.cacheManager.Delete(ctx, keysToDelete...); err != nil {
			ts.loggerManager.Logger().Warnf("Failed to invalidate cache: %v", err)
		} else {
			ts.loggerManager.Logger().Debugf("Cache invalidated for keys: %v", keysToDelete)
		}
	}()
}
