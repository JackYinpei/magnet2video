// Package transcode provides the transcode bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package transcode

import "context"

// JobRepository abstracts persistence for the TranscodeJob aggregate.
type JobRepository interface {
	FindByID(ctx context.Context, id int64) (*Job, error)
	Create(ctx context.Context, job *Job) error
	Save(ctx context.Context, job *Job) error
	FindByTorrentID(ctx context.Context, torrentID int64) ([]Job, error)
	DeleteByTorrentID(ctx context.Context, torrentID int64) error
	ListAll(ctx context.Context, page, pageSize int, status int) ([]Job, int64, error)
}
