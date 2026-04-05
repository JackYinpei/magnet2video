package torrent

import "testing"

func TestTorrentFile_IsOriginal(t *testing.T) {
	tests := []struct {
		source FileSource
		want   bool
	}{
		{"", true},
		{FileSourceOriginal, true},
		{FileSourceTranscoded, false},
		{FileSourceExtracted, false},
	}
	for _, tt := range tests {
		f := &TorrentFile{Source: tt.source}
		if got := f.IsOriginal(); got != tt.want {
			t.Errorf("IsOriginal() with source %q = %v, want %v", tt.source, got, tt.want)
		}
	}
}

func TestTorrentFile_IsVideo(t *testing.T) {
	f := &TorrentFile{Type: FileTypeVideo}
	if !f.IsVideo() {
		t.Error("IsVideo() = false, want true")
	}
	f.Type = FileTypeOther
	if f.IsVideo() {
		t.Error("IsVideo() = true for 'other', want false")
	}
}

func TestTorrentFile_NeedsTranscode(t *testing.T) {
	f := &TorrentFile{
		IsSelected:      true,
		Type:            FileTypeVideo,
		Source:          FileSourceOriginal,
		TranscodeStatus: TranscodeNone,
	}
	if !f.NeedsTranscode() {
		t.Error("NeedsTranscode() = false, want true")
	}

	// Already transcoded
	f.TranscodeStatus = TranscodeCompleted
	if f.NeedsTranscode() {
		t.Error("NeedsTranscode() = true for completed, want false")
	}
}

func TestTorrentFile_TranscodeStatusFlow(t *testing.T) {
	f := &TorrentFile{}

	f.MarkTranscodePending()
	if f.TranscodeStatus != TranscodePending {
		t.Errorf("status = %v, want Pending", f.TranscodeStatus)
	}

	f.MarkTranscoding()
	if f.TranscodeStatus != TranscodeProcessing {
		t.Errorf("status = %v, want Processing", f.TranscodeStatus)
	}

	f.MarkTranscodeCompleted("/output/file.mp4")
	if f.TranscodeStatus != TranscodeCompleted {
		t.Errorf("status = %v, want Completed", f.TranscodeStatus)
	}
	if f.TranscodedPath != "/output/file.mp4" {
		t.Errorf("transcoded path = %q", f.TranscodedPath)
	}
	if f.TranscodeError != "" {
		t.Error("error should be cleared on completion")
	}
}

func TestTorrentFile_TranscodeFailed(t *testing.T) {
	f := &TorrentFile{}
	f.MarkTranscodeFailed("codec not supported")
	if f.TranscodeStatus != TranscodeFailed {
		t.Errorf("status = %v, want Failed", f.TranscodeStatus)
	}
	if f.TranscodeError != "codec not supported" {
		t.Errorf("error = %q", f.TranscodeError)
	}
}

func TestTorrentFile_ResetTranscodeStatus(t *testing.T) {
	f := &TorrentFile{
		TranscodeStatus: TranscodeFailed,
		TranscodedPath:  "/old/path.mp4",
		TranscodeError:  "some error",
	}
	f.ResetTranscodeStatus()
	if f.TranscodeStatus != TranscodeNone || f.TranscodedPath != "" || f.TranscodeError != "" {
		t.Error("reset should clear all transcode fields")
	}
}

func TestTorrentFile_CloudStatusFlow(t *testing.T) {
	f := &TorrentFile{}

	f.MarkCloudPending()
	if f.CloudUploadStatus != CloudPending {
		t.Errorf("status = %v, want Pending", f.CloudUploadStatus)
	}

	f.MarkCloudUploading()
	if f.CloudUploadStatus != CloudUploading {
		t.Errorf("status = %v, want Uploading", f.CloudUploadStatus)
	}

	f.MarkCloudCompleted("torrents/abc/file.mp4")
	if f.CloudUploadStatus != CloudCompleted {
		t.Errorf("status = %v, want Completed", f.CloudUploadStatus)
	}
	if f.CloudPath != "torrents/abc/file.mp4" {
		t.Errorf("cloud path = %q", f.CloudPath)
	}
	if f.CloudUploadError != "" {
		t.Error("error should be cleared on completion")
	}
}

func TestTorrentFile_CloudFailed(t *testing.T) {
	f := &TorrentFile{}
	f.MarkCloudFailed("network timeout")
	if f.CloudUploadStatus != CloudFailed {
		t.Errorf("status = %v, want Failed", f.CloudUploadStatus)
	}
	if f.CloudUploadError != "network timeout" {
		t.Errorf("error = %q", f.CloudUploadError)
	}
}

func TestTorrentFile_CanRetryCloudUpload(t *testing.T) {
	f := &TorrentFile{CloudUploadStatus: CloudFailed}
	if !f.CanRetryCloudUpload() {
		t.Error("CanRetryCloudUpload() = false for Failed, want true")
	}
	f.CloudUploadStatus = CloudCompleted
	if f.CanRetryCloudUpload() {
		t.Error("CanRetryCloudUpload() = true for Completed, want false")
	}
}

func TestTorrentFile_ResetCloudStatus(t *testing.T) {
	f := &TorrentFile{
		CloudUploadStatus: CloudFailed,
		CloudPath:         "old/path",
		CloudUploadError:  "error",
	}
	f.ResetCloudStatus()
	if f.CloudUploadStatus != CloudNone || f.CloudPath != "" || f.CloudUploadError != "" {
		t.Error("reset should clear all cloud fields")
	}
}
