// Package impl provides torrent service implementation
// Author: Done-0
// Created: 2026-01-22
package impl

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	"github.com/Done-0/gin-scaffold/internal/middleware/auth"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	"github.com/Done-0/gin-scaffold/internal/torrent"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// TorrentServiceImpl torrent service implementation
type TorrentServiceImpl struct {
	loggerManager  logger.LoggerManager
	dbManager      db.DatabaseManager
	torrentManager torrent.TorrentManager
}

// NewTorrentService creates torrent service implementation
func NewTorrentService(
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	torrentManager torrent.TorrentManager,
) service.TorrentService {
	instance := &TorrentServiceImpl{
		loggerManager:  loggerManager,
		dbManager:      dbManager,
		torrentManager: torrentManager,
	}

	// Restore torrents in background
	go instance.restoreTorrents()

	return instance
}

// ParseMagnet parses a magnet URI and returns available files
func (ts *TorrentServiceImpl) ParseMagnet(c *gin.Context, req *dto.ParseMagnetRequest) (*vo.ParseMagnetResponse, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	client := ts.torrentManager.Client()

	torrentInfo, err := client.ParseMagnet(ctx, req.MagnetURI, req.Trackers)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to parse magnet URI: %v", err)
		return nil, err
	}

	// Convert to response format
	files := make([]vo.TorrentFileInfo, len(torrentInfo.Files))
	for i, file := range torrentInfo.Files {
		files[i] = vo.TorrentFileInfo{
			Index:        i,
			Path:         file.Path,
			Size:         file.Size,
			SizeReadable: formatSize(file.Size),
			IsStreamable: file.IsStreamable,
		}
	}

	return &vo.ParseMagnetResponse{
		InfoHash:  torrentInfo.InfoHash,
		Name:      torrentInfo.Name,
		TotalSize: torrentInfo.TotalSize,
		Files:     files,
	}, nil
}

// StartDownload starts downloading selected files
func (ts *TorrentServiceImpl) StartDownload(c *gin.Context, req *dto.StartDownloadRequest) (*vo.StartDownloadResponse, error) {
	ctx := c.Request.Context()
	client := ts.torrentManager.Client()

	// Start the download
	if err := client.StartDownload(ctx, req.InfoHash, req.SelectedFiles, req.Trackers); err != nil {
		ts.loggerManager.Logger().Errorf("failed to start download: %v", err)
		return nil, err
	}

	// Get torrent info including files
	torrentInfo, err := client.GetTorrentInfo(req.InfoHash)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to get torrent info: %v", err)
		return nil, err
	}

	// Build selected files map
	selectedMap := make(map[int]bool)
	for _, idx := range req.SelectedFiles {
		selectedMap[idx] = true
	}

	// Convert to model files with selection status
	files := make(torrentModel.TorrentFiles, len(torrentInfo.Files))
	for i, f := range torrentInfo.Files {
		files[i] = torrentModel.TorrentFile{
			Path:         f.Path,
			Size:         f.Size,
			IsSelected:   selectedMap[i],
			IsShareable:  false,
			IsStreamable: f.IsStreamable,
		}
	}

	// Check if torrent already exists in database
	var existingTorrent torrentModel.Torrent
	result := ts.dbManager.DB().Where("info_hash = ?", req.InfoHash).First(&existingTorrent)

	// Get current user ID from context
	userID := auth.GetUserID(c)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new torrent record
		newTorrent := &torrentModel.Torrent{
			InfoHash:     req.InfoHash,
			Name:         torrentInfo.Name,
			TotalSize:    torrentInfo.TotalSize,
			Files:        files,
			DownloadPath: "./download",
			Status:       torrentModel.StatusDownloading,
			Progress:     0,
			Trackers:     torrentModel.StringSlice(req.Trackers),
			CreatorID:    userID,
			IsPublic:     false, // Default to private
		}

		if err := ts.dbManager.DB().Create(newTorrent).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to create torrent record: %v", err)
			return nil, err
		}
	} else if result.Error == nil {
		// Update existing record - may be a soft-deleted record being re-added
		// Update the record: reset deleted flag, update status, files and user
		updates := map[string]any{
			"status":     torrentModel.StatusDownloading,
			"deleted":    false,
			"name":       torrentInfo.Name,
			"total_size": torrentInfo.TotalSize,
			"files":      files,
		}

		// Update creator_id only if it was 0 or if this user is adding it
		if existingTorrent.CreatorID == 0 {
			updates["creator_id"] = userID
		}

		if err := ts.dbManager.DB().Model(&existingTorrent).Updates(updates).Error; err != nil {
			ts.loggerManager.Logger().Errorf("failed to update torrent record: %v", err)
			return nil, err
		}
	}

	return &vo.StartDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download started successfully",
	}, nil
}

// GetProgress returns download progress for a torrent
func (ts *TorrentServiceImpl) GetProgress(c *gin.Context, req *dto.GetProgressRequest) (*vo.DownloadProgressResponse, error) {
	client := ts.torrentManager.Client()

	progress, err := client.GetProgress(req.InfoHash)
	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to get progress: %v", err)
		return nil, err
	}

	// Update database with progress
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Updates(map[string]any{
			"progress": progress.Progress,
			"status":   getStatusFromString(progress.Status),
		}).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent progress in database: %v", err)
	}

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

// PauseDownload pauses a torrent download
func (ts *TorrentServiceImpl) PauseDownload(c *gin.Context, req *dto.PauseDownloadRequest) (*vo.PauseDownloadResponse, error) {
	client := ts.torrentManager.Client()

	if err := client.PauseDownload(req.InfoHash); err != nil {
		ts.loggerManager.Logger().Errorf("failed to pause download: %v", err)
		return nil, err
	}

	// Update database
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusPaused).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}

	return &vo.PauseDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download paused successfully",
	}, nil
}

// ResumeDownload resumes a paused torrent download
func (ts *TorrentServiceImpl) ResumeDownload(c *gin.Context, req *dto.ResumeDownloadRequest) (*vo.ResumeDownloadResponse, error) {
	client := ts.torrentManager.Client()

	if err := client.ResumeDownload(req.InfoHash, req.SelectedFiles); err != nil {
		ts.loggerManager.Logger().Errorf("failed to resume download: %v", err)
		return nil, err
	}

	// Update database
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("status", torrentModel.StatusDownloading).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to update torrent status in database: %v", err)
	}

	return &vo.ResumeDownloadResponse{
		InfoHash: req.InfoHash,
		Message:  "Download resumed successfully",
	}, nil
}

// RemoveTorrent removes a torrent from the system
func (ts *TorrentServiceImpl) RemoveTorrent(c *gin.Context, req *dto.RemoveTorrentRequest) (*vo.RemoveTorrentResponse, error) {
	client := ts.torrentManager.Client()

	if err := client.RemoveTorrent(req.InfoHash, req.DeleteFiles); err != nil {
		ts.loggerManager.Logger().Errorf("failed to remove torrent: %v", err)
		return nil, err
	}

	// Remove from database (soft delete)
	if err := ts.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("info_hash = ?", req.InfoHash).
		Update("deleted", true).Error; err != nil {
		ts.loggerManager.Logger().Warnf("failed to delete torrent from database: %v", err)
	}

	return &vo.RemoveTorrentResponse{
		InfoHash: req.InfoHash,
		Message:  "Torrent removed successfully",
	}, nil
}

// ListTorrents lists torrents for the current user
// If user is authenticated, only returns their torrents
// If not authenticated, returns empty list
func (ts *TorrentServiceImpl) ListTorrents(c *gin.Context) (*vo.TorrentListResponse, error) {
	userID := auth.GetUserID(c)

	var torrents []torrentModel.Torrent

	query := ts.dbManager.DB().Where("deleted = ?", false)

	// If user is authenticated, filter by their user ID
	if userID > 0 {
		query = query.Where("creator_id = ?", userID)
	} else {
		// Not authenticated, return empty list
		return &vo.TorrentListResponse{
			Torrents: []vo.TorrentListItem{},
			Total:    0,
		}, nil
	}

	if err := query.Order("created_at DESC").Find(&torrents).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to list torrents: %v", err)
		return nil, err
	}

	items := ts.torrentListToItems(torrents)

	return &vo.TorrentListResponse{
		Torrents: items,
		Total:    len(items),
	}, nil
}

// ListPublicTorrents lists all public torrents
func (ts *TorrentServiceImpl) ListPublicTorrents(c *gin.Context) (*vo.TorrentListResponse, error) {
	var torrents []torrentModel.Torrent

	if err := ts.dbManager.DB().Where("deleted = ? AND is_public = ?", false, true).
		Order("created_at DESC").
		Find(&torrents).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to list public torrents: %v", err)
		return nil, err
	}

	items := ts.torrentListToItems(torrents)

	return &vo.TorrentListResponse{
		Torrents: items,
		Total:    len(items),
	}, nil
}

// torrentListToItems converts a list of torrent models to list items with real-time stats
func (ts *TorrentServiceImpl) torrentListToItems(torrents []torrentModel.Torrent) []vo.TorrentListItem {
	items := make([]vo.TorrentListItem, len(torrents))
	for i, t := range torrents {
		items[i] = vo.TorrentListItem{
			InfoHash:   t.InfoHash,
			Name:       t.Name,
			TotalSize:  t.TotalSize,
			Progress:   t.Progress,
			Status:     t.Status,
			PosterPath: t.PosterPath,
			CreatedAt:  t.CreatedAt,
			IsPublic:   t.IsPublic,
		}

		// Mix in real-time stats if downloading or seeding
		if t.Status == torrentModel.StatusDownloading || t.Status == torrentModel.StatusCompleted {
			if progress, err := ts.torrentManager.Client().GetProgress(t.InfoHash); err == nil {
				items[i].Progress = progress.Progress
				items[i].DownloadSpeed = progress.DownloadSpeed
				items[i].DownloadSpeedReadable = formatSpeed(progress.DownloadSpeed)
			}
		}
	}
	return items
}

// GetTorrentDetail gets detailed information about a torrent
func (ts *TorrentServiceImpl) GetTorrentDetail(c *gin.Context, infoHash string) (*vo.TorrentDetailResponse, error) {
	var t torrentModel.Torrent

	if err := ts.dbManager.DB().Where("info_hash = ? AND deleted = ?", infoHash, false).
		First(&t).Error; err != nil {
		ts.loggerManager.Logger().Errorf("failed to get torrent detail: %v", err)
		return nil, err
	}

	files := make([]vo.TorrentFileInfo, len(t.Files))
	for i, f := range t.Files {
		files[i] = vo.TorrentFileInfo{
			Index:        i,
			Path:         f.Path,
			Size:         f.Size,
			SizeReadable: formatSize(f.Size),
			IsStreamable: f.IsStreamable,
		}
	}

	return &vo.TorrentDetailResponse{
		InfoHash:     t.InfoHash,
		Name:         t.Name,
		TotalSize:    t.TotalSize,
		Files:        files,
		PosterPath:   t.PosterPath,
		DownloadPath: t.DownloadPath,
		Status:       t.Status,
		Progress:     t.Progress,
		CreatedAt:    t.CreatedAt,
	}, nil
}

// GetFilePath returns the file path for serving
func (ts *TorrentServiceImpl) GetFilePath(c *gin.Context, infoHash string, filePath string) (string, error) {
	client := ts.torrentManager.Client()
	return client.GetFilePath(infoHash, filePath)
}

// Helper functions

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

func getStatusFromString(status string) int {
	switch status {
	case "downloading":
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

// restoreTorrents restores active torrents from database
func (ts *TorrentServiceImpl) restoreTorrents() {
	// Give the server some time to start up
	time.Sleep(2 * time.Second)

	ts.loggerManager.Logger().Info("Starting torrent restoration...")

	var torrents []torrentModel.Torrent
	// Restore Downloading and Completed (for seeding) torrents
	// StatusDownloading = 1, StatusCompleted = 2
	err := ts.dbManager.DB().Where("deleted = ? AND status IN ?", false, []int{
		torrentModel.StatusDownloading,
		torrentModel.StatusCompleted,
	}).Find(&torrents).Error

	if err != nil {
		ts.loggerManager.Logger().Errorf("failed to load torrents for restoration: %v", err)
		return
	}

	if len(torrents) == 0 {
		ts.loggerManager.Logger().Info("No torrents to restore")
		return
	}

	ts.loggerManager.Logger().Infof("Found %d torrents to restore", len(torrents))

	for _, t := range torrents {
		// Collect selected file indices
		var selectedFiles []int
		for i, f := range t.Files {
			if f.IsSelected {
				selectedFiles = append(selectedFiles, i)
			}
		}

		// Call client to restore
		// Note: We use the raw client methods which we know exist on the implementation
		if err := ts.torrentManager.Client().RestoreTorrent(t.InfoHash, t.Trackers, selectedFiles); err != nil {
			ts.loggerManager.Logger().Errorf("failed to restore torrent %s: %v", t.InfoHash, err)
		} else {
			ts.loggerManager.Logger().Infof("Restored torrent: %s", t.Name)
		}
	}
}
