// Package transcode provides the transcode bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package transcode

import "errors"

var (
	// ErrJobAlreadyRunning indicates the job is already processing.
	ErrJobAlreadyRunning = errors.New("transcode job is already running")

	// ErrJobCannotRetry indicates the job is not in a retryable state.
	ErrJobCannotRetry = errors.New("transcode job cannot be retried")

	// ErrJobNotFound indicates the job does not exist.
	ErrJobNotFound = errors.New("transcode job not found")
)
