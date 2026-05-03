// Package dto provides cloud storage related data transfer object definitions
// Author: Done-0
// Created: 2026-02-01
package dto

// GetCloudURLRequest request for getting a signed cloud URL
type GetCloudURLRequest struct {
	InfoHash  string `json:"info_hash" validate:"required"` // Info hash of the torrent
	FileIndex int    `json:"file_index" validate:"gte=0"`   // Index of the file within the torrent
}

// RetryCloudUploadRequest request for retrying cloud uploads.
// Force=true also re-queues files currently in Pending/Uploading (use with care:
// if a worker is genuinely uploading the file, this will produce two concurrent
// uploads). Default Force=false only retries Failed and never-attempted files.
type RetryCloudUploadRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Info hash of the torrent
	Force    bool   `json:"force"`                          // Override Pending/Uploading mid-states
}

// RetryCloudUploadFileRequest request for retrying cloud upload for a single file.
// Force semantics same as RetryCloudUploadRequest.
type RetryCloudUploadFileRequest struct {
	InfoHash  string `json:"info_hash" validate:"required"`  // Info hash of the torrent
	FileIndex int    `json:"file_index" validate:"gte=0"`    // Index of the file to re-upload
	Force     bool   `json:"force"`                          // Override Pending/Uploading mid-states
}

// DeleteLocalFilesRequest request for deleting local files of a torrent.
// Force=true bypasses the "all files must be uploaded to cloud" safety check
// — the caller is acknowledging that some files may not have a cloud copy
// and is willing to drop the local files anyway. Use sparingly: combined
// with a missing or failed cloud upload it deletes the only copy.
type DeleteLocalFilesRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Info hash of the torrent
	Force    bool   `json:"force"`                         // Bypass cloud-completed safety check
}
