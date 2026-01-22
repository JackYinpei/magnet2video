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
	Path         string `json:"path"`          // File path within the torrent
	Size         int64  `json:"size"`          // File size in bytes
	IsSelected   bool   `json:"is_selected"`   // Whether the file is selected for download
	IsShareable  bool   `json:"is_shareable"`  // Whether the file can be shared
	IsStreamable bool   `json:"is_streamable"` // Whether the file can be streamed in browser (H.264)
}

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
	InfoHash     string       `gorm:"type:varchar(64);unique;not null;index" json:"info_hash"`   // Torrent info hash
	Name         string       `gorm:"type:varchar(512)" json:"name"`                             // Torrent name
	TotalSize    int64        `gorm:"type:bigint;default:0" json:"total_size"`                   // Total size in bytes
	Files        TorrentFiles `gorm:"type:json" json:"files"`                                    // Files in the torrent
	PosterPath   string       `gorm:"type:varchar(512)" json:"poster_path"`                      // Poster file path or URL
	DownloadPath string       `gorm:"type:varchar(512)" json:"download_path"`                    // Download directory path
	Status       int          `gorm:"type:int;default:0" json:"status"`                          // Download status: 0=pending, 1=downloading, 2=completed, 3=failed, 4=paused
	Progress     float64      `gorm:"type:decimal(5,2);default:0" json:"progress"`               // Download progress percentage
	Trackers     StringSlice  `gorm:"type:json" json:"trackers"`                                 // Custom trackers
	CreatorID    int64        `gorm:"type:bigint;default:0" json:"creator_id"`                   // Creator user ID (0 if no user system)
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

// TableName specifies table name
func (Torrent) TableName() string {
	return "torrents"
}
