// Package vo provides torrent-related value object definitions
// Author: Done-0
// Created: 2026-01-22
package vo

// TorrentFileInfo represents a file in a torrent response
type TorrentFileInfo struct {
	Index        int    `json:"index"`         // File index
	Path         string `json:"path"`          // File path
	Size         int64  `json:"size"`          // File size in bytes
	SizeReadable string `json:"size_readable"` // Human readable size
	IsStreamable bool   `json:"is_streamable"` // Whether the file can be streamed
}

// ParseMagnetResponse response for parsing a magnet URI
type ParseMagnetResponse struct {
	InfoHash  string            `json:"info_hash"`  // Info hash
	Name      string            `json:"name"`       // Torrent name
	TotalSize int64             `json:"total_size"` // Total size in bytes
	Files     []TorrentFileInfo `json:"files"`      // Files in the torrent
}

// StartDownloadResponse response for starting a download
type StartDownloadResponse struct {
	InfoHash string `json:"info_hash"` // Info hash
	Message  string `json:"message"`   // Status message
}

// DownloadProgressResponse response for download progress
type DownloadProgressResponse struct {
	InfoHash              string  `json:"info_hash"`               // Info hash
	Name                  string  `json:"name"`                    // Torrent name
	TotalSize             int64   `json:"total_size"`              // Total size in bytes
	DownloadedSize        int64   `json:"downloaded_size"`         // Downloaded size in bytes
	Progress              float64 `json:"progress"`                // Progress percentage
	Status                string  `json:"status"`                  // Status: downloading, completed, paused, etc.
	Peers                 int     `json:"peers"`                   // Active peers
	Seeds                 int     `json:"seeds"`                   // Connected seeds
	DownloadSpeed         int64   `json:"download_speed"`          // Bytes per second
	DownloadSpeedReadable string  `json:"download_speed_readable"` // Human readable speed
}

// PauseDownloadResponse response for pausing a download
type PauseDownloadResponse struct {
	InfoHash string `json:"info_hash"` // Info hash
	Message  string `json:"message"`   // Status message
}

// ResumeDownloadResponse response for resuming a download
type ResumeDownloadResponse struct {
	InfoHash string `json:"info_hash"` // Info hash
	Message  string `json:"message"`   // Status message
}

// RemoveTorrentResponse response for removing a torrent
type RemoveTorrentResponse struct {
	InfoHash string `json:"info_hash"` // Info hash
	Message  string `json:"message"`   // Status message
}

// TorrentListItem represents a torrent in the list response
type TorrentListItem struct {
	InfoHash              string  `json:"info_hash"`               // Info hash
	Name                  string  `json:"name"`                    // Torrent name
	TotalSize             int64   `json:"total_size"`              // Total size in bytes
	Progress              float64 `json:"progress"`                // Progress percentage
	Status                int     `json:"status"`                  // Status code
	PosterPath            string  `json:"poster_path"`             // Poster path or URL
	CreatedAt             int64   `json:"created_at"`              // Creation timestamp
	DownloadSpeed         int64   `json:"download_speed"`          // Bytes per second
	DownloadSpeedReadable string  `json:"download_speed_readable"` // Human readable speed
	IsPublic              bool    `json:"is_public"`               // Whether the torrent is publicly shared
	TranscodeStatus       int     `json:"transcode_status"`        // Transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodeProgress     int     `json:"transcode_progress"`      // Transcode progress 0-100
	TranscodedCount       int     `json:"transcoded_count"`        // Number of transcoded files
	TotalTranscode        int     `json:"total_transcode"`         // Total files needing transcode
}

// TorrentListResponse response for listing torrents
type TorrentListResponse struct {
	Torrents []TorrentListItem `json:"torrents"` // List of torrents
	Total    int               `json:"total"`    // Total count
}

// TorrentDetailResponse response for torrent details
type TorrentDetailResponse struct {
	InfoHash     string            `json:"info_hash"`     // Info hash
	Name         string            `json:"name"`          // Torrent name
	TotalSize    int64             `json:"total_size"`    // Total size in bytes
	Files        []TorrentFileInfo `json:"files"`         // Files in the torrent
	PosterPath   string            `json:"poster_path"`   // Poster path or URL
	DownloadPath string            `json:"download_path"` // Download directory path
	Status       int               `json:"status"`        // Status code
	Progress     float64           `json:"progress"`      // Progress percentage
	CreatedAt    int64             `json:"created_at"`    // Creation timestamp
}
