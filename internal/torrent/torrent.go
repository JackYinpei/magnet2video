// Package torrent provides torrent manager interface and initialization
// Author: Done-0
// Created: 2026-01-22
package torrent

import (
	"magnet2video/configs"
	"magnet2video/internal/torrent/internal"
)

// TorrentManager defines the interface for torrent management operations
type TorrentManager interface {
	Client() *internal.Client
	Close() error
}

// New creates a new torrent manager instance
func New(config *configs.Config) (TorrentManager, error) {
	return internal.NewManager(config)
}
