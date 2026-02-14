// Package internal provides cloud storage implementation
// Author: Done-0
// Created: 2026-02-01
package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/logger"
)

// Multipart upload threshold: files larger than 100MB use multipart upload
const multipartUploadThreshold = 100 * 1024 * 1024 // 100MB

// s3Manager implements CloudStorageManager for AWS S3 and S3-compatible storage
type s3Manager struct {
	cfg           *configs.Config
	loggerManager logger.LoggerManager
	client        *s3.Client
	uploader      *manager.Uploader
	presignClient *s3.PresignClient
	credentials   aws.CredentialsProvider
	usePathStyle  bool
	useSigV2      bool
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
	m.credentials = awsCfg.Credentials

	addressingStyle := strings.ToLower(strings.TrimSpace(m.cfg.CloudStorageConfig.AddressingStyle))
	signatureVersion := strings.ToLower(strings.TrimSpace(m.cfg.CloudStorageConfig.SignatureVersion))
	m.useSigV2 = isSigV2SignatureVersion(signatureVersion)
	m.usePathStyle = resolveS3PathStyle(addressingStyle, m.cfg.CloudStorageConfig.Endpoint != "", m.useSigV2)

	// Create S3 client options
	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			if m.cfg.CloudStorageConfig.Endpoint != "" {
				o.BaseEndpoint = aws.String(m.cfg.CloudStorageConfig.Endpoint)
			}
			o.UsePathStyle = m.usePathStyle
			if m.useSigV2 {
				o.HTTPSignerV4 = &sigV2Signer{}
			}
		},
	}

	m.client = s3.NewFromConfig(awsCfg, s3Opts...)
	if m.useSigV2 {
		m.presignClient = nil
	} else {
		m.presignClient = s3.NewPresignClient(m.client)
	}

	// Create uploader with multipart support for large files
	m.uploader = manager.NewUploader(m.client, func(u *manager.Uploader) {
		u.PartSize = 64 * 1024 * 1024 // 64MB per part
		u.Concurrency = 3             // 3 concurrent uploads
	})

	// Verify bucket access
	_, err = m.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
	})
	if err != nil {
		m.client = nil
		m.presignClient = nil
		m.uploader = nil
		m.credentials = nil
		return fmt.Errorf("failed to access bucket %s: %w", m.cfg.CloudStorageConfig.BucketName, err)
	}

	signatureLabel := "v4"
	if m.useSigV2 {
		signatureLabel = "v2"
	}
	addressingLabel := "virtual"
	if m.usePathStyle {
		addressingLabel = "path"
	}

	m.loggerManager.Logger().Infof("S3 storage initialized: bucket=%s, addressingStyle=%s, signature=%s",
		m.cfg.CloudStorageConfig.BucketName, addressingLabel, signatureLabel)
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

// UploadWithProgress uploads a file, using multipart upload for large files
func (m *s3Manager) UploadWithProgress(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64, progressCallback func(bytesWritten int64)) error {
	if !m.IsEnabled() {
		return fmt.Errorf("S3 storage is not enabled")
	}

	// Large files: use multipart upload
	if size > multipartUploadThreshold {
		return m.multipartUpload(ctx, objectPath, reader, contentType, size)
	}

	// Small files: use simple PutObject
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

// multipartUpload uploads a large file using S3 multipart upload
func (m *s3Manager) multipartUpload(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(m.cfg.CloudStorageConfig.BucketName),
		Key:    aws.String(objectPath),
		Body:   reader,
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	m.loggerManager.Logger().Infof("Starting multipart upload: %s (size: %d bytes, partSize: 64MB)", objectPath, size)

	_, err := m.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to %s: %w", objectPath, err)
	}

	m.loggerManager.Logger().Infof("Multipart upload completed: %s (size: %d bytes)", objectPath, size)
	return nil
}

// GenerateSignedURL generates a presigned URL for temporary access
func (m *s3Manager) GenerateSignedURL(ctx context.Context, objectPath string, expiration time.Duration) (string, error) {
	if !m.IsEnabled() {
		return "", fmt.Errorf("S3 storage is not enabled")
	}

	if m.useSigV2 {
		return m.generateSignedURLV2(ctx, objectPath, expiration)
	}
	if m.presignClient == nil {
		return "", fmt.Errorf("S3 presign client is not initialized")
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

// generateSignedURLV2 generates an S3 Signature V2 signed URL.
func (m *s3Manager) generateSignedURLV2(ctx context.Context, objectPath string, expiration time.Duration) (string, error) {
	if m.credentials == nil {
		return "", fmt.Errorf("S3 credentials provider is not initialized")
	}

	creds, err := m.credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve S3 credentials for signed URL: %w", err)
	}
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return "", fmt.Errorf("S3 credentials are missing access key or secret key")
	}

	endpointURL, err := parseS3EndpointURL(m.cfg.CloudStorageConfig.Endpoint)
	if err != nil {
		return "", err
	}

	key := strings.TrimLeft(objectPath, "/")
	if key == "" {
		return "", fmt.Errorf("object path cannot be empty")
	}
	escapedKey := escapeS3ObjectKey(key)

	var rawPath string
	if m.usePathStyle {
		rawPath = joinURLPath(endpointURL.Path, m.cfg.CloudStorageConfig.BucketName, escapedKey)
	} else {
		endpointURL.Host = fmt.Sprintf("%s.%s", m.cfg.CloudStorageConfig.BucketName, endpointURL.Host)
		rawPath = joinURLPath(endpointURL.Path, escapedKey)
	}
	// Set RawPath to the pre-encoded path so that url.URL.String() uses it directly,
	// avoiding double-encoding (e.g. %E5 → %25E5) of non-ASCII characters.
	endpointURL.RawPath = rawPath
	if decoded, err := url.PathUnescape(rawPath); err == nil {
		endpointURL.Path = decoded
	} else {
		endpointURL.Path = rawPath
	}

	expires := strconv.FormatInt(time.Now().UTC().Add(expiration).Unix(), 10)
	query := endpointURL.Query()
	query.Set("AWSAccessKeyId", creds.AccessKeyID)
	query.Set("Expires", expires)
	if creds.SessionToken != "" {
		query.Set("x-amz-security-token", creds.SessionToken)
	}
	query.Del("Signature")
	endpointURL.RawQuery = query.Encode()

	req := &http.Request{
		Method: http.MethodGet,
		URL:    endpointURL,
		Header: http.Header{},
	}
	signature := sigV2Signature(creds.SecretAccessKey, buildSigV2StringToSign(req, expires))
	query.Set("Signature", signature)
	endpointURL.RawQuery = query.Encode()

	return endpointURL.String(), nil
}

func parseS3EndpointURL(rawEndpoint string) (*url.URL, error) {
	endpoint := strings.TrimSpace(rawEndpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("S3 endpoint is required when using signature v2 signed URL")
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 endpoint %q: %w", rawEndpoint, err)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid S3 endpoint %q: missing host", rawEndpoint)
	}

	return parsed, nil
}

func resolveS3PathStyle(addressingStyle string, hasCustomEndpoint bool, forcePathStyle bool) bool {
	if forcePathStyle {
		return true
	}

	switch addressingStyle {
	case "path", "path-style", "url":
		return true
	case "virtual", "virtual-hosted", "host":
		return false
	default:
		// S3-compatible endpoints and SigV2 deployments typically need path-style.
		return hasCustomEndpoint || forcePathStyle
	}
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
