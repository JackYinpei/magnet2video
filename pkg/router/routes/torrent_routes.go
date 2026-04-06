// Package routes provides torrent route registration functionality
// Author: Done-0
// Created: 2026-01-22
package routes

import (
	"github.com/gin-gonic/gin"

	"magnet2video/internal/middleware/auth"
	"magnet2video/pkg/wire"
)

// RegisterTorrentRoutes registers torrent module routes
func RegisterTorrentRoutes(container *wire.Container, v1, v2 *gin.RouterGroup) {
	// Public routes (no auth required) - for browsing public resources
	publicTorrent := v1.Group("/torrent")
	{
		// List public/internal torrents (uses optional auth to determine visibility)
		publicTorrent.GET("/public", auth.OptionalJWTMiddleware(), container.TorrentController.ListPublicTorrents)

		// Get torrent detail (public access for shared torrents)
		publicTorrent.GET("/detail/:info_hash", container.TorrentController.GetTorrentDetail)

		// Serve downloaded file with streaming support (public access for shared files)
		publicTorrent.GET("/file/:info_hash/*file_path", container.TorrentController.ServeFile)

		// Serve transcoded file from download directory
		publicTorrent.GET("/transcoded/*file_path", container.TorrentController.ServeTranscodedFile)

		// Get signed cloud URL for a file
		publicTorrent.GET("/cloud-url/:info_hash/:file_index", container.TorrentController.GetCloudURL)
	}

	// Protected routes (auth required) - for managing own resources
	protectedTorrent := v1.Group("/torrent")
	protectedTorrent.Use(auth.JWTMiddleware())
	{
		// Parse magnet URI and get file list
		protectedTorrent.POST("/parse", container.TorrentController.ParseMagnet)

		// Start download with selected files
		protectedTorrent.POST("/download", container.TorrentController.StartDownload)

		// Get download progress
		protectedTorrent.GET("/progress/:info_hash", container.TorrentController.GetProgress)

		// Pause download
		protectedTorrent.POST("/pause", container.TorrentController.PauseDownload)

		// Resume download
		protectedTorrent.POST("/resume", container.TorrentController.ResumeDownload)

		// Remove torrent
		protectedTorrent.POST("/remove", container.TorrentController.RemoveTorrent)

		// List user's own torrents
		protectedTorrent.GET("/list", container.TorrentController.ListTorrents)

		// Set poster from existing file
		protectedTorrent.POST("/poster", container.TorrentController.SetPoster)

		// Upload poster to cloud storage
		protectedTorrent.POST("/poster/upload", container.TorrentController.UploadPoster)

		// Retry failed cloud uploads
		protectedTorrent.POST("/cloud-upload/retry", container.TorrentController.RetryCloudUpload)

		// Retry cloud upload for a single file
		protectedTorrent.POST("/cloud-upload/retry-file", container.TorrentController.RetryCloudUploadFile)

		// Delete local files after cloud upload
		protectedTorrent.POST("/delete-local", container.TorrentController.DeleteLocalFiles)

		// Bind IMDB ID to torrent
		protectedTorrent.POST("/imdb", container.TorrentController.BindIMDB)

		// Search TMDB for movies/TV shows
		protectedTorrent.GET("/tmdb/search", container.TorrentController.SearchTMDB)

		// Get IMDB ID from TMDB ID
		protectedTorrent.GET("/tmdb/imdb/:tmdb_id", container.TorrentController.GetTMDBImdbID)
	}
}
