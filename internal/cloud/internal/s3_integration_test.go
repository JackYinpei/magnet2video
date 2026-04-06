package internal

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"magnet2video/configs"
)

const runS3IntegrationEnv = "RUN_S3_INTEGRATION_TEST"

type integrationLoggerManager struct {
	logger *logrus.Logger
}

func newIntegrationLoggerManager() *integrationLoggerManager {
	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)
	return &integrationLoggerManager{logger: l}
}

func (m *integrationLoggerManager) Logger() *logrus.Logger { return m.logger }
func (m *integrationLoggerManager) Initialize() error      { return nil }
func (m *integrationLoggerManager) Close() error           { return nil }

// TestS3UploadWithProdConfig performs a real upload/download roundtrip against S3/Ceph.
// It is disabled by default; set RUN_S3_INTEGRATION_TEST=1 to run.
func TestS3UploadWithProdConfig(t *testing.T) {
	if os.Getenv(runS3IntegrationEnv) != "1" {
		t.Skipf("set %s=1 to run real S3/Ceph integration test", runS3IntegrationEnv)
	}

	cfg := loadProdConfigForIntegration(t)
	if !cfg.CloudStorageConfig.Enabled {
		t.Fatalf("cloud storage is disabled in config.yml")
	}
	if strings.ToLower(cfg.CloudStorageConfig.Provider) != "s3" {
		t.Fatalf("cloud storage provider must be s3, got %q", cfg.CloudStorageConfig.Provider)
	}

	manager := NewS3Manager(&cfg, newIntegrationLoggerManager())
	t.Cleanup(func() {
		_ = manager.Close()
	})

	if !manager.IsEnabled() {
		t.Fatalf("S3 manager is not enabled; check bucket/endpoint/credentials in configs/config.yml")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	content := []byte(fmt.Sprintf("magnet2video integration %d", time.Now().UnixNano()))
	contentHash := md5.Sum(content)
	objectPath := buildIntegrationObjectPath(manager.GetPathPrefix())

	t.Cleanup(func() {
		_ = manager.Delete(context.Background(), objectPath)
	})

	if err := manager.Upload(ctx, objectPath, strings.NewReader(string(content)), "text/plain; charset=utf-8"); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	exists, err := manager.Exists(ctx, objectPath)
	if err != nil {
		t.Fatalf("exists check failed: %v", err)
	}
	if !exists {
		t.Fatalf("uploaded object does not exist: %s", objectPath)
	}

	signedURL, err := manager.GenerateSignedURL(ctx, objectPath, 2*time.Minute)
	if err != nil {
		t.Fatalf("generate signed URL failed: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	if err != nil {
		t.Fatalf("create signed URL request failed: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("download through signed URL failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		t.Fatalf("signed URL returned status %d: %s", resp.StatusCode, string(body))
	}

	downloaded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read signed URL response failed: %v", err)
	}

	downloadedHash := md5.Sum(downloaded)
	if downloadedHash != contentHash {
		t.Fatalf("downloaded content md5 mismatch: got %x, want %x", downloadedHash, contentHash)
	}
}

func loadProdConfigForIntegration(t *testing.T) configs.Config {
	t.Helper()

	configPath := prodConfigPath(t)
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read %s: %v", configPath, err)
	}

	var cfg configs.Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", configPath, err)
	}

	applyCloudOverridesFromEnv(&cfg)
	return cfg
}

func prodConfigPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	return filepath.Join(repoRoot, "configs", "config.yml")
}

func applyCloudOverridesFromEnv(cfg *configs.Config) {
	if val := os.Getenv("CLOUD_STORAGE_ENABLED"); val != "" {
		cfg.CloudStorageConfig.Enabled = val == "1" || strings.EqualFold(val, "true")
	}
	if val := os.Getenv("CLOUD_STORAGE_PROVIDER"); val != "" {
		cfg.CloudStorageConfig.Provider = val
	}
	if val := os.Getenv("CLOUD_STORAGE_BUCKET_NAME"); val != "" {
		cfg.CloudStorageConfig.BucketName = val
	}
	if val := os.Getenv("S3_BUCKET_NAME"); val != "" {
		cfg.CloudStorageConfig.BucketName = val
	}
	if val := os.Getenv("S3_REGION"); val != "" {
		cfg.CloudStorageConfig.Region = val
	}
	if val := os.Getenv("S3_ACCESS_KEY_ID"); val != "" {
		cfg.CloudStorageConfig.AccessKeyID = val
	}
	if val := os.Getenv("S3_SECRET_ACCESS_KEY"); val != "" {
		cfg.CloudStorageConfig.SecretAccessKey = val
	}
	if val := os.Getenv("S3_ENDPOINT"); val != "" {
		cfg.CloudStorageConfig.Endpoint = val
	}
	if val := os.Getenv("S3_ADDRESSING_STYLE"); val != "" {
		cfg.CloudStorageConfig.AddressingStyle = val
	}
	if val := os.Getenv("S3_SIGNATURE_VERSION"); val != "" {
		cfg.CloudStorageConfig.SignatureVersion = val
	}
}

func buildIntegrationObjectPath(prefix string) string {
	testName := "integration-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ".txt"
	base := strings.Trim(prefix, "/")
	if base == "" {
		return "integration-tests/" + testName
	}
	return base + "/integration-tests/" + testName
}
