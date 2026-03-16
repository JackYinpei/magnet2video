// Package torrent provides the torrent bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package torrent

// TorrentFile represents a single file entity within a torrent aggregate.
type TorrentFile struct {
	ID        int64
	TorrentID int64
	Index     int
	Path      string
	Size      int64

	IsSelected   bool
	IsShareable  bool
	IsStreamable bool

	Type       FileType
	Source     FileSource
	ParentPath string

	TranscodeStatus TranscodeStatus
	TranscodedPath  string
	TranscodeError  string

	CloudUploadStatus CloudUploadStatus
	CloudPath         string
	CloudUploadError  string

	// Subtitle metadata
	StreamIndex   int
	Language      string
	LanguageName  string
	Title         string
	Format        string
	OriginalCodec string
}

// IsOriginal returns true if this file is an original torrent file.
func (f *TorrentFile) IsOriginal() bool {
	return f.Source == "" || f.Source == FileSourceOriginal
}

// IsVideo returns true if this file is a video file.
func (f *TorrentFile) IsVideo() bool {
	return f.Type == FileTypeVideo
}

// IsSubtitle returns true if this file is a subtitle file.
func (f *TorrentFile) IsSubtitle() bool {
	return f.Type == FileTypeSubtitle
}

// NeedsTranscode returns true if this is an original video that has not been transcoded.
func (f *TorrentFile) NeedsTranscode() bool {
	return f.IsSelected && f.IsOriginal() && f.IsVideo() && f.TranscodeStatus == TranscodeNone
}

// CanRetryCloudUpload returns true if this file can be re-queued for cloud upload.
func (f *TorrentFile) CanRetryCloudUpload() bool {
	return f.CloudUploadStatus == CloudFailed
}

// MarkTranscodePending sets the file to pending transcode.
func (f *TorrentFile) MarkTranscodePending() {
	f.TranscodeStatus = TranscodePending
}

// MarkTranscoding sets the file to currently transcoding.
func (f *TorrentFile) MarkTranscoding() {
	f.TranscodeStatus = TranscodeProcessing
}

// MarkTranscodeCompleted marks the file as successfully transcoded.
func (f *TorrentFile) MarkTranscodeCompleted(transcodedPath string) {
	f.TranscodeStatus = TranscodeCompleted
	f.TranscodedPath = transcodedPath
	f.TranscodeError = ""
}

// MarkTranscodeFailed marks the file as failed to transcode.
func (f *TorrentFile) MarkTranscodeFailed(errMsg string) {
	f.TranscodeStatus = TranscodeFailed
	f.TranscodeError = errMsg
}

// ResetTranscodeStatus resets the transcode status for admin retry.
func (f *TorrentFile) ResetTranscodeStatus() {
	f.TranscodeStatus = TranscodeNone
	f.TranscodedPath = ""
	f.TranscodeError = ""
}

// MarkCloudPending sets the file to pending cloud upload.
func (f *TorrentFile) MarkCloudPending() {
	f.CloudUploadStatus = CloudPending
}

// MarkCloudUploading sets the file to currently uploading.
func (f *TorrentFile) MarkCloudUploading() {
	f.CloudUploadStatus = CloudUploading
}

// MarkCloudCompleted marks the file as successfully uploaded.
func (f *TorrentFile) MarkCloudCompleted(cloudPath string) {
	f.CloudUploadStatus = CloudCompleted
	f.CloudPath = cloudPath
	f.CloudUploadError = ""
}

// MarkCloudFailed marks the file as failed to upload.
func (f *TorrentFile) MarkCloudFailed(errMsg string) {
	f.CloudUploadStatus = CloudFailed
	f.CloudUploadError = errMsg
}

// ResetCloudStatus resets the cloud upload status for retry.
func (f *TorrentFile) ResetCloudStatus() {
	f.CloudUploadStatus = CloudNone
	f.CloudPath = ""
	f.CloudUploadError = ""
}
