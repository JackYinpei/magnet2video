// Package handler provides worker-side message handler for file-ops-jobs.
// Author: magnet2video
// Created: 2026-05-01
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"magnet2video/configs"
	"magnet2video/internal/events/gateway"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	torrentTypes "magnet2video/internal/torrent/types"
)

// FileOpsHandler executes filesystem-mutating commands sent from the server.
// It refuses paths that escape the configured download directory and reports
// outcome through the worker gateway.
type FileOpsHandler struct {
	config        *configs.Config
	loggerManager logger.LoggerManager
	gateway       gateway.WorkerGateway
}

// NewFileOpsHandler builds a FileOpsHandler.
func NewFileOpsHandler(
	config *configs.Config,
	loggerManager logger.LoggerManager,
	workerGateway gateway.WorkerGateway,
) *FileOpsHandler {
	return &FileOpsHandler{
		config:        config,
		loggerManager: loggerManager,
		gateway:       workerGateway,
	}
}

// Handle dispatches one file-ops message.
func (h *FileOpsHandler) Handle(ctx context.Context, msg *queue.Message) error {
	var op torrentTypes.FileOpMessage
	if err := json.Unmarshal(msg.Value, &op); err != nil {
		h.loggerManager.Logger().Errorf("unmarshal file-op job: %v", err)
		return nil
	}

	// Multi-worker routing: in split deployments only the worker that
	// actually holds the files should run a deletion. Any other worker
	// that grabs the same message off the shared queue must put it back
	// so the rightful owner can pick it up. Returning queue.ErrNotForMe
	// causes the consumer to NACK + requeue with a small delay.
	if op.TargetWorkerID != "" && h.gateway != nil && op.TargetWorkerID != h.gateway.WorkerID() {
		h.loggerManager.Logger().Debugf(
			"file-op not for me: op=%s target=%s self=%s torrent=%d — requeue",
			op.Op, op.TargetWorkerID, h.gateway.WorkerID(), op.TorrentID,
		)
		return queue.ErrNotForMe
	}

	result := eventTypes.FileOpResultPayload{
		OpID:      op.OpID,
		Op:        op.Op,
		TorrentID: op.TorrentID,
		InfoHash:  op.InfoHash,
	}

	switch op.Op {
	case torrentTypes.FileOpDeleteTorrentDir:
		h.handleDeleteTorrentDir(ctx, op, &result)
	case torrentTypes.FileOpDeleteDerived, torrentTypes.FileOpDeletePaths:
		h.handleDeletePaths(ctx, op, &result)
	default:
		result.ErrorMsg = "unknown op: " + op.Op
	}

	if result.ErrorMsg != "" {
		_ = h.gateway.FileOpFailed(ctx, result)
		h.loggerManager.Logger().Warnf("file-op failed: op=%s opID=%s torrent=%d err=%s",
			op.Op, op.OpID, op.TorrentID, result.ErrorMsg)
		return nil
	}
	_ = h.gateway.FileOpCompleted(ctx, result)
	h.loggerManager.Logger().Infof("file-op completed: op=%s opID=%s torrent=%d deleted=%d notFound=%d",
		op.Op, op.OpID, op.TorrentID, result.DeletedCount, result.NotFoundCount)
	return nil
}

// downloadDir returns the configured download directory cleaned to an absolute path.
// Empty string means the worker is misconfigured and we should refuse all ops.
func (h *FileOpsHandler) downloadDir() string {
	dir := strings.TrimSpace(h.config.TorrentConfig.DownloadDir)
	if dir == "" {
		return ""
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(dir)
}

// withinDownloadDir reports whether absPath sits under the configured
// download directory. Both arguments must already be cleaned + absolute.
// We refuse to delete anything outside this root.
func (h *FileOpsHandler) withinDownloadDir(absPath string) bool {
	root := h.downloadDir()
	if root == "" || absPath == "" {
		return false
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolveAbs takes a path that may be absolute (worker view) or relative
// (resolved against download dir) and returns the cleaned absolute form.
func (h *FileOpsHandler) resolveAbs(p string) string {
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(h.downloadDir(), p)
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

func (h *FileOpsHandler) handleDeleteTorrentDir(_ context.Context, op torrentTypes.FileOpMessage, result *eventTypes.FileOpResultPayload) {
	dir := h.downloadDir()
	if dir == "" {
		result.ErrorMsg = "worker download dir not configured"
		return
	}
	if op.TorrentName == "" {
		result.ErrorMsg = "torrent_name is required"
		return
	}
	target := filepath.Clean(filepath.Join(dir, op.TorrentName))
	if !h.withinDownloadDir(target) {
		result.ErrorMsg = fmt.Sprintf("refusing path outside download dir: %s", target)
		return
	}
	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.NotFoundCount = 1
			return
		}
		result.ErrorMsg = err.Error()
		return
	}
	if err := os.RemoveAll(target); err != nil {
		result.ErrorMsg = err.Error()
		return
	}
	result.DeletedCount = 1
}

func (h *FileOpsHandler) handleDeletePaths(_ context.Context, op torrentTypes.FileOpMessage, result *eventTypes.FileOpResultPayload) {
	for _, p := range op.Paths {
		abs := h.resolveAbs(p)
		if !h.withinDownloadDir(abs) {
			h.loggerManager.Logger().Warnf("file-op skipping path outside download dir: %s", abs)
			continue
		}
		if err := os.Remove(abs); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.NotFoundCount++
				continue
			}
			// Don't abort the whole batch on a single failure — record and
			// keep going. The first error wins on the result envelope.
			if result.ErrorMsg == "" {
				result.ErrorMsg = fmt.Sprintf("delete %s: %v", abs, err)
			}
			h.loggerManager.Logger().Warnf("file-op delete %s failed: %v", abs, err)
			continue
		}
		result.DeletedCount++
	}
}
