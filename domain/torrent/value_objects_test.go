package torrent

import "testing"

func TestDownloadStatus_String(t *testing.T) {
	tests := []struct {
		status DownloadStatus
		want   string
	}{
		{DownloadPending, "pending"},
		{DownloadDownloading, "downloading"},
		{DownloadCompleted, "completed"},
		{DownloadFailed, "failed"},
		{DownloadPaused, "paused"},
		{DownloadStatus(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("DownloadStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestDownloadStatus_IsValid(t *testing.T) {
	for s := DownloadPending; s <= DownloadPaused; s++ {
		if !s.IsValid() {
			t.Errorf("DownloadStatus(%d).IsValid() = false, want true", s)
		}
	}
	if DownloadStatus(-1).IsValid() {
		t.Error("DownloadStatus(-1).IsValid() = true, want false")
	}
	if DownloadStatus(5).IsValid() {
		t.Error("DownloadStatus(5).IsValid() = true, want false")
	}
}

func TestDownloadStatus_IsTerminal(t *testing.T) {
	if !DownloadCompleted.IsTerminal() {
		t.Error("Completed should be terminal")
	}
	if !DownloadFailed.IsTerminal() {
		t.Error("Failed should be terminal")
	}
	if DownloadDownloading.IsTerminal() {
		t.Error("Downloading should not be terminal")
	}
}

func TestVisibility_String(t *testing.T) {
	tests := []struct {
		v    Visibility
		want string
	}{
		{VisibilityPrivate, "private"},
		{VisibilityInternal, "internal"},
		{VisibilityPublic, "public"},
		{Visibility(99), "unknown(99)"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("Visibility(%d).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestVisibility_IsValid(t *testing.T) {
	for v := VisibilityPrivate; v <= VisibilityPublic; v++ {
		if !v.IsValid() {
			t.Errorf("Visibility(%d).IsValid() = false, want true", v)
		}
	}
	if Visibility(3).IsValid() {
		t.Error("Visibility(3).IsValid() = true, want false")
	}
}

func TestTranscodeStatus_String(t *testing.T) {
	tests := []struct {
		s    TranscodeStatus
		want string
	}{
		{TranscodeNone, "none"},
		{TranscodePending, "pending"},
		{TranscodeProcessing, "processing"},
		{TranscodeCompleted, "completed"},
		{TranscodeFailed, "failed"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("TranscodeStatus(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestTranscodeStatus_IsValid(t *testing.T) {
	for s := TranscodeNone; s <= TranscodeFailed; s++ {
		if !s.IsValid() {
			t.Errorf("TranscodeStatus(%d).IsValid() = false, want true", s)
		}
	}
}

func TestCloudUploadStatus_String(t *testing.T) {
	tests := []struct {
		s    CloudUploadStatus
		want string
	}{
		{CloudNone, "none"},
		{CloudPending, "pending"},
		{CloudUploading, "uploading"},
		{CloudCompleted, "completed"},
		{CloudFailed, "failed"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("CloudUploadStatus(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestCloudUploadStatus_IsValid(t *testing.T) {
	for s := CloudNone; s <= CloudFailed; s++ {
		if !s.IsValid() {
			t.Errorf("CloudUploadStatus(%d).IsValid() = false, want true", s)
		}
	}
}
