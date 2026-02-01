// Package cloud provides cloud storage abstraction layer
// Author: Done-0
// Created: 2026-02-01
package cloud

import (
	"strings"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/cloud/internal"
	"github.com/Done-0/gin-scaffold/internal/logger"
)

// CloudStorageManager defines cloud storage operations interface
type CloudStorageManager = internal.CloudStorageManager

// New creates a new CloudStorageManager instance based on provider configuration
func New(config *configs.Config, loggerManager logger.LoggerManager) CloudStorageManager {
	provider := strings.ToLower(config.CloudStorageConfig.Provider)
	switch provider {
	case "s3":
		return internal.NewS3Manager(config, loggerManager)
	default:
		// Default to GCS for backward compatibility
		return internal.New(config, loggerManager)
	}
}
