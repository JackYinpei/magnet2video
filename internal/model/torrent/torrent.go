// Package torrent provides torrent-related data model definitions
// Author: Done-0
// Created: 2026-01-22
package torrent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/Done-0/gin-scaffold/internal/model/base"
)

// TorrentFile represents a single file in a torrent
type TorrentFile struct {
	Path         string `json:"path"`          // File path within the torrent or absolute path for derived files
	Size         int64  `json:"size"`          // File size in bytes
	IsSelected   bool   `json:"is_selected"`   // Whether the file is selected for download
	IsShareable  bool   `json:"is_shareable"`  // Whether the file can be shared
	IsStreamable bool   `json:"is_streamable"` // Whether the file can be streamed in browser (H.264)

	// Unified file metadata
	Type       string `json:"type"`        // File type: "video", "subtitle", "other"
	Source     string `json:"source"`      // File source: "original", "transcoded", "extracted"
	ParentPath string `json:"parent_path"` // Parent file path (for transcoded/subtitle files)

	// Transcode fields (kept for backward compatibility on original files)
	TranscodeStatus int    `json:"transcode_status"` // Transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodedPath  string `json:"transcoded_path"`  // Path to the transcoded file (legacy)
	TranscodeError  string `json:"transcode_error"`  // Transcode error message if failed

	// Cloud Storage fields
	CloudUploadStatus int    `json:"cloud_upload_status"` // Cloud upload status: 0=none, 1=pending, 2=uploading, 3=completed, 4=failed
	CloudPath         string `json:"cloud_path"`          // Cloud storage object path
	CloudUploadError  string `json:"cloud_upload_error"`  // Cloud upload error message if failed

	// Subtitle metadata (when Type == "subtitle")
	StreamIndex   int    `json:"stream_index"`   // FFmpeg stream index
	Language      string `json:"language"`       // ISO 639-2 language code (eng, chi, jpn)
	LanguageName  string `json:"language_name"`  // Human-readable name (English, Chinese)
	Title         string `json:"title"`          // Subtitle title/label
	Format        string `json:"format"`         // Output format: srt, ass, vtt
	OriginalCodec string `json:"original_codec"` // Original codec: subrip, ass, mov_text
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

// TorrentFiles is a slice of TorrentFile that can be stored in database
type TorrentFiles []TorrentFile

// Scan implements sql.Scanner interface
func (tf *TorrentFiles) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*tf = make(TorrentFiles, 0)
	case []byte:
		return json.Unmarshal(v, tf)
	default:
		return errors.New("cannot scan into TorrentFiles")
	}
	return nil
}

// Value implements driver.Valuer interface
func (tf TorrentFiles) Value() (driver.Value, error) {
	return json.Marshal(tf)
}

// Torrent represents a torrent download record
type Torrent struct {
	base.Base
	InfoHash          string       `gorm:"type:varchar(64);unique;not null;index" json:"info_hash"` // Torrent info hash
	Name              string       `gorm:"type:varchar(512)" json:"name"`                           // Torrent name
	TotalSize         int64        `gorm:"type:bigint;default:0" json:"total_size"`                 // Total size in bytes
	Files             TorrentFiles `gorm:"type:json" json:"files"`                                  // Files in the torrent
	PosterPath        string       `gorm:"type:varchar(512)" json:"poster_path"`                    // Poster file path or URL
	DownloadPath      string       `gorm:"type:varchar(512)" json:"download_path"`                  // Download directory path
	Status            int          `gorm:"type:int;default:0" json:"status"`                        // Download status: 0=pending, 1=downloading, 2=completed, 3=failed, 4=paused
	Progress          float64      `gorm:"type:decimal(5,2);default:0" json:"progress"`             // Download progress percentage
	Trackers          StringSlice  `gorm:"type:json" json:"trackers"`                               // Custom trackers
	CreatorID         int64        `gorm:"type:bigint;default:0;index" json:"creator_id"`           // Creator user ID (0 if no user system)
	Visibility        int          `gorm:"type:int;default:0" json:"visibility"`                    // Visibility level: 0=private, 1=internal, 2=public
	TranscodeStatus   int          `gorm:"type:int;default:0" json:"transcode_status"`              // Overall transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodeProgress int          `gorm:"type:int;default:0" json:"transcode_progress"`            // Transcode progress percentage (0-100)
	TranscodedCount   int          `gorm:"type:int;default:0" json:"transcoded_count"`              // Number of transcoded files
	TotalTranscode    int          `gorm:"type:int;default:0" json:"total_transcode"`               // Total number of files to transcode

	// Cloud Storage overall status
	CloudUploadStatus   int `gorm:"type:int;default:0" json:"cloud_upload_status"`   // Overall cloud upload status
	CloudUploadProgress int `gorm:"type:int;default:0" json:"cloud_upload_progress"` // Cloud upload progress percentage (0-100)
	CloudUploadedCount  int `gorm:"type:int;default:0" json:"cloud_uploaded_count"`  // Number of uploaded files
	TotalCloudUpload    int `gorm:"type:int;default:0" json:"total_cloud_upload"`    // Total number of files to upload
}

// StringSlice is a slice of strings that can be stored in database
type StringSlice []string

// Scan implements sql.Scanner interface
func (ss *StringSlice) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*ss = make(StringSlice, 0)
	case []byte:
		return json.Unmarshal(v, ss)
	default:
		return errors.New("cannot scan into StringSlice")
	}
	return nil
}

// Value implements driver.Valuer interface
func (ss StringSlice) Value() (driver.Value, error) {
	return json.Marshal(ss)
}

// TorrentStatus constants
const (
	StatusPending     = 0 // Waiting to start
	StatusDownloading = 1 // Currently downloading
	StatusCompleted   = 2 // Download completed
	StatusFailed      = 3 // Download failed
	StatusPaused      = 4 // Download paused
)

// Visibility constants
const (
	VisibilityPrivate  = 0 // Only creator can see
	VisibilityInternal = 1 // Logged-in users can see
	VisibilityPublic   = 2 // Everyone can see
)

// TableName specifies table name
func (Torrent) TableName() string {
	return "torrents"
}
