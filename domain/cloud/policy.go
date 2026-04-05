// Package cloud provides the cloud upload bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package cloud

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// UploadPolicy encapsulates domain rules for cloud uploads.
type UploadPolicy struct {
	MaxRetries int
	PathPrefix string
}

// NewUploadPolicy creates a new upload policy.
func NewUploadPolicy(maxRetries int, pathPrefix string) *UploadPolicy {
	if pathPrefix == "" {
		pathPrefix = "torrents"
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &UploadPolicy{
		MaxRetries: maxRetries,
		PathPrefix: pathPrefix,
	}
}

// BuildCloudPath constructs the cloud object path for a file.
func (p *UploadPolicy) BuildCloudPath(infoHash, fileName string) string {
	return fmt.Sprintf("%s/%s/%s", p.PathPrefix, infoHash, fileName)
}

// DetermineContentType returns the MIME type for a file based on extension.
func (p *UploadPolicy) DetermineContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	contentTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".m4v":  "video/x-m4v",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".ts":   "video/mp2t",
		".mp3":  "audio/mpeg",
		".flac": "audio/flac",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".srt":  "application/x-subrip",
		".ass":  "text/x-ssa",
		".vtt":  "text/vtt",
	}
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// ShouldRetry returns true if the retry count is below the maximum.
func (p *UploadPolicy) ShouldRetry(retryCount int) bool {
	return retryCount < p.MaxRetries
}

// BackoffDuration returns the wait time before the next retry.
// Uses exponential backoff: 5s * 2^retryCount (5s, 10s, 20s, ...).
func (p *UploadPolicy) BackoffDuration(retryCount int) time.Duration {
	return time.Duration(5<<retryCount) * time.Second
}

// NewUploadSpec creates an UploadSpec value object.
func (p *UploadPolicy) NewUploadSpec(torrentID int64, infoHash string, fileIndex int, localPath, fileName, contentType string, fileSize int64, isTranscoded bool, creatorID int64) UploadSpec {
	cloudPath := p.BuildCloudPath(infoHash, fileName)
	if contentType == "" {
		contentType = p.DetermineContentType(localPath)
	}
	return UploadSpec{
		TorrentID:    torrentID,
		InfoHash:     infoHash,
		FileIndex:    fileIndex,
		LocalPath:    localPath,
		CloudPath:    cloudPath,
		ContentType:  contentType,
		FileSize:     fileSize,
		IsTranscoded: isTranscoded,
		CreatorID:    creatorID,
	}
}
