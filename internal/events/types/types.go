// Package types provides event types for worker/server communication
// Author: magnet2video
// Created: 2026-04-20
package types

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Topic constants for event queues
const (
	TopicWorkerEvents    = "worker-events"
	TopicWorkerHeartbeat = "worker-heartbeat"
	TopicDownloadJobs    = "download-jobs"
)

// EventType constants
const (
	// Transcode lifecycle
	EventTypeTranscodeJobStarted   = "transcode.job.started"
	EventTypeTranscodeJobProgress  = "transcode.job.progress"
	EventTypeTranscodeJobCompleted = "transcode.job.completed"
	EventTypeTranscodeJobFailed    = "transcode.job.failed"
	EventTypeSubtitleExtracted     = "subtitle.extracted"

	// Cloud upload lifecycle
	EventTypeCloudUploadStarted   = "cloud.upload.started"
	EventTypeCloudUploadCompleted = "cloud.upload.completed"
	EventTypeCloudUploadFailed    = "cloud.upload.failed"

	// Download lifecycle
	EventTypeDownloadProgress  = "download.progress"
	EventTypeDownloadCompleted = "download.completed"
	EventTypeDownloadFailed    = "download.failed"

	// Poster candidates
	EventTypePosterCandidateUploaded = "poster.candidate.uploaded"
)

// DownloadAction constants for DownloadJob
const (
	DownloadActionStart  = "start"
	DownloadActionPause  = "pause"
	DownloadActionResume = "resume"
	DownloadActionRemove = "remove"
)

// WorkerEvent is the envelope for all worker-to-server events
type WorkerEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	WorkerID  string          `json:"worker_id"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent builds a WorkerEvent with an auto-generated ID
func NewEvent(eventType, workerID string, payload any) (*WorkerEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &WorkerEvent{
		EventID:   GenerateEventID(),
		EventType: eventType,
		WorkerID:  workerID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   data,
	}, nil
}

// DecodePayload unmarshals Payload into the target struct
func (e *WorkerEvent) DecodePayload(target any) error {
	return json.Unmarshal(e.Payload, target)
}

// GenerateEventID produces a unique event id
func GenerateEventID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int31())
}

// ---- Transcode payloads ----

type TranscodeJobStartedPayload struct {
	JobID     int64 `json:"job_id"`
	TorrentID int64 `json:"torrent_id"`
	FileIndex int   `json:"file_index"`
}

type TranscodeJobProgressPayload struct {
	JobID     int64 `json:"job_id"`
	TorrentID int64 `json:"torrent_id"`
	FileIndex int   `json:"file_index"`
	Progress  int   `json:"progress"`
}

type TranscodeJobCompletedPayload struct {
	JobID      int64  `json:"job_id"`
	TorrentID  int64  `json:"torrent_id"`
	InfoHash   string `json:"info_hash"`
	FileIndex  int    `json:"file_index"`
	OutputPath string `json:"output_path"`
	OutputSize int64  `json:"output_size"`
	Duration   int64  `json:"duration"`
	CreatorID  int64  `json:"creator_id"`
}

type TranscodeJobFailedPayload struct {
	JobID     int64  `json:"job_id"`
	TorrentID int64  `json:"torrent_id"`
	FileIndex int    `json:"file_index"`
	ErrorMsg  string `json:"error_msg"`
}

type SubtitleExtractedPayload struct {
	TorrentID       int64  `json:"torrent_id"`
	InfoHash        string `json:"info_hash"`
	ParentFileIndex int    `json:"parent_file_index"`
	CreatorID       int64  `json:"creator_id"`
	FilePath        string `json:"file_path"`
	FileSize        int64  `json:"file_size"`
	StreamIndex     int    `json:"stream_index"`
	Language        string `json:"language"`
	LanguageName    string `json:"language_name"`
	Title           string `json:"title"`
	Format          string `json:"format"`
	OriginalCodec   string `json:"original_codec"`
}

// ---- Cloud upload payloads ----

type CloudUploadStartedPayload struct {
	TorrentID int64 `json:"torrent_id"`
	FileIndex int   `json:"file_index"`
}

type CloudUploadCompletedPayload struct {
	TorrentID int64  `json:"torrent_id"`
	FileIndex int    `json:"file_index"`
	CloudPath string `json:"cloud_path"`
}

type CloudUploadFailedPayload struct {
	TorrentID int64  `json:"torrent_id"`
	FileIndex int    `json:"file_index"`
	ErrorMsg  string `json:"error_msg"`
}

// ---- Download payloads ----

type DownloadProgressPayload struct {
	InfoHash       string  `json:"info_hash"`
	Name           string  `json:"name"`
	Progress       float64 `json:"progress"`
	Status         string  `json:"status"`
	DownloadedSize int64   `json:"downloaded_size"`
	TotalSize      int64   `json:"total_size"`
	Peers          int     `json:"peers"`
	Seeds          int     `json:"seeds"`
	DownloadSpeed  int64   `json:"download_speed"`
}

type DownloadCompletedPayload struct {
	InfoHash string `json:"info_hash"`
}

type DownloadFailedPayload struct {
	InfoHash string `json:"info_hash"`
	ErrorMsg string `json:"error_msg"`
}

// ---- Poster candidate ----

type PosterCandidateUploadedPayload struct {
	TorrentID int64  `json:"torrent_id"`
	FileIndex int    `json:"file_index"`
	FilePath  string `json:"file_path"`
	CloudPath string `json:"cloud_path"`
}

// ---- Heartbeat ----

// Heartbeat is sent periodically from worker to server
type Heartbeat struct {
	WorkerID    string         `json:"worker_id"`
	Timestamp   int64          `json:"timestamp"`
	CurrentJobs []HeartbeatJob `json:"current_jobs"`
	DiskFreeGB  int64          `json:"disk_free_gb"`
	Version     string         `json:"version"`
}

// HeartbeatJob describes an in-flight job on the worker
type HeartbeatJob struct {
	JobType  string `json:"job_type"` // "download" | "transcode" | "cloud_upload"
	InfoHash string `json:"info_hash"`
	FileName string `json:"file_name"`
	Progress int    `json:"progress"`
}

// ---- Download Job (server -> worker) ----

// DownloadJob is a command sent from server to worker
type DownloadJob struct {
	Action        string   `json:"action"` // start | pause | resume | remove
	InfoHash      string   `json:"info_hash"`
	MagnetURI     string   `json:"magnet_uri"`
	SelectedFiles []int    `json:"selected_files"`
	Trackers      []string `json:"trackers"`
	DeleteFiles   bool     `json:"delete_files"`
}
