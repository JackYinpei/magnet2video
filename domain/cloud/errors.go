// Package cloud provides the cloud upload bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package cloud

import "errors"

var (
	// ErrMaxRetriesExceeded indicates all retry attempts have been exhausted.
	ErrMaxRetriesExceeded = errors.New("maximum retry count exceeded")

	// ErrCloudStorageDisabled indicates cloud storage is not enabled.
	ErrCloudStorageDisabled = errors.New("cloud storage is disabled")
)
