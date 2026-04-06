// Package internal provides cloud storage implementation
// Author: Done-0
// Created: 2026-02-01
package internal

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"magnet2video/configs"
	"magnet2video/internal/logger"
)

// CloudStorageManager defines cloud storage operations interface
type CloudStorageManager interface {
	// Upload uploads a file to cloud storage
	Upload(ctx context.Context, objectPath string, reader io.Reader, contentType string) error

	// UploadWithProgress uploads a file with progress callback
	UploadWithProgress(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64, progressCallback func(bytesWritten int64)) error

	// GenerateSignedURL generates a signed URL for temporary access
	GenerateSignedURL(ctx context.Context, objectPath string, expiration time.Duration) (string, error)

	// Delete deletes an object from cloud storage
	Delete(ctx context.Context, objectPath string) error

	// Exists checks if an object exists
	Exists(ctx context.Context, objectPath string) (bool, error)

	// IsEnabled returns whether cloud storage is enabled
	IsEnabled() bool

	// GetBucketName returns the bucket name
	GetBucketName() string

	// GetPathPrefix returns the path prefix
	GetPathPrefix() string

	// GetSignedURLExpiration returns the signed URL expiration duration
	GetSignedURLExpiration() time.Duration

	// Close closes the client connection
	Close() error
}

// gcsManager implements CloudStorageManager for Google Cloud Storage
type gcsManager struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	client        *storage.Client
	bucket        *storage.BucketHandle
}

// New creates a new CloudStorageManager instance
func New(config *configs.Config, loggerManager logger.LoggerManager) CloudStorageManager {
	manager := &gcsManager{
		config:        config,
		loggerManager: loggerManager,
	}

	// Only initialize GCS client if enabled
	if config.CloudStorageConfig.Enabled {
		if err := manager.initialize(); err != nil {
			loggerManager.Logger().Errorf("failed to initialize cloud storage: %v", err)
			// Return manager anyway, IsEnabled() will return false due to nil client
		}
	}

	return manager
}

// initialize initializes the GCS client
func (m *gcsManager) initialize() error {
	ctx := context.Background()

	var opts []option.ClientOption
	if m.config.CloudStorageConfig.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(m.config.CloudStorageConfig.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	m.client = client
	m.bucket = client.Bucket(m.config.CloudStorageConfig.BucketName)

	// Verify bucket exists
	_, err = m.bucket.Attrs(ctx)
	if err != nil {
		m.client.Close()
		m.client = nil
		return fmt.Errorf("failed to access bucket %s: %w", m.config.CloudStorageConfig.BucketName, err)
	}

	m.loggerManager.Logger().Infof("Cloud storage initialized: bucket=%s", m.config.CloudStorageConfig.BucketName)
	return nil
}

// IsEnabled returns whether cloud storage is enabled and properly initialized
func (m *gcsManager) IsEnabled() bool {
	return m.config.CloudStorageConfig.Enabled && m.client != nil
}

// GetBucketName returns the bucket name
func (m *gcsManager) GetBucketName() string {
	return m.config.CloudStorageConfig.BucketName
}

// GetPathPrefix returns the path prefix
func (m *gcsManager) GetPathPrefix() string {
	prefix := m.config.CloudStorageConfig.PathPrefix
	if prefix == "" {
		return "torrents"
	}
	return prefix
}

// GetSignedURLExpiration returns the signed URL expiration duration
func (m *gcsManager) GetSignedURLExpiration() time.Duration {
	hours := m.config.CloudStorageConfig.SignedURLExpireHours
	if hours <= 0 {
		hours = 3 // Default 3 hours
	}
	return time.Duration(hours) * time.Hour
}

// Upload uploads a file to cloud storage
func (m *gcsManager) Upload(ctx context.Context, objectPath string, reader io.Reader, contentType string) error {
	if !m.IsEnabled() {
		return fmt.Errorf("cloud storage is not enabled")
	}

	obj := m.bucket.Object(objectPath)
	writer := obj.NewWriter(ctx)

	if contentType != "" {
		writer.ContentType = contentType
	}

	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload to %s: %w", objectPath, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize upload to %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Uploaded file to cloud storage: %s", objectPath)
	return nil
}

// UploadWithProgress uploads a file with progress callback
func (m *gcsManager) UploadWithProgress(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64, progressCallback func(bytesWritten int64)) error {
	if !m.IsEnabled() {
		return fmt.Errorf("cloud storage is not enabled")
	}

	obj := m.bucket.Object(objectPath)
	writer := obj.NewWriter(ctx)

	if contentType != "" {
		writer.ContentType = contentType
	}

	// Use a custom writer to track progress
	progressReader := &progressReaderWrapper{
		reader:           reader,
		totalSize:        size,
		bytesRead:        0,
		progressCallback: progressCallback,
	}

	if _, err := io.Copy(writer, progressReader); err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload to %s: %w", objectPath, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize upload to %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Uploaded file to cloud storage: %s (size: %d bytes)", objectPath, size)
	return nil
}

// progressReaderWrapper wraps a reader to track read progress
type progressReaderWrapper struct {
	reader           io.Reader
	totalSize        int64
	bytesRead        int64
	progressCallback func(bytesWritten int64)
}

func (p *progressReaderWrapper) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	if n > 0 {
		p.bytesRead += int64(n)
		if p.progressCallback != nil {
			p.progressCallback(p.bytesRead)
		}
	}
	return n, err
}

// GenerateSignedURL generates a signed URL for temporary access
func (m *gcsManager) GenerateSignedURL(ctx context.Context, objectPath string, expiration time.Duration) (string, error) {
	if !m.IsEnabled() {
		return "", fmt.Errorf("cloud storage is not enabled")
	}

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	url, err := m.bucket.SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for %s: %w", objectPath, err)
	}

	return url, nil
}

// Delete deletes an object from cloud storage
func (m *gcsManager) Delete(ctx context.Context, objectPath string) error {
	if !m.IsEnabled() {
		return fmt.Errorf("cloud storage is not enabled")
	}

	obj := m.bucket.Object(objectPath)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Deleted file from cloud storage: %s", objectPath)
	return nil
}

// Exists checks if an object exists
func (m *gcsManager) Exists(ctx context.Context, objectPath string) (bool, error) {
	if !m.IsEnabled() {
		return false, fmt.Errorf("cloud storage is not enabled")
	}

	obj := m.bucket.Object(objectPath)
	_, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existence of %s: %w", objectPath, err)
	}

	return true, nil
}

// Close closes the GCS client connection
func (m *gcsManager) Close() error {
	if m.client != nil {
		if err := m.client.Close(); err != nil {
			return fmt.Errorf("failed to close GCS client: %w", err)
		}
		m.loggerManager.Logger().Info("Cloud storage client closed")
	}
	return nil
}
