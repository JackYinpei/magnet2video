// Package errno provides transcode-level error code definitions
// Author: Done-0
// Created: 2026-01-26
package errno

import (
	"github.com/Done-0/gin-scaffold/internal/utils/errorx/code"
)

// Transcode-level error codes: 50000 ~ 59999
// Used: 50001-50006
// Next available: 50007
const (
	ErrTranscodeNotNeeded    = 50001 // File does not need transcoding
	ErrTranscodeInProgress   = 50002 // Transcoding already in progress
	ErrTranscodeFailed       = 50003 // Transcoding failed
	ErrFFmpegNotFound        = 50004 // FFmpeg not found
	ErrInvalidVideoFormat    = 50005 // Invalid video format
	ErrTranscodeJobNotFound  = 50006 // Transcode job not found
)

func init() {
	code.Register(ErrTranscodeNotNeeded, "file does not need transcoding: {{.path}}")
	code.Register(ErrTranscodeInProgress, "transcoding already in progress for {{.info_hash}}")
	code.Register(ErrTranscodeFailed, "transcoding failed: {{.error}}")
	code.Register(ErrFFmpegNotFound, "ffmpeg not found at path: {{.path}}")
	code.Register(ErrInvalidVideoFormat, "invalid video format: {{.format}}")
	code.Register(ErrTranscodeJobNotFound, "transcode job not found: {{.id}}")
}
