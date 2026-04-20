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

	"magnet2video/configs"
	"magnet2video/internal/cloud"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/events/gateway"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
)

// CloudUploadHandler handles cloud upload job messages. It publishes worker
// events through the gateway rather than writing to the DB directly.
type CloudUploadHandler struct {
	config              *configs.Config
	loggerManager       logger.LoggerManager
	gateway             gateway.WorkerGateway
	cloudStorageManager cloud.CloudStorageManager
	queueProducer       queue.Producer
}

// NewCloudUploadHandler creates a new cloud upload handler
func NewCloudUploadHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	workerGateway gateway.WorkerGateway,
	cloudStorageManager cloud.CloudStorageManager,
	queueProducer queue.Producer,
) *CloudUploadHandler {
	return &CloudUploadHandler{
		config:              config,
		loggerManager:       loggerManager,
		gateway:             workerGateway,
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

	h.loggerManager.Logger().Infof("Processing cloud upload: torrentID=%d, fileIndex=%d, retry=%d, path=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.RetryCount, uploadMsg.LocalPath)

	// Skip upload if object already exists in cloud storage
	if uploadMsg.CloudPath != "" && h.cloudStorageManager != nil && h.cloudStorageManager.IsEnabled() {
		exists, err := h.cloudStorageManager.Exists(ctx, uploadMsg.CloudPath)
		if err != nil {
			h.loggerManager.Logger().Warnf("failed to check cloud object existence: %v", err)
		} else if exists {
			_ = h.gateway.CloudUploadCompleted(ctx, uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath)
			h.loggerManager.Logger().Infof("Cloud object already exists, skip upload: cloudPath=%s", uploadMsg.CloudPath)
			return nil
		}
	}

	if err := h.gateway.CloudUploadStarted(ctx, uploadMsg.TorrentID, uploadMsg.FileIndex); err != nil {
		h.loggerManager.Logger().Warnf("failed to publish upload started event: %v", err)
	}

	fileInfo, err := os.Stat(uploadMsg.LocalPath)
	if os.IsNotExist(err) {
		h.failUpload(ctx, uploadMsg, fmt.Sprintf("local file not found: %s", uploadMsg.LocalPath))
		return nil
	}
	if err != nil {
		h.failUpload(ctx, uploadMsg, fmt.Sprintf("failed to stat file: %v", err))
		return nil
	}

	file, err := os.Open(uploadMsg.LocalPath)
	if err != nil {
		h.failUpload(ctx, uploadMsg, fmt.Sprintf("failed to open file: %v", err))
		return nil
	}
	defer file.Close()

	if err := h.cloudStorageManager.UploadWithProgress(ctx, uploadMsg.CloudPath, file, uploadMsg.ContentType, fileInfo.Size(), nil); err != nil {
		if h.requeueForRetry(uploadMsg, err) {
			return nil
		}
		h.failUpload(ctx, uploadMsg, fmt.Sprintf("cloud upload failed after %d retries: %v", uploadMsg.RetryCount, err))
		return nil
	}

	if err := h.gateway.CloudUploadCompleted(ctx, uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath); err != nil {
		h.loggerManager.Logger().Errorf("failed to publish upload completed event: %v", err)
	}

	h.loggerManager.Logger().Infof("Cloud upload completed: torrentID=%d, fileIndex=%d, cloudPath=%s",
		uploadMsg.TorrentID, uploadMsg.FileIndex, uploadMsg.CloudPath)
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

	msg.RetryCount++
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.loggerManager.Logger().Errorf("failed to marshal retry message: %v", err)
		return false
	}

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

// failUpload publishes a failure event
func (h *CloudUploadHandler) failUpload(ctx context.Context, msg cloudTypes.CloudUploadMessage, errMsg string) {
	h.loggerManager.Logger().Errorf("Cloud upload failed: torrentID=%d, fileIndex=%d, error=%s",
		msg.TorrentID, msg.FileIndex, errMsg)
	_ = h.gateway.CloudUploadFailed(ctx, msg.TorrentID, msg.FileIndex, errMsg)
}
