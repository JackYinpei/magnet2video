// Package routes provides torrent route registration functionality
// Author: Done-0
// Created: 2026-01-22
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/pkg/wire"
)

// RegisterTorrentRoutes registers torrent module routes
func RegisterTorrentRoutes(container *wire.Container, v1, v2 *gin.RouterGroup) {
	// V1 routes
	torrent := v1.Group("/torrent")
	{
		// Parse magnet URI and get file list
		torrent.POST("/parse", container.TorrentController.ParseMagnet)

		// Start download with selected files
		torrent.POST("/download", container.TorrentController.StartDownload)

		// Get download progress
		torrent.GET("/progress/:info_hash", container.TorrentController.GetProgress)

		// Pause download
		torrent.POST("/pause", container.TorrentController.PauseDownload)

		// Resume download
		torrent.POST("/resume", container.TorrentController.ResumeDownload)

		// Remove torrent
		torrent.POST("/remove", container.TorrentController.RemoveTorrent)

		// List all torrents
		torrent.GET("/list", container.TorrentController.ListTorrents)

		// Get torrent detail
		torrent.GET("/detail/:info_hash", container.TorrentController.GetTorrentDetail)

		// Serve downloaded file with streaming support
		torrent.GET("/file/:info_hash/*file_path", container.TorrentController.ServeFile)
	}
}
