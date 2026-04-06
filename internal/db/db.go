// Package db provides database management functionality
// Author: Done-0
// Created: 2025-08-24
package db

import (
	"gorm.io/gorm"

	"magnet2video/configs"
	"magnet2video/internal/db/internal"
)

// DatabaseManager defines the interface for database management operations
type DatabaseManager interface {
	DB() *gorm.DB
	Initialize() error
	Close() error
}

// New creates a new database manager instance
func New(config *configs.Config) DatabaseManager {
	return internal.NewManager(config)
}
