// Package torrent provides torrent-related data model definitions
// Author: Done-0
// Created: 2026-02-03
package torrent

import (
	"path/filepath"
	"strings"
)

// DetectFileType returns the file type based on extension.
// Allowed types: "video", "subtitle", "other".
func DetectFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := map[string]bool{".mkv": true, ".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true, ".webm": true, ".m4v": true, ".ts": true}
	subtitleExts := map[string]bool{".srt": true, ".ass": true, ".ssa": true, ".vtt": true, ".sub": true}

	switch {
	case videoExts[ext]:
		return "video"
	case subtitleExts[ext]:
		return "subtitle"
	default:
		return "other"
	}
}
