// Package handler provides message handler for cloud upload jobs
// Author: Done-0
// Created: 2026-02-01
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/cloud"
	cloudTypes "github.com/Done-0/gin-scaffold/internal/cloud/types"
	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	"github.com/Done-0/gin-scaffold/internal/queue"
)

// CloudUploadHandler handles cloud upload job messages
type CloudUploadHandler struct {
	config              *configs.Config
	loggerManager       logger.LoggerManager
	dbManager           db.DatabaseManager
	cloudStorageManager cloud.CloudStorageManager
	queueProducer       queue.Producer
}

// NewCloudUploadHandler creates a new cloud upload handler
func NewCloudUploadHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	cloudStorageManager cloud.CloudStorageManager,
	queueProducer queue.Producer,
) *CloudUploadHandler {
	return &CloudUploadHandler{
		config:              config,
		loggerManager:       loggerManager,
		dbManager:           dbManager,
		cloudStorageManager: cloudStorageManager,
		queueProducer:       queueProducer,
	}
}

// Handle processes a cloud upload job message
func (h *CloudUploadHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var uploadMsg cloudTypes.CloudUploadMessage
	if err := json.Unmarshal(msg.Value, &uploadMsg); err != nil {
		h.loggerManager.Logger().Errorf("failed to unmarshal cloud upload message: %v", err)
		return err
	}

	h.loggerManager.Logger().Infof("Processing cloud upload: torrentID=%d, fileIndex=%d, subtitleIndex=%d, retry=%d, path=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.SubtitleIndex, uploadMsg.RetryCount, uploadMsg.LocalPath)

	// Skip upload if object already exists in cloud storage
	if uploadMsg.CloudPath != "" && h.cloudStorageManager != nil && h.cloudStorageManager.IsEnabled() {
		exists, err := h.cloudStorageManager.Exists(ctx, uploadMsg.CloudPath)
		if err != nil {
			h.loggerManager.Logger().Warnf("failed to check cloud object existence: %v", err)
		} else if exists {
			if err := h.updateTorrentFileCloudCompleted(uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath); err != nil {
				h.loggerManager.Logger().Errorf("failed to update existing cloud upload status: %v", err)
			}
			if err := h.checkAndUpdateTorrentCloudStatus(uploadMsg.TorrentID); err != nil {
				h.loggerManager.Logger().Errorf("failed to update torrent cloud status: %v", err)
			}
			h.loggerManager.Logger().Infof("Cloud object already exists, skip upload: torrentID=%d, fileIndex=%d, cloudPath=%s",
				uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath)
			return nil
		}
	}

	// Update file cloud upload status to uploading
	if err := h.updateTorrentFileCloudStatus(uploadMsg.TorrentID, uploadMsg.FileIndex, torrentModel.CloudUploadStatusUploading, ""); err != nil {
		h.loggerManager.Logger().Errorf("failed to update cloud upload status: %v", err)
		return err
	}

	// Check if local file exists
	fileInfo, err := os.Stat(uploadMsg.LocalPath)
	if os.IsNotExist(err) {
		errMsg := fmt.Sprintf("local file not found: %s", uploadMsg.LocalPath)
		h.handleUploadFailure(uploadMsg, errMsg)
		return nil // Don't retry if file doesn't exist
	}
	if err != nil {
		errMsg := fmt.Sprintf("failed to stat file: %v", err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return nil
	}

	// Open local file
	file, err := os.Open(uploadMsg.LocalPath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open file: %v", err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return nil
	}
	defer file.Close()

	// Upload to cloud storage
	err = h.cloudStorageManager.UploadWithProgress(ctx, uploadMsg.CloudPath, file, uploadMsg.ContentType, fileInfo.Size(), nil)
	if err != nil {
		// Try to re-queue for retry
		if h.requeueForRetry(uploadMsg, err) {
			return nil // Successfully re-queued, don't mark as failed yet
		}
		// Max retries exceeded, mark as failed
		errMsg := fmt.Sprintf("cloud upload failed after %d retries: %v", uploadMsg.RetryCount, err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return nil
	}

	// Update torrent file with cloud path
	if err := h.updateTorrentFileCloudCompleted(uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent file cloud path: %v", err)
	}

	// Update overall torrent cloud upload status
	if err := h.checkAndUpdateTorrentCloudStatus(uploadMsg.TorrentID); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent cloud status: %v", err)
	}

	h.loggerManager.Logger().Infof("Cloud upload completed: torrentID=%d, fileIndex=%d, subtitleIndex=%d, cloudPath=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.SubtitleIndex, uploadMsg.CloudPath)

	return nil
}

// requeueForRetry attempts to re-queue a failed upload for retry
func (h *CloudUploadHandler) requeueForRetry(msg cloudTypes.CloudUploadMessage, uploadErr error) bool {
	if msg.RetryCount >= cloudTypes.MaxRetryCount {
		h.loggerManager.Logger().Warnf("Max retries exceeded for cloud upload: torrentID=%d, fileIndex=%d, error=%v",
			msg.TorrentID, msg.FileIndex, uploadErr)
		return false
	}

	if h.queueProducer == nil {
		return false
	}

	// Increment retry count
	msg.RetryCount++

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.loggerManager.Logger().Errorf("failed to marshal retry message: %v", err)
		return false
	}

	// Wait before retry (exponential backoff: 5s, 10s, 20s)
	backoff := time.Duration(5<<msg.RetryCount) * time.Second
	h.loggerManager.Logger().Infof("Re-queuing cloud upload for retry %d/%d in %v: torrentID=%d, fileIndex=%d, error=%v",
		msg.RetryCount, cloudTypes.MaxRetryCount, backoff, msg.TorrentID, msg.FileIndex, uploadErr)

	go func() {
		time.Sleep(backoff)
		if err := h.queueProducer.Send(context.Background(), cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
			h.loggerManager.Logger().Errorf("failed to re-queue upload: %v", err)
		}
	}()

	return true
}

// handleUploadFailure handles upload failure by updating status and logging
func (h *CloudUploadHandler) handleUploadFailure(msg cloudTypes.CloudUploadMessage, errMsg string) {
	h.loggerManager.Logger().Errorf("Cloud upload failed: torrentID=%d, fileIndex=%d, subtitleIndex=%d, error=%s",
		msg.TorrentID, msg.FileIndex, msg.SubtitleIndex, errMsg)

	if err := h.updateTorrentFileCloudStatus(msg.TorrentID, msg.FileIndex, torrentModel.CloudUploadStatusFailed, errMsg); err != nil {
		h.loggerManager.Logger().Errorf("failed to mark cloud upload as failed: %v", err)
	}
	if err := h.checkAndUpdateTorrentCloudStatus(msg.TorrentID); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent cloud status after failure: %v", err)
	}
}

// updateTorrentFileCloudStatus updates the cloud upload status of a torrent file
func (h *CloudUploadHandler) updateTorrentFileCloudStatus(torrentID int64, fileIndex int, status int, errMsg string) error {
	updates := map[string]interface{}{
		"cloud_upload_status": status,
	}
	if errMsg != "" {
		updates["cloud_upload_error"] = errMsg
	}

	return h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(updates).Error
}

func (h *CloudUploadHandler) updateTorrentFileCloudCompleted(torrentID int64, fileIndex int, cloudPath string) error {
	// First get current status to see if we need to increment uploaded count? 
	// Actually checkAndUpdateTorrentCloudStatus recalculates everything, so we don't need to maintain incremental count here manually 
	// if we just call that function afterwards. 
	// The original code did: torrentRecord.CloudUploadedCount++ if prevStatus != Completed.
	// But checkAndUpdateTorrentCloudStatus re-sums it anyway.
	
	updates := map[string]interface{}{
		"cloud_upload_status": torrentModel.CloudUploadStatusCompleted,
		"cloud_path":          cloudPath,
		"cloud_upload_error":  "",
	}

	return h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(updates).Error
}

func (h *CloudUploadHandler) checkAndUpdateTorrentCloudStatus(torrentID int64) error {
	type Result struct {
		Status int
		Count  int
	}
	var results []Result
	// Count files by status
	if err := h.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Select("cloud_upload_status as status, count(*) as count").
		Where("torrent_id = ?", torrentID).
		Group("cloud_upload_status").
		Scan(&results).Error; err != nil {
		return err
	}

	var pending, uploading, completed, failed int
	var total, uploaded int

	for _, r := range results {
		count := r.Count
		status := r.Status
		
		if status != torrentModel.CloudUploadStatusNone {
			total += count
		}
		if status == torrentModel.CloudUploadStatusCompleted {
			uploaded += count
			completed += count
		} else if status == torrentModel.CloudUploadStatusPending {
			pending += count
		} else if status == torrentModel.CloudUploadStatusUploading {
			uploading += count
		} else if status == torrentModel.CloudUploadStatusFailed {
			failed += count
		}
	}

	updates := map[string]interface{}{
		"total_cloud_upload":   total,
		"cloud_uploaded_count": uploaded,
		"cloud_upload_progress": 0,
		"cloud_upload_status":   torrentModel.CloudUploadStatusNone,
	}

	if total > 0 {
		updates["cloud_upload_progress"] = int(float64(uploaded) * 100 / float64(total))
	}

	if uploading > 0 {
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusUploading
	} else if pending > 0 {
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusPending
	} else if failed > 0 && completed == 0 {
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusFailed
	} else if completed > 0 || (completed == 0 && pending == 0 && uploading == 0 && failed == 0) {
		// If we have some completed, or if everything is done (which here means empty or all None? No, total>0 logic covers)
		// Actually if total > 0 and we are here, it means we have mixed states but no active uploading/pending.
		// If failed > 0, it's partial failure? 
		// Original logic: "else if failed > 0 && completed == 0" -> Failed.
		// "else if completed > 0 || ..." -> Completed.
		// So if we have some failed and some completed -> Completed (or partial).
		// Let's stick to original logic:
		updates["cloud_upload_status"] = torrentModel.CloudUploadStatusCompleted
	}
	
	// If total is 0, status remains None (default)

	return h.dbManager.DB().Model(&torrentModel.Torrent{}).Where("id = ?", torrentID).Updates(updates).Error
}
