// Package torrent provides the torrent bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package torrent

import "errors"

var (
	// ErrInvalidStateTransition indicates an illegal status change.
	ErrInvalidStateTransition = errors.New("invalid state transition")

	// ErrNotOwner indicates the user does not own the torrent.
	ErrNotOwner = errors.New("not the owner of this torrent")

	// ErrCannotDeleteWhileDownloading prevents deletion during active download.
	ErrCannotDeleteWhileDownloading = errors.New("cannot delete while downloading")

	// ErrCannotDeleteWhileUploading prevents local file deletion during upload.
	ErrCannotDeleteWhileUploading = errors.New("cannot delete local files while uploading to cloud")

	// ErrFileIndexOutOfRange indicates a file index beyond the files slice.
	ErrFileIndexOutOfRange = errors.New("file index out of range")

	// ErrTorrentNotFound indicates the torrent does not exist.
	ErrTorrentNotFound = errors.New("torrent not found")
)
