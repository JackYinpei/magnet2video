// Package torrent provides the torrent bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package torrent

// Torrent is the aggregate root for a BT download task.
type Torrent struct {
	ID           int64
	InfoHash     string
	Name         string
	TotalSize    int64
	Files        []TorrentFile
	PosterPath   string
	DownloadPath string
	Trackers     []string

	Status     DownloadStatus
	Progress   float64
	Visibility Visibility
	CreatorID  int64

	TranscodeStatus   TranscodeStatus
	TranscodeProgress int
	TranscodedCount   int
	TotalTranscode    int

	CloudUploadStatus   CloudUploadStatus
	CloudUploadProgress int
	CloudUploadedCount  int
	TotalCloudUpload    int

	LocalDeleted bool

	CreatedAt int64
	UpdatedAt int64
}

// NewTorrent creates a new Torrent in pending state.
func NewTorrent(infoHash, name string, totalSize int64, creatorID int64) *Torrent {
	return &Torrent{
		InfoHash:   infoHash,
		Name:       name,
		TotalSize:  totalSize,
		CreatorID:  creatorID,
		Status:     DownloadPending,
		Visibility: VisibilityPrivate,
	}
}

// StartDownload transitions from pending/failed/paused to downloading.
func (t *Torrent) StartDownload(downloadPath string) error {
	switch t.Status {
	case DownloadPending, DownloadFailed, DownloadPaused:
		t.Status = DownloadDownloading
		t.DownloadPath = downloadPath
		return nil
	default:
		return ErrInvalidStateTransition
	}
}

// MarkCompleted transitions from downloading to completed.
func (t *Torrent) MarkCompleted() error {
	if t.Status != DownloadDownloading {
		return ErrInvalidStateTransition
	}
	t.Status = DownloadCompleted
	t.Progress = 100
	return nil
}

// MarkFailed transitions the torrent to failed state.
func (t *Torrent) MarkFailed() error {
	if t.Status != DownloadDownloading {
		return ErrInvalidStateTransition
	}
	t.Status = DownloadFailed
	return nil
}

// Pause transitions from downloading to paused.
func (t *Torrent) Pause() error {
	if t.Status != DownloadDownloading {
		return ErrInvalidStateTransition
	}
	t.Status = DownloadPaused
	return nil
}

// Resume transitions from paused to downloading.
func (t *Torrent) Resume() error {
	if t.Status != DownloadPaused {
		return ErrInvalidStateTransition
	}
	t.Status = DownloadDownloading
	return nil
}

// SetVisibility updates the visibility level.
func (t *Torrent) SetVisibility(v Visibility) error {
	if !v.IsValid() {
		return ErrInvalidStateTransition
	}
	t.Visibility = v
	return nil
}

// SetPoster sets the poster path/URL.
func (t *Torrent) SetPoster(path string) {
	t.PosterPath = path
}

// MarkLocalFilesDeleted marks local files as deleted.
// Rejects if download or cloud upload is in progress.
func (t *Torrent) MarkLocalFilesDeleted() error {
	if t.Status == DownloadDownloading {
		return ErrCannotDeleteWhileDownloading
	}
	if t.CloudUploadStatus == CloudUploading {
		return ErrCannotDeleteWhileUploading
	}
	t.LocalDeleted = true
	return nil
}

// IsOwnedBy checks if the torrent belongs to the given user.
func (t *Torrent) IsOwnedBy(userID int64) bool {
	return t.CreatorID == userID
}

// IsVisibleTo checks if the torrent is visible to a given user.
func (t *Torrent) IsVisibleTo(userID int64, isAuthenticated bool) bool {
	if t.CreatorID == userID {
		return true
	}
	switch t.Visibility {
	case VisibilityPublic:
		return true
	case VisibilityInternal:
		return isAuthenticated
	default:
		return false
	}
}

// UpdateProgress updates the download progress and status.
func (t *Torrent) UpdateProgress(progress float64, status DownloadStatus) {
	t.Progress = progress
	t.Status = status
}

// FileByIndex returns the file at the given index, or nil if out of range.
func (t *Torrent) FileByIndex(index int) *TorrentFile {
	for i := range t.Files {
		if t.Files[i].Index == index {
			return &t.Files[i]
		}
	}
	return nil
}

// GetTranscodableFiles returns original video files that need transcoding.
func (t *Torrent) GetTranscodableFiles() []*TorrentFile {
	var result []*TorrentFile
	for i := range t.Files {
		if t.Files[i].NeedsTranscode() {
			result = append(result, &t.Files[i])
		}
	}
	return result
}

// GetCloudUploadableFiles returns files eligible for cloud upload.
func (t *Torrent) GetCloudUploadableFiles() []*TorrentFile {
	var result []*TorrentFile
	for i := range t.Files {
		f := &t.Files[i]
		if f.CloudUploadStatus == CloudNone && f.IsSelected {
			result = append(result, f)
		}
	}
	return result
}

// RecalculateTranscodeSummary recomputes the aggregate transcode counters from files.
func (t *Torrent) RecalculateTranscodeSummary() {
	var total, completed, failed, processing, pending int

	for i := range t.Files {
		f := &t.Files[i]
		if !f.IsOriginal() || !f.IsVideo() || !f.IsSelected {
			continue
		}
		switch f.TranscodeStatus {
		case TranscodeNone:
			// not counted
		case TranscodePending:
			total++
			pending++
		case TranscodeProcessing:
			total++
			processing++
		case TranscodeCompleted:
			total++
			completed++
		case TranscodeFailed:
			total++
			failed++
		}
	}

	t.TotalTranscode = total
	t.TranscodedCount = completed

	if total == 0 {
		t.TranscodeStatus = TranscodeNone
		t.TranscodeProgress = 0
		return
	}

	t.TranscodeProgress = int(float64(completed) * 100 / float64(total))

	switch {
	case processing > 0:
		t.TranscodeStatus = TranscodeProcessing
	case pending > 0:
		t.TranscodeStatus = TranscodePending
	case failed > 0 && completed == 0:
		t.TranscodeStatus = TranscodeFailed
	default:
		t.TranscodeStatus = TranscodeCompleted
	}
}

// RecalculateCloudSummary recomputes the aggregate cloud upload counters from files.
func (t *Torrent) RecalculateCloudSummary() {
	var total, uploaded, pending, uploading, failed, completed int

	for i := range t.Files {
		s := t.Files[i].CloudUploadStatus
		if s == CloudNone {
			continue
		}
		total++
		switch s {
		case CloudPending:
			pending++
		case CloudUploading:
			uploading++
		case CloudCompleted:
			uploaded++
			completed++
		case CloudFailed:
			failed++
		}
	}

	t.TotalCloudUpload = total
	t.CloudUploadedCount = uploaded

	if total == 0 {
		t.CloudUploadStatus = CloudNone
		t.CloudUploadProgress = 0
		return
	}

	t.CloudUploadProgress = int(float64(uploaded) * 100 / float64(total))

	switch {
	case uploading > 0:
		t.CloudUploadStatus = CloudUploading
	case pending > 0:
		t.CloudUploadStatus = CloudPending
	case failed > 0 && completed == 0:
		t.CloudUploadStatus = CloudFailed
	default:
		t.CloudUploadStatus = CloudCompleted
	}
}
