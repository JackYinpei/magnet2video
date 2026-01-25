// Package service provides torrent service interfaces
// Author: Done-0
// Created: 2026-01-22
package service

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// TorrentService torrent service interface
type TorrentService interface {
	// ParseMagnet parses a magnet URI and returns available files
	ParseMagnet(c *gin.Context, req *dto.ParseMagnetRequest) (*vo.ParseMagnetResponse, error)
	// StartDownload starts downloading selected files
	StartDownload(c *gin.Context, req *dto.StartDownloadRequest) (*vo.StartDownloadResponse, error)
	// GetProgress returns download progress for a torrent
	GetProgress(c *gin.Context, req *dto.GetProgressRequest) (*vo.DownloadProgressResponse, error)
	// PauseDownload pauses a torrent download
	PauseDownload(c *gin.Context, req *dto.PauseDownloadRequest) (*vo.PauseDownloadResponse, error)
	// ResumeDownload resumes a paused torrent download
	ResumeDownload(c *gin.Context, req *dto.ResumeDownloadRequest) (*vo.ResumeDownloadResponse, error)
	// RemoveTorrent removes a torrent from the system
	RemoveTorrent(c *gin.Context, req *dto.RemoveTorrentRequest) (*vo.RemoveTorrentResponse, error)
	// ListTorrents lists torrents for the current user
	ListTorrents(c *gin.Context) (*vo.TorrentListResponse, error)
	// ListPublicTorrents lists all public torrents
	ListPublicTorrents(c *gin.Context) (*vo.TorrentListResponse, error)
	// GetTorrentDetail gets detailed information about a torrent
	GetTorrentDetail(c *gin.Context, infoHash string) (*vo.TorrentDetailResponse, error)
	// GetFilePath returns the file path for serving
	GetFilePath(c *gin.Context, infoHash string, filePath string) (string, error)
}
