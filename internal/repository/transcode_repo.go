// Package repository provides GORM-based implementations of domain repository interfaces.
// Author: Done-0
// Created: 2026-03-16
package repository

import (
	"context"

	"gorm.io/gorm"

	domain "magnet2video/domain/transcode"
	transcodeModel "magnet2video/internal/model/transcode"
	"magnet2video/internal/db"
	"magnet2video/internal/repository/mapper"
)

// GormTranscodeJobRepository implements domain.JobRepository using GORM.
type GormTranscodeJobRepository struct {
	dbManager db.DatabaseManager
}

// NewTranscodeJobRepository creates a new GormTranscodeJobRepository.
func NewTranscodeJobRepository(dbManager db.DatabaseManager) *GormTranscodeJobRepository {
	return &GormTranscodeJobRepository{dbManager: dbManager}
}

func (r *GormTranscodeJobRepository) db() *gorm.DB {
	return r.dbManager.DB()
}

// FindByID finds a transcode job by ID.
func (r *GormTranscodeJobRepository) FindByID(ctx context.Context, id int64) (*domain.Job, error) {
	var m transcodeModel.TranscodeJob
	if err := r.db().WithContext(ctx).Where("id = ? AND deleted = ?", id, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrJobNotFound
		}
		return nil, err
	}
	return mapper.JobToDomain(&m), nil
}

// Create inserts a new transcode job.
func (r *GormTranscodeJobRepository) Create(ctx context.Context, job *domain.Job) error {
	m := mapper.JobToModel(job)
	if err := r.db().WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	job.ID = m.ID
	job.CreatedAt = m.CreatedAt
	job.UpdatedAt = m.UpdatedAt
	return nil
}

// Save updates an existing transcode job.
func (r *GormTranscodeJobRepository) Save(ctx context.Context, job *domain.Job) error {
	m := mapper.JobToModel(job)
	return r.db().WithContext(ctx).Save(m).Error
}

// FindByTorrentID finds all transcode jobs for a torrent.
func (r *GormTranscodeJobRepository) FindByTorrentID(ctx context.Context, torrentID int64) ([]domain.Job, error) {
	var models []transcodeModel.TranscodeJob
	if err := r.db().WithContext(ctx).Where("torrent_id = ? AND deleted = ?", torrentID, false).
		Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]domain.Job, len(models))
	for i := range models {
		result[i] = *mapper.JobToDomain(&models[i])
	}
	return result, nil
}

// DeleteByTorrentID soft-deletes all transcode jobs for a torrent.
func (r *GormTranscodeJobRepository) DeleteByTorrentID(ctx context.Context, torrentID int64) error {
	return r.db().WithContext(ctx).Model(&transcodeModel.TranscodeJob{}).
		Where("torrent_id = ?", torrentID).Update("deleted", true).Error
}

// ListAll lists all transcode jobs with optional status filter and pagination.
func (r *GormTranscodeJobRepository) ListAll(ctx context.Context, page, pageSize int, status int) ([]domain.Job, int64, error) {
	var total int64
	query := r.db().WithContext(ctx).Model(&transcodeModel.TranscodeJob{}).Where("deleted = ?", false)

	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []transcodeModel.TranscodeJob
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.Job, len(models))
	for i := range models {
		result[i] = *mapper.JobToDomain(&models[i])
	}
	return result, total, nil
}
