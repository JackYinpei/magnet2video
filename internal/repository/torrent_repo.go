// Package repository provides GORM-based implementations of domain repository interfaces.
// Author: Done-0
// Created: 2026-03-16
package repository

import (
	"context"

	"gorm.io/gorm"

	domain "github.com/Done-0/gin-scaffold/domain/torrent"
	"github.com/Done-0/gin-scaffold/internal/db"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	"github.com/Done-0/gin-scaffold/internal/repository/mapper"
)

// GormTorrentRepository implements domain.TorrentRepository using GORM.
type GormTorrentRepository struct {
	dbManager db.DatabaseManager
}

// NewTorrentRepository creates a new GormTorrentRepository.
func NewTorrentRepository(dbManager db.DatabaseManager) *GormTorrentRepository {
	return &GormTorrentRepository{dbManager: dbManager}
}

func (r *GormTorrentRepository) db() *gorm.DB {
	return r.dbManager.DB()
}

// FindByID finds a torrent by ID without preloading files.
func (r *GormTorrentRepository) FindByID(ctx context.Context, id int64) (*domain.Torrent, error) {
	var m torrentModel.Torrent
	if err := r.db().WithContext(ctx).Where("id = ? AND deleted = ?", id, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTorrentNotFound
		}
		return nil, err
	}
	return mapper.TorrentToDomain(&m), nil
}

// FindByInfoHash finds a torrent by info hash without preloading files.
func (r *GormTorrentRepository) FindByInfoHash(ctx context.Context, infoHash string) (*domain.Torrent, error) {
	var m torrentModel.Torrent
	if err := r.db().WithContext(ctx).Where("info_hash = ? AND deleted = ?", infoHash, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTorrentNotFound
		}
		return nil, err
	}
	return mapper.TorrentToDomain(&m), nil
}

// FindByIDWithFiles finds a torrent by ID with files preloaded.
func (r *GormTorrentRepository) FindByIDWithFiles(ctx context.Context, id int64) (*domain.Torrent, error) {
	var m torrentModel.Torrent
	if err := r.db().WithContext(ctx).Preload("Files").Where("id = ? AND deleted = ?", id, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTorrentNotFound
		}
		return nil, err
	}
	return mapper.TorrentToDomain(&m), nil
}

// FindByInfoHashWithFiles finds a torrent by info hash with files preloaded.
func (r *GormTorrentRepository) FindByInfoHashWithFiles(ctx context.Context, infoHash string) (*domain.Torrent, error) {
	var m torrentModel.Torrent
	if err := r.db().WithContext(ctx).Preload("Files").Where("info_hash = ? AND deleted = ?", infoHash, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTorrentNotFound
		}
		return nil, err
	}
	return mapper.TorrentToDomain(&m), nil
}

// Save updates an existing torrent record.
func (r *GormTorrentRepository) Save(ctx context.Context, t *domain.Torrent) error {
	m := mapper.TorrentToModel(t)
	return r.db().WithContext(ctx).Save(m).Error
}

// Create inserts a new torrent record.
func (r *GormTorrentRepository) Create(ctx context.Context, t *domain.Torrent) error {
	m := mapper.TorrentToModel(t)
	if err := r.db().WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	t.ID = m.ID
	t.CreatedAt = m.CreatedAt
	t.UpdatedAt = m.UpdatedAt
	return nil
}

// Delete soft-deletes a torrent by ID.
func (r *GormTorrentRepository) Delete(ctx context.Context, id int64) error {
	return r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("id = ?", id).Update("deleted", true).Error
}

// ListByCreator lists torrents owned by a specific user with pagination.
func (r *GormTorrentRepository) ListByCreator(ctx context.Context, creatorID int64, page, pageSize int) ([]domain.Torrent, int64, error) {
	var total int64
	query := r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("creator_id = ? AND deleted = ?", creatorID, false)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []torrentModel.Torrent
	offset := (page - 1) * pageSize
	if err := query.Preload("Files").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.Torrent, len(models))
	for i := range models {
		result[i] = *mapper.TorrentToDomain(&models[i])
	}
	return result, total, nil
}

// ListPublic lists public (and optionally internal) torrents with pagination.
func (r *GormTorrentRepository) ListPublic(ctx context.Context, includeInternal bool, page, pageSize int) ([]domain.Torrent, int64, error) {
	var total int64
	query := r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("deleted = ?", false)

	if includeInternal {
		query = query.Where("visibility IN ?", []int{int(domain.VisibilityPublic), int(domain.VisibilityInternal)})
	} else {
		query = query.Where("visibility = ?", int(domain.VisibilityPublic))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []torrentModel.Torrent
	offset := (page - 1) * pageSize
	if err := query.Preload("Files").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.Torrent, len(models))
	for i := range models {
		result[i] = *mapper.TorrentToDomain(&models[i])
	}
	return result, total, nil
}

// ListAll lists all torrents with optional filters and pagination.
func (r *GormTorrentRepository) ListAll(ctx context.Context, search string, status int, creatorID int64, page, pageSize int) ([]domain.Torrent, int64, error) {
	var total int64
	query := r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("deleted = ?", false)

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if creatorID > 0 {
		query = query.Where("creator_id = ?", creatorID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []torrentModel.Torrent
	offset := (page - 1) * pageSize
	if err := query.Preload("Files").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.Torrent, len(models))
	for i := range models {
		result[i] = *mapper.TorrentToDomain(&models[i])
	}
	return result, total, nil
}

// SaveFile updates or creates a torrent file record.
func (r *GormTorrentRepository) SaveFile(ctx context.Context, torrentID int64, f *domain.TorrentFile) error {
	m := mapper.FileToModel(f)
	m.TorrentID = torrentID
	return r.db().WithContext(ctx).Save(m).Error
}

// UpdateFileFields updates specific fields on a torrent file.
func (r *GormTorrentRepository) UpdateFileFields(ctx context.Context, torrentID int64, fileIndex int, updates map[string]interface{}) error {
	return r.db().WithContext(ctx).Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND `index` = ?", torrentID, fileIndex).
		Updates(updates).Error
}

// FindActiveForRestore finds torrents that were downloading or paused (for startup restore).
func (r *GormTorrentRepository) FindActiveForRestore(ctx context.Context) ([]domain.Torrent, error) {
	var models []torrentModel.Torrent
	if err := r.db().WithContext(ctx).Preload("Files").
		Where("deleted = ? AND status IN ?", false, []int{int(domain.DownloadDownloading), int(domain.DownloadPaused)}).
		Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]domain.Torrent, len(models))
	for i := range models {
		result[i] = *mapper.TorrentToDomain(&models[i])
	}
	return result, nil
}

// FindCompletedPendingTranscode finds completed torrents with pending transcode files.
func (r *GormTorrentRepository) FindCompletedPendingTranscode(ctx context.Context) ([]domain.Torrent, error) {
	var models []torrentModel.Torrent
	if err := r.db().WithContext(ctx).Preload("Files").
		Where("deleted = ? AND status = ? AND transcode_status IN ?",
			false, int(domain.DownloadCompleted),
			[]int{int(domain.TranscodePending), int(domain.TranscodeProcessing)}).
		Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]domain.Torrent, len(models))
	for i := range models {
		result[i] = *mapper.TorrentToDomain(&models[i])
	}
	return result, nil
}

// UpdateCloudSummary recalculates and updates cloud upload aggregate fields from file statuses.
func (r *GormTorrentRepository) UpdateCloudSummary(ctx context.Context, torrentID int64) error {
	// Load torrent with files, recalculate, and save aggregate fields
	t, err := r.FindByIDWithFiles(ctx, torrentID)
	if err != nil {
		return err
	}
	t.RecalculateCloudSummary()

	return r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("id = ?", torrentID).Updates(map[string]interface{}{
		"cloud_upload_status":   int(t.CloudUploadStatus),
		"cloud_upload_progress": t.CloudUploadProgress,
		"cloud_uploaded_count":  t.CloudUploadedCount,
		"total_cloud_upload":    t.TotalCloudUpload,
	}).Error
}

// UpdateTranscodeSummary recalculates and updates transcode aggregate fields from file statuses.
func (r *GormTorrentRepository) UpdateTranscodeSummary(ctx context.Context, torrentID int64) error {
	t, err := r.FindByIDWithFiles(ctx, torrentID)
	if err != nil {
		return err
	}
	t.RecalculateTranscodeSummary()

	return r.db().WithContext(ctx).Model(&torrentModel.Torrent{}).Where("id = ?", torrentID).Updates(map[string]interface{}{
		"transcode_status":   int(t.TranscodeStatus),
		"transcode_progress": t.TranscodeProgress,
		"transcoded_count":   t.TranscodedCount,
		"total_transcode":    t.TotalTranscode,
	}).Error
}
