// Package service provides transcode service interface
// Author: Done-0
// Created: 2026-01-26
package service

import (
	"github.com/gin-gonic/gin"

	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/vo"
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

	// RequeueTranscode resets the transcode state for a torrent (or one file)
	// and re-runs CheckAndQueueTranscode. Verifies the caller is the creator.
	// Pass FileIndex=nil for whole-torrent requeue. Force=true also resets files
	// in Pending/Processing state.
	RequeueTranscode(c *gin.Context, req *dto.RequeueTranscodeRequest, callerUserID int64) (*vo.RequeueTranscodeResponse, error)
}
