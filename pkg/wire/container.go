// Package wire holds the dependency-injection container for the three
// deployment modes (all / server / worker).
//
// Wiring is hand-maintained. We deliberately do not use the `wire` CLI
// because the three modes need disjoint provider subsets (server has no
// torrent client, worker has no DB/Redis, etc.) — expressing that with
// `wire.NewSet` requires more scaffolding than the codegen saves. Edit the
// builder helpers below directly when adding a new dependency.
//
// (The package keeps the name `wire` for backwards-compat with imports and
// the historical layout; nothing in this file is generated.)

package wire

import (
	"fmt"

	"magnet2video/configs"
	"magnet2video/internal/ai"
	"magnet2video/internal/cache"
	"magnet2video/internal/cloud"
	cloudHandler "magnet2video/internal/cloud/handler"
	"magnet2video/internal/db"
	eventGateway "magnet2video/internal/events/gateway"
	"magnet2video/internal/events/heartbeat"
	"magnet2video/internal/events/processor"
	"magnet2video/internal/i18n"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/redis"
	"magnet2video/internal/sse"
	"magnet2video/internal/tmdb"
	"magnet2video/internal/torrent"
	torrentHandler "magnet2video/internal/torrent/handler"
	"magnet2video/internal/torrent/replybus"
	transcodeHandler "magnet2video/internal/transcode/handler"

	"magnet2video/pkg/serve/controller"
	"magnet2video/pkg/serve/service"
	"magnet2video/pkg/serve/service/impl"
)

// Container is a unified container. For split deployments, some fields are nil.
type Container struct {
	Config *configs.Config

	// --- Shared infra ---
	LoggerManager logger.LoggerManager
	QueueProducer queue.Producer
	WorkerGateway eventGateway.WorkerGateway

	// --- Server-only infra ---
	AIManager           *ai.AIManager
	CacheManager        cache.CacheManager
	CloudStorageManager cloud.CloudStorageManager
	DatabaseManager     db.DatabaseManager
	RedisManager        redis.RedisManager
	I18nManager         i18n.I18nManager
	SSEManager          sse.SSEManager
	TorrentManager      torrent.TorrentManager
	TMDBClient          *tmdb.TMDBClient

	// --- Event plumbing ---
	EventProcessor     *processor.WorkerEventProcessor
	StatusStore        *heartbeat.StatusStore
	HeartbeatConsumer  *heartbeat.Consumer
	HeartbeatPublisher *heartbeat.Publisher
	ProgressReporter   *torrentHandler.ProgressReporter

	// --- Worker-side handlers ---
	TranscodeHandler           *transcodeHandler.TranscodeHandler
	CloudUploadHandler         *cloudHandler.CloudUploadHandler
	DownloadJobHandler         *torrentHandler.DownloadJobHandler
	FileOpsHandler             *torrentHandler.FileOpsHandler
	ParseMagnetHandler         *torrentHandler.ParseMagnetHandler

	// --- Server-side parse-magnet reply plumbing ---
	ParseMagnetBus             *replybus.ParseMagnetBus
	ParseMagnetResultsConsumer *torrentHandler.ParseMagnetResultsConsumer

	// --- Controllers (server/all only) ---
	TestController    *controller.TestController
	TorrentController *controller.TorrentController
	UserController    *controller.UserController
	AdminController   *controller.AdminController
	WorkerController  *controller.WorkerController

	// --- Services ---
	TorrentService   service.TorrentService
	TranscodeService service.TranscodeService
}

// NewContainer builds the all-in-one container (mode=all). The server-side
// and worker-side dependencies coexist in one process; communication still
// flows through the queue (GoChannel in-process) so the code path matches
// split deployment and there's no "all-mode shortcut" that bypasses the MQ.
func NewContainer(config *configs.Config) (*Container, error) {
	c := &Container{Config: config}
	if err := buildShared(c, config); err != nil {
		return nil, err
	}
	if err := buildServerInfra(c, config); err != nil {
		return nil, err
	}
	if err := buildWorkerInfra(c, config); err != nil {
		return nil, err
	}
	buildServerEventPlumbing(c, config)
	buildWorkerEventPlumbing(c, config)
	buildWorkerHandlers(c, config)
	buildServicesAndControllers(c, config)
	return c, nil
}

// NewServerContainer builds a container for mode=server. The torrent client,
// ffmpeg, and the worker-side event publisher / progress reporter are NOT
// constructed — server's TorrentManager field stays nil. Server-side code
// that historically needed it falls back to config defaults; code paths
// that genuinely required it (local file streaming) return 503.
func NewServerContainer(config *configs.Config) (*Container, error) {
	c := &Container{Config: config}
	if err := buildShared(c, config); err != nil {
		return nil, err
	}
	if err := buildServerInfra(c, config); err != nil {
		return nil, err
	}
	buildServerEventPlumbing(c, config)
	buildServicesAndControllers(c, config)
	return c, nil
}

// NewWorkerContainer builds a container for mode=worker. No DB / Redis /
// services / controllers — the worker is a stateless executor.
func NewWorkerContainer(config *configs.Config) (*Container, error) {
	c := &Container{Config: config}
	if err := buildShared(c, config); err != nil {
		return nil, err
	}
	if err := buildWorkerInfra(c, config); err != nil {
		return nil, err
	}
	buildWorkerEventPlumbing(c, config)
	buildWorkerHandlers(c, config)
	return c, nil
}

// ---- builder helpers ----

func buildShared(c *Container, config *configs.Config) error {
	l, err := logger.New(config)
	if err != nil {
		return fmt.Errorf("logger init: %w", err)
	}
	c.LoggerManager = l

	p, err := queue.NewProducer(config)
	if err != nil {
		return fmt.Errorf("queue producer init: %w", err)
	}
	c.QueueProducer = p
	return nil
}

func buildServerInfra(c *Container, config *configs.Config) error {
	aiMgr, err := ai.New(config)
	if err != nil {
		return fmt.Errorf("ai init: %w", err)
	}
	c.AIManager = aiMgr

	r, err := redis.New(config)
	if err != nil {
		return fmt.Errorf("redis init: %w", err)
	}
	c.RedisManager = r

	c.CacheManager = cache.New(r, c.LoggerManager)
	c.CloudStorageManager = cloud.New(config, c.LoggerManager)
	c.DatabaseManager = db.New(config)
	c.I18nManager = i18n.New()
	c.SSEManager = sse.New(config)
	c.TMDBClient = tmdb.New(config)
	return nil
}

func buildWorkerInfra(c *Container, config *configs.Config) error {
	// Cloud manager may already exist (server path); don't overwrite.
	if c.CloudStorageManager == nil {
		c.CloudStorageManager = cloud.New(config, c.LoggerManager)
	}
	if c.TorrentManager == nil {
		tm, err := torrent.New(config)
		if err != nil {
			return fmt.Errorf("torrent manager init: %w", err)
		}
		c.TorrentManager = tm
	}
	return nil
}

// buildServerEventPlumbing wires the server-side consumers of worker events
// (DB writer + heartbeat tracker). Does NOT touch TorrentManager — the
// server may not have one.
func buildServerEventPlumbing(c *Container, config *configs.Config) {
	c.EventProcessor = processor.NewWorkerEventProcessor(config, c.LoggerManager, c.DatabaseManager, c.RedisManager, c.QueueProducer)
	c.StatusStore = heartbeat.NewStatusStore(c.RedisManager, c.LoggerManager)
	c.HeartbeatConsumer = heartbeat.NewConsumer(c.StatusStore, c.LoggerManager)
}

// buildWorkerEventPlumbing wires the worker-side gateway + publishers.
// Requires TorrentManager already built (caller must run buildWorkerInfra
// first).
func buildWorkerEventPlumbing(c *Container, config *configs.Config) {
	c.WorkerGateway = eventGateway.NewMQGateway(c.QueueProducer, c.LoggerManager, workerIDFor(config))
	c.HeartbeatPublisher = heartbeat.NewPublisher(c.WorkerGateway, c.LoggerManager, config.TorrentConfig.DownloadDir, config.AppConfig.AppName)
	c.ProgressReporter = torrentHandler.NewProgressReporter(c.TorrentManager, c.WorkerGateway, c.LoggerManager)
}

func buildWorkerHandlers(c *Container, config *configs.Config) {
	c.TranscodeHandler = transcodeHandler.NewTranscodeHandler(config, c.LoggerManager, c.WorkerGateway)
	c.CloudUploadHandler = cloudHandler.NewCloudUploadHandler(config, c.LoggerManager, c.WorkerGateway, c.CloudStorageManager, c.QueueProducer)
	c.DownloadJobHandler = torrentHandler.NewDownloadJobHandler(config, c.LoggerManager, c.TorrentManager, c.WorkerGateway, c.ProgressReporter)
	c.FileOpsHandler = torrentHandler.NewFileOpsHandler(config, c.LoggerManager, c.WorkerGateway)
	c.ParseMagnetHandler = torrentHandler.NewParseMagnetHandler(c.LoggerManager, c.TorrentManager, c.WorkerGateway, c.QueueProducer)
}

func buildServicesAndControllers(c *Container, config *configs.Config) {
	testService := impl.NewTestService(c.LoggerManager, c.RedisManager, c.AIManager)
	c.TestController = controller.NewTestController(testService, c.SSEManager)

	// Parse-magnet reply plumbing lives wherever ParseMagnet is invoked
	// (mode=all + mode=server). Worker-only mode never builds this.
	if c.ParseMagnetBus == nil {
		c.ParseMagnetBus = replybus.NewParseMagnetBus()
	}
	if c.ParseMagnetResultsConsumer == nil {
		c.ParseMagnetResultsConsumer = torrentHandler.NewParseMagnetResultsConsumer(c.LoggerManager, c.ParseMagnetBus)
	}

	torrentSvc := impl.NewTorrentService(config, c.LoggerManager, c.DatabaseManager, c.TorrentManager, c.CacheManager, c.QueueProducer, c.ParseMagnetBus)
	c.TorrentService = torrentSvc

	transcodeSvc := impl.NewTranscodeService(config, c.LoggerManager, c.DatabaseManager, c.QueueProducer)
	c.TranscodeService = transcodeSvc

	c.TorrentController = controller.NewTorrentController(config, torrentSvc, transcodeSvc, c.DatabaseManager, c.CloudStorageManager, c.QueueProducer, c.TMDBClient)

	userSvc := impl.NewUserService(c.LoggerManager, c.DatabaseManager)
	c.UserController = controller.NewUserController(userSvc)

	adminSvc := impl.NewAdminService(c.LoggerManager, c.DatabaseManager, c.QueueProducer, c.StatusStore, c.CloudStorageManager, c.CacheManager)
	c.AdminController = controller.NewAdminController(adminSvc)

	c.WorkerController = controller.NewWorkerController(c.StatusStore)
}

// workerIDFor picks the worker id: explicit config > hostname-based default.
func workerIDFor(config *configs.Config) string {
	if config.AppConfig.WorkerID != "" {
		return config.AppConfig.WorkerID
	}
	return defaultWorkerID()
}
