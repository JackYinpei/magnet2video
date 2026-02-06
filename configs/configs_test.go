// Package configs provides tests for configuration helpers
// Author: Done-0
// Created: 2026-02-04
package configs

import (
	"testing"
)

type testInner struct {
	Name string
	Age  int
}

type testOuter struct {
	Inner testInner
	Count int
}

func TestCompareStructs_NoChanges(t *testing.T) {
	oldObj := testOuter{Inner: testInner{Name: "Alice", Age: 10}, Count: 1}
	newObj := testOuter{Inner: testInner{Name: "Alice", Age: 10}, Count: 1}

	changes := make(map[string][2]any)
	if ok := compareStructs(oldObj, newObj, "", changes); !ok {
		t.Fatalf("compareStructs() should return true for same type")
	}
	if len(changes) != 0 {
		t.Fatalf("compareStructs() changes = %v, want empty", changes)
	}
}

func TestCompareStructs_WithChanges(t *testing.T) {
	oldObj := testOuter{Inner: testInner{Name: "Alice", Age: 10}, Count: 1}
	newObj := testOuter{Inner: testInner{Name: "Bob", Age: 11}, Count: 2}

	changes := make(map[string][2]any)
	if ok := compareStructs(oldObj, newObj, "", changes); !ok {
		t.Fatalf("compareStructs() should return true for same type")
	}

	if changes["Inner.Name"][0] != "Alice" || changes["Inner.Name"][1] != "Bob" {
		t.Fatalf("Inner.Name change = %v, want Alice -> Bob", changes["Inner.Name"])
	}
	if changes["Inner.Age"][0] != 10 || changes["Inner.Age"][1] != 11 {
		t.Fatalf("Inner.Age change = %v, want 10 -> 11", changes["Inner.Age"])
	}
	if changes["Count"][0] != 1 || changes["Count"][1] != 2 {
		t.Fatalf("Count change = %v, want 1 -> 2", changes["Count"])
	}
}

func TestCompareStructs_TypeMismatch(t *testing.T) {
	oldObj := testOuter{Inner: testInner{Name: "Alice", Age: 10}, Count: 1}
	newObj := struct{
		Inner testInner
		Count int
		Flag  bool
	}{Inner: testInner{Name: "Alice", Age: 10}, Count: 1, Flag: true}

	changes := make(map[string][2]any)
	if ok := compareStructs(oldObj, newObj, "", changes); ok {
		t.Fatalf("compareStructs() should return false for type mismatch")
	}
}

func TestCompareStructs_NonStruct(t *testing.T) {
	changes := make(map[string][2]any)
	if ok := compareStructs(1, 1, "", changes); !ok {
		t.Fatalf("compareStructs() should return true for non-struct types")
	}
	if len(changes) != 0 {
		t.Fatalf("compareStructs() changes = %v, want empty", changes)
	}
}

func TestOverrideFromEnv(t *testing.T) {
	t.Setenv("DB_DIALECT", "mysql")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "root")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "appdb")
	t.Setenv("DB_PATH", "/tmp/app.db")

	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_PASSWORD", "redispass")
	t.Setenv("REDIS_DB", "1")

	t.Setenv("APP_HOST", "0.0.0.0")
	t.Setenv("APP_PORT", "8080")

	t.Setenv("JWT_SECRET", "jwt-secret")

	t.Setenv("FROM_EMAIL", "noreply@example.com")
	t.Setenv("EMAIL_SMTP", "smtp.example.com")

	t.Setenv("SUPER_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("SUPER_ADMIN_PASSWORD", "adminpass")

	t.Setenv("CLOUD_STORAGE_ENABLED", "1")
	t.Setenv("CLOUD_STORAGE_PROVIDER", "s3")
	t.Setenv("CLOUD_STORAGE_BUCKET_NAME", "cloud-bucket")
	t.Setenv("GCS_CREDENTIALS_FILE", "/tmp/gcs.json")
	t.Setenv("GCS_BUCKET_NAME", "gcs-bucket")
	t.Setenv("S3_REGION", "us-east-1")
	t.Setenv("S3_ACCESS_KEY_ID", "ak")
	t.Setenv("S3_SECRET_ACCESS_KEY", "sk")
	t.Setenv("S3_ENDPOINT", "https://s3.example.com")
	t.Setenv("S3_BUCKET_NAME", "s3-bucket")

	config := &Config{}
	overrideFromEnv(config)

	if config.DBConfig.DBDialect != "mysql" || config.DBConfig.DBHost != "localhost" {
		t.Fatalf("DB overrides failed: %+v", config.DBConfig)
	}
	if config.RedisConfig.RedisHost != "127.0.0.1" || config.RedisConfig.RedisDB != "1" {
		t.Fatalf("Redis overrides failed: %+v", config.RedisConfig)
	}
	if config.AppConfig.AppHost != "0.0.0.0" || config.AppConfig.AppPort != "8080" {
		t.Fatalf("App overrides failed: %+v", config.AppConfig)
	}
	if config.AppConfig.JWT.Secret != "jwt-secret" {
		t.Fatalf("JWT overrides failed: %+v", config.AppConfig.JWT)
	}
	if config.AppConfig.Email.FromEmail != "noreply@example.com" || config.AppConfig.Email.EmailSmtp != "smtp.example.com" {
		t.Fatalf("Email overrides failed: %+v", config.AppConfig.Email)
	}
	if config.AppConfig.User.SuperAdminEmail != "admin@example.com" || config.AppConfig.User.SuperAdminPassword != "adminpass" {
		t.Fatalf("Admin overrides failed: %+v", config.AppConfig.User)
	}
	if !config.CloudStorageConfig.Enabled {
		t.Fatalf("Cloud storage enabled override failed")
	}
	if config.CloudStorageConfig.Provider != "s3" {
		t.Fatalf("Cloud storage provider override failed: %q", config.CloudStorageConfig.Provider)
	}
	if config.CloudStorageConfig.BucketName != "s3-bucket" {
		t.Fatalf("Cloud storage bucket override failed: %q", config.CloudStorageConfig.BucketName)
	}
	if config.CloudStorageConfig.CredentialsFile != "/tmp/gcs.json" {
		t.Fatalf("Cloud storage credentials override failed: %q", config.CloudStorageConfig.CredentialsFile)
	}
	if config.CloudStorageConfig.Endpoint != "https://s3.example.com" {
		t.Fatalf("Cloud storage endpoint override failed: %q", config.CloudStorageConfig.Endpoint)
	}
}

func TestGetConfig_NotInitialized(t *testing.T) {
	old := instance
	instance = nil
	t.Cleanup(func() { instance = old })

	if _, err := GetConfig(); err == nil {
		t.Fatalf("GetConfig() should return error when not initialized")
	}
}
