// Package errno provides cloud storage error codes
// Author: Done-0
// Created: 2026-02-01
package errno

import "github.com/Done-0/gin-scaffold/internal/utils/errorx"

func init() {
	// Cloud Storage errors (20400-20499)
	errorx.Register(20400, "cloud storage not enabled")
	errorx.Register(20401, "cloud upload failed: {{.msg}}")
	errorx.Register(20402, "file not uploaded to cloud")
	errorx.Register(20403, "signed URL generation failed: {{.msg}}")
	errorx.Register(20404, "cloud storage initialization failed")
}

// Cloud Storage error code constants
const (
	ErrCloudStorageDisabled   = 20400 // Cloud storage not enabled
	ErrCloudUploadFailed      = 20401 // Cloud upload failed
	ErrFileNotInCloud         = 20402 // File not uploaded to cloud
	ErrSignedURLFailed        = 20403 // Signed URL generation failed
	ErrCloudStorageInitFailed = 20404 // Cloud storage initialization failed
)
