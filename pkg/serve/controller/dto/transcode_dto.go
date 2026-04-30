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

// RequeueTranscodeRequest re-runs the transcode pipeline for a torrent.
//
// Without FileIndex (or FileIndex < 0): clears transcode state on every
// original file of this torrent and re-queues whichever ones still need it.
// With FileIndex >= 0: same but only for that one file.
//
// Force=true also re-queues files currently in Pending/Processing (be aware
// that this will produce a parallel job if a worker really is still running).
// Default Force=false skips Pending/Processing.
type RequeueTranscodeRequest struct {
	InfoHash  string `json:"info_hash" validate:"required"` // Info hash of the torrent
	FileIndex *int   `json:"file_index,omitempty"`          // Optional: only this file's index
	Force     bool   `json:"force"`                         // Override Pending/Processing mid-states
}
