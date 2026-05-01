// Package internal provides torrent manager implementation
// Author: Done-0
// Created: 2026-01-22
package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"io"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"golang.org/x/time/rate"

	"magnet2video/configs"
)

// DefaultTrackers are used when user doesn't provide custom trackers
var DefaultTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://tracker.bittor.pw:1337/announce",
	"udp://public.popcorn-tracker.org:6969/announce",
	"udp://tracker.dler.org:6969/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://open.demonii.si:1337/announce",
}

// FileInfo represents information about a file in a torrent
type FileInfo struct {
	Path         string `json:"path"`          // File path
	Size         int64  `json:"size"`          // File size in bytes
	IsStreamable bool   `json:"is_streamable"` // Whether the file can be streamed (h264 video)
}

// TorrentInfo represents parsed torrent information
type TorrentInfo struct {
	InfoHash   string     `json:"info_hash"`   // Info hash
	Name       string     `json:"name"`        // Torrent name
	TotalSize  int64      `json:"total_size"`  // Total size in bytes
	Files      []FileInfo `json:"files"`       // Files in the torrent
	NumPieces  int        `json:"num_pieces"`  // Number of pieces
	PieceSize  int64      `json:"piece_size"`  // Piece size in bytes
	IsPrivate  bool       `json:"is_private"`  // Whether the torrent is private
	Comment    string     `json:"comment"`     // Torrent comment
	CreatedBy  string     `json:"created_by"`  // Created by
	CreateDate int64      `json:"create_date"` // Creation date timestamp
}

// DownloadProgress represents download progress information
type DownloadProgress struct {
	InfoHash       string  `json:"info_hash"`
	Name           string  `json:"name"`
	TotalSize      int64   `json:"total_size"`
	DownloadedSize int64   `json:"downloaded_size"`
	Progress       float64 `json:"progress"`
	Status         string  `json:"status"`
	Peers          int     `json:"peers"`
	Seeds          int     `json:"seeds"`
	DownloadSpeed  int64   `json:"download_speed"` // Bytes per second
}

// speedStat holds information to calculate download speed
type speedStat struct {
	LastBytes int64
	LastTime  time.Time
}

// Client is the torrent client that manages downloads
type Client struct {
	client      *torrent.Client
	downloadDir string
	mu          sync.RWMutex
	// locks stores mutexes for each info hash to prevent concurrent operations
	locks map[string]*sync.Mutex
	// torrents stores active torrent handles
	torrents map[string]*torrent.Torrent
	// speedStats stores speed calculation data
	speedStats map[string]*speedStat
}

// Manager wraps the torrent client
type Manager struct {
	client *Client
}

// NewManager creates a new torrent manager
func NewManager(config *configs.Config) (*Manager, error) {
	// Use config or default values
	downloadDir := config.TorrentConfig.DownloadDir
	if downloadDir == "" {
		downloadDir = "./download"
	}

	// Ensure download directory exists
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Configure the torrent client
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = downloadDir

	// Configure seeding
	clientConfig.NoUpload = !config.TorrentConfig.EnableSeeding
	clientConfig.Seed = config.TorrentConfig.EnableSeeding

	// Configure listen port
	if config.TorrentConfig.ListenPort > 0 {
		clientConfig.ListenPort = config.TorrentConfig.ListenPort
	}

	// Configure upload rate limit (KB/s -> bytes/s)
	if config.TorrentConfig.UploadRateLimit > 0 {
		bytesPerSec := config.TorrentConfig.UploadRateLimit * 1024
		clientConfig.UploadRateLimiter = rate.NewLimiter(
			rate.Limit(bytesPerSec),
			bytesPerSec, // burst size equals to rate for smooth limiting
		)
	}

	// Configure download rate limit (KB/s -> bytes/s)
	if config.TorrentConfig.DownloadRateLimit > 0 {
		bytesPerSec := config.TorrentConfig.DownloadRateLimit * 1024
		clientConfig.DownloadRateLimiter = rate.NewLimiter(
			rate.Limit(bytesPerSec),
			bytesPerSec,
		)
	}

	// Use file-based storage to avoid SQLite-related conflicts with mattn/go-sqlite3
	clientConfig.DefaultStorage = storage.NewFile(downloadDir)

	torrentClient, err := torrent.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	client := &Client{
		client:      torrentClient,
		downloadDir: downloadDir,
		locks:       make(map[string]*sync.Mutex),
		torrents:    make(map[string]*torrent.Torrent),
		speedStats:  make(map[string]*speedStat),
	}

	return &Manager{client: client}, nil
}

// Client returns the underlying torrent client
func (m *Manager) Client() *Client {
	return m.client
}

// Close closes the torrent manager
func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// getInfoHashLock returns (or creates) a mutex for the given info hash
func (c *Client) getInfoHashLock(infoHash string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()

	if lock, exists := c.locks[infoHash]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	c.locks[infoHash] = lock
	return lock
}

// ParseMagnet parses a magnet URI and returns torrent information
// This blocks until metadata is received
func (c *Client) ParseMagnet(ctx context.Context, magnetURI string, customTrackers []string) (*TorrentInfo, error) {
	// Parse the magnet URI
	spec, err := torrent.TorrentSpecFromMagnetUri(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("invalid magnet URI: %w", err)
	}

	infoHash := spec.InfoHash.HexString()

	// Get lock for this info hash
	lock := c.getInfoHashLock(infoHash)
	lock.Lock()
	defer lock.Unlock()

	// Check if already exists
	c.mu.RLock()
	existingTorrent, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if exists && existingTorrent.Info() != nil {
		// Return existing torrent info
		info := existingTorrent.Info()
		torrentInfo := &TorrentInfo{
			InfoHash:  infoHash,
			Name:      existingTorrent.Name(),
			TotalSize: info.TotalLength(),
			NumPieces: info.NumPieces(),
			PieceSize: info.PieceLength,
		}

		for _, file := range existingTorrent.Files() {
			fileInfo := FileInfo{
				Path:         file.Path(),
				Size:         file.Length(),
				IsStreamable: isStreamableFile(file.Path()),
			}
			torrentInfo.Files = append(torrentInfo.Files, fileInfo)
		}

		return torrentInfo, nil
	}

	// Add custom trackers or default trackers
	trackers := customTrackers
	if len(trackers) == 0 {
		trackers = DefaultTrackers
	}
	for _, tracker := range trackers {
		spec.Trackers = append(spec.Trackers, []string{tracker})
	}

	// Add the torrent
	t, _, err := c.client.AddTorrentSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}

	// Wait for metadata
	select {
	case <-ctx.Done():
		t.Drop()
		return nil, ctx.Err()
	case <-t.GotInfo():
		// Metadata received
	}

	info := t.Info()
	torrentInfo := &TorrentInfo{
		InfoHash:  infoHash,
		Name:      t.Name(),
		TotalSize: info.TotalLength(),
		NumPieces: info.NumPieces(),
		PieceSize: info.PieceLength,
	}

	// Get files information
	for _, file := range t.Files() {
		fileInfo := FileInfo{
			Path:         file.Path(),
			Size:         file.Length(),
			IsStreamable: isStreamableFile(file.Path()),
		}
		torrentInfo.Files = append(torrentInfo.Files, fileInfo)
	}

	// Store the torrent handle for later use
	c.mu.Lock()
	c.torrents[infoHash] = t
	c.mu.Unlock()

	return torrentInfo, nil
}

// StartDownload starts downloading selected files from a torrent
func (c *Client) StartDownload(ctx context.Context, infoHash string, selectedFiles []int, customTrackers []string) error {
	// Get lock for this info hash
	lock := c.getInfoHashLock(infoHash)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("torrent not found: %s", infoHash)
	}

	if len(selectedFiles) == 0 {
		return fmt.Errorf("no files selected for download")
	}

	// No need to create specific directory, the torrent client handles it based on file paths
	// which are relative to the client's data directory.

	// Set file priorities - only download selected files
	files := t.Files()
	selectedMap := make(map[int]bool)
	for _, idx := range selectedFiles {
		selectedMap[idx] = true
	}

	for i, file := range files {
		if selectedMap[i] {
			file.Download()
		} else {
			file.SetPriority(torrent.PiecePriorityNone)
		}
	}

	return nil
}

// GetTorrentInfo returns torrent information for an existing torrent
func (c *Client) GetTorrentInfo(infoHash string) (*TorrentInfo, error) {
	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("torrent not found: %s", infoHash)
	}

	info := t.Info()
	if info == nil {
		return nil, fmt.Errorf("torrent metadata not available: %s", infoHash)
	}

	torrentInfo := &TorrentInfo{
		InfoHash:  infoHash,
		Name:      t.Name(),
		TotalSize: info.TotalLength(),
		NumPieces: info.NumPieces(),
		PieceSize: info.PieceLength,
	}

	// Get files information
	for _, file := range t.Files() {
		fileInfo := FileInfo{
			Path:         file.Path(),
			Size:         file.Length(),
			IsStreamable: isStreamableFile(file.Path()),
		}
		torrentInfo.Files = append(torrentInfo.Files, fileInfo)
	}

	return torrentInfo, nil
}

// GetProgress returns the download progress for a torrent
func (c *Client) GetProgress(infoHash string) (*DownloadProgress, error) {
	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("torrent not found: %s", infoHash)
	}

	info := t.Info()
	if info == nil {
		return &DownloadProgress{
			InfoHash: infoHash,
			Status:   "fetching_metadata",
		}, nil
	}

	// Calculate progress based on selected files only
	var selectedTotalBytes int64 = 0
	var selectedCompletedBytes int64 = 0

	for _, file := range t.Files() {
		// Check if file has download priority (not PiecePriorityNone)
		// A file with priority None means it's not selected for download
		if file.Priority() != torrent.PiecePriorityNone {
			selectedTotalBytes += file.Length()
			selectedCompletedBytes += file.BytesCompleted()
		}
	}

	// Fallback to total if no files selected (shouldn't happen normally)
	if selectedTotalBytes == 0 {
		selectedTotalBytes = info.TotalLength()
		selectedCompletedBytes = t.BytesCompleted()
	}

	progress := float64(0)
	if selectedTotalBytes > 0 {
		progress = float64(selectedCompletedBytes) / float64(selectedTotalBytes) * 100
	}

	// "seeding" implies "downloaded and now uploading"; anacrolix's
	// t.Seeding() returns true whenever the client is *willing* to upload
	// (i.e. clientConfig.Seed=true), independent of progress, so we must
	// gate it on actually being done.
	status := "downloading"
	if selectedTotalBytes > 0 && selectedCompletedBytes >= selectedTotalBytes {
		status = "completed"
		if t.Seeding() {
			status = "seeding"
		}
	}

	// Calculate speed
	now := time.Now()
	var speed int64 = 0

	stat, ok := c.speedStats[infoHash]
	if !ok {
		c.speedStats[infoHash] = &speedStat{
			LastBytes: selectedCompletedBytes,
			LastTime:  now,
		}
	} else {
		duration := now.Sub(stat.LastTime)
		if duration >= time.Second {
			diff := selectedCompletedBytes - stat.LastBytes
			// Handle potential restart or check weirdness
			if diff < 0 {
				diff = 0
			}
			speed = int64(float64(diff) / duration.Seconds())

			// Update stat
			stat.LastBytes = selectedCompletedBytes
			stat.LastTime = now
		}
	}

	stats := t.Stats()

	return &DownloadProgress{
		InfoHash:       infoHash,
		Name:           t.Name(),
		TotalSize:      selectedTotalBytes,
		DownloadedSize: selectedCompletedBytes,
		Progress:       progress,
		Status:         status,
		Peers:          stats.ActivePeers,
		Seeds:          stats.ConnectedSeeders,
		DownloadSpeed:  speed,
	}, nil
}

// PauseDownload pauses a torrent download
func (c *Client) PauseDownload(infoHash string) error {
	lock := c.getInfoHashLock(infoHash)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("torrent not found: %s", infoHash)
	}

	// Cancel all file downloads
	for _, file := range t.Files() {
		file.SetPriority(torrent.PiecePriorityNone)
	}

	return nil
}

// ResumeDownload resumes a paused torrent download
func (c *Client) ResumeDownload(infoHash string, selectedFiles []int) error {
	lock := c.getInfoHashLock(infoHash)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("torrent not found: %s", infoHash)
	}

	files := t.Files()
	selectedMap := make(map[int]bool)
	for _, idx := range selectedFiles {
		selectedMap[idx] = true
	}

	for i, file := range files {
		if selectedMap[i] {
			file.Download()
		}
	}

	return nil
}

// RestoreTorrent restores a torrent download from database
func (c *Client) RestoreTorrent(infoHash string, trackers []string, selectedFiles []int) error {
	// Check if already exists
	c.mu.RLock()
	if _, exists := c.torrents[infoHash]; exists {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	magnetURI := fmt.Sprintf("magnet:?xt=urn:btih:%s", infoHash)
	// Add default trackers if none provided
	if len(trackers) == 0 {
		trackers = DefaultTrackers
	}
	for _, tr := range trackers {
		magnetURI += fmt.Sprintf("&tr=%s", tr)
	}

	spec, err := torrent.TorrentSpecFromMagnetUri(magnetURI)
	if err != nil {
		return fmt.Errorf("invalid magnet URI: %w", err)
	}

	t, _, err := c.client.AddTorrentSpec(spec)
	if err != nil {
		return fmt.Errorf("failed to add torrent: %w", err)
	}

	c.mu.Lock()
	c.torrents[infoHash] = t
	c.mu.Unlock()

	// Async waiting for metadata and starting download
	go func() {
		<-t.GotInfo()

		// Once we have info, we can select files
		files := t.Files()
		selectedMap := make(map[int]bool)
		for _, idx := range selectedFiles {
			selectedMap[idx] = true
		}

		for i, file := range files {
			if selectedMap[i] {
				file.Download()
			} else {
				file.SetPriority(torrent.PiecePriorityNone)
			}
		}
	}()

	return nil
}

// RemoveTorrent removes a torrent from the client
func (c *Client) RemoveTorrent(infoHash string, deleteFiles bool) error {
	lock := c.getInfoHashLock(infoHash)
	lock.Lock()
	defer lock.Unlock()

	c.mu.Lock()
	t, exists := c.torrents[infoHash]
	if exists {
		delete(c.torrents, infoHash)
	}
	delete(c.locks, infoHash)
	c.mu.Unlock()

	if !exists {
		return fmt.Errorf("torrent not found: %s", infoHash)
	}

	t.Drop()

	if deleteFiles {
		downloadPath := filepath.Join(c.downloadDir, infoHash)
		if err := os.RemoveAll(downloadPath); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
	}

	return nil
}

// GetFilePath returns the full path to a downloaded file
func (c *Client) GetFilePath(infoHash string, filePath string) (string, error) {
	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("torrent not found: %s", infoHash)
	}

	// Find the file in the torrent
	var foundFile *torrent.File
	for _, file := range t.Files() {
		if file.Path() == filePath {
			foundFile = file
			break
		}
	}

	if foundFile == nil {
		return "", fmt.Errorf("file not found in torrent: %s", filePath)
	}

	// file.Path() is the relative path from the download dir, including the
	// torrent's parent directory for multi-file torrents — exactly anacrolix's
	// on-disk layout.
	fullPath := filepath.Join(c.downloadDir, filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not downloaded yet: %s", filePath)
	}

	return fullPath, nil
}

// GetFileReader returns a reader for the file and its info, supporting fuzzy path matching
// Falls back to direct disk read if torrent is not in memory
func (c *Client) GetFileReader(infoHash string, filePath string) (io.ReadSeeker, *FileInfo, error) {
	c.mu.RLock()
	t, exists := c.torrents[infoHash]
	c.mu.RUnlock()

	// If torrent exists in memory, try to read from it
	if exists {
		// Find the file in the torrent
		var foundFile *torrent.File

		// First pass: exact match
		for _, file := range t.Files() {
			if file.Path() == filePath {
				foundFile = file
				break
			}
		}

		// Second pass: fuzzy match (match by filename only)
		if foundFile == nil {
			targetBase := filepath.Base(filePath)
			for _, file := range t.Files() {
				if filepath.Base(file.Path()) == targetBase {
					foundFile = file
					break
				}
			}
		}

		if foundFile != nil {
			// Prioritize this file for download
			foundFile.SetPriority(torrent.PiecePriorityNow)

			// Create reader
			reader := foundFile.NewReader()

			// Create file info
			fileInfo := &FileInfo{
				Path:         foundFile.Path(),
				Size:         foundFile.Length(),
				IsStreamable: isStreamableFile(foundFile.Path()),
			}

			return reader, fileInfo, nil
		}
	}

	// Fallback: try to read directly from disk
	// This handles cases where torrent is not restored but file exists on disk
	fullPath := filepath.Join(c.downloadDir, filePath)

	// Also try fuzzy match on disk
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Try to find file by name in the download directory
		targetBase := filepath.Base(filePath)
		found := false
		filepath.Walk(c.downloadDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Base(path) == targetBase {
				fullPath = path
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		if !found {
			return nil, nil, fmt.Errorf("file not found: %s", filePath)
		}
	}

	// Open file from disk
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileInfo := &FileInfo{
		Path:         filePath,
		Size:         stat.Size(),
		IsStreamable: isStreamableFile(filePath),
	}

	return file, fileInfo, nil
}

// Close closes the torrent client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, t := range c.torrents {
		t.Drop()
	}

	c.client.Close()
	return nil
}

// GetDownloadDir returns the download directory
func (c *Client) GetDownloadDir() string {
	return c.downloadDir
}

// HasTorrent checks if a torrent exists in the client
func (c *Client) HasTorrent(infoHash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.torrents[infoHash]
	return exists
}

// ListInfoHashes returns all infoHashes currently known to the client.
func (c *Client) ListInfoHashes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.torrents))
	for h := range c.torrents {
		out = append(out, h)
	}
	return out
}

// isStreamableFile checks if a file is likely to be streamable in browser
// Based on file extension and common video codecs
func isStreamableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Common streamable video extensions (typically H.264/H.265 encoded)
	streamableExts := map[string]bool{
		".mp4":  true,
		".m4v":  true,
		".webm": true,
		".mov":  true,
	}

	// Non-streamable extensions (typically require transcoding)
	nonStreamableExts := map[string]bool{
		".mkv":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".rmvb": true,
		".rm":   true,
		".mpeg": true,
		".mpg":  true,
		".3gp":  true,
	}

	if streamableExts[ext] {
		return true
	}

	if nonStreamableExts[ext] {
		return false
	}

	// For other extensions (like audio files), assume not streamable video
	return false
}

// Compile-time check to ensure metainfo is used (avoid unused import error)
var _ = metainfo.Hash{}
