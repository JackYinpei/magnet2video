// Package transcode provides transcode job model tests
// Author: Done-0
// Created: 2026-01-31
package transcode

import "testing"

func TestTranscodeJob_TableName(t *testing.T) {
	var job TranscodeJob
	if name := job.TableName(); name != "transcode_jobs" {
		t.Fatalf("TableName() = %q, want %q", name, "transcode_jobs")
	}
}

func TestTranscodeJobStatusConstants(t *testing.T) {
	if JobStatusPending != 0 {
		t.Fatalf("JobStatusPending = %d, want 0", JobStatusPending)
	}
	if JobStatusProcessing != 1 {
		t.Fatalf("JobStatusProcessing = %d, want 1", JobStatusProcessing)
	}
	if JobStatusCompleted != 2 {
		t.Fatalf("JobStatusCompleted = %d, want 2", JobStatusCompleted)
	}
	if JobStatusFailed != 3 {
		t.Fatalf("JobStatusFailed = %d, want 3", JobStatusFailed)
	}
	if JobStatusCancelled != 4 {
		t.Fatalf("JobStatusCancelled = %d, want 4", JobStatusCancelled)
	}
}
