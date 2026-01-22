// Package torrent provides torrent manager interface and initialization
// Author: Done-0
// Created: 2026-01-22
package torrent

import (
	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/torrent/internal"
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
