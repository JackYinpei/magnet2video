// Package impl provides torrent service implementation tests
// Author: Done-0
// Created: 2026-02-06
package impl

import (
	"testing"
	"time"

	torrentModel "magnet2video/internal/model/torrent"
)

// setupTorrentServiceDirect creates a TorrentServiceImpl without calling the constructor
// to avoid goroutine side effects (restoreTorrents) and the need for a real TorrentManager.Client()
func setupTorrentServiceDirect(t *testing.T) (*TorrentServiceImpl, *MockDatabaseManager, *MockCacheManager) {
	t.Helper()
	dbMgr := setupTestDB(t)
	logMgr := newMockLoggerManager()
	cacheMgr := newMockCacheManager()
	svc := &TorrentServiceImpl{
		loggerManager: logMgr,
		dbManager:     dbMgr,
		cacheManager:  cacheMgr,
	}
	return svc, dbMgr, cacheMgr
}

// --- ListTorrents Tests ---

func TestTorrentService_ListTorrents(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		seedCount int
		wantTotal int
	}{
		{name: "unauthenticated returns empty", userID: 0, seedCount: 0, wantTotal: 0},
		{name: "empty list", userID: 500, seedCount: 0, wantTotal: 0},
		{name: "returns user torrents only", userID: 500, seedCount: 3, wantTotal: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr, _ := setupTorrentServiceDirect(t)

			// Seed torrents for user 500 with Paused status (avoids Client().GetProgress())
			for i := 0; i < tt.seedCount; i++ {
				tor := &torrentModel.Torrent{
					InfoHash:  "hash" + string(rune('a'+i)),
					Name:      "torrent" + string(rune('a'+i)),
					CreatorID: 500,
					Status:    torrentModel.StatusPaused,
				}
				dbMgr.DB().Create(tor)
			}

			// Seed a torrent for a different user (should not appear)
			other := &torrentModel.Torrent{
				InfoHash:  "otherhash",
				Name:      "other torrent",
				CreatorID: 999,
				Status:    torrentModel.StatusPaused,
			}
			dbMgr.DB().Create(other)

			c := newTestGinContext(tt.userID)
			resp, err := svc.ListTorrents(c)

			if err != nil {
				t.Fatalf("ListTorrents() unexpected error: %v", err)
			}
			if resp.Total != tt.wantTotal {
				t.Errorf("ListTorrents() total = %d, want %d", resp.Total, tt.wantTotal)
			}

			// Verify no other-user torrents leaked
			for _, item := range resp.Torrents {
				if item.InfoHash == "otherhash" {
					t.Error("ListTorrents() returned torrent belonging to another user")
				}
			}
		})
	}
}

// --- ListPublicTorrents Tests ---

func TestTorrentService_ListPublicTorrents(t *testing.T) {
	svc, dbMgr, _ := setupTorrentServiceDirect(t)

	// Seed torrents with different visibility levels
	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "pub1", Name: "public1", Visibility: torrentModel.VisibilityPublic,
		CreatorID: 1, Status: torrentModel.StatusPaused,
	})
	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "pub2", Name: "public2", Visibility: torrentModel.VisibilityPublic,
		CreatorID: 2, Status: torrentModel.StatusPaused,
	})
	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "int1", Name: "internal1", Visibility: torrentModel.VisibilityInternal,
		CreatorID: 1, Status: torrentModel.StatusPaused,
	})
	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "priv1", Name: "private1", Visibility: torrentModel.VisibilityPrivate,
		CreatorID: 1, Status: torrentModel.StatusPaused,
	})

	// Anonymous user should only see public (visibility=2)
	c := newTestGinContext(0)
	resp, err := svc.ListPublicTorrents(c)
	if err != nil {
		t.Fatalf("ListPublicTorrents() unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("ListPublicTorrents(anonymous) total = %d, want 2", resp.Total)
	}

	// Verify only public torrents returned for anonymous
	for _, item := range resp.Torrents {
		if item.InfoHash == "priv1" || item.InfoHash == "int1" {
			t.Errorf("ListPublicTorrents(anonymous) returned non-public torrent %s", item.InfoHash)
		}
	}

	// Logged-in user should see internal + public (visibility IN 1,2)
	c2 := newTestGinContext(100)
	resp2, err := svc.ListPublicTorrents(c2)
	if err != nil {
		t.Fatalf("ListPublicTorrents(logged-in) unexpected error: %v", err)
	}
	if resp2.Total != 3 {
		t.Errorf("ListPublicTorrents(logged-in) total = %d, want 3", resp2.Total)
	}

	// Verify private torrent not returned for logged-in user
	for _, item := range resp2.Torrents {
		if item.InfoHash == "priv1" {
			t.Error("ListPublicTorrents(logged-in) returned a private torrent")
		}
	}
}

// --- ListTorrents Excludes Deleted ---

func TestTorrentService_ListTorrents_ExcludesDeleted(t *testing.T) {
	svc, dbMgr, _ := setupTorrentServiceDirect(t)

	tor := &torrentModel.Torrent{
		InfoHash: "deleted1", Name: "deleted torrent", CreatorID: 600,
		Status: torrentModel.StatusPaused,
	}
	tor.Deleted = true
	dbMgr.DB().Create(tor)

	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "alive1", Name: "alive torrent", CreatorID: 600,
		Status: torrentModel.StatusPaused,
	})

	c := newTestGinContext(int64(600))
	resp, err := svc.ListTorrents(c)
	if err != nil {
		t.Fatalf("ListTorrents() unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("ListTorrents() total = %d, want 1 (should exclude deleted)", resp.Total)
	}
}

// --- ListTorrents Cache Behavior ---

func TestTorrentService_ListTorrents_CacheMiss(t *testing.T) {
	svc, dbMgr, _ := setupTorrentServiceDirect(t)

	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "cached1", Name: "cached torrent", CreatorID: 700,
		Status: torrentModel.StatusPaused,
	})

	c := newTestGinContext(int64(700))

	// First call: cache miss, should load from DB
	resp, err := svc.ListTorrents(c)
	if err != nil {
		t.Fatalf("ListTorrents() first call error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("ListTorrents() first call total = %d, want 1", resp.Total)
	}

	// Give async cache write time to complete
	time.Sleep(100 * time.Millisecond)
}

// --- Helper Function Tests ---

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "bytes", bytes: 500, want: "500 B"},
		{name: "kilobytes", bytes: 1024, want: "1.0 KB"},
		{name: "megabytes", bytes: 1024 * 1024, want: "1.0 MB"},
		{name: "gigabytes", bytes: 1024 * 1024 * 1024, want: "1.0 GB"},
		{name: "zero", bytes: 0, want: "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	got := formatSpeed(1024)
	want := "1.0 KB/s"
	if got != want {
		t.Errorf("formatSpeed(1024) = %q, want %q", got, want)
	}
}

func TestIsPosterImage(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"poster.jpg", true},
		{"poster.jpeg", true},
		{"poster.png", true},
		{"poster.gif", true},
		{"poster.webp", true},
		{"poster.bmp", true},
		{"video.mp4", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isPosterImage(tt.path); got != tt.want {
				t.Errorf("isPosterImage(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestToRelativePath(t *testing.T) {
	tests := []struct {
		name        string
		absPath     string
		downloadDir string
		want        string
	}{
		{name: "absolute to relative", absPath: "/data/download/movie/file.mp4", downloadDir: "/data/download", want: "movie/file.mp4"},
		{name: "exact match", absPath: "/data/download", downloadDir: "/data/download", want: "."},
		{name: "empty path", absPath: "", downloadDir: "/data/download", want: ""},
		{name: "not under download dir", absPath: "/other/path/file.mp4", downloadDir: "/data/download", want: "/other/path/file.mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toRelativePath(tt.absPath, tt.downloadDir)
			if got != tt.want {
				t.Errorf("toRelativePath(%q, %q) = %q, want %q", tt.absPath, tt.downloadDir, got, tt.want)
			}
		})
	}
}

func TestGetStatusFromString(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"downloading", torrentModel.StatusDownloading},
		{"completed", torrentModel.StatusCompleted},
		{"seeding", torrentModel.StatusCompleted},
		{"paused", torrentModel.StatusPaused},
		{"failed", torrentModel.StatusFailed},
		{"unknown", torrentModel.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := getStatusFromString(tt.input); got != tt.want {
				t.Errorf("getStatusFromString(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// --- Benchmarks ---

func BenchmarkTorrentService_ListTorrents(b *testing.B) {
	t := &testing.T{}
	svc, dbMgr, _ := setupTorrentServiceDirect(t)

	for i := 0; i < 10; i++ {
		dbMgr.DB().Create(&torrentModel.Torrent{
			InfoHash:  "benchhash" + string(rune('a'+i)),
			Name:      "bench torrent",
			CreatorID: 800,
			Status:    torrentModel.StatusPaused,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestGinContext(int64(800))
		svc.ListTorrents(c)
	}
}

func BenchmarkFormatSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatSize(1024 * 1024 * 1024)
	}
}
