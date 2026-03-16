package torrent

import "testing"

func newTestTorrent() *Torrent {
	return NewTorrent("abc123", "test-torrent", 1024*1024, 1)
}

func TestNewTorrent(t *testing.T) {
	tor := newTestTorrent()
	if tor.Status != DownloadPending {
		t.Errorf("new torrent status = %v, want Pending", tor.Status)
	}
	if tor.Visibility != VisibilityPrivate {
		t.Errorf("new torrent visibility = %v, want Private", tor.Visibility)
	}
}

func TestStartDownload_FromPending(t *testing.T) {
	tor := newTestTorrent()
	if err := tor.StartDownload("/tmp/download"); err != nil {
		t.Fatalf("StartDownload from Pending failed: %v", err)
	}
	if tor.Status != DownloadDownloading {
		t.Errorf("status = %v, want Downloading", tor.Status)
	}
}

func TestStartDownload_FromFailed(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadFailed
	if err := tor.StartDownload("/tmp/download"); err != nil {
		t.Fatalf("StartDownload from Failed failed: %v", err)
	}
	if tor.Status != DownloadDownloading {
		t.Errorf("status = %v, want Downloading", tor.Status)
	}
}

func TestStartDownload_FromCompleted_ShouldFail(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadCompleted
	if err := tor.StartDownload("/tmp/download"); err != ErrInvalidStateTransition {
		t.Errorf("StartDownload from Completed err = %v, want ErrInvalidStateTransition", err)
	}
}

func TestPause_FromDownloading(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadDownloading
	if err := tor.Pause(); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	if tor.Status != DownloadPaused {
		t.Errorf("status = %v, want Paused", tor.Status)
	}
}

func TestPause_FromPaused_ShouldFail(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadPaused
	if err := tor.Pause(); err != ErrInvalidStateTransition {
		t.Errorf("Pause from Paused err = %v, want ErrInvalidStateTransition", err)
	}
}

func TestResume_FromPaused(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadPaused
	if err := tor.Resume(); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if tor.Status != DownloadDownloading {
		t.Errorf("status = %v, want Downloading", tor.Status)
	}
}

func TestResume_FromDownloading_ShouldFail(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadDownloading
	if err := tor.Resume(); err != ErrInvalidStateTransition {
		t.Errorf("Resume from Downloading err = %v, want ErrInvalidStateTransition", err)
	}
}

func TestMarkCompleted(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadDownloading
	if err := tor.MarkCompleted(); err != nil {
		t.Fatalf("MarkCompleted failed: %v", err)
	}
	if tor.Status != DownloadCompleted {
		t.Errorf("status = %v, want Completed", tor.Status)
	}
	if tor.Progress != 100 {
		t.Errorf("progress = %v, want 100", tor.Progress)
	}
}

func TestMarkCompleted_FromPending_ShouldFail(t *testing.T) {
	tor := newTestTorrent()
	if err := tor.MarkCompleted(); err != ErrInvalidStateTransition {
		t.Errorf("MarkCompleted from Pending err = %v, want ErrInvalidStateTransition", err)
	}
}

func TestMarkFailed(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadDownloading
	if err := tor.MarkFailed(); err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}
	if tor.Status != DownloadFailed {
		t.Errorf("status = %v, want Failed", tor.Status)
	}
}

func TestIsVisibleTo_Private(t *testing.T) {
	tor := newTestTorrent()
	tor.Visibility = VisibilityPrivate
	if tor.IsVisibleTo(2, true) {
		t.Error("private torrent should not be visible to other users")
	}
	if !tor.IsVisibleTo(1, true) {
		t.Error("private torrent should be visible to owner")
	}
}

func TestIsVisibleTo_Internal(t *testing.T) {
	tor := newTestTorrent()
	tor.Visibility = VisibilityInternal
	if !tor.IsVisibleTo(2, true) {
		t.Error("internal torrent should be visible to authenticated users")
	}
	if tor.IsVisibleTo(2, false) {
		t.Error("internal torrent should not be visible to unauthenticated users")
	}
}

func TestIsVisibleTo_Public(t *testing.T) {
	tor := newTestTorrent()
	tor.Visibility = VisibilityPublic
	if !tor.IsVisibleTo(2, false) {
		t.Error("public torrent should be visible to everyone")
	}
}

func TestMarkLocalFilesDeleted_WhileDownloading(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadDownloading
	if err := tor.MarkLocalFilesDeleted(); err != ErrCannotDeleteWhileDownloading {
		t.Errorf("err = %v, want ErrCannotDeleteWhileDownloading", err)
	}
}

func TestMarkLocalFilesDeleted_WhileUploading(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadCompleted
	tor.CloudUploadStatus = CloudUploading
	if err := tor.MarkLocalFilesDeleted(); err != ErrCannotDeleteWhileUploading {
		t.Errorf("err = %v, want ErrCannotDeleteWhileUploading", err)
	}
}

func TestMarkLocalFilesDeleted_Success(t *testing.T) {
	tor := newTestTorrent()
	tor.Status = DownloadCompleted
	if err := tor.MarkLocalFilesDeleted(); err != nil {
		t.Fatalf("MarkLocalFilesDeleted failed: %v", err)
	}
	if !tor.LocalDeleted {
		t.Error("LocalDeleted should be true")
	}
}

func TestRecalculateTranscodeSummary(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeCompleted},
		{Index: 1, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeProcessing},
		{Index: 2, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeFailed},
		{Index: 3, IsSelected: true, Type: FileTypeSubtitle, Source: FileSourceExtracted, TranscodeStatus: TranscodeNone},
		{Index: 4, IsSelected: false, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeNone},
	}

	tor.RecalculateTranscodeSummary()

	if tor.TotalTranscode != 3 {
		t.Errorf("TotalTranscode = %d, want 3", tor.TotalTranscode)
	}
	if tor.TranscodedCount != 1 {
		t.Errorf("TranscodedCount = %d, want 1", tor.TranscodedCount)
	}
	if tor.TranscodeStatus != TranscodeProcessing {
		t.Errorf("TranscodeStatus = %v, want Processing", tor.TranscodeStatus)
	}
	if tor.TranscodeProgress != 33 {
		t.Errorf("TranscodeProgress = %d, want 33", tor.TranscodeProgress)
	}
}

func TestRecalculateTranscodeSummary_AllCompleted(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeCompleted},
		{Index: 1, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeCompleted},
	}

	tor.RecalculateTranscodeSummary()

	if tor.TranscodeStatus != TranscodeCompleted {
		t.Errorf("TranscodeStatus = %v, want Completed", tor.TranscodeStatus)
	}
	if tor.TranscodeProgress != 100 {
		t.Errorf("TranscodeProgress = %d, want 100", tor.TranscodeProgress)
	}
}

func TestRecalculateTranscodeSummary_NoTranscodableFiles(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, IsSelected: true, Type: FileTypeOther, Source: FileSourceOriginal},
	}

	tor.RecalculateTranscodeSummary()

	if tor.TranscodeStatus != TranscodeNone {
		t.Errorf("TranscodeStatus = %v, want None", tor.TranscodeStatus)
	}
}

func TestRecalculateCloudSummary(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, CloudUploadStatus: CloudCompleted},
		{Index: 1, CloudUploadStatus: CloudUploading},
		{Index: 2, CloudUploadStatus: CloudPending},
		{Index: 3, CloudUploadStatus: CloudNone},
	}

	tor.RecalculateCloudSummary()

	if tor.TotalCloudUpload != 3 {
		t.Errorf("TotalCloudUpload = %d, want 3", tor.TotalCloudUpload)
	}
	if tor.CloudUploadedCount != 1 {
		t.Errorf("CloudUploadedCount = %d, want 1", tor.CloudUploadedCount)
	}
	if tor.CloudUploadStatus != CloudUploading {
		t.Errorf("CloudUploadStatus = %v, want Uploading", tor.CloudUploadStatus)
	}
	if tor.CloudUploadProgress != 33 {
		t.Errorf("CloudUploadProgress = %d, want 33", tor.CloudUploadProgress)
	}
}

func TestRecalculateCloudSummary_AllFailed(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, CloudUploadStatus: CloudFailed},
		{Index: 1, CloudUploadStatus: CloudFailed},
	}

	tor.RecalculateCloudSummary()

	if tor.CloudUploadStatus != CloudFailed {
		t.Errorf("CloudUploadStatus = %v, want Failed", tor.CloudUploadStatus)
	}
}

func TestRecalculateCloudSummary_MixedFailedAndCompleted(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, CloudUploadStatus: CloudCompleted},
		{Index: 1, CloudUploadStatus: CloudFailed},
	}

	tor.RecalculateCloudSummary()

	// Per original logic: failed > 0 && completed == 0 → Failed, else → Completed
	if tor.CloudUploadStatus != CloudCompleted {
		t.Errorf("CloudUploadStatus = %v, want Completed (mixed state)", tor.CloudUploadStatus)
	}
}

func TestFileByIndex(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, Path: "file0.mp4"},
		{Index: 5, Path: "file5.mp4"},
	}

	f := tor.FileByIndex(5)
	if f == nil || f.Path != "file5.mp4" {
		t.Error("FileByIndex(5) did not return correct file")
	}
	if tor.FileByIndex(99) != nil {
		t.Error("FileByIndex(99) should return nil")
	}
}

func TestGetTranscodableFiles(t *testing.T) {
	tor := newTestTorrent()
	tor.Files = []TorrentFile{
		{Index: 0, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeNone},
		{Index: 1, IsSelected: true, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeCompleted},
		{Index: 2, IsSelected: false, Type: FileTypeVideo, Source: FileSourceOriginal, TranscodeStatus: TranscodeNone},
		{Index: 3, IsSelected: true, Type: FileTypeOther, Source: FileSourceOriginal, TranscodeStatus: TranscodeNone},
		{Index: 4, IsSelected: true, Type: FileTypeVideo, Source: FileSourceTranscoded, TranscodeStatus: TranscodeNone},
	}

	files := tor.GetTranscodableFiles()
	if len(files) != 1 {
		t.Fatalf("GetTranscodableFiles() returned %d files, want 1", len(files))
	}
	if files[0].Index != 0 {
		t.Errorf("first transcodable file index = %d, want 0", files[0].Index)
	}
}

func TestSetVisibility(t *testing.T) {
	tor := newTestTorrent()
	if err := tor.SetVisibility(VisibilityPublic); err != nil {
		t.Fatalf("SetVisibility failed: %v", err)
	}
	if tor.Visibility != VisibilityPublic {
		t.Errorf("visibility = %v, want Public", tor.Visibility)
	}
}

func TestSetVisibility_Invalid(t *testing.T) {
	tor := newTestTorrent()
	if err := tor.SetVisibility(Visibility(99)); err != ErrInvalidStateTransition {
		t.Errorf("err = %v, want ErrInvalidStateTransition", err)
	}
}

func TestIsOwnedBy(t *testing.T) {
	tor := newTestTorrent()
	if !tor.IsOwnedBy(1) {
		t.Error("should be owned by user 1")
	}
	if tor.IsOwnedBy(2) {
		t.Error("should not be owned by user 2")
	}
}
