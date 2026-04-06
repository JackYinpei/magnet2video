// Package model provides database model definitions and management
// Author: Done-0
// Created: 2025-08-24
package model

import (
	"magnet2video/internal/model/torrent"
	"magnet2video/internal/model/transcode"
	"magnet2video/internal/model/user"
)

// GetAllModels gets and registers all models for database migration
func GetAllModels() []any {
	return []any{
		&user.User{},             // User model
		&torrent.Torrent{},       // Torrent model
		&torrent.TorrentFile{},   // TorrentFile model
		&transcode.TranscodeJob{}, // Transcode job model
	}
}
