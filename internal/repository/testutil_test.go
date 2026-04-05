// Package repository provides GORM-based implementations of domain repository interfaces.
// Author: Done-0
// Created: 2026-03-16
package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/internal/model"
)

// mockDBManager wraps a real GORM DB backed by SQLite in-memory for testing
type mockDBManager struct {
	db *gorm.DB
}

func (m *mockDBManager) DB() *gorm.DB      { return m.db }
func (m *mockDBManager) Initialize() error  { return nil }
func (m *mockDBManager) Close() error       { return nil }

// setupTestDB creates a SQLite in-memory database with all models migrated
func setupTestDB(t *testing.T) *mockDBManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(model.GetAllModels()...); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return &mockDBManager{db: db}
}
