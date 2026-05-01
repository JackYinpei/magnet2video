// Package controller provides torrent controller
// Author: Done-0
// Created: 2026-01-22
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"magnet2video/configs"
	"magnet2video/internal/cloud"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/db"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/middleware/auth"
	torrentModel "magnet2video/internal/model/torrent"
	"magnet2video/internal/queue"
	"magnet2video/internal/tmdb"
	torrentTypes "magnet2video/internal/torrent/types"
	"magnet2video/internal/types/consts"
	"magnet2video/internal/types/errno"
	"magnet2video/internal/utils/errorx"
	"magnet2video/internal/utils/validator"
	"magnet2video/internal/utils/vo"
	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/serve/service"
	pkgVo "magnet2video/pkg/vo"
	"gorm.io/gorm"
)

// TorrentController torrent HTTP controller
type TorrentController struct {
	config              *configs.Config
	torrentService      service.TorrentService
	transcodeService    service.TranscodeService
	dbManager           db.DatabaseManager
	cloudStorageManager cloud.CloudStorageManager
	queueProducer       queue.Producer
	tmdbClient          *tmdb.TMDBClient
}

// NewTorrentController creates torrent controller
func NewTorrentController(
	config *configs.Config,
	torrentService service.TorrentService,
	transcodeService service.TranscodeService,
	dbManager db.DatabaseManager,
	cloudStorageManager cloud.CloudStorageManager,
	queueProducer queue.Producer,
	tmdbClient *tmdb.TMDBClient,
) *TorrentController {
	return &TorrentController{
		config:              config,
		torrentService:      torrentService,
		transcodeService:    transcodeService,
		dbManager:           dbManager,
		cloudStorageManager: cloudStorageManager,
		queueProducer:       queueProducer,
		tmdbClient:          tmdbClient,
	}
}

// publishFileOp dispatches a filesystem-mutating command to the worker.
// Failure is logged but never blocks the API response — the caller has
// already updated DB state, and the stuck-state reaper will retry if needed.
func (tc *TorrentController) publishFileOp(ctx context.Context, op torrentTypes.FileOpMessage) {
	if tc.queueProducer == nil {
		return
	}
	data, err := json.Marshal(op)
	if err != nil {
		return
	}
	_ = tc.queueProducer.Send(ctx, torrentTypes.TopicFileOps, nil, data)
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

	for i := range response.Torrents {
		response.Torrents[i].PosterPath = tc.resolvePosterPath(response.Torrents[i].InfoHash, response.Torrents[i].PosterPath)
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

	for i := range response.Torrents {
		response.Torrents[i].PosterPath = tc.resolvePosterPath(response.Torrents[i].InfoHash, response.Torrents[i].PosterPath)
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

	response.PosterPath = tc.resolvePosterPath(infoHash, response.PosterPath)

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// SetPoster handles setting poster from an existing file
// @Router /api/v1/torrent/poster [post]
func (tc *TorrentController) SetPoster(c *gin.Context) {
	req := &dto.SetPosterRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errors := validator.Validate(req)
	if errors != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errors, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	if req.FileIndex == nil || *req.FileIndex < 0 {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "file_index is required"))))
		return
	}

	response, err := tc.torrentService.SetPosterFromFile(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", err.Error()))))
		return
	}

	response.PosterPath = tc.resolvePosterPath(response.InfoHash, response.PosterPath)
	c.JSON(http.StatusOK, vo.Success(c, response))
}

// UploadPoster handles uploading a poster file to cloud storage
// @Router /api/v1/torrent/poster/upload [post]
func (tc *TorrentController) UploadPoster(c *gin.Context) {
	if !tc.cloudStorageManager.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, nil, errorx.New(errno.ErrCloudStorageDisabled)))
		return
	}

	infoHash := strings.TrimSpace(c.PostForm("info_hash"))
	if infoHash == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "info_hash is required"))))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "poster file is required"))))
		return
	}
	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "poster file is empty"))))
		return
	}
	if !isPosterImageFile(fileHeader.Filename) {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "poster file must be an image"))))
		return
	}

	userID := auth.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized, errorx.KV("msg", "unauthorized"))))
		return
	}

	// Ensure torrent exists and is owned by current user before uploading
	var torrentRecord torrentModel.Torrent
	if err := tc.dbManager.DB().
		Select("id").
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", infoHash, userID, false).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrResourceNotFound, errorx.KV("resource", "torrent"), errorx.KV("id", infoHash))))
			return
		}
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", "failed to open poster file"))))
		return
	}
	defer file.Close()

	objectPath := tc.buildPosterObjectPath(infoHash, fileHeader.Filename)
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = getContentType(fileHeader.Filename)
	}

	if err := tc.cloudStorageManager.Upload(context.Background(), objectPath, file, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrCloudUploadFailed, errorx.KV("msg", err.Error()))))
		return
	}

	posterPath := consts.PosterPathCloudPrefix + objectPath
	response, err := tc.torrentService.UpdatePosterPath(c, infoHash, posterPath)
	if err != nil {
		_ = tc.cloudStorageManager.Delete(context.Background(), objectPath)
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", err.Error()))))
		return
	}

	response.PosterPath = tc.resolvePosterPath(infoHash, response.PosterPath)
	response.CloudPath = objectPath
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

	// In split deployment the file lives on the worker, not on this host.
	// We still try the cloud redirect path below so users with already-uploaded
	// content get a working URL, but if the file isn't in cloud yet there is
	// nothing the server can stream.
	splitDeployment := tc.config != nil && tc.config.AppConfig.Mode == configs.ModeServer

	// Try to lookup file in DB and redirect to cloud if uploaded
	if tc.cloudStorageManager.IsEnabled() {
		cleanParamPath := strings.TrimPrefix(filePath, "/")
		var torrentRecord torrentModel.Torrent
		if err := tc.dbManager.DB().Preload("Files").
			Where("info_hash = ? AND deleted = ?", infoHash, false).
			First(&torrentRecord).Error; err == nil {

			for _, file := range torrentRecord.Files {
				if (file.Source == "" || file.Source == "original") && strings.HasSuffix(file.Path, cleanParamPath) {
					if file.CloudUploadStatus == torrentModel.CloudUploadStatusCompleted && file.CloudPath != "" {
						if redirectURL := tc.buildPublicCloudURL(file.CloudPath); redirectURL != "" {
							c.Redirect(http.StatusFound, redirectURL)
							return
						}

						expiration := tc.cloudStorageManager.GetSignedURLExpiration()
						if signedURL, err := tc.cloudStorageManager.GenerateSignedURL(c.Request.Context(), file.CloudPath, expiration); err == nil {
							c.Redirect(http.StatusFound, signedURL)
							return
						}
					}
					break
				}
			}
		}
	}

	// Local streaming requires the worker to live in the same process. In
	// split (mode=server) deployment the torrent client runs on a remote
	// host, so there is nothing local to read from.
	if splitDeployment {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c,
			"local streaming is unavailable in split deployment; wait for cloud upload to complete",
			errorx.New(errno.ErrCloudStorageDisabled)))
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

// ServeTranscodedFile serves a transcoded file from the download directory
// @Router /api/v1/torrent/transcoded/*file_path [get]
func (tc *TorrentController) ServeTranscodedFile(c *gin.Context) {
	filePath := c.Param("file_path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "file_path is required"))))
		return
	}

	// Security: Only allow files with _transcoded.mp4 suffix
	if !strings.HasSuffix(filePath, "_transcoded.mp4") {
		c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "only transcoded files can be served"))))
		return
	}

	splitDeployment := tc.config != nil && tc.config.AppConfig.Mode == configs.ModeServer

	// Try to lookup file in DB and redirect to cloud if uploaded
	if tc.cloudStorageManager.IsEnabled() {
		cleanParamPath := strings.TrimPrefix(filePath, "/")
		var file torrentModel.TorrentFile
		if err := tc.dbManager.DB().
			Where("path LIKE ? AND source = ?", "%"+cleanParamPath, "transcoded").
			First(&file).Error; err == nil {

			if file.CloudUploadStatus == torrentModel.CloudUploadStatusCompleted && file.CloudPath != "" {
				if redirectURL := tc.buildPublicCloudURL(file.CloudPath); redirectURL != "" {
					c.Redirect(http.StatusFound, redirectURL)
					return
				}

				expiration := tc.cloudStorageManager.GetSignedURLExpiration()
				if signedURL, err := tc.cloudStorageManager.GenerateSignedURL(c.Request.Context(), file.CloudPath, expiration); err == nil {
					c.Redirect(http.StatusFound, signedURL)
					return
				}
			}
		}
	}

	// Local fallback requires worker + server in the same process.
	if splitDeployment {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c,
			"transcoded streaming is unavailable in split deployment; wait for cloud upload to complete",
			errorx.New(errno.ErrCloudStorageDisabled)))
		return
	}

	// Build full path (download directory + file path)
	downloadDir := tc.torrentService.GetDownloadDir()
	fullPath := filepath.Join(downloadDir, filePath)

	// Security: Prevent path traversal
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(downloadDir)) {
		c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid file path"))))
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(cleanPath)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrFileNotFound, errorx.KV("path", filePath))))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), nil))
		return
	}

	// Open file
	file, err := os.Open(cleanPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), nil))
		return
	}
	defer file.Close()

	// Set headers
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(cleanPath)+"\"")

	// Serve file with Range request support
	http.ServeContent(c.Writer, c.Request, filepath.Base(cleanPath), fileInfo.ModTime(), file)
}

// BindIMDB handles binding an IMDB ID to a torrent
// @Router /api/v1/torrent/imdb [post]
func (tc *TorrentController) BindIMDB(c *gin.Context) {
	req := &dto.BindIMDBRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	errs := validator.Validate(req)
	if errs != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errs, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	response, err := tc.torrentService.BindIMDB(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", err.Error()))))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, response))
}

// SearchTMDB handles searching TMDB for movies/TV shows
// @Router /api/v1/torrent/tmdb/search [get]
func (tc *TorrentController) SearchTMDB(c *gin.Context) {
	if tc.tmdbClient == nil || !tc.tmdbClient.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, "TMDB API not configured", nil))
		return
	}

	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "query is required"))))
		return
	}

	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}

	result, err := tc.tmdbClient.SearchMulti(query, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, result))
}

// GetTMDBImdbID handles fetching IMDB ID from TMDB for a given TMDB ID
// @Router /api/v1/torrent/tmdb/imdb/:tmdb_id [get]
func (tc *TorrentController) GetTMDBImdbID(c *gin.Context) {
	if tc.tmdbClient == nil || !tc.tmdbClient.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, "TMDB API not configured", nil))
		return
	}

	tmdbIDStr := c.Param("tmdb_id")
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil || tmdbID <= 0 {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid tmdb_id"))))
		return
	}

	mediaType := c.DefaultQuery("type", "movie")
	imdbID, err := tc.tmdbClient.GetImdbID(tmdbID, mediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, vo.Success(c, map[string]string{"imdb_id": imdbID}))
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
		".bmp":  "image/bmp",
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

func (tc *TorrentController) resolvePosterPath(infoHash string, posterPath string) string {
	if posterPath == "" {
		return ""
	}

	if strings.HasPrefix(posterPath, "http://") || strings.HasPrefix(posterPath, "https://") {
		return posterPath
	}

	if strings.HasPrefix(posterPath, consts.PosterPathLocalPrefix) {
		relPath := strings.TrimPrefix(posterPath, consts.PosterPathLocalPrefix)
		return tc.buildLocalPosterURL(infoHash, relPath)
	}

	if strings.HasPrefix(posterPath, consts.PosterPathCloudPrefix) {
		if !tc.cloudStorageManager.IsEnabled() {
			return ""
		}
		objectPath := strings.TrimPrefix(posterPath, consts.PosterPathCloudPrefix)

		if publicURL := tc.buildPublicCloudURL(objectPath); publicURL != "" {
			return publicURL
		}

		expiration := tc.cloudStorageManager.GetSignedURLExpiration()
		signedURL, err := tc.cloudStorageManager.GenerateSignedURL(context.Background(), objectPath, expiration)
		if err != nil {
			return ""
		}
		return signedURL
	}

	if strings.HasPrefix(posterPath, "/") {
		return posterPath
	}

	return tc.buildLocalPosterURL(infoHash, posterPath)
}

func (tc *TorrentController) buildLocalPosterURL(infoHash string, relPath string) string {
	escaped := url.PathEscape(relPath)
	return fmt.Sprintf("/api/v1/torrent/file/%s/%s", infoHash, escaped)
}

func (tc *TorrentController) buildPublicCloudURL(objectPath string) string {
	baseURL := strings.TrimSpace(tc.config.CloudStorageConfig.PublicURL)
	if baseURL == "" {
		return ""
	}

	trimmedPath := strings.TrimLeft(objectPath, "/")
	if trimmedPath == "" {
		return strings.TrimRight(baseURL, "/") + "/"
	}

	segments := strings.Split(trimmedPath, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.TrimRight(baseURL, "/") + "/" + strings.Join(segments, "/")
}

func (tc *TorrentController) buildPosterObjectPath(infoHash string, filename string) string {
	prefix := tc.cloudStorageManager.GetPathPrefix()
	if prefix == "" {
		prefix = "torrents"
	}
	cleanName := filepath.Base(filename)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s/posters/%s/%d_%s", prefix, infoHash, timestamp, cleanName)
}

func isPosterImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

// GetCloudURL handles generating a signed cloud URL for a file
// @Router /api/v1/torrent/cloud-url/:info_hash/:file_index [get]
func (tc *TorrentController) GetCloudURL(c *gin.Context) {
	infoHash := c.Param("info_hash")
	fileIndexStr := c.Param("file_index")

	if infoHash == "" || fileIndexStr == "" {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "info_hash and file_index are required"))))
		return
	}

	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "invalid file_index"))))
		return
	}
	if fileIndex < 0 {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "file_index out of range"))))
		return
	}

	// Check if cloud storage is enabled
	if !tc.cloudStorageManager.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, nil, errorx.New(errno.ErrCloudStorageDisabled)))
		return
	}

	// Get torrent record
	var torrentRecord torrentModel.Torrent
	if err := tc.dbManager.DB().Where("info_hash = ? AND deleted = ?", infoHash, false).First(&torrentRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", infoHash))))
		return
	}

	var file torrentModel.TorrentFile
	if err := tc.dbManager.DB().
		Where("torrent_id = ? AND `index` = ?", torrentRecord.ID, fileIndex).
		First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "file_index out of range"))))
			return
		}
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	// Check if file is uploaded to cloud
	if file.CloudUploadStatus != torrentModel.CloudUploadStatusCompleted || file.CloudPath == "" {
		c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrFileNotInCloud)))
		return
	}

	// Generate signed URL or Public URL
	if redirectURL := tc.buildPublicCloudURL(file.CloudPath); redirectURL != "" {
		c.JSON(http.StatusOK, vo.Success(c, pkgVo.CloudURLResponse{
			URL:       redirectURL,
			ExpiresAt: 0,
		}))
		return
	}

	expiration := tc.cloudStorageManager.GetSignedURLExpiration()
	signedURL, err := tc.cloudStorageManager.GenerateSignedURL(context.Background(), file.CloudPath, expiration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, nil, errorx.New(errno.ErrSignedURLFailed, errorx.KV("msg", err.Error()))))
		return
	}

	expiresAt := time.Now().Add(expiration).Unix()

	c.JSON(http.StatusOK, vo.Success(c, pkgVo.CloudURLResponse{
		URL:       signedURL,
		ExpiresAt: expiresAt,
	}))
}

// RetryCloudUpload handles retrying failed cloud uploads for a torrent
// @Router /api/v1/torrent/cloud-upload/retry [post]
func (tc *TorrentController) RetryCloudUpload(c *gin.Context) {
	if !tc.cloudStorageManager.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, nil, errorx.New(errno.ErrCloudStorageDisabled)))
		return
	}

	req := &dto.RetryCloudUploadRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	userID := auth.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized)))
		return
	}

	// Find torrent and verify ownership
	var torrentRecord torrentModel.Torrent
	if err := tc.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", req.InfoHash, userID, false).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", req.InfoHash))))
			return
		}
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	// Collect failed files and re-queue
	pathPrefix := tc.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}

	var retriedCount int
	for i, file := range torrentRecord.Files {
		if !shouldRetryCloudUploadFile(file, req.Force) {
			continue
		}

		// Resolve the worker-side local path from DB-known fields. We do not
		// os.Stat anything here — the server may not even share a filesystem
		// with the worker. The worker re-validates / falls through to its own
		// path on consume.
		localPath := resolveRetryLocalPath(torrentRecord, file)

		// Build cloud path
		fileName := filepath.Base(localPath)
		cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, torrentRecord.InfoHash, fileName)

		// Determine content type
		contentType := getContentType(localPath)
		if contentType == "application/octet-stream" {
			contentType = ""
		}

		// Build and send message — file size comes from DB (worker wrote it on
		// download.completed / transcode.completed).
		msg := cloudTypes.CloudUploadMessage{
			TorrentID:     torrentRecord.ID,
			InfoHash:      torrentRecord.InfoHash,
			FileIndex:     file.Index,
			SubtitleIndex: -1,
			LocalPath:     localPath,
			CloudPath:     cloudPath,
			ContentType:   contentType,
			FileSize:      file.Size,
			IsTranscoded:  file.Source == "transcoded",
			CreatorID:     torrentRecord.CreatorID,
			RetryCount:    0,
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		if err := tc.queueProducer.Send(context.Background(), cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
			continue
		}

		// Reset file status to pending in DB
		tc.dbManager.DB().Model(&torrentModel.TorrentFile{}).
			Where("torrent_id = ? AND `index` = ?", torrentRecord.ID, file.Index).
			Updates(map[string]interface{}{
				"cloud_upload_status": torrentModel.CloudUploadStatusPending,
				"cloud_upload_error":  "",
			})

		// Update in-memory record for summary recalculation
		torrentRecord.Files[i].CloudUploadStatus = torrentModel.CloudUploadStatusPending
		torrentRecord.Files[i].CloudUploadError = ""
		retriedCount++
	}

	if retriedCount == 0 {
		// Reconcile stale aggregate fields so UI won't keep showing a phantom failed state.
		recalculateTorrentCloudSummary(&torrentRecord)
		tc.dbManager.DB().Save(&torrentRecord)

		c.JSON(http.StatusOK, vo.Success(c, pkgVo.RetryCloudUploadResponse{
			InfoHash:     req.InfoHash,
			RetriedCount: 0,
			Message:      "No failed cloud uploads to retry",
		}))
		return
	}

	// Update torrent cloud status
	recalculateTorrentCloudSummary(&torrentRecord)
	torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusPending
	tc.dbManager.DB().Save(&torrentRecord)

	c.JSON(http.StatusOK, vo.Success(c, pkgVo.RetryCloudUploadResponse{
		InfoHash:     req.InfoHash,
		RetriedCount: retriedCount,
		Message:      fmt.Sprintf("Re-queued %d file(s) for cloud upload", retriedCount),
	}))
}

// RetryCloudUploadFile handles retrying cloud upload for a single file
// @Router /api/v1/torrent/cloud-upload/retry-file [post]
func (tc *TorrentController) RetryCloudUploadFile(c *gin.Context) {
	if !tc.cloudStorageManager.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, vo.Fail(c, nil, errorx.New(errno.ErrCloudStorageDisabled)))
		return
	}

	req := &dto.RetryCloudUploadFileRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	userID := auth.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized)))
		return
	}

	// Find torrent and verify ownership
	var torrentRecord torrentModel.Torrent
	if err := tc.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", req.InfoHash, userID, false).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", req.InfoHash))))
			return
		}
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	// Validate file index
	if req.FileIndex < 0 || req.FileIndex >= len(torrentRecord.Files) {
		c.JSON(http.StatusBadRequest, vo.Fail(c, nil, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "file_index out of range"))))
		return
	}

	file := torrentRecord.Files[req.FileIndex]

	// Apply the same eligibility check as bulk retry — single-file retry shouldn't
	// silently bypass the safety net (e.g. produce concurrent uploads on Uploading).
	if !shouldRetryCloudUploadFile(file, req.Force) {
		c.JSON(http.StatusOK, vo.Success(c, pkgVo.RetryCloudUploadResponse{
			InfoHash:     req.InfoHash,
			RetriedCount: 0,
			Message:      "File not eligible for retry (already completed, or in-flight — pass force=true to override)",
		}))
		return
	}

	// Determine local path
	localPath := resolveRetryLocalPath(torrentRecord, file)

	// Build cloud path
	pathPrefix := tc.config.CloudStorageConfig.PathPrefix
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	fileName := filepath.Base(localPath)
	cloudPath := fmt.Sprintf("%s/%s/%s", pathPrefix, torrentRecord.InfoHash, fileName)

	// Determine content type
	contentType := getContentType(localPath)
	if contentType == "application/octet-stream" {
		contentType = ""
	}

	// Outbox-style: send first; only on success update DB statuses. If sending
	// fails the file row stays as-is so a future retry can pick it up.
	// File size comes from DB — we do not os.Stat the worker's path here.
	msg := cloudTypes.CloudUploadMessage{
		TorrentID:     torrentRecord.ID,
		InfoHash:      torrentRecord.InfoHash,
		FileIndex:     file.Index,
		SubtitleIndex: -1,
		LocalPath:     localPath,
		CloudPath:     cloudPath,
		ContentType:   contentType,
		FileSize:      file.Size,
		IsTranscoded:  file.Source == "transcoded",
		CreatorID:     torrentRecord.CreatorID,
		RetryCount:    0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	if err := tc.queueProducer.Send(context.Background(), cloudTypes.TopicCloudUploadJobs, nil, msgBytes); err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	// Send succeeded → mark file Pending in DB.
	tc.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentRecord.ID, file.Index).
		Updates(map[string]interface{}{
			"cloud_upload_status": torrentModel.CloudUploadStatusPending,
			"cloud_upload_error":  "",
		})

	// Update in-memory record for summary recalculation
	torrentRecord.Files[req.FileIndex].CloudUploadStatus = torrentModel.CloudUploadStatusPending
	torrentRecord.Files[req.FileIndex].CloudUploadError = ""

	// Recalculate torrent cloud summary
	recalculateTorrentCloudSummary(&torrentRecord)
	torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusPending
	tc.dbManager.DB().Save(&torrentRecord)

	c.JSON(http.StatusOK, vo.Success(c, pkgVo.RetryCloudUploadResponse{
		InfoHash:     req.InfoHash,
		RetriedCount: 1,
		Message:      fmt.Sprintf("Re-queued file #%d for cloud upload", req.FileIndex),
	}))
}

// RequeueTranscode re-runs the transcode pipeline for a torrent (creator only).
// @Router /api/v1/torrent/transcode/requeue [post]
func (tc *TorrentController) RequeueTranscode(c *gin.Context) {
	req := &dto.RequeueTranscodeRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}
	if errs := validator.Validate(req); errs != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, errs, errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "validation failed"))))
		return
	}

	userID := auth.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized)))
		return
	}

	resp, err := tc.transcodeService.RequeueTranscode(c, req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer, errorx.KV("msg", err.Error()))))
		return
	}
	c.JSON(http.StatusOK, vo.Success(c, resp))
}

// DeleteLocalFiles handles deleting local files for a torrent after cloud upload
// @Router /api/v1/torrent/delete-local [post]
func (tc *TorrentController) DeleteLocalFiles(c *gin.Context) {
	req := &dto.DeleteLocalFilesRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, vo.Fail(c, err.Error(), errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "bind JSON failed"))))
		return
	}

	userID := auth.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized)))
		return
	}

	// Find torrent and verify ownership
	var torrentRecord torrentModel.Torrent
	if err := tc.dbManager.DB().
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index` asc")
		}).
		Where("info_hash = ? AND creator_id = ? AND deleted = ?", req.InfoHash, userID, false).
		First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, vo.Fail(c, nil, errorx.New(errno.ErrTorrentNotFound, errorx.KV("info_hash", req.InfoHash))))
			return
		}
		c.JSON(http.StatusInternalServerError, vo.Fail(c, err.Error(), errorx.New(errno.ErrInternalServer)))
		return
	}

	// Check: all cloud-upload files must be completed
	for _, file := range torrentRecord.Files {
		if file.CloudUploadStatus == torrentModel.CloudUploadStatusPending ||
			file.CloudUploadStatus == torrentModel.CloudUploadStatusUploading {
			c.JSON(http.StatusBadRequest, vo.Fail(c, "cannot delete local files while uploading", errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "cloud upload in progress"))))
			return
		}
	}

	// Check: torrent must not be downloading
	if torrentRecord.Status == torrentModel.StatusDownloading {
		c.JSON(http.StatusBadRequest, vo.Fail(c, "cannot delete local files while downloading", errorx.New(errno.ErrInvalidParams, errorx.KV("msg", "download in progress"))))
		return
	}

	// Check: already deleted
	if torrentRecord.LocalDeleted {
		c.JSON(http.StatusOK, vo.Success(c, map[string]string{"message": "local files already deleted"}))
		return
	}

	// Delegate the actual deletion to the worker. We dispatch one
	// delete-torrent-dir op (covers the {download_path}/{name} subtree) plus
	// one delete-paths op for any derived files that live elsewhere — the
	// worker enforces the download-dir boundary on every path.
	if torrentRecord.DownloadPath != "" {
		tc.publishFileOp(c.Request.Context(), torrentTypes.FileOpMessage{
			Op:           torrentTypes.FileOpDeleteTorrentDir,
			OpID:         eventTypes.GenerateEventID(),
			TorrentID:    torrentRecord.ID,
			InfoHash:     req.InfoHash,
			DownloadPath: torrentRecord.DownloadPath,
			TorrentName:  torrentRecord.Name,
			CreatorID:    userID,
		})

		var derivedPaths []string
		for _, f := range torrentRecord.Files {
			if (f.Source == "transcoded" || f.Source == "extracted") && f.Path != "" {
				derivedPaths = append(derivedPaths, f.Path)
			}
		}
		if len(derivedPaths) > 0 {
			tc.publishFileOp(c.Request.Context(), torrentTypes.FileOpMessage{
				Op:           torrentTypes.FileOpDeletePaths,
				OpID:         eventTypes.GenerateEventID(),
				TorrentID:    torrentRecord.ID,
				InfoHash:     req.InfoHash,
				DownloadPath: torrentRecord.DownloadPath,
				Paths:        derivedPaths,
				CreatorID:    userID,
			})
		}
	}

	// Mark as local deleted in DB and pause to prevent re-download on restart
	updates := map[string]interface{}{
		"local_deleted": true,
	}
	if torrentRecord.Status == torrentModel.StatusDownloading || torrentRecord.Status == torrentModel.StatusPending {
		updates["status"] = torrentModel.StatusPaused
	}
	tc.dbManager.DB().Model(&torrentModel.Torrent{}).
		Where("id = ?", torrentRecord.ID).
		Updates(updates)

	// Pause the torrent in the torrent manager if active, to prevent re-seeding
	pauseReq := &dto.PauseDownloadRequest{InfoHash: req.InfoHash}
	_, _ = tc.torrentService.PauseDownload(c, pauseReq)

	c.JSON(http.StatusOK, vo.Success(c, map[string]string{"message": "local files deleted"}))
}

// shouldRetryCloudUploadFile decides whether a file is eligible for re-queueing.
//
// Default (force=false):
//   - Completed:           never retry (object already in cloud)
//   - Failed:              retry — this is the obvious "fix it" case
//   - Pending / Uploading: skip — worker may genuinely still be doing it.
//     Re-queueing here would cause two concurrent uploads racing each other
//     and one of them clobbering the other's CloudPath / status. Use force=true
//     to override (e.g. when the user knows the queue message was actually lost).
//   - None:                retry only if the file is a likely cloud candidate
//     (selected video/subtitle/transcoded/extracted that was never attempted)
//
// Force=true:
//   - Treat Pending / Uploading the same as Failed (will be re-queued).
//   - Completed still won't be retried (no point — it's done).
//   - None falls back to the candidate heuristic, same as default.
func shouldRetryCloudUploadFile(file torrentModel.TorrentFile, force bool) bool {
	if file.CloudUploadStatus == torrentModel.CloudUploadStatusCompleted {
		return false
	}

	if file.CloudUploadStatus == torrentModel.CloudUploadStatusFailed {
		return true
	}

	if file.CloudUploadStatus == torrentModel.CloudUploadStatusPending ||
		file.CloudUploadStatus == torrentModel.CloudUploadStatusUploading {
		return force
	}

	// Status == None below this line. CloudPath being set means a previous attempt
	// at least chose a target — not a fresh candidate, so skip.
	if file.CloudUploadStatus != torrentModel.CloudUploadStatusNone || file.CloudPath != "" {
		return false
	}

	fileType := file.Type
	if fileType == "" {
		fileType = torrentModel.DetectFileType(file.Path)
	}

	switch file.Source {
	case "transcoded", "extracted":
		return true
	case "original", "":
		return file.IsSelected &&
			file.TranscodeStatus == torrentModel.TranscodeStatusNone &&
			(fileType == "video" || fileType == "subtitle")
	default:
		return false
	}
}

// resolveRetryLocalPath returns the worker-side canonical path for a file.
// It does NOT touch the local filesystem — server and worker may not even
// share one. We trust file.Path: for original files it's anacrolix's relative
// path including the torrent root directory; for derived files (transcoded /
// extracted) the worker recorded it at creation time. If the path is already
// absolute, leave it alone.
func resolveRetryLocalPath(torrentRecord torrentModel.Torrent, file torrentModel.TorrentFile) string {
	if filepath.IsAbs(file.Path) || torrentRecord.DownloadPath == "" {
		return file.Path
	}
	return filepath.Join(torrentRecord.DownloadPath, file.Path)
}

func recalculateTorrentCloudSummary(torrentRecord *torrentModel.Torrent) {
	if torrentRecord == nil {
		return
	}

	var pending, uploading, completed, failed int
	var total, uploaded int
	for _, file := range torrentRecord.Files {
		if file.CloudUploadStatus != torrentModel.CloudUploadStatusNone {
			total++
		}
		if file.CloudUploadStatus == torrentModel.CloudUploadStatusCompleted {
			uploaded++
		}
		switch file.CloudUploadStatus {
		case torrentModel.CloudUploadStatusPending:
			pending++
		case torrentModel.CloudUploadStatusUploading:
			uploading++
		case torrentModel.CloudUploadStatusCompleted:
			completed++
		case torrentModel.CloudUploadStatusFailed:
			failed++
		}
	}

	torrentRecord.TotalCloudUpload = total
	torrentRecord.CloudUploadedCount = uploaded
	if total > 0 {
		torrentRecord.CloudUploadProgress = int(float64(uploaded) * 100 / float64(total))
	} else {
		torrentRecord.CloudUploadProgress = 0
	}

	switch {
	case uploading > 0:
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusUploading
	case pending > 0:
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusPending
	case failed > 0 && completed == 0:
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusFailed
	case completed > 0:
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusCompleted
		torrentRecord.CloudUploadProgress = 100
	default:
		torrentRecord.CloudUploadStatus = torrentModel.CloudUploadStatusNone
	}
}
