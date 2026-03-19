// Package dto provides IMDB-related data transfer object definitions
// Author: Done-0
// Created: 2026-03-19
package dto

// BindIMDBRequest request for binding IMDB ID to a torrent
type BindIMDBRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Torrent info hash
	ImdbID   string `json:"imdb_id" validate:"required"`   // IMDB ID (e.g. tt1234567)
}
