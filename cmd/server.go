// Package cmd: server-only bootstrap. Runs Gin, DB, Redis, and the worker-event
// & heartbeat consumers. Does NOT run transcode / cloud-upload / download job
// consumers — those belong to the remote worker.
// Author: magnet2video
// Created: 2026-04-20
package cmd

import (
	"context"
	"log"

	"magnet2video/configs"
	"magnet2video/internal/events/processor"
	"magnet2video/pkg/wire"
)

// runServer boots mode=server: API + DB + Redis + event sink + heartbeat sink.
func runServer(cfg *configs.Config) {
	container, err := wire.NewServerContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize server container: %v", err)
	}
	defer container.LoggerManager.Close()

	if err := container.DatabaseManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer container.DatabaseManager.Close()

	if err := createSuperAdmin(cfg, container.DatabaseManager); err != nil {
		log.Printf("Warning: Failed to create super admin: %v", err)
	}

	if err := container.RedisManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer container.RedisManager.Close()

	// Server has no torrent client (PR3 split). Only the queue producer
	// needs explicit shutdown.
	defer container.QueueProducer.Close()

	container.TorrentService.SetTranscodeChecker(container.TranscodeService)
	container.EventProcessor.SetTranscodeChecker(container.TranscodeService)
	log.Println("[server mode] TranscodeChecker wired into EventProcessor")

	// On worker fresh-boot, re-dispatch any torrents that were active before
	// the restart. The worker forgets in-flight state across restarts (the
	// torrent client persists nothing useful between processes), so without
	// this hook every worker reboot would silently drop active downloads.
	container.HeartbeatConsumer.SetFreshBootHook(func(ctx context.Context, workerID string) {
		log.Printf("[server mode] worker %s fresh-boot detected; re-dispatching active torrents", workerID)
		container.TorrentService.RedispatchActiveTorrents(ctx, "fresh-boot:"+workerID)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumers := startConsumers(cfg, container, consumerConfig{
		workerJobs:         false, // server does NOT run torrent/transcode/upload work
		workerEvents:       true,  // server listens for worker lifecycle events
		workerHeartbeat:    true,  // server listens for worker heartbeats
		parseMagnetResults: true,  // server waits for parse-magnet replies on this topic
	})
	defer closeConsumers(consumers)

	// Stuck-state reaper: catches files left in Pending/Processing/Uploading
	// when the worker crashes mid-job or a queue message is lost. Opt-in via
	// EVENTS.REAPER.ENABLED in config.
	go processor.NewReaperFromConfig(container.EventProcessor, cfg.EventsConfig.Reaper).Run(ctx)

	runHTTPServer(cfg, container)
}
