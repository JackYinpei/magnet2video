// Package vo provides transcode-related value object definitions
// Author: Done-0
// Created: 2026-01-26
package vo

// TranscodeJobInfo represents a transcode job's information
type TranscodeJobInfo struct {
	ID            int64  `json:"id"`             // Job ID
	TorrentID     int64  `json:"torrent_id"`     // Associated torrent ID
	InfoHash      string `json:"info_hash"`      // Torrent info hash
	FileIndex     int    `json:"file_index"`     // File index in torrent
	InputPath     string `json:"input_path"`     // Input file path
	OutputPath    string `json:"output_path"`    // Output file path
	Status        int    `json:"status"`         // Job status
	Progress      int    `json:"progress"`       // Progress percentage
	InputCodec    string `json:"input_codec"`    // Input video codec
	OutputCodec   string `json:"output_codec"`   // Output video codec
	TranscodeType string `json:"transcode_type"` // remux or transcode
	ErrorMessage  string `json:"error_message"`  // Error message if failed
	StartedAt     int64  `json:"started_at"`     // Job start timestamp
	CompletedAt   int64  `json:"completed_at"`   // Job completion timestamp
	CreatedAt     int64  `json:"created_at"`     // Job creation timestamp
}

// TranscodeFileInfo represents transcode info for a file
type TranscodeFileInfo struct {
	FileIndex       int    `json:"file_index"`       // File index
	FilePath        string `json:"file_path"`        // Original file path
	TranscodeStatus int    `json:"transcode_status"` // Transcode status
	TranscodedPath  string `json:"transcoded_path"`  // Transcoded file path
	TranscodeError  string `json:"transcode_error"`  // Error message if failed
	NeedsTranscode  bool   `json:"needs_transcode"`  // Whether file needs transcoding
}

// TranscodeStatusResponse response for getting transcode status
type TranscodeStatusResponse struct {
	TorrentID         int64               `json:"torrent_id"`         // Torrent ID
	InfoHash          string              `json:"info_hash"`          // Torrent info hash
	OverallStatus     int                 `json:"overall_status"`     // Overall transcode status
	OverallProgress   int                 `json:"overall_progress"`   // Overall progress percentage
	TotalFiles        int                 `json:"total_files"`        // Total files in torrent
	TranscodeFiles    int                 `json:"transcode_files"`    // Files that need transcoding
	CompletedFiles    int                 `json:"completed_files"`    // Completed transcode files
	Files             []TranscodeFileInfo `json:"files"`              // File-level transcode info
	Jobs              []TranscodeJobInfo  `json:"jobs"`               // Active transcode jobs
}

// RetryTranscodeResponse response for retrying a transcode job
type RetryTranscodeResponse struct {
	JobID   int64  `json:"job_id"`  // New job ID
	Message string `json:"message"` // Status message
}

// CancelTranscodeResponse response for canceling a transcode job
type CancelTranscodeResponse struct {
	JobID   int64  `json:"job_id"`  // Canceled job ID
	Message string `json:"message"` // Status message
}

// RequeueTranscodeResponse response for re-running the transcode pipeline
type RequeueTranscodeResponse struct {
	InfoHash     string `json:"info_hash"`     // Torrent info hash
	RequeuedFiles int   `json:"requeued_files"` // Number of files that were re-queued for transcode
	Message      string `json:"message"`        // Status message
}
