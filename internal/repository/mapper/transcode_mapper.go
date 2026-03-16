// Package mapper provides bidirectional mapping between domain models and GORM models.
// Author: Done-0
// Created: 2026-03-16
package mapper

import (
	domain "github.com/Done-0/gin-scaffold/domain/transcode"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
)

// JobToDomain converts a GORM TranscodeJob to a domain Job.
func JobToDomain(m *transcodeModel.TranscodeJob) *domain.Job {
	if m == nil {
		return nil
	}
	return &domain.Job{
		ID:            m.ID,
		TorrentID:     m.TorrentID,
		InfoHash:      m.InfoHash,
		FileIndex:     m.FileIndex,
		InputPath:     m.InputPath,
		OutputPath:    m.OutputPath,
		InputCodec:    m.InputCodec,
		OutputCodec:   m.OutputCodec,
		TranscodeType: m.TranscodeType,
		Duration:      m.Duration,
		Status:        domain.JobStatus(m.Status),
		Progress:      m.Progress,
		ErrorMessage:  m.ErrorMessage,
		StartedAt:     m.StartedAt,
		CompletedAt:   m.CompletedAt,
		CreatorID:     m.CreatorID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// JobToModel converts a domain Job to a GORM TranscodeJob.
func JobToModel(d *domain.Job) *transcodeModel.TranscodeJob {
	if d == nil {
		return nil
	}
	m := &transcodeModel.TranscodeJob{
		TorrentID:     d.TorrentID,
		InfoHash:      d.InfoHash,
		FileIndex:     d.FileIndex,
		InputPath:     d.InputPath,
		OutputPath:    d.OutputPath,
		InputCodec:    d.InputCodec,
		OutputCodec:   d.OutputCodec,
		TranscodeType: d.TranscodeType,
		Duration:      d.Duration,
		Status:        int(d.Status),
		Progress:      d.Progress,
		ErrorMessage:  d.ErrorMessage,
		StartedAt:     d.StartedAt,
		CompletedAt:   d.CompletedAt,
		CreatorID:     d.CreatorID,
	}
	m.ID = d.ID
	m.CreatedAt = d.CreatedAt
	m.UpdatedAt = d.UpdatedAt
	return m
}
