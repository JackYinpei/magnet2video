// Package types provides transcode-related message type definitions
// Author: Done-0
// Created: 2026-01-26
package types

// TranscodeMessage represents a transcoding job message for Kafka
type TranscodeMessage struct {
	JobID       int64  `json:"job_id"`       // Transcode job ID
	TorrentID   int64  `json:"torrent_id"`   // Associated torrent ID
	InfoHash    string `json:"info_hash"`    // Torrent info hash
	FileIndex   int    `json:"file_index"`   // File index in torrent
	InputPath   string `json:"input_path"`   // Input file path
	OutputPath  string `json:"output_path"`  // Output file path
	InputCodec  string `json:"input_codec"`  // Detected input codec
	Operation   string `json:"operation"`    // Operation type: "remux" or "transcode"
	Priority    int    `json:"priority"`     // Job priority (higher = more urgent)
	CreatorID   int64  `json:"creator_id"`   // User who created the torrent
	Preset      string `json:"preset"`       // FFmpeg preset (for transcode)
	CRF         int    `json:"crf"`          // Constant Rate Factor (for transcode)
}

// TranscodeProgressMessage represents a progress update message
type TranscodeProgressMessage struct {
	JobID     int64   `json:"job_id"`     // Transcode job ID
	TorrentID int64   `json:"torrent_id"` // Associated torrent ID
	InfoHash  string  `json:"info_hash"`  // Torrent info hash
	FileIndex int     `json:"file_index"` // File index in torrent
	Progress  float64 `json:"progress"`   // Progress percentage (0-100)
	Status    int     `json:"status"`     // Job status
	Error     string  `json:"error"`      // Error message if failed
}

// TranscodeResultMessage represents the result of a transcoding job
type TranscodeResultMessage struct {
	JobID        int64  `json:"job_id"`        // Transcode job ID
	TorrentID    int64  `json:"torrent_id"`    // Associated torrent ID
	InfoHash     string `json:"info_hash"`     // Torrent info hash
	FileIndex    int    `json:"file_index"`    // File index in torrent
	Success      bool   `json:"success"`       // Whether transcoding succeeded
	OutputPath   string `json:"output_path"`   // Output file path
	OutputCodec  string `json:"output_codec"`  // Output codec used
	OutputSize   int64  `json:"output_size"`   // Output file size in bytes
	Duration     int64  `json:"duration"`      // Transcoding duration in milliseconds
	ErrorMessage string `json:"error_message"` // Error message if failed
}

// Kafka topic names
const (
	TopicTranscodeJobs     = "transcode-jobs"     // Topic for new transcode jobs
	TopicTranscodeProgress = "transcode-progress" // Topic for progress updates
	TopicTranscodeResults  = "transcode-results"  // Topic for job results
)

// Operation types
const (
	OperationRemux     = "remux"     // Container conversion only
	OperationTranscode = "transcode" // Full video transcoding
)
