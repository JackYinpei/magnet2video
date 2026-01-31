// Package types provides transcode-related message type tests
// Author: Done-0
// Created: 2026-01-31
package types

import (
	"encoding/json"
	"testing"
)

func TestTranscodeMessage_JSON(t *testing.T) {
	msg := TranscodeMessage{
		JobID:      12345,
		TorrentID:  67890,
		InfoHash:   "abc123def456",
		FileIndex:  0,
		InputPath:  "/input/video.mkv",
		OutputPath: "/output/video.mp4",
		InputCodec: "hevc",
		Operation:  OperationTranscode,
		Priority:   10,
		CreatorID:  1001,
		Preset:     "medium",
		CRF:        23,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored TranscodeMessage
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.JobID != msg.JobID {
		t.Errorf("JobID = %d, want %d", restored.JobID, msg.JobID)
	}
	if restored.TorrentID != msg.TorrentID {
		t.Errorf("TorrentID = %d, want %d", restored.TorrentID, msg.TorrentID)
	}
	if restored.InfoHash != msg.InfoHash {
		t.Errorf("InfoHash = %s, want %s", restored.InfoHash, msg.InfoHash)
	}
	if restored.Operation != msg.Operation {
		t.Errorf("Operation = %s, want %s", restored.Operation, msg.Operation)
	}
}

func TestTranscodeProgressMessage_JSON(t *testing.T) {
	msg := TranscodeProgressMessage{
		JobID:     12345,
		TorrentID: 67890,
		InfoHash:  "abc123def456",
		FileIndex: 0,
		Progress:  75.5,
		Status:    2,
		Error:     "",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored TranscodeProgressMessage
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.Progress != msg.Progress {
		t.Errorf("Progress = %f, want %f", restored.Progress, msg.Progress)
	}
	if restored.Status != msg.Status {
		t.Errorf("Status = %d, want %d", restored.Status, msg.Status)
	}
}

func TestTranscodeResultMessage_JSON(t *testing.T) {
	msg := TranscodeResultMessage{
		JobID:        12345,
		TorrentID:    67890,
		InfoHash:     "abc123def456",
		FileIndex:    0,
		Success:      true,
		OutputPath:   "/output/video.mp4",
		OutputCodec:  "h264",
		OutputSize:   1073741824,
		Duration:     360000,
		ErrorMessage: "",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored TranscodeResultMessage
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.Success != msg.Success {
		t.Errorf("Success = %v, want %v", restored.Success, msg.Success)
	}
	if restored.OutputSize != msg.OutputSize {
		t.Errorf("OutputSize = %d, want %d", restored.OutputSize, msg.OutputSize)
	}
	if restored.Duration != msg.Duration {
		t.Errorf("Duration = %d, want %d", restored.Duration, msg.Duration)
	}
}

func TestTranscodeResultMessage_WithError(t *testing.T) {
	msg := TranscodeResultMessage{
		JobID:        12345,
		Success:      false,
		ErrorMessage: "FFmpeg failed: unsupported codec",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var restored TranscodeResultMessage
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.Success {
		t.Error("Success should be false")
	}
	if restored.ErrorMessage != msg.ErrorMessage {
		t.Errorf("ErrorMessage = %s, want %s", restored.ErrorMessage, msg.ErrorMessage)
	}
}

func TestTopicConstants(t *testing.T) {
	if TopicTranscodeJobs != "transcode-jobs" {
		t.Errorf("TopicTranscodeJobs = %s, want transcode-jobs", TopicTranscodeJobs)
	}
	if TopicTranscodeProgress != "transcode-progress" {
		t.Errorf("TopicTranscodeProgress = %s, want transcode-progress", TopicTranscodeProgress)
	}
	if TopicTranscodeResults != "transcode-results" {
		t.Errorf("TopicTranscodeResults = %s, want transcode-results", TopicTranscodeResults)
	}
}

func TestOperationConstants(t *testing.T) {
	if OperationRemux != "remux" {
		t.Errorf("OperationRemux = %s, want remux", OperationRemux)
	}
	if OperationTranscode != "transcode" {
		t.Errorf("OperationTranscode = %s, want transcode", OperationTranscode)
	}
}

func TestTranscodeMessage_JSONFieldNames(t *testing.T) {
	msg := TranscodeMessage{
		JobID:     123,
		TorrentID: 456,
	}

	data, _ := json.Marshal(msg)
	jsonStr := string(data)

	// Verify JSON field names
	expectedFields := []string{
		`"job_id"`,
		`"torrent_id"`,
		`"info_hash"`,
		`"file_index"`,
		`"input_path"`,
		`"output_path"`,
		`"input_codec"`,
		`"operation"`,
		`"priority"`,
		`"creator_id"`,
		`"preset"`,
		`"crf"`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain field %s", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkTranscodeMessage_Marshal(b *testing.B) {
	msg := TranscodeMessage{
		JobID:      12345,
		TorrentID:  67890,
		InfoHash:   "abc123def456",
		FileIndex:  0,
		InputPath:  "/input/video.mkv",
		OutputPath: "/output/video.mp4",
		InputCodec: "hevc",
		Operation:  OperationTranscode,
	}

	for i := 0; i < b.N; i++ {
		json.Marshal(msg)
	}
}

func BenchmarkTranscodeMessage_Unmarshal(b *testing.B) {
	data := []byte(`{"job_id":12345,"torrent_id":67890,"info_hash":"abc123","file_index":0,"input_path":"/input/video.mkv","output_path":"/output/video.mp4","input_codec":"hevc","operation":"transcode","priority":0,"creator_id":0,"preset":"","crf":0}`)

	for i := 0; i < b.N; i++ {
		var msg TranscodeMessage
		json.Unmarshal(data, &msg)
	}
}
