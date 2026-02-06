// Package impl provides shared test utilities for service layer tests
// Author: Done-0
// Created: 2026-02-06
package impl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/internal/cache"
	"github.com/Done-0/gin-scaffold/internal/model"
)

// --- MockLoggerManager ---

// MockLoggerManager provides a logger for testing
type MockLoggerManager struct {
	logger *logrus.Logger
}

func newMockLoggerManager() *MockLoggerManager {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return &MockLoggerManager{logger: l}
}

func (m *MockLoggerManager) Logger() *logrus.Logger { return m.logger }
func (m *MockLoggerManager) Initialize() error      { return nil }
func (m *MockLoggerManager) Close() error            { return nil }

// --- MockDatabaseManager ---

// MockDatabaseManager wraps a real GORM DB backed by SQLite in-memory
type MockDatabaseManager struct {
	db *gorm.DB
}

func (m *MockDatabaseManager) DB() *gorm.DB      { return m.db }
func (m *MockDatabaseManager) Initialize() error  { return nil }
func (m *MockDatabaseManager) Close() error       { return nil }

// setupTestDB creates a SQLite in-memory database with all models migrated
func setupTestDB(t *testing.T) *MockDatabaseManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(model.GetAllModels()...); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return &MockDatabaseManager{db: db}
}

// --- MockCacheManager ---

// MockCacheManager provides an in-memory cache that returns cache.ErrCacheMiss on miss
type MockCacheManager struct {
	mu    sync.RWMutex
	store map[string]any
}

func newMockCacheManager() *MockCacheManager {
	return &MockCacheManager{store: make(map[string]any)}
}

func (m *MockCacheManager) Get(_ context.Context, key string, _ any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.store[key]; !ok {
		return cache.ErrCacheMiss
	}
	return nil
}

func (m *MockCacheManager) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
	return nil
}

func (m *MockCacheManager) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		delete(m.store, key)
	}
	return nil
}

func (m *MockCacheManager) DeleteByPattern(_ context.Context, _ string) error {
	return nil
}

func (m *MockCacheManager) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.store[key]
	return ok, nil
}

func (m *MockCacheManager) GetOrSet(_ context.Context, key string, _ any, _ time.Duration, loader func() (any, error)) error {
	m.mu.RLock()
	_, ok := m.store[key]
	m.mu.RUnlock()
	if ok {
		return nil
	}
	val, err := loader()
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.store[key] = val
	m.mu.Unlock()
	return nil
}

// --- MockQueueProducer ---

// MockQueueProducer records sent messages for assertions
type MockQueueProducer struct {
	mu       sync.Mutex
	messages []mockQueueMessage
}

type mockQueueMessage struct {
	Topic string
	Key   []byte
	Value []byte
}

func newMockQueueProducer() *MockQueueProducer {
	return &MockQueueProducer{}
}

func (m *MockQueueProducer) Send(_ context.Context, topic string, key, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, mockQueueMessage{Topic: topic, Key: key, Value: value})
	return nil
}

func (m *MockQueueProducer) Close() error { return nil }

func (m *MockQueueProducer) len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// --- MockTranscodeChecker ---

// MockTranscodeChecker records TriggerTranscodeCheck calls
type MockTranscodeChecker struct {
	mu         sync.Mutex
	torrentIDs []int64
}

func newMockTranscodeChecker() *MockTranscodeChecker {
	return &MockTranscodeChecker{}
}

func (m *MockTranscodeChecker) TriggerTranscodeCheck(torrentID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.torrentIDs = append(m.torrentIDs, torrentID)
}

// --- Gin Context Helper ---

// newTestGinContext creates a gin.Context with user_id and a valid http.Request
func newTestGinContext(userID int64) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if userID > 0 {
		c.Set("user_id", userID)
	}
	return c
}
