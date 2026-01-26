// Package controller provides torrent controller
// Author: Done-0
// Created: 2026-01-22
package controller

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/internal/types/errno"
	"github.com/Done-0/gin-scaffold/internal/utils/errorx"
	"github.com/Done-0/gin-scaffold/internal/utils/validator"
	"github.com/Done-0/gin-scaffold/internal/utils/vo"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
)

// TorrentController torrent HTTP controller
type TorrentController struct {
	torrentService service.TorrentService
}

// NewTorrentController creates torrent controller
func NewTorrentController(torrentService service.TorrentService) *TorrentController {
	return &TorrentController{
		torrentService: torrentService,
	}
}

// ParseMagnet handles magnet URI parsing
// @Router /api/v1/torrent/parse [post]
func (tc *TorrentController) ParseMagnet(c *gin.Context) {
	req := &dto.ParseMagnetRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.ParseMagnet(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentParseFailed, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// StartDownload handles starting a torrent download
// @Router /api/v1/torrent/download [post]
func (tc *TorrentController) StartDownload(c *gin.Context) {
	req := &dto.StartDownloadRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.StartDownload(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentAddFailed, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetProgress handles getting download progress
// @Router /api/v1/torrent/progress/:info_hash [get]
func (tc *TorrentController) GetProgress(c *gin.Context) {
	infoHash := c.Param("info_hash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "info_hash is required"))))
		return
	}

	req := &dto.GetProgressRequest{InfoHash: infoHash}
	response, err := tc.torrentService.GetProgress(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", infoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// PauseDownload handles pausing a download
// @Router /api/v1/torrent/pause [post]
func (tc *TorrentController) PauseDownload(c *gin.Context) {
	req := &dto.PauseDownloadRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.PauseDownload(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", req.InfoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ResumeDownload handles resuming a download
// @Router /api/v1/torrent/resume [post]
func (tc *TorrentController) ResumeDownload(c *gin.Context) {
	req := &dto.ResumeDownloadRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.ResumeDownload(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", req.InfoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// RemoveTorrent handles removing a torrent
// @Router /api/v1/torrent/remove [post]
func (tc *TorrentController) RemoveTorrent(c *gin.Context) {
	req := &dto.RemoveTorrentRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.RemoveTorrent(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentRemoveFailed, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ListTorrents handles listing user's torrents
// @Router /api/v1/torrent/list [get]
func (tc *TorrentController) ListTorrents(c *gin.Context) {
	response, err := tc.torrentService.ListTorrents(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ListPublicTorrents handles listing all public torrents
// @Router /api/v1/torrent/public [get]
func (tc *TorrentController) ListPublicTorrents(c *gin.Context) {
	response, err := tc.torrentService.ListPublicTorrents(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// GetTorrentDetail handles getting torrent details
// @Router /api/v1/torrent/detail/:info_hash [get]
func (tc *TorrentController) GetTorrentDetail(c *gin.Context) {
	infoHash := c.Param("info_hash")
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "info_hash is required"))))
		return
	}

	response, err := tc.torrentService.GetTorrentDetail(c, infoHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", infoHash))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// ServeFile handles serving a downloaded file
// @Router /api/v1/torrent/file/:info_hash/*file_path [get]
func (tc *TorrentController) ServeFile(c *gin.Context) {
	infoHash := c.Param("info_hash")
	filePath := c.Param("file_path")

	if infoHash == "" || filePath == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "info_hash and file_path are required"))))
		return
	}

	// Try to get file stream (with fuzzy matching support)
	reader, fileInfo, err := tc.torrentService.GetFileStream(c, infoHash, filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, vo.Fail(c, err.Error(), errorx.New(errno.ErrFileNotFound, errorx.KV("path", filePath))))
		return
	}

	// Determine content type based on actual file path
	contentType := getContentType(fileInfo.Path)

	// Set headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(fileInfo.Path)+"\"")
	
	// Delegate to http.ServeContent which handles Range requests and multipart ranges automatically
	// We use a zero time for ModTime to avoid caching issues with changing content, 
	// or we could use the torrent creation time if available.
	http.ServeContent(c.Writer, c.Request, filepath.Base(fileInfo.Path), time.Time{}, reader)
}

// getContentType determines the content type based on file extension
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	contentTypes := map[string]string{
		".mp4":  "video/mp4",
		".m4v":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".ogg":  "audio/ogg",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".txt":  "text/plain; charset=utf-8",
		".srt":  "text/plain; charset=utf-8",
		".vtt":  "text/vtt; charset=utf-8",
		".ass":  "text/plain; charset=utf-8",
		".ssa":  "text/plain; charset=utf-8",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
