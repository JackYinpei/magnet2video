// Package types provides cloud upload message type definitions
// Author: Done-0
// Created: 2026-02-01
package types

// Topic constants for cloud upload
const (
	TopicCloudUploadJobs = "cloud-upload-jobs"
)

// CloudUploadMessage represents a cloud upload job message
type CloudUploadMessage struct {
	TorrentID     int64  `json:"torrent_id"`     // Torrent ID
	InfoHash      string `json:"info_hash"`      // Torrent info hash
	FileIndex     int    `json:"file_index"`     // File index in torrent
	SubtitleIndex int    `json:"subtitle_index"` // Deprecated: use file_index in flat files (set to -1)
	LocalPath     string `json:"local_path"`     // Local file path to upload
	CloudPath     string `json:"cloud_path"`     // Target cloud object path
	ContentType   string `json:"content_type"`   // File content type
	FileSize      int64  `json:"file_size"`      // File size in bytes
	IsTranscoded  bool   `json:"is_transcoded"`  // Whether this is a transcoded file
	CreatorID     int64  `json:"creator_id"`     // Creator user ID
	RetryCount    int    `json:"retry_count"`    // Retry count for failed uploads
}

// MaxRetryCount is the maximum number of retry attempts for failed uploads
const MaxRetryCount = 3
