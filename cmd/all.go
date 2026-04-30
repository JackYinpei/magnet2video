// Package cmd: single-process (mode=all) bootstrap — everything runs in one binary.
// Author: magnet2video
// Created: 2026-04-20
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"magnet2video/configs"
	cloudTypes "magnet2video/internal/cloud/types"
	"magnet2video/internal/events/processor"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/middleware"
	"magnet2video/internal/queue"
	torrentTypes "magnet2video/internal/torrent/types"
	transcodeTypes "magnet2video/internal/transcode/types"
	"magnet2video/pkg/router"
	"magnet2video/pkg/wire"
	"magnet2video/web"
)

// runAll boots the single-process container with every consumer in-process.
func runAll(cfg *configs.Config) {
	container, err := wire.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
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

	defer container.TorrentManager.Close()
	defer container.QueueProducer.Close()

	container.TorrentService.SetTranscodeChecker(container.TranscodeService)
	container.EventProcessor.SetTranscodeChecker(container.TranscodeService)
	log.Println("TranscodeChecker wired into TorrentService + EventProcessor")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// All-mode runs every consumer in-process.
	consumers := startConsumers(cfg, container, consumerConfig{
		workerJobs:        true,
		workerEvents:      true,
		workerHeartbeat:   true,
		cloudUpload:       cfg.CloudStorageConfig.Enabled,
	})
	defer closeConsumers(consumers)

	// Worker-side loops (heartbeat, progress reporter) also run in-process.
	go container.HeartbeatPublisher.Start(ctx)
	go container.ProgressReporter.Start(ctx)

	// Stuck-state reaper: in all-mode the same process owns DB writes, so the
	// reaper runs here for the same reason as in server-mode. Opt-in via
	// EVENTS.REAPER.ENABLED.
	go processor.NewReaperFromConfig(container.EventProcessor, cfg.EventsConfig.Reaper).Run(ctx)

	runHTTPServer(cfg, container)
}

// runHTTPServer installs Gin middleware/routes and blocks until SIGINT/SIGTERM.
func runHTTPServer(cfg *configs.Config, container *wire.Container) {
	setGinMode()

	r := gin.New()
	middleware.New(r, cfg)
	router.New(r, container)
	web.RegisterStaticRoutes(r)

	serverAddr := fmt.Sprintf("%s:%s", cfg.AppConfig.AppHost, cfg.AppConfig.AppPort)
	srv := &http.Server{Addr: serverAddr, Handler: r}

	go func() {
		log.Printf("⇨ Gin server starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

// consumerConfig toggles which queue consumers to start in this process.
type consumerConfig struct {
	workerJobs      bool // transcode-jobs + cloud-upload-jobs + download-jobs (worker-side)
	workerEvents    bool // worker-events (server-side, writes to DB)
	workerHeartbeat bool // worker-heartbeat (server-side, writes to Redis)
	cloudUpload     bool // cloud-upload-jobs consumer active (required for uploads)
}

type runningConsumers struct {
	transcode     queue.Consumer
	cloudUpload   queue.Consumer
	downloadJobs  queue.Consumer
	workerEvents  queue.Consumer
	heartbeat     queue.Consumer
}

func startConsumers(cfg *configs.Config, container *wire.Container, cc consumerConfig) *runningConsumers {
	out := &runningConsumers{}

	if cc.workerJobs {
		c, err := queue.NewConsumer(cfg, container.TranscodeHandler)
		if err != nil {
			log.Printf("Warning: transcode consumer init failed: %v", err)
		} else if err := c.Subscribe([]string{transcodeTypes.TopicTranscodeJobs}); err != nil {
			log.Printf("Warning: transcode subscribe failed: %v", err)
		} else {
			out.transcode = c
			log.Printf("Transcode consumer started (queue=%s)", cfg.QueueConfig.Type)
		}

		if cc.cloudUpload {
			c2, err := queue.NewConsumer(cfg, container.CloudUploadHandler)
			if err != nil {
				log.Printf("Warning: cloud upload consumer init failed: %v", err)
			} else if err := c2.Subscribe([]string{cloudTypes.TopicCloudUploadJobs}); err != nil {
				log.Printf("Warning: cloud upload subscribe failed: %v", err)
			} else {
				out.cloudUpload = c2
				log.Println("Cloud upload consumer started")
			}
		}

		c3, err := queue.NewConsumer(cfg, container.DownloadJobHandler)
		if err != nil {
			log.Printf("Warning: download-jobs consumer init failed: %v", err)
		} else if err := c3.Subscribe([]string{torrentTypes.TopicDownloadJobs}); err != nil {
			log.Printf("Warning: download-jobs subscribe failed: %v", err)
		} else {
			out.downloadJobs = c3
			log.Println("Download-jobs consumer started")
		}
	}

	if cc.workerEvents {
		c, err := queue.NewConsumer(cfg, container.EventProcessor)
		if err != nil {
			log.Printf("Warning: worker-events consumer init failed: %v", err)
		} else if err := c.Subscribe([]string{eventTypes.TopicWorkerEvents}); err != nil {
			log.Printf("Warning: worker-events subscribe failed: %v", err)
		} else {
			out.workerEvents = c
			log.Println("Worker-events consumer started")
		}
	}

	if cc.workerHeartbeat {
		c, err := queue.NewConsumer(cfg, container.HeartbeatConsumer)
		if err != nil {
			log.Printf("Warning: heartbeat consumer init failed: %v", err)
		} else if err := c.Subscribe([]string{eventTypes.TopicWorkerHeartbeat}); err != nil {
			log.Printf("Warning: heartbeat subscribe failed: %v", err)
		} else {
			out.heartbeat = c
			log.Println("Heartbeat consumer started")
		}
	}

	return out
}

func closeConsumers(c *runningConsumers) {
	if c == nil {
		return
	}
	for _, consumer := range []queue.Consumer{c.transcode, c.cloudUpload, c.downloadJobs, c.workerEvents, c.heartbeat} {
		if consumer != nil {
			_ = consumer.Close()
		}
	}
}

func setGinMode() {
	env := os.Getenv("ENV")
	switch env {
	case "prod", "production":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}
}
