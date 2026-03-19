// Package vo provides IMDB/TMDB-related view object definitions
// Author: Done-0
// Created: 2026-03-19
package vo

import "github.com/Done-0/gin-scaffold/internal/tmdb"

// BindIMDBResponse response for binding IMDB ID to a torrent
type BindIMDBResponse struct {
	InfoHash string `json:"info_hash"` // Torrent info hash
	ImdbID   string `json:"imdb_id"`   // IMDB ID
	Message  string `json:"message"`   // Status message
}

// TMDBSearchResponse wraps the TMDB search results for the API response
type TMDBSearchResponse = tmdb.SearchResponse
