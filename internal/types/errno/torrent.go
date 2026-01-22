// Package errno provides torrent-related error code definitions
// Author: Done-0
// Created: 2026-01-22
package errno

import (
	"github.com/Done-0/gin-scaffold/internal/utils/errorx/code"
)

// Torrent-related error codes: 20000 ~ 20999
// Used: 20001-20010
// Next available: 20011
const (
	ErrInvalidMagnetURI     = 20001 // Invalid magnet URI
	ErrTorrentNotFound      = 20002 // Torrent not found
	ErrTorrentAlreadyExists = 20003 // Torrent already exists
	ErrTorrentDownloading   = 20004 // Torrent is currently downloading
	ErrTorrentParseFailed   = 20005 // Failed to parse torrent metadata
	ErrTorrentAddFailed     = 20006 // Failed to add torrent
	ErrTorrentRemoveFailed  = 20007 // Failed to remove torrent
	ErrNoFilesSelected      = 20008 // No files selected for download
	ErrFileNotFound         = 20009 // File not found
	ErrFileNotStreamable    = 20010 // File is not streamable
)

func init() {
	code.Register(ErrInvalidMagnetURI, "invalid magnet URI: {{.msg}}")
	code.Register(ErrTorrentNotFound, "torrent not found: {{.info_hash}}")
	code.Register(ErrTorrentAlreadyExists, "torrent already exists: {{.info_hash}}")
	code.Register(ErrTorrentDownloading, "torrent is currently downloading: {{.info_hash}}")
	code.Register(ErrTorrentParseFailed, "failed to parse torrent metadata: {{.msg}}")
	code.Register(ErrTorrentAddFailed, "failed to add torrent: {{.msg}}")
	code.Register(ErrTorrentRemoveFailed, "failed to remove torrent: {{.msg}}")
	code.Register(ErrNoFilesSelected, "no files selected for download")
	code.Register(ErrFileNotFound, "file not found: {{.path}}")
	code.Register(ErrFileNotStreamable, "file is not streamable: {{.path}}")
}
