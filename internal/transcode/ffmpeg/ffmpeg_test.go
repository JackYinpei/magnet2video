// Package ffmpeg provides FFmpeg helper utilities tests
// Author: Done-0
// Created: 2026-01-31
package ffmpeg

import (
	"path/filepath"
	"testing"
)

func TestDetermineTranscodeType(t *testing.T) {
	ff := New("", "")

	tests := []struct {
		name string
		info *VideoInfo
		path string
		want TranscodeType
	}{
		{
			name: "mp4 with h264",
			info: &VideoInfo{Codec: "h264"},
			path: "video.mp4",
			want: TranscodeTypeNone,
		},
		{
			name: "webm with vp9",
			info: &VideoInfo{Codec: "vp9"},
			path: "video.webm",
			want: TranscodeTypeNone,
		},
		{
			name: "mkv with h264",
			info: &VideoInfo{Codec: "h264"},
			path: "video.mkv",
			want: TranscodeTypeRemux,
		},
		{
			name: "avi with vp9",
			info: &VideoInfo{Codec: "vp9"},
			path: "video.avi",
			want: TranscodeTypeRemux,
		},
		{
			name: "mkv with mpeg2",
			info: &VideoInfo{Codec: "mpeg2video"},
			path: "video.mkv",
			want: TranscodeTypeTranscode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ff.DetermineTranscodeType(tt.info, tt.path); got != tt.want {
				t.Fatalf("DetermineTranscodeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateOutputPath(t *testing.T) {
	input := filepath.Join("/tmp", "sample.mkv")
	want := filepath.Join("/tmp", "sample_transcoded.mp4")

	if got := GenerateOutputPath(input); got != want {
		t.Fatalf("GenerateOutputPath() = %q, want %q", got, want)
	}
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"movie.mkv", true},
		{"clip.MP4", true},
		{"document.txt", false},
	}

	for _, tt := range tests {
		if got := IsVideoFile(tt.path); got != tt.want {
			t.Fatalf("IsVideoFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestNeedsTranscoding(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"movie.mkv", true},
		{"clip.MP4", false},
		{"trailer.avi", true},
		{"sample.webm", false},
	}

	for _, tt := range tests {
		if got := NeedsTranscoding(tt.path); got != tt.want {
			t.Fatalf("NeedsTranscoding(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
