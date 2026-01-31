// Package internal provides torrent manager utilities tests
// Author: Done-0
// Created: 2026-01-31
package internal

import "testing"

func TestIsStreamableFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"movie.mp4", true},
		{"clip.m4v", true},
		{"sample.webm", true},
		{"movie.mkv", false},
		{"document.txt", false},
	}

	for _, tt := range tests {
		if got := isStreamableFile(tt.path); got != tt.want {
			t.Fatalf("isStreamableFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
