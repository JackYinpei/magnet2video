// Package transcode provides the transcode bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package transcode

import "time"

// JobStatus represents the processing state of a transcode job.
type JobStatus int

const (
	JobPending    JobStatus = 0
	JobProcessing JobStatus = 1
	JobCompleted  JobStatus = 2
	JobFailed     JobStatus = 3
	JobCancelled  JobStatus = 4
)

// Job is the aggregate root for a transcoding task.
type Job struct {
	ID            int64
	TorrentID     int64
	InfoHash      string
	FileIndex     int
	InputPath     string
	OutputPath    string
	InputCodec    string
	OutputCodec   string
	TranscodeType string // "remux" or "transcode"
	Duration      int64
	Status        JobStatus
	Progress      int
	ErrorMessage  string
	StartedAt     int64
	CompletedAt   int64
	CreatorID     int64
	CreatedAt     int64
	UpdatedAt     int64
}

// NewJob creates a new transcode job in pending state.
func NewJob(torrentID int64, infoHash string, fileIndex int, inputPath, outputPath, transcodeType string, creatorID int64) *Job {
	return &Job{
		TorrentID:     torrentID,
		InfoHash:      infoHash,
		FileIndex:     fileIndex,
		InputPath:     inputPath,
		OutputPath:    outputPath,
		TranscodeType: transcodeType,
		Status:        JobPending,
		CreatorID:     creatorID,
	}
}

// Start transitions from pending to processing.
func (j *Job) Start() error {
	if j.Status != JobPending {
		return ErrJobAlreadyRunning
	}
	j.Status = JobProcessing
	j.StartedAt = time.Now().UnixMilli()
	return nil
}

// UpdateProgress sets the progress percentage.
func (j *Job) UpdateProgress(progress int) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	j.Progress = progress
}

// Complete transitions from processing to completed.
func (j *Job) Complete(outputPath string) error {
	if j.Status != JobProcessing {
		return ErrJobAlreadyRunning
	}
	j.Status = JobCompleted
	j.OutputPath = outputPath
	j.Progress = 100
	j.CompletedAt = time.Now().UnixMilli()
	return nil
}

// Fail marks the job as failed with an error message.
func (j *Job) Fail(errMsg string) error {
	j.Status = JobFailed
	j.ErrorMessage = errMsg
	j.CompletedAt = time.Now().UnixMilli()
	return nil
}

// CanRetry returns true if the job can be retried.
func (j *Job) CanRetry() bool {
	return j.Status == JobFailed
}

// ResetForRetry resets the job to pending state for retry.
func (j *Job) ResetForRetry() error {
	if !j.CanRetry() {
		return ErrJobCannotRetry
	}
	j.Status = JobPending
	j.Progress = 0
	j.ErrorMessage = ""
	j.StartedAt = 0
	j.CompletedAt = 0
	return nil
}
