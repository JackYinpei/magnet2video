// Package processor: stuck-state reaper.
//
// The event-driven path can leave torrent_files rows stranded in mid-states
// (Pending / Processing / Uploading) when the worker crashes mid-job, the
// queue drops a message, or any code path that flips a row to Pending fails
// to follow through. The reaper is a periodic safety net: it scans for rows
// whose updated_at is older than per-state thresholds, demotes them to
// Failed, then triggers torrent-level aggregate recompute so the UI reflects
// the demotion.
//
// Author: magnet2video
// Created: 2026-04-30
package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"magnet2video/configs"
	torrentModel "magnet2video/internal/model/torrent"
)

// Reaper periodically scans torrent_files for rows stuck in mid-states and
// demotes them to Failed once their updated_at is older than the configured
// per-state timeout. After demotion it triggers torrent-level recompute.
type Reaper struct {
	processor *WorkerEventProcessor
	cfg       reaperOpts
}

// reaperOpts is the operational config after defaults are applied.
type reaperOpts struct {
	enabled           bool
	interval          time.Duration
	pendingTimeout    time.Duration
	processingTimeout time.Duration
	uploadingTimeout  time.Duration
}

// Defaults: healthy long jobs should never trip these thresholds; only
// crashed workers and lost messages should.
const (
	defaultReaperInterval          = 5 * time.Minute
	defaultReaperPendingTimeout    = 30 * time.Minute
	defaultReaperProcessingTimeout = 4 * time.Hour
	defaultReaperUploadingTimeout  = 1 * time.Hour
)

// NewReaperFromConfig builds a reaper from a configs.ReaperConfig, filling
// any non-positive timeout with the corresponding default. Enabled is taken
// verbatim from config — the safety net is opt-in via EVENTS.REAPER.ENABLED.
func NewReaperFromConfig(p *WorkerEventProcessor, cfg configs.ReaperConfig) *Reaper {
	return &Reaper{
		processor: p,
		cfg: reaperOpts{
			enabled:           cfg.Enabled,
			interval:          secondsOrDefault(cfg.IntervalSeconds, defaultReaperInterval),
			pendingTimeout:    secondsOrDefault(cfg.PendingTimeoutSeconds, defaultReaperPendingTimeout),
			processingTimeout: secondsOrDefault(cfg.ProcessingTimeoutSeconds, defaultReaperProcessingTimeout),
			uploadingTimeout:  secondsOrDefault(cfg.UploadingTimeoutSeconds, defaultReaperUploadingTimeout),
		},
	}
}

func secondsOrDefault(s int, d time.Duration) time.Duration {
	if s <= 0 {
		return d
	}
	return time.Duration(s) * time.Second
}

// Run blocks until ctx is cancelled. Errors from individual scans are logged
// but do not stop the loop — a transient DB blip should not kill the safety
// net.
func (r *Reaper) Run(ctx context.Context) {
	if !r.cfg.enabled {
		log.Println("Reaper: disabled by config; not running")
		return
	}
	log.Printf("Reaper: started (interval=%s pending_timeout=%s processing_timeout=%s uploading_timeout=%s)",
		r.cfg.interval, r.cfg.pendingTimeout, r.cfg.processingTimeout, r.cfg.uploadingTimeout)

	// Run once at startup so a fresh restart clears stale state from a previous
	// crash without waiting an interval.
	if err := r.reapOnce(ctx); err != nil {
		log.Printf("Reaper: initial scan error: %v", err)
	}

	ticker := time.NewTicker(r.cfg.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Reaper: shutting down")
			return
		case <-ticker.C:
			if err := r.reapOnce(ctx); err != nil {
				log.Printf("Reaper: scan error: %v", err)
			}
		}
	}
}

// reapOnce runs a single scan + demotion pass across all four mid-states.
func (r *Reaper) reapOnce(ctx context.Context) error {
	now := time.Now().Unix()
	db := r.processor.dbManager.DB().WithContext(ctx)

	affectedTranscode := make(map[int64]struct{})
	affectedCloud := make(map[int64]struct{})

	// Transcode applies only to original files — transcoded outputs do not get
	// re-transcoded, and recomputeTranscodeStatus filters the same way.
	const originalsOnly = "(source = '' OR source = 'original')"

	if err := r.scanAndDemote(db, now,
		"transcode_status", "transcode_error", originalsOnly,
		torrentModel.TranscodeStatusPending, torrentModel.TranscodeStatusFailed,
		r.cfg.pendingTimeout, "transcode pending", affectedTranscode,
	); err != nil {
		return err
	}
	if err := r.scanAndDemote(db, now,
		"transcode_status", "transcode_error", originalsOnly,
		torrentModel.TranscodeStatusProcessing, torrentModel.TranscodeStatusFailed,
		r.cfg.processingTimeout, "transcode processing", affectedTranscode,
	); err != nil {
		return err
	}
	if err := r.scanAndDemote(db, now,
		"cloud_upload_status", "cloud_upload_error", "",
		torrentModel.CloudUploadStatusPending, torrentModel.CloudUploadStatusFailed,
		r.cfg.pendingTimeout, "cloud upload pending", affectedCloud,
	); err != nil {
		return err
	}
	if err := r.scanAndDemote(db, now,
		"cloud_upload_status", "cloud_upload_error", "",
		torrentModel.CloudUploadStatusUploading, torrentModel.CloudUploadStatusFailed,
		r.cfg.uploadingTimeout, "cloud upload uploading", affectedCloud,
	); err != nil {
		return err
	}

	for tid := range affectedTranscode {
		if err := r.processor.recomputeTranscodeStatus(tid); err != nil {
			log.Printf("Reaper: recomputeTranscodeStatus(torrent=%d): %v", tid, err)
		}
	}
	for tid := range affectedCloud {
		if err := r.processor.recomputeCloudStatus(tid); err != nil {
			log.Printf("Reaper: recomputeCloudStatus(torrent=%d): %v", tid, err)
		}
	}
	return nil
}

// scanAndDemote selects rows in a single mid-state past their timeout and
// flips them to failedStatus with an explanatory error message. The UPDATE
// is conditional on the status still being currentStatus, which protects
// against a real event landing between our scan and our update.
func (r *Reaper) scanAndDemote(
	db *gorm.DB,
	now int64,
	statusCol, errorCol, sourceFilter string,
	currentStatus, failedStatus int,
	timeout time.Duration,
	label string,
	outAffected map[int64]struct{},
) error {
	cutoff := now - int64(timeout.Seconds())

	q := db.Model(&torrentModel.TorrentFile{}).
		Where(statusCol+" = ? AND updated_at < ?", currentStatus, cutoff)
	if sourceFilter != "" {
		q = q.Where(sourceFilter)
	}

	var stuck []torrentModel.TorrentFile
	if err := q.Find(&stuck).Error; err != nil {
		return fmt.Errorf("scan %s: %w", label, err)
	}
	if len(stuck) == 0 {
		return nil
	}

	for _, f := range stuck {
		errMsg := fmt.Sprintf("reaper: %s state stale for %ds (timeout=%s) — possible worker crash or queue loss",
			label, now-f.UpdatedAt, timeout)
		res := db.Model(&torrentModel.TorrentFile{}).
			Where("id = ? AND "+statusCol+" = ?", f.ID, currentStatus).
			Updates(map[string]any{
				statusCol: failedStatus,
				errorCol:  errMsg,
			})
		if err := res.Error; err != nil {
			log.Printf("Reaper: failed to demote file_id=%d (%s): %v", f.ID, label, err)
			continue
		}
		if res.RowsAffected == 0 {
			continue // Lost the race with a real event — fine.
		}
		log.Printf("Reaper: demoted file_id=%d torrent_id=%d %s→Failed (stale %ds)",
			f.ID, f.TorrentID, label, now-f.UpdatedAt)
		outAffected[f.TorrentID] = struct{}{}
	}
	return nil
}
