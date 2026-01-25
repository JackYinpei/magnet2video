// Package controller provides torrent controller
// Author: Done-0
// Created: 2026-01-22
package controller

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

	// Remove leading slash from file path
	filePath = strings.TrimPrefix(filePath, "/")

	fullPath, err := tc.torrentService.GetFilePath(c, infoHash, filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, vo.Fail(c, err.Error(), errorx.New(errno.ErrFileNotFound, errorx.KV("path", filePath))))
		return
	}

	// Open the file
	file, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrFileNotFound, errorx.KV("path", filePath))))
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	// Determine content type
	contentType := getContentType(filePath)

	// Handle range requests for streaming
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		tc.serveRangeRequest(c, file, fileInfo, contentType, rangeHeader)
		return
	}

	// Serve the entire file
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(filePath)+"\"")

	io.Copy(c.Writer, file)
}

// serveRangeRequest handles HTTP range requests for video streaming
func (tc *TorrentController) serveRangeRequest(c *gin.Context, file *os.File, fileInfo os.FileInfo, contentType string, rangeHeader string) {
	fileSize := fileInfo.Size()

	// Parse range header
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	rangeParts := strings.Split(rangeHeader, "-")

	var start, end int64
	var err error

	if rangeParts[0] != "" {
		start, err = strconv.ParseInt(rangeParts[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid range"))))
			return
		}
	}

	if len(rangeParts) > 1 && rangeParts[1] != "" {
		end, err = strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid range"))))
			return
		}
	} else {
		// Default chunk size: 1MB
		end = start + 1024*1024 - 1
		if end >= fileSize {
			end = fileSize - 1
		}
	}

	if start >= fileSize || end >= fileSize || start > end {
		c.Header("Content-Range", "bytes */"+strconv.FormatInt(fileSize, 10))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Seek to start position
	_, err = file.Seek(start, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	contentLength := end - start + 1

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(fileSize, 10))
	c.Header("Accept-Ranges", "bytes")
	c.Status(http.StatusPartialContent)

	// Copy only the requested range
	io.CopyN(c.Writer, file, contentLength)
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
		".txt":  "text/plain",
		".srt":  "text/plain",
		".ass":  "text/plain",
		".ssa":  "text/plain",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
