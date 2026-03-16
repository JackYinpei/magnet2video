package transcode

import "testing"

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"movie.mp4", true},
		{"movie.mkv", true},
		{"movie.avi", true},
		{"movie.webm", true},
		{"movie.ts", true},
		{"movie.MKV", true},
		{"subtitle.srt", false},
		{"document.pdf", false},
		{"noext", false},
	}
	for _, tt := range tests {
		if got := IsVideoFile(tt.path); got != tt.want {
			t.Errorf("IsVideoFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestDetermineTranscodeType(t *testing.T) {
	tests := []struct {
		codec, container, want string
	}{
		{"h264", "mkv", "remux"},
		{"h264", "avi", "remux"},
		{"h264", "mp4", "transcode"},
		{"hevc", "mkv", "transcode"},
		{"H264", "MKV", "remux"},
		{"vp9", "webm", "transcode"},
	}
	for _, tt := range tests {
		if got := DetermineTranscodeType(tt.codec, tt.container); got != tt.want {
			t.Errorf("DetermineTranscodeType(%q, %q) = %q, want %q", tt.codec, tt.container, got, tt.want)
		}
	}
}
