// Package mapper provides bidirectional mapping between domain models and GORM models.
// Author: Done-0
// Created: 2026-03-16
package mapper

import (
	domain "github.com/Done-0/gin-scaffold/domain/user"
	userModel "github.com/Done-0/gin-scaffold/internal/model/user"
)

// UserToDomain converts a GORM User model to a domain User.
func UserToDomain(m *userModel.User) *domain.User {
	if m == nil {
		return nil
	}
	return &domain.User{
		ID:           m.ID,
		Email:        m.Email,
		Password:     m.Password,
		Nickname:     m.Nickname,
		Avatar:       m.Avatar,
		Role:         m.Role,
		IsSuperAdmin: m.IsSuperAdmin,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// UserToModel converts a domain User to a GORM User model.
func UserToModel(d *domain.User) *userModel.User {
	if d == nil {
		return nil
	}
	m := &userModel.User{
		Email:        d.Email,
		Password:     d.Password,
		Nickname:     d.Nickname,
		Avatar:       d.Avatar,
		Role:         d.Role,
		IsSuperAdmin: d.IsSuperAdmin,
	}
	m.ID = d.ID
	m.CreatedAt = d.CreatedAt
	m.UpdatedAt = d.UpdatedAt
	return m
}
