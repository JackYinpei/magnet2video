// Package dto provides transcode-related data transfer object definitions
// Author: Done-0
// Created: 2026-01-26
package dto

// RetryTranscodeRequest request for retrying a transcode job
type RetryTranscodeRequest struct {
	JobID int64 `json:"job_id" validate:"required"` // Job ID to retry
}

// CancelTranscodeRequest request for canceling a transcode job
type CancelTranscodeRequest struct {
	JobID int64 `json:"job_id" validate:"required"` // Job ID to cancel
}
