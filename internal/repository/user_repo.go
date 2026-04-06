// Package repository provides GORM-based implementations of domain repository interfaces.
// Author: Done-0
// Created: 2026-03-16
package repository

import (
	"context"

	"gorm.io/gorm"

	domain "magnet2video/domain/user"
	"magnet2video/internal/db"
	userModel "magnet2video/internal/model/user"
	"magnet2video/internal/repository/mapper"
)

// GormUserRepository implements domain.UserRepository using GORM.
type GormUserRepository struct {
	dbManager db.DatabaseManager
}

// NewUserRepository creates a new GormUserRepository.
func NewUserRepository(dbManager db.DatabaseManager) *GormUserRepository {
	return &GormUserRepository{dbManager: dbManager}
}

func (r *GormUserRepository) db() *gorm.DB {
	return r.dbManager.DB()
}

// FindByID finds a user by ID.
func (r *GormUserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var m userModel.User
	if err := r.db().WithContext(ctx).Where("id = ? AND deleted = ?", id, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapper.UserToDomain(&m), nil
}

// FindByEmail finds a user by email.
func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m userModel.User
	if err := r.db().WithContext(ctx).Where("email = ? AND deleted = ?", email, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapper.UserToDomain(&m), nil
}

// FindByNickname finds a user by nickname.
func (r *GormUserRepository) FindByNickname(ctx context.Context, nickname string) (*domain.User, error) {
	var m userModel.User
	if err := r.db().WithContext(ctx).Where("nickname = ? AND deleted = ?", nickname, false).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return mapper.UserToDomain(&m), nil
}

// Create inserts a new user.
func (r *GormUserRepository) Create(ctx context.Context, u *domain.User) error {
	m := mapper.UserToModel(u)
	if err := r.db().WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	u.ID = m.ID
	u.CreatedAt = m.CreatedAt
	u.UpdatedAt = m.UpdatedAt
	return nil
}

// Save updates an existing user.
func (r *GormUserRepository) Save(ctx context.Context, u *domain.User) error {
	m := mapper.UserToModel(u)
	return r.db().WithContext(ctx).Save(m).Error
}

// Delete soft-deletes a user by ID.
func (r *GormUserRepository) Delete(ctx context.Context, id int64) error {
	return r.db().WithContext(ctx).Model(&userModel.User{}).Where("id = ?", id).Update("deleted", true).Error
}

// List lists users with optional search and role filter, with pagination.
func (r *GormUserRepository) List(ctx context.Context, search, role string, page, pageSize int) ([]domain.User, int64, error) {
	var total int64
	query := r.db().WithContext(ctx).Model(&userModel.User{}).Where("deleted = ?", false)

	if search != "" {
		query = query.Where("nickname LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []userModel.User
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]domain.User, len(models))
	for i := range models {
		result[i] = *mapper.UserToDomain(&models[i])
	}
	return result, total, nil
}

// CountAll returns the total number of active users.
func (r *GormUserRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db().WithContext(ctx).Model(&userModel.User{}).Where("deleted = ?", false).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
