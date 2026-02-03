// Package vo provides poster-related view object definitions
// Author: Done-0
// Created: 2026-02-03
package vo

// PosterResponse response for poster updates or uploads
type PosterResponse struct {
	InfoHash   string `json:"info_hash"`            // Torrent info hash
	PosterPath string `json:"poster_path"`          // Poster path or URL
	CloudPath  string `json:"cloud_path,omitempty"` // Cloud storage object path (if any)
	Message    string `json:"message"`              // Status message
}
