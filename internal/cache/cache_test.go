// Package cache provides cache utilities tests
// Author: Done-0
// Created: 2026-01-31
package cache

import (
	"testing"
	"time"
)

func TestTorrentListKey(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		want   string
	}{
		{"user 1", 1, "torrent:list:user:1"},
		{"user 12345", 12345, "torrent:list:user:12345"},
		{"user 0", 0, "torrent:list:user:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TorrentListKey(tt.userID)
			if got != tt.want {
				t.Errorf("TorrentListKey(%d) = %s, want %s", tt.userID, got, tt.want)
			}
		})
	}
}

func TestPublicTorrentListKey(t *testing.T) {
	got := PublicTorrentListKey()
	want := "torrent:list:public"
	if got != want {
		t.Errorf("PublicTorrentListKey() = %s, want %s", got, want)
	}
}

func TestTorrentDetailKey(t *testing.T) {
	tests := []struct {
		name     string
		infoHash string
		want     string
	}{
		{"simple hash", "abc123", "torrent:detail:abc123"},
		{"full hash", "a1b2c3d4e5f6g7h8i9j0", "torrent:detail:a1b2c3d4e5f6g7h8i9j0"},
		{"empty hash", "", "torrent:detail:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TorrentDetailKey(tt.infoHash)
			if got != tt.want {
				t.Errorf("TorrentDetailKey(%s) = %s, want %s", tt.infoHash, got, tt.want)
			}
		})
	}
}

func TestTorrentProgressKey(t *testing.T) {
	tests := []struct {
		name     string
		infoHash string
		want     string
	}{
		{"simple hash", "abc123", "torrent:progress:abc123"},
		{"full hash", "a1b2c3d4e5f6g7h8i9j0", "torrent:progress:a1b2c3d4e5f6g7h8i9j0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TorrentProgressKey(tt.infoHash)
			if got != tt.want {
				t.Errorf("TorrentProgressKey(%s) = %s, want %s", tt.infoHash, got, tt.want)
			}
		})
	}
}

func TestInvalidateUserTorrentsPattern(t *testing.T) {
	got := InvalidateUserTorrentsPattern()
	want := "torrent:list:user:*"
	if got != want {
		t.Errorf("InvalidateUserTorrentsPattern() = %s, want %s", got, want)
	}
}

func TestInvalidateTorrentCachesPattern(t *testing.T) {
	got := InvalidateTorrentCachesPattern()
	want := "torrent:*"
	if got != want {
		t.Errorf("InvalidateTorrentCachesPattern() = %s, want %s", got, want)
	}
}

func TestPrefixConstants(t *testing.T) {
	if PrefixTorrent != "torrent:" {
		t.Errorf("PrefixTorrent = %s, want torrent:", PrefixTorrent)
	}
	if PrefixUser != "user:" {
		t.Errorf("PrefixUser = %s, want user:", PrefixUser)
	}
}

func TestTTLConstants(t *testing.T) {
	if TTLTorrentList != 30*time.Second {
		t.Errorf("TTLTorrentList = %v, want 30s", TTLTorrentList)
	}
	if TTLTorrentDetail != 2*time.Minute {
		t.Errorf("TTLTorrentDetail = %v, want 2m", TTLTorrentDetail)
	}
	if TTLPublicList != 1*time.Minute {
		t.Errorf("TTLPublicList = %v, want 1m", TTLPublicList)
	}
}

func TestErrCacheMiss(t *testing.T) {
	if ErrCacheMiss == nil {
		t.Error("ErrCacheMiss should not be nil")
	}
	if ErrCacheMiss.Error() != "cache miss" {
		t.Errorf("ErrCacheMiss.Error() = %s, want 'cache miss'", ErrCacheMiss.Error())
	}
}

func TestKeyUniqueness(t *testing.T) {
	// Verify different functions produce different keys
	key1 := TorrentListKey(123)
	key2 := TorrentDetailKey("123")
	key3 := TorrentProgressKey("123")

	if key1 == key2 {
		t.Error("TorrentListKey and TorrentDetailKey should produce different keys")
	}
	if key2 == key3 {
		t.Error("TorrentDetailKey and TorrentProgressKey should produce different keys")
	}
	if key1 == key3 {
		t.Error("TorrentListKey and TorrentProgressKey should produce different keys")
	}
}

func BenchmarkTorrentListKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TorrentListKey(int64(i))
	}
}

func BenchmarkTorrentDetailKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TorrentDetailKey("abc123def456")
	}
}
