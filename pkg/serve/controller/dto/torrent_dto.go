// Package dto provides torrent-related data transfer object definitions
// Author: Done-0
// Created: 2026-01-22
package dto

// ParseMagnetRequest request for parsing a magnet URI
type ParseMagnetRequest struct {
	MagnetURI string   `json:"magnet_uri" validate:"required"` // Magnet URI to parse
	Trackers  []string `json:"trackers"`                       // Optional custom trackers
}

// StartDownloadRequest request for starting a torrent download
type StartDownloadRequest struct {
	InfoHash      string   `json:"info_hash" validate:"required"`      // Info hash of the torrent
	SelectedFiles []int    `json:"selected_files" validate:"required"` // Indices of files to download
	Trackers      []string `json:"trackers"`                           // Optional custom trackers
}

// GetProgressRequest request for getting download progress
type GetProgressRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Info hash of the torrent
}

// PauseDownloadRequest request for pausing a download
type PauseDownloadRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Info hash of the torrent
}

// ResumeDownloadRequest request for resuming a download
type ResumeDownloadRequest struct {
	InfoHash      string `json:"info_hash" validate:"required"`      // Info hash of the torrent
	SelectedFiles []int  `json:"selected_files" validate:"required"` // Indices of files to resume
}

// RemoveTorrentRequest request for removing a torrent
type RemoveTorrentRequest struct {
	InfoHash    string `json:"info_hash" validate:"required"` // Info hash of the torrent
	DeleteFiles bool   `json:"delete_files"`                  // Whether to delete downloaded files
}

// ServeFileRequest request for serving a file
type ServeFileRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Info hash of the torrent
	FilePath string `json:"file_path" validate:"required"` // Path of the file within the torrent
}

// UpdateTorrentRequest request for updating torrent metadata
type UpdateTorrentRequest struct {
	InfoHash   string `json:"info_hash" validate:"required"` // Info hash of the torrent
	PosterPath string `json:"poster_path"`                   // Poster file path or URL
}
