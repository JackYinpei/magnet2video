// Package torrent provides the torrent bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package torrent

import "fmt"

// DownloadStatus represents the download state of a torrent.
type DownloadStatus int

const (
	DownloadPending     DownloadStatus = 0
	DownloadDownloading DownloadStatus = 1
	DownloadCompleted   DownloadStatus = 2
	DownloadFailed      DownloadStatus = 3
	DownloadPaused      DownloadStatus = 4
)

// String returns the human-readable name.
func (s DownloadStatus) String() string {
	switch s {
	case DownloadPending:
		return "pending"
	case DownloadDownloading:
		return "downloading"
	case DownloadCompleted:
		return "completed"
	case DownloadFailed:
		return "failed"
	case DownloadPaused:
		return "paused"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsValid checks whether the status value is defined.
func (s DownloadStatus) IsValid() bool {
	return s >= DownloadPending && s <= DownloadPaused
}

// IsTerminal returns true if no further download progress is expected.
func (s DownloadStatus) IsTerminal() bool {
	return s == DownloadCompleted || s == DownloadFailed
}

// Visibility represents who can see the torrent.
type Visibility int

const (
	VisibilityPrivate  Visibility = 0
	VisibilityInternal Visibility = 1
	VisibilityPublic   Visibility = 2
)

// String returns the human-readable name.
func (v Visibility) String() string {
	switch v {
	case VisibilityPrivate:
		return "private"
	case VisibilityInternal:
		return "internal"
	case VisibilityPublic:
		return "public"
	default:
		return fmt.Sprintf("unknown(%d)", int(v))
	}
}

// IsValid checks whether the visibility value is defined.
func (v Visibility) IsValid() bool {
	return v >= VisibilityPrivate && v <= VisibilityPublic
}

// TranscodeStatus represents the transcoding state of a file or torrent.
type TranscodeStatus int

const (
	TranscodeNone       TranscodeStatus = 0
	TranscodePending    TranscodeStatus = 1
	TranscodeProcessing TranscodeStatus = 2
	TranscodeCompleted  TranscodeStatus = 3
	TranscodeFailed     TranscodeStatus = 4
)

// String returns the human-readable name.
func (s TranscodeStatus) String() string {
	switch s {
	case TranscodeNone:
		return "none"
	case TranscodePending:
		return "pending"
	case TranscodeProcessing:
		return "processing"
	case TranscodeCompleted:
		return "completed"
	case TranscodeFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsValid checks whether the status value is defined.
func (s TranscodeStatus) IsValid() bool {
	return s >= TranscodeNone && s <= TranscodeFailed
}

// CloudUploadStatus represents the cloud upload state of a file or torrent.
type CloudUploadStatus int

const (
	CloudNone      CloudUploadStatus = 0
	CloudPending   CloudUploadStatus = 1
	CloudUploading CloudUploadStatus = 2
	CloudCompleted CloudUploadStatus = 3
	CloudFailed    CloudUploadStatus = 4
)

// String returns the human-readable name.
func (s CloudUploadStatus) String() string {
	switch s {
	case CloudNone:
		return "none"
	case CloudPending:
		return "pending"
	case CloudUploading:
		return "uploading"
	case CloudCompleted:
		return "completed"
	case CloudFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsValid checks whether the status value is defined.
func (s CloudUploadStatus) IsValid() bool {
	return s >= CloudNone && s <= CloudFailed
}

// FileType represents the type of a torrent file.
type FileType string

const (
	FileTypeVideo    FileType = "video"
	FileTypeSubtitle FileType = "subtitle"
	FileTypeOther    FileType = "other"
)

// FileSource represents the origin of a torrent file.
type FileSource string

const (
	FileSourceOriginal   FileSource = "original"
	FileSourceTranscoded FileSource = "transcoded"
	FileSourceExtracted  FileSource = "extracted"
)
