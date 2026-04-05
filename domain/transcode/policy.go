// Package transcode provides the transcode bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package transcode

import (
	"path/filepath"
	"strings"
)

// IsVideoFile checks whether a file path has a video extension.
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true, ".ts": true,
	}
	return videoExts[ext]
}

// DetermineTranscodeType decides whether to remux or transcode based on codec.
// H.264 video in a non-MP4 container can be remuxed; anything else needs full transcode.
func DetermineTranscodeType(codec, containerFormat string) string {
	codec = strings.ToLower(codec)
	container := strings.ToLower(containerFormat)

	if codec == "h264" && container != "mp4" {
		return "remux"
	}
	return "transcode"
}
