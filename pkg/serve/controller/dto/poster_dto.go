// Package dto provides poster-related data transfer object definitions
// Author: Done-0
// Created: 2026-02-03
package dto

// SetPosterRequest request for setting poster from an existing file
type SetPosterRequest struct {
	InfoHash  string `json:"info_hash" validate:"required"`  // Torrent info hash
	FileIndex *int   `json:"file_index" validate:"required"` // File index in torrent files
}
