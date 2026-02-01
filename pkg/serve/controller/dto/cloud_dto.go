// Package dto provides cloud storage related data transfer object definitions
// Author: Done-0
// Created: 2026-02-01
package dto

// GetCloudURLRequest request for getting a signed cloud URL
type GetCloudURLRequest struct {
	InfoHash  string `json:"info_hash" validate:"required"` // Info hash of the torrent
	FileIndex int    `json:"file_index" validate:"gte=0"`   // Index of the file within the torrent
}
