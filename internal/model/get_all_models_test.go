// Package model provides model registry tests
// Author: Done-0
// Created: 2026-01-31
package model

import (
	"testing"

	"magnet2video/internal/model/torrent"
	"magnet2video/internal/model/transcode"
	"magnet2video/internal/model/user"
)

func TestGetAllModels(t *testing.T) {
	models := GetAllModels()
	if len(models) != 3 {
		t.Fatalf("GetAllModels() length = %d, want 3", len(models))
	}

	var hasUser, hasTorrent, hasTranscode bool
	for _, model := range models {
		switch model.(type) {
		case *user.User:
			hasUser = true
		case *torrent.Torrent:
			hasTorrent = true
		case *transcode.TranscodeJob:
			hasTranscode = true
		}
	}

	if !hasUser || !hasTorrent || !hasTranscode {
		t.Fatalf("GetAllModels() missing types: user=%v torrent=%v transcode=%v", hasUser, hasTorrent, hasTranscode)
	}
}
