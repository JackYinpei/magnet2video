package cloud

import (
	"testing"
	"time"
)

func TestNewUploadPolicy_Defaults(t *testing.T) {
	p := NewUploadPolicy(0, "")
	if p.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", p.MaxRetries)
	}
	if p.PathPrefix != "torrents" {
		t.Errorf("PathPrefix = %q, want torrents", p.PathPrefix)
	}
}

func TestBuildCloudPath(t *testing.T) {
	p := NewUploadPolicy(3, "media")
	got := p.BuildCloudPath("abc123", "video.mp4")
	want := "media/abc123/video.mp4"
	if got != want {
		t.Errorf("BuildCloudPath = %q, want %q", got, want)
	}
}

func TestDetermineContentType(t *testing.T) {
	p := NewUploadPolicy(3, "")
	tests := []struct {
		path string
		want string
	}{
		{"video.mp4", "video/mp4"},
		{"video.mkv", "video/x-matroska"},
		{"video.webm", "video/webm"},
		{"sub.srt", "application/x-subrip"},
		{"sub.ass", "text/x-ssa"},
		{"sub.vtt", "text/vtt"},
		{"image.jpg", "image/jpeg"},
		{"image.png", "image/png"},
		{"unknown.xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		if got := p.DetermineContentType(tt.path); got != tt.want {
			t.Errorf("DetermineContentType(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	p := NewUploadPolicy(3, "")
	if !p.ShouldRetry(0) {
		t.Error("ShouldRetry(0) = false, want true")
	}
	if !p.ShouldRetry(2) {
		t.Error("ShouldRetry(2) = false, want true")
	}
	if p.ShouldRetry(3) {
		t.Error("ShouldRetry(3) = true, want false")
	}
	if p.ShouldRetry(10) {
		t.Error("ShouldRetry(10) = true, want false")
	}
}

func TestBackoffDuration(t *testing.T) {
	p := NewUploadPolicy(3, "")
	tests := []struct {
		retry int
		want  time.Duration
	}{
		{0, 5 * time.Second},
		{1, 10 * time.Second},
		{2, 20 * time.Second},
		{3, 40 * time.Second},
	}
	for _, tt := range tests {
		if got := p.BackoffDuration(tt.retry); got != tt.want {
			t.Errorf("BackoffDuration(%d) = %v, want %v", tt.retry, got, tt.want)
		}
	}
}

func TestNewUploadSpec(t *testing.T) {
	p := NewUploadPolicy(3, "files")
	spec := p.NewUploadSpec(1, "abc123", 0, "/tmp/video.mp4", "video.mp4", "", 1024, true, 42)

	if spec.CloudPath != "files/abc123/video.mp4" {
		t.Errorf("CloudPath = %q", spec.CloudPath)
	}
	if spec.ContentType != "video/mp4" {
		t.Errorf("ContentType = %q, want video/mp4", spec.ContentType)
	}
	if spec.TorrentID != 1 || spec.FileIndex != 0 || spec.CreatorID != 42 {
		t.Error("spec fields not set correctly")
	}
}

func TestNewUploadSpec_ExplicitContentType(t *testing.T) {
	p := NewUploadPolicy(3, "files")
	spec := p.NewUploadSpec(1, "abc123", 0, "/tmp/file", "file", "custom/type", 1024, false, 1)

	if spec.ContentType != "custom/type" {
		t.Errorf("ContentType = %q, want custom/type", spec.ContentType)
	}
}
