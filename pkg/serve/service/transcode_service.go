// Package service provides transcode service interface
// Author: Done-0
// Created: 2026-01-26
package service

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// TranscodeChecker interface for triggering transcode check
type TranscodeChecker interface {
	TriggerTranscodeCheck(torrentID int64)
}

// TranscodeService defines the interface for transcode operations
type TranscodeService interface {
	TranscodeChecker

	// CheckAndQueueTranscode checks a torrent for files that need transcoding and queues jobs
	CheckAndQueueTranscode(c *gin.Context, torrentID int64) error

	// GetTranscodeStatus returns the transcode status for a torrent
	GetTranscodeStatus(c *gin.Context, torrentID int64) (*vo.TranscodeStatusResponse, error)

	// RetryTranscode retries a failed transcode job
	RetryTranscode(c *gin.Context, req *dto.RetryTranscodeRequest) (*vo.RetryTranscodeResponse, error)

	// CancelTranscode cancels a pending or processing transcode job
	CancelTranscode(c *gin.Context, jobID int64) (*vo.CancelTranscodeResponse, error)
}
