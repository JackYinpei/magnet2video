// Package torrent provides torrent-related data model tests
// Author: Done-0
// Created: 2026-01-31
package torrent

import (
	"encoding/json"
	"testing"
)

func TestTorrentFiles_ScanNil(t *testing.T) {
	var tf TorrentFiles
	err := tf.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) error = %v", err)
	}
	if tf == nil {
		t.Error("Scan(nil) should initialize empty slice")
	}
	if len(tf) != 0 {
		t.Errorf("Scan(nil) should create empty slice, got len=%d", len(tf))
	}
}

func TestTorrentFiles_ScanBytes(t *testing.T) {
	var tf TorrentFiles
	data := []byte(`[{"path": "video.mp4", "size": 1024, "is_selected": true}]`)

	err := tf.Scan(data)
	if err != nil {
		t.Errorf("Scan(bytes) error = %v", err)
	}

	if len(tf) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(tf))
	}
	if tf[0].Path != "video.mp4" {
		t.Errorf("Path = %s, want video.mp4", tf[0].Path)
	}
	if tf[0].Size != 1024 {
		t.Errorf("Size = %d, want 1024", tf[0].Size)
	}
	if !tf[0].IsSelected {
		t.Error("IsSelected should be true")
	}
}

func TestTorrentFiles_ScanInvalidBytes(t *testing.T) {
	var tf TorrentFiles
	err := tf.Scan([]byte(`invalid json`))
	if err == nil {
		t.Error("Scan(invalid json) should return error")
	}
}

func TestTorrentFiles_ScanUnsupportedType(t *testing.T) {
	var tf TorrentFiles
	err := tf.Scan(12345)
	if err == nil {
		t.Error("Scan(int) should return error")
	}
}

func TestTorrentFiles_Value(t *testing.T) {
	tf := TorrentFiles{
		{Path: "file1.mp4", Size: 1000, IsSelected: true},
		{Path: "file2.mkv", Size: 2000, IsSelected: false},
	}

	value, err := tf.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}

	bytes, ok := value.([]byte)
	if !ok {
		t.Fatal("Value() should return []byte")
	}

	var result []TorrentFile
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Errorf("Value() returned invalid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result))
	}
}

func TestTorrentFiles_RoundTrip(t *testing.T) {
	original := TorrentFiles{
		{
			Path:            "movie.mp4",
			Size:            1073741824,
			IsSelected:      true,
			IsStreamable:    true,
			TranscodeStatus: TranscodeStatusCompleted,
			TranscodedPath:  "/output/movie.mp4",
		},
	}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	var restored TorrentFiles
	err = restored.Scan(value)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(restored) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(restored))
	}
	if restored[0].Path != original[0].Path {
		t.Errorf("Path mismatch: got %s, want %s", restored[0].Path, original[0].Path)
	}
	if restored[0].TranscodeStatus != original[0].TranscodeStatus {
		t.Errorf("TranscodeStatus mismatch: got %d, want %d",
			restored[0].TranscodeStatus, original[0].TranscodeStatus)
	}
}

func TestStringSlice_ScanNil(t *testing.T) {
	var ss StringSlice
	err := ss.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) error = %v", err)
	}
	if ss == nil {
		t.Error("Scan(nil) should initialize empty slice")
	}
}

func TestStringSlice_ScanBytes(t *testing.T) {
	var ss StringSlice
	data := []byte(`["tracker1", "tracker2", "tracker3"]`)

	err := ss.Scan(data)
	if err != nil {
		t.Errorf("Scan(bytes) error = %v", err)
	}

	if len(ss) != 3 {
		t.Errorf("Expected 3 items, got %d", len(ss))
	}
	if ss[0] != "tracker1" {
		t.Errorf("ss[0] = %s, want tracker1", ss[0])
	}
}

func TestStringSlice_ScanInvalidBytes(t *testing.T) {
	var ss StringSlice
	err := ss.Scan([]byte(`invalid json`))
	if err == nil {
		t.Error("Scan(invalid json) should return error")
	}
}

func TestStringSlice_ScanUnsupportedType(t *testing.T) {
	var ss StringSlice
	err := ss.Scan(12345)
	if err == nil {
		t.Error("Scan(int) should return error")
	}
}

func TestStringSlice_Value(t *testing.T) {
	ss := StringSlice{"a", "b", "c"}

	value, err := ss.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}

	bytes, ok := value.([]byte)
	if !ok {
		t.Fatal("Value() should return []byte")
	}

	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Errorf("Value() returned invalid JSON: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
}

func TestStringSlice_RoundTrip(t *testing.T) {
	original := StringSlice{"http://tracker1.com", "http://tracker2.com"}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	var restored StringSlice
	err = restored.Scan(value)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(restored) != len(original) {
		t.Fatalf("Length mismatch: got %d, want %d", len(restored), len(original))
	}
	for i := range original {
		if restored[i] != original[i] {
			t.Errorf("Index %d: got %s, want %s", i, restored[i], original[i])
		}
	}
}

func TestTorrent_TableName(t *testing.T) {
	torrent := Torrent{}
	if torrent.TableName() != "torrents" {
		t.Errorf("TableName() = %s, want torrents", torrent.TableName())
	}
}

func TestTorrentStatus_Constants(t *testing.T) {
	// Verify status constants are defined correctly
	if StatusPending != 0 {
		t.Errorf("StatusPending = %d, want 0", StatusPending)
	}
	if StatusDownloading != 1 {
		t.Errorf("StatusDownloading = %d, want 1", StatusDownloading)
	}
	if StatusCompleted != 2 {
		t.Errorf("StatusCompleted = %d, want 2", StatusCompleted)
	}
	if StatusFailed != 3 {
		t.Errorf("StatusFailed = %d, want 3", StatusFailed)
	}
	if StatusPaused != 4 {
		t.Errorf("StatusPaused = %d, want 4", StatusPaused)
	}
}

func TestTranscodeStatus_Constants(t *testing.T) {
	if TranscodeStatusNone != 0 {
		t.Errorf("TranscodeStatusNone = %d, want 0", TranscodeStatusNone)
	}
	if TranscodeStatusPending != 1 {
		t.Errorf("TranscodeStatusPending = %d, want 1", TranscodeStatusPending)
	}
	if TranscodeStatusProcessing != 2 {
		t.Errorf("TranscodeStatusProcessing = %d, want 2", TranscodeStatusProcessing)
	}
	if TranscodeStatusCompleted != 3 {
		t.Errorf("TranscodeStatusCompleted = %d, want 3", TranscodeStatusCompleted)
	}
	if TranscodeStatusFailed != 4 {
		t.Errorf("TranscodeStatusFailed = %d, want 4", TranscodeStatusFailed)
	}
}

func TestTorrentFile_Struct(t *testing.T) {
	file := TorrentFile{
		Path:            "/videos/movie.mp4",
		Size:            1073741824,
		IsSelected:      true,
		IsShareable:     true,
		IsStreamable:    true,
		TranscodeStatus: TranscodeStatusCompleted,
		TranscodedPath:  "/transcoded/movie.mp4",
		TranscodeError:  "",
	}

	if file.Path != "/videos/movie.mp4" {
		t.Errorf("Path = %s, want /videos/movie.mp4", file.Path)
	}
	if file.Size != 1073741824 {
		t.Errorf("Size = %d, want 1073741824", file.Size)
	}
}

func BenchmarkTorrentFiles_Scan(b *testing.B) {
	data := []byte(`[{"path":"file1.mp4","size":1000},{"path":"file2.mp4","size":2000}]`)

	for i := 0; i < b.N; i++ {
		var tf TorrentFiles
		tf.Scan(data)
	}
}

func BenchmarkTorrentFiles_Value(b *testing.B) {
	tf := TorrentFiles{
		{Path: "file1.mp4", Size: 1000},
		{Path: "file2.mp4", Size: 2000},
	}

	for i := 0; i < b.N; i++ {
		tf.Value()
	}
}
