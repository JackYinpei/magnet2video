// Package torrent provides the torrent bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package torrent

import "context"

// TorrentRepository abstracts persistence for the Torrent aggregate.
type TorrentRepository interface {
	FindByID(ctx context.Context, id int64) (*Torrent, error)
	FindByInfoHash(ctx context.Context, infoHash string) (*Torrent, error)
	FindByIDWithFiles(ctx context.Context, id int64) (*Torrent, error)
	FindByInfoHashWithFiles(ctx context.Context, infoHash string) (*Torrent, error)
	Save(ctx context.Context, t *Torrent) error
	Create(ctx context.Context, t *Torrent) error
	Delete(ctx context.Context, id int64) error

	// List queries
	ListByCreator(ctx context.Context, creatorID int64, page, pageSize int) ([]Torrent, int64, error)
	ListPublic(ctx context.Context, includeInternal bool, page, pageSize int) ([]Torrent, int64, error)
	ListAll(ctx context.Context, search string, status int, creatorID int64, page, pageSize int) ([]Torrent, int64, error)

	// File operations
	SaveFile(ctx context.Context, torrentID int64, f *TorrentFile) error
	UpdateFileFields(ctx context.Context, torrentID int64, fileIndex int, updates map[string]interface{}) error

	// Special queries
	FindActiveForRestore(ctx context.Context) ([]Torrent, error)
	FindCompletedPendingTranscode(ctx context.Context) ([]Torrent, error)

	// Aggregate status updates
	UpdateCloudSummary(ctx context.Context, torrentID int64) error
	UpdateTranscodeSummary(ctx context.Context, torrentID int64) error
}
