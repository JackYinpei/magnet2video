// Package torrent provides torrent-related data model definitions
// Author: Done-0
// Created: 2026-02-14
package torrent

import (
	"github.com/Done-0/gin-scaffold/internal/model/base"
)

// TorrentFile represents a single file in a torrent
type TorrentFile struct {
	base.Base
	TorrentID    int64  `gorm:"index;not null" json:"torrent_id"`   // Foreign key to torrents table
	Index        int    `gorm:"not null" json:"index"`              // Original file index in the torrent
	Path         string `gorm:"type:varchar(1024);not null" json:"path"` // File path within the torrent or absolute path for derived files
	Size         int64  `gorm:"type:bigint;default:0" json:"size"`          // File size in bytes
	IsSelected   bool   `gorm:"default:false" json:"is_selected"`   // Whether the file is selected for download
	IsShareable  bool   `gorm:"default:false" json:"is_shareable"`  // Whether the file can be shared
	IsStreamable bool   `gorm:"default:false" json:"is_streamable"` // Whether the file can be streamed in browser (H.264)

	// Unified file metadata
	Type       string `gorm:"type:varchar(32)" json:"type"`        // File type: "video", "subtitle", "other"
	Source     string `gorm:"type:varchar(32)" json:"source"`      // File source: "original", "transcoded", "extracted"
	ParentPath string `gorm:"type:varchar(1024)" json:"parent_path"` // Parent file path (for transcoded/subtitle files)

	// Transcode fields (kept for backward compatibility on original files)
	TranscodeStatus int    `gorm:"default:0" json:"transcode_status"` // Transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodedPath  string `gorm:"type:varchar(1024)" json:"transcoded_path"`  // Path to the transcoded file (legacy)
	TranscodeError  string `gorm:"type:text" json:"transcode_error"`  // Transcode error message if failed

	// Cloud Storage fields
	CloudUploadStatus int    `gorm:"default:0" json:"cloud_upload_status"` // Cloud upload status: 0=none, 1=pending, 2=uploading, 3=completed, 4=failed
	CloudPath         string `gorm:"type:varchar(1024)" json:"cloud_path"`          // Cloud storage object path
	CloudUploadError  string `gorm:"type:text" json:"cloud_upload_error"`  // Cloud upload error message if failed

	// Subtitle metadata (when Type == "subtitle")
	StreamIndex   int    `gorm:"default:0" json:"stream_index"`   // FFmpeg stream index
	Language      string `gorm:"type:varchar(32)" json:"language"`       // ISO 639-2 language code (eng, chi, jpn)
	LanguageName  string `gorm:"type:varchar(64)" json:"language_name"`  // Human-readable name (English, Chinese)
	Title         string `gorm:"type:varchar(256)" json:"title"`          // Subtitle title/label
	Format        string `gorm:"type:varchar(16)" json:"format"`         // Output format: srt, ass, vtt
	OriginalCodec string `gorm:"type:varchar(32)" json:"original_codec"` // Original codec: subrip, ass, mov_text
}

// Transcode status constants for TorrentFile
const (
	TranscodeStatusNone       = 0 // No transcoding needed
	TranscodeStatusPending    = 1 // Waiting for transcoding
	TranscodeStatusProcessing = 2 // Currently transcoding
	TranscodeStatusCompleted  = 3 // Transcoding completed
	TranscodeStatusFailed     = 4 // Transcoding failed
)

// Cloud upload status constants for TorrentFile
const (
	CloudUploadStatusNone      = 0 // No cloud upload needed
	CloudUploadStatusPending   = 1 // Waiting for upload
	CloudUploadStatusUploading = 2 // Currently uploading
	CloudUploadStatusCompleted = 3 // Upload completed
	CloudUploadStatusFailed    = 4 // Upload failed
)

// TableName specifies table name
func (TorrentFile) TableName() string {
	return "torrent_files"
}
