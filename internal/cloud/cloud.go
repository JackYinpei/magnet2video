// Package cloud provides cloud storage abstraction layer.
//
// Only S3 (and S3-compatible services such as MinIO / Ceph / self-hosted
// providers) is supported. The factory used to switch on a `Provider` config
// field, but with GCS removed there's nothing to switch on — every call
// returns the S3 manager.
//
// Author: Done-0
// Created: 2026-02-01
package cloud

import (
	"magnet2video/configs"
	"magnet2video/internal/cloud/internal"
	"magnet2video/internal/logger"
)

// CloudStorageManager defines cloud storage operations interface.
type CloudStorageManager = internal.CloudStorageManager

// New creates a new CloudStorageManager backed by S3 (or S3-compatible
// storage configured via CLOUD_STORAGE.ENDPOINT).
func New(config *configs.Config, loggerManager logger.LoggerManager) CloudStorageManager {
	return internal.NewS3Manager(config, loggerManager)
}
