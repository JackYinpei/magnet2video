// Package types provides torrent-related message type definitions for the queue.
// Author: magnet2video
// Created: 2026-04-20
package types

// Topic names for queue messages owned by this package.
const (
	TopicDownloadJobs = "download-jobs"
	TopicFileOps      = "file-ops-jobs"
)

// File-op kinds. The worker is the only party that owns the download
// directory and any derived (transcoded/extracted) files, so any deletion
// triggered by the server has to be expressed as one of these and dispatched
// as a queue message.
const (
	// FileOpDeleteTorrentDir removes the entire on-disk tree for a torrent
	// (download dir + name subdir + any transcoded/extracted siblings).
	FileOpDeleteTorrentDir = "delete-torrent-dir"

	// FileOpDeleteDerived removes only the derived files (transcoded /
	// extracted) for a torrent, keeping the originals intact. Used by
	// "reset transcode" admin flows.
	FileOpDeleteDerived = "delete-derived"

	// FileOpDeletePaths removes a specific list of paths (must be inside
	// the worker's download dir; the worker enforces this).
	FileOpDeletePaths = "delete-paths"
)

// FileOpMessage is the envelope for filesystem-mutating commands sent from
// server to worker. The worker is expected to ack with a worker-events
// message of type EventTypeFileOpCompleted (success) or EventTypeFileOpFailed.
type FileOpMessage struct {
	Op           string   `json:"op"`            // one of FileOp* constants
	OpID         string   `json:"op_id"`         // server-generated id, echoed in the result event
	TorrentID    int64    `json:"torrent_id"`    // 0 if not torrent-scoped
	InfoHash     string   `json:"info_hash"`     // optional, for logging
	DownloadPath string   `json:"download_path"` // absolute path on worker; advisory
	TorrentName  string   `json:"torrent_name"`  // used for FileOpDeleteTorrentDir
	Paths        []string `json:"paths"`         // used for FileOpDeletePaths / FileOpDeleteDerived
	CreatorID    int64    `json:"creator_id"`    // for audit/log
}
