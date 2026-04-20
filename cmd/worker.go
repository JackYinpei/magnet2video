// Package cmd: worker-only bootstrap. Runs torrent download + transcode +
// cloud upload consumers, and publishes progress/heartbeat events to the
// server. No Gin, no DB, no Redis.
// Author: magnet2video
// Created: 2026-04-20
package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"magnet2video/configs"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/queue"
	torrentTypes "magnet2video/internal/torrent/types"
	transcodeTypes "magnet2video/internal/transcode/types"
	"magnet2video/pkg/wire"
)

// runWorker boots mode=worker: consumers only, plus heartbeat + progress loops.
func runWorker(cfg *configs.Config) {
	container, err := wire.NewWorkerContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize worker container: %v", err)
	}
	defer container.LoggerManager.Close()
	defer container.TorrentManager.Close()
	defer container.QueueProducer.Close()

	log.Printf("[worker mode] starting as %s (download_dir=%s, queue=%s)",
		workerIDFor(cfg), cfg.TorrentConfig.DownloadDir, cfg.QueueConfig.Type)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Download-jobs consumer
	dlConsumer, err := queue.NewConsumer(cfg, container.DownloadJobHandler)
	if err != nil {
		log.Fatalf("download-jobs consumer init failed: %v", err)
	}
	if err := dlConsumer.Subscribe([]string{torrentTypes.TopicDownloadJobs}); err != nil {
		log.Fatalf("download-jobs subscribe failed: %v", err)
	}
	defer dlConsumer.Close()

	// Transcode consumer
	trConsumer, err := queue.NewConsumer(cfg, container.TranscodeHandler)
	if err != nil {
		log.Fatalf("transcode consumer init failed: %v", err)
	}
	if err := trConsumer.Subscribe([]string{transcodeTypes.TopicTranscodeJobs}); err != nil {
		log.Fatalf("transcode subscribe failed: %v", err)
	}
	defer trConsumer.Close()

	// Cloud upload consumer (if enabled)
	var cuConsumer queue.Consumer
	if cfg.CloudStorageConfig.Enabled {
		cuConsumer, err = queue.NewConsumer(cfg, container.CloudUploadHandler)
		if err != nil {
			log.Fatalf("cloud-upload consumer init failed: %v", err)
		}
		if err := cuConsumer.Subscribe([]string{cloudTypes.TopicCloudUploadJobs}); err != nil {
			log.Fatalf("cloud-upload subscribe failed: %v", err)
		}
		defer cuConsumer.Close()
	}

	// Background loops
	go container.HeartbeatPublisher.Start(ctx)
	go container.ProgressReporter.Start(ctx)

	log.Println("[worker mode] all consumers started; waiting for jobs")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[worker mode] shutting down")
}

// workerIDFor returns the worker id used for logging only (wire_gen.go has the
// same logic for dependency injection).
func workerIDFor(cfg *configs.Config) string {
	if cfg.AppConfig.WorkerID != "" {
		return cfg.AppConfig.WorkerID
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "worker"
	}
	return host
}
