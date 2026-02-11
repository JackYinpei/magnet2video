// Package vo provides cloud storage related view object definitions
// Author: Done-0
// Created: 2026-02-01
package vo

// CloudURLResponse response for getting a signed cloud URL
type CloudURLResponse struct {
	URL       string `json:"url"`        // Signed URL for accessing the file
	ExpiresAt int64  `json:"expires_at"` // Expiration time as Unix timestamp
}

// CloudUploadStatusResponse response for cloud upload status
type CloudUploadStatusResponse struct {
	TorrentID           int64                    `json:"torrent_id"`
	InfoHash            string                   `json:"info_hash"`
	OverallStatus       int                      `json:"overall_status"`
	OverallProgress     int                      `json:"overall_progress"`
	TotalFiles          int                      `json:"total_files"`
	CloudUploadFiles    int                      `json:"cloud_upload_files"`
	CompletedFiles      int                      `json:"completed_files"`
	Files               []CloudUploadFileInfo    `json:"files"`
}

// CloudUploadFileInfo file cloud upload status info
type CloudUploadFileInfo struct {
	FileIndex         int    `json:"file_index"`
	FilePath          string `json:"file_path"`
	CloudUploadStatus int    `json:"cloud_upload_status"`
	CloudPath         string `json:"cloud_path"`
	CloudUploadError  string `json:"cloud_upload_error"`
}

// RetryCloudUploadResponse response for retrying failed cloud uploads
type RetryCloudUploadResponse struct {
	InfoHash    string `json:"info_hash"`    // Info hash of the torrent
	RetriedCount int   `json:"retried_count"` // Number of files re-queued for upload
	Message     string `json:"message"`      // Status message
}
