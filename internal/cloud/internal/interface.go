// Package internal provides cloud storage implementation.
//
// Only S3 (and S3-compatible services such as MinIO / Ceph / hi168) are
// supported — the previous GCS path was removed. The interface lives in
// this file so consumers don't depend on a specific implementation file.
//
// Author: Done-0
// Created: 2026-02-01
package internal

import (
	"context"
	"io"
	"time"
)

// CloudStorageManager defines cloud storage operations interface.
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
