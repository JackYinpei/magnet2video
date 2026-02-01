// Package dto provides data transfer object tests
// Author: Done-0
// Created: 2026-01-31
package dto

import (
	"encoding/json"
	"testing"
)

func TestParseMagnetRequest_JSON(t *testing.T) {
	req := ParseMagnetRequest{
		MagnetURI: "magnet:?xt=urn:btih:abc123",
		Trackers:  []string{"http://tracker1.com", "http://tracker2.com"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored ParseMagnetRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.MagnetURI != req.MagnetURI {
		t.Errorf("MagnetURI = %s, want %s", restored.MagnetURI, req.MagnetURI)
	}
	if len(restored.Trackers) != len(req.Trackers) {
		t.Errorf("Trackers length = %d, want %d", len(restored.Trackers), len(req.Trackers))
	}
}

func TestParseMagnetRequest_JSONFieldNames(t *testing.T) {
	req := ParseMagnetRequest{
		MagnetURI: "magnet:?xt=urn:btih:abc123",
	}

	data, _ := json.Marshal(req)
	jsonStr := string(data)

	if !containsStr(jsonStr, `"magnet_uri"`) {
		t.Error("JSON should contain field 'magnet_uri'")
	}
}

func TestStartDownloadRequest_JSON(t *testing.T) {
	req := StartDownloadRequest{
		InfoHash:      "abc123def456",
		SelectedFiles: []int{0, 2, 5},
		Trackers:      []string{"http://tracker.com"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored StartDownloadRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
	if len(restored.SelectedFiles) != 3 {
		t.Errorf("SelectedFiles length = %d, want 3", len(restored.SelectedFiles))
	}
}

func TestStartDownloadRequest_JSONFieldNames(t *testing.T) {
	req := StartDownloadRequest{
		InfoHash:      "abc123",
		SelectedFiles: []int{0},
	}

	data, _ := json.Marshal(req)
	jsonStr := string(data)

	if !containsStr(jsonStr, `"info_hash"`) {
		t.Error("JSON should contain field 'info_hash'")
	}
	if !containsStr(jsonStr, `"selected_files"`) {
		t.Error("JSON should contain field 'selected_files'")
	}
}

func TestGetProgressRequest_JSON(t *testing.T) {
	req := GetProgressRequest{
		InfoHash: "abc123def456",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored GetProgressRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
}

func TestPauseDownloadRequest_JSON(t *testing.T) {
	req := PauseDownloadRequest{
		InfoHash: "abc123def456",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored PauseDownloadRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
}

func TestResumeDownloadRequest_JSON(t *testing.T) {
	req := ResumeDownloadRequest{
		InfoHash:      "abc123def456",
		SelectedFiles: []int{0, 1},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored ResumeDownloadRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
	if len(restored.SelectedFiles) != 2 {
		t.Errorf("SelectedFiles length = %d, want 2", len(restored.SelectedFiles))
	}
}

func TestRemoveTorrentRequest_JSON(t *testing.T) {
	req := RemoveTorrentRequest{
		InfoHash:    "abc123def456",
		DeleteFiles: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored RemoveTorrentRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
	if !restored.DeleteFiles {
		t.Error("DeleteFiles should be true")
	}
}

func TestRemoveTorrentRequest_JSONFieldNames(t *testing.T) {
	req := RemoveTorrentRequest{
		InfoHash:    "abc123",
		DeleteFiles: false,
	}

	data, _ := json.Marshal(req)
	jsonStr := string(data)

	if !containsStr(jsonStr, `"info_hash"`) {
		t.Error("JSON should contain field 'info_hash'")
	}
	if !containsStr(jsonStr, `"delete_files"`) {
		t.Error("JSON should contain field 'delete_files'")
	}
}

func TestServeFileRequest_JSON(t *testing.T) {
	req := ServeFileRequest{
		InfoHash: "abc123def456",
		FilePath: "/videos/movie.mp4",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored ServeFileRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
	if restored.FilePath != req.FilePath {
		t.Errorf("FilePath = %s, want %s", restored.FilePath, req.FilePath)
	}
}

func TestServeFileRequest_JSONFieldNames(t *testing.T) {
	req := ServeFileRequest{
		InfoHash: "abc123",
		FilePath: "/test.mp4",
	}

	data, _ := json.Marshal(req)
	jsonStr := string(data)

	if !containsStr(jsonStr, `"info_hash"`) {
		t.Error("JSON should contain field 'info_hash'")
	}
	if !containsStr(jsonStr, `"file_path"`) {
		t.Error("JSON should contain field 'file_path'")
	}
}

func TestUpdateTorrentRequest_JSON(t *testing.T) {
	req := UpdateTorrentRequest{
		InfoHash:   "abc123def456",
		PosterPath: "/posters/movie.jpg",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored UpdateTorrentRequest
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.InfoHash != req.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, req.InfoHash)
	}
	if restored.PosterPath != req.PosterPath {
		t.Errorf("PosterPath = %s, want %s", restored.PosterPath, req.PosterPath)
	}
}

func TestUpdateTorrentRequest_JSONFieldNames(t *testing.T) {
	req := UpdateTorrentRequest{
		InfoHash:   "abc123",
		PosterPath: "/poster.jpg",
	}

	data, _ := json.Marshal(req)
	jsonStr := string(data)

	if !containsStr(jsonStr, `"info_hash"`) {
		t.Error("JSON should contain field 'info_hash'")
	}
	if !containsStr(jsonStr, `"poster_path"`) {
		t.Error("JSON should contain field 'poster_path'")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkParseMagnetRequest_Marshal(b *testing.B) {
	req := ParseMagnetRequest{
		MagnetURI: "magnet:?xt=urn:btih:abc123def456",
		Trackers:  []string{"http://tracker1.com", "http://tracker2.com"},
	}

	for i := 0; i < b.N; i++ {
		json.Marshal(req)
	}
}

func BenchmarkStartDownloadRequest_Marshal(b *testing.B) {
	req := StartDownloadRequest{
		InfoHash:      "abc123def456",
		SelectedFiles: []int{0, 1, 2, 3, 4},
	}

	for i := 0; i < b.N; i++ {
		json.Marshal(req)
	}
}
