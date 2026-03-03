// Package vo provides torrent-related value object definitions
// Author: Done-0
// Created: 2026-01-22
package vo

// SubtitleVO represents a subtitle file in API responses
type SubtitleVO struct {
	Language     string `json:"language"`      // ISO 639-2 language code
	LanguageName string `json:"language_name"` // Human-readable language name
	Title        string `json:"title"`         // Subtitle title
	Format       string `json:"format"`        // File format: srt, ass, vtt
	FilePath     string `json:"file_path"`     // Local file path
	CloudPath    string `json:"cloud_path"`    // Cloud storage path
	FileSize     int64  `json:"file_size"`     // File size in bytes
}

// TorrentFileInfo represents a file in a torrent response (flattened structure)
type TorrentFileInfo struct {
	Index           int    `json:"index"`                      // File index in the flattened array
	Path            string `json:"path"`                       // File path (relative to download dir)
	Size            int64  `json:"size"`                       // File size in bytes
	SizeReadable    string `json:"size_readable"`              // Human readable size
	Type            string `json:"type"`                       // File type: "video", "subtitle", "other"
	Source          string `json:"source"`                     // Source: "original", "transcoded", "extracted"
	ParentPath      string `json:"parent_path"`                // Parent file path (for derived files)
	OriginalIndex   int    `json:"original_index"`             // Original file index (-1 for original files)
	IsStreamable    bool   `json:"is_streamable"`              // Whether the file can be streamed directly
	TranscodeStatus int    `json:"transcode_status,omitempty"` // Transcode status (only for original video files): 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	Language        string `json:"language,omitempty"`         // Language code (for subtitles)
	LanguageName    string `json:"language_name,omitempty"`    // Human-readable language name (for subtitles)
	Title           string `json:"title,omitempty"`            // Title (for subtitles, e.g. "English [SDH]")
	CloudPath       string `json:"cloud_path,omitempty"`       // Cloud storage path
	CloudStatus     int    `json:"cloud_status"`               // Cloud upload status: 0=none, 1=pending, 2=uploading, 3=completed, 4=failed
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

// CloudFileInfo represents minimal per-file cloud upload info for the list view
type CloudFileInfo struct {
	FileIndex         int    `json:"file_index"`                   // File index
	FileName          string `json:"file_name"`                    // File name (basename)
	CloudUploadStatus int    `json:"cloud_upload_status"`          // Cloud upload status: 0=none, 1=pending, 2=uploading, 3=completed, 4=failed
	CloudUploadError  string `json:"cloud_upload_error,omitempty"` // Error message if failed
}

// TorrentListItem represents a torrent in the list response
type TorrentListItem struct {
	InfoHash              string          `json:"info_hash"`               // Info hash
	Name                  string          `json:"name"`                    // Torrent name
	TotalSize             int64           `json:"total_size"`              // Total size in bytes
	Progress              float64         `json:"progress"`                // Progress percentage
	Status                int             `json:"status"`                  // Status code
	PosterPath            string          `json:"poster_path"`             // Poster path or URL
	CreatedAt             int64           `json:"created_at"`              // Creation timestamp
	DownloadSpeed         int64           `json:"download_speed"`          // Bytes per second
	DownloadSpeedReadable string          `json:"download_speed_readable"` // Human readable speed
	IsPublic              bool            `json:"is_public"`               // Whether the torrent is publicly shared (deprecated, use visibility)
	Visibility            int             `json:"visibility"`              // Visibility level: 0=private, 1=internal, 2=public
	TranscodeStatus       int             `json:"transcode_status"`        // Transcode status: 0=none, 1=pending, 2=processing, 3=completed, 4=failed
	TranscodeProgress     int             `json:"transcode_progress"`      // Transcode progress 0-100
	TranscodedCount       int             `json:"transcoded_count"`        // Number of transcoded files
	TotalTranscode        int             `json:"total_transcode"`         // Total files needing transcode
	CloudUploadStatus     int             `json:"cloud_upload_status"`     // Cloud upload status: 0=none, 1=pending, 2=uploading, 3=completed, 4=failed
	CloudUploadedCount    int             `json:"cloud_uploaded_count"`    // Number of uploaded files
	TotalCloudUpload      int             `json:"total_cloud_upload"`      // Total files needing upload
	CloudFiles            []CloudFileInfo `json:"cloud_files,omitempty"`   // Per-file cloud upload status
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
	IsPublic     bool              `json:"is_public"`     // Whether visibility >= public (deprecated, use visibility)
	Visibility   int               `json:"visibility"`    // Visibility level: 0=private, 1=internal, 2=public
}
