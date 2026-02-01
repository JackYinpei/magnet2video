// Package transcode provides transcode job data model definitions
// Author: Done-0
// Created: 2026-01-26
package transcode

import "github.com/Done-0/gin-scaffold/internal/model/base"

// TranscodeJob represents a video transcoding job
type TranscodeJob struct {
	base.Base
	TorrentID     int64  `gorm:"type:bigint;index" json:"torrent_id"`            // Associated Torrent ID
	InfoHash      string `gorm:"type:varchar(64);index" json:"info_hash"`        // Torrent info hash
	InputPath     string `gorm:"type:varchar(512);not null" json:"input_path"`   // Input file path
	OutputPath    string `gorm:"type:varchar(512)" json:"output_path"`           // Output file path
	FileIndex     int    `gorm:"type:int" json:"file_index"`                     // File index in torrent
	Status        int    `gorm:"type:int;default:0" json:"status"`               // Job status
	Progress      int    `gorm:"type:int;default:0" json:"progress"`             // Progress percentage (0-100)
	InputCodec    string `gorm:"type:varchar(32)" json:"input_codec"`            // Input video codec
	OutputCodec   string `gorm:"type:varchar(32)" json:"output_codec"`           // Output video codec
	TranscodeType string `gorm:"type:varchar(16)" json:"transcode_type"`         // "remux" or "transcode"
	Duration      int64  `gorm:"type:bigint" json:"duration"`                    // Video duration in seconds
	ErrorMessage  string `gorm:"type:text" json:"error_message"`                 // Error message if failed
	StartedAt     int64  `gorm:"type:bigint" json:"started_at"`                  // Start timestamp
	CompletedAt   int64  `gorm:"type:bigint" json:"completed_at"`                // Completion timestamp
	CreatorID     int64  `gorm:"type:bigint;index" json:"creator_id"`            // Creator user ID
}

// TableName specifies table name
func (TranscodeJob) TableName() string {
	return "transcode_jobs"
}

// Job status constants
const (
	JobStatusPending    = 0 // Waiting for processing
	JobStatusProcessing = 1 // Currently processing
	JobStatusCompleted  = 2 // Completed successfully
	JobStatusFailed     = 3 // Failed
	JobStatusCancelled  = 4 // Cancelled
)
