// Package torrent provides torrent-related data model definitions
// Author: Done-0
// Created: 2026-01-22
package torrent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"magnet2video/internal/model/base"
)

// Torrent represents a torrent download record
type Torrent struct {
	base.Base
	InfoHash          string        `gorm:"type:varchar(64);unique;not null;index" json:"info_hash"` // Torrent info hash
	Name              string        `gorm:"type:varchar(512)" json:"name"`                           // Torrent name
	TotalSize         int64         `gorm:"type:bigint;default:0" json:"total_size"`                 // Total size in bytes
	Files             []TorrentFile `gorm:"foreignKey:TorrentID" json:"files"`                       // Files in the torrent
	PosterPath        string        `gorm:"type:varchar(512)" json:"poster_path"`                    // Poster file path or URL
	ImdbID            string        `gorm:"type:varchar(20)" json:"imdb_id"`                         // IMDB ID (e.g. tt1234567)
	DownloadPath      string        `gorm:"type:varchar(512)" json:"download_path"`                  // Download directory path
	Status            int           `gorm:"type:int;default:0" json:"status"`                        // Download status: 0=pending, 1=downloading, 2=completed, 3=failed, 4=paused
	Progress          float64       `gorm:"type:decimal(5,2);default:0" json:"progress"`             // Download progress percentage
	Trackers          StringSlice   `gorm:"type:json" json:"trackers"`                               // Custom trackers
	CreatorID         int64         `gorm:"type:bigint;default:0;index" json:"creator_id"`           // Creator user ID (0 if no user system)
	Visibility        int           `gorm:"type:int;default:0" json:"visibility"`                    // Visibility level: 0=private, 1=internal, 2=public
	TranscodeStatus   int           `gorm:"type:int;default:0" json:"transcode_status"`              // Overall transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodeProgress int           `gorm:"type:int;default:0" json:"transcode_progress"`            // Transcode progress percentage (0-100)
	TranscodedCount   int           `gorm:"type:int;default:0" json:"transcoded_count"`              // Number of transcoded files
	TotalTranscode    int           `gorm:"type:int;default:0" json:"total_transcode"`               // Total number of files to transcode

	// Cloud Storage overall status
	CloudUploadStatus   int `gorm:"type:int;default:0" json:"cloud_upload_status"`   // Overall cloud upload status
	CloudUploadProgress int `gorm:"type:int;default:0" json:"cloud_upload_progress"` // Cloud upload progress percentage (0-100)
	CloudUploadedCount  int `gorm:"type:int;default:0" json:"cloud_uploaded_count"`  // Number of uploaded files
	TotalCloudUpload    int `gorm:"type:int;default:0" json:"total_cloud_upload"`    // Total number of files to upload

	// Local file management
	LocalDeleted bool `gorm:"default:false" json:"local_deleted"` // Whether local files have been deleted

	// WorkerID identifies the worker node that owns this torrent's on-disk
	// state. It is set by the worker when it first picks up a download/start
	// job and used by the server to route subsequent commands (download
	// control, file ops) back to the same worker. Empty means the torrent
	// has not been claimed yet — typical for freshly-created records or for
	// single-worker deployments where targeting is unnecessary.
	WorkerID string `gorm:"type:varchar(64);index;default:''" json:"worker_id"`
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
	StatusPending        = 0 // Waiting to start
	StatusDownloading    = 1 // Currently downloading
	StatusCompleted      = 2 // Download completed
	StatusFailed         = 3 // Download failed
	StatusPaused         = 4 // Download paused
	StatusSeedingStopped = 5 // Removed from swarm by user; local files retained
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
