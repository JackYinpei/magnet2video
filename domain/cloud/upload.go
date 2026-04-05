// Package cloud provides the cloud upload bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package cloud

// UploadSpec is a value object describing a single cloud upload request.
type UploadSpec struct {
	TorrentID    int64
	InfoHash     string
	FileIndex    int
	LocalPath    string
	CloudPath    string
	ContentType  string
	FileSize     int64
	IsTranscoded bool
	CreatorID    int64
	RetryCount   int
}
