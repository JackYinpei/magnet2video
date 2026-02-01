// Package handler provides message handler for cloud upload jobs
// Author: Done-0
// Created: 2026-02-01
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
}

// NewCloudUploadHandler creates a new cloud upload handler
func NewCloudUploadHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	cloudStorageManager cloud.CloudStorageManager,
) *CloudUploadHandler {
	return &CloudUploadHandler{
		config:              config,
		loggerManager:       loggerManager,
		dbManager:           dbManager,
		cloudStorageManager: cloudStorageManager,
	}
}

// Handle processes a cloud upload job message
func (h *CloudUploadHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var uploadMsg cloudTypes.CloudUploadMessage
	if err := json.Unmarshal(msg.Value, &uploadMsg); err != nil {
		h.loggerManager.Logger().Errorf("failed to unmarshal cloud upload message: %v", err)
		return err
	}

	h.loggerManager.Logger().Infof("Processing cloud upload: torrentID=%d, fileIndex=%d, path=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.LocalPath)

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
		return fmt.Errorf("local file not found: %s", uploadMsg.LocalPath)
	}
	if err != nil {
		errMsg := fmt.Sprintf("failed to stat file: %v", err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Open local file
	file, err := os.Open(uploadMsg.LocalPath)
	if err != nil {
		errMsg := fmt.Sprintf("failed to open file: %v", err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload to cloud storage
	err = h.cloudStorageManager.UploadWithProgress(ctx, uploadMsg.CloudPath, file, uploadMsg.ContentType, fileInfo.Size(), nil)
	if err != nil {
		errMsg := fmt.Sprintf("cloud upload failed: %v", err)
		h.handleUploadFailure(uploadMsg, errMsg)
		return fmt.Errorf("cloud upload failed: %w", err)
	}

	// Update torrent file with cloud path
	if err := h.updateTorrentFileCloudCompleted(uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent file cloud path: %v", err)
	}

	// Update overall torrent cloud upload status
	if err := h.checkAndUpdateTorrentCloudStatus(uploadMsg.TorrentID); err != nil {
		h.loggerManager.Logger().Errorf("failed to update torrent cloud status: %v", err)
	}

	h.loggerManager.Logger().Infof("Cloud upload completed: torrentID=%d, fileIndex=%d, cloudPath=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath)

	return nil
}

// handleUploadFailure handles upload failure by updating status and logging
func (h *CloudUploadHandler) handleUploadFailure(msg cloudTypes.CloudUploadMessage, errMsg string) {
	h.loggerManager.Logger().Errorf("Cloud upload failed: torrentID=%d, fileIndex=%d, error=%s",
		msg.TorrentID, msg.FileIndex, errMsg)

	h.updateTorrentFileCloudStatus(msg.TorrentID, msg.FileIndex, torrentModel.CloudUploadStatusFailed, errMsg)
}

// updateTorrentFileCloudStatus updates the cloud upload status of a torrent file
func (h *CloudUploadHandler) updateTorrentFileCloudStatus(torrentID int64, fileIndex int, status int, errMsg string) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].CloudUploadStatus = status
		if errMsg != "" {
			torrentRecord.Files[fileIndex].CloudUploadError = errMsg
		}
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}

// updateTorrentFileCloudCompleted marks a torrent file's cloud upload as completed
func (h *CloudUploadHandler) updateTorrentFileCloudCompleted(torrentID int64, fileIndex int, cloudPath string) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	if fileIndex >= 0 && fileIndex < len(torrentRecord.Files) {
		torrentRecord.Files[fileIndex].CloudUploadStatus = torrentModel.CloudUploadStatusCompleted
		torrentRecord.Files[fileIndex].CloudPath = cloudPath
		torrentRecord.Files[fileIndex].CloudUploadError = ""
		torrentRecord.CloudUploadedCount++
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}

// checkAndUpdateTorrentCloudStatus checks and updates overall torrent cloud upload status
func (h *CloudUploadHandler) checkAndUpdateTorrentCloudStatus(torrentID int64) error {
	var torrentRecord torrentModel.Torrent
	if err := h.dbManager.DB().Where("id = ?", torrentID).First(&torrentRecord).Error; err != nil {
		return err
	}

	var pending, uploading, completed, failed int
	for _, file := range torrentRecord.Files {
		switch file.CloudUploadStatus {
		case torrentModel.CloudUploadStatusPending:
			pending++
		case torrentModel.CloudUploadStatusUploading:
			uploading++
		case torrentModel.CloudUploadStatusCompleted:
			completed++
		case torrentModel.CloudUploadStatusFailed:
			failed++
		}
	}

	// Determine overall status
	if uploading > 0 {
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusUploading
	} else if pending > 0 {
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusPending
	} else if failed > 0 && completed == 0 {
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusFailed
	} else if completed > 0 || (completed == 0 && pending == 0 && uploading == 0 && failed == 0) {
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusCompleted
		torrentRecord.CloudUploadProgress = 100
	}

	return h.dbManager.DB().Save(&torrentRecord).Error
}
