// Package internal provides cloud storage implementation
// Author: Done-0
// Created: 2026-02-01
package internal

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/logger"
)

// s3Manager implements CloudStorageManager for AWS S3 and S3-compatible storage
type s3Manager struct {
	cfg           *configs.Config
	loggerManager logger.LoggerManager
	client        *s3.Client
	presignClient *s3.PresignClient
}

// NewS3Manager creates a new S3 CloudStorageManager instance
func NewS3Manager(cfg *configs.Config, loggerManager logger.LoggerManager) CloudStorageManager {
	manager := &s3Manager{
		cfg:           cfg,
		loggerManager: loggerManager,
	}

	if cfg.CloudStorageConfig.Enabled {
		if err := manager.initialize(); err != nil {
			loggerManager.Logger().Errorf("failed to initialize S3 storage: %v", err)
		}
	}

	return manager
}

// initialize initializes the S3 client
func (m *s3Manager) initialize() error {
	ctx := context.Background()

	var opts []func(*awsconfig.LoadOptions) error

	// Configure credentials if provided
	if m.cfg.CloudStorageConfig.AccessKeyID != "" && m.cfg.CloudStorageConfig.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				m.cfg.CloudStorageConfig.AccessKeyID,
				m.cfg.CloudStorageConfig.SecretAccessKey,
				"",
			),
		))
	}

	// Configure region
	if m.cfg.CloudStorageConfig.Region != "" {
		opts = append(opts, awsconfig.WithRegion(m.cfg.CloudStorageConfig.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	s3Opts := []func(*s3.Options){}

	// Configure custom endpoint for S3-compatible storage (MinIO, etc.)
	if m.cfg.CloudStorageConfig.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(m.cfg.CloudStorageConfig.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible storage
		})
	}

	m.client = s3.NewFromConfig(awsCfg, s3Opts...)
	m.presignClient = s3.NewPresignClient(m.client)

	// Verify bucket access
	_, err = m.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
	})
	if err != nil {
		m.client = nil
		m.presignClient = nil
		return fmt.Errorf("failed to access bucket %s: %w", m.cfg.CloudStorageConfig.BucketName, err)
	}

	m.loggerManager.Logger().Infof("S3 storage initialized: bucket=%s", m.cfg.CloudStorageConfig.BucketName)
	return nil
}

// IsEnabled returns whether S3 storage is enabled and properly initialized
func (m *s3Manager) IsEnabled() bool {
	return m.cfg.CloudStorageConfig.Enabled && m.client != nil
}

// GetBucketName returns the bucket name
func (m *s3Manager) GetBucketName() string {
	return m.cfg.CloudStorageConfig.BucketName
}

// GetPathPrefix returns the path prefix
func (m *s3Manager) GetPathPrefix() string {
	prefix := m.cfg.CloudStorageConfig.PathPrefix
	if prefix == "" {
		return "torrents"
	}
	return prefix
}

// GetSignedURLExpiration returns the signed URL expiration duration
func (m *s3Manager) GetSignedURLExpiration() time.Duration {
	hours := m.cfg.CloudStorageConfig.SignedURLExpireHours
	if hours <= 0 {
		hours = 3
	}
	return time.Duration(hours) * time.Hour
}

// Upload uploads a file to S3
func (m *s3Manager) Upload(ctx context.Context, objectPath string, reader io.Reader, contentType string) error {
	if !m.IsEnabled() {
		return fmt.Errorf("S3 storage is not enabled")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:    aws.String(objectPath),
		Body:   reader,
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := m.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Uploaded file to S3: %s", objectPath)
	return nil
}

// UploadWithProgress uploads a file with progress callback
func (m *s3Manager) UploadWithProgress(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64, progressCallback func(bytesWritten int64)) error {
	if !m.IsEnabled() {
		return fmt.Errorf("S3 storage is not enabled")
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:           aws.String(objectPath),
		Body:          reader,
		ContentLength: aws.Int64(size),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := m.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Uploaded file to S3: %s (size: %d bytes)", objectPath, size)
	return nil
}

// GenerateSignedURL generates a presigned URL for temporary access
func (m *s3Manager) GenerateSignedURL(ctx context.Context, objectPath string, expiration time.Duration) (string, error) {
	if !m.IsEnabled() {
		return "", fmt.Errorf("S3 storage is not enabled")
	}

	presignResult, err := m.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:    aws.String(objectPath),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", objectPath, err)
	}

	return presignResult.URL, nil
}

// Delete deletes an object from S3
func (m *s3Manager) Delete(ctx context.Context, objectPath string) error {
	if !m.IsEnabled() {
		return fmt.Errorf("S3 storage is not enabled")
	}

	_, err := m.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:    aws.String(objectPath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Deleted file from S3: %s", objectPath)
	return nil
}

// Exists checks if an object exists in S3
func (m *s3Manager) Exists(ctx context.Context, objectPath string) (bool, error) {
	if !m.IsEnabled() {
		return false, fmt.Errorf("S3 storage is not enabled")
	}

	_, err := m.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:    aws.String(objectPath),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}

	return true, nil
}

// Close closes the S3 client (no-op for S3)
func (m *s3Manager) Close() error {
	m.loggerManager.Logger().Info("S3 storage client closed")
	return nil
}
